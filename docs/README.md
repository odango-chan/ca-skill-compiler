# CA Skill Compiler — Documentation

[简体中文](README.zh-CN.md) · [Project](../README.md) · [Release v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0)

**Current:** Compiler v2.1.0, CA-SL `connectionagent.io/v1alpha5` only. Distinguish the released Compiler product from external Host/Owner/Provider/5GC qualification.

## Start here

| Need | Canonical entry |
|---|---|
| CLI usage and inputs | [CLI](cli.md) · [中文](cli.zh-CN.md) |
| Product state / outstanding integration | [Current state](WORK.md) |
| Release SHA, assets and acceptance | [v2.1.0 release record](releases/ca-skill-compiler-v2.1.0.md) · [release workflow](https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980) |
| Compiler/Host responsibility boundary | [Independent acceptance](COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md) |
| Current constraints | [Known limitations](known-limitations-v1alpha5.md) |
| Source interfaces | [Frozen v1alpha5 contracts](../contracts/v1alpha5/README.md) · [Development v1alpha1 schemas](../contracts/development/v1alpha1/README.md) |
| Architecture and diagrams | [Design index](design/README.md) · [Archify diagrams](architecture/archify/README.md) |
| Positive test fixtures | [Examples](../examples/README.md) |
| Historical plans, prior versions | [Workstreams archive index](workstreams/README.md) · [historical work log](WORK-history.md) |

## Normative order

1. CA-SL v1alpha5 [released language specification](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md) and IngressCatalog v1alpha1 in ca-skills.
2. Frozen Compiler-owned [v1alpha5](../contracts/v1alpha5/README.md) and [Development v1alpha1](../contracts/development/v1alpha1/README.md) machine contracts from their published releases.
3. [Architecture](design/ca-toolchain-architecture-vnext.zh-CN.md), [Compiler Core](design/compiler-core-v1alpha5.md), implementation reports and version-specific acceptance.
4. Historical v0/v0.1/v1alpha4 documents, annotated as archival, **never current build inputs**.

Development Build is a supported v2.1.0 product interface, but its Artifact is always `NOT_ADMITTED`. No Compiler document is a substitute for an Owner signature, Deployment Grant, Host Admission or runtime qualification.
