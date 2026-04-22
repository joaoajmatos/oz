# Code

This directory is the source-code layer of the workspace.

Use this README as an index for code projects that live under `code/`.

In a meta-repo setup, each project under `code/` can be a git submodule so teams keep repositories separated while sharing one `oz` workspace for docs, rules, and agent routing.

## Projects

- `code/oz/` (Go): the `oz` CLI implementation used by this repository.

## Working in `code/`

- Each subdirectory should be a self-contained project with its own README.
- Build, test, and lint from the project directory (for example `code/oz/`).
- Keep code-level contributor guidance close to the project that owns it.
- If a project is a submodule, manage its lifecycle (`add`, `update`, `sync`) with git submodule commands from the workspace root.

## Add a new project

When adding a new code project under `code/`, include:

1. A directory entry in the table above.
2. A project README at `code/<project>/README.md`.
3. The minimum local commands (`build`, `test`, and `lint`).
