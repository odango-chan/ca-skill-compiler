# Procedure / Binding / Library Link Contract v0.1

**Status:** Compiler implementation notes, subordinate to the normative [ca-skills Skill Language v0.2](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/skill-language-spec.md). This document defines linker behavior only; it must not become a second grammar authority.

Read together with:

- [Script Contract v0](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/script-contract.md)
- [Schema Ownership](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/schema-ownership.md)
- [Compiler Contract v0.1](compiler-contract-v0.1.md)

## 1. Action positions and scopes

The parser must identify action positions by AST structure and source path, never by recursively scanning every YAML key.

Main Procedure nodes support exactly:

- call: scalar exact Operation ID.
- script: package-relative Skill Script path.
- procedure: local Procedure name or explicit {skill, procedure} identity.
- verify: Predicate ID.
- reconcile: exact original mutation node reference.
- terminal: declared terminal result.

Preconditions/postconditions contain ordered named steps and an explicit on route. A step supports only call, script or verify. A nested procedure or tool field is unsupported and must produce a source-positioned diagnostic.

with is a binding map. Each argument has exactly one of:

- {from: <source-ref>}
- {value: <JSON-literal>}

No implicit previous-node binding, string interpolation, dynamic selector or source-specific alias is permitted.

## 2. Authoritative source and schema namespaces

ca-skills is the only official Skill source input. ca-agent-demo is not a second project source and must not be fetched by Compiler. The official current source packages are the two 5GC Skills in ca-skills; other historical Skill directories are not current grammar authorities.

Schema ownership is fixed:

| Reference namespace | Schema authority |
|---|---|
| context.request.* | spec.inputs.requestSchema |
| context.subject.* | spec.context.schemas.subject |
| context.facts.* | spec.context.schemas.facts |
| context.observations.* | spec.context.schemas.observations |
| context.evidence.* | spec.context.schemas.evidence |
| input.* | Current Procedure inputs, whose from resolves to a Producer |
| <node>.output.* | Exact Tool/Script/Procedure Producer output Schema |
| <node>.preconditions.<step>.output.* | That exact node's named precondition step output |
| <node>.postconditions.<step>.output.* | That exact node's named postcondition step output |
| preconditions.<step>.output.* / postconditions.<step>.output.* | Current node's named step output in its local phase scope |

The spec.context.schemas mapping is optional only when the corresponding context namespace is unused. If a source references a namespace, its authoritative Owner Schema must be present in the BuildRequest under the logical lib/** path. Missing paths produce EXTERNAL_SCHEMA_MISSING.

spec.context.requiredFacts is a list of identifiers, not a field-level Schema. It cannot establish the shape of context.facts.*.

spec.inputs.requestSchema is the one public Request Schema authority. It may point to a Skill-owned schemas/** file or a real imported Owner Schema under lib/**; there is no implicit same-name search.

## 3. Procedure input and with binding rules

- Resolve every declared Procedure input, including inputs not consumed by a node.
- Procedure.inputs.<name>.from must resolve against the declared Producer Schema.
- Optional Procedure.inputs.<name>.type is an author assertion checked against the Producer; it cannot manufacture a missing Schema.
- Every with.<argument>.from resolves the exact source path and checks it against the destination Tool Operation, Procedure or Script projection.
- Every with.<argument>.value is a literal checked against the destination Schema.
- Unknown source fields, absent required arguments, incompatible types and unsupported Schema keywords fail explicitly.
- A broad type: object with no properties does not authorize arbitrary nested fields.
- Local $ref and document-local ToolCatalog anchors are resolved only from the supplied file set. Remote $ref is rejected.

## 4. Script input projection and Output Schema

Procedure Script input is derived separately for each invocation from that step's with map. It is not a global Script Input Schema and must not be written into author source.

Predicate Script input is the declared Predicate Evaluation Context:

- subject
- request
- facts
- observations keyed by declared Observation aliases
- evidence

The context is linked to the real Owner Schemas and the Skill Request Schema. Compiler must not infer it from Script body access, requiredFacts or Case fixtures.

A Script uses exactly main(input: dict) -> OutputTypedDict. The input dictionary is only the invocation container. Its output type is a non-empty statically declared TypedDict in the same file; functional TypedDict is supported for non-identifier wire keys. Compiler generates the read-only Output JSON Schema and never runs Python. No legacy metadata, independent local Tool ID or Script Input Schema fallback exists.

## 5. Reconcile and dependency closure

- reconcile must identify the exact original mutation node; do not treat it as an ordinary new Tool call.
- Reconcile output references use the original ToolCatalog's declared reconcile Output Schema.
- A cross-Skill Procedure reference is explicit {skill, procedure}; local names resolve only in the current Skill.
- Missing Skill/Procedure, ambiguous Operation ID and dependency cycles are errors.
- targets omitted/empty selects all Skills; explicit targets select their transitive Skill dependency closure.
- Build packages only the selected Skill sources and imported Tool/Schema files actually required by their closure.

## 6. Known current-source blockers

The official ca-skills v0.2 source intentionally exposes unresolved external ownership rather than inventing Schemas:

- Host Subject/Facts/Evidence Owner Schemas are not bundled.
- Observation payload Owner Schema is not bundled.
- sbi.udm.sm_data.get currently exposes result as a broad object without nested properties, while the conventional selection Script reads nested subscription/provenance fields. The Tool Owner must provide the complete Output Schema.
- MAS references 16 ca.pdu_session.* Operations for which no approved MAS ToolCatalog is supplied.

Compiler must report precise missing/unsupported producer-schema and Operation diagnostics. Do not restore deleted Skill-owned Script Input Schemas or fabricate Owner contracts to force Build PASS.

## 7. Minimum diagnostic behavior

At minimum, diagnostics must distinguish:

- malformed/unsupported Procedure action;
- missing Skill, Procedure, node target or Script;
- unknown/duplicate Operation ID;
- missing external Owner Schema;
- unknown Producer field;
- unresolved binding;
- incompatible binding type;
- unsupported binding/Schema feature;
- unknown named step scope;
- missing Predicate.

Every diagnostic includes severity, stable code, source path, line/column when available and a useful message. Ordering must be deterministic.
