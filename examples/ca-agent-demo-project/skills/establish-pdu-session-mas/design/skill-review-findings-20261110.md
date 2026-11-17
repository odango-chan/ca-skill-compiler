# PR #182 MAS Skill 评审发现与整改台账

> 评审时间 2026-11-10；范围：`skills/establish-pdu-session-mas` 及当前 main 的 Runtime/Tool/Host 文档；对照 MAS 方案 C v0.3.3（第 3 轮仍待人工确认）。**本页是修复计划与证据索引，不代表所列问题已关闭**。具体通用要求见 [编写规范](skill-authoring-review-rules.md)，外部步骤矩阵见 [MAS 对照](mas-scheme-c-contract-gap-matrix.md)。

> **对应源码整改**：[#184 Draft · 完整 14 步建立候选 + entry eligibility + 16 处原生 Reconcile + 6 个合成 Cases](https://github.com/odango-chan/ca-agent-demo/pull/184) 已提交候选修复；目前只通过 GitHub 回读/静态交叉检查，**pytest、Toolchain compile、MAS Mock/Live 未运行**。F01–F05/F11/F12 的源码/测试提案已经进入 #184，但仍是 `PROPOSED / NOT VERIFIED`，不是 CLOSED。两 PR 独立分支，合并前需分别复审。

## 1. 分级及关闭策略

- **P0 / SOURCE BLOCKER**：当前候选 Skill 内部的语法/机制不相容或业务流程不相容；不修不能把候选称为可执行。
- **P1 / CONTRACT BLOCKER**：需要外部 MAS 或 Host 提供接口/Owner 证据后才能 READY。
- **P2 / AUTHORING DEBT**：模型可读材料、名称、测试/可维护性欠缺；不应成为永久实现习惯。
- **M**：MAS 设计自身仍开放的问题（不擅自将本仓建议认定为 MAS 事实）。

## 2. 已确认的源材料问题

| ID | 级别 | 定位 | 现象 / 证据 | 目标修复（与完成判据） | 状态 |
|---|---|---|---|---|---|
| F01 | P0 | `assurance/procedure.yaml` 各 `reconcile-*`；`contract.yaml`；`design/tool-contract.yaml` | 把 `ca.pdu_session.operation.reconcile` 注册为 Effect=RECONCILE 普通 `call`；现有 `runtime/tool_contract.py` Tool Effect 只允许 READ/MUTATION，`runtime/engine.py` 使用 `reconcile: original-node` | 逐个原操作原生 Reconcile、独立对账输出 Schema、原操作身份和未完成责任；无适配时仍 NOT_READY | **PROPOSED PR #184 / UNVERIFIED** |
| F02 | P0 | `assurance/procedure.yaml`: `start: validate-subscription`；`contract.yaml` `preconditions` | Root `mas.establishment.eligible` 仅声明，首节点可能 MUTATION；当前 `_check_hard_preconditions` 对 ROOT 跳过，`admit_skill_entry` 未执行 Predicate | 首个可能 Mutate 前显式 `verify eligible`；UNKNOWN/FAIL 0 dispatch；后续可统一根前置机制 | **PROPOSED PR #184 / UNVERIFIED** |
| F03 | P0 | `SKILL.md` + `assurance/procedure.yaml` | 先 Anchor/Address 后 `policy.create`；MAS policy-first 认证 Plan 第 5 步 PA.CREATE 在锚/地址之前，地址后第 8 步 PA.UPDATE | 显式两次调用与对应 `with/from`、状态判断、错误及对账；需 MAS schema 对齐并跑流程对照 | **PROPOSED #184 / MAS SCHEMA OPEN** |
| F04 | P0 | `assurance/procedure.yaml`；`design/tool-adapter.md` | 未表达 MAS s2 SAA、s6u PA UPDATE、s9 IAM CONFIRM、s13 SE EMIT；`register-subscription` 混合 SDM 与 UECM；对应认证计划不等价 | 对每个认证 Op 明确存在/可跳过条件和真实 owner；不得从“当前 profile 不需要”直接删掉已认证步骤 | **PROPOSED #184 / MAS DECISION OPEN** |
| F05 | P1 | `design/tool-contract.yaml` | #182 每一步只给 `operationRef/stateRef`，缺 `sub_summary, session_keys, policy_assoc_id, ue_addr, n4_seid, an_tunnel_*` 的类型化消费证明 | 按 MAS `sm-mas-ops/3.2` 建字段级、producer 级映射；允许受信 stateRef 间接投影，但必须可校验 | **PROPOSED #184 / TYPE+OWNER OPEN** |
| F06 | P1 | `assurance/predicates.yaml` | 全部同一 `mas.establishment.snapshot` 布尔字段，当前没有真实 Observation 后端；错实例、旧证据可能被错误概括为 true | 每个布尔字段证明权威 producer、operationRef、instance generation、freshness/correlation，未知保持 UNKNOWN；负例通过 | **OPEN / Observation** |
| F07 | P1 | `contract.yaml` / `design/tool-contract.yaml` | 原先 13 个、#184 提案 16 个 Tool 仅为设计文件，没有已登记 MAS Effect/Implementation/READY；当前默认注册仍针对旧 CreatedSession | 独立 expected Effect、真实 implementation、启动时冻结 Tool Registration，缺项 fail-closed | **OPEN / Host+MAS** |
| F08 | P1 | `assurance/procedure.yaml`: cleanup/finalize；`SKILL.md` | `SC.COMPENSATE_RELEASE` 设计为 best-effort，但候选程序要求 cleanup_complete/resources_managed；MAS 尚未提供逐资源持久责任结果 | 明确谁拥有残余资源与迟到结果，重启后有原操作查证；凭据不足不报 PASS | **OPEN / MAS resource owner** |
| F09 | P1 | `SKILL.md` / `contract.yaml` + `schemas/establishment-request.schema.yaml` | 当前 flat Fabric 请求与 MAS Nsmf Ingress 的字段集合不同，`pti, requestType, servingPlmn, requestedSscMode` 来源尚未一一对齐 | 字段来源矩阵；合法源缺失时明确拒绝，不编造 `SSC1/IPV4` 默认 | **OPEN / Host ingress** |
| F10 | P2 | `contract.yaml` `createdsession_ingress/lifecycle`；`predicates.yaml` scopes | 新 Skill 继承了旧场景名称而不是中性 Subject/Ingress/Lifecycle；改名不能只改 YAML 而不改 trusted ingress 和 owner observation | 在 Host / ingress adapter 建稳定中性 facts，按版本迁移并保持旧 Skill 不受影响 | **OPEN / WP Host dependency** |
| F11 | P2 | `SKILL.md` | 包含 Procedure 文件路径/Adapter 机制/Host 内部词，模型文档业务目标和当前“policy-first”实际决策不一致 | 将模型文档限制为领域含义、事实、禁止推断、完成/失败原则；运行机制移入 `design/` | **PROPOSED PR #184 / UNVERIFIED** |
| F12 | P1 | `cases/`（#184 新增 6 个合成 Case） | 尚缺完整规范化 Qualification 覆盖：每个 op 失败、UNKNOWN、错关联/迟到、非法 Profile、取消、预算 | 最小 Case set+真实 schema/compile；不足则仅设计稿、不声称 qualification | **OPEN / tests** |
| F13 | P1 | `contract.yaml` / 统一 Host Route | MAS Skill 当前未进入 RootRoutingProfile / default registration，且 SMF `agent_step` 要求同会话防 OES/Agent 双编排 | 按部署只启用一个 Root、一个准确的 execution owner；未覆盖时 NOT_READY | **OPEN / deployment** |

## 3. MAS 方案 C 本身的待闭环问题

| ID | MAS v0.3.3 原评审 | 接口/方案的实质缺口 | 本仓接受条件 | 当前结论 |
|---|---|---|---|---|
| M01 | R-3-01 / M-3-01，Critical | SIGNAL park 结束只回调 OES 完成槽，而方案 C `agent_step` 禁 OES Execute | 没有 `smf_oes_sess_active` 也能经 I-SB OnSouthbound 结束**原 I-INV**，真实网络正负例 | **BLOCKED_EXTERNAL** |
| M02 | R-3-03，Major | 旧 Nsmf 入 SMF 可能误触发 OES Start/Execute 双 cursor | `agent_step` 下除 outstanding 过程 Update/FailureNotify 与 I-ACT 例外外明确拒绝/旁路 | **BLOCKED_EXTERNAL** |
| M03 | R-3-04，Major | MODIFY Plan 无 `on_error.compensate` | 无补偿时明确失败终态，未知副作用仍留责任；不调用 Establish 补偿 | **BLOCKED_EXTERNAL** |
| M04 | R-3-02 / v0.3.3 | `amf_ref` 已更名 `sm_context_ref`（文档声称关闭） | MAS 当前 Catalog/真实 I-INV 返回证明，Host 以本键形成 SMF Location | **DESIGN_CLOSED / LIVE_NOT_VERIFIED** |
| M05 | `3.3.2`/`4.9`（补充 gap） | MAS 504/非2xx 未自然证明 NOT_COMMITTED；没有原 operation status read 详细 contract | idempotencyKey exact original 查询、超时迟到与跨重启责任 | **BLOCKED_EXTERNAL** |
| M06 | `1.5 A10`（补充 gap） | 方案 C Agent Assoc/Ledger 内存，重启丢状态 | 本仓 Journal/owner 维持 unknown mutation/late resource 责任；内存仅缓存 | **ARCHITECTURE_OPEN** |
| M07 | `3.3.1` + 本仓现行架构 | MAS 要 Nsmf 兼容 Facade；本仓现行唯一生产 Fabric SBI | 同一 Host 的协议 ingress adapter 和受控部署模式切换；不产生第二 Root/Journal/Runtime | **ARCHITECTURE_OPEN** |

## 4. 推荐的 PR 切分（不相互冒充进度）

| PR/阶段 | 改动边界 | 可单独关闭哪些点 | 绝不能宣称 |
|---|---|---|---|
| D0（当前设计 PR） | `design/` 规范与对照、模型业务文档候选修订方案 | 文本审计覆盖 | Capability/Source/E2E READY |
| S0（#184 Source Candidate） | entry eligibility、16 处原生 Reconcile、完整 14 步 MAS certified Establish、字段级绑定、16 个 Tool 草案、17 个 Predicates、6 个合成 Cases | F01–F05/F11/F12 的**源码提案**，外部真实 Schema/Adapter 仍 OPEN | MAS policy-first flow 真实运行成功 |
| S1（等待 MAS/Host 接口确认） | 固定 mas/op/inputs/outputs 的真实类型、外部 owner 证据、Effect/Observation/backend/Host ingress，扩展 Cases 实测 | F03–F10/F12 的**运行证据和接口闭环** | 以模拟或源码对齐假称 LIVE PASS |
| H1（Host 模式） | 统一 Ingress/RootRoutingProfile/双执行互斥/责任 | F10/F13、M02/M06/M07 | 自动让 MAS M01/M05 成立 |
| L1（联合联调） | MAS M01/M04/M05、真 AMF/RAN/N4、重启/迟到 | 可验收剩余 P1 | 用 mock PASS 代替生产 E2E |

## 5. 复审清单

评审完必须回答：Scope 有没有漂移？Schema 与真实 API 是否一一对应？每个 allowed Tool 可否由 frozen backend 精确实现且 Effect 正确？每个结果字段是否带 producer/instance/operation 关联？UNKNOWN 是否始终留原引用？结束后哪些资源谁负责？缺必要接口时是否 NOT_READY？未来 LLM 是否仍走同一工具和证据边界？只有回答有实测/编译证据才关闭对应项。
