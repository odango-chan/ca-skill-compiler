> **HISTORICAL / NOT CURRENT INPUT (v2.1.0).** This design/work-package document is retained for original source and acceptance provenance only. The active Compiler accepts **CA-SL v1alpha5**; earlier versions are not compatibility modes. Original status statements and instructions below describe their historical snapshot. See [documentation](../README.md), [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), [Development contracts](../../contracts/development/v1alpha1/README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Procedure Linker v1alpha4

**Status:** FROZEN Compiler implementation contract subordinate to the frozen CA-SL specification.

Normative authority: [CA-SL v1alpha4, immutable commit](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md).

## 1. Linker model

The linker builds one typed, scope-aware graph from the Skill Contract, Procedure, PredicateSet and trusted dependency exports. It MUST NOT implement a second language, infer missing declarations, or recursively search arbitrary YAML for keys that happen to resemble a call.

The author graph has one Procedure root with start/nodes; one Node per named business interaction; up to four ordered stages per Node (precondition, operation, observation, postcondition); stage-level outcome routes; a single Contract Goal and optional single global Precondition reference.

The stages execute in this order when declared. Precondition/Postcondition are ordered Predicate groups whose members have no individual routes, Node IDs or Action Budget entries; Operation is a Tool or local Script; Observation is an Owner-published READ query. Repeated Predicate IDs are legal when each call has its own explicit arguments.

**Node cardinality is fixed:** a Node has at most one Operation stage and at most one Observation stage. The Operation stage selects exactly one of tool or script; it is not a list of calls. The Observation stage selects exactly one Owner-published Observation Tool. A Node may contain both stages only in the frozen stage order. If a business interaction requires multiple sequential Operations, represent them as separate Nodes unless the real Owner publishes one actual composite callable. A local Script that prepares a later external request and that external Tool call are separate Nodes. Do not use a multi-call Node abstraction to compress independent effects or hide failure / Evidence boundaries.

## 2. Exact source grammar and rejected legacy syntax

Compiler MUST accept only the frozen v1alpha4 shape:
- Contract fields defined by §§10–11;
- Procedure root start and nodes;
- singular Node stage keys;
- nonempty ordered predicates list inside each Precondition/Postcondition;
- operation.tool or operation.script, exactly one;
- observation.tool for Observation;
- named arguments for Tool/Script and positional arguments for PredicateCall;
- stage-level routes.

Reject duplicate YAML keys, duplicate stages, unknown fields, invalid identifiers, invalid qualified references and noncanonical scalar literals.

Do not accept or translate:
- legacy v0.2 call / with / implicit verify constructs (diagnose the exact recognized legacy construct as UNSUPPORTED_V02_SYNTAX);
- Wait / Await control structures (UNSUPPORTED_WAIT);
- operation.target, observation.target, operation.reconcile and operation.resolve (UNKNOWN_FIELD; do not rewrite target as tool or expose Host reconciliation as an author stage);
- postconditions / preconditions as alternate field spellings (UNKNOWN_FIELD);
- Node-level goal;
- per-Predicate route maps;
- implicit previous-node outputs, implicit global Context or dynamic path expressions.

## 3. Reference identities

### 3.1 Dependency Alias and exports

A qualified external callable or Guarantee reference uses Alias::ExportId. The Alias must be declared in the selected Skill Contract's dependencies and resolve to exactly one matching DependencySnapshot. The snapshot's trusted Stable Package ID must exactly equal the Contract value.

The exact exported identity is case-sensitive. The compiler must not resolve by file path, file name, Schema $id prefix, Owner display name, adapter name, ToolCatalog metadata name, approximate/case-folded ID or first-match file ordering.

ToolCatalog callable IDs share one unique namespace across all ToolCatalog files and both callable kinds in the same snapshot. A duplicate ID is an error even if the Owner or kind differs.

### 3.2 Schema identity

- Skill-owned Schema: exact package-relative path under that Skill's schemas/**, as permitted by the source grammar.
- External Schema: Alias::absolute-URI, resolved to one exact, trusted $id.
- External references resolve only within the selected, verified dependency closure.
- No remote fetch, JSON Pointer fragment, file-name inference, cyclic reference or duplicate $id.
- Every participating Schema root explicitly declares Draft 2020-12 and conforms to G.4.

### 3.3 Callable identity and Capability

- operation.tool must resolve to kind: operation.
- observation.tool must resolve to kind: observation and Effect READ.
- The Skill Contract's Capability Grants are matched to trusted CapabilityContracts.
- Compiler checks exact Owner, ID, kind and Effect for all matching members. Effect conflict in an ungranted member is still an error.
- At least one matching Capability must be granted for the callable. Multiple consistent Capability memberships are legal; Scope is the intersection of all matching granted policies.
- A Capability ID is never accepted as a Procedure Tool reference.

CapabilityContracts are separate deployment input, not files inferred from the ordinary dependency snapshot.

## 4. Procedure graph and routes

Compiler MUST validate:
- start resolves to one declared Node;
- Node IDs satisfy G.1 and do not collide with reserved names/terminal statuses;
- each Node declares at least one valid stage;
- stage order is fixed and stages are unique;
- every route key is valid for that stage and every route target is an existing Node or allowed terminal;
- no cycle exists;
- every nonterminal route target resolves to a declared Node and is structurally valid;
- Nodes not reachable from start produce the normative UNREACHABLE_NODE warning (not a build-blocking error); their syntax, references, Owner links and static contracts are still checked;
- required Outcome routes are present according to §§12 and 22;
- when a later stage is declared in the same Node, that stage's normal Outcome automatically advances to the next declared stage and MUST NOT be repeated in routes; non-normal Outcomes route explicitly. A last-stage normal Outcome MUST have an explicit route;
- Precondition/Postcondition members are evaluated in list order and aggregate to one stage Outcome: all PASS → PASS; any FAIL → FAIL; otherwise any UNKNOWN → UNKNOWN. UNKNOWN does not short-circuit later members because a later FAIL takes precedence; after the first FAIL, remaining members may be skipped and are not recorded as PASS;
- terminal statuses satisfy §12.8;
- only final Postcondition aggregate PASS may route directly to SUCCEEDED.

The compiler must use normative error precedence and (Code, Phase) in §§24 and H; it must not invent graph diagnostics.

### 4.1 Global Precondition dominance

If Contract declares precondition: <node>.precondition, that stage is a global entry gate. It must be reached and aggregate PASS before every business Mutation and before any route that requests SUCCEEDED. A bypass path is a static error. The global gate does not create a new Procedure entrypoint or a second goal.

### 4.2 Final Postcondition

The Contract has one goal. It is not copied into Procedure Node or Postcondition. A Postcondition can be local or final:
- local aggregate PASS may route to another Node and proves only that local check;
- final aggregate PASS must route directly to SUCCEEDED;
- a local PASS cannot be carried forward as final Goal proof;
- all Predicate arguments and Evidence bindings in the final aggregate are checked individually.

## 5. Arguments and SourcePath

Operation and Observation arguments are named mappings. Predicate arguments are ordered positional lists. Every provided argument must match the target's declared parameter/input schema; all required keys/positions must be supplied and unknown keys are rejected.

CA-SL v1alpha4 SourcePath is static and deliberately small:

~~~text
context.<identifier>[.<identifier>...]
<node>.operation.output[.<identifier>...]
<node>.observation.output[.<identifier>...]
~~~

No array index, wildcard, function call, interpolation, dynamic selector, path concatenation or implicit context is allowed. Argument values are only a valid static SourcePath or a permitted scalar literal. Follow §13.2 for the exact YAML scalar subset: strings must be quoted; bare null/boolean/finite decimal values use the frozen forms; inline maps and arrays are not argument expressions.

A Producer output is available only after the corresponding Operation/Observation has produced a valid SUCCEEDED result and only on paths where that Producer dominates the consumer. Precondition and Postcondition produce Outcomes, not bindable output objects. A Predicate PASS does not narrow an optional field's Schema. A field that may be absent cannot be used as a required direct SourcePath.

For every argument:
1. resolve the root and Producer identity;
2. validate path segments against the exact Schema;
3. verify Producer dominance on every reachable path;
4. reject optional/missing path segments where a guaranteed value is required;
5. prove Source Schema assignable to the target parameter/property Schema;
6. validate Subject/Session/Lifecycle/Correlation/resource identity provenance where the path participates in Scope.

Procedure Script call-site inputs are derived from that invocation's named arguments; there is no author-maintained Input Schema.

## 6. Schema assignability

Use the exact CA-SL G.4 Schema subset and F.5 assignment rules. Compiler must prove Source ⊆ Target conservatively.

Reject unsupported Schema keywords, dynamic/remote/cyclic refs, open objects where a static field is required, invalid required sets, unsupported unions and any compatibility that would require coercion or guessing. Default is annotation only and never materializes a missing value.

## 7. Evidence linking

Each Predicate's Evidence requirement must link to:
- one exact Guarantee descriptor;
- the actual ToolCatalog Producer identity and outcome;
- the Producer's declared guarantees.evidence.provides[].path;
- the precise value path and Schema;
- Scope and Correlation mappings;
- freshness/selection rules;
- sourceIdentity / factIdentity and trusted deployment relation when independence or cross-Owner comparison is used.

The descriptor's record, query, arguments and execution roots are distinct. Do not substitute Operation output for an Observation record, or a Tool result for Host Evidence. Two different Evidence refs do not prove independent sources.

If trusted IndependentDomains / common-fact bindings are required and unavailable, reject the link using the frozen Evidence diagnostic. Do not defer an invalid static trust relation to a runtime UNKNOWN.

## 8. Mutation, Reconcile and retry

The Skill author writes a normal operation.tool and, only where the frozen retry rules permit it, retryOf on a distinct later Mutation Node. A retry must follow the original Mutation's proven NOT_COMMITTED edge, preserve the original business identity / Scope / Correlation / Lifecycle binding, target a compatible callable and consume the one permitted retry for that logical Mutation family. A retry Node cannot itself be retried. There is no author-level Reconcile operation or third stage.

The Tool Owner publishes Mutation receipt/commit/failure/resource/reconcile contracts. Compiler validates:
- receipt Schema roots and the receipt. / output. channel anchors;
- commit proof and original Invocation binding;
- failure classes, Owner Outcome membership, no-side-effect and retryable constraints;
- Reconcile callable identity, same Owner, READ Effect, independent Capability/Scope and exact proof paths; its inputSchema is a closed object containing only required nonempty originalInvocationId, materialized by Host from the persisted original Invocation;
- original Mutation proof paths resolve against the original Mutation receiptSchema; Reconcile proofPath resolves against the Reconcile READ's own receiptSchema;
- optional recoverable output Schema/path pairing, exact optional subobject Schema identity and assignability to the original Mutation output Schema;
- same-snapshot dedicated Reconcile READ anti-bypass checks across all published Mutation contracts, plus a separately versioned trusted association for any cross-snapshot Reconcile link;
- resource responsibility output paths and schemas.

A Reconcile READ is an operation export with Effect READ where required by the frozen contract; it is not an author-visible Observation substitute. The Reconcile response/receipt is never exposed as a normal Producer. Only a fully verified recovered business object may be merged into the original Mutation's SUCCEEDED output, with provenance retained.

The compiler must reject unsupported cross-snapshot Reconcile associations when the separately versioned trusted association contract is unavailable. It must not infer association from names, README text or a coincidentally matching READ callable.

## 9. Static vs runtime responsibility

Compiler checks only what can be proven from author source and verified contract bytes. It does not run a Tool or Observation, decide a real-world Predicate, verify a live UE registration or session, execute the Reconcile READ, prove current Evidence freshness, execute Python, establish runtime source independence from values not present in static trust metadata, or decide whether a live 5GC procedure succeeded.

Runtime/Host owns actual invocation, receipts, commit settlement, deadlines, cancellation, Evidence records, freshness, Scope enforcement and terminal resource responsibility.
