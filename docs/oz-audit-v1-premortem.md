# Pre-Mortem: oz audit V1

**Date**: 2026-04-19
**Status**: Open
**PRD**: docs/oz-audit-v1-prd.md
**Sprint plan**: docs/oz-audit-v1-sprints.md

---

## Risk Summary

- **Tigers**: 5 (2 launch-blocking, 2 fast-follow, 1 track)
- **Paper Tigers**: 3
- **Elephants**: 2

---

## Launch-Blocking Tigers

| # | Risk | Likelihood | Impact | Mitigation | Owner | Deadline |
|---|---|---|---|---|---|---|
| AT-01 | **Drift identifier scanner produces a wall of false positives.** The spec scanner extracts backticked PascalCase tokens as Go identifier candidates. The PRD already accepts that words like `Read`, `Write`, `Build` will hit and recommend "ignore if not a Go symbol". On the real oz repo, the spec contains many capitalized prose words that look like identifiers. If `oz audit drift` on this repo produces dozens of `DRIFT002` errors, no developer will trust the tool and it becomes a ratchet of `--only=orphans,coverage,staleness`. The CI gate value is then zero. | High | Critical | Before Sprint A4 ends, run `oz audit drift` against the oz repo. If `DRIFT002` count > 5, narrow the scanner: require fences (e.g. `Pkg.Symbol` form, or appears in a code block, or appears next to a `code/` link). Tune until clean-repo `DRIFT002` count is 0. Alternative: downgrade `DRIFT002` to `warn` until the scanner is precise. The decision must be made with empirical numbers, not a guess. | oz-coding | End of Sprint A4 |
| AT-02 | **In-memory `context.Build` rebuild is non-deterministic relative to on-disk graph because of inputs that drift between runs.** `STALE001` compares an in-memory rebuild against on-disk hash. If the rebuild reads files that change naturally between commits (timestamps in markdown, OS-level metadata, line endings), the hash will mismatch even when the workspace is "clean". This would make `STALE001` fire on every CI run and break the auditor exit code immediately. | Medium | Critical | Before Sprint A3 ends, write a determinism test: scaffold a workspace, run `oz context build`, then run `audit/staleness` and assert no `STALE001`. If it fires, the staleness logic is wrong (or `context.Build` is non-deterministic — which would be a separate bug to fix in `internal/context`). The test must be the gate. Audit cannot ship if staleness has even a 1% false-positive rate. | oz-coding | End of Sprint A3 |

---

## Fast-Follow Tigers

| # | Risk | Likelihood | Impact | Planned Response | Owner |
|---|---|---|---|---|---|
| AT-03 | **`COV002` and `COV003` produce noise on the oz repo because `agents/oz-coding` claims `code/` and the convention is loose.** The Responsibilities section says "implement and extend `oz` subcommands" with informal scope hints, not strict path declarations. If `COV002` (top-level `code/` dir not owned) and `COV003` (overlapping ownership) fire on this repo, the warning class loses meaning. | Medium | Medium | Sprint A2 ends with a self-validation pass on the oz repo. Tune the coverage check so it walks down only one level under `code/` (not into nested subdirs) and treats glob/wildcard scope paths as intentionally broad. Document the tuning in the PRD §6 "Solution Overview". If still noisy, mark `COV002`/`COV003` as `info` and revisit in V2 once a `scope:` frontmatter or stricter ownership convention exists. | oz-coding |
| AT-04 | **The Go AST extractor sees test fixtures (e.g. `internal/audit/drift/testdata/`) and treats them as real `code/` symbols, polluting the symbol set.** Test fixtures often contain symbols designed to exercise drift. If they leak into the extractor's symbol set, drift findings against the real `code/` are wrong. | Medium | Medium | Default behaviour: `drift/govet` skips any path containing `/testdata/` (Go convention — `go build` already ignores it). Document the rule. Add a test that confirms `testdata/` symbols do not appear in `ExtractResult.Symbols` even when `--include-tests` is set. | oz-coding |

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
Be honest in the PRD and the docs. `oz audit drift` V1 catches the common drift modes (renamed file, deleted symbol, never-mentioned symbol). It does not catch *semantic* drift — when a spec section still references the right code but no longer describes what it does. That requires an LLM (A-15, V2). V1's value is being right about what it claims to detect, not detecting everything.

**AE-02 — The auditor's value depends on the workspace following the convention.**
If a workspace's `AGENT.md` files lack scope paths, coverage checks degrade to `info` noise. If specs don't reference `code/...` paths or symbols, drift checks have nothing to catch. This is fine — audit is value-additive for workspaces that take the convention seriously, and silent for those that don't. We document this expectation in the PRD §4 and accept that audit is more useful in mature oz workspaces than in fresh `oz init` ones.

---

## Go / No-Go Checklist

### Before Sprint A1 (runner) begins
- [ ] PRD §6 finding code catalogue reviewed and frozen — codes are immutable after this point.
- [ ] AT-02 test strategy agreed: a determinism test for `STALE001` is committed in Sprint A3, run in CI.

### Phase 1 (A1–A3) definition of done
- [ ] AT-02 resolved: determinism test committed; staleness false-positive rate is zero on a fresh build.
- [ ] AT-03 monitored: coverage check noise on oz repo measured. If > 3 warnings, tuning applied before exit.
- [ ] `oz audit --only=orphans,coverage,staleness` against this repo exits 0.

### Before Sprint A4 (drift) begins
- [ ] Decision recorded: `go/parser` is the V1 extractor. Tree-sitter is V2 scope. (Resolved in PRD §7.)
- [ ] Test fixtures for drift live under `internal/audit/drift/testdata/` and are confirmed excluded from extraction (AT-04).

### Phase 2 (A4–A5) definition of done
- [ ] AT-01 resolved: `oz audit drift` against this repo produces zero `DRIFT002` errors. If the scanner is too noisy, scope was narrowed and the change is documented in PRD §6.
- [ ] AT-04 resolved: test confirms `testdata/` symbols are excluded from `Extract` even with `--include-tests`.
- [ ] `oz audit drift` against this repo exits 0 (or surfaces real drift, which is then fixed or documented as known).

### Ship (Sprint A6) definition of done
- [ ] All P0 user stories from the PRD accepted.
- [ ] All launch-blocking and fast-follow Tigers resolved.
- [ ] AT-05 baseline benchmark recorded.
- [ ] `oz audit` runs cleanly (or with documented findings) on this repo.
- [ ] `docs/architecture.md`, `specs/oz-project-specification.md`, `context/implementation/summary.md` updated.
- [ ] `go test ./...` passes with no skips.

---

*This pre-mortem will be updated to "All Tigers resolved — V1 shipped" once the ship checklist clears.*
