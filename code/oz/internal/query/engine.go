package query

import (
	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
	"github.com/oz-tools/oz/internal/semantic"
)

// Options configures a query execution.
type Options struct {
	IncludeNotes bool
	RawMode      bool
}

// routingState holds intermediate values from a query run (for --raw debug).
type routingState struct {
	G      *graph.Graph
	Cfg    ScoringConfig
	Terms  []string
	Docs   []AgentDoc
	Scores []Score
	Conf   []float64
	Route  RouteResult
	Result Result
}

// runRouting executes load → tokenize → score → route → assemble Result.
func runRouting(workspacePath, queryText string, opts Options) routingState {
	var st routingState

	g, err := ozcontext.LoadGraph(workspacePath)
	if err != nil {
		result, buildErr := ozcontext.Build(workspacePath)
		if buildErr != nil {
			st.Result = Result{Reason: "no_clear_owner"}
			return st
		}
		g = result.Graph
	}
	st.G = g

	st.Cfg = LoadConfig(workspacePath)
	st.Cfg.IncludeNotes = opts.IncludeNotes

	st.Terms = TokenizeQuery(queryText, st.Cfg.UseBigrams)
	if len(st.Terms) == 0 {
		st.Result = Result{Reason: "no_clear_owner"}
		return st
	}

	st.Docs = BuildAgentDocs(g.Nodes, st.Cfg)
	if len(st.Docs) == 0 {
		st.Result = Result{Reason: "no_clear_owner"}
		return st
	}

	st.Scores = ComputeBM25F(st.Terms, st.Docs, st.Cfg)
	if len(st.Scores) > 0 {
		st.Conf = Softmax(st.Scores, st.Cfg.Temperature)
	}

	st.Route = Route(st.Scores, st.Cfg)
	if st.Route.NoClearOwner {
		st.Result = Result{Reason: "no_clear_owner"}
		return st
	}

	blocks, excluded := BuildContextBlocks(g, st.Route.Agent, st.Cfg)
	scope := BuildScopeForAgent(g, st.Route.Agent)
	st.Result = Result{
		Agent:         st.Route.Agent,
		Confidence:    st.Route.Confidence,
		Scope:         scope,
		ContextBlocks: blocks,
		Excluded:      excluded,
	}
	if len(st.Route.Candidates) > 0 {
		st.Result.CandidateAgents = st.Route.Candidates
	}
	st.Result.RelevantConcepts = loadRelevantConcepts(workspacePath, st.Route.Agent, g)
	return st
}

// RunWithOptions executes a query against the oz workspace at workspacePath
// using the provided options. Loads or builds the structural graph, scores
// agents with BM25F, applies softmax, and returns a routing packet.
func RunWithOptions(workspacePath, queryText string, opts Options) Result {
	return runRouting(workspacePath, queryText, opts).Result
}

// Run executes a query against the oz workspace at workspacePath.
func Run(workspacePath, queryText string) Result {
	return RunWithOptions(workspacePath, queryText, Options{})
}

// loadRelevantConcepts reads context/semantic.json (if present) and returns
// concept names owned by agentName. Returns nil when no overlay exists.
func loadRelevantConcepts(workspacePath, agentName string, _ *graph.Graph) []string {
	o, err := semantic.Load(workspacePath)
	if err != nil || o == nil {
		return nil
	}
	return semantic.ConceptsForAgent(o, agentName)
}
