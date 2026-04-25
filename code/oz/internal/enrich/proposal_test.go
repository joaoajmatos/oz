package enrich

import (
	"fmt"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/semantic"
)

// --- BuildAllowlist tests (CCA-1-02) ---

func TestBuildAllowlist_EligibleTypes(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			{ID: "spec_section:specs/foo.md:overview", Type: graph.NodeTypeSpecSection},
			{ID: "decision:0001-foo", Type: graph.NodeTypeDecision},
			{ID: "code_package:github.com/x/y", Type: graph.NodeTypeCodePackage},
			// These must be excluded:
			{ID: "code_file:code/foo.go", Type: graph.NodeTypeCodeFile},
			{ID: "code_symbol:Foo", Type: graph.NodeTypeCodeSymbol},
			{ID: "doc:docs/arch.md:intro", Type: graph.NodeTypeDoc},
			{ID: "note:notes/foo.md", Type: graph.NodeTypeNote},
		},
	}
	ids := BuildAllowlist(g, nil)
	want := map[string]bool{
		"agent:coding":                       true,
		"spec_section:specs/foo.md:overview": true,
		"decision:0001-foo":                  true,
		"code_package:github.com/x/y":        true,
	}
	if len(ids) != len(want) {
		t.Errorf("got %d IDs, want %d; ids = %v", len(ids), len(want), ids)
	}
	for _, id := range ids {
		if !want[id] {
			t.Errorf("unexpected ID in allowlist: %q", id)
		}
	}
}

func TestBuildAllowlist_IncludesReviewedConceptIDs(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
		},
	}
	existing := &semantic.Overlay{
		Concepts: []semantic.Concept{
			{ID: "concept:reviewed-concept", Reviewed: true},
			{ID: "concept:unreviewed", Reviewed: false},
		},
	}
	ids := BuildAllowlist(g, existing)
	found := map[string]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found["concept:reviewed-concept"] {
		t.Error("reviewed concept ID must be in allowlist")
	}
	if found["concept:unreviewed"] {
		t.Error("unreviewed concept ID must not be in allowlist")
	}
}

func TestBuildAllowlist_Sorted(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "spec_section:z.md:sec", Type: graph.NodeTypeSpecSection},
			{ID: "agent:b", Type: graph.NodeTypeAgent},
			{ID: "code_package:a/pkg", Type: graph.NodeTypeCodePackage},
		},
	}
	ids := BuildAllowlist(g, nil)
	for i := 1; i < len(ids); i++ {
		if ids[i] < ids[i-1] {
			t.Errorf("allowlist not sorted: %q before %q", ids[i-1], ids[i])
		}
	}
}

// TestBuildAllowlist_Integration tests a mid-size workspace and asserts the
// allowlist stays within the measured ~2k token budget (CCA-1-02 AC).
func TestBuildAllowlist_Integration(t *testing.T) {
	var nodes []graph.Node
	for i := 0; i < 4; i++ {
		nodes = append(nodes, graph.Node{
			ID:   fmt.Sprintf("agent:agent-%d", i),
			Type: graph.NodeTypeAgent,
		})
	}
	for i := 0; i < 30; i++ {
		nodes = append(nodes, graph.Node{
			ID:   fmt.Sprintf("spec_section:specs/s%d.md:section", i),
			Type: graph.NodeTypeSpecSection,
		})
	}
	for i := 0; i < 5; i++ {
		nodes = append(nodes, graph.Node{
			ID:   fmt.Sprintf("decision:000%d-decision", i),
			Type: graph.NodeTypeDecision,
		})
	}
	for i := 0; i < 35; i++ {
		nodes = append(nodes, graph.Node{
			ID:   fmt.Sprintf("code_package:github.com/example/pkg%d", i),
			Type: graph.NodeTypeCodePackage,
		})
	}
	// Non-eligible nodes: must be excluded.
	for i := 0; i < 50; i++ {
		nodes = append(nodes, graph.Node{
			ID:   fmt.Sprintf("code_symbol:Sym%d", i),
			Type: graph.NodeTypeCodeSymbol,
		})
	}
	g := &graph.Graph{Nodes: nodes}
	ids := BuildAllowlist(g, nil)

	const wantCount = 4 + 30 + 5 + 35
	if len(ids) != wantCount {
		t.Errorf("expected %d IDs, got %d", wantCount, len(ids))
	}
	// Token budget: allowlist text must stay under 2000 tokens (~8000 chars).
	estTokens := len(strings.Join(ids, "\n")) / 4
	if estTokens > 2000 {
		t.Errorf("allowlist ~%d tokens exceeds 2000-token cap", estTokens)
	}
}

// --- BuildProposalPrompt tests ---

func TestBuildProposalPrompt_ContainsName(t *testing.T) {
	opts := ProposeOptions{Name: "Query Routing"}
	prompt, err := BuildProposalPrompt(opts, []string{"agent:coding"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prompt, "Query Routing") {
		t.Error("prompt must contain the concept name")
	}
	if !strings.Contains(prompt, "agent:coding") {
		t.Error("prompt must contain the allowlist ID")
	}
}

func TestBuildProposalPrompt_RetrievalBlocksPresent(t *testing.T) {
	opts := ProposeOptions{
		Name: "Semantic Overlay",
		Blocks: []RetrievedBlock{
			{File: "specs/semantic-overlay.md", Section: "Merge contract", Trust: "high"},
		},
	}
	prompt, err := BuildProposalPrompt(opts, []string{"agent:oz-spec"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prompt, "specs/semantic-overlay.md") {
		t.Error("prompt must contain retrieved block file")
	}
	if !strings.Contains(prompt, "trust: high") {
		t.Error("prompt must contain trust level")
	}
}

func TestBuildProposalPrompt_NoBlocksOmitsSection(t *testing.T) {
	opts := ProposeOptions{Name: "Foo"}
	prompt, err := BuildProposalPrompt(opts, []string{"agent:coding"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(prompt, "## Retrieved context") {
		t.Error("prompt must not have retrieved context section when no blocks provided")
	}
}

func TestBuildProposalPrompt_ExactlyOneConceptRule(t *testing.T) {
	opts := ProposeOptions{Name: "Routing Pipeline"}
	prompt, err := BuildProposalPrompt(opts, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(prompt, "exactly ONE") {
		t.Error("prompt must instruct the model to return exactly one concept")
	}
}

// --- ParseSingleConcept tests (CCA-1-06) ---

func TestParseSingleConcept_ExactlyOne(t *testing.T) {
	nodeIDs := map[string]struct{}{"agent:coding": {}}
	raw := `{"concepts":[{"id":"concept:routing","name":"Routing","tag":"EXTRACTED","confidence":0.9}],"edges":[{"from":"concept:routing","to":"agent:coding","type":"agent_owns_concept","tag":"EXTRACTED","confidence":0.9}]}`
	c, edges, _, err := ParseSingleConcept(raw, nodeIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != "concept:routing" {
		t.Errorf("got concept ID %q, want concept:routing", c.ID)
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(edges))
	}
}

func TestParseSingleConcept_ZeroConcepts_ReturnsError(t *testing.T) {
	_, _, _, err := ParseSingleConcept(`{"concepts":[],"edges":[]}`, map[string]struct{}{})
	if err == nil {
		t.Fatal("expected error for zero concepts")
	}
	if !strings.Contains(err.Error(), "no concepts") {
		t.Errorf("error should mention no concepts; got: %v", err)
	}
}

func TestParseSingleConcept_MultipleConcepts_ReturnsError(t *testing.T) {
	raw := `{"concepts":[{"id":"concept:a","name":"A","tag":"EXTRACTED","confidence":1.0},{"id":"concept:b","name":"B","tag":"EXTRACTED","confidence":1.0}],"edges":[]}`
	_, _, _, err := ParseSingleConcept(raw, map[string]struct{}{})
	if err == nil {
		t.Fatal("expected error for multiple concepts")
	}
	if !strings.Contains(err.Error(), "2 concepts") {
		t.Errorf("error should mention count; got: %v", err)
	}
	if !strings.Contains(err.Error(), "exactly 1") {
		t.Errorf("error should mention expected count; got: %v", err)
	}
}

func TestParseSingleConcept_SkippedItemCarriesError(t *testing.T) {
	// Concept with bad ID prefix gets skipped → zero valid concepts.
	raw := `{"concepts":[{"id":"bad:id","name":"X","tag":"EXTRACTED","confidence":1.0}],"edges":[]}`
	_, _, skipped, err := ParseSingleConcept(raw, map[string]struct{}{})
	if err == nil {
		t.Fatal("expected error for all-skipped concepts")
	}
	if len(skipped) == 0 {
		t.Error("expected skipped items to be returned alongside error")
	}
}

// --- Near-duplicate detection tests (T5) ---

func TestNormalizeConceptName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Query Routing", "queryrouting"},
		{"BM25F Scoring", "bm25fscoring"},
		{"日本語", ""},
		{"  Foo  Bar  ", "foobar"},
		{"already-normalized", "alreadynormalized"},
	}
	for _, tc := range cases {
		got := normalizeConceptName(tc.in)
		if got != tc.want {
			t.Errorf("normalizeConceptName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"routing", "routng", 1},
	}
	for _, tc := range cases {
		got := levenshtein(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFindNearDuplicates_ExactNormMatch(t *testing.T) {
	existing := []semantic.Concept{
		{ID: "concept:query-routing", Name: "Query Routing", Reviewed: true},
	}
	got := findNearDuplicates("query routing", existing)
	if len(got) != 1 || got[0] != "Query Routing" {
		t.Errorf("expected [Query Routing], got %v", got)
	}
}

func TestFindNearDuplicates_TypoMatch(t *testing.T) {
	existing := []semantic.Concept{
		{ID: "concept:routing", Name: "Routing", Reviewed: true},
	}
	// "Routng" normalizes to "routng", levenshtein("routng","routing")=1 ≤ 2
	got := findNearDuplicates("Routng", existing)
	if len(got) != 1 {
		t.Errorf("expected typo to match, got %v", got)
	}
}

func TestFindNearDuplicates_UnreviewedIgnored(t *testing.T) {
	existing := []semantic.Concept{
		{ID: "concept:routing", Name: "Routing", Reviewed: false},
	}
	got := findNearDuplicates("Routing", existing)
	if len(got) != 0 {
		t.Errorf("unreviewed concept must not match; got %v", got)
	}
}

func TestFindNearDuplicates_DistinctNamesNoMatch(t *testing.T) {
	existing := []semantic.Concept{
		{ID: "concept:audit-pipeline", Name: "Audit Pipeline", Reviewed: true},
	}
	got := findNearDuplicates("Semantic Overlay", existing)
	if len(got) != 0 {
		t.Errorf("distinct names must not match; got %v", got)
	}
}

func TestParseSingleConcept_ReviewedAlwaysFalse(t *testing.T) {
	// Model output claims reviewed:true — ParseResponse must reset it to false.
	nodeIDs := map[string]struct{}{"agent:coding": {}}
	raw := `{"concepts":[{"id":"concept:foo","name":"Foo","tag":"EXTRACTED","confidence":1.0,"reviewed":true}],"edges":[]}`
	c, _, _, err := ParseSingleConcept(raw, nodeIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Reviewed {
		t.Error("new concept must have reviewed=false regardless of model output")
	}
}
