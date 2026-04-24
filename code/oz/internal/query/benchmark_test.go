package query_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/testws"
)

// BenchmarkQueryWarmCache measures steady-state query latency after the
// retrieval body-token cache has been populated for the current graph hash.
func BenchmarkQueryWarmCache(b *testing.B) {
	ws := testws.New(b).
		WithAgent("oz-coding",
			testws.Scope("code/oz/**", "specs/**", "docs/**"),
			testws.Role("Implements the oz CLI and query engine"),
			testws.ReadChain("specs/audit-catalogue.md", "docs/architecture.md"),
		).
		WithAgent("oz-spec",
			testws.Scope("specs/**"),
			testws.Role("Maintains architectural specs and ADRs"),
			testws.ReadChain("specs/oz-project-specification.md"),
		).
		WithSpec("specs/audit-catalogue.md",
			testws.Section("DRIFT001 spec and code divergence", "Detect divergence between spec references and exported symbols."),
			testws.Section("DRIFT002 symbol tracking", "Track symbol movement across packages."),
		).
		WithSpec("specs/oz-project-specification.md",
			testws.Section("Context Query", "Route queries and retrieve relevant workspace context."),
		).
		WithDecision("0001-use-go-parser", "Use go/parser for exported symbol extraction in V1.").
		WithContextSnapshot("architecture", "overview.md", "Architecture overview for query, audit, and context pipelines.").
		WithNote("planning/audit-v1-prd.md", "Rationale for drift auditing and retrieval requirements.").
		Build()
	benchQueries := []string{
		"how is drift detection implemented in the audit package",
		"what does the audit catalogue say DRIFT001 detects",
		"where is the architecture overview context snapshot for oz packages",
	}

	// Warm retrieval body cache and graph load once before timing.
	for _, q := range benchQueries {
		_ = query.Run(ws.Path(), q)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = query.Run(ws.Path(), benchQueries[i%len(benchQueries)])
	}
}

// BenchmarkQueryColdCache measures query latency when the retrieval body-token
// cache is invalidated before each run. This approximates cold-cache behavior.
func BenchmarkQueryColdCache(b *testing.B) {
	ws := testws.New(b).
		WithAgent("oz-coding",
			testws.Scope("code/oz/**", "specs/**", "docs/**"),
			testws.Role("Implements the oz CLI and query engine"),
			testws.ReadChain("specs/audit-catalogue.md", "docs/architecture.md"),
		).
		WithAgent("oz-spec",
			testws.Scope("specs/**"),
			testws.Role("Maintains architectural specs and ADRs"),
			testws.ReadChain("specs/oz-project-specification.md"),
		).
		WithSpec("specs/audit-catalogue.md",
			testws.Section("DRIFT001 spec and code divergence", "Detect divergence between spec references and exported symbols."),
			testws.Section("DRIFT002 symbol tracking", "Track symbol movement across packages."),
		).
		WithSpec("specs/oz-project-specification.md",
			testws.Section("Context Query", "Route queries and retrieve relevant workspace context."),
		).
		WithDecision("0001-use-go-parser", "Use go/parser for exported symbol extraction in V1.").
		WithContextSnapshot("architecture", "overview.md", "Architecture overview for query, audit, and context pipelines.").
		WithNote("planning/audit-v1-prd.md", "Rationale for drift auditing and retrieval requirements.").
		Build()
	benchQueries := []string{
		"how is drift detection implemented in the audit package",
		"what does the audit catalogue say DRIFT001 detects",
		"where is the architecture overview context snapshot for oz packages",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.ResetRetrievalBodyCacheForBenchmark()
		_ = query.Run(ws.Path(), benchQueries[i%len(benchQueries)])
	}
}

