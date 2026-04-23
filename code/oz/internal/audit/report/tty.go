package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/termstyle"
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
	fmt.Fprintf(w, "  %s  %s\n", termstyle.AuditTitle.Render("oz audit"), termstyle.Subtle.Render("workspace health"))

	fmt.Fprintf(w, "  %s\n", styledCountLine(r, false))
	fmt.Fprintln(w, "  "+termstyle.AuditRule.Render(strings.Repeat("─", 58)))

	labels := map[audit.Severity]struct {
		text  string
		style lipgloss.Style
	}{
		audit.SeverityError: {"Errors", termstyle.SectionErr},
		audit.SeverityWarn:  {"Warnings", termstyle.SectionWarn},
		audit.SeverityInfo:  {"Info", termstyle.SectionInfo},
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
			fmt.Fprintf(w, "  %s", termstyle.Subtle.Render(fmt.Sprintf("(showing %d of %d)", maxFindingsPerSeverityStyled, len(group))))
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
					termstyle.AuditCode.Render(f.Code),
					termstyle.AuditPath.Render(loc),
					termstyle.Subtle.Render("·"),
					f.Message,
				)
			} else {
				fmt.Fprintf(w, "    %s  %s  %s\n",
					termstyle.AuditCode.Render(f.Code),
					termstyle.Subtle.Render("·"),
					f.Message,
				)
			}
			if f.Hint != "" {
				fmt.Fprintf(w, "    %s\n", termstyle.AuditHint.Render("hint: "+f.Hint))
			}
			printed++
		}
		extra := len(group) - lim
		if extra > 0 {
			fmt.Fprintf(w, "    %s\n", termstyle.Subtle.Render(
				fmt.Sprintf("… and %d more in this section not shown. Use `oz audit --json` (or a narrower `--severity` / `--only`) for the full set.", extra)))
		}
	}

	if printed == 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s  %s\n", termstyle.AuditOKLine.Render("All clear"), termstyle.Subtle.Render("— no findings at or above the severity threshold."))
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", styledCountLine(r, true))
	fmt.Fprintln(w)
}

// styledCountLine renders errors / warnings / info with matching severity colors.
// withLabels uses "errors: 38" style; otherwise "errors 38" (for the top summary).
func styledCountLine(r *audit.Report, withLabels bool) string {
	ne, nw, ni := r.Counts[audit.SeverityError], r.Counts[audit.SeverityWarn], r.Counts[audit.SeverityInfo]
	sep := termstyle.Subtle.Render("  ·  ")
	ef := func(n int) string {
		if withLabels {
			return termstyle.CountError.Render(fmt.Sprintf("errors: %d", n))
		}
		return termstyle.CountError.Render(fmt.Sprintf("errors %d", n))
	}
	wf := func(n int) string {
		if withLabels {
			return termstyle.CountWarn.Render(fmt.Sprintf("warnings: %d", n))
		}
		return termstyle.CountWarn.Render(fmt.Sprintf("warnings %d", n))
	}
	ifN := func(n int) string {
		if withLabels {
			return termstyle.CountInfo.Render(fmt.Sprintf("info: %d", n))
		}
		return termstyle.CountInfo.Render(fmt.Sprintf("info %d", n))
	}
	return strings.Join(
		[]string{ef(ne), wf(nw), ifN(ni)},
		sep,
	)
}
