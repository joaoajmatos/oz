package query

import (
	"fmt"
	"math"
	"sort"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/query/bm25"
)

// ProposalRetrieval is the result of a retrieval-only pass for concept proposal.
// Unlike Result it has no routing fields: no Agent, Confidence, Scope, or
// CandidateAgents. Used by oz context concept add (CCA-0).
type ProposalRetrieval struct {
	ContextBlocks        []ContextBlock   `json:"context_blocks,omitempty"`
	RelevantConcepts     []string         `json:"relevant_concepts,omitempty"`
	ImplementingPackages []string         `json:"implementing_packages,omitempty"`
	CodeEntryPoints      []CodeEntryPoint `json:"code_entry_points,omitempty"`
	Excluded             []string         `json:"excluded,omitempty"`
}

// RetrievalForProposal runs the retrieval pipeline without agent routing.
// It returns ranked context blocks, relevant concepts, implementing packages,
// and code entry points using the same BM25 scoring and trust boosts as
// oz context query — but skips agent scoring and routing entirely.
//
// Differences from RunWithOptions:
//   - No BM25F agent scoring or softmax routing.
//   - No affinity boost: all blocks get affinityBoost=1.0 (agentName="").
//   - ensureScopeSurvivor is a no-op (no agent scope).
//   - Code entry points are concept-reach only (no winning-agent scope).
//
// This is the retrieval entry point for oz context concept add (CCA-0-03).
func RetrievalForProposal(workspacePath, queryText string) (ProposalRetrieval, error) {
	g, err := ozcontext.LoadGraph(workspacePath)
	if err != nil {
		result, buildErr := ozcontext.Build(workspacePath)
		if buildErr != nil {
			return ProposalRetrieval{}, fmt.Errorf("load graph: %w", buildErr)
		}
		g = result.Graph
	}

	cfg := LoadConfig(workspacePath)
	terms := TokenizeQuery(queryText, cfg.UseBigrams)
	if len(terms) == 0 {
		return ProposalRetrieval{}, nil
	}

	// Proposal mode: empty agentName disables affinity boost and scope survivor.
	// See BuildContextBlocks for the full proposal-mode contract.
	blocks, excluded, _ := BuildContextBlocks(workspacePath, g, "", terms, cfg)

	return ProposalRetrieval{
		ContextBlocks:        blocks,
		RelevantConcepts:     loadRelevantConcepts(workspacePath, terms, cfg),
		ImplementingPackages: loadImplementingPackages(workspacePath, terms),
		CodeEntryPoints:      loadCodeEntryPointsForProposal(workspacePath, g, terms, cfg),
		Excluded:             excluded,
	}, nil
}

// loadCodeEntryPointsForProposal is the proposal-mode variant of loadCodeEntryPoints.
// It does not require a winning agent: only symbols reachable via reviewed semantic
// implements edges for query-relevant concepts are eligible (no scope-based inclusion).
func loadCodeEntryPointsForProposal(workspacePath string, g *graph.Graph, queryTerms []string, cfg ScoringConfig) []CodeEntryPoint {
	if g == nil || len(queryTerms) == 0 {
		return nil
	}
	dterms := SemanticRetrievalQueryTerms(queryTerms)
	if len(dterms) == 0 {
		return nil
	}
	packageConceptReach := eligiblePackagesByConcept(workspacePath, dterms, cfg)
	if len(packageConceptReach) == 0 {
		return nil
	}

	var nodes []graph.Node
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeCodeSymbol {
			continue
		}
		if _, ok := packageConceptReach[n.Package]; !ok {
			continue
		}
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		return nil
	}

	fields := codeSymbolRetrievalFields(cfg)
	docs := make([]bm25.FieldDoc, len(nodes))
	for i := range nodes {
		docs[i] = codeSymbolFieldDoc{m: codeSymbolFieldTokens(nodes[i], cfg.UseBigrams)}
	}
	avgLen := bm25.AvgFieldLengths(docs, fields)
	df := bm25.ComputeDF(docs)
	k1 := cfg.RetrievalK1
	if k1 <= 0 {
		k1 = 1.2
	}

	type scored struct {
		point           CodeEntryPoint
		rel             float64
		pkgConceptScore float64
		orig            int
	}
	scoredList := make([]scored, 0, len(nodes))
	for i, n := range nodes {
		pkgCS := packageConceptReach[n.Package]
		bm := bm25.BM25Score(
			dterms,
			codeSymbolFieldTokens(n, cfg.UseBigrams),
			fields,
			k1,
			avgLen,
			df,
			len(nodes),
		)
		affinity := 1.0
		if cfg.RetrievalAgentAffinity > 0 {
			affinity = cfg.RetrievalAgentAffinity
		}
		rel := bm * codeTrustBoostSymbols * affinity
		scoredList = append(scoredList, scored{
			point: CodeEntryPoint{
				File:      n.File,
				Symbol:    n.Name,
				Kind:      n.SymbolKind,
				Line:      n.Line,
				Package:   n.Package,
				Relevance: rel,
			},
			rel:             rel,
			pkgConceptScore: pkgCS,
			orig:            i,
		})
	}

	passed := scoredList[:0]
	for _, s := range scoredList {
		if s.rel >= cfg.RetrievalMinRelevance {
			passed = append(passed, s)
		}
	}
	if len(passed) == 0 {
		return nil
	}
	sort.SliceStable(passed, func(i, j int) bool {
		ri, rj := passed[i].rel, passed[j].rel
		d := math.Abs(ri - rj)
		if d > 1e-9 && d > codeEntryRelevanceTieMaxDelta {
			return ri > rj
		}
		if passed[i].pkgConceptScore != passed[j].pkgConceptScore {
			return passed[i].pkgConceptScore > passed[j].pkgConceptScore
		}
		if passed[i].point.Package != passed[j].point.Package {
			return passed[i].point.Package > passed[j].point.Package
		}
		if passed[i].point.File != passed[j].point.File {
			return passed[i].point.File < passed[j].point.File
		}
		if passed[i].point.Symbol != passed[j].point.Symbol {
			return passed[i].point.Symbol < passed[j].point.Symbol
		}
		if passed[i].point.Line != passed[j].point.Line {
			return passed[i].point.Line < passed[j].point.Line
		}
		return passed[i].orig < passed[j].orig
	})

	maxK := int(math.Round(cfg.RetrievalMaxCodeEntryPoints))
	if maxK < 1 {
		maxK = 1
	}
	if len(passed) > maxK {
		passed = passed[:maxK]
	}
	out := make([]CodeEntryPoint, len(passed))
	for i := range passed {
		out[i] = passed[i].point
	}
	return out
}
