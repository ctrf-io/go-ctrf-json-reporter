package fixtures

import (
	"encoding/json"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"strings"
	"testing"
)

func TestCapabilityInventory(t *testing.T) {
	m, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != len(s.Catalogue.Entries) {
		t.Fatal("capability inventory differs")
	}
	for _, e := range s.Catalogue.Entries {
		if _, err = s.Disposition(e, m); err != nil {
			t.Fatal(err)
		}
	}
}
func TestUnitFixtureEvidence(t *testing.T) {
	c, err := Unit()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Samples) != 10 || len(c.Repeated) != 2 || len(c.Shards) != 2 {
		t.Fatal("missing fixture inventory")
	}
	for _, sm := range c.Samples {
		if err = s.Validate(sm.Bytes); err != nil {
			t.Fatalf("%s: %v", sm.Name, err)
		}
		if strings.HasPrefix(sm.Name, "repeat-") && sm.Report["runId"] != nil {
			t.Fatal("omitted runId must stay omitted")
		}
		if sm.Name == "empty" && len(s.Tests(sm.Report)) != 0 {
			t.Fatal("empty fixture not empty")
		}
		if sm.Name == "legacy-build" {
			env := s.Obj(s.Results(sm.Report)["environment"])
			if env["buildNumber"] != nil || s.Obj(env["extra"])["buildNumber"] != "release-42" {
				t.Fatal("legacy build identifier lost")
			}
		}
	}
}
func TestSourceOracleKeepsFinalAttempt(t *testing.T) {
	o := Source(retryEvents("fail", "pass", "fail"))
	test := s.Obj(s.Tests(o)[0])
	if test["retries"] != float64(2) || test["status"] != "failed" || test["duration"] != float64(31) || len(s.Array(test["sourceAttempts"])) != 3 {
		t.Fatal(test)
	}
}
func TestSourceObserverRejectsDroppedTests(t *testing.T) {
	if err := checkNative("mixed", UnitEvents("pass"), ""); err == nil {
		t.Fatal("incomplete native observations accepted")
	}
}
func TestSourceObserverRejectsWrongOutcome(t *testing.T) {
	es := retryEvents("pass")
	if err := checkNative("recovery", es, "failed"); err == nil {
		t.Fatal("wrong native outcome accepted")
	}
}
func TestRetryConformanceWithoutExceptions(t *testing.T) {
	c, err := Unit()
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{
		"conformance/property/results.summary.passed",
		"conformance/property/results.summary.failed",
		"conformance/property/results.tests[].duration",
		"conformance/property/results.tests[].retries",
		"conformance/report/summary-matches-final-results",
		"conformance/report/history-excludes-final-attempt",
		"conformance/report/flaky-matches-attempt-outcomes",
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			var entry *s.Entry
			for i := range s.Catalogue.Entries {
				if s.Catalogue.Entries[i].ID == id {
					entry = &s.Catalogue.Entries[i]
					break
				}
			}
			if entry == nil {
				t.Fatal("missing family", id)
			}
			if KnownIssue(*entry) != nil {
				t.Fatal("retry defect still exempted")
			}
			count := 0
			for _, sample := range c.Samples {
				if sample.Name != "recovery" && sample.Name != "exhaustion" && sample.Name != "regression" {
					continue
				}
				one := c.With(sample)
				if err := s.Check(*entry, &one); err != nil {
					t.Fatalf("%s: %v", sample.Name, err)
				}
				count++
			}
			if count != 3 {
				t.Fatal("missing retry evidence")
			}
		})
	}
}
func TestUnsupportedEmissionCannotHideSchemaViolation(t *testing.T) {
	c, err := Unit()
	if err != nil {
		t.Fatal(err)
	}
	o := s.Clone(c.Samples[0].Report)
	o["insights"] = true
	b, _ := json.Marshal(o)
	if s.Validate(b) == nil {
		t.Fatal("unsupported malformed output accepted")
	}
}
