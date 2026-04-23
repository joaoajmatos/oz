# Routing Packet

This document normatively defines the routing packet returned by `oz context query` and by the MCP `query_graph` tool.

## Scope

- `oz context query <task>` returns the routing packet directly.
- MCP `query_graph` returns the same packet serialized as JSON text inside an MCP text content block.
- `oz context query --raw` returns a different debug envelope and is outside this contract.

## Routing packet object

The packet is a JSON object with these fields:

- `agent` (string): winning agent name. Omitted or empty when there is no clear owner.
- `confidence` (number): softmax confidence for the winning agent.
- `scope` (array of strings, optional): the winning agent's declared scope paths.
- `context_blocks` (array of context blocks, optional)
- `relevant_concepts` (array of strings, optional): concept names owned by the winning agent when a semantic overlay exists.
- `excluded` (array of strings, optional): path prefixes filtered from context selection. In the shipped default flow this is `["notes/"]` when notes exist and notes are excluded.
- `reason` (string, optional): currently `"no_clear_owner"` when no winner is returned.
- `candidate_agents` (array of candidate-agent objects, optional): populated only for ambiguous but still routed results.

## Context block object

Each `context_blocks` element MUST include:

- `file` (workspace-relative path)
- `section` (string; empty string is allowed for file-level nodes)
- `trust` (`"high" | "medium" | "low"`)

The shipped packet does not inline file contents.

Trust values are derived from graph tier:

- `"high"` -> `specs`
- `"medium"` -> `docs` and `context`
- `"low"` -> `notes`

Context blocks are sorted by:

1. trust rank (`high`, then `medium`, then `low`)
2. `file`
3. `section`

## Candidate agent object

Each `candidate_agents` element MUST include:

- `name` (string)
- `confidence` (number)

When present, candidates are sorted by descending `confidence`.

## Selection semantics

The shipped implementation builds the packet as follows:

- The BM25F corpus is agent-only; code nodes do not participate in routing.
- `scope` is the full declared scope of the winning agent, not a query-specific subset.
- `context_blocks` always include all spec sections and decisions.
- `context_blocks` additionally include docs and context snapshots reached from the winning agent's `reads` edges.
- Notes are included only when `include_notes` is enabled.

## no_clear_owner

The result MUST be treated as `no_clear_owner` when any of these conditions applies:

- the query tokenizes to zero terms
- there are zero agent documents to score
- the top raw BM25F score is below `routing.min_score`
- the graph cannot be loaded and an in-memory rebuild also fails

In this case the packet uses:

- `reason: "no_clear_owner"`
- no `candidate_agents`

## Ambiguity threshold

An ambiguous-but-routed result occurs when:

- a top agent exists, and
- its confidence is below `routing.confidence_threshold`

In that case `candidate_agents` MUST include every agent whose confidence is at least `routing.min_candidate_confidence`.

## Scoring config contract

If `context/scoring.toml` exists, the query engine reads these sections and keys:

`[bm25]`

- `k1` (default `1.2`)

`[fields]`

- `b_text` (default `0.75`)
- `b_path` (default `0.5`)

`[weights]`

- `scope` (default `4.0`)
- `role` (default `2.5`)
- `responsibilities` (default `2.5`)
- `readchain` (default `0.0`)
- `out_of_scope_penalty` (default `2.5`)

`[routing]`

- `confidence_threshold` (default `0.7`)
- `min_score` (default `0.01`)
- `temperature` (default `0.2`)
- `min_candidate_confidence` (default `0.2`)
- `include_notes` (default `false`)

`[tokenize]`

- `use_bigrams` (default `false`)

Fallback behavior:

- If `context/scoring.toml` is absent, built-in defaults are used.
- Unknown keys are ignored by the query loader (use `oz context scoring validate` to catch typos and unknown tables before running queries).
- If the file is present but not valid TOML, the loader keeps built-in defaults for the whole file. Valid TOML with out-of-range values is rejected by `oz context scoring set` and `validate` where those checks apply.

**Preferred editing path:** `oz context scoring` (`list`, `describe`, `get`, `set`, `show`, `validate`) — see `docs/implementation.md`.
