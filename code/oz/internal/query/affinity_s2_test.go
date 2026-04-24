package query_test

import (
	"strings"
	"testing"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/testws"
)

// TestRetrievalAffinityDoesNotDominateS2 verifies Sprint-2 affinity behavior on
// 03_large by selecting a query whose winning agent has the widest read-chain,
// then checking top-10 context blocks still prioritize query-matching content.
func TestRetrievalAffinityDoesNotDominateS2(t *testing.T) {
	suites, err := testws.LoadGoldenSuites(t, "testdata/golden")
	if err != nil {
		t.Fatalf("load golden suites: %v", err)
	}
	var suite03 *testws.GoldenSuite
	for _, s := range suites {
		if s.Name == "03_large" {
			suite03 = s
			break
		}
	}
	if suite03 == nil {
		t.Fatal("03_large suite not found")
	}

	ws := suite03.Build(t)
	g, err := ozcontext.LoadGraph(ws.Path())
	if err != nil {
		built, buildErr := ozcontext.Build(ws.Path())
		if buildErr != nil {
			t.Fatalf("build graph for 03_large: %v", buildErr)
		}
		g = built.Graph
	}

	baseCfg := query.DefaultScoringConfig()
	offCfg := query.DefaultScoringConfig()
	offCfg.RetrievalAgentAffinity = 1.0

	// Find a query where affinity-off still surfaces lexical matches, and among
	// those choose the one with the widest winning-agent read-chain.
	var chosenQ string
	var chosenOn query.Result
	var chosenOff query.Result
	maxReadChain := -1
	for _, q := range suite03.Queries {
		onRes := runQueryWithConfig(t, ws.Path(), baseCfg, q.Query)
		if onRes.Agent == "" {
			continue
		}
		offRes := runQueryWithConfig(t, ws.Path(), offCfg, q.Query)
		if firstStrongRank(offRes.ContextBlocks, query.TokenizeQuery(q.Query, false), 10) < 0 {
			continue
		}
		rc := readChainLen(g, onRes.Agent)
		if rc > maxReadChain {
			maxReadChain = rc
			chosenQ = q.Query
			chosenOn = onRes
			chosenOff = offRes
		}
	}
	if chosenQ == "" {
		t.Fatal("no suitable 03_large query found where affinity-off has lexical-match evidence")
	}

	terms := query.TokenizeQuery(chosenQ, false)
	onBlocks := chosenOn.ContextBlocks
	offBlocks := chosenOff.ContextBlocks
	if len(onBlocks) == 0 || len(offBlocks) == 0 {
		t.Fatalf("chosen query returned no context blocks: %q", chosenQ)
	}
	limit := len(onBlocks)
	if limit > 10 {
		limit = 10
	}

	firstStrongOn := firstStrongRank(onBlocks, terms, limit)
	firstStrongOff := firstStrongRank(offBlocks, terms, limit)

	t.Logf("S2-09 chosen query: %q (winner=%s, readchain_len=%d)", chosenQ, chosenOn.Agent, maxReadChain)
	for i := 0; i < limit; i++ {
		cb := onBlocks[i]
		overlap := lexicalOverlap(terms, cb.File+" "+cb.Section)
		matchKind := "weak"
		if overlap > 0 {
			matchKind = "strong"
		}
		t.Logf("affinity-on top[%d] %s#%s trust=%s overlap=%d (%s)", i+1, cb.File, cb.Section, cb.Trust, overlap, matchKind)
	}
	t.Logf("first strong rank (on=%d off=%d)", firstStrongOn+1, firstStrongOff+1)

	if firstStrongOn < 0 {
		t.Fatalf("no strong lexical-match block in affinity-on top-%d for query %q", limit, chosenQ)
	}
	if firstStrongOff >= 0 && firstStrongOn > firstStrongOff {
		t.Fatalf(
			"affinity dominance suspected: first strong-match rank worsened with affinity (on=%d off=%d) for query %q",
			firstStrongOn+1, firstStrongOff+1, chosenQ,
		)
	}
}

func runQueryWithConfig(t *testing.T, workspacePath string, cfg query.ScoringConfig, q string) query.Result {
	t.Helper()
	if err := query.WriteScoringTOML(workspacePath, cfg); err != nil {
		t.Fatalf("write scoring.toml: %v", err)
	}
	return query.Run(workspacePath, q)
}

func firstStrongRank(blocks []query.ContextBlock, terms []string, limit int) int {
	if limit > len(blocks) {
		limit = len(blocks)
	}
	for i := 0; i < limit; i++ {
		cb := blocks[i]
		if lexicalOverlap(terms, cb.File+" "+cb.Section) > 0 {
			return i
		}
	}
	return -1
}

func readChainLen(g *graph.Graph, agentName string) int {
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent && n.Name == agentName {
			return len(n.ReadChain)
		}
	}
	return 0
}

func lexicalOverlap(queryTerms []string, text string) int {
	if len(queryTerms) == 0 || strings.TrimSpace(text) == "" {
		return 0
	}
	docTerms := query.TokenizeMulti(text, false)
	seen := make(map[string]bool, len(docTerms))
	for _, t := range docTerms {
		seen[t] = true
	}
	count := 0
	for _, q := range queryTerms {
		if seen[q] {
			count++
		}
	}
	return count
}

