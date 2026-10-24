package checker_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func entry(kind, key string) s.Entry {
	for _, e := range s.Catalogue.Entries {
		if e.Kind == kind && (e.Path == key || e.Rule == key) {
			return e
		}
	}
	panic(key)
}
func valid() s.Context {
	o, err := s.Decode([]byte(`{"reportFormat":"CTRF","specVersion":"0.2.0","results":{"tool":{"name":"independent-control"},"summary":{"tests":1,"passed":1,"failed":0,"pending":0,"skipped":0,"other":0,"start":100,"stop":120},"tests":[{"name":"case","status":"passed","duration":12,"retries":1,"flaky":true,"attemptId":"final","retryAttempts":[{"attempt":1,"attemptId":"first","status":"failed"}]}]}}`))
	if err != nil {
		panic(err)
	}
	expected, _ := s.Decode([]byte(`{"results":{"summary":{"tests":1},"tests":[{"duration":12,"sourceAttempts":[{"status":"failed"},{"status":"passed","duration":12}]}]}}`))
	b, _ := json.Marshal(o)
	return s.Context{Samples: []s.Sample{{Name: "valid", Report: o, Bytes: b, Expected: expected}}}
}
func accepted(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func rejected(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("mutation unexpectedly accepted")
	}
}
func check(c *s.Context, path string) error     { return s.Check(entry("property", path), c) }
func checkRule(c *s.Context, name string) error { return s.Check(entry("report", name), c) }
func TestChecker(t *testing.T) {
	ids := []string{
		"schema/catalogue-covers-expanded-properties-and-relationships", "schema/accepts-independent-valid-document", "schema/rejects-unknown-property", "schema/rejects-wrong-type", "schema/rejects-missing-required-field", "schema/requires-positive-evidence", "schema/required-child-is-checked-when-parent-exists", "schema/capabilities-require-entry-and-reason", "schema/unsupported-declaration-cannot-hide-emitted-invalid-field", "specification/accepts-independent-retry-history",
		"specification/rejects-utf8-without-bom-mutation", "specification/rejects-summary-matches-final-results-mutation", "specification/rejects-history-orders-contiguous-attempts-mutation", "specification/rejects-retry-count-matches-history-length-mutation", "specification/rejects-history-excludes-final-attempt-mutation", "specification/rejects-flaky-matches-attempt-outcomes-mutation", "specification/rejects-attempt-identity-scoped-to-execution-mutation", "specification/rejects-respects-reserved-extension-ownership-mutation", "specification/rejects-source-duration-unit-mismatch", "specification/missing-contextual-evidence-fails", "specification/timing-recommendation-warns", "specification/total-count-recommendation-warns-but-bounds-fail",
		"expected-failure/accepts-only-recorded-mismatch", "expected-failure/rejects-unexpected-pass", "expected-failure/rejects-partial-fix", "expected-failure/rejects-different-value", "expected-failure/rejects-missing-sample", "expected-failure/rejects-duplicate-sample", "expected-failure/rejects-unrecorded-broken-sample", "expected-failure/rejects-infrastructure-error", "expected-failure/does-not-apply-at-unrecorded-level", "expected-failure/allows-optional-absence-outside-defect-samples", "expected-failure/rejects-unrelated-error-in-later-sample", "schema/shared-snapshot-matches-declared-files",
	}
	actualIDs := []string{}
	for _, id := range ids {
		actualIDs = append(actualIDs, "checker/"+id)
	}
	if !s.Equal(actualIDs, s.CheckerIDs) {
		t.Fatal("shared control inventory differs from reviewed source")
	}
	for _, id := range ids {
		id := id
		t.Run("checker/"+id, func(t *testing.T) {
			c := valid()
			o := c.Samples[0].Report
			test := s.Obj(s.Tests(o)[0])
			history := s.Obj(s.Array(test["retryAttempts"])[0])
			switch id {
			case "schema/catalogue-covers-expanded-properties-and-relationships":
				var schema s.Object
				accepted(t, json.Unmarshal(s.SchemaBytes, &schema))
				var paths []string
				var walk func(s.Object, string)
				walk = func(o s.Object, prefix string) {
					if ref, ok := o["$ref"].(string); ok {
						var v interface{} = schema
						for _, key := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
							v = s.Obj(v)[key]
						}
						o = s.Obj(v)
					}
					for key, v := range s.Obj(o["properties"]) {
						path := key
						if prefix != "" {
							path = prefix + "." + key
						}
						paths = append(paths, path)
						walk(s.Obj(v), path)
					}
					if i := s.Obj(o["items"]); i != nil {
						walk(i, prefix+"[]")
					}
				}
				walk(schema, "")
				var got []string
				seen := map[string]bool{}
				for _, e := range s.Catalogue.Entries {
					if seen[e.ID] {
						t.Fatal("duplicate ID")
					}
					seen[e.ID] = true
					if e.Kind == "property" {
						got = append(got, e.Path)
					} else {
						if err := s.CheckRule(e.Rule, &c); err != nil && strings.Contains(err.Error(), "unimplemented") {
							t.Fatal(err)
						}
					}
				}
				sort.Strings(paths)
				sort.Strings(got)
				if !s.Equal(paths, got) {
					t.Fatal("schema/catalogue paths differ")
				}
			case "schema/accepts-independent-valid-document":
				accepted(t, checkRule(&c, "single-json-object"))
			case "schema/rejects-unknown-property", "schema/rejects-wrong-type", "schema/rejects-missing-required-field", "schema/unsupported-declaration-cannot-hide-emitted-invalid-field":
				switch id {
				case "schema/rejects-unknown-property":
					o["unknown"] = true
				case "schema/rejects-wrong-type":
					test["duration"] = "12"
				case "schema/rejects-missing-required-field":
					delete(test, "status")
				default:
					test["extra"] = "invalid"
				}
				b, _ := json.Marshal(o)
				c.Samples[0].Bytes = b
				rejected(t, checkRule(&c, "single-json-object"))
			case "schema/requires-positive-evidence":
				rejected(t, check(&c, "results.tests[].screenshot"))
			case "schema/required-child-is-checked-when-parent-exists":
				delete(test, "status")
				rejected(t, check(&c, "results.tests[].status"))
			case "schema/capabilities-require-entry-and-reason":
				e := entry("property", "reportFormat")
				_, err := s.Disposition(e, s.Manifest{})
				rejected(t, err)
				_, err = s.Disposition(e, s.Manifest{e.ID: {Supported: true}})
				rejected(t, err)
			case "specification/accepts-independent-retry-history":
				for _, name := range []string{"history-requires-retry-count", "positive-retries-require-history", "retry-count-matches-history-length", "history-orders-contiguous-attempts", "history-excludes-final-attempt", "flaky-matches-attempt-outcomes"} {
					accepted(t, checkRule(&c, name))
				}
			case "specification/rejects-source-duration-unit-mismatch":
				test["duration"] = float64(12000)
				rejected(t, check(&c, "results.tests[].duration"))
			case "specification/missing-contextual-evidence-fails":
				for _, name := range []string{"case-identity-stable-across-runs", "coordinates-run-and-shard-identity"} {
					rejected(t, checkRule(&c, name))
				}
			case "specification/timing-recommendation-warns":
				s.Summary(o)["stop"] = float64(50)
				accepted(t, checkRule(&c, "orders-start-and-stop"))
				if len(c.Warnings) != 1 {
					t.Fatal(c.Warnings)
				}
			case "specification/total-count-recommendation-warns-but-bounds-fail":
				s.Summary(o)["tests"] = float64(2)
				accepted(t, check(&c, "results.summary.tests"))
				accepted(t, checkRule(&c, "summary-matches-final-results"))
				if len(c.Warnings) != 1 {
					t.Fatal(c.Warnings)
				}
				s.Summary(o)["tests"] = float64(-1)
				rejected(t, check(&c, "results.summary.tests"))
			case "schema/shared-snapshot-matches-declared-files":
				b, err := os.ReadFile("../shared-files.json")
				accepted(t, err)
				var manifest struct {
					Revision string
					Files    map[string]string
				}
				accepted(t, json.Unmarshal(b, &manifest))
				if manifest.Revision != "standard-checker-controls-v1" || len(manifest.Files) < 5 {
					t.Fatal("missing native snapshot provenance")
				}
				for path, want := range manifest.Files {
					b, err := os.ReadFile(filepath.Join("..", path))
					accepted(t, err)
					h := sha256.Sum256(b)
					if hex.EncodeToString(h[:]) != want {
						t.Fatal("shared file drift: " + path)
					}
				}
			default:
				if strings.HasPrefix(id, "expected-failure/") {
					defect(t, id)
					return
				}
				name := strings.TrimSuffix(strings.TrimPrefix(id, "specification/rejects-"), "-mutation")
				switch name {
				case "utf8-without-bom":
					c.Samples[0].Bytes = []byte{239, 187, 191, 123, 125}
				case "summary-matches-final-results":
					s.Summary(o)["passed"] = float64(2)
				case "history-orders-contiguous-attempts":
					history["attempt"] = float64(2)
				case "retry-count-matches-history-length":
					test["retries"] = float64(2)
				case "history-excludes-final-attempt":
					test["retryAttempts"] = append(s.Array(test["retryAttempts"]), s.Object{"attempt": float64(2), "status": "passed"})
				case "flaky-matches-attempt-outcomes":
					test["flaky"] = false
				case "attempt-identity-scoped-to-execution":
					test["attemptId"] = "first"
				case "respects-reserved-extension-ownership":
					test["extra"] = s.Object{"ctrf.not-owned": true}
				default:
					t.Fatal("unimplemented control " + id)
				}
				rejected(t, s.CheckRule(name, &c))
			}
		})
	}
}
func defect(t *testing.T, id string) {
	c := s.Context{}
	for _, name := range []string{"first", "second"} {
		c.Samples = append(c.Samples, s.Sample{Name: name, Report: s.Object{"results": s.Object{"environment": s.Object{"buildNumber": 1.5}}}})
	}
	set := func(i int, v interface{}) { s.Obj(s.Results(c.Samples[i].Report)["environment"])["buildNumber"] = v }
	name := strings.TrimPrefix(id, "expected-failure/")
	switch name {
	case "rejects-unexpected-pass":
		set(0, float64(0))
		set(1, float64(0))
	case "rejects-partial-fix":
		set(1, float64(0))
	case "rejects-different-value":
		set(0, "wrong")
	case "rejects-missing-sample":
		c.Samples = c.Samples[:1]
	case "rejects-duplicate-sample":
		c.Samples = append(c.Samples, c.Samples[0])
	case "rejects-unrecorded-broken-sample":
		sm := c.Samples[0]
		sm.Name = "third"
		c.Samples = append(c.Samples, sm)
	case "allows-optional-absence-outside-defect-samples":
		c.Samples = append(c.Samples, s.Sample{Name: "absent", Report: s.Object{"results": s.Object{}}})
	}
	e := entry("property", "results.environment.buildNumber")
	issue := &s.Issue{Key: "control-defect", Reason: "Synthetic checker control", FollowUp: "https://example.test/control", Samples: map[string][]string{"unit": {"first", "second"}}}
	checker := func(_ s.Entry, c *s.Context) error {
		if name == "rejects-infrastructure-error" {
			return fmt.Errorf("infrastructure")
		}
		for _, sm := range c.Samples {
			if name == "rejects-unrelated-error-in-later-sample" && sm.Name == "second" {
				return fmt.Errorf("different error")
			}
			v := s.Obj(s.Results(sm.Report)["environment"])["buildNumber"]
			if v != float64(0) {
				return &s.Mismatch{Code: "control mismatch", Actual: v, Expected: float64(0)}
			}
		}
		return nil
	}
	matcher := func(_ s.Entry, _ s.Sample, _ s.Issue, err error) bool {
		m, ok := err.(*s.Mismatch)
		return ok && m.Code == "control mismatch" && m.Actual == 1.5 && m.Expected == float64(0)
	}
	level := "unit"
	if name == "does-not-apply-at-unrecorded-level" {
		level = "integration"
	}
	marked, err := s.RunExpectedFailure(e, &c, level, issue, checker, matcher)
	if strings.HasPrefix(name, "accepts-") || strings.HasPrefix(name, "allows-") {
		accepted(t, err)
		if !marked {
			t.Fatal("missing expected-failure record")
		}
	} else {
		rejected(t, err)
	}
}
