# Schema Engine Migration Sequencing Decision v0.1

**Status: FROZEN — sequencing decision only**  
**Date:** 2026-11-12  
**Authority:** Compiler architecture decision; cross-repository dependency on ca-skills vNext  
**Related:** PR #5, PR #6, PR #7, PR #8; ca-skills PR #24

## 1. Decision

Production Schema Engine migration is **paused until the next SKILL language and contract model is reviewed and frozen**.

This freezes the order of work, not the vNext SKILL syntax and not the unfinished Schema Engine migration design.

The current v0.2 SKILL language and Compiler behavior remain authoritative. No proposed vNext syntax, Binding Plan, Schema identity, or dependency model is considered supported by production Compiler or Runtime until its own migration is approved and implemented.

The isolated Schema Engine result is retained as technical evidence. The candidate decision remains **CONDITIONAL ADOPT** for San v6.0.3 with the tested regexp2 v1.11.0 configuration. It is not yet a production dependency decision.

## 2. Why the pause is required

The current ca-skills vNext proposal is explicitly marked DESIGN-REVIEW (syntax not frozen). Its semantic direction is useful, but multiple compiler-facing decisions remain open:

- Predicate input declarations and the explicit scheduling point for READ acquisition;
- Procedure Context Projection shape and location;
- placement and precedence of exceptional explicit bindings (with/from);
- owner-governed Semantic Field Identity vocabulary and annotation syntax;
- optional/required input and guarded/conditional source semantics;
- Capability-to-Operation membership and authorization expansion;
- Evidence vocabulary, provenance, scope, correlation, freshness, and temporal fields;
- normalized projection schemas and lineage;
- deployment binding overlays and their versioning;
- canonical Procedure file placement.

These decisions affect the Compiler's parse model, source map, Schema registry, binding analysis, linked IR, and BuildRequest. Migrating the current v0.1 Schema implementation before they settle would create avoidable rework and risk encoding the old lib/** and Observation-ref model into the new Compiler.

## 3. Additional implementation blockers

The language design cannot be frozen from illustrative YAML alone. It must be reconciled with the actual executable 5GC source and owner contracts.

The current cross-repository audit identifies these gates:

1. The only executable acceptance baseline is conventional establish-pdu-session in ca-agent-demo. MAS remains design-only and must not be used to claim runtime or owner-contract completeness.
2. Host Subject, Facts, Evidence, and Observation-facing exports are not yet one complete, merged, versioned dependency set. ca-agent-demo PR #198 and #200 are proposals, not current mainline owner contracts.
3. The 5GC Observation inventory identifies 17 refs, but mapping a source class does not prove a formal READ Operation, output Schema, evidence guarantee, or equivalent acquisition point for each fact.
4. The MAS shared snapshot producer/owner and its governed READ contract remain unresolved.
5. Fabric pending/claim wire contracts and some normalized Provider/Host projections still have source/schema mismatches or missing owner publications.
6. Compiler BuildRequest and dependency snapshot identity must align with Skill Studio's Plugin-managed Configuration contract. Skill YAML must not acquire physical lib/ paths or a second dependency manifest.

Do not fabricate Owner schemas in a Skill, infer contracts from runtime implementation details, or turn a current Observation Source into a READ Operation merely to fit the new model.

## 4. Binding Resolver remains experimental

Compiler PR #7 is a feasibility spike, not accepted vNext behavior.

Its proposed x-ca-semanticId annotations are not yet published by the authoritative contract owners. Its pinned-fixture tests are not fully verified, its availability facts are supplied as test preconditions rather than derived from the real Procedure CFG, and its compiler-package call-site prototype has not been verified by a full test run.

Keep PR #7 Draft and isolated. Do not merge its compiler-package changes or migrate existing with/from mappings until:
- the semantic identity vocabulary and ownership are frozen in ca-skills;
- the actual 5GC owner schemas are available;
- the Compiler proves uniqueness, type/nullability, and definite availability on real CFG paths;
- old and generated Binding Plans are compared for behavior and evidence equivalence.

The existing Compiler checkBindings remains authoritative. No second weaker verifier is accepted.

## 5. Schema Engine evidence is preserved

PR #8 remains an isolated Spike. Its reported evidence is retained:

- 25/25 shared semantic vectors and 8/8 reference-security vectors;
- Native and Node WASM validation;
- three Chromium Worker runs with the same 25/25 and 8/8 results;
- pinned Draft 2020-12 Required suite: 1,319 PASS, 0 FAIL, 475 SKIP;
- Optional suite limitations remain explicitly documented.

The production candidate remains CONDITIONAL ADOPT. Format assertion stays disabled. Regex timeout/ReDoS hardening remains deferred to Issue #9 for the current trusted Demo input scope; it is not a v0.1 migration blocker.

Do not import tools/schema-engine-spike/ into the production Compiler. Do not add San or regexp2 to the production go.mod during this pause.

## 6. Work authorization during the pause

Allowed:
- review and refine ca-skills vNext language/semantic contracts;
- complete source/owner inventories and obtain owner-published schemas;
- align the Compiler BuildRequest input contract with Plugin-managed dependency snapshots;
- keep PR #8 as isolated technical evidence and complete its independent review;
- perform read-only design audits and update design documents;
- keep PR #6 Draft and revise it after vNext language freeze.

Not authorized:
- production Schema Engine Adapter/Registry implementation;
- production Schema validator or reference resolver replacement;
- changes to production go.mod / go.sum for San or regexp2;
- rewriting bindings.go or enabling automatic binding in production;
- migrating current Skill YAML or removing Observation Source behavior;
- changing current v0.2 Compiler/Runtime semantics;
- merging PR #5, #6, #7, or #8 without separate explicit approval.

## 7. Exit gates before production Compiler migration

All of the following must be satisfied:

### A. Language and semantic contract freeze in ca-skills
- One reviewed canonical Skill Language specification defines Capability, Operation, Tool, Procedure, Predicate, Fact, Observation, Evidence, and Binding Plan.
- YAML shape, requiredness, explicit override precedence, Context Projection, and Predicate input/acquisition scheduling are unambiguous.
- Semantic Field Identity is owner-controlled, versioned, and has duplicate/missing/incompatible identity rules.
- Capability membership and exact Operation authorization semantics are explicit.
- Source examples and schemas are consistent with the frozen language contract.

### B. Real 5GC owner contract completeness
- The conventional 5GC Skill remains the only executable acceptance baseline.
- Required Host Context and Evidence schemas are owner-published and merged.
- Every migrated Observation/READ dependency has an exact owner, schema, acquisition point, scope/correlation, temporal/freshness, and UNKNOWN behavior.
- Provider and Fabric schema mismatches are resolved by their owners.
- No Skill-owned shadow copy of external contracts is introduced.

### C. Studio / Compiler input alignment
- Plugin-managed Workspace Configuration owns dependency resolution, lock, and immutable snapshots.
- Compiler receives a versioned deterministic BuildRequest containing resolved snapshots and verifies their identity/integrity.
- Skill source contains logical exported contract identities, not dependency aliases, repositories, digests, or physical paths.
- No mandatory skill-project.yaml, skill.lock.yaml, physical lib/, or output/ layout is introduced contrary to the Studio contract.

### D. Compiler migration design re-review
After A–C, re-review PR #6 and resolve its outstanding findings:
- occurrence-aware YAML Anchor/Alias source mapping;
- explicit supported Draft 2020-12 Compiler Schema profile;
- explicit reference ownership compatibility policy;
- machine-readable ToolCatalog/Capability maturity and the MAS NOT_READY boundary;
- replacement of obsolete lib/** path assumptions with owner-controlled Schema IDs and resolved snapshots;
- exact Binding Plan and BuildRequest boundaries.

Only then create a production migration branch from the then-current, explicitly confirmed PR #5 implementation checkpoint. Do not assume today's checkpoint is still current at that future date.

## 8. Sequencing

1. Finish and freeze ca-skills vNext semantics and syntax.
2. Complete the actual 5GC Owner contract bundle and source-backed migration mapping.
3. Align Plugin-managed dependency snapshots with the Compiler BuildRequest contract.
4. Re-review and revise the Compiler migration design (PR #6).
5. Reassess the Binding Resolver prototype (PR #7) against the frozen owner contracts and real CFG.
6. Start Schema Engine production migration using the preserved PR #8 evidence.
7. Run full Compiler, Studio/LSP, Native, WASM, Chromium, and real 5GC regression acceptance.

## 9. Review outcome

The sequencing decision is approved and frozen.

The vNext language, PR #6 migration design, PR #7 Binding Resolver, and production Schema migration are **not** frozen or approved by this record. Existing v0.2 source and runtime behavior remain authoritative until a separately reviewed migration is accepted.
