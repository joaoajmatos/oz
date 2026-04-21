package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/oz-tools/oz/internal/convention"
	"github.com/oz-tools/oz/internal/scaffold"
)

var (
	initClaudeFlag  bool
	initNoHooksFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Scaffold a new oz workspace",
	Long:  `Interactively scaffold a new oz-compliant workspace.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initClaudeFlag, "claude", false, "generate CLAUDE.md for Claude Code integration")
	initCmd.Flags().BoolVar(&initNoHooksFlag, "no-hooks", false, "skip IDE hook configuration for Claude Code and Cursor")
}

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	ozPurple = lipgloss.Color("#7C3AED")
	ozFaint  = lipgloss.Color("#6B7280")
	ozGreen  = lipgloss.Color("#10B981")
	ozLavend = lipgloss.Color("#A78BFA")

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ozPurple).
			Padding(0, 2)

	styleBrand = lipgloss.NewStyle().
			Bold(true).
			Foreground(ozPurple)

	styleSubtle = lipgloss.NewStyle().
			Foreground(ozFaint)

	styleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(ozGreen)

	styleCmd = lipgloss.NewStyle().
			Foreground(ozLavend).
			Bold(true)

	styleTreeRoot = lipgloss.NewStyle().
			Bold(true).
			Foreground(ozPurple)

	styleTreeDir = lipgloss.NewStyle().
			Foreground(ozLavend)

	styleTreeFile = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	styleSectionTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#F9FAFB"))
)

// ── Theme ─────────────────────────────────────────────────────────────────────

func ozTheme() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.Base = t.Focused.Base.BorderForeground(ozPurple)
	t.Focused.Title = t.Focused.Title.Foreground(ozPurple).Bold(true)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ozPurple)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(ozLavend)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Background(ozPurple).Foreground(lipgloss.Color("#FFFFFF"))
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(ozFaint)
	return t
}

// ── Command ───────────────────────────────────────────────────────────────────

func runInit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	printInitHeader()

	// ── Step 1: main form ────────────────────────────────────────────────────
	var (
		name        string
		description string
		codeMode    string
		useDefaults bool
	)

	mainForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Description("A short identifier for this workspace.").
				Placeholder("my-project").
				CharLimit(64).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("required")
					}
					return nil
				}).
				Value(&name),

			huh.NewInput().
				Title("Description").
				Description("One-line description of the project (optional).").
				Placeholder("An LLM-first development workspace").
				CharLimit(120).
				Value(&description),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Code directory mode").
				Description("How source code is organized in this workspace.").
				Options(
					huh.NewOption("inline     code lives in this repository", "inline"),
					huh.NewOption("submodule  code in a separate git submodule", "submodule"),
				).
				Value(&codeMode),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use default agents? (recommended)").
				Description("Creates a single 'maintainer' agent. Choose No to define custom agents.").
				Affirmative("Yes").
				Negative("No, I'll define them").
				Value(&useDefaults),
		),
	).WithTheme(ozTheme())

	if err := mainForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return err
	}

	// ── Step 2: agent configuration ──────────────────────────────────────────
	agents, err := collectAgents(useDefaults)
	if err != nil {
		return err
	}

	// ── Step 3: hooks confirmation ───────────────────────────────────────────
	setupHooks := !initNoHooksFlag
	if !initNoHooksFlag {
		hooksForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Configure IDE hooks?").
					Description("Writes Claude Code + Cursor hook configs that enforce oz convention on every edit and commit.").
					Affirmative("Yes").
					Negative("No").
					Value(&setupHooks),
			),
		).WithTheme(ozTheme())
		if err := hooksForm.Run(); err != nil && !errors.Is(err, huh.ErrUserAborted) {
			return err
		}
	}

	// ── Step 4: scaffold with spinner ────────────────────────────────────────
	cfg := scaffold.Config{
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		CodeMode:    codeMode,
		Agents:      agents,
		ClaudeMD:    initClaudeFlag,
		Hooks:       setupHooks,
	}

	var scaffoldErr error
	err = spinner.New().
		Title(" Scaffolding workspace…").
		Action(func() {
			scaffoldErr = scaffold.Scaffold(path, cfg)
		}).
		Run()

	if err != nil {
		return err
	}
	if scaffoldErr != nil {
		return fmt.Errorf("scaffold: %w", scaffoldErr)
	}

	// ── Step 4: success output ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println(styleSuccess.Render("  ✓ Workspace ready"))
	fmt.Println()
	printTree(path, initClaudeFlag, setupHooks, agents)
	fmt.Println()
	printNextSteps(setupHooks)

	return nil
}

// collectAgents gathers agent configs either from defaults or interactively.
func collectAgents(useDefaults bool) ([]scaffold.AgentConfig, error) {
	if useDefaults {
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

	fmt.Println()
	fmt.Println(styleSubtle.Render("  Define your agents — press Enter with an empty name when done."))
	fmt.Println()

	var agents []scaffold.AgentConfig
	for {
		var agentName string

		nameForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Agent name %s", styleSubtle.Render(fmt.Sprintf("(%d defined so far — leave empty to finish)", len(agents))))).
					Placeholder("e.g. coding, reviewer, devops").
					Value(&agentName),
			),
		).WithTheme(ozTheme())

		if err := nameForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				break
			}
			return nil, err
		}

		agentName = strings.TrimSpace(agentName)
		if agentName == "" {
			break
		}

		var agentDesc, agentType string
		addAnother := true

		detailForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Description").
					Description("What is this agent responsible for?").
					Placeholder("Builds and maintains source code").
					Value(&agentDesc),

				huh.NewSelect[string]().
					Title("Agent type").
					Options(
						huh.NewOption("generic  general-purpose agent", "generic"),
						huh.NewOption("coding   builds and modifies source code", "coding"),
					).
					Value(&agentType),

				huh.NewConfirm().
					Title("Add another agent?").
					Affirmative("Yes").
					Negative("No, done").
					Value(&addAnother),
			),
		).WithTheme(ozTheme())

		if err := detailForm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				break
			}
			return nil, err
		}

		t := agentType
		if t == "generic" {
			t = ""
		}
		agents = append(agents, scaffold.AgentConfig{
			Name:        agentName,
			Description: agentDesc,
			Type:        t,
		})

		if !addAnother {
			break
		}
	}

	// Fallback: if no agents defined, use defaults
	if len(agents) == 0 {
		for _, n := range convention.DefaultAgents {
			agents = append(agents, scaffold.AgentConfig{Name: n})
		}
	}

	return agents, nil
}

// ── Output helpers ────────────────────────────────────────────────────────────

func printInitHeader() {
	PrintBanner()
	subtitle := styleSubtle.Render("scaffold a new oz workspace")
	fmt.Println("  " + subtitle)
	fmt.Println()
}

type treeEntry struct {
	name string
	dir  bool
}

func printTree(root string, claudeMD bool, hooks bool, agents []scaffold.AgentConfig) {
	entries := []treeEntry{
		{".gitignore", false},
		{"AGENTS.md", false},
		{"OZ.md", false},
		{"README.md", false},
	}
	if claudeMD {
		entries = append([]treeEntry{{"CLAUDE.md", false}}, entries...)
	}

	// Show registered agents
	for _, a := range agents {
		entries = append(entries, treeEntry{fmt.Sprintf("agents/%s/AGENT.md", a.Name), false})
	}

	entries = append(entries,
		treeEntry{"specs/decisions/_template.md", false},
		treeEntry{"docs/", true},
		treeEntry{"context/", true},
		treeEntry{"rules/coding-guidelines.md", false},
		treeEntry{"skills/workspace-management/", true},
		treeEntry{"notes/", true},
		treeEntry{"code/", true},
		treeEntry{".oz/", true},
	)
	if hooks {
		entries = append(entries,
			treeEntry{".cursor/hooks.json", false},
			treeEntry{".cursor/hooks/", true},
			treeEntry{".claude/settings.json", false},
		)
	}

	fmt.Println("  " + styleTreeRoot.Render(root+"/"))
	for i, e := range entries {
		connector := "├── "
		if i == len(entries)-1 {
			connector = "└── "
		}
		if e.dir {
			fmt.Printf("  %s%s\n", connector, styleTreeDir.Render(e.name))
		} else {
			fmt.Printf("  %s%s\n", connector, styleTreeFile.Render(e.name))
		}
	}
}

func printNextSteps(hooks bool) {
	fmt.Println("  " + styleSectionTitle.Render("Next steps"))
	fmt.Println()

	steps := []struct{ cmd, desc string }{
		{"oz context build", "build the initial agent context snapshot"},
		{"oz audit drift  ", "check workspace convention compliance"},
		{"oz add claude   ", "add Claude Code integration (CLAUDE.md + hooks)"},
		{"oz add cursor   ", "add Cursor integration (hooks)"},
	}
	for _, s := range steps {
		fmt.Printf("  %s  %s\n", styleCmd.Render(s.cmd), styleSubtle.Render(s.desc))
	}
	if hooks {
		fmt.Println()
		fmt.Println("  " + styleSubtle.Render("IDE hooks configured — oz convention is enforced on every edit and commit."))
	}
	fmt.Println()
}
