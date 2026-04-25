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
	BM25      *scoringTOMLBM25In      `toml:"bm25"`
	Fields    *scoringTOMLFieldsIn    `toml:"fields"`
	Weights   *scoringTOMLWeightsIn   `toml:"weights"`
	Routing   *scoringTOMLRoutingIn   `toml:"routing"`
	Tokenize  *scoringTOMLTokenIn     `toml:"tokenize"`
	Retrieval *scoringTOMLRetrievalIn `toml:"retrieval"`
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
	Skills           *float64 `toml:"skills"`
	ContextTopics    *float64 `toml:"context_topics"`
	Rules            *float64 `toml:"rules"`
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

type scoringTOMLRetrievalIn struct {
	IncludeNotes  *bool                      `toml:"include_notes"`
	MinRelevance  *float64                   `toml:"min_relevance"`
	MaxBlocks     *float64                   `toml:"max_blocks"`
	MaxCodeEntryPoints *float64              `toml:"max_code_entry_points"`
	MaxImplementingPackages *float64         `toml:"max_implementing_packages"`
	MaxRelevantConcepts     *float64         `toml:"max_relevant_concepts"`
	ConceptMinRelevance     *float64 `toml:"concept_min_relevance"`
	ConceptMinFractionOfTop *float64 `toml:"concept_min_fraction_of_top"`
	ConceptMinQueryCoverage *float64 `toml:"concept_min_query_coverage"`
	ConceptUseBigrams       *bool    `toml:"concept_use_bigrams"`
	BM25          *scoringTOMLRetrievalBM25  `toml:"bm25"`
	Fields        *scoringTOMLRetrievalField `toml:"fields"`
	Concepts      *scoringTOMLRetrievalConcepts `toml:"concepts"`
	TrustBoost    *scoringTOMLRetrievalTrust `toml:"trust_boost"`
	AgentAffinity *float64                   `toml:"agent_affinity"`
}

type scoringTOMLRetrievalBM25 struct {
	K1 *float64 `toml:"k1"`
}

type scoringTOMLRetrievalField struct {
	WeightTitle *float64 `toml:"weight_title"`
	WeightPath  *float64 `toml:"weight_path"`
	WeightBody  *float64 `toml:"weight_body"`
	WeightKind  *float64 `toml:"weight_kind"`
}

type scoringTOMLRetrievalConcepts struct {
	WeightName         *float64 `toml:"weight_name"`
	WeightDescription  *float64 `toml:"weight_description"`
	WeightSourceFiles  *float64 `toml:"weight_source_files"`
}

type scoringTOMLRetrievalTrust struct {
	Specs   *float64 `toml:"specs"`
	Docs    *float64 `toml:"docs"`
	Context *float64 `toml:"context"`
	Notes   *float64 `toml:"notes"`
}

// knownTopLevelTables lists allowed [section] names in scoring.toml.
var knownTopLevelTables = map[string]struct{}{
	"bm25": {}, "fields": {}, "weights": {}, "routing": {}, "tokenize": {}, "retrieval": {},
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
		"readchain": {}, "skills": {}, "context_topics": {}, "rules": {},
		"out_of_scope_penalty": {},
	},
	"routing": {
		"confidence_threshold": {}, "min_score": {},
		"temperature": {}, "min_candidate_confidence": {}, "include_notes": {},
	},
	"tokenize": {
		"use_bigrams": {},
	},
	"retrieval": {
		"include_notes": {},
		"min_relevance": {}, "max_blocks": {}, "max_code_entry_points": {},
		"max_implementing_packages": {}, "max_relevant_concepts": {}, "concept_min_relevance": {},
		"concept_min_fraction_of_top": {}, "concept_min_query_coverage": {},
		"concept_use_bigrams": {}, "agent_affinity": {},
		"bm25": {}, "fields": {}, "concepts": {}, "trust_boost": {},
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
		if in.Weights.Skills != nil {
			cfg.WeightSkills = *in.Weights.Skills
		}
		if in.Weights.ContextTopics != nil {
			cfg.WeightContextTopics = *in.Weights.ContextTopics
		}
		if in.Weights.Rules != nil {
			cfg.WeightRules = *in.Weights.Rules
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
	if in.Retrieval != nil {
		if in.Retrieval.IncludeNotes != nil {
			cfg.IncludeNotes = *in.Retrieval.IncludeNotes
		}
		if in.Retrieval.MinRelevance != nil {
			cfg.RetrievalMinRelevance = *in.Retrieval.MinRelevance
		}
		if in.Retrieval.MaxBlocks != nil {
			cfg.RetrievalMaxBlocks = *in.Retrieval.MaxBlocks
		}
		if in.Retrieval.MaxCodeEntryPoints != nil {
			cfg.RetrievalMaxCodeEntryPoints = *in.Retrieval.MaxCodeEntryPoints
		}
		if in.Retrieval.MaxImplementingPackages != nil {
			cfg.RetrievalMaxImplementingPackages = *in.Retrieval.MaxImplementingPackages
		}
		if in.Retrieval.MaxRelevantConcepts != nil {
			cfg.RetrievalMaxRelevantConcepts = *in.Retrieval.MaxRelevantConcepts
		}
		if in.Retrieval.ConceptMinRelevance != nil {
			cfg.RetrievalConceptMinRelevance = *in.Retrieval.ConceptMinRelevance
		}
		if in.Retrieval.ConceptMinFractionOfTop != nil {
			cfg.RetrievalConceptMinFractionOfTop = *in.Retrieval.ConceptMinFractionOfTop
		}
		if in.Retrieval.ConceptMinQueryCoverage != nil {
			cfg.RetrievalConceptMinQueryCoverage = *in.Retrieval.ConceptMinQueryCoverage
		}
		if in.Retrieval.ConceptUseBigrams != nil {
			cfg.RetrievalConceptUseBigrams = *in.Retrieval.ConceptUseBigrams
		}
		if in.Retrieval.AgentAffinity != nil {
			cfg.RetrievalAgentAffinity = *in.Retrieval.AgentAffinity
		}
		if in.Retrieval.BM25 != nil && in.Retrieval.BM25.K1 != nil {
			cfg.RetrievalK1 = *in.Retrieval.BM25.K1
		}
		if in.Retrieval.Fields != nil {
			if in.Retrieval.Fields.WeightTitle != nil {
				cfg.RetrievalWeightTitle = *in.Retrieval.Fields.WeightTitle
			}
			if in.Retrieval.Fields.WeightPath != nil {
				cfg.RetrievalWeightPath = *in.Retrieval.Fields.WeightPath
			}
			if in.Retrieval.Fields.WeightBody != nil {
				cfg.RetrievalWeightBody = *in.Retrieval.Fields.WeightBody
			}
			if in.Retrieval.Fields.WeightKind != nil {
				cfg.RetrievalWeightKind = *in.Retrieval.Fields.WeightKind
			}
		}
		if in.Retrieval.Concepts != nil {
			if in.Retrieval.Concepts.WeightName != nil {
				cfg.RetrievalConceptWeightName = *in.Retrieval.Concepts.WeightName
			}
			if in.Retrieval.Concepts.WeightDescription != nil {
				cfg.RetrievalConceptWeightDescription = *in.Retrieval.Concepts.WeightDescription
			}
			if in.Retrieval.Concepts.WeightSourceFiles != nil {
				cfg.RetrievalConceptWeightSourceFiles = *in.Retrieval.Concepts.WeightSourceFiles
			}
		}
		if in.Retrieval.TrustBoost != nil {
			if in.Retrieval.TrustBoost.Specs != nil {
				cfg.RetrievalTrustBoostSpecs = *in.Retrieval.TrustBoost.Specs
			}
			if in.Retrieval.TrustBoost.Docs != nil {
				cfg.RetrievalTrustBoostDocs = *in.Retrieval.TrustBoost.Docs
			}
			if in.Retrieval.TrustBoost.Context != nil {
				cfg.RetrievalTrustBoostContext = *in.Retrieval.TrustBoost.Context
			}
			if in.Retrieval.TrustBoost.Notes != nil {
				cfg.RetrievalTrustBoostNotes = *in.Retrieval.TrustBoost.Notes
			}
		}
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
			Scope              float64 `toml:"scope"`
			Role               float64 `toml:"role"`
			Responsibilities   float64 `toml:"responsibilities"`
			Readchain          float64 `toml:"readchain"`
			Skills             float64 `toml:"skills"`
			ContextTopics      float64 `toml:"context_topics"`
			Rules              float64 `toml:"rules"`
			OutOfScopePenalty  float64 `toml:"out_of_scope_penalty"`
		} `toml:"weights"`
		Routing struct {
			Confidence             float64 `toml:"confidence_threshold"`
			MinScore               float64 `toml:"min_score"`
			Temperature            float64 `toml:"temperature"`
			MinCandidateConfidence float64 `toml:"min_candidate_confidence"`
		} `toml:"routing"`
		Tokenize struct {
			UseBigrams bool `toml:"use_bigrams"`
		} `toml:"tokenize"`
		Retrieval struct {
			IncludeNotes  bool    `toml:"include_notes"`
			MinRelevance  float64 `toml:"min_relevance"`
			MaxBlocks     float64 `toml:"max_blocks"`
			MaxCodeEntryPoints float64 `toml:"max_code_entry_points"`
			MaxImplementingPackages float64 `toml:"max_implementing_packages"`
			MaxRelevantConcepts     float64 `toml:"max_relevant_concepts"`
			ConceptMinRelevance     float64 `toml:"concept_min_relevance"`
			ConceptMinFractionOfTop float64 `toml:"concept_min_fraction_of_top"`
			ConceptMinQueryCoverage float64 `toml:"concept_min_query_coverage"`
			ConceptUseBigrams       bool    `toml:"concept_use_bigrams"`
			AgentAffinity           float64 `toml:"agent_affinity"`
			BM25          struct {
				K1 float64 `toml:"k1"`
			} `toml:"bm25"`
			Fields struct {
				WeightTitle float64 `toml:"weight_title"`
				WeightPath  float64 `toml:"weight_path"`
				WeightBody  float64 `toml:"weight_body"`
				WeightKind  float64 `toml:"weight_kind"`
			} `toml:"fields"`
			Concepts struct {
				WeightName         float64 `toml:"weight_name"`
				WeightDescription  float64 `toml:"weight_description"`
				WeightSourceFiles  float64 `toml:"weight_source_files"`
			} `toml:"concepts"`
			TrustBoost struct {
				Specs   float64 `toml:"specs"`
				Docs    float64 `toml:"docs"`
				Context float64 `toml:"context"`
				Notes   float64 `toml:"notes"`
			} `toml:"trust_boost"`
		} `toml:"retrieval"`
	}{}
	enc.BM25.K1 = cfg.K1
	enc.Fields.BText = cfg.BText
	enc.Fields.BPath = cfg.BPath
	enc.Weights.Scope = cfg.WeightScope
	enc.Weights.Role = cfg.WeightRole
	enc.Weights.Responsibilities = cfg.WeightResponsibilities
	enc.Weights.Readchain = cfg.WeightReadchain
	enc.Weights.Skills = cfg.WeightSkills
	enc.Weights.ContextTopics = cfg.WeightContextTopics
	enc.Weights.Rules = cfg.WeightRules
	enc.Weights.OutOfScopePenalty = cfg.OutOfScopePenalty
	enc.Routing.Confidence = cfg.ConfidenceThreshold
	enc.Routing.MinScore = cfg.MinScore
	enc.Routing.Temperature = cfg.Temperature
	enc.Routing.MinCandidateConfidence = cfg.MinCandidateConfidence
	enc.Tokenize.UseBigrams = cfg.UseBigrams
	enc.Retrieval.IncludeNotes = cfg.IncludeNotes
	enc.Retrieval.MinRelevance = cfg.RetrievalMinRelevance
	enc.Retrieval.MaxBlocks = cfg.RetrievalMaxBlocks
	enc.Retrieval.MaxCodeEntryPoints = cfg.RetrievalMaxCodeEntryPoints
	enc.Retrieval.MaxImplementingPackages = cfg.RetrievalMaxImplementingPackages
	enc.Retrieval.MaxRelevantConcepts = cfg.RetrievalMaxRelevantConcepts
	enc.Retrieval.ConceptMinRelevance = cfg.RetrievalConceptMinRelevance
	enc.Retrieval.ConceptMinFractionOfTop = cfg.RetrievalConceptMinFractionOfTop
	enc.Retrieval.ConceptMinQueryCoverage = cfg.RetrievalConceptMinQueryCoverage
	enc.Retrieval.ConceptUseBigrams = cfg.RetrievalConceptUseBigrams
	enc.Retrieval.AgentAffinity = cfg.RetrievalAgentAffinity
	enc.Retrieval.BM25.K1 = cfg.RetrievalK1
	enc.Retrieval.Fields.WeightTitle = cfg.RetrievalWeightTitle
	enc.Retrieval.Fields.WeightPath = cfg.RetrievalWeightPath
	enc.Retrieval.Fields.WeightBody = cfg.RetrievalWeightBody
	enc.Retrieval.Fields.WeightKind = cfg.RetrievalWeightKind
	enc.Retrieval.Concepts.WeightName = cfg.RetrievalConceptWeightName
	enc.Retrieval.Concepts.WeightDescription = cfg.RetrievalConceptWeightDescription
	enc.Retrieval.Concepts.WeightSourceFiles = cfg.RetrievalConceptWeightSourceFiles
	enc.Retrieval.TrustBoost.Specs = cfg.RetrievalTrustBoostSpecs
	enc.Retrieval.TrustBoost.Docs = cfg.RetrievalTrustBoostDocs
	enc.Retrieval.TrustBoost.Context = cfg.RetrievalTrustBoostContext
	enc.Retrieval.TrustBoost.Notes = cfg.RetrievalTrustBoostNotes
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
				return fmt.Errorf("unknown top-level table %q (allowed: bm25, fields, weights, routing, tokenize, retrieval)", k)
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
	if err := validateNonnegFinite("weights.skills", cfg.WeightSkills); err != nil {
		return err
	}
	if err := validateNonnegFinite("weights.context_topics", cfg.WeightContextTopics); err != nil {
		return err
	}
	if err := validateNonnegFinite("weights.rules", cfg.WeightRules); err != nil {
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
	if err := validateNonnegFinite("retrieval.min_relevance", cfg.RetrievalMinRelevance); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.max_blocks", cfg.RetrievalMaxBlocks); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.max_code_entry_points", cfg.RetrievalMaxCodeEntryPoints); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.max_implementing_packages", cfg.RetrievalMaxImplementingPackages); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.max_relevant_concepts", cfg.RetrievalMaxRelevantConcepts); err != nil {
		return err
	}
	if err := validateNonnegFinite("retrieval.concept_min_relevance", cfg.RetrievalConceptMinRelevance); err != nil {
		return err
	}
	if err := validate01("retrieval.concept_min_fraction_of_top", cfg.RetrievalConceptMinFractionOfTop, false); err != nil {
		return err
	}
	if err := validate01("retrieval.concept_min_query_coverage", cfg.RetrievalConceptMinQueryCoverage, false); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.bm25.k1", cfg.RetrievalK1); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.agent_affinity", cfg.RetrievalAgentAffinity); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.fields.weight_title", cfg.RetrievalWeightTitle); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.fields.weight_path", cfg.RetrievalWeightPath); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.fields.weight_body", cfg.RetrievalWeightBody); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.fields.weight_kind", cfg.RetrievalWeightKind); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.concepts.weight_name", cfg.RetrievalConceptWeightName); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.concepts.weight_description", cfg.RetrievalConceptWeightDescription); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.concepts.weight_source_files", cfg.RetrievalConceptWeightSourceFiles); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.trust_boost.specs", cfg.RetrievalTrustBoostSpecs); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.trust_boost.docs", cfg.RetrievalTrustBoostDocs); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.trust_boost.context", cfg.RetrievalTrustBoostContext); err != nil {
		return err
	}
	if err := validatePositiveFinite("retrieval.trust_boost.notes", cfg.RetrievalTrustBoostNotes); err != nil {
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
