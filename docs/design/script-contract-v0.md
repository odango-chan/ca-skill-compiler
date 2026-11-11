# Script Contract v0 · Producer-owned input / Script-owned output

**Status: ACCEPTED DESIGN (2026-11-11); NOT_IMPLEMENTED; tests NOT_RUN.**
This supersedes the prior requirement that every Script re-declare external Input TypedDict and the earlier embedded-YAML-header proposal.

## 1. Data ownership is authoritative

| Data producer | Source of truth | Compiler action |
| --- | --- | --- |
| Tool / MCP Operation | imported read-only `lib/tools/*.yaml` | link its inputSchema/outputSchema by operation ID |
| Host request, context, facts, observation | actual Host/Ingress/Observation owner (or authored Skill requestSchema if Skill-owned) | resolve source/schema; missing contract = explicit incomplete/error |
| Script call parameter container | Procedure `with:` bindings to producer outputs | derive an input projection per call site, **not a new authority** |
| Script's newly computed result | Script-owned strict output type | generate the output JSON Schema |
| Unchanged pass-through | original producer | use original schema directly, not a broad copy |
| Existing complex Script | existing authored external input/output Schema | retain verbatim, do not infer from empty Python dict classes |

**No green validation from unknown fields.** A generic `type: object` without declared fields cannot justify a deep `from: tool.output.foo`. A missing Host Schema is not fixed by synthesizing one.

## 2. Authoring a new Script

~~~python
from typing import Annotated, Literal, TypedDict

class NormalizeOutput(TypedDict):
    normalized: Annotated[str, {"minLength": 1}]
    accepted: Literal[True]

def main(input: dict) -> NormalizeOutput:
    return {"normalized": input["value"].strip(), "accepted": True}
~~~

The Python `dict` is only a **runtime parameter envelope**. Its effective keys/types must all be derived from the calling Procedure's `with:` and producer schemas. It does **not** permit uncontrolled input. The Script's own **output** must be a fully typed, nonempty `TypedDict` (including all nested fields); broad output `dict`, `Any`, `object`, unknown nested types, empty classes and unsupported unions must fail closed.

Type whitelist for new outputs: `str/int/float/bool`, nonempty local `TypedDict`, `list[T]`, `Literal[...]`, `NotRequired[T]`, static `Annotated[T, {"minimum": ..., "pattern": ...}]` with type-correct bounded JSON Schema constraints. A generated TypedDict object is closed (`additionalProperties: false`). Cross-field conditions that cannot be expressed by this subset must retain an explicit existing external Schema; do not weaken constraints to force generation.

The Compiler statically analyzes declarations and binding paths; it must not run Python, Tool, MCP, network code or arbitrary imports. Real execution sandboxing, validation and readiness belong to the Host.

## 3. Input type comes from Procedure binding

~~~yaml
procedures:
  normalize:
    start: read
    nodes:
      read:
        call: demo.read
        with:
          key: {from: input.dnn}
        'on': {SUCCEEDED: normalize-step, FAILED: failed}
      normalize-step:
        preconditions:
          steps:
          - name: normalize-value
            script: scripts/normalize.py
            with:
              value: {from: read.output.value}
        terminal: SUCCEEDED
      failed: {terminal: FAILED}
~~~

- `input.dnn` is typed by this Skill's authored requestSchema.
- `demo.read` input/output fields are typed only by the imported `lib/tools/demo.tool.yaml`.
- `read.output.value` must be a declared Tool output field. The derived Script input projection is per invocation site and never a second editable Schema.
- The new strict Script Output generates only `output/skills/normalize/schemas/generated/normalize.output.schema.json`. It must preserve all specified constraints.
- Missing/ambiguous producer fields, unknown operation, incompatible binding, unavailable Host schema or broad undifferentiated object => precise diagnostics; no guessed type.
- Source maps should point to Procedure binding and original lib/Skill schema.

## 4. Import actual external libraries for tests

Already committed in `examples/ca-agent-demo-project/lib/tools/`:
- `createdsession.tool.yaml`: exact real ca-agent-demo ToolCatalog, 12 Operation declarations.
- `mas-proposed.tool.yaml`: exact MAS design ToolCatalog, 16 **PROPOSED / NOT_READY** declarations.

Pinned upstream commit: `8ab5145b992f74ee793e0d35a4ec96e6b26500a3`. All 79 authentic Skill files and the official ToolCatalog match source Git blob IDs byte-for-byte. Importing a ToolCatalog does **not** prove Tool implementation, authorization, independent Effects or Runtime READY.

**Not yet available:** a confirmed authoritative schema covering all Host context/request/facts/observations fields. The existing Root Routing Profile schema is deployment/routing configuration, **not** that interface. When the real Owner defines/contracts those producer values, add snapshots under `lib/host/` or `lib/observations/`. Do not fabricate them just to make a test pass. Existing Skill-owned `schemas/establishment-request.schema.yaml` stays authored source.

Synthetic `examples/script-contract-project` now imports a real read-only copy of `demo.tool.yaml`. The Procedure binds Tool Output to a Script input and the Script declares only its own generated Output.

## 5. Backward compatibility and build output

Five existing ca-agent-demo Python scripts still use their original `# /// connectionagent-tool` headers, external input/output Schema names and authored Schema files. Keep all untouched. Do not rewrite complex `allOf / if / then` schemas as incomplete Python annotations. A separately reviewed owner migration may replace them after semantic equivalence is proven.

New output schema generation is deterministic, uses JSON Schema Draft 2020-12, has no timestamps/digests/releases, emits only to `outputFiles`, and never writes into author-owned `skills/`. Imported libraries remain read-only in Studio. The old embedded YAML comment mode is **withdrawn**.

## 6. Required acceptance (all currently NOT_RUN)

- SO-01: load 12 real Tool Operations and 16 proposed MAS Operations; preserve NOT_READY distinction.
- SO-02: Tool Output -> Procedure with -> Script Input projection, with no duplicate Input TypedDict.
- SO-03: missing Tool/Host producer field or weak generic object -> diagnostic, no false PASS.
- SO-04: strict Script-generated Output Schema retains required/Literal/Annotated constraints.
- SO-05: broad/empty Script Output class -> fail closed.
- SO-06: five legacy Scripts and all external source schemas remain byte-identical/linked.
- SO-07: no Tool Schema is re-owned by Script; exact origin reference preserved.
- SO-08: same file map yields identical Native/WASM diagnostics and output bytes.
- SO-09: compiler never runs Python or calls external Tools/MCP.
- SO-10: incomplete Host context schema prevents claims of complete static typing.

Compile PASS is not a runtime capability or 5GC readiness result.
