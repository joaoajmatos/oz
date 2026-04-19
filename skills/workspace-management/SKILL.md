---
name: workspace-management
description: Workspace management for the oz-maintainer — create or tweak an agent definition, skill, or rule in this oz workspace, following the oz convention exactly.
triggers:
  - workspace management
  - create agent
  - create skill
  - create rule
  - new agent
  - new skill
  - new rule
  - tweak agent
  - tweak skill
  - tweak rule
---

# Skill: workspace-management

> Guides the oz-maintainer through workspace management — creating or tweaking an agent definition, skill, or rule
> in this oz workspace, following the oz convention exactly.

## When to invoke

Invoke this skill when asked to:

- Create a new agent (`agents/<name>/AGENT.md`)
- Create or modify a skill (`skills/<name>/`)
- Create or modify a rule file (`rules/<name>.md`)
- Tweak any existing agent, skill, or rule of the above types

Do not invoke this skill for changes to specs, docs, code, or context — those belong to
oz-spec, oz-coding, and the relevant docs/context owners respectively.

---

## Steps

1. **Identify the artifact type.**
   Determine whether the target is an **agent**, a **skill**, or a **rule**.

2. **Read the corresponding reference.**
   Open and follow the reference for the artifact type:
   - Agent → `references/create-agent.md`
   - Skill → `references/create-skill.md`
   - Rule → `references/create-rule.md`

3. **Use the template from `assets/` as your starting point.**
   Every artifact type has a canonical template. Do not write from scratch.

4. **Follow the steps in the reference to completion.**
   The reference defines required sections, registration steps, and validation checks.
   Do not skip steps.

5. **Verify against the spec.**
   After creating the artifact, re-read the relevant section of
   `specs/oz-project-specification.md` and confirm the artifact satisfies all
   required sections and structure. If it does not, fix it before finishing.
