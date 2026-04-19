# Reference: Create or Tweak a Rule

Use this reference when creating a new rule file under `rules/` or modifying an existing one.

---

## What rules are

Rules are hard behavioral constraints for agents. An agent reads its rule files and must follow
them without exception — they are not suggestions or reading material. This is what separates
the `## Rules` section from the `## Read-chain` in an `AGENT.md`.

Rules live in `rules/` at the workspace root. They are shared across agents, but each agent
explicitly declares which rule files it follows in its `AGENT.md`.

---

## Steps

### Creating a new rule file

1. Decide the rule filename. Use lowercase kebab-case (e.g. `coding-guidelines.md`).
2. Copy `assets/rule.md.tmpl` to `rules/<name>.md`.
3. Write the rule content. Rules must be:
   - **Specific** — no vague guidance. Every statement must be actionable.
   - **Unambiguous** — if an LLM could interpret it two ways, rewrite it.
   - **Durable** — rules should not need frequent updates. If something changes often,
     it belongs in `docs/` or `context/`, not `rules/`.
4. Declare the rule in every agent that must follow it. In each relevant `AGENT.md`,
   add a bullet under `## Rules`:
   `- \`rules/<name>.md\` — <one-line summary of what this rule enforces>`
5. Verify: re-read the rule and confirm every statement passes the specificity test above.

### Tweaking an existing rule

1. Open `rules/<name>.md`.
2. Make the targeted change.
3. Confirm the rule remains specific, unambiguous, and durable after the edit.
4. If the change affects which agents the rule applies to, update the relevant `AGENT.md`
   `## Rules` sections accordingly.
5. Verify: re-read the full rule file and confirm no statements became vague or contradictory.

---

## Notes

- Do not put process descriptions or background context in rule files. Rules state constraints
  only. Background belongs in `docs/` or `specs/`.
- If a rule applies to all agents, consider whether it should be part of the workspace
  convention itself (a spec concern) rather than a standalone rule file.
- Rules are read at session start. Keep them concise — an LLM loading a 2,000-word rule
  file may miss the key constraints buried inside it.
