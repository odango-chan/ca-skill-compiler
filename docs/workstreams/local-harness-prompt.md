# DeepSeek / OpenCode Harness · ca-skill-compiler (breaking-only)

你需要**直接实现代码并验收**，不是继续整理架构文档。仓库：`odango-chan/ca-skill-compiler`。从最新 `origin/feat/compiler-v0-bootstrap` 创建独立实现分支；**不要从 main 或旧 ca-skill-toolchain 开始**。

> **已归档 / ARCHIVED — PRE-v1alpha4.** 本文记录早期 v0/v0.1 原型、迁移检查或设计阶段的验收结果。文中的语法、分支/PR 指令、样例诊断数量和验收结论均属于历史快照，不定义 CA-SL v1alpha4 或 Compiler v1.0.0 的当前要求。当前状态请以[工作流索引](README.md)和[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)为准。


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
