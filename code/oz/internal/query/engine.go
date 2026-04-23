package query

import (
	"sort"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/semantic"
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
	// File can enable notes; --include-notes also enables (union).
	st.Cfg.IncludeNotes = st.Cfg.IncludeNotes || opts.IncludeNotes

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
	st.Result.ImplementingPackages = loadImplementingPackages(workspacePath, st.Terms)
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

// ListAgents returns a workspace overview: all agents as CandidateAgents with
// confidence 0.0. Used when no query text is provided (e.g. sessionStart hooks).
func ListAgents(workspacePath string) Result {
	g, err := ozcontext.LoadGraph(workspacePath)
	if err != nil {
		result, buildErr := ozcontext.Build(workspacePath)
		if buildErr != nil {
			return Result{Reason: "no_clear_owner"}
		}
		g = result.Graph
	}

	cfg := LoadConfig(workspacePath)
	docs := BuildAgentDocs(g.Nodes, cfg)
	if len(docs) == 0 {
		return Result{Reason: "no_clear_owner"}
	}

	candidates := make([]CandidateAgent, len(docs))
	for i, d := range docs {
		candidates[i] = CandidateAgent{Name: d.Name, Confidence: 0.0}
	}
	return Result{
		Reason:          "workspace_overview",
		CandidateAgents: candidates,
	}
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

// loadImplementingPackages returns the import paths of code_package nodes that
// implement concepts relevant to the query terms, via reviewed implements edges.
// Relevance is determined by token overlap between query terms and concept
// name+description — so "how is drift detection implemented?" returns audit/drift
// rather than packages owned by the routed agent.
func loadImplementingPackages(workspacePath string, queryTerms []string) []string {
	o, err := semantic.Load(workspacePath)
	if err != nil || o == nil {
		return nil
	}
	if len(queryTerms) == 0 {
		return nil
	}
	termSet := make(map[string]struct{}, len(queryTerms))
	for _, t := range queryTerms {
		termSet[t] = struct{}{}
	}

	// Find concepts whose name/description shares tokens with the query.
	relevant := make(map[string]struct{})
	for _, c := range o.Concepts {
		conceptTokens := TokenizeMulti(c.Name+" "+c.Description, false)
		for _, t := range conceptTokens {
			if _, ok := termSet[t]; ok {
				relevant[c.ID] = struct{}{}
				break
			}
		}
	}
	if len(relevant) == 0 {
		return nil
	}

	// Collect packages for relevant concepts via reviewed implements edges.
	seen := make(map[string]struct{})
	var pkgs []string
	for _, e := range o.Edges {
		if e.Type != semantic.EdgeTypeImplements || !e.Reviewed {
			continue
		}
		if _, ok := relevant[e.From]; !ok {
			continue
		}
		if _, ok := seen[e.To]; !ok {
			seen[e.To] = struct{}{}
			pkgs = append(pkgs, e.To)
		}
	}
	if len(pkgs) == 0 {
		return nil
	}
	sort.Strings(pkgs)
	return pkgs
}
