# Evidence / Assurance 职责审计与纠偏（Development Build）

**状态：纠偏实施记录。** 本页审计 Development Build 首轮实现中所有 Evidence 相关代码，按 Compile、Interface Link、Assurance、Host Admission、Runtime 五个边界重新归类，并记录本次最小调整。语言规范仍以冻结的 CA-SL v1alpha5（含继承的 v1alpha4 语义）为准；本页不新增、不删除、不弱化任何冻结语言规则。

## 1. 结论

| 模块 | 当前实现 | 应有归属 | 处理方式 |
|---|---|---|---|
| Evidence 声明与引用解析（`predicates.yaml` 的 `requires` / `independence`） | `validateEvidenceRequirements`、`collectDependencyReferences` 解析并检查别名/参数绑定 | Compiler（语言语义） | 保留 |
| Predicate Evidence 参数绑定、SourcePath、数据流、Schema 类型 | 继承静态流水线 `validatePredicateEvidence`、`validateSourcePath*`、`predicate_argument_typing` 等 | Compiler（语言语义） | 保留 |
| Evidence Descriptor 物理文档（authority/fact/producers/scope/correlation/factValue/selection/freshness） | Development 首轮放在独立 `assurance[]` 输入 | Interface Contract | **移入 Interface Package 的正式导出**，由 Dependency Lock 固定；删除 `assurance[]` |
| Evidence Descriptor 结构、Producer 身份/路径、Producer-Consumer 关系、Schema 引用 | `parseEvidenceDescriptor` + `validateEvidenceDescriptorReference` + `validateHostRecoveryDualProducer` | Compiler（静态接口语义） | 保留；作为 Declared Export 参与 Link，缺失即拒绝 Build |
| Evidence Independence 静态结构（成员存在、组形状、sourceIdentity 声明） | `validateIndependenceGroups`（源结构）+ `validateEvidenceIndependenceGroup`（声明形状） | Compiler | 保留静态部分 |
| Evidence Independence 的域授权与域对独立性 | `IndependentDomains` 信任输入 | Assurance / Host | 移出 Compiler BuildRequest；转为 Host Admission Requirement |
| CommonFact 命名空间/事实授权 | `EvidenceTrustBindings` 信任输入 | Assurance / Host | 移出；转为 Host Admission Requirement |
| sourceIdentity 域的真实来源与真实性 | 原 `assurance` 校验隐含 | Assurance / Host | 移出；Compiler 只检查声明形状 |
| Owner 真实身份、签发权限、授权链 | 首轮以包 `publisher` 建立接口身份并生成 `OWNER_TRUST` 义务 | Host Admission（Owner / Deployment Authority） | 包声明身份仍用于静态一致性；真实信任判定移出，转为 Host Admission Requirement |
| 运行时 Evidence 的 Correlation / Freshness 实值 | 无（Compiler 从不读取运行记录） | Runtime / Host | 明确不进入 BuildRequest；声明形状仍由 Compiler 检查 |
| Mutation 静态约束（Effect、receipt/proof Schema、bind、path、failure class、资源责任） | `mutation_contracts.go`、`mutation_positive.go`、`reconcile_contracts.go` | Compiler | 保留 |
| Mutation 是否真实提交、Commit Proof 是否可信 | 首轮生成 `COMMIT_PROOF` 未决义务 | Runtime / Host | 静态约束保留；真实判定转为 Runtime Requirement |
| Reconcile 静态契约（READ callable、proof Schema/path、recoverable output） | `reconcile_contracts.go`、路由/重试可达性检查 | Compiler | 保留 |
| Reconcile 实际执行与恢复结论 | 首轮生成 `RECONCILE` 未决义务 | Runtime / Host | 转为 Runtime Requirement |
| Invocation 是否可执行、Deployment Grant、Provider Binding | Compiler 不读取 | Host Admission | 保持 `NOT_ADMITTED`，不进入 Compiler 输入 |

## 2. 纠偏后的数据流

```
CA-SL source ───────────────┐
Interface Packages ─────────┤
  package.json exports      ├─► Compiler: compile + interface link + static evidence semantics
  Callable/Schema/Capability│        │
  Ingress/EvidenceDescriptor│        ├─ 静态失败 → Build 拒绝，无 Artifact
Dependency Lock ────────────┘        ├─ 静态通过 → Development Artifact (NOT_ADMITTED)
                                     │     ├─ manifest.json（精确锁、接口链接、文件摘要）
                                     │     ├─ host-admission-requirements.json
                                     │     └─ compiled Procedure / Ingress links
                                     │
Owner / Assurance Authority ─────────┼─► Host Admission: 身份、签发权限、域独立性、
Runtime records / Commit Proof ──────┘   真实 Commit Proof / Reconcile、Invocation 资格
```

## 3. 本轮实际调整

1. **输入契约**：删除 `assurance[]`；`EvidenceDescriptor` 成为 Interface Package 的正式导出类型（`{kind: "EvidenceDescriptor", id, path}`），描述字节由 Dependency Lock 的快照摘要固定。
2. **静态完整性**：Predicate 引用的 Evidence Descriptor 缺失、重复、结构非法、Producer/路径不匹配，均为 Build 失败（冻结 `EVIDENCE_GUARANTEE_LINK_INVALID` / `SCHEMA_MISSING` 等身份），不再用 `UNRESOLVED` 豁免。
3. **信任边界**：`IndependentDomains`、CommonFact、Owner 授权链不再作为 Compiler 输入；Compiler 只校验声明形状和静态一致性。需要外部判定的位置记录为 Host Admission Requirement。
4. **产物模型**：`obligations.json` 改为 `host-admission-requirements.json`；条目状态固定 `HOST_VERIFICATION_REQUIRED`，不再使用 `UNRESOLVED` / `DESCRIPTOR_PRESENT` 表述。静态 Evidence 检查结果不进入该文件（全部已由 Compiler 执行通过）。
5. **Commit Proof / Reconcile**：静态契约检查保持原样；真实提交与恢复结论作为 Runtime Requirement 记录。
6. **旧入口**：v2.0.2 的 BuildRequest/BuildResult/Artifact 与全部静态检查未改动；黄金基线逐字节一致。

## 4. 明确保留的静态检查

- Evidence 引用的 Alias、Guarantee ExportId、Predicate 参数绑定。
- Descriptor 的 authority/fact/producers/scope/correlation/factValue/selection/freshness 形状与静态引用根。
- Producer 身份与 ToolCatalog Callable、`provides.path` 的唯一匹配。
- Producer/Consumer 关系与 Host-recoverable Mutation 双 Producer 要求。
- Independence 组成员存在、参数绑定、sourceIdentity 声明形状。
- Evidence Schema 的类型与 `$ref` 闭包。
- Mutation receipt/commitProof/reconcile/recoverable 的 Schema、bind、path、failure class、资源责任、路由与重试可达性。

## 5. 明确不在 Compiler 内实现

- 任何 Assurance Authority 信任根、签名链、证书、撤销列表。
- IndependentDomains 授权、CommonFact 归属、Owner 签发权限。
- 真实 Evidence 记录、运行时 Freshness/Correlation。
- 真实 Commit Proof 判定、实际 Reconcile 结论、Invocation 准入。
- Deployment Grant、Provider Binding、Runtime 派发。

这些材料不进入 BuildRequest，也不改变 Artifact 的静态内容；Host 在派发前以受信材料补齐并重新校验语言要求。\n\n## 6. 设计文档同步\n\n本审计对应的设计更新已提交到 PR #48 分支 `docs/toolchain-architecture-vnext-20261119`，提交 `4a260be`：工具链架构、Development Build 实施契约和 Harness Goal 均已按本页边界修改；请求样例同步为 `EvidenceDescriptor` 导出形态。
