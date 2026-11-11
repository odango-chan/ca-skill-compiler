# ca-skill-compiler · Harness rules

This is a clean, stateless Compiler repo; **breaking-only v0, NO legacy compatibility layer.**

Read authoritative specs in this order:
1. [Script Contract](docs/design/script-contract-v0.md)
2. [Procedure & Binding](docs/design/procedure-link-v0.md)
3. [Project Compiler Design](docs/design/skill-compiler-v0.md)
4. [Acceptance Matrix](docs/workstreams/acceptance-matrix-v0.md)
5. [Implementation Goal](docs/workstreams/local-implementation-goal.md)
6. [Studio/WASM ABI](docs/studio-integration.md)
7. [WORK status](docs/WORK.md)

Implementation must be a pure Go, in-memory, stateless Core: input project files -> static parse/link/diagnostics -> output files. No Python execution, MCP call, live 5GC, SQLite, Release, Qualification, Runtime Admission or HTTP server. Native and WebAssembly share exactly this Core; Studio owns UI and ZIP.

**External Producer owns Input Schema**: import real Tool/Host/Observation contracts in lib/, bind via Procedure with:. Script declares **only its newly computed Output type** as a strict nonempty TypedDict (main(input: dict) -> Output). No repeated Script Input TypedDict, no old script-header schemaMode/external fallback, no acceptance via guessed object fields. Missing authoritative external contract must diagnose.

Pinned `examples/ca-agent-demo-project` is a **pre-migration reference**: 79 source Skill blobs + one ToolCatalog match upstream at `8ab5145b992f74ee793e0d35a4ec96e6b26500a3`; MAS candidate lib is a separate matching copy. Preserve these originals unmodified. Raw old scripts do not need to compile under v0; their migration errors are evidence. If migrating examples, do so in a separate v0 directory. MAS Tools are PROPOSED/NOT_READY.

Work on a new branch based on latest origin/feat/compiler-v0-bootstrap and open PR targeting that branch. Run real Go tests, native/WASM and browser tests where available; report PASS/FAIL/BLOCKED/NOT_RUN honestly. Do not modify other repositories, force-push or merge failing code into main.
