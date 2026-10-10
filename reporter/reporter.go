package reporter

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/google/uuid"
)

// TestEvent matches the field names emitted by go test -json.
//
//nolint:tagliatelle // Go test events use capitalized JSON field names.
type TestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
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

// IdentityOptions controls coordinated runs and explicit repeated-execution input.
type IdentityOptions struct {
	RunID   string
	ShardID string
	// Set for go test -count input; every completion is a separate execution, not a retry.
	SeparateExecutions bool
	TestIDResolver     func(packageName, testName string) string
}

func ParseTestResults(r io.Reader, verbose bool, env *ctrf.Environment, identityOptions ...IdentityOptions) (*ctrf.Report, error) {
	buildOutput = nil
	var testEvents []TestEvent

	options := IdentityOptions{}
	if len(identityOptions) > 0 {
		options = identityOptions[0]
	}
	if (options.RunID != "" && strings.TrimSpace(options.RunID) == "") || (options.ShardID != "" && strings.TrimSpace(options.ShardID) == "") {
		return nil, errors.New("runId and shardId must be non-empty")
	}
	report := ctrf.NewReport("gotest", env)
	applyReportIdentity(report, options)

	report.Results.Summary.Start = time.Now().UnixMilli()
	report.Results.Summary.Stop = report.Results.Summary.Start

	testStartTimes := make(map[string]int64)
	extraMap := make(map[string]any)
	buildOutputEvents := make([]TestEvent, 0)
	buildFailEvents := make([]TestEvent, 0)

	report.Results.Extra = extraMap

	var err error
	testEvents, err = decodeTestEvents(r, verbose)
	if err != nil {
		return nil, err
	}

	for i, event := range testEvents {
		if err := updateSummaryTime(report.Results.Summary, event, i); err != nil {
			return nil, err
		}
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
			continue
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
			return nil, fmt.Errorf("invalid test event time %q: %w", event.Time, err)
		}
		// A run event starts a new attempt for this test.
		if event.Action == ActionRun {
			testStartTimes[testNameKey(event.Package, event.Test)] = eventTime
		}

		// From this point on, we only deal with pass, fail, and skip events, which indicate that the
		// test has completed, and we can create/update a TestResult for it.
		if event.Action == ActionPass || event.Action == ActionFail || event.Action == ActionSkip {
			if err := addCompletedEvent(report, testEvents, i, testStartTimes, options); err != nil {
				return nil, err
			}
		}
	}

	enrichReportWithFilenames(report)

	return report, nil
}

func updateSummaryTime(summary *ctrf.Summary, event TestEvent, index int) error {
	if event.Time != "" {
		eventTime, err := parseTimeString(event.Time)
		if err != nil {
			return fmt.Errorf("invalid event time %q: %w", event.Time, err)
		}
		if index == 0 || eventTime < summary.Start {
			summary.Start = eventTime
		}
		if index == 0 || eventTime > summary.Stop {
			summary.Stop = eventTime
		}
	}
	return nil
}

func applyReportIdentity(report *ctrf.Report, options IdentityOptions) {
	if options.RunID != "" {
		report.RunID = options.RunID
	}
	if options.ShardID != "" {
		if report.Results.Environment == nil {
			report.Results.Environment = &ctrf.Environment{}
		}
		// Copy the caller's environment before attaching document metadata.
		copied := *report.Results.Environment
		copied.ShardID = options.ShardID
		report.Results.Environment = &copied
	}
}

func decodeTestEvents(r io.Reader, verbose bool) ([]TestEvent, error) {
	var testEvents []TestEvent
	decoder := json.NewDecoder(r)
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
				fmt.Fprint(os.Stdout, event.Output)
			}
		}
	}

	return testEvents, nil
}

func addCompletedEvent(report *ctrf.Report, testEvents []TestEvent, index int, testStartTimes map[string]int64, options IdentityOptions) error {
	event := testEvents[index]
	// Look up the start time, and use this event's time as the endTime, to mark the start/stop times
	// for the test result. Duration we get from the event.Elapsed field, which better takes into
	// account parallel tests, setup/teardown time, etc...
	startTime, ok := testStartTimes[testNameKey(event.Package, event.Test)]
	if ok {
		delete(testStartTimes, testNameKey(event.Package, event.Test))
	}
	stopTime, err := parseTimeString(event.Time)
	if err != nil {
		return fmt.Errorf("invalid test event time %q: %w", event.Time, err)
	}

	// Determine the message for this test result. We only include messages on failures though,
	// per the CTRF spec, so if this is not a failure, we pass an empty string for the message.
	message := ""
	if event.Action == ActionFail {
		message = getMessagesForTest(testEvents, index, event.Package, event.Test, startTime)
	}

	// Build the TestResult for this test event, and add it to the report.
	tuple, err := json.Marshal([]string{event.Package, event.Test})
	if err != nil {
		return fmt.Errorf("encode test identity: %w", err)
	}
	logicalID := fmt.Sprintf("ctrf:go:v1:%x", sha256.Sum256(tuple))
	if options.TestIDResolver != nil {
		logicalID = options.TestIDResolver(event.Package, event.Test)
	}
	if strings.TrimSpace(logicalID) == "" {
		return errors.New("testId resolver must return a non-empty string")
	}
	newResult := &ctrf.TestResult{
		TestID:      logicalID,
		ExecutionID: uuid.NewString(),
		AttemptID:   uuid.NewString(),
		Suite:       []string{event.Package},
		Name:        event.Test,
		Status:      actionToTestResult(event.Action),
		Duration:    secondsToMillis(event.Elapsed),
		Message:     message,
		Start:       startTime,
		Stop:        stopTime,
	}

	// Search through the existing results for a prior run. If this is a duplicate of an existing failure,
	// then this is likely a retry of a potentially flaky test, so update the existing test result instead
	// of creating a new one.
	if existingResult := findTest(newResult.Suite, newResult.Name, report.Results.Tests); existingResult == nil || options.SeparateExecutions {
		addResult(report, newResult)
	} else {
		updateResult(report, existingResult, newResult)
	}
	return nil
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
	oldResult.RetryAttempts = append(oldResult.RetryAttempts, ctrf.RetryAttempt{
		Attempt:   len(oldResult.RetryAttempts) + 1,
		AttemptID: oldResult.AttemptID,
		Status:    oldResult.Status, Message: oldResult.Message,
		Duration: oldResult.Duration, Start: oldResult.Start, Stop: oldResult.Stop,
	})
	executionID := oldResult.ExecutionID
	history := oldResult.RetryAttempts
	wasFlaky := oldResult.Flaky
	// The top-level fields always describe the final attempt.
	*oldResult = *newResult
	oldResult.ExecutionID = executionID
	oldResult.RetryAttempts = history
	oldResult.Retries = len(history)
	if oldResult.Status == ctrf.TestPassed {
		for _, attempt := range history {
			if attempt.Status == ctrf.TestFailed {
				oldResult.Flaky = true
				break
			}
		}
	}
	if wasFlaky {
		report.Results.Summary.Flaky--
	}
	if oldResult.Flaky {
		report.Results.Summary.Flaky++
	}
	summary := report.Results.Summary
	summary.Passed, summary.Failed, summary.Skipped, summary.Pending, summary.Other = 0, 0, 0, 0, 0
	for _, result := range report.Results.Tests {
		switch result.Status {
		case ctrf.TestPassed:
			summary.Passed++
		case ctrf.TestFailed:
			summary.Failed++
		case ctrf.TestSkipped:
			summary.Skipped++
		case ctrf.TestPending:
			summary.Pending++
		default:
			summary.Other++
		}
	}
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
