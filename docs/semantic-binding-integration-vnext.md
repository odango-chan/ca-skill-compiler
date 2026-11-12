# Semantic Binding Resolver — Compiler Integration Design

Status: **SPIKE IMPLEMENTATION COMMITTED / EXECUTION NOT VERIFIED**  
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

The **Compiler PR #5 demo copy is stale**: its MAS and 5GC SkillContract copies omit context.schemas even though their Procedure inputs reference context.subject/facts. The current official ca-skills main contracts do declare these logical Owner paths:
- both Skills: lib/host/subject.schema.yaml and lib/host/facts.schema.yaml;
- both Skills: lib/observations/context-observations.schema.yaml;
- conventional 5GC additionally: lib/host/evidence.schema.yaml.

However, ca-skills main's current official lib snapshot does **not** contain these Host/Observation Owner Schema files. docs/schema-ownership.md explicitly says these are externally owned dependencies and must be supplied by the Host/Observation owners; they must not be fabricated by the Skill or Compiler. Thus a complete real Compiler build is currently blocked by missing external owner files, not merely by a test fixture oversight. The focused source-level Procedure/ToolCatalog comparison in real_fixture_test.go does not resolve or prove those Context paths.

The 5GC Operation catalog is an existing Tool Owner contract source. The MAS ToolCatalog remains proposed/not-ready and must not be used as proof of a deployable Operation owner contract.

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

Correction based on the current ca-skills main contracts and docs/schema-ownership.md: the logical Context Owner paths are already declared in the official SkillContracts, but the actual owner files are not present in the current shared lib snapshot. The older Compiler PR #5 demo copies are additionally stale because they omit those declarations. These are distinct issues; do not conflate them.


## 8. Current spike implementation (2026-11-12)

The isolated branch now contains a narrow compiler-package candidate generator:

- `internal/compiler/semantic_bindings.go` reads explicit `x-ca-semanticId` values from the already resolved Procedure input Producer Schemas and Operation Consumer Schema properties.
- `internal/compiler/bindings.go` invokes it only for Operation call sites (main node and named pre/postcondition calls), then passes the in-memory merged `with` map into the existing `checkBindings`.
- The resolver considers missing **required** inputs only, never overwrites explicit authored mappings, preserves malformed explicit `with` for the existing diagnostic path, and emits `BINDING_AMBIGUOUS` when multiple Procedure inputs carry the same semantic ID.
- A unique semantic match is proposed even if its schemas are incompatible; the existing `checkBindings` must then reject it with the canonical `BINDING_TYPE_MISMATCH`. The candidate generator deliberately does not call `schemasCompatible` itself and has no independent CFG/type proof.
- `internal/compiler/semantic_bindings_test.go` covers ID parsing, unique candidate generation, ambiguity, same-name/different-ID, delegation of schema mismatch, and explicit mapping precedence.

The static binding-schema allowlist now admits exactly `x-ca-semanticId` as non-validating metadata; other unknown extensions remain unsupported by that conservative proof subset. General Schema shape validation already permits `x-*` annotations.

**Not yet verified:** no Go test execution or CI result was available for this follow-up compiler-package implementation. The environment could not resolve GitHub for a local clone, and this repository has no configured workflow run attached to the spike head. Treat this as committed prototype code awaiting execution, not PASS. Full pipeline tests still require owner-approved Context Schemas; see §5.


## 9. Executable acceptance baseline correction (2026-11-12)

The first real acceptance target is `ca-agent-demo`'s runnable `skills/establish-pdu-session` 5GC Skill. MAS is not currently end-to-end runnable; its ToolCatalog is explicitly proposed / NOT_READY. The earlier 11 fields × 16 calls = 176 mapping test is a synthetic shape/feasibility example only, not a source-backed MAS migration result and not an acceptance criterion.

Use [ca-skills 5GC Schema & Observation Source Inventory](https://github.com/odango-chan/ca-skills/blob/design/capability-authoring-model-vnext/docs/design/5gc-schema-source-inventory-vnext.md) to locate the actual 5GC owner authorities. The current Compiler PR #5 fixture is a copied demo project and may be stale; before using it for acceptance, refresh it from the exact current `ca-agent-demo` commit and bring in the real owner contracts without copying or inventing schemas. Do not migrate/remove 5GC `with` mappings until full compile and exact binding/evidence equivalence pass.
