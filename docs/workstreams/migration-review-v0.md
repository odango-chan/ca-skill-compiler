> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# PDU Session source migration review

Updated: 2026-11-11. These checks use the pinned, unmodified `examples/ca-agent-demo-project` source and select each Skill independently with the current Compiler. The diagnostic counts are local static migration evidence; they do not describe Runtime or 5GC readiness.

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
