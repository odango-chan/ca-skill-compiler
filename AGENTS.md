# ca-skill-compiler · Active Harness Instructions

**Active target: CA-SL v1alpha4 (breaking-only).** The former v0.1 / CA-SL v0.2 instructions are archived in `docs/archive/AGENTS-v0.1.md` for history and are not active development authority.

## 1. Immutable authority and implementation scope

Follow, in order:
1. The **frozen CA-SL v1alpha4 single-file specification**, `odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`, file `docs/skill-language-spec.v1alpha4.zh-CN.md`, blob `b70064c5737b8ee2b3d1435fe496357f44b42773`.
2. Frozen Compiler-owned v1alpha4 implementation contracts in `docs/design/*-v1alpha4.md`, machine schemas in `contracts/v1alpha4/`, and the OSS dependency-selection ADR.
3. `docs/workstreams/conformance-v1alpha4.md`, `tests/conformance/v1alpha4/case-index.json`, and `tests/conformance/v1alpha4/diagnostic-index.json`.
4. Current bounded WP task documents and active implementation/PR evidence. Older status notes are historical snapshots, not proof of present progress.

Never select the v0.1 / v0.2 implementation target because historical documents, comments, or examples mention it. Do **not** add compatibility, translation or fallback behavior. If active authority appears contradictory, check the exact pinned files and report a real unresolved conflict; do not let outdated repository instructions cause a project-wide BLOCKED status.

## 2. Existing work and continuation

The ongoing v1alpha4 implementation uses stacked PRs #11 through #23. In particular:
- PR #13: WP1 BuildRequest and digest.
- PR #14: WP2 normalized YAML parser/AST.
- PRs #15–#18: WP3A–WP3D dependency/Owner/Capability linking.
- PRs #19–#21: WP4 Schema/F.5/CEL.
- PR #22: WP5 Procedure graph.
- PR #23: WP6 SourcePath, producer availability/dominance, argument typing.

Do not discard, close, silently replace or reimplement these slices. First inspect current branch/PR state, then reuse working code and continue forward. Keep changes small, auditable and stacked in dependency order. Do not merge or force-push without explicit approval.

The ongoing Goal is **complete Compiler v1alpha4 implementation**, not only WP3A/WP6. Continue with remaining independent work (especially WP9 Python CST/Script ABI, WP10 Artifact/JCS, WP11 Native/WASM/Chromium parity, WP12 Studio integration, and the implementable WP7/WP8 static pieces) without repeatedly requesting a new Goal for each WP. Existing bounded WP files remain relevant to their own slices; they do not limit authorization for later WPs under the project-wide Goal. Do not combine unrelated changes in a single PR.

## 3. Compiler ownership and trust boundaries

Use one deterministic stateless Go Core for Native and Go/WASM. Consume author Skill source, immutable dependency snapshots, and trusted build context. No database, package manager, network dependency acquisition, Runtime dispatch, Python execution, live 5GC mutation, release or qualification implementation belongs in Compiler.

Use the already approved OSS dependencies and the single WP2 YAML AST; do not implement parallel parsers, digest algorithms or Schema/CEL engines.

No actual Evidence Owner descriptor protocol, identity-input trusted publication format, or cross-snapshot Reconcile association protocol has been supplied as a frozen interoperable release. Do not invent these formats. Implement all independent static checks and fail-closed gates; keep missing-publication positive paths NOT_QUALIFIED. External qualification and real 5GC E2E are WP13 and must not block independent Compiler implementation.

## 4. Frozen diagnostics

Every emitted (Code, Phase) must be justified by Appendix H of the pinned CA-SL spec; source locations must remain grounded in the original source. Never invent diagnostic codes or use an existing code for a semantically unrelated failure.

See `docs/workstreams/v1alpha4-resume-decision-20261116.md` for the bounded diagnostic disposition. Specifically, inline object/array values in an Expression position are invalid under §13.2 and should use `SOURCE_PATH_INVALID / C_STATIC`; do not prohibit the legal empty outer `arguments: {}` or `arguments: []` containers.

If a malformed imported contract or source shape has no unambiguous frozen diagnostic, do not silently accept it, falsify a PASS, or guess a new diagnostic. Record the specific blocked fixture/branch, request a normative clarification for that small issue, and continue unrelated WPs.

## 5. Testing, evidence and GitHub

For this run the maintainer chose **local commands as execution evidence**; GitHub Actions is not a required gate and must not be reported as passing if unused. Use the Go 1.27.4 toolchain and exact tasks' build commands. At minimum run Go tests, vet, CGo-free Native build, Go/WASM build and `git diff --check`; run an actual Chromium Worker test once implementing browser parity. Record accurate PASS/FAIL/BLOCKED/NOT_RUN outcomes with command outputs and commit IDs.

Work on a new branch from the current reviewed dependency head for each subsequent slice; create a Draft PR targeting its immediate parent branch. Do not push directly to main, merge PRs, change the language-owner repository, or perform any live network operation. Advance automatically when independent tasks remain. Reserve human decisions for genuinely unresolved normative/security questions and report them precisely.
