> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP4B — Schema Registry, Exact Reference Closure and San v6.0.3 Adapter

**Task ID:** `WP4B-SCHEMA-REGISTRY-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — slice 2 of WP4 implemented on `feat/compiler-v1alpha4-wp4b-schema-registry-20261116` for Draft PR review. WP4 slices: WP4A G.4 allowlist (PR #19), **WP4B** registry/closure/San (this document), WP4C F.5 assignability, WP4D CEL. Agent-proposed pending maintainer ratification.  
**Depends on:** WP4A (`6663725`, PR #19 Draft)  
**Task branch:** `feat/compiler-v1alpha4-wp4b-schema-registry-20261116`

## 1. Goal

Build the exact Schema registry for the selected Skill, resolve the participating Schema reference closure without guessing, enforce the §11.5/G.6 Request/Context root shape through a root `$ref` chain, and integrate the ADR-selected San v6.0.3 engine only over Schemas that already passed the CA-SL G.4 allowlist.

## 2. Frozen authority

CA-SL v1alpha4 `ca-skills@7f84393`: G.4, G.6, §11.5, §24.1. [Compiler Contract](../design/compiler-contract-v1alpha4.md) §7. [OSS ADR](../design/oss-dependency-selection-v1alpha4.md) §4 (San: in-memory resources, deny-by-default loader, exact `$id`, no ambient fetch, no G.4 broadening). E110.

## 3. Required behavior

1. Registry identity: Skill-owned Schemas are keyed by their exact Skill-relative source path; external Schemas are keyed by their exact absolute `$id` obtained from WP3A-verified snapshot bytes and WP3B-attributed Owner.
2. Reference closure starts from participating roots: the Contract `requestSchema` (local), `contextSchema` (external `Alias::URI`) and every WP3B-linked external Schema export. `$ref` targets resolve only by exact `$id`, optionally alias-qualified; fragments, prefixes, file names and order are never used.
3. Missing target → `SCHEMA_MISSING`; ambiguous target → `EXPORT_ID_AMBIGUOUS`; cycle → `SCHEMA_INVALID`; G.4 violation → `SCHEMA_INVALID`. Reference-identity diagnostics already owned by WP3B are not duplicated.
4. External Schema root closure follows the permitted finite root `$ref` chain. Host Context effective root must be a closed object Schema and must not declare or require the reserved property `request`; violation is `CONTEXT_REQUEST_COLLISION`, other unsupported root structure is `SCHEMA_INVALID`.
5. San v6.0.3 is used only after G.4 acceptance: resources are registered in memory, the loader rejects all ambient retrieval, and compile failures are `SCHEMA_INVALID`. San does not define the accepted keyword set.
6. Diagnostics: local-resource errors retain path/line/column; external-resource/closure errors use empty path and zero location. No new diagnostic code.

## 4. Out of scope

F.5 conservative assignability (WP4C); CEL subset/type checking and Predicate signatures (WP4D); Procedure graph, SourcePath, Evidence, Mutation, Script, Artifact, Chromium, Studio.

## 5. Allowed paths

`internal/compiler/schema_registry.go` (new), `internal/compiler/schema_registry_test.go` (new), `internal/compiler/v1alpha4.go` (integration only), `go.mod`/`go.sum` (only the ADR-pinned `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` and its reviewed transitive requirement), `tests/conformance/v1alpha4/wp4b-fixture-plan.json` (new), `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only), this document. Forbidden: CA-SL spec, frozen design/machine contracts, other dependency versions, other repositories, Host/Runtime/Studio.

## 6. Fixtures and acceptance

Inventory: `tests/conformance/v1alpha4/wp4b-fixture-plan.json`. Acceptance: exact (Code, Phase)/location assertions; positive closure accepted including San compile; external missing ref, cycle, Context root collision, no duplicate missing-reference diagnostic; Check/Build agreement; full Go test/vet/Native/WASM/diff gates; CI evidence linked (external blockers recorded honestly).

## 7. Stop conditions

Stop rather than guess if exact `$id` resolution is ambiguous without a frozen precedence, a needed diagnostic is absent from Appendix H, or the slice would require F.5/CEL/Runtime semantics.
