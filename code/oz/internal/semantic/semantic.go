// Package semantic defines the schema for context/semantic.json — the LLM-produced
// semantic overlay that annotates the structural graph with extracted concepts and
// typed relationships.
//
// # Concept nodes
//
//   - Concept — an abstract idea that spans multiple files or agents.
//     Tag EXTRACTED for directly observable facts; INFERRED for reasonable inferences.
//
// # Edge types
//
//   - agent_owns_concept       — an agent is primarily responsible for a concept
//   - implements_spec          — a concept is implemented by a spec section or decision
//   - drifted_from             — code has diverged from the concept's spec
//   - semantically_similar_to  — two concepts are closely related
//
// # Human review
//
// Every concept and edge carries a Reviewed field. When Reviewed is false the item
// has not been accepted by a human. Run 'oz context review' to mark items as reviewed.
// 'oz validate' warns when unreviewed items are present.
package semantic

// SchemaVersion is the current semantic.json schema version.
const SchemaVersion = "1"

// Tag values for concepts and edges.
const (
	TagExtracted = "EXTRACTED" // directly observable fact
	TagInferred  = "INFERRED"  // reasonable inference
)

// Edge type constants.
const (
	EdgeTypeAgentOwnsConcept      = "agent_owns_concept"
	EdgeTypeImplementsSpec        = "implements_spec"
	EdgeTypeDriftedFrom           = "drifted_from"
	EdgeTypeSemanticallySimilarTo = "semantically_similar_to"
)

// Overlay is the full contents of context/semantic.json.
type Overlay struct {
	// SchemaVersion identifies the semantic.json format version.
	SchemaVersion string `json:"schema_version"`

	// GraphHash is the ContentHash of the graph.json this overlay was built from.
	// Used for staleness detection: if it differs from the current graph.json hash,
	// the overlay may be stale and should be refreshed via 'oz context enrich'.
	GraphHash string `json:"graph_hash"`

	// Model is the OpenRouter model ID used to generate this overlay.
	Model string `json:"model,omitempty"`

	// GeneratedAt is the RFC3339 UTC timestamp of the last enrichment run.
	GeneratedAt string `json:"generated_at,omitempty"`

	Concepts []Concept     `json:"concepts"`
	Edges    []ConceptEdge `json:"edges"`
}

// Concept is an abstract idea extracted from the workspace.
type Concept struct {
	// ID is the stable unique key. Format: "concept:<slug>".
	ID string `json:"id"`

	// Name is a human-readable concept label.
	Name string `json:"name"`

	// Description is a 1-2 sentence summary of the concept.
	Description string `json:"description,omitempty"`

	// SourceFiles lists file paths (relative to workspace root) where this concept
	// was observed.
	SourceFiles []string `json:"source_files,omitempty"`

	// Tag is EXTRACTED or INFERRED.
	Tag string `json:"tag"`

	// Confidence is a 0.0–1.0 score from the LLM.
	// EXTRACTED items typically carry 1.0; INFERRED items carry < 1.0.
	Confidence float64 `json:"confidence"`

	// Reviewed is set to true when a human has accepted this concept via
	// 'oz context review'. Merge preserves reviewed items across re-enrichment.
	Reviewed bool `json:"reviewed"`
}

// ConceptEdge is a typed relationship in the semantic overlay.
type ConceptEdge struct {
	// From is the source node ID (always a concept ID: "concept:<slug>").
	From string `json:"from"`

	// To is the target node ID (a concept ID or a structural graph node ID).
	To string `json:"to"`

	// Type is one of the EdgeType* constants.
	Type string `json:"type"`

	Tag        string  `json:"tag"`
	Confidence float64 `json:"confidence"`
	Reviewed   bool    `json:"reviewed"`
}
