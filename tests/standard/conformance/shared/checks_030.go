package shared

import (
	"fmt"
	"sort"
	"strings"
)

// Intrinsic lineage constraints apply even when provenance is unsupported.
func checkProvenanceIdentities(report Object) error {
	seen := map[string]bool{}
	self, _ := report["reportId"].(string)
	for _, input := range Array(Obj(report["provenance"])["inputs"]) {
		id, ok := Obj(input)["reportId"].(string)
		if !ok {
			return failure("provenance reportId must be a string")
		}
		id = strings.ToLower(id)
		if seen[id] {
			return failure("duplicate provenance reportId")
		}
		if id == strings.ToLower(self) {
			return failure("provenance references its own reportId")
		}
		seen[id] = true
	}
	return nil
}

// A nil source expectation explicitly requires omission. It is independent
// oracle data, never serialized into a subject report as JSON null.
func matchSourceValue(actual, expected Object, key, label string) (bool, error) {
	value, exists := expected[key]
	if !exists {
		return false, nil
	}
	got, emitted := actual[key]
	if value == nil {
		if emitted {
			return false, failure(label + ": expected omission")
		}
		return false, nil
	}
	return true, expect(label, got, value)
}
func checkAttemptEvidence(c *Context, key string) error {
	witnessed := false
	for _, sample := range c.Samples {
		for index, value := range Tests(sample.Report) {
			test := Obj(value)
			var expected Object
			if index < len(Tests(sample.Expected)) {
				expected = Obj(Tests(sample.Expected)[index])
			}
			label := fmt.Sprintf("%s: %v %s", sample.Name, test["name"], key)
			positive, err := matchSourceValue(test, expected, key, label)
			if err != nil {
				return err
			}
			witnessed = witnessed || positive
			for i, value := range Array(expected["retryAttempts"]) {
				attempt := Obj(value)
				if _, exists := attempt[key]; !exists {
					continue
				}
				if i >= len(children(test)) {
					return failure(label + ": missing history attempt")
				}
				positive, err = matchSourceValue(Obj(children(test)[i]), attempt, key, fmt.Sprintf("%s (attempt %d)", label, i+1))
				if err != nil {
					return err
				}
				witnessed = witnessed || positive
			}
		}
	}
	if !witnessed {
		return failure("missing positive native attempt evidence for " + key)
	}
	return nil
}
func checkNewRule(rule string, c *Context) (bool, error) {
	switch rule {
	case "assertion-failures-belong-to-attempt":
		return true, checkAttemptEvidence(c, "assertionFailures")
	case "thread-ids-belong-to-attempt":
		return true, checkAttemptEvidence(c, "threadId")
	case "provenance-identities-are-distinct", "provenance-records-direct-sources", "global-errors-belong-to-run", "global-error-merge-preserves-inputs":
	default:
		return false, nil
	}
	witnessed := false
	for _, sample := range c.Samples {
		report := sample.Report
		switch rule {
		case "provenance-identities-are-distinct":
			witnessed = witnessed || report["provenance"] != nil
			if err := checkProvenanceIdentities(report); err != nil {
				return true, err
			}
		case "provenance-records-direct-sources":
			if sample.DirectSources == nil {
				continue
			}
			if len(sample.DirectSources) == 0 {
				return true, failure("missing direct source evidence")
			}
			ids := map[string]bool{}
			unidentified := false
			for _, source := range sample.DirectSources {
				id, ok := source["reportId"].(string)
				if !ok {
					unidentified = true
				} else {
					ids[strings.ToLower(id)] = true
				}
			}
			if unidentified {
				if _, exists := report["provenance"]; exists {
					return true, failure("omit lineage when a source lacks an ID")
				}
				continue
			}
			if report["provenance"] == nil {
				return true, failure("missing provenance for implemented scenario")
			}
			if err := checkProvenanceIdentities(report); err != nil {
				return true, err
			}
			expected := []string{}
			for id := range ids {
				expected = append(expected, id)
			}
			actual := []string{}
			for _, input := range Array(Obj(report["provenance"])["inputs"]) {
				id, ok := Obj(input)["reportId"].(string)
				if !ok {
					return true, failure("invalid provenance input identity")
				}
				actual = append(actual, strings.ToLower(id))
			}
			sort.Strings(expected)
			sort.Strings(actual)
			if err := expect("record all and only direct source IDs", actual, expected); err != nil {
				return true, err
			}
			if id, _ := report["reportId"].(string); id == "" {
				c.Warnings = append(c.Warnings, sample.Name+": a derived report should have a new reportId (SHOULD, Core §20.1)")
			}
			witnessed = true
		case "global-errors-belong-to-run":
			expected := Results(sample.Expected)
			positive, err := matchSourceValue(Results(report), expected, "errors", sample.Name+": run error mapping")
			if err != nil {
				return true, err
			}
			witnessed = witnessed || positive
			if _, exists := expected["errors"]; exists && Results(report)["errors"] != nil {
				if _, ok := expected["tests"].([]interface{}); !ok {
					return true, failure("missing independent test case evidence")
				}
				names := func(tests []interface{}) []interface{} {
					result := []interface{}{}
					for _, test := range tests {
						result = append(result, Obj(test)["name"])
					}
					return result
				}
				if err := expect("global errors must not become test results", names(Tests(report)), names(Tests(sample.Expected))); err != nil {
					return true, err
				}
			}
		case "global-error-merge-preserves-inputs":
			if len(sample.DirectSources) < 2 {
				continue
			}
			expected := []interface{}{}
			for _, source := range sample.DirectSources {
				expected = append(expected, Array(Results(source)["errors"])...)
			}
			if len(expected) == 0 {
				continue
			}
			witnessed = true
			extra := append([]interface{}{}, Array(Results(report)["errors"])...)
			missing := false
			for _, value := range expected {
				index := -1
				for i, candidate := range extra {
					if Equal(candidate, value) {
						index = i
						break
					}
				}
				if index < 0 {
					missing = true
				} else {
					extra = append(extra[:index], extra[index+1:]...)
				}
			}
			if missing || len(extra) > 0 {
				c.Warnings = append(c.Warnings, sample.Name+": merge should retain all input errors without deduplicating (SHOULD, Core §6.5)")
			}
		}
	}
	if !witnessed {
		return true, failure("missing positive evidence for " + rule)
	}
	return true, nil
}
