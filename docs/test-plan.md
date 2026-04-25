# Test Plan

> Testing strategy for the oz toolset.
> Owned by oz-coding. Reviewed for coverage gaps by oz-maintainer.

---

## Principles

- Test behaviour, not implementation. Tests verify what the binary does, not how it does it.
- Every non-trivial function in `internal/` has a unit test.
- Every subcommand has at least one integration test that exercises the full command path.
- Tests must pass on CI with no external dependencies (no network, no global state).
- Use `t.TempDir()` for all filesystem tests — never write to fixed paths.

---

## Package coverage

### `internal/convention`

No logic — constants and maps only. No tests required.

### `internal/workspace`

| Behaviour | Test type |
|---|---|
| `New` resolves relative paths to absolute | unit |
| `Valid` returns true when required root files exist | unit |
| `Valid` returns false when a required file is missing | unit |
| `ReadManifest` extracts `project` and `description` from OZ.md | unit |
| `ReadManifest` returns zero value when fields are absent | unit |
| `Agents` returns subdirectory names under `agents/` | unit |
| `Agents` returns empty slice when `agents/` is empty | unit |
| `HierarchyLayers` reports correct existence for each layer | unit |

### `internal/scaffold`

| Behaviour | Test type |
|---|---|
| `Scaffold` creates all required directories | integration |
| `Scaffold` creates all required root files with correct content | integration |
| `Scaffold` creates one `AGENT.md` per agent in `cfg.Agents` | integration |
| `Scaffold` with `CodeMode=inline` creates `code/README.md` | integration |
| `Scaffold` with `CodeMode=submodule` leaves `code/` empty | integration |
| `Scaffold` with `ClaudeMD=true` creates `CLAUDE.md` | integration |
| `Scaffold` with `ClaudeMD=false` does not create `CLAUDE.md` | integration |
| `WriteCLAUDEMD` writes correct content using name and description | integration |
| Templates render project name, description, and oz version correctly | unit |

### `internal/validate`

| Behaviour | Test type |
|---|---|
| `Validate` returns no findings for a fully valid workspace | integration |
| `Validate` returns error for missing required root file | unit |
| `Validate` returns warning for missing recommended root file | unit |
| `Validate` returns error for missing required directory | unit |
| `Validate` returns warning for missing recommended directory | unit |
| `Validate` returns error when OZ.md has no `oz standard` field | unit |
| `Validate` returns error when `oz standard` field is present but empty | unit |
| `Validate` returns error when an agent dir has no AGENT.md | unit |
| `Validate` returns error when AGENT.md is missing a required section | unit |
| `Result.Valid` returns true when there are only warnings | unit |
| `Result.Valid` returns false when there is at least one error | unit |

---

## Command integration tests

Integration tests run the full subcommand via `cobra` (not via `exec`) against a temp
directory. They assert on exit behaviour and filesystem state.

### `oz init`

| Scenario | Assertion |
|---|---|
| Happy path with default agents | workspace passes `oz validate` |
| Custom agent name and description | `agents/<name>/AGENT.md` exists with correct content |
| `--claude` flag | `CLAUDE.md` created with `@AGENTS.md` import |
| No `--claude` flag | `CLAUDE.md` not created |
| `CodeMode=submodule` | `code/README.md` not created |
| Missing project name | command returns a non-nil error |

### `oz validate`

| Scenario | Assertion |
|---|---|
| Valid workspace (scaffolded by `oz init`) | exit 0, prints `ok` |
| Missing `AGENTS.md` | exit 1, error line printed to stderr |
| Missing `OZ.md` | exit 1, error line printed to stderr |
| Missing recommended `README.md` | exit 0 (warning only), warning printed to stderr |
| Agent dir with no `AGENT.md` | exit 1, error line printed to stderr |
| AGENT.md missing `## Role` | exit 1, error line printed to stderr |

### `oz add claude`

| Scenario | Assertion |
|---|---|
| Valid workspace, no existing CLAUDE.md | `CLAUDE.md` created, exit 0 |
| CLAUDE.md already exists without `--force` | error returned, file unchanged |
| CLAUDE.md already exists with `--force` | file overwritten, exit 0 |
| Non-oz directory | error returned, exit non-zero |

---

### `oz shell run` (SHL-2 implemented baseline + extended filter set)

| Scenario | Assertion |
|---|---|
| `git status` compact mode | deterministic summary preserves staged/unstaged signal and exit code |
| `git diff` compact mode | summary includes file/change totals and preserves non-zero exits |
| `rg` compact mode | grouped output preserves file paths and representative match lines |
| `go test` compact mode | failures are preserved, passing noise is reduced |
| extended specialized filters (golden + reduction + determinism in `internal/shell/filter`) | `ls`, `find`, `tree`, `json` (`jq` + JSON stdout sniff), `git log`, `git blame`, `git show`, `go build`/`go run`/`go install`, `go vet` + `staticcheck`, `make`, `npm`/`yarn`/`pnpm`, `docker build`/`docker run`, `curl`/`wget`/`http`/`https`, `env`/`printenv`, `wc`, `diff`/`patch`, `ps`, `top -b -n*`, `df`, `cargo`, `pytest` / `python -m pytest` |
| unknown command in compact mode | generic profile applies without losing error visibility |
| unknown command in raw mode | output matches raw passthrough behavior |
| filter internal error | falls back to raw output with warning metadata |
| `--json` | schema includes token/perf fields and matched filter id |
| `--tee failures` with failing command | raw output artifact is persisted and path is reported |
| transparent rewrite enabled | command is rewritten to `oz shell run -- ...` when eligible |
| transparent rewrite excluded command | command bypasses rewrite and runs unchanged |

### `oz shell read` (implemented baseline)

| Scenario | Assertion |
|---|---|
| file input (`oz shell read <file>`) | language-aware reader selected by extension; output remains non-empty for non-empty input |
| stdin input (`oz shell read -`) | stdin is read and filtered via unknown/generic path when no extension signal exists |
| missing file mixed with valid file | valid content still emitted; error surfaced for missing path; non-zero exit status |
| safety fallback triggered | if reader returns empty for non-empty content, raw content is emitted with warning |
| `--max-lines` | deterministic truncation applied after filtering |
| `--tail-lines` | deterministic tail window preserves newline behavior |
| `--line-numbers` | output uses stable, right-aligned line numbering |
| `--json` | envelope includes language, token before/after, warnings, and exit metadata |

### `oz shell pipe` (implemented baseline)

| Scenario | Assertion |
|---|---|
| auto-detect mode (`oz shell pipe`) | stdin is compacted with a deterministic best-effort filter choice |
| explicit filter (`--filter rg`, `--filter git-diff`, etc.) | named filter is used regardless of auto-detect heuristics |
| `--passthrough` | stdin is relayed unchanged to stdout |
| `--json` | envelope includes matched filter and token reduction metrics |
| unknown filter name | command returns a validation error and exits non-zero |
| stdin size over cap | command returns an explicit size-limit error and exits non-zero |

### `oz shell gain` (SHL-3 implemented baseline)

| Scenario | Assertion |
|---|---|
| no tracking records | returns empty-state summary and exit 0 |
| tracked records present | returns aggregate totals for commands, token savings, and average reduction |
| retention boundary | excluded records older than retention window are not counted |
| `--json` output | valid schema with deterministic key set and numeric fields |
| tracking store temporarily unavailable | returns actionable error without corrupting store |

### `oz shell` transparent interception (SHL-3 baseline)

| Scenario | Assertion |
|---|---|
| suggest-mode default | command yields wrapper suggestion and does not rewrite |
| explicit rewrite opt-in | command rewrites to `oz shell run -- ...` |
| excluded command | command bypasses suggestion/rewrite |
| hooks disabled or config error | fail-open behavior leaves command unchanged |

### `oz shell` stabilization gates (SHL-4)

| Scenario | Assertion |
|---|---|
| repeated shell package test runs | no flaky failures across repeated CI-style runs |
| specialized filter failure | deterministic generic fallback and explicit warning metadata |
| concurrent tracking access | inserts/queries remain stable without sqlite lock failures |
| shell command tests | global CLI flag/env state is isolated per test |

### `oz shell run` unit and golden tests

| Behaviour | Test type |
|---|---|
| token estimator (`ceil(chars/4.0)`) | unit |
| filter strategy transforms (stats/grouping/dedupe/failure-focus) | unit |
| deterministic output for same fixture input | unit |
| fixture-based output snapshots for MVP + extended command families | golden |
| deterministic output for same fixture input across runs | golden/unit |
| exit-code propagation from wrapped command | integration |
| fallback path (no specialized filter / filter failure) | integration |
| token reduction thresholds for family fixtures + median gate | unit |
| overhead budget assertions for short commands | benchmark |

---

## CI

- Run `go test ./...` on every pull request.
- Run `go vet ./...` and `go build ./...` as a smoke check.
- No test should require network access or write outside `t.TempDir()`.
- Planned: `oz shell run` golden suites should run in CI and gate regressions in compact output schemas.
