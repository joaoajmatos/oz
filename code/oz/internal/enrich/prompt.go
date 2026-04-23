package enrich

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/joaoajmatos/oz/internal/graph"
)

// promptAgent is the agent representation sent to the LLM.
type promptAgent struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Role             string   `json:"role,omitempty"`
	Scope            []string `json:"scope,omitempty"`
	Responsibilities string   `json:"responsibilities,omitempty"`
}

// promptSection is the spec/doc section representation sent to the LLM.
type promptSection struct {
	ID      string `json:"id"`
	File    string `json:"file"`
	Section string `json:"section"`
}

// promptDecision is the decision representation sent to the LLM.
type promptDecision struct {
	ID   string `json:"id"`
	File string `json:"file"`
}

// promptPackage is the code package representation sent to the LLM.
type promptPackage struct {
	ID         string   `json:"id"`
	Package    string   `json:"package"`
	DocComment string   `json:"doc_comment,omitempty"`
	Symbols    []string `json:"symbols,omitempty"`
}

// promptGraph is the condensed graph sent to the LLM.
// Only structural nodes needed for concept extraction are included.
type promptGraph struct {
	Agents       []promptAgent    `json:"agents"`
	SpecSections []promptSection  `json:"spec_sections,omitempty"`
	Decisions    []promptDecision `json:"decisions,omitempty"`
	DocSections  []promptSection  `json:"doc_sections,omitempty"`
	Packages     []promptPackage  `json:"code_packages,omitempty"`
}

func buildPromptGraph(g *graph.Graph) promptGraph {
	// Collect symbol names per package import path for code_packages.
	pkgSymbols := map[string][]string{}
	for _, n := range g.Nodes {
		if n.Type == graph.NodeTypeCodeSymbol && n.Package != "" {
			pkgSymbols[n.Package] = append(pkgSymbols[n.Package], n.Name)
		}
	}

	var pg promptGraph
	for _, n := range g.Nodes {
		switch n.Type {
		case graph.NodeTypeAgent:
			pg.Agents = append(pg.Agents, promptAgent{
				ID:               n.ID,
				Name:             n.Name,
				Role:             n.Role,
				Scope:            n.Scope,
				Responsibilities: n.Responsibilities,
			})
		case graph.NodeTypeSpecSection:
			pg.SpecSections = append(pg.SpecSections, promptSection{
				ID:      n.ID,
				File:    n.File,
				Section: n.Section,
			})
		case graph.NodeTypeDecision:
			pg.Decisions = append(pg.Decisions, promptDecision{
				ID:   n.ID,
				File: n.File,
			})
		case graph.NodeTypeDoc:
			pg.DocSections = append(pg.DocSections, promptSection{
				ID:      n.ID,
				File:    n.File,
				Section: n.Section,
			})
		case graph.NodeTypeCodePackage:
			syms := pkgSymbols[n.Package]
			sort.Strings(syms)
			pg.Packages = append(pg.Packages, promptPackage{
				ID:         n.ID,
				Package:    n.Package,
				DocComment: n.DocComment,
				Symbols:    syms,
			})
		}
	}
	// Sort all lists for determinism.
	sort.Slice(pg.Agents, func(i, j int) bool { return pg.Agents[i].ID < pg.Agents[j].ID })
	sort.Slice(pg.SpecSections, func(i, j int) bool { return pg.SpecSections[i].ID < pg.SpecSections[j].ID })
	sort.Slice(pg.Decisions, func(i, j int) bool { return pg.Decisions[i].ID < pg.Decisions[j].ID })
	sort.Slice(pg.DocSections, func(i, j int) bool { return pg.DocSections[i].ID < pg.DocSections[j].ID })
	sort.Slice(pg.Packages, func(i, j int) bool { return pg.Packages[i].ID < pg.Packages[j].ID })
	return pg
}

// BuildPrompt constructs the LLM enrichment prompt for the given graph.
// The prompt is deterministic: the same graph always produces the same prompt.
func BuildPrompt(g *graph.Graph) (string, error) {
	pg := buildPromptGraph(g)
	graphJSON, err := json.MarshalIndent(pg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal prompt graph: %w", err)
	}

	// Build explicit list of valid target node IDs to prevent the LLM from
	// inventing file-path references instead of using real graph node IDs.
	var validIDs []string
	for _, n := range g.Nodes {
		switch n.Type {
		case graph.NodeTypeAgent, graph.NodeTypeSpecSection,
			graph.NodeTypeDecision, graph.NodeTypeCodePackage:
			validIDs = append(validIDs, n.ID)
		}
	}
	sort.Strings(validIDs)
	validIDList := strings.Join(validIDs, "\n")

	return fmt.Sprintf(`You are analyzing an oz workspace structural graph to extract semantic concepts.

Source-of-truth ranking (highest to lowest): specs > docs > code.
When a doc-derived concept conflicts with a spec-derived one, the spec version takes precedence.

## Workspace Graph

%s

## Valid target node IDs

The "to" field in every edge MUST be one of these exact IDs (copy-paste, do not invent):

%s

## Task

1. Identify abstract concepts that span multiple files or agents.
2. For each concept, create edges using these rules:
   - agent_owns_concept: from=concept:<slug>, to=agent:<name> — which agent owns this concept
   - implements_spec: from=concept:<slug>, to=<spec_section or decision id> — spec that defines it
   - implements: from=concept:<slug>, to=<code_package id> — package that implements it

Return ONLY a valid JSON object — no markdown fences, no explanation outside the JSON.

Schema:
{
  "concepts": [
    {
      "id": "concept:<slug>",
      "name": "<human-readable name>",
      "description": "<1-2 sentence description>",
      "source_files": ["<file paths>"],
      "tag": "EXTRACTED",
      "confidence": 1.0
    }
  ],
  "edges": [
    {
      "from": "<concept-id>",
      "to": "<exact node ID from the valid list above>",
      "type": "agent_owns_concept | implements_spec | implements | drifted_from | semantically_similar_to",
      "tag": "EXTRACTED | INFERRED",
      "confidence": 0.0-1.0
    }
  ]
}

Rules:
- All concept IDs must start with "concept:"
- The "to" field MUST be an exact ID from the valid node ID list above — never a file path
- Tag EXTRACTED for directly observable facts (confidence ≥ 0.9)
- Tag INFERRED for reasonable inferences (confidence 0.5–0.89)
- Extract 5–20 concepts; skip low-confidence inferences
- Return pure JSON only`, string(graphJSON), validIDList), nil
}
