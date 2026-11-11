# Local Harness Goal · Finish Skill Compiler v0 (one-shot)

- **Status**: implementation Goal; not an assertion of completion
- **Owner repository**: `odango-chan/ca-skill-compiler`
- **Start point**: `origin/feat/compiler-v0-bootstrap`, the head of [Draft PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2), **not `main`**
- **Issue**: [#1](https://github.com/odango-chan/ca-skill-compiler/issues/1)
- **Target API**: [Studio integration](../studio-integration.md)
- **Accepted scope**: [Compiler v0 design](../design/skill-compiler-v0.md)
- **Real sample**: [ca-agent-demo-project](../../examples/ca-agent-demo-project/README.md)

## 0. Non-negotiable boundaries

This project is a stateless, GCC-like compiler, **not** a platform. Keep these out of the new repository: SQLite/DB, HTTP service, Qualification, Release, Registry server, managed Workspace, Scenario/Planning/Mock runner, Runtime execution/Admission, mandatory digest/identity release lifecycle, tool implementation or MCP server invocation. No copying whole legacy Toolchain packages. Do not edit `ca-agent-demo`, `ca-runtime`, `ca-skill-studio-demo` or `connectionagent` in this Goal.

Authoring project consists of three logical roots: editable `skills/` (multiple Skills), imported **read-only** `lib/` and generated `output/`. Compiler only transforms input byte maps to diagnostics/output byte maps. CLI owns filesystem access, Studio owns editor, export ZIP and file-save permissions. Native and WASM MUST use the **same** Go compiler Core.

No unattended modifications to Tool contract truths: an imported Tool contract describes an interface, not installation, READY, authorization or successful execution. Never execute Python/Tool/MCP operations during compilation.

## 1. Current baseline (audit before coding)

- [Draft PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2) includes initial `internal/compiler`, native `cmd/skillc`, `cmd/skillc-wasm`, Worker bridge, tests and docs. **Static code only: Go test/build and Chrome/WASM acceptance are NOT_RUN**.
- Early Core's `skills.go` scans generic keys and may report **spurious `PROCEDURE_NOT_FOUND`** for a cross-Skill node `procedure: {skill: child, procedure: run}`: the recursive walker sees the nested `procedure: run` again as a local call. Fix by typed/position-aware traversal, not by hiding the diagnostic.
- `go.mod` currently references `gopkg.in/yaml.v3`; check whether `go.sum` is absent. Run `go mod tidy` in a network-enabled environment, commit lockfile. No hand-rolled YAML parser.
- `Check/Build` currently do not provide a Library inspection API, comprehensive Schema validation, Script I/O generation, exact Procedure resolution, true target closure semantics or real consumer validation. Do not describe them as implemented.
- Synthetic `examples/minimal-project` is NOT the real example.
- **Real sample NOW PRESENT** in `examples/ca-agent-demo-project`: exact 2 Skill directory snapshots (79 files) + one Agent ToolCatalog file; source commit `8ab5145b992f74ee793e0d35a4ec96e6b26500a3`. All 80 original file Git blob IDs matched byte-for-byte against the copied paths. Another identical copy of the MAS proposed ToolCatalog appears in `lib/tools/mas-proposed.tool.yaml`. Source snapshots are intentionally not edited for passing tests.
- MAS lib is from `establish-pdu-session-mas/design/tool-contract.yaml` and is explicitly **NOT_READY** (external adapters, independent Effects, reconcile and resource owner proofs unimplemented). Its 16 Operation declarations support static-link experiments only. Conventional Agent ToolCatalog declares 12 operations and is not an MCP server.
- `ca-agent-demo/tools/registrations/default.yaml` is deployment Operation → Implementation binding, **not a Tool Schema library**.

## 2. Execution sequence (finish in order; do not stop at a design-only handoff)

### WP0 · Make existing Go bootstrap compile and tests run

- Start from exact branch above; inspect the existing code and `docs/WORK.md`, reproduce failures.
- `go mod tidy`; `gofmt -w` Go source; `go test ./...`; `go vet ./...`.
- Correct Go syntax/runtime issues, parser panics, and tests. Add tests first for repros.
- **Known P0**: make synthetic cross-Skill procedure test PASS by fixing the recursive `procedure` token misclassification. Parse AST/nodes by meaningful contexts.
- **Known P0**: ensure duplicate YAML keys, malformed aliases, oversized inputs, duplicate Tool IDs, path traversal, parent-path escape and bad data never crash/panic or silently choose one source.
- Avoid migrating old platform's parser wholesale.
- Goal exits only with real local logs of tests/build, not a claim inferred from code review.

### WP1 · Implement static Skill source analysis and dependency resolution

- Read all Skill roots under `skills/<name>/`. Derive identity from real manifest/contract; match authored name to directory. Validate entry Procedure exists.
- Analyze Procedure graph: step/node references, Script path refs, tool `call` refs, local child references, explicit cross-Skill refs, required Predicate refs. Avoid generic text search and accidental nested-key handling.
- Build a deterministic Skill-to-Skill graph; detect missing dependency, duplicate/ambiguous identity and cycles explicitly. With `targets=[]` build whole project; with a target, build its **transitive closure only**. Invalid unrelated Skill must not block a valid selected target unless it changes shared/global namespace or input safety.
- Preserve original authored Skill files; no silent deletion, rewriting or weakening of unknown construct. Unsupported features must produce clear warning/error where they impact semantics.
- No run-time graph execution, no business-specific `CreatedSession` cases.

### WP2 · Library import and inspection API

- Read `lib/tools/*.yaml` as **ToolCatalog**. Support the real `createdsession.tool.yaml` including YAML anchor/alias. Read `mas-proposed.tool.yaml` as a design-state fixture with explicit NOT_READY info; do not turn it into a validated runtime Tool.
- Validate Operation IDs, declared Effect, input/output Schema presence/shape, source location and duplicate/conflicting IDs; return precise diagnostics, no auto-winner.
- Add a read-only Tool Library projection/API for Skill Studio: name/ID, description (if present), I/O Schema, optional reconcile Schema, declared Effect + origin, and status/unknowns. Do not make Studio implement a second catalog parser.
- Make a **real input fixture test**: 12 conventional and 16 MAS declared Operations are discoverable. The effect contract is only a claim from the imported file; no READY test.
- MCP `tools/list` JSON import is an optional later adapter; do not connect to live MCP or add a network dependency to compiler Core.

### WP3 · Strict Script Contract v0 (normative)

Read [Script Contract v0](../design/script-contract-v0.md) first. **Do not implement the superseded long YAML comment `schemaMode: embedded` design.** The single recommended new-script mode is type-first: Python standard-library `TypedDict` + `main(input: In) -> Out` + `Annotated` static JSON Schema constraints. Input/output types and all nested object fields must be explicit; reject `dict`, `dict[str,T]`, `Any`, `object`, empty `TypedDict`, unsupported unions, unknown metadata and untyped values, never degrade to loose object.

- Implement a bounded **static** parser for the supported annotation syntax; no Python import/execution, cgo-only dependency, remote ref resolution or unconstrained default. Use `Annotated[T, {"minLength":...}]` and a documented whitelist, preserving all constraints; `NotRequired` controls optional keys. Make each `TypedDict` a closed object in the emitted Schema as specified.
- Legacy mode: existing `# /// connectionagent-tool` header with external `inputSchema/outputSchema` continues to use the existing author-owned YAML Schema. **All five real scripts must remain byte-identical**. Do not infer missing types from their empty dict subclasses, or rewrite/remove complex `allOf/if/then` constraints.
- Reject typed/external double authority, unsupported annotation constructs and superseded `schemaMode: embedded`; include precise file/line diagnostics.
- Output generated schemas under `output/skills/<skill>/schemas/generated/<script-relative>.input.schema.json` and matching output; never write back into source. Do not let Studio implement separate type inference.
- Validate that a proposed future migration from an existing complex external schema doesn't lose fields or constraints. When strict typed subset cannot express it, keep the external Schema instead of weakening semantics.
- Add all ST-01..ST-11 cases from the new canonical Script Contract; native/WASM bytes and diagnostics must match. Runtime/Host still owns actual execution, validation and sandboxing.

### WP4 · Deterministic package and Native CLI

- Build selected Skills + transitive dependent Skills, necessary source files, generated schemas and used Tool library snapshots; preserve exact relative paths for copy/ZIP.
- Output files sorted, deep byte-copy, no timestamps/run IDs/forced SHA digest. Output cannot become Source on a subsequent build.
- Native CLI `skillc check PROJECT`, `skillc build PROJECT --out DIR`, optional `--skill NAME`; return nonzero for error.
- Never follow symlinks or overwrite Source/Lib; reject unsafe output paths and stale nonempty output. Tests for both Windows-style and POSIX path traversal.
- Keep ZIP creation in Studio, not the compiler Core.

### WP5 · WASM + Pages + Studio handoff

- Build `GOOS=js GOARCH=wasm` using same Core. Provide minimal Worker bridge (JSON base64 `[]byte` encoding) and a browser demo that actually invokes it, if useful.
- Test same payload and verify diagnostics/output paths/bytes equal native Core (not just “WASM starts”).
- Verify on real Chromium static-site serving (Pages-like), no server API/database, no remote tool execution, no user file upload.
- Document observed size/time and limits, no need to build a custom UI. Keep [Studio integration doc](../studio-integration.md) up to date.

### WP6 · Real fixtures & compatibility acceptance

Tests must use these **unchanged** checked-in source snapshots:
- `examples/ca-agent-demo-project/skills/establish-pdu-session/`
- `examples/ca-agent-demo-project/skills/establish-pdu-session-mas/`
- `examples/ca-agent-demo-project/lib/tools/createdsession.tool.yaml`
- `examples/ca-agent-demo-project/lib/tools/mas-proposed.tool.yaml`

Run whole-project and both individual targets. Record actual diagnostics by code + source line. Fix compiler mistakes in compiler, not by modifying the imported Skill files or by weakening checks. If the MAS sample cannot pass because its Tool contracts are genuinely incomplete, report **expected NOT_READY / static-only**; do not synthesize production implementations. Keep a clear distinction between:
1. \`Build.success\`: static source, required files, references and link checks accepted;
2. \`READY\`: separately determined by Runtime/Host and real Tool owners (not Compiler).

Inspect ca-agent-demo's actual Skill loader/admission input requirements **read-only**, document whether copying `output/skills/*` alone works, or needs Host-local configuration. Do not modify that external owner in this workstream.

### WP7 · Final verification, documentation and PR flow

Run locally and record actual outputs:
~~~bash
go mod tidy
gofmt -w ./cmd ./internal
go test ./... -count=1
go vet ./...
go build ./cmd/skillc
GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm
go run ./cmd/skillc check ./examples/minimal-project
go run ./cmd/skillc check ./examples/ca-agent-demo-project
go run ./cmd/skillc build ./examples/ca-agent-demo-project --skill establish-pdu-session --out ./dist/pdu
go run ./cmd/skillc build ./examples/ca-agent-demo-project --skill establish-pdu-session-mas --out ./dist/mas
git diff --check
~~~
Use a fresh `dist/` for every run; adapt shell paths as necessary, keep generated binaries ignored. A test failure must remain FAIL, not “blocked” unless environment truly prevents running it. Use source-backed diagnostic expectations for MAS / incomplete external contracts.

Document precisely what passed, what did not run, and missing consumer/runtime proof in `docs/WORK.md` and a small acceptance report; do not declare Host/5GC E2E PASS.

Create a new implementation branch from `origin/feat/compiler-v0-bootstrap`, commit and push changes, and open a PR **targeting `feat/compiler-v0-bootstrap`**. Do NOT target old Toolchain and do not force-push. Only after local checks and review of that PR should initial [Draft PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2) be marked ready and considered for merge to `main`. Do not merge to `main` with open real compiler failures. Do not create GitHub Actions just to get verification.

## 3. Handoff / output requirements

Return in the same local Harness execution:
1. Implementation branch, HEAD SHA, PR URL;
2. completed WP0–WP7 matrix with IMPLEMENTED/PARTIAL/NOT_STARTED and PASS/FAIL/BLOCKED/NOT_RUN separately;
3. precise static results for the traditional and MAS Skills, including unmet Tool contracts;
4. native CLI and WASM build verification, and browser/Chrome check if performed;
5. exact independent consumer questions that remain, without expanding into other repositories;
6. final diff summary (do not edit source snapshots without explicit justification).

**This is a one-shot implementation Goal, not a request for another report or handoff.** Work through stages sequentially, run tests, fix errors, commit/push and create PR; do not stop after merely designing the solution. If a step truly cannot be completed, leave a precise error, tests and remaining work, then complete all nonblocked steps.
