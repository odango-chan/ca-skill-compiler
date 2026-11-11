# Skill Compiler · skillc

A stateless, breaking-only Skill Compiler. The v0 Core is retained on [PR #3](https://github.com/odango-chan/ca-skill-compiler/pull/3); the active implementation branch is migrating it to the narrower Compiler v0.1 Build API and current Skill Language contract. There is no legacy compatibility layer, Package Manager, Runtime execution, live MCP call, database, Release or Qualification subsystem.

~~~text
project/
  skills/            # editable multi-Skill authoring sources
  lib/               # imported Tool/Host/Observation interface snapshots (read-only)
  output/            # generated result (not compiler input)
~~~

## Local verification (Go 1.23+)

~~~bash
GOWORK=off go mod tidy
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go run ./cmd/skillc check ./examples/minimal-project
GOWORK=off go run ./cmd/skillc check ./examples/script-contract-project
GOWORK=off go run ./cmd/skillc build ./examples/script-contract-project --out ./dist/script-contract
~~~

In this environment, `GOWORK=off` avoids a parent `go.work` that refers to three absent sibling repositories. See [WORK status](docs/WORK.md) for the v0 closeout evidence and [v0.1 language alignment](docs/workstreams/language-alignment-v0.1.md) for current implementation status and open owner/spec issues. Passing synthetic examples do not imply complete Language conformance or real Skill acceptance.

## Official Skill sources and preserved legacy reference

`examples/ca-skills-v0.2/` pins ca-skills Skill Language v0.2, its two official Skills and the approved 12-Operation ToolCatalog. `examples/ca-agent-demo-project/` is preserved as a pre-migration reference and is not a current Skill grammar source. Host/Observation Owner Schemas and nested UDM Output fields are missing from the official input snapshot and produce explicit diagnostics; the MAS Operation catalog remains unavailable. The Compiler uses no `lib/manifest.yaml` or source lifecycle state.

A separate synthetic `examples/script-contract-project` is a **positive contract fixture**: Tool Output imported from lib → Procedure `with` → Script strict generated Output Schema.

## Design and local implementation

- [Compiler v0 project contract](docs/design/skill-compiler-v0.md)
- [Compiler v0.1 Language & Linking Contract](docs/design/compiler-contract-v0.1.md)
- [Script Contract v0 (no compatibility)](docs/design/script-contract-v0.md)
- [Procedure Link v0](docs/design/procedure-link-v0.md)
- [Acceptance matrix G0–G9](docs/workstreams/acceptance-matrix-v0.md)
- [One-shot local Harness Goal WP0–WP7](docs/workstreams/local-implementation-goal.md)
- [Copy/paste Harness prompt](docs/workstreams/local-harness-prompt.md)
- [Skill Studio/WASM ABI](docs/studio-integration.md)
- [v0.1 Language Alignment](docs/workstreams/language-alignment-v0.1.md)
- [WORK status](docs/WORK.md)

The original bootstrap is [Draft PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2). v0 implementation and closeout are on [PR #3](https://github.com/odango-chan/ca-skill-compiler/pull/3), targeting `feat/compiler-v0-bootstrap`. The narrowed v0.1 contract is [Draft PR #4](https://github.com/odango-chan/ca-skill-compiler/pull/4); implementation work is kept in a separate dependent branch/PR. No PR is merged into `main` as part of these tasks.

Historical [ca-skill-toolchain](https://github.com/odango-chan/ca-skill-toolchain) is background only, not a code dependency.
