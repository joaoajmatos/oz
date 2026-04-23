# Contributing to oz

## Proposition

oz is shared in the spirit of [The Unlicense](https://unlicense.org/). Use it, fork it, or ignore it, for any purpose, with no conditions from the authors and **no warranty**—express or implied. That is the posture: minimal legal ceremony, maximum clarity of intent.

If you submit a change, you are offering that change on the same terms: you dedicate it to the public domain to the extent you can, you warrant you have the right to do so, and you do not add obligations the rest of the project did not sign up for. Keep the patch honest and small; match what is already here.

## Before you open a pull request

From the repository root, keep CI expectations green:

- In **`code/oz`**: `go test ./...` and `go vet ./...` are clean.
- From the **repo root**: `oz validate` passes.

When you touch **convention** files (`OZ.md`, `AGENTS.md`, `agents/`, skills, and the like) or meaningful **`.md` / `.go` edits**, run **`oz context build`** so the workspace graph and audits stay in sync (see the project hooks if you use them).

## How to work in this tree

- The workspace standard is **[`specs/oz-project-specification.md`](./specs/oz-project-specification.md)**. Architecture and how pieces fit is in **[`docs/architecture.md`](./docs/architecture.md)** (and [`docs/implementation.md`](./docs/implementation.md) for CLI-focused detail). The **[`README.md`](./README.md)** doc table lists the rest.
- Under **`code/oz/`**, follow **[`rules/coding-guidelines.md`](./rules/coding-guidelines.md)**: change only what your fix or feature needs, don’t add abstractions you weren’t asked for, and add tests when the behavior is easy to get wrong.
- **Code wins, spec follows.** If behavior and `specs/` disagree, the implementation is authoritative until someone updates the spec; do not “paper over” with silent reverts.

## Questions and scope

- If you are changing *what the standard says* (or adding an ADR in [`specs/decisions/`](./specs/decisions/)), start from [`specs/oz-project-specification.md`](./specs/oz-project-specification.md) and open a discussion or issue for anything that will affect a lot of layout or process—those moves need consensus, not a drive-by edit.
- Gaps: [`docs/open-items.md`](./docs/open-items.md) is a fair place to note follow-ups you cannot finish in one pass.

Issues and pull requests are welcome. Favor direct communication and small, reviewable changes over process for its own sake.
