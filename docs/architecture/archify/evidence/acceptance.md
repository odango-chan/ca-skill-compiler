# 验收记录：Archify 图形集

> 本文件只记录**实际执行**的检查与结果。未执行的检查一律标 NOT_RUN / BLOCKED，不得以“应该可以”代替。
> 本轮（2026-11-20 架构冻结轮）重新生成并验范围为图 01–03；04、05a-ops 与 05a–05g 的源文件未变，保留此前通过的验收记录。
> 随后在**最终评审轮**（同日）对 01–03 再次执行 `finalize` 与 `visual-check`：源文件与生成产物逐字节未变（spec/artifact sha256 与上一轮相同），仅回执 ID、时间戳与暗色截图编码刷新，见 §8。

## 1. 环境

| 项 | 值 |
|---|---|
| Archify Skill 包 | `@tt-a1i/archify-dsh@1.1.0` |
| Archify 渲染器版本 | **3.2.0** |
| 本轮执行环境 | 本机 Windows（DSH danger-full-access） |
| Node.js | v24.18.0 |
| 浏览器 | 真实 Google Chrome 158.0.8127.37（Playwright 自动启动） |
| 质量档 | `showcase` |
| 截图状态 | 01–03 本轮 PASS（1440×900 / 2048×1320 × 明/暗）；最终评审轮再次 PASS，05a–05g 沿用 `evidence/sequence-vnext/` |

## 2. 逐图四道门结果

`finalize` 一次调用依次执行 `validate` / `deliver` / 严格 `check` / 真实浏览器 `browser-check`。
01–03 于 2026-11-20 架构冻结轮重新生成；04、05a-ops、05a–05g 沿用未修改源文件的既有回执。

| 图 | 类型 | validate | deliver | check | browser-check | diagnostics | artifact sha256（前 16 位） | 本轮 |
|---|---|---|---|---|---|---|---|---|
| 01-toolchain-panorama | architecture | PASS | PASS | PASS | PASS | 0 | `9f57122f67c9c77e` | **2026-11-20 重绘** |
| 02-offline-build | workflow | PASS | PASS | PASS | PASS | 0 | `035eddbae1071f8a` | **2026-11-20 重绘** |
| 03-host-admission | architecture | PASS | PASS | PASS | PASS | 0 | `a484e6224f086b87` | **2026-11-20 重绘** |
| 04-scp-amf-smf | architecture | PASS | PASS | PASS | PASS | 0 | `50caeee35c765713` | 保留既有证据 |
| 05a-establishment-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `9fa7d9f575e7981f` | 保留既有证据 |
| 05a-ops-establishment-capabilities | architecture | PASS | PASS | PASS | PASS | 0 | `2730c37fa861a835` | 保留既有证据 |
| 05b-ue-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `481d755d6ad38134` | 保留既有证据 |
| 05c-network-modification-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `3e905f274ccd919a` | 保留既有证据 |
| 05d-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `ca7abf6b85ce337a` | 保留既有证据 |
| 05e-network-release-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `c6179200c8d3204d` | 保留既有证据 |
| 05f-service-request-handover-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `bb8e44d84354f196` | 保留既有证据 |
| 05g-callback-failure-sequence | sequence | PASS | PASS | PASS | PASS | 0 | `f0c8d52e36d70572` | 保留既有证据 |

**结果：01–03 本轮四道门 3/3 PASS，并通过 `visual-check`（3/3，含真实截图）；04 与 05 系列沿用既有四道门 PASS 回执。**

### 2.1 早期受限沙箱回执（已作废）

本轮第一次尝试在禁止管道子进程的 DSH workspace-write 沙箱中运行，`browser-check` 报
`viewer/browser-check-runtime: spawn EPERM`。那批 `NOT_RUN` 回执已被随后的
四道门 PASS 回执替换，并归档在 `evidence/history/freeze-20261120/`；它们不是图形缺陷，
也不再代表当前状态。当前证据以 `evidence/01..03-*.finalize-summary.json` 与
`.browser-check.json` 为准（`gates.browser-check = "pass"`）。

## 3. 真实浏览器截图

- 01–03：本轮 `visual-check` **PASS**，每张图产出 1440×900 与 2048×1320、明/暗共 4 张截图，
  以及接触表和 `visual-check.json`，位于 `evidence/01..03-*.visual-check.*`。
- 上一版截图与回执归档在 `evidence/history/freeze-20261120/`，避免与新源文件混淆。
- 04、05a-ops、05a–05g：继续使用此前归档的截图与回执；05a–05g 当前截图以 `evidence/sequence-vnext/` 为准。

## 4. 自动视觉检查结果

01–03 本轮 `visual-check` 均为 `ok: true`、`diagnostics: []`；05a–05g 七张图此前同样通过：

| 检查项 | 结果 | 范围 |
|---|---|---|
| containment | PASS | 3/3（01–03 本轮）+ 7/7（05a–05g） |
| readability | PASS | 3/3 + 7/7 |
| viewerChrome | PASS | 3/3 + 7/7 |
| themeStates | PASS | 3/3 + 7/7（light / dark） |
| captures | PASS | 3/3 + 7/7（两种视口 × 两种主题） |

**人工目视复核状态：PENDING（01、02）/ 已抽查（03）。** 03 的 1440×900 与 2048×1320 明色截图已人工查看，
主体节点、图例和两类 Requirement 标注可读；01、02 尚未逐张人工签字。自动检查不冒充人工签字。

最终评审轮的代理人目视记录（**不是**人工签字）：本轮查看了
`01-*.1440x900.light.png`、`01-*.1440x900.dark.png`、`02-*.1440x900.light.png`、`03-*.2048x1320.light.png`
四张截图——标题/副标题、三条泳道或四阶段分组、两类 Requirement 文件标签、`NOT_ADMITTED` 与
`COMMIT_PROOF` / `RECONCILE` 执行期标注、图例与查看器外框均可读，深色主题对比度正常。
01、02 的人工逐张签字仍为 **PENDING**，不得用上述记录替代。

## 5. 未执行 / 不适用的检查

| 检查 | 状态 | 原因 |
|---|---|---|
| 01、02 人工逐张目视复核 | PENDING | 03 已抽查两种视口，01、02 尚未逐张签字 |
| 05a–05g 人工逐张目视签字 | PENDING | 自动视觉检查此前已通过，但尚未记录人工复核 |
| 独立 SVG 静态导出 | NOT_RUN | 本次交付以 HTML + JSON 源及真实浏览器截图为准 |
| 图 04 的四责任域矩形边框 | PARTIAL | `region` 边界要求互不重叠的紧凑矩形，见 README §6 |
| E2E / 真实 5GC 验证 | NOT_APPLICABLE | 本任务只做设计表达与图形资产，不触碰运行时 |
| Compiler Link / Host 准入 | NOT_APPLICABLE | 图形不构成 Link 或准入证据 |

## 6. 复核建议（advisory）

图 04 的既有 `finalize-summary.json` 含 `visualReviewRecommendation.action = inspect-route-readability`，
信号为 1 条绕行路线（`store → openamsm`，5 个折点）。这是 Archify 的建议项，不是失败项；
此前已人工确认该虚线可辨识，未再做几何重排。

本轮图 03 的 `finalize` 也给出 1 条建议（`artifact → admission` 3 个折点，`routesOverBends`），
不是失败项。已查看 1440×900 与 2048×1320 明色截图，该路径标注“产物 + 两类记录”可辨识，
未做几何重排。

## 7. 2026-11-20 架构冻结轮（01–03）

- 变更内容：按两类 Requirement 模型重绘 01–03（Host Admission Requirement 与 Runtime Execution Requirement 分离、移除「可选 Assurance Export」、修正 Host 准入材料清单、Host 准入与 runtime 执行结果分开表达）。
- 实际执行命令（`$ARCHIFY` 为 `@tt-a1i/archify-dsh` 的 `skills/archify/bin/archify.mjs`；在允许子进程的环境中直接执行，无 preload shim）：

  ```bash
  node "$ARCHIFY" finalize architecture docs/architecture/archify/01-toolchain-panorama.architecture.json docs/architecture/archify/01-toolchain-panorama.html --quality showcase --json
  # 02 用 workflow + 02-offline-build.workflow.json；03 用 architecture + 03-host-admission.architecture.json
  node "$ARCHIFY" visual-check docs/architecture/archify/<figure>.html --require-provenance --out-dir docs/architecture/archify/evidence --summary
  ```

- 回执：`evidence/<figure>.finalize-summary.json` 与 `evidence/<figure>.browser-check.json` 均记录
  `gates.browser-check = "pass"`、`ok: true`、`diagnostics: []`；`evidence/<figure>.visual-check.json`
  记录 containment / readability / viewerChrome / themeStates / captures 全部 PASS。
- 四份截图（两种视口 × 两种主题）与接触表位于 `evidence/01..03-*.visual-check.*`。
- 早期受限沙箱产生的 `NOT_RUN` 回执与旧截图归档在 `evidence/history/freeze-20261120/`，仅作历史痕迹。

## 8. 2026-11-20 最终评审轮（01–03 重新验收）

- 目的：确认“源与生成产物一致”，并在最终提交上重新执行四道门与视觉检查。
- 源未改：三份 spec 的 sha256 与上轮相同（01 `b450f4da…`、02 `8df239e6…`、03 `8f76300a…`），
  生成的 HTML artifact sha256 也未变（01 `9f57122f67c9c77e`、02 `035eddbae1071f8a`、03 `a484e6224f086b87`），
  说明生成产物确实由当前源产出，不存在源/产物漂移。
- 实际执行命令（`$ARCHIFY` = `C:\Users\z00587616\.dsh\profiles\desktop\node_modules\@tt-a1i\archify-dsh\skills\archify\bin\archify.mjs`，
  工作目录为仓库根）：

  ```bash
  node "$ARCHIFY" finalize architecture docs/architecture/archify/01-toolchain-panorama.architecture.json docs/architecture/archify/01-toolchain-panorama.html --quality showcase --json
  node "$ARCHIFY" finalize workflow     docs/architecture/archify/02-offline-build.workflow.json        docs/architecture/archify/02-offline-build.html          --quality showcase --json
  node "$ARCHIFY" finalize architecture docs/architecture/archify/03-host-admission.architecture.json  docs/architecture/archify/03-host-admission.html        --quality showcase --json
  node "$ARCHIFY" visual-check docs/architecture/archify/01-toolchain-panorama.html --require-provenance --out-dir docs/architecture/archify/evidence --summary
  node "$ARCHIFY" visual-check docs/architecture/archify/02-offline-build.html       --require-provenance --out-dir docs/architecture/archify/evidence --summary
  node "$ARCHIFY" visual-check docs/architecture/archify/03-host-admission.html      --require-provenance --out-dir docs/architecture/archify/evidence --summary
  ```

- 结果：三次 `finalize` 均 `ok: true`，`gates.{validate,deliver,check,browser-check} = pass`，
  `diagnostics: []`；三次 `visual-check` 均 `ok: true`、`provenance: current`，
  containment / readability / viewerChrome / themeStates / captures 全部 `pass`。
- 03 仍给出建议项 `inspect-route-readability`（`artifact → admission` 3 个折点）。已查看
  1440×900 与 2048×1320 截图，该绿色路径可辨识，不是失败项，不做几何重排。
- 本轮第一次 `finalize` 在禁止子进程的沙箱中尝试，返回 `finalize/runtime: spawn EPERM`；
  该 NOT_RUN 结果不作数，随后在允许子进程的环境中重新执行并取得上述 PASS 回执。
- 人工逐张签字：01、02 **仍为 PENDING**；03 保持“已抽查”。自动化与代理人目视均不替代人工签字。
