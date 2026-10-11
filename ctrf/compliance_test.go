package ctrf

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExistingProperties(t *testing.T) {
	extra := map[string]any{"go-ctrf.custom": map[string]any{"nested": []any{1, true, "value"}}}
	report := NewReport("gotest", &Environment{
		AppName: "app", AppVersion: "1", OSPlatform: "linux", OSRelease: "release",
		OSVersion: "version", BuildName: "build", BuildNumber: "0", Extra: extra,
	})
	report.Extra, report.Results.Extra, report.Results.Tool.Extra = extra, extra, extra
	report.Results.Tool.Version = "1"
	report.Results.Summary = &Summary{Tests: 1, Failed: 1, Suites: 1, Start: 1, Stop: 3, Extra: extra}
	report.Results.Tests = []*TestResult{{
		AttemptID: "final-attempt", Name: "test", Status: TestFailed, Duration: 1, Start: 2, Stop: 3, Suite: []string{"suite", "child"},
		Message: "message", Trace: "trace", RawStatus: "pass", Tags: []string{"unit"}, Type: "unit",
		Filepath: "test.go", Retries: 1, Browser: "browser", Device: "device", Screenshot: "aW1hZ2U=",
		Parameters: map[string]any{"value": 1}, Steps: []any{map[string]any{"name": "step", "status": TestPassed, "extra": extra}},
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

// Struct metadata and arbitrary step objects remain supported by the Go API.
func TestLegacyEnvironmentAPI(t *testing.T) {
	type metadata struct {
		Team string `json:"team"`
	}
	for _, value := range []string{"0", "7", "build-7"} {
		env := Environment{BuildNumber: value, Extra: metadata{Team: "qa"}}
		data, err := json.Marshal(env)
		require.NoError(t, err)
		if value == "build-7" {
			require.JSONEq(t, `{"extra":{"team":"qa","buildNumber":"build-7"}}`, string(data))
		} else {
			require.JSONEq(t, `{"buildNumber":`+value+`,"extra":{"team":"qa"}}`, string(data))
			var decoded Environment
			require.NoError(t, json.Unmarshal(data, &decoded))
			require.Equal(t, value, decoded.BuildNumber)
		}
		require.Equal(t, metadata{Team: "qa"}, env.Extra)
	}
	var legacy Environment
	require.NoError(t, json.Unmarshal([]byte(`{"buildNumber":"build-7"}`), &legacy))
	require.Equal(t, "build-7", legacy.BuildNumber)
	extra := map[string]any{"team": "qa"}
	_, err := json.Marshal(Environment{BuildNumber: "build-7", Extra: extra})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"team": "qa"}, extra)
	test := TestResult{
		Parameters: metadata{Team: "qa"}, Extra: metadata{Team: "qa"},
		Steps: []any{map[string]any{"name": "step", "status": "passed"}},
	}
	_, err = json.Marshal(test)
	require.NoError(t, err)
}
