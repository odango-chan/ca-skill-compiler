# Next CA Skill Compiler Development Plan

**Status:** DRAFT — isolated v1alpha5 candidate implementation may start; public target enablement and release remain gated.  
**Baseline:** CA Skill Compiler v1.0.0; released v1alpha4 remains immutable.  
**Next language target:** `connectionagent.io/v1alpha5` is assigned in the CA-SL review candidate; it is NOT RELEASED / NOT QUALIFIED.

## 1. Objective

Extend the existing Go Core incrementally for the next CA-SL revision, covering Node-entry Event Wait and package-scoped Integration Ingress Operation linking. This is not a rewrite. The published v1.0.0 release and all v1alpha4 semantics remain immutable.

The target capabilities are:
- Node-entry Wait syntax and AST;
- trusted package-scoped IngressCatalog admission/linking;
- exact triggering ToolCatalog Operation → typed `armInputSchema` binding;
- `wait.output` Schema resolution and path-sensitive availability;
- Wait entry-control graph edges, reachability, dominance and DAG checks;
- `wait.response.after` Producer-stage selection and response Schema mapping;
- Native / Go-WASM / Chromium Worker parity.

## 2. Contract status

| Contract | Repository / PR | Status | Compiler impact |
|---|---|---|---|
| Node-entry Wait semantic baseline | ca-skills #60 | Merged | Semantic basis |
| EVENT_ACCEPTED tombstone / stale-message isolation | ca-skills #64 | Merged | Runtime invariant; Compiler links the declared correlation policy |
| v1alpha5 single-file language candidate | ca-skills #61, head `9fadd3f4` | Draft; target assigned; semantic baseline marked frozen; full text and conformance material under independent review | 5,474-line single-file candidate exists; Appendix L defines Wait grammar/semantics; Appendix H adds 9 C_STATIC WAIT diagnostics; EW-001–EW-036 remain specification-only |
| Integration Ingress Operation | ca-skills #62, head `1c55ab8` | Draft; current schema/vector workflows PASS; contract not frozen | IngressCatalog, exact linker semantics and diagnostic ownership still require finalization; raw-byte and nested-resource resolver vector gaps have been addressed in current head |
| Generic Host / Runtime continuation | ca-agent-demo #210 | Draft, design-only | Compiler consumes language/export contracts; it does not implement Host/Runtime |
| 5GC SBI continuation profile | ca-agent-demo #213 (supersedes closed #209) | Draft, design-only | Example profile, not 5GC-specific Compiler semantics |

v1alpha4 must continue rejecting author-level `wait`, `wait.output` and `ingressOperation`. The existing frozen diagnostics remain unchanged. The v1alpha5 target identity is no longer unassigned: the current #61 candidate names `connectionagent.io/v1alpha5` and provides a complete single-file review artifact. Its header marks the semantic baseline frozen, but the full normative text and conformance material remain under independent review; it is NOT RELEASED / NOT QUALIFIED. The 36 EW obligations are explicitly `SPECIFICATION_ONLY`, not executed tests. v1alpha4 must continue rejecting author-level `wait`, `wait.output` and `ingressOperation`; its decoder, diagnostics and released bytes remain unchanged.

The Ingress Operation is a separate package-scoped export, not a ToolCatalog callable, Observation or Capability. The current #62 head now verifies exact raw source bytes against each `artifactDigest`, proves nested-`$id` fragment translation to a non-root artifact pointer, and passes its two current workflows. The PR remains Draft because the export contract and the cross-contract `INGRESS_*` / existing-diagnostic ownership map are not frozen.

**Implementation decision:** start isolated WP2 Wait grammar/AST work against the exact v1alpha5 candidate and continue WP1 Schema Closure work behind internal interfaces. Do not expose a public v1alpha5 Build target, alter v1alpha4, or claim Compiler conformance until #61/#62 and the versioned API contracts are frozen.

The first isolated WP2 implementation is [ca-skill-compiler PR #30](https://github.com/odango-chan/ca-skill-compiler/pull/30): a private-package Wait AST/parser slice with fixed-duration, selector, response-boundary and TIMEOUT-route checks. It intentionally does not add BuildRequest dispatch or link IngressCatalog. Its GitHub Actions job currently has no assigned runner (`runner_id=0`, no executed steps); this is an infrastructure/CI availability gap, not a test failure result. Only the fixed-duration grammar has received a separate local Go check so far; full Go tests/vet/Native/WASM acceptance remains NOT_RUN.

Real Owner/Deployment publications and live 5GC qualification do not block independent static Compiler implementation. Synthetic trusted fixtures are sufficient for static development; they must never be reported as real interoperability evidence.

## 3. Versioned API impact

The current v1alpha4 machine contracts use closed fields/enums and cannot be silently extended.

| Existing contract | Closed point | Required next decision |
|---|---|---|
| BuildRequest v1alpha4 | `TrustedDependencyBinding.exportOwners[].exportKind` allows only operation / observation / schema / evidenceGuarantee; languageTarget is fixed to v1alpha4 | New BuildRequest contract must admit the distinct `ingressOperation` export kind and bind one exact new language target |
| DependencyInspection v1alpha4 | `callables[]` contains only operation / observation plus schemas[]; no package-level Ingress inventory; diagnostics use the closed v1alpha4 envelope | New contract must add typed `ingressOperations[]` and the v1alpha5 diagnostic envelope; never disguise Ingress as a callable |
| Artifact Manifest v1alpha4 | languageTarget and dependencyClosure exportKind are closed | New manifest contract must record the new language target and `ingressOperation` dependency export |
| BuildResult v1alpha4 | Output envelope and each Diagnostic item use `additionalProperties: false` | **A new BuildResult API version is required** because the Ingress diagnostic contract requires `registryId`, `messageKey`, typed `sourceLocation` and optional `relatedLocations`; these cannot be represented by the v1alpha4 Diagnostic schema |
| Diagnostics / conformance | v1alpha5 Appendix H composes 9 WAIT diagnostics and the separate 9-code IngressCatalog v1alpha1 registry; identity remains `(Code, Phase)` | Freeze the exact combined index and preserve the Ingress `registryId` as provenance, not as a second identity key |

BuildRequest, DependencyInspection, Artifact Manifest, CA-SL language and Compiler product versions are independent identities. Do not assume they share a version number.

### 3.2 Diagnostic envelope forces BuildResult and DependencyInspection versioning

The current `Diagnostic` Go type carries only `severity`, `phase`, `code`, `path`, optional `line/column` and rendered `message`. The v1alpha4 BuildResult JSON Schema sets `additionalProperties: false` on each Diagnostic item.

The IngressCatalog v1alpha1 diagnostic registry now requires the logical Compiler Diagnostic record to preserve:
- `code`, `phase=C_STATIC`, `severity=ERROR`;
- `registryId=connectionagent.io/ingress-diagnostics/v1alpha1`;
- stable `messageKey`;
- `sourceLocation` with `dependencyAlias`, `snapshotPath` and RFC 6901 `jsonPointer`;
- optional `relatedLocations` for duplicate and cross-artifact explanations.

Therefore this is no longer a conditional BuildResult decision: **BuildResult and DependencyInspection both require a new versioned diagnostic envelope**. The new contract must support both Skill-source locations and dependency-snapshot locations without flattening the latter into an ambiguous string `path`. For Ingress records, `registryId` is provenance; diagnostic identity remains the globally unique `(Code, Phase)`. The exact tagged source-location shape and conditional requirements (e.g. Ingress records require `registryId/messageKey/sourceLocation`) are WP0 API design outputs.

The v1alpha4 Go types, JSON Schemas, diagnostic wire shape and released bytes remain unchanged. The new API version identifiers are independent of `connectionagent.io/v1alpha5`; do not assign matching suffixes merely for naming symmetry.

### 3.1 API dispatch and v1alpha4 isolation

The current public entry path is version-specific, not a generic target router:
- `DecodeBuildRequestJSON` strictly decodes the v1alpha4 Go type;
- `validateBuildRequest` requires the v1alpha4 BuildRequest API and `connectionagent.io/v1alpha4`;
- `Check`, `Build` and `InspectDependencies` dispatch directly to v1alpha4 behavior;
- the v1alpha4 DependencyInspection and Artifact Manifest have closed field sets and export-kind enums.

The next implementation MUST introduce an explicit **API-version dispatch before target dispatch**:

1. Read only the top-level `apiVersion` discriminator using a bounded envelope decoder.
2. Dispatch to exactly one strict, version-specific BuildRequest decoder. Unknown or unsupported API versions fail closed; no fallback decoder is attempted.
3. Validate that API version's own schema and limits.
4. Dispatch the exact `languageTarget` to its version-specific parser/linker. An unsupported language target fails closed; it MUST NOT be interpreted as v1alpha4.
5. Route Check / Build / InspectDependencies and result verification through the same selected API/target profile.

Do not decode a future request into the v1alpha4 struct and then tolerate extra fields. Do not widen the v1alpha4 JSON Schemas, Go structs, diagnostic index, artifact manifest, CLI behavior or released v1.0.0 bytes.

The API-version decisions must cover these separate surfaces:

| Surface | Required next-version decision |
|---|---|
| BuildRequest | New strict API schema/Go type for the exact language target and the distinct `ingressOperation` trusted export kind |
| DependencyInspection | New strict output contract with a typed `ingressOperations[]` projection; keep Ingress separate from `callables[]` |
| Artifact Manifest | New manifest contract that records the exact language target and `ingressOperation` in dependency closure; do not embed the entire external IngressCatalog |
| BuildResult | A new version is required: the Ingress diagnostic registry adds `registryId`, `messageKey`, dependency-aware `sourceLocation` and optional `relatedLocations`; v1alpha4's closed Diagnostic item cannot represent them |
| CLI project input | Extend the local `trusted.json` adapter deliberately to represent the new export kind; this local input description is not the BuildRequest wire contract |
| WASM / Studio adapters | Consume the same versioned BuildRequest / DependencyInspection contracts and the same Core dispatch; no second parser or linker |

The IngressCatalog's own `apiVersion`, the CA-SL `languageTarget`, BuildRequest, DependencyInspection, Artifact Manifest and Compiler product version are separate identities. A change to one MUST NOT silently imply a change to another.

### 3.2 v1.0.0 implementation limits inherited by the next target

The v1.0.0 release is immutable and locally accepted for its declared scope, but its final acceptance record explicitly marks some positive paths PARTIAL / fail-closed:
- WP6 identity-input provenance requires a future trusted publication;
- WP7 Evidence Descriptor content/Producer/Scope/Correlation/Freshness linking remains fail-closed where the physical Owner descriptor format is unavailable;
- WP8 cross-snapshot Reconcile association and complete retry admission remain fail-closed / outside Compiler;
- WP9 Evidence projection in the Script ABI remains blocked.

The current artifact pipeline deliberately refuses to emit a successful artifact when a selected Skill uses Predicate Evidence, a MUTATION Operation, or `retryOf` while those complete positive-path obligations are unavailable. Static diagnostics and sub-checks still run; this is not the same as those features being fully compiled.

This limitation does **not** prevent independent Node-entry Wait / Ingress Compiler work. It does affect the acceptance claim for a real PDU/MAS Skill that uses those features. Therefore:
- carry the v1alpha4 regression suite forward unchanged;
- keep the existing WP6/WP7/WP8 blocked cases explicit;
- implement every Compiler-owned static rule that has a frozen contract using synthetic trusted fixtures;
- do not fabricate real Owner/Deployment publications to make a positive Build pass;
- do not mark a full PDU/MAS artifact build PASS until its required static positive paths are implemented and its external trust inputs are supplied.

The next target is an extension of the existing Go Core, not permission to silently treat the partial v1alpha4 gates as complete.

## 4. Target architecture

Retain the current pipeline:

`BuildRequest → Source Admission → Parse/AST → Dependency Linking → Schema/CEL → Procedure Graph → Script ABI → Artifact`

Add a target-specific branch rather than inserting new grammar into v1alpha4:

1. **Language target selection:** choose exactly one target before parsing; unknown targets fail closed; no fallback to v1alpha4.
2. **IngressCatalog index:** parse the package-scoped export from the trusted dependency snapshot and verify exact apiVersion, exportKind, Package ID, Owner and artifact/Schema identity.
3. **Wait AST:** recognize Node-entry `wait` only for the new language target; retain v1alpha4 rejection.
4. **Ingress linker:** resolve exact source/service/operation; no fuzzy name matching.
5. **Trigger binding linker:** bind each incoming Wait path to an exact ToolCatalog Operation and resolve the typed `armInputSchema` from Operation arguments or admitted Invocation Context.
6. **Wait output resolver:** bind `NodeId.wait.output` to the normalized request Schema; validate JSON Pointer, assignability and path availability.
7. **Procedure graph extension:** model Wait as a Node-entry control vertex with EVENT_RECEIVED and TIMEOUT edges; include it in DAG, reachability, dominance, route obligations and SourcePath availability.
8. **Response boundary linker:** derive the exact Producer SourcePath from `wait.response.after` and the Ingress-declared producer stage type (operation / observation); validate output Schema, responseResultSchema and JSON/multipart mappings.
9. **Artifact assembly:** record the exact new language target and dependency closure in a new manifest; do not copy external IngressCatalog data into Skill source files.

### 4.1 Static Compiler scope versus cross-component conformance

The EW and IN matrices are joint conformance inventories. Compiler must execute only the statically decidable obligations and must not simulate Host, Runtime or Integration behavior.

**Compiler-owned Wait cases from #61:**

- **EW-012–EW-016:** Wait output path availability; nonempty business-stage requirement; forbidden timeout-to-SUCCEEDED route; exact Ingress selector resolution; pre-dispatch correlation binding.
- **EW-018–EW-019:** all incoming paths resolve to the same trusted Ingress contract; response boundary cannot be bypassed.
- **EW-021–EW-022:** typed body-bearing response mapping and the permitted fixed bodyless response case.
- **EW-027, EW-029–EW-030:** exact duration grammar; forbidden response-result sources; response boundary Node identity/reachability.
- **EW-031:** static preservation of global Precondition dominance; Wait acceptance does not create Goal proof.
- **EW-033–EW-036:** exactly one TIMEOUT route; no author-declared EVENT_RECEIVED route; selector field shape; timeout scalar/type.

**Compiler-owned Ingress cases from #62:**

- **IN-001–IN-007:** package-scoped dependency requirement, exact export resolution/ambiguity, trusted Owner/export identity, duplicate IDs, pinned OpenAPI artifact digest and Service-Operation versus OpenAPI operationId distinction.
- **IN-013–IN-015:** normalized Schema support, normalization digest and required wire-field coverage.
- **IN-018–IN-021:** exact ToolCatalog Operation trigger identity, pre-dispatch argument binding, start-Context binding and all incoming-path coverage.
- **IN-023:** statically decidable stale-delivery proof failures; deployment-dependent admission facts remain separately identified.
- **IN-028–IN-032:** response disposition cardinality, body-bearing Producer requirement, exact response boundary stage, Schema assignability and required response-field coverage.
- **IN-039–IN-040:** accepted media type/source Schema consistency and duplicate disposition identity/status.
- **IN-042, IN-044–IN-047:** statically decidable binary-content retention constraints, pinned multipart response completeness, 3GPP rejection-response mapping completeness and discriminator-retention arithmetic.

The following remain Integration / Host / Runtime / deployment-conformance obligations, not Compiler tests: EW-001–EW-011, EW-017, EW-020, EW-023–EW-026, EW-028, EW-032; and IN-008–IN-012, IN-016–IN-017, IN-022, IN-024–IN-027, IN-033–IN-038, IN-041, IN-043, IN-048–IN-051, IN-053–IN-054. Shared cases such as IN-023, IN-039, IN-042 and IN-044–IN-047 may have both a Compiler static fixture and a separate Admission/Integration fixture; one must not be reported as the other.

### 4.3 WP1 Schema Closure Digest v1 implementation boundary

The current #62 proposal is more specific than a generic recursive `$ref` walk. The Compiler implementation must cover all of the following as one deterministic linker input:

1. **Trusted artifact binding:** bind each opaque `artifactRef` one-to-one to its normalized snapshot-relative path, canonical document URI, source dialect, parsed document and complete raw source bytes. Recompute `artifactDigest` from the exact immutable bytes; never trust the digest declared by IngressCatalog or a vector.
2. **Canonical URI rules:** validate the absolute hierarchical snapshot base URI; derive canonical document URIs from percent-encoded normalized paths; normalize scheme/host/default ports, dot segments and percent escapes; reject user-info, query, fragment, whitespace, backslash, malformed escapes and ambiguous document mappings. URI lookup is in-memory only; no network fetch.
3. **Dialect-aware Schema discovery:** OpenAPI 3.0 roots come only from Schema Object positions in the OpenAPI document and use OpenAPI 3.0 Schema rules; JSON Schema Draft 2020-12 uses its declared dialect and Schema-valued keywords. Do not reinterpret one dialect as the other.
4. **Nested resource identity:** OpenAPI 3.0 Schema Objects MUST reject `$id` as a base-URI mechanism. JSON Schema 2020-12 MUST resolve nested `$id` against the inherited base and register a unique immutable resource-URI alias to its exact artifact-relative Schema pointer.
5. **Reference resolution:** resolve each `$ref` against the containing Schema's effective base URI; allow only canonical document URIs or registered nested-`$id` aliases in the same trusted snapshot. Percent-decode fragments exactly once as strict UTF-8; validate RFC 6901; translate a nested-resource-relative fragment back to the exact artifact-relative pointer.
6. **Exact closure:** traverse only Schema-valued substructures supported by the declared dialect; include the root plus every and only transitive referenced Schema node; identify nodes by exact `(artifactRef, pointer)`; visit cycles once; reject missing/extra nodes, boolean Schemas, non-Schema pointers, unsupported anchors/dynamic references, invalid IDs and snapshot escapes.
7. **Digest:** construct the specified manifest with each node's complete source-artifact digest and exact parsed Schema value; sort nodes by UTF-8 `artifactRef` then pointer; serialize with RFC 8785 JCS; hash the exact v1 domain separator plus JCS bytes.

The Compiler conformance vector must prove both **source-byte binding** and **resolution**: a byte-only artifact change with an unchanged parsed Schema value still changes `artifactDigest` and the closure digest; a nested-`$id` fragment must resolve to a non-root artifact-relative pointer. A vector that supplies only pre-resolved edges or hard-coded digests is not sufficient to qualify the resolver.

Keep this implementation isolated from the frozen v1alpha4 Schema Registry. v1alpha4's exact Schema identity and local-reference behavior must not be silently rewritten to adopt the new Ingress-specific URI-reference algorithm.

### 4.2 Diagnostic ownership and precedence

Before code is written, the language and export owners must publish one collision-free diagnostic map:

- `WAIT_*` identifies a Skill-authored Wait declaration, Wait graph, path availability, trigger-to-Wait binding or Skill response-boundary failure.
- `INGRESS_*` identifies an invalid/untrusted IngressCatalog export or its own protocol, normalization, correlation, trigger declaration or response mapping.
- Existing frozen `SOURCE_PATH_*`, `CONTROL_FLOW_*`, `INPUT_TYPE_MISMATCH` and `DEPENDENCY_*` diagnostics are reused wherever #61 explicitly assigns them.
- A missing/ambiguous selected Ingress export is a Wait selector failure (`WAIT_INGRESS_EXPORT_NOT_FOUND` / `WAIT_INGRESS_EXPORT_AMBIGUOUS`); a malformed catalog or duplicate export is an Ingress export failure.
- A trigger Operation missing from the trusted snapshot is an Ingress trigger-export failure; a Skill path that cannot materialize the declared arm input on every incoming path is a Wait correlation-binding failure.
- An invalid Ingress-owned response mapping is distinct from a Skill's missing/unavailable `wait.response.after` Producer.
- Every new Compiler diagnostic must have one exact `(Code, Phase)`, severity, precedence and source-location rule. #62's candidate `INGRESS_*` list currently does not freeze all of those properties; Compiler MUST NOT invent or silently alias identities.

## 5. Work packages

| WP | Work | Main output | Entry gate |
|---|---|---|---|
| WP0 | Language/API freeze inventory | Complete single-file spec, version identity, BuildRequest / DependencyInspection / Artifact contract decisions, diagnostic catalogue | Independent review of #61 / #62 |
| WP1 | IngressCatalog admission/linking | Export parser, trusted Owner binding, raw-artifact digest verification, dialect-aware Schema Closure Digest v1 and exact transitive reference closure | WP0 |
| WP2 | Wait syntax / AST | Node-entry grammar, duration parser, strict unknown-field diagnostics | WP0 |
| WP3 | Trigger binding | Exact Operation identity, armInputSchema binding, start-context binding | WP1 + WP2 |
| WP4 | wait.output dataflow | Normalized Schema resolution, JSON Pointer, assignability and edge/path-sensitive availability | WP1 + WP2 |
| WP5 | Wait graph | Entry-control vertex, EVENT_RECEIVED/TIMEOUT, DAG/dominance/reachability/route obligations | WP2–WP4 |
| WP6 | Response boundary | response.after, Producer-stage derivation, responseResultSchema and JSON/multipart/binary mapping | WP1–WP5 |
| WP7 | Diagnostics / conformance | Exact Code/Phase, positive/negative static fixtures, case-to-test map | WP0–WP6 |
| WP8 | Native / WASM / Worker parity | Go tests, vet, CGO-free Native, WASM, real Chromium parity and artifact byte parity | WP1–WP7 |
| WP9 | Studio / LSP integration | DependencyInspection UI, Wait/Ingress diagnostics, completion and Schema navigation | API contracts + WP8 |

The EW-001–EW-036 and IN-001–IN-054 matrices are cross-component conformance obligations. They are not all Compiler-owned tests and are not claimed as passed.

## 6. Compiler invariants

- Ingress Operation is an inbound protocol export, not an outbound Tool callable.
- Outbound triggers resolve to exact ToolCatalog Operations; a Capability ID, Observation, network address or free-form label is not a substitute.
- `wait.output` is a distinct Producer SourcePath; it is not global Context, Operation Result, Observation Result or Evidence.
- Wait is not a fifth business stage or a separate Wait Node kind. The business stages remain precondition / operation / observation / postcondition.
- EVENT_RECEIVED and TIMEOUT are control-flow edges, not Stage Outcomes.
- `responseObligationRef`, `expectedIngressRef`, `deliveryRef`, Host `ingressRef/lifecycleRef`, Run IDs and protocol-private correlation must not become Skill inputs or inferred business fields.
- `responseDeadline` is the Host/Runtime logical response-obligation cap; per-HTTP-attempt deadlines belong to Integration and are not Wait timeouts.
- Compiler does not implement event reception, timers, persistence, atomic arbitration, retry execution, tombstone lifecycle, Temporal workflows or HTTP/2/multipart transport.

## 7. 3GPP reference boundary

The current 5GC example profile uses independently pinned Release 19 versions checked on 2026-11-17: TS 23.502 V19.9.0, TS 29.502 V19.7.0, TS 29.571 V19.7.0 (Common Data Types companion to the Nsmf OpenAPI), and TS 29.500 V19.7.1. These are external protocol references, not a global CA-SL version.

The automated source-lock fixture in ca-skills #62 remains on TS 29.502/29.571 V19.5.0 because its ETSI companion ZIP/YAML bytes have reproducible checksums. It is explicitly a historical structural fixture, not the current production 5GC API pin. A real Integration admission must re-lock and verify the V19.7.0 official companion artifacts.

## 8. Non-goals

- No Host, Runtime, Integration Adapter, Temporal or SBI client implementation.
- No 5GC, SMF, AMF, UPF, OpenAMSM or MAS-specific logic in Compiler.
- No package manager, Registry, database, release signing, Qualification service or Runtime dispatcher.
- No v0/v0.1/v0.2 compatibility or automatic migration.
- No change to CA-SL v1alpha4 semantics, diagnostics or released artifacts.
- Missing real Owner/Deployment artifacts do not block static Compiler implementation, but real product qualification remains NOT_QUALIFIED.

## 9. Implementation gate

**Can start now (isolated candidate work):**
- WP2: implement and test the v1alpha5 Wait grammar/AST against the exact #61 single-file candidate; keep it outside the v1alpha4 dispatch.
- WP1: implement the Schema Closure resolver behind an internal interface using immutable synthetic snapshots; current #62 vector now covers raw-byte binding and nested-resource pointer translation.
- Build the v1alpha4 compatibility harness and EW/IN static fixture inventory.
- Design API-version dispatch before language-target dispatch.

**Still gated before public v1alpha5 target integration / release:**

1. #61 completes independent full-text review and freezes/publishes the v1alpha5 normative single-file specification and its exact diagnostic index.
2. #62 freezes the IngressCatalog schema, Schema Closure Digest, correlation/stale-delivery, trigger binding and response mapping contract.
3. #62 and #61 publish one collision-free diagnostic ownership/precedence contract: reuse existing frozen dependency/export/schema/source-path diagnostics where applicable; reserve WAIT diagnostics for Skill Wait selection/binding/boundary errors; define any remaining INGRESS diagnostics with exact `(Code, Phase)`, severity, precedence and source-location rules.
4. New BuildRequest, DependencyInspection and Artifact Manifest API versions are decided independently; BuildResult is explicitly retained or versioned based on its actual shape.
5. Target dispatch preserves v1alpha4 behavior and the published v1.0.0 Artifact unchanged.
6. The inherited v1alpha4 WP6/WP7/WP8 limitations have explicit carry-forward dispositions so the next target does not claim positive Build support for features that still fail closed.

Real 5GC Owner publication or E2E qualification is not a prerequisite for isolated static Compiler development; it remains a separate product-qualification gate.

## 10. Acceptance

- v1alpha4 regression: no change to existing positive/negative fixtures, diagnostic identities, Artifact bytes or Native/WASM parity.
- New target: fail closed on wrong apiVersion, missing export, wrong Owner, Schema digest mismatch, invalid trigger binding, unavailable wait.output, invalid Wait graph, wrong response boundary or incomplete multipart mapping.
- Native, WASM and Chromium Worker produce identical Diagnostic[], DependencyInspection and Artifact bytes for the same BuildRequest.
- Local acceptance and GitHub Actions status are reported separately. If CI is unavailable, report LOCAL_ACCEPTANCE_PASS, never CI PASS.
- Real Owner/Deployment/5GC qualification remains NOT_QUALIFIED until backed by real versioned publications and live evidence.

