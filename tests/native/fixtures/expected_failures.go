package fixtures

import (
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"strings"
)

// Public follow-up for the reproduced retry contract defects.
// These records describe observed defects; the shared assertions stay unchanged.
func KnownIssue(e s.Entry) *s.Issue {
	var names []string
	switch e.ID {
	case "conformance/property/results.summary.passed":
		names = []string{"recovery"}
	case "conformance/property/results.summary.failed":
		names = []string{"regression"}
	case "conformance/property/results.tests[].duration", "conformance/property/results.tests[].retries", "conformance/report/history-excludes-final-attempt":
		names = []string{"recovery", "exhaustion", "regression"}
	case "conformance/report/summary-matches-final-results":
		names = []string{"recovery", "regression"}
	case "conformance/report/flaky-matches-attempt-outcomes":
		names = []string{"regression"}
	default:
		return nil
	}
	return &s.Issue{Key: "go-retry-contract", Reason: "Repeated test events retain legacy retry history, cumulative duration, counters and flaky state.", FollowUp: "https://github.com/ctrf-io/go-ctrf-json-reporter/issues/34", Samples: map[string][]string{"unit": names, "integration": names}}
}
func MatchKnownFailure(e s.Entry, sm s.Sample, issue s.Issue, err error) bool {
	if issue.Key != "go-retry-contract" {
		return false
	}
	m, ok := err.(*s.Mismatch)
	if !ok {
		return false
	}
	exp := s.Tests(sm.Expected)
	if len(exp) != 1 {
		return false
	}
	source := s.Obj(exp[0])
	attempts := s.Array(source["sourceAttempts"])
	wantStatuses := map[string][]string{"recovery": {"failed", "passed"}, "exhaustion": {"failed", "failed"}, "regression": {"failed", "passed", "failed"}}[sm.Name]
	if len(attempts) != len(wantStatuses) || len(attempts) < 2 {
		return false
	}
	total := 0.
	for i, a := range attempts {
		if s.Obj(a)["status"] != wantStatuses[i] {
			return false
		}
		total += s.Num(s.Obj(a)["duration"])
	}
	var code string
	var actual, expected interface{}
	switch e.ID {
	case "conformance/property/results.summary.passed":
		code = "mapping:results.summary.passed"
		actual = []interface{}{float64(0)}
		expected = []interface{}{float64(1)}
	case "conformance/property/results.summary.failed":
		code = "mapping:results.summary.failed"
		actual = []interface{}{float64(0)}
		expected = []interface{}{float64(1)}
	case "conformance/property/results.tests[].duration":
		code = "mapping:results.tests[].duration"
		actual = []interface{}{total}
		expected = []interface{}{s.Obj(attempts[len(attempts)-1])["duration"]}
	case "conformance/property/results.tests[].retries":
		code = "mapping:results.tests[].retries"
		actual = []interface{}{float64(len(attempts))}
		expected = []interface{}{float64(len(attempts) - 1)}
	case "conformance/report/summary-matches-final-results":
		code = "summary:passed"
		if sm.Name == "regression" {
			code = "summary:failed"
		}
		actual = float64(0)
		expected = float64(1)
	case "conformance/report/history-excludes-final-attempt":
		code = "history-length"
		actual = float64(len(attempts))
		expected = float64(len(attempts) - 1)
	case "conformance/report/flaky-matches-attempt-outcomes":
		code = "flaky-matches-attempt-outcomes"
		actual = true
		expected = false
	default:
		return false
	}
	return m.Code == code && s.Equal(m.Actual, actual) && s.Equal(m.Expected, expected) && strings.HasPrefix(e.ID, "conformance/")
}
