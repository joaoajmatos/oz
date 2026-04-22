package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/joaoajmatos/oz/internal/scaffold"
)

var addListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List integrations and optional packages you can add",
	Args:    cobra.NoArgs,
	RunE:    runAddList,
}

type integrationRow struct {
	id      string
	summary string
	example string
	flags   string
}

var integrationCatalog = []integrationRow{
	{
		id:      "claude",
		summary: "Claude Code — CLAUDE.md, .claude/settings.json, shared hook scripts",
		example: "oz add claude [path]",
		flags:   "--force  overwrite CLAUDE.md if it already exists",
	},
	{
		id:      "cursor",
		summary: "Cursor — hooks.json + shared hook scripts under .cursor/hooks/",
		example: "oz add cursor [path]",
		flags:   "",
	},
}

func runAddList(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	rule := lipgloss.NewStyle().
		Foreground(ozFaint).
		Render(strings.Repeat("─", 56))

	fmt.Fprintln(out)
	printHelpSection(out, "Integrations")
	fmt.Fprintln(out, styleSubtle.Render("Editor and IDE wiring (separate from optional packages)"))
	fmt.Fprintln(out, rule)

	for _, row := range integrationCatalog {
		fmt.Fprintf(out, "  %s  %s\n", styleCmd.Render(row.id), row.summary)
		fmt.Fprintf(out, "      %s\n", styleSubtle.Render(row.example))
		if row.flags != "" {
			fmt.Fprintf(out, "      %s\n", styleSubtle.Render(row.flags))
		}
	}

	fmt.Fprintln(out)
	printHelpSection(out, "Optional packages")
	fmt.Fprintln(out, styleSubtle.Render("Shipped inside the oz binary — agent + skill trees"))
	fmt.Fprintln(out, rule)

	for _, p := range scaffold.PackageCatalog() {
		fmt.Fprintf(out, "  %s  %s\n", styleCmd.Render(p.ID), p.Summary)
		fmt.Fprintf(out, "      %s\n", styleSubtle.Render("oz add "+p.ID+" [path]"))
		if p.SupportsForce {
			fmt.Fprintf(out, "      %s\n", styleSubtle.Render("--force  replace existing package files"))
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, styleSubtle.Render("Tip: run "+styleCmd.Render("oz add list")+" any time. With no [path], the workspace root is inferred from AGENTS.md + OZ.md."))
	fmt.Fprintln(out)
	return nil
}
