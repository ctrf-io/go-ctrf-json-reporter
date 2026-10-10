package reporter

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestIdentityLifecycle(t *testing.T) {
	input := `{"Time":"2026-10-07T10:00:00Z","Action":"run","Package":"example/pkg","Test":"TestCase/sub"}
{"Time":"2026-10-07T10:00:01Z","Action":"fail","Package":"example/pkg","Test":"TestCase/sub","Elapsed":1}
{"Time":"2026-10-07T10:00:02Z","Action":"run","Package":"example/pkg","Test":"TestCase/sub"}
{"Time":"2026-10-07T10:00:03Z","Action":"pass","Package":"example/pkg","Test":"TestCase/sub","Elapsed":1}`
	one, err := ParseTestResults(strings.NewReader(input), false, nil, IdentityOptions{RunID: "run", ShardID: "one"})
	require.NoError(t, err)
	two, err := ParseTestResults(strings.NewReader(input), false, nil, IdentityOptions{RunID: "run", ShardID: "two"})
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
	repeated, err := ParseTestResults(strings.NewReader(input), false, nil, IdentityOptions{SeparateExecutions: true})
	require.NoError(t, err)
	require.Len(t, repeated.Results.Tests, 2)
	require.Equal(t, repeated.Results.Tests[0].TestID, repeated.Results.Tests[1].TestID)
	require.NotEqual(t, repeated.Results.Tests[0].ExecutionID, repeated.Results.Tests[1].ExecutionID)
	require.Empty(t, repeated.Results.Tests[0].RetryAttempts)
	require.Empty(t, repeated.Validate())
}
