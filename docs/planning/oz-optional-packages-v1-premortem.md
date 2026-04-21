# Pre-Mortem Analysis: oz optional packages V1

**Date**: 2026-04-21
**Status**: Open — pre-launch
**PRD**: docs/planning/oz-optional-packages-v1-prd.md
**Sprint plan**: docs/planning/oz-optional-packages-v1-sprints.md

---

## Tigers (real risks)

### OP-01 — Accidental overwrite of user work

**Category**: Launch-blocking  
**Story**: A user runs `oz add pm` twice or already has `agents/pm/AGENT.md`; the tool overwrites their edits.  
**Mitigation**: Default to **fail if destination exists** for package-owned paths; document `--force` only if explicitly implemented and scoped. Add tests for idempotence and conflict.  
**Owner**: oz-coding  
**Due**: Before V1 merge

### OP-02 — AGENTS.md / OZ.md drift after install

**Category**: Launch-blocking (if we promise automatic registration)  
**Story**: Files land on disk but the entry table never lists `pm`, so LLMs skip the agent.  
**Mitigation**: Either (a) implement a safe merge for the agent table in V1, or (b) ship a clear post-install step in stdout + `skills/pm` README; PRD F-06 tracks the decision.  
**Owner**: oz-coding + oz-spec  
**Due**: End of Sprint P1

### OP-03 — “Packages” confuse users with `oz add claude` / `cursor`

**Category**: Fast-follow  
**Story**: Users think Claude integration is `oz add claude` but PM is `oz add pm` — inconsistent mental model.  
**Mitigation**: Help text on `oz add` lists **integrations** vs **packages**; README line in scaffold output. Consider `oz add --help` grouping.  
**Owner**: oz-coding  
**Due**: Sprint P2

### OP-04 — PM skill set out of sync with Cursor commands

**Category**: Fast-follow  
**Story**: PM agent references slash-commands that are not installed in the user’s editor.  
**Mitigation**: PM agent Skills section distinguishes “in-repo playbooks” vs “editor commands”; prefer linking to **files under `skills/pm/`** as source of truth.  
**Owner**: oz-maintainer  
**Due**: Sprint P2

### OP-05 — Validate / audit noise from new paths

**Category**: Track  
**Story**: New agent or skills trigger `COV002`/`ORPH002`-class warnings in some workspaces.  
**Mitigation**: Run self-validation on oz repo after landing; document accepted noise in `docs/open-items.md` if any.  
**Owner**: oz-coding  
**Due**: Before V1 tag

---

## Paper Tigers (overblown concerns)

**PP-01 — “We need a full plugin marketplace like npm.”**  
V1 is embedded-only by design. A marketplace is not required to prove value; it would add signing, networking, and support burden.

**PP-02 — “Packages must be Turing-complete.”**  
No. Packages are **files + templates**. That matches oz’s markdown-first architecture.

**PP-03 — “Every package must auto-update across oz versions.”**  
Not in V1. Users re-run `oz add` or bump oz and follow release notes.

---

## Elephants (unspoken worries)

**PE-01 — Who owns package content quality?**  
Packages ship **opinionated** workflows. If PM templates are weak, users blame oz. Mitigation: keep V1 **minimal** (agent + skeleton skills); iterate from dogfooding.

**PE-02 — Licensing of bundled “skill” text**  
If templates copy external material, legal risk. Mitigation: only ship original or properly licensed text; cite sources in package README if needed.

---

## Action plans for launch-blocking tigers

| Risk | Mitigation | Owner | Due date |
|------|------------|-------|----------|
| OP-01 Overwrite | Exist-check before write; tests; documented `--force` policy | oz-coding | Sprint P1 |
| OP-02 Registration | Implement merge OR explicit manual step + verification test | oz-coding / oz-spec | Sprint P1 |

---

## Go / No-Go checklist (before tagging V1)

- [ ] `oz add pm` and init opt-in produce expected tree; conflict behaviour tested
- [ ] `oz add not-a-package` errors with list of valid IDs
- [ ] PM agent lists authorized skills: create-prd, pre-mortem, write-stories, sprint modes
- [ ] Self-validation: `go test ./...` green; note any new audit/validate noise
- [ ] PRD + this pre-mortem + sprint plan committed under `docs/planning/`
