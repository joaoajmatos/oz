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
- `context_blocks` (array of context blocks, optional): ranked by query relevance.
- `code_entry_points` (array of code-entry-point objects, optional): top-ranked `code_symbol` references for code-level queries.
- `relevant_concepts` (array of strings, optional): concept names owned by the winning agent when a semantic overlay exists.
- `implementing_packages` (array of strings, optional): import paths of `code_package` nodes that implement query-relevant concepts via reviewed `implements` edges, ranked and capped.
- `excluded` (array of strings, optional): path prefixes hard-excluded from the retrieval corpus. Empty under shipped defaults (`retrieval.include_notes = true`); contains `"notes/"` when `retrieval.include_notes = false` and notes exist.
- `reason` (string, optional): `"no_clear_owner"` when no winner is returned; `"no_relevant_context"` when a winner is returned but no retrieval block cleared `retrieval.min_relevance` (and `context_blocks` is then omitted or empty). Other protocol reasons (e.g. list-mode overviews) may be added as needed.
- `candidate_agents` (array of candidate-agent objects, optional): populated only for ambiguous but still routed results.

## Context block object

Each `context_blocks` element MUST include:

- `file` (workspace-relative path)
- `section` (string; empty string is allowed for file-level nodes)
- `trust` (`"high" | "medium" | "low"`)
- `relevance` (number): the block's query-relevance score, produced by the
  retrieval scorer. Always ≥ `retrieval.min_relevance` when present.

The shipped packet does not inline file contents.

Trust values are derived from graph tier:

- `"high"` -> `specs`
- `"medium"` -> `docs`, `context`, and `code` (synthetic tier for
  `code_package` and `code_symbol` nodes that enter the retrieval corpus)
- `"low"` -> `notes`

Context blocks are sorted by:

1. `relevance` descending
2. trust rank (`high`, then `medium`, then `low`)
3. `file`
4. `section`

## Code entry point object

Each `code_entry_points` element MUST include:

- `file` (workspace-relative path)
- `symbol` (string; symbol name)
- `kind` (string; mirrors `code_symbol.symbol_kind`, e.g. `func`, `type`,
  `method`)
- `line` (number; 1-based line of the symbol)
- `package` (string; Go import path of the containing `code_package`)
- `relevance` (number)

Code entry points are sorted by `relevance` descending. They are capped at
`retrieval.max_code_entry_points`.

## Candidate agent object

Each `candidate_agents` element MUST include:

- `name` (string)
- `confidence` (number)

When present, candidates are sorted by descending `confidence`.

## Selection semantics

Routing and retrieval are separate pipelines with separate corpora. See
[`specs/decisions/0004-context-retrieval-ranking.md`](decisions/0004-context-retrieval-ranking.md)
for the design rationale.

### Routing (unchanged)

- The BM25F **routing** corpus is agent-only; code nodes and context blocks
  do not participate in agent selection.
- `scope` is the full declared scope of the winning agent, not a
  query-specific subset.

### Retrieval

The retrieval corpus includes these node types: `spec_section`, `decision`,
`doc`, `context_snapshot`, `note`, `code_package`, `code_symbol`.

Each candidate block is scored:

```
score(block) =
    BM25(query_terms, block_fields)
  * trust_boost(block.tier)
  * agent_affinity(block, winning_agent)
```

- `BM25` uses the same `normTF`/IDF form as agent routing, parameterised by
  `[retrieval.fields]` weights and `[retrieval.bm25]`.
- `trust_boost` is `retrieval.trust_boost_<tier>` (see scoring config).
- `agent_affinity` is `retrieval.agent_affinity_boost` (default `1.2`)
  when the block is connected to the winning agent via a `reads`, `owns`,
  or `agent_owns_concept → implements` chain; otherwise `1.0`.

Selection then:

1. Drops blocks with `score < retrieval.min_relevance`.
2. Sorts surviving blocks by `(relevance DESC, trust, file, section)`.
3. Keeps the first `retrieval.max_blocks` entries, with the constraint
   that if any block under each declared `scope` path cleared the
   threshold, at least one such block MUST survive truncation.

### `code_entry_points`

Populated from `code_symbol` blocks in the scored corpus. A symbol is
eligible when **either**:

- its `file` is under a path in the winning agent's `scope`, **or**
- its containing `code_package` has a reviewed `implements` edge to a
  concept whose retrieval score is at least
  `retrieval.concept_min_relevance`.

Eligible symbols are ranked by block `relevance` and truncated to
`retrieval.max_code_entry_points`.

### `implementing_packages`

1. Score each `concept` against the query using the retrieval BM25 with
   `[retrieval.concepts]` field weights.
2. Keep concepts with `score >= retrieval.concept_min_relevance`.
3. Walk reviewed `implements` edges from those concepts to `code_package`
   nodes.
4. Sort packages by the maximum concept score reaching them; truncate to
   `retrieval.max_implementing_packages`.

### Notes

Notes participate in retrieval like any other tier and are downweighted by
`retrieval.trust_boost_notes` (default `0.6`). The `[retrieval].include_notes`
flag hard-excludes `note` nodes from the corpus when set to `false`;
otherwise they are eligible and surface only when they outscore
higher-trust material. When hard-excluded, `excluded` MUST include the
`"notes/"` prefix.

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

`[retrieval]`

- `max_blocks` (default `12`): maximum `context_blocks` entries after
  ranking and truncation.
- `min_relevance` (default `0.05`): blocks with a final score below this
  are dropped.
- `max_code_entry_points` (default `5`): maximum `code_entry_points`
  entries.
- `max_implementing_packages` (default `5`): maximum
  `implementing_packages` entries.
- `concept_min_relevance` (default `0.1`): minimum concept retrieval score
  required for a concept to contribute to `implementing_packages` or to
  promote code symbols via the `implements` path.
- `agent_affinity_boost` (default `1.2`): multiplicative factor when a
  block is connected to the winning agent.
- `trust_boost_specs` (default `1.3`)
- `trust_boost_docs` (default `1.0`)
- `trust_boost_context` (default `1.0`)
- `trust_boost_code` (default `0.9`)
- `trust_boost_notes` (default `0.6`)
- `include_notes` (default `true`): when `false`, `note` nodes are
  excluded from the retrieval corpus and `excluded` in the packet
  includes `"notes/"`.

`[retrieval.bm25]`

- `k1` (default `1.2`)

`[retrieval.fields]`

Per-field length-normalisation factors (`b`) and weights for block scoring:

- `b_text` (default `0.75`)
- `b_path` (default `0.5`)
- `weight_title` (default `3.0`): applies to `spec_section.section`,
  `decision.title`, `doc.section`, `code_symbol.name`,
  `code_package.name`.
- `weight_body` (default `1.0`): applies to loaded body tokens for
  `spec_section`, `decision`, `doc`, `context_snapshot`, `note`, and to
  `code_package.doc_comment`.
- `weight_path` (default `0.5`): applies to `file` and to
  `code_symbol.package`.
- `weight_kind` (default `0.3`): applies to `code_symbol.symbol_kind`.

`[retrieval.concepts]`

Field weights for scoring concepts against the query in the
`implementing_packages` path:

- `weight_name` (default `3.0`)
- `weight_description` (default `1.0`)

`[tokenize]`

- `use_bigrams` (default `false`)

Fallback behavior:

- If `context/scoring.toml` is absent, built-in defaults are used.
- Unknown keys are ignored by the query loader (use `oz context scoring validate` to catch typos and unknown tables before running queries).
- If the file is present but not valid TOML, the loader keeps built-in defaults for the whole file. Valid TOML with out-of-range values is rejected by `oz context scoring set` and `validate` where those checks apply.

**Preferred editing path:** `oz context scoring` (`list`, `describe`, `get`, `set`, `show`, `validate`) — see `docs/implementation.md`.
