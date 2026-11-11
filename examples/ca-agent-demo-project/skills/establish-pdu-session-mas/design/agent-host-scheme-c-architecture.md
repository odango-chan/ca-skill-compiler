# Agent Host × SMF-MAS 方案 C：架构归属与演进设计

> 状态：设计草案（2026-11-10），**非实现完成声明**。本文件只规划继承与对接；PR #182 的 Skill、Runtime、MAS 代码均暂不修改。
>
> 对照材料：《NW-Agent 逐步直调 SMF-MAS（方案 C）系统设计说明书》v0.3.3（2026-11-10，外部 MAS 在研文档）；本仓 PR #182（合并提交 `9de90deb`）。MAS 文档描述的是目标/部分既有实现，不能等同于 MAS 当前 As-Built。

## 0. 基本纠偏（不可再误读）

1. MAS 方案 C 所称 **NW-Agent / Agent = 本仓 Agent Host**，是一个系统而不是两个 Agent 或两个编排器；无需再实现一个独立“NW-Agent 执行引擎”。
2. Agent Host 是多执行模式的业务执行主体：**现阶段确定性 Skill Procedure 优先；LLM 路线仅推迟，后续保留/恢复**。不能把 Host 抽象成纯确定性工作流平台；不能用一份外部 MAS Plan 替换未来 LLM 选 Skill / 理解和推理能力。
3. 同一请求恰好选中一个 exact Skill，经过相同的 Admission、Effect、Budget、Journal、Observation、Predicate、Trace、Finalize 机制；不同执行策略不得绕过这些门禁，也不得对同一会话并发推进两个 cursor。
4. 方案 C 下 Host 直接逐步调用 SMF-MAS Catalog 能力，**SMF OES Execute 不参与**。SMF MAS 是会话资源权威；Host 保存受信关联、执行进度和必要的责任证据，不成为 SM Context / N4 的业务权威。
5. MAS 的 `oes-dsl/1.3` 认证计划是对照 MAS 业务语义与 Catalog 入出参的 **外部基线**；本仓 `assurance/procedure.yaml` 是当前确定性执行源。两者不可在同一次请求中各自执行。后续可提供经过验证的版本化投影/编译与语义等价性检查，但**不在当前 Runtime 再造 MAS DSL 解释器**。

## 0.1 与现有 Host 统一架构的兼容条件（OPEN）

本仓 `AGENTS.md`、`docs/design/5gc-agent-host-architecture.md` 和 `docs/design/execution-routes.md` 目前明确：**现行唯一生产入口是 Fabric SBI / CreatedSession flat ingress**（v0.1 不含业务 intent），不得另建独立 Host、第二套 Root Routing、Journal 或 Execution Core。MAS 方案 C v0.3.3 则要求 **AMF→Agent Host Nsmf-compatible HTTP 门面**。这构成真实的**入口协议/部署演进差异**，本设计不能未经裁决就把 Nsmf 门面宣布为第二个现役生产入口。

**建议待批准的收敛方案**：Host 内将 Nsmf Facade 视为一个可部署、受版本/模式约束的 *ingress protocol adapter*，归一化后进入**同一个** RootRoutingProfile → Admission → Link → Invocation → Runtime/Tool → Journal/Finalizer 的 Host API；既有 Fabric SBI Resident 路径继续保持其既有入口与默认路由，在方案 C 联调配置明确切换生产消息入口及外部网元路由，禁止相同会话的 Fabric 与 Nsmf 入口被双重认领。需要一次**明确的统一 Host 入口架构批准**和配置/claim/身份/路由互斥设计，不能靠当前草案直接扩展生产监听。

这与“同一个 Agent Host”不冲突：**协议适配可以演进，业务执行权威不能复制**。另须保留：现阶段 `IntentGoalModelAdapter` / `SkillRoutingModelAdapter` 与 Planning V2 不是全部延期；延期的是完整自由形式、多步 LLM Skill Executor。

## 1. 系统与拥有权

| 职责 | 唯一权威 / 组件 | 不得发生 |
|---|---|---|
| 入站 Nsmf / 身份归一化、选择 Skill | Agent Host ingress + routing/admission（现有路径需补接） | MAS/Adapter 隐式改选 Skill、对同一请求二次准入 |
| 当前确定性业务流程 | 选中 Skill 的 frozen Procedure + Runtime | 在 MAS OES 另执行一份 Plan |
| 未来模型理解/选择/执行策略 | Host model-facing 层，仍受同一个 admitted action/evidence/Effect 边界约束 | LLM 直接发任意 MAS HTTP、伪造业务 PASS |
| MAS Catalog 调用、网络与副作用 | 外部 MAS owner，Host 中的 Tool Adapter 负责精确投影和分类 | Host 直连 UDM/PCF/UPF 或伪造 MAS resource state |
| 会话事实 | MAS 各自权威状态、经独立 Observation 读取 | 将 Ledger HTTP 200 或回执布尔量当成业务权威 |
| 回调、中继和过程 Update | **同一个 Host** 的 Nsmf/Namf 接口与会话关联表；原始消息透传 | 制造额外 `Southbound` wrapper 或第二条 MODIFY Plan |
| 外部未知结果/资源责任 | Host 持久 Journal + MAS 原操作查询/接管凭据；没有凭据则 unresolved | 发送超时被当成未提交后重发、静默丢失资源 |
| 网络入口互斥 | 配置/部署治理 + SMF 入口门禁 | Agent 与 OES 同时推进同一 SMF 会话 |

### 1.1 执行链（当前与未来）

```text
Current production: Fabric SBI (flat ingress) --> SAME Agent Host
Future scheme C: AMF -- Nsmf-compatible facade --> SAME Agent Host
                                                   (mutually exclusive production ownership per session)
                                  | normalized ingress / trusted context
                                  v
                             Skill routing + exact artifact admission
                                  |
                       +----------+----------+
                       |                     |
                  deterministic            LLM (deferred)
                  frozen Procedure        constrained action selection
                       |                     |
                       +---------+-----------+
                                 v
                    ONE governed action/evidence seam
                    Effect / Budget / Journal / Trace /
                    Predicate / Reconcile / Finalizer
                                 |
                           MAS Tool Adapter
                                 |
                Nsmf_MasOrchestration.Invoke (MAS SBI)
                                 |
                         SMF MAS private states
                              /     \
                   QueryContext       N1N2 → Agent Host Relay → AMF
                                                    |
                    process Update/FailureNotify → Host → SMF original SBI
```

- 未来 LLM 不应为同一个 Skill 重写旁路网络栈；它只改变“如何选择下一步已准入动作”，业务能否成功仍由权威 Observation 与 SkillContract 决定。
- 单次执行选定的是一种 **有效的执行策略**，不是两个独立 cursor；切换策略应重新准入/冻结，禁止中途不明归属切换。
- 在目前 LLM 未实现阶段，只验收确定性执行；文档保留策略抽象但**不把 LLM 标记为 READY**。

## 2. MAS 方案 C 的三种“Plan”不要混淆

| 名称 | 作用 | 谁执行 |
|---|---|---|
| MAS 已认证 `oes-dsl/1.3` Plan（例如 `wf-certified-pdu-est-policy-first-v1`） | 业务顺序、Catalog `mas/op` 与 `inputs/outputs` 权威对照材料 | 方案 A：SMF OES；方案 C：Agent Host 选择已准入的等价执行材料，不能再让 OES 运行 |
| 本仓 `SkillContract + Procedure + PredicateSet` | 当前 Agent Host 执行、权限、证据判断的机器契约 | Agent Host deterministic Runtime（当前） |
| 后续 LLM 形成的行动决策 | 动态选择/理解，不替代外部所有权、原操作分类或事实判断 | Agent Host 内受控执行器（后续） |

**对齐策略（推荐）**：先以 MAS Catalog/认证 Plan 为外部互操作基线，修订本仓 MAS Skill Procedure 的逐步顺序与 Tool 参数；保留本仓现有可信 Invocation、Procedure 引用、原操作 Reconcile 等安全语义；对每一步建“等价性矩阵”。不引入第二条 DSL cursor。若未来需要接收动态 `oes-dsl`，必须经版本锁定、验证/编译为单一 admitted frozen graph；它不能靠运行时自由解释替代准入。

## 3. 继承已有能力，不重新造一套

| 本仓已有机制 | MAS 版本继承方式 | 尚缺/需验证 |
|---|---|---|
| Exact Skill admission / `entryProcedure` / frozen graph | 新 MAS Skill 自行声明 `entryProcedure: establish-session`；Host 只信该入口 | 将 MAS 候选加入 deployment routing，保证只准入一个 |
| 独立 `CapabilityContractRegistry` | 对每项 MAS Tool 建部署拥有方批准的 READ/MUTATION Effect，对实际 backend `effect_of()` 交叉核验 | MAS 独立注册、Tool backend 和 Effect 缺失时 NOT READY |
| `ToolContractRegistry` / Tool schema | Skill 调用的能力名与 MAS wire `mas + op + inputs + outputs` 在 Adapter 精确映射 | `design/tool-contract.yaml` 还是草案，不是注册文件 |
| `reconcile: original-node` | 对每项可能副作用的原 operation 使用 Runtime 原操作 Reconcile；Adapter 保存对应外部 idempotencyKey、session/step/op | 外部 MAS 要提供原操作查询/确定性重放结果的证据；不可当成新的业务 `call ...reconcile` |
| 受信冻结调用上下文与 Journal | 由 Host 注入 transaction、invocation、node、occurrence；生成**稳定** idempotencyKey 和原操作 identity | 原 MAS `agentTxnId:stepId:attempt` 必须和本仓“同一原操作不换键”语义对齐 |
| Observation / Predicate 三态 | 通过独立 QueryContext/权威读取证明同一实例/producer/operation 的阶段状态 | 单一布尔 `snapshot` 必须有字段级 producer 和 source evidence；错关联 → UNKNOWN |
| 资源责任 / finalizer | 统一处理已知失败、未知写、晚到回调；无接管凭据则不能报成功清理 | MAS `COMPENSATE_RELEASE` 的尽力清理不等于持久释放证明 |
| Trace、预算、Fail closed | 当前 Runtime 通用机制保持不变 | 每项 Mutation 和超时边界验收 |
| 旧 CreatedSession / Fabric ingress | 将旧字段通过 ingress adapter 映射到中性的 Host 可信事实 | 不让新 MAS Skill 继续依赖 `createdsession_*` 名称；过渡期使用兼容层而非 Runtime 特例 |

**注意**：既有 MAS 设计 A10 的 Agent 关联表纯内存与本仓“UNKNOWN 后保留责任”的治理目标不一致。方案 C 的内存表可用作索引/缓存，但一旦发送了可能产生副作用的 Invoke，其操作身份、结果不确定性和资源责任必须有耐重启的 owner/Journal 或可证明的外部持久接管；否则不应通过本仓 READY。

## 4. HTTP 与双向消息职责（直接继承方案 C）

- 模式：MAS `smf.mas.plan_provider=agent_step`；`local|agent` 是方案 A，不将“Agent Host 的未来 LLM 模式”和 SMF 配置值 `agent` 混为一谈。两者属于不同维度。
- I-IN：**方案 C 目标（尚待 Host 入口架构批准）**是 Host 对 AMF 提供 Nsmf-compatible Create/Modify/Release；不得新增第二个 Host/业务执行链，Create 终态 `201` + **SMF** `Location`，不擅自改写为 Host URL。需要用 AMF 实际路由行为验证“后续仍经 Host 基址”的 MAS 假设。
- I-INV：`POST /nsmf-mas-orchestration/v1/invoke`，`sessionRef, smContextRef?, stepId, mas, op, inputs, idempotencyKey`；外部 Op 的 `outputs` 按 `sm-mas-ops/3.2` 校验，`sm_context_ref` 才是新输出键（不是旧 `amf_ref`）。
- I-RLY：`SSM→Host→AMF` 的 `Namf_Communication.N1N2MessageTransfer` 同 Method/Path/body/multipart；Transfer 应答**同连接**回 SSM，不另造 Relay payload。
- I-SB：AMF 的 F-TEID Update 与 FailureNotify 由同一个 Host 原样转发 SMF 已有 Nsmf/Namf 入口，以唤醒当前 Invoke；存在 outstanding SIGNAL 时不得新启 MODIFY。
- I-ACT：ACTIVATING / Service Request 走同步 `SmContextUpdateSync`，不能排队等待 SIGNAL/Namf park，和普通 MODIFY 路线分离。
- ACL：HTTP 设计优先沿用仓库现有 HTTP-only 原则，独立 Orch-Token + 精确目标 allowlist + 角色/模式限制；MAS 文档可选 mTLS 不应变成本 Demo 必需项。绝不可在 Skill 请求材料里暴露 token/部署 URL。
- 监听和外部调用必须并发，不能因为 Invoke 等待而阻塞 Relay/Update，否则形成挂起死锁；并发不等于同一会话允许两个执行 cursor。

## 5. 对 MAS 当前设计的依赖与阻断（对照 v0.3.3 第三轮）

| ID | 状态 | 对齐决定/验收 |
|---|---|---|
| R-3-01 / M-3-01：SIGNAL park 完成槽依赖 OES | **Critical / MAS 待裁决** | `agent_step` 下即使没有 `smf_oes_sess_active`，OnSouthbound 仍能向原 I-INV HTTP 完成槽交付 2xx/非2xx；双方使用真实相关键验收。Host 无法靠轮询布尔值修复 SMF 完成槽 |
| R-3-03：SMF 旧 Nsmf 路径可能误触发 OES | **Major / MAS 待实现** | `agent_step` 时 Create/非 outstanding 的业务 Update/Release 必须拒收或受控旁路；只允许 outstanding SIGNAL/Failure 的原始过程回灌与 I-ACT 规定路径 |
| R-3-04：MODIFY 无补偿语义 | **Major / 共同设计** | 没有 `on_error.compensate` 时确定地失败/终结；不得隐式运行 Establish 的 `SC.COMPENSATE_RELEASE`；若存在未知副作用，最终责任仍需持续追踪 |
| R-3-02：`sm_context_ref` 输出更名 | **设计稿宣称关闭** | 仅当 MAS 实际 Catalog/Invoke 返回该键且 Host 绑定通过才算集成已验证 |
| A10：Agent 进程内 Assoc/Ledger | **和本仓治理目标冲突** | 分离缓存可丢与持久责任；重启后的未知原操作不能被标成 NOT_COMMITTED |
| D12：Location host=SMF，但后续走 Host | **受消费者行为约束** | 必须用目标 AMF 验证基址/发现路由，不推测任意 AMF 均遵循 |
| I-INV `504 E_TIMEOUT` 和原操作结论 | **重大语义差异** | **HTTP 非2xx 不是 NOT_COMMITTED 证明**：发送后 504/断线/不可证明无副作用 → OUTCOME_UNKNOWN → exact-original reconcile，不能立即发新幂等键重试或无依据补偿 |

本仓对这些 MAS 服务端事项只提出依赖与可验收条件，不改动/批准 MAS 服务端设计。以 MAS As-Built/API 和独立联调证据关闭，不仅凭文档字符串“已关闭”。

## 6. 清晰的分期与禁止提前实施的内容

- **D0（仅文档）**：记录差异、继承矩阵、MAS 需确认的缺失契约与负例；不修改 `assurance/*`、`tools/*`、Runtime。
- **D1（候选源整改）**：与 MAS 确认 Profile / Catalog / 顺序后，重构 `assurance/procedure.yaml`、Predicate、Skill 说明，修正 Reconcile 与前置 eligibility。
- **D2（Adapter & coverage）**：独立 Tool Effect/Schema/Implementation、Adapter、Owner 事实和查询/责任接口；READY fail-closed。
- **D3（连接和反向链路）**：Nsmf Facade + Relay/Update/I-ACT 复用 Host，而不是造另一个 NW-Agent；MAS 完成槽和旧入口隔离。
- **D4（验证）**：先 mock 完整路径与失败矩阵，再真实 MAS、AMF/RAN、N4；保留老 `establish-pdu-session` 独立回退，不在证据不足时覆盖默认部署。

## 7. 设计阶段的关闭标准

1. 统一界定一个 Agent Host、一个 admitted Skill、单一有效业务 cursor、可扩展执行策略；区分 MAS `plan_provider=agent` 与 Host LLM 模式。
2. 建立 MAS 认证 Plan 的每一步 `mas/op/inputs/outputs` → Skill Capability / Predicate / Effect / Reconcile 映射；不偷换业务顺序。
3. 标注无法从 MAS 文档确认的服务端接口、原操作查询、证据来源、资源责任为 **OPEN**。
4. 保证 HTTP 非2xx、MAS 504、丢帧、回灌竞态和重启行为不会被文字规则误判为安全重试/成功。
5. 此阶段仅形成 Reviewable Design，任何验证通过必须区分静态检查、mock 及真实 E2E。
