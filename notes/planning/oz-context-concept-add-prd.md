# Product Requirements Document: `oz context concept add` (V1)

**Author**: oz-spec (draft; align with oz-coding for implementation)  
**Date**: 2026-04-25  
**Status**: Draft  
**Stakeholders**: oz-coding, oz-maintainer, oz-spec  
**Links**: [Sprint plan](oz-context-concept-add-sprints.md) · [Pre-mortem](oz-context-concept-add-premortem.md)

---

## 1. Summary

V1 adds a **focused CLI path** to propose **one** new semantic concept (and candidate edges) into `context/semantic.json`, using an LLM (OpenRouter, same as `oz context enrich`) and **retrieval-grounded** context from the existing query stack. The command does **not** replace full `enrich`; it gives maintainers a way to add a single concept without re-running a whole-graph extraction. Proposals are **unreviewed** until `oz context review` (or an explicit accept path). V1 is **CLI-first**; an MCP tool is a follow-up.

---

## 2. Contacts

| Role | Who | Notes |
|------|-----|--------|
| Product / spec | oz-spec | PRD, `specs/semantic-overlay.md` updates, ADR if needed |
| Implementation | oz-coding | `internal/query`, `internal/enrich`, `cmd` |
| Conventions / graph health | oz-maintainer | AGENTS, validate, docs that describe workflow |

---

## 3. Background

Today, `context/semantic.json` is populated by **`oz context enrich`** (LLM over the **full** structural graph) and by **manual JSON edits**. Enrich is the right tool for broad extraction but is heavy for “I want one new reviewed concept with sensible `implements` edges.” Manual editing is error-prone (invalid `To` node IDs, merge mistakes).

**Why now:** the query engine already ranks **context blocks**, **relevant_concepts**, and **implementing_packages** for a text query. Reusing that retrieval for a **single-concept proposal** dogfoods oz and grounds the model in the same chunks a user would get from `context query`—**without** sending the entire graph prompt every time.

**Pre-mortem:** [oz-context-concept-add-premortem.md](oz-context-concept-add-premortem.md) lists launch-blocking risks (retrieval-only must not diverge in surprising ways; no-agent survivors must be tested; prompt size). Those mitigations are **in scope** for V1 ship criteria.

---

## 4. Objective

### Problem

Adding one well-formed concept to the semantic overlay today requires either a full enrich run or hand-editing JSON with knowledge of graph node IDs and merge rules.

### Outcome

A maintainer can run a command (working name: **`oz context concept add`**) with at least a **name** and optional **seed text** and **file anchors**, receive a **single** proposed concept and edges, validate structurally, merge as **unreviewed**, and run the existing **review** flow to accept or reject.

### Company / product benefit

- Faster, safer evolution of the semantic layer for the oz workspace and downstream **routing + retrieval** quality.
- Clearer story: “enrich = broad extract; concept add = one focused proposal.”

### Key results (measurable)

| Key result | Target (V1) |
|------------|-------------|
| Structural validity | 100% of written edges pass existing `ParseResponse` / node-ID validation (no invalid `To`) |
| Review gate | New items have `reviewed: false` until `oz context review` (or documented accept flag) |
| Retrieval contract | Pre-mortem T1–T2 mitigations: contract or integration tests for **retrieval-only** path (no agent routing) |
| Operability | `oz validate` and `go test` green; docs in `specs/semantic-overlay.md` + one user-facing blurb |
| Dogfood | At least one successful dry-run on this repo (propose throwaway → reject in review) |

---

## 5. Market segment(s)

**Primary: oz core maintainers and power users**  
People who already run `context build`, `enrich`, and `review` and need to add or adjust concepts without a full enrich.

**Secondary: LLM / agent workflows**  
Agents that call the CLI (and later MCP) to propose concepts from a task description; V1 may only document CLI, not automate review.

**Out of scope for “segment” definition:** end users who never touch `context/`—they are unaffected.

---

## 6. Value proposition

| Job to be done | Today | With V1 |
|----------------|--------|--------|
| Add one concept with valid graph links | Manual JSON or hope enrich infers it | One command + review |
| Ground the LLM in “what this workspace says” | Full graph prompt or none | **Retrieval slice** (blocks + concepts + packages) from the same engine as `context query` |
| Stay consistent with product behavior | Risk of a one-off prompt | **Retrieval-only, no agent routing** (per plan) so we do not pretend a single “winning agent” for this use case |
| Trust | Opaque | **Explicit** `--print` / dry path optional; unreviewed until review |

**Competition:** “use ChatGPT to write JSON” — we win on **validated IDs**, **merge contract**, and **same retrieval policy** as the rest of oz.

---

## 7. Solution

### 7.1 User experience (CLI)

- **Command:** `oz context concept add` (alias `propose` optional) under `oz context`.
- **Inputs (minimum):** `--name` (string).  
- **Optional:** `--seed` (free text), `--from` (repeatable paths), `--retrieval-k`, `--no-retrieval`, `--print` (no write), JSON/stdin for automation (exact shape in implementation spec).
- **Preconditions:** fresh enough `context/graph.json` (build if needed); warn or fail on semantic staleness per existing contracts.
- **Flow:** (1) optional retrieval-only bundle, (2) build proposal prompt with filtered structural catalog + allowlisted `To` IDs, (3) OpenRouter, (4) parse/validate single concept, (5) merge + write with `reviewed: false`, (6) user runs `oz context review`.

**Important:** **Skip agent routing** for this flow. Use a dedicated **retrieval-only** API (see [pre-mortem T1–T2](oz-context-concept-add-premortem.md)).

### 7.2 Key features (V1)

1. **Retrieval-only query slice** — Expose an internal API that returns ranked blocks, `relevant_concepts`, `implementing_packages`, and optionally code entry points **without** choosing a winning agent or agent-scoped survivors.
2. **Propose-concept prompt** — Single-concept JSON contract; instruction that `To` must be from supplied catalog; prompt labels retrieval as *context*, not authority for new IDs.
3. **Parse and merge** — Reuse or extend `enrich` parser; reject multi-concept responses with clear errors.
4. **Flags and safety** — Token cap for prompt; document `OPENROUTER_API_KEY` requirement (same as enrich).

### 7.3 Technology

- **Go** packages: `internal/query` (retrieval entry), `internal/enrich` (prompt + call + merge), `cmd` (Cobra).  
- **Dependency direction:** `enrich` may call `query`; avoid cycles (`query` must not import `enrich`).

### 7.4 Assumptions (to validate)

- OpenRouter availability and cost are acceptable for occasional use (not per keystroke).  
- Single-concept JSON is reliably parseable with existing guardrails.  
- Maintainers will run **review** before treating concepts as production-ready.

### 7.5 Non-goals (V1)

- Automatic near-duplicate detection against existing concept names (track for V2).  
- MCP tool (V2) unless V1 ships early.  
- Editing or deleting existing reviewed concepts via this command.

---

## 8. Release

**Relative sizing (solo or small team):** about **2 sprints** of focused work (see [oz-context-concept-add-sprints.md](oz-context-concept-add-sprints.md)): first sprint lock spec + land retrieval API + tests; second sprint wire enrich + CLI + docs + dogfood.

**V1 scope:** CLI + tests + `semantic-overlay` doc updates (+ optional small ADR).

**V2+:** MCP `propose_concept`, duplicate warnings, stricter automation guardrails for review.

---

## Open questions

1. Exact flag names and whether `--accept` on the command is allowed in V1 or only via `review`.  
2. Whether proposal mode uses a **fixed pseudo-agent** vs **empty agent** for `BuildContextBlocks`—decided in implementation with tests (pre-mortem T2).  
3. **Governance (elephant E1):** document who may merge reviewed concepts in shared repos (out of V1 code scope; doc in `docs/` or `AGENTS.md`).

---

*End of PRD V1 draft.*
