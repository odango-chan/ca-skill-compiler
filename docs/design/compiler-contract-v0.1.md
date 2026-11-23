> **HISTORICAL / NOT CURRENT INPUT (v2.1.0).** Retained at its original path as frozen older-version or design evidence, not an active BuildRequest, syntax or CLI target. The current Compiler accepts **only CA-SL v1alpha5**; v1alpha4 inherited *semantics* remain implemented, but v1alpha4 API inputs are rejected. See [documentation](../README.md), [Development contracts](../../contracts/development/v1alpha1/README.md) and [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0). Statements below about old implementation states are historical. <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL — SUPERSEDED FOR NEW DEVELOPMENT.** This v0.1 design targets the former Skill Language v0.2 contract. Do not use it to implement CA-SL v1alpha4. The active contract is [Compiler Contract v1alpha4](compiler-contract-v1alpha4.md), with [BuildRequest v1alpha4](build-request-v1alpha4.md) and the [machine contracts](../../contracts/v1alpha4/README.md). This file is retained only as design history.

---

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
- Compute selected Skill dependency closure.
- Return deterministic build files and diagnostics.
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
type BuildRequest = { files: File[]; targets?: string[] };
~~~

For JSON/WASM transport, Go []byte is represented as a base64 string. Native callers pass raw bytes.

- files contains every input byte needed for this compilation: official Skill source and imported Tool/Host/Observation Owner contracts.
- targets omitted or empty means all Skills; a non-empty list selects those Skills and their transitive Skill dependencies.
- File paths are project-relative Compiler input paths, not instructions to read the physical filesystem.
- Accepted namespaces are skills/** and lib/**. skills/** is authored Skill source. lib/** is the logical namespace for caller-supplied imported contracts.
- A Studio Host may assemble lib/** entries in memory from files, MCP import results or built-in contracts. Compiler does not know or care where the bytes came from. A physical lib/ directory is not required in the user's workspace.
- output/ is never an input namespace. Build returns outputFiles; Studio decides whether and where to export them. No physical output/ directory is required.

There is no separate Project, DependencyDeclaration, ResolvedDependency, DependencyLock, LibraryManifest, SnapshotStore or ProjectContext object in the Compiler API.

Compiler validates normalized relative paths, duplicate paths, file/count/aggregate size limits and namespace boundaries before parsing. It never opens a path from the request.

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
| Tool Operation Input/Output/Reconcile | Tool owner | imported ToolCatalog under lib/** |
| Subject / Facts / Evidence | Agent Host owner | explicit spec.context.schemas.* paths |
| Observation payload | Observation owner | explicit spec.context.schemas.observations path |
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

call is always a scalar Operation ID, for example:

~~~yaml
call: sbi.udm.sm_data.get
~~~

It is not a Capability group, dependency name, MCP tool name or dependency/operation object.

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

A ToolCatalog contains:

- apiVersion: ca-agent-demo.tool/v1
- kind: ToolCatalog
- metadata.name and metadata.owner
- optional document-local x-common Schema fragments
- spec.contracts[], each with a globally unique operation, required effect and inputSchema, and optional outputSchema / reconcileOutputSchema

The ca-agent-demo.tool/v1 string is a file-format identifier only; it does not make Compiler depend on or fetch the ca-agent-demo repository.

Linking rules:

- Operation IDs are unique across all ToolCatalog files in one BuildRequest. Duplicate IDs are errors; there is no source-order precedence.
- effect is a declared static classification (READ or MUTATION), not proof of Owner authority, permission, implementation or Runtime READY.
- adapter is descriptive metadata. Compiler does not resolve or invoke it.
- YAML anchors/aliases and document-local Schema fragments are supported with bounded expansion.
- Only local references resolvable from the supplied file set are accepted. Remote $ref, network lookup and source discovery are outside Compiler scope.
- inspectLibraries may return a read-only projection of imported Operation IDs, source paths, Effects and Schemas for Studio display. It does not maintain source maturity, package status, manifests or lifecycle state.

There is **no** lib/manifest.yaml, sourceStatus, package manifest or dependency state in v0.1. If Studio needs labels such as imported or read-only, those are Studio presentation metadata, not Compiler input.

Current ca-skills lib/ contains the approved 12-Operation conventional PDU ToolCatalog. The MAS Skill's 16 ca.pdu_session.* references remain unresolved until an approved MAS ToolCatalog is supplied. Its design draft is not an approved import.

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
9. Do not infer Owner Schema from Python access patterns, Case fixtures, requiredFacts or deployment routing profiles.

Current known external blockers in the official PDU source are the Host Subject/Facts/Evidence and Observation Owner Schemas, plus the broad UDM Tool Output Schema described above. MAS has 16 unresolved Operation references. These are expected explicit diagnostics, not reasons to weaken the linker.

## 7. Compiler outputs and determinism

The response shapes are [build-result.schema.json](../../contracts/v0.1/build-result.schema.json) and [library-inspection.schema.json](../../contracts/v0.1/library-inspection.schema.json).

- check and build share the same static analysis.
- On any ERROR, Build returns success: false and an empty outputFiles array.
- On success, output files are sorted by normalized path. Include selected Skill source, generated files and only imported contracts required by selected Skills and their transitive Schema references.
- Diagnostics have stable ordering and identify the actual input path and source position when available.
- Identical normalized file bytes and targets produce identical diagnostics and output paths/bytes, independent of input array order.
- Native Go and WASM use the same Go Core and must produce equivalent results.
- Content digests, lockfiles and cache keys are not Compiler inputs.

## 8. Ownership boundaries and explicit exclusions

**Skill Studio / caller**
- Owns project files and editor state.
- Imports files from Upload, MCP discovery, Git or built-in examples.
- Prepares the skills/** and lib/** byte set.
- Owns source provenance, persistence, refresh, credentials, storage and export.
- May use the same prepared bytes for LSP and Compiler analysis.

**Agent Host / Runtime**
- Owns Provider bindings, connections, credentials, authorization, Effect admission, implementation readiness, Python execution and Tool execution.

**Compiler**
- Knows only paths, bytes, Skill semantics, ToolCatalog semantics and selected targets.
- Does not know whether a file was uploaded, discovered through MCP, fetched from Git or bundled with Studio.

Explicitly out of scope: Git/Upload/MCP adapters, dependency resolver lifecycle, Project Settings, package manifests, DependencyLock, version solvers, registry, Snapshot Store, digest protocol, caches, Provider Binding, Runtime admission, Qualification, Release, Python execution and live 5GC testing.

## 9. Acceptance gates

1. BuildRequest accepts skills/** plus caller-supplied lib/**; no physical lib/ directory is assumed.
2. Invalid namespace, absolute/traversal path, duplicate path, oversized file and duplicate Operation ID fail deterministically.
3. ToolCatalog positive/negative fixtures cover aliases, malformed Operations, missing Effect/Input Schema, duplicate IDs and unsupported/remote Schema references.
4. Current v0.2 Procedure and Predicate constructs parse according to ca-skills; unsupported actions fail explicitly.
5. Script Contract v0 is statically analyzed and Output TypedDict schemas are generated without running Python.
6. Procedure and Predicate Script input projections link only to real Producer/Owner Schemas.
7. Missing Host/Observation/Tool nested Output Schemas produce explicit diagnostics.
8. Empty targets compile the whole supplied project; explicit targets compile their Skill dependency closure.
9. Same input bytes/targets in different file order produce identical results.
10. Native and WASM produce the same diagnostics, output paths and bytes.
11. No test requires Git, MCP, network access, project settings, package manifests, lockfiles or physical lib/ / output/ folders.

**Freeze rule:** keep the API small. Any proposal to add source acquisition, dependency lifecycle, storage, manifests, locks or cache management belongs to Studio/caller and must be rejected from the Compiler contract.
