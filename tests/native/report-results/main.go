package main

import (
	"flag"
	"fmt"
	"github.com/ctrf-io/go-ctrf-json-reporter/tests/native/reporting"
	s "github.com/ctrf-io/go-ctrf-json-reporter/tests/standard/conformance/shared"
	"os"
	"path/filepath"
)

func main() {
	input := flag.String("input", "", "go test records")
	output := flag.String("output", "", "CTRF output")
	scope := flag.String("scope", "reporter-tests", "result scope")
	exit := flag.Int("exit", 0, "native runner exit")
	inventoryPath := flag.String("inventory", "", "independent registered-test inventory")
	flag.Parse()
	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "input and output are required")
		os.Exit(1)
	}
	f, err := os.Open(*input)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	var inventory reporting.Inventory
	if *inventoryPath != "" {
		in, err := os.Open(*inventoryPath)
		if err != nil {
			fail(err)
		}
		inventory, err = reporting.Discover(in)
		in.Close()
		if err != nil {
			fail(err)
		}
	}
	b, err := reporting.ReportWithInventory(f, *scope, *exit, inventory)
	if err != nil {
		fail(err)
	}
	if err = os.MkdirAll(filepath.Dir(*output), 0700); err != nil {
		fail(err)
	}
	if err = os.WriteFile(*output, append(b, '\n'), 0600); err != nil {
		fail(err)
	}
	report, _ := s.Decode(b)
	summary := s.Summary(report)
	fmt.Printf("%s: %v passed, %v failed, %v skipped, %v expected failures/other\n", *scope, summary["passed"], summary["failed"], summary["skipped"], summary["other"])
	if s.Num(summary["failed"]) > 0 {
		os.Exit(1)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
