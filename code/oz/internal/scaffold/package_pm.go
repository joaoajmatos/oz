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
	if strings.Contains(content, "| **pm** |") {
		return content, false, nil
	}
	row := "| **pm** | PRDs, pre-mortems, backlog stories, sprint plan/retro/release notes, or other PM workflows using `skills/pm/` | `agents/pm/AGENT.md` |"
	next, ok := insertAgentMarkdownTableRow(content, "## Agents", row)
	return next, ok, nil
}

// insertAgentMarkdownTableRow appends row after the last markdown table body line that
// starts with "| **" and contains "`agents/`", in the section from sectionHeading up to
// "## Source of Truth Hierarchy". The section must include an | Agent | header table.
func insertAgentMarkdownTableRow(content, sectionHeading, row string) (string, bool) {
	if !strings.Contains(content, sectionHeading) || !strings.Contains(content, "## Source of Truth Hierarchy") {
		return content, false
	}
	agentsStart := strings.Index(content, sectionHeading)
	sourceRel := strings.Index(content[agentsStart:], "## Source of Truth Hierarchy")
	if sourceRel < 0 {
		return content, false
	}
	sourceIdx := agentsStart + sourceRel
	section := content[agentsStart:sourceIdx]
	if !strings.Contains(section, "| Agent |") || !strings.Contains(section, "|---|") {
		return content, false
	}
	lines := strings.Split(content[:sourceIdx], "\n")
	insertAfter := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "| **") && strings.Contains(line, "`agents/") {
			insertAfter = i
		}
	}
	if insertAfter < 0 {
		return content, false
	}
	lines = append(lines[:insertAfter+1], append([]string{row}, lines[insertAfter+1:]...)...)
	return strings.Join(lines, "\n") + content[sourceIdx:], true
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
		row := "| **pm** | PRDs, pre-mortems, backlog stories, sprint plan/retro/release notes, or other PM workflows using `skills/pm/` | `agents/pm/AGENT.md` |"
		next, ok := insertAgentMarkdownTableRow(content, "## Registered Agents", row)
		if ok {
			return next, true, nil
		}
		return content, false, nil
	}

	if strings.Contains(section, "- **") && strings.Contains(section, "`agents/") {
		row := "- **pm**: `agents/pm/AGENT.md`"
		return content[:sourceIdx] + row + "\n\n" + content[sourceIdx:], true, nil
	}

	return content, false, nil
}
