# 附页：最小示例、字段来源与跨图追踪

> 配套图：`01-toolchain-panorama`、`02-offline-build`、`03-host-admission`、`04-scp-amf-smf`、
> `05a-establishment-sequence`、`05a-ops-establishment-capabilities`、`05b`–`05g` 六张业务时序图。
> 导航入口：[index.html](index.html)。
> 状态：**设计附录（DESIGN / NOT IMPLEMENTED）**。现行基线是冻结的 CA-SL v1alpha5 与已发布的 Compiler v2.0.2；
> 文中标注“设计草案”的 Schema、Manifest、锁字段均**不是**已发布契约，也不是 v2.0.2 的现行输出。
> 本附录只服务评审，不构成 E2E 通过、Owner 恢复完成或生产资格结论。

## 1. 全局约定

1. 统一案例名：`session-demo`。四张图使用同一组追踪摘要标记。
2. 摘要标记是**追踪占位符**，省略号不是摘要值：

   | 标记 | 指向 | 出现在 |
   |---|---|---|
   | `sha256:…mas` | MAS Interface Package 快照（含 Capability Catalog 成员） | 图 1、图 2 |
   | `sha256:…ctx` | `ca-context` / Ingress Subject Context 快照 | 图 1、图 2 |
   | `sha256:…artifact` | Linked Development Artifact（manifest 摘要） | 图 1、图 2、图 3 |
   | `sha256:…provider` | Runtime Provider Package / 已部署实例绑定 | 图 1、图 3 |

   实际构建记录必须写入真实摘要；不得把 `sha256:…mas` 当作校验和提交。
3. 图中出现的 JSON、YAML、锁文件片段都是**节选**，用于说明字段来源与责任边界，不可独立编译或直接发送。
4. Requirement 口径统一为两类：`HOST_VERIFICATION_REQUIRED`（派发前外部事实，`host-admission-requirements.json`）与 `RUNTIME_EXECUTION_REQUIRED`（执行期真实结果，`runtime-execution-requirements.json`）。`COMMIT_PROOF` / `RECONCILE` 只属于后者，图 1–图 3 不得把它们画成 Build 或派发前可满足的 Host Admission 条件。

## 2. 图 1 / 图 2 的最小示例

### 2.1 真实 MAS Procedure 局部节选

来源：`ca-skills` `examples/skills/establish-pdu-session-mas/assurance/procedure.yaml`（CA-SL v1alpha5）。
以下为**局部节选**，省略了 Pre/Postcondition 与其余节点，**不可独立编译**；完整字段、路径与 outcome 以 ca-skills v1alpha5 源文件为准。

```yaml
start: validate-sm-subscription
nodes:
  validate-sm-subscription:
    operation:
      tool: mas::SC.VALIDATE_SUBSCRIPTION
      arguments:
        ue_identity: context.supi
        psi: context.request.pduSessionId
        dnn: context.request.dnn
        sNssai: context.request.sNssai
      routes:
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: UNRESOLVED
        DENIED: FAILED
```

要点：

- `context.supi` 是**身份**，不是 SBI JSON 的业务字段；Host 在准入后绑定，不交给 Skill 自行推断。
- `context.request.*` 的四个字段分别来自 SBI JSON 与 NAS 解码（见 §3.2）。
- `mas::` 前缀的 Callable 目前仍是**候选引用**：没有受信任的 MAS ToolCatalog / CapabilityContracts 公开签发。
- `OUTCOME_UNKNOWN: UNRESOLVED` 中的 `UNRESOLVED` 是冻结 CA-SL 语言的 route 目标名，**不是** Requirement 状态；Requirement 状态只有 `HOST_VERIFICATION_REQUIRED` 与 `RUNTIME_EXECUTION_REQUIRED` 两种。

### 2.2 `dependencies.lock` 与两类 Requirement 记录（设计草案）

```jsonc
// 设计草案：字段名与形状尚未冻结，v2.0.2 不产出该格式
{
  "roots": {
    "mas": { "packageId": "demo/mas", "version": "0.1.0", "snapshotDigest": "sha256:…mas" },
    "ctx": { "packageId": "demo/ca-context", "version": "0.1.0", "snapshotDigest": "sha256:…ctx" }
  }
}
```

```jsonc
// 设计草案：host-admission-requirements.json 单条节选
{
  "requirementId": "sha256:…",
  "kind": "OWNER_TRUST",
  "location": { "skill": "establish-pdu-session-mas", "path": "assurance/procedure.yaml",
                "node": "validate-sm-subscription", "stage": "postcondition", "selector": "/nodes/…" },
  "subjects": [{ "key": { "packageId": "demo/mas", "version": "0.1.0", "snapshotDigest": "sha256:…mas" },
                 "id": "mas::SC.VALIDATE_SUBSCRIPTION", "sourcePath": "callables.yaml",
                 "schemaIds": ["demo/mas/schema/…"] }],
  "status": "HOST_VERIFICATION_REQUIRED"
}
```

```jsonc
// 设计草案：runtime-execution-requirements.json 单条节选
{
  "requirementId": "sha256:…",
  "kind": "COMMIT_PROOF",
  "location": { "skill": "establish-pdu-session-mas", "path": "assurance/procedure.yaml",
                "node": "create-sm-context", "stage": "postcondition", "selector": "/nodes/…" },
  "subjects": [{ "key": { "packageId": "demo/mas", "version": "0.1.0", "snapshotDigest": "sha256:…mas" },
                 "id": "mas::SC.CREATE_SM_CONTEXT", "sourcePath": "callables.yaml",
                 "schemaIds": ["demo/mas/schema/…"] }],
  "status": "RUNTIME_EXECUTION_REQUIRED"
}
```

- `HOST_VERIFICATION_REQUIRED` 只覆盖派发前可验证的外部事实（`OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST`、Deployment Grant、Provider Binding、Invocation 授权）。
- `RUNTIME_EXECUTION_REQUIRED` 只覆盖执行期才产生的真实结果（`COMMIT_PROOF`：实际提交与证明核验；`RECONCILE`：`UNKNOWN` 后精确原尝试的恢复与实际结论）。它们**不是**可在 Build 或派发前满足的 Host Admission Requirement，真实结果也不回写产物。
- 两种状态都不是执行授权。静态结构非法是 Build FAIL；只有外部授权缺失才是 Host Requirement。

`EVIDENCE_INDEPENDENCE_TRUST` 另带 `independence{groupIndex,members[{dependencyAlias,key,exportId,sourcePath,domain}],domainPairs}`，一组一条，`selector` 形如 `/predicates/<predicate>/evidence/independence/<index>`；不使用 `env.roots[0]` 或单一全局布尔值。

### 2.3 Artifact manifest（设计草案）

```jsonc
// 设计草案（compiler-development-artifact / v1alpha1）：不是 v2.0.2 的现行产物
{
  "apiVersion": "connectionagent.io/compiler-development-artifact/v1alpha1",
  "languageTarget": "connectionagent.io/v1alpha5",
  "generator": { "product": "ca-skill-compiler", "version": "vNext (proposal)" },
  "profile": "DEVELOPMENT",
  "sourceDigest": "sha256:…mas",
  "dependencyLock": { "roots": ["mas", "ctx"] },
  "procedures": ["establish-pdu-session-mas"],
  "links": { "callables": ["mas::SC.VALIDATE_SUBSCRIPTION"], "ingress": [], "capabilityCatalog": "sha256:…mas" },
  "requirements": ["host-admission-requirements.json", "runtime-execution-requirements.json"],
  "executionState": "NOT_ADMITTED"
}
```

**重要区分**：图 2 的 `LINKED / BUILT / NOT_ADMITTED` 是**设计目标状态**。
Compiler v2.0.2 没有该 Development Build 入口，也不得被描述成已经支持上述 manifest。

## 3. 图 4 的最小示例

### 3.1 AMF 经 SCP 发来的 Create（结构示意）

```text
POST /nsmf-pdusession/v1/sm-contexts                     HTTP/2
:authority = <SCP 的下一跳 authority>；Target-apiRoot = Host Nsmf apiRoot
Content-Type: multipart/related; boundary=...

JSON part:
  { "supi": "imsi-xxxxxxxxxxxxxxx",     // 已脱敏，非真实 SUPI
    "pduSessionId": 10,
    "dnn": "internet",
    "sNssai": { "sst": 1 },
    "requestType": 1,
    "n1SmMsg": { "contentId": "n1-part" } }

binary part:
  Content-ID: <n1-part>
  Content-Type: application/vnd.3gpp.5gnas
  payload: 原始 N1 NAS 字节
```

这是**结构说明**，不是可直接发送的完整 3GPP multipart 报文。Authorization Header、OAuth scope、
`3gpp-Sbi-*` 头与 3GPP 版本协商按锁定 OpenAPI 处理，图中不逐一展开。

### 3.2 解码后交给 Skill 的类型化请求（示意，非 CA-SL 语法）

```yaml
# session-demo / 业务请求投影（字段来源已标注）
request:
  pduSessionId: 10          # SBI JSON
  pti: 1                    # N1 NAS 解码
  requestType: 1            # SBI JSON
  dnn: internet             # SBI JSON
  sNssai: { sst: 1 }        # SBI JSON
  requestedPduSessionType: IPV4   # N1 NAS 解码
  requestedSscMode: 1       # N1 NAS 解码
```

字段与 `establishment-request.schema.yaml` 对齐；每个字段保留来源并交叉校验。原始 N1 字节**不进入** Skill 参数。
Skill 只消费这些类型化字段：

```yaml
# session-demo / 局部节选，说明字段如何被消费
nodes:
  create-sm-context:
    operation:
      tool: mas::SC.CREATE_SM_CONTEXT
      arguments:
        ue_identity: context.supi
        psi: context.request.pduSessionId
        dnn: context.request.dnn
        sNssai: context.request.sNssai
```

### 3.3 201 与后续路由（M1 目标）

```text
AMF → SCP → Host       Create
Host  → SCP → AMF      201 Created
                       Location: https://<Host 外部 apiRoot>/nsmf-pdusession/v1/sm-contexts/<host-ref>
AMF → SCP → Host       POST .../<host-ref>/modify     （同一 Host 资源）
Host → Client Tools → OpenAMSM   Owner-private 固定绑定
OpenAMSM → Host Namf Relay → Client Tools → SCP → AMF    Namf N1N2
```

- `Location` 的主机名与 path 前缀属于 **Host**；`<host-ref>` 与 OpenAMSM 内部 `<owner-ref>` 的映射持久化在
  Host SM Context URI Map，SCP 不保存该映射。
- M0（201 指向 OpenAMSM）只是历史过渡对照，不是本轮验收 Profile。

### 3.4 NAS 的两条路径

| 路径 | 现状 | 数据 | 用途 |
|---|---|---|---|
| A：解码投影 | 设计已定，实现未完成 | 原始 N1 → Artifact Store → 只读 Codec → `request.*` | 让 Skill 能按类型化字段编排 |
| B：Owner 权威输入 | **未实现**（图 4 虚线） | 原始 N1 → Owner-private 受信绑定 → OpenAMSM GSM Handler | 让 SMF 消费 PTI / PDU Type / SSC / ePCO 等完整 NAS 语义 |

只完成 A 不能宣称 NAS 语义保真的 E2E。B 由 ca-client-tools #18 跟踪。

## 4. 跨图追踪表

| 概念 | 图 1 | 图 2 | 图 3 | 图 4 |
|---|---|---|---|---|
| `session-demo` 源 | Interface Package | 构建请求 | Artifact 输入 | Skill 节点 `establish-pdu-session-mas` |
| `sha256:…mas` | Interface Package 标签 | Interface 快照 | — | — |
| `sha256:…ctx` | `dependencies.lock` 标签 | Interface 快照 | — | — |
| `sha256:…artifact` | Linked Skill Artifact | Linked Development Artifact | Artifact 入站材料 | — |
| `sha256:…provider` | Runtime Provider 绑定 | — | Provider 绑定 | — |
| LINKED / BUILT | Artifact 标签 | Artifact 标签 | 准入前提 | — |
| NOT_ADMITTED | Artifact 标签 | Artifact + Host 拒绝派发 | 生产 Host 拒收 | — |
| HOST_VERIFICATION_REQUIRED | 由 Host 补齐的外部事实 | `host-admission-requirements.json` | 准入前逐项验证 | — |
| RUNTIME_EXECUTION_REQUIRED | 不在编译区满足 | `runtime-execution-requirements.json` | 派发前只查机制就位 | 执行期产生 `COMMIT_PROOF` / `RECONCILE` 结果 |
| UNKNOWN | — | — | 结果 UNKNOWN → Reconcile | Owner pending / 无 proof |
| SCP | 不出现（离线区） | 不出现 | Invocation Context 来源 | 唯一网络入口 |
| 201 Location | — | — | — | Host SM Context URI Map |

## 5. 与两条 PR 基线的逐项一致性

对照 `ca-skill-compiler` PR #48（工具链架构 + 绘图简报）与 `ca-agent-demo` PR #217（Host SBI / SCP 路由）。

| # | 约束 | 落点 | 结果 |
|---|---|---|---|
| 1 | Interface Package 与 Runtime Provider 独立发布 | 图 1：两条发布链；`dependencies.lock` 与 Provider 绑定分开 | 一致 |
| 2 | Capability Catalog 与 Deployment Grant 分离 | 图 1：Catalog 随包发布（`导出`），Grant 由部署方签发（`只交 Host`） | 一致 |
| 3 | 缺 Evidence 可生成 Artifact，但保留两类 Requirement / NOT_ADMITTED；缺 Interface 或静态结构非法则 Link FAIL 无产物 | 图 2 两条失败/成功分支与两个 Requirement 文件节点 | 一致 |
| 4 | 不得把 v2.0.2 说成已支持新 Development Build | 图 1/图 2 副标题与卡片、本附录 §2.2/§2.3 | 一致 |
| 5 | Host 只接管 AMF–SMF SBI，不替代 OpenAMSM 的 SMF 业务 | 图 4：`OpenAMSM SMF` 保留 5GSM/UDM/PCF/N4 | 一致 |
| 6 | 所有 H1/H2/H3 网络测试必须含 SCP | 图 4：SCP 为唯一网络入口，无直连/无 Direct Mock Relay 路径 | 一致 |
| 7 | AMF → SCP → Host；OpenAMSM → Host 私有 Namf Relay → Client Tools → SCP → AMF | 图 4：`Nsmf 入站` 与 `Namf N1N2 / 正式 Namf（经 SCP）` 两条独立连线 | 一致 |
| 8 | Host → OpenAMSM 是受信内部绑定；OpenAMSM → UDM/PCF/CHF 仍走原 SCP/NRF | 图 4：`Owner 私有绑定` 与 `既有 SBI（经 SCP）` | 一致 |
| 9 | NAS 解码进 Skill；原始 N1 保留并传给 Owner | 图 4：`store`, `codec`, `ctx`, `skill` 与虚线 `原始 N1（未实现）` | 一致 |
| 10 | 201 Location 指向 Host，后续 Update/Release 回到 Host | 图 4：`URI Map` 标签与 `201 + Location` 连线 | 一致 |

补充核对：

- **客户端不接收 AMF 请求**：图 4 中 Client Tools 只有出站箭头（`MAS 调用`、`Nsmf 转发`、`正式 Namf（经 SCP）`），
  入站接收者是 Host HTTP/2 SBI Listener 与 Host Namf Relay Ingress。
- **AMF 不会发现第二个 SMF 候选**：图 4 只画一个 AMF-facing Host apiRoot；OpenAMSM 的
  `Owner-private apiRoot` 标注为内部端点且不对 AMF 发布。
- **Owner Recovery 未完成**：图 3 的 Owner Pending 明确为进程内 waiter（30 秒上限、非 Temporal 恢复）；
  图 4 的 Runtime 标注 `不恢复 Owner waiter`。本附录不宣称恢复能力。

## 6. 未实现 / 未验收边界（图例口径）

| 边界 | 图中表达 | 状态 |
|---|---|---|
| Development Build / 新 BuildRequest、BuildResult、Artifact 契约 | 全部标为设计提案/设计草案 | NOT IMPLEMENTED |
| Host 原生 HTTP/2 Nsmf Listener | 图 4 节点 + 设计文档 | NOT IMPLEMENTED |
| SBI Decoder / multipart Content-ID | 图 4 节点 | NOT IMPLEMENTED |
| 5GSM 只读 Codec | 图 4 节点 | NOT IMPLEMENTED（候选） |
| Owner-private 原始 N1 输入 | 图 4 虚线 | NOT IMPLEMENTED |
| Host SM Context URI Map（M1） | 图 4 节点 | NOT IMPLEMENTED |
| Host Namf Relay 接收环 | 图 4 节点 | NOT IMPLEMENTED |
| NRF 双地址 / 单一 NF Profile 发布者 | 图 4 NRF 标签 | NOT ACCEPTED（未验收） |
| SCP 按调用方 NF 类型自动分流 | 图 4 SCP 标签 | 明确否定，未验证 |
| Owner Pending 持久化 / Owner Recovery | 图 3 标签、图 4 标签 | 阻塞项 |

## 7. 05 系列：命名分层、Capability 追踪与业务盘点

本目录现在交付的是**四张系统图 + 七张业务时序图 + 一张 Capability 追踪图 + 本附页**。
七张时序图对应绘图简报 §3 “05”与 `05x` 复审要求。

### 7.1 三层命名不可合并

图形中同一个动作会同时出现三种身份，**不得合并成一个 ID**：

| 层 | 名称来源 | 示例 | 图形位置 |
|---|---|---|---|
| 3GPP SBI | TS 29.502 / 29.518 §5.2 的操作表 | `Nsmf_PDUSession_CreateSMContext` | 箭头主标签 |
| OpenAPI | 实现里的 `operationId` + method + URI | `PostSmContexts` / `POST /nsmf-pdusession/v1/sm-contexts` | 卡片；不替代 Service Operation |
| OpenAMSM 私有 | OpenAMSM 自有 OpenAPI | `Nsmf_MasOrchestration` / `masOrchestrationInvoke` / `smContextUpdateSync` | 箭头或卡片，**始终标为 Private API** |
| CA-SL | `contract.yaml` 的 Capability ID + `mas::` 候选 Callable | `mas.sc.context.create` / `mas::SC.CREATE_SM_CONTEXT` | 05a-ops 左右两列 |
| MAS 内部操作 | OpenAMSM OES 源码原名 | `SC.CREATE_SM_CONTEXT` / `UPN.MODIFY_UP_SESSION` / `SSM.SIGNAL_UE_AN` | OES step 列 |

关键区分：**外部 `CreateSMContext` 对应整个 Establishment Skill，不等于内部的 `SC.CREATE_SM_CONTEXT`**。
前者是一个 Nsmf Service Operation，后者是 13 步编排中的第 3 步。

### 7.2 Establishment 13 × 13 Capability 追踪

来源：`ca-skills` @05be9ee8 `assurance/procedure.yaml`（13 个 operation 节点）与
`assurance/contract.yaml`（13 个 capability），对照 `ca-client-tools` @e2922ca4
`contracts/mas-owner-candidate/v0.2.0/source-map.json`（13 个 stepId / capabilityCandidate / publicCallable）。
**四个集合两两相等，差异 0。** 逐项见 `05a-ops-establishment-capabilities.html`。

| # | OES Step | Skill Node ID | Capability ID | Callable Ref（= OpenAMSM MAS Operation 加 `mas::`） |
|---|---|---|---|---|
| 1 | s1 | `validate-sm-subscription` | `mas.sc.subscription.validate` | `mas::SC.VALIDATE_SUBSCRIPTION` |
| 2 | s2 | `skip-secondary-authentication` | `mas.saa.secondary.auth` | `mas::SAA.SECONDARY_AUTH` |
| 3 | s3 | `create-sm-context` | `mas.sc.context.create` | `mas::SC.CREATE_SM_CONTEXT` |
| 4 | s3r | `register-uecm` | `mas.sc.uecm.register` | `mas::SC.REGISTER_UECM` |
| 5 | s3p | `establish-sm-policy-association` | `mas.pa.sm.policy.create` | `mas::PA.CREATE_SM_POLICY` |
| 6 | s4 | `select-upf-anchor` | `mas.ups.anchor.decide` | `mas::UPS.DECIDE_ANCHOR` |
| 7 | s5 | `allocate-pdu-session-address` | `mas.iam.address.allocate` | `mas::IAM.ALLOCATE_ADDRESS` |
| 8 | s6u | `modify-sm-policy-association` | `mas.pa.sm.policy.update` | `mas::PA.UPDATE_SM_POLICY` |
| 9 | s7 | `execute-charging-step` | `mas.ca.charging.create` | `mas::CA.CREATE_CHARGING` |
| 10 | s8 | `establish-n4-session` | `mas.upn.up.session.create` | `mas::UPN.CREATE_UP_SESSION` |
| 11 | s10 | `signal-ue-and-access-network` | `mas.ssm.ue.an.signal` | `mas::SSM.SIGNAL_UE_AN` |
| 12 | s11 | `modify-n4-session` | `mas.upn.up.session.modify` | `mas::UPN.MODIFY_UP_SESSION` |
| 13 | s13 | `record-session-establishment-event` | `mas.se.events.emit` | `mas::SE.EMIT_EVENTS` |

终态节点 `verify-pdu-session-establishment` 只聚合 13 个 postcondition predicate，
**不是 MAS Tool**，单独绘制，不计入 13。

全部 13 项都是**候选引用**：MAS ToolCatalog、输入输出 Schema、CapabilityContracts、
Owner Receipt、Evidence Guarantee 均未发布，`source-map.json` 的 gates 全为 `BLOCKED`，
`status = NOT_QUALIFIED`、`deployable = false`。

### 7.3 业务 Feature 盘点（F01–F13）

| 流程 | 业务 | 本图集 | 状态 |
|---|---|---|---|
| F01 | 初始建立 | 05a | 有冻结 v1alpha5 Procedure；链路未实现 |
| F02 | 建立中的 N2 结果回传 | 05a（同一图的后半段） | 设计已定：**恢复原 s10 pending occurrence**，不新建 Modify Skill |
| F03 | UE 请求修改 | 05b | 无冻结 v1alpha5 Procedure；`agent_step` 拒绝普通业务 Update |
| F04 | RAN/AMF 请求修改 | 05b | 同上；部分 N2 结果 / upCnxState 分支例外 |
| F05 | 网络/PCF 触发修改 | 05c | 受信 Owner 事件桥接未实现 |
| F06 | UE 请求释放 | 05d | 无冻结 v1alpha5 Procedure；Release 被拒 |
| F07 | AMF 请求释放 | 05d | 同上 |
| F08 | 网络侧触发释放 | 05e | 触发路径与回调闭环未实现 |
| F09 | Service Request / 用户面激活 | 05f | 操作既有 Session；`smContextUpdateSync` 已定义 |
| F10 | Handover / Path Switch | 05f | 子场景按 Feature Profile 逐项验收，未全量支持 |
| F11 | 失败通知 / 状态回调 / UNKNOWN | 05g | 回调 Router 与 occurrence 映射未实现 |
| F12 | Retrieve SM Context / Send MO Data | **NOT_IMPLEMENTED** | 未绘图。OpenAPI 有 `RetrieveSmContext` / `SendMoData`，但无 Host 实现与 Skill 定义 |
| F13 | AMF 状态/事件订阅、UE Reachability | **NOT_IMPLEMENTED** | 未绘图。属可选 Feature Profile，`Namf_EventExposure` / `Namf_MT` 未接入 |

F12 / F13 在业务盘点中保留，但没有实现证据，因此**不虚构流程**、不单独成图。

### 7.4 标准 Operation / operationId / OpenAMSM MAS Operation / Capability 对照

| 标准 Operation（3GPP） | OpenAPI operationId | HTTP + URI | OpenAMSM MAS Operation | 关联 CA-SL Capability | 来源 |
|---|---|---|---|---|---|
| `Nsmf_PDUSession_CreateSMContext` | `PostSmContexts` | POST `/nsmf-pdusession/v1/sm-contexts` | `[s3] SC.CREATE_SM_CONTEXT`（整个 Skill 对应本操作，不是单步绑定） | 13 项建立 Capability 全体 | env-openamsm `TS29502_Nsmf_PDUSession.yaml` V19.5.0 |
| `Nsmf_PDUSession_UpdateSMContext` | `UpdateSmContext` | POST `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/modify` | `[s3] SC.UPDATE_SM_CONTEXT` / `[m0] SC.UPDATE_SM_CONTEXT`（按业务分类） | Establishment：`mas.sc.context.create` 结果续用；Modify：TBD | 同上 |
| `Nsmf_PDUSession_ReleaseSMContext` | `ReleaseSmContext` | POST `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/release` | `[r5] SC.RELEASE_SM_CONTEXT` | Modify/Release：TBD | 同上 |
| `Nsmf_PDUSession_RetrieveSMContext` | `RetrieveSmContext` | POST `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/retrieve` | 未定义 | NOT_IMPLEMENTED（F12） | 同上 |
| `Nsmf_PDUSession_SendMOData` | `SendMoData` | POST `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/send-mo-data` | 未定义 | NOT_IMPLEMENTED（F12） | 同上 |
| `Namf_Communication_N1N2MessageTransfer` | `N1N2MessageTransfer` | POST `/namf-comm/v1/ue-contexts/{ueContextId}/n1-n2-messages` | `[s10] SSM.SIGNAL_UE_AN` / `[m4] SSM.SIGNAL_UE_AN` / `[r3] SSM.SIGNAL_UE_AN` | `mas.ssm.ue.an.signal`（候选复用） | env-openamsm `TS29518_Namf_Communication.yaml` V19.6.0 |
| `Namf_Communication_N1N2TransferFailureNotification` | `N1N2TransferFailureNotification` | AMF 回送到 SMF 提供的 `n1n2FailureTxfNotifURI`（**不是**固定 `/n1-n2-messages` 集合资源） | 原 occurrence 的 Owner callback | —（不是 Skill Capability） | 同上 + flowbook §4.7 |
| NAS：`PDU Session Establishment Request` | — | — | `[s3]` 前段输入 | — | TS 24.501 消息类型 `0xC1` |
| NAS：`PDU Session Modification Request` | — | — | `[m0]` 前段输入 | — | `0xC9` |
| NAS：`PDU Session Release Request` | — | — | `[r5]` 前段输入 | — | `0xD1` |
| NAS：`PDU Session Release Command` / `Complete` | — | — | `[r3]` 信令 | — | `0xD3` / `0xD4` |
| N2 SM：`n2SmInfoType=PDU_RES_SETUP_RSP` | — | — | `[s10]` 结果关联 | — | N2 SM 信息类型，**不是** NGAP Message / 不是 NGAP 数字消息 ID |
| NGAP：`PDUSessionResourceSetupResponseTransfer` | — | — | — | — | NGAP **Transfer 载荷结构**，不是一条完整 NGAP Message |
| PFCP：`Session Establishment Request` | — | — | `[s8] UPN.CREATE_UP_SESSION` | `mas.upn.up.session.create`（候选） | Type 50 |
| PFCP：`Session Modification Request` | — | — | `[s11]` / `[m3]` / `[m5] UPN.MODIFY_UP_SESSION` | `mas.upn.up.session.modify`（候选） | Type 52 |
| PFCP：`Session Deletion Request` | — | — | `[r2] UPN.DELETE_UP_SESSION` | TBD | Type 54 |
| OpenAMSM 私有：`Nsmf_MasOrchestration` | `masOrchestrationInvoke` | POST `/nsmf-mas-orchestration/v1/invoke` | `stepId, mas, op` 组合，如 `[s3] SC.CREATE_SM_CONTEXT` | 见 7.2 | env-openamsm `Nsmf_MasOrchestration.yaml` v1.0.0 |
| OpenAMSM 私有：`Nsmf_MasOrchestration` | `smContextUpdateSync` | POST `/nsmf-mas-orchestration/v1/sm-context-update-sync` | ACTIVATING 同步包 | — | 同上 |

`Nsmf_MasOrchestration` 是 **OpenAMSM Private API**，不是 3GPP `Nsmf_PDUSession` Service。

### 7.5 Modify / Release 的 Capability 对照（候选复用与 TBD）

**Modify（`pdu-mod-v3.yaml`）**

| Step | MAS Operation | Capability | 状态 |
|---|---|---|---|
| m0 | `SC.UPDATE_SM_CONTEXT` | TBD（无冻结 ID） | 候选复用需 Schema / Effect / Owner 校验 |
| m2（可选） | `CA.UPDATE_CHARGING` | TBD | 可能为 marker/stub，不得推断真实 CHF 计费 |
| m3 | `UPN.MODIFY_UP_SESSION` | `mas.upn.up.session.modify` | **候选复用**（待校验）；与 m5 是两个独立 Invocation |
| m4 | `SSM.SIGNAL_UE_AN` | `mas.ssm.ue.an.signal` | **候选复用**（待校验）；与建立 s10 的 outcome / correlation 不可混淆 |
| m5 | `UPN.MODIFY_UP_SESSION` | `mas.upn.up.session.modify` | **候选复用**；第二次独立 occurrence，不复用 m3 的 receipt（05c 中显式画为两条消息） |

原 plan 不自动执行 `PA.UPDATE_SM_POLICY`，与建立阶段的 s6u 不同，不可照抄。

**Release（`pdu-rel-v4.yaml`）**

| Step | MAS Operation | Capability | 状态 |
|---|---|---|---|
| r1（可选） | `CA.DELETE_CHARGING` | TBD | 需证明实现，不得推断 CHF 已释放 |
| r2 | `UPN.DELETE_UP_SESSION` | TBD | 需独立 Owner 结果与资源结算 |
| r3 | `SSM.SIGNAL_UE_AN`（release=1） | `mas.ssm.ue.an.signal` | **候选复用**；需 Callback / Pending / timeout 合同 |
| r4 | `PA.DELETE_SM_POLICY` | TBD | 精确资源关联 |
| r5 | `SC.RELEASE_SM_CONTEXT` | TBD | Host 外部 URI Map 的结束与 tombstone |
| r6 | `SE.EMIT_EVENTS` | `mas.se.events.emit` | 候选；**不是** 3GPP 状态 Notify 成功证据 |
| r7（skip） | `SC.COMPENSATE_RELEASE` | 无（不是通用业务 Tool） | OES 补偿节点；不照搬 `compensate_then_replan_once`，只用授权的 exact-original Reconcile |

外部 SBI、NAS Codec、N4/PFCP **不会**因为出现在图中就被列入 Skill Capability；
Owner exact-original Reconcile 也不是通用业务 Tool。

## 8. 规范版本差异（摘要）

实现侧 OpenAPI 声明 **TS 29.502 V19.5.0 / TS 29.518 V19.6.0**；
业务设计文档引用 **V19.7.0 / V19.8.0**，Stage 2 目标为 **TS 23.502 V19.9.0**。
本图集只用已核对的实现版本标注 operationId 与 URI；目标版本的字段级差异**未核对（BLOCKED）**。
同一版本内 `standard` 与 `modified` 的 path+operationId 集合完全一致（29.502：16 项；29.518：23 项）。
完整审计见 [evidence/version-audit.md](evidence/version-audit.md)。

## 9. 仍未闭合的部分（本轮）

| 项 | 状态 | 说明 |
|---|---|---|
| `05a`–`05g` 七张业务时序图 | **DELIVERED** | 全部通过 Archify 四道门，附 1440/2048 真实截图 |
| Establishment 13 项 Capability 追踪 | **DELIVERED** | 13 对 13，差异 0 |
| 四责任域矩形边框（图 04） | **PARTIAL** | Archify `region` 边界要求互不重叠紧凑矩形；Client Tools 节点位于 CA-Agent 矩形内的消息路径上。当前以 CA-Agent 区域边框 + 责任域卡片 + 节点标签表达，详见 README §5 |
| UML `opt` / `alt` 组合片段 | **PARTIAL** | Archify 3.2.0 sequence 渲染器不支持该边框；改用命名 phase 段 + 标签前缀 + 卡片条件说明 |
| `3gpp-openamsm-mas-naming-crosswalk-scp-only.zh-CN.md` | **BLOCKED（来源缺失）** | 该文件在指定分支与 main 上均不存在；命名改用 OpenAPI + inventory 重建，见 evidence/source-inventory.md §3 |
| `mas-capability-traceability-scp-only.{zh-CN.md,json}` | **BLOCKED（来源缺失）** | 同上；追踪改用 contract.yaml + source-map.json 机械对照 |
| F12 / F13 流程 | **NOT_IMPLEMENTED** | 无实现证据，未绘图 |
| V19.7.0 / V19.8.0 差异核对 | **BLOCKED** | 本机无对应 OpenAPI 副本 |

已按 05x 纠错修订的项：

- 纠错 #1：Host SBI Decoder（只做 HTTP/2 / JSON / MIME / Content-ID）与 CA Client Tools 内部只读 5GSM Codec（类型化投影）分属不同节点、不同责任域；Codec 与 Tool Dispatch 同属 CA Client Tools 责任域，05a / 05b / 05d 以同一 Client Tools lifeline 表达，并在副标签与卡片中区分内部职责。
- 纠错 #2：`201 + Host Location` 由 Host 产生并沿 SCP 返回；OpenAMSM 的 `smContextRef` 不是 AMF 可见 Location。
- 纠错 #3 / #4：UDM/PCF/CHF 经 SCP，UPF 走 N4/PFCP 且不经 SCP，二者从不合并为一条线。
- 纠错 #5：AMF→SCP→Host 为公开 Nsmf；OpenAMSM→Host 私有 Namf Relay→Client Tools→SCP→AMF 为 Namf；Host→OpenAMSM 为受信私有绑定。
- 纠错 #6：F02 的 N2 回传继续原 s10 pending occurrence，不创建新 Modify Skill（05a 后半段）。
- 纠错 #7：Modify/Release 的 OES Plan 存在但无 CA-SL Procedure，图中虚线 + `DESIGN` + 阻塞标注。
- 纠错 #8：F12 / F13 在 §7.3 盘点中保留并标 `NOT_IMPLEMENTED`。

## 10. 复现方式

```bash
# 需要本机已安装的 Archify（3.2.0）。以下路径相对于仓库根目录。
ARCHIFY=<archify skill>/bin/archify.mjs

node "$ARCHIFY" finalize architecture docs/architecture/archify/01-toolchain-panorama.architecture.json docs/architecture/archify/01-toolchain-panorama.html --quality showcase --json
node "$ARCHIFY" finalize workflow     docs/architecture/archify/02-offline-build.workflow.json         docs/architecture/archify/02-offline-build.html      --quality showcase --json
node "$ARCHIFY" finalize architecture docs/architecture/archify/03-host-admission.architecture.json   docs/architecture/archify/03-host-admission.html     --quality showcase --json
node "$ARCHIFY" finalize architecture docs/architecture/archify/04-scp-amf-smf.architecture.json      docs/architecture/archify/04-scp-amf-smf.html        --quality showcase --json
node "$ARCHIFY" finalize sequence     docs/architecture/archify/05a-establishment-sequence.sequence.json docs/architecture/archify/05a-establishment-sequence.html --quality showcase --json
node "$ARCHIFY" finalize architecture docs/architecture/archify/05a-ops-establishment-capabilities.architecture.json docs/architecture/archify/05a-ops-establishment-capabilities.html --quality showcase --json
# 05b–05g 同构，按 <name>.sequence.json → <name>.html 替换
```

可编辑源文件与 HTML 并列存放于 `docs/architecture/archify/`，便于评审直接修改与重渲染。
2026-11-20 架构冻结轮重绘了图 01–03（两类 Requirement 模型）；这三张图的 `validate` / `deliver` / 严格 `check`
已通过，`browser-check` 与 `visual-check` 因运行环境禁止管道子进程（`spawn EPERM`）为 NOT_RUN/BLOCKED，
详见 [evidence/acceptance.md](evidence/acceptance.md)。
验收记录见 [evidence/acceptance.md](evidence/acceptance.md)。
