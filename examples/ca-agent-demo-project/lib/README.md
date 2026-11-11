# Imported read-only Tool Catalogs

- `tools/createdsession.tool.yaml`: 12 Agent-facing Operations, original ca-agent-demo ToolCatalog at pinned `8ab5145b`.
- `tools/mas-proposed.tool.yaml`: 16 proposed MAS Operations, copied from its design contract, **NOT_READY**.

These snapshots provide Tool interface schemas for **static** linking only; they do not prove running implementations or permission to execute. A full authoritative Host context/facts/observations schema is **not** included in these two files. Do not invent a replacement to obtain green tests. Skill-owned public request schemas remain under `skills/*/schemas/`.
