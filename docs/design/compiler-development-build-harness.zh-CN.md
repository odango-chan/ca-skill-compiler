# CA Skill Compiler 首轮实现任务

**状态：可以交接。** 本轮按 [Compiler #48](https://github.com/odango-chan/ca-skill-compiler/pull/48) 中的[实施契约](compiler-development-build-pilot.zh-CN.md)提交试验实现。架构依据见[工具链架构](ca-toolchain-architecture-vnext.zh-CN.md)，交付顺序见[演进路线](compiler-evolution-roadmap-vnext.zh-CN.md)，Evidence / Assurance 边界见[职责审计](compiler-evidence-assurance-responsibility-audit.zh-CN.md)。本轮目标是用完整的接口快照在离线环境生成可复验的 Development Artifact。在没有运行时 Evidence、Deployment Grant 或 Provider 时，成功结果为 `LINKED / BUILT / NOT_ADMITTED`。

## 可直接交给 DeepSeek Harness 的 Goal

> 在 `odango-chan/ca-skill-compiler` 实现一个可验证的 Development Build 垂直切片。从当前 main 新建实现分支。用本页链接的实施契约和请求样例开工，以 v2.0.2 发布提交 `e98a7b26acabfa2a5e1587974ab9be5742e70bf7` 建立旧入口黄金基线。复用现有 Go 静态检查与确定性装配能力。完成离线接口闭包链接、Evidence 静态语义、Host Admission Requirement 记录、开发产物 Verify，以及 Native CLI / Go-WASM / Chromium Worker 测试。在独立实现分支提交，交付 draft PR、测试证据和 MAS G0/G1 实际状态。保持 CA-SL v1alpha5 及冻结入口的既有行为。在 Draft PR 记录所使用的设计提交 SHA。公开契约冻结与合入另行评审。

## 开工前核对

- 从[实施契约](compiler-development-build-pilot.zh-CN.md)复制精确入口和请求形状。源分支取当前 main，设计提交与实现提交分别记录。
- 用[首个正例请求](fixtures/registration-development-request.json)做 Native/WASM 共用的验收输入。它覆盖两个根别名、传递 Schema 和一个由锁固定的 `EvidenceDescriptor` 导出。Mutation/Reconcile 补用 `internal/compiler/evidence_recovery_test.go` 的 `recoveryFixture`，Wait/Ingress 补用 `web/parity-v1alpha5/trusted-build-request.json` 和同目录 `valid-procedure.yaml`。将上述严格输入显式打包为测试接口快照，保留原始 Skill 语义。
- 新模式只接收源码、精确快照和锁。Evidence Descriptor 属于 Interface Package 导出，不是独立 Assurance 输入；Resolver 的版本选择、下载与来源核验属于调用方。开发构建目标保持 `connectionagent.io/v1alpha5`。
- Owner 身份、签发权限、IndependentDomains、CommonFact、真实 Commit Proof 和运行时 Evidence 都不得进入 BuildRequest。

## 实现顺序

| 步骤 | 交付和验收 |
|---|---|
| 1. 旧入口基线 | 从 v2.0.2 运行 `Check`、`Build`、`InspectDependencies`、`VerifyBuildResult` 的代表性正反例，保存诊断、退出码与产物文件摘要。后续在新代码上用相同输入比对。 |
| 2. 离线 Link | 按已锁定的根与传递边复算快照摘要，验证包、导出、Schema 片段、Effect、Capability 成员、EvidenceDescriptor 和 Ingress 闭包。沿用现有快照摘要向量。明确报告包、导出或使用位置。缺接口或闭包不一致时，构建失败且无产物。 |
| 3. 静态 Evidence 语义 + Host Requirement | 完整执行冻结的 Evidence 语言语义：声明/引用、Descriptor 形状、Producer/Consumer、SourcePath、Schema 类型、Independence 静态部分、Mutation/Commit Proof/Reconcile 静态约束。Descriptor 缺失或非法即失败。静态检查通过后，仅把外部权威事实记录为 `OWNER_TRUST`、`EVIDENCE_DOMAIN_TRUST`、`EVIDENCE_FACT_TRUST`、`EVIDENCE_INDEPENDENCE_TRUST`、`COMMIT_PROOF`、`RECONCILE` 的 Host Admission Requirement。 |
| 4. Build 与 Verify | 对规范化锁、链接记录、Host Requirement 和产物文件生成确定性摘要。装配后自验。Verify 复验产物字节和精确快照闭包，并核对 requirement 覆盖。单纯复算 manifest 摘要仅证明字节一致性。提交至少一个在重新计算外层摘要后仍会被拒绝的删改 requirement 负例。复验用原始请求重跑核心并校对完整产物字节。 |
| 5. 入口及回归 | 在 Native 与 WASM 入口显式选择新契约。新旧入口交叉传错版本均失败。以真实 Chromium Worker 比对成功样例的完整产物字节和失败样例的结构化诊断。 |

首轮需区分三类结果：

| 输入变化 | 预期 |
|---|---|
| 接口与静态 Evidence Descriptor 完整，运行时 Evidence、Grant 和 Provider 均缺失 | 构建、完整离线 Verify 成功。输出带适用的 Host Admission Requirement，执行状态固定为 `NOT_ADMITTED`。 |
| 删除传递包、类型或片段、Capability 成员、必须的 Ingress 导出、EvidenceDescriptor 或 Effect | 构建失败，无产物。 |
| 提供错误绑定的 Descriptor、替换锁定快照、删除必需 requirement、或声明需要外部域/事实/独立性授权 | 静态失败在 Build 阶段拒绝；纯外部信任缺失不阻止 Build，但保留 Host Admission Requirement。通过完整性检查的产物仍需要 Host 验证准入材料。 |

增加一组同版异摘要、重复符号和未锁外部引用的负例。验证重排输入后的产物字节一致。以静态契约完整但外部信任缺失的请求为基线，增加信任材料后核对 requirement 状态变化。构建入口按定版契约拒绝未知字段。Grant 和 Provider 由 Host 接收。

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

## 交接前已验证的基线

使用经发布 SHA256SUMS 核对的 v2.0.2 Linux amd64 二进制。取 `tests/conformance/v1alpha4/fixtures/dependency-identity/base` 的 `files/` 和 `snapshots/`，移除 fixture 的 `snapshots/unreferenced-extra/`，从 `web/parity-success` 复制 `trusted.json` 和 `capability/`。在项目目录外写产物。对同一输入运行发布版 `check`、`inspect` 和两次 `build`，均退出 0。两次三文件产物逐字节一致，manifest SHA-256 为 `96d79739d94b29800fce7aea7ee2aab6fa35480b47776fe7bf5948b92bdbe60b`。删除 `ca/evidence-guarantees.yaml` 后严格 Check/Build 退出 1 并报 `EVIDENCE_GUARANTEE_LINK_INVALID`。把新的 Development 请求送入旧 CLI 退出 2。两个负例均未创建产物目录。

独立复算新请求的三份快照摘要和 manifest 文件摘要，核对两个根、一个传递依赖边和原始源码字节。摘要程序已对照仓库的三个冻结向量。首轮 Development Build、Verify 和跨执行环境测试由 Harness 实现并实跑。

## MAS 验收边界

当前已知基线是 v2.0.2 对 MAS 候选 fixture 的静态 Check / Inspect / Build 通过，见 [Compiler #47](https://github.com/odango-chan/ca-skill-compiler/pull/47) 和 [ca-skills #76](https://github.com/odango-chan/ca-skills/pull/76)。[ca-client-tools #17](https://github.com/odango-chan/ca-client-tools/pull/17) 是 WTLab 候选包发布方案。它提供可核对的 Operation 与 Schema 输入，G0 仍需验证可链接的精确接口快照、Effect、`ca-context` 与实际引用的 Ingress 闭包。

| 结果 | 所需证据 |
|---|---|
| 首轮 Compiler 验收通过 | 合成闭包生成可 Verify 的 Development Artifact。旧入口、Native/WASM/Chromium 和负例通过。 |
| 正式 MAS G1 通过 | 固定 `ca-skills/examples/skills/establish-pdu-session-mas/` 源提交，用通过 G0 的精确快照与锁对不修改的 Skill Build + Verify，并记录全部 13 个 Callable 的绑定及实际所需 Schema 闭包、EvidenceDescriptor 闭包、产物摘要与 Host Admission Requirement。G0 未就绪时保留候选结果并报告 G1 等待的具体包或导出。 |
| 真实 MAS E2E | [ca-agent-demo #217](https://github.com/odango-chan/ca-agent-demo/pull/217) 的 Host Admission、Runtime/Temporal 和 Provider 测试继续按 G2–G5 验收。先经本地 NRF/SCP 使用 Mock AMF 走真实 SMF SBI handler，再用真实 AMF / UERANSIM 验证完整会话和用户面。复合 SSM 当前仍需要可持久恢复的 Owner continuation。 |

首轮实现的提交范围限 Compiler。Host 的运行准入和真实网络证据继续在对应仓库交付。
