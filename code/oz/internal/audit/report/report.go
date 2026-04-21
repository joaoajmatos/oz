// Package report provides human-readable and JSON renderers for audit reports.
package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/joaoajmatos/oz/internal/audit"
)

// WriteJSON writes r to w as indented JSON followed by a newline.
func WriteJSON(w io.Writer, r *audit.Report) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

// WriteHuman writes a human-readable summary of r to w.
// Severities below minSeverity (less severe) are skipped.
// Within each severity group findings are printed as:
//
//	CODE · file:line · message
//	  hint (if present)
func WriteHuman(w io.Writer, r *audit.Report, minSeverity audit.Severity) {
	minRank := severityRank(minSeverity)
	printed := 0

	for _, sev := range []audit.Severity{audit.SeverityError, audit.SeverityWarn, audit.SeverityInfo} {
		if severityRank(sev) > minRank {
			continue
		}
		for _, f := range r.Findings {
			if f.Severity != sev {
				continue
			}
			loc := f.File
			if f.Line > 0 && loc != "" {
				loc = fmt.Sprintf("%s:%d", loc, f.Line)
			}
			if loc != "" {
				fmt.Fprintf(w, "  %s  %s · %s · %s\n", sev, f.Code, loc, f.Message)
			} else {
				fmt.Fprintf(w, "  %s  %s · %s\n", sev, f.Code, f.Message)
			}
			if f.Hint != "" {
				fmt.Fprintf(w, "    hint: %s\n", f.Hint)
			}
			printed++
		}
	}

	if printed == 0 {
		fmt.Fprintf(w, "  ok — no findings\n")
	}

	fmt.Fprintf(w, "\n  errors: %d  warnings: %d  info: %d\n",
		r.Counts[audit.SeverityError],
		r.Counts[audit.SeverityWarn],
		r.Counts[audit.SeverityInfo],
	)
}

func severityRank(s audit.Severity) int {
	switch s {
	case audit.SeverityError:
		return 0
	case audit.SeverityWarn:
		return 1
	default:
		return 2
	}
}
