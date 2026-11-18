# WP7 - v1alpha5 Diagnostics / Conformance traceability (post-integration)

**Status:** Draft traceability record on the integrated v1alpha5 baseline `feat/compiler-v1alpha5-integration-20261117`.
**Language target:** `connectionagent.io/v1alpha5` (FROZEN / RELEASED by ca-skills PR #66). IngressCatalog v1alpha1 and its diagnostic registry are FROZEN / RELEASED by PR #62.

This record classifies every Appendix L.12 `EW-001`-`EW-036` and released #62 section 14 `IN-001`-`IN-066` case against the **integrated** compiler tree (WP1-WP8 merged). It does not add language semantics and does not qualify the product.

Hard limits stated up front:

- **No row is CI PASS or qualification.** CI did not execute any step for this branch; only the local commands in section 7 were executed and their exact results are recorded there.
- 51 rows are `COMPILER_STATIC_IMPLEMENTED_TESTED` and name an executable Go test. Rows not naming a test are **not** executed or claimed.
- `HOST_RUNTIME_INTEGRATION_OWNED` rows are Host/Runtime/Integration obligations. They are **not Compiler obligations** and are not implemented, executed or claimed by this work package.
- `COMPILER_STATIC_DEFERRED` rows name the remaining owning work package and, where a normative identity exists, the expected `(Code, Phase, Severity)`. Some deferred rows document an identity/coverage divergence rather than a missing code path.
- Real Owner artifacts, real Integration wire conformance and 5GC/MAS qualification remain **NOT_QUALIFIED**.

## 1. Classification vocabulary

| Classification | Meaning |
|---|---|
| `COMPILER_STATIC_IMPLEMENTED_TESTED` | Implemented in the integrated v1alpha5 static path and covered by an executable Go test named in the row. |
| `COMPILER_STATIC_DEFERRED` | A Compiler-owned static obligation that this tree does not implement or does not cover with an executable assertion; the row names the remaining owner and the expected identity. |
| `HOST_RUNTIME_INTEGRATION_OWNED` | Host / Runtime / Integration obligation (or an architecture boundary). Not a Compiler obligation; not executed here. |
| `SPECIFICATION_ONLY_NOT_EXECUTED` | Normative obligation with no dedicated diagnostic identity and no executable fixture. Never a PASS. |

## 2. Classification counts

| Set | Implemented + tested | Deferred | Host/Runtime/Integration | Spec-only | Total |
|---|---|---|---|---|---|
| EW-001..EW-036 | 16 | 1 | 18 | 1 | 36 |
| IN-001..IN-066 | 35 | 4 | 27 | 0 | 66 |
| **All** | **51** | **5** | **45** | **1** | **102** |

Deferred rows by remaining owner: WP1 = 1, WP3 = 0, WP5 = 0, WP6 = 4.

## 3. EW-001..EW-036 matrix (Appendix L.12)

| Case | Owner layer | Classification | Expected result / diagnostic | Executable test |
|---|---|---|---|---|
| EW-001 | Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Same Run resumes once; wait.output is available before the first business stage. | NOT_COMPILER_OWNED (external obligation) |
| EW-002 | Host/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Host durably buffers the event; consume it only if that exact Node is selected. | NOT_COMPILER_OWNED (external obligation) |
| EW-003 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Never create a new Invocation and never attach the event to a future Wait. | NOT_COMPILER_OWNED (external obligation) |
| EW-004 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Integration rejects; the Wait remains pending and the timer is not reset. | NOT_COMPILER_OWNED (external obligation) |
| EW-005 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject/quarantine unless the exact pinned export defines the normalized representation. | NOT_COMPILER_OWNED (external obligation) |
| EW-006 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject/quarantine as ambiguous; never choose the newest match. | NOT_COMPILER_OWNED (external obligation) |
| EW-007 | Integration/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | A stale prior delivery cannot satisfy the later Wait; require a wire discriminator or a proven single-flight/tombstone rule. | NOT_COMPILER_OWNED (external obligation) |
| EW-008 | Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Exactly one durable atomic winner between event acceptance and Wait timeout. | NOT_COMPILER_OWNED (external obligation) |
| EW-009 | Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Apply global Invocation stop semantics; do not route the TIMEOUT edge. | NOT_COMPILER_OWNED (external obligation) |
| EW-010 | Host/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Idempotent disposition; no second Node entry. | NOT_COMPILER_OWNED (external obligation) |
| EW-011 | Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Cannot reopen the Wait or populate wait.output. | NOT_COMPILER_OWNED (external obligation) |
| EW-012 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | wait.output referenced on a timeout/bypass path must emit the inherited SOURCE_NOT_AVAILABLE diagnostic. | `TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable` |
| EW-013 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Wait-bearing Node with no business stage emits WAIT_NODE_STAGE_REQUIRED. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-014 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Wait TIMEOUT route targeting SUCCEEDED emits WAIT_TIMEOUT_ROUTE_INVALID. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-015 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Zero matches emit WAIT_INGRESS_EXPORT_NOT_FOUND; multiple matches emit WAIT_INGRESS_EXPORT_AMBIGUOUS. | `TestV1Alpha5WP3SelectorNotFound`, `TestV1Alpha5WP3SelectorAmbiguous` |
| EW-016 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | One incoming path that cannot derive the correlation before the trigger emits WAIT_CORRELATION_BINDING_INVALID. | `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| EW-017 | Integration/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Without a wire discriminator reject the second arming until tombstone expiry; with one, identify the old request as already consumed and never deliver it to Wait N+1. | NOT_COMPILER_OWNED (external obligation) |
| EW-018 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Different incoming paths resolving to different Ingress exports/Schemas are rejected with WAIT_CORRELATION_BINDING_INVALID; authors must use distinct Wait Nodes. | `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| EW-019 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A normal accepted path that bypasses response.after emits WAIT_RESPONSE_BOUNDARY_INVALID. (The WP2 parser only rejects a present-but-syntactically-invalid after; path reachability/dominance is not EW-019 coverage.) | `TestV1Alpha5WP6SuccessBeforeBoundary` |
| EW-020 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Integration emits the declared failure response; Runtime does not fabricate a response. | NOT_COMPILER_OWNED (external obligation) |
| EW-021 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Body-bearing paired response without a typed Producer mapping emits WAIT_RESPONSE_MAPPING_INVALID. | `TestV1Alpha5WP6BoundaryNodeWithoutProducerStage` |
| EW-022 | Compiler/Integration | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A single fixed bodyless paired response disposition completes without a typed response-result Producer (Compiler acceptance rule, WP6); Integration delivers the bodyless response and fabricates nothing. | `TestV1Alpha5WP6PositiveSingleBodylessDisposition` |
| EW-023 | Host/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Durable relay/recovery delivers the same deliveryRef after restart; no new Invocation. | NOT_COMPILER_OWNED (external obligation) |
| EW-024 | Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Runtime checkpoint resumes the same Node exactly once from the durable checkpoint. | NOT_COMPILER_OWNED (external obligation) |
| EW-025 | Host/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Host/Integration recover the exact response obligation or apply the declared failure policy. | NOT_COMPILER_OWNED (external obligation) |
| EW-026 | Compiler | `SPECIFICATION_ONLY_NOT_EXECUTED` | Non-conforming: Event Wait and Evidence Wait remain distinct; no dedicated L.11 diagnostic identity exists and no executable EW-026 fixture is implemented. Polling-only author fields are structurally rejected as the inherited UNKNOWN_FIELD identity by WP2, but that is not EW-026 semantic coverage. | NOT_EXECUTED |
| EW-027 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Malformed, fractional, zero, calendar-month or leading-zero timeout emits WAIT_TIMEOUT_INVALID. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-028 | Host admission | `HOST_RUNTIME_INTEGRATION_OWNED` | Host admission rejects before Invocation creation with WAIT_DURATION_LIMIT_EXCEEDED (Appendix L.11.2, not C_STATIC); no clamping. | NOT_COMPILER_OWNED (external obligation) |
| EW-029 | Compiler | `COMPILER_STATIC_DEFERRED` | Body-bearing response mapping that selects wait.output, Context or raw request data instead of the declared boundary Producer emits WAIT_RESPONSE_MAPPING_INVALID. | NOT_IMPLEMENTED (remaining WP6) |
| EW-030 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | response.after naming an unknown or unreachable Node emits WAIT_RESPONSE_BOUNDARY_INVALID. | `TestV1Alpha5GraphUnknownResponseAfterTarget`, `TestV1Alpha5WP6UnreachableBoundary` |
| EW-031 | Compiler/Runtime | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A global Precondition that is not satisfied on every incoming path to a protected Mutation/success is rejected with the inherited ENTRY_PRECONDITION_BYPASSED identity; a Wait edge never establishes the gate. | `TestV1Alpha5EW031EntryPreconditionBypass` |
| EW-032 | Runtime/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Do not dispatch the triggering Operation when expected-ingress arming cannot be acknowledged. | NOT_COMPILER_OWNED (external obligation) |
| EW-033 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Missing wait.routes mapping or missing TIMEOUT key emits WAIT_TIMEOUT_ROUTE_INVALID. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-034 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An author-declared EVENT_RECEIVED or any extra key under wait.routes emits WAIT_TIMEOUT_ROUTE_INVALID; the normal event edge remains implicit. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-035 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | event.source/service/operation empty or wrong YAML type emits WAIT_DECLARATION_INVALID. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-036 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Missing wait.timeout or a timeout that is not a YAML string emits WAIT_TIMEOUT_INVALID. | `TestV1Alpha5DiagnosticContractEWAnchors` |

### 3.1 Implemented EW rows: exact diagnostic identity

`Phase=C_STATIC`, `Severity=error`, build-blocking. The compiler.Diagnostic envelope has no `messageKey`/`sourceLocation` wire field; the normative mapping is a package-local registry in `internal/compiler/v1alpha5_diagnostic_contract_test.go`, asserted against the actually emitted messages. The JSON envelope is asserted unchanged by `TestV1Alpha5DiagnosticEnvelopeUnchanged`.

| EW | Code | messageKey | sourceLocation rule | Executable test |
|---|---|---|---|---|
| EW-012 | `SOURCE_NOT_AVAILABLE` | `-` | Compiler Diagnostic.path = the trusted dependency snapshot path of the IngressCatalog snapshot. The canonical field path is rendered into the message; the full registry sourceLocation record is not on the frozen compiler.Diagnostic wire. | `TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable` |
| EW-013 | `WAIT_NODE_STAGE_REQUIRED` | `wait.node.stage_required` | Line/column of the `wait` mapping value node of the Wait-bearing Node. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-014 | `WAIT_TIMEOUT_ROUTE_INVALID` | `wait.timeout.route.invalid` | Line/column of the offending route key/target value node, the `wait.routes` value node, or the `wait` mapping when missing. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-015 | `WAIT_INGRESS_EXPORT_NOT_FOUND` | `wait.ingress.export.not_found` | Line/column of the `wait.event` selector node. | `TestV1Alpha5WP3SelectorNotFound` |
| EW-016 | `WAIT_CORRELATION_BINDING_INVALID` | `wait.correlation.binding.invalid` | Line/column of the `wait.event` selector node. | `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| EW-018 | `WAIT_CORRELATION_BINDING_INVALID` | `wait.correlation.binding.invalid` | Line/column of the `wait.event` selector node. | `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| EW-019 | `WAIT_RESPONSE_BOUNDARY_INVALID` | `wait.response.boundary.invalid` | Line/column of the `wait.response.after` value node, else the `response`/`wait` node. | `TestV1Alpha5WP6SuccessBeforeBoundary` |
| EW-021 | `WAIT_RESPONSE_MAPPING_INVALID` | `wait.response.mapping.invalid` | Line/column of the `wait.response.after` value node, else the `response`/`wait` node. | `TestV1Alpha5WP6BoundaryNodeWithoutProducerStage` |
| EW-022 | (acceptance row; no diagnostic) | - | - | `TestV1Alpha5WP6PositiveSingleBodylessDisposition` |
| EW-027 | `WAIT_TIMEOUT_INVALID` | `wait.timeout.invalid` | Line/column of the `wait.timeout` value node; nearest existing parent (`wait` mapping) when missing. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-030 | `WAIT_RESPONSE_BOUNDARY_INVALID` | `wait.response.boundary.invalid` | Line/column of the `wait.response.after` value node, else the `response`/`wait` node. | `TestV1Alpha5GraphUnknownResponseAfterTarget` |
| EW-033 | `WAIT_TIMEOUT_ROUTE_INVALID` | `wait.timeout.route.invalid` | Line/column of the offending route key/target value node, the `wait.routes` value node, or the `wait` mapping when missing. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-034 | `WAIT_TIMEOUT_ROUTE_INVALID` | `wait.timeout.route.invalid` | Line/column of the offending route key/target value node, the `wait.routes` value node, or the `wait` mapping when missing. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-035 | `WAIT_DECLARATION_INVALID` | `wait.declaration.invalid` | Line/column of the offending `event`/`response` value node; nearest existing parent (`event`, else `wait` mapping) when missing. | `TestV1Alpha5DiagnosticContractEWAnchors` |
| EW-036 | `WAIT_TIMEOUT_INVALID` | `wait.timeout.invalid` | Line/column of the `wait.timeout` value node; nearest existing parent (`wait` mapping) when missing. | `TestV1Alpha5DiagnosticContractEWAnchors` |

## 4. IN-001..IN-066 matrix (released #62 section 14)

| Case | Owner layer | Classification | Expected result / diagnostic | Executable test |
|---|---|---|---|---|
| IN-001 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Resolve Ingress only in declared dependencies; zero matches emit WAIT_INGRESS_EXPORT_NOT_FOUND / C_STATIC; never guess from a global catalog. | `TestV1Alpha5WaitWithoutIngressExportIsNotFound`, `TestV1Alpha5WP3PositiveTrustedIngressLink` |
| IN-002 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Selector with no matching export emits WAIT_INGRESS_EXPORT_NOT_FOUND / C_STATIC. | `TestV1Alpha5WP3SelectorNotFound` |
| IN-003 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Selector matching multiple exports emits WAIT_INGRESS_EXPORT_AMBIGUOUS / C_STATIC. | `TestV1Alpha5WP3SelectorAmbiguous` |
| IN-004 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Missing or mismatching Owner binding emits DEPENDENCY_EXPORT_OWNER_UNVERIFIED or DEPENDENCY_OWNER_MISMATCH (reused dependency identities); no Ingress alias. | `TestV1Alpha5WP3IngressExportOwnerMismatch`, `TestV1Alpha5WP3IngressExportOwnerRecordMissing` |
| IN-005 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Duplicate export id emits INGRESS_EXPORT_ID_DUPLICATE / C_STATIC. | `TestCatalogDuplicateExportID` |
| IN-006 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | OpenAPI artifact raw-byte digest mismatch emits the reused DEPENDENCY_IDENTITY_UNVERIFIED identity; must not emit INGRESS_PROTOCOL_ARTIFACT_INVALID. | `TestCatalogProtocolArtifactIdentityUnverified` |
| IN-007 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Mixing 3GPP Service-Operation names with the pinned OpenAPI operationId emits INGRESS_PROTOCOL_ARTIFACT_INVALID / C_STATIC. | `TestCatalogProtocolMetadataMismatch` |
| IN-008 | Compiler/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | A wire method/path/API-version mismatch is rejected by Integration; Catalog metadata contradicting the trusted OpenAPI artifact emits INGRESS_PROTOCOL_ARTIFACT_INVALID / C_STATIC (IN-057). | NOT_COMPILER_OWNED (external obligation) |
| IN-009 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject an unauthenticated peer or a peer NF type not equal to event.source; the Wait is not satisfied. | NOT_COMPILER_OWNED (external obligation) |
| IN-010 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject an unauthorized peer calling an authenticated API; the Wait is not satisfied. | NOT_COMPILER_OWNED (external obligation) |
| IN-011 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject an unsupported media type. | NOT_COMPILER_OWNED (external obligation) |
| IN-012 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject a failing source request Schema; do not reset the Wait timeout. | NOT_COMPILER_OWNED (external obligation) |
| IN-013 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Missing or unsupported normalized Schema emits SCHEMA_MISSING or SCHEMA_INVALID (reused identities). | `TestCatalogMissingSchemaClosure` |
| IN-014 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An inconsistent request normalization declaration emits INGRESS_NORMALIZATION_INVALID / C_STATIC. | `TestNormalizationDuplicateTarget`, `TestNormalizationMultipartBoundsInconsistent` |
| IN-015 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A required wire field not mapped to the normalized Schema emits INGRESS_NORMALIZATION_INVALID / C_STATIC. | `TestNormalizationRequiredNormalizedPropertyUncovered` |
| IN-016 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A multipart/binary part without a restricted immutable reference and lifecycle declaration emits INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID / C_STATIC; runtime payload-reference mismatch remains an Integration reject/quarantine. | `TestBinaryContentReferenceSchemaUnresolved` |
| IN-017 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Host correlation metadata leaking into wait.output is an Integration conformance failure. | NOT_COMPILER_OWNED (external obligation) |
| IN-018 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A trigger callable that is not an exact ToolCatalog Operation emits EXPORT_ID_MISSING or EXPORT_ID_AMBIGUOUS in the exact Package ID + Owner scope. | `TestV1Alpha5WP3TriggerOperationMissing`, `TestV1Alpha5WP3TriggerOperationAmbiguous` |
| IN-019 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A trigger argument that depends on the trigger's own not-yet-executed output emits SOURCE_NOT_AVAILABLE / C_STATIC; the dependent correlation binding is also rejected. | `TestV1Alpha5WP3TriggerSelfOutputUnavailable` |
| IN-020 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A start Wait without a verifiable correlation Context binding emits WAIT_CORRELATION_BINDING_INVALID / C_STATIC. | `TestV1Alpha5WP3StartWaitWithoutArmingStart` |
| IN-021 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An incoming path that cannot uniquely derive the trigger Operation/correlation binding emits WAIT_CORRELATION_BINDING_INVALID / C_STATIC. | `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| IN-022 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject/quarantine a request matching two active expectedIngressRef values; never select the newest. | NOT_COMPILER_OWNED (external obligation) |
| IN-023 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An Ingress-own declaration without a provable wire discriminator or single-flight/tombstone stale-delivery policy emits INGRESS_CORRELATION_INVALID / C_STATIC; a selected Skill path that cannot prove the binding emits WAIT_CORRELATION_BINDING_INVALID. | `TestV1Alpha5WP3RetentionWindowInvalid`, `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| IN-024 | Integration/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject a stale message received after the Wait timed out; it satisfies neither the old nor a new Wait. | NOT_COMPILER_OWNED (external obligation) |
| IN-025 | Host | `HOST_RUNTIME_INTEGRATION_OWNED` | A retried identical deliveryRef gets an idempotent disposition and no repeated Node entry. | NOT_COMPILER_OWNED (external obligation) |
| IN-026 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject/quarantine two indistinguishable independent requests under one Wait. | NOT_COMPILER_OWNED (external obligation) |
| IN-027 | Compiler | `COMPILER_STATIC_DEFERRED` | A paired response missing required deadline/recovery/disposition declarations emits INGRESS_RESPONSE_CONTRACT_INVALID / C_STATIC; a complete declaration that the Adapter does not honor is an Integration conformance failure. | NOT_IMPLEMENTED (remaining WP6) |
| IN-028 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A single fixed bodyless paired response disposition does not require a response-result Producer; multiple dispositions still require a typed result selector. | `TestV1Alpha5WP6PositiveSingleBodylessDisposition` |
| IN-029 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A body-bearing paired response without a result Producer emits WAIT_RESPONSE_MAPPING_INVALID / C_STATIC. | `TestV1Alpha5WP6BoundaryNodeWithoutProducerStage` |
| IN-030 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A response.after Node without the export-required Producer stage emits WAIT_RESPONSE_BOUNDARY_INVALID / C_STATIC; the same Node also has no valid typed Producer-to-response mapping (EW-021). | `TestV1Alpha5WP6BoundaryNodeWithoutProducerStage` |
| IN-031 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A boundary Producer Output not assignable to responseResultSchema emits INPUT_TYPE_MISMATCH / C_STATIC; an unavailable Producer SourcePath emits SOURCE_NOT_AVAILABLE; WAIT_RESPONSE_MAPPING_INVALID must not substitute for ordinary type diagnostics. | `TestV1Alpha5WP6ProducerOutputSchemaIncompatible`, `TestV1Alpha5WP6ProducerOutputSchemaUnavailable` |
| IN-032 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An Ingress-own response mapping that misses required protocol response fields emits INGRESS_RESPONSE_MAPPING_INVALID / C_STATIC; a missing Skill Producer mapping emits WAIT_RESPONSE_MAPPING_INVALID. | `TestV1Alpha5WP6MappingCoverageIncomplete` |
| IN-033 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A Run ending in an allowed failure terminal before the response boundary while the export declares no valid early-failure response mapping emits INGRESS_RESPONSE_CONTRACT_INVALID / C_STATIC; with a complete declaration the execution half is an Integration conformance responsibility. | `TestV1Alpha5WP6EarlyFailureMappingMissing` |
| IN-034 | Integration/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | An expired response deadline must not be routed as a CA-SL Wait TIMEOUT Outcome. | NOT_COMPILER_OWNED (external obligation) |
| IN-035 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | An unrecoverable HTTP/2 stream after restart is handled by the declared recovery/failure policy; never assume the handle survives. | NOT_COMPILER_OWNED (external obligation) |
| IN-036 | Host/Runtime | `HOST_RUNTIME_INTEGRATION_OWNED` | Architecture conformance failure: Host/Runtime must not parse raw 3GPP HTTP requests. | NOT_COMPILER_OWNED (external obligation) |
| IN-037 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Architecture conformance failure: Integration must not heuristically select a Run/Node. | NOT_COMPILER_OWNED (external obligation) |
| IN-038 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Integration must reject re-arming immediately after a closing Wait while the tombstone is unexpired. | NOT_COMPILER_OWNED (external obligation) |
| IN-039 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | acceptedMediaTypes, sourceSchemas and normalization.representations not in one-to-one correspondence emit INGRESS_NORMALIZATION_INVALID / C_STATIC. | `TestNormalizationMediaTypeCorrespondence`, `TestNormalizationDuplicateSourceMediaType` |
| IN-040 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Duplicate response disposition id or HTTP status in one Ingress export emits INGRESS_RESPONSE_CONTRACT_INVALID / C_STATIC. | `TestV1Alpha5WP6DuplicateDispositionID`, `TestV1Alpha5WP6DuplicateDispositionStatus` |
| IN-041 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | After EVENT_ACCEPTED, a duplicate old request before tombstone expiry or a premature next arming must be rejected/quarantined and never delivered to the next Wait. | NOT_COMPILER_OWNED (external obligation) |
| IN-042 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Request binaryContent retention shorter than the last declared recovery window emits INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID / C_STATIC. | `TestBinaryContentRetentionBelowRecoveryWindow` |
| IN-043 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject an over-limit multipart body/part before unbounded buffering; do not submit the normalized delivery. | NOT_COMPILER_OWNED (external obligation) |
| IN-044 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A pinned OpenAPI response requiring multipart/related content while the Ingress export declares no multipart representation emits INGRESS_RESPONSE_MAPPING_INVALID / C_STATIC; binary parts are never dropped or stringified. | `TestPinnedOpenAPIResponseMultipartRequiresMultipartMapping` |
| IN-045 | Compiler | `COMPILER_STATIC_DEFERRED` | A Step 15 user-plane setup rejection requiring an N1 SM Reject binary container while the export declares only JSON or 204 emits INGRESS_RESPONSE_MAPPING_INVALID / C_STATIC; a missing N1 binary mapping stays NOT LINKABLE. | NOT_IMPLEMENTED (remaining WP6) |
| IN-046 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A staleDelivery authenticated-header-parameter discriminator whose duplicate identity source/scope is not the same 3gpp-sbi-idempotency-key target-nf identity emits INGRESS_CORRELATION_INVALID / C_STATIC. | `TestV1Alpha5WP3WireDiscriminatorDuplicateIdentityInvalid` |
| IN-047 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A discriminatorRetention less than or equal to maximumLateArrival + clockSkewAllowance emits INGRESS_CORRELATION_INVALID / C_STATIC. | `TestV1Alpha5WP3RetentionWindowInvalid` |
| IN-048 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Never send an over-limit body; apply the declared protocol failure policy; no truncation or silent omission. | NOT_COMPILER_OWNED (external obligation) |
| IN-049 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject/quarantine a mismatched, missing, duplicate or dangling multipart Content-ID; no normalized delivery. | NOT_COMPILER_OWNED (external obligation) |
| IN-050 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject/quarantine an application/json representation with an unresolvable RefToBinaryData; no fabricated binary reference. | NOT_COMPILER_OWNED (external obligation) |
| IN-051 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Treat a reused idempotency-key against the same target NF as already consumed, regardless of peer/export. | NOT_COMPILER_OWNED (external obligation) |
| IN-052 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Reject configurations that use Max-Rsp-Time or timestamps as a request discriminator or maximumLateArrival. | NOT_COMPILER_OWNED (external obligation) |
| IN-053 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Recompute the private attemptResponseDeadline for a retried transport attempt; keep the original responseObligationRef, deliveryRef and expectedIngressRef; create no new Wait or business execution. | NOT_COMPILER_OWNED (external obligation) |
| IN-054 | Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | An earlier attemptResponseDeadline only ends that attempt; the original response obligation and Run are not reset; a new attempt may attach to the original obligation under the fixed logical responseDeadline. | NOT_COMPILER_OWNED (external obligation) |
| IN-055 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Missing, wrong or unsupported Catalog apiVersion/kind/root structure emits INGRESS_CATALOG_INVALID / C_STATIC. | `TestCatalogInvalidAPIVersionAndKind` |
| IN-056 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | Two different export IDs declaring the same exact event.source + event.service + event.operation emit INGRESS_EVENT_SELECTOR_DUPLICATE / C_STATIC. | `TestCatalogDuplicateEventSelector` |
| IN-057 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A trusted OpenAPI artifact whose release, API URI version, method, path or operationId contradicts the Catalog declaration (after byte-digest verification) emits INGRESS_PROTOCOL_ARTIFACT_INVALID / C_STATIC. | `TestCatalogProtocolMetadataMismatch` |
| IN-058 | Compiler | `COMPILER_STATIC_DEFERRED` | Peer profile reference **format** validation is implemented (`INGRESS_PEER_PROFILE_REFERENCE_INVALID / C_STATIC`, `TestPeerProfileReferenceInvalid`); resolution to an exact expected profile resource remains deferred because the released #62 contract publishes no profile resource representation or identity format, and deployment binding is Host Admission. | PARTIAL (format implemented; resource resolution deferred) |
| IN-059 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A missing, duplicate, incomplete or invalid-target/transform normalization mapping for an accepted media representation emits INGRESS_NORMALIZATION_INVALID / C_STATIC; invalid source/normalized Schemas still use the Schema diagnostics. | `TestNormalizationDanglingTarget`, `TestNormalizationTransformTargetTypeMismatch` |
| IN-060 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An invalid Ingress-own correlation key/duplicate identity/scope or staleDelivery declaration emits INGRESS_CORRELATION_INVALID / C_STATIC; an unprovable Skill path emits WAIT_CORRELATION_BINDING_INVALID. | `TestV1Alpha5WP3WireDiscriminatorDuplicateIdentityInvalid` |
| IN-061 | Compiler | `COMPILER_STATIC_DEFERRED` | A missing or contradictory Ingress response mode/deadline policy/disposition/failure response/delivery recovery contract emits INGRESS_RESPONSE_CONTRACT_INVALID / C_STATIC. | NOT_IMPLEMENTED (remaining WP6) |
| IN-062 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | An invalid Ingress-own response representation/Schema mapping/multipart part mapping/required field mapping emits INGRESS_RESPONSE_MAPPING_INVALID / C_STATIC; a Skill Producer mapping failure emits WAIT_RESPONSE_MAPPING_INVALID. | `TestV1Alpha5WP6MappingCoverageIncomplete`, `TestV1Alpha5WP6MultipartBinaryPartUnmapped` |
| IN-063 | Compiler | `COMPILER_STATIC_IMPLEMENTED_TESTED` | A Trigger ToolCatalog Operation missing or ambiguous in the exact Package ID + Owner scope emits EXPORT_ID_MISSING or EXPORT_ID_AMBIGUOUS; no Ingress or Wait selector diagnostic. | `TestV1Alpha5WP3TriggerOperationMissing`, `TestV1Alpha5WP3TriggerOperationAmbiguous` |
| IN-064 | Skill/Runtime/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Step 15 user-plane failure variants must not duplicate N4 Release; Step 17 response is not delayed, and Step 18 cleans remaining N4 / PDU address / PCF resources. | NOT_COMPILER_OWNED (domain procedure / runtime / integration obligation) |
| IN-065 | Skill/Runtime/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Do not unsubscribe shared SUPI / DNN / HPLMN S-NSSAI data while another matching PDU Session remains. | NOT_COMPILER_OWNED (domain procedure / runtime / integration obligation) |
| IN-066 | Skill/Runtime/Integration | `HOST_RUNTIME_INTEGRATION_OWNED` | Unsubscribe when no other matching PDU Session remains; keep Step 21 subscription cleanup distinct from Step 18 resource cleanup. | NOT_COMPILER_OWNED (domain procedure / runtime / integration obligation) |

## 5. Normative diagnostic code sets used by this record

**Appendix L.11 v1alpha5 static codes (9):** all implemented by the integrated tree: `WAIT_DECLARATION_INVALID`, `WAIT_TIMEOUT_INVALID`, `WAIT_INGRESS_EXPORT_NOT_FOUND`, `WAIT_INGRESS_EXPORT_AMBIGUOUS`, `WAIT_CORRELATION_BINDING_INVALID`, `WAIT_TIMEOUT_ROUTE_INVALID`, `WAIT_NODE_STAGE_REQUIRED`, `WAIT_RESPONSE_BOUNDARY_INVALID`, `WAIT_RESPONSE_MAPPING_INVALID`.

**Appendix L.11.2 Host admission code (not `C_STATIC`, not a Compiler obligation):** `WAIT_DURATION_LIMIT_EXCEEDED`.

**Ingress diagnostic registry v1alpha1 codes emitted by this tree (8):** `INGRESS_CATALOG_INVALID`, `INGRESS_EXPORT_ID_DUPLICATE`, `INGRESS_EVENT_SELECTOR_DUPLICATE`, `INGRESS_PROTOCOL_ARTIFACT_INVALID`, `INGRESS_CORRELATION_INVALID`, `INGRESS_RESPONSE_CONTRACT_INVALID`, `INGRESS_RESPONSE_MAPPING_INVALID`, `INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID`. The registry-only identities `INGRESS_PEER_PROFILE_REFERENCE_INVALID` and `INGRESS_NORMALIZATION_INVALID` are not emitted yet (deferred WP1 rows).

**Inherited frozen v1alpha4 identities explicitly reused by L.11:** `YAML_PARSE_ERROR`, `UNKNOWN_FIELD`, `CONTROL_FLOW_DANGLING`, `CONTROL_FLOW_CYCLE`, `SOURCE_PATH_INVALID`, `SOURCE_NOT_AVAILABLE`, `INPUT_TYPE_MISMATCH`, `SCHEMA_MISSING`, `SCHEMA_INVALID`, `DEPENDENCY_EXPORT_OWNER_UNVERIFIED`, `DEPENDENCY_OWNER_MISMATCH`, `DEPENDENCY_IDENTITY_UNVERIFIED`, `EXPORT_ID_MISSING`, `EXPORT_ID_AMBIGUOUS`, `ENTRY_PRECONDITION_BYPASSED`. No `WAIT_*` alias is invented for them.

## 6. Deferred and external ownership summary

| Owner | Remaining scope in this record | Rows |
|---|---|---|
| WP1 | IngressCatalog request normalization, request binary-content lifecycle, peer-profile resolution; not implemented by the WP1 admission slice | IN-014, IN-015, IN-016, IN-039, IN-042, IN-058, IN-059 |
| WP3 | trigger-argument availability identity alignment, idempotency-key discriminator, full correlation/duplicate declaration matrix, exact Owner-binding negative fixture | IN-004, IN-019, IN-046, IN-060 |
| WP5 | v1alpha5 static global-Precondition-bypass/dominance obligation | EW-031 |
| WP6 | typed-mapping negative fixture, boundary-vs-mapping identity alignment, duplicate-disposition fixture, deadline/recovery contract identity, pinned-OpenAPI response cross-checks | EW-029, IN-027, IN-030, IN-040, IN-044, IN-045, IN-061 |

External obligations (Skill/Host/Runtime/Integration; not Compiler-owned static obligations): EW-001, EW-002, EW-003, EW-004, EW-005, EW-006, EW-007, EW-008, EW-009, EW-010, EW-011, EW-017, EW-020, EW-023, EW-024, EW-025, EW-028, EW-032, IN-008, IN-009, IN-010, IN-011, IN-012, IN-017, IN-022, IN-024, IN-025, IN-026, IN-034, IN-035, IN-036, IN-037, IN-038, IN-041, IN-043, IN-048, IN-049, IN-050, IN-051, IN-052, IN-053, IN-054, IN-064, IN-065, IN-066.

## 7. Executed local evidence

CI: **CI_UNAVAILABLE / no test steps executed**. GitHub Actions run `44091765010` concluded failure, but its job had zero executed steps; no Go test/build command ran. Local commands recorded by Harness on 2026-11-17 (Windows PowerShell, `go1.26.6 windows/amd64`, `GOWORK=off`, `CGO_ENABLED=0`):

Local toolchain: `go version go1.26.6 windows/amd64`, `GOWORK=off`, `CGO_ENABLED=0`.

| Command | Result |
|---|---|
| `go test ./... -count=1` | **PASS (exit 0)**: all 9 test packages `ok`, including `internal/compiler 5.256s` and `internal/ingresscatalog 2.049s`; 2 packages have no test files. Local execution only, not CI. |
| `go vet ./...` | **PASS (exit 0)**, no findings. |
| `CGO_ENABLED=0 go build ./...` | **PASS (exit 0)**, no output. |
| `go test ./internal/compiler -count=1 -run 'V1Alpha5|Conformance' -v` | **PASS (exit 0)**: 115 top-level tests passed, 0 failures; `ok internal/compiler 1.102s`. |
| `git diff --check` | **PASS (exit 0)**, no whitespace errors. |

No row is thereby converted to CI PASS, language publication or product qualification.

## 8. Limitations

- This is a traceability and diagnostic-contract slice for the v1alpha5 Compiler implementation. The v1alpha5 language and Ingress contracts are already FROZEN / RELEASED; Compiler and product qualification remain incomplete.
- `EW-026` has no dedicated diagnostic identity in Appendix L.11 and no executable fixture; it is recorded as `SPECIFICATION_ONLY_NOT_EXECUTED`.
- 19 rows remain deferred with the remaining owner recorded; several rows document an identity/coverage divergence between the current implementation and the normative matrix rather than a missing code path.
- No real Owner artifact, Integration Adapter wire conformance, Host/Runtime recovery or 5GC/MAS qualification is executed or implied. Those remain NOT_QUALIFIED.
