# Skill Compiler Language & Linking Contract v0.1

**Status: DESIGN ONLY — narrowed Compiler contract for the next implementation iteration.** This document replaces the previous over-broad Plugin-managed Project design. It does not describe an implemented API, and it does not change the current v0 implementation or PR #3.

## 1. Purpose and boundary

The Compiler is a deterministic, side-effect-free transformation:

**Skill source files + imported contract files + selected targets → diagnostics + output file tree.**

Compiler responsibilities:
- Parse and validate Skill source.
- Link Procedure, Tool Operation, Script, Predicate and Schema references.
- Validate statically knowable input/output bindings against supplied Schemas.
- Compute selected Skill dependency closure.
- Generate deterministic package files and diagnostics.
- Expose the same semantics through Native Go and Go/WASM.

Compiler does **not** acquire or manage dependencies. It does not read project settings, browse Git, upload files, connect to MCP, resolve registries, persist snapshots, update locks, manage caches, hold credentials, bind Runtime Providers or execute Tools/Scripts.

The Compiler consumes bytes supplied by its caller. The caller owns all filesystem, workspace, network, storage and authorization concerns.

## 2. BuildRequest — one simple in-memory input

The wire contract is [build-request.schema.json](../../contracts/v0.1/build-request.schema.json):

```typescript
type File = { path: string; bytes: Uint8Array };
type BuildRequest = { files: File[]; targets?: string[] };
```

For JSON/WASM transport, Go `[]byte` is represented as a base64 string. Native callers pass raw bytes.

- `files` contains every input byte needed for this compilation: authored Skill files and imported Tool/Owner/Observation contract files.
- `targets` omitted or empty means all Skills; a non-empty list selects those Skills and their transitive Skill dependencies.
- File paths are project-relative **Compiler input paths**, not instructions to read the physical filesystem.
- The accepted namespaces are `skills/**` and `lib/**`. `skills/**` is authored Skill source. `lib/**` is the logical namespace for imported external contracts.
- A Studio Host may construct `lib/**` entries in memory from plugin-managed dependencies. A physical `lib/` directory is **not required** in the user's workspace.
- `output/` is never an input namespace. Build returns `outputFiles`; Studio decides whether and where to export them. No physical `output/` directory is required.

There is no separate `Project`, `DependencyDeclaration`, `ResolvedDependency`, `DependencyLock` or `ProjectContext` object in the Compiler API. Do not add one merely to describe where the bytes came from.

The Compiler validates normalized relative paths, duplicate paths, file/count/aggregate size limits and namespace boundaries before parsing. It never opens a path from the request.

## 3. Imported ToolCatalog contract

The Compiler consumes the existing ToolCatalog file format, described by [tool-catalog.schema.json](../../contracts/v0.1/tool-catalog.schema.json). The format identifier `ca-agent-demo.tool/v1` identifies the file contract; it does **not** make the Compiler dependent on the ca-agent-demo repository.

A ToolCatalog contains:
- `apiVersion: ca-agent-demo.tool/v1`
- `kind: ToolCatalog`
- `metadata.name` and `metadata.owner`
- optional document-local `x-common` Schema fragments
- `spec.contracts[]`, each with a globally unique `operation`, required `effect` and `inputSchema`, and optional `outputSchema` / `reconcileOutputSchema`

The current optional `lib/manifest.yaml` is only lightweight inspection metadata. It may associate an imported library path with `declared`, `proposed` or `unspecified`; omitted status means `unspecified`. It is not a dependency manifest, source locator, version lock or Runtime admission record.

Compiler linking rules:
- Operation IDs are unique across all ToolCatalog files in one BuildRequest. Duplicate IDs are errors; no source-order precedence.
- A Procedure references the exact Operation ID using the existing scalar form, for example `call: demo.read`. No `dependency:` / `operation:` wrapper or source-specific alias is introduced.
- `effect` is a declared static classification (`READ` or `MUTATION`). Compiler checks its presence and shape; this is not proof of Owner authority, Provider compatibility, permission or Runtime READY.
- `adapter`, when present in a catalog, is descriptive metadata. Compiler does not resolve or invoke it.
- Tool and Schema content is parsed only from the supplied file set. Remote `$ref`, network lookup and source discovery are outside Compiler scope.

## 4. Procedure and Schema references

The current Procedure Link grammar remains the baseline:
- Tool calls use `call: <operationId>`.
- `with:` bindings are statically checked against the selected Operation's input Schema.
- Producer references resolve to declared Tool/Script/Procedure outputs or named pre/postcondition step outputs.
- Skill-owned Schemas use Skill-relative paths.
- Imported Owner/Tool/Observation Schemas may use the logical `lib/**` input namespace.
- Missing, ambiguous or unsupported producer/consumer Schemas fail with diagnostics. Compiler must not invent a permissive Schema to make a link succeed.

This contract does not add a general import language, a package namespace resolver or automatic dependency selection. If a future requirement needs qualified operation namespaces or a new Procedure output declaration, propose that as a narrowly scoped language change with positive and negative examples.

## 5. Compiler outputs and determinism

The response shapes are [build-result.schema.json](../../contracts/v0.1/build-result.schema.json) and [library-inspection.schema.json](../../contracts/v0.1/library-inspection.schema.json).

- `Check` and `Build` share the same static analysis. Check returns diagnostics; Build returns diagnostics and output files.
- On an error, Build returns `success: false` and an empty `outputFiles` array.
- On success, output files are sorted by normalized path. Include selected Skill source, generated files and only imported library files required by the selected Skills and their transitive Schema references.
- Diagnostics have stable ordering and identify the input path and source position when available.
- The same normalized file bytes and targets must produce the same diagnostics and output paths/bytes, independent of input array order.
- Native Go and WASM use the same Go Core and must produce equivalent results.

Content digests, lockfiles and cache keys are not part of this contract. Determinism is defined over the actual BuildRequest bytes and targets. A caller may compute hashes for its own persistence/cache needs without making them Compiler inputs.

## 6. Ownership boundaries

**Skill Studio / caller**
- Owns project and Extension Settings.
- Acquires Tool/Schema files from Upload, MCP discovery, Git, built-in examples or any future source.
- Owns source provenance, persistence, refresh/update UX, credentials, storage, cache and export.
- Prepares one BuildRequest by mapping the selected Skill sources and imported contract bytes into the Compiler's logical `skills/**` and `lib/**` paths.
- May use the same prepared bytes for LSP and Compiler analysis. LSP cache identity and invalidation remain Studio concerns.

**Agent Host / Runtime**
- Owns Provider Binding, MCP/HTTP connections, credentials, authorization, Effect admission, implementation validation and Tool execution.
- Independently validates the real implementation and runtime permission. A successful Compiler build is not Runtime readiness.

**Compiler**
- Knows only paths, bytes, Skill semantics, ToolCatalog semantics and selected targets.
- Does not know whether a file was uploaded, discovered through MCP, fetched from Git or bundled with the extension.

## 7. Explicitly out of Compiler v0.1 scope

Do not implement these in this repository:
- Git / Upload / MCP / Built-in Source Adapters.
- Dependency Resolver lifecycle: Add, Refresh, UpdateLock, Remove.
- Project Settings or VS Code Extension Configuration.
- DependencyLock, package registry, version solver or package manager.
- Snapshot Store, snapshot hash algorithm, cache consistency or LSP ProjectContext.
- Provider Binding Registry, connection/authentication or Runtime execution.

These are caller/Studio/Host responsibilities. Compiler may validate the bytes and references it receives, but it does not acquire, persist or manage them.

## 8. Acceptance gates for the Compiler contract

1. Valid BuildRequest with Skill files and virtual `lib/**` files passes; no physical project `lib/` directory is assumed.
2. Invalid namespace, absolute/traversal path, duplicate path, oversized file and duplicate Operation ID fail deterministically.
3. ToolCatalog positive/negative fixtures cover missing Operation, invalid Effect, missing Input Schema, malformed Output Schema, duplicate IDs and unsupported/remote Schema references.
4. Procedure scalar Tool calls link exact Operation IDs; unknown IDs and ambiguous duplicates fail.
5. Input bindings are checked against actual supplied producer/consumer Schemas; missing Owner Schema is an explicit error.
6. Empty targets compile the whole project; explicit targets compile their dependency closure.
7. Same input bytes/targets in different file order produce identical results.
8. Native and WASM produce the same diagnostics, output paths and bytes.
9. No test requires Git, MCP, network access, project settings, lockfiles or a physical `lib/` / `output/` folder.

**Freeze rule:** keep the API small. Any proposal to add source acquisition, dependency lifecycle, storage or cache management must be rejected from the Compiler contract and handled by the caller/Studio.
