# CA Skill Compiler — v1alpha5 compiler-owned artifact contract

**Language authority:** CA-SL v1alpha5 is FROZEN / RELEASED by ca-skills PR #66.  
**Ingress authority:** IngressCatalog v1alpha1 and its diagnostic registry are FROZEN / RELEASED by ca-skills PR #62.  
**Compiler implementation:** CANDIDATE / PARTIAL STATIC COVERAGE; NOT READY TO RELEASE.

This directory contains the Compiler-owned artifact manifest schema for the v1alpha5 implementation. It does not republish or override the language or Ingress logical contracts.

Normative sources:

- [CA-SL v1alpha5](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md)
- [IngressCatalog v1alpha1 logical contract](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md)
- [Ingress diagnostic registry v1alpha1](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/ingress-diagnostic-registry-v1alpha1.md)

## `artifact-manifest.schema.json`

JSON Schema (Draft 2020-12) for the top-level `manifest.json` emitted by the current candidate v1alpha5 Build. It is serialized as canonical RFC 8785 JCS, emitted by `internal/compiler/v1alpha5_artifact.go` and verified by `internal/compiler/v1alpha5_artifact_verify.go`.

Candidate artifact layout:

```text
manifest.json
skills/<skill-name>/compiled/procedure.json
skills/<skill-name>/compiled/ingress-links.json
```

The manifest currently records:

- exact Compiler Artifact API version and CA-SL language target;
- generator product identity without fabricated timestamps or environment values;
- the measured dependency closure for the linked Wait Ingress / trigger Operations;
- source digest and source / compiled file paths per Skill;
- Wait event, timeout, route, linked Ingress identity, Schema closure and paired-response metadata;
- sorted output file paths and SHA-256 digests.

The manifest now declares `validationScope.status = FULL_LANGUAGE_VALIDATION`: every Compiler-owned static obligation in the released v1alpha5 language and IngressCatalog v1alpha1 contracts is gated by the unified pipeline, including ordinary non-Wait callable linking, inherited package / Schema / CEL / Predicate / Script / Evidence / Mutation / Reconcile / retry semantics and the Wait / Ingress extensions. `notValidated` retains only non-Compiler scopes (Host/Runtime integration and real Owner/product qualification). This is a static-validation statement, not product qualification.

The artifact layout and validation-scope representation are Compiler-owned candidate decisions. They must be reviewed and frozen before a release-eligible artifact is published. This schema is not an IngressCatalog/ToolCatalog schema, not a Runtime artifact-loading contract and not a product qualification record.
