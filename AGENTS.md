# ca-skill-compiler · Harness rules

This is a clean, stateless Compiler repo. The active implementation is **breaking-only v0.1**, with no legacy compatibility layer.

Read and reconcile authoritative sources in this order:
1. Latest `ca-skills` Skill Language Specification and Authoring Model (language owner).
2. [Compiler v0.1 Language & Linking Contract](docs/design/compiler-contract-v0.1.md) and its machine schemas.
3. [Strict Script Output Contract](docs/design/script-contract-v0.md) and [Procedure Link baseline](docs/design/procedure-link-v0.md), amended by current owner-language semantics where they differ.
4. [v0.1 Language Alignment](docs/workstreams/language-alignment-v0.1.md), Compiler design, Studio ABI and acceptance/work status.

The current `ca-skills` language spec §3.8 still documents legacy Script I/O metadata; the Compiler requirement remains producer-owned input plus a strict nonempty Output `TypedDict`. Never use legacy metadata or a guessed mapping as a compatibility fallback. Record unresolved owner decisions and keep their diagnostics visible.

Implementation must be a pure Go, in-memory, stateless Core: input project files -> static parse/link/diagnostics -> output files. No Python execution, MCP call, live 5GC, SQLite, Release, Qualification, Runtime Admission or HTTP server. Native and WebAssembly share exactly this Core; Studio owns UI and ZIP.

**External Producer owns Input Schema**: import real Tool/Host/Observation contracts as `lib/**` bytes; bind via Procedure `with`. Script declares **only its newly computed Output type** as a strict nonempty TypedDict (`main(input: dict) -> Output`). No repeated Script Input TypedDict, no old script-header schemaMode/external fallback, no acceptance via guessed object fields. Missing authoritative external contract must diagnose.

Pinned `examples/ca-agent-demo-project` is a **pre-migration reference**; preserve its source/catalog originals unmodified. Current business-source authority is latest `ca-agent-demo/main`, with exact commit recorded in `docs/workstreams/language-alignment-v0.1.md`. Raw old Scripts do not need to compile; their migration errors are evidence. MAS design Tools are not an approved ToolCatalog.

Work on a new branch based on the current v0.1 design PR branch and open a dependent implementation PR targeting it (retarget only after design PR merges). Run Go tests, native/WASM and real Chromium tests where available; report PASS/FAIL/BLOCKED/NOT_RUN honestly. Do not modify other repositories, force-push or merge into main.
