# Script Contract v0 · embedded schema test input

This is a **new synthetic fixture** to validate the [Script Contract v0](../../docs/design/script-contract-v0.md) implementation, not a live Skill runtime fixture.

The file `skills/normalize/scripts/normalize.py` declares its **complete** input/output JSON Schema under the existing `# /// connectionagent-tool` comment header, with `schemaMode: embedded`. A static compiler must produce:

- `skills/normalize/schemas/generated/normalize.input.schema.json`
- `skills/normalize/schemas/generated/normalize.output.schema.json`

Each generated Schema must be JSON Schema Draft 2020-12, retain the declared `required`, `additionalProperties`, `minLength` and `pattern` constraints, and use `NormalizeInput` and `NormalizeOutput` as the respective titles. No inferred fields, execution, network calls or source rewrites are allowed.

The embedded mode is **not implemented yet** in the initial `internal/compiler` code; this fixture is a WP3 acceptance input, not a currently passing build.

The old 5GC scripts in `examples/ca-agent-demo-project` retain their existing external Schema files, without modification.
