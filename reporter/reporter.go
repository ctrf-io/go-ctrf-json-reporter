package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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

func ParseTestResults(r io.Reader, verbose bool, env *ctrf.Environment) (*ctrf.Report, error) {
	var testEvents []TestEvent
	decoder := json.NewDecoder(r)

	for {
		var event TestEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		testEvents = append(testEvents, event)
	}

	report := ctrf.NewReport("gotest", env)

	for _, event := range testEvents {
		if verbose {
			jsonEvent, err := json.Marshal(event)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			fmt.Println(string(jsonEvent))
		}
		if event.Test != "" {
			if event.Action == "pass" {
				report.Results.Summary.Tests++
				report.Results.Summary.Passed++
				report.Results.Tests = append(report.Results.Tests, &ctrf.TestResult{
					Name:     event.Test,
					Status:   ctrf.TestPassed,
					Duration: event.Elapsed,
				})
			} else if event.Action == "fail" {
				report.Results.Summary.Tests++
				report.Results.Summary.Failed++
				report.Results.Tests = append(report.Results.Tests, &ctrf.TestResult{
					Name:     event.Test,
					Status:   ctrf.TestFailed,
					Duration: event.Elapsed,
				})
			} else if event.Action == "skip" {
				report.Results.Summary.Tests++
				report.Results.Summary.Skipped++
				report.Results.Tests = append(report.Results.Tests, &ctrf.TestResult{
					Name:     event.Test,
					Status:   ctrf.TestSkipped,
					Duration: event.Elapsed,
				})
			}
		}
	}

	return report, nil
}

func WriteReportToFile(filename string, report *ctrf.Report) error {
	err := report.WriteFile(filename)
	if err != nil {
		return err
	}

	fmt.Println("go-ctrf-json-reporter: successfully written ctrf json to", filename)
	return nil
}
