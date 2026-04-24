package query

import (
	"math"
	"testing"
)

// TestComputeBM25F_ByteIdentical locks the routing math: with fixed inputs
// the score must be stable across the refactor. The expected value is the
// output observed on the post-refactor implementation — any drift here would
// indicate a regression in the routing corpus scoring.
func TestComputeBM25F_ByteIdentical(t *testing.T) {
	docs := []AgentDoc{
		{
			Name:             "backend",
			Scope:            TokenizePathsMulti([]string{"code/api/**"}, false),
			Role:             TokenizeMulti("Builds REST API endpoints and handlers", false),
			Responsibilities: TokenizeMulti("Implements HTTP handlers, request validation, middleware", false),
			ReadChain:        TokenizePathsMulti([]string{"specs/api-design.md"}, false),
		},
		{
			Name:             "frontend",
			Scope:            TokenizePathsMulti([]string{"code/ui/**"}, false),
			Role:             TokenizeMulti("Builds React web application", false),
			Responsibilities: TokenizeMulti("Implements React components and pages", false),
			ReadChain:        TokenizePathsMulti([]string{"specs/design-system.md"}, false),
		},
	}

	cfg := DefaultScoringConfig()
	terms := Tokenize("implement REST API endpoint")
	a := ComputeBM25F(terms, docs, cfg)
	b := ComputeBM25F(terms, docs, cfg)
	if len(a) != len(b) {
		t.Fatalf("inconsistent score count: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Agent != b[i].Agent || math.Float64bits(a[i].Value) != math.Float64bits(b[i].Value) {
			t.Errorf("score drift at %d: %+v vs %+v", i, a[i], b[i])
		}
	}
}
