# Owner Contract Linking v1alpha4

**Status:** FROZEN Compiler-side linking contract for consuming the CA-SL v1alpha4 Owner and deployment contract profiles. Real Owner publications remain NOT_QUALIFIED where explicitly stated below.

Normative authority: CA-SL §§9, 13.3, 17, 34, F.0–F.4, G.2–G.3.1 at the [pinned specification commit](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).

## 1. Two different publishers and trust domains

Do not collapse ToolCatalog and CapabilityContracts into one file or one authority.

- **Tool Owner / DependencySnapshot:** publishes callable definitions, Owner identity, kind, Effect, Owner Outcomes, Input / QueryContext / Output / Receipt Schemas, Evidence guarantees, Mutation proof and resource responsibility.
- **Deployment / Capability Owner:** publishes the allowed Capability groups, exact callable membership, independently expected Effect and Scope policy references for a particular deployment.
- **Trusted build entry:** binds the imported snapshot bytes to Stable Package ID and authorized Owner identities, and binds CapabilityContracts / Evidence trust relations to deployment and publisher authorization.
- **Compiler:** checks the exact supplied bytes and bindings; it does not fetch, discover, authorize or execute a Tool.
- **Host/Runtime:** revalidates current deployment authorization, Scope and Invocation provenance at admission and dispatch.

A valid ToolCatalog does not imply that a Skill may call it. A Capability grant does not prove that the Tool exists or that its Effect is correct.

## 2. ToolCatalog v1alpha4

The exact imported format is defined by [tool-catalog.schema.json](../../contracts/v1alpha4/tool-catalog.schema.json).

Required root properties:
- apiVersion: connectionagent.io/toolcatalog/v1alpha4;
- kind: ToolCatalog;
- metadata.owner;
- spec.contracts[].

Each callable has an exact id, kind, Effect, ownerOutcomes and outputSchema.$ref.

### Operation

An Operation has kind operation and inputSchema.$ref. It may be READ or MUTATION. A MUTATION requires receiptSchema.$ref and mutation proof metadata. The compiler validates:
- commitProof schema/path/bind;
- optional failureClassPath and failureClasses;
- failure outcome membership and noSideEffect/retryable constraints;
- optional mutation.reconcile callable, originalInvocationId binding, proof Schema/path and paired recoverable output Schema/path;
- resourceOutputs and responsibility Schema;
- Evidence guarantee Producer declarations.

Owner Outcomes are only those the Owner actually publishes. SUCCEEDED is required. FAILED and NOT_COMMITTED are allowed only under their frozen rules. COMMITTED, DENIED, UNKNOWN and OUTCOME_UNKNOWN are not Owner Outcomes; they belong to Runtime/Host.

### Observation

An Observation has kind observation, Effect READ, queryContextSchema.$ref and outputSchema.$ref. Its Owner Outcomes are SUCCEEDED and FAILED only. It is not an Operation renamed as a query and cannot publish Mutation or NOT_COMMITTED behavior.

### Exact identities and schemas

- Callable IDs share one unique namespace across every ToolCatalog file and both callable kinds in a single DependencySnapshot.
- Schema references resolve to complete, exact, unique $id values in the trusted snapshot closure.
- A Schema file path is not a public Schema identity.
- ToolCatalog adapter is Owner-internal binding metadata, not a Skill reference or authorization identity.
- Unknown fields and unsupported Owner contract shapes are rejected.

A Reconcile READ is represented as an operation callable with Effect READ where the frozen contract requires it. It is not a generic Observation and it is not a third author stage.

## 3. CapabilityContracts v1alpha4

The exact deployment document format is defined by [capability-contracts.schema.json](../../contracts/v1alpha4/capability-contracts.schema.json).

Required root properties:
- apiVersion: connectionagent.io/capability-contracts/v1alpha4;
- kind: CapabilityContracts;
- spec.capabilities[].

Each Capability has id, owner and callables[]. Each member has owner, id, kind and effect. A granted member must also resolve a trusted scopePolicyRef. A reject-all policy must be explicit; absence of Scope is never interpreted as allow-all.

Compiler must:
1. find all membership records matching the exact Owner + callable ID + kind;
2. reject Effect conflicts even in ungranted members;
3. reject duplicate membership within one Capability;
4. permit consistent membership across different Capabilities;
5. require at least one matching member in a Skill-granted Capability;
6. resolve every matching granted Scope policy and use their intersection; unresolved policy is CAPABILITY_SCOPE_UNRESOLVED and a statically empty intersection is CAPABILITY_SCOPE_EMPTY_INTERSECTION;
7. reject missing, ambiguous or empty Scope intersections.

A Skill Grant is a requested authorization, not proof that deployment policy has granted it. CapabilityContracts is a separate trusted BuildRequest input and must not be inferred from ordinary dependency snapshots or Skill source.

## 4. Evidence Guarantee descriptors

Evidence Guarantee descriptors are independent exports in trusted dependency material. Their logical properties are fixed by CA-SL F.3 / §17:
- authority and fact;
- producer identities and exact output paths;
- Scope and Correlation;
- factValue Schema;
- selection mode and orderBy where latest is used;
- freshness timestamp and maxAgeMs;
- sourceIdentity and, where required, factIdentity.

The producer must match the actual ToolCatalog callable, kind, Owner Outcome and guarantees.evidence.provides entry. The descriptor roots record, query, arguments and execution are not interchangeable.

Compiler must not invent an Evidence Descriptor from ToolCatalog output Schema or infer guarantees from a field name. Missing descriptors or missing Producer links fail closed.

## 4.1 Evidence Descriptor physical-format boundary

CA-SL F.0/F.3 freezes the logical Evidence Guarantee properties but does not invent a universal Evidence Descriptor apiVersion or a v1alpha4 physical file schema. The actual Evidence Owner must publish an explicitly versioned, trusted descriptor format inside its dependency package. Compiler must pin and validate that exact external format before parsing its descriptor exports.

The BuildRequest-owned EvidenceTrustBindings document is a different artifact: it carries deployment authorization for IndependentDomains and cross-Owner common-fact relations. It is not the Evidence Guarantee descriptor and cannot replace it.

Do not create a generic descriptor schema by guessing the shape of authority, fact, Scope, Correlation or factValue. Until a real Owner publishes a consumable descriptor format, Evidence-dependent linking remains fail-closed and cannot be marked qualified.

## 4.2 Exact Reconcile READ contract

For every Mutation that declares mutation.reconcile:
- The target must be a unique ToolCatalog callable with kind operation, Effect READ, the same trusted Owner as the original Mutation, ownerOutcomes containing SUCCEEDED and optionally FAILED, and its own receiptSchema.$ref.
- Its inputSchema must resolve to a root-closed object whose only required and allowed property is originalInvocationId, a nonempty string. Host materializes this value only from the persisted original Mutation Invocation; an author cannot supply it.
- mutation.reconcile.proofPath resolves against the Reconcile READ callable's own receiptSchema root, not the original Mutation receipt. proofSchema must exactly describe the value at that path.
- The original Mutation's commitProof.path, failureClassPath and failureClasses[].proofPath resolve against the original Mutation callable's own receiptSchema root.
- If recoverableOutputSchema / recoverableOutputPath are declared, both are required. recoverableOutputPath begins with output. and resolves against the Reconcile READ outputSchema root to an optional subobject; that subobject's exact Schema identity must equal recoverableOutputSchema. Compiler proves that Schema assignable to the original Mutation's normal outputSchema and that the original consumer's Evidence requirements remain satisfied.
- The complete Reconcile READ output and receipt are never author-bindable. Only a fully verified recovered subobject may be merged into the original Mutation's successful output while retaining the Reconcile Producer and Invocation lineage.

Compiler scans every Mutation contract in the selected trusted snapshot closure, not only those directly referenced by the selected Skill, to identify dedicated Reconcile READ callable identities. A dedicated Reconcile READ cannot also be called as an ordinary operation.tool; reject that bypass with RECONCILE_CONTRACT_INVALID. A Capability Grant for the READ callable does not make it an ordinary query.

Cross-snapshot / cross-publication Reconcile association is accepted only when a separately versioned trusted Owner/deployment publication proves the exact Owner, original Mutation callable, dedicated Reconcile READ callable, original Invocation binding and Scope relation. Compiler must not infer this relation from names, matching originalInvocationId fields or a separate READ Grant. If the required trusted association format is unavailable, fail closed with the frozen Reconcile / dependency-link diagnostic. No new identityInputs or Reconcile association field is invented in ToolCatalog by this document.

## 5. Identity-input provenance is not a new ToolCatalog field

CA-SL §13.3/F.1 defines the logical Owner annotation needed to identify Subject, Session, Lifecycle, Correlation and resource input paths. The frozen specification explicitly does **not** define a physical identityInputs field.

Therefore:
- do not add identityInputs to the ToolCatalog schema in this PR;
- do not infer identity paths from Schema names, property names or values;
- consume a separately versioned trusted Owner publication when available;
- if a callable requires this provenance and the publication cannot be consumed, use the frozen CALLABLE_ARGUMENT_PROVENANCE_INVALID diagnostic and fail closed; if Runtime cannot verify the original Invocation binding, it uses CALLABLE_ARGUMENT_PROVENANCE_UNVERIFIED;
- Runtime's inability to verify the original Invocation binding remains a Runtime concern.

Cross-snapshot Reconcile association is also not inferred from README text, callable names or a separate READ Grant. If the required versioned trusted association contract is unavailable, reject that interop case rather than guessing.

## 6. Trusted deployment relation bundles

The BuildRequest v1alpha4 contract carries the physical transport for trusted deployment inputs. These bytes and metadata are not author-editable.

### CapabilityContracts bundle

The wrapper includes deploymentId, publisherId, digest and raw files. Compiler recomputes the digest, checks the exact CapabilityContracts apiVersion and verifies that the trusted caller's deployment authorization binds the same deployment and publisher.

The trusted authorization record is a caller-supplied trusted input. The compiler does not authenticate a self-asserted field or contact an authorization server.

### EvidenceTrustBindings bundle

The wrapper includes deploymentId, publisherId, snapshotDigest and raw files. The embedded documents use the BuildRequest v1alpha4-owned EvidenceTrustBindings shape, validated by [evidence-trust-bindings.schema.json](../../contracts/v1alpha4/evidence-trust-bindings.schema.json). This is not a new Evidence Descriptor apiVersion or an Owner-published descriptor format.

It carries:
- IndependentDomains: deployment, publisher, unique absolute domain URI set and unordered, nonduplicate, non-reflexive independent pairs;
- common-fact relations: namespace/fact, participating Owner set, per-Owner Scope dimension paths, value paths and comparable Schema identity.

Compiler recomputes the raw file digest and checks publisher/deployment authorization, exact Guarantee sourceIdentity domain membership, pair validity, Owner membership, Scope mapping and comparable Schema identity. Owner self-assertion is not trust. Missing/invalid binding is EVIDENCE_GUARANTEE_LINK_INVALID.

## 7. Product qualification boundary

A schema-valid example is not a real Owner publication. A successful static link is not proof of deployment authorization or live service availability.

The v1.0.0 release acceptance did not have a qualified real v1alpha4 Owner/Deployment publication set, and the MAS baseline was not runnable. A synthetic fixture or legacy ToolCatalog is not a substitute for a trusted v1alpha4 publication. Do not change version strings or fabricate Owner schemas to make a real build pass. Reassess this boundary only against actual versioned Owner/Deployment publications and real qualification evidence.
