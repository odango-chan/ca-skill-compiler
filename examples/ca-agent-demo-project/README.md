# Real dual-Skill fixture from ca-agent-demo

Two complete, **unaltered** Skill source directories are copied from
[`odango-chan/ca-agent-demo`](https://github.com/odango-chan/ca-agent-demo/commit/8ab5145b992f74ee793e0d35a4ec96e6b26500a3)
at commit `8ab5145b992f74ee793e0d35a4ec96e6b26500a3`:

- `skills/establish-pdu-session/` — SKILL.md, assurance, scripts, schemas, references, cases, design;
- `skills/establish-pdu-session-mas/` — SKILL.md, assurance, schema, cases, design.

All **79 Skill files plus one ToolCatalog (80 original files)** were checked by Git blob ID against the source commit and matched byte-for-byte. The imported library `lib/tools/createdsession.tool.yaml` is exactly the source `tools/contracts/createdsession.tool.yaml` (12 Agent-facing Operations).

`lib/tools/mas-proposed.tool.yaml` is a separate, byte-identical copy of MAS's own `design/tool-contract.yaml`. That document declares 16 candidate operations and explicitly says **NOT_READY**: external adapters, independent Effect contracts, reconcile and resource-owner proof remain missing. Treat it as a **design-state static linker fixture only**; never claim the MAS tool operations are available/executable/authorized just because the compiler can resolve their names.

All files are checked-in **test examples**, not new authority for official Skill authors. Do not edit the 79 Skill sources or ToolCatalog snapshots just to make static checks green. Refresh them only by a separately reviewed sync from a pinned upstream commit. `ca-agent-demo/tools/registrations/default.yaml` is **not** copied because it is a deployment implementation map, not the Tool Schema library.

## Intended local checks (not yet marked PASS)

~~~bash
go test ./... -count=1
go run ./cmd/skillc check ./examples/ca-agent-demo-project
go run ./cmd/skillc build ./examples/ca-agent-demo-project --skill establish-pdu-session --out ./dist/establish
go run ./cmd/skillc build ./examples/ca-agent-demo-project --skill establish-pdu-session-mas --out ./dist/mas
~~~

A copyable output directory still requires independent Agent Host/Runtime admission and deployment bindings to execute. Static build success is NOT READY and is not a live 5GC or MAS acceptance result.

See [the local implementation Goal](../../docs/workstreams/local-implementation-goal.md).
