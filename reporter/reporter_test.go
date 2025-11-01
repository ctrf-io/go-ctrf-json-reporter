package reporter_test

import (
	"bytes"
	"testing"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/ctrf-io/go-ctrf-json-reporter/reporter"
	"github.com/stretchr/testify/assert"
)

func Test_Enrich_Reporter(t *testing.T) {
	expected := &ctrf.Report{Results: &ctrf.Results{Tests: []*ctrf.TestResult{
		{
			Name:     "Test_Enrich_Reporter",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
	}}}
	input := `{"Time":"2025-03-02T01:08:01.832222033+01:00","Action":"start","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter"}
{"Time":"2025-03-02T01:08:01.832309292+01:00","Action":"run","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter"}
{"Time":"2025-03-02T01:08:01.832321979+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Output":"=== RUN   Test_Enrich_Reporter\n"}
{"Time":"2025-03-02T01:08:01.832333869+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Output":"--- PASS: Test_Enrich_Reporter (0.00s)\n"}
{"Time":"2025-03-02T01:08:01.832339962+01:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Test":"Test_Enrich_Reporter","Elapsed":0}
{"Time":"2025-03-02T01:08:01.832347177+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Output":"PASS\n"}
{"Time":"2025-03-02T01:08:01.83235318+01:00","Action":"output","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Output":"ok  \tgithub.com/ctrf-io/go-ctrf-json-reporter/reporter\t(cached)\n"}
{"Time":"2025-03-02T01:08:01.832359242+01:00","Action":"pass","Package":"github.com/ctrf-io/go-ctrf-json-reporter/reporter","Elapsed":0}`

	actual, err := reporter.ParseTestResults(bytes.NewBufferString(input), false, &ctrf.Environment{})

	assert.Nil(t, err)
	assert.Equal(t, expected.Results.Tests, actual.Results.Tests)
}

func Test_Enrich_ReporterWithUnorderedMessages(t *testing.T) {
	expected := &ctrf.Report{Results: &ctrf.Results{Tests: []*ctrf.TestResult{
		{
			Name:     "Test_Enrich_Reporter/Test1",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
		{
			Name:     "Test_Enrich_Reporter/Test2",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
		{
			Name:     "Test_Enrich_Reporter/Test3",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
		{
			Name:     "Test_Enrich_Reporter/Test4",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
		{
			Name:     "Test_Enrich_Reporter/Test5",
			Status:   "failed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Message:  "=== RUN   Test_Enrich_Reporter/Test5\n    reporter:59: Something.Skip() = false, want true\n    --- FAIL: Test_Enrich_Reporter/Test5 (0.00s)\n",
		},
		{
			Name:     "Test_Enrich_Reporter/Test6",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
		{
			Name:     "Test_Enrich_Reporter/Test7",
			Status:   "passed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
		},
		{
			Name:     "Test_Enrich_Reporter",
			Status:   "failed",
			Suite:    "github.com/ctrf-io/go-ctrf-json-reporter/reporter",
			Filepath: "reporter_test.go",
			Message:  "=== RUN   Test_Enrich_Reporter\n--- FAIL: Test_Enrich_Reporter (0.00s)\n",
		},
	}}}
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

	assert.Nil(t, err)
	assert.Equal(t, expected.Results.Tests, actual.Results.Tests)
}
