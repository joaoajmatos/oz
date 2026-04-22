package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCrystallizeCmd_AcceptAll_HeuristicPromotesAndDeletesNote(t *testing.T) {
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
	crAcceptAll = true
	crForce = false
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

	if _, err := os.Stat(notePath); !os.IsNotExist(err) {
		t.Fatalf("expected note to be removed, stat err=%v", err)
	}

	// Ensure an ADR was written.
	decisionsDir := filepath.Join(dir, "specs", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "0001-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected an ADR file in %s, got %d entries", decisionsDir, len(entries))
	}

	// Ensure crystallize log exists and references the note.
	logPath := filepath.Join(dir, "context", "crystallize.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "notes/decision.md") {
		t.Fatalf("expected log to mention notes/decision.md, got:\n%s", string(data))
	}
}
