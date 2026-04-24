package testws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/query"
)

// Matches reports whether a QueryCase's expectations are satisfied by result.
// Used by the golden suite harness.
func (q QueryCase) Matches(r query.Result) bool {
	// No-owner expectation.
	if q.ExpectedAgent == "" || q.ExpectedAgent == "null" {
		return r.Agent == "" && (q.Reason == "" || r.Reason == q.Reason)
	}

	if r.Agent != q.ExpectedAgent {
		// Accept if the expected agent appears in CandidateAgents when caller
		// signals ambiguity is acceptable via ExpectedCandidates.
		if len(q.ExpectedCandidates) > 0 {
			for _, c := range r.CandidateAgents {
				if c.Name == q.ExpectedAgent {
					return true
				}
			}
		}
		return false
	}

	if q.MinConfidence > 0 && r.Confidence < q.MinConfidence {
		return false
	}

	return true
}

// ExpectAgent asserts that result routes to the expected agent.
func ExpectAgent(t *testing.T, result query.Result, expected string) {
	t.Helper()
	if result.Agent != expected {
		t.Errorf("expected agent %q, got %q (confidence %.2f)", expected, result.Agent, result.Confidence)
	}
}

// ExpectConfidenceAtLeast asserts that result.Confidence >= min.
func ExpectConfidenceAtLeast(t *testing.T, result query.Result, min float64) {
	t.Helper()
	if result.Confidence < min {
		t.Errorf("expected confidence >= %.2f, got %.2f (agent=%q)", min, result.Confidence, result.Agent)
	}
}

// ExpectContextBlock asserts that result contains a context block matching
// the given file path. Section is optional — pass "" to match any section.
func ExpectContextBlock(t *testing.T, result query.Result, file, section string) {
	t.Helper()
	for _, cb := range result.ContextBlocks {
		if cb.File == file && (section == "" || cb.Section == section) {
			return
		}
	}
	if section != "" {
		t.Errorf("expected context block %s#%s, not found in %v", file, section, formatContextBlocks(result.ContextBlocks))
	} else {
		t.Errorf("expected context block %s, not found in %v", file, formatContextBlocks(result.ContextBlocks))
	}
}

// ExpectTrustOrder asserts that context blocks are ordered by trust tier.
// tiers should be provided highest-to-lowest, e.g. "specs", "docs", "context", "notes".
func ExpectTrustOrder(t *testing.T, result query.Result, tiers ...string) {
	t.Helper()
	rank := make(map[string]int, len(tiers))
	for i, tier := range tiers {
		rank[tier] = i
	}

	prev := -1
	for _, cb := range result.ContextBlocks {
		r, ok := rank[cb.Trust]
		if !ok {
			continue
		}
		if r < prev {
			t.Errorf("context blocks out of trust order: %v appears after higher trust content", cb)
			return
		}
		prev = r
	}
}

// ExpectExcluded asserts that the given path prefix appears in result.Excluded.
func ExpectExcluded(t *testing.T, result query.Result, prefix string) {
	t.Helper()
	for _, e := range result.Excluded {
		if strings.HasPrefix(e, prefix) || e == prefix {
			return
		}
	}
	t.Errorf("expected %q in excluded, got %v", prefix, result.Excluded)
}

// ExpectAmbiguous asserts that result signals ambiguity
// (confidence < 0.7 and candidate_agents populated).
func ExpectAmbiguous(t *testing.T, result query.Result) {
	t.Helper()
	if result.Confidence >= 0.7 {
		t.Errorf("expected ambiguous result (confidence < 0.7), got %.2f for agent %q", result.Confidence, result.Agent)
	}
	if len(result.CandidateAgents) == 0 {
		t.Errorf("expected candidate_agents populated for ambiguous result")
	}
}

// ExpectNoOwner asserts that no agent owns the task.
func ExpectNoOwner(t *testing.T, result query.Result) {
	t.Helper()
	if result.Agent != "" {
		t.Errorf("expected no owner, got agent %q (confidence %.2f)", result.Agent, result.Confidence)
	}
	if result.Reason != "no_clear_owner" {
		t.Errorf("expected reason %q, got %q", "no_clear_owner", result.Reason)
	}
}

func formatContextBlocks(blocks []query.ContextBlock) string {
	parts := make([]string, len(blocks))
	for i, cb := range blocks {
		parts[i] = fmt.Sprintf("%s#%s(%s)", cb.File, cb.Section, cb.Trust)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ---- retrieval matchers (Sprint 1) ------------------------------------------
//
// These assertions back the retrieval golden suite. They consume only the
// public fields on query.Result — any new retrieval data (code_entry_points,
// block.Relevance) must be read here once Sprints 2–4 populate it.

// ExpectBlockInTopK asserts that a context_block matching file (and, if
// non-empty, section) is present in the first k entries of
// result.ContextBlocks. k <= 0 searches the whole slice.
func ExpectBlockInTopK(t *testing.T, result query.Result, file, section string, k int) {
	t.Helper()
	limit := len(result.ContextBlocks)
	if k > 0 && k < limit {
		limit = k
	}
	for i := 0; i < limit; i++ {
		cb := result.ContextBlocks[i]
		if cb.File == file && (section == "" || cb.Section == section) {
			return
		}
	}
	if section != "" {
		t.Errorf("expected context block %s#%s in top-%d, got %v", file, section, k, formatContextBlocks(result.ContextBlocks))
	} else {
		t.Errorf("expected context block %s in top-%d, got %v", file, k, formatContextBlocks(result.ContextBlocks))
	}
}

// ExpectCodeEntryPoint asserts that a code_entry_point with the given symbol
// name is present in the first k entries. Symbol may be bare ("Run") or
// qualified ("drift.Run"); matching accepts either form once Sprint 3 wires
// the field.
func ExpectCodeEntryPoint(t *testing.T, result query.Result, symbol string, k int) {
	t.Helper()
	entries := codeEntryPoints(result)
	limit := len(entries)
	if k > 0 && k < limit {
		limit = k
	}
	for i := 0; i < limit; i++ {
		if matchesSymbol(entries[i], symbol) {
			return
		}
	}
	t.Errorf("expected code_entry_point %q in top-%d, got %v", symbol, k, formatCodeEntryPoints(entries))
}

// ExpectPackageInTopK asserts that pkg is present in the first k entries of
// result.ImplementingPackages.
func ExpectPackageInTopK(t *testing.T, result query.Result, pkg string, k int) {
	t.Helper()
	limit := len(result.ImplementingPackages)
	if k > 0 && k < limit {
		limit = k
	}
	for i := 0; i < limit; i++ {
		if result.ImplementingPackages[i] == pkg {
			return
		}
	}
	t.Errorf("expected implementing_package %q in top-%d, got %v", pkg, k, result.ImplementingPackages)
}

// ExpectRelevanceDescending asserts that context_blocks are sorted by
// relevance in descending order. Activates once blocks carry a Relevance
// field (R-01, Sprint 2); until then it is a no-op on zero values.
func ExpectRelevanceDescending(t *testing.T, result query.Result) {
	t.Helper()
	var last float64 = -1
	for i, cb := range result.ContextBlocks {
		r := blockRelevance(cb)
		if i > 0 && r > last+1e-9 {
			t.Errorf("context blocks not descending by relevance at %d: %.4f > %.4f (%s#%s)", i, r, last, cb.File, cb.Section)
			return
		}
		last = r
	}
}

// ExpectTrustBeats asserts that at least one block at winnerTier outranks
// every block at loserTier in result.ContextBlocks. Cross-tier queries use
// this to guarantee specs outrank notes on ties.
func ExpectTrustBeats(t *testing.T, result query.Result, winnerTier, loserTier string) {
	t.Helper()
	firstWinner, firstLoser := -1, -1
	for i, cb := range result.ContextBlocks {
		if firstWinner < 0 && cb.Trust == winnerTier {
			firstWinner = i
		}
		if firstLoser < 0 && cb.Trust == loserTier {
			firstLoser = i
		}
	}
	if firstWinner < 0 {
		t.Errorf("expected at least one block at trust %q, got %v", winnerTier, formatContextBlocks(result.ContextBlocks))
		return
	}
	if firstLoser >= 0 && firstWinner > firstLoser {
		t.Errorf("expected trust %q to outrank %q, got %v", winnerTier, loserTier, formatContextBlocks(result.ContextBlocks))
	}
}

// ExpectNoRelevantContext asserts that retrieval produced no context_blocks
// — used for low-relevance queries (R-09, Sprint 4).
func ExpectNoRelevantContext(t *testing.T, result query.Result) {
	t.Helper()
	if len(result.ContextBlocks) != 0 {
		t.Errorf("expected empty context_blocks (low-relevance), got %v", formatContextBlocks(result.ContextBlocks))
	}
}

// --- code_entry_point compatibility shims -----------------------------------
//
// code_entry_points is a new query.Result field added in Sprint 3. Until it
// lands, codeEntryPoints returns an empty slice and the matcher no-ops on
// empty input. The shim lets Sprint 1 lock the assertion surface without
// blocking on the packet-shape change.

type codeEntryPoint struct {
	Symbol  string
	Package string
}

func codeEntryPoints(r query.Result) []codeEntryPoint {
	// Replaced in Sprint 3 with reflection over r.CodeEntryPoints (or a
	// typed accessor on query.Result).
	_ = r
	return nil
}

func matchesSymbol(e codeEntryPoint, want string) bool {
	if e.Symbol == want {
		return true
	}
	if e.Package != "" && e.Package+"."+e.Symbol == want {
		return true
	}
	return false
}

func formatCodeEntryPoints(entries []codeEntryPoint) string {
	parts := make([]string, len(entries))
	for i, e := range entries {
		if e.Package != "" {
			parts[i] = e.Package + "." + e.Symbol
		} else {
			parts[i] = e.Symbol
		}
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// blockRelevance reads the Sprint-2 Relevance field off a ContextBlock. Until
// the field lands, returns 0 — which keeps ExpectRelevanceDescending inert on
// current output instead of falsely failing.
func blockRelevance(cb query.ContextBlock) float64 {
	_ = cb
	return 0
}
