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

var buildOutput []string

func ParseTestResults(r io.Reader, verbose bool, env *ctrf.Environment) (*ctrf.Report, error) {
	var testEvents []TestEvent
	decoder := json.NewDecoder(r)

	report := ctrf.NewReport("gotest", env)
	report.Results.Summary.Start = time.Now().UnixNano() / int64(time.Millisecond)

	testStartTimes := make(map[string]int64)

	for {
		var event TestEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		testEvents = append(testEvents, event)

		if verbose {
			if event.Action == "build-output" || event.Action == "output" {
				fmt.Print(event.Output)
			}
		}
	}

	for i, event := range testEvents {

		if event.Action == "build-output" || event.Action == "build-fail" || event.Action == "fail" {
			if report.Results.Extra == nil {
				report.Results.Extra = make(map[string]any)
			}
			extraMap, ok := report.Results.Extra.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected a map, but got %T instead", report.Results.Extra)
			}

			if event.Action == "fail" {
				if _, ok := extraMap["FailedBuild"]; !ok {
					extraMap["FailedBuild"] = true
				}
			}

			if event.Action == "build-output" {
				if _, ok := extraMap["buildOutput"]; !ok {
					extraMap["buildOutput"] = []TestEvent{}
				}
				buildOutputEvents := extraMap["buildOutput"].([]TestEvent)
				extraMap["buildOutput"] = append(buildOutputEvents, event)
				buildOutput = append(buildOutput, event.Output)
				continue
			}

			if event.Action == "build-fail" {
				if _, ok := extraMap["buildFail"]; !ok {
					extraMap["buildFail"] = []TestEvent{}
				}
				buildFailEvents := extraMap["buildFail"].([]TestEvent)
				extraMap["buildFail"] = append(buildFailEvents, event)
				break
			}
		}

		if event.Action == "output" {
			buildOutput = append(buildOutput, event.Output)
		}

		if event.Test == "" {
			continue
		}
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
			if event.Action == "run" {
				testStartTimes[testNameKey(event.Package, event.Test)] = eventTime
			}
		}

		// From this point on, we only deal with pass, fail, and skip events, which indicate that the
		// test has completed, and we can create/update a TestResult for it.
		if event.Action == "pass" || event.Action == "fail" || event.Action == "skip" {
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
			if event.Action == "fail" {
				message = getMessagesForTest(testEvents, i, event.Package, event.Test, startTime)
			}

			// Build the TestResult for this test event, and add it to the report.
			newResult := &ctrf.TestResult{
				Suite:    event.Package,
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

// addResult adds a new test result to the report, filling out all the relevant details
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
	}

	// Append the result to the report's results
	report.Results.Tests = append(report.Results.Tests, result)
}

func updateResult(report *ctrf.Report, existing, new *ctrf.TestResult) {
	// If the existing result does not have a retries field, initialize it, and move the
	// results to the first RetryAttempts object
	if existing.RetryAttempts == nil {
		existing.Retries = 1
		existing.RetryAttempts = append(existing.RetryAttempts, ctrf.RetryAttempt{
			Attempt:  1,
			Status:   existing.Status,
			Message:  existing.Message,
			Duration: existing.Duration,
			Start:    existing.Start,
			Stop:     existing.Stop,
		})
	}

	// If this is a pass after a failure, mark the test as flaky, not failed,
	// and update the summary counts accordingly
	if existing.Status == ctrf.TestFailed && new.Status == ctrf.TestPassed {
		existing.Flaky = true
		report.Results.Summary.Flaky++
		report.Results.Summary.Failed--
	}

	// Update the overall test status to match that of the new result
	existing.Status = new.Status

	// Clear out the top-level message on the overall result, since the messages are in the retries
	existing.Message = ""

	// Update the times of the overall test result
	existing.Duration += new.Duration
	if new.Stop > existing.Stop {
		existing.Stop = new.Stop
	}
	if new.Start < existing.Start {
		existing.Start = new.Start
	}

	// Now add the new attempt to the retries
	existing.Retries++
	existing.RetryAttempts = append(existing.RetryAttempts, ctrf.RetryAttempt{
		Attempt:  existing.Retries,
		Status:   new.Status,
		Message:  new.Message,
		Duration: new.Duration,
		Start:    new.Start,
		Stop:     new.Stop,
	})
}

// testNameKey generates a unique key for a map lookup based on the test name and suite.
func testNameKey(suite, name string) string {
	return fmt.Sprintf("%s.%s", suite, name)
}

func actionToTestResult(action string) ctrf.TestStatus {
	switch action {
	case "pass":
		return ctrf.TestPassed
	case "fail":
		return ctrf.TestFailed
	case "skip":
		return ctrf.TestSkipped
	default:
		return ctrf.TestOther
	}
}

// findTest searches through the already-parsed test results to find a matching test.
func findTest(suite string, name string, tests []*ctrf.TestResult) *ctrf.TestResult {
	for _, test := range tests {
		if test.Suite == suite && test.Name == name {
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

			if testEvents[i].Action == "output" {
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
