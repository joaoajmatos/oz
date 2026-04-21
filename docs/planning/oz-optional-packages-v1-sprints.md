# oz optional packages V1 — Sprint plan & backlog

**PRD**: docs/planning/oz-optional-packages-v1-prd.md  
**Pre-mortem**: docs/planning/oz-optional-packages-v1-premortem.md  
**Format**: 1-week sprints, small team (adjust owners to your roster)

---

## Product backlog

**Format**: User stories (3 C’s + INVEST-friendly)  
**Total stories**: 10  
**Estimated total effort**: M–L (two to three weeks for one engineer, depending on F-06 merge work)

### Stories

#### Story 1: Package registry in code

**As an** oz maintainer, **I want** a single registry that maps package IDs to installers, **so that** adding a new package does not require scattered `if` chains.

Acceptance criteria:

- [ ] Registry lists valid package IDs used by help/errors
- [ ] Unknown ID returns a typed error suitable for CLI
- [ ] Unit test covers unknown ID and at least one known ID

Priority: P0 | Effort: S | Dependencies: none

---

#### Story 2: `pm` package templates

**As a** workspace adopter, **I want** the `pm` package to create a PM agent and skill tree, **so that** I can run PM workflows in-repo.

Acceptance criteria:

- [ ] `agents/pm/AGENT.md` exists with all required sections per oz spec
- [ ] `skills/pm/` contains entry `SKILL.md` (and optional `references/` / `assets/`) describing PM workflows
- [ ] PM agent **Skills** authorizes: create-prd, pre-mortem, write-stories (user/job/wwa), sprint (plan/retro/release-notes) — by name and where to find them

Priority: P0 | Effort: M | Dependencies: Story 1

---

#### Story 3: Install `pm` on existing workspace

**As a** user with an oz repo, **I want** `oz add pm`, **so that** I can opt in without re-running init.

Acceptance criteria:

- [ ] `oz add pm` resolves workspace root via `AGENTS.md` + `OZ.md`
- [ ] On success, stdout lists created paths
- [ ] Integration test exercises happy path in `t.TempDir()`

Priority: P0 | Effort: M | Dependencies: Stories 1–2

---

#### Story 4: Conflict safety

**As a** user, **I want** the CLI to refuse overwriting my files by default, **so that** I do not lose edits.

Acceptance criteria:

- [ ] If a package-owned file already exists, command fails with clear message (unless `--force` is implemented and documented)
- [ ] Test covers duplicate install

Priority: P0 | Effort: S | Dependencies: Story 3

---

#### Story 5: Opt-in during `oz init`

**As a** new project owner, **I want** to select optional packages during init, **so that** my first commit includes PM scaffolding.

Acceptance criteria:

- [ ] Interactive flow offers optional packages (V1: `pm`)
- [ ] Scaffold passes selection into installers
- [ ] Test or harness verifies filesystem output when package selected

Priority: P1 | Effort: M | Dependencies: Stories 1–2

---

#### Story 6: Helpful `oz add` UX

**As a** user, **I want** `oz add` help to explain packages vs integrations, **so that** I am not confused by `claude`/`cursor` subcommands.

Acceptance criteria:

- [ ] `oz add --help` (or parent help) mentions packages: `oz add <package>` and lists example IDs
- [ ] Error for bad package name lists valid IDs

Priority: P1 | Effort: S | Dependencies: Story 1

---

#### Story 7: Register `pm` in manifest tables (F-06)

**As an** LLM entering the workspace, **I want** `AGENTS.md` / `OZ.md` to list the PM agent after install, **so that** routing works without manual edits.

Acceptance criteria:

- [ ] After `oz add pm`, manifest tables include `pm` **or** documented one-step manual update with a test that fails if docs drift
- [ ] Pre-mortem OP-02 closed with recorded decision

Priority: P1 | Effort: M–L | Dependencies: Stories 2–3 — **spike if merge is risky**

---

#### Story 8: Documentation parity

**As a** contributor, **I want** `docs/test-plan.md` updated for new commands, **so that** CI expectations stay clear.

Acceptance criteria:

- [ ] Test plan lists `oz add pm` scenarios
- [ ] Architecture or open-items note if validate/audit noise changes

Priority: P2 | Effort: S | Dependencies: Story 3

---

#### Story 9: Dogfood PM package in oz repo (optional)

**As the** oz team, **I want** to optionally install `pm` here, **so that** we exercise real workflows.

Acceptance criteria:

- [ ] Decision recorded: ship templates only vs also adopt `agents/pm` in this repo
- [ ] If adopted, `oz validate` still passes

Priority: P2 | Effort: S | Dependencies: Story 3

---

#### Story 10: Release notes snippet

**As a** user upgrading oz, **I want** release notes to mention optional packages, **so that** I discover `pm`.

Acceptance criteria:

- [ ] Short user-facing blurb: `oz add pm`, init opt-in
- [ ] Links to PRD for detail

Priority: P2 | Effort: S | Dependencies: V1 complete

---

### Story map

- **Must-have (V1)**: Stories 1–5, 7 (or explicit manual F-06 fallback), 6  
- **Should-have**: Story 8  
- **Nice-to-have**: Stories 9–10  

### Technical notes

- Keep **`claude` / `cursor`** as dedicated subcommands in V1 (per PRD).
- Shared installer logic between **init** and **add** to avoid drift.
- Templates live under `internal/scaffold/templates/packages/pm/...` (path indicative).

### Open questions

- Exact **F-06** strategy: programmatic merge of markdown tables vs post-install instructions?
- Should **`oz add pm`** support `--dry-run` in V1 or defer to V1.1?

---

## Sprint P0 — Lock docs & scope (no code)

**Goal**: PRD, pre-mortem, backlog frozen; F-06 decision picked.

### Tasks

| # | Task | Done when |
|---|------|-----------|
| P0-01 | Review PRD with stakeholders | No open blocking comments |
| P0-02 | Resolve F-06 approach | Decision paragraph added to PRD or sprint notes |
| P0-03 | Pre-mortem sign-off | OP-01/OP-02 owners acknowledge |

**Capacity**: ~0.5 engineer-week.

---

## Sprint P1 — Registry + `pm` + `oz add pm`

**Goal**: `oz add pm` works; conflict rules tested; core templates committed.

### Selected stories

| # | Story | Owner | Risk |
|---|-------|-------|------|
| 1 | Package registry | eng | Low |
| 2 | `pm` templates | eng + PM reviewer | Medium (content quality) |
| 3 | `oz add pm` | eng | Low |
| 4 | Conflict safety | eng | Medium (edge cases) |
| 7 | F-06 manifest | eng | High if merge auto |

**Recommended commitment**: Stories 1–4 required; Story 7 if spike says merge is feasible; else ship manual step + test.

### Sprint risks

- **F-06** underestimation — time-box spike to 1 day; fallback to documented manual registration.

### Definition of done

- [ ] `go test ./...` green
- [ ] Pre-mortem OP-01 mitigations verified by tests
- [ ] Valid/invalid package ID behaviour covered

---

## Sprint P2 — Init integration + UX + docs

**Goal**: Init multi-select; help text; test plan + release blurb.

### Selected stories

| # | Story |
|---|-------|
| 5 | Init opt-in |
| 6 | Help UX |
| 8 | Test plan |
| 10 | Release notes (if shipping immediately) |

### Definition of done

- [ ] Init path covered by test or integration harness
- [ ] User-facing help distinguishes packages vs integrations
- [ ] `docs/test-plan.md` updated

---

## Capacity template (fill for your team)

| Member | Available days | Hours | Notes |
|--------|----------------|-------|-------|
| _Eng 1_ | | | |
| _Eng 2_ | | | |

**Rule of thumb**: Plan to **70%** of raw hours; keep buffer for F-06 spike and review of PM templates.

---

## Definition of done (V1 release)

- [ ] All P0–P1 must-have stories accepted
- [ ] Pre-mortem go/no-go checklist complete
- [ ] No launch-blocking Tigers open without explicit waiver
