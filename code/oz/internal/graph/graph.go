// Package graph defines the schema for context/graph.json — the structural
// graph produced by oz context build.
//
// # Node types
//
//   - agent           — an agent from agents/<name>/AGENT.md
//   - spec_section    — a heading from a file under specs/ (excluding decisions/)
//   - decision        — a file under specs/decisions/
//   - doc             — a heading from a file under docs/
//   - context_snapshot — a file under context/
//   - note            — a file under notes/
//
// # Edge types
//
//   - reads            — agent declares a file in its read-chain
//   - owns             — agent declares a scope path
//   - references       — a document contains a file-path reference to another node
//   - supports         — a doc references a spec/decision (doc supports the spec)
//   - crystallized_from — a spec section was crystallized from a note
package graph

import "github.com/oz-tools/oz/internal/convention"

// SchemaVersion is the current graph.json schema version.
// Increment when the schema changes in a backwards-incompatible way.
const SchemaVersion = "1"

// Node types.
const (
	NodeTypeAgent           = "agent"
	NodeTypeSpecSection     = "spec_section"
	NodeTypeDecision        = "decision"
	NodeTypeDoc             = "doc"
	NodeTypeContextSnapshot = "context_snapshot"
	NodeTypeNote            = "note"
)

// Edge types.
const (
	EdgeTypeReads            = "reads"
	EdgeTypeOwns             = "owns"
	EdgeTypeReferences       = "references"
	EdgeTypeSupports         = "supports"
	EdgeTypeCrystallizedFrom = "crystallized_from"
)

// Node is a vertex in the structural graph.
type Node struct {
	// ID is the stable, unique node key. Format: "<type>:<discriminator>".
	//   agent:coding
	//   spec_section:specs/api.md:Overview
	//   decision:0001-use-go
	//   doc:docs/architecture.md:Components
	//   context_snapshot:context/auth/snapshot.md
	//   note:notes/ideas.md
	ID   string `json:"id"`
	Type string `json:"type"`

	// File is the path to the source file, relative to the workspace root.
	File string `json:"file"`

	// Name is a human-readable display name for the node.
	Name string `json:"name"`

	// Tier is the source-of-truth trust band. Omitted for agent nodes.
	Tier convention.Tier `json:"tier,omitempty"`

	// Section is the markdown heading for section nodes (spec_section, doc).
	Section string `json:"section,omitempty"`

	// Agent-specific fields (Type == NodeTypeAgent).
	Role             string   `json:"role,omitempty"`
	Scope            []string `json:"scope,omitempty"`
	Responsibilities string   `json:"responsibilities,omitempty"`
	OutOfScope       string   `json:"out_of_scope,omitempty"`
	ReadChain        []string `json:"read_chain,omitempty"`
	Rules            []string `json:"rules,omitempty"`
	Skills           []string `json:"skills,omitempty"`
	ContextTopics    []string `json:"context_topics,omitempty"`
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// Graph is the full structural graph of an oz workspace.
type Graph struct {
	// SchemaVersion identifies the graph.json format version.
	SchemaVersion string `json:"schema_version"`

	// ContentHash is the SHA-256 of the canonical serialized nodes+edges,
	// computed before embedding this field. It provides a fast change-detection
	// key and is used to link context/semantic.json to the graph it was built from.
	ContentHash string `json:"content_hash"`

	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
