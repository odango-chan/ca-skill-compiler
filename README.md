# CA Skill Compiler

**Current target: CA-SL v1alpha4.** This repository implements a deterministic, stateless compiler for the frozen ConnectionAgent Skill Language. The current language target is not the old v0.1 / v0.2 contract.

## Normative authority

- **CA-SL v1alpha4:** [ca-skills immutable commit 7f84393](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md)
- **Frozen specification file blob:** b70064c5737b8ee2b3d1435fe496357f44b42773
- Companion authoring and schema notes are informative only. The single-file specification is the only normative language authority.

Do not use floating main, old v0.2 language docs, or old v0.1 Compiler contracts as the v1alpha4 implementation authority.

## Compiler contract set

| Document | Purpose |
|---|---|
| [Compiler Contract v1alpha4](docs/design/compiler-contract-v1alpha4.md) | Scope, pipeline, semantics and acceptance boundary |
| [BuildRequest v1alpha4](docs/design/build-request-v1alpha4.md) | Source, dependency snapshots and trusted build context |
| [Owner Contract Linking v1alpha4](docs/design/owner-contracts-v1alpha4.md) | ToolCatalog, CapabilityContracts, Evidence and trusted Owner attribution |
| [Procedure Linker v1alpha4](docs/design/procedure-link-v1alpha4.md) | AST, graph, SourcePath, Schema and Capability linking |
| [Script Contract v1alpha4](docs/design/script-contract-v1alpha4.md) | Procedure / Predicate Script ABI and derived Schemas |
| [Compiler Artifact v1alpha4](docs/design/compiler-artifact-v1alpha4.md) | Deterministic output tree and generated Schema serialization |
| [Dependency digest](docs/design/dependency-snapshot-digest-v1alpha4.md) | Cross-language immutable snapshot digest |
| [Conformance plan](docs/workstreams/conformance-v1alpha4.md) | E01–E120, H diagnostics, Native/WASM and 5GC gates |
| [5GC SMF Profile](docs/profiles/5gc-smf-procedure-profile.md) | 3GPP → Tool → Skill traceability; outside generic Compiler semantics |
| [Studio integration](docs/studio-integration-v1alpha4.md) | Go/WASM and Studio boundary |
| [Machine contracts](contracts/v1alpha4/README.md) | JSON Schemas and golden vectors |

## Version identifiers

These identifiers are independent and must be checked explicitly:

- BuildRequest API: connectionagent.io/compiler-build-request/v1alpha4
- Language target: connectionagent.io/v1alpha4
- BuildResult API: connectionagent.io/compiler-build-result/v1alpha4
- ToolCatalog: connectionagent.io/toolcatalog/v1alpha4
- CapabilityContracts: connectionagent.io/capability-contracts/v1alpha4
- Evidence trust bindings: connectionagent.io/evidence-trust-bindings/v1alpha4
- Compiler Artifact: connectionagent.io/compiler-artifact/v1alpha4
- Dependency snapshot digest framing: CA-SKILL-DEPENDENCY-SNAPSHOT\0v1\0

A BuildRequest API version is not a substitute for languageTarget; ToolCatalog and CapabilityContracts apiVersion values are not inferred from either one.

## Current implementation status

- Compiler PR #5 is the Go v0.1 implementation base.
- Compiler PR #11 adds a v1alpha4 source/graph admission slice and deliberately fails closed when trusted Owner / CapabilityContracts inputs are unavailable.
- This document set defines the next implementation target. It does **not** claim Compiler v1alpha4 conformance, complete linking, Native/WASM parity, Owner qualification or live 5GC acceptance.

The old v0.1 documents and schemas remain historical records. They are not fallback contracts and must not be silently reused for v1alpha4.
