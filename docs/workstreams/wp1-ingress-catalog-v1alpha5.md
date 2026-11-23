> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP1 — IngressCatalog v1alpha1 admission and Schema Closure Digest v1

**Status:** candidate / DRAFT implementation slice. Not a frozen contract, not an
Owner publication, not deployment or 5GC qualification.

**Branch:** `feat/compiler-v1alpha5-wp1-ingress-catalog-20261117`
**Base:** `main` @ `b0ec4d04cba952522088d20b6e34358c1188b340`

## 1. Specification sources

| Source | Revision / path | Use |
|---|---|---|
| Goal doc (doc-only PR branch) | `origin/docs/next-compiler-plan-rebased-20261117` @ `02540a1fa300c9137b6b72f9a443226b53db8318`, `docs/workstreams/deepseek-harness-v1alpha5-implementation-goal.zh-CN.md` | WP1 definition, §7 API rules, §8 tests, §10/§11 reporting |
| Schema Closure Digest v1 | `ca-skills` PR #62 head `0fc6a0b0633a50afc478d7380accb46819c01afa`, `docs/design/schema-closure-digest-v1.md` | Normative algorithm implemented in `internal/ingresscatalog` |
| Ingress Operation Contract vnext | same PR head, `docs/design/integration-ingress-operation-contract-vnext.zh-CN.md` | IngressCatalog semantics (§7, §8, §10, §12, §13) |
| Ingress Diagnostic Registry v1alpha1 | same PR head, `docs/design/ingress-diagnostic-registry-v1alpha1.md` / `.json` | Exact diagnostic identities, precedence, messageKey, sourceLocation, ordering |
| IngressCatalog structural schema | same PR head, `docs/design/schemas/ingresscatalog.v1alpha1.schema.json` | Copied to `contracts/v1alpha1/ingresscatalog.schema.json` |
| Golden vector | same PR head, `docs/design/schemas/schema-closure-digest-v1.vector.json` | Copied to `internal/ingresscatalog/testdata/` |

`contracts/v1alpha4/` and the frozen v1alpha4 language contract are untouched.

## 2. Implemented surface

New self-contained package `internal/ingresscatalog` (package name
`ingresscatalog`). It imports the single normalized YAML AST
(`internal/compiler/yamlast`) and the pinned `jsonschema/v6` and
`gowebpki/jcs` dependencies; `github.com/gowebpki/jcs v1.0.2` was promoted from
an indirect to a direct requirement in `go.mod`. No v1alpha4 compiler file is
modified.

### 2.1 Schema Closure Digest v1

- `SnapshotArtifact` — immutable artifact record: `ArtifactRef`,
  `SnapshotPath`, optional `DocumentURI`, `Dialect`
  (`openapi-3.0` | `json-schema-2020-12`), `Raw`, `Parsed`, optional
  declared `ArtifactDigest`.
- `NewResolutionContext(snapshotBaseURI, artifacts)` validates the snapshot base
  URI (absolute, hierarchical, trailing `/`), derives each canonical
  `DocumentURI` by percent-encoding the snapshot path and resolving it against
  the base, binds `ArtifactRef` / `SnapshotPath` / `DocumentURI` 1:1, computes
  `SHA-256(raw)` and rejects a stale declared digest, parses the raw bytes with
  the normalized YAML AST and rejects a supplied `Parsed` value that differs
  from that parse. It then indexes dialect-aware Schema locations, nested
  `$id` resource-URI aliases and OpenAPI 3.0 Schema Object locations.
- `ResolveRoot(RootRef)` resolves an exact `(artifactRef, pointer)` root; no
  name search, fallback, floating lookup or network access exists.
- `ComputeSchemaClosure(RootRef)` returns `ClosureResult{Root, Nodes, Manifest,
  JCS, Digest}`: exact root + transitive `$ref` closure, one node per exact
  `(artifactRef, pointer)` identity (cycles allowed, visited once), array sorted
  by UTF-8 `artifactRef` then `pointer`, exact parsed Schema value per node,
  JCS (RFC 8785) manifest, and
  `sha256:` + hex of `SHA-256(ASCII "CA-SKILL-SCHEMA-CLOSURE\0v1\0" || JCS)`.
- Dialect traversal: OpenAPI 3.0 `properties`, single Schema `items`,
  Schema-valued `additionalProperties`, `not`, `allOf`, `anyOf`, `oneOf`
  (boolean `additionalProperties` contributes no child; tuple `items` rejected);
  JSON Schema 2020-12 `$defs`, `definitions`, `properties`,
  `patternProperties`, `dependentSchemas`, `additionalProperties`,
  `unevaluatedProperties`, `propertyNames`, `items`, `unevaluatedItems`,
  `contains`, `not`, `if`, `then`, `else`, `contentSchema`, `prefixItems`,
  `allOf`, `anyOf`, `oneOf`.
- Fragments: empty or JSON Pointer only; percent-decoded exactly once as strict
  UTF-8; RFC 6901 escaping validated; canonicalized to `#` / `#/...`; nested
  `$id` fragment pointers are translated to artifact-relative pointers (the
  golden vector expects `#/$defs/Shared/$defs/Leaf`).
- Fail-closed rejections: reachable OpenAPI `$id`, `$anchor`, `$dynamicRef`,
  `$dynamicAnchor`, `$recursiveRef`, Boolean Schemas, non-Schema targets,
  missing pointers, unmapped/ambiguous/out-of-snapshot targets, duplicate YAML
  mapping keys, non-string object keys, unsupported tags, non-finite numbers and
  integers outside `±(2^53-1)`.

### 2.2 IngressCatalog v1alpha1 admission

- `AdmitCatalog(CatalogInput, *TrustedSnapshot) *Admission` parses the Catalog
  bytes strictly through the normalized YAML AST (duplicate-key rejection),
  validates against the copied 2020-12 structural schema with an in-memory
  loader only (`offlineLoader`; the meta-schema is resolved from the pinned
  library's embedded copy), then runs the implemented semantic checks.
- Semantic checks: export ID uniqueness, exact event-selector tuple
  uniqueness, trusted protocol artifact byte-digest/metadata verification
  (`operationId`, HTTP method, path against the pinned OpenAPI artifact), and
  verification of every declared `(schemaRef, schemaDigest)` pair against a
  supplied `TrustedSnapshot` Schema-export binding map by recomputing Schema
  Closure Digest v1.
- Typed deterministic projection for WP3 linking: `Catalog` → `Operation` →
  `EventSelector`, `Protocol` / `OpenAPIProtocol`, `RequestProjection` /
  `SchemaBinding`; plus `DeclaredSchemaBindings` (with exact source pointers)
  and `VerifiedSchemaClosures`.
- Package-local `Diagnostic` record with `Code`, `Phase=C_STATIC`,
  `Severity=ERROR`, `RegistryID`, `MessageKey`, stable message,
  `SourceLocation{dependencyAlias, snapshotPath, jsonPointer}` and
  `RelatedLocations`; deterministic ordering by numeric precedence, then
  dependency alias, snapshot path, JSON Pointer and Code in UTF-8 byte order.

## 3. Clause / case → test traceability

| Clause / case | Implemented behavior | Executable test(s) |
|---|---|---|
| #62 §1 algorithm identity and domain separator | `AlgorithmIdentifier`, `DomainSeparator`, `sha256:` wire value | `TestSchemaClosureDigestGoldenVector` |
| #62 §2 root identity `(artifactRef, digest, pointer)` | exact root resolution; no fallback | `TestRootPointerMustBeSchemaLocation`, `TestCrossArtifactRelativeReferenceResolution` |
| #62 §2.1 raw-byte artifact digest; stale digest rejected | compute SHA-256 from `Raw`, compare declared | `TestStaleDeclaredArtifactDigestRejected` |
| #62 §2.1 parsed document must equal parse of raw bytes | strict re-parse + normalized deep equality | `TestParsedDocumentMustEqualRawParse` |
| #62 §2.1 1:1 artifactRef / snapshotPath / documentUri binding | uniqueness maps | `TestArtifactBindingConflictsRejected`, `TestDocumentURIMustEqualDerivedURI` |
| #62 §2.1 canonical document URI rules | lowercase scheme/host, default ports, dot segments, percent normalization, repeated separators, user-info/query/fragment | `TestDocumentURICanonicalization`, `TestInvalidSnapshotBaseVariantsRejected` |
| #62 §2.1 snapshot path normalization | relative, no empty/`.`/`..`/leading-slash/backslash segments | `TestInvalidSnapshotPathRejected` |
| #62 §2.1 OpenAPI relative `$ref` against containing document URI | RFC 3986 resolution | golden vector (`common.yaml`), `TestCrossArtifactRelativeReferenceResolution` |
| #62 §2.1 reachable OpenAPI `$id` rejected | closure-time `$id` rejection | `TestOpenAPIReachableIDRejected` |
| #62 §2.1 nested JSON Schema `$id` base + immutable alias | nearest-enclosing `$id` index, alias conflict rejection | golden vector, `TestAmbiguousIDAliasRejected`, `TestCrossArtifactRelativeReferenceResolution` |
| #62 §2.1 unmapped/ambiguous/out-of-closure targets | fail closed | `TestRefOutsideSnapshotRejected`, `TestRelativeRefEscapesSnapshotRejected`, `TestMissingRefTargetRejected` |
| #62 §2.1 fragment decoding / RFC 6901 / canonical tokens | decode once, strict UTF-8, escaping validation | `TestFragmentPercentEncodingAndRFC6901Escaping`, `TestPercentEncodedUnreservedFragmentIsEquivalent`, `TestInvalidPointerEscapingRejected` |
| #62 §2.1 nested-resource fragment translation | resource-root + fragment tokens | golden vector node `#/$defs/Shared/$defs/Leaf` |
| #62 §2.1 OpenAPI Schema traversal; boolean `additionalProperties`; tuple `items` invalid | dialect traversal | `TestOpenAPIAdditionalPropertiesBooleanContributesNoChild`, `TestOpenAPITupleItemsRejected` |
| #62 §2.1 JSON Schema traversal incl. `$defs`/`unevaluated*` | keyword tables in `index.go`/`closure.go` | golden vector, `TestCyclicReferencesVisitedOncePerIdentity` |
| #62 §2.1 Boolean Schemas unsupported | root and nested rejection | `TestBooleanSchemaRootRejected`, `TestBooleanSubschemaRejected` |
| #62 §2.1 `$anchor` / `$dynamicRef` / `$dynamicAnchor` / `$recursiveRef` unsupported | closure-time rejection | `TestAnchorRejected`, `TestDynamicRefRejected`, `TestRecursiveRefRejected` |
| #62 §2.1 exact closure, cycles once per identity, sorted nodes | node set + sort | `TestSchemaClosureDigestGoldenVector`, `TestCyclicReferencesVisitedOncePerIdentity`, `TestManifestNodesSortedByArtifactRefThenPointer` |
| #62 §2.1 parser rejections (duplicate keys, unsupported values) | normalized AST + safe-number conversion | `TestDuplicateRawMappingKeysRejected`, `TestUnsafeIntegerRejected`, `TestNonFiniteNumberRejected` |
| #62 §3 canonical manifest, JCS RFC 8785 | manifest struct + `jcs.Transform` | golden vector + `testdata/schema-closure-digest-v1.manifest.jcs` |
| #62 §4 domain-separated digest | `SHA-256(domain || JCS)` | golden vector `expectedDigest` |
| #62 §5 byte identity changes digest | formatting-only source change | `TestRawBytesChangeWithIdenticalParsedValueChangesDigest` |
| #62 §6 golden vector (OpenAPI root + nested `$id` + fragment translation) | full vector conformance | `TestSchemaClosureDigestGoldenVector`, `TestSchemaClosureDigestInputOrderIndependent` |
| #62 §10(1) dependency/version prerequisites | apiVersion/kind structural check | `TestCatalogInvalidAPIVersionAndKind`, `TestCatalogMissingOwnerIsStructural` |
| #62 §10(2) unique export ID and one exact selector | duplicate checks | `TestCatalogDuplicateExportID`, `TestCatalogDuplicateEventSelector` |
| #62 §10(3) pinned OpenAPI provenance/metadata | artifactRef/digest + operationId/method/path | `TestCatalogProtocolMetadataMismatch`, `TestCatalogProtocolArtifactIdentityUnverified` |
| #62 §10(4)(5) trusted Schema closure and CA-SL-usable normalized Schema | Schema Closure recomputation from `schemaRef` | `TestValidSyntheticCatalogAdmission`, `TestCatalogMissingSchemaClosure`, `TestCatalogSchemaDigestMismatch`, `TestCatalogInvalidSchemaClosure` |
| #62 §13 diagnostic identity/priority/source locations/ordering | package-local `Diagnostic` + `sortDiagnostics` | `TestCatalogDuplicateExportID`, `TestCatalogDuplicateEventSelector`, `TestCatalogProtocolArtifactIdentityUnverified`, `TestCatalogDiagnosticOrdering` |
| IN-005 duplicate export id | `INGRESS_EXPORT_ID_DUPLICATE` | `TestCatalogDuplicateExportID` |
| IN-006 artifact raw-byte digest vs trusted identity | reused `DEPENDENCY_IDENTITY_UNVERIFIED`, no `INGRESS_PROTOCOL_ARTIFACT_INVALID` alias | `TestCatalogProtocolArtifactIdentityUnverified` |
| IN-007 / IN-057 pinned OpenAPI metadata contradiction | `INGRESS_PROTOCOL_ARTIFACT_INVALID` | `TestCatalogProtocolMetadataMismatch` |
| IN-013 missing/invalid Schema | reused `SCHEMA_MISSING` / `SCHEMA_INVALID` | `TestCatalogMissingSchemaClosure`, `TestCatalogSchemaDigestMismatch`, `TestCatalogInvalidSchemaClosure` |
| IN-055 API version / kind / root invalid | `INGRESS_CATALOG_INVALID` | `TestCatalogInvalidAPIVersionAndKind`, `TestCatalogMissingOwnerIsStructural` |
| IN-056 duplicate exact event selector | `INGRESS_EVENT_SELECTOR_DUPLICATE` | `TestCatalogDuplicateEventSelector` |
| IN-014 / IN-039 normalization mapping completeness | **deferred** (see §7) | NOT_IMPLEMENTED |

## 4. Diagnostics implemented

All Ingress-owned records carry registry
`connectionagent.io/ingress-diagnostics/v1alpha1`, `phase=C_STATIC`,
`severity=ERROR`. Reused globally-unique CA-SL / dependency identities are not
Ingress-owned and therefore carry no Ingress `registryId`/`messageKey`.

| Code | Owner | Precedence | messageKey | Implemented trigger |
|---|---|---|---|---|
| `YAML_PARSE_ERROR` | reused (CA-SL) | 0 | — | Catalog bytes cannot be safely parsed, including duplicate YAML keys and unsafe/non-finite scalars |
| `DEPENDENCY_IDENTITY_UNVERIFIED` | reused (dependency) | 5 | — | Protocol OpenAPI artifact absent from the trusted snapshot, or its raw-byte digest disagrees with the declared digest |
| `INGRESS_CATALOG_INVALID` | Ingress | 10 | `ingress.catalog.invalid` | Structural schema failure, unsupported/missing apiVersion or kind, missing owner |
| `INGRESS_EXPORT_ID_DUPLICATE` | Ingress | 20 | `ingress.export.id.duplicate` | `operations[].id` repeated; primary points to the later `id`, related to the first |
| `INGRESS_EVENT_SELECTOR_DUPLICATE` | Ingress | 21 | `ingress.event.selector.duplicate` | exact `event.source + event.service + event.operation` tuple repeated |
| `INGRESS_PROTOCOL_ARTIFACT_INVALID` | Ingress | 30 | `ingress.protocol.artifact.invalid` | Pinned OpenAPI operation metadata conflicts with `operationId`/`method`/`path` declarations |
| `SCHEMA_MISSING` | reused (CA-SL) | 60 | — | `schemaRef` is not declared by the trusted Schema export bindings |
| `SCHEMA_INVALID` | reused (CA-SL) | 61 | — | Schema Closure computation fails, or declared `schemaDigest` does not equal the recomputed digest |

`INGRESS_CATALOG_INVALID` also uses the registry's canonical message template
with `{catalogPath}`; duplicate and protocol templates use their exact
`{exportId}`, `{source}/{service}/{operation}` and `{fieldPath}` placeholders.

## 5. Files added

- `internal/ingresscatalog/errors.go`
- `internal/ingresscatalog/uri.go`
- `internal/ingresscatalog/jsonpointer.go`
- `internal/ingresscatalog/jsonvalue.go`
- `internal/ingresscatalog/artifact.go`
- `internal/ingresscatalog/index.go`
- `internal/ingresscatalog/ref.go`
- `internal/ingresscatalog/closure.go`
- `internal/ingresscatalog/diagnostics.go`
- `internal/ingresscatalog/catalog_schema.go`
- `internal/ingresscatalog/catalog.go`
- `internal/ingresscatalog/schemas/ingresscatalog.schema.json`
- `internal/ingresscatalog/testdata/schema-closure-digest-v1.vector.json`
- `internal/ingresscatalog/testdata/schema-closure-digest-v1.manifest.jcs`
- `internal/ingresscatalog/testdata/valid-catalog.yaml.tmpl`
- `internal/ingresscatalog/closure_golden_test.go`
- `internal/ingresscatalog/closure_negative_test.go`
- `internal/ingresscatalog/closure_edge_test.go`
- `internal/ingresscatalog/catalog_test.go`
- `internal/ingresscatalog/testhelpers_test.go`
- `contracts/v1alpha1/ingresscatalog.schema.json`
- `contracts/v1alpha1/README.md`
- `docs/workstreams/wp1-ingress-catalog-v1alpha5.md` (this file)
- `go.mod` (promote `github.com/gowebpki/jcs v1.0.2` to a direct require)

## 6. Local verification (actually executed)

Environment: `go version go1.26.6 windows/amd64`, Windows PowerShell,
`GOWORK=off`; CGO-free gates with `CGO_ENABLED=0`.

| # | Exact command | Result |
|---|---|---|
| 1 | `$env:GOWORK='off'; go test ./... -count=1` | **PASS** (all packages `ok`, including `internal/ingresscatalog` 1.323s) |
| 2 | `$env:GOWORK='off'; go vet ./...` | **PASS** (exit 0) |
| 3 | `$env:GOWORK='off'; $env:CGO_ENABLED='0'; go build ./...` | **PASS** (exit 0) |
| 4 | `git diff --check` | **PASS** (exit 0) |
| 5 | `$env:GOWORK='off'; $env:CGO_ENABLED='0'; $env:GOOS='js'; $env:GOARCH='wasm'; go build -o <temp> ./cmd/skillc-wasm` | **PASS** (exit 0; additional gate, not required by the WP1 task) |

CI: not run / unavailable for this branch; the results above are local only.
No Chromium parity gate was executed by this work package.

## 7. Not yet implemented / deferred to WP3+

This slice deliberately implements the Schema Closure Digest and the
catalog admission/identity/enumeration layer only. Deferred:

- WP3+ Trigger Operation linking and typed `armInputSchema` derivation
  (#62 §10(6)–(10)), Skill Wait selector resolution
  (`WAIT_INGRESS_EXPORT_NOT_FOUND` / `WAIT_INGRESS_EXPORT_AMBIGUOUS`).
- Request normalization semantics: media-type/representation one-to-one
  correspondence, mapping coverage/target/transform validation,
  `INGRESS_NORMALIZATION_INVALID` (IN-014, IN-039, IN-015, IN-059).
- Multipart/binary content lifecycle (`INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID`)
  and response contract/mapping validation (`INGRESS_RESPONSE_CONTRACT_INVALID`,
  `INGRESS_RESPONSE_MAPPING_INVALID`).
- Peer profile reference resolution (`INGRESS_PEER_PROFILE_REFERENCE_INVALID`)
  and correlation/stale-delivery safety (`INGRESS_CORRELATION_INVALID`).
- `release` / `apiUriVersion` verification against a pinned OpenAPI artifact:
  #62 defines no authoritative OpenAPI field mapping for those two values in
  this slice, so only `operationId`, HTTP method and path are linked. Recorded
  as a contract question rather than invented semantics.
- `responseResultSchemaRef`, `correlation.armInputSchemaRef`,
  `binaryContent.referenceSchemaRef` are projected but no closure digest is
  declared for them in v1alpha1, so they are not verified.
- Build/Check/InspectDependencies wiring, CLI/WASM/Studio integration and
  Diagnostic envelope serialization.
- Real Owner artifacts, real dependency snapshots and real 5GC/3GPP
  qualification: none of the fixtures here are Owner artifacts.

## 8. Synthetic fixture warning

Every fixture in this workstream is synthetic:

- `internal/ingresscatalog/testdata/valid-catalog.yaml.tmpl` is explicitly
  labeled `SYNTHETIC TEST FIXTURE ONLY`; its Owner, adapter, profile refs,
  Schema refs and artifact refs are placeholders, and its digests are computed
  in-test from synthetic raw artifacts.
- The golden vector and its raw artifacts are the candidate #62 conformance
  vector, not a published Owner snapshot.
- No fixture may be reported as real Owner, deployment or 5GC interoperability
  evidence.
