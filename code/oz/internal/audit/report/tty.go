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
	ozRed    = lipgloss.Color("#EF4444")

	auditStyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ozPurple)
	auditStyleSubtle   = lipgloss.NewStyle().Foreground(ozFaint)
	auditStyleCode     = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	auditStylePath     = lipgloss.NewStyle().Foreground(ozFaint)
	auditStyleOK       = lipgloss.NewStyle().Bold(true).Foreground(ozGreen)
	auditStyleRule     = lipgloss.NewStyle().Foreground(ozFaint)
	auditStyleSectionE = lipgloss.NewStyle().Bold(true).Foreground(ozRed)
	auditStyleSectionW = lipgloss.NewStyle().Bold(true).Foreground(ozOrange)
	auditStyleSectionI = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	auditStyleHint     = lipgloss.NewStyle().Foreground(ozFaint)
)

// maxFindingsPerSeverityStyled caps how many findings print per severity in the
// TTY view. The full report is always available via `oz audit --json`.
const maxFindingsPerSeverityStyled = 25

// WriteHumanStyled writes a TTY-oriented, colorized report using lipgloss.
// It mirrors WriteHuman: same fields, minSeverity filter, and count footer.
// To keep long runs readable, at most maxFindingsPerSeverityStyled items print
// per severity; the rest are summarized with a pointer to --json.
// Call only when stdout is a color-capable TTY; for automation or LLM tooling,
// use WriteJSON or WriteHuman (plain text) instead.
func WriteHumanStyled(w io.Writer, r *audit.Report, minSeverity audit.Severity) {
	minRank := severityRank(minSeverity)
	printed := 0

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s\n", auditStyleTitle.Render("oz audit"), auditStyleSubtle.Render("workspace health"))

	fmt.Fprintf(w, "  %s\n", styledCountLine(r, false))
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
		fmt.Fprintf(w, "  %s", meta.style.Render(meta.text))
		if len(group) > maxFindingsPerSeverityStyled {
			fmt.Fprintf(w, "  %s", auditStyleSubtle.Render(fmt.Sprintf("(showing %d of %d)", maxFindingsPerSeverityStyled, len(group))))
		}
		fmt.Fprintln(w)
		lim := len(group)
		if lim > maxFindingsPerSeverityStyled {
			lim = maxFindingsPerSeverityStyled
		}
		for i := 0; i < lim; i++ {
			f := group[i]
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
		extra := len(group) - lim
		if extra > 0 {
			fmt.Fprintf(w, "    %s\n", auditStyleSubtle.Render(
				fmt.Sprintf("… and %d more in this section not shown. Use `oz audit --json` (or a narrower `--severity` / `--only`) for the full set.", extra)))
		}
	}

	if printed == 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s  %s\n", auditStyleOK.Render("All clear"), auditStyleSubtle.Render("— no findings at or above the severity threshold."))
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", styledCountLine(r, true))
	fmt.Fprintln(w)
}

// styledCountLine renders errors / warnings / info with matching severity colors.
// withLabels uses "errors: 38" style; otherwise "errors 38" (for the top summary).
func styledCountLine(r *audit.Report, withLabels bool) string {
	ne, nw, ni := r.Counts[audit.SeverityError], r.Counts[audit.SeverityWarn], r.Counts[audit.SeverityInfo]
	sep := auditStyleSubtle.Render("  ·  ")
	ef := func(n int) string {
		if withLabels {
			return lipgloss.NewStyle().Foreground(ozRed).Bold(true).Render(fmt.Sprintf("errors: %d", n))
		}
		return lipgloss.NewStyle().Foreground(ozRed).Bold(true).Render(fmt.Sprintf("errors %d", n))
	}
	wf := func(n int) string {
		if withLabels {
			return lipgloss.NewStyle().Foreground(ozOrange).Bold(true).Render(fmt.Sprintf("warnings: %d", n))
		}
		return lipgloss.NewStyle().Foreground(ozOrange).Bold(true).Render(fmt.Sprintf("warnings %d", n))
	}
	ifN := func(n int) string {
		if withLabels {
			return lipgloss.NewStyle().Foreground(ozLavend).Render(fmt.Sprintf("info: %d", n))
		}
		return lipgloss.NewStyle().Foreground(ozLavend).Render(fmt.Sprintf("info %d", n))
	}
	return strings.Join(
		[]string{ef(ne), wf(nw), ifN(ni)},
		sep,
	)
}
