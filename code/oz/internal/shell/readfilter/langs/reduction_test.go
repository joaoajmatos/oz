package langs

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/envelope"
	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

func TestLanguageFixturesGoldenOutput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  string
		expect string
		reader readfilter.LanguageReader
	}{
		{name: "go", input: "go/input.go", expect: "go/expected.txt", reader: golangReader{}},
		{name: "markdown", input: "markdown/input.md", expect: "markdown/expected.txt", reader: markdownReader{}},
		{name: "json", input: "json/input.json", expect: "json/expected.txt", reader: jsonReader{}},
		{name: "generic", input: "generic/input.txt", expect: "generic/expected.txt", reader: genericReader{}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := readFixture(t, tc.input)
			want := readFixture(t, tc.expect)
			got, err := tc.reader.Filter(in, readfilter.Options{})
			if err != nil {
				t.Fatalf("Filter returned error: %v", err)
			}
			if strings.TrimSpace(got) != strings.TrimSpace(want) {
				t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tc.name, got, want)
			}
		})
	}
}

func TestLanguageReductionFloors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  string
		reader readfilter.LanguageReader
	}{
		{name: "go", input: "go/input.go", reader: golangReader{}},
		{name: "markdown", input: "markdown/input.md", reader: markdownReader{}},
		{name: "json", input: "json/input.json", reader: jsonReader{}},
		{name: "generic", input: "generic/input.txt", reader: genericReader{}},
	}

	reductions := make([]float64, 0, len(cases))
	for _, tc := range cases {
		in := readFixture(t, tc.input)
		out, err := tc.reader.Filter(in, readfilter.Options{UltraCompact: true})
		if err != nil {
			t.Fatalf("%s filter error: %v", tc.name, err)
		}
		before := envelope.EstimateTokens(in)
		after := envelope.EstimateTokens(out)
		if before == 0 {
			t.Fatalf("%s fixture produced zero tokens", tc.name)
		}
		reduction := (float64(before-after) / float64(before)) * 100
		reductions = append(reductions, reduction)
		if tc.name != "generic" && reduction < 40 {
			t.Fatalf("%s reduction %.2f%% below 40%% floor", tc.name, reduction)
		}
	}

	sorted := append([]float64(nil), reductions...)
	slices.Sort(sorted)
	median := sorted[len(sorted)/2]
	if median < 50 {
		t.Fatalf("median reduction %.2f%% below 50%% floor (reductions=%v)", median, reductions)
	}
}

func readFixture(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join("..", "testdata", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	return string(data)
}
