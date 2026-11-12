> **Current v0 wire implementation only.** The successor v0.1 design is [Compiler Language & Linking Contract v0.1](design/compiler-contract-v0.1.md), aligned with Skill Studio's merged [Plugin-managed Configuration v0.1](https://github.com/odango-chan/ca-skill-studio-demo/pull/150). v0.1 is not yet available in current native/Go/WASM binaries; do not send v0.1 requests to the v0 Worker.

# Skill Studio ↔ Skill Compiler · Browser Integration

Status: **v0 wire implementation exists / v0.1 contract design only**. v0.1 is a breaking API migration; do not mix its request shape with v0.

## 1. File and ownership boundary

```text
Skill Project/
  skills/                         # author-editable Skill source
  .vscode/settings.json           # host-native workspace settings (where supported)
  .skill-studio/dependencies/     # Studio-owned immutable snapshot storage
  BuildResult                     # in-memory/virtual Compiler output
```

- ca-skills is the only official Skill source and Skill-language authority.
- Studio owns Explorer, source editor, Dependencies TreeView, dependency source acquisition, resource-scoped Workspace Configuration, lock, immutable snapshot persistence, refresh/Update Lock, Problems, file authorization and export.
- The on-disk snapshot store is a Studio storage detail; Compiler receives the normalized snapshot bytes, not its filesystem path.
- Compiler owns the unique Parser/AST/Linker/Schema Generator and canonical snapshot digest definition. Studio implements the same digest algorithm and supplies lock-pinned snapshots.
- Compiler does not require a physical or logical global `lib/**` namespace. There is no requirement to create `lib/` or `output/` folders in the user's project.
- BuildResult output files are read-only generated/selected Skill files. Dependency bytes are not copied into `outputFiles`; the result reports the exact dependency snapshot closure needed to reproduce the build.
- Imported Tool/Host/Observation contracts are immutable snapshots in Studio. A user edits the upstream source or explicitly imports/updates a new snapshot; they do not edit imported contract bytes as if they were Skill source.

## 2. v0.1 Core API (design contract)

~~~typescript
type File = { path: string, bytes: Uint8Array }; // path under skills/**
type ResolvedDependency = {
  dependencyId: string;           // opaque ID from Workspace Configuration
  digest: `sha256:${string}`;    // canonical snapshot content digest
  files: Array<{ path: string, bytes: Uint8Array }>; // snapshot-relative paths
};
type BuildRequest = {
  files: File[];
  resolvedDependencies: ResolvedDependency[];
  targets?: string[];
};
type DependencyClosureEntry = { dependencyId: string, digest: `sha256:${string}` };
type BuildResult = {
  success: boolean;
  diagnostics: Diagnostic[];
  dependencyClosure: DependencyClosureEntry[];
  outputFiles: File[];             // Skill source + generated Skill artifacts only
};
type DependencyTool = {
  operationId: string;
  dependencyId: string;
  sourcePath: string;              // relative to that dependency snapshot
  declaredEffect: "READ" | "MUTATION";
  description?: string;
  inputSchema: object;
  outputSchema?: object;
  reconcileOutputSchema?: object;
};
type DependencyInspection = { diagnostics: Diagnostic[], tools: DependencyTool[] };

check(input: BuildRequest): BuildResult;
build(input: BuildRequest): BuildResult;
inspectDependencies(input: BuildRequest): DependencyInspection;
~~~

- Studio's resource-scoped `dependencies` and `dependencyLock` settings own source, revision, aliases, digest pins and snapshot lifecycle. The Compiler sees only the stable dependencyId, verified digest and immutable bytes.
- Skill source never refers to dependencyId, alias, repository, URL, source revision, digest, cache key or local path.
- Tool calls keep exact Operation IDs. Owner Schemas are referenced by stable JSON Schema `$id` in the vNext language. Current v0.2 `lib/...` schema paths remain a temporary compatibility syntax; when present, they resolve only by exact match against supplied snapshot-relative paths and must be unambiguous.
- Operation IDs and exported Schema IDs are indexed from snapshot contents. No separate dependency manifest or exports list is added to Skill source or Compiler input.
- `dependencyClosure` lists only snapshots actually used by the selected Skill build, sorted by dependencyId. On any ERROR, `success=false`, `dependencyClosure=[]`, and `outputFiles=[]`.
- `inspectDependencies` returns a read-only Operation/Schema projection including dependencyId and snapshot-relative sourcePath. It does not maintain dependency lifecycle, source maturity or Runtime readiness.

## 3. JS ↔ Go/WASM wire

Go encoding/json represents []byte as a base64 string at the JSON Worker boundary. Studio may keep Uint8Array internally.

~~~json
{
  "files": [
    { "path": "skills/example/SKILL.md", "bytes": "IyBFeGFtcGxlCg==" }
  ],
  "resolvedDependencies": [
    {
      "dependencyId": "host-contracts",
      "digest": "sha256:<64 lowercase hex characters>",
      "files": [
        { "path": "schemas/subject.schema.json", "bytes": "eyJ0eXBlIjoib2JqZWN0In0=" }
      ]
    }
  ],
  "targets": ["example"]
}
~~~

The digest is SHA-256 over a versioned binary preimage of sorted snapshot-relative UTF-8 paths and raw bytes. It is independent of file array order and dependencyId. See [dependency-snapshot-digest-v0.1.md](design/dependency-snapshot-digest-v0.1.md) and [machine-readable golden vectors](../contracts/v0.1/dependency-snapshot-digest-vectors.json).

The v0.1 Go Core exposes Check, Build and InspectDependencies; matching WASM callbacks are planned as:

- skillcCheck(jsonRequestString): jsonResultString
- skillcBuild(jsonRequestString): jsonResultString
- skillcInspectDependencies(jsonRequestString): jsonResultString

Worker request/response:

~~~typescript
postMessage({ id: "r1", command: "check" | "build" | "inspectDependencies", request: BuildRequestAsBase64 });
onmessage({ data: { id: "r1", result?: BuildResult | DependencyInspection, error?: string } });
~~~

- error is a Worker/Go/WASM host failure, not a Compiler diagnostic. Static validation errors arrive in result.diagnostics.
- ToolCatalog YAML anchors/aliases are parsed by Go Core; Studio does not implement semantic YAML processing.
- Worker never carries MCP credentials, Server URL or Runtime admission token and never performs remote Tool calls.
- Native Go and WASM implementations must pass identical digest vectors and return equivalent diagnostics, dependencyClosure, output paths and output bytes.

## 4. Browser build and acceptance

The same Go Core must be compiled to GOOS=js GOARCH=wasm and used by the Browser Worker. wasm_exec.js must match the Go toolchain version.

Native and WASM must return equivalent diagnostic codes, source positions, dependencyClosure, output paths and output bytes for the same normalized BuildRequest. A WASM startup smoke test alone is insufficient; run actual Chromium tests over positive and negative source fixtures.

The first real build fixture is the conventional, runnable 5GC `establish-pdu-session` Skill. MAS is not currently end-to-end runnable and is excluded from executable acceptance. The fixture must use actual Host/Tool/Observation Owner contracts; no owner Schema may be fabricated to make the build pass.

## 5. Host / MCP boundary

- A ToolCatalog snapshot is an imported interface contract, not a Tool implementation, MCP connection or deployment registration.
- Studio may obtain or normalize an external MCP/file import into the supported ToolCatalog input format. Compiler only receives the resulting immutable bytes and verifies their digest.
- Host/Observation Schemas must come from their real Owners. Current PDU source has known missing Host Subject/Facts/Evidence Owner contracts and an underspecified UDM Tool Output Schema; Compiler must report the gaps.
- Host package loading and Runtime admission are separate from Compiler dependency resolution.
- BuildResult.success means only static compilation and packaging succeeded. It does not prove Python execution, MCP availability, MAS readiness or live 5GC E2E.

## 6. Compatibility

The current v0 Worker still accepts `{files,targets}` and the historical `skills/** + lib/**` input layout. It remains unchanged until the breaking v0.1 Go/WASM migration is implemented and browser-verified. Studio must not silently send v0.1 requests to v0 binaries or create fake `lib/` / `output/` folders to bridge the versions.
