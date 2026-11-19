# Development Build 实施契约

**状态：可进入试验实现。** 本页固定首个 Compiler Draft PR 的实现和验收基线。版本标识用于该试验分支。公开契约冻结、合并和发布另行评审。上位设计见[工具链架构](ca-toolchain-architecture-vnext.zh-CN.md)，工作顺序见[交接任务](compiler-development-build-harness.zh-CN.md)。

## 1. 入口与输入

| 入口 | 首轮接口 |
|---|---|
| Go | `BuildDevelopment(request) DevelopmentBuildResult` 和 `VerifyDevelopment(request, result) error`。单一 Go Core 供 Native 与 WASM 调用。 |
| CLI | `skillc build-development --request REQUEST --out DIR --json` 将结果写到 stdout 并原子写入 Artifact 目录。`skillc verify-development --request REQUEST --result RESULT` 读取包含实际 Artifact 字节的 BuildResult JSON。退出码沿用现有 0 成功、1 校验失败、2 用法或输入错误。 |
| WASM | `skillcBuildDevelopment(requestJSON)` 返回结果 JSON 字符串。`skillcVerifyDevelopment(requestJSON, resultJSON)` 返回空字符串或错误信息。 |
| 请求 | `apiVersion=connectionagent.io/compiler-development-build-request/v1alpha1`、`languageTarget=connectionagent.io/v1alpha5`、`profile=DEVELOPMENT`、`files[]`、`targets[]`、`lock`、`snapshots[]`，以及可省略的 `assurance[]`。 |

`files` 沿用现行 `{path,bytes}`，JSON 中的 `bytes` 用标准 Base64，`targets` 沿用现行按 Skill 名称选择的语义，传空数组选择全部。前三个身份字段及其余必填字段必须显式存在。JSON 输入严格拒绝未知字段、重复键、类型不匹配和必填字段为 null。路径、大小、文件数沿用已发布上限，全请求文件预算包含快照、源码、保证材料及 manifest，并逐文件复用现行路径验证。禁止读盘搜索、网络解析或默默选择替代版本。

## 2. 接口快照与锁

- 包键为 `{packageId,version,snapshotDigest}`。包版本采用 SemVer，试验切片的直接依赖版本约束只收精确版本。每个快照为 `{key,files}`，必含根 `package.json`。复用 `ComputeDependencySnapshotDigest` 对包括 `package.json` 在内的文件字节计算快照摘要。manifest 的 `files[]` 仅列其余文件，使用 `{path,sha256}`，由 Compiler 核对清单与实际文件集合完全相等。
- `package.json` 必须包含样例所示的 `apiVersion,packageId,version,languageTarget,publisher,dependencies,exports,files` 字段并使用 `connectionagent.io/interface-package/v1alpha1`。依赖项是 `{alias,packageId,version}`。导出索引为 `{kind,id,path}`，Callable/Ingress 可按精确 ID 从同一原有目录文件中提取多项。同一包同 kind/ID 只允许一个导出，索引的 kind 和 ID 必须与载荷相符。Callable Owner 来自目录声明，Ingress Owner 来自原导出，独立 Schema 与 Capability 的声明身份使用其包发布者。CallableCatalog 和 IngressCatalog 复用已发布逻辑载荷。Schema 使用现有受支持 dialect，导出 ID 与规范 Schema 身份相同。OpenAPI 工件型导出保留明确的 `artifactBindings` 和 `schemaBindings`，沿用冻结 DTO 的元素字段，`dependencyId` 使用当前包的直接依赖别名或保留字 `self`。普通快照省略这两个数组。
- 锁采用 `connectionagent.io/interface-dependency-lock/v1alpha1`、`roots[]` 与 `packages[]`。根为 `{alias,key}`，包记录为 `{key,dependencies}`，边为 `{alias,key}`。每一条锁边都要匹配 manifest 的直接依赖。根别名和包身份与 Skill Contract 逐字匹配。同一图中包 ID 唯一，禁止环、缺边、额外边、重复根和未引用快照。源码 Alias 只选根。包内引用可以解析本包、锁定的依赖闭包和 Schema 本地片段。禁止通过其他根或全局搜索补齐未声明依赖。
- CapabilityCatalog 导出文件为 `{capabilities:[{id,members:[{dependencyAlias,callableId}]}]}`。同包成员省略 `dependencyAlias`。跨包成员只能用本包声明的直接依赖别名。成员通过包键和精确 Callable ID 绑定，Kind/Effect 取自 Callable。目录中不存在 Scope 策略和启动授权。
- 出现在公共接口中的输入/输出、错误、成功/失败 Outcome、Effect、Receipt/Reconcile 类型及需要的 READ Callable 必须完整。对于 Mutation，既有目录中的 receipt 和 Reconcile 数据形状用于链接。只有需要权威材料才能证明的真实性、独立性和可恢复保证可延期。包中声明的 Owner 始终作为接口身份处理，待 Host 验证签发权限。

## 3. 可延期检查

本模式生成 **接口链接产物**。CA-SL v1alpha5 §17.2、附录 G.3 和 §34.0 的保证要求仍适用。使用类型化检查结果标记缺少哪一项权威输入，禁止在冻结校验执行后按错误码过滤。先做格式、源码语法和本地声明检查，再做接口及关联源规则检查。源码故障对 sourceStatus 报错。需要已链接类型的 SourcePath、控制流、路由、Capability、Ingress、保证材料与接口声明进入 Link 阶段。任何内在矛盾或非法声明均失败。

| rule | 何时生成义务 | 仍需立即完成的检查 |
|---|---|---|
| `OWNER_TRUST` | 每个实际使用的跨包导出及其声称 Owner，按消费位置绑定 | 精确包键、导出声明、Owner 字段以及无歧义解析 |
| `EVIDENCE_GUARANTEE` | 每个 Predicate 调用的参数和保证，以及可能到达该消费点的精确 Producer 或恢复生产路径 | SourcePath 可用性、类型、Producer 声明和投影合法性。提供描述时检查形状、绑定、Scope、Correlation、Freshness |
| `EVIDENCE_INDEPENDENCE` | 每个独立性组及其消费位置与相关 Producer 组合 | 成员存在、组结构和可由已知身份判定的冲突 |
| `COMMIT_PROOF` / `RECONCILE` | 每个按静态规则需要保证的 Mutation 发生点，含恢复相关绑定 | Effect、Receipt/Reconcile 类型和 Schema、引用的精确 READ Callable、路由、重试结构及原调用关联字段 |

义务键采用 `{rule,location,subjects}`。location 包含 `skill,path,node,stage,selector`，无法应用的 node/stage 用空串。selector 为源码 JSON Pointer，含列表序号。subjects 是按各项 JCS 字节顺序排序、去重的 `{key,id,sourcePath,schemaIds}` 数组。每项 schemaIds 排序并去重，Schema 字节由锁与闭包固定。相同 Producer/消费在多条等价控制流边复现时去重。不同恢复生产路径使用不同 sourcePath 或不同 subjects。编号为 `sha256:` 加上 `CA-SKILL-DEVELOPMENT-OBLIGATION\0v1\0` 域和义务键 JCS 字节的 SHA-256。每条义务为义务键字段另加 `obligationId,status`，有静态匹配的描述时加 `descriptorDigest`。义务文件内容为按 obligationId 排序的 JCS JSON 数组。

首轮可选 `assurance[]` 仅承载 Evidence Descriptor 静态输入，每项为 `{apiVersion,key,digest,files}`，标识 `connectionagent.io/assurance-export/v1alpha1`。key 是保证所属接口包键，文件为该包下原逻辑格式的 Evidence 描述，digest 用快照算法复算。包内保证引用 + 精确 ID 必须唯一，给定材料逐项解析和验证。至少一个义务实际消费每个已提交描述。非法、无关、摘要不符或静态语义冲突时阻断。满足可判定部分时对应义务标记 `DESCRIPTOR_PRESENT` 并附该 Assurance Export 的 `digest` 作为 `descriptorDigest`。缺失材料、Owner 权限、跨 Owner 信任及 Mutation 权威保证均保持 `UNRESOLVED`，用 `descriptorDigest` 缺省表达无静态描述。两种状态都保留完整义务，描述存在不提供信任。以测试样例覆盖描述缺失、正确与错误三态。

## 4. 结果与复验

- BuildResult 采用 `connectionagent.io/compiler-development-build-result/v1alpha1`。必含 `success`、`sourceStatus`、`interfaceLinkStatus`、`artifactStatus`、`executionStatus`、`diagnostics[]`、`outputFiles[]`。成功为 `true / VALIDATED / LINKED / BUILT / NOT_ADMITTED`。失败时发生错误的阶段为 `FAILED`，后续未运行阶段为 `NOT_RUN`，无输出文件和 artifactDigest。executionStatus 恒为 `NOT_ADMITTED`。成功时另带 `artifactDigest`。静态语言诊断沿用附录 H 的身份。新封装诊断使用新入口单独登记，代码为 `DEVELOPMENT_REQUEST_INVALID`、`DEVELOPMENT_SNAPSHOT_INVALID`、`DEVELOPMENT_LOCK_INVALID`、`DEVELOPMENT_ASSURANCE_INVALID`、`DEVELOPMENT_ARTIFACT_INVALID`。诊断 DTO 沿用现有结构，path 指向源码或相应输入文件，消息标明失败身份。无法解码的输入按 CLI 输入错误处理。
- 产物由 `manifest.json`、`obligations.json` 和现有每 Skill 的相对 Procedure/Ingress 文件组成。manifest 必含 `apiVersion=connectionagent.io/compiler-development-artifact/v1alpha1`、`languageTarget`、`profile`、`generator`、`executionStatus`、`sourceDigest`、`targets`、`lock`、`links`、`assuranceDigests`、`files`。sourceDigest 用快照算法对标准化源码输入计算。links 复用冻结产物中绑定的逻辑内容并带精确包键。assuranceDigests 是实际输入摘要的排序数组。files 为 manifest 之外的 `{path,sha256}`。使用现有 JCS 库，所有语义为集合的数组按元素身份稳定排序，Procedure 的有序列表保留语言顺序。artifactDigest 对 JCS manifest 字节求 SHA-256。
- 首轮 Verify 明确要求**原始请求和实际产物字节**，或由调用方重建相同的规范请求。先验严格格式、文件清单与摘要，再用同一生成器重跑共享的源码、接口和义务计算并逐字节比较所有文件。抽取内部构建函数，避免 Verify 调用会自验的顶层 Build。缺少原请求、锁定快照或相同生成器时，无法完成完整 Verify。覆盖重新哈希后删改义务或 Procedure 仍被拒绝的反例。
- 当前完成条件只到 G1。后续 Host 准入需要可信发布来源、准确的 Provider 和 Grant，并以可信输入完成冻结语言要求的静态复核后再派发。生产 Host 拒收 DEVELOPMENT profile。

## 5. 首轮输入与预期

[registration-development-request.json](fixtures/registration-development-request.json) 是可直接用作新入口正例的原始 JSON。它复用 v2.0.2 dependency-identity fixture 的源码，固定两个根包和一个传递 Schema 包。Evidence 描述从接口快照中移出，Capability 成员从现行 fixture 提取，环境 Scope 留在现行部署上下文中。正例没有 `assurance` 字段。预期输出含 `ue.registration-state` 的 `EVIDENCE_GUARANTEE / UNRESOLVED` 及所用外部身份义务。将原 fixture 的 `evidence-guarantees.yaml` 按 §3 加回时，对应 Evidence 义务变为 `DESCRIPTOR_PRESENT`。Mutation/Reconcile 和 Wait 使用[交接任务](compiler-development-build-harness.zh-CN.md)列出的现有测试构造。
