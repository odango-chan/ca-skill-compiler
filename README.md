# CA Skill Compiler

**Current target: CA-SL v1alpha4.** A deterministic, stateless Compiler for the frozen
ConnectionAgent Skill Language. The Compiler consumes author Skill source, immutable
dependency snapshots and a trusted build context, and produces frozen diagnostics plus a
deterministic artifact file tree. One Go Core serves Native Go, Go/WASM and the reference
CLI; it performs no network access, dependency acquisition, Python execution or Runtime
dispatch.

## Release

**CA Skill Compiler v1.0.0** (CA-SL v1alpha4 language target) is the first formal
Compiler CLI release. Product version and language version are independent; the release
contains prebuilt Native CLI binaries, the Go/WASM module and SHA-256 checksums. See the
[release notes](docs/release-notes-v1.0.0.md) and the GitHub Releases page.

The Compiler product version recorded in Artifact manifests is `v1.0.0`; the language
target remains the frozen `connectionagent.io/v1alpha4`.

## Normative authority

- **CA-SL v1alpha4:** [ca-skills immutable commit 7f84393](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md)
- **Frozen specification file blob:** `b70064c5737b8ee2b3d1435fe496357f44b42773`
- Compiler-owned frozen contracts: `docs/design/*-v1alpha4.md` and `contracts/v1alpha4/`.
- Companion authoring/schema notes are informative only. The single-file specification is
  the only normative language authority.

v0.1 and v0.2 are historical records only. They are not fallback contracts, this
repository does not translate or infer them, and no compatibility mode exists.

## Quick start

Build the reference CLI and run the checked-in trusted fixture:

```bash
CGO_ENABLED=0 go build -o skillc ./cmd/skillc

./skillc version
./skillc check   web/parity-success
./skillc inspect web/parity-success
./skillc build   web/parity-success --out /tmp/skillc-out
```

Machine-readable output:

```bash
./skillc version
./skillc check   web/parity-success --json   # Diagnostic[]
./skillc inspect web/parity-success --json   # DependencyInspection
./skillc build   web/parity-success --out /tmp/skillc-out-2 --json  # BuildResult
```

`build` verifies the artifact tree before writing it and refuses a nonempty destination.
Dependencies are supplied explicitly through a project layout (`snapshots/` +
`trusted.json`) or an exact BuildRequest JSON document (`--request FILE`); a missing
dependency is reported, never inferred. Full reference: [CLI v1alpha4](docs/cli-v1alpha4.md).

## Repository layout

| Path | Purpose |
|---|---|
| `internal/compiler` | active v1alpha4 Core: BuildRequest envelope, YAML AST, dependency/Owner/Capability linking, Schema/CEL, Procedure graph, SourcePath, Script ABI |
| `internal/compiler/yamlast` | the single normalized YAML parser adapter |
| `internal/pythoncst` | Python CST adapter (parse only; never executes Python) |
| `internal/artifact` | deterministic artifact tree, JCS manifest and integrity verification |
| `internal/projectinput` | shared local project loader used by the CLI and parity tooling |
| `cmd/skillc` | reference Native CLI (check / build / inspect / version) |
| `cmd/skillc-wasm` | Go/WASM entry points for the same Core |
| `cmd/parity-replay` | Native reference projection for the Chromium parity harness |
| `web/` | Worker/Studio pages; `web/parity-success` is the current positive Build fixture |
| `contracts/v1alpha4` | frozen machine schemas and golden vectors |
| `examples/` | labelled historical/migration fixtures; see [Examples Guide](examples/README.md) before use |
| `tests/conformance/v1alpha4` | case index, diagnostic index, fixtures and E-coverage |
| `docs/` | frozen contracts, workstreams, CLI/verification/limitations and acceptance status |

## Documentation

| Document | Purpose |
|---|---|
| [Compiler Contract v1alpha4](docs/design/compiler-contract-v1alpha4.md) | Scope, semantics and acceptance boundary |
| [Compiler Pipeline v1alpha4](docs/design/compiler-pipeline-v1alpha4.md) | Contract pipeline stage → implementation file → test map |
| [BuildRequest v1alpha4](docs/design/build-request-v1alpha4.md) | Source, dependency snapshots and trusted build context |
| [Compiler Output API v1alpha4](docs/design/compiler-output-contract-v1alpha4.md) | Check, BuildResult, DependencyInspection and deterministic ordering |
| [Owner Contract Linking v1alpha4](docs/design/owner-contracts-v1alpha4.md) | ToolCatalog, CapabilityContracts, Evidence and trusted Owner attribution |
| [Procedure Linker v1alpha4](docs/design/procedure-link-v1alpha4.md) | AST, graph, SourcePath, Schema and Capability linking |
| [Script Contract v1alpha4](docs/design/script-contract-v1alpha4.md) | Script ABI and derived Schemas |
| [Compiler Artifact v1alpha4](docs/design/compiler-artifact-v1alpha4.md) | Deterministic output tree and generated Schema serialization |
| [Dependency Snapshot Digest v1](docs/design/dependency-snapshot-digest-v1alpha4.md) | Cross-language immutable snapshot digest |
| [OSS Dependency Selection v1alpha4](docs/design/oss-dependency-selection-v1alpha4.md) | Pinned YAML, Schema, CEL, Python parser and JCS dependencies |
| [CLI v1alpha4](docs/cli-v1alpha4.md) | Commands, options, project layout, exit codes, JSON output |
| [Local Verification v1alpha4](docs/local-verification-v1alpha4.md) | Reproducible acceptance commands and coverage map |
| [Known Limitations v1alpha4](docs/known-limitations-v1alpha4.md) | External contracts, fail-closed branches, WP13 boundary |
| [Release Notes v1.0.0](docs/release-notes-v1.0.0.md) | Formal release features, verification, artifacts and known limitations |
| [Acceptance Status](docs/COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) | Live implementation/E-coverage status and evidence |
| [Conformance plan](docs/workstreams/conformance-v1alpha4.md) | E01–E120, H diagnostics, Native/WASM and 5GC gates |
| [Studio integration](docs/studio-integration-v1alpha4.md) | Go/WASM Worker and read-only Studio boundary |
| [Machine contracts](contracts/v1alpha4/README.md) | JSON Schemas and golden vectors |
| [Examples Guide](examples/README.md) | distinguishes the current positive Build fixture from historical migration inputs |

## Version identifiers

| Boundary | Identifier |
|---|---|
| BuildRequest | `connectionagent.io/compiler-build-request/v1alpha4` |
| CA-SL language target | `connectionagent.io/v1alpha4` |
| BuildResult | `connectionagent.io/compiler-build-result/v1alpha4` |
| DependencyInspection | `connectionagent.io/compiler-dependency-inspection/v1alpha4` |
| ToolCatalog | `connectionagent.io/toolcatalog/v1alpha4` |
| CapabilityContracts | `connectionagent.io/capability-contracts/v1alpha4` |
| Evidence trust relations | BuildRequest v1alpha4-owned embedded format |
| Compiler Artifact | `connectionagent.io/compiler-artifact/v1alpha4` |
| Dependency snapshot digest | `CA-SKILL-DEPENDENCY-SNAPSHOT\0v1\0` |

All are independent exact-match authorities. A BuildRequest API version is never inferred
from `languageTarget`, and no version sniffing, fallback or negotiation exists.

## Implementation status

Authoritative evidence: [Acceptance Status](docs/COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md) and
`tests/conformance/v1alpha4/e-coverage.json`. Status values are evidence-based, not
historical completion claims.

| WP | Scope | Status |
|---|---|---|
| WP0 | OSS selection / frozen contracts | DONE — goccy/go-yaml v1.19.2, San v6.0.3, cel-go v0.33.2, gositter v0.2.0, gowebpki/jcs v1.0.2 integrated |
| WP1 | BuildRequest, digest, limits, Native/WASM transport | PASS (local) |
| WP2 | Single normalized YAML AST with source positions | PASS (local) |
| WP3A–WP3D | Dependency identity, Owner/ToolCatalog, Capability, integrated linking/inspection | PASS (local, fixtures) |
| WP4A–WP4D | G.4 Schema allowlist, registry/closure, F.5 assignability, CEL subset | PASS (local, fixtures) |
| WP5 | Procedure graph, stages, routes, dominance, retry/reconcile | PASS (local) — edge-exact global-Precondition dominance and reachability-exact final-Postcondition classification; Runtime join-AND remains Host-side |
| WP6 | SourcePath grammar, typing, dominance/availability | PARTIAL (fail-closed) — implemented slices pass; identity-input provenance is fail-closed (unfrozen publication) |
| WP7 | Evidence binding, independence, static satisfiability | PARTIAL — static linking/checks pass; Descriptor content/Producer/Scope/Correlation/Freshness fail closed (unfrozen format) |
| WP8 | Mutation/Reconcile/retry contracts | PARTIAL — same-snapshot semantics pass; cross-snapshot association and Runtime retry admission fail closed / outside Compiler |
| WP9 | Python CST + Script ABI | PARTIAL (near complete) — CST, typed input/strict TypedDict output, call-site keys, F.5 typing, key access/guards, PredicateInput projection; Evidence projection blocked |
| WP10 | Deterministic Artifact (JCS tree/manifest) | PASS (local) — successful builds emit byte-deterministic, integrity-verified artifacts |
| WP11 | Native/WASM/Chromium parity | PASS (local, real headless Chromium) — check/build/inspect and artifact bytes identical |
| WP12 | Skill Studio read-only integration | PASS (local, real Chromium) — unified Worker result, diagnostics with positions, read-only artifact view; Studio product editor is outside this repo |
| WP13 | Real 5GC/MAS qualification | NOT_QUALIFIED (external) — no real v1alpha4 publications, MAS not runnable |

**Compiler implementation:** complete for all frozen semantics that do not require an
unfrozen external publication, and locally verified. **Not claimed:** product conformance
certification, external Owner integration readiness, or real 5GC qualification. See
[Known Limitations](docs/known-limitations-v1alpha4.md).

## Local verification

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
git diff --check

# CLI-level end-to-end (builds and executes the real binary)
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v

# Real headless-Chromium parity (requires node + Chrome)
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
```

Details and the requirement coverage map: [Local Verification](docs/local-verification-v1alpha4.md).

## Scope boundary

The Compiler does **not** own Git/HTTP/MCP acquisition, dependency resolution or locking,
package registries, credentials, deployment-authorization decisions, Runtime admission,
Tool/Python execution, Qualification, Release or live 5GC acceptance. It consumes bytes and
trusted metadata supplied by its caller and never opens a path from BuildRequest. A
successful BuildResult is a static compilation result, not an authorization token.

## Development workflow

The v1alpha4 implementation was delivered through stacked PRs #11–#24, merged into `main`
in dependency order for the v1.0.0 release; no force-push was used and #2–#8 (superseded
v0/v0.1 work) were not merged. Release metadata and release notes were merged through the
`release/v1.0.0` PR. See [Acceptance Status](docs/COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)
for the current state; historical status notes are not present evidence.
