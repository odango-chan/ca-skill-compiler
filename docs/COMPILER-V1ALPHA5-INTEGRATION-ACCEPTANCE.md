# CA Skill Compiler v1alpha5 — integration acceptance (local, candidate)

**Status: CANDIDATE / DRAFT LOCAL ACCEPTANCE — NOT a release, NOT CI PASS, NOT product
qualification.** The `connectionagent.io/v1alpha5` language target and the ca-skills
PR #61/#62 contracts are Draft. `v1alpha4` remains the only published authoring
contract. Real Owner / Deployment / 5GC / MAS qualification remains
**NOT_QUALIFIED**. No PR was merged to `main`; the work is delivered as reviewed
Draft PRs on dedicated branches.

## 1. Delivered branches and PRs (all Draft, all pushed)

| WP | Branch | Remote HEAD at acceptance | PR |
|---|---|---|---|
| WP1 IngressCatalog + Schema Closure Digest v1 | `feat/compiler-v1alpha5-wp1-ingress-catalog-20261117` | `42a16d37c7d73d3becbd6a2e35e3cbbfedb67f48` | [#31](https://github.com/odango-chan/ca-skill-compiler/pull/31) |
| WP2 Wait parser/AST + v1alpha5 API admission | `feat/compiler-v1alpha5-wait-ast-20261117` | `34993dc30b66f4cf7d573cbc81d884c356062a12` | [#30](https://github.com/odango-chan/ca-skill-compiler/pull/30) |
| WP5 Procedure graph / Wait entry control | `feat/compiler-v1alpha5-wp5-procedure-graph-20261117` | `d33fe0d980ea423f9121de2e50cd5476da8bac57` | [#32](https://github.com/odango-chan/ca-skill-compiler/pull/32) |
| WP7 Diagnostic contract / EW-IN traceability | `feat/compiler-v1alpha5-wp7-conformance-20261117` | `10d45cb757054077a16ff234a4f7aa306e4269a5` | [#33](https://github.com/odango-chan/ca-skill-compiler/pull/33) |
| WP8 CLI + Native/WASM/Chromium parity (Check slice) | `feat/compiler-v1alpha5-wp8-parity-20261117` | `d95f97dd9bafb77690efd3ccc4fb97ab5384ee7a` | [#34](https://github.com/odango-chan/ca-skill-compiler/pull/34) |
| WP3 Ingress exact link / trigger-correlation | `feat/compiler-v1alpha5-wp3-trigger-binding-20261117` | `ed9bebd9b5aa9eb14cb7c814de56d1fb935fcbe2` | [#35](https://github.com/odango-chan/ca-skill-compiler/pull/35) |
| WP4 `wait.output` SourcePath | `feat/compiler-v1alpha5-wp4-wait-output-20261117` | `09d7e6beb78fbbfb01c2fb8f5e2286c13e050e39` | [#36](https://github.com/odango-chan/ca-skill-compiler/pull/36) |
| WP6 Response boundary / typed mapping | `feat/compiler-v1alpha5-wp6-response-boundary-20261117` | `32f1de5d95dac9d8b21c0b3b2b2e1843a7a35972` | [#37](https://github.com/odango-chan/ca-skill-compiler/pull/37) |
| Post-integration WP7 + Ingress DependencyInspection | `feat/compiler-v1alpha5-wp7b-conformance-inspect-20261117` | `45f7c03aecff68688e1202cef6559fc50b44a75c` | [#39](https://github.com/odango-chan/ca-skill-compiler/pull/39) |
| Deterministic Build artifacts + artifact parity | `feat/compiler-v1alpha5-artifact-20261117` | `057c8dce49017ce9d82ff6f75e707c64f7db55e2` | [#40](https://github.com/odango-chan/ca-skill-compiler/pull/40) |
| Consolidated integration baseline | `feat/compiler-v1alpha5-integration-20261117` | acceptance run at `e9b7d98` (see PR head) | [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38) |

Recommended merge order if/when the owner decides to merge: #31 → #30 → #32 → #35 →
#36 → #37 → #34 → #33 → #39 → #40 → #38, retargeting stacked PRs as their bases land.

## 2. What the candidate compiler now does

- **WP1** — Schema Closure Digest v1 implemented exactly (raw-byte artifact digest
  binding, canonical document URI, dialect-aware `$id`/fragment resolution, exact
  transitive `$ref` closure, JCS manifest, domain-separated SHA-256; #62 golden
  vector reproduced: `sha256:ee0952a9ee8e7c14c83716cc101a6776556214f6d701f8e4e1517a89b8e5b74c`).
  IngressCatalog v1alpha1 structural + identity admission with
  `INGRESS_*` diagnostics, trusted-only discovery and typed projections.
- **WP2** — Complete v1alpha5 Node-entry Wait grammar/AST with exact L.1.1 duration
  subset, selector, response, TIMEOUT route and Wait-bearing-Node rules, precise
  diagnostics/anchors, integrated `V1Alpha5ProcedureAST` and independent strict
  v1alpha5 request/result/inspection API versions.
- **WP3** — Trusted-only Ingress exact link (`WAIT_INGRESS_EXPORT_NOT_FOUND` /
  `AMBIGUOUS`), exact ToolCatalog trigger Operation identity, typed
  `armInputSchema`/normalized Schema binding through Schema Closure Digest v1,
  correlation/stale-delivery and per-incoming-path trigger dominance/pre-dispatch
  materialization (`WAIT_CORRELATION_BINDING_INVALID`).
- **WP4** — `NodeId.wait.output` Producer SourcePath bound to the exact normalized
  request Schema, event-accepted-path availability via graph dominance
  (`SOURCE_NOT_AVAILABLE` on TIMEOUT/bypass), conservative consumer typing.
- **WP5** — Procedure graph with Wait entry-control vertices, implicit
  EVENT_RECEIVED and TIMEOUT control edges, DAG/reachability/dominance and the
  final-Postcondition-PASS Goal rule; event receipt is never a Goal proof.
- **WP6** — Paired-response boundary selection, post-dominance/early-failure
  proof, typed response-result Producer/Output-Schema assignability and
  Ingress-owned response contract/mapping coverage (`WAIT_RESPONSE_BOUNDARY_INVALID`,
  `WAIT_RESPONSE_MAPPING_INVALID`, `INGRESS_RESPONSE_*`).
- **WP7** — Normative diagnostic registry (all 9 L.11 Wait identities + emitted
  Ingress identities with message keys/templates), 99-case EW/IN conformance
  register (37 implemented+tested / 19 deferred / 42 external / 1 spec-only) with
  JSON↔Markdown↔test-name integrity; no row is a conformance PASS.
- **WP8** — Deterministic candidate Build artifact assembly and verification
  (`manifest.json`, `skills/<name>/compiled/procedure.json`,
  `skills/<name>/compiled/ingress-links.json`; JCS, sorted digests, measured
  dependency closure, linked Ingress/response metadata, `validationScope`), CLI
  support for both targets, v1alpha5 `DependencyInspection.ingressOperations`
  inventory, and Native/WASM/real-Chromium Worker parity including per-file
  artifact byte parity.

## 3. Executed local acceptance evidence

Environment: Windows, Go 1.26.6 windows/amd64, Node v24.18.0, Chrome 158.0.8127.37.
All commands were actually executed on the integration branch; no CI result is claimed.

| Gate | Command | Result |
|---|---|---|
| Unit/integration suite | `GOWORK=off go test ./... -count=1` | **PASS** (all packages) |
| Vet | `GOWORK=off go vet ./...` | **PASS** |
| CGO-free native build | `CGO_ENABLED=0 GOWORK=off go build ./...` | **PASS** |
| Go/WASM build | `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | **PASS** |
| CLI end-to-end | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | **PASS** (13 v1alpha4+v1alpha5 e2e tests) |
| Real Chromium Worker parity | `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | **PASS** (`TestChromiumWorkerParity` 36.27s) |
| Whitespace | `git diff --check` | **PASS** |

Real-Chromium parity output includes, unchanged v1alpha4 projections, plus:
`ok: v1alpha5 valid Check returns zero diagnostics`,
`ok: v1alpha5 valid Build succeeds with deterministic artifact`,
`ok: v1alpha5 trusted Build succeeds with dependency closure (3 files)`,
`ok: v1alpha5 successful-Build artifact byte parity (3 files)`,
`ok: v1alpha5 trusted successful-Build artifact byte parity (3 files)`,
`ok: v1alpha5 trusted linked Ingress metadata identical`,
`ok: v1alpha5 Chromium Worker VerifyArtifact PASS`,
`ok: v1alpha5 inspect projection identical`,
`PASS: Native and real Chromium Worker v1alpha5 artifact parity`,
`PASS: Native and real Chromium Worker projections are identical`.

**CI: CI_UNAVAILABLE.** No GitHub Actions runner executed any step for these
branches in this environment; `gh pr checks` reports no checks. Local acceptance is
never written as CI PASS.

## 4. Not validated / deferred (see `docs/known-limitations-v1alpha5.md`)

- Non-Wait tool/dependency linking, full package requirements (`SKILL.md`,
  contract/predicate registries, package Schemas, Script ABI) and the v1alpha4
  Schema/CEL/predicate/Script/Evidence/Mutation/Reconcile/retry static semantics
  are not implemented as v1alpha5 passes. The artifact manifest declares this via
  `validationScope.status = CANDIDATE_PARTIAL` and lists the unvalidated
  obligations; a consumer requiring full validation must reject such an artifact.
- v1alpha5 `DependencyInspection` ingress inventory is trusted-binding scoped and
  the real-Chromium trusted-ingress inventory parity phase is not claimed.
- Studio v1alpha5 successful-Build rendering is not added.
- Host / Runtime / Integration semantics (Wait lifecycle, event ingress,
  tombstone/stale-delivery execution, response delivery, retries, authorization)
  are out of Compiler scope.
- All trusted Catalogs/ToolCatalogs/Schemas/OpenAPI artifacts used by tests and
  parity are clearly marked SYNTHETIC compiler-owned fixtures, not real Owner
  publications. Real Owner / Deployment / 5GC / MAS qualification remains
  **NOT_QUALIFIED**.

## 5. Reproduction

From the integration branch:

```powershell
$env:GOWORK='off'
go test ./... -count=1
go vet ./...
$env:CGO_ENABLED='0'; go build ./...
$env:CGO_ENABLED='0'; $env:GOOS='js'; $env:GOARCH='wasm'; go build -o dist/skillc.wasm ./cmd/skillc-wasm
$env:CGO_ENABLED='1'
go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
$env:CHROMIUM_PARITY='1'
go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
```

`dist/`, `.parity-web/` and Chrome profiles are local build artifacts and are not
committed.

## 6. Release boundary

This acceptance covers **Compiler-owned static capability on a candidate language
target**, not a language release and not product qualification. `v1alpha4` behavior,
diagnostics and artifact bytes are unchanged by this work. No PR was merged and no
tag/release asset was modified.
