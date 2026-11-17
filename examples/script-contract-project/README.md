# Synthetic producer-owned input / Script-owned output fixture

- `lib/tools/demo.tool.yaml` is an exact, read-only copy of the synthetic ToolCatalog in `examples/minimal-project/lib/tools/demo.tool.yaml`.
- `skills/normalize/schemas/normalize-request.schema.yaml` is the Skill-owned input contract.
- Procedure binds `demo.read.outputSchema.properties.value` via `with.value.from: read.output.value` into the Script.
- Script uses `main(input: dict) -> NormalizeOutput`: input is the derived argument envelope; **only Script's newly computed output is locally typed**.
- Compiler target output is `output/skills/normalize/schemas/generated/normalize.output.schema.json` with all declared strict constraints; do not generate an independently authored duplicate Input Schema.

This is a new **test fixture**, not proof that current Compiler implements link typing or passes tests. See [Script Contract v0](../../docs/design/script-contract-v0.md).
