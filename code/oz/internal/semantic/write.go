package semantic

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Write serializes the overlay to context/semantic.json atomically.
// It creates context/ if it does not exist.
func Write(workspacePath string, o *Overlay) error {
	path := filepath.Join(workspacePath, "context", "semantic.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	// Write atomically: write to a temp file in the same directory, then rename.
	tmp, err := os.CreateTemp(filepath.Dir(path), "semantic-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Merge produces a new Overlay by combining incoming with reviewed items
// preserved from existing. existing may be nil.
//
// Strategy:
//   - incoming's graph_hash, model, generated_at replace existing's metadata.
//   - All incoming concepts and edges are included.
//   - Reviewed concepts/edges from existing that are NOT present in incoming
//     are appended (human-accepted knowledge survives re-enrichment).
func Merge(existing, incoming *Overlay) *Overlay {
	result := &Overlay{
		SchemaVersion: incoming.SchemaVersion,
		GraphHash:     incoming.GraphHash,
		Model:         incoming.Model,
		GeneratedAt:   incoming.GeneratedAt,
		Concepts:      append([]Concept(nil), incoming.Concepts...),
		Edges:         append([]ConceptEdge(nil), incoming.Edges...),
	}
	if existing == nil {
		return result
	}

	// Index incoming concept IDs so we can skip duplicates.
	incomingConceptIDs := make(map[string]struct{}, len(incoming.Concepts))
	for _, c := range incoming.Concepts {
		incomingConceptIDs[c.ID] = struct{}{}
	}
	for _, c := range existing.Concepts {
		if c.Reviewed {
			if _, exists := incomingConceptIDs[c.ID]; !exists {
				result.Concepts = append(result.Concepts, c)
			}
		}
	}

	// Index incoming edges.
	incomingEdgeKeys := make(map[string]struct{}, len(incoming.Edges))
	for _, e := range incoming.Edges {
		incomingEdgeKeys[edgeKey(e)] = struct{}{}
	}
	for _, e := range existing.Edges {
		if e.Reviewed {
			if _, exists := incomingEdgeKeys[edgeKey(e)]; !exists {
				result.Edges = append(result.Edges, e)
			}
		}
	}

	return result
}

func edgeKey(e ConceptEdge) string {
	return e.From + "|" + e.Type + "|" + e.To
}
