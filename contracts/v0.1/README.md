# Skill Compiler v0.1 machine contracts

**Compiler-only contract.** No project settings, dependency resolver, package manager, source acquisition, lockfile, snapshot store or cache API is defined here.

- [build-request.schema.json](build-request.schema.json) — in-memory input: `files` and optional `targets`.
- [build-result.schema.json](build-result.schema.json) — deterministic diagnostics and generated file tree.
- [tool-catalog.schema.json](tool-catalog.schema.json) — existing imported ToolCatalog YAML data model (`ca-agent-demo.tool/v1`).
- [library-inspection.schema.json](library-inspection.schema.json) — read-only ToolCatalog inspection result.

Normative responsibilities and boundaries: [Skill Compiler Language & Linking Contract v0.1](../../docs/design/compiler-contract-v0.1.md).

The `lib/**` prefix is a **logical Compiler input namespace**. Studio/callers may assemble these files in memory from any source; no physical `lib/` workspace folder is required. `outputFiles` are BuildResult values; no physical `output/` folder is required.

The current Go/WASM v0 implementation still exposes `{files,targets}`; this v0.1 contract freezes that small input shape and clarifies its semantics. This PR is design-only and does not claim the implementation has been migrated or fully accepted.
