package context_test

import (
	"strings"
	"testing"

	ozcontext "github.com/oz-tools/oz/internal/context"
	"github.com/oz-tools/oz/internal/query"
	"github.com/oz-tools/oz/internal/review"
	"github.com/oz-tools/oz/internal/semantic"
	"github.com/oz-tools/oz/internal/testws"
	"github.com/oz-tools/oz/internal/validate"
	"github.com/oz-tools/oz/internal/workspace"
)

// TestEndToEnd exercises the full oz V1 pipeline in a single test:
//
//	oz init (via testws scaffold)
//	→ oz validate (clean)
//	→ oz context build
//	→ oz context query
//	→ oz context enrich (simulated with a pre-built overlay)
//	→ oz context review --accept-all
//	→ oz validate (clean)
//
// S7-03 acceptance criterion.
func TestEndToEnd(t *testing.T) {
	// ── oz init (scaffold via testws) ────────────────────────────────────────

	ws := testws.New(t).
		WithAgent("backend",
			testws.Role("Builds the REST API and business logic layer"),
			testws.Scope("code/api/**"),
			testws.Responsibilities("Implements API endpoints, handlers, and middleware"),
			testws.OutOfScope("UI work"),
			testws.Rules("rules/coding-guidelines.md"),
			testws.Skills("skills/commit/"),
		).
		WithAgent("frontend",
			testws.Role("Builds the React web application"),
			testws.Scope("code/ui/**"),
			testws.Responsibilities("Implements React components and pages"),
			testws.OutOfScope("Backend logic"),
			testws.Rules("rules/coding-guidelines.md"),
			testws.Skills("skills/commit/"),
		).
		WithSpec("specs/api-design.md",
			testws.Section("Overview", "REST API following OpenAPI 3.0"),
			testws.Section("Authentication", "Bearer token authentication"),
		).
		WithDecision("0001-use-rest", "REST chosen over gRPC for simplicity and broad tooling support").
		Build()

	root := ws.Path()

	// ── oz validate (initial — must be clean) ────────────────────────────────

	w, err := workspace.New(root)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	result := validate.Validate(w)
	if !result.Valid() {
		for _, f := range result.Findings {
			if f.Severity == validate.Error {
				t.Errorf("validate: error — %s", f.Message)
			}
		}
		t.Fatal("initial oz validate failed")
	}

	// ── oz context build ─────────────────────────────────────────────────────

	buildResult, err := ozcontext.Build(root)
	if err != nil {
		t.Fatalf("context build: %v", err)
	}
	if buildResult.NodeCount == 0 {
		t.Error("context build produced 0 nodes")
	}
	if err := ozcontext.Serialize(root, buildResult.Graph); err != nil {
		t.Fatalf("context serialize: %v", err)
	}
	t.Logf("oz context build: %d nodes, %d edges", buildResult.NodeCount, buildResult.EdgeCount)

	// ── oz context query ─────────────────────────────────────────────────────

	queryResult := query.Run(root, "implement a REST API endpoint for user login")
	if queryResult.Agent == "" {
		t.Error("oz context query: no agent routed")
	}
	if queryResult.Confidence <= 0 {
		t.Error("oz context query: confidence is zero")
	}
	t.Logf("oz context query: agent=%s confidence=%.2f", queryResult.Agent, queryResult.Confidence)

	// ── oz context enrich (simulated — no network call in tests) ─────────────
	// Write a pre-built semantic overlay directly, simulating what
	// 'oz context enrich' produces after calling the LLM.

	overlay := &semantic.Overlay{
		SchemaVersion: semantic.SchemaVersion,
		GraphHash:     buildResult.Graph.ContentHash,
		Concepts: []semantic.Concept{
			{
				ID:          "concept:rest-api",
				Name:        "REST API",
				Description: "HTTP API following RESTful conventions",
				Tag:         semantic.TagExtracted,
				Confidence:  0.95,
				Reviewed:    false,
			},
			{
				ID:          "concept:react-components",
				Name:        "React Components",
				Description: "Reusable UI building blocks built with React",
				Tag:         semantic.TagExtracted,
				Confidence:  0.92,
				Reviewed:    false,
			},
		},
		Edges: []semantic.ConceptEdge{
			{
				From:       "concept:rest-api",
				To:         "agent:backend",
				Type:       semantic.EdgeTypeAgentOwnsConcept,
				Tag:        semantic.TagExtracted,
				Confidence: 0.95,
				Reviewed:   false,
			},
			{
				From:       "concept:react-components",
				To:         "agent:frontend",
				Type:       semantic.EdgeTypeAgentOwnsConcept,
				Tag:        semantic.TagExtracted,
				Confidence: 0.92,
				Reviewed:   false,
			},
		},
	}
	if err := semantic.Write(root, overlay); err != nil {
		t.Fatalf("write simulated semantic overlay: %v", err)
	}
	t.Logf("oz context enrich: wrote %d concepts, %d edges", len(overlay.Concepts), len(overlay.Edges))

	// ── oz context review --accept-all ───────────────────────────────────────

	var reviewOut strings.Builder
	summary, err := review.Run(root, review.Options{
		AcceptAll: true,
		Out:       &reviewOut,
	})
	if err != nil {
		t.Fatalf("oz context review --accept-all: %v", err)
	}
	if summary.Accepted == 0 {
		t.Error("oz context review: expected items to be accepted, got 0")
	}
	t.Logf("oz context review --accept-all: accepted %d items", summary.Accepted)

	// Verify all items are now marked reviewed.
	reviewed, loadErr := semantic.Load(root)
	if loadErr != nil {
		t.Fatalf("reload semantic.json after review: %v", loadErr)
	}
	for _, c := range reviewed.Concepts {
		if !c.Reviewed {
			t.Errorf("concept %q still unreviewed after --accept-all", c.Name)
		}
	}
	for _, e := range reviewed.Edges {
		if !e.Reviewed {
			t.Errorf("edge %q→%q still unreviewed after --accept-all", e.From, e.To)
		}
	}

	// ── oz validate (final — must still be clean with no unreviewed warning) ─

	w2, err := workspace.New(root)
	if err != nil {
		t.Fatalf("workspace.New (final): %v", err)
	}
	finalResult := validate.Validate(w2)
	if !finalResult.Valid() {
		for _, f := range finalResult.Findings {
			if f.Severity == validate.Error {
				t.Errorf("final validate error: %s", f.Message)
			}
		}
		t.Fatal("final oz validate failed")
	}
	// Confirm no unreviewed-semantic-node warning.
	for _, f := range finalResult.Findings {
		if strings.Contains(f.Message, "unreviewed") {
			t.Errorf("unexpected unreviewed warning after review: %s", f.Message)
		}
	}
	t.Logf("final oz validate: clean (no errors, no unreviewed warnings)")
}
