# ca-skill-compiler · Local implementation instructions

This is a new, minimal, stateless compiler repository. **Do not follow old ca-skill-toolchain/Platform AGENTS.md.**

Authority and order:
1. [Local Implementation Goal](docs/workstreams/local-implementation-goal.md)
2. [Compiler v0 design](docs/design/skill-compiler-v0.md)
3. [Script Contract v0](docs/design/script-contract-v0.md) — producer-owned external inputs resolved from lib/Procedure with; strict Script-owned computed outputs; do not duplicate Tool schemas
4. [Skill Studio ABI](docs/studio-integration.md)
5. [Status](docs/WORK.md)

Do not add database, API service, Qualification/Release, Runtime executor, Tool execution, hidden global registry or mandatory digest lifecycle. Compiler: in-memory files → parse/link/minimal checks → output files + diagnostics. Use the same Go Core for native and WASM. Scripts must not execute during build.

The real integration fixture `examples/ca-agent-demo-project` is a fixed snapshot from ca-agent-demo at `8ab5145b992f74ee793e0d35a4ec96e6b26500a3`. Traditional 12-operation ToolCatalog is an existing Agent-facing contract; MAS 16-operation catalog is explicitly PROPOSED / NOT_READY. Do not alter these test sources to hide compiler failures or claim Runtime/Tool readiness.

Start from `origin/feat/compiler-v0-bootstrap` until Draft PR #2 merges. Make new implementation branch and PR against that branch. Run real local Go tests/build and JS/WASM/Chromium verification as available. Do not claim PASS for NOT_RUN. Do not touch other repositories or real 5GC. Keep the implementation small and the future Skill Studio interface stable.
