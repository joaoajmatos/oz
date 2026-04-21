# Product Requirements Document: oz optional packages (V1)

**Author**: oz-spec (draft for review)
**Date**: 2026-04-21
**Status**: Draft
**Stakeholders**: oz-coding (implementation), oz-maintainer (convention), oz-spec (spec alignment)
**Related**: Pre-mortem `docs/planning/oz-optional-packages-v1-premortem.md`, sprint plan `docs/planning/oz-optional-packages-v1-sprints.md`

---

## 1. Summary

This document defines **optional packages** for oz: bundles of workspace files (agents, skills, and supporting assets) that users can turn on when they run `oz init` or add later with **`oz add <package>`**. V1 ships one package, **`pm`** (project management), which adds a dedicated PM agent and a starter set of product-management workflows. Existing **`oz add claude`** and **`oz add cursor`** stay as they are today; they are not migrated into the package registry in V1.

---

## 2. Contacts

| Name / role | Responsibility |
|-------------|----------------|
| **Product / spec (oz-spec)** | PRD owner, convention wording, acceptance of V1 scope |
| **Engineering (oz-coding)** | `init` / `add` / scaffold changes, tests, embedded templates |
| **Convention health (oz-maintainer)** | AGENTS.md / OZ.md patterns, drift and validate implications |
| **Users** | Teams adopting oz who want PM workflows without forking the core repo |

---

## 3. Background

oz already scaffolds a **core** workspace: agents, specs, docs, context, skills, rules, and optional IDE hooks. Many teams want **optional** content on top of that—similar in spirit to **plugins** in other tools: opt-in bundles that extend the workspace without bloating the default template.

Today there is no first-class way to say “give me the PM agent and PM skills” except copying files by hand. That breaks the “convention over configuration” story and makes upgrades painful.

**Why now**

- The core scaffold and `add` patterns are stable enough to hang a small extension model on.
- A **`pm`** package unlocks dogfooding: planning artifacts (PRD, pre-mortem, sprint) live beside code in a repeatable shape.

**What became possible**

- Packages can ship as **embedded templates** in the Go binary (same pattern as `internal/scaffold`), so optional content stays versioned with `oz` and installs offline.

---

## 4. Objective

### Objective

Deliver a **simple, documented** way to install optional workspace bundles so users can grow an oz workspace beyond the default layout without manual file copying.

### Why it matters

- **Users** get a predictable path from “empty oz workspace” to “PM-ready workspace.”
- **Maintainers** can add new packages later without redesigning `init` each time.
- **The oz project** can ship opinionated workflows (PM first) while keeping the **core convention** small.

### Alignment

Fits oz’s vision: markdown-first, single binary, machine-checkable structure. Packages add **files** that `oz validate` and `oz context build` already understand—not a parallel plugin runtime.

### Key results (SMART)

| Key result | Target | Timeframe |
|------------|--------|-----------|
| **KR1 — Install path** | A user can install the `pm` package via `oz init` (opt-in) or `oz add pm` on an existing workspace | V1 ship |
| **KR2 — Discoverability** | `oz add` with missing/invalid package name prints a clear error listing **available** package IDs | V1 ship |
| **KR3 — Safety** | Default install does **not** silently overwrite user-edited files; conflicts surface an actionable error (or documented `--force` if implemented) | V1 ship |
| **KR4 — Quality** | New behaviour covered by automated tests (`internal/scaffold` and/or `cmd`) per `docs/test-plan.md` | V1 ship |
| **KR5 — PM agent skills** | The `pm` agent definition lists authorized skills/workflows: **create-prd**, **pre-mortem**, **write-stories** (user / job / WWA), **sprint** (plan / retro / release-notes), with pointers to skill entry points or in-repo docs | V1 ship |

---

## 5. Market segment(s)

**Primary — oz workspace adopters who run product discovery and delivery in-repo**

- Small teams or solo developers using oz for LLM-first development.
- They need PRDs, risk reviews, backlogs, and sprint plans **next to** specs and code—not only in external tools.

**Constraints**

- Must work **offline** after installing the `oz` binary (no network fetch for V1 packages).
- Must not require a new runtime (still one Go binary, embedded templates).
- Package content must respect existing **AGENT.md** and **SKILL.md** conventions (`specs/oz-project-specification.md`).

---

## 6. Value proposition(s)

| Job / need | Gain | Pain removed |
|------------|------|----------------|
| “I want PM rituals in my repo” | One command adds agent + skills shape | No manual copy/paste from another repo |
| “I want to opt in later” | `oz add pm` on existing workspace | No need to re-run full init |
| “I want room for more packages later” | Registry pattern for new IDs | No special-case explosion in `init` |
| “I want LLMs to know what PM can do” | PM agent read-chain + explicit **Skills** section | Less improvisation; clearer routing |

**Compared to ad-hoc markdown**: oz packages version the **shape** of the workspace with the tool, so teams share the same layout and validation rules.

---

## 7. Solution

### 7.1 UX / flows

**Init (interactive)**

- After core questions (name, code mode, agents, hooks), user can **multi-select optional packages** (V1: `pm` only).
- Scaffold runs core steps, then runs **package installers** for selected IDs.

**Add (non-interactive primary)**

- Command: **`oz add <package> [path]`**
  - Optional `path` resolves workspace root like today (`AGENTS.md` + `OZ.md`).
- Invalid ID → non-zero exit, stderr lists valid packages.
- Successful install → stdout lists files created or merged (human-readable).

**Note**: `oz add claude` and `oz add cursor` remain **separate subcommands** in V1 (no regression).

### 7.2 Key features

| ID | Feature | V1 behaviour |
|----|---------|--------------|
| F-01 | **Package registry** | Central map: package ID → installer (embedded templates) |
| F-02 | **`pm` package** | Adds `agents/pm/AGENT.md` and `skills/pm/...` (flows you refine later) |
| F-03 | **Init integration** | Selected packages installed during scaffold |
| F-04 | **`oz add <package>`** | Install package on existing workspace |
| F-05 | **Conflict policy** | If a tracked file already exists and would be written, **fail** with a clear message unless `--force` is explicitly specified and documented for that path class |
| F-06 | **AGENTS.md / OZ.md** | Installing `pm` updates **registered agents** in `AGENTS.md` and the manifest table in `OZ.md` **if** the scaffold templates support merge or full re-render; if merge is too risky in V1, installer documents manual “register pm in AGENTS.md” step — **decision**: prefer **idempotent append** or **template section** only if implementation can do so safely; otherwise ship `pm` with a short `skills/pm/README` telling users to add the table row (tracked as fast-follow if needed) |

**PM agent — authorized skills and workflows**

The PM agent’s **Skills** section MUST authorize at least the following (by name and path or Cursor command reference):

| Capability | Purpose |
|------------|---------|
| **create-prd** | Produce an 8-section PRD (summary, contacts, background, objective, segments, value props, solution, release) |
| **pre-mortem** | Tigers / Paper Tigers / Elephants + launch-blocking mitigations |
| **write-stories** | Backlog decomposition: **user stories**, **job stories**, or **WWA** with acceptance criteria |
| **sprint** | **plan** (capacity + story selection), **retro**, **release-notes** |

V1 may reference these as **Cursor slash-commands** and/or as **skills under `skills/pm/`** that embed or link to the same playbooks; the exact file layout is implementation detail as long as the PM agent read-chain makes them discoverable.

### 7.3 Technology

- **Go**, `embed` for templates under `internal/scaffold/templates/packages/<id>/...`
- Reuse existing `writeTemplate` / directory creation patterns from `internal/scaffold/scaffolder.go`
- Tests: `t.TempDir()`, no network

### 7.4 Assumptions

- V1 packages are **shipped with the binary**, not downloaded.
- Package installs are **file writes** only; no database or config service.
- Users who heavily customized `AGENTS.md` may need **manual** merge for registry updates if automatic merge is deferred (F-06).

---

## 8. Release

| Phase | Scope | Relative timeframe |
|-------|--------|---------------------|
| **V1.0** | Registry, `pm` package, `oz add <package>`, `oz init` multi-select, tests, docs in `docs/planning/` | First release after implementation complete |
| **V1.1** | Second package (TBD), safer manifest merge, `oz add pm --dry-run` | After V1 feedback |
| **V2** | Migrate `claude` / `cursor` into the same registry (if desired), optional remote package feeds | Future |

**Out of scope (V1)**

- Remote plugin marketplace, signing, or versioning across repos
- Changing Claude/Cursor add commands

---

## Appendix: Package ID rules (normative for V1)

- Lowercase, `[a-z][a-z0-9-]*` (hyphen allowed), max length 32.
- IDs are stable; renaming is a breaking change for scripts—avoid.
