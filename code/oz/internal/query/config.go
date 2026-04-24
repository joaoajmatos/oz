package query

// ScoringConfig holds all tunable BM25F parameters and routing thresholds.
// Values can be overridden via context/scoring.toml.
type ScoringConfig struct {
	// BM25F term-frequency saturation parameter.
	K1 float64
	// Document-length normalisation for text fields (role, responsibilities).
	BText float64
	// Document-length normalisation for path fields (scope, readchain).
	BPath float64

	// Per-field boost weights.
	WeightScope            float64
	WeightRole             float64
	WeightResponsibilities float64
	WeightReadchain        float64
	// Out-of-scope penalty subtracted per matching query term.
	OutOfScopePenalty float64

	// Routing thresholds.
	ConfidenceThreshold float64 // below this, populate candidate_agents
	MinScore            float64 // below this, return no_clear_owner
	Temperature         float64 // softmax temperature (lower = more decisive)
	// MinCandidateConfidence: minimum softmax confidence to include an agent in
	// candidate_agents when the winner is below ConfidenceThreshold (PRD: 0.2).
	MinCandidateConfidence float64

	// When true, note nodes are included in context blocks.
	IncludeNotes bool
	// When true, tokenization emits adjacent stem bigrams as "a_b" after unigrams.
	UseBigrams bool

	// Retrieval thresholds and caps.
	RetrievalMinRelevance float64
	RetrievalMaxBlocks    float64
	RetrievalMaxCodeEntryPoints float64

	// Retrieval BM25 and ranking boosts.
	RetrievalK1                float64
	RetrievalAgentAffinity     float64
	RetrievalTrustBoostSpecs   float64
	RetrievalTrustBoostDocs    float64
	RetrievalTrustBoostContext float64
	RetrievalTrustBoostNotes   float64

	// Retrieval field weights.
	RetrievalWeightTitle float64
	RetrievalWeightPath  float64
	RetrievalWeightBody  float64
	RetrievalWeightKind  float64
}

// DefaultScoringConfig returns the default parameters.
func DefaultScoringConfig() ScoringConfig {
	return ScoringConfig{
		K1:                     1.2,
		BText:                  0.75,
		BPath:                  0.5,
		WeightScope:            4.0,
		WeightRole:             2.5,
		WeightResponsibilities: 2.5,
		WeightReadchain:        0.0, // shared readchains pollute IDF; disabled
		OutOfScopePenalty:      2.5,
		ConfidenceThreshold:    0.7,
		MinScore:               0.01, // low threshold: prefer routing to best guess
		Temperature:            0.2,  // decisive routing
		MinCandidateConfidence: 0.2,
		IncludeNotes:           false,
		UseBigrams:             false,
		RetrievalMinRelevance:  0.05,
		RetrievalMaxBlocks:     12,
		RetrievalMaxCodeEntryPoints: 5,
		RetrievalK1:            1.2,
		RetrievalAgentAffinity: 1.2,
		RetrievalTrustBoostSpecs:   1.3,
		RetrievalTrustBoostDocs:    1.0,
		RetrievalTrustBoostContext: 1.0,
		RetrievalTrustBoostNotes:   0.6,
		RetrievalWeightTitle:       2.0,
		RetrievalWeightPath:        1.5,
		RetrievalWeightBody:        1.0,
		RetrievalWeightKind:        1.0,
	}
}

// LoadConfig is defined in scoring_toml.go (TOML decode and merge with defaults).
