package enrich

import (
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/semantic"
)

// nodeSet is a helper for building the graphNodeIDs map used in ParseResponse.
func nodeSet(ids ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

// parseCase is a single table-driven test case for ParseResponse / ParseSingleConcept.
type parseCase struct {
	name         string
	input        string
	nodeIDs      map[string]struct{}
	wantConcepts int
	wantEdges    int
	wantSkipped  int // minimum number of skipped-item messages expected
	wantErrSub   string // non-empty → ParseSingleConcept must return an error containing this
}

var goldenCases = []parseCase{
	{
		name: "valid single concept with edge",
		input: `{"concepts":[{"id":"concept:routing","name":"Routing","description":"Routes tasks.","tag":"EXTRACTED","confidence":0.95}],"edges":[{"from":"concept:routing","to":"agent:coding","type":"agent_owns_concept","tag":"EXTRACTED","confidence":0.95}]}`,
		nodeIDs:      nodeSet("agent:coding"),
		wantConcepts: 1,
		wantEdges:    1,
	},
	{
		name: "markdown-fenced JSON with language tag",
		input: "```json\n{\"concepts\":[{\"id\":\"concept:fence\",\"name\":\"Fenced\",\"tag\":\"EXTRACTED\",\"confidence\":1.0}],\"edges\":[]}\n```",
		nodeIDs:      nodeSet(),
		wantConcepts: 1,
		wantEdges:    0,
	},
	{
		name: "markdown-fenced JSON without language tag",
		input: "```\n{\"concepts\":[{\"id\":\"concept:fence2\",\"name\":\"Fenced2\",\"tag\":\"EXTRACTED\",\"confidence\":1.0}],\"edges\":[]}\n```",
		nodeIDs:      nodeSet(),
		wantConcepts: 1,
		wantEdges:    0,
	},
	{
		name: "multi-concept response ParseResponse gets all",
		input: `{"concepts":[{"id":"concept:a","name":"A","tag":"EXTRACTED","confidence":1.0},{"id":"concept:b","name":"B","tag":"EXTRACTED","confidence":1.0}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 2,
		wantEdges:    0,
	},
	{
		name: "multi-concept ParseSingleConcept errors with count",
		input: `{"concepts":[{"id":"concept:a","name":"A","tag":"EXTRACTED","confidence":1.0},{"id":"concept:b","name":"B","tag":"EXTRACTED","confidence":1.0}],"edges":[]}`,
		nodeIDs:    nodeSet(),
		wantErrSub: "2 concepts",
	},
	{
		name: "edge with invalid To node ID is skipped",
		input: `{"concepts":[{"id":"concept:ok","name":"OK","tag":"EXTRACTED","confidence":1.0}],"edges":[{"from":"concept:ok","to":"nonexistent:node","type":"agent_owns_concept","tag":"EXTRACTED","confidence":1.0}]}`,
		nodeIDs:      nodeSet(), // nonexistent:node not in graph
		wantConcepts: 1,
		wantEdges:    0,
		wantSkipped:  1,
	},
	{
		name: "unicode concept name",
		input: `{"concepts":[{"id":"concept:unicode-name","name":"日本語の概念","description":"A concept with a Unicode name.","tag":"EXTRACTED","confidence":0.9}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 1,
	},
	{
		name:        "empty string returns JSON parse error in skipped",
		input:       "",
		nodeIDs:     nodeSet(),
		wantSkipped: 1,
		wantErrSub:  "no valid concept",
	},
	{
		name:        "whitespace only",
		input:       "   \n\t  ",
		nodeIDs:     nodeSet(),
		wantSkipped: 1,
		wantErrSub:  "no valid concept",
	},
	{
		name:        "malformed JSON",
		input:       `{"concepts": [{broken`,
		nodeIDs:     nodeSet(),
		wantSkipped: 1,
		wantErrSub:  "no valid concept",
	},
	{
		name:  "concept missing name is skipped",
		input: `{"concepts":[{"id":"concept:noname","tag":"EXTRACTED","confidence":1.0}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 0,
		wantSkipped:  1,
		wantErrSub:   "no valid concept",
	},
	{
		name:  "concept with bad ID prefix is skipped",
		input: `{"concepts":[{"id":"badprefix:foo","name":"Foo","tag":"EXTRACTED","confidence":1.0}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 0,
		wantSkipped:  1,
		wantErrSub:   "no valid concept",
	},
	{
		name:  "confidence <= 0 is clamped to 1.0",
		input: `{"concepts":[{"id":"concept:clamp","name":"Clamp","tag":"EXTRACTED","confidence":-0.5}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 1,
	},
	{
		name:  "confidence > 1 is clamped to 1.0",
		input: `{"concepts":[{"id":"concept:over","name":"Over","tag":"EXTRACTED","confidence":1.5}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 1,
	},
	{
		name:  "unknown tag is normalized to EXTRACTED",
		input: `{"concepts":[{"id":"concept:badtag","name":"BadTag","tag":"UNKNOWN","confidence":1.0}],"edges":[]}`,
		nodeIDs:      nodeSet(),
		wantConcepts: 1,
	},
	{
		name:  "unknown edge type is skipped",
		input: `{"concepts":[{"id":"concept:x","name":"X","tag":"EXTRACTED","confidence":1.0}],"edges":[{"from":"concept:x","to":"agent:y","type":"made_up_type","tag":"EXTRACTED","confidence":1.0}]}`,
		nodeIDs:      nodeSet("agent:y"),
		wantConcepts: 1,
		wantEdges:    0,
		wantSkipped:  1,
	},
	{
		name:  "edge from unknown concept is skipped",
		input: `{"concepts":[{"id":"concept:real","name":"Real","tag":"EXTRACTED","confidence":1.0}],"edges":[{"from":"concept:ghost","to":"agent:y","type":"agent_owns_concept","tag":"EXTRACTED","confidence":1.0}]}`,
		nodeIDs:      nodeSet("agent:y"),
		wantConcepts: 1,
		wantEdges:    0,
		wantSkipped:  1,
	},
	{
		name: "high-confidence edge auto-reviewed",
		input: `{"concepts":[{"id":"concept:hi","name":"Hi","tag":"EXTRACTED","confidence":1.0}],"edges":[{"from":"concept:hi","to":"agent:coding","type":"agent_owns_concept","tag":"EXTRACTED","confidence":0.9}]}`,
		nodeIDs:      nodeSet("agent:coding"),
		wantConcepts: 1,
		wantEdges:    1,
	},
	{
		name: "zero concepts ParseSingleConcept no-agents error",
		input: `{"concepts":[],"edges":[]}`,
		nodeIDs:    nodeSet(),
		wantErrSub: "no concepts",
	},
}

func TestParseResponse_GoldenCases(t *testing.T) {
	for _, tc := range goldenCases {
		if tc.wantErrSub != "" {
			continue // ParseSingleConcept cases tested separately below
		}
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			concepts, edges, skipped := ParseResponse(tc.input, tc.nodeIDs)
			if tc.wantConcepts >= 0 && len(concepts) != tc.wantConcepts {
				t.Errorf("concepts: got %d, want %d (skipped=%v)", len(concepts), tc.wantConcepts, skipped)
			}
			if tc.wantEdges > 0 && len(edges) != tc.wantEdges {
				t.Errorf("edges: got %d, want %d", len(edges), tc.wantEdges)
			}
			if len(skipped) < tc.wantSkipped {
				t.Errorf("skipped: got %d, want >= %d", len(skipped), tc.wantSkipped)
			}
		})
	}
}

func TestParseSingleConcept_GoldenCases(t *testing.T) {
	for _, tc := range goldenCases {
		if tc.wantErrSub == "" {
			continue // ParseResponse cases tested above
		}
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := ParseSingleConcept(tc.input, tc.nodeIDs)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSub)
			}
		})
	}
}

// TestParseResponse_ConfidenceClamped verifies confidence values are normalised
// to [0,1] and that tags are coerced to valid values.
func TestParseResponse_ConfidenceClamped(t *testing.T) {
	tests := []struct {
		raw      string
		wantConf float64
		wantTag  string
	}{
		{
			`{"concepts":[{"id":"concept:a","name":"A","tag":"EXTRACTED","confidence":-1.0}],"edges":[]}`,
			1.0, semantic.TagExtracted,
		},
		{
			`{"concepts":[{"id":"concept:b","name":"B","tag":"INFERRED","confidence":2.5}],"edges":[]}`,
			1.0, semantic.TagInferred,
		},
		{
			`{"concepts":[{"id":"concept:c","name":"C","tag":"BOGUS","confidence":0.7}],"edges":[]}`,
			0.7, semantic.TagExtracted,
		},
	}
	for _, tt := range tests {
		concepts, _, _ := ParseResponse(tt.raw, nodeSet())
		if len(concepts) != 1 {
			t.Fatalf("expected 1 concept, got %d", len(concepts))
		}
		if concepts[0].Confidence != tt.wantConf {
			t.Errorf("confidence: got %v, want %v", concepts[0].Confidence, tt.wantConf)
		}
		if concepts[0].Tag != tt.wantTag {
			t.Errorf("tag: got %q, want %q", concepts[0].Tag, tt.wantTag)
		}
	}
}

// TestParseResponse_ReviewedAlwaysFalseOnConcepts verifies that no matter what
// the model outputs in the "reviewed" field, all concepts come out reviewed=false.
func TestParseResponse_ReviewedAlwaysFalseOnConcepts(t *testing.T) {
	raw := `{"concepts":[{"id":"concept:x","name":"X","tag":"EXTRACTED","confidence":1.0,"reviewed":true}],"edges":[]}`
	concepts, _, _ := ParseResponse(raw, nodeSet())
	if len(concepts) != 1 {
		t.Fatalf("expected 1 concept")
	}
	if concepts[0].Reviewed {
		t.Error("concept Reviewed must be false regardless of model output")
	}
}

// TestParseResponse_HighConfidenceEdgeAutoReviewed checks that edges with
// confidence >= 0.85 are auto-promoted to reviewed=true (ADR-0003).
func TestParseResponse_HighConfidenceEdgeAutoReviewed(t *testing.T) {
	raw := `{"concepts":[{"id":"concept:hi","name":"Hi","tag":"EXTRACTED","confidence":1.0}],"edges":[{"from":"concept:hi","to":"agent:x","type":"agent_owns_concept","tag":"EXTRACTED","confidence":0.9},{"from":"concept:hi","to":"agent:x","type":"implements_spec","tag":"INFERRED","confidence":0.5}]}`
	nodeIDs := nodeSet("agent:x")
	_, edges, _ := ParseResponse(raw, nodeIDs)
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
	if !edges[0].Reviewed {
		t.Error("high-confidence edge (0.9) should be auto-reviewed")
	}
	if edges[1].Reviewed {
		t.Error("low-confidence edge (0.5) should NOT be auto-reviewed")
	}
}
