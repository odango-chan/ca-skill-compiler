# CA Skill Compiler v2.0.0

**Product version:** CA Skill Compiler **v2.0.0**
**Language target:** CA-SL **v1alpha5** (`connectionagent.io/v1alpha5`) — the only
supported target
**Artifact API:** `connectionagent.io/compiler-artifact/v1alpha5`
**Release status:** release inputs prepared; publication requires explicit human
approval. This document does not announce a published release.

## Highlights

- **Single-version architecture.** The Compiler now has one language target, one
  BuildRequest/BuildResult/DependencyInspection API identity, one CLI target,
  one static pipeline and one deterministic artifact path.
- **CA-SL v1alpha5 support** including node-entry Wait, trusted IngressCatalog
  linking, trigger/correlation binding, `wait.output` typing, paired response
  boundaries and deterministic artifact metadata.
- **Inherited v1alpha4 semantics** (package, Contract, PredicateSet, dependency
  identity, Owner/Tool/Capability linking, Schema closure, CEL, Python Script
  ABI, Procedure graph, SourcePath, Evidence Descriptor, Mutation/Reconcile and
  retry assurance) are implemented by the same v1alpha5 pipeline.
- **Contract freeze.** `contracts/v1alpha5/` contains the frozen BuildRequest,
  BuildResult, DependencyInspection, artifact-manifest and digest-vector
  contracts. The frozen BuildRequest schema accepts trusted artifact/Schema
  bindings only at `trustedBuildContext.artifactBindings` and
  `trustedBuildContext.schemaBindings`; the public schema and the strict JSON
  decoder agree on the accepted property set, and the artifact manifest schema
  declares its FROZEN v2.0.0 status in its own metadata.

## Breaking changes

- The v1alpha4 BuildRequest API identity
  (`connectionagent.io/compiler-build-request/v1alpha4`), BuildResult identity,
  DependencyInspection identity and `connectionagent.io/v1alpha4` language
  target are no longer accepted. Every such request fails closed with
  `REQUEST_INVALID` / `LANGUAGE_TARGET_UNSUPPORTED`.
- The v1alpha4 CLI target selection and v1alpha4 artifact generator/verifier are
  removed. `skillc` always compiles CA-SL v1alpha5.
- A successful BuildResult must carry the v1alpha5 BuildResult API version to
  pass `VerifyBuildResult`.
- The product version is **v2.0.0** because the public compiler API identity and
  accepted language target changed. The published **v1.0.0** release remains
  immutable historical evidence and is not rewritten or re-tagged.

## API and artifact

| Item | Value |
|---|---|
| BuildRequest | `connectionagent.io/compiler-build-request/v1alpha5` |
| BuildResult | `connectionagent.io/compiler-build-result/v1alpha5` |
| DependencyInspection | `connectionagent.io/compiler-dependency-inspection/v1alpha5` |
| Artifact manifest | `connectionagent.io/compiler-artifact/v1alpha5` |
| Language target | `connectionagent.io/v1alpha5` |

Successful Build output:

```text
manifest.json
skills/<skill>/compiled/procedure.json
skills/<skill>/compiled/ingress-links.json
```

`manifest.json` is canonical RFC 8785 JCS with source digests, dependency
closure, Wait/Ingress link metadata, output paths, SHA-256 digests and
`validationScope.status = FULL_LANGUAGE_VALIDATION` for the Compiler-owned static
scope.

## Compatibility and migration

- Update `apiVersion` and `languageTarget` to the v1alpha5 identities.
- A project directory is loaded as a v1alpha5 request. `trusted.json` may declare
  `snapshotBaseURI`, `artifactBindings` and `schemaBindings` in addition to
  dependency bindings and deployment bundles.
- Trusted ToolCatalog and CapabilityContracts documents keep their released
  v1alpha4 artifact contract identities; they are external immutable formats,
  not Compiler language versions.
- There is no translation layer, version sniffing or fallback.

## Verification

Local acceptance at the integration head `ffeff37`: `go test ./... -count=1`,
`go vet`, CGO-free native build, Go/WASM build, CLI end-to-end, real Chromium
Worker parity and `git diff --check` all PASS (Go 1.26.6, Node 24.18.0, Chromium
158.0.8127.37 on Windows/amd64; the Chromium parity harness runs through
`GOTOOLCHAIN=go1.27.4` as required by `scripts/chromium-parity.mjs`). The
BuildRequest schema/decoder alignment and the artifact-manifest freeze
declaration are covered by executable contract tests. GitHub Actions did not
execute (billing restriction, zero steps); no CI PASS is claimed. Full record:
[docs/COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md](COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

## Limitations

The Compiler proves static language obligations only. Host/Runtime event
admission, Wait lifecycle, timeout scheduling, Operation/Python execution,
deployment authorization and real 5GC/MAS interoperability are out of scope and
remain **NOT_QUALIFIED**. See
[docs/known-limitations-v1alpha5.md](known-limitations-v1alpha5.md).

## Release assets (prepared, not yet published)

When approved, the v2.0.0 release should publish the Native CLI binaries for the
supported platforms, the Go/WASM module and SHA-256 checksums, generated from the
merge commit. No tag or Release is created by this document.
