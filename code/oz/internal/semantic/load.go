package semantic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// IsStale reports whether the overlay was built from a different graph than
// the one identified by currentGraphHash.
// Returns false when the overlay is nil or has no recorded hash.
func IsStale(o *Overlay, currentGraphHash string) bool {
	return o != nil && o.GraphHash != "" && o.GraphHash != currentGraphHash
}

// Load reads context/semantic.json from the workspace root.
// Returns nil, nil if the file does not exist (no overlay present).
func Load(workspacePath string) (*Overlay, error) {
	path := filepath.Join(workspacePath, "context", "semantic.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var o Overlay
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("parse semantic.json: %w", err)
	}
	return &o, nil
}

// ConceptsForAgent returns the names of all reviewed concepts owned by agentName.
// An agent "owns" a concept when a reviewed agent_owns_concept edge exists with
// From=concept:<slug> and To=agent:<name>.
// Unreviewed edges are excluded — they must be accepted via 'oz context review' first.
// Returns nil when the overlay is nil or the agent owns no reviewed concepts.
func ConceptsForAgent(o *Overlay, agentName string) []string {
	if o == nil {
		return nil
	}
	agentNodeID := "agent:" + agentName
	ownedIDs := make(map[string]struct{})
	for _, e := range o.Edges {
		if e.Type == EdgeTypeAgentOwnsConcept && e.To == agentNodeID && e.Reviewed {
			ownedIDs[e.From] = struct{}{}
		}
	}
	if len(ownedIDs) == 0 {
		return nil
	}
	var names []string
	for _, c := range o.Concepts {
		if _, ok := ownedIDs[c.ID]; ok && c.Reviewed {
			names = append(names, c.Name)
		}
	}
	sort.Strings(names)
	return names
}

// PackagesForConcept returns the code_package node IDs that implement conceptID,
// restricted to reviewed implements edges. Unreviewed edges are never traversed
// by query — they are only visible to 'oz audit'. See ADR-0003.
func PackagesForConcept(o *Overlay, conceptID string) []string {
	if o == nil {
		return nil
	}
	var pkgs []string
	for _, e := range o.Edges {
		if e.Type == EdgeTypeImplements && e.From == conceptID && e.Reviewed {
			pkgs = append(pkgs, e.To)
		}
	}
	sort.Strings(pkgs)
	return pkgs
}
