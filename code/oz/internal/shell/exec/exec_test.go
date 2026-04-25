package exec_test

import (
	"testing"

	ozexec "github.com/joaoajmatos/oz/internal/shell/exec"
)

func TestRunSuccess(t *testing.T) {
	t.Parallel()

	result, err := ozexec.Run([]string{"sh", "-c", "echo hello"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode=%d, want 0", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Fatalf("expected stdout to be captured")
	}
}

func TestRunFailureExitCode(t *testing.T) {
	t.Parallel()

	result, err := ozexec.Run([]string{"sh", "-c", "echo boom >&2; exit 7"})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("ExitCode=%d, want 7", result.ExitCode)
	}
	if result.Stderr == "" {
		t.Fatalf("expected stderr to be captured")
	}
}

func TestRunStartFailure(t *testing.T) {
	t.Parallel()

	_, err := ozexec.Run([]string{"oz-command-that-does-not-exist-12345"})
	if err == nil {
		t.Fatalf("expected start failure error")
	}
}
