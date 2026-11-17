# CA Skill Compiler 下一语言版本开发计划

**状态：** DRAFT——架构与工作包准备；尚未授权针对新语言目标实施代码。  
**基线：** CA Skill Compiler v1.0.0；目前唯一发布的语言目标是冻结的 CA-SL v1alpha4。  
**下一语言版本：** 尚未分配。本文件不推测或分配版本标识。

## 1. 目标

在现有 Go Core 上增量扩展下一版 CA-SL，支持 Node-entry Event Wait 和包级 Integration Ingress Operation Linking。不是重写；已发布的 v1.0.0 和 v1alpha4 语义保持不可变。

目标能力：
- Node-entry Wait 语法与 AST；
- 受信、包级 IngressCatalog 的准入与链接；
- 精确绑定触发 ToolCatalog Operation，并将其实参映射到类型化 `armInputSchema`；
- `wait.output` Schema 解析及路径敏感可用性分析；
- Wait 入口控制图边、可达性、支配关系和 DAG 检查；
- `wait.response.after` Producer 阶段选择与响应 Schema 映射；
- Native / Go-WASM / Chromium Worker 一致性。

## 2. 契约状态

| 契约 | 仓库 / PR | 状态 | 对 Compiler 的影响 |
|---|---|---|---|
| Node-entry Wait 语义基线 | ca-skills #60 | 已合并 | 语义基础 |
| EVENT_ACCEPTED tombstone / 旧消息隔离 | ca-skills #64 | 已合并 | Runtime 不变量；Compiler 链接声明的 correlation policy |
| Wait 规范性修订 | ca-skills #61 | Draft；语言版本未分配 | Grammar、诊断、SourcePath 和图规则尚未冻结 |
| Integration Ingress Operation | ca-skills #62 | Draft | IngressCatalog、Schema Closure Digest、Normalization 和响应映射尚未冻结 |
| 通用 Host / Runtime 续接 | ca-agent-demo #210 | Draft，仅设计 | Compiler 消费语言/导出契约，不实现 Host/Runtime |
| 5GC SBI 续接示例 | ca-agent-demo #213（替代已关闭的 #209） | Draft，仅设计 | 示例 Profile，不引入 5GC 专用 Compiler 语义 |

v1alpha4 必须继续拒绝作者级 `wait`、`wait.output` 和 `ingressOperation`。现有冻结诊断保持不变。Ingress Operation 是独立的包级导出，不是 ToolCatalog callable、Observation 或 Capability。

真实 Owner / Deployment 发布物及真实 5GC 资格认证，不阻塞独立 Compiler 开发。契约冻结后，Compiler 测试可使用合成受信 Fixture；不得把它们报告为真实互操作证据。

## 3. 版本化 API 影响

当前 v1alpha4 机器契约使用封闭字段和枚举，不能静默扩展。

| 现有契约 | 当前封闭点 | 下一版必须决定 |
|---|---|---|
| BuildRequest v1alpha4 | `TrustedDependencyBinding.exportOwners[].exportKind` 仅允许 operation / observation / schema / evidenceGuarantee；languageTarget 固定为 v1alpha4 | 新 BuildRequest 契约增加独立的 `ingressOperation` 导出类型，并绑定唯一的新语言目标 |
| DependencyInspection v1alpha4 | `callables[]` 仅含 operation / observation，另有 schemas[]；没有包级 Ingress 清单 | 增加类型化 `ingressOperations[]`；不得伪装成 callable |
| Artifact Manifest v1alpha4 | languageTarget 和 dependencyClosure.exportKind 均为封闭集合 | 新 Manifest 记录准确语言目标和 `ingressOperation` 依赖导出 |
| BuildResult v1alpha4 | 输出 Envelope 与 Artifact Manifest 分离 | 先比较实际形状；仅在 Envelope 改变时单独升级 |
| Diagnostics / conformance | Code、Phase 和 E01–E120 / Appendix H 身份已冻结 | 新诊断身份由语言 Owner 发布；Compiler 不得自行发明 |

BuildRequest、DependencyInspection、Artifact Manifest、CA-SL 语言版本和 Compiler 产品版本是独立身份，不能假定它们共用一个版本号。

### 3.1 API Dispatch 与 v1alpha4 隔离

当前入口是版本专用入口，不是通用 Target Router：
- `DecodeBuildRequestJSON` 直接解码 v1alpha4 Go 类型；
- `validateBuildRequest` 强制要求 v1alpha4 BuildRequest API 和 `connectionagent.io/v1alpha4`；
- `Check`、`Build`、`InspectDependencies` 直接进入 v1alpha4 路径；
- v1alpha4 DependencyInspection 和 Artifact Manifest 的字段集合、导出类型枚举均为封闭定义。

下一版必须先做 **API 版本 Dispatch，再做语言 Target Dispatch**：

1. 使用有界 Envelope Decoder，仅读取顶层 `apiVersion`。
2. 按精确 API 版本分派到唯一的严格版本专用 BuildRequest Decoder。未知或不支持的 API 版本必须 fail-closed，不得 fallback。
3. 校验该 API 版本自己的 Schema 和输入限制。
4. 再按精确 `languageTarget` 分派到版本专用 Parser / Linker。未知语言目标必须 fail-closed，不得解释为 v1alpha4。
5. Check / Build / InspectDependencies 和结果验证必须使用同一组选定的 API / Target Profile。

不得先解码成 v1alpha4 结构体再容忍未来字段；不得扩大 v1alpha4 JSON Schema、Go 类型、诊断索引、Artifact Manifest、CLI 行为或 v1.0.0 发布字节。

版本决策必须覆盖以下接口：

| 接口 | 下一版必须作出的决定 |
|---|---|
| BuildRequest | 新的严格 API Schema / Go 类型，绑定精确语言目标，并在受信导出 Owner 绑定中支持独立的 `ingressOperation` |
| DependencyInspection | 新的严格输出契约，增加类型化 `ingressOperations[]`；Ingress 不得混入 `callables[]` |
| Artifact Manifest | 新 Manifest 记录精确语言目标及 dependency closure 中的 `ingressOperation`；不嵌入完整外部 IngressCatalog |
| BuildResult | 仅当输出 Envelope 形状和失败语义不变时保留；否则单独版本化 |
| CLI 工程输入 | 明确扩展本地 `trusted.json` 表达新导出类型；它不是 BuildRequest Wire Contract |
| WASM / Studio Adapter | 使用同一套版本化 API 和 Core Dispatch；不得另写 Parser / Linker |

IngressCatalog 自身的 `apiVersion`、CA-SL `languageTarget`、BuildRequest、DependencyInspection、Artifact Manifest 和 Compiler 产品版本彼此独立。

### 3.2 v1.0.0 继承的实现限制

v1.0.0 发布物保持不可变；其最终验收记录明确标记了若干尚未完成的正向路径：
- WP6：Identity-input provenance 需要后续受信发布契约；
- WP7：当 Owner 尚未发布物理 Descriptor 格式时，Evidence Descriptor 内容、Producer、Scope、Correlation 和 Freshness Linking 仍 fail-closed；
- WP8：跨快照 Reconcile 关联和完整 Retry Admission 仍 fail-closed / 属于 Compiler 之外的部分；
- WP9：Script ABI 中的 Evidence Projection 仍受阻。

当前 Artifact Pipeline 在选中的 Skill 使用 Predicate Evidence、MUTATION Operation 或 `retryOf`，而完整正向路径尚不可用时，会有意拒绝生成成功 Artifact。静态诊断和子检查仍会执行；这不等于这些功能已经完整编译。

这些限制**不阻止独立开发 Node-entry Wait / Ingress Compiler 能力**，但会影响真实 PDU/MAS Skill 的 Artifact 验收。因此：
- 原样保留 v1alpha4 回归测试；
- 明确保留 WP6/WP7/WP8 的受阻案例；
- 对已有冻结契约的 Compiler 静态规则，使用合成受信 Fixture 实现；
- 不得伪造真实 Owner / Deployment 发布物来让正向 Build 通过；
- 在所需静态正向路径完成、外部信任输入到位之前，不得把完整 PDU/MAS Artifact Build 标记为 PASS。

## 4. 目标架构

保留现有 Pipeline：

`BuildRequest → Source Admission → Parse/AST → Dependency Linking → Schema/CEL → Procedure Graph → Script ABI → Artifact`

增加按语言目标分派的独立分支，而不是把新语法插入 v1alpha4：

1. **Language Target Selection：** 解析前选择唯一 Target；未知 Target fail-closed；不回退 v1alpha4。
2. **IngressCatalog Index：** 从受信依赖快照解析包级导出，并验证精确 apiVersion、exportKind、Package ID、Owner、Artifact / Schema 身份。
3. **Wait AST：** 仅新语言 Target 识别 Node-entry `wait`；v1alpha4 仍拒绝。
4. **Ingress Linker：** 精确解析 source / service / operation；不做模糊名称匹配。
5. **Trigger Binding Linker：** 将每条进入 Wait 的路径绑定到精确 ToolCatalog Operation，并从 Operation arguments 或已准入 Invocation Context 解析类型化 `armInputSchema`。
6. **Wait Output Resolver：** 将 `NodeId.wait.output` 绑定到 Normalized Request Schema；验证 JSON Pointer、可赋值性和路径可用性。
7. **Procedure Graph Extension：** 将 Wait 建模为 Node-entry 控制顶点，包含 EVENT_RECEIVED 和 TIMEOUT 边；纳入 DAG、可达性、支配关系、路由义务和 SourcePath 可用性。
8. **Response Boundary Linker：** 根据 `wait.response.after` 和 Ingress 声明的 Producer 阶段类型（operation / observation）推导精确 Producer SourcePath；验证 Output Schema、responseResultSchema 及 JSON / multipart 映射。
9. **Artifact Assembly：** 在新 Manifest 记录精确语言目标和依赖闭包；不把外部 IngressCatalog 复制进 Skill 源文件。

### 4.1 Compiler 静态范围与跨组件符合性

EW / IN 矩阵是联合符合性清单。Compiler 只执行静态可判定义务，不模拟 Host、Runtime 或 Integration 行为。

**#61 中归属 Compiler 的 Wait 案例：**

- **EW-012–EW-016：** Wait Output 路径可用性、业务阶段非空、禁止 TIMEOUT 指向 SUCCEEDED、精确 Ingress Selector 解析、派发前 Correlation Binding。
- **EW-018–EW-019：** 所有入边必须解析到同一受信 Ingress 契约；Response Boundary 不得被绕过。
- **EW-021–EW-022：** Body-bearing 响应的类型化 Producer 映射，以及允许不带 Producer 的固定 Bodyless 响应。
- **EW-027、EW-029–EW-030：** 精确 Duration 语法；禁止非法 Response Result Source；Response Boundary Node 的身份和可达性。
- **EW-031：** 静态保持全局 Precondition 支配关系；Wait 接收事件不产生 Goal 证明。
- **EW-033–EW-036：** 仅允许一个 TIMEOUT 路由；禁止作者声明 EVENT_RECEIVED 路由；Selector 字段形状；Timeout 标量类型。

**#62 中归属 Compiler 的 Ingress 案例：**

- **IN-001–IN-007：** 包级依赖要求、精确导出解析 / 歧义、受信 Owner / Export 身份、重复 ID、固定 OpenAPI Artifact Digest，以及 Service-Operation 与 OpenAPI operationId 的区分。
- **IN-013–IN-015：** Normalized Schema 支持、Normalization Digest 和必需 Wire 字段覆盖。
- **IN-018–IN-021：** 精确 ToolCatalog Operation Trigger 身份、派发前参数绑定、Start Context 绑定及所有入边覆盖。
- **IN-023：** 静态可判定的旧消息安全证明失败；依赖部署事实的 Admission 检查必须单独标记。
- **IN-028–IN-032：** Response Disposition 数量规则、Body-bearing Producer 要求、精确 Response Boundary Stage、Schema 可赋值性和必需响应字段覆盖。
- **IN-039–IN-040：** Accepted Media Type / Source Schema 一致性、重复 Disposition ID / Status。
- **IN-042、IN-044–IN-047：** 静态可判定的 Binary Content 保留期限约束、固定 Multipart Response 完整性、3GPP 拒绝响应映射完整性和 Discriminator Retention 时间计算。

以下仍属于 Integration / Host / Runtime / Deployment 符合性义务，不是 Compiler 测试：EW-001–EW-011、EW-017、EW-020、EW-023–EW-026、EW-028、EW-032；以及 IN-008–IN-012、IN-016–IN-017、IN-022、IN-024–IN-027、IN-033–IN-038、IN-041、IN-043、IN-048–IN-051、IN-053–IN-054。IN-023、IN-039、IN-042、IN-044–IN-047 等共享案例可同时有 Compiler 静态 Fixture 和独立 Admission / Integration Fixture；不得把其中一个冒充另一个。

### 4.3 WP1 Schema Closure Digest v1 实施边界

当前 #62 提案已经超出普通递归 `$ref` 遍历。Compiler 必须把以下规则作为一个确定性的 Linking 输入整体实现：

1. **受信 Artifact 绑定：** 将每个不透明 `artifactRef` 与规范化快照相对路径、Canonical Document URI、源 Dialect、解析后的 Document 及完整原始字节一一绑定。必须从不可变快照的真实字节重新计算 `artifactDigest`；不得信任 IngressCatalog 或测试向量声明的 Digest。
2. **Canonical URI 规则：** 校验绝对、分层的 Snapshot Base URI；从百分号编码后的规范化路径推导 Document URI；规范化 Scheme / Host / 默认端口、Dot Segment 和 Percent Escape；拒绝 User-info、Query、Fragment、空白、反斜杠、非法 Escape 和歧义映射。URI 查询只能查内存快照，不得访问网络。
3. **Dialect-aware Schema 发现：** OpenAPI 3.0 的根节点只能来自 OpenAPI Document 中的 Schema Object 位置，并遵循 OpenAPI 3.0 Schema 规则；JSON Schema Draft 2020-12 必须声明对应 Dialect，并使用其 Schema-valued Keywords。不得把一个 Dialect 转换解释成另一个。
4. **嵌套资源身份：** OpenAPI 3.0 Schema Object 不得用 `$id` 建立 Base URI。JSON Schema 2020-12 必须按继承 Base 解析嵌套 `$id`，并将唯一、不可变的 Resource URI Alias 注册到精确的 Artifact-relative Schema Pointer。
5. **引用解析：** 每个 `$ref` 按所在 Schema 的 Effective Base URI 解析；只允许解析到同一受信快照中的 Canonical Document URI 或已注册嵌套 `$id` Alias。Fragment 必须按严格 UTF-8 只解码一次，校验 RFC 6901，并将嵌套资源相对 Fragment 转换回精确的 Artifact-relative Pointer。
6. **精确闭包：** 仅遍历对应 Dialect 支持的 Schema-valued 子结构；闭包必须包含 Root 及其全部且仅有的传递引用 Schema 节点；节点身份为精确的 `(artifactRef, pointer)`；循环按身份只访问一次；缺失 / 多余节点、Boolean Schema、非 Schema Pointer、不支持的 Anchor / Dynamic Reference、非法 ID 和快照越界均 fail-closed。
7. **Digest：** 按规范构造 Manifest，包含每个节点的完整源 Artifact Digest 和精确解析后的 Schema 值；按 UTF-8 `artifactRef`、Pointer 排序；使用 RFC 8785 JCS；对精确的 v1 Domain Separator 与 JCS 字节求哈希。

Compiler 符合性向量必须同时证明**原始字节绑定**和**引用解析**：即使解析后的 Schema 值不变，只要 Artifact 字节变化，`artifactDigest` 和 Closure Digest 也必须变化；嵌套 `$id` 的 Fragment 必须能解析到非 Root 的 Artifact-relative Pointer。只提供预解析引用边或硬编码 Digest 的向量不足以证明 Resolver 符合性。

该实现必须与冻结的 v1alpha4 Schema Registry 隔离。不得静默改写 v1alpha4 的 Schema 身份和本地引用语义以采用新的 Ingress URI-reference 算法。

### 4.2 诊断归属与优先级

代码实施前，语言 Owner 与导出契约 Owner 必须发布无冲突的诊断映射：

- `WAIT_*` 表示 Skill 作者声明的 Wait、Wait 图、路径可用性、Trigger-to-Wait 绑定或 Skill Response Boundary 问题。
- `INGRESS_*` 表示 IngressCatalog 导出自身结构不合法、不受信，或其 Protocol、Normalization、Correlation、Trigger 声明、Response Mapping 不合法。
- #61 明确要求复用现有诊断的场景，必须复用冻结的 `SOURCE_PATH_*`、`CONTROL_FLOW_*`、`INPUT_TYPE_MISMATCH` 和 `DEPENDENCY_*`。
- Wait Selector 找不到 / 匹配多个 Ingress Export，分别使用 `WAIT_INGRESS_EXPORT_NOT_FOUND` / `WAIT_INGRESS_EXPORT_AMBIGUOUS`；Catalog 格式错误或 Export 重复属于 Ingress Export 错误。
- 触发 Operation 不在受信快照中，属于 Ingress Trigger Export 错误；Skill 路径无法在所有入边上派生 Arm Input，属于 Wait Correlation Binding 错误。
- Ingress 自身 Response Mapping 错误，与 Skill 缺失或无法到达 `wait.response.after` Producer 是不同问题。
- 每个新 Compiler 诊断必须有唯一 `(Code, Phase)`、Severity、优先级和源码位置规则。#62 当前候选 `INGRESS_*` 列表尚未冻结全部属性；Compiler 不得自行发明或静默别名化诊断身份。

## 5. 工作包

| WP | 工作 | 主要产出 | 入口条件 |
|---|---|---|---|
| WP0 | Language / API Freeze Inventory | 完整单文件规范、语言版本身份、BuildRequest / DependencyInspection / Artifact 决策、诊断目录 | 独立复审 #61 / #62 |
| WP1 | IngressCatalog Admission / Linking | Export Parser、受信 Owner Binding、原始 Artifact Digest 校验、Dialect-aware Schema Closure Digest v1 和精确传递引用闭包 | WP0 |
| WP2 | Wait Syntax / AST | Node-entry Grammar、Duration Parser、严格未知字段诊断 | WP0 |
| WP3 | Trigger Binding | 精确 Operation 身份、armInputSchema 绑定、Start Context 绑定 | WP1 + WP2 |
| WP4 | wait.output Dataflow | Normalized Schema 解析、JSON Pointer、可赋值性、边 / 路径敏感可用性 | WP1 + WP2 |
| WP5 | Wait Graph | 入口控制顶点、EVENT_RECEIVED / TIMEOUT、DAG / Dominance / Reachability / 路由义务 | WP2–WP4 |
| WP6 | Response Boundary | response.after、Producer 阶段推导、responseResultSchema、JSON / multipart / binary 映射 | WP1–WP5 |
| WP7 | Diagnostics / Conformance | 精确 Code / Phase、正反静态 Fixture、Case-to-Test 映射 | WP0–WP6 |
| WP8 | Native / WASM / Worker Parity | Go Test、Vet、CGO-free Native、WASM、真实 Chromium Parity、Artifact 字节一致性 | WP1–WP7 |
| WP9 | Studio / LSP Integration | DependencyInspection UI、Wait / Ingress 诊断、Completion 和 Schema 导航 | API 契约 + WP8 |

EW-001–EW-036 和 IN-001–IN-054 是跨组件符合性义务，不是全部由 Compiler 负责的测试，也没有声称已经通过。

## 6. Compiler 不变量

- Ingress Operation 是入站协议导出，不是出站 Tool Callable。
- 出站 Trigger 必须解析到精确 ToolCatalog Operation；Capability ID、Observation、网络地址或自由文本标签均不能替代。
- `wait.output` 是独立 Producer SourcePath，不是全局 Context、Operation Result、Observation Result 或 Evidence。
- Wait 不是第五种业务阶段，也不是独立的 Wait Node 类型。业务阶段仍为 precondition / operation / observation / postcondition。
- EVENT_RECEIVED 和 TIMEOUT 是控制流边，不是 Stage Outcome。
- `responseObligationRef`、`expectedIngressRef`、`deliveryRef`、Host `ingressRef/lifecycleRef`、Run ID 和协议私有 Correlation 不得成为 Skill Input 或推断出的业务字段。
- `responseDeadline` 是 Host / Runtime 逻辑响应义务上限；每次 HTTP Attempt Deadline 由 Integration 管理，不是 Wait Timeout。
- Compiler 不实现事件接收、计时器、持久化、原子仲裁、Retry 执行、Tombstone 生命周期、Temporal Workflow 或 HTTP/2 / Multipart Transport。

## 7. 3GPP 参考边界

当前 5GC 示例 Profile 使用分别锁定的 Release 19 版本（2026-11-17 核对）：TS 23.502 V19.9.0、TS 29.502 V19.7.0、TS 29.571 V19.7.0（Nsmf OpenAPI 的 Common Data Types 配套规范）和 TS 29.500 V19.7.1。这些是外部协议参考，不是全局 CA-SL 版本。

ca-skills #62 中自动化 Source Lock Fixture 仍使用 TS 29.502 / TS 29.571 V19.5.0，因为对应 ETSI ZIP / YAML 字节具有可复现校验和。它明确是历史结构验证 Fixture，不是当前生产 5GC API Pin。真实 Integration Admission 必须重新锁定并验证 V19.7.0 官方配套 Artifact。

## 8. 非目标

- 不实现 Host、Runtime、Integration Adapter、Temporal 或 Tool。
- Compiler 不包含 5GC、SMF、AMF、UPF、OpenAMSM 或 MAS 专用逻辑。
- 不实现包管理器、Registry、数据库、Release Signing、Qualification Service 或 Runtime Dispatcher。
- 不支持 v0 / v0.1 / v0.2 兼容或自动迁移。
- 不改变 CA-SL v1alpha4 语义、诊断或已发布 Artifact。
- 缺少真实 Owner / Deployment Artifact 不阻塞静态 Compiler 开发，但真实产品资格认证仍为 NOT_QUALIFIED。

## 9. 实施门槛

**现在可以开始：** 契约复审、API 版本 / Target Dispatch 设计、v1alpha4 兼容性 Harness、Compiler 静态 EW / IN Fixture 清单、内部接口下的 Schema Closure Digest 实现 Spike，以及合成 Snapshot / IngressCatalog Fixture 准备。这些工作不得暴露新公开 Target，也不得改变已发布 v1alpha4 路径。

**正式实施新 Target 前必须满足：**

1. CA-SL 语言 Owner 分配下一语言版本，并发布完整单文件规范。
2. #61 的 Grammar、Wait Output、Graph、Response Boundary、诊断和 Compiler 静态符合性子集冻结。
3. #62 的 IngressCatalog Schema、Schema Closure Digest、Correlation / Stale Delivery、Trigger Binding 和 Response Mapping 契约冻结，包括精确诊断命名空间及优先级。
4. 独立决定新的 BuildRequest、DependencyInspection、Artifact Manifest API 版本；根据实际结构明确保留或升级 BuildResult。
5. Target Dispatch 保持 v1alpha4 行为和 v1.0.0 发布 Artifact 完全不变。
6. 对 v1alpha4 WP6 / WP7 / WP8 的既有实现限制作出明确承接决定，不能让新 Target 对仍会 fail-closed 的功能宣称正向 Build 支持。

满足这些门槛后，WP1–WP8 可以使用合成受信 Snapshot 实施。真实 5GC Owner 发布物或 E2E 资格认证不是 Compiler 开发前置条件；它们属于独立的产品资格认证门槛。

## 10. 验收

- v1alpha4 回归：现有正反 Fixture、诊断身份、Artifact 字节和 Native / WASM 一致性均不改变。
- 新 Target：对错误 apiVersion、缺失 Export、Owner 错误、Schema Digest 不匹配、Trigger Binding 无效、wait.output 不可用、Wait Graph 错误、Response Boundary 错误或 Multipart 映射不完整均 fail-closed。
- 相同 BuildRequest 在 Native、WASM 和 Chromium Worker 产生完全一致的 Diagnostic[]、DependencyInspection 和 Artifact 字节。
- Local Acceptance 与 GitHub Actions 状态分开报告。CI 不可用时报告 LOCAL_ACCEPTANCE_PASS，不得写 CI PASS。
- 真实 Owner / Deployment / 5GC 资格认证在有版本化真实发布物和在线证据前保持 NOT_QUALIFIED。
