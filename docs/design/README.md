# Compiler design and contracts

[简体中文](README.zh-CN.md) · [Documentation](../README.md)

**Current product:** v2.1.0 (released). **Language authority:** [CA-SL v1alpha5 frozen specification](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md), owned by ca-skills. Design notes never override frozen language or published machine contracts.

## Released Compiler contracts

- [v1alpha5 machine contracts](../../contracts/v1alpha5/README.md): frozen BuildRequest / BuildResult / DependencyInspection / Artifact contract family. The embedded Artifact generator stamp remains `v2.0.2` for byte parity.
- [Development v1alpha1 machine contracts](../../contracts/development/v1alpha1/README.md): released with v2.1.0; exact immutable Interface Package + Lock, Build/Verify, always `NOT_ADMITTED`.

## Active implementation and architecture reference

- [Single v1alpha5 Compiler Core](compiler-core-v1alpha5.md)
- [v1alpha5 Artifact model](compiler-artifact-v1alpha5.md)
- [Toolchain architecture: Compile / Interface Link / Build / Host Admission](ca-toolchain-architecture-vnext.zh-CN.md)
- [Development implementation contract](compiler-development-build-pilot.zh-CN.md)
- [Development implementation and final architecture review (historical evidence)](compiler-development-build-implementation-report.zh-CN.md)
- [Evidence / Assurance responsibility audit](compiler-evidence-assurance-responsibility-audit.zh-CN.md)
- [Compiler-only roadmap and external integration boundary](compiler-evolution-roadmap-vnext.zh-CN.md)
- [Archify source and diagrams](../architecture/archify/README.md) · [5GC profile review snapshot](../profiles/5gc-smf-procedure-profile.md)

## Historical design provenance

The 14 pre-v1alpha5 design-prose documents are **not** active Compiler inputs and no longer crowd this directory. They remain available at the [immutable historical-design snapshot](https://github.com/odango-chan/ca-skill-compiler/tree/72b41548a9c4d29c6c49abed117a04a963454f22/docs/design) together with their original citations. Frozen **machine** contracts in `/contracts/` remain in the active tree, as do the current v1alpha5/Development design documents.

Current product evidence: [v2.1.0 release](../releases/ca-skill-compiler-v2.1.0.md). Past WP records: [historical index](../workstreams/README.md).
