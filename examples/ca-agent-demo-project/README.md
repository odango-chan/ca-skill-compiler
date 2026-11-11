# ca-agent-demo real dual-Skill source snapshots · MIGRATION INPUT

These are **pinned upstream source references**, not a legacy-compatibility test target. All **79 Skill source blobs plus one official ToolCatalog** match `ca-agent-demo@8ab5145b992f74ee793e0d35a4ec96e6b26500a3` byte-for-byte (based on Git blob identity).

~~~text
examples/ca-agent-demo-project/
  skills/
    establish-pdu-session/        # original Script metadata + Procedures/Predicates/Cases/References
    establish-pdu-session-mas/    # original MAS Skill + proposal
  lib/
    tools/
      createdsession.tool.yaml    # 12 existing Agent-facing operation interfaces
      mas-proposed.tool.yaml     # 16 PROPOSED / NOT_READY interfaces
    manifest.yaml                 # import source/status labels (added by compiler project)
~~~

- `lib/tools/createdsession.tool.yaml` is the authoritative original ToolCatalog snapshot; `mas-proposed.tool.yaml` is byte-identical to MAS `design/tool-contract.yaml`.
- `lib/manifest.yaml` is an added project-local **interface provenance/maturity** manifest, not an upstream file. `declared` / `proposed` do not mean Runtime READY or Tool authorization.
- The two Skills' existing `schemas/establishment-request.schema.yaml` are authored Skill input schemas, not an invented Host context schema. Full external Host/Observation owner schemas are **not provided** by these ToolCatalogs.
- The new v0 Compiler has **NO obligation to compile the old Script metadata as-is**. In v0 validation, these snapshots are expected to reveal which source changes and which external contracts are still needed.
- Do not alter these pinned files to force PASS. If migrating, create a separate `examples/ca-agent-demo-v0/` with redesigned Script signatures, authoritative lib references and new Procedure grammar, then validate there.
- Compiling the ZIP from these original files does not imply actual Agent Host acceptance, ready Tool implementations, or 5GC E2E.

## Tests

Positive v0 fixtures:
- `examples/minimal-project` (two Skills and a synthetic ToolCatalog)
- `examples/script-contract-project` (Tool library + Procedure binding + strict Script Output Schema)

Raw pre-migration diagnostic probe (expected to report unsupported / missing Owner interface until migrated):

~~~bash
go run ./cmd/skillc check ./examples/ca-agent-demo-project
~~~

This command is **not** a universal zero-exit gate. A nonzero exit with accurate source-backed migration diagnostics can be correct under new strict semantics. Do not hide errors by falling back to old Script/Schema behavior.

Implementation and fixture acceptance results: [WORK.md](../../docs/WORK.md) and [Acceptance Matrix](../../docs/workstreams/acceptance-matrix-v0.md).
