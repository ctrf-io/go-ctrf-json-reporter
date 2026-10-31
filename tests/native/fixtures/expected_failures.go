package fixtures

import s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"

// Retry defects from issue #34 now receive ordinary conformance assertions.
func KnownIssue(s.Entry) *s.Issue                              { return nil }
func MatchKnownFailure(s.Entry, s.Sample, s.Issue, error) bool { return false }
