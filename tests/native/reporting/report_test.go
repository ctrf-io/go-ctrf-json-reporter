package reporting

import (
	"encoding/json"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"strings"
	"testing"
)

func records(events ...s.Object) string {
	var b strings.Builder
	for _, event := range events {
		v, _ := json.Marshal(event)
		b.Write(v)
		b.WriteByte('\n')
	}
	return b.String()
}
func report(t *testing.T, input, scope string, exit int) s.Object {
	t.Helper()
	b, err := Report(strings.NewReader(input), scope, exit)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Validate(b); err != nil {
		t.Fatal(err)
	}
	o, err := s.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func runnerEvent(action, name string) s.Object {
	return s.Object{"Action": action, "Package": "independent", "Test": name}
}
func TestCompletedResultsUseLeafTests(t *testing.T) {
	input := records(runnerEvent("pass", "TestParent/child"), runnerEvent("pass", "TestParent"))
	o := report(t, input, "reporter-tests", 0)
	if len(s.Tests(o)) != 1 || s.Summary(o)["passed"] != float64(1) {
		t.Fatal(o)
	}
}
func TestRunnerFailuresBecomeVisible(t *testing.T) {
	o := report(t, "", "unit", 1)
	if s.Summary(o)["failed"] != float64(1) {
		t.Fatal(o)
	}
}
func TestMissingInventoryFails(t *testing.T) {
	o := report(t, records(runnerEvent("pass", "TestConformance/conformance/property/reportFormat")), "unit", 0)
	if s.Summary(o)["failed"] != float64(1) {
		t.Fatal(o)
	}
}
func TestExpectedFailuresUseOther(t *testing.T) {
	e := runnerEvent("output", "TestConformance/conformance/property/reportFormat")
	e["Output"] = "CTRF_EXPECTED_FAILURE {\"issue\":\"local-defect\",\"reason\":\"known\",\"followUp\":\"pending issue\"}\n"
	o := report(t, records(e, runnerEvent("pass", "TestConformance/conformance/property/reportFormat")), "unit", 0)
	test := s.Obj(s.Tests(o)[0])
	if test["status"] != "other" || test["rawStatus"] != "expected-failure" || s.Obj(s.Obj(test["extra"])["expectedFailure"])["issue"] != "local-defect" {
		t.Fatal(test)
	}
	if !strings.Contains(strings.Join(toStrings(s.Array(test["suite"])), "/"), "Expected failures") {
		t.Fatal(test)
	}
}
func toStrings(a []interface{}) []string {
	var out []string
	for _, v := range a {
		out = append(out, v.(string))
	}
	return out
}
func TestFailureWinsOverExpectedMarker(t *testing.T) {
	e := runnerEvent("output", "case")
	e["Output"] = "CTRF_EXPECTED_FAILURE {\"issue\":\"local-defect\"}\n"
	o := report(t, records(e, runnerEvent("fail", "case")), "reporter-tests", 1)
	if s.Summary(o)["failed"] != float64(1) || s.Summary(o)["other"] != float64(0) {
		t.Fatal(o)
	}
}
func TestSkipReasonIsRetained(t *testing.T) {
	e := runnerEvent("output", "case")
	e["Output"] = "unsupported optional feature\n"
	o := report(t, records(e, runnerEvent("skip", "case")), "reporter-tests", 0)
	test := s.Obj(s.Tests(o)[0])
	if test["status"] != "skipped" || !strings.Contains(test["message"].(string), "unsupported") {
		t.Fatal(test)
	}
}
func TestMalformedRunnerDataRejected(t *testing.T) {
	if _, err := Report(strings.NewReader("invalid"), "unit", 0); err == nil {
		t.Fatal("malformed stream accepted")
	}
}
func TestDuplicateRunnerIdentityRejected(t *testing.T) {
	if _, err := Report(strings.NewReader(records(runnerEvent("pass", "case"), runnerEvent("run", "case"))), "reporter-tests", 0); err == nil {
		t.Fatal("duplicate execution accepted")
	}
}
func TestPackageFailureWithoutFailingTestIsVisible(t *testing.T) {
	o := report(t, records(runnerEvent("pass", "case"), s.Object{"Action": "fail", "Package": "independent"}), "reporter-tests", 1)
	if s.Summary(o)["failed"] != float64(1) {
		t.Fatal(o)
	}
}
func TestWarningsAreRetained(t *testing.T) {
	e := runnerEvent("output", "case")
	e["Output"] = "WARNING: recommendation\n"
	o := report(t, records(e, runnerEvent("pass", "case")), "reporter-tests", 0)
	test := s.Obj(s.Tests(o)[0])
	if len(s.Array(s.Obj(test["extra"])["warnings"])) != 1 {
		t.Fatal(test)
	}
}

func TestLegacyFixtureRecordsDoNotInflateHarnessResults(t *testing.T) {
	input := records(runnerEvent("pass", "TestReal"), runnerEvent("fail", "Test_Flaky_Fail"), runnerEvent("run", "Test_Flaky_Fail"), runnerEvent("fail", "Test_Flaky_Fail"))
	inventory := Inventory{"independent": {"TestReal": true}}
	b, err := ReportWithInventory(strings.NewReader(input), "reporter-tests", 0, inventory)
	if err != nil {
		t.Fatal(err)
	}
	o, err := s.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if s.Summary(o)["passed"] != float64(1) || s.Summary(o)["failed"] != float64(0) || len(s.Tests(o)) != 1 {
		t.Fatal(o)
	}
}
func TestMissingRegisteredNativeTestFails(t *testing.T) {
	b, err := ReportWithInventory(strings.NewReader(records(runnerEvent("pass", "TestReal"))), "reporter-tests", 0, Inventory{"independent": {"TestReal": true, "TestMissing": true}})
	if err != nil {
		t.Fatal(err)
	}
	o, _ := s.Decode(b)
	if s.Summary(o)["failed"] != float64(1) {
		t.Fatal(o)
	}
}
func TestInventoryUsesIndependentRegisteredNames(t *testing.T) {
	input := s.Object{"Action": "output", "Package": "independent", "Output": "TestReal\n"}
	inventory, err := Discover(strings.NewReader(records(input)))
	if err != nil || !inventory["independent"]["TestReal"] {
		t.Fatal(inventory, err)
	}
	if _, err = Discover(strings.NewReader("")); err == nil {
		t.Fatal("empty inventory accepted")
	}
}
