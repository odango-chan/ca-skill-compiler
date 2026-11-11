# 5GC Executable Skill 编写与评审规范（MAS 试点）

> 状态：**本仓候选评审规范 / Design**，2026-11-10。适用于当前 ca-agent-demo 的 5GC Skill 原型，尤其 `establish-pdu-session-mas`。不另造正式 Toolchain Schema 或生产平台规范；正式 authoring authority 仍为 `odango-chan/ca-skills/docs/authoring.md`、`ca-skill-toolchain` 的已 pin Schema/Compiler；现有 `AGENTS.md` 与统一 Host 架构优先。本文不宣称实现/测试通过。

## 1. 规范目标与适用范围

一个 Skill 的业务目标与安全完成条件要在**确定性执行和未来 LLM Skill Executor** 下具有同样的含义。当前可执行的 deterministic Procedure 是先期路径；完整自由式 LLM executor 延期但保留。**Agent Host = MAS 方案 C 的 NW-Agent**，绝无第二个专用 MAS Agent。Skill **不是**一份 HTTP API 文档、Adapter 接口、SMF OES Workflow 或 Host/Runtime 行为指南。

### 强制资产与唯一责任

| 资产 | 仅应承担 | 不得承担 | 评审方法 |
|---|---|---|---|
| `SKILL.md` | 模型可读的适用条件、业务目标、须知道的事实、不得推断的事情、成功/失败语义 | 原始 MAS `mas/op`、URL、token、机器 Binding、Runtime 机制教程、状态打印或迁移说明 | 完整 LLM 接入后模型是否只学会业务语义而非绕过 Tool |
| `assurance/contract.yaml` | exact `entryProcedure`、request Schema、必要 facts、Capability 上界、前后条件、预算/失败语义 | Tool effect 的第二份审批权威、部署 endpoint、其他 Skill 专用流程 | 与本 Skill 的 Procedure/Predicates 一致且可机器校验 |
| `assurance/procedure.yaml` | 确定性业务步骤/真实前置依赖、显式 `with/from`、可执行成功/失败/未知分支、收尾 | 对外 wire 伪造、无授权副作用、业务事实隐含在 Adapter、第二个 Plan cursor | 全路径遍历；每条外部调用有正确 Effect 和输出来源 |
| `assurance/predicates.yaml` | 从权威 Observation 确定 PASS/FAIL/UNKNOWN | 以 HTTP 200、Tool 回执或 model 判断直接给出 PASS；隐式网络读 | producer、scope、freshness、实例代次、负例 |
| `schemas/` | Skill 真正拥有的请求/业务 payload（如需） | 复制 Host model response 或 MAS 外部 API 全套 Schema | 缺字段、版本、范围与命名一致 |
| `scripts/` | 必需时做被准入的纯本地业务计算/参数构造 | 网络调用、持久 Journal、调度下一个节点 | 受治理 script metadata / freeze / 可测试性 |
| `references/*.md` | 模型选读的业务补充材料（可无） | 运行时治理权威、密钥、机器契约、默认加载所有知识 | 被 `SKILL.md` 精确引用，Host 路径/digest 校验；不造空目录 |
| `cases/<case>/case.yaml` | 正负与不确定情形的 Skill Qualification | Benchmark 答案泄漏、声称等于 Live E2E | PASS/FAIL/UNKNOWN、未知写、预算、取消/迟到、错误关联 |
| `design/` | 人工跨仓对照、待裁决事项、演进和 Adapter 方案 | 运行时批准的 Tool/Effect/Predicate 权威 | 明确 DRAFT vs READY，OPEN 项有 owner 与关闭证据 |

**特殊说明**：ToolCatalog 与 CapabilityEffect 的生产归属分别为本仓 `tools/contracts/`、独立部署 approved Effect 与注册；`design/tool-contract.yaml` 是草案。模型读取 `SKILL.md` 不意味着获得任意 Tool 权限。

## 2. 可执行机器资产的 20 条审查规则

### A. 身份、目标、边界

- **S01 类型与粒度**：可执行业务目标才能叫 Skill。单次 UDM/PCF/MAS Op 是 Capability，不因已有 endpoint 就建一个 Skill。路由-policy（如 `5gc-intent`）不伪装成执行 Skill。
- **S02 目标唯一**：一个 Skill 说明唯一可判定业务目标、支持的 profiles 和所有不支持模式；`establish-pdu-session-mas` 当前只负责 initial establishment，不能暗称 Modify/Release/ACTIVATING 已支持。
- **S03 单入口/单执行 owner**：`contract.spec.entryProcedure` 与 Procedure 真正入口一一对应、一次请求只准入一个 exact artifact；SMF `agent_step` 禁 OES Execute 同时推进同会话。架构允许多 Skill 与多执行策略，但一次 invocation 不由两个 cursor 执行。
- **S04 上下文与身份**：Runtime 只传受信冻结的 generic invocation；Skill 业务字段通过类型化 `context.subject/request/facts` 与 `input` 显式绑定。不要新添 `createdsession.*`、MAS、Fabric 的 Runtime 名字特判；历史输入先在 ingress adapter 兼容。
- **S05 无编造参数**：请求缺 PTI、PLMN、SSC、session type 时，从批准的协议或配置证明来源；不能从 `plan` 或 `SKILL.md` 猜默认值。拒绝不受支持/歧义的请求须在**首个可能副作用前**生效。

### B. 业务流程与数据

- **S06 与外部认证计划对齐**：如果依赖现有 MAS 认证 Plan（`oes-dsl/1.3` + `sm-mas-ops/3.2`），逐步列出 `mas/op`、真实 `inputs/outputs`、依赖、可跳过条件以及 Host Skill Capability。不能仅按语义近似名称合并有独立副作用的阶段。
- **S07 真实依赖决定顺序**：业务顺序取外部契约及输入依赖，不为图形简化强制线性屏障，也不能把 `PA.CREATE_SM_POLICY` 与 `PA.UPDATE_SM_POLICY` 合成掩藏副作用的一步。
- **S08 显式 producer**：用于下一步的 `sub_summary, session_keys, policy_assoc_id, up_intent, ue_addr, n4_seid, an_tunnel_*` 必须明确来自**哪个** MAS Op，以及与本会话的绑定。`stateRef` 可是只读句柄，但必须证明具体字段而非仅“有一个句柄”。
- **S09 资源权限边界**：Skill 用稳定 Capability；Adapter 负责 MAS `mas/op` 和 HTTP wire 投影，Host 不自行直连 UDM/PCF/UPF。不暴露 URL/token/环境拓扑到 Skill。
- **S10 独立 Effect Authority**：Capability Contract 的 expected Effect（READ/MUTATION）不能由 Skill 作者通过正文降低。可能创建本地会话、预留或订阅的“validate/select”保守按 MUTATION，直到 MAS owner 明确证明纯读。

### C. 三态、证据、副作用

- **S11 Mutation 输出 ≠ 业务 PASS**：成功回执只证明调用有结果；必须经过独立新鲜 Observation 判定 Predicate。Namf Transfer 接受≠RAN 已采纳，`CREATE_UP_SESSION`≠N3 下行可用。
- **S12 Predicate 三态**：权威源/producer/operation/session instance generation/关联/时效任一缺失或冲突 → UNKNOWN；不要将 Missing/Stale 当 PASS 或简单 FAIL。任一当前会话 snapshot 布尔值必须可追踪到真实权威。
- **S13 业务前置检查必须可执行**：Contract 声明的资格不能只停留文档/metadata；Root 首个 Mutation 前必须有实际 admission/`verify` 门禁。验收“资格拒绝 = 0 外部 Mutation”。
- **S14 Outcome Unknown 不自动重试**：MAS 504、断连、未知 4xx/5xx、无效回执等**不能**仅按 HTTP 码判断 NOT_COMMITTED。保留原操作身份/幂等键，走原操作 `reconcile: <original-node>` 或 UNRESOLVED/SAFE_STOPPED；重启后亦必须有 durable responsibility。
- **S15 Reconcile 不是普通业务 CALL**：Runtime 的 `reconcile` Node 针对**原 operation identity**，Backend 的 `reconcile()` 返回 `OperationEffect.RECONCILE`；`ToolContract` 与独立 Effect 只登记 READ/MUTATION。Skill 不新增 `call: ...operation.reconcile` 来绕过机制。
- **S16 资源补偿与责任不等价**：`SC.COMPENSATE_RELEASE` 是 MAS 尽力补偿入口；若没有逐资源清理/接管证据，不能让 `cleanup_complete` PASS 或把失败归零。所有已可能创建资源、迟到结果需在原 owner 或 Host Journal 可继续追踪。
- **S17 “已知失败”和“未知副作用”分流**：明确未发送/未提交可进入已知失败；未知写先保留责任且不可直接使用新幂等键执行破坏性补偿。无 `on_error.compensate` 的 MODIFY 不能隐式复用建立补偿。
- **S18 External callbacks 不造新协议**：MAS `SSM.SIGNAL_UE_AN` 挂起期间的 I-RLY/I-SB 必须由同一 Host 并发处理；Namf 和 Nsmf 原消息透传（含 multipart），不包装成自定义 `Southbound` payload，也不能因 F-TEID Update 新启 MODIFY。
- **S19 终态有独立事实**：SUCCEEDED = 本次 RAN 采纳 + 同一 N4 实例下行转发 ready + 资源责任已接管；无 Source 证明则 UNRESOLVED，不由 LLM 或端口 200 宣称 SUCCEEDED。
- **S20 可维护/可演进**：修改 Skill 时同时更新 Case 与设计对照，记录外部契约版本、实际 Effect、差异与逆向兼容；LLM 执行未来接同一 admission/action/evidence，而不是第二套副作用机制。

## 3. 规范化的失败分支

```text
before any mutation:
  applicability / eligibility / exact identity / schema / readiness
  FAIL/UNKNOWN -> FAILED or UNRESOLVED; ZERO mutation

one admitted MUTATION:
  COMMITTED -> independently verify stage Predicate
      PASS    -> next stage
      FAIL    -> known failure + governed cleanup (only with ownership facts)
      UNKNOWN -> UNRESOLVED; responsibility retained
  known NOT_COMMITTED / DENIED -> governed failure path (no false resource claim)
  OUTCOME_UNKNOWN -> reconcile exact original operation
      COMMITTED / NOT_COMMITTED -> SAFE_STOPPED + preserved provenance
      UNRESOLVED -> UNRESOLVED + owner responsibility, never blind retry

completion:
  each Contract postcondition independently PASS
  finalizer receives durable responsibility evidence before SUCCESS
```

此为语义规范，不替代实际 Runtime `MutationStatus`、Procedure authoring schema 或运行时 `finalizer`；例如哪个结果可过渡到 cleanup 仍由 admitted graph 和可信 owner 明确决定。

## 4. MAS 试点的评审闸门（测试分层）

| Gate | 检查 | 失败处理 |
|---|---|---|
| G0 Source | SkillContract/ProcedureSource/PredicateSet/Case schema；入口、引用、edges、Effect 与 Tool required bindings 静态一致 | 不能提交“可执行 Skill 已就绪” |
| G1 Domain | 认证 Plan 逐步映射，Policy-first 顺序，SAA/UECM、PA Update、IAM Confirm、SE，成功/异常分支齐备 | 标为 BUSINESS_SEMANTIC_MISMATCH |
| G2 Authority | 每项 Tool/Observation 及 MAS owner 证据、独立 Effect 与注册；UNKNOWN 与重启责任 | 标为 NOT_READY |
| G3 Mock | 资格拒绝 0 Mutations、输入差错、错 producer/session、超时迟到、RECONCILE 原操作、清理残留、N1N2/Update 竞态 | 不标 MOCK_PASS |
| G4 Live | MAS SBI + AMF/RAN + 原 N4 实例与 SMF 状态，真实失败/超时，以及 OES 关闭时 SIGNAL 仍可完成 | 不标 LIVE_E2E_PASS |
| G5 LLM（延期） | Model 选择的是已准入动作，无法伪造业务 PASS，不可绕过 Effect/Owner；不擅自启用生产 | 仅 DESIGN_OPEN |

**报告纪律**：标明 source-version、main/head、实际运行的命令与 exit code、mock/live 区分；不能以官方 Toolchain compile PASS 代替 MAS E2E。没有新的 Rule Schema 时，不增加 `rules.yaml`、`SkillRuleSet` 或 `ruleRefs` 作为生产机器资产。

## 5. 当前适用文档

- 本仓：[方案 C 单 Host 架构](agent-host-scheme-c-architecture.md)、[MAS 步骤差异](mas-scheme-c-contract-gap-matrix.md)、[具体评审发现](skill-review-findings-20261110.md)。
- 官方 authoring：`odango-chan/ca-skills/docs/authoring.md`（本仓参考，不在此复制成第二份官方规范）。
- 语法来源：本仓 pin 的 `tests/contracts/ca-skill-toolchain/<sha>/` 及实际 Toolchain Compiler；本目录表格不能授权不存在的 Machine Schema。
