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

## Historical specifications — read only

The files named `*-v1alpha4.md`, `*-v0.md`, and `*-v0.1.md` document earlier released or prototyped versions, not currently accepted Compiler input formats. Inherited v1alpha4 *language semantics* remain implemented in the single v1alpha5 pipeline, but a v1alpha4 BuildRequest or language target is rejected. Historic materials were retained for auditability rather than deleted or silently rewritten.

- [v1alpha4 Compiler contract](compiler-contract-v1alpha4.md) · [BuildRequest](build-request-v1alpha4.md) · [Output](compiler-output-contract-v1alpha4.md)
- [v1alpha4 Artifact](compiler-artifact-v1alpha4.md) · [Dependency digest](dependency-snapshot-digest-v1alpha4.md)
- [v1alpha4 Owner/Procedure/Script](owner-contracts-v1alpha4.md) · [Procedure](procedure-link-v1alpha4.md) · [Script](script-contract-v1alpha4.md)
- [v0 architecture brief](skill-compiler-v0.md) · [v0.1 contract](compiler-contract-v0.1.md)

For current product delivery, use the [v2.1.0 release record](../releases/ca-skill-compiler-v2.1.0.md); historic work-package status belongs in the [workstreams archive](../workstreams/README.md).
