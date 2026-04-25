package enrich

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/openrouter"
	"github.com/joaoajmatos/oz/internal/semantic"
)

// RetrievedBlock is a retrieval context block for the proposal prompt.
// The cmd layer converts query.ContextBlock to this type so enrich stays
// decoupled from the query package.
type RetrievedBlock struct {
	File    string
	Section string
	Trust   string // "high", "medium", "low"
}

// ProposeOptions configures a concept proposal run.
type ProposeOptions struct {
	Name      string           // required concept name
	Seed      string           // optional seed description
	FromFiles []string         // optional file anchors
	Blocks    []RetrievedBlock // retrieved context; nil when --no-retrieval
	Model     string           // OpenRouter model ID (default: openrouter/free)
	Progress  func(string)     // optional progress callback
}

// ProposeResult is the outcome of a single-concept proposal.
type ProposeResult struct {
	Concept        semantic.Concept
	Edges          []semantic.ConceptEdge
	Skipped        []string
	NearDuplicates []string // existing reviewed concept names that closely match the proposed name
	Cost           float64
	Model          string
	PromptText     string // the full prompt sent to the model
}

// ProposeConcept runs the single-concept proposal pipeline:
//
//  1. Build allowlist and proposal prompt
//  2. Send to OpenRouter
//  3. Parse: require exactly 1 concept (strict)
//  4. Merge with existing semantic.json (new items get reviewed: false)
//  5. Write context/semantic.json
func ProposeConcept(workspacePath string, g *graph.Graph, opts ProposeOptions) (*ProposeResult, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("concept name is required (--name)")
	}
	report := func(stage string) {
		if opts.Progress != nil {
			opts.Progress(stage)
		}
	}

	report("initializing OpenRouter client")
	client, err := openrouter.New(opts.Model)
	if err != nil {
		return nil, err
	}

	report("loading existing semantic overlay")
	existing, err := semantic.Load(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("load existing overlay: %w", err)
	}

	report("building allowlist")
	allowlist := BuildAllowlist(g, existing)

	var existingNames []string
	if existing != nil {
		for _, c := range existing.Concepts {
			if c.Reviewed {
				existingNames = append(existingNames, c.Name)
			}
		}
	}

	report("building proposal prompt")
	prompt, err := BuildProposalPrompt(opts, allowlist, existingNames)
	if err != nil {
		return nil, fmt.Errorf("build proposal prompt: %w", err)
	}

	report("requesting model response")
	resp, err := client.Complete([]openrouter.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}

	// Node ID set for edge validation: graph nodes + allowlist (includes concept IDs).
	nodeIDs := make(map[string]struct{}, len(g.Nodes)+len(allowlist))
	for _, n := range g.Nodes {
		nodeIDs[n.ID] = struct{}{}
	}
	for _, id := range allowlist {
		nodeIDs[id] = struct{}{}
	}

	report("parsing and validating response")
	concept, edges, skipped, err := ParseSingleConcept(resp.Choices[0].Message.Content, nodeIDs)
	if err != nil {
		return nil, err
	}

	report("merging and writing semantic overlay")
	incoming := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     g.ContentHash,
		Model:         client.Model,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Concepts:      []semantic.Concept{concept},
		Edges:         edges,
	}
	merged := semantic.Merge(existing, incoming)
	if err := semantic.Write(workspacePath, merged); err != nil {
		return nil, fmt.Errorf("write semantic.json: %w", err)
	}

	var nearDups []string
	if existing != nil {
		nearDups = findNearDuplicates(opts.Name, existing.Concepts)
	}

	res := &ProposeResult{
		Concept:        concept,
		Edges:          edges,
		Skipped:        skipped,
		NearDuplicates: nearDups,
		Model:          client.Model,
		PromptText:     prompt,
	}
	if resp.Usage != nil {
		res.Cost = resp.Usage.Cost
	}
	return res, nil
}

// BuildAllowlist returns sorted node IDs eligible as edge `to` targets:
// agent, spec_section, decision, code_package nodes, plus reviewed concept IDs
// (for semantically_similar_to edges). existing may be nil.
func BuildAllowlist(g *graph.Graph, existing *semantic.Overlay) []string {
	eligible := map[string]bool{
		graph.NodeTypeAgent:       true,
		graph.NodeTypeSpecSection: true,
		graph.NodeTypeDecision:    true,
		graph.NodeTypeCodePackage: true,
	}
	seen := make(map[string]struct{})
	var ids []string
	for _, n := range g.Nodes {
		if eligible[n.Type] {
			seen[n.ID] = struct{}{}
			ids = append(ids, n.ID)
		}
	}
	if existing != nil {
		for _, c := range existing.Concepts {
			if c.Reviewed {
				if _, ok := seen[c.ID]; !ok {
					seen[c.ID] = struct{}{}
					ids = append(ids, c.ID)
				}
			}
		}
	}
	sort.Strings(ids)
	return ids
}

// BuildProposalPrompt builds the single-concept proposal prompt.
// allowlist is the sorted list of valid `to` IDs (from BuildAllowlist).
// existingConceptNames are reviewed concept names shown as context hints.
func BuildProposalPrompt(opts ProposeOptions, allowlist []string, existingConceptNames []string) (string, error) {
	var sb strings.Builder

	sb.WriteString("You are adding one new concept to an oz semantic overlay.\n\n")
	sb.WriteString("## Concept to propose\n\n")
	sb.WriteString("Name: " + opts.Name + "\n")
	if opts.Seed != "" {
		sb.WriteString("Description seed: " + opts.Seed + "\n")
	}
	if len(opts.FromFiles) > 0 {
		sb.WriteString("Anchored to files:\n")
		for _, f := range opts.FromFiles {
			sb.WriteString("  - " + f + "\n")
		}
	}

	if len(opts.Blocks) > 0 {
		sb.WriteString("\n## Retrieved context (oz context query — no agent routing)\n\n")
		for i, b := range opts.Blocks {
			sb.WriteString(fmt.Sprintf("[%d] %s § %s (trust: %s)\n", i+1, b.File, b.Section, b.Trust))
		}
	}

	sb.WriteString("\n## Valid target node IDs\n\n")
	sb.WriteString("The 'to' field in every edge MUST be one of these exact IDs:\n\n")
	sb.WriteString(strings.Join(allowlist, "\n"))

	if len(existingConceptNames) > 0 {
		sb.WriteString("\n\n## Existing reviewed concepts (context only — not requirements)\n\n")
		for _, n := range existingConceptNames {
			sb.WriteString("- " + n + "\n")
		}
	}

	sb.WriteString("\n\n## Task\n\nPropose exactly ONE concept named \"" + opts.Name + "\".")
	sb.WriteString(" Use the retrieved context as grounding. Return ONLY valid JSON — no markdown fences.\n\n")
	sb.WriteString(`Schema:
{
  "concepts": [
    {
      "id": "concept:<slug>",
      "name": "` + opts.Name + `",
      "description": "<1-2 sentence description>",
      "source_files": ["<workspace-relative paths>"],
      "tag": "EXTRACTED | INFERRED",
      "confidence": 0.0-1.0
    }
  ],
  "edges": [
    {
      "from": "concept:<slug>",
      "to": "<exact ID from valid list above>",
      "type": "agent_owns_concept | implements_spec | implements | semantically_similar_to",
      "tag": "EXTRACTED | INFERRED",
      "confidence": 0.0-1.0
    }
  ]
}

Rules:
- Return exactly ONE concept in the "concepts" array
- id MUST be "concept:" followed by a lowercase-kebab slug of the name
- "to" MUST be an exact ID from the valid list — never a bare file path
- Include 1-4 edges
- Return pure JSON only`)

	return sb.String(), nil
}

// normalizeConceptName lowercases s and strips all non-alphanumeric characters,
// producing a canonical form for near-duplicate comparison.
func normalizeConceptName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	row := make([]int, lb+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := row[0]
		row[0] = i
		for j := 1; j <= lb; j++ {
			tmp := row[j]
			if ra[i-1] == rb[j-1] {
				row[j] = prev
			} else {
				row[j] = 1 + min3(prev, row[j], row[j-1])
			}
			prev = tmp
		}
	}
	return row[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// findNearDuplicates returns the names of reviewed concepts in existing whose
// normalized form either exactly matches norm(proposed) or has edit distance ≤
// threshold (2 for names ≤ 10 chars, 3 for longer names).
func findNearDuplicates(proposed string, existing []semantic.Concept) []string {
	normProp := normalizeConceptName(proposed)
	threshold := 2
	if len(normProp) > 10 {
		threshold = 3
	}
	var matches []string
	for _, c := range existing {
		if !c.Reviewed {
			continue
		}
		normExist := normalizeConceptName(c.Name)
		if normExist == normProp || levenshtein(normProp, normExist) <= threshold {
			matches = append(matches, c.Name)
		}
	}
	return matches
}

// ParseSingleConcept is the strict proposal-mode variant of ParseResponse.
// It returns an error when the model returns zero or more than one concept,
// with a user-actionable message in each case (pre-mortem T4).
func ParseSingleConcept(content string, nodeIDs map[string]struct{}) (semantic.Concept, []semantic.ConceptEdge, []string, error) {
	concepts, edges, skipped := ParseResponse(content, nodeIDs)
	switch len(concepts) {
	case 0:
		if len(skipped) > 0 {
			return semantic.Concept{}, nil, skipped,
				fmt.Errorf("model returned no valid concept: %s", strings.Join(skipped, "; "))
		}
		return semantic.Concept{}, nil, skipped,
			fmt.Errorf("model returned no concepts — try a more specific --name or --seed")
	case 1:
		// Proposal path: force all edges unreviewed so oz context review sees them.
		// ADR-0003 auto-review applies to bulk enrich only, not targeted proposals.
		for i := range edges {
			edges[i].Reviewed = false
		}
		return concepts[0], edges, skipped, nil
	default:
		return semantic.Concept{}, nil, skipped,
			fmt.Errorf("model returned %d concepts, expected exactly 1 — use a more specific --name or --seed", len(concepts))
	}
}
