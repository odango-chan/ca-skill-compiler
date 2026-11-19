# Archify 绘图简报：CA-SL 工具链与 SCP-only 运行架构

> 状态：DESIGN REVIEW BRIEF。绘图不是 E2E 通过或实现完成的证明。请用本机已安装的 Archify Skill 生成图，不要升级 Skill 或另造绘图器。案例统一为 `session-demo`。

## 1. 目标与原始资料

原来的三张工具链图保留主题，第四张画 SCP 必选的 AMF–SMF SBI 边界；**再增加一组第 5 号业务流程时序图**，用 3GPP 风格逐业务画出 Establishment、Modification、Release、Service Request、Handover 和异常回调。不要把所有业务画进一张巨型图。每张图有可编辑的 Archify JSON、离线 HTML 和可读的静态图（如果 Skill 支持导出 SVG/PNG）。文件源应提交到仓库的可评审目录，不使用本机绝对 Windows 路径作为链接。

**规范与设计基线：**

- 编译器架构：`ca-skill-compiler/docs/design/ca-toolchain-architecture-vnext.zh-CN.md`
- Compiler 演进路线：`ca-skill-compiler/docs/design/compiler-evolution-roadmap-vnext.zh-CN.md`
- Compiler Development Build Pilot：`ca-skill-compiler/docs/design/compiler-development-build-pilot.zh-CN.md`
- SCP-only 业务流程全集和七张 Mermaid 时序图：ca-agent-demo/docs/design/amf-smf-business-procedure-flowbook-scp-only.zh-CN.md（PR #217）；此文件是第 5 号流程图的事实基线
- Host SCP 路由：`ca-agent-demo/docs/design/scp-dual-smf-endpoint-routing.zh-CN.md`（见 Draft PR #217）
- Host 原生 SBI / NAS → Skill：`ca-agent-demo/docs/design/host-sbi-http2-decoded-skill-architecture.zh-CN.md`
- Host/Client Tools 收发：`ca-agent-demo/docs/design/host-client-tools-amf-smf-boundary.zh-CN.md`
- AMF–SMF 操作全集：`ca-agent-demo/docs/design/amf-smf-sbi-interface-inventory.zh-CN.md`
- MAS Runtime/E2E 手册：`ca-agent-demo/docs/design/mas-e2e-harness.zh-CN.md`
- Frozen Skill 示例：`ca-skills/examples/skills/establish-pdu-session-mas/`

旧稿（先审查再决定能否复用）：
`.archify/workflow-ca-sl-toolchain-20261119-1350/01-toolchain-panorama.architecture.json`、
`02-offline-build.workflow.json`、
`03-host-admission.architecture.json`。
若旧源不存在，可用已安装 Archify Skill 重新生成相同主题。绝不伪称找到了旧文件。

## 2. 这次已经确定的架构约束

1. Capability Catalog 和 Deployment Grant 分离；前者属于 Interface Package 的导出契约，后者由部署方交给 Host。
2. Interface Package 与 Runtime Provider 独立发布；Compiler 只使用精确 Interface snapshots，不承担发布/注册/安装。
3. 缺少 Evidence / Assurance 时，**下一代设计**允许生成 Linked Development Artifact，`UNRESOLVED / NOT_ADMITTED`；但缺少 Interface/Callable/Schema 时 Link 失败、**没有 Artifact**。该特性尚未实装于 Compiler v2.0.2。
4. Host 使用冻结的 CA-SL v1alpha5 与单一通用 Runtime；MAS 是示例，通用 Runtime 不能硬编码 MAS 或 OpenAMSM。
5. **所有后续 H1/H2/H3 网络集成和 E2E 测试必须使用 SCP。** H0 离线单测不代表网络拓扑。不要再绘无 SCP、AMF 直连 Host、Direct Mock Relay 作为正式测试分支。
6. AMF → SCP → Host 的 Nsmf 正式 SBI；OpenAMSM → Host **内部私有** Namf Relay → Client Tools → SCP → AMF；AMF callbacks → SCP → Host。OpenAMSM SMF → UDM/PCF/CHF/NRF 的 SBI 仍经既有 SCP/NRF；N4/PFCP 独立于 SCP。
7. 对 AMF 可见的 Nsmf apiRoot 与 201 Location 必须属于 Host；OpenAMSM SMF 是内部 Owner endpoint。Host → OpenAMSM 使用固定、受信、**内部** Client Tools Binding，不重新经 SCP 路回 Host。
8. HTTP/2 SBI Listener、SBI JSON+multipart Decoder、只读 5GSM Codec、类型化 Host Context、Skill/Runtime、Client Tools Egress、OpenAMSM GSM Handler 是不同节点。
9. 原始 N1/N2 二进制保留在 Host Artifact Store；只读 Codec 的解码投影给 Skill，**原始 N1**另经 Owner-private 受信绑定到 OpenAMSM。此绑定当前尚未实现，图中用虚线标注。Skill 不接收裸 NAS/NGAP bytes。
10. M0（201 Location 指向 OpenAMSM）仅是历史过渡对照，不作为本轮 E2E 实施/验收 Profile。M1 才是目标。
11. OpenAMSM 的 Owner SSM /invoke waiter 仍存在进程内状态和 30 秒 Provider 原型限制；Temporal 不能虚构 Owner continuation 或把未知副作用重试为成功。
12. NRF 注册时必须确认**单一受控 NF Profile 发布者**，禁止 Host 与 OpenAMSM 的冲突 Nsmf Profile 抢注册；stock Open5GS SCP 的按调用方 NF 类型自动分流尚未验证，不能画成既有功能。

## 3. 交付四张系统图和一组 3GPP 风格业务流程时序图

### 01 — 工具链全景（Panorama）

三个主要责任区：发布/依赖准备、离线 Compile/Link/Build、Host Admission/Runtime。节点用真实名称，不画“·”分隔的长条。明确 Interface Package 与 Provider Package 独立发布，Catalog 与 Grant 的归属不同。

用 `session-demo` 贯穿图，摘要占位使用 `sha256:…mas`、`sha256:…ctx`、`sha256:…artifact`、`sha256:…provider`。这些只作追踪示例，须标明非真实校验和。

### 02 — 离线构建（Compile → Link → Build）

依次显示 YAML/Procedure 源、Interface Package、dependencies.lock、Compile、Interface Link、Assurance evaluation、Artifact assembly、manifest / obligations.json。

**必须有两个分叉：**
- 缺 Interface：Link FAIL → no artifact。
- Interface 闭合、Evidence 缺失：Build PASS（设计态）→ Linked Artifact + `UNRESOLVED` obligations → `NOT_ADMITTED`。

使用小型源码窗，给外行展示真正的 Procedure 片段、字段对应关系，譬如：

```yaml
start: validate-sm-subscription
nodes:
  validate-sm-subscription:
    operation:
      tool: mas::SC.VALIDATE_SUBSCRIPTION
      arguments:
        psi: context.request.pduSessionId
        dnn: context.request.dnn
```

该代码是**节选**，不能标为独立可编译的完整 Procedure；正式字段、路径与 outcome 须对照 ca-skills 原件。其他 manifest/obligations/lock 例子若是未来草案，显著标“示意 schema”，不可伪装为 Compiler v2.0.2 现行输出。

### 03 — Host Admission 和 Runtime

展示 Artifact + Provider Binding + Assurance + Deployment Grant + Invocation Context 五种入站材料，Host Admission 及允许/拒绝分支，进入通用 Runtime 后的固定 Typed Provider、Owner Operation、UNKNOWN/Reconcile。说明开发 Artifact 只可在显式隔离测试策略下准入；生产 Host 严格拒绝无资格产物。调度和 Owner Pending 分开画，不把 Temporal 当成 SMF Owner waiter。

不把 SCP 放到离线编译区；SCP 只出现在真实 NF 通信一侧。

### 04 — SCP-only AMF–SMF SBI 系统架构（新增）

图必须分开画**实际网络下一跳**与**目标 NF Service apiRoot**：

- NRF：NF Profile / Nsmf Service 的正式 Host apiRoot，**发布权归一个受控 Owner**；
- AMF → SCP → Host HTTP/2 SBI Listener；
- SBI Decoder → 原始 N1 存储 → 只读 Codec → Typed Host Context → CA-SL Skill → Runtime；
- Host Context Router 处理 Update / Release / Callback；**不是每条消息都新建 Skill**；
- Runtime / Router → Client Tools → 受信私有 OpenAMSM SMF；
- OpenAMSM → Host 私有 Namf Relay → Client Tools → SCP → AMF；
- AMF callback → SCP → Host callback；
- OpenAMSM → UDM/PCF/CHF/NRF 的常规 SBI（按 local 配置经 SCP），OpenAMSM → UPF 的 N4/PFCP；
- 201 Host Location + 内外 URI Map、后续 Update 到相同 Host apiRoot 的路由闭环；
- 原始 N1 binary → Owner-private ingress 未实现，用虚线。

**外行也能理解的紧凑示例（不要构造过度复杂的 3GPP 报文）：**

```text
AMF 发送:
POST /nsmf-pdusession/v1/sm-contexts  HTTP/2
JSON: {pduSessionId: 10, dnn: internet, n1SmMsg: {contentId: n1-part}}
附件 n1-part: 原始 N1 NAS bytes
```

```yaml
# 解码后映射给 Skill 的示意数据，非新语言语法
request:
  pduSessionId: 10
  dnn: internet
  sNssai: {sst: 1}
  requestedPduSessionType: IPV4
```

并用 4–6 行真实 MAS Procedure 节选说明 Skill 读取 `context.request.*`、调用 Capability 的方式。区分 SBI JSON 字段与 NAS 字段来源，示意数据不能当作真实运行时日志。

### 05 — AMF–SMF 完整业务生命周期（3GPP 风格，多张时序）

权威资料：[ca-agent-demo 的 SCP-only 业务流程全集](https://github.com/odango-chan/ca-agent-demo/blob/docs/mas-v1alpha5-temporal-e2e-review-20261118/docs/design/amf-smf-business-procedure-flowbook-scp-only.zh-CN.md)。

**至少分别重绘以下七个子流程**，沿用图 04 的真实网络拓扑与名词：

| 子图 | 业务 | 必须体现的路径 |
|---|---|---|
| 05a | 初始 PDU Session Establishment + N2 回传 | UE → AMF → SCP → Host；13 个 MAS 初始建立动作归到对应阶段；s10 Namf 经 Host/Client Tools/SCP 到 AMF；N2 Setup Response → SCP → Host → 原 Owner Pending；s11 N4 |
| 05b | UE/RAN 发起 Modification | UpdateSMContext JSON/N1/N2 → Host 解码/分类；同一 Session，新 Modify Run 或继续原 Owner；m0/m2/m3/m4/m5（适用时）；OpenAMSM agent_step 的 business Update 拒绝是阻塞 |
| 05c | 网络/PCF 触发 Modification | PCF → OpenAMSM 内部事件 → Host 受信事件 → Modify Run → Namf → SCP → AMF → Update 返回；不重建 Session |
| 05d | UE/AMF 发起 Release | N1 Release Request 或 ReleaseSMContext → SCP → Host → r1–r6 → OpenAMSM N4/PCF 清理；Host URI tombstone；当前 agent_step Release 拒绝是阻塞 |
| 05e | 网络触发 Release | SMF/PCF 内部事件 → Host Release Run → Namf → AMF → SBI 回传 → 资源结算；不得把取消当成功 |
| 05f | Service Request / Handover | upCnxState ACTIVATING / DEACTIVATED、hoState、N2 resource response / AN F-TEID；只更新现有 Session，不调用 Create Skill |
| 05g | Callback + Failure / UNKNOWN | Namf N1N2TransferFailureNotification → SCP → Host exact occurrence；晚到/重复/超时/Owner 重启；只有 authorized Reconcile，没有盲重派 |

**重要的真实能力区分：**

- OpenAMSM 中存在 establish / modify / release 三个 OES YAML 计划，且 MAS facade 有分派分支；但是**当前 CA-SL v1alpha5 的 MAS 示例只正式定义 Establishment Procedure**。不能把 Modify/Release 源码计划说成已存在的可链接 CA-SL Skill。
- `pdu-mod-v3.yaml` 主要包含 m0 SC.UPDATE_SM_CONTEXT、m2 CA.UPDATE_CHARGING（可选）、m3/m5 UPN.MODIFY_UP_SESSION、m4 SSM.SIGNAL_UE_AN。
- `pdu-rel-v4.yaml` 包含 r1 CA.DELETE_CHARGING（可选）、r2 UPN.DELETE_UP_SESSION、r3 SSM.SIGNAL_UE_AN(release)、r4 PA.DELETE_SM_POLICY、r5 SC.RELEASE_SM_CONTEXT、r6 SE.EMIT_EVENTS；r7 COMPENSATE_RELEASE 在原计划中标为 skip，不能照搬自动重规划。
- 源码 `env-openamsm/upstream/openamsm/src/smf/nsmf-handler.c` 在 agent_step 下拒绝通用业务 Update 和 Release，但对某些 N2 Setup Result、upCnxState-only 等已有特殊分支。必须画出“目标调用”与“当前阻塞”，不要画成已实现。
- 5GSM Codec 只读解码并将 Typed Context 给 Skill；原始 N1/N2 Owner-private ingestion 需要单独虚线。
- 已有的 s10 N2 回传必须关联**原始 pending Owner occurrence**，不能绘成另一个新 Skill。
- PCF/UDM/UPF 的内部行为由 OpenAMSM 执行；对外公开 AMF–SMF SBI 均经过 SCP，内部 Owner-private Binding 是明确例外；PFCP/N4 不走 SCP。
- 每张序列图按 3GPP 风格按时间由上到下，用请求/响应和可选步骤（opt/alt）表示变化，避免过度使用卡片、口号、装饰性图例或同一图内过多泳道。

请从同一 `session-demo` 的外部 Create 开始，向现有 Session 的 Update、Release、Service Request 延展，并在附页说明一个 Nsmf UpdateSMContext API 为什么会触发不同业务流程。每张图下面附 2–4 行“当前源代码证据 / 未实现缺口 / 下一步归属仓”，不要编造 E2E PASS。

## 4. 表现方式

- 文档风格：类似 3GPP/OASIS/系统架构评审资料，非宣传海报；白底/浅灰底、深灰线条、最多一种低饱和度强调色。
- **一个节点代表一个责任**。不要用“发布 · 编译 · 链接 · 运行”这样的字符串假装流程节点；用准确的箭头连接独立节点。
- 网络协议写在线上（HTTP/2 SBI、SCP、N4/PFCP、Owner-private Binding），数据类型/产物写在边缘或正文。
- 少用圆角大卡片、渐变、emoji、装饰图标、冗余英文口号和“AI式”小标题；统一中文技术术语和字重。
- 没有足够空间就拆成同主题的子视图，不以缩小字体或挤入大段文字解决。
- 所有图标出哪些已实现、哪些是目标设计、哪些被阻塞；图例简洁，不用彩色徽章制造完成假象。
- 1440 宽度下字可读，2048 输出不超界；真实浏览器检查 HTML 并保存截图。可编辑 JSON 必须经过 Archify 校验。

## 5. 文件与验收

目标目录：`.archify/workflow-ca-sl-toolchain-scp-20261119/`，至少生成四张系统图、七张业务流程图（可以通过统一索引 HTML 访问）和附页：

- `01-toolchain-panorama.html` + `.architecture.json`
- `02-offline-build.html` + `.workflow.json`
- `03-host-admission.html` + `.architecture.json`
- `04-scp-amf-smf.html` + `.architecture.json`
- `05a-establishment-sequence.html` + 可编辑 Archify Workflow JSON
- `05b-ue-modification-sequence.html` + 可编辑 Archify Workflow JSON
- `05c-network-modification-sequence.html` + 可编辑 Archify Workflow JSON
- `05d-release-sequence.html` + 可编辑 Archify Workflow JSON
- `05e-network-release-sequence.html` + 可编辑 Archify Workflow JSON
- `05f-service-request-handover-sequence.html` + 可编辑 Archify Workflow JSON
- `05g-callback-failure-sequence.html` + 可编辑 Archify Workflow JSON
- `06-appendix-minimal-examples.zh-CN.md`（字段解释、流程分类表、跨图摘要追踪表）

若要作为 GitHub PR 中可访问的正式资产，应把生成文件保存到仓库跟踪目录（例如 `docs/architecture/archify/`），而非只存在被 gitignore 的本地 `.archify/`。不得上传原始 NAS、Token、真实 SUPI 或敏感配置。仅修改文档/图形；不更改运行代码、语言契约、Compiler v2.0.2 实现，不合并 PR。

验收报告必须包含：图文件列表、Archify 实际版本与执行命令、JSON 校验结果、真实浏览器渲染截图尺寸、是否包含每条已批准的关键消息路径、与原图差异、未实现边界和 PR 链接。失败要报 FAIL/BLOCKED，不得用“应该可以”替代。
