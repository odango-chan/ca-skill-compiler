# Archived Harness rules — CA Skill Compiler v0.1

**ARCHIVED / NOT ACTIVE.** Retained verbatim for project history only. The active instructions are in repository-root `AGENTS.md` and target CA-SL v1alpha4.

---

# ca-skill-compiler · Harness rules

This is a clean, stateless Compiler repo. The active implementation is **breaking-only v0.1**, with no legacy compatibility layer.

Read and reconcile authoritative sources in this order:
1. Pinned `ca-skills` Skill Language v0.2, Script Contract, Schema Ownership and Authoring Model; the exact commit is recorded in `docs/workstreams/language-alignment-v0.1.md`.
2. [Compiler v0.1 Language & Linking Contract](docs/design/compiler-contract-v0.1.md) and its machine schemas.
3. [Strict Script Output Contract](docs/design/script-contract-v0.md) and [Procedure Link baseline](docs/design/procedure-link-v0.md), as reconciled with the pinned ca-skills contract.
4. [v0.1 Language Alignment](docs/workstreams/language-alignment-v0.1.md), Studio ABI and current acceptance/work status.

The Compiler follows ca-skills v0.2: Procedure Script input comes from call-site `with`; Predicate Script input is the Host-created Predicate Evaluation Context; Script output is a strict nonempty `TypedDict`. No legacy metadata, `localTools`, manifests or guessed schemas are valid fallbacks. Keep missing authoritative Owner inputs visible as diagnostics.

Implementation must be a pure Go, in-memory, stateless Core: input project files -> static parse/link/diagnostics -> output files. No Python execution, MCP call, live 5GC, SQLite, Release, Qualification, Runtime Admission or HTTP server. Native and WebAssembly share exactly this Core; Studio owns UI and ZIP.

**External Producer owns Input Schema**: import real Tool/Host/Observation contracts as `lib/**` bytes; bind via Procedure `with`. Script declares **only its newly computed Output type** as a strict nonempty TypedDict (`main(input: dict) -> Output`). No repeated Script Input TypedDict, no old script-header schemaMode/external fallback, no acceptance via guessed object fields. Missing authoritative external contract must diagnose.

Pinned `examples/ca-agent-demo-project` is a **pre-migration reference**; preserve its source/catalog originals unmodified. Current official Skill sources are pinned under `examples/ca-skills-v0.2` from the ca-skills language-owner branch. `ca-agent-demo` is a behavior/ToolCatalog reference, not a Skill language source tree. The approved MAS ToolCatalog is unavailable; do not import its draft.

Work on a new branch based on the current v0.1 design PR branch and open a dependent implementation PR targeting it (retarget only after design PR merges). Run Go tests, native/WASM and real Chromium tests where available; report PASS/FAIL/BLOCKED/NOT_RUN honestly. Do not modify other repositories, force-push or merge into main.
