package ctrf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExistingProperties(t *testing.T) {
	build := int64(0)
	extra := map[string]any{"go-ctrf.custom": map[string]any{"nested": []any{1, true, "value"}}}
	report := NewReport("gotest", &Environment{
		AppName: "app", AppVersion: "1", OSPlatform: "linux", OSRelease: "release",
		OSVersion: "version", BuildName: "build", BuildNumber: &build, Extra: extra,
	})
	report.Extra, report.Results.Extra, report.Results.Tool.Extra = extra, extra, extra
	report.Results.Tool.Version = "1"
	report.Results.Summary = &Summary{Tests: 1, Passed: 1, Flaky: 1, Suites: 1, Start: 1, Stop: 3, Extra: extra}
	report.Results.Tests = []*TestResult{{
		AttemptID: "final-attempt", Name: "test", Status: TestPassed, Duration: 1, Start: 2, Stop: 3, Suite: []string{"suite", "child"},
		Message: "message", Trace: "trace", RawStatus: "pass", Tags: []string{"unit"}, Type: "unit",
		Filepath: "test.go", Retries: 1, Flaky: true, Browser: "browser", Device: "device", Screenshot: "aW1hZ2U=",
		Parameters: map[string]any{"value": 1}, Steps: []Step{{Name: "step", Status: TestPassed, Extra: extra}},
		RetryAttempts: []RetryAttempt{{
			AttemptID: "previous-attempt", Attempt: 1, Status: TestFailed, Duration: 1, Message: "failure",
			Trace: "trace", Line: 1, Snippet: "snippet", Stdout: []string{"out"}, Stderr: []string{"err"}, Start: 1, Stop: 2, Extra: extra,
		}},
		Extra: extra,
	}}
	data, err := report.ToJsonPretty()
	require.NoError(t, err)
	require.Contains(t, data, `"buildNumber": 0`)
	// Optional local integration output, validated by the actual CLI outside this test.
	if output := os.Getenv("CTRF_COMPLIANCE_OUTPUT"); output != "" {
		require.NoError(t, os.WriteFile(output, []byte(data), 0o600)) //nolint:gosec // Explicit local test output path supplied by the developer.
	}
}

func TestInvalidExistingProperties(t *testing.T) {
	cases := map[string]func(*Report){
		"format":    func(r *Report) { r.ReportFormat = "other" },
		"version":   func(r *Report) { r.SpecVersion = "bad" },
		"reportId":  func(r *Report) { r.ReportId = "bad" },
		"nil test":  func(r *Report) { r.Results.Tests[0] = nil },
		"name":      func(r *Report) { r.Results.Tests[0].Name = "" },
		"status":    func(r *Report) { r.Results.Tests[0].Status = "pass" },
		"duration":  func(r *Report) { r.Results.Tests[0].Duration = -1 },
		"attemptId": func(r *Report) { r.Results.Tests[0].AttemptID = "   " },
		"retries":   func(r *Report) { r.Results.Tests[0].Retries = 1 },
		"retry sequence": func(r *Report) {
			r.Results.Tests[0].Retries = 1
			r.Results.Tests[0].RetryAttempts = []RetryAttempt{{Attempt: 2, Status: TestFailed}}
		},
		"retry status": func(r *Report) {
			r.Results.Tests[0].Retries = 1
			r.Results.Tests[0].RetryAttempts = []RetryAttempt{{Attempt: 1, Status: "bad"}}
		},
		"retry duration": func(r *Report) {
			r.Results.Tests[0].Retries = 1
			r.Results.Tests[0].RetryAttempts = []RetryAttempt{{Attempt: 1, Status: TestFailed, Duration: -1}}
		},
		"flaky":       func(r *Report) { r.Results.Tests[0].Flaky = true },
		"step status": func(r *Report) { r.Results.Tests[0].Steps = []Step{{Name: "step", Status: "bad"}} },
		"screenshot":  func(r *Report) { r.Results.Tests[0].Screenshot = "https://example.com/image.png" },
		"extra value": func(r *Report) { r.Extra = map[string]any{"bad": make(chan int)} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReport("gotest", nil)
			r.Results.Summary.Tests, r.Results.Summary.Passed = 1, 1
			r.Results.Tests = []*TestResult{{Name: "test", Status: TestPassed}}
			mutate(r)
			require.NotEmpty(t, r.Validate())
			var output bytes.Buffer
			require.Error(t, r.Write(&output, false))
			require.Empty(t, output.String())
		})
	}
}

func TestInvalidReportDoesNotTruncateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))
	require.Error(t, (&Report{}).WriteFile(path))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "original", string(content))
}
