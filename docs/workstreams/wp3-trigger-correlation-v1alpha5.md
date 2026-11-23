> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP3 — v1alpha5 Ingress exact link and trigger/correlation binding

**Status:** implemented on branch `feat/compiler-v1alpha5-wp3-trigger-binding-20261117`
(stacked on WP5; contains the WP1 merge). Candidate/DRAFT contract sources; not a
language or product qualification claim.

**Authority**

- ca-skills #61 candidate single-file spec: Appendix L.3.1, L.3.2, L.4.1, L.4.2,
  L.11, L.11.1.
- ca-skills #62 Integration Ingress Operation contract: §6 (correlation and
  pre-dispatch arming), §7 (concurrency/stale delivery), §10 (Compiler linking and
  static checks), §12 (minimal IngressCatalog structure), §13 (Ingress diagnostic
  ownership/precedence).
- Ingress diagnostic registry `connectionagent.io/ingress-diagnostics/v1alpha1`
  (candidate/DRAFT).
- Goal doc WP3 row: “每条 Wait 入边绑定精确 ToolCatalog Operation；推导 typed
  armInputSchema；验证 Invocation Context 和所有入边一致性”.

## 1. Implemented semantics

### 1.1 Trusted dependency scope and trusted-only Catalog discovery

For one selected v1alpha5 Skill with at least one Wait-bearing Node (dependency
snapshots and Contract dependencies are optional inputs; when absent the trusted
export set is empty and every Wait selector resolves to zero trusted exports):

1. Contract `dependencies:` (alias → Stable Package ID) are read through the
   inherited v1alpha4 Contract parser (`collectContractDependencies`); no second
   dependency or YAML parser is introduced.
2. Every declared alias that is supplied as a `ResolvedDependency` is proven to be
   `alias → one snapshot → measured original-byte digest → one
   TrustedDependencyBinding → Stable Package ID` (the WP3A machinery), and the
   trusted binding must be present (fail-closed).
3. Integration package snapshots are scanned for `IngressCatalog` v1alpha1
   documents. A Catalog is **trusted** only when its `metadata.owner` is an
   authorized Owner of the verified snapshot binding **and** every declared
   operation id has an exact trusted `exportKind: ingressOperation` export-owner
   record with that same Owner. Any other Catalog is completely suppressed: it
   produces no export and no Ingress diagnostics (#62 §13).
4. Trusted Catalogs are admitted through the WP1 `AdmitCatalog` implementation,
   which verifies the declared protocol artifact digest from raw bytes, the Schema
   Closure digests, export-id/event-selector uniqueness and the copied
   IngressCatalog v1alpha1 structural schema. No digest is accepted from the caller.

### 1.2 Exact Wait selector linking (L.3.1)

Each Wait `event` selector `(source, service, operation)` is resolved by exact
string equality against the trusted export set:

- zero matches → `WAIT_INGRESS_EXPORT_NOT_FOUND`;
- more than one match → `WAIT_INGRESS_EXPORT_AMBIGUOUS`;
- exactly one match → that export is the linked Ingress Operation export.

No similar-name search, newest-export selection, untrusted-body inspection or
global EventCatalog fallback exists. The diagnostic message uses the L.11.1
template with `{nodeId}` and `{selector}` = `source/service/operation`, anchored at
the `wait.event` mapping.

### 1.3 Correlation and arming declaration validation (L.3.2, #62 §6/§7)

For the linked export the WP3 pass derives the typed correlation contract through
the trusted v1alpha5 Schema bindings and the Schema Closure Digest v1
implementation:

- `request.normalizedSchemaRef` and `correlation.armInputSchemaRef` must be declared
  in `TrustedBuildContext.schemaBindings` and must produce a valid Schema Closure
  Digest v1 closure;
- every `correlation.keyFields[].requestPath` must exist in the normalized request
  Schema through the exact `$ref`-following Schema lookup and must resolve to a
  stably comparable scalar Schema (`string`/`integer`/`number`/`boolean`, or a
  scalar-only `enum`/`const`; multi-type Schemas are rejected as ambiguous);
- every arming binding must cover exactly one key field (`requestKey` ↔
  `requestPath`), its `bindingPath` must exist in the armInputSchema as a scalar,
  and its `source.kind` must be the exact mode (`invocation-context` for start
  arming, `operation-argument` for triggers);
- `staleDelivery` must be one of `single-flight-tombstone` /
  `wire-discriminator`, declare all fixed-duration windows in the L.1.1 duration
  subset, and enforce `retention > maximumLateArrival + clockSkewAllowance`
  strictly (#62 §7.2/§7.3); a `single-flight-tombstone` `resourceKey` and a
  `normalized-request-path` discriminator path must resolve in the normalized
  Schema;
- a `wire-discriminator` strategy requires its exact declared source kind and
  basisRef; `authenticated-header-parameter` keeps the catalog-validated
  `3gpp-Sbi-Request-Info`/`idempotency-key` restriction.

Export-side failures use the Ingress-owned identity
`INGRESS_CORRELATION_INVALID` with the registry message template, and suppress the
dependent Skill-path diagnostics (no cascade). Missing/invalid trusted Schemas use
the existing `SCHEMA_MISSING` / `SCHEMA_INVALID` identities, and dependency
identity failures use the existing `DEPENDENCY_*` identities (#62 §13 ownership
table).

### 1.4 Exact trigger Operation identity (L.4.2, #62 §6.2/§13)

Each `arming.triggers[].operation` is resolved by exact Package ID + trusted Owner
+ `kind=operation` + exact Operation ID against the ToolCatalog export index of the
trusted snapshot:

- the declared `packageId` must match exactly one verified dependency binding’s
  Stable Package ID;
- the Operation ID is matched against the ToolCatalog export id. Both exact forms
  shown by the candidate contract are accepted: the package-local export id and the
  alias-qualified author reference `<dependencyAlias>::<exportID>` whose alias is
  the already-matched dependency; nothing else is guessed;
- the trusted Owner attribution reuses the inherited v1alpha4
  `buildSnapshotExportIndex` / `attributeExportOwner` machinery (no second
  ToolCatalog parser); the resolved Owner must equal the declared trigger Owner;
- the resolved export’s Input Schema identity must be declared in the trusted
  v1alpha5 Schema bindings.

Failures use the existing `EXPORT_ID_MISSING` / `EXPORT_ID_AMBIGUOUS` /
`DEPENDENCY_OWNER_MISMATCH` identities — never an Ingress Wait code — anchored at
the catalog snapshot for export-side declarations.

### 1.5 Per-incoming-path trigger dominance (L.4.2, #62 §6.3)

Using the WP5 Procedure graph (Wait entry-control vertices, dominators, start
reachability):

- non-start Wait Node: for every reachable incoming path (every reachable
  predecessor of the Node’s Wait entry vertex), exactly one Operation Node whose
  tool reference matches a declared trigger must dominate that predecessor. Zero or
  more than one dominating declared trigger means the path cannot prove one exact
  binding;
- the dominating trigger Node’s materialized `operation.arguments` must contain
  every declared `source.path` value, and that value must not be the trigger’s own
  `operation.output` (unavailable before dispatch);
- the Wait Node’s own operation and any downstream Node can never satisfy the rule,
  because they cannot dominate the Wait entry;
- start-Node Wait: the export must declare `arming.start.allowed: true` with a
  complete start binding set; Runtime arms it from the already-admitted Invocation
  Context before Node activation.

Failures use `WAIT_CORRELATION_BINDING_INVALID` with the L.11.1 template, anchored
at `wait.event`.

## 2. Candidate v1alpha5 trusted-context API decisions

`TrustedBuildContext` (shared struct) gains three v1alpha5-only fields:

| Field | Shape | Decision |
|---|---|---|
| `snapshotBaseURI` | string | Canonical base URI ending with `/`; required whenever any binding is present; WP1 derives each artifact’s document URI from it. |
| `artifactBindings[]` | `{artifactRef, dependencyId, snapshotPath, dialect}` | Exact 1:1 binding of an artifact identity to immutable dependency snapshot bytes. `dialect ∈ {openapi-3.0, json-schema-2020-12}`. No digest: WP1 always recomputes it from raw bytes. |
| `schemaBindings[]` | `{exportId, artifactRef, pointer}` | Exact trusted Schema export → artifact + canonical root pointer (`#` or `#/...`). This is the only way `normalizedSchemaRef`, `armInputSchemaRef` and ToolCatalog Input Schema identities resolve. |

Envelope rules: unique `artifactRef`, unique `(dependencyId, snapshotPath)`, existing
dependency and snapshot path, existing `artifactRef` for every Schema binding,
canonical pointer, unique `exportId`, required `snapshotBaseURI`. Violations are
`REQUEST_INVALID`. The frozen v1alpha4 envelope validation **explicitly rejects**
any non-empty value of these three fields, so the v1alpha4 API stays closed.

Deliberate scope decisions (documented, not faked):

- Every Wait-bearing Node runs the exact selector resolution. If the request is
  well-formed but supplies no trusted dependency/catalog (including a
  dependency-free request), zero trusted Ingress Operation exports match and
  `WAIT_INGRESS_EXPORT_NOT_FOUND` is emitted at `wait.event`; the Wait is never
  silently accepted. Only a failed dependency identity / trusted-context
  prerequisite suppresses the dependent selector/trigger diagnostics (the
  prerequisite diagnostic itself is still emitted). Build remains fail-closed
  separately.
- WP2/WP5 syntax/AST/graph tests assert their own parser/graph layer
  (`parseV1Alpha5SkillAST` + `validateV1Alpha5ProcedureGraph`) and are not
  dependency-free public-`Check` positives; the end-to-end public `Check` positive
  path uses the complete WP3 trusted fixture.
- Ingress/trigger diagnostics produced while checking dependency-side declarations
  carry the dependency snapshot path as `Diagnostic.path`; the full versioned
  diagnostic envelope with `registryId`/`messageKey`/`sourceLocation` is WP7.

## 3. Diagnostics implemented

| Code | Phase/Severity | Trigger in this slice | Anchor |
|---|---|---|---|
| `WAIT_INGRESS_EXPORT_NOT_FOUND` | C_STATIC / error | selector matches zero trusted Ingress Operation exports | `wait.event` |
| `WAIT_INGRESS_EXPORT_AMBIGUOUS` | C_STATIC / error | selector matches more than one trusted export | `wait.event` |
| `WAIT_CORRELATION_BINDING_INVALID` | C_STATIC / error | no unique dominating trigger on some reachable incoming path; trigger argument unavailable or depends on its own output; start Wait without start arming | `wait.event` |
| `INGRESS_CORRELATION_INVALID` | C_STATIC / error | invalid export-side correlation/arming/stale-delivery declaration | catalog snapshot |
| `SCHEMA_MISSING` / `SCHEMA_INVALID` | C_STATIC / error | unbound/undigestible trusted Schema, Schema Closure failure | catalog snapshot |
| `EXPORT_ID_MISSING` / `EXPORT_ID_AMBIGUOUS` | C_STATIC / error | trigger ToolCatalog Operation identity unresolved/ambiguous | catalog snapshot |
| `DEPENDENCY_*` (existing) | C_STATIC / error | dependency snapshot/binding/Stable Package/owner identity failure; suppresses dependent Ingress/Wait diagnostics | dependency scope |
| `REQUEST_INVALID` | C_STATIC / error | malformed v1alpha5-only trusted-context bindings; v1alpha4 use of them | envelope |

No new diagnostic Code is invented.

## 4. Clause → executable test traceability

Tests live in `internal/compiler/v1alpha5_binding_test.go`. All fixtures are
clearly labelled synthetic; digests are derived from the fixture bytes by the same
WP1 Schema Closure Digest v1 code.

| Clause / case | Test |
|---|---|
| L.3.2, #62 §10/§12 positive trusted link (IN-001 type positive path) | `TestV1Alpha5WP3PositiveTrustedIngressLink` |
| L.3.1 well-formed Wait with zero trusted exports, including a dependency-free request (IN-018 / EW-015) | `TestV1Alpha5WaitWithoutIngressExportIsNotFound` |
| L.3.1 zero match with supplied trusted dependencies (IN-018 / EW-015) | `TestV1Alpha5WP3SelectorNotFound` |
| L.3.1 multi match (IN-019 / EW-015) | `TestV1Alpha5WP3SelectorAmbiguous` |
| #62 §13 trusted-only discovery / untrusted suppression (IN-020/021) | `TestV1Alpha5WP3UntrustedCatalogSuppressed`, `TestV1Alpha5WP3UntrustedOwnerSuppressed` |
| #62 §13 prerequisite suppression + Schema Closure digest (IN-004/063) | `TestV1Alpha5WP3SchemaClosureDigestMismatch` |
| L.4.2 exact trigger identity missing (IN-003) | `TestV1Alpha5WP3TriggerOperationMissing` |
| L.4.2 / §13 ambiguous trigger identity | `TestV1Alpha5WP3TriggerOperationAmbiguous` |
| L.4.2 exact Owner scoping | `TestV1Alpha5WP3TriggerOwnerMismatch` |
| L.3.2 typed armInputSchema missing | `TestV1Alpha5WP3MissingArmInputSchema` |
| L.3.2 / #62 §6.1 key field path + scalar type | `TestV1Alpha5WP3MissingKeyFieldPath`, `TestV1Alpha5WP3NonScalarKeyField` |
| #62 §6.2 bindingPath ↔ armInputSchema | `TestV1Alpha5WP3BindingPathMissing` |
| L.4.2 per-incoming-path dominance (IN-063 / EW-016/018) | `TestV1Alpha5WP3TriggerNotDominatingOnOnePath` |
| L.4.2 pre-dispatch materialization | `TestV1Alpha5WP3TriggerArgumentsMissing` |
| L.4.2 start-Node arming | `TestV1Alpha5WP3StartWaitWithoutArmingStart`, `TestV1Alpha5WP3StartWaitWithCompleteStartArming`, `TestV1Alpha5WP3IncompleteStartArming` |
| #62 §7.2/§7.3 retention window | `TestV1Alpha5WP3RetentionWindowInvalid` |
| v1alpha5-only trusted-context envelope | `TestV1Alpha5WP3TrustedContextBindingValidation` |
| v1alpha4 closed API | `TestV1Alpha5WP3V1Alpha4RejectsV1Alpha5OnlyTrustedFields` |
| dependency identity prerequisite suppression | `TestV1Alpha5WP3DependencyIdentityFailureSuppressesLinking` |
| end-to-end public `Check` positive (API layer) | `TestCheckV1Alpha5AcceptsValidProcedure` |
| Build residual gate with a fully valid trusted fixture + dependency-free fail-closed | `TestBuildV1Alpha5FailsClosedAfterASTValidation` |

## 5. Explicitly deferred

- Normalization mapping completeness/typing (source→normalized field coverage,
  transform validity) and peer-profile reference resolution: Ingress-owned
  `INGRESS_NORMALIZATION_INVALID` / `INGRESS_PEER_PROFILE_REFERENCE_INVALID`
  remain WP7/later.
- Response boundary and mapping (WP6): `WAIT_RESPONSE_BOUNDARY_INVALID`,
  `WAIT_RESPONSE_MAPPING_INVALID`; the WP1 projection now exposes the response
  section additively for that slice.
- `wait.output` SourcePath availability/typing (WP4).
- Full Invocation Context Schema typing for start arming: source paths are
  validated as canonical JSON Pointers; the referenced Host Context/Request
  Schema typing and assignability are deferred (no Host Context Schema is part of
  the candidate sources yet).
- Trigger Operation Input Schema property typing/assignability against the
  materialized argument values: the export identity is resolved and its Input
  Schema must be trusted, but argument-value type checking is deferred alongside
  WP4/WP7 type linking.
- `canonical-request-digest` / duplicate identity semantics beyond the structural
  schema constraints.
- Real Owner/Integration Catalog artifacts and 5GC profile fixture pins remain
  outside the compiler repository; every fixture here is synthetic and is not an
  Owner publication or interoperability claim.
- Conformance mapping (WP7), artifact assembly (WP8), WASM/Chromium parity (WP8).
