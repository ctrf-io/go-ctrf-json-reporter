package fixtures

import (
	"encoding/json"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"os"
	"path/filepath"
	"testing"
)

func Manifest() (s.Manifest, error) {
	root, err := rootDir()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(root, "tests/standard/conformance/go.json"))
	if err != nil {
		return nil, err
	}
	var m s.Manifest
	err = json.Unmarshal(b, &m)
	return m, err
}
func Run(t *testing.T, level string) {
	m, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	var c s.Context
	if level == "unit" {
		c, err = Unit()
	} else {
		c, err = Integration()
	}
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	for _, entry := range s.Catalogue.Entries {
		entry := entry
		t.Run(entry.ID, func(t *testing.T) {
			cap, err := s.Disposition(entry, m)
			if err != nil {
				t.Fatal(err)
			}
			if !cap.Supported {
				t.Skip(cap.Reason)
			}
			context := c
			context.Warnings = nil
			// This rule requires independent retry evidence. Give its strict
			// per-sample engine the complete set of retry scenarios, rather than
			// unrelated empty/non-retried scenarios with no applicable history.
			if entry.Rule == "history-excludes-final-attempt" {
				context.Samples = nil
				for _, sample := range c.Samples {
					for _, test := range s.Tests(sample.Expected) {
						if s.Obj(test)["sourceAttempts"] != nil {
							context.Samples = append(context.Samples, sample)
							break
						}
					}
				}
			}
			issue := KnownIssue(entry)
			expected, err := s.RunExpectedFailure(entry, &context, level, issue, s.Check, MatchKnownFailure)
			if err != nil {
				t.Fatal(err)
			}
			for _, warning := range context.Warnings {
				t.Log("WARNING: " + warning)
			}
			if expected {
				b, err := json.Marshal(issue)
				if err != nil {
					t.Fatal(err)
				}
				t.Log("CTRF_EXPECTED_FAILURE " + string(b))
			}
		})
	}
}
