# CA Skill Compiler

[简体中文](README.zh-CN.md)

CA Skill Compiler compiles [ConnectionAgent Skill Language (CA-SL)](https://github.com/odango-chan/ca-skills) packages into deterministic artifact directories.

It provides a Go compiler core, a command-line tool (`skillc`), and a Go/WASM build for browser-based integrations.

**Current release:** [v1.0.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0)  
**Language target:** CA-SL `connectionagent.io/v1alpha4`

## What it does

- Checks Skill source and reports diagnostics with file and source positions.
- Resolves declared dependencies against supplied immutable snapshots and trusted metadata.
- Validates schemas, predicates, procedure graphs and Script interfaces.
- Builds a deterministic artifact directory and verifies it before writing.
- Provides the same compiler core to the native CLI and browser/WASM integrations.

Python files referenced by a Skill are parsed for their declared interface. The compiler does not execute them.

## Install

Download the binary for your platform from [Releases](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v1.0.0).

| Platform | Release asset |
|---|---|
| Windows x86-64 | [skillc-v1.0.0-windows-amd64.exe](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-windows-amd64.exe) |
| Linux x86-64 | [skillc-v1.0.0-linux-amd64](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-linux-amd64) |
| macOS Apple Silicon | [skillc-v1.0.0-darwin-arm64](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-darwin-arm64) |
| macOS Intel | [skillc-v1.0.0-darwin-amd64](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0-darwin-amd64) |

The release also includes the [Go/WASM module](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/skillc-v1.0.0.wasm) and [SHA-256 checksums](https://github.com/odango-chan/ca-skill-compiler/releases/download/v1.0.0/SHA256SUMS).

## Quick start

Clone the repository to try the included v1alpha4 sample:

```bash
git clone https://github.com/odango-chan/ca-skill-compiler.git
cd ca-skill-compiler

skillc version
skillc check web/parity-success
skillc inspect web/parity-success
skillc build web/parity-success --out ./dist/skillc-output
```

Use a new or empty output directory for each build. The sample is a compiler test fixture; it is not a deployment or 5GC integration.

For a project of your own, the CLI accepts a project directory containing Skill source and, when required, dependency snapshots and trusted build metadata. It can also read an explicit BuildRequest JSON document with `--request`.

See the [CLI guide](docs/cli-v1alpha4.md) for input layouts, options, JSON output and exit codes.

## Build from source

Requirements: Go 1.25 or later.

```bash
go build -o skillc ./cmd/skillc
go test ./...
```

The same Go core is used by the native CLI and the WASM build.

## Language and compatibility

Version 1.0.0 targets CA-SL v1alpha4:

- Language specification: [CA-SL v1alpha4](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md)
- Compiler release notes: [v1.0.0](docs/release-notes-v1.0.0.md)

The compiler does not translate older v0.1/v0.2 Skill formats. The newer author-level Wait semantics are not part of v1alpha4 and are not supported by this release.

Some Evidence, Mutation/Reconcile and Retry positive Build paths remain fail-closed until their required contracts and static checks are complete. See [Known limitations](docs/known-limitations-v1alpha4.md).

## Documentation

- [CLI reference](docs/cli-v1alpha4.md)
- [Examples and fixtures](examples/README.md)
- [Documentation index](docs/README.md)
- [Release notes](docs/release-notes-v1.0.0.md)

Compiler design contracts, integration notes and maintenance records are listed separately in the documentation index.

## License

No `LICENSE` file or open-source license is currently declared for this repository. Do not assume that use, modification or redistribution is permitted. A license must be selected before the repository is presented as an open-source project.
