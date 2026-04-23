package query

import (
	"errors"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"math"
	"os"
	"path/filepath"
)

const scoringHeader = `# Default BM25F and routing parameters for "oz context query".
# Same values as internal/query.DefaultScoringConfig() unless overridden below.
# Tuning: "oz context scoring" (list, describe, set) — do not hand-edit without checking keys.
`

// ScoringTOMLPath returns the path to context/scoring.toml.
func ScoringTOMLPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, "context", "scoring.toml")
}

// scoringTOMLIn is the on-disk shape. Pointer fields mean "key was present" for merge
// and for distinguishing explicit false in booleans.
type scoringTOMLIn struct {
	BM25     *scoringTOMLBM25In    `toml:"bm25"`
	Fields   *scoringTOMLFieldsIn  `toml:"fields"`
	Weights  *scoringTOMLWeightsIn `toml:"weights"`
	Routing  *scoringTOMLRoutingIn `toml:"routing"`
	Tokenize *scoringTOMLTokenIn   `toml:"tokenize"`
}

type scoringTOMLBM25In struct {
	K1 *float64 `toml:"k1"`
}

type scoringTOMLFieldsIn struct {
	BText *float64 `toml:"b_text"`
	BPath *float64 `toml:"b_path"`
}

type scoringTOMLWeightsIn struct {
	Scope            *float64 `toml:"scope"`
	Role             *float64 `toml:"role"`
	Responsibilities *float64 `toml:"responsibilities"`
	Readchain        *float64 `toml:"readchain"`
	OutOfScope       *float64 `toml:"out_of_scope_penalty"`
}

type scoringTOMLRoutingIn struct {
	Confidence   *float64 `toml:"confidence_threshold"`
	MinScore     *float64 `toml:"min_score"`
	Temperature  *float64 `toml:"temperature"`
	MinCandidate *float64 `toml:"min_candidate_confidence"`
	IncludeNotes *bool    `toml:"include_notes"`
}

type scoringTOMLTokenIn struct {
	UseBigrams *bool `toml:"use_bigrams"`
}

// knownTopLevelTables lists allowed [section] names in scoring.toml.
var knownTopLevelTables = map[string]struct{}{
	"bm25": {}, "fields": {}, "weights": {}, "routing": {}, "tokenize": {},
}

// allowedKeysInSection maps TOML section (table) name to allowed key names.
var allowedKeysInSection = map[string]map[string]struct{}{
	"bm25": {"k1": {}},
	"fields": {
		"b_text": {},
		"b_path": {},
	},
	"weights": {
		"scope": {}, "role": {}, "responsibilities": {},
		"readchain": {}, "out_of_scope_penalty": {},
	},
	"routing": {
		"confidence_threshold": {}, "min_score": {},
		"temperature": {}, "min_candidate_confidence": {}, "include_notes": {},
	},
	"tokenize": {
		"use_bigrams": {},
	},
}

// LoadConfig reads context/scoring.toml from workspaceRoot, if present,
// and overrides defaults with the values it finds.
// Missing file or unrecognised keys are silently ignored (per routing-packet spec).
func LoadConfig(workspaceRoot string) ScoringConfig {
	cfg := DefaultScoringConfig()
	b, err := os.ReadFile(ScoringTOMLPath(workspaceRoot))
	if err != nil {
		return cfg
	}
	var in scoringTOMLIn
	if err := toml.Unmarshal(b, &in); err != nil {
		return cfg
	}
	mergeScoringTOML(&cfg, &in)
	return cfg
}

func mergeScoringTOML(cfg *ScoringConfig, in *scoringTOMLIn) {
	if in.BM25 != nil && in.BM25.K1 != nil {
		cfg.K1 = *in.BM25.K1
	}
	if in.Fields != nil {
		if in.Fields.BText != nil {
			cfg.BText = *in.Fields.BText
		}
		if in.Fields.BPath != nil {
			cfg.BPath = *in.Fields.BPath
		}
	}
	if in.Weights != nil {
		if in.Weights.Scope != nil {
			cfg.WeightScope = *in.Weights.Scope
		}
		if in.Weights.Role != nil {
			cfg.WeightRole = *in.Weights.Role
		}
		if in.Weights.Responsibilities != nil {
			cfg.WeightResponsibilities = *in.Weights.Responsibilities
		}
		if in.Weights.Readchain != nil {
			cfg.WeightReadchain = *in.Weights.Readchain
		}
		if in.Weights.OutOfScope != nil {
			cfg.OutOfScopePenalty = *in.Weights.OutOfScope
		}
	}
	if in.Routing != nil {
		if in.Routing.Confidence != nil {
			cfg.ConfidenceThreshold = *in.Routing.Confidence
		}
		if in.Routing.MinScore != nil {
			cfg.MinScore = *in.Routing.MinScore
		}
		if in.Routing.Temperature != nil {
			cfg.Temperature = *in.Routing.Temperature
		}
		if in.Routing.MinCandidate != nil {
			cfg.MinCandidateConfidence = *in.Routing.MinCandidate
		}
		if in.Routing.IncludeNotes != nil {
			cfg.IncludeNotes = *in.Routing.IncludeNotes
		}
	}
	if in.Tokenize != nil && in.Tokenize.UseBigrams != nil {
		cfg.UseBigrams = *in.Tokenize.UseBigrams
	}
}

// WriteScoringTOML writes cfg to context/scoring.toml using a canonical section order
// and an atomic replace.
func WriteScoringTOML(workspaceRoot string, cfg ScoringConfig) error {
	if err := ValidateScoringConfig(cfg); err != nil {
		return err
	}
	out := buildTOMLDocument(cfg)
	path := ScoringTOMLPath(workspaceRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return writeFileAtomic(path, out)
}

func buildTOMLDocument(cfg ScoringConfig) []byte {
	// Struct field order is stable; maps would randomize TOML key order.
	enc := struct {
		BM25 struct {
			K1 float64 `toml:"k1"`
		} `toml:"bm25"`
		Fields struct {
			BText float64 `toml:"b_text"`
			BPath float64 `toml:"b_path"`
		} `toml:"fields"`
		Weights struct {
			Scope             float64 `toml:"scope"`
			Role              float64 `toml:"role"`
			Responsibilities  float64 `toml:"responsibilities"`
			Readchain         float64 `toml:"readchain"`
			OutOfScopePenalty float64 `toml:"out_of_scope_penalty"`
		} `toml:"weights"`
		Routing struct {
			Confidence             float64 `toml:"confidence_threshold"`
			MinScore               float64 `toml:"min_score"`
			Temperature            float64 `toml:"temperature"`
			MinCandidateConfidence float64 `toml:"min_candidate_confidence"`
			IncludeNotes           bool    `toml:"include_notes"`
		} `toml:"routing"`
		Tokenize struct {
			UseBigrams bool `toml:"use_bigrams"`
		} `toml:"tokenize"`
	}{}
	enc.BM25.K1 = cfg.K1
	enc.Fields.BText = cfg.BText
	enc.Fields.BPath = cfg.BPath
	enc.Weights.Scope = cfg.WeightScope
	enc.Weights.Role = cfg.WeightRole
	enc.Weights.Responsibilities = cfg.WeightResponsibilities
	enc.Weights.Readchain = cfg.WeightReadchain
	enc.Weights.OutOfScopePenalty = cfg.OutOfScopePenalty
	enc.Routing.Confidence = cfg.ConfidenceThreshold
	enc.Routing.MinScore = cfg.MinScore
	enc.Routing.Temperature = cfg.Temperature
	enc.Routing.MinCandidateConfidence = cfg.MinCandidateConfidence
	enc.Routing.IncludeNotes = cfg.IncludeNotes
	enc.Tokenize.UseBigrams = cfg.UseBigrams
	b, err := toml.Marshal(enc)
	if err != nil {
		panic("toml.Marshal scoring config: " + err.Error())
	}
	return append([]byte(scoringHeader+"\n"), b...)
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "scoring.toml.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// ValidateScoringFile checks that the file, if it exists, contains only known keys
// and that the resulting ScoringConfig passes ValidateScoringConfig. Missing file is OK.
func ValidateScoringFile(workspaceRoot string) error {
	p := ScoringTOMLPath(workspaceRoot)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if uerr := unknownKeysError(b); uerr != nil {
		return uerr
	}
	cfg := DefaultScoringConfig()
	var in scoringTOMLIn
	if err := toml.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("parse %s: %w", p, err)
	}
	mergeScoringTOML(&cfg, &in)
	return ValidateScoringConfig(cfg)
}

func unknownKeysError(b []byte) error {
	var root map[string]any
	if err := toml.Unmarshal(b, &root); err != nil {
		return fmt.Errorf("parse TOML: %w", err)
	}
	return walkUnknownRoot(root)
}

func walkUnknownRoot(m map[string]any) error {
	for k, v := range m {
		if _, known := knownTopLevelTables[k]; !known {
			if _, isTable := v.(map[string]any); isTable {
				return fmt.Errorf("unknown top-level table %q (allowed: bm25, fields, weights, routing, tokenize)", k)
			}
		}
		allow, ok := allowedKeysInSection[k]
		if !ok {
			continue
		}
		sub, isTable := v.(map[string]any)
		if !isTable {
			if v != nil {
				return fmt.Errorf("table %q must be a TOML table of key/value pairs", k)
			}
			continue
		}
		for subK := range sub {
			if _, allowed := allow[subK]; !allowed {
				return fmt.Errorf("unknown key %q in [%s] (see: oz context scoring list)", subK, k)
			}
		}
	}
	return nil
}

// ValidateScoringConfig enforces value ranges for routing and BM25 parameters.
func ValidateScoringConfig(cfg ScoringConfig) error {
	if err := validatePositiveFinite("bm25.k1", cfg.K1); err != nil {
		return err
	}
	if err := validatePositiveFinite("fields.b_text", cfg.BText); err != nil {
		return err
	}
	if err := validatePositiveFinite("fields.b_path", cfg.BPath); err != nil {
		return err
	}
	if err := validateNonnegFinite("weights.scope", cfg.WeightScope); err != nil {
		return err
	}
	if err := validateNonnegFinite("weights.role", cfg.WeightRole); err != nil {
		return err
	}
	if err := validateNonnegFinite("weights.responsibilities", cfg.WeightResponsibilities); err != nil {
		return err
	}
	if err := validateNonnegFinite("weights.readchain", cfg.WeightReadchain); err != nil {
		return err
	}
	if err := validatePositiveFinite("weights.out_of_scope_penalty", cfg.OutOfScopePenalty); err != nil {
		return err
	}
	if err := validate01("routing.confidence_threshold", cfg.ConfidenceThreshold, true); err != nil {
		return err
	}
	if err := validatePositiveFinite("routing.min_score", cfg.MinScore); err != nil {
		return err
	}
	if err := validatePositiveFinite("routing.temperature", cfg.Temperature); err != nil {
		return err
	}
	if err := validate01("routing.min_candidate_confidence", cfg.MinCandidateConfidence, true); err != nil {
		return err
	}
	return nil
}

func validate01(key string, v float64, exclusiveZero bool) error {
	if err := checkFinite(v); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	if exclusiveZero {
		if v <= 0 || v > 1 {
			return fmt.Errorf("%s: must be in (0,1], got %g", key, v)
		}
	} else {
		if v < 0 || v > 1 {
			return fmt.Errorf("%s: must be in [0,1], got %g", key, v)
		}
	}
	return nil
}

func validatePositiveFinite(key string, v float64) error {
	if err := checkFinite(v); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	if v <= 0 {
		return fmt.Errorf("%s: must be > 0, got %g", key, v)
	}
	return nil
}

func validateNonnegFinite(key string, v float64) error {
	if err := checkFinite(v); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	if v < 0 {
		return fmt.Errorf("%s: must be >= 0, got %g", key, v)
	}
	return nil
}

func checkFinite(v float64) error {
	if math.IsNaN(v) {
		return errors.New("value is NaN")
	}
	if math.IsInf(v, 0) {
		return errors.New("value is infinite")
	}
	return nil
}
