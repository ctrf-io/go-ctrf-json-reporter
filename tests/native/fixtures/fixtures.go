// Package fixtures drives the real producer. Expectations come from input events
// and native go test records, never from the emitted CTRF document.
package fixtures

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/ctrf-io/go-ctrf-json-reporter/reporter"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
)

//go:embed fixture_test.go.txt
var fixtureSource []byte

type Event struct {
	Time                  string
	Action, Package, Test string
	Elapsed               float64
	Output                string
}

func encode(events []Event) []byte {
	var b bytes.Buffer
	for _, e := range events {
		if err := json.NewEncoder(&b).Encode(e); err != nil {
			panic(err)
		}
	}
	return b.Bytes()
}
func Source(events []Event) s.Object {
	out := []interface{}{}
	indices := map[string]int{}
	attempts := map[string][]interface{}{}
	for _, e := range events {
		if e.Test == "" || (e.Action != "pass" && e.Action != "fail" && e.Action != "skip") {
			continue
		}
		key := e.Package + "\x00" + e.Test
		status := map[string]string{"pass": "passed", "fail": "failed", "skip": "skipped"}[e.Action]
		duration := float64(int64(e.Elapsed * 1000))
		a := s.Object{"status": status, "duration": duration}
		attempts[key] = append(attempts[key], a)
		test := s.Object{"name": e.Test, "status": status, "duration": duration, "suite": []interface{}{e.Package}}
		if len(attempts[key]) > 1 {
			test["sourceAttempts"] = attempts[key]
			test["retries"] = float64(len(attempts[key]) - 1)
			if status == "passed" && hasFailed(attempts[key][:len(attempts[key])-1]) {
				test["flaky"] = true
			}
		}
		if i, ok := indices[key]; ok {
			out[i] = test
		} else {
			indices[key] = len(out)
			out = append(out, test)
		}
	}
	summary := s.Object{"tests": float64(len(out)), "passed": float64(0), "failed": float64(0), "skipped": float64(0), "pending": float64(0), "other": float64(0)}
	for _, v := range out {
		t := s.Obj(v)
		key := t["status"].(string)
		summary[key] = s.Num(summary[key]) + 1
	}
	return s.Object{"reportFormat": "CTRF", "specVersion": "0.2.0", "generatedBy": "go-ctrf-json-reporter", "results": s.Object{"tool": s.Object{"name": "gotest"}, "summary": summary, "tests": out}}
}
func hasFailed(a []interface{}) bool {
	for _, v := range a {
		if s.Obj(v)["status"] == "failed" {
			return true
		}
	}
	return false
}
func UnitEvents(statuses ...string) []Event {
	var events []Event
	t := time.Now().Add(-time.Second)
	for i, status := range statuses {
		name := []string{"TestPass", "TestFail", "TestSkip"}[i%3]
		if len(statuses) > 3 {
			name = "TestRetry"
		}
		events = append(events, Event{Time: t.Add(time.Duration(i*60) * time.Millisecond).Format(time.RFC3339Nano), Action: "run", Package: "example.com/fixture", Test: name})
		events = append(events, Event{Time: t.Add(time.Duration(i*60+20) * time.Millisecond).Format(time.RFC3339Nano), Action: "output", Package: "example.com/fixture", Test: name, Output: "independent failure\n"})
		events = append(events, Event{Time: t.Add(time.Duration(i*60+25) * time.Millisecond).Format(time.RFC3339Nano), Action: status, Package: "example.com/fixture", Test: name, Elapsed: 0.025})
	}
	return events
}
func retryEvents(statuses ...string) []Event {
	var out []Event
	for i, status := range statuses {
		es := UnitEvents(status)
		for j := range es {
			es[j].Test = "TestRetry"
			es[j].Elapsed = 0.011 + float64(i)*0.01
		}
		out = append(out, es...)
	}
	return out
}
func environment(build string) *ctrf.Environment {
	return &ctrf.Environment{AppName: "fixture-app", AppVersion: "1.2.3", OSPlatform: "fixture-os", OSRelease: "fixture-release", OSVersion: "fixture-version", BuildName: "fixture-build", BuildNumber: build}
}
func expectedEnvironment(o s.Object, build, shard string) {
	env := s.Object{"appName": "fixture-app", "appVersion": "1.2.3", "osPlatform": "fixture-os", "osRelease": "fixture-release", "osVersion": "fixture-version", "buildName": "fixture-build"}
	switch build {
	case "0":
		env["buildNumber"] = float64(0)
	case "42":
		env["buildNumber"] = float64(42)
	default:
		env["extra"] = s.Object{"buildNumber": build}
	}
	if shard != "" {
		env["shardId"] = shard
	}
	s.Results(o)["environment"] = env
}
func sample(name string, b []byte, events []Event, window [2]int64, build, run, shard string) (s.Sample, error) {
	o, err := s.Decode(b)
	if err != nil {
		return s.Sample{}, err
	}
	expected := Source(events)
	if build != "" {
		expectedEnvironment(expected, build, shard)
	}
	if run != "" {
		expected["runId"] = run
	}
	return s.Sample{Name: name, Bytes: b, Report: o, Expected: expected, RunWindow: window}, nil
}
func Unit() (s.Context, error) {
	var c s.Context
	old, err := os.Getwd()
	if err != nil {
		return c, err
	}
	dir, err := os.MkdirTemp("", "ctrf-unit-")
	if err != nil {
		return c, err
	}
	defer os.RemoveAll(dir)
	if err = os.Chdir(dir); err != nil {
		return c, err
	}
	defer os.Chdir(old)
	if err = os.WriteFile("fixture_test.go", fixtureSource, 0600); err != nil {
		return c, err
	}
	cases := []struct {
		name              string
		events            []Event
		build, run, shard string
	}{
		{"mixed", UnitEvents("pass", "fail", "skip"), "42", "", ""}, {"repeat-a", UnitEvents("pass"), "0", "", ""}, {"repeat-b", UnitEvents("pass"), "0", "", ""},
		{"recovery", retryEvents("fail", "pass"), "42", "", ""}, {"exhaustion", retryEvents("fail", "fail"), "42", "", ""}, {"regression", retryEvents("fail", "pass", "fail"), "42", "", ""},
		{"shard-a", UnitEvents("pass"), "42", "coordinated", "a"}, {"shard-b", UnitEvents("pass"), "42", "coordinated", "b"}, {"legacy-build", UnitEvents("pass"), "release-42", "", ""}, {"empty", nil, "", "", ""},
	}
	for _, cs := range cases {
		start := time.Now().Add(-2 * time.Second).UnixMilli()
		var env *ctrf.Environment
		if cs.build != "" {
			env = environment(cs.build)
		}
		r, err := reporter.ParseTestResultsWithOptions(bytes.NewReader(encode(cs.events)), false, env, reporter.IdentityOptions{RunID: cs.run, ShardID: cs.shard})
		if err != nil {
			return c, err
		}
		var b bytes.Buffer
		if err = r.Write(&b, false); err != nil {
			return c, fmt.Errorf("%s: %w", cs.name, err)
		}
		sm, err := sample(cs.name, b.Bytes(), cs.events, [2]int64{start, time.Now().UnixMilli() + 1}, cs.build, cs.run, cs.shard)
		if err != nil {
			return c, err
		}
		for _, test := range s.Tests(sm.Expected) {
			s.Obj(test)["filePath"] = "fixture_test.go"
		}
		if err := retain("unit", sm, encode(cs.events), nil); err != nil {
			return c, err
		}
		c.Samples = append(c.Samples, sm)
		if strings.HasPrefix(cs.name, "repeat-") {
			c.Repeated = append(c.Repeated, sm)
		}
		if strings.HasPrefix(cs.name, "shard-") {
			c.Shards = append(c.Shards, sm)
		}
	}
	return c, nil
}
func command(dir string, env []string, name string, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return b, stderr.Bytes(), e.ExitCode(), nil
		}
		return b, stderr.Bytes(), -1, err
	}
	return b, stderr.Bytes(), 0, nil
}
func rootDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "reporter/reporter.go")); err == nil {
			return wd, nil
		}
		p := filepath.Dir(wd)
		if p == wd {
			return "", fmt.Errorf("repository root not found")
		}
		wd = p
	}
}
func Integration() (s.Context, error) {
	var c s.Context
	root, err := rootDir()
	if err != nil {
		return c, err
	}
	dir, err := os.MkdirTemp("", "ctrf-installed-")
	if err != nil {
		return c, err
	}
	defer os.RemoveAll(dir)
	bin := filepath.Join(dir, "bin")
	if err = os.MkdirAll(bin, 0700); err != nil {
		return c, err
	}
	_, stderr, code, err := command(root, []string{"GOBIN=" + bin}, "go", "install", "./cmd/go-ctrf-json-reporter")
	if err != nil || code != 0 {
		return c, fmt.Errorf("install: %v exit=%d %s", err, code, stderr)
	}
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fixture\n\ngo 1.19\n"), 0600); err != nil {
		return c, err
	}
	if err = os.WriteFile(filepath.Join(dir, "fixture_test.go"), fixtureSource, 0600); err != nil {
		return c, err
	}
	cases := []struct {
		name, pattern, build, run, shard string
		outcomes                         []string
	}{
		{"mixed", "^Test(Pass|Fail|Skip|Nested|ParallelA|ParallelB)$", "42", "", "", []string{""}},
		{"repeat-a", "^TestPass$", "0", "", "", []string{""}}, {"repeat-b", "^TestPass$", "0", "", "", []string{""}},
		{"recovery", "^TestRetry$", "42", "", "", []string{"failed", "passed"}}, {"exhaustion", "^TestRetry$", "42", "", "", []string{"failed", "failed"}}, {"regression", "^TestRetry$", "42", "", "", []string{"failed", "passed", "failed"}},
		{"shard-a", "^TestPass$", "42", "coordinated", "a", []string{""}}, {"shard-b", "^TestSkip$", "42", "coordinated", "b", []string{""}},
		{"legacy-build", "^TestPass$", "release-42", "", "", []string{""}}, {"empty", "^NoTests$", "", "", "", []string{""}}, {"panic", "^TestPanic$", "42", "", "", []string{""}},
	}
	for _, cs := range cases {
		start := time.Now().UnixMilli() - 1
		var stream bytes.Buffer
		var events []Event
		for _, outcome := range cs.outcomes {
			b, se, exit, err := command(dir, []string{"CTRF_FIXTURE_OUTCOME=" + outcome}, "go", "test", "-json", "-count=1", "-parallel=2", "-run", cs.pattern, ".")
			if err != nil {
				return c, err
			}
			wantFail := cs.name == "mixed" || cs.name == "panic" || outcome == "failed"
			want := 0
			if wantFail {
				want = 1
			}
			if exit != want {
				return c, fmt.Errorf("native %s exit %d want %d: %s", cs.name, exit, want, se)
			}
			stream.Write(b)
			decoder := json.NewDecoder(bytes.NewReader(b))
			var observed []Event
			for {
				var e Event
				err := decoder.Decode(&e)
				if err == io.EOF {
					break
				}
				if err != nil {
					return c, err
				}
				events = append(events, e)
				observed = append(observed, e)
			}
			// Independently fixed native fixture outcomes; a dropped test may not shrink the oracle.
			if err := checkNative(cs.name, observed, outcome); err != nil {
				return c, err
			}
		}
		output := filepath.Join(dir, cs.name+".json")
		args := []string{"-output", output, "-quiet"}
		if cs.build != "" {
			args = append(args, "-appName", "fixture-app", "-appVersion", "1.2.3", "-osPlatform", "fixture-os", "-osRelease", "fixture-release", "-osVersion", "fixture-version", "-buildName", "fixture-build", "-buildNumber", cs.build)
		}
		if cs.run != "" {
			args = append(args, "-runId", cs.run, "-shardId", cs.shard)
		}
		cmd := exec.Command(filepath.Join(bin, "go-ctrf-json-reporter"), args...)
		cmd.Dir = dir
		nativeStream := append([]byte(nil), stream.Bytes()...)
		cmd.Stdin = bytes.NewReader(nativeStream)
		combined, err := cmd.CombinedOutput()
		exit := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				exit = e.ExitCode()
			} else {
				return c, err
			}
		}
		// Existing converter exits nonzero for any failed source attempt, even recovered ones.
		want := 0
		for _, e := range events {
			if e.Action == "fail" {
				want = 1
			}
		}
		if exit != want {
			return c, fmt.Errorf("converter %s exit %d want %d: %s", cs.name, exit, want, combined)
		}
		b, err := os.ReadFile(output)
		if err != nil {
			return c, err
		}
		sm, err := sample(cs.name, b, events, [2]int64{start, time.Now().UnixMilli() + 1}, cs.build, cs.run, cs.shard)
		if err != nil {
			return c, err
		}
		for _, test := range s.Tests(sm.Expected) {
			s.Obj(test)["filePath"] = "fixture_test.go"
		}
		if err := retain("integration", sm, nativeStream, combined); err != nil {
			return c, err
		}
		c.Samples = append(c.Samples, sm)
		if strings.HasPrefix(cs.name, "repeat-") {
			c.Repeated = append(c.Repeated, sm)
		}
		if strings.HasPrefix(cs.name, "shard-") {
			c.Shards = append(c.Shards, sm)
		}
	}
	return c, nil
}
func checkNative(name string, events []Event, outcome string) error {
	got := map[string]string{}
	parallel := map[string]bool{}
	continued := map[string]time.Time{}
	finished := map[string]time.Time{}
	for _, e := range events {
		if e.Test != "" && (e.Action == "pass" || e.Action == "fail" || e.Action == "skip") {
			got[e.Test] = e.Action
		}
		if e.Action == "pause" {
			parallel[e.Test] = true
		}
		if e.Action == "cont" {
			continued[e.Test], _ = time.Parse(time.RFC3339Nano, e.Time)
		}
		if e.Action == "pass" {
			finished[e.Test], _ = time.Parse(time.RFC3339Nano, e.Time)
		}
	}
	want := map[string]string{}
	switch name {
	case "mixed":
		want = map[string]string{"TestPass": "pass", "TestFail": "fail", "TestSkip": "skip", "TestNested": "pass", "TestNested/same": "pass", "TestParallelA": "pass", "TestParallelB": "pass"}
		if !parallel["TestParallelA"] || !parallel["TestParallelB"] || continued["TestParallelA"].IsZero() || continued["TestParallelB"].IsZero() || !continued["TestParallelA"].Before(finished["TestParallelB"]) || !continued["TestParallelB"].Before(finished["TestParallelA"]) {
			return fmt.Errorf("missing overlapping native parallel execution evidence")
		}
	case "recovery", "exhaustion", "regression":
		v := "pass"
		if outcome == "failed" {
			v = "fail"
		}
		want["TestRetry"] = v
	case "panic":
		want["TestPanic"] = "fail"
	case "shard-b":
		want["TestSkip"] = "skip"
	case "empty":
	default:
		want["TestPass"] = "pass"
	}
	// Some Go versions terminate a panic before emitting the per-test fail record.
	if name == "panic" {
		found := false
		for _, e := range events {
			found = found || strings.Contains(e.Output, "independent panic")
		}
		if !found {
			return fmt.Errorf("missing native panic output")
		}
		return nil
	}
	if !s.Equal(got, want) {
		return fmt.Errorf("native %s inventory: got %v want %v", name, got, want)
	}
	return nil
}
func SortedNames(c s.Context) []string {
	var names []string
	for _, sm := range c.Samples {
		names = append(names, sm.Name)
	}
	sort.Strings(names)
	return names
}

// Fixture evidence is retained separately from assessment results. Publication
// globs include reports/*.json, never these subject reports or native streams.
func retain(level string, sm s.Sample, native, logs []byte) error {
	artifacts := os.Getenv("CTRF_STANDARD_ARTIFACTS")
	if artifacts == "" {
		return nil
	}
	if !filepath.IsAbs(artifacts) {
		return fmt.Errorf("CTRF_STANDARD_ARTIFACTS must be absolute")
	}
	dir := filepath.Join(artifacts, "fixtures", level, sm.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	source, err := json.MarshalIndent(s.Object{"expected": sm.Expected, "runWindow": sm.RunWindow}, "", "  ")
	if err != nil {
		return err
	}
	for name, b := range map[string][]byte{"report.json": sm.Bytes, "source.json": source, "native.jsonl": native, "converter.log": logs} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			return err
		}
	}
	return nil
}
