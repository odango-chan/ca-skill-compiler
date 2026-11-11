# CreatedSession authoring principles

**Status: AUTHORING MODEL / NOT YET RUNTIME-COMPATIBLE**

For deterministic execution, the author-facing machine source for `establish-pdu-session` is:

- `../assurance/procedure.yaml`

There is one deterministic authoring graph. Toolchain/Runtime compatibility must be derived from it;
authors should not maintain a second execution graph by hand.

The Skill also has an **experimental LLM-driven execution view**. `../SKILL.md` is the
model-facing CreatedSession business guide for that experiment; it is not a second graph
and does not copy deterministic next-edges.

Repository-wide executor/context placement rules live in
[`../../../docs/design/llm-skill-execution.md`](../../../docs/design/llm-skill-execution.md).
The CreatedSession-specific experiment plan is recorded in
[`llm-execution-experiment.md`](llm-execution-experiment.md).

Both execution styles are expected to share the same Harness safety boundary, admitted
capabilities, Predicate truth, Observation authority and final Contract.

## 0. Dual execution views

The current deterministic Runtime remains unchanged by the LLM experiment.

```text
SKILL.md                         assurance/procedure.yaml
   |                                      |
LLM business guidance             deterministic executor
   |                                      |
   +------------------+-------------------+
                      |
                   Harness
                      |
       Capability / Predicate / Observation
                      |
                   Contract
```

This Skill-specific document does not define the generic LLM executor protocol,
concurrency model, Decision Context or MCP grounding rules. Those belong to the
repository-wide LLM execution design.

The CreatedSession-specific rule is only that its `SKILL.md` must be sufficient as
business guidance for a weak model without becoming a prose copy of deterministic
next-edges.

## 1. Preserve the existing Procedure language

The authoring model keeps the existing core structure:

- `procedures`
- `inputs`
- `start`
- `nodes`
- `call`
- `script`
- `verify`
- `procedure`
- `with` / `from`
- `on`
- `reconcile`
- `terminal`

Do not redesign the Skill DSL merely to make one Procedure shorter.

The only structural extensions currently justified by repeated authoring pain are
node-level `preconditions` and `postconditions`.

## 2. A main node represents a business-significant step

A main-flow node should normally correspond to at least one of:

- a real external operation;
- a mutation or other side-effect boundary;
- an independent asynchronous network procedure;
- a recovery/reconciliation boundary;
- a recognizable 3GPP procedure, service operation, or message boundary.

Pure input projection, deterministic local calculation, read-only context preparation,
or a single fact check should not become a standalone business node merely because the
Runtime needs a step.

## 3. Preconditions are the ordered node-entry phase

`preconditions` describe everything that must be completed immediately before the
node's main operation may execute.

They may contain ordered, side-effect-free preparation steps such as:

- deterministic local `script` computation;
- read-only `ca.*` context/evidence acquisition;
- `verify` of current facts;
- a read-only child Procedure that waits for convergence.

A precondition may read or refresh authoritative evidence so that a fact is evaluated
against the current occurrence. It must not create the business fact by mutating the
network.

Preconditions must not contain:

- NF/SBI mutations;
- any operation requiring mutation reconciliation;
- hidden resource creation;
- a side effect that would itself deserve a business/recovery boundary.

The block has an aggregate failure route. This matters because the main operation has
not executed yet.

Before resource responsibility is activated, precondition failure exits directly as
`FAILED` or `UNRESOLVED`. After responsibility is active, precondition failure must
first follow the resource-finalization path.

Node preconditions are business entry conditions, not the only mutation safety gate.
Runtime must atomically recheck the current attempt/lifecycle/authority and register the
real invocation identity immediately before **every** mutation dispatch. Do not copy that
Runtime hard guard into every Skill node merely to make the graph look safe.

## 3.1 Precondition outcome folding

`preconditions.steps` are evaluated in order. The block terminates at the first
non-successful step.

The authoring semantics are:

- deterministic script / read-only call:
  - `SUCCEEDED` -> next step;
  - `FAILED` or `DENIED` -> preconditions `FAILED`;
  - `UNKNOWN` -> preconditions `UNRESOLVED`;
- `verify`:
  - `PASS` -> next step;
  - `FAIL` -> preconditions `FAILED`;
  - `UNKNOWN` -> preconditions `UNRESOLVED`;
- read-only / convergence child Procedure:
  - `SUCCEEDED` -> next step;
  - `FAILED` -> preconditions `FAILED`;
  - `UNRESOLVED` -> preconditions `UNRESOLVED`.

All steps must succeed before the main operation may be dispatched. A mutation is never
partially entered because an earlier precondition completed.

## 4. Postconditions establish facts after the main operation

A `postcondition` verifies facts that must hold after the main operation before
business progression may continue.

The common pattern is:

```text
Preconditions
    ↓
Main operation
    ↓
Postconditions
    ↓
Next business node
```

Operation completion is not enough by itself. For example:

- `Nudm_SDM_Get` returning successfully does not prove that the returned subscription
  evidence is usable for the current occurrence;
- `Npcf_SMPolicyControl_Create` being committed does not by itself prove the required
  policy association is authoritatively owned.

A Confirm/Verify step that only checks the preceding operation therefore belongs in
that operation's `postconditions`, not in a separate business Procedure.

## 4.1 Successful operation outcome enters postconditions

When a node declares `postconditions`, the main operation's successful outcome does not
transition directly to the next business node.

- read / local compute `SUCCEEDED` enters the postcondition phase;
- mutation `COMMITTED` enters the postcondition phase;
- only the declared postcondition route may advance the business flow.

Therefore `COMMITTED` means only that the mutation happened. It does not mean that the
node's required business fact has been established.

If a node has no postconditions, its ordinary successful transition remains explicit in
`on`.

Postconditions use the same block shape as preconditions:

```yaml
postconditions:
  steps:
    - verify: createdsession.example.first_fact
    - verify: createdsession.example.second_fact
  on:
    PASS: next-stage
    FAIL: finalize-failed
    UNKNOWN: finalize-unresolved
```

The example only illustrates the block shape; it does not move RAN setup and N4
forwarding facts into one business node. Each block verifies facts that belong to its
own main operation or convergence procedure.

Postcondition steps are ordered and may contain only `verify` plus optional `with`
bindings. The block stops at the first non-PASS result: FAIL takes the block's FAIL
route, UNKNOWN takes its UNKNOWN route, and the PASS route is taken only after every
step passes. Routing belongs to the block, not to individual steps.

Do not invent a composite Predicate merely to express “all of these facts must hold”.
A composite Predicate is appropriate only when the combined fact has its own business
meaning; ordinary conjunction belongs in the block.

Node-level `postconditions` govern whether the current operation may advance. They do
not replace `SkillContract.spec.postconditions`, which remains the independent final
acceptance contract for the whole Skill. Keep those two scopes separate.


## 4.2 Preparation results: node-local versus shared

A preparation result belongs inside a node when only that node consumes it. If later
business nodes also consume the result, it is **shared procedure data semantically**,
even though the current authoring syntax still names its producer through a path such as
`<node>.preconditions.<step>.output`.

Current shared results are:

| Named preparation result | Meaning | Downstream consumers |
| --- | --- | --- |
| `select-subscription-configuration.output` | Subscription-Constrained Selection | Policy-context preparation, PCF Create and AMF delivery preparation |
| `resolve-pcf-create-inputs.output.pcfCreateInputsRef` | Occurrence-bound PCF create input readiness handle (input facts only; no ownership, creation, callback or sender claim) | UDM Subscribe request evidence, PCF Create and initial N4 convergence |
| `resolve-effective-session-configuration.output.contextRef` | Effective Session Configuration context | Establishment-response preparation, AMF transfer, PDU Session Resource Setup, N4 forwarding convergence and UE configuration wait |

Keep these references explicit for now. Do not move a pure preparation step into the
main business flow just to shorten its reference; do not duplicate the calculation in
each consumer; and do not hide the dependency with YAML anchors or implicit global
state.

The producer path is the current syntax for locating the value, not a claim that the
value is private to the producing node. A typed procedure-scope output/context contract
may be considered when the same need is demonstrated across multiple Skills. That
syntax is not introduced in this revision.

## 4.3 Maintainer-only Provider handle details

The following exact integration detail moved from model-facing `SKILL.md`.
It documents Harness/Provider bindings rather than model-selectable arguments.

Two Provider reads produce different opaque handles and must not be substituted for
each other:

- `ca.session.operation_context.read` returns an operation-context `contextRef`. Use
  it only for its matching purpose (`nas-and-n2-delivery` for N1/N2 delivery and
  downstream waits). `purpose=policy-create` never returns `AVAILABLE`.
- `ca.session.pcf_create_inputs.read` returns `pcfCreateInputsRef`. It is a
  five-second PCF-create input-readiness handle and never represents operation
  context, association ownership, creation, sender reservation or callback
  ownership.

UDM Subscribe obtains DNN/S-NSSAI through the PCF inputs read plus the governed
selection, with an exact lifecycle/invocation binding, a required match between
the Provider-bound `ingressRef` and the dispatch `ingressRef`, and a locally
recomputed selection-digest comparison. Digest equality means both sides
normalised the same DNN/S-NSSAI/PDU-Session-Type/SSC selection; it does not mean
the Provider independently verified all five fields, because its SMF-builder
comparison covers DNN, SST and SD. The initial N4 wait joins on the exact
`currentSessionInstanceRef` session-instance value, not on either handle.

## 5. Predicate, operation, mutation, and Procedure outcomes are different layers

Keep state domains separate:

- Predicate: `PASS / FAIL / UNKNOWN`
- mutation certainty: `COMMITTED / NOT_COMMITTED / OUTCOME_UNKNOWN`
- Procedure terminal: `SUCCEEDED / FAILED / UNRESOLVED / SAFE_STOPPED`

No state from one layer substitutes for another.

In particular:

- operation success is scope-local and does not imply Procedure `SUCCEEDED`;
- mutation `COMMITTED` does not imply a Predicate `PASS`;
- timeout alone does not imply `FAILED`;
- stale, missing, or ambiguous evidence remains unknown;
- mutation uncertainty follows reconcile + safe-stop semantics rather than ordinary
  unknown-evidence handling.

## 6. Async network convergence remains explicit

Do not collapse a genuine asynchronous network procedure into a one-shot `verify`.

Examples in CreatedSession include:

- N4 Session Establishment;
- PDU Session Resource Setup;
- N4 Session Modification / forwarding convergence.

These remain explicit child Procedures that perform governed evidence waiting and then
verify the authoritative postcondition.

An initial `UNKNOWN` observation may simply mean the network has not converged yet.
The governed await decides when waiting has actually terminated.

## 7. UE configuration is a completion fact, not a fake linear stage

UE session-configuration convergence is causally downstream of N1/N2 delivery, but it
should not be presented as if the 3GPP business flow were:

`N4 Session Modification -> UE Session Configuration`.

CreatedSession therefore waits for UE configuration as part of the successful-completion
entry conditions before final resource-responsibility transfer.

## 7.1 Provenance is part of Predicate identity

Evidence must be scoped to the exact occurrence and, where a producing operation exists,
to the exact producer.

Do not evaluate final facts from a latest-per-UE or latest-per-session observation.

CreatedSession currently requires:

- UDM read result -> exact UDM invocation/transaction receipt;
- AMF transfer result -> exact N1N2 transfer operation reference;
- RAN/N4-forwarding/UE evidence acquisition -> exact transfer operation; final Predicate scope -> current lifecycle;
- resource responsibility -> current lifecycle.

The lifecycle identity is attempt-specific. A later admitted recovery decision must
receive a new lifecycle identity; it must not reuse the prior invocation's lifecycle
as a shortcut to old RAN/N4/UE/resource evidence. If a new recovery attempt cannot be
bound to a distinct authoritative lifecycle, lifecycle-scoped evidence is ambiguous and
must not PASS.

`createdsession.ingress.eligible` is checked at admission and rechecked immediately
before the first resource mutation to close the admission-to-mutation TOCTOU window.

## 8. Resource responsibility follows the first mutation boundary

For CreatedSession, resource responsibility is admitted immediately before the first
resource-creating mutation.

Rules:

1. before the first mutation, failures do not finalize resources;
2. after responsibility is active, every **normal** success / FAILED / UNRESOLVED exit
   passes through resource-responsibility finalization;
3. an original mutation with `OUTCOME_UNKNOWN` is the exception to normal finalization:
   reconcile that exact mutation first and do not invent a resource disposition while
   the side effect is uncertain;
4. uncertain mutation outcome never triggers blind retry;
5. after mutation uncertainty, the current invocation always `SAFE_STOPPED`, even if
   reconciliation later determines COMMITTED or NOT_COMMITTED;
6. safe-stop does not discard ownership: the lifecycle keeps late-result/resource
   responsibility durably retained for a later admitted recovery decision;
7. later recovery is a new admitted decision with a new lifecycle identity.

## 9. Namespaces and names have stable meanings

Use:

- `sbi.*` for real NF/SBI operations;
- `ca.*` for ConnectionAgent-owned capabilities, regardless of effect;
- `skill.*` for Skill-local deterministic tools/evaluators;
- `createdsession.*` for CreatedSession business facts / Predicates.

Capability namespace does not determine effect. A `ca.*` capability may be read-only,
local-compute/evidence-wait, or a governed mutation such as resource-responsibility
transfer. Effect metadata and mutation semantics remain authoritative.

Where a 3GPP canonical operation or procedure name exists, preserve its tokenization in
the author-facing node name using kebab-case. For example:

- `Nudm_SDM_Get` -> `nudm-sdm-get`
- `Npcf_SMPolicyControl_Create` -> `npcf-sm-policy-control-create`
- `Namf_Communication_N1N2MessageTransfer` -> `namf-communication-n1n2-message-transfer`

Do not invent a CA alias for an existing 3GPP business operation.

## 10. Local preparation belongs to the consumer's preconditions

The three configuration layers remain distinct:

- Requested Session Parameters;
- Subscription-Constrained Selection;
- Effective Session Configuration.

The calculations remain semantically important, but they do not need standalone
business nodes when they are deterministic/read-only preparation for the next real
operation.

Current examples:

- subscription selection and policy-operation-context preparation are preconditions of
  `Nudm_SDM_Subscribe`;
- effective-session-context resolution and establishment-response construction are
  preconditions of `Namf_Communication_N1N2MessageTransfer`.

No earlier layer is evidence that a later layer is already satisfied.

## 10.1 Profile exclusions are explicit

The first CreatedSession profile does not cover every conditional branch of the 3GPP
procedure.

It is applicable only when the admitted request does **not** require secondary PDU
Session authentication/authorization with a DN-AAA server. A request that requires that
branch is outside this Skill profile until an explicit business Procedure is added.

Likewise, a conditional SM Policy Association Modification is not inferred from the
earlier Create result. If the current policy trigger requires Modification, that branch
must be explicitly represented by a future profile.

Omitting an unsupported conditional branch is never interpreted as that branch having
succeeded.

Profile enforcement is split by when the fact can be known:

- request-shape restrictions such as initial Request Type, IPv4 and SSC mode 1 are
  enforced by the request schema;
- ingress-known profile admission is part of the semantics of
  `createdsession.ingress.eligible == PASS`; `actionAuthorized` means this specific
  profile was admitted, not merely that some generic action is permitted;
- a branch requirement discovered only from later UDM/PCF evidence must make the
  relevant preparation/readiness precondition fail closed. The absence of an authored
  branch never counts as successful execution of that branch.

## 11. CreatedSession main business flow

The author-facing main flow should read approximately as:

1. `Nudm_SDM_Get`
2. `Nudm_SDM_Subscribe`
3. `Npcf_SMPolicyControl_Create`
4. N4 Session Establishment
5. `Namf_Communication_N1N2MessageTransfer`
6. PDU Session Resource Setup
7. N4 Session Modification / forwarding convergence
8. completion convergence + resource-responsibility finalization

Preparation, fact verification, evidence waiting details, reconcile branches, and
terminal mechanics must not obscure that business order.

## 12. Runtime adapts to the authoring model

Do not contort `assurance/procedure.yaml` merely to satisfy the currently pinned
Runtime/schema.

Toolchain/Runtime may lower author-facing constructs into explicit execution nodes,
including:

- precondition steps;
- postcondition verification;
- evidence polling/subscription;
- reconcile branches;
- terminal routing;
- trace identities.

The lowered graph is an execution artifact, not a competing authoring source of truth.
