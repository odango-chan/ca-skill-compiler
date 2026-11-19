# CA Skill Compiler v2.0.2 Release Record

**Status:** FROZEN / RELEASED  
**Release date:** 2026-11-19  
**Product version:** `v2.0.2`  
**Release tag:** `v2.0.2`  
**Immutable release source:** `e98a7b26acabfa2a5e1587974ab9be5742e70bf7`  
**Language target:** CA-SL v1alpha5 (`connectionagent.io/v1alpha5`)

## Release scope

CA Skill Compiler v2.0.2 is a patch release on the v2.0.0 product line. It retains the FROZEN Compiler-owned v1alpha5 machine contract identities and the single supported CA-SL v1alpha5 language target.

The release includes:

- one Go Compiler Core shared by native CLI and Go/WASM Worker;
- removal of v1alpha4 build, CLI and artifact compatibility paths;
- complete Compiler-owned static CA-SL v1alpha5 validation scope, including inherited package, contract, Predicate, Tool, Capability, Owner, Schema/CEL, Procedure Graph, SourcePath, Evidence, Mutation, Reconcile, retry and Wait/Ingress linking;
- deterministic compiled artifacts with self-verification and canonical RFC 8785 JCS serialization;
- the v2.0.2 patch fix preserving producer outputs after a successful later stage while rejecting paths that bypass or fail the producer.

## Released Compiler contracts

All contracts below are FROZEN and are sourced from the exact release commit `e98a7b26acabfa2a5e1587974ab9be5742e70bf7`.

| Contract ID | Contract Version | Normative artifact |
|---|---|---|
| `compiler-api-build-request` | `v1alpha5` | `contracts/v1alpha5/build-request.schema.json` |
| `compiler-api-build-result` | `v1alpha5` | `contracts/v1alpha5/build-result.schema.json` |
| `compiler-api-dependency-inspection` | `v1alpha5` | `contracts/v1alpha5/dependency-inspection.schema.json` |
| `compiler-artifact-manifest` | `v1alpha5` | `contracts/v1alpha5/artifact-manifest.schema.json` |

The dependency snapshot digest vectors at `contracts/v1alpha5/dependency-snapshot-digest-vectors.json` are published alongside DependencyInspection as its conformance fixture.

The Compiler does not republish or override CA-SL v1alpha5, IngressCatalog v1alpha1, Ingress Diagnostics v1alpha1, ToolCatalog or CapabilityContracts. Their ownership remains with their respective Owner repositories.

## Validation and qualification boundary

The v2.0.2 release workflow gates the release on Go unit tests, vet, CGO-free build, CLI end-to-end and real Chromium Worker parity.

These checks establish the released Compiler build's implementation scope. They do not establish production Owner publication, Deployment authorization, Host/Runtime integration, real 5GC or MAS qualification. Those remain **NOT_QUALIFIED** until separately verified with real implementation and deployment evidence.

## Versioning

Product version `v2.0.2`, Compiler contract versions `v1alpha5`, and language target `connectionagent.io/v1alpha5` are independent identities. A future change to a FROZEN contract meaning requires a new Contract Version; it must not silently reinterpret these released artifacts.
