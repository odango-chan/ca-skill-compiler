# CA Skill Compiler 首轮实现任务

本文供 DeepSeek Harness 在 [Compiler #48](https://github.com/odango-chan/ca-skill-compiler/pull/48) 评审后执行。架构依据见[工具链架构](ca-toolchain-architecture-vnext.zh-CN.md)，交付顺序见[演进路线](compiler-evolution-roadmap-vnext.zh-CN.md)。本轮目标是用完整的接口快照在离线环境生成可复验的 Development Artifact。在没有 Evidence、Deployment Grant 或 Provider 时，成功结果为 `LINKED / BUILT / NOT_ADMITTED`。

## 可直接交给 DeepSeek Harness 的 Goal

> 在 `odango-chan/ca-skill-compiler` 实现一个可验证的 Development Build 垂直切片。先读 Compiler #48 的架构和本页，确认评审后的接口包、锁和 Build/Verify 契约，以 v2.0.2 发布提交 `e98a7b26acabfa2a5e1587974ab9be5742e70bf7` 建立旧入口黄金基线。复用现有 Go 静态检查与确定性装配能力。完成离线接口闭包链接、Evidence 义务生成、开发产物 Verify，以及 Native CLI / Go-WASM / Chromium Worker 测试。在独立实现分支提交，交付 draft PR、测试证据和 MAS G0/G1 实际状态。保持 CA-SL v1alpha5 及冻结入口的既有行为。合入与发布另走仓库流程。

## 开工前核对

- 记录 #48 的评审提交，并列出拟实现的 API 标识、必填字段、文件布局、义务适用性和 Verify 输入。若 V0 仍在评审，可先做本地独立样例及旧入口基线，公开线格式与入口以上述契约定版为准。
- 第一份样例由测试侧提供精确快照和预生成锁，至少涵盖两个根别名与一个传递包、一个可变更状态的 Callable、一条消费外部 Evidence 的路径，以及有 Wait 或 Trigger 的 Ingress 引用。使用完全声明的合成接口，标记 fixture 的包身份和来源。正式 MAS 接口包可随后复用同一验收入口。
- 新模式只接收源码、精确快照、锁和可选 Assurance 描述。Resolver 的版本选择、下载与来源核验属于调用方。开发构建目标保持 `connectionagent.io/v1alpha5`。

## 实现顺序

| 步骤 | 交付和验收 |
|---|---|
| 1. 旧入口基线 | 从 v2.0.2 运行 `Check`、`Build`、`InspectDependencies`、`VerifyBuildResult` 的代表性正反例，保存诊断、退出码与产物文件摘要。后续在新代码上用相同输入比对。 |
| 2. 离线 Link | 按已锁定的根与传递边复算快照摘要，验证包、导出、Schema 片段、Effect、Capability 成员和 Ingress 闭包。沿用现有快照摘要向量。明确报告包、导出或使用位置。缺接口或闭包不一致时，构建失败且无产物。 |
| 3. 按检查生成义务 | 对每条适用规则及消费位置记录结果。缺失可外补材料为 `UNRESOLVED`。有效结构和绑定的描述为 `DESCRIPTOR_PRESENT`。语言静态约束失败、提供的描述损坏或目标不符则阻断构建。先提交检查清单，再覆盖普通 Evidence 消费、Commit Proof、Reconcile 与特定路径充分性。 |
| 4. Build 与 Verify | 对规范化锁、链接记录、义务和产物文件生成确定性摘要。装配后自验。Verify 复验产物字节和精确快照闭包，并核对义务覆盖。单纯复算 manifest 摘要仅证明字节一致性。提交至少一个在重新计算外层摘要后仍会被拒绝的遗漏义务负例。若产物缺少重验义务覆盖所需的消费信息，提交契约补充供评审。 |
| 5. 入口及回归 | 在 Native 与 WASM 入口显式选择新契约。新旧入口交叉传错版本均失败。以真实 Chromium Worker 比对成功样例的完整产物字节和失败样例的结构化诊断。 |

首轮需区分三类结果：

| 输入变化 | 预期 |
|---|---|
| 接口完整，Evidence、Grant 和 Provider 均缺失 | 构建、完整离线 Verify 成功。输出带适用的未决义务，执行状态固定为 `NOT_ADMITTED`。 |
| 删除传递包、类型或片段、Capability 成员、必须的 Ingress 导出或 Effect | 构建失败，无产物。 |
| 提供错误绑定的描述、替换锁定快照或删除必需义务 | 按阶段在 Build 或 Verify 失败。通过完整性检查的产物仍需要 Host 验证准入材料。 |

增加一组同版异摘要、重复符号和未锁外部引用的负例。验证重排输入后的产物字节一致。以三者均缺失的请求为基线，增加有效描述时核对对应义务的状态变化。构建入口按定版契约拒绝未知字段。Grant 和 Provider 由 Host 接收。

## 重跑命令和验收记录

先执行仓库现有基线门禁。新增 Development Build 用例必须通过新入口运行，并将 CLI 与 Chromium 用例的实际命令写进 PR。

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
git diff --check
```

记录源提交和工具版本、每项退出码、fixture 锁与产物摘要、重复构建及跨入口比对结果。若 GitHub Actions 因账单无法执行且本地门禁全部通过，记录 `LOCAL_ACCEPTANCE_PASS` 与 `CI_UNAVAILABLE`。

## MAS 验收边界

当前已知基线是 v2.0.2 对 MAS 候选 fixture 的静态 Check / Inspect / Build 通过，见 [Compiler #47](https://github.com/odango-chan/ca-skill-compiler/pull/47) 和 [ca-skills #76](https://github.com/odango-chan/ca-skills/pull/76)。[ca-client-tools #17](https://github.com/odango-chan/ca-client-tools/pull/17) 是 WTLab 候选包发布方案。它提供可核对的 Operation 与 Schema 输入，G0 仍需验证可链接的精确接口快照、Effect、`ca-context` 与实际引用的 Ingress 闭包。

| 结果 | 所需证据 |
|---|---|
| 首轮 Compiler 验收通过 | 合成闭包生成可 Verify 的 Development Artifact。旧入口、Native/WASM/Chromium 和负例通过。 |
| 正式 MAS G1 通过 | 固定 `ca-skills/examples/skills/establish-pdu-session-mas/` 源提交，用通过 G0 的精确快照与锁对不修改的 Skill Build + Verify，并记录全部 13 个 Callable 的绑定及实际所需 Schema 闭包、产物摘要与未决义务。G0 未就绪时保留候选结果并报告 G1 等待的具体包或导出。 |
| 真实 MAS E2E | [ca-agent-demo #217](https://github.com/odango-chan/ca-agent-demo/pull/217) 的 Host Admission、Runtime/Temporal 和 Provider 测试继续按 G2–G5 验收。先经本地 NRF/SCP 使用 Mock AMF 走真实 SMF SBI handler，再用真实 AMF / UERANSIM 验证完整会话和用户面。复合 SSM 当前仍需要可持久恢复的 Owner continuation。 |

首轮实现的提交范围限 Compiler。Host 的运行准入和真实网络证据继续在对应仓库交付。
