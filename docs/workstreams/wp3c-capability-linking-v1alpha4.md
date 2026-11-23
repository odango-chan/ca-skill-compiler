> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP3C — Trusted CapabilityContracts, Membership, Effect and Scope Presence

**Task ID:** `WP3C-CAPABILITY-LINK-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — implemented on `feat/compiler-v1alpha4-wp3c-capability-20261116` for Draft PR review. The frozen [Harness profile](deepseek-harness-v1alpha4.md) assigns WP3C to the human maintainer's queue; this document records the exact envelope used so the maintainer can ratify, amend or reject it before merge. It does not amend the frozen Harness profile, the Compiler contracts or CA-SL.  
**Depends on:** WP3B (`bca99de`, PR #16 Draft)  
**Task branch:** `feat/compiler-v1alpha4-wp3c-capability-20261116`

## 1. Goal

For every external callable actually referenced by the selected Skill and already proven by WP3A/WP3B, verify against the trusted deployment `CapabilityContracts` input:

`trusted deployment input -> exact callable membership -> Effect consistency -> granted Capability -> trusted Scope policy reference`

WP3C is static authorization *consistency* only. It is not proof that a deployment is operationally authorized, not Runtime admission and not a Scope-evaluation engine.

## 2. Frozen authority

- CA-SL v1alpha4 `ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`: §§9.3, 24.1, 34.0–34.0.1, F.2, G.1, Appendix E (E06, E07).
- [Owner Contract Linking](../design/owner-contracts-v1alpha4.md) §3.
- [BuildRequest v1alpha4](../design/build-request-v1alpha4.md) §§2.1, 5.2.
- Machine contract: `contracts/v1alpha4/capability-contracts.schema.json`.
- Fixture baseline: Appendix E.4 capability-contracts document; WP3A/WP3B fixtures.

## 3. Required behavior

1. A referenced external callable requires the trusted `trustedBuildContext.capabilityContracts` bundle. Missing bundle → `CAPABILITY_INPUT_UNTRUSTED`.
2. The trusted caller must supply a `deploymentAuthorizations` record whose exact `deploymentId` and `publisherId` equal the bundle's `deploymentId` and `publisherId`. Missing/mismatched authorization → `CAPABILITY_INPUT_UNTRUSTED`. The Compiler does not dereference `authorizationRef`.
3. Parse every YAML/JSON file of the trusted bundle through the single WP2 `yamlast` AST. A file participates only with exact `apiVersion: connectionagent.io/capability-contracts/v1alpha4` and `kind: CapabilityContracts`. If no participating document exists → `CAPABILITY_INPUT_UNTRUSTED`.
4. Capability identity: unique `id` (DottedId ≤240), nonempty `owner` (≤160), nonempty `callables[]`. A Capability ID defined more than once in the artifact set → `CAPABILITY_DEFINITION_CONFLICT` (first definition retained; later ones are not indexed).
5. Member identity: exact `(owner, callableId, kind)` with `kind` ∈ {operation, observation}, `effect` ∈ {READ, MUTATION}. The same member identity declared twice inside one Capability → `CAPABILITY_MEMBERSHIP_DUPLICATE`. Consistent membership across different Capabilities is legal.
6. For each referenced callable export proven by WP3B (operation/observation with trusted Owner):
   - match all member records by exact trusted `(owner, callableId, kind)`;
   - no matching record → `CAPABILITY_MAPPING_MISSING`;
   - every matching record, granted or not, must have the same Effect as the trusted ToolCatalog callable, else `EFFECT_MISMATCH`;
   - at least one matching record must belong to a Capability the selected Skill grants, else `CAPABILITY_NOT_GRANTED`;
   - every matching granted record must carry a nonempty trusted `scopePolicyRef` (≤512), else `CAPABILITY_SCOPE_UNRESOLVED`.
7. Skill Grants are read from `Contract.capabilities` as ordered `CapabilityId` scalars. An invalid Grant identifier → `IDENTIFIER_INVALID` at the Contract source node. A malformed (non-sequence) Grant list grants nothing and therefore surfaces the frozen per-callable conditions.
8. No Scope-policy content format is consumed because none is frozen. `CAPABILITY_SCOPE_EMPTY_INTERSECTION` is therefore **not** emitted in this slice: static emptiness cannot be proven without inventing a policy format. This is a documented fail-closed limitation, not an acceptance of wider authorization.
9. Result metadata (parsed grants, capabilities and members) is retained for WP3D. No `DependencyInspection` inventory is published and no overall success is claimed.

## 4. Frozen diagnostics in this task

| Code | Phase | Trigger |
|---|---|---|
| `CAPABILITY_INPUT_UNTRUSTED` | `C_STATIC` | Trusted deployment input missing, unsupported, or not bound to an authorization |
| `CAPABILITY_DEFINITION_CONFLICT` | `C_STATIC` | Same Capability ID defined more than once in the artifact set |
| `CAPABILITY_MEMBERSHIP_DUPLICATE` | `C_STATIC` | Same member identity declared twice inside one Capability |
| `CAPABILITY_MAPPING_MISSING` | `C_STATIC` | Referenced callable has no matching member record |
| `EFFECT_MISMATCH` | `C_STATIC` | Any matching member Effect differs from the ToolCatalog Effect |
| `CAPABILITY_NOT_GRANTED` | `C_STATIC` | No matching Capability is granted by the selected Skill |
| `CAPABILITY_SCOPE_UNRESOLVED` | `C_STATIC` | Granted matching member lacks a trusted resolvable Scope policy reference |
| `IDENTIFIER_INVALID` | `C_STATIC` | Contract CapabilityId violates the frozen DottedId grammar |

No new diagnostic code or phase is introduced.

### Source location convention used by this envelope

- Trusted deployment failures (`CAPABILITY_INPUT_*`, `CAPABILITY_DEFINITION_CONFLICT`, `CAPABILITY_MEMBERSHIP_DUPLICATE`, `CAPABILITY_MAPPING_MISSING`, `EFFECT_MISMATCH`, `CAPABILITY_NOT_GRANTED`, `CAPABILITY_SCOPE_UNRESOLVED`) use empty path and zero line/column.
- The Contract Grant identifier error (`IDENTIFIER_INVALID`) retains the original WP2 Contract source path/line/column.
- Deployment bundle file paths are never reported as Skill source paths.

## 5. Explicitly out of scope

- Runtime deployment authorization, Scope evaluation, dispatch admission and scope intersection runtime checks (Host/Runtime).
- Scope policy document formats and `CAPABILITY_SCOPE_EMPTY_INTERSECTION` (no frozen format; must not be invented).
- Mutation/Reconcile, Evidence, Schema G.4/assignability, CEL, Procedure graph, SourcePath, Script, Artifact, Chromium, Studio (later WPs).
- Full CapabilityContracts meta-schema validation beyond the identity envelope above; malformed/unknown fields are not indexed and fail closed through the frozen missing/identity conditions when referenced.

## 6. Allowed paths

- `internal/compiler/capability_linking.go` (new);
- `internal/compiler/capability_linking_test.go` (new);
- `internal/compiler/v1alpha4.go` (integration and residual fail-closed message constant only);
- `internal/compiler/v1alpha4_test.go` (update one expectation invalidated by the precise WP3C behavior);
- `tests/conformance/v1alpha4/fixtures/capability-linking/**` (new Appendix E.4 based fixture);
- `tests/conformance/v1alpha4/wp3c-fixture-plan.json` (new inventory);
- `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only);
- this document.

Forbidden: CA-SL spec, frozen `docs/design/**`, machine contracts, `go.mod`/`go.sum`, dependency versions, other repositories, Host/Runtime/Studio, unrelated v0/v0.1 code.

## 7. Fixtures and acceptance

Fixture inventory: `tests/conformance/v1alpha4/wp3c-fixture-plan.json`.

Acceptance:
1. Every fixture entry executes with exact expected (Code, Phase) and location rule.
2. E06 and E07 variants have executable negative fixtures derived from the Appendix E baseline by minimal change.
3. Positive baseline emits no WP3C diagnostic and still fails closed at later stages.
4. Check and Build use the same WP3C result; Build remains failure-atomic.
5. Bundle file order permutations do not change indexing or diagnostics.
6. `GOWORK=off go test ./... -count=1`, `GOWORK=off go vet ./...`, native build, Go/WASM build and `git diff --check` pass with Go 1.27.4.
7. CI evidence is linked; any external CI blocker is recorded honestly.

## 8. Stop conditions

Stop and report rather than guess if:
- a required diagnostic identity is missing from Appendix H (e.g. a Scope policy document format that would be needed to prove an empty intersection);
- a source-location rule other than §4 above is required;
- satisfying a fixture requires Runtime Scope evaluation, provider authorization or Schema/Evidence semantics;
- the trusted deployment input shape is ambiguous.
