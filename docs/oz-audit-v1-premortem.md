# Pre-Mortem: oz audit V1

**Date**: 2026-04-19
**Status**: Closed — V1 shipped (2026-04-20)
**PRD**: docs/oz-audit-v1-prd.md
**Sprint plan**: docs/oz-audit-v1-sprints.md

---

## Risk Summary

- **Tigers**: 5 (all resolved or explicitly tracked post-V1 — see checklist below)
- **Paper Tigers**: 3 (closed as non-issues)
- **Elephants**: 2 (acknowledged in PRD / docs)

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|---|---|---|---|---|---|
| AT-01 | **Drift identifier scanner produces a wall of false positives.** | High | Critical | **Resolved:** scanner narrowed (no underscores in default Go identifier pattern; optional `specscan.Options.IdentPatterns`). Self-validation: `DRIFT002` = 0 on this repo. Documented in PRD §6 and `internal/audit/drift/specscan`. | oz-coding | Sprint A5 |
| AT-02 | **In-memory `context.Build` rebuild is non-deterministic relative to on-disk graph because of inputs that drift between runs.** `STALE001` compares an in-memory rebuild against on-disk hash. If the rebuild reads files that change naturally between commits (timestamps in markdown, OS-level metadata, line endings), the hash will mismatch even when the workspace is "clean". This would make `STALE001` fire on every CI run and break the auditor exit code immediately. | Medium | Critical | Before Sprint A3 ends, write a determinism test: scaffold a workspace, run `oz context build`, then run `audit/staleness` and assert no `STALE001`. If it fires, the staleness logic is wrong (or `context.Build` is non-deterministic — which would be a separate bug to fix in `internal/context`). The test must be the gate. Audit cannot ship if staleness has even a 1% false-positive rate. | oz-coding | End of Sprint A3 |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Planned Response | Owner |
|---|---|---|---|---|---|
| AT-03 | **`COV002` and `COV003` produce noise on the oz repo because `agents/oz-coding` claims `code/` and the convention is loose.** The Responsibilities section says "implement and extend `oz` subcommands" with informal scope hints, not strict path declarations. If `COV002` (top-level `code/` dir not owned) and `COV003` (overlapping ownership) fire on this repo, the warning class loses meaning. | Medium | Medium | Sprint A2 ends with a self-validation pass on the oz repo. Tune the coverage check so it walks down only one level under `code/` (not into nested subdirs) and treats glob/wildcard scope paths as intentionally broad. Document the tuning in the PRD §6 "Solution Overview". If still noisy, mark `COV002`/`COV003` as `info` and revisit in V2 once a `scope:` frontmatter or stricter ownership convention exists. | oz-coding |
| AT-04 | **The Go AST extractor sees test fixtures under `code/` and pollutes the symbol set.** | Medium | Medium | **Resolved:** `codeindex.WalkCode` skips `testdata/` and skips `*_test.go` for graph builds; drift `--include-tests` only merges `*_test.go`, not fixture trees. Symbols for drift default to graph `code_symbol` nodes (ADR 0001). | oz-coding |

---

## Track Tigers

*Monitor post-V1 — trigger conditions noted.*

| # | Risk | Trigger to act |
|---|---|---|
| AT-05 | **Audit runtime exceeds 1s on real-world workspaces.** The `< 1s on the oz repo` target is set on a small, all-Go workspace. Larger monorepos with dozens of agents and thousands of Go files may slow `drift/govet` significantly, since `go/parser` is single-threaded. If users complain or benchmarks regress, parallelise the extractor: parse files concurrently with a `runtime.NumCPU()` worker pool. The deterministic-output guarantee still holds because findings are sorted before render. | First post-V1 issue reporting `oz audit` taking > 5s, or a deliberate stress benchmark on a 500-file Go workspace exceeding 3s. |

---

## Paper Tigers

**AP-01 — "Tree-sitter is required for credible AST drift."**
Not a real risk. The relevant property is *language-aware symbol extraction*, not the specific parser. Go's stdlib `go/parser` produces a complete AST and the only language in `code/` is Go. Tree-sitter is the right answer when extractor #2 (TypeScript, Python) lands; it is wrong-cost in V1.

**AP-02 — "Audit will fight `oz validate` and produce duplicate output."**
Not a real risk. Validate is per-file structural (does this AGENT.md have section X?). Audit is cross-file relational (does this AGENT.md mention a path that exists?). They share zero check codes. CI typically runs both and developers get a clean separation. The only overlap is `STALE003` (unreviewed semantic nodes) which intentionally surfaces the existing validate warning so audit's exit code can react to it — that overlap is a feature, not a duplication.

**AP-03 — "Findings as JSON is over-engineering for V1."**
Not a real risk. The marginal cost is one `encoding/json` call. The marginal value is real: MCP integration in V2, CI bots that diff finding sets across PRs, and `oz crystallize` consuming drift findings as input. JSON-first is cheaper than retrofitting later.

---

## Elephants in the Room

**AE-01 — Drift detection without an LLM is a heuristic, not an oracle.**
**Acknowledged** in PRD §6 / architecture: V1 catches path/symbol reference drift and
never-mentioned exports; semantic drift remains V2 (A-15).

**AE-02 — The auditor's value depends on the workspace following the convention.**
**Acknowledged**; documented in PRD §4 and `docs/open-items.md` for this repo’s accepted
warn-level noise (`COV002`, `DRIFT003`, etc.).

---

## Go / No-Go Checklist

### Before Sprint A1 (runner) begins
- [x] PRD §6 finding code catalogue reviewed and frozen — codes are immutable after this point.
- [x] AT-02 test strategy agreed: a determinism test for `STALE001` is committed in Sprint A3, run in CI.

### Phase 1 (A1–A3) definition of done
- [x] AT-02 resolved: determinism test committed; staleness false-positive rate is zero on a fresh build.
- [x] AT-03 monitored: coverage check noise on oz repo measured. If > 3 warnings, tuning applied before exit.
- [x] `oz audit --only=orphans,coverage,staleness` against this repo exits 0 (with documented accepted warns).

### Before Sprint A4 (drift) begins
- [x] Decision recorded: `go/parser` via code indexing is the V1 symbol source. Tree-sitter is V2 scope. ADR 0001.
- [x] Drift fixtures under `internal/audit/drift/testdata/`; `testdata/` paths skipped by walker (AT-04).

### Phase 2 (A4–A5) definition of done
- [x] AT-01 resolved: zero `DRIFT002` on this repo; narrowing in specscanner + PRD §6.
- [x] AT-04 resolved: policy via graph + `codeindex` skips; `--include-tests` limited to `*_test.go`.
- [x] `oz audit drift` / `oz audit` exit 0 under default `--exit-on=error` with documented warn noise.

### Ship (Sprint A6) definition of done
- [x] P0 user stories from the PRD accepted (except unshipped codes `DRIFT004`/`DRIFT900`, tracked as V2).
- [x] Launch-blocking and fast-follow Tigers resolved or explicitly documented.
- [x] AT-05 baseline: `go test ./internal/audit -bench=BenchmarkAuditAll` (fixture workspace ≪ 1s/op on dev hardware).
- [x] `oz audit` against this repo: errors only when `graph.json` stale; otherwise see `docs/open-items.md`.
- [x] `docs/architecture.md`, `specs/oz-project-specification.md`, `context/implementation/summary.md` updated.
- [x] `go test ./... -race` passes with no skips.

---

*Ship checklist cleared 2026-04-20.*
