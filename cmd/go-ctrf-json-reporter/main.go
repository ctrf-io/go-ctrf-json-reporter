package main

import (
	"flag"
	"fmt"
	"github.com/jmaitrehenry/go-ctrf-json-reporter/ctrf"
	"github.com/jmaitrehenry/go-ctrf-json-reporter/reporter"
	"os"
)

var buildFailed bool

func main() {
	var outputFile string
	var verbose bool
	var quiet bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
	flag.BoolVar(&quiet, "quiet", false, "Disable all log output")
	flag.BoolVar(&quiet, "q", false, "Disable all log output (shorthand)")
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

	effectiveVerbose := verbose && !quiet

	report, err := reporter.ParseTestResults(os.Stdin, effectiveVerbose, env)
	if err != nil {
		if !quiet {
			_, _ = fmt.Fprintf(os.Stderr, "Error parsing test results: %v\n", err)
		}
		os.Exit(1)
	}

	err = reporter.WriteReportToFile(outputFile, report)
	if err != nil {
		if !quiet {
			_, _ = fmt.Fprintf(os.Stderr, "Error writing the report to file: %v\n", err)
		}
		os.Exit(1)
	}

	if !verbose && !quiet {
		buildOutput := reporter.GetBuildOutput()
		fmt.Println(buildOutput)
	}

	if report.Results.Extra != nil {
		extraMap := report.Results.Extra.(map[string]interface{})
		if _, ok := extraMap["buildFail"]; ok {
			buildFailed = true
		}
		if _, ok := extraMap["FailedBuild"]; ok {
			buildFailed = true
		}
	}

	if report.Results.Summary.Failed > 0 {
		buildFailed = true
	}

	if buildFailed {
		os.Exit(1)
	}
}
