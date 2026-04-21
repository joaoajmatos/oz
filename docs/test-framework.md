# oz Test Framework

**Purpose**: Give every `oz` subcommand a first-class way to spin up real, convention-compliant workspaces in tests, and give `oz context` a validation environment honest enough to expose routing failures that the oz workspace itself is too small to surface.

**Resolves**: E-02 (oz's own workspace is too small to validate routing accuracy), plus general test infrastructure for all subcommands.

---

## Principles

1. **Real files, not mocks.** Tests materialize actual directories on disk that pass `oz validate`. No stubbed file-system interfaces.
2. **Declarative where possible, programmatic where necessary.** Routine workspace shapes come from YAML fixtures. Edge cases are built imperatively.
3. **Fast.** A builder call should materialize a workspace in < 50ms. CI runs hundreds of these.
4. **Self-contained.** Each test owns its workspace in `t.TempDir()`. No shared fixtures, no cross-test state.
5. **Ownable by LLMs.** Tests are readable. Adding a golden fixture does not require reading framework source.

---

## Package layout

```
code/oz/internal/testws/
├── builder.go        # fluent programmatic API
├── fixture.go        # YAML fixture loader
├── assertions.go     # domain-specific matchers (ExpectAgent, ExpectTrust, ...)
└── testdata/
    └── golden/       # routing accuracy golden suites
        ├── 01_minimal/
        ├── 02_medium/
        ├── 03_large/
        ├── 04_adversarial/
        └── 05_semantic/
```

Tests live next to the code they exercise (`internal/query/query_test.go`, `cmd/audit_test.go`) and import `internal/testws`.

---

## Programmatic builder API

```go
ws := testws.New(t).
    WithAgent("backend",
        testws.Scope("code/api/**", "code/server/**"),
        testws.Role("Builds REST endpoints and server-side logic"),
        testws.Responsibilities("Implements API handlers, middleware, request validation"),
        testws.OutOfScope("UI changes, design decisions"),
        testws.ReadChain("AGENTS.md", "specs/api-design.md", "rules/coding-guidelines.md"),
    ).
    WithAgent("frontend",
        testws.Scope("code/ui/**"),
        testws.Role("React component development"),
    ).
    WithSpec("api-design.md",
        testws.Section("overview", "..."),
        testws.Section("authentication", "..."),
    ).
    WithDecision("0001-use-grpc", "...").
    WithContextSnapshot("implementation", "current-status.md", "...").
    WithNote("brainstorm.md", "...").
    WithSemanticOverlay(testws.Overlay{
        Concepts: []testws.Concept{
            {Name: "authentication", OwnedBy: "backend"},
        },
    }).
    Build()
defer ws.Cleanup()

// ws.Path() returns the absolute path to the workspace root
// ws.Run(args...) runs oz against the workspace
// ws.ReadFile(rel) reads a workspace file
```

Under the hood, `Build()`:
1. Creates a temp dir via `t.TempDir()` (auto-cleaned by the test framework)
2. Writes `AGENTS.md`, `OZ.md`, `README.md` using the same templates the `oz init` command uses
3. Writes each agent's `AGENT.md` with the declared sections
4. Writes spec/decision/doc/context/note files with convention-compliant frontmatter and structure
5. Optionally runs `oz context build` to produce `context/graph.json`
6. Optionally materializes `context/semantic.json` from the overlay

The builder uses the same embedded templates as `oz init` — tests cannot drift from production scaffolding.

---

## Declarative YAML fixtures

For golden suites and repeated workspace shapes, the builder accepts YAML:

```yaml
# testdata/golden/02_medium/workspace.yaml
agents:
  - name: backend
    scope: ["code/api/**", "code/server/**"]
    role: "Builds REST endpoints"
    responsibilities: "Implements handlers, middleware, validation"
    out_of_scope: "UI work, design decisions"
    readchain: ["AGENTS.md", "specs/api-design.md"]
  - name: frontend
    scope: ["code/ui/**"]
    role: "React component development"
  # ... 8 more agents

specs:
  - path: api-design.md
    sections:
      overview: "..."
      authentication: "..."

decisions:
  - id: "0001-use-grpc"
    content: "..."

semantic_overlay:
  concepts:
    - name: "authentication"
      owned_by: "backend"
      edges:
        - type: "implements_spec"
          target: "api-design.md#authentication"
          tag: EXTRACTED
```

```go
ws := testws.FromFixture(t, "testdata/golden/02_medium/workspace.yaml").Build()
```

---

## Golden suites for routing accuracy validation

The core validation environment for T-01 / E-02.

```
testdata/golden/
├── 01_minimal/        # 3 agents — smoke test, mirrors oz itself
├── 02_medium/         # 10 agents — realistic project
├── 03_large/          # 25 agents with overlapping scopes — stress test
├── 04_adversarial/    # near-synonym scope declarations, deliberately ambiguous queries
└── 05_semantic/       # same as 02 + curated semantic overlay — validates concept boost
```

Each suite contains:

- `workspace.yaml` — the workspace definition
- `queries.yaml` — expected routing outcomes

```yaml
# testdata/golden/02_medium/queries.yaml
queries:
  - query: "add a new REST endpoint for user preferences"
    expected_agent: backend
    min_confidence: 0.7

  - query: "improve the button hover state on the settings page"
    expected_agent: frontend
    min_confidence: 0.7

  - query: "document the new authentication flow"
    expected_agent: docs
    min_confidence: 0.6

  - query: "refactor the queue consumer to handle back-pressure"
    expected_agent: backend
    expected_candidates: ["backend", "infra"]    # ambiguity is allowed
    min_confidence: 0.3

  - query: "what's the capital of France"
    expected_agent: null                          # no clear owner
    reason: "no_clear_owner"
```

### The accuracy test

A single Go test iterates every suite and every query:

```go
func TestRoutingAccuracy(t *testing.T) {
    suites := testws.LoadGoldenSuites(t, "testdata/golden")
    for _, suite := range suites {
        t.Run(suite.Name, func(t *testing.T) {
            ws := suite.Build(t)
            defer ws.Cleanup()

            hits, total := 0, 0
            for _, q := range suite.Queries {
                result := query.Run(ws.Path(), q.Query)
                if q.Matches(result) {
                    hits++
                }
                total++
            }
            accuracy := float64(hits) / float64(total)
            t.Logf("%s: %d/%d (%.1f%%)", suite.Name, hits, total, accuracy*100)
            if accuracy < suite.MinAccuracy {
                t.Fail()
            }
        })
    }
}
```

CI output:

```
Routing accuracy:
  01_minimal:       20/20  (100.0%)  ✓
  02_medium:        47/50  ( 94.0%)  ✓
  03_large:        118/125 ( 94.4%)  ✓
  04_adversarial:   14/20  ( 70.0%)  ✓  (min: 65%)
  05_semantic:      45/50  ( 90.0%)  ✓
  ──────────────────────────────────
  overall:         244/265 ( 92.0%)
```

### Minimum accuracy per suite

| Suite | Min accuracy | Reasoning |
|---|---|---|
| 01_minimal | 100% | 3 agents, unambiguous — any failure is a bug |
| 02_medium | 95% | Headline V1 target |
| 03_large | 90% | Stress test; some loss expected with overlapping scopes |
| 04_adversarial | 65% | Deliberately ambiguous; measures graceful degradation, not correctness |
| 05_semantic | 93% | Should *outperform* 02_medium when the overlay contributes usefully |

The `05_semantic` target being higher than its structural counterpart is the empirical test that the semantic overlay is actually worth the LLM spend. If it doesn't improve accuracy, the overlay isn't doing its job.

---

## Domain-specific assertions

Thin helpers on top of `testing.T` that produce readable failures:

```go
testws.ExpectAgent(t, result, "backend")
testws.ExpectConfidenceAtLeast(t, result, 0.7)
testws.ExpectContextBlock(t, result, "specs/api-design.md#authentication")
testws.ExpectTrustOrder(t, result, "specs", "docs", "context", "notes")
testws.ExpectExcluded(t, result, "notes/")
testws.ExpectAmbiguous(t, result)   // confidence < 0.7 and candidate_agents populated
testws.ExpectNoOwner(t, result)     // agent == null
```

---

## Routing weight tuning (maintainers only)

This is **not** end-user tooling: there is no `oz` subcommand and no repo script for parameter sweeps. Weights and BM25F knobs live in [`context/scoring.toml`](../context/scoring.toml) for real workspaces (`oz context query` loads that file from the workspace root). The golden routing suites under `code/oz/internal/query/testdata/golden/` usually **do not** ship a `context/scoring.toml`, so [`TestRoutingAccuracy`](../code/oz/internal/query/query_test.go) exercises the same defaults as [`DefaultScoringConfig()`](../code/oz/internal/query/config.go) until you add a file to a fixture on purpose.

**Typical maintainer loop**

1. From `code/oz`, run the accuracy gate:

   ```bash
   go test ./internal/query -run TestRoutingAccuracy -v
   ```

2. To try different defaults against the golden suites, edit `DefaultScoringConfig()` in `internal/query/config.go`, re-run the test, and iterate until per-suite minimums are met.

3. Mirror the chosen numbers into the repo-root [`context/scoring.toml`](../context/scoring.toml) (that file’s header already says it should match `DefaultScoringConfig()`). Keep TOML key names aligned with the `[weights]`, `[routing]`, `[bm25]`, `[fields]`, and `[tokenize]` sections — not informal aliases from planning docs.

4. Optionally sanity-check routing on this workspace with `oz context query "…"` after editing `context/scoring.toml`.

There is no supported env var or shell sweep in-tree; if you need a grid search, do it ad hoc (e.g. local shell loop over temp TOML files) outside the shipped `oz` CLI.

---

## Non-goals for V1

- **Property-based routing tests.** Hypothesis-style random workspace generation is appealing but non-deterministic in ways that complicate CI. Defer to V2.
- **Benchmarking as a test target.** Performance regressions are caught by a separate `go test -bench` pass, not by the accuracy test.
- **Cross-version compatibility fixtures.** Golden suites track the current `oz` convention. If the convention changes, fixtures are regenerated.

---

## Build order

1. `internal/testws/builder.go` — programmatic API. Needed by every Phase 1 test.
2. `internal/testws/fixture.go` — YAML loader. Needed to write golden suites.
3. `testdata/golden/01_minimal/` — smoke test. Validates the framework itself.
4. `testdata/golden/02_medium/` — the main V1 accuracy gate.
5. `internal/testws/assertions.go` — readability sugar. Can be added incrementally.
6. `testdata/golden/03_large/`, `04_adversarial/`, `05_semantic/` — land alongside or after the query engine.

The framework ships in Phase 1 *before* any routing logic is written. Weight tuning and `MIN_SCORE` / `CONFIDENCE_THRESHOLD` calibration happen against the golden suites as part of the Phase 1 definition of done.
