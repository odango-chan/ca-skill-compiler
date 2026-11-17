# Compiler v1alpha4 Conformance and Acceptance Plan

**Status:** FROZEN CONFORMANCE PLAN — NOT A LIVE PROGRESS TRACKER. Current executed evidence and case dispositions are in the final acceptance record and `tests/conformance/v1alpha4/e-coverage.json`.

> **已归档 / ARCHIVED — COMPLETED RELEASE WORK.** 本文是 v1alpha4 实施期间形成的计划或决策记录。PR #11–#24 已合并，v1.0.0 已通过 PR #25 发布；原分支、Draft 状态、执行顺序和任务授权仅保留历史意义，不构成当前待办或新任务授权。当前实现状态以[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)、[已知限制](../known-limitations-v1alpha4.md)和[工作流索引](README.md)为准。


Normative source: [CA-SL v1alpha4 exact frozen commit](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).

## 1. Conformance principle

Appendix E's E01–E120 expected outcomes are normative. They are not evidence that a machine test has run. Appendix H defines exact (Code, Phase) diagnostic identity and precedence. A test is not complete merely because it references an E number or sees a matching code without its Phase.

Compiler executes only the statically decidable variants of E01–E120. Runtime-only variants remain Host/Runtime tests. Do not force runtime behavior into Compiler CI.

## 2. Test case identity

Every executable case records:
- immutable normative specification commit;
- exact E case / subcase identifier, or exact H (Code, Phase) diagnostic fixture;
- input file set and trusted context;
- expected success/failure;
- expected (Code, Phase) and severity for each diagnostic;
- source path, line and column expectations where normative;
- Native result and WASM result;
- test runner and toolchain versions.

Do not add or rename normative diagnostic codes. A test gap is a gap, not permission to invent a diagnostic.

## 3. Required fixture groups

| Group | Required coverage | Normative anchors |
|---|---|---|
| Request and digest | exact versions, path validation, limits, duplicate files, digest vectors and tamper rejection | BuildRequest contract; §34 |
| YAML / grammar | goccy/go-yaml v1.19.2 AST/token adapter; YAML 1.2 scalars, duplicate keys, aliases/depth bounds, unknown fields, identifier grammar, Contract/Procedure/PredicateSet structure | §§10–14; G.1; OSS ADR |
| Procedure graph | Node stages, route keys, terminals, DAG, reachability, global Precondition dominance, final Postcondition direct success | §§12, 22, E01–E120 |
| Callable linking | exact Alias/Export ID, kind, Owner, Effect, duplicate IDs across files/kinds, no fallback | §§9, 34, F.0–F.2 |
| Capability | exact grants, all matching Effect consistency, Scope policy resolution, Scope intersection, deployment/publisher/digest trust | §9.3; F.2; §34.0.1 |
| Schema | San v6.0.3 engine behind G.4 allowlist; exact Draft 2020-12 dialect, local-only $ref closure, no fragments/remote/cycles, Source ⊆ Target, nullability/required/enum/ranges/arrays | G.4; F.5; OSS ADR |
| CEL | cel-go v0.33.2 parse/check adapter; exact AST whitelist, declared parameter types, optional/null guards, int64/binary64 separation, static boolean result | §7.2; F.6; OSS ADR |
| SourcePath | root/segment validity, Producer dominance on every path, optional-field rejection, identity and Scope provenance | §13; E101/E104 |
| Evidence | Guarantee/Producer/output path, descriptor roots, freshness, Scope, Correlation, IndependentDomains and common-fact bindings | §17; F.3/F.4; G.3/G.3.1 |
| Mutation | receipt channels, commit proof, failure classes, Reconcile READ, recoverable output, retryOf and resource responsibility | §§9.2.1, 12.7, 20; G.2/G.2.2 |
| Script | gositter v0.2.0 / tree-sitter-python v0.25.0 CST adapter; both ABIs, strict TypedDict, functional TypedDict, virtual PredicateInput, optional-key access, dynamic-key rejection, generated Schema; no Python execution | §32; F.6; OSS ADR |
| Artifact | gowebpki/jcs v1.0.2 RFC 8785 bytes, byte-preserved source, generated Schemas, manifests, sorted dependency closure, failure atomicity | Artifact Contract; OSS ADR |
| Parity | Native/WASM identical diagnostics, phases, locations, file order, paths and bytes | §45.2 items 3–4 |

### 3.1 WP1 BuildRequest / Digest / output-envelope gates

WP1 is accepted only when executable tests cover all of the following:

- exact independent BuildRequest apiVersion and languageTarget checks, including missing / unsupported versions;
- required nonempty `files`, required `resolvedDependencies: []`, strict unknown-property rejection and base64 byte decoding;
- source-only `skills/<name>/**` paths, snapshot-relative dependency paths, invalid UTF-8 / traversal / absolute / duplicate paths and 240-byte path bounds;
- 2 MiB per-file, 16 MiB aggregate, 512 aggregate files, 128 dependency snapshots and 256 unique targets;
- all three frozen Dependency Snapshot Digest golden vectors, reversed input order, content/path mutation and duplicate-path rejection;
- recomputation of dependency, CapabilityContracts and EvidenceTrustBindings bundle digests before any parser/linker work;
- BuildResult and DependencyInspection exact apiVersion, required non-null arrays, diagnostic phase and failure atomicity;
- strict request handling and identical request/result shapes on Native and Go/WASM builds.

A work-package test is not a full Compiler qualification. Until later work packages implement the linker and artifact pipeline, source-only requests still fail closed and nonempty DependencyInspection inventories remain NOT_QUALIFIED.

### 3.2 WP2 YAML parser / normalized AST gates

WP2 is accepted only when executable tests and the build matrix cover:

- exactly one YAML parser: goccy/go-yaml v1.19.2 through the public lexer/parser/AST API; no yaml.v3 dependency or fallback;
- one normalized Compiler-owned AST representation shared by v1alpha4 source checks, Schema source checks, Skill frontmatter and remaining internal analysis helpers;
- typed v1alpha4 Skill / Contract / PredicateSet / Procedure / Node / Stage / PredicateCall views built from the normalized nodes, preserving source order and source spans;
- every author-maintained .yaml / .yml file in a selected Skill is parsed once; Contract, Procedure, PredicateSet and local Schema checks consume the same parsed node, with no reparse of the same source file;
- original scalar value, raw token, YAML tag, quote / block style, line and column are retained; exponent-only finite decimal notation is classified as a float independently of upstream implicit coercion;
- duplicate mapping keys, non-string and complex mapping keys, multiple documents, custom / unsupported tags, directives, merge keys, undefined aliases and cyclic aliases are rejected;
- YAML anchor rebinding follows the most-recent-preceding-anchor rule; alias expansion is bounded by 64 nesting levels, 4096 aliases and 20000 expanded nodes; token count is limited to 80000 and each document to 2 MiB;
- strict ScalarLiteral helpers cover lowercase Boolean / null, signed 64-bit decimal integers, finite binary64 decimal floats, quoted one-line strings and rejection of block scalars / explicit tags / aliases;
- JSON-originated values used by the existing Schema binding path enter the same normalized AST through ParseValue / ToValue; author-source Parse still requires a mapping root;
- parser errors and semantic diagnostics retain file-local source positions;
- Go unit tests, go vet, CGo-free Native build and Go/WASM build all pass.

WP2 does not qualify the full CA-SL scalar / SourcePath grammar, Schema semantics, Procedure linking, CEL, Script ABI, Artifact, Chromium Worker runtime parity or real 5GC Owner publications. Those remain later work-package gates.

## 4. E01–E120 coverage ledger

The test suite MUST maintain a machine-readable ledger with one row per normative E case and each independently testable variant. Each row is marked:
- NOT_IMPLEMENTED;
- FIXTURE_MISSING;
- IMPLEMENTED_NOT_RUN;
- PASS;
- FAIL;
- NOT_COMPILER_SCOPE (only for a runtime-only variant, with the owning Host/Runtime suite identified).

The ledger must never collapse multiple E variants into one green checkbox. E115–E120 in G.9 are included. A static case is not waived because a corresponding runtime case belongs elsewhere.

Appendix H items marked partial / pending fixtures require a separate executable fixture for the exact (Code, Phase). The informative Appendix K gap list helps locate candidate diagnostics but does not expand E01–E120 or prove coverage.

## 4.1 Machine-readable case and diagnostic indexes

The repository carries two machine-readable inventories derived from the pinned normative source:

- [case-index.json](../../tests/conformance/v1alpha4/case-index.json): one index row for each E01–E120 case, including the responsibility assignment recorded in Appendix K.1. Its initial status is NOT_IMPLEMENTED, matching the pinned source. It is an inventory, not an executable fixture suite.
- [diagnostic-index.json](../../tests/conformance/v1alpha4/diagnostic-index.json): all 99 unique (Code, Phase) identities in Appendix H (98 unique Code strings; EFFECT_MISMATCH occurs in two phases). It preserves the normative coverage label and records execution as NOT_VERIFIED.

Before a work package can be marked complete, its applicable E cases MUST be expanded into separate executable fixtures for every independently testable variant in Appendix E. A single case row or a test that checks only Code without Phase is insufficient. H entries marked FIXTURE_MISSING or PARTIAL require the exact missing fixture or an explicit owning Host/Runtime test reference. The indexes MUST be regenerated or updated only from the pinned normative source; they MUST NOT invent new E numbers or diagnostic identities.

## 5. Native / WASM / Studio parity

For every compiler-static fixture, run the same normalized BuildRequest through:
1. Native Go Core;
2. Go/WASM Core in an actual Chromium Worker;
3. Studio/LSP integration using the same Core result or equivalent shared implementation.

Compare success state, every diagnostic field (especially Code + Phase), source path/line/column, dependencyClosure order, output file path/order, every output byte and deterministic result under reversed input file ordering.

A WASM startup test is not parity evidence. A single positive fixture is not diagnostic parity.

## 6. Script and Schema gates

Script tests must include positive and negative cases for the gositter v0.2.0 Python CST adapter against the pinned tree-sitter-python v0.25.0 grammar, exact main signatures, Procedure call-site derived input TypedDict, Predicate-generated virtual PredicateInput, nested and functional TypedDict, Literal, Required / NotRequired and supported Annotated constraints, optional-key guard behavior, unknown literal and dynamic key rejection, forbidden Any/open dict/union/unsupported metadata, malformed/pathological source limits, generated Output Schema and runtime output validation contract. Never execute Python in Compiler tests.

Schema tests must not silently widen G.4. A general Draft 2020-12 library passing a keyword is not proof that CA-SL accepts it. San v6.0.3 is the selected engine, but production adapter conformance remains NOT_QUALIFIED until the exact G.4 allowlist, local-only reference closure, security and Native/WASM gates pass. PR #8's spike is technical evidence, not production conformance.


### 6.1 Selected dependency adapter gates

The pinned implementation dependencies and their versions / licenses are defined by the [OSS Dependency Selection ADR](../design/oss-dependency-selection-v1alpha4.md). This conformance plan does not permit substitution by an unreviewed library or floating version.

- YAML: use the goccy/go-yaml AST/token API; test YAML 1.2 scalar behavior, duplicate keys, alias cycles/expansion limits, source positions and unknown fields.
- JSON Schema: use San v6.0.3 only after G.4 pre-validation; deny all non-snapshot resource loading and prove no network or filesystem access.
- CEL: use cel-go v0.33.2 only for parsing/type checking; disable macros and reject AST nodes outside the frozen v1alpha4 subset.
- Python: use gositter v0.2.0 with only its Python grammar package; test the CST adapter against the Script ABI fixtures and a pinned Tree-sitter C oracle in a test-only job. The Compiler must remain CGo-free and must never execute Python.
- JCS: use gowebpki/jcs v1.0.2 and the RFC 8785 vectors; preserve the separate Dependency Snapshot Digest algorithm unchanged.
- All production dependencies must build with `CGO_ENABLED=0` for Native and Go/WASM. CI pins Go 1.27.4. License inventory, resource bounds and Native/WASM byte parity are release gates.

The ADR selects libraries; it does not mark any adapter or Compiler work package PASS. Add each direct dependency to go.mod only in the work package that first consumes it, together with its tests.

## 7. 5GC executable acceptance

The only executable 5GC baseline is the conventional establish-pdu-session flow after it is migrated to v1alpha4 and supplied with real, approved Owner contracts.

Required gates:
- real ToolCatalog v1alpha4 and CapabilityContracts v1alpha4 publications, plus an explicitly versioned trusted Evidence Descriptor publication for any Evidence-dependent Skill;
- actual Host Context, Tool Input/Output, Observation, Evidence and responsibility Schemas;
- Owner confirmation for the three asynchronous-completion Operations required by the frozen spec;
- complete static compile and exact diagnostic closure;
- regression of all 36 existing Case obligations against the migrated source;
- end-to-end evidence from the actual 5GC environment.

MAS is not an executable acceptance baseline. Its proposed ToolCatalog and unresolved operations must remain explicitly NOT_READY; do not make MAS green with synthetic Owner contracts.

The current ca-skills source inventory says the present 5GC Skill files are legacy and NOT v1alpha4-conforming. Do not change only the version string. See [ca-skills current 5GC baseline](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/current-5gc-skill-baseline.md).

## 8. Current implementation evidence boundary

Compiler PR #11 adds source/graph admission and explicit fail-closed behavior when trusted Owner / CapabilityContracts inputs are missing. It does not implement the full linker, SourcePath/Evidence proof, Script ABI, E01–E120, complete H diagnostics or Native/WASM parity for v1alpha4.

This document refresh itself is documentation only. No Go tests, WASM build, Chromium parity or 5GC tests are claimed PASS by this change.


## WP3A fixture plan — first executable dependency slice

The first DeepSeek task has a dedicated machine-readable fixture plan at [wp3a-fixture-plan.json](../../tests/conformance/v1alpha4/wp3a-fixture-plan.json).

It covers the dependency identity diagnostics and the Appendix E cases E03, E04 and E16 that belong to this slice. The plan is **PLANNED_NOT_EXECUTED**: entries are not passing tests until executable fixture files, test assertions and CI evidence exist. E05/E39/E43 (Owner/export attribution) are reserved for WP3B; Capability/Scope cases are reserved for WP3C.

No E or H status is changed to PASS by adding this plan.
