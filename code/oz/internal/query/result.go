// Package query implements the oz context query engine.
// The BM25F scoring engine is implemented in Sprint 3.
// This file defines the Result type used by the testws framework and the CLI.
package query

// Result is the routing packet returned by a query.
type Result struct {
	// Agent is the name of the agent that owns this task.
	// Empty string means no clear owner.
	Agent string `json:"agent"`

	// Confidence is a 0.0–1.0 score. Values below 0.7 indicate ambiguity.
	Confidence float64 `json:"confidence"`

	// Scope is the list of file paths within this agent's declared scope
	// that are relevant to the task.
	Scope []string `json:"scope,omitempty"`

	// ContextBlocks are the workspace files/sections to load, ordered by
	// source-of-truth tier (specs > docs > context > notes).
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`

	// RelevantConcepts lists concept node names from the semantic overlay
	// that are relevant to this task. Omitted when no overlay is present.
	RelevantConcepts []string `json:"relevant_concepts,omitempty"`

	// Excluded lists path prefixes that were filtered from context blocks.
	// notes/ is excluded by default unless --include-notes is set.
	Excluded []string `json:"excluded,omitempty"`

	// Reason is set when Agent is empty. Currently: "no_clear_owner".
	Reason string `json:"reason,omitempty"`

	// CandidateAgents is populated when Confidence < 0.7, listing all agents
	// with confidence >= 0.2, sorted descending.
	CandidateAgents []CandidateAgent `json:"candidate_agents,omitempty"`
}

// ContextBlock is a file/section pair with its source-of-truth trust tier.
type ContextBlock struct {
	File    string `json:"file"`
	Section string `json:"section"`
	Trust   string `json:"trust"` // "high" (specs), "medium" (docs/context), "low" (notes)
}

// CandidateAgent is an agent with its confidence score for ambiguous results.
type CandidateAgent struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// Run executes a query against the oz workspace at workspacePath.
// Stub — full BM25F implementation in Sprint 3.
func Run(workspacePath, queryText string) Result {
	return Result{}
}
