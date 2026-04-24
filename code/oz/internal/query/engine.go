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
	st.Result.RelevantConcepts = loadRelevantConcepts(workspacePath, st.Terms, st.Cfg)
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

// loadRelevantConcepts returns up to [ScoringConfig.RetrievalMaxRelevantConcepts]
// reviewed concept **names** from the semantic overlay, ranked by the same
// BM25 concept scorer used for [loadImplementingPackages], filtered by
// [effectiveConceptRelevanceThreshold], and sorted by score DESC then name ASC.
// Returns nil when there is no overlay, no query terms, or no passing concepts.
func loadRelevantConcepts(workspacePath string, queryTerms []string, cfg ScoringConfig) []string {
	o, err := semantic.Load(workspacePath)
	if err != nil || o == nil || len(queryTerms) == 0 {
		return nil
	}
	terms := SemanticRetrievalQueryTerms(queryTerms)
	if len(terms) == 0 {
		return nil
	}
	var reviewed []semantic.Concept
	for i := range o.Concepts {
		if o.Concepts[i].Reviewed {
			reviewed = append(reviewed, o.Concepts[i])
		}
	}
	if len(reviewed) == 0 {
		return nil
	}
	scores := scoreConcepts(reviewed, terms, cfg, cfg.UseBigrams)
	threshold := effectiveConceptRelevanceThreshold(scores, cfg)
	type item struct {
		name  string
		score float64
	}
	var items []item
	for _, c := range reviewed {
		s, ok := scores[c.ID]
		if !ok || s < threshold {
			continue
		}
		items = append(items, item{name: c.Name, score: s})
	}
	if len(items) == 0 {
		return nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].name < items[j].name
	})
	maxK := int(math.Round(cfg.RetrievalMaxRelevantConcepts))
	if maxK < 1 {
		maxK = 1
	}
	if len(items) > maxK {
		items = items[:maxK]
	}
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].name
	}
	return out
}

type conceptFieldDoc struct {
	m map[string][]string
}

func (d conceptFieldDoc) Fields() map[string][]string {
	return d.m
}

func conceptFieldTokens(c semantic.Concept, useBigrams bool) map[string][]string {
	return map[string][]string{
		"name":         TokenizeMulti(c.Name, useBigrams),
		"description":  TokenizeMulti(c.Description, useBigrams),
		"source_paths": TokenizePathsMulti(c.SourceFiles, useBigrams),
	}
}

func conceptRetrievalFields(cfg ScoringConfig) []bm25.BM25Field {
	wSrc := cfg.RetrievalConceptWeightSourceFiles
	if wSrc <= 0 {
		wSrc = 1.0
	}
	return []bm25.BM25Field{
		{Name: "name", Weight: cfg.RetrievalConceptWeightName, B: cfg.BText},
		{Name: "description", Weight: cfg.RetrievalConceptWeightDescription, B: cfg.BText},
		{Name: "source_paths", Weight: wSrc, B: cfg.BPath},
	}
}

// loadImplementingPackages returns code packages connected through reviewed
// semantic implements edges from concepts ranked by query relevance.
// effectiveConceptRelevanceThreshold is the score floor for keeping a concept
// when walking reviewed implements edges. It is the maximum of the absolute
// floor (retrieval.concept_min_relevance) and, when
// retrieval.concept_min_fraction_of_top > 0, (best concept score) × that
// fraction. The relative term drops weak secondary matches (e.g. a concept
// that only matches a generic query stem).
func effectiveConceptRelevanceThreshold(scores map[string]float64, cfg ScoringConfig) float64 {
	eff := cfg.RetrievalConceptMinRelevance
	if cfg.RetrievalConceptMinFractionOfTop <= 0 {
		return eff
	}
	var maxS float64
	for _, s := range scores {
		if s > maxS {
			maxS = s
		}
	}
	if maxS <= 0 {
		return eff
	}
	rel := maxS * cfg.RetrievalConceptMinFractionOfTop
	if rel > eff {
		return rel
	}
	return eff
}

// packageBestConceptReachScores returns, for each code package import path, the
// maximum query→concept BM25 score among concepts that (a) pass the
// effective relevance floor and (b) have a reviewed implements edge to that
// package. Matches loadImplementingPackages ranking input; conceptUseBigrams
// mirrors scoreConcepts (loadImplementingPackages uses false; code entry
// points use cfg.UseBigrams to stay aligned with previous eligiblePackages
// behavior).
func packageBestConceptReachScores(workspacePath string, queryTerms []string, cfg ScoringConfig, conceptUseBigrams bool) map[string]float64 {
	if len(queryTerms) == 0 {
		return nil
	}
	q := SemanticRetrievalQueryTerms(queryTerms)
	if len(q) == 0 {
		return nil
	}
	o, err := semantic.Load(workspacePath)
	if err != nil || o == nil {
		return nil
	}
	conceptScores := scoreConcepts(o.Concepts, q, cfg, conceptUseBigrams)
	if len(conceptScores) == 0 {
		return nil
	}
	threshold := effectiveConceptRelevanceThreshold(conceptScores, cfg)
	packageBest := make(map[string]float64)
	for _, e := range o.Edges {
		if e.Type != semantic.EdgeTypeImplements || !e.Reviewed {
			continue
		}
		score, ok := conceptScores[e.From]
		if !ok || score < threshold {
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
	return packageBest
}

func loadImplementingPackages(workspacePath string, queryTerms []string) []string {
	cfg := LoadConfig(workspacePath)
	packageBest := packageBestConceptReachScores(workspacePath, queryTerms, cfg, false)
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

// codeEntryRelevanceTieMaxDelta: when two symbols' raw relevance scores are
// within this absolute gap, the engine treats the first sort key as tied and
// uses pkgConceptScore, then import-path specificity, then file/symbol order.
// This is what lets .../drift and .../drift/specscan (often ~0.1 apart on path
// length) be ordered by the semantic package list instead of raw BM25 only.
const codeEntryRelevanceTieMaxDelta = 0.12

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
	dterms := SemanticRetrievalQueryTerms(queryTerms)
	if len(dterms) == 0 {
		return nil
	}
	scope := BuildScopeForAgent(g, winningAgent)
	packageConceptReach := eligiblePackagesByConcept(workspacePath, dterms, cfg)

	var nodes []graph.Node
	for _, n := range g.Nodes {
		if n.Type != graph.NodeTypeCodeSymbol {
			continue
		}
		inScope := fileInAnyScope(n.File, scope)
		_, byConcept := packageConceptReach[n.Package]
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
		point           CodeEntryPoint
		rel             float64
		pkgConceptScore float64
		orig            int
	}
	scoredList := make([]scored, 0, len(nodes))
	for i, n := range nodes {
		inScope := fileInAnyScope(n.File, scope)
		pkgCS, inConceptMap := packageConceptReach[n.Package]
		byConcept := inConceptMap
		bm := bm25.BM25Score(
			dterms,
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
			rel:             rel,
			pkgConceptScore: pkgCS,
			orig:            i,
		})
	}

	// Drop below min_relevance, then sort with near-tie clustering (see
	// codeEntryRelevanceTieMaxDelta).
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
		// When BM25 and concept-reach match, prefer more specific import paths
		// (e.g. .../drift/specscan over .../drift) so subpackage entry points
		// are not always cut off by file-path lexicographic order.
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

// eligiblePackagesByConcept returns the maximum query→concept relevance score
// for each import path reachable via a passing concept. Used to rank
// same-BM25 code symbols (tie-break) and to determine concept eligibility in
// loadCodeEntryPoints.
func eligiblePackagesByConcept(workspacePath string, queryTerms []string, cfg ScoringConfig) map[string]float64 {
	return packageBestConceptReachScores(workspacePath, queryTerms, cfg, cfg.UseBigrams)
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
