package semantic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

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

// ConceptsForAgent returns the names of all concepts owned by agentName.
// An agent "owns" a concept when an agent_owns_concept edge exists with
// From=concept:<slug> and To=agent:<name>.
// Returns nil when the overlay is nil or the agent owns no concepts.
func ConceptsForAgent(o *Overlay, agentName string) []string {
	if o == nil {
		return nil
	}
	agentNodeID := "agent:" + agentName
	ownedIDs := make(map[string]struct{})
	for _, e := range o.Edges {
		if e.Type == EdgeTypeAgentOwnsConcept && e.To == agentNodeID {
			ownedIDs[e.From] = struct{}{}
		}
	}
	if len(ownedIDs) == 0 {
		return nil
	}
	var names []string
	for _, c := range o.Concepts {
		if _, ok := ownedIDs[c.ID]; ok {
			names = append(names, c.Name)
		}
	}
	sort.Strings(names)
	return names
}
