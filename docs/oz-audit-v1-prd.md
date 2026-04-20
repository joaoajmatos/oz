# Product Requirements Document: oz audit (V1)

**Author**: oz-spec
**Date**: 2026-04-19
**Status**: Draft
**Stakeholders**: oz-coding, oz-spec, oz-maintainer
**Supersedes**: the `oz audit` block in `specs/oz-project-specification.md` (§ "oz audit (partial)") — once V1 ships, the spec will be updated to match.

---

## 1. Executive Summary

`oz audit` is the oz binary's drift detector. It reads the structural graph produced by `oz context build` and the workspace itself, runs a battery of structural and code↔spec checks, and emits a deterministic, severity-tagged findings report.

V1 ships two layers of checks:

- **Tier A — structural** (graph-only, no code parsing): orphan documents, scope coverage, ownership conflicts, graph staleness, semantic-overlay staleness.
- **Tier B — code↔spec drift** (Go AST today, pluggable extractor for other languages): spec mentions of `code/...` paths or Go symbols that no longer resolve; exported Go symbols never mentioned in any spec or doc.

V1 closes the loop the convention has promised since day one: *"code is the source of truth for behaviour; when code and spec diverge, the spec is flagged"*. Until now that promise has been a comment in `specs/oz-project-specification.md`. After V1 it is enforced by tooling and visible in CI.

---

## 2. Background & Context

The oz convention defines a strict source-of-truth hierarchy and a "code wins, spec follows" policy. `oz validate` already enforces *structure* (required files, AGENT.md sections, unreviewed semantic nodes). `oz context` already produces a deterministic graph and routing layer. Neither of them looks at the relationship between specs and code, and neither tells a developer whether the workspace they shipped yesterday still describes the code they are shipping today.

Today's `oz audit` ([code/oz/cmd/audit.go](../code/oz/cmd/audit.go)) is a stub — it loads `context/graph.json` and prints a node/edge summary. It exists only to prove that `oz context build` produces a consumable artifact (acceptance criterion C-05 of the context PRD).

V1 turns the stub into a real auditor. It is the first oz tool that:

- treats the structural graph as input rather than output;
- reads code (not just markdown);
- reports findings as structured data, not just human prose.

The structural-graph dependency is also why this is the right next step after context V1 and not before. Every Tier A check is a graph traversal. Tier B uses the graph as the index of what specs exist and where they live.

This PRD adopts the V1 framing of an MVP: pure-Go drift, single-language extractor (Go), no LLM. Multi-language extractors and LLM-assisted semantic drift are explicit V2 scope.

---

## 3. Objectives & Success Metrics

### Goals

1. A developer (or CI run) can answer "is this workspace internally consistent?" with a single `oz audit` invocation that exits non-zero on real problems.
2. Every finding has a stable code, severity, location, and remediation hint — so findings can be diffed across runs and grepped in CI logs.
3. `oz audit --json` produces a schema-versioned, machine-readable report that downstream tools (MCP server, CI bots, future `oz crystallize`) can consume without re-parsing human output.
4. Spec/code drift in Go projects is detected at the file-path and exported-symbol level, with no false positives on a clean oz repo.
5. The auditor is composable: each check is its own subcommand and can be filtered or scoped by a flag — large workspaces should never be forced into a single 30-second run when they only want one signal.

### Non-Goals

1. **Multi-language code extraction.** V1 ships a Go extractor only. The extractor surface is an interface so other languages can be added in V2; V1 does not ship them.
2. **Tree-sitter integration.** V1 uses Go's standard library (`go/parser`, `go/ast`). Tree-sitter is a V2 option once a non-Go extractor is needed; it adds CGo and binary-size cost we do not need today. (See Decision §7.)
3. **LLM-assisted semantic drift.** Asking an LLM "does this spec section still describe this code?" is deferred. V1 is fully offline.
4. **Auto-fix.** V1 reports drift; it does not rewrite specs or code. Auto-fix would couple audit to the specs writer and is its own product.
5. **Watch mode / git hooks.** V1 is on-demand only. Hooks are a V2 packaging concern.
6. **Cross-workspace drift.** V1 scopes to a single workspace root, identical to `oz context`.

### Success Metrics

| Metric | Target | Measurement |
|---|---|---|
| Clean oz workspace exits 0 | `oz audit` against this repo exits 0 (no `error`-severity findings) | CI step in this repo |
| Drift detection precision | Zero false-positive `error` findings on the clean oz repo across all checks | Self-validation suite (A4 fixtures + this repo) |
| Drift detection recall | A deliberately-drifted fixture (renamed Go symbol still cited in a spec; deleted `code/...` path still cited; orphaned spec section) produces ≥ 1 `error` per planted defect | `internal/audit/audit_e2e_test.go` |
| Audit runtime | `oz audit` completes in < 1s on the oz repo (~80 nodes, ~50 Go files) | Benchmark |
| Determinism | `oz audit --json` produces byte-identical output across two consecutive runs on an unchanged workspace | Determinism test |
| Token cost for MCP consumers | `oz audit --json` report ≤ 5KB on the clean oz repo | Measured at ship time |

---

## 4. Target Users & Segments

**Primary: oz workspace developers (humans + their CI)**
The audit runner is a CI gate. A developer pushes a PR; CI runs `oz audit`; failures block the merge. Locally, developers run `oz audit drift` after large refactors to surface stale spec references before opening the PR.

**Secondary: LLMs working in oz workspaces**
An agent considering a spec change should run `oz audit --json` and read the findings before editing. Drift findings tell the LLM where the spec is already known to be wrong. This makes the auditor a complement to `oz context query`: query tells the LLM what to read; audit tells it what to distrust.

**Internal: future `oz crystallize`**
Crystallize promotes notes up the trust hierarchy. Audit findings are an input — a note that contradicts a spec section flagged as drifted is a higher-priority crystallize candidate.

---

## 5. User Stories & Requirements

### P0 — Must Have

| # | User Story | Acceptance Criteria |
|---|---|---|
| A-01 | As a developer, I can run `oz audit` and get a single human-readable report with findings grouped by severity (`error`, `warn`, `info`) and exit code 0/1/2 driven by `--exit-on` | Default output groups by severity; exit code is non-zero when any `error` finding is present. `--exit-on=warn` makes warnings also fail. `--exit-on=none` always exits 0. |
| A-02 | As a CI consumer, I can run `oz audit --json` and get a schema-versioned, deterministic findings report | JSON includes `schema_version`, `findings[]` (each with `check`, `code`, `severity`, `message`, optional `file`, `line`, `hint`, `refs[]`), and `counts` per severity. Two consecutive runs on an unchanged workspace produce byte-identical JSON. |
| A-03 | As a developer, I can run a single check by name (`oz audit orphans`, `oz audit coverage`, `oz audit staleness`, `oz audit drift`) without running the others | Each subcommand executes only its own checks. Shared flags (`--json`, `--severity`, `--exit-on`) work the same on the parent and every subcommand. |
| A-04 | As `oz audit orphans`, I report convention files that nothing references | `ORPH001` (error): `spec_section`/`decision` with zero inbound `reads`/`references`/`supports` edges. `ORPH002` (warn): `doc` node with zero inbound edges. `ORPH003` (info): `context_snapshot` not in any agent's `context_topics` and zero inbound edges. |
| A-05 | As `oz audit coverage`, I report scope declarations that don't match anything and code paths nobody owns | `COV001` (error): an `owns` edge whose target is a `path:...` pseudo-id and the path does not exist on disk. `COV002` (warn): a top-level directory under `code/` with no agent owning any prefix of it. `COV003` (warn): two agents own overlapping non-glob scope paths. |
| A-06 | As `oz audit staleness`, I report when generated artifacts are out of sync | `STALE001` (error): in-memory `context.Build` produces a `ContentHash` that differs from the on-disk `graph.json` — graph is stale. `STALE002` (warn): `semantic.json` exists and `graph_hash != graph.ContentHash`. `STALE003` (warn): unreviewed concepts/edges in `semantic.json`. |
| A-07 | As `oz audit drift`, I detect spec mentions of code that no longer exists, and exported Go symbols specs never mention | `DRIFT001` (error): a `code/...` path mentioned in `specs/` does not exist. `DRIFT002` (error): a backticked Go identifier (e.g. `Pkg.Symbol`) cited in `specs/` is not present in the Go AST extractor's symbol set. `DRIFT003` (warn): an exported symbol in `code/oz/cmd/` or `code/oz/internal/` is never mentioned in any spec or doc. |
| A-08 | As a developer, the auditor never crashes on a partially-broken workspace | A Go file that fails to parse produces an `info` finding (`DRIFT900`), not a hard error. A missing `graph.json` returns a clear error message recommending `oz context build`. A missing `semantic.json` is silent for `oz audit drift` and `oz audit orphans`; only `oz audit staleness` notes its absence as `info`. |

### P1 — Should Have

| # | User Story | Acceptance Criteria |
|---|---|---|
| A-09 | As a developer, I can scope `oz audit` to a subset of checks via `--only` | `oz audit --only=drift,orphans` runs exactly those checks and skips the rest. Unknown check names are a hard error with a list of valid names. |
| A-10 | As a developer, I can filter findings by severity via `--severity` | `oz audit --severity=warn` hides `info` findings from output (but they still appear in `--json`). `--severity=error` hides both `info` and `warn`. |
| A-11 | As a developer, I can include test files in drift extraction with `--include-tests` | By default `_test.go` files are excluded from `drift/govet`. With the flag, they are included. |
| A-12 | As a developer, I can include `docs/` in spec scanning with `--include-docs` | By default only `specs/` is scanned for code references. With the flag, `docs/` is also scanned and findings carry the doc file as `file`. |
| A-13 | As a debug helper, I can still get the V0 stub output via `oz audit graph-summary` | Subcommand prints node/edge counts (current stub behaviour). All-info severity. Exit code 0. |

### P2 — Nice to Have / Future (V2 scope)

| # | User Story | Acceptance Criteria |
|---|---|---|
| A-14 | As a developer, `oz audit` runs additional language extractors when present in the workspace | Extractor interface defined in V1; concrete TypeScript / Python / Rust extractors are V2. |
| A-15 | As a developer, `oz audit drift` calls an LLM to ask whether a spec section still describes the linked code | LLM-assisted semantic drift via OpenRouter. Tagged `INFERRED` like the semantic overlay. Reviewed by `oz context review`-style flow. |
| A-16 | As a developer, `oz audit watch` runs continuously and re-runs affected checks on file change | File watcher integration. Out of V1 scope. |
| A-17 | As an MCP client, I can call `audit_run` and `audit_check` tools over the MCP server | MCP tools that wrap the Tier A and Tier B checks for agent consumption. V2 once V1 JSON shape is stable. |

---

## 6. Solution Overview

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  cmd/audit.go (parent cobra command)                         │
│  Subcommands: orphans · coverage · staleness · drift ·       │
│               graph-summary · (no-arg = run all)             │
└─────────────────────────┬────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│  internal/audit/audit.go                                     │
│  Finding · Severity · Check interface · Report · RunAll      │
└─────┬──────────┬──────────────┬──────────────┬───────────────┘
      │          │              │              │
      ▼          ▼              ▼              ▼
 ┌────────┐ ┌─────────┐ ┌────────────┐  ┌──────────────────┐
 │orphans │ │coverage │ │ staleness  │  │ drift             │
 │(graph) │ │(graph + │ │(rebuild +  │  │ ┌──────────────┐ │
 │        │ │ disk)   │ │ overlay)   │  │ │ govet (Go    │ │
 │        │ │         │ │            │  │ │ AST extract) │ │
 └────────┘ └─────────┘ └────────────┘  │ ├──────────────┤ │
                                         │ │ specscan     │ │
                                         │ └──────────────┘ │
                                         └──────────────────┘
```

All checks read through the existing structural graph (`internal/graph` + `context.LoadGraph`) and the existing semantic overlay package (`internal/semantic`). Audit adds no new on-disk state.

### Findings model

```go
type Severity string // "error" | "warn" | "info"

type Finding struct {
    Check    string   `json:"check"`     // e.g. "orphans", "drift"
    Code     string   `json:"code"`      // stable id, e.g. "DRIFT001"
    Severity Severity `json:"severity"`
    Message  string   `json:"message"`
    File     string   `json:"file,omitempty"`     // workspace-relative
    Line     int      `json:"line,omitempty"`
    Hint     string   `json:"hint,omitempty"`
    Refs     []string `json:"refs,omitempty"`     // related node IDs or paths
}

type Report struct {
    SchemaVersion string             `json:"schema_version"` // "1"
    Counts        map[Severity]int   `json:"counts"`
    Findings      []Finding          `json:"findings"`
}
```

Deterministic ordering: `findings[]` is sorted by `(severity rank, check, code, file, line, message)`. Severity ranks: `error=0`, `warn=1`, `info=2` (i.e. errors first).

### Check catalogue

| Code | Severity | Check | Trigger |
|---|---|---|---|
| `ORPH001` | error | orphans | `spec_section` or `decision` node with no inbound `reads`/`references`/`supports` edges |
| `ORPH002` | warn | orphans | `doc` node with no inbound edges |
| `ORPH003` | info | orphans | `context_snapshot` not in any agent's `context_topics` and no inbound edges |
| `COV001` | error | coverage | `owns` edge with `path:...` target whose path doesn't exist on disk |
| `COV002` | warn | coverage | top-level directory under `code/` not owned by any agent |
| `COV003` | warn | coverage | two agents own overlapping non-glob scope paths |
| `COV004` | info | coverage | agent declares scope paths but Responsibilities text is empty |
| `STALE001` | error | staleness | `context.Build` (in-memory) hash differs from on-disk `graph.json` hash |
| `STALE002` | warn | staleness | `semantic.json` exists and its `graph_hash` differs from current graph hash |
| `STALE003` | warn | staleness | `semantic.json` contains nodes/edges with `reviewed: false` |
| `STALE004` | info | staleness | no `semantic.json` present (informational, not a problem) |
| `DRIFT001` | error | drift | spec text mentions a `code/...` path that does not exist |
| `DRIFT002` | error | drift | spec text mentions a Go identifier the AST extractor never found |
| `DRIFT003` | warn | drift | exported symbol in `code/oz/cmd/` or `code/oz/internal/` not mentioned in any spec or doc |
| `DRIFT004` | info | drift | spec mentions a symbol that exists but in a different file than the graph entry suggests |
| `DRIFT900` | info | drift | a Go file failed to parse (extractor tolerated, surfaced as info) |

Codes are stable. New checks get new code numbers; renames are forbidden after ship.

### Drift extractor

```go
package drift

type Symbol struct {
    Pkg   string // import path relative to module root
    Name  string // exported identifier
    Kind  string // "func" | "type" | "const" | "var"
    File  string // workspace-relative
    Line  int
}

type ExtractResult struct {
    Symbols   []Symbol
    Files     []string  // every code file the extractor saw
    SkipNotes []string  // human-readable parse errors (DRIFT900)
}

type Extractor interface {
    Name() string                                // "govet"
    Match(path string) bool                      // *.go, not _test.go (unless include tests)
    Extract(root string, opts Options) (ExtractResult, error)
}
```

`govet` is the only extractor in V1. It uses `go/parser.ParseFile` with `parser.SkipObjectResolution` and walks the resulting `*ast.File` for top-level exported `FuncDecl`, `TypeSpec`, `ValueSpec` declarations. It looks up the module root by walking up from each parsed file to the nearest `go.mod`.

### Spec scanner

`specscan` walks `specs/` (and `docs/` when `--include-docs` is set) and extracts:

- backticked tokens matching `^[A-Z][A-Za-z0-9_]*(\.[A-Z][A-Za-z0-9_]*)?$` — likely Go identifiers, optionally `Pkg.Symbol`-qualified;
- markdown link targets and backtick contents starting with `code/`.

For each candidate it records the source file and line. It then cross-references against the extractor's `Symbols` and `Files` lists to produce `DRIFT001`–`DRIFT004`.

False-positive control: the identifier regex is intentionally restrictive (PascalCase, optional dotted owner). Words like `Read`, `Write`, `Build` will hit; that is the cost of doing this without an LLM. The Hint field always says "if this isn't a Go symbol, ignore — V2 will narrow this." We accept some noise in V1 in exchange for shipping.

### CLI surface

```
oz audit                    # run all checks; exit non-zero if errors
oz audit orphans            # one check
oz audit coverage
oz audit staleness
oz audit drift
oz audit graph-summary      # legacy stub output, info only

# Shared flags (parent + every subcommand):
--json                      # emit Report as JSON
--severity={error|warn|info}    # output filter (does not affect --json)
--exit-on={error|warn|none}     # exit code policy (default: error)
--only=check1,check2        # only on parent; restrict to a subset
--include-tests             # drift only
--include-docs              # drift only
```

### Key design decisions

- **`go/parser` over tree-sitter in V1.** Zero new dependency, exact for Go, keeps the binary single-file. Tree-sitter is a sensible V2 once a second language is needed. The wording in `context/implementation/summary.md` about "tree-sitter AST comparison" is the conceptual goal; the V1 implementation choice is `go/parser` and the result is the same: AST-level extraction.
- **Findings are first-class data, not log lines.** Every check returns `[]Finding`. The human renderer is a thin wrapper around the JSON renderer. This is what makes audit composable with MCP, CI bots, and future tooling.
- **Audit never writes.** It is read-only. The only artifacts are stdout/stderr text and `--json` output. This keeps audit safe to run from any context (CI, hooks, agents).
- **Audit reuses, not duplicates, validate.** Validate enforces structure on a single file (`AGENT.md` has section X). Audit enforces relationships across files (`AGENT.md` mentions a path that doesn't exist; spec mentions a symbol that doesn't exist). They are complementary; they share zero rules.
- **Stale graph is `error`, not `warn`.** A stale graph means every other check is operating on outdated data. Loud failure is better than silent inconsistency.

---

## 7. Decisions

| Question | Decision |
|---|---|
| Tree-sitter or `go/parser` for V1 drift? | `go/parser` (stdlib). Tree-sitter deferred to V2 when a non-Go extractor lands. Documented in §6 "Key design decisions". |
| Single command or subcommands? | Subcommands per check, plus parent `oz audit` that runs all. Resolves the composability requirement (A-03, A-09). |
| Where do findings live? | Returned by each check, aggregated by `Report`, rendered by either the human writer or the JSON writer. No on-disk storage in V1. |
| Should audit auto-rebuild the graph? | No. If the graph is missing, audit returns a clear error and recommends `oz context build`. If the graph is stale, audit reports `STALE001` (error) and exits non-zero. Auto-rebuild would hide a real problem. |
| Does audit call any network APIs? | No. V1 is fully offline. The LLM-assisted drift in A-15 is V2. |
| Where does the V0 stub go? | Becomes `oz audit graph-summary` (A-13). It is still a useful one-line health check. |
| What is the exit code policy by default? | Non-zero iff any `error` finding is produced. Configurable via `--exit-on`. CI defaults are sensible without flags. |
| Include `_test.go` and `docs/` by default in drift? | No. Both opt-in via `--include-tests` and `--include-docs`. Conservative default keeps the false-positive rate low on first adoption. |
| What's the relationship between `oz validate` and `oz audit`? | Validate is structural (per-file schema). Audit is relational (cross-file consistency, code↔spec drift). They share zero rules. CI typically runs `oz validate && oz audit`. |
| Are findings stable codes or human strings? | Stable codes (`ORPH001`, `DRIFT002`, ...). Codes never get reused after ship. Renames forbidden. Adds get new numbers. |

---

## 8. Timeline & Phasing

### Phase 1 — Runner + structural checks (Sprints A1–A3)
Delivers A-01, A-02, A-03, A-04, A-05, A-06, A-08, A-10, A-13.

- Findings model + parent cobra command + JSON output.
- Orphans + coverage + staleness checks ship.
- `oz audit graph-summary` preserved as legacy stub.
- The clean oz repo passes `oz audit --only=orphans,coverage,staleness` with exit code 0.

This phase delivers the structural-drift CI gate. It is independently shippable — Tier B can land later without rework.

### Phase 2 — Drift (Sprints A4–A5)
Delivers A-07, A-09, A-11, A-12.

- `drift/govet` extractor.
- `drift/specscan`.
- Drift orchestrator + four `DRIFT00x` checks + `DRIFT900` parse-error info finding.
- `--only`, `--include-tests`, `--include-docs` flags.

### Phase 3 — Hardening + ship (Sprint A6)
- Determinism test, e2e test on `internal/testws` fixtures, performance benchmark.
- `docs/architecture.md` updated. `specs/oz-project-specification.md` `oz audit` block updated from "partial" to "complete (Go drift)". `context/implementation/summary.md` audit fast-follow removed.
- Pre-mortem items closed.

### V2 scope (out of V1)
- A-14 multi-language extractors (TS/Python/Rust) — likely tree-sitter at this point.
- A-15 LLM-assisted semantic drift via OpenRouter, mirroring the `enrich`/`review` flow.
- A-16 watch mode and git hooks.
- A-17 MCP `audit_run` / `audit_check` tools.

---

*This PRD is paired with `docs/oz-audit-v1-premortem.md` (risk register) and `docs/oz-audit-v1-sprints.md` (sprint plan). Implementation begins after the pre-mortem go/no-go is signed off.*
