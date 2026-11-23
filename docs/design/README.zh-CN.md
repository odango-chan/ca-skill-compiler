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

## 历史设计资料

14 份 v0/v0.1/v1alpha4 的旧设计正文已从活动目录移出，避免被误读为当前 Compiler 输入契约。全部原文仍可在[固定历史提交的设计文档目录](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs/design)查看。冻结的**机器**契约（`/contracts/`）仍在 main，当前 v1alpha5 与 Development 设计文档也未删除。

现行发布事实以 [v2.1.0 发布记录](../releases/ca-skill-compiler-v2.1.0.md)为准；旧任务包见[历史工作包索引](../workstreams/README.zh-CN.md)。
