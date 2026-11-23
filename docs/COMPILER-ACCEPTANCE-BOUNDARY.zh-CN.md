> **现行产品结论（2026-11-23）：Compiler v2.1.0 与 Development v1alpha1 已正式发布。** [发布 SHA / 八项资产 / 全量验收](releases/ca-skill-compiler-v2.1.0.md)。下文原有 2026-11-20 实现与候选状态只作为审计记录；Development Artifact 仍为 `NOT_ADMITTED`，真实 Owner/Provider/Host/Runtime/5GC 资格仍由外部系统评估。

# CA Skill Compiler — 独立交付与验收边界

> 状态：2026-11-20。本文区分 **Compiler 自身是否正确、源码是否进入 main、产品是否发布、跨仓是否集成、运行时是否获得执行资格**。五种结论不能相互替代。

## 1. 唯一职责

Compiler 的输入是 CA-SL 源码、调用方提供的准确接口声明与 Schema、不可变依赖快照、Dependency Lock 及适用的**静态构建上下文**；输出是诊断、链接结果、确定性 Artifact 和离线 Verify 结果。Compiler 不接管 Registry、Owner 发布、Provider 部署、Host Admission、Runtime 或实际 5GC/NF 操作。

| 事项 | Compiler 自身必须做到 | 不在 Compiler 交付范围 |
|---|---|---|
| CA-SL 与 Procedure | 按冻结语言对源码做全部 Compiler-owned 静态规则检查，异常 fail-closed | 实际执行 Procedure / Nsmf 或 Namf 消息 |
| Interface Package、Tool / Capability / Evidence | 验证精确导出 ID、Schema、Effect、静态 Owner 声明、包摘要、依赖边、Capability 选择、EvidenceDescriptor 和类型闭包 | 证明 Publisher 在真实世界中可信、代签 Owner 信任根、发布接口包 |
| 构建 | 用提供的精确快照与 Lock 生成可复验 Artifact；无源或静态接口缺口就 Build FAIL | 访问网络解析依赖、包管理、安装 Provider 或发放部署权限 |
| Development Artifact | 记录静态 `capabilityRequirements[]`、Host Admission/Runtime Execution 的待满足要求，并保证 `NOT_ADMITTED` | 把待满足要求当成已兑现的 Grant、Attestation、真实 Commit Proof 或 Reconcile |
| 验证 | 校验输入、产物的确定性、哈希、链接和字节级一致性；Native/Go-WASM/真实 Chromium parity | 在 Host 验证实际 Grant、Provider Binding、Invocation 权限，或给出 Qualification |

**正式 Owner/Provider 材料、Production Admission 和 Qualification 都不是 Compiler 完成、合并或独立发版的前置门槛。** 对外部材料缺失应在集成/准入责任域如实记录，但不得把 `NOT_QUALIFIED` 冒充 Compiler 测试失败。

## 2. 分离的状态与门槛

| 轨道 | 定义 | 能否阻塞 Compiler 自身发布 |
|---|---|---|
| **Compiler Core** | 规范静态义务、源/接口校验、确定性 Build/Verify、失败无产物、Native/CLI/WASM/Chromium 一致性、回归和可复验交付物 | **能**。这是 Compiler 的质量门禁 |
| **Compiler Product Release** | 代码验收、准确版本、Release Notes、可复现二进制/WASM、校验和与正式 Tag | **能**。由 Compiler 仓独立决定 |
| **MAS Source Offline** | 真实 Create / Modify / Release Skill 源配合完整但可标注为测试用途的接口快照，静态 Build + Verify | **不是通用发布前置门槛**；作为独立跨仓集成回归 |
| **MAS 正式接口 G0/G1** | 按 MAS 跨仓定义使用其正式来源和接口包验证的联调门槛 | **不能**。只能阻塞相应 MAS 集成里程碑 |
| **Host / Runtime / Production** | 发布者信任、Provider 实现、Grant、Invocation、真实 Evidence、Commit Proof / Reconcile、现场 E2E 和资格评估 | **不能**。归外部系统自己评估 |

### 当前确定的状态

- **已发布 Compiler**：v2.1.0，目标 `connectionagent.io/v1alpha5`；既有冻结入口与 Artifact 不因 Development Build 合并而改变。
- **Development Build 已正式发布**：架构 [#48](https://github.com/odango-chan/ca-skill-compiler/pull/48)、实现 [#51](https://github.com/odango-chan/ca-skill-compiler/pull/51) 与一致性修复 [#56](https://github.com/odango-chan/ca-skill-compiler/pull/56) 已合并；Development v1alpha1 随 [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0) 发布并冻结，固定 SHA 和完整发布验收见[记录](releases/ca-skill-compiler-v2.1.0.md)。
- **Development 执行状态**：`NOT_ADMITTED` 是预期契约输出，不是编译失败。
- **MAS 真实源离线测试**：[Issue #53](https://github.com/odango-chan/ca-skill-compiler/issues/53) 独立跟踪；没有实跑记录就标 `NOT_RUN`，不能由合成 Compiler fixture 的 PASS 替代。
- **正式 MAS G0/G1**：在本次收敛前记录为 `NOT_RUN`；即使 Issue #53 的 synthetic test 通过，也不能直接改写正式 G0/G1 结论。
- **外部 Owner / Provider / Host / Qualification**：由对应发布方、部署方和执行系统负责，不属于 Compiler backlog 的阻塞事项。

## 3. 真实 MAS 源验证的边界

[Issue #53](https://github.com/odango-chan/ca-skill-compiler/issues/53) 固定来自 `ca-skills/examples/skills/` 的三套 v1alpha5 MAS 源码，以 `ca-client-tools` 的源对齐接口候选构造**准确、完整且明确标记测试用途**的 Interface Package snapshots + Lock，再运行 Development Build / Verify。

该测试的目的是发现真实源码与静态接口链接上的缺口，而不是强制 Owner 先正式发布或 Host 先能执行。测试中不得为凑 PASS 伪造 Callable/Schema/EvidenceDescriptor 的缺失静态语义；缺口必须精确上报源码、包、Alias、Export ID 与责任仓库。没有正式 Owner/Provider 证据不会令 Compiler 静态检查自动失败；缺少**必需的静态接口声明**则仍应 fail-closed。

特别注意：`ca-skills/scripts/compile_mas_v1alpha5.py` 当前使用**已发布 v2.0.2 旧 BuildRequest / trusted.json 路径**；它的既有结果不能未经适配就标为新 Development Build 的结果。

## 4. 禁止的状态混用

1. 不得因为 Owner 未正式发布而把 `COMPILER_CORE_PASS` 记为 FAIL。
2. 不得因为 Compiler Build 成功而把 `HOST_ADMITTED` 或 `PRODUCTION_QUALIFIED` 记为 PASS。
3. 不得因为 `DEVELOPMENT/NOT_ADMITTED` 而把 Compiler 的 Build/Verify 成功记为失败。
4. 不得把源对齐候选 Package 的静态离线测试写成 MAS 正式 G0/G1 或实际 5GC E2E。
5. 不得因为公开源码已合并就称该功能包含于 v2.0.2 的已发布二进制；新入口仍需要单独产品发版决策。

相关历史：[v1alpha5 既有发布跟踪 #41](https://github.com/odango-chan/ca-skill-compiler/issues/41)、[v1alpha4 旧事项 #10](https://github.com/odango-chan/ca-skill-compiler/issues/10)、[Development 实施报告](design/compiler-development-build-implementation-report.zh-CN.md)。
