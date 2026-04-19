# Architecture

> High-level architecture of oz.

## Overview

**oz** is an open workspace convention and toolset for LLM-first development. It ships as a
single Go binary (`oz`) that scaffolds, validates, and indexes oz-compliant workspaces so any
LLM can understand them without custom integrations.

The oz repository is itself an oz workspace — every structural claim below can be verified by
running `oz context build && oz context query` in this repo.

---

## Three-layer architecture

```
┌──────────────────────────────────────────┐
│  Layer 1: Convention                      │
│  AGENTS.md · OZ.md · agents/*/AGENT.md   │
│  specs/ · docs/ · context/ · notes/      │
│  — Human-readable markdown convention     │
├──────────────────────────────────────────┤
│  Layer 2: Structural graph               │
│  context/graph.json                      │
│  — Deterministic machine-readable index   │
│    built by `oz context build`            │
├──────────────────────────────────────────┤
│  Layer 3: Semantic overlay               │
│  context/semantic.json                   │
│  — LLM-extracted concepts + typed edges  │
│    produced by `oz context enrich`        │
│    human-reviewed via `oz context review` │
└──────────────────────────────────────────┘
```

Layer 1 is always the source of truth. Layers 2 and 3 are derived artefacts; they can be
rebuilt at any time from the workspace convention files.

---

## Components

### oz binary (`code/oz/`)

Single Go binary built with `go build`. No runtime dependencies. Subcommands:

| Command | Package | Purpose |
|---------|---------|---------|
| `oz init` | `cmd/init.go` + `internal/scaffold/` | Scaffold a new oz-compliant workspace |
| `oz validate` | `cmd/validate.go` + `internal/validate/` | Lint workspace against the convention |
| `oz audit` | `cmd/audit.go` | Detect spec-code drift (stub: loads `graph.json`) |
| `oz context build` | `internal/context/` | Build the structural graph |
| `oz context query` | `internal/query/` | Route a task to the best-matching agent |
| `oz context enrich` | `internal/enrich/` | LLM enrichment pass via OpenRouter |
| `oz context review` | `internal/review/` | Human review of semantic overlay |
| `oz context serve` | `internal/mcp/` | MCP stdio server for LLM tool calls |

### internal packages

| Package | Responsibility |
|---------|---------------|
| `internal/convention` | Go-typed source of truth for the oz workspace standard |
| `internal/workspace` | Workspace discovery (walks up from `cwd` to find AGENTS.md + OZ.md) |
| `internal/scaffold` | Scaffolding templates (embedded via `//go:embed all:templates`) |
| `internal/validate` | Validation rules: required files, required AGENT.md sections, semantic warnings |
| `internal/graph` | `Graph`, `Node`, and `Edge` types; schema version constant |
| `internal/context` | Walker → parsers → indexer → extractor → serializer |
| `internal/query` | Tokenizer → BM25F scorer → softmax → context block builder |
| `internal/enrich` | Prompt builder → OpenRouter client → response parser → overlay writer |
| `internal/review` | Diff view + interactive accept/reject → semantic.json writer |
| `internal/mcp` | MCP stdio server: JSON-RPC 2.0 framing, four tool implementations |
| `internal/semantic` | `Overlay` schema, load/write helpers, staleness check |
| `internal/openrouter` | Isolated HTTP client for OpenRouter API |
| `internal/testws` | Test workspace builder + YAML fixture loader + golden suite runner |

---

## Data flow

### oz context build

```
workspace root
  └─ Walker (internal/context/walker.go)
       — discovers all oz-convention files
       — respects .ozignore
  └─ Parsers
       ParseAgentMD  → agent nodes (role, scope, read-chain, rules, skills)
       IndexMarkdownFile → spec_section, decision, doc, context_snapshot, note nodes
  └─ Cross-reference extractor (internal/context/extractor.go)
       — reads, owns, references, supports edges
  └─ Deterministic serializer (internal/context/serializer.go)
       — sorts nodes and edges by stable key
       — computes SHA-256 content hash
       — writes context/graph.json
```

### oz context query

```
query text
  └─ LoadGraph (or Build if graph.json absent)
  └─ LoadConfig (context/scoring.toml or defaults)
  └─ TokenizeQuery
       lowercase → strip punctuation → split → filter stopwords → stem (Porter)
       optional bigrams if use_bigrams = true in scoring.toml
  └─ BuildAgentDocs
       5 BM25F fields per agent: scope, role, responsibilities, read-chain, out-of-scope
  └─ ComputeBM25F (multi-field BM25F, IDF floor = 1.0 for small corpora)
  └─ Softmax (temperature-scaled; configurable T in scoring.toml)
  └─ Route (MIN_SCORE floor, CONFIDENCE_THRESHOLD for ambiguous routing)
  └─ BuildContextBlocks
       — selects relevant spec/doc/context nodes from graph
       — sorted by trust tier (specs > docs > context > notes)
       — notes excluded by default; enabled with --include-notes
  └─ Assemble routing packet (agent, confidence, scope, context_blocks, excluded)
  └─ loadRelevantConcepts (concepts from semantic.json owned by winning agent)
```

### oz context enrich

```
graph.json
  └─ EnrichmentPromptBuilder
       — builds structured JSON prompt from graph subgraph
  └─ OpenRouter client (OPENROUTER_API_KEY)
       — sends prompt; requests JSON response conforming to semantic.json schema
  └─ ResponseParser
       — validates LLM JSON against schema
       — rejects malformed edges; logs skipped items; best-effort
  └─ Overlay writer (semantic.Write)
       — merges with existing semantic.json
       — preserves reviewed: true items across re-runs
       — embeds graph_hash for staleness detection
```

### oz context serve (MCP)

```
stdin (newline-delimited JSON-RPC 2.0)
  └─ Capability negotiation (initialize / notifications/initialized)
  └─ tools/list → four tool schemas
  └─ tools/call dispatcher:
       query_graph     → full routing packet (same as oz context query)
       get_node        → node by ID from graph.json
       get_neighbors   → adjacent nodes, optional edge-type filter
       agent_for_task  → agent name + confidence only (low token cost)
stdout (newline-delimited JSON-RPC 2.0 responses)
```

---

## Structural context graph (`context/graph.json`)

`oz context build` walks the workspace (respecting `.ozignore`), parses convention files, extracts cross-references, and writes a deterministic JSON graph. The on-disk artefact is `context/graph.json`. The Go types and constants live in `code/oz/internal/graph`; the schema version is `graph.SchemaVersion` (currently `"1"`).

Each graph object includes `schema_version`, `content_hash` (SHA-256 hex of the canonical `nodes` + `edges` JSON, excluding `content_hash`), `nodes`, and `edges`. Nodes and edges are sorted before write so repeated builds with unchanged inputs are byte-identical.

### Node types

| `type` | Meaning |
|--------|---------|
| `agent` | `agents/<name>/AGENT.md` |
| `spec_section` | An H2 section in a file under `specs/` (excluding `specs/decisions/`) |
| `decision` | A file under `specs/decisions/` |
| `doc` | An H2 section in a file under `docs/` |
| `context_snapshot` | A markdown file under `context/` |
| `note` | A markdown file under `notes/` |

Agent nodes carry optional fields parsed from AGENT.md (`role`, `scope`, `read_chain`, `rules`, `skills`, and related prose). Other nodes include `tier` (`specs`, `docs`, `context`, or `notes`) where applicable.

### Edge types

| `type` | Meaning |
|--------|---------|
| `reads` | Agent read-chain entry resolves to another graph node |
| `owns` | Agent scope path (may target a `path:…` pseudo-id if unmatched) |
| `references` | A document cites another file path (backticks or markdown links) |
| `supports` | A doc-tier node references a spec or decision |
| `crystallized_from` | Reserved for future crystallize integration |

`oz audit` loads this file (stub today) to prove downstream tools can consume the contract before full drift checks exist.

---

## Semantic overlay (`context/semantic.json`)

Schema version `"1"`. Produced by `oz context enrich`, reviewed via `oz context review`.

| Field | Description |
|-------|-------------|
| `schema_version` | Always `"1"` |
| `graph_hash` | SHA-256 of `graph.json` at generation time. Used for staleness detection. |
| `model` | OpenRouter model ID used for enrichment (default: `anthropic/claude-haiku-4`) |
| `generated_at` | RFC3339 UTC timestamp of last enrichment run |
| `concepts` | Extracted concept nodes (see below) |
| `edges` | Typed relationships between concepts and structural nodes |

**Concept node fields**: `id` (`concept:<slug>`), `name`, `description`, `source_files`, `tag` (`EXTRACTED` | `INFERRED`), `confidence`, `reviewed`.

**Edge types**: `agent_owns_concept`, `implements_spec`, `drifted_from`, `semantically_similar_to`.

### Staleness detection

On `oz context query` and `oz context serve` startup, the `graph_hash` embedded in `semantic.json` is compared against the SHA-256 of the current `graph.json`. If they differ:

```
warning: semantic overlay may be stale — run 'oz context enrich' to update
```

---

## Query pipeline: BM25F scoring

The query engine scores agents using multi-field BM25F. All parameters are configurable in `context/scoring.toml` (falls back to defaults if absent).

### BM25F fields (per agent)

| Field | Weight (default) | Source |
|-------|-----------------|--------|
| `scope` | 3.0 | Scope paths from AGENT.md Responsibilities section |
| `role` | 2.0 | Role paragraph from AGENT.md |
| `responsibilities` | 1.5 | Responsibilities section text |
| `read_chain` | 1.0 | Read-chain item paths and descriptions |
| `out_of_scope` | −0.5 | Out-of-scope section (negative weight) |

### IDF floor

IDF is floored at 1.0 to prevent degenerate scores on small corpora (3–5 agents). This resolves pre-mortem risk T-02 (Leiden degeneration) — BM25F with an IDF floor is deterministic and scale-invariant regardless of corpus size.

### Routing decision

1. Compute raw BM25F scores for all agents.
2. Apply temperature-scaled softmax (`T = 1.0` default) to convert to confidence values.
3. If top score < `MIN_SCORE` (default 0.0): return `no_clear_owner`.
4. If top confidence < `CONFIDENCE_THRESHOLD` (default 0.5): include `candidate_agents`.
5. Otherwise: route to the top-scoring agent.

---

## MCP interface

`oz context serve` implements MCP stdio protocol version `2024-11-05`. JSON-RPC 2.0 framing, one JSON object per line.

### Wire it in

```json
{"mcpServers":{"oz":{"command":"oz","args":["context","serve"]}}}
```

### Tools

| Tool | Input | Output |
|------|-------|--------|
| `query_graph` | `task` (string) | Full routing packet (agent, confidence, scope, context blocks, concepts) |
| `get_node` | `node_id` (string) | Node object with all fields and edges |
| `get_neighbors` | `node_id`, optional `edge_type` | Adjacent node list |
| `agent_for_task` | `task` (string) | `{ agent, confidence }` only — low token cost |

---

## Key decisions

See `specs/decisions/` for architectural decision records (ADRs).
