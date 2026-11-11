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
4. 支持 Script 接口声明到 Schema 的**渐进式**生成；旧 Script 没有足够接口声明时，保留作者现有 Schema、给出明确诊断，不能推断函数体中不存在的约束；
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
    shared/                       # 后续公共接口/其他非 Skill 依赖
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
| Script I/O Schema（可由明确类型/元数据推导） | Script 接口声明 | 间接编辑声明 | 生成到 Output |
| 不能可靠推导的 Script I/O Schema | 现有作者 Schema | 迁移前可编辑 | 保留现有文件并提示将来迁移 |
| Tool Operation Input/Output/Reconcile Schema / Effect | 外部 Tool Contract Owner | 否 | 导入链接、只读浏览、按需打包快照 |
| Studio 用来支持 YAML 诊断的 Meta-Schema | 语言规范/Compiler | 否 | 内置编译器/Studio，不进入 Skill 包 |
| Qualification/Profile/Release/Runtime 状态 | 外部相应 Owner | 否 | 不纳入 v0 |

### 3.1 Script-to-Schema · Producer-owned 接口

**唯一规范：[Script Contract v0](script-contract-v0.md)，设计确认，代码未实现。** 此规约取代了「所有 Script 输入输出均需 TypedDict」的旧方案。

- **外部输入**来自其真实 Producer 的 Schema：Tool/MCP 从只读 `lib/tools/` 导入，Host context/observation 从权威 Host/Ingress/Observation 契约导入，Skill 自有 request 从对应 `skills/*/schemas/` 读取。
- Procedure `with:` 根据每个 `from:` 绑定形成 Script 的调用点 **input projection**；作者**不必重复定义**外部 Input TypedDict。但缺少外部 Schema、字段未知或类型冲突不能宣称严格验证通过。
- Script **自己计算产生的 Output** 必须有完整 `TypedDict` / 受限 `Annotated` 声明，由 Compiler 静态生成只读 Output Schema。`main(input: dict) -> OwnOutput` 里的 dict 只是 Python 输入容器，不意味着允许无契约数据。
- 现有 5 个 5GC Script 保留原始 external input/output Schema，不篡改复杂 `allOf/if/then` 约束；正式 ToolCatalog 和 MAS proposed ToolCatalog 已导入样例项目。**MAS 仅供静态链接，不 READY**。
- 不执行 Python、不从函数体推断接口。新旧两个被撤回方案——脚本头内 embedded YAML、强制所有 Script Input TypedDict——**均不得实现**。
- Root Routing Profile 是部署路由配置，不是完整 Host 输入 Schema；缺少权威 Host 数据契约时需要诊断，禁止自行造一个。

## 4. Library v0：先文件导入，后 MCP

### 4.1 第一版导入文件

主样本：`odango-chan/ca-agent-demo/tools/contracts/createdsession.tool.yaml`
- 当前是 `ToolCatalog`（`apiVersion: ca-agent-demo.tool/v1`）；包含 12 个 Agent-facing Operations。
- 包含 `spec.contracts[*].operation`、`adapter`、`effect`、`inputSchema`、`outputSchema`、可选 `reconcileOutputSchema`。
- 使用 YAML Anchor/Alias（如 `&ref`、`*ref`），导入必须正确处理，限制循环、膨胀及非法结构。
- 这是 Agent-facing Tool 合同，不是外部 Provider/Fabric API 的权威定义，也不是原生 MCP 消息。

**不是 Library 主输入**：`ca-agent-demo/tools/registrations/default.yaml`。它只将 Capability/Operation ID 映射到 Deployment 已装配的 Implementation ID；不能提供 I/O Schema。Compiler 不读取这份配置来判断工具已实现、已授权、READY 或安全可执行。

### 4.2 MCP 导入扩展

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
        -> Generate declared Script I/O Schemas
        -> Minimal Static Diagnostics
        -> Link per-target Skill dependency closure
        -> Materialize output file map
             |
             +-> Native CLI writes directory
             +-> Studio displays build result and exports ZIP/folder
~~~

- 使用 Skill 本来声明的 `entryProcedure`、Script 文件、Predicate 引用、依赖关系；不硬编码 `CreatedSession`、`MAS`、`5GC` 或 Tool Server 名。
- 允许跨 Skill `procedure: {skill, procedure}` 关系，但在 v0 先保证依赖解析/存在性/目标文件集合正确；旧平台高度复杂的 Runtime 图 lowering 不能以“构建校验”名义重新引入。
- 对于需要 Runtime 端已经认得的格式，不擅自改变 Skill Source / Procedure 的语法；不支持的特性要清晰报错，不得无声丢失语义。
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

## 7. 对现有 Agent Host / Runtime 的兼容边界

现在 `ca-agent-demo` 使用自己的 exact `RuntimeSkillIdentity/ExactSkillIdentity`、Admission、工具注册与 Runtime 加载。v0 Compiler 不强制产出 Toolchain 旧式 `SkillBuildArtifact/RuntimeBundleDigest/ReleaseId`；这**不意味着当前 Agent Host 不需修改就能运行新包**。

迁移前必须先做消费者审查：
1. `skills/<name>` 是否可被原样读取；
2. 哪些现有 source Schema、Script metadata、tool contracts 和 exact identity 仍是 Host 的真实必需输入；
3. 缺少旧 Artifact / Digest / Release 时，新包能否进入当前 Link / Admission；若不能，最薄的 **Host-side** 构建/身份步骤是什么；
4. 对照真实 `establish-pdu-session` 与 `establish-pdu-session-mas`、一个跨 Skill 引用案例，证明可构建与可消费；不能以 ZIP 成功替代 Host/Runtime 验收。

**原则**：Runtime 执行安全语义不能为了简化编辑器而静默删除；但也不得强行将旧平台生命周期复制到新 Compiler。

## 8. 重写策略与切换顺序

1. **D0 文档与边界**：冻结本设计、Studio handoff、命名；现有 `main` 代码保持 legacy，不宣称新架构已实现。
2. **C1 新纯 Core**：在本独立仓库实现 `check/build`，只参考必需 Parser/Validator/Skill linker；不做旧 Service 的删改式重构。
3. **C2 Library**：导入真实 ToolCatalog（含 Alias）、负例、只读接口投影。
4. **C3 输出**：多 Skill / 单 Skill / 依赖闭包 / generated schemas / 可拷贝文件集 / Native CLI。
5. **C4 WASM**：同一 Core 构建并在 Browser Worker 运行；对齐 Native golden vectors、时间和体积。
6. **C5 消费者验收**：Studio 新工程集成；Agent Host 读取生成目录并跑基础 Source/Link/Admission 回归。
7. **C6 消费者切换与旧仓退役**：新仓和 Go module 均已命名为 `ca-skill-compiler`；在通过 C5 之后更新消费者 imports/schema snapshot/文档和工作流。历史 `ca-skill-toolchain` 仅保留归档参考，不为兼容而恢复旧平台。

旧仓 Draft [#132](https://github.com/odango-chan/ca-skill-toolchain/pull/132) 是旧平台上 Predicate/Artifact 变更，需提取必要语言语义与测试，不直接合并重构分支；Issue [#129](https://github.com/odango-chan/ca-skill-toolchain/issues/129) 的 invocation-local Predicate 绑定风险仍须在跨 Skill 消费端审视。

## 9. 验收矩阵与交接（给 Skill Studio 新项目）

| Case | 检查点 | 最低预期 |
| --- | --- | --- |
| SC-01 | 导入 Demo ToolCatalog YAML | 12 Operations 可读，包括 Anchor/Alias，参数 Schema 可展示 |
| SC-02 | `skills/` 下 2 个以上 Skill | 全部/单 Skill 构建；子目录身份不串 |
| SC-03 | Skill 依赖另一个 Skill | 找到依赖、输出闭包、缺依赖诊断 |
| SC-04 | Tool 引用 | 使用导入 Operation 可链接；未知/重名报错 |
| SC-05 | Script Schema | 明确类型生成；仅空 `dict` 不伪造字段；老 Schema 保留 |
| SC-06 | Studio 的只读 `lib/` | 显示来源，无法直接写入；更新经重新导入 |
| SC-07 | Native/WASM | 同样 BuildRequest 得到同样 outputFiles + diagnostics |
| SC-08 | ZIP / Folder | 解压/复制后路径一致，`output/` 不回流进 Source |
| SC-09 | Host 消费 | 用真实生成目录验证现有 Link/Admission 兼容，未跑则标 NOT_RUN |
| SC-10 | 负例/安全 | 非法路径、重复工具、坏 YAML、Schema 不匹配等拒绝；无 Tool/Python 执行 |

**Skill Studio 新项目的最小交接**：采用三根顶层逻辑树 `skills/`（可编辑）、`lib/`（外部只读）、`output/`（虚拟只读）；由 Studio 提交完整文件字节到 `check/build`，只读取 diagnostics/outputFiles；Studio 自己控制菜单、Explorer、Problems、Visual、预览和 ZIP。不要复制 Parser/Linker/Schema Generator 的规则。接口以本文件 §6 和 §9 为准。

## 10. 现状与信心边界

- 旧 `ca-skill-toolchain/main` 仍包含 Registry/Profile/Report、SQLite、Scenario/Planning/Mock、Qualification/Release；这些组件不会迁入新仓。
- 目前已观察到 `backend/internal/compiler/`、`backend/internal/service/release.go`、`backend/internal/store/` 等遗留组件。
- 本文中的“ACCEPTED”只表示产品范围与原则已确认；C1-C4 仅有部分初始源码；C5-C6/SC-01..10 验收均 **NOT_RUN**，不能宣称新编译器或 WASM/Studio 集成成功。
- 实现追踪见 [本仓 Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1)；架构决策历史见 [旧仓 Issue #133](https://github.com/odango-chan/ca-skill-toolchain/issues/133)。