# Pre-Mortem: `oz context concept add` (retrieval-augmented, routing-free)

**Assumed launch:** we ship the LLM-assisted “add one concept” flow (CLI first) with retrieval-only context and no agent routing, then merge unreviewed items into `context/semantic.json` for human `oz context review`.

**Failure scenario (imagined):** Three months later, maintainers distrust the command. People still hand-edit `semantic.json` or run full `enrich` because proposals are noisy, retrieval context does not match what `context query` would show, or the feature breaks in edge workspaces. Optional `OPENROUTER_API_KEY` and cost surprise new contributors.

---

## Tigers (real risks)

Risks that could actually derail adoption or correctness; each includes suggested urgency.

| Risk | Why it bites | Urgency |
|------|----------------|--------|
| **T1. Retrieval-only path diverges from `oz context query` behavior** | We add a second way to score/survive blocks (no agent, neutral `BuildContextBlocks`, different survivors). Users expect “what the graph would surface for this text” and get a different ordering or code/spec mix. Trust in dogfooding collapses. | **Launch-blocking** for the “retrieval-augmented” story—need explicit contract tests or shared core. |
| **T2. `BuildContextBlocks` / survivor logic with “no agent” is wrong or unstable** | `ensureScopeSurvivor` / `ensureCodePackageSurvivor` assume a routed agent today. Empty or sentinel agent may surface the wrong `code/` file, omit specs, or thrash golden tests. | **Launch-blocking**—ship only with tests that lock behavior for proposal mode. |
| **T3. Prompt is too large or too weak** | Full node-ID allowlist + retrieval slice + filtered `promptGraph` may exceed practical context; conversely, a small allowlist loses valid `To` targets and the model invents IDs (then parse fails repeatedly). Users see flaky runs. | **Launch-blocking**—cap catalog intelligently (e.g. only types the model may edge to) and measure token budget. |
| **T4. Parse/validate rejects good outputs or accepts bad ones** | Reusing `ParseResponse` for a *single* concept may not match the stricter prompt; multi-concept or markdown-wrapped JSON causes confusing errors. Edge `To` typos slip through if validation gaps exist. | **Fast-follow**—harden errors and golden tests on parser. |
| **T5. No duplicate / near-duplicate detection (v1)** | Repeated `concept add` for the same idea with different slugs clutters the overlay and worsens retrieval. | **Track** (v1 OK) but document; **fast-follow** if early adopters hit it. |
| **T6. Review step is skipped in automation** | CI or agents run merge and `--accept-all` on review without reading; bad edges get `reviewed: true` and pollute query forever. | **Fast-follow**—docs and guardrails; optional “no write without TTY” for review. |

---

## Paper Tigers (overblown concerns)

| Concern | Why it is likely overblown here |
|--------|-----------------------------------|
| **“Another subcommand clutters the CLI”** | The surface is one clear verb (`context concept add`); discoverability beats hiding behind `enrich` flags. |
| **“OpenRouter dependency is new risk”** | Full `enrich` already requires the same key; this feature does not change the security model—only call frequency. |
| **“Users will confuse this with `enrich`”** | Solvable with one paragraph in `semantic-overlay.md` and `context concept add --help` copy. |

---

## Elephants (unspoken or under-discussed)

| Topic | Why it matters | Suggested investigation |
|-------|----------------|-------------------------|
| **E1. Semantic authority** | If proposals are easy, **who** is allowed to land concepts in team workflows—any branch, or maintainer-only? | Decide policy in AGENTS/MAINTAINERS or docs before wide internal launch. |
| **E2. Two sources of “truth” for retrieval** | `relevant_concepts` in the prompt might reinforce existing mistakes; the model might over-anchor to them. | Prompt text should say: existing concepts are *hints*, not requirements; new edges must still validate. |
| **E3. Agentic misuse** | An automated agent could spam `concept add` in a loop. | Rate limits, idempotency keys, or “dry run default” in CI—track for MCP follow-up. |

---

## Action plans (launch-blocking tigers only)

| ID | Risk | Mitigation (concrete) | Owner | By when |
|----|------|------------------------|-------|--------|
| T1 | Divergence from `context query` | Add contract tests: same query string → compare ranked **file+section** lists between proposal retrieval and a documented “reference” mode, or assert shared helper usage in both paths. Document any intentional difference in spec. | Eng | Before merge to main |
| T2 | No-agent survivors | **Before ship:** integration tests on real fixture workspaces for proposal mode; list expected top block files. Refactor so “proposal mode” is one code path, not a flag soup. | Eng | Before ship |
| T3 | Token / catalog size | Prototype max prompt with largest workspace; trim node catalog to edge-relevant ID prefixes; enforce `--retrieval-k` default that matches scoring.toml spirit. | Eng | Before first internal dogfood |

**Fast-follow (30 days):** T4 (parser UX), T6 (automation + review story in docs).

**Track:** T5 (duplicate heuristics), E1–E3 (governance and abuse).

---

## Revisit

Re-run this pre-mortem after the **retrieval-only** API exists and after **one** full dry-run on the oz repo (propose a throwaway concept, review, reject).

*Written from plan: add concept workflow (retrieval-only, no agent routing). 2026-04-25.*
