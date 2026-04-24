package query

import (
	"math"
	"path/filepath"
	"strings"

	"github.com/joaoajmatos/oz/internal/convention"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/query/bm25"
	"github.com/joaoajmatos/oz/internal/query/contextretrieval"
)

// BuildContextBlocks returns the ordered context blocks for the winning agent.
//
// Selection strategy:
//  1. Build retrieval corpus from eligible node types.
//  2. Score each candidate against query terms (BM25 * trust * affinity).
//  3. Drop scores below min_relevance and cap at max_blocks.
//
// Blocks are ordered by relevance with deterministic tie-breakers.
// Notes are excluded unless cfg.IncludeNotes is true.
// Notes are always listed in Excluded when not included.
func BuildContextBlocks(workspacePath string, g *graph.Graph, agentName string, queryTerms []string, cfg ScoringConfig) (blocks []ContextBlock, excluded []string) {
	candidates := buildRetrievalCandidates(workspacePath, g, cfg)
	retrievalCfg := defaultRetrievalConfig(cfg)
	scored := contextretrieval.Score(queryTerms, candidates, retrievalCfg, agentName)
	maxBlocks := int(math.Round(cfg.RetrievalMaxBlocks))
	if maxBlocks < 1 {
		maxBlocks = 1
	}
	var passed []contextretrieval.ScoredBlock
	for _, s := range scored {
		if s.Relevance < cfg.RetrievalMinRelevance {
			continue
		}
		passed = append(passed, s)
		if len(blocks) < maxBlocks {
			blocks = append(blocks, ContextBlock{
				File:      s.Block.File,
				Section:   s.Block.Section,
				Trust:     s.Block.Trust,
				Relevance: s.Relevance,
			})
		}
	}
	ensureScopeSurvivor(&blocks, passed, BuildScopeForAgent(g, agentName))

	// Excluded: note paths (when not included).
	if !cfg.IncludeNotes {
		hasNotes := false
		for _, n := range g.Nodes {
			if n.Type == graph.NodeTypeNote {
				hasNotes = true
				break
			}
		}
		if hasNotes {
			excluded = []string{"notes/"}
		}
	}

	return blocks, excluded
}

func ensureScopeSurvivor(blocks *[]ContextBlock, passed []contextretrieval.ScoredBlock, scope []string) {
	if len(*blocks) == 0 || len(scope) == 0 {
		return
	}
	hasScopeCandidate := false
	for _, s := range passed {
		if fileInAnyScope(s.Block.File, scope) {
			hasScopeCandidate = true
			break
		}
	}
	if !hasScopeCandidate {
		return
	}
	for _, b := range *blocks {
		if fileInAnyScope(b.File, scope) {
			return
		}
	}
	// Candidate exists but truncation removed all scope blocks; force the best
	// threshold-clearing scope block to survive by replacing the last slot.
	for _, s := range passed {
		if !fileInAnyScope(s.Block.File, scope) {
			continue
		}
		(*blocks)[len(*blocks)-1] = ContextBlock{
			File:      s.Block.File,
			Section:   s.Block.Section,
			Trust:     s.Block.Trust,
			Relevance: s.Relevance,
		}
		return
	}
}

// BuildScopeForAgent returns the scope paths for the winning agent node.
func BuildScopeForAgent(g *graph.Graph, agentName string) []string {
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent && n.Name == agentName {
			return n.Scope
		}
	}
	return nil
}

// ---- helpers ----------------------------------------------------------------

// tierToTrust maps graph tier strings to context block trust strings.
func tierToTrust(tier convention.Tier) string {
	switch tier {
	case convention.TierSpecs:
		return "high"
	case convention.TierDocs, convention.TierContext:
		return "medium"
	case convention.TierNotes:
		return "low"
	default:
		return "medium"
	}
}

func defaultRetrievalConfig(cfg ScoringConfig) contextretrieval.RetrievalConfig {
	return contextretrieval.RetrievalConfig{
		K1: cfg.RetrievalK1,
		Fields: []bm25.BM25Field{
			{Name: "title", Weight: cfg.RetrievalWeightTitle, B: cfg.BText},
			{Name: "path", Weight: cfg.RetrievalWeightPath, B: cfg.BPath},
			{Name: "body", Weight: cfg.RetrievalWeightBody, B: cfg.BText},
		},
		TrustBoost: map[string]float64{
			"specs":   cfg.RetrievalTrustBoostSpecs,
			"docs":    cfg.RetrievalTrustBoostDocs,
			"context": cfg.RetrievalTrustBoostContext,
			"notes":   cfg.RetrievalTrustBoostNotes,
			"high":    cfg.RetrievalTrustBoostSpecs,
			"medium":  cfg.RetrievalTrustBoostDocs,
			"low":     cfg.RetrievalTrustBoostNotes,
		},
		AgentAffinityBoost: cfg.RetrievalAgentAffinity,
	}
}

func buildRetrievalCandidates(workspacePath string, g *graph.Graph, cfg ScoringConfig) []contextretrieval.Block {
	readersByNode := buildReadersByNode(g)
	scopesByAgent := buildScopesByAgent(g)
	out := make([]contextretrieval.Block, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		if !isRetrievalNodeType(n.Type, cfg.IncludeNotes) {
			continue
		}
		connected := make(map[string]bool)
		for agent := range readersByNode[n.ID] {
			connected[agent] = true
		}
		for agent, scopes := range scopesByAgent {
			for _, scope := range scopes {
				if pathInScope(n.File, scope) {
					connected[agent] = true
					break
				}
			}
		}
		title := strings.TrimSpace(n.Section)
		if title == "" {
			title = strings.TrimSpace(n.Name)
		}
		out = append(out, contextretrieval.Block{
			File:    n.File,
			Section: n.Section,
			Trust:   tierToTrust(n.Tier),
			Tier:    string(n.Tier),
			TokenFields: map[string][]string{
				"title": TokenizeMulti(title, cfg.UseBigrams),
				"path":  TokenizePathsMulti([]string{n.File}, cfg.UseBigrams),
				"body":  loadRetrievalBodyTokens(workspacePath, g.ContentHash, n, cfg.UseBigrams),
			},
			ConnectedAgents: connected,
		})
	}
	return out
}

func isRetrievalNodeType(nodeType string, includeNotes bool) bool {
	switch nodeType {
	case graph.NodeTypeSpecSection, graph.NodeTypeDecision, graph.NodeTypeDoc, graph.NodeTypeContextSnapshot:
		return true
	case graph.NodeTypeNote:
		return includeNotes
	default:
		return false
	}
}

func buildReadersByNode(g *graph.Graph) map[string]map[string]bool {
	out := make(map[string]map[string]bool)
	for _, e := range g.Edges {
		if e.Type != graph.EdgeTypeReads {
			continue
		}
		agentName := strings.TrimPrefix(e.From, "agent:")
		if _, ok := out[e.To]; !ok {
			out[e.To] = make(map[string]bool)
		}
		out[e.To][agentName] = true
	}
	return out
}

func buildScopesByAgent(g *graph.Graph) map[string][]string {
	out := make(map[string][]string)
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeAgent {
			out[n.Name] = append([]string(nil), n.Scope...)
		}
	}
	return out
}

func pathInScope(path, scope string) bool {
	if scope == "" {
		return false
	}
	prefix := strings.TrimSuffix(scope, "**")
	prefix = strings.TrimSuffix(prefix, "*")
	if strings.HasSuffix(prefix, "/") {
		return strings.HasPrefix(path, prefix)
	}
	matched, err := filepath.Match(scope, path)
	if err == nil && matched {
		return true
	}
	return strings.HasPrefix(path, prefix)
}

func fileInAnyScope(path string, scopes []string) bool {
	for _, s := range scopes {
		if pathInScope(path, s) {
			return true
		}
	}
	return false
}
