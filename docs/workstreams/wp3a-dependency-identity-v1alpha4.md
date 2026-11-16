# WP3A — Dependency Snapshot Identity and Trust Binding

**Task ID:** `WP3A-DEPENDENCY-IDENTITY-001`  
**Status:** READY FOR DEEPSEEK IMPLEMENTATION  
**Scope:** Compiler / static linking only  
**Task branch:** `feat/compiler-v1alpha4-wp3a-dependency-identity-20261116`  
**Required PR base:** `feat/compiler-v1alpha4-wp2-yaml-ast-20261116`

## 1. Goal

Implement the smallest complete and testable Compiler slice that establishes the identity of each dependency snapshot actually used by the selected Skill.

For every used Dependency Alias, the Compiler must prove this exact relation:

`Contract Alias -> one ResolvedDependency -> recomputed snapshot digest -> one TrustedDependencyBinding -> exact Stable Package ID`

No identity may be inferred from a filename, repository URL, package display name, Owner name, Schema ID prefix, ToolCatalog metadata, or a matching digest without the trusted binding.

A valid WP3A result is **only** proof that the supplied snapshot bytes are bound to the requested package identity. It is not proof that a callable exists, an Owner is authorized to publish a particular export, a Capability is granted, or a Skill can be built successfully.

## 2. Frozen authority

### Normative language authority

- Repository: `odango-chan/ca-skills`
- Commit: `7f84393ee8c395e4d8c39a74cb5000a1f896588d`
- File: `docs/skill-language-spec.v1alpha4.zh-CN.md`
- Blob: `b70064c5737b8ee2b3d1435fe496357f44b42773`

Relevant normative sections:
- §24.1: exact diagnostic conditions and precedence;
- §§34.0–34.0.2: Dependency Alias, immutable snapshot identity, trusted binding and Owner attribution;
- F.0–F.2: dependency and callable export contract boundaries;
- Appendix E: concrete positive and negative examples;
- Appendix H: diagnostic Code + Phase identities;
- Appendix K: execution responsibility classification.

### Compiler-owned implementation contracts

Read and follow:
- `docs/design/compiler-contract-v1alpha4.md`
- `docs/design/build-request-v1alpha4.md`
- `docs/design/compiler-output-contract-v1alpha4.md`
- `docs/design/procedure-link-v1alpha4.md` (only for locating the exact Alias references)
- `docs/workstreams/conformance-v1alpha4.md`
- `tests/conformance/v1alpha4/case-index.json`
- `tests/conformance/v1alpha4/diagnostic-index.json`
- `tests/conformance/v1alpha4/wp3a-fixture-plan.json`

Machine contracts:
- `contracts/v1alpha4/build-request.schema.json`
- `contracts/v1alpha4/build-result.schema.json`
- `contracts/v1alpha4/dependency-inspection.schema.json`

The compiler contract documents are subordinate to the pinned CA-SL specification. If they conflict, stop and report the exact conflict; do not silently choose.

## 3. Current baseline and existing code

The task branch starts from WP2 head:

`55ba5ebc37e9d54b6ee21e547dae188ca17785c9`

Relevant existing implementation:
- `internal/compiler/dependency_snapshot_digest.go`: WP1 digest algorithm `ComputeDependencySnapshotDigest`. Reuse it; do not implement a second digest.
- `internal/compiler/api_v1alpha4.go`: `BuildRequest`, `ResolvedDependency`, `TrustedDependencyBinding`, `TrustedBuildContext`, `Check`, `Build`, `InspectDependencies`, request validation and result envelopes.
- `internal/compiler/ast_v1alpha4.go`: typed Contract / Procedure / PredicateSet AST projections.
- `internal/compiler/v1alpha4.go`: current v1alpha4 source validation and fail-closed placeholders.
- `internal/compiler/yamlast`: the single normalized YAML AST introduced by WP2.

Do not copy the separate experimental branch `feat/compiler-v1alpha4-wp3-linker-20261116`. It is an unreviewed one-file draft, is not the approved base and is not evidence of a working linker.

Before editing, run the baseline commands in §10. If they fail on the clean WP2 base, stop and report the exact failure. Do not repair unrelated WP1/WP2 defects inside WP3A.

## 4. Required behavior

### 4.1 Verify supplied snapshot bytes

For every supplied `ResolvedDependency`:
1. Recompute the digest from the original file paths and original file bytes using `ComputeDependencySnapshotDigest`.
2. Compare the measured digest with `ResolvedDependency.digest`.
3. Do not parse snapshot YAML before the digest check succeeds.
4. Do not normalize YAML and then hash the normalized representation.
5. Preserve the WP1 digest framing and order independence.

A mismatch must fail closed using the existing frozen dependency identity diagnostic. Do not introduce a new diagnostic code.

### 4.2 Resolve Contract dependencies

For the selected Skill only:
1. Read the Contract's `dependencies` mapping from the normalized WP2 AST.
2. Each Alias must satisfy the frozen identifier grammar.
3. Each value must be a Stable Package ID string; it is not a display name or repository location.
4. Collect Alias references only from the selected Skill's supported source positions: external callable references, external Schema references, and Evidence Guarantee references. The Evidence descriptor itself remains WP7 scope.
5. Do not recursively scan arbitrary YAML keys for strings containing `::`.
6. A referenced Alias absent from Contract.dependencies is an error.
7. A declared dependency that is not referenced is a warning, not an implicit global export.

### 4.3 Bind each used Alias to exactly one snapshot

For each used Alias:
1. Find the supplied `ResolvedDependency` whose `dependencyId` exactly equals the Alias.
2. Require exactly one match.
3. Find the trusted `TrustedDependencyBinding` matching both the exact `dependencyId` and the recomputed digest.
4. Require exactly one matching binding.
5. Compare `TrustedDependencyBinding.stablePackageId` with the Contract's Stable Package ID using exact, case-sensitive equality.
6. Retain the trusted binding as metadata for later WP3B Owner attribution; do not infer or synthesize Owner records.

Do not match by list order, package name, path, URI prefix, case folding, partial digest, or first match.

### 4.4 Selected dependency closure

- Only aliases actually referenced by the selected Skill enter that Skill's dependency closure.
- A supplied but unreferenced extra snapshot is not a global export source and cannot satisfy any reference. This is Appendix E16.
- Recompute the digest of every supplied snapshot as required by the Compiler pipeline, but do not parse/index/expose unreferenced exports during WP3A.
- Preserve deterministic ordering of aliases and diagnostics.
- If multiple Skills are selected, their alias sets and closures must remain isolated; no cross-Skill export leakage.

### 4.5 Public API and fail-closed behavior

- `Check` and `Build` must consume the same WP3A validation result; do not implement divergent identity checks.
- `Build` must remain failure-atomic: if any later stage is not implemented or any error exists, return no output files and no partial dependency closure.
- Replace the generic dependency placeholder only when WP3A can report the exact frozen dependency diagnostic.
- A valid WP3A identity result must not remove the later Capability/Owner/Schema/Evidence fail-closed boundary.
- `InspectDependencies` must not claim a trusted callable/schema inventory before WP3B/3D. It may return precise WP3A identity diagnostics, but until the inventory linker is implemented it must remain explicitly fail-closed with empty inventory arrays.
- Do not return `Success=true` merely because dependency identity passes.

## 5. Frozen diagnostics in this task

Only these diagnostics are in WP3A scope. Preserve their exact Code + Phase identity.

| Code | Phase | Trigger in WP3A |
|---|---|---|
| `DEPENDENCY_REQUIREMENT_INVALID` | `C_STATIC` | Contract dependency value is not a valid Stable Package ID string |
| `DEPENDENCY_ALIAS_MISSING` | `C_STATIC` | A supported external reference uses an Alias not declared by the Contract |
| `DEPENDENCY_SNAPSHOT_MISSING` | `C_STATIC` | A used Contract Alias has no supplied snapshot |
| `DEPENDENCY_IDENTITY_UNVERIFIED` | `C_STATIC` | No trusted binding matches the exact Alias and measured snapshot digest |
| `DEPENDENCY_ID_MISMATCH` | `C_STATIC` | Trusted Stable Package ID differs from the Contract value |
| `DEPENDENCY_DUPLICATE` | `C_STATIC` | Multiple snapshots or ambiguous trusted bindings match the same Alias/identity |
| `UNUSED_DEPENDENCY` | `C_WARNING` | Contract declares an Alias not referenced by the selected Skill |

Do not rename codes, merge phases, emit a warning as an error, or invent a digest-specific diagnostic.

Source location rules:
- Errors attributable to Contract source must retain the original YAML path, line and column from the WP2 AST.
- Errors attributable only to trusted BuildRequest metadata use an empty source path and zero line/column.
- Do not report a line number from a normalized/generated YAML serialization.

## 6. Explicitly out of scope

Do not implement these in WP3A:

- ToolCatalog full schema validation or callable indexing (WP3B).
- ToolCatalog metadata Owner authorization and per-export Owner attribution (WP3B).
- CapabilityContracts parsing, trusted deployment authorization, membership, Effect comparison or Scope intersection (WP3C).
- Procedure callable linking / DependencyInspection inventory integration beyond the fail-closed boundary stated above (WP3D).
- JSON Schema Draft 2020-12 / CA-SL G.4 / F.5 engine (WP4).
- CEL parser/type checking (WP4).
- Procedure graph / route / Goal proof (WP5).
- SourcePath availability, dominance, assignability and identity-input provenance (WP6).
- Evidence Descriptor format or Evidence linking (WP7).
- Mutation/Reconcile/Retry (WP8).
- Python CST / Script ABI (WP9).
- Artifact/JCS (WP10).
- Chromium Worker / Studio integration (WP11/WP12).
- Real 5GC / MAS qualification (WP13).

## 7. Allowed and forbidden paths

### Allowed paths

- `internal/compiler/dependency_identity.go` (new helper package file);
- `internal/compiler/api_v1alpha4.go` (only to integrate WP3A validation with the frozen API);
- `internal/compiler/v1alpha4.go` (only to replace the dependency identity placeholder with WP3A results);
- `internal/compiler/dependency_snapshot_digest.go` (read-only preferred; modify only if a WP1 defect is proven and stop for maintainer approval before changing it);
- `internal/compiler/dependency_identity_test.go` (new);
- `internal/compiler/dependency_identity_integration_test.go` (new, if needed);
- `tests/conformance/v1alpha4/fixtures/dependency-identity/**` (new fixture files);
- `tests/conformance/v1alpha4/wp3a-fixture-plan.json` (update status/actual fixture paths only; do not change expected semantics);
- `docs/workstreams/wp3a-dependency-identity-v1alpha4.md` (update implementation status and run record only).

### Forbidden paths

- `go.mod`, `go.sum`, all dependency versions and all OSS ADR decisions;
- `contracts/v1alpha4/**` and all frozen machine contract schemas;
- `docs/design/**` frozen contracts;
- CA-SL specification or any other repository;
- WP2 parser / AST implementation, except to report a blocking defect;
- Host, Runtime, Skill Studio, ca-agent-demo, ca-skills and real 5GC;
- unrelated v0/v0.1 implementation or historical files.

If a change outside the allowed list appears necessary, stop and request maintainer approval.

## 8. Required fixtures and assertions

Implement every fixture in `tests/conformance/v1alpha4/wp3a-fixture-plan.json`. The JSON file is the task's exact checklist.

For every negative fixture:
- assert the exact diagnostic Code and Phase;
- assert no unrelated earlier diagnostic masks the target;
- assert source path/line/column where the source itself is responsible;
- assert `Build.Success=false`, `OutputFiles=[]`, and `DependencyClosure=[]`;
- assert `Check` and `Build` agree on the semantic diagnostics.

For the positive identity fixture:
- assert no WP3A dependency-identity diagnostic;
- assert the resolved identity tuple is stable under input-file, snapshot and binding order permutations;
- assert later unimplemented stages still fail closed; do not assert overall Build success.

For E16:
- assert the unreferenced snapshot does not appear in the selected closure and cannot satisfy an Alias or provide an export;
- the extra snapshot itself must not create a new error merely because it is outside the selected Skill's declared closure, provided request-level integrity constraints are valid.

## 9. Implementation constraints

- Use the existing `yamlast.Node` tree and source positions. Do not add a YAML parser or parse the same author YAML again.
- Reuse WP1's digest implementation. No second digest implementation.
- Do not add external dependencies.
- Do not access the filesystem or network from Compiler Core.
- Do not use map iteration order as observable output order.
- Do not silently coerce or normalize package identity strings.
- Do not treat `authorizedOwners` or `exportOwners` as proof of export validity in this task; WP3B consumes them.
- Do not invent physical Owner, Evidence Descriptor or CapabilityContracts formats.
- Keep the public BuildResult failure atomic and deterministic.

## 10. Commands and acceptance

Use Go 1.27.4 exactly (CI toolchain). `go.mod` may declare the lower compatibility floor; do not change it.

Run from repository root:

~~~sh
go version
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
mkdir -p dist
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
git diff --check
~~~

Acceptance:
1. Baseline passes before edits, or a baseline failure is reported without unrelated repair.
2. Every WP3A fixture passes with exact expected diagnostics.
3. All relevant H identities in §5 have executable tests.
4. `Check` and `Build` agree on diagnostics; Build remains failure-atomic.
5. No unsupported later-stage behavior is accepted.
6. Native and Go/WASM builds pass.
7. Deterministic ordering is tested with reordered inputs.
8. Diff contains only allowed paths.
9. CI is green on the assigned branch / Draft PR.
10. The run record includes exact model ID/revision, Harness version/commit, Go version, commands, results, changed paths, fixture IDs and CI URL.

Chromium Worker is **not applicable to WP3A** because this task does not change browser Worker behavior or the WASM wire contract. Do not invent a browser command. WP11 owns the actual Chromium Worker parity harness and gate.

## 11. Stop conditions

Stop and report rather than guess if:
- the pinned CA-SL spec and Compiler-owned contract disagree;
- the BuildRequest schema does not match the Go API fields needed for the task;
- digest mismatch semantics would require a diagnostic not present in Appendix H;
- duplicate identity precedence is unclear;
- an E fixture cannot be reduced without changing its normative expected result;
- the code would need ToolCatalog, CapabilityContracts, Evidence or Runtime semantics to pass;
- a test appears to require relaxing or deleting a normative case.

A stop report must include the exact file/section, observed conflict, reproduction and the smallest maintainer decision required.

## 12. Required deliverables

- WP3A implementation on the assigned branch;
- unit and integration tests for all fixture-plan entries;
- actual fixture files with stable names;
- updated fixture-plan statuses (NOT_IMPLEMENTED -> PASS only with test evidence; use BLOCKED where appropriate);
- exact Go/WASM command output and CI link;
- a concise PR description listing scope, diagnostics, tests and explicit non-claims;
- a concise machine-readable or structured run record in the Draft PR description, including exact model ID/revision, Harness version/commit, Go version, commands, results, changed paths, fixture IDs and CI URL.

Do not merge the PR.
