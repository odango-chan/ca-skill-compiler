# Compiler Artifact Contract v1alpha5 (FROZEN)

**Status: FROZEN for the v2.0.0 product line — compiler-owned.** This document
describes the `connectionagent.io/v1alpha5` artifact assembled by a successful
`Build`. It is independent of the historical v1alpha4 artifact contract
(`docs/design/compiler-artifact-v1alpha4.md`, `contracts/v1alpha4/`). The CA-SL
v1alpha5 language specification (ca-skills PR #66) and the Integration Ingress
Operation contract v1alpha1 (ca-skills PR #62) are released and are the
normative authority. This contract publishes neither Owner artifacts nor
deployment authorization, Runtime admission or 5GC/MAS qualification.

Machine contract: [`contracts/v1alpha5/artifact-manifest.schema.json`](../../contracts/v1alpha5/artifact-manifest.schema.json)
and [`contracts/v1alpha5/README.md`](../../contracts/v1alpha5/README.md). The
machine contract declares its FROZEN v2.0.0 status in its own `title` and
`description`; a successful Build's manifest bytes and every field meaning are
unchanged by that status declaration.

## 1. Build path

`compiler.Build(req)` is the single Build path. It:

1. runs the single-version request/dependency/trusted-context admission
   (`validateBuildRequest`);
2. runs the complete static pipeline — Procedure/Wait AST, Procedure graph,
   trigger/correlation binding, `wait.output` SourcePath binding, response
   boundary/mapping, and the inherited package/linking/Schema/CEL/Predicate/
   Script/Evidence/Mutation/Reconcile/retry passes (`checkSources` →
   `validateSkillPackage`);
3. **only when no error diagnostic was produced**, assembles the complete
   deterministic artifact tree (`assembleSuccessResult`) and returns
   `BuildResult.Success = true` with `OutputFiles`,
   `DependencyClosure` and `apiVersion =
   connectionagent.io/compiler-build-result/v1alpha5`;
4. otherwise returns the real diagnostics (never a fabricated residual gate)
   and an atomic empty result: `success = false`, `outputFiles = []`,
   `dependencyClosure = []`. An internal assembly inconsistency fails the whole
   Build with a clearly marked `CAPABILITY_INPUT_UNTRUSTED` diagnostic whose
   message is prefixed `fail-closed:`.

There is no residual fail-closed gate and no version dispatch. Remaining
external (non-Compiler) obligations are recorded in
[`docs/known-limitations-v1alpha5.md`](../known-limitations-v1alpha5.md); a
successful Build is static-compilation evidence only.

## 2. Layout

The layout is a frozen compiler-owned decision for the v2.0.0 line:

```text
manifest.json
skills/<skill-name>/compiled/procedure.json
skills/<skill-name>/compiled/ingress-links.json
```

- `manifest.json` is the only top-level file and is canonical RFC 8785 JCS.
- `procedure.json` is the validated Procedure document converted from the single
  normalized YAML parse result to canonical JCS JSON. It preserves the validated
  author semantics (`start`, `nodes`, stage/route fields) and is never a
  re-parse or a second compiler implementation.
- `ingress-links.json` is `{"links":[...]}`: the per-Wait linked Ingress
  Operation identity and response-boundary metadata, exactly as embedded in the
  manifest so the two views cannot disagree.
- Source author bytes are **not** copied into the v1alpha5 artifact; the
  manifest records their domain-separated digest and paths instead.

## 3. Manifest content

| Field | Content |
|---|---|
| `apiVersion` | `connectionagent.io/compiler-artifact/v1alpha5` |
| `languageTarget` | `connectionagent.io/v1alpha5` |
| `validationScope` | `{"status":"FULL_LANGUAGE_VALIDATION","validatedPasses":[...],"notValidated":[...]}`. `status` is `FULL_LANGUAGE_VALIDATION` when every Compiler-owned static obligation is gated; `validatedPasses` names the static passes this Build actually gated on; `notValidated` names only non-Compiler scopes (Host/Runtime integration, real Owner/product qualification). A successful artifact is a static-validation statement, never product qualification |
| `generator` | `{"name":"ca-skill-compiler","version":"<CompilerVersion>"}`; no timestamp/host/commit is fabricated |
| `dependencyClosure[]` | Digest-verified dependency snapshots referenced by linked Wait Ingress Operations and resolved trigger Operations: `dependencyId`, `stablePackageId`, sorted `exports[]` (`exportKind`, `exportId`, `owner`). No dependency bytes/digests/URLs/credentials are embedded |
| `skills[]` | Per Skill: `name`, `sourceDigest` (domain-separated digest of the exact author bytes, sorted paths), sorted `sourceFiles[]`, sorted `compiledFiles[]`, `waitLinks[]` |
| `waitLinks[]` | Per Wait-bearing Node: `nodeId`, exact `event` selector, `timeout`, `timeoutRoute`, `ingress` identity, WP6 `response` metadata |
| `ingress` | `dependencyAlias`, `catalogOwner`, `snapshotPath` (IngressCatalog file inside the dependency snapshot), `operationId`, `normalizedSchemaRef`, `normalizedSchemaClosureDigest` (Schema Closure Digest v1), optional `normalizedSchemaSnapshotPath`, and the pinned protocol metadata (`specification`, `release`, `apiUriVersion`, `openapi.operationId`, `method`, `path`) |
| `response` | `mode` (`none`/`paired`), author-`declared`, effective `boundaryNodeId` (default = Wait-bearing Node), typed `producerStage`, `responseResultSchemaRef` + closure digest when present, sorted `earlyFailureTerminals[]` |
| `outputFiles[]` | Every emitted file **except the manifest itself**, sorted by path, with `sha256` of the exact emitted bytes |

A manifest cannot contain its own digest; `BuildResult.OutputFiles` carries the
manifest bytes and the manifest lists every other file so the tree is still
closed.

## 4. Determinism

- Identical `BuildRequest` input produces byte-identical `outputFiles` and the
  same `dependencyClosure`.
- All JSON documents are canonical JCS; output file paths and manifest arrays
  are sorted; maps are serialized deterministically.
- No timestamps, random values, wall clock, environment probes, absolute host
  paths or map-iteration order reach the output.
- The Skill source digest excludes request file ordering (sorted,
  length-prefixed path/bytes with a domain separator).
- Every assertion of determinism is executed by tests: double Build equality,
  and equality across an equivalent request.

## 5. Verification

`compiler.VerifyBuildResult(result)` accepts only a successful result carrying
the `connectionagent.io/compiler-build-result/v1alpha5` API version; any other
result version is rejected. It uses `verifyV1Alpha5ArtifactTree` and rejects:
  - duplicate, empty or non-normalized output paths;
  - a missing or non-canonical top-level manifest;
  - invalid `apiVersion`/`languageTarget`/generator identity;
  - a manifest `outputFiles[]` that is unsorted, missing a tree file, listing a
    missing file or carrying a mismatched sha256;
  - Skill entries with unsorted/duplicate/out-of-tree source or compiled files,
    compiled files claimed twice or not listed in `outputFiles`;
  - an `ingress-links.json` that disagrees with the manifest's `waitLinks`;
  - incomplete Wait/Ingress/response identity or closure export references;
  - a non-canonical Procedure document or a `BuildResult.DependencyClosure`
    that disagrees with the manifest.

The CLI verifies a successful result with the same function before writing the
tree, and writes atomically to a fresh empty output directory only.

## 6. Traceability

| Contract area | Implementation | Tests |
|---|---|---|
| Deterministic assembly and layout | `internal/compiler/v1alpha5_artifact.go` | `TestV1Alpha5BuildAssemblesTrustedArtifact`, `TestV1Alpha5BuildIsDeterministic`, `TestV1Alpha5BuildWaitFreeSucceeds` |
| Manifest/link fields incl. WP6 response metadata | `internal/compiler/v1alpha5_artifact.go` | `TestV1Alpha5BuildAssemblesTrustedArtifact`, `TestV1Alpha5BuildPairedResponseMetadata` |
| Verification and tamper rejection | `internal/compiler/v1alpha5_artifact_verify.go` | `TestV1Alpha5ArtifactVerificationDetectsTampering` |
| Failed Build atomicity | `internal/compiler/v1alpha5.go`, `internal/compiler/api_test.go` | `TestV1Alpha5BuildInvalidEmitsNoArtifact`, `TestBuildReportsWaitDiagnosticsAndEmitsNoArtifact` |
| CLI write/verify | `cmd/skillc/main.go`, `cmd/skillc/e2e_test.go` | `TestCLIEndToEndV1Alpha5Build` |
| Native/WASM/real-Chromium artifact byte parity | `scripts/chromium-parity.mjs`, `internal/parity/*` | `TestChromiumWorkerParity`, `TestV1Alpha5TrustedParityFixtureBuilds` |

## 7. Non-goals

This contract is not an Owner publication, an IngressCatalog/ToolCatalog
contract, a DependencyInspection contract, a Runtime artifact loader, an
authorization token or a product qualification. It does not transport
dependency bytes, credentials, deployment authorizations or Runtime state.
