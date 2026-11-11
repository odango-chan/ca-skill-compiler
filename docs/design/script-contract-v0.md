# Script Contract v0：严格类型接口与自动 Schema

- **决策**：2026-11-11 接受「Type-first / Strict」；实现 **NOT_STARTED**，所有 ST 测试 **NOT_RUN**
- **权威**：此文件是 Compiler v0 唯一 Script-to-Schema 规约；[Skill Compiler 总设计](skill-compiler-v0.md) / [本地实施 Goal](../workstreams/local-implementation-goal.md)
- **作用域**：Python Skill-owned 本地 Script 的接口；不修改现有两套 5GC Skill 真实快照；不负责运行时执行/Admission。
- **被撤回的方案**：Script 头部内嵌大段 YAML/JSON Schema（`schemaMode: embedded`）不再是 v0 支持的模式；弱模型不得按旧草案实现它。

## 1. 核心原则

**新 Script 接口必须完整声明字段结构，不允许以笼统 `dict`、空的 `class X(dict): pass`、`Any` 或 `object` 代替输入/输出。** Python 函数体里用普通 `dict` 构造/处理数据仍然合法，限制仅针对输入/输出及其递归字段的**类型契约**。

接口权威采用互斥的两种模式：
- **`typed`（新脚本默认/推荐）**：Python 标准库 `TypedDict` + `main(input: In) -> Out` + 可选 `Annotated` 约束；Compiler 从静态类型 AST 生成 Output Schema。没有额外接口 YAML、无需为同一个类型再手写 Schema。
- **`external`（仅已有旧脚本和复杂 Schema）**：保留现有 Script metadata 声明的 `inputSchema` / `outputSchema`，以独立 Schema 文件为权威；不得因空 `dict` class 而推断结构。可长期支持复杂 `allOf/if/then` 等不适合表示为 Python 类型的契约。**同一端口不能同时由 typed 和 external 两套 Schema 定义。**

缺少支持的类型、未声明的字段、歧义、未知约束必须有明确的 error/diagnostic，不准降级为 `object` 或悄悄丢掉约束。静态编译成功不代表脚本函数体或 Runtime 执行正确。

## 2. 最小而严格的新脚本

~~~python
from typing import Annotated, Literal, NotRequired, TypedDict

class SNssai(TypedDict):
    sst: Annotated[int, {"minimum": 0, "maximum": 255}]
    sd: NotRequired[Annotated[int, {"minimum": 0, "maximum": 16777214}]]

class EstablishmentSelectionInput(TypedDict):
    dnn: Annotated[str, {"minLength": 1, "pattern": r"^[a-z0-9][a-z0-9.-]*$"}]
    sNssai: SNssai
    requestedPduSessionType: Literal["IPV4"]
    requestedSscMode: Literal[1]

class EstablishmentSelection(TypedDict):
    dnn: Annotated[str, {"minLength": 1}]
    sNssai: SNssai
    pduSessionType: Literal["IPV4"]
    sscMode: Literal[1]

def main(input: EstablishmentSelectionInput) -> EstablishmentSelection:
    return {
        "dnn": input["dnn"],
        "sNssai": input["sNssai"],
        "pduSessionType": "IPV4",
        "sscMode": 1,
    }
~~~

**为何是这些语法：**
- `TypedDict` 明确列出每一个字段；嵌套对象必须引用另一个完整 `TypedDict`，如 `SNssai`。
- `Literal` 明确常量或枚举，而不是宽泛的 `str` / `int`。
- `Annotated[T, {...}]` 中的**静态 Python dict 字面量只是类型元数据**，不是允许使用宽泛 `dict` 类型；其键是受限的 JSON Schema 属性约束，不依赖 Pydantic、特殊 Python 包或运行时反射。
- `NotRequired[T]` 表示键可缺省；字段值本身可为 `None` 与键是否存在是不同概念。
- v0 强制 `TypedDict` 转换出的 JSON Schema object 为**封闭对象**（`additionalProperties: false`）；这是本产品比普通 Python TypedDict 更严格的 authoring 契约。作者必须接受这一规则并由 Runtime 验证额外字段，不能把静态注解误称为 Python 运行时验证。
- 具体领域的 `pattern`、`minimum`、`enum` 等约束需由作者明确声明。编译器不从业务 Python 代码猜测它们。

## 3. v0 类型语法白名单（Strict）

| Python type AST | 输出 Schema | v0 条件 |
| --- | --- | --- |
| `str`、`int`、`float`、`bool` | string / integer / number / boolean | 完整定义 |
| `class X(TypedDict)` | object/properties/required，`additionalProperties: false` | 字段均明确类型，不能是空类 |
| `list[T]` | array/items | `T` 递归满足白名单 |
| `Literal[v]` / `Literal[v1, v2]` | const/enum | 只接收 JSON 可表示的字符串、整数、布尔等字面量；不能混用类型 |
| `NotRequired[T]` | 该字段不在 required | 仅可用于 TypedDict 字段 |
| `Annotated[T, {"minimum": ..., ...}]` | 保留静态声明的 Schema 约束 | 仅允许白名单关键词并做类型校验 |
| 同文件 `TypedDict` 的引用 | 本地 `$defs/$ref` | 显式有定义，检测循环与悬空引用 |

**v0 拒绝**（诊断 `SCRIPT_TYPE_UNSUPPORTED` 或更具体代码）：`dict`、`dict[str, T]`、`Mapping`、`Any`、`object`、裸 `list`、`tuple`、`set`、`TypedDict(total=False)`、未限定泛型、运行时变量类型表达式、动态反射、未解析的外部类型、递归对象、任意 Union 或尚不支持的 `T | None`、类型别名与不被支持的继承。需要动态键对象/联合/复杂条件时，**使用 external Schema 模式而不是把强约束降级**。

字段只能来自完整的静态声明；不能根据 `input["foo"]`、`input.get("bar")`、`return {...}`、分支逻辑自动推断 required、可空、类型或者跨字段约束。

### 3.1 静态约束白名单

v0 `Annotated` 元数据只允许一个**常量 dict 字面量**，键限定：
- 字符串：`minLength`、`maxLength`、`pattern`
- 数值：`minimum`、`maximum`、`exclusiveMinimum`、`exclusiveMaximum`、`multipleOf`
- 数组：`minItems`、`maxItems`、`uniqueItems`

如果出现对基础类型不合法的键、未知键、负长度、范围矛盾、无效 pattern、非字面量或多份相互矛盾的元数据：编译 FAIL。`Annotated` 不负责 `allOf` / `if` / `then` 等跨字段约束，后者必须留在 external Schema。

**类型完整**与**业务约束完整**是两层要求。Compiler 能精确转换被声明的限制，但无法自行证明作者没有忘记重要业务限制。迁移旧 Schema 时必须做结构、required、约束与条件分支的等价对照，若表达力不足则留在 external。

### 3.2 最小函数签名

- 唯一入口 `main(input: InputType) -> OutputType`（输入参数恰好一个，类型明确）；两个类型均是当前 Script 中可解析、**非空** TypedDict。
- 输入/输出的每个嵌套对象都必须递归满足白名单。
- 支持普通 Python 文件中的其他函数、逻辑和内部 dict 使用；Compiler **不执行**脚本，不运行其顶层 import，不对函数体假装完成业务正确性验证。
- Compiler 需使用独立、有限的 Python 类型声明解析器（或具有明确 JS/WASM 兼容性的纯 Go 解析依赖），不能用全文正则误判函数体局部变量/注释；解析失败保守报错。不能依赖 Python subprocess、cgo 或 Node 才能生成 Schema。

## 4. Script 元信息：尽量省掉额外配置

新 `typed` Script 可**不写**旧 `# /// connectionagent-tool` 头；相对路径 `scripts/x.py` 即接口身份，不需要重写 `inputSchema/outputSchema` 两个名字。Compiler 可从唯一 `main` 注解获取类型定义。

旧 Runtime 可能仍要求该头部及 sandbox/timeout 约束，因此“Compiler 生成 Schema”与“Host 能执行”必须区分。为了兼容现有部署，允许旧头部仅保留必要的运行声明（`effect: local_compute`、`sandbox.network: denied`、`timeout: 1s` 等）；**不凭省略头部断言旧 Runtime 已接受**。宿主需要配置默认超时和拒绝网络，不由 Compiler 提供运行沙箱。

v0 按以下方式选择接口模式：
- 存在有效、可静态解析的 `main` 注解及完整 TypedDict 且**没有**外部 `inputSchema/outputSchema` 字段 → `typed`。
- 带已有外部 `inputSchema/outputSchema` 元数据 → `external`；以外部 Schema 文件为权威，允许旧的空 dict 类但不得把空类用来生成 Schema。
- 同时出现 `schemaMode: typed` 和外部 Schema 字段、或 `schemaMode: external` 和 typed 生成要求 → **报冲突，不悄悄选择**。
- 旧 `schemaMode: embedded` 一律报不支持；正式约定不再允许该模式。

## 5. 编译结果与路径

对新 `typed` Script `skills/<skill>/scripts/common/normalize.py`，输出：
- `skills/<skill>/schemas/generated/common/normalize.input.schema.json`
- `skills/<skill>/schemas/generated/common/normalize.output.schema.json`

输出 Schema 为 JSON Schema Draft 2020-12，包含稳定的 `title`、`$defs`（适用时）以及完整约束；文件按稳定顺序、无时间戳/Release/Digest；生成到 `outputFiles`，**不能写回 Source**。名称身份由 Script 的**路径 + input/output 方向**确定，遇到目标路径与作者文件冲突必须报错。

external 模式通过旧 `inputSchema/outputSchema` 符号名，在同 Skill 的 Schema 内按显式映射或唯一 `title` 定位；找不到/多个同名/不支持的 Schema 关键字须报错，不允许随机择一。旧 JSON Schema 2020-12 的复杂约束必须完整保留，不必“为了自动生成”而减少约束。

Tool 的 Input/Output/Reconcile Schema 仍来自只读 `lib/`，Skill Script 的生成类型只能用于连接与兼容性检查，不能覆盖 Tool 权威接口。

## 6. 与 Studio / Runtime 的界限

Studio 编辑 Script，能显示“Typed schema generated”或“External schema”来源；库契约和输出文件只读，不实现第二套类型推断。Native/Go WASM 共用相同静态 parser、诊断和生成字节。

Runtime/Host 独立负责脚本执行环境、网络拒绝、时间和资源限制、实际输入/输出 Schema 校验、权限和 Admission。`Compile PASS ≠ Python 行为正确 ≠ Runtime READY`。

## 7. 实施与验收（测试必须真实运行）

1. **ST-01**：新脚本用完整 Input/Output TypedDict + 嵌套类型 + Annotated → 输出两份 Schema，保留每个必填/可选键、范围、Literal、pattern。
2. **ST-02**：`dict`、`dict[str, Any]`、空 TypedDict、`Any`、`object` 出现在任一输入/输出递归类型 → 明确拒绝；不得悄悄推断。
3. **ST-03**：未注解 `main`、未知类、额外函数、多个入口、未支持 Union → 精确诊断。
4. **ST-04**：旧 5 个 Script 未修改，external Schema 无损链接；不能从空 dict class 推断。
5. **ST-05**：新 typed 与 external 混用同一个接口 → 冲突拒绝。
6. **ST-06**：不合法 `Annotated` 键、负长度、矛盾 minimum/maximum、远程引用 → 错误，不削弱 Schema。
7. **ST-07**：重复文件名/嵌套 Script 输出路径无冲突；与作者 Schema 冲突时报错。
8. **ST-08**：Native 和 WASM 相同 input file map → 诊断、Schema 字节一致。
9. **ST-09**：Python 顶层包含可能引发副作用的代码 → 不能运行 Python（含 import、表达式与函数体）。
10. **ST-10**：已有复杂 external Schema 含 `allOf / if / then` → 原样保留，不降级成“简化” TypedDict。
11. **ST-11**：生成的 Schema 与旧 Schema 迁移对照；若不能等价，保持 external / 禁止假 PASS。

当前只冻结此规约，不宣称已实现 ST-01..11。实现前请确认本地 [Goal](../workstreams/local-implementation-goal.md) 使用该文件作为唯一权威。
