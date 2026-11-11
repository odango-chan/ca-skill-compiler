# Script Contract v0 · 唯一新规约（BREAKING）

状态：**设计冻结 / 代码未验收** · 2026-11-11 · Owner: ca-skill-compiler

> **明确决策：不提供旧 Script 格式兼容模式。** 不解析旧 `# /// connectionagent-tool` 里的 `inputSchema` / `outputSchema` 作为 fallback，不实现 `schemaMode: external/embedded`，不支持空 `dict` 类暗示 Schema。旧 5GC Python Script 是**迁移输入**，不是必须原样通过的回归兼容对象。

## 1. 数据生产者决定 Schema

| 数据来源 | Schema 权威位置 | Compiler 处理 |
| --- | --- | --- |
| Tool/MCP 的 Input/Output | `lib/tools/*.yaml` 中 Operation 定义 | 解析和链接，**不再生成** |
| Host / Ingress / Observation 的输入 | 真实 Owner 提供的 `lib/host/`、`lib/observations/` | 明确导入，缺失即静态诊断 |
| Skill 对外公开请求（由 Skill 自己定义的接口） | `skills/<skill>/schemas/` | 直接使用；不是 Script 生成的 Schema |
| Procedure 传递给 Script 的参数 | 每个 `with.<field>.from` 的源端口 Schema | 调用点派生 input projection，**不作为第二份权威文件** |
| Script **自己计算出的新结果** | Script 中静态声明的 Output Type | 自动生成只读 Output Schema |
| 透传一个已有结果 | 上游实际 Producer | Procedure 直接转发来源，不造重复类型 |

**导入不是认证**：外部 Tool Catalog 说明接口存在，不保证 Operation 已实现/可调用/有权限。新 Compiler 不负责 Runtime READY。

## 2. 新 Script 唯一写法

~~~python
from typing import Annotated, Literal, TypedDict

class SelectionOutput(TypedDict):
    dnn: Annotated[str, {"minLength": 1, "pattern": r"^[a-z0-9.-]+$"}]
    accepted: Literal[True]

def main(input: dict) -> SelectionOutput:
    return {"dnn": input["value"].strip(), "accepted": True}
~~~

- 一个 Skill-owned Script 必须位于 `skills/<skill>/scripts/*.py`，入口固定为 `main`；唯一命名身份是**Skill + Script 相对路径**，不靠 metadata 的 `name` 另造一套身份。
- `main(input: dict)` 中的 `dict` **仅是调用点传入的字典容器**。它的每个入参由 Procedure `with:` 绑定的 Producer 严格决定；Compiler 必须解析绑定，不能把 `dict` 当作接受任意值的 Schema。
- Script 新生成的 Output 类型是**非空** `TypedDict`。所有递归嵌套对象必须有明确定义；禁止 Output 为 `dict`、`dict[str, T]`、`Any`、`object`、空类、无类型注解和不受支持的 Union。
- Output 白名单：`str/int/float/bool`、同文件有限的 `TypedDict`、`list[T]`、`Literal[constants]`、`NotRequired[T]`、受限 `Annotated[T, {...}]`；超出白名单直接 `SCRIPT_TYPE_UNSUPPORTED`。
- `Annotated` 仅接受字面量约束（例如字符串 `minLength/maxLength/pattern`，数值 `minimum/maximum`，列表 `minItems/maxItems/uniqueItems`），必须按基础类型匹配。未知约束、未知类型、非法范围、动态表达式都应报错，不能降级。
- `TypedDict` 生成 JSON Schema Draft 2020-12，`additionalProperties: false`、`required` / `NotRequired` 必须正确；`Literal` 生成 `const` 或 `enum`；不推断未声明的业务规则。
- **不运行 Python，也不自动从函数体推断接口**。不必引入 Python/Pydantic 运行环境或 subprocess。编译只保证接口声明和链接，不保证函数体真的返回正确值，也不证明脚本确定性；Runtime 对实际 I/O 校验与沙箱负责。
- 复杂跨字段约束（例如 `allOf/if/then`）若超出 Type 白名单，应**调整新 Script 的数据契约或提出单独的语言扩展**，不能复活旧 Script external 兼容路径来掩盖缺口。

## 3. Source 与 Output

Script `skills/<skill>/scripts/common/normalize.py` → **仅产生**：

`skills/<skill>/schemas/generated/common/normalize.output.schema.json`

Output path 相对于 BuildResult 的 `outputFiles`，实际位于 `output/skills/...`；不得回写作者源码，也不能生成一份重复权威的 `normalize.input.schema.json`。若本次调用需要输入类型，只在诊断/Library Inspector 中显示从 `with:` 派生的投影。

Script 结果中的原样上游数据应优先直接在 Procedure 链接而不返回无类型 `dict`；任何新算出的对象都必须满足完整类型白名单。Compiler 对重复/碰撞输出路径报错。

## 4. 旧 5GC 源码的角色（不是兼容承诺）

`examples/ca-agent-demo-project/` 的两套真实 Skill 是固定的**迁移前基线**：79 个 Skill 源文件 + 一份正式 ToolCatalog 与 upstream 固定 Git blob 一致，MAS 设计库另行导入。不要为拿绿色测试而偷偷篡改原始 Git 快照。

实际新规约迁移在**单独的新版 Project 目录**进行（推荐 `examples/ca-agent-demo-v0/`，不要覆盖 pinned reference）；允许移除旧头部与旧重复 Schema、改写 Python Output TypedDict 和 Procedure 参数引用。**绝无要求迁移前源码原样编译 PASS**。如果外部 Host/Observation 的权威 Schema 缺失，则迁移版对应严格类型检查仍须 FAIL/BLOCKED，不可造假。

## 5. 确定性的真正含义

- Script “确定性”是输入固定时结果一致，或失败按确定的条件表现；静态 `TypedDict` 只能描述 Output 结构，**不能单独保证**无随机、无网络、无时间或运行时副作用。
- Compiler 必须**不执行 Script**；Runtime/Harness 控制权限、隔离与执行验证。
- 接口不完整必须有明确诊断；不自动给未声明字段加通配型 `object`。
- Studio 只编辑 `skills/`，只读显示 `lib/` 和 `output/`。静态类型生成与外部数据链接唯一实现位于 Compiler Core。

## 6. 必测项（详见 acceptance matrix）

- ST-01 正向：导入 Tool Catalog、`with:` 引用 Tool 输出 → Script 输入投影 → 生成新 Output TypedDict Schema。
- ST-02 负向：未知 Tool/字段/外部 Host schema、泛化 Tool 输出对象，必须诊断。
- ST-03 负向：Script 输出 `dict/Any/object`、空类/非静态约束，必须拒绝。
- ST-04 新 Script 不写旧 metadata；旧 Script metadata 不得作为 fallback。
- ST-05 Native/WASM 对同一输入集合产生同样的诊断和输出字节。
- ST-06 固定的 5GC 源快照在迁移前允许得到明确的“待迁移”诊断，不宣称 PASS。
- ST-07 编译不执行 Python/Tool/MCP、不联网。

本文件是**最终 v0 选择**，不再需要兼容选择开关。
