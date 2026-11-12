# Semantic Binding Resolver — Compiler Integration Design

Status: **SPIKE DESIGN / NOT IMPLEMENTED IN COMPILER PIPELINE**  
Base: Compiler v0.1 implementation branch (feat/compiler-v01-implementation-20261111)  
Related: [ca-skills Semantic Field Identity vNext](https://github.com/odango-chan/ca-skills/blob/design/capability-authoring-model-vnext/docs/design/semantic-field-identity-vnext.md), [isolated PR #7](https://github.com/odango-chan/ca-skill-compiler/pull/7).

## 1. Finding: do not build a second type or CFG checker

Inspection of current Compiler source shows that v0.1 already has the relevant proof machinery:

- analysis.procedureInputSchemas and prepareProcedureInputSchemas resolve Procedure input aliases to authoritative source schemas.
- procedureProducerSchemas resolves node and named step output schemas.
- resolveSource resolves explicit dotted source paths against Request, Context, Procedure-input and Operation-output schemas.
- schemasCompatible and its conservative Draft 2020-12 subset check Producer-to-Consumer compatibility.
- bindingSourceAvailable / nodeOutputDominates reject producers not available on every relevant control-flow path.
- checkBindings combines source resolution, availability, schema compatibility, literals, required inputs and diagnostics.

The isolated package's Type and AvailableAt checks are feasibility scaffolding only. They must not become a second production checker.

## 2. Correct production shape

The Semantic Resolver should be a **candidate generator**, not a second validator:

1. Read semantic IDs from already linked, owner-controlled Producer and Consumer Schema property nodes.
2. Consider only sources in the current explicit scope (first slice: resolved top-level Procedure inputs).
3. For each missing required Operation input, find candidates with exactly the same semantic ID.
4. If no candidate exists, leave the input unbound; the existing checkBindings reports BINDING_UNRESOLVED.
5. If more than one candidate exists, emit BINDING_AMBIGUOUS; do not choose by name/order.
6. If exactly one candidate exists, construct an in-memory candidate binding {from: input.<alias>}.
7. Merge it with authored with without overwriting any explicit mapping.
8. Pass the merged map through the existing checkBindings path. It independently resolves the source, checks CFG availability, and proves Schema compatibility. If any proof fails, the normal compiler diagnostic blocks the build.

No auto-generated mapping is written back into the source Procedure. The source remains concise; the compiled representation/Binding Plan is a separate future output decision.

## 3. First slice and explicit non-goals

The first safe slice is only **Procedure input → Operation input** for missing required inputs. This directly targets the 11 shared MAS context inputs.

Not included:
- Operation output → Operation input (requires exact producer instance/generation and CFG reasoning).
- Observation acquisition or Evidence reuse.
- Nested Procedure argument inference.
- Script input inference.
- Optional consumer inputs (do not change behavior by opportunistically injecting values).
- Literal defaults, coercions, transforms, name-based fallback, or implicit scope widening.
- Writing generated with blocks into authored YAML.

## 4. Semantic annotation compatibility

Current validateSchemaShape accepts vendor extension keywords beginning x-, but current supportedStaticSchema uses a narrower allowlist. Before a semantic annotation can participate in compatibility checks, the Compiler must explicitly admit **only** x-ca-semanticId as non-validating metadata in that static checker, while continuing to reject unknown constraints/keywords.

The semantic ID parser must validate:
- nonempty, namespaced identifier syntax;
- scalar string value only;
- exactly one annotation per Schema object;
- no interpretation as a JSON value constraint;
- identity copied through local $ref expansion without changing the source schema.

This change does not turn arbitrary x-* metadata into trusted semantics.

## 5. Source contract prerequisites

The MAS source fixture currently has:
- 11 Procedure inputs with from paths;
- 16 call nodes and 176 corresponding explicit context with/from mappings;
- a MAS ToolCatalog explicitly marked NOT_READY.

Its context request Schema is present. The checked-in MAS SkillContract fixture currently does not declare context.schemas.subject/facts, even though Procedure inputs reference context.subject.ueRef and context.facts.*. Those authoritative context Schema files/links must be added to a migration fixture before the real Compiler can resolve those paths. Do not silently invent a built-in Host schema.

The traditional 5GC Procedure also references Context paths whose authoritative Owner Schemas are not present in this Compiler fixture. Its four UDM Operation input schemas can be checked as a focused explicit mapping comparison, but end-to-end semantic auto-binding needs those owner Schemas imported and declared.

The 5GC Operation catalog is an existing contract source. The MAS ToolCatalog remains proposed/not-ready and must not be used as proof of a deployable Operation owner contract.

## 6. Test design for production integration

Add compiler-package tests (not only bindingspike) that exercise the real analysis pipeline with test-only owner annotations and complete context-schema fixtures:

- all 11 MAS fields receive owner-controlled semantic IDs on Context/Request Producer schemas and required Operation Consumer schemas;
- generated candidate bindings equal the 176 existing explicit paths, byte-for-byte at the source-reference level;
- source Procedure bytes remain unchanged;
- same ID + two Procedure inputs => BINDING_AMBIGUOUS;
- same field name + different semantic IDs => unresolved, never bound;
- same ID + incompatible schemas => existing BINDING_TYPE_MISMATCH;
- a producer output from a bypassable branch is rejected by existing bindingSourceAvailable;
- explicit authored with always wins and is validated unchanged;
- missing or invalid semantic IDs do not infer anything;
- deterministic repeated builds produce identical Binding Plan output/diagnostics.

Only after these pass should the Compiler expose a stable Binding Plan in its build output contract. Until that output shape is approved, keep generated bindings as internal candidates used by validation only.

## 7. Acceptance sequence

1. Close the Context Owner Schema gap in isolated test fixtures; do not fabricate production schemas.
2. Implement semantic-ID extraction and candidate generation in a new compiler source file.
3. Add the narrow x-ca-semanticId exception to static Schema keyword handling.
4. Feed candidates through existing checkBindings; do not duplicate type/CFG logic.
5. Run all Compiler Go tests and actual fixture builds.
6. Compare generated candidate plan with existing authored mappings for both Skills.
7. Review the BuildResult/IR contract separately before any source YAML migration.

No production integration, syntax freeze, or PR merge is implied by this design note.