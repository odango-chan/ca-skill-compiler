# CA-SL 工具链与 SCP-only AMF–SMF 架构图（Archify）

本目录是 CA-SL 工具链与 SCP-only MAS 业务架构的正式图形资产。全部图形由本机已安装的
**Archify 3.2.0** 生成，每张图都保留可编辑的 JSON 源文件，并附真实浏览器验收证据。

打开 [index.html](index.html) 浏览全部图形。

## 1. 图形清单

| 编号 | 主题 | 类型 | 可编辑源 | 成品 | 状态 |
|---|---|---|---|---|---|
| 01 | CA-SL Toolchain Panorama | architecture | `01-toolchain-panorama.architecture.json` | `01-toolchain-panorama.html` | DESIGN |
| 02 | Offline Compile / Link / Build | workflow | `02-offline-build.workflow.json` | `02-offline-build.html` | DESIGN |
| 03 | Host Admission / Runtime | architecture | `03-host-admission.architecture.json` | `03-host-admission.html` | DESIGN |
| 04 | SCP-only AMF–SMF SBI Architecture | architecture | `04-scp-amf-smf.architecture.json` | `04-scp-amf-smf.html` | DESIGN |
| 05a | PDU Session Establishment + N2 结果回传 | sequence | `05a-establishment-sequence.sequence.json` | `05a-establishment-sequence.html` | DESIGN |
| 05a-ops | Establishment 13 Operation Node Capability 追踪 | architecture | `05a-ops-establishment-capabilities.architecture.json` | `05a-ops-establishment-capabilities.html` | 静态追踪 PASS |
| 05b | UE / RAN 发起 Session Modification | sequence | `05b-ue-modification-sequence.sequence.json` | `05b-ue-modification-sequence.html` | DESIGN TARGET |
| 05c | 网络 / PCF 触发 Session Modification | sequence | `05c-network-modification-sequence.sequence.json` | `05c-network-modification-sequence.html` | DESIGN TARGET |
| 05d | UE / AMF 发起 Session Release | sequence | `05d-release-sequence.sequence.json` | `05d-release-sequence.html` | DESIGN TARGET |
| 05e | 网络侧触发 Session Release | sequence | `05e-network-release-sequence.sequence.json` | `05e-network-release-sequence.html` | DESIGN TARGET |
| 05f | Service Request / Handover / Path Switch | sequence | `05f-service-request-handover-sequence.sequence.json` | `05f-service-request-handover-sequence.html` | DESIGN |
| 05g | Callback / Failure / UNKNOWN / Reconcile | sequence | `05g-callback-failure-sequence.sequence.json` | `05g-callback-failure-sequence.html` | DESIGN |

`DESIGN TARGET` 表示按既定业务架构表达的目标流程，不代表当前实现状态或 E2E 通过。本轮时序图不再把当前代码阻塞作为流程终点；分支、异步回调和安全停止按目标设计完整表达。

## 2. Archify 版本与命令

- Skill 包：`@tt-a1i/archify-dsh@1.1.0`，内含 **archify 3.2.0**（`package.json` 的 `version`）。
- 未升级、未重装、未另建绘图器。

每张图的生成与验收命令（`$ARCHIFY` 为已安装 Skill 的 `bin/archify.mjs`）：

```bash
node "$ARCHIFY" finalize architecture <source>.architecture.json <out>.html --quality showcase --json
node "$ARCHIFY" finalize workflow     <source>.workflow.json     <out>.html --quality showcase --json
node "$ARCHIFY" finalize sequence     <source>.sequence.json     <out>.html --quality showcase --json
node "$ARCHIFY" visual-check <out>.html --require-provenance --out-dir <dir> --summary
```

`finalize` 依次执行 `validate` / `deliver` / 严格 `check` / 真实浏览器 `browser-check` 四道门。
`visual-check` 额外产出 1440×900 与 2048×1320、明/暗两套真实截图。

## 3. 验收结果

12 张图 **12/12 通过全部四道门**（`ok: true`，`diagnostics: []`）。逐图回执见
`evidence/<figure>.finalize-summary.json` 与 `evidence/<figure>.visual-check.json`。
完整验收记录见 [evidence/acceptance.md](evidence/acceptance.md)。

## 4. 七张业务时序图的设计表达口径

05a–05g 以既定 SCP-only 业务设计为准，描述目标消息流，不按当前仓库实现状态裁剪流程。所有外部 SBI（AMF / PCF / UDM / Namf）经 SCP；OpenAMSM ↔ UPF 的 N4 / PFCP 直连；Host ↔ Client Tools ↔ OpenAMSM 的 Owner-private 调用属于内部受信边界，不经过 SCP。CA-Agent Host / Host Runtime、CA Client Tools、OpenAMSM SMF、外部 5GC 的归属通过 participant sublabel 和消息方向表达。

- 05a 明确 s3 后返回 201 + Host Location，后续 OES 步骤继续同一 Run；s10 的 N2 结果返回原 occurrence。
- 05b 区分新业务 Modify 与命中既有 Pending 的 N2 结果；按 m0 → m2（optional）→ m3 → m4 → m5 表达。
- 05c 区分 PCF callback 与 Owner scoped event；不把 PCF 通知误画成 AMF Nsmf 请求。
- 05d / 05e 按 r1（optional）→ r2 → r3 → r4 → r5 → r6 表达释放资源结算。
- 05f 把 Service Request 与 Handover / Path Switch 作为两个互斥分支，始终操作既有 Session。
- 05g 分开表达有效 callback、重复/迟到 callback，以及 UNKNOWN / exact-original Reconcile。

## 5. 与绘图简报的两处有意偏差

1. **时序图类型**：绘图简报建议使用 Archify Workflow JSON，但 3GPP 风格时序要求时间自上而下、
   请求/响应方向明确。Archify 的原生 `sequence` 渲染器正是为此设计，因此 05a–05g 使用
   `sequence` 类型并保存为 `.sequence.json`（而非 `.workflow.json`）。语义内容仍按简报要求。
2. **UML 组合片段**：Archify 3.2.0 的 sequence 渲染器不支持 `opt` / `alt` 边框。分支使用
   **命名 phase 段（segment）+ 消息标签前缀（`alt` / `重复 / 迟到回调`）** 表达，并在卡片中说明
   分支条件；未以颜色单独承载分支语义。

## 6. 已知限制（构件层面，非内容层面）

- `architecture` 的 `region` 边界要求"只包住成员、无空槽"的紧凑矩形
  （`layout/boundary-encloses-non-member`）。图 04 中 Client Tools 的 Codec 与 Provider 位于
  CA-Agent 矩形内的消息路径上，四个责任域无法同时画成互不重叠的矩形边框。当前图 04 用
  CA-Agent 区域边框 + "责任域"卡片 + 节点标签三者共同表达四域归属；**四域矩形边框为 PARTIAL**。
- `tag` 文本在 Read 视图属 `data-detail="fine"`，默认不显示；因此图 05a-ops 把
  Callable Ref 放进可见的节点标签，而不是依赖 `tag`。
- sequence 图在 1440×900 下有纵向滚动（`data-reader-fit="width-first"`）。这是 Archify 对
  sequence 的既定阅读契约：宽度自适应、纵向滚动；横向不溢出。

## 7. 事实基线

全部命名与流程取自下列来源（含版本锁定），逐文件 SHA 见
[evidence/source-inventory.md](evidence/source-inventory.md)：

- `ca-agent-demo` @ `docs/mas-v1alpha5-temporal-e2e-review-20261118`（6168912e）
- `ca-skills` @ `05be9ee8`（v1alpha5 正式评审示例 `establish-pdu-session-mas`）
- `ca-client-tools` @ `e2922ca4`（`contracts/mas-owner-candidate/v0.2.0`，NOT_QUALIFIED）
- `env-openamsm` @ `594dec78`（OpenAPI、OES plans、`nsmf-handler.c`）
- 本仓库 `ca-skill-compiler` @ PR #48 分支 `docs/toolchain-architecture-vnext-20261119`

## 8. 规范版本差异

实现侧 OpenAPI 为 **TS 29.502 V19.5.0 / TS 29.518 V19.6.0**，业务设计文档引用
**V19.7.0 / V19.8.0**。本图集只用已核对的实现版本标注 operationId 与 URI，设计目标版本
标为未核对。详见 [evidence/version-audit.md](evidence/version-audit.md)。

## 9. 复现

```bash
# 需要本机已安装 Archify 3.2.0；以下路径相对于仓库根目录
ARCHIFY=<archify skill>/bin/archify.mjs

node "$ARCHIFY" finalize architecture docs/architecture/archify/01-toolchain-panorama.architecture.json \
  docs/architecture/archify/01-toolchain-panorama.html --quality showcase --json
# 其余图同构：按第 1 节表格的"类型"和"可编辑源"替换参数
```

图形是设计表达，不是实现完成或 E2E 通过的证明。
