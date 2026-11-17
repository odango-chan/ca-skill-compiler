# Maintenance Status

## Release

- Product: CA Skill Compiler v1.0.0
- Language target: CA-SL v1alpha4
- Release: [v1.0.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0)
- Release commit: `252fd6c6dab15fb92806426aeb19d49a36b08d5e`

The release tag and its published assets are retained as the v1.0.0 baseline.

## Completed maintenance

- [PR #26](https://github.com/odango-chan/ca-skill-compiler/pull/26) merged the post-release repository cleanup.
- The unreachable pre-v1alpha4 implementation files were removed. `legacy_syntax.go` remains to report the frozen v1alpha4 unsupported-syntax diagnostics.
- Repository instructions, acceptance records and example descriptions were brought up to date.
- The project README is maintained in English and Simplified Chinese. Detailed engineering records are separated from the user-facing documentation.

## Workstream status

The v1alpha4 implementation work packages were merged through PRs #11–#24 (see the [final acceptance record](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)), and v1.0.0 was published through [PR #25](https://github.com/odango-chan/ca-skill-compiler/pull/25). There is no active Compiler implementation work package in the repository. Remaining items are external-contract or normative-mapping blockers, not permission to invent inputs or weaken fail-closed behaviour. See the [workstream index](workstreams/README.md), [remaining-work inventory](workstreams/REMAINING-WORK-v1alpha4.json) and [final acceptance record](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md).

## Current language boundary

The compiler release supports `connectionagent.io/v1alpha4`. The newer author-level Wait semantics are not part of that language version and are not implemented in v1.0.0. They require a separately versioned language and compiler target, aligned with the Host and Ingress contracts.

## Known implementation limits

Some positive Build paths for Evidence, Mutation/Reconcile and Retry remain fail-closed where the required external contracts or complete static checks are not available. The exact scope is documented in [Known Limitations](known-limitations-v1alpha4.md).

Real Owner/Deployment integration and 5GC qualification have not been established by the Compiler release. See the [acceptance record](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) for implementation evidence and the [local verification guide](local-verification-v1alpha4.md) for reproducible checks.

## References

- [CLI reference](cli-v1alpha4.md)
- [Documentation index](README.md)
- [Work history](WORK-history.md)
