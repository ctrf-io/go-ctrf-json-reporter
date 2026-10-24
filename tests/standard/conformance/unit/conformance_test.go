package unit_test

import (
	"github.com/ctrf-io/go-ctrf-json-reporter/tests/native/fixtures"
	"testing"
)

func TestConformance(t *testing.T) { fixtures.Run(t, "unit") }
