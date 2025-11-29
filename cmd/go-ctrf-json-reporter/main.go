package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/ctrf-io/go-ctrf-json-reporter/reporter"
)

// commandContext holds the global context of the command.
//
// For now, this boils down to just CLI flags and default stdin/stdout.
type commandContext struct {
	commandFlags

	reader io.Reader
	writer io.Writer // makes it easier to test execute() independently
}

// commandFlags stores parsed command line flags.
type commandFlags struct {
	outputFile  string
	verbose     bool
	quiet       bool
	appName     string
	appVersion  string
	oSPlatform  string
	oSRelease   string
	oSVersion   string
	buildName   string
	buildNumber string
}

// NOTE(fredbi)
// Suggestions (future enhancements):
//
//   - outputFile could be provided as an io.Writer: this makes the package easier to test
//   - outputFile is currently required but could default to stdout
//   - outputFile set to "-" would also mean stdout (common with unix-like tools)
//
// A similar approach could work for stdin, which is currently not an option, when CLI args (not flags)
// could represent the input files (e.g. could be useful when used with xargs for example).

func main() {
	var ctx commandContext
	ctx.reader = os.Stdin
	ctx.writer = os.Stdout

	registerFlags(&ctx.commandFlags)

	if err := execute(&ctx); err != nil {
		if ctx.quiet {
			os.Exit(1) // exit silently
		}

		log.Fatalf("%v", err)
	}
}

func execute(cmd *commandContext) error {
	env := ctrfEnvFromFlags(cmd)
	effectiveVerbose := cmd.verbose && !cmd.quiet

	report, err := reporter.ParseTestResults(cmd.reader, effectiveVerbose, env)
	if err != nil {
		return fmt.Errorf("error parsing test results: %w", err)
	}

	err = reporter.WriteReportToFile(cmd.outputFile, report)
	if err != nil {
		return fmt.Errorf("error writing the report to file: %w", err)
	}

	if !cmd.verbose && !cmd.quiet { // when verbose is enabled, output is already written during parsing
		buildOutput := reporter.GetBuildOutput()
		fmt.Fprint(cmd.writer, buildOutput)
	}

	var buildFailed bool
	if report.Results.Extra != nil {
		extraMap, isMap := report.Results.Extra.(map[string]any)
		if !isMap {
			err = fmt.Errorf("expected a map, but got %T instead", report.Results.Extra)
			return fmt.Errorf("error extracting report results: %w", err)
		}
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
		return errors.New("build failed")
	}

	return nil
}

func registerFlags(flags *commandFlags) {
	flag.BoolVar(&flags.verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&flags.verbose, "v", false, "Enable verbose output (shorthand)")
	flag.BoolVar(&flags.quiet, "quiet", false, "Disable all log output")
	flag.BoolVar(&flags.quiet, "q", false, "Disable all log output (shorthand)")

	flag.StringVar(&flags.outputFile, "output", "ctrf-report.json", "The output file for the test results")
	flag.StringVar(&flags.outputFile, "o", "ctrf-report.json", "The output file for the test results (shorthand)")

	flag.StringVar(&flags.appName, "appName", "", "The name of the application being tested.")
	flag.StringVar(&flags.appVersion, "appVersion", "", "The version of the application being tested.")
	flag.StringVar(&flags.oSPlatform, "osPlatform", "", "The operating system platform (e.g., Windows, Linux).")
	flag.StringVar(&flags.oSRelease, "osRelease", "", "The release version of the operating system.")
	flag.StringVar(&flags.oSVersion, "osVersion", "", "The version number of the operating system.")
	flag.StringVar(&flags.buildName, "buildName", "", "The name of the build (e.g., feature branch name).")
	flag.StringVar(&flags.buildNumber, "buildNumber", "", "The build number or identifier.")

	// parsing errors result in os.Exit(1). Perhaps we should call the flagset version and capture the error instead.
	flag.Parse()
}

func ctrfEnvFromFlags(cmd *commandContext) *ctrf.Environment {
	if cmd.appName == "" && cmd.appVersion == "" && cmd.oSPlatform == "" &&
		cmd.oSRelease == "" && cmd.oSVersion == "" && cmd.buildName == "" &&
		cmd.buildNumber == "" {
		return nil
	}

	return &ctrf.Environment{
		AppName:     cmd.appName,
		AppVersion:  cmd.appVersion,
		OSPlatform:  cmd.oSPlatform,
		OSRelease:   cmd.oSRelease,
		OSVersion:   cmd.oSVersion,
		BuildName:   cmd.buildName,
		BuildNumber: cmd.buildNumber,
	}
}
