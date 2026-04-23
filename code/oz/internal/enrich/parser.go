package enrich

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joaoajmatos/oz/internal/semantic"
)

// llmResponse is the raw JSON structure the LLM is asked to return.
type llmResponse struct {
	Concepts []semantic.Concept     `json:"concepts"`
	Edges    []semantic.ConceptEdge `json:"edges"`
}

// ParseResponse parses and validates LLM output. It returns the valid
// concepts and edges, plus a slice of skipped-item messages for logging.
//
// Tolerant parsing: malformed items are skipped rather than failing the run.
// The caller receives whatever the LLM got right.
func ParseResponse(content string, graphNodeIDs map[string]struct{}) ([]semantic.Concept, []semantic.ConceptEdge, []string) {
	content = strings.TrimSpace(content)

	// Strip markdown code fences if the LLM wrapped its JSON.
	if strings.HasPrefix(content, "```") {
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) == 2 {
			content = lines[1]
		}
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		content = strings.TrimSpace(content)
	}

	var raw llmResponse
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, nil, []string{fmt.Sprintf("JSON parse error: %v", err)}
	}

	var concepts []semantic.Concept
	var skipped []string
	validConceptIDs := make(map[string]struct{})

	for _, c := range raw.Concepts {
		switch {
		case !strings.HasPrefix(c.ID, "concept:"):
			skipped = append(skipped, fmt.Sprintf("concept %q skipped: id must start with 'concept:'", c.ID))
			continue
		case c.Name == "":
			skipped = append(skipped, fmt.Sprintf("concept %q skipped: missing name", c.ID))
			continue
		}
		if c.Tag != semantic.TagExtracted && c.Tag != semantic.TagInferred {
			c.Tag = semantic.TagExtracted
		}
		if c.Confidence <= 0 || c.Confidence > 1 {
			c.Confidence = 1.0
		}
		c.Reviewed = false
		concepts = append(concepts, c)
		validConceptIDs[c.ID] = struct{}{}
	}

	// autoReviewThreshold is the minimum confidence for an edge to be auto-promoted
	// to reviewed:true without human approval. See ADR-0003.
	const autoReviewThreshold = 0.85

	validEdgeTypes := map[string]bool{
		semantic.EdgeTypeAgentOwnsConcept:      true,
		semantic.EdgeTypeImplementsSpec:        true,
		semantic.EdgeTypeDriftedFrom:           true,
		semantic.EdgeTypeSemanticallySimilarTo: true,
		semantic.EdgeTypeImplements:            true,
	}

	var edges []semantic.ConceptEdge
	for _, e := range raw.Edges {
		// "from" must be a known concept.
		if _, ok := validConceptIDs[e.From]; !ok {
			skipped = append(skipped, fmt.Sprintf("edge %s→%s skipped: unknown from-concept %q", e.From, e.To, e.From))
			continue
		}
		// "to" must be either a concept or a structural graph node.
		_, toIsConcept := validConceptIDs[e.To]
		_, toIsGraphNode := graphNodeIDs[e.To]
		if !toIsConcept && !toIsGraphNode {
			skipped = append(skipped, fmt.Sprintf("edge %s→%s skipped: unknown to-node %q", e.From, e.To, e.To))
			continue
		}
		if !validEdgeTypes[e.Type] {
			skipped = append(skipped, fmt.Sprintf("edge %s→%s skipped: unknown type %q", e.From, e.To, e.Type))
			continue
		}
		if e.Tag != semantic.TagExtracted && e.Tag != semantic.TagInferred {
			e.Tag = semantic.TagExtracted
		}
		if e.Confidence <= 0 || e.Confidence > 1 {
			e.Confidence = 1.0
		}
		// Auto-promote high-confidence edges per ADR-0003.
		e.Reviewed = e.Confidence >= autoReviewThreshold
		edges = append(edges, e)
	}

	return concepts, edges, skipped
}
