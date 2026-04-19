# Open Items

> Open questions, known issues, and pending decisions.
> Resolved items move to `specs/decisions/`.

## Open Questions

<!-- Questions that need answers before proceeding. -->

## Known Issues

<!-- Known bugs or problems, with workarounds if available. -->

## Pending Decisions

### AGENT.md required sections updated — spec needs to reflect this

The oz workspace convention now requires `Rules` and `Skills` sections in every `AGENT.md`, in addition to the existing Role, Read-chain, Responsibilities, Out of scope, and Context topics sections.

- **Rules**: lists which rule files govern this agent's behavior (separate from read-chain, which is context-only)
- **Skills**: lists which skills this agent is authorized to invoke

`oz validate` should be updated to check for these sections.
`specs/oz-project-specification.md` should be updated to include Rules and Skills in the canonical AGENT.md template.
Owner: oz-spec (spec update) + oz-coding (validate enforcement)
