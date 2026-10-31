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
	return ParseTestResultsWithOptions(r, verbose, env, IdentityOptions{})
}

// ParseTestResultsWithOptions parses results with optional identity metadata.
//
//nolint:cyclop,gocognit,gocyclo // Preserve the established parser flow; refactoring is outside this change.
func ParseTestResultsWithOptions(r io.Reader, verbose bool, env *ctrf.Environment, options IdentityOptions) (*ctrf.Report, error) {
	var testEvents []TestEvent
	decoder := json.NewDecoder(r)

	if err := validateIdentityOptions(options); err != nil {
		return nil, err
	}
	report := ctrf.NewReport("gotest", env)
	applyReportIdentity(report, options)
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

			if err := applyTestIdentity(newResult, event.Package, options); err != nil {
				return nil, err
			}

			// Search through the existing results for a prior run. If this is a duplicate of an existing failure,
			// then this is likely a retry of a potentially flaky test, so update the existing test result instead
			// of creating a new one.
			if existingResult := findTest(newResult.Suite, newResult.Name, report.Results.Tests); existingResult == nil {
				addResult(report, newResult)
			} else {
				updateResult(report, existingResult, newResult)
			}
		}
	}

	// Keep empty reports writable without changing event timing for test results.
	if len(report.Results.Tests) == 0 {
		report.Results.Summary.Stop = report.Results.Summary.Start
	}

	enrichReportWithFilenames(report)

	return report, nil
}

// addResult adds a new test result to the report, filling out all the relevant details.
func addResult(report *ctrf.Report, result *ctrf.TestResult) {
	// Update the overall test count in the Summary
	report.Results.Summary.Tests++

	adjustStatusCount(report.Results.Summary, result.Status, 1)

	// Append the result to the report's results
	report.Results.Tests = append(report.Results.Tests, result)
}

func adjustStatusCount(summary *ctrf.Summary, status ctrf.TestStatus, delta int) {
	switch status {
	case ctrf.TestPassed:
		summary.Passed += delta
	case ctrf.TestFailed:
		summary.Failed += delta
	case ctrf.TestSkipped:
		summary.Skipped += delta
	case ctrf.TestPending:
		summary.Pending += delta
	default:
		summary.Other += delta
	}
}

func updateResult(report *ctrf.Report, oldResult, newResult *ctrf.TestResult) {
	// The previous final result becomes history only when another attempt finishes.
	history := oldResult.RetryAttempts
	history = append(history, ctrf.RetryAttempt{
		AttemptID: oldResult.AttemptID,
		Attempt:   len(oldResult.RetryAttempts) + 1,
		Status:    oldResult.Status,
		Message:   oldResult.Message,
		Duration:  oldResult.Duration,
		Start:     oldResult.Start,
		Stop:      oldResult.Stop,
	})
	newResult.RetryAttempts = history
	newResult.Retries = len(history)
	newResult.TestID = oldResult.TestID
	newResult.ExecutionID = oldResult.ExecutionID
	if newResult.Status == ctrf.TestPassed {
		for _, attempt := range newResult.RetryAttempts {
			if attempt.Status == ctrf.TestFailed {
				newResult.Flaky = true
				break
			}
		}
	}
	adjustStatusCount(report.Results.Summary, oldResult.Status, -1)
	adjustStatusCount(report.Results.Summary, newResult.Status, 1)
	if oldResult.Flaky {
		report.Results.Summary.Flaky--
	}
	if newResult.Flaky {
		report.Results.Summary.Flaky++
	}
	*oldResult = *newResult
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

func sliceIsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// findTest searches through the already-parsed test results to find a matching test.
func findTest(suite []string, name string, tests []*ctrf.TestResult) *ctrf.TestResult {
	for _, test := range tests {
		if sliceIsEqual(suite, test.Suite) && test.Name == name {
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
