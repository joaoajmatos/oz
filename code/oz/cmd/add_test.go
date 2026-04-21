package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/oz-tools/oz/internal/scaffold"
	"github.com/oz-tools/oz/internal/validate"
	"github.com/oz-tools/oz/internal/workspace"
)

func TestAddPM_HappyPath(t *testing.T) {
	dir := t.TempDir()
	cfg := scaffold.Config{
		Name:        "proj",
		Description: "test",
		CodeMode:    "inline",
		Agents:      []scaffold.AgentConfig{{Name: "coding", Type: "coding"}},
	}
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "pm", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Added package") || !strings.Contains(out, "agents/pm/AGENT.md") {
		t.Fatalf("unexpected stdout: %q", out)
	}

	ws, err := workspace.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	res := validate.Validate(ws)
	if !res.Valid() {
		t.Fatalf("validate after add pm: %+v", res.Findings)
	}
}

func TestAddPM_DuplicateWithoutForce(t *testing.T) {
	dir := t.TempDir()
	cfg := scaffold.Config{
		Name: "p", Description: "d", CodeMode: "inline",
		Agents: []scaffold.AgentConfig{{Name: "c", Type: "coding"}},
	}
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"add", "pm", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetArgs([]string{"add", "pm", dir})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error on duplicate add")
	}
}
