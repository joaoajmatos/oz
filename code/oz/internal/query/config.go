package query

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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
	WeightScope          float64
	WeightRole           float64
	WeightResponsibilities float64
	WeightReadchain      float64
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
	}
}

// LoadConfig reads context/scoring.toml from workspaceRoot, if present,
// and overrides defaults with the values it finds.
// Missing file or unrecognised keys are silently ignored.
func LoadConfig(workspaceRoot string) ScoringConfig {
	cfg := DefaultScoringConfig()
	path := filepath.Join(workspaceRoot, "context", "scoring.toml")
	f, err := os.Open(path)
	if err != nil {
		return cfg // file absent — use defaults
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Strip inline comments.
		if idx := strings.Index(v, "#"); idx >= 0 {
			v = strings.TrimSpace(v[:idx])
		}
		applyConfigKey(&cfg, section, k, v)
	}
	return cfg
}

func applyConfigKey(cfg *ScoringConfig, section, key, val string) {
	f := parseFloat(val)
	b := parseBool(val)

	switch section + "." + key {
	case "bm25.k1":
		cfg.K1 = f
	case "fields.b_text":
		cfg.BText = f
	case "fields.b_path":
		cfg.BPath = f
	case "weights.scope":
		cfg.WeightScope = f
	case "weights.role":
		cfg.WeightRole = f
	case "weights.responsibilities":
		cfg.WeightResponsibilities = f
	case "weights.readchain":
		cfg.WeightReadchain = f
	case "weights.out_of_scope_penalty":
		cfg.OutOfScopePenalty = f
	case "routing.confidence_threshold":
		cfg.ConfidenceThreshold = f
	case "routing.min_score":
		cfg.MinScore = f
	case "routing.temperature":
		cfg.Temperature = f
	case "routing.include_notes":
		cfg.IncludeNotes = b
	case "routing.min_candidate_confidence":
		cfg.MinCandidateConfidence = f
	case "tokenize.use_bigrams":
		cfg.UseBigrams = b
	}
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}
