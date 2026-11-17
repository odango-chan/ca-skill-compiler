# Official Skill static acceptance and diagnostic classification

Updated: 2026-11-12. This is a static Compiler result for the pinned ca-skills v0.2 snapshot and the approved CreatedSession ToolCatalog. It is not execution, Runtime qualification, or live 5GC evidence.

> **已归档 / ARCHIVED — PRE-v1alpha4.** 本文记录早期 v0/v0.1 原型、迁移检查或设计阶段的验收结果。文中的语法、分支/PR 指令、样例诊断数量和验收结论均属于历史快照，不定义 CA-SL v1alpha4 或 Compiler v1.0.0 的当前要求。当前状态请以[工作流索引](README.md)和[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)为准。


## Inputs and reproducibility

- Official Skill source: `examples/ca-skills-v0.2`, pinned ca-skills commit `eeae62badf29f90eee257c7ef2ab504211bf7e30`.
- Approved ToolCatalog: `examples/ca-skills-v0.2/lib/tools/createdsession.tool.yaml`, ca-agent-demo commit `4395982fa6efb97714b97901603b2dfb7018c4d0`.
- Command: `GOWORK=off go run ./cmd/skillc check examples/ca-skills-v0.2`.
- Result: exit 1 with 235 Compiler diagnostics. The command did not emit a success result.
- No official input file was modified. The MAS draft ToolCatalog was not imported.

## Classification rules

| Class | Meaning in this report |
|---|---|
| A | Compiler defect: a valid authoritative source is rejected or an invalid source is accepted. |
| B | Skill source violates the pinned Skill Language v0.2 or its own declared authorization. |
| C | Required Host, Observation, or Tool Owner Schema is missing or too broad to resolve a field. |
| D | A referenced Tool Operation is absent from the approved ToolCatalog. |
| E | The required static Schema implication is outside the Compiler's explicitly supported prover subset. |

Classes describe diagnostic causes, not live system behavior. Several diagnostics can be consequences of one missing Owner contract or Operation.

## `establish-pdu-session`

| Diagnostic area | Result | Class / explanation |
|---|---:|---|
| Parser / language envelope | 0 findings | No current metadata, Contract, Predicate, Procedure, Case, or Script Output syntax diagnostic. |
| Procedure / outcome / flow | 0 findings | Action outcomes, explicit phase routes, node reachability, terminal values, and reconcile causality all pass. |
| Tool Operation identity | 0 findings | All referenced Operations resolve in the approved catalog and pass declared-resource checks. |
| Producer / Consumer binding | 4 findings | B: one Skill Script Output does not guarantee the Tool's `maxLength: 512` bound. E: three string pattern-to-length implications are not proven by the static binding checker. |
| Owner Schema resolution | 98 findings | C: 97 missing Host/Observation Owner schemas and one `transactionRef` field absent from the imported UDM `result` Output Schema. |
| **Total** | **102** | A=0, B=1, C=98, D=0, E=3. |

The UDM Tool Owner's current `result` schema is a broad object without nested properties. The Skill references `result.transactionRef`; the Compiler correctly leaves it unresolved rather than deriving it from Script access. The Script Output for `build_n1n2_delivery` limits `operationContextRef` and `policyResultRef` to strings of at least one character, while the Tool also requires at most 512 characters. The declared Producer Schema does not prove that upper bound, so the binding is incompatible. The Tool's type-less `profileRef.const` does match the Script's typed finite `const` and is no longer an unsupported-schema finding.

## `establish-pdu-session-mas`

| Diagnostic area | Result | Class / explanation |
|---|---:|---|
| Parser / language envelope | 0 findings | Current metadata, Contract, Predicate, Procedure, Case, and Script forms parse. |
| Procedure / outcome / flow | 16 findings | D: reconcile targets reference Operations not present in the approved catalog, so no authoritative reconcile Output Schema can be linked. No independent flow/outcome defect was found. |
| Tool Operation identity | 23 findings | D: 16 `LIB_OPERATION_UNKNOWN`; B: 7 calls are not present in the Skill's `allowedResources`. The seven authorization findings concern absent Operations too, but are attributed to the separate source authorization violation here. |
| Producer / Consumer binding | 43 findings | D: referenced nodes have no declared output Schema because their Operations are unavailable. |
| Owner Schema resolution | 51 findings | C: Host/Observation Owner schemas are not supplied. |
| **Total** | **133** | A=0, B=7, C=51, D=75, E=0. |

The 16 missing MAS Operation identities are not supplied by the approved ToolCatalog. `skills/establish-pdu-session-mas/design/tool-contract.yaml` remains NOT_READY and was not promoted to input. Consequently the 43 downstream output references and 16 reconcile links remain unresolved; no replacement schemas were inferred.

## Gate conclusion

The two official packages do not reach zero diagnostics because required Owner and Tool contracts are absent and several PDU schema implications are outside the current static proof subset. The run found no classified Compiler defect (A) and did not mislabel the documented Owner/Operation gaps as Compiler bugs. This is a PARTIAL static acceptance result, not a PASS for compilation or runtime behavior.
