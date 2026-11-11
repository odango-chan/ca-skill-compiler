# MAS 方案 C × PR #182：逐步骤、协议、证据和失败语义对照

> 2026-11-10；状态：**DESIGN / NO IMPLEMENTATION**。本表供双方讨论和逐条验收。参照：《NW-Agent 逐步直调 SMF-MAS（方案 C）系统设计说明书》v0.3.3，尤其第 3.3、4.2、4.3、4.9 节和第 3 轮评审；本仓 PR #182 `skills/establish-pdu-session-mas`。本仓的 **Agent Host 就是 MAS 方案 C 的 NW-Agent**，不是两个独立 Agent；当前执行确定性 Procedure，未来 LLM 模式延后但仍在架构范围。
>
> 本文件不确认 MAS 服务端 As-Built。文档声明、接口真实实现、Mock PASS、真实 E2E PASS 必须分开记。`agent_step` 是 SMF 侧方案 C 开关，不等于 Agent Host 的“LLM”执行模式。

## A. 范围：只审查 Initial Establishment

MAS v0.3.3 覆盖建立/修改/释放及 ACTIVATING；PR #182 只实现 **候选 Initial Establishment Skill 源**，并锁定非漫游 IPv4、SSC1、一个默认非 GBR Flow，**不是**完整方案 C。MODIFY/RELEASE/ACTIVATING 作为同一 Host 的 ingress 分流与后续业务 Skill/受控操作记录依赖，不可伪装成当前 Skill 已支持。

MAS 认证 Establish Plan：`wf-certified-pdu-est-policy-first-v1`，`oes-dsl/1.3`，`sm-mas-ops/3.2`。该顺序是跨仓比对基线。不能同时启动 MAS OES Execute 和 Agent Host frozen Procedure。

## B. ESTABLISH 逐步骤映射（先修设计，未实施）

| # / MAS step | 认证 Catalog `mas.op` | #182 对应 | 差异 / 待裁决 |
|---|---|---|---|
| 1 s1 | `SC.VALIDATE_SUBSCRIPTION` | `subscription.validate` | MAS `sub_summary` (及 validated/secondary_auth_required) 必须是可信订阅选择输入；不得以布尔 `subscription_valid` 代替完整订阅事实。验证是否可能创建本地/外部状态，保守 MUTATION 合理。 |
| 2 s2 | `SAA.SECONDARY_AUTH` | **缺失** | #182 不支持二次认证；MAS Plan 即使 `required=0` 仍显式调用并返回 `phase=SKIPPED`。需确认 no-auth profile 是否仍必须调用 SAA；不能无说明删步，也不能把 required=0 直接当作来自订阅的已证明事实。 |
| 3 s3 | `SC.CREATE_SM_CONTEXT` | `context.create` | 必须绑定 `sub_summary`，消费 `session_keys, lifecycle, sm_context_ref`；`sm_context_ref` 是 v0.3.3 新键（禁止旧 `amf_ref`），Host 保存 SMF authority Location 而不是生成 Agent URI。 |
| 4 s3r | `SC.REGISTER_UECM` | `subscription.register` | #182 说明复合执行 SDM Subscribe + REGISTER_UECM；MAS 认证 Plan 只显式包含 REGISTER_UECM。必须核实 SDM Subscribe 的实际 MAS owner 与调用时机，不可凭空拆出隐藏外部操作或只验证 UECM 就宣称两项完成。 |
| 5 s3p | `PA.CREATE_SM_POLICY` | **未在对应时机调用** | 认证 Plan 为 **policy-first**，在选锚/地址分配之前 Create；其输入使用 `session_keys`，输出 `up_intent, authorized_qos, policy_assoc_id`。与 #182 当前地址之后单次 `policy.create` 冲突。 |
| 6 s4 | `UPS.DECIDE_ANCHOR` | `anchor.select` | 必须在 Policy Create 之后；返回 `up_anchor` 而不是凭习惯写 `anchor`。若纯选择可归 READ，需要 MAS owner 的无副作用证明后才能改 Effect。 |
| 7 s5 | `IAM.ALLOCATE_ADDRESS` | `address.allocate` | 输入 `allocator, dnn, up_anchor`，返回 `ue_addr`。`allocator=LOCAL_POOL` 是认证 Plan 值，不能自动适用于其他 deployment profile。 |
| 8 s6u | `PA.UPDATE_SM_POLICY` | **缺失** | 以已分配 `ue_addr`、`policy_assoc_id` 更新授权 Policy；最终 UP intent / QoS 必须使用 Update 的权威输出，不能用 Create 的过期值。 |
| 9 s7 | `CA.CREATE_CHARGING` | `charging.create` | 绑定 `supi, psi, rating_group_refs`；必须确认 `default` 来源和负责者，回执 `charging_ref`。 |
| 10 s8 | `UPN.CREATE_UP_SESSION` | `user_plane.establish` | 输入来自 s6u 的 `up_intent`、s4 `up_anchor` 和 s5 `ue_addr`；返回 `n4_seid, est_done`；初始 N4 不应假称已有 N3。 |
| 11 s9 | `IAM.CONFIRM_ADDRESS` | **缺失** | 认证 Plan 为 optional 但显式步骤。确认何时允许 SKIP、租约证据以及与最终清理责任的关系。 |
| 12 s10 | `SSM.SIGNAL_UE_AN` | `access.signal` | 必须容纳 SSM→Agent Host→AMF 的原样 Namf Transfer 与 AMF→Host→SMF 的原样 Nsmf Update；**Namf 200 不等于 RAN 已完成**。完成槽当前受 R-3-01 阻断。 |
| 13 s11 | `UPN.MODIFY_UP_SESSION` | `user_plane.activate` | 用 s10 的 `an_tunnel_teid/ip` 和 s8 的 **同一个** `n4_seid`，验证 `n3_synced` 及同一 PFCP/N4 session。 |
| 14 s13 | `SE.EMIT_EVENTS` | **缺失** | MAS 认证 Plan 使用 `ev=ESTABLISHED`、`async_ok`。需确定是必需业务步骤、允许异步成功、或由资源责任步骤替代，不能无理由删去。 |
| error s14 | `SC.COMPENSATE_RELEASE` | `resources.cleanup` | 认证 Plan 只有**尽力补偿**，与本仓“完全释放或持久责任接管”语义不同；需明确逐资源清理/残留责任和 outcome unknown 后规则。 |
| Host-only | 不属于 MAS Catalog | `resources.finalize` | 应建模为 Host/外部 owner 的责任确认或独立已准入能力，不能伪称某个已实现 MAS Op。 |
| Host-only | MAS v0.3.3 未提供独立查询契约 | `operation.reconcile` | **不得**把 reconcile 作为普通业务 Tool `call`；使用 Runtime 原生 `reconcile: original-step`，MAS 端需提供原操作确定结果的 API/可核验回执。 |

**建议设计决定**：候选执行源严格对齐 policy-first 顺序，而不是让 Adapter 在一个 `policy.create` 中隐藏 Create→Update；如确实要抽象/合并，必须证明业务等价、输出/证据/Effect 和异常可回放性，并经联合审查。首期最稳妥：Procedure 显式拆两步。`subscription.register` 等组合也同理。

## B.1 当前唯一 Fabric SBI 与方案 C Nsmf 门面的迁移冲突（OPEN）

按 `AGENTS.md` 及 `docs/design/5gc-agent-host-architecture.md`，当前生产只有 Fabric SBI / CreatedSession ingress；MAS v0.3.3 要求 AMF 对 Agent Host Nsmf-compatible HTTP Facade。这**不是第二个 Agent**，却是新的生产**协议入口形态**，因此必须经本仓统一 Host 入口架构/配置准入批准：推荐 Nsmf 作为同一 Host API 的协议 Adapter，共用 Routing、Admission、Journal、Runtime，且同一会话禁止同时被 Fabric 与 Nsmf 入口认领。未批准/未验证前将 Nsmf 生产门面标记为 `OPEN / NOT_READY`，不能直接以 MAS 方案 C 文档覆盖现有唯一 Fabric SBI 约束。

## C. 入参、输出与时序的硬边界

### C1. 入站归一化 ≠ MAS wire

- AMF 的 `Nsmf CreateSMContext` 是 Host **方案 C 目标入口（不是当前 main 的已准入生产入口）**；Host 必须从合法协议中取得 `supi/ue_identity、psi、dnn、slice、type` 等，并结合受信部署配置/操作上下文建立 Skill input。
- 本仓请求 Schema 还要求 `pti, requestType, servingPlmn, requestedSscMode`；MAS `start_inputs` 没有一一列出。必须写出字段来源/推导资格和缺失拒绝，不可随意填固定值。尤其 `requestType=1`、`requestedSscMode=1` 只能是**验证通过的受支持条件**。
- Host 的 `ingressRef/lifecycleRef` 是受信本地执行事实，与 MAS wire `sessionRef, smContextRef, stepId, idempotencyKey` 不是同一数据模型。Adapter 做单向、有类型、可溯源投影；不把 MAS 私有字段误当 Host 通用根上下文。
- 既有 `createdsession_ingress/createdsession_lifecycle` 名称仅在旧接入兼容边界处理。MAS Skill 的目标契约使用中性的 invocation/ingress/lifecycle facts；Runtime 不增加按 Skill 名硬编码。

### C2. Tool 输出/跨步骤绑定

- MAS wire 的 `outputs` 包含 `sub_summary, session_keys, sm_context_ref, up_intent, authorized_qos, policy_assoc_id, up_anchor, ue_addr, charging_ref, n4_seid, an_tunnel_*, n3_synced` 等。#182 ToolCatalog 当前将每步输出统一收缩为 `operationRef + stateRef`；这只是 **Adapter 内部句柄设计**，不足以证明等价绑线。
- 任一下一步消费的 MAS 输出必须有显式字段映射 **或** 通过经安全校验的 `stateRef` 做受信内部解析；需记录 producer、source operation、value 型别、会话代次，并防止跨 Invocation/会话混淆。不得用“同一个 stateRef”掩盖缺字段或实际值不匹配。
- `sm_context_ref` 首次由 `SC.CREATE_SM_CONTEXT` 得到；`Location` 仍由 SMF 权威 URI/受控拼接形成，Host 不改写 host 为自身。AMF 后续使用 Host 基址而不是 Location host 是外部消费者假设，需实测。
- MAS 会话态以各 MAS 为权威；Agent Host Ledger 是审计投影。Replay / 重启不得依赖纯内存 Ledger 去证明未知原操作未提交。

### C3. 外部 I-INV 分类

| 观察到的行为 | Host 应有解释 |
|---|---|
| 未离开 Host 的输入/ACL 拒绝，已确认无外部发送 | 可按 fail-closed DENIED / NOT_COMMITTED（具体结果契约另定） |
| MAS `200 + outputs` 且可信 operation/producer/关联/输出校验通过 | 只证明本次 Tool 的操作结果；业务阶段还要独立 Observation/PASS |
| MAS 明确业务拒绝的 4xx（可证明操作未提交） | 分类已知失败；何时执行已准入补偿需由资源事实决定 |
| MAS 504/502、断线、超时、输出无法解析、或 4xx/5xx 无副作用保证 | **OUTCOME_UNKNOWN**；原 operationRef 和同一 idempotencyKey 保留；先 exact-original reconcile / 安全停止，不得立即新的补偿写 |
| MAS 回执报告 COMMITTED，但权威 Observation 缺失/错关联/旧实例 | UNKNOWN；不进入下一步 |
| 原操作对账无法由 MAS owner 提供 | UNRESOLVED（阻止本执行继续），外部资源责任不能静默消失 |

MAS 的 `idempotencyKey=agentTxnId:stepId:attempt` 里 `attempt` 必须指原操作实例，而**不是每次 HTTP retry 都加一**。同一原操作的对账/重放需稳定不变；新的 attempt 是**新准入授权**，不得用在不确定的前一次写上。

### C4. Reconcile 兼容性（确定的现存阻断）

当前 `runtime/tool_contract.py` 的 Tool/Capability Effect 只接受 `READ / MUTATION`，`RECONCILE` 是 Backend 对原始 Operation 的方法返回 Effect，而非注册在 ToolCatalog 的普通 Operation Effect；当前 `runtime/engine.py` 也有独立 `reconcile` Node 和原操作记录。

因此 #182 的 `call: ca.pdu_session.operation.reconcile`、`effect: RECONCILE`、`from: <原 mutation>.output.operationRef` **不能作为现有模型直接执行**。建议将每个 OUTCOME_UNKNOWN 路径改为 `reconcile: <原 mutation node id>` 并保留 `COMMITTED/NOT_COMMITTED/UNRESOLVED → SAFE_STOPPED/UNRESOLVED`。对外确认原操作查询 API 后，再补 `reconcileOutputSchema`，不要修改通用 Effect gate 以放过该草案。

### C5. Root eligibility

#182 `SkillContract.spec.preconditions` 写 `mas.establishment.eligible`，但 Procedure 首个节点直接调用 subscription.validate。本仓当前 `Runtime._check_hard_preconditions()` 仅对 CHILD 有效，`admit_skill_entry()` 只核查 required facts 存在。目标改法：在任何可能变更的外部调用之前通过显式 verify 或经正式审查的 Root precondition enforcement；当前优先显式 eligibility verify，验证拒绝时 0 Mutation。

### C6. Host 与 MAS 的业务互斥和回调

- `plan_provider=agent_step` 时，SMF 旧 Nsmf 不可误入 OES Execute；只有 outstanding SIGNAL / FailureNotify 的 **原始消息**回灌、ACTIVATING 规定路径可用。
- 某个 I-INV SIGNAL park 时，同一个 Host 仍须并发受理 `Namf_Communication.N1N2MessageTransfer`、F-TEID Nsmf Update；同连接回 Namf 的 HTTP 状态/body；F-TEID 切到 SMF 既有 OnSouthbound，**不启动另一个 MODIFY Skill**。
- MAS 第三轮 **R-3-01 Critical** 仍待解决：`smf_oes_sess_active` 门禁与 `smf_oes_on_mas_response` 绑定 OES；方案 C 必须具备**非 OES 的完成槽**，否则 SIGNAL 的正向 I-INV 无法结束。该问题由 MAS 服务端解决，Host 侧只验收。
- ACTIVATING / Service Request 走 `MasOrchestration.SmContextUpdateSync` 立即同步回 Update N1/N2，不进入本 Establish 流程；普通 MODIFY 缺少补偿声明时直接明确终止，不推断可用 Establish 补偿。

## D. 负例与联调关闭标准（先审设计，不将 Draft 判通过）

| ID | 场景 | 必须验证 |
|---|---|---|
| T01 | 有效 policy-first Initial 建立 | 完整步骤次序、`PA.CREATE` 先于地址、`PA.UPDATE` 在地址后；所有从前步输出绑定的是本 session 实值 |
| T02 | Unsupported profile / secondary auth | 进入网络写操作前拒绝或使用另行认证的扩展 profile；不能偷偷跳 SAA |
| T03 | 缺 `pti/PLMN/SSC` 等入参 | 明确的受信来源或拒绝；不合成未知字段 |
| T04 | `sm_context_ref` 缺失或仍是旧 `amf_ref` | 不组装假 Location，按契约错误失败 |
| T05 | MAS `HTTP 504` 后迟到成功 | 原操作不换幂等键、不重复送、exact-original reconcile；责任持久保留 |
| T06 | MAS 4xx 但已部分创建资源 | 不简单宣称 NOT_COMMITTED；资源由 owner 清理/接管，缺证据 UNRESOLVED |
| T07 | 仅 Namf HTTP 200、未收到 RAN setup | 不宣称 `ran_accepted`，不得推进 N4 forwarding |
| T08 | SMF OES 无活动实例但 SIGNAL outstanding | F-TEID 仍唤醒方案 C I-INV；没有第二条 OES cursor |
| T09 | 过程 Nsmf Update / FailureNotify | 原始消息路径与 multipart/headers 实际一致，不产生 MAS `Southbound` wrapper |
| T10 | `N4 modify` 在不同 SEID/实例 | Predicate UNKNOWN / FAIL，不允许 success |
| T11 | `COMPENSATE_RELEASE` 部分完成 / Host 重启 | 未结束责任仍有持久 owner；不存在“清理 200 即释放”的假证据 |
| T12 | 普通 MODIFY 失败但无补偿 Plan | 明确失败终态，不调用建立补偿，不静默挂起 |
| T13 | ACTIVATING Update | 同步 N1/N2 回包，不被 SIGNAL park 阻塞 |
| T14 | 错 token、错 mas/op、错 sessionRef 与重放 | 拒绝越权；相同 idempotencyKey 只关联**同一原操作** |
| T15 | 选择 MAS Skill 但缺任何 Effect/Adapter/Observation | READY fail-closed；不会退化为不受控 HTTP 调用 |
| T16 | 未来 LLM 选择动作（暂不执行） | 设计须继承相同 admission/action/evidence/Effect/Trace/Finalize；不许替代/绕过 MAS Context 权威 |

## E. 双方需回答的开放问题（阻断 D1 或 D2）

| ID | 接口拥有方 | 需要确证的事实 | 关闭标准 |
|---|---|---|---|
| O01 | MAS SC/SAA | no-auth profile 必须显式调用 SAA 吗；SDM Subscribe 实际在哪一步执行？ | Catalog/As-Built + 同路 mock |
| O02 | MAS PA | `CREATE_SM_POLICY`→`UPDATE_SM_POLICY` 输入与具体 QoS 资源所有权 | 精确 schema + 前后地址约束测试 |
| O03 | MAS SMF `mas-orchestration-sbi` | `sessionRef` 建绑、`sm_context_ref` 输出、新旧键兼容策略 | 真实 Invoke 输出证据 |
| O04 | MAS 所有 Mutation owner | 发送超时后按 `idempotencyKey` 查询原操作的结果、是否提交、子操作资源列表 | 原操作查询 API + 迟到成功测试 |
| O05 | MAS SSM / NGAP / OnSouthbound | **非 OES 的 park completion slot**；旧 Nsmf 入口互斥门禁 | R-3-01/R-3-03 关闭 + 真回灌证据 |
| O06 | MAS SC / 资源 Owner | `COMPENSATE_RELEASE` 是 best-effort 还是有逐资源释放/接管凭据？ | 可核验清理凭据，否则 Host 保持 UNRESOLVED |
| O07 | MAS SE/IAM/CA | `CONFIRM_ADDRESS`、`EMIT_EVENTS async_ok` 的必要条件与完成语义 | 标准字段及已确认的可跳过政策 |
| O08 | Host ingress + AMF consumer | Nsmf 入站可推导哪些 canonical Skill fields？AMF 是否持续路由到 Host 而不按 Location host？ | 字段来源表和真实 AMF 消费者测试 |
| O10 | Host ingress / deployment owner | 当前唯一 Fabric SBI 生产入口如何演进到 Nsmf Facade，保证同一 Host、同一 session 互斥 claim？ | 版本化 ingress adapter + config gate + 无双调度测试；未经批准不开第二生产入口 |
| O09 | Host Tool/Effect owner | MAS 能力按 READ/MUTATION 的独立契约、观察者可信来源与 fresh/correlation 条件 | 每项 Effect / Observation 的拒绝负例 |

## F. 推进顺序（每一项都先出设计/证据）

1. **当前 PR（仅文档）**：保留 #182 的实际程序原样，记录本表差异与继承原则；不宣称 MAS ready。
2. **D1 业务程序对齐**：按认证 Plan 修订 Skill source、分离 Policy Create/Update、修正原生 Reconcile/eligibility；不得加入 MAS wire 到 Runtime 通用规则。
3. **D2 安全工具实现**：逐项 Adapter+独立 Effect+单一 Tool Registry+原操作 Journal/Observation+资源责任；缺任何证明时 NOT READY。
4. **D3 双向会话与流程**：同一个 Host 处理 Nsmf/Namf/Update/ACTIVATING；MAS 完成槽和路由互斥必须先关闭。
5. **D4 联合验收**：模拟覆盖 T01–T16（LLM T16 为架构设计审核，不是运行 PASS）；真实 E2E 建立后再扩展 Modify/Release。保留既有 Skill 回退，不能让 MAS 设计变化影响旧运行链。

**非目标**：在设计 PR 中修改 MAS 源码、将 `oes-dsl` 解释器并入 Runtime、把 Host 限定为永久纯确定性、将 MAS 的 `Agent` 误解成另一个平台、以接口草案宣称联调成功。
