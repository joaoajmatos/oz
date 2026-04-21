# Graph Schema

This document normatively defines `context/graph.json`, the structural graph produced by `oz context build`.

## Top-level object

`context/graph.json` MUST be a JSON object with these fields:

- `schema_version` (string): the current shipped value is `"2"`.
- `content_hash` (string): lowercase hex SHA-256 of the canonical `nodes` + `edges` payload.
- `nodes` (array of node objects)
- `edges` (array of edge objects)

`schema_version` and `content_hash` MUST always be present.

## Node object

Every node MUST include:

- `id` (string)
- `type` (string)
- `file` (string, workspace-relative path)
- `name` (string)

Optional fields are:

- `tier` (`"specs" | "docs" | "context" | "notes"`)
- `section` (string)
- `language` (string)
- `symbol_kind` (`"func" | "type" | "value"`)
- `package` (string)
- `line` (positive integer)
- `role` (string)
- `scope` (array of strings)
- `responsibilities` (string)
- `out_of_scope` (string)
- `read_chain` (array of strings)
- `rules` (array of strings)
- `skills` (array of strings)
- `context_topics` (array of strings)

### Node types

`agent`

- ID format: `agent:<name>`
- Required extra semantics: `name` is the agent name.
- Optional agent-only fields: `role`, `scope`, `responsibilities`, `out_of_scope`, `read_chain`, `rules`, `skills`, `context_topics`.
- `tier` and `section` are omitted.

`spec_section`

- ID format: `spec_section:<file>:<heading>`
- Produced from `specs/*.md` headings.
- `tier` MUST be `"specs"`.
- `section` is the heading text when a heading-backed node exists; it MAY be omitted for file-level fallback nodes.

`decision`

- ID format: `decision:<basename-without-.md>`
- Produced from `specs/decisions/*.md`.
- `tier` MUST be `"specs"`.

`doc`

- ID format: `doc:<file>:<heading>`
- Produced from `docs/*.md` headings.
- `tier` MUST be `"docs"`.
- `section` is the heading text when a heading-backed node exists; it MAY be omitted for file-level fallback nodes.

`context_snapshot`

- ID format: `context_snapshot:<file>`
- Produced from files under `context/`.
- `tier` MUST be `"context"`.

`note`

- ID format: `note:<file>`
- Produced from files under `notes/`.
- `tier` MUST be `"notes"`.

`code_file`

- ID format: `code_file:<file>`
- Produced from source files under `code/`.
- `language` is currently `"go"` for shipped indexed files.
- `package` is the resolved Go import path for the file.
- `tier` and `section` are omitted.

`code_symbol`

- ID format: `code_symbol:<package>.<name>`
- Produced for exported Go declarations.
- `language` MUST be `"go"` in the current implementation.
- `symbol_kind` MUST be one of `"func"`, `"type"`, or `"value"`.
- `package` is the resolved Go import path.
- `line` is the 1-based declaration line.
- `tier` and `section` are omitted.

## Edge object

Every edge MUST include:

- `from` (string): source node ID
- `to` (string): target node ID
- `type` (string)

### Edge types

`reads`

- Direction: `agent -> referenced node`
- Meaning: an agent read-chain entry resolved to a graph node.

`owns`

- Direction: `agent -> owned target`
- Meaning: an agent scope entry.
- The target MAY be a real node ID or a synthetic `path:<scope>` pseudo-ID when no graph node resolves the scope path.

`references`

- Direction: `source node -> referenced node`
- Meaning: a markdown file mentioned another file path and the reference resolved to a graph node.

`supports`

- Direction: `doc node -> spec or decision node`
- Meaning: a document references spec-tier material and therefore supports it.

`crystallized_from`

- Reserved edge type in the schema.
- The current shipped builder does not emit these edges.

`contains`

- Direction: `code_file -> code_symbol`
- Meaning: the source file declares the exported symbol.

## Invariants

- Serialization MUST sort `nodes` by `id`.
- Serialization MUST sort `edges` by `(from, to, type)`.
- `content_hash` MUST be computed from the canonical JSON encoding of `{nodes, edges}` only; it MUST NOT include `content_hash` itself.
- Re-running normalization and serialization on unchanged input MUST produce byte-identical output.
- The graph MAY contain multiple nodes for the same file; file-backed reference resolution uses the lexicographically first node ID for that file.
