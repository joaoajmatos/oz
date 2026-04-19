package context_test

import (
	"bytes"
	"encoding/json"
	"testing"

	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/graph"
	"github.com/oz-tools/oz/internal/testws"
)

// TestBuild_Minimal runs oz context build against the 01_minimal golden fixture
// and validates node/edge counts and node type presence.
func TestBuild_Minimal(t *testing.T) {
	ws := testws.FromFixture(t, "../query/testdata/golden/01_minimal/workspace.yaml").Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// 3 agents in the fixture → at least 3 agent nodes.
	agentCount := countByType(result.Graph.Nodes, graph.NodeTypeAgent)
	if agentCount < 3 {
		t.Errorf("expected ≥3 agent nodes, got %d", agentCount)
	}

	// Spec file with 4 sections → at least 4 spec_section nodes.
	specCount := countByType(result.Graph.Nodes, graph.NodeTypeSpecSection)
	if specCount < 4 {
		t.Errorf("expected ≥4 spec_section nodes, got %d (one per section in specs/project-spec.md)", specCount)
	}

	// 2 decision files → at least 2 decision nodes.
	decisionCount := countByType(result.Graph.Nodes, graph.NodeTypeDecision)
	if decisionCount < 2 {
		t.Errorf("expected ≥2 decision nodes, got %d", decisionCount)
	}

	// Each agent has a read-chain → at least some reads edges.
	readsCount := countByEdgeType(result.Graph.Edges, graph.EdgeTypeReads)
	if readsCount == 0 {
		t.Error("expected reads edges, got 0")
	}

	t.Logf("01_minimal: %d nodes (%d agents, %d spec sections, %d decisions), %d edges (%d reads)",
		result.NodeCount, agentCount, specCount, decisionCount, result.EdgeCount, readsCount)
}

// TestBuild_Medium runs oz context build against the 02_medium golden fixture.
func TestBuild_Medium(t *testing.T) {
	ws := testws.FromFixture(t, "../query/testdata/golden/02_medium/workspace.yaml").Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	agentCount := countByType(result.Graph.Nodes, graph.NodeTypeAgent)
	if agentCount < 5 {
		t.Errorf("expected ≥5 agent nodes in medium fixture, got %d", agentCount)
	}

	t.Logf("02_medium: %d nodes, %d edges", result.NodeCount, result.EdgeCount)
}

// TestBuild_Determinism verifies that running Build twice produces byte-identical
// JSON output. This is the core guarantee of S2-06.
func TestBuild_Determinism(t *testing.T) {
	ws := testws.FromFixture(t, "../query/testdata/golden/01_minimal/workspace.yaml").Build()

	// First run — write to disk.
	result1, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build (first): %v", err)
	}
	if err := ozcontext.Serialize(ws.Path(), result1.Graph); err != nil {
		t.Fatalf("Serialize (first): %v", err)
	}
	first, err := ws.ReadFile("context/graph.json")
	if err != nil {
		t.Fatalf("read graph.json (first): %v", err)
	}

	// Second run — overwrite.
	result2, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build (second): %v", err)
	}
	if err := ozcontext.Serialize(ws.Path(), result2.Graph); err != nil {
		t.Fatalf("Serialize (second): %v", err)
	}
	second, err := ws.ReadFile("context/graph.json")
	if err != nil {
		t.Fatalf("read graph.json (second): %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Error("determinism violation: two runs produced different graph.json output")
		t.Logf("first  hash: %s", result1.Graph.ContentHash)
		t.Logf("second hash: %s", result2.Graph.ContentHash)
	}
}

// TestBuild_AgentFields verifies that agent nodes carry the fields from AGENT.md.
func TestBuild_AgentFields(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend",
			testws.Role("Builds REST endpoints"),
			testws.Scope("code/api/**"),
			testws.Responsibilities("Implements handlers and middleware"),
			testws.OutOfScope("UI work"),
			testws.ReadChain("AGENTS.md", "specs/api.md"),
			testws.Rules("rules/coding-guidelines.md"),
			testws.Skills("skills/commit/"),
		).
		Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var agentNode *graph.Node
	for i, n := range result.Graph.Nodes {
		if n.Type == graph.NodeTypeAgent && n.Name == "backend" {
			agentNode = &result.Graph.Nodes[i]
			break
		}
	}
	if agentNode == nil {
		t.Fatal("agent node 'backend' not found")
	}

	if agentNode.Role != "Builds REST endpoints" {
		t.Errorf("role: got %q, want %q", agentNode.Role, "Builds REST endpoints")
	}
	if len(agentNode.Scope) == 0 {
		t.Error("scope: expected at least one scope path")
	}
	if len(agentNode.ReadChain) == 0 {
		t.Error("read_chain: expected at least one read-chain item")
	}
	if len(agentNode.Rules) == 0 {
		t.Error("rules: expected at least one rule")
	}
	if len(agentNode.Skills) == 0 {
		t.Error("skills: expected at least one skill")
	}
}

// TestBuild_SectionNodes verifies that spec sections become spec_section nodes
// with correct tier and file references.
func TestBuild_SectionNodes(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coding", testws.Role("Writes code")).
		WithSpec("specs/api.md",
			testws.Section("Overview", "REST API"),
			testws.Section("Authentication", "Bearer tokens"),
		).
		Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	found := map[string]bool{}
	for _, n := range result.Graph.Nodes {
		if n.Type == graph.NodeTypeSpecSection && n.File == "specs/api.md" {
			found[n.Section] = true
			if n.Tier != graph.TierSpecs {
				t.Errorf("node %q: tier %q, want %q", n.ID, n.Tier, graph.TierSpecs)
			}
		}
	}
	for _, want := range []string{"Overview", "Authentication"} {
		if !found[want] {
			t.Errorf("spec_section node for section %q not found", want)
		}
	}
}

// TestBuild_ReadsEdges verifies that reads edges are produced for read-chain items
// that resolve to nodes in the graph.
func TestBuild_ReadsEdges(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coder",
			testws.Role("Writes code"),
			testws.ReadChain("specs/api.md"),
		).
		WithSpec("specs/api.md", testws.Section("Overview", "API spec")).
		Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var hasReadsEdge bool
	for _, e := range result.Graph.Edges {
		if e.Type == graph.EdgeTypeReads && e.From == "agent:coder" {
			hasReadsEdge = true
			break
		}
	}
	if !hasReadsEdge {
		t.Error("expected a reads edge from agent:coder to specs/api.md node")
	}
}

// TestBuild_DecisionNodes verifies decisions produce NodeTypeDecision nodes
// with the specs tier.
func TestBuild_DecisionNodes(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coder", testws.Role("Writes code")).
		WithDecision("0001-use-go", "Go chosen for single-binary distribution").
		Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var found bool
	for _, n := range result.Graph.Nodes {
		if n.Type == graph.NodeTypeDecision && n.Name == "0001-use-go" {
			found = true
			if n.Tier != graph.TierSpecs {
				t.Errorf("decision tier: got %q, want %q", n.Tier, graph.TierSpecs)
			}
		}
	}
	if !found {
		t.Error("decision node 0001-use-go not found")
	}
}

// TestBuild_GraphJSON verifies that graph.json is valid JSON after Serialize.
func TestBuild_GraphJSON(t *testing.T) {
	ws := testws.New(t).
		WithAgent("coder", testws.Role("Writes code")).
		Build()

	result, err := ozcontext.Build(ws.Path())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := ozcontext.Serialize(ws.Path(), result.Graph); err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	data, err := ws.ReadFile("context/graph.json")
	if err != nil {
		t.Fatalf("read graph.json: %v", err)
	}

	var g graph.Graph
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("graph.json is not valid JSON: %v", err)
	}

	if g.SchemaVersion == "" {
		t.Error("graph.json missing schema_version")
	}
	if g.ContentHash == "" {
		t.Error("graph.json missing content_hash")
	}
	if len(g.Nodes) == 0 {
		t.Error("graph.json has no nodes")
	}
}

// --- helpers -----------------------------------------------------------------

func countByType(nodes []graph.Node, nodeType string) int {
	n := 0
	for _, node := range nodes {
		if node.Type == nodeType {
			n++
		}
	}
	return n
}

func countByEdgeType(edges []graph.Edge, edgeType string) int {
	n := 0
	for _, e := range edges {
		if e.Type == edgeType {
			n++
		}
	}
	return n
}
