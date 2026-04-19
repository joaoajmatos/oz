package query

import (
	"math"
	"sort"
)

// Score holds the BM25F score for one agent before softmax.
type Score struct {
	Agent string
	Value float64
}

// ComputeBM25F returns a Score for each agent document.
// Terms is the tokenized query (stemmed, deduplicated).
func ComputeBM25F(terms []string, docs []AgentDoc, cfg ScoringConfig) []Score {
	if len(docs) == 0 || len(terms) == 0 {
		return nil
	}

	// Compute per-field average lengths.
	avgScope, avgRole, avgResp, avgRC := avgFieldLengths(docs)

	// Compute per-term document frequency (across all fields combined).
	df := computeDF(docs)

	N := float64(len(docs))

	scores := make([]Score, len(docs))
	for i, doc := range docs {
		score := 0.0
		for _, term := range terms {
			// IDF with floor of 1.0 to handle small corpora.
			dfVal := float64(df[term])
			idf := math.Log((N-dfVal+0.5)/(dfVal+0.5) + 1)
			if idf < 1.0 {
				idf = 1.0
			}

			// Pseudo-TF: weighted sum across fields.
			scopeTF := normTF(termFreq(term, doc.Scope), len(doc.Scope), avgScope, cfg.BPath)
			roleTF := normTF(termFreq(term, doc.Role), len(doc.Role), avgRole, cfg.BText)
			respTF := normTF(termFreq(term, doc.Responsibilities), len(doc.Responsibilities), avgResp, cfg.BText)
			rcTF := normTF(termFreq(term, doc.ReadChain), len(doc.ReadChain), avgRC, cfg.BPath)

			tfTilde := cfg.WeightScope*scopeTF +
				cfg.WeightRole*roleTF +
				cfg.WeightResponsibilities*respTF +
				cfg.WeightReadchain*rcTF

			score += idf * tfTilde / (cfg.K1 + tfTilde)
		}

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
		const minCandidateConf = 0.15
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

// ---- helpers ----------------------------------------------------------------

// normTF returns the BM25F normalised term frequency for one field.
// tf = raw count, fieldLen = length of this field in the doc,
// avgLen = average field length across corpus, b = normalisation factor.
func normTF(tf, fieldLen int, avgLen, b float64) float64 {
	if tf == 0 || fieldLen == 0 {
		return 0
	}
	norm := 1 - b + b*float64(fieldLen)/avgLen
	if norm == 0 {
		norm = 1
	}
	return float64(tf) / norm
}

// avgFieldLengths returns average lengths (in tokens) for scope, role,
// responsibilities, and readchain across all docs.
func avgFieldLengths(docs []AgentDoc) (scope, role, resp, rc float64) {
	if len(docs) == 0 {
		return 1, 1, 1, 1
	}
	for _, d := range docs {
		scope += float64(len(d.Scope))
		role += float64(len(d.Role))
		resp += float64(len(d.Responsibilities))
		rc += float64(len(d.ReadChain))
	}
	n := float64(len(docs))
	scope /= n
	role /= n
	resp /= n
	rc /= n
	// Floor to 1 to avoid division by zero.
	if scope < 1 {
		scope = 1
	}
	if role < 1 {
		role = 1
	}
	if resp < 1 {
		resp = 1
	}
	if rc < 1 {
		rc = 1
	}
	return
}

// computeDF returns how many agents contain each term in any field.
func computeDF(docs []AgentDoc) map[string]int {
	df := make(map[string]int)
	for _, d := range docs {
		seen := make(map[string]bool)
		for _, tokens := range [][]string{d.Scope, d.Role, d.Responsibilities, d.ReadChain} {
			for _, t := range tokens {
				if !seen[t] {
					df[t]++
					seen[t] = true
				}
			}
		}
	}
	return df
}
