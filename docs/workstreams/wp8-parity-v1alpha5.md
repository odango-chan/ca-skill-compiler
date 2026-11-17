# WP8 鈥?Native / WASM / real Chromium Worker parity (v1alpha5 candidate slice)

**Status:** candidate-target slice implemented and locally verified 鈥?**PASS**
for the checked projections. This is cross-runtime parity evidence for the
candidate `connectionagent.io/v1alpha5` target, not language publication,
product qualification, Host/Runtime implementation or 5GC/MAS qualification.
v1alpha4 remains the only published authoring contract and its behavior is
unchanged.

**Branch:** `feat/compiler-v1alpha5-wp8-parity-20261117` (based on WP2 HEAD
`34993dc`). Stacked Draft PR with base `feat/compiler-v1alpha5-wait-ast-20261117`
(PR #30).

## 1. Scope proven now

| Claim | Evidence |
|---|---|
| One deterministic Go Core is shared by the CLI, the Native parity reference, Go/WASM and the real Chromium Web Worker | `cmd/skillc`, `cmd/parity-replay`, `cmd/skillc-wasm`, `web/skillc-worker.js` all call `internal/compiler.Check`/`Build`/`InspectDependencies`; only the transport differs |
| CLI `check --request` dispatches a `BuildRequest` with `languageTarget: connectionagent.io/v1alpha5` to the v1alpha5 Check | `TestCLIEndToEndV1Alpha5CheckRequest` |
| A valid v1alpha5 Procedure without a Wait produces `Diagnostics == []`-equivalent output (JSON `[]`, exit 0) | CLI e2e test; Chromium `ok: v1alpha5 valid Check returns zero diagnostics` |
| A syntactically valid dependency-free Wait resolves to zero trusted Ingress Operation exports and reports `WAIT_INGRESS_EXPORT_NOT_FOUND` (exact L.3.1 identity), never a silent pass | CLI `TestCLIEndToEndV1Alpha5CheckRequest` (`wait-no-ingress` fixture) |
| An invalid v1alpha5 Procedure produces deterministic, positioned `WAIT_*` diagnostics (6 diagnostics, exact code/line/column) | CLI e2e test; Chromium `ok: v1alpha5 invalid WAIT_* diagnostics positioned and deterministic (6 diagnostics)` |
| Validation diagnostics are identical between Native Go and the real Chromium Worker for the same request | `ok: v1alpha5 check projection identical`, `ok: v1alpha5 invalid-diagnostics projection identical` |
| `Build` stays fail-closed for v1alpha5 in both directions: a valid Procedure yields exactly the one documented internal residual diagnostic; an invalid Procedure preserves its `WAIT_*` diagnostics | CLI e2e test; `ok: v1alpha5 build.success projection identical`, `ok: v1alpha5 build.diagnostics projection identical`, `ok: v1alpha5 invalid build.diagnostics projection identical` |
| Every v1alpha5 `BuildResult` is stamped with `connectionagent.io/compiler-build-result/v1alpha5`, including fail-closed results, and emits no artifact files | CLI e2e test; `ok: v1alpha5 build.apiVersion projection identical` |
| The frozen v1alpha4 projections 鈥?including successful-Build artifact byte parity 鈥?still pass unchanged | `ok: successful-Build artifact byte parity (5 files)` and the unchanged v1alpha4 markers |
| CLI `version`/`help` document both language targets and the independent request/result API versions | `TestCLIEndToEndV1Alpha5VersionAndHelp` |

The invalid fixture exercises `WAIT_NODE_STAGE_REQUIRED`,
`WAIT_DECLARATION_INVALID`, `WAIT_TIMEOUT_INVALID`,
`WAIT_RESPONSE_BOUNDARY_INVALID` and `WAIT_TIMEOUT_ROUTE_INVALID`; all are
`Severity=error`, `Phase=C_STATIC`, path
`skills/example/assurance/procedure.yaml`, with non-zero line/column.

## 2. Deferred scope (explicit)

| Deferred item | Why it is not claimed |
|---|---|
| Successful-Build v1alpha5 **artifact byte parity** | Artifact assembly and the remaining static pipeline are WP3鈥揥P6 work; WP2 `Build` intentionally fails closed after AST validation, so there is no successful v1alpha5 artifact to compare yet |
| v1alpha5 `DependencyInspection` ingress inventory (`ingressOperations` or equivalent) | The v1alpha5 inspection API version is reserved but the ingress projection is not implemented; `inspect` currently reports the inherited v1alpha4 envelope rejection rather than guessing a contract |
| Studio v1alpha5 success-path rendering | Follows from the deferred successful Build; Studio remains v1alpha4-success + both fail-closed presentations |
| Real Owner/Deployment/5GC/MAS qualification | Compiler parity is not product qualification; remains **NOT_QUALIFIED** |
| GitHub Actions CI evidence | No runner executed steps in this environment 鈥?**CI_UNAVAILABLE** (see 搂6) |

## 3. Changes in this slice

| File | Change |
|---|---|
| `cmd/skillc/main.go` | Additive `version` output (both language targets, independent request/result API versions) and help text documenting `--request` target selection, language targets and API versions; no command or exit-code behavior changed |
| `cmd/skillc/e2e_test.go` | New v1alpha5 CLI end-to-end tests for version/help, valid + invalid `check --request`, fail-closed `build --request` (`apiVersion`, residual, invalid diagnostics, no output files) and deferred `inspect` behavior; existing v1alpha4 tests untouched |
| `cmd/parity-replay/main.go` | Additive `--request=<BuildRequest.json>` mode using the same strict `compiler.DecodeBuildRequestJSON` path as the CLI/WASM bridge; project mode unchanged. The reference view now also carries `build.apiVersion` |
| `scripts/chromium-parity.mjs` | New v1alpha5 phase: writes explicit v1alpha5 `BuildRequest` JSON from checked-in synthetic author fixtures, runs the Native reference through `--request`, runs the same request through `web/parity.html` 鈫?`web/skillc-worker.js` 鈫?the same Go/WASM Core in real headless Chromium, and compares check/build/result-version/inspect projections; prints explicit v1alpha5 markers |
| `web/parity-v1alpha5/valid-procedure.yaml`, `invalid-procedure.yaml` | Synthetic v1alpha5 Wait author fixtures (author files only; no trusted dependency snapshot needed) |
| `internal/parity/chromium_test.go` | Requires the new v1alpha5 markers so a missing/regressed v1alpha5 comparison fails the opt-in test |
| `docs/workstreams/wp8-parity-v1alpha5.md` | This record |

`web/parity.html` and `web/skillc-worker.js` needed no change: the page already
fetches a named request file and the worker already delegates to the shared Go
Core callbacks, so the candidate target flows through the same transport.

## 4. Harness sequence (real Chromium)

`scripts/chromium-parity.mjs` now runs, in order:

1. CGo-free Go/WASM build (`GOOS=js GOARCH=wasm`) into `.parity-web/skillc.wasm`
   plus the pinned toolchain `wasm_exec.js`, Worker, parity page and Studio page.
2. Frozen v1alpha4 project phase (`web/parity-project`) 鈥?unchanged comparisons.
3. Frozen v1alpha4 Studio integration phase 鈥?unchanged comparisons.
4. Frozen v1alpha4 successful-Build phase (`web/parity-success`) 鈥?unchanged,
   including SHA-256 artifact byte parity for all 5 files.
5. **v1alpha5 candidate phase** (new): writes
   `.parity-web/parity-v1alpha5-{valid,invalid}-request.json`, computes the
   Native reference with `cmd/parity-replay --request=...`, then loads the same
   request in real headless Chromium through `web/parity.html` + the Web Worker,
   and compares `check`, `build.success`, `build.apiVersion`,
   `build.diagnostics`, `inspect`, and the invalid-case diagnostics.

The phase asserts natively first that the valid fixture Checks clean, that the
valid Build is fail-closed with exactly one `CAPABILITY_INPUT_UNTRUSTED`
residual message prefixed `fail-closed:`, that every invalid diagnostic is a
`WAIT_*` code with non-zero line/column and the expected path, and that the
invalid Build echoes the same diagnostics.

## 5. Local verification (executed)

Environment: Windows; Go `go1.26.6 windows/amd64`; Node `v24.18.0`; Chrome
`158.0.8127.37`; `GOWORK=off`, `CGO_ENABLED=0` for the native gates. Worktree
`wt/ca-wp8` on the branch above.

| Command | Result |
|---|---|
| `go test ./... -count=1` | **PASS** (`cmd/skillc`, `cmd/skillc-wasm`, `internal/artifact`, `internal/compiler`, `internal/compiler/yamlast`, `internal/parity`, `internal/projectinput` all `ok`) |
| `go vet ./...` | **PASS** (exit 0, no findings) |
| `CGO_ENABLED=0 go build ./...` | **PASS** (exit 0) |
| `CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm` | **PASS** (exit 0) |
| `go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | **PASS** (13/13, including the 4 new v1alpha5 tests) |
| `git diff --check` | **PASS** (no whitespace errors) |
| `CHROMIUM_PARITY=1 go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | **PASS** (49.32 s; see output below) |

Real Chromium Worker parity output (from the official gate command above; markers
are required by `internal/parity/chromium_test.go`):

```text
ok: check projection identical
ok: build.success projection identical
ok: build.apiVersion projection identical
ok: build.diagnostics projection identical
ok: build.dependencyClosure projection identical
ok: build.outputFiles.paths projection identical
ok: inspect projection identical
ok: studio.diagnostics projection identical
ok: studio.buildSuccess projection identical
ok: studio.artifactPaths projection identical
ok: studio diagnostics/artifact read-only integration identical
ok: Chromium Worker artifact integrity verification PASS
ok: success.check projection identical
ok: success.build.success projection identical
ok: success.build.apiVersion projection identical
ok: success.build.diagnostics projection identical
ok: success.build.dependencyClosure projection identical
ok: success.build.outputFiles.paths projection identical
ok: successful-Build artifact byte parity (5 files)
ok: studio-success.buildSuccess projection identical
ok: studio-success.artifactPaths projection identical
ok: Studio success-path artifact presentation identical
ok: v1alpha5 valid Check returns zero diagnostics
ok: v1alpha5 invalid WAIT_* diagnostics positioned and deterministic (6 diagnostics)
ok: v1alpha5 check projection identical
ok: v1alpha5 build.success projection identical
ok: v1alpha5 build.apiVersion projection identical
ok: v1alpha5 build.diagnostics projection identical
ok: v1alpha5 inspect projection identical
ok: v1alpha5 invalid-diagnostics projection identical
ok: v1alpha5 invalid build.diagnostics projection identical
ok: v1alpha5 invalid inspect projection identical
PASS: Native and real Chromium Worker v1alpha5 parity
PASS: Native and real Chromium Worker projections are identical
--- PASS: TestChromiumWorkerParity (49.32s)
```

CLI surface:

```text
skillc v1.0.0
language target: connectionagent.io/v1alpha4 (frozen)
language target: connectionagent.io/v1alpha5 (candidate)
build request apiVersion: connectionagent.io/compiler-build-request/v1alpha4 (frozen)
build request apiVersion: connectionagent.io/compiler-build-request/v1alpha5 (candidate)
build result apiVersion: connectionagent.io/compiler-build-result/v1alpha4 (frozen)
build result apiVersion: connectionagent.io/compiler-build-result/v1alpha5 (candidate)
dependency inspection apiVersion: connectionagent.io/compiler-dependency-inspection/v1alpha4 (frozen; v1alpha5 ingress not yet wired)
```

`dist/` build output is local only and not committed.

## 6. CI status

**CI_UNAVAILABLE.** No GitHub Actions runner executed any step for this branch in
this environment; the workflow files were not exercised and no remote CI result
is claimed. All evidence above is local execution evidence only. `go test` and
every listed gate were actually executed on the worktree; the real Chromium
Worker run did launch Chrome and the WASM Worker, as recorded by the markers.
