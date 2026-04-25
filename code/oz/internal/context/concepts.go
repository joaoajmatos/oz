package context

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/joaoajmatos/oz/internal/codeindex"
)

const conceptsSchemaVersion = "1"

// ConceptsFile is the schema for context/code-concepts.json.
type ConceptsFile struct {
	SchemaVersion string                 `json:"schema_version"`
	GraphHash     string                 `json:"graph_hash"`
	Concepts      []codeindex.CodeConcept `json:"concepts"`
}

// WriteConcepts serializes the collected code concepts to context/code-concepts.json.
// An empty concepts slice is valid and produces a file with an empty array.
func WriteConcepts(root string, concepts []codeindex.CodeConcept, graphHash string) error {
	if concepts == nil {
		concepts = []codeindex.CodeConcept{}
	}
	cf := ConceptsFile{
		SchemaVersion: conceptsSchemaVersion,
		GraphHash:     graphHash,
		Concepts:      concepts,
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dest := filepath.Join(root, "context", "code-concepts.json")
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

// LoadConcepts reads context/code-concepts.json from the workspace root.
func LoadConcepts(root string) (*ConceptsFile, error) {
	data, err := os.ReadFile(filepath.Join(root, "context", "code-concepts.json"))
	if err != nil {
		return nil, err
	}
	var cf ConceptsFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, err
	}
	return &cf, nil
}
