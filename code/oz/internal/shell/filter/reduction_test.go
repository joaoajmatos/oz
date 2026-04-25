package filter_test

import (
	"sort"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/envelope"
	"github.com/joaoajmatos/oz/internal/shell/filter"
	"github.com/joaoajmatos/oz/internal/shell/testfixture"
)

func TestTokenReductionFromFixtures(t *testing.T) {
	t.Parallel()

	fixture := testfixture.New("testdata")
	cases := []struct {
		name string
		args []string
	}{
		{name: "git_status", args: []string{"git", "status"}},
		{name: "git_diff", args: []string{"git", "diff"}},
		{name: "rg", args: []string{"rg", "TODO"}},
		{name: "go_test", args: []string{"go", "test", "./..."}},
	}

	reductions := make([]float64, 0, len(cases))
	for _, tc := range cases {
		input, _ := fixture.Load(t, tc.name)
		compactOut, compactErr, _, err := filter.Apply(tc.args, input, "", 0, false)
		if err != nil {
			t.Fatalf("%s filter failed: %v", tc.name, err)
		}
		before := envelope.EstimateTokens(input)
		after := envelope.EstimateTokens(compactOut + compactErr)
		if before == 0 {
			t.Fatalf("%s fixture has empty input", tc.name)
		}
		if after >= before {
			t.Fatalf("%s should reduce tokens: before=%d after=%d", tc.name, before, after)
		}
		reductionPct := (float64(before-after) / float64(before)) * 100
		if reductionPct < 15 {
			t.Fatalf("%s reduction too small: %.2f%%", tc.name, reductionPct)
		}
		reductions = append(reductions, reductionPct)
	}

	sort.Float64s(reductions)
	median := reductions[len(reductions)/2]
	if median < 60 {
		t.Fatalf("median reduction %.2f%% is below 60%% target", median)
	}
}
