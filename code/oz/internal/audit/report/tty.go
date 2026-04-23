package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/joaoajmatos/oz/internal/audit"
)

// Oz-themed palette; matches cmd/ui.go so audit output looks like the rest of the CLI.
var (
	ozFaint  = lipgloss.Color("#6B7280")
	ozGreen  = lipgloss.Color("#10B981")
	ozLavend = lipgloss.Color("#A78BFA")
	ozOrange = lipgloss.Color("#F59E0B")
	ozPurple = lipgloss.Color("#7C3AED")
	ozRed = lipgloss.Color("#EF4444")

	auditStyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ozPurple)
	auditStyleSubtle = lipgloss.NewStyle().Foreground(ozFaint)
	auditStyleCode   = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	auditStylePath = lipgloss.NewStyle().Foreground(ozFaint)
	auditStyleOK   = lipgloss.NewStyle().Bold(true).Foreground(ozGreen)
	auditStyleRule = lipgloss.NewStyle().Foreground(ozFaint)
	auditStyleSectionE = lipgloss.NewStyle().Bold(true).Foreground(ozRed)
	auditStyleSectionW = lipgloss.NewStyle().Bold(true).Foreground(ozOrange)
	auditStyleSectionI = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	auditStyleHint     = lipgloss.NewStyle().Foreground(ozFaint)
)

// WriteHumanStyled writes a TTY-oriented, colorized report using lipgloss.
// It mirrors WriteHuman: same fields, minSeverity filter, and count footer.
// Call only when stdout is a color-capable TTY; for automation or LLM tooling,
// use WriteJSON or WriteHuman (plain text) instead.
func WriteHumanStyled(w io.Writer, r *audit.Report, minSeverity audit.Severity) {
	minRank := severityRank(minSeverity)
	printed := 0

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s\n", auditStyleTitle.Render("oz audit"), auditStyleSubtle.Render("workspace health"))

	countBox := buildCountLine(r)
	fmt.Fprintf(w, "  %s\n", countBox)
	fmt.Fprintln(w, "  "+auditStyleRule.Render(strings.Repeat("─", 58)))

	labels := map[audit.Severity]struct {
		text  string
		style lipgloss.Style
	}{
		audit.SeverityError: {"Errors", auditStyleSectionE},
		audit.SeverityWarn:  {"Warnings", auditStyleSectionW},
		audit.SeverityInfo:  {"Info", auditStyleSectionI},
	}

	for _, sev := range []audit.Severity{audit.SeverityError, audit.SeverityWarn, audit.SeverityInfo} {
		if severityRank(sev) > minRank {
			continue
		}
		var group []audit.Finding
		for _, f := range r.Findings {
			if f.Severity == sev {
				group = append(group, f)
			}
		}
		if len(group) == 0 {
			continue
		}
		meta := labels[sev]
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", meta.style.Render(meta.text))
		for _, f := range group {
			loc := f.File
			if f.Line > 0 && loc != "" {
				loc = fmt.Sprintf("%s:%d", loc, f.Line)
			}
			if loc != "" {
				fmt.Fprintf(w, "    %s  %s  %s  %s\n",
					auditStyleCode.Render(f.Code),
					auditStylePath.Render(loc),
					auditStyleSubtle.Render("·"),
					f.Message,
				)
			} else {
				fmt.Fprintf(w, "    %s  %s  %s\n",
					auditStyleCode.Render(f.Code),
					auditStyleSubtle.Render("·"),
					f.Message,
				)
			}
			if f.Hint != "" {
				fmt.Fprintf(w, "    %s\n", auditStyleHint.Render("hint: "+f.Hint))
			}
			printed++
		}
	}

	if printed == 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s  %s\n", auditStyleOK.Render("All clear"), auditStyleSubtle.Render("— no findings at or above the severity threshold."))
	}

	footer := fmt.Sprintf("errors: %d  warnings: %d  info: %d",
		r.Counts[audit.SeverityError],
		r.Counts[audit.SeverityWarn],
		r.Counts[audit.SeverityInfo],
	)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", auditStyleSubtle.Render(footer))
	fmt.Fprintln(w)
}

func buildCountLine(r *audit.Report) string {
	ne, nw, ni := r.Counts[audit.SeverityError], r.Counts[audit.SeverityWarn], r.Counts[audit.SeverityInfo]
	sep := auditStyleSubtle.Render("  ·  ")
	return strings.Join(
		[]string{
			lipgloss.NewStyle().Foreground(ozRed).Bold(true).Render(fmt.Sprintf("errors %d", ne)),
			lipgloss.NewStyle().Foreground(ozOrange).Bold(true).Render(fmt.Sprintf("warnings %d", nw)),
			lipgloss.NewStyle().Foreground(ozLavend).Render(fmt.Sprintf("info %d", ni)),
		},
		sep,
	)
}
