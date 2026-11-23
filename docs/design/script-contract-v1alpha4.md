> **HISTORICAL / NOT CURRENT INPUT (v2.1.0).** This design/work-package document is retained for original source and acceptance provenance only. The active Compiler accepts **CA-SL v1alpha5**; earlier versions are not compatibility modes. Original status statements and instructions below describe their historical snapshot. See [documentation](../README.md), [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), [Development contracts](../../contracts/development/v1alpha1/README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Skill-owned Script Contract v1alpha4

**Status:** FROZEN Compiler implementation contract subordinate to CA-SL v1alpha4 §§32 and F.6.

Normative authority: [the frozen CA-SL specification](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).

## 1. Two Script roles

A Skill-owned Python Script has exactly two supported roles:

1. **Procedure Script:** deterministic local computation, selection, transformation or structured-data construction, invoked by a Node's Operation stage.
2. **Predicate Script:** pure Predicate evaluator, invoked by a Precondition or Postcondition PredicateCall.

Neither role is a Tool, Observation, Capability, external callable or general-purpose execution environment. There is no localTools, Tool metadata block or independently invocable Script identity. Script identity is Skill package identity + package-relative path under scripts/**.

## 2. Procedure Script ABI

The only entry point is:

~~~python
def main(input: InputTypedDict) -> OutputTypedDict:
    ...
~~~

- Exactly one main entry point.
- Exactly one positional argument named input.
- Input and return annotations must be statically resolvable strict TypedDict types.
- The return type is nonempty and fully declared.
- Compiler MUST NOT execute the function body to infer types, outputs or behavior.

### 2.1 Derived input

Procedure Script authors do not create an Input Schema. At each call site, the Operation's named arguments define the complete input projection:
- every required TypedDict key is supplied;
- no unknown key is supplied;
- each SourcePath is resolved against the actual Producer Schema;
- each literal is checked against the destination property Schema;
- Source-to-target assignability is proven by F.5.

The same Script may have different valid input projections at different call sites. Compiler creates a distinct static input type for each invocation. Runtime materializes and validates the actual input before execution.

Inside the Script, the generated input TypedDict is the authoritative type. Unknown literal keys, unguarded optional-key reads and dynamic key access that cannot be statically proven are rejected with the frozen Script diagnostic. Compiler does not accept an open dict[str, Any] escape hatch.

### 2.2 Derived output

Compiler derives Output Schema from the return TypedDict. The author maintains the Python signature, not a duplicate YAML/JSON Script Output Schema.

Output Schema generation must preserve required vs NotRequired fields, nested TypedDict structure, primitive types, Literal values and supported Annotated constraints. Functional TypedDict declarations are accepted only under the canonical source form frozen in §4.

The derived Schema is read-only generated output. Runtime validates the returned Python value against it. A Python exception or invalid return value is a runtime failure, not a successful build.

## 3. Predicate Script ABI

The only entry point is:

~~~python
from __future__ import annotations

def main(input: PredicateInput) -> PredicateEvaluationResult:
    ...
~~~

PredicateInput is a compiler-generated virtual TypedDict, derived from the exact owning Predicate's ordered parameters and linked parameter Schemas. It also contains the required read-only evidence key using the §17.3 Evidence projection shape; business parameter keys are required, while evidence members may be absent where the normative projection permits it. The Script MUST NOT declare, import, shadow or override that generated type.

The Predicate Script output is exactly the strict PredicateEvaluationResult TypedDict defined by §32.2: result (Literal PASS / FAIL / UNKNOWN), reasonCode (nonempty string) and evidenceUsed (list of strings). It is not a free-form Boolean return.

If one Script file is reused by multiple Predicate definitions, the generated input shape must be identical for every use. If it differs, the Script contract is invalid; the compiler must not choose one caller's shape.

The Predicate's Evidence requirements and actual Producer bindings remain authoritative. Script code cannot create, upgrade, refresh or manufacture Evidence guarantees.

## 4. Supported typing subset

The exact supported set is CA-SL F.6, not all Python typing features. The minimum supported declarations are:
- str, int, float, bool;
- strict nested TypedDict;
- list[T];
- Literal;
- Required and NotRequired;
- Annotated[T, {...}] only where metadata is a literal mapping to a G.4-supported Schema constraint.

Functional TypedDict is accepted as a declaration syntax for the same strict TypedDict / JSON-object type described by F.6; it does not add a CA-SL value type or alter requiredness / nullability semantics. The canonical supported form is:

~~~python
from typing import TypedDict

Policy = TypedDict("Policy", {"5qi": int, "dnn": str})
~~~

The assignment target and first literal string argument MUST be the same identifier. The second argument MUST be a literal mapping with unique literal string keys and supported F.6 type expressions. total may be omitted or the literal True; total=False, computed keys, mapping unpacking, dynamic type expressions, aliases of the factory, arbitrary factory calls and runtime-generated declarations are rejected. Optional fields use explicit NotRequired[T]; required overrides use Required[T] where supported. Nested declarations may reference only previously resolved strict TypedDict types. This is a compiler source-syntax profile for the frozen strict TypedDict concept, not a new CA-SL type. Annotation parsing MUST preserve the distinction between a missing property and a nullable property. The compiler parses this form statically and MUST NOT import or execute it.

Reject Any, open dict, object, dynamic/unresolved annotations, unsupported union/optional forms, total=False, unknown or dynamic Annotated metadata, unsupported JSON Schema keywords (including pattern and format), dynamic key reads that cannot be proven, redefinition of compiler-generated PredicateInput and noncanonical Functional TypedDict forms; imports are never executed by Compiler and annotation dependencies must be statically resolvable.

The Python interpreter version used by Host/Runtime is not selected by this Compiler contract. Runtime MUST pin and qualify an interpreter that supports the typing constructs used by the package; that runtime version MUST be recorded separately and MUST NOT be inferred by Compiler.

Do not infer a Schema from sample data, comments, function bodies or observed return values.

## 5. Static AST checks

Compiler parses the Python AST and typing declarations without importing or executing the Script. It must locate one supported main, resolve strict input/output TypedDict declarations, derive JSON Schema types and constraints, validate every call-site input key and source type, analyze literal-key reads against the generated input type, reject unsafe reads of NotRequired fields unless presence is proven by the supported guard/read pattern, reject dynamic key paths that cannot be statically bounded, and generate a stable Output Schema and source mapping.

The exact diagnostic code and precedence are from §§24 / H. No Script-specific code is invented by this companion document.

## 6. Purity and execution boundary

Procedure Scripts and Predicate Scripts are deterministic local computations. They MUST NOT access network, HTTP, MCP, shell/subprocess, filesystem, databases, credentials, environment secrets, wall-clock time or random sources, or perform external side effects.

Host/Runtime owns sandbox, memory/CPU budget, timeout, cancellation and actual input/output validation. Compiler never executes Python and a static TypedDict does not prove runtime correctness.

## 7. Generated files

Compiler emits only Output Schemas, never Script Input Schemas. Exact paths and canonical serialization are defined by the [Compiler Artifact Contract](compiler-artifact-v1alpha4.md).

The default generated path for scripts/<relative>.py is:

~~~text
skills/<skill>/schemas/generated/<relative>.output.schema.json
~~~

Example: scripts/common/build_policy.py becomes schemas/generated/common/build_policy.output.schema.json within that Skill package.

Generated files are read-only and MUST NOT be accepted as author-maintained source in a subsequent BuildRequest.
