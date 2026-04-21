# oz audit V1 — Sprint Plan

**PRD**: notes/planning/oz-audit-v1-prd.md
**Pre-mortem**: notes/planning/oz-audit-v1-premortem.md
**Test framework**: docs/test-framework.md
**Format**: 1-week sprints, solo developer

Pre-mortem go/no-go before Sprint A1:
- [ ] PRD §6 finding-code catalogue frozen (codes are immutable after this point)
- [ ] AT-02 determinism test for `STALE001` agreed and scheduled in Sprint A3

---

## Sprint A0 — Scoping & alignment
**Goal**: Lock the PRD, pre-mortem, and finding-code catalogue. No code in this sprint.

Already produced this session:
- [x] PRD: `notes/planning/oz-audit-v1-prd.md`
- [x] Pre-mortem: `notes/planning/oz-audit-v1-premortem.md`
- [x] Sprint plan: `notes/planning/oz-audit-v1-sprints.md` (this file)
- [x] Tier scope confirmed: Tier A (structural) + Tier B (Go AST), no LLM in V1
- [x] Command shape confirmed: subcommands per check + parent `oz audit` aggregator

### Definition of done
- All three docs committed
- Finding-code catalogue (PRD §6) reviewed and frozen
- Open-items updated; no blocking pre-mortem items remain

---

## Sprint A1 — Findings model + runner skeleton
**Goal**: `oz audit` parent and subcommands wire up against an empty `Report`. No checks yet. CI can already shell out to `oz audit --json` and parse the schema.

### Stories

| # | Story | AC |
|---|---|---|
| A1-01 | New package `internal/audit/audit.go` — `Severity`, `Finding`, `Check`, `Report`, `RunAll` | Public types per PRD §6. `Severity` constants `error`/`warn`/`info`. `Check` interface: `Name() string`, `Codes() []string`, `Run(root, opts) ([]Finding, error)`. `RunAll` aggregates and sorts. Unit tested. |
| A1-02 | Deterministic finding sorter | Sort by `(severityRank, check, code, file, line, message)`. Severity ranks: `error=0`, `warn=1`, `info=2`. Sort tested with a shuffled fixture; output is byte-identical across 100 shuffles. |
| A1-03 | `internal/audit/report/` — human + JSON renderers | Human renderer groups by severity, prints `code · file:line · message · hint`. JSON renderer emits the `Report` struct with `schema_version: "1"`. Both tested. |
| A1-04 | `cmd/audit.go` becomes parent cobra command | Removes the V0 stub body; registers subcommands. Shared flags wired on parent and subcommands: `--json`, `--severity`, `--exit-on`. Unknown `--exit-on` value is a hard error. |
| A1-05 | `cmd/audit_orphans.go`, `audit_coverage.go`, `audit_staleness.go`, `audit_drift.go`, `audit_summary.go` — empty subcommand stubs | Each registers a no-op `Check` that returns `nil` findings. `oz audit` exits 0 with `{"findings":[],"counts":{}}`. |
| A1-06 | `oz audit graph-summary` — preserve V0 stub behaviour | Subcommand reproduces current node/edge count output as `info` findings (or as raw stdout for parity, decision recorded in code comment). Exit 0 always. |
| A1-07 | `--only=check1,check2` flag on parent | Restricts which checks `RunAll` invokes. Unknown name is a hard error with the list of valid names. |
| A1-08 | E2E test `internal/audit/audit_e2e_test.go` (initial) | Scaffolds a workspace via `internal/testws`, runs `oz audit --json`, asserts `schema_version: "1"`, `counts` empty, `findings: []`. |

### Sprint risks
- The cobra wiring across many subcommands is mechanically tedious — get the shared-flag plumbing right once in `cmd/audit.go` so subcommand files stay short. If shared flags are duplicated across files, the next sprint will be painful.

### Definition of done
- `oz audit`, `oz audit orphans`, `oz audit coverage`, `oz audit staleness`, `oz audit drift`, `oz audit graph-summary` all run and exit 0
- `oz audit --json` produces the empty `Report` deterministically
- `--severity`, `--exit-on`, `--only` flags parse and validate

---

## Sprint A2 — orphans + coverage
**Goal**: First real checks land. Tier A structural drift visible in CI.

### Stories

| # | Story | AC |
|---|---|---|
| A2-01 | `internal/audit/orphans/` — implements `ORPH001`/`ORPH002`/`ORPH003` | Loads graph via `context.LoadGraph`. Builds inbound-edge counts per node. Emits findings per the catalogue. Each finding has `file` (and `line` if section header is parsed; otherwise omitted), a hint pointing to `oz context build` or to the responsible agent. |
| A2-02 | `internal/audit/coverage/` — implements `COV001`/`COV002`/`COV003`/`COV004` | `COV001` walks `owns` edges with `path:` targets and `os.Stat`s. `COV002` reads `code/` top-level dirs and matches against any agent's `Scope`. `COV003` detects overlapping non-glob scope paths via prefix comparison. `COV004` checks agents whose `Scope` is non-empty but `Responsibilities` is empty. |
| A2-03 | Extend `internal/testws` golden suite for audit fixtures | New fixture set under `internal/audit/testdata/`: `clean/` (no findings), `orphan-spec/` (one orphan spec section), `unowned-code/` (one top-level code dir nobody owns), `overlap/` (two agents overlapping), `dangling-scope/` (an `owns` path that doesn't exist). Used by all check tests. |
| A2-04 | Unit tests per check package | Table-driven, deterministic, no I/O outside `t.TempDir()`. Each test asserts the exact finding codes produced for a given fixture. |
| A2-05 | Self-validation: run `oz audit orphans` + `oz audit coverage` against this repo | Document any findings in `docs/open-items.md`. If `COV002`/`COV003` produce > 3 warnings on this repo (AT-03 trigger), apply the tuning described in the pre-mortem and document the decision. |
| A2-06 | Update `cmd/audit_orphans.go` / `audit_coverage.go` to invoke real checks | Subcommand stubs from A1-05 wired to the real `Check` implementations. |

### Sprint risks
- AT-03 (coverage noise on the oz repo). Run A2-05 mid-sprint, not at the end, so any tuning lands inside the same sprint.

### Definition of done
- `oz audit orphans` and `oz audit coverage` produce real findings
- All A2 unit tests pass; e2e test extended to cover both subcommands
- AT-03 monitored: coverage warnings on this repo are either zero or explicitly documented as accepted noise

---

## Sprint A3 — staleness
**Goal**: Catch the case "you forgot to re-run `oz context build`" and the existing `semantic.json` warnings, with no false positives.

### Stories

| # | Story | AC |
|---|---|---|
| A3-01 | `internal/audit/staleness/` — `STALE001` (graph stale) | Calls `context.Build` in-memory (no disk write). Compares the resulting `ContentHash` against the on-disk `graph.json` `ContentHash`. Difference produces an `error` finding with hint `run 'oz context build'`. |
| A3-02 | `STALE002` (semantic overlay stale) | Loads `semantic.json` via `internal/semantic`. Compares its `graph_hash` to the current graph's `ContentHash`. Mismatch produces `warn` finding with hint `run 'oz context enrich'`. |
| A3-03 | `STALE003` (unreviewed semantic items) | Counts `reviewed: false` concepts/edges in `semantic.json`. Produces a single `warn` summary finding with the count. Hint: `run 'oz context review'`. |
| A3-04 | `STALE004` (no overlay present) | Optional `info` finding when `semantic.json` is absent. Off by default; surfaced only when severity filter includes `info`. |
| A3-05 | **Determinism test (resolves AT-02)** | Test scaffolds workspace, runs `oz context build`, runs `audit/staleness`, asserts no `STALE001` finding. Then calls `audit/staleness` 100 times in a row; asserts findings are byte-identical every time. |
| A3-06 | E2E coverage in `audit_e2e_test.go` | Add cases: (a) build + audit → clean; (b) build + edit a markdown file + audit → `STALE001`; (c) build + write a `semantic.json` with mismatched `graph_hash` → `STALE002`. |
| A3-07 | Wire `cmd/audit_staleness.go` to the real check | Subcommand functional. Update `cmd/audit.go` so `oz audit` aggregator includes staleness. |

### Sprint risks
- AT-02 (false-positive staleness). A3-05 is the gate — if the determinism test fails, the bug is either in `audit/staleness` or in `internal/context`. The latter would be a separate fix in a different package; in that case, pause Sprint A3 and address `internal/context` first.

### Definition of done
- Three `STALE` checks shipped and tested
- AT-02 resolved: determinism test green
- `oz audit --only=orphans,coverage,staleness` against this repo exits 0 (or fails on real, documented findings)

---

## Sprint A4 — drift foundation (graph-sourced symbols + spec scanner)
**Goal**: `internal/audit/drift` provides a `Symbol` type loaded from `graph.json` and a spec scanner that extracts code references from markdown. No findings yet — just the two primitives that Sprint A5 wires together.

> RE-SCOPED (2026-04-20) — The standalone Go AST extractor (original A4-01–A4-06) is
> superseded by the code-indexing sprint (CI-1). `oz context build` now indexes all
> exported Go symbols into `graph.json` as `code_symbol` nodes. Sprint A4 now loads
> symbols from the graph instead of re-parsing source files, and moves the spec scanner
> here from A5-01.

### Stories

| # | Story | AC |
|---|---|---|
| A4-R01 | `internal/audit/drift/drift.go` — `Symbol` type + `LoadSymbols(*graph.Graph) []Symbol` | `Symbol` has `Pkg`, `Name`, `Kind`, `File`, `Line` fields mapped from `code_symbol` graph nodes. `LoadSymbols` filters `NodeTypeCodeSymbol` nodes and returns a sorted slice. |
| A4-R02 | Unit tests for `LoadSymbols` | Table-driven: graph with zero, one, and mixed node types; assert correct Symbol slice and sort order. |
| A4-R03 | `internal/audit/drift/specscan/scan.go` — spec scanner | Walks `specs/` (and `docs/` if `Options.IncludeDocs`). For each markdown file, extracts: backticked tokens matching the default exported-Go pattern (see `DefaultGoExportedIdentPattern` in `specscan` — **underscores excluded** for AT-01), optionally OR’d with `Options.IdentPatterns`; and backtick or markdown-link targets starting with `code/`. Records `(File, Line, Candidate)`. |
| A4-R04 | Unit tests for spec scanner | Fixture files covering: exported identifier, `Pkg.Symbol` form, `code/` path link, unexported identifier (not captured), inline code vs block code. |
| A4-R05 | Self-validation | Load this repo's `context/graph.json` and call `LoadSymbols`; assert >= 50 symbols. Document actual count. Resolves AT-04 (originally testdata exclusion — now moot since goindexer already skips testdata). |

### Sprint risks
- AT-04 is resolved by CI-1 (goindexer already excludes testdata/ and _test.go). No separate test needed here.

### Definition of done
- `LoadSymbols` correctly maps all `code_symbol` nodes from a graph
- Spec scanner extracts candidates with file + line positions
- All A4 tests pass
- Self-validation: this repo's graph produces >= 50 symbols via `LoadSymbols`
- AT-04 closed

---

## Sprint A5 — drift orchestrator + spec scanner
**Goal**: `oz audit drift` produces real findings. Final feature sprint.

### Stories

| # | Story | AC |
|---|---|---|
| A5-01 | `internal/audit/drift/specscan/scan.go` — spec scanner | Same as A4-R03: default pattern `DefaultGoExportedIdentPattern` (no underscores); `IdentPatterns` override; `code/` path refs. |
| A5-02 | Drift orchestrator — wires extractor + scanner → findings | Produces `DRIFT001` (missing `code/...` path), `DRIFT002` (missing Go identifier), `DRIFT003` (exported symbol never mentioned in any spec or doc), `DRIFT004` (spec mentions symbol that exists but in a different file). |
| A5-03 | Findings carry source location | Every drift finding sets `file` to the spec file path and `line` to the markdown line where the candidate was extracted. `Refs` includes the symbol or path that triggered it. |
| A5-04 | `--include-tests` and `--include-docs` flags wired | Default: tests excluded, docs not scanned for drift. Both flags pass through to `drift.Options`. |
| A5-05 | Self-validation (resolves AT-01) | Run `oz audit drift` against this repo. If `DRIFT002` count > 5, apply the scanner narrowing described in PRD §6 and the pre-mortem AT-01 mitigation: require `Pkg.Symbol` form, or proximity to a `code/` link, or appearance inside a fenced code block. Iterate until the clean-repo `DRIFT002` count is 0. Document the chosen narrowing in PRD §6. |
| A5-06 | Drift e2e fixture | New fixtures under `internal/audit/drift/testdata/`: `clean/` (matching specs and code), `drifted-path/` (spec mentions deleted `code/...`), `drifted-symbol/` (spec mentions removed `Foo` function), `unmentioned-symbol/` (exported `Bar` not cited anywhere). Tests assert exactly the expected drift codes per fixture. |
| A5-07 | Wire `cmd/audit_drift.go` to the orchestrator | Subcommand functional; aggregator picks it up. |

### Sprint risks
- AT-01 (drift identifier false positives) is the headline risk of the entire project. A5-05 is non-negotiable — iterate until clean.
- The scanner narrowing decided in A5-05 must be documented in PRD §6 *before* the sprint closes, so downstream readers know the actual heuristic.

### Definition of done
- `oz audit drift` produces real findings against drifted fixtures
- AT-01 resolved: self-validation on this repo produces zero `DRIFT002` errors (or surfaces real, documented drift)
- All A5 tests pass

---

## Sprint A6 — Hardening + ship
**Goal**: All pre-mortem items closed. Performance verified. Documentation complete. V1 ready.

### Stories

| # | Story | AC |
|---|---|---|
| A6-01 | Performance benchmark | `go test -bench=BenchmarkAuditAll` in `internal/audit/`. Targets: < 1s on this repo (~80 graph nodes, ~50 Go files). Profile and fix if slow. |
| A6-02 | Determinism test for `oz audit --json` end-to-end | Two runs on an unchanged workspace produce byte-identical JSON. Failure indicates non-determinism in some check; fix at source. |
| A6-03 | E2E integration test | Single test: `oz init` → `oz validate` → `oz context build` → `oz audit --json` → assert empty `findings` (or expected baseline). |
| A6-04 | Self-audit pass | Run `oz audit` against this repo. All findings either fixed in-tree or documented as known/expected in `docs/open-items.md`. |
| A6-05 | `docs/architecture.md` updated | New section on `oz audit`: subcommands, finding catalogue, deterministic ordering, JSON schema. Update the existing audit row in §Components. |
| A6-06 | `specs/oz-project-specification.md` updated | `oz audit` block changes from "partial — loads graph.json and prints summary" to "complete — structural + Go drift checks, JSON output, CI exit codes". The `oz audit graph-summary` subcommand documented. |
| A6-07 | `context/implementation/summary.md` updated | Audit fast-follow item removed. New audit subsection added covering finding catalogue and JSON schema version. |
| A6-08 | All pre-mortem items verified closed | Every Tiger and Elephant in `notes/planning/oz-audit-v1-premortem.md` marked resolved or moved to a tracked V2 follow-up. |
| A6-09 | Promote a decision record | `specs/decisions/0002-go-parser-over-tree-sitter-for-audit-v1.md` (or next ADR number). Records the V1 extractor choice. |

### Definition of done
- All P0 + P1 user stories from the PRD accepted
- All launch-blocking and fast-follow Tigers from the pre-mortem resolved
- `go test ./... -race` passes with no skips
- `oz audit` against this repo exits 0 (or with documented expected findings)
- All three documentation surfaces updated

---

## Summary

| Sprint | Goal | PRD stories | Key pre-mortem items |
|---|---|---|---|
| A0 | Scoping & alignment | — | Catalogue freeze |
| A1 | Findings model + runner | A-01, A-02, A-03, A-10, A-13 | — |
| A2 | orphans + coverage | A-04, A-05 | AT-03 |
| A3 | staleness | A-06 | AT-02 |
| A4 | drift extractor | (foundation for A-07) | AT-04 |
| A5 | drift orchestrator + scanner | A-07, A-09, A-11, A-12 | AT-01 |
| A6 | Hardening + ship | All remaining | All remaining |

**Critical path**: A1 (runner) blocks every check sprint. A2 and A3 can technically run in parallel after A1 but it's a solo developer; serial. A4 must complete before A5 (drift orchestrator depends on the extractor). A6 ships everything.

**Earliest meaningful release**: end of A3. At that point the auditor has all Tier A checks and is a usable CI gate, even without drift. Tier B (drift) can be a follow-up release if needed for time.

**Total estimate**: 6 sprints solo developer (~6 weeks) including hardening. Earliest CI-gate release at end of week 3.
