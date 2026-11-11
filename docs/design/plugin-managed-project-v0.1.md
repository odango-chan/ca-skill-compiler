# Plugin-managed Skill Project & Tool Integration Contract v0.1

**Design status: FROZEN FOR IMPLEMENTATION, not implemented.** Frozen on 2026-11-11 as a **breaking v0.1 successor** to the existing compiler v0 `skills/lib/output` input conventions. The active Go Core, WASM bridge, CLI, Studio OPFS importer and Procedure Link v0 still implement the older contract. In particular, running current `Check/Build/InspectLibraries` does **not** test this v0.1 contract. No migration or merge of ongoing implementation PRs is implied.

## Ownership, required files and stores

- Skill **source** is the project-owned `skills/<skill>/**` tree. No mandatory `lib/`, `output/`, `skill-project.yaml` or `skill.lock.yaml`.
- Skill Studio's VS Code-compatible **Extension Configuration** is the author-facing project configuration. The extension contributes `connectionAgent.skillStudio.skillsRoot` (workspace, fixed default `skills`), `connectionAgent.skillStudio.dependencies` (workspace, structured), `connectionAgent.skillStudio.dependencyLock` (workspace, managed), and `connectionAgent.skillStudio.defaultBuildTarget` (user, `all`). The last key is a convenience preference, not a source of build semantics.
- Use `workspace.getConfiguration(section, resourceUri)` and explicit `ConfigurationTarget.WorkspaceFolder` when available; in a multi-root environment **one selected root** owns the project. Refuse ambiguous roots or workspace-only config with no durable target; do not silently fall back to global configuration. Theia compatibility must be verified in the Browser-Only host.
- User Settings hold ergonomic preferences only; **no dependencies or locks** in user/global settings or `workspaceState`. For portability the workspace configuration file must be persisted and shareable (for VS Code commonly `.vscode/settings.json`; do not promise this physical path for every Theia host).
- A workspace-managed, immutable store persists exact pinned dependency bytes at `.skill-studio/dependencies/<digest>/` **only after a dependency is imported**. The store is an implementation detail, not an authored `lib` root or an identifier visible to the Skill language. Including it in a project export/clone is necessary for offline portability of uploads and MCP snapshots. The store manifest verifies bytes on read. Never write secrets here.
- Extension Storage may cache **rebuildable** content; SecretStorage holds credentials; `workspaceState` holds ephemeral UI state; Build outputs remain an in-memory file tree until explicit export. Exported results are ordinary user-owned files and are never re-ingested automatically.
- A standalone CLI is **not required** to understand VS Code settings: a host adapter or portable input bundle supplies `BuildRequest` (source bytes plus locked snapshots). The CLI must not silently discover live dependencies on build.

## Contract 1 — Configuration / declaration / lock

Machine layout: [project-settings.schema.json](../../contracts/v0.1/project-settings.schema.json). This schema models the **normalized projection** of the three workspace settings; the UI Settings API can store them under separate keys.

`DependencyDeclaration = {id, kind, source}`. `id` is a project-local lowercase kebab-case alias, unique across all declarations. `kind` initially: `tool-catalog` or `schema-bundle`. Source forms:
- `git`: `{type, repository, path, ref}` for a Git HTTPS repository and exact file.
- `upload`: `{type, upload, contentDigest}`, where `upload` is a stable project-local import alias and digest binds the original uploaded bytes.
- `mcp`: `{type, connectionId, tools}`, where `connectionId` is a **user-configured Studio discovery connection alias** and `tools` is an explicit selected set of names returned by `tools/list`; neither is a Runtime Provider Binding nor a secret. Another workstation may have to reconnect for Refresh; locked bytes remain portable.
- `builtin`: `{type, bundleId, version}`, resolved from versioned plugin resources.
- URL/Registry/local-path sources are **not v0.1 required**; additions require a separately versioned Source Adapter contract.

`DependencyLock = {version, entries[]}`. Each entry repeats `id/kind/source` from the declaration and contains `snapshotDigest`, a sorted `files[]` list of `{path,size,sha256}` and optional `resolvedRevision`. Matching is by exact typed declaration value, not just the alias: any source or tool selection change invalidates the entry. Entry IDs and file paths are unique; unequal duplicate IDs are errors. The schema enforces types; Resolver must enforce cross-entry uniqueness and declaration matching. `resolvedRevision` is required for Git and names the immutable full commit SHA; Built-in has its declared exact version. Upload and MCP are pinned primarily by content digest. Do not use branch names or server identity as an immutable version.

State transitions: **Add + confirm** acquires snapshot and commits lock, **LoadLocked/Build** only uses existing valid lock + bytes, **Refresh** only inspects upstream drift, **UpdateLock** explicitly replaces lock and snapshot together. Network failure, missing snapshot, digest mismatch or missing authority is a visible BLOCKED/ERROR, never silent upgrade. Updates to settings and managed store use a staged write + recheck; failed partial write must be reported UNCERTAIN and not treated as a successful lock update.

## Contract 2 — canonical contract and snapshot

Tool normalization schema: [tool-contract.schema.json](../../contracts/v0.1/tool-contract.schema.json).
- An acquired ToolCatalog YAML/JSON, built-in catalog or MCP `tools/list` selection is **adapted** to the canonical Tool Contract shape. One canonical file is named `tool-contract.json` for a `tool-catalog` snapshot; content is UTF-8 **RFC 8785 JCS** serialization of the normalized JSON object. The source/provenance/connection/credentials are **outside** these semantic bytes. A `schema-bundle` retains exact validated source file bytes under relative paths.
- An Operation has `id`, `inputSchema` (JSON Schema object), optional `outputSchema` and `reconcileOutputSchema`, and `effect={classification,authority}`. Unknown output is *absent*, never an invented permissive schema.
- For MCP, map `tools/list` name to `id`, description to description (if supplied), exact returned input/output schemas, and **always** `effect={classification:"UNKNOWN",authority:"UNVERIFIED"}`. MCP annotations or natural-language descriptions are untrusted hints, not verified Effect. Missing/invalid inputSchema is a diagnostic, not an inferred `{}`. No `tools/call` is ever issued by Compiler.
- READ/MUTATION may only be recorded with `authority:"OWNER"` and separate explicit verified owner evidence provided by the ingestion policy; the JSON Schema only checks shape, not trust. Imported `proposed/declared` provenance labels do **not** imply Runtime READY. Duplicate operations in a single canonical contract fail. Across dependencies, only qualified operation identity is valid.
- File bytes are bounded (per file <= 2 MiB, aggregate <= 16 MiB, files <= 512). Normalize each relative path to UTF-8 NFC and `/` separators; segments must match `[A-Za-z0-9_.-]+` but not `.` or `..`; forbid absolute, `\\`, drive prefixes, symlinks, repeated paths and other special files. Reject path collisions after normalization.
- **Exact digest algorithm**: let files be sorted by lexicographic UTF-8 path bytes. For each file take SHA-256 of raw content bytes. Define `preimage = UTF8("skillc/snapshot/v1") || 0x00 || U32BE(number_of_files) || concat_for_each(U32BE(len(path_utf8)) || path_utf8 || U64BE(len(content_bytes)) || SHA256(content_bytes))`. `snapshotDigest = "sha256:" + lowerhex(SHA256(preimage))`. Lock entries include matching per-file size/hash. No timestamps, source URLs, platform line-ending normalization or ordering variance enter this digest. Both Go and WASM MUST validate it before use; future algorithm changes require a version bump.
- Keep a versioned cross-language positive vector, order-permutation vector and tamper/path collision negative vectors as an **implementation gate**. No claim is made here that a new digest implementation/test already exists.

## Contract 3 — Dependency Resolver

Resolver is a host-independent **service contract** with host-specific source IO adapters:

`Resolve(declarations, existingLock, storage)`, `LoadLocked(declarations, lock, storage)`, `Refresh(declarations, lock, sourceAdapter)`, `UpdateLock(declarations, lock, sourceAdapter, storage)`. Return `{status, snapshots, diagnostics, candidateLock?}`, where status is `READY | BLOCKED | ERROR`. Only an explicit UpdateLock/Add confirmation commits a changed lock. The Resolver does acquisition, validation, staging and immutability; it cannot authorize Tool execution. Disallow automatic outbound requests during Check/Build or LSP analysis. A cache hit and byte-identical store must produce identical snapshots.

MCP discovery is for interface acquisition; Provider connection, credentials, admission, implementation validation, `tools/call` and Runtime readiness remain owned by Agent Host/its deployment owner. Studio may hold **discovery-only** connection preferences and Secrets, never claim them to be Runtime Provider bindings.

## Contract 4 — Compiler core / logical references

Wire shape: [build-request.schema.json](../../contracts/v0.1/build-request.schema.json). A `BuildRequest` is `{apiVersion:"skillc.build/v0.1",sourceFiles:[{path,bytes}],resolvedDependencies:[{id,kind,snapshotDigest,files:[{path,bytes}]}],targets?:string[]}` (bytes are base64 **only on JSON wire**; native Core accepts raw bytes). `sourceFiles` contains authored `skills/**` only, excludes `.vscode/**`, managed store and output. A dependency is namespace-qualified by ID and its own relative file paths. No `lib/**` filenames are accepted as authored source or language references.

The Compiler MUST validate snapshot digest, path safety, duplicate identities, referenced file existence and graph closure. It does not read Settings, secrets, disk/network, Git/MCP, clock or Python; it does not execute any Tool. Build output is a deterministic sorted `outputFiles[]` map, not automatically a physical `output/` directory. Diagnostics cite originating source/dependency IDs rather than fabricated paths. Native Go and WASM must agree on results and bytes.

**Exact new Procedure Link v0.1 syntax** (breaking; no legacy fallback):
```yaml
# node or named pre/postcondition step
call:
  dependency: agent-tools
  operation: create-session

# current Skill-owned Schema (unchanged):
spec:
  inputs:
    requestSchema: schemas/establishment-request.schema.yaml
  context:
    schemas:
      subject:
        dependency: host-contracts
        path: schemas/subject.schema.json
```
Use the same `call: {dependency,operation}` in every governed call position. A local Schema path remains a Skill-relative string; an external Schema is the pair `{dependency,path}` with optional `pointer` for a local JSON Pointer. No physical `lib` prefixes or unqualified operation-name guessing. Missing Owner Schema, ambiguous tool ID, missing Effect authority or unsupported bindings produce precise diagnostics, not fabricated schemas. Compile success is **not** runtime qualification or Provider READY.

## Contract 5 — shared LSP context

The Studio Host assembles `ProjectContext={workspaceRoot,configurationRevision,sourceFiles,resolvedDependencies,snapshotIdentity}`, with `snapshotIdentity` derived from stable source revision and dependency snapshot IDs/digests. LSP and Compiler consume the **same** locked bytes; LSP does not independently fetch MCP/Git or invent authority. Editing a Source buffer invalidates that buffer's analysis revision; Update Lock invalidates all dependency-derived diagnostics/completion/definition/reference caches. An LSP Editor hint is not a Compiler PASS.

### ProjectContext snapshotIdentity framing

For deterministic LSP/Compiler cache keys, sort `sourceFiles` by UTF-8 path bytes and `resolvedDependencies` by dependency ID; within each snapshot sort `files` by UTF-8 path bytes. Use the same base64 encoding for wire `bytes` and RFC 8785 JCS serialization of `{sourceFiles,resolvedDependencies}` (no workspace URI, configurationRevision, timestamps or UI selection). Then `snapshotIdentity = "sha256:" + lowerhex(SHA256(UTF8("skillc/context/v1") || 0x00 || UTF8(JCS(payload))))`. The context ID changes on any source/dependency byte change, while `configurationRevision` is a host-local monotonic freshness guard. `workspaceRoot` is a display/lookup URI only, not compiler identity.

Machine envelopes: [resolver-result.schema.json](../../contracts/v0.1/resolver-result.schema.json) and [project-context.schema.json](../../contracts/v0.1/project-context.schema.json). Resolver `READY` means all requested declarations have lock-matching, verified snapshots; `BLOCKED` and `ERROR` cannot be passed as successful Build input. `candidateLock` is proposed new metadata, never an implicit applied write. Semantic checks for full coverage, exact equality and hash verification are required in addition to structural Schema validation.

## Migration and gates

- Current Compiler PR #2/#3, Studio main and the `lib/output` protection PR are **pre-v0.1 implementations**. Treat this document as a **target design contract**, not a release note or compatibility claim; migrate with explicit breaking changes and replace old positive tests.
- Update Compiler request/parser/linker/CLI/WASM together, then Studio configuration contributions, resolver + store, Dependencies and Build views. Build/Export never auto-invoke an external tool. Any bridge from old `lib` to logical dependency is a bounded migration utility, not a permanent compatibility path.
- Machine Schema gate: valid examples and invalid duplicate/unsafe/unknown-effect cases; duplicate ID/operation and cross-entry lock matching require semantic tests beyond JSON Schema.
- Snapshot gate: independently computed Go vs JavaScript digests; reordered input equal; one-byte tamper fail; normalize path/case collision tests.
- Acquisition gate: Upload YAML/JSON, MCP tools/list (no tools/call), Git pinned commit, Built-in; missing MCP output/effect remain UNKNOWN; no credentials in logs or store.
- Studio gate: saved workspace configuration round-trip across reload, isolated multi-root behavior, fresh Browser-Only OPFS, repeat imports without overwrite, unchanged User Settings on project switch; exact two current Example Skills, old demo kept untouched but not mounted.
- Compiler gate: full/target builds, missing Owner schema explicit error, Native/WASM parity, deterministic output, no `lib/output` requirements.
- Runtime boundary gate: no Provider Binding/authorization/READY claims produced by Studio or Compiler.

**Freeze boundary:** normative on object identity, status transitions, digest frame, source/lock/snapshot roles, logical refs and wire shapes. No claim that v0.1 is implemented or tested in the active binaries. Review changing any of these as a new version instead of silent reinterpretation.
