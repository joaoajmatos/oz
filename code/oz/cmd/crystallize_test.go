package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCrystallizeCmd_ReportOnly_DoesNotDeleteNote(t *testing.T) {
	dir := scaffoldValidWorkspace(t)

	// Add a high-confidence ADR-like note.
	notePath := filepath.Join(dir, "notes", "decision.md")
	if err := os.WriteFile(notePath, []byte(`# Decision

We decided to adopt X.

## Decision
We chose X.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Configure flags (non-interactive, heuristic-only).
	crDryRun = false
	crTopic = ""
	crNoEnrich = true
	crNoCache = true
	crVerbose = false

	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)
	c.SetIn(strings.NewReader(""))
	if err := runCrystallize(c, nil); err != nil {
		t.Fatalf("runCrystallize: %v", err)
	}

	// Default mode is report-only; no writes or deletes.
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("expected note to remain, stat err=%v", err)
	}

	// Ensure no crystallize log exists.
	logPath := filepath.Join(dir, "context", "crystallize.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected no log file in report-only mode, stat err=%v", err)
	}

	if !strings.Contains(buf.String(), "Next steps:") {
		t.Fatalf("expected Next steps in output, got:\n%s", buf.String())
	}
}

func TestCrystallizeCmd_DryRun_PrintsDiffAndDoesNotDeleteNote(t *testing.T) {
	dir := scaffoldValidWorkspace(t)

	notePath := filepath.Join(dir, "notes", "decision.md")
	if err := os.WriteFile(notePath, []byte(`# Decision

We decided to adopt X.

## Decision
We chose X.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	crDryRun = true
	crTopic = ""
	crNoEnrich = true
	crNoCache = true
	crVerbose = false

	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)
	c.SetIn(strings.NewReader(""))
	if err := runCrystallize(c, nil); err != nil {
		t.Fatalf("runCrystallize: %v", err)
	}

	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("expected note to remain, stat err=%v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Dry Run") {
		t.Fatalf("expected Dry Run section in output, got:\n%s", out)
	}
	if !strings.Contains(out, "--- notes/decision.md") {
		t.Fatalf("expected unified diff header in output, got:\n%s", out)
	}
}
