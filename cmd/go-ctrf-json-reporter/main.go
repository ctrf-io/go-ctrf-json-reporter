package main

import (
	"flag"
	"fmt"
	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/ctrf-io/go-ctrf-json-reporter/reporter"
	"os"
)

func main() {
	var outputFile string
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
	flag.StringVar(&outputFile, "output", "ctrf-report.json", "The output file for the test results")
	flag.StringVar(&outputFile, "o", "ctrf-report.json", "The output file for the test results (shorthand)")

	var tempAppName, tempAppVersion, tempOSPlatform, tempOSRelease, tempOSVersion, tempBuildName, tempBuildNumber string

	flag.StringVar(&tempAppName, "appName", "", "The name of the application being tested.")
	flag.StringVar(&tempAppVersion, "appVersion", "", "The version of the application being tested.")
	flag.StringVar(&tempOSPlatform, "osPlatform", "", "The operating system platform (e.g., Windows, Linux).")
	flag.StringVar(&tempOSRelease, "osRelease", "", "The release version of the operating system.")
	flag.StringVar(&tempOSVersion, "osVersion", "", "The version number of the operating system.")
	flag.StringVar(&tempBuildName, "buildName", "", "The name of the build (e.g., feature branch name).")
	flag.StringVar(&tempBuildNumber, "buildNumber", "", "The build number or identifier.")

	flag.Parse()

	var env *ctrf.Environment

	if tempAppName != "" || tempAppVersion != "" || tempOSPlatform != "" ||
		tempOSRelease != "" || tempOSVersion != "" || tempBuildName != "" || tempBuildNumber != "" {
		env = &ctrf.Environment{}

		if tempAppName != "" {
			env.AppName = tempAppName
		}
		if tempAppVersion != "" {
			env.AppVersion = tempAppVersion
		}
		if tempOSPlatform != "" {
			env.OSPlatform = tempOSPlatform
		}
		if tempOSRelease != "" {
			env.OSRelease = tempOSRelease
		}
		if tempOSVersion != "" {
			env.OSVersion = tempOSVersion
		}
		if tempBuildName != "" {
			env.BuildName = tempBuildName
		}
		if tempBuildNumber != "" {
			env.BuildNumber = tempBuildNumber
		}
	}

	report, err := reporter.ParseTestResults(os.Stdin, verbose, env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing test results: %v\n", err)
		os.Exit(1)
	}

	err = reporter.WriteReportToFile(outputFile, report)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error writing the report to file: %v\n", err)
		os.Exit(1)
	}
}
