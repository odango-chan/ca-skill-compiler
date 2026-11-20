# Evidence / Assurance 职责审计与纠偏（Development Build）

**状态（2026-11-20）：职责基线已通过最终审查并随架构 [PR #48](https://github.com/odango-chan/ca-skill-compiler/pull/48) 合入 `main`（`22860a2`），对应实现 [PR #51](https://github.com/odango-chan/ca-skill-compiler/pull/51) 已合并（`1d7ea81`）；公开契约未另行冻结或发布。** 本页仍是 Architecture → Audit → Pilot → Implementation 中 Evidence / Assurance 职责归属的单一基线，不增加、删除或弱化冻结的 CA-SL v1alpha5 规则。原评审阶段 SHA 与未决人工/外部验收事项仍如实保留。

## 1. 结论

| 模块 | 当前实现 | 应有归属 | 处理方式 |
|---|---|---|---|
| Evidence 声明与引用解析（`predicates.yaml` 的 `requires` / `independence`） | `validateEvidenceRequirements`、`collectDependencyReferences` 解析并检查别名/参数绑定 | Compiler（语言语义） | 保留 |
| Predicate Evidence 参数绑定、SourcePath、数据流、Schema 类型 | 继承静态流水线 `validatePredicateEvidence`、`validateSourcePath*`、`predicate_argument_typing` 等 | Compiler（语言语义） | 保留 |
| Evidence Descriptor 物理文档（authority/fact/producers/scope/correlation/factValue/selection/freshness） | Development 首轮放在独立 `assurance[]` 输入 | Interface Contract | **移入 Interface Package 的正式导出**，由 Dependency Lock 固定；删除 `assurance[]` |
| Evidence Descriptor 结构、Producer 身份/路径、Producer-Consumer 关系、Schema 引用 | `parseEvidenceDescriptor` + `validateEvidenceDescriptorReference` + `validateHostRecoveryDualProducer` | Compiler（静态接口语义） | 保留；作为 Declared Export 参与 Link，缺失即拒绝 Build；Producer 输出 Schema 通过其 Callable 解析，Descriptor 不复制接口 Schema |
| Evidence Independence 静态结构（成员存在、组形状、sourceIdentity 声明） | `validateIndependenceGroups`（源结构）+ `validateEvidenceIndependenceGroup`（声明形状） | Compiler | 保留静态部分，并按组展开为精确 Requirement |
| Evidence Independence 的域授权与域对独立性 | `IndependentDomains` 信任输入 | Assurance / Host Admission | 移出 Compiler BuildRequest；转为 `EVIDENCE_INDEPENDENCE_TRUST` Host Admission Requirement |
| CommonFact 命名空间/事实授权 | `EvidenceTrustBindings` 信任输入 | Assurance / Host Admission | 移出；转为 `EVIDENCE_FACT_TRUST` Host Admission Requirement |
| sourceIdentity 域的真实来源与真实性 | 原 `assurance` 校验隐含 | Assurance / Host Admission | 移出；转为 `EVIDENCE_DOMAIN_TRUST`；Compiler 只检查声明形状 |
| Owner 真实身份、签发权限、授权链 | 以包 `publisher` / Descriptor `authority` 建立接口身份，并按真实消费位置生成 `OWNER_TRUST` | Host Admission（Owner / Deployment Authority） | 包声明身份仍用于静态一致性；真实信任判定移出，转为 `OWNER_TRUST` Host Admission Requirement。EvidenceDescriptor 自身的 Owner Trust 不能被 `EVIDENCE_DOMAIN_TRUST` / `EVIDENCE_FACT_TRUST` 替代 |
| 运行时 Evidence 的 Correlation / Freshness 实值 | 无（Compiler 从不读取运行记录） | Runtime / Host | 明确不进入 BuildRequest；声明形状仍由 Compiler 检查 |
| Mutation 静态约束（Effect、receipt/proof Schema、bind、path、failure class、资源责任） | `mutation_contracts.go`、`mutation_positive.go`、`reconcile_contracts.go` | Compiler | 保留 |
| Mutation 是否真实提交、Commit Proof 是否可信 | 首轮记为待决义务 | Runtime / Host | 静态约束保留；真实判定转为 `COMMIT_PROOF` Runtime Execution Requirement |
| Reconcile 静态契约（READ callable、proof Schema/path、recoverable output） | `reconcile_contracts.go`、路由/重试可达性检查 | Compiler | 保留 |
| Reconcile 实际执行与恢复结论 | 首轮记为待决义务 | Runtime / Host | 转为 `RECONCILE` Runtime Execution Requirement |
| Assurance Attestation（签发者声明的保证结论） | 不在 Compiler 内 | Owner / Assurance Authority → Host | 独立信任产物；绝不进入 Compiler 输入 |
| Invocation 是否可执行、Deployment Grant、Provider Binding | Compiler 不读取 | Host Admission | 保持 `NOT_ADMITTED`，不进入 Compiler 输入 |

## 2. 最终两类 Requirement 模型

Development Build 只产出**两类互不替代**的 Requirement。二者都是 Compiler 对「哪些事实已静态成立、哪些事实必须由外部权威补齐」的确定性记录，都不是 Compiler 的通过结论，也都不是执行授权。

### 2.1 分类

| 类别 | 满足时机 | 产物文件 | 固定状态 |
|---|---|---|---|
| Host Admission Requirement | 派发之前（Build 之外，由 Host 用独立信任根验证外部事实） | `host-admission-requirements.json` | `HOST_VERIFICATION_REQUIRED` |
| Runtime Execution Requirement | 实际执行期间（由 Runtime / Provider 在执行中产生并验证） | `runtime-execution-requirements.json` | `RUNTIME_EXECUTION_REQUIRED` |

两类 Requirement 必须分别成文件、分别计算、分别校验。**Host Admission Requirement 恰好只覆盖四类**派发前可验证的外部信任事实：`OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST`。Deployment Grant、Provider Binding 与当前 Invocation 授权**同属 Host Admission 检查，但不是 Host Admission Requirement 条目**：它们同样在派发前由 Host 用独立信任根校验，但 Compiler 无法从源码定位，因此不进入 `host-admission-requirements.json`、不进入 BuildRequest、不属于本页定义的任何一个 Requirement kind。

其中只有前四类会由 Compiler 依据源码/接口位置写成 `host-admission-requirements.json` 条目；Deployment Grant、Provider Binding 与当前 Invocation 授权由 Host 按其独立信任根在派发时校验，**不是** Compiler 可识别、可定位或可伪造的产物字段。`runtime-execution-requirements.json` 同理只记录 Compiler 能定位的执行期保证位置，不记录实际执行结果。

`Runtime Execution Requirement` 是独立的一类，只有两个 kind：

| kind | 含义 | 执行期必须真正发生的事 |
|---|---|---|
| `COMMIT_PROOF` | Mutation 的真实提交与提交证明核验 | 实际提交发生，并核验与该次 Mutation 绑定的 proof |
| `RECONCILE` | `UNKNOWN` 之后对**精确原尝试**的恢复与实际恢复结论 | 对原 occurrence 执行 Reconcile，并给出实际结论 |

`COMMIT_PROOF` 与 `RECONCILE` 不得被描述为「可在 Build 阶段或派发前满足的 Host Admission Requirement」。它们的真实提交结果与恢复结论**永不进入已编译产物**，也不由 Compiler 伪造或预测。Runtime / Host 必须分别区分：

1. **运行时机制支持检查**：Contract、Schema、path、READ Callable、路由与重试结构是否就位（静态可判定，Compiler 已执行；Host 可复核）。
2. **实际执行结果**：这一次 Mutation 到底提交没有、这一次 Reconcile 到底得出什么结论（只能由 Runtime / Provider 在执行期给出）。

把二者混为一谈，正是把「机制存在」当成「副作用已发生」的错误。

两类 Requirement 的语义边界必须同时成立：

- **Runtime Execution Requirement 不是外部授权。** 它不是 Grant、不是 Attestation、也不是 Host 放行判定；Host Admission Requirement 同样不授权执行。
- **实际 Commit Proof 不在 Build 阶段产生**，**实际 Reconcile 结论也不在 Build 阶段产生**；两者只能由 Runtime / Provider 针对具体 Invocation 与具体 Operation occurrence 产生。
- **Compiler 只检查静态契约并记录执行期要求**：机制是否就位可以静态判定，副作用是否真的发生不能。
- **Host Admission 不得替代实际 Commit Proof。** 派发前外部事实全部成立，也不代表这一次 Mutation 已经提交；`UNKNOWN` 之后的恢复结论只能来自经授权的 exact-original Reconcile。
- **Build 成功不代表 Artifact 已获得执行资格。** 成功只表示静态语言契约、接口闭包与确定性产物成立；执行资格由 Host 对具体 Invocation 逐次准入。

### 2.2 Requirement 记录形状

```
{
  requirementId,
  kind,
  location { skill, path, node, stage, selector },
  subjects [ { key { packageId, version, snapshotDigest }, id, sourcePath, schemaIds } ],
  status
}
```

- `location.selector` 为源码 JSON Pointer（含列表序号）；不可应用的 `node` / `stage` 用空串。
- `subjects` 按元素 JCS 字节稳定排序并去重；每项 `schemaIds` 排序去重，Schema 字节由锁与闭包固定。
- 相同 Producer / 消费在多条等价控制流边复现时去重。
- Evidence 引用位置必须精确：`requires` 写作 `/predicates/<predicate>/evidence/requires/<parameter>/<index>`，Independence 成员写作 `/predicates/<predicate>/evidence/independence/<group>/members/<member>/guarantee`。同一个 Descriptor 被多个 Predicate、多个参数、多个序号或 `requires`/`independence` 双重要用时逐一记录，不因内容相同而合并；`OWNER_TRUST` 按实际消费位置记录。
- `EVIDENCE_INDEPENDENCE_TRUST` 记录另带可选的 `independence` 明细（见 2.3）。
- `requirementId` = `sha256:` + SHA-256(`CA-SKILL-DEVELOPMENT-REQUIREMENT\0v1\0` + 类型化键的 JCS 字节)，`kind` 参与类型化键，**包含 `independence` 明细**。因此不同组、不同成员或不同域对必然得到不同 ID；删改明细无法靠复算外层摘要掩盖。该摘要域是**两类共用的唯一域**且类名中性：Host 与 Runtime 的隔离由 `kind`、所在文件与 `status` 三者共同判定，不能靠摘要前缀区分——若沿用 Host 专属前缀，Runtime Execution Requirement 会被误读成 Host Admission 命名空间的一部分。Verify 必须能拒绝旧摘要域或任意错误摘要域产生的条目。
- 文件内容为按 `requirementId` 排序的 JCS JSON 数组；`status` 按 2.1 固定。
- 顺序语义：语义上无序的输入（文件、快照、锁、映射顺序）规范化后不得改变产物字节；但真实源码列表序号是位置本身，序号变化允许改变 `requirementId` 与产物摘要。不得为维持错误的顺序无关性而丢弃真实位置。

### 2.3 Independence 明细与多组规则

`EVIDENCE_INDEPENDENCE_TRUST` 除去键字段外另带：

```
independence {
  groupIndex,
  members [ { dependencyAlias, key, exportId, sourcePath, domain } ],
  domainPairs [ [domainA, domainB], ... ]
}
```

`independence` 不含 `predicate` 字段：Predicate 身份只由 `location.node` 表达，`selector` 固定为 `/predicates/<predicate>/evidence/independence/<index>`。这样组身份只有一个权威来源，不会出现 `location.node` 与 `independence.predicate` 互相矛盾的第二份真相。

规则：

1. **一个 Independence Group 一条 Requirement**，不得压成一条全局记录。
2. 组来源位置精确到 `selector=/predicates/<predicate>/evidence/independence/<index>`。
3. 成员逐项列出：每个成员 Descriptor、每个 Source Domain、每个被授权的 Domain Pair、每个精确 Interface Package key，全部写入该条的 `subjects` / `independence`。
4. **不使用 `env.roots[0]`，也不使用单一全局布尔值**。多组、多根、多域对互相独立；组之间不得共享一条「独立性已满足」的结论。
5. 组的静态结构非法（成员缺失、组形状非法、绑定不成立）是 **Build FAIL**；只有「外部域授权尚缺」才是 Host Admission Requirement。
6. Verify 必须校验 `independence` 的每个成员、每个域、每个域对与组成员序号，并从统一摘要域重新推导 `requirementId`。

### 2.4 多信任事实不互相覆盖

同一个 EvidenceDescriptor 同时声明 `sourceIdentity` 与 `factIdentity` 时，**同时**产生 `EVIDENCE_DOMAIN_TRUST` 与 `EVIDENCE_FACT_TRUST` 两条独立 Requirement，任一条都不得覆盖另一条。语义上无序的输入（文件顺序、快照顺序、锁与映射顺序）不得改变产物字节或 Requirement 集合；Predicate 在 `predicates` 映射中的顺序同样不承载语义。但 `requires` 列表中的真实序号属于源码位置本身：列表顺序变化导致序号变化时，允许对应 `requirementId` 与产物摘要变化，不得为了错误的顺序无关性而丢失真实位置。

**Owner Trust 独立于 Domain / Fact Trust。** 每个实际消费的 EvidenceDescriptor 都按其 `authority` 与锁定快照路径产生自己的 `OWNER_TRUST`；`EVIDENCE_DOMAIN_TRUST` 只授权 `sourceIdentity` 的来源域，`EVIDENCE_FACT_TRUST` 只授权 CommonFact。三者是三个不同的外部事实，任何一条都不得被另一条替代，也不得因为 Descriptor 已声明 `sourceIdentity` / `factIdentity` 就省略 Owner 身份验证。Descriptor 的 `OWNER_TRUST` 同样使用真实消费位置（`requires` 参数序号或 Independence 成员位置），而不是粗粒度的 `evidence` 块指针。

### 2.5 静态失败与外部缺失的分界

- 声明或引用的静态结构非法（Descriptor 缺失、重复、结构非法、Producer/路径不匹配、Producer 输出 Schema 无法解析或与 Predicate 参数 Schema 不兼容、Independence 组形状非法、Ingress 导出身份不成立、Mutation/Reconcile 静态契约不成立）⇒ **Build FAIL，无产物**。不得用任何「未决」状态豁免。
- 静态结构成立、仅缺外部授权或外部事实 ⇒ **Build 成功**，对应可定位位置记入两类 Requirement 文件。缺 Deployment Grant、Provider Binding、Invocation 授权不进入 Compiler 输入，也不阻止 Build，更不写成条目。
- **「静态 Descriptor 缺失」与「运行时 Evidence 缺失」不是同一种缺失。** 前者意味着源码引用所需的接口契约不闭合，是静态可判定的 Build FAIL；后者意味着外部信任/执行记录尚未产生，Build 仍成功、状态恒为 `NOT_ADMITTED`。Development Build 可以用合成但精确的 Interface 快照补齐静态 Descriptor，但不能用实际 Runtime Evidence 或 Assurance Attestation 填补静态接口缺口，因为这两类材料从不进入 BuildRequest。
- **静态 Descriptor 齐备不代表真实 Evidence 已被验证。** Compiler 只证明证据保证的声明者、Producer/provides.path、事实与 Scope/Correlation/Selection/Freshness 的静态语义；实际来源、时效、运行时 Scope 与 Correlation 由 Runtime 对具体 Invocation 验证，外部信任材料由 Host Admission 验证。

## 3. Compiler 输入契约与静态/外部边界

### 3.1 输入只有三类

Compiler 的输入**只有**：

1. CA-SL 源字节；
2. 精确 Interface Package 快照（`package.json` 导出索引 + 各导出载荷）；
3. Dependency Lock。

没有可选的 `assurance[]` 输入，也没有「可选的 Assurance Export」输入。Assurance Attestation 是外部信任产物，绝不进入 Compiler。

### 3.2 EvidenceDescriptor 是正式导出

`EvidenceDescriptor` 是 Interface Package 的正式导出类型 `{kind: "EvidenceDescriptor", id, path}`，由 Dependency Lock 固定。Predicate 引用的 Producer 或路径缺失、重复、结构非法或不匹配 ⇒ Build FAIL。静态 Descriptor 与 Assurance Attestation 是两件事：前者是接口契约的一部分，后者是外部信任声明。

### 3.3 IngressCatalog 导出身份

IngressCatalog 导出的身份必须由 `package.json` 导出索引精确绑定到载荷，而不是「文件存在」或「apiVersion 正确」：

1. 载荷 `apiVersion` 与 `kind` 与声明一致。
2. 导出 `id` 精确等于载荷 `spec.operations[].id` 中的**恰好一项**。
3. 导出 `path` 精确等于实际 catalog 文件路径。
4. Operation ID 在文件中唯一。
5. catalog `metadata.owner` 等于包 `publisher`。
6. 每个声明的 operation 都在**同一路径**有精确导出：不得有未导出的 operation，也不得出现 id/path 漂移。
7. `artifactBindings` / `schemaBindings` 必须指向精确的包/依赖快照路径和已声明 Schema 身份。

任一条不成立即 Build FAIL。

### 3.4 产物与准入状态

Development Artifact 的执行状态恒为 `NOT_ADMITTED`。生产 Host 拒绝 `DEVELOPMENT` profile；隔离测试 Host 只有在显式 opt-in 时才可接收。Build 成功既不等于授权，也不等于资格结论（Qualification）。

## 4. 纠偏后的数据流

```
CA-SL source ───────────────┐
Interface Packages ─────────┤
  package.json exports      ├─► Compiler: compile + interface link + static evidence semantics
  Callable/Schema/Capability│        │
  Ingress/EvidenceDescriptor│        ├─ 静态结构非法 → Build FAIL，无 Artifact
Dependency Lock ────────────┘        ├─ 静态通过 → Development Artifact (NOT_ADMITTED)
                                     │     ├─ manifest.json（精确锁、接口链接、文件摘要）
                                     │     ├─ host-admission-requirements.json
                                     │     │     status=HOST_VERIFICATION_REQUIRED
                                     │     ├─ runtime-execution-requirements.json
                                     │     │     status=RUNTIME_EXECUTION_REQUIRED
                                     │     └─ compiled Procedure / Ingress links
                                     │
Owner / Assurance Authority ─────────┼─► Host Admission（派发前）:
Runtime Provider / 部署清单 ─────────┤     OWNER_TRUST · EVIDENCE_DOMAIN_TRUST
Deployment Authority ───────────────┤     EVIDENCE_FACT_TRUST · EVIDENCE_INDEPENDENCE_TRUST
Invocation 上下文 ──────────────────┘     Deployment Grant · Provider Binding · Invocation 授权

Runtime / Provider 执行期 ─────────────► COMMIT_PROOF（真实提交与证明核验）
                                          RECONCILE（精确原尝试恢复与实际结论）
                                          结果不回写 Compiler 产物
```

## 5. 保留的静态检查、移出的职责与 Verify 覆盖

### 5.1 明确保留的静态检查

- Evidence 引用的 Alias、Guarantee ExportId、Predicate 参数绑定。
- Descriptor 的 authority/fact/producers/scope/correlation/factValue/selection/freshness 形状与静态引用根。
- Producer 身份与 `CallableCatalog` 导出中 Callable 的精确匹配、`provides.path` 的唯一匹配（`ToolCatalog` 只是该导出载荷的冻结逻辑 `kind`）。
- Producer/Consumer 关系与 Host-recoverable Mutation 双 Producer 要求。
- Independence 组成员存在、参数绑定、sourceIdentity 声明形状、组形状与组序号。
- IngressCatalog 导出身份（§3.3 七条）。
- Evidence Schema 的类型与 `$ref` 闭包。
- Mutation receipt/commitProof/reconcile/recoverable 的 Schema、bind、path、failure class、资源责任、路由与重试可达性。

### 5.2 明确不在 Compiler 内实现

| 职责 | 归属 |
|---|---|
| 真实 Owner 身份与签发权限 | Owner |
| IndependentDomains 授权、CommonFact 授权 | Owner / Assurance Authority |
| 运行时 Evidence 记录、Correlation / Freshness 实值 | Runtime / Host |
| 实际 Commit Proof 判定 | Runtime / Provider |
| 实际 Reconcile 结论 | Runtime / Provider（经授权） |
| Deployment Grant 签发与撤销 | Deployment Authority → Host |
| Provider Admission、Invocation 授权、派发 | Host / Runtime |

这些材料不进入 BuildRequest，也不改变已编译产物的静态内容；Host 在派发前以受信材料补齐并重新校验语言要求。

### 5.3 Verify（Development）覆盖

Verify 必须同时检查**两个** Requirement 文件：

1. 两份文件都必须严格解码：拒绝未知字段、非数组文档、重复 Requirement 与未按 `requirementId` 排序的条目。
2. 规范化 JCS、逐条从**统一摘要域**重新推导 `requirementId`、`kind` 与所在文件类别一致、`status` 正确、`subjects` / `independence` 已规范化、覆盖完整。
3. `location` 必须是该 kind 的真实位置：Independence 恰好指向一个组 JSON Pointer，Evidence 信任要求（含 Descriptor 的 `OWNER_TRUST`）指向 `location.node` 所指 Predicate 的 Evidence 中真实引用位置；粗粒度、悬空或借用其他 Predicate 的位置一律拒绝。
4. 以原始请求重建全部产物并**逐字节**比较每个产物文件；不能只依赖外层摘要。

因此：删除任意一条 Host 或 Runtime Requirement、把 `COMMIT_PROOF` 移入 Host 文件、把 `OWNER_TRUST` 移入 Runtime 文件、改动 `status` / `requirementId` / `location` / independence 成员 / `domainPairs`，即使重新封装 manifest 摘要仍然失败；改动 Procedure、Lock 或 Descriptor 同样失败。

## 6. 设计提交与本轮评审记录

- 本页职责基线的设计提交：`551a0bba3455cfe4e7dbeede274b686a523316d6`（`docs(design)`，见 PR #48）。本轮收敛在该提交之上继续修改，最终 HEAD 以 PR #48 的最后一个提交为准，并由 PR #51 的实施报告精确引用。
- 本页与 PR #48 架构文档处于**最终评审**状态，不是已冻结：冻结由最终评审者在两个 PR 都通过后宣布。
- PR #51 的实现与本页契约一一对应；本轮同时修正了 Requirement ID 摘要域、Evidence 引用定位、Independence 明细结构与文档状态语义。
- 本页是 Architecture → Audit → Pilot → Implementation 链条中的职责权威来源；其他三份文档引用本页，不重复维护相互冲突的职责矩阵。
- 产物执行状态恒为 `NOT_ADMITTED`；生产 Host 拒收 `DEVELOPMENT`；MAS G0 / G1 仍为 **NOT_RUN**，本文不构成 MAS 通过结论。

## 7. 旧入口与回归边界

- 输入契约：删除 `assurance[]`；`EvidenceDescriptor` 成为 Interface Package 的正式导出类型，描述字节由 Dependency Lock 的快照摘要固定。
- 产物模型：旧的单一「义务」文件被 `host-admission-requirements.json` 与 `runtime-execution-requirements.json` 取代，状态分别固定为 `HOST_VERIFICATION_REQUIRED` / `RUNTIME_EXECUTION_REQUIRED`。
- 旧入口：v2.0.2 的 BuildRequest/BuildResult/Artifact 与全部静态检查未改动；黄金基线逐字节一致。
