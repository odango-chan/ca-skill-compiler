# 验收记录：Archify 图形集

> 本文件只记录**实际执行**的检查与结果。未执行的检查一律标 NOT_RUN，不得以"应该可以"代替。

## 1. 环境

| 项 | 值 |
|---|---|
| Archify Skill 包 | `@tt-a1i/archify-dsh@1.1.0` |
| Archify 渲染器版本 | **3.2.0**（Skill 内 `package.json` 的 `version`） |
| Node.js | v24.18.0 |
| 浏览器 | Google Chrome `C:\Program Files\Google\Chrome\Application\chrome.exe` |
| 质量档 | `showcase`（12 张图全部） |
| `visual_preset` | 未设置（统一使用 `classic`，与 01–04 一致） |

## 2. 逐图四道门结果

`finalize` 一次调用依次执行 `validate` / `deliver` / 严格 `check` / 真实浏览器 `browser-check`。
下表全部来自 `finalize-summary.json` 的实际返回值。

| 图 | 类型 | validate | deliver | check | browser-check | diagnostics | artifact sha256（前 16 位） |
|---|---|---|---|---|---|---|---|
| 01-toolchain-panorama | architecture | PASS | PASS | PASS | PASS | 0 | `e4318de8e3db7f95` |
| 02-offline-build | workflow | PASS | PASS | PASS | PASS | 0 | `e135ca29a1e0f907` |
| 03-host-admission | architecture | PASS | PASS | PASS | PASS | 0 | `b9480171e3afbc81` |
| 04-scp-amf-smf | architecture | PASS | PASS | PASS | PASS | 0 | `50caeee35c765713` |
| 05a-establishment-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `5db44c51f467b6d7` |
| 05a-ops-establishment-capabilities | architecture | PASS | PASS | PASS | PASS | 0 | `2730c37fa861a835` |
| 05b-ue-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `888346351a0507b0` |
| 05c-network-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `6025e3fe1523132f` |
| 05d-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `42219294808dcfa1` |
| 05e-network-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `5927727fe34a921d` |
| 05f-service-request-handover-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `86a660423eb31201` |
| 05g-callback-failure-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `8ee8be8ef158ceae` |

**结论：12/12 通过四道门，`diagnostics: []`（0 error、0 warning）。**

`finalize` 不等于"渲染成功"：其中 `validate` 是模式与布局校验，`check` 是严格产物检查（含 provenance），
`browser-check` 是真实 Chromium 在 1440×900 / 1600×1000 / 1920×1080 / 2048×1320 与明暗主题下的
溢出与可读性测量。

## 3. 真实浏览器截图

`visual-check --require-provenance` 逐图执行，产出 1440×900 与 2048×1320、明/暗四张截图。
下表为归档到本目录的明色两档（暗色截图保留在本地工作目录，未入库）。

| 图 | 1440×900 | 2048×1320 | visual-check 结果 |
|---|---|---|---|
| 01-toolchain-panorama | `01-toolchain-panorama.1440x900.png` | `01-toolchain-panorama.2048x1320.png` | PASS |
| 02-offline-build | `02-offline-build.1440x900.png` | `02-offline-build.2048x1320.png` | PASS |
| 03-host-admission | `03-host-admission.1440x900.png` | `03-host-admission.2048x1320.png` | PASS |
| 04-scp-amf-smf | `04-scp-amf-smf.1440x900.png` | `04-scp-amf-smf.2048x1320.png` | PASS |
| 05a-establishment-sequence | `05a-establishment-sequence.1440x900.png` | `05a-establishment-sequence.2048x1320.png` | PASS |
| 05a-ops-establishment-capabilities | `05a-ops-establishment-capabilities.1440x900.png` | `05a-ops-establishment-capabilities.2048x1320.png` | PASS |
| 05b-ue-modification-sequence | `05b-ue-modification-sequence.1440x900.png` | `05b-ue-modification-sequence.2048x1320.png` | PASS |
| 05c-network-modification-sequence | `05c-network-modification-sequence.1440x900.png` | `05c-network-modification-sequence.2048x1320.png` | PASS |
| 05d-release-sequence | `05d-release-sequence.1440x900.png` | `05d-release-sequence.2048x1320.png` | PASS |
| 05e-network-release-sequence | `05e-network-release-sequence.1440x900.png` | `05e-network-release-sequence.2048x1320.png` | PASS |
| 05f-service-request-handover-sequence | `05f-service-request-handover-sequence.1440x900.png` | `05f-service-request-handover-sequence.2048x1320.png` | PASS |
| 05g-callback-failure-sequence | `05g-callback-failure-sequence.1440x900.png` | `05g-callback-failure-sequence.2048x1320.png` | PASS |

截图是 `visual-check` 的真实 Chromium 捕获，不是静态生成或占位图。逐图回执见
`<figure>.visual-check.json`。

## 4. 人工目视检查

在真实截图（1440×900）上逐项确认：

| 检查项 | 结果 | 说明 |
|---|---|---|
| 时间自上而下、请求/响应方向 | PASS | 全部 7 张 sequence |
| 泳道/参与者命名与四域归属 | PASS | `CA-Agent Host`、`Host Runtime`、`Client Tools`、`OpenAMSM SMF`、`SCP` 等 |
| 跨域箭头标注协议或接口 | PASS | 例如 `Nsmf_PDUSession_CreateSMContext`、`Namf_Communication_N1N2MessageTransfer`、`PFCP Establishment Type 50`、`N4 / PFCP` |
| 实线/虚线 + 状态文字 | PASS | 虚线用于 DESIGN / NOT_IMPLEMENTED 路径 |
| 分支表达（alt/phase） | PASS（有偏差） | 用命名 phase 段 + 标签前缀，Archify sequence 无 UML 组合片段边框 |
| 1440 宽度可读 | PASS | 截图逐张目视确认 |
| 2048 输出无越界/裁切 | PASS | `browser-check` 在 2048×1320 无横向溢出 |
| 字体、箭头、标签无重叠 | PASS | sequence 由 `sequence/note-overlap`、`composition/label-*` 门保证 |

## 5. 未执行的检查（明确列出）

| 检查 | 状态 | 原因 |
|---|---|---|
| 逐图 SVG / PNG 静态导出 | NOT_RUN | 本次交付以 HTML + JSON 源为准；`browser-check` 已提供真实截图 |
| 图 04 的四责任域矩形边框 | PARTIAL | 见 README §5：`region` 边界要求互不重叠的紧凑矩形，Client Tools 节点位于 CA-Agent 矩形内的消息路径上 |
| E2E / 真实 5GC 验证 | NOT_APPLICABLE | 本任务只做设计表达与图形资产，不触碰运行时 |
| Compiler Link / Host 准入 | NOT_APPLICABLE | 图形不构成 Link 或准入证据 |

## 6. 复核建议（advisory）

图 04 的 `finalize-summary.json` 含 `visualReviewRecommendation.action = inspect-route-readability`，
信号为 1 条绕行路线（`store → openamsm`，5 个折点）。这是 Archify 的建议项，不是失败项；
已人工目视确认该虚线可辨识，未再做几何重排。
