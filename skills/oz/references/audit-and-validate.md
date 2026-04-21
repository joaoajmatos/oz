# Reference: `oz validate` and `oz audit`

## `oz validate`

Structural checks: required files and directories, `OZ.md` manifest fields, agent `AGENT.md` sections, and related convention rules.

Run after changing **templates**, **`agents/`**, **`OZ.md`**, **`AGENTS.md`**, or **top-level workspace layout**. Fix **errors** before trusting `graph.json` or audit output that depends on a valid workspace.

---

## `oz audit`

Run from the workspace root. Most checks need an up-to-date **`context/graph.json`** — run **`oz context build`** first when the tree changed.

| Command | Purpose |
|--------|---------|
| `oz audit` | All registered checks. |
| `oz audit staleness` | On-disk graph hash vs live workspace; semantic overlay warnings (e.g. STALE002, STALE003). |
| `oz audit drift` | Markdown code references vs `code_symbol` nodes in the graph. |
| `oz audit orphans` | Inbound-edge / orphan hygiene from the graph. |
| `oz audit graph-summary` | Node and edge counts (sanity check). |

### Common flags

| Flag | Applies to | Effect |
|------|----------------|--------|
| `--json` | `oz audit` (aggregator) | Machine-readable report. |
| `--severity` | audit | Minimum severity to print (`error`, `warn`, `info`). |
| `--exit-on` | audit | Exit non-zero on `error`, `warn`, or `none`. |
| `--include-docs` | drift (via parent) | Also scan `docs/` for code references. |
| `--include-tests` | drift (via parent) | Include exported symbols from `*_test.go`. |

Example: `oz audit drift --include-docs` when spec and **docs** references must both match the graph.

### Severity handling

Treat **errors** as blocking. Treat **warnings** as follow-ups. File intentional gaps in **`docs/open-items.md`** when the team accepts known debt.
