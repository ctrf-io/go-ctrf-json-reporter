package reporter_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/ctrf-io/go-ctrf-json-reporter/reporter"
	"github.com/stretchr/testify/require"
)

func TestRetryFinalResult(t *testing.T) {
	cases := []struct {
		name                           string
		actions                        []string
		status                         ctrf.TestStatus
		passed, failed, skipped, flaky int
	}{
		{"recovery", []string{"fail", "pass"}, ctrf.TestPassed, 1, 0, 0, 1},
		{"exhaustion", []string{"fail", "fail"}, ctrf.TestFailed, 0, 1, 0, 0},
		{"regression", []string{"fail", "pass", "fail"}, ctrf.TestFailed, 0, 1, 0, 0},
		{"repeated recovery", []string{"fail", "pass", "fail", "pass"}, ctrf.TestPassed, 1, 0, 0, 1},
		{"pass repeated", []string{"pass", "pass"}, ctrf.TestPassed, 1, 0, 0, 0},
		{"skipped after failure", []string{"fail", "skip"}, ctrf.TestSkipped, 0, 0, 1, 0},
		{"recovery through skip", []string{"fail", "skip", "pass"}, ctrf.TestPassed, 1, 0, 0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var input bytes.Buffer
			encoder := json.NewEncoder(&input)
			base := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
			for i, action := range tc.actions {
				start := base.Add(time.Duration(i) * time.Second)
				for _, event := range []map[string]any{
					{"Time": start.Format(time.RFC3339Nano), "Action": "run", "Package": "retry/package", "Test": "TestAttempt"},
					{
						"Time": start.Add(time.Millisecond).Format(time.RFC3339Nano), "Action": "output",
						"Package": "retry/package", "Test": "TestAttempt", "Output": fmt.Sprintf("attempt %d\n", i+1),
					},
					{
						"Time": start.Add(time.Duration((i+1)*10) * time.Millisecond).Format(time.RFC3339Nano), "Action": action,
						"Package": "retry/package", "Test": "TestAttempt", "Elapsed": float64((i+1)*10) / 1000,
					},
				} {
					require.NoError(t, encoder.Encode(event))
				}
			}
			calls := 0
			report, err := reporter.ParseTestResultsWithOptions(&input, false, &ctrf.Environment{}, reporter.IdentityOptions{
				TestIDResolver: func(_, _ string) string { calls++; return fmt.Sprintf("identity-%d", calls) },
			})
			require.NoError(t, err)
			require.Len(t, report.Results.Tests, 1)
			result := report.Results.Tests[0]
			n := len(tc.actions)
			require.Equal(t, tc.status, result.Status)
			require.EqualValues(t, n*10, result.Duration)
			require.Equal(t, base.Add(time.Duration(n-1)*time.Second).UnixMilli(), result.Start)
			require.Equal(t, result.Start+int64(n*10), result.Stop)
			require.Equal(t, n-1, result.Retries)
			require.Len(t, result.RetryAttempts, n-1)
			require.Equal(t, "identity-1", result.TestID)
			require.NotEmpty(t, result.ExecutionID)
			require.Equal(t, tc.flaky == 1, result.Flaky)
			if tc.status == ctrf.TestFailed {
				require.Equal(t, fmt.Sprintf("attempt %d\n", n), result.Message)
			} else {
				require.Empty(t, result.Message)
			}
			ids := map[string]bool{result.AttemptID: true}
			require.NotEmpty(t, result.AttemptID)
			statuses := map[string]ctrf.TestStatus{"pass": ctrf.TestPassed, "fail": ctrf.TestFailed, "skip": ctrf.TestSkipped}
			for i, attempt := range result.RetryAttempts {
				require.Equal(t, i+1, attempt.Attempt)
				require.Equal(t, statuses[tc.actions[i]], attempt.Status)
				require.EqualValues(t, (i+1)*10, attempt.Duration)
				require.Equal(t, base.Add(time.Duration(i)*time.Second).UnixMilli(), attempt.Start)
				require.Equal(t, attempt.Start+int64((i+1)*10), attempt.Stop)
				if tc.actions[i] == "fail" {
					require.Equal(t, fmt.Sprintf("attempt %d\n", i+1), attempt.Message)
				} else {
					require.Empty(t, attempt.Message)
				}
				require.NotEmpty(t, attempt.AttemptID)
				require.False(t, ids[attempt.AttemptID], "duplicate attempt ID")
				ids[attempt.AttemptID] = true
			}
			summary := report.Results.Summary
			require.Equal(t, 1, summary.Tests)
			require.Equal(t, tc.passed, summary.Passed)
			require.Equal(t, tc.failed, summary.Failed)
			require.Equal(t, tc.skipped, summary.Skipped)
			require.Equal(t, tc.flaky, summary.Flaky)
			require.Empty(t, summary.Validate())
			hadFailure := false
			for _, action := range tc.actions {
				if action == "fail" {
					hadFailure = true
				}
			}
			extra, ok := report.Results.Extra.(map[string]any)
			require.True(t, ok)
			require.Equal(t, hadFailure, extra["FailedBuild"] == true)
		})
	}
}
