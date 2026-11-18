# CA Skill Compiler

A deterministic, stateless reference compiler for the **ConnectionAgent Skill
Language (CA-SL)**. It consumes author Skill source, immutable dependency
snapshots and a trusted build context, and produces stable diagnostics plus a
deterministic artifact tree.

The Compiler is a single Go core shared by the native CLI and the Go/WASM
module. It performs no network access, dependency acquisition, Python execution
or Runtime dispatch.

## Status

| Item | Value |
|---|---|
| Compiler product version | **v2.0.1** (YAML depth hotfix) |
| Language target | **`connectionagent.io/v1alpha5`** — the only supported target |
| BuildRequest API | `connectionagent.io/compiler-build-request/v1alpha5` |
| BuildResult API | `connectionagent.io/compiler-build-result/v1alpha5` |
| DependencyInspection API | `connectionagent.io/compiler-dependency-inspection/v1alpha5` |
| Artifact API | `connectionagent.io/compiler-artifact/v1alpha5` |
| Static validation scope | `FULL_LANGUAGE_VALIDATION` (Compiler-owned obligations only) |
| Product qualification | **NOT_QUALIFIED** (real Owner, Deployment, Host, Runtime, 5GC and MAS evidence is external) |

CA-SL v1alpha5 inherits the released v1alpha4 language semantics. Those
inherited obligations are implemented by the single v1alpha5 pipeline; the
Compiler no longer exposes a v1alpha4 build entry, CLI target or compatibility
mode. A v1alpha4 `apiVersion` or `languageTarget` fails closed like any other
unsupported version.

The previously released **v2.0.0** product remains immutable. v2.0.1 fixes an erroneous YAML token indentation-depth rejection for CA-SL Procedures with many sibling Nodes, without changing any v1alpha5 input/output contract. The earlier **v1.0.0** product (v1alpha4 target) remains an
immutable historical release. It is documented under
[docs/release-notes-v1.0.0.md](docs/release-notes-v1.0.0.md).

## Normative authority

- **CA-SL v1alpha5:** [release commit `0abfd0e`](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md). It inherits the released v1alpha4 semantics.
- **IngressCatalog v1alpha1:** [release commit `9412be9`](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md).
- **Ingress diagnostic registry v1alpha1:** released with the same commit.
- Compiler-owned contracts: [contracts/v1alpha5](contracts/v1alpha5/README.md).
- Compiler core design: [docs/design/compiler-core-v1alpha5.md](docs/design/compiler-core-v1alpha5.md).

ToolCatalog and CapabilityContracts documents keep their released v1alpha4
artifact contract versions; those are immutable trusted-input formats, not
Compiler language targets.

## Capabilities

The single pipeline enforces the complete released Compiler-owned static scope:

- BuildRequest, dependency-snapshot and trusted-context admission, including
  SHA-256 snapshot verification and exact API/language identities;
- Skill package, Contract, PredicateSet, Schema and Python Script ABI
  admission (Python is parsed, never executed);
- dependency identity, trusted Owner attribution, ToolCatalog callable and
  CapabilityContracts linking;
- JSON Schema registry, `$ref` closure, assignability, CEL subset and Predicate
  typing;
- Procedure graph reachability, dominance, routes, terminals and final
  Postcondition goal proof;
- Evidence Descriptor linking, Mutation / Reconcile / retry static assurance;
- CA-SL v1alpha5 node-entry Wait: trusted IngressCatalog selection,
  trigger/correlation binding, `wait.output` SourcePath typing, event-accepted
  path availability, paired response boundary, response mapping and Schema
  closure;
- deterministic artifact assembly, RFC 8785 JCS serialization and full
  self-verification before any file is emitted.

## Build and install

Go 1.25 or newer is required. The Compiler is CGo-free.

```bash
CGO_ENABLED=0 go build -o skillc ./cmd/skillc
CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm
```

## CLI

```text
skillc check   <PROJECT>                       static check a Skill source project
skillc check   --request <REQUEST.json>        static check an explicit BuildRequest
skillc build   <PROJECT> --out <DIR>           compile and write the artifact tree
skillc build   --request <REQUEST.json> --out <DIR>
skillc inspect <PROJECT>                       inspect dependency snapshots and bindings
skillc inspect --request <REQUEST.json>
skillc version
```

`--skill NAME` selects one Skill package (repeatable); `--json` emits the
machine-readable BuildResult, Diagnostic array or DependencyInspection. Exit
codes: `0` success, `1` compilation failure with diagnostics, `2` usage/input
error.

Project layout:

```text
skills/...                       author Skill source (or files/skills/...)
snapshots/<dependencyId>/...     immutable dependency snapshot file sets
trusted.json                     trusted dependency bindings, deployment bundles
                                 and Ingress/Schema artifact bindings
```

`trusted.json` is a local convenience description for trusted callers, not a
wire contract. Machine callers holding a complete BuildRequest should use
`--request`.

```bash
./skillc check   web/parity-success
./skillc build   web/parity-success --out /tmp/skillc-out
```

## BuildRequest

`BuildRequest` is the only Compiler input API:

```json
{
  "apiVersion": "connectionagent.io/compiler-build-request/v1alpha5",
  "languageTarget": "connectionagent.io/v1alpha5",
  "files": [{"path": "skills/demo/SKILL.md", "bytes": "<base64>"}],
  "resolvedDependencies": [
    {"dependencyId": "ca", "digest": "sha256:...", "files": [{"path": "catalog.yaml", "bytes": "<base64>"}]}
  ],
  "targets": ["demo"],
  "trustedBuildContext": {
    "dependencyBindings": [{"dependencyId": "ca", "digest": "sha256:...", "stablePackageId": "urn:...", "authorizedOwners": ["owner"]}],
    "capabilityContracts": {"deploymentId": "urn:...", "publisherId": "urn:...", "digest": "sha256:...", "files": []},
    "deploymentAuthorizations": [{"deploymentId": "urn:...", "publisherId": "urn:...", "authorizationRef": "urn:..."}],
    "snapshotBaseURI": "https://...",
    "artifactBindings": [{"artifactRef": "urn:...", "dependencyId": "ca", "snapshotPath": "presentation/openapi/x.json", "dialect": "openapi-3.0"}],
    "schemaBindings": [{"exportId": "urn:...", "artifactRef": "urn:...", "pointer": "#"}]
  }
}
```

All digests are recomputed from the supplied bytes; a caller-supplied digest is
never trusted. The full machine contract is
[contracts/v1alpha5/build-request.schema.json](contracts/v1alpha5/build-request.schema.json).

## Artifact

A successful Build emits one deterministic tree:

```text
manifest.json
skills/<skill>/compiled/procedure.json
skills/<skill>/compiled/ingress-links.json
```

`manifest.json` is canonical JCS and records the artifact API version, language
target, generator identity, source digests, dependency closure, Wait/Ingress
link metadata, output paths and SHA-256 digests. `skillc` verifies the tree with
`VerifyBuildResult` before writing it and refuses partial or nonempty
destinations. Contract: [contracts/v1alpha5/artifact-manifest.schema.json](contracts/v1alpha5/artifact-manifest.schema.json).

A successful Build is static-compilation evidence. It is not Runtime
authorization, deployment qualification or product qualification.

## Testing

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
git diff --check
```

Acceptance commands and recorded results are in
[docs/COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md](docs/COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

## Limits and boundaries

- The Compiler proves static language obligations only. It does not receive or
  isolate live events, schedule timeouts, execute Operations or Python, acquire
  dependencies, authorize deployment or qualify a product.
- External responsibilities are recorded in
  [docs/known-limitations-v1alpha5.md](docs/known-limitations-v1alpha5.md).
- Real Owner, Deployment, Host, Runtime, 5GC and MAS evidence remains
  **NOT_QUALIFIED**.
- Historical v1alpha4 documentation and fixtures are retained for traceability
  and are marked as historical; they are not current compiler targets.

## Repository layout

| Path | Purpose |
|---|---|
| `internal/compiler` | the single v1alpha5 Compiler core and public API |
| `internal/compiler/yamlast` | the one normalized YAML parser adapter |
| `internal/ingresscatalog` | IngressCatalog v1alpha1 static resolution |
| `internal/pythoncst` | Python CST adapter (parse only) |
| `internal/projectinput` | project-directory to BuildRequest loader |
| `internal/parity` | Native/WASM/Chromium parity harness |
| `cmd/skillc` | native CLI |
| `cmd/skillc-wasm` | Go/WASM module |
| `contracts/v1alpha5` | Compiler-owned machine contracts |
| `docs` | current design/acceptance records; historical records are labelled |
