# Compiler Output API Contract v1alpha4

**Status:** FROZEN Compiler-owned output contract. **Implementation conformance:** NOT_QUALIFIED until the conformance gates have executable CI evidence.

This document defines the observable outputs of Check, Build and InspectDependencies. It is subordinate to the frozen CA-SL language specification and the Compiler Contract.

## 1. Version authority

The output identifiers are independent exact-match contracts:

| Operation | Output shape / identifier |
|---|---|
| Check | JSON array of Diagnostic objects; no wrapper object or separate apiVersion |
| Build | connectionagent.io/compiler-build-result/v1alpha4 |
| InspectDependencies | connectionagent.io/compiler-dependency-inspection/v1alpha4 |

The exact machine schemas are:

- [BuildResult](../../contracts/v1alpha4/build-result.schema.json)
- [DependencyInspection](../../contracts/v1alpha4/dependency-inspection.schema.json)

BuildRequest apiVersion and languageTarget remain separate input authorities.

## 2. Diagnostic object

Every Diagnostic contains:

- severity: error or warning;
- code: exact frozen diagnostic Code;
- phase: C_STATIC or C_WARNING;
- path: normalized source path, or the empty string for a request-level diagnostic with no source file;
- message: stable, nonempty human-readable text;
- line and column: optional 1-based source position when known.

The pair (code, phase) is the diagnostic identity. Compiler outputs MUST NOT include R_RUNTIME diagnostics. Severity is determined by phase: C_STATIC is error; C_WARNING is warning.

Diagnostic selection and precedence MUST follow the pinned CA-SL §§24 / H.0 rules. Sorting is applied only after the normative set and primary diagnostic decisions are made; sorting MUST NOT change precedence or suppress a required diagnostic.

## 3. Deterministic ordering

All output ordering is deterministic and independent of input file-array order, YAML map iteration order, Go map iteration order or host platform.

Use unsigned lexicographic UTF-8 byte order for paths, aliases and identifiers. Diagnostics are sorted by this tuple:

1. normalized path bytes (empty path first);
2. line number (missing treated as 0);
3. column number (missing treated as 0);
4. phase rank: C_STATIC before C_WARNING;
5. Code in ASCII byte order;
6. message in UTF-8 byte order.

Severity is derived from Phase and is not an independent ordering key.

BuildResult dependencyClosure is sorted by dependency alias bytes and contains each alias once. BuildResult outputFiles is sorted by normalized path bytes. DependencyInspection callables are sorted by (dependencyId, owner, kind, id, sourcePath); schemas are sorted by (dependencyId, owner, id, sourcePath), each tuple component compared by UTF-8 bytes.

## 4. Check

Check(BuildRequest) runs the same complete static semantic pipeline as Build and returns the Diagnostic array in the order defined by §3.

- Check does not emit an Artifact or dependency closure.
- Check does not execute Python, call a Tool, perform an Observation or contact an Owner.
- A successful Check is only static validation of the supplied source and trusted contract bytes.
- The Check response is not a runtime authorization token.

## 5. BuildResult

BuildResult has exactly these top-level fields:

- apiVersion: connectionagent.io/compiler-build-result/v1alpha4;
- success: boolean;
- diagnostics: Diagnostic[];
- dependencyClosure: sorted unique dependency aliases actually referenced by the selected Skill package set;
- outputFiles: sorted virtual files {path, bytes}, with bytes base64-encoded on the JSON/WASM wire.

success is true if and only if diagnostics contains no C_STATIC error. C_WARNING diagnostics do not block a build.

If any C_STATIC error exists:

- success MUST be false;
- dependencyClosure MUST be empty;
- outputFiles MUST be empty;
- no partial Artifact or generated file may be returned.

On success, outputFiles contains the complete selected Skill Artifact tree defined by the Artifact Contract. Imported dependency bytes, dependency digests, Git revisions, lock state, credentials and Runtime state are never returned.

## 6. DependencyInspection

InspectDependencies(BuildRequest) is a read-only inventory operation for Studio. It validates the supplied immutable dependency snapshots and their trusted dependency / Owner bindings, then returns callable and Schema metadata for exports whose identity and Owner can be established from those trusted inputs.

- It inspects the supplied dependency snapshots, not a Skill's reachable build closure.
- It does not parse or compile Skill Procedures, resolve author references, evaluate Capability Grants, authorize invocation or prove runtime availability.
- It does not fetch remote content, mutate a dependency, persist a lock or cache a trust decision.
- It MUST NOT expose a callable or Schema whose Owner attribution cannot be established. Emit the applicable frozen dependency diagnostic and omit that unverified export.
- A malformed or untrusted snapshot contributes no exports. Other independently verified snapshots may still be listed alongside diagnostics; Studio MUST display the diagnostics and MUST NOT present a partial inventory as a complete trusted dependency set.
- The callables and schemas arrays contain only metadata described by the machine schema. They do not contain dependency bytes, credentials, capability grants or Scope authorization decisions.

If the request envelope itself is invalid, return diagnostics and empty callables / schemas arrays.

## 7. Failure and trust boundaries

- Build and Check fail closed on missing or untrusted dependency / Owner / deployment inputs.
- InspectDependencies is an inventory projection, not a substitute for Build's exact alias, Owner, Capability, Schema, Evidence or SourcePath linking.
- Runtime / Host revalidates deployment authorization, Scope, current Invocation identity, Evidence freshness and actual callable availability before dispatch.
- A successful Build proves static contract compatibility only. It does not qualify a Tool Owner, prove that a 5GC procedure ran, or authorize live execution.

## 8. Versioning

This output contract is frozen for Compiler API v1alpha4. Any incompatible output shape, diagnostic field, ordering rule or failure-atomicity change requires a new Compiler API version. It MUST NOT be introduced as a silent implementation detail.
