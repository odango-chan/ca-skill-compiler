> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP4A — CA-SL G.4 Schema Allowlist and Request Schema Root Shape

**Task ID:** `WP4A-SCHEMA-G4-001`  
**Status:** AGENT-PROPOSED BOUNDED ENVELOPE — slice 1 of WP4 implemented on `feat/compiler-v1alpha4-wp4a-schema-g4-20261116` for Draft PR review. WP4 is split into independently reviewable slices: **WP4A** G.4 allowlist + Request Schema root shape (this document), **WP4B** Schema registry, exact local reference closure and the San v6.0.3 adapter, **WP4C** F.5 conservative assignability, **WP4D** CEL subset parsing/type checking and Predicate signatures. The frozen [Harness profile](deepseek-harness-v1alpha4.md) assigns WP4 to the human maintainer's queue; this envelope is agent-proposed pending ratification.  
**Depends on:** WP3D (`142ef78`, PR #18 Draft)  
**Task branch:** `feat/compiler-v1alpha4-wp4a-schema-g4-20261116`

## 1. Goal

Enforce the frozen CA-SL G.4 JSON Schema dialect allowlist and structural consistency for every Skill-owned Schema, and enforce the §11.5 / G.6 Request Schema root shape, before any generic Schema engine or later type analysis may consume those Schemas.

WP4A does not resolve external Schema references, prove F.5 assignability, parse CEL, evaluate a Predicate or accept arbitrary Draft 2020-12 documents.

## 2. Frozen authority

- CA-SL v1alpha4 `ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`: G.4, G.4.1, G.6, §11.5, §24.1, F.5 boundary, E110.
- [Compiler Contract](../design/compiler-contract-v1alpha4.md) §7.
- [OSS Dependency Selection ADR](../design/oss-dependency-selection-v1alpha4.md) §2, §4 (library parses; CA-SL defines meaning; San adapter is a later slice).

## 3. Required behavior

1. Every Skill-owned `schemas/**` YAML document is validated during Check/Build.
2. A participating Schema root MUST declare exactly `$schema: https://json-schema.org/draft/2020-12/schema`; a missing or different dialect is `SCHEMA_INVALID`.
3. The accepted keyword set is exactly:
   - assertions: `$ref`, `type`, `properties`, `required`, `additionalProperties`, `items`, `minItems`, `maxItems`, `minLength`, `maxLength`, `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `enum`, `const`;
   - annotations: `$schema`, `$id`, `title`, `description`, `default`, `examples`.
   Any other keyword (`$defs`, `definitions`, `allOf`, `anyOf`, `oneOf`, `not`, `if/then/else`, `pattern`, `patternProperties`, `unevaluatedProperties`, `contains`, `prefixItems`, `format`, dynamic references, extension keywords, …) is `SCHEMA_INVALID`; unknown keywords are never ignored.
4. `type` accepts one supported type (`string`, `integer`, `number`, `boolean`, `object`, `array`, `null`) or exactly `[T, "null"]` with two distinct members; general unions, duplicate members and malformed unions are `SCHEMA_INVALID`.
5. Structural consistency:
   - `required` is a unique list of names that is a subset of declared `properties`;
   - `additionalProperties` is the boolean `true`/`false`;
   - `items` is one Schema, never a tuple array;
   - `minItems`/`maxItems` and `minLength`/`maxLength` are non-negative integers with a satisfiable range;
   - `minimum`/`maximum`/`exclusiveMinimum`/`exclusiveMaximum` are finite numbers with a satisfiable range;
   - numeric/length/item constraints require a compatible declared `type`;
   - `properties`/`required`/`items` require the compatible declared `type`;
   - `enum`/`const` values must be accepted by the declared type and constraints, and `enum` values must be unique.
6. `$ref` is an independent reference object: it may only be accompanied by `$schema`/`$id`/`title`/`description`; the value must be an absolute Schema identity (`Alias::absoluteURI` or absolute URI) without a fragment. `#/...` fragments and `$defs` are `SCHEMA_INVALID`. Reference closure resolution is WP4B.
7. §11.5 / G.6 Request Schema root shape: the effective Skill-owned Request Schema root must be an object Schema with root fields declared by `properties` and `additionalProperties: false`, else `SCHEMA_INVALID`. A root `$ref` is deferred to the WP4B registry slice and is not guessed. `REQUEST_SCHEMA_MISSING` remains unchanged when the Contract names a missing Skill-owned Request Schema.
8. Diagnostics use the frozen `SCHEMA_INVALID` (C_STATIC) identity and retain the offending Schema file path/line/column. No new diagnostic code is introduced.

## 4. Explicitly out of scope (later WP4 slices)

- WP4B: Schema registry, exact local reference closure, cycle/missing-reference diagnostics, San v6.0.3 adapter, external Schema Owner binding into the registry.
- WP4C: F.5 conservative `Source ⊆ Target` assignability, nullable/integer-number rules, input type diagnostics.
- WP4D: CEL v1alpha4 subset parsing/type checking, Predicate parameter signatures, `CEL_INVALID` / `PREDICATE_*`.
- Non-Schema author checks, Procedure graph, SourcePath, Evidence, Mutation, Script, Artifact, Chromium, Studio.

## 5. Allowed paths

- `internal/compiler/schema_g4.go` (new);
- `internal/compiler/schema_g4_test.go` (new);
- `internal/compiler/v1alpha4.go` (replace the legacy partial Schema walk with the G.4 validator; Request Schema root check only);
- `tests/conformance/v1alpha4/wp4a-fixture-plan.json` (new inventory);
- `.github/workflows/ca-sl-v1alpha4-source.yml` (branch filters only);
- this document.

Forbidden: CA-SL spec, frozen `docs/design/**`, machine contracts, `go.mod`/`go.sum` (the San/CEL adapters land in their own slices), dependency versions, other repositories, Host/Runtime/Studio, unrelated v0/v0.1 code.

## 6. Fixtures and acceptance

Fixture inventory: `tests/conformance/v1alpha4/wp4a-fixture-plan.json`.

Acceptance:
1. Every WP4A fixture entry executes with exact expected (Code, Phase) and source location.
2. E110 Schema variants (anyOf / pattern / dynamic reference) are rejected as `SCHEMA_INVALID`.
3. Valid G.4 Schemas are accepted; no over-rejection of allowed keywords, nullable unions or `$ref` annotations.
4. Request Schema root shape and missing Request Schema behave as specified.
5. Every Skill-owned Schema file is validated, not only the Contract-referenced one.
6. Check and Build use the same validation; Build remains fail-closed and failure-atomic.
7. `GOWORK=off go test ./... -count=1`, `GOWORK=off go vet ./...`, native build, Go/WASM build and `git diff --check` pass with Go 1.27.4.
8. CI evidence is linked; external CI blockers are recorded honestly.

## 7. Stop conditions

Stop and report rather than guess if:
- G.4 requires accepting a keyword not in the frozen allowlist;
- a Request Schema root shape question cannot be resolved from §11.5 / G.6;
- a required diagnostic identity is missing from Appendix H;
- the slice would need external reference resolution, F.5 assignability or CEL semantics.
