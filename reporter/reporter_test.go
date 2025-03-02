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
