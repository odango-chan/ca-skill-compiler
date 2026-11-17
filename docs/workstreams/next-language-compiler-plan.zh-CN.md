# 下一代 CA Skill Compiler 开发计划

**状态：** DRAFT — 可以开始隔离的 v1alpha5 候选实现；公开 Target 启用与发布仍受门槛约束。  
**基线：** CA Skill Compiler v1.0.0；已发布的 v1alpha4 保持不可变。  
**下一语言目标：** `connectionagent.io/v1alpha5` 已在 CA-SL 评审候选中分配，但仍为 NOT RELEASED / NOT QUALIFIED。

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
| v1alpha5 单文件语言候选 | ca-skills #61，head `e6569713` | Draft；Target 已分配；语义基线标记为冻结；全文与符合性材料仍在独立评审 | 已有约 5,474 行单文件候选；附录 L 定义 Wait 语法/语义；附录 H 增加 9 个 C_STATIC WAIT 诊断；EW-001–EW-036 仍为仅规范义务 |
| Integration Ingress Operation | ca-skills #62，head `d0fe2be` | Draft；当前 Schema / Vector 两个工作流 PASS；契约未冻结 | IngressCatalog、精确链接语义及诊断归属仍需收敛；原始字节与嵌套资源 Resolver 向量缺口已在当前 Head 修复 |
| Generic Host / Runtime continuation | ca-agent-demo #210 | Draft，设计文档 | Compiler 只消费语言和 Ingress 契约，不实现 Host/Runtime |
| 5GC SBI continuation profile | ca-agent-demo #213（替代已关闭的 #209） | Draft，设计文档 | 3GPP 示例，不是 Compiler 的 5GC 专用语义 |

**重要边界：**

- v1alpha4 不接受作者级 `wait`、`wait.output` 或 `ingressOperation`。现有 `UNSUPPORTED_WAIT` 等冻结诊断必须继续保持。
- v1alpha5 的 Target 身份已经明确，不再是未分配：当前 #61 候选指定 `connectionagent.io/v1alpha5`，并已提供完整单文件评审稿。文件头将语义基线标记为冻结，但完整规范文本与符合性材料仍待独立评审；当前仍为 NOT RELEASED / NOT QUALIFIED。36 项 EW 义务全部明确标记为 `SPECIFICATION_ONLY`，不是已执行测试。
- v1alpha4 必须继续拒绝作者级 `wait`、`wait.output` 和 `ingressOperation`；其 Decoder、诊断和已发布字节不得改变。
- 当前 #62 已增加对每个 Artifact 原始字节的 `artifactDigest` 复算，并验证嵌套 `$id` Fragment 能映射到非根 Artifact Pointer；两个工作流通过。但 PR 仍为 Draft，Ingress 导出契约以及 `INGRESS_*` 与既有诊断的归属/优先级尚未冻结。

**实施决定：** 可以基于 #61 精确的 v1alpha5 候选启动隔离的 WP2 Wait Grammar / AST；WP1 Schema Closure 继续通过内部接口实现。不得启用公开 v1alpha5 Build Target，不得修改 v1alpha4，也不得提前宣称 Compiler 符合性。

第一项隔离的 WP2 实现已建立为 [ca-skill-compiler PR #30](https://github.com/odango-chan/ca-skill-compiler/pull/30)：独立包内的 Wait AST / Parser，覆盖固定时长、Selector、Response Boundary 和 TIMEOUT 路由检查。它没有增加 BuildRequest Dispatch，也没有链接 IngressCatalog。当前 GitHub Actions Job 没有分配 Runner（`runner_id=0`，没有执行步骤）；这是 CI / Runner 可用性问题，不能当作测试失败结论。目前只有固定时长语法做了独立本地 Go 检查；完整 Go tests / vet / Native / WASM 验收仍为 NOT_RUN。
- #62 的 IngressCatalog 是独立 package-scoped export，不是 ToolCatalog Operation、Observation 或 Capability。
- #62 的 Schema Closure Digest v1 目前仍是提案。其 golden vector / CI 通过只证明算法样例，不等于 Owner 已发布真实 Schema。
- 真实 Owner / Deployment 发布与实时 5GC qualification 不阻止 Compiler 独立实现；实现阶段可使用明确标记的 synthetic trusted fixtures。不得将 synthetic fixtures 报告为真实互操作通过。

## 3. 必须先做的 API 版本盘点

v1alpha4 的机器契约使用封闭字段/枚举，不能在原版本上静默添加 Ingress export kind。

| 现有契约 | 已发现的封闭点 | 下一版需要的决策 |
|---|---|---|
| BuildRequest v1alpha4 | `TrustedDependencyBinding.exportOwners[].exportKind` 只允许 operation / observation / schema / evidenceGuarantee；languageTarget 固定 v1alpha4 | 新 API contract 增加独立 `ingressOperation` export kind，并绑定唯一新 languageTarget |
| DependencyInspection v1alpha4 | `callables[]` 仅 operation / observation，另有 schemas[]；没有 package-level Ingress inventory；诊断使用封闭的 v1alpha4 Envelope | 新契约增加类型化 `ingressOperations[]` 和 v1alpha5 Diagnostic Envelope；Ingress 不得伪装成 callable |
| Artifact Manifest v1alpha4 | `languageTarget` 固定 v1alpha4；dependencyClosure.exports 的 exportKind 为封闭枚举 | 新 Artifact manifest contract 必须记录新 languageTarget 和 `ingressOperation` dependency export |
| BuildResult v1alpha4 | 输出 Envelope 及每个 Diagnostic 项均为 `additionalProperties: false` | **必须升级 BuildResult API**：Ingress 诊断要求 `registryId`、`messageKey`、类型化 `sourceLocation` 和可选 `relatedLocations`，v1alpha4 Diagnostic Schema 无法表达 |
| Diagnostic / Conformance | v1alpha5 Appendix H 组合 9 个 WAIT 诊断和独立的 9-code IngressCatalog v1alpha1 注册表；身份仍为 `(Code, Phase)` | 冻结精确组合索引；Ingress `registryId` 是来源信息，不是第二套身份键 |

BuildRequest、DependencyInspection、Artifact Manifest 的版本号彼此独立于 CA-SL 语言版本和 Compiler 产品版本。**不得假定它们必须同号，也不得在版本身份未裁决前写入猜测值。**

### 3.2 Diagnostic Envelope 迫使 BuildResult 与 DependencyInspection 升级

当前 `Diagnostic` Go 类型只包含 `severity`、`phase`、`code`、`path`、可选 `line/column` 和渲染后的 `message`。v1alpha4 BuildResult JSON Schema 对每个 Diagnostic 项设置了 `additionalProperties: false`。

IngressCatalog v1alpha1 诊断注册表要求 Compiler 的逻辑 Diagnostic 记录保留：
- `code`、`phase=C_STATIC`、`severity=ERROR`；
- `registryId=connectionagent.io/ingress-diagnostics/v1alpha1`；
- 稳定的 `messageKey`；
- 包含 `dependencyAlias`、`snapshotPath` 和 RFC 6901 `jsonPointer` 的 `sourceLocation`；
- 用于重复项及跨 Artifact 解释的可选 `relatedLocations`。

因此 BuildResult 已不再是“结构变化时才升级”：**BuildResult 与 DependencyInspection 都必须使用新的版本化 Diagnostic Envelope**。新契约必须同时表达 Skill 源码位置和依赖快照位置，不能把后者压扁成含义不明确的字符串 `path`。对于 Ingress 诊断，`registryId` 是来源信息；诊断身份仍为全局唯一的 `(Code, Phase)`。精确的 tagged source-location 结构及条件约束（例如 Ingress 诊断必须有 `registryId/messageKey/sourceLocation`）属于 WP0 API 设计产物。

v1alpha4 Go 类型、JSON Schema、诊断 Wire Shape 和发布字节保持不变。新 API 版本标识独立于 `connectionagent.io/v1alpha5`；不得仅为命名整齐而假定使用相同后缀。

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

| 接口 | 下一版必须作出的决定 |
|---|---|
| BuildRequest | 新的严格 API Schema / Go 类型，绑定精确语言目标，并在受信导出 Owner 绑定中支持独立的 `ingressOperation` |
| DependencyInspection | 新的严格输出契约，增加类型化 `ingressOperations[]`；Ingress 不得混入 `callables[]` |
| Artifact Manifest | 新 Manifest 记录精确语言目标及 dependency closure 中的 `ingressOperation`；不嵌入完整外部 IngressCatalog |
| BuildResult | 必须使用新版本：Ingress 诊断注册表增加 `registryId`、`messageKey`、依赖感知 `sourceLocation` 和可选 `relatedLocations`；v1alpha4 封闭 Diagnostic 项无法表达 |
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

这些限制不阻止独立开发 Node-entry Wait / Ingress Compiler 能力，但会影响真实 PDU/MAS Skill 的 Artifact 验收。原样保留 v1alpha4 回归测试；对已有冻结契约的 Compiler 静态规则使用合成受信 Fixture；不得伪造真实 Owner / Deployment 发布物来让正向 Build 通过；所需静态正向路径完成、外部信任输入到位前，不得把完整 PDU/MAS Artifact Build 标记为 PASS。

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

### 4.1 Compiler 静态范围与跨组件符合性

EW / IN 矩阵是联合符合性清单。Compiler 只执行静态可判定义务，不模拟 Host、Runtime 或 Integration 行为。

**#61 中归属 Compiler 的 Wait 案例：**
- EW-012–EW-016：Wait Output 路径可用性、业务阶段非空、禁止 TIMEOUT 指向 SUCCEEDED、精确 Ingress Selector 解析、派发前 Correlation Binding。
- EW-018–EW-019：所有入边必须解析到同一受信 Ingress 契约；Response Boundary 不得被绕过。
- EW-021–EW-022：Body-bearing 响应的类型化 Producer 映射，以及允许不带 Producer 的固定 Bodyless 响应。
- EW-027、EW-029–EW-030：精确 Duration 语法；禁止非法 Response Result Source；Response Boundary Node 的身份和可达性。
- EW-031：静态保持全局 Precondition 支配关系；Wait 接收事件不产生 Goal 证明。
- EW-033–EW-036：仅允许一个 TIMEOUT 路由；禁止作者声明 EVENT_RECEIVED 路由；Selector 字段形状；Timeout 标量类型。

**#62 中归属 Compiler 的 Ingress 案例：**
- IN-001–IN-007：包级依赖要求、精确导出解析 / 歧义、受信 Owner / Export 身份、重复 ID、固定 OpenAPI Artifact Digest，以及 Service-Operation 与 OpenAPI operationId 的区分。
- IN-013–IN-015：Normalized Schema 支持、Normalization Digest 和必需 Wire 字段覆盖。
- IN-018–IN-021：精确 ToolCatalog Operation Trigger 身份、派发前参数绑定、Start Context 绑定及所有入边覆盖。
- IN-023：静态可判定的旧消息安全证明失败；依赖部署事实的 Admission 检查必须单独标记。
- IN-028–IN-032：Response Disposition 数量规则、Body-bearing Producer 要求、精确 Response Boundary Stage、Schema 可赋值性和必需响应字段覆盖。
- IN-039–IN-040：Accepted Media Type / Source Schema 一致性、重复 Disposition ID / Status。
- IN-042、IN-044–IN-047：静态可判定的 Binary Content 保留期限约束、固定 Multipart Response 完整性、3GPP 拒绝响应映射完整性和 Discriminator Retention 时间计算。

以下仍属于 Integration / Host / Runtime / Deployment 符合性义务，不是 Compiler 测试：EW-001–EW-011、EW-017、EW-020、EW-023–EW-026、EW-028、EW-032；以及 IN-008–IN-012、IN-016–IN-017、IN-022、IN-024–IN-027、IN-033–IN-038、IN-041、IN-043、IN-048–IN-051、IN-053–IN-054。IN-023、IN-039、IN-042、IN-044–IN-047 等共享案例可同时有 Compiler 静态 Fixture 和独立 Admission / Integration Fixture；不得把其中一个冒充另一个。

### 4.2 诊断归属与优先级

代码实施前，语言 Owner 与导出契约 Owner 必须发布无冲突的诊断映射：
- `WAIT_*` 表示 Skill 作者声明的 Wait、Wait 图、路径可用性、Trigger-to-Wait 绑定或 Skill Response Boundary 问题。
- `INGRESS_*` 表示 IngressCatalog 导出自身结构不合法、不受信，或其 Protocol、Normalization、Correlation、Trigger 声明、Response Mapping 不合法。
- #61 明确要求复用现有诊断的场景，必须复用冻结的 `SOURCE_PATH_*`、`CONTROL_FLOW_*`、`INPUT_TYPE_MISMATCH` 和 `DEPENDENCY_*`。
- Wait Selector 找不到 / 匹配多个 Ingress Export，分别使用 `WAIT_INGRESS_EXPORT_NOT_FOUND` / `WAIT_INGRESS_EXPORT_AMBIGUOUS`；Catalog 格式错误或 Export 重复属于 Ingress Export 错误。
- 触发 Operation 不在受信快照中，属于 Ingress Trigger Export 错误；Skill 路径无法在所有入边上派生 Arm Input，属于 Wait Correlation Binding 错误。
- Ingress 自身 Response Mapping 错误，与 Skill 缺失或无法到达 `wait.response.after` Producer 是不同问题。
- 每个新 Compiler 诊断必须有唯一 `(Code, Phase)`、Severity、优先级和源码位置规则。#62 当前候选 `INGRESS_*` 列表尚未冻结全部属性；Compiler 不得自行发明或静默别名化诊断身份。

### 4.3 WP1 Schema Closure Digest v1 实施边界

当前 #62 提案已经超出普通递归 `$ref` 遍历。Compiler 必须整体实现：

1. **受信 Artifact 绑定：** 将每个不透明 `artifactRef` 与规范化快照相对路径、Canonical Document URI、源 Dialect、解析后的 Document 及完整原始字节一一绑定；从不可变快照真实字节重新计算 `artifactDigest`，不得信任 IngressCatalog 声明值。
2. **Canonical URI 规则：** 校验绝对、分层的 Snapshot Base URI；从百分号编码后的规范化路径推导 Document URI；规范化 Scheme / Host / 默认端口、Dot Segment 和 Percent Escape；拒绝 User-info、Query、Fragment、空白、反斜杠、非法 Escape 和歧义映射。URI 查询只能查内存快照，不得访问网络。
3. **Dialect-aware Schema 发现：** OpenAPI 3.0 的根节点只能来自 OpenAPI Document 中的 Schema Object 位置，并遵循 OpenAPI 3.0 Schema 规则；JSON Schema Draft 2020-12 必须声明 Dialect 并使用其 Schema-valued Keywords。不得混用两种 Dialect。
4. **嵌套资源身份：** OpenAPI 3.0 Schema Object 不得用 `$id` 建立 Base URI。JSON Schema 2020-12 必须按继承 Base 解析嵌套 `$id`，并将唯一资源 URI Alias 注册到精确的 Artifact-relative Schema Pointer。
5. **引用解析：** 每个 `$ref` 按所在 Schema 的 Effective Base URI 解析；只允许同一受信快照中的 Canonical Document URI 或嵌套 `$id` Alias。Fragment 按严格 UTF-8 只解码一次，校验 RFC 6901，并将嵌套资源相对 Fragment 转换为精确 Artifact-relative Pointer。
6. **精确闭包：** 仅遍历对应 Dialect 支持的 Schema-valued 子结构；闭包包含 Root 及其全部且仅有的传递引用 Schema 节点；节点身份为精确 `(artifactRef, pointer)`；循环按身份只访问一次；缺失 / 多余节点、Boolean Schema、非 Schema Pointer、不支持的 Anchor / Dynamic Reference、非法 ID 和快照越界均 fail-closed。
7. **Digest：** Manifest 包含每个节点的完整源 Artifact Digest 和精确解析后的 Schema 值；按 UTF-8 `artifactRef`、Pointer 排序；使用 RFC 8785 JCS；对精确 v1 Domain Separator 与 JCS 字节求哈希。

Compiler 向量还必须证明原始字节绑定：即使解析后的 Schema 值不变，只要 Artifact 字节变化，`artifactDigest` 和 Closure Digest 也必须变化；并测试嵌套 `$id` Fragment 解析到非 Root 的 Artifact-relative Pointer。只提供预解析引用边或硬编码 Digest 不足以证明 Resolver 符合性。

该实现必须与冻结的 v1alpha4 Schema Registry 隔离，不得静默改写 v1alpha4 的 Schema 身份和本地引用语义。

## 5. Work Packages

| WP | 工作 | 主要产物 | 开始条件 |
|---|---|---|---|
| WP0 | 语言与 API 冻结盘点 | 完整单文件规范、版本身份、BuildRequest / DependencyInspection / Artifact contract 决策、诊断清单 | #61 / #62 完成独立审查 |
| WP1 | IngressCatalog admission / linking | export parser、trusted Owner binding、原始 Artifact Digest 校验、Dialect-aware Schema Closure Digest v1、exact transitive reference closure | WP0 |
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

**现在可以启动（隔离的候选实现）：**
- WP2：针对 #61 精确的 v1alpha5 单文件候选实现并测试 Wait Grammar / AST；不接入 v1alpha4 Dispatch。
- WP1：通过内部接口实现 Schema Closure Resolver，使用不可变合成 Snapshot；当前 #62 Vector 已覆盖原始字节绑定和嵌套资源 Pointer 转换。
- 建立 v1alpha4 兼容性 Harness 与 EW / IN 静态 Fixture 清单。
- 设计先 API-version、后 language-target 的双层 Dispatch。

**公开 v1alpha5 Target 集成 / 发布前仍须满足：**

1. #61 完成全文独立评审，冻结并发布 v1alpha5 规范单文件及精确诊断索引。
2. #62 冻结 IngressCatalog Schema、Schema Closure Digest、Correlation / Stale Delivery、Trigger Binding 和 Response Mapping 契约。
3. #61 / #62 发布无冲突的诊断归属与优先级契约：适用时复用现有依赖 / Export / Schema / SourcePath 诊断；WAIT 诊断只负责 Skill Wait 选择、绑定和响应边界问题；剩余 INGRESS 诊断必须有精确 `(Code, Phase)`、Severity、优先级和源码位置规则。
4. 独立决定新的 BuildRequest、DependencyInspection、Artifact Manifest API 版本；按实际结构明确保留或升级 BuildResult。
5. Target Dispatch 保持 v1alpha4 行为和 v1.0.0 发布 Artifact 完全不变。
6. 对 v1alpha4 WP6 / WP7 / WP8 的既有实现限制作出明确承接决定，不能让新 Target 对仍会 fail-closed 的功能宣称正向 Build 支持。

真实 5GC Owner 发布物或 E2E 资格认证不阻塞隔离的静态 Compiler 开发；它们仍属于独立的产品资格认证门槛。

## 10. 验收定义

- v1alpha4 regression：所有原有正反例、诊断身份、Artifact bytes 和 Native/WASM parity 不回退。
- 新目标：严格拒绝错误 apiVersion、未声明 export、错误 Owner、Schema digest mismatch、错误 trigger binding、不可用 wait.output、非法 Wait graph、错误 response boundary 和不完整 multipart mapping。
- Native / WASM / Chromium Worker 对同一 BuildRequest 产生相同 Diagnostic[]、DependencyInspection 和 Artifact bytes。
- 本地验收状态必须与 GitHub Actions 分开记录；CI 不可用时可报告 LOCAL_ACCEPTANCE_PASS，但不能写成 CI PASS。
- 真实 Owner / Deployment / 5GC qualification 独立保持 NOT_QUALIFIED，除非有真实版本化发布和现场证据。

