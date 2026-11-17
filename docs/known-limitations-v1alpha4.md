# Known Limitations and External Dependencies v1alpha4

**Status:** Truthful boundary record. Items here do **not** mean an implementable Compiler
stage is unfinished; they are either (a) branches whose external trusted publication
format is not frozen, (b) Host/Runtime responsibilities outside the Compiler, or (c)
Qualification work, which this repository explicitly does not claim.

The Compiler is designed to **fail closed** on every item in section 1 and 2: it reports a
frozen diagnostic instead of inferring an identity, inventing a format or emitting a
partial success.

## 1. Unfrozen external contract formats (BLOCKED_EXTERNAL_CONTRACT)

| Item | Why it is blocked | Compiler behaviour |
|---|---|---|
| Evidence Descriptor physical format (WP7 positive path, E73) | No versioned Owner Evidence Descriptor publication exists in the frozen authority; inventing one is prohibited | Evidence guarantee references, requirement binding, independence structure and static satisfiability are checked; descriptor content/producer/scope/correlation/freshness relations fail closed |
| Identity-input trusted publication provenance | No frozen trusted publication format | `SOURCE_PATH_INVALID` / provenance checks fail closed; no synthetic trust root is created |
| Cross-snapshot Reconcile association | Requires a separately versioned trusted association contract | Dedicated Reconcile READ and same-snapshot semantics are checked; cross-snapshot association fails closed |

Eight E-cases are recorded with disposition `BLOCKED_EXTERNAL_CONTRACT`:
E38, E46, E68, E73, E87, E101, E102, E109. See
[E-coverage](COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md#4-e-coverage) and
[`tests/conformance/v1alpha4/e-coverage.json`](../tests/conformance/v1alpha4/e-coverage.json).

## 2. Normative mapping questions (BLOCKED_NORMATIVE_MAPPING)

E71 and E74 require `PREDICATE_EVIDENCE_REQUIRED`, while the only unambiguous implemented
trigger emits `PREDICATE_EVIDENCE_UNSATISFIABLE`. The frozen text does not distinguish the
two triggers precisely enough to implement without guessing. Tracked for language-owner
clarification in [ca-skills #59](https://github.com/odango-chan/ca-skills/issues/59).

## 3. Host / Runtime responsibilities (N/A to Compiler)

43 E-cases are `NOT_APPLICABLE_TO_COMPILER` because their owner in
`tests/conformance/v1alpha4/case-index.json` is the Host or Runtime, for example:

- live Invocation, dispatch, admission and authorization decisions;
- Runtime retry admission/resource responsibility and Trace serialization;
- Tool execution, Python execution and Observation calls;
- MAS runtime behaviour, deployment authorization servers and release processes.

The Compiler consumes trusted metadata and produces static diagnostics/artifacts; a
successful build is **not** an authorization token. Runtime must independently re-verify
deployment authorization, dependency identity, Owner provenance, Scope and the current
Invocation.

## 4. Qualification (WP13)

Real 5GC/SMF and MAS end-to-end qualification is **NOT_QUALIFIED**:

- no confirmed real v1alpha4 Owner/Deployment publications were available;
- MAS was not runnable as an executable baseline.

Synthetic fixtures in `tests/conformance/v1alpha4/fixtures` and `web/parity-*` are evidence
of Compiler behaviour, never evidence of product conformance. WP13 does not block Compiler
implementation and is not counted as PASS.

## 5. CI and release boundary

- GitHub Actions is unavailable for this run (billing block). Acceptance evidence is the
  local command set in [Local Verification](local-verification-v1alpha4.md); CI is never
  reported as PASS.
- No release artifact, package registry publication, deployment or version-management
  system is produced or required by this round.
- The reference CLI writes an artifact directory; it does not publish or sign it.

## 6. CLI scope limits

- In project mode, `trusted.json` is a local convenience description; it is not a frozen
  wire contract. Host integrations must use explicit BuildRequest JSON (`--request`) or
  construct the trusted context from their authenticated trust store.
- The CLI does not resolve, download or refresh dependencies, and does not maintain a lock
  file; dependency selection/lock lifecycle belongs to Studio/Host, not the Compiler.
- The CLI does not start a server, database, package manager or Runtime dispatcher.

## 7. Repository cleanup known item

The pre-v1alpha4 analysis pipeline files (`api.go` `analyze`, `ast.go`, `bindings.go`,
`library.go`, `skills.go`, `scripts.go`, `skill_markdown.go`, `yaml.go` analysis method,
`schema_refs.go`) are isolated behind the `legacy_v01` build tag: they are not compiled into
the active package and are unreachable from the public v1alpha4 APIs. They contain obsolete
v0.1 diagnostic identities and remain the only reason the untagged build is clean of
legacy-only identities. Removing them from history is a recorded cleanup task, not a
functional gap.

## 8. Frozen C_STATIC identities not yet emitted

Three frozen Appendix H identities are not emitted by the active v1alpha4 pipeline. Each
depends on an external contract, deployment Scope algebra or internal-only invariant that
is not available, so the branch fails closed with a different frozen identity rather than
inventing behaviour:

| Identity | Reason |
|---|---|
| `CALLABLE_ARGUMENT_PROVENANCE_INVALID` | unfrozen identity-input trusted publication format |
| `CAPABILITY_SCOPE_EMPTY_INTERSECTION` | deployment Scope algebra absent from frozen CapabilityContracts |
| `DUPLICATE_ID` | internal-only IR uniqueness invariant; an author-visible duplicate uses `EXPORT_ID_AMBIGUOUS` / `DEPENDENCY_DUPLICATE` and the internal invariant is unit-tested as not author-triggerable |

`CONTEXT_SCHEMA_MISMATCH`, `PREDICATE_EVIDENCE_UNSATISFIABLE`, `PREDICATE_EVIDENCE_REQUIRED`
(the unambiguous trigger) and `RECONCILE_OUTPUT_UNAVAILABLE` **are** emitted by the current
pipeline. `PREDICATE_EVIDENCE_REQUIRED` remains subject to the E71/E74 normative ambiguity
described in section 2.

This list is reviewed by `internal/compiler/diagnostic_inventory_test.go` and the
acceptance status document. It is not an excuse to invent new identities.
