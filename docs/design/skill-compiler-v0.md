> **Historical compatibility design.** This document describes the current v0 `{files,targets}` + `skills/**` / `lib/**` implementation. It is superseded as the target architecture by [Compiler v0.1 Contract](compiler-contract-v0.1.md): Skill source files and Studio-resolved immutable dependency snapshots are separate BuildRequest inputs; no physical or global `lib/**` namespace is required. Keep this document only to explain the existing v0 behavior until the breaking v0.1 migration is implemented.

# Skill Compiler · Product and Architecture Brief

**Status:** Product boundary accepted; v0 implementation remains partial. The normative v0.1 contract is [compiler-contract-v0.1.md](compiler-contract-v0.1.md). The normative Skill language is owned by [ca-skills](https://github.com/odango-chan/ca-skills).

## 1. Product goal

Skill Compiler is a small, deterministic, stateless compile-and-link tool, similar in role to GCC:

**Skill source + caller-supplied read-only contracts + selected Skill targets → diagnostics + a copyable file tree.**

Native CLI writes the file tree to a directory. Skill Studio can display the output and export a ZIP or copyable folder. Browser/Pages uses the same Go Core compiled to WASM; no backend compiler service or database is required.

## 2. Project-facing logical layout

~~~text
project/
  skills/       # multiple editable official Skill source packages
  lib/          # imported read-only Tool/Host/Observation contracts
  output/       # virtual/read-only BuildResult projection; never a build input
~~~

skills/ is the only Skill source root. lib/ is an input namespace for imported contracts; Studio may build it in memory from a file import or other source. output/ is generated output and must never flow back into the next compilation.

Compiler does not require these three physical directories. Its only input is a file-byte map containing skills/** and lib/**; its output is a deterministic file map.

## 3. Ownership

- ca-skills is the only official Skill source and Skill-language authority.
- Skill Studio owns project Explorer, source editing, external file import, read-only Library display, user authorization, persistence, refresh, ZIP/folder export and UI.
- Compiler owns the unique Skill parser, AST, reference linker, static Schema/binding validation, Script Output Schema generator, diagnostics and BuildResult.
- Agent Host/Runtime owns Python execution, sandbox, Tool authorization, implementation readiness and Tool execution.

Compiler does not acquire dependencies, connect to Git/MCP/Upload, maintain manifests/locks/caches, manage databases, execute Python/Tools, run Qualification or Release, or claim Runtime/5GC readiness.

## 4. Source and generated material

| Material | Owner | Compiler behavior |
|---|---|---|
| SKILL.md, Contract, Procedure, Predicate | Skill author | Parse and validate according to ca-skills v0.2 |
| Public Request Schema | Skill | Resolve from spec.inputs.requestSchema |
| Python Script | Skill | Static parse only; never execute |
| Procedure Script input | Producer + with: | Derive per invocation; no source Input Schema |
| Script Output Schema | Script Output TypedDict | Generate read-only BuildResult file |
| Tool Operation Schemas/Effect | Tool owner | Consume imported ToolCatalog |
| Host/Observation Schemas | Actual Owners | Consume explicit imported Owner files; missing files remain errors |
| Cases | Skill author | Preserve as source obligations; do not run |
| References | Skill author | Preserve as optional model knowledge; not executable/evidence |

No compatibility fallback for old Script metadata or duplicate Script Input/Output Schema files is provided.

## 5. Library input

Compiler accepts the current ToolCatalog contract as an imported file under logical lib/**. It may expose a read-only Operation/Schema projection for Studio, but does not maintain lib/manifest.yaml, source maturity states, dependency identities, lockfiles or import lifecycle.

The current ca-skills library contains one approved 12-Operation conventional PDU ToolCatalog. The MAS Skill's 16 Operation references remain unresolved without an approved MAS catalog. Host Subject/Facts/Evidence and Observation Owner Schemas are also not bundled. The UDM Tool Output Schema is currently too broad for nested subscription field linking. Compiler must report these exact gaps; no fallback or fabricated Schema is allowed.

The ca-agent-demo.tool/v1 string is the imported ToolCatalog file-format identifier only. Compiler does not fetch ca-agent-demo and does not use it as a second Skill source.

## 6. One Core API

~~~typescript
type File = { path: string; bytes: Uint8Array };
type Diagnostic = {
  severity: "error" | "warning" | "info";
  code: string;
  path: string;
  line?: number;
  column?: number;
  message: string;
};
type BuildRequest = { files: File[]; targets?: string[] };
type BuildResult = {
  success: boolean;
  diagnostics: Diagnostic[];
  outputFiles: File[];
};
~~~

check and build share one static analysis implementation. Native CLI and WASM are thin adapters over the same Go Core. No database, HTTP backend, ProjectContext, manifest, dependency resolver or digest protocol is part of this API.

outputFiles is not ZIP bytes. CLI writes it to --out; Studio decides where to save it and whether to wrap it in ZIP.

## 7. Implementation and acceptance

Implementation must follow the ca-skills v0.2 language and the Compiler v0.1 Build API. Acceptance includes parser/AST coverage, exact Tool/Predicate/Script/Schema linking, per-call Script input projections, Output TypedDict generation, multi-Skill closure, deterministic output, Native/WASM parity and real Chromium Worker tests.

The current raw/legacy ca-agent-demo source is not a second compile input and is not required to pass without migration. The current official ca-skills packages are the compile inputs. Missing external Owner contracts must be reported rather than invented.

Compiler success means only static compilation and packaging completed. It does not mean Script execution, Runtime READY, Qualification, release or live 5GC acceptance.
