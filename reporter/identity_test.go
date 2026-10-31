package reporter

import (
	"io"
	"strings"
	"testing"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/stretchr/testify/require"
)

func TestIdentityLifecycle(t *testing.T) {
	var parser func(io.Reader, bool, *ctrf.Environment) (*ctrf.Report, error) = ParseTestResults //nolint:staticcheck // Compile-check the original public function signature.
	_ = parser
	input := `{"Time":"2026-10-07T10:00:00Z","Action":"run","Package":"example/pkg","Test":"TestCase/sub"}
{"Time":"2026-10-07T10:00:01Z","Action":"fail","Package":"example/pkg","Test":"TestCase/sub","Elapsed":1}
{"Time":"2026-10-07T10:00:02Z","Action":"run","Package":"example/pkg","Test":"TestCase/sub"}
{"Time":"2026-10-07T10:00:03Z","Action":"pass","Package":"example/pkg","Test":"TestCase/sub","Elapsed":1}`
	one, err := ParseTestResultsWithOptions(strings.NewReader(input), false, nil, IdentityOptions{RunID: "run", ShardID: "one"})
	require.NoError(t, err)
	two, err := ParseTestResultsWithOptions(strings.NewReader(input), false, nil, IdentityOptions{RunID: "run", ShardID: "two"})
	require.NoError(t, err)
	require.Equal(t, "run", one.RunID)
	require.Equal(t, "one", one.Results.Environment.ShardID)
	require.Equal(t, one.Results.Tests[0].TestID, two.Results.Tests[0].TestID)
	require.NotEqual(t, one.Results.Tests[0].ExecutionID, two.Results.Tests[0].ExecutionID)
	require.NotEmpty(t, one.Results.Tests[0].AttemptID)
	require.NotEqual(t, one.Results.Tests[0].AttemptID, one.Results.Tests[0].RetryAttempts[0].AttemptID)
	require.NotEmpty(t, one.Results.Tests[0].RetryAttempts[0].AttemptID)
	require.NotEqual(t, one.Results.Tests[0].ExecutionID, one.Results.Tests[0].RetryAttempts[0].AttemptID)
	require.Empty(t, one.Validate())
	repeated, err := ParseTestResults(strings.NewReader(input), false, nil)
	require.NoError(t, err)
	require.Empty(t, repeated.RunID)
	require.Len(t, repeated.Results.Tests, 1)
	require.Len(t, repeated.Results.Tests[0].RetryAttempts, 1)
}
