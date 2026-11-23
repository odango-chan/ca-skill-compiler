> **历史文档 / HISTORY ONLY — Compiler v1.0.0 / CA-SL v1alpha4。** 本文件完整保留已发布的早期 CLI 描述以供追溯；当前 Compiler v2.1.0 **仅**支持 CA-SL v1alpha5，且新增 Development v1alpha1 入口。应使用 [当前中文 CLI](cli.zh-CN.md) · [English](cli.md)。不得把下列命令与格式作为当前输入规范。

---

# skillc CLI v1alpha4

[English](cli-v1alpha4.md)

本文说明 CA Skill Compiler v1.0.0 提供的 `skillc` 命令行工具。该版本面向冻结的 CA-SL v1alpha4。

CLI 只是编译器核心的命令行适配层，与 Go/WASM Worker 和 Skill Studio 使用相同的 `compiler.Check`、`compiler.Build` 和 `compiler.InspectDependencies`。CLI 不包含另一套解析器、链接器、摘要算法或分析流程。

## 1. 命令

| 命令 | 用途 | 编译器核心调用 |
|---|---|---|
| `skillc check` | 仅执行静态检查，不写入制品 | `Check(BuildRequest)` |
| `skillc build` | 编译、链接、验证并写入确定性制品目录 | `Build(BuildRequest)` + `VerifyBuildResult` |
| `skillc inspect` | 只读查看依赖快照和可信绑定清单 | `InspectDependencies(BuildRequest)` |
| `skillc version` | 输出 Compiler 产品版本和目标语言版本 | `compiler.CompilerVersion` |
| `skillc help` | 显示帮助信息 | — |

CLI 没有其他子命令。它不负责依赖解析、包管理、网络下载、发布制品或 Runtime 调度。

## 2. 参数

| 参数 | 适用命令 | 说明 |
|---|---|---|
| `--request FILE` | 全部 | 读取完整的 BuildRequest v1alpha4 JSON 文档。严格解码，拒绝未知字段；使用该参数时不再提供 `PROJECT` 参数。 |
| `--out DIR` | `build` | 制品输出目录。目录必须不存在或为空。项目目录模式默认写入 `<PROJECT>/output`；使用 `--request` 时必须显式指定。 |
| `--skill NAME` | 全部 | 选择一个 Skill 构建目标；可重复指定多个。若 `--request` 中已包含 `targets`，则不能同时使用此参数。 |
| `--json` | 全部 | 将机器可读结果写到 stdout，并抑制面向人的诊断和摘要文本。 |

参数可以放在 `PROJECT` 前面或后面。使用 `--` 后，后续内容不再按参数解析。

## 3. 输入方式 A：项目目录

项目目录采用以下布局：

```text
PROJECT/
  skills/...                        Skill 源码（必需）
  # 或 files/skills/...              parity / Studio 测试使用的等价布局
  snapshots/<dependencyId>/...      每个已声明 Alias 对应的不可变依赖快照
  trusted.json                      可信依赖绑定和部署信息
  <capabilityDirectory>/...         可信 CapabilityContracts 文件集
  <evidenceDirectory>/...           可信 EvidenceTrustBindings 文件集
```

如果存在 `PROJECT/files`，它会成为源码根目录，Skill 文件从 `PROJECT/files/skills` 读取；否则从 `PROJECT/skills` 读取。只有 `skills/` 下的文件属于作者输入。符号链接、非普通文件、超过 2 MiB 的文件，以及超过 512 个文件或总计 16 MiB 的项目，都会在解析前被拒绝。

`trusted.json` 是供可信调用方使用的**本地输入描述**，不是冻结的线上契约。CLI 会使用冻结的 `CA-SKILL-DEPENDENCY-SNAPSHOT\0v1\0` 算法自行计算每个依赖快照的摘要。`trusted.json` 中没有可供调用方填写的摘要字段，因此调用方不能自行声明一个摘要并要求 CLI 接受。该文件描述哪些字节受到信任，而不是预先声明信任校验结果。

示例：

```json
{
  "dependencyBindings": [
    {
      "dependencyId": "ca",
      "stablePackageId": "urn:example:ca:tools",
      "authorizedOwners": ["example-ue-owner"],
      "exportOwners": [
        { "exportKind": "operation", "exportId": "ue.registration", "owner": "example-ue-owner" }
      ]
    }
  ],
  "deployment": {
    "deploymentId": "urn:example:deployment:smf",
    "publisherId": "urn:example:publisher:deployment",
    "authorizationRef": "urn:example:authorization:example-deployment",
    "capabilityDirectory": "capability",
    "evidenceDirectory": "evidence"
  }
}
```

- 只有当一个快照包含多个 Owner 的导出项时，才需要提供 `exportOwners`；单 Owner 快照使用 `authorizedOwners`。
- `capabilityDirectory` 和 `evidenceDirectory` 是相对于 `PROJECT` 的路径。CLI 会分别将它们作为可信 Bundle 处理并重新计算摘要。只要其中任意一项存在，部署授权记录也会一并传给 Compiler。
- 如果某个绑定所指向的快照目录不存在或为空，则属于输入错误，退出码为 `2`。CLI 不会用空快照或推断出的快照代替。

`trusted.json` 是可选的。如果没有该文件，BuildRequest 中就不会包含已解析依赖和可信构建上下文。若 Skill 声明了依赖，Compiler 会给出冻结诊断 `DEPENDENCY_SNAPSHOT_MISSING`，并以退出码 `1` 失败关闭。

## 4. 输入方式 B：显式 BuildRequest JSON

```bash
skillc check   --request request.json
skillc build   --request request.json --out DIR
skillc inspect --request request.json --json
```

该方式直接使用冻结的 [BuildRequest v1alpha4](design/build-request-v1alpha4.md) 传输结构：`files[].bytes` 使用 Base64 字符串；`resolvedDependencies` 携带完整的不可变快照文件集和摘要；`trustedBuildContext` 携带依赖绑定、CapabilityContracts 和 Evidence 信任 Bundle。解码时会拒绝未知属性，以及本应存在却为 `null` 的字段。

当经过认证的 Host 已经持有可信输入（参见 BuildRequest §5.4），或调用方需要重放一份精确捕获的请求时，应使用这种方式。

CLI 不会在项目目录模式和 BuildRequest 模式之间隐式转换，也不会写出 request JSON。`cmd/parity-replay` 可以为一致性测试生成从项目目录派生的请求，但它是测试工具，不属于 CLI 契约。

## 5. 输出

### 5.1 面向人的文本输出

诊断信息写入 stderr，每条一行，包含源码位置、阶段和冻结的诊断代码：

```text
skills/parity/assurance/procedure.yaml:30:9: error C_STATIC YAML_PARSE_ERROR: [30:9] sequence end token ']' not found
<request>: error C_STATIC DEPENDENCY_SNAPSHOT_MISSING: No supplied immutable DependencySnapshot has the used Dependency Alias as its dependencyId: ca
```

请求级诊断使用 `<request>` 作为位置。解析错误的消息可能包含长度受限的源码上下文。

命令成功时，会向 stdout 输出简短摘要，例如：

```text
Skill project static check OK
Dependency inspection OK: 1 callable(s), 3 schema(s)
Skill build complete: 5 file(s), 2 dependenc(ies) -> .../output
```

### 5.2 机器可读输出（`--json`）

| 命令 | stdout 内容 |
|---|---|
| `check` | `Diagnostic` 对象数组，即 [Compiler Output API §1](design/compiler-output-contract-v1alpha4.md) 定义的冻结 Check 输出结构。 |
| `build` | [BuildResult v1alpha4](design/compiler-output-contract-v1alpha4.md)，包含 `success`、`diagnostics`、排序后的 `dependencyClosure` 和 Base64 编码的 `outputFiles`。 |
| `inspect` | [DependencyInspection v1alpha4](design/compiler-output-contract-v1alpha4.md)。 |

即使使用 `--json`，进程退出码仍然表示成功或失败。JSON 文档是编译器结果，不是授权令牌。

## 6. 退出码

| 退出码 | 含义 |
|---|---|
| `0` | 成功：没有 `C_STATIC` 错误诊断；对于 `build`，制品已通过验证并写入。 |
| `1` | 编译、链接、制品验证或输出写入失败，并已报告诊断。 |
| `2` | 在编译前或编译后的输入/输出阶段发生用法错误、输入缺失或格式错误、不安全的项目目录布局或 I/O 错误。 |

`check` 和 `inspect` 在存在任何 `severity=error` 的诊断时返回 `1`。`build` 在 `BuildResult.success` 为 `false` 时返回 `1`。

失败的 Build 不会写入制品目录。成功的 Build 也会先通过 `VerifyBuildResult` 进行结构验证，之后才写入文件。

## 7. Fail-closed 行为

CLI 与 Compiler Core 遵循相同的信任边界：

1. 已声明依赖但没有提供快照时，产生 `DEPENDENCY_SNAPSHOT_MISSING`（退出码 `1`）；可信绑定指向的快照目录不存在时，属于输入错误（退出码 `2`）。CLI 不会推断或下载依赖。
2. Alias 必须且只能解析到一个可信绑定和一个快照。Alias 不能绕过 Stable Package ID 或 Owner 校验。
3. 导出项的 Owner 缺失或存在歧义时，使用冻结的 Dependency/Owner 诊断报错，不会根据文件顺序、路径或显示名称猜测。
4. Capability Grant 只会根据调用方提供的 CapabilityContracts 部署 Bundle 校验，CLI 不会自行创建 Grant。
5. 输出只能写入新的目录，不能经过符号链接祖先目录，也不能覆盖作者源码；写入前会重新验证每个输出路径。
6. 构建失败或验证不完整时，不会留下任何制品目录内容。

## 8. 使用示例

使用仓库中已准备好的可信测试项目：

```bash
skillc version
skillc check   web/parity-success
skillc inspect web/parity-success
skillc build   web/parity-success --out /tmp/skillc-out

skillc check   web/parity-success --json
skillc build   web/parity-success --out /tmp/skillc-out --json
skillc inspect web/parity-success --json

skillc check   web/parity-success --skill verify-ue-registration
```

不安装二进制文件，直接从 Go 源码运行：

```bash
go run ./cmd/skillc check ./web/parity-success
```

该示例构建的预期制品如下。重复构建时，已验证生成字节完全一致：

```text
skills/verify-ue-registration/SKILL.md
skills/verify-ue-registration/assurance/contract.yaml
skills/verify-ue-registration/assurance/predicates.yaml
skills/verify-ue-registration/assurance/procedure.yaml
skills/verify-ue-registration/generated/artifact-manifest.json
```

## 9. 验证

CLI 端到端测试会先构建真实二进制，再以子进程方式执行：

| 测试 | 覆盖内容 |
|---|---|
| `TestCLIEndToEndTrustedBuildAndDeterminism` | 成功的 check/build、真实文件写入、重复构建字节一致性。 |
| `TestCLIEndToEndJSONContracts` | Check 诊断数组、BuildResult、DependencyInspection 的线格式。 |
| `TestCLIEndToEndHumanInspect` | 带 Owner 归属信息的文本依赖清单。 |
| `TestCLIEndToEndDiagnosticsArePositioned` | 文本和 JSON 中的 `path:line:column`、阶段及冻结诊断代码。 |
| `TestCLIEndToEndMissingDependencyFailsClosed` | 依赖快照缺失时不会静默成功。 |
| `TestCLIEndToEndRequestTransport` | BuildRequest JSON 模式、错误请求和参数冲突。 |
| `TestCLIEndToEndOutputFreshness` | 拒绝已存在且非空的输出目录。 |
| `TestCLIEndToEndSkillTarget` | `--skill` 目标选择和未知目标处理。 |
| `TestCLIEndToEndVersion` | 产品版本、目标语言版本输出及非法参数处理。 |

运行命令：

```bash
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
```
