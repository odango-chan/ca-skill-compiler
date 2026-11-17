# Examples and fixtures

The `examples/` directory is a collection of historical migration inputs and synthetic fixtures from earlier Compiler iterations. It is **not** the current v1alpha4 positive Build sample set.

## Current v1alpha4 positive Build sample

Use [`web/parity-success`](../web/parity-success) for the current v1alpha4 positive path. It is the fixture exercised by the CLI and Native/WASM/Chromium parity acceptance tests. It uses the current BuildRequest, dependency snapshots and trusted build context shape.

For broader compiler tests, use `tests/conformance/v1alpha4/` and the test fixtures referenced by the acceptance matrix.

## Historical examples

| Path | Classification | Guidance |
|---|---|---|
| `ca-agent-demo-project/` | Pinned ca-agent-demo source snapshot and migration input | Preserve as source-history evidence. It is not asserted to be a current v1alpha4 Build fixture. |
| `ca-skills-v0.2/` | Historical official Skill Language v0.2 snapshot | Preserve for migration and source provenance. v0.2 is not the current language authority. |
| `minimal-project/` | Synthetic pre-v1alpha4 Compiler fixture | Retained for historical reference; do not use as a v1alpha4 CLI smoke test. |
| `script-contract-project/` | Synthetic pre-v1alpha4 Script Contract fixture | Retained for historical reference; its v0 Script contract is not the v1alpha4 Script ABI. |

These directories are intentionally kept at their existing paths to preserve source provenance and avoid silently breaking historical links. They must not be presented as current examples or used as evidence that the v1alpha4 Compiler accepts legacy syntax.

## Version boundary

- Current language target: `connectionagent.io/v1alpha4`.
- Compiler v1.0.0 does not implement the latest author-level Wait semantics.
- No v0.1/v0.2 compatibility, translation or fallback is supported.
