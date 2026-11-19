# CA-SL 工具链架构：编译、接口链接与执行资格分离

> **状态：DESIGN PROPOSAL / NON-NORMATIVE / NOT IMPLEMENTED**
>
> **当前基线：** CA-SL v1alpha5（FROZEN）；CA Skill Compiler v2.0.2（RELEASED）。
>
> 本文定义下一代工具链的演进方向，不修改 v1alpha5 语言语义、Compiler v2.0.2 的 BuildRequest / BuildResult / Artifact Contract，也不声称新流程已经实现。若本文与当前冻结契约冲突，当前冻结契约优先；本文只作为下一版本设计输入。

## 1. 目标

将四个经常被混为一谈的问题分开：

1. **Compile：** Skill 源码是否符合 CA-SL 语言和静态语义。
2. **Link：** Skill 引用的 Callable、Schema、Capability 接口是否能从明确的依赖包中解析。
3. **Build：** 是否能生成确定、可复现、可验证的 Linked Skill Artifact。
4. **Runtime Admission：** 当前 Host、Provider、环境和授权是否允许执行这个 Artifact。

Evidence、Deployment 和真实 Provider 部署属于执行保证或运行环境，不应成为生成一个接口完整的开发 Artifact 的普遍前置条件。

设计目标是允许以下状态合法存在：

| 阶段 | 结果 |
|---|---|
| Source validation | PASS |
| Interface linking | PASS |
| Artifact build | PASS |
| Evidence assurance | PENDING / NOT PROVIDED |
| Deployment authorization | NOT CHECKED |
| Runtime execution | BLOCKED until Host Admission |

这不是“放宽检查后假装通过”。Artifact 必须明确保留未满足的保证要求；不能把缺失的 Evidence、Owner 证明或 Deployment 授权伪造成 PASS。

## 2. 责任边界

### 2.1 Compiler 负责什么

- 解析 CA-SL 源码并执行语言规定的静态检查。
- 消费调用方已经解析好的、不可变的依赖快照。
- 解析依赖别名、Callable、Schema、Capability 接口和数据类型。
- 检查 Procedure Graph、Predicate、参数绑定和可静态证明的约束。
- 生成确定性的 Linked Artifact，并验证产物自身完整性。
- 对可选提供的 Assurance 描述执行格式和静态一致性检查。
- 明确报告未解析的接口引用或尚未满足的外部保证要求。

### 2.2 Compiler 不负责什么

- 不实现 Package Registry，不下载或发现依赖。
- 不负责依赖版本选择、包发布、发布签名或 Provider Release。
- 不生成或授予 Deployment Authorization。
- 不运行 Provider、Operation、Observation 或 Python Script。
- 不处理真实 SBI、HTTP/2、网络事件或 Temporal Workflow。
- 不替 Owner 证明 Mutation 已提交，不生成真实 Evidence。
- 不执行产品 Qualification，也不决定生产环境是否允许执行。

**Digest 的边界：** Package Publisher / Resolver 负责为发布的不可变包分配并发布版本和内容摘要；Build 调用方把精确快照及其摘要交给 Compiler。Compiler 可以重新计算并校验输入字节，也可以为自己生成的 Artifact 输出完整性摘要，但不承担包版本管理、包摘要发布或 Registry 生命周期。

### 2.3 外部组件职责

| 组件 | 责任 |
|---|---|
| Provider 源码与 CI | 构建实际实现；从类型定义、注册表或 OpenAPI 等来源生成候选接口导出 |
| Package Publisher / Registry | 发布不可变 Interface Package、Runtime Provider Package 和可选 Assurance Package |
| Resolver（Studio / CI / 调用方） | 选择版本、取得依赖、生成 lock，并将精确快照传给 Compiler |
| Compiler | 只消费已解析输入，完成静态编译、接口链接和 Artifact Build |
| Host / Runtime | 验证 Artifact，绑定 Provider，检查授权和运行时保证，执行 Procedure |
| Owner / Evidence Authority | 对实际 Operation 效果、Commit Proof、Reconcile 和 Evidence Guarantee 负责 |
| Deployment Authority | 对某个环境中的 Capability Grant、Scope 和执行策略负责 |

## 3. Package 模型

下一代工具链应把“接口可链接”和“运行实现可执行”拆开。下面是逻辑模型，不是已经冻结的文件布局。

### 3.1 Interface Package

Interface Package 是 Compiler 完成接口链接所需的最小外部依赖。

至少包含：

- 稳定 Package ID、版本和不可变内容摘要；
- 对外导出的 Callable ID；
- Callable kind、参数和结果 Schema 引用；
- 完整的 Input / Output Schema 闭包；
- Callable 的声明 Effect（例如 READ / MUTATION）；
- Capability Catalog：Capability 身份及其 Callable 成员关系；
- 声明的 Package / Publisher 来源信息。

它描述“有什么接口、接口如何调用”，不证明接口实现已经部署，也不证明声明的行为已经获得生产信任。

Interface Package 的 Publisher 可以从 Go 类型、Python 类型、OpenAPI、注册表或手工维护的 Owner 契约生成接口候选。自动生成适合参数、字段、Schema 和导出清单；Effect、业务副作用和错误语义必须由接口 Owner 确认，不能只从函数签名推断。

### 3.2 Runtime Provider Package

Runtime Provider Package 是实际执行实现，独立于 Interface Package 发布。

它包含或引用：

- Provider 二进制、容器或运行时模块；
- Callable 到具体实现的映射；
- 运行时依赖与兼容性信息；
- 可部署的 Provider 版本及其内容摘要。

Interface Package 可以先发布，Runtime Provider Package 可以后续发布。Host 只有在运行时找到与 Artifact 中接口绑定兼容的实现后，才能派发 Operation。

### 3.3 Assurance Export

Assurance Export 是可选的保证材料，可以与 Interface Package 同包发布，也可以独立版本化，但必须精确绑定 Callable、Schema 和适用版本。

可能包含：

- Mutation Receipt / Commit Proof 的结构与绑定规则；
- 原始 Operation 的 Reconcile 契约；
- Owner Outcomes、无副作用证明和安全重试约束；
- Evidence Guarantee、Producer、Scope、Correlation、Freshness；
- 对应的信任来源和验证材料。

没有 Assurance Export 不应阻止接口链接和开发 Artifact 生成。但缺少必要保证时，Host 不得把未知结果解释为已提交，不得盲目重试，也不得宣称生产级 Evidence。

### 3.4 Capability 定义与 Deployment Grant 必须分开

这是下一代接口契约需要明确拆分的部分：

- **Capability Catalog**：定义 Capability ID、其所属接口包以及它允许映射的 Callable。它是接口身份和链接信息。
- **Deployment Grant / Capability Authorization**：决定某个部署环境是否授予这个 Capability、允许哪些 Scope、资源和操作。它是环境授权。

Skill 中声明 Capability，表示请求使用某项能力，不代表已经获得部署授权。

当前 v1alpha5 的 CapabilityContracts 同时承载成员关系和部署 Scope 等信息。下一代若要允许没有 Deployment 的 Interface Link，需要通过新的、版本化的契约把 Capability Catalog 与 Deployment Grant 分开；不得静默改变现有 v1alpha5 CapabilityContracts 的含义。

### 3.5 Owner 身份的两个层次

- Interface Package 可以声明来源命名空间和发布者主张，供开发阶段识别和链接。
- 经验证的 Owner 身份、签名和发布授权属于可信发布 / 运行资格要求。

开发链接不能把未验证的来源主张升级为可信 Owner；生产准入也不能只凭 Package ID 或仓库 URL 推断 Owner。

### 3.6 与 C / C++ 工具链的类比

这个类比可以帮助理解职责，但不是一一对应：

| C / C++ | CA-SL 工具链 | 说明 |
|---|---|---|
| .c / .cpp | SKILL.md、procedure.yaml、predicates.yaml 等 | 作者维护的业务源代码 |
| .h + linker symbol table / import library | Interface Package | 导出 Callable、Schema、Effect 和 Capability 接口 |
| gcc / clang compile | Compile | 解析语言、检查类型和静态语义 |
| ld / link.exe | Interface Link | 解析依赖别名、Callable 符号和 Schema 引用 |
| .so / DLL / shared library | Runtime Provider Package | 真实运行实现；可以是二进制、容器或远程服务 |
| linked executable | Linked Skill Artifact | 已链接的 Procedure 和接口绑定；仍可能需要运行时 Provider |
| runtime loader / process policy | Host Runtime Admission | 检查实现是否存在、授权是否有效、运行条件是否满足 |

因此，Interface Package 更像头文件加符号表，Runtime Provider 更像共享库，而 Linked Skill Artifact 更接近一个已链接的可执行程序或部署包，不是 .so 本身。

区别在于 CA-SL 的业务逻辑是声明式 Procedure Graph，Operation 由 Host 通过 Provider 边界分派；Artifact 不会把 Go / Python 实现静态打包进来。即使 Link + Build 成功，运行时 Provider 缺失或未授权，Artifact 仍然不能执行。

Evidence Guarantee 和 Deployment Grant 也不是“另一个头文件”：它们分别描述结果可信保证和目标环境授权，属于运行资格边界，而不是基础接口符号解析。

## 4. Compile → Interface Link → Build

### 4.1 依赖准备在 Compiler 之外

~~~mermaid
flowchart LR
    SRC["Provider source / types / OpenAPI"] --> CI["Provider CI"]
    CI --> IFACE["Interface Package"]
    CI --> RUNTIME["Runtime Provider Package"]
    OWNER["Owner review / assurance evidence"] --> ASSURE["Optional Assurance Export"]
    IFACE --> REG["Package Registry"]
    RUNTIME --> REG
    ASSURE --> REG
    REG --> RESOLVER["Resolver: Studio / CI / caller"]
    RESOLVER --> LOCK["Resolved snapshots + dependency lock"]
    LOCK --> COMPILER["skillc"]
    SKILL["Skill source"] --> COMPILER
    COMPILER --> ARTIFACT["Linked Skill Artifact"]
~~~

Registry / Resolver 可以自动选择版本、下载包和生成 lock，但这些能力不属于 Compiler。Compiler 接收的是调用方已经确定的输入快照；它不访问网络，也不自行选择依赖版本。

### 4.2 Compiler 内部流程

~~~mermaid
flowchart TD
    A["Skill source + resolved Interface snapshots"] --> B["Source admission and CA-SL validation"]
    B --> C{"Source valid?"}
    C -- "No" --> X["Diagnostics; no Artifact"]
    C -- "Yes" --> D["Build normalized AST / Procedure IR"]
    D --> E["Resolve dependency aliases and Package identities"]
    E --> F["Resolve Callable IDs and kinds"]
    F --> G["Resolve Input / Output Schema closure and type bindings"]
    G --> H["Resolve Capability Catalog identities and membership"]
    H --> I["Validate Procedure dataflow, routes and static semantics"]
    I --> J{"Interface closure complete?"}
    J -- "No" --> X
    J -- "Yes" --> K["Optional Assurance analysis"]
    K --> L["Record unresolved external obligations; never fabricate PASS"]
    L --> M["Deterministic Artifact assembly"]
    M --> N["Artifact self-verification"]
    N --> O["Linked Artifact"]
~~~

开发 Build 的关键规则：

- Callable、Schema、Capability Catalog 等接口身份必须真实存在于输入的 Interface Package 中。
- 缺失 Callable、Schema、Schema closure、Capability 定义或类型不匹配，仍然是 Link Error。
- 没有 Evidence Guarantee 时，可以将相应保证记录为未解析义务；不能将它标为已验证。
- 没有 Deployment Grant 时，Compiler 仍可生成 Artifact；Host 在目标环境执行 Admission。
- 已提供但格式错误、相互矛盾或绑定到错误 Callable 的 Assurance 数据仍必须报错，不能因为是 Development Build 而忽略损坏输入。
- 声明 Effect 是接口信息；真实 Commit Proof、Reconcile 和 Evidence 是更高层的保证。未知 Effect 不得被默认为 READ。

### 4.3 结果状态必须分层

建议使用不同维度表达结果，而不是一个含义模糊的 PASS：

| 维度 | 示例状态 | 解释 |
|---|---|---|
| Source | VALID / INVALID | 语言和源码检查结果 |
| Interface Link | LINKED / UNRESOLVED | Callable、Schema、Capability 接口闭合情况 |
| Artifact | BUILT / NOT_BUILT | 是否生成并验证产物 |
| Assurance | COMPLETE / PENDING / INVALID | 可选保证材料的静态闭合情况 |
| Deployment | NOT_CHECKED | Compiler 不检查目标环境授权 |
| Execution | NOT_ADMITTED | Compiler 不决定运行准入 |

LINKED + BUILT + ASSURANCE_PENDING 是合法的开发产物状态。它不是 FULLY_QUALIFIED，也不意味着可以执行。

## 5. Linked Artifact 的概念模型

下一代 Artifact 应至少能表达：

- Skill 源码身份和语言目标；
- 规范化的 Procedure Graph / IR；
- 精确的依赖闭包和 Package 版本引用；
- 每个 Operation / Observation 的 Callable 绑定；
- 参数、结果和 Schema 的解析结果；
- Capability 请求及其接口身份；
- 未满足的 Assurance 义务；
- Artifact 自身的确定性完整性信息。

示意布局：

~~~text
linked-skill/
  manifest.json
  skills/
    establish-pdu-session-mas/
      compiled/
        procedure.json
        linked-bindings.json
        assurance-obligations.json
~~~

以上文件名仅用于说明职责，不是当前或下一版本已冻结的 Artifact Contract。

Artifact 不应嵌入 Provider endpoint、凭据、运行时 token、SUPI、PSI、SM Context 私有引用或部署授权。它应绑定接口身份，而不是绑定某台机器或某个运行环境。

Artifact 一旦构建即保持不可变。后续增加 Evidence、部署授权或 Provider 实现，不应静默修改旧 Artifact；新的接口闭包需要重新 Build，环境授权和资格材料则由外部系统独立绑定到精确 Artifact 身份。

## 6. Runtime Admission 与 Qualification

### 6.1 执行架构

~~~mermaid
flowchart TD
    ART["Linked Skill Artifact"] --> HOST["Generic Host / Runtime Admission"]
    PROVIDER["Runtime Provider Package + local binding"] --> HOST
    GRANT["Deployment Grant / Scope Policy"] --> HOST
    ASSURANCE["Owner / Mutation / Evidence trust material"] --> HOST
    CONTEXT["Authenticated Invocation Context"] --> HOST
    HOST --> DECIDE{"All target-profile requirements satisfied?"}
    DECIDE -- "No" --> BLOCK["Reject before dispatch / retain unresolved state"]
    DECIDE -- "Yes" --> RUN["Existing Procedure Graph Runtime"]
    RUN --> DISPATCH["Authorized Tool Dispatch"]
    DISPATCH --> PROVIDER
    PROVIDER --> NETWORK["OpenAMSM / 5GC / external systems"]
    NETWORK --> OBS["Authoritative result / Observation"]
    OBS --> VERIFY["Runtime evidence and outcome verification"]
    VERIFY --> RESULT["Invocation terminal result"]
~~~

Host Admission 至少检查：

1. Artifact 格式、完整性和受信来源；
2. 目标 Runtime 与 Artifact / Provider 接口兼容；
3. 所有需要执行的 Callable 都有精确 Provider Binding；
4. 目标环境授予了 Skill 请求的 Capability 和 Scope；
5. 该执行路径所需的 Owner、Mutation、Reconcile、Evidence 保证已满足；
6. Invocation Context、身份、预算、幂等和并发约束有效。

如果只是离线编译，或在隔离开发环境中运行受控 Mock，以上生产要求可以不满足；但不能因此让未授权的真实网络 Mutation 进入生产执行路径。

### 6.2 Qualification 不属于 Compiler

Qualification 是针对具体 Artifact、Provider、Owner、Deployment 和环境的外部结论。它应由相应的发布 / 验收流程产生，而不是由 Compiler 根据缺少诊断推断。

Compiler 可以报告“哪些静态接口已链接、哪些 Assurance 引用未解析”；它不能发布 Owner 证明、授予部署权限或签发生产资格。

## 7. 现有 v1alpha5 与下一代设计的兼容边界

| 项目 | 当前 v1alpha5 / Compiler v2.0.2 | 下一代设计目标 |
|---|---|---|
| 语言语义 | FROZEN | 不因本设计而改变 |
| BuildRequest / Result / Artifact | FROZEN | 如需改变，发布新版本契约 |
| Interface Package | 通过现有依赖快照和 ToolCatalog 等材料表达 | 形成明确、可独立发布的接口包 |
| Capability 定义与授权 | 当前 CapabilityContracts 承担多个职责 | 分离 Capability Catalog 与 Deployment Grant |
| Evidence | 当前静态链接流程要求相应可信材料 | 允许 Interface Build 保留未满足的 Assurance 义务 |
| Deployment | 当前 Build 输入包含可信部署材料 | 不作为接口链接前置；由 Host Admission 检查 |
| Provider 实现 | 不由 Compiler 执行 | 独立发布并在运行时绑定 |
| Package 获取 / 发布 | Compiler 不负责 | 仍由外部 Publisher / Registry / Resolver 负责 |

**不得在 v2.0.2 中偷偷改变现有 Build 的成功条件。** Development Interface Build 应通过新版本化的 Build/API/Artifact Contract 引入，语言目标仍可保持 connectionagent.io/v1alpha5。

## 8. MAS 示例：如何先生成开发 Artifact

截至 2026-11-19，MAS 已经有较完整的源映射候选：

- 13 个固定 MAS Operation 的映射；
- 13 个 CA-facing Input Schema；
- 12 个 normalized result Schema；
- 共 25 个候选 JSON Schema，已进行结构检查；
- ca-client-tools PR #16 已合并 Schema ID / 文件名迁移；
- ca-client-tools PR #14 仍是 Typed Provider 原型，不是正式 Owner 发布；
- ca-skills MAS 仍缺少正式接口包身份和可信依赖快照；ca-context 也需要可解析的接口快照。

因此，下一代 Development Build 的第一步应当是：

1. 将现有 13 个 Operation 和 25 个 Schema 整理成一个**接口候选包**，给出稳定 Package ID、版本和精确 Schema ID。
2. 为 13 个 Skill Capability 提供独立的 Capability Catalog，明确每个 Capability 与 Callable 的关系；不混入部署授权。
3. 为 ca-context 提供可解析的 Subject Context Schema Interface Package。
4. 把这些精确接口快照交给 Compiler，生成 INTERFACE_LINKED 的开发 Artifact。
5. Evidence Guarantee、正式 Owner 信任和 Deployment Grant 暂时缺失时，保留 PENDING / NOT_CHECKED；不声称可生产执行。
6. 等真实 Provider 和 Host Admission 完成后，再进入 Integration E2E。

这一步不要求 ca-client-tools 已经实现所有 Mutation，也不要求 OpenAMSM 已经为每个 Operation 提供正式 Commit Proof；但接口定义必须准确，Effect 不能凭空猜测，实际执行仍受 Host 的安全策略限制。

相关当前记录：

- [ca-skills #71 — MAS source/contract closure](https://github.com/odango-chan/ca-skills/issues/71)
- [ca-client-tools #14 — typed MAS Provider prototype](https://github.com/odango-chan/ca-client-tools/pull/14)
- [ca-client-tools #16 — MAS Schema ID migration](https://github.com/odango-chan/ca-client-tools/pull/16)
- [ca-agent-demo #215 — MAS E2E](https://github.com/odango-chan/ca-agent-demo/issues/215)

## 9. 设计决策与待定项

### 本文建议冻结的原则

1. Interface Link 不依赖 Evidence Guarantee 已经发布。
2. Interface Link 不依赖 Deployment Authorization 已经授予。
3. Interface Link 需要真实、精确的 Callable、Schema、Effect 和 Capability Catalog 定义。
4. Provider 实现可以晚于 Interface Package 发布；没有 Runtime Binding 的 Artifact 不可执行。
5. Evidence / Mutation / Reconcile 是 Assurance 层；缺失时记录未满足义务，不得伪造证明。
6. Deployment Grant 是环境授权；由 Host Admission 检查，不属于 Compiler Link。
7. Compiler 只消费已解析依赖，不做 Package Registry、版本选择、网络下载、发布或 Qualification。
8. v1alpha5 与 v2.0.2 的冻结契约不变；新行为必须使用版本化的新 API / Artifact Contract。

### 下一轮需要单独评审的具体问题

- Interface Package 的物理格式：单一 manifest + catalogs + schema closure，还是可组合的多份 export？
- Capability Catalog 是否独立于 ToolCatalog，还是作为 Interface Package 的独立 export kind？
- Assurance Export 是否允许由不同权威独立发布，如何绑定到精确 Callable / Schema / Interface Package 版本？
- dependencies.lock 由 Studio / CI Resolver 生成，Compiler 校验并嵌入哪些字段？
- Development Artifact 的 unresolved obligations 如何编码，如何确保生产 Host 不会把它当成已授权 Artifact？
- BuildRequest / BuildResult / Artifact Contract 需要怎样的新版本化边界？
- Host 如何对接不同版本的 Runtime Provider，而不把 endpoint 或 credentials 写进 Skill Artifact？

在这些问题完成评审前，示意文件名、字段和状态值均不构成冻结契约。

## 10. 当前实现基线

- CA-SL：connectionagent.io/v1alpha5，FROZEN。
- CA Skill Compiler：v2.0.2，RELEASED。
- Compiler v2.0.2 当前仍按冻结 v1alpha5 BuildRequest / trusted dependency / assurance / deployment 输入执行静态闭合检查。
- 本文提出的 Interface-only Development Build、Capability Catalog 拆分和外部 Runtime Admission 边界均 **NOT IMPLEMENTED**。

当前 Compiler 仍然是静态编译器，不执行 Skill，也不拥有 Package 管理、Provider 部署、Qualification 或真实 E2E。
