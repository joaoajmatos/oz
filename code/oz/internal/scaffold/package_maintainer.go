package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func installMaintainerPackage(root string, force bool) ([]string, error) {
	var written []string

	agentPath := filepath.Join(root, "agents", "maintainer", "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(agentPath), 0755); err != nil {
		return written, fmt.Errorf("creating maintainer agent directory: %w", err)
	}
	if err := writeTemplateIfNeeded(agentPath, "templates/agent/AGENT.md.tmpl", AgentConfig{
		Name:        "maintainer",
		Description: "Keeps the workspace convention healthy and maintains agents, skills, and rules.",
		Type:        "maintainer",
	}, force); err != nil {
		return written, err
	}
	if _, err := os.Stat(agentPath); err == nil {
		written = append(written, "agents/maintainer/AGENT.md")
	}

	skillFiles := []struct {
		dest string
		tmpl string
	}{
		{"skills/workspace-management/SKILL.md", "templates/skills/workspace-management/SKILL.md.tmpl"},
		{"skills/workspace-management/references/create-agent.md", "templates/skills/workspace-management/references/create-agent.md.tmpl"},
		{"skills/workspace-management/references/create-skill.md", "templates/skills/workspace-management/references/create-skill.md.tmpl"},
		{"skills/workspace-management/references/create-rule.md", "templates/skills/workspace-management/references/create-rule.md.tmpl"},
		{"skills/workspace-management/assets/AGENT.md.tmpl", "templates/skills/workspace-management/assets/AGENT.md.tmpl"},
		{"skills/workspace-management/assets/SKILL.md.tmpl", "templates/skills/workspace-management/assets/SKILL.md.tmpl"},
		{"skills/workspace-management/assets/rule.md.tmpl", "templates/skills/workspace-management/assets/rule.md.tmpl"},
	}
	for _, f := range skillFiles {
		dest := filepath.Join(root, f.dest)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return written, fmt.Errorf("creating directory for %s: %w", f.dest, err)
		}
		if err := writeTemplateIfNeeded(dest, f.tmpl, nil, force); err != nil {
			return written, err
		}
		if _, err := os.Stat(dest); err == nil {
			written = append(written, filepath.ToSlash(f.dest))
		}
	}

	extra, err := mergeMaintainerManifests(root)
	if err != nil {
		return written, err
	}
	written = append(written, extra...)
	return written, nil
}

func writeTemplateIfNeeded(dest, tmpl string, data any, force bool) error {
	if _, err := os.Stat(dest); err == nil && !force {
		return nil
	}
	return writeTemplate(dest, tmpl, data)
}

func mergeMaintainerManifests(root string) ([]string, error) {
	var updated []string

	agentsPath := filepath.Join(root, "AGENTS.md")
	if b, err := os.ReadFile(agentsPath); err == nil {
		next, changed, err := mergeAgentsMaintainer(string(b))
		if err != nil {
			return updated, err
		}
		if changed {
			if err := os.WriteFile(agentsPath, []byte(next), 0644); err != nil {
				return updated, fmt.Errorf("writing AGENTS.md: %w", err)
			}
			updated = append(updated, "AGENTS.md (registered maintainer)")
		}
	}

	ozPath := filepath.Join(root, "OZ.md")
	if b, err := os.ReadFile(ozPath); err == nil {
		next, changed, err := mergeOZMaintainer(string(b))
		if err != nil {
			return updated, err
		}
		if changed {
			if err := os.WriteFile(ozPath, []byte(next), 0644); err != nil {
				return updated, fmt.Errorf("writing OZ.md: %w", err)
			}
			updated = append(updated, "OZ.md (registered maintainer)")
		}
	}
	return updated, nil
}

func mergeAgentsMaintainer(content string) (string, bool, error) {
	if strings.Contains(content, "| **maintainer** |") {
		return content, false, nil
	}
	row := "| **maintainer** | Creating or updating agents, skills, or rules (workspace-management), manifests (`AGENTS.md`, `OZ.md`), `oz validate` / `oz audit`, layout — not the main application code under `code/`. | `agents/maintainer/AGENT.md` |"
	next, ok := insertAgentMarkdownTableRow(content, "## Agents", row)
	return next, ok, nil
}

func mergeOZMaintainer(content string) (string, bool, error) {
	if strings.Contains(content, "| **maintainer** |") || strings.Contains(content, "- **maintainer**:") {
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
		row := "| **maintainer** | Creating or updating agents, skills, or rules (workspace-management), manifests (`AGENTS.md`, `OZ.md`), `oz validate` / `oz audit`, layout — not the main application code under `code/`. | `agents/maintainer/AGENT.md` |"
		next, ok := insertAgentMarkdownTableRow(content, "## Registered Agents", row)
		if ok {
			return next, true, nil
		}
		return content, false, nil
	}

	if strings.Contains(section, "- **") && strings.Contains(section, "`agents/") {
		row := "- **maintainer**: `agents/maintainer/AGENT.md`"
		return content[:sourceIdx] + row + "\n\n" + content[sourceIdx:], true, nil
	}

	return content, false, nil
}
