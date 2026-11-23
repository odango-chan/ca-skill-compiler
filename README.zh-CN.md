# CA Skill Compiler

[English](README.md) · [中文文档目录](docs/README.zh-CN.md) · [v2.1.0 正式发布](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0)

CA Skill Compiler 是 CA-SL 的确定性、无状态编译器。Native `skillc`、Go/WASM Worker 共用同一套 Go Core；检查 Skill 源、链接调用方提供的不可变接口快照、构建并验证 Artifact。Compiler **不运行** Tool、Python、网络协议或实际 Procedure。

**当前产品：v2.1.0（已发布，2026-11-23）**。**唯一语言目标：`connectionagent.io/v1alpha5`**。不支持 v1alpha4、v0.1 等旧语言模式，也不提供兼容回退。

## 下载与运行

从 [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0) 下载对应平台的程序和 `SHA256SUMS`：Windows x86-64、Linux x86-64、macOS Intel 或 Apple Silicon。将程序重命名为 `skillc`（Windows 为 `skillc.exe`），并加入 PATH。

也可以使用 Go 1.25+ 从源码构建：

```bash
CGO_ENABLED=0 GOWORK=off go build -o skillc ./cmd/skillc
./skillc version
./skillc check web/parity-success
./skillc inspect web/parity-success
./skillc build web/parity-success --out ./dist/example-artifact
```

输出目录须不存在或为空。上述示例是测试快照，不是真实 Owner 发布物或产品验收结论。

## 两类编译入口

| 接口 | 命令 | 边界 |
|---|---|---|
| 冻结 v1alpha5 入口 | `check`、`build`、`inspect` | 依赖可信 Build 上下文，生成冻结 v1alpha5 Artifact |
| Development v1alpha1 | `build-development`、`verify-development` | 用源码、精确 Interface Package 快照和 Dependency Lock 生成 `NOT_ADMITTED` Artifact |

Development 正例（仓内可复验的合成测试请求）：

```bash
./skillc build-development --request docs/design/fixtures/registration-development-request.json --out ./dist/dev-artifact --json > ./dist/dev-result.json
./skillc verify-development --request docs/design/fixtures/registration-development-request.json --result ./dist/dev-result.json
```

Development 会输出静态 `capabilityRequirements[]`、`host-admission-requirements.json` 和 `runtime-execution-requirements.json`。**`NOT_ADMITTED` 是设计规定的成功产物状态，不等于获得执行授权。生产 Host 必须拒绝 DEVELOPMENT。**

## 文档入口

- [中文 CLI 使用手册](docs/cli.zh-CN.md) · [English CLI](docs/cli.md)
- [中文文档索引](docs/README.zh-CN.md) · [English index](docs/README.md)
- [冻结 v1alpha5 机器契约](contracts/v1alpha5/README.md) · [Development v1alpha1 机器契约](contracts/development/v1alpha1/README.md)
- [工具链架构](docs/design/ca-toolchain-architecture-vnext.zh-CN.md) · [Archify 架构图](docs/architecture/archify/README.md)
- [Compiler 独立交付边界](docs/COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md)
- [v2.1.0 发布记录](docs/releases/ca-skill-compiler-v2.1.0.md) · [发布验收 Actions](https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980)

CA-SL v1alpha5 / IngressCatalog v1alpha1 的规范所有权在 [ca-skills](https://github.com/odango-chan/ca-skills)。Compiler 不负责 Owner/Provider 签发、部署 Grant、Host 准入或真实 5GC/MAS 验收；真实 MAS 源码离线测试另见 [#53](https://github.com/odango-chan/ca-skill-compiler/issues/53)。

旧 v1alpha5 Build/Verify 的 Artifact 字节及内嵌 `generator.version=v2.0.2` 保持冻结，该标识是**既有 Artifact 契约身份**，并非当前 CLI 版本；新的 Development Manifest 使用 v2.1.0。

## 验证和历史资料

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o /tmp/skillc.wasm ./cmd/skillc-wasm
```

CI 还执行真实 Chromium Worker 一致性验收。旧版本规范、迁移工作包和测试资料保留在[历史工作索引](docs/workstreams/README.zh-CN.md)，**不代表当前仍支持旧语法**。
