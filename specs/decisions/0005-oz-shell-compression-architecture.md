# ADR-0005: oz shell compression architecture

Date: 2026-04-25
Status: Accepted

## Context

`oz` currently optimizes context selection and retrieval (`oz context query`) but does not
optimize verbose shell output that LLM agents consume during execution-heavy workflows.

The project needs a shell-output compression layer that reduces token usage while preserving
correctness, determinism, and CI-safe behavior.

Two high-level decisions were required:

1. **Execution/filter engine implementation**:
   - pure Go inside `code/oz`
   - external dependency or sidecar runtime
2. **Interception model**:
   - explicit wrapper only
   - transparent rewrite only
   - both explicit and optional transparent rewrite

## Decision

We will implement shell compression as a **pure Go subsystem inside `oz`** and ship **both**
interaction modes in v1:

- explicit mode: `oz shell run -- <cmd...>` (always available)
- optional transparent interception: hook rewrite to `oz shell run -- ...`

Normative behavior is defined in:

- `specs/oz-shell-compression-specification.md`

Key guarantees:

- exit code preservation
- deterministic compaction output
- fail-open interception behavior
- fallback to raw output when filtering fails

## Consequences

### Positive

- Aligns with existing single-binary Go architecture.
- Avoids cross-language runtime/dependency complexity in v1.
- Keeps deterministic behavior and testability within the existing Go toolchain.
- Supports progressive adoption: explicit mode first, transparent mode opt-in.

### Negative

- Re-implementing mature filter taxonomy in Go requires engineering effort.
- Hook integrations across agent ecosystems add maintenance surface.
- Initial command coverage is narrower than mature external tools.

## Alternatives considered

### 1. Vendor/adapt external Rust implementation

Rejected for v1 because:

- introduces dual-runtime complexity and integration risk
- complicates build/release pipeline and contributor ergonomics
- weakens single-binary Go constraint

### 2. External optional binary adapter only

Rejected for v1 as primary path because:

- creates uneven behavior across environments
- complicates support and deterministic contracts
- adds user setup burden for core feature value

### 3. Transparent interception only

Rejected because:

- no guaranteed baseline mode when hooks are unavailable
- harder debugging and rollout safety

### 4. Explicit wrapper only

Rejected because:

- lower adoption and weaker token impact without transparent mode option
