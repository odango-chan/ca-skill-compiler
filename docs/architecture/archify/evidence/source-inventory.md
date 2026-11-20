# 事实基线清单（含逐文件 SHA-256）

> 图形中的每个名称、步骤和协议都来自下表来源。未在此列出的内容不作为事实依据。

## 1. 仓库与提交

| 仓库 | 引用 | 提交 | 用途 |
|---|---|---|---|
| `odango-chan/ca-skill-compiler` | 本分支 `docs/toolchain-architecture-vnext-20261119`（PR #48） | `fd40ad0a`（本轮图形源与工作流输入） | 图形、简报、文档落点 |
| `odango-chan/ca-agent-demo` | 分支 `docs/mas-v1alpha5-temporal-e2e-review-20261118`（PR #217） | `6168912e` | 业务流程全集、SBI 接口全集、Host 架构、边界文档 |
| `odango-chan/ca-skills` | `main` | `05be9ee8` | v1alpha5 正式评审示例 `establish-pdu-session-mas` |
| `odango-chan/ca-client-tools` | `feat/nas5gsm-codec-20261119` | `e2922ca4` | MAS Owner 候选契约 `v0.2.0`（NOT_QUALIFIED） |
| `odango-chan/env-openamsm` | `main` | `594dec78` | OpenAMSM OpenAPI、OES plans、Nsmf handler |

本轮 05a–05g 的源文件与工作流输入提交为 `fd40ad0a381473440ff6bae0a2a8d98f7c534810`；GitHub Actions run [#44663313398](https://github.com/odango-chan/ca-skill-compiler/actions/runs/44663313398) 生成 HTML 与浏览器证据后，产物提交为 `2cb58957909b74286cfaf56726b2fc66028df991`。

## 2. 逐文件 SHA-256（前 16 位）

| 文件 | 仓库 @ 提交 | SHA-256 |
|---|---|---|
| `examples/skills/establish-pdu-session-mas/assurance/procedure.yaml` | ca-skills @ 05be9ee8 | `cda8da9921533882` |
| `examples/skills/establish-pdu-session-mas/assurance/contract.yaml` | ca-skills @ 05be9ee8 | `10071303e22b90d9` |
| `contracts/mas-owner-candidate/v0.2.0/source-map.json` | ca-client-tools @ e2922ca4 | `6159d44ac5b09183` |
| `upstream/openamsm/tools/nw-agent/openapi/Nsmf_MasOrchestration.yaml` | env-openamsm @ 594dec78 | `096cf15a0c87ee85` |
| `upstream/openamsm/lib/sbi/support/r19-20260402-openapitools-7.20.0/modified/TS29502_Nsmf_PDUSession.yaml` | env-openamsm @ 594dec78 | `1ace04891e89017a` |
| `upstream/openamsm/lib/sbi/support/r19-20260402-openapitools-7.20.0/modified/TS29518_Namf_Communication.yaml` | env-openamsm @ 594dec78 | `dbdc656ce986d4b4` |
| `upstream/openamsm/tools/nw-agent/plans/modify/pdu-mod-v3.yaml` | env-openamsm @ 594dec78 | `be6d7b800ad088cf` |
| `upstream/openamsm/tools/nw-agent/plans/release/pdu-rel-v4.yaml` | env-openamsm @ 594dec78 | `2cfbbf27735bb7b5` |
| `upstream/openamsm/src/smf/nsmf-handler.c` | env-openamsm @ 594dec78 | `a51d023ee57c858f` |

`ca-agent-demo` 文档按 Git blob 记录：

| 文档 | blob |
|---|---|
| `docs/design/amf-smf-business-procedure-flowbook-scp-only.zh-CN.md` | `36aa8811dcb2a4fb` |
| `docs/design/amf-smf-sbi-interface-inventory.zh-CN.md` | `ac9976ed6ab3dfd9` |
| `docs/design/host-sbi-http2-decoded-skill-architecture.zh-CN.md` | `4873ca5823510b65` |
| `docs/design/host-client-tools-amf-smf-boundary.zh-CN.md` | `51ff36f49fd49e8f` |
| `docs/design/scp-dual-smf-endpoint-routing.zh-CN.md` | `e02f32f5521e46d3` |
| `docs/design/mas-e2e-harness.zh-CN.md` | `fa5d93427ff4fcfb` |

## 3. 任务点名但**不存在**的权威文档（BLOCKED 来源）

在 `ca-agent-demo` 的 `docs/mas-v1alpha5-temporal-e2e-review-20261118` 与 `main` 上均查无此三份文件：

| 文档 | 状态 | 替代来源 |
|---|---|---|
| `docs/design/3gpp-openamsm-mas-naming-crosswalk-scp-only.zh-CN.md` | ABSENT | 3GPP 名称取自 `TS29502` / `TS29518` OpenAPI 的 path + operationId，并对齐 `amf-smf-sbi-interface-inventory` 的 N01–N13 / A01–A12 表；MAS 名称取自 `Nsmf_MasOrchestration.yaml` |
| `docs/design/mas-capability-traceability-scp-only.zh-CN.md` | ABSENT | 用 `ca-skills` `contract.yaml` 的 13 个 capability 与 `ca-client-tools` `source-map.json` 的 13 个 stepId/capabilityCandidate/publicCallable 机械对照 |
| `docs/design/mas-capability-traceability-scp-only.json` | ABSENT | 同上 |

结论：命名对照与 13×13 追踪**未使用**上述三份文件，而是用可核验的机器可读来源重建。
任何依赖这三份文件独有内容的结论都未在本图集中出现。

## 4. 明确未采用的历史/候选材料

| 材料 | 原因 |
|---|---|
| `ca-skills/skills/establish-pdu-session-mas/`（仓库根目录） | 旧语法实现，不是 v1alpha5 正式评审示例 |
| `ca-client-tools/contracts/mas-owner-candidate/v0.1.0` | 被 `v0.2.0` 取代 |
| OpenAMSM `master`/更早 release 的 OpenAPI（r16/r17） | 与 R19 实现版本不符 |
| `M0`（201 Location 指向 OpenAMSM） | 历史过渡对照，不是本轮验收 Profile |
| 无 SCP / AMF 直连 Host / Direct Mock Relay | 已废弃，不作为任何正式网络测试方案 |
