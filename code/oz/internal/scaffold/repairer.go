package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oz-tools/oz/internal/convention"
)

// RepairResult reports which default files were restored vs already present.
// Paths are workspace-relative and use forward slashes.
type RepairResult struct {
	Created []string
	Skipped []string
}

// writeIfMissing renders tmplPath to destPath only if destPath does not exist.
// Returns true if the file was created.
func writeIfMissing(destPath, tmplPath string, data any) (bool, error) {
	if _, err := os.Stat(destPath); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat %s: %w", destPath, err)
	}
	if err := writeTemplate(destPath, tmplPath, data); err != nil {
		return false, err
	}
	return true, nil
}

func relPath(root, dest string) (string, error) {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func ensureCreateWorkspaceArtifactDirs(root string) error {
	skill := "create-workspace-artifact"
	base := filepath.Join(root, "skills", skill)
	for _, d := range []string{
		base,
		filepath.Join(base, "references"),
		filepath.Join(base, "assets"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating skill directory %s: %w", d, err)
		}
	}
	return nil
}

// Repair restores missing default workspace files that oz init would generate.
// Existing files are never overwritten. Directories are created as needed but
// are not listed in RepairResult.
func Repair(root string, cfg Config) (RepairResult, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return RepairResult{}, fmt.Errorf("resolving path: %w", err)
	}

	data := templateData{
		Name:        cfg.Name,
		Description: cfg.Description,
		OZVersion:   convention.Version,
		Agents:      cfg.Agents,
	}

	var result RepairResult

	record := func(destAbs, tmpl string, data any) error {
		created, err := writeIfMissing(destAbs, tmpl, data)
		if err != nil {
			return err
		}
		rel, err := relPath(abs, destAbs)
		if err != nil {
			return err
		}
		if created {
			result.Created = append(result.Created, rel)
		} else {
			result.Skipped = append(result.Skipped, rel)
		}
		return nil
	}

	if err := createDirectories(abs); err != nil {
		return RepairResult{}, err
	}
	if err := ensureCreateWorkspaceArtifactDirs(abs); err != nil {
		return RepairResult{}, err
	}

	rootFiles := []struct {
		dest string
		tmpl string
	}{
		{".gitignore", "templates/root.gitignore.tmpl"},
		{"OZ.md", "templates/OZ.md.tmpl"},
		{"AGENTS.md", "templates/AGENTS.md.tmpl"},
		{"README.md", "templates/README.md.tmpl"},
	}
	for _, f := range rootFiles {
		if err := record(filepath.Join(abs, f.dest), f.tmpl, data); err != nil {
			return RepairResult{}, err
		}
	}

	for _, agent := range cfg.Agents {
		dir := filepath.Join(abs, "agents", agent.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return RepairResult{}, fmt.Errorf("creating agent dir %s: %w", agent.Name, err)
		}
		dest := filepath.Join(dir, "AGENT.md")
		if err := record(dest, "templates/agent/AGENT.md.tmpl", agent); err != nil {
			return RepairResult{}, err
		}
	}

	docFiles := []struct {
		dest string
		tmpl string
	}{
		{"docs/architecture.md", "templates/docs/architecture.md.tmpl"},
		{"docs/open-items.md", "templates/docs/open-items.md.tmpl"},
	}
	for _, f := range docFiles {
		if err := record(filepath.Join(abs, f.dest), f.tmpl, data); err != nil {
			return RepairResult{}, err
		}
	}

	if err := record(
		filepath.Join(abs, "specs/decisions/_template.md"),
		"templates/specs/decisions/_template.md.tmpl",
		nil,
	); err != nil {
		return RepairResult{}, err
	}

	if err := record(
		filepath.Join(abs, "rules/coding-guidelines.md"),
		"templates/rules/coding-guidelines.md.tmpl",
		nil,
	); err != nil {
		return RepairResult{}, err
	}

	skill := "create-workspace-artifact"
	tmplBase := "templates/skills/" + skill
	base := filepath.Join(abs, "skills", skill)
	skillFiles := []struct {
		dest string
		tmpl string
	}{
		{filepath.Join(base, "SKILL.md"), tmplBase + "/SKILL.md.tmpl"},
		{filepath.Join(base, "references", "create-agent.md"), tmplBase + "/references/create-agent.md.tmpl"},
		{filepath.Join(base, "references", "create-skill.md"), tmplBase + "/references/create-skill.md.tmpl"},
		{filepath.Join(base, "references", "create-rule.md"), tmplBase + "/references/create-rule.md.tmpl"},
		{filepath.Join(base, "assets", "AGENT.md.tmpl"), tmplBase + "/assets/AGENT.md.tmpl"},
		{filepath.Join(base, "assets", "SKILL.md.tmpl"), tmplBase + "/assets/SKILL.md.tmpl"},
		{filepath.Join(base, "assets", "rule.md.tmpl"), tmplBase + "/assets/rule.md.tmpl"},
	}
	for _, f := range skillFiles {
		if err := record(f.dest, f.tmpl, nil); err != nil {
			return RepairResult{}, err
		}
	}

	if cfg.CodeMode != "submodule" {
		if err := record(
			filepath.Join(abs, "code/README.md"),
			"templates/code/README.md.tmpl",
			data,
		); err != nil {
			return RepairResult{}, err
		}
	}

	if err := createOZDir(abs); err != nil {
		return RepairResult{}, err
	}

	if cfg.ClaudeMD {
		if err := record(filepath.Join(abs, "CLAUDE.md"), "templates/CLAUDE.md.tmpl", data); err != nil {
			return RepairResult{}, err
		}
	}

	return result, nil
}
