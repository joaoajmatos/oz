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

// BuildContextBlocks returns ranked, thresholded context blocks, any excluded
// path prefixes, and the full retrieval score list (for --raw debug).
//
// Selection strategy:
//  1. Build retrieval corpus from eligible node types.
//  2. Score each candidate against query terms (BM25 * trust * affinity).
//  3. Drop scores below min_relevance and cap at max_blocks.
//
// Blocks are ordered by relevance with deterministic tie-breakers.
// When cfg.IncludeNotes is false, note nodes are omitted from the retrieval corpus
// and "notes/" is added to Excluded when the graph has notes.
func BuildContextBlocks(workspacePath string, g *graph.Graph, agentName string, queryTerms []string, cfg ScoringConfig) (blocks []ContextBlock, excluded []string, scored []contextretrieval.ScoredBlock) {
	candidates := buildRetrievalCandidates(workspacePath, g, cfg)
	retrievalCfg := defaultRetrievalConfig(cfg)
	scored = contextretrieval.Score(queryTerms, candidates, retrievalCfg, agentName)
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
	ensureCodePackageSurvivor(&blocks, passed, queryTerms, maxBlocks)

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

	return blocks, excluded, scored
}

// querySuggestsCodeContext is true when stems read like an implementation
// or indexing task (so surfacing a code/ file may help). Deliberately excludes
// generic "implement" so "how is drift detection implemented" remains ranked
// by the normal spec-heavy retrieval mix.
func querySuggestsCodeContext(terms []string) bool {
	for _, t := range terms {
		switch t {
		case "index", "symbol", "packag", "graph", "load", "build", "api", "func", "type", "test", "debug", "rout", "migrat":
			return true
		}
	}
	return false
}

// ensureCodePackageSurvivor only when the ranked slice has no code/ entry but
// some code_package cleared the relevance floor — typically because specs
// (high trust) pushed all Go files out of the first max_blocks. If the best
// code block beats the lowest-ranked kept block, swap the last slot; if there
// is still capacity, append instead. Never downgrades a higher-relevance tail.
func ensureCodePackageSurvivor(blocks *[]ContextBlock, passed []contextretrieval.ScoredBlock, queryTerms []string, maxBlocks int) {
	if len(passed) == 0 {
		return
	}
	if !querySuggestsCodeContext(queryTerms) {
		return
	}
	for _, b := range *blocks {
		if strings.HasPrefix(b.File, "code/") {
			return
		}
	}
	var best *contextretrieval.ScoredBlock
	for i := range passed {
		if !strings.HasPrefix(passed[i].Block.File, "code/") {
			continue
		}
		if best == nil || passed[i].Relevance > best.Relevance {
			best = &passed[i]
		}
	}
	if best == nil {
		return
	}
	add := ContextBlock{
		File:      best.Block.File,
		Section:   best.Block.Section,
		Trust:     best.Block.Trust,
		Relevance: best.Relevance,
	}
	if len(*blocks) < maxBlocks {
		*blocks = append(*blocks, add)
		return
	}
	last := len(*blocks) - 1
	if best.Relevance <= (*blocks)[last].Relevance {
		return
	}
	(*blocks)[last] = add
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

// retrievalTitleAndPathParts returns a display title and path strings to
// tokenize for BM25. code_package uses the import path in the title and
// includes both file path and import path so segments like "index" match
// .../graph/index in addition to the on-disk path.
func retrievalTitleAndPathParts(n graph.Node) (title string, pathParts []string) {
	if n.Type == graph.NodeTypeCodePackage {
		if n.Package != "" {
			title = n.Package
		} else {
			title = strings.TrimSpace(n.Name)
		}
		if n.File != "" {
			pathParts = append(pathParts, n.File)
		}
		if n.Package != "" {
			pathParts = append(pathParts, n.Package)
		}
		if len(pathParts) == 0 && n.File != "" {
			pathParts = []string{n.File}
		}
		return title, pathParts
	}
	title = strings.TrimSpace(n.Section)
	if title == "" {
		title = strings.TrimSpace(n.Name)
	}
	if n.File != "" {
		pathParts = []string{n.File}
	}
	return title, pathParts
}

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
				if n.File != "" && pathInScope(n.File, scope) {
					connected[agent] = true
					break
				}
			}
		}
		title, pathParts := retrievalTitleAndPathParts(n)
		section := n.Section
		if n.Type == graph.NodeTypeCodePackage && n.Package != "" {
			section = n.Package
		}
		out = append(out, contextretrieval.Block{
			File:    n.File,
			Section: section,
			Trust:   tierToTrust(n.Tier),
			Tier:    string(n.Tier),
			TokenFields: map[string][]string{
				"title": TokenizeMulti(title, cfg.UseBigrams),
				"path":  TokenizePathsMulti(pathParts, cfg.UseBigrams),
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
	case graph.NodeTypeCodePackage:
		// One block per Go package: path + import path + package doc, so
		// implementation queries (e.g. "indexing") can surface code/ in context_blocks.
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
