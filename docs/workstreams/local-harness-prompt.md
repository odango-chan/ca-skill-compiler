# 本地弱模型执行提示词 · ca-skill-compiler v0

你在本地仓库 `odango-chan/ca-skill-compiler` 执行实现任务，不是做审计/设计交接。**一次执行尽量完成 WP0–WP7**。必须先完整阅读：
- `docs/workstreams/local-implementation-goal.md`（唯一执行任务书）
- `docs/design/skill-compiler-v0.md`（架构约束）
- `docs/studio-integration.md`
- `docs/design/script-contract-v0.md`（Studio/WASM 消费接口）
- `docs/WORK.md`（事实状态）
- `examples/ca-agent-demo-project/README.md`（真实样例与 MAS 非 READY 边界）
- `AGENTS.md`

基线要求：从最新远端 `origin/feat/compiler-v0-bootstrap` 开始；在该 commit 上创建新实现分支，不从当前 `main`（仅有初始 README）开始，也不把旧 `ca-skill-toolchain` 复制过来。先运行 go test/go build 记录初始错误，修复。然后完成两个真实 Skill 的静态构建、ToolCatalog 导入、Schema/依赖/Procedure 校验、CLI 与 WASM，提供有实际结果的测试证据。脚本不得执行、Tool/MCP 不得调用。MAS 候选 ToolCatalog 只用于静态链接，不能声称 READY。

开发原则：重构是允许的，优先小而清晰的 Go Core。凡已发现缺陷，先补失败测试再修代码；不可修改真实 Skill 快照或放宽校验以得到绿色。Native 和 WASM 共享 Core。尽量实现完成的功能，别新增 DB/Qualification/Release/HTTP Server/复杂 Artifact 系统/泛化插件框架。

权限：可以在新仓新分支内自主编辑、执行本地测试、提交、推送、创建 PR（目标分支为 feat/compiler-v0-bootstrap）；不 force push，不对其他仓库写入，不启动真实 5GC，不触发 GitHub Actions，不在测试失败时合并 main。

最终必须输出：分支+SHA+PR、WP0–WP7 状态、Go test/build/wasm 的真实结果、两个真实 Skill 的诊断、未闭合项，并同步 docs/WORK.md。不要只给后续规划或让另一个模型接手。
