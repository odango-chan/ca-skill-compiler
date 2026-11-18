> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP6 — v1alpha5 response boundary and typed response mapping

**Status:** implemented on branch
`feat/compiler-v1alpha5-wp6-response-boundary-20261117` (stacked on the corrected
WP3 branch `feat/compiler-v1alpha5-wp3-trigger-binding-20261117`, which contains
WP1, WP2, WP3 and WP5). Candidate/DRAFT contract sources; not a language, product,
Host/Runtime or Integration qualification claim.

**Authority**

- ca-skills #61 candidate single-file spec: Appendix L.8.1 (boundary selection and
  post-dominance), L.8.2 (typed response-result binding), L.8.3 (response
  completion/ownership), L.11 (`WAIT_RESPONSE_BOUNDARY_INVALID`,
  `WAIT_RESPONSE_MAPPING_INVALID`) and L.11.1 message templates, L.11.2 reuse of
  existing SourcePath/type diagnostics.
- ca-skills #62 Integration Ingress Operation contract: §7 (concurrency/duplicate/
  late delivery) and §8 (paired protocol response: mode, boundary and Producer
  source, response Schema and dispositions, representations incl. JSON,
  multipart root part and binary parts, body required/none, responseResultSchema,
  `response.result.producer`, `response.result.mappings`,
  `dispositionSelector`, `earlyFailureResponses`, `binaryContent` lifecycle,
  `deliveryRecovery`, deadline policy).
- ca-skills #62 Ingress diagnostic registry v1alpha1 (candidate/DRAFT):
  `INGRESS_RESPONSE_CONTRACT_INVALID`, `INGRESS_RESPONSE_MAPPING_INVALID`,
  `INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID`.
- Goal doc WP6 row: “`wait.response.after` 精确 Producer stage、Producer output
  Schema、responseResultSchema、response dispositions、JSON / multipart / binary
  映射”.
- EW cases: EW-019 (normal accepted path bypasses `response.after`), EW-020
  (mapped early failure disposition), EW-021 (body-bearing response lacks typed
  Producer mapping), EW-022 (bodyless declared status may complete without a
  response-result Producer), EW-029 (mapping selects `wait.output`/Context/raw
  request data), EW-030 (`response.after` names an unknown/unreachable Node).

## 1. Implemented semantics

The WP6 pass runs inside the WP3 orchestration after the exact trusted Ingress
Operation export link and its correlation/arming contract were proven, and before
Build’s residual fail-closed gate. It reuses the WP3 trusted link context, the
trusted dependency scope, the ToolCatalog export index, the trusted Schema binding
map and the Schema Closure Digest v1 machinery; no second parser, linker, catalog
or Schema implementation is introduced.

### 1.1 Response mode and boundary selection (L.8.1, #62 §8.2)

- Export `response.mode: none` (or any non-`paired` mode): authoring
  `wait.response` is invalid → `WAIT_RESPONSE_BOUNDARY_INVALID` anchored at the
  authored `wait.response` mapping. No `wait.response` is valid.
- Export `response.mode: paired`: the response boundary is
  - the exact Node named by `wait.response.after` when present, else
  - the Wait-bearing Node’s own completion (default).
- The boundary Node must be a declared Node reachable from the event-accepted edge
  (the Wait-bearing Node’s business vertex). A declared-but-unreachable
  `wait.response.after` → `WAIT_RESPONSE_BOUNDARY_INVALID`. An unknown
  `wait.response.after` is already diagnosed by the WP5 graph pass with the same
  identity and is not duplicated.

### 1.2 Post-dominance and Producer availability (L.8.1, #62 §8.2)

The pass proves the boundary over the WP5 Procedure graph (`Vertices`,
`EdgesFrom`, `PostDominates`) with a deterministic `(vertex, completed)` traversal
from the event-accepted edge:

- `completed` becomes true only when the boundary Node’s **last declared business
  stage** routes its **normal Outcome** (`precondition`/`operation`/`observation`/
  `postcondition` normal outcome per the inherited stage auto-advance semantics).
  Leaving the boundary Node through any earlier or non-normal route means the
  boundary (and its Producer stage) was not completed on that path.
- Every event-accepted path must either reach that completion or terminate earlier
  in `FAILED`, `UNRESOLVED` or `SAFE_STOPPED` with an explicit trusted
  `response.earlyFailureResponses` mapping for that exact disposition.
- A path that reaches `SUCCEEDED` before the completion is invalid →
  `WAIT_RESPONSE_BOUNDARY_INVALID` (EW-019).
- A permitted early terminal without an explicit trusted mapping is an invalid
  Ingress-owned early-failure policy → `INGRESS_RESPONSE_CONTRACT_INVALID` at
  `/response/earlyFailureResponses` (EW-020). The mapping itself is owned by the
  reusable Integration export, so the defect is not reported as a Skill-side
  mapping failure.
- `PostDominates` is used as the graph-level proof and for the defensive
  non-post-dominating/no-terminal-path case; the earlier-exit distinction is
  modelled by the completion state, because the WP5 graph vertex represents the
  Node (not its internal stage sequence).

### 1.3 Typed response-result Producer binding (L.8.2, #62 §8.2/§8.3)

For a paired export that requires a typed result (any `body: required`
disposition, or more than one disposition):

- the Producer stage type is taken **only** from the export’s
  `response.result.producer` (`operation` | `observation`); the Skill cannot
  declare a NodeId/SourcePath inside the reusable export;
- the response SourcePath is derived deterministically as
  `<response-boundary-node>.<producer>.output`;
- the boundary Node must declare that exact business stage. If the default
  boundary (the Wait-bearing Node) cannot declare it, the export requires a later
  boundary and omitting `wait.response.after` is
  `WAIT_RESPONSE_BOUNDARY_INVALID`; if an explicit boundary Node cannot declare it,
  it is `WAIT_RESPONSE_MAPPING_INVALID` (EW-021);
- the boundary stage’s qualified Tool reference is resolved through the same
  trusted ToolCatalog export index and Owner attribution used by WP3; the exact
  ToolCatalog Operation/observation export must declare the trusted Schema
  binding used as its Output Schema. Missing/ambiguous/untrusted/missing-Schema
  cases reuse `SOURCE_NOT_AVAILABLE` (L.11.2), not a response-mapping code;
- the Producer Output Schema is materialized from the trusted snapshot and its
  Schema Closure is verified, then proven assignable to the export’s
  `responseResultSchema` (also closure-verified) with the frozen v1alpha4 F.5
  Source-Subset-Target assignability engine. An ordinary incompatibility reuses
  `INPUT_TYPE_MISMATCH`; an unresolved/unavailable Schema uses
  `SOURCE_NOT_AVAILABLE`/`SCHEMA_*` (L.11.2). This is not a Skill-specific
  response mapping defect (EW-029 covers mapping `from` selecting
  `wait.output`/Context/raw request data, which is impossible by construction
  because every mapping `from` is resolved inside the trusted
  `responseResultSchema`).

A single fixed `body: none` disposition with no typed `response.result` is the
L.8.2 exemption and requires no Producer stage (EW-022). Multiple bodyless
dispositions still require the typed result + `dispositionSelector`.

### 1.4 Ingress-owned response contract and representation/mapping validation (#62 §8.3/§8.4)

The projected response contract is validated once per trusted export (cached) and
failures suppress dependent Skill-side response diagnostics (#62 §13 precedence):

- `response.dispositions[]`: non-empty; unique `id` and unique `status` (the same
  status must be modelled as representations of one disposition);
  `body: required` needs representations, `body: none` must not have any;
- `responseResultSchemaRef` must resolve through the trusted v1alpha5 Schema
  bindings to a valid Schema Closure Digest v1 closure (`SCHEMA_MISSING` /
  `SCHEMA_INVALID` otherwise);
- `response.result.mappings[]`: each `dispositionId`/`representationId` must name
  a body-bearing disposition/representation; `from` must resolve as a JSON
  Pointer inside `responseResultSchema`; the closed target set is checked —
  `json-body` only for `application/json`, `multipart-json` only for the declared
  JSON root part of `multipart/related`, `multipart-binary` only for a declared
  binary part with a `contentIdField` string property inside the JSON root part;
  `transform` is `identity` except `binary-content-ref` for binary targets; target
  pointers must resolve in the exact protocol response Schema and duplicate
  targets are rejected; a conservative JSON type compatibility check is applied
  to JSON targets;
- coverage: every required protocol response field (recursively expanded through
  required object properties and exact local `$ref` indirection) of every
  body-bearing representation must be supplied by a mapping of that
  disposition/representation. The multipart JSON root part is covered by
  `multipart-json` mappings; the Integration-generated Content-ID property
  declared by `contentIdField` is excluded from the mapping obligation because
  §8.3 assigns it to Integration at serialization time;
- multipart/related: the root part is the single JSON part and its
  Schema/digest must equal the representation `responseSchemaRef`/digest; each
  declared binary part has exactly one `multipart-binary` mapping; no mapping
  references an undeclared part; `contentIdField` targets are unique; each part
  `maxBytes` must not exceed the representation lifecycle limits; declared binary
  part bounds must fit the declared parts;
- `response.binaryContent` is required for any multipart/related representation;
  its `referenceSchemaRef` must resolve in the trusted Schema bindings, and the
  immutable SHA-256/`integration-response-only`/retention/maxBytes lifecycle is
  checked (`INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID` on defects);
- `dispositionSelector` (required for multiple dispositions or multiple
  representations of a body-bearing disposition): its `sourcePath` must resolve
  to a closed `enum` in `responseResultSchema`, case values must be unique enum
  members, every allowed disposition/representation combination must be covered
  exactly once, bodyless dispositions must not select a representation and
  body-bearing ones must (`INGRESS_RESPONSE_CONTRACT_INVALID` on defects);
- malformed `response` shapes are already rejected by the copied IngressCatalog
  v1alpha1 structural schema (`INGRESS_CATALOG_INVALID`); this slice validates the
  cross-field/typed obligations the structural schema cannot express.

### 1.5 Projection extension

`internal/ingresscatalog` gains an additive-only response projection (no WP1
admission semantic change): dispositions → representations → `rootPart`/`parts`/
`minBinaryParts`/`maxBinaryParts`; typed `response.result.mappings` with their
closed targets and `contentIdField`; `dispositionSelector` cases;
`response.binaryContent`; `deliveryRecovery` fields. The WP1 projection of the
pre-existing fields is untouched.

## 2. Diagnostics implemented

| Code | Owner | Phase/Severity | Trigger in this slice | Anchor |
|---|---|---|---|---|
| `WAIT_RESPONSE_BOUNDARY_INVALID` | CA-SL v1alpha5 | C_STATIC / error | `wait.response` on an unpaired export; unknown/unreachable/RESPONSE.after; success before boundary; export-required later boundary omitted | `wait.response` / `wait` |
| `WAIT_RESPONSE_MAPPING_INVALID` | CA-SL v1alpha5 | C_STATIC / error | body-bearing/multi-disposition response whose boundary Node lacks the export’s Producer stage | `wait.response` / `wait` |
| `INGRESS_RESPONSE_CONTRACT_INVALID` | Ingress registry | C_STATIC / error | disposition set, early-failure policy, dispositionSelector coverage or resulting response contract invalid | catalog snapshot |
| `INGRESS_RESPONSE_MAPPING_INVALID` | Ingress registry | C_STATIC / error | representation/multipart part mapping, required-field coverage, target/`from` Pointer or binary-part mapping invalid | catalog snapshot |
| `INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID` | Ingress registry | C_STATIC / error | multipart response without a valid immutable `response.binaryContent` lifecycle | catalog snapshot |
| `INPUT_TYPE_MISMATCH` (existing) | reused | C_STATIC / error | Producer Output Schema not assignable to `responseResultSchema` (L.11.2) | boundary Producer tool |
| `SOURCE_NOT_AVAILABLE` (existing) | reused | C_STATIC / error | boundary Producer Tool/Output Schema unavailable or unresolved (L.11.2) | boundary Producer tool |
| `SCHEMA_MISSING` / `SCHEMA_INVALID` (existing) | reused | C_STATIC / error | unbound/undigestible trusted response Schema | catalog snapshot |

No new Code was invented: the three Ingress identities are the
`connectionagent.io/ingress-diagnostics/v1alpha1` entries with their normative
message keys/templates; the Skill-side failures reuse the two Appendix L.11
identities and the L.11.2 reuse rules.

`WAIT_RESPONSE_BOUNDARY_INVALID` message uses the normative L.11.1 template
`wait.response.boundary.invalid`; `WAIT_RESPONSE_MAPPING_INVALID` uses
`wait.response.mapping.invalid`; the Ingress diagnostics render the registry
templates exactly.

## 3. Clause → executable test traceability

Tests live in `internal/compiler/v1alpha5_response_test.go`. Every fixture is
clearly labelled synthetic; all Schema Closure Digests are derived from the exact
fixture bytes with the same WP1 Schema Closure Digest v1 implementation the
production path uses, and the WP1-resolved projection is exercised through the
public `Check` path.

| Clause / case | Test |
|---|---|
| L.8.2 single fixed bodyless disposition exemption (EW-022) | `TestV1Alpha5WP6PositiveSingleBodylessDisposition` |
| L.8.1/L.8.2 paired JSON body-bearing Producer + complete mapping (positive) | `TestV1Alpha5WP6PositiveJSONBodyProducer` |
| L.8.1 explicit `wait.response.after` boundary + post-dominance (positive) | `TestV1Alpha5WP6PositiveDeclaredBoundary` |
| #62 §8.3 multipart root part + binary part + Content-ID + binaryContent (positive) | `TestV1Alpha5WP6PositiveMultipartResponse` |
| #62 §8.3 multiple bodyless dispositions + complete dispositionSelector (positive) | `TestV1Alpha5WP6PositiveMultiBodylessSelector` |
| L.8.1 authoring `wait.response` on `mode: none` export | `TestV1Alpha5WP6WaitResponseOnUnpairedExport` |
| L.8.1 export-required later boundary omitted | `TestV1Alpha5WP6ExportRequiresLaterBoundary` |
| L.8.2 body-bearing response whose boundary Node lacks the Producer stage (EW-021) | `TestV1Alpha5WP6BoundaryNodeWithoutProducerStage` |
| L.8.1 success terminal before the boundary (EW-019) | `TestV1Alpha5WP6SuccessBeforeBoundary` |
| L.11/EW-030 `response.after` names an unreachable Node | `TestV1Alpha5WP6UnreachableBoundary` |
| L.8.1/§8.4 early-failure response policy missing (EW-020) | `TestV1Alpha5WP6EarlyFailureMappingMissing` |
| L.8.2/L.11.2 Producer Output Schema incompatible → `INPUT_TYPE_MISMATCH` | `TestV1Alpha5WP6ProducerOutputSchemaIncompatible` |
| L.11.2 Producer Output Schema unavailable → `SOURCE_NOT_AVAILABLE` | `TestV1Alpha5WP6ProducerOutputSchemaUnavailable` |
| #62 §8.3 mapping coverage incomplete (`INGRESS_RESPONSE_MAPPING_INVALID`) | `TestV1Alpha5WP6MappingCoverageIncomplete` |
| #62 §8.3 binary part unmapped (`INGRESS_RESPONSE_MAPPING_INVALID`) | `TestV1Alpha5WP6MultipartBinaryPartUnmapped` |
| #62 §8.3 dispositionSelector coverage invalid (`INGRESS_RESPONSE_CONTRACT_INVALID`) | `TestV1Alpha5WP6DispositionSelectorCoverageInvalid` |
| determinism of the WP6 pass | `TestV1Alpha5WP6Deterministic` |
| Build stays fail-closed after the WP6 static pass | `TestV1Alpha5WP6BuildRemainsFailClosed` |

## 4. Explicitly deferred / isolated (not faked)

- **Real wire response mapping is Integration-owned.** The Compiler validates the
  typed contract, disposition/representation/part mapping completeness and
  pointer existence in the trusted Schemas; it never executes an integration
  adapter, serializes a wire response, generates Content-IDs, validates the actual
  `multipart/related` boundary/framing bytes or performs HTTP `Accept` negotiation.
  Those stay Integration responsibilities per #62 §8.3/§8.5.
- **`response.deadlinePolicyRef` resolution** to a trusted deployment policy
  (logical `responseDeadline` vs per-attempt `attemptResponseDeadline`, §8.4) is
  Host Admission/deployment-owned. The Compiler projects the reference and lets
  IngressCatalog admission enforce its presence/shape; it does not invent a
  deployment-policy binding input.
- **`deliveryRecovery` durable recovery contract** resolution is Host/Integration
  owned (§8.5). Only the structural presence/shape is enforced.
- **`earlyFailureResponses` responseRef payloads / fixed error bodies** are
  Integration-package immutable artifacts validated against protocol Schemas by
  Ingress/Integration admission; the compiler checks the mapping presence/shape
  and disposition key, not the referenced payload.
- **Full F.5 assignability across the OpenAPI response dialect**: the mapping
  `from`→target JSON-type check is conservative (declared JSON type sets); the
  authoritative protocol response Schema validation remains the Integration
  adapter’s pinned-OpenAPI concern (§8.3).
- **`binaryContent.referenceSchemaRef` deep typing**: the reference Schema must be
  a trusted, closure-valid Schema binding; the typed binary reference is a
  Producer output field and its wire payloads are Integration-owned.
- **Real Owner/Integration IngressCatalog artifacts and 5GC profile pins** remain
  outside the compiler repository; all fixtures here are synthetic and are not an
  Owner publication or interoperability claim.
- **WP4 `wait.output` SourcePath**, WP7 diagnostics/conformance envelope and WP8
  artifact assembly/parity remain later work packages. Build stays fail-closed
  through the residual gate.

## 5. Exact verification commands and results

Environment: Windows 10/11, Go `go1.26.6 windows/amd64`; commands run from the
worktree root with `GOWORK=off` as required by `AGENTS.md`.

| Command | Result |
|---|---|
| `$env:GOWORK='off'; go test ./... -count=1` | PASS — all packages `ok`; WP6 tests included (`internal/compiler` `ok`) |
| `$env:GOWORK='off'; go vet ./...` | PASS — exit 0, no output |
| `$env:CGO_ENABLED='0'; $env:GOWORK='off'; go build ./...` | PASS — exit 0, no output |
| `$env:GOWORK='off'; go test ./internal/compiler -count=1 -run 'V1Alpha5' -v` | PASS — all prior WP2/WP3/WP5 and all 18 WP6 tests pass |
| `$env:GOWORK='off'; go test ./internal/compiler -count=1 -run 'V1Alpha5WP6' -v` | PASS — 18 tests, 0 failures |
| `git diff --check` | PASS — no whitespace errors |

No gate was skipped or unavailable. This is local static-compiler evidence only;
real Owner/Integration artifacts, Host admission, 5GC/MAS qualification and
Chromium parity remain out of scope / NOT_QUALIFIED.
