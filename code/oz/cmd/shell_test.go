package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joaoajmatos/oz/internal/shell/track"
	"github.com/spf13/cobra"
)

var shellTestMu sync.Mutex

func lockShellTest(t *testing.T) {
	t.Helper()
	shellTestMu.Lock()
	t.Cleanup(func() {
		shellTestMu.Unlock()
	})
}

func saveShellGlobals(t *testing.T) {
	t.Helper()
	prevMode := shellMode
	prevTee := shellTee
	prevJSON := shellJSON
	prevNoTrack := shellNoTrack
	prevUltra := shellUltraCompact
	prevGainJSON := shellGainJSON
	prevGainAllTime := shellGainAllTime
	prevGainDays := shellGainDays
	prevGainPeriod := shellGainPeriod
	prevVerbosity := shellVerbosity
	prevRewriteExclude := append([]string(nil), shellRewriteExclude...)
	t.Cleanup(func() {
		shellMode = prevMode
		shellTee = prevTee
		shellJSON = prevJSON
		shellNoTrack = prevNoTrack
		shellUltraCompact = prevUltra
		shellGainJSON = prevGainJSON
		shellGainAllTime = prevGainAllTime
		shellGainDays = prevGainDays
		shellGainPeriod = prevGainPeriod
		shellVerbosity = prevVerbosity
		shellRewriteExclude = prevRewriteExclude
	})
}

func TestRunShellRewriteKnownCommand(t *testing.T) {
	t.Parallel()
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	shellRewriteExclude = nil
	err := runShellRewrite(cmd, []string{"git status"})
	if err != nil {
		t.Fatalf("runShellRewrite returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "oz shell run -- git status") {
		t.Fatalf("expected rewritten command, got %q", stdout.String())
	}
}

func TestRunShellRewriteAlreadyWrappedExitCode(t *testing.T) {
	t.Parallel()
	lockShellTest(t)
	saveShellGlobals(t)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := runShellRewrite(cmd, []string{"oz shell run -- git status"})
	if err == nil {
		t.Fatalf("expected non-nil error for already wrapped command")
	}
	var withCode interface{ ExitCode() int }
	if !errors.As(err, &withCode) {
		t.Fatalf("expected exit-code carrying error, got %T", err)
	}
	if withCode.ExitCode() != 1 {
		t.Fatalf("ExitCode=%d, want 1", withCode.ExitCode())
	}
}

func TestRunShellRewriteCompound(t *testing.T) {
	t.Parallel()
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := runShellRewrite(cmd, []string{"git status && go test ./..."})
	if err != nil {
		t.Fatalf("runShellRewrite returned error: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	want := "oz shell run -- git status && oz shell run -- go test ./..."
	if out != want {
		t.Fatalf("rewritten=%q, want %q", out, want)
	}
}

func TestRunShellRewritePipeOnlyLeftSide(t *testing.T) {
	t.Parallel()
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := runShellRewrite(cmd, []string{"git status | wc -l"})
	if err != nil {
		t.Fatalf("runShellRewrite returned error: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	want := "oz shell run -- git status | wc -l"
	if out != want {
		t.Fatalf("rewritten=%q, want %q", out, want)
	}
}

func TestRunShellRewriteExcludedExitCode(t *testing.T) {
	t.Parallel()
	lockShellTest(t)
	saveShellGlobals(t)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	shellRewriteExclude = []string{"git"}

	err := runShellRewrite(cmd, []string{"git status"})
	if err == nil {
		t.Fatalf("expected non-nil error for excluded command")
	}
	var withCode interface{ ExitCode() int }
	if !errors.As(err, &withCode) {
		t.Fatalf("expected exit-code carrying error, got %T", err)
	}
	if withCode.ExitCode() != 2 {
		t.Fatalf("ExitCode=%d, want 2", withCode.ExitCode())
	}
}

func TestRunShellRunExitCodePropagation(t *testing.T) {
	t.Parallel()
	lockShellTest(t)
	saveShellGlobals(t)

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
	lockShellTest(t)
	saveShellGlobals(t)

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
	lockShellTest(t)
	saveShellGlobals(t)

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

func TestRunShellGainEmptyState(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)
	customDataHome := t.TempDir()
	original := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", original)
		}
	})
	if err := os.Setenv("XDG_DATA_HOME", customDataHome); err != nil {
		t.Fatalf("set XDG_DATA_HOME: %v", err)
	}

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	shellGainJSON = false
	shellGainAllTime = false
	shellGainDays = 90
	shellGainPeriod = "daily"
	if err := runShellGain(cmd, nil); err != nil {
		t.Fatalf("runShellGain returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "no tracked runs") {
		t.Fatalf("expected empty-state message, got %q", stdout.String())
	}
}

func TestRunShellGainJSON(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)
	customDataHome := t.TempDir()
	original := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", original)
		}
	})
	if err := os.Setenv("XDG_DATA_HOME", customDataHome); err != nil {
		t.Fatalf("set XDG_DATA_HOME: %v", err)
	}

	dbPath := filepath.Join(customDataHome, "oz", "shell-track.db")
	store, err := track.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	now := time.Now().Unix()
	if err := store.Insert(track.Run{
		Command:       "echo hello",
		RecordedAt:    now,
		DurationMs:    5,
		TokenBefore:   20,
		TokenAfter:    10,
		TokenSaved:    10,
		ReductionPct:  50,
		MatchedFilter: "generic",
		ExitCode:      0,
	}); err != nil {
		t.Fatalf("insert run: %v", err)
	}
	_ = store.Close()

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	shellGainJSON = true
	shellGainAllTime = true
	shellGainDays = 90
	shellGainPeriod = "weekly"
	if err := runShellGain(cmd, nil); err != nil {
		t.Fatalf("runShellGain returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	summary, ok := payload["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary missing or invalid: %#v", payload["summary"])
	}
	if got := int(summary["invocation_count"].(float64)); got != 1 {
		t.Fatalf("invocation_count=%d, want 1", got)
	}
	if payload["period"].(string) != "weekly" {
		t.Fatalf("period=%q, want weekly", payload["period"])
	}
}

func TestRunShellGainHumanOutputIncludesSections(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	customDataHome := t.TempDir()
	original := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", original)
		}
	})
	if err := os.Setenv("XDG_DATA_HOME", customDataHome); err != nil {
		t.Fatalf("set XDG_DATA_HOME: %v", err)
	}

	dbPath := filepath.Join(customDataHome, "oz", "shell-track.db")
	store, err := track.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	now := time.Now().Unix()
	_ = store.Insert(track.Run{Command: "git status", RecordedAt: now, DurationMs: 3, TokenBefore: 20, TokenAfter: 10, TokenSaved: 10, ReductionPct: 50, MatchedFilter: "git.status", ExitCode: 0})
	_ = store.Insert(track.Run{Command: "go test ./...", RecordedAt: now, DurationMs: 4, TokenBefore: 30, TokenAfter: 10, TokenSaved: 20, ReductionPct: 66, MatchedFilter: "go.test", ExitCode: 0})
	_ = store.Close()

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	shellGainJSON = false
	shellGainAllTime = true
	shellGainDays = 90
	shellGainPeriod = "daily"
	if err := runShellGain(cmd, nil); err != nil {
		t.Fatalf("runShellGain returned error: %v", err)
	}
	out := stdout.String()
	for _, needle := range []string{"Trend", "Command breakdown", "Top savers"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected %q in output, got:\n%s", needle, out)
		}
	}
}
