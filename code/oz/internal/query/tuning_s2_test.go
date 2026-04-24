package query_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/testws"
)

type tuningCandidate struct {
	minRelevance float64
	notesBoost   float64
	affinity     float64
}

type tuningResult struct {
	candidate       tuningCandidate
	retrievalScore  float64
	routingAccuracy float64
}

// TestRetrievalTuningGridS2 runs the Sprint-2 retrieval tuning grid:
//   - retrieval.min_relevance ∈ {0.03, 0.05, 0.08}
//   - retrieval.trust_boost.notes ∈ {0.5, 0.6, 0.7}
//   - retrieval.agent_affinity ∈ {1.1, 1.2, 1.3}
//
// Gate: reject any candidate that regresses 02_medium routing accuracy against
// the default-config baseline for this run.
//
// Score: maximize top-K block-hit rate across 04_retrieval expect_blocks_in_topk.
func TestRetrievalTuningGridS2(t *testing.T) {
	suites, err := testws.LoadGoldenSuites(t, "testdata/golden")
	if err != nil {
		t.Fatalf("load golden suites: %v", err)
	}

	var suite02, suite04 *testws.GoldenSuite
	for _, s := range suites {
		switch s.Name {
		case "02_medium":
			suite02 = s
		case "04_retrieval":
			suite04 = s
		}
	}
	if suite02 == nil || suite04 == nil {
		t.Fatalf("required suites missing: 02_medium=%v 04_retrieval=%v", suite02 != nil, suite04 != nil)
	}

	baselineCfg := query.DefaultScoringConfig()
	baselineRoutingAcc := routingAccuracyForSuite(t, suite02, baselineCfg)
	t.Logf("tuning baseline 02_medium routing accuracy: %.3f", baselineRoutingAcc)

	grid := buildS2Grid()
	var accepted []tuningResult
	for _, c := range grid {
		cfg := query.DefaultScoringConfig()
		cfg.RetrievalMinRelevance = c.minRelevance
		cfg.RetrievalTrustBoostNotes = c.notesBoost
		cfg.RetrievalAgentAffinity = c.affinity

		routingAcc := routingAccuracyForSuite(t, suite02, cfg)
		if routingAcc+1e-9 < baselineRoutingAcc {
			t.Logf("reject min=%.2f notes=%.1f affinity=%.1f: routing %.3f < baseline %.3f",
				c.minRelevance, c.notesBoost, c.affinity, routingAcc, baselineRoutingAcc)
			continue
		}

		retrievalScore := retrievalTopKBlockScore(t, suite04, cfg)
		accepted = append(accepted, tuningResult{
			candidate:       c,
			retrievalScore:  retrievalScore,
			routingAccuracy: routingAcc,
		})
	}

	if len(accepted) == 0 {
		t.Fatalf("no accepted candidates after 02_medium routing gate")
	}

	best := accepted[0]
	for _, r := range accepted[1:] {
		if betterTuningResult(r, best) {
			best = r
		}
	}

	t.Logf("accepted candidates: %d/%d", len(accepted), len(grid))
	for _, r := range accepted {
		t.Logf(
			"candidate min=%.2f notes=%.1f affinity=%.1f => retrieval=%.3f routing=%.3f",
			r.candidate.minRelevance,
			r.candidate.notesBoost,
			r.candidate.affinity,
			r.retrievalScore,
			r.routingAccuracy,
		)
	}
	t.Logf(
		"winner min=%.2f notes=%.1f affinity=%.1f retrieval=%.3f routing=%.3f",
		best.candidate.minRelevance,
		best.candidate.notesBoost,
		best.candidate.affinity,
		best.retrievalScore,
		best.routingAccuracy,
	)

	// S4-07: lock = `DefaultScoringConfig` retrieval triplet. If this fails, re-run
	// the grid, update defaults in config.go (and context/scoring.toml if needed), and ADR-0004.
	const wantMin, wantNotes, wantAff = 0.05, 0.6, 1.2
	if best.candidate.minRelevance != wantMin || best.candidate.notesBoost != wantNotes || best.candidate.affinity != wantAff {
		t.Fatalf("grid winner (min_relevance=%.2f trust_boost.notes=%.1f agent_affinity=%.1f) != locked V1 defaults (%.2f / %.1f / %.1f)",
			best.candidate.minRelevance, best.candidate.notesBoost, best.candidate.affinity,
			wantMin, wantNotes, wantAff)
	}
}

func buildS2Grid() []tuningCandidate {
	mins := []float64{0.03, 0.05, 0.08}
	notes := []float64{0.5, 0.6, 0.7}
	aff := []float64{1.1, 1.2, 1.3}
	out := make([]tuningCandidate, 0, len(mins)*len(notes)*len(aff))
	for _, m := range mins {
		for _, n := range notes {
			for _, a := range aff {
				out = append(out, tuningCandidate{
					minRelevance: m,
					notesBoost:   n,
					affinity:     a,
				})
			}
		}
	}
	return out
}

func routingAccuracyForSuite(t *testing.T, suite *testws.GoldenSuite, cfg query.ScoringConfig) float64 {
	t.Helper()
	ws := suite.Build(t)
	if err := query.WriteScoringTOML(ws.Path(), cfg); err != nil {
		t.Fatalf("write scoring.toml for suite %s: %v", suite.Name, err)
	}
	hits, total := 0, 0
	for _, q := range suite.Queries {
		res := query.Run(ws.Path(), q.Query)
		if q.Matches(res) {
			hits++
		}
		total++
	}
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func retrievalTopKBlockScore(t *testing.T, suite *testws.GoldenSuite, cfg query.ScoringConfig) float64 {
	t.Helper()
	ws := suite.Build(t)
	if err := query.WriteScoringTOML(ws.Path(), cfg); err != nil {
		t.Fatalf("write scoring.toml for suite %s: %v", suite.Name, err)
	}

	// Sprint-2 scope: score only expect_blocks_in_topk assertions.
	hits, total := 0, 0
	for _, q := range suite.Queries {
		res := query.Run(ws.Path(), q.Query)
		for _, b := range q.ExpectBlocksInTopK {
			total++
			if blockInTopK(res.ContextBlocks, b.File, b.Section, b.K) {
				hits++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func blockInTopK(blocks []query.ContextBlock, file, section string, k int) bool {
	limit := len(blocks)
	if k > 0 && k < limit {
		limit = k
	}
	for i := 0; i < limit; i++ {
		cb := blocks[i]
		if cb.File == file && (section == "" || cb.Section == section) {
			return true
		}
	}
	return false
}

func betterTuningResult(a, b tuningResult) bool {
	if a.retrievalScore != b.retrievalScore {
		return a.retrievalScore > b.retrievalScore
	}
	// Deterministic tie-break: prefer candidates closer to ADR defaults.
	da := tuningDistanceFromDefaults(a.candidate)
	db := tuningDistanceFromDefaults(b.candidate)
	if da != db {
		return da < db
	}
	// Final stable tie-break.
	if a.candidate.minRelevance != b.candidate.minRelevance {
		return a.candidate.minRelevance < b.candidate.minRelevance
	}
	if a.candidate.notesBoost != b.candidate.notesBoost {
		return a.candidate.notesBoost < b.candidate.notesBoost
	}
	return a.candidate.affinity < b.candidate.affinity
}

func tuningDistanceFromDefaults(c tuningCandidate) float64 {
	// Defaults from ADR/ScoringConfig: min=0.05, notes=0.6, affinity=1.2
	return abs(c.minRelevance-0.05) + abs(c.notesBoost-0.6) + abs(c.affinity-1.2)
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

