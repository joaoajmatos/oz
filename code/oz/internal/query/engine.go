package query

import (
	"math"
	"sort"
	"strings"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/query/bm25"
	"github.com/joaoajmatos/oz/internal/query/contextretrieval"
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

	// RetrievalScored is the full ranked retrieval list (for --raw). Empty when
	// routing returned no owner or query had no terms.
	RetrievalScored []contextretrieval.ScoredBlock
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

	blocks, excluded, scored := BuildContextBlocks(workspacePath, g, st.Route.Agent, st.Terms, st.Cfg)
	st.RetrievalScored = scored
	scope := BuildScopeForAgent(g, st.Route.Agent)
	st.Result = Result{
		Agent:         st.Route.Agent,
		Confidence:    st.Route.Confidence,
		Scope:         scope,
		ContextBlocks: blocks,
		Excluded:      excluded,
	}
	if len(blocks) == 0 {
		st.Result.Reason = "no_relevant_context"
	}
	if len(st.Route.Candidates) > 0 {
		st.Result.CandidateAgents = st.Route.Candidates
	}
	st.Result.RelevantConcepts = loadRelevantConcepts(workspacePath, st.Route.Agent, g)
	st.Result.ImplementingPackages = loadImplementingPackages(workspacePath, st.Terms)
	st.Result.CodeEntryPoints = loadCodeEntryPoints(workspacePath, g, st.Route.Agent, st.Terms, st.Cfg)
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

type conceptFieldDoc struct {
	m map[string][]string
}

func (d conceptFieldDoc) Fields() map[string][]string {
	return d.m
}

func conceptFieldTokens(c semantic.Concept, useBigrams bool) map[string][]string {
	return map[string][]string{
		"name":        TokenizeMulti(c.Name, useBigrams),
		"description": TokenizeMulti(c.Description, useBigrams),
	}
}

func conceptRetrievalFields(cfg ScoringConfig) []bm25.BM25Field {
	return []bm25.BM25Field{
		{Name: "name", Weight: cfg.RetrievalConceptWeightName, B: cfg.BText},
		{Name: "description", Weight: cfg.RetrievalConceptWeightDescription, B: cfg.BText},
	}
}

// loadImplementingPackages returns code packages connected through reviewed
// semantic implements edges from concepts ranked by query relevance.
func loadImplementingPackages(workspacePath string, queryTerms []string) []string {
	cfg := LoadConfig(workspacePath)
	o, err := semantic.Load(workspacePath)
	if err != nil || o == nil {
		return nil
	}
	if len(queryTerms) == 0 {
		return nil
	}
	conceptScores := scoreConcepts(o.Concepts, queryTerms, cfg, false)
	if len(conceptScores) == 0 {
		return nil
	}
	packageBest := make(map[string]float64)
	for _, e := range o.Edges {
		if e.Type != semantic.EdgeTypeImplements || !e.Reviewed {
			continue
		}
		score, ok := conceptScores[e.From]
		if !ok || score < cfg.RetrievalConceptMinRelevance {
			continue
		}
		pkg := packageFromImplementsTo(e.To)
		if pkg == "" {
			continue
		}
		if current, exists := packageBest[pkg]; !exists || score > current {
			packageBest[pkg] = score
		}
	}
	if len(packageBest) == 0 {
		return nil
	}
	type scoredPkg struct {
		pkg   string
		score float64
	}
	pkgs := make([]scoredPkg, 0, len(packageBest))
	for pkg, score := range packageBest {
		pkgs = append(pkgs, scoredPkg{pkg: pkg, score: score})
	}
	sort.SliceStable(pkgs, func(i, j int) bool {
		if pkgs[i].score != pkgs[j].score {
			return pkgs[i].score > pkgs[j].score
		}
		return pkgs[i].pkg < pkgs[j].pkg
	})
	maxPkgs := int(math.Round(cfg.RetrievalMaxImplementingPackages))
	if maxPkgs < 1 {
		maxPkgs = 1
	}
	if len(pkgs) > maxPkgs {
		pkgs = pkgs[:maxPkgs]
	}
	out := make([]string, len(pkgs))
	for i := range pkgs {
		out[i] = pkgs[i].pkg
	}
	return out
}

// codeTrustBoostSymbols is the PRD/ADR default trust multiplier for code-tier
// material (not yet a separate scoring.toml key).
const codeTrustBoostSymbols = 0.9

// codeSymbolFieldDoc adapts a code_symbol node to bm25.FieldDoc.
type codeSymbolFieldDoc struct {
	m map[string][]string
}

func (d codeSymbolFieldDoc) Fields() map[string][]string {
	return d.m
}

func codeSymbolFieldTokens(n graph.Node, useBigrams bool) map[string][]string {
	// title/path/kind line up with [retrieval.fields] and PRD field roles for symbols.
	return map[string][]string{
		"title": TokenizeMulti(n.Name, useBigrams),
		"path":  TokenizePathsMulti([]string{n.Package}, useBigrams),
		"kind":  TokenizeMulti(n.SymbolKind, useBigrams),
	}
}

func codeSymbolRetrievalFields(cfg ScoringConfig) []bm25.BM25Field {
	return []bm25.BM25Field{
		{Name: "title", Weight: cfg.RetrievalWeightTitle, B: cfg.BText},
		{Name: "path", Weight: cfg.RetrievalWeightPath, B: cfg.BPath},
		{Name: "kind", Weight: cfg.RetrievalWeightKind, B: cfg.BText},
	}
}

// loadCodeEntryPoints returns eligible symbol-level entry points for code-level
// queries. Eligibility (Sprint 3 Story 3):
//   - symbol file is under winning-agent scope, OR
//   - symbol package is reachable from reviewed semantic implements edges for
//     concepts relevant to the query.
//
// Rank/cap (Sprint 3 Story 4): BM25 over (name, package, kind), × code trust
// × agent affinity, threshold at retrieval.min_relevance, cap at
// retrieval.max_code_entry_points.
func loadCodeEntryPoints(workspacePath string, g *graph.Graph, winningAgent string, queryTerms []string, cfg ScoringConfig) []CodeEntryPoint {
	if g == nil || winningAgent == "" || len(queryTerms) == 0 {
		return nil
	}
	scope := BuildScopeForAgent(g, winningAgent)
	eligiblePackages := eligiblePackagesByConcept(workspacePath, queryTerms, cfg)

	var nodes []graph.Node
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeCodeSymbol {
			continue
		}
		inScope := fileInAnyScope(n.File, scope)
		byConcept := eligiblePackages[n.Package]
		if !inScope && !byConcept {
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
	nDocs := len(nodes)
	k1 := cfg.RetrievalK1

	if k1 <= 0 {
		k1 = 1.2
	}

	type scored struct {
		point CodeEntryPoint
		rel   float64
		orig  int
	}
	scoredList := make([]scored, 0, len(nodes))
	for i, n := range nodes {
		inScope := fileInAnyScope(n.File, scope)
		byConcept := eligiblePackages[n.Package]
		bm := bm25.BM25Score(
			queryTerms,
			codeSymbolFieldTokens(n, cfg.UseBigrams),
			fields,
			k1,
			avgLen,
			df,
			nDocs,
		)
		affinity := 1.0
		if inScope || byConcept {
			if cfg.RetrievalAgentAffinity > 0 {
				affinity = cfg.RetrievalAgentAffinity
			}
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
			rel:  rel,
			orig: i,
		})
	}

	// Drop below min_relevance, then sort by relevance DESC with deterministic ties.
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
		if passed[i].rel != passed[j].rel {
			return passed[i].rel > passed[j].rel
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

func eligiblePackagesByConcept(workspacePath string, queryTerms []string, cfg ScoringConfig) map[string]bool {
	if len(queryTerms) == 0 {
		return nil
	}
	o, err := semantic.Load(workspacePath)
	if err != nil || o == nil {
		return nil
	}
	conceptScores := scoreConcepts(o.Concepts, queryTerms, cfg, cfg.UseBigrams)
	relevantConcepts := make(map[string]bool)
	for id, score := range conceptScores {
		if score >= cfg.RetrievalConceptMinRelevance {
			relevantConcepts[id] = true
		}
	}
	if len(relevantConcepts) == 0 {
		return nil
	}
	out := make(map[string]bool)
	for _, e := range o.Edges {
		if e.Type != semantic.EdgeTypeImplements || !e.Reviewed {
			continue
		}
		if !relevantConcepts[e.From] {
			continue
		}
		if pkg := packageFromImplementsTo(e.To); pkg != "" {
			out[pkg] = true
		}
	}
	return out
}

func scoreConcepts(concepts []semantic.Concept, queryTerms []string, cfg ScoringConfig, useBigrams bool) map[string]float64 {
	if len(concepts) == 0 || len(queryTerms) == 0 {
		return nil
	}
	fields := conceptRetrievalFields(cfg)
	docs := make([]bm25.FieldDoc, len(concepts))
	for i := range concepts {
		docs[i] = conceptFieldDoc{m: conceptFieldTokens(concepts[i], useBigrams)}
	}
	avgLen := bm25.AvgFieldLengths(docs, fields)
	df := bm25.ComputeDF(docs)
	k1 := cfg.RetrievalK1
	if k1 <= 0 {
		k1 = 1.2
	}
	out := make(map[string]float64, len(concepts))
	for i, c := range concepts {
		score := bm25.BM25Score(queryTerms, docs[i].Fields(), fields, k1, avgLen, df, len(concepts))
		out[c.ID] = score
	}
	return out
}

func packageFromImplementsTo(to string) string {
	if strings.HasPrefix(to, "code_package:") {
		return strings.TrimPrefix(to, "code_package:")
	}
	return to
}
