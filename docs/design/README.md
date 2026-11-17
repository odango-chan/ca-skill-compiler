[简体中文](README.zh-CN.md)

# Compiler Design Documents

This directory contains Compiler-owned contracts, implementation maps and historical architecture notes. It does not replace the CA-SL language specification.

## Authority and status

The authority order is:

1. **CA-SL language semantics:** the frozen [CA-SL v1alpha4 specification](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md), pinned to commit `7f84393ee8c395e4d8c39a74cb5000a1f896588d`. This is the language authority.
2. **Compiler behaviour:** [Compiler Contract v1alpha4](compiler-contract-v1alpha4.md), subordinate to the language specification. It defines what the Compiler must validate and what remains outside Compiler ownership.
3. **Compiler-owned interfaces and artifacts:** the BuildRequest, Output API, Dependency Snapshot Digest and Artifact contracts below.
4. **Specialized linking contracts:** Owner, Procedure and Script contracts below.
5. **Implementation references:** the Pipeline map describes where checks and tests live; it does not amend the contracts. The OSS ADR records library choices; it does not define CA-SL semantics.

For implementation status and qualification, use the [final acceptance record](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) and [known limitations](../known-limitations-v1alpha4.md). `LOCAL_ACCEPTANCE_PASS` records executed local verification; it does not mean every E-case is a whole-case PASS or that real Owner/Deployment and 5GC qualification has been completed. Product qualification remains **NOT_QUALIFIED**.

## Domain and procedure profiles

- [5GC SMF Procedure Profile](../profiles/5gc-smf-procedure-profile.md) — SMF-oriented 3GPP → Tool → Skill mapping, Node/Observation boundaries, v1alpha4 asynchronous behavior and the pinned PDU Session reference map.

## Current v1alpha4 contracts

- [Compiler Contract](compiler-contract-v1alpha4.md) — Compiler responsibilities, compilation pipeline, static obligations and trust boundaries.
- [BuildRequest](build-request-v1alpha4.md) — versioned input envelope, source files, immutable dependency snapshots and trusted build context.
- [Compiler Output API](compiler-output-contract-v1alpha4.md) — Check, BuildResult, DependencyInspection, diagnostics and deterministic ordering.
- [Dependency Snapshot Digest](dependency-snapshot-digest-v1alpha4.md) — canonical byte-level digest shared by Compiler and Studio.
- [Owner Contract Linking](owner-contracts-v1alpha4.md) — ToolCatalog, CapabilityContracts, Evidence and Mutation/Reconcile linking boundaries.
- [Procedure Linker](procedure-link-v1alpha4.md) — graph, stage, route, argument, Schema and Evidence static linking.
- [Script Contract](script-contract-v1alpha4.md) — Skill-owned Python Script roles, derived input/output types and static analysis boundary.
- [Compiler Artifact](compiler-artifact-v1alpha4.md) — deterministic output tree, manifest and generated Schema serialization.
- [Compiler Pipeline map](compiler-pipeline-v1alpha4.md) — implementation files and test anchors; descriptive, not normative.
- [Open-source Dependency Selection](oss-dependency-selection-v1alpha4.md) — pinned third-party libraries, adapters and implementation constraints.

## Historical design records

These files are retained for traceability. Their layouts, API shapes, source versions, ToolCatalog assumptions and acceptance plans are superseded; do not use them as requirements for Compiler v1.0.0 or CA-SL v1alpha4.

- [Skill Compiler v0 architecture brief](skill-compiler-v0.md)
- [Compiler Contract v0.1](compiler-contract-v0.1.md)
- [Procedure Link Contract v0](procedure-link-v0.md)
- [Script Contract v0](script-contract-v0.md)

The historical documents are not deleted because they explain earlier design decisions. Their archive notices point to the current contracts.
