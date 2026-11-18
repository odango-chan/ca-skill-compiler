# 下一代 CA Skill Compiler 开发计划

**规范状态：** CA-SL v1alpha5 与 IngressCatalog v1alpha1 已正式 FROZEN / RELEASED。  
**Compiler 状态：** v1alpha5 候选实现已进入集成阶段，但仍为 PARTIAL STATIC COVERAGE，尚不能合并或发布。  
**当前集成 PR：** [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38)，分支 `feat/compiler-v1alpha5-integration-20261117`。

## 1. 已发布的规范基线

| 契约 | 正式身份 | 发布依据 |
|---|---|---|
| CA-SL | `connectionagent.io/v1alpha5` | ca-skills PR #66，merge commit `0abfd0e1` |
| IngressCatalog | `connectionagent.io/ingresscatalog/v1alpha1` | ca-skills PR #62，merge commit `9412be98` |
| Ingress Diagnostic Registry | `connectionagent.io/ingress-diagnostics/v1alpha1` | ca-skills PR #62，merge commit `9412be98` |

PR #61 是已关闭的预发布 Draft，不是当前规范发布依据。后续实施必须读取 ca-skills main 上的正式单文件规范与 Ingress 合同。

**重要语义：** v1alpha5 继承 v1alpha4 的规范性语义，除非 v1alpha5 明确增补或替代。Compiler 必须实现这些继承语义，但不要求保留旧 v1alpha4 API、旧项目兼容、旧 CLI 或双 Parser / Linker。v1.0.0 历史 Release 不重写；当前开发线允许为 v1alpha5 重构或删除旧实现。

## 2. 当前实现状态

WP1–WP8 已有集成候选：IngressCatalog / Schema Closure、Wait AST、Trigger / Correlation、wait.output、Procedure Graph、Response Boundary、诊断追踪、确定性 Artifact 与 Native/WASM/Chromium parity。

但当前 v1alpha5 Build 尚未实现完整的继承语义，主要缺口包括：

- 完整 Skill package admission：SKILL.md、Contract、PredicateSet、Schemas、Scripts；
- 普通非 Wait Operation / Observation 的 ToolCatalog、Owner、Schema 链接；
- Capability linking；
- Schema allowlist / registry / closure / assignability、CEL；
- Predicate 参数签名、Evaluator、Script ABI；
- 完整 global Precondition、Evidence、Mutation、Reconcile、retry 静态规则；
- 19 项当前标记为 deferred 的 Compiler-static EW/IN obligations；
- IN-064、IN-065、IN-066 尚未进入当前 99-case conformance register；
- 当前 API DTO 仍存在 v1alpha4 / v1alpha5 共享类型耦合；
- Artifact manifest 仍标记 `CANDIDATE_PARTIAL`。

因此不能把当前实现称为完整 v1alpha5 Compiler。

## 3. PR 结构

#38 集成分支已经包含 #30–#37、#39、#40 对应分支的提交。应以 #38 作为唯一集成候选，不再按旧计划把这些 stacked PR 逐个合入 main。完成集成审查后，将子 PR 标记为被 #38 取代并关闭；保留它们的提交历史。

#38 当前保持 Draft，不合并。

## 4. 下一阶段

完整任务见 [v1alpha5 Full Language Completion Goal](v1alpha5-full-language-completion-goal.zh-CN.md)。

| WP | 任务 | 目标 |
|---|---|---|
| WP9 | Core / API 架构收敛 | 清理共享 v1alpha4 DTO 耦合，建立干净 v1alpha5 pipeline |
| WP10 | Package / Dependency / Owner / Tool linking | 完整 Skill package 与非 Wait callable linking |
| WP11 | Schema / CEL / Predicate / Script | 实现全部继承的静态类型与脚本契约 |
| WP12 | Procedure / Evidence / Mutation / Reconcile / Retry | 完整 assurance 语义并与 Wait Graph 融合 |
| WP13 | Ingress / Wait conformance | 完成 19 项 deferred，并补齐 IN-064..066 |
| WP14 | Artifact / CI / Native-WASM-Chromium | 移除 partial scope、完成完整验收 |

Studio/LSP 成功 Build 展示作为后续产品集成，不应混入 Compiler 静态符合性结论。

## 5. 验收与资格边界

- Harness 已记录本地 Go tests、Vet、Native、WASM、CLI 与真实 Chromium parity PASS；这属于 Harness 报告，当前复核未独立重跑。
- 最新 GitHub Actions Run `44091765010` 的 Job 没有执行任何步骤，不能视为 CI PASS，也不能据此判断源码测试失败。
- 真实 Owner / Deployment / Integration Adapter / Host / Runtime / 5GC / MAS qualification 仍为 NOT_QUALIFIED。
- 完整 v1alpha5 Compiler 只有在继承静态语义、全部 Compiler-owned EW/IN obligations、Native/WASM/Chromium parity 和真实 CI 均通过后，才可进入 Ready / Merge / Release 评审。

