# 下一代 CA Skill Compiler 开发计划

**状态：** DRAFT — 架构与工作包准备已开始；正式语言实现尚未授权。  
**基线：** CA Skill Compiler v1.0.0，当前唯一正式语言目标为冻结的 CA-SL v1alpha4。  
**下一语言版本：** 未分配。本文不推断或创建版本号。

## 1. 目标

在现有 Go Core 上增量实现下一版 CA-SL 的静态编译能力，重点支持：

- Node-entry Event Wait；
- package-scoped Integration Ingress Operation / IngressCatalog；
- 精确的 triggering Operation → `armInputSchema` 绑定；
- `wait.output` 类型解析与路径可用性；
- Wait gate 的 Procedure graph / dominance / reachability 分析；
- `wait.response.after` 的 Response Producer 绑定与 Schema 映射；
- Native / WASM / Chromium Worker 一致性。

这不是 Compiler 重写。v1.0.0 是已发布基线，必须保持不可变。

## 2. 当前契约依据与冻结状态

| 契约 | 仓库 / PR | 当前状态 | 对 Compiler 的影响 |
|---|---|---|---|
| Node-entry Wait 语义 | ca-skills #60 | 已合并 | 语义基线 |
| EVENT_ACCEPTED tombstone / stale-message 隔离 | ca-skills #64 | 已合并 | 运行时语义约束；Compiler 必须链接 correlation policy |
| Wait 规范性修订 | ca-skills #61 | Draft，版本身份未分配 | 语法、诊断、SourcePath、图语义尚未冻结 |
| Integration Ingress Operation | ca-skills #62 | Draft | IngressCatalog、Schema Closure Digest、归一化与 response mapping 尚未冻结 |
| Generic Host / Runtime continuation | ca-agent-demo #210 | Draft，设计文档 | Compiler 只消费语言和 Ingress 契约，不实现 Host/Runtime |
| 5GC SBI continuation profile | ca-agent-demo #209 | Draft，设计文档 | 3GPP 示例，不是 Compiler 的 5GC 专用语义 |

**重要边界：**

- v1alpha4 不接受作者级 `wait`、`wait.output` 或 `ingressOperation`。现有 `UNSUPPORTED_WAIT` 等冻结诊断必须继续保持。
- #61 已明确禁止将 Draft 当成 v1alpha4，也禁止在没有语言 Owner 裁决时推断下一语言版本号。
- #62 的 IngressCatalog 是独立 package-scoped export，不是 ToolCatalog Operation、Observation 或 Capability。
- #62 的 Schema Closure Digest v1 目前仍是提案。其 golden vector / CI 通过只证明算法样例，不等于 Owner 已发布真实 Schema。
- 真实 Owner / Deployment 发布与实时 5GC qualification 不阻止 Compiler 独立实现；实现阶段可使用明确标记的 synthetic trusted fixtures。不得将 synthetic fixtures 报告为真实互操作通过。

## 3. 必须先做的 API 版本盘点

v1alpha4 的机器契约使用封闭字段/枚举，不能在原版本上静默添加 Ingress export kind。

| 现有契约 | 已发现的封闭点 | 下一版需要的决策 |
|---|---|---|
| BuildRequest v1alpha4 | `TrustedDependencyBinding.exportOwners[].exportKind` 只允许 operation / observation / schema / evidenceGuarantee；languageTarget 固定 v1alpha4 | 新 API contract 增加独立 `ingressOperation` export kind，并绑定唯一新 languageTarget |
| DependencyInspection v1alpha4 | `callables[]` 仅 operation / observation，另有 schemas[]；没有 package-level Ingress inventory | 决定新增 `ingressOperations[]`，或改为可扩展的 typed export inventory；不能把 Ingress 伪装成 callable |
| Artifact Manifest v1alpha4 | `languageTarget` 固定 v1alpha4；dependencyClosure.exports 的 exportKind 为封闭枚举 | 新 Artifact manifest contract 必须记录新 languageTarget 和 `ingressOperation` dependency export |
| BuildResult v1alpha4 | 输出 envelope 与 Artifact manifest 分离 | 先比较结构；只有输出 envelope 变化时才升级 BuildResult API，不为版本整齐而无理由重复升级 |
| Diagnostic / Conformance | Code、Phase、E01–E120 / Appendix H 有冻结身份 | 新语言的诊断必须由语言 Owner 发布；Compiler 不自行分配语义身份 |

BuildRequest、DependencyInspection、Artifact Manifest 的版本号彼此独立于 CA-SL 语言版本和 Compiler 产品版本。**不得假定它们必须同号，也不得在版本身份未裁决前写入猜测值。**

## 4. 目标编译架构

保留现有流水线：

`BuildRequest → Source Admission → Parse/AST → Dependency Linking → Schema/CEL → Procedure Graph → Script ABI → Artifact`

增加 target-specific 分支，而不是把新语法塞入 v1alpha4：

1. **Language Target Selection**：在解析前精确选择一个语言目标；未知目标 fail closed；绝不 fallback 到 v1alpha4。
2. **IngressCatalog Index**：解析可信 dependency snapshot 中的 package-scoped IngressCatalog，验证精确 apiVersion、exportKind、Package ID、Owner、artifact/schema identity。
3. **Wait AST**：只在新语言目标识别 Node.entry `wait`；v1alpha4 分支继续拒绝。
4. **Ingress Linker**：按 source/service/operation 精确解析 Ingress Operation；不按名称模糊匹配。
5. **Trigger Binding Linker**：将每条 Wait 入边绑定到精确 ToolCatalog Operation callable，并从 Operation arguments / 已准入 Invocation Context 解析 typed `armInputSchema`。
6. **Wait Output Resolver**：把 `NodeId.wait.output` 绑定到 Ingress normalized request Schema；验证 JSON Pointer、类型赋值与路径可用性。
7. **Procedure Graph Extension**：把 Wait 建模为 Node-entry control vertex，加入 EVENT_RECEIVED / TIMEOUT 两条控制边；参与 DAG、reachability、dominance、route obligation 和 SourcePath availability。
8. **Response Boundary Linker**：由 `wait.response.after` 和 Ingress 声明的 Producer stage type（operation / observation）推导唯一 Producer SourcePath；验证其输出 Schema、responseResultSchema 和 JSON/multipart mapping。
9. **Artifact Assembly**：写入精确的新 languageTarget、dependency closure 和新 Artifact manifest；不把外部 IngressCatalog 内容伪装成 Skill 源文件。

## 5. Work Packages

| WP | 工作 | 主要产物 | 开始条件 |
|---|---|---|---|
| WP0 | 语言与 API 冻结盘点 | 完整单文件规范、版本身份、BuildRequest / DependencyInspection / Artifact contract 决策、诊断清单 | #61 / #62 完成独立审查 |
| WP1 | IngressCatalog admission / linking | export parser、trusted Owner binding、Schema Closure Digest v1、exact reference closure | WP0 |
| WP2 | Wait syntax / AST | Node-entry Wait grammar、duration parser、strict unknown-field diagnostics | WP0 |
| WP3 | Trigger binding | exact Operation identity、armInputSchema binding、start-context binding | WP1 + WP2 |
| WP4 | wait.output dataflow | normalized Schema resolution、JSON Pointer、assignability、edge/path-sensitive availability | WP1 + WP2 |
| WP5 | Wait graph | entry-control vertex、EVENT_RECEIVED/TIMEOUT、DAG/dominance/reachability/route obligations | WP2–WP4 |
| WP6 | Response boundary | response.after、Producer-stage derivation、responseResultSchema、JSON/multipart/binary mapping | WP1–WP5 |
| WP7 | Diagnostics / conformance | exact Code/Phase、negative/positive static fixtures、case-to-test map | WP0–WP6 |
| WP8 | Native / WASM / Worker parity | Go tests、vet、CGO-free Native、WASM、real Chromium parity、artifact byte parity | WP1–WP7 |
| WP9 | Studio / LSP integration | DependencyInspection display、Wait/Ingress diagnostics、completion/schema navigation | API contracts + WP8 |

WP7 必须把 Compiler-owned 静态用例与 Host/Runtime/Integration-owned 动态用例分开。#61 的 EW-001–EW-036 与 #62 的 IN-001–IN-054 是跨组件 conformance obligation，不代表全部由 Compiler 执行，也不代表当前已通过。

## 6. Compiler 必须保持的语义边界

- Ingress Operation 是 inbound protocol export，不是 outbound Tool callable。
- Outbound trigger 必须链接到精确 ToolCatalog Operation；不得用 Capability ID、Observation、网络地址或自由文本代替。
- `wait.output` 是独立 Producer SourcePath；不注入全局 Context，不伪装成 Operation Result、Observation Result 或 Evidence。
- Wait 不是第五种业务阶段，也不是独立 Wait Node kind；业务阶段仍是 precondition / operation / observation / postcondition。
- EVENT_ACCEPTED / TIMEOUT 是控制流边，不是 Stage Outcome。
- `responseObligationRef`、`expectedIngressRef`、`deliveryRef`、Host `ingressRef/lifecycleRef`、Run ID 和协议私有 correlation 不得成为 Skill 作者输入或 Compiler 推断出的业务字段。
- `responseDeadline` 是 Host/Runtime 的逻辑 response obligation 上限；每次 HTTP attempt deadline 属于 Integration，不是 Wait timeout。
- Compiler 不实现事件接收、定时器、持久化、原子竞争、重试执行、tombstone 生命周期、Temporal workflow 或 HTTP/2/multipart transport。

## 7. 3GPP 示例与版本边界

当前 5GC 参考 profile 使用独立固定的 Release 19 版本（2026-11-17）：TS 23.502 V19.9.0、TS 29.502 V19.7.0、TS 29.571 V19.7.0（Nsmf OpenAPI 的 Common Data Types companion）、TS 29.500 V19.7.1。它们是示例/Integration 的外部协议依据，不是 CA-SL 语言全局版本。

ca-skills #62 的自动化 source-lock fixture 仍固定在 TS 29.502/29.571 V19.5.0，因为该版本的 ETSI companion ZIP / YAML 已有可复现字节摘要。它明确是历史结构校验 fixture，不是当前 5GC production API pin。真实 Integration Admission 前必须用 V19.7.0 的官方 companion artifacts 重新锁定和验证。Compiler 不能将旧 fixture 宣称为当前 5GC 互操作证据。

## 8. 非目标

- 不实现 Host、Runtime、Integration Adapter、Temporal 或 SBI 客户端。
- 不把 5GC、SMF、AMF、UPF、OpenAMSM 或 MAS 内部逻辑硬编码进 Compiler。
- 不实现包管理、Registry、数据库、发布签名、Qualification 服务或 Runtime Dispatcher。
- 不做 v0/v0.1/v0.2 兼容或自动迁移。
- 不改变 CA-SL v1alpha4 的任何语义、诊断或发布 Artifact。
- 不以真实 Owner / Deployment 缺失为由阻止静态 Compiler 开发；但必须保持真实产品资格 NOT_QUALIFIED。

## 9. 开发启动 Gate

**现在可以做：**契约审查、API 版本盘点、Schema Closure Digest 算法审查、诊断/测试矩阵设计、Compiler extension point 设计和 synthetic fixture 规划。

**正式实现 Gate：**

1. CA-SL 语言 Owner 分配明确的下一语言版本身份，并发布完整单文件规范；不能仅凭 #61 Draft 推断版本。
2. #61 的 grammar、Wait output、graph、response boundary 和诊断条款冻结。
3. #62 的 IngressCatalog schema、Schema Closure Digest、correlation/stale-delivery、response mapping 契约冻结。
4. BuildRequest、DependencyInspection、Artifact Manifest 的新 API 版本决策完成。
5. 版本 dispatch 明确保证 v1alpha4 行为和既有 v1.0.0 Artifact 完全不变。

上述 Gate 完成后即可使用 synthetic trusted snapshots 开始 WP1–WP8；不需要等待真实 5GC Owner 发布或 E2E qualification。产品资格仍独立记录。

## 10. 验收定义

- v1alpha4 regression：所有原有正反例、诊断身份、Artifact bytes 和 Native/WASM parity 不回退。
- 新目标：严格拒绝错误 apiVersion、未声明 export、错误 Owner、Schema digest mismatch、错误 trigger binding、不可用 wait.output、非法 Wait graph、错误 response boundary 和不完整 multipart mapping。
- Native / WASM / Chromium Worker 对同一 BuildRequest 产生相同 Diagnostic[]、DependencyInspection 和 Artifact bytes。
- 本地验收状态必须与 GitHub Actions 分开记录；CI 不可用时可报告 LOCAL_ACCEPTANCE_PASS，但不能写成 CI PASS。
- 真实 Owner / Deployment / 5GC qualification 独立保持 NOT_QUALIFIED，除非有真实版本化发布和现场证据。

