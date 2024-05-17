package ctrf

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredProperties(t *testing.T) {
	// Arrange
	report := Report{
		Results: &Results{
			Tool: &Tool{
				Name: "my tool",
			},
			Summary: &Summary{
				Tests:   15,
				Passed:  5,
				Failed:  4,
				Pending: 3,
				Skipped: 2,
				Other:   1,
				Start:   42,
				Stop:    1337,
			},
			Tests: []*TestResult{
				{
					Name:     "test 1",
					Status:   TestPassed,
					Duration: 10,
				},
				{
					Name:     "test 2",
					Status:   TestFailed,
					Duration: 11,
				},
				{
					Name:     "test 3",
					Status:   TestPending,
					Duration: 12,
				},
				{
					Name:     "test 4",
					Status:   TestSkipped,
					Duration: 13,
				},
				{
					Name:     "test 5",
					Status:   TestOther,
					Duration: 14,
				},
			},
		},
	}

	expectedJson := `{
  "results": {
    "tool": {
      "name": "my tool"
    },
    "summary": {
      "tests": 15,
      "passed": 5,
      "failed": 4,
      "pending": 3,
      "skipped": 2,
      "other": 1,
      "start": 42,
      "stop": 1337
    },
    "tests": [
      {
        "name": "test 1",
        "status": "passed",
        "duration": 10
      },
      {
        "name": "test 2",
        "status": "failed",
        "duration": 11
      },
      {
        "name": "test 3",
        "status": "pending",
        "duration": 12
      },
      {
        "name": "test 4",
        "status": "skipped",
        "duration": 13
      },
      {
        "name": "test 5",
        "status": "other",
        "duration": 14
      }
    ]
  }
}
`

	// Act
	actualJson, err := report.ToJsonPretty()
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assert.Equal(t, expectedJson, actualJson)
}

func TestValidation(t *testing.T) {
	forEachValidationTestCase(t, allTestCase, func(t *testing.T, testCase validationTestCase, report *Report) {
		errs := report.Validate()
		if len(testCase.ExpectedErrors) == 0 {
			assert.Nil(t, errs)
		} else {
			assert.NotNil(t, errs)
			for _, expectedError := range testCase.ExpectedErrors {
				if errorNotPresent(errs, expectedError) {
					t.Error("Expected error not found:", expectedError)
				}
			}
		}
	})
}

func TestWriteFailsForInvalidReport(t *testing.T) {
	forEachValidationTestCase(t, failingTestCase, func(t *testing.T, testCase validationTestCase, report *Report) {
		err := report.Write(os.Stdout, true)
		assert.Error(t, err, "report is invalid")
	})
}

func TestWriteFileFailsForInvalidReport(t *testing.T) {
	forEachValidationTestCase(t, failingTestCase, func(t *testing.T, testCase validationTestCase, report *Report) {
		err := report.WriteFile("report.json")
		assert.Error(t, err, "report is invalid")
	})
}

func TestToJsonFailsForInvalidReport(t *testing.T) {
	forEachValidationTestCase(t, failingTestCase, func(t *testing.T, testCase validationTestCase, report *Report) {
		_, err := report.ToJson()
		assert.Error(t, err, "report is invalid")
	})
}

func TestToJsonPrettyFailsForInvalidReport(t *testing.T) {
	forEachValidationTestCase(t, failingTestCase, func(t *testing.T, testCase validationTestCase, report *Report) {
		_, err := report.ToJsonPretty()
		assert.Error(t, err, "report is invalid")
	})
}

func forEachValidationTestCase(t *testing.T, testCaseFilter func(validationTestCase) bool, test func(*testing.T, validationTestCase, *Report)) {
	data, err := os.ReadFile("validation-test-cases.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var testCases []validationTestCase
	err = yaml.Unmarshal(data, &testCases)
	if err != nil {
		t.Fatal(err)
	}

	for _, testCase := range testCases {
		if !testCaseFilter(testCase) {
			continue
		}
		t.Run(testCase.Name, func(t *testing.T) {
			report := &Report{}
			err = json.Unmarshal([]byte(testCase.Report), report)
			if err != nil {
				t.Fatal(err)
			}
			test(t, testCase, report)
		})
	}
}

func allTestCase(validationTestCase) bool {
	return true
}

func failingTestCase(testCase validationTestCase) bool {
	return len(testCase.ExpectedErrors) > 0
}

func errorNotPresent(errs []error, theErrorYouAreLookingFor string) bool {
	for _, err := range errs {
		if err.Error() == theErrorYouAreLookingFor {
			return false
		}
	}
	return true
}

type validationTestCase struct {
	Name           string   `json:"name"          yaml:"name"`
	Report         string   `json:"report"        yaml:"report"`
	ExpectedErrors []string `json:"expected_errors" yaml:"expected_errors"`
}
