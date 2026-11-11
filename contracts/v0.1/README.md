# Skill Compiler v0.1 machine contracts

**Design frozen for implementation, runtime not yet migrated.** These JSON Schemas validate structural shapes (Draft 2020-12), not all semantic conditions (e.g. unique aliases across arrays, digest verification, source/lock equality, trusted Effect provenance).

- [project-settings.schema.json](project-settings.schema.json) — normalized Workspace extension settings and lock.
- [tool-contract.schema.json](tool-contract.schema.json) — tool-catalog normalization target. MCP effect always UNKNOWN/UNVERIFIED.
- [build-request.schema.json](build-request.schema.json) — host-independent native/WASM BuildRequest wire shape.

Normative behavior, hash algorithm, logical Procedure syntax, security and migration gates: [Plugin-managed contract v0.1](../../docs/design/plugin-managed-project-v0.1.md).

Do **not** pass this v0.1 request to current Compiler v0 Go/WASM; those still consume `{files,targets}` with physical `skills/` and `lib/` inputs.
