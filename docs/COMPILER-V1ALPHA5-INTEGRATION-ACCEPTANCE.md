# CA Skill Compiler v1alpha5 — integration acceptance

**Implementation status: `FULL_LANGUAGE_VALIDATION` for the Compiler-owned static scope. Product/Owner/Host qualification remains NOT_QUALIFIED. WP9 public API separation is complete. Review readiness for PR #38 is based on local acceptance at the final PR HEAD.**

The language and Ingress logical contracts are published. The EW/IN register classifies 55 cases as implemented and tested, with 0 Compiler-static deferred. The earlier feature-scoped fail-closed gates (Evidence Descriptor, Mutation/Reconcile, retry) are removed and replaced by complete static checks.

- CA-SL v1alpha5: **FROZEN / RELEASED**, `connectionagent.io/v1alpha5`, ca-skills PR #66, merge commit `0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5`.
- IngressCatalog: **FROZEN / RELEASED**, `connectionagent.io/ingresscatalog/v1alpha1`, ca-skills PR #62, merge commit `9412be9894ef36acf846c757a5870dc0c48b33ac`.
- Ingress diagnostics: **FROZEN / RELEASED**, `connectionagent.io/ingress-diagnostics/v1alpha1`, released with PR #62.
- CA-SL PR #61 is closed without merge; PR #66 is the release PR and the normative authority. Do not cite #61 as the current publication.
- Real Owner / Deployment / Integration Adapter / Host / Runtime / 5GC / MAS product qualification remains **NOT_QUALIFIED**.

## 1. Current integration candidate

| Item | Value |
|---|---|
| Repository | `odango-chan/ca-skill-compiler` |
| Integration branch | `feat/compiler-v1alpha5-integration-20261117` |
| Integration HEAD | Tracked by live PR #38; see GitHub for its current head SHA |
| Base | `main` |
| PR | [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38) |
| PR status and changed files | See live PR #38 |

The following implementation slices are represented in #38. The child PRs are not independent merge candidates for `main`:

- #30 WP2 Wait AST / Procedure admission
- #31 WP1 IngressCatalog / Schema Closure
- #32 WP5 Procedure Graph
- #33 WP7 conformance / diagnostic traceability
- #34 WP8 CLI / Native / WASM / Chromium parity
- #35 WP3 Trigger / Correlation
- #36 WP4 wait.output SourcePath
- #37 WP6 Response Boundary
- #39 WP7b DependencyInspection
- #40 deterministic Build artifacts

These are stacked implementation slices, not independent changes to merge to `main`. #38 is the single integration candidate. As of 2026-11-18, PRs #30–#37 are closed without merging to `main`; #39 and #40 were merged into the #38 feature branch. Their commits are retained in #38. No child PR needs to be merged independently or replayed.

## 2. What WP1–WP9 currently implement

- **WP1 — IngressCatalog / Schema Closure:** trusted IngressCatalog admission; raw artifact-byte digest verification; dialect-aware Schema root and nested `$id` resolution; exact transitive reference closure; JCS Schema Closure Digest v1; golden vector coverage.
- **WP2 — Wait AST:** Node-entry Wait parsing integrated into a v1alpha5 Procedure AST; FixedDuration subset; exact selector fields; response boundary syntax; TIMEOUT route rules; source-positioned diagnostics.
- **WP3 — Trigger / Correlation:** exact trusted Ingress export selection; exact ToolCatalog trigger Operation binding; typed arm-input and correlation checks on incoming paths.
- **WP4 — Wait Output:** `NodeId.wait.output` SourcePath resolution against the normalized request Schema; path availability and conservative argument typing.
- **WP5 — Procedure Graph:** Wait entry-control vertices; implicit EVENT_RECEIVED / TIMEOUT edges; graph reachability, DAG, dominance and final Postcondition-PASS goal checks.
- **WP6 — Response Boundary:** paired response boundary, producer stage and response-result Schema mapping checks.
- **WP7 — Diagnostic / Conformance:** Wait and emitted Ingress diagnostic traceability; current EW/IN register.
- **WP8 — Build / CLI / Parity:** deterministic candidate artifacts, manifest verification, CLI target dispatch, Ingress inventory projection, Native/WASM/real-Chromium parity for the projections exercised by the current harness.
- **WP9 — Unified Core:** one target-neutral inherited static semantic pipeline shared by the v1alpha4 historical entry and the v1alpha5 active target; the Appendix L Wait passes compose with it on the same normalized parse result and the same Diagnostic stream; the artifact dependency closure is the full inherited closure.

## 3. Compiler static scope

### 3.1 v1alpha5 inherits the v1alpha4 language semantics — pipeline unified

The released v1alpha5 single-file specification explicitly defines v1alpha4 as its immutable normative inheritance baseline. Appendix L adds or replaces the Wait-related rules; it does not discard the inherited language.

The v1alpha5 entry path now runs **one** inherited static semantic pipeline (`validateInheritedSkillPackage` in `internal/compiler/inherited_semantics.go`) on the same normalized parse result as the Wait passes. The historical v1alpha4 entry point uses the same function, so there is no second semantic implementation. The pipeline covers:

- required Skill package files and package structure (`SKILL.md`, Contract, PredicateSet, package Schemas and Script sources);
- non-Wait Operation / Observation ToolCatalog linking, Owner attribution, dependency closure and ordinary callable Schema binding;
- Capability linking and ordinary Procedure argument / SourcePath typing;
- inherited Schema allowlist, registry/closure, assignability and CEL checks;
- Predicate declarations, parameter signatures, evaluator and Predicate Script ABI;
- global Precondition, stage aggregation, route and final Postcondition obligations;
- Evidence trust, Producer, independence, scope, correlation and freshness checks;
- Mutation, Reconcile and retry-family static contracts;
- inherited diagnostic ownership and static conformance coverage.

`v1alpha5_artifact.go` re-runs the same passes during assembly and refuses to emit an artifact if any would fail. The artifact dependency closure and `BuildResult.dependencyClosure` now include the full inherited closure, not just the Wait Ingress links.

The inherited Mutation/Reconcile/retry positive-path obligations and Evidence Descriptor linking are now implemented and gated by the same pipeline (commit-proof/failure-class/reconcile receipt anchors, minimal proof Schema, Owner/READ/Capability coverage, recoverable-output assignability, retry business-identity and NOT_COMMITTED-edge proofs; guarantee descriptor resolution/uniqueness, Producer/provides.path matching, reference-root/selection/freshness rules, trusted IndependentDomains/CommonFact binding and PREDICATE_EVIDENCE_REQUIRED). All feature-scoped Build gates are removed. The §17.1 Host-recovery dual-Producer static check is also implemented.

### 3.2 Ingress / Wait conformance register

The register now reports all 102 cases: 36 EW + 66 IN. The released Ingress contract defines **IN-001 through IN-066**; IN-064, IN-065 and IN-066 are present and classified as Skill/Runtime/Integration obligations, not Compiler-static ones. The three cases cover Step 15/18 N4 and PDU resource cleanup distinctions and the conditional Step 21 UDM unsubscribe behavior for other sessions sharing the same SUPI / DNN / HPLMN S-NSSAI.

Current register classification: 55 implemented+tested, 0 Compiler-static deferred, 46 Host/Runtime/Integration-owned, 1 specification-only. Each implemented row links to named tests. IN-058 is Host/Admission-owned for exact profile-resource resolution because released #62 publishes no profile resource representation. IN-030 now aligns the missing export-declared Producer stage at the response boundary with WAIT_RESPONSE_BOUNDARY_INVALID, and IN-044 cross-checks the pinned OpenAPI response shape (multipart/related) against the declared response representations. The unified pipeline also covers the inherited EW-031 global-Precondition dominance proof, the inherited IN-019 self-output SOURCE_NOT_AVAILABLE rule, exact Ingress dependency Owner binding (IN-004), duplicate response disposition id/status (IN-040) and the authenticated-header-parameter wire-discriminator identity consistency (IN-046/IN-060). The request-normalization, immutable binary-content lifecycle and peer profile reference form rules are implemented with negative tests; expected-profile-resource resolution belongs to Host Admission for IN-058 because the released contract does not publish a profile resource representation.

### 3.3 Public API separation: completed

The shared public wire DTOs and strict JSON decoder now live in the target-neutral `internal/compiler/api.go`; the v1alpha5 public entries (`CheckV1Alpha5`, `BuildV1Alpha5`, `InspectV1Alpha5Dependencies`) live in `api_v1alpha5.go`; and `api_v1alpha4_target.go` holds only the frozen historical v1alpha4 envelope admission. The internal semantic model remains shared and version-neutral with one parser, one pipeline and one artifact path. See [Compiler core and public API](../design/compiler-core-v1alpha5.md).
## 4. Review acceptance

Run the local gates in [AGENTS.md](../AGENTS.md) against the final PR HEAD and record the commit, tool versions and actual results in [PR #38](https://github.com/odango-chan/ca-skill-compiler/pull/38). Check the entire PR diff for whitespace errors. The EW/IN register classifies cases and links Compiler-owned entries to named tests.

GitHub Actions is excluded from this PR's review-readiness decision. The checks on the previously reviewed commit `25ef0f5` stopped before any steps ran because GitHub reported a billing or spending-limit restriction. Local results are recorded separately in the PR.

The CLI still labels v1alpha5 as candidate and exposes the frozen v1alpha4 target. v1alpha5 is the active language target for ongoing development.

## 5. Qualification and release boundary

The following are independent of Compiler static implementation and remain unqualified until real evidence exists:

- production Owner packages and trusted export publication;
- real Deployment authorization / profile binding;
- Integration Adapter wire conformance, including HTTP/2 and multipart/binary handling;
- Host/Runtime event admission, durable Wait lifecycle, tombstone/replay isolation and response delivery;
- real 5GC / MAS end-to-end interoperability.

Synthetic fixtures are acceptable for deterministic Compiler tests, but must never be reported as real Owner, Deployment or 5GC qualification.

Ready for Review opens code review. Merge and release decisions are separate.
