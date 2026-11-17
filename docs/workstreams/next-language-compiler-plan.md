# Next CA Skill Compiler Development Plan

**Status:** DRAFT — architecture and work-package preparation only; implementation against a new language target is not yet authorized.  
**Baseline:** CA Skill Compiler v1.0.0; the only released language target is frozen CA-SL v1alpha4.  
**Next language version:** Unassigned. This document does not infer or allocate a version identifier.

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
| Normative Wait amendment | ca-skills #61 | Draft; language version unassigned | Grammar, diagnostics, SourcePath and graph rules are not frozen |
| Integration Ingress Operation | ca-skills #62 | Draft | IngressCatalog, Schema Closure Digest, normalization and response mapping are not frozen |
| Generic Host / Runtime continuation | ca-agent-demo #210 | Draft, design-only | Compiler consumes language/export contracts; it does not implement Host/Runtime |
| 5GC SBI continuation profile | ca-agent-demo #209 | Draft, design-only | Example profile, not 5GC-specific Compiler semantics |

v1alpha4 must continue rejecting author-level `wait`, `wait.output` and `ingressOperation`. The existing frozen diagnostics remain unchanged. PR #61 explicitly prohibits treating its draft as v1alpha4 or inferring the next language version. The Ingress Operation is a separate package-scoped export, not a ToolCatalog callable, Observation or Capability.

Real Owner/Deployment publications and live 5GC qualification do not block independent Compiler implementation. Once the format is frozen, synthetic trusted fixtures are sufficient for Compiler tests; they must never be reported as real interoperability evidence.

## 3. Versioned API impact

The current v1alpha4 machine contracts use closed fields/enums and cannot be silently extended.

| Existing contract | Closed point | Required next decision |
|---|---|---|
| BuildRequest v1alpha4 | `TrustedDependencyBinding.exportOwners[].exportKind` allows only operation / observation / schema / evidenceGuarantee; languageTarget is fixed to v1alpha4 | New BuildRequest contract must admit the distinct `ingressOperation` export kind and bind one exact new language target |
| DependencyInspection v1alpha4 | `callables[]` contains only operation / observation plus schemas[]; no package-level Ingress inventory | Add typed `ingressOperations[]` or a generic typed export inventory; never disguise Ingress as a callable |
| Artifact Manifest v1alpha4 | languageTarget and dependencyClosure exportKind are closed | New manifest contract must record the new language target and `ingressOperation` dependency export |
| BuildResult v1alpha4 | Output envelope is separate from Artifact manifest | Compare shapes first; version it only if the envelope changes |
| Diagnostics / conformance | Code, Phase and E01–E120 / Appendix H identities are frozen | New diagnostics must be published by the language owner; Compiler must not invent semantic identities |

BuildRequest, DependencyInspection, Artifact Manifest, CA-SL language and Compiler product versions are independent identities. Do not assume they share a version number.

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

## 5. Work packages

| WP | Work | Main output | Entry gate |
|---|---|---|---|
| WP0 | Language/API freeze inventory | Complete single-file spec, version identity, BuildRequest / DependencyInspection / Artifact contract decisions, diagnostic catalogue | Independent review of #61 / #62 |
| WP1 | IngressCatalog admission/linking | Export parser, trusted Owner binding, Schema Closure Digest v1 and exact reference closure | WP0 |
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

**Can start now:** contract review, API version inventory, Schema Closure Digest review, diagnostic/test matrix design, Compiler extension-point design and synthetic fixture planning.

**Formal implementation requires:**

1. The CA-SL language owner assigns the next language version and publishes the complete single-file specification.
2. The #61 grammar, Wait output, graph, response-boundary and diagnostic clauses are frozen.
3. The #62 IngressCatalog schema, Schema Closure Digest, correlation/stale-delivery and response-mapping contract is frozen.
4. New BuildRequest, DependencyInspection and Artifact Manifest API versions are decided.
5. Target dispatch preserves v1alpha4 behavior and the published v1.0.0 Artifact unchanged.

After these gates, WP1–WP8 may proceed using synthetic trusted snapshots. Real 5GC Owner publication or E2E qualification is not a prerequisite for Compiler implementation.

## 10. Acceptance

- v1alpha4 regression: no change to existing positive/negative fixtures, diagnostic identities, Artifact bytes or Native/WASM parity.
- New target: fail closed on wrong apiVersion, missing export, wrong Owner, Schema digest mismatch, invalid trigger binding, unavailable wait.output, invalid Wait graph, wrong response boundary or incomplete multipart mapping.
- Native, WASM and Chromium Worker produce identical Diagnostic[], DependencyInspection and Artifact bytes for the same BuildRequest.
- Local acceptance and GitHub Actions status are reported separately. If CI is unavailable, report LOCAL_ACCEPTANCE_PASS, never CI PASS.
- Real Owner/Deployment/5GC qualification remains NOT_QUALIFIED until backed by real versioned publications and live evidence.

