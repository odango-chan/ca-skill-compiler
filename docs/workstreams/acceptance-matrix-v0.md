> **HISTORICAL / NOT CURRENT INPUT (v2.1.0).** This design/work-package document is retained for original source and acceptance provenance only. The active Compiler accepts **CA-SL v1alpha5**; earlier versions are not compatibility modes. Original status statements and instructions below describe their historical snapshot. See [documentation](../README.md), [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), [Development contracts](../../contracts/development/v1alpha1/README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Compiler v0 · 单一验收矩阵（弱模型执行版）

状态：全部初始 **NOT_RUN**。本表与 [Script Contract](../design/script-contract-v0.md)、[Procedure Link](../design/procedure-link-v0.md) 一起构成可验收的目标，不是对现有实现的声明。

## 1. 实际样本：两套不能混为一谈

- **绿色正例**：`examples/minimal-project`（双 Skill + ToolCatalog）与 `examples/script-contract-project`（Tool Output → with → Script Output 类型生成）。
- **破坏性迁移输入**：`examples/ca-agent-demo-project`，包含两份从 `ca-agent-demo@8ab5145b` 固定的原始 Skill。保留原始文件以便对比：**不要求未经迁移的旧 Script/Schema/Procedure 原样 PASS**。
- **新版迁移产物**：如需真实 PDU Session 工程在 v0 完成编译，建立**独立** `examples/ca-agent-demo-v0/`；按新规范迁移源码，**不静默改写 pinned 参考文件**。尚缺真实 Host/Observation Schema 的部分标为 BLOCKED/FAIL，禁止自造。
- **Tool Library**：真实 12 Operation 的 `lib/tools/createdsession.tool.yaml` 与 MAS 16 Operation 的 `mas-proposed.tool.yaml` 已导入；这只证明可进行静态引用，不代表 Tool 实现 READY。
- 不能只数“文件存在”就宣布正例 PASS，必须以 CLI/Go Test 的实际结果为准。

## 2. 阶段门槛

| Gate | 正向/反向测试 | 成功标准 |
| --- | --- | --- |
| G0 基础启动 | `go mod tidy`、`go test ./... -count=1`、`go vet ./...` | 实际执行 PASS；go.sum 提交；无 panic |
| G1 Library | 12 + 16 ToolCatalog、YAML Anchor/Alias、重复 Operation/缺 Schema | 正常解析；重复/无字段有精确诊断 |
| G2 Graph | 多 Skill 依赖、子 Procedure、branch、preconditions；未知目标、循环 | 无 AST 递归误判；目标/闭包正确 |
| G3 Binding | Skill request + Tool Output、nested preconditions、缺 Host/Tool 字段 | 从真实 Producer 获得 Schema；缺失 fail closed |
| G4 Script | 新 strict TypedDict Output、Annotated/NotRequired/Literal；宽泛输出、旧 header 输入 | 只生成自身 Output Schema；旧格式拒绝或明确迁移诊断 |
| G5 Package | 全项目/单 Skill/闭包/路径碰撞/只读 Library | deterministic output files；不写原始 Source |
| G6 Native/WASM | 同样输入负载与重放 | 同一诊断、同一 outputFiles 内容/顺序 |
| G7 Browser | Worker + wasm_exec.js + skillc.wasm，静态服务 Chromium | 页面运行真结果，不依赖 DB/Node server API |
| G8 真实迁移 | 原样快照（预期移植诊断）+ 新版迁移工程 | 分开报告；不能因缺 Host Schema 造假 PASS |
| G9 交接 | README/AGENTS/Studio API/WORK/提交/PR | 状态真实，PR reviewable，不未经验证合 main |

## 3. 必须固定的回归用例

1. 单 Skill、双 Skill、根 Skill 的传递依赖、缺依赖、跨 Skill Procedure。
2. 同名 Operation 由两份库重复声明 → ERROR，不随机挑一份。
3. 上游 Tool output 的 `value: string` 通过 `with.value.from` 绑定 Script，新 Output Schema 只有 Script 自身的结果。
4. `context.facts.unknown` 没有 `spec.context.schemas.facts` 或指向未导入 Host 文件 → `EXTERNAL_SCHEMA_MISSING`，不允许 PASS；`requiredFacts` 不能代替 Schema。
5. Tool Schema `type: object` 但没有 properties，`someTool.output.missingField` → `SOURCE_FIELD_UNKNOWN`。
6. `preconditions.<step>.output` 和 `<node>.preconditions.<step>.output` 区分本地/远端 node 作用域。
7. 旧 Script `# /// connectionagent-tool`、`main(input: OldDict) -> OldDict` 没有明确 Output 类型 → 新 v0 必须拒绝，而非运行兼容解析器。
8. 新 Script Output `Any` / `dict` / 空 TypedDict / 未声明嵌套字段 → 必须拒绝；未知 Annotated 约束不得忽略。
9. 确定性：打乱 Request.Files 输入顺序、相同内容多次构建、Native 与 WASM → 一致。
10. 安全：`..`、绝对路径、symlink、YAML alias 爆炸、重复键、远端 `$ref`、非法输出路径、超限文件 → 拒绝不崩溃。
11. MAS 提案是否 READY：**永不由 Compiler 宣称**；用于静态链接需保留 provenance 与 PROPOSED 标签。
12. 所有本次测试必须记录实际命令、Exit Code、失败诊断码/源路径和是否为外部 Owner 阻塞。

## 4. 输出报告模板

~~~text
Commit/branch/PR:
G0: PASS | FAIL | BLOCKED | NOT_RUN + evidence
...
G9: PASS | FAIL | BLOCKED | NOT_RUN + evidence
Native CLI: ...
WASM + Browser: ...
Synthetic 2-Skill build: ...
Script Contract build: ...
Pinned old PDU Skill migration diagnostics: ...
Pinned old MAS Skill migration diagnostics: ...
Migrated v0 Project (if created): ...
Missing Owner contracts: ...
Other code blockers: ...
Next concrete change:
~~~

*不能把 MALFORMED/UNSUPPORTED 的旧快照报为编译器回归；也不能把编译报错美化成 Host Runtime READY。*
