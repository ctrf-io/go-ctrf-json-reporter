package main

import (
	"flag"
	"fmt"
	"myreporter"
	"os"
)

func main() {

	var outputFile string
	var verbose bool

	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")

	flag.StringVar(&outputFile, "output", "ctrf-report.json", "The output file for the test results")
	flag.StringVar(&outputFile, "o", "ctrf-report.json", "The output file for the test results (shorthand)")

	flag.Parse()

	report, err := reporter.ParseTestResults(os.Stdin, verbose) 
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing test results: %v\n", err)
		os.Exit(1)
	}

	err = reporter.WriteReportToFile(outputFile, report) 
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing the report to file: %v\n", err)
		os.Exit(1)
	}
}
