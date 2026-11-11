# Local Harness Goal · Implement Skill Compiler v0 (BREAKING ONLY)

- Repo: `odango-chan/ca-skill-compiler`
- Start from the **latest** `origin/feat/compiler-v0-bootstrap` (Draft [PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2)), not the bare `main` and not the old `ca-skill-toolchain`.
- Issue: [#1](https://github.com/odango-chan/ca-skill-compiler/issues/1)
- Status at handoff: design frozen, prototype exists, **compiler Go/WASM and browser checks NOT_RUN**.
- Goal: implement one clear v0 contract, test it, push a scoped implementation PR back to the bootstrap branch. **No legacy compatibility adapters.**

## 0. Read in this order

1. [AGENTS.md](../../AGENTS.md)
2. [Skill Compiler v0 overall design](../design/skill-compiler-v0.md)
3. [Script Contract v0](../design/script-contract-v0.md) — **producer-owned input / strict self-computed output**.
4. [Procedure / Binding Link Contract v0](../design/procedure-link-v0.md) — grammar and source scopes.
5. [Single acceptance matrix](acceptance-matrix-v0.md) — positive/negative gates.
6. [Studio API / WASM wire](../studio-integration.md)
7. [Current WORK status](../WORK.md)
8. [Real upstream migration fixture](../../examples/ca-agent-demo-project/README.md)

If old comments and examples disagree with these documents, **these v0 documents take precedence**. Document the conflict and correct the implementation/old test, not the contract.

## 1. Invariants

- **Breaking-only**: NO support for legacy `# /// connectionagent-tool` as an alternative Script input/output Schema source; NO `schemaMode: external/embedded` switch; NO old Release/Qualification/DB/API or legacy SkillBuildArtifact.
- Project layout: `skills/` editable authored source, `lib/` imported read-only snapshots, `output/` derived artifact. WASM and Native use the exact same Go Core.
- **External input is imported, not retyped.** Tool inputs/outputs come from `lib/tools/`; Host/Observation facts from real Owner-provided library files; Skill-defined public request from explicit Skill contract. Procedure `with:` derives Script per-call input projections.
- A Script's own *new* Output Schema is generated from fully declared `TypedDict / Literal / Annotated / NotRequired`; Python `main(input: dict) -> OutputType` is allowed, but input must resolve to exact Producer fields. **No output `dict/Any/object`**, no field guessing or unsupported-annotation downgrade.
- Compilation **never runs** Python, MCP, Tool implementation, Host, network, CEL evaluator or 5GC; declaration of a Tool does not prove readiness. No new dependencies on a server, DB, Node, Pydantic or live MCP.
- Unresolved owner, unknown field, ambiguous operation, invalid source syntax, output collision, unsupported runtime-dependent binding => **ERROR**, not a passing build.
- Paths/size/alias protection, reproducible output bytes, negative tests and accurate source locations are essential and should not be called a Qualification product.

## 2. What is already there (fix rather than assuming pass)

- `internal/compiler` implements an **incomplete** file-map core; `cmd/skillc`, `cmd/skillc-wasm` and `web/skillc-worker.js` are unverified scaffolding.
- `go.mod` references `gopkg.in/yaml.v3`, check whether `go.sum` needs generation. Run real tests; do not claim PASS from source inspection.
- Existing recursive YAML `inspectMapping` treats nested `procedure: run` as a local Procedure reference; fix with typed node AST traversal.
- Existing `validPath` rejects valid two-segment `lib/manifest.yaml`; fix while keeping unsafe-path rejection.
- Existing `parseSkills` / `validateSkills` and unit tests do not validate dataflow/source schemas, nested steps, predicate references, generated output contracts, real local / WASM parity.
- `lib/manifest.yaml` in the real fixture labels imported contract maturity `declared` / `proposed`. Both are **interface only**; neither equals Runtime READY. If absent, default status `unspecified`.
- Frozen upstream source fixture: `examples/ca-agent-demo-project/` contains both original PDU Session Skills, 79 unaltered Skill files plus real conventional Tool Catalog; MAS lib is another faithful copy. This is a **pre-migration reference**, NOT a build-PASS expected fixture.
- Missing authoritative Host context/Observation schemas are **real external dependencies**. Do not import `root-routing-profile-v0.5.schema.json` as a substitute; that is a deployment settings Schema.

## 3. WP0 — compile, test, pin baseline

- Fetch latest branch, create an implementation branch `feat/v0-implementation-<topic>`. Record HEAD and actual current errors.
- Run: `go mod tidy`, `gofmt -w` on actual Go files, `go test ./... -count=1`, `go vet ./...`, then `go build -o ./dist/skillc ./cmd/skillc`.
- Repair initial syntax/runtime issues and write regression tests for `lib/manifest.yaml`, YAML duplicate keys/aliases, bad input and cross-Skill nested refs.
- Commit reproducible Go module lockfile and update status with actual PASS/FAIL; no invented logs.

## 4. WP1 — canonical Procedure AST + Skill graph

- Parse `procedures:<name>: {inputs,start,nodes}` and the exact v0 action grammar in [Procedure Link v0](../design/procedure-link-v0.md). Reject unknown semantic constructs with `PROCEDURE_UNSUPPORTED` instead of silently ignoring.
- Validate entry Procedure, start node, node outcome targets, nested pre/postconditions step names, Script references, Predicate IDs, local/cross-Skill Procedure names and reconcile refs.
- Implement proper lexical binding scopes; never string-search keys to infer a Procedure action. Test real historic references as **migration samples**, without accepting every old construct.
- Target selection builds transitive Skill dependencies only. Missing/cyclic/ambiguous identity must fail; one bad unrelated Skill must not block a targeted root except for shared/global namespace problems.

## 5. WP2 — import Library with ownership and provenance

- Parse real `lib/tools/createdsession.tool.yaml` (12 Operations), `mas-proposed.tool.yaml` (16 proposed Operations) including YAML anchor/alias; reject duplicate Operation IDs, malformed schemas and conflicting declarations.
- Parse optional `lib/manifest.yaml`: each imported catalog has relative path and `status: declared|proposed|unspecified`. This is contract maturity only; **no Runtime READY state can be returned**.
- Provide read-only, inspectable Library data for Studio via single Core API (as specified in [Studio API](../studio-integration.md)): operation ID, source path, interface schemas, declared Effect, provenance/status. Do not invent a second Studio parser.
- The absence of a Host/Observation schema must be reflected in diagnostics when a Procedure binds its fields. Not everything in `context.*` belongs to ToolCatalog.

## 6. WP3 — link external input + generate Script-owned Output

- Resolve `context.request.*` against explicitly linked request Schema, `context.subject/facts/observations.*` against real imported Owner schema, `input.*` against Procedure inputs, `<node>.output.*` against source Output Schema, and nested preconditions step paths per [Procedure Link v0](../design/procedure-link-v0.md).
- For each Script `with:` resolve every `from:` and concrete literal into a **per-site input projection**. Do not make the author retype Tool/Host interface as Script Input TypedDict.
- Accept exactly new Script contract `main(input: dict) -> NonEmptyOutputTypedDict`; generate **only** Script's new Output JSON Schema at `skills/<skill>/schemas/generated/<script-relative>.output.schema.json` in outputFiles.
- Never accept a broad output type, dynamic annotation, empty class or missing upstream Producer by downgrading it to loose object. Do not use Python interpreter/subprocess or runtime reflection. Annotated constraints must be checked.
- **No legacy fallback** from historical Script metadata / old external Input+Output Schema. Instead, produce migration diagnostics for raw ca-agent-demo samples.
- Synthetic `examples/script-contract-project` is the positive demonstration: request field → Tool call → Tool Output → Script with → typed Script Output; use its real read-only ToolCatalog fixture.

## 7. WP4 — deterministic materialization + Native CLI

- Core API remains `Check/Build` from memory file bytes. Implement `InspectLibraries` (name/shape defined by Studio API). No new HTTP server.
- Build whole project or root dependency closure and include only relevant imported catalog snapshots. Output relative paths deterministic and sorted, no run ID/release/digest. Never materialize intermediate Script input projections as editable Source files.
- CLI supports `skillc check PROJECT` / `skillc build PROJECT --out FRESH_DIR` and `--skill NAME`; nonzero on compiler ERROR, reject symlinks/path traversal, never overwrite source/library or stale nonempty output.

## 8. WP5 — WASM / Worker / Pages

- Build the same Core for `GOOS=js GOARCH=wasm`. Bundle matching `wasm_exec.js`, worker script and static test harness; do not install a server API.
- Compare Native and browser WASM diagnostics/outputFiles **content bytes and ordering** for both successful and failing fixtures. Document size/time and make worker error returns distinct from compiler errors.
- A Go/WASM binary merely existing is NOT a passed browser test. Run actual Chromium static-site invocation if environment supports it; otherwise mark BLOCKED/NOT_RUN precisely.

## 9. WP6 — real 5GC source migration evidence (NO COMPAT)

- Read pinned real fixture without modifying originals. Run strict compiler against it; **expect actionable migration errors, not success by supporting old Python Script headers**.
- If feasible, create a separate `examples/ca-agent-demo-v0/` containing migrated Source, preserving upstream snapshot untouched. Do not patch the upstream repository; do not invent Host/Observation contract or MAS implementation to make this green.
- Give an explicit report for each real Skill: which Script Outputs require typed migration, which Procedure bindings/Predicate contracts are unsupported, which Tool/Host schema owners are still missing, which checks are demonstrably complete.
- The MAS contract carries `status: proposed`; CLI/Compiler cannot claim MAS runtime READY.
- Host consumer changes may be necessary by design. Record them for a separate Owner PR, do not implement a compatibility adapter to preserve old identity/artifact format.

## 10. WP7 — finish, verify, push, PR

In a network-enabled local environment execute at least:

~~~bash
go mod tidy
gofmt -w $(git ls-files '*.go')
go test ./... -count=1
go vet ./...
mkdir -p dist
go build -o ./dist/skillc ./cmd/skillc
GOOS=js GOARCH=wasm go build -o ./dist/skillc.wasm ./cmd/skillc-wasm
go run ./cmd/skillc check ./examples/minimal-project
go run ./cmd/skillc check ./examples/script-contract-project
# Strict checks on the raw 5GC references MUST report migration defects;
# do NOT change validation to make the old source pass.
go run ./cmd/skillc check ./examples/ca-agent-demo-project
git diff --check
~~~

All outputs must be recorded as observed (Exit Code and diagnostic codes). The last command for raw 5GC is an **expected nonzero / migration-diagnostic** check when legacy constructs exist; do not gate this as “all commands exit 0”. For Windows adjust shell syntax, preserve identical semantics. For repeatable builds, clear dist/fresh output dirs explicitly.

- Update `docs/WORK.md` with IMPLEMENTED/PARTIAL/NOT_STARTED and PASS/FAIL/BLOCKED/NOT_RUN separately, add a concise acceptance report structured by [G0–G9](acceptance-matrix-v0.md).
- Commit and push to **new implementation branch**, open PR **targeting `feat/compiler-v0-bootstrap`**, not main. Do not force-push, do not modify other repos, run GitHub Actions or real 5GC, and do not merge while compiler tests fail.
- Work through as much as possible in one Harness execution; at most one implementation PR is preferred. If a genuine external owner or environment blocks a gate, finish the remaining independent gates and report precise blockers, rather than changing the spec.
- Final response in same run: branch SHA/PR, G0–G9 matrix, native/wasm/browser test evidence, synthetic fixture results, raw 5GC migration diagnostics, pending Owner dependencies.

**Scope is Compiler v0 implementation only; not a new Skill Studio project or tool management platform.**
