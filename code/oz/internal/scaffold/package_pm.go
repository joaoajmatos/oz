package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// pmPackageFiles lists embedded templates and their destination paths under root.
var pmPackageFiles = []struct {
	dest string // relative to workspace root
	tmpl string // path under internal/scaffold/templates/
}{
	{"agents/pm/AGENT.md", "templates/packages/pm/agents/pm/AGENT.md.tmpl"},
	{"skills/pm/SKILL.md", "templates/packages/pm/skills/pm/SKILL.md.tmpl"},
	{"skills/pm/README.md", "templates/packages/pm/skills/pm/README.md.tmpl"},
	{"skills/pm/references/create-prd.md", "templates/packages/pm/skills/pm/references/create-prd.md.tmpl"},
	{"skills/pm/references/pre-mortem.md", "templates/packages/pm/skills/pm/references/pre-mortem.md.tmpl"},
	{"skills/pm/references/write-stories-user.md", "templates/packages/pm/skills/pm/references/write-stories-user.md.tmpl"},
	{"skills/pm/references/write-stories-job.md", "templates/packages/pm/skills/pm/references/write-stories-job.md.tmpl"},
	{"skills/pm/references/write-stories-wwa.md", "templates/packages/pm/skills/pm/references/write-stories-wwa.md.tmpl"},
	{"skills/pm/references/sprint-plan.md", "templates/packages/pm/skills/pm/references/sprint-plan.md.tmpl"},
	{"skills/pm/references/sprint-retro.md", "templates/packages/pm/skills/pm/references/sprint-retro.md.tmpl"},
	{"skills/pm/references/sprint-release-notes.md", "templates/packages/pm/skills/pm/references/sprint-release-notes.md.tmpl"},
}

func installPMPackage(root string, force bool) ([]string, error) {
	var written []string
	for _, f := range pmPackageFiles {
		dest := filepath.Join(root, f.dest)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return written, fmt.Errorf("creating directory for %s: %w", f.dest, err)
		}
		if _, err := os.Stat(dest); err == nil && !force {
			return written, fmt.Errorf("refusing to overwrite existing file %s — re-run with --force to replace package files", f.dest)
		}
		if err := writeTemplate(dest, f.tmpl, nil); err != nil {
			return written, err
		}
		written = append(written, filepath.ToSlash(f.dest))
	}

	extra, err := mergePMManifests(root)
	if err != nil {
		return written, err
	}
	written = append(written, extra...)
	return written, nil
}

// mergePMManifests registers the pm agent in AGENTS.md and OZ.md when those files
// use supported layouts and pm is not already listed.
func mergePMManifests(root string) ([]string, error) {
	var updated []string
	agentsPath := filepath.Join(root, "AGENTS.md")
	if b, err := os.ReadFile(agentsPath); err == nil {
		next, changed, err := mergeAgentsPM(string(b))
		if err != nil {
			return updated, err
		}
		if changed {
			if err := os.WriteFile(agentsPath, []byte(next), 0644); err != nil {
				return updated, fmt.Errorf("writing AGENTS.md: %w", err)
			}
			updated = append(updated, "AGENTS.md (registered pm)")
		}
	}

	ozPath := filepath.Join(root, "OZ.md")
	if b, err := os.ReadFile(ozPath); err == nil {
		next, changed, err := mergeOZPM(string(b))
		if err != nil {
			return updated, err
		}
		if changed {
			if err := os.WriteFile(ozPath, []byte(next), 0644); err != nil {
				return updated, fmt.Errorf("writing OZ.md: %w", err)
			}
			updated = append(updated, "OZ.md (registered pm)")
		}
	}
	return updated, nil
}

func mergeAgentsPM(content string) (string, bool, error) {
	if strings.Contains(content, "agents/pm/AGENT.md") {
		return content, false, nil
	}
	if !strings.Contains(content, "## Source of Truth Hierarchy") {
		return content, false, nil
	}
	block := "### pm\n\n" +
		"**Runs product management workflows in-repo (PRDs, risk reviews, backlog items, sprint rituals).**\n\n" +
		"Agent definition: `agents/pm/AGENT.md`\n"

	marker := "\n---\n\n## Source of Truth Hierarchy"
	if pos := strings.Index(content, marker); pos >= 0 {
		return content[:pos] + "\n" + block + content[pos:], true, nil
	}
	idx := strings.Index(content, "## Source of Truth Hierarchy")
	if idx < 0 {
		return content, false, nil
	}
	return content[:idx] + block + "\n" + content[idx:], true, nil
}

func mergeOZPM(content string) (string, bool, error) {
	if strings.Contains(content, "| **pm** |") || strings.Contains(content, "- **pm**:") {
		return content, false, nil
	}
	if !strings.Contains(content, "## Registered Agents") {
		return content, false, nil
	}
	if !strings.Contains(content, "## Source of Truth Hierarchy") {
		return content, false, nil
	}

	agentsStart := strings.Index(content, "## Registered Agents")
	sourceIdx := strings.Index(content[agentsStart:], "## Source of Truth Hierarchy")
	if sourceIdx < 0 {
		return content, false, nil
	}
	sourceIdx += agentsStart
	section := content[agentsStart:sourceIdx]

	if strings.Contains(section, "| Agent |") && strings.Contains(section, "|---|") {
		lines := strings.Split(content[:sourceIdx], "\n")
		insertAfter := -1
		for i, line := range lines {
			if strings.HasPrefix(line, "| **") && strings.Contains(line, "`agents/") {
				insertAfter = i
			}
		}
		if insertAfter < 0 {
			return content, false, nil
		}
		row := "| **pm** | Runs product management workflows in-repo | `agents/pm/AGENT.md` |"
		lines = append(lines[:insertAfter+1], append([]string{row}, lines[insertAfter+1:]...)...)
		return strings.Join(lines, "\n") + content[sourceIdx:], true, nil
	}

	if strings.Contains(section, "- **") && strings.Contains(section, "`agents/") {
		row := "- **pm**: `agents/pm/AGENT.md`"
		return content[:sourceIdx] + row + "\n\n" + content[sourceIdx:], true, nil
	}

	return content, false, nil
}
