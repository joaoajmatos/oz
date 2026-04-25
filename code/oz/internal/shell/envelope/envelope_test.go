package envelope

import (
	"encoding/json"
	"testing"
)

func TestRunResultJSONFieldNames(t *testing.T) {
	t.Parallel()

	raw := "raw.txt"
	result := RunResult{
		SchemaVersion:     "1",
		Command:           "echo hello",
		Mode:              "compact",
		MatchedFilter:     "generic",
		ExitCode:          0,
		DurationMs:        12,
		TokenEstBefore:    10,
		TokenEstAfter:     4,
		TokenEstSaved:     6,
		TokenReductionPct: 60.0,
		Stdout:            "hello",
		Stderr:            "",
		Warnings:          []string{"warning"},
		RawOutputRef:      &raw,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal into map: %v", err)
	}

	wantKeys := []string{
		"schema_version",
		"command",
		"mode",
		"matched_filter",
		"exit_code",
		"duration_ms",
		"token_est_before",
		"token_est_after",
		"token_est_saved",
		"token_reduction_pct",
		"stdout",
		"stderr",
		"warnings",
		"raw_output_ref",
	}
	for _, key := range wantKeys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected key %q to be present", key)
		}
	}
}

func TestRunResultRawOutputRefNullable(t *testing.T) {
	t.Parallel()

	withNil := RunResult{RawOutputRef: nil}
	nilData, err := json.Marshal(withNil)
	if err != nil {
		t.Fatalf("marshal nil: %v", err)
	}

	var nilPayload map[string]any
	if err := json.Unmarshal(nilData, &nilPayload); err != nil {
		t.Fatalf("unmarshal nil payload: %v", err)
	}
	if value, ok := nilPayload["raw_output_ref"]; !ok || value != nil {
		t.Fatalf("expected raw_output_ref to be null, got %#v", value)
	}

	ref := "raw.log"
	withRef := RunResult{RawOutputRef: &ref}
	refData, err := json.Marshal(withRef)
	if err != nil {
		t.Fatalf("marshal with ref: %v", err)
	}

	var refPayload map[string]any
	if err := json.Unmarshal(refData, &refPayload); err != nil {
		t.Fatalf("unmarshal ref payload: %v", err)
	}
	if got, ok := refPayload["raw_output_ref"].(string); !ok || got != ref {
		t.Fatalf("expected raw_output_ref=%q, got %#v", ref, refPayload["raw_output_ref"])
	}
}

func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty", in: "", want: 0},
		{name: "exact_chunk", in: "abcd", want: 1},
		{name: "round_up", in: "abcde", want: 2},
		{name: "hello_world", in: "hello world", want: 3},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := EstimateTokens(tc.in); got != tc.want {
				t.Fatalf("EstimateTokens(%q)=%d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
