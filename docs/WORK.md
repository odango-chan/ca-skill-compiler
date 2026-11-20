# CA Skill Compiler — Current State

- **Product:** CA Skill Compiler **v2.0.2 — RELEASED** ([release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.0.2), 2026-11-19).
- **Release source:** e98a7b26acabfa2a5e1587974ab9be5742e70bf7.
- **Language:** CA-SL connectionagent.io/v1alpha5 — **FROZEN / RELEASED**; the only supported language target.
- **Current compiler scope:** single Go core, native CLI and Go/WASM Worker; static validation, dependency linking, deterministic Artifact assembly and self-verification.
- **Product qualification:** **NOT_QUALIFIED** for real Owner / Deployment / Host / Runtime / 5GC / MAS execution. This does not invalidate the released Compiler's static acceptance.

## Current released baseline

- [v2.0.2 release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.0.2), published 2026-11-19.
- Release source commit: e98a7b26acabfa2a5e1587974ab9be5742e70bf7.
- Release assets include native CLI builds for Linux amd64, macOS amd64/arm64, Windows amd64, the Go/WASM module and SHA256SUMS.
- v2.0.2 retains the v1alpha5 BuildRequest, BuildResult, ToolCatalog / CapabilityContracts and Artifact API identities from v2.0.1. Its patch fixes producer output availability after a successful later Procedure stage.
- Previous release assets and tags remain immutable.

## Current language and architecture authority

- CA-SL v1alpha5 specification: [ca-skills release source](https://github.com/odango-chan/ca-skills/blob/main/docs/skill-language-spec.v1alpha5.zh-CN.md).
- IngressCatalog v1alpha1: [ca-skills design/release source](https://github.com/odango-chan/ca-skills/blob/main/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md).
- Compiler Artifact Contract: [compiler-artifact-v1alpha5.md](design/compiler-artifact-v1alpha5.md).
- Compiler Core: [compiler-core-v1alpha5.md](design/compiler-core-v1alpha5.md).
- Full local / release acceptance: [COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

## Status

| Area | Status |
|---|---|
| CA-SL v1alpha5 language target | FROZEN / RELEASED |
| Compiler v2.0.2 | RELEASED |
| Compiler-owned v1alpha5 static validation | Implemented and release-verified |
| Deterministic Artifact assembly / verification | Implemented |
| Native / WASM / real Chromium Worker parity | Release acceptance PASS |
| Real Owner / Deployment / Host / Runtime / 5GC / MAS qualification | NOT_QUALIFIED |
| Interface-only Development Build without Evidence / Deployment | **PROPOSED; NOT IMPLEMENTED** |
| Separate Capability Catalog and Deployment Grant | **PROPOSED; NOT IMPLEMENTED** |
| Package Publisher / Registry / Resolver | External responsibility; not a Compiler subsystem |

## Current MAS integration status

- Canonical Skill source: ca-skills/examples/skills/establish-pdu-session-mas/.
- The source defines 13 MAS Operation callables for the fixed policy-first profile.
- ca-client-tools PR #11 merged the source mapping; PR #16 merged the 25 candidate Schema ID / filename migration.
- ca-client-tools PR #14 remains an open Draft Typed Provider prototype; it is not a published trusted Owner package or production Provider release.
- ca-skills #71 tracks source-to-Owner contract closure.
- ca-agent-demo #215 tracks the actual MAS E2E; #216 tracks Temporal integration and generic Node-entry Wait conformance.
- Current MAS formal Build with Compiler v2.0.2 remains blocked by missing formal dependency / Owner / Evidence / Deployment closure. The next-generation interface-only build proposal is intended to remove Evidence and Deployment from the development Artifact prerequisite, not to claim production qualification.

## Maintenance rules

- Keep CA-SL v1alpha5 semantics and Compiler v2.0.2 contracts immutable.
- Do not introduce a second language target or compatibility layer.
- Compiler consumes exact dependency snapshots supplied by Studio / CI / callers; it does not own package discovery, download, version selection, Registry, Provider deployment, Qualification or Runtime dispatch.
- Compiler may verify supplied bytes and generate deterministic Artifact integrity data; this is not Package Registry or release ownership.
- Keep static language / interface proof separate from Assurance, Host Admission and real product qualification.
- New development Artifact behavior must use a versioned Build API / Artifact Contract; do not silently relax the v1alpha5 frozen Build path.

## Next design documents

- [CA-SL toolchain architecture — Compile / Link / Build / Admission](design/ca-toolchain-architecture-vnext.zh-CN.md)
- [Compiler next-stage evolution roadmap](design/compiler-evolution-roadmap-vnext.zh-CN.md)
