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
- Compiler v2.0.0 targets CA-SL v1alpha5 and does not compile this historical source tree as-is. Use this snapshot only for migration analysis and source-provenance review; do not treat a nonzero check result as a compiler regression.
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

This directory is a historical migration input, not a current v1alpha4 acceptance fixture. See [Examples Guide](../README.md) and the current [Compiler Acceptance](https://github.com/odango-chan/ca-skill-compiler/blob/72b41548a9c4d29c6c49abed117a04a963454f22/docs/COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md).
