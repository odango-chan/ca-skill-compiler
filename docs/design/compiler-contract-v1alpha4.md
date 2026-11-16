# Compiler Contract v1alpha4

**Status:** implementation contract for the frozen CA-SL v1alpha4 target. **Conformance:** NOT_QUALIFIED until the gates in the conformance plan have actual CI evidence.

This document defines Compiler behavior. It does not amend or duplicate language semantics.

## 1. Normative authority

The only normative language authority is the complete CA-SL specification at [ca-skills commit 7f84393](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).

- Repository: odango-chan/ca-skills
- Commit: 7f84393ee8c395e4d8c39a74cb5000a1f896588d
- File blob: b70064c5737b8ee2b3d1435fe496357f44b42773

Compiler, tests and conformance reports MUST pin this immutable commit. They MUST NOT use floating main, infer a later language revision, or silently translate v0.2 / v1alpha3 source. Companion Authoring Model, Schema Ownership and Script notes are informative only.

## 2. Purpose and ownership

Compiler is a deterministic, side-effect-free transformation:

**author-maintained Skill source + immutable resolved dependency snapshots + trusted build context → diagnostics + generated Skill artifact file tree.**

Compiler owns:
- strict parsing and AST construction for the selected CA-SL language target;
- exact dependency, ToolCatalog, CapabilityContracts, Schema and Evidence linking;
- Procedure graph, route, dominance and SourcePath validation;
- conservative static Schema assignability and identity-provenance checks;
- static Script signature / TypedDict analysis and generated Output Schema creation;
- deterministic diagnostics and artifact generation;
- one semantic Core shared by Native Go and Go/WASM.

Compiler does not own:
- Git, HTTP, MCP, upload or built-in dependency acquisition;
- Workspace Configuration, project settings, dependency selection, lock persistence, refresh, cache or snapshot lifecycle;
- package registry, version solver, manifest or resolver lifecycle;
- credentials, provider connection, deployment authorization decisions or Runtime Admission;
- Tool invocation, Python execution, Qualification, Release, live network validation or Runtime Trace serialization.

Compiler consumes bytes and trusted metadata supplied by its caller. It never opens a path from BuildRequest and never fetches a dependency.

## 3. Independent versioned interfaces

| Boundary | Exact identifier |
|---|---|
| BuildRequest | connectionagent.io/compiler-build-request/v1alpha4 |
| CA-SL language target | connectionagent.io/v1alpha4 |
| BuildResult | connectionagent.io/compiler-build-result/v1alpha4 |
| DependencyInspection | connectionagent.io/compiler-dependency-inspection/v1alpha4 |
| ToolCatalog | connectionagent.io/toolcatalog/v1alpha4 |
| CapabilityContracts | connectionagent.io/capability-contracts/v1alpha4 |
| Evidence trust relations | BuildRequest v1alpha4-owned embedded format |
| Compiler Artifact | connectionagent.io/compiler-artifact/v1alpha4 |
| JSON Schema dialect | Draft 2020-12 with the exact CA-SL G.4 whitelist |

All version strings are exact-match inputs. Missing, unknown, v1alpha3 or legacy v0 identifiers fail closed. There is no version sniffing, fallback, negotiation or automatic conversion. BuildRequest API version, languageTarget, ToolCatalog apiVersion, CapabilityContracts apiVersion and Artifact apiVersion are separate version authorities.

## 4. Public Core operations

- Check(BuildRequest) → Diagnostic[]
- Build(BuildRequest) → BuildResult
- InspectDependencies(BuildRequest) → DependencyInspection

Check and Build MUST run the same semantic pipeline. Build MUST NOT skip checks performed by Check. InspectDependencies is a read-only projection for Studio; it does not grant access or qualify a dependency.

For Go/WASM transport, byte arrays are base64 JSON strings. Native Go callers pass the same raw bytes.

## 5. Required compilation pipeline

1. **Request envelope:** validate BuildRequest API version, exact language target, limits, duplicate paths, path normalization and base64 decoding.
2. **Source inventory:** identify Skill package roots and required source files; reject generated artifacts fed back as author source.
3. **Safe parse:** parse YAML 1.2 with duplicate-key detection, bounded depth/alias expansion, no custom tags, no cyclic aliases and no ambiguous scalar coercion.
4. **Author AST:** parse Contract, Procedure and PredicateSet using the exact v1alpha4 grammar. Unknown fields are errors.
5. **Snapshot integrity:** recompute every supplied dependency / trusted bundle digest over raw bytes before parsing or linking.
6. **Trust binding:** bind each declared Dependency Alias to exactly one snapshot and trusted Stable Package ID; validate authorized Owner and export-owner attribution. Validate separate deployment CapabilityContracts and required Evidence trust relations.
7. **Contract index:** index exact ToolCatalog callable IDs, Schema $id values and Evidence Guarantee exports. No lookup by filename, prefix, display name or fuzzy match.
8. **Capability linking:** resolve every callable against trusted CapabilityContracts; compare Owner, kind and Effect; validate requested Grants and Scope policy references.
9. **Schema linking:** resolve exact external Schema IDs and complete reference closure. Reject missing, duplicate, cyclic, remote or unsupported references.
10. **Procedure linking:** validate Node stages, route keys/targets, terminal edges, reachability, acyclicity, global Precondition dominance and final Postcondition success paths.
11. **Dataflow/type checking:** resolve SourcePaths per reachable path; prove Producer dominance and Source-to-Target Schema assignability; validate identity and Scope provenance.
12. **Evidence checking:** link each Guarantee to its real Producer and output path; validate Scope, Correlation, Freshness, sourceIdentity and trusted factIdentity / IndependentDomains when used.
13. **Script analysis:** statically parse the approved Python subset, derive Procedure call-site input types and Script Output Schemas, and validate Predicate Script stubs. Never execute Python.
14. **Result:** sort diagnostics deterministically. If there are no errors, emit the complete artifact. Otherwise emit no artifact and no partial dependency closure.

Internal package boundaries may differ, but observable diagnostics and results may not.

## 6. Language obligations

| CA-SL area | Compiler obligation | Normative source |
|---|---|---|
| Contract | Goal, singular global precondition, Request/Context Schema refs, dependency Alias → Stable Package ID and Capability Grants | §§10–11 |
| Procedure | One graph with start/nodes; singular precondition, operation, observation, postcondition stages | §§10.1, 12 |
| PredicateSet | Ordered parameters, Evidence requirements, evaluator and exact Predicate IDs | §§10.1, 14, 17 |
| Operation | tool: Alias::ExportId or Skill-owned script, never a Capability or URL | §§12.3, 32 |
| Observation | Exact ToolCatalog kind=observation, Effect READ, QueryContext and Output Schema | §§9, 12.4 |
| Predicate stages | Nonempty ordered predicates list; positional arguments; one stage-level route map | §§12.2, 12.5 |
| SourcePath | Only declared Context or dominating Operation/Observation outputs | §13 |
| Evidence | Exact Guarantee/Producer, provenance, Scope, Correlation, Freshness and independence | §17, F.3, G.3 |
| Mutation | Owner outcomes, receipts, commit proof, failure classes, Reconcile READ, retryOf and resource responsibility | §§9.2.1, 12.7, 20, G.2 |
| Script | Procedure and Predicate Script ABI; strict TypedDict; derived inputs and generated output schemas | §32, F.6 |
| Schema | Draft 2020-12 restricted to G.4; conservative assignability | G.4, F.5 |
| Diagnostics | Exact (Code, Phase) identity and precedence | §§24, H |

Only a final Postcondition aggregate PASS directly routed to SUCCEEDED is a successful completion path. A local Postcondition PASS is not final Goal proof.

## 7. Schema and CEL semantics

CA-SL v1alpha4 does not mean “accept every Draft 2020-12 keyword.” Compiler must implement exactly G.4.

### 7.1 Schema allowlist

The only validation keywords are:
- $ref, type, properties, required, additionalProperties, items;
- minItems, maxItems, minLength, maxLength;
- minimum, maximum, exclusiveMinimum, exclusiveMaximum;
- enum, const.

The only non-assertion annotations are:
- $schema, $id, title, description, default, examples.

Every participating Schema root MUST explicitly declare the exact Draft 2020-12 dialect URI. The type value is one scalar type or exactly [T, "null"]; arbitrary unions are forbidden. required is a unique subset of properties. additionalProperties is only true or false; a SourcePath-addressable object must be closed. items is one Schema, not a tuple array. A $ref is an independent reference object and may only be accompanied by the permitted annotations.

Reject $defs, definitions, JSON Pointer fragments, allOf, anyOf, oneOf, not, if/then/else, pattern, patternProperties, unevaluatedProperties, contains, prefixItems, format, dynamic references, extension keywords and every unknown keyword. Do not ignore unsupported keywords or use a general validator's broader acceptance as proof of CA-SL compatibility.

For a binding, prove Source ⊆ Target. Integer-to-number is allowed for JSON Schema assignment where F.5 permits it; reverse coercion, string/number coercion, implicit defaults and nullable-to-nonnullable binding are not. Required keys, optional keys, enum/const, ranges, string limits and array constraints must be checked conservatively. Default is annotation only and never materializes a value.

### 7.2 CEL subset and type checking

CEL is not “whatever the selected CEL library accepts.” The exact v1alpha4 expression subset is:
- declared parameters and deterministic scalar constants;
- static field dot access;
- parentheses;
- boolean !, && and ||;
- equality / inequality;
- same-numeric-type ordering comparisons;
- has(x.field);
- size(x) for string / array.

Reject other calls, macros, comprehensions, loops, array indexing, object construction, dynamic member names, reflection, time, randomness and external data. The expression's result type must be statically boolean.

Compiler must statically check every field path and operand type. Optional fields may be read only when the same short-circuit expression proves presence with has. Nullable fields require a non-null guard. A Predicate PASS does not narrow an optional field for later SourcePath use.

CEL int is signed 64-bit; CEL double is finite IEEE-754 binary64. CEL int and double do not mix implicitly. When an integer JSON Schema source is bound to a CEL number parameter, Compiler must prove it lies in the exact safe-integer range [-9007199254740991, 9007199254740991]. Runtime overflow, non-finite values, unverified numeric serialization or evaluation exceptions produce UNKNOWN, never PASS.

The Schema Engine and CEL parser/evaluator implementation choices are not selected by this contract. They must implement this exact language profile and pass the pinned fixtures. PR #8's Schema library spike is technical evidence, not production adoption. Do not add a production Schema dependency or broaden the supported keyword/expression set without a separately reviewed decision.

## 8. Tool and Capability semantics

ToolCatalog is the callable contract published by the real Tool Owner. CapabilityContracts is a separate trusted deployment authorization input. Skill Contract declares requested Capability Grants; Procedure references exact callable IDs.

- Operation and Observation are distinct callable kinds.
- Observation MUST have Effect READ.
- ToolCatalog owns callable kind, Owner, Effect, outcomes and Schema identities.
- CapabilityContracts owns exact membership, independently declared Effect and Scope policy.
- Compiler checks all matching membership records, including ungranted records for Effect conflicts. Multiple consistent Capability memberships are legal; granted Scope boundaries are intersected.
- At least one matching Capability must be granted for the callable to pass the static contract check.
- A Capability is not a Tool and is not a Procedure step.

CA-SL §13.3/F.1 defines logical identity-input requirements but explicitly does not define an identityInputs field. Compiler must consume a separately versioned trusted Owner publication. If absent or un-linkable, fail closed; never infer identity paths from field names.

## 9. Evidence and trusted deployment data

Compiler validates static Evidence linkage only. It does not create Evidence, establish real-world source independence or decide a runtime Predicate result.

If a Skill uses cross-domain independence or cross-Owner common-fact comparison, BuildRequest MUST carry the corresponding trusted Evidence relation bundle, raw bytes, digest, deployment identity and publisher authorization. Compiler verifies the bundle and its relation to the referenced Guarantee descriptors. Missing or invalid binding is an Evidence link error, not a runtime UNKNOWN shortcut.

The authenticated caller / trusted build entry supplies the trusted context and publisher authorization. Compiler verifies exact bytes, digest, identity and binding relationships; it does not contact an authorization server or create a trust root. Runtime/Host MUST revalidate authorization at admission and dispatch. A successful static build is not an authorization token.

## 10. Diagnostics and failure atomicity

Every v1alpha4 diagnostic carries severity, exact code, exact phase, normalized source path, 1-based line/column when available and stable nonempty message.

Compiler emits static phases only: C_STATIC and C_WARNING. R_RUNTIME belongs to Host/Runtime and is not emitted by Compiler.

Use the exact codes and precedence in §§24 and H. Do not create look-alike diagnostics, silently downgrade an error or invent a new code for an implementation gap.

If any ERROR exists: success=false, outputFiles=[], dependencyClosure=[], and no partial artifact is emitted.

## 11. Determinism and parity

For the same normalized BuildRequest bytes and trusted context, Native Go and Go/WASM MUST produce identical success state, diagnostics, source positions, dependency closure, output paths, file order and output bytes. Input file-array order and map iteration order MUST NOT affect results. Studio/LSP must consume the same Core semantics and must not implement a second, weaker parser or type checker.

## 12. 3GPP is a profile, not Compiler semantics

The architecture mapping is **3GPP → Tool → Skill**:

1. 3GPP defines network procedures and service-operation interfaces.
2. The real Owner publishes a ToolCatalog callable implementing the corresponding interface contract.
3. A CA-SL Skill declares Capability Grants and composes those exact Tool callables into Procedure Nodes.

For 5GC, see [the SMF Procedure Profile](../profiles/5gc-smf-procedure-profile.md). Compiler validates Tool identity, Owner, Effect, Schema and Skill graph. It MUST NOT hard-code TS 23.502 procedures, 5GC network functions, SBI names or SMF business logic into the generic compiler core.

## 13. Acceptance boundary

A successful Build proves only that supplied source and trusted contract bytes satisfy the static v1alpha4 compilation contract. It does not prove Owner publication is operational, deployment authorization is safe at runtime, Python executes successfully, Tools are reachable, 5GC E2E is correct, MAS is runnable, or Runtime Trace / Qualification / Release is ready.

The current PR #11 is a source/graph admission slice, not full v1alpha4 conformance.
