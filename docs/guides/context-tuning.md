# Context Tuning

Tune `oz` routing and retrieval behavior with `context/scoring.toml` using `oz context scoring`.

## Goal

Adjust routing confidence and retrieval relevance in a controlled, testable way without hand-editing TOML blindly.

## Preconditions

- Existing `oz` workspace with `context/graph.json` available.
- You can run representative routing queries from your team.

## Steps

1. Inspect current effective scoring values.

```bash
oz context scoring show
```

1. Discover available keys and meaning.

```bash
oz context scoring list
oz context scoring describe routing.confidence_threshold
oz context describe retrieval.min_relevance
```

1. Change one key at a time and validate file correctness.

```bash
oz context scoring set routing.confidence_threshold 0.62
oz context scoring validate
```

1. Evaluate behavior on representative tasks.

```bash
oz context query "implement a new oz subcommand"
oz context query "update AGENTS.md routing hints"
```

1. If needed, inspect debug routing/retrieval math for one query.

```bash
oz context query --raw "update AGENTS.md routing hints"
```

1. For semantic workflows, propose one concept incrementally instead of full re-enrichment.

```bash
oz context concept add --name "Workspace health checks"
oz context review
```

## Verify

- `oz context scoring validate` reports success.
- Query outcomes become more consistent with your intended agent routing.
- Retrieval includes relevant context blocks above your configured relevance floor.
- New concept proposals are reviewed before being treated as accepted context.

## Common pitfalls

- Changing multiple keys at once and losing the causal link to behavior changes.
- Tuning scoring before rebuilding `context/graph.json` after structural edits.
- Using only synthetic queries instead of real recurring tasks from your workflow.
