> **Current v0 wire implementation only.** The frozen successor design is [Plugin-managed Skill Project v0.1](design/plugin-managed-project-v0.1.md), with [machine schemas](../contracts/v0.1/README.md). v0.1 is NOT available in current native/Go/WASM binaries; do not send its requests to the v0 Worker.

# Skill Studio ↔ Skill Compiler v0 · Browser ABI

状态：**DESIGN ACCEPTED / API PARTIAL IMPLEMENTATION / BROWSER SMOKE PASS**。适用 **breaking-only v0**；不对接旧 Toolchain / legacy Script Parser。

## 1. Project 与读写职责

~~~text
project/
  skills/   # author editable, multiple Skills
  lib/**    # logical Compiler input namespace; Studio may supply imported contracts in memory
  outputFiles  # virtual BuildResult output; no physical output/ directory required
~~~

- Studio 管理 Explorer、Source Editor、导入/删除/重新导入库、文件授权、Problems、ZIP/文件夹导出；Compiler 管理唯一的 Parse/Link/Schema Generator。来源发现、持久化、刷新、缓存、配置和导出均由 Studio/调用方负责，Compiler 只接收准备好的文件字节。
- Browser/Pages 中全部源码+Library 通过 in-memory 文件字节传入 Go/WASM Worker；不需要服务器端编译服务/数据库/代理。`lib/**` 是 Compiler 输入命名空间，不要求用户工作空间存在物理 `lib/` 目录。
- Studio 不得把 `output/` 文件作为 Source 提交，不允许编辑 imported Tool Schema 或生成文件本体。重新导入 Library 是替换 imported snapshot，不是改源脚本。

## 2. Core API（实现时冻结 Go/TS 对齐）

~~~typescript
type File = {path: string, bytes: Uint8Array}; // absolute disallowed, project-relative only
type Diagnostic = {severity: "error"|"warning"|"info", code: string, path: string,
  line?: number, column?: number, message: string};
type BuildRequest = {files: File[], targets?: string[]};
type BuildResult = {success: boolean, diagnostics: Diagnostic[], outputFiles: File[]};
type LibraryTool = {
  operationId: string, sourcePath: string,
  sourceStatus: "declared"|"proposed"|"unspecified",
  declaredEffect?: "READ"|"MUTATION",
  description?: string,
  inputSchema: object, outputSchema?: object, reconcileOutputSchema?: object
};
type LibraryResult = {diagnostics: Diagnostic[], tools: LibraryTool[]};

check(input: BuildRequest): Diagnostic[];
build(input: BuildRequest): BuildResult;
inspectLibraries(input: BuildRequest): LibraryResult;
~~~

- `check` 与 `build` 静态语义相同；`check` 不返回 output 文件。`build` 有 ERROR 时 `success=false` 且 outputFiles 为空。输出按稳定 path 排序。`inspectLibraries` 只读取 lib 的契约，展示来源与接口，不自行运行代码，不由 Studio 再造第二份 Parser。
- Imported `lib/manifest.yaml` 与 `lib/tools/*.yaml` 同时作为请求文件。Manifest 的 `libraries[*].path` 相对 Project root，状态只有 `declared/proposed/unspecified`，**任何状态都不是 Runtime READY**；缺 Manifest 默认 unspecified。
- V0 不为每个 Script 构造一个可编辑的 Input Schema；输入在 Procedure 的 `with:` 中通过真实 Producer Schema **逐调用点**派生。Script 自己计算出的结果要求严格 Output TypedDict；Compiler 仅输出其生成的 Output Schema。
- 无真实 Host/Observation Owner schema 时，Compiler 报 `EXTERNAL_SCHEMA_MISSING`（即使 request 的其他输入字段能被检查），**不能声称完整静态类型检查通过**。

## 3. JS ↔ Go/WASM wire

Go `encoding/json` 会把 `[]byte` 作为 base64 string；**Worker 边界**使用该表现，Studio 可使用 Uint8Array：

~~~json
{
  "files": [
    {"path": "skills/example/SKILL.md", "bytes": "IyBFeGFtcGxlCg=="}
  ],
  "targets": ["example"]
}
~~~

Go Core exposes `Check/Build/InspectLibraries`; matching WASM callbacks are:
- `skillcCheck(jsonRequestString): jsonResultString`
- `skillcBuild(jsonRequestString): jsonResultString`
- `skillcInspectLibraries(jsonRequestString): jsonResultString`

Worker request/response:

~~~typescript
postMessage({id: "r1", command: "check"|"build"|"inspect", request: BuildRequestAsBase64});
onmessage({data: {id: "r1", result?: BuildResult|LibraryResult, error?: string}});
~~~

- `error` 表示 Worker/Go/WASM 宿主异常，**不是**普通 Compiler Diagnostic；Compiler 静态验证错误通过 `result.diagnostics` 返回。不要把二者混为一谈。
- ToolCatalog 可能含 YAML anchors/aliases；由 Go Parser 消解，Studio 不承担 YAML Schema 的语义解析。
- 不在 Worker 中携带 MCP 凭据、Server URL 或 Runtime admission token，不发生远程 Tools 调用。

## 4. Browser 构建与验收

~~~bash
mkdir -p dist
GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm
WASM_EXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC" ]; then WASM_EXEC="$(go env GOROOT)/misc/wasm/wasm_exec.js"; fi
cp "$WASM_EXEC" dist/wasm_exec.js
cp web/skillc-worker.js dist/skillc-worker.js
cp web/diagnostics-golden.json dist/diagnostics-golden.json
~~~

静态资源应与 Go/WASM runtime **版本匹配**。Worker 的资源路径相对于 Worker 文件解析。`web/skillc-browser-test.html` 是可执行 smoke page；它加载 `script-contract-project` 与 Native CLI 生成的文件 golden，并调用 Worker 的三个 API 与负例。`web/diagnostics-golden.json` 由 Native Go Core 测试验证，供浏览器对照相同的失败诊断。

从仓库根目录准备静态目录：

~~~bash
mkdir -p dist
GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/wasm_exec.js
cp web/skillc-worker.js dist/skillc-worker.js
cp web/skillc-browser-test.html dist/index.html
cp web/diagnostics-golden.json dist/diagnostics-golden.json
cp -R examples/script-contract-project dist/project
GOWORK=off go run ./cmd/skillc build ./examples/script-contract-project --out dist/native-output
python3 -m http.server 8000 --directory dist
~~~

Open `/` in Chromium; the page marks `data-test="pass"` only after check/build/inspect, exact missing-target and binding diagnostic goldens, stable file order, and every WASM output byte match the Native golden files, including the generated Script Output Schema. This smoke page passed in the local Chromium run. It is not a full Native/WASM diagnostic parity suite or Pages deployment test.

最低通过项：同样的 project 文件集合（包括 Tool YAML/Manifest）在 Native Go 与 JS/WASM 返回一样的诊断码、source path、output file path 和字节；两份真实 5GC pinned 快照只用于迁移缺口说明，**不要求未经迁移的旧源码 PASS**。

## 5. 与 Host / MCP 的边界

- Tool Library 是**被导入的接口文件**，不是 Tool Implementation、MCP Server 连接或部署注册。
- 外部 Host request/context/observation 的 Schema 只能从真实 Owner 导入。当前 5GC 工程中这部分尚不完整；不可复制路由配置 Schema、不得自造以通过测试。
- Host 对新构建包的加载可以**另立破坏性切换 PR**。不要为了兼容历史 Runtime Artifact/Release 把这些内容重新放进 Compiler。
- Compiler Build.success 只表示新 DSL 的静态链接完成；不证明脚本执行、MCP 可用、MAS READY 或 5GC E2E。
