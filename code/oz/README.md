# oz CLI (Go)

This directory contains the Go implementation of the `oz` binary.

## What lives here

- `main.go`: binary entrypoint.
- `cmd/`: Cobra commands and command wiring.
- `internal/`: implementation packages for scaffold, validate, audit, context, and related modules.
- `Makefile`: local developer shortcuts for build, test, and lint.

## Prerequisites

- Go `1.24.2` (see `go.mod`)

## Build and run

From this directory:

```bash
make build
./bin/oz --help
```

Alternative:

```bash
go build -o bin/oz .
./bin/oz --help
```

## Test and lint

```bash
make test
make lint
```

Equivalent direct commands:

```bash
go test ./...
go vet ./...
```

## Install

Local install via Go:

```bash
go install .
```

Or install using the Makefile target:

```bash
make install
```

## Command surface

The binary exposes:

- `oz init`
- `oz add`
- `oz validate`
- `oz repair`
- `oz audit`
- `oz context`

Run `./bin/oz --help` (or `oz --help` after install) for full usage.

## Related docs

- Workspace overview: `README.md` (repo root)
- Code index: `code/README.md`
- Architecture details: `docs/architecture.md`
- Normative spec: `specs/oz-project-specification.md`
