# Skill Compiler · skillc

**A small stateless Skill project compiler/linker, not a Skill management platform.**

Project roots:

~~~text
project/
  skills/             # multiple editable SKILL source directories
  lib/tools/          # imported, read-only Tool Catalog snapshots
  output/             # generated (not editable source)
~~~

The repository deliberately contains **no database, Workspace server, Qualification, Release, Runtime or MCP tool execution**.

## Quick start (Go 1.23+)

~~~bash
go run ./cmd/skillc check ./examples/minimal-project
go run ./cmd/skillc check ./examples/ca-agent-demo-project
go run ./cmd/skillc build ./examples/ca-agent-demo-project --skill establish-pdu-session --out ./dist/establish
go run ./cmd/skillc build ./examples/ca-agent-demo-project --skill establish-pdu-session-mas --out ./dist/mas
~~~

`--out` must be empty/nonexistent; a compiler static check does not imply Runtime READY.

## Real integration samples

[`examples/ca-agent-demo-project`](examples/ca-agent-demo-project/README.md) contains **both original Skills copied exactly from ca-agent-demo** and their imported Agent ToolCatalogs. The MAS ToolCatalog is explicitly **NOT_READY**, suitable only for static linking. The compiler should diagnose unsupported syntax accurately instead of modifying the reference Skill files.

## Browser

The same Core exports `skillcCheck` and `skillcBuild` with Go/WASM, bridged by `web/skillc-worker.js`. It accepts JSON with base64 file bytes. See [Skill Studio integration](docs/studio-integration.md).

## Local weak-model handoff

- [One-shot implementation Goal (WP0–WP7)](docs/workstreams/local-implementation-goal.md)
- [Copyable local Harness prompt](docs/workstreams/local-harness-prompt.md)

**Development state**: Draft PR [#2](https://github.com/odango-chan/ca-skill-compiler/pull/2); Go, WASM, real consumer acceptance NOT_RUN until verified locally.

## Other docs

- [Canonical compiler design](docs/design/skill-compiler-v0.md)
- [Script Contract v0: external vs embedded Schema](docs/design/script-contract-v0.md)
- [Implementation status](docs/WORK.md)
- [Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1)
- [Old Toolchain history](https://github.com/odango-chan/ca-skill-toolchain) (not a dependency)
