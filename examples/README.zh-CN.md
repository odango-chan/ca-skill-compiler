# 示例、测试输入与历史素材

[English](README.md) · [Compiler v2.1.0](../README.zh-CN.md)

**当前唯一语言目标：CA-SL `connectionagent.io/v1alpha5`。**

## 正向测试素材

- [`web/parity-success`](../web/parity-success)：可用于当前 CLI Check / Build 的合成项目，也用于 Native/WASM/Chromium 一致性测试。
- [`web/parity-v1alpha5`](../web/parity-v1alpha5)：冻结入口的 Wait/Ingress 正反例和可信合成请求。
- [Development v1alpha1 完整请求](../docs/design/fixtures/registration-development-request.json)：用于 `build-development` / `verify-development`。它的 Interface Package 快照是测试材料，绝非正式 Owner 发布。

```bash
skillc check web/parity-success
skillc build web/parity-success --out ./dist/example
mkdir -p ./dist
skillc build-development --request docs/design/fixtures/registration-development-request.json --out ./dist/dev --json > ./dist/dev-result.json
skillc verify-development --request docs/design/fixtures/registration-development-request.json --result ./dist/dev-result.json
```

## 历史资料（不代表当前输入）

- `ca-agent-demo-project/`：固定的 ca-agent-demo 迁移素材。
- `ca-skills-v0.2/`：历史 CA-SL v0.2 快照。
- `minimal-project/`：旧 Compiler 的合成示例。
- `script-contract-project/`：旧 Script Contract 原型示例。

历史目录原路径保留，不为使其被现行编译器接受而隐式改写或降级。更多背景见[英文说明](README.md)和[历史任务索引](../docs/workstreams/README.zh-CN.md)。
