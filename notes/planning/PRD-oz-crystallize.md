---
crystallize: spec
crystallize-title: "PRD: oz crystallize command"
---

# PRD: oz crystallize

## 1. Summary

`oz crystallize` is a new CLI command that promotes raw notes from `notes/` into canonical workspace artifacts (`specs/`, `docs/`, ADRs). It closes the loop on the `notes/ → specs/` lifecycle that the oz convention describes but cannot yet automate, making the workspace's source-of-truth hierarchy self-maintaining.

---

## 2. Contacts

| Name | Role | Responsibility |
|------|------|----------------|
| João Matos | Author / Maintainer | Implementation owner, spec alignment |
| oz-coding agent | Implementation agent | Go code, tests, CLI integration |
| oz-spec agent | Spec agent | Convention alignment, ADR authoring |
| oz-maintainer agent | Maintainer agent | Workspace manifest updates, oz validate/audit |

---

## 3. Background

### What is this about?

oz enforces a clear source-of-truth hierarchy:

```
specs/ > docs/ > context/ > notes/
```

`notes/` is the lowest-trust tier — it is where raw thinking, planning artifacts, spike notes, and decisions-in-progress live. Over time, material in `notes/` crystallizes into higher-trust artifacts: a decision becomes an ADR, a planning note becomes a spec, a walkthrough becomes a guide.

Today this promotion happens manually, if at all. There is no tooling to:
- Detect which notes are ready to promote
- Determine the correct target location and format
- Execute the promotion with proper scaffolding (e.g., ADR numbering)

As a result, `notes/` accumulates stale material and the specs/docs tiers remain incomplete. `oz audit` flags these as orphans and staleness warnings, but has no power to fix them.

### Why now?

oz V1 shipped all five core commands (init, validate, audit, context, add). `oz crystallize` was explicitly deferred to a post-V1 milestone. With V1 stable and the workspace self-validating, this is the natural next feature — it is the only planned command not yet implemented.

The oz repo itself currently has:
- 4 orphaned spec files that should be cross-linked (audit errors on its own repo)
- 5 orphaned guides under `docs/guides/` that are undiscoverable
- A `notes/planning/` tree with historical PRDs and sprint notes that have never been promoted

`oz crystallize` would resolve all of these directly.

### Is this newly possible?

`oz context enrich` has already proven the OpenRouter integration pattern. The same `internal/openrouter/` client can power the crystallize classifier — giving it workspace-aware context (existing ADRs, specs, guides as few-shot examples) so it understands this workspace's conventions, not just generic patterns. The diff-first review UX is already established in `oz context review`. All the building blocks exist.

---

## 4. Objective

### What is the objective?

Make the oz convention self-maintaining by automating the promotion of raw notes into canonical workspace artifacts.

### Why does it matter?

- **For users**: eliminates manual file-moving, renaming, and formatting work. Teams can write freely in `notes/` knowing they can promote to canon at any time.
- **For the convention**: a workspace that can heal its own `notes/` backlog is dramatically more trustworthy than one that relies on humans to maintain the hierarchy.
- **For oz as a product**: completes the V1 feature set. Every planned command ships. The workspace dogfoods its own crystallize workflow.

### Alignment with oz principles

- **Convention over config** — the workspace's own artifacts teach the classifier what belongs where; no user configuration needed.
- **Code is source of truth for behaviour** — the command enforces the hierarchy by making promotion the path of least resistance.
- **LLM-augmented by default, deterministic by fallback** — OpenRouter is the primary classifier; the heuristic engine is the fallback for offline / no-key environments.
- **Self-dogfooding** — oz's own `notes/planning/` will be the first corpus crystallized.

### Key Results (SMART)

| # | Key Result | Measure |
|---|-----------|---------|
| KR1 | `oz crystallize --dry-run` correctly classifies ≥90% of oz's own `notes/` files without user correction | Validated against manual ground-truth labels |
| KR2 | `oz audit` error count on the oz repo drops from 4 to 0 after running `oz crystallize` on the orphaned specs | Measured by `oz audit --json` output |
| KR3 | Zero partial-state writes: every write is atomic (temp-file + rename) | Verified by test that kills the process mid-write |
| KR4 | `oz crystallize` added to oz-coding agent read-chain and covered by `oz validate` | `oz validate` exits 0 on the oz repo |

---

## 5. Market Segment

### Who is this for?

**Primary**: Individual developers and small teams using oz workspaces for LLM-assisted development. They write frequently in `notes/` during planning, spikes, and design sessions but rarely promote that material to `specs/` or `docs/` because it requires knowing the right format and location.

**Secondary**: oz itself as a product (self-dogfooding). The oz repo accumulates planning notes and the maintainer benefits from the same automation.

### Constraints

- Users may have large `notes/` directories with mixed content — the classifier must handle noise gracefully (skip unknowns, surface ambiguous cases).
- Some users will run `oz crystallize` in CI (`--dry-run`, `--accept-all`); the command must be safe to run non-interactively.
- Users expect the convention to be stable — the command must never silently overwrite existing files.

---

## 6. Value Proposition

### Jobs to be done

| Job | Current pain | After crystallize |
|-----|-------------|------------------|
| Promote a design decision to an ADR | Manually create file, pick number, copy template, move content | `oz crystallize` detects it, numbers it, scaffolds it, prompts for review |
| Turn a planning note into a spec | Unsure of correct location, format, or whether it overlaps existing specs | Command routes it correctly, diffs against existing content |
| Clean up `notes/` backlog | No tooling — it just sits there | One command surfaces everything ready to promote |
| Fix `oz audit` orphan errors | Manually find and link orphaned files | Run `oz crystallize`, re-run audit, see delta |
| Onboard a new agent to the workspace | Agent reads stale notes as if they were canonical | Notes are promoted to specs/docs; notes/ contains only active drafts |

### Gains

- Workspace hierarchy stays accurate without manual maintenance.
- ADRs get sequential numbers automatically — no collision risk.
- The crystallize log (`context/crystallize.log`) provides a traceable lineage from note to canonical artifact.
- `--dry-run` mode builds confidence before any files are written.

### Pains avoided

- Accidental overwrites (command diffs and prompts before writing).
- Partial writes on interrupt (atomic write pattern).
- Classification non-determinism (LLM results cached by file hash; same input → same output on repeated runs).

---

## 7. Solution

### 7.1 User Flow

```
$ oz crystallize

Scanning notes/ ... 7 files

  notes/planning/sprint-3.md         → adr        (confidence: high)
  notes/planning/auth-redesign.md    → spec        (confidence: high)
  notes/howto/setup-cursor.md        → guide       (confidence: high)
  notes/thinking/graph-layers.md     → arch        (confidence: medium)
  notes/scratch/open-qs.md           → open-item   (confidence: high)
  notes/mixed/decision-with-steps.md → ?           (ambiguous: adr or guide)
  notes/scratch/random-notes.md      → unknown     (score too low, skipping)

[1/6] Reviewing: notes/planning/sprint-3.md → specs/decisions/0003-auth-rewrite.md

  --- notes/planning/sprint-3.md
  +++ specs/decisions/0003-auth-rewrite.md
  @@ -1,3 +1,10 @@
  +---
  +status: accepted
  +date: 2026-04-22
  +---
  +
  +# ADR-0003: Auth Rewrite
  +
   We decided to rewrite the auth middleware because...

  (a)ccept  (e)dit  (s)kip  (q)uit

...

Done. 5 files promoted.

  specs/decisions/0003-auth-rewrite.md   ✓
  specs/auth-redesign.md                 ✓
  docs/guides/setup-cursor.md            ✓
  docs/graph-layers.md                   ✓
  docs/open-items.md (appended)          ✓

Audit delta:
  Before: 4 errors, 7 warnings
  After:  1 error,  5 warnings   (-3 errors, -2 warnings)

To undo: git checkout -- notes/ && git rm specs/decisions/0003-auth-rewrite.md ...
Logged to context/crystallize.log
```

### 7.2 Key Features

#### F1: LLM Classifier — primary (`internal/crystallize/classifier`)

Classifies each note using OpenRouter with workspace-aware context. The prompt includes:

- The oz source-of-truth hierarchy and artifact type → target-path mapping
- Few-shot examples pulled from the workspace's own existing artifacts (first 30 lines of a real ADR, spec, and guide)
- The note content

The LLM returns a structured JSON response: `{"type": "adr|spec|guide|arch|open-item|unknown", "confidence": "high|medium|low", "title": "<suggested title>", "reason": "<one sentence>"}`.

**Cache** (`internal/crystallize/cache`): LLM results are cached in `.oz/crystallize-cache.json` keyed by `{path, sha256(content), model}`. Same input → same output on repeated runs. `--dry-run` in CI is deterministic. `--no-cache` forces a fresh LLM call.

**Explicit tag wins**: frontmatter `crystallize: <type>` skips both LLM and heuristic entirely.

**Types**: `adr`, `spec`, `guide`, `arch`, `open-item`

#### F1b: Heuristic Classifier — fallback (`internal/crystallize/classifier/heuristic`)

Used when no API key is configured, `--no-enrich` is passed, or the LLM call fails. Keyword + structure signal scoring:

- Strong signals ×3 (e.g., "we decided", MUST/SHALL, "how to")
- Structure signals ×2 (e.g., H2 sections named "Decision", ordered step lists)
- Supporting signals ×1, Anti-signals −2
- Score < 4 → unknown; gap < 2 → ambiguous

#### F2: Interactive Review (`internal/crystallize/review`)

Mirrors the `oz context review` UX.

- Shows a diff between source note and proposed canonical artifact before writing
- Per-file actions: (a)ccept, (e)dit, (s)kip, (q)uit
- Quit exits cleanly — no partial writes
- `--accept-all` skips interactive review (for CI/scripting)

#### F3: Atomic Promotion (`internal/crystallize/promote`)

- Writes to `<target>.tmp`, then renames to `<target>` on success
- For `open-item` type: appends to `docs/open-items.md` with a new `## <title>` section
- ADR auto-numbering: scans `specs/decisions/` for the highest `NNNN` and increments
- Collision guard: if target path already exists, diffs and prompts (never silent overwrite)
- Scaffold injection: ADR template frontmatter (`status`, `date`) added automatically

#### F4: Crystallize Log (`context/crystallize.log`)

Append-only log recording each promotion:

```
2026-04-22T14:30:00Z  notes/planning/sprint-3.md  →  specs/decisions/0003-auth-rewrite.md  [adr]
```

Provides traceable lineage. Used by `oz audit` staleness check in future.

#### F5: Audit Delta Report

After all promotions, re-runs the oz audit checks that were affected (orphans, coverage, staleness) and prints a before/after delta. Gives users immediate feedback that the workspace is healthier.

#### F6: Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show classification and diff without writing anything |
| `--topic <t>` | Only process notes whose content matches the given topic string |
| `--no-enrich` | Skip OpenRouter call; use heuristic classifier instead (for offline / no-key environments) |
| `--no-cache` | Force a fresh LLM call even if a valid cache entry exists |
| `--accept-all` | Skip interactive review, accept all high-confidence classifications |

### 7.3 Technology

- **Package layout**:
  ```
  internal/crystallize/
    classifier/           — LLM classifier (OpenRouter) + heuristic fallback
    classifier/heuristic/ — keyword + structure signal scoring engine
    cache/                — .oz/crystallize-cache.json read/write
    promote/              — atomic write, ADR numbering, template injection
    review/               — diff-first interactive review TUI
    log/                  — crystallize.log append
  cmd/crystallize.go — cobra command, flag wiring
  ```
- **Dependencies**: `internal/openrouter/` (existing) for LLM calls. `charmbracelet/huh` for interactive review (already a dep). `gopkg.in/yaml.v3` for frontmatter parsing (already a dep). stdlib for heuristic fallback.
- **Testing**: table-driven unit tests for heuristic classifier with golden fixtures, LLM classifier tested with a mock OpenRouter client, cache hit/miss/invalidation tests, integration test using `testws` package, determinism test (cache-hit path: 100 runs, byte-identical dry-run output).

### 7.4 Assumptions

| # | Assumption | Risk if wrong |
|---|-----------|---------------|
| A1 | LLM with few-shot workspace context classifies ≥90% of real-world notes correctly | Classifier is too noisy; users lose trust and stop using it |
| A2 | Users are comfortable with a diff-first interactive review before each write | Review feels too slow for large notes/ backlogs |
| A3 | ADR sequential numbering from filesystem scan is reliable (no concurrent writers) | Number collision in team workflows (mitigated: rare for CLI tools) |
| A4 | `--dry-run` is sufficient for CI use cases; `--accept-all` covers scripting | Teams want fully automated crystallize in CI with zero prompts |
| A5 | The `open-item` append pattern (to a single file) scales to large backlogs | `docs/open-items.md` becomes unmanageable; should be a directory |
| A6 | File-hash-keyed cache makes LLM classification deterministic enough for CI | Cache misses or invalidation on unchanged files causes flaky CI |

---

## 8. Release

### V1 (this PRD)

- LLM classifier (OpenRouter) as primary — workspace-aware few-shot context
- Heuristic classifier as fallback (`--no-enrich`, no API key, LLM failure)
- Classification cache (`.oz/crystallize-cache.json`, keyed by file hash + model)
- Interactive diff-first review
- Atomic promotion for all types
- ADR auto-numbering
- Crystallize log
- Audit delta report
- Flags: `--dry-run`, `--topic`, `--accept-all`, `--no-enrich`, `--no-cache`
- Full test coverage (unit + integration + mock LLM client + cache)

### V2 (future)

- Watch mode (`oz crystallize --watch`): monitor `notes/` for new files and suggest crystallization
- Reverse crystallize: flag a canonical doc as "needs rethinking" and demote back to `notes/`
- `open-item` directory mode: split large backlogs into `docs/open-items/` directory
- Staleness integration: `oz audit staleness` surfaces notes older than N days as crystallize candidates
- Multi-workspace support (crystallize across linked workspaces)

### What is NOT in V1

- Automatic git commits after crystallization
- Tree-sitter / multi-language awareness
- LLM-generated content (enrich only classifies, does not rewrite)
- Web UI or non-CLI interface
