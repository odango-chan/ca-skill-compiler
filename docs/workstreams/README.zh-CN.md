# 工作流记录

[English](README.md)

本目录保存编译器实施计划、分阶段工作包记录和历史迁移报告。它不是当前任务队列。

## 当前发布状态

CA Skill Compiler v1.0.0 的当前状态以以下文件为准：

- [Compiler 实施与验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) — 权威的实现状态、本地验证证据、E-case 处置结果和产品符合性边界。
- [已知限制](../known-limitations-v1alpha4.md) — 当前 fail-closed 行为及外部阻塞项。
- [剩余工作清单](REMAINING-WORK-v1alpha4.json) — 机器可读清单。当前 `implementableInRepo` 为空；剩余项目需要外部发布物或规范澄清，不是待执行的代码任务。
- [符合性测试计划](conformance-v1alpha4.md) — 已冻结的测试策略，不是实时进度表。当前执行结果以最终验收记录和 `tests/conformance/v1alpha4/e-coverage.json` 为准。

## 已完成的 v1alpha4 实施记录

以下工作包说明记录的实现均已合并到 `main`，并包含在 CA Skill Compiler v1.0.0 中。文档里的原始分支名、Draft 审查状态和停止条件均为历史信息。

| 工作包 | 合并 PR | 记录 |
|---|---:|---|
| WP3A — 依赖身份 | [#15](https://github.com/odango-chan/ca-skill-compiler/pull/15) | [WP3A](wp3a-dependency-identity-v1alpha4.md) |
| WP3B — Owner/导出项链接 | [#16](https://github.com/odango-chan/ca-skill-compiler/pull/16) | [WP3B](wp3b-owner-export-linking-v1alpha4.md) |
| WP3C — Capability 链接 | [#17](https://github.com/odango-chan/ca-skill-compiler/pull/17) | [WP3C](wp3c-capability-linking-v1alpha4.md) |
| WP3D — 集成依赖链接 | [#18](https://github.com/odango-chan/ca-skill-compiler/pull/18) | [WP3D](wp3d-integrated-linking-v1alpha4.md) |
| WP4A/WP4B — Schema 白名单与注册表 | [#19](https://github.com/odango-chan/ca-skill-compiler/pull/19) | [WP4A](wp4a-schema-g4-v1alpha4.md)、[WP4B](wp4b-schema-registry-v1alpha4.md) |
| WP4C — Schema 可赋值性 | [#20](https://github.com/odango-chan/ca-skill-compiler/pull/20) | [WP4C](wp4c-assignability-v1alpha4.md) |
| WP4D — CEL 与 Predicate 签名 | [#21](https://github.com/odango-chan/ca-skill-compiler/pull/21) | [WP4D](wp4d-cel-v1alpha4.md) |
| WP5 — Procedure Graph | [#22](https://github.com/odango-chan/ca-skill-compiler/pull/22) | [WP5](wp5-procedure-graph-v1alpha4.md) |
| WP6 — SourcePath 与数据流 | [#23](https://github.com/odango-chan/ca-skill-compiler/pull/23) | 见[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) |
| WP9–WP12 — Script ABI、制品、Parity 与 Studio 集成 | [#24](https://github.com/odango-chan/ca-skill-compiler/pull/24) | [WP11](wp11-parity-v1alpha4.md)、[WP12](wp12-studio-integration-v1alpha4.md)、[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) |

WP0–WP2、WP7–WP8 和 CLI 的实现状态见已合并的契约/实现 PR 及最终验收记录。剩余的 Evidence、身份来源证明和跨快照 Reconcile 问题属于外部契约阻塞，必须继续 fail-closed，不能擅自标记为 PASS。真实 5GC/MAS 产品资格仍为 NOT_QUALIFIED。

## 已归档的执行计划

以下文档保存 v1alpha4 实施期间的 Harness 工作流和维护者决策，仅供追溯，不再是当前执行指令：

- [DeepSeek 交接说明](deepseek-handoff-v1alpha4.md)
- [DeepSeek Harness 工作规范](deepseek-harness-v1alpha4.md)
- [v1alpha4 继续实施决策](v1alpha4-resume-decision-20261116.md)

## 已归档的 v0 / v0.1 记录

以下文档描述已被取代的原型、Schema、测试样例和迁移检查。保留它们是为了追溯历史，不得将其作为当前 CA-SL v1alpha4 或 Compiler v1.0.0 的要求：

- [v0 验收矩阵](acceptance-matrix-v0.md)
- [v0 实施 Goal](local-implementation-goal.md)
- [v0 Harness 提示词](local-harness-prompt.md)
- [v0 PDU/MAS 迁移评审](migration-review-v0.md)
- [v0.1 JSON Schema 支持矩阵](json-schema-support-v0.1.md)
- [v0.1 语言对齐报告](language-alignment-v0.1.md)
- [v0.1 官方 Skill 验收报告](official-skill-acceptance-v0.1.md)
- [v0.1 ToolCatalog 报告](tool-catalog-v0.1.md)

不删除任何历史报告。查看当前发布状态时，请从最终验收记录和已知限制开始，而不是从旧工作包提示词开始。
