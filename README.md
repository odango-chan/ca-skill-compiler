# Skill Compiler · skillc

A **stateless, breaking-only v0** compiler/linker for multi-Skill projects. No database, Release, Qualification, managed Workspace, Runtime execution, live MCP call or legacy format compatibility adapters.

~~~text
project/
  skills/            # editable multi-Skill authoring sources
  lib/               # imported Tool/Host/Observation interface snapshots (read-only)
  output/            # generated result (not compiler input)
~~~

## Getting started (target commands, Go 1.23+)

~~~bash
go mod tidy
go test ./... -count=1
go run ./cmd/skillc check ./examples/minimal-project
go run ./cmd/skillc check ./examples/script-contract-project
go run ./cmd/skillc build ./examples/script-contract-project --out ./dist/script-contract
~~~

**Status caution**: Current Core is a prototype. These commands are verification goals, **not yet confirmed PASS**. The compiler may not yet recognize v0 strict Script Output types, Procedure scopes or `lib/manifest.yaml`.

## Real 5GC examples are migration inputs, NOT legacy success fixtures

`examples/ca-agent-demo-project/` includes two complete, unmodified ca-agent-demo Skills at pinned upstream commit, plus two read-only ToolCatalog snapshots. It is an **old-source baseline** for v0 migration and will need Script/Procedure refactoring. Strict new Compiler **must not add compatibility code just to make the old source build green**. MAS proposed catalog still says NOT_READY. Real Host/Observation producer schemas are not fully available; missing owner data is an explicit dependency.

A separate synthetic `examples/script-contract-project` is the intended **positive v0** case: Tool Output imported from lib → Procedure with → Script own strict generated Output Schema.

## Design and local implementation

- [Compiler v0 project contract](docs/design/skill-compiler-v0.md)
- [Script Contract v0 (no compatibility)](docs/design/script-contract-v0.md)
- [Procedure Link v0](docs/design/procedure-link-v0.md)
- [Acceptance matrix G0–G9](docs/workstreams/acceptance-matrix-v0.md)
- [One-shot local Harness Goal WP0–WP7](docs/workstreams/local-implementation-goal.md)
- [Copy/paste Harness prompt](docs/workstreams/local-harness-prompt.md)
- [Skill Studio/WASM ABI](docs/studio-integration.md)
- [WORK status](docs/WORK.md)

Current work: [Draft PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2), tracking [Issue #1](https://github.com/odango-chan/ca-skill-compiler/issues/1). Do not merge main without actual Go, WASM and fixture verification.

Historical [ca-skill-toolchain](https://github.com/odango-chan/ca-skill-toolchain) is background only, not a code dependency.
