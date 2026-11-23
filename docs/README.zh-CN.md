# CA Skill Compiler — 文档导航

[English](README.md) · [项目主页](../README.zh-CN.md) · [v2.1.0 正式 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0)

**现行基线：** Compiler v2.1.0，唯一支持 CA-SL `connectionagent.io/v1alpha5`。已发布的 Compiler、真实跨仓联调和生产准入是不同状态。

## 选择入口

| 想做什么 | 推荐文档 |
|---|---|
| 安装、编译、CLI 参数 | [中文 CLI](cli.zh-CN.md) · [English CLI](cli.md) |
| 查看当前状态和后续集成 | [当前状态](WORK.md) |
| 查正式发布与验收证据 | [v2.1.0 发布记录](releases/ca-skill-compiler-v2.1.0.md) · [Actions](https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980) |
| 了解 Compiler 与 Host 的职责 | [独立交付与验收边界](COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md) |
| 查实现限制 | [现行限制](known-limitations-v1alpha5.md) |
| 接入机器 API | [冻结 v1alpha5 契约](../contracts/v1alpha5/README.md) · [Development v1alpha1 契约](../contracts/development/v1alpha1/README.md) |
| 查架构/流程图 | [设计文档索引](design/README.zh-CN.md) · [Archify 图形索引](architecture/archify/README.md) |
| 找测试正例 | [示例索引](../examples/README.zh-CN.md) |
| 查历史工作包与旧版本 | [历史工作索引](workstreams/README.zh-CN.md) · [旧工作日志](https://github.com/odango-chan/ca-skill-compiler/blob/72b41548a9c4d29c6c49abed117a04a963454f22/docs/WORK-history.md) |

## 权威顺序

1. [ca-skills 正式冻结的 CA-SL v1alpha5 规范](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md)及 IngressCatalog v1alpha1。
2. Compiler 已发布的 [v1alpha5](../contracts/v1alpha5/README.md) 与 [Development v1alpha1](../contracts/development/v1alpha1/README.md) 机器契约。
3. [现行工具链架构](design/ca-toolchain-architecture-vnext.zh-CN.md)、Compiler Core、实施报告和按版本固定的验收。
4. v0/v0.1/v1alpha4 历史材料；即使包含曾经冻结的合同，也**不表示当前 Compiler 支持旧版本输入**。

Development Build 是 v2.1.0 已发布的产品入口，但 Artifact 必须为 `NOT_ADMITTED`。任何文档均不能替代 Owner 信任、Deployment Grant、Host Admission 或运行时 Qualification。
