# PDU Session source migration review

Updated: 2026-11-11. These checks use the pinned, unmodified `examples/ca-agent-demo-project` source and select each Skill independently with the current Compiler. The diagnostic counts are local static migration evidence; they do not describe Runtime or 5GC readiness.

> **已归档 / ARCHIVED — PRE-v1alpha4.** 本文记录早期 v0/v0.1 原型、迁移检查或设计阶段的验收结果。文中的语法、分支/PR 指令、样例诊断数量和验收结论均属于历史快照，不定义 CA-SL v1alpha4 或 Compiler v1.0.0 的当前要求。当前状态请以[工作流索引](README.md)和[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)为准。


## `establish-pdu-session`

Observed: **124 errors**.

| Diagnostic | Count | Migration evidence |
| --- | ---: | --- |
| `EXTERNAL_SCHEMA_MISSING` | 83 | Owner Schemas for `context.subject` and `context.facts` are not included. This includes direct Procedure input sources, which are now checked even when no node consumes the input. The request Schema is present and Skill-owned. |
| `PROCEDURE_UNSUPPORTED` | 18 | Legacy nested steps do not satisfy the v0 named-step/action shape. |
| `STEP_SCOPE_UNKNOWN` | 9 | Six references use unresolved nested step scope `nudm-sdm-subscribe.preconditions.select-subscription-configuration`; three other references use local step names absent from the parsed v0 scope. |
| `BINDING_TYPE_MISMATCH` | 7 | Existing Procedure input declarations and Tool consumer fields disagree in names or declared types (`request-type`/`requestType` and `dnn`). |
| `SCRIPT_TYPE_UNSUPPORTED` | 3 | `scripts/select_establishment_configuration.py`, `scripts/build_pcf_policy_inputs.py`, and `scripts/build_n1n2_delivery.py` still rely on legacy metadata instead of `main(input: dict) -> NonEmptyTypedDict`. |
| `SOURCE_FIELD_UNKNOWN` | 1 | The imported UDM Tool output does not declare `result.transactionRef`. |
| `BINDING_UNSUPPORTED` | 3 | Two `input.slice` bindings and one producer output used by `preconditions.prepare-establishment-response` use schemas outside the supported static projection subset. |

## `establish-pdu-session-mas`

Observed: **85 errors**.

| Diagnostic | Count | Migration evidence |
| --- | ---: | --- |
| `EXTERNAL_SCHEMA_MISSING` | 51 | Owner Schemas for `context.subject` and `context.facts` are not included. This includes direct Procedure input sources, which are now checked even when no node consumes the input. The request Schema is present and Skill-owned. |
| `BINDING_UNSUPPORTED` | 18 | Bindings through outputs such as `ue_addr`, `session_keys`, `policy_assoc_id`, `n4_seid`, and `up_anchor` use producer or consumer schemas outside the supported static projection subset; these are not asserted to be type mismatches. |
| `PROCEDURE_UNSUPPORTED` | 16 | Reconcile steps target proposed Operations without declared `reconcileOutputSchema`; the Compiler cannot derive their result contract. |

## Shared source and authority findings

- `lib/tools/createdsession.tool.yaml` imports 12 traditional Operations and is marked `declared`. This label describes interface provenance only; it does not assert implementation or readiness.
- `lib/tools/mas-proposed.tool.yaml` imports 16 candidate Operations and is marked `proposed`. None is promoted to READY by this Compiler.
- Neither Tool catalog supplies the missing Host `subject` or `facts` Owner contracts. `requiredFacts` names do not define field Schemas. No replacement Schema was inferred.
- Raw files remain unchanged. No separate `examples/ca-agent-demo-v0/` project was created: an authoritative Host Schema is still unavailable, and a copied workflow with those source boundaries omitted would not constitute a faithful migration. The available Skill-owned request Schemas, Tool contracts, migration diagnostics, and current Compiler fixtures are checked independently.
