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
go run ./cmd/skillc check ./project
go run ./cmd/skillc build ./project --out ./project/output
go run ./cmd/skillc build ./project --skill establish-pdu-session --out ./dist
~~~

`--out` must point to an empty or nonexistent directory so stale build output cannot masquerade as current artifacts. Imported libraries can be placed under `lib/tools/*.yaml`. For the first integration use `ca-agent-demo/tools/contracts/createdsession.tool.yaml`.

**This is an early bootstrap**, not a production-complete compiler: current static checks include root/entry presence, YAML syntax, local Script references, cross-Skill dependencies, ToolCatalog links and basic Operation admission. Full Script Schema generation, advanced Procedure validation/lowering, Native vs WASM parity and real Agent Host acceptance remain open.

## Browser

The same Core exports `skillcCheck` and `skillcBuild` with a Go/WASM main, bridged through `web/skillc-worker.js`. It accepts a JSON request with base64 encoded file contents. See [Skill Studio integration](docs/studio-integration.md). There is no HTTP backend.

## Documentation

- [Canonical compiler v0 design](docs/design/skill-compiler-v0.md)
- [Studio handoff / WASM contract](docs/studio-integration.md)
- [Implementation status](docs/WORK.md)
- [Bootstrap / acceptance Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1)
- [Old Toolchain history](https://github.com/odango-chan/ca-skill-toolchain) (not a dependency)
