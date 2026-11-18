# CA Skill Compiler — v1alpha5 single-version acceptance

**Implementation status:** `FULL_LANGUAGE_VALIDATION` for the Compiler-owned
static scope.
**Product version:** CA Skill Compiler **v2.0.0**.
**Language target:** `connectionagent.io/v1alpha5` — the only supported target.
**Local acceptance:** `LOCAL_ACCEPTANCE_PASS` (see §5 for toolchain and commit).
**GitHub CI:** `CI_UNAVAILABLE` — no CI PASS is claimed.
**Product qualification:** **NOT_QUALIFIED** (Owner / Deployment / Integration
Adapter / Host / Runtime / 5GC / MAS).

- CA-SL v1alpha5: **FROZEN / RELEASED**, `connectionagent.io/v1alpha5`,
  ca-skills PR #66, merge commit `0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5`.
- IngressCatalog: **FROZEN / RELEASED**, `connectionagent.io/ingresscatalog/v1alpha1`,
  ca-skills PR #62, merge commit `9412be9894ef36acf846c757a5870dc0c48b33ac`.
- Ingress diagnostics: **FROZEN / RELEASED**,
  `connectionagent.io/ingress-diagnostics/v1alpha1`, released with PR #62.
- CA-SL PR #61 is closed without merge; PR #66 is the normative authority.

## 1. Single-version architecture

| Concern | Result |
|---|---|
| Language targets | exactly one: `connectionagent.io/v1alpha5` |
| Public Compiler entries | `Check`, `Build`, `InspectDependencies`, `VerifyBuildResult` |
| BuildRequest / BuildResult / DependencyInspection API | one identity each, all `/v1alpha5` |
| v1alpha4 API / CLI / artifact entry | removed; v1alpha4 requests fail closed |
| Compiler pipeline | one admission + one inherited semantic pipeline + Wait/Ingress passes |
| Artifact path | one: v1alpha5 manifest, assembly and verification |
| Inherited semantics | preserved (package, Contract, PredicateSet, dependency identity, Owner/Tool/Capability linking, Schema closure, CEL, Python Script ABI, Procedure graph, SourcePath, Evidence Descriptor, Mutation/Reconcile, retry) |

The internal AST/linker/validator symbols are version-neutral; `v1alpha4`
remains only as the normative inheritance source, as an immutable trusted-input
artifact version (ToolCatalog / CapabilityContracts) and in historical release
records.

## 2. What the single pipeline implements

- strict BuildRequest admission: exact API version and language target,
  normalized source paths, file/size/count limits, dependency snapshot count,
  measured SHA-256 snapshot digests, trusted dependency/Owner bindings,
  CapabilityContracts and EvidenceTrustBindings bundle digests, deployment
  authorizations and trusted artifact/Schema binding structure;
- Skill package, Contract, PredicateSet, package Schema and Python Script ABI
  admission (Python parsed, never executed);
- dependency identity and exact trusted Owner attribution;
- ordinary Operation / Observation ToolCatalog linking and Capability
  membership;
- Schema registry, `$ref` closure, assignability, CEL subset and Predicate
  typing;
- Procedure graph reachability, global Precondition dominance, stage
  aggregation, route/terminal rules and the final-Postcondition Goal proof;
- Evidence Descriptor linking, Evidence independence, Mutation, Reconcile and
  retry-family static assurance;
- CA-SL v1alpha5 node-entry Wait: trusted IngressCatalog selection,
  trigger/correlation binding on every reachable entry path, `wait.output`
  SourcePath typing and event-accepted dominance, paired response boundary,
  typed response mapping and Schema closure;
- deterministic artifact assembly with canonical RFC 8785 JCS serialization and
  self-verification before any output is returned.

`Check` and `Build` run the same static pipeline; `Build` assembles an artifact
only when that pipeline returns no error diagnostic. A failed Build returns no
output files and an empty dependency closure.

## 3. Contract freeze

`contracts/v1alpha5/` is **FROZEN** for the v2.0.0 line:

- `build-request.schema.json`
- `build-result.schema.json`
- `dependency-inspection.schema.json`
- `artifact-manifest.schema.json`
- `dependency-snapshot-digest-vectors.json`

The artifact API version and the CA-SL language version are independent. The
manifest records source digests, dependency closure, Wait/Ingress link metadata,
output paths and SHA-256 digests; `validationScope.status =
FULL_LANGUAGE_VALIDATION` is a Compiler-owned static statement, and
`notValidated` retains Host/Runtime and real Owner/product qualification.
Changing a field meaning requires a new contract version.

## 4. EW / IN register

`docs/workstreams/wp7-conformance-v1alpha5.json` contains all **102** cases:

- 36 EW (`EW-001`…`EW-036`) and 66 IN (`IN-001`…`IN-066`), all unique;
- 55 classified `COMPILER_STATIC_IMPLEMENTED_TESTED`, each linked to executable
  Go tests;
- 46 classified `HOST_RUNTIME_INTEGRATION_OWNED` with the owning role recorded;
- 1 classified `SPECIFICATION_ONLY_NOT_EXECUTED`;
- 0 Compiler-static deferred.

All 48 distinct test names referenced by the register exist in the repository.
Host/Runtime obligations are not counted as Compiler PASS.

## 5. Local acceptance

Executed on Windows/amd64 on 2026-11-18 at commit `0e5fd60a96107c4bd136a6a5b544b78649bfd853` (PR #38 head):

| Tool | Version |
|---|---|
| Go | go1.26.6 windows/amd64 |
| Node.js | v24.18.0 |
| Chromium | 158.0.8127.37 (real headless browser) |

| Gate | Command | Result |
|---|---|---|
| Go tests | `GOWORK=off go test ./... -count=1` | PASS |
| Vet | `GOWORK=off go vet ./...` | PASS |
| Native build | `CGO_ENABLED=0 GOWORK=off go build ./...` | PASS |
| WASM build | `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS |
| CLI end-to-end | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS |
| Chromium parity | `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | PASS (real Chromium Worker) |
| Whitespace | `git diff --check` | PASS |

Additional verified behaviors:

- real CLI `version`, `check`, `inspect` and `build` on the checked-in trusted
  project and on explicit BuildRequest documents;
- successful and failed compilation, multi-Skill selection, ordinary
  Operations, Wait/Ingress, Evidence/Mutation/Reconcile fixtures;
- illegal input fails closed; failed Build emits no artifact;
- artifact determinism across repeated builds and native/WASM/Chromium byte
  parity;
- v1alpha4 BuildRequest API identity and language target are rejected by both
  `Check`/`Build` and the CLI (`REQUEST_INVALID` / `LANGUAGE_TARGET_UNSUPPORTED`);
- a v1alpha4 BuildResult cannot pass `VerifyBuildResult`.

GitHub Actions did not run the acceptance workflow because the account's
billing/spending limit blocked job execution. No CI PASS is claimed and no
failed step is treated as a source failure.

## 6. Status classification

| Status | Meaning here |
|---|---|
| IMPLEMENTED | single-version Compiler core, Wait/Ingress, Evidence/Mutation/Reconcile/retry and deterministic artifact pipeline are implemented |
| VERIFIED | full local gate set above PASS, including real Chromium Worker parity |
| FROZEN | CA-SL v1alpha5, IngressCatalog v1alpha1, and the Compiler-owned v1alpha5 contracts |
| RELEASE_READY | v2.0.0 release inputs prepared; awaiting human approval |
| RELEASED | only the historical v1.0.0 product; v2.0.0 is **not** released and no tag/Release is created |
| NOT_QUALIFIED | real Owner / Deployment / Host / Runtime / 5GC / MAS product qualification |

## 7. PR and release boundary

PR [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38) on
`feat/compiler-v1alpha5-integration-20261117` is the single integration
candidate. Child PRs #30–#37, #39 and #40 are stacked slices already represented
in that branch and must not be merged again.

Merging to `main`, creating a `v2.0.0` tag and publishing a Release each require
explicit human approval. The v1.0.0 tag, assets and release notes are immutable
and are not rewritten.

## 8. Limits

- Static proof only: the Compiler does not receive or isolate live events,
  schedule timeouts, execute Operations or Python, acquire dependencies,
  authorize deployment or qualify a product.
- Trusted fixtures are synthetic; they prove static algorithm behavior, not real
  Owner publication or deployment readiness.
- See [known limitations](known-limitations-v1alpha5.md) for the external
  responsibility boundary.
