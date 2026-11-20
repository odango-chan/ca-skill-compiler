# Development Build 最终架构一致性收敛：实现与验收报告

**状态：PR #51 最终评审版（不再称 Draft）。** 本页记录 `feat/compiler-development-build-20261120` 上实现与 [PR #48 架构基线](https://github.com/odango-chan/ca-skill-compiler/pull/48) 的一致性证据。PR #51 保持 OPEN、READY FOR FINAL REVIEW；它不冻结公开契约、不合并、不发布。

| 项 | 值 |
|---|---|
| 设计契约提交（PR #48 设计内容） | `551a0bba3455cfe4e7dbeede274b686a523316d6` |
| 本报告对应的 PR #48 最终 HEAD | `a76ea350d981d945631b04f07a0bc61c83f525a8` |
| 权威职责基线（固定到该 SHA） | [`compiler-evidence-assurance-responsibility-audit.zh-CN.md`](https://github.com/odango-chan/ca-skill-compiler/blob/a76ea350d981d945631b04f07a0bc61c83f525a8/docs/design/compiler-evidence-assurance-responsibility-audit.zh-CN.md) |
| 产品版本 / 语言目标 | CA Skill Compiler v2.0.0（未改）/ `connectionagent.io/v1alpha5`（未改） |
| 冻结入口 | v2.0.2 BuildRequest / BuildResult / Artifact 逐字节未改 |
| MAS G0 / G1 | **NOT_RUN** |

职责基线只由 PR #48 维护：本分支不再保留该文档的副本，避免两个 PR 各自维护相互冲突的职责矩阵；上文链接固定在 PR #48 的最终 HEAD 上。

## 1. 本轮实现范围

仅 `ca-skill-compiler`。未修改 ca-skills 冻结规范、Compiler v2.0.2 已发布入口/Artifact、ca-agent、ca-agent-demo、ca-client-tools、env-openamsm，也未改任何稳定 ID 或历史产物。

### 1.1 Requirement 两类分离

- 新增独立产物 `runtime-execution-requirements.json`；`host-admission-requirements.json` 保留原有 Host Admission 语义。
- 类别由 `kind` 唯一确定：
  - Host Admission（`HOST_VERIFICATION_REQUIRED`）：`OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST`。
  - Runtime Execution（`RUNTIME_EXECUTION_REQUIRED`）：`COMMIT_PROOF`、`RECONCILE`。
- Host Admission Requirement **恰为四类**；Deployment Grant、Provider Binding 与当前 Invocation 授权是 Host 派发时的检查项，但 Compiler 无法从源码定位，因此不写成 Requirement 条目、不进入任一 Requirement 文件。
- `COMMIT_PROOF` / `RECONCILE` 由同一个 collector 自动路由到 Runtime 文件，不再进入 Host Admission 集合。Runtime Execution Requirement 不是外部授权，实际 Commit Proof 与实际 Reconcile 结论都不在 Build 阶段产生，Host Admission 也不能替代实际 Commit Proof。
- `manifest.json` 新增 `requirements { hostAdmission, runtimeExecution }`，显式声明两个文件的契约身份。
- `VerifyDevelopment` 同时校验两个文件：严格解码（拒绝未知字段与非数组文档）、JCS 规范化、排序、逐条重算 `requirementId`、kind/file/status 三者一致、`location` 为该 kind 的真实位置、independence 明细结构、规范化程度；随后以原始请求重建并逐字节比较全部产物。
- 实际 Commit Proof 与实际 Reconcile 结论不进入编译产物，Compiler 不输出任何“已提交/已恢复”的判定。

### 1.2 Requirement ID 统一摘要域

- 摘要域由 `CA-SKILL-DEVELOPMENT-HOST-REQUIREMENT\0v1\0` 统一为 **`CA-SKILL-DEVELOPMENT-REQUIREMENT\0v1\0`**。
- 原因：旧域名会让 Runtime Execution Requirement 被按 HOST 命名空间解读，与“两类互不替代”的契约冲突。域名现在类名中性，两类共用同一算法。
- 仍保留：`kind` 参与类型化键计算；Host 与 Runtime 的隔离由 `kind`、所在文件与 `status` 三者同时校验，不靠摘要前缀。
- 该域名只被本 PR 尚未合并的实现使用，仓库内没有已发布契约、机器契约或外部消费者依赖它；因此这是安全的重命名，不构成破坏性契约变更。
- Verify 会拒绝用旧域名或任意错误域名重新计算的 Requirement：既在孤立产物校验阶段拒绝（逐条重推导），也在以原始请求重建时拒绝。

### 1.3 Evidence 引用精确定位

- `collectEvidenceGuaranteeReferences` 现在返回每个引用的精确 JSON Pointer：
  - `requires`：`/predicates/<predicate>/evidence/requires/<parameter>/<index>`（参数键与真实数组序号都保留）；
  - Independence 成员：`/predicates/<predicate>/evidence/independence/<group>/members/<member>/guarantee`。
- `developmentReferenceLocations()` 直接使用该位置，不再把所有 Evidence 引用折叠成 `/predicates/<predicate>/evidence`。
- 同一个 Descriptor 被多个 Predicate、多个参数、同一列表的多个序号或 `requires` 与 `independence` 双重要用时，按每个真实位置分别生成 Requirement；内容相同不会被错误合并。
- 顺序语义被明确区分：相同源码字节与相同请求产生逐字节一致的产物；语义上无序的输入（文件、快照、锁、映射顺序）继续规范化；但真实源码列表序号属于位置本身，序号变化允许改变 `requirementId` 与产物摘要，不得为维持错误的顺序无关性而丢失真实位置。

### 1.4 Evidence 多重 Requirement

- 将原先 `alias::exportID -> reason` 的单值映射改为 `alias::exportID -> set(reason)`。
- 同一 `EvidenceDescriptor` 同时声明 `sourceIdentity` 与 `factIdentity` 时，分别产生 `EVIDENCE_DOMAIN_TRUST` 与 `EVIDENCE_FACT_TRUST`，互不覆盖。
- 输入数组重排（文件/快照/锁）不改变产物字节；`requires` 列表重排按 §1.3 处理。

### 1.5 Independence 精确建模

- 每个 Independence Group 记录一条 `EVIDENCE_INDEPENDENCE_TRUST`，携带：

  ```
  independence { groupIndex, members[{dependencyAlias,key,exportId,sourcePath,domain}], domainPairs[[a,b],...] }
  ```

- `independence` **不含 `predicate` 字段**：Predicate 身份只由 `location.node` 表达。此前实现报告把 `predicate` 写进 independence 明细，与实际对外 JSON 契约不一致；现在以契约为准统一。
- 组位置精确为 `/predicates/<predicate>/evidence/independence/<index>`；不再使用 `env.roots[0]` 或单一布尔值。
- 每个成员解析到唯一 Descriptor、唯一 Interface Package key 和声明域；域对逐对列出并规范化排序。
- 组成员缺失、Descriptor 不唯一、缺少 `sourceIdentity`、少于两个声明域等静态非法情形直接 Build FAIL；只有外部 IndependentDomains 授权缺失才记录 Host Requirement。
- 多 Group、多 Root（一个组跨两个根包）、多 Domain Pair 均有组合测试；Verify 逐组校验成员、域、域对与组序号。

### 1.6 IngressCatalog 精确身份

`package.json` 的 `IngressCatalog` 导出现在必须与载荷精确绑定：

1. `apiVersion` / `kind` 精确；
2. 导出 `id` 恰好对应 `spec.operations[].id` 中的一项；
3. 导出 `path` 就是实际 catalog 文件；
4. Operation ID 在文件内唯一；
5. `metadata.owner` 等于包 `publisher`；
6. 文件内每个 operation 必须在同一路径有精确导出（无未导出 operation、无 id/path 漂移）；
7. `artifactBindings` / `schemaBindings` 的依赖别名、快照路径、artifactRef 与 Schema 身份必须落在锁定闭包内。

另修复 `legacyRequest` 把未声明依赖默默回退为 `self` 的隐式路径；现在未声明的依赖别名在准入阶段即失败。

## 2. 架构与实现差异清单

| 架构契约（PR #48） | 实现（PR #51） | 差异 |
|---|---|---|
| 输入只有源码、精确 Interface Package 快照、Dependency Lock；无 `assurance[]` / 可选 Assurance Export | `DevelopmentBuildRequest` 只有 `files/lock/snapshots`；`DevelopmentAssurance` 类型已删除 | 无 |
| `EvidenceDescriptor` 是正式导出，缺失/重复/非法/Producer 不匹配即 FAIL | `EvidenceDescriptor` 导出索引 + 静态链接；失败即无产物 | 无 |
| Host Admission 四类 kind + `HOST_VERIFICATION_REQUIRED` | `host-admission-requirements.json` 仅接受四类 | 无 |
| Runtime 两类 kind + `RUNTIME_EXECUTION_REQUIRED` | `runtime-execution-requirements.json` 仅接受两类 | 无 |
| 两类共用统一摘要域 `CA-SKILL-DEVELOPMENT-REQUIREMENT\0v1\0`，`kind` 参与类型化键 | `developmentRequirementDigestDomain` 已统一；`kind` 在 `developmentRequirementKey` 内参与 JCS 摘要 | 无（本轮修正） |
| 类别由 `kind` + 所在文件 + `status` 三者共同判定 | collector 路由 + `verifyDevelopmentRequirementDocument` 三重校验 | 无 |
| Evidence 引用精确到真实 JSON Pointer（参数键 + 数组序号 / 成员位置） | `collectEvidenceGuaranteeReferences` 返回 `selector`，`developmentReferenceLocations` 直接使用 | 无（本轮修正） |
| `independence` 明细为 `{groupIndex,members,domainPairs}`，Predicate 只在 `location.node` | `DevelopmentIndependenceGroup.Predicate` 标注 `json:"-"`，serialize 后不含该字段 | 无（本轮修正文档） |
| 一个 Independence Group 一条 Requirement，精确 selector | 实现按组生成，selector 精确到组序号 | 无 |
| 多信任事实不互相覆盖 | reason 集合实现，两条并存 | 无 |
| Ingress 导出身份七条 | 七条全部在 `validateDevelopmentPackageExports` / `validateDevelopmentBindingTargets` 实施 | 无 |
| Artifact 恒 `NOT_ADMITTED`，生产 Host 拒收 `DEVELOPMENT` | 未改；测试固定执行状态 | 无 |
| Verify 同时检查两类、拒绝未知字段/重复条目并逐字节重建 | 实现；两份文档严格解码 + 位置校验 + 重建比较 | 无（本轮加强） |

未发现架构文档与实现之间的未决契约级差异。仅存的非契约差异是测试夹具的组织方式（`developmentRequestFromLegacy` 为测试专用转换，不构成对外接口）。

## 3. 测试报告（实际执行）

环境：Windows；Go **go1.26.6 windows/amd64**；Node **v24.18.0**；Google Chrome **158.0.8127.37**；Archify 渲染器 **3.2.0**。

| 门禁 | 命令 | 结果 |
|---|---|---|
| 单元/集成 | `GOWORK=off go test ./... -count=1` | PASS（全部包：compiler / yamlast / ingresscatalog / parity / projectinput / cmd） |
| 静态检查 | `GOWORK=off go vet ./...` | PASS（exit 0，无输出） |
| Native build | `CGO_ENABLED=0 GOWORK=off go build ./...` | PASS（exit 0） |
| Go/WASM build | `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS（exit 0） |
| CLI E2E | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS（3 个子测试：Build / InspectIngressInventory / RejectsV1Alpha4Request） |
| Chromium Worker parity | `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | PASS（Development 5 文件逐字节一致；重排输入逐字节一致；失败诊断一致） |
| 空白/编码 | `git diff --check` | PASS（exit 0） |
| v2.0.2 黄金基线 | `GOWORK=off go test ./internal/compiler -count=1 -run "TestLegacyEntryGoldenBaseline\|TestDependencySnapshotDigestGoldenVectors" -v` | PASS（旧入口行为与 `tests/golden/legacy-entry-v2.0.2.json` 完全一致；三个冻结摘要向量一致） |

> 本轮执行环境使用 danger-full-access 文件策略（Archify 需要启动真实 Chromium 子进程）；Go 缓存仍为默认位置。命令与 AGENTS.md 列出的可复现门禁一致。

专项测试（`internal/compiler/development_requirements_test.go`、`development_locations_test.go` 等）：

| # | 验收项 | 测试 |
|---|---|---|
| 1 | `COMMIT_PROOF` / `RECONCILE` 不出现在 Host Admission | `TestDevelopmentRequirementClassSeparation`、`TestDevelopmentMutationAndReconcileRequirements` |
| 2 | Runtime Requirement 精确覆盖静态 Mutation/Reconcile 契约 | `TestDevelopmentRequirementClassSeparation`（恰好 1 条 COMMIT_PROOF + 1 条 RECONCILE，位置完整） |
| 3 | 同一 Descriptor 两类信任要求同时存在且位置相同 | `TestDevelopmentEvidenceTrustRequirementsAreIndependent`、`TestDevelopmentEvidenceLocationBothIdentitiesShareOnePosition` |
| 4 | 多 Independence Group、多 Root、多 Domain Pair | `TestDevelopmentIndependenceGroupsAreExact`（2 组 / 2 根 / 4 域 / 7 对组合，重排确定性）+ `TestDevelopmentIndependenceSameDomainFails` |
| 5 | IngressCatalog Export 精确身份 | `TestDevelopmentIngressExportIdentity`（错误 id、错误路径、缺失 operation、重复 operation、错误 Owner、未导出 operation） |
| 6 | Evidence 引用精确定位 | `TestDevelopmentEvidenceLocationSinglePredicate`、`…MultiplePredicates`、`…RepeatedInOnePredicate`、`…MultipleDescriptorsInOneList`、`…DrivesRequirementID`、`…ReorderIsLocal` |
| 7 | 统一摘要域；旧域/错误域被拒绝 | `TestDevelopmentRequirementDigestDomainIsUnified`（逐条重推导 + legacy/foreign 域负例） |
| 8 | Requirement 文档篡改矩阵 | `TestDevelopmentVerifyRejectsDocumentTampering`（删除 Host/Runtime、双向跨类迁移、改 status/requirementId/location、加未知字段、重复条目、重算 ID 后跨类） |
| 9 | Independence 明细篡改 | `TestDevelopmentVerifyRejectsIndependenceTampering`（成员域名/ID、成员删除、域对增删改、组序号、粗粒度 selector、多组合并；均重算 ID 并重封 manifest） |
| 10 | 修改 Procedure / Lock / Descriptor 后 Verify FAIL | `TestDevelopmentVerifyRejectsSourceChanges`、`TestDevelopmentVerifyRejectsRequirementClassMigration` |
| 11 | Native / WASM / Chromium 结果一致 | Chromium parity 门禁（build/status/diagnostics/artifactDigest/outputFiles 全部一致） |

CI：PR #51 的 checks 页面 <https://github.com/odango-chan/ca-skill-compiler/pull/51/checks>。PR #48 的文档与图无关 CI 见其 checks 页面。

## 4. 已知限制

- MAS **G0 = NOT_RUN**（没有通过 G0 的真实精确接口快照），**G1 = NOT_RUN**。本报告不宣称 MAS 通过。
- 所有正例仍为 Compiler 自有的**合成 Interface Package 快照**，不代表真实 Owner 发布或产品资格。
- Owner 正式发布（真实 Owner 签名、真实 Interface Package 发布记录）**尚未完成**；本实现只消费调用方提供的精确快照与锁。
- Development Artifact 恒为 `NOT_ADMITTED`；生产 Host 必须拒收 `DEVELOPMENT` profile。
- 真实 Commit Proof、实际 Reconcile、运行时 Evidence、Deployment Grant、Provider Binding 与 Invocation 授权仍由 Host/Runtime 负责，本实现只记录其静态位置与契约。
- Archify 图 01、02 的**人工逐张目视签字仍为 PENDING**；自动化检查与代理人目视记录都不冒充人工签字。图 03 维持“已抽查”。
- v2.0.2 入口保持不变；本报告不构成发布、合并、冻结或打 Tag 的决定。最终判断只到 `READY FOR FINAL REVIEW`。

## 5. 最终职责矩阵（摘要）

本节的完整权威版本只由 PR #48 的[职责审计](https://github.com/odango-chan/ca-skill-compiler/blob/a76ea350d981d945631b04f07a0bc61c83f525a8/docs/design/compiler-evidence-assurance-responsibility-audit.zh-CN.md)（固定在最终 HEAD）与架构文档 §6 维护；下表只是便于实现评审的摘要，不构成第二份职责来源。

| 责任 | Compiler | Interface Package | Host Admission | Runtime / Provider | Owner / Assurance Authority |
|---|---|---|---|---|---|
| Evidence 声明/引用、参数绑定、SourcePath、Schema 类型 | 执行 | 提供快照 | 复核 | — | — |
| EvidenceDescriptor 结构与 Producer/路径 | 执行 | 作为正式导出承载 | 复核 | — | 发布声明 |
| Owner 真实身份、签发权限 | 不判定 | 声明 `publisher` | 验证 `OWNER_TRUST` | — | 承担 |
| IndependentDomains / CommonFact 授权 | 只检查声明形状 | — | 验证两类 TRUST | — | 授权 |
| Independence 组静态结构与域对枚举 | 执行 | 提供 Descriptor | 验证 `EVIDENCE_INDEPENDENCE_TRUST` | — | 授权域对 |
| Mutation / Commit Proof / Reconcile 静态契约 | 执行 | 提供 Callable/Schema | 复核 | 实际执行 | — |
| 实际提交、实际恢复结论 | 不产生、不伪造 | — | 不足以满足 | 产生并验证 `COMMIT_PROOF` / `RECONCILE` | — |
| Deployment Grant / Provider Binding / Invocation 授权 | 不读取 | — | 验证 | 执行期复核 | 部署方签发 |
| 产物执行状态 | 固定 `NOT_ADMITTED` | — | 拒绝未准入产物 | 仅执行已准入 Invocation | — |

完整矩阵见 PR #48 架构文档 §6。
