# Workstream Records

[简体中文](README.md)

This directory contains implementation plans, bounded work-package records and historical migration reports. It is not a live task queue.

## Current release status

Use these files for the current v1.0.0 state:

- [Compiler implementation and acceptance](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) — authoritative implementation status, local verification evidence, E-case dispositions and qualification boundary.
- [Known limitations](../known-limitations-v1alpha4.md) — current fail-closed behaviour and external blockers.
- [Remaining-work inventory](REMAINING-WORK-v1alpha4.json) — machine-readable inventory. `implementableInRepo` is empty; remaining entries require external publications or normative clarification. It is not an active implementation queue.
- [Conformance plan](conformance-v1alpha4.md) — frozen test strategy, not a live progress report. Use the acceptance record and `tests/conformance/v1alpha4/e-coverage.json` for current dispositions.

## Completed v1alpha4 implementation records

The following task envelopes describe work that has already been merged into `main` and included in CA Skill Compiler v1.0.0. Their original branch names, Draft-review wording and stop conditions are historical.

| Work package | Merged PR | Record |
|---|---:|---|
| WP3A — dependency identity | [#15](https://github.com/odango-chan/ca-skill-compiler/pull/15) | [WP3A](wp3a-dependency-identity-v1alpha4.md) |
| WP3B — Owner/export linking | [#16](https://github.com/odango-chan/ca-skill-compiler/pull/16) | [WP3B](wp3b-owner-export-linking-v1alpha4.md) |
| WP3C — Capability linking | [#17](https://github.com/odango-chan/ca-skill-compiler/pull/17) | [WP3C](wp3c-capability-linking-v1alpha4.md) |
| WP3D — integrated linking | [#18](https://github.com/odango-chan/ca-skill-compiler/pull/18) | [WP3D](wp3d-integrated-linking-v1alpha4.md) |
| WP4A/WP4B — Schema allowlist and registry | [#19](https://github.com/odango-chan/ca-skill-compiler/pull/19) | [WP4A](wp4a-schema-g4-v1alpha4.md), [WP4B](wp4b-schema-registry-v1alpha4.md) |
| WP4C — Schema assignability | [#20](https://github.com/odango-chan/ca-skill-compiler/pull/20) | [WP4C](wp4c-assignability-v1alpha4.md) |
| WP4D — CEL and Predicate signatures | [#21](https://github.com/odango-chan/ca-skill-compiler/pull/21) | [WP4D](wp4d-cel-v1alpha4.md) |
| WP5 — Procedure graph | [#22](https://github.com/odango-chan/ca-skill-compiler/pull/22) | [WP5](wp5-procedure-graph-v1alpha4.md) |
| WP6 — SourcePath and data-flow | [#23](https://github.com/odango-chan/ca-skill-compiler/pull/23) | See [acceptance record](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) |
| WP9–WP12 — Script ABI, artifact, parity and Studio integration | [#24](https://github.com/odango-chan/ca-skill-compiler/pull/24) | [WP11](wp11-parity-v1alpha4.md), [WP12](wp12-studio-integration-v1alpha4.md), [acceptance record](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) |

WP0–WP2, WP7–WP8 and the CLI are covered in the merged contract/implementation PRs and the final acceptance record. The remaining Evidence, identity-provenance and cross-snapshot Reconcile gaps are external-contract blockers; they remain fail-closed and are not silently promoted to PASS. Real 5GC/MAS qualification remains NOT_QUALIFIED.

## Archived execution plans

These documents preserve the historical Harness workflow and maintainer decisions. They are not current execution instructions:

- [DeepSeek handoff](deepseek-handoff-v1alpha4.md)
- [DeepSeek Harness profile](deepseek-harness-v1alpha4.md)
- [v1alpha4 resume decision](v1alpha4-resume-decision-20261116.md)

## Archived v0 / v0.1 records

These documents describe superseded prototypes, schemas, fixtures and migration checks. They are retained for traceability only and must not be used as current CA-SL v1alpha4 or Compiler v1.0.0 requirements:

- [v0 acceptance matrix](acceptance-matrix-v0.md)
- [v0 implementation Goal](local-implementation-goal.md)
- [v0 Harness prompt](local-harness-prompt.md)
- [v0 PDU/MAS migration review](migration-review-v0.md)
- [v0.1 JSON Schema support matrix](json-schema-support-v0.1.md)
- [v0.1 language alignment report](language-alignment-v0.1.md)
- [v0.1 official Skill acceptance report](official-skill-acceptance-v0.1.md)
- [v0.1 ToolCatalog report](tool-catalog-v0.1.md)

No historical report is deleted. For the current release, start with the acceptance record and known limitations rather than an old work-package prompt.
