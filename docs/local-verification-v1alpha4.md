> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Local Verification v1alpha4

**Status:** Reproducible local acceptance procedure for the CA-SL v1alpha4 Compiler.
**Policy:** This run uses local commands as execution evidence. GitHub Actions is not
available and is never reported as PASS. External Owner/infrastructure qualification is
reported separately as BLOCKED or NOT_QUALIFIED, never folded into a local PASS.

## Toolchain

The verified environment for the recorded evidence is:

```text
go version go1.26.6 windows/amd64
node v24.18.0            (Chromium parity only)
Chrome for Testing / system Google Chrome (Chromium parity only)
```

`go.mod` declares `go 1.25.0`; no CGO is required. `GOWORK=off` is used so an unrelated
workspace file cannot change module selection.

## Core gates

Run from the repository root:

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
git diff --check
```

Expected result: exit `0` for every command. `go test ./...` covers the Compiler Core,
artifact, YAML AST, Python CST adapter, CLI package and WASM request-size tests. The
Chromium parity test is opt-in and skipped in this default run.

## CLI v1 end-to-end

The CLI tests build the real `skillc` executable and run it as a subprocess, so the
command line, exit codes, stderr diagnostics and written artifact tree are exercised:

```bash
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
```

Covered behaviour is listed in [CLI v1alpha4 §9](cli-v1alpha4.md#9-verification).
Also verify the reference commands by hand:

```bash
GOWORK=off go run ./cmd/skillc check   web/parity-success
GOWORK=off go run ./cmd/skillc inspect web/parity-success
GOWORK=off go run ./cmd/skillc build   web/parity-success --out "$(mktemp -d)/skillc-out"
GOWORK=off go run ./cmd/skillc check   web/parity-success --json
```

Expected: check and inspect exit `0`, build writes five files including
`skills/verify-ue-registration/generated/artifact-manifest.json`.

## Native / WASM / Chromium parity

Real headless Chromium is required for the final parity gate; a successful WASM build is
not accepted as a substitute for browser execution:

```bash
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
```

`CHROME_PATH` may override the browser executable. The harness
([`scripts/chromium-parity.mjs`](../scripts/chromium-parity.mjs)) builds the CGo-free
Go/WASM module, serves the Worker/Studio pages, drives real Chromium, and compares the
browser projection against the Native Go reference (`cmd/parity-replay`):

- check diagnostics identical;
- build success, diagnostics, dependency closure and output paths identical;
- inspect projection identical;
- successful-Build artifact files byte-identical by SHA-256;
- in-Worker artifact integrity verification PASS;
- Studio diagnostics and read-only artifact presentation identical.

Expected: `PASS: Native and real Chromium Worker projections are identical`, exit `0`.

## Requirement coverage map

| Required check | Where it is executed |
|---|---|
| Full unit/package suite | `GOWORK=off go test ./... -count=1` |
| Normal Skill build | `TestCLIEndToEndTrustedBuildAndDeterminism`; `internal/compiler` build fixtures |
| Missing dependency | `TestCLIEndToEndMissingDependencyFailsClosed`; `dependency_identity_test.go` |
| Owner identity error | `owner_linking_test.go`, `e_mid_test.go`, `e_route_test.go` |
| Schema reference resolution | `schema_registry_test.go`, `schema_g4_test.go`, `source_path_schema_test.go` |
| CEL subset and guards | `cel_check_test.go` |
| Procedure graph | `procedure_graph_test.go`, `procedure_stages_test.go`, `route_*_test.go` |
| Script ABI | `script_abi_test.go`, `script_*_test.go` |
| Static slicing / dataflow | `source_path_*_test.go`, `argument_*_test.go` |
| Artifact determinism / JCS | `artifact_generate_test.go`, `artifact_verify_test.go`, `TestCLIEndToEndTrustedBuildAndDeterminism` |
| Native/WASM/Chromium consistency | `CHROMIUM_PARITY=1 go test ./internal/parity` |
| CGo-free Native / WASM builds | `CGO_ENABLED=0 go build`, `GOOS=js GOARCH=wasm go build` |
| Format/whitespace gate | `git diff --check` |

## PASS / PARTIAL / BLOCKED / N/A discipline

- **PASS** requires an executed command or test with a recorded exit status in this
  environment.
- **PARTIAL** means some sub-branches have executed static evidence while the remaining
  branch needs an unfrozen external contract; the exact sub-branch is recorded in
  `tests/conformance/v1alpha4/e-coverage.json`.
- **BLOCKED** is used only for external publications or normative clarifications the
  maintainer owns; it is never counted as PASS.
- **N/A** is used only for E-cases whose owner is exclusively Host/Runtime, never for
  Compiler-owned work.

Current disposition totals are maintained in
[Acceptance Status §4](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md#4-e-coverage) and the
machine-readable `tests/conformance/v1alpha4/e-coverage.json`.

## Not part of local acceptance

The following are outside this repository's local acceptance and must not be simulated:

- real 5GC / MAS end-to-end qualification (WP13);
- real Owner ToolCatalog and CapabilityContracts releases;
- deployment authorization servers, package registries or release platforms;
- GitHub Actions execution.
