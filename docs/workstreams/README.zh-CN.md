# 历史工作包（归档入口）

[English](README.md) · [当前文档](../README.zh-CN.md)

**现行产品是 Compiler v2.1.0，唯一语言目标为 CA-SL v1alpha5。** 本目录**不是当前任务队列**。之前的 WP0–WP12 和 v1alpha5 实施工作包均为当时的阶段记录；里面的「OPEN」「DRAFT」「执行 Goal」指令不代表今天还需要照做。

32 份非必需的旧工作包 Markdown 已移出活动代码树，原始内容和相关 PR 仍可从[固定历史提交的工作包目录](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs/workstreams)逐份查看。v1alpha4 本地验收及早期设计见[完整历史快照](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs)。

当前开发只需参考[当前产品状态](../WORK.md)、[v2.1.0 发布与测试记录](../releases/ca-skill-compiler-v2.1.0.md)和[Compiler 独立交付边界](../COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md)。真实 MAS Create/Modify/Release 作者源码离线静态回归仍单独跟踪于 [#53](https://github.com/odango-chan/ca-skill-compiler/issues/53)，不是 Compiler 发版阻塞。

测试可能使用的**机器可读历史验收清单 JSON** 保留原路径；它们是历史证据，不构成今天的执行指令。没有修改语言语义、代码或接口兼容行为。

测试仍读取的 [WP7 v1alpha5 符合性矩阵](wp7-conformance-v1alpha5.md) 保留在 `main`，因为 `TestV1Alpha5ConformanceMarkdownSync` 需要与机器可读清单比较。它是测试证据，**不是当前开发任务**。
