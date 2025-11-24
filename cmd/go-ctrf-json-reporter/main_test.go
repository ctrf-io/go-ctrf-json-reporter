package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir() // auto-cleanup when the test tears down

	t.Run("with no flags", func(t *testing.T) {
		t.Run("should error because no output file is provided", func(t *testing.T) {
			ctx := freshContext(nil, nil)

			err := execute(ctx)
			require.Error(t, err)
			require.ErrorContains(t, err, "no such file")
		})

		t.Run("should error because no report data is provided", func(t *testing.T) {
			ctx := freshContext(nil, nil)
			output := filepath.Join(tempDir, "test-report-ko.json")
			ctx.outputFile = output

			err := execute(ctx)
			require.Error(t, err)
			require.ErrorContains(t, err, "report is invalid")
		})

		t.Run("should parse go test json data", func(t *testing.T) {
			fixture, err := os.Open(filepath.Join("testdata", "test.json"))
			require.NoError(t, err)

			var stdout bytes.Buffer
			ctx := freshContext(&stdout, fixture)
			output := filepath.Join(tempDir, "test-report-ok.json")
			ctx.outputFile = output

			err = execute(ctx)
			require.NoError(t, err)
			require.FileExists(t, output)

			t.Run("stdout should contain some go test output", func(t *testing.T) {
				require.Contains(t, stdout.String(), "=== RUN   TestExecute")
			})

			t.Run("report file should be valid JSON", func(t *testing.T) {
				buf, err := os.ReadFile(output)
				require.NoError(t, err)

				jazon := make(map[string]any)
				require.NoError(t, json.Unmarshal(buf, &jazon))
				require.Contains(t, jazon, "reportFormat")
			})
		})
	})

	t.Run("with appName flag", func(t *testing.T) {
		t.Run("should parse go test json data", func(t *testing.T) {
			fixture, err := os.Open(filepath.Join("testdata", "test.json"))
			require.NoError(t, err)
			output := filepath.Join(tempDir, "test-report-app.json")

			ctx := freshContext(nil, fixture)
			ctx.appName = "my-app"
			ctx.outputFile = output

			err = execute(ctx)
			require.NoError(t, err)
			require.FileExists(t, output)

			t.Run("report file should be valid JSON", func(t *testing.T) {
				buf, err := os.ReadFile(output)
				require.NoError(t, err)

				jazon := make(map[string]any)
				require.NoError(t, json.Unmarshal(buf, &jazon))
				raw, ok := jazon["results"]
				require.True(t, ok)
				results, ok := raw.(map[string]any)
				require.True(t, ok)
				raw, ok = results["environment"]
				require.True(t, ok)
				environment, ok := raw.(map[string]any)
				require.True(t, ok)
				raw, ok = environment["appName"]
				require.True(t, ok)
				appName, ok := raw.(string)
				require.True(t, ok)
				require.Equal(t, "my-app", appName)
			})
		})
	})

	t.Run("with verbose flag", func(t *testing.T) {
		t.Run("should parse go test json data", func(t *testing.T) {
			fixture, err := os.Open(filepath.Join("testdata", "test.json"))
			require.NoError(t, err)
			output := filepath.Join(tempDir, "test-report-app.json")

			ctx := freshContext(nil, fixture)
			ctx.verbose = true
			ctx.outputFile = output

			err = execute(ctx)
			require.NoError(t, err)
			require.FileExists(t, output)

			t.Run("report file should be valid JSON, without environment", func(t *testing.T) {
				buf, err := os.ReadFile(output)
				require.NoError(t, err)

				jazon := make(map[string]any)
				require.NoError(t, json.Unmarshal(buf, &jazon))
				raw, ok := jazon["results"]
				require.True(t, ok)
				results, ok := raw.(map[string]any)
				require.True(t, ok)
				_, ok = results["environment"]
				require.False(t, ok)
			})
		})
	})
}

func freshContext(writer io.Writer, reader io.Reader) *commandContext {
	if reader == nil {
		reader = os.Stdin
	}
	if writer == nil {
		writer = new(bytes.Buffer)
	}

	return &commandContext{
		writer: writer,
		reader: reader,
	}
}
