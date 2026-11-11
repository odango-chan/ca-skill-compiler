# ca-agent-demo current-main Skill source snapshot

This folder is a read-only verification snapshot of two current business Skills from `odango-chan/ca-agent-demo` at `main` commit `4395982fa6efb97714b97901603b2dfb7018c4d0` (2026-11-11). The 79 Skill files and the single imported ToolCatalog were checked file-by-file against their exact upstream Git blob IDs. No source file was adapted or edited.

The snapshot contains:

- `skills/establish-pdu-session/`
- `skills/establish-pdu-session-mas/`
- `lib/tools/createdsession.tool.yaml` (the approved 12-Operation ToolCatalog)

The MAS Skill's `design/tool-contract.yaml` is intentionally not copied to `lib/`: it is a proposal and not an approved ToolCatalog. This snapshot also contains no fabricated Host `subject` / `facts` or Observation Owner Schema.

The strict Compiler is expected to diagnose unresolved Owner Schemas, the legacy Script source contract and MAS Operations missing from the approved ToolCatalog. A nonzero compile result is therefore migration/linking evidence, not a failed source-integrity check. Cases remain synthetic obligations; they are not live or Runtime execution evidence.

```bash
GOWORK=off go run ./cmd/skillc check ./examples/ca-agent-demo-current-main --skill establish-pdu-session
GOWORK=off go run ./cmd/skillc check ./examples/ca-agent-demo-current-main --skill establish-pdu-session-mas
```

Current diagnostic counts and categories are recorded in [the v0.1 language alignment report](../../docs/workstreams/language-alignment-v0.1.md); counts may change as unsupported but documented language forms are implemented.
