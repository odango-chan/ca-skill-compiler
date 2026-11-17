# 示例与测试样例

当前 v1alpha4 正向构建示例为 [`web/parity-success`](../web/parity-success)，CLI 和浏览器一致性测试均使用该项目。请在仓库根目录运行：

```bash
skillc check web/parity-success
skillc inspect web/parity-success
skillc build web/parity-success --out ./dist/skillc-output
```

`examples/` 下的目录是早期开发和迁移阶段留下的参考材料，并非当前 v1alpha4 的正向构建示例。

| 目录 | 内容 |
|---|---|
| `ca-agent-demo-project/` | 从 ca-agent-demo 固定版本保存的源码快照，用于迁移分析 |
| `ca-skills-v0.2/` | 历史 CA-SL v0.2 Skill 源码快照 |
| `minimal-project/` | 旧版 Compiler 使用的合成测试样例 |
| `script-contract-project/` | 旧版 Script Contract 合成测试样例 |

这些历史材料仅供参考，不应为了通过 v1alpha4 编译而修改。验证当前 Compiler 时，请使用 `web/parity-success`。

更多编译器符合性测试见 [tests/conformance/v1alpha4](../tests/conformance/v1alpha4)，验收结果见 [Compiler 验收记录](../docs/COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)。
