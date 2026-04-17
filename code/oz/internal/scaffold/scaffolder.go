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

	if err := createDirectories(abs); err != nil {
		return err
	}
	if err := createRootFiles(abs, data); err != nil {
		return err
	}
	if err := createAgentFiles(abs, cfg.Agents); err != nil {
		return err
	}
	if err := createDocFiles(abs, data); err != nil {
		return err
	}
	if err := createSpecFiles(abs); err != nil {
		return err
	}
	if err := createRulesFiles(abs); err != nil {
		return err
	}
	if err := createCodeDir(abs, cfg.CodeMode, data); err != nil {
		return err
	}
	if err := createOZDir(abs); err != nil {
		return err
	}
	if cfg.ClaudeMD {
		if err := createClaudeMD(abs, data); err != nil {
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
		name string
		tmpl string
	}{
		{".gitignore", rootGitignoreTmpl},
		{"OZ.md", ozMDTmpl},
		{"AGENTS.md", agentsMDTmpl},
		{"README.md", readmeTmpl},
	}
	for _, f := range files {
		if err := writeTemplate(filepath.Join(root, f.name), f.tmpl, data); err != nil {
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
		if err := writeTemplate(filepath.Join(dir, "AGENT.md"), agentMDTmpl, agent); err != nil {
			return err
		}
	}
	return nil
}

func createDocFiles(root string, data templateData) error {
	files := []struct {
		name string
		tmpl string
	}{
		{"docs/architecture.md", architectureTmpl},
		{"docs/open-items.md", openItemsTmpl},
	}
	for _, f := range files {
		if err := writeTemplate(filepath.Join(root, f.name), f.tmpl, data); err != nil {
			return err
		}
	}
	return nil
}

func createSpecFiles(root string) error {
	return writeTemplate(
		filepath.Join(root, "specs/decisions/_template.md"),
		decisionTemplateTmpl,
		nil,
	)
}

func createRulesFiles(root string) error {
	return writeTemplate(
		filepath.Join(root, "rules/coding-guidelines.md"),
		codingGuidelinesTmpl,
		nil,
	)
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
	return writeTemplate(filepath.Join(root, "CLAUDE.md"), claudeMDTmpl, data)
}

func createCodeDir(root, mode string, data templateData) error {
	if mode == "submodule" {
		// Leave code/ empty; the user will add submodules manually.
		return nil
	}
	return writeTemplate(
		filepath.Join(root, "code/README.md"),
		codeREADMETmpl,
		data,
	)
}

// writeTemplate renders tmpl with data and writes the result to path.
// If data is nil the template is written as-is (no substitutions).
func writeTemplate(path, tmplStr string, data any) error {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parsing template for %s: %w", filepath.Base(path), err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("rendering template for %s: %w", filepath.Base(path), err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
