# WP5 — Procedure Graph Reachability and Global Precondition Dominance

**Task ID:** `WP5-PROCEDURE-GRAPH-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE (slice 1) — implemented on `feat/compiler-v1alpha4-wp5-procedure-graph-20261116` for Draft PR review. Existing v1alpha4.go already enforces the frozen stage/route/cycle/terminal/final-success checks; this slice adds reachability and dominance proofs that require the reachable graph and trusted callable Effects.  
**Depends on:** WP4D (`d78aedf`, PR #21)  
**Task branch:** `feat/compiler-v1alpha4-wp5-procedure-graph-20261116`

## 1. Goal

Add the Procedure graph proofs missing from the existing single-file checks: `UNREACHABLE_NODE` warning (reachability from the unique start) and `ENTRY_PRECONDITION_BYPASSED` when a reachable path reaches a trusted business MUTATION callable or a `SUCCEEDED` route target without visiting the Contract global Precondition stage.

## 2. Frozen authority

CA-SL v1alpha4 `ca-skills@7f84393`: §12.2, §12.5, §12.8, §12.9, §24.1, G.8, E24/E44/E51/E52/E65. [Procedure Linker contract](../design/procedure-link-v1alpha4.md).

## 3. Required behavior

1. Build the reachable graph from the unique `start` over declared stage route targets (terminals excluded). A declared Node not reachable from `start` emits `UNREACHABLE_NODE` / `C_WARNING` (warning does not independently block Build; existing result remains fail-closed).
2. When Contract declares a valid single `<node>.precondition` global gate, prove that no reachable path reaches a `SUCCEEDED` route target or a trusted `effect: MUTATION` callable without visiting the gate Node; otherwise `ENTRY_PRECONDITION_BYPASSED` / `C_STATIC` at the Contract precondition reference.
3. Trusted MUTATION identity comes only from WP3 linked callables (`Alias::ExportID` + exact `effect: MUTATION`); no Effect is inferred from names or author text. Nodes whose Effects are not linked are not classified.
4. No new diagnostic code or phase. The existing grammar/cardinality/route/cycle/terminal/final-success checks remain authoritative and unchanged.
5. Documented slice limitation: auto-advance semantics are approximated by treating a visited gate Node as satisfying the gate; full stage-level dominance (and mutation classification without trusted Effects) remains follow-up WP5 work.

## 4. Allowed paths

`internal/compiler/procedure_graph.go` (new), `internal/compiler/procedure_graph_test.go` (new), `internal/compiler/v1alpha4.go` (integration only), `tests/conformance/v1alpha4/wp5-fixture-plan.json` (new), `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only), this document.

## 5. Fixtures and acceptance

Inventory: `tests/conformance/v1alpha4/wp5-fixture-plan.json`. Acceptance: positive graph clean; unreachable Node warning exact (Code, Phase); global-gate bypass error with Contract source location; Check/Build agreement; full Go test/vet/Native/WASM/diff gates; CI evidence linked (external blockers recorded honestly).

## 6. Stop conditions

Stop rather than guess if dominance needs runtime Outcome semantics, a diagnostic identity is absent from Appendix H, or a route/terminal rule conflicts with the frozen §12 text.
