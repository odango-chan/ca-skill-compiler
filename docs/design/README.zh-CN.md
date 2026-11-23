# 编译器设计与契约索引

[English](README.md) · [文档导航](../README.zh-CN.md)

**现行产品：v2.1.0（正式发布）。** [CA-SL v1alpha5 冻结规范](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md)由 ca-skills 负责。架构描述不能覆盖冻结的语言或 Compiler 机器契约。

## 已发布机器契约

- [Compiler 冻结 v1alpha5](../../contracts/v1alpha5/README.md)：BuildRequest / BuildResult / DependencyInspection / Artifact。旧 Artifact 内嵌 generator 版本仍为 `v2.0.2`，用于字节一致性。
- [Development v1alpha1](../../contracts/development/v1alpha1/README.md)：随 v2.1.0 发布，显式 Interface Package、Dependency Lock、Build/Verify；Artifact 始终为 `NOT_ADMITTED`。

## 现行架构与实施

- [单一 v1alpha5 Compiler Core](compiler-core-v1alpha5.md)
- [冻结 Artifact 设计](compiler-artifact-v1alpha5.md)
- [工具链架构：Compile / Link / Build / Host Admission](ca-toolchain-architecture-vnext.zh-CN.md)
- [Development Build 实施契约](compiler-development-build-pilot.zh-CN.md)
- [Development 最终实施评审（历史测试证据）](compiler-development-build-implementation-report.zh-CN.md)
- [Evidence / Assurance 职责审计](compiler-evidence-assurance-responsibility-audit.zh-CN.md)
- [架构演进与外部责任边界](compiler-evolution-roadmap-vnext.zh-CN.md)
- [Archify 架构图](../architecture/archify/README.md) · [5GC Profile 历史评审](../profiles/5gc-smf-procedure-profile.md)

## 历史规范（只读）

名称含 `v1alpha4`、`v0`、`v0.1` 的设计文件保留用于追溯，不是当前 Compiler 支持的 BuildRequest/语言目标。v1alpha4 语言**语义**由唯一的 v1alpha5 Pipeline 继承；但旧版本 API 输入仍应直接拒绝。

- [v1alpha4 编译器契约](compiler-contract-v1alpha4.md) · [BuildRequest](build-request-v1alpha4.md) · [Output](compiler-output-contract-v1alpha4.md)
- [Artifact](compiler-artifact-v1alpha4.md) · [快照摘要](dependency-snapshot-digest-v1alpha4.md)
- [Owner](owner-contracts-v1alpha4.md) · [Procedure](procedure-link-v1alpha4.md) · [Script](script-contract-v1alpha4.md)
- [v0 架构简报](skill-compiler-v0.md) · [v0.1 契约](compiler-contract-v0.1.md)

产品发布结论见 [v2.1.0 发布记录](../releases/ca-skill-compiler-v2.1.0.md)，旧实施工作包见[历史任务索引](../workstreams/README.zh-CN.md)。
