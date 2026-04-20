// Package staleness implements the oz audit staleness check.
// It reports when graph.json or semantic.json are out of date
// relative to the current workspace content.
package staleness

import (
	"errors"
	"fmt"
	"os"

	"github.com/oz-tools/oz/internal/audit"
	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
	"github.com/oz-tools/oz/internal/semantic"
)

// Check implements audit.Check for staleness detection.
type Check struct{}

// Name returns the check name.
func (c *Check) Name() string { return "staleness" }

// Codes returns the finding codes this check may produce.
func (c *Check) Codes() []string {
	return []string{"STALE001", "STALE002", "STALE003", "STALE004"}
}

// Run loads graph.json and semantic.json from root and returns staleness findings.
func (c *Check) Run(root string, _ audit.Options) ([]audit.Finding, error) {
	ondisk, err := ozcontext.LoadGraph(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("graph.json not found — run 'oz context build' first")
		}
		return nil, fmt.Errorf("staleness: load graph: %w", err)
	}

	result, err := ozcontext.Build(root)
	if err != nil {
		return nil, fmt.Errorf("staleness: build graph: %w", err)
	}
	ozcontext.Normalise(result.Graph)

	overlay, err := semantic.Load(root)
	if err != nil {
		return nil, fmt.Errorf("staleness: load semantic: %w", err)
	}

	return runCheck(ondisk, result.Graph, overlay), nil
}

// runCheck runs staleness detection against pre-loaded artifacts.
// Separated from Run for testability.
func runCheck(ondisk *graph.Graph, fresh *graph.Graph, overlay *semantic.Overlay) []audit.Finding {
	var findings []audit.Finding

	// STALE001: workspace has changed since graph.json was last written.
	if fresh.ContentHash != ondisk.ContentHash {
		findings = append(findings, audit.Finding{
			Check:    "staleness",
			Code:     "STALE001",
			Severity: audit.SeverityError,
			Message:  "graph.json is stale — workspace has changed since last build",
			Hint:     "run 'oz context build' to regenerate",
		})
	}

	// STALE004: no semantic overlay present.
	if overlay == nil {
		findings = append(findings, audit.Finding{
			Check:    "staleness",
			Code:     "STALE004",
			Severity: audit.SeverityInfo,
			Message:  "semantic.json not found — no semantic overlay present",
			Hint:     "run 'oz context enrich' to generate",
		})
		return findings
	}

	// STALE002: semantic overlay was built from a different graph.
	if semantic.IsStale(overlay, ondisk.ContentHash) {
		findings = append(findings, audit.Finding{
			Check:    "staleness",
			Code:     "STALE002",
			Severity: audit.SeverityWarn,
			Message:  "semantic.json is stale — graph has changed since last enrichment",
			Hint:     "run 'oz context enrich' to refresh",
		})
	}

	// STALE003: unreviewed items in the semantic overlay.
	if n := countUnreviewed(overlay); n > 0 {
		findings = append(findings, audit.Finding{
			Check:    "staleness",
			Code:     "STALE003",
			Severity: audit.SeverityWarn,
			Message:  fmt.Sprintf("semantic.json has %d unreviewed item(s)", n),
			Hint:     "run 'oz context review' to mark items as reviewed",
		})
	}

	return findings
}

// countUnreviewed returns the total number of unreviewed concepts and edges.
func countUnreviewed(o *semantic.Overlay) int {
	n := 0
	for _, c := range o.Concepts {
		if !c.Reviewed {
			n++
		}
	}
	for _, e := range o.Edges {
		if !e.Reviewed {
			n++
		}
	}
	return n
}
