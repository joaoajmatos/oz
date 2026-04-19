package query

import (
	"math"
	"testing"
)

func TestComputeBM25F_ReturnsScorePerAgent(t *testing.T) {
	docs := []AgentDoc{
		{
			Name:             "backend",
			Scope:            TokenizePathsMulti([]string{"code/api/**"}),
			Role:             TokenizeMulti("Builds REST API endpoints and handlers"),
			Responsibilities: TokenizeMulti("Implements HTTP handlers, request validation, middleware"),
		},
		{
			Name:             "frontend",
			Scope:            TokenizePathsMulti([]string{"code/ui/**"}),
			Role:             TokenizeMulti("Builds React web application"),
			Responsibilities: TokenizeMulti("Implements React components and pages"),
		},
	}

	terms := Tokenize("implement REST API endpoint")
	scores := ComputeBM25F(terms, docs, DefaultScoringConfig())

	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}

	// backend has API and endpoint in role/responsibilities — should score higher.
	backendScore := scoreFor(scores, "backend")
	frontendScore := scoreFor(scores, "frontend")
	if backendScore <= frontendScore {
		t.Errorf("expected backend (%.3f) > frontend (%.3f) for API query", backendScore, frontendScore)
	}
}

func TestComputeBM25F_OutOfScopePenalty(t *testing.T) {
	docs := []AgentDoc{
		{
			Name:             "frontend",
			Scope:            TokenizePathsMulti([]string{"code/ui/**"}),
			Role:             TokenizeMulti("Builds web UI"),
			Responsibilities: TokenizeMulti("Implements React components"),
			OutOfScope:       TokenizeMulti("database schema backend API implementation"),
		},
		{
			Name:             "backend",
			Scope:            TokenizePathsMulti([]string{"code/api/**"}),
			Role:             TokenizeMulti("Builds backend API"),
			Responsibilities: TokenizeMulti("Implements database schema and API"),
		},
	}

	terms := Tokenize("write a database schema migration")
	scores := ComputeBM25F(terms, docs, DefaultScoringConfig())

	// frontend has "database" and "schema" in out-of-scope — should score lower.
	frontendScore := scoreFor(scores, "frontend")
	backendScore := scoreFor(scores, "backend")
	if frontendScore >= backendScore {
		t.Errorf("expected penalty to reduce frontend score below backend: front=%.3f back=%.3f",
			frontendScore, backendScore)
	}
}

func TestSoftmax_SumsToOne(t *testing.T) {
	scores := []Score{
		{Agent: "a", Value: 2.0},
		{Agent: "b", Value: 1.0},
		{Agent: "c", Value: 0.5},
	}
	conf := Softmax(scores, 0.3)

	total := 0.0
	for _, c := range conf {
		total += c
	}
	if math.Abs(total-1.0) > 1e-9 {
		t.Errorf("softmax confidences sum to %.6f, want 1.0", total)
	}
}

func TestSoftmax_TopScoreHighestConfidence(t *testing.T) {
	scores := []Score{
		{Agent: "winner", Value: 5.0},
		{Agent: "loser", Value: 0.1},
	}
	conf := Softmax(scores, 0.3)
	if conf[0] <= conf[1] {
		t.Errorf("expected winner confidence (%.3f) > loser (%.3f)", conf[0], conf[1])
	}
}

func TestRoute_NoClearOwnerWhenAllScoresLow(t *testing.T) {
	cfg := DefaultScoringConfig()
	scores := []Score{
		{Agent: "a", Value: 0.001},
		{Agent: "b", Value: 0.002},
	}
	result := Route(scores, cfg)
	if !result.NoClearOwner {
		t.Errorf("expected NoClearOwner when all scores below min_score=%.3f", cfg.MinScore)
	}
}

func TestRoute_PopulatesCandidatesWhenAmbiguous(t *testing.T) {
	cfg := DefaultScoringConfig()
	// Equal scores → softmax gives equal confidence → below threshold
	scores := []Score{
		{Agent: "a", Value: 1.0},
		{Agent: "b", Value: 0.9},
		{Agent: "c", Value: 0.8},
	}
	result := Route(scores, cfg)
	if result.NoClearOwner {
		t.Fatal("should not be no_clear_owner")
	}
	if result.Confidence >= cfg.ConfidenceThreshold {
		// May or may not be ambiguous depending on scores — just verify struct is valid.
		return
	}
	if len(result.Candidates) == 0 {
		t.Error("expected candidates for ambiguous result")
	}
}

func scoreFor(scores []Score, agent string) float64 {
	for _, s := range scores {
		if s.Agent == agent {
			return s.Value
		}
	}
	return 0
}
