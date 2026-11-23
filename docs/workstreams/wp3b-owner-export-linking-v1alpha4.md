> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP3B — ToolCatalog Export Identity and Trusted Owner Attribution

**Task ID:** `WP3B-OWNER-EXPORT-001`
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — implemented on `feat/compiler-v1alpha4-wp3b-owner-export-20261116` for Draft PR review. The frozen [Harness profile](deepseek-harness-v1alpha4.md) assigns WP3B to the human maintainer's queue; this document records the exact envelope used so the maintainer can ratify, amend or reject it before merge. It does not amend the frozen Harness profile, the Compiler contracts or CA-SL.
**Depends on:** WP3A (`50acc70`, PR #15 Draft)
**Task branch:** `feat/compiler-v1alpha4-wp3b-owner-export-20261116`

## 1. Goal

For each DependencySnapshot selected by the WP3A trusted identity chain, prove the export identity and trusted Owner attribution actually used by the selected Skill:

`used Alias -> verified snapshot -> ToolCatalog callable / Schema $id export -> unique trusted Owner`

WP3B is only identity/linking proof. It does not prove a Capability grant, dispatch authority, Schema type compatibility, Evidence quality or Runtime availability.

## 2. Frozen authority

- CA-SL v1alpha4 `ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`: §§24.1, 34.0–4.2, F.1, G.1, G.4, Appendix E (E05, E39, E43, E91).
- [Owner Contract Linking](../design/owner-contracts-v1alpha4.md) §§2, 4.1, 5.
- [BuildRequest v1alpha4](../design/build-request-v1alpha4.md) §§4–.
- Machine contracts: `contracts/v1alpha4/tool-catalog.schema.json`, `contracts/v1alpha4/build-request.schema.json`.
- Fixture baseline: `tests/conformance/v1alpha4/fixtures/dependency-identity/base` (Appendix E.1–E.4).

## 3. Required behavior

1. Parse every YAML file of every snapshot in the selected Skill's verified WP3A closure through the single WP2 `yamlast` AST. Do not add a second YAML parser and do not parse files of unreferenced snapshots.
2. Index ToolCatalog documents whose `apiVersion` is exactly `connectionagent.io/toolcatalog/v1alpha4` and `kind` is exactly `ToolCatalog`:
   - required identity envelope: nonempty `metadata.owner` (≤60 runes), nonempty `spec.contracts[]`;
   - each callable: `kind` ∈{operation, observation}; unique DottedId `id` (≤40 bytes); `effect` ∈{READ, MUTATION}; nonempty unique `ownerOutcomes` ⊆{SUCCEEDED, FAILED, NOT_COMMITTED} containing SUCCEEDED; `outputSchema.$ref` absolute URI; operation requires `inputSchema.$ref`; observation requires `queryContextSchema.$ref` and effect READ.
   - callable IDs share one namespace across all ToolCatalog files and both kinds in one snapshot. A duplicate ID is `EXPORT_ID_AMBIGUOUS` even when kind or Owner differs (E91).
3. Index JSON Schema documents in the same closure by exact absolute `$id` (requires `$schema: https://json-schema.org/draft/2020-12/schema`). Duplicate `$id` in one snapshot is `EXPORT_ID_AMBIGUOUS`.
4. Resolve only references collected from frozen v1alpha4 positions (the WP3A scanner): `operation.tool`/`observation.tool` →callable kind operation/observation; `Contract.contextSchema`, Predicate parameter QualifiedSchemaId and Skill-owned Schema `$ref` →schema export; `evidence.requires`/`independence` guarantee →evidenceGuarantee export.
   - no export for the exact reference identity →`EXPORT_ID_MISSING` (callable/guarantee) or `SCHEMA_MISSING` (Schema) at the Skill source reference node;
   - callable kind mismatch →`EXPORT_ID_MISSING` (the required export identity does not exist).
5. Trusted Owner attribution for every referenced export:
   - every ToolCatalog `metadata.owner` in the used closure must be in that snapshot's `TrustedDependencyBinding.authorizedOwners`, else `DEPENDENCY_OWNER_MISMATCH` (E05);
   - single-authorized-Owner snapshot: the sole authorized Owner is the trusted export Owner; an export that self-declares a different Owner is `DEPENDENCY_OWNER_MISMATCH`;
   - multi-Owner snapshot: every actually referenced export requires exactly one explicit `(exportKind, exportId) →owner` trusted mapping. Missing, repeated or ambiguous mapping is `DEPENDENCY_EXPORT_OWNER_UNVERIFIED` (E39). A unique owner outside `authorizedOwners`, or inconsistent with a self-declared Owner, is `DEPENDENCY_OWNER_MISMATCH` (E43);
   - evidenceGuarantee exports have no self-declared Owner field; only trusted attribution applies.
6. Result is retained as structured metadata for WP3C/WP3D: per used Alias, the indexed export records and attributed Owners. WP3B must not publish a `DependencyInspection` inventory (WP3D) and must not claim overall success.

## 4. Frozen diagnostics in this task

| Code | Phase | Trigger |
|---|---|---|
| `EXPORT_ID_AMBIGUOUS` | `C_STATIC` | Duplicate callable ID or Schema `$id` in one selected snapshot |
| `EXPORT_ID_MISSING` | `C_STATIC` | Referenced callable/guarantee export does not exist |
| `SCHEMA_MISSING` | `C_STATIC` | Referenced Schema `$id` does not exist |
| `DEPENDENCY_OWNER_MISMATCH` | `C_STATIC` | Owner not authorized / inconsistent with trusted attribution |
| `DEPENDENCY_EXPORT_OWNER_UNVERIFIED` | `C_STATIC` | Multi-Owner export attribution missing, repeated or ambiguous |

No new diagnostic code or phase is introduced.

### Source location convention used by this envelope

- Skill-author reference failures (`EXPORT_ID_MISSING`, `SCHEMA_MISSING`) use the original WP2 reference node path/line/column.
- Imported-contract and trusted-binding failures (`EXPORT_ID_AMBIGUOUS`, `DEPENDENCY_OWNER_MISMATCH`, `DEPENDENCY_EXPORT_OWNER_UNVERIFIED`) have no author-source position; they use an empty path and zero line/column in line with the frozen trusted-input rule.
- Snapshot file paths are never reported as if they were Skill source paths.

## 5. Explicitly out of scope

- Full ToolCatalog meta-schema validation, Mutation/Reconcile, guarantee descriptors and unknown-field rejection beyond the identity envelope above (WP7/WP8; documented coverage gap).
- CapabilityContracts, membership, Effect consistency, Scope intersection (WP3C).
- DependencyInspection inventory and selected-closure projection (WP3D).
- Schema G.4 validation/assignability and CEL (WP4).
- Procedure graph beyond WP3A baseline, SourcePath, Evidence, Script, Artifact, Chromium, Studio (WP5–WP12).
- Non-referenced malformed catalogs: not indexed, therefore not a source of exports; the later-stage fail-closed boundary is preserved.

## 6. Allowed paths

- `internal/compiler/dependency_identity.go` (extend reference metadata only);
- `internal/compiler/owner_linking.go` (new);
- `internal/compiler/owner_linking_test.go` (new);
- `internal/compiler/v1alpha4.go` (integration only);
- `tests/conformance/v1alpha4/fixtures/dependency-identity/**` (reuse/extend the Appendix E baseline);
- `tests/conformance/v1alpha4/wp3b-fixture-plan.json` (new inventory);
- `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only, so the WP3B branch/PR gaits run);
- this document.

Forbidden: CA-SL spec, frozen `docs/design/**`, machine contracts, `go.mod`/`go.sum`, dependency versions, other repositories, Host/Runtime/Studio, unrelated v0/v0.1 code.

## 7. Fixtures and acceptance

Fixture inventory: `tests/conformance/v1alpha4/wp3b-fixture-plan.json`.

Acceptance:
1. All WP3B fixture entries execute with exact expected (Code, Phase) and location rule.
2. E05, E39, E43, E91 variants each have an executable negative fixture derived from the Appendix E baseline by minimal change.
3. Positive baseline emits no WP3B diagnostic and still fails closed at later stages.
4. Check and Build use the same WP3B result; Build remains failure-atomic.
5. Input file/snapshot/binding order permutations do not change WP3B diagnostics or attribution.
6. `GOWORK=off go test ./... -count=1`, `GOWORK=off go vet ./...`, native build, Go/WASM build and `git diff --check` pass with Go 1.27.4.
7. CI evidence is linked; any external CI blocker is recorded honestly.

## 8. Stop conditions

Stop and report rather than guess if:
- the frozen spec/contracts require a diagnostic identity not present in Appendix H;
- a source-location rule other than §4 above is required;
- an export identity or Owner attribution has more than one defensible frozen interpretation;
- satisfying a fixture requires Capability, Schema-type, Evidence or Runtime semantics.
