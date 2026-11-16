# Compiler v1alpha4 — Implementation & Acceptance Status

**Status:** IN PROGRESS — NOT_QUALIFIED. This is a truthful live status record, not a conformance claim.
**Normative authority:** CA-SL v1alpha4, `ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`.
**Acceptance method:** local reproducible commands only (maintainer instruction); GitHub Actions is not used as an acceptance dependency.

## 1. Stacked implementation PRs

| PR | Scope | State |
|---|---|---|
| #13 | WP1 BuildRequest / digest | Open |
| #14 | WP2 normalized YAML parser / AST | Open |
| #15 | WP3A dependency snapshot identity & trust binding | Draft |
| #16 | WP3B ToolCatalog export identity & Owner attribution | Draft |
| #17 | WP3C trusted CapabilityContracts linking | Draft |
| #18 | WP3D integrated linking & DependencyInspection | Draft |
| #19 | WP4A/WP4B G.4 allowlist, registry/closures, San adapter | Draft |
| #20 | WP4C F.5 conservative assignability engine | Draft |
| #21 | WP4D CEL subset & Predicate signatures | Draft |
| #22 | WP5 Procedure reachability, dominance, stage routes | Draft |
| #23 | WP6 SourcePath grammar, typing, dominance, availability, F.5 argument typing | Draft |

## 2. Work-package status

| WP | Status | Evidence / notes |
|---|---|---|
| WP0 OSS dependency selection | **DONE (ADR frozen)** | goccy/go-yaml v1.19.2, San v6.0.3, cel-go v0.33.2 integrated for their slices |
| WP1 BuildRequest / digest / APIs | **PASS (PR #13)** | strict decoding, limits, digest golden vectors, Native/WASM entrypoints |
| WP2 normalized YAML AST | **PASS (PR #14)** | single goccy AST adapter, duplicate keys, source positions |
| WP3A dependency identity | **PASS slice** | WP3A fixtures 10/10; aliases, trust binding, exact Stable Package ID, closure |
| WP3B ToolCatalog / Owner | **PASS slice** | WP3B fixtures 9/9; callable/Schema export identity, E05/E39/E43/E91 |
| WP3C CapabilityContracts | **PASS slice** | WP3C fixtures 11/11; membership, Effect, Grant, Scope-ref presence; empty-intersection not claimed (no frozen policy format) |
| WP3D integrated linking | **PASS slice** | WP3D fixtures 7/7; snapshot-scoped DependencyInspection, Reconcile anti-bypass |
| WP4A G.4 allowlist | **PASS slice** | 9/9 fixtures; complete keyword allowlist, Request root shape |
| WP4B registry / closure / San | **PASS slice** | 6/6 fixtures; exact `$id` closure, deny-by-default loader, Context root rules |
| WP4C F.5 assignability | **PASS engine** | 6/6 unit fixtures; call sites in WP6 |
| WP4D CEL subset | **PASS slice** | 5/5 fixtures; subset whitelist + Predicate signatures; guard proofs deferred WP6 |
| WP5 Procedure graph | **PARTIAL** | reachability, global-gate dominance, stage-route obligations done (7/7); exact auto-advance dominance, Owner-outcome route eligibility, final-Postcondition precision remain |
| WP6 SourcePath / data flow | **PARTIAL** | grammar/arity, context/request typing, producer dominance/availability, Tool/Predicate F.5 typing (22/22); identity-input provenance remains fail-closed (no frozen trusted publication format) |
| WP7 Evidence | **PARTIAL** | requirement→declared-parameter binding implemented (`PREDICATE_EVIDENCE_REQUIRED`, 2/2); Descriptor/Producer/Scope/Correlation/Freshness/trusted relation remain NOT_IMPLEMENTED and fail-closed (no frozen descriptor format) |
| WP8 Mutation / Reconcile | **PARTIAL boundary** | dedicated Reconcile READ anti-bypass + full Reconcile contract validation / retryOf remain |
| WP9 Python Script ABI | **PARTIAL** | LocalScriptPath grammar/presence and script-evaluator shape implemented (`SCRIPT_CONTRACT_INVALID`, 9 executed); Python CST/TypedDict/ABI and generated Output Schemas remain NOT_IMPLEMENTED and fail-closed |
| WP10 Artifact | **NOT_IMPLEMENTED** | deterministic tree / manifest / JCS not implemented; Build always fail-closed |
| WP11 Native/WASM/Chromium parity | **PARTIAL** | Native + Go/WASM builds PASS locally; real Chromium Worker gate NOT_IMPLEMENTED |
| WP12 Skill Studio integration | **NOT_IMPLEMENTED** | worker transport / UI consumption absent |
| WP13 Real 5GC qualification | **BLOCKED (external)** | no qualified real v1alpha4 Owner/Deployment publications; MAS not runnable; NOT_QUALIFIED |

## 3. Conformance coverage (honest)

- Executed fixtures: WP3A 10, WP3B 9, WP3C 11, WP3D 7, WP4A 9, WP4B 6, WP4C 6, WP4D 5, WP5 7, WP6 22 (total 119 executed cases).
- Spec E-cases with executable coverage: E03, E04, E05, E06, E07, E16, E24/E44/E51 (partial), E25, E39, E43, E91, E110 (schema subset); others remain NOT_IMPLEMENTED/NOT_RUN.
- No claim that E01–E120 are covered; `tests/conformance/v1alpha4/*-fixture-plan.json` records only the executed slices.

## 4. Diagnostics with executable evidence

`DEPENDENCY_REQUIREMENT_INVALID`, `DEPENDENCY_ALIAS_MISSING`, `DEPENDENCY_SNAPSHOT_MISSING`, `DEPENDENCY_IDENTITY_UNVERIFIED`, `DEPENDENCY_ID_MISMATCH`, `DEPENDENCY_DUPLICATE`, `UNUSED_DEPENDENCY`, `EXPORT_ID_MISSING`, `EXPORT_ID_AMBIGUOUS`, `SCHEMA_MISSING`, `SCHEMA_INVALID`, `DEPENDENCY_OWNER_MISMATCH`, `DEPENDENCY_EXPORT_OWNER_UNVERIFIED`, `CAPABILITY_INPUT_UNTRUSTED`, `CAPABILITY_DEFINITION_CONFLICT`, `CAPABILITY_MEMBERSHIP_DUPLICATE`, `CAPABILITY_MAPPING_MISSING`, `EFFECT_MISMATCH`, `CAPABILITY_NOT_GRANTED`, `CAPABILITY_SCOPE_UNRESOLVED`, `IDENTIFIER_INVALID`, `CONTEXT_REQUEST_COLLISION`, `CONTEXT_SCHEMA_UNDECLARED`, `REQUEST_SCHEMA_MISSING`, `UNREACHABLE_NODE`, `ENTRY_PRECONDITION_BYPASSED`, `ROUTE_NORMAL_OUTCOME_NOT_ALLOWED`, `ROUTE_OUTCOME_MISSING`, `ROUTE_OUTCOME_UNKNOWN`, `SOURCE_PATH_INVALID`, `SOURCE_NOT_AVAILABLE`, `INPUT_REQUIRED_MISSING`, `INPUT_TYPE_MISMATCH`, `INPUT_NULLABILITY_MISMATCH`, `PREDICATE_TYPE_MISMATCH`, `PREDICATE_ARITY_MISMATCH`, `PREDICATE_GROUP_INVALID`, `CEL_INVALID`, `RECONCILE_CONTRACT_INVALID`.

Not yet emitted/tested: `PREDICATE_EVIDENCE_*`, `EVIDENCE_GUARANTEE_LINK_INVALID` (WP7), `RETRY_CONTRACT_INVALID`, `RECONCILE_CONTRACT_INVALID` full set, `SCRIPT_CONTRACT_INVALID`, `CAPABILITY_SCOPE_EMPTY_INTERSECTION`, Artifact-related identities, Runtime identities.

## 5. Local acceptance commands (Go 1.27.4)

All currently PASS (`0` exit): `GOWORK=off go test ./... -count=1`, `GOWORK=off go vet ./...`, `CGO_ENABLED=0 GOWORK=off go build ./...`, `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm`, `git diff --check`.

## 6. Unresolved / blockers

- Build remains intentionally fail-closed until WP7–WP12 are implemented; no successful Build/Artifact is claimed.
- Evidence Descriptor format, identity-input provenance publication and cross-snapshot Reconcile association are not frozen/available; no format is invented.
- Real Owner/Deployment v1alpha4 publications are NOT_QUALIFIED; WP13 BLOCKED (external).
- Chromium Worker parity and Studio integration are NOT_IMPLEMENTED.

Last updated on the WP6 branch; this document must be regenerated as each WP completes.


## 7. Recent verified increments (stack on WP6 branch, local gates PASS)

- Contract: limits.maxActions (ACTION_LIMIT_INVALID / UNKNOWN_FIELD), GoalId grammar (IDENTIFIER_INVALID), contextSchema QualifiedSchemaId (QUALIFIED_REFERENCE_INVALID).
- Predicate: parameter name Identifier grammar (IDENTIFIER_INVALID); evaluator script shape/presence (SCRIPT_CONTRACT_INVALID); Alias::ExportId tool references (QUALIFIED_REFERENCE_INVALID).
- Procedure Script LocalScriptPath + file presence (SCRIPT_CONTRACT_INVALID); retryOf reference validity and retry-chain rejection (RETRY_CONTRACT_INVALID).
- Mutation contract identity requirements across the selected snapshot closure (RECONCILE_CONTRACT_INVALID).
- Determinism/diagnostic-contract gates; single-YAML-parser guard.
- Machine-readable executed coverage (tests/conformance/v1alpha4/executed-coverage.json) and remaining-work inventory (docs/workstreams/REMAINING-WORK-v1alpha4.json).

All increments verified locally with Go 1.27.4 (test/vet/native/WASM/diff --check = 0). Build remains fail-closed; no v1alpha4 conformance or Owner qualification is claimed.


## 8. WP9/WP10 progress after maintainer resume (branch feat/compiler-v1alpha4-wp9-script-abi-20261116, Draft PR #24)

Stack base: WP6 branch at 30f71ca. Commits: 4ebfd56 (WP9 slice 1), 351028d (WP9 slice 2), cfda6d0 (WP9 slice 3), 3f1fd9c (WP10 slice 1), plus status commit.

- WP9 PARTIAL: gositter v0.2.0 Python CST adapter (internal/pythoncst, Python grammar package only, no Python execution, source/node/stall budgets); Script ABI derivation for main(input: InputTypedDict) -> OutputTypedDict with strict nonempty TypedDict output, F.6 Literal/Annotated/NotRequired subset, rejection of untyped input, missing return annotation, extra parameters, duplicates, fields with defaults, 	otal=, broad/unknown types; SCRIPT_CONTRACT_INVALID wiring; Procedure Script call-site required/unknown keys (INPUT_REQUIRED_MISSING / INPUT_TYPE_MISMATCH, E67); F.5 typing of Script argument SourcePaths against the derived input Schema. NOT DONE: PredicateInput projection from parameters + Evidence, Python body key-access (literal existence, NotRequired guards, dynamic-key rejection), PredicateEvaluationResult semantic checks, generated output Schema emission into Build.
- WP10 PARTIAL: internal/artifact implements the frozen Artifact Contract tree/manifest with JCS (gowebpki/jcs v1.0.2), exact generated Script Output Schema path and  derivation (contract example golden), duplicate/collision rejection, byte-deterministic output, frozen manifest schema validation via San. NOT DONE: assembly is not yet wired into a successful Build (Build remains intentionally fail-closed), and no BuildResult-level artifact determinism test exists yet.
- Resolved maintainer decision implemented: inline object/array Expression argument values are SOURCE_PATH_INVALID / C_STATIC; legal empty outer rguments: {} / rguments: [] are not rejected.
- Remaining scoped BLOCKED_NORMATIVE_MAPPING fixtures (Owner observation illegal ownerOutcomes; empty/duplicate evidence.requires; unsupported evaluator.type) stay blocked per ca-skills #59; no diagnostic was invented.

Local evidence (Go 1.27.4): go test ./...=0, go vet=0, Native build=0, Go/WASM build=0, git diff --check=0.

Overall Compiler: INCOMPLETE, NOT_QUALIFIED, Build fail-closed. Next slices: WP9 PredicateInput/key-access, WP10 artifact wiring + determinism through BuildResult, then WP8/WP7 independent checks, WP11 parity, WP12 Studio.

- WP8 slice 3 (retry family): retryOf target and retry Node must link to MUTATION callables; at most one new dispatch per original Mutation family; a retry Node must not use the dedicated Reconcile READ callable. All map to RETRY_CONTRACT_INVALID. Unit-tested core plus pipeline wiring; full retry failure-class/path-coverage contract remains later WP8.

- WP11 (partial, locally PASS): real Chromium Worker parity harness scripts/chromium-parity.mjs + cmd/parity-replay + web/parity.html/web/parity-project. It builds CGo-free Go/WASM, launches real headless Chromium (observed Chrome/158.0.8127.37), runs Check/Build/Inspect through the same Worker transport as Studio, and compares against the Native Go reference: check diagnostics, build success/diagnostics/dependencyClosure/output paths and inspect are identical (PASS). Fixed the Worker transport to wait for/call skillcInspectDependencies instead of the removed skillcInspectLibraries. Not covered yet: successful-Build Artifact byte parity (Build still fail-closed) and Studio UI acceptance.

- WP12 (partial, locally PASS): read-only Skill Studio integration surface web/studio.html consumes the unified Compiler Worker result (check/build/inspect), renders diagnostics with code/phase/path/line/column/message, and presents Build output files with a read-only artifact view. Real headless Chromium acceptance extends the parity harness: studio.diagnostics/buildSuccess/artifactPaths match the Native reference, rows rendered, artifact view read-only, no parser reimplementation. Honest limit: fixture exercises the fail-closed path; full Studio product (editor, project management, ZIP export) is outside the Compiler repository.

- WP10 Build success path: the blanket residual fail-closed gate was replaced by a feature-scoped one. Skills whose semantics are implemented now produce a real successful BuildResult: author files preserved byte-for-byte, generated Script Output Schemas, JCS artifact manifest, sorted dependency closure, atomic failure otherwise. Build remains fail-closed (explicit fail-closed marker) for semantics still not implemented in this branch: WP7 Evidence guarantee linking, deep WP8 Mutation/Reconcile, full retry contract. New test TestV1Alpha4ValidSkillBuildsDeterministicArtifact proves success + byte-identical repeated Build + sorted closure on a fully validated fixture (trusted bindings + CapabilityContracts, Evidence removed). Chromium parity re-run still PASS.

- Successful-Build cross-runtime byte parity (WP10/WP11/WP12): added web/parity-success fixture (Skill + real snapshots + capability bundle + trusted bindings), cmd/parity-replay trusted-request mode (computes snapshot/capability digests, emits exact BuildRequest JSON), and a second Chromium phase. Verified in real headless Chromium: success.check/build.success/build.diagnostics/build.dependencyClosure/output paths identical, every generated artifact file byte-identical by SHA-256 (5 files, manifest + author sources), and Studio success-path artifact presentation identical/read-only. Local run PASS (15.1s) via CHROMIUM_PARITY=1 go test ./internal/parity.

- WP8 slice 4: trusted Mutation/Reconcile static semantics. For every MUTATION callable in the selected snapshot closure, the receipt/proof Schema identities must resolve in the trusted snapshot, commitProof.path and Reconcile proofPath must exist in their declared Schemas, the dedicated Reconcile callable must resolve to exactly one same-snapshot READ operation with the same Owner, and recoverableOutputSchema/Path must be paired and resolvable. Violations use frozen RECONCILE_CONTRACT_INVALID. Cross-snapshot Reconcile association remains NOT_IMPLEMENTED/fail-closed (no separately versioned trusted association contract). Fixtures covering positive, unresolved commit path, non-READ Reconcile, and missing Reconcile target.

- WP7 slice 2: Evidence guarantee references (evidence.requires entries and independence member guarantees) must satisfy the frozen Alias::GuaranteeId qualified-reference syntax; violations use QUALIFIED_REFERENCE_INVALID. Descriptor content, Producer/Scope/Correlation/Freshness and trusted relations remain NOT_IMPLEMENTED and fail-closed because no Evidence Descriptor physical format is frozen.
