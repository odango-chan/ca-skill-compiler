# 验收记录：Archify 图形集

> 本文件只记录**实际执行**的检查与结果。未执行的检查一律标 NOT_RUN，不得以“应该可以”代替。
> 本轮重新验收范围为 05a–05g 七张业务时序图；01–04 与 05a-ops 的源文件未变，保留此前通过的验收记录。

## 1. 环境

| 项 | 值 |
|---|---|
| Archify Skill 包 | `@tt-a1i/archify-dsh@1.1.0` |
| Archify 渲染器版本 | **3.2.0** |
| 本轮执行环境 | GitHub Actions `ubuntu-24.04` |
| Node.js | 24.x |
| 浏览器 | Playwright 1.56.1 / Chromium 1194 |
| 质量档 | `showcase` |
| 截图状态 | 1440×900 与 2048×1320，light / dark |

## 2. 逐图四道门结果

`finalize` 一次调用依次执行 `validate` / `deliver` / 严格 `check` / 真实浏览器 `browser-check`。
05a–05g 的结果来自 2026-11-20 GitHub Actions run [#44659918942](https://github.com/odango-chan/ca-skill-compiler/actions/runs/44659918942)；
01–04 与 05a-ops 的结果沿用未修改源文件的既有验收回执。

| 图 | 类型 | validate | deliver | check | browser-check | diagnostics | artifact sha256（前 16 位） | 本轮 |
|---|---|---|---|---|---|---|---|---|
| 01-toolchain-panorama | architecture | PASS | PASS | PASS | PASS | 0 | `e4318de8e3db7f95` | 保留既有证据 |
| 02-offline-build | workflow | PASS | PASS | PASS | PASS | 0 | `e135ca29a1e0f907` | 保留既有证据 |
| 03-host-admission | architecture | PASS | PASS | PASS | PASS | 0 | `b9480171e3afbc81` | 保留既有证据 |
| 04-scp-amf-smf | architecture | PASS | PASS | PASS | PASS | 0 | `50caeee35c765713` | 保留既有证据 |
| 05a-establishment-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `3cf05b2997ad9278` | **2026-11-20 刷新** |
| 05a-ops-establishment-capabilities | architecture | PASS | PASS | PASS | PASS | 0 | `2730c37fa861a835` | 保留既有证据 |
| 05b-ue-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `623993b7714abb46` | **2026-11-20 刷新** |
| 05c-network-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `a4ccd845ab49dba` | **2026-11-20 刷新** |
| 05d-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `9ac2a50cb5b76dcb` | **2026-11-20 刷新** |
| 05e-network-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `d7cebc4eb4bdb220` | **2026-11-20 刷新** |
| 05f-service-request-handover-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `e9f57e3773d4302d` | **2026-11-20 刷新** |
| 05g-callback-failure-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `b5ea01393e4096ab` | **2026-11-20 刷新** |

**结果：12 张图均有四道门 PASS 回执；本轮重新执行的 05a–05g 为 7/7 PASS，全部 `diagnostics: []`。**

`browser-check` 在真实 Chromium 中测量 1440×900 / 1600×1000 / 1920×1080 / 2048×1320 的溢出与可读性。
这不是 E2E 或真实 5GC 验收。

## 3. 真实浏览器截图

本轮对 05a–05g 执行 `visual-check --require-provenance`，每张图均生成四张真实 Chromium 截图：
1440×900 light / dark，以及 2048×1320 light / dark。所有截图、接触表和 JSON 回执均提交在
`evidence/sequence-vnext/`。

| 图 | 1440×900 | 2048×1320 | visual-check |
|---|---|---|---|
| 05a-establishment-sequence | light / dark | light / dark | PASS |
| 05b-ue-modification-sequence | light / dark | light / dark | PASS |
| 05c-network-modification-sequence | light / dark | light / dark | PASS |
| 05d-release-sequence | light / dark | light / dark | PASS |
| 05e-network-release-sequence | light / dark | light / dark | PASS |
| 05f-service-request-handover-sequence | light / dark | light / dark | PASS |
| 05g-callback-failure-sequence | light / dark | light / dark | PASS |

文件命名为 `<figure>.visual-check.<width>x<height>.<light|dark>.png`；
逐图机器回执为 `<figure>.visual-check.json`，接触表为 `<figure>.visual-check.html`。
例如：[05a 当前视觉验收目录](sequence-vnext/)。

01–04 与 05a-ops 未在本轮修改，继续使用根目录 `evidence/` 中此前归档的截图与回执。
根目录中 05a–05g 同名截图属于前一版源文件的历史快照；当前截图以 `sequence-vnext/` 为准。

## 4. 自动视觉检查结果

05a–05g 七张图的 `visual-check` 均为 `ok: true`、`diagnostics: []`：

| 检查项 | 结果 | 范围 |
|---|---|---|
| containment | PASS | 7/7 |
| readability | PASS | 7/7 |
| viewerChrome | PASS | 7/7 |
| themeStates | PASS | 7/7（light / dark） |
| captures | PASS | 7/7（两种视口 × 两种主题） |

**人工目视复核状态：PENDING。** 当前回执中的 `visualReview` 仍为 `pending`；本文件只声明自动浏览器检查通过，不把自动检查冒充人工目视签字。

## 5. 未执行 / 不适用的检查

| 检查 | 状态 | 原因 |
|---|---|---|
| 05a–05g 人工逐张目视签字 | PENDING | 自动视觉检查已通过，但尚未记录人工复核 |
| 独立 SVG 静态导出 | NOT_RUN | 本次交付以 HTML + JSON 源及真实浏览器截图为准 |
| 图 04 的四责任域矩形边框 | PARTIAL | `region` 边界要求互不重叠的紧凑矩形，见 README §6 |
| E2E / 真实 5GC 验证 | NOT_APPLICABLE | 本任务只做设计表达与图形资产，不触碰运行时 |
| Compiler Link / Host 准入 | NOT_APPLICABLE | 图形不构成 Link 或准入证据 |

## 6. 复核建议（advisory）

图 04 的既有 `finalize-summary.json` 含 `visualReviewRecommendation.action = inspect-route-readability`，
信号为 1 条绕行路线（`store → openamsm`，5 个折点）。这是 Archify 的建议项，不是失败项；
此前已人工确认该虚线可辨识，未再做几何重排。
