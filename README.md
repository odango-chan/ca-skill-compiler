# CA Skill Compiler

[简体中文](README.zh-CN.md) · [Documentation](docs/README.md) · [v2.1.0 release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0)

A deterministic, stateless CA-SL compiler. One Go Core powers the native `skillc` CLI and Go/WASM Worker. It validates Skill source, links supplied immutable interfaces, emits a deterministic Artifact and verifies it without executing tools, Python, or network procedures.

**Current product:** **v2.1.0 (RELEASED, 2026-11-23)**. **Only language target:** `connectionagent.io/v1alpha5`. No legacy language mode or compatibility fallback.

## Install and quick start

Download the platform binary and `SHA256SUMS` from the [v2.1.0 release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0): Linux x86-64, macOS Intel / Apple Silicon, or Windows x86-64. Rename the executable to `skillc` (or `skillc.exe`) and place it on your PATH.

Alternatively build from source (Go 1.25+):

```bash
CGO_ENABLED=0 GOWORK=off go build -o skillc ./cmd/skillc
./skillc version
./skillc check web/parity-success
./skillc inspect web/parity-success
./skillc build web/parity-success --out ./dist/example-artifact
```

Use a new or empty output directory. The `web/parity-success` fixture is synthetic, not an actual Owner publication or qualification result.

## Two public build paths

| Path | CLI | Input / output |
|---|---|---|
| Frozen v1alpha5 API | `check`, `build`, `inspect` | `compiler-build-request/v1alpha5`, trusted Build context; frozen deterministic v1alpha5 artifact |
| Development v1alpha1 | `build-development`, `verify-development` | Exact CA-SL source, immutable Interface Package snapshots and Dependency Lock; deterministically linked **`NOT_ADMITTED`** artifact |

Development example using the committed synthetic, complete request:

```bash
./skillc build-development --request docs/design/fixtures/registration-development-request.json --out ./dist/dev-artifact --json > ./dist/dev-result.json
./skillc verify-development --request docs/design/fixtures/registration-development-request.json --result ./dist/dev-result.json
```

The Development Build creates `capabilityRequirements[]`, `host-admission-requirements.json` and `runtime-execution-requirements.json`. **`NOT_ADMITTED` is an expected successful build status, not an execution grant.** Production Host must reject the Development profile.

## Contracts and authority

- [English CLI reference](docs/cli.md) · [中文 CLI](docs/cli.zh-CN.md)
- [Documentation index](docs/README.md) · [中文目录](docs/README.zh-CN.md)
- [Frozen Compiler API v1alpha5](contracts/v1alpha5/README.md) and [Development v1alpha1 machine contracts](contracts/development/v1alpha1/README.md)
- [Toolchain architecture](docs/design/ca-toolchain-architecture-vnext.zh-CN.md) · [Diagram index](docs/architecture/archify/README.md)
- [Compiler-only acceptance and trust boundary](docs/COMPILER-ACCEPTANCE-BOUNDARY.zh-CN.md)
- [v2.1.0 release record](docs/releases/ca-skill-compiler-v2.1.0.md) and [exact-source release workflow](https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980)

Normative CA-SL v1alpha5 and IngressCatalog v1alpha1 are **owned by [ca-skills](https://github.com/odango-chan/ca-skills)**. Compiler checks only static conditions and artifact reproducibility; it does not publish Owners/Providers, issue Deployment Grants, authorize Host execution or qualify 5GC/MAS behavior. Real MAS-source offline compilation is a [separate, nonblocking integration check](https://github.com/odango-chan/ca-skill-compiler/issues/53).

The previously released v1alpha5 Build/Verify artifact bytes remain frozen at v2.0.2. Their embedded `generator.version=v2.0.2` identifies the **frozen artifact format**, not the installed v2.1.0 binary. Development manifests identify v2.1.0 independently.

## Development and history

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o /tmp/skillc.wasm ./cmd/skillc-wasm
```

Real Chromium Worker parity also runs in CI. Historical v0/v0.1/v1alpha4 documents and fixtures remain available via the [history index](docs/workstreams/README.md); they are **not** currently supported input versions.
