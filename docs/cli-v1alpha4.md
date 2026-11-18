> **HISTORICAL DOCUMENT.** This file describes the published CA Skill Compiler v1.0.0 (CA-SL v1alpha4). The current CLI is [cli.md](cli.md) and targets CA-SL v1alpha5 only.

# skillc CLI v1alpha4

**Status:** Implemented reference CLI for the frozen CA-SL v1alpha4 Compiler Core.
**Verified by:** `go test ./cmd/skillc -count=1` (CLI-level end-to-end tests that build and execute the real `skillc` binary) plus the repository-wide local gates in [Local Verification](local-verification-v1alpha4.md).
**Boundary:** the CLI is a thin adapter over the same `compiler.Check`, `compiler.Build` and `compiler.InspectDependencies` calls used by the Go/WASM Worker and Skill Studio. It does not contain a second parser, linker, digest implementation or analysis pipeline.

## 1. Commands

| Command | Purpose | Compiler Core call |
|---|---|---|
| `skillc check` | Static analysis only; never writes artifacts | `Check(BuildRequest)` |
| `skillc build` | Compile, link, verify and write the deterministic artifact tree | `Build(BuildRequest)` + `VerifyBuildResult` |
| `skillc inspect` | Read-only dependency snapshot / trusted binding inventory | `InspectDependencies(BuildRequest)` |
| `skillc version` | Print the Compiler product version and language target | `compiler.CompilerVersion` |
| `skillc help` | Usage text | — |

No other subcommands exist. There is deliberately no dependency resolver, package manager, network fetch, release publisher or Runtime dispatcher in the CLI.

## 2. Options

| Option | Applies to | Meaning |
|---|---|---|
| `--request FILE` | all | Read a complete BuildRequest v1alpha4 JSON document with strict decoding (unknown fields rejected). Replaces the `PROJECT` argument. |
| `--out DIR` | `build` | Artifact destination. Must be absent or empty. Defaults to `<PROJECT>/output` in project mode; required with `--request`. |
| `--skill NAME` | all | Select one Skill package target; repeat for several. Cannot be combined with `targets` already present in `--request`. |
| `--json` | all | Machine-readable output on stdout. Human diagnostics and summaries are then suppressed. |

Flags may appear before or after the `PROJECT` argument. `--` ends flag parsing.

## 3. Input mode A — project directory

```text
PROJECT/
  skills/...                        author Skill source (required)
  # or: files/skills/...            equivalent layout used by the parity/Studio fixtures
  snapshots/<dependencyId>/...      immutable dependency snapshot per declared Alias
  trusted.json                      trusted dependency bindings and deployment bundles
  <capabilityDirectory>/...         trusted CapabilityContracts file set
  <evidenceDirectory>/...           trusted EvidenceTrustBindings file set
```

When `PROJECT/files` exists it becomes the source root and the `skills/` tree is read from
`PROJECT/files/skills`; otherwise `PROJECT/skills` is the source root. Only paths under
`skills/` are author input. Symbolic links, nonregular files, files over 2 MiB and projects
over 512 files / 16 MiB are rejected before parsing.

`trusted.json` is a **local input description for trusted callers**, not a frozen wire
contract. The CLI computes every dependency snapshot digest itself with the frozen
`CA-SKILL-DEPENDENCY-SNAPSHOT\0v1\0` algorithm; a `trusted.json` digest field does not
exist, so a caller cannot assert a digest the CLI would otherwise verify. The document
describes which bytes are trusted, not what the trust result should be.

```json
{
  "dependencyBindings": [
    {
      "dependencyId": "ca",
      "stablePackageId": "urn:example:ca:tools",
      "authorizedOwners": ["example-ue-owner"],
      "exportOwners": [
        { "exportKind": "operation", "exportId": "ue.registration", "owner": "example-ue-owner" }
      ]
    }
  ],
  "deployment": {
    "deploymentId": "urn:example:deployment:smf",
    "publisherId": "urn:example:publisher:deployment",
    "authorizationRef": "urn:example:authorization:example-deployment",
    "capabilityDirectory": "capability",
    "evidenceDirectory": "evidence"
  }
}
```

- `exportOwners` is only required for exports of a multi-Owner snapshot; a single-Owner
  snapshot uses `authorizedOwners`.
- `capabilityDirectory` and `evidenceDirectory` are relative to `PROJECT`. Each becomes a
  trusted bundle whose digest the CLI recomputes. When either is present, the deployment
  authorization record is also supplied to the Compiler.
- A binding whose snapshot directory is missing or empty is a hard input error (exit `2`).
  The CLI never substitutes an empty or inferred snapshot.

`trusted.json` is optional. Without it the request has no resolved dependencies and no
trusted build context; a Skill that declares a dependency then receives the frozen
`DEPENDENCY_SNAPSHOT_MISSING` diagnostic and fails closed (exit `1`).

## 4. Input mode B — explicit BuildRequest JSON

```text
skillc check   --request request.json
skillc build   --request request.json --out DIR
skillc inspect --request request.json --json
```

This is the frozen [BuildRequest v1alpha4](design/build-request-v1alpha4.md) transport:
`files[].bytes` are base64 strings, `resolvedDependencies` carry the complete immutable
snapshot file sets and digests, and `trustedBuildContext` carries dependency bindings,
CapabilityContracts and Evidence trust bundles. Decoding rejects unknown properties and
null-for-present fields. Use this mode when an authenticated Host already holds the trusted
input (see build-request §5.4) or when a caller wants to replay an exactly captured request.

The CLI does not convert between project mode and request mode implicitly and does not
write request JSON. `cmd/parity-replay` can emit the request it derives from a project for
parity tooling, but that is a test tool, not part of the CLI contract.

## 5. Output

### 5.1 Human-readable

Diagnostics go to stderr, one per line, with source position, phase and frozen code:

```text
skills/parity/assurance/procedure.yaml:30:9: error C_STATIC YAML_PARSE_ERROR: [30:9] sequence end token ']' not found
<request>: error C_STATIC DEPENDENCY_SNAPSHOT_MISSING: No supplied immutable DependencySnapshot has the used Dependency Alias as its dependencyId: ca
```

Request-level diagnostics use `<request>` as the location. The message may contain bounded
source context for parse diagnostics.

Successful commands print a short summary to stdout:

```text
Skill project static check OK
Dependency inspection OK: 1 callable(s), 3 schema(s)
Skill build complete: 5 file(s), 2 dependenc(ies) -> .../output
```

### 5.2 Machine-readable (`--json`)

| Command | Stdout document |
|---|---|
| `check` | JSON array of Diagnostic objects — the frozen Check output shape in [Compiler Output API §1](design/compiler-output-contract-v1alpha4.md) |
| `build` | [BuildResult v1alpha4](design/compiler-output-contract-v1alpha4.md) with `success`, `diagnostics`, sorted `dependencyClosure` and `outputFiles` (base64 bytes) |
| `inspect` | [DependencyInspection v1alpha4](design/compiler-output-contract-v1alpha4.md) |

`--json` still uses the process exit code for success/failure; the JSON document is the
compiler result, not an authorization token.

## 6. Exit codes

| Code | Meaning |
|---|---|
| `0` | Success: no `C_STATIC` error diagnostics; for `build`, artifact verified and written |
| `1` | Compilation, linking, artifact-verification or output-write failure: diagnostics were reported |
| `2` | Usage, missing/malformed input, unsafe project layout or I/O error before/after compilation |

`check` and `inspect` return `1` if any diagnostic has severity `error`. `build` returns `1`
if `BuildResult.success` is false. A failed build never writes an artifact tree, and a
successful build is structurally verified with `VerifyBuildResult` before any file is
written.

## 7. Fail-closed properties

The CLI enforces the same trust boundary as the Compiler Core:

1. A declared dependency with no supplied snapshot produces `DEPENDENCY_SNAPSHOT_MISSING`
   (exit `1`); a trusted binding naming a missing snapshot directory is an input error
   (exit `2`). No dependency is ever inferred or downloaded.
2. Alias values must resolve to exactly one trusted binding and snapshot; aliases never
   bypass Stable Package ID or Owner verification.
3. Ambiguous or missing export ownership fails with the frozen dependency/Owner
   diagnostics instead of guessing from file order, path or display name.
4. Capability Grants are only checked against a supplied CapabilityContracts deployment
   bundle; the CLI never invents a grant.
5. Output is written only to a fresh directory, through no symlink ancestor, and never over
   the author source; every output path is re-validated before writing.
6. A failed or partially verified build produces no artifact directory content.

## 8. Examples

Using the checked-in trusted fixture:

```bash
skillc version
skillc check   web/parity-success
skillc inspect web/parity-success
skillc build   web/parity-success --out /tmp/skillc-out

skillc check   web/parity-success --json
skillc build   web/parity-success --out /tmp/skillc-out --json
skillc inspect web/parity-success --json

skillc check   web/parity-success --skill verify-ue-registration
```

From Go source without installing the binary:

```bash
go run ./cmd/skillc check ./web/parity-success
```

Expected artifacts for the fixture (verified byte-identical across repeated builds):

```text
skills/verify-ue-registration/SKILL.md
skills/verify-ue-registration/assurance/contract.yaml
skills/verify-ue-registration/assurance/predicates.yaml
skills/verify-ue-registration/assurance/procedure.yaml
skills/verify-ue-registration/generated/artifact-manifest.json
```

## 9. Verification

CLI-level end-to-end tests build the real binary and execute it as a subprocess:

| Test | Covers |
|---|---|
| `TestCLIEndToEndTrustedBuildAndDeterminism` | success check/build, real files, repeated-build byte equality |
| `TestCLIEndToEndJSONContracts` | Check diagnostic array, BuildResult, DependencyInspection wire shapes |
| `TestCLIEndToEndHumanInspect` | human inventory output with Owner attribution |
| `TestCLIEndToEndDiagnosticsArePositioned` | `path:line:column`, phase and frozen code in text and JSON |
| `TestCLIEndToEndMissingDependencyFailsClosed` | no silent success when snapshots are absent |
| `TestCLIEndToEndRequestTransport` | BuildRequest JSON mode, malformed request and flag conflicts |
| `TestCLIEndToEndOutputFreshness` | stale/nonempty output rejected |
| `TestCLIEndToEndSkillTarget` | `--skill` target selection and unknown target failure |
| `TestCLIEndToEndVersion` | product version / language target output and argument rejection |

Run them with:

```bash
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
```
