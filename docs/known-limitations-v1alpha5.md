> **Current product baseline (2026-11-23):** Compiler [v2.1.0 RELEASED](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), CA-SL v1alpha5 only, plus separate Development v1alpha1 Build/Verify. Historical discussion of candidate gaps further down this document must not override the [release record](releases/ca-skill-compiler-v2.1.0.md) or the [Compiler-only qualification boundary](COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md).

# Known Limitations — CA-SL v1alpha5 Compiler implementation

**Language status:** FROZEN / RELEASED.
**Compiler static validation status:** `FULL_LANGUAGE_VALIDATION` — every Compiler-owned EW/IN case and every released Compiler-owned static semantic obligation is implemented and tested locally.
**External integration / production qualification (informational only):** NOT_QUALIFIED (Owner / Deployment / Integration Adapter / Host / Runtime / 5GC / MAS). **This is not a Compiler static-validation or release blocker.** See [Compiler-only acceptance](COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md).

Normative authorities:

- [CA-SL v1alpha5](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md), released by ca-skills PR #66.
- [IngressCatalog v1alpha1 logical contract](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md), released by ca-skills PR #62.
- [Ingress diagnostic registry v1alpha1](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/ingress-diagnostic-registry-v1alpha1.md).

## 1. What the released v2.1.0 Build validates

The v1alpha5 path is one pipeline. It validates and assembles:

- strict v1alpha5 request and trusted dependency context admission;
- YAML parsing and duplicate-key handling for supplied author YAML files;
- **the complete inherited v1alpha4 static semantic pipeline** (`validateSkillPackage`), implemented once for the single v1alpha5 target:
  - SKILL.md / Contract / PredicateSet / package Schema source admission;
  - Dependency Alias → immutable snapshot digest → trusted binding → Stable Package ID identity;
  - exact ToolCatalog / Owner / callable / CapabilityContracts linking for every Operation and Observation, Wait or not;
  - Schema Registry, `$ref` closure, root shape, assignability, context/request collision rules;
  - CEL subset checks, Predicate declarations, parameters, evidence requirements, independence and Script ABI;
  - Procedure graph reachability, dominance, stage aggregation, route-key eligibility, final-Postcondition Goal proof;
  - Evidence / Mutation / Reconcile / retry-family static contract checks;
- the v1alpha5 Procedure / Node-entry Wait AST slice;
- Ingress-owned request normalization, immutable binary-content lifecycle and peer profile reference form checks (media-type one-to-one correspondence, mapping target/source/transform semantics, required normalized and wire-field coverage, multipart part and binary bounds, binary retention vs declared recovery window, `alias::resource` profile reference form);
- Wait-specific Procedure graph control flow (EVENT_RECEIVED / TIMEOUT edges composed with the inherited graph);
- trusted Ingress Operation selection and Wait trigger / correlation binding;
- `wait.output` SourcePath and event-accepted-path availability;
- paired-response boundary and typed response mapping;
- deterministic artifact assembly, whose dependency closure now includes the full inherited closure (ordinary callables, Capability members, Schema exports), and self-verification.

The artifact dependency closure and `BuildResult.dependencyClosure` are therefore the same inherited closure for Wait-free and Wait-bearing Skills.

The manifest declares `validationScope.status = FULL_LANGUAGE_VALIDATION` because every Compiler-owned static obligation in the released v1alpha5 and IngressCatalog v1alpha1 contracts is gated. The `notValidated` list retains only non-Compiler scopes (Host/Runtime integration and real Owner/product qualification); it is not a language-validation gap.

## 2. Inherited-language gap: resolved architecture, remaining scope

CA-SL v1alpha5 is not a new language that discards v1alpha4. Its released single-file specification explicitly inherits the v1alpha4 normative semantics except where v1alpha5 Appendix L adds or replaces rules.

The previous candidate ran a detached Wait-only pipeline. That is fixed: one target-neutral inherited semantic pipeline now runs for every selected Skill package, and the v1alpha5 Wait passes compose with it on the same normalized parse result and the same Diagnostic stream. `v1alpha5_artifact.go` re-runs the same passes during assembly and refuses to emit an artifact if any of them would fail.

No Compiler-owned static obligation remains deferred: the Ingress response-contract cases are closed by the released structural admission plus the response-contract checks, and the public API separation is complete (section 6). The inherited Mutation/Reconcile/retry positive-path obligations and Evidence Descriptor linking are implemented and gated by the same pipeline: commit-proof/failure-class/reconcile receipt anchors, minimal commit-proof Schema, failure-class Outcome/no-side-effect/retry rules, same-Owner dedicated READ and Capability coverage, recoverable-output assignability, retry business-identity and NOT_COMMITTED-edge proofs, guarantee descriptor resolution/uniqueness, Producer/`provides.path` matching, reference-root and selection/freshness rules, trusted IndependentDomains/CommonFact binding, the PREDICATE_EVIDENCE_REQUIRED source-exemption rule, and the §17.1 Host-recovery dual-Producer check for consumed recoverable Mutation outputs. All feature-scoped fail-closed Build gates are removed. The 2026-11-18 review hardening additionally rejects malformed trusted Evidence Descriptor containers/bindings (including Producer, selection, freshness, sourceIdentity and factIdentity shapes) and malformed Mutation failure-class/responsibility-record declarations through the public Check/Build entries.

## 3. Non-Wait Tool linking: implemented

Ordinary `operation` and `observation` references are now resolved to exact trusted ToolCatalog exports, Owner identities and callable Schemas by the same inherited pipeline that the v1alpha4 release used for its fully validated cases. A Wait-free Skill that references an unpublishable, ambiguous or untrusted callable fails closed with the inherited `EXPORT_ID_*`, `DEPENDENCY_*` or capability identities.

The remaining limitation is that some callable members are exercised only by synthetic fixtures, not by real Owner publications.

## 4. Ingress / Wait conformance register: complete case inventory, no Compiler-static deferred

`wp7-conformance-v1alpha5.json` now covers all 36 EW and all 66 IN cases. IN-064, IN-065 and IN-066 are present and classified as Skill/Runtime/Integration obligations because their normative conditions are execution-layer behaviors of the released Ingress contract, not Compiler-static checks:

- **IN-064:** Step 15/18 N4 and PDU resource cleanup ordering;
- **IN-065:** no UDM unsubscribe while another matching PDU Session remains;
- **IN-066:** unsubscribe when no matching PDU Session remains.

Current recorded register counts: 55 implemented+tested, 0 Compiler-static deferred, 46 external, 1 spec-only. No Compiler-owned EW/IN obligation remains deferred, so the register supports the `FULL_LANGUAGE_VALIDATION` static-status declaration. IN-058 is classified Host/Admission: the Compiler validates the profile reference form, while exact profile-resource resolution and deployment binding are Host responsibilities because released #62 publishes no profile resource representation.

## 5. Compiler-static EW / IN obligations: none deferred

Every released Compiler-owned EW/IN case is implemented and covered by at least one executed negative or positive test. The register classification is 55 implemented+tested, 0 deferred, 46 external and 1 specification-only.

## 6. API and architecture: single version

The public wire DTOs (`BuildRequest`, `BuildResult`, `Diagnostic`, `DependencyInspection`, inspection rows), the strict JSON transport decoder and the four public entries (`Check`, `Build`, `InspectDependencies`, `VerifyBuildResult`) live in `internal/compiler/api.go`. Admission is single-version (`admission.go`); the internal semantic model is version-neutral (`SkillAST`, linking results, `artifactInput`): one parser, one pipeline, one artifact path and no compatibility adapter.

`connectionagent.io/v1alpha5` is the only supported language target. The retired v1alpha4 API identity, CLI target, artifact generator and verifier are removed; a v1alpha4 `apiVersion` or `languageTarget` fails closed. The v1alpha4 language semantics remain implemented because v1alpha5 inherits them. The published v1.0.0 tag/release remains immutable historical evidence. See [Compiler core and public API](../design/compiler-core-v1alpha5.md).
## 7. Other boundaries

- **Studio:** v1alpha5 successful-Build rendering is not included in the current Compiler integration candidate. Studio/LSP integration requires a separate acceptance.
- **Host / Runtime / Integration:** event admission, durable Wait lifecycle, atomic event-vs-timeout arbitration, tombstone/replay execution, response delivery, retries, authorization revalidation and live SBI transport are not Compiler responsibilities.
- **Real Owner artifacts:** current trusted Catalog / ToolCatalog / Schema / OpenAPI fixtures are synthetic. They prove static algorithm behavior only, not production Owner publication or deployment readiness.
- **Real qualification:** production Owner, Deployment, Integration Adapter, Host/Runtime and 5GC/MAS qualification remain NOT_QUALIFIED.

## 8. Verification status

Local acceptance at commit `0e5fd60a96107c4bd136a6a5b544b78649bfd853` (2026-11-18, Windows/amd64, Go 1.26.6, Node 24.18.0, Chromium 158.0.8127.37): Go tests, vet, CGO-free Native build, WASM build, CLI end-to-end and real Chromium Worker parity PASS. Full record: [v1alpha5 single-version acceptance](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md). Treat this as local evidence until independently reproduced.

GitHub Actions runs `44091765010` and, for this head, `44269138587` concluded failure with zero executed steps (job completed in ~2 seconds, no steps). No source test or build ran in those jobs. CI status is **CI_UNAVAILABLE / NO TEST STEPS EXECUTED**, not CI PASS and not a source-code test failure.

Full acceptance must include actual execution of:

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
git diff --check
```

Local acceptance and GitHub Actions status must be reported separately.
