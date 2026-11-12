# Skill Compiler v0.1 machine contracts

**Compiler-only contract.** No source acquisition, dependency resolver lifecycle, lock management, snapshot store, cache, or package-manager API is defined here.

- [build-request.schema.json](build-request.schema.json) — in-memory input: Skill source files, resolved immutable dependency snapshots, and optional targets.
- [build-result.schema.json](build-result.schema.json) — deterministic diagnostics, used dependency closure, and generated Skill file tree.
- [tool-catalog.schema.json](tool-catalog.schema.json) — imported ToolCatalog YAML data model (`ca-agent-demo.tool/v1`).
- [dependency-inspection.schema.json](library-inspection.schema.json) — read-only projection of Operations found in supplied dependency snapshots.

Normative responsibilities and boundaries: [Skill Compiler Language & Linking Contract v0.1](../../docs/design/compiler-contract-v0.1.md).

Skill source files use the `skills/**` namespace. Resolved dependency snapshots are separate BuildRequest values with their own snapshot-relative paths, stable Workspace Configuration IDs, and verified content digests. There is no Compiler `lib/**` namespace and no physical `lib/` workspace folder requirement. `outputFiles` are Skill build outputs; `dependencyClosure` records the exact snapshots used without copying dependency bytes into the output tree. Studio decides how and where to persist or export build results and dependency snapshots.

The current Go/WASM v0 implementation still exposes the compatibility `{files,targets}` API and path-based `lib/**` behavior. This v0.1 contract is a design proposal; do not claim implementation readiness until Native Go and WASM accept the new request/result schemas, digest golden vectors pass, and the real 5GC source fixture builds without a physical `lib/` folder.
