# Minimal multi-Skill project

This is a **synthetic** compile/link fixture, not a 5GC execution example. It exercises:

- `skills/reader` depending on `skills/writer`;
- a read-only `lib/tools/demo.tool.yaml` ToolCatalog;
- a shared YAML anchor in the Tool input/output Schema;
- selecting `--skill reader` and copying the dependency closure.

From repository root:

```bash
go run ./cmd/skillc check ./examples/minimal-project
go run ./cmd/skillc build ./examples/minimal-project --skill reader --out ./examples/minimal-project/output
```

The CLI refuses overwriting non-empty output directories. Delete or choose another directory for subsequent builds.

Build produces **files**, not an executable Tool implementation or Runtime READY result.
