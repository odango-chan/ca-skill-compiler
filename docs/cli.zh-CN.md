# skillc 中文命令行手册（v2.1.0）

[English](cli.md) · [文档目录](README.zh-CN.md) · [正式 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0)

**正式产品：v2.1.0。唯一语言目标：`connectionagent.io/v1alpha5`。** 命令行与 Go/WASM 共用一个 Compiler Core；不会访问网络解析依赖、运行 Python/Tool、发布包或执行 Host 准入。

## 命令

| 命令 | 输入 | 输出 |
|---|---|---|
| `skillc check PROJECT` / `--request F` | 冻结 v1alpha5 BuildRequest 或项目目录 | 静态诊断，不写入 Artifact |
| `skillc inspect PROJECT` / `--request F` | 冻结 v1alpha5 请求 | 只读 DependencyInspection |
| `skillc build PROJECT --out DIR` / `--request F --out DIR` | 冻结 v1alpha5 请求 | 经验证的确定性 Artifact |
| `skillc build-development --request F --out DIR` | Development v1alpha1 请求 | 确定性 `NOT_ADMITTED` Artifact |
| `skillc verify-development --request F --result R` | 原始 Development 请求和完整 BuildResult 字节 | 重新构建并逐字节验证 |
| `skillc version` / `skillc help` | 无 | 产品/API 版本与帮助 |

普通 `check / inspect / build` 支持重复 `--skill NAME`、`--json`、`--request FILE`。当显式请求已指定非空 `targets` 时不能用 `--skill` 覆盖。Development 子命令**仅**接受 `--request` 与对应的 `--out` 或 `--result`，另可加 `--json`；不接受项目目录或 `--skill`。

**退出码：** `0` 成功；`1` 静态校验/Artifact/Verify 失败；`2` 用法、输入格式或 I/O 错误。

## 普通项目编译（冻结 v1alpha5）

```text
PROJECT/
  skills/...                     或 files/skills/...
  snapshots/<dependencyId>/...   调用方提供的不可变依赖字节
  trusted.json                   本地依赖与 Owner/Capability/Evidence 绑定配置
```

`trusted.json` 仅供本地加载，**不是**发布的机器协议。Compiler 会复算摘要并严格校验依赖和可信上下文，不猜测缺失的 Owner 或 Schema。机器调用方应使用完整的 `--request`。

```bash
skillc check web/parity-success
skillc inspect web/parity-success --json
skillc build web/parity-success --out ./dist/frozen-artifact
skillc build --request request.json --out ./dist/frozen-from-json --json
```

[冻结 BuildRequest 机器契约](../contracts/v1alpha5/build-request.schema.json)。构建成功后先验证再写入；非空目标目录、不安全路径、符号链接均 fail-closed。旧接口 Artifact 的 `generator.version=v2.0.2` 是冻结身份，**不是**当前 CLI 版本。

## Development Build v1alpha1

请求必须显式提供 `files[]`、`targets[]`、`lock`、`snapshots[]` 及版本身份；依赖快照精确包含 `package.json`、导出文件和直接依赖闭包。此接口不会接收实际 Owner 信任根、Provider、Deployment Grant 或运行时 Evidence。

```bash
mkdir -p ./dist
skillc build-development --request docs/design/fixtures/registration-development-request.json --out ./dist/development --json > ./dist/development-result.json
skillc verify-development --request docs/design/fixtures/registration-development-request.json --result ./dist/development-result.json --json
```

第二条命令需要第一条的**完整 BuildResult JSON（含 Base64 Artifact 字节）**，不能只传 Manifest。输出目录至少包括：

```text
manifest.json
host-admission-requirements.json
runtime-execution-requirements.json
skills/<skill>/compiled/procedure.json
skills/<skill>/compiled/ingress-links.json
```

Manifest 含 `capabilityRequirements[]`，`executionStatus=NOT_ADMITTED`。**`NOT_ADMITTED` 是编译成功时正常的状态，生产 Host 不得据此执行。** 五份 [Development v1alpha1 机器契约](../contracts/development/v1alpha1/README.md) 在 v2.1.0 正式发布并冻结。

## 验收

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
```

Chromium parity 需要 Node 与真正的 Chrome/Chromium；CI 会安装。v1alpha4 CLI 说明仅供历史查阅：[中文旧版](https://github.com/odango-chan/ca-skill-compiler/blob/72b41548a9c4d29c6c49abed117a04a963454f22/docs/cli-v1alpha4.zh-CN.md) · [English old version](https://github.com/odango-chan/ca-skill-compiler/blob/72b41548a9c4d29c6c49abed117a04a963454f22/docs/cli-v1alpha4.md)。
