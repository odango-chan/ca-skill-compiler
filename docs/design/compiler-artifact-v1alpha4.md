# Compiler Artifact Contract v1alpha4

**Status:** proposed physical artifact contract for the CA-SL v1alpha4 compiler. This contract is independent of language semantics and does not define Runtime Trace format.

Machine-readable schema: [artifact-manifest.schema.json](../../contracts/v1alpha4/artifact-manifest.schema.json).

## 1. Artifact boundary

Compiler returns a deterministic virtual file tree. It does not create a ZIP, write to the user's filesystem or persist a package. Skill Studio decides whether to export the returned tree as a directory or ZIP.

A successful Skill artifact contains author-maintained Skill files copied byte-for-byte, generated read-only Script Output Schemas and one generated artifact manifest per selected Skill.

It does not contain imported DependencySnapshot files, the full ToolCatalog, CapabilityContracts or unrelated Owner data, credentials, deployment secrets, lockfiles, source URLs, cache metadata, Runtime Trace, execution journal or Host state.

The Artifact Contract does not define Runtime loading or execution. If Host later needs a normalized executable IR, it requires a separately versioned Runtime Artifact contract and acceptance.

## 2. Output tree

All paths are normalized POSIX paths rooted at skills/<skill-name>/.

For each selected Skill, output every accepted author source file unchanged, generated Script Output Schemas under schemas/generated/**, and generated/artifact-manifest.json.

Procedure Script and Predicate Script Input Schemas are not generated. Procedure Script input is derived per call site; PredicateInput is a compiler-only virtual type.

Default Script Output Schema path:

~~~text
skills/<skill>/schemas/generated/<script-relative-path-without-scripts-prefix-and-py-suffix>.output.schema.json
~~~

The path must be deterministic and collision-free. A collision with an author source file or another generated output is a hard build error.

## 3. Artifact manifest

The generated skills/<skill>/generated/artifact-manifest.json has:
- apiVersion: connectionagent.io/compiler-artifact/v1alpha4;
- languageTarget: connectionagent.io/v1alpha4;
- exact Skill package name and Contract Goal;
- generator name, semantic version and immutable build commit (when available);
- sorted sourceFiles list;
- sorted generatedFiles list, excluding the manifest itself;
- sorted dependencyClosure containing only dependency aliases used by that Skill.

The manifest MUST NOT export dependency digests, Git revisions, source locations, credentials or deployment authorization tokens. Studio retains dependency lock / digest state. The manifest records logical aliases only; the original Contract retains alias → Stable Package ID.

For a multi-Skill Build, each Skill receives its own manifest. BuildResult's top-level dependencyClosure is the sorted union of aliases actually used by selected Skills.

## 4. Schema serialization

Generated JSON Schemas use UTF-8 without BOM, the exact Draft 2020-12 $schema URI, stable exact $id values, RFC 8785 JSON Canonicalization Scheme (JCS), no insignificant whitespace and no trailing newline. They use only CA-SL G.4-supported keywords and Artifact-Contract metadata.

The same semantic Schema must produce byte-identical output in Native Go and WASM. Do not serialize via map iteration order or a platform-default formatter. The artifact manifest uses the same UTF-8/JCS rules. Author source bytes are not canonicalized or rewritten.

## 5. BuildResult behavior

BuildResult includes exact apiVersion, success, sorted diagnostics with exact (Code, Phase), sorted dependencyClosure aliases and sorted outputFiles containing base64-encoded bytes.

On any ERROR: success=false, outputFiles=[] and dependencyClosure=[].

On success, output includes source files, generated Schemas and artifact manifests only. Dependency bytes are never copied into outputFiles. BuildResult never exports dependency snapshot digest values.

The BuildResult schema is [build-result.schema.json](../../contracts/v1alpha4/build-result.schema.json).

## 6. Deterministic ordering

Sort paths and aliases by unsigned lexicographic UTF-8 byte order. Preserve source file bytes exactly. Generated JSON uses JCS. Duplicate output paths are rejected before emission. A failed build emits no partial files, even if some Skills individually passed.

## 7. Export and ownership

Studio treats generated files and artifact manifests as read-only. Rebuild replaces the generated view atomically; it must not overwrite author source with generated data. ZIP/folder export is a Studio responsibility and must preserve exact paths and bytes.

This artifact contract does not define Runtime Trace, Host journal, package installation, release signing or deployment authorization.
