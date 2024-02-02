package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type TestEvent struct {
	Time    string  
	Action  string  
	Package string  
	Test    string  
	Elapsed float64 
	Output  string  
}

type Summary struct {
	Tests       int `json:"tests"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Pending     int `json:"pending"`
	Skipped     int `json:"skipped"`
	Other       int `json:"other"`
	Start		int `json:"start"`
	Stop		int `json:"stop"`
}

type TestResult struct {
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Duration float64 `json:"duration"`
}

type FinalReport struct {
	Results struct {
		Tool struct {
			Name string `json:"name"`
		} `json:"tool"`
		Summary Summary  `json:"summary"`
		Tests  []TestResult `json:"tests"`
	} `json:"results"`
}

func ParseTestResults(r io.Reader, verbose bool) (*FinalReport, error) {
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

	report := &FinalReport{}
	report.Results.Tool.Name = "gotest"
	report.Results.Summary = Summary{}
	report.Results.Tests = make([]TestResult, 0)

	for _, event := range testEvents {
		if verbose {
			jsonEvent, err := json.Marshal(event)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			fmt.Println(string(jsonEvent))
		}
		if event.Test != "" {
			if event.Action == "pass" {
				report.Results.Summary.Tests++
				report.Results.Summary.Passed++
				report.Results.Tests = append(report.Results.Tests, TestResult{
					Name:     event.Test,
					Status:   "passed",
					Duration: event.Elapsed,
				})
			} else if event.Action == "fail" {
				report.Results.Summary.Tests++
				report.Results.Summary.Failed++
				report.Results.Tests = append(report.Results.Tests, TestResult{
					Name:     event.Test,
					Status:   "failed",
					Duration: event.Elapsed,
				})
			} else if event.Action == "skip" {
				report.Results.Summary.Tests++
				report.Results.Summary.Skipped++
				report.Results.Tests = append(report.Results.Tests, TestResult{
					Name:     event.Test,
					Status:   "skipped",
					Duration: event.Elapsed,
				})
			}
		}
	}

	return report, nil
}

func WriteReportToFile(filename string, report *FinalReport) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error writing ctrf json report: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(report)
	if err != nil {
		return fmt.Errorf("error writing ctrf json report: %v", err)
	}

	fmt.Println("go-ctrf-json-reporter: successfully written ctrf json to", filename)
	return nil
}
