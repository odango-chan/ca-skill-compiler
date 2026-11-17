> **Skill 评审规范与问题台账（2026-11-10）**：
> - [5GC Executable Skill 编写规范（20 条可审查规则与 G0–G5 闸门）](skill-authoring-review-rules.md)
> - [PR #182 逐项问题、P0/P1/P2 优先级和关闭证据](skill-review-findings-20261110.md)
>
> **2026-11-10 设计更新 / 注意**：方案 C 的 NW-Agent **就是本仓 Agent Host**；当前确定性 Procedure 优先，**LLM 执行仅延期，并非架构排除**。本目录的 #182 ToolCatalog 与 Adapter 表仍是候选草案，**与 MAS v0.3.3 的 policy-first 认证 Plan 存在顺序、步骤和错误语义差异**，实施前先阅读：
>
> - [Agent Host × 方案 C：架构归属、继承和未来 LLM](agent-host-scheme-c-architecture.md)
> - [MAS v0.3.3 × #182：逐步骤、协议和失败语义差异表](mas-scheme-c-contract-gap-matrix.md)
>
> **本次只更新设计文档**，没有调整 Skill/Runtime/MAS 源；不代表 MAS、Adapter、READY 或 E2E 已完成。MAS 方案 C 第三轮中的 R-3-01（非 OES 的 SIGNAL 完成槽）、R-3-03（双执行路由门禁）、R-3-04（无补偿失败终态）仍须以 MAS As-Built 与联调证据关闭。

# MAS PDU 会话建立 Skill · 业务源码与联调边界

> 2026-11-10。**Candidate source / NOT_READY**：本 Skill 的确定性 Procedure 已按 MAS 方案 C v0.3.3 的认证 Establish Plan（`wf-certified-pdu-est-policy-first-v1`，`oes-dsl/1.3`，`sm-mas-ops/3.2`）完成业务顺序和显式输出绑定设计；Adapter、独立 Effect、权威 Observation、原操作查询与真实 E2E 仍未实现/未验收。**禁止把源码对齐等同于可连接的 MAS Skill。**

## 本目录

- [面向模型的业务指南](../SKILL.md)
- [SkillContract：权限、前置与最终完成条件](../assurance/contract.yaml)
- [确定性 Procedure：逐步业务顺序、确切数据绑定和失败边](../assurance/procedure.yaml)
- [PredicateSet：独立业务事实判断](../assurance/predicates.yaml)
- [Agent-facing ToolCatalog 草案](tool-contract.yaml)
- [MAS Adapter / 外部 owner 对照](tool-adapter.md)

正式 Skill authoring 的机读语法权威是已 pin 的 ca-skill-toolchain；本目录 `design/` 是人类审查材料，不是第二个 Tool/Effect Registry。MAS 外部服务端仍是其权威，不在 Host 重造 Open5GS 或第二个 Agent。

## 当前 Skill 的 14+1 阶段

MAS Establish 认证步骤（14）顺序：

1. SC VALIDATE_SUBSCRIPTION
2. SAA SECONDARY_AUTH（本候选 no-auth profile 仍保留显式调用，`required=0`）
3. SC CREATE_SM_CONTEXT（必须有 `session_keys` 和 `sm_context_ref`）
4. SC REGISTER_UECM
5. PA CREATE_SM_POLICY
6. UPS DECIDE_ANCHOR
7. IAM ALLOCATE_ADDRESS
8. PA UPDATE_SM_POLICY（必须消耗上一步已分配地址和原 `policy_assoc_id`）
9. CA CREATE_CHARGING
10. UPN CREATE_UP_SESSION
11. IAM CONFIRM_ADDRESS（认证 Plan 的可选步骤：跳过条件待 MAS owner 证明，当前候选保守保留）
12. SSM SIGNAL_UE_AN（等 SMF 采纳 RAN 结果，不用 Namf ACK 冒充）
13. UPN MODIFY_UP_SESSION（使用相同 `n4_seid`）
14. SE EMIT_EVENTS（认证 Plan 的 `async_ok` 不代表直接丢弃通知责任）

Procedure 额外有一个 Host 资源责任确认阶段 `resources.finalize`，并有失败 `resources.cleanup`（MAS `SC.COMPENSATE_RELEASE` 是 **best-effort**，不能冒充“全部资源已释放”）。

**新 Skill 的 Agent Host 就是 MAS 文档的 NW-Agent。** 现阶段使用已有确定性 Procedure；未来 LLM Skill Executor 仅延期，同样必须经过 Admission、Effect、Budget、Journal、Tool、Observation、Reconcile 与 Finalizer。SMF 端 `agent_step` 不启动 OES Execute。

## 精确数据约束

- `ToolCatalog` 的 `operationRef + stateRef` 属于受信 Agent-side 操作记录；MAS 认证 `outputs` 中列出的 `sub_summary`、`session_keys`、`sm_context_ref`、`policy_assoc_id`、`up_intent`、`ue_addr`、`n4_seid` 和 `an_tunnel_*` **也被直接要求并绑定**，不能由 Adapter 虚构。
- 一些 MAS 文档仅列出字段名，尚未给出完整 Machine Schema；ToolCatalog 用 `opaque` 表示“类型待 MAS 确认”，**不是实际实施时允许任意类型的生产批准**。Adapter 必须按照 MAS 真实 Schema 提供已验证类型、producer、同一 invocation/operation/session generation 以及不可变签收记录。只有确认后才可正式注册。
- `sm_context_ref` 是 MAS v0.3.3 的新 Catalog 输出名，不能继续读取旧 `amf_ref`。Host 将其用于 SMF 权威 Location，不改写成 Agent URI。
- `requestedPduSessionType=IPV4`、`requestedSscMode=1`、`requestType=1`、`secondary_auth.required=0`、`allocator=LOCAL_POOL` 等均限定在**已验证的 profile**。缺值不得由 Skill 读取外部隐式默认，也不得从 MAS 输出猜入站参数。
- 当前保留的 `createdsession_ingress/lifecycle` 是待迁移的 **Host trusted ingress facts**，并非允许 Skill/Runtime 使用旧业务专用机制。需要待批准的 Nsmf ingress adapter 将它们迁移到中性事实，期间不可假称 Nsmf 入口已可运行。

## Predicate 与运行安全

独立 `mas.establishment.snapshot` 当前只有 Source 契约，**尚无已批准的权威实现**。即使 Procedure 源含完整 verify，仍缺 MAS backend，不能通过 READY。每个布尔字段须以可信 producer / operationRef / SMF instance generation / sessionRef 及 freshness 证明。缺证据、错关联和陈旧实例 → UNKNOWN。

首个可能副作用前显式 `mas.establishment.eligible` 验证；每一 Mutation 的 `OUTCOME_UNKNOWN` 必须走 Runtime 原生 `reconcile: <原操作节点>`，并安全停止或未解决。MAS 504、断连、部分错误不等于未提交，不能换新的 idempotencyKey 盲重试或无证据调用破坏性补偿。

## 阻断及验收优先级

- MAS 方案 C R-3-01：非 OES 的 SIGNAL park completion slot；R-3-03：旧 Nsmf 入口双 cursor 隔离；R-3-04：无补偿 MODIFY 失败终态（本建立 Skill 不负责 MODIFY）。
- MAS Catalog OpenAPI、`mas/op` 精确输入输出类型、原操作 reconcile 与幂等键、各资源 owner 清理/持久接管；Host 侧单一 ingress/Root routing 迁移方案。
- G0：Toolchain source schema + 所有 `with/from` producer 路径 + fail-closed 图结构。
- G1：有源字段对照与读/写 Effect 的真实批准。
- G2：mock 成功、拒绝、错误 producer、超时后迟到成功、N1N2/Update 竞态、cleanup 残留。
- G3：真实 MAS/AMF/RAN/N4 E2E。

### 本次未完成

当前版本不包含真实 MAS Adapter，不保证官方认证流程和本仓投影行为已经运行等价。不能发布 READY/Live PASS，也不改变当前已有 `establish-pdu-session` 旧 Skill 生产运行链。
