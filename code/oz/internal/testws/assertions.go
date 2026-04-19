package testws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/oz-tools/oz/internal/query"
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
