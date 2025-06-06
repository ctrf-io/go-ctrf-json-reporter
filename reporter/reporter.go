package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
)

type TestEvent struct {
	Time    string
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}

var buildOutput []string

func ParseTestResults(r io.Reader, verbose bool, env *ctrf.Environment) (*ctrf.Report, error) {
	var testEvents []TestEvent
	decoder := json.NewDecoder(r)

	report := ctrf.NewReport("gotest", env)
	report.Results.Summary.Start = time.Now().UnixNano() / int64(time.Millisecond)

	for {
		var event TestEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		testEvents = append(testEvents, event)

		if verbose {
			if event.Action == "build-output" || event.Action == "output" {
				fmt.Print(event.Output)
			}
		}
	}

	for i, event := range testEvents {

		if event.Action == "build-output" || event.Action == "build-fail" || event.Action == "fail" {
			if report.Results.Extra == nil {
				report.Results.Extra = make(map[string]interface{})
			}
			extraMap := report.Results.Extra.(map[string]interface{})

			if event.Action == "fail" {
				if _, ok := extraMap["FailedBuild"]; !ok {
					extraMap["FailedBuild"] = true
				}
			}

			if event.Action == "build-output" {
				if _, ok := extraMap["buildOutput"]; !ok {
					extraMap["buildOutput"] = []TestEvent{}
				}
				buildOutputEvents := extraMap["buildOutput"].([]TestEvent)
				extraMap["buildOutput"] = append(buildOutputEvents, event)
				buildOutput = append(buildOutput, event.Output)
				continue
			}

			if event.Action == "build-fail" {
				if _, ok := extraMap["buildFail"]; !ok {
					extraMap["buildFail"] = []TestEvent{}
				}
				buildFailEvents := extraMap["buildFail"].([]TestEvent)
				extraMap["buildFail"] = append(buildFailEvents, event)
				break
			}
		}

		if event.Action == "output" {
			buildOutput = append(buildOutput, event.Output)
		}

		if event.Test == "" {
			continue
		}
		startTime, err := parseTimeString(event.Time)
		duration := secondsToMillis(event.Elapsed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing test event start time '%s' : %v\n", event.Time, err)
		} else {
			if report.Results.Summary.Start > startTime {
				report.Results.Summary.Start = startTime
			}
			endTime := startTime + duration
			if report.Results.Summary.Stop < endTime {
				report.Results.Summary.Stop = endTime
			}
		}
		if event.Action == "pass" {
			report.Results.Summary.Tests++
			report.Results.Summary.Passed++
			report.Results.Tests = append(report.Results.Tests, &ctrf.TestResult{
				Suite:    event.Package,
				Name:     event.Test,
				Status:   ctrf.TestPassed,
				Duration: duration,
			})
		} else if event.Action == "fail" {
			report.Results.Summary.Tests++
			report.Results.Summary.Failed++
			report.Results.Tests = append(report.Results.Tests, &ctrf.TestResult{
				Suite:    event.Package,
				Name:     event.Test,
				Status:   ctrf.TestFailed,
				Duration: duration,
				Message:  getMessagesForTest(testEvents, i, event.Package, event.Test),
			})
		} else if event.Action == "skip" {
			report.Results.Summary.Tests++
			report.Results.Summary.Skipped++
			report.Results.Tests = append(report.Results.Tests, &ctrf.TestResult{
				Suite:    event.Package,
				Name:     event.Test,
				Status:   ctrf.TestSkipped,
				Duration: duration,
			})
		}

	}

	enrichReportWithFilenames(report)

	return report, nil
}

func generateTestMap() map[string][]string {
	tests := map[string][]string{}

	r := regexp.MustCompile(`Test.\w+`)
	if err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if !strings.HasSuffix(path, "_test.go") || err != nil {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		tests[path] = r.FindAllString(string(data), -1)

		return nil
	}); err != nil {
		return tests
	}

	return tests
}

func enrichReportWithFilenames(report *ctrf.Report) {
	tests := generateTestMap()

	for i, testResult := range report.Results.Tests {
		for file, names := range tests {
			for _, name := range names {
				if strings.Contains(testResult.Name, name) {
					report.Results.Tests[i].Filepath = file
				}
			}
		}
	}
}

func getMessagesForTest(testEvents []TestEvent, index int, packageName, testName string) string {
	var messages []string
	for i := index; i >= 0; i-- {
		if testEvents[i].Package == packageName && testEvents[i].Test == testName {
			if testEvents[i].Action == "output" {
				messages = append(messages, testEvents[i].Output)
			}
		} else {
			break
		}
	}
	reverse(messages)
	return strings.Join(messages, "")
}

func WriteReportToFile(filename string, report *ctrf.Report) error {
	err := report.WriteFile(filename)
	if err != nil {
		return err
	}
	fmt.Println("go-ctrf-json-reporter: successfully written ctrf json to", filename)
	return nil
}

func GetBuildOutput() string {
	return strings.Join(buildOutput, "")
}

func secondsToMillis(seconds float64) int64 {
	return int64(seconds * 1000)
}

func parseTimeString(timeString string) (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, timeString)
	if err != nil {
		return 0, err
	}
	return t.UnixNano() / int64(time.Millisecond), nil
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
