# Skill Compiler Language & Linking Contract v0.1

**Status: DESIGN ONLY.** This is the Compiler-only implementation contract. It does not claim v0.1 is implemented. The language authority is the current **ca-skills Skill Language v0.2**; this document defines how Compiler consumes and compiles that language, not a competing DSL.

## 1. Purpose and boundary

The Compiler is a deterministic, side-effect-free transformation:

**caller-supplied Skill files + imported contract files + selected targets → diagnostics + output file tree.**

Compiler responsibilities:

- Parse the current Skill Language.
- Build Procedure and Predicate syntax structures.
- Link Skill, Procedure, Tool Operation, Script and Schema references.
- Validate statically knowable Producer/Consumer bindings.
- Generate Script Output JSON Schemas from strict Output TypedDict declarations.
- Verify supplied dependency snapshot digests and exact exported contract identities.
- Compute the exact dependency snapshot closure used by selected Skills.
- Return deterministic Skill build files, dependency closure metadata and diagnostics.
- Expose identical semantics through Native Go and Go/WASM.

Compiler does **not** acquire, discover, persist or manage dependencies. It does not read project settings, browse Git, upload files, connect to MCP, resolve registries, update locks, manage caches, hold credentials, bind Runtime Providers or execute Tools/Scripts.

The Compiler receives bytes supplied by its caller. Studio/caller owns filesystem, workspace, network, storage, import, authorization and export concerns.

## 2. Language authority: ca-skills v0.2

The normative source contracts are:

- [Skill Language Specification v0.2](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/skill-language-spec.md)
- [Script Contract v0](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/script-contract.md)
- [Schema Ownership](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/schema-ownership.md)
- [Authoring Model](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/authoring-model.md)
- [Current 5GC source baseline](https://github.com/odango-chan/ca-skills/blob/codex/semantic-token-contract-v01/docs/current-5gc-skill-baseline.md)

ca-skills is the **only authoritative input for official Skill source and Skill-authored language semantics**. ca-agent-demo may be consulted to refresh business behavior or locate the owner of an imported contract, but Compiler and Studio must not treat it as a second Skill source tree or fetch it during compilation.

The current official source examples are skills/establish-pdu-session/ and skills/establish-pdu-session-mas/ from ca-skills. Historical Skill directories are not current v0.2 grammar authorities.

## 3. BuildRequest — one simple in-memory input

The wire contract is [build-request.schema.json](../../contracts/v0.1/build-request.schema.json):

~~~typescript
type File = { path: string; bytes: Uint8Array };
type ResolvedDependency = {
  dependencyId: string; // Workspace alias / reference namespace
  digest: `sha256:${string}`; // input integrity check only
  files: Array<{ path: string; bytes: Uint8Array }>;
};
type BuildRequest = {
  files: File[];                         // Skill source only: skills/**
  resolvedDependencies: ResolvedDependency[];
  targets?: string[];
};
~~~

For JSON/WASM transport, Go []byte is represented as a base64 string. Native callers pass raw bytes.

- `files` contains authored Skill source only. Every path is rooted in `skills/**`.
- `resolvedDependencies` contains immutable, already-resolved snapshots supplied by Studio. Each dependency has a stable Workspace Configuration identity, a canonical content digest, and its own snapshot-relative file set. No `lib/` prefix or workspace path is used.
- `dependencyId` is an opaque stable alias from Studio's resource-scoped Workspace Configuration. Skill source may use it only as a logical namespace to qualify an exported contract reference (`<alias>::<contract-id>`); it is not a source URL, repository, revision, digest, cache key or path.
- Exported contracts are discovered and linked from the supplied snapshot contents: exact Operation IDs from ToolCatalogs, Capability IDs and their Operation membership from the independently owned CapabilityContracts, and stable Schema IDs from Schema `$id`. No separate dependency manifest or exports list is introduced.
- Duplicate dependency IDs, duplicate paths within one snapshot, digest mismatches, duplicate/ambiguous exported contract identities, missing exports and incompatible schemas are errors. Source order must never select a winner.
- `targets` omitted or empty means all Skills; a non-empty list selects those Skills and their transitive Skill dependencies.
- All file paths are normalized relative paths inside their own namespace/snapshot. They are data labels, never instructions to read the physical filesystem.
- `output/` is never an input namespace. Build returns `outputFiles`; Studio decides whether and where to export them. No physical output directory is required.

There is no Compiler-side Project, Dependency Resolver, DependencyLock, DependencyManifest, SnapshotStore, cache or ProjectContext lifecycle. Studio owns acquisition, source selection, revision pinning, lock state, immutable persistence, refresh/Update Lock, export and cache. Compiler only verifies the supplied snapshot bytes/digests and links their exported contracts.

The request is bounded by the existing limits: at most 512 files across Skill source and all dependency snapshots, 128 dependency snapshots, 2 MiB per file and 16 MiB aggregate raw bytes; duplicate paths are rejected within each namespace/snapshot. Compiler validates normalized relative paths, duplicate IDs/paths, digest integrity, aggregate size limits and namespace boundaries before parsing. It never opens a path from the request.

Canonical snapshot digest is specified in [dependency-snapshot-digest-v0.1.md](dependency-snapshot-digest-v0.1.md). The Studio and Compiler implementations must pass the same golden vectors before this contract is considered implemented.

## 4. Current Skill Language v0.2 semantics

Compiler must implement the source contract; it must not silently flatten unsupported constructs or restore legacy compatibility paths.

### 4.1 SkillContract and Schema ownership

- spec.inputs.requestSchema identifies the Skill-owned public Request Schema.
- spec.context.requiredFacts is a list of fact identifiers, **not** a field-level Schema.
- spec.context.schemas explicitly identifies Owner Schemas for subject, facts, observations and evidence when those context namespaces are used.
- spec.allowedResources.capabilities[] is the resource-grant upper bound. It may contain deployment Capability IDs or direct Operation IDs. It is not a Script allowlist.
- allowedResources.localTools does not exist in v0.2. A Script is a Skill-owned source asset, not an independently invocable Tool.

Schema ownership:

| Data | Schema owner | Compiler source |
|---|---|---|
| Public Request | Skill | spec.inputs.requestSchema |
| Tool Operation Input/Output/Reconcile | Tool owner | imported ToolCatalog in resolved dependency snapshots |
| Subject / Facts / Evidence | Agent Host owner | explicit Owner Schema references resolved from dependency snapshots |
| Observation payload | Observation owner | explicit Owner Schema references resolved from dependency snapshots |
| Procedure Script input | Derived per invocation | with: Producer bindings |
| Predicate Script input | Host + Observation owners | Predicate Evaluation Context |
| Script output | Skill Script | strict Output TypedDict; generated by Compiler |

Missing or non-authoritative Owner Schemas are explicit errors. requiredFacts, arbitrary object declarations, old Script input schemas and deployment configuration files are not substitutes.

### 4.2 Procedure

The supported main-node intents are exactly:

- call: invoke one exact imported Tool Operation.
- script: run a Skill-owned Script by package-relative path.
- procedure: invoke a local Procedure or explicitly identified cross-Skill Procedure.
- verify: evaluate one Predicate.
- reconcile: reconcile the exact original mutation node.
- terminal: end with a declared Procedure terminal result.

call identifies one exact Operation exported by a dependency snapshot, qualified by its configured dependency alias when required to disambiguate sources, for example:

~~~yaml
call: fivegc-tools::sbi.udm.sm_data.get
~~~

It is not a Capability group or MCP display name. The resolved Operation must belong to exactly one imported Capability contract. The Skill grants the corresponding Capability identity in `allowedResources.capabilities`; Compiler verifies that every called Operation is a member of a granted Capability. A Tool Operation is never treated as an ungrouped standalone permission.

Preconditions and postconditions are ordered node-entry/node-result phases. Their named steps support only call, script and verify; nested procedure and tool steps are unsupported. Unknown execution-affecting fields fail explicitly.

with bindings are explicit. Each binding has exactly one of from or value. Procedure.inputs.<name>.type is only an author assertion checked against the real Producer Schema; it cannot create or replace a missing Schema. No implicit previous-node binding, dynamic selector, interpolation or automatic Skill routing exists.

### 4.3 PredicateSet

An evaluator is either:

- type: cel with expression for an already-materialized evidence expression; or
- type: script with ref: scripts/foo.py for a Skill-owned Predicate Script.

evaluator.ref is a package-relative Script path, not a Tool ID. Compiler validates source shape and references; it does not execute CEL, acquire Observations or decide PASS/FAIL/UNKNOWN.

### 4.4 Script Contract

The only supported entry point is:

~~~python
def main(input: dict) -> OutputType:
    ...
~~~

- The parameter is named input and annotated exactly dict; this is only the transport container, not an open Input Schema.
- Procedure Script input is derived separately at each invocation from that step's with: Producer bindings.
- Predicate Script input is the Host-created Predicate Evaluation Context: subject, validated request, facts, declared/materialized observations and evidence.
- The return annotation names a non-empty TypedDict declared statically in the same file.
- Supported output declarations include primitive scalar types, nested TypedDict, functional TypedDict("Name", {...}) for wire keys such as 5qi, list[T], Literal, NotRequired and supported literal Annotated JSON Schema constraints.
- dict, Any, object, empty Output types, unsupported unions, dynamic constraints and inferred function-body schemas are forbidden.
- Compiler generates a read-only Draft 2020-12 Output Schema in BuildResult at skills/<skill>/schemas/generated/<script-path-with-scripts-prefix-and-py-extension-removed>.output.schema.json; e.g. scripts/common/normalize.py becomes schemas/generated/common/normalize.output.schema.json under that Skill. It does not generate an author-editable Script Input Schema.
- The old # /// connectionagent-tool metadata, independent local Tool IDs and external Script Input/Output Schema compatibility paths are not supported.
- Compiler never executes Python. Runtime owns execution, sandbox and actual I/O validation.

The five conventional PDU Scripts in the official ca-skills source are the primary migration examples. PcfPolicyInputsOutput uses functional TypedDict for the JSON wire key 5qi.

## 5. Imported ToolCatalog and read-only inspection

The current imported format is the ToolCatalog file contract described by [tool-catalog.schema.json](../../contracts/v0.1/tool-catalog.schema.json).

A ToolCatalog is one file within a resolved dependency snapshot and contains:

- apiVersion: ca-agent-demo.tool/v1
- kind: ToolCatalog
- metadata.name and metadata.owner
- optional document-local x-common Schema fragments
- spec.contracts[], each with a globally unique operation, required effect and inputSchema, and optional outputSchema / reconcileOutputSchema

The ca-agent-demo.tool/v1 string is a file-format identifier only; it does not make Compiler depend on or fetch the ca-agent-demo repository.

Linking rules:

- Operation IDs are unique across all ToolCatalog files in all supplied dependency snapshots in one BuildRequest. Duplicate IDs are errors; there is no source-order precedence.
- effect is a declared static classification (READ or MUTATION), not proof of Owner authority, permission, implementation or Runtime READY.
- adapter is descriptive metadata. Compiler does not resolve or invoke it.
- YAML anchors/aliases and document-local Schema fragments are supported with bounded expansion.
- Local references are resolved within their containing contract/snapshot according to the contract format. Stable exported Schema `$id` references may be resolved across supplied snapshots only by exact identity. Remote `$ref`, network lookup and source discovery are outside Compiler scope.
- inspectDependencies may return a read-only projection of imported Operation IDs, dependency IDs, snapshot-relative source paths, Effects and Schemas for Studio display. It does not maintain source maturity, package status, manifests or lifecycle state.

There is **no** Compiler-side `lib/manifest.yaml`, sourceStatus, package manifest, dependency resolver or lifecycle state. Dependency aliases and snapshot digests are normalized inputs from Studio's locked Workspace Configuration; aliases qualify exported contract references, while digests verify input integrity. Neither encodes source acquisition or permission. If Studio needs labels such as imported or read-only, those are Studio presentation metadata, not Compiler input.

The conventional 5GC ToolCatalog is supplied as a resolved dependency snapshot from the approved Tool Owner source. MAS remains excluded from executable acceptance until an approved MAS ToolCatalog and runnable end-to-end evidence exist.

The current UDM ToolCatalog Output Schema is broad (result: {type: object}) and does not declare nested subscription fields used by the selection Script. Compiler must report the missing nested Producer Schema information; it must not infer it from the Script or restore a Skill-owned Input Schema.

## 6. Procedure and Schema linking

Compiler must perform structural, scope-aware linking rather than recursively scanning YAML keys.

Supported data sources include:

- context.request.*
- context.subject.*
- context.facts.*
- context.observations.*
- context.evidence.*
- input.*
- <node>.output.*
- preconditions.<step>.output.*
- postconditions.<step>.output.*
- <node>.preconditions.<step>.output.*
- <node>.postconditions.<step>.output.*

Rules:

1. Resolve each source path against the exact authoritative Producer Schema.
2. Validate nested object properties, arrays, required fields, nullability, scalar types and supported constraints.
3. Validate with literals against the exact consumer Schema.
4. Keep named precondition/postcondition step scopes separate.
5. Resolve cross-Skill Procedure references only by explicit Skill + Procedure identity.
6. Resolve reconcile to the exact earlier mutation node and its declared reconcile Output Schema.
7. Resolve local JSON Pointer references only within the supplied input file set.
8. Missing, ambiguous, unknown or unsupported Schema semantics produce precise diagnostics. Do not accept an unknown nested field merely because its parent says type: object.
10. Do not infer Owner Schema from Python access patterns, Case fixtures, requiredFacts or deployment routing profiles.

Current known external blockers in the official PDU source are the Host Subject/Facts/Evidence and Observation Owner Schemas, plus the broad UDM Tool Output Schema described above. MAS has 16 unresolved Operation references. These are expected explicit diagnostics, not reasons to weaken the linker.

## 7. Compiler outputs and determinism

The proposed response shape is [BuildResult v0.1](../../contracts/v0.1/build-result.schema.json). Its dependencyClosure reports used dependency aliases only; snapshot digests are verified BuildRequest inputs and are not exported. DependencyInspection remains a separate inspection response.

- check and build share the same static analysis.
- On any ERROR, Build returns success: false with empty dependencyClosure and outputFiles arrays.
- On success, `outputFiles` contains selected Skill source and generated Skill files only, sorted by normalized Skill path. Imported dependency files are not copied into this tree.
- `dependencyClosure` contains the unique dependency aliases actually used by selected Skills and their transitive Schema/Operation/Capability references, sorted by alias. It does **not** export snapshot digests; lock and digest metadata remain Studio-owned. BuildResult is a compilation result, not a dependency-lock export.
- Diagnostics have stable ordering and identify the actual input path and source position when available.
- Identical normalized Skill bytes, dependency aliases, verified dependency snapshot contents and targets produce identical diagnostics, dependencyClosure and output paths/bytes, independent of input array order.
- Native Go and WASM use the same Go Core and must produce equivalent results.
- Dependency content digests are Compiler inputs and are verified. Source acquisition, lock mutation, cache keys, lock lifecycle and cache storage remain Studio-owned.

## 8. Ownership boundaries and explicit exclusions

**Skill Studio / caller**
- Owns project files and editor state.
- Imports files from Upload, MCP discovery, Git or built-in examples.
- Owns resource-scoped dependency declarations, source provenance, revision pins, locks, immutable snapshot persistence, refresh/Update Lock, credentials, storage and export.
- Prepares Skill source files plus resolved immutable dependency snapshots for the Compiler.
- Computes the canonical snapshot digest using the Compiler-owned algorithm and validates the Workspace Configuration lock.
- May use the same pinned snapshots for LSP and Compiler analysis.

**Agent Host / Runtime**
- Owns Provider bindings, connections, credentials, authorization, Effect admission, implementation readiness, Python execution and Tool execution.

**Compiler**
- Knows Skill-source paths, snapshot-relative dependency paths, bytes, dependency aliases, Capability-to-Operation membership, ToolCatalog semantics and selected targets. It verifies input snapshot integrity but does not export digests in BuildResult.
- Does not know whether a file was uploaded, discovered through MCP, fetched from Git or bundled with Studio.

Explicitly out of scope: Git/Upload/MCP adapters, dependency resolver lifecycle, Project Settings, package manifests, DependencyLock mutation, version solvers, registry, Snapshot Store, cache lifecycle, Provider Binding, Runtime admission, Qualification, Release, Python execution and live 5GC testing. Canonical snapshot digest definition and verification are in scope; digest generation for Workspace Configuration is implemented by Studio.

## 9. Acceptance gates

1. BuildRequest accepts Skill files under skills/** plus separate resolved dependency snapshots; no lib/** namespace or physical lib/ directory is required.
2. Invalid namespace, absolute/traversal path, duplicate path, oversized file and duplicate Operation ID fail deterministically.
3. ToolCatalog positive/negative fixtures cover aliases, malformed Operations, missing Effect/Input Schema, duplicate IDs and unsupported/remote Schema references.
4. Current v0.2 Procedure and Predicate constructs parse according to ca-skills; unsupported actions fail explicitly.
5. Script Contract v0 is statically analyzed and Output TypedDict schemas are generated without running Python.
6. Procedure and Predicate Script input projections link only to real Producer/Owner Schemas.
7. Missing Host/Observation/Tool nested Output Schemas produce explicit diagnostics.
8. Empty targets compile the whole supplied Skill project; explicit targets compile their Skill dependency closure and report the exact used dependency snapshot closure.
9. Same Skill bytes, snapshot file bytes, dependency IDs/digests and targets in different input array orders produce identical results.
10. Native and WASM produce the same diagnostics, output paths and bytes.
11. No Compiler test requires Git, MCP, network access, dependency acquisition, lock management, or physical lib/ / output/ folders. Studio integration tests separately prove Workspace Configuration lock-to-snapshot assembly.

**Freeze rule:** keep the API small. Any proposal to add source acquisition, dependency lifecycle, storage, manifests, locks or cache management belongs to Studio/caller and must be rejected from the Compiler contract.


## 10. Capability is the permission boundary; Operation is the callable member

A ToolCatalog Operation is not an independent authorization unit. The imported, independently owned CapabilityContracts define the exact mapping:

`Capability ID -> owner -> { Operation ID, expected Effect }`.

Each ToolCatalog Operation supplies its callable contract (input/output/reconcile Schema, declared Effect, adapter binding). CapabilityContracts independently groups that Operation under one Capability and supplies the expected Effect for cross-check. The two sources must agree exactly; missing membership, duplicate membership, mismatched Effect or unknown Operation is a hard linking/admission diagnostic.

SkillContract `spec.allowedResources.capabilities[]` grants Capability identities, not arbitrary individual Tool calls. In vNext, both the grant and the Procedure call use the same dependency-alias namespace (for example `fivegc-tools::provider-5gc.createdsession.udm` and `fivegc-tools::sbi.udm.sm_data.get`). Compiler must prove the called Operation belongs to exactly one Capability in that resolved dependency snapshot and that the qualified Capability is granted by the Skill. It must not infer or synthesize a Capability from an Operation name. Existing 5GC source currently lists Operation IDs in this field as a compatibility-era representation; migration to actual Capability IDs requires a separately reviewed language change and must preserve the exact 5GC operation coverage.

Observation acquisition follows the same model: a read-only Observation provider is exposed as a READ Operation, and that Operation belongs to a Capability. Predicate declares the Observation/fact it consumes; the Runtime obtains it through the registered READ Operation/source. Observation is not a parallel unowned tool namespace.

## 11. Digest output is not part of the Compiler's current deliverable

Compiler still verifies the supplied snapshot digest on BuildRequest input because Studio's immutable lock and the frozen snapshot-integrity contract require it. This is input validation only.

Compiler does not need to export digest values in BuildResult. The proposed v0.2 BuildResult returns the dependency aliases actually used, diagnostics and generated Skill files. Studio remains the owner of lock entries, source revision, snapshot persistence and digest display/export. Do not duplicate or regenerate the dependency lock from BuildResult.
