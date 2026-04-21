# PM package — skill inventory (implementation staging)

**Purpose**: Hold the **canonical list** of PM workflows and how they map to oz skill layout before `oz add pm` (or scaffold templates) materializes real files under `skills/`.

**Do not use a temp file** for this: temp paths are not versioned, are ignored by PR review, and will not show up in `oz context build`. Use **this doc** (or `notes/` for rough dumps you plan to crystallize) until the package lands.

**Normative skill shape** (see `specs/oz-project-specification.md`): each skill is `skills/<name>/SKILL.md` with YAML frontmatter (`name`, `description`, `triggers`), plus `## When to invoke` and `## Steps`. Use `references/` for branching paths (formats, sprint modes).

---

## Naming: avoid nested `skills/pm/...`

The oz spec defines **`skills/<name>/`** at workspace root with **one kebab-case segment** for `<name>`. Nesting like `skills/pm/create-prd/` is not the documented shape.

**Recommended**: prefix skill names so the PM bundle stays grouped and searchable:

| Planned directory | `name` (frontmatter) | Rationale |
|-------------------|----------------------|-----------|
| `skills/pm-create-prd/` | `pm-create-prd` | PRD workflow |
| `skills/pm-pre-mortem/` | `pm-pre-mortem` | Pre-mortem |
| `skills/pm-write-stories/` | `pm-write-stories` | Router + formats in `references/` |
| `skills/pm-sprint/` | `pm-sprint` | Router + modes in `references/` |

Alternative acceptable pattern: shorter prefixes (`prd`, `pre-mortem`) if you want fewer `pm-` prefixes — but then PM and non-PM skills share one flat namespace.

---

## Skill definitions (summary for scaffold / AGENT.md)

### 1. `pm-create-prd`

- **Description**: Create a Product Requirements Document using the 8-section template (summary, contacts, background, objective, segments, value propositions, solution, release).
- **Triggers** (examples): `PRD`, `product requirements`, `write PRD`, `create-prd`.
- **Steps**: Gather inputs → apply template sections → save as `PRD-<product-name>.md` (or project convention under `docs/planning/`).
- **References** (optional): `references/eight-section-outline.md` with the section checklist only (keep SKILL.md short).

**Source body**: Use the **create-prd** playbook you maintain in Cursor/plugins or paste the canonical instructions into `references/` during implementation — do not rely on chat-only copies.

### 2. `pm-pre-mortem`

- **Description**: Run a pre-mortem on a PRD or launch plan (Tigers, Paper Tigers, Elephants; launch-blocking / fast-follow / track; action plans).
- **Triggers**: `pre-mortem`, `premortem`, `risk analysis`, `launch risks`.
- **Steps**: Read target doc → assume failure → classify → output `PreMortem-<product>-<date>.md`.
- **References** (optional): `references/output-template.md` for the fixed markdown structure.

### 3. `pm-write-stories`

- **Description**: Break work into backlog items with acceptance criteria; supports **user stories**, **job stories**, and **WWA**.
- **Triggers**: `write stories`, `backlog`, `user stories`, `job stories`, `WWA`, `/write-stories`.
- **Steps**: Confirm format → decompose 5–15 items → AC per item → story map + technical notes + open questions.
- **References** (required for branching):
  - `references/user-stories.md`
  - `references/job-stories.md`
  - `references/wwa.md`

### 4. `pm-sprint`

- **Description**: Sprint lifecycle helpers — **plan**, **retro**, **release-notes** (aligned with `/sprint` command modes).
- **Triggers**: `sprint plan`, `sprint retro`, `release notes`, `/sprint`.
- **Steps**: Route to the correct reference by mode; fill the template for that mode.
- **References** (required):
  - `references/plan.md`
  - `references/retro.md`
  - `references/release-notes.md`

---

## PM agent `AGENT.md` — Skills section (target wording)

The PM agent should authorize invocation of:

- `skills/pm-create-prd/`
- `skills/pm-pre-mortem/`
- `skills/pm-write-stories/`
- `skills/pm-sprint/`

Optional: add **Context topics** pointing at `docs/planning/` for shipped PRDs and this inventory.

---

## Relation to Cursor slash commands

Slash commands (`/write-stories`, `/sprint`) are **editor affordances**. Oz skills are **repo-local playbooks**. Keep them aligned by:

1. Putting the **same procedural content** in `references/*.md` where it makes sense.
2. Documenting in each SKILL.md: “If your editor exposes `/write-stories`, it should route to the same formats as `references/`.”

---

## Implementation order (suggested)

1. Add this inventory (done).
2. When implementing the `pm` package: scaffold the four directories above with minimal valid `SKILL.md` files + `references/` stubs.
3. Port full text from your Cursor skill definitions into `references/` (not into ephemeral chat).
4. Run `oz validate` and `oz context build` on a fixture workspace in tests.

---

## Open decisions

- [ ] Exact trigger keyword list per skill (keep short vs exhaustive).
- [ ] Whether PRD/pre-mortem output paths default to `docs/planning/` only or also `notes/` for drafts.
