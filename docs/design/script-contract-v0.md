# Script Contract v0 · Python 脚本与 Schema 生成规约

- 状态：**ACCEPTED DESIGN / NOT IMPLEMENTED**（2026-11-11）；[实施 Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1)
- Owner：`ca-skill-compiler`（静态解析/生成）；Script 作者拥有接口声明，Studio 只编辑源码、显示生成结果
- 兼容基线：[ca-agent-demo 双 Skill 原样快照](../../examples/ca-agent-demo-project/README.md)，不修改这些真实样例来使 Compiler 过关
- **本规约是 v0 唯一 Script-to-Schema 机制**。Python `TypedDict` / `Annotated` 自动推断可以以后讨论，当前不要另实现第二个推断器。

## 1. 现状与原则

当前 5 个 `establish-pdu-session/scripts/*.py` 脚本均已含 `# /// connectionagent-tool` 头部，声明 `name`、`inputSchema`、`outputSchema`、`effect: local_compute`、`sandbox.network: denied`、`timeout`。但诸如 `class EstablishmentSelectionInput(dict): pass` 的空类完全没有字段级规范；现有独立 `schemas/*.schema.yaml` 还包含 `required`、`const`、`enum`、`pattern`、`allOf / if / then` 等复杂约束。仅凭 Python 函数体**不能等价推断出它们**。不要使用动态导入、函数体数据流推断或执行脚本来“猜 Schema”。

**唯一权威规则：**
- 一份 Script interface 只能有一种有效权威来源：现有独立 Schema（`external`）或 Script 头内声明（`embedded`）。
- `embedded` 时 Compiler 把声明转换为生成文件，并在构建结果中链接；生成物不允许在 Studio 作为作者 Source 编辑。
- `external` 时 Compiler 引用/校验并原样携带现有独立 Schema；**不生成第二个看似权威的 Schema**。
- Tool Operation 的 input/output Schema 依旧由 imported `lib/` 的 Tool Contract Owner 决定；Script 输出能被用作 Tool 入参不意味着 Script 可以覆盖 Tool Schema。
- `check/build` 完全静态且无状态；不导入 Python，不执行 Tool，不能判定脚本运行正确或 Runtime READY。

## 2. 共用脚本格式

一个 Script 是 Skill 作者源文件 `skills/<skill>/scripts/<relative>.py`。其头部使用现有格式：

~~~python
# /// connectionagent-tool
# name: normalize-selection
# entrypoint: main
# inputSchema: NormalizeSelectionInput
# outputSchema: NormalizeSelectionOutput
# schemaMode: embedded
# effect: local_compute
# sandbox:
#   network: denied
# timeout: 1s
# interface:
#   input:
#     type: object
#     additionalProperties: false
#     required: [dnn, sst]
#     properties:
#       dnn: {type: string, minLength: 1, pattern: "^[a-z0-9.-]+$"}
#       sst: {type: integer, minimum: 0, maximum: 255}
#   output:
#     type: object
#     additionalProperties: false
#     required: [dnn, accepted]
#     properties:
#       dnn: {type: string, minLength: 1}
#       accepted: {type: boolean}
# ///

def main(input: dict) -> dict:
    return {"dnn": input["dnn"], "accepted": True}
~~~

### 2.1 元数据解析要求

- 仅脚本最开头的**一个** `# /// connectionagent-tool` … `# ///` 区块有效；不在函数体、三引号、任意注释中搜索片段。允许前导 Python shebang/编码声明与空行，但不得存在两个元数据区块。
- 去掉每行前面的 `#` 和**至多一个**后随空格后，剩余文本视为 YAML；保留缩进，使用同一受限、安全的 YAML 解析器，禁止重复键、YAML alias 递归/膨胀、无限层级。
- 字段约束：`name` 非空且在所属 Skill 中唯一；`entrypoint` 默认为 `main`，v0 只支持 `main`；`inputSchema` / `outputSchema` 是非空接口名称，不等价于文件路径；`schemaMode` 可省略（默认 `external`）；`effect` 为 `local_compute`；`sandbox.network` 必须是 `denied`。`timeout` 是正数、带单位的有效时长。
- Compiler 不凭脚本类型注解推断网络能力、时间限制或副作用；头部安全声明不代表已执行沙箱防护，执行阶段由 Runtime/Harness 独立负责。
- v0 Script 约定：`main(input)` 接收一个 JSON 可表示的对象，成功返回一个 JSON 可表示的对象；失败由执行宿主管理。Compiler 可以检查声明中的函数名称/签名，但不执行/验证 Python 函数实际行为。避免脚本副作用、全局状态、随机数、读取环境、网络或文件系统作为业务数据来源。
- 原有头部未显式 `entrypoint` 或 `schemaMode` 的 5 个脚本继续合法，按 `main` + `external` 处理，不应报错要求立即迁移。

## 3. schemaMode: external（现有兼容模式）

~~~python
# /// connectionagent-tool
# name: select-establishment-configuration
# inputSchema: EstablishmentSelectionInput
# outputSchema: EstablishmentSelection
# effect: local_compute
# sandbox:
#   network: denied
# timeout: 1s
# ///
~~~

- `inputSchema/outputSchema` 是现有的**符号接口名**，例如 `EstablishmentSelectionInput`；不可错误假设其必然对应 `schemas/<symbol>.schema.yaml`。
- 为减少猜测，允许新增可选字段 `inputSchemaPath` / `outputSchemaPath`（仅 external 模式），例如 `schemas/selection-input.schema.yaml`。这些路径必须是当前 Skill 内规范化相对路径，禁止目录逃逸。
- 如果未声明 Path，按当前 Skill 的 `schemas/*.{yaml,yml,json}` 中的**唯一** `title` / 明确 ID 映射查找；找不到报 `SCRIPT_SCHEMA_NOT_FOUND`，多个候选报 `SCRIPT_SCHEMA_AMBIGUOUS`，不能选择碰巧第一个文件。
- Compiler 解析 JSON Schema 并校验最低结构、重复/冲突引用；已有的 `$schema: 2020-12`、`allOf/if/then` 不应因迁移而丢失。
- `external` 不接受 `interface.input/output` 同时声明，避免双源矛盾。不能把类型为空的 `dict` class 当作另一个权威 Schema。
- 适用于全部冻结的 5 个真实 Python 脚本；本次不修改真实样例。

## 4. schemaMode: embedded（新作者默认模式）

- `interface.input` 和 `interface.output` 均为内嵌的 JSON Schema object（在注释 YAML 中表达），必须同时存在。
- 允许常用 JSON Schema Draft 2020-12 关键词：`type`、`properties`、`required`、`additionalProperties`、`items`、`enum`、`const`、`minimum`、`maximum`、`minLength`、`pattern`、`$defs` 和本地 `$ref` 等。**不得向 schema 偷偷加约束**；没有声明 `required` 就不自动设必填，没有声明 `additionalProperties` 就不默认 false。
- 一旦包含无法解析/不受支持但会影响语义的 Schema 关键字，报明确 `SCRIPT_SCHEMA_UNSUPPORTED`；不能静默删除。不可远程下载 `$ref`，仅支持显式限定的文档内引用。
- Compiler 生成符合 JSON Schema 2020-12 的 JSON 文件。若用户没有指定 `$schema`，补标准 Draft 2020-12 声明；若 `title` 缺失，从 `inputSchema` / `outputSchema` 补充；已声明但与符号名矛盾时给出冲突诊断。
- **稳定输出路径**：从 `scripts/<relative>.py` 生成
  - `skills/<skill>/schemas/generated/<relative-without-ext>.input.schema.json`
  - `skills/<skill>/schemas/generated/<relative-without-ext>.output.schema.json`
  - 嵌套 Script `scripts/common/foo.py` 则生成 `schemas/generated/common/foo.*.schema.json`，避免文件名碰撞。
- 上述路径是 **outputFiles 的相对路径**，Compiler 不向 `skills/` 作者原目录回写。与作者文件发生路径冲突必须报错；不能覆盖。
- Script 节点根据 Script 路径解析输入/输出接口，不按可重复的全局 `title` 随机选择。若多个 Script 共享同一符号输出名，各 Script 的接口身份仍由 `skill + script path + direction` 唯一确定。
- 生成内容必须排序/可重复，不添加时间戳、Release、Digest 或环境信息。使用标准 JSON 序列化，确保同一输入在 Native/WASM 字节一致。

## 5. 运行时接口约束与实际执行责任

脚本的接口负责明确：参数名/形状、输出形状、错误处理边界，不负责证明执行结果真实。最小运行方式概念上是 `main(input: object) -> object`。现有 Python 标注 `main(input: EstablishmentSelectionInput) -> EstablishmentSelection` 继续合法——它们不是唯一 Schema 权威。

编译器静态校验：
1. Metadata 合法与 Schema 解析；
2. 当前 Script 所在 Skill 的引用准确，且声明的 Input/Output 可解析；
3. 参数绑定的字段级静态兼容（仅当有充分字段信息时）。未知/动态表达式不能伪称已验证；
4. 调用外部 Tool 时，仍以 `lib/tools` 的 Tool Schema 作目标校验；
5. 输出 deterministic 文件/诊断。

**运行宿主**负责真实 Python 环境、依赖安装、资源限制、网络拒绝、运行前/后的 Schema 检查、超时、异常处理、Evidence/Outcome；这些不是本编译器的新 Runtime 功能。

## 6. 迁移策略

- **M0**：冻结规约；Compiler 优先支持 `external`（5 个历史脚本原样 PASS 的静态解析与 Schema 引用）以及最小 `embedded` fixture。
- **M1**：把新写 Script 推荐为 `embedded`；新增小型样本证明一个文件声明即可生成 Schema。
- **M2**：在 Skill Owner 处**另外开 PR** 逐个迁移真实脚本，并以已有独立 Schema 的结构/约束作等价对照；复杂 400 行 `allOf` 的 Schema 不要求立即变成冗长的脚本头部。如果写进源码会使可维护性下降，可以长期保持 `external`。
- **M3（未来）**：如确有必要，评估受限 `TypedDict/Literal/Annotated` 到 Schema 的静态生成；必须有独立规范与 parser/Native/WASM 一致性测试，不能在 v0 临时加第三套权威。

**核心结论：自动生成的是构建产物，不是凭空推断约束。** embedded 由脚本头部提供 Schema 声明，Compiler 生成文件；external 由现有 Schema 提供权威，Compiler 负责链接。这两者不应该相互覆盖。

## 7. 验收 Case

| Case | Input | Expected |
| --- | --- | --- |
| PY-01 | 现有 5 个 Script，不修改 | `external` 解析正确；保留现有独立 Schema 约束 |
| PY-02 | embedded 完整 input/output | 自动生成两个稳定路径的 JSON Schema，保留 required/enum/pattern 等约束 |
| PY-03 | 空 `dict` 类，无 Schema 来源 | 不推断字段；返回明确缺少 Schema 错误 |
| PY-04 | 缺 external Schema / ambiguous title | 报来源路径与原因，不默选 |
| PY-05 | 两份接口来源冲突 | fail closed |
| PY-06 | embedded 脚本嵌套路径与重复文件名 | 输出路径不冲突 |
| PY-07 | 包含 unsupported Schema / 远程 $ref | 明确诊断；不降级或联网 |
| PY-08 | 修改 schemaMode 的 Script | Native/WASM 相同诊断和生成文件字节 |
| PY-09 | Python 文件包含恶意导入/执行副作用 | Compiler 静态分析，不执行 Python / 不访问网络 |
| PY-10 | Script 输出绑定到 Tool Input | Tool Schema 仍是 Library 权威；不复制可编辑契约 |

所有 Case 初始均为 **NOT_RUN**。实现后必须维护负例与准确的诊断定位；不允许只检查生成文件存在。

## 8. Studio 显示

- `skills/<name>/scripts/foo.py` 仍是可编辑作者文件。
- Inspector 展示 Script 的 `Input / Output` 声明与来源：`External Schema` 或 `Embedded → Generated`；脚本头部提供导航，不需要额外的 Schema Designer 副本。
- `output/skills/<name>/schemas/generated/` 展示只读生成结果；点击可以预览、复制和对照，却不可编辑回 Source。
- `lib/` 的 Tool Schema 是外部只读契约，不由 Script 编辑器改写。
- 编译诊断指向脚本元数据行或真实外部 Schema 源文件，不让作者去修改生成物。

参考：[Studio 集成 ABI](../studio-integration.md)，生成 Schema 仍由 Compiler 单独负责，不在 Studio 实现一套推断逻辑。
