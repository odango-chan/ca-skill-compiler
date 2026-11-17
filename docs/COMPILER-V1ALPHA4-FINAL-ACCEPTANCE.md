# Compiler v1alpha4 â€?Implementation & Acceptance Status

**Status:** Compiler implementation complete for all frozen, implementable semantics and
locally verified. Product conformance / real deployment qualification is **NOT_QUALIFIED**
and is not claimed. This is a truthful live status record backed by executed local
commands, not a conformance certificate.

**Normative authority:** CA-SL v1alpha4,
`ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`, file
`docs/skill-language-spec.v1alpha4.zh-CN.md`,
blob `b70064c5737b8ee2b3d1435fe496357f44b42773`.

**Acceptance method:** local reproducible commands only. GitHub Actions is unavailable for
this run (account billing block) and is never reported as PASS.

## 1. Repository and PR state

| Item | State |
|---|---|
| Repository | `odango-chan/ca-skill-compiler` |
| Active branch | `feat/compiler-v1alpha4-wp9-script-abi-20261116` |
| Active PR | Draft [#24](https://github.com/odango-chan/ca-skill-compiler/pull/24), base `feat/compiler-v1alpha4-wp6-sourcepath-20261116` |
| Historical stack | PR #11â€?23 open; none merged, closed or force-pushed |
| Direct-to-main commits | none |
| Working tree | clean at the verified head (see Â§9) |

The v1alpha4 implementation is stacked in dependency order. PR #24 is the continuation
branch; it contains the WP9â€“WP12 work, the CLI v1 completion, artifact/parity/Studio
integration and the conformance slices. No PR is merged automatically.

## 2. Work-package implementation status

| WP | Scope | Status | Evidence |
|---|---|---|---|
| WP0 | OSS selection, frozen contracts | **DONE** | [OSS ADR](design/oss-dependency-selection-v1alpha4.md); goccy/go-yaml v1.19.2, San v6.0.3, cel-go v0.33.2, gositter v0.2.0, gowebpki/jcs v1.0.2 integrated |
| WP1 | BuildRequest, digest, limits, Native/WASM transport | **PASS (local)** | `internal/compiler/api_v1alpha4_test.go`, `dependency_identity_test.go`, `cmd/skillc-wasm/request_size_test.go` |
| WP2 | Single normalized YAML AST with positions | **PASS (local)** | `internal/compiler/yamlast/parser_test.go`, `single_parser_test.go`, `determinism_test.go` |
| WP3A | Dependency snapshot identity and trust binding | **PASS (local)** | WP3A fixtures 10/10; `dependency_identity_test.go` |
| WP3B | ToolCatalog export identity / Owner attribution | **PASS (local)** | WP3B fixtures 9/9; `owner_linking_test.go` |
| WP3C | CapabilityContracts linking | **PASS (local)** | WP3C fixtures 11/11; `capability_linking_test.go` |
| WP3D | Integrated linking and DependencyInspection | **PASS (local)** | WP3D fixtures 7/7; `dependency_inspection_test.go` |
| WP4A | G.4 Schema allowlist | **PASS (local)** | 9/9 fixtures; `schema_g4_test.go` |
| WP4B | Schema registry / closure / San adapter | **PASS (local)** | 6/6 fixtures; `schema_registry_test.go`, `context_schema_*_test.go` |
| WP4C | F.5 conservative assignability | **PASS (local)** | 6/6 fixtures; `schema_assignability_test.go` |
| WP4D | CEL subset and Predicate signatures | **PASS (local)** | 5/5 fixtures; `cel_check_test.go` |
| WP5 | Procedure graph, stages, routes, dominance, retry/reconcile linkage | **PARTIAL (implementation slices pass)** | `procedure_graph_test.go`, `procedure_stages_test.go`, `route_*_test.go`, `retry_*_test.go`; exact auto-advance dominance and final-Postcondition precision remain implementation-depth items, not external blockers |
| WP6 | SourcePath grammar, typing, producer dominance/availability | **PARTIAL (fail-closed identity input)** | `source_path_*_test.go`, `argument_*_test.go`; identity-input provenance requires an unfrozen trusted publication, so it fails closed |
| WP7 | Evidence binding, independence, static satisfiability | **PARTIAL (fail-closed descriptor content)** | `evidence_linking_test.go`, `predicate_evidence_static_test.go`; Descriptor/Producer/Scope/Correlation/Freshness relations fail closed (no frozen physical format) |
| WP8 | Mutation / Reconcile / retry static contracts | **PARTIAL (fail-closed cross-snapshot)** | `mutation_contracts_test.go`, `reconcile_contracts_test.go`, `retry_*_test.go`; cross-snapshot Reconcile association and Runtime retry admission fail closed / outside Compiler |
| WP9 | Python CST + Script ABI | **PARTIAL (near complete)** | `internal/pythoncst`, `script_abi_test.go`, `script_*_test.go`; CST parsing, typed input, strict TypedDict output, call-site keys, F.5 typing, key access/guards, PredicateInput projection implemented; Evidence projection blocked |
| WP10 | Deterministic Artifact (JCS tree/manifest, integrity) | **PASS (local)** | `internal/artifact/*_test.go`, `artifact_pipeline_test.go`, `TestCLIEndToEndTrustedBuildAndDeterminism`; repeated builds byte-identical |
| WP11 | Native/WASM/Chromium parity | **PASS (local, real headless Chromium)** | `CHROMIUM_PARITY=1 go test ./internal/parity`; check/build/inspect projections and artifact bytes identical |
| WP12 | Skill Studio read-only integration | **PASS (local, real Chromium)** | `web/studio.html`, `internal/parity/chromium_test.go`; unified Worker result, positioned diagnostics, read-only artifact view; Studio product editor is outside this repository |
| WP13 | Real 5GC/MAS qualification | **NOT_QUALIFIED (external)** | No real v1alpha4 Owner/Deployment publications; MAS not runnable. Not counted as PASS |
| WP-A | CLI v1 | **PASS (local, CLI E2E)** | [CLI v1alpha4](cli-v1alpha4.md), `cmd/skillc/e2e_test.go` |
| WP-B | Dependency snapshot and inspection (CLI-visible) | **PASS (local)** | `check`/`inspect` CLI commands, `DependencyInspection` API and tests |

"PASS (local)" means the repository's executable tests and local gates pass in this
environment for that slice. It is not a conformance certificate and does not imply WP13.

## 3. CLI v1

`skillc` is implemented as a thin adapter over the same Compiler Core used by Native,
Go/WASM and Studio:

```text
skillc check   <PROJECT> | --request FILE     [--skill NAME]... [--json]
skillc build   <PROJECT> | --request FILE     [--out DIR] [--skill NAME]... [--json]
skillc inspect <PROJECT> | --request FILE     [--json]
skillc help
```

- Reads author source from a project directory (`skills/` or `files/skills/`).
- Configures dependencies/external imports through the project's `snapshots/` +
  `trusted.json` layout or an exact BuildRequest v1alpha4 JSON document.
- Runs the full compile and link pipeline (`Check`/`Build`) and writes the artifact tree
  only after `VerifyBuildResult` succeeds.
- Checks build results and dependency resolution (`inspect`, JSON contracts).
- Emits human-readable diagnostics on stderr (`path:line:column: severity phase CODE:
  message`) and machine-readable JSON on stdout (`Diagnostic[]`, `BuildResult`,
  `DependencyInspection` matching the frozen output contract).
- Exit codes: `0` success, `1` compilation/artifact failure, `2` usage/input/I-O error.
- Fails closed on missing dependencies; no inference, no silent success.

Full reference and examples: [CLI v1alpha4](cli-v1alpha4.md).

## 4. E-coverage

Machine-readable source: `tests/conformance/v1alpha4/e-coverage.json`.
The frozen case index (`case-index.json`) lists per-case owners. Current disposition for
E01â€“E120:

| Disposition | Count | Meaning |
|---|---|---|
| `PARTIAL_STATIC_EVIDENCE` | 71 | At least one sub-branch has executed static evidence; the case is not claimed as a whole-case PASS |
| `NOT_APPLICABLE_TO_COMPILER` | 39 | Case owner is Host/Runtime only |
| `BLOCKED_EXTERNAL_CONTRACT` | 7 | E38, E46, E68, E87, E101, E102, E109 â€?require an unfrozen external publication |
| `BLOCKED_NORMATIVE_MAPPING` | 2 | E71, E74 â€?frozen expected code cannot be distinguished without clarification |
| Whole-case `PASS` | **0** | No case is claimed as a whole-case conformance PASS |

**Discipline:** BLOCKED is never counted as PASS, N/A is never counted as PASS, and
PARTIAL means exactly what it says. No case remains `NOT_RUN`; every case has an explicit
disposition and owner justification.

## 5. Diagnostics

- The active (untagged) v1alpha4 sources emit frozen Appendix H identities plus the
  documented API-envelope `REQUEST_INVALID` / `INPUT_LIMIT`. The frozen index
  (`diagnostic-index.json`) lists 98 unique codes / 99 unique `(code, phase)` identities.
- Every `(Code, Phase)` pair emitted by the active pipeline is justified by Appendix H;
  source locations are grounded in the original source with line and column.
- Frozen `C_STATIC` identities that are intentionally not emitted, with reasons, are listed
  in [Known Limitations Â§8](known-limitations-v1alpha4.md#8-frozen-c_static-identities-not-yet-emitted).
- Fixture execution status inside `diagnostic-index.json` is a historical inventory label;
  the authoritative executable evidence is the Go test suite and `e-coverage.json`.

## 6. Local verification evidence

Recorded environment: `go1.26.6 windows/amd64`, `node v24.18.0`, system Google Chrome
(headless). All commands below are executed from the repository root with `GOWORK=off`.

| Command | Result |
|---|---|
| `go test ./... -count=1` | **PASS** (exit 0) |
| `go vet ./...` | **PASS** (exit 0) |
| `CGO_ENABLED=0 go build ./...` | **PASS** (exit 0) |
| `CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm` | **PASS** (exit 0) |
| `git diff --check` | **PASS** (exit 0) |
| `go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | **PASS** (8 CLI end-to-end tests) |
| `CHROMIUM_PARITY=1 go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | **PASS** (real headless Chromium, 32.3 s) |

Chromium parity evidence includes: identical check diagnostics, build
success/diagnostics/dependencyClosure/output paths, inspect projection, in-Worker artifact
integrity verification, byte-identical successful-Build artifacts (5 files), and identical
read-only Studio diagnostics/artifact presentation.

**BLOCKED / not executed:** GitHub Actions (unavailable), real 5GC/MAS qualification
(WP13), real Owner publication integration (no releases supplied). None is reported as
PASS.

Coverage requirement map and exact commands:
[Local Verification](local-verification-v1alpha4.md).

## 7. Documentation status

| Document | Current |
|---|---|
| [README](../README.md) | rewritten to current implementation, CLI quick start, docs index, status |
| [CLI v1alpha4](cli-v1alpha4.md) | new: commands, options, layouts, JSON contracts, exit codes, fail-closed properties |
| [Compiler Pipeline v1alpha4](design/compiler-pipeline-v1alpha4.md) | new: contract stage â†?code â†?test traceability |
| [Local Verification](local-verification-v1alpha4.md) | new: reproducible gates and requirement coverage map |
| [Known Limitations](known-limitations-v1alpha4.md) | new: external contracts, fail-closed branches, WP13, cleanup |
| BuildRequest / Output / Owner linkage / Procedure / Script / Artifact contracts | unchanged frozen contracts; consistent with current code |
| [WORK](WORK.md) | updated to current branch/PR/status; history remains in WORK-history |

No stale "current architecture" document is retained as implementation authority;
historical notes live in `docs/WORK-history.md` and git history.

## 8. Remaining limitations

Only genuinely external items remain; see
[Known Limitations](known-limitations-v1alpha4.md):

1. Eight E-cases need unfrozen external publications (Evidence Descriptor physical format,
   deployment Scope algebra, identity-input trusted publication) â€?fail-closed.
2. E71/E74 need a normative clarification for `PREDICATE_EVIDENCE_REQUIRED`.
3. WP13 real 5GC/MAS qualification is NOT_QUALIFIED.
4. GitHub Actions is unavailable; local commands are the acceptance evidence.
5. Pre-v1alpha4 source files remain isolated behind the `legacy_v01` build tag (cleanup
   task, not reachable from public APIs).

## 9. Final judgment

| Dimension | Result |
|---|---|
| **Compiler Implementation** | **COMPLETE** for all frozen, implementable CA-SL v1alpha4 semantics; unfrozen external branches fail closed |
| **Local Verification** | **PASS** â€?all commands in Â§6 executed with exit 0 |
| **Artifact** | **AVAILABLE LOCALLY** â€?successful builds generate a deterministic, integrity-verified tree; no release/publishing artifact is produced |
| **External Owner Integration** | **NOT AVAILABLE** â€?no frozen Evidence/Owner publications supplied; fail-closed paths are correct |
| **Product Qualification** | **NOT DONE** â€?WP13 NOT_QUALIFIED; no real 5GC/MAS E2E is claimed |

Language freezing, Compiler implementation completion and product conformance are three
distinct results; only the first two are asserted here.
