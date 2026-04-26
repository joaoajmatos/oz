package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/scaffold"
	"github.com/joaoajmatos/oz/internal/validate"
	"github.com/joaoajmatos/oz/internal/workspace"
)

func TestAddList_IncludesIntegrationsAndPackages(t *testing.T) {
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"add", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	for _, want := range []string{"Integrations", "Optional packages", "claude", "cursor", "pm"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q\n%s", want, out)
		}
	}
}

func TestAddList_AliasLs(t *testing.T) {
	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"add", "ls"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Optional packages") {
		t.Fatal("alias ls should run list")
	}
}

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

func TestAddCursor_WritesShellRewriteHook(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}

	cfg := scaffold.Config{
		Name: "p", Description: "d", CodeMode: "inline",
		Agents: []scaffold.AgentConfig{{Name: "c", Type: "coding"}},
	}
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "cursor", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr.String())
	}

	hooksJSON, err := os.ReadFile(filepath.Join(dir, ".cursor", "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	for _, want := range []string{
		"\"command\": \".oz/hooks/oz-pre-commit.sh\"",
		"\"command\": \".oz/hooks/oz-shell-rewrite-cursor.sh\"",
		"\"command\": \".oz/hooks/oz-read-rewrite-cursor.sh\"",
		"\"command\": \".oz/hooks/oz-read-policy-cursor.sh\"",
		"\"matcher\": \"Read\"",
	} {
		if !strings.Contains(string(hooksJSON), want) {
			t.Errorf(".cursor/hooks.json: expected %q", want)
		}
	}

	for _, rel := range []string{
		"skills/oz/SKILL.md",
		"skills/oz/references/audit-and-validate.md",
		"skills/oz/references/context-and-mcp.md",
		"skills/oz-shell/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}

	for _, rel := range []string{
		".cursor/skills-cursor/oz/SKILL.md",
		".cursor/skills-cursor/oz/references/audit-and-validate.md",
		".cursor/skills-cursor/oz/references/context-and-mcp.md",
		".cursor/skills-cursor/oz-shell/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(home, rel)); err != nil {
			t.Errorf("expected global cursor skill %s to exist: %v", rel, err)
		}
	}
}

func TestAddClaude_WritesGlobalCursorSkills(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
	})
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}

	cfg := scaffold.Config{
		Name: "p", Description: "d", CodeMode: "inline",
		Agents: []scaffold.AgentConfig{{Name: "c", Type: "coding"}},
	}
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"add", "claude", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v stderr=%s", err, stderr.String())
	}

	for _, rel := range []string{
		"skills/oz/SKILL.md",
		"skills/oz/references/audit-and-validate.md",
		"skills/oz/references/context-and-mcp.md",
		"skills/oz-shell/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}

	for _, rel := range []string{
		".cursor/skills-cursor/oz/SKILL.md",
		".cursor/skills-cursor/oz/references/audit-and-validate.md",
		".cursor/skills-cursor/oz/references/context-and-mcp.md",
		".cursor/skills-cursor/oz-shell/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(home, rel)); err != nil {
			t.Errorf("expected global cursor skill %s to exist: %v", rel, err)
		}
	}
}
