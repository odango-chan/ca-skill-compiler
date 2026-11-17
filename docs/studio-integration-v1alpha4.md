# Skill Studio ↔ Skill Compiler v1alpha4 Integration

**Status:** FROZEN target integration contract; not a claim that the current Studio build already implements it.

## 1. Ownership

- **ca-skills** owns official Skill source and the only normative CA-SL language specification.
- **Skill Studio** owns project/workspace state, source editing, dependency selection/acquisition, trusted local configuration, immutable snapshot persistence, lock values, Problems UI and directory/ZIP export.
- **Compiler** owns the only parser/AST/linker/static checker/Script type derivation/Artifact generator.
- **Tool Owners** own ToolCatalog and all callable Input/Query/Output/Receipt/Evidence contracts.
- **Deployment / trusted configuration owner** owns CapabilityContracts, Scope policies and publisher authorization.
- **Host / Runtime** owns admission, actual invocation, authorization revalidation, Python sandbox, Evidence records, commit settlement, Trace and resource responsibility.

Studio MUST NOT implement a second semantic parser, weaker Schema checker or alternative SourcePath resolver.

## 2. Workspace and input assembly

A project may contain author Skill files and Studio-managed dependency configuration. It does not need physical lib/ or output/ directories.

Studio assembles:
- Skill source files into BuildRequest files;
- immutable imported dependency bytes into resolvedDependencies;
- authenticated dependency/Owner attribution and deployment CapabilityContracts into trustedBuildContext;
- selected Skill package names into optional targets.

Compiler receives normalized in-memory bytes and trusted metadata. It never receives a filesystem path to fetch, a Git URL, MCP server URL, credential or lock lifecycle command.

Imported Owner contracts are immutable/read-only in the Studio UI. Updating an import means selecting and resolving a new snapshot, not editing the Owner's bytes as Skill source.

## 3. Browser / Go-WASM wire

Go []byte is base64 in JSON. Studio may retain Uint8Array internally.

~~~json
{
  "apiVersion": "connectionagent.io/compiler-build-request/v1alpha4",
  "languageTarget": "connectionagent.io/v1alpha4",
  "files": [
    { "path": "skills/example/SKILL.md", "bytes": "IyBFeGFtcGxlCg==" }
  ],
  "resolvedDependencies": [],
  "targets": ["example"]
}
~~~

The example omits trustedBuildContext only because it contains no external dependencies or callable use. A real Skill that declares external dependencies or Capability Grants must receive matching trusted context; absence is a fail-closed build error.

Core operations:
- skillcCheck(requestJSON) → diagnosticsJSON;
- skillcBuild(requestJSON) → buildResultJSON;
- skillcInspectDependencies(requestJSON) → inspectionJSON.

Worker request/response keeps a correlation ID and command envelope, but command names and payload types must match v1alpha4 schemas. A Worker/Go/WASM host failure is transport failure, not a Compiler diagnostic.

## 4. Diagnostics and editor integration

The same (Code, Phase) identity must appear in Native, WASM and Studio. Compiler static phases are C_STATIC and C_WARNING; R_RUNTIME belongs to Runtime. Studio may decorate diagnostics but must not rewrite their code, phase, path or source position.

Completion, Definition, References, Hover and Outline must be derived from the same parsed/linkable identities: Node/stage/route, Predicate and ordered parameters, exact ToolCatalog callable, Capability Grant and membership, Owner Schema $id, Script path and generated type, Evidence Guarantee and Producer.

A successful completion item or syntax color is not proof that a callable is linked or authorized.

## 5. Artifact and export

Compiler returns virtual output files. Studio renders them read-only in the Package/Artifact view and performs folder or ZIP export. It does not rewrite author source with generated data.

The output contains author source, generated Script Output Schemas and per-Skill artifact manifests. Dependency bytes are not copied. Dependency digests and lock state remain in Studio configuration; BuildResult returns only the used dependency alias closure.

## 6. Security and trust boundary

The trusted build context is populated from Studio's trusted dependency/deployment configuration, not user-editable Skill files. Compiler verifies supplied bytes, digests, identities and consistency; it does not contact remote registries or establish a trust root.

The browser build is not a security boundary. Host/Runtime must independently verify deployment authorization, current Scope, dependency identity and actual Invocation bindings before dispatch.

Do not send credentials, Provider tokens, MCP secrets or Runtime admission tokens to the Compiler Worker.
