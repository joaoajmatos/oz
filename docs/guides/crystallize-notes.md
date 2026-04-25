# Crystallize Notes

Use `oz crystallize` to classify note files and review promotion-ready diffs before manual promotion.

## Goal

Turn low-trust planning notes into actionable promotion candidates for canonical layers (`specs/`, `docs/`, ADRs) without unsafe automated writes.

## Preconditions

- Existing `oz` workspace with markdown files under `notes/`.
- You understand where canonical content belongs (`specs/` vs `docs/` vs ADRs).

## Steps

1. Run report-only classification first.

```bash
oz crystallize
```

1. If needed, narrow by topic.

```bash
oz crystallize --topic routing
```

1. Inspect proposed diffs without writing files.

```bash
oz crystallize --dry-run
```

1. If LLM classification is not desired, run heuristic-only mode.

```bash
oz crystallize --no-enrich --dry-run
```

1. Manually promote selected content into canonical targets, then refresh context.

```bash
oz context build
oz validate
```

## Verify

- Classification table identifies plausible targets for candidate notes.
- Dry-run diff previews are sufficient for manual promotion decisions.
- Promoted content now lives in `specs/`, `docs/`, or `specs/decisions/` as intended.
- `oz validate` passes after promotion.

## Common pitfalls

- Treating dry-run output as an automatic write path.
- Promoting too much historical note content at once.
- Skipping `oz context build` after promotion and then querying stale graph state.
