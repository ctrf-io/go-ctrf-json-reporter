package ctrf

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Report struct {
	RunID        string         `json:"runId,omitempty"`
	ReportFormat string         `json:"reportFormat"`
	SpecVersion  string         `json:"specVersion"`
	ReportId     string         `json:"reportId,omitempty"` //nolint:revive // this field is part of the interface: careful.
	Timestamp    time.Time      `json:"timestamp,omitempty"`
	GeneratedBy  string         `json:"generatedBy,omitempty"`
	Results      *Results       `json:"results"`
	Extra        map[string]any `json:"extra,omitempty"`
}

const (
	ReportFormatCTRF   = "CTRF"
	SpecVersionCTRF    = "0.2.0"
	GeneratedByDefault = "go-ctrf-json-reporter"
)

func NewReport(toolName string, env *Environment) *Report {
	return &Report{
		ReportFormat: ReportFormatCTRF,
		SpecVersion:  SpecVersionCTRF,
		ReportId:     uuid.New().String(),
		Timestamp:    time.Now(),
		Results: &Results{
			Tool:        &Tool{Name: toolName},
			Environment: env,
			Summary:     &Summary{},
			Tests:       []*TestResult{},
		},
		GeneratedBy: GeneratedByDefault,
	}
}

func (report *Report) ToJson() (string, error) {
	stringBuilder := &strings.Builder{}

	err := report.Write(stringBuilder, false)
	if err != nil {
		return "", err
	}

	return stringBuilder.String(), nil
}

func (report *Report) ToJsonPretty() (string, error) {
	stringBuilder := &strings.Builder{}

	err := report.Write(stringBuilder, true)
	if err != nil {
		return "", err
	}

	return stringBuilder.String(), nil
}

func (report *Report) Write(w io.Writer, pretty bool) error {
	if errs := report.Validate(); len(errs) > 0 {
		return fmt.Errorf("report is invalid: %v", errs)
	}
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	err := encoder.Encode(report)
	if err != nil {
		return fmt.Errorf("error writing ctrf json report: %v", err)
	}
	return nil
}

func (report *Report) WriteFile(filePath string) error {
	if errs := report.Validate(); len(errs) > 0 {
		return fmt.Errorf("report is invalid: %v", errs)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error writing ctrf json report: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	return report.Write(file, true)
}

func (report *Report) Validate() []error {
	if report.RunID != "" && strings.TrimSpace(report.RunID) == "" {
		return singleError("runId must be non-empty")
	}
	results := report.Results
	if results == nil {
		return singleError("missing property 'results'")
	}

	var errs []error
	if report.ReportFormat != ReportFormatCTRF {
		errs = append(errs, errors.New("invalid reportFormat: must be CTRF"))
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(report.SpecVersion) {
		errs = append(errs, errors.New("invalid specVersion"))
	}
	if report.ReportId != "" {
		if len(report.ReportId) != 36 || uuid.Validate(report.ReportId) != nil {
			errs = append(errs, errors.New("invalid reportId: must be a hyphenated UUID"))
		}
	}
	if results.Environment != nil && results.Environment.ShardID != "" && strings.TrimSpace(results.Environment.ShardID) == "" {
		errs = append(errs, errors.New("shardId must be non-empty"))
	}
	if results.Tool == nil {
		errs = append(errs, errors.New("missing property 'results.tool'"))
	} else {
		errs = append(errs, results.Tool.Validate()...)
	}
	if results.Summary == nil {
		errs = append(errs, errors.New("missing property 'results.summary'"))
	} else {
		errs = append(errs, results.Summary.Validate()...)
	}
	if results.Tests == nil {
		errs = append(errs, errors.New("missing property 'results.tests'"))
	}
	for i, test := range results.Tests {
		if test == nil {
			errs = append(errs, fmt.Errorf("results.tests[%d] must be an object", i))
			continue
		}
		for _, err := range test.Validate() {
			errs = append(errs, fmt.Errorf("results.tests[%d]: %w", i, err))
		}
	}
	// Also reject values in parameters/extra that cannot be represented as JSON.
	if _, err := json.Marshal(report); err != nil {
		errs = append(errs, fmt.Errorf("invalid JSON value: %w", err))
	}
	return errs
}

func singleError(errorMessage string) []error {
	return []error{errors.New(errorMessage)}
}

type Results struct {
	Tool        *Tool          `json:"tool"`
	Summary     *Summary       `json:"summary"`
	Tests       []*TestResult  `json:"tests"`
	Environment *Environment   `json:"environment,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

type Tool struct {
	Name    string         `json:"name"`
	Version string         `json:"version,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`
}

func (tool *Tool) Validate() []error {
	if tool.Name == "" {
		return singleError("missing property 'results.tool.name'")
	}
	return nil
}

type Summary struct {
	Tests   int            `json:"tests"`
	Passed  int            `json:"passed"`
	Failed  int            `json:"failed"`
	Pending int            `json:"pending"`
	Skipped int            `json:"skipped"`
	Other   int            `json:"other"`
	Flaky   int            `json:"flaky,omitempty"`
	Suites  int            `json:"suites,omitempty"`
	Start   int64          `json:"start"`
	Stop    int64          `json:"stop"`
	Extra   map[string]any `json:"extra,omitempty"`
}

func (summary *Summary) Validate() []error {
	var errs []error
	if summary.Tests < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.tests'"))
	}
	if summary.Passed < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.passed'"))
	}
	if summary.Failed < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.failed'"))
	}
	if summary.Pending < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.pending'"))
	}
	if summary.Skipped < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.skipped'"))
	}
	if summary.Other < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.other'"))
	}
	if summary.Start < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.start'"))
	}
	if summary.Stop < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.stop'"))
	}
	if summary.Flaky < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.flaky'"))
	}
	if summary.Suites < 0 {
		errs = append(errs, errors.New("invalid property 'results.summary.suites'"))
	}
	if summary.Start > summary.Stop {
		errs = append(errs, errors.New("invalid summary timestamps: start can't be greater than stop"))
	}
	testsSum := summary.Passed + summary.Failed + summary.Pending + summary.Skipped + summary.Other
	if summary.Tests != testsSum {
		errs = append(errs, fmt.Errorf("invalid summary counts: tests (%d) must be the sum of passed, failed, pending, skipped, and other (%d)", summary.Tests, testsSum))
	}
	return errs
}

type TestStatus string

const (
	TestPassed  TestStatus = "passed"
	TestFailed  TestStatus = "failed"
	TestSkipped TestStatus = "skipped"
	TestPending TestStatus = "pending"
	TestOther   TestStatus = "other"
)

type RetryAttempt struct {
	AttemptID string         `json:"attemptId,omitempty"`
	Attempt   int            `json:"attempt"`
	Status    TestStatus     `json:"status"`
	Duration  int64          `json:"duration,omitempty"`
	Message   string         `json:"message,omitempty"`
	Trace     string         `json:"trace,omitempty"`
	Line      int            `json:"line,omitempty"`
	Snippet   string         `json:"snippet,omitempty"`
	Stdout    []string       `json:"stdout,omitempty"`
	Stderr    []string       `json:"stderr,omitempty"`
	Start     int64          `json:"start,omitempty"`
	Stop      int64          `json:"stop,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

type TestResult struct {
	AttemptID     string         `json:"attemptId,omitempty"`
	TestID        string         `json:"testId,omitempty"`
	ExecutionID   string         `json:"executionId,omitempty"`
	Name          string         `json:"name"`
	Status        TestStatus     `json:"status"`
	Duration      int64          `json:"duration"`
	Start         int64          `json:"start,omitempty"`
	Stop          int64          `json:"stop,omitempty"`
	Suite         []string       `json:"suite,omitempty"`
	Message       string         `json:"message,omitempty"`
	Trace         string         `json:"trace,omitempty"`
	RawStatus     string         `json:"rawStatus,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Type          string         `json:"type,omitempty"`
	Filepath      string         `json:"filePath,omitempty"`
	Retries       int            `json:"retries,omitempty"`
	Flaky         bool           `json:"flaky,omitempty"`
	Browser       string         `json:"browser,omitempty"`
	Device        string         `json:"device,omitempty"`
	Screenshot    string         `json:"screenshot,omitempty"`
	Parameters    map[string]any `json:"parameters,omitempty"`
	Steps         []Step         `json:"steps,omitempty"`
	RetryAttempts []RetryAttempt `json:"retryAttempts,omitempty"`
	Extra         map[string]any `json:"extra,omitempty"`
}

type Environment struct {
	ShardID     string         `json:"shardId,omitempty"`
	AppName     string         `json:"appName,omitempty"`
	AppVersion  string         `json:"appVersion,omitempty"`
	OSPlatform  string         `json:"osPlatform,omitempty"`
	OSRelease   string         `json:"osRelease,omitempty"`
	OSVersion   string         `json:"osVersion,omitempty"`
	BuildName   string         `json:"buildName,omitempty"`
	BuildNumber *int64         `json:"buildNumber,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// Step is a CTRF test step. Custom properties belong in Extra.
type Step struct {
	Name   string         `json:"name"`
	Status TestStatus     `json:"status"`
	Extra  map[string]any `json:"extra,omitempty"`
}

func validStatus(status TestStatus) bool {
	switch status {
	case TestPassed, TestFailed, TestSkipped, TestPending, TestOther:
		return true
	default:
		return false
	}
}

func (test *TestResult) Validate() []error {
	if errs := test.validateIdentity(); len(errs) > 0 {
		return errs
	}

	var errs []error
	if test.Name == "" {
		errs = append(errs, errors.New("name must be non-empty"))
	}
	if !validStatus(test.Status) {
		errs = append(errs, errors.New("invalid status"))
	}
	if test.Duration < 0 {
		errs = append(errs, errors.New("duration must be non-negative"))
	}
	if test.Start != 0 && test.Stop != 0 && test.Stop < test.Start {
		errs = append(errs, errors.New("stop precedes start"))
	}
	if test.Retries < 0 || test.Retries != len(test.RetryAttempts) {
		errs = append(errs, errors.New("retries must equal the number of previous attempts"))
	}
	failedBefore := false
	for i, attempt := range test.RetryAttempts {
		errs = append(errs, attempt.validate(i+1)...)
		if attempt.Status == TestFailed {
			failedBefore = true
		}
	}
	if test.Flaky && (test.Status != TestPassed || !failedBefore) {
		errs = append(errs, errors.New("flaky requires a final pass after a failed attempt"))
	}
	for _, step := range test.Steps {
		if !validStatus(step.Status) {
			errs = append(errs, errors.New("invalid step status"))
		}
	}
	if test.Screenshot != "" {
		if _, err := base64.StdEncoding.DecodeString(test.Screenshot); err != nil {
			errs = append(errs, fmt.Errorf("screenshot must be base64: %w", err))
		}
	}
	return errs
}

func (test *TestResult) validateIdentity() []error {
	if test.AttemptID != "" && strings.TrimSpace(test.AttemptID) == "" {
		return singleError("attemptId must be non-empty")
	}
	if test.TestID != "" && strings.TrimSpace(test.TestID) == "" {
		return singleError("identity must be non-empty")
	}
	if test.ExecutionID != "" && strings.TrimSpace(test.ExecutionID) == "" {
		return singleError("identity must be non-empty")
	}

	return nil
}

func (attempt *RetryAttempt) validate(number int) []error {
	if attempt.AttemptID != "" && strings.TrimSpace(attempt.AttemptID) == "" {
		return singleError("attemptId must be non-empty")
	}
	var errs []error
	if attempt.Attempt != number {
		errs = append(errs, errors.New("retry attempts must be contiguous starting at 1"))
	}
	if !validStatus(attempt.Status) {
		errs = append(errs, errors.New("invalid retry status"))
	}
	if attempt.Duration < 0 {
		errs = append(errs, errors.New("retry duration must be non-negative"))
	}
	if attempt.Start != 0 && attempt.Stop != 0 && attempt.Stop < attempt.Start {
		errs = append(errs, errors.New("retry stop precedes start"))
	}
	return errs
}
