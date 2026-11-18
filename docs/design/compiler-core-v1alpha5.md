# Compiler Core and Public API — CA-SL v1alpha5

**Status:** implemented architecture record for the active `connectionagent.io/v1alpha5` target.

**Language authority:** CA-SL v1alpha5 release commit `0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5` (inherits the released v1alpha4 normative semantics) and the IngressCatalog v1alpha1 release commit `9412be9894ef36acf846c757a5870dc0c48b33ac`.

## 1. One compiler core

The Compiler has exactly one static pipeline and one deterministic artifact path. The v1alpha5 target does not use a separate Wait pipeline: Appendix L Node-entry Wait passes compose with the inherited semantic pipeline on the same normalized YAML parse result and the same `[]compiler.Diagnostic` stream.

```
Source bytes
  -> request / trusted-context admission
  -> one normalized YAML AST per file
  -> v1alpha5 Procedure / Node-entry Wait AST
  -> inherited semantic pipeline
       package + Contract + PredicateSet + Script admission
       dependency identity + immutable snapshot binding
       Owner / ToolCatalog / callable / CapabilityContracts linking
       Schema registry + $ref closure + assignability + CEL
       Predicate + Evidence Descriptor linking + Script ABI
       Procedure graph (reachability, dominance, routes, terminals, Goal)
       Evidence / Mutation / Reconcile / retry assurance
       Wait trigger / correlation / wait.output / response boundary
  -> deterministic Artifact (manifest + compiled files + self-verification)
```

`validateInheritedSkillPackage` in `inherited_semantics.go` is the single implementation of the inherited Skill semantics. Both the historical v1alpha4 dispatcher and the active v1alpha5 dispatcher call it; no semantic rule is implemented twice.

## 2. Public API separation

| Concern | Location |
|---|---|
| Target-neutral public DTOs (`BuildRequest`, `BuildResult`, `Diagnostic`, `DependencyInspection`, inspection rows) | `internal/compiler/api.go` |
| Strict JSON transport decoder (`DecodeBuildRequestJSON`) and language-target dispatch | `internal/compiler/api.go` |
| v1alpha5 admission + public entries (`CheckV1Alpha5`, `BuildV1Alpha5`, `InspectV1Alpha5Dependencies`) | `internal/compiler/api_v1alpha5.go` |
| v1alpha4 historical-target envelope admission only | `internal/compiler/api_v1alpha4_target.go` |
| Internal semantic model | version-neutral `V1Alpha4*` AST/link/artifact types shared by both targets |
| Artifact contracts | `contracts/v1alpha5/` (active), `contracts/v1alpha4/` (historical) |

The public DTOs carry no parser, linker or artifact-version types. The internal semantic model does not depend on the public wire API version: the same request struct, AST and pipeline serve both target dispatches, and v1alpha5-only trusted-context fields are rejected by the frozen historical v1alpha4 envelope validation.

## 3. Target boundary

- `connectionagent.io/v1alpha5` is the active language target. All new development and acceptance target v1alpha5.
- `connectionagent.io/v1alpha4` remains a frozen historical dispatch target for release traceability. It is not the active development target and it receives no new language semantics. Published v1.0.0 tags, artifacts and release notes are immutable and are not rewritten.
- Compiler static validation is not product qualification. Owner / Deployment / Integration Adapter / Host / Runtime / 5GC / MAS qualification remains NOT_QUALIFIED and is owned outside this Compiler.

## 4. Determinism and boundaries

- Native, WASM and the Chromium Worker load the same Go core; parity is asserted over Check/Build/Inspect projections and artifact bytes.
- Compiler performs static parsing, admission, linking, semantic/type validation and deterministic artifact generation only. It performs no Tool/Observation execution, no SBI transport, no dependency acquisition, no network access and no Runtime dispatch.
- The artifact manifest declares `validationScope.status = FULL_LANGUAGE_VALIDATION` for the Compiler-owned static scope; `notValidated` retains only Host/Runtime integration and real Owner/product qualification.
