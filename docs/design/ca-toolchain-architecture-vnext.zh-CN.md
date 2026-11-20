# CA-SL 工具链架构：编译、接口链接与执行资格分离

> **状态：原则已通过评审；以下契约为提案，尚未冻结或实现。**
>
> 当前基线为冻结的 CA-SL v1alpha5（`connectionagent.io/v1alpha5`）和已发布的 Compiler v2.0.2。现有语言规则、BuildRequest / BuildResult / Artifact 契约及实现行为以冻结基线为准。迁移步骤见[演进路线](compiler-evolution-roadmap-vnext.zh-CN.md)，Host 和 MAS E2E 边界见 [ca-agent-demo #217](https://github.com/odango-chan/ca-agent-demo/pull/217)。

## 1. 已批准的原则

1. Capability Catalog 定义能力身份与 Callable 成员关系；Deployment Grant 由部署方签发，授权具体环境中的使用。
2. 接口与静态 EvidenceDescriptor 闭合时，即使缺少运行时 Evidence、Assurance Attestation 或 Deployment Grant 仍可完成 Development Build。产物保留显式的两类 Requirement：派发前的外部事实记入 Host Admission Requirement，执行期才产生的真实提交与恢复结论记入 Runtime Execution Requirement。在 Host Admission 成功前禁止执行。静态 Descriptor 缺失或非法仍是 Build 失败。
3. Interface Package 与 Runtime Provider Package 各自发布、定版和回滚。Provider 以精确接口身份声明兼容性。

Compile 验证源码与语言静态规则，Interface Link 固定接口和类型闭包，Build 生成确定性产物，Qualification 由外部验证机构给出特定组合的保证结论，Host Admission 针对具体 Invocation 决定是否允许执行。

## 2. 发布与信任边界

~~~mermaid
flowchart LR
    SRC["Types, OpenAPI, Owner contracts"] --> PUB["Interface Publisher"]
    PUB --> IP["Interface Package"]
    IMPL["Provider implementation"] --> RP["Runtime Provider Package"]
    IP --> IR["Interface distribution"]
    IR --> RES["External Resolver / Studio / CI"]
    RES --> IN["Exact snapshots + dependencies.lock"]
    IN --> C["Offline Compiler"]
    SK["Skill source"] --> C
    C --> ART["Linked Skill Artifact"]
    RP --> DEP["Deployment inventory"]
    EA["Owner / Assurance Authority"] --> AE["Assurance Attestation（外部信任产物）"]
    EA -->|static EvidenceDescriptor as Interface Package export| IP
    AE -->|verified claims| H["Host Admission"]
    ART --> H
    DEP --> H
    DA["Deployment Authority"] -->|Deployment Grant| H
~~~

| 角色 | 产物或权限 |
|---|---|
| Interface Publisher | 确认公共接口、Effect 和错误语义；签发接口包发布声明。类型与导出清单可自动提取，副作用语义需 Owner 确认 |
| Runtime Provider Publisher | 发布实现及其受支持的接口身份；声明兼容范围并提交逐摘要兼容证明 |
| Resolver | 选择版本、获取字节、核验来源与发布权限、生成锁定输入；显式更新锁文件 |
| Compiler | 离线复算摘要、验证闭包、链接、生成产物及诊断；签名来源与执行授权由各自权威验证 |
| Owner / Evidence Authority | 以 Host 认可的权限对 Operation 保证、Evidence 规则或资格结论签发声明；Assurance Attestation 是外部信任产物，不是 Compiler 输入 |
| Deployment Authority | 发行和撤销环境及主体范围内的 Deployment Grant |
| Host | 依据独立配置的信任根、环境策略及当前 Invocation 验证以上材料；默认拒绝，只有授权路径可派发 |
| Runtime / Provider | 在执行期产生真实的 Mutation 提交证明与 Reconcile 结论；结果不进入已编译产物 |

包名是逻辑身份，版本是发布标识，摘要固定字节，签名关联明确的身份和授权。Registry、包内的 Owner 声明、Compiler 的静态 PASS 和摘要匹配均不能代替 Host 的信任判定。

## 3. 包、锁和运行授权

### 3.1 Interface Package 与 Capability Catalog

首版采用一个不可变目录快照：根 `package.json`、显式列出的 export 文件和 Schema 文件。Manifest 包含 `apiVersion`、`packageId`、`version`、语言目标、发布者命名空间、直接依赖约束、导出项 `{kind,id,path}` 及文件清单 `{path,sha256}`。发布声明在快照外签名，签名覆盖包身份、版本与快照摘要。包发布采用 SemVer：对既有消费者破坏接口、Effect、成员关系或协议约束时升主版本；新增兼容导出升次版本；兼容修订升修订版本。任一内容变更发布新版本和摘要；同一版本不得覆写。首版快照摘要沿用[现有快照摘要编码与测试向量](../../contracts/v1alpha5/dependency-snapshot-digest-vectors.json)，通过不同清单类型选择契约。

| Export kind | 内容 |
|---|---|
| CallableCatalog | 命名空间下的 Callable ID、kind、Effect、输入/输出及相关错误、结果 Schema 引用 |
| Schema | 固定 dialect、规范 Schema ID 和内容路径；外部引用只能进入已锁定快照 |
| CapabilityCatalog | `capabilityId`、所属 Interface Package、精确 Callable 成员；独立 export kind，可与 CallableCatalog 同包发布 |
| IngressCatalog | 保留已发布 Ingress Operation 契约，固定 Operation、协议版本、标准化事件、响应及引用的 Schema；供 Trigger / Wait 和响应边界链接 |

IngressCatalog 的导出身份由 `package.json` 精确绑定到载荷：`apiVersion`/`kind` 一致、导出 ID 恰好等于一个真实 `spec.operations[].id`、导出 path 等于实际 catalog 文件、Operation ID 唯一、`metadata.owner` 等于包 `publisher`、每个声明的 operation 在同一路径都有精确导出（无未导出 operation、无 id/path 漂移），且 `artifactBindings`/`schemaBindings` 指向精确的包或依赖快照路径与已声明 Schema 身份。文件存在或 `apiVersion` 正确本身不构成合法导出。

跨包 Capability 成员使用声明过的依赖别名和精确 Callable ID。Subject Context、Observation、错误/结果、声明中引用的 Receipt / Reconcile 类型，以及 Wait 输出和响应映射所需类型均进入 Schema 闭包。保证材料可缺失；源码方面必需的声明、明确的 Effect、接口符号和类型仍须通过静态检查。发布过程可组合多个源 export，Resolver 与 Compiler 交接时每个发布版本只有一个权威快照。

### 3.2 依赖锁与闭包

`dependencies.lock` 由 Resolver 生成。首版字段为版本身份、根别名指向的包键、`packages[]`。包键为 `{packageId,version,snapshotDigest}`，每项另记录每条直接依赖的本地别名和目标包键。Compiler 接收锁和精确快照，按 manifest 核验格式、声明、路径、摘要、依赖约束与边，并遍历至固定点。拒绝缺包、多余输入、歧义别名、冲突 Schema ID、同一 ID/版本异摘要、首版包依赖环和外部未锁定引用。Schema 内部递归继续遵循冻结语言支持的规则，解析时按规范 URI 和片段去重。首版每个包 ID 在同一图中只允许一个版本，遇到冲突由 Resolver 显式报错；禁止 Compiler 隐式选版本、下载或回退。

Artifact 完整嵌入规范化锁、已绑定导出与闭包摘要。快照本身由交付方按锁提供；离线 Verify 先验证产物清单与文件，再以原始构建请求重验接口闭包和两类 Requirement 覆盖，并逐字节比对全部产物文件。仅可访问 Artifact 时可做完整性校验。Host 最终准入必须完成闭包验证和适用的冻结保证检查。锁文件不含下载地址、凭据或部署数据。

### 3.3 Runtime Provider Package 与绑定

Provider manifest 提供 Provider ID / 版本、实现摘要、运行时协议版本、支持的精确 Interface Package 键、Callable → 实现映射及兼容证明引用。Provider 版本独立演进；范围声明用于发现候选，Host 只接受枚举的接口摘要和与其绑定的兼容证明。接口先发布、多个实现支持同一接口、只更新 Provider 版本均可；只要接口键和语义未变，Artifact 可复用，仍须重新 Admission。

部署清单由 Host 受信配置管理，将 Provider 实现摘要和 Callable 映射到具有可验证身份的实例、协议、endpoint 和凭据引用。每次 Admission 固定选择；无匹配或选择歧义即拒绝。切换实现、部署配置或目标环境时重新 Admission。网络 endpoint、凭据与私有上下文只驻留在 Host / Provider 边界。

### 3.4 Assurance Attestation 与 Deployment Grant

Assurance Attestation 是外部信任产物，独立版本化和签发，**不是 Compiler 输入**。最小绑定为接口包键、Callable ID、相关 Schema 闭包摘要、义务类型及策略范围；实现相关声明另绑定 Provider 实现摘要，结论关联目标环境时另绑定环境和适用期。独立 Evidence Authority 可签发其被委派的 Evidence 声明；Owner 的 Commit Proof / Reconcile 保证须由相应 Owner 或其经验证授权的签发者承担。结构正确只意味着可供 Host 验证。实跑 Evidence 另按 Invocation、原 Operation、主体、来源和新鲜度验证。HTTP 成功响应不构成 Commit Proof。Attestation 与静态 `EvidenceDescriptor` 必须分开表述：后者是 Interface Package 的正式导出声明的接口契约，由 Dependency Lock 固定字节；前者是签发者在 Compiler 之外给出的保证结论。

Deployment Grant 单独签名，至少含 `grantId`、签发者、环境及策略版本、主体、Artifact 摘要、允许的接口能力和 Callable、Provider 实现或实例范围、资源 Scope、有效期和撤销引用。Host 将 Skill 请求、Catalog 成员、Grant 范围和 Invocation 上下文求交；任一必需绑定缺失、过期、撤销、签发权限不明或范围不匹配即拒绝。Catalog 不携带授权数据，Resolver 不把 Grant 交给 Compiler。

## 4. Development Build 契约

Compiler 首个 Draft PR 的具体入口、复验方式与样例见[实施契约](compiler-development-build-pilot.zh-CN.md)。Evidence / Assurance 的职责拆分见[职责审计](compiler-evidence-assurance-responsibility-audit.zh-CN.md)。试验实现可独立于公开契约定版推进。

提案使用独立契约族，首版以 `connectionagent.io/<kind>/v1alpha1` 标识，其中 Build 相关 kind 为 `compiler-development-build-request`、`compiler-development-build-result`、`compiler-development-artifact`，输入 kind 为 `interface-package`、`interface-dependency-lock`。Evidence Descriptor 是 Interface Package 的正式导出（`EvidenceDescriptor`），不是独立 Assurance 输入。Provider 和 Grant 自有 `runtime-provider-package` 与 `deployment-grant` 版本。产品版本、语言目标、线格式版本和包发布版本独立。调用方显式选择新入口与契约版本；任何未知版本或不匹配的入口拒绝处理。

| 契约 | 提案字段与规则 |
|---|---|
| BuildRequest | `apiVersion`、`languageTarget=connectionagent.io/v1alpha5`、`profile=DEVELOPMENT`、源码文件、Interface 快照（含 EvidenceDescriptor 与 Ingress/Schema 导出）、Dependency Lock。输入只有这三类，没有 `assurance[]`，也没有可选的 Assurance Export |
| BuildResult | `apiVersion`、Source / Interface Link / Artifact 状态、结构化诊断、两类 Requirement 记录、Artifact 文件清单及摘要；失败原子返回无产物 |
| Artifact | `apiVersion`、语言目标、生成器版本、`profile=DEVELOPMENT`、源码摘要、规范化锁、Procedure、Callable / Ingress 链接、两类 Requirement 记录、文件摘要；执行状态固定 `NOT_ADMITTED` |

~~~mermaid
flowchart TD
    IN["Source + lock + exact snapshots (including EvidenceDescriptor exports)"] --> S["Compile and validate v1alpha5 source"]
    S --> L["Link Callable, Schema, Capability, EvidenceDescriptor and Ingress closure"]
    L --> A["Validate statically decidable Evidence, Mutation and Reconcile semantics"]
    A --> O["Classify requirements: host-admission vs runtime-execution"]
    O --> B["Assemble and verify deterministic Development Artifact"]
    B --> DONE["LINKED / BUILT / NOT_ADMITTED"]
    S -->|Invalid source| X["Diagnostics / no Artifact"]
    L -->|Incomplete or inconsistent interface| X
    A -->|Malformed, contradictory or missing static contract| X
    B -->|Assembly or verification failure| X
~~~

Evidence 语义必须在 Compiler 内完整执行：声明与引用解析、Descriptor 结构、Producer/Consumer 关系、Predicate 参数绑定、SourcePath、数据流、Schema 类型、Independence 的静态可判定部分，以及 Mutation/Commit Proof/Reconcile 的静态约束。Descriptor 缺失、重复、结构非法或与引用不符，按冻结语言规则拒绝 Build；不得以任何「未决」状态替代静态检查。

只有需要外部权威材料才能判定的事实在静态检查通过后记录为 Requirement，并分成两类、写成两个文件：

| 类别 | 覆盖范围 | 产物文件 | 固定状态 |
|---|---|---|---|
| **Host Admission Requirement** | 派发前可验证、且 Compiler 能从源码/接口精确定位的**外部事实**：`OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST` | `host-admission-requirements.json` | `HOST_VERIFICATION_REQUIRED` |
| **Runtime Execution Requirement** | 只能在**执行期**发生的真实结果：`COMMIT_PROOF`（实际 Mutation 提交与证明核验）、`RECONCILE`（`UNKNOWN` 后对精确原尝试的恢复与实际结论） | `runtime-execution-requirements.json` | `RUNTIME_EXECUTION_REQUIRED` |

Deployment Grant、Provider Binding 与当前 Invocation 授权同样属于 Host Admission 在派发时校验的外部事实，但 Compiler 无法从源码定位它们，因此**不写成 Requirement 条目、不进入 BuildRequest、不进入任一 Requirement 文件**；Host 必须在准入时以独立信任根补齐并重新校验。

`COMMIT_PROOF` 与 `RECONCILE` 不是可在 Build 或派发前满足的 Host Admission Requirement；它们的真实提交证据与恢复结论永不进入已编译产物，Compiler 也不预测或伪造。Runtime / Host 必须分别区分「运行时机制支持检查」（Contract、Schema、path、READ Callable、路由与重试结构是否就位）与「实际执行结果」（这一次到底提交/恢复了什么）。

每条记录含确定性 `requirementId`（`sha256:` + `CA-SKILL-DEVELOPMENT-HOST-REQUIREMENT\0v1\0` 域与类型化键 JCS 字节的 SHA-256）、`kind`、`location{skill,path,node,stage,selector}`、`subjects[{key{packageId,version,snapshotDigest},id,sourcePath,schemaIds}]` 与 `status`。`EVIDENCE_INDEPENDENCE_TRUST` 另带 `independence{groupIndex,members[{dependencyAlias,key,exportId,sourcePath,domain}],domainPairs}`，**一个 Independence Group 一条记录**，来源精确定位到 `selector=/predicates/<predicate>/evidence/independence/<index>`，不使用 `env.roots[0]` 或单一全局布尔值。同一个 Descriptor 同时声明 `sourceIdentity` 与 `factIdentity` 时，两条记录各自独立生成、互不覆盖；输入顺序不得改变结果。静态结构非法是 Build 失败，只有外部授权缺失才是 Host Requirement。

这些记录是 Host 与 Runtime 必须补齐并验证的外部事实，不是 Compiler 的通过结论，也不表示 Build 失败。Compiler 不内建 Assurance Engine、签名验证、信任根或运行时 Evidence 检查；Owner 的真实身份、Assurance 签发者可信性、真实 Evidence 来源/Freshness/Correlation、实际 Commit Proof 和 Invocation 资格均属于 Owner、Assurance Authority、Host / Runtime。

产物采用 JCS JSON、排序路径及稳定顺序；对 manifest 之外的每个产物文件逐字节求摘要，Artifact 身份由含版本域的规范 manifest 摘要给出，manifest 不包含自身摘要。相同源码字节、快照与生成器版本应产生同一产物。Verify 检查两个 Requirement 文件的规范化 JCS、排序、逐条重新推导的 `requirementId`、kind 所属类别、`status` 与覆盖完整，并以原始请求重建全部产物逐字节比对。产物禁止嵌入 endpoint、凭据、主体私有数据、Grant 或可执行授权令牌。

## 5. Host Admission、Qualification 与运行生命周期

~~~mermaid
flowchart TD
    A["Immutable Development Artifact + locked snapshots"] --> V["Verify format, provenance, closure and both requirement files"]
    P["Verified Provider and instance binding"] --> G["Evaluate target policy and authenticated Invocation"]
    D["Deployment Grant and revocation state"] --> G
    Q["Verified Assurance Attestation and optional qualification record"] --> G
    V --> G
    G --> J{"Profile and all Host Admission Requirements satisfied?"}
    J -->|No| R["Reject before dispatch"]
    J -->|Yes| S["Invocation-scoped Host admission"]
    S --> RUN["Existing Procedure Runtime"]
    RUN --> CHECK["Revalidate authorization before dispatch / resume"]
    CHECK -->|Valid| DIS["Dispatch to pinned Provider"]
    CHECK -->|Invalid| HOLD["Block dispatch and retain settlement state"]
    DIS --> CP["Runtime execution: actual COMMIT_PROOF / RECONCILE result"]
    CP --> SET["Verify correlated result / Evidence and settle"]
    SET -->|Unknown outcome| HOLD
    HOLD --> REC["Authorized exact-original RECONCILE"]
~~~

- **开发或集成。** 仅显式配置 opt-in 的隔离 Host 可接收 `DEVELOPMENT`，必须验证实际测试 Grant、身份与 Provider Binding；测试信任根或替代保证由目标策略明确规定。需要真实 Mutation 的路径必须满足相应 Owner、结算和重试约束。任何未满足的强制 Host Admission Requirement 均阻断相关 Invocation；测试设施和模拟结果仅用于其批准范围。
- **生产。** 生产 Host 拒绝 `DEVELOPMENT`，即使外部材料后来补齐。生产入口保留冻结基线，未来若要支持新格式须定义专门的生产产物契约与准入策略并重新构建。Qualification 是外部签发、绑定 Artifact 摘要、Provider 实现、环境策略及有效期的保证结论；它不能授予 Scope 或替代每次 Admission。
- **准入生命周期。** Host 的决定绑定 Invocation、Artifact、Provider、Grant、策略版本和有效期。派发和恢复前复核身份、Scope 与撤销；策略改变、材料失效或绑定变化后重新准入。阻断新派发不抹除已有副作用或待结算记录。出现 `UNKNOWN` 时保留原尝试标识并执行经授权的 Reconcile；无可信结论时阻止盲目重试。
- **执行期要求。** `COMMIT_PROOF` 与 `RECONCILE` 属于 Runtime Execution Requirement：派发前只能检查机制是否就位（Contract、Schema、path、READ Callable、路由与重试结构），真实提交与恢复结论只在执行期产生，永不回写已编译产物。
- **不可变性。** Evidence、部署绑定和 Grant 通过外部精确引用补齐，不改写产物或其 Requirement 状态。接口、源码或产物契约变化后重新 Build。Host 的已准入、拒绝和结算状态归 Invocation 所有。

## 6. 职责矩阵

| 责任 | Compiler | Interface Package | Host Admission | Runtime / Provider | Owner / Assurance Authority |
|---|---|---|---|---|---|
| 语言静态校验与确定性产物 | **负责**（唯一实现） | 提供精确快照 | 复核产物完整性 | 不参与 | 不参与 |
| EvidenceDescriptor 静态结构、Producer/路径、Schema 闭包 | **负责** | **以正式导出声明并固定字节** | 不重复解析语言规则 | 不参与 | 不参与 |
| Independence 组静态结构与逐组 Requirement | **负责**（一组一条） | 声明成员与域 | 验证域授权与域对独立性 | 不参与 | 授权 IndependentDomains 与域对 |
| Owner 真实身份与签发权限 | 只记录包声明身份 | 声明 `publisher` / Owner 字段 | **验证** | 不参与 | 签发并承担 |
| sourceIdentity / factIdentity 的真实来源与授权 | 只检查声明形状 | 声明 | **验证** domain / fact 信任 | 提供运行记录 | 授权 CommonFact / 域 |
| Assurance Attestation | **绝不接收** | 不包含 | **验证签名与范围** | 不参与 | **签发** |
| Deployment Grant | 不可见 | 不可见 | **验证环境/主体/Scope/撤销** | 不参与 | 不参与（部署方签发） |
| Provider Binding / 实例选择 | 不参与 | 只声明兼容接口键 | **固定本次绑定** | 接受派发 | 不参与 |
| Invocation 授权 | 不参与 | 不参与 | **每次派发与恢复前复核** | 不参与 | 不参与 |
| 实际 Commit Proof | 只校验静态契约并记录 `COMMIT_PROOF` Runtime Requirement | 不参与 | 派发前只检查机制是否就位 | **产生并核验真实证明** | 承担 Owner 保证 |
| 实际 Reconcile 结论 | 只校验静态契约并记录 `RECONCILE` Runtime Requirement | 不参与 | 派发前只检查机制是否就位 | **执行 exact-original 恢复并给出结论** | 授权恢复 |
| 执行资格 / Qualification | 不声明 | 不声明 | 每次准入决定 | 不声明 | 外部签发结论 |

**一句话边界：** Compiler 负责「接口与静态语义是否成立、外部与执行期还缺什么」，Host 负责「派发前的那些外部事实现在成立吗」，Runtime / Provider 负责「执行期究竟发生了什么」，Owner / Assurance Authority 负责「谁有权作出这些声明」。

## 7. MAS 对接与冻结边界

先从 ca-client-tools 的 13 个 Operation 与 25 个候选 Schema 整理 Callable 和 Capability Catalog，补足 `ca-context` Subject Context 及实际引用的 Ingress / Schema 闭包。完整候选包可开启新 Development Build 的离线验收；独立 Provider 与测试 Grant 通过 Host 验证后才进入真实派发。Mock AMF 是 5GC 测试对端；**所有 H1/H2/H3 网络集成/E2E 测试强制 SCP**。H1 为 SCP Mock AMF 完整双向 SBI；H2 使用同一 SCP 拓扑测试 NRF Discovery、Target-apiRoot、Binding、SCP 故障恢复和多 UE；H3 是 SCP + 真实 AMF/UERANSIM。OpenAMSM → Host 是受信内部 Namf Relay，Host → SCP → AMF 才是正式 Namf 通信。Host → Owner 使用固定私有绑定；UDM/PCF 等其他 NF 仍使用本地 SCP/NRF。不存在无 SCP 或 Direct Mock Relay 的正式集成 Profile。Mock AMF 响应不生成 Owner Commit Proof，也不解决复合 SSM 的 30 秒上限和缺少持久化 Owner continuation。已派发操作的 `OWNER_DEFERRED` 与进入节点的 `INGRESS_WAIT` 继续分开。阶段对应 [ca-agent-demo #217 的 G0–G5](https://github.com/odango-chan/ca-agent-demo/pull/217)。

冻结的 [v1alpha5 Artifact 契约](compiler-artifact-v1alpha5.md) 及 [v2.0.2 机器契约](../../contracts/v1alpha5/README.md) 保持现有结果含义。新产物不得声明冻结契约的 `FULL_LANGUAGE_VALIDATION` 状态；须列出已执行检查与因缺失保证材料而延期的检查；延期检查按原规则完成后方可执行。现有 fixture、严格构建和生产接入继续按各自已发布契约验收。本文仅供评审，当前没有新的构建模式或 MAS qualification；MAS G0 / G1 仍为 **NOT_RUN**，本文不构成 MAS 通过结论。

## 8. 视觉评审基线

本轮工具链评审原有三张独立可读的架构图，另增加一张 SCP-only AMF–SMF SBI 网络架构图；重绘工作要求见 [Archify 专项简报](archify-redraw-brief-scp-only-20261119.zh-CN.md)。图中案例统一为 **session-demo**；跨图摘要标记统一使用：

- MAS 源：\`sha256:…mas\`
- Context Interface：\`sha256:…ctx\`
- Linked Artifact：\`sha256:…artifact\`
- Runtime Provider：\`sha256:…provider\`

这四个摘要在图中是追踪占位符；实际构建记录必须写入真实摘要，不得把省略号当成摘要值。

### 8.1 工具链全景

全景图使用三条泳道：

1. **发布与依赖准备**：Interface Package 与 Runtime Provider Package 分别发布；Capability Catalog 属于 Interface Package 的导出内容；Deployment Grant 由部署方独立签发并交给 Host。
2. **离线 Compiler**：只接收 Skill 源、精确 Interface snapshots 和 dependencies.lock；不接收 Assurance Export，也不访问 Registry、不部署 Provider、不签发 Grant。
3. **Host / Runtime**：验证 Artifact、Provider Binding、Grant、Assurance Attestation 和当前 Invocation；通过 Admission 后才允许派发；Commit Proof 与 Reconcile 属执行期。

### 8.2 离线构建

离线构建图按四个阶段展示，每个阶段明确标注输入、处理和输出：

| 阶段 | 输入 | 处理 | 输出 |
|---|---|---|---|
| Compile | SKILL.md、Procedure / Predicate source | 语言解析与静态校验 | 编译后的 Skill IR / 诊断 |
| Resolve Interface Link | dependencies.lock、Interface Package snapshots | Callable、Schema、Capability、Ingress 闭包解析 | 精确链接记录 |
| Classify requirements | 锁定的 EvidenceDescriptor 导出、静态声明、Mutation/Reconcile 静态契约 | 执行完整 Evidence 静态语义，并把外部事实与执行期结果分成两类记录 | host-admission-requirements.json + runtime-execution-requirements.json |
| Assemble Artifact | 源摘要、锁、链接记录、两类 Requirement 记录 | 规范化、完整性验证、确定性打包 | manifest + Linked Skill Artifact |

错误边界：

- 缺少或冲突的 Interface / Schema / Callable / EvidenceDescriptor，或 Ingress 导出身份不成立：Link 失败，**不得产生 Artifact**。
- 接口与静态 Descriptor 闭合但缺运行时 Evidence、Assurance Attestation 或 Deployment Grant：仍可产生 Development Artifact；派发前外部事实记为 Host Admission Requirement，执行期真实结果记为 Runtime Execution Requirement，执行状态为 \`NOT_ADMITTED\`。
- Deployment Grant 不进入离线 Build 输入；它属于部署方与 Host Admission 的交接材料。

### 8.3 Host 准入与执行

准入图应明确五类材料汇入 Host Admission：

1. Linked Skill Artifact 与锁定依赖闭包；
2. Runtime Provider Package / 已部署实例绑定（Provider Binding）；
3. Capability Catalog 中的接口成员关系；
4. Owner / Evidence 权威材料：`OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST`，以及可选的 Assurance Attestation；
5. Deployment Grant 与当前 Invocation 授权。

隔离 Host 只有在显式 opt-in 且测试策略允许时，才可接收 Development Artifact；生产 Host 必须拒收 Development Artifact。通过准入后，派发、恢复前复核、结算仍须按原始 Operation occurrence 执行。`COMMIT_PROOF` 与 `RECONCILE` 属 Runtime Execution Requirement：派发前只检查机制是否就位，真实提交与恢复结论在执行期产生。UNKNOWN 只能进入经授权的 exact-original Reconcile，不得盲目重试。

## 9. 成图与可编辑源

本节语义已按 scp-only 简报重绘为四张正式图，可编辑源与离线 HTML 并列存放在仓库目录
[`docs/architecture/archify/`](../architecture/archify/05-appendix-minimal-examples.zh-CN.md)：

| 图 | 离线 HTML | 可编辑源 | 图表类型 |
|---|---|---|---|
| 01 工具链全景 | `01-toolchain-panorama.html` | `01-toolchain-panorama.architecture.json` | architecture |
| 02 离线编译与链接 | `02-offline-build.html` | `02-offline-build.workflow.json` | workflow |
| 03 Host Admission 与执行 | `03-host-admission.html` | `03-host-admission.architecture.json` | architecture |
| 04 SCP-only AMF–SMF SBI | `04-scp-amf-smf.html` | `04-scp-amf-smf.architecture.json` | architecture |

字段来源、最小示例、跨图摘要追踪表与逐条一致性核对见
[附页 05](../architecture/archify/05-appendix-minimal-examples.zh-CN.md)。图中统一案例为 `session-demo`，
`sha256:…mas` / `…ctx` / `…artifact` / `…provider` 只是追踪占位符。

这些图是设计评审材料，不是实现完成或 E2E 通过的证据：Development Build、Host 原生 SBI 入口、
Owner-private 原始 N1 输入、NRF 单一发布者与 Owner Recovery 均未实现或未验收，图中以虚线、
“未实现”标签或明确的 NOT_IMPLEMENTED 说明表达。不要在仓库文档中使用某台开发机的绝对 Windows 路径作为共享链接。

