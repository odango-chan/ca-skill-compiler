> **Current v0 wire implementation only.** The successor v0.1 design is [Compiler Language & Linking Contract v0.1](design/compiler-contract-v0.1.md), aligned to the official ca-skills Skill Language v0.2. v0.1 is not yet available in the current native/Go/WASM binaries; do not send v0.1 requests to the v0 Worker.

# Skill Studio ↔ Skill Compiler · Browser Integration

状态：**v0 wire implementation exists / v0.1 design only**。新接口采用 breaking-only；不对接旧 Toolchain 或 legacy Script Parser。

## 1. File and ownership boundary

~~~text
project/
  skills/      # author-editable official Skill source; multiple Skills
  lib/**       # logical namespace for imported read-only contracts
  outputFiles  # virtual BuildResult output; no physical output/ directory required
~~~

- ca-skills is the only official Skill source and Skill-language authority.
- Studio owns Explorer, source editor, file import, read-only Library presentation, Problems, file authorization and ZIP/folder export.
- Compiler owns the unique Parser/AST/Linker/Schema Generator. Studio must not implement a second semantic parser.
- Studio/caller prepares all source and imported contract bytes in memory and passes them to Go/WASM. A physical lib/ directory is not required.
- outputFiles are read-only generated results. They must never be submitted as source in the next BuildRequest.
- Imported Tool/Host/Observation contracts are read-only in Studio. Refresh/import replaces an imported file; it does not edit Skill source.

## 2. Core API

~~~typescript
type File = { path: string, bytes: Uint8Array };
type Diagnostic = {
  severity: "error" | "warning" | "info",
  code: string,
  path: string,
  line?: number,
  column?: number,
  message: string
};
type BuildRequest = { files: File[], targets?: string[] };
type BuildResult = { success: boolean, diagnostics: Diagnostic[], outputFiles: File[] };
type LibraryTool = {
  operationId: string,
  sourcePath: string,
  declaredEffect: "READ" | "MUTATION",
  description?: string,
  inputSchema: object,
  outputSchema?: object,
  reconcileOutputSchema?: object
};
type LibraryResult = { diagnostics: Diagnostic[], tools: LibraryTool[] };

check(input: BuildRequest): Diagnostic[];
build(input: BuildRequest): BuildResult;
inspectLibraries(input: BuildRequest): LibraryResult;
~~~

- check and build share the same static analysis. check returns diagnostics; build returns diagnostics and files. If any ERROR exists, success=false and outputFiles=[].
- inspectLibraries reads the supplied ToolCatalog files and returns a read-only Operation/Schema projection. It does not maintain manifests, source maturity, dependency states or import lifecycle.
- There is no lib/manifest.yaml, sourceStatus, lock, resolver or package-manager API.
- Procedure Script input is derived from real Producer Schemas at each with: call site. Predicate Script input is the declared Host Predicate Evaluation Context. Script Output Schema is generated from a strict Output TypedDict.
- Missing real Host/Observation Owner schemas or missing nested Tool Output properties produce explicit diagnostics; they must not be fabricated or hidden.

## 3. JS ↔ Go/WASM wire

Go encoding/json represents []byte as a base64 string at the JSON Worker boundary. Studio may keep Uint8Array internally.

~~~json
{
  "files": [
    { "path": "skills/example/SKILL.md", "bytes": "IyBFeGFtcGxlCg==" }
  ],
  "targets": ["example"]
}
~~~

Go Core exposes Check, Build and InspectLibraries; matching WASM callbacks are:

- skillcCheck(jsonRequestString): jsonResultString
- skillcBuild(jsonRequestString): jsonResultString
- skillcInspectLibraries(jsonRequestString): jsonResultString

Worker request/response:

~~~typescript
postMessage({ id: "r1", command: "check" | "build" | "inspect", request: BuildRequestAsBase64 });
onmessage({ data: { id: "r1", result?: BuildResult | LibraryResult, error?: string } });
~~~

- error is a Worker/Go/WASM host failure, not a Compiler diagnostic. Static validation errors arrive in result.diagnostics.
- ToolCatalog YAML anchors/aliases are parsed by Go Core; Studio does not implement semantic YAML processing.
- Worker never carries MCP credentials, Server URL or Runtime admission token and never performs remote Tool calls.

## 4. Browser build and acceptance

The same Go Core must be compiled to GOOS=js GOARCH=wasm and used by the Browser Worker. wasm_exec.js must match the Go toolchain version.

Native and WASM must return equivalent diagnostic codes, source positions, output paths and output bytes for the same BuildRequest. A WASM startup smoke test alone is insufficient; run actual Chromium tests over positive and negative source fixtures.

The official current source inputs are the ca-skills v0.2 packages. Do not treat raw ca-agent-demo Skill directories as a second source of truth or require unrefreshed legacy files to compile.

## 5. Host / MCP boundary

- Tool Library is an imported interface file, not a Tool implementation, MCP connection or deployment registration.
- Studio may obtain or normalize an external MCP/file import into the supported ToolCatalog input format. Compiler only receives the resulting bytes.
- Host/Observation Schemas must come from their real Owners. Current PDU source has known missing Owner contracts and an underspecified UDM Tool Output Schema; Compiler must report the gaps.
- Host package loading and runtime admission are a separate breaking migration, not part of Compiler v0.1.
- BuildResult.success means only static compilation and packaging succeeded. It does not prove Python execution, MCP availability, MAS readiness or live 5GC E2E.
