# Ops and Recovery

Run a lightweight day-2 operations loop for workspace health and recover missing defaults safely.

## Goal

Keep the workspace healthy during normal development and quickly recover from accidental deletions or merge damage without overwriting real work.

## Preconditions

- Existing `oz` workspace.
- `oz` is installed and available in your `PATH`.

## Steps

1. Run baseline health checks.

```bash
oz validate
oz audit
```

1. Rebuild structural context after meaningful `.md` or indexed code changes.

```bash
oz context build
```

1. If required default files are missing, repair safely.

```bash
oz repair
```

1. Re-run health checks after repair to confirm the workspace is consistent.

```bash
oz validate
oz audit staleness
```

## Verify

- `oz validate` exits successfully.
- `oz audit` has no unexpected errors for your policy.
- `context/graph.json` exists and reflects recent structure updates.
- `oz repair` only creates missing files and does not overwrite existing ones.

## Common pitfalls

- Running `oz repair` before confirming you are in the intended workspace root.
- Treating `oz audit` warnings as always actionable rather than triaging accepted noise.
- Forgetting to rebuild context after docs/agent/spec changes, then debugging stale query output.
