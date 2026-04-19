# Reference: Create or Tweak an Agent

Use this reference when creating a new `agents/<name>/AGENT.md` or modifying an existing one.

---

## Required sections (in order)

Every `AGENT.md` must contain all seven sections below, in this exact order.
Use `assets/AGENT.md.tmpl` as your starting point.

| Section | What it must contain |
|---|---|
| `## Role` | One paragraph: what this agent does and its operating constraints. No bullet lists. |
| `## Read-chain` | Ordered numbered list of files to load before starting any task. Context only — not rules. |
| `## Rules` | Bullet list of rule files that govern this agent's behavior. Hard constraints. |
| `## Skills` | Bullet list of skill paths this agent is authorized to invoke (`skills/<name>/`). |
| `## Responsibilities` | Bullet list of what this agent owns and produces. |
| `## Out of scope` | Bullet list of what this agent must not do. Be explicit. |
| `## Context topics` | Bullet list of `context/` topics this agent reads when relevant. |

---

## Steps

### Creating a new agent

1. Decide the agent name. Use lowercase kebab-case (e.g. `oz-maintainer`).
2. Create the directory: `agents/<name>/`.
3. Copy `assets/AGENT.md.tmpl` to `agents/<name>/AGENT.md`.
4. Fill in all seven required sections. Do not leave placeholder text.
5. Register the agent in `AGENTS.md`:
   - Add a `### <name>` subsection under `## Agents`.
   - Include a one-line bold description and `Agent definition: agents/<name>/AGENT.md`.
6. Register the agent in `OZ.md`:
   - Add a row to the `## Registered Agents` table.
7. Verify: re-read the spec section on AGENT.md required sections and confirm compliance.

### Tweaking an existing agent

1. Open `agents/<name>/AGENT.md`.
2. Make the targeted change.
3. Confirm all seven required sections are still present and in order.
4. If the agent's role or name changed, update `AGENTS.md` and `OZ.md` to match.
5. Verify: re-read the spec section on AGENT.md required sections and confirm compliance.

---

## Notes

- The read-chain loads context. Rules enforce behavior. Keep them in separate sections — the spec
  and `oz validate` treat them differently.
- Do not add a skill to `## Skills` unless the skill directory actually exists in `skills/`.
- Context topics in `## Context topics` must map to real subdirectories under `context/`.
  If the topic does not exist yet, note it as an open item in `docs/open-items.md`.
