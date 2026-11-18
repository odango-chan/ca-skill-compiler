# CA Skill Compiler — v1alpha5 compiler-owned contracts

**Status: FROZEN** for the CA Skill Compiler v2.0.0 product line.

**Language authority:** CA-SL v1alpha5 is FROZEN / RELEASED by ca-skills PR #66.
**Ingress authority:** IngressCatalog v1alpha1 and its diagnostic registry are FROZEN / RELEASED by ca-skills PR #62.

This directory contains the Compiler-owned machine contracts for the single
supported language target, `connectionagent.io/v1alpha5`. It does not
republish or override the language or Ingress logical contracts.

Normative sources:

- [CA-SL v1alpha5](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md)
- [IngressCatalog v1alpha1 logical contract](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md)
- [Ingress diagnostic registry v1alpha1](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/ingress-diagnostic-registry-v1alpha1.md)

## Contracts

| File | Contract |
|---|---|
| `build-request.schema.json` | the only accepted `BuildRequest` API: exact API/language identity, source files, immutable dependency snapshots, dependency/Owner bindings, deployment bundles and trusted Ingress/Schema artifact bindings |
| `build-result.schema.json` | `BuildResult` envelope: diagnostics, dependency closure and output files; a failed result carries no output |
| `dependency-inspection.schema.json` | read-only projection: inherited callables/Schemas plus the independent `ingressOperations` inventory |
| `artifact-manifest.schema.json` | the deterministic Build artifact manifest emitted as canonical RFC 8785 JCS |
| `dependency-snapshot-digest-vectors.json` | golden vectors for the inherited cross-language dependency snapshot digest |

The ToolCatalog and CapabilityContracts trusted-input documents retain their
released v1alpha4 artifact contract identities. They are immutable external
contracts, not Compiler language targets.

## Artifact layout

```text
manifest.json
skills/<skill-name>/compiled/procedure.json
skills/<skill-name>/compiled/ingress-links.json
```

The manifest records:

- the Compiler Artifact API version and the CA-SL language target, which are
  independent;
- generator product identity without timestamps or environment values;
- the measured dependency closure for ordinary callables, Capability members,
  Schema exports and Wait/Ingress trigger Operations;
- per-Skill source digest and source/compiled file paths;
- Wait event, timeout, route, linked Ingress identity, Schema closure and
  paired-response metadata;
- sorted output file paths and SHA-256 digests.

`manifest.json` declares `validationScope.status =
FULL_LANGUAGE_VALIDATION`: every Compiler-owned static obligation in the
released v1alpha5 language and IngressCatalog v1alpha1 contracts is gated by the
single pipeline, including inherited package / Schema / CEL / Predicate /
Script / Evidence / Mutation / Reconcile / retry semantics and the Wait / Ingress
extensions. `notValidated` retains only non-Compiler scopes (Host/Runtime
integration and real Owner/product qualification). This is a static-validation
statement, not product qualification.

## Freeze policy

Field meanings in these files are frozen for the v2.0.0 release line. A future
release that changes a field meaning must publish a new contract version; it
must not silently reinterpret an existing one.
