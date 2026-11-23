> **Product status:** Compiler v2.1.0 RELEASED. This document describes the frozen CA-SL v1alpha5 entry. The [Development v1alpha1](../../contracts/development/v1alpha1/README.md) API has a separate request/manifest and remains `NOT_ADMITTED`, using the same Compiler Core. [Release record](../releases/ca-skill-compiler-v2.1.0.md).

# Compiler Core and Public API — CA-SL v1alpha5

**Status:** implemented architecture record for the single supported language
target, `connectionagent.io/v1alpha5`.

**Language authority:** CA-SL v1alpha5 release commit
`0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5` (inherits the released v1alpha4
normative semantics) and the IngressCatalog v1alpha1 release commit
`9412be9894ef36acf846c757a5870dc0c48b33ac`.

## 1. One compiler core

For the **frozen v1alpha5 Build/Verify profile**, the Compiler has one API
identity set, one admission path and one deterministic artifact path. Since
**v2.1.0**, a separately versioned Development v1alpha1 Build/Verify profile
adds exact Interface Package/Lock linking and `NOT_ADMITTED` artifacts, while
reusing the **same Go Core and language semantic implementation**; it is not a
second CA-SL language target. Appendix L Wait passes compose with inherited
semantics on the same normalized YAML parse result and diagnostic stream.

```
Source bytes
  -> BuildRequest / trusted-context admission
  -> one normalized YAML AST per file
  -> v1alpha5 Procedure / node-entry Wait AST
  -> inherited semantic pipeline
       package + Contract + PredicateSet + Script admission
       dependency identity + immutable snapshot binding
       Owner / ToolCatalog / callable / CapabilityContracts linking
       Schema registry + $ref closure + assignability + CEL
       Predicate + Evidence Descriptor linking + Script ABI
       Procedure graph (reachability, dominance, routes, terminals, Goal)
       Evidence / Mutation / Reconcile / retry assurance
       Wait trigger / correlation / wait.output / response boundary
  -> deterministic artifact (manifest + compiled files + self-verification)
```

`validateSkillPackage` in `inherited_semantics.go` is the single
implementation of the inherited Skill semantics. The v1alpha5 Wait passes are
additional stages in the same pipeline, not a second language implementation.

## 2. Public API

| Concern | Location |
|---|---|
| Public DTOs (`BuildRequest`, `BuildResult`, `Diagnostic`, `DependencyInspection`, inspection rows) | `internal/compiler/api.go` |
| Strict JSON transport decoder (`DecodeBuildRequestJSON`) | `internal/compiler/api.go` |
| Single-version admission (exact `apiVersion`, exact `languageTarget`, paths, limits, trust bindings) | `internal/compiler/admission.go` |
| Public entries: `Check`, `Build`, `InspectDependencies`, `VerifyBuildResult` | `internal/compiler/api.go` |
| Inherited semantic pipeline | `internal/compiler/inherited_semantics.go` |
| Internal semantic model | version-neutral `SkillAST` / linking / `artifactInput` types |
| Contracts | `contracts/v1alpha5/` |

The public DTOs carry no parser, linker or artifact-version types. There is no
version dispatch: an unsupported `apiVersion` or `languageTarget` fails closed
during admission. The retired v1alpha4 API identity, CLI target and artifact
generator are removed; `v1alpha4` remains only as the normative inheritance
source and as historical release documentation.

## 3. Target and release boundary

- `connectionagent.io/v1alpha5` is the only supported language target. All
  development and acceptance target v1alpha5.
- Compiler product version (v2.0.0) is independent of the language version and
  of the BuildRequest/BuildResult/DependencyInspection API versions.
- The published v1.0.0 tag, release assets and release notes remain immutable
  historical records.
- Compiler static validation is not product qualification. Owner / Deployment /
  Integration Adapter / Host / Runtime / 5GC / MAS qualification remains
  NOT_QUALIFIED and is owned outside this Compiler.

## 4. Determinism and boundaries

- Native, WASM and the Chromium Worker load the same Go core; parity is
  asserted over Check/Build/Inspect projections and artifact bytes.
- The Compiler performs static parsing, admission, linking, semantic/type
  validation and deterministic artifact generation only. It performs no
  Tool/Observation execution, no SBI transport, no dependency acquisition, no
  network access and no Runtime dispatch.
- The artifact manifest declares `validationScope.status =
  FULL_LANGUAGE_VALIDATION` for the Compiler-owned static scope; `notValidated`
  retains only Host/Runtime integration and real Owner/product qualification.
