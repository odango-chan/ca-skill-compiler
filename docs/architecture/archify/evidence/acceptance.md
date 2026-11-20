# 验收记录：Archify 图形集

> 本文件只记录**实际执行**的检查与结果。未执行的检查一律标 NOT_RUN / BLOCKED，不得以“应该可以”代替。
> 本轮（2026-11-20 架构冻结轮）重新生成并验范围为图 01–03；04、05a-ops 与 05a–05g 的源文件未变，保留此前通过的验收记录。

## 1. 环境

| 项 | 值 |
|---|---|
| Archify Skill 包 | `@tt-a1i/archify-dsh@1.1.0` |
| Archify 渲染器版本 | **3.2.0** |
| 本轮执行环境 | 本机 Windows（DSH workspace-write 沙箱） |
| Node.js | v24.18.0 |
| 浏览器 | Chromium（Playwright）— **本轮无法启动，见 §2.1** |
| 质量档 | `showcase` |
| 截图状态 | 01–03 本轮 NOT_RUN（浏览器不可用）；05a–05g 沿用 `evidence/sequence-vnext/` |

## 2. 逐图四道门结果

`finalize` 一次调用依次执行 `validate` / `deliver` / 严格 `check` / 真实浏览器 `browser-check`。
01–03 于 2026-11-20 架构冻结轮重新生成；04、05a-ops、05a–05g 沿用未修改源文件的既有回执。

| 图 | 类型 | validate | deliver | check | browser-check | diagnostics | artifact sha256（前 16 位） | 本轮 |
|---|---|---|---|---|---|---|---|---|
| 01-toolchain-panorama | architecture | PASS | PASS | PASS | **NOT_RUN (BLOCKED)** | 1（browser-check 运行时） | `9f57122f67c9c77e` | **2026-11-20 重绘** |
| 02-offline-build | workflow | PASS | PASS | PASS | **NOT_RUN (BLOCKED)** | 1（browser-check 运行时） | `035eddbae1071f8a` | **2026-11-20 重绘** |
| 03-host-admission | architecture | PASS | PASS | PASS | **NOT_RUN (BLOCKED)** | 1（browser-check 运行时） | `a484e6224f086b87` | **2026-11-20 重绘** |
| 04-scp-amf-smf | architecture | PASS | PASS | PASS | PASS | 0 | `50caeee35c765713` | 保留既有证据 |
| 05a-establishment-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `9fa7d9f575e7981f` | 保留既有证据 |
| 05a-ops-establishment-capabilities | architecture | PASS | PASS | PASS | PASS | 0 | `2730c37fa861a835` | 保留既有证据 |
| 05b-ue-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `481d755d6ad38134` | 保留既有证据 |
| 05c-network-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `3e905f274ccd919a` | 保留既有证据 |
| 05d-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `ca7abf6b85ce337a` | 保留既有证据 |
| 05e-network-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `c6179200c8d3204d` | 保留既有证据 |
| 05f-service-request-handover-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `bb8e44d84354f196` | 保留既有证据 |
| 05g-callback-failure-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `f0c8d52e36d70572` | 保留既有证据 |

**结果：01–03 本轮 validate / deliver / check 三道门 3/3 PASS；browser-check 因浏览器无法启动而 NOT_RUN（BLOCKED）。04 与 05 系列沿用既有四道门 PASS 回执。**

### 2.1 browser-check 阻塞原因（精确错误）

```
viewer/browser-check-runtime:
  browser-check could not complete its Chrome inspection.
  evidence.reason = "spawn EPERM"
```

本机 DSH 沙箱禁止进程以管道方式创建子进程（`spawn` / `spawnSync` 带 pipe stdio 均返回 `EPERM`），
Playwright 启动 Chromium 需要该能力，因此 Chromium 无法启动。该阻塞不是图形缺陷：
`validate` / `deliver` / 严格 `check`（含真实的合成与版面检查）全部通过。
为让这三道门在本沙箱内可运行，本轮通过 `NODE_OPTIONS=--require .archify/pipe-shim.cjs`
把流水线内部的管道 stdio 重定向到临时普通文件（不修改 Archify 包、不手工编辑 HTML）。
该 shim 只位于被 gitignore 的 `.archify/`，不随产物提交。

**不得把本轮 01–03 描述为“四道门 PASS”或“视觉验收通过”。**

## 3. 真实浏览器截图

- 01–03：本轮 `visual-check` **NOT_RUN (BLOCKED)**，原因同上（Chromium 无法启动）。上一版截图与回执已归档到
  `evidence/history/freeze-20261120/`，避免与新源文件混淆；当前目录不再保留 01–03 的旧截图。
- 04、05a-ops、05a–05g：继续使用此前归档的截图与回执；05a–05g 当前截图以 `evidence/sequence-vnext/` 为准。

## 4. 自动视觉检查结果

05a–05g 七张图此前的 `visual-check` 均为 `ok: true`、`diagnostics: []`：

| 检查项 | 结果 | 范围 |
|---|---|---|
| containment | PASS | 7/7 |
| readability | PASS | 7/7 |
| viewerChrome | PASS | 7/7 |
| themeStates | PASS | 7/7（light / dark） |
| captures | PASS | 7/7（两种视口 × 两种主题） |

**人工目视复核状态：PENDING。** 本文件只声明自动浏览器检查通过，不把自动检查冒充人工目视签字。
01–03 本轮没有自动或人工视觉检查结果。

## 5. 未执行 / 不适用的检查

| 检查 | 状态 | 原因 |
|---|---|---|
| 01–03 `browser-check`（Chromium） | **NOT_RUN (BLOCKED)** | 沙箱禁止管道子进程，Playwright 无法启动 Chromium（`spawn EPERM`） |
| 01–03 `visual-check` 截图 | **NOT_RUN (BLOCKED)** | 同上 |
| 01–03 人工目视复核 | PENDING | 无当前源文件的截图可复核 |
| 05a–05g 人工逐张目视签字 | PENDING | 自动视觉检查此前已通过，但尚未记录人工复核 |
| 独立 SVG 静态导出 | NOT_RUN | 本次交付以 HTML + JSON 源及真实浏览器截图为准 |
| 图 04 的四责任域矩形边框 | PARTIAL | `region` 边界要求互不重叠的紧凑矩形，见 README §6 |
| E2E / 真实 5GC 验证 | NOT_APPLICABLE | 本任务只做设计表达与图形资产，不触碰运行时 |
| Compiler Link / Host 准入 | NOT_APPLICABLE | 图形不构成 Link 或准入证据 |

## 6. 复核建议（advisory）

图 04 的既有 `finalize-summary.json` 含 `visualReviewRecommendation.action = inspect-route-readability`，
信号为 1 条绕行路线（`store → openamsm`，5 个折点）。这是 Archify 的建议项，不是失败项；
此前已人工确认该虚线可辨识，未再做几何重排。

## 7. 2026-11-20 架构冻结轮（01–03）

- 变更内容：按两类 Requirement 模型重绘 01–03（Host Admission Requirement 与 Runtime Execution Requirement 分离、移除「可选 Assurance Export」、修正 Host 准入材料清单、Host 准入与 runtime 执行结果分开表达）。
- 实际执行命令（`$ARCHIFY` 为 `@tt-a1i/archify-dsh` 的 `skills/archify/bin/archify.mjs`）：

  ```bash
  node --require .archify/pipe-shim.cjs "$ARCHIFY" finalize architecture \
    docs/architecture/archify/01-toolchain-panorama.architecture.json \
    docs/architecture/archify/01-toolchain-panorama.html \
    --quality showcase --json --out-dir .archify/archify-freeze-20261120/f01
  # 02 用 workflow + 02-offline-build.workflow.json；03 用 architecture + 03-host-admission.architecture.json
  ```

- 回执已复制到 `evidence/<figure>.finalize-summary.json` 与 `evidence/<figure>.browser-check.json`；
  两条回执都如实记录 `gates.browser-check = "fail"`、`failedStage = "browser-check"`、`reason = "spawn EPERM"`。
- 因此本轮**没有**新的 1440×900 / 2048×1320 截图；需要真实浏览器证据时必须在允许子进程管道、或已安装 Playwright Chromium 的环境重新执行 `finalize` 与 `visual-check`。
