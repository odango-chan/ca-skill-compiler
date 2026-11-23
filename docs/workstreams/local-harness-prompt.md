> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# DeepSeek / OpenCode Harness · ca-skill-compiler (breaking-only)

你需要**直接实现代码并验收**，不是继续整理架构文档。仓库：`odango-chan/ca-skill-compiler`。从最新 `origin/feat/compiler-v0-bootstrap` 创建独立实现分支；**不要从 main 或旧 ca-skill-toolchain 开始**。

依次阅读：
1. `AGENTS.md`
2. `docs/design/skill-compiler-v0.md`
3. `docs/design/script-contract-v0.md`
4. `docs/design/procedure-link-v0.md`
5. `docs/workstreams/acceptance-matrix-v0.md`
6. `docs/workstreams/local-implementation-goal.md`（WP0–WP7）
7. `docs/studio-integration.md`
8. `docs/WORK.md`

关键纠偏：**无需兼容旧格式**。两套 ca-agent-demo 真实 Skill 是迁移前参考，必须原样保留以做差异/诊断，**不要求旧脚本原样编译 PASS**。不要实现旧 `connectionagent-tool` Script 头部兼容分支、旧 external/embedded SchemaMode、Release/Qualification/DB、历史 Runtime Artifact；不得通过放宽校验或捏造 Host/Observation Schema 让测试变绿。

唯一新规则：从 `lib/` 导入 Tool/Host/Observation 的**真实 Producer Schema**；Procedure 的 `with:` 将外部输入绑定到 Script，按调用点解析类型；Script 只声明自己计算得到的严格 Output TypedDict 并生成 Output Schema。不要在 Script 中重复定义 Tool Input TypedDict。缺少权威外部契约必须有明确错误，不可默认 object。

优先推进：WP0 Go 编译 + 测试 → WP1 AST/Scope/Graph → WP2 ToolCatalog + manifest/Inspector API → WP3 Binding + Script Output Schema → WP4 CLI package → WP5 WASM/Chromium → WP6 真实 Skill 迁移报告（可在隔离目录转换，不改原快照）→ WP7 实际验收与 PR。

允许自动提交/推送、创建 **目标分支为 feat/compiler-v0-bootstrap** 的 PR；不能强推、不能在测试失败时合并 main、不能写其他仓或运行真实 5GC。按 G0–G9 回报真实测试证据，未跑就标 NOT_RUN，Owner 缺 Schema 就标 BLOCKED，更新 docs/WORK.md。不要停止于计划或再次交接。
