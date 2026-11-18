# CA Skill Compiler — Current State

- **Product:** CA Skill Compiler v2.0.0 — RELEASE INPUTS PREPARED (not released)
- **Language:** CA-SL v1alpha5 — FROZEN / RELEASED; the only supported target
- **Integration PR:** [#38](https://github.com/odango-chan/ca-skill-compiler/pull/38) on `feat/compiler-v1alpha5-integration-20261117`
- **Product qualification:** NOT_QUALIFIED (real Owner / deployment / Host / Runtime / 5GC / MAS evidence is external).

## Previously released baseline (immutable)

- v1.0.0 product: [release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0), release commit `252fd6c6dab15fb92806426aeb19d49a36b08d5e`.
- v1.0.0 targeted CA-SL v1alpha4 and is historical evidence. Its tag, assets and release notes are not rewritten or re-tagged.
- v1alpha4 language authority (inheritance source): [ca-skills@7f84393](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).

## Current line

- Language authority: CA-SL v1alpha5 at [ca-skills@0abfd0e](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md); IngressCatalog v1alpha1 at [ca-skills@9412be9](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md).
- Architecture: one BuildRequest API, one CLI target, one static pipeline and one deterministic artifact path. There is no v1alpha4 API, CLI target or compatibility layer.
- Acceptance: `LOCAL_ACCEPTANCE_PASS`; GitHub CI `CI_UNAVAILABLE` (billing restriction, no steps executed). No CI PASS is claimed.

## Status

| Area | Status |
|---|---|
| Single-version v1alpha5 Compiler core / CLI | Implemented; locally verified |
| Inherited v1alpha4 semantics | Implemented in the same pipeline |
| Wait / Ingress | Implemented and locally verified |
| Evidence / Mutation / Reconcile / retry | Implemented static assurance; locally verified |
| Native and Go/WASM | Build verified; shared core |
| Chromium Worker parity | PASS (real headless Chromium) |
| Compiler-owned EW/IN register | 102/102 classified; 55 implemented+tested; 0 Compiler-static deferred |
| v1alpha5 contracts | FROZEN |
| Real Owner / Deployment / Host / Runtime / 5GC / MAS | NOT_QUALIFIED |
| v2.0.0 tag / GitHub Release | Not created; requires explicit human approval |

Detailed records:

- [v1alpha5 single-version acceptance](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md)
- [v2.0.0 release notes (prepared)](release-notes-v2.0.0.md)
- [Known limitations](known-limitations-v1alpha5.md)
- [CLI reference](cli.md)
- EW/IN register: `docs/workstreams/wp7-conformance-v1alpha5.json`

## Maintenance rules

- v0.1/v0.2 specifications, old work logs, previous work-package documents and source snapshots are historical records, not implementation authority.
- Current positive compile example: `web/parity-success` (ordinary Skill) and the trusted v1alpha5 fixture `web/parity-v1alpha5/`. The `examples/` directory contains explicitly labelled historical/migration fixtures.
- Do not introduce a second language target, package management, dependency acquisition, Runtime dispatch, Python execution or network access into the Compiler.
- Keep Compiler-owned static proof separate from Host/Runtime behavior and product qualification.
- v1.0.0 tag/assets remain immutable; publish v2.0.0 only after explicit approval.
