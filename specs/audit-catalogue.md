# Audit Catalogue

This document normatively defines the machine-readable report emitted by `oz audit --json` and the finding codes emitted by the shipped checks.

## Finding object

Every finding object MUST include:

- `check` (string)
- `code` (string)
- `severity` (`"error" | "warn" | "info"`)
- `message` (string)

Optional fields are:

- `file` (workspace-relative path)
- `line` (positive integer)
- `hint` (string)
- `refs` (array of related node IDs, symbols, or paths)

## Report object

The JSON report MUST be a single object with:

- `schema_version` (string): current shipped value is `"1"`.
- `counts` (object): severity counts.
- `findings` (array of finding objects).

`counts` MUST contain all three severity keys even when zero:

- `error`
- `warn`
- `info`

## Determinism

`findings` MUST be sorted by:

1. severity rank (`error` before `warn` before `info`)
2. `check`
3. `code`
4. `file`
5. `line`
6. `message`

The shipped JSON renderer uses Go's `encoding/json`; object keys are serialized in sorted order.

## Shipped finding codes

### orphans

`ORPH001` (`error`)

- A `spec_section` file or `decision` node has no inbound `reads`, `references`, or `supports` edges.

`ORPH002` (`warn`)

- A `doc` file has no inbound edges.

`ORPH003` (`info`)

- A `context_snapshot` has no inbound edges and no agent declares its topic in `context_topics`.

### coverage

`COV001` (`error`)

- An `owns` edge points to a `path:<scope>` pseudo-target that does not exist on disk.

`COV002` (`warn`)

- A top-level directory under `code/` is not owned by any agent scope.

`COV003` (`warn`)

- Two different agents declare overlapping non-glob scope paths.

`COV004` (`info`)

- An agent declares scope paths but has an empty responsibilities description.

### staleness

`STALE001` (`error`)

- Rebuilding the graph in memory yields a different `content_hash` than the on-disk `context/graph.json`.

`STALE002` (`warn`)

- `context/semantic.json` exists and its `graph_hash` differs from the current graph `content_hash`.

`STALE003` (`warn`)

- `context/semantic.json` contains one or more concepts or edges with `reviewed: false`.

`STALE004` (`info`)

- `context/semantic.json` is absent.

### drift

`DRIFT001` (`error`)

- A scanned markdown file references a `code/...` path that does not exist on disk.

`DRIFT002` (`error`)

- A scanned markdown file references an identifier candidate that is not present in the known symbol set.

`DRIFT003` (`info`)

- An exported symbol exists in the known symbol set but is never mentioned by any scanned identifier candidate.
- Demoted from `warn` to `info` — not every exported symbol warrants documentation. Use COV005 to detect concepts with no implementing code instead.
- Review gate design for concept–code edges: see [`specs/decisions/0003-implements-edge-review-gate.md`](decisions/0003-implements-edge-review-gate.md).

## Out of scope for the shipped contract

The current implementation does not emit these codes:

- `DRIFT004`
- `DRIFT900`

They are not part of the current report contract and MUST NOT be expected from a clean V1 implementation.
