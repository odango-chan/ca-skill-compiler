> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP2 — v1alpha5 Node-entry Wait parser + Procedure AST

**Status:** candidate implementation on `feat/compiler-v1alpha5-wait-ast-20261117`
(PR #30, Draft). This is static Compiler source admission for a **candidate**
language, not language publication, product qualification, Host/Runtime
implementation or 5GC/MAS qualification. v1alpha4 remains the only published
authoring contract.

**Language target:** `connectionagent.io/v1alpha5` (ca-skills #61 candidate).

## 1. Scope delivered

| Area | Delivered |
|---|---|
| Wait grammar | Exact Appendix L.1 Node/Wait/event/response/routes field and type rules |
| FixedDuration | Exact Appendix L.1.1 lexical subset, no Host maximum check |
| Selector | Exact three-part non-empty YAML string selector, compared exactly |
| Response | Optional `response`; only field `after`; `after` itself is optional (default boundary = Wait-bearing Node completion per L.8.1); a present `after` must be a NodeId, not a terminal, not `context` |
| TIMEOUT route | Exactly one `TIMEOUT` key; NodeId or `FAILED`/`UNRESOLVED`/`SAFE_STOPPED`; `SUCCEEDED` forbidden |
| Stage rule | A wait-bearing Node must still declare ≥1 of the four business stages |
| Unknown fields | Normal `UNKNOWN_FIELD` at the offending key node |
| Diagnostics | Exact `WAIT_*` codes, `C_STATIC`, `error`, deterministic order, precise line/column |
| AST | Real v1alpha5 Procedure AST (`start`/`nodes`/`ByID`, raw stage nodes, Wait parsed in the same pass) |
| API | Strict independent v1alpha5 request/result/inspection version constants, `CheckV1Alpha5`, exact target dispatch, fail-closed Build residual gate |
| v1alpha4 | Byte-identical behavior; author Wait still rejected with the frozen `UNSUPPORTED_WAIT` |

## 2. Files

| File | Responsibility |
|---|---|
| `internal/compiler/v1alpha5_wait.go` | Wait/event/response/routes parser, duration grammar, `WAIT_*` diagnostics |
| `internal/compiler/v1alpha5_ast.go` | `V1Alpha5Target`, `V1Alpha5SkillAST`, `V1Alpha5ProcedureAST`, `V1Alpha5ProcedureNodeAST`; Wait parsed inside the Procedure pass |
| `internal/compiler/v1alpha5.go` | v1alpha5 source admission (`checkV1Alpha5Sources`), Build residual gate (`validateV1Alpha5`) |
| `internal/compiler/api_v1alpha5.go` | Independent API version constants, `validateBuildRequestV1Alpha5`, `CheckV1Alpha5` |
| `internal/compiler/api_v1alpha4.go` | Exact-target dispatch in public `Check`/`Build` before the v1alpha4 envelope validation |
| `internal/compiler/v1alpha5_wait_test.go` | Per-rule positive/negative diagnostics with exact Code/line/column, duration table, duplicate-key, determinism |
| `internal/compiler/v1alpha5_ast_test.go` | AST integration, spans, root/start/node-id rules, multi-Skill isolation, public Check positive |
| `internal/compiler/api_v1alpha5_test.go` | API admission, exact dispatch, fail-closed Build, v1alpha4 no-regression |
| `.github/workflows/ca-sl-v1alpha5-wait-ast.yml` | Runs the focused v1alpha5 tests plus full/vet/native/WASM gates |
| `docs/workstreams/wp2-wait-ast-v1alpha5.md` | This traceability record |

## 3. Clause → rule → diagnostic → test traceability

### 3.1 Appendix L.1 — Node-entry Wait grammar

| Clause rule | Diagnostic | Test |
|---|---|---|
| Node allowed fields exactly `wait`, `precondition`, `operation`, `observation`, `postcondition`; unknown field rejected at key node | `UNKNOWN_FIELD` | `TestV1Alpha5WaitDiagnosticAnchors/unknown_Node_field_anchors_at_key` |
| `wait` is optional but, when present, must be a mapping | `WAIT_DECLARATION_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/wait_not_a_mapping_anchors_at_value` |
| `wait` allowed fields exactly `event`, `timeout`, `response`, `routes`; no `poll`/`onTimeout`/`await`/`payloadSchema` | `UNKNOWN_FIELD` | `TestV1Alpha5WaitDiagnosticAnchors/unknown_wait_field_anchors_at_key` |
| `wait.event` required mapping with exactly `source`/`service`/`operation`; missing section anchored at Wait parent | `WAIT_DECLARATION_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/missing_event_anchors_at_wait_parent` |
| `wait.event` must be a mapping | `WAIT_DECLARATION_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/event_not_a_mapping_anchors_at_value` |
| Every selector field required, non-empty `!!str`, compared exactly (case/punctuation preserved) | `WAIT_DECLARATION_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/missing_event_source_anchors_at_event_parent`, `/event_selector_wrong_type_anchors_at_value`, `/event_selector_empty_string_anchors_at_value`, `TestV1Alpha5WaitSelectorValuesComparedExactly` |
| Unknown selector field rejected at key node | `UNKNOWN_FIELD` | `TestV1Alpha5WaitDiagnosticAnchors/unknown_event_field_anchors_at_key` |
| `wait.timeout` required, YAML string, exact L.1.1 grammar | `WAIT_TIMEOUT_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/missing_timeout_anchors_at_wait_parent`, `/non-string_timeout_anchors_at_value`, `/calendar_month_timeout_anchors_at_value`, `TestV1Alpha5FixedDurationLexicalSubset` |
| `wait.response` optional; when present only `after`; `after` is optional — absence means the default response boundary is the Wait-bearing Node's own completion (Appendix L.1 `WaitResponse = map(after?: NodeId)`; Appendix L.8.1 "when wait.response.after is absent" / omission is valid). A present `after` must be a NodeId, not a terminal and not `context`; extra key rejected at key node. A present non-mapping `response` is a Wait object shape violation (`WAIT_DECLARATION_INVALID`). | `UNKNOWN_FIELD`, `WAIT_DECLARATION_INVALID`, `WAIT_RESPONSE_BOUNDARY_INVALID` (present-but-invalid `after` only) | `TestV1Alpha5WaitResponseWithoutAfterUsesDefaultBoundary` (positive, `response: {}` + `After == ""`), `TestV1Alpha5WaitDiagnosticAnchors/response_not_a_mapping_is_a_Wait_shape_violation`, `/response.after_terminal_anchors_at_value`, `/response.after_context_anchors_at_value`, `/unknown_response_field_anchors_at_key`, `TestV1Alpha5WaitResponseBoundaryGrammar` |
| `wait.routes` required mapping with exactly one `TIMEOUT` key; extra author route key rejected | `WAIT_TIMEOUT_ROUTE_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/missing_routes_anchors_at_wait_parent`, `/routes_not_a_mapping_anchors_at_value`, `/EVENT_RECEIVED_extra_route_key_anchors_at_key`, `TestV1Alpha5WaitMissingRequiredSections` |
| `TIMEOUT` target is a NodeId or exactly `FAILED`/`UNRESOLVED`/`SAFE_STOPPED`; `SUCCEEDED`/other terminal forbidden | `WAIT_TIMEOUT_ROUTE_INVALID` | `TestV1Alpha5WaitDiagnosticAnchors/TIMEOUT_to_SUCCEEDED_anchors_at_value`, `TestV1Alpha5WaitPositiveRoutesWithoutResponse`, `TestV1Alpha5WaitTimeoutTargetGrammar` |
| A wait-bearing Node must declare ≥1 of the four business stages; Wait is not a stage | `WAIT_NODE_STAGE_REQUIRED` | `TestV1Alpha5WaitDiagnosticAnchors/wait-only_node_anchors_at_wait_mapping`, `TestV1Alpha5ProcedureASTParsesWaitInsideProcedurePass` |
| No fifth stage, Wait Node kind or `EVENT_RECEIVED` author route | `WAIT_TIMEOUT_ROUTE_INVALID` / `UNKNOWN_FIELD` | `TestV1Alpha5WaitDiagnosticAnchors/EVENT_RECEIVED_extra_route_key_anchors_at_key` |
| Malformed YAML / duplicate keys use the inherited parse diagnostic | `YAML_PARSE_ERROR` | `TestV1Alpha5WaitDuplicateYAMLKeyIsParseError` |
| Positive end-to-end admission with public `Check` and zero diagnostics | — | `TestV1Alpha5WaitPositiveProcedure`, `TestCheckV1Alpha5AcceptsValidProcedure` |

### 3.2 Appendix L.1.1 — FixedDuration lexical rules

| Rule | Test |
|---|---|
| Accepted: `PT30S`, `PT1M`, `PT5M`, `PT2H`, `P1D`, `P1DT2H30M`, `PT1H2S`, large positive components | `TestV1Alpha5FixedDurationLexicalSubset` (accepted table) |
| Rejected: empty, `P`, `PT`, `P1DT`, calendar `P1M`/`P1Y`/`P1W`, zero `PT0S`/`P0D`, leading zero `PT01M`/`P01D`, signs, decimals, lowercase, wrong component order, whitespace | `TestV1Alpha5FixedDurationLexicalSubset` (rejected table) |
| Host maximum duration is NOT a Compiler check (admission belongs to Host/Runtime) | Large-component accepted cases; see §6 deferred items |

### 3.3 Appendix L.2.1 / L.2.2 — control semantics and timeout route

| Rule | Test |
|---|---|
| Wait is entry control; a Wait-bearing Node is otherwise an ordinary business Node | `TestV1Alpha5ProcedureASTParsesWaitInsideProcedurePass` |
| Normal event edge is implicit; author `EVENT_RECEIVED` route is invalid | `TestV1Alpha5WaitDiagnosticAnchors/EVENT_RECEIVED_extra_route_key_anchors_at_key` |
| `TIMEOUT` cannot prove the Goal: `SUCCEEDED` forbidden | `TestV1Alpha5WaitDiagnosticAnchors/TIMEOUT_to_SUCCEEDED_anchors_at_value` |
| Exactly one `TIMEOUT` route; missing/extra rejected | `TestV1Alpha5WaitMissingRequiredSections`, anchors table |
| Undeclared NodeId timeout targets use `CONTROL_FLOW_DANGLING` | Deferred to WP5 (graph) — see §6 |

### 3.4 Appendix L.11 / L.11.1 — diagnostics

Implemented codes: `WAIT_DECLARATION_INVALID`, `WAIT_TIMEOUT_INVALID`,
`WAIT_TIMEOUT_ROUTE_INVALID`, `WAIT_NODE_STAGE_REQUIRED`,
`WAIT_RESPONSE_BOUNDARY_INVALID`, plus reused `UNKNOWN_FIELD`,
`YAML_PARSE_ERROR`, `PROCEDURE_START_MISSING`, `NODE_ID_RESERVED`,
`SOURCE_ROOT_INVALID`.

All are `Phase=C_STATIC`, `Severity=error`, build-blocking, and ordered by the
shared `sortDiagnostics` (path, line, column, phase, code, message). Messages use
the L.11.1 canonical templates adapted to the repository message style; the
frozen API envelope has no `messageKey` field, so template keys are documented
here rather than carried on the wire.

Anchor policy (asserted exactly by tests):

- extra field / unknown field / extra route key → offending key node;
- wrong YAML type / malformed scalar → offending value node;
- missing required field/section → nearest existing parent object
  (`wait` mapping, `event` mapping, `response` mapping, Procedure root for
  `start`).

### 3.5 §7 API and exact target selection

| Rule | Test |
|---|---|
| Independent `v1alpha5` request/result/inspection version constants | `TestV1Alpha5APIVersionsAreIndependent` |
| API version validated before target; wrong envelope rejected | `TestValidateBuildRequestV1Alpha5RejectsWrongAPIVersion` |
| Exact language target; no fallback/sniffing; unknown target fails closed through `LANGUAGE_TARGET_UNSUPPORTED` | `TestCheckDispatchesV1Alpha5TargetAndFailsClosedOnUnknownTarget` |
| Non-empty files, normalized paths, duplicate/case-folded paths, nil bytes, size/count limits | `TestValidateBuildRequestV1Alpha5RejectsEmptyFiles`, `.../RejectsUnnormalizedPath`, `.../RejectsDuplicatePath`, `.../RejectsNilBytes` |
| `Check` returns only real diagnostics; `Build` currently assembles the candidate artifact after zero-error validation (the WP2-era residual gate was retired by the WP8 artifact slice; see `docs/design/compiler-artifact-v1alpha5.md`) | `TestBuildV1Alpha5SucceedsAfterFullValidation`, `TestBuildV1Alpha5ReportsWaitDiagnosticsAndEmitsNoArtifact` |
| v1alpha4 keeps rejecting author Wait; no v1alpha5 diagnostics on the v1alpha4 path | `TestV1Alpha4TargetStillRejectsAuthorWait` |

### 3.6 Appendix L.12 executable-slice mapping

| Case | WP2 executable coverage |
|---|---|
| EW-013 | `WAIT_NODE_STAGE_REQUIRED` anchor test |
| EW-014 | `TIMEOUT -> SUCCEEDED` anchor test |
| EW-027 | FixedDuration rejected table |
| EW-033 | missing routes / missing TIMEOUT tests |
| EW-034 | `EVENT_RECEIVED` extra route-key test |
| EW-035 | event selector missing/empty/wrong-type tests |
| EW-036 | missing / non-string timeout tests |
| EW-030, EW-019 | deferred to WP6 (reachability/post-dominance); only NodeId/terminal/`context` identity is checked here |
| EW-012, EW-029 | deferred to WP4/WP6 |
| EW-001–EW-011, EW-015–EW-018, EW-020–EW-026, EW-028, EW-031, EW-032 | Host/Runtime/Integration or later Compiler work packages — NOT implemented, NOT executed |

## 4. API and Build behavior

- `Check(req)` dispatches on `req.LanguageTarget == V1Alpha5Target` **before**
  the v1alpha4 envelope validation, then calls `CheckV1Alpha5`.
- `Build(req)` dispatches the same way and returns
  `BuildResult.apiVersion = connectionagent.io/compiler-build-result/v1alpha5`
  in all outcomes.
- **Update (WP8 artifact integration, `dfad653`+):** the WP2-era internal
  residual gate was retired. After the integrated WP2/WP3/WP4/WP5/WP6 pipeline
  returns zero error diagnostics, Build assembles the candidate deterministic
  v1alpha5 artifact tree and returns `Success=true`; real diagnostics still block
  Build atomically with no output files. See
  [`docs/design/compiler-artifact-v1alpha5.md`](../design/compiler-artifact-v1alpha5.md)
  and [`docs/known-limitations-v1alpha5.md`](../known-limitations-v1alpha5.md).
- All other targets keep the v1alpha4 path byte-identically, including the
  frozen rejection of author Wait in v1alpha4.
- WP2 `CheckV1Alpha5` parses only `skills/<name>/assurance/procedure.yaml` as a
  consumed author source for Wait rules. Other YAML files in the package are
  parsed once (so malformed YAML surfaces) but their semantic rules are later
  work packages; `Build` stays fail-closed.

## 5. Local verification (executed)

Environment: Windows, Go `go1.26.6 windows/amd64`, `GOWORK=off`; a fresh
worktree at the commit containing this document. Node/Chromium parity was not
required or executed for this WP.

| Command | Result |
|---|---|
| `go test ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| `CGO_ENABLED=0 go build ./...` | PASS |
| `CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS |
| `go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS |
| `go test ./internal/compiler -count=1 -run 'V1Alpha5\|Wait' -v` | PASS |
| `git diff --check` | PASS (no whitespace errors) |

`dist/` build output is local only and not committed.

## 6. Deferred scope (explicit)

| Item | Owner / next step |
|---|---|
| `wait.event` → exact trusted Ingress Operation export resolution (`WAIT_INGRESS_EXPORT_NOT_FOUND` / `WAIT_INGRESS_EXPORT_AMBIGUOUS`) | WP1/WP3 (IngressCatalog + linking) |
| Trigger Operation resolution, correlation binding, `WAIT_CORRELATION_BINDING_INVALID`, armInputSchema | WP3 |
| `wait.output` SourcePath binding, normalized request Schema, path sensitivity, `SOURCE_NOT_AVAILABLE` / `SOURCE_PATH_INVALID` | WP4 |
| Wait entry-control vertex, `EVENT_RECEIVED`/`TIMEOUT` graph edges, DAG/reachability/dominance, terminal reachability, `CONTROL_FLOW_DANGLING` for undeclared TIMEOUT/`response.after` targets | WP5 |
| `response.after` reachability/post-dominance (both when present and for the default Wait-bearing-Node boundary when absent), response-boundary Producer and `responseResultSchema`, `WAIT_RESPONSE_BOUNDARY_INVALID` reachability cases, `WAIT_RESPONSE_MAPPING_INVALID` | WP6 |
| EW-001–EW-036 conformance fixtures/diagnostic index/messageKey mapping | WP7 |
| Native/WASM/Chromium parity for v1alpha5 API | WP8 — delivered by the WP8 artifact slice (see `docs/workstreams/wp8-parity-v1alpha5.md`) |
| Host maximum duration admission (`WAIT_DURATION_LIMIT_EXCEEDED`, not C_STATIC) | Host/Runtime contract |
| `response.after` self-target/cycle and `TIMEOUT` self-target detection | WP5 graph (parsed lexically today only as NodeId) |
| Full v1alpha5 Skill package requirements (`SKILL.md`, `contract.yaml`, `predicates.yaml`, schemas, dependencies) and artifact assembly | Artifact assembly delivered by the WP8 artifact slice for the implemented pipeline; remaining candidate package obligations are recorded in `docs/known-limitations-v1alpha5.md` |

## 7. CI status

The `ca-sl-v1alpha5-wait-ast` workflow was updated to run the focused
`-run 'V1Alpha5'` tests plus full tests, vet, native and WASM builds. No GitHub
Actions runner result was observed in this session; CI is therefore
**NOT_RUN / CI_UNAVAILABLE**, never claimed as PASS. All claims above are local
execution evidence only.
