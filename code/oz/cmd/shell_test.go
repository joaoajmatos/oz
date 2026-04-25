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

	"github.com/joaoajmatos/oz/internal/shell/gain"
	shellrun "github.com/joaoajmatos/oz/internal/shell/run"
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
	prevReadMaxLines := shellReadMaxLines
	prevReadTailLines := shellReadTailLines
	prevReadLineNumbers := shellReadLineNumbers
	prevReadNoFilter := shellReadNoFilter
	prevReadJSON := shellReadJSON
	prevReadUltraCompact := shellReadUltraCompact
	prevReadNoTrack := shellReadNoTrack
	prevPipeFilter := shellPipeFilter
	prevPipePassthrough := shellPipePassthrough
	prevPipeJSON := shellPipeJSON
	prevPipeUltraCompact := shellPipeUltraCompact
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
		shellReadMaxLines = prevReadMaxLines
		shellReadTailLines = prevReadTailLines
		shellReadLineNumbers = prevReadLineNumbers
		shellReadNoFilter = prevReadNoFilter
		shellReadJSON = prevReadJSON
		shellReadUltraCompact = prevReadUltraCompact
		shellReadNoTrack = prevReadNoTrack
		shellPipeFilter = prevPipeFilter
		shellPipePassthrough = prevPipePassthrough
		shellPipeJSON = prevPipeJSON
		shellPipeUltraCompact = prevPipeUltraCompact
	})
}

func addShellReadFlagsForTest(cmd *cobra.Command) {
	if cmd.Flags().Lookup("max-lines") == nil {
		cmd.Flags().Int("max-lines", 0, "")
	}
	if cmd.Flags().Lookup("tail-lines") == nil {
		cmd.Flags().Int("tail-lines", 0, "")
	}
	if cmd.Flags().Lookup("line-numbers") == nil {
		cmd.Flags().Bool("line-numbers", false, "")
	}
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

	shellMode = string(shellrun.ModeRaw)
	shellTee = string(shellrun.TeeModeNever)
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
	addShellReadFlagsForTest(cmd)
	addShellReadFlagsForTest(cmd)

	shellMode = string(shellrun.ModeRaw)
	shellTee = string(shellrun.TeeModeNever)
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

func TestRunShellRunJSONEnvelopeMatchedFilters(t *testing.T) {
	runShellJSONCase := func(t *testing.T, args []string, wantFilter string) {
		t.Helper()
		lockShellTest(t)
		saveShellGlobals(t)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		cmd := &cobra.Command{}
		cmd.SetOut(stdout)
		cmd.SetErr(stderr)

		shellMode = string(shellrun.ModeCompact)
		shellTee = string(shellrun.TeeModeNever)
		shellNoTrack = true
		shellJSON = true
		shellUltraCompact = false

		if err := runShellRun(cmd, args); err != nil {
			t.Fatalf("runShellRun: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal envelope: %v", err)
		}
		got, _ := payload["matched_filter"].(string)
		if got != wantFilter {
			t.Fatalf("matched_filter=%q want %q (stdout=%s)", got, wantFilter, stdout.String())
		}
		if code, ok := payload["exit_code"].(float64); !ok || int(code) != 0 {
			t.Fatalf("exit_code=%v want 0", payload["exit_code"])
		}
	}

	t.Run("json_sniff", func(t *testing.T) {
		t.Parallel()
		runShellJSONCase(t, []string{"sh", "-c", `echo '{"sniff":true,"n":42}'`}, "json")
	})
	t.Run("git_log", func(t *testing.T) {
		t.Parallel()
		runShellJSONCase(t, []string{"git", "log", "-1", "--oneline"}, "git.log")
	})
	t.Run("wc", func(t *testing.T) {
		t.Parallel()
		p := filepath.Join(t.TempDir(), "wcfile.txt")
		if err := os.WriteFile(p, []byte("hello world\n"), 0o644); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
		runShellJSONCase(t, []string{"wc", p}, "wc")
	})
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

	shellMode = string(shellrun.ModeCompact)
	shellTee = string(shellrun.TeeModeNever)
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
	shellGainPeriod = string(gain.PeriodDaily)
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
	shellGainPeriod = string(gain.PeriodWeekly)
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
	if payload["period"].(string) != string(gain.PeriodWeekly) {
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
	shellGainPeriod = string(gain.PeriodDaily)
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

func TestRunShellReadFile(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	path := filepath.Join(t.TempDir(), "sample.go")
	content := "package main\n\n// comment\nfunc main() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	shellReadNoTrack = true
	if err := runShellRead(cmd, []string{path}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}
	if stdout.String() == "" {
		t.Fatalf("expected non-empty output")
	}
	if strings.Contains(stdout.String(), "// comment") {
		t.Fatalf("expected go reader to compact comments, got %q", stdout.String())
	}
}

func TestRunShellReadStdin(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader("hello\nworld\n"))
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = true
	if err := runShellRead(cmd, []string{"-"}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}
	if !strings.Contains(stdout.String(), "hello") {
		t.Fatalf("expected stdin content in output, got %q", stdout.String())
	}
}

func TestRunShellReadMissingFile(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	path := filepath.Join(t.TempDir(), "ok.txt")
	if err := os.WriteFile(path, []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = true
	err := runShellRead(cmd, []string{path, filepath.Join(t.TempDir(), "missing.txt")})
	if err == nil {
		t.Fatalf("expected missing file error")
	}
	var withCode interface{ ExitCode() int }
	if !errors.As(err, &withCode) || withCode.ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got %v", err)
	}
	if !strings.Contains(stderr.String(), "missing file") {
		t.Fatalf("expected missing-file stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("expected valid file output to still be present")
	}
}

func TestRunShellReadTailLines(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	path := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(path, []byte("1\n2\n3\n4\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = true
	shellReadTailLines = 2
	if err := cmd.Flags().Set("tail-lines", "2"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := runShellRead(cmd, []string{path}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}
	out := strings.TrimSpace(stdout.String())
	if out != "3\n4" {
		t.Fatalf("tail output=%q, want %q", out, "3\n4")
	}
}

func TestRunShellReadLineNumbers(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	lines := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		lines = append(lines, "x")
	}
	path := filepath.Join(t.TempDir(), "lines.txt")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = true
	shellReadLineNumbers = true
	if err := cmd.Flags().Set("line-numbers", "true"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := runShellRead(cmd, []string{path}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "  1|x") {
		t.Fatalf("expected right-aligned first line number, got %q", out)
	}
	if !strings.Contains(out, "100|x") {
		t.Fatalf("expected line 100, got %q", out)
	}
}

func TestRunShellReadJSONEnvelopeFields(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	path := filepath.Join(t.TempDir(), "sample.go")
	content := "package main\n\n// comment\nfunc main() {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = true
	shellReadJSON = true
	if err := runShellRead(cmd, []string{path}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}

	var payload []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("len(payload)=%d want 1", len(payload))
	}
	item := payload[0]
	if item["schema_version"] != "1" {
		t.Fatalf("schema_version=%v want 1", item["schema_version"])
	}
	if item["lang"] != "go" {
		t.Fatalf("lang=%v want go", item["lang"])
	}
	if item["matched_filter"] != "read.go" {
		t.Fatalf("matched_filter=%v want read.go", item["matched_filter"])
	}
	if _, ok := item["stdout"].(string); !ok {
		t.Fatalf("stdout missing or non-string: %#v", item["stdout"])
	}
}

func TestRunShellReadDuplicateStdinWarnsOnce(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader("hello\nworld\n"))
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = true
	if err := runShellRead(cmd, []string{"-", "-"}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}
	if !strings.Contains(stderr.String(), "duplicate stdin marker") {
		t.Fatalf("expected duplicate stdin warning, got %q", stderr.String())
	}
}

func TestRunShellReadTracksCanonicalCommand(t *testing.T) {
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

	path := filepath.Join(t.TempDir(), "tracked.txt")
	if err := os.WriteFile(path, []byte("tracked\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	addShellReadFlagsForTest(cmd)

	shellReadNoTrack = false
	if err := runShellRead(cmd, []string{path}); err != nil {
		t.Fatalf("runShellRead: %v", err)
	}

	store, err := track.Open(track.DefaultPath())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	runs, err := store.Query(track.QueryOpts{Limit: 1})
	if err != nil {
		t.Fatalf("query runs: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("expected at least one tracked run")
	}
	want := "oz shell read -- " + path
	if runs[0].Command != want {
		t.Fatalf("tracked command=%q, want %q", runs[0].Command, want)
	}
}

func TestRunShellPipePassthrough(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("a\nb\n"))

	shellPipePassthrough = true
	if err := runShellPipe(cmd, nil); err != nil {
		t.Fatalf("runShellPipe: %v", err)
	}
	if got := stdout.String(); got != "a\nb\n" {
		t.Fatalf("stdout=%q want %q", got, "a\nb\n")
	}
}

func TestRunShellPipeExplicitFilter(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("src/main.rs:42:fn main() {}\n"))

	shellPipeFilter = "rg"
	if err := runShellPipe(cmd, nil); err != nil {
		t.Fatalf("runShellPipe: %v", err)
	}
	if !strings.Contains(stdout.String(), "src/main.rs") {
		t.Fatalf("expected rg compact output, got %q", stdout.String())
	}
}

func TestRunShellPipeAutoDetectFind(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("./a/b/c.go\n./a/b/d.go\n./x/y/z.go\n"))

	shellPipeFilter = "auto"
	if err := runShellPipe(cmd, nil); err != nil {
		t.Fatalf("runShellPipe: %v", err)
	}
	if !strings.Contains(stdout.String(), "find summary") {
		t.Fatalf("expected find compact output, got %q", stdout.String())
	}
}

func TestRunShellPipeJSONEnvelope(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	stdout := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{"ok":true}` + "\n"))

	shellPipeFilter = "json"
	shellPipeJSON = true
	if err := runShellPipe(cmd, nil); err != nil {
		t.Fatalf("runShellPipe: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload["matched_filter"] != "json" {
		t.Fatalf("matched_filter=%v want json", payload["matched_filter"])
	}
}

func TestRunShellPipeUnknownFilter(t *testing.T) {
	lockShellTest(t)
	saveShellGlobals(t)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("x\n"))

	shellPipeFilter = "nope"
	err := runShellPipe(cmd, nil)
	if err == nil {
		t.Fatalf("expected error for unknown filter")
	}
	if !strings.Contains(err.Error(), "unknown pipe filter") {
		t.Fatalf("unexpected error: %v", err)
	}
}
