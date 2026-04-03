//go:build examples

package fake

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Flaky_Pass(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
}

func Test_Flaky_Fail(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	t.Fatal("This test is designed to fail.")
}

func Test_Flaky_Skipped(t *testing.T) {
	t.Skip("This test is designed to be skipped.")
	time.Sleep(50 * time.Millisecond)
}

func Test_Flaky_Flaky(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	file := path.Join(os.TempDir(), "flaky_test.tmp")
	if _, err := os.Stat(file); errors.Is(err, fs.ErrNotExist) {
		// First run: file does not exist, create it with count "1", and fail.
		if err := os.WriteFile(file, []byte("1"), 0644); err != nil {
			t.Fatalf("Failed to create flaky test file: %v", err)
		}
		t.Fatal("Flaky Failure (attempt 1)")
	} else {
		// File exists, read the current count
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read flaky test file: %v", err)
		}
		count, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			t.Fatalf("Failed to parse failure count: %v", err)
		}

		if count == 1 {
			// Second run: increment count to 2 and fail again
			if err := os.WriteFile(file, []byte("2"), 0644); err != nil {
				t.Fatalf("Failed to update flaky test file: %v", err)
			}
			t.Fatal("Flaky Failure (attempt 2)")
		} else if count == 2 {
			// Third run: remove file and pass
			if err := os.Remove(file); err != nil {
				t.Fatalf("Failed to remove flaky test file: %v", err)
			}
		}
	}
}
