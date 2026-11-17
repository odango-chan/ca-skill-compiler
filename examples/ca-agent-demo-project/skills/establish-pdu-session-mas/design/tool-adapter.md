> **2026-11-10 对接提醒**：下表仍是 PR #182 的**初步映射，不是批准实现的最终表**。请先对照 [同一个 Agent Host 的方案 C 架构](agent-host-scheme-c-architecture.md) 与 [MAS policy-first 认证步骤差异表](mas-scheme-c-contract-gap-matrix.md)。特别注意 `PA.CREATE_SM_POLICY` 在地址分配前、`PA.UPDATE_SM_POLICY` 在分配后，SAA/UECM/IAM Confirm/SE 额外步骤及本仓原生 `reconcile: original-step` 语义。**方案 C 的 NW-Agent 就是本仓 Agent Host**；后续 LLM 执行策略保留。不得据此草案直接变更 Runtime、宣称 MAS READY，或增加一个独立 NW-Agent Executor。

# SMF-MAS Agent-side Tool Adapter 映射草案

> Design / NOT_READY。MAS 外部协议权威在 MAS owner，本文是 [MAS Skill Procedure](../assurance/procedure.yaml) 到方案 C v0.3.3 的 `sm-mas-ops/3.2` 认证 Establish Plan 的**逐步映射**，不是已安装 Tool。方案 C 的 NW-Agent 就是本仓 Agent Host，不另建 Runner。

## Catalog 对齐（业务调用唯一生产者）

| 认证步骤 | Agent Skill Capability 后缀 | MAS Catalog `mas.op` | 必需的响应字段 |
|---|---|---|---|
| s1 | subscription.validate | SC.VALIDATE_SUBSCRIPTION | `sub_summary` |
| s2 | secondary_auth.disposition | SAA.SECONDARY_AUTH | `phase`；本候选要求 `required=0`，禁止把 literal 当已认证的选择 |
| s3 | context.create | SC.CREATE_SM_CONTEXT | `session_keys`, `sm_context_ref` |
| s3r | uecm.register | SC.REGISTER_UECM | `uecm_registered` |
| s3p | policy.create | PA.CREATE_SM_POLICY | `up_intent`, `authorized_qos`, `policy_assoc_id` |
| s4 | anchor.select | UPS.DECIDE_ANCHOR | `up_anchor` |
| s5 | address.allocate | IAM.ALLOCATE_ADDRESS | `ue_addr` |
| s6u | policy.update | PA.UPDATE_SM_POLICY | `up_intent`, `authorized_qos`, `policy_assoc_id`（**必须**是更新后的） |
| s7 | charging.create | CA.CREATE_CHARGING | `charging_ref` |
| s8 | user_plane.establish | UPN.CREATE_UP_SESSION | `n4_seid`, `est_done` |
| s9 | address.confirm | IAM.CONFIRM_ADDRESS | `ue_addr` |
| s10 | access.signal | SSM.SIGNAL_UE_AN | `an_tunnel`, `an_tunnel_teid`, `an_tunnel_ip` |
| s11 | user_plane.activate | UPN.MODIFY_UP_SESSION | `n3_synced` |
| s13 | events.emit | SE.EMIT_EVENTS | `last_event`（认证 Plan `async_ok`） |
| Host-only | resources.finalize | 外部责任 owner / Host 接管 API（**非 MAS Catalog Op**） | 持久责任交接与晚到结果的可验证依据 |
| Error | resources.cleanup | SC.COMPENSATE_RELEASE + 精确 owner 回执（**非保证完全释放**） | 已知资源的逐项清理、仍有责任的不可变记录 |

**原操作 Reconcile 不作为 Capability 或普通 `call` 注册。** Runtime 只允许针对原 mutation node 的 `reconcile:` 分支，调用原 implementation 的 Backend.reconcile；必须保留原 Host Invocation/node/occurrence、MAS `sessionRef/stepId/mas/op/idempotencyKey` 和外部操作身份。SMF owner 未提供确切状态查询或迟到结果证据时保持 UNRESOLVED。

## 字段生产者及消费者的精确映射

- `context.create.subSummary` ← s1 `sub_summary`；`authStateRef` ← s2 的同一 session 结果。
- `uecm.register.sessionKeys`、`policy.create.sessionKeys` ← s3 `session_keys`。
- `address.allocate.upAnchor` ← s4 `up_anchor`。
- `policy.update.sessionKeys` ← s3；`ueAddress` ← s5 `ue_addr`；`policyAssociationId` ← s3p `policy_assoc_id`。
- `user_plane.establish.upIntent` ← s6u **PA.UPDATE_SM_POLICY** 的 `up_intent`；`upAnchor` ← s4；`ueAddress` ← s5。
- `address.confirm.ueAddress` ← s5；`access.signal` 消费已确认地址的 `stateRef`，实值来自 s5 `ue_addr`，以及 s3 `session_keys`、s8 `n4_seid`。
- `user_plane.activate.anTunnelTeid/Ip` ← s10 `an_tunnel_teid/ip`；`n4Seid` ← s8 相同 N4 实例。
- `events.emit.sessionKeys` ← s3；`eventName=ESTABLISHED`；责任收尾同时消费 s13 的 event receipt 与 s11 的 forwarding state。

所有字段必须与本次 invocation、entry request、SMF session generation、producer operation 及原始 MAS outputs 一一匹配，`stateRef` 不是任意通过的 opaque 授权。当前 [ToolCatalog](tool-contract.yaml) 的若干 `opaque` 类型是服务端 Schema 未冻结的待确认项；严禁将其视为可部署的弱校验生产契约。特别是 `sm_context_ref` 为新名；不读旧 `amf_ref`。

## 尚待确认的 MAS 语义

1. MAS §4.9.3 s2 已指定 `required=0`，但必须确认 subscription 事实是否会推翻该 no-auth profile。若请求需真实 secondary authentication，应路由到其他已准入 profile，不能借 literal 跳过。
2. 认证 Plan 未独立列出 SDM Subscribe 的 Catalog Op。不可再将它暗藏在 `uecm.register` Adapter 中；若 SC 内部有副作用，须由 SC owner 保留其资源及清理证据。
3. `IAM.CONFIRM_ADDRESS` 说明 optional，但我们保守显式执行；若要跳过，必须有基于 MAS 实际 allocator/profile 的已验证选择，而不能由 Host 直接默认跳过。
4. `SE.EMIT_EVENTS async_ok` 允许不等异步终态，不允许凭“请求已发”假称通知责任已持久接管。
5. `SC.COMPENSATE_RELEASE` 为 best-effort，Resource finalizer 和 cleanup_owner 的确切 API 未定义，保持 NOT_READY。
6. 方案 C R-3-01：SSM SIGNAL 的非 OES park 完成槽仍是外部 blocker；单收到 Namf Transfer ACK 不构成 RAN setup 已被采纳。

## Adapter 建议模块职责

- `client.py`：严格的 `MasOrchestration.Invoke` HTTP wire、ACL/模式、idempotencyKey、MAS 错误分类、超时/断连。MAS 非 2xx 不能自动推断 NOT_COMMITTED。
- `session.py`：按本表做单一 Op 的输入投影；从**已验证**上一操作字段映射 `mas/op/inputs`，保证源字段实际存在且未变更会话代次，不在 Adapter 中调度下一步骤或暗藏 Policy Update。
- `evidence.py`：权威 QueryContext / MAS owner 事实；为每个 Predicate 记录 source、producer、session generation、operationRef、freshness；缺证据 UNKNOWN。
- `responsibility.py`：持久原操作记录、exact-original reconcile、外部 owner 精确资源清理/交接，避免迟到成功及重启漏管资源。

### 回调与 Host 入口约束

当前生产 Host 的 Fabric SBI 是唯一已允许 ingress。方案 C 的 AMF→Host Nsmf-compatible Facade 是待批准的**同一个 Host** 的协议适配，不是第二个生产 Host。I-RLY 继续用原始 Namf N1N2MessageTransfer，I-SB 继续用原始 Nsmf Update/FailureNotify，不能自造 MAS Southbound wrapper。ACTIVATING 的同步 I-ACT 不是 Establish Skill 的一个隐藏步骤。单会话只有一个编排 cursor；SMF OES Execute 在 `agent_step` 下不得双跑。
