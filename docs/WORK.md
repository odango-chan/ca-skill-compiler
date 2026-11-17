# CA Skill Compiler — Current State

- **Product:** CA Skill Compiler v1.0.0 — RELEASED
- **Language:** CA-SL v1alpha4 — FROZEN
- **Release commit:** `252fd6c6dab15fb92806426aeb19d49a36b08d5e`
- **Product qualification:** NOT_QUALIFIED (real Owner / deployment / 5GC evidence is external).

## Release baseline

- Repository: [odango-chan/ca-skill-compiler](https://github.com/odango-chan/ca-skill-compiler)
- Release: [v1.0.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0)
- Language authority: [ca-skills@7f84393](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md), blob `b70064c5737b8ee2b3d1435fe496357f44b42773`.
- v1.0.0 tag points to the release commit on `main`; published assets include Native CLI binaries, Go/WASM and SHA-256 checksums.
- Release acceptance: `LOCAL_ACCEPTANCE_PASS`. The recorded local test, vet, Native/WASM build, CLI E2E and real Chromium parity commands passed on the release commit. GitHub Actions was unavailable and is not claimed as PASS.

## Current maintenance work

The post-release repository hygiene work is being performed on `chore/post-v1.0.0-repository-hygiene`.

This work does not change the v1.0.0 tag or language semantics. It removes unreachable pre-v1alpha4 implementation code, corrects stale maintenance/acceptance records, documents the current example set and closes superseded pre-v1alpha4 pull requests. Review and verification status are tracked in [PR #26](https://github.com/odango-chan/ca-skill-compiler/pull/26).

## Supported language and Wait boundary

The current Compiler accepts only `connectionagent.io/v1alpha4`.

**The latest author-level Wait semantics are not implemented in v1.0.0.** The compiler recognizes and rejects legacy Wait/Await syntax as unsupported v1alpha4 syntax. Do not backport Wait into v1alpha4 or infer semantics from Runtime behavior. Future Wait support requires a separately versioned language/compiler target and aligned Host/Runtime/Ingress contracts, including event-generation isolation and late/duplicate event handling.

## Build behavior and current limitations

Check and Build use the same v1alpha4 static validation pipeline. A successful Build produces a deterministic artifact tree, but only after all static checks and feature-scoped fail-closed gates pass.

The following positive paths remain fail-closed in the current release:

1. Evidence-bearing Skills that require the unfrozen Owner Evidence Descriptor physical format.
2. Mutation-bearing Skills, pending complete Mutation/Reconcile positive-path semantics and the cross-snapshot association contract.
3. Retry-bearing Skills, pending full retry contract compatibility and path-coverage analysis.

Static sub-checks for Evidence, Mutation, Reconcile and Retry are implemented; this does not mean the complete positive Build path is qualified. See [Known Limitations](known-limitations-v1alpha4.md).

## Implementation and qualification status

| Area | Status |
|---|---|
| v1alpha4 Compiler Core / CLI | Implemented; locally verified |
| Native and Go/WASM | Build verified; shared Core |
| Chromium Worker parity | PASS on the release commit |
| Skill Studio read-only integration | PASS on the release commit |
| External Evidence / identity-input / cross-snapshot publications | BLOCKED_EXTERNAL_CONTRACT |
| E71/E74 diagnostic mapping | BLOCKED_NORMATIVE_MAPPING; see [ca-skills #59](https://github.com/odango-chan/ca-skills/issues/59) |
| Real 5GC / SMF / MAS qualification | NOT_QUALIFIED |
| Latest Wait language semantics | Not part of v1alpha4; future version |

The authoritative case-level disposition is `tests/conformance/v1alpha4/e-coverage.json`; the summary is [Compiler Acceptance](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md).

## Maintenance rules

- v0.1/v0.2 specifications, old work logs and source snapshots are historical records, not implementation authority.
- Current positive compile example: `web/parity-success`. The `examples/` directory contains explicitly labelled historical/migration fixtures.
- Do not introduce package management, dependency acquisition, Runtime dispatch, Python execution or network access into Compiler.
- Preserve the frozen language target and diagnostic identities.
