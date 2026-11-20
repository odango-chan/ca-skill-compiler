# Development Build 最终架构一致性收敛：实现与验收报告

**状态（2026-11-20）：Development Build 已评审通过并随 [PR #51](https://github.com/odango-chan/ca-skill-compiler/pull/51) 合并到 `main`（`1d7ea81`），依赖的架构 [PR #48](https://github.com/odango-chan/ca-skill-compiler/pull/48) 已合并（`22860a2`）。** 本页保留合并前实现分支、固定 SHA、测试与责任审计证据作为历史记录；未另行打 Tag、发布新版本或获得生产 Qualification。MAS G0/G1 仍 NOT_RUN。

本轮为 Compiler 最终收敛，只修复三个 P1：Capability 静态需求进入 Artifact、统一 Package-scoped Schema Resolver、多 Skill Lock Root 并集验证。未扩大 Compiler 职责，未修改 EvidenceDescriptor 设计、CA-SL v1alpha5 语言语义或 v2.0.2 已发布入口与 Artifact。

| 项 | 值 |
|---|---|
| 设计契约提交（PR #48 设计内容） | `551a0bba3455cfe4e7dbeede274b686a523316d6` |
| 本报告当时固定的 PR #48 设计参考 HEAD（后续仅补 CI 安全修复） | `27324bcf273ae640acdf27637f9a7de89ed2ffe0` |
| 本报告对应的 PR #51 实现代码提交 | `2e2a5840d2972f17c46c1cb1a6249ab9c50df58e` |
| 权威职责基线（固定到 PR #48 设计参考 HEAD） | [`compiler-evidence-assurance-responsibility-audit.zh-CN.md`](https://github.com/odango-chan/ca-skill-compiler/blob/27324bcf273ae640acdf27637f9a7de89ed2ffe0/docs/design/compiler-evidence-assurance-responsibility-audit.zh-CN.md) |
| 已发布产品 / 语言目标 | CA Skill Compiler v2.0.2（未改）/ `connectionagent.io/v1alpha5`（未改） |
| 冻结入口 | v2.0.2 BuildRequest / BuildResult / Artifact 逐字节未改 |
| MAS G0 / G1 | **NOT_RUN** |
| 合并检查点（不代表新版本 Release） | #48 `22860a26717a5a87d5af42c4674eec77ab4f7021`；#51 `1d7ea81c4b163b09add0314ecf949505128e3546` |

职责基线由 PR #48 维护；原实现分支不保留该文档的副本，以免两处维护互相冲突的职责矩阵。上文链接固定在合并前的 PR #48 设计参考提交上；随后的 CI 安全修复不改变职责契约。

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

### 1.9 Capability Grant 进入最终 Artifact（P1-1）

**根因。** `validateDevelopmentCapabilityLinking()` 已能解析 Skill 的 `capabilities`、定位 CapabilityCatalog、验证 Callable 成员与 Grant 覆盖，但结果中的 Grants/Capabilities 只用于诊断：Compiler 验证了 Skill 选择了哪个 Capability，却没有把选择交给 Host。Host 只能重新读取 SKILL 源码或从 Callable 集合反推，静态链接不完整。

**最终实现。**

- `manifest.json` 新增 `capabilityRequirements[]`。每条记录：

  ```
  { skill,
    capabilityId,
    catalog { key { packageId, version, snapshotDigest }, owner, exportId, sourcePath, pointer },
    callables [ { key { packageId, version, snapshotDigest }, owner, exportId, kind, effect, sourcePath } ],
    declaration { path, pointer } }
  ```

- `capabilityId` 来自 Skill 的声明；`catalog` 是实际绑定的 CapabilityCatalog 导出，`key` 为精确 Package Key，`owner` 取自**锁定 Package manifest 的 `publisher`**（不采用 Catalog 文件内自声明），`exportId`/`sourcePath`/`pointer` 为精确导出身份与 `/capabilities/<index>` 位置。
- `callables[]` 保留该 Capability 实际包含的全部成员，每项带自己的 Package Key、Export ID、SourcePath、Kind 与 Effect。
- `declaration` 是 Skill 原始声明位置：`skills/<skill>/assurance/contract.yaml` + `/capabilities/<index>`。
- 多个 Capability 包含同一 Callable 时，只记录 Skill 实际声明的那一条；不按 Callable 反推、不把所有匹配项都当成授权需求。
- Skill 声明的 Capability 找不到唯一锁定 Catalog 时 Build FAIL（`CAPABILITY_MAPPING_MISSING`），不再静默丢弃。
- CapabilityCatalog 发布者身份按既有 `OWNER_TRUST` 语义记录一条 Host Admission Requirement：`location` 为 Skill 的 `/capabilities/<index>` 声明位置，`subjects[0]` 为 Catalog 的 Package Key、Export ID 与 SourcePath。Host 仍负责真实 Owner 信任与准入；Deployment Grant 不进入 Artifact。
- 该字段是 manifest 的一部分，随 manifest 摘要与逐字节重建验证；Verify 另做结构校验（Package Key/Export/Owner/SourcePath/Pointer/Callable/声明位置必须完整、Kind/Effect 合法、成员不重复、按规范化身份排序）。

**正向样例（即 `tests/golden/development-capability-requirements-v1.json` 的黄金基线）：**

```json
{"skill":"verify-ue-registration","capabilityId":"example.registration-read",
 "catalog":{"key":{"packageId":"urn:example:ca:tools","version":"1.0.0","snapshotDigest":"sha256:0f41…"},
            "owner":"example-ue-owner","exportId":"example.registration-read",
            "sourcePath":"capabilities.json","pointer":"/capabilities/0"},
 "callables":[{"key":{"packageId":"urn:example:ca:tools","version":"1.0.0","snapshotDigest":"sha256:0f41…"},
               "owner":"example-ue-owner","exportId":"ue.registration.current",
               "kind":"observation","effect":"READ","sourcePath":"tool-catalog.yaml"}],
 "declaration":{"path":"skills/verify-ue-registration/assurance/contract.yaml","pointer":"/capabilities/0"}}
```

Host 直接读取 Artifact 即可得到 Skill 请求的 Capability、精确 Catalog Package/Owner/导出与成员，无需读取原始 SKILL 源码；该记录是静态需求，不是部署授权。

### 1.10 统一 Package-scoped Schema Resolver（P1-2）

**根因。** 带 Alias 的 `$ref` 直接使用 `scope.env.rootByAlias[alias]`，只适用于 Skill 的 Root Alias；Interface Package 自己的 `packageAliases` 无法参与。同时 `producerOutputSchemas` 的全局 `$id` fallback 与 `hasSchema()` 的闭包全局接受，允许使用「已锁定但当前 Package 未声明依赖」的 Schema——这是依赖边界错误，也解释了 `Skill → A → B → C` 链无法通过 B 自己的别名解析 C。

**最终实现（单一解析入口）。**

- `development_link.go` 新增 `developmentSchemaContext` 与唯一入口 `resolveSchemaReference`；`resolveSchemaInPackage` 固定「本包 → 本包声明的直接依赖」两段解析。
- 规则：Skill 源使用 Skill Contract 声明的根别名；Interface Package 内的 Schema 使用该 Package manifest 声明的直接依赖别名；绝对 `$id` 只解析到声明文档所属 Package 自身或其直接依赖；被引用 Schema 内的后续 `$ref` 按该 Schema 所属 Package 重新确定作用域。全局 `$id` 只用于身份识别，绝不作为跨 Package 依赖的授权。
- `validateDevelopmentSchemaClosure` 用该入口遍历 Context/Request Schema 与每个 Callable 的完整 Schema；失败诊断包含当前 Package、Alias、SourcePath 与 `$ref` 的 JSON Pointer。
- `schema_registry.go` 增加可选 Development 语义：`schemaResource.packageKey` + `resolveReferenceFrom()`。Development 下所有继承的 Schema 入口（Closure、Context root、SourcePath、参数/脚本/Predicate/Mutation typing、Producer 输出）共用该入口；frozen v1alpha5 的 `development == nil` 分支逐字节保持原行为。
- `source_path_producer.go` 的 Producer 输出 Schema 在 Development 下不再做全局 `$id` fallback：只按 Callable 所属 Package 的依赖作用域解析；嵌套 `$ref` 跟踪所属 Package。
- JSON Schema 编译视图仅在 Development 下移除已证明的 Alias 限定符（`alias::uri` → `uri`），供 San 引擎编译；冻结入口从不改写。
- 删除 `scope.hasSchema()` 的全局接受路径与 `resolveDevelopmentSchemaRef()` 对 `rootByAlias` 的直接依赖。

**测试矩阵。**

| # | 用例 | 结果 |
|---|---|---|
| 1 | Skill Root Alias（`contextSchema`/`requestSchema` 引用根包） | PASS |
| 2 | Package 自身 Schema 引用（fixture `tools` 内） | PASS |
| 3 | Package 直接依赖 Schema（`tools → types`） | PASS |
| 4 | `Skill → A → B → C`，B 用自身别名引用 C，Schema 内继续解析 | PASS |
| 5 | A 未声明 C，但 C 在 Lock Closure，A 用绝对 `$id` 引用 C | FAIL（`SCHEMA_MISSING`） |
| 6 | 未声明 Alias | FAIL（`SCHEMA_MISSING`） |
| 7 | Alias 指向不导出该 Schema 的 Package | FAIL（`SCHEMA_MISSING`） |
| 8 | 非法相对引用 | FAIL（`SCHEMA_MISSING`） |
| 9 | 重复 Schema 身份 | FAIL（`DEVELOPMENT_LOCK_INVALID`） |

### 1.11 多 Skill 共用 Lock 的 Root 并集验证（P1-3）

**根因。** `validateDevelopmentRootsAgainstContracts()` 对每个选中 Skill 都要求「整个 BuildRequest 的所有 Lock Roots 都被当前 Skill 声明」。多 Skill 共用一个 Lock 时，这等价于要求每个 Skill 声明其他 Skill 的依赖，语义错误。

**最终实现。**

- 先求根依赖并集：`R = D(S1) ∪ … ∪ D(Sn)`，再验证：
  1. 每个 Skill 声明的 Alias 存在对应 Lock Root；
  2. 每个 Alias 解析到与其声明一致的 Stable Package ID；
  3. 每个 Root 至少被一个选中 Skill 声明（无未引用 Root）；
  4. 同一 Alias 不得跨 Skill 绑定不同 Package 身份；
  5. Lock Closure 等于 `R` 的传递依赖闭包（现有 reachability 准入强制，无缺失传递依赖、无不可达 Package）。
- 单 Skill 行为保持不变。

**测试矩阵。**

| # | 用例 | 结果 |
|---|---|---|
| 1 | 两个 Skill、不同 Root、共同 Lock | PASS（逐字节可复验） |
| 2 | 缺少 Skill A 的 Root | FAIL（`DEVELOPMENT_LOCK_INVALID`） |
| 3 | Alias 对应错误 Package ID | FAIL |
| 4 | 存在未被任何 Skill 引用的额外 Root | FAIL |
| 5 | Lock Closure 缺少传递依赖 | FAIL |
| 6 | 同名 Alias 跨 Skill 指向不同 Package | FAIL |
| 7 | 单 Skill 正例 | PASS（不回退） |
| 8 | 多 Skill 输入顺序重排 | 产物逐字节一致 |

### 1.12 本轮修改文件

| 文件 | 修改 | 原因 |
|---|---|---|
| `internal/compiler/development_artifact.go` | 新增 `capabilityRequirements` 记录类型、manifest 字段、确定性排序与规范化 | Capability 静态需求必须成为 Artifact 完整性的一部分 |
| `internal/compiler/development_obligations.go` | 生成 Capability 需求；声明了却无唯一 Catalog 时失败；记录精确声明位置 | 不把 Skill 的选择丢在诊断里 |
| `internal/compiler/development_semantics.go` | 汇总 Capability 需求并记录 Catalog `OWNER_TRUST` | 复用既有 Owner Trust 语义，不新造信任模型 |
| `internal/compiler/development_build.go` | Verify 结构校验 Capability 需求；Capability `OWNER_TRUST` 精确位置校验 | 重封外层摘要也不能绕过语义校验 |
| `internal/compiler/development_link.go` | 统一 Package-scoped Schema Resolver；删除全局 `hasSchema()`/`rootByAlias` 解析路径 | 依赖边界错误 |
| `internal/compiler/schema_registry.go` | Development 可选语义：Package key、`resolveReferenceFrom`、编译视图去 Alias 限定符 | 所有 Schema 入口共用同一规则 |
| `internal/compiler/source_path_producer.go` | Development 下删除全局 `$id` fallback，嵌套 `$ref` 跟踪所属 Package | 关闭绕过路径 |
| `internal/compiler/development_request.go` | Root 并集、跨 Skill 别名身份一致、未引用 Root 检查 | 多 Skill 共享 Lock 的正确语义 |
| `internal/compiler/api.go` | 私有 `development` 字段，仅为内部请求携带 resolver | 冻结入口不设置它，行为不变 |
| `internal/compiler/development_capability_artifact_test.go` | 正例、精确选择、缺失 Catalog、12 项篡改矩阵 | 锁定 P1-1 |
| `internal/compiler/development_schema_scope_test.go` | A→B→C 正例 + 5 类反例 + 测试包图构造器 | 锁定 P1-2 |
| `internal/compiler/development_multi_skill_test.go` | 并集正例、5 类反例、重排确定性、单 Skill 回归 | 锁定 P1-3 |
| `tests/golden/development-capability-requirements-v1.json` | Capability 需求黄金基线 | 新字段有对应基线 |

## 2. 架构与实现差异清单

| 架构契约（PR #48） | 实现（PR #51） | 差异 |
|---|---|---|
| 输入只有源码、精确 Interface Package 快照、Dependency Lock；无 `assurance[]` / 可选 Assurance Export | `DevelopmentBuildRequest` 只有 `files/lock/snapshots`；`DevelopmentAssurance` 类型已删除 | 无 |
| `EvidenceDescriptor` 是正式导出，缺失/重复/非法/Producer 不匹配即 FAIL | `EvidenceDescriptor` 导出索引 + 静态链接；失败即无产物 | 无 |
| 每个被消费的 Descriptor `authority` 与其他跨包导出一样受 Owner Trust 约束，且不能被 Domain/Fact Trust 替代 | `ownerLinking.Exports` 保留 Descriptor 的声明 Owner 与精确 SourcePath；`recordOwnerTrustRequirements` 按真实消费位置记录 `OWNER_TRUST` | 无（本轮修正） |
| Producer 输出 Schema 由 Producer 的 Callable 唯一持有并参与 Predicate 参数 Schema 比对 | Development 下按 Callable 所属 Package 的依赖作用域解析，不做全局 `$id` fallback；不兼容即 `PREDICATE_TYPE_MISMATCH` | 无（本轮按 Package-scoped 规则收敛） |
| Skill 的 Capability 选择进入 Artifact，Host 不读 SKILL 源码（PR #48 §3.1/§4） | manifest `capabilityRequirements[]` 记录精确 Catalog Package/Owner/Export/SourcePath/Pointer、Callable 成员与 Skill 声明位置；Verify 覆盖篡改 | 无（本轮修正） |
| Schema 解析遵守 Package Dependency（PR #48 §3.2） | 单一 Package-scoped Resolver；全局 `$id` 仅识别身份 | 无（本轮修正） |
| 多 Skill 共用 Lock 使用 Root 并集（PR #48 §3.2） | `validateDevelopmentRootsAgainstContracts` 先求并集再验证 | 无（本轮修正） |
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

环境：Windows；Go **go1.26.6 windows/amd64**；Node **v24.18.0**；Google Chrome **158.0.8127.37**。

| 门禁 | 命令 | 结果 |
|---|---|---|
| 单元/集成 | `GOWORK=off go test ./... -count=1` | PASS（全部包：compiler / yamlast / ingresscatalog / parity / projectinput / cmd） |
| 静态检查 | `GOWORK=off go vet ./...` | PASS（exit 0，无输出） |
| Native build | `CGO_ENABLED=0 GOWORK=off go build ./...` | PASS（exit 0） |
| Go/WASM build | `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS（exit 0） |
| CLI E2E | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS（12 个子测试全部通过） |
| Chromium Worker parity | `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | PASS（真实 Chromium Worker；Development 5 文件逐字节一致；重排输入逐字节一致；失败诊断一致） |
| 空白/编码 | `git diff --check` | PASS（exit 0） |
| v2.0.2 黄金基线 | `GOWORK=off go test ./internal/compiler -count=1 -run "TestLegacyEntryGoldenBaseline" -v` | PASS（与 `tests/golden/legacy-entry-v2.0.2.json` 完全一致） |
| Capability 黄金基线 | `GOWORK=off go test ./internal/compiler -count=1 -run TestDevelopmentCapabilityGoldenBaseline -v` | PASS（与 `tests/golden/development-capability-requirements-v1.json` 逐字节一致） |
| 重复构建/重排 | `TestDevelopmentBuildPositiveFixture`、`TestDevelopmentBuildIsReorderingInvariant`、`TestDevelopmentMultiSkillReorderingInvariant` | PASS（相同输入逐字节一致；无序输入重排后逐字节一致） |
| GitHub Actions | run [44717886662](https://github.com/odango-chan/ca-skill-compiler/actions/runs/44717886662)（实现提交 `2e2a584`）及其后文档提交的 synchronize 运行（如 [44719034235](https://github.com/odango-chan/ca-skill-compiler/actions/runs/44719034235)） | `v1alpha5-integration` job success（1m10s 级：focused 测试、全量测试、vet、Native/WASM build、CLI E2E、真实 Chromium Worker parity、diff check 全部通过） |

> 本轮在 Windows + danger-full-access 文件策略下执行；Chromium 为真实浏览器 Worker，不是 Native 结果替代。CI 结果是实际 GitHub Actions 运行，不是本地结果冒充。PR #51 的每次 synchronize 都会按 PR 整体 diff 触发 `v1alpha5-integration`，后续文档提交也实际跑过并通过；PR #48 分支当前没有 checks 上报（`gh pr checks 48` 无记录），其最终提交 `27324bc` 只改 `docs/design/**` 契约文字，因此记为 **NOT_RUN**，不写成 PASS。

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
| 20 | Capability 静态需求进入 Artifact 且精确选择 | `TestDevelopmentCapabilityRequirementInArtifact`、`TestDevelopmentCapabilitySelectionIsExact`、`TestDevelopmentCapabilityGoldenBaseline`、`TestDevelopmentCapabilityDeclarationIsRequired` |
| 21 | Capability 字段篡改（重封摘要后） | `TestDevelopmentVerifyRejectsCapabilityTampering`（删除需求、Capability ID、Catalog Key/Owner/Export/SourcePath/Pointer、Callable、声明位置，12 个子用例） |
| 22 | Package-scoped Schema 解析正反例 | `TestDevelopmentSchemaResolverPackageChain`、`…RejectsUndeclaredGlobalID`、`…RejectsUndeclaredAlias`、`…RejectsWrongPackage`、`…RejectsIllegalRelativeRef`、`…RejectsAmbiguousIdentity` |
| 23 | 多 Skill 共用 Lock | `TestDevelopmentMultiSkillSharedLockPasses`、`…MissingRootFails`、`…WrongPackageFails`、`…ExtraRootFails`、`…AliasIdentityConflictFails`、`…BrokenClosureFails`、`…ReorderingInvariant`、`TestDevelopmentSingleSkillRootsUnchanged` |

CI：实现提交 `2e2a5840d2972f17c46c1cb1a6249ab9c50df58e` 的 `v1alpha5-integration` 运行
<https://github.com/odango-chan/ca-skill-compiler/actions/runs/44717886662> PASS；其后文档提交的 PR synchronize 运行（如 <https://github.com/odango-chan/ca-skill-compiler/actions/runs/44719034235>）同样 PASS。代码在实现提交后未再变化。
PR #48 分支没有 checks 上报，其最终提交 `27324bcf273ae640acdf27637f9a7de89ed2ffe0` 只改 `docs/design/**` 契约文字，记为 **NOT_RUN**（不是 PASS）；本轮流程图的阶段与边界未变化，故未重绘 Archify 图，上一轮已提交的目视回执保持有效，人工签字仍为 PENDING。

## 4. 已知限制

- MAS **G0 = NOT_RUN**（没有通过 G0 的真实精确接口快照），**G1 = NOT_RUN**。本报告不宣称 MAS 通过。
- 所有正例仍为 Compiler 自有的**合成 Interface Package 快照**，不代表真实 Owner 发布或产品资格。CapabilityCatalog 的 Publisher 也是合成值；其真实签发权限仍由 Host `OWNER_TRUST` 验证。
- Development profile 的 Owner 归属来自每个包的单一 `publisher`，因此 Development 路径本身不构造多 Owner 快照；多 Owner 下 Descriptor 缺少精确归属的负例在共享的 owner-linking 层（`TestWP3B010MultiOwnerEvidenceGuaranteeMissingAttribution`）验证。真实多 Owner Interface Package 发布仍待 Provider/Owner 侧材料。
- PR #48 本轮只改契约文字（Capability 静态需求、Package-scoped Schema、Root 并集）：流程阶段与边界未变，故未重绘 Archify 图 01–03，既有图与截图检查保持原样；仅文档提交换了 HEAD。
- Owner 正式发布（真实 Owner 签名、真实 Interface Package 发布记录）**尚未完成**；本实现只消费调用方提供的精确快照与锁。
- Development Artifact 恒为 `NOT_ADMITTED`；生产 Host 必须拒收 `DEVELOPMENT` profile。
- 真实 Commit Proof、实际 Reconcile、运行时 Evidence、Deployment Grant、Provider Binding 与 Invocation 授权仍由 Host/Runtime 负责，本实现只记录其静态位置与契约。
- Archify 图 01、02 的**人工逐张目视签字仍为 PENDING**；自动化检查与代理人目视记录都不冒充人工签字。图 03 维持“已抽查”。
- v2.0.2 入口保持不变；本报告不构成发布、合并、冻结或打 Tag 的决定。最终判断只到 `READY FOR FINAL REVIEW`。

## 5. 最终职责矩阵（摘要）

本节的完整权威版本只由 PR #48 的[职责审计](https://github.com/odango-chan/ca-skill-compiler/blob/27324bcf273ae640acdf27637f9a7de89ed2ffe0/docs/design/compiler-evidence-assurance-responsibility-audit.zh-CN.md)（固定在最终 HEAD）与架构文档 §6 维护；下表只是便于实现评审的摘要，不构成第二份职责来源。

| 责任 | Compiler | Interface Package | Host Admission | Runtime / Provider | Owner / Assurance Authority |
|---|---|---|---|---|---|
| Evidence 声明/引用、参数绑定、SourcePath、Schema 类型 | 执行 | 提供快照 | 复核 | — | — |
| EvidenceDescriptor 结构与 Producer/路径 | 执行 | 作为正式导出承载 | 复核 | — | 发布声明 |
| EvidenceDescriptor 的 Owner 身份（`authority`） | 只做静态一致性并记录 `OWNER_TRUST` | 以正式导出声明 | 验证 `OWNER_TRUST`，不得用 Domain/Fact Trust 替代 | — | 承担 |
| Owner 真实身份、签发权限 | 不判定 | 声明 `publisher` | 验证 `OWNER_TRUST` | — | 承担 |
| IndependentDomains / CommonFact 授权 | 只检查声明形状 | — | 验证两类 TRUST | — | 授权 |
| Capability 静态选择与 Catalog Owner | 记录精确 Capability 需求（Package/Owner/Export/成员/声明位置）与 `OWNER_TRUST` | 以 `publisher` 发布 CapabilityCatalog | 验证 Owner 与部署授权 | — | 承担发布者身份 |
| Independence 组静态结构与域对枚举 | 执行 | 提供 Descriptor | 验证 `EVIDENCE_INDEPENDENCE_TRUST` | — | 授权域对 |
| Mutation / Commit Proof / Reconcile 静态契约 | 执行 | 提供 Callable/Schema | 复核 | 实际执行 | — |
| 实际提交、实际恢复结论 | 不产生、不伪造 | — | 不足以满足 | 产生并验证 `COMMIT_PROOF` / `RECONCILE` | — |
| Deployment Grant / Provider Binding / Invocation 授权 | 不读取 | — | 验证 | 执行期复核 | 部署方签发 |
| 产物执行状态 | 固定 `NOT_ADMITTED` | — | 拒绝未准入产物 | 仅执行已准入 Invocation | — |

完整矩阵见 PR #48 架构文档 §6。
