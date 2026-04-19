package query

import (
	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
)

// Options configures a query execution.
type Options struct {
	IncludeNotes bool
	RawMode      bool
}

// RunWithOptions executes a query against the oz workspace at workspacePath
// using the provided options. Loads or builds the structural graph, scores
// agents with BM25F, applies softmax, and returns a routing packet.
func RunWithOptions(workspacePath, queryText string, opts Options) Result {
	// 1. Load context graph (build if missing).
	g, err := ozcontext.LoadGraph(workspacePath)
	if err != nil {
		result, buildErr := ozcontext.Build(workspacePath)
		if buildErr != nil {
			return Result{Reason: "no_clear_owner"}
		}
		g = result.Graph
	}

	// 2. Load scoring config (falls back to defaults if scoring.toml absent).
	cfg := LoadConfig(workspacePath)
	cfg.IncludeNotes = opts.IncludeNotes

	// 3. Tokenize query.
	terms := Tokenize(queryText)
	if len(terms) == 0 {
		return Result{Reason: "no_clear_owner"}
	}

	// 4. Build agent documents from graph.
	docs := BuildAgentDocs(g.Nodes)
	if len(docs) == 0 {
		return Result{Reason: "no_clear_owner"}
	}

	// 5. Score agents.
	scores := ComputeBM25F(terms, docs, cfg)

	// 6. Route.
	route := Route(scores, cfg)
	if route.NoClearOwner {
		return Result{Reason: "no_clear_owner"}
	}

	// 7. Build context blocks and scope for the winning agent.
	blocks, excluded := BuildContextBlocks(g, route.Agent, cfg)
	scope := BuildScopeForAgent(g, route.Agent)

	// 8. Assemble routing packet.
	result := Result{
		Agent:         route.Agent,
		Confidence:    route.Confidence,
		Scope:         scope,
		ContextBlocks: blocks,
		Excluded:      excluded,
	}
	if len(route.Candidates) > 0 {
		result.CandidateAgents = route.Candidates
	}

	// 9. Enrich with semantic overlay concepts if present.
	result.RelevantConcepts = loadRelevantConcepts(workspacePath, route.Agent, g)

	return result
}

// loadRelevantConcepts reads context/semantic.json (if present) and returns
// concept names owned by agentName. Returns nil when no overlay exists.
func loadRelevantConcepts(workspacePath, agentName string, _ *graph.Graph) []string {
	// Semantic overlay is Sprint 5 — stub returns nil.
	return nil
}
