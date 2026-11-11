# Procedure / Binding / Library Link Contract v0

状态：**设计冻结 / 编译器尚待实现**，与 [Script Contract v0](script-contract-v0.md) 同级权威。本文件用于消除弱模型对 YAML 递归扫描和 `from:` 的猜测。

## 1. v0 最小静态语法（不为了历史兼容增加分支）

- 根为 `procedures:<procedure-name>: {inputs?, start, nodes}`；`start` 必须指向当前 procedure 的一个 node。
- 一个 node 的主动作只允许其一：`call`（Tool Operation ID）、`script`（本 Skill 相对脚本路径）、`procedure`（本地 procedure 名或显式 `{skill: name, procedure: name}`）、`verify`（Predicate ID）、`reconcile`（本 procedure 的先前目标 node ID）、`terminal`。允许在同一 node 的 `preconditions.steps[]` 与 `postconditions.steps[]` 内使用受限的 `script/verify/call` 步骤；这些 step 是独立命名作用域。
- `with:` 是实参绑定：每项使用 `{from: <source-ref>}` 或 `{value: <JSON-literal>}`；两种不能同时出现。嵌套对象/数组只在拥有明确目标 Schema 时递归检查，不能悄悄当作任意 object。
- `'on': {outcome: targetNode}` 的每个目标必须是当前 procedure 的 node 名；只做静态目标和显式 outcome 检查，不执行状态机。
- `predicate` 声明来自当前 Skill 的 `assurance/predicates.yaml`（v0 验证 ID/引用存在与定义基本合法；不做 CEL evaluator 执行）。
- 所有未知但可能影响执行语义的动作字段，必须报 `PROCEDURE_UNSUPPORTED`；不要忽略以得到 Build.success。
- **禁止**用遍历所有 YAML keys 的方式把 nested `procedure: run` 误当作另一个本地 node。按 AST 节点类型/位置解析，每个动作只访问一次。

## 2. 引用名字与来源规则

表中标识是**解析规范**，不是调用时刻的数据。需要用词法作用域与 Schema 来验证。

| 引用 | 从哪里获得 Schema | 查不到时 |
| --- | --- | --- |
| `input.<field>[.nested]` | 本 Procedure `inputs` 中同名参数；参数的来源再追溯 `context.*` 或调用方 binding | `BINDING_UNRESOLVED` |
| `context.request.<field>` | 当前 Skill 合同显式声明的 Request Schema（可以来自真实 imported lib，也可由 Skill 作者声明） | `EXTERNAL_SCHEMA_MISSING` / `SOURCE_FIELD_UNKNOWN` |
| `context.subject.<field>`、`context.facts.<field>`、`context.observations.<field>` | 具名 `lib/host/` / `lib/observations/` 中的 Owner Schema，要求**显式声明引用路径** | `EXTERNAL_SCHEMA_MISSING` |
| `<nodeId>.output[.field]` | 同 Procedure 的 nodeId 指向的 Operation/Script/Procedure Output Schema | `BINDING_UNRESOLVED` / `SOURCE_FIELD_UNKNOWN` |
| `preconditions.<stepId>.output[.field]` | **当前 node** 的 precondition named step，按其 call/script output schema | `STEP_SCOPE_UNKNOWN` |
| `postconditions.<stepId>.output[.field]` | 同上，但仅在允许的声明作用域里 | `STEP_SCOPE_UNKNOWN` |
| `<nodeId>.preconditions.<stepId>.output[.field]` | 指定同 Procedure node 的 nested precondition step（真实传统 5GC 样例使用） | `STEP_SCOPE_UNKNOWN` |
| `<nodeId>.postconditions.<stepId>.output[.field]` | 同理 | `STEP_SCOPE_UNKNOWN` |

v0 遇到未列出的动态 selector / 复杂 Runtime 表达式必须报 `BINDING_UNSUPPORTED`，不能按字符串分割后取最后一段蒙混通过。

**最长作用域优先**：在识别 `<nodeId>.preconditions.<stepId>.output` 之后再解析其子字段；不可把 `preconditions` 误认成根 node。`'on'` 必须按 outcome → nextNode 解析，不能当作 Script/Tool output 字段。

### 2.1 外部 Owner Schema 的显式导入位置（唯一规则）

新 v0 contract 使用以下**显式路径**，供 Compiler 绑定 `context.*`；这是新语言要求，不是兼容字段：

~~~yaml
spec:
  inputs:
    requestSchema: schemas/establishment-request.schema.yaml
    # 也可直接指向 lib/host/... 中 Owner 的 request Schema
  context:
    schemas:
      subject: lib/host/subject.schema.yaml
      facts: lib/host/facts.schema.yaml
      observations: lib/observations/context-observations.schema.yaml
~~~

- `requestSchema` 仅此一个权威，可以指向当前 Skill 的 `schemas/...`（由 Skill 自己拥有的业务接口），也可以指向 Project `lib/...`（由真实外部 Owner 提供）。**没有隐式同名搜索**。
- `spec.context.schemas.subject/facts/observations` 是可选映射，**但在出现对应 `context.subject/context.facts/context.observations` 引用时变为必填**；必须指向本次输入中的相应 `lib/host/...` 或 `lib/observations/...` 权威 Schema，不能指向 Root Routing Profile。
- 每个导入文件须为可以静态解析的 JSON Schema object。若路径缺失/未导入/指向无权威数据的配置文件，报 `EXTERNAL_SCHEMA_MISSING` 或 `LIB_SCHEMA_INVALID`；不猜测、自动生成或寻找替代品。
- `context.request.*` 必须使用 `spec.inputs.requestSchema`；`spec.context.requiredFacts` 只是名称集合**不是**字段级 Schema，不能拿它证明 `facts.createdsession_ingress` 的类型。
- 源端字段和 node/step 名称区分大小写，允许短横线（如 `ue-ref`、`nudm-sdm-get`），不能只用正则 `\w+` 接受一半名称。`context/inputs/nodes/steps` 的 Namespace 优先级按 §2 固定。
- 为让弱模型完成测试，`examples/script-contract-project` 只引用自己的 requestSchema 和 imported Tool，因此不需要虚构 Host schemas 作为正例；真实两个 5GC 迁移工程引用 subject/facts 时必须先导入真正 Owner 文件。

## 3. Schema path 校验

- Source path、lib path 均来自本次 `Request.Files`，全部相对于 Project 根，拒绝绝对路径、`..`、重复文件名/身份、目录逃逸、远程 `$ref`、无法解析的引用循环。
- `from: value` 绑定时从权威 Producer Schema 取字段类型，逐层进入 object.properties；若字段无定义，不能仅因为上层 `type: object` 就接受它。
- `with.key: {value: ...}` 对明确的目标 Input Schema 做静态字面量约束检查；对于表达式源，至少做明确类型匹配（对象/数组/标量、required/可空）。
- 将每个 Script 调用的 InputProjection 绑定到具体 call site（`skill+procedure+node+step`），**不产生全局 Script Input Schema 文件**。
- Producer 的 `$ref` 仅解析受支持的当前文档/本次文件集内显式引用；若 Host 或 Observation Owner 契约未导入，报缺失，而不是从配置文件猜测。
- 错误位置指向实际 `assurance/procedure.yaml` 的 binding 行，附 Producer/Consumer 来源路径；不能用错误中的任意运行数据充当 Schema。

## 4. 依赖与选择目标

- `targets=[]` 是整个 Project；非空是所选 Skills 的**传递依赖闭包**。若目标名称不存在，报 `TARGET_NOT_FOUND`。不能让无关 Skill 的局部 Script 错误阻塞一个不依赖它的目标。
- 跨 Skill Procedure 必须明确目标 Skill + Procedure；本地 Procedure 只在同一个 Skill 中解析。缺 Skill/Procedure 或图循环必须明确诊断，不引入自动选路。
- 在 build 输出里保留被选择 Skill 的全部必要 source，原样携带被实际引用的只读 ToolCatalog snapshot；不要把 Tool 实现或部署 Registration 当作 Skill 代码。
- **上下游数据的依赖性**应单独追踪。仅静态检查引用和 Schema，不模拟运行时 ordering/availability，也不读取网络事实。

## 5. Predicate / Observation 范围（静态最小值）

- 对 `verify: predicate-id`、contract 前后条件和 applicability 的 Predicate ID 检查存在性。
- Predicate 中引用的观察值定义只能检查结构和可解析的明确引用；执行时 freshness、Trace、CEL 真值属于 Host/Runtime，Compiler 不判定 PASS/FAIL/UNKNOWN。
- 不要求为迁移前历史 Predicate 的所有运行时内部字段造完整 Data Schema。若这些字段属于新严格链接范围而 Owner 未提供，给 `EXTERNAL_SCHEMA_MISSING` / `PREDICATE_SOURCE_INCOMPLETE`，不能制造假 PASS。

## 6. MAS / 实际样例

已核验固定源样本：
- 传统 `establish-pdu-session` 使用 `context.subject/context.request/context.facts`、`input.*`、`<node>.output.*`、`preconditions.<step>.output`、`<node>.preconditions.<step>.output`。
- MAS `establish-pdu-session-mas` 使用 `context.*`、`input.*`、`<node>.output.*`，以及 verify/reconcile 分支。
- 当前真实 ToolCatalog 12 个 Operations；MAS design Catalog 16 个 Operations，但有 `NOT_READY` 声明且缺落地 Effect/Adapter，因此仅作静态依赖测试。
- 这两套源码**是迁移前快照**：旧 Script metadata、复杂 Procedure 构造或未显式定义的 Host Schema，可以使新 v0 编译 FAIL。**不做旧格式兼容，不把这些失败解释成 Parser 必须支持旧语义**。
- 迁移后的新版 Project 必须针对 v0 语法与真实导入契约验收；如果还缺 Owner Schema，按缺口记录，不修改上游固定快照。

## 7. 最小诊断码表

`PROJECT_PATH_INVALID`、`PROJECT_DUPLICATE_FILE`、`YAML_PARSE`、`YAML_DUPLICATE_KEY`、`SKILL_ENTRY_MISSING`、`TARGET_NOT_FOUND`、`DEPENDENCY_NOT_FOUND`、`PROCEDURE_UNSUPPORTED`、`PROCEDURE_NOT_FOUND`、`NODE_TARGET_UNKNOWN`、`SCRIPT_NOT_FOUND`、`SCRIPT_TYPE_UNSUPPORTED`、`LIB_OPERATION_DUPLICATE`、`LIB_OPERATION_UNKNOWN`、`LIB_SCHEMA_INVALID`、`EXTERNAL_SCHEMA_MISSING`、`SOURCE_FIELD_UNKNOWN`、`BINDING_TYPE_MISMATCH`、`BINDING_UNRESOLVED`、`BINDING_UNSUPPORTED`、`STEP_SCOPE_UNKNOWN`、`PREDICATE_NOT_FOUND`。

实现可以保留已有诊断码的临时别名，但在 v0 最终正式对接前**统一到这张表**；不能同时存在两套不同含义的错误码。每项至少包含 severity、code、path、line/column（可得时）与 message；返回顺序稳定可重复。
