[简体中文](README.zh-CN.md)

# Documentation

This directory contains the technical reference and maintenance records for CA Skill Compiler.

## For users

- [CLI reference](cli-v1alpha4.md) — commands, input formats, options, output and exit codes.
- [Known limitations](known-limitations-v1alpha4.md) — features that are intentionally rejected or remain unavailable.
- [Release notes](release-notes-v1.0.0.md) — changes and verification details for v1.0.0.
- [Examples and fixtures](../examples/README.md) — current sample and historical fixtures.

## Language and compiler contracts

- [5GC SMF Procedure Profile](profiles/5gc-smf-procedure-profile.md) — SMF-oriented 3GPP → Tool → Skill authoring and traceability rules.
- [Compiler design documents](design/README.md) — current v1alpha4 Compiler contracts and archived design records.
- [CA-SL v1alpha4 specification](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md) — normative language definition.
- [Compiler Contract](design/compiler-contract-v1alpha4.md) — compiler responsibilities and behaviour.
- [Compiler Pipeline](design/compiler-pipeline-v1alpha4.md) — validation stages and implementation mapping.
- [BuildRequest](design/build-request-v1alpha4.md) — compiler input contract.
- [Dependency Snapshot Digest](design/dependency-snapshot-digest-v1alpha4.md) — digest calculation and cross-language test vectors.
- [Compiler Output API](design/compiler-output-contract-v1alpha4.md) — Check, BuildResult and DependencyInspection.
- [Owner Contract Linking](design/owner-contracts-v1alpha4.md) — dependency ownership and capability linking.
- [Procedure Linker](design/procedure-link-v1alpha4.md) — Procedure graph and static analysis.
- [Script Contract](design/script-contract-v1alpha4.md) — Python Script interface and Schema derivation.
- [Compiler Artifact](design/compiler-artifact-v1alpha4.md) — artifact layout and manifest.
- [Studio integration](studio-integration-v1alpha4.md) — browser/WASM integration boundary.
- [Machine contracts](../contracts/v1alpha4/README.md) — JSON Schemas and golden vectors.

## Development and maintenance

These documents are intended for contributors and maintainers; they are not required for ordinary CLI use.

- [Local verification](local-verification-v1alpha4.md) — reproducible test and build commands.
- [Compiler acceptance record](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) — implementation evidence and conformance coverage.
- [Current maintenance status](WORK.md) — repository maintenance notes.
- [Work history](WORK-history.md) — historical development log.
- [Workstream records](workstreams/README.md) — implementation and conformance planning.
- [Dependency selection](design/oss-dependency-selection-v1alpha4.md) — selected third-party libraries and rationale.

The root [README](../README.md) is the project introduction and quick-start guide. This index keeps detailed engineering records out of the main entry point.
