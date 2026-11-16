# CA Skill Compiler — Current Work (v1alpha4)

**Updated:** 2026-11-16  
**Language target:** CA-SL v1alpha4  
**Overall implementation status:** NOT_QUALIFIED

## Frozen authority

- Normative specification: [CA-SL v1alpha4](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md)
- Immutable commit: `7f84393ee8c395e4d8c39a74cb5000a1f896588d`
- File blob: `b70064c5737b8ee2b3d1435fe496357f44b42773`
- Compiler implementation contracts: `docs/design/*-v1alpha4.md`
- No floating branch, v0, v0.1, v0.2 or v1alpha3 document is an implementation authority.

## Current PR stack

| PR | Scope | Status |
|---|---|---|
| [#11](https://github.com/odango-chan/ca-skill-compiler/pull/11) | v1alpha4 source admission | Open / Draft |
| [#12](https://github.com/odango-chan/ca-skill-compiler/pull/12) | Frozen Compiler contracts, OSS ADR and Harness profile | Open |
| [#13](https://github.com/odango-chan/ca-skill-compiler/pull/13) | WP1 BuildRequest and digest | Open; PR description records Go/build gates PASS |
| [#14](https://github.com/odango-chan/ca-skill-compiler/pull/14) | WP2 normalized YAML parser and AST | Open; PR description records Go/build gates PASS |

Merge order remains #11 → #12 → #13 → #14. Do not merge a later stacked PR before its base is present.

## Current implementation facts

- WP1: BuildRequest / BuildResult / DependencyInspection models, strict request decoding, limits, digest and Native/WASM entrypoints are in PR #13.
- WP2: goccy/go-yaml v1.19.2 normalized AST, source positions, scalar preservation and legacy parser-path removal are in PR #14.
- The selected San, CEL, Python CST and JCS adapters are not yet integrated.
- Full dependency/Owner/Capability linking, Schema/CEL, Procedure graph, SourcePath, Evidence, Mutation/Reconcile, Script ABI, Artifact, Chromium parity and Studio integration remain incomplete.
- Real v1alpha4 5GC Owner/Deployment publications are NOT_QUALIFIED; MAS is NOT_READY as an executable baseline.

## Active DeepSeek handoff

- Handoff entrypoint: [DeepSeek Handoff](workstreams/deepseek-handoff-v1alpha4.md)
- First bounded task: [WP3A — Dependency Snapshot Identity](workstreams/wp3a-dependency-identity-v1alpha4.md)
- Machine-readable fixture plan: [WP3A fixture plan](../tests/conformance/v1alpha4/wp3a-fixture-plan.json)
- Assigned branch: `feat/compiler-v1alpha4-wp3a-dependency-identity-20261116`
- Branch base: WP2 branch `feat/compiler-v1alpha4-wp2-yaml-ast-20261116` at `55ba5ebc37e9d54b6ee21e547dae188ca17785c9`

A separate exploratory WP3 branch exists but is not the approved handoff base. The assigned task branch starts cleanly from WP2. The task branch contains handoff documentation only; WP3A code has not yet been implemented.

## Non-claims

- CI PASS for WP1/WP2 is not full v1alpha4 conformance.
- A valid dependency digest is not trusted identity without the matching trusted binding.
- Trusted dependency identity is not Owner export qualification or Capability authorization.
- A synthetic fixture is not a real Owner publication.
- Compiler static linking is not Host/Runtime execution or live 5GC acceptance.

## Historical record

The prior v0 and v0.1 implementation log is preserved in [WORK history](WORK-history.md). It is historical context only and must not be used as v1alpha4 implementation authority.
