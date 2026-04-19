# Architecture

> High-level architecture of oz.

## Overview

Open workspace convention and toolset for LLM-first development

## Components

<!-- Describe the major components. -->

## Data flow

<!-- Describe how data flows through the system. -->

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

## Key decisions

See `specs/decisions/` for architectural decision records (ADRs).
