// Package shared implements the standard-checker-controls-v1 assertions in Go.
// Catalogue and control IDs follow the reviewed JavaScript snapshot. The Core
// schema/specification, rather than producer output, define the oracle.
package shared

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed catalogue.json
var CatalogueBytes []byte

//go:embed checker-ids.json
var checkerIDsBytes []byte
var CheckerIDs []string

//go:embed ctrf.schema.json
var SchemaBytes []byte

type Object = map[string]interface{}
type Entry struct {
	ID, Kind, Path, Rule string
	Required             bool
}
type CatalogueType struct {
	Version     int
	SpecVersion string
	Entries     []Entry
}

var Catalogue CatalogueType
var documentSchema *jsonschema.Schema
var schemaObject Object

func init() {
	must(json.Unmarshal(CatalogueBytes, &Catalogue))
	must(json.Unmarshal(checkerIDsBytes, &CheckerIDs))
	must(json.Unmarshal(SchemaBytes, &schemaObject))
	documentSchema = compile(SchemaBytes)
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func compile(b []byte) *jsonschema.Schema {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft7
	c.AssertFormat = true
	must(c.AddResource("schema.json", bytes.NewReader(b)))
	s, err := c.Compile("schema.json")
	must(err)
	return s
}
func Decode(b []byte) (Object, error) {
	var o Object
	err := json.Unmarshal(b, &o)
	if err == nil && o == nil {
		err = fmt.Errorf("expected a JSON object")
	}
	return o, err
}
func Clone(o Object) Object {
	b, err := json.Marshal(o)
	must(err)
	v, err := Decode(b)
	must(err)
	return v
}
func Obj(v interface{}) Object          { o, _ := v.(map[string]interface{}); return o }
func Array(v interface{}) []interface{} { a, _ := v.([]interface{}); return a }
func Num(v interface{}) float64         { n, _ := v.(float64); return n }
func Equal(a, b interface{}) bool       { return reflect.DeepEqual(a, b) }
func Values(o interface{}, path string) []interface{} {
	values := []interface{}{o}
	for _, part := range strings.Split(path, ".") {
		key := strings.TrimSuffix(part, "[]")
		var next []interface{}
		for _, parent := range values {
			if v, ok := Obj(parent)[key]; ok {
				if key != part {
					next = append(next, Array(v)...)
				} else {
					next = append(next, v)
				}
			}
		}
		values = next
	}
	return values
}
func Results(o Object) Object      { return Obj(o["results"]) }
func Tests(o Object) []interface{} { return Array(Results(o)["tests"]) }
func Summary(o Object) Object      { return Obj(Results(o)["summary"]) }

type Sample struct {
	Name             string
	Bytes            []byte
	Report, Expected Object
	RunWindow        [2]int64
}
type Context struct {
	Samples, Repeated, Shards []Sample
	Warnings                  []string
	CTRFDefinedExtensions     []string
}

func (c Context) With(sample Sample) Context { c.Samples = []Sample{sample}; return c }
func Validate(b []byte) error {
	o, e := Decode(b)
	if e != nil {
		return e
	}
	return documentSchema.Validate(o)
}
func mismatch(code string, actual, expected interface{}) error {
	return &Mismatch{Code: code, Actual: actual, Expected: expected}
}

type Mismatch struct {
	Code             string
	Actual, Expected interface{}
}

func (e *Mismatch) Error() string {
	return fmt.Sprintf("%s: got %#v, want %#v", e.Code, e.Actual, e.Expected)
}
func expect(code string, a, b interface{}) error {
	if !Equal(a, b) {
		return mismatch(code, a, b)
	}
	return nil
}
func failure(s string) error { return fmt.Errorf("%s", s) }
func resolve(o Object) Object {
	if ref, ok := o["$ref"].(string); ok {
		var v interface{} = schemaObject
		for _, key := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
			v = Obj(v)[key]
		}
		return Obj(v)
	}
	return o
}
func localSchema(path string) *jsonschema.Schema {
	o := schemaObject
	for _, part := range strings.Split(path, ".") {
		o = Obj(Obj(resolve(o)["properties"])[strings.TrimSuffix(part, "[]")])
		if strings.HasSuffix(part, "[]") {
			o = Obj(resolve(o)["items"])
		}
	}
	o = Clone(resolve(o))
	if props := Obj(o["properties"]); props != nil {
		for k := range props {
			props[k] = Object{}
		}
	}
	if Obj(Obj(o["items"])["properties"]) != nil {
		o["items"] = Object{"type": "object"}
	}
	b, err := json.Marshal(o)
	must(err)
	return compile(b)
}
func primitive(v interface{}) bool {
	switch a := v.(type) {
	case map[string]interface{}:
		return false
	case []interface{}:
		for _, i := range a {
			switch i.(type) {
			case map[string]interface{}, []interface{}:
				return false
			}
		}
	}
	return true
}
func CheckProperty(e Entry, c *Context) error {
	validator := localSchema(e.Path)
	witness := 0
	for _, s := range c.Samples {
		actual := Values(s.Report, e.Path)
		witness += len(actual)
		parts := strings.Split(e.Path, ".")
		key := parts[len(parts)-1]
		parents := []interface{}{s.Report}
		if len(parts) > 1 {
			parents = Values(s.Report, strings.Join(parts[:len(parts)-1], "."))
		}
		if e.Required {
			for _, p := range parents {
				if _, ok := Obj(p)[key]; !ok {
					return failure("missing required " + e.Path)
				}
			}
		}
		for _, v := range actual {
			if err := validator.Validate(v); err != nil {
				return err
			}
			if strings.HasSuffix(e.Path, ".duration") || strings.HasSuffix(e.Path, ".retries") || strings.HasPrefix(e.Path, "results.summary.") && e.Path != "results.summary.extra" {
				if n, ok := v.(float64); ok && n < 0 {
					return failure("negative " + e.Path)
				}
			}
			if strings.HasSuffix(e.Path, ".suite") && len(Array(v)) == 0 {
				return failure("empty suite")
			}
			if strings.HasSuffix(e.Path, ".screenshot") {
				b, err := base64.StdEncoding.DecodeString(v.(string))
				if err != nil || !(bytes.HasPrefix(b, []byte{137, 80, 78, 71, 13, 10, 26, 10}) || bytes.HasPrefix(b, []byte{255, 216, 255})) {
					return failure("screenshot must contain PNG or JPEG bytes")
				}
			}
			if s.RunWindow[0] != 0 && (e.Path == "timestamp" || e.Path == "results.summary.start" || e.Path == "results.summary.stop") {
				n := Num(v)
				if e.Path == "timestamp" {
					t, err := time.Parse(time.RFC3339Nano, v.(string))
					if err != nil {
						return err
					}
					n = float64(t.UnixMilli())
				}
				if !(n == 0 && len(Tests(s.Report)) == 0) && (n < float64(s.RunWindow[0]) || n > float64(s.RunWindow[1])) {
					return failure(e.Path + " outside independent run window")
				}
			}
		}
		wanted := Values(s.Expected, e.Path)
		all := len(wanted) > 0
		for _, v := range wanted {
			all = all && primitive(v)
		}
		if e.Path != "results.summary.tests" && all {
			if err := expect("mapping:"+e.Path, actual, wanted); err != nil {
				return err
			}
		}
	}
	if witness == 0 {
		return failure("no positive evidence for " + e.Path)
	}
	return nil
}

var recommended = map[string]bool{"orders-start-and-stop": true, "case-identity-stable-across-runs": true, "execution-identity-distinguishes-executions": true, "attempt-identity-scoped-to-execution": true, "attachment-identity-scoped-to-parent": true, "document-identity-follows-emission": true, "coordinates-run-and-shard-identity": true}

func Check(e Entry, c *Context) error {
	if len(c.Samples) == 0 {
		return failure("missing samples")
	}
	if e.Kind == "property" {
		return CheckProperty(e, c)
	}
	if e.Rule == "case-identity-stable-across-runs" && len(c.Repeated) < 2 {
		return failure("missing repeated-run evidence")
	}
	if e.Rule == "coordinates-run-and-shard-identity" && len(c.Shards) < 2 {
		return failure("missing shard evidence")
	}
	err := CheckRule(e.Rule, c)
	if err != nil && recommended[e.Rule] {
		c.Warnings = append(c.Warnings, e.ID+" (SHOULD): "+err.Error())
		return nil
	}
	return err
}
func unique(v []interface{}) error {
	seen := map[string]bool{}
	for _, i := range v {
		if i == nil {
			continue
		}
		s := fmt.Sprint(i)
		if seen[s] {
			return mismatch("duplicate-identity", s, "unique")
		}
		seen[s] = true
	}
	return nil
}
func children(t Object) []interface{} { return Array(t["retryAttempts"]) }
func statuses(a []interface{}, status string) float64 {
	n := 0.
	for _, v := range a {
		if Obj(v)["status"] == status {
			n++
		}
	}
	return n
}
func CheckRule(rule string, c *Context) error {
	if rule == "case-identity-stable-across-runs" {
		if len(c.Repeated) < 2 {
			return failure("missing repeated runs")
		}
		var first []interface{}
		for i, s := range c.Repeated {
			var keys []interface{}
			for _, v := range Tests(s.Report) {
				t := Obj(v)
				keys = append(keys, []interface{}{t["name"], t["testId"]})
			}
			if i == 0 {
				first = keys
			} else if e := expect(rule, keys, first); e != nil {
				return e
			}
		}
		return nil
	}
	if rule == "coordinates-run-and-shard-identity" {
		if len(c.Shards) < 2 {
			return failure("missing shards")
		}
		var ids []interface{}
		for _, s := range c.Shards {
			if e := expect(rule, s.Report["runId"], c.Shards[0].Report["runId"]); e != nil {
				return e
			}
			ids = append(ids, Obj(Results(s.Report)["environment"])["shardId"])
		}
		return unique(ids)
	}
	var allExecution, allDocument []interface{}
	witness := false
	for _, s := range c.Samples {
		sum := Summary(s.Report)
		tests := Tests(s.Report)
		switch rule {
		case "utf8-without-bom":
			if !utf8.Valid(s.Bytes) || bytes.HasPrefix(s.Bytes, []byte{239, 187, 191}) {
				return failure("invalid UTF-8 or BOM")
			}
		case "single-json-object":
			if err := Validate(s.Bytes); err != nil {
				return err
			}
		case "declares-supported-contract":
			if s.Report["reportFormat"] != "CTRF" || s.Report["specVersion"] != Catalogue.SpecVersion {
				return failure("unsupported contract")
			}
		case "summary-matches-final-results":
			if Num(sum["tests"]) != float64(len(tests)) {
				c.Warnings = append(c.Warnings, "summary.tests differs from test array length (SHOULD, Core §8.1)")
			}
			for _, status := range []string{"passed", "failed", "skipped", "pending", "other"} {
				if err := expect("summary:"+status, sum[status], statuses(tests, status)); err != nil {
					return err
				}
			}
		case "summary-matches-flaky-results":
			if v, ok := sum["flaky"]; ok {
				witness = true
				n := 0.
				for _, t := range tests {
					if Obj(t)["flaky"] == true {
						n++
					}
				}
				if err := expect(rule, v, n); err != nil {
					return err
				}
			}
		case "orders-start-and-stop":
			var timed []interface{}
			timed = append(timed, sum)
			for _, test := range tests {
				timed = append(timed, test)
				timed = append(timed, children(Obj(test))...)
			}
			for _, v := range timed {
				o := Obj(v)
				if o["start"] != nil && o["stop"] != nil && Num(o["stop"]) < Num(o["start"]) {
					c.Warnings = append(c.Warnings, "stop precedes start (SHOULD)")
				}
			}
		case "document-identity-follows-emission":
			allDocument = append(allDocument, s.Report["reportId"])
		}
		for idx, v := range tests {
			t := Obj(v)
			h := children(t)
			source := Object{}
			if exp := Tests(s.Expected); idx < len(exp) {
				source = Obj(exp[idx])
			}
			switch rule {
			case "history-requires-retry-count":
				if t["retryAttempts"] != nil {
					witness = true
					if n, ok := t["retries"].(float64); !ok || n != float64(int(n)) {
						return failure("history requires integer retries")
					}
				}
			case "positive-retries-require-history":
				if Num(t["retries"]) > 0 && len(h) == 0 {
					return failure("positive retries require history")
				}
			case "zero-retries-omit-history":
				if t["retries"] == float64(0) && t["retryAttempts"] != nil {
					return failure("zero retries must omit history")
				}
			case "retry-count-matches-history-length":
				if t["retryAttempts"] != nil {
					if err := expect(rule, t["retries"], float64(len(h))); err != nil {
						return err
					}
				}
			case "history-orders-contiguous-attempts":
				for i, a := range h {
					if err := expect(rule, Obj(a)["attempt"], float64(i+1)); err != nil {
						return err
					}
				}
				if t["retryAttempts"] != nil && Num(t["retries"]) != float64(len(h)) {
					return mismatch(rule, Num(t["retries"]), float64(len(h)))
				}
			case "history-excludes-final-attempt":
				if h != nil && source["sourceAttempts"] != nil {
					witness = true
					a := Array(source["sourceAttempts"])
					if err := expect("history-length", float64(len(h)), float64(len(a)-1)); err != nil {
						return err
					}
					for i, past := range h {
						for _, key := range []string{"status", "duration"} {
							if err := expect("history:"+key, Obj(past)[key], Obj(a[i])[key]); err != nil {
								return err
							}
						}
					}
					if err := expect("final-duration", t["duration"], Obj(a[len(a)-1])["duration"]); err != nil {
						return err
					}
				}
			case "flaky-matches-attempt-outcomes":
				if f, ok := t["flaky"]; ok {
					earlier := h
					if a := Array(source["sourceAttempts"]); len(a) > 0 {
						earlier = a[:len(a)-1]
					}
					want := t["status"] == "passed" && statuses(earlier, "failed") > 0
					if err := expect(rule, f, want); err != nil {
						return err
					}
				}
			case "execution-identity-distinguishes-executions":
				if id := t["executionId"]; id != nil {
					allExecution = append(allExecution, id)
				}
			case "attempt-identity-scoped-to-execution":
				ids := []interface{}{t["attemptId"]}
				for _, a := range h {
					ids = append(ids, Obj(a)["attemptId"])
				}
				if err := unique(ids); err != nil {
					return err
				}
			case "attachment-identity-scoped-to-parent":
				for _, p := range append([]interface{}{t}, h...) {
					var ids []interface{}
					for _, a := range Array(Obj(p)["attachments"]) {
						ids = append(ids, Obj(a)["attachmentId"])
					}
					if err := unique(ids); err != nil {
						return err
					}
				}
			}
		}
		if rule == "respects-reserved-extension-ownership" {
			for _, e := range Catalogue.Entries {
				if e.Kind == "property" && (e.Path == "extra" || strings.HasSuffix(e.Path, ".extra")) {
					for _, v := range Values(s.Report, e.Path) {
						for key := range Obj(v) {
							if strings.HasPrefix(key, "ctrf.") || strings.HasPrefix(key, "ctrf/") {
								allowed := false
								for _, known := range c.CTRFDefinedExtensions {
									allowed = allowed || known == key
								}
								if !allowed {
									return failure("reserved extension: " + key)
								}
							}
						}
					}
				}
			}
		}
	}
	switch rule {
	case "summary-matches-flaky-results", "history-requires-retry-count", "history-excludes-final-attempt":
		if !witness {
			return failure("missing positive relationship evidence")
		}
	case "execution-identity-distinguishes-executions":
		if len(allExecution) == 0 {
			return failure("missing execution identities")
		}
		return unique(allExecution)
	case "document-identity-follows-emission":
		return unique(allDocument)
	}
	switch rule {
	case "utf8-without-bom",
		"single-json-object",
		"declares-supported-contract",
		"summary-matches-final-results",
		"summary-matches-flaky-results",
		"orders-start-and-stop",
		"history-requires-retry-count",
		"positive-retries-require-history",
		"zero-retries-omit-history",
		"retry-count-matches-history-length",
		"history-orders-contiguous-attempts",
		"history-excludes-final-attempt",
		"flaky-matches-attempt-outcomes",
		"case-identity-stable-across-runs",
		"execution-identity-distinguishes-executions",
		"attempt-identity-scoped-to-execution",
		"attachment-identity-scoped-to-parent",
		"document-identity-follows-emission",
		"coordinates-run-and-shard-identity",
		"respects-reserved-extension-ownership":
		return nil
	default:
		return failure("unimplemented rule " + rule)
	}
}

type Capability struct {
	Supported bool   `json:"supported"`
	Reason    string `json:"reason"`
}
type Manifest map[string]Capability

func Disposition(e Entry, m Manifest) (Capability, error) {
	c, ok := m[e.ID]
	if !ok || c.Reason == "" {
		return c, failure("capability requires entry and reason: " + e.ID)
	}
	return c, nil
}
