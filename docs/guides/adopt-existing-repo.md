# Adopt oz in an Existing Repository

Introduce `oz` incrementally in a repo that already has code, docs, and workflows.

## Goal

Add `oz` convention files and pass validation without disrupting existing delivery flow, while giving the team one shared workspace environment with consistent rules.

## Preconditions

- Existing repository with active development.
- Agreement on initial agent routing boundaries.

## Steps

1. Initialize `oz` at repo root and choose a code layout that matches your structure.
2. Choose your adoption mode:
   - keep code in this repository, or
   - use this workspace as a meta repo and attach existing code repositories as git submodules under `code/`.
3. Keep your existing code organization; use agent scopes to map current ownership.
4. Move or author core decision docs under `specs/` and architecture/how-to docs under `docs/`.
5. Run convention checks and fix critical findings.

```bash
oz validate
oz context build
oz audit
```

1. Introduce hooks/integrations once baseline health is stable.

```bash
oz add cursor
oz add claude
```

## Verify

- `oz validate` passes.
- `oz audit` has no unexpected errors.
- `context/graph.json` reflects current docs/agents/code references.
- If using submodules, each code repository remains independently versioned while `oz` manages shared workspace conventions at the root.

## Common pitfalls

- Trying to migrate all historical documentation at once.
- Defining agent boundaries that overlap heavily without clear ownership language.
- Enabling strict checks in CI before the repository reaches a stable baseline.
