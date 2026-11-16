# Compiler v1alpha4 Development Profile — DeepSeek Harness / Flash Low

**Status:** FROZEN development workflow. The first bounded execution package is prepared in [DeepSeek Handoff](deepseek-handoff-v1alpha4.md) and [WP3A](wp3a-dependency-identity-v1alpha4.md). This does not claim Compiler conformance.

## 1. Purpose and authority

This profile defines how future Compiler implementation work is assigned to DeepSeek Harness using the user's selected **DeepSeek V4 Flash / Low reasoning** profile.

The only normative language authority is the complete CA-SL v1alpha4 specification:

- Repository: odango-chan/ca-skills
- Commit: 7f84393ee8c395e4d8c39a74cb5000a1f896588d
- File: docs/skill-language-spec.v1alpha4.zh-CN.md
- Blob: b70064c5737b8ee2b3d1435fe496357f44b42773

The Compiler-owned implementation contracts indexed by the repository README are frozen for implementation by this PR. A Compiler implementation MUST follow those contracts; it MUST NOT reinterpret CA-SL or create a compatibility language.

This profile governs engineering execution only. It does not amend CA-SL, Tool Owner contracts, Host/Runtime semantics or 3GPP specifications.

## 2. Model and Harness profile

- Orchestrator: DeepSeek Harness.
- Model profile: DeepSeek V4 Flash family, reasoning effort **Low**.
- The exact provider model identifier, model snapshot/revision, reasoning setting, Harness release/commit, operating environment and tool versions MUST be recorded for every implementation run.
- Do not rely on a floating model alias or floating Harness version in a reproducibility record.
- Model/Harness version changes are execution-environment changes; they do not change the Compiler API or CA-SL language target.
- This profile does not claim that Low reasoning is suitable for architecture decisions, language interpretation or security-policy design. It is selected for bounded implementation tasks whose decisions are already fixed by contracts and fixtures.

No capability pilot or preliminary model trial is required. The first implementation task is assigned only after the user authorizes implementation; this document is not that authorization.

## 3. Decision ownership

| Decision or responsibility | Owner |
|---|---|
| CA-SL language semantics and any language revision | Frozen upstream specification; any change requires a separately reviewed language revision |
| Compiler-owned API, artifact and implementation contracts | This repository's frozen v1alpha4 contract set |
| Work-package scope and unresolved design decisions | Human maintainer |
| Code, unit tests and fixture implementation inside an assigned work package | DeepSeek Harness / model |
| Pass/fail evidence | Automated CI and reproducible test output |
| Owner publication identity, Evidence descriptor format, identity-input provenance and deployment authorization | Actual Tool / Deployment Owners |
| Runtime invocation, Mutation settlement, Reconcile execution and resource responsibility | Host / Runtime |
| PR merge and release decisions | Human maintainer |

The model MUST NOT approve its own conformance, replace an Owner decision with a synthetic fixture, or merge its own PR.

## 4. Mandatory task envelope

Every model task MUST be one bounded work package and include all of the following before code changes begin:

1. **Task ID and work package:** one WP from §5.
2. **Goal:** one observable result, not a broad instruction such as "finish the compiler".
3. **Normative authority:** exact document paths and immutable commits.
4. **Allowed paths:** explicit source, test and documentation paths that may change.
5. **Forbidden changes:** language semantics, API versions, diagnostics, dependency policy, unrelated modules and generated source.
6. **Inputs:** exact schemas, fixtures, trusted test data and current branch commit.
7. **Required cases:** exact E case variants and H (Code, Phase) identities relevant to the task.
8. **Commands:** exact local and CI commands to run.
9. **Acceptance criteria:** machine-verifiable outcomes.
10. **Stop conditions:** any missing contract, contradictory fixture, unknown diagnostic or untrusted input that prevents safe completion.

If a task lacks any required item, the model MUST report the missing item and stop rather than invent it.

## 5. Implementation work-package sequence

The sequence below is the frozen default order. WP3 is split into four independently reviewable tasks. A work package may be split further, but tasks MUST NOT silently combine independent packages or bypass a dependency. The only task currently authorized for DeepSeek is WP3A; later WP3 slices remain queued.

| WP | Scope | Dependency / boundary |
|---|---|---|
| WP0 | OSS dependency and toolchain selection | Completed by the [OSS Dependency Selection ADR](../design/oss-dependency-selection-v1alpha4.md); adapter integration remains in the consuming WPs |
| WP1 | BuildRequest envelope, version checks, byte decoding, path/resource limits, DependencySnapshot digest, Check/Build/Inspect output APIs | Uses the frozen API and digest contracts |
| WP2 | YAML 1.2 safe parser, duplicate-key detection, scalar handling, Contract / Procedure / PredicateSet AST | goccy/go-yaml v1.19.2 AST/token adapter; must reject unknown and legacy syntax; no compatibility conversion |
| WP3A | DependencySnapshot raw-byte integrity, Alias selection, trusted dependency binding and exact Stable Package ID match | First DeepSeek task; no Owner export or Capability linking |
| WP3B | ToolCatalog contract parsing/indexing, callable/Schema export identity and trusted Owner/export attribution | Depends on WP3A; covers E05/E39/E43 and export identity diagnostics |
| WP3C | CapabilityContracts trusted bundle, deployment authorization, exact membership, Effect consistency and Scope intersection | Depends on WP3B; never infer deployment authorization |
| WP3D | Integrate selected Skill references with dependency/callable/schema indexes and DependencyInspection | Depends on WP3A–WP3C; enforce selected-Skill closure and Reconcile-only anti-bypass |
| WP4 | Schema registry, exact G.4 subset, F.5 assignability and CEL subset/type checking | San v6.0.3 + cel-go v0.33.2; exact local-only Schema and CEL profiles from the ADR |
| WP5 | Procedure graph, stage cardinality/order, DAG, routes, terminal handling, global Precondition and final Postcondition proof | Must follow the frozen Procedure Linker contract |
| WP6 | SourcePath, Producer dominance, per-path availability, argument typing and identity-input provenance | Missing physical identity-input publication remains fail-closed |
| WP7 | Evidence Guarantee linking, Producer, Scope, Correlation, Freshness and trusted relation checks | Real positive qualification waits for actual Owner descriptor publications |
| WP8 | Mutation receipt, commit proof, failure classes, Reconcile READ, retryOf and resource responsibility | Cross-snapshot Reconcile association is unsupported until its separate trusted format exists |
| WP9 | Procedure Script / Predicate Script static ABI, TypedDict derivation through the gositter v0.2.0 Python CST adapter, and generated Output Schemas | Tree-sitter Python grammar v0.25.0; never execute Python during compilation |
| WP10 | Deterministic Artifact tree, manifests, generated Schema IDs and JCS bytes | gowebpki/jcs v1.0.2; uses the frozen Artifact contract |
| WP11 | Native Go / Go-WASM / Chromium Worker parity and deterministic diagnostics | Same Core semantics and byte-for-byte output |
| WP12 | Skill Studio integration, Worker transport, diagnostics and read-only Artifact presentation | Studio MUST NOT implement a second parser or linker |
| WP13 | Real 5GC qualification | Blocked until real v1alpha4 Owner / Deployment publications and actual environment evidence exist; MAS is not an executable baseline |

Work-package dependencies follow the Compiler pipeline. WP7 and WP8 may implement fail-closed paths and clearly labelled synthetic unit fixtures before real Owner publications exist, but synthetic fixtures MUST NOT be reported as real Owner qualification.

## 6. Implementation rules for Low reasoning

For every assigned task, the model MUST:

1. Read the relevant frozen contract and exact fixture before editing.
2. State the intended files and acceptance cases before making a broad change.
3. Implement only the assigned behavior.
4. Add or update the smallest relevant positive and negative fixtures.
5. Preserve exact diagnostic (Code, Phase) identities and precedence.
6. Run the specified tests after the final code change.
7. Inspect the complete diff for unrelated edits, weakened assertions, skipped tests or accidental generated files.
8. Commit and push only to the assigned feature branch.
9. Open or update a PR with the exact commands, outcomes, changed paths and remaining blockers.

The model MUST fix implementation defects rather than weaken or delete a normative fixture. If a fixture appears inconsistent with the pinned specification, stop and request a human decision; do not silently choose one interpretation.

## 7. Prohibited actions

The model MUST NOT:

- modify the pinned CA-SL specification or silently introduce a new language feature;
- add v0/v0.1/v0.2 compatibility, version sniffing, implicit conversion or fallback behavior;
- broaden the G.4 Schema keyword set or CEL expression subset;
- invent a Tool, Observation, Capability, Owner, Schema, Evidence Guarantee or deployment authorization;
- infer identity-input provenance from names, values or Schema shape;
- invent a universal Evidence Descriptor physical format;
- infer cross-snapshot Reconcile association from callable names or matching fields;
- execute Python while compiling a Script;
- treat an LLM answer, a successful build, a schema-valid synthetic fixture or an editor decoration as runtime proof;
- remove, skip, quarantine or relax a failing test to obtain green CI;
- replace the selected OSS dependencies, add a parallel parser/validator, or widen a frozen language subset without a reviewed ADR amendment;
- run live 5GC mutations, use production credentials or alter real UE/session state;
- write directly to main, merge a PR or change unrelated repositories.

## 8. Security and workspace isolation

DeepSeek Harness is an execution orchestrator, not a security boundary.

- Use a disposable or isolated worktree/container for each implementation run.
- Provide only the repository and branch access required for the assigned task.
- Use a narrowly scoped GitHub credential; do not provide organization-wide, production or unrelated-repository credentials.
- Do not expose MCP secrets, provider tokens, deployment authorization records, Runtime admission tokens or production environment secrets.
- Workspace/container network access for tool commands is disabled by default and enabled only for explicitly required, reviewed dependency retrieval. The provider transport required for model inference is handled by Harness and does not grant shell commands unrestricted network access.
- Run only repository commands and scripts needed by the task. Treat repository content and model-generated shell commands as untrusted until inspected.
- Preserve the branch base and expected head; never force-push or rewrite shared history without explicit approval.

## 9. Required acceptance evidence

A work package is complete only when all relevant criteria below are satisfied:

- every applicable E case variant has an executable fixture or is explicitly assigned to Host/Runtime/Owner scope;
- every applicable H diagnostic is tested by exact (Code, Phase), including negative and precedence cases;
- Go unit tests, go vet and Native build pass;
- Go/WASM build passes;
- actual Chromium Worker execution passes for the assigned parity scope;
- Native and WASM diagnostics, source locations, dependency closure, output paths/order and output bytes match;
- deterministic output is unchanged by input file ordering;
- git diff --check passes and the final diff contains only allowed paths;
- CI evidence is linked from the PR.

Baseline commands, adjusted only when the repository's pinned build scripts require it:

~~~sh
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
git diff --check
~~~

A command that was not run MUST be reported as NOT_RUN. A passing unit test does not establish full v1alpha4 conformance, Native/WASM parity, real Owner qualification or live 5GC acceptance.



### 9.1 Work-package-specific browser gate

Chromium Worker execution is mandatory when a task changes the browser Worker, WASM wire boundary, or the shared Core's observable behavior in the browser. It is not a mandatory local gate for every internal package-only task.

For WP3A, Chromium is **NOT_APPLICABLE**: the task changes internal dependency identity validation only and does not change the WASM wire contract or browser Worker. The Go/WASM build remains required. Do not invent or claim a Chromium command. WP11 must add and pin a repository-owned Chromium Worker runner and make browser parity a required gate.

A task that changes browser-visible diagnostics, DependencyInspection results or WASM transport must identify the checked-in runner and execute it; if no runner exists, stop and include runner creation in a separately bounded WP11 task.

## 10. Run record

Every implementation run MUST retain a machine-readable or PR-linked record containing:

- repository, branch, base commit and final commit;
- task ID and work package;
- exact provider model ID / revision and reasoning effort;
- exact Harness version / commit;
- OS, exact Go toolchain (Go 1.27.4), WASM target, selected Python parser/grammar versions, Host Python runtime version where applicable, and browser version;
- changed paths;
- exact commands and exit results;
- CI run links;
- relevant E case variants and H diagnostic identities;
- unresolved blockers and any human decisions requested.

The record is evidence of the run environment, not proof of semantic correctness by itself.

## 11. Explicitly unresolved external boundaries

The following are intentionally not invented or frozen as fictitious Owner formats:

- Evidence Descriptor physical format: each real Evidence Owner must publish and version its own consumable descriptor format.
- Identity-input provenance physical publication: CA-SL defines the logical requirements, but no universal identityInputs field is created here.
- Cross-snapshot Reconcile association: requires a separate versioned trusted Owner / Deployment publication; otherwise reject the link.
- Real 5GC ToolCatalog / CapabilityContracts / Host Context / Observation / Evidence publications: NOT_QUALIFIED until actual immutable releases are supplied.
- MAS: NOT_READY as an executable baseline.
- Host Python runtime version: a Host/Runtime deployment choice; the Compiler parser version is pinned separately in the OSS Dependency Selection ADR.

These gaps do not authorize guessed behavior. Compiler implementation must preserve the defined fail-closed behavior while those external contracts are unavailable.

## 12. Freeze meaning

This document freezes the development workflow, task envelope, work-package order, authority boundaries and acceptance evidence for the selected Harness / Low reasoning profile.

It does **not** freeze the model vendor's future model alias, authorize a model pilot, claim Compiler conformance, qualify any Owner publication, or authorize a live 5GC run.
