# CA Skill Compiler — candidate v1alpha5 contracts

**Status: CANDIDATE / DRAFT.** These contracts belong to the candidate
`connectionagent.io/v1alpha5` language/compiler target and are owned by this
repository. They are independent of the frozen v1alpha4 contracts in
`contracts/v1alpha4/`, which remain unchanged. The CA-SL v1alpha5 language
specification (ca-skills #61) and the Integration Ingress Operation contract
(ca-skills #62) are still Draft; nothing in this directory publishes a language,
an Owner artifact, a deployment profile or a product/5GC qualification.

## `artifact-manifest.schema.json`

JSON Schema (Draft 2020-12) for the single top-level `manifest.json` emitted by a
successful v1alpha5 `Build`. The manifest is serialized as canonical RFC 8785
JCS. It is emitted by `internal/compiler/v1alpha5_artifact.go` and verified by
`internal/compiler/v1alpha5_artifact_verify.go` through
`compiler.VerifyBuildResult`.

Artifact layout (candidate, compiler-owned):

```text
manifest.json
skills/<skill-name>/compiled/procedure.json
skills/<skill-name>/compiled/ingress-links.json
```

Key properties:

- `apiVersion` is exactly `connectionagent.io/compiler-artifact/v1alpha5`;
  `languageTarget` is exactly `connectionagent.io/v1alpha5`.
- `generator` records the compiler product name/version only. No build commit,
  timestamp, host or environment value is fabricated.
- `dependencyClosure` lists only digest-verified dependency snapshots that are
  referenced by linked Wait Ingress Operations / resolved trigger Operations.
  Entries carry logical identity only (`dependencyId`, `stablePackageId`,
  `exports[]`); no dependency bytes, digests, URLs or credentials are embedded.
- `skills[]` records the Skill name, a domain-separated digest of its exact
  author source bytes, its source file paths, its compiled file paths and one
  `waitLinks[]` entry per Wait-bearing Node. Each entry records the exact event
  selector, timeout, TIMEOUT route, linked Ingress Operation identity
  (`dependencyAlias`, `catalogOwner`, `snapshotPath`, `operationId`,
  `normalizedSchemaRef`, `normalizedSchemaClosureDigest`) and the paired-response
  boundary/producer metadata proven by WP6.
- `outputFiles[]` lists every emitted file except `manifest.json` itself, sorted
  by path, with the sha256 of the exact emitted bytes. A manifest cannot contain
  its own digest; the BuildResult carries the manifest bytes.

Determinism: the same `BuildRequest` produces byte-identical `outputFiles`
(sorted paths, JCS canonical JSON, no timestamps or randomness) and the same
`dependencyClosure`. The source digest excludes request file ordering.

Verification: `compiler.VerifyBuildResult` dispatches on the v1alpha5
`BuildResult.apiVersion` and rejects duplicate/normalized-invalid paths, a
non-canonical or self-inconsistent manifest, a missing or unlisted file, a
sha256 mismatch, duplicate Skill/compiled-file claims, inconsistent
`ingress-links.json` content and a `BuildResult.DependencyClosure` that
disagrees with the manifest.

These contracts are **not** an IngressCatalog/ToolCatalog contract, not a
DependencyInspection contract and not a Runtime artifact-loading contract.
