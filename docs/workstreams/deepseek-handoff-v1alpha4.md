# DeepSeek Handoff — CA Skill Compiler v1alpha4

**Status:** READY FOR FIRST BOUNDED TASK  
**Prepared:** 2026-11-16  
**First task:** `WP3A-DEPENDENCY-IDENTITY-001`  
**Model profile:** DeepSeek V4 Flash / Low reasoning, as selected by the maintainer.

This is an execution handoff, not a Compiler conformance declaration. The only authorized first task is WP3A in [the task specification](wp3a-dependency-identity-v1alpha4.md). Do not start WP3B or any later work package in the same run.

## 1. What to give DeepSeek

| Item | Exact value |
|---|---|
| Repository | `odango-chan/ca-skill-compiler` |
| Assigned branch | `feat/compiler-v1alpha4-wp3a-dependency-identity-20261116` |
| Branch base | `feat/compiler-v1alpha4-wp2-yaml-ast-20261116` |
| Base commit | `55ba5ebc37e9d54b6ee21e547dae188ca17785c9` |
| Task | `WP3A-DEPENDENCY-IDENTITY-001` |
| Task instructions | `docs/workstreams/wp3a-dependency-identity-v1alpha4.md` |
| Fixture plan | `tests/conformance/v1alpha4/wp3a-fixture-plan.json` |
| Normative language source | `odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d` |
| Normative specification blob | `b70064c5737b8ee2b3d1435fe496357f44b42773` |

The assigned branch was created from the clean WP2 branch at the exact base commit above. It intentionally does **not** include the separate exploratory WP3 branch's unreviewed linker commit.

## 2. Current PR stack and implementation boundary

As checked on 2026-11-16:

| PR | Scope | State | Dependency |
|---|---|---|---|
| [#11](https://github.com/odango-chan/ca-skill-compiler/pull/11) | v1alpha4 Source Admission | Open / Draft | Existing v0.1 implementation base |
| [#12](https://github.com/odango-chan/ca-skill-compiler/pull/12) | Frozen implementation contracts, ADR and Harness profile | Open | #11 |
| [#13](https://github.com/odango-chan/ca-skill-compiler/pull/13) | WP1 BuildRequest / digest | Open | #12 |
| [#14](https://github.com/odango-chan/ca-skill-compiler/pull/14) | WP2 normalized YAML parser / AST | Open | #13 |

None of #11–#14 was merged at handoff preparation time. #14 reports Go tests, vet, Native build and Go/WASM build PASS in its PR description; this is not full Compiler conformance.

A separate branch, `feat/compiler-v1alpha4-wp3-linker-20261116`, exists and is one commit ahead of WP2 with an added `internal/compiler/dependency_linker.go`. That draft is not the approved task base: it is not integrated into the public Check/Build path, has no corresponding test files in its one-file branch delta, and the v1alpha4 path still contains the earlier fail-closed placeholder. Do not cherry-pick it or assume it is tested. The assigned WP3A branch starts cleanly from WP2 so the task can be implemented and reviewed as a bounded slice.

## 3. First task only: WP3A

WP3A implements the trusted identity boundary for dependency snapshots:

- recompute snapshot digests from original bytes;
- bind each used Contract Dependency Alias to exactly one supplied immutable snapshot;
- verify the trusted binding for the exact `dependencyId + measured digest`;
- compare the trusted Stable Package ID exactly with `Contract.dependencies`;
- preserve selected-Skill closure boundaries and deterministic ordering;
- emit only the frozen diagnostics listed in the task and fixture plan.

WP3A does **not** implement ToolCatalog contract validation, export Owner attribution, CapabilityContracts, Scope intersection, Schema linking, Evidence descriptors, Procedure graph linking or Script analysis.

The compiler must continue to fail closed for later, unimplemented stages. A passing WP3A identity check must not make Build succeed or qualify an Owner.

## 4. Required reading order

1. This handoff.
2. [WP3A task specification](wp3a-dependency-identity-v1alpha4.md).
3. [WP3A fixture plan](../../tests/conformance/v1alpha4/wp3a-fixture-plan.json).
4. [Compiler Contract](../design/compiler-contract-v1alpha4.md).
5. [BuildRequest contract](../design/build-request-v1alpha4.md).
6. [Compiler Output API](../design/compiler-output-contract-v1alpha4.md).
7. [Procedure Linker contract](../design/procedure-link-v1alpha4.md), only for the alias/reference locations consumed by WP3A.
8. Pinned CA-SL v1alpha4 specification, especially §§24, 34.0–34.0.2, F.0–F.2 and Appendix E/K/H.

Do not use floating `main`, historical v0/v0.1/v0.2 docs, or a different CA-SL commit as implementation authority.

## 5. Run instructions

- Work only on the assigned branch.
- First run the baseline commands from the task document before editing.
- Keep one task per branch/PR. Do not start WP3B.
- Do not modify CA-SL, ca-skills, Host, Runtime, Studio or real 5GC.
- Do not use production credentials or perform live network mutations.
- Do not merge the PR. Submit a Draft PR targeting `feat/compiler-v1alpha4-wp2-yaml-ast-20261116`.
- If a required contract or expected diagnostic is ambiguous, stop and report the exact conflict. Do not invent a schema, diagnostic or fallback.

## 6. Copy-ready task prompt

> Work only on `WP3A-DEPENDENCY-IDENTITY-001` in `docs/workstreams/wp3a-dependency-identity-v1alpha4.md`.
>
> Repository: `odango-chan/ca-skill-compiler`  
> Branch: `feat/compiler-v1alpha4-wp3a-dependency-identity-20261116`  
> Normative CA-SL commit: `odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`
>
> Read the task document and its machine-readable fixture plan before changing code. Run the baseline commands first. Implement only the dependency snapshot identity/trust-binding slice, add every WP3A fixture and exact (Code, Phase) assertion, and preserve fail-closed behavior for all later stages. Do not implement ToolCatalog, CapabilityContracts, Schema/CEL, Evidence, Procedure linking or Artifact work. Do not change language semantics, API versions, diagnostics, dependencies or unrelated paths. If the baseline fails or a contract is ambiguous, stop and report evidence instead of guessing.
>
> Before finishing, run every required command, inspect the complete diff, update the task run record, and open/update a Draft PR against `feat/compiler-v1alpha4-wp2-yaml-ast-20261116`. Do not merge.

## 7. Handoff status

The documentation, task envelope, first-task fixture plan and CI branch filters are prepared on the assigned branch. Code implementation has **not** been performed by this handoff change. The first DeepSeek run remains subject to the baseline test result and its own CI evidence.
