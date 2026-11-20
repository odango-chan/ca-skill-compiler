# Development Build 实施契约

**状态：可进入试验实现。** 本页固定首个 Compiler Draft PR 的实现和验收基线。版本标识用于该试验分支。公开契约冻结、合并和发布另行评审。上位设计见[工具链架构](ca-toolchain-architecture-vnext.zh-CN.md)，工作顺序见[交接任务](compiler-development-build-harness.zh-CN.md)，Evidence / Assurance 边界见[职责审计](compiler-evidence-assurance-responsibility-audit.zh-CN.md)。

## 1. 入口与输入

| 入口 | 首轮接口 |
|---|---|
| Go | `BuildDevelopment(request) DevelopmentBuildResult` 和 `VerifyDevelopment(request, result) error`。单一 Go Core 供 Native 与 WASM 调用。 |
| CLI | `skillc build-development --request REQUEST --out DIR --json` 将结果写到 stdout 并原子写入 Artifact 目录。`skillc verify-development --request REQUEST --result RESULT` 读取包含实际 Artifact 字节的 BuildResult JSON。退出码沿用现有 0 成功、1 校验失败、2 用法或输入错误。 |
| WASM | `skillcBuildDevelopment(requestJSON)` 返回结果 JSON 字符串。`skillcVerifyDevelopment(requestJSON, resultJSON)` 返回空字符串或错误信息。 |
| 请求 | `apiVersion=connectionagent.io/compiler-development-build-request/v1alpha1`、`languageTarget=connectionagent.io/v1alpha5`、`profile=DEVELOPMENT`、`files[]`、`targets[]`、`lock`、`snapshots[]`。没有独立 `assurance[]` 输入，也没有可选的 Assurance Export 输入。 |

Compiler 的输入**只有**这三类：CA-SL 源字节、精确 Interface Package 快照（`package.json` 导出索引 + 各导出载荷）、Dependency Lock。Assurance Attestation 是外部信任产物，永远不是 Compiler 输入。

`files` 沿用现行 `{path,bytes}`，JSON 中的 `bytes` 用标准 Base64，`targets` 沿用现行按 Skill 名称选择的语义，传空数组选择全部。前三个身份字段及其余必填字段必须显式存在。JSON 输入严格拒绝未知字段、重复键、类型不匹配和必填字段为 null。路径、大小、文件数沿用已发布上限，全请求文件预算包含快照、源码及 manifest，并逐文件复用现行路径验证。禁止读盘搜索、网络解析或默默选择替代版本。

Compiler 不接收 Owner 信任根、Assurance Attestation、真实 Evidence、Deployment Grant 或 Provider。这些属于 Host / Runtime，不能进入 BuildRequest。

## 2. 接口快照与锁

- 包键为 `{packageId,version,snapshotDigest}`。包版本采用 SemVer，试验切片的直接依赖版本约束只收精确版本。每个快照为 `{key,files}`，必含根 `package.json`。复用 `ComputeDependencySnapshotDigest` 对包括 `package.json` 在内的文件字节计算快照摘要。manifest 的 `files[]` 仅列其余文件，使用 `{path,sha256}`，由 Compiler 核对清单与实际文件集合完全相等。
- `package.json` 必须包含样例所示的 `apiVersion,packageId,version,languageTarget,publisher,dependencies,exports,files` 字段并使用 `connectionagent.io/interface-package/v1alpha1`。依赖项是 `{alias,packageId,version}`。导出索引为 `{kind,id,path}`，Callable/Ingress/Capability/EvidenceDescriptor 可按精确 ID 从同一原有目录文件中提取多项。同一包同 kind/ID 只允许一个导出，索引的 kind 和 ID 必须与载荷相符。Callable Owner 来自目录声明，Ingress Owner 来自原导出，独立 Schema、Capability 与 EvidenceDescriptor 的声明身份使用其包发布者。CallableCatalog 和 IngressCatalog 复用已发布逻辑载荷。Schema 使用现有受支持 dialect，导出 ID 与规范 Schema 身份相同。OpenAPI 工件型导出保留明确的 `artifactBindings` 和 `schemaBindings`，沿用冻结 DTO 的元素字段，`dependencyId` 使用当前包的直接依赖别名或保留字 `self`。普通快照省略这两个数组。
- **IngressCatalog 导出身份（七条，全部强制）。** 导出必须由 `package.json` 精确绑定到载荷，不能只凭文件存在或 `apiVersion` 正确：① 载荷 `apiVersion` 与 `kind` 与声明一致；② 导出 `id` 精确等于载荷 `spec.operations[].id` 中的恰好一项；③ 导出 `path` 等于实际 catalog 文件；④ Operation ID 在文件中唯一；⑤ catalog `metadata.owner` 等于包 `publisher`；⑥ 每个声明的 operation 在同一路径都有精确导出（不得有未导出 operation，不得有 id/path 漂移）；⑦ `artifactBindings` / `schemaBindings` 指向精确的包或依赖快照路径和已声明 Schema 身份。任一条不成立即 Build 失败。
- **EvidenceDescriptor 是静态接口契约。** 它是 Interface Package 的正式导出类型，导出 ID 为保证 ID，文件为原逻辑格式的 Evidence 描述。其字节由 Dependency Lock 固定，缺失、重复、结构非法或与引用不符都按冻结语言规则使 Build 失败。Evidence 描述不是 Assurance Attestation，也不是运行时 Evidence。
- 锁采用 `connectionagent.io/interface-dependency-lock/v1alpha1`、`roots[]` 与 `packages[]`。根为 `{alias,key}`，包记录为 `{key,dependencies}`，边为 `{alias,key}`。每一条锁边都要匹配 manifest 的直接依赖。根别名和包身份与 Skill Contract 逐字匹配。同一图中包 ID 唯一，禁止环、缺边、额外边、重复根和未引用快照。源码 Alias 只选根。包内引用可以解析本包、锁定的依赖闭包和 Schema 本地片段。禁止通过其他根或全局搜索补齐未声明依赖。
- CapabilityCatalog 导出文件为 `{capabilities:[{id,members:[{dependencyAlias,callableId}]}]}`。同包成员省略 `dependencyAlias`。跨包成员只能用本包声明的直接依赖别名。成员通过包键和精确 Callable ID 绑定，Kind/Effect 取自 Callable。目录中不存在 Scope 策略和启动授权。
- 出现在公共接口中的输入/输出、错误、成功/失败 Outcome、Effect、Receipt/Reconcile 类型及需要的 READ Callable 必须完整。对于 Mutation，既有目录中的 receipt 和 Reconcile 数据形状用于链接。包中声明的 Owner 始终作为接口身份处理，待 Host 验证签发权限。

## 3. Evidence 静态语义与外部待决

本模式生成**接口链接产物**。CA-SL v1alpha5 §17.2、附录 G.3 和 §34.0 的 Evidence 语言要求必须由 Compiler 完整执行：

- Evidence 声明、引用解析、Predicate 参数绑定。
- Evidence Descriptor 结构、authority/fact/producers/scope/correlation/factValue/selection/freshness 的静态形状。
- Producer 身份与 trusted ToolCatalog Callable、`provides.path` 的唯一匹配；Producer 与 Consumer 关系。
- SourcePath、数据流、Schema 类型与 `$ref` 闭包。
- Independence 组成员存在、组形状、sourceIdentity 声明形状及静态可判定冲突。
- Mutation / Commit Proof / Reconcile 的静态约束、路由、恢复与重试可达性。

**Descriptor 缺失或非法即失败。** 不得以任何「未决」状态替代冻结语言要求的静态检查，也不得在冻结校验执行后按错误码过滤后放行。

只有需要外部权威材料才能判定的事实在静态检查通过后记录为 Requirement；Requirement 分成**两类、两个文件**，互不替代：

| 类别 | 覆盖范围 | 产物文件 | 固定 status |
|---|---|---|---|
| Host Admission Requirement | 派发前可验证的外部事实 | `host-admission-requirements.json` | `HOST_VERIFICATION_REQUIRED` |
| Runtime Execution Requirement | 只能在执行期产生的真实结果 | `runtime-execution-requirements.json` | `RUNTIME_EXECUTION_REQUIRED` |

**Host Admission Requirement 的 kind 与记录时机**（仅这七项外部事实）：

| kind | 记录时机 | 仍立即完成的静态检查 |
|---|---|---|
| `OWNER_TRUST` | 每个实际使用的跨包导出及其声明 Owner，按消费位置绑定 | 精确包键、导出声明、Owner 字段以及无歧义解析 |
| `EVIDENCE_DOMAIN_TRUST` | 静态合法的 Descriptor 声明了 `sourceIdentity`，需要外部域授权 | Descriptor 结构、sourceIdentity 形状、Producer 匹配 |
| `EVIDENCE_FACT_TRUST` | 静态合法的 Descriptor 声明了 `factIdentity`，需要外部 CommonFact 授权 | Descriptor 结构、factIdentity 形状、Producer 匹配 |
| `EVIDENCE_INDEPENDENCE_TRUST` | 每个 Independence Group 静态结构成立且跨多个声明域，需要 IndependentDomains 授权 | 成员存在、组结构、sourceIdentity 声明、组序号 |
| Deployment Grant | 每条需要环境授权的 Invocation 路径 | Grant 不进入 Compiler；此处仅记录缺少派发前环境授权 |
| Provider Binding | 每条需要固定 Provider 实例的派发路径 | 接口键与兼容范围静态一致，实例绑定由 Host 固定 |
| Invocation 授权 | 每次派发/恢复路径 | 不读取调用方凭据；仅记录派发前授权待验 |

**Runtime Execution Requirement 的 kind**（仅这两项，且都不是可在 Build 或派发前满足的 Host Admission Requirement）：

| kind | 记录时机 | 执行期必须真正发生的事 |
|---|---|---|
| `COMMIT_PROOF` | 每个按静态规则需要保证的 Mutation 发生点 | 实际 Mutation 提交，并核验与该次提交绑定的 proof |
| `RECONCILE` | 带 Reconcile 契约的 Mutation（含恢复相关绑定） | `UNKNOWN` 后对精确原尝试执行 Reconcile，并给出实际恢复结论 |

真实 Commit Proof 与 Reconcile 结论**永不进入已编译产物**，Compiler 不预测、不伪造。Runtime / Host 必须分别区分「运行时机制支持检查」（Contract、Schema、path、READ Callable、路由与重试结构是否就位，Build 已静态判定）与「实际执行结果」（这一次提交/恢复了什么）。

Requirement 记录形状为：

```
{ requirementId, kind,
  location { skill, path, node, stage, selector },
  subjects [ { key { packageId, version, snapshotDigest }, id, sourcePath, schemaIds } ],
  status }
```

`location` 无法应用的 `node`/`stage` 用空串；`selector` 为源码 JSON Pointer，含列表序号。subjects 是按各项 JCS 字节顺序排序、去重的数组；每项 `schemaIds` 排序并去重，Schema 字节由锁与闭包固定。相同 Producer/消费在多条等价控制流边复现时去重。`requirementId` 为 `sha256:` 加上 `CA-SKILL-DEVELOPMENT-HOST-REQUIREMENT\0v1\0` 域和类型化键 JCS 字节（**含 independence 明细**）的 SHA-256。文件内容为按 `requirementId` 排序的 JCS JSON 数组。

`EVIDENCE_INDEPENDENCE_TRUST` 另带 `independence` 明细：

```
independence { groupIndex,
  members [ { dependencyAlias, key, exportId, sourcePath, domain } ],
  domainPairs [ [domainA, domainB], ... ] }
```

**一个 Independence Group 一条 Requirement**：来源精确到 `selector=/predicates/<predicate>/evidence/independence/<index>`；每个成员 Descriptor、每个 Source Domain、每个被授权的 Domain Pair、每个精确 Interface Package key 都写入该条。不使用 `env.roots[0]`，也不使用单一全局布尔值；多组、多根、多域对互相独立。组的静态结构非法（成员缺失、组形状非法、绑定不成立）是 Build 失败；只有外部域授权缺失才是 Host Admission Requirement。

**多信任事实不互相覆盖。** 同一个 EvidenceDescriptor 同时声明 `sourceIdentity` 与 `factIdentity` 时，同时产生 `EVIDENCE_DOMAIN_TRUST` 与 `EVIDENCE_FACT_TRUST` 两条独立 Requirement，任一条不得覆盖另一条。输入顺序（文件、Predicate、快照）不得改变产物字节或 Requirement 集合。

这些 Requirement 不是 Compiler 的验收结论，也不表示 Build 失败或通过；它们分别记录 Host 派发前必须补齐的外部事实与 Runtime 执行期必须真正发生的结果。Compiler 不内建 Assurance Engine、签名验证、信任根或运行时 Evidence 检查。

## 4. 结果与复验

- BuildResult 采用 `connectionagent.io/compiler-development-build-result/v1alpha1`。必含 `success`、`sourceStatus`、`interfaceLinkStatus`、`artifactStatus`、`executionStatus`、`diagnostics[]`、`outputFiles[]`。成功为 `true / VALIDATED / LINKED / BUILT / NOT_ADMITTED`。失败时发生错误的阶段为 `FAILED`，后续未运行阶段为 `NOT_RUN`，无输出文件和 artifactDigest。executionStatus 恒为 `NOT_ADMITTED`。成功时另带 `artifactDigest`。静态语言诊断沿用附录 H 的身份。新封装诊断使用 `DEVELOPMENT_REQUEST_INVALID`、`DEVELOPMENT_SNAPSHOT_INVALID`、`DEVELOPMENT_LOCK_INVALID`、`DEVELOPMENT_ARTIFACT_INVALID`。诊断 DTO 沿用现有结构，path 指向源码或相应输入文件，消息标明失败身份。无法解码的输入按 CLI 输入错误处理。
- 产物由 `manifest.json`、`host-admission-requirements.json`、`runtime-execution-requirements.json` 和现有每 Skill 的相对 Procedure/Ingress 文件组成。manifest 必含 `apiVersion=connectionagent.io/compiler-development-artifact/v1alpha1`、`languageTarget`、`profile`、`generator`、`executionStatus`、`sourceDigest`、`targets`、`lock`、`links`、`files`。sourceDigest 用快照算法对标准化源码输入计算。links 复用冻结产物中绑定的逻辑内容并带精确包键。files 为 manifest 之外的 `{path,sha256}`。使用现有 JCS 库，所有语义为集合的数组按元素身份稳定排序，Procedure 的有序列表保留语言顺序。artifactDigest 对 JCS manifest 字节求 SHA-256。两个 Requirement 文件都为空时仍生成（空数组），保证产物布局固定。
- 首轮 Verify 明确要求**原始请求和实际产物字节**，或由调用方重建相同的规范请求。先验严格格式、文件清单与摘要，并逐项检查**两个** Requirement 文件：规范化 JCS、排序、逐条重新推导 `requirementId`（含 independence 明细）、kind 与所在文件类别一致、`status` 正确、覆盖完整；再用同一生成器重跑共享的源码、接口和 Requirement 计算并逐字节比较所有产物文件。抽取内部构建函数，避免 Verify 调用会自验的顶层 Build。缺少原请求、锁定快照或相同生成器时，无法完成完整 Verify。删除一条 Requirement 后重新封装外层摘要仍被拒绝；改动 Procedure、Lock、Descriptor 或 Requirement 均被拒绝。
- 当前完成条件只到 G1。后续 Host 准入需要可信发布来源、准确的 Provider 和 Grant，并以可信输入完成冻结语言要求的静态复核后再派发。生产 Host 拒收 DEVELOPMENT profile；隔离测试 Host 只有在显式 opt-in 时才可接收。开发产物执行状态恒为 `NOT_ADMITTED`，Build 成功不等于授权或资格结论。

## 5. 首轮输入与预期

[registration-development-request.json](fixtures/registration-development-request.json) 是可直接用作新入口正例的原始 JSON。它复用 v2.0.2 dependency-identity fixture 的源码，固定两个根包和一个传递 Schema 包，并把 `evidence-guarantees.yaml` 作为 `ca` 包的 `EvidenceDescriptor` 导出，由锁固定。Capability 成员从现行 fixture 提取，环境 Scope 留在现行部署上下文中。正例没有 Assurance、Grant 或 Provider 输入。预期输出含 `OWNER_TRUST` Host Admission Requirement；如该请求不使用 Mutation/Reconcile，则 `runtime-execution-requirements.json` 为空数组。静态 Evidence 检查全部通过，执行状态为 `NOT_ADMITTED`。删除该 Descriptor 导出会按 `EVIDENCE_GUARANTEE_LINK_INVALID` 拒绝 Build。Mutation/Reconcile 和 Wait 使用[交接任务](compiler-development-build-harness.zh-CN.md)列出的现有测试构造。
