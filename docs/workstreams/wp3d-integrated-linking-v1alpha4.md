> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP3D — Integrated Dependency Linking, DependencyInspection and Reconcile Anti-bypass

**Task ID:** `WP3D-INTEGRATED-LINK-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — implemented on `feat/compiler-v1alpha4-wp3d-integrated-linking-20261116` for Draft PR review. The frozen [Harness profile](deepseek-harness-v1alpha4.md) assigns WP3D to the human maintainer's queue; this document records the exact envelope used so the maintainer can ratify, amend or reject it before merge. It does not amend the frozen Harness profile, the Compiler contracts or CA-SL.  
**Depends on:** WP3C (`eccebc4`, PR #17 Draft)  
**Task branch:** `feat/compiler-v1alpha4-wp3d-integrated-linking-20261116`

## 1. Goal

Integrate the already-proven WP3A–WP3C dependency identity, export and capability results into one coherent, deterministic linking slice:

1. one selected-Skill link scope and dependency closure boundary;
2. a real read-only `DependencyInspection` projection for supplied snapshots;
3. same-snapshot Reconcile dedicated-READ anti-bypass;
4. unchanged fail-closed behavior for everything later.

WP3D does not add language semantics and does not make Build succeed.

## 2. Frozen authority

- CA-SL v1alpha4 `ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`: §§9.3, 24.1, 34.0–34.2, F.1, F.2, G.2/G.2.1.
- [Compiler Contract](../design/compiler-contract-v1alpha4.md) §§4–5, 10.
- [Compiler Output API Contract](../design/compiler-output-contract-v1alpha4.md) §§3, 5, 6 (DependencyInspection is snapshot-scoped and must omit unverified exports).
- [Owner Contract Linking](../design/owner-contracts-v1alpha4.md) §§2, 4.2 (Reconcile-only anti-bypass).
- Machine contract: `contracts/v1alpha4/dependency-inspection.schema.json`.

## 3. Required behavior

### 3.1 Selected Skill link scope and closure

- Keep the WP3A per-Skill dependency closure: only aliases actually referenced by that Skill enter it; multiple selected Skills never share or leak alias sets.
- WP3B/WP3C results remain per selected Skill; the merged link scope is the deterministic union of the selected Skills' verified identities.
- BuildResult.dependencyClosure stays empty on failure; the internal closure is still computed and tested for determinism and isolation.

### 3.2 DependencyInspection (read-only, snapshot-scoped)

`InspectDependencies(BuildRequest)`:

1. Runs the same request-envelope validation as Check/Build. An invalid envelope returns diagnostics with empty arrays.
2. Inspects **every supplied** `resolvedDependencies` entry, not a Skill closure; it does not parse Skill source, resolve author references or evaluate Capability Grants.
3. For each supplied snapshot:
   - the snapshot needs exactly one trusted `TrustedDependencyBinding` for its `dependencyId` (request envelope already guarantees digest/path integrity and at most one binding). Missing binding → `DEPENDENCY_IDENTITY_UNVERIFIED` and no exports from that snapshot;
   - parse ToolCatalog callables and Schema `$id` exports through the shared WP2 AST;
   - duplicate callable ID / Schema `$id` → `EXPORT_ID_AMBIGUOUS` and both records are omitted from the projection;
   - attribute an export Owner only through trusted attribution: catalog `metadata.owner` must be authorized; single-authorized-Owner snapshots attribute the sole Owner; multi-Owner snapshots require exactly one explicit `(exportKind, exportId) → owner` mapping. Unverifiable exports are omitted with `DEPENDENCY_OWNER_MISMATCH` or `DEPENDENCY_EXPORT_OWNER_UNVERIFIED`;
   - a malformed or unsupported document contributes no exports (later-stage structural validation remains fail-closed).
4. Emits no `CAPABILITY_*`, `EVIDENCE_*` or Skill-reference diagnostics. It never authorizes anything.
5. Callables are sorted by `(dependencyId, owner, kind, id, sourcePath)` and Schemas by `(dependencyId, owner, id, sourcePath)`, UTF-8 byte order, per the frozen output contract.
6. Returns only `DependencyInspection` metadata; no dependency bytes, digests, capabilities or Scope decisions.

### 3.3 Same-snapshot Reconcile dedicated-READ anti-bypass

- While indexing a verified snapshot, collect every Mutation callable's `mutation.reconcile.callable` ExportId. It names a dedicated Reconcile READ callable in the same snapshot.
- If the selected Skill's `operation.tool` directly references a dedicated Reconcile READ ExportId, emit `RECONCILE_CONTRACT_INVALID` at the Skill source reference and do not link that reference as an ordinary callable.
- The scan covers all Mutation contracts in the selected snapshot closure, not only the ones the Skill references.
- Cross-snapshot / cross-publication Reconcile association remains unsupported: no trusted association format is frozen, so it is not inferred. Full Reconcile contract validation (commit proof, failure classes, recoverable output) remains WP8.

## 4. Frozen diagnostics in this task

| Code | Phase | Trigger |
|---|---|---|
| `EXPORT_ID_AMBIGUOUS` | `C_STATIC` | Duplicate callable ID or Schema `$id` encountered by inspection |
| `DEPENDENCY_IDENTITY_UNVERIFIED` | `C_STATIC` | Supplied snapshot has no trusted dependency binding |
| `DEPENDENCY_OWNER_MISMATCH` | `C_STATIC` | Owner attribution unauthorized or inconsistent |
| `DEPENDENCY_EXPORT_OWNER_UNVERIFIED` | `C_STATIC` | Multi-Owner export attribution missing/repeated/ambiguous |
| `RECONCILE_CONTRACT_INVALID` | `C_STATIC` | Author directly calls a dedicated Reconcile READ callable |

No new diagnostic code or phase is introduced.

### Source location convention used by this envelope

- `RECONCILE_CONTRACT_INVALID` for a direct author call keeps the original WP2 Skill reference path/line/column.
- Snapshot-scoped inspection diagnostics use empty path and zero line/column (no Skill source exists for inventory inspection).
- Snapshot file paths are never reported as Skill source paths.

## 5. Explicitly out of scope

- Full Reconcile/Mutation contract validation, cross-snapshot Reconcile association, retryOf (WP8).
- Evidence descriptors/relations (WP7), Schema G.4/assignability/CEL (WP4), Procedure graph beyond the WP3A baseline (WP5), SourcePath (WP6), Script (WP9), Artifact (WP10), Chromium (WP11), Studio (WP12).
- Capability Grants/Scope evaluation inside DependencyInspection (must not happen).
- Full ToolCatalog/CapabilityContracts meta-schema validation with unknown-field rejection.

## 6. Allowed paths

- `internal/compiler/dependency_inspection.go` (new);
- `internal/compiler/dependency_inspection_test.go` (new);
- `internal/compiler/owner_linking.go` (Reconcile target metadata and anti-bypass only);
- `internal/compiler/api_v1alpha4.go` (InspectDependencies integration only);
- `internal/compiler/v1alpha4.go` (only if integration requires it);
- `tests/conformance/v1alpha4/fixtures/integrated-linking/**` (new fixture files);
- `tests/conformance/v1alpha4/wp3d-fixture-plan.json` (new inventory);
- `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only);
- this document.

Forbidden: CA-SL spec, frozen `docs/design/**`, machine contracts, `go.mod`/`go.sum`, dependency versions, other repositories, Host/Runtime/Studio, unrelated v0/v0.1 code.

## 7. Fixtures and acceptance

Fixture inventory: `tests/conformance/v1alpha4/wp3d-fixture-plan.json`.

Acceptance:
1. All WP3D fixture entries execute with exact (Code, Phase) and location rule.
2. DependencyInspection returns exact metadata ordering, omits unverified exports and never invents Owner/Capability authorization.
3. Reconcile anti-bypass is proven on a same-snapshot Mutation/Reconcile fixture.
4. Multi-Skill closure isolation and determinism remain proven.
5. Check and Build remain consistent; Build remains failure-atomic and fail-closed.
6. `GOWORK=off go test ./... -count=1`, `GOWORK=off go vet ./...`, native build, Go/WASM build and `git diff --check` pass with Go 1.27.4.
7. CI evidence is linked; external CI blockers are recorded honestly.

## 8. Stop conditions

Stop and report rather than guess if:
- DependencyInspection would need Skill-source or Capability/Scope semantics that the frozen output contract excludes;
- an export Owner cannot be attributed by the frozen rules and the frozen diagnostic set cannot express it;
- a Reconcile association would require a cross-snapshot trusted format that is not frozen;
- a required diagnostic identity is missing from Appendix H.
