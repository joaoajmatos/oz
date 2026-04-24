# Semantic Overlay

This document normatively defines `context/semantic.json`, the optional semantic overlay produced by `oz context enrich`.

## Top-level object

`context/semantic.json` MUST be a JSON object with these fields:

- `schema_version` (string): the current shipped value is `"1"`.
- `graph_hash` (string): the `content_hash` of the `context/graph.json` used to produce the overlay.
- `model` (string, optional): the OpenRouter model ID used for enrichment.
- `generated_at` (string, optional): RFC3339 timestamp for the enrichment run.
- `concepts` (array of concept objects)
- `edges` (array of concept-edge objects)

## Concept object

Every concept object MUST include:

- `id` (string): stable key of the form `concept:<slug>`
- `name` (string)
- `tag` (`"EXTRACTED" | "INFERRED"`)
- `confidence` (number)
- `reviewed` (boolean)

Optional concept fields are:

- `description` (string)
- `source_files` (array of workspace-relative paths)

Interpretation:

- `EXTRACTED` means the model reported a directly observed concept.
- `INFERRED` means the model reported a reasonable inference.
- `confidence` is emitted for both tags; the implementation expects values in the `0.0` to `1.0` range.
- `reviewed` tracks human acceptance state for that exact concept object.

## Edge object

Every semantic edge MUST include:

- `from` (string): currently expected to be a concept ID
- `to` (string): a concept ID or a structural graph node ID
- `type` (string)
- `tag` (`"EXTRACTED" | "INFERRED"`)
- `confidence` (number)
- `reviewed` (boolean)

The shipped edge types are:

- `agent_owns_concept`
- `implements_spec`
- `drifted_from`
- `semantically_similar_to`
- `implements` — a concept is implemented by a code package (direction: `concept:<slug>` → `code_package:<import-path>`)

## Staleness contract

- If `graph_hash` differs from the current `context/graph.json` `content_hash`, the overlay is stale.
- `oz context query` and `oz context serve` may still load the overlay, but `oz context serve` warns that it may be stale.
- `oz audit staleness` emits `STALE002` when this mismatch exists.

## Merge contract

`oz context enrich` merges a newly generated overlay with an existing one using these rules:

- Incoming `schema_version`, `graph_hash`, `model`, and `generated_at` replace existing metadata.
- All incoming concepts and edges are included as-is.
- Reviewed concepts from the existing overlay are appended only when their `id` does not appear in the incoming overlay.
- Reviewed edges from the existing overlay are appended only when the tuple `(from, type, to)` does not appear in the incoming overlay.

This means the current shipped merge preserves reviewed items that disappear from a new run; identical incoming IDs or edge keys replace older entries.
