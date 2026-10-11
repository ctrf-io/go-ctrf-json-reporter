package reporter

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ctrf-io/go-ctrf-json-reporter/ctrf"
	"github.com/google/uuid"
)

// IdentityOptions supplies optional run, shard, and test identity.
type IdentityOptions struct {
	RunID          string
	ShardID        string
	TestIDResolver func(packageName, testName string) string
}

func validateIdentityOptions(options IdentityOptions) error {
	if (options.RunID != "" && strings.TrimSpace(options.RunID) == "") ||
		(options.ShardID != "" && strings.TrimSpace(options.ShardID) == "") {
		return errors.New("runId and shardId must be non-empty")
	}
	return nil
}

func applyReportIdentity(report *ctrf.Report, options IdentityOptions) {
	report.RunID = options.RunID
	if options.ShardID != "" {
		env := ctrf.Environment{}
		if report.Results.Environment != nil {
			env = *report.Results.Environment
		}
		env.ShardID = options.ShardID
		report.Results.Environment = &env
	}
}

func applyTestIdentity(test *ctrf.TestResult, packageName string, options IdentityOptions) error {
	tuple, err := json.Marshal([]string{packageName, test.Name})
	if err != nil {
		return fmt.Errorf("encode test identity: %w", err)
	}
	test.TestID = fmt.Sprintf("ctrf:go:v1:%x", sha256.Sum256(tuple))
	if options.TestIDResolver != nil {
		test.TestID = options.TestIDResolver(packageName, test.Name)
	}
	if strings.TrimSpace(test.TestID) == "" {
		return errors.New("testId resolver must return a non-empty string")
	}
	test.ExecutionID = uuid.NewString()
	test.AttemptID = uuid.NewString()
	return nil
}
