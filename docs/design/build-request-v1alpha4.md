# BuildRequest v1alpha4

**Status:** proposed Compiler API transport contract for review; not yet an implemented or qualified API. Machine-readable schema: [build-request.schema.json](../../contracts/v1alpha4/build-request.schema.json).

## 1. Two independent versions

Every request MUST carry both:
- apiVersion: connectionagent.io/compiler-build-request/v1alpha4
- languageTarget: connectionagent.io/v1alpha4

The first versions the Compiler transport/API. The second selects the frozen CA-SL language. Neither can be omitted, inferred from source, or substituted for the other.

The request does not add apiVersion to author-maintained Contract, Procedure or PredicateSet files.

## 2. Logical shape

~~~typescript
type BytesFile = { path: string; bytes: Uint8Array };
type ResolvedDependency = {
  dependencyId: string;
  digest: string;      // sha256:<64 lowercase hex>, recomputed by Compiler
  files: BytesFile[];  // snapshot-relative paths
};
type TrustedDependencyBinding = {
  dependencyId: string;
  digest: string;
  stablePackageId: string;
  authorizedOwners: string[];
  exportOwners?: Array<{
    exportKind: "operation" | "observation" | "schema" | "evidenceGuarantee";
    exportId: string;
    owner: string;
  }>;
};
type TrustedBuildContext = {
  dependencyBindings?: TrustedDependencyBinding[];
  capabilityContracts?: {
    deploymentId: string; publisherId: string; digest: string; files: BytesFile[];
  };
  deploymentAuthorizations?: Array<{
    deploymentId: string; publisherId: string; authorizationRef: string;
  }>;
  evidenceTrustBindings?: {
    deploymentId: string; publisherId: string; snapshotDigest: string; files: BytesFile[];
  };
};
type BuildRequest = {
  apiVersion: "connectionagent.io/compiler-build-request/v1alpha4";
  languageTarget: "connectionagent.io/v1alpha4";
  files: BytesFile[];                  // Skill source only: skills/**
  resolvedDependencies: ResolvedDependency[];
  targets?: string[];                  // selected Skill package names
  trustedBuildContext?: TrustedBuildContext;
};
~~~

JSON/WASM serializes every byte array as base64.

## 3. Author source files

files contains only author-maintained Skill source. Each path is a normalized POSIX path rooted at skills/<skill-name>/.

It may contain SKILL.md, assurance/contract.yaml, assurance/procedure.yaml, assurance/predicates.yaml, Skill-owned schemas/**, Skill-owned scripts/** and other explicitly permitted Skill reference/source files.

It MUST NOT contain dependency files under a synthetic lib/** namespace, generated Output Schemas or prior Build artifacts, dependency lock files, repository URLs, Git refs, branch names, download instructions, cache metadata or Host/Tool Owner schemas copied into the Skill package.

There is no physical or logical global lib/** namespace in this request. Imported contracts are separate immutable snapshots.

## 4. Resolved DependencySnapshot

Each resolvedDependencies item represents one immutable dependency snapshot:
- dependencyId is the Workspace Dependency ID and the Contract Dependency Alias. For each referenced alias, values MUST match byte-for-byte and case-sensitively.
- digest is recomputed from the complete snapshot file set using the [snapshot digest contract](dependency-snapshot-digest-v1alpha4.md).
- files[].path is relative to that snapshot, not a project path.
- files[].bytes are original bytes. Compiler MUST NOT normalize YAML, JSON, line endings or Unicode before hashing.
- one request cannot contain duplicate dependencyId values.

Compiler validates every supplied snapshot's paths and digest before parsing or linking. For a declared and referenced alias, exactly one matching snapshot is required: missing snapshot is DEPENDENCY_SNAPSHOT_MISSING; duplicate alias snapshot is DEPENDENCY_DUPLICATE; absent trusted binding is DEPENDENCY_IDENTITY_UNVERIFIED; Stable Package ID mismatch is DEPENDENCY_ID_MISMATCH; duplicate callable ID is EXPORT_ID_AMBIGUOUS. Missing/ambiguous multi-Owner export attribution is DEPENDENCY_EXPORT_OWNER_UNVERIFIED; a known but unauthorized or conflicting Owner is DEPENDENCY_OWNER_MISMATCH. An extra snapshot MUST NOT enter the link scope merely because it was supplied. Compiler indexes only snapshots selected by an exact Contract Alias and trusted binding.

The same snapshot may contain multiple Owners. For a single-Owner snapshot, the trusted binding may assign its sole authorized Owner to exports without an explicit Owner field. For a multi-Owner snapshot, every used export identity MUST have one explicit trusted (exportKind, exportId) → owner mapping. Compiler MUST NOT infer ownership from path, Schema ID prefix, file order or display name.

exportKind is restricted to operation, observation, schema and evidenceGuarantee. Host Context Schemas are schema; a Reconcile READ is operation. Do not invent hostContext or reconcile export kinds.

## 5. TrustedBuildContext

This section gives the BuildRequest v1alpha4 physical transport to the logical trusted properties required by CA-SL §§34.0–34.0.2 and F.4.

### 5.1 Dependency bindings

For each dependency alias used by the selected Skill, the trusted caller supplies exact dependencyId, the same measured digest that Compiler recomputes, exact stablePackageId equal to Contract.dependencies[alias], the authorized Owner identity set, and for multi-Owner snapshots explicit export-level ownership for every used export.

These are trusted caller metadata, not author-controlled source declarations. A Skill's own dependencies string is a requested package identity, not proof of its own publisher or Owner.

### 5.2 CapabilityContracts deployment input

CapabilityContracts is not an ordinary dependency snapshot and MUST be carried separately.

trustedBuildContext.capabilityContracts contains deploymentId, publisherId, the actual file-set digest and all raw files that make up the trusted CapabilityContracts v1alpha4 input.

Compiler recomputes the digest, parses every document, requires exact apiVersion connectionagent.io/capability-contracts/v1alpha4, and validates the capability/callable structure. The matching deploymentAuthorizations entry must authorize that exact publisher for that exact deployment and include a trusted authorizationRef. No Skill field, ordinary dependency snapshot or Capability name can substitute for this deployment input.

The authorization records and TrustedBuildContext are supplied by the authenticated/trusted build entry, not read from the Skill workspace. authorizationRef is an opaque identifier for the caller-verified authorization record; Compiler MUST NOT dereference it, fetch a URL or treat a syntactically valid URI as proof of authorization. Compiler verifies that the trusted caller supplied a consistent deployment/publisher binding; it does not contact an authorization server or create a trust root. Host/Runtime MUST re-check deployment authorization at admission and dispatch.

### 5.3 Evidence trust bindings

If and only if the selected Skill uses cross-domain IndependentDomains or cross-Owner common-fact comparison, the caller MUST supply evidenceTrustBindings:
- deployment identity;
- authorized publisher identity;
- snapshotDigest for the actual relation file set;
- raw files containing the v1alpha4 EvidenceTrustBindings documents.

The embedded EvidenceTrustBindings document shape is owned and versioned by this BuildRequest v1alpha4 contract; it is not a new Evidence Descriptor apiVersion or a standalone Owner API. Compiler recomputes the bundle digest, validates deployment/publisher authorization and links exact domains, independent pairs, common namespace/fact, participating Owners, Scope mappings and comparable-value Schema identities to the referenced Evidence Guarantee descriptors. Missing, repeated, self-pairs, undeclared domains, unauthorized relations, digest mismatch or ambiguous mappings use the existing Evidence link diagnostic.

The BuildRequest schema defines this transport; [evidence-trust-bindings.schema.json](../../contracts/v1alpha4/evidence-trust-bindings.schema.json) defines the relation document. These are trusted deployment artifacts, not Skill-authored declarations.

### 5.4 Trust boundary

The Compiler does not make an untrusted JSON object trustworthy. Host integration MUST construct trustedBuildContext from its authenticated trust store / authorized workspace configuration and MUST NOT let an author edit it as ordinary Skill source. A standalone CLI may read it from an explicitly configured trusted source.

A browser/Studio build result is a static compilation result, not an authorization token. Runtime must independently verify deployment authorization, dependency identity, Owner provenance, Scope and current Invocation before dispatch.

## 6. Digest and resource limits

The digest framing is fixed in [Dependency Snapshot Digest v1](dependency-snapshot-digest-v1alpha4.md). It is independent of dependencyId, file-array order, repository URL and Git revision.

Initial interoperable limits:
- at most 512 total source + dependency + trusted bundle files in one request;
- at most 2 MiB raw bytes per file;
- at most 16 MiB total raw bytes;
- at most 128 dependency snapshots;
- at most 256 selected Skill targets;
- normalized relative UTF-8 POSIX paths, at most 240 UTF-8 bytes;
- bounded YAML depth and alias expansion, fixed by the implementation acceptance report.

The compiler must enforce aggregate limits before expensive parsing.

## 7. Targets and link scope

Omitted or empty targets selects all Skill packages in files. A nonempty targets list selects those exact Skill package names. CA-SL v1alpha4 does not define a Skill-to-Skill dependency resolver. A target name not present in the request is a hard diagnostic.

Only selected Skill packages contribute author Contract aliases and callable references to the link scope. Unreferenced snapshots never become global exports. The caller supplies the complete immutable input closure; Compiler does not resolve or download transitive dependencies.

## 8. Build result boundary

BuildResult returns diagnostics with exact (Code, Phase), sorted dependencyClosure containing aliases actually used, and output Skill source plus generated files.

It does **not** export snapshot digests, Git revisions, lock state, dependency source URLs, cache keys or raw dependency files. Studio owns those project/lock values.

## 9. Rejected legacy inputs

For v1alpha4, reject rather than translate old lib/** files inside files, old ToolCatalog / CapabilityContracts apiVersion, v0.2 call / with / target, author-level reconcile / resolve, implicit dependency manifests, inferred package identity, old Script metadata and duplicate Script Input Schemas.

No compatibility adapter is part of this API.
