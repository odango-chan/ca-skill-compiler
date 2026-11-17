# Imported (read-only) Library files

- `tools/createdsession.tool.yaml`: exact imported Agent-facing ToolCatalog, 12 Operations.
- `tools/mas-proposed.tool.yaml`: exact imported MAS proposal, 16 Operations, explicitly PROPOSED/NOT_READY.
- `manifest.yaml`: **new local import metadata**. Its statuses `declared` / `proposed` / `unspecified` describe contract maturity, not running implementation/authorization/READY.

The original ToolCatalog files are unchanged; only the import metadata belongs to the compiler example project. Host/Ingress/Observation data contracts are **not included** because no confirmed authoritative snapshot has been identified. Never fabricate those schemas or mistake Root Routing Profile (deployment settings) for runtime data.

See [Script Contract](../../../docs/design/script-contract-v0.md) and [Procedure Link Contract](../../../docs/design/procedure-link-v0.md).
