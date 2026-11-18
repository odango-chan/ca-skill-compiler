> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# CA-SL v1alpha4 Compiler Goal — Maintainer Resume Decision (2026-11-16)

**Status:** GOAL RESUMED FOR INDEPENDENT IMPLEMENTATION · SCOPED NORMATIVE MAPPING BLOCKERS REMAIN  
**Applies to:** `odango-chan/ca-skill-compiler`, current stacked PR chain #11–#23.  
**Language authority:** `odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d` (blob `b70064c5737b8ee2b3d1435fe496357f44b42773`).  
**Language-owner errata request:** [ca-skills #59](https://github.com/odango-chan/ca-skills/issues/59).

## 1. Maintainer decisions

1. **Continue CA-SL v1alpha4.** The root `AGENTS.md` v0.1/v0.2 language authority was obsolete; it is corrected on the WP6 branch. The old text is archived under `docs/archive/AGENTS-v0.1.md`. Do not downgrade the implementation, add compatibility or use the archived rules.
2. **Retain all existing v1alpha4 PRs #11–#23.** Keep the stacked history, tests and code. No closing, repurposing, recreating or merging PRs is authorized by this clarification.
3. **Continue one long-running Compiler Goal, but execute bounded slices.** WP9 Python CST/Script ABI, WP10 Artifact/JCS, WP11 Native/WASM/Chromium parity, WP12 Studio integration and independent WP7/WP8 work may continue without waiting for new user prompts. Create subsequent Draft PRs stacked on the verified predecessor, and keep each change reviewable.
4. **Do not invent external Owner formats.** Missing Evidence Descriptor physical publication, identity-input provenance publication, cross-snapshot Reconcile association publication and real v1alpha4 5GC Owner/Deployment releases remain fail-closed / NOT_QUALIFIED. These block only those positive integration/qualification assertions, not independent compiler work.
5. **Local validation is the selected execution evidence for this run.** Go 1.27.4, test/vet/Native/WASM build/diff checks; actually run Chromium Worker parity when applicable. GitHub Actions is not a mandatory success gate and must not be claimed PASS unless actually run and observed.
6. **The existing WP6 branch is the continuation baseline**, `feat/compiler-v1alpha4-wp6-sourcepath-20261116`, PR #23. Check the current head before creating the next stacked branch; do not assume the previous reported `a3264e2` remains the latest head after this instruction correction.

## 2. Diagnostic disposition without changing frozen semantics

Only exact Appendix H `(Code,Phase)` identities are legal for Compiler output.

| Invalid input | Decision | Status |
|---|---|---|
| Nested YAML object/array used as an `Expression` in `arguments` | Reject with `SOURCE_PATH_INVALID / C_STATIC` under frozen §13.2. Do not reject permitted outer `arguments: {}` / zero-arity `arguments: []`. | **RESOLVED FOR IMPLEMENTATION** |
| Imported Observation/READ Owner contract publishes illegal `ownerOutcomes` | F.1 forbids it. `ROUTE_OUTCOME_UNKNOWN` concerns author route keys and `EFFECT_MISMATCH` concerns Tool vs Capability Effect, so do not misuse them. The exact H diagnostic is not clear. | **BLOCKED_NORMATIVE_MAPPING**, ca-skills #59 |
| `evidence.requires` has an empty list or duplicate guarantee ID | §17.1 forbids it. Do not treat this as `PREDICATE_EVIDENCE_REQUIRED` (which means a mandatory guarantee declaration was omitted). A precise structural-invalid code has not been confirmed. | **BLOCKED_NORMATIVE_MAPPING**, ca-skills #59 |
| Unsupported `Predicate.evaluator.type` discriminator | §14 allows `cel` and `script` only. Do not arbitrarily classify unknown discriminators as invalid CEL expressions or Script ABI failures. | **BLOCKED_NORMATIVE_MAPPING**, ca-skills #59 |

### Handling a scoped blocked fixture

- Never silently accept an invalid source/Owner contract; never mark it PASS or QUALIFIED.
- Never introduce a novel diagnostic or assert a fake exact mapping.
- Record the exact fixture and expected invalidity as `BLOCKED_NORMATIVE_MAPPING`, with a link to ca-skills #59.
- Do not bypass the whole invalidity check by weakening schema validation or skipping an existing safety guard.
- Implement all independent paths and other conformance cases; revisit only the blocked diagnostic mapping after the language owner publishes an approved clarification.
- If the blocked case prevents a full positive Build, report that bounded limitation. It is not permission to fabricate Owner publications.

These statements do not amend the frozen CA-SL text; only the language owner may publish an erratum or revised language version.

## 3. Resume sequence

1. Verify all PR #11–#23 bases and the latest PR #23 branch head; preserve the history.
2. Confirm current source baseline with local commands before changing code. Fix only proven compiler or test defects within the appropriate bounded slice; do not downgrade versions.
3. Continue **WP9** (`gositter` / Tree-sitter Python CST, Procedure and Predicate Script ABI, Derived Output Schema). Never execute author Python.
4. Continue **WP10** (deterministic artifact tree, manifest, JCS and exact output bytes).
5. Complete independent **WP8** mutation/reconcile static checks and **WP7** Evidence checks while missing external publication paths remain fail-closed.
6. Continue **WP11** parity; a Go/WASM build is not Chromium Worker runtime parity.
7. Continue **WP12** Studio integration; no duplicated parser or linker.
8. Close remaining E01–E120 independent variants and Appendix H diagnostics with executable evidence; retain blocked or Host/Runtime/Owner-owned cases under their accurate labels.
9. Keep **WP13** real 5GC qualification NOT_QUALIFIED until real, versioned and trusted Owner/Deployment publications and live environment evidence exist.

## 4. Required reporting

For each WP/PR report:
- starting commit/base and final commit;
- exact changed files;
- commands, exit codes and actual evidence;
- E variants and H diagnostics exercised;
- independent work completed versus specifically blocked cases;
- Draft PR URL and stack base;
- final status `PASS`, `FAIL`, `NOT_RUN`, `NOT_APPLICABLE`, or a precise `BLOCKED_*` reason.

Do not end the whole Goal just because three diagnostic-mapping fixtures await errata or real Owner publications remain unavailable. Stop only the affected branch, then proceed with independent work. Do not merge PRs or alter `ca-skills` in the Compiler agent run.
