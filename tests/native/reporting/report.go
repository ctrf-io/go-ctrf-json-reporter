// Package reporting converts completed go test records into assessment CTRF.
// It deliberately does not use the subject converter to report its own defects.
package reporting

import (
	"bufio"
	"encoding/json"
	"fmt"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"io"
	"math"
	"sort"
	"strings"
)

type event struct {
	Action, Package, Test, Output string
	Elapsed                       float64
}
type record struct {
	Package, Name, Action string
	Elapsed               float64
	Output                strings.Builder
	Issue                 s.Object
	Finished              bool
}

type Inventory map[string]map[string]bool

// Discover reads an independent go test -list . -json run; legacy test2json can
// misinterpret a converter fixture's raw === RUN output as real harness tests.
func Discover(reader io.Reader) (Inventory, error) {
	inventory := Inventory{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var e event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, err
		}
		if e.Action != "output" || e.Test != "" {
			continue
		}
		name := strings.TrimSpace(e.Output)
		if strings.ContainsAny(name, " \t\n") || !(strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Example") || strings.HasPrefix(name, "Fuzz")) {
			continue
		}
		if inventory[e.Package] == nil {
			inventory[e.Package] = map[string]bool{}
		}
		inventory[e.Package][name] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(inventory) == 0 {
		return nil, fmt.Errorf("no registered native tests discovered")
	}
	return inventory, nil
}
func Report(reader io.Reader, scope string, exit int) ([]byte, error) {
	return ReportWithInventory(reader, scope, exit, nil)
}
func ReportWithInventory(reader io.Reader, scope string, exit int, inventory Inventory) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 8*1024*1024)
	records := map[string]*record{}
	packageFailures := map[string]bool{}
	var diagnostics strings.Builder
	for scanner.Scan() {
		var e event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("invalid go test record: %w", err)
		}
		if e.Test == "" {
			if e.Action == "fail" {
				packageFailures[e.Package] = true
			}
			diagnostics.WriteString(e.Output)
			continue
		}
		if inventory != nil && !inventory[e.Package][strings.Split(e.Test, "/")[0]] {
			diagnostics.WriteString(e.Output)
			continue
		}
		key := e.Package + "\x00" + e.Test
		r := records[key]
		if r == nil {
			r = &record{Package: e.Package, Name: e.Test}
			records[key] = r
		}
		if r.Finished && e.Action == "run" {
			return nil, fmt.Errorf("duplicate completed test identity: %s", key)
		}
		r.Output.WriteString(e.Output)
		if i := strings.Index(e.Output, "CTRF_EXPECTED_FAILURE "); i >= 0 {
			var issue s.Object
			if err := json.Unmarshal([]byte(strings.TrimSpace(e.Output[i+len("CTRF_EXPECTED_FAILURE "):])), &issue); err != nil {
				return nil, err
			}
			r.Issue = issue
		}
		if e.Action == "pass" || e.Action == "fail" || e.Action == "skip" {
			r.Action = e.Action
			r.Elapsed = e.Elapsed
			r.Finished = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	var keys []string
	for k := range records {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tests := []interface{}{}
	seenIDs := map[string]bool{}
	hasFailure := false
	packageTestFailed := map[string]bool{}
	for _, key := range keys {
		r := records[key]
		parent := false
		descendantFailed := false
		for _, other := range records {
			if other.Package == r.Package && strings.HasPrefix(other.Name, r.Name+"/") {
				parent = true
				descendantFailed = descendantFailed || other.Action == "fail"
			}
		}
		if parent && (r.Action != "fail" || descendantFailed) {
			continue
		}
		name := r.Name
		group := []interface{}{"Reporter tests", r.Package}
		id := r.Package + "/" + name
		if scope == "unit" || scope == "integration" {
			name = strings.TrimPrefix(name, "TestConformance/")
			id = scope + "/" + name
			kind := "Schema properties"
			if strings.HasPrefix(name, "conformance/report/") {
				kind = "Specification relationships"
			}
			group = []interface{}{strings.ToUpper(scope[:1]) + scope[1:] + " conformance", kind}
			if !strings.HasPrefix(name, "conformance/") {
				name = "SETUP: " + name
				group = []interface{}{strings.ToUpper(scope[:1]) + scope[1:] + " conformance", "Setup failures"}
			}
		} else if scope == "checker" {
			name = strings.TrimPrefix(name, "TestChecker/")
			id = name
			parts := strings.Split(name, "/")
			group = []interface{}{"Shared checker controls"}
			if len(parts) > 1 {
				group = append(group, parts[1])
			}
		}
		if seenIDs[id] {
			return nil, fmt.Errorf("duplicate published identity %s", id)
		}
		seenIDs[id] = true
		status := map[string]string{"pass": "passed", "fail": "failed", "skip": "skipped"}[r.Action]
		if !r.Finished {
			status = "failed"
		}
		o := s.Object{"name": name, "testId": id, "status": status, "duration": math.Round(r.Elapsed * 1000), "suite": group}
		output := r.Output.String()
		if status == "failed" || status == "skipped" {
			o["message"] = output
		}
		if r.Issue != nil && status == "passed" {
			o["status"] = "other"
			o["rawStatus"] = "expected-failure"
			o["name"] = "EXPECTED FAILURE: " + name
			o["extra"] = s.Object{"expectedFailure": r.Issue}
			o["suite"] = []interface{}{strings.ToUpper(scope[:1]) + scope[1:] + " conformance", "Expected failures (tracked defects)"}
		}
		if strings.Contains(output, "WARNING:") {
			extra := s.Obj(o["extra"])
			if extra == nil {
				extra = s.Object{}
				o["extra"] = extra
			}
			var warnings []interface{}
			for _, line := range strings.Split(output, "\n") {
				if i := strings.Index(line, "WARNING:"); i >= 0 {
					warnings = append(warnings, strings.TrimSpace(line[i+len("WARNING:"):]))
				}
			}
			extra["warnings"] = warnings
		}
		if o["status"] == "failed" {
			hasFailure = true
			packageTestFailed[r.Package] = true
		}
		tests = append(tests, o)
	}
	for pkg := range packageFailures {
		if !packageTestFailed[pkg] {
			tests = append(tests, s.Object{"name": "SETUP: " + pkg, "status": "failed", "duration": float64(0), "suite": []interface{}{scope, "Setup failures"}, "message": diagnostics.String()})
			hasFailure = true
		}
	}
	for pkg, names := range inventory {
		for name := range names {
			r := records[pkg+"\x00"+name]
			if r == nil || !r.Finished {
				tests = append(tests, s.Object{"name": "SETUP: missing native test " + pkg + "/" + name, "status": "failed", "duration": float64(0), "suite": []interface{}{scope, "Setup failures"}})
				hasFailure = true
			}
		}
	}
	wanted := map[string]bool{}
	if scope == "unit" || scope == "integration" {
		for _, e := range s.Catalogue.Entries {
			wanted[scope+"/"+e.ID] = true
		}
	} else if scope == "checker" {
		for _, id := range s.CheckerIDs {
			wanted[id] = true
		}
	}
	inventoryMismatch := false
	if len(wanted) > 0 {
		if len(seenIDs) != len(wanted) {
			inventoryMismatch = true
		}
		for id := range wanted {
			if !seenIDs[id] {
				inventoryMismatch = true
			}
		}
	}
	if exit != 0 && !hasFailure || len(tests) == 0 || inventoryMismatch && !hasFailure {
		tests = append(tests, s.Object{"name": "SETUP: runner did not complete the requested inventory", "status": "failed", "duration": float64(0), "suite": []interface{}{scope, "Setup failures"}, "message": fmt.Sprintf("exit=%d completed=%d expected=%d\n%s", exit, len(tests), len(wanted), diagnostics.String())})
	}
	summary := s.Object{"tests": float64(len(tests)), "passed": float64(0), "failed": float64(0), "skipped": float64(0), "pending": float64(0), "other": float64(0), "start": float64(0), "stop": float64(0)}
	for _, v := range tests {
		test := s.Obj(v)
		status := test["status"].(string)
		summary[status] = s.Num(summary[status]) + 1
	}
	report := s.Object{"reportFormat": "CTRF", "specVersion": "0.2.0", "results": s.Object{"tool": s.Object{"name": "Go test assessment"}, "summary": summary, "tests": tests}}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	if err = s.Validate(b); err != nil {
		return nil, err
	}
	return b, nil
}
