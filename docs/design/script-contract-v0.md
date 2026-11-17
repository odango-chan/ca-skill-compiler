> **HISTORICAL — SUPERSEDED FOR NEW DEVELOPMENT.** This Script note targets the former Script Contract v0. The active contract is [Script Contract v1alpha4](script-contract-v1alpha4.md), subordinate to the frozen CA-SL v1alpha4 specification. Retained only as design history.

---

# Script Contract v0 · Compiler Implementation Contract

**Status:** Implement the current ca-skills Script Contract v0. The normative source document is [ca-skills/docs/script-contract.md](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/script-contract.md). This file records the corresponding Compiler obligations and is not a separate language authority.

## 1. Identity and entry point

A Script is a Python source file owned by one Skill at skills/<skill-name>/scripts/<relative-path>.py.

Its only identity is **Skill name + package-relative Script path**. No Python metadata may declare a second name, Tool ID, Operation ID, Capability ID or version.

The only supported entry point is:

~~~python
def main(input: dict) -> OutputType:
    ...
~~~

- Exactly one synchronous main.
- Exactly one positional parameter named input, annotated exactly dict.
- No extra positional/keyword parameters, *args, **kwargs or default values.
- The return annotation names a non-empty TypedDict declared statically in the same file.
- No old # /// connectionagent-tool metadata or fallback parser.

## 2. Input projections

Scripts do not author Input Schemas.

### Procedure Script

For each script: scripts/foo.py invocation, Compiler derives a distinct input projection from that step's with: bindings. Each from: resolves against the exact Producer Schema; each value: is checked against the destination type. The same Script can have different projections at different call sites.

Compiler must not treat input: dict as permission to access arbitrary fields. Missing Producers, unknown fields and incompatible types are errors. No editable Script Input Schema is created or generated.

### Predicate evaluator Script

For evaluator: {type: script, ref: scripts/foo.py}, the input is the Host-created Predicate Evaluation Context:

- subject: exact trusted Subject projection;
- request: Skill Request validated by spec.inputs.requestSchema;
- facts: Host-owned fact projection;
- observations: declared aliases mapped to materialized payloads;
- evidence: Host-owned provenance/evidence projection.

The Compiler links these fields using the explicit spec.context.schemas Owner references and Predicate observation declarations. It does not infer schemas from Python access patterns, requiredFacts or Case fixtures.

## 3. Output TypedDict and generated Schema

Compiler statically parses the return TypedDict and generates a read-only JSON Schema Draft 2020-12.

Supported declarations:

- str, int, float, bool;
- nested TypedDict declarations in the same file;
- functional TypedDict("Name", {...}) for wire keys that are not valid Python identifiers (for example 5qi);
- list[T];
- Literal[...];
- NotRequired[T];
- Annotated[T, {<literal JSON Schema constraints>}].

Supported literal constraints: string minLength, maxLength, pattern; numeric minimum, maximum; array minItems, maxItems, uniqueItems. Unknown constraints, dynamic expressions, unsupported annotations and invalid ranges fail explicitly.

Output rules:

- Output is non-empty and fully declared; generated additionalProperties is false.
- Required and NotRequired fields are preserved exactly.
- Literal maps to const or enum.
- dict, dict[str, T], Any, object, empty Output classes and broad/dynamic unions are forbidden.
- No field, type or constraint is inferred from the function body. Python is never executed.
- Generated output path removes the leading scripts/ and trailing .py, then places the remaining relative path under skills/<skill-name>/schemas/generated/ and appends .output.schema.json. Example: scripts/common/normalize.py becomes skills/<skill-name>/schemas/generated/common/normalize.output.schema.json in the BuildResult file map.
- The generated file is not written back into author source and is never accepted as a subsequent BuildRequest input.

If a Script only forwards an existing Producer value, prefer a direct Procedure binding rather than introducing a duplicate Output type.

## 4. No per-Script execution policy

Scripts are restricted to deterministic local computation. They must not access network/MCP/HTTP, shell/subprocess, filesystem, environment secrets, clocks, random sources, databases or credentials.

Host owns the fixed execution sandbox and timeout policy. A Script cannot widen it through source metadata. Compiler does not run Python and does not claim that a static TypedDict proves runtime determinism.

A Script is not an independently invocable Tool or Capability. There is no allowedResources.localTools.

## 5. Current official PDU examples

The five conventional PDU Scripts in ca-skills are the current migration examples:

| Script | Output |
|---|---|
| scripts/select_establishment_configuration.py | EstablishmentSelectionOutput |
| scripts/build_pcf_policy_inputs.py | PcfPolicyInputsOutput (functional TypedDict for 5qi) |
| scripts/build_n1n2_delivery.py | N1N2DeliveryOutput |
| scripts/evaluate_udm_read.py and scripts/evaluate_transfer_start.py | PredicateEvaluationResult |

Do not compile a second copy of the same Skill source from ca-agent-demo. The official source is ca-skills; ca-agent-demo is a behavior reference only.

The current imported UDM Tool Output Schema is too broad to prove nested subscription fields read by the selection Script. Until the Tool Owner supplies the complete schema, the linker must report the gap. Do not restore the removed selection-input.schema.yaml as a Skill-owned Script Input Schema.
