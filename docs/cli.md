# skillc CLI

**Status:** implemented reference CLI for CA Skill Compiler v2.0.0.
**Language target:** `connectionagent.io/v1alpha5` only.

The CLI is a thin adapter over the same `compiler.Check`, `compiler.Build`,
`compiler.InspectDependencies` and `compiler.VerifyBuildResult` calls used by the
Go/WASM Worker. It contains no second parser, linker, digest implementation or
analysis pipeline.

## Commands

| Command | Purpose | Core call |
|---|---|---|
| `skillc check` | static analysis only; never writes artifacts | `Check` |
| `skillc build` | compile, verify and write the deterministic artifact tree | `Build` + `VerifyBuildResult` |
| `skillc inspect` | read-only dependency snapshot/binding inventory | `InspectDependencies` |
| `skillc version` | product version, language target and API versions | constants |
| `skillc help` | usage text | — |

No other subcommands exist: no dependency resolver, package manager, network
fetch, release publisher or Runtime dispatcher.

## Options

| Option | Applies to | Meaning |
|---|---|---|
| `--request FILE` | all | read a complete BuildRequest JSON document with strict decoding (unknown fields rejected); replaces `PROJECT` |
| `--out DIR` | `build` | artifact destination; must be absent or empty. Defaults to `<PROJECT>/output` in project mode; required with `--request` |
| `--skill NAME` | all | select one Skill package; repeatable. Cannot be combined with `targets` already present in `--request` |
| `--json` | all | machine-readable output on stdout |

Flags may appear before or after `PROJECT`; `--` ends flag parsing.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | compilation failure (diagnostics were reported) or artifact verification/write failure |
| 2 | usage, input or I/O error |

## Project input

```text
PROJECT/
  skills/...                        author Skill source (required)
  # or: files/skills/...            equivalent layout used by parity/Studio fixtures
  snapshots/<dependencyId>/...      immutable dependency snapshot per declared Alias
  trusted.json                      trusted bindings and deployment/artifact configuration
  <capabilityDirectory>/...         trusted CapabilityContracts file set
  <evidenceDirectory>/...           trusted EvidenceTrustBindings file set
```

`trusted.json` fields:

```json
{
  "dependencyBindings": [
    {"dependencyId": "tools", "stablePackageId": "urn:...", "authorizedOwners": ["owner"], "exportOwners": []}
  ],
  "deployment": {
    "deploymentId": "urn:...",
    "publisherId": "urn:...",
    "authorizationRef": "urn:...",
    "capabilityDirectory": "capability",
    "evidenceDirectory": "evidence"
  },
  "snapshotBaseURI": "https://snapshots.example/build/",
  "artifactBindings": [
    {"artifactRef": "urn:...", "dependencyId": "integration", "snapshotPath": "presentation/openapi/x.json", "dialect": "openapi-3.0"}
  ],
  "schemaBindings": [
    {"exportId": "urn:...", "artifactRef": "urn:...", "pointer": "#"}
  ]
}
```

All snapshot digests are computed by the loader; the Compiler re-verifies them
and every trusted artifact digest from the raw bytes. A missing snapshot,
binding, digest or Owner attribution is a hard failure, never inferred.

`trusted.json` is a local convenience description for trusted callers, not a
wire contract. Machine callers holding a complete BuildRequest should use
`--request`.

## Request input

```bash
skillc check   --request request.json
skillc build   --request request.json --out ./artifact
skillc inspect --request request.json
```

The document must carry `apiVersion = connectionagent.io/compiler-build-request/v1alpha5`
and `languageTarget = connectionagent.io/v1alpha5`. The contract is
[contracts/v1alpha5/build-request.schema.json](../contracts/v1alpha5/build-request.schema.json).

## Output

- `check --json` writes a `Diagnostic[]`.
- `inspect --json` writes one `DependencyInspection`, including the independent
  `ingressOperations` array.
- `build --json` writes the `BuildResult`; human mode reports diagnostics and a
  one-line summary.
- Successful output is written atomically only after `VerifyBuildResult` passes.
  Existing nonempty destinations and unsafe or symlinked paths are refused.

Artifact layout:

```text
manifest.json
skills/<skill>/compiled/procedure.json
skills/<skill>/compiled/ingress-links.json
```

## Historical note

The v1alpha4 CLI documentation is retained as historical evidence at
[cli-v1alpha4.md](cli-v1alpha4.md). It does not describe this Compiler version.
