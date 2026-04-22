# ADR-0002: oz crystallize — LLM Primary Classifier, Heuristic Fallback

**Status**: accepted
**Date**: 2026-04-22

---

## Context

`oz crystallize` needs to classify notes from `notes/` into one of five artifact
types (`adr`, `spec`, `guide`, `arch`, `open-item`) before promoting them to their
canonical location. Two approaches were considered:

**Option A — Heuristic-only**: keyword + structure signal scoring with no external
dependencies. Deterministic, fast, offline.

**Option B — LLM primary, heuristic fallback**: use OpenRouter (already integrated
via `internal/openrouter/`) as the primary classifier with workspace-aware few-shot
context. Fall back to heuristic when no API key is present or the LLM call fails.

The original design used Option A with the LLM as opt-in (`--enrich`). This was
reconsidered because keyword matching fundamentally cannot handle informal or
idiomatic writing — a note saying "I think we should go with X for now, here's
why" will never match `"we decided"`, even though it is clearly a decision record.

The pre-mortem (T1) flagged classifier accuracy as the highest-risk assumption.

## Decision

**Use Option B: LLM primary, heuristic fallback.**

- OpenRouter is the primary classifier. The prompt includes the workspace's
  source-of-truth hierarchy, the type→target-path table, and few-shot examples
  drawn from the workspace's own existing artifacts (first 30 lines of one real
  ADR, spec, and guide). This grounds the LLM in *this* workspace's conventions.
- The heuristic engine (`internal/crystallize/classifier/heuristic/`) is the
  fallback for: no `OPENROUTER_API_KEY`, `--no-enrich` flag, or LLM call failure.
  It is never silently skipped — the fallback path is tested and logged.
- A file-hash-keyed cache (`.oz/crystallize-cache.json`) stores LLM results so
  repeated `--dry-run` calls are deterministic. Cache is invalidated automatically
  when the note content changes.
- `--no-enrich` replaces the former `--enrich` flag: LLM is default, opt-out.
- The confidence threshold required for `--accept-all` auto-acceptance applies to
  both paths (high = score ≥ 6 AND gap ≥ 4 for heuristic; `"confidence": "high"`
  from the LLM).

## Consequences

- LLM accuracy on well-written notes will be significantly higher than keyword
  matching. The 90% accuracy target on oz's own `notes/` is expected to be met
  comfortably; the 85% target on external corpus is the real validation gate
  (Sprint 4).
- `oz crystallize` requires `OPENROUTER_API_KEY` for optimal results. Teams
  running in CI without a key get heuristic classification with a clear warning.
- Non-determinism is resolved by the cache, not by the LLM itself. Cache misses
  (first run or changed notes) will call the LLM; subsequent runs with unchanged
  notes use the cache.
- Prompt quality is the new primary risk surface, not signal table completeness.
  Prompt changes must be accompanied by accuracy re-measurement against the
  ground-truth corpus.
