# Go CTRF JSON format support

## Go JSON Reporter

A Go JSON test reporter to create test reports that follow the CTRF standard.

[Common Test Report Format](https://ctrf.io) ensures the generation of uniform JSON test reports, independent of programming languages or test framework in use.

<div align="center">
<div style="padding: 1.5rem; border-radius: 8px; margin: 1rem 0; border: 1px solid #30363d;">
<span style="font-size: 23px;">💚</span>
<h3 style="margin: 1rem 0;">CTRF tooling is open source and free to use</h3>
<p style="font-size: 16px;">You can support the project with a follow and a star</p>

<div style="margin-top: 1.5rem;">
<a href="https://github.com/ctrf-io/go-ctrf-json-reporter">
<img src="https://img.shields.io/github/stars/ctrf-io/go-ctrf-json-reporter?style=for-the-badge&color=2ea043" alt="GitHub stars">
</a>
<a href="https://github.com/ctrf-io">
<img src="https://img.shields.io/github/followers/ctrf-io?style=for-the-badge&color=2ea043" alt="GitHub followers">
</a>
</div>
</div>

<p style="font-size: 14px; margin: 1rem 0;">
Maintained by <a href="https://github.com/ma11hewthomas">Matthew Thomas</a><br/>
Contributions are very welcome! <br/>
Explore more <a href="https://www.ctrf.io/integrations">integrations</a>
</p>
</div>

## Features

- Generate JSON test reports that are [CTRF](https://ctrf.io) compliant
- Straightforward integration with Go

```json
{
  "results": {
    "tool": {
      "name": "gotest"
    },
    "summary": {
      "tests": 1,
      "passed": 1,
      "failed": 0,
      "pending": 0,
      "skipped": 0,
      "other": 0,
      "start": 1706828654274,
      "stop": 1706828655782
    },
    "tests": [
      {
        "name": "ctrf should generate the same report with any tool",
        "status": "passed",
        "duration": 100
      }
    ],
    "environment": {
      "appName": "MyApp",
      "buildName": "MyBuild",
      "buildNumber": "1"
    }
  }
}
```

## Installation

To install go-ctrf-json-reporter, ensure you have Go installed on your system, then run:

``` bash
go install github.com/ctrf-io/go-ctrf-json-reporter/cmd/go-ctrf-json-reporter@latest
```

This command will install the latest version of go-ctrf-json-reporter.

After installation, you can use go-ctrf-json-reporter by piping the output of go test -json into it:

``` bash
go test -json ./... | go-ctrf-json-reporter -output ctrf-report.json
```

## Reporter Options

``` bash
go test -json ./... | go-ctrf-json-reporter \
-output custom-name.json \
-verbose \
-quiet \
-appName "MyApp" \
-appVersion "1.0.0" \
-osPlatform "Linux" \
-osRelease "18.04" \
-osVersion "5.4.0" \
-buildName "MyAppBuild" \
-buildNumber "100"
```

## Integration with gotestsum

go-ctrf-json-reporter can be used in conjunction with gotestsum

``` bash
gotestsum --jsonfile gotestsum.json && go-ctrf-json-reporter < gotestsum.json
```

## Generate a CTRF JSON report in your own testing tool written in go

If you are writting your own testing tool and wish to generate a CTRF JSON report, you can use the `ctrf` package.

```go
import (
  "github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
)

func runTests(destinationReportFile string) error {
  env := ctrf.Environment{
    // add your environment details here
  }	
  report := ctrf.NewReport("my-awesome-testing-tool", &env)
    
  // run your tests and populate the report object here

  return report.WriteFile(destinationReportFile)
}

```

## Test Object Properties

The test object in the report includes the following [CTRF properties](https://ctrf.io/docs/schema/test):

| Name       | Type   | Required | Details                                                                             |
| ---------- | ------ | -------- | ----------------------------------------------------------------------------------- |
| `name`     | String | Required | The name of the test.                                                               |
| `status`   | String | Required | The outcome of the test. One of: `passed`, `failed`, `skipped`, `pending`, `other`. |
| `duration` | Number | Required | The time taken for the test execution, in milliseconds.                             |
| `message`  | String | Optional | The failure message if the test failed.                                             |
| `suite`    | String | Required | The name of go package containing the test.                                         |

## Troubleshoot

### Command Not Found

When running go-ctrf-json-reporter results in a "command not found" error this usually means that the Go bin directory is not in your system's PATH.

## Development

Contributions are welcome! See [Contributing](CONTRIBUTING.md) for more information.

### Build the binary

```bash
go build -o go-ctrf-json-reporter ./cmd/go-ctrf-json-reporter
```

### Running Tests

```bash
go test ./...
```

### Testing the Reporter

```bash
go test -json ./... | ./go-ctrf-json-reporter -output ctrf-report.json

cat ctrf-report.json
```

### Development Workflow

1. Make changes to the code
2. Run tests: `go test ./...`
3. Test the reporter: `go test -json ./... | ./go-ctrf-json-reporter -output test-results.json`
4. Check the generated CTRF report: `cat test-results.json`

### Troubleshooting

If you encounter issues with test execution, ensure your Go environment is set correctly for your platform:

```bash
# Check your Go environment
go env GOOS GOARCH

# Set for your platform if needed
export GOOS=darwin GOARCH=arm64  # For macOS ARM64
export GOOS=linux GOARCH=amd64   # For Linux x86_64
export GOOS=windows GOARCH=amd64 # For Windows x86_64
```

Make sure the binary is executable:

```bash
chmod +x go-ctrf-json-reporter
```

## What is CTRF?

CTRF is a universal JSON test report schema that addresses the lack of a standardized format for JSON test reports.

**Consistency Across Tools:** Different testing tools and frameworks often produce reports in varied formats. CTRF ensures a uniform structure, making it easier to understand and compare reports, regardless of the testing tool used.

**Language and Framework Agnostic:** It provides a universal reporting schema that works seamlessly with any programming language and testing framework.

**Facilitates Better Analysis:** With a standardized format, programatically analyzing test outcomes across multiple platforms becomes more straightforward.

## Support Us

If you find this project useful, consider giving it a GitHub star ⭐ It means a lot to us.
