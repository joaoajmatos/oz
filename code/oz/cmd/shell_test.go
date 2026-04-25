package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunShellRunExitCodePropagation(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	shellMode = "raw"
	shellTee = "never"
	shellNoTrack = true
	shellJSON = false
	shellUltraCompact = false

	err := runShellRun(cmd, []string{"sh", "-c", "exit 5"})
	if err == nil {
		t.Fatalf("expected non-nil error for non-zero exit")
	}

	var withCode interface{ ExitCode() int }
	if !errors.As(err, &withCode) {
		t.Fatalf("expected exit-code carrying error, got %T", err)
	}
	if withCode.ExitCode() != 5 {
		t.Fatalf("ExitCode=%d, want 5", withCode.ExitCode())
	}
}

func TestRunShellRunRawPassthrough(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	shellMode = "raw"
	shellTee = "never"
	shellNoTrack = true
	shellJSON = false
	shellUltraCompact = false

	err := runShellRun(cmd, []string{"sh", "-c", "echo out; echo err >&2"})
	if err != nil {
		t.Fatalf("runShellRun returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "out") {
		t.Fatalf("expected stdout passthrough, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "err") {
		t.Fatalf("expected stderr passthrough, got %q", stderr.String())
	}
}

func TestRunShellRunCompactFailureVisibility(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	shellMode = "compact"
	shellTee = "never"
	shellNoTrack = true
	shellJSON = false
	shellUltraCompact = false

	err := runShellRun(cmd, []string{"sh", "-c", "echo FAIL >&2; exit 2"})
	if err == nil {
		t.Fatalf("expected non-zero exit error")
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Fatalf("expected failure context in compact stderr, got %q", stderr.String())
	}
}
