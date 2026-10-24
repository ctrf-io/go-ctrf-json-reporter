package fixtures

import (
	"encoding/json"
	"errors"
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
func TestNativeExpectedFailureMatchers(t *testing.T) {
	c, err := Unit()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range s.Catalogue.Entries {
		issue := KnownIssue(e)
		if issue == nil {
			continue
		}
		e := e
		t.Run(e.ID, func(t *testing.T) {
			for _, name := range issue.Samples["unit"] {
				var sm s.Sample
				for _, candidate := range c.Samples {
					if candidate.Name == name {
						sm = candidate
					}
				}
				one := c.With(sm)
				err := s.Check(e, &one)
				if err == nil || !MatchKnownFailure(e, sm, *issue, err) {
					t.Fatalf("recorded defect did not reproduce: %s: %v", name, err)
				}
				if MatchKnownFailure(e, sm, *issue, errors.New("infrastructure")) {
					t.Fatal("infrastructure swallowed")
				}
				m, ok := err.(*s.Mismatch)
				if !ok {
					t.Fatal(err)
				}
				changed := *m
				changed.Expected = "changed"
				if MatchKnownFailure(e, sm, *issue, &changed) {
					t.Fatal("changed mismatch swallowed")
				}
				fixed := sm
				fixed.Name = "unrecorded"
				if MatchKnownFailure(e, fixed, *issue, err) {
					t.Fatal("unrecorded sample swallowed")
				}
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
