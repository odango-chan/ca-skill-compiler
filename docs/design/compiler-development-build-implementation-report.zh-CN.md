# Development Build 最终架构一致性收敛：实现与验收报告

**状态：Draft PR #51 评审版。** 本页记录 `feat/compiler-development-build-20261120` 上实现与 [PR #48 架构基线](https://github.com/odango-chan/ca-skill-compiler/pull/48) 的一致性证据。它不冻结公开契约、不合并、不发布。

| 项 | 值 |
|---|---|
| 设计契约提交（PR #48） | `551a0bba3455cfe4e7dbeede274b686a523316d6` |
| 本报告生成时 PR #48 头部 | `e333b79399827e391a8e74afbc693c4b572b7699` |
| 权威职责基线 | [`compiler-evidence-assurance-responsibility-audit.zh-CN.md`](compiler-evidence-assurance-responsibility-audit.zh-CN.md) |
| 产品版本 / 语言目标 | CA Skill Compiler v2.0.0（未改）/ `connectionagent.io/v1alpha5`（未改） |
| 冻结入口 | v2.0.2 BuildRequest / BuildResult / Artifact 逐字节未改 |
| MAS G0 / G1 | **NOT_RUN** |

## 1. 本轮实现范围

仅 `ca-skill-compiler`。未修改 ca-skills 冻结规范、Compiler v2.0.2 已发布入口/Artifact、ca-agent、ca-agent-demo、ca-client-tools、env-openamsm，也未改任何稳定 ID 或历史产物。

### 1.1 Requirement 两类分离

- 新增独立产物 `runtime-execution-requirements.json`；`host-admission-requirements.json` 保留原有 Host Admission 语义。
- 类别由 `kind` 唯一确定：
  - Host Admission（`HOST_VERIFICATION_REQUIRED`）：`OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST`。
  - Runtime Execution（`RUNTIME_EXECUTION_REQUIRED`）：`COMMIT_PROOF`、`RECONCILE`。
- `COMMIT_PROOF` / `RECONCILE` 由同一个 collector 自动路由到 Runtime 文件，不再进入 Host Admission 集合。
- `manifest.json` 新增 `requirements { hostAdmission, runtimeExecution }`，显式声明两个文件的契约身份。
- `VerifyDevelopment` 同时校验两个文件：JCS 规范化、排序、逐条重算 `requirementId`、kind/file/status 三者一致、independence 明细结构、规范化程度；随后以原始请求重建并逐字节比较全部产物。
- 实际 Commit Proof 与实际 Reconcile 结论不进入编译产物，Compiler 不输出任何“已提交/已恢复”的判定。

### 1.2 Evidence 多重 Requirement

- 将原先 `alias::exportID -> reason` 的单值映射改为 `alias::exportID -> set(reason)`。
- 同一 `EvidenceDescriptor` 同时声明 `sourceIdentity` 与 `factIdentity` 时，分别产生 `EVIDENCE_DOMAIN_TRUST` 与 `EVIDENCE_FACT_TRUST`，互不覆盖。
- 输入数组重排不改变产物字节（确定性测试覆盖）。

### 1.3 Independence 精确建模

- 每个 Independence Group 记录一条 `EVIDENCE_INDEPENDENCE_TRUST`，携带：

  ```
  independence { predicate, groupIndex, members[{dependencyAlias,key,exportId,sourcePath,domain}], domainPairs[[a,b],...] }
  ```

- 组位置精确为 `/predicates/<predicate>/evidence/independence/<index>`；不再使用 `env.roots[0]` 或单一布尔值。
- 每个成员解析到唯一 Descriptor、唯一 Interface Package key 和声明域；域对逐对列出并规范化排序。
- 组成员缺失、Descriptor 不唯一、缺少 `sourceIdentity`、少于两个声明域等静态非法情形直接 Build FAIL；只有外部 IndependentDomains 授权缺失才记录 Host Requirement。
- 多 Group、多 Root（一个组跨两个根包）、多 Domain Pair 均有组合测试。

### 1.4 IngressCatalog 精确身份

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
| Requirement 记录形状含 location/subjects/status，independence 另带明细 | 字段、排序、摘要域与文档一致 | 无 |
| 一个 Independence Group 一条 Requirement，精确 selector | 实现按组生成，selector 精确到组序号 | 无 |
| 多信任事实不互相覆盖 | reason 集合实现，两条并存 | 无 |
| Ingress 导出身份七条 | 七条全部在 `validateDevelopmentPackageExports` / `validateDevelopmentBindingTargets` 实施 | 无 |
| Artifact 恒 `NOT_ADMITTED`，生产 Host 拒收 `DEVELOPMENT` | 未改；测试固定执行状态 | 无 |
| Verify 同时检查两类并逐字节重建 | 实现 | 无 |

未发现架构文档与实现之间的未决差异。已知的非契约差异只有测试夹具的组织方式（`developmentRequestFromLegacy` 为测试专用转换，不构成对外接口）。

## 3. 测试报告（实际执行）

环境：Windows；Go **go1.26.6 windows/amd64**；Node **v24.18.0**；Google Chrome **158.0.8127.37**；Archify 渲染器 **3.2.0**。

| 门禁 | 命令 | 结果 |
|---|---|---|
| 单元/集成 | `GOWORK=off go test ./... -count=1` | PASS（全部包） |
| 静态检查 | `GOWORK=off go vet ./...` | PASS |
| Native build | `CGO_ENABLED=0 GOWORK=off go build ./...` | PASS |
| Go/WASM build | `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | PASS |
| CLI E2E | `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | PASS |
| Chromium Worker parity | `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | PASS（Development 5 文件逐字节一致） |
| 空白/编码 | `git diff --check` | PASS |
| v2.0.2 黄金基线 | `internal/compiler` legacy golden tests | PASS（逐字节一致） |

专项测试（`internal/compiler/development_requirements_test.go` 等）：

| # | 验收项 | 测试 |
|---|---|---|
| 1 | `COMMIT_PROOF` / `RECONCILE` 不出现在 Host Admission | `TestDevelopmentRequirementClassSeparation`、`TestDevelopmentMutationAndReconcileRequirements` |
| 2 | Runtime Requirement 精确覆盖静态 Mutation/Reconcile 契约 | `TestDevelopmentRequirementClassSeparation`（恰好 1 条 COMMIT_PROOF + 1 条 RECONCILE，位置完整） |
| 3 | 同一 Descriptor 两类信任要求同时存在 | `TestDevelopmentEvidenceTrustRequirementsAreIndependent`（source-only / fact-only / both） |
| 4 | 多 Independence Group、多 Root、多 Domain Pair | `TestDevelopmentIndependenceGroupsAreExact`（2 组 / 2 根 / 4 域 / 7 对组合，重排确定性）+ `TestDevelopmentIndependenceSameDomainFails` |
| 5 | IngressCatalog Export 精确身份 | `TestDevelopmentIngressExportIdentity`（错误 id、错误路径、缺失 operation、重复 operation、错误 Owner、未导出 operation） |
| 6 | 删除任意 Requirement 后重封外层摘要仍 Verify FAIL | `TestDevelopmentVerifyRejectsRequirementClassMigration` |
| 7 | 修改 Procedure / Lock / Descriptor / Requirement 后 Verify FAIL | `TestDevelopmentVerifyRejectsResealedTampering`、`TestDevelopmentVerifyRejectsSourceChanges`、`TestDevelopmentVerifyRejectsRequirementClassMigration` |
| 8 | Native / WASM / Chromium 结果一致 | Chromium parity 门禁（build/status/diagnostics/artifactDigest/outputFiles 全部一致） |

CI：PR #51 的 checks 页面 <https://github.com/odango-chan/ca-skill-compiler/pull/51/checks>。PR #48 的文档与图无关 CI 见其 checks 页面。

## 4. 已知限制

- MAS **G0 = NOT_RUN**（没有通过 G0 的真实精确接口快照），**G1 = NOT_RUN**。本报告不宣称 MAS 通过。
- 所有正例仍为 Compiler 自有的合成 Interface Package 快照，不代表真实 Owner 发布或产品资格。
- Development Artifact 恒为 `NOT_ADMITTED`；生产 Host 必须拒收 `DEVELOPMENT` profile。
- 真实 Commit Proof、实际 Reconcile、运行时 Evidence、Deployment Grant、Provider Binding 与 Invocation 授权仍由 Host/Runtime 负责，本实现只记录其静态位置与契约。
- v2.0.2 入口保持不变；本报告不构成发布或合并决定。

## 5. 最终职责矩阵（摘要）

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
