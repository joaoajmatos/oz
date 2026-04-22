# Adopt oz in an Existing Repository

Introduce `oz` incrementally in a repo that already has code, docs, and workflows.

## Goal

Add `oz` convention files and pass validation without disrupting existing delivery flow.

## Preconditions

- Existing repository with active development.
- Agreement on initial agent routing boundaries.

## Steps

1. Initialize `oz` at repo root and choose a code layout that matches your structure.
2. Keep your existing code organization; use agent scopes to map current ownership.
3. Move or author core decision docs under `specs/` and architecture/how-to docs under `docs/`.
4. Run convention checks and fix critical findings.

```bash
oz validate
oz context build
oz audit
```

5. Introduce hooks/integrations once baseline health is stable.

```bash
oz add cursor
oz add claude
```

## Verify

- `oz validate` passes.
- `oz audit` has no unexpected errors.
- `context/graph.json` reflects current docs/agents/code references.

## Common pitfalls

- Trying to migrate all historical documentation at once.
- Defining agent boundaries that overlap heavily without clear ownership language.
- Enabling strict checks in CI before the repository reaches a stable baseline.
