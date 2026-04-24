package query

import (
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/convention"
	"github.com/joaoajmatos/oz/internal/graph"
)

// TestRetrievalForProposal_EmptyQuery verifies that an empty query returns an
// empty result without error.
func TestRetrievalForProposal_EmptyQuery(t *testing.T) {
	ws := t.TempDir()
	r, err := RetrievalForProposal(ws, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.ContextBlocks) != 0 || len(r.RelevantConcepts) != 0 {
		t.Errorf("expected empty result for empty query, got %+v", r)
	}
}

// TestRetrievalForProposal_ContractT2_NoScopeSurvivor verifies that proposal
// mode (agentName="") does not force a scope-connected file to survive in
// context_blocks when a higher-trust block exists. This locks the T2 pre-mortem
// behavior: ensureScopeSurvivor is a no-op when scope is empty.
func TestRetrievalForProposal_ContractT2_NoScopeSurvivor(t *testing.T) {
	ws := t.TempDir()
	mustWrite(t, ws, "specs/auth.md", "## overview\nauth authentication JWT token bearer")
	mustWrite(t, ws, "code/auth/handler.go", "package auth\n")

	g := &graph.Graph{
		ContentHash: "proposal-t2",
		Nodes: []graph.Node{
			{
				ID:    "agent:auth",
				Type:  graph.NodeTypeAgent,
				Name:  "auth",
				Scope: []string{"code/auth/**"},
			},
			{
				ID:      "spec_section:specs/auth.md:overview",
				Type:    graph.NodeTypeSpecSection,
				File:    "specs/auth.md",
				Section: "overview",
				Name:    "overview",
				Tier:    convention.TierSpecs,
			},
		},
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.0
	cfg.RetrievalMaxBlocks = 1

	// Routing mode with auth agent: ensureScopeSurvivor may force code/auth
	// into the single slot when specs/auth.md already fills it; this test
	// validates proposal mode never does the same replacement.
	blocksProposal, _, _ := BuildContextBlocks(ws, g, "", []string{"auth", "jwt"}, cfg)
	if len(blocksProposal) != 1 {
		t.Fatalf("expected 1 block in proposal mode, got %d", len(blocksProposal))
	}
	// Without scope survivor, the spec block (high trust) must win.
	if blocksProposal[0].File != "specs/auth.md" {
		t.Errorf("T2: proposal mode should keep highest-trust block; got %s", blocksProposal[0].File)
	}
}

// TestRetrievalForProposal_ContractT1_TrustBoostPreserved verifies that trust
// boosts (specs > docs) still apply in proposal mode — the same scoring pipeline
// as routing mode. This is the T1 "shared core" contract.
func TestRetrievalForProposal_ContractT1_TrustBoostPreserved(t *testing.T) {
	ws := t.TempDir()
	// Identical content in spec and doc — only trust boost differentiates them.
	mustWrite(t, ws, "specs/api.md", "## routing\nrouting query context blocks")
	mustWrite(t, ws, "docs/guide.md", "## routing\nrouting query context blocks")

	g := &graph.Graph{
		ContentHash: "proposal-t1",
		Nodes: []graph.Node{
			{
				ID:      "spec_section:specs/api.md:routing",
				Type:    graph.NodeTypeSpecSection,
				File:    "specs/api.md",
				Section: "routing",
				Name:    "routing",
				Tier:    convention.TierSpecs,
			},
			{
				ID:      "doc:docs/guide.md:routing",
				Type:    graph.NodeTypeDoc,
				File:    "docs/guide.md",
				Section: "routing",
				Name:    "routing",
				Tier:    convention.TierDocs,
			},
		},
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.0
	cfg.RetrievalMaxBlocks = 2

	terms := TokenizeQuery("routing query context", cfg.UseBigrams)
	blocksProposal, _, _ := BuildContextBlocks(ws, g, "", terms, cfg)
	if len(blocksProposal) < 2 {
		t.Fatalf("T1: expected 2 blocks in proposal mode, got %d", len(blocksProposal))
	}
	if blocksProposal[0].File != "specs/api.md" {
		t.Errorf("T1: specs must rank above docs on equal BM25 (trust boost); got %s first", blocksProposal[0].File)
	}
}

// TestRetrievalForProposal_ContractT1_SharedCoreWithRouting verifies that for
// queries with a clear routing winner, the top context blocks from proposal mode
// overlap with those from routing mode. The only expected difference is ranking
// within the set (affinity boost may elevate scope-connected blocks in routing).
func TestRetrievalForProposal_ContractT1_SharedCoreWithRouting(t *testing.T) {
	ws := t.TempDir()
	mustWrite(t, ws, "specs/auth.md", "## mfa\nMFA TOTP authentication multi-factor")
	mustWrite(t, ws, "specs/security.md", "## policy\nsecurity policy vulnerability review")
	mustWrite(t, ws, "docs/guide.md", "## onboarding\nonboarding guide setup")

	g := &graph.Graph{
		ContentHash: "proposal-t1-shared",
		Nodes: []graph.Node{
			{
				ID:    "agent:auth",
				Type:  graph.NodeTypeAgent,
				Name:  "auth",
				Scope: []string{"code/auth/**"},
			},
			{
				ID:      "spec_section:specs/auth.md:mfa",
				Type:    graph.NodeTypeSpecSection,
				File:    "specs/auth.md",
				Section: "mfa",
				Name:    "mfa",
				Tier:    convention.TierSpecs,
			},
			{
				ID:      "spec_section:specs/security.md:policy",
				Type:    graph.NodeTypeSpecSection,
				File:    "specs/security.md",
				Section: "policy",
				Name:    "policy",
				Tier:    convention.TierSpecs,
			},
			{
				ID:      "doc:docs/guide.md:onboarding",
				Type:    graph.NodeTypeDoc,
				File:    "docs/guide.md",
				Section: "onboarding",
				Name:    "onboarding",
				Tier:    convention.TierDocs,
			},
		},
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.0
	cfg.RetrievalMaxBlocks = 3

	terms := TokenizeQuery("MFA TOTP authentication", cfg.UseBigrams)

	blocksRouted, _, _ := BuildContextBlocks(ws, g, "auth", terms, cfg)
	blocksProposal, _, _ := BuildContextBlocks(ws, g, "", terms, cfg)

	// Both modes must surface specs/auth.md (highest BM25 for this query).
	routedFiles := fileSet(blocksRouted)
	proposalFiles := fileSet(blocksProposal)

	if !proposalFiles["specs/auth.md"] {
		t.Errorf("T1: proposal mode missing specs/auth.md; blocks = %v", blocksProposal)
	}
	if !routedFiles["specs/auth.md"] {
		t.Errorf("T1: routing mode missing specs/auth.md; blocks = %v", blocksRouted)
	}

	// The low-relevance doc block should not displace the spec blocks in either mode.
	for mode, blocks := range map[string][]ContextBlock{"routed": blocksRouted, "proposal": blocksProposal} {
		if len(blocks) > 0 && strings.HasPrefix(blocks[0].File, "docs/") {
			t.Errorf("T1 %s: docs/ block ranked above specs/ — trust boost broken; first = %s", mode, blocks[0].File)
		}
	}
}

// TestBuildContextBlocks_ProposalMode_NoScopeSurvivorEnforced is an explicit
// unit test that proposal mode (agentName="") never replaces a high-trust block
// with a scope-connected file when max_blocks=1. Mirrors the routing behavior
// test in TestBuildContextBlocks_EnsuresScopeSurvivorAfterTruncation.
func TestBuildContextBlocks_ProposalMode_NoScopeSurvivorEnforced(t *testing.T) {
	ws := t.TempDir()
	// specs/top.md has stronger BM25 signal (repeated term).
	mustWrite(t, ws, "specs/top.md", "## top\napi api api api api")
	mustWrite(t, ws, "code/win/impl.md", "## impl\napi")

	g := &graph.Graph{
		ContentHash: "proposal-no-scope",
		Nodes: []graph.Node{
			{
				ID:    "agent:winner",
				Type:  graph.NodeTypeAgent,
				Name:  "winner",
				Scope: []string{"code/win/**"},
			},
			{
				ID:      "spec_section:specs/top.md:top",
				Type:    graph.NodeTypeSpecSection,
				File:    "specs/top.md",
				Section: "top",
				Name:    "top",
				Tier:    convention.TierSpecs,
			},
			{
				ID:      "doc:code/win/impl.md:impl",
				Type:    graph.NodeTypeDoc,
				File:    "code/win/impl.md",
				Section: "impl",
				Name:    "impl",
				Tier:    convention.TierDocs,
			},
		},
	}

	cfg := DefaultScoringConfig()
	cfg.RetrievalMinRelevance = 0.0
	cfg.RetrievalMaxBlocks = 1

	// Routing mode: ensureScopeSurvivor forces code/win/impl.md into the slot.
	blocksRouted, _, _ := BuildContextBlocks(ws, g, "winner", []string{"api"}, cfg)
	if len(blocksRouted) != 1 || blocksRouted[0].File != "code/win/impl.md" {
		t.Fatalf("routing mode: expected scope survivor code/win/impl.md, got %v", blocksRouted)
	}

	// Proposal mode: no scope survivor; specs/top.md wins on BM25+trust.
	blocksProposal, _, _ := BuildContextBlocks(ws, g, "", []string{"api"}, cfg)
	if len(blocksProposal) != 1 {
		t.Fatalf("proposal mode: expected 1 block, got %d", len(blocksProposal))
	}
	if blocksProposal[0].File != "specs/top.md" {
		t.Errorf("proposal mode: expected specs/top.md (no scope survivor), got %s", blocksProposal[0].File)
	}
}

func fileSet(blocks []ContextBlock) map[string]bool {
	m := make(map[string]bool, len(blocks))
	for _, b := range blocks {
		m[b.File] = true
	}
	return m
}
