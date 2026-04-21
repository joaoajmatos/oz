package enrich

import (
	"encoding/json"
	"fmt"
	"sort"

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

// promptSection is the spec section representation sent to the LLM.
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

// promptGraph is the condensed graph sent to the LLM.
// Only structural nodes needed for concept extraction are included.
type promptGraph struct {
	Agents       []promptAgent    `json:"agents"`
	SpecSections []promptSection  `json:"spec_sections,omitempty"`
	Decisions    []promptDecision `json:"decisions,omitempty"`
}

func buildPromptGraph(g *graph.Graph) promptGraph {
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
		}
	}
	// Sort all lists for determinism.
	sort.Slice(pg.Agents, func(i, j int) bool { return pg.Agents[i].ID < pg.Agents[j].ID })
	sort.Slice(pg.SpecSections, func(i, j int) bool { return pg.SpecSections[i].ID < pg.SpecSections[j].ID })
	sort.Slice(pg.Decisions, func(i, j int) bool { return pg.Decisions[i].ID < pg.Decisions[j].ID })
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
	return fmt.Sprintf(`You are analyzing an oz workspace structural graph to extract semantic concepts.

## Workspace Graph

%s

## Task

Identify abstract concepts that span multiple files or agents. For each concept:
1. Determine which agent is primarily responsible for it (agent_owns_concept edge)
2. Identify which spec sections implement or relate to it (implements_spec edge)

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
      "to": "<graph-node-id>",
      "type": "agent_owns_concept | implements_spec | drifted_from | semantically_similar_to",
      "tag": "EXTRACTED | INFERRED",
      "confidence": 0.0-1.0
    }
  ]
}

Rules:
- All concept IDs must start with "concept:"
- agent_owns_concept: from=concept:<slug>, to=agent:<name>
- implements_spec: from=concept:<slug>, to=spec_section or decision node ID
- Tag EXTRACTED for directly observable facts (confidence ≥ 0.9)
- Tag INFERRED for reasonable inferences (confidence 0.5–0.89)
- Extract 5–20 concepts; skip low-confidence inferences
- Return pure JSON only`, string(graphJSON)), nil
}
