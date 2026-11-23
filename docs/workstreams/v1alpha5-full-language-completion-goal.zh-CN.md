> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# CA Skill Compiler v1alpha5 — Full Language Completion Goal

**Goal:** deliver a complete, deterministic Compiler for the released CA-SL v1alpha5 language and its released IngressCatalog v1alpha1 companion contract. This is implementation work, not another specification-design phase.

**Repository:** `odango-chan/ca-skill-compiler`

**Integration PR:** [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38)

**Integration branch:** `feat/compiler-v1alpha5-integration-20261117`

**Current integration HEAD at 2026-11-18 review:** `a351b4cd5861b9974726f5e8af5380c876b6d00d`

## 1. Normative authority — use released sources

1. CA-SL v1alpha5: [ca-skills release commit `0abfd0e1`](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md), released through PR #66.
2. IngressCatalog v1alpha1: [ca-skills release commit `9412be98`](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md), released through PR #62.
3. Ingress diagnostic registry v1alpha1: same #62 release commit.
4. Wait semantic and accepted-event tombstone/replay clarifications are incorporated into the released v1alpha5 language / Ingress contract. Use the released single-file documents as authority, not the superseded #61 Draft or #62 pre-release heads.
5. Existing compiler code and WP1–WP8 PRs are implementation material only; they do not override the released normative text.

## 2. Critical language interpretation

CA-SL v1alpha5 explicitly inherits the v1alpha4 normative language semantics except where v1alpha5 adds or replaces them. Appendix L adds Node-entry Event Wait and associated semantics; it does not remove the rest of the language.

Therefore:

- The v1alpha5 Compiler MUST implement the inherited package, Contract, Predicate, Schema, Tool, Capability, Procedure, SourcePath, Script, Evidence, Mutation, Reconcile and retry semantics that are statically owned by Compiler.
- This does **not** mean the Compiler must preserve v1alpha4 API compatibility, old project compatibility, old artifact bytes, old CLI behavior or a second v1alpha4 implementation.
- Published v1.0.0 / v1alpha4 tags and release assets are historical records and MUST NOT be rewritten.
- The active development line may remove, replace or substantially refactor v1alpha4 implementation code when that yields a cleaner v1alpha5 architecture.
- Do not call the old public v1alpha4 Build path as a shortcut. Refactor correct reusable semantic components into target-neutral internal services and make one coherent v1alpha5 pipeline.
- Do not keep dual Parser / Linker / API / artifact paths solely for backward compatibility.

## 3. Current implementation audit

The integrated WP1–WP8 candidate already has useful implementation slices:

- IngressCatalog admission and Schema Closure Digest v1;
- Node-entry Wait AST and Procedure parsing;
- Wait graph control flow;
- exact Ingress export and trigger/correlation binding;
- wait.output SourcePath;
- paired response boundary and typed mapping;
- diagnostic / conformance traceability;
- candidate artifacts, CLI and local Native/WASM/Chromium parity.

However, the current v1alpha5 pipeline does not execute the full inherited static semantics. In particular:

- `internal/compiler/v1alpha5.go` runs the v1alpha5 request admission, YAML parsing, Procedure/Wait AST, Wait graph, Wait/Ingress binding, wait.output and response passes; it does not run the full inherited package / ToolCatalog / Capability / Schema / CEL / Predicate / Script / Evidence / Mutation / Reconcile / retry pipeline.
- ordinary non-Wait `operation` / `observation` references are not all resolved to trusted ToolCatalog exports, Owner identities and callable Schemas;
- required Skill package files and PredicateSet / Contract semantics are not fully admitted;
- historical note: at goal start the artifact declared `validationScope.status = CANDIDATE_PARTIAL`; it now declares `FULL_LANGUAGE_VALIDATION` after all Compiler-owned static obligations were gated;
- the current API model is coupled: `BuildRequest` / `TrustedBuildContext` are declared in `api_v1alpha4.go`, with v1alpha5-only fields on the shared context type;
- the current EW/IN register lists 36 EW + 63 IN = 99 cases, while the released Ingress contract defines IN-001..IN-066. IN-064..IN-066 are missing;
- the current register marks 19 Compiler-static cases deferred;
- the local acceptance report is Harness-reported; the latest recorded GitHub Actions job failed before any step executed.

Treat these as known release blockers, not documentation-only gaps.

## 4. Work packages

### WP9 — Core and API architecture consolidation

1. Define the clean v1alpha5 public BuildRequest, BuildResult, Diagnostic and DependencyInspection contracts.
2. Separate public wire DTOs from target-neutral internal canonical build inputs and trusted dependency models.
3. Remove v1alpha5-only fields from v1alpha4-owned structs; do not preserve the old API solely for compatibility.
4. Consolidate YAML parsing, dependency identity, Owner binding, ToolCatalog / Capability indexes, Schema services, diagnostics and artifact primitives where semantics are genuinely shared.
5. Keep one normalized YAML AST per source file and one coherent v1alpha5 pipeline.
6. Decide which v1alpha4 code should be deleted from the active line after reusable semantic services are extracted. Historical release tags remain untouched.
7. Update CLI help/version and API documentation so v1alpha5 is clearly the active development target and v1alpha4 is only a historical release unless an explicit product decision says otherwise.

**Exit:** no shared public v1alpha4 DTO carrying v1alpha5 fields; no duplicated parser/linker kept only for compatibility; all public API versions are strict and independently documented.

### WP10 — Full Skill package and dependency admission

Implement the inherited package admission and dependency semantics required by v1alpha5:

- required package files and path layout, including `SKILL.md`, Contract, PredicateSet and referenced package Schemas / Script files;
- exact package target selection and multi-Skill isolation;
- dependency declaration ↔ immutable snapshot ↔ trusted binding identity;
- Stable Package ID, digest, Owner binding, export ownership and fail-closed diagnostics;
- ToolCatalog and CapabilityContracts admission and exact export indexing;
- dependency inspection that includes all statically referenced exports, not only Wait-trigger Operations.

Ordinary `operation` and `observation` calls MUST resolve to exact trusted ToolCatalog exports and Owner identities. No name search, fallback or untrusted catalog inference.

**Exit:** Wait-free and Wait-bearing Skills both reject missing/ambiguous/untrusted ordinary callable references; dependency closure includes every referenced export.

### WP11 — Inherited Schema, CEL, Predicate and Script semantics

Implement and test all inherited Compiler-owned static rules:

- Schema root allowlist and package Schema validation;
- Schema Registry, reference closure, dialect rules and exact resolution;
- Source / Output / Context Schema references and assignability;
- the frozen CEL subset and its type/error diagnostics;
- PredicateSet declaration, parameters, argument binding, evaluator and evidence references;
- Script ABI, Python CST-only parsing, input Schema derivation, output typing, call-site argument binding, key access and guards;
- consistent source spans and diagnostics.

Do not execute Python. Do not add a second Schema / CEL / Predicate implementation if a target-neutral existing implementation is correct and reusable.

**Exit:** every inherited static Schema / CEL / Predicate / Script obligation has a clause → implementation → diagnostic → executable test mapping.

### WP12 — Full Procedure and assurance semantics

Implement inherited Procedure semantics in the same graph / type pipeline as Wait:

- start, node IDs, four business stages and stage aggregation;
- global Precondition gate and dominance across every protected path;
- Operation / Observation / Predicate argument and result typing;
- SourcePath grammar, producer identity, type assignability and path availability;
- route target, reachability, DAG and terminal rules;
- final Postcondition PASS as the only successful Goal proof;
- Evidence trust binding, Producer, scope, independence, correlation and freshness;
- Mutation commit / reconcile semantics and retry-family static rules.

Wait entry control must compose with these rules. EVENT_RECEIVED is a control-flow edge, never an Outcome or Goal proof. A Wait edge never satisfies a global Precondition.

**Exit:** a v1alpha5 Skill that exercises ordinary Procedure / assurance semantics receives the same normative static guarantees as the released language, with Wait control integrated rather than bolted on.

### WP13 — Complete Ingress / Wait static conformance

1. Resolve the 19 currently deferred Compiler-static cases:
   - EW-029, EW-031;
   - IN-004, IN-014, IN-015, IN-016, IN-019, IN-027, IN-030, IN-039, IN-040, IN-042, IN-044, IN-045, IN-046, IN-058, IN-059, IN-060, IN-061.
2. Add missing IN-064, IN-065 and IN-066 to the conformance register.
3. Classify every new case against the released contract's owning layer. Do not assume all are Compiler-owned; distinguish static Compiler obligations from Host / Runtime / Integration execution.
4. Preserve the existing diagnostic ownership boundary: reuse dependency, Schema, Tool export and Wait diagnostics where the registry assigns them; do not invent new codes.
5. Reconcile the complete register against EW-001..EW-036 and IN-001..IN-066: exactly 102 case IDs, no omissions or duplicates.
6. Reconcile Appendix H's full diagnostic index. The released language declares 107 base Codes / 108 identities and composes ten Ingress identities (117 Codes / 118 identities total); map each identity to owner, phase and implementation. Compiler must implement all Compiler-owned C_STATIC diagnostics, not all Host/Runtime codes.

**Exit:** every Compiler-owned static EW/IN obligation is implemented and executable-tested; every non-Compiler case has an explicit owner and expected external evidence; the conformance register exactly matches the released sources.

### WP14 — Complete artifact scope and release acceptance

1. Recalculate the artifact dependency closure from all linked non-Wait and Wait exports, Schemas, Capabilities and other referenced dependencies.
2. Update the compiled Skill representation so it does not merely preserve unchecked author references.
3. Reconcile the artifact manifest and JSON Schema with the complete v1alpha5 static validation scope.
4. Remove `CANDIDATE_PARTIAL` only after all Compiler-owned static obligations are genuinely gated by Build and the artifact contract has been reviewed. Do not invent a replacement status without documenting and testing its meaning.
5. Verify BuildResult / Diagnostic / DependencyInspection and artifact self-verification.
6. Run the complete Native, WASM, CLI and real Chromium Worker parity suite against successful and failing full-language fixtures.
7. Add a GitHub Actions workflow that actually executes the complete suite for the integration PR. A run with zero steps is CI_UNAVAILABLE, not PASS or a source-code failure.

**Exit:** a full-language Build cannot succeed with an unchecked Compiler-owned semantic obligation; deterministic artifact bytes and projections agree across Native/WASM/Chromium; CI has real executed steps.

## 5. Acceptance gates

Required commands (plus focused WP tests):

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
git diff --check
```

Also require:

- all inherited v1alpha5 Compiler-owned static rules covered;
- all 102 EW/IN cases classified and all Compiler-static cases executable-tested;
- no unresolved non-Wait Operation / Observation / Capability links;
- no successful full-validation Build with `CANDIDATE_PARTIAL`;
- actual GitHub Actions steps executed and passing;
- synthetic fixtures labelled SYNTHETIC; no false Owner / Deployment / 5GC qualification.

Local acceptance and CI evidence must be reported separately. A local PASS is never a CI PASS.

## 6. GitHub delivery rules

- Continue on the integration branch `feat/compiler-v1alpha5-integration-20261117` and update PR #38; it already contains the WP1–WP8 child PR heads.
- Do not create duplicate PRs for the same integrated WP work.
- Do not push directly to `main`; do not force-push; do not discard another Agent's commits.
- Commit and push every implementation slice to GitHub, update PR #38 with the actual scope and test evidence, then verify remote HEAD SHA.
- Keep #38 Draft until all acceptance gates pass. Do not merge it automatically.
- Child PR status checked 2026-11-18: #30–#37 are already closed without merging to `main`; #39 and #40 are already merged into the integration feature branch. Preserve all commits in #38; do not reopen, merge independently, or replay child PRs.
- If remote write or CI is blocked, report `REMOTE_WRITE_BLOCKED` or `CI_UNAVAILABLE` with exact evidence. Never describe local-only work as repository delivery.

## 7. Per-run report

Every run must state:

1. branch, PR and exact remote HEAD SHA;
2. WP completed and files changed;
3. normative clauses and diagnostics covered;
4. exact commands executed, exit codes and results;
5. whether GitHub Actions actually executed steps;
6. remaining blockers and next WP;
7. any destructive refactor and why it improves the v1alpha5 architecture;
8. whether the remote push succeeded and the verified remote SHA.

**Final Goal:** one clean, fully validated CA-SL v1alpha5 Compiler implementation. Language publication is already complete; do not spend this phase re-freezing documents that are already released.
