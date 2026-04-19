package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/oz-tools/oz/internal/convention"
)

// Config holds the parameters collected during oz init.
type Config struct {
	Name        string
	Description string
	CodeMode    string // "inline" or "submodule"
	Agents      []AgentConfig
	ClaudeMD    bool // generate CLAUDE.md for Claude Code native integration
}

// AgentConfig describes a single agent to scaffold.
type AgentConfig struct {
	Name        string
	Description string
	Type        string // e.g. "coding" — controls which rules appear in the read-chain
}

// templateData is the data passed to every template.
type templateData struct {
	Name        string
	Description string
	OZVersion   string
	Agents      []AgentConfig
}

// scaffoldStep is one ordered phase of workspace scaffolding.
type scaffoldStep func() error

// Scaffold creates a full oz workspace at path using cfg.
func Scaffold(path string, cfg Config) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	data := templateData{
		Name:        cfg.Name,
		Description: cfg.Description,
		OZVersion:   convention.Version,
		Agents:      cfg.Agents,
	}

	steps := []scaffoldStep{
		func() error { return createDirectories(abs) },
		func() error { return createRootFiles(abs, data) },
		func() error { return createAgentFiles(abs, cfg.Agents) },
		func() error { return createDocFiles(abs, data) },
		func() error { return createSpecFiles(abs) },
		func() error { return createRulesFiles(abs) },
		func() error { return createSkillFiles(abs) },
		func() error { return createCodeDir(abs, cfg.CodeMode, data) },
		func() error { return createOZDir(abs) },
	}
	if cfg.ClaudeMD {
		steps = append(steps, func() error { return createClaudeMD(abs, data) })
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func createDirectories(root string) error {
	dirs := []string{
		"agents",
		"specs/decisions",
		"docs",
		"context",
		"skills",
		"rules",
		"notes",
		"code",
		"tools",
		"scripts",
		".oz",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

func createRootFiles(root string, data templateData) error {
	files := []struct {
		dest string
		tmpl string
	}{
		{".gitignore", "templates/root.gitignore.tmpl"},
		{"OZ.md", "templates/OZ.md.tmpl"},
		{"AGENTS.md", "templates/AGENTS.md.tmpl"},
		{"README.md", "templates/README.md.tmpl"},
	}
	for _, f := range files {
		if err := writeTemplate(filepath.Join(root, f.dest), f.tmpl, data); err != nil {
			return err
		}
	}
	return nil
}

func createAgentFiles(root string, agents []AgentConfig) error {
	for _, agent := range agents {
		dir := filepath.Join(root, "agents", agent.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating agent dir %s: %w", agent.Name, err)
		}
		if err := writeTemplate(filepath.Join(dir, "AGENT.md"), "templates/agent/AGENT.md.tmpl", agent); err != nil {
			return err
		}
	}
	return nil
}

func createDocFiles(root string, data templateData) error {
	files := []struct {
		dest string
		tmpl string
	}{
		{"docs/architecture.md", "templates/docs/architecture.md.tmpl"},
		{"docs/open-items.md", "templates/docs/open-items.md.tmpl"},
	}
	for _, f := range files {
		if err := writeTemplate(filepath.Join(root, f.dest), f.tmpl, data); err != nil {
			return err
		}
	}
	return nil
}

func createSpecFiles(root string) error {
	return writeTemplate(
		filepath.Join(root, "specs/decisions/_template.md"),
		"templates/specs/decisions/_template.md.tmpl",
		nil,
	)
}

func createRulesFiles(root string) error {
	return writeTemplate(
		filepath.Join(root, "rules/coding-guidelines.md"),
		"templates/rules/coding-guidelines.md.tmpl",
		nil,
	)
}

// createSkillFiles generates built-in skills into skills/ at workspace root.
// Built-in skills are always generated; they form the foundation of the
// oz-maintainer's toolset in every oz workspace.
func createSkillFiles(root string) error {
	skill := "create-workspace-artifact"
	base := filepath.Join(root, "skills", skill)

	dirs := []string{
		base,
		filepath.Join(base, "references"),
		filepath.Join(base, "assets"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating skill directory %s: %w", d, err)
		}
	}

	type skillFile struct {
		dest string
		tmpl string
	}
	tmplBase := "templates/skills/" + skill
	files := []skillFile{
		{filepath.Join(base, "SKILL.md"), tmplBase + "/SKILL.md.tmpl"},
		{filepath.Join(base, "references", "create-agent.md"), tmplBase + "/references/create-agent.md.tmpl"},
		{filepath.Join(base, "references", "create-skill.md"), tmplBase + "/references/create-skill.md.tmpl"},
		{filepath.Join(base, "references", "create-rule.md"), tmplBase + "/references/create-rule.md.tmpl"},
		// asset files are LLM templates; written with .tmpl extension preserved
		{filepath.Join(base, "assets", "AGENT.md.tmpl"), tmplBase + "/assets/AGENT.md.tmpl"},
		{filepath.Join(base, "assets", "SKILL.md.tmpl"), tmplBase + "/assets/SKILL.md.tmpl"},
		{filepath.Join(base, "assets", "rule.md.tmpl"), tmplBase + "/assets/rule.md.tmpl"},
	}
	for _, f := range files {
		if err := writeTemplate(f.dest, f.tmpl, nil); err != nil {
			return err
		}
	}
	return nil
}

// WriteCLAUDEMD writes a CLAUDE.md file to root for an existing oz workspace.
// Used by `oz add claude` to add Claude Code integration without re-scaffolding.
func WriteCLAUDEMD(root, name, description string) error {
	return createClaudeMD(root, templateData{Name: name, Description: description})
}

func createOZDir(root string) error {
	return os.MkdirAll(filepath.Join(root, ".oz"), 0755)
}

func createClaudeMD(root string, data templateData) error {
	return writeTemplate(filepath.Join(root, "CLAUDE.md"), "templates/CLAUDE.md.tmpl", data)
}

func createCodeDir(root, mode string, data templateData) error {
	if mode == "submodule" {
		// Leave code/ empty; the user will add submodules manually.
		return nil
	}
	return writeTemplate(
		filepath.Join(root, "code/README.md"),
		"templates/code/README.md.tmpl",
		data,
	)
}

// writeTemplate loads an embedded text/template by tmplPath (relative to
// internal/scaffold/templates/), renders it with data, and writes to destPath.
// If data is nil and the template has no actions, the file is written unchanged.
func writeTemplate(destPath, tmplPath string, data any) error {
	raw, err := scaffoldTemplates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("loading template %s: %w", tmplPath, err)
	}

	t, err := template.New(filepath.Base(tmplPath)).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("parsing template for %s: %w", filepath.Base(destPath), err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("rendering template for %s: %w", filepath.Base(destPath), err)
	}

	if err := os.WriteFile(destPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", destPath, err)
	}
	return nil
}
