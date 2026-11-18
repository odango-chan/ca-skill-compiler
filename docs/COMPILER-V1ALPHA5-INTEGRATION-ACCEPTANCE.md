# CA Skill Compiler v1alpha5 — integration acceptance

**Implementation status: CANDIDATE / PARTIAL STATIC COVERAGE. NOT READY TO MERGE OR RELEASE.**

The language and Ingress logical contracts are now published. The Compiler implementation is not yet a complete implementation of the released language.

- CA-SL v1alpha5: **FROZEN / RELEASED**, `connectionagent.io/v1alpha5`, ca-skills PR #66, merge commit `0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5`.
- IngressCatalog: **FROZEN / RELEASED**, `connectionagent.io/ingresscatalog/v1alpha1`, ca-skills PR #62, merge commit `9412be9894ef36acf846c757a5870dc0c48b33ac`.
- Ingress diagnostics: **FROZEN / RELEASED**, `connectionagent.io/ingress-diagnostics/v1alpha1`, released with PR #62.
- CA-SL PR #61 is closed without merge; PR #66 is the release PR and the normative authority. Do not cite #61 as the current publication.
- Compiler PR #38 remains Draft. No Compiler PR has been merged to `main`.
- Real Owner / Deployment / Integration Adapter / Host / Runtime / 5GC / MAS product qualification remains **NOT_QUALIFIED**.

## 1. Current integration candidate

| Item | Value |
|---|---|
| Repository | `odango-chan/ca-skill-compiler` |
| Integration branch | `feat/compiler-v1alpha5-integration-20261117` |
| Integration HEAD | Tracked by live PR #38; see GitHub for its current head SHA |
| Base | `main` |
| PR | [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38) |
| Commits / changed files | 47 commits / 76 files at the latest integration review; documentation/CI maintenance commits are also on this branch |
| PR status | OPEN / DRAFT |

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

## 2. What WP1–WP8 currently implement

- **WP1 — IngressCatalog / Schema Closure:** trusted IngressCatalog admission; raw artifact-byte digest verification; dialect-aware Schema root and nested `$id` resolution; exact transitive reference closure; JCS Schema Closure Digest v1; golden vector coverage.
- **WP2 — Wait AST:** Node-entry Wait parsing integrated into a v1alpha5 Procedure AST; FixedDuration subset; exact selector fields; response boundary syntax; TIMEOUT route rules; source-positioned diagnostics.
- **WP3 — Trigger / Correlation:** exact trusted Ingress export selection; exact ToolCatalog trigger Operation binding; typed arm-input and correlation checks on incoming paths.
- **WP4 — Wait Output:** `NodeId.wait.output` SourcePath resolution against the normalized request Schema; path availability and conservative argument typing.
- **WP5 — Procedure Graph:** Wait entry-control vertices; implicit EVENT_RECEIVED / TIMEOUT edges; graph reachability, DAG, dominance and final Postcondition-PASS goal checks.
- **WP6 — Response Boundary:** paired response boundary, producer stage and response-result Schema mapping checks.
- **WP7 — Diagnostic / Conformance:** Wait and emitted Ingress diagnostic traceability; current EW/IN register.
- **WP8 — Build / CLI / Parity:** deterministic candidate artifacts, manifest verification, CLI target dispatch, Ingress inventory projection, Native/WASM/real-Chromium parity for the projections exercised by the current harness.

These capabilities are useful and should be retained where they conform to the released specifications. They do not establish full v1alpha5 compiler conformance.

## 3. Critical uncovered scope

### 3.1 v1alpha5 inherits the v1alpha4 language semantics

The released v1alpha5 single-file specification explicitly defines v1alpha4 as its immutable normative inheritance baseline. Appendix L adds or replaces the Wait-related rules; it does not discard the inherited language.

The current v1alpha5 entry path in `internal/compiler/v1alpha5.go` runs its own request admission, YAML parsing, Procedure/Wait AST, Wait graph, Ingress/trigger/correlation, wait.output, response-boundary and artifact passes. It does **not** run the full inherited static semantic pipeline.

Consequently, the current candidate does not yet validate all inherited v1alpha5 obligations, including:

- required Skill package files and package structure (`SKILL.md`, Contract, PredicateSet, package Schemas and Script sources);
- non-Wait Operation / Observation ToolCatalog linking, Owner attribution, dependency closure and ordinary callable Schema binding;
- Capability linking and all ordinary Procedure argument / SourcePath typing;
- inherited Schema allowlist, registry/closure, assignability and CEL checks;
- Predicate declarations, parameter signatures, evaluator and Predicate Script ABI;
- global Precondition, stage aggregation, route and final Postcondition obligations beyond the Wait-specific graph slice;
- Evidence trust, Producer, independence, scope, correlation and freshness checks;
- Mutation, Reconcile and retry-family static contracts;
- complete inherited diagnostic ownership and static conformance coverage.

The current code can preserve ordinary Operation / Observation references in the compiled Procedure while not resolving them to trusted ToolCatalog exports. A Wait-free Skill can therefore produce a candidate Build even though ordinary tool references and package semantics have not been fully validated.

This is a **v1alpha5 implementation gap**, not a request to preserve the old v1alpha4 API or old-project compatibility.

### 3.2 Ingress / Wait conformance register is incomplete

The current register reports 99 cases: 36 EW + 63 IN. The released Ingress contract defines **IN-001 through IN-066**, not IN-001 through IN-063.

The register is missing IN-064, IN-065 and IN-066. These cover Step 15/18 N4 and PDU resource cleanup distinctions, and the conditional Step 21 UDM unsubscribe behavior for other sessions sharing the same SUPI / DNN / HPLMN S-NSSAI. Add all three and classify their owning layer from the released contract; do not assume all are Compiler-owned.

Current register classification: 37 implemented+tested, 19 Compiler-static deferred, 42 Host/Runtime/Integration-owned, 1 specification-only. These counts must be recalculated after adding IN-064..IN-066 and reconciling every row against the released contract.

### 3.3 Versioned API types are not yet cleanly separated

The current `BuildRequest`, `TrustedBuildContext` and shared result types are declared in `api_v1alpha4.go`; v1alpha5-only trusted artifact / Schema fields have been added to the shared context type and are rejected by the v1alpha4 validation path.

This is an implementation coupling, not a required compatibility design. Refactor toward a clean v1alpha5 public request/result/inspection model and a target-neutral internal build model. Do not retain a v1alpha4 adapter or parallel API solely for backward compatibility.

## 4. Local acceptance evidence recorded by Harness

The integration acceptance report records the following commands as executed on Windows / Go 1.26.6 / Node 24.18.0 / Chrome 158:

| Gate | Recorded result |
|---|---|
| `GOWORK=off go test ./... -count=1` | PASS |
| `GOWORK=off go vet ./...` | PASS |
| `CGO_ENABLED=0 GOWORK=off go build ./...` | PASS |
| `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS |
| CLI end-to-end | PASS (13 recorded v1alpha4/v1alpha5 tests) |
| Real Chromium Worker parity | PASS (recorded `TestChromiumWorkerParity`, 36.27s) |
| `git diff --check` | PASS |

These are **Harness-reported local results** from the integration branch. They have not been independently rerun as part of this review and must not be described as GitHub Actions results.

### GitHub Actions

Latest integration workflow runs are `44221402383` (pull_request) and `44221394375` (push). Both concluded `failure`, but jobs `134523879855` and `134523853739` have **zero executed steps**. No Go test, vet, Native/WASM build or Chromium command ran. Earlier run `44220845990` had the same no-step failure mode. This is not evidence of a source-code test failure.

Status: **CI_UNAVAILABLE / NO TEST STEPS EXECUTED**. The workflow was updated to target the integration branch and now defines Go tests, vet, Native/WASM builds, CLI E2E and real Chromium Worker parity. GitHub still fails before the first step on both push and pull_request events; the runner/infrastructure cause remains undetermined from available job metadata. Harness-reported local PASS has not been independently reproduced at the latest integration HEAD; rerun locally before using it as release evidence.

## 5. Required next implementation phase

The next phase is defined in [v1alpha5 Full Language Completion Goal](workstreams/v1alpha5-full-language-completion-goal.zh-CN.md).

It must:

1. Refactor the API and compiler pipeline so v1alpha5 is the clean active target; preserve only target-neutral semantic components that are correct and useful.
2. Implement all inherited v1alpha4 normative semantics as v1alpha5 semantics, without preserving the old v1alpha4 API or old project compatibility.
3. Resolve every non-Wait Operation / Observation / Capability / Schema / Predicate / Script / Evidence / Mutation / Reconcile / retry obligation owned by Compiler.
4. Complete the 19 currently deferred Compiler-static EW/IN cases and add/classify IN-064..IN-066.
5. Reconcile the complete v1alpha5 diagnostic index and all Compiler-owned C_STATIC rules.
6. Remove `CANDIDATE_PARTIAL` from any release-eligible artifact only after the artifact contract and full static validation scope are complete and reviewed.
7. Re-run Go tests, vet, Native, WASM, CLI, real Chromium Worker parity and the actual GitHub Actions workflow.

## 6. Qualification and release boundary

The following are independent of Compiler static implementation and remain unqualified until real evidence exists:

- production Owner packages and trusted export publication;
- real Deployment authorization / profile binding;
- Integration Adapter wire conformance, including HTTP/2 and multipart/binary handling;
- Host/Runtime event admission, durable Wait lifecycle, tombstone/replay isolation and response delivery;
- real 5GC / MAS end-to-end interoperability.

Synthetic fixtures are acceptable for deterministic Compiler tests, but must never be reported as real Owner, Deployment or 5GC qualification.

**Merge decision: HOLD.** Keep #38 Draft until the inherited static semantics, missing Ingress obligations, API architecture, complete artifact scope and real CI execution are addressed. No merge or release is authorized by this acceptance document.
