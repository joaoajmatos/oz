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

	"github.com/oz-tools/oz/internal/semantic"
)

// Options controls the review workflow.
type Options struct {
	// AcceptAll marks every unreviewed item as reviewed without prompting.
	AcceptAll bool

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
		fmt.Fprintln(opts.Out, "nothing to review — all items already reviewed")
		return Summary{NothingToReview: true}, nil
	}

	// Print the diff table.
	printConceptTable(opts.Out, overlay, pendingConcepts)
	printEdgeTable(opts.Out, overlay, pendingEdges)

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
		fmt.Fprintf(opts.Out, "\naccepted %d item(s) (--accept-all)\n", total)
		return Summary{Accepted: total}, nil
	}

	// Interactive mode.
	accepted, rejected, err := interactiveReview(opts, overlay, pendingConcepts, pendingEdges)
	if err != nil {
		return Summary{}, err
	}

	if accepted+rejected > 0 {
		// Compact: remove rejected edges (rejected concepts are already removed).
		overlay.Edges = removeRejectedEdges(overlay.Edges)
		if err := semantic.Write(workspacePath, overlay); err != nil {
			return Summary{}, fmt.Errorf("write semantic.json: %w", err)
		}
	}

	fmt.Fprintf(opts.Out, "\naccepted %d, rejected %d\n", accepted, rejected)
	return Summary{Accepted: accepted, Rejected: rejected}, nil
}

// printConceptTable prints unreviewed concepts in a formatted table.
func printConceptTable(w io.Writer, o *semantic.Overlay, indices []int) {
	if len(indices) == 0 {
		return
	}
	fmt.Fprintf(w, "\nUnreviewed concepts (%d):\n", len(indices))
	fmt.Fprintf(w, "  %-28s  %-10s  %-4s  %s\n", "NAME", "TAG", "CONF", "SOURCE")
	fmt.Fprintln(w, "  "+strings.Repeat("─", 72))
	for _, i := range indices {
		c := o.Concepts[i]
		sources := strings.Join(c.SourceFiles, ", ")
		if len(sources) > 36 {
			sources = sources[:33] + "..."
		}
		fmt.Fprintf(w, "  %-28s  %-10s  %.2f  %s\n",
			truncate(c.Name, 28), c.Tag, c.Confidence, sources)
	}
}

// printEdgeTable prints unreviewed edges in a formatted table.
func printEdgeTable(w io.Writer, o *semantic.Overlay, indices []int) {
	if len(indices) == 0 {
		return
	}
	fmt.Fprintf(w, "\nUnreviewed edges (%d):\n", len(indices))
	fmt.Fprintf(w, "  %-28s  %-26s  %-28s  %-4s\n", "FROM", "TYPE", "TO", "CONF")
	fmt.Fprintln(w, "  "+strings.Repeat("─", 90))
	for _, i := range indices {
		e := o.Edges[i]
		fmt.Fprintf(w, "  %-28s  %-26s  %-28s  %.2f\n",
			truncate(e.From, 28), truncate(e.Type, 26), truncate(e.To, 28), e.Confidence)
	}
}

// interactiveReview prompts the user to accept or reject each unreviewed item.
// Returns the number accepted and rejected.
func interactiveReview(
	opts Options,
	overlay *semantic.Overlay,
	pendingConcepts, pendingEdges []int,
) (accepted, rejected int, err error) {
	scanner := bufio.NewScanner(opts.In)

	fmt.Fprintln(opts.Out, "\nReview each item: [y]es / [n]o / [q]uit")

	// Track which concept indices are rejected so we can remove them.
	rejectedConceptSet := make(map[int]bool)

	for _, i := range pendingConcepts {
		c := overlay.Concepts[i]
		fmt.Fprintf(opts.Out, "\nconcept: %s (%s, %.2f)\n  %s\n  accept? [y/N/q] ",
			c.Name, c.Tag, c.Confidence, c.Description)

		ans, quit := prompt(scanner)
		if quit {
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
		e := overlay.Edges[i]
		fmt.Fprintf(opts.Out, "\nedge: %s -[%s]-> %s (%.2f)\n  accept? [y/N/q] ",
			e.From, e.Type, e.To, e.Confidence)

		ans, quit := prompt(scanner)
		if quit {
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

	return accepted, rejected, nil
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
