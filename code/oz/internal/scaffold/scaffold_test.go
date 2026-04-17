package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oz-tools/oz/internal/scaffold"
)

var defaultCfg = scaffold.Config{
	Name:        "testproject",
	Description: "a test project",
	CodeMode:    "inline",
	Agents: []scaffold.AgentConfig{
		{Name: "coding", Type: "coding"},
		{Name: "maintainer"},
	},
}

func TestScaffold_RequiredDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	for _, d := range []string{
		"agents", "specs/decisions", "docs", "context", "skills", "rules", "notes", "tools", "scripts", ".oz",
	} {
		info, err := os.Stat(filepath.Join(dir, d))
		if err != nil {
			t.Errorf("expected directory %q to exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", d)
		}
	}
}

func TestScaffold_RequiredRootFiles(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"AGENTS.md", "OZ.md", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected root file %q to exist: %v", f, err)
		}
	}
}

func TestScaffold_RootFilesContainProjectName(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	for _, f := range []string{"AGENTS.md", "OZ.md", "README.md"} {
		content := readFile(t, filepath.Join(dir, f))
		if !strings.Contains(content, "testproject") {
			t.Errorf("%s: expected project name %q in content", f, "testproject")
		}
	}
}

func TestScaffold_OZmd_ContainsVersion(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "OZ.md"))
	if !strings.Contains(content, "oz standard:") {
		t.Error("OZ.md: missing 'oz standard' field")
	}
}

func TestScaffold_AgentFiles(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	for _, agent := range defaultCfg.Agents {
		agentMD := filepath.Join(dir, "agents", agent.Name, "AGENT.md")
		if _, err := os.Stat(agentMD); err != nil {
			t.Errorf("expected AGENT.md for agent %q: %v", agent.Name, err)
		}
	}
}

func TestScaffold_CodingAgent_HasCodingReadChain(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "agents", "coding", "AGENT.md"))
	if !strings.Contains(content, "coding-guidelines.md") {
		t.Error("coding agent AGENT.md: expected coding-guidelines.md in read-chain")
	}
}

func TestScaffold_CodeMode_Inline(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultCfg
	cfg.CodeMode = "inline"
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "code", "README.md")); err != nil {
		t.Errorf("expected code/README.md for inline mode: %v", err)
	}
}

func TestScaffold_CodeMode_Submodule(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultCfg
	cfg.CodeMode = "submodule"
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "code", "README.md")); err == nil {
		t.Error("expected code/README.md to NOT exist for submodule mode")
	}
}

func TestScaffold_ClaudeMD_True(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultCfg
	cfg.ClaudeMD = true
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(content, "@AGENTS.md") {
		t.Error("CLAUDE.md: expected @AGENTS.md import")
	}
}

func TestScaffold_ClaudeMD_False(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultCfg
	cfg.ClaudeMD = false
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		t.Error("expected CLAUDE.md to NOT exist when ClaudeMD=false")
	}
}

func TestWriteCLAUDEMD(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.WriteCLAUDEMD(dir, "myproject", "my description"); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(content, "myproject") {
		t.Error("CLAUDE.md: expected project name")
	}
	if !strings.Contains(content, "@AGENTS.md") {
		t.Error("CLAUDE.md: expected @AGENTS.md import")
	}
}

// helpers

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}
