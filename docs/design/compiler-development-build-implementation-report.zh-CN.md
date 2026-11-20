# Development Build 最终架构一致性收敛：实现与验收报告

**状态：PR #51 最终评审版（不再称 Draft）。** 本页记录 `feat/compiler-development-build-20261120` 上实现与 [PR #48 架构基线](https://github.com/odango-chan/ca-skill-compiler/pull/48) 的一致性证据。PR #51 保持 OPEN、READY FOR FINAL REVIEW；它不冻结公开契约、不合并、不发布。

| 项 | 值 |
|---|---|
| 设计契约提交（PR #48 设计内容） | `551a0bba3455cfe4e7dbeede274b686a523316d6` |
| 本报告对应的 PR #48 最终 HEAD | `5a0340eec26aea6fad744d99743631995a4355d9` |
| 权威职责基线（固定到该 SHA） | [`compiler-evidence-assurance-responsibility-audit.zh-CN.md`](https://github.com/odango-chan/ca-skill-compiler/blob/5a0340eec26aea6fad744d99743631995a4355d9/docs/design/compiler-evidence-assurance-responsibility-audit.zh-CN.md) |
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

### 1.7 EvidenceDescriptor 的 Owner Trust 与物理来源

- `recordOwnerTrustRequirements()` 现在把 `EvidenceDescriptor`（导出 kind `evidenceGuarantee`）与 operation / observation / schema 同等处理：每个被 Predicate 消费的 Descriptor 在每个真实消费位置产生一条 `OWNER_TRUST`，Subject 携带精确 Package Key、Export ID、声明 Owner 与锁定快照路径。
- `owner_linking.go` 的 Descriptor 分支不再只保留 Owner 与 Export ID：它同时保留 Dependency Alias、Package Key、Export ID、声明 Owner（`authority`）与精确 `SourcePath`（Descriptor 文件在锁定快照中的路径）。Descriptor 不携带第二份 Callable 接口 Schema，因此 `schemaIds` 为空；Producer 的输出 Schema 仍由该 Producer 的 Callable 唯一持有。
- Descriptor 缺失或歧义时不做 Owner 归属，避免产生命名不存在 Descriptor 的 `OWNER_TRUST`；静态 Evidence pass 已经使 Build 失败。
- `authority` 与 trusted Owner 不一致、Descriptor 缺失、或多 Owner 快照缺少精确归属，分别按既有 `DEPENDENCY_OWNER_MISMATCH` / `EVIDENCE_GUARANTEE_LINK_INVALID` / `DEPENDENCY_EXPORT_OWNER_UNVERIFIED` 拒绝。
- `EVIDENCE_DOMAIN_TRUST` / `EVIDENCE_FACT_TRUST` 与 `OWNER_TRUST` 是三个独立 Requirement；同一 Descriptor 同时声明 `sourceIdentity` 与 `factIdentity` 时三条并存，Owner 身份不能被其中任何一条替代。
- `VerifyDevelopment` 现在要求 Evidence 阶段的 `OWNER_TRUST` 的 `selector` 必须是真实引用位置：`/predicates/<p>/evidence/requires/<parameter>/<index>` 或 `/predicates/<p>/evidence/independence/<group>/members/<member>/guarantee`；粗粒度 `/evidence` 指针即使重算 `requirementId` 与外层摘要也会被拒绝。
- Producer 输出 Schema 的解析改为：先按 Callable 所在 alias 限定解析，失败后按稳定 `$id` 在已锁定闭包内唯一解析。位于传递锁定包的输出 Schema 因此真正参与 Predicate 参数 Schema 比对，而不是被静默跳过；无法解析或与参数 Schema 不兼容仍是 Build FAIL。
- 新增的定向测试覆盖：Descriptor Authority 精确记录；同一 Descriptor 被多个 Predicate / 多个列表序号引用；多个 Producer 提供同一 Guarantee 且输出 Schema 不同；Producer 输出 Schema 与参数的关联；缺失 / 重复 / 非法 / 未知 Producer / 错误 Owner / Outcome / `provides.path` / 未声明依赖 / Schema 不兼容的负例矩阵；删除 Descriptor `OWNER_TRUST` 与粗粒度 selector 在重封摘要后仍被 Verify 拒绝。

### 1.8 修改文件与逐项原因

| 文件 | 修改 | 原因 |
|---|---|---|
| `internal/compiler/owner_linking.go` | Descriptor 分支保留 `SourcePath`；缺失/歧义时不做归属 | 只保留 Owner 与 Export ID 会丢失物理来源，且可能为不存在的 Descriptor 记录 `OWNER_TRUST` |
| `internal/compiler/development_semantics.go` | `recordOwnerTrustRequirements` 接受 `evidenceGuarantee`；Descriptor 无精确位置时不生成条目 | EvidenceDescriptor 是实际消费的正式导出，必须遵循 Owner Trust；不得用粗粒度位置兜底 |
| `internal/compiler/development_link.go` | Development 信任绑定为 `EvidenceDescriptor` 导出补 `evidenceGuarantee` ExportOwner | 合成绑定应与声明的导出面一致；多 Owner 归属规则才能覆盖 Descriptor |
| `internal/compiler/source_path_producer.go` | 输出 Schema 先按 alias 限定解析，失败后按稳定 `$id` 在锁定闭包内唯一解析 | 传递锁定包中的输出 Schema 原先被静默跳过，Producer 输出 Schema 与 Predicate 参数 Schema 的比对没有真正发生 |
| `internal/compiler/development_build.go` | Verify 对 Evidence 阶段 `OWNER_TRUST` 复用精确 Evidence selector 语法 | 粗粒度 `/evidence` 指针在重算 ID 与外层摘要后仍可能被接受 |
| `internal/compiler/development_evidence_owner_test.go` | 新增正例/负例/Verify 拒绝/静态与运行时缺失区分测试 | 逐项锁定上述契约 |
| `internal/compiler/owner_linking_test.go` | 断言 Descriptor `SourcePath`；新增多 Owner 缺精确归属负例 | 防止物理来源与归属规则回退 |

**修正前后的正反例。**

- 修正前：`OWNER_TRUST` 只覆盖 operation / observation / schema；Descriptor 的 `ownerLinking` 记录只有 Owner 与 Export ID，`SourcePath` 为空。
- 正例（修正后）：fixture 的 `ue.registration-state` 在 `example.ue.registered` 的 `registration` 参数位置被消费，`host-admission-requirements.json` 增加一条 `OWNER_TRUST`：
  - Subject：`key={packageId: urn:example:ca:tools, version: 1.0.0, snapshotDigest: sha256:…}`、`id=ue.registration-state`、`sourcePath=evidence-guarantees.yaml`、`schemaIds=[]`；
  - Location：`selector=/predicates/example.ue.registered/evidence/requires/registration/0`、`stage=evidence`。
- 正例（同时声明两类身份）：`sourceIdentity` + `factIdentity` 使 `OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST` 三条并存，ID 互异。
- 反例：把 Descriptor 的 `authority` 改成未授权 Owner ⇒ `DEPENDENCY_OWNER_MISMATCH`；Producer 不存在 / Owner 不符 / Outcome 非 `SUCCEEDED` / `provides.path` 不匹配 ⇒ `EVIDENCE_GUARANTEE_LINK_INVALID`；未声明依赖别名 ⇒ Build FAIL；删除 `OWNER_TRUST` 或改粗粒度 selector 后重封全部外层摘要 ⇒ `VerifyDevelopment` 仍拒绝。

## 2. 架构与实现差异清单

| 架构契约（PR #48） | 实现（PR #51） | 差异 |
|---|---|---|
| 输入只有源码、精确 Interface Package 快照、Dependency Lock；无 `assurance[]` / 可选 Assurance Export | `DevelopmentBuildRequest` 只有 `files/lock/snapshots`；`DevelopmentAssurance` 类型已删除 | 无 |
| `EvidenceDescriptor` 是正式导出，缺失/重复/非法/Producer 不匹配即 FAIL | `EvidenceDescriptor` 导出索引 + 静态链接；失败即无产物 | 无 |
| 每个被消费的 Descriptor `authority` 与其他跨包导出一样受 Owner Trust 约束，且不能被 Domain/Fact Trust 替代 | `ownerLinking.Exports` 保留 Descriptor 的声明 Owner 与精确 SourcePath；`recordOwnerTrustRequirements` 按真实消费位置记录 `OWNER_TRUST` | 无（本轮修正） |
| Producer 输出 Schema 由 Producer 的 Callable 唯一持有并参与 Predicate 参数 Schema 比对 | `producerOutputSchemas` 先按 alias 限定、再按稳定 `$id` 在锁定闭包内唯一解析；不兼容即 `PREDICATE_TYPE_MISMATCH` | 无（本轮修正） |
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
| CLI E2E | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS（12 个子测试全部通过） |
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
| 12 | Descriptor `OWNER_TRUST` 精确记录 | `TestDevelopmentEvidenceDescriptorOwnerTrustIsExact`（kind/status/Package Key/Export ID/SourcePath/selector 逐项断言） |
| 13 | 同一 Descriptor 被多个 Predicate、多个列表序号引用 | `TestDevelopmentEvidenceDescriptorOwnerTrustPerReferencePosition`（3 个真实位置、3 条 Requirement） |
| 14 | 多个 Producer 提供同一 Guarantee、多个输出 Schema | `TestDevelopmentEvidenceMultipleProducersAndSchemas`、`TestDevelopmentEvidenceProducerSchemaAssociation` |
| 15 | Descriptor 静态非法矩阵 | `TestDevelopmentEvidenceDescriptorNegativeMatrix`（重复/非法/Producer 缺失/Owner 错误/authority 错误/Outcome/provides.path/未声明依赖/Schema 不兼容） |
| 16 | 删除或粗粒度化 Descriptor `OWNER_TRUST` 后重封摘要 | `TestDevelopmentVerifyRejectsDeletedDescriptorOwnerTrust`、`TestDevelopmentVerifyRejectsCoarseDescriptorOwnerTrust` |
| 17 | Owner Trust 不被 Domain / Fact Trust 替代 | `TestDevelopmentEvidenceDescriptorOwnerTrustIsNotReplacedByDomainOrFact`（三条 Requirement ID 互异） |
| 18 | 无实际 Runtime Evidence / Attestation 的静态 Build | `TestDevelopmentBuildNeedsNoRuntimeEvidenceOrAttestation`（请求信封无这些字段、Build 成功、`NOT_ADMITTED`） |
| 19 | 多 Owner 快照缺少 Descriptor 精确归属 | `TestWP3B010MultiOwnerEvidenceGuaranteeMissingAttribution`（`DEPENDENCY_EXPORT_OWNER_UNVERIFIED`） |

CI：本轮实现提交 `1a7c7bdaaf9d5079304a22e1e4d6aa454dc0f61e` 的 `v1alpha5-integration` 运行
<https://github.com/odango-chan/ca-skill-compiler/actions/runs/44710540260>（job PASS，1m6s，全部门禁步骤通过）。
PR #48 分支只改文档与职责边界文字，现有 workflow 的 path filter 不覆盖 `docs/**`，因此其最终提交 `5a0340eec26aea6fad744d99743631995a4355d9` **不触发**任何 check（NOT_RUN，不是 PASS）；职责/流程没有变化，Archify 图不重绘，上一轮已提交的回执保持有效。

## 4. 已知限制

- MAS **G0 = NOT_RUN**（没有通过 G0 的真实精确接口快照），**G1 = NOT_RUN**。本报告不宣称 MAS 通过。
- 所有正例仍为 Compiler 自有的**合成 Interface Package 快照**，不代表真实 Owner 发布或产品资格。
- Development profile 的 Owner 归属来自每个包的单一 `publisher`，因此 Development 路径本身不构造多 Owner 快照；多 Owner 下 Descriptor 缺少精确归属的负例在共享的 owner-linking 层（`TestWP3B010MultiOwnerEvidenceGuaranteeMissingAttribution`）验证。真实多 Owner Interface Package 发布仍待 Provider/Owner 侧材料。
- PR #48 本轮只改职责与边界文字：职责/流程未变，故未重绘 Archify 图 01–03，既有图与截图检查保持原样；仅文档提交换了 HEAD。
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
| EvidenceDescriptor 的 Owner 身份（`authority`） | 只做静态一致性并记录 `OWNER_TRUST` | 以正式导出声明 | 验证 `OWNER_TRUST`，不得用 Domain/Fact Trust 替代 | — | 承担 |
| Owner 真实身份、签发权限 | 不判定 | 声明 `publisher` | 验证 `OWNER_TRUST` | — | 承担 |
| IndependentDomains / CommonFact 授权 | 只检查声明形状 | — | 验证两类 TRUST | — | 授权 |
| Independence 组静态结构与域对枚举 | 执行 | 提供 Descriptor | 验证 `EVIDENCE_INDEPENDENCE_TRUST` | — | 授权域对 |
| Mutation / Commit Proof / Reconcile 静态契约 | 执行 | 提供 Callable/Schema | 复核 | 实际执行 | — |
| 实际提交、实际恢复结论 | 不产生、不伪造 | — | 不足以满足 | 产生并验证 `COMMIT_PROOF` / `RECONCILE` | — |
| Deployment Grant / Provider Binding / Invocation 授权 | 不读取 | — | 验证 | 执行期复核 | 部署方签发 |
| 产物执行状态 | 固定 `NOT_ADMITTED` | — | 拒绝未准入产物 | 仅执行已准入 Invocation | — |

完整矩阵见 PR #48 架构文档 §6。
