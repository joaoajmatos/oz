package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/joaoajmatos/oz/internal/convention"
)

// Config holds the parameters collected during oz init.
type Config struct {
	Name        string
	Description string
	CodeMode    string // "inline" or "submodule"
	Agents      []AgentConfig
	ClaudeMD    bool // generate CLAUDE.md for Claude Code native integration
	Hooks       bool // generate IDE hook configurations for Claude Code and Cursor
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
	CodeMode    string // "inline" or "submodule"
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
		CodeMode:    cfg.CodeMode,
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
	if cfg.Hooks {
		steps = append(steps, func() error { return createHooksFiles(abs) })
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
	skill := "workspace-management"
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

// WriteClaudeHooks writes Claude Code hook configuration to an existing oz workspace:
//   - .claude/settings.json  (merged if already present)
//   - .oz/hooks/oz-*.sh      (shared + provider-specific hook scripts, executable)
func WriteClaudeHooks(root string) error {
	if err := writeHookScripts(root); err != nil {
		return err
	}
	return mergeClaudeSettings(root)
}

// WriteCursorHooks writes Cursor hook configuration to an existing oz workspace:
//   - .cursor/hooks.json
//   - .oz/hooks/oz-*.sh      (shared + provider-specific hook scripts, executable)
func WriteCursorHooks(root string) error {
	if err := writeHookScripts(root); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".cursor"), 0755); err != nil {
		return fmt.Errorf("creating .cursor: %w", err)
	}
	return writeTemplate(filepath.Join(root, ".cursor/hooks.json"), "templates/hooks/cursor-hooks.json.tmpl", nil)
}

// WriteCursorSkills writes the oz and oz-shell skills required by Cursor integration:
//   - skills/oz/SKILL.md
//   - skills/oz/references/audit-and-validate.md
//   - skills/oz/references/context-and-mcp.md
//   - skills/oz-shell/SKILL.md
func WriteCursorSkills(root string) error {
	type skillFile struct {
		dest string
		tmpl string
	}
	files := []skillFile{
		{"skills/oz/SKILL.md", "templates/skills/oz/SKILL.md.tmpl"},
		{"skills/oz/references/audit-and-validate.md", "templates/skills/oz/references/audit-and-validate.md.tmpl"},
		{"skills/oz/references/context-and-mcp.md", "templates/skills/oz/references/context-and-mcp.md.tmpl"},
		{"skills/oz-shell/SKILL.md", "templates/skills/oz-shell/SKILL.md.tmpl"},
	}
	for _, f := range files {
		dest := filepath.Join(root, f.dest)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(f.dest), err)
		}
		if err := writeTemplate(dest, f.tmpl, nil); err != nil {
			return err
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving user home directory: %w", err)
	}
	cursorSkillsRoot := filepath.Join(home, ".cursor", "skills-cursor")
	for _, f := range files {
		// Cursor discovers global skills under ~/.cursor/skills-cursor/<skill-name>/...
		parts := strings.SplitN(f.dest, string(filepath.Separator), 3)
		if len(parts) < 3 {
			return fmt.Errorf("invalid skill destination path %q", f.dest)
		}
		dest := filepath.Join(cursorSkillsRoot, parts[1], parts[2])
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
		}
		if err := writeTemplate(dest, f.tmpl, nil); err != nil {
			return err
		}
	}
	return nil
}

// writeHookScripts writes the shared hook scripts to .oz/hooks/ with 0755.
func writeHookScripts(root string) error {
	hooksDir := filepath.Join(root, ".oz", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating .oz/hooks: %w", err)
	}
	scripts := []struct{ dest, tmpl string }{
		{".oz/hooks/oz-session-init.sh", "templates/hooks/oz-session-init.sh.tmpl"},
		{".oz/hooks/oz-after-edit.sh", "templates/hooks/oz-after-edit.sh.tmpl"},
		{".oz/hooks/oz-pre-commit.sh", "templates/hooks/oz-pre-commit.sh.tmpl"},
		{".oz/hooks/oz-shell-rewrite-cursor.sh", "templates/hooks/oz-shell-rewrite-cursor.sh.tmpl"},
		{".oz/hooks/oz-read-rewrite-cursor.sh", "templates/hooks/oz-read-rewrite-cursor.sh.tmpl"},
		{".oz/hooks/oz-read-policy-cursor.sh", "templates/hooks/oz-read-policy-cursor.sh.tmpl"},
		{".oz/hooks/oz-shell-rewrite-claude.sh", "templates/hooks/oz-shell-rewrite-claude.sh.tmpl"},
		{".oz/hooks/oz-read-policy-claude.sh", "templates/hooks/oz-read-policy-claude.sh.tmpl"},
		{".oz/hooks/oz-shell-rewrite.sh", "templates/hooks/oz-shell-rewrite.sh.tmpl"},
	}
	for _, s := range scripts {
		if err := writeTemplateMode(filepath.Join(root, s.dest), s.tmpl, nil, 0755); err != nil {
			return err
		}
	}
	return nil
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

// createHooksFiles writes IDE hook configurations for Cursor and Claude Code:
//   - .oz/hooks/oz-session-init.sh  (executable)
//   - .oz/hooks/oz-after-edit.sh    (executable)
//   - .oz/hooks/oz-pre-commit.sh    (executable)
//   - .oz/hooks/oz-shell-rewrite-cursor.sh (executable)
//   - .oz/hooks/oz-read-rewrite-cursor.sh  (executable)
//   - .oz/hooks/oz-read-policy-cursor.sh   (executable)
//   - .oz/hooks/oz-shell-rewrite-claude.sh (executable)
//   - .oz/hooks/oz-read-policy-claude.sh   (executable)
//   - .oz/hooks/oz-shell-rewrite.sh        (compatibility shim, executable)
//   - .cursor/hooks.json
//   - .claude/settings.json         (merged if file already exists)
func createHooksFiles(root string) error {
	if err := writeHookScripts(root); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, ".cursor"), 0755); err != nil {
		return fmt.Errorf("creating .cursor: %w", err)
	}
	if err := writeTemplate(filepath.Join(root, ".cursor/hooks.json"), "templates/hooks/cursor-hooks.json.tmpl", nil); err != nil {
		return err
	}
	return mergeClaudeSettings(root)
}

// mergeClaudeSettings writes the oz hook stanza into .claude/settings.json,
// preserving any existing keys (e.g. permissions).
func mergeClaudeSettings(root string) error {
	settingsPath := filepath.Join(root, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("creating .claude: %w", err)
	}

	// Load oz hook stanza from embedded template.
	raw, err := scaffoldTemplates.ReadFile("templates/hooks/claude-settings.json.tmpl")
	if err != nil {
		return fmt.Errorf("loading claude-settings template: %w", err)
	}

	var ozStanza map[string]json.RawMessage
	if err := json.Unmarshal(raw, &ozStanza); err != nil {
		return fmt.Errorf("parsing claude-settings template: %w", err)
	}

	// Read existing settings.json if present.
	existing := make(map[string]json.RawMessage)
	if data, readErr := os.ReadFile(settingsPath); readErr == nil {
		if err := json.Unmarshal(data, &existing); err != nil {
			// Unreadable existing file — don't overwrite; return error.
			return fmt.Errorf("parsing existing .claude/settings.json: %w", err)
		}
	}

	// Merge: oz stanza keys overwrite existing keys of the same name.
	for k, v := range ozStanza {
		existing[k] = v
	}

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling .claude/settings.json: %w", err)
	}
	if err := os.WriteFile(settingsPath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("writing .claude/settings.json: %w", err)
	}
	return nil
}

// writeTemplateMode is like writeTemplate but sets the file mode explicitly.
func writeTemplateMode(destPath, tmplPath string, data any, mode os.FileMode) error {
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

	if err := os.WriteFile(destPath, buf.Bytes(), mode); err != nil {
		return fmt.Errorf("writing %s: %w", destPath, err)
	}
	return nil
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
