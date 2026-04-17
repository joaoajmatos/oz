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

## CI

- Run `go test ./...` on every pull request.
- Run `go vet ./...` and `go build ./...` as a smoke check.
- No test should require network access or write outside `t.TempDir()`.
