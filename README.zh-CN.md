# CA Skill Compiler

[English](README.md) · 简体中文

CA Skill Compiler 用于将 [ConnectionAgent Skill Language（CA-SL）](https://github.com/odango-chan/ca-skills) 的 Skill 源码编译为确定性的制品目录。

项目提供 Go 编译器核心、命令行工具 `skillc`，以及供浏览器集成使用的 Go/WASM 版本。

**当前发布版本：** [v1.0.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0)  
**支持的语言版本：** CA-SL `connectionagent.io/v1alpha4`

## 功能

- 检查 Skill 源码，并报告包含文件路径和源码位置的诊断信息。
- 根据调用方提供的不可变依赖快照和可信元数据解析依赖。
- 校验 Schema、Predicate、Procedure Graph 和 Script 接口。
- 生成确定性的制品目录，并在写入前验证制品。
- Native CLI 与浏览器/WASM 集成共用同一套编译器核心。

Skill 引用的 Python 文件只会被静态解析以提取接口声明，不会被执行。

## 安装

从 [GitHub Releases](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0) 下载对应平台的二进制文件。

| 平台 | 发布文件 |
|---|---|
| Windows x86-64 | [skillc-v1.0.0-windows-amd64.exe](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-windows-amd64.exe) |
| Linux x86-64 | [skillc-v1.0.0-linux-amd64](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-linux-amd64) |
| macOS Apple Silicon | [skillc-v1.0.0-darwin-arm64](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-darwin-arm64) |
| macOS Intel | [skillc-v1.0.0-darwin-amd64](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-darwin-amd64) |

发布页还提供 [Go/WASM 模块](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0.wasm) 和 [SHA-256 校验文件](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/SHA256SUMS)。

## 快速开始

克隆仓库并运行项目内置的 v1alpha4 示例：

```bash
git clone https://github.com/odango-chan/ca-skill-compiler.git
cd ca-skill-compiler

skillc version
skillc check web/parity-success
skillc inspect web/parity-success
skillc build web/parity-success --out ./dist/skillc-output
```

每次构建请使用新的或空的输出目录。这里的示例用于验证编译器行为，不代表真实部署或 5GC 集成。

编译自己的项目时，CLI 接受包含 Skill 源码的项目目录；如果 Skill 声明了依赖，还需要提供依赖快照和可信构建元数据。也可以通过 `--request` 直接传入完整的 BuildRequest JSON。

输入目录结构、命令参数、JSON 输出和退出码见 [CLI 使用说明](docs/cli-v1alpha4.md)。

## 从源码构建

环境要求：Go 1.25 或更高版本。

```bash
go build -o skillc ./cmd/skillc
go test ./...
```

Native CLI 和 WASM 版本使用同一套 Go 编译器核心。

## 语言版本与兼容性

v1.0.0 支持 CA-SL v1alpha4：

- 语言规范：[CA-SL v1alpha4](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md)
- 编译器发布说明：[v1.0.0](docs/release-notes-v1.0.0.md)

当前版本不转换旧版 v0.1/v0.2 Skill 格式。新版作者级 Wait 语义不属于 v1alpha4，因此本版本不支持。

Evidence、Mutation/Reconcile 和 Retry 的部分正向 Build 路径仍采用 fail-closed 处理，直到所需契约和静态检查完整。具体范围见[已知限制](docs/known-limitations-v1alpha4.md)。

## 文档

- [CLI 使用说明](docs/cli-v1alpha4.md)
- [示例与测试样例](examples/README.md)
- [文档索引](docs/README.md)
- [v1.0.0 发布说明](docs/release-notes-v1.0.0.md)

编译器设计契约、集成说明和维护记录统一列在文档索引中。

## 许可证

当前仓库尚未提供 `LICENSE` 文件，也未声明开源许可证。在许可证确定之前，不应默认允许使用、修改或再分发本项目。正式对外发布为开源项目之前，需要先确定并添加许可证。
