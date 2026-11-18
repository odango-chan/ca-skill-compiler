> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP4D — CEL Subset and Predicate Parameter Signatures

**Task ID:** `WP4D-CEL-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — slice 4 of WP4 implemented on `feat/compiler-v1alpha4-wp4d-cel-20261116` for Draft PR review. Agent-proposed pending maintainer ratification.  
**Depends on:** WP4A/WP4B/WP4C (PR #19 head `1dcc4d3`, PR #20 `755f621`)  
**Task branch:** `feat/compiler-v1alpha4-wp4d-cel-20261116`

## 1. Goal

Integrate the ADR-selected cel-go v0.33.2 parser/type checker behind a CA-SL-owned v1alpha4 subset adapter and validate Predicate parameter signatures. cel-go does not define the accepted subset; a successful generic compile is never sufficient.

## 2. Frozen authority

CA-SL v1alpha4 `ca-skills@7f84393`: G.4, G.4.1, §24.1 (`CEL_INVALID`, `PREDICATE_GROUP_INVALID`). [OSS ADR](../design/oss-dependency-selection-v1alpha4.md) §5. E110/E111 scope.

## 3. Required behavior

1. Predicate parameters: ordered one-key declarations with unique nonempty names; type is a PrimitiveType (`string`/`integer`/`number`/`boolean`) or an exact alias-qualified external Schema identity; otherwise `PREDICATE_GROUP_INVALID`.
2. cel evaluator: `evaluator.type == cel` with a nonempty `expression`. `script` evaluators are WP9 scope.
3. Environment is built only from declared parameters; no ambient/global Context. Static result must be exactly bool.
4. Parsed-tree whitelist (enforced after successful compile): literals, identifiers, static field dot access, parentheses, boolean `&&`/`||`/`!`, equality/inequality, same-numeric-type ordering, `has`, `size`. Methods, macros/comprehensions, indexing, list/map/object construction, unknown functions and other constructs are `CEL_INVALID`.
5. The expression is never evaluated at compile time; int/double mixing is rejected by the CEL type checker. Runtime overflow/non-finite handling remains Host/Runtime.

## 4. Documented WP4D limitations (deferred)

- External Schema-typed parameters use a dynamic CEL type in this slice; exact field typing from the linked Schema is WP6.
- Optional/null guard proofs (`has`/`!= null`) and safe-integer binding range enforcement are WP6.
- No Predicate Evidence signature checks (WP7).

## 5. Allowed paths

`internal/compiler/cel_check.go` (new), `internal/compiler/cel_check_test.go` (new), `internal/compiler/v1alpha4.go` (integration only), `go.mod`/`go.sum` (only the ADR-pinned `cel.dev/cel-go v0.33.2` and reviewed transitive requirements), `tests/conformance/v1alpha4/wp4d-fixture-plan.json` (new), `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only), this document. Forbidden: CA-SL spec, frozen design/machine contracts, other dependency versions, other repositories, Host/Runtime/Studio.

## 6. Fixtures and acceptance

Inventory: `tests/conformance/v1alpha4/wp4d-fixture-plan.json`. Acceptance: valid expressions accepted; out-of-subset/non-boolean/type-error/undeclared-parameter expressions `CEL_INVALID` with predicates source location; invalid parameter type `PREDICATE_GROUP_INVALID`; full Go test/vet/Native/WASM/diff gates; CI evidence linked (external blockers recorded honestly).

## 7. Stop conditions

Stop rather than guess if the frozen subset is ambiguous, a needed diagnostic is absent from Appendix H, or the slice would require Runtime evaluation or Schema field typing that WP6 owns.
