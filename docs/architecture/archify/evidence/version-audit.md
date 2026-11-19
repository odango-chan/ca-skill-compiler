# 3GPP 规范版本差异审计

> 目的：区分**当前实现实际使用的 OpenAPI 版本**与**业务设计文档采用的规范目标版本**，
> 并说明哪些结论已核对、哪些仍未核对。未完成核对的部分不得宣称版本一致。

## 1. 已核对的版本事实

### 1.1 实现侧（env-openamsm @ `594dec78`）

`upstream/openamsm/lib/sbi/support/r19-20260402-openapitools-7.20.0/` 下的 OpenAPI 声明：

| 文件 | `externalDocs.description` 声明的规范版本 | `info.version` |
|---|---|---|
| `TS29502_Nsmf_PDUSession.yaml` | **3GPP TS 29.502 V19.5.0** | `1.4.0` |
| `TS29518_Namf_Communication.yaml` | **3GPP TS 29.518 V19.6.0** | `1.4.1` |

同一 OpenAPI 在 `standard/`、`modified/`、`patched/` 三个目录中都声明相同的规范版本。

### 1.2 设计文档侧

| 文档 | 引用版本 |
|---|---|
| `amf-smf-business-procedure-flowbook-scp-only.zh-CN.md`（头部与 §7） | TS 29.502 **V19.7.0**、TS 29.518 **V19.8.0**、TS 23.502 **V19.9.0**、TS 29.500 V19.7.1 |
| `ca-skills` `establish-pdu-session-mas/SKILL.md` 的 3GPP Traceability | 同上（29.502 V19.7.0 / 29.518 V19.8.0 / 23.502 V19.9.0） |
| `amf-smf-sbi-interface-inventory.zh-CN.md`（规范锁说明） | 29.502 V19.7.0、29.518 V19.8.0 |

### 1.3 差异

| 规范 | 实现 | 设计目标 | 差异 |
|---|---|---|---|
| TS 29.502（Nsmf_PDUSession） | V19.5.0 | V19.7.0 | 落后 2 个小版本 |
| TS 29.518（Namf_Communication） | V19.6.0 | V19.8.0 | 落后 2 个小版本 |
| TS 23.502（Stage 2 流程） | 不适用（Stage 2 无 OpenAPI） | V19.9.0 | 实现侧只有 Stage 3 |

## 2. 已完成的结构性核对

### 2.1 同一版本内 `standard` 与 `modified` 的 API 面比对

`env-openamsm` 对 r19 OpenAPI 做过本地修改。为避免"本地改动悄悄改了接口"这一风险，做了集合比对：

| 文件 | `standard` 的 path+operationId 条目 | `modified` 的条目 | 差异 |
|---|---|---|---|
| `TS29502_Nsmf_PDUSession.yaml` | 16 | 16 | **0（集合完全相同）** |
| `TS29518_Namf_Communication.yaml` | 23 | 23 | **0（集合完全相同）** |

结论：**V19.5.0 / V19.6.0 基线内的本地修改没有改变 path 与 operationId 集合**。
（字段、枚举与描述层面存在差异，本轮未逐字段比对。）

### 2.2 图集中使用的方法/路径/operationId（均来自 V19.5.0 / V19.6.0 实现）

| Service Operation（3GPP §5.2） | operationId | HTTP | URI（相对 apiRoot） |
|---|---|---|---|
| `Nsmf_PDUSession_CreateSMContext` | `PostSmContexts` | POST | `/nsmf-pdusession/v1/sm-contexts` |
| `Nsmf_PDUSession_UpdateSMContext` | `UpdateSmContext` | POST | `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/modify` |
| `Nsmf_PDUSession_ReleaseSMContext` | `ReleaseSmContext` | POST | `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/release` |
| `Nsmf_PDUSession_RetrieveSMContext` | `RetrieveSmContext` | POST | `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/retrieve` |
| `Nsmf_PDUSession_SendMOData` | `SendMoData` | POST | `/nsmf-pdusession/v1/sm-contexts/{smContextRef}/send-mo-data` |
| `Namf_Communication_N1N2MessageTransfer` | `N1N2MessageTransfer` | POST | `/namf-comm/v1/ue-contexts/{ueContextId}/n1-n2-messages` |
| `Namf_Communication_N1N2TransferFailureNotification` | `N1N2TransferFailureNotification` | POST | 由 AMF 回送到 SMF 提供的 callback URI |

说明：

- Service Operation 名称不是 OpenAPI 里的机器可读字段，它来自 TS 29.502 §5.2 / TS 29.518 §5.2 的
  操作表；OpenAPI 只提供 operationId、method 与 path。图中两者**分行标注**，不互相替代。
- `N1N2TransferFailureNotification` 在 OpenAPI 中定义在 `callbacks` 段，没有静态 path；
  因此图中只标注 operationId 与 callback 语义，不虚构 URI。
- OpenAMSM 私有 API `Nsmf_MasOrchestration`（`masOrchestrationInvoke`、
  `smContextUpdateSync`）来自 `tools/nw-agent/openapi/Nsmf_MasOrchestration.yaml` v1.0.0，
  **不是** 3GPP 的 `Nsmf_PDUSession` Service，图中始终标为 OpenAMSM Private API。

## 3. 尚未完成的核对（BLOCKED）

| 事项 | 状态 | 原因 |
|---|---|---|
| V19.5.0 → V19.7.0 的 Service Operation / operationId / URI 变化 | **BLOCKED** | 本机只有 V19.5.0 基线（r19-20260402）；无 V19.7.0 的 OpenAPI 副本，且 3GPP 归档需要授权访问，无法离线比对 |
| V19.6.0 → V19.8.0 的同上变化 | **BLOCKED** | 同上 |
| 上述版本间的字段增删与枚举变化 | **BLOCKED** | 同上 |
| TS 23.502 V19.9.0 对应条款的逐步核对 | **BLOCKED** | Stage 2 文本不在仓库内；图中 23.502 步骤引用沿用 flowbook 的既有引用，标为设计目标 |

因此本图集的处理方式是：

1. **只用已核对的实现版本**（V19.5.0 / V19.6.0）标注 operationId、method 与 URI；
2. 设计目标版本（V19.7.0 / V19.8.0 / 23.502 V19.9.0）只作为**引用版本**出现在卡片与附页中，
   不用于断言字段级一致性；
3. 对未经核对的字段一律不写入图形，避免"未经核实的字段被画成事实"。

## 4. 对图形的影响

- 若 V19.7.0 / V19.8.0 改变了上述 method、path 或 operationId，图中标注需同步更新；
  由于与本轮结论无冲突证据，当前标注对 V19.5.0 / V19.6.0 **成立**。
- Service Operation 名称本身在 Release 19 内保持稳定，风险主要在字段与可选性层面，
  而图集不表达字段级契约，因此本次绘图的可信度不受该 BLOCKED 项影响。
- 该 BLOCKED 项仍是**实现升级前的必办事项**，归属 `env-openamsm`（OpenAPI 升级）与
  `ca-skills`（引用版本同步）。
