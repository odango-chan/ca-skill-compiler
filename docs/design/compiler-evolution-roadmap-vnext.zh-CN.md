# CA Skill Compiler 下一阶段演进路线

> **状态：三项架构原则已批准，契约与路线待评审，尚未实现。**
>
> 现行基线为 CA-SL v1alpha5 和 Compiler v2.0.2。v2.1.x 是产品规划候选；API、Artifact 与各发布包独立定版。契约见[工具链架构](ca-toolchain-architecture-vnext.zh-CN.md)，MAS G0–G5 验收见 [ca-agent-demo #217](https://github.com/odango-chan/ca-agent-demo/pull/217)。

## 1. 交付顺序与退出条件

| 阶段 | 责任与交付 | 依赖与退出标准 |
|---|---|---|
| V0 — 契约评审 | Compiler、Skills、Host 与 Provider Owner 核对接口快照、锁、两类 Requirement、独立发布、授权与准入 | 对照冻结规范逐项标记静态检查和外部要求；批准新入口、正反样例和生产拒绝规则后定版 |
| V1 — 接口发布 | ca-client-tools / Owner 发布 MAS Interface Package；ca-context 维护方提供 Subject Context；相关 Owner 提供实际引用的 Ingress 导出 | V0；以 13 Operation、25 候选 Schema 为输入，退出以实际可达闭包、精确 Effect、稳定 ID 和不可变快照为准；对应 G0 |
| V2 — 锁定与分发 | Studio / CI / 调用方实现外部 Resolver、签名信任检查、锁生成和快照准备 | V0 可并行，V1 真实样例用于验收；锁可离线重放，显式报告约束冲突与传递依赖缺失 |
| V3 — 开发构建 | Compiler 实现显式版本化 Development Build / Verify；按 Host Admission 与 Runtime Execution 两类记录 Requirement | V1 + V2；接口完整时无 Evidence、Grant、Provider 也能生成并 Verify；Native / WASM / Chromium Worker 字节一致；对应 G1 |
| V4 — Host Admission | ca-agent-demo 增加显式新产物入口，复用现有 Procedure Runtime；ca-client-tools 独立发布 Provider | 可与 V3 并行设计；G2 验收要求真实 V3 产物、同摘要闭包重验、精确 Provider Binding、测试授权、拒绝与恢复用例；G3 是独立运行时门槛 |
| V5 — MAS 集成 | ca-agent-demo + ca-client-tools + env-openamsm 验收 Provider、Mock AMF，再验证真实 AMF / UERANSIM | G4 依赖 G0–G3 和独立 Provider；G5 依赖 G4、持久化 Owner continuation、真实协议关联、可信 Evidence 及目标授权；各阶段分别记录实际证据 |

V0–V5 是工具链交付顺序，G0–G5 是 MAS 验收门槛。G3 可独立于 G1 推进，最终联调仍需 G2 和 G3 同时满足。Mock AMF 经本地 NRF/SCP 验证完整 SBI 路径；直接调用 handler 单独记录。Mock AMF 不解决复合 SSM 的 30 秒限制、恢复句柄或 Owner Commit Proof。G5 的现场结论与生产 qualification 分别评估。

首轮 Compiler 实现及验收见 [DeepSeek Harness 任务](compiler-development-build-harness.zh-CN.md)。试验分支按[实施契约](compiler-development-build-pilot.zh-CN.md)和预生成样例直接开工。公开产物契约在 V0 定版后对外使用。正式 MAS G1 需要通过 G0 的接口输入。

## 2. 兼容和迁移

1. **保留现行入口。** 固定 v2.0.2 基线及冻结 Build / Verify、可信依赖和 fixture 流程。为现有 Native / WASM / Studio 调用方保留原契约。以发布时正反样例对比结果、诊断及产物摘要，检查旧入口行为。
2. **从发布源重建接口快照。** Owner 从现有 ToolCatalog、Schema、CapabilityContracts 和 Ingress 材料提取接口内容，确认 ID、Effect、成员关系及复用的协议版本。环境 Scope 由部署方使用独立 Grant 管理，保证由相应权威重新绑定并签发。原 v1alpha5 输入保持其既有含义；发布过程不自动转换授权或推断 Owner。
3. **显式试用。** Studio / CI 并行生成新锁和开发产物，先做离线验证；调用方显式选择新 BuildRequest，分发对应 BuildResult / Artifact 和锁定快照。Host 按入口及确切版本拒绝混用和未知契约。冻结 fixture 继续用于原测试目的。
4. **隔离部署。** G2 验收后以 Host 环境级 opt-in 接收开发产物，把测试信任根、Grant 与 Provider 绑定限定在目标环境。运行中变更策略或撤销 Grant 时阻断新派发，保留在途结算；继续保证生产入口拒绝开发产物。
5. **回滚与后续生产契约。** 关闭开发接入口，停止新开发 Invocation，保留在途状态与审计并按原绑定完成授权结算。既有部署继续使用已验证的冻结产物与绑定。回滚 Provider 时按旧摘要重新验兼容和准入。生产如需采用新格式，另评审契约后重新构建和签发对应保证；禁止就地修改开发 Artifact 或仅凭补齐外部材料提升其 profile。

## 3. 验收矩阵

| 检查 | 预期证据 |
|---|---|
| 接口闭合与确定性 | 无 Evidence、Grant、Provider 的完整接口可以生成 `LINKED / BUILT / NOT_ADMITTED`；重排输入、跨 Native / WASM / Chromium Worker 重建一致；离线 Verify 能检查精确快照、两个 Requirement 文件与全部产物字节 |
| 依赖与类型负例 | 缺传递包或 Schema 片段、版本冲突、同版异摘要、未锁引用、重复符号、缺失 Capability 成员、未知 Effect、Wait / Ingress / 响应绑定缺失、Ingress 导出身份不成立（id 不存在/重复、path 漂移、未导出 operation、binding 错误），均失败且无产物 |
| Requirement 两类分离 | 仅缺外部授权时 Build 成功并写入 `host-admission-requirements.json`（`HOST_VERIFICATION_REQUIRED`）；`COMMIT_PROOF` / `RECONCILE` 只出现在 `runtime-execution-requirements.json`（`RUNTIME_EXECUTION_REQUIRED`），不出现在 Host Admission 文件；二者都不表示执行授权 |
| 独立性多组 | 多组、多根、多域对各生成独立 `EVIDENCE_INDEPENDENCE_TRUST`，`selector` 精确到 `/predicates/<predicate>/evidence/independence/<index>`；不使用 `env.roots[0]` 或全局布尔值；同一 Descriptor 同时声明 sourceIdentity/factIdentity 时两条记录互不覆盖 |
| 篡改与重封 | 删除一条 Requirement 后重新封装外层摘要仍被拒绝；编辑 Procedure、Lock、Descriptor 或 Requirement 均导致 Verify 失败 |
| Provider 独立发版 | Interface 先发布即可构建；同一接口摘要的第二个 Provider 需单独证明兼容并重新 Admission；只有范围匹配、协议不合或映射歧义均拒绝 |
| 授权与恢复 | 无 Grant、错误环境或主体、错 Scope、已撤销、生产接收开发产物、未知 API / Artifact 均失败；恢复复核身份与权限；`UNKNOWN` 保留原尝试，不盲重试 |
| 基线与迁移 | v2.0.2 冻结输入、输出与诊断基线不变；已发布产物和授权继续按原入口验收；回滚能封住新派发并保留未结算 Invocation |
| MAS 跨仓库 | G0 记录实际闭包，G1 记录开发产物，G2/G3 证明拒绝与可恢复运行，G4 验证真实 Provider / Mock AMF 路径，G5 记录真实 AMF / UERANSIM 和用户面证据 |

以上是后续交付门槛。当前评审验证文档和契约提案；代码验收、G0–G5 实跑与生产资格均待交付，MAS G0 / G1 仍为 **NOT_RUN**。
