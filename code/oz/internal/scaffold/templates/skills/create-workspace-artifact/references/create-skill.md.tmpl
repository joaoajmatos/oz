# Reference: Create or Tweak a Skill

Use this reference when creating a new `skills/<name>/` or modifying an existing one.

---

## Required structure

```
skills/<name>/
├── SKILL.md             # Required. Entry point: when to invoke and steps to follow.
├── references/          # Optional (required if the skill has multiple execution paths).
│   └── <path>.md
└── assets/              # Optional (required if the skill uses templates or support files).
    └── <file>
```

`SKILL.md` is always required. `references/` and `assets/` are only required when the skill
has branching logic or templated outputs. For trivial single-path skills, a standalone
`SKILL.md` is sufficient.

---

## SKILL.md required frontmatter

Every `SKILL.md` must begin with a YAML frontmatter block containing:

| Field | What it must contain |
|---|---|
| `name` | The skill name. Must match the directory name (kebab-case). |
| `description` | One sentence describing what the skill does and when it is useful. |
| `triggers` | A list of keywords or phrases that should invoke this skill. Be specific — these are used by `oz validate` and `oz context` for skill discovery. |

## SKILL.md required sections

Every `SKILL.md` must contain these sections (after the frontmatter):

| Section | What it must contain |
|---|---|
| `## When to invoke` | The conditions or signals that should trigger this skill. Be specific. |
| `## Steps` | Ordered, actionable instructions. Numbered list. No vague steps. |

Additional sections (`## References`, `## Notes`) are allowed but not required.

---

## Steps

### Creating a new skill

1. Decide the skill name. Use lowercase kebab-case describing the task (e.g. `create-workspace-artifact`).
2. Create the directory: `skills/<name>/`.
3. Copy `assets/SKILL.md.tmpl` to `skills/<name>/SKILL.md`.
4. Fill in the frontmatter: set `name`, `description`, and `triggers`. Do not leave placeholder text.
5. Fill in `## When to invoke` and `## Steps`. Do not leave placeholder text.
6. If the skill has multiple execution paths, create `skills/<name>/references/` and add one
   markdown file per path. Reference each file explicitly in `## Steps`.
7. If the skill uses templates or support files, add them to `skills/<name>/assets/`.
8. Register the skill in the authorizing agent's `AGENT.md` under `## Skills`:
   `- \`skills/<name>/\` — <one-line description>`
9. Verify: confirm `SKILL.md` has frontmatter with all three fields, both required sections, and that all referenced files exist.

### Tweaking an existing skill

1. Open `skills/<name>/SKILL.md`.
2. Make the targeted change.
3. Confirm frontmatter is present with `name`, `description`, and `triggers` populated.
4. Confirm `## When to invoke` and `## Steps` are still present and accurate.
5. If references or assets changed, update the relevant files in `references/` or `assets/`.
6. Verify: confirm all files referenced in `SKILL.md` still exist and are up to date.

---

## Notes

- Skills are playbooks, not code. Write them for an LLM to follow, not for a human to read.
  Be direct. Use numbered steps. Avoid prose where a list will do.
- If a skill grows too large to follow linearly, split it: one `SKILL.md` as the router,
  and one file per path under `references/`.
- Skills must not contain implementation code. If a skill needs to invoke a tool or run
  a command, it states what to do — the agent decides how.
