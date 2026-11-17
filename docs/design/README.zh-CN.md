[English](README.md)

# 编译器设计文档

本目录包含 Compiler 自有契约、实现映射和历史架构说明。它不替代 CA-SL 语言规范。

## 权威来源与状态

文档的权威关系如下：

1. **CA-SL 语言语义：** 冻结的 [CA-SL v1alpha4 规范](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md)，固定在提交 `7f84393ee8c395e4d8c39a74cb5000a1f896588d`。它是语言语义的唯一权威来源。
2. **Compiler 行为：** [Compiler Contract v1alpha4](compiler-contract-v1alpha4.md)，从属于语言规范，定义 Compiler 必须校验的内容及其职责边界。
3. **Compiler 自有接口和制品：** 下文列出的 BuildRequest、Output API、Dependency Snapshot Digest 和 Artifact 契约。
4. **专项链接契约：** 下文列出的 Owner、Procedure 和 Script 契约。
5. **实现参考：** Pipeline 文档只描述实现代码和测试的位置，不修改契约。OSS ADR 记录第三方库选型，不定义 CA-SL 语义。

实现状态和产品符合性以[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)及[已知限制](../known-limitations-v1alpha4.md)为准。`LOCAL_ACCEPTANCE_PASS` 表示记录中的本地验证命令已执行通过，不代表所有 E-case 均达到完整案例 PASS，也不代表真实 Owner/Deployment 或 5GC 产品资格已经完成。产品资格仍为 **NOT_QUALIFIED**。

## 领域与流程 Profile

- [5GC SMF Procedure Profile](../profiles/5gc-smf-procedure-profile.md) — 从 SMF 视角约束 3GPP → Tool → Skill 映射、Node/Observation 边界、v1alpha4 异步语义和 PDU Session 参考流程。

## 当前 v1alpha4 契约

- [Compiler Contract](compiler-contract-v1alpha4.md) — Compiler 职责、编译流水线、静态校验义务和信任边界。
- [BuildRequest](build-request-v1alpha4.md) — 版本化输入封套、Skill 源文件、不可变依赖快照和可信构建上下文。
- [Compiler Output API](compiler-output-contract-v1alpha4.md) — Check、BuildResult、DependencyInspection、诊断和确定性排序。
- [Dependency Snapshot Digest](dependency-snapshot-digest-v1alpha4.md) — Compiler 与 Studio 共用的字节级规范化摘要算法。
- [Owner Contract Linking](owner-contracts-v1alpha4.md) — ToolCatalog、CapabilityContracts、Evidence 和 Mutation/Reconcile 链接边界。
- [Procedure Linker](procedure-link-v1alpha4.md) — Procedure Graph、阶段、路由、参数、Schema 和 Evidence 静态链接。
- [Script Contract](script-contract-v1alpha4.md) — Skill 自有 Python Script 角色、派生输入/输出类型和静态分析边界。
- [Compiler Artifact](compiler-artifact-v1alpha4.md) — 确定性制品目录、Manifest 和生成 Schema 的序列化。
- [Compiler Pipeline 映射](compiler-pipeline-v1alpha4.md) — 实现文件和测试位置索引；仅作描述，不具有规范效力。
- [开源依赖选型](oss-dependency-selection-v1alpha4.md) — 固定的第三方库、适配层和实现约束。

## 历史设计记录

以下文件保留用于追溯。它们描述的目录布局、API 形状、源语言版本、ToolCatalog 假设和验收计划均已被取代，不得作为 Compiler v1.0.0 或 CA-SL v1alpha4 的当前要求。

- [Skill Compiler v0 架构说明](skill-compiler-v0.md)
- [Compiler Contract v0.1](compiler-contract-v0.1.md)
- [Procedure Link Contract v0](procedure-link-v0.md)
- [Script Contract v0](script-contract-v0.md)

保留历史文档是为了说明早期设计决策，不代表继续支持旧版语法。每份归档文档均提供了指向当前契约的说明。
