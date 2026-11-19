# CA Skill Compiler 下一阶段演进路线

> **状态：PROPOSAL / NON-NORMATIVE**
>
> **基线：** CA-SL v1alpha5 FROZEN；Compiler v2.0.2 RELEASED。
>
> 本路线不启动语言 vNext，也不改变当前冻结契约。目标是先完善接口包与链接模型，再让 Compiler 生成可以离线交付、但尚未获得执行资格的开发 Artifact。

## 1. 下一版本的目标

建议下一轮以 Compiler 产品线的 v2.1.x 作为候选演进窗口（版本号待评审，不是已批准的 Release 承诺），继续使用 CA-SL v1alpha5。

核心目标：

> 一个 Skill 只要依赖的接口定义完整，就可以完成 Compile、Interface Link 和 Artifact Build；缺少 Evidence、Deployment 或实际 Provider 时，产物仍然可以生成，但必须明确不可据此执行。

这轮不做：

- 新的 CA-SL 语言语义；
- 第二套 Procedure / Runtime 引擎；
- Compiler 内置 Package Registry / Package Manager；
- Compiler 自动下载依赖；
- Compiler 执行 Provider 或网络调用；
- Compiler 签发 Owner / Deployment / Qualification 证明。

## 2. 分阶段路线

| 阶段 | 交付 | 主要责任 | 退出标准 |
|---|---|---|---|
| V0 — Architecture Freeze | Interface Package、Capability Catalog、Assurance Export、Runtime Provider、Deployment Grant 的边界文档 | Compiler + Skills + Host + Provider Owner | 概念、身份、版本和责任边界无歧义 |
| V1 — Interface Export | 可发布的 Package Manifest、Callable Catalog、Schema Closure、Capability Catalog | Provider / Owner CI；不是 Compiler | MAS 13 个 Callable 和 25 个 Schema 可以形成稳定接口快照 |
| V2 — Resolver Input | 外部 Resolver 生成锁定依赖输入，Compiler 只消费快照 | Studio / CI / 调用方 | 同一锁文件能取得相同包版本和字节；Compiler 无网络依赖 |
| V3 — Interface Link Build | 新版本化 Build API 支持仅基于接口包生成 Development Artifact | Compiler | 没有 Evidence / Deployment / Runtime Provider 时，合法 Skill 仍能 Link + Build |
| V4 — Host Admission | Host 单独绑定 Runtime Provider、Capability Grant、Assurance 与 Invocation Context | ca-agent-demo | 未授权 / 未绑定 / 保证不足的 Artifact 在真实 Dispatch 前被拒绝 |
| V5 — MAS Integration | MAS Development Artifact 进入现有 Host / Runtime / Provider 执行链 | ca-agent-demo + ca-client-tools + env-openamsm | 本地真实 Provider / Mock AMF 测试通过，再进入真实 AMF / UERANSIM |

## 3. Compiler vNext 的接口演进

下一版 Build API 需要把当前 v1alpha5 的输入职责拆清楚：

- **必需输入：** Skill 源码、确定的 Interface Package 快照、Compiler Target。
- **可选输入：** Assurance Export。提供时必须严格校验；缺失时只记录未满足的保证要求。
- **不属于 Compiler Build 输入：** 目标环境的 Deployment Grant、Provider endpoint、凭据和运行时状态。
- **外部 Resolver 输入：** 精确 Package ID / version / digest 与对应快照；Compiler 校验输入内容，不负责版本选择和下载。

如果要改变当前 BuildRequest / BuildResult / Artifact Manifest 的结构或成功条件，必须新增版本化 API / Artifact Contract。不能在 v1alpha5 的冻结契约中通过一个隐式开关改变含义。

## 4. 必须通过的设计与测试

### 4.1 正向测试

1. 有完整 Callable、Schema、Effect 和 Capability Catalog，无 Evidence / Deployment / Provider，Interface Link PASS。
2. 同一输入重复 Build，Artifact 字节完全一致。
3. 同一锁定快照在 Native / WASM / Chromium Worker 的结果一致。
4. Artifact 能被独立 Verify，并能准确列出依赖闭包和 Callable Binding。
5. 加入可选 Assurance Export 后，Compiler 能校验其结构与精确绑定。

### 4.2 负向测试

1. 缺少 Callable 或 Callable 身份不唯一：Link FAIL。
2. 缺少 Input / Output Schema 或 $ref closure 不完整：Link FAIL。
3. Schema 类型不匹配、Capability Catalog 缺少成员：Link FAIL。
4. Package ID、版本或内容摘要与实际输入不匹配：Link FAIL。
5. Assurance Export 已提供但结构损坏、Producer 错误或 Schema 绑定错误：FAIL。
6. 没有 Evidence：允许 Development Artifact，但明确 PENDING，不得声称保证已满足。
7. 没有 Deployment Grant：允许 Artifact Build；真实 Host Admission 必须拒绝未授权执行。
8. 没有 Runtime Provider：允许 Artifact Build；Host 不得派发到不存在的实现。
9. Development Artifact 被当作生产 Artifact 使用：Host Admission 必须拒绝。

### 4.3 MAS 专项验收

- 使用 ca-client-tools 当前 13 Operation source map 和 25 个 Schema 候选构造 Interface Package。
- 为 ca-context 提供独立 Subject Context Interface Package。
- 在没有 Evidence / Deployment / Provider 运行实例的条件下生成 Development Artifact。
- Artifact 显示准确的未满足 Assurance 项，不把候选接口冒充可信 Owner 发布。
- 再接入真实 Provider、Host Admission 和 env-openamsm，逐步进入 Mock AMF 与真实 AMF E2E。

## 5. 跨仓库职责

| 仓库 | 下一阶段职责 | 不应该承担 |
|---|---|---|
| ca-skills | 保持 v1alpha5 源码和接口引用清晰；维护 MAS 示例 | 不承担 Provider Release 或 Host 授权 |
| ca-skill-compiler | Interface Link、确定性 Artifact、契约验证 | 不做 Registry、下载、部署、Qualification、Runtime |
| ca-client-tools | 发布准确的 MAS Interface Export；实现独立 Runtime Provider | 不把私有 backend operation ID 当作 Skill 公共 Callable ID |
| ca-agent-demo | Artifact Admission、Provider Binding、Temporal 与唯一 Procedure Runtime | 不重写 Compiler 或另造 MAS 图引擎 |
| env-openamsm | 真实 5GC 环境、Mock AMF 测试对端、真实 AMF / UERANSIM 验收 | 不替 Host 产生虚假的 Owner Proof |

## 6. 版本与发布纪律

- v2.0.2 保持不变，是当前发布基线。
- 下一版产品版本号和新 API / Artifact 版本号独立决定。
- CA-SL v1alpha5 保持冻结；本路线不引入语言兼容层。
- 设计评审通过前不改动 v2.0.2 的 Build 成功条件。
- 新开发模式必须显式区分 INTERFACE_LINKED、ARTIFACT_BUILT 与运行时 ADMITTED；不以一个 PASS 表示全部阶段。
- Compiler 生成的 Artifact 是不可变编译产物；Owner / Deployment / Qualification 结果由外部权威独立发布和验证。

## 7. 近期建议

第一优先不是改 Compiler 算法，而是先从 MAS 现有资料中整理一个最小 Interface Package：

1. 稳定 mas Package ID 和版本；
2. 13 个 Callable 导出及准确 Effect 声明；
3. 13 个 Input / 12 个 Output Schema；
4. 独立 Capability Catalog；
5. ca-context Subject Context Schema Package；
6. 可重复生成的快照与锁定输入。

完成后，才能用一个真实 MAS Skill 验证新 Link 模型，而不是继续用合成 fixture 证明 Compiler 自己能编译 fixture。
