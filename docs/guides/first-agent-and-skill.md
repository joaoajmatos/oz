# First Agent and Skill

Create one focused agent and one supporting skill for a repeated workflow.

## Goal

Add a new agent route and a reusable skill that improve task routing quality without over-engineering.

## Preconditions

- Existing `oz` workspace with baseline validation passing.
- A recurring task that deserves a dedicated playbook.
- A maintainer agent is available (`oz init` opt-in or `oz add maintainer`).

## Steps

1. Add `agents/<name>/AGENT.md` with required sections:
   - Role
   - Read-chain
   - Rules
   - Skills
   - Responsibilities
   - Out of scope
   - Context topics
2. Register the agent in both `AGENTS.md` and `OZ.md` with aligned "Use when" text.
3. Create `skills/<name>/SKILL.md` with required frontmatter and sections.
4. Add `references/` and `assets/` only if the skill has branching logic or templates.
5. Validate and rebuild context.

```bash
oz validate
oz context build
```

If your workspace does not yet include a maintainer agent, add it first:

```bash
oz add maintainer
```

## Verify

- New agent appears in `AGENTS.md` and `OZ.md`.
- `oz validate` reports no missing required sections.
- `oz context query "<representative task>"` prefers the new agent.

## Common pitfalls

- Writing a broad agent that overlaps most other agents.
- Adding a skill without clear invocation triggers.
- Forgetting to keep "Use when" wording aligned between `AGENTS.md` and `OZ.md`.
