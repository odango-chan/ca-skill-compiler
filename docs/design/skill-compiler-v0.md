# Skill Compiler v0 · 多 Skill 工程、Library 链接与浏览器编译

- **设计状态**：ACCEPTED（2026-11-11 用户确认产品范围与目录）；实现状态：PARTIAL（首批 Core/CLI/WASM 源码）；验收：NOT_RUN
- **目标产品 / 仓库名**：`ca-skill-compiler`；CLI：`skillc`
- **当前代码仓**：`odango-chan/ca-skill-compiler`（全新仓库，与旧 `ca-skill-toolchain` 保持独立）
- **跟踪**：[新仓 Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1)；历史决策 [旧仓 #133](https://github.com/odango-chan/ca-skill-toolchain/issues/133)
- **设计所有者**：Skill Compiler；**消费者**：Skill Studio、Agent Host / Runtime 的包加载端
- 本文件是新仓的设计基线，首批代码实现范围需以测试和仓库实际文件为准。旧 `ca-skill-toolchain` 的 Artifact/Variant/Qualification/Release 仅供历史参考。

## 1. 产品目标与非目标

Skill Compiler 是类似 GCC 的**无状态编译与链接工具**。输入多 Skill 源码与显式只读依赖，返回诊断和可拷贝文件树；可选由宿主压缩为 ZIP。必须支持同一 Core 的原生 CLI 和 GitHub Pages 内的 Go/WASM 使用，不能依赖常驻后端。

**v0 必须提供**：
1. 将 `skills/<name>/` 下的多个独立 Skill 编译成可交付的 Skill 目录；支持全项目、单 Skill 和必要的 Skill-to-Skill 依赖闭包；
2. 通过 `lib/` 导入外部工具接口（先支持 `ca-agent-demo/tools/contracts/createdsession.tool.yaml`），解析 Operation、input/output/reconcile Schema、声明的 Effect，不执行工具；
3. 解析 `SKILL.md`、`assurance/`、`scripts/`、`references/` 与作者拥有的 `schemas/`；校验最小必要语法、入口、引用与依赖可解；
4. **仅支持新 Script 契约**：输入类型从 Procedure 绑定的外部 Producer Schema 链接，Script 自己的严格 Output TypedDict 自动生成 Output Schema；旧 Script metadata / external I/O Schema 不做兼容；
5. 生成 `output/` 文件视图，以及机器可用的文件列表与诊断结果；Native CLI 写盘，Studio 做 ZIP / 用户授权目录保存。

**v0 明确不做**：SQLite/数据库、HTTP 后端、Workspace/Revision、Catalog、Profile、Qualification（SQ1/SQ2）、Planning/Mock/Scenario Runner、Release/Version/Gate、ProcedureVariant/Patch 发行体系、强制生成 Digest/Provenance、Runtime Admission、ProviderBinding、MCP 服务器执行、Python 运行/沙箱、Agent LLM 循环、真实 5GC 测试。编译时仍可使用常规内部缓存/数据结构，但不得形成产品级状态管理。

**最小安全与正确性不省略**：禁止 `..` / 绝对路径、重复/冲突文件、危险链接/ZIP 路径、歧义 Tool 名称、悬空引用、编译器执行不可信脚本；资源大小和递归深度应有限制。这些是构建安全，而非 Qualification 平台。

## 2. 项目目录：skills + lib + output

~~~text
project/
  skills/                         # 可编辑、多 Skill、唯一业务源码
    establish-pdu-session/
      SKILL.md
      assurance/
      scripts/
      schemas/                    # 仅作者自己维护的业务接口 Schema
      references/
    establish-pdu-session-mas/
      ...
    5gc-intent/
      ...
  lib/                            # 外部导入，Studio 中只读
    tools/
      createdsession.tool.yaml    # 从 ca-agent-demo 导入的契约原文
    host/                         # 真实 Host Owner 导出的输入契约（未提供时不能伪造）
    observations/                 # 真实 Observation Owner 的契约（如有）
    manifest.yaml                 # 可选导入来源/状态说明，非部署 READY
  output/                          # 构建投影/产物，非作者 Source
    skills/
      <built-skill>/
        ...
    lib/
      tools/
        <used-imported-catalog>.yaml
    skills.zip                     # Studio 生成的可选下载件
~~~

- `skills/` 是多 Skill 作者源码根目录，不额外引入第四层名为 `source/` 的容器。
- `lib/` 是项目依赖快照，作者可以导入、删除、重新导入以更新；不能在普通源编辑器中直接修改外部接口。它不应被复制进每个 Skill。
- `output/` 在 Pages / Studio 可以是虚拟、内存中的构建视图；默认不得重新参与编译或保存回 `skills/`；在本地 CLI 是实际目录。避免 build -> import -> build 递归污染。
- 生成文件保留“generated”来源信息，导入契约保留“imported / read-only”信息；不要伪装成作者文件。
- 构建一个根 Skill 时，只带上其需要的 Skill-to-Skill 依赖；库快照 v0 可以原样拷贝**被使用的** ToolCatalog 文件，不必对一个 YAML 内部每个 Operation 做拆分和新身份定义。
- ZIP 是交付容器，不等于 Runtime 已部署；既有 Host 仍需单独具备 Tool 实现/Registration、Admission 等能力。

## 3. Source、Generated 与 Imported 的边界

| 材料 | 权威来源 | Studio 可编辑 | 编译器动作 |
| --- | --- | --- | --- |
| `SKILL.md`、`assurance/contract.yaml`、`assurance/procedure.yaml`、`assurance/predicates.yaml` | Skill 作者 | 是 | 读取、最小必要解析/链接 |
| Skill 公共请求/业务接口 Schema | Skill 作者 | 是 | 校验并原样输出 |
| Script Python 源文件 | Skill 作者 | 是 | 静态读取，不执行 |
| Procedure 调用 Script 的参数 | 上游 Producer + Procedure `with:` | 修改绑定，不重新定义 | 调用点 InputProjection（不作为单独权威 Schema） |
| Script 新计算得到的 Output Schema | Script Output TypedDict | 在 Python 中编辑类型 | 在 Output 下生成唯一 Output Schema |
| Tool Operation Input/Output/Reconcile Schema / Effect | 外部 Tool Contract Owner | 否 | 导入链接、只读浏览、按需打包快照 |
| Studio 用来支持 YAML 诊断的 Meta-Schema | 语言规范/Compiler | 否 | 内置编译器/Studio，不进入 Skill 包 |
| Qualification/Profile/Release/Runtime 状态 | 外部相应 Owner | 否 | 不纳入 v0 |

### 3.1 新 Script 和链接语义（BREAKING ONLY）

**唯一规范：**[Script Contract v0](script-contract-v0.md) + [Procedure Link v0](procedure-link-v0.md)，两者均为已确认设计、未完成实现。

- Tool / MCP Input/Output 来自只读 `lib/tools/`；Host/Observation 必须由真实 Owner 提供导入契约；由 Skill 自己定义的公共请求仍属于 Skill Source。
- Procedure `with:` 的 `from:` 路径决定调用点 Script Input Projection；不要求在 Python 再写重复的 Input TypedDict，也不把临时投影输出为第二份权威 Schema。
- 新 Script 使用 `main(input: dict) -> OutputTypedDict`；仅 Script **新计算的输出**从严格 TypedDict / Annotated 类型生成只读 Output Schema；Python 函数体不执行。
- **没有旧格式兼容模式**：旧 Script metadata + 旧 Schema 只用于固定源快照的迁移分析；不会通过 fallback 得到 Build PASS。若需升级，创建单独的迁移工程，改变脚本和 DSL 后按新规范验收。
- `examples/ca-agent-demo-project` 是迁移前 reference，**不要求原样 PASS**；真实 Host/Observation Schema 暂未确认齐全，不能发明一个替代品。
- 任何未知字段、缺失数据 Owner、无法静态解析的输出类型都必须诊断；不能因为方便 Demo 就忽略错误。

**外部 Host Schema 绑定位置**：新 contract 明确使用 `spec.inputs.requestSchema` 和 `spec.context.schemas.subject/facts/observations`，具体路径/缺失规则以 [Procedure Link v0 §2.1](procedure-link-v0.md) 为唯一权威；不允许靠 `requiredFacts` 名称推断数据字段。

## 4. Library v0：先文件导入，后 MCP

### 4.1 第一版导入文件

主样本：`odango-chan/ca-agent-demo/tools/contracts/createdsession.tool.yaml`
- 当前是 `ToolCatalog`（`apiVersion: ca-agent-demo.tool/v1`）；包含 12 个 Agent-facing Operations。
- 包含 `spec.contracts[*].operation`、`adapter`、`effect`、`inputSchema`、`outputSchema`、可选 `reconcileOutputSchema`。
- 使用 YAML Anchor/Alias（如 `&ref`、`*ref`），导入必须正确处理，限制循环、膨胀及非法结构。
- 这是 Agent-facing Tool 合同，不是外部 Provider/Fabric API 的权威定义，也不是原生 MCP 消息。

**不是 Library 主输入**：`ca-agent-demo/tools/registrations/default.yaml`。它只将 Capability/Operation ID 映射到 Deployment 已装配的 Implementation ID；不能提供 I/O Schema。Compiler 不读取这份配置来判断工具已实现、已授权、READY 或安全可执行。

### 4.2 导入描述清单（非部署配置）

可选 `lib/manifest.yaml`（真实样例中已提供）声明 `libraries[*].path/source/status`，以路径匹配已导入的 ToolCatalog；`status` 仅允许 `declared`、`proposed`、`unspecified`，未声明默认 `unspecified`。这只是 Library 接口成熟度，**三种状态都不代表 Runtime READY**。不能从文件名、YAML 注释或 Operation 名称推断状态。

注意：现有早期 `validPath` 实现要求路径至少三段，会错误拒绝 `lib/manifest.yaml`（两段）；本地 WP0 必须修复并新增负例，禁止为通过测试删除 manifest。

### 4.3 MCP 导入扩展

按可替换解析器分离 `import/toolcatalog-v1`、`import/mcp-tools-list-json`（后者可作为第二步或后续功能）和未来 `import/live-mcp`。三者统一到只读库模型：

~~~text
LibraryTool {
  libraryId
  sourcePath
  operationId
  description?
  inputSchema
  outputSchema?              # 缺失明确标注 UNKNOWN
  reconcileOutputSchema?     # ToolCatalog 扩展
  declaredEffect?            # 仅来自本地受信合同
  annotationHints?           # MCP hints 仅展示，不视为 Effect 权威
  sourceStatus               # imported manifest: declared | proposed | unspecified (all NOT_READY)
}
~~~

- v0 仅离线导入文件；不连接 MCP Server、不调用 `tools/call`、不执行脚本、不保存凭据/endpoint。
- MCP 标准 `tools/list` 常有工具名称与 JSON Schema，但 `annotations` 是提示，不是安全验证/授权。
- 工具名冲突、同一 Operation 被不同库重复声明时给出诊断，不默选胜者。
- 未来从 MCP Server 刷新时形成新的导入快照与差异提示，不默默修改作者源码或已构建输出。

## 5. 编译与链接职责

~~~text
skills/** (Source)          lib/** (read-only contracts)
       \                      /
        -> Parse / Normalize
        -> Resolve Skill + Tool References
        -> Link producer schemas into per-call Script input projections
        -> Generate strictly typed Script-owned OUTPUT schemas
        -> Minimal Static Diagnostics
        -> Link per-target Skill dependency closure
        -> Materialize output file map
             |
             +-> Native CLI writes directory
             +-> Studio displays build result and exports ZIP/folder
~~~

- 使用 Skill 本来声明的 `entryProcedure`、Script 文件、Predicate 引用、依赖关系；不硬编码 `CreatedSession`、`MAS`、`5GC` 或 Tool Server 名。
- 允许跨 Skill `procedure: {skill, procedure}` 关系，但在 v0 先保证依赖解析/存在性/目标文件集合正确；旧平台高度复杂的 Runtime 图 lowering 不能以“构建校验”名义重新引入。
- 本仓没有旧格式兼容义务。Compiler 只实现 Procedure Link v0 定义的语法；旧 Skill 中不能精确表达的新要求须在迁移工程中修改，不能加 fallback 或默默略过。
- 不验证 Tool implementation/运行环境；“编译成功”只声明包结构与静态链接检查完成。
- 输出的 ToolCatalog snapshot 是方便携带的**外部合同副本**，不赋予 Tool、Capability、Deployment 的真实权限，也不隐含 Tool 实现可用。

## 6. 唯一 Core API（Native 与 Web 共用）

Conceptual API；正式语言绑定在实现时以测试固定。Studio **不**把 UI 状态、Workspace Revision、Release 等传给 Compiler。

~~~typescript
type SkillFile = { path: string; bytes: Uint8Array }; // relative to project root
type Diagnostic = {
  severity: "error" | "warning" | "info";
  code: string;
  path: string;                // points to real source or imported lib
  line?: number;
  column?: number;
  message: string;
};
type BuildRequest = {
  files: SkillFile[];          // skills/** and lib/** only; no output/**
  targets?: string[];         // absent = all Skills
};
type BuildResult = {
  success: boolean;
  diagnostics: Diagnostic[];
  outputFiles: SkillFile[];    // relative to output/, e.g. skills/...
};
interface SkillCompiler {
  check(input: BuildRequest): Promise<Diagnostic[]>;
  build(input: BuildRequest): Promise<BuildResult>;
}
~~~

- 同一 Core 的纯文件 Map 构建（无数据库/宿主 File IO）；CLI 和 WASM 都是薄 adapter。
- `outputFiles` 不是 ZIP 字节；原生 CLI 将其写到 `--out`，Studio 使用已经存在的 ZIP 和目录保存适配。
- 诊断必须有来源路径与行/列（能确定时），未知时不得捏造；Studio 用于 Problems / 编辑器定位和只读 Library 预览。
- 同一规范输入在 Native 与 WASM 应有相同诊断集合与输出字节；比较以内容为准，无需发布 Digest。
- 默认不引入第三方服务或远程请求。

### 6.1 CLI 目标

~~~bash
skillc check ./project
skillc build ./project --out ./project/output
skillc build ./project --skill establish-pdu-session --out ./dist
~~~

命令仅是目标接口，尚未实现；不保留 `caskill workspace/qualify/release` 为新产品 API。

### 6.2 Browser / Pages

- Go Core 构建为 `GOOS=js GOARCH=wasm`（具体工具链/打包大小/性能须原型验证）。
- Worker 按需加载 WASM，不依赖服务器端 Go 进程、SQLite、Node.js 或外部 Python。
- Studio 管理文件会话、只读 Lib、Source Patch、授权保存、ZIP 下载；Compiler 不管理编辑器或用户权限。
- 静态站托管时应连同正确版本的 Go WASM JS runtime 一起分发；要有真实 Chromium + Pages 离线数据/性能验收。
- 不把“WASM 能启动”当“与原生编译语义一致”，必须进行真实 Skill 样本端到端对照。

## 7. Agent Host / Runtime 消费边界（允许破坏性切换）

**无需兼容旧 Skill / Toolchain 产物格式。** 在新 Compiler 的 v0 DSL 和 Bundle 确认之后，消费端可以通过**独立 PR**调整其加载/身份/Admission 接口，不能要求 Compiler 复活 Release/Artifact/legacy Script adapter。Compiler 本次工作流**不得写入其它仓**。

真实 Host 的执行安全（动态运行校验、sandbox、Tool authorization、admission）仍归执行宿主所有，不因为容许破坏性 API 切换就省略。新 ZIP/文件树成功只能证明静态编译；不证明 Host READY。Host-side 适配/真实环境执行标记为 **单独验收阶段，NOT_RUN**。

固定的两套 ca-agent-demo 原始 Skill 是迁移材料，不是运行兼容性 PASS 门槛。新格式的迁移工程须单独构建；缺真实 Host/Observation Owner Schema 的字段应继续报精确依赖缺口。

## 8. 重写策略与切换顺序

1. **D0 文档与边界**：冻结新 v0 的 Script/Procedure/Library/Studio 接口；无需保留旧产品的兼容层。
2. **C1 新纯 Core**：在本独立仓库实现 `check/build`，只参考必需 Parser/Validator/Skill linker；不做旧 Service 的删改式重构。
3. **C2 Library**：导入真实 ToolCatalog（含 Alias）、负例、只读接口投影。
4. **C3 输出**：多 Skill / 单 Skill / 依赖闭包 / generated schemas / 可拷贝文件集 / Native CLI。
5. **C4 WASM**：同一 Core 构建并在 Browser Worker 运行；对齐 Native golden vectors、时间和体积。
6. **C5 消费者验收**：Studio 使用新 API 集成；Agent Host 的新包格式与 Admission 消费作为独立破坏性切换 PR，未做不得声称 Runtime READY。
7. **C6 消费者切换与旧仓退役**：新仓和 Go module 均已命名为 `ca-skill-compiler`；在通过 C5 之后更新消费者 imports/schema snapshot/文档和工作流。历史 `ca-skill-toolchain` 仅保留归档参考，不为兼容而恢复旧平台。

旧仓 Draft [#132](https://github.com/odango-chan/ca-skill-toolchain/pull/132) 是旧平台上 Predicate/Artifact 变更，需提取必要语言语义与测试，不直接合并重构分支；Issue [#129](https://github.com/odango-chan/ca-skill-toolchain/issues/129) 的 invocation-local Predicate 绑定风险仍须在跨 Skill 消费端审视。

## 9. 验收矩阵与交接（给 Skill Studio 新项目）

| Case | 检查点 | 最低预期 |
| --- | --- | --- |
| SC-01 | 导入 Demo ToolCatalog YAML | 12 Operations 可读，包括 Anchor/Alias，参数 Schema 可展示 |
| SC-02 | `skills/` 下 2 个以上 Skill | 全部/单 Skill 构建；子目录身份不串 |
| SC-03 | Skill 依赖另一个 Skill | 找到依赖、输出闭包、缺依赖诊断 |
| SC-04 | Tool 引用 | 使用导入 Operation 可链接；未知/重名报错 |
| SC-05 | Script Schema | Tool/Host 输入从真实 Producer 绑定；Script 自有严格 Output Schema 生成，旧脚本不得 fallback |
| SC-06 | Studio 的只读 `lib/` | 显示来源，无法直接写入；更新经重新导入 |
| SC-07 | Native/WASM | 同样 BuildRequest 得到同样 outputFiles + diagnostics |
| SC-08 | ZIP / Folder | 解压/复制后路径一致，`output/` 不回流进 Source |
| SC-09 | Host 消费 | 与新格式配套的新 Host 端 Adapter 单独验证；不为旧格式保留兼容代码；NOT_RUN |
| SC-10 | 负例/安全 | 非法路径、重复工具、坏 YAML、Schema 不匹配等拒绝；无 Tool/Python 执行 |

**验收细则：[v0 Acceptance Matrix](../workstreams/acceptance-matrix-v0.md)。**

**Skill Studio 新项目的最小交接**：采用三根顶层逻辑树 `skills/`（可编辑）、`lib/`（外部只读）、`output/`（虚拟只读）；由 Studio 提交完整文件字节到 `check/build`，只读取 diagnostics/outputFiles；Studio 自己控制菜单、Explorer、Problems、Visual、预览和 ZIP。不要复制 Parser/Linker/Schema Generator 的规则。接口以本文件 §6 和 §9 为准。

## 10. 现状与信心边界

- 旧 `ca-skill-toolchain/main` 仍包含 Registry/Profile/Report、SQLite、Scenario/Planning/Mock、Qualification/Release；这些组件不会迁入新仓。
- 目前已观察到 `backend/internal/compiler/`、`backend/internal/service/release.go`、`backend/internal/store/` 等遗留组件。
- 本文中的“ACCEPTED”只表示产品范围与原则已确认；C1-C4 仅有部分初始源码；C5-C6/SC-01..10 验收均 **NOT_RUN**，不能宣称新编译器或 WASM/Studio 集成成功。
- 实现追踪见 [本仓 Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1)；架构决策历史见 [旧仓 Issue #133](https://github.com/odango-chan/ca-skill-toolchain/issues/133)。