# Skill Studio ↔ Compiler v0

Status: initial API implemented; Pages/real Skill compatibility not yet verified.

## One shared project

`skills/<name>/**` is editable authoring source, `lib/tools/*.yaml` is imported/read-only, `output/**` is derived. Studio sends a complete byte snapshot of `skills/` and `lib/` to Compiler. It must not send `output/` or allow generated files to be edited as source.

## Core API

`compiler.Check(Request) []Diagnostic` and `compiler.Build(Request) Result` accept a Go slice of `File{Path string, Bytes []byte}`; `Result.OutputFiles` is a sorted, fresh byte snapshot relative to the output directory. Compiler does not read disk, run Python, call Tools, open sockets or create ZIPs.

The WASM bridge accepts and returns JSON strings through `skillcCheck` and `skillcBuild` exported to the Worker. Go's `encoding/json` encodes each `[]byte` as **base64**, not as a JS array:

~~~json
{
  "files": [
    { "path": "skills/example/SKILL.md", "bytes": "IyBFeGFtcGxlCg==" }
  ],
  "targets": ["example"]
}
~~~

Result: `{success, diagnostics:[{severity,code,path,line?,column?,message}], outputFiles:[{path,bytes}]}`. Decode base64 at the Studio boundary. Project UI/Source Editor/Problems/ZIP remain Studio-owned.

The initial implementation checks Skill folder identity, entry Procedure, file references, local scripts, cross-Skill dependency presence, ToolCatalog Operation declarations, and YAML syntax. It **does not** yet implement full Procedure semantic lowering, comprehensive Schema validation, or Script I/O Schema generation. These are later compiler tasks; do not display "fully validated" or "runtime READY" on an initial build.

## Script interface / generated schemas

Canonical [Script Contract v0](design/script-contract-v0.md): a new Script uses complete, strict `TypedDict` input/output annotation types at `main(input: In) -> Out`. Nested object fields must be typed; `Literal`, `NotRequired` and static `Annotated` constraints are supported by the target format. Generic `dict`, `Any`, `object` and empty classes are **not admissible as typed interface contracts**. Plain dicts in the Python function body remain legal. Older `# /// connectionagent-tool` scripts keep their existing separately-authored external Schema in a compatibility mode.

The previously proposed YAML comment-header `schemaMode: embedded` design is **withdrawn** and must not be implemented. Compiler is the sole generator; it emits `skills/<skill>/schemas/generated/*.input.schema.json` / `*.output.schema.json` only in `outputFiles`. Studio shows these outputs as read-only and offers navigation from Script declarations. Neither Studio nor a browser-side second implementation should infer Schema from Python body.

**Implementation: NOT_STARTED; browser proof: NOT_RUN.**

## Library format v0

Use a copy of `ca-agent-demo/tools/contracts/createdsession.tool.yaml` under `lib/tools/`. It defines 12 Agent-facing Operations and Schema/Effect; YAML anchors/aliases are allowed. Do not import `tools/registrations/default.yaml` as a Tool schema library: it is a deployment mapping. Do not treat a declared Tool as installed, ready or authorized.

MCP `tools/list` support is a separate importer to the **same internal Library model**, not a second compiler. Remote MCP connection, tool execution and credentials are not part of Pages Compiler.

## How to build WASM assets (requires Go 1.23+)

~~~bash
GOOS=js GOARCH=wasm go build -o web/skillc.wasm ./cmd/skillc-wasm
WASM_EXEC="$(go env GOROOT)/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC" ]; then WASM_EXEC="$(go env GOROOT)/misc/wasm/wasm_exec.js"; fi
cp "$WASM_EXEC" web/wasm_exec.js
~~~

Host those two files together with `web/skillc-worker.js`. It uses the Go runtime, not Node or a native backend. Production Pages/Chromium tests are **NOT_RUN** until the actual Skill Studio integration is attempted.

## Browser ownership boundary

Studio builds ZIP with its existing exporter from `outputFiles`; it may optionally save the output folder with explicit permission. The compiler does not own `fs`, database, persistent catalogue, run IDs, Release or Qualification. `output/skills` may need an additional Agent Host admission step before executable use.
