# DeepSeek Harness Goal：CA Skill Compiler 下一目标实施

**状态：** 可交给 DeepSeek Harness 执行的开发 Goal（基于候选契约，不代表语言或产品已发布）。  
**仓库：** `odango-chan/ca-skill-compiler`  
**目标语言：** `connectionagent.io/v1alpha5`（ca-skills #61 候选）  
**开发方式：** 代码实施优先；契约有缺口时才回写文档。

## 1. Goal

在现有 Go Compiler 仓库中完成下一目标的静态编译能力：Node-entry Event Wait、package-scoped Integration Ingress Operation、触发 Operation 绑定、Wait Output SourcePath、Procedure Graph 控制流、Response Boundary，以及一致的 Build / Inspect API 和 Native / WASM / Chromium Worker 实现。

最终交付必须是可执行、可测试、可重复构建的 Compiler，而不是只有 AST 原型或设计文档。

## 2. 规范与契约来源

按以下顺序核对，不得自行补造语义：

1. **ca-skills PR #61**：完整单文件候选 `docs/skill-language-spec.v1alpha5.zh-CN.md`，语言目标为 `connectionagent.io/v1alpha5`。当前 Draft；语义基线标记 FROZEN，但规范发布和产品符合性仍未完成。
2. **ca-skills PR #62**：package-scoped IngressCatalog、`ingressOperation`、Schema Closure Digest、Trigger / Correlation / Response Mapping 契约。当前 Draft；不得把合成 Fixture 当成真实 Owner 发布物。
3. **ca-skills PR #60 / #64**：已合并的 Wait 语义和 EVENT_ACCEPTED tombstone / stale-delivery 隔离原则。
4. **本仓库现有代码、测试、PR #29 / #30**：作为实现现状，不高于规范契约。

如规范之间存在冲突：记录精确条款、字段、诊断和影响；将受影响能力隔离在内部接口后继续其他独立工作。不得静默猜测、放宽校验或发明诊断 Code。

## 3. 明确的兼容性决策

**不要求向后兼容。正确性和架构简洁优先。**

- 不要求保留 v1alpha4 Compiler 行为、内部 API、Go 类型、AST、诊断、BuildRequest、DependencyInspection 或 Artifact Manifest。
- 不要求继续支持旧 Skill 项目，不要求自动迁移，不要求双 Parser / 双 Linker / 兼容 Adapter。
- 可以重构、替换或删除旧 Compiler 代码、Schema、CLI 和 WASM Adapter。
- 只有在复用旧组件确实更简单且语义正确时才复用；不能为了兼容而保留技术债。
- 已发布历史 Tag / Release 作为历史记录保留，不重写其发布物；这不限制当前开发分支进行破坏性重构。
- 新 API 必须严格、闭合、版本化；不应为了兼容旧 API 而扩大旧 Schema。

## 4. 架构边界

Compiler 负责静态解析、链接、Schema / 类型分析、Procedure Graph 和 Artifact 构建；不实现 Host / Runtime 的动态职责。

必须保持以下语义：

- Ingress Operation 是 package-scoped inbound protocol export，不是 ToolCatalog callable、Observation 或 Capability。
- Outbound trigger 必须绑定精确的 ToolCatalog Operation；不得用 Capability ID、网络地址或自由文本代替。
- Wait 是 Node-entry control，不是第五种业务阶段，也不是独立 Wait Node kind；业务阶段仍是 precondition / operation / observation / postcondition。
- `wait.output` 是独立 Producer SourcePath，不注入全局 Context，不伪装成 Operation Result、Observation Result 或 Evidence。
- EVENT_ACCEPTED / TIMEOUT 是控制流边，不是 Stage Outcome；Wait 接收事件本身不产生 Goal proof。
- Compiler 不实现事件接收、计时器、持久化、原子竞争、重试执行、tombstone 生命周期、Temporal Workflow 或 SBI HTTP/2 / multipart transport。

## 5. Work Packages 与执行顺序

WP1 与 WP2 可以并行；其余按依赖推进。

| WP | 工作 | 完成定义 |
|---|---|---|
| WP0 | 代码与契约盘点、API 决策 | 核对 main、PR #30、#61、#62；形成文件级实现清单、API 字段决策和未决问题列表 |
| WP1 | IngressCatalog / Schema Closure | 从完整原始快照字节验证 artifactDigest；Dialect-aware Schema 根发现；canonical URI / nested $id / fragment 映射；精确传递闭包；JCS Digest；负向向量 |
| WP2 | Wait Parser / AST | 完整解析 v1alpha5 Node-entry Wait；Duration、Selector、Response、TIMEOUT 路由、未知字段、类型和源码位置诊断符合候选规范 |
| WP3 | Trigger / Correlation Binding | 每条 Wait 入边绑定精确 ToolCatalog Operation；推导 typed armInputSchema；验证 Invocation Context 和所有入边一致性 |
| WP4 | wait.output SourcePath | 绑定 normalized request Schema；JSON Pointer、类型赋值、路径可用性及分支敏感性 |
| WP5 | Procedure Graph | Wait entry control vertex、EVENT_RECEIVED / TIMEOUT 控制边、DAG、reachability、dominance、route obligation；不将事件接收当作 Goal proof |
| WP6 | Response Boundary | `wait.response.after` 精确 Producer stage、Producer output Schema、responseResultSchema、response dispositions、JSON / multipart / binary 映射 |
| WP7 | Diagnostics / Conformance | 精确 Code / Phase / Severity / messageKey / sourceLocation；EW / IN 静态用例映射到可执行测试；动态用例明确归属 Host / Runtime / Integration |
| WP8 | Core / Native / WASM / Chromium parity | CLI 与 WASM 共用同一 Go Core；同一输入输出一致的 Diagnostics、DependencyInspection 和 Artifact bytes；真实 Chromium Worker 验证 |
| WP9 | Studio / LSP integration | 消费同一版本化 API；展示 Ingress / Wait 诊断、Schema、Completion 和导航，不另造 Parser / Linker |

不要把 WP0 的文档盘点做成长期阻塞。契约明确的 WP1 / WP2 立即实施；遇到局部缺口时记录并隔离，不阻塞无关工作。

## 6. 首轮必须处理的 PR #30

PR #30 是 WP2 起点，不是完成品。Harness 首先要：

1. 对照 #61 单文件规范逐字段核查当前 `v1alpha5_wait.go`。
2. 确认 Procedure Node 允许字段集合、Wait 字段集合、YAML 类型、必填/可选关系、重复键处理、源码位置和诊断 Code。
3. 核查 FixedDuration 的完整 lexical subset，禁止月份、年份、周、零值、前导零、小数、负数、乱序及空的时间部分；不要把 Host 的最大时长限制错误放进 Compiler。
4. 核查 Selector 必须是非空 YAML string；核查 `response.after` 的 NodeId 和可达性约束。
5. 核查 TIMEOUT 恰好一个；禁止作者声明 EVENT_RECEIVED 路由和 TIMEOUT → SUCCEEDED；校验目标 Node / terminal 的精确集合。
6. Wait-bearing Node 必须含至少一个业务阶段；Wait 不是独立业务 Node。
7. 将解析结果接入 v1alpha5 Procedure AST，而不止停留在孤立 helper；后续接入 Graph / Build pipeline。
8. 对每个规范规则增加正例、负例和精确诊断断言。

如 PR #30 的实现结构不利于完整 Parser，可直接重构，不必为了保留现有 AST 形状兼容。

## 7. API 与模块设计要求

新 API 需要统一考虑以下表面，版本号分别决策，不要假设与语言版本同号：

- BuildRequest
- BuildResult / Diagnostic Envelope
- DependencyInspection（Ingress 必须有独立类型，不混入 callables）
- Artifact Manifest / dependency closure
- CLI JSON 输入输出
- WASM / Studio Adapter

先校验 API 版本，再校验目标语言版本；如果当前产品只支持一个新 API，使用一个严格入口即可，不要实现无意义的多版本兼容 Router。CLI 与 WASM 必须调用同一 Core，不允许出现两套 Parser 或 Linker。

## 8. 测试和验收

每个 WP 都必须提供规范条款到测试用例的追踪表。最少包括：

- 聚焦单元测试和负向 Fixture；
- `go test ./...`；
- `go vet ./...`；
- `CGO_ENABLED=0 go build ./...`；
- `CGO_ENABLED=0 GOOS=js GOARCH=wasm go build ./cmd/skillc-wasm`；
- 真实 Chromium Worker parity；
- 同一输入在 Native / WASM / Worker 上的 Diagnostic、Inspection 和 Artifact byte parity。

GitHub Actions 如果没有实际分配 Runner 或 Job 在任何步骤执行前失败，必须如实标记 CI unavailable / no test steps executed。此时执行完整本地验收，并分别报告 `LOCAL_ACCEPTANCE_PASS` 与 `CI_UNAVAILABLE`，不得把本地结果写成 CI PASS。

合成 Snapshot 只用于静态 Compiler 测试；不得将其报告为真实 Owner / Deployment / 5GC 互操作资格。

## 9. 完成标准

- WP1–WP8 的 Compiler-owned 静态能力完成；未完成项有明确阻塞和诊断归属。
- 所有实现规则均可追溯到 #61 / #62 的规范条款或经确认的决策。
- Build / Check / InspectDependencies 使用同一目标与同一 Core。
- Native / WASM / Chromium Worker 一致性通过。
- 新 API、Schema、诊断索引和 Artifact Manifest 自洽、严格、可复现。
- 不要求 v1alpha4 兼容；不为兼容旧实现保留重复架构。
- PR 只有在真实测试已执行、验收记录完整后才可标记 Ready / 合并；不得因 CI 初始化失败而误判源码失败。

## 10. Harness 每轮报告格式

每轮报告必须包含：

1. 当前分支、HEAD SHA、关联 PR。
2. 本轮完成的 WP / 具体文件。
3. 对应规范条款与诊断 Code。
4. 实际执行的测试命令及 PASS / FAIL / NOT_RUN。
5. CI 是否真正执行步骤；若未执行，说明原因和本地验收情况。
6. 尚未完成的阻塞、契约问题及下一步。
7. 是否存在对旧实现的删除或破坏性重构，以及理由。

不要只报告“代码已提交”或“CI 失败”；必须区分源码失败、测试失败、CI 初始化失败和未执行。
