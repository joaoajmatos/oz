package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	ozPurple = lipgloss.Color("#7C3AED")
	ozFaint  = lipgloss.Color("#6B7280")
	ozGreen  = lipgloss.Color("#10B981")
	ozLavend = lipgloss.Color("#A78BFA")
	ozSoft   = lipgloss.Color("#D1D5DB")

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
			Foreground(ozSoft)

	styleSectionTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ozSoft)
)

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

func installBrandedHelp(root *cobra.Command) {
	root.SetHelpFunc(func(c *cobra.Command, _ []string) {
		renderBrandedHelp(c.OutOrStdout(), c)
	})
}

func renderBrandedHelp(out io.Writer, c *cobra.Command) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s  %s\n", styleBrand.Render("oz"), styleSubtle.Render(c.CommandPath()))
	fmt.Fprintln(out)

	desc := strings.TrimSpace(c.Long)
	if desc == "" {
		desc = strings.TrimSpace(c.Short)
	}
	if desc != "" {
		fmt.Fprintln(out, styleSubtle.Render(indentLines(desc, "  ")))
		fmt.Fprintln(out)
	}

	printHelpSection(out, "Usage")
	fmt.Fprintf(out, "  %s\n", styleCmd.Render(c.UseLine()))

	subcommands := availableSubcommands(c)
	if len(subcommands) > 0 {
		fmt.Fprintln(out)
		printHelpSection(out, "Commands")
		for _, sub := range subcommands {
			fmt.Fprintf(out, "  %-24s %s\n", styleCmd.Render(sub.Name()), sub.Short)
		}
	}

	localUsages := strings.TrimSpace(c.LocalFlags().FlagUsages())
	if localUsages != "" {
		fmt.Fprintln(out)
		printHelpSection(out, "Flags")
		fmt.Fprintln(out, indentLines(localUsages, "  "))
	}

	inheritedUsages := strings.TrimSpace(c.InheritedFlags().FlagUsages())
	if inheritedUsages != "" {
		fmt.Fprintln(out)
		printHelpSection(out, "Global Flags")
		fmt.Fprintln(out, indentLines(inheritedUsages, "  "))
	}

	if c.HasHelpSubCommands() {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s %s\n", styleSubtle.Render("Use"), styleCmd.Render(c.CommandPath()+" [command] --help"))
	}
	fmt.Fprintln(out)
}

func printHelpSection(out io.Writer, title string) {
	fmt.Fprintf(out, "  %s\n", styleSectionTitle.Render(title))
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			lines[i] = ""
			continue
		}
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func availableSubcommands(c *cobra.Command) []*cobra.Command {
	var subs []*cobra.Command
	for _, sub := range c.Commands() {
		if !sub.IsAvailableCommand() || sub.Hidden {
			continue
		}
		if sub.Name() == "help" {
			continue
		}
		subs = append(subs, sub)
	}
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Name() < subs[j].Name()
	})
	return subs
}
