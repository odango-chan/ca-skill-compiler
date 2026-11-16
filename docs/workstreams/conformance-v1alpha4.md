# Compiler v1alpha4 Conformance and Acceptance Plan

**Status:** FROZEN implementation acceptance plan; this document does not claim full v1alpha4 conformance.

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
| YAML / grammar | YAML 1.2 scalars, duplicate keys, unknown fields, identifier grammar, Contract/Procedure/PredicateSet structure | §§10–14; G.1 |
| Procedure graph | Node stages, route keys, terminals, DAG, reachability, global Precondition dominance, final Postcondition direct success | §§12, 22, E01–E120 |
| Callable linking | exact Alias/Export ID, kind, Owner, Effect, duplicate IDs across files/kinds, no fallback | §§9, 34, F.0–F.2 |
| Capability | exact grants, all matching Effect consistency, Scope policy resolution, Scope intersection, deployment/publisher/digest trust | §9.3; F.2; §34.0.1 |
| Schema | exact Draft 2020-12 dialect, G.4 allowlist, $ref closure, no fragments/remote/cycles, Source ⊆ Target, nullability/required/enum/ranges/arrays | G.4; F.5 |
| SourcePath | root/segment validity, Producer dominance on every path, optional-field rejection, identity and Scope provenance | §13; E101/E104 |
| Evidence | Guarantee/Producer/output path, descriptor roots, freshness, Scope, Correlation, IndependentDomains and common-fact bindings | §17; F.3/F.4; G.3/G.3.1 |
| Mutation | receipt channels, commit proof, failure classes, Reconcile READ, recoverable output, retryOf and resource responsibility | §§9.2.1, 12.7, 20; G.2/G.2.2 |
| Script | both ABIs, strict TypedDict, functional TypedDict, virtual PredicateInput, optional-key access, dynamic-key rejection, generated Schema | §32; F.6 |
| Artifact | byte-preserved source, JCS generated Schemas, manifests, sorted dependency closure, failure atomicity | Artifact Contract |
| Parity | Native/WASM identical diagnostics, phases, locations, file order, paths and bytes | §45.2 items 3–4 |

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

Script tests must include positive and negative cases for exact main signatures, Procedure call-site derived input TypedDict, Predicate-generated virtual PredicateInput, nested and functional TypedDict, Literal, Required / NotRequired and supported Annotated constraints, optional-key guard behavior, unknown literal and dynamic key rejection, forbidden Any/open dict/union/unsupported metadata, generated Output Schema and runtime output validation contract.

Schema tests must not silently widen G.4. A general Draft 2020-12 library passing a keyword is not proof that CA-SL accepts it. Production Schema Engine adoption remains a separately reviewed implementation choice; PR #8's spike is not production adoption.

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
