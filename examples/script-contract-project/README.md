# Strict TypedDict Script Contract fixture

[Script Contract v0](../../docs/design/script-contract-v0.md) is the authority. This is a **compile-only**, synthetic test source: no old YAML script header, no external I/O Schema, no Pydantic runtime dependency.

`skills/normalize/scripts/normalize.py` defines fully typed nested `TypedDict` input/output and static `Annotated` constraints. A correct compiler generates:

- `skills/normalize/schemas/generated/normalize.input.schema.json`
- `skills/normalize/schemas/generated/normalize.output.schema.json`

Required keys, `NotRequired`, literal values, bounds, pattern and `additionalProperties: false` must be preserved. The Compiler must never run Python or silently replace nested types with unconstrained `dict`/object. **This is an expected future test, not currently PASS.**

The real five older Python scripts live unchanged under `examples/ca-agent-demo-project/`; their external Schema remains the authority until explicitly migrated and proven equivalent.
