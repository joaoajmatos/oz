# oz context V1 — Sprint Plan

**PRD**: docs/oz-context-v1-prd.md
**Pre-mortem**: docs/oz-context-v1-premortem.md
**Test framework**: docs/test-framework.md
**Format**: 1-week sprints, solo developer

Pre-mortem go/no-go checks before Sprint 1:
- [x] T-01: Scoring algorithm designed and documented (BM25F — PRD §6)
- [x] E-02: Multi-agent validation environment designed (golden suites — test-framework.md)

---

## Sprint 0 — Convention & Foundations
**Goal**: Lock the workspace convention changes so all subsequent work builds on stable ground.

Already completed this session:
- [x] AGENT.md: Rules and Skills sections added to all three agents
- [x] `specs/oz-project-specification.md`: AGENT.md required sections documented
- [x] `oz validate`: new required sections flagged in open-items
- [x] PRD §6: BM25F scoring algorithm specified (resolves T-01)
- [x] `docs/test-framework.md`: builder API and golden suites designed (resolves E-02)
- [x] `docs/oz-context-v1-premortem.md`: risk register complete

**Definition of done**: All docs committed, open-items updated, no blocking pre-mortem items remain.

---

## Sprint 1 — Test Infrastructure
**Goal**: `internal/testws` builder ships. Every subsequent sprint can write tests from day one.

### Stories

| # | Story | AC |
|---|-------|-----|
| S1-01 | Build `internal/testws/builder.go` — programmatic workspace builder | `testws.New(t).WithAgent(...).Build()` creates a real, convention-compliant workspace in `t.TempDir()`. Uses same embedded templates as `oz init`. Cleanup is automatic. |
| S1-02 | Build `internal/testws/fixture.go` — YAML workspace loader | `testws.FromFixture(t, path)` materializes a workspace from `workspace.yaml`. All fields from the builder API are expressible in YAML. |
| S1-03 | Build `internal/testws/assertions.go` — domain-specific matchers | `ExpectAgent`, `ExpectConfidenceAtLeast`, `ExpectContextBlock`, `ExpectTrustOrder`, `ExpectAmbiguous`, `ExpectNoOwner` all produce readable failure messages. |
| S1-04 | Golden suite `01_minimal` — 3-agent smoke fixture | `workspace.yaml` + `queries.yaml` with 20 queries. Minimum accuracy: 100%. Mirrors oz's own workspace shape. |
| S1-05 | Golden suite `02_medium` — 10-agent realistic fixture | `workspace.yaml` + `queries.yaml` with 50 queries. Minimum accuracy: 95%. Covers realistic project shapes. |
| S1-06 | `TestRoutingAccuracy` harness | Single test that iterates all golden suites, runs queries, reports per-suite accuracy, fails if any suite falls below its minimum. |

### Sprint risks
- Builder must use the same templates as `oz init` or tests will silently test a different convention than production. Verify template parity before writing first test.

### Definition of done
- `go test ./internal/testws/...` passes
- `TestRoutingAccuracy` (`code/oz/internal/query/query_test.go`) loads every suite under `code/oz/internal/query/testdata/golden/`, runs `query.Run` per case, and fails if any suite is below its `min_accuracy` (query engine is real — no skip)
- Fixtures are committed and human-readable

---

## Sprint 2 — Structural Graph (`oz context build`)
**Goal**: `oz context build` produces a deterministic `context/graph.json`. Resolves C-01 and C-05.

### Stories

| # | Story | AC |
|---|-------|-----|
| S2-01 | `graph.json` schema — define and document | Schema covers all node types (`agent`, `spec_section`, `decision`, `doc`, `context_snapshot`, `note`) and edge types (`reads`, `owns`, `supports`, `references`, `crystallized_from`). Schema version field included. Document in `docs/architecture.md`. |
| S2-02 | Workspace walker | Traverses workspace from root, discovers all oz-convention files, respects `.ozignore` if present. Returns typed file list. |
| S2-03 | AGENT.md parser | Extracts: name, role text, read-chain items, rules files, skills, responsibilities text, out-of-scope text, scope paths, context topics. All fields nullable — parser is tolerant. |
| S2-04 | Spec / decision / doc / note indexer | Parses markdown files in `specs/`, `docs/`, `context/`, `notes/`. Extracts section headings as nodes. Tags each node with its source-of-truth tier. |
| S2-05 | Cross-reference extractor | Finds file path references across documents. Produces `reads`, `supports`, `references` edges between nodes. |
| S2-06 | Deterministic serializer | Sorts all node and edge lists by stable key before writing. SHA256 of content (not timestamps) embedded in output. Running twice with no changes produces byte-identical output. |
| S2-07 | `oz context build` subcommand | Wires walker → parsers → extractor → serializer. Writes `context/graph.json`. Prints node/edge count summary. |
| S2-08 | Tests: build against `testws` fixtures | Build runs against `01_minimal` and `02_medium` fixtures. Tests assert minimum node/edge counts by type (not exact totals). Determinism test: run twice, serialized `graph.json` bytes must match. |
| S2-09 | `oz audit` integration point | `oz audit` reads `context/graph.json` directly. Write a stub `oz audit` that loads the graph and prints a summary — proves the contract works before audit logic is built. |

### Sprint risks
- Markdown parsing is where non-determinism sneaks in (map iteration order, encoding edge cases). Write the determinism test early and run it on every PR.

### Definition of done
- C-01 accepted: `graph.json` produced, byte-identical on re-run
- C-05 accepted: `oz audit` stub consumes graph without error
- All S2 tests passing

**Implementation note:** `crystallized_from` is a defined edge type in the schema; edges of this type are not produced until a crystallize workflow exists (see `docs/architecture.md`).

---

## Sprint 3 — Query Engine (`oz context query`)
**Goal**: `oz context query` returns accurate JSON routing packets. Resolves C-02, C-03, C-09. This is the hardest sprint.

### Stories

| # | Story | AC |
|---|-------|-----|
| S3-01 | Porter stemmer — vendored Go implementation | Deterministic, no external package. Unit tested with 20 known word/stem pairs. |
| S3-02 | Tokenizer | Lowercase → strip punctuation → split → filter stopwords → stem → optional bigrams (`stem_i_stem_j`). Bigrams gated by `use_bigrams` in `context/scoring.toml` (default `false`, per sprint risk). Deterministic. Unit tested. Stopword list committed to source. |
| S3-03 | Per-agent document builder | Builds the 5 BM25F fields per agent from graph.json nodes. Field lengths computed and stored for normalization. |
| S3-04 | BM25F scorer | Implements multi-field BM25F with configurable `k1`, `b`, and per-field weights. Returns raw scores per agent. Unit tested against known inputs. |
| S3-05 | Softmax confidence + routing decision | Converts raw scores to confidence via temperature-scaled softmax. Applies MIN_SCORE floor. Returns top agent + `candidate_agents` array when confidence < CONFIDENCE_THRESHOLD. |
| S3-06 | `scoring.toml` config loader | Reads `context/scoring.toml` if present; falls back to defaults. All BM25F parameters and thresholds configurable. |
| S3-07 | Context block builder | Given the winning agent, selects relevant spec/doc/context nodes from graph. Sorts by trust tier (specs > docs > context > notes). Excludes notes by default. Respects `--include-notes` flag. |
| S3-08 | Routing packet assembler | Combines agent, confidence, scope paths, context blocks, excluded paths into the JSON packet defined in PRD §6. |
| S3-09 | `oz context query` subcommand | Wires graph load → tokenize → score → route → output. Supports `--raw` (C-09): JSON with `query`, routing `result`, per-agent raw scores + softmax confidences, and a query-relevant `subgraph` (agents + context-block nodes and edges with both ends in that set — not the full graph). |
| S3-10 | Golden suite `03_large` — 25-agent stress fixture | `workspace.yaml` with 25 agents, overlapping scopes. 125 queries. Minimum accuracy: 90%. |
| S3-11 | Golden suite `04_adversarial` — ambiguity fixture | 20 deliberately ambiguous queries. Minimum accuracy: 65%. Tests graceful degradation. |
| S3-12 | Weight tuning pass | Run `TestRoutingAccuracy` across all suites with default weights. If accuracy targets not met, sweep key weights (`w_scope`, `w_concept`, `T`) and document chosen defaults in `context/scoring.toml`. |

### Sprint risks
- BM25F IDF on N=3 agents (minimal fixture) can produce degenerate scores. Test with `01_minimal` first — if accuracy fails, add a minimum-IDF floor before tuning weights.
- Bigrams can over-index on compound terms that appear in many agents. If bigrams hurt accuracy, make them optional and off by default.

### Definition of done
- C-02, C-03, C-09 accepted
- `TestRoutingAccuracy` passes all suites at or above minimums
- Default weights committed to `context/scoring.toml`

---

## Sprint 4 — oz validate: enforce new convention
**Goal**: `oz validate` enforces Rules and Skills sections in AGENT.md. Closes the open-item from Sprint 0.

### Stories

| # | Story | AC |
|---|-------|-----|
| S4-01 | Update `oz validate` AGENT.md checks | Check for all 7 required sections: Role, Read-chain, Rules, Skills, Responsibilities, Out of scope, Context topics. Report missing sections with file path and section name. |
| S4-02 | `oz validate --with-context` flag | When flag is set, runs `oz context build` after standard validation and reports node/edge count. Does not fail validation if graph build fails — reports as a warning. |
| S4-03 | Validate against `testws` fixtures | Add validation tests to `internal/testws` golden suites. All valid fixtures must pass validate. Build a deliberately invalid fixture to confirm error reporting. |
| S4-04 | Update oz workspace AGENT.md files via validate | Run `oz validate` against the oz workspace itself. All three agents should pass. Fix any issues found. |

### Definition of done
- `oz validate` reports missing Rules/Skills sections
- oz workspace passes `oz validate` cleanly
- `--with-context` flag functional

---

## Sprint 5 — Semantic Overlay (`oz context enrich`)
**Goal**: LLM enrichment pass via OpenRouter writes `context/semantic.json`. Resolves C-06.

### Stories

| # | Story | AC |
|---|-------|-----|
| S5-01 | OpenRouter client — isolated package | `internal/openrouter` package. Handles auth (`OPENROUTER_API_KEY`), model selection (`--model`), request/response, error handling. No other package in the binary uses network I/O. Unit tested with mock HTTP. |
| S5-02 | `semantic.json` schema — define and document | Schema covers: concept nodes (name, description, source files), typed edges (`implements_spec`, `drifted_from`, `semantically_similar_to`, `agent_owns_concept`), tag (`EXTRACTED`/`INFERRED`), confidence score, `reviewed` bool, `graph_hash` (SHA256 of `graph.json` at generation time). |
| S5-03 | Enrichment prompt builder | Builds structured prompt from `graph.json` subgraph. Instructs LLM to extract concept nodes and relationships. Requests JSON output conforming to `semantic.json` schema. Prompt is deterministic given the same graph input. |
| S5-04 | LLM response parser and validator | Validates LLM JSON output against schema. Rejects malformed edges. Logs skipped items. Does not fail on partial output — best-effort. |
| S5-05 | `semantic.json` writer | Merges new LLM output with any existing `semantic.json` (preserving `reviewed: true` nodes). Writes atomically. Embeds `graph_hash` of current `graph.json`. |
| S5-06 | `oz context enrich` subcommand | Wires OpenRouter client → prompt builder → parser → writer. Prints: model used, nodes extracted, edges added, cost estimate (from OpenRouter response headers if available). Supports `--model` flag. |
| S5-07 | Routing packet enriched with concepts | When `semantic.json` is present, query engine adds `relevant_concepts` field to routing packet (C-08). When absent, field omitted gracefully. |
| S5-08 | Golden suite `05_semantic` | Extends `02_medium` with a pre-built `semantic.json`. 50 queries. Minimum accuracy: 93% (must outperform structural-only `02_medium` baseline). |

### Sprint risks
- T-06: Default model must be decided before this sprint. Pick a model with reliable JSON output mode via OpenRouter and document it. Suggested: evaluate `google/gemini-flash-1.5` or `anthropic/claude-haiku-4` on cost/quality.
- LLM output is non-deterministic. The enrichment prompt must request structured JSON strictly. Add a retry with stricter instructions if the first pass fails schema validation.

### Definition of done
- C-06 accepted: `semantic.json` produced with correct schema
- C-08 accepted: routing packets include `relevant_concepts` when overlay present
- `05_semantic` golden suite accuracy ≥ 93%
- T-06 resolved: default model documented in PRD

---

## Sprint 6 — Review Tooling + MCP Server
**Goal**: `oz context review` makes the semantic overlay actually reviewable. `oz context serve` exposes the MCP interface. Resolves C-04, C-07, T-03, T-04, T-05.

### Stories

| # | Story | AC |
|---|-------|-----|
| S6-01 | Staleness detection (T-05) | On `oz context query` and `oz context serve` startup: compare `graph_hash` in `semantic.json` against SHA256 of current `graph.json`. If mismatch, print warning: "semantic overlay may be stale — run `oz context enrich` to update". |
| S6-02 | `oz context review` command — diff view | Presents only new/changed nodes and edges since last commit (via `git diff context/semantic.json`). Formats as a table: node name, type, source file, tag, confidence. Not raw JSON. |
| S6-03 | `oz context review` — interactive accept/reject | For each unreviewed node/edge, prompts accept (sets `reviewed: true`) or reject (removes). Writes result back to `semantic.json`. Falls back to non-interactive `--accept-all` flag for CI. |
| S6-04 | `oz validate` warns on unreviewed semantic nodes | If `semantic.json` exists and contains nodes with `reviewed: false`, `oz validate` prints a warning (not an error). Exit code unaffected. Resolves E-03. |
| S6-05 | MCP stdio server — core (T-04) | `oz context serve` implements MCP stdio protocol. Handles capability negotiation, tool schema registration, request/response framing. Conformance-tested against Claude Code MCP runtime. |
| S6-06 | MCP tool: `query_graph` | Accepts task description string. Returns routing packet JSON. Identical logic to `oz context query`. |
| S6-07 | MCP tool: `get_node` | Accepts node ID. Returns node with all fields and edges. |
| S6-08 | MCP tool: `get_neighbors` | Accepts node ID and optional edge type filter. Returns adjacent nodes. |
| S6-09 | MCP tool: `agent_for_task` | Shorthand: accepts task string, returns only `agent` and `confidence`. Lower token cost for simple routing. |
| S6-10 | `.mcp.json` example in README | Shows how to wire `oz context serve` into a Claude Code or Cursor MCP config. |
| S6-11 | MCP conformance integration test | Test spawns `oz context serve` as subprocess against a `testws` fixture, exercises all four tool calls over stdio, asserts correct JSON responses. |

### Sprint risks
- T-04: MCP protocol conformance is the highest-risk item in this sprint. Build S6-05 first and validate against a real client before building tool implementations. A protocol-level error blocks everything else in this sprint.
- S6-03 (interactive review) has UX complexity. If it takes too long, ship `--accept-all` and the diff view, and defer the interactive flow to a fast-follow.

### Definition of done
- C-04 accepted: MCP server validated against Claude Code
- C-07 accepted: `oz context review` ships, `reviewed: true` workflow works
- T-03 resolved: semantic overlay is genuinely reviewable
- T-04 resolved: MCP conformance test passing
- T-05 resolved: staleness warning implemented
- E-03 resolved: `oz validate` warns on unreviewed nodes

---

## Sprint 7 — Hardening & V1 Ship
**Goal**: All pre-mortem items closed. Performance targets met. Documentation complete. V1 ready.

### Stories

| # | Story | AC |
|---|-------|-----|
| S7-01 | Performance benchmark: `oz context build` | `< 500ms` on a 50-file workspace. Benchmark committed as `go test -bench`. If slow, profile and fix before shipping. |
| S7-02 | Token efficiency measurement | Compare `oz context query` output token count vs. full workspace read for 10 representative queries. Target: ≤ 10% of full read. Document result in PRD success metrics table. |
| S7-03 | End-to-end integration test | Single test: `oz init` → `oz validate` → `oz context build` → `oz context query` → `oz context enrich` → `oz context review --accept-all` → `oz validate` (clean). Uses `testws` fixture. |
| S7-04 | oz workspace self-validation | Run `oz context build`, `oz context query`, and `oz validate` against the oz repo itself. All must pass. Fix any issues found. |
| S7-05 | `docs/architecture.md` — fill in the stub | Document the three-layer architecture, graph schema, query pipeline, and MCP interface. |
| S7-06 | `specs/oz-project-specification.md` sync | Update the `oz context` section to match the V1 implementation. Remove "coming soon" markers. |
| S7-07 | All remaining pre-mortem items verified closed | Check every Tiger and Elephant in `docs/oz-context-v1-premortem.md`. Mark resolved or create a tracked follow-up. |
| S7-08 | `MEMORY.md` context snapshot | Write `context/implementation/summary.md` capturing the oz context V1 implementation state for future agents. |

### Definition of done
- All P0 and P1 user stories from the PRD accepted
- All launch-blocking and fast-follow Tigers from pre-mortem resolved
- `go test ./...` passes with no skips
- oz workspace passes `oz validate` cleanly
- `docs/architecture.md` is no longer a stub

---

## Summary

| Sprint | Goal | PRD stories | Key pre-mortem items |
|--------|------|-------------|----------------------|
| 0 | Convention lock | — | T-01 ✓, E-02 ✓ |
| 1 | Test infrastructure | — | E-02 (implementation) |
| 2 | `oz context build` | C-01, C-05 | — |
| 3 | `oz context query` | C-02, C-03, C-09 | T-02 ✓ |
| 4 | `oz validate` updates | — | E-03 (partial) |
| 5 | `oz context enrich` | C-06, C-08 | T-06 |
| 6 | Review + MCP | C-04, C-07 | T-03, T-04, T-05, E-03 |
| 7 | Hardening + ship | All remaining | All remaining |

**Critical path**: Sprint 1 (testws) must complete before Sprint 2 (build), which must complete before Sprint 3 (query). Sprint 5 (enrich) depends on Sprint 3. Sprint 6 (MCP + review) depends on Sprint 5. Sprints 4 and 7 are largely independent.
