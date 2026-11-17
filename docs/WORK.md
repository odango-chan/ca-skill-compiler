# CA Skill Compiler — Current Work (v1alpha4)

**Updated:** 2026-11-16  
**Language:** CA-SL v1alpha4 · frozen at `odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`  
**Compiler product status:** NOT_QUALIFIED  
**Execution state:** CONTINUE INDEPENDENT WPs; NARROW DIAGNOSTIC/OWNER BLOCKERS

## Authority

- Frozen language spec: [CA-SL v1alpha4](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md), blob `b70064c5737b8ee2b3d1435fe496357f44b42773`.
- Compiler contracts: `docs/design/*-v1alpha4.md` and `contracts/v1alpha4/`.
- Root `AGENTS.md` is v1alpha4; the former v0.1 rules are archived in `docs/archive/AGENTS-v0.1.md`.
- [Maintainer's resume decision](workstreams/v1alpha4-resume-decision-20261116.md).
- [Language-owner diagnostic errata request #59](https://github.com/odango-chan/ca-skills/issues/59).

## Live PR chain

Open stacked PR #11–#23; **none is authorized to merge automatically**.

| PR range | Status / scope |
|---|---|
| #11–#14 | v1alpha4 source admission, frozen contracts, WP1 BuildRequest/digest, WP2 YAML AST |
| #15–#18 | WP3A–WP3D Dependency identity, Owner/ToolCatalog, Capability and integrated linking |
| #19–#21 | WP4A–WP4D Schema G.4/registry/F.5 assignability/CEL |
| #22 | WP5 Procedure graph |
| [#23](https://github.com/odango-chan/ca-skill-compiler/pull/23) | WP6 SourcePath, producer dominance/availability, argument typing; draft, base `feat/compiler-v1alpha4-wp5-procedure-graph-20261116` |

Implementations are present in these PR branches; **do not treat the above as overall conformance PASS**. The most recent submitted local report lists Go 1.27.4 tests/vet/CGo-free Native/WASM build/diff checks as passing, without GitHub Actions. Re-execute acceptance commands after changes.

## Current execution target

Start from the reviewed PR #23 branch head and create a new stacked Draft PR for WP9 Script ABI. Continue with WP10, then independent WP7/WP8 checks, WP11 Chromium and WP12 Studio. See [current handoff](workstreams/deepseek-handoff-v1alpha4.md) and [resume decision](workstreams/v1alpha4-resume-decision-20261116.md). Each WP must have its own tests and evidence.

## Exact blockers (not the whole Goal)

- Invalid Observation Owner outcomes; malformed `evidence.requires` lists; unknown evaluator type: exact Appendix H Code/Phase mapping needs language-owner clarification [#59](https://github.com/odango-chan/ca-skills/issues/59). Block only those mapping fixtures.
- `SOURCE_PATH_INVALID / C_STATIC` applies to nested inline map/array Expression values; allowed outer empty arguments containers are unaffected.
- Real Evidence Descriptor / identity-input / cross-snapshot Reconcile trusted publication is missing: keep positive owner-integrated paths fail-closed.
- Real v1alpha4 5GC qualification WP13 is NOT_QUALIFIED. MAS is NOT_READY for executable acceptance.

## Historical notes

Prior v0/v0.1 implementation status is archived in [WORK history](WORK-history.md). Earlier WP3A-only instructions in PR #15 are past handoff snapshots, not current global Goal authority.
