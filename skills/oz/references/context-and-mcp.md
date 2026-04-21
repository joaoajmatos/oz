# Reference: `oz context` and MCP

## `oz context build`

Regenerates gitignored `context/graph.json`: agents, markdown sections under convention paths, Go files and exported symbols under `code/`. **Run before** `oz context query`, `oz audit`, or `oz context serve` when those trees changed.

Never hand-edit `context/graph.json`. **STALE001** from `oz audit staleness` means rebuild.

---

## `oz context query`

Prints a **JSON routing packet**: `agent`, `confidence`, `scope`, `context_blocks`, `excluded`, and `relevant_concepts` when `context/semantic.json` exists and its `graph_hash` matches the graph.

| Flag | Effect |
|------|--------|
| `--raw` | Debug: BM25F scores, softmax confidences, bounded subgraph (not the full workspace graph). |
| `--include-notes` | Include `notes/` in context blocks (default excludes low-trust noise). |

If the result is **`no_clear_owner`**, widen the task string or use the read-chain manually.

---

## After `query` or MCP `query_graph`

1. Note **`agent`** and **`confidence`**.
2. Read **`context_blocks`** in trust order (specs before notes unless `--include-notes`).
3. Use **`scope`** as path guardrails; respect **`excluded`**.

---

## `oz context enrich` / `oz context review`

- **enrich** — sends the structural graph to an LLM via OpenRouter; requires **`OPENROUTER_API_KEY`**. Optional **`--model`**. Writes/merges `context/semantic.json`.
- **review** — interactive review of unreviewed overlay items, or **`--accept-all`** for CI-style acceptance.

---

## `oz context serve` (MCP)

Stdio MCP (protocol `2024-11-05`). Tools:

| Tool | Role |
|------|------|
| `query_graph` | Full routing packet (same as `oz context query`). |
| `get_node` | One graph node by ID. |
| `get_neighbors` | Adjacent nodes (optional edge-type filter). |
| `agent_for_task` | Shorthand: agent name + confidence only. |

**Wire-up** (project or global MCP config): see [README.md](../../../README.md). Command is `oz` with args `context`, `serve`. Keep **`oz context build`** current before relying on MCP in automation.
