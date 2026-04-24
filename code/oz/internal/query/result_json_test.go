package query

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultJSON_OmitsEmptyCodeEntryPoints(t *testing.T) {
	result := Result{
		Agent:      "oz-coding",
		Confidence: 0.9,
	}

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "code_entry_points") {
		t.Fatalf("expected code_entry_points to be omitted, got %s", s)
	}
}

func TestResultJSON_EmitsCodeEntryPointsWhenPresent(t *testing.T) {
	result := Result{
		Agent:      "oz-coding",
		Confidence: 0.9,
		CodeEntryPoints: []CodeEntryPoint{
			{
				File:      "code/oz/internal/audit/drift/run.go",
				Symbol:    "drift.Run",
				Kind:      "func",
				Line:      42,
				Package:   "audit/drift",
				Relevance: 1.23,
			},
		},
	}

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "code_entry_points") {
		t.Fatalf("expected code_entry_points to be present, got %s", s)
	}
}
