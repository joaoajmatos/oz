// Package review implements the oz context review workflow.
//
// It loads context/semantic.json, presents unreviewed concepts and edges in
// a human-readable table, and lets the user accept or reject each item.
// Accepted items have Reviewed set to true; rejected items are removed.
//
// Use --accept-all to mark every unreviewed item as reviewed without prompting
// (suitable for CI pipelines).
package review

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/joaoajmatos/oz/internal/semantic"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// --- Oz CLI palette (aligned with cmd/ui.go and internal/audit/report) -----

var (
	ozPurple = lipgloss.Color("#7C3AED")
	ozFaint  = lipgloss.Color("#6B7280")
	ozGreen  = lipgloss.Color("#10B981")
	ozLavend = lipgloss.Color("#A78BFA")
	ozSoft   = lipgloss.Color("#D1D5DB")
	ozOrange = lipgloss.Color("#F59E0B")
)

// Options controls the review workflow.
type Options struct {
	// AcceptAll marks every unreviewed item as reviewed without prompting.
	AcceptAll bool

	// NoColor forces plain text with no ANSI sequences, even on a TTY.
	NoColor bool

	// Out is where the review UI is written. Defaults to os.Stdout.
	Out io.Writer

	// In is the source of user input. Defaults to os.Stdin.
	In io.Reader
}

// Summary reports how many items were accepted and rejected during a run.
type Summary struct {
	Accepted int
	Rejected int
	// NothingToReview is true when semantic.json has no unreviewed items.
	NothingToReview bool
}

// Run presents unreviewed items from context/semantic.json and processes them
// according to opts. Returns a summary of changes and writes semantic.json back
// to disk if any changes were made.
func Run(workspacePath string, opts Options) (Summary, error) {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.In == nil {
		opts.In = os.Stdin
	}
	ui := newUI(opts)

	overlay, err := semantic.Load(workspacePath)
	if err != nil {
		return Summary{}, fmt.Errorf("load semantic.json: %w", err)
	}
	if overlay == nil {
		return Summary{}, fmt.Errorf("context/semantic.json not found — run 'oz context enrich' first")
	}

	// Collect unreviewed concepts and edges.
	var pendingConcepts []int // indices into overlay.Concepts
	var pendingEdges []int    // indices into overlay.Edges
	for i, c := range overlay.Concepts {
		if !c.Reviewed {
			pendingConcepts = append(pendingConcepts, i)
		}
	}
	for i, e := range overlay.Edges {
		if !e.Reviewed {
			pendingEdges = append(pendingEdges, i)
		}
	}

	total := len(pendingConcepts) + len(pendingEdges)
	if total == 0 {
		fmt.Fprintln(opts.Out, ui.render(ui.styleOK, "✓")+" "+ui.render(ui.styleSubtle, "nothing to review — all items already marked reviewed"))
		return Summary{NothingToReview: true}, nil
	}

	// Lead-in title (match enrich / audit tone).
	fmt.Fprintln(opts.Out)
	fmt.Fprintln(opts.Out, ui.render(ui.styleTitle, "context review")+"  "+ui.render(ui.styleSubtle, "unreviewed semantic items"))
	fmt.Fprintln(opts.Out, "  "+ui.render(ui.styleRule, strings.Repeat("─", 58)))

	printConceptTable(ui, overlay, pendingConcepts)
	printEdgeTable(ui, overlay, pendingEdges)

	if opts.AcceptAll {
		for _, i := range pendingConcepts {
			overlay.Concepts[i].Reviewed = true
		}
		for _, i := range pendingEdges {
			overlay.Edges[i].Reviewed = true
		}
		if err := semantic.Write(workspacePath, overlay); err != nil {
			return Summary{}, fmt.Errorf("write semantic.json: %w", err)
		}
		fmt.Fprintln(opts.Out)
		fmt.Fprintf(opts.Out, "%s %s\n", ui.render(ui.styleOK, "✓"), ui.render(ui.styleSubtle, fmt.Sprintf("marked %d item(s) as reviewed (--accept-all)", total)))
		return Summary{Accepted: total}, nil
	}

	// Interactive mode uses a staged copy. Changes are only written when the
	// review completes, so quitting mid-run cannot persist partial decisions.
	staged := cloneOverlay(overlay)
	accepted, rejected, completed, err := interactiveReview(opts, ui, staged, pendingConcepts, pendingEdges, total)
	if err != nil {
		return Summary{}, err
	}

	if completed && accepted+rejected > 0 {
		// Compact: remove rejected edges (rejected concepts are already removed).
		staged.Edges = removeRejectedEdges(staged.Edges)
		if err := semantic.Write(workspacePath, staged); err != nil {
			return Summary{}, fmt.Errorf("write semantic.json: %w", err)
		}
	}
	if !completed {
		fmt.Fprintln(opts.Out)
		fmt.Fprintln(opts.Out, ui.render(ui.styleWarn, "aborted")+" — "+ui.render(ui.styleSubtle, "in-session changes were not saved"))
	}

	fmt.Fprintln(opts.Out)
	fmt.Fprintln(opts.Out, ui.render(ui.styleSubtle, "—"))
	fmt.Fprintf(opts.Out, "  %s  %s  %s  %s\n",
		ui.render(ui.styleSubtle, "result"),
		ui.render(ui.styleOK, fmt.Sprintf("accepted %d", accepted)),
		ui.render(ui.styleSubtle, "·"),
		ui.render(ui.styleSubtle, fmt.Sprintf("rejected %d", rejected)))
	return Summary{Accepted: accepted, Rejected: rejected}, nil
}

// ui holds lipgloss styles and whether to apply color.
type ui struct {
	w     io.Writer
	color bool
	// Pre-built styles
	styleTitle, styleSubtle, styleHeader, styleDim, styleValue, styleTag, styleTableText lipgloss.Style
	styleOK, styleWarn, stylePrompt, styleDesc, styleRule, styleAccent lipgloss.Style
}

func newUI(opts Options) *ui {
	color := !opts.NoColor && os.Getenv("NO_COLOR") == "" && useColor(opts.Out)
	u := &ui{w: opts.Out, color: color}
	plain := lipgloss.NewStyle()

	u.styleTitle = lipgloss.NewStyle().Bold(true).Foreground(ozPurple)
	u.styleSubtle = lipgloss.NewStyle().Foreground(ozFaint)
	u.styleHeader = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	u.styleDim = lipgloss.NewStyle().Foreground(ozFaint)
	u.styleValue = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	u.styleTag = lipgloss.NewStyle().Foreground(ozSoft)
	u.styleTableText = lipgloss.NewStyle().Foreground(ozSoft)
	u.styleOK = lipgloss.NewStyle().Bold(true).Foreground(ozGreen)
	u.styleWarn = lipgloss.NewStyle().Foreground(ozOrange)
	u.stylePrompt = lipgloss.NewStyle().Bold(true).Foreground(ozLavend)
	u.styleDesc = lipgloss.NewStyle().Foreground(ozSoft)
	u.styleRule = lipgloss.NewStyle().Foreground(ozFaint)
	u.styleAccent = lipgloss.NewStyle().Foreground(ozPurple).Bold(true)

	if !color {
		for _, s := range []*lipgloss.Style{&u.styleTitle, &u.styleSubtle, &u.styleHeader, &u.styleDim, &u.styleValue, &u.styleTag, &u.styleTableText, &u.styleOK, &u.styleWarn, &u.stylePrompt, &u.styleDesc, &u.styleRule, &u.styleAccent} {
			*s = plain
		}
	}
	_ = plain
	return u
}

func (u *ui) render(st lipgloss.Style, s string) string {
	if u.color {
		return st.Render(s)
	}
	return s
}

func useColor(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

// printConceptTable prints unreviewed concepts in a formatted table.
func printConceptTable(u *ui, o *semantic.Overlay, indices []int) {
	if len(indices) == 0 {
		return
	}
	fmt.Fprintln(u.w)
	fmt.Fprintln(u.w, "  "+u.render(u.styleAccent, fmt.Sprintf("Unreviewed concepts (%d)", len(indices))))
	fmt.Fprintf(u.w, "  %s  %s  %s  %s\n",
		u.render(u.styleHeader, padRunes("name", 28)),
		u.render(u.styleHeader, "tag       "),
		u.render(u.styleHeader, "conf "),
		u.render(u.styleHeader, "source"))
	fmt.Fprintln(u.w, "  "+u.render(u.styleRule, strings.Repeat("─", 72)))
	for _, i := range indices {
		c := o.Concepts[i]
		sources := strings.Join(c.SourceFiles, ", ")
		if runewidth(sources) > 36 {
			sources = truncateRunes(sources, 36)
		}
		fmt.Fprintf(u.w, "  %s  %s  %s  %s\n",
			u.render(u.styleValue, padRunes(truncateRunes(c.Name, 28), 28)),
			u.render(u.styleTag, padRunes(c.Tag, 9)),
			u.render(u.styleTableText, fmt.Sprintf("%4.2f", c.Confidence)),
			u.render(u.styleDim, sources),
		)
	}
}

// printEdgeTable prints unreviewed edges in a formatted table.
func printEdgeTable(u *ui, o *semantic.Overlay, indices []int) {
	if len(indices) == 0 {
		return
	}
	fmt.Fprintln(u.w)
	fmt.Fprintln(u.w, "  "+u.render(u.styleAccent, fmt.Sprintf("Unreviewed edges (%d)", len(indices))))
	fmt.Fprintf(u.w, "  %s  %s  %s  %s\n",
		u.render(u.styleHeader, padRunes("from", 28)),
		u.render(u.styleHeader, padRunes("type", 26)),
		u.render(u.styleHeader, padRunes("to", 28)),
		u.render(u.styleHeader, "conf "),
	)
	fmt.Fprintln(u.w, "  "+u.render(u.styleRule, strings.Repeat("─", 90)))
	for _, i := range indices {
		e := o.Edges[i]
		fmt.Fprintf(u.w, "  %s  %s  %s  %s\n",
			u.render(u.styleValue, padRunes(truncateRunes(e.From, 28), 28)),
			u.render(u.styleTableText, padRunes(truncateRunes(e.Type, 26), 26)),
			u.render(u.styleValue, padRunes(truncateRunes(e.To, 28), 28)),
			u.render(u.styleTableText, fmt.Sprintf("%4.2f", e.Confidence)),
		)
	}
}

// interactiveReview prompts the user to accept or reject each unreviewed item.
// totalItems is concepts + edges for progress display.
func interactiveReview(
	opts Options,
	ui *ui,
	overlay *semantic.Overlay,
	pendingConcepts, pendingEdges []int,
	totalItems int,
) (accepted, rejected int, completed bool, err error) {
	scanner := bufio.NewScanner(opts.In)

	fmt.Fprintln(opts.Out)
	fmt.Fprintln(opts.Out, "  "+ui.render(ui.styleAccent, "Review each item")+"  "+ui.render(ui.styleSubtle, "[y]es  [n]o  [q]uit"))
	fmt.Fprintln(opts.Out, "  "+ui.render(ui.styleRule, strings.Repeat("─", 58)))

	completed = true
	step := 0

	// Track which concept indices are rejected so we can remove them.
	rejectedConceptSet := make(map[int]bool)

	for _, i := range pendingConcepts {
		step++
		c := overlay.Concepts[i]
		fmt.Fprintln(opts.Out)
		fmt.Fprintf(opts.Out, "  %s\n", ui.render(ui.styleSubtle, fmt.Sprintf("%d / %d — concept", step, totalItems)))
		fmt.Fprintf(opts.Out, "  %s  %s  %s\n",
			ui.render(ui.styleTitle, c.Name),
			ui.render(ui.styleSubtle, "·"),
			ui.render(ui.styleTag, fmt.Sprintf("%s · %.2f", c.Tag, c.Confidence)))
		for _, line := range wrapText(c.Description, 78, "    ") {
			fmt.Fprintln(opts.Out, ui.render(ui.styleDesc, line))
		}
		fmt.Fprint(opts.Out, "  "+ui.render(ui.stylePrompt, "accept?")+" "+ui.render(ui.styleSubtle, "[y/N/q] "))

		ans, quit := prompt(scanner)
		if quit {
			completed = false
			break
		}
		if ans {
			overlay.Concepts[i].Reviewed = true
			accepted++
		} else {
			rejectedConceptSet[i] = true
			rejected++
		}
	}

	// Remove rejected concepts (in reverse order to keep indices valid).
	if len(rejectedConceptSet) > 0 {
		var kept []semantic.Concept
		for j, c := range overlay.Concepts {
			if !rejectedConceptSet[j] {
				kept = append(kept, c)
			}
		}
		overlay.Concepts = kept
	}

	for _, i := range pendingEdges {
		step++
		e := overlay.Edges[i]
		fmt.Fprintln(opts.Out)
		fmt.Fprintf(opts.Out, "  %s\n", ui.render(ui.styleSubtle, fmt.Sprintf("%d / %d — edge", step, totalItems)))
		edgeLine := ui.render(ui.styleValue, e.From) + ui.render(ui.styleDim, "  —[") + ui.render(ui.styleTableText, e.Type) + ui.render(ui.styleDim, "]→  ") + ui.render(ui.styleValue, e.To) + ui.render(ui.styleSubtle, "  ·  ") + ui.render(ui.styleTableText, fmt.Sprintf("%.2f", e.Confidence))
		fmt.Fprintln(opts.Out, "  "+edgeLine)
		fmt.Fprint(opts.Out, "  "+ui.render(ui.stylePrompt, "accept?")+" "+ui.render(ui.styleSubtle, "[y/N/q] "))

		ans, quit := prompt(scanner)
		if quit {
			completed = false
			break
		}
		if ans {
			overlay.Edges[i].Reviewed = true
			accepted++
		} else {
			overlay.Edges[i].Reviewed = false // mark for removal via sentinel
			// Use a sentinel: set type to empty so removeRejectedEdges can filter it.
			overlay.Edges[i].Type = ""
			rejected++
		}
	}

	return accepted, rejected, completed, nil
}

// removeRejectedEdges removes edges that were marked for rejection (Type == "").
func removeRejectedEdges(edges []semantic.ConceptEdge) []semantic.ConceptEdge {
	var kept []semantic.ConceptEdge
	for _, e := range edges {
		if e.Type != "" {
			kept = append(kept, e)
		}
	}
	return kept
}

// prompt reads a single line and returns (accepted bool, quit bool).
// Default (empty / n) means rejected.
func prompt(scanner *bufio.Scanner) (accepted bool, quit bool) {
	if !scanner.Scan() {
		return false, true
	}
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	switch ans {
	case "q", "quit":
		return false, true
	case "y", "yes":
		return true, false
	default:
		return false, false
	}
}

func runewidth(s string) int { return utf8.RuneCountInString(s) }

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

func padRunes(s string, w int) string {
	n := utf8.RuneCountInString(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

// wrapText wraps space-separated text to lines at most maxLine runes, each prefixed with indent.
func wrapText(s string, maxLine int, indent string) []string {
	words := strings.Fields(strings.TrimSpace(s))
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var b strings.Builder
	for _, word := range words {
		if b.Len() == 0 {
			b.WriteString(indent)
			b.WriteString(word)
			continue
		}
		candidate := b.String() + " " + word
		if utf8.RuneCountInString(candidate) > maxLine {
			lines = append(lines, b.String())
			b.Reset()
			b.WriteString(indent)
			b.WriteString(word)
		} else {
			b.WriteString(" ")
			b.WriteString(word)
		}
	}
	if b.Len() > 0 {
		lines = append(lines, b.String())
	}
	return lines
}

func cloneOverlay(in *semantic.Overlay) *semantic.Overlay {
	if in == nil {
		return nil
	}

	out := &semantic.Overlay{
		SchemaVersion: in.SchemaVersion,
		GraphHash:     in.GraphHash,
		Model:         in.Model,
		GeneratedAt:   in.GeneratedAt,
		Concepts:      make([]semantic.Concept, len(in.Concepts)),
		Edges:         make([]semantic.ConceptEdge, len(in.Edges)),
	}
	for i, c := range in.Concepts {
		cCopy := c
		if c.SourceFiles != nil {
			cCopy.SourceFiles = append([]string(nil), c.SourceFiles...)
		}
		out.Concepts[i] = cCopy
	}
	copy(out.Edges, in.Edges)
	return out
}