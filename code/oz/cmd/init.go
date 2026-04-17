package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oz-tools/oz/internal/convention"
	"github.com/oz-tools/oz/internal/scaffold"
)

var initClaudeFlag bool

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Scaffold a new oz workspace",
	Long: `Interactively scaffold a new oz-compliant workspace.

Asks for a project name, description, code directory mode, and which agents
to register. Then generates the full directory structure and all required files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initClaudeFlag, "claude", false, "generate CLAUDE.md for Claude Code native integration")
}

func runInit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	r := bufio.NewReader(os.Stdin)

	fmt.Println("oz init — scaffolding a new oz workspace")
	fmt.Println()

	name, err := prompt(r, "Project name", "")
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("project name is required")
	}

	description, err := prompt(r, "Description", "")
	if err != nil {
		return err
	}

	codeMode, err := promptChoice(r,
		"Code directory mode",
		[]string{"inline", "submodule"},
		"inline",
	)
	if err != nil {
		return err
	}

	agents, err := promptAgents(r)
	if err != nil {
		return err
	}

	cfg := scaffold.Config{
		Name:        name,
		Description: description,
		CodeMode:    codeMode,
		Agents:      agents,
		ClaudeMD:    initClaudeFlag,
	}

	fmt.Println()
	fmt.Printf("Scaffolding oz workspace in %s ...\n", path)

	if err := scaffold.Scaffold(path, cfg); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	fmt.Println()
	fmt.Println("Done. Workspace structure:")
	printTree(path, initClaudeFlag)

	return nil
}

// promptAgents asks whether to use the default agent set or define custom ones.
// If the user declines defaults, it loops prompting for name + description
// until an empty name is entered.
func promptAgents(r *bufio.Reader) ([]scaffold.AgentConfig, error) {
	fmt.Printf("Use default agents? [Y/n]: ")
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)

	if line == "" || strings.EqualFold(line, "y") {
		agents := make([]scaffold.AgentConfig, 0, len(convention.DefaultAgents))
		for _, n := range convention.DefaultAgents {
			t := ""
			if n == "coding" {
				t = "coding"
			}
			agents = append(agents, scaffold.AgentConfig{Name: n, Type: t})
		}
		return agents, nil
	}

	var agents []scaffold.AgentConfig
	for {
		name, err := prompt(r, "Agent name", "")
		if err != nil {
			return nil, err
		}
		if name == "" {
			break
		}
		description, err := prompt(r, "Description", "")
		if err != nil {
			return nil, err
		}
		agentType, err := promptChoice(r, "Agent type", []string{"coding", "generic"}, "generic")
		if err != nil {
			return nil, err
		}
		if agentType == "generic" {
			agentType = ""
		}
		agents = append(agents, scaffold.AgentConfig{Name: name, Description: description, Type: agentType})
	}
	return agents, nil
}

// prompt prints a prompt with an optional default and returns the trimmed input.
func prompt(r *bufio.Reader, label, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return defaultVal, nil
	}
	return v, nil
}

// promptChoice prints a multiple-choice prompt and returns the chosen option.
func promptChoice(r *bufio.Reader, label string, choices []string, defaultVal string) (string, error) {
	fmt.Printf("%s [%s] (%s): ", label, defaultVal, strings.Join(choices, "/"))
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return defaultVal, nil
	}
	for _, c := range choices {
		if strings.EqualFold(v, c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("invalid choice %q — expected one of: %s", v, strings.Join(choices, ", "))
}

// printTree prints a minimal view of the scaffolded workspace.
func printTree(root string, claudeMD bool) {
	entries := []string{
		"AGENTS.md",
		"OZ.md",
		"README.md",
		"agents/<name>/AGENT.md",
		"specs/decisions/_template.md",
		"docs/architecture.md",
		"docs/open-items.md",
		"context/",
		"rules/coding-guidelines.md",
		"notes/",
		"code/",
		"skills/",
		"tools/",
		"scripts/",
		".oz/",
	}
	if claudeMD {
		entries = append([]string{"CLAUDE.md"}, entries...)
	}
	fmt.Printf("%s/\n", root)
	for i, e := range entries {
		prefix := "├── "
		if i == len(entries)-1 {
			prefix = "└── "
		}
		fmt.Printf("  %s%s\n", prefix, e)
	}
}
