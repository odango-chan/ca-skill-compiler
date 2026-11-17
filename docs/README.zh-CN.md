# 文档索引

[English](README.md)

本目录包含 CA Skill Compiler 的技术参考文档和维护记录。

## 使用说明

- [CLI 使用说明](cli-v1alpha4.md) — 命令、输入格式、参数、输出和退出码。
- [已知限制](known-limitations-v1alpha4.md) — 当前明确不支持或仍受限制的功能。
- [v1.0.0 发布说明](release-notes-v1.0.0.md) — 本版本的功能和验收记录。
- [示例与测试样例](../examples/README.zh-CN.md) — 当前示例和历史参考项目。

## 语言规范与编译器契约

- [CA-SL v1alpha4 语言规范](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md) — 语言规范的唯一权威版本。
- [Compiler Contract](design/compiler-contract-v1alpha4.md) — 编译器职责和行为边界。
- [Compiler Pipeline](design/compiler-pipeline-v1alpha4.md) — 校验阶段及实现映射。
- [BuildRequest](design/build-request-v1alpha4.md) — 编译器输入契约。
- [Dependency Snapshot Digest](design/dependency-snapshot-digest-v1alpha4.md) — 依赖快照摘要算法及跨语言测试向量。
- [Compiler Output API](design/compiler-output-contract-v1alpha4.md) — Check、BuildResult 和 DependencyInspection。
- [Owner Contract Linking](design/owner-contracts-v1alpha4.md) — 依赖所有者和 Capability 链接。
- [Procedure Linker](design/procedure-link-v1alpha4.md) — Procedure Graph 和静态分析。
- [Script Contract](design/script-contract-v1alpha4.md) — Python Script 接口和 Schema 派生规则。
- [Compiler Artifact](design/compiler-artifact-v1alpha4.md) — 制品目录结构和 Manifest。
- [Skill Studio 集成](studio-integration-v1alpha4.md) — 浏览器/WASM 集成边界。
- [机器可读契约](../contracts/v1alpha4/README.md) — JSON Schema 和测试向量。

## 开发与维护

以下文档面向贡献者和维护者，日常使用 CLI 时通常不需要阅读。

- [本地验证方法](local-verification-v1alpha4.md) — 可复现的测试和构建命令。
- [Compiler 验收记录](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) — 实现证据和符合性覆盖情况。
- [维护状态](WORK.md) — 当前仓库维护记录。
- [开发历史](WORK-history.md) — 历史开发记录。
- [Workstream 记录](workstreams/) — 实施和符合性规划。
- [依赖选型](design/oss-dependency-selection-v1alpha4.md) — 第三方库及选型依据。

项目介绍和快速开始见[中文 README](../README.zh-CN.md)。详细技术记录集中在本索引中，不放入项目首页。
