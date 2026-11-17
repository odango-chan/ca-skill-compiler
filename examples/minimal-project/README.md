# Historical synthetic multi-Skill fixture

This is a **historical pre-v1alpha4 synthetic** compile/link fixture, not a current v1alpha4 example and not a 5GC execution example. It exercises:

- `skills/reader` depending on `skills/writer`;
- a read-only `lib/tools/demo.tool.yaml` ToolCatalog;
- a shared YAML anchor in the Tool input/output Schema;
- selecting `--skill reader` and copying the dependency closure.

The commands from the earlier Compiler iteration are intentionally not listed as current v1alpha4 smoke commands. Use [`web/parity-success`](../../web/parity-success) for the current positive Build path.
