package query

import (
	"fmt"
	"strconv"
	"strings"
)

// ScoringValueKind is the type of a tunable key for describe/list output.
type ScoringValueKind string

const (
	ScoringKindFloat ScoringValueKind = "float"
	ScoringKindBool  ScoringValueKind = "bool"
)

// ScoringKeyMeta is static help for one tunable key (source of truth for list + describe).
type ScoringKeyMeta struct {
	Key         string
	Title       string
	Description string
	Kind        ScoringValueKind
}

// ScoringDescribe is JSON output for "oz context scoring describe --json".
type ScoringDescribe struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Default     string `json:"default"`
	Current     string `json:"current"`
}

// AllScoringKeyMeta is every valid section.name key, sorted for display in list.
var AllScoringKeyMeta = []ScoringKeyMeta{
	{
		Key:   "bm25.k1",
		Title: "BM25F term frequency saturation (k1)",
		Description: `Controls how strongly additional term matches in a field increase the score. ` +
			`Higher k1 means term frequency has more weight before saturating. Change this when ` +
			`routing seems too sensitive or too flat to small wording differences across agents.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "fields.b_text",
		Title: "Length normalization for text fields (b_text)",
		Description: `Length normalization for role and responsibilities. Higher values mean long text ` +
			`is penalized more (typical ~0.75). Adjust if your agent docs are very long or very short ` +
			`and routing feels length-biased.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "fields.b_path",
		Title: "Length normalization for path fields (b_path)",
		Description: `Length normalization for scope and readchain path fields. Higher means longer ` +
			`paths are discounted more. Tune when scope lines dominate or under-contribute to scores.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.scope",
		Title: "Weight for scope field",
		Description: `Boost for matches in the agent’s scope (path) field. Raising it routes tasks ` +
			`more toward agents whose scope tokens overlap the query; lowering blurs scope boundaries.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.role",
		Title: "Weight for role field",
		Description: `Boost for matches in the agent’s Role section. Use when job titles and role ` +
			`phrases are the best routing signal in your workspace.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.responsibilities",
		Title: "Weight for responsibilities field",
		Description: `Boost for matches in Responsibilities. Increase if tasks usually match bulleted ` +
			`responsibilities; decrease if that field is noisy.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.readchain",
		Title: "Weight for readchain field",
		Description: `Boost for readchain path tokens. Default 0: shared readchains can pollute IDF. ` +
			`Only raise if you have a small curated graph and readchain is a strong signal.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.out_of_scope_penalty",
		Title: "Out-of-scope penalty per term",
		Description: `Subtracted for each query term that matches the agent’s out-of-scope list. ` +
			`Higher values push “wrong topic” traffic away from that agent even when other fields match.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "routing.confidence_threshold",
		Title: "Softmax confidence for a clear owner",
		Description: `If the top agent’s softmax confidence is below this, the result is treated as ` +
			`ambiguous: candidate_agents is filled (per routing packet). Lower = more often “no single owner”. ` +
			`Must be in (0,1].`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "routing.min_score",
		Title: "Minimum raw BM25F score",
		Description: `If the best raw BM25F score is below this, the engine returns no_clear_owner. ` +
			`Keep small (e.g. 0.01) to prefer a best guess on weak matches; raise to be stricter.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "routing.temperature",
		Title: "Softmax temperature",
		Description: `Temperature for converting raw scores to confidences. Lower = more decisive ` +
			`(winner takes most probability). Higher = softer distribution and more candidate overlap.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "routing.min_candidate_confidence",
		Title: "Minimum confidence to list a candidate",
		Description: `When the winner is below confidence_threshold, only agents with softmax ` +
			`confidence at least this value appear in candidate_agents. In (0,1].`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "routing.include_notes",
		Title: "Include notes/ in context blocks",
		Description: `When true, note-tier nodes can appear in context blocks. ` +
			`The “oz context query --include-notes” flag OR this setting can enable notes (union). ` +
			`Set via file for a workspace default, or use the flag for a one-off query.`,
		Kind: ScoringKindBool,
	},
	{
		Key:   "tokenize.use_bigrams",
		Title: "Emit adjacent stem bigrams",
		Description: `When true, the tokenizer adds stem bigrams (e.g. rest_api) after unigrams. ` +
			`Can help compound phrases; may over-index on large corpora. Off by default.`,
		Kind: ScoringKindBool,
	},
}

// scoringKeyIndex maps "section.name" to metadata.
var scoringKeyIndex = func() map[string]ScoringKeyMeta {
	m := make(map[string]ScoringKeyMeta, len(AllScoringKeyMeta))
	for _, meta := range AllScoringKeyMeta {
		m[meta.Key] = meta
	}
	return m
}()

// ScoringKeyMetaByName returns metadata for a valid key, or false.
func ScoringKeyMetaByName(key string) (ScoringKeyMeta, bool) {
	meta, ok := scoringKeyIndex[key]
	return meta, ok
}

// GetScoringValueString returns a stable string for use in get/show/describe.
func GetScoringValueString(cfg ScoringConfig, key string) (string, error) {
	v, err := getScoringValue(cfg, key)
	if err != nil {
		return "", err
	}
	return formatScoringAny(v, key)
}

func getScoringValue(cfg ScoringConfig, key string) (any, error) {
	switch key {
	case "bm25.k1":
		return cfg.K1, nil
	case "fields.b_text":
		return cfg.BText, nil
	case "fields.b_path":
		return cfg.BPath, nil
	case "weights.scope":
		return cfg.WeightScope, nil
	case "weights.role":
		return cfg.WeightRole, nil
	case "weights.responsibilities":
		return cfg.WeightResponsibilities, nil
	case "weights.readchain":
		return cfg.WeightReadchain, nil
	case "weights.out_of_scope_penalty":
		return cfg.OutOfScopePenalty, nil
	case "routing.confidence_threshold":
		return cfg.ConfidenceThreshold, nil
	case "routing.min_score":
		return cfg.MinScore, nil
	case "routing.temperature":
		return cfg.Temperature, nil
	case "routing.min_candidate_confidence":
		return cfg.MinCandidateConfidence, nil
	case "routing.include_notes":
		return cfg.IncludeNotes, nil
	case "tokenize.use_bigrams":
		return cfg.UseBigrams, nil
	default:
		return nil, fmt.Errorf("unknown key %q (oz context scoring list)", key)
	}
}

func formatScoringAny(v any, key string) (string, error) {
	switch t := v.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case bool:
		if t {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("internal: bad type for %s", key)
	}
}

// ParseScoringValue parses a user string for the given key kind.
func ParseScoringValue(key, raw string) (any, error) {
	meta, ok := ScoringKeyMetaByName(key)
	if !ok {
		return nil, fmt.Errorf("unknown key %q", key)
	}
	s := strings.TrimSpace(raw)
	switch meta.Kind {
	case ScoringKindBool:
		switch strings.ToLower(s) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return nil, fmt.Errorf("expected true or false, got %q", raw)
		}
	case ScoringKindFloat:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("expected a number, got %q", raw)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("internal: unknown kind for %s", key)
	}
}

// ApplyScoringValue sets one field on cfg from a parsed value (from ParseScoringValue).
func ApplyScoringValue(cfg *ScoringConfig, key string, v any) error {
	if _, ok := ScoringKeyMetaByName(key); !ok {
		return fmt.Errorf("unknown key %q", key)
	}
	switch key {
	case "bm25.k1":
		x, ok := v.(float64)
		if !ok {
			return fmt.Errorf("internal: float expected for %s", key)
		}
		cfg.K1 = x
	case "fields.b_text":
		x, _ := v.(float64)
		cfg.BText = x
	case "fields.b_path":
		x, _ := v.(float64)
		cfg.BPath = x
	case "weights.scope":
		x, _ := v.(float64)
		cfg.WeightScope = x
	case "weights.role":
		x, _ := v.(float64)
		cfg.WeightRole = x
	case "weights.responsibilities":
		x, _ := v.(float64)
		cfg.WeightResponsibilities = x
	case "weights.readchain":
		x, _ := v.(float64)
		cfg.WeightReadchain = x
	case "weights.out_of_scope_penalty":
		x, _ := v.(float64)
		cfg.OutOfScopePenalty = x
	case "routing.confidence_threshold":
		x, _ := v.(float64)
		cfg.ConfidenceThreshold = x
	case "routing.min_score":
		x, _ := v.(float64)
		cfg.MinScore = x
	case "routing.temperature":
		x, _ := v.(float64)
		cfg.Temperature = x
	case "routing.min_candidate_confidence":
		x, _ := v.(float64)
		cfg.MinCandidateConfidence = x
	case "routing.include_notes":
		x, ok := v.(bool)
		if !ok {
			return fmt.Errorf("internal: bool expected for %s", key)
		}
		cfg.IncludeNotes = x
	case "tokenize.use_bigrams":
		x, _ := v.(bool)
		cfg.UseBigrams = x
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}

// SetScoringKey updates one key in context/scoring.toml (full effective config write).
func SetScoringKey(workspaceRoot, key, value string) error {
	if _, ok := ScoringKeyMetaByName(key); !ok {
		return fmt.Errorf("unknown key %q (oz context scoring list)", key)
	}
	parsed, err := ParseScoringValue(key, value)
	if err != nil {
		return err
	}
	cfg := LoadConfig(workspaceRoot)
	if err := ApplyScoringValue(&cfg, key, parsed); err != nil {
		return err
	}
	return WriteScoringTOML(workspaceRoot, cfg)
}

// DefaultStringForKey returns the default value as a string (for describe).
func DefaultStringForKey(key string) (string, error) {
	d := DefaultScoringConfig()
	return GetScoringValueString(d, key)
}

// BuildScoringDescribe returns JSON-friendly describe payload for a key.
func BuildScoringDescribe(workspaceRoot, key string) (ScoringDescribe, error) {
	meta, ok := ScoringKeyMetaByName(key)
	if !ok {
		return ScoringDescribe{}, fmt.Errorf("unknown key %q", key)
	}
	cur := LoadConfig(workspaceRoot)
	def := DefaultScoringConfig()
	defStr, err := GetScoringValueString(def, key)
	if err != nil {
		return ScoringDescribe{}, err
	}
	curStr, err := GetScoringValueString(cur, key)
	if err != nil {
		return ScoringDescribe{}, err
	}
	return ScoringDescribe{
		Key:         key,
		Type:        string(meta.Kind),
		Title:       meta.Title,
		Description: meta.Description,
		Default:     defStr,
		Current:     curStr,
	}, nil
}
