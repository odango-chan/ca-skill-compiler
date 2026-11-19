# 附页：最小示例、字段来源与跨图追踪

> 配套图：`01-toolchain-panorama`、`02-offline-build`、`03-host-admission`、`04-scp-amf-smf`。
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

### 2.2 `dependencies.lock` 与义务记录（设计草案）

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
// 设计草案：单条义务记录节选
{
  "kind": "EVIDENCE_GUARANTEE",
  "scope": { "node": "validate-sm-subscription", "stage": "postcondition" },
  "sourceCallable": "mas::SC.VALIDATE_SUBSCRIPTION",
  "interfaceSnapshotDigest": "sha256:…mas",
  "requiredGuarantee": "session.exists",
  "status": "UNRESOLVED"
}
```

`status` 只允许 `UNRESOLVED`（材料缺失，须由 Host 逐项核验）或 `DESCRIPTOR_PRESENT`（材料静态匹配，仍须 Host 核验）。
两种情况都不是执行授权。

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
  "obligations": ["obligation://…"],
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
| UNRESOLVED | Assurance 可缺失 | 义务 UNRESOLVED | Assurance 可缺失 | — |
| UNKNOWN | — | — | 结果 UNKNOWN → Reconcile | Owner pending / 无 proof |
| SCP | 不出现（离线区） | 不出现 | Invocation Context 来源 | 唯一网络入口 |
| 201 Location | — | — | — | Host SM Context URI Map |

## 5. 与两条 PR 基线的逐项一致性

对照 `ca-skill-compiler` PR #48（工具链架构 + 绘图简报）与 `ca-agent-demo` PR #217（Host SBI / SCP 路由）。

| # | 约束 | 落点 | 结果 |
|---|---|---|---|
| 1 | Interface Package 与 Runtime Provider 独立发布 | 图 1：两条发布链；`dependencies.lock` 与 Provider 绑定分开 | 一致 |
| 2 | Capability Catalog 与 Deployment Grant 分离 | 图 1：Catalog 随包发布（`导出`），Grant 由部署方签发（`只交 Host`） | 一致 |
| 3 | 缺 Evidence 可生成 Artifact，但保留 UNRESOLVED / NOT_ADMITTED；缺 Interface 则 Link FAIL 无产物 | 图 2 两条失败/成功分支 | 一致 |
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

## 7. 复现方式

```bash
# 需要本机已安装的 Archify（v3.2.0）。以下路径相对于仓库根目录。
ARCHIFY=<archify skill>/bin/archify.mjs

node "$ARCHIFY" finalize architecture .archify/workflow-ca-sl-toolchain-scp-20261119/01-toolchain-panorama.architecture.json docs/architecture/archify/01-toolchain-panorama.html --quality showcase --json
node "$ARCHIFY" finalize workflow     .archify/workflow-ca-sl-toolchain-scp-20261119/02-offline-build.workflow.json       docs/architecture/archify/02-offline-build.html      --quality showcase --json
node "$ARCHIFY" finalize architecture .archify/workflow-ca-sl-toolchain-scp-20261119/03-host-admission.architecture.json  docs/architecture/archify/03-host-admission.html     --quality showcase --json
node "$ARCHIFY" finalize architecture .archify/workflow-ca-sl-toolchain-scp-20261119/04-scp-amf-smf.architecture.json    docs/architecture/archify/04-scp-amf-smf.html        --quality showcase --json
```

可编辑源文件与 HTML 并列存放于 `docs/architecture/archify/`，便于评审直接修改与重渲染。
