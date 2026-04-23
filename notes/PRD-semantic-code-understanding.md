# PRD: oz Semantic Code Understanding

**Status**: Draft  
**Date**: 2026-04-23

---

## 1. Summary

oz currently collects exported Go symbols but does not understand what they do. This feature adds a semantic layer that maps code packages to workspace concepts, replacing noisy symbol-level drift checks with meaningful coverage gaps. The result: you can ask "which code implements concept X?" and get a real answer.

---

## 2. Contacts

| Name | Role | Comment |
|------|------|---------|
| João Matos | Owner / Lead | Driving vision and implementation |
| LLM agents (oz-coding, oz-spec) | Implementors | Will use this to answer code questions |

---

## 3. Background

### What is this?

oz builds a graph of a workspace — agents, specs, docs, and code. Code is represented as a flat list of exported symbol names. That is all oz knows about code right now.

When `oz audit` runs, it fires a check called DRIFT003: "this exported symbol is not mentioned in any doc or spec." This warning is almost always noise. Not every function belongs in a spec. The check treats "mentioned in a document" as a stand-in for "understood," which is a bad proxy.

### Why now?

The enrichment pipeline (`oz context enrich`) already extracts abstract concepts from agents and specs using an LLM. The infrastructure exists. The missing piece is: the LLM never sees the code. So concepts like `workspace-convention` float in `semantic.json` with no grounding — no edges connecting them to the code that actually implements them.

The graph is structurally complete but semantically empty on the code side. That limits what agents and humans can ask of it.

### What changed?

The enrichment prompt and semantic overlay format are stable enough to extend. This is the right moment to close the loop: add code to the enrichment input and produce concept-to-implementation edges.

---

## 4. Objective

### Goal

Enable oz to understand what code does at the package level, and link that understanding to the workspace's concept graph — so agents and humans can navigate from concept to implementation and back.

### Why it matters

- LLM agents querying the graph today get structural answers ("these symbols exist") not semantic ones ("this is where X is implemented")
- Spec writers have no way to know if a spec concept is actually built
- Developers have no way to ask "what does this package do in the context of the project?"

### Key Results

| # | Result | Measure |
|---|--------|---------|
| KR1 | DRIFT003 false-positive rate drops to near zero | Manually review 20 warnings before/after; target <2 real positives in the noisy set |
| KR2 | Every package in `code/oz/internal/` has at least one concept edge | `oz context query` returns concept for any package path |
| KR3 | `oz audit` surfaces at least one true coverage gap (concept with no implementing code) in oz's own codebase | Confirmed by human review |
| KR4 | Query "show code for concept X" returns correct packages | Spot-check 5 concepts |

---

## 5. Market Segment

### Primary user

**Developers working in LLM-first workspaces** — specifically people who use oz to give LLMs structured context about a codebase. They want the LLM to answer questions like "how is audit implemented?" or "does any code back the spec for drift detection?" today, those questions return symbol lists, not understanding.

### Secondary user

**The oz LLM agents themselves** — oz-coding, oz-spec, oz-notes. They query the graph to do their work. Better concept-to-code edges means better answers with less hallucination.

### Constraints

- Go-only in V1 (language support is already scoped to Go)
- Requires an LLM call (OpenRouter) — enrichment is not free or instant
- Human review step (`reviewed: false` → `true`) must be preserved — oz does not auto-trust LLM output
- Must not break existing graph schema or query API

---

## 6. Value Proposition

### Jobs being done

1. "I want to understand what a part of the codebase does without reading every file"
2. "I want to know if my spec is actually implemented"
3. "I want to ask an LLM a question about the code and get a grounded answer, not a hallucination"

### Gains

- Navigate from concept → implementation in one query
- Spec coverage becomes verifiable, not assumed
- LLM agents get richer context, produce better output

### Pains eliminated

- DRIFT003 spam: hundreds of warnings that mean nothing
- "Concepts with no code home" go undetected today — after this, they surface in audit
- Developers manually grep for where a concept is implemented — after this, the graph knows

### Why better than the current state

The current enrichment only processes convention documents. It produces a concept graph that is disconnected from implementation. This feature closes that gap with zero new infrastructure — just an extended enrichment prompt and new edge types in the existing semantic overlay.

---

## 7. Solution

### 7.1 User flows

**Flow A: Enrich with code understanding**
```
$ oz context enrich
  → collects package summaries + symbol lists from code/
  → sends to LLM alongside existing agents/specs
  → LLM produces: package intent descriptions + concept edges
  → merged into semantic.json (reviewed: false)
  → user reviews with: oz context review
```

**Flow B: Query concept → code**
```
$ oz context query "how is drift detection implemented?"
  → query engine traverses concept:drift-detection → implements → code/oz/internal/audit/drift
  → returns package description + relevant symbols
```

**Flow C: Audit coverage gaps**
```
$ oz audit
  → new check COV005: concept node with no inbound "implements" edges → warn
  → new check COV006: package with no outbound "implements" edges → info
  → DRIFT003 demoted to info or removed
```

### 7.2 Key features

#### F1: Package-level intent extraction

Instead of indexing individual exported symbols for semantic purposes, extract a natural-language description of what each Go **package** does. Packages are the right semantic unit — a package has a coherent purpose; a single function usually doesn't.

Each package gets:
- A one-sentence `intent` field
- A list of the concepts it serves (from the existing concept graph)

This is stored in `semantic.json` as a new node type: `code_package`.

#### F2: Concept-implementation edges

New edge type: `implements`  
Direction: `code_package → concept`  
Produced by: LLM enrichment  
Stored in: `semantic.json`

Example:
```json
{ "from": "code:github.com/joaoajmatos/oz/internal/audit/drift", "to": "concept:drift-detection", "kind": "implements", "confidence": 0.92, "reviewed": false }
```

#### F3: Enrichment prompt extension

`oz context enrich` prompt gains two new input sections:

**Docs** — `docs/` content included alongside specs. Docs describe implementation detail that specs skip (architecture, how things are structured, open items). This raises concept graph density before the LLM ever sees code, which is the primary fix for sparse concept graphs.

**Code Packages** — For each package:
- Package import path
- Exported symbol names and kinds
- (optional) First comment block / doc comment from each file

The LLM is asked: "For each package, write one sentence describing its purpose. Then, for each package, list which concept IDs from the concept graph it implements."

**Source-of-truth note in prompt**: docs rank below specs in authority. When a concept derived from a doc conflicts with one from a spec, the spec version wins. The prompt must state this explicitly so the LLM doesn't blend them.

#### F4: Audit check replacements

| Old check | New check | Change |
|-----------|-----------|--------|
| DRIFT003: symbol not in docs (warn) | Demoted to `info` or removed | Was noise |
| — | COV005: concept has no `implements` edges (warn) | New, meaningful |
| — | COV006: package has no `implements` edges (info) | New, discovery aid |

#### F5: Query routing update

`oz context query` traverses `implements` edges when the query involves implementation or code. This is a scoring weight update in `scoring.toml`, not new code.

### 7.3 Technology

- No new infrastructure — extends existing enrichment pipeline (`code/oz/internal/enrich/`)
- New node type `code_package` added to graph schema (`code/oz/internal/graph/graph.go`)
- New edge kind `implements` added to schema
- `goindexer` extended to emit package-level nodes alongside symbol nodes
- Enrichment prompt in `code/oz/internal/enrich/prompt.go` extended with docs section and code section (in that order — docs first to seed concepts, then code to map them)
- Audit checks updated in `code/oz/cmd/audit.go` and `code/oz/internal/audit/`

### 7.4 Assumptions

| # | Assumption | Risk if wrong |
|---|-----------|--------------|
| A1 | LLM can reliably map Go packages to abstract concepts given the concept graph as input | Concept edges will be low-quality; enrichment becomes noise |
| A2 | Package-level granularity is the right unit (not file, not function) | Either too coarse (large packages) or too fine (many packages with same concept) |
| A3 | The concept graph (from spec + doc enrichment) has enough concepts to map to | Code packages map to concepts that don't exist yet; creates orphan edges. Docs are the partial fix; V2 concept discovery from code is the full fix. |
| A4 | Human review workflow (reviewed: false → true) is sufficient quality gate | Bad edges pollute queries and audit before review |
| A5 | Enrichment LLM calls stay within acceptable cost/latency | Users skip `oz context enrich` because it's too slow or expensive |

---

## 8. Release

### V1 — Concept-implementation edges (this sprint)

- Package-level node extraction from Go code
- Enrichment prompt extended with code packages
- `implements` edge type in semantic.json
- COV005 / COV006 audit checks
- DRIFT003 demoted to info

**Does not include**: multi-language support, automatic concept creation from code (only maps to existing concepts), UI/query changes beyond edge traversal weights.

### V2 — Concept discovery from code

- LLM can propose *new* concept nodes from code (not just map to existing ones)
- Cross-language package support (tree-sitter)
- Richer package descriptions (not just one sentence)

### V3 — Interactive coverage view

- `oz context show concept:X` renders a tree: concept → implementing packages → key symbols
- Integration with IDE / MCP for inline concept attribution

---

*Saved to `notes/PRD-semantic-code-understanding.md`*
