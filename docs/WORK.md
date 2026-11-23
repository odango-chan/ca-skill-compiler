# CA Skill Compiler — Current State

**Snapshot:** 2026-11-23 · [中文索引](README.zh-CN.md) · [release record](releases/ca-skill-compiler-v2.1.0.md)

| Track | Verified state |
|---|---|
| Compiler product | **v2.1.0 RELEASED** — [GitHub Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0) |
| Immutable release source | `b67fbb56c34d15b8d5a0903c8a7ca93d608f8f65` |
| Release acceptance | [Run #45100406980](https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980) **PASS**; eight published assets independently re-downloaded and SHA-256 verified |
| Current language | CA-SL `connectionagent.io/v1alpha5` **FROZEN**; only supported target |
| Frozen v1alpha5 Compiler API/Artifact | **RELEASED**, byte-identical v2.0.2 Artifact golden baseline on v2.1.0 |
| Development v1alpha1 | **RELEASED** API family; exact Interface Package snapshots and Dependency Lock |
| Development execution status | **NOT_ADMITTED** by design; Host admission is external |
| Real MAS author source | [#53](https://github.com/odango-chan/ca-skill-compiler/issues/53) separate offline integration regression, **not** a Compiler release gate |
| Owner, Provider, Deployment Grant, Host/Runtime/5GC qualification | External systems; Compiler release makes **no qualification claim** |

## Compiler-owned scope

One deterministic Go Core runs Native CLI and Go/WASM, with real Chromium Worker parity. It validates CA-SL source, static Tool/Capability/Schema/Evidence obligations, immutable snapshot closure, package-scoped Schema linking, exact Capability selections, two disjoint requirement classes and Artifact digest/bytes. A failed static check emits no authoritative artifact; Development Verify rebuilds and compares.

The v2.1.0 release contains [five Development machine contracts](../contracts/development/v1alpha1/README.md). The [v1alpha5 frozen contracts](../contracts/v1alpha5/README.md) and their artifact bytes remain unchanged. Historical frozen artifacts retain `generator.version=v2.0.2`; that is an artifact identity, not the current binary version.

## Architecture and remaining work

- [Compiler-only acceptance boundary](COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md), [current design](design/ca-toolchain-architecture-vnext.zh-CN.md), [Architecture diagrams](architecture/archify/README.md).
- [v2.1.0 release record](releases/ca-skill-compiler-v2.1.0.md), [v1alpha5 integration acceptance](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).
- The actual MAS Create/Modify/Release author-source offline integration test remains in [#53](https://github.com/odango-chan/ca-skill-compiler/issues/53); it is not interchangeable with Owner/Provider qualification.
- Package discovery, Registry, version resolution, signing, Publisher identity trust, Deployment Grants, Host Admission and live Tool invocation remain **outside Compiler**.

This file records the **current** released baseline only. The prior design and work-package diary is preserved in [WORK-history.md](https://github.com/odango-chan/ca-skill-compiler/blob/72b41548a9c4d29c6c49abed117a04a963454f22/docs/WORK-history.md) and the [workstreams archive](workstreams/README.md).
