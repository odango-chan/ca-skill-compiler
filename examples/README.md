# Examples and fixtures

The `examples/` directory contains historical migration inputs and synthetic
fixtures from earlier Compiler iterations. It is **not** the current positive
Build sample set.

## Current positive samples

- [`web/parity-success`](../web/parity-success) — an ordinary (Wait-free) Skill
  project exercised by the CLI and Native/WASM/Chromium parity acceptance tests.
- [`web/parity-v1alpha5/`](../web/parity-v1alpha5) — a synthetic trusted
  BuildRequest, Procedure fixtures and the invalid Wait fixture used by the
  v1alpha5 acceptance tests.

For broader compiler tests, use the fixtures referenced by the acceptance
matrix and `docs/workstreams/wp7-conformance-v1alpha5.json`.

## Historical examples

| Path | Classification | Guidance |
|---|---|---|
| `ca-agent-demo-project/` | Pinned ca-agent-demo source snapshot and migration input | Preserve as source-history evidence. It is not a current v1alpha5 Build fixture. |
| `ca-skills-v0.2/` | Historical official Skill Language v0.2 snapshot | Preserve for migration and source provenance. v0.2 is not the current language authority. |
| `minimal-project/` | Synthetic pre-v1alpha4 Compiler fixture | Retained for historical reference only. |
| `script-contract-project/` | Synthetic pre-v1alpha4 Script Contract fixture | Retained for historical reference; its v0 Script contract is not the current Script ABI. |

These directories keep their existing paths to preserve source provenance and
avoid silently breaking historical links. They must not be presented as current
examples or as evidence that the Compiler accepts legacy syntax.

## Version boundary

- Current language target: `connectionagent.io/v1alpha5` (only).
- The Compiler accepts no v0.1/v0.2 or v1alpha4 compatibility, translation or
  fallback.
- Compiler product version v2.0.0 does not execute Python and does not own
  Runtime event/Wait behaviour.
