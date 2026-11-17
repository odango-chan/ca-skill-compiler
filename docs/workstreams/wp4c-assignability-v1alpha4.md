# WP4C — F.5 Conservative Schema Assignability

**Task ID:** `WP4C-ASSIGNABILITY-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — slice 3 of WP4 implemented on `feat/compiler-v1alpha4-wp4c-assignability-20261116` for Draft PR review. Agent-proposed pending maintainer ratification.  
**Depends on:** WP4A/WP4B (`1dcc4d3`, PR #19 Draft)  
**Task branch:** `feat/compiler-v1alpha4-wp4c-assignability-20261116`

## 1. Goal

Provide the Compiler-owned F.5 conservative `Source ⊆ Target` proof engine for Schema pairs that already passed G.4 and whose `$ref`s are resolved by the caller. Later work packages (WP6 argument typing, WP7 Evidence value typing) consume this engine; WP4C itself does not invent call sites or runtime semantics.

## 2. Frozen authority

CA-SL v1alpha4 `ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`: F.5, G.4, G.4.1, §24.1 (`INPUT_TYPE_MISMATCH`, `INPUT_NULLABILITY_MISMATCH`). Compiler Contract §7.

## 3. Required behavior

- integer is assignable to number; number is not assignable to integer; no string/number coercion.
- Source enum/const values must all be accepted by the Target enum/const.
- Source nullability must be accepted by the Target; otherwise the caller emits `INPUT_NULLABILITY_MISMATCH`.
- Target required properties must be guaranteed by Source required properties; each declared Source property must be assignable to the corresponding Target property; Source may not add keys forbidden by a closed Target.
- String length bounds, numeric bounds, array item type and array length bounds must be implied by Source.
- `$ref` targets must be resolved by the caller/registry before assignment; unresolved references are never guessed (`UNRESOLVED`).
- Anything not provable is rejected (`TYPE_MISMATCH`), never accepted through a generic Schema evaluation, coercion, `any` widening or default materialization.

## 4. Out of scope

Schema registry/closure (WP4B); CEL (WP4D); SourcePath/argument binding (WP6); Evidence (WP7); Runtime. No diagnostics are emitted by this slice; it returns a reason for callers to map.

## 5. Allowed paths

`internal/compiler/schema_assignability.go` (new), `internal/compiler/schema_assignability_test.go` (new), `tests/conformance/v1alpha4/wp4c-fixture-plan.json` (new), `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only), this document. Forbidden: CA-SL spec, frozen design/machine contracts, `go.mod`/`go.sum`, other repositories, Host/Runtime/Studio.

## 6. Fixtures and acceptance

Inventory: `tests/conformance/v1alpha4/wp4c-fixture-plan.json`. Acceptance: positive and negative F.5 cases (numbering, enum/const, nullability, required/extra keys, bounds, arrays, unresolved refs); full Go test/vet/Native/WASM/diff gates; CI evidence linked (external blockers recorded honestly).

## 7. Stop conditions

Stop rather than guess if F.5 cannot decide a case without coercion/defaults, a caller needs a diagnostic identity absent from Appendix H, or the slice would require Runtime/CEL semantics.
