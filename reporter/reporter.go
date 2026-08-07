package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
)

type TestEvent struct {
	Time    string
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}

const (
	ActionBuildOutput = "build-output"
	ActionBuildFail   = "build-fail"
	ActionOutput      = "output"
	ActionRun         = "run"
	ActionPass        = "pass"
	ActionFail        = "fail"
	ActionSkip        = "skip"
)

var buildOutput []string

func ParseTestResults(r io.Reader, verbose bool, env *ctrf.Environment) (*ctrf.Report, error) {
	var testEvents []TestEvent
	decoder := json.NewDecoder(r)

	report := ctrf.NewReport("gotest", env)
	report.Results.Summary.Start = time.Now().UnixNano() / int64(time.Millisecond)

	testStartTimes := make(map[string]int64)
	extraMap := make(map[string]any)
	buildOutputEvents := make([]TestEvent, 0)
	buildFailEvents := make([]TestEvent, 0)

	report.Results.Extra = extraMap

	for {
		var event TestEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		testEvents = append(testEvents, event)

		if verbose {
			if event.Action == ActionBuildOutput || event.Action == ActionOutput {
				fmt.Print(event.Output)
			}
		}
	}

	for i, event := range testEvents {
		// If we see any test failures, mark an overall failure in the Extra fields
		if event.Action == ActionFail {
			extraMap["FailedBuild"] = true
		}

		if event.Action == ActionBuildOutput {
			// Capture the full events to the extras
			buildOutputEvents = append(buildOutputEvents, event)
			extraMap["buildOutput"] = buildOutputEvents

			// Capture the actual build output as well
			buildOutput = append(buildOutput, event.Output)
			continue
		}

		// Mark if we see a build failure in the extras field
		if event.Action == ActionBuildFail {
			buildFailEvents = append(buildFailEvents, event)
			extraMap["buildFail"] = buildFailEvents
			break
		}

		if event.Action == ActionOutput {
			buildOutput = append(buildOutput, event.Output)
		}

		// From this point, we only care about events associated with an actual test
		if event.Test == "" {
			continue
		}

		// Parse timestamp data from the event
		eventTime, err := parseTimeString(event.Time)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing test event start time '%s' : %v\n", event.Time, err)
		} else {
			if eventTime < report.Results.Summary.Start {
				report.Results.Summary.Start = eventTime
			}
			if eventTime > report.Results.Summary.Stop {
				report.Results.Summary.Stop = eventTime
			}

			// If this is a "run" event, record the start time of the test. We'll look this up later when
			// we process the "pass"/"fail"/"skip" event for the test to create the TestResult
			if event.Action == ActionRun {
				testStartTimes[testNameKey(event.Package, event.Test)] = eventTime
			}
		}

		// From this point on, we only deal with pass, fail, and skip events, which indicate that the
		// test has completed, and we can create/update a TestResult for it.
		if event.Action == ActionPass || event.Action == ActionFail || event.Action == ActionSkip {
			// Look up the start time, and use this event's time as the endTime, to mark the start/stop times
			// for the test result. Duration we get from the event.Elapsed field, which better takes into
			// account parallel tests, setup/teardown time, etc...
			startTime, ok := testStartTimes[testNameKey(event.Package, event.Test)]
			if ok {
				delete(testStartTimes, testNameKey(event.Package, event.Test))
			}
			stopTime := eventTime

			// Determine the message for this test result. We only include messages on failures though,
			// per the CTRF spec, so if this is not a failure, we pass an empty string for the message.
			message := ""
			if event.Action == ActionFail {
				message = getMessagesForTest(testEvents, i, event.Package, event.Test, startTime)
			}

			// Build the TestResult for this test event, and add it to the report.
			newResult := &ctrf.TestResult{
				Suite:    []string{event.Package},
				Name:     event.Test,
				Status:   actionToTestResult(event.Action),
				Duration: secondsToMillis(event.Elapsed),
				Message:  message,
				Start:    startTime,
				Stop:     stopTime,
			}

			// Search through the existing results for a prior run. If this is a duplicate of an existing failure,
			// then this is likely a retry of a potentially flaky test, so update the existing test result instead
			// of creating a new one.
			if existingResult := findTest(event.Package, event.Test, report.Results.Tests); existingResult == nil {
				addResult(report, newResult)
			} else {
				updateResult(report, existingResult, newResult)
			}
		}
	}

	enrichReportWithFilenames(report)

	return report, nil
}

// addResult adds a new test result to the report, filling out all the relevant details.
func addResult(report *ctrf.Report, result *ctrf.TestResult) {
	// Update the overall test count in the Summary
	report.Results.Summary.Tests++

	// Update the sub-count based on the test result status
	switch result.Status {
	case ctrf.TestPassed:
		report.Results.Summary.Passed++
	case ctrf.TestFailed:
		report.Results.Summary.Failed++
	case ctrf.TestSkipped:
		report.Results.Summary.Skipped++
	case ctrf.TestPending:
		report.Results.Summary.Pending++
	default:
		report.Results.Summary.Other++
	}

	// Append the result to the report's results
	report.Results.Tests = append(report.Results.Tests, result)
}

func updateResult(report *ctrf.Report, oldResult, newResult *ctrf.TestResult) {
	// If the existing result does not have a retries field, initialize it, and move the
	// results to the first RetryAttempts object
	if oldResult.RetryAttempts == nil {
		oldResult.Retries = 1
		oldResult.RetryAttempts = append(oldResult.RetryAttempts, ctrf.RetryAttempt{
			Attempt:  1,
			Status:   oldResult.Status,
			Message:  oldResult.Message,
			Duration: oldResult.Duration,
			Start:    oldResult.Start,
			Stop:     oldResult.Stop,
		})
	}

	// If this is a pass after a failure, mark the test as flaky, not failed,
	// and update the summary counts accordingly
	if oldResult.Status == ctrf.TestFailed && newResult.Status == ctrf.TestPassed {
		oldResult.Flaky = true
		report.Results.Summary.Flaky++
		report.Results.Summary.Failed--
	}

	// Update the overall test status to match that of the new result
	oldResult.Status = newResult.Status

	// Clear out the top-level message on the overall result, since the messages are in the retries
	oldResult.Message = ""

	// Update the times of the overall test result
	oldResult.Duration += newResult.Duration
	if newResult.Stop > oldResult.Stop {
		oldResult.Stop = newResult.Stop
	}
	if newResult.Start < oldResult.Start {
		oldResult.Start = newResult.Start
	}

	// Now add the new attempt to the retries
	oldResult.Retries++
	oldResult.RetryAttempts = append(oldResult.RetryAttempts, ctrf.RetryAttempt{
		Attempt:  oldResult.Retries,
		Status:   newResult.Status,
		Message:  newResult.Message,
		Duration: newResult.Duration,
		Start:    newResult.Start,
		Stop:     newResult.Stop,
	})
}

// testNameKey generates a unique key for a map lookup based on the test name and suite.
func testNameKey(suite, name string) string {
	return fmt.Sprintf("%s.%s", suite, name)
}

func actionToTestResult(action string) ctrf.TestStatus {
	switch action {
	case ActionPass:
		return ctrf.TestPassed
	case ActionFail:
		return ctrf.TestFailed
	case ActionSkip:
		return ctrf.TestSkipped
	default:
		return ctrf.TestOther
	}
}

// findTest searches through the already-parsed test results to find a matching test.
func findTest(suite string, name string, tests []*ctrf.TestResult) *ctrf.TestResult {
	for _, test := range tests {
		if len(test.Suite) == 1 && test.Suite[0] == suite && test.Name == name {
			return test
		}
	}
	return nil
}

func generateTestMap() map[string][]string {
	tests := map[string][]string{}

	r := regexp.MustCompile(`Test.\w+`)
	if err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, "_test.go") || err != nil {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		tests[path] = r.FindAllString(string(data), -1)

		return nil
	}); err != nil {
		return tests
	}

	return tests
}

func enrichReportWithFilenames(report *ctrf.Report) {
	tests := generateTestMap()

	for i, testResult := range report.Results.Tests {
		for file, names := range tests {
			for _, name := range names {
				if strings.Contains(testResult.Name, name) {
					report.Results.Tests[i].Filepath = file
				}
			}
		}
	}
}

func getMessagesForTest(testEvents []TestEvent, index int, packageName, testName string, startTime int64) string {
	var messages []string
	for i := index; i >= 0; i-- {
		if testEvents[i].Package == packageName && testEvents[i].Test == testName {
			// If we are only getting the messages for a single test retry, then we only want the messages
			// that occurred on or after the start time of that retry attempt. If startTime is 0, then we
			// want all messages for the test regardless of time.
			if startTime != 0 {
				eventTime, err := parseTimeString(testEvents[i].Time)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error parsing test event time '%s' : %v\n", testEvents[i].Time, err)
				}
				if eventTime < startTime {
					continue
				}
			}

			if testEvents[i].Action == ActionOutput {
				messages = append(messages, testEvents[i].Output)
			}
		}
	}
	reverse(messages)
	return strings.Join(messages, "")
}

func WriteReportToFile(filename string, report *ctrf.Report) error {
	err := report.WriteFile(filename)
	if err != nil {
		return err
	}
	fmt.Println("go-ctrf-json-reporter: successfully written ctrf json to", filename)
	return nil
}

func GetBuildOutput() string {
	return strings.Join(buildOutput, "")
}

func secondsToMillis(seconds float64) int64 {
	return int64(seconds * 1000)
}

func parseTimeString(timeString string) (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, timeString)
	if err != nil {
		return 0, err
	}
	return t.UnixNano() / int64(time.Millisecond), nil
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
