# Known Limitations — CA-SL v1alpha5 Compiler implementation

**Language status:** FROZEN / RELEASED.  
**Compiler status:** CANDIDATE / PARTIAL STATIC COVERAGE — NOT READY TO RELEASE.  
**Product qualification:** NOT_QUALIFIED.

Normative authorities:

- [CA-SL v1alpha5](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md), released by ca-skills PR #66.
- [IngressCatalog v1alpha1 logical contract](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md), released by ca-skills PR #62.
- [Ingress diagnostic registry v1alpha1](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/ingress-diagnostic-registry-v1alpha1.md).

## 1. What the current candidate Build actually validates

The current v1alpha5 path validates and assembles:

- strict v1alpha5 request and trusted dependency context admission;
- YAML parsing and duplicate-key handling for supplied author YAML files;
- the v1alpha5 Procedure / Node-entry Wait AST slice;
- Wait-specific Procedure graph control flow;
- trusted Ingress Operation selection and Wait trigger / correlation binding;
- `wait.output` SourcePath and event-accepted-path availability;
- paired-response boundary and typed response mapping;
- candidate deterministic artifact assembly and self-verification.

The manifest marks this honestly as `validationScope.status = CANDIDATE_PARTIAL`. A consumer requiring full static validation MUST reject a partial artifact.

## 2. Critical inherited-language gap

CA-SL v1alpha5 is not a new language that discards v1alpha4. Its released single-file specification explicitly inherits the v1alpha4 normative semantics except where v1alpha5 Appendix L adds or replaces rules.

The current v1alpha5 pipeline does not execute the full inherited static semantic pipeline. It must not be described as a complete v1alpha5 Compiler.

Missing or not yet demonstrated in the v1alpha5 path:

- complete Skill package admission: `SKILL.md`, `contract.yaml`, `predicates.yaml`, package Schemas and Script source/ABI;
- non-Wait Operation and Observation ToolCatalog export linking, exact Owner attribution, dependency closure and callable input/output Schema binding;
- Capability linking and normal Procedure argument typing;
- inherited Schema allowlist, Schema registry/closure, assignability and CEL checks;
- Predicate declarations, parameter signatures, evaluator and Script ABI checks;
- complete global Precondition, stage, route, terminal and final Postcondition rules;
- Evidence trust binding, Producer / scope / independence / correlation / freshness checks;
- Mutation, Reconcile and retry-family static semantics;
- complete inherited C_STATIC diagnostic coverage and conformance traceability.

The v1alpha4 implementation may provide reusable algorithms and tests, but v1alpha5 must implement the released v1alpha5 contract through a coherent pipeline. Calling the old public v1alpha4 Build entry point is not a substitute for a reviewed v1alpha5 pipeline.

## 3. Non-Wait Tool linking is a release blocker

Only Wait-linked Ingress exports and their trigger Operations are currently linked in the v1alpha5 candidate. Ordinary `operation` and `observation` references outside Wait triggers may be preserved in the compiled Procedure document without being resolved to trusted ToolCatalog exports, Owner identity or Schema bindings.

This means a Wait-free candidate Skill can Build while ordinary callable references remain unchecked. Full v1alpha5 Build must fail closed on unresolved or untrusted non-Wait callable references.

## 4. Ingress / Wait conformance register is incomplete

The current `wp7-conformance-v1alpha5.json` lists 36 EW and only 63 IN cases. The released Ingress contract defines IN-001..IN-066. The register must add and classify:

- **IN-064:** Step 15 failure variants and Step 18 N4 / PDU resource cleanup without duplicate release or delayed response;
- **IN-065:** do not unsubscribe the shared SUPI / DNN / HPLMN S-NSSAI subscription while another matching PDU Session remains;
- **IN-066:** unsubscribe when no other matching PDU Session remains, separately from Step 18 cleanup.

Do not assume these three obligations are all Compiler-owned. Assign Compiler / Host / Runtime / Integration ownership from the normative conditions and add executable tests only for the Compiler-owned static portions.

Current recorded register counts (37 implemented+tested, 19 Compiler-static deferred, 42 external, 1 spec-only) are provisional because the three missing IN cases have not been classified.

## 5. Deferred Compiler-static EW / IN obligations

The current register marks 19 Compiler-owned static obligations deferred:

| Case(s) | Required static coverage |
|---|---|
| EW-029 | Reject body-bearing response mapping sourced from wait.output, Context or raw request instead of the declared boundary Producer |
| EW-031 | Global Precondition dominance over protected Mutation / success; Wait edge does not satisfy the gate |
| IN-004 | Exact dependency Owner binding and mismatch diagnostics |
| IN-014, IN-015, IN-039, IN-059 | Complete normalization declaration, one-to-one media/schema/representation correspondence, required wire-field mapping |
| IN-016, IN-042 | Immutable multipart/binary content reference and retention/recovery lifecycle |
| IN-019 | Reject trigger arguments that depend on the trigger Operation's own not-yet-produced output |
| IN-027, IN-040, IN-061 | Complete paired-response deadline, recovery, disposition and response-contract validation |
| IN-030 | Require the export-declared Producer stage at the response boundary |
| IN-044, IN-045 | Validate multipart/binary and N1 SM Reject response representation/mapping |
| IN-046, IN-060 | Validate Ingress-owned correlation identity, scope, duplicate and stale-delivery declarations |
| IN-058 | Validate authenticationProfileRef / authorizationPolicyRef syntax and resolution |

The Compiler must implement every Compiler-owned static condition in this list or record a precise normative reason why it is owned elsewhere. It must not silently omit the cases from the register.

## 6. API and architecture limitations

The current code places the shared `BuildRequest` and `TrustedBuildContext` declarations in `api_v1alpha4.go`, while the shared context struct also carries v1alpha5-only artifact and Schema fields. The v1alpha4 validation rejects those fields.

This is an unnecessary coupling. There is no requirement to preserve v1alpha4 public API or old project compatibility on the active development line. Refactor toward:

- a clean v1alpha5 public BuildRequest / BuildResult / DependencyInspection contract;
- a target-neutral internal canonical input model;
- one normalized YAML AST and reusable semantic services;
- one coherent v1alpha5 pipeline, including inherited language semantics and Wait / Ingress additions;
- no dual Parser, Linker or compatibility adapters retained solely for v1alpha4.

The published v1.0.0 tag/release remains a historical artifact and is not rewritten. The active development branch may remove or replace v1alpha4 code when doing so improves the architecture.

## 7. Other boundaries

- **Studio:** v1alpha5 successful-Build rendering is not included in the current Compiler integration candidate. Studio/LSP integration requires a separate acceptance.
- **Host / Runtime / Integration:** event admission, durable Wait lifecycle, atomic event-vs-timeout arbitration, tombstone/replay execution, response delivery, retries, authorization revalidation and live SBI transport are not Compiler responsibilities.
- **Real Owner artifacts:** current trusted Catalog / ToolCatalog / Schema / OpenAPI fixtures are synthetic. They prove static algorithm behavior only, not production Owner publication or deployment readiness.
- **Real qualification:** production Owner, Deployment, Integration Adapter, Host/Runtime and 5GC/MAS qualification remain NOT_QUALIFIED.

## 8. Verification status

Harness recorded local PASS for Go tests, vet, CGO-free Native build, WASM build, CLI end-to-end and real Chromium Worker parity on the integration branch. Treat these as Harness-reported local evidence until independently reproduced.

GitHub Actions run `44091765010` concluded failure, but its job had zero executed steps. No source test or build ran in that job. CI status is therefore **CI_UNAVAILABLE / NO TEST STEPS EXECUTED**, not CI PASS and not a source-code test failure.

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
