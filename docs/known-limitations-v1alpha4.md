> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Known Limitations and External Dependencies v1alpha4

**Status:** Current boundary record for Compiler v1.0.0. The Compiler is locally verified for the implemented v1alpha4 static paths; product qualification and the positive paths listed below remain separate.

The Compiler fails closed rather than inventing missing trusted formats or treating partial static analysis as complete compilation.

## 1. Unfrozen external contract formats (BLOCKED_EXTERNAL_CONTRACT)

| Item | Why it is blocked | Current behaviour |
|---|---|---|
| Evidence Descriptor physical format (WP7 positive path, E73) | No versioned Owner Evidence Descriptor publication exists in the frozen authority | Requirement binding, independence structure and static satisfiability are checked; descriptor content and Producer/Scope/Correlation/Freshness trusted relations are not accepted for a successful Build |
| Identity-input trusted publication provenance | No frozen trusted publication format | Identity provenance checks fail closed; no trust root is inferred from field names |
| Cross-snapshot Reconcile association | Requires a separately versioned trusted association contract | Dedicated Reconcile READ and same-snapshot Schema/Owner semantics are checked; cross-snapshot association is not accepted |

Eight E-cases have disposition `BLOCKED_EXTERNAL_CONTRACT`: E38, E46, E68, E73, E87, E101, E102 and E109. See [E-coverage](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md#4-e-coverage) and [the machine-readable inventory](../tests/conformance/v1alpha4/e-coverage.json).

## 2. Normative mapping questions (BLOCKED_NORMATIVE_MAPPING)

E71 and E74 require `PREDICATE_EVIDENCE_REQUIRED`, while the unambiguous implemented trigger emits `PREDICATE_EVIDENCE_UNSATISFIABLE`. The frozen text does not distinguish the two triggers precisely enough to implement without guessing. Tracked for language-owner clarification in [ca-skills #59](https://github.com/odango-chan/ca-skills/issues/59).

## 3. Host / Runtime responsibilities (NOT_APPLICABLE_TO_COMPILER)

39 E-cases are owned exclusively by Host or Runtime. Examples include:

- Invocation admission, dispatch and authorization revalidation;
- Runtime retry admission, event ingress, Wait lifecycle, resource responsibility and Trace serialization;
- Tool invocation, Python execution and Observation calls;
- MAS runtime behaviour, deployment authorization servers and live network operations.

Compiler consumes trusted metadata and produces static diagnostics/artifacts. A successful Build is **not** an authorization token. Host/Runtime must independently re-verify deployment authorization, dependency identity, Owner provenance, Scope and the current Invocation.

## 4. Qualification (WP13)

Real 5GC/SMF and MAS end-to-end qualification is **NOT_QUALIFIED**:

- no confirmed real v1alpha4 Owner/Deployment publication set was available for qualification;
- MAS was not runnable as an executable baseline.

Synthetic fixtures in `tests/conformance/v1alpha4/fixtures` and `web/parity-*` prove Compiler behaviour only; they are not product-conformance evidence. WP13 does not block independent Compiler implementation and is not counted as PASS.

## 5. CI and release boundary

- CA Skill Compiler v1.0.0 is published with Native CLI binaries, the Go/WASM module and SHA-256 checksums.
- Release acceptance is `LOCAL_ACCEPTANCE_PASS`; the executed local commands are recorded in [Local Verification](local-verification-v1alpha4.md).
- GitHub Actions was unavailable for the release run and is **not** reported as PASS. Its unavailability does not invalidate the recorded local acceptance.
- The Compiler does not publish to a package registry, deploy artifacts, sign runtime authorization, or perform product qualification. The CLI writes a verified artifact directory.

## 6. CLI scope limits

- In project mode, `trusted.json` is a local convenience description, not a frozen wire contract. Host integrations must use explicit BuildRequest JSON (`--request`) or construct trusted context from their authenticated trust store.
- The CLI does not resolve, download or refresh dependencies, and does not maintain a lock file; dependency selection and lock lifecycle belong to Studio/Host.
- The CLI does not start a server, database, package manager or Runtime dispatcher.

## 7. Feature-scoped Build gates

The current implementation performs static sub-checks for these areas, but deliberately rejects a successful Build while the complete positive-path obligations remain unfinished:

| Feature | Implemented static checks | Why Build remains fail-closed |
|---|---|---|
| Evidence | Exact Guarantee references, requirement parameter binding, independence structure and static satisfiability | The physical Owner Evidence Descriptor and trusted Producer/Scope/Correlation/Freshness relations are not frozen |
| Mutation / Reconcile | Receipt and commit-proof identity, same-snapshot Schema/path resolution, dedicated same-Owner READ checks and route eligibility | The complete Mutation/Reconcile positive path is not implemented; cross-snapshot association lacks a versioned trusted contract |
| Retry | retryOf reference shape, Mutation family constraints and NOT_COMMITTED-only graph reachability | Full retry contract compatibility and path-coverage analysis are not implemented; Runtime retry admission remains Host/Runtime-owned |

These are intentionally conservative gates in `internal/compiler/artifact_pipeline.go`. Do not remove them solely to make an example or synthetic fixture build. Complete the missing frozen static obligations and obtain the required external publications before enabling the corresponding positive Build path.

## 8. Frozen C_STATIC identities not yet emitted

Three frozen Appendix H identities are not emitted by the active v1alpha4 pipeline. Each depends on an external contract, deployment Scope algebra or an internal-only invariant that is not available:

| Identity | Reason |
|---|---|
| `CALLABLE_ARGUMENT_PROVENANCE_INVALID` | Unfrozen identity-input trusted publication format |
| `CAPABILITY_SCOPE_EMPTY_INTERSECTION` | Deployment Scope algebra is absent from frozen CapabilityContracts |
| `DUPLICATE_ID` | Internal-only IR uniqueness invariant; author-visible duplicates use `EXPORT_ID_AMBIGUOUS` / `DEPENDENCY_DUPLICATE` |

`CONTEXT_SCHEMA_MISMATCH`, `PREDICATE_EVIDENCE_UNSATISFIABLE`, `PREDICATE_EVIDENCE_REQUIRED` (the unambiguous trigger) and `RECONCILE_OUTPUT_UNAVAILABLE` are emitted by the current pipeline. `PREDICATE_EVIDENCE_REQUIRED` remains subject to the E71/E74 normative ambiguity in section 2.

This list is reviewed by `internal/compiler/diagnostic_inventory_test.go` and the acceptance status document. It is not permission to invent new identities.
