# CA Skill Compiler — Current Work (v1alpha4)

**Language:** CA-SL v1alpha4, frozen at
`odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`
(blob `b70064c5737b8ee2b3d1435fe496357f44b42773`).

**Compiler implementation status:** all frozen, implementable semantics implemented and
locally verified; external-contract branches fail closed.
**Product qualification status:** NOT_QUALIFIED (WP13 external).
**Evidence:** [Acceptance Status](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md),
[Local Verification](local-verification-v1alpha4.md),
`tests/conformance/v1alpha4/e-coverage.json`.

## Authority

- Frozen language spec: [CA-SL v1alpha4](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).
- Compiler contracts: `docs/design/*-v1alpha4.md` and `contracts/v1alpha4/`.
- Root `AGENTS.md` selects v1alpha4; v0.1 rules are archived in `docs/archive/AGENTS-v0.1.md`.
- [Language-owner diagnostic errata request #59](https://github.com/odango-chan/ca-skills/issues/59).

## Active work state

| Item | State |
|---|---|
| Branch | `feat/compiler-v1alpha4-wp9-script-abi-20261116` |
| Active PR | Draft [#24](https://github.com/odango-chan/ca-skill-compiler/pull/24), base `feat/compiler-v1alpha4-wp6-sourcepath-20261116` |
| Historical stack | PR #11–#23 remain open; none is merged or auto-mergeable |
| CLI | `skillc check` / `build` / `inspect` implemented; see [CLI](cli-v1alpha4.md) |
| Local gates | PASS (recorded in Local Verification; GitHub Actions unavailable, not used as evidence) |

Do not discard, close, silently replace or force-push the stacked PR history. No PR is
merged and no branch is pushed directly to `main`.

## Remaining items

All remaining items are recorded in [Known Limitations](known-limitations-v1alpha4.md):

1. Eight E-cases need an unfrozen external publication (Evidence Descriptor format,
   deployment Scope algebra, identity-input trusted publication) — `BLOCKED_EXTERNAL_CONTRACT`.
2. E71/E74 need a normative clarification for `PREDICATE_EVIDENCE_REQUIRED` —   `BLOCKED_NORMATIVE_MAPPING`.
3. WP13 real 5GC/MAS qualification is NOT_QUALIFIED.
4. GitHub Actions remains unavailable; local commands are the acceptance evidence.

Optional implementation-depth items that are not external blockers (exact WP5
auto-advance dominance, final-Postcondition precision, WP8 cross-snapshot depth) are listed
in [Acceptance Status](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) and are not counted as
product conformance.

## Historical notes

Prior v0/v0.1 implementation status and the earlier WP1–WP2 milestone narrative are
archived in [WORK history](WORK-history.md). Historical status notes are not present
evidence.
