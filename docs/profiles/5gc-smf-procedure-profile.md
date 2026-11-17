# 5GC SMF Procedure Profile — 3GPP → Tool → Skill

**Status:** FROZEN for the CA-SL v1alpha4 structural and authoring boundaries defined here.

**Language authority:** [CA-SL v1alpha4](https://github.com/odango-chan/ca-skills/blob/7f84393ee8c395e4d8c39a74cb5000a1f896588d/docs/skill-language-spec.v1alpha4.zh-CN.md), pinned to `7f84393ee8c395e4d8c39a74cb5000a1f896588d`.

**3GPP reference baseline:** Release 19, version 19.5.0 for the specifications listed in §8. This is a fixed review baseline, not a floating "latest" reference.

**Qualification:** Real Owner callable, Schema, Evidence and Deployment publications remain **NOT_QUALIFIED**. This profile does not claim that a sample is linked, runnable or accepted by a live 5GC.

This profile defines how a 5GC Skill maps the SMF's standard business procedure onto CA-SL v1alpha4. It does not extend CA-SL syntax, replace 3GPP specifications or authorize an unpublished Tool.

## 1. Governing principle

The correct relationship is **3GPP → Tool → Skill**:

1. **3GPP** defines the normative network procedure, participating NFs, service operations, protocol messages and Request/Response semantics.
2. **Tool** is the concrete callable contract published by the responsible Owner. It exposes a real interface or a clearly identified CA adapter/extension. Its ToolCatalog identity, Owner, kind, Effect, Input/QueryContext Schema, Output Schema and guarantees are authoritative.
3. **Skill** declares stable dependencies and required Capability Grants, then composes exact Tool callables into its Procedure graph to implement the SMF-owned business flow.

A Skill does not define or approximate an external interface. It consumes the Owner's published Tool contract. A Skill-authored Schema must not copy or redefine an externally owned request, response or context structure.

A Tool Owner may use an adapter internally, but the adapter must not obscure whether the callable represents a standard NF operation, a CA adapter around one or more standard interactions, or a CA-only extension.

## 2. SMF perspective and Procedure Node granularity

A 5GC Skill whose business actor is the SMF is written from the **SMF's point of view**. Its Procedure describes the complete SMF-owned business procedure and may span multiple 3GPP protocol exchanges.

A Node represents one logical SMF business interaction or one tightly related group of protocol exchanges. It commonly maps to one 3GPP Request/Response interaction, but it is **not** mechanically one Node per message, numbered substep, helper function or local variable.

The CA-SL v1alpha4 structure and stage order remain authoritative:

- A Node has at most one `operation` stage. That stage selects exactly one of `tool` or `script`.
- A Node has at most one `observation` stage. It selects one exact Owner-published READ callable.
- The stage order is `precondition → operation → observation → postcondition`; omitted stages are skipped.
- An Operation is the Node's primary business action. An Observation is normally a supplementary READ in the same Node, after its Operation and before its Postcondition. It does not create a second business action.
- A final Postcondition-only Node is valid when it represents the final Goal proof; it is not an excuse to add implementation-helper Nodes.
- If a logical 3GPP step requires multiple sequential independent Operations, represent those Operations as separate Nodes. A traceability group may link the Nodes to one broader 3GPP step, but does not permit multiple Operation calls in one Node.
- A single composite callable may represent multiple exchanges only when the real Owner publishes that composite interface and its complete request, response, Effect and proof contract.

### 2.1 Observation placement

An Observation is a separate CA-SL stage and remains distinct from an Operation callable, even when both are executed by the common Host Tool Dispatcher.

- Prefer co-locating a supplementary Observation with the business Operation whose decision or Postcondition needs that fresh fact.
- Reuse the Operation Result and its Evidence when they already provide authoritative, correctly scoped, correlated and fresh facts. Do not query the same fact again without a business reason.
- Do not create an Observation-only Node merely to fetch a fact needed by an adjacent Operation. When the fact must be read before a mutation, attach the Observation to the earliest suitable read-only business Operation Node, after that read-only Operation and before its Postcondition.
- A standalone Observation-only Node is an exception. It requires an independently meaningful business decision boundary and an explicit rationale; it must not disguise Host admission, lifecycle checks or callback polling.
- CA-SL stage order does not permit an Observation before an Operation in the same Node. If a required read must precede a later Operation and cannot be attached to an earlier suitable read-only Operation Node, it belongs in a preceding Node with a justified business boundary.

An Observation is an authoritative state query, not a way to poll an asynchronous Operation. The Observation Owner must publish the exact query identity, READ Effect, QueryContext and Output Schemas, Scope, Correlation, temporal/freshness semantics and Evidence guarantees. For the SMF-focused Skills in this profile, query only the exact SMF-owned context/facts required by the next business decision and not already present in the Operation Result or published Host Subject Context. Non-SMF context required to complete an Operation remains inside the responsible Tool/Owner adapter; do not make the Skill query every NF's internal context.

### 2.2 Local Script placement

A local Script is not automatically a separate 3GPP interaction.

- Use `operation.script` only for deterministic, pure, Skill-owned computation.
- A separate Script Node is justified only when it represents an independently meaningful, typed SMF-local business decision/result that is consumed independently by the Procedure.
- Do not create a Script Node solely for a temporary variable, local copy, one-line mapping, request reshaping or revalidation of data already checked by an earlier interaction.
- Do not move native SMF processing into a Skill Script merely to make it visible in the graph. In the initial PDU Session profile, Step 8 UPF/session-address selection belongs inside the SMF N4 Owner operation that performs N4 Session Establishment; it is not a separate Skill Script Node.
- A Script Operation is not an external Tool, has no ToolCatalog export and requires no Capability Grant.

### 2.3 Subject identity and Host-private correlation

Use the actual 3GPP identity fields required by the admitted business procedure and the published Context Schema. For the current PDU Session profile, the admitted SUPI is the UE subject identity; do not invent a Host-generated `ueRef` or make it a Skill parameter. A future Skill-visible context field is valid only when its exact Schema and trusted Owner publication exist.

Host admission identity, ingress correlation, lifecycle/generation references, Invocation/Run identity, request ledgers and transport metadata remain Host/Runtime-private. They are not substitutes for SUPI, PDU Session ID or other 3GPP business fields and must not be copied into Skill-authored arguments. Tool/Owner adapters bind private execution metadata under their own contracts.

## 3. Standard-to-Tool mapping

Map each external 3GPP interaction to a real Owner-published callable before referencing it from a Skill.

| 3GPP concept | Tool contract |
|---|---|
| SBI service operation / Request-Response | Exact `kind=operation` callable with the Owner's Input and Output Schemas, Effect and declared Outcomes |
| Authoritative state query | Exact `kind=observation` callable with Effect READ, QueryContext and normalized Output Schemas, Scope and Evidence guarantees |
| Mutation with uncertain completion | Normal Operation plus the Owner-published receipt, commit/failure and dedicated Reconcile READ contract required by CA-SL |
| N4 / PFCP interaction | Owner-published Tool for the actual PFCP/N4 operation; do not invent an SBI operation for N4 |
| SMF-local deterministic computation | Skill-owned Procedure Script only when it is an independently meaningful local business result |

The Tool's Schemas must describe the actual Owner request and response contract. A Skill must not substitute its own guessed Schema, field mapping or output projection.

The same callable may be reused by multiple Skills and Nodes. Do not create a new Tool merely to give one Skill a convenient business-step name. Multiple callable identities for one standard service are acceptable only when the Owner publishes distinct real interfaces or variants.

A local CLI or client implementation repository is not automatically a CA-SL Tool Owner. In particular, `ca-client-tools` may provide the fixed local CLI backend for an adapter, but it does not by itself publish the ToolCatalog, CapabilityContracts, Owner identity or Deployment grants. Those remain separate trusted publications.

## 4. Capability mapping

A Capability is a deployment authorization grant over exact callable identities and Effects. It is **not**:

- a 3GPP Service Operation;
- a Tool callable ID;
- the Skill's business workflow; or
- a substitute for a ToolCatalog export.

The Skill's `operation.tool` and `observation.tool` fields reference the exact callable as `Alias::ExportId`. Capability Grants are declared separately in the Contract and must resolve through trusted CapabilityContracts.

Capability names and descriptions should make the owning NF/domain and permitted service or procedure visible. Prefer least-privilege grants for UDM SDM read/subscribe, PCF SM Policy Association create, AMF N1N2 transfer, SMF context update and N4 establishment/modification over one opaque all-purpose 5GC grant.

A Capability ID must come from the trusted CapabilityContracts publication. Do not invent an ID to make a sample appear linkable. A clearly marked `urn:example:` placeholder may be used in a non-qualified design example, but it is not an authorization and must fail closed at link time.

## 5. Classify standard interactions and CA extensions

Every Node/Operation mapping must be classified in traceability documentation as one of:

| Classification | Meaning | Authoring rule |
|---|---|---|
| `3GPP_NATIVE` | Directly represents a named 3GPP NF Service Operation or protocol procedure | Use the canonical 3GPP name and exact clause/step; the callable still comes from the real Owner ToolCatalog |
| `3GPP_LOCAL` | SMF-local pure computation/decision that is independently meaningful in the cited procedure and has no external NF call | Use a deterministic Skill Script; do not claim a network side effect or external Capability |
| `CA_ADAPTER` | A CA Owner implements or orchestrates one or more 3GPP interactions behind a governed callable | Keep the CA Owner identity; state the exact 3GPP steps it supports and which completion proof it owns |
| `CA_EXTENSION` | A CA-only business function with no direct 3GPP operation counterpart | Use a clearly CA-prefixed identity; explicitly state that it has no direct 3GPP operation |

A CA adapter must not claim that a standard NF Service Operation exists when it actually performs multi-step internal orchestration. A CA extension must not be disguised as a standard 3GPP interaction.

MAS is not a 3GPP-defined NF, service or procedure. A MAS-oriented Skill is a CA-specific policy/adapter profile and its Nodes must be classified as `CA_ADAPTER` or `CA_EXTENSION` unless a Node maps to an exact published 3GPP NF Service Operation. Candidate MAS callable names from an OES plan are not Owner exports and are not qualified until the responsible Owner publishes their exact contracts.

Host ingress admission, identity and duplicate checking, business-instance association, Invocation lifecycle, correlation, persistence, authorization, Action Budget accounting and internal mutation reconciliation are not Skill Nodes. They remain Host/Runtime/Owner responsibilities. Do not create `admit-*`, `hold-*`, `wait-*` or artificial 3GPP Nodes for these internals.

A Skill's request-profile eligibility is a business Precondition on the first suitable business Node; it is not Host admission and must not be named or described as such.

## 6. Asynchronous inbound requests and the v1alpha4 boundary

This profile applies to **CA-SL v1alpha4 only**.

CA-SL v1alpha4 §21 does not define author-level `wait`, `await` or polling Nodes. The Skill must not add a Wait stage, Wait-only Node, `wait.output` SourcePath or generic polling Tool.

For a v1alpha4 asynchronous interaction:

1. The business Node contains the declared Owner Operation for the interaction.
2. The Owner Tool contract describes the actual request, pending/completion behavior, normalized final result, Evidence, correlation and failure/timeout semantics.
3. A later inbound protocol request is authenticated and admitted by the Host/Integration boundary, then correlated to the exact pending Owner Operation in the existing Invocation.
4. The Owner returns the contractually validated completion result and Evidence; Runtime resumes that same Invocation. The later request does not create a new Skill or Invocation.
5. The Skill does not treat request acceptance, an HTTP acknowledgement or an intermediate protocol response as proof of final business completion.

The Node-level Event Wait design frozen by [ca-skills PR #60](https://github.com/odango-chan/ca-skills/pull/60) is for a language revision **after** v1alpha4. Its proposed Wait is Node-entry control, not a fifth business stage or a Wait Node. The normative amendment remains a separate, unreleased draft in [ca-skills PR #61](https://github.com/odango-chan/ca-skills/pull/61). Neither may be used as v1alpha4 syntax or silently mixed into a v1alpha4 package.

Do not confuse inbound Event Wait with a bounded Evidence Observation. An Observation is a governed READ query; it is not an inbound event listener and must not be used to poll for an AMF request.

## 7. PDU Session Establishment reference map

The reference procedure is UE-requested initial PDU Session Establishment from the SMF perspective, based on TS 23.502 §4.3.2.2.1. The following map uses the fixed Release 19 V19.5.0 baseline listed in §8.

| 3GPP step | SMF-side interaction | Tool / Owner boundary | v1alpha4 Skill representation |
|---|---|---|---|
| 2 | AMF selects the SMF | AMF-owned selection; not an SMF callable | Outside the Skill Procedure |
| 3 | AMF invokes the `Create SM Context` service operation with `Nsmf_PDUSession_CreateSMContext Request` | The Nsmf_PDUSession API is specified by TS 29.502 §5.2.2.2 and OpenAPI Annex A.2; Host/Integration ingress authenticates, validates and admits the request | Ingress trigger, not a Skill Node or Skill Tool |
| 4 | SMF retrieves and subscribes to Session Management Subscription data when required: `Nudm_SDM_Get`, `Nudm_SDM_Subscribe` | UDM Owner publishes two exact service-operation callables and their Schemas; do not collapse independent calls into one invented callable | Separate business Nodes when both operations are required; an exact SMF-context Observation may be supplementary in the read-only Step 4 Node |
| 5 | SMF returns `Nsmf_PDUSession_CreateSMContext Response` | Host/Integration owns the paired early service-response boundary and its request/response continuity | Not a Skill Node; the response is early and is not final Goal proof |
| 6 | Optional DN-AAA secondary authentication/authorization | DN-AAA/SMF Owner contracts for the admitted profile | Omit only when the Skill's supported profile explicitly excludes it; otherwise model the real branch |
| 7a–7b | PCF selection and SM Policy Association Establishment; `Npcf_SMPolicyControl_Create` | PCF Owner publishes the exact Create callable and Request/Response Schemas; SMF-local PCF selection may remain inside the Owner Operation | One business Node for the Create interaction; do not invent a separate Tool for an internal selection helper |
| 8 | SMF selects SSC mode/UPF and allocates session address as applicable | Native SMF processing; for this profile it is owned inside the SMF N4 establishment boundary | No separate Skill Node or Capability for helper selection/address preparation |
| 9 | Conditional SMF-initiated SM Policy Association Modification | PCF Owner publishes a separate exact callable if the admitted profile requires this interaction | Separate Node when in scope; otherwise explicitly exclude the conditional branch |
| 10a–10b | N4 Session Establishment Request/Response for an initial request | SMF N4 Owner publishes the actual PFCP/N4 Operation, Effect, Input/Output Schemas and completion proof | One Node for N4 Session Establishment; the Owner operation also owns native Step 8 selection for this profile |
| 11 | SMF invokes `Namf_Communication_N1N2MessageTransfer` | AMF Owner publishes the exact N1N2 transfer callable and Schemas | One Node for transfer initiation; its acknowledgement is not RAN adoption |
| 12–14 | AMF, (R)AN and UE perform N2 resource setup and return the access-network result | AMF/(R)AN/UE own these actions; they are not SMF Tool Operations | No Skill Node for another NF's internal action; the result returns to the SMF at Step 15 |
| 15 | AMF invokes the `Update SM Context` service operation with `Nsmf_PDUSession_UpdateSMContext Request` carrying the N2 result | TS 29.502 §5.2.2.3 and OpenAPI Annex A.2 define the service contract; the SMF/Integration Owner validates and normalizes the inbound request and correlates it to the exact pending Operation | The existing pending `update-sm-context` Owner Operation in the same Invocation completes; no new Skill invocation or v1alpha4 Wait Node |
| 16a–16b | SMF performs N4 Session Modification and receives the UPF response | SMF N4 Owner publishes the actual Modification callable and completion proof | Separate Node from initial N4 Session Establishment |
| 16c | Conditional `Nudm_UECM_Registration` when the specified request and registration conditions apply | UDM Owner publishes the exact UECM Registration callable and Schemas | Separate Node when required; it must complete before the paired Step 17 response |
| 17 | SMF returns `Nsmf_PDUSession_UpdateSMContext Response` for the exact Step 15 request | The paired response is the response side of the TS 29.502 Update SM Context service operation; Host/Integration owns wire mapping and request/response continuity | Host/Integration service exit, not an invented Tool or final Goal proof; it follows all applicable Step 16 work |
| 18 | Conditional `Nsmf_PDUSession_SMContextStatusNotify (Release)` when establishment fails after Step 5 | SMF/Host Owner owns the published failure notification and associated resource release behavior | Failure-path responsibility must be explicitly assigned; do not silently infer it from a Skill terminal label |
| 19 | IPv6 Router Advertisement where the PDU Session type/profile requires it | SMF/UPF and the applicable transport path | Out of scope for an IPv4-only Skill; add only for an admitted IPv6/IPv4v6 profile |
| 20 | Conditional SM Policy Association Modification for Bridge/Router information when the trigger is armed | PCF Owner publishes the exact callable if the feature is in scope | Explicitly exclude when not supported; do not create a placeholder success |
| 21 | Conditional UDM subscription unsubscription on the specified failure path when the SMF no longer handles a matching PDU Session | UDM Owner publishes the exact unsubscribe callable; Host/Owner retains cleanup responsibility | Failure-path behavior must be assigned to a real Owner/Host contract; do not invent a Skill cleanup Tool |

The table is a traceability map, not a claim that every optional or conditional step is required in every Skill. Clause §4.3.2.2.1 covers non-roaming and roaming with local breakout; each Skill must state which admitted deployment profile it supports. Home-routed roaming under §4.3.2.2.2 is outside this map unless separately reviewed. Each Skill must also state its supported optional branches and exclusions. A required 3GPP behavior may be delegated to an Owner/Host adapter only when the responsible Owner publishes the exact contract and accepts that responsibility.

### 7.1 Proof boundaries that must remain distinct

- Step 5 CreateSMContext Response is an early protocol response, not final PDU Session establishment.
- PCF association creation is not N4 Session Establishment.
- Step 8 UPF selection/address allocation is not an independent SBI operation.
- N4 Session Establishment is not N4 Session Modification.
- Step 11 N1/N2 transfer initiation or acknowledgement is not RAN resource acceptance.
- Step 14 is the (R)AN result; Step 15 carries that result to the SMF.
- Step 16a–16b N4 forwarding modification is not the paired Step 17 response.
- Step 16c UECM registration, when applicable, is part of the Step 16 work and precedes Step 17.
- Step 17 protocol response is not itself the Skill Goal proof. The Skill Contract's final Postcondition and Host/Runtime resource-responsibility gate remain authoritative.
- A confirmed business failure, an unresolved result and an unknown Mutation outcome are distinct. Failure handling must preserve the exact original Invocation, resource responsibility and late-result handling.

### 7.2 Failure and conditional branches

The normal success path must not erase the standard's failure branches:

- If establishment fails between Step 5 and Step 11, Step 11 carries the PDU Session Establishment Reject and Steps 12–17 are skipped. Do not wait for a Step 15 request that will not arrive.
- If Step 15 reports failed user-plane resource setup, or the required User Plane Enforcement Policy cannot be fulfilled, the normal Step 16 modification path is skipped. The SMF performs the required N4 release and Step 17 returns the paired rejection response. Do not record N4 forwarding as modified or the PDU Session as established.
- If Step 15 succeeds and the applicable Step 16 actions complete, the paired Step 17 response follows those actions, including conditional Step 16c registration where applicable.
- Step 18 is a conditional failure notification and resource-release path. Step 21 is conditional UDM unsubscription. Both must have explicit Owner/Host responsibility, exact request/response contracts and failure evidence; neither is implied by a generic Skill terminal status.
- Step 19 and Step 20 are conditional features. A Skill that excludes IPv6 Router Advertisement or Bridge/Router information must state that scope; it must not emit placeholder successful results for those steps.

## 8. Fixed 3GPP reference set

The reference map in §7 is pinned to the following Release 19 V19.5.0 documents:

| Specification | Version | Scope in this profile | Official source |
|---|---|---|---|
| TS 23.501 | V19.5.0, Release 19 | 5GS architecture and SMF/UPF selection context | [ETSI PDF](https://www.etsi.org/deliver/etsi_ts/123500_123599/123501/19.05.00_60/ts_123501v190500p.pdf) |
| TS 23.502 | V19.5.0, Release 19 | PDU Session Establishment procedure and N4 interaction steps | [ETSI PDF](https://www.etsi.org/deliver/etsi_ts/123500_123599/123502/19.05.00_60/ts_123502v190500p.pdf) |
| TS 29.502 | V19.5.0, Release 19 | SMF Session Management SBI service; `TS29502_Nsmf_PDUSession.yaml` OpenAPI definition in Annex A.2 | [ETSI PDF / Annex A.2](https://www.etsi.org/deliver/etsi_ts/129500_129599/129502/19.05.00_60/ts_129502v190500p.pdf) |
| TS 29.503 | V19.5.0, Release 19 | UDM services, including SDM and UECM | [ETSI PDF](https://www.etsi.org/deliver/etsi_ts/129500_129599/129503/19.05.00_60/ts_129503v190500p.pdf) |
| TS 29.512 | V19.5.0, Release 19 | SM Policy Control service | [ETSI PDF](https://www.etsi.org/deliver/etsi_ts/129500_129599/129512/19.05.00_60/ts_129512v190500p.pdf) |
| TS 29.518 | V19.5.0, Release 19 | AMF services, including N1N2 Message Transfer | [ETSI PDF](https://www.etsi.org/deliver/etsi_ts/129500_129599/129518/19.05.00_60/ts_129518v190500p.pdf) |
| TS 29.244 | V19.5.0, Release 19 | PFCP/N4 protocol between SMF and UPF | [ETSI PDF](https://www.etsi.org/deliver/etsi_ts/129200_129299/129244/19.05.00_60/ts_129244v190500p.pdf) |

V19.5.0 is a deliberately fixed review snapshot, not an assertion that it is the newest published Release 19 version. Do not silently float to a later version. Any baseline update must review the affected procedure steps, API/OpenAPI contracts and Tool mappings together.

A version pin for a 3GPP document does not qualify an Owner implementation. For each callable, the separate Owner traceability record must still pin the actual ToolCatalog publication, exact callable ID, Owner identity, kind, Effect, Schema IDs, guarantees, implementation repository and immutable source revision.

## 9. Required traceability and qualification record

Every business Node in a 5GC Procedure must have a traceability entry in the Skill's `SKILL.md` and a concise source comment beside the Node in `procedure.yaml`.

For each Node, record:

1. Node ID and human-readable 3GPP interaction name.
2. Exact 3GPP TS number, Release/version, clause, figure and step(s), where applicable.
3. Protocol direction and participating NFs.
4. Canonical 3GPP service operation or protocol procedure/message.
5. SMF responsibility before the interaction and what its result actually proves.
6. Classification: `3GPP_NATIVE`, `3GPP_LOCAL`, `CA_ADAPTER` or `CA_EXTENSION`.
7. Exact ToolCatalog alias/export ID, Owner, callable kind and Effect, where a Tool is used.
8. Required Capability Grant and its trusted CapabilityContracts source.
9. Exact Input/QueryContext/Output Schema identities and resulting Facts/Evidence/Postcondition Predicate IDs.
10. Async completion, failure, response and resource-responsibility ownership where applicable.

For each Tool export, the Owner-side mapping record must additionally contain the API resource and HTTP method or PFCP message type, Request/Response data types, receipt/commit/failure/Reconcile contracts, and immutable implementation source revision.

These traceability records belong in profile documentation, Skill comments and `SKILL.md`; they are not new CA-SL fields. Do not add ad-hoc `3gpp`, `standard`, `step`, `target` or `mapping` fields to `contract.yaml` or `procedure.yaml`.

Before any row is claimed as qualified, the actual trusted Owner and Deployment publications must exist and resolve. A syntactically valid example, `urn:example:` identity, legacy ToolCatalog, README or source-code symbol is not a published contract. Missing Owner exports remain missing; do not invent them to make Compiler pass.

The Compiler repository's `examples/ca-agent-demo-project/` is a pinned historical migration input, not a current v1alpha4 positive Build sample. Its presence does not qualify this profile or the upstream 5GC examples. Use the repository's current v1alpha4 Compiler smoke sample only for Compiler smoke testing; it is not a 5GC qualification fixture.

## 10. Release and version boundary

This document describes only the frozen CA-SL v1alpha4 authoring model. In particular:

- `operation.tool` and `observation.tool` reference exact `Alias::ExportId` callable identities.
- Capability IDs are separate deployment authorization grants.
- Observation is a READ callable and a distinct stage.
- The final Goal is proved only by the final Postcondition aggregate PASS routed directly to `SUCCEEDED`.
- Author-level Wait/Await/Poll and author-level Reconcile/Resolve are not part of v1alpha4.

Do not import v0.2/v1alpha3 author syntax, legacy `call/with` fields, `waiterRef`, author-level Wait/Poll or future-revision `wait.output` into a v1alpha4 Skill. The Compiler must reject unsupported source rather than translate it.

The frozen Node-level Event Wait design for the next language revision is not a v1alpha4 feature. It does not amend this profile's v1alpha4 rules and must not be used until a new language version, Compiler, Host/Runtime and Integration contracts are independently published and qualified.
