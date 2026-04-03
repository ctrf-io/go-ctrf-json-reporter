package reporter_test

import (
	"bytes"
	"testing"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/ctrf-io/go-ctrf-json-reporter/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Enrich_Reporter(t *testing.T) {
	expected := &ctrf.Report{Results: &ctrf.Results{Tests: []*ctrf.TestResult{
		{
			Name:     "Test_Enrich_Reporter",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1740874081832,
			Stop:     1740874081832,
		},
	}}}

	//nolint:lll // The test inputs are raw strings taken from real test runs
	input := `{"Time":"2025-03-02T01:08:01.832222033+01:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter"}
{"Time":"2025-03-02T01:08:01.832309292+01:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter"}
{"Time":"2025-03-02T01:08:01.832321979+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Output":"=== RUN   Test_Enrich_Reporter\n"}
{"Time":"2025-03-02T01:08:01.832333869+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Output":"--- PASS: Test_Enrich_Reporter (0.00s)\n"}
{"Time":"2025-03-02T01:08:01.832339962+01:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Elapsed":0}
{"Time":"2025-03-02T01:08:01.832347177+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Output":"PASS\n"}
{"Time":"2025-03-02T01:08:01.83235318+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Output":"ok  \tgithub.com/ctrf-io/go-ctrf-json-reporter/reporter\t(cached)\n"}
{"Time":"2025-03-02T01:08:01.832359242+01:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Elapsed":0}`

	actual, err := reporter.ParseTestResults(bytes.NewBufferString(input), false, &ctrf.Environment{})

	require.NoError(t, err)
	assert.Equal(t, expected.Results.Tests, actual.Results.Tests)
}

func Test_Enrich_ReporterWithUnorderedMessages(t *testing.T) {
	expected := &ctrf.Report{Results: &ctrf.Results{Tests: []*ctrf.TestResult{
		{
			Name:     "Test_Enrich_Reporter/Test1",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter/Test2",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter/Test3",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter/Test4",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter/Test5",
			Status:   "failed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Message:  "=== RUN   Test_Enrich_Reporter/Test5\n    reporter:59: Something.Skip() = false, want true\n    --- FAIL: Test_Enrich_Reporter/Test5 (0.00s)\n",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter/Test6",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter/Test7",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
		{
			Name:     "Test_Enrich_Reporter",
			Status:   "failed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Message:  "=== RUN   Test_Enrich_Reporter\n--- FAIL: Test_Enrich_Reporter (0.00s)\n",
			Start:    1760718477126,
			Stop:     1760718477126,
		},
	}}}

	//nolint:lll // The test inputs are raw strings taken from real test runs
	input := `{"Time":"2025-10-17T12:27:57.126761-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter"}
{"Time":"2025-10-17T12:27:57.126764-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Output":"=== RUN   Test_Enrich_Reporter\n"}
{"Time":"2025-10-17T12:27:57.126769-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test1"}
{"Time":"2025-10-17T12:27:57.126771-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test1","Output":"=== RUN   Test_Enrich_Reporter/Test1\n"}
{"Time":"2025-10-17T12:27:57.126779-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test2"}
{"Time":"2025-10-17T12:27:57.126782-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test2","Output":"=== RUN   Test_Enrich_Reporter/Test2\n"}
{"Time":"2025-10-17T12:27:57.126785-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test3"}
{"Time":"2025-10-17T12:27:57.126788-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test3","Output":"=== RUN   Test_Enrich_Reporter/Test3\n"}
{"Time":"2025-10-17T12:27:57.1268-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test4"}
{"Time":"2025-10-17T12:27:57.126803-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test4","Output":"=== RUN   Test_Enrich_Reporter/Test4\n"}
{"Time":"2025-10-17T12:27:57.126809-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test5"}
{"Time":"2025-10-17T12:27:57.126815-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test5","Output":"=== RUN   Test_Enrich_Reporter/Test5\n"}
{"Time":"2025-10-17T12:27:57.126818-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test5","Output":"    reporter:59: Something.Skip() = false, want true\n"}
{"Time":"2025-10-17T12:27:57.126821-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test6"}
{"Time":"2025-10-17T12:27:57.126824-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test6","Output":"=== RUN   Test_Enrich_Reporter/Test6\n"}
{"Time":"2025-10-17T12:27:57.126827-04:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test7"}
{"Time":"2025-10-17T12:27:57.12683-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test7","Output":"=== RUN   Test_Enrich_Reporter/Test7\n"}
{"Time":"2025-10-17T12:27:57.126839-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Output":"--- FAIL: Test_Enrich_Reporter (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126842-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test1","Output":"    --- PASS: Test_Enrich_Reporter/Test1 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126845-04:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test1","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126848-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test2","Output":"    --- PASS: Test_Enrich_Reporter/Test2 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126851-04:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test2","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126854-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test3","Output":"    --- PASS: Test_Enrich_Reporter/Test3 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126858-04:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test3","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126861-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test4","Output":"    --- PASS: Test_Enrich_Reporter/Test4 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126864-04:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test4","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126867-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test5","Output":"    --- FAIL: Test_Enrich_Reporter/Test5 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.12687-04:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test5","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126873-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test6","Output":"    --- PASS: Test_Enrich_Reporter/Test6 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126876-04:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test6","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126879-04:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test7","Output":"    --- PASS: Test_Enrich_Reporter/Test7 (0.00s)\n"}
{"Time":"2025-10-17T12:27:57.126881-04:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter/Test7","Elapsed":0}
{"Time":"2025-10-17T12:27:57.126884-04:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Elapsed":0}`

	actual, err := reporter.ParseTestResults(bytes.NewBufferString(input), false, &ctrf.Environment{})

	require.NoError(t, err)
	assert.Equal(t, expected.Results.Tests, actual.Results.Tests)
}

func TestDetectFlakyTests(t *testing.T) {
	expected := &ctrf.Report{Results: &ctrf.Results{
		Summary: &ctrf.Summary{
			Tests:   4,
			Passed:  1,
			Failed:  1,
			Skipped: 1,
			Flaky:   1,
			Start:   1775245677812, // The event timestamp of the first processed event
			Stop:    1775245679646, // The event timestamp of the last processed event
		},
		Extra: map[string]any{
			"FailedBuild": true,
		},
		Tests: []*ctrf.TestResult{
			{
				Name:     "Test_Flaky_Pass",
				Status:   ctrf.TestPassed,
				Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky",
				Filepath: "reporter_test.go", // Filepath enrichment is based on grepping the current path, so this file is the one that gets found
				Start:    1775245677812,
				Stop:     1775245677863,
				Duration: 50,
			},
			{
				Name:     "Test_Flaky_Fail",
				Status:   ctrf.TestFailed,
				Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky",
				Filepath: "reporter_test.go",
				Retries:  3,
				Start:    1775245677863, // The start time of the first event for the first retry
				Stop:     1775245679247, // The stop time of the last event for the last retry
				Duration: 150,
				Message:  "",
				RetryAttempts: []ctrf.RetryAttempt{
					{
						Attempt: 1, Status: ctrf.TestFailed, Start: 1775245677863, Stop: 1775245677914, Duration: 50,
						Message: "=== RUN   Test_Flaky_Fail\n    flaky_test.go:21: This test is designed to fail.\n--- FAIL: Test_Flaky_Fail (0.05s)\n",
					},
					{
						Attempt: 2, Status: ctrf.TestFailed, Start: 1775245678350, Stop: 1775245678401, Duration: 50,
						Message: "=== RUN   Test_Flaky_Fail\n    flaky_test.go:21: This test is designed to fail.\n--- FAIL: Test_Flaky_Fail (0.05s)\n",
					},
					{
						Attempt: 3, Status: ctrf.TestFailed, Start: 1775245679196, Stop: 1775245679247, Duration: 50,
						Message: "=== RUN   Test_Flaky_Fail\n    flaky_test.go:21: This test is designed to fail.\n--- FAIL: Test_Flaky_Fail (0.05s)\n",
					},
				},
			},
			{
				Name:     "Test_Flaky_Skipped",
				Status:   ctrf.TestSkipped,
				Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky",
				Filepath: "reporter_test.go",
				Start:    1775245677914,
				Stop:     1775245677914,
			},
			{
				Name:     "Test_Flaky_Flaky",
				Status:   ctrf.TestPassed,
				Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky",
				Filepath: "reporter_test.go",
				Retries:  3,
				Flaky:    true,
				Duration: 150,
				Start:    1775245677914,
				Stop:     1775245679646,
				RetryAttempts: []ctrf.RetryAttempt{
					{
						Attempt: 1, Status: ctrf.TestFailed, Duration: 50, Start: 1775245677914, Stop: 1775245677967,
						Message: "=== RUN   Test_Flaky_Flaky\n    flaky_test.go:37: Flaky Failure (attempt 1)\n--- FAIL: Test_Flaky_Flaky (0.05s)\n",
					},
					{
						Attempt: 2, Status: ctrf.TestFailed, Duration: 50, Start: 1775245678784, Stop: 1775245678837,
						Message: "=== RUN   Test_Flaky_Flaky\n    flaky_test.go:54: Flaky Failure (attempt 2)\n--- FAIL: Test_Flaky_Flaky (0.05s)\n",
					},
					{Attempt: 3, Status: ctrf.TestPassed, Duration: 50, Start: 1775245679595, Stop: 1775245679646},
				},
			},
		},
	}}

	//nolint:lll // The test inputs are raw strings taken from real test runs
	input := `{"Time":"2026-04-03T13:47:57.561046-06:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky"}
{"Time":"2026-04-03T13:47:57.812415-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Pass"}
{"Time":"2026-04-03T13:47:57.812541-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Pass","Output":"=== RUN   Test_Flaky_Pass\n"}
{"Time":"2026-04-03T13:47:57.863054-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Pass","Output":"--- PASS: Test_Flaky_Pass (0.05s)\n"}
{"Time":"2026-04-03T13:47:57.863087-06:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Pass","Elapsed":0.05}
{"Time":"2026-04-03T13:47:57.863153-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail"}
{"Time":"2026-04-03T13:47:57.863169-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"=== RUN   Test_Flaky_Fail\n"}
{"Time":"2026-04-03T13:47:57.914515-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"    flaky_test.go:21: This test is designed to fail.\n"}
{"Time":"2026-04-03T13:47:57.914602-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"--- FAIL: Test_Flaky_Fail (0.05s)\n"}
{"Time":"2026-04-03T13:47:57.914618-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Elapsed":0.05}
{"Time":"2026-04-03T13:47:57.914632-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Skipped"}
{"Time":"2026-04-03T13:47:57.914639-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Skipped","Output":"=== RUN   Test_Flaky_Skipped\n"}
{"Time":"2026-04-03T13:47:57.914663-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Skipped","Output":"    flaky_test.go:25: This test is designed to be skipped.\n"}
{"Time":"2026-04-03T13:47:57.914686-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Skipped","Output":"--- SKIP: Test_Flaky_Skipped (0.00s)\n"}
{"Time":"2026-04-03T13:47:57.914707-06:00","Action":"skip","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Skipped","Elapsed":0}
{"Time":"2026-04-03T13:47:57.914718-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky"}
{"Time":"2026-04-03T13:47:57.914725-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"=== RUN   Test_Flaky_Flaky\n"}
{"Time":"2026-04-03T13:47:57.96746-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"    flaky_test.go:37: Flaky Failure (attempt 1)\n"}
{"Time":"2026-04-03T13:47:57.967528-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"--- FAIL: Test_Flaky_Flaky (0.05s)\n"}
{"Time":"2026-04-03T13:47:57.967543-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Elapsed":0.05}
{"Time":"2026-04-03T13:47:57.967559-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\n"}
{"Time":"2026-04-03T13:47:57.969244-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\tgithub.com/ctrf-io/go-ctrf-json-reporter/examples/flaky\t0.408s\n"}
{"Time":"2026-04-03T13:47:57.969316-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Elapsed":0.408}
{"Time":"2026-04-03T13:47:58.151436-06:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky"}
{"Time":"2026-04-03T13:47:58.350514-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail"}
{"Time":"2026-04-03T13:47:58.350555-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"=== RUN   Test_Flaky_Fail\n"}
{"Time":"2026-04-03T13:47:58.40176-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"    flaky_test.go:21: This test is designed to fail.\n"}
{"Time":"2026-04-03T13:47:58.401856-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"--- FAIL: Test_Flaky_Fail (0.05s)\n"}
{"Time":"2026-04-03T13:47:58.401873-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Elapsed":0.05}
{"Time":"2026-04-03T13:47:58.401904-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\n"}
{"Time":"2026-04-03T13:47:58.403834-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\tgithub.com/ctrf-io/go-ctrf-json-reporter/examples/flaky\t0.252s\n"}
{"Time":"2026-04-03T13:47:58.403863-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Elapsed":0.252}
{"Time":"2026-04-03T13:47:58.566312-06:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky"}
{"Time":"2026-04-03T13:47:58.784809-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky"}
{"Time":"2026-04-03T13:47:58.784889-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"=== RUN   Test_Flaky_Flaky\n"}
{"Time":"2026-04-03T13:47:58.837476-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"    flaky_test.go:54: Flaky Failure (attempt 2)\n"}
{"Time":"2026-04-03T13:47:58.837581-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"--- FAIL: Test_Flaky_Flaky (0.05s)\n"}
{"Time":"2026-04-03T13:47:58.837595-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Elapsed":0.05}
{"Time":"2026-04-03T13:47:58.837643-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\n"}
{"Time":"2026-04-03T13:47:58.839298-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\tgithub.com/ctrf-io/go-ctrf-json-reporter/examples/flaky\t0.273s\n"}
{"Time":"2026-04-03T13:47:58.839328-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Elapsed":0.273}
{"Time":"2026-04-03T13:47:58.994008-06:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky"}
{"Time":"2026-04-03T13:47:59.196497-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail"}
{"Time":"2026-04-03T13:47:59.196544-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"=== RUN   Test_Flaky_Fail\n"}
{"Time":"2026-04-03T13:47:59.247768-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"    flaky_test.go:21: This test is designed to fail.\n"}
{"Time":"2026-04-03T13:47:59.247873-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Output":"--- FAIL: Test_Flaky_Fail (0.05s)\n"}
{"Time":"2026-04-03T13:47:59.247889-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Fail","Elapsed":0.05}
{"Time":"2026-04-03T13:47:59.247917-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\n"}
{"Time":"2026-04-03T13:47:59.250129-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"FAIL\tgithub.com/ctrf-io/go-ctrf-json-reporter/examples/flaky\t0.256s\n"}
{"Time":"2026-04-03T13:47:59.250164-06:00","Action":"fail","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Elapsed":0.256}
{"Time":"2026-04-03T13:47:59.413168-06:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky"}
{"Time":"2026-04-03T13:47:59.595591-06:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky"}
{"Time":"2026-04-03T13:47:59.595652-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"=== RUN   Test_Flaky_Flaky\n"}
{"Time":"2026-04-03T13:47:59.646973-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Output":"--- PASS: Test_Flaky_Flaky (0.05s)\n"}
{"Time":"2026-04-03T13:47:59.646984-06:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Test":"Test_Flaky_Flaky","Elapsed":0.05}
{"Time":"2026-04-03T13:47:59.646998-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"PASS\n"}
{"Time":"2026-04-03T13:47:59.648315-06:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Output":"ok  \tgithub.com/ctrf-io/go-ctrf-json-reporter/examples/flaky\t0.235s\n"}
{"Time":"2026-04-03T13:47:59.648334-06:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/examples/flaky","Elapsed":0.235}`

	actual, err := reporter.ParseTestResults(bytes.NewBufferString(input), true, &ctrf.Environment{})

	require.NoError(t, err)
	assert.Equal(t, expected.Results.Summary, actual.Results.Summary)
	assert.Equal(t, expected.Results.Tests, actual.Results.Tests)
	assert.Equal(t, expected.Results.Extra, actual.Results.Extra)
}
