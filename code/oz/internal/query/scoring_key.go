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
		Key:   "weights.skills",
		Title: "Weight for skills (backtick path lines)",
		Description: `Boost for tokens from the agent Skills section. Fills routing gaps for terms that ` +
			`only appear there (e.g. “graph”, “MCP”) without raising readchain IDF noise.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.context_topics",
		Title: "Weight for context topics list",
		Description: `Boost for the Context topics section. Use for query hints that are deliberately ` +
			`authored for oz context query without duplicating the whole role text.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.rules",
		Title: "Weight for rules file paths",
		Description: `Boost for path tokens from the Rules section. Lower than role; useful when ` +
			`rules paths disambiguate agents.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "weights.out_of_scope_penalty",
		Title: "Out-of-scope penalty per term",
		Description: `Subtracted for each query term that matches the agent’s out-of-scope list, unless ` +
			`the same term also appears in a positive routing field (role, scope, skills, etc.). ` +
			`Higher values push “wrong topic” traffic away when the term is OOS-only.`,
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
		Title: "Include notes/ in context blocks (legacy key)",
		Description: `Same as retrieval.include_notes. Kept for older workspace files. ` +
			`If both are set, retrieval.include_notes wins. ` +
			`The "oz context query --include-notes" flag ORs with this (union).`,
		Kind: ScoringKindBool,
	},
	{
		Key:   "tokenize.use_bigrams",
		Title: "Emit adjacent stem bigrams",
		Description: `When true, the tokenizer adds stem bigrams (e.g. rest_api) after unigrams. ` +
			`Can help compound phrases; may over-index on large corpora. Off by default.`,
		Kind: ScoringKindBool,
	},
	{
		Key:   "retrieval.include_notes",
		Title: "Include note-tier nodes in the retrieval corpus",
		Description: `When true, nodes under notes/ are scored in the same ranked pipeline as other tiers and ` +
			`downweighted by retrieval.trust_boost.notes. When false, they are hard-excluded and the ` +
			`"notes/" prefix is added to excluded. Default true. "oz context query --include-notes" ORs on top (union).`,
		Kind: ScoringKindBool,
	},
	{
		Key:   "retrieval.min_relevance",
		Title: "Minimum retrieval relevance threshold",
		Description: `Blocks scoring below this value are removed before top-K truncation. ` +
			`Raise to return fewer, higher-signal blocks; lower to increase recall when queries are broad.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.max_blocks",
		Title: "Maximum number of context blocks",
		Description: `Hard cap on ranked context_blocks after thresholding. ` +
			`Lower reduces token load; higher gives more coverage at the cost of potential noise.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.max_code_entry_points",
		Title: "Maximum number of code entry points",
		Description: `Hard cap on ranked code_entry_points returned for implementation queries. ` +
			`Lower keeps packets compact; higher increases recall for larger code surfaces.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.max_implementing_packages",
		Title: "Maximum number of implementing packages",
		Description: `Hard cap on ranked implementing_packages returned from concept matches. ` +
			`Lower increases precision; higher increases recall.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.max_relevant_concepts",
		Title: "Maximum number of query-ranked relevant concepts",
		Description: `Hard cap on relevant_concepts (semantic concept names) after query BM25 scoring ` +
			`and the same concept floor used for implementing_packages.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concept_min_relevance",
		Title: "Minimum concept relevance threshold",
		Description: `Concepts scoring below this threshold are excluded before walking reviewed ` +
			`implements edges to packages and symbols.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concept_min_fraction_of_top",
		Title: "Relative concept floor (fraction of best match)",
		Description: `In [0,1]. When > 0, a concept is kept only if its score is at least this ` +
			`fraction of the highest-scoring concept in the overlay, in addition to ` +
			`retrieval.concept_min_relevance. Set to 0 to use only the absolute floor.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concept_min_query_coverage",
		Title: "Minimum query-term coverage for concepts",
		Description: `In [0,1]. When > 0, a concept is kept only if at least this fraction of ` +
			`the distinct unigram query terms appear in the concept's name, description, or source_paths. ` +
			`0 disables the check. 0.33 requires ≥1/3 coverage (e.g. at least 1 of 3 terms, 2 of 5). ` +
			`Prevents single-stem false positives from matching unrelated concepts.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concept_use_bigrams",
		Title: "Enable bigrams for concept BM25 scoring",
		Description: `When true, adjacent stem bigrams (e.g. semantic_overlay) are added to both ` +
			`the query terms and concept field tokens for concept scoring, independent of ` +
			`tokenize.use_bigrams. Sharpens compound-phrase matching and reduces single-stem ` +
			`collisions where the same word means different things in different contexts.`,
		Kind: ScoringKindBool,
	},
	{
		Key:   "retrieval.agent_affinity",
		Title: "Agent affinity boost",
		Description: `Multiplier for blocks connected to the winning agent (reads/owns/scope links). ` +
			`Raise to prefer agent-local context; lower if read-chain breadth starts overpowering query relevance.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.bm25.k1",
		Title: "Retrieval BM25 term frequency saturation",
		Description: `BM25 k1 used only for block retrieval (independent from routing). ` +
			`Higher values reward repeated term matches more; lower values saturate earlier.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.fields.weight_title",
		Title: "Retrieval weight for title field",
		Description: `Weight for section headings/titles. ` +
			`Raise when query intent is usually captured by headings; lower when headings are generic.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.fields.weight_path",
		Title: "Retrieval weight for path field",
		Description: `Weight for file path tokens (directories, filenames). ` +
			`Raise when users query by package/path terms; lower if path tokens bias ranking too much.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.fields.weight_body",
		Title: "Retrieval weight for body field",
		Description: `Weight for section/file body tokens. ` +
			`Raise when details are in prose/code-adjacent text; lower when body text is verbose and noisy.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.fields.weight_kind",
		Title: "Retrieval weight for symbol kind field",
		Description: `Weight for symbol kind tokens (func, type, method, etc.) in code-symbol retrieval. ` +
			`Raise if kind terms are query-significant; lower if they add noise.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concepts.weight_name",
		Title: "Concept scoring weight for concept name",
		Description: `BM25 field weight for semantic concept names when ranking concepts ` +
			`for implementing package retrieval.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concepts.weight_description",
		Title: "Concept scoring weight for concept description",
		Description: `BM25 field weight for semantic concept descriptions when ranking concepts ` +
			`for implementing package retrieval.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.concepts.weight_source_files",
		Title: "Concept scoring weight for source_files paths",
		Description: `BM25 field weight for tokenized paths in each concept’s source_files list ` +
			`(e.g. ADR filenames with codeindex, .../graph/index). Lets “index”-style ` +
			`queries match without relying on the word “code” in the description alone.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.trust_boost.specs",
		Title: "Retrieval trust boost for specs tier",
		Description: `Trust multiplier for blocks from specs/decisions. ` +
			`Typically highest, so canonical policy/spec sources win ties on similar relevance.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.trust_boost.docs",
		Title: "Retrieval trust boost for docs tier",
		Description: `Trust multiplier for docs/ architecture material. ` +
			`Use to tune how strongly practical docs compete against specs and notes.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.trust_boost.context",
		Title: "Retrieval trust boost for context tier",
		Description: `Trust multiplier for context snapshots under context/. ` +
			`Raise if snapshots are highly curated; lower if they are stale or secondary evidence.`,
		Kind: ScoringKindFloat,
	},
	{
		Key:   "retrieval.trust_boost.notes",
		Title: "Retrieval trust boost for notes tier",
		Description: `Trust multiplier for notes/ planning rationale. ` +
			`Keep below specs/docs to avoid polluting top slots, but high enough for “why” queries to surface notes.`,
		Kind: ScoringKindFloat,
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
	case "weights.skills":
		return cfg.WeightSkills, nil
	case "weights.context_topics":
		return cfg.WeightContextTopics, nil
	case "weights.rules":
		return cfg.WeightRules, nil
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
	case "retrieval.include_notes":
		return cfg.IncludeNotes, nil
	case "tokenize.use_bigrams":
		return cfg.UseBigrams, nil
	case "retrieval.min_relevance":
		return cfg.RetrievalMinRelevance, nil
	case "retrieval.max_blocks":
		return cfg.RetrievalMaxBlocks, nil
	case "retrieval.max_code_entry_points":
		return cfg.RetrievalMaxCodeEntryPoints, nil
	case "retrieval.max_implementing_packages":
		return cfg.RetrievalMaxImplementingPackages, nil
	case "retrieval.max_relevant_concepts":
		return cfg.RetrievalMaxRelevantConcepts, nil
	case "retrieval.concept_min_relevance":
		return cfg.RetrievalConceptMinRelevance, nil
	case "retrieval.concept_min_fraction_of_top":
		return cfg.RetrievalConceptMinFractionOfTop, nil
	case "retrieval.agent_affinity":
		return cfg.RetrievalAgentAffinity, nil
	case "retrieval.bm25.k1":
		return cfg.RetrievalK1, nil
	case "retrieval.fields.weight_title":
		return cfg.RetrievalWeightTitle, nil
	case "retrieval.fields.weight_path":
		return cfg.RetrievalWeightPath, nil
	case "retrieval.fields.weight_body":
		return cfg.RetrievalWeightBody, nil
	case "retrieval.fields.weight_kind":
		return cfg.RetrievalWeightKind, nil
	case "retrieval.concepts.weight_name":
		return cfg.RetrievalConceptWeightName, nil
	case "retrieval.concepts.weight_description":
		return cfg.RetrievalConceptWeightDescription, nil
	case "retrieval.concepts.weight_source_files":
		return cfg.RetrievalConceptWeightSourceFiles, nil
	case "retrieval.trust_boost.specs":
		return cfg.RetrievalTrustBoostSpecs, nil
	case "retrieval.trust_boost.docs":
		return cfg.RetrievalTrustBoostDocs, nil
	case "retrieval.trust_boost.context":
		return cfg.RetrievalTrustBoostContext, nil
	case "retrieval.trust_boost.notes":
		return cfg.RetrievalTrustBoostNotes, nil
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
	case "weights.skills":
		x, _ := v.(float64)
		cfg.WeightSkills = x
	case "weights.context_topics":
		x, _ := v.(float64)
		cfg.WeightContextTopics = x
	case "weights.rules":
		x, _ := v.(float64)
		cfg.WeightRules = x
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
	case "retrieval.include_notes":
		x, ok := v.(bool)
		if !ok {
			return fmt.Errorf("internal: bool expected for %s", key)
		}
		cfg.IncludeNotes = x
	case "tokenize.use_bigrams":
		x, _ := v.(bool)
		cfg.UseBigrams = x
	case "retrieval.min_relevance":
		x, _ := v.(float64)
		cfg.RetrievalMinRelevance = x
	case "retrieval.max_blocks":
		x, _ := v.(float64)
		cfg.RetrievalMaxBlocks = x
	case "retrieval.max_code_entry_points":
		x, _ := v.(float64)
		cfg.RetrievalMaxCodeEntryPoints = x
	case "retrieval.max_implementing_packages":
		x, _ := v.(float64)
		cfg.RetrievalMaxImplementingPackages = x
	case "retrieval.max_relevant_concepts":
		x, _ := v.(float64)
		cfg.RetrievalMaxRelevantConcepts = x
	case "retrieval.concept_min_relevance":
		x, _ := v.(float64)
		cfg.RetrievalConceptMinRelevance = x
	case "retrieval.concept_min_fraction_of_top":
		x, _ := v.(float64)
		cfg.RetrievalConceptMinFractionOfTop = x
	case "retrieval.agent_affinity":
		x, _ := v.(float64)
		cfg.RetrievalAgentAffinity = x
	case "retrieval.bm25.k1":
		x, _ := v.(float64)
		cfg.RetrievalK1 = x
	case "retrieval.fields.weight_title":
		x, _ := v.(float64)
		cfg.RetrievalWeightTitle = x
	case "retrieval.fields.weight_path":
		x, _ := v.(float64)
		cfg.RetrievalWeightPath = x
	case "retrieval.fields.weight_body":
		x, _ := v.(float64)
		cfg.RetrievalWeightBody = x
	case "retrieval.fields.weight_kind":
		x, _ := v.(float64)
		cfg.RetrievalWeightKind = x
	case "retrieval.concepts.weight_name":
		x, _ := v.(float64)
		cfg.RetrievalConceptWeightName = x
	case "retrieval.concepts.weight_description":
		x, _ := v.(float64)
		cfg.RetrievalConceptWeightDescription = x
	case "retrieval.concepts.weight_source_files":
		x, _ := v.(float64)
		cfg.RetrievalConceptWeightSourceFiles = x
	case "retrieval.trust_boost.specs":
		x, _ := v.(float64)
		cfg.RetrievalTrustBoostSpecs = x
	case "retrieval.trust_boost.docs":
		x, _ := v.(float64)
		cfg.RetrievalTrustBoostDocs = x
	case "retrieval.trust_boost.context":
		x, _ := v.(float64)
		cfg.RetrievalTrustBoostContext = x
	case "retrieval.trust_boost.notes":
		x, _ := v.(float64)
		cfg.RetrievalTrustBoostNotes = x
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
