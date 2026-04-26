package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/scaffold"
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

func TestScaffold_README_hasQuickstartAndCodeMode(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "README.md"))
	for _, s := range []string{"oz validate", "AGENTS.md", "oz context build", "Inline mode", "code/README.md"} {
		if !strings.Contains(content, s) {
			t.Errorf("README.md: missing expected snippet %q", s)
		}
	}
}

func TestScaffold_README_submoduleMode(t *testing.T) {
	dir := t.TempDir()
	cfg := defaultCfg
	cfg.CodeMode = "submodule"
	if err := scaffold.Scaffold(dir, cfg); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "README.md"))
	if !strings.Contains(content, "Submodule mode") {
		t.Error("README.md: expected Submodule mode row for code/")
	}
	if strings.Contains(content, "](./code/README.md)") {
		t.Error("README.md: should not link code/README.md in submodule mode")
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
	if !strings.Contains(content, "| Agent | Use when | Definition |") {
		t.Error("OZ.md: expected Registered Agents markdown table with Use when column")
	}
}

func TestScaffold_AGENTSmd_HasAgentRoutingTable(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}
	content := readFile(t, filepath.Join(dir, "AGENTS.md"))
	for _, want := range []string{"| Agent | Use when | Definition |", "| **coding** |", "| **maintainer** |", "`agents/coding/AGENT.md`", "`agents/maintainer/AGENT.md`"} {
		if !strings.Contains(content, want) {
			t.Errorf("AGENTS.md: missing %q\n%s", want, content)
		}
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

func TestWriteCursorHooks_IncludesShellRewriteHook(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.WriteCursorHooks(dir); err != nil {
		t.Fatal(err)
	}

	hooksJSON := readFile(t, filepath.Join(dir, ".cursor", "hooks.json"))
	for _, want := range []string{
		"\"command\": \".oz/hooks/oz-pre-commit.sh\"",
		"\"command\": \".oz/hooks/oz-shell-rewrite-cursor.sh\"",
		"\"command\": \".oz/hooks/oz-read-rewrite-cursor.sh\"",
		"\"command\": \".oz/hooks/oz-read-policy-cursor.sh\"",
		"\"matcher\": \"Read\"",
	} {
		if !strings.Contains(hooksJSON, want) {
			t.Errorf(".cursor/hooks.json: expected %q", want)
		}
	}

	for _, script := range []string{
		".oz/hooks/oz-session-init.sh",
		".oz/hooks/oz-after-edit.sh",
		".oz/hooks/oz-pre-commit.sh",
		".oz/hooks/oz-shell-rewrite-cursor.sh",
		".oz/hooks/oz-read-rewrite-cursor.sh",
		".oz/hooks/oz-read-policy-cursor.sh",
		".oz/hooks/oz-shell-rewrite-claude.sh",
		".oz/hooks/oz-shell-rewrite.sh",
	} {
		if _, err := os.Stat(filepath.Join(dir, script)); err != nil {
			t.Errorf("expected hook script %q to exist: %v", script, err)
		}
	}
}

func TestScaffold_SkillFiles(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	skill := "workspace-management"
	for _, f := range []string{
		filepath.Join("skills", skill, "SKILL.md"),
		filepath.Join("skills", skill, "references", "create-agent.md"),
		filepath.Join("skills", skill, "references", "create-skill.md"),
		filepath.Join("skills", skill, "references", "create-rule.md"),
		filepath.Join("skills", skill, "assets", "AGENT.md.tmpl"),
		filepath.Join("skills", skill, "assets", "SKILL.md.tmpl"),
		filepath.Join("skills", skill, "assets", "rule.md.tmpl"),
	} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected skill file %q to exist: %v", f, err)
		}
	}
}

func TestScaffold_SkillMD_HasRequiredSections(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.Scaffold(dir, defaultCfg); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(dir, "skills", "workspace-management", "SKILL.md"))
	for _, section := range []string{"## When to invoke", "## Steps"} {
		if !strings.Contains(content, section) {
			t.Errorf("SKILL.md: missing required section %q", section)
		}
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
