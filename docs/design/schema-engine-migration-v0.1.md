# Schema Engine Migration v0.1

**Status:** Proposed implementation plan; design review required. No production migration is authorized by this document.
**Date:** 2026-11-12
**Repository:** odango-chan/ca-skill-compiler
**Production baseline:** PR #5 checkpoint 49753ce345f8cf99654566995dd3b5aff070f844
**Architecture basis:** PR #6 commit 751e018d4b34c2f7cdbedc625f3a3f7843180c01
**Spike evidence:** PR #8 commit 3558a2a2c965a6c063fea0d9fdcea3ae0c887b1f

This document turns the completed isolated Spike into a concrete, staged Compiler migration plan. It does not authorize implementation until this plan and PR #8 have been reviewed.

## 1. Decision

Carry the following candidate into production migration planning:

- github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
- github.com/dlclark/regexp2 v1.11.0, configured with the exact ECMAScript/Unicode mode and the two narrowly tested Unicode property-name normalizations from the Spike.

**Decision: CONDITIONAL ADOPT.**

Evidence from PR #8:
- 25/25 shared semantic vectors and 8/8 reference-security vectors pass.
- Native and Node WASM pass.
- Three actual Chromium Worker runs pass the same semantic and security vectors.
- Pinned Draft 2020-12 Required suite: 1,319 pass, 0 fail, 475 skipped with the custom regex configuration.
- Optional suite failures remain documented: 6 regex compilation failures and 131 format/optional validation failures.
- The stripped Spike WASM harness is 6.70 MB, +2.29 MB over its no-engine harness. This is not the production Compiler size delta.
- YAML source mapping, stable Compiler diagnostic mapping, actual production footprint, and final production dependency review remain migration work.

This is not a claim of complete ECMA-262 or all optional Format-Assertion conformance. The current Demo uses the default Draft 2020-12 Format annotation behavior; format assertion remains disabled.

## 2. Product assumptions and deferred hardening

The current Demo's Skill and ToolCatalog Schema files are trusted, project-controlled project assets. This migration does not add adversarial regular-expression hardening, a regex timeout policy, or multi-tenant isolation work.

Issue #9 tracks bounded regex execution as a future enhancement. Revisit it before accepting untrusted/external Schema patterns or moving to a multi-tenant execution model. It is not a blocker for this Demo migration.

The following remain required because they are core Compiler behavior and deterministic-build requirements, not extra threat-model hardening:
- references resolve only from the caller-supplied in-memory BuildRequest file set;
- no runtime filesystem or network loader is installed;
- project ownership and path boundaries are enforced;
- YAML aliases and non-JSON YAML values are rejected safely;
- exact JSON number semantics are preserved.

## 3. Scope and non-goals

### In scope

- Replace hand-written standard Draft 2020-12 Schema validation with San behind a Compiler-owned adapter.
- Replace eager standard $ref expansion/validation with San reference compilation backed by a closed project resource registry.
- Route concrete JSON literal validation through the same standard Schema engine.
- Preserve the existing conservative Producer/Consumer implication proof.
- Validate ToolCatalog, Skill-owned Schema documents, and generated Script Output Schemas through one adapter.
- Preserve BuildRequest, BuildResult, LibraryResult, CLI, and WASM public contracts.
- Preserve YAML authoring and generated artifact paths/format.

### Explicitly out of scope

- General Producer/Consumer subsumption or theorem proving.
- Expanding the current binding proof subset.
- Changing Skill Language, ToolCatalog ownership, or Operation maturity.
- Fetching ToolCatalogs or Schemas from Git/network at build time.
- Executing Python or Tools.
- Changing the Compiler Go baseline from 1.23.
- Adding a package manager, lockfile, remote registry, or dependency lifecycle.
- Regex timeout/ReDoS hardening (tracked separately in Issue #9).
- Full Format-Assertion vocabulary support.

## 4. Target package and data model

### 4.1 Dependency boundary

Add a small internal package, preferably internal/schemaengine, which contains all direct imports and types from San and regexp2. The internal/compiler package must not expose either library's types through public API structs or function signatures.

The Compiler owns the project registry and supplies a loader implementation to the engine package. The engine package owns:
- Draft selection and San compiler configuration;
- custom regex engine configuration;
- Schema compilation;
- concrete instance validation;
- conversion of library errors into a compiler-neutral structured error.

The Compiler owns:
- YAML parsing and project source identity;
- exact YAML-to-JSON conversion and source locations;
- BuildRequest membership and ownership policy;
- reference-to-project-file mapping;
- stable diagnostic codes and messages;
- binding projection and Producer/Consumer proof.

### 4.2 Compiler-owned source document

Introduce a per-analysis immutable document representation conceptually equivalent to:

~~~go
type schemaDocument struct {
    Path          string
    CanonicalURI  string
    Root          *yaml.Node
    Value         any
    SourceMap     map[string]SourceSpan // JSON Pointer -> YAML source span
    PointerByNode map[*yaml.Node]string
}
~~~

A Schema embedded in a ToolCatalog is not a separate YAML file. Its location identifies the containing document plus a JSON Pointer, for example:

- document: lib/tools/createdsession.tool.yaml
- pointer: /spec/contracts/0/inputSchema

A Skill-owned Schema file normally uses the empty pointer for its document root. An x-common fragment uses a pointer into the ToolCatalog document.

The actual Go representation may differ, but it must preserve original YAML nodes, canonical document identity, JSON-compatible values using json.Number for all numbers, JSON Pointer to YAML line/column mapping, and mapping from YAML node identity back to its document and JSON Pointer.

Do not use generic YAML Decode into any for the engine path if it can introduce float64 or otherwise alter exact numeric meaning.

### 4.3 Schema Engine Adapter API

Keep the adapter narrow and Compiler-neutral. Its logical API must support:
- registering/compiling a source document using a supplied resource loader;
- compiling a Schema at a document JSON Pointer (not only at document root);
- validating an instance against a compiled root or subschema;
- returning structured errors containing schema/resource URI, keyword location, instance location, and nested causes where available.

The San v6 API supports compiling a URI fragment. Use that capability for embedded ToolCatalog Schemas and binding literal validation against a property Schema. Prove this in adapter tests before depending on it.

Conceptual API:

~~~go
type ResourceLoader interface {
    Load(absoluteURI string) (Document, error)
}

type Engine interface {
    CompileAt(document Document, pointer string, loader ResourceLoader) (CompiledSchema, error)
}

type CompiledSchema interface {
    Validate(instance any) error
}
~~~

All San/regexp2 types remain private to internal/schemaengine.

## 5. YAML conversion and source mapping

Use the existing yaml.v3 parser and its limits as the single authoring boundary. Extend/refactor conversion rather than introducing a second YAML parser.

The converter must:
1. Preserve source filename, line, and column for every value node.
2. Build a deterministic JSON Pointer source map.
3. Convert integer and decimal scalars to exact json.Number values without float64.
4. Reject duplicate mapping keys, non-string JSON object keys, unsupported/custom YAML tags, timestamps, NaN/Infinity, malformed numbers, unresolved aliases, and alias cycles.
5. Preserve the current bounded YAML depth/node/alias rules.
6. Keep YAML aliases separate from JSON Schema references.

JSON Pointer generation must escape ~ as ~0 and / as ~1; array positions use numeric path segments. Mapping key order may be retained for source mapping, but serialized diagnostics and outputs must remain deterministic.

For an engine error, map its absolute resource URI plus keyword JSON Pointer to the originating YAML document's SourceMap. If no exact location can be mapped, fall back to the nearest known Schema root and retain the stable Compiler diagnostic code. Do not expose raw library error strings as a public contract.

## 6. Project Schema Registry and reference behavior

### 6.1 Canonical resources

Assign each project file a stable internal hierarchical URI, for example:

~~~text
ca-compiler://project/skills/establish-pdu-session/schemas/request.schema.yaml
ca-compiler://project/lib/tools/createdsession.tool.yaml
~~~

The URI must be generated from a validated project-relative path, not from arbitrary user text.

The registry is an immutable snapshot of the current BuildRequest file set. It may return only documents present in that snapshot and allowed by the ownership policy.

### 6.2 Ownership policy

Preserve the current intended boundaries and make them explicit:

- A Library-owned Schema may reference only resources under lib/.
- A Skill-owned Schema may reference resources in the same Skill and shared Library resources under lib/.
- A Skill may not reference another Skill's private Schema.
- A reference target must be present in BuildRequest and parse as a supported project document.
- Unknown URI schemes, network retrieval, filesystem retrieval, absolute filesystem paths, and paths escaping the permitted project boundary are denied.
- Relative references are allowed when their canonical target remains inside the permitted owner boundary. A ../ segment is not rejected solely by spelling; the normalized destination and owner are what matter.

Absolute $id values are identifiers, not permission to access the network. They may be registered as aliases only when explicitly declared by an in-memory project resource. A $ref to an absolute URI that is not a registered project alias is denied without external access.

The registry's loader must be the only loader installed in San. No default HTTP, HTTPS, file, or filesystem fallback may be reachable.

### 6.3 Reference semantics versus binding projection

Do not continue using expandSchemaRefs as the standard Schema evaluator.

San owns Draft 2020-12 reference semantics, including:
- $ref sibling behavior;
- relative base URI handling;
- recursive references supported by the dialect;
- nested resources and anchors supported by the selected implementation.

A valid recursive JSON Schema reference is not an invalid cycle. Keep rejecting cyclic YAML aliases, but do not reject valid recursive JSON Schema merely because a resource graph is cyclic.

The Compiler still needs a read-only Schema view for its bounded static binding proof. Introduce a separate bindingSchemaView/projection step:
- resolve only the reference shapes needed by the current proof subset;
- preserve the source document and pointer for each projected node;
- mark unsupported constructs (for example, $ref siblings, dynamic/recursive cases, or unsupported applicators) as unproven for binding;
- do not turn a valid Schema into LIB_SCHEMA_INVALID solely because the binding prover cannot inspect it;
- emit BINDING_UNSUPPORTED only when a binding actually requires the unsupported proof.

Reference policy violations (missing project resource, denied owner crossing, path escape, unknown scheme) remain Schema/reference diagnostics, not binding-unsupported results.

### 6.4 Build output dependency tracking

The current analysis.schemaRefs graph feeds addSchemaDependencies and determines which referenced Library files are copied into BuildResult.

Preserve that behavior. While compiling a root Schema, the registry must record every project resource loaded for that root, including transitive resources. Record paths in schemaRefs[rootPath] (or an equivalent deterministic graph) so a Build never drops a referenced Library file.

Add regression tests for ToolCatalog references, Skill request/context Schema references, and transitive Library dependencies.

## 7. Draft and engine configuration

Production configuration must match the accepted Spike:

- Pin github.com/santhosh-tekuri/jsonschema/v6 v6.0.3.
- Pin github.com/dlclark/regexp2 v1.11.0.
- Use Draft 2020-12 as the default.
- If $schema is explicitly authored, accept only the Draft 2020-12 URI selected by the Compiler contract; reject other dialects rather than silently changing semantics.
- Keep Format as annotation by default; do not call AssertFormat().
- Configure regexp2.ECMAScript | regexp2.Unicode.
- Preserve only the Spike-tested \p{Letter}/\P{Letter} alias normalization; do not build a general regex translation layer.
- Do not set up a second or fallback regex engine for a given Schema.
- Keep the current Demo trust assumption and defer regex timeout hardening to Issue #9.

Optional format assertion failures and unsupported optional regex constructs remain documented limitations, not reasons to implement a second Schema engine.

## 8. Binding Analyzer: what stays Compiler-owned

The following responsibilities remain in internal/compiler/bindings.go:
- Procedure node/step Producer discovery;
- Procedure inputs and aliases;
- Scope and source availability;
- node-output dominance;
- source path resolution;
- projection through object properties;
- the bounded Producer ⊆ Consumer proof;
- exact rational comparisons for numeric implication;
- explicit outcomes: proven compatible, proven incompatible, and unproven.

Keep the current proof subset and its positive/negative golden tests. Do not add arbitrary allOf/anyOf/oneOf implication, recursive implication, regex-language inclusion, or a general solver.

Concrete literal validation is standard JSON Schema instance validation and should use the adapter. This includes:
- with.value checked against a consumer property;
- finite const/enum values checked against source/target schemas during the existing bounded proof.

Refactor the finite-value proof path to call the adapter's compiled subschema validator instead of extending literalMatchesSchema as a second JSON Schema instance evaluator. Keep exact rational comparison helpers where they are part of Producer/Consumer constraint implication, not concrete instance validation.

If a Schema is valid but cannot be projected into the proof subset, preserve the Schema and return BINDING_UNSUPPORTED when a binding depends on that proof. Never treat missing projection data as “no consumer Schema” or silently pass.

## 9. File-by-file migration disposition

| Current file | Migration |
|---|---|
| internal/compiler/schema_validation.go | Remove standard keyword shape/type/constraint validation after cutover. Replace with a small Compiler Schema profile policy if needed (allowed keywords, Draft pin, project-specific root restrictions). Do not retain duplicate standard semantics. |
| internal/compiler/schema_refs.go | Replace eager expansion as a validator with Project Schema Registry policy and a separate binding-only projection. Preserve ownership and dependency tracking. |
| internal/compiler/bindings.go | Preserve Producer/Consumer and data-flow proof. Route concrete literal checks through adapter. Add explicit projection status/source location metadata. |
| internal/compiler/library.go | Preserve ToolCatalog envelope, caller ownership, Operation maturity and object-root rules. Compile each input/output/reconcile Schema at its document pointer; validate each x-common fragment as an allowed Schema resource. |
| internal/compiler/skills.go | Preserve SkillContract path/owner rules. Compile requestSchema and context Owner Schemas through the same adapter; cache per analysis. |
| internal/compiler/scripts.go | Preserve static Python parsing and generated artifact bytes/path. Validate generated Script Output Schema through the same adapter before emitting it. Never execute Python. |
| internal/compiler/yaml.go | Preserve parser, duplicate-key and size/depth limits. Share the new exact converter/source map; do not create another parser. |
| internal/compiler/api.go | Keep BuildRequest, BuildResult, LibraryResult, Check, Build, and InspectLibraries public contracts unchanged. Add only per-analysis caches/registry state internally. |
| docs/workstreams/json-schema-support-v0.1.md | Update the matrix to distinguish engine validation, project reference policy, and binding proof. |
| go.mod / go.sum | Add only the pinned San and regexp2 dependencies (plus unavoidable transitive dependencies); keep Go 1.23.0. |

The existing LibraryTool JSON representation and generated Script Schema artifact format must remain stable. Add serialized golden tests before changing any value conversion used by public result types.

## 10. Implementation work packages

### WP0 — Baseline and design freeze

- Review PR #8 and this migration plan.
- Confirm the exact PR #5 base checkpoint 49753ce3.
- Capture current Check, Build, and InspectLibraries golden diagnostics, output paths, output bytes, and actual production Native/WASM artifact sizes.
- Inventory all calls to validateSchemaShape, expandSchemaRefs, literalMatchesSchema, schemaProperty, and schemasCompatible.
- No production changes in WP0.

### WP1 — Adapter and lossless YAML/source-map layer

- Add the internal Schema Engine adapter with pinned dependencies.
- Implement exact YAML-to-JSON conversion and JSON Pointer source maps.
- Prove fragment compilation and literal validation at a nested ToolCatalog Schema pointer.
- Implement structured error normalization and YAML line/column mapping.
- Add Native and WASM unit tests.
- Do not change existing production call sites yet; no double active validator.

### WP2 — Project Schema Registry

- Implement canonical internal URIs and immutable BuildRequest-backed resources.
- Enforce owner/path rules and no external loader.
- Support registered $id aliases and relative references.
- Preserve valid Draft 2020-12 recursive references.
- Record all loaded dependencies for Build output packaging.
- Add positive/negative reference and transitive dependency tests.
- Keep the current call sites on the old path until WP3 cutover.

### WP3 — Schema document validation cutover

Route all standard Schema document compilation through the adapter:
- ToolCatalog operation input/output/reconcile Schemas;
- ToolCatalog x-common fragments;
- Skill requestSchema and context Owner Schemas;
- generated Script Output Schemas.

Preserve ToolCatalog envelope validation, object-root restrictions, Skill path ownership, allowed-keyword policy, and diagnostic codes. Keep Format assertion disabled.

At the end of WP3, no production call site should use validateSchemaShape as its standard JSON Schema validator.

### WP4 — Binding and literal integration

- Preserve the current data-flow and bounded implication implementation.
- Replace literalMatchesSchema concrete validation with adapter validation at a compiled Schema pointer.
- Preserve finite const/enum proof behavior through adapter validation.
- Introduce binding-only Schema projection with explicit supported/unproven state.
- Ensure unsupported projection cannot silently skip a consumer check.
- Keep mismatch versus unproven diagnostics distinct.

### WP5 — Remove legacy standard evaluator

- Delete superseded standard keyword validation from schema_validation.go.
- Delete eager standard $ref expansion from schema_refs.go.
- Retain only project registry, policy, source mapping, YAML boundary, and binding projection code.
- Remove duplicate standard literal-validation logic.
- Update the Schema support matrix and architecture docs.

### WP6 — Full regression and browser acceptance

Run and report:
- Go 1.23 Native tests and go vet;
- Native CLI build;
- GOOS=js GOARCH=wasm build;
- Node WASM tests;
- actual Chromium Worker tests;
- pinned official Required suite with the accepted custom regex configuration;
- current 12-operation approved ToolCatalog and all six x-common fragments;
- 5GC Skill Compiler Check/Build;
- MAS Skill static Compiler Check/Build only (do not claim runtime 5GC/MAS execution);
- Script Output generated Schema golden bytes;
- positive, mismatch, unsupported, and missing-source binding goldens;
- no-network/no-filesystem reference tests;
- deterministic diagnostics/output ordering and bytes.

Measure the actual production Compiler Native/WASM size and browser initialization delta against the WP0 baseline. The Spike harness size is reference evidence only. Do not invent a size budget; report the actual delta and pause for review if it materially harms the online Demo experience.

## 11. PR and branch strategy

- PR #8 remains an isolated Spike and is not merged as part of migration.
- PR #6 remains the design record; merge/review decisions are separate.
- Create a new migration branch from PR #5 head 49753ce345f8cf99654566995dd3b5aff070f844, not from main and not from PR #8.
- Target the PR #5 head branch feat/compiler-v01-implementation-20261111 so the migration is a clean stacked PR.
- Do not alter PR #5's existing commits or the two preserved uncommitted Compiler-worktree documentation changes.
- Use the Spike report as evidence, but do not import tools/schema-engine-spike as a production package. Implement the production adapter in its intended internal package and keep all candidate-specific dependencies behind it.
- Prefer separate PRs for adapter/registry foundation, cutover/binding integration, and legacy cleanup if the diff becomes difficult to review. Each PR must be independently testable.
- Do not merge PR #5, PR #6, or PR #8 automatically.

## 12. Acceptance and stop conditions

Migration is complete only when:
- public Compiler APIs and YAML authoring remain unchanged;
- all standard Schema validation call sites use San through the adapter;
- all project reference access is closed and in-memory;
- source diagnostics map to YAML path/line/column;
- concrete literal validation uses the engine;
- Producer/Consumer proof remains Compiler-owned, bounded, and fail-closed;
- generated Script Output Schema remains deterministic;
- Native/WASM/Chromium behavior and Build outputs match the accepted goldens;
- production size/performance delta is reported and accepted;
- the old duplicate standard Schema evaluator is removed.

Stop after the migration PR(s) are ready for review. Do not merge or change main without explicit authorization.

**Design decision:** CONDITIONAL ADOPT San v6.0.3 with the tested regexp2 configuration for the trusted Demo input scope. Issue #9 is deferred hardening, not a v0.1 migration blocker. No production implementation has been authorized by this document.
