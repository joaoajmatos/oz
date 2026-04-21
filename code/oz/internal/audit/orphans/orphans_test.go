package orphans

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/graph"
)

// hasCode reports whether any finding in fs has the given code.
func hasCode(fs []audit.Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

// countCode returns how many findings have the given code.
func countCode(fs []audit.Finding, code string) int {
	n := 0
	for _, f := range fs {
		if f.Code == code {
			n++
		}
	}
	return n
}

// --- ORPH001: spec_section nodes ---

func TestORPH001_UnreferencedSpecFile(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
		},
	}
	fs := runCheck(g)
	if !hasCode(fs, "ORPH001") {
		t.Error("expected ORPH001 for unreferenced spec file, got none")
	}
}

func TestORPH001_ReferencedSpecFile_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "spec_section:specs/api.md:Overview", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH001") {
		t.Error("expected no ORPH001 for spec with reads edge")
	}
}

func TestORPH001_MultiSection_OneSectionReferenced_NoFinding(t *testing.T) {
	// If any section of a spec file has an inbound edge, the file is not orphaned.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			// Two sections from the same file; reads edge only touches the first.
			{ID: "spec_section:specs/api.md:Architecture", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "spec_section:specs/api.md:Architecture", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH001") {
		t.Error("expected no ORPH001 when at least one section of a spec file is referenced")
	}
}

func TestORPH001_ReferencedViaSupports_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "doc:docs/arch.md:Intro", Type: graph.NodeTypeDoc, File: "docs/arch.md"},
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
		},
		Edges: []graph.Edge{
			{From: "doc:docs/arch.md:Intro", To: "spec_section:specs/api.md:Overview", Type: graph.EdgeTypeSupports},
			// arch.md itself has an inbound references edge to avoid ORPH002.
			{From: "spec_section:specs/api.md:Overview", To: "doc:docs/arch.md:Intro", Type: graph.EdgeTypeReferences},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH001") {
		t.Error("expected no ORPH001 for spec referenced via supports edge")
	}
}

func TestORPH001_TwoOrphanedSpecFiles(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
			{ID: "spec_section:specs/auth.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/auth.md"},
		},
	}
	fs := runCheck(g)
	if n := countCode(fs, "ORPH001"); n != 2 {
		t.Errorf("expected 2 ORPH001 findings, got %d", n)
	}
}

// --- ORPH001: decision nodes ---

func TestORPH001_UnreferencedDecision(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "decision:0001-use-go", Type: graph.NodeTypeDecision, File: "specs/decisions/0001-use-go.md", Name: "0001-use-go"},
		},
	}
	fs := runCheck(g)
	if !hasCode(fs, "ORPH001") {
		t.Error("expected ORPH001 for unreferenced decision")
	}
}

func TestORPH001_ReferencedDecision_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			{ID: "decision:0001-use-go", Type: graph.NodeTypeDecision, File: "specs/decisions/0001-use-go.md", Name: "0001-use-go"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "decision:0001-use-go", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH001") {
		t.Error("expected no ORPH001 for referenced decision")
	}
}

// --- ORPH002: doc nodes ---

func TestORPH002_UnreferencedDoc(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "doc:docs/arch.md:Overview", Type: graph.NodeTypeDoc, File: "docs/arch.md"},
		},
	}
	fs := runCheck(g)
	if !hasCode(fs, "ORPH002") {
		t.Error("expected ORPH002 for unreferenced doc")
	}
}

func TestORPH002_ReferencedDoc_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			{ID: "doc:docs/arch.md:Overview", Type: graph.NodeTypeDoc, File: "docs/arch.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "doc:docs/arch.md:Overview", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH002") {
		t.Error("expected no ORPH002 for doc with reads edge")
	}
}

func TestORPH002_MultiSection_OneSectionReferenced_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			{ID: "doc:docs/arch.md:Intro", Type: graph.NodeTypeDoc, File: "docs/arch.md"},
			{ID: "doc:docs/arch.md:Components", Type: graph.NodeTypeDoc, File: "docs/arch.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "doc:docs/arch.md:Intro", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH002") {
		t.Error("expected no ORPH002 when at least one section of a doc file is referenced")
	}
}

// --- ORPH003: context snapshots ---

func TestORPH003_UnreferencedSnapshot(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "context_snapshot:context/auth/snapshot.md", Type: graph.NodeTypeContextSnapshot, File: "context/auth/snapshot.md"},
		},
	}
	fs := runCheck(g)
	if !hasCode(fs, "ORPH003") {
		t.Error("expected ORPH003 for unreferenced context snapshot")
	}
}

func TestORPH003_SnapshotInAgentTopics_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, ContextTopics: []string{"auth"}},
			{ID: "context_snapshot:context/auth/snapshot.md", Type: graph.NodeTypeContextSnapshot, File: "context/auth/snapshot.md"},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH003") {
		t.Error("expected no ORPH003 when topic is declared in an agent's context_topics")
	}
}

func TestORPH003_SnapshotWithInboundEdge_NoFinding(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent},
			{ID: "context_snapshot:context/auth/snapshot.md", Type: graph.NodeTypeContextSnapshot, File: "context/auth/snapshot.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "context_snapshot:context/auth/snapshot.md", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH003") {
		t.Error("expected no ORPH003 when snapshot has inbound edge")
	}
}

func TestORPH003_TopicCaseInsensitive(t *testing.T) {
	// Topic matching should be case-insensitive.
	g := &graph.Graph{
		Nodes: []graph.Node{
			{ID: "agent:coding", Type: graph.NodeTypeAgent, ContextTopics: []string{"AUTH"}},
			{ID: "context_snapshot:context/auth/snapshot.md", Type: graph.NodeTypeContextSnapshot, File: "context/auth/snapshot.md"},
		},
	}
	fs := runCheck(g)
	if hasCode(fs, "ORPH003") {
		t.Error("expected no ORPH003 when topic matches case-insensitively")
	}
}

// --- Clean graph produces no findings ---

func TestCleanGraph_NoFindings(t *testing.T) {
	g := &graph.Graph{
		Nodes: []graph.Node{
			{
				ID:            "agent:coding",
				Type:          graph.NodeTypeAgent,
				ContextTopics: []string{"auth"},
			},
			{ID: "spec_section:specs/api.md:Overview", Type: graph.NodeTypeSpecSection, File: "specs/api.md"},
			{ID: "decision:0001-use-go", Type: graph.NodeTypeDecision, File: "specs/decisions/0001-use-go.md", Name: "0001-use-go"},
			{ID: "doc:docs/arch.md:Overview", Type: graph.NodeTypeDoc, File: "docs/arch.md"},
			{ID: "context_snapshot:context/auth/snapshot.md", Type: graph.NodeTypeContextSnapshot, File: "context/auth/snapshot.md"},
		},
		Edges: []graph.Edge{
			{From: "agent:coding", To: "spec_section:specs/api.md:Overview", Type: graph.EdgeTypeReads},
			{From: "agent:coding", To: "decision:0001-use-go", Type: graph.EdgeTypeReads},
			{From: "agent:coding", To: "doc:docs/arch.md:Overview", Type: graph.EdgeTypeReads},
		},
	}
	fs := runCheck(g)
	if len(fs) != 0 {
		t.Errorf("expected 0 findings for clean graph, got %d: %+v", len(fs), fs)
	}
}
