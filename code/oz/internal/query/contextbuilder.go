package query

import (
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
	for _, s := range scored {
		if s.Relevance < retrievalMinRelevanceDefault {
			continue
		}
		blocks = append(blocks, ContextBlock{
			File:    s.Block.File,
			Section: s.Block.Section,
			Trust:   s.Block.Trust,
		})
		if len(blocks) == retrievalMaxBlocksDefault {
			break
		}
	}

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

const (
	retrievalMinRelevanceDefault = 0.05
	retrievalMaxBlocksDefault    = 12
)

func defaultRetrievalConfig(cfg ScoringConfig) contextretrieval.RetrievalConfig {
	return contextretrieval.RetrievalConfig{
		K1: cfg.K1,
		Fields: []bm25.BM25Field{
			{Name: "title", Weight: 2.0, B: cfg.BText},
			{Name: "path", Weight: 1.5, B: cfg.BPath},
			{Name: "body", Weight: 1.0, B: cfg.BText},
		},
		TrustBoost: map[string]float64{
			"high":   1.3,
			"medium": 1.0,
			"low":    0.6,
		},
		AgentAffinityBoost: 1.2,
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
