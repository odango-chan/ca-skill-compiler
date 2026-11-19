# CA-SL 工具链架构：编译、接口链接与执行资格分离

> **状态：原则已通过评审；以下契约为提案，尚未冻结或实现。**
>
> 当前基线为冻结的 CA-SL v1alpha5（`connectionagent.io/v1alpha5`）和已发布的 Compiler v2.0.2。现有语言规则、BuildRequest / BuildResult / Artifact 契约及实现行为以冻结基线为准。迁移步骤见[演进路线](compiler-evolution-roadmap-vnext.zh-CN.md)，Host 和 MAS E2E 边界见 [ca-agent-demo #217](https://github.com/odango-chan/ca-agent-demo/pull/217)。

## 1. 已批准的原则

1. Capability Catalog 定义能力身份与 Callable 成员关系；Deployment Grant 由部署方签发，授权具体环境中的使用。
2. 接口闭合时，Evidence 缺失仍可完成 Development Build。产物保留显式义务，在 Host Admission 成功前禁止执行。
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
    EA["Owner / Evidence Authority"] --> AE["Assurance Export"]
    AE -->|Optional static descriptors| C
    AE -->|Verified claims| H["Host Admission"]
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
| Owner / Evidence Authority | 以 Host 认可的权限对 Operation 保证、Evidence 规则或资格结论签发声明 |
| Deployment Authority | 发行和撤销环境及主体范围内的 Deployment Grant |
| Host | 依据独立配置的信任根、环境策略及当前 Invocation 验证以上材料；默认拒绝，只有授权路径可派发 |

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

跨包 Capability 成员使用声明过的依赖别名和精确 Callable ID。Subject Context、Observation、错误/结果、声明中引用的 Receipt / Reconcile 类型，以及 Wait 输出和响应映射所需类型均进入 Schema 闭包。保证材料可缺失；源码方面必需的声明、明确的 Effect、接口符号和类型仍须通过静态检查。发布过程可组合多个源 export，Resolver 与 Compiler 交接时每个发布版本只有一个权威快照。

### 3.2 依赖锁与闭包

`dependencies.lock` 由 Resolver 生成。首版字段为版本身份、根别名指向的包键、`packages[]`。包键为 `{packageId,version,snapshotDigest}`，每项另记录每条直接依赖的本地别名和目标包键。Compiler 接收锁和精确快照，按 manifest 核验格式、声明、路径、摘要、依赖约束与边，并遍历至固定点。拒绝缺包、多余输入、歧义别名、冲突 Schema ID、同一 ID/版本异摘要、首版包依赖环和外部未锁定引用。Schema 内部递归继续遵循冻结语言支持的规则，解析时按规范 URI 和片段去重。首版每个包 ID 在同一图中只允许一个版本，遇到冲突由 Resolver 显式报错；禁止 Compiler 隐式选版本、下载或回退。

Artifact 完整嵌入规范化锁、已绑定导出与闭包摘要。快照本身由交付方按锁提供；离线 Verify 先验证产物清单与文件，再以原始构建请求重验接口闭包和义务覆盖。仅可访问 Artifact 时可做完整性校验。Host 最终准入必须完成闭包验证和适用的冻结保证检查。锁文件不含下载地址、凭据或部署数据。

### 3.3 Runtime Provider Package 与绑定

Provider manifest 提供 Provider ID / 版本、实现摘要、运行时协议版本、支持的精确 Interface Package 键、Callable → 实现映射及兼容证明引用。Provider 版本独立演进；范围声明用于发现候选，Host 只接受枚举的接口摘要和与其绑定的兼容证明。接口先发布、多个实现支持同一接口、只更新 Provider 版本均可；只要接口键和语义未变，Artifact 可复用，仍须重新 Admission。

部署清单由 Host 受信配置管理，将 Provider 实现摘要和 Callable 映射到具有可验证身份的实例、协议、endpoint 和凭据引用。每次 Admission 固定选择；无匹配或选择歧义即拒绝。切换实现、部署配置或目标环境时重新 Admission。网络 endpoint、凭据与私有上下文只驻留在 Host / Provider 边界。

### 3.4 Assurance Export 与 Deployment Grant

Assurance Export 独立版本化和签发。最小绑定为接口包键、Callable ID、相关 Schema 闭包摘要、义务类型及策略范围；实现相关声明另绑定 Provider 实现摘要，结论关联目标环境时另绑定环境和适用期。独立 Evidence Authority 可签发其被委派的 Evidence 声明；Owner 的 Commit Proof / Reconcile 保证须由相应 Owner 或其经验证授权的签发者承担。结构正确只意味着可供 Host 验证。实跑 Evidence 另按 Invocation、原 Operation、主体、来源和新鲜度验证。HTTP 成功响应不构成 Commit Proof。

Deployment Grant 单独签名，至少含 `grantId`、签发者、环境及策略版本、主体、Artifact 摘要、允许的接口能力和 Callable、Provider 实现或实例范围、资源 Scope、有效期和撤销引用。Host 将 Skill 请求、Catalog 成员、Grant 范围和 Invocation 上下文求交；任一必需绑定缺失、过期、撤销、签发权限不明或范围不匹配即拒绝。Catalog 不携带授权数据，Resolver 不把 Grant 交给 Compiler。

## 4. Development Build 契约

Compiler 首个 Draft PR 的具体入口、复验方式与样例见[实施契约](compiler-development-build-pilot.zh-CN.md)。试验实现可独立于公开契约定版推进。

提案使用独立契约族，首版以 `connectionagent.io/<kind>/v1alpha1` 标识，其中 Build 相关 kind 为 `compiler-development-build-request`、`compiler-development-build-result`、`compiler-development-artifact`，输入 kind 为 `interface-package`、`interface-dependency-lock`、`assurance-export`。Provider 和 Grant 自有 `runtime-provider-package` 与 `deployment-grant` 版本。产品版本、语言目标、线格式版本和包发布版本独立。调用方显式选择新入口与契约版本；任何未知版本或不匹配的入口拒绝处理。

| 契约 | 提案字段与规则 |
|---|---|
| BuildRequest | `apiVersion`、`languageTarget=connectionagent.io/v1alpha5`、`profile=DEVELOPMENT`、源码文件、Interface 快照、锁、可选 Assurance Export |
| BuildResult | `apiVersion`、Source / Interface Link / Artifact 状态、结构化诊断、静态保证记录、Artifact 文件清单及摘要；失败原子返回无产物 |
| Artifact | `apiVersion`、语言目标、生成器版本、`profile=DEVELOPMENT`、源码摘要、规范化锁、Procedure、Callable / Ingress 链接、义务记录、文件摘要；执行状态固定 `NOT_ADMITTED` |

~~~mermaid
flowchart TD
    IN["Source + lock + exact snapshots + optional Assurance Export"] --> S["Compile and validate v1alpha5 source"]
    S --> L["Link Callable, Schema, Capability and Ingress closure"]
    L --> A["Validate static contracts and supplied Assurance descriptors"]
    A --> O["Emit complete obligation list"]
    O --> B["Assemble and verify deterministic Development Artifact"]
    B --> DONE["LINKED / BUILT / NOT_ADMITTED"]
    S -->|Invalid source| X["Diagnostics / no Artifact"]
    L -->|Incomplete or inconsistent interface| X
    A -->|Malformed, contradictory or wrong target| X
    B -->|Assembly or verification failure| X
~~~

义务记录从静态链接后的 Skill 声明及已知的保证适用规则生成。每条记录含确定性 `obligationId`（由类型、来源位置与绑定身份计算）、`kind`（如 `OWNER_TRUST`、`COMMIT_PROOF`、`RECONCILE`、`EVIDENCE_GUARANTEE`）、稳定的 Node/Stage 或 Skill 范围、包键、适用的 Callable 与 Schema 摘要、所需保证及 `status`。缺失材料为 `UNRESOLVED` 并带原因；材料静态匹配为 `DESCRIPTOR_PRESENT` 并附材料摘要。两者都需 Host 核验。必需语法声明缺失、静态约束不成立或已提供描述损坏是错误。冻结规范 §17.2 / G.3 要求的 Evidence 描述符链接和逐路径充分性检查，在材料缺失时明确记为待验证义务（含对应规则及消费路径）。Compiler 仍执行可判定的检查；提供材料后若违反绑定、独立性、Scope、Correlation 或 Freshness 约束，即失败。Host 在派发前须以受信材料逐项完成适用的冻结规则检查，并将校验结果绑定到 Artifact 和目标 Provider；隔离策略只能改变受信权威与额外环境要求，不能免除冻结语言要求。Provider Binding、环境授权及 Invocation 资格单独记录为 `NOT_CHECKED`；Build 不推断部署状态。

产物采用 JCS JSON、排序路径及稳定顺序；对 manifest 之外的每个产物文件逐字节求摘要，Artifact 身份由含版本域的规范 manifest 摘要给出，manifest 不包含自身摘要。相同源码字节、快照、描述与生成器版本应产生同一产物。Verify 还检查必需义务与链接记录一致、覆盖完整。产物禁止嵌入 endpoint、凭据、主体私有数据、Grant 或可执行授权令牌。

## 5. Host Admission、Qualification 与运行生命周期

~~~mermaid
flowchart TD
    A["Immutable Development Artifact + locked snapshots"] --> V["Verify format, provenance, closure and obligations"]
    P["Verified Provider and instance binding"] --> G["Evaluate target policy and authenticated Invocation"]
    D["Deployment Grant and revocation state"] --> G
    Q["Verified Assurance and optional qualification record"] --> G
    V --> G
    G --> J{"Profile and all applicable obligations satisfied?"}
    J -->|No| R["Reject before dispatch"]
    J -->|Yes| S["Invocation-scoped Host admission"]
    S --> RUN["Existing Procedure Runtime"]
    RUN --> CHECK["Revalidate authorization before dispatch / resume"]
    CHECK -->|Valid| DIS["Dispatch to pinned Provider"]
    CHECK -->|Invalid| HOLD["Block dispatch and retain settlement state"]
    DIS --> SET["Verify correlated result / Evidence and settle"]
    SET -->|Unknown outcome| HOLD
~~~

- **开发或集成。** 仅显式配置 opt-in 的隔离 Host 可接收 `DEVELOPMENT`，必须验证实际测试 Grant、身份与 Provider Binding；测试信任根或替代保证由目标策略明确规定。需要真实 Mutation 的路径必须满足相应 Owner、结算和重试约束。任何未满足的强制义务均阻断相关 Invocation；测试设施和模拟结果仅用于其批准范围。
- **生产。** 生产 Host 拒绝 `DEVELOPMENT`，即使外部材料后来补齐。生产入口保留冻结基线，未来若要支持新格式须定义专门的生产产物契约与准入策略并重新构建。Qualification 是外部签发、绑定 Artifact 摘要、Provider 实现、环境策略及有效期的保证结论；它不能授予 Scope 或替代每次 Admission。
- **准入生命周期。** Host 的决定绑定 Invocation、Artifact、Provider、Grant、策略版本和有效期。派发和恢复前复核身份、Scope 与撤销；策略改变、材料失效或绑定变化后重新准入。阻断新派发不抹除已有副作用或待结算记录。出现 `UNKNOWN` 时保留原尝试标识并执行经授权的 Reconcile；无可信结论时阻止盲目重试。
- **不可变性。** Evidence、部署绑定和 Grant 通过外部精确引用补齐，不改写产物或其义务状态。接口、源码或产物契约变化后重新 Build。Host 的已准入、拒绝和结算状态归 Invocation 所有。

## 6. MAS 对接与冻结边界

先从 ca-client-tools 的 13 个 Operation 与 25 个候选 Schema 整理 Callable 和 Capability Catalog，补足 `ca-context` Subject Context 及实际引用的 Ingress / Schema 闭包。完整候选包可开启新 Development Build 的离线验收；独立 Provider 与测试 Grant 通过 Host 验证后才进入真实派发。Mock AMF 是 5GC 测试对端，完整 SBI 用例经本地 NRF/SCP 到真实 SMF handler。其响应不生成 Owner Commit Proof，也不解决复合 SSM 的 30 秒上限和缺少持久化 Owner continuation。已派发操作的 `OWNER_DEFERRED` 与进入节点的 `INGRESS_WAIT` 继续分开。阶段对应 [ca-agent-demo #217 的 G0–G5](https://github.com/odango-chan/ca-agent-demo/pull/217)。

冻结的 [v1alpha5 Artifact 契约](compiler-artifact-v1alpha5.md) 及 [v2.0.2 机器契约](../../contracts/v1alpha5/README.md) 保持现有结果含义。新产物不得声明冻结契约的 `FULL_LANGUAGE_VALIDATION` 状态；须列出已执行检查与因缺失保证材料而延期的检查；延期检查按原规则完成后方可执行。现有 fixture、严格构建和生产接入继续按各自已发布契约验收。本文仅供评审，当前没有新的构建模式或 MAS qualification。
