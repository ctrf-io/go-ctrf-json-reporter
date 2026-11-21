package checker_test

import (
	"encoding/json"
	"strings"
	"testing"

	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
)

var controls030 = []string{
	"optional-fields-accept-omission-and-valid-emission",
	"rejects-description-even-without-capability",
	"rejects-retry-thread-even-without-capability",
	"rejects-empty-assertions-even-without-capability",
	"rejects-unformatted-comparison-even-without-capability",
	"rejects-empty-global-error-even-without-capability",
	"rejects-missing-provenance-inputs-even-without-capability",
	"rejects-bad-source-uuid-even-without-capability",
	"duplicate-identity-ignores-case-and-input-metadata",
	"no-self-reference-even-with-different-case",
	"direct-lineage-excludes-flattened-ancestors-and-omits-unidentified-sources",
	"assertion-and-thread-mapping-uses-independent-attempt-evidence",
	"global-errors-do-not-become-tests-even-if-counts-are-consistent",
	"error-merge-multiplicity-is-advisory-but-evidence-is-required",
	"core-accepts-opaque-extension-values-without-extension-validation",
}

const sourceA = "abcdefab-e29b-41d4-a716-446655440001"
const sourceB = "abcdefab-e29b-41d4-a716-446655440002"
const derived = "abcdefab-e29b-41d4-a716-446655440003"

func inputs(ids ...string) s.Object {
	values := []interface{}{}
	for _, id := range ids {
		values = append(values, s.Object{"reportId": id})
	}
	return s.Object{"inputs": values}
}
func check030(t *testing.T, id string) {
	c := valid()
	sample := &c.Samples[0]
	report := sample.Report
	test := s.Obj(s.Tests(report)[0])
	history := s.Obj(s.Array(test["retryAttempts"])[0])
	validate := func() error {
		b, err := json.Marshal(report)
		if err != nil {
			return err
		}
		sample.Bytes = b
		return checkRule(&c, "single-json-object")
	}
	switch id {
	case "optional-fields-accept-omission-and-valid-emission":
		accepted(t, validate())
		accepted(t, checkRule(&c, "declares-supported-contract"))
		test["description"] = ""
		test["threadId"] = "host-b/worker-2"
		history["threadId"] = "host-a/worker-1"
		history["assertionFailures"] = []interface{}{s.Object{"expected": "", "extra": s.Object{}}}
		s.Results(report)["errors"] = []interface{}{s.Object{"extra": s.Object{}}}
		report["provenance"] = inputs(sourceA)
		s.Obj(report["provenance"])["extra"] = s.Object{}
		accepted(t, validate())
		for _, path := range []string{"results.tests[].description", "results.tests[].retryAttempts[].threadId", "results.tests[].retryAttempts[].assertionFailures[].expected", "results.errors[].extra", "provenance.inputs[].reportId"} {
			accepted(t, check(&c, path))
		}
	case "rejects-description-even-without-capability":
		test["description"] = nil
		rejected(t, validate())
	case "rejects-retry-thread-even-without-capability":
		history["threadId"] = float64(1)
		rejected(t, validate())
	case "rejects-empty-assertions-even-without-capability":
		test["assertionFailures"] = []interface{}{}
		rejected(t, validate())
	case "rejects-unformatted-comparison-even-without-capability":
		history["assertionFailures"] = []interface{}{s.Object{"actual": s.Object{"unformatted": true}}}
		rejected(t, validate())
	case "rejects-empty-global-error-even-without-capability":
		s.Results(report)["errors"] = []interface{}{s.Object{}}
		rejected(t, validate())
	case "rejects-missing-provenance-inputs-even-without-capability":
		report["provenance"] = s.Object{"extra": s.Object{}}
		rejected(t, validate())
	case "rejects-bad-source-uuid-even-without-capability":
		report["provenance"] = inputs("not-a-uuid")
		rejected(t, validate())
	case "duplicate-identity-ignores-case-and-input-metadata":
		report["provenance"] = inputs(sourceA, strings.ToUpper(sourceA))
		s.Obj(s.Array(s.Obj(report["provenance"])["inputs"])[1])["extra"] = s.Object{"shard": float64(2)}
		rejected(t, validate())
		rejected(t, checkRule(&c, "provenance-identities-are-distinct"))
	case "no-self-reference-even-with-different-case":
		report["reportId"] = sourceA
		report["provenance"] = inputs(strings.ToUpper(sourceA))
		rejected(t, validate())
	case "direct-lineage-excludes-flattened-ancestors-and-omits-unidentified-sources":
		report["reportId"] = derived
		sample.DirectSources = []s.Object{{"reportId": sourceA, "provenance": inputs("ancestor")}, {"reportId": sourceB}}
		report["provenance"] = inputs(sourceB, strings.ToUpper(sourceA))
		unidentified := valid().Samples[0]
		unidentified.DirectSources = []s.Object{{"reportId": sourceA}, {}}
		c.Samples = append(c.Samples, unidentified)
		run := func() error { return checkRule(&c, "provenance-records-direct-sources") }
		accepted(t, run())
		delete(report, "reportId")
		accepted(t, run())
		if len(c.Warnings) == 0 || !strings.Contains(c.Warnings[len(c.Warnings)-1], "SHOULD") {
			t.Fatal(c.Warnings)
		}
		s.Array(s.Obj(report["provenance"])["inputs"])[0] = s.Object{"reportId": "ancestor"}
		rejected(t, run())
		s.Array(s.Obj(report["provenance"])["inputs"])[0] = s.Object{"reportId": sourceB}
		c.Samples[1].Report["provenance"] = inputs(sourceA)
		rejected(t, run())
	case "assertion-and-thread-mapping-uses-independent-attempt-evidence":
		rejected(t, checkRule(&c, "assertion-failures-belong-to-attempt"))
		native := s.Obj(s.Tests(sample.Expected)[0])
		assertions := []interface{}{s.Object{"message": "comparison", "actual": ""}}
		native["assertionFailures"] = nil
		native["threadId"] = "host-b/worker-2"
		native["retryAttempts"] = []interface{}{s.Object{"threadId": "host-a/worker-1", "assertionFailures": assertions}}
		test["threadId"] = "host-b/worker-2"
		history["threadId"] = "host-a/worker-1"
		// Keep native and emitted data independent for mutation controls.
		history["assertionFailures"] = []interface{}{s.Object{"message": "comparison", "actual": ""}}
		accepted(t, checkRule(&c, "assertion-failures-belong-to-attempt"))
		accepted(t, checkRule(&c, "thread-ids-belong-to-attempt"))
		history["threadId"] = test["threadId"]
		rejected(t, checkRule(&c, "thread-ids-belong-to-attempt"))
		assertion := s.Obj(s.Array(history["assertionFailures"])[0])
		assertion["expected"] = "inferred"
		rejected(t, checkRule(&c, "assertion-failures-belong-to-attempt"))
		delete(assertion, "expected")
		test["assertionFailures"] = assertions
		rejected(t, checkRule(&c, "assertion-failures-belong-to-attempt"))
	case "global-errors-do-not-become-tests-even-if-counts-are-consistent":
		errors := []interface{}{s.Object{"message": "worker setup"}}
		s.Results(sample.Expected)["errors"] = errors
		s.Obj(s.Tests(sample.Expected)[0])["name"] = "case"
		s.Results(report)["errors"] = errors
		accepted(t, checkRule(&c, "global-errors-belong-to-run"))
		s.Results(report)["tests"] = append(s.Tests(report), s.Object{"name": "worker setup", "status": "failed", "duration": float64(0)})
		s.Summary(report)["tests"] = float64(2)
		s.Summary(report)["failed"] = float64(1)
		accepted(t, checkRule(&c, "summary-matches-final-results"))
		rejected(t, checkRule(&c, "global-errors-belong-to-run"))
	case "error-merge-multiplicity-is-advisory-but-evidence-is-required":
		run := func() error { return checkRule(&c, "global-error-merge-preserves-inputs") }
		rejected(t, run())
		value := s.Object{"message": "setup", "extra": s.Object{"retryable": true}}
		sample.DirectSources = []s.Object{{"results": s.Object{}}, {"results": s.Object{"errors": []interface{}{value, value}}}}
		s.Results(report)["errors"] = []interface{}{s.Clone(value), s.Clone(value)}
		accepted(t, run())
		if len(c.Warnings) != 0 {
			t.Fatal(c.Warnings)
		}
		s.Results(report)["errors"] = s.Array(s.Results(report)["errors"])[:1]
		accepted(t, run())
		if len(c.Warnings) != 1 || !strings.Contains(c.Warnings[0], "SHOULD") {
			t.Fatal(c.Warnings)
		}
	case "core-accepts-opaque-extension-values-without-extension-validation":
		report["extra"] = s.Object{"example/private": nil, "ctrf.future": s.Object{"extensionVersion": float64(42)}}
		test["extra"] = s.Object{"ctrf.future": []interface{}{"opaque", false}}
		accepted(t, validate())
	default:
		t.Fatal("unimplemented 0.3.0 control " + id)
	}
}
