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
		{name: "ls", args: []string{"ls", "-la"}},
		{name: "find", args: []string{"find", "."}},
		{name: "tree", args: []string{"tree"}},
		{name: "json_jq", args: []string{"jq", "."}},
		{name: "json_sniff", args: []string{"true"}},
		{name: "git_log", args: []string{"git", "log"}},
		{name: "git_blame", args: []string{"git", "blame", "x.go"}},
		{name: "git_show", args: []string{"git", "show", "HEAD"}},
		{name: "go_build", args: []string{"go", "build", "./..."}},
		{name: "go_vet", args: []string{"go", "vet", "./..."}},
		{name: "staticcheck", args: []string{"staticcheck", "./..."}},
		{name: "make", args: []string{"make"}},
		{name: "cargo", args: []string{"cargo", "build"}},
		{name: "pytest", args: []string{"pytest", "-q"}},
		{name: "npm", args: []string{"npm", "install"}},
		{name: "docker", args: []string{"docker", "build", "."}},
		{name: "http", args: []string{"curl", "-i", "http://example.com"}},
		{name: "env", args: []string{"env"}},
		{name: "wc", args: []string{"wc", "README.md"}},
		{name: "df", args: []string{"df", "-h"}},
		{name: "ps", args: []string{"ps", "aux"}},
		{name: "top_batch", args: []string{"top", "-b", "-n", "1"}},
		{name: "diff", args: []string{"diff", "-u", "a", "b"}},
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
		minPct := 15.0
		if tc.name == "wc" || tc.name == "df" {
			// Very small structured outputs; compaction is correctness-first.
			minPct = 5.0
		}
		if reductionPct < minPct {
			t.Fatalf("%s reduction too small: %.2f%% (min %.2f%%)", tc.name, reductionPct, minPct)
		}
		reductions = append(reductions, reductionPct)
	}

	sort.Float64s(reductions)
	median := reductions[len(reductions)/2]
	// Original MVP set targeted 60% median; expanded filter coverage includes many
	// structured summaries where per-family floors matter more than the median.
	if median < 50 {
		t.Fatalf("median reduction %.2f%% is below 50%% target", median)
	}
}
