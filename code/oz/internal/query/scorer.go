package query

import (
	"math"
	"sort"

	"github.com/joaoajmatos/oz/internal/query/bm25"
)

// Score holds the BM25F score for one agent before softmax.
type Score struct {
	Agent string
	Value float64
}

// ComputeBM25F returns a Score for each agent document.
// Terms is the tokenized query (stemmed, deduplicated). The routing math is
// delegated to the generic BM25 core (bm25.go); only the out-of-scope
// penalty is agent-specific and applied here.
func ComputeBM25F(terms []string, docs []AgentDoc, cfg ScoringConfig) []Score {
	if len(docs) == 0 || len(terms) == 0 {
		return nil
	}

	fields := agentBM25Fields(cfg)

	generic := make([]bm25.FieldDoc, len(docs))
	for i, d := range docs {
		generic[i] = d
	}
	avgLen := bm25.AvgFieldLengths(generic, fields)
	df := bm25.ComputeDF(generic)

	scores := make([]Score, len(docs))
	for i, doc := range docs {
		score := bm25.BM25Score(terms, doc.Fields(), fields, cfg.K1, avgLen, df, len(docs))

		// Out-of-scope penalty: subtract for each query term that appears
		// in the agent's out-of-scope declaration.
		for _, term := range terms {
			if containsTerm(term, doc.OutOfScope) {
				score -= cfg.OutOfScopePenalty * (1.0 / float64(len(terms)))
			}
		}
		if score < 0 {
			score = 0
		}

		scores[i] = Score{Agent: doc.Name, Value: score}
	}
	return scores
}

// agentBM25Fields returns the field set used to score the agent-routing
// corpus, in a stable order (matters for floating-point reproducibility).
func agentBM25Fields(cfg ScoringConfig) []bm25.BM25Field {
	return []bm25.BM25Field{
		{Name: AgentFieldScope, Weight: cfg.WeightScope, B: cfg.BPath},
		{Name: AgentFieldRole, Weight: cfg.WeightRole, B: cfg.BText},
		{Name: AgentFieldResponsibilities, Weight: cfg.WeightResponsibilities, B: cfg.BText},
		{Name: AgentFieldReadChain, Weight: cfg.WeightReadchain, B: cfg.BPath},
	}
}

// Softmax converts raw BM25F scores to confidences using temperature scaling.
// Returns confidences in the same order as scores.
func Softmax(scores []Score, temperature float64) []float64 {
	if len(scores) == 0 {
		return nil
	}

	// Find max for numerical stability.
	maxVal := scores[0].Value
	for _, s := range scores[1:] {
		if s.Value > maxVal {
			maxVal = s.Value
		}
	}

	exp := make([]float64, len(scores))
	sum := 0.0
	for i, s := range scores {
		e := math.Exp((s.Value - maxVal) / temperature)
		exp[i] = e
		sum += e
	}

	conf := make([]float64, len(scores))
	for i, e := range exp {
		conf[i] = e / sum
	}
	return conf
}

// Route picks the winning agent and assembles candidate_agents when confidence
// is below the threshold.
type RouteResult struct {
	Agent           string
	Confidence      float64
	Candidates      []CandidateAgent
	NoClearOwner    bool
}

// Route applies softmax, checks thresholds, and returns routing decision.
func Route(scores []Score, cfg ScoringConfig) RouteResult {
	if len(scores) == 0 {
		return RouteResult{NoClearOwner: true}
	}

	// Check if any score exceeds the minimum threshold.
	maxRaw := 0.0
	for _, s := range scores {
		if s.Value > maxRaw {
			maxRaw = s.Value
		}
	}
	if maxRaw < cfg.MinScore {
		return RouteResult{NoClearOwner: true}
	}

	conf := Softmax(scores, cfg.Temperature)

	// Find top agent.
	topIdx := 0
	for i, c := range conf {
		if c > conf[topIdx] {
			topIdx = i
		}
	}

	result := RouteResult{
		Agent:      scores[topIdx].Agent,
		Confidence: conf[topIdx],
	}

	// Populate candidates when confidence is below threshold.
	if conf[topIdx] < cfg.ConfidenceThreshold {
		minCandidateConf := cfg.MinCandidateConfidence
		if minCandidateConf <= 0 {
			minCandidateConf = 0.2
		}
		for i, s := range scores {
			if conf[i] >= minCandidateConf {
				result.Candidates = append(result.Candidates, CandidateAgent{
					Name:       s.Agent,
					Confidence: conf[i],
				})
			}
		}
		sort.Slice(result.Candidates, func(i, j int) bool {
			return result.Candidates[i].Confidence > result.Candidates[j].Confidence
		})
	}

	return result
}

