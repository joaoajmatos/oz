# ADR-0003: Implements Edge Review Gate

Date: 2026-04-23  
Status: Accepted

## Context

The enrichment pipeline (`oz context enrich`) produces `implements` edges — links from `code_package` nodes to `concept` nodes — with a `confidence` score and a `reviewed: false` flag. These edges must not affect `oz context query` results until they are trustworthy, because a wrong concept-to-code mapping would silently corrupt query output.

The original design required human review of every edge before it became active. In practice, on a solo project, "review later" means never. A blanket review requirement would mean the semantic layer is permanently inactive.

## Decision

Edges with `confidence ≥ 0.85` are automatically marked `reviewed: true` at write time.  
Edges below that threshold are written with `reviewed: false` and require explicit approval via `oz context review`.

The threshold of 0.85 is the initial value. It can be adjusted in `scoring.toml` without a code change.

`oz context query` only traverses edges with `reviewed: true`.  
`oz audit` (COV005) sees all edges regardless of review status.  
`oz context review` lists unreviewed edges with enough context to approve or reject quickly.

## Consequences

**Positive**
- High-confidence mappings are immediately useful without manual overhead
- The quality gate still exists for uncertain mappings
- Threshold is configurable without a code change

**Negative**
- A confident-but-wrong edge auto-promotes and silently affects queries until caught
- The 0.85 threshold is arbitrary until calibrated against real enrichment output — it may need tuning after the first real runs

## Alternatives considered

**All edges require human review**  
Cleaner in theory. Rejected because it depends on a review UX that doesn't exist yet and would leave the semantic layer permanently inactive on a solo project.

**No review gate — all edges active immediately**  
Rejected because LLM output is not reliable enough to trust unconditionally, and a wrong `implements` edge would corrupt query results with no audit trail.
