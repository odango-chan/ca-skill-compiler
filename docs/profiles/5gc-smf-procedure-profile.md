# 5GC SMF Procedure Profile — 3GPP → Tool → Skill

**Status:** architecture and traceability profile. It does not extend CA-SL syntax and does not qualify current 5GC Owner publications.

## 1. Core mapping principle

The correct relationship is **3GPP → Tool → Skill**:

1. **3GPP** defines the normative network procedure, participating NFs, service operations, messages and Request/Response semantics.
2. **Tool** is the concrete, Owner-published callable contract that implements or exposes one of those standard interfaces in the actual system. ToolCatalog records its exact identity, Owner, kind, Effect, input/query Schema, output Schema and guarantees.
3. **Skill** declares required Capability Grants and composes exact Tool callables into its Procedure graph to implement the business flow.

Skill is not the source of the interface definition; it consumes the Tool contract.

## 2. SMF perspective and Node granularity

For a 5GC Skill whose business actor is the SMF, the Procedure is written from the **SMF's point of view**.

A Procedure Node SHOULD correspond to one semantically coherent SMF interaction unit in the relevant TS 23.502 procedure:
- usually one standard Request/Response exchange;
- or a tightly coupled group of Request/Response exchanges that together form one indivisible SMF-side step in the standard procedure.

This is a modeling guideline, not a claim that every Node must contain exactly one Tool call. One Node may invoke more than one Tool where the normative step genuinely requires multiple distinct interfaces or local deterministic preparation. Independent 3GPP steps should normally remain separate Nodes so that routing, failure, Evidence and recovery boundaries remain explicit.

The Node ID and description should make the corresponding 3GPP step recognizable. Avoid generic business labels that hide the standard interaction.

## 3. Standard-to-Tool mapping

A 3GPP interface / service operation is mapped first to a real ToolCatalog callable:

| 3GPP concept | ToolCatalog representation |
|---|---|
| SBI service operation / Request-Response | exact kind=operation callable, with Owner Input and Output Schemas |
| Authoritative state query | exact kind=observation callable, Effect READ, QueryContext and normalized Output Schemas |
| Mutation with uncertain completion | normal Operation plus Owner-published receipt / commit / failure / Reconcile READ contract |
| N4 / PFCP interaction | Owner-published Tool contract for the actual PFCP operation; do not model it as an imaginary SBI call |
| Local SMF deterministic transformation | Skill-owned Procedure Script; it is not a Tool or Capability |

The Tool Owner may wrap a real implementation behind an adapter, but the adapter label does not replace callable identity or standard reference. Tool Schemas must reflect the real Owner's request and response contract, not a Skill-authored approximation.

One standard service operation can have more than one concrete callable only when the Owner publishes distinct, real interfaces or variants. One callable can be reused by many Skills and Nodes. Do not create a new Tool merely to give one Skill a convenient business-step name.

## 4. Skill mapping

After the Tool contract exists:
- Contract declares stable dependency identities and required Capability Grants.
- Procedure Node uses operation.tool: Alias::ExportId or observation.tool: Alias::ExportId.
- Operation / Observation arguments are explicit and statically checked against Owner Schemas.
- Predicate arguments use the declared ordered parameter signature.
- Postcondition checks the actual Result and required Evidence; it does not redefine the standard API.
- Mutation uncertainty is handled by the Owner/Host contract, not a Skill-authored Reconcile call or arbitrary Wait stage.

A Capability is a deployment authorization grouping of exact callable identities and Effects. It is not a 3GPP Service Operation and it is not the Skill's business workflow.

### 4.1 Local processing and Host-internal coordination

A 3GPP procedure step labelled **Local processing** is not automatically a missing Tool or Capability. If it is deterministic selection, validation, transformation or construction of the next external Request, represent it as a Skill-owned Procedure Script (`operation.script`), not as an external Tool. A Node has one Operation stage and that stage selects exactly one of `tool` or `script`; therefore a local Script Operation and a subsequent external Tool Operation are separate explicit Nodes when both are needed. Label the Script Node as local processing so readers can distinguish it from a 3GPP Request/Response exchange. Bind its output explicitly to the following Tool Node's arguments. It needs no ToolCatalog entry and no Capability Grant. Do not invent a Capability just to make local computation appear as an external call.

Host-internal coordination primitives such as hold/reservation, lease bookkeeping, Invocation state, authorization checks, Action Budget accounting and the dedicated Mutation reconciliation mechanism are not 3GPP service operations and are not author-visible Procedure stages or Nodes. They remain Host/Runtime implementation details governed by Owner and Host contracts. In particular, do not add a visible `hold` Node, an author `wait` stage, or an artificial 3GPP request/response to represent internal coordination. The Procedure graph should show the SMF's standard external interactions and the local computation that prepares them—not the Host's internal machinery.

## 5. Required traceability record

For each Tool export, maintain an Owner-side mapping record containing:
- 3GPP specification number, Release and exact version;
- clause and service-operation name;
- NF consumer and NF producer;
- interface / API resource and HTTP method or PFCP message type;
- Request and Response message / data type names;
- actual ToolCatalog Owner, kind, exact id and Effect;
- exact Input / QueryContext / Output Schema $id values;
- Mutation receipt / commit / failure / Reconcile contract, where applicable;
- implementation repository and immutable source revision for Owner documentation only, never Skill source metadata.

For each Procedure Node, maintain a Skill-side mapping record containing:
- TS 23.502 exact clause and procedure step;
- SMF role and interaction direction;
- the Request/Response exchange or tightly coupled exchange group represented by the Node;
- ToolCatalog callable IDs used by its stages;
- Capability Grants required;
- resulting facts / Evidence and Postcondition Predicate IDs.

These traceability records are profile documentation, not new fields in contract.yaml or procedure.yaml. Do not add ad-hoc 3gpp, target, api or custom metadata fields to CA-SL source.

## 6. PDU Session Establishment reference map

The initial executable profile should use the UE-requested PDU Session Establishment procedure and the SMF perspective. Pin the exact 3GPP Release and document versions before freezing any row.

| SMF-side interaction | Standard reference to pin | Tool mapping to publish | Skill Node mapping |
|---|---|---|---|
| Receive AMF request to create SM Context and return the corresponding response | TS 23.502 V18.12.0 §4.3.2.2.1; Nsmf_PDUSession_CreateSMContext in the matching TS 29.502 Release | AMF→SMF interface contract / SMF Owner's exact callable and Request/Response Schemas | Node for initial SM Context admission |
| Retrieve / subscribe to required subscription data | TS 23.502 §4.3.2.2.1; Nudm_SDM_Get / subscription service operations in TS 29.503 | UDM Owner's exact READ / subscription callable contracts and Schemas | Node(s) for subscription retrieval and subscription lifecycle |
| Establish SM Policy Association | TS 23.502 §4.3.2.2.1; Npcf_SMPolicyControl_Create in TS 29.512 | PCF Owner's exact Create callable, Effect and Request/Response Schemas | Node for policy association |
| Establish user-plane session resources | TS 23.502 §4.3.2.2.1; PFCP / N4 procedures in TS 29.244 | UPF / N4 Owner's actual PFCP Tool contract, including asynchronous completion semantics where applicable | Node for N4 session establishment and resource result |
| Transfer N1/N2 information toward AMF | TS 23.502 §4.3.2.2.1; Namf_Communication_N1N2MessageTransfer in TS 29.518 | AMF Owner's exact N1N2MessageTransfer callable and Schemas | Node for N1/N2 transfer and resulting SMF-side outcome |

This table is a traceability checklist, not an assertion that current repository ToolCatalogs already publish these v1alpha4 callables. Fill Tool IDs, Owner identities and exact Schema IDs only from real Owner releases. Missing Owner exports remain missing; do not invent them to make Compiler pass.

## 7. Asynchronous completion and Observation

If a standard interaction completes asynchronously, its Owner Tool contract must describe the actual request/response and completion/result semantics. The Skill must not pretend that request acceptance proves final resource establishment.

Use an Observation only when an authoritative state query is required and has a real Owner-published QueryContext/Output contract. Reuse a sufficient Operation Result when it already provides the required facts, Scope, Correlation, Freshness and Evidence.

Do not add a generic wait or polling Tool to the Skill language. Mutation completion, commit proof, Reconcile and responsibility remain under the Owner/Host contract. The Procedure models the business interaction and its explicit evidence checks.

## 8. Release pinning and sources

At profile freeze, pin 3GPP Release, exact TS 23.502 version, each applicable TS 29.xxx version, actual ToolCatalog and Schema release identities, and implementation source commits for each Owner (in Owner documentation, not Skill source).

Primary specifications:
- [3GPP TS 23.502 — Procedures for the 5G System](https://www.3gpp.org/dynareport/23502.htm)
- [3GPP TS 29.502 — 5G System; Session Management Services](https://www.3gpp.org/dynareport/29502.htm)
- [3GPP TS 29.503 — Unified Data Management Services](https://www.3gpp.org/dynareport/29503.htm)
- [3GPP TS 29.512 — Policy Control Service](https://www.3gpp.org/dynareport/29512.htm)
- [3GPP TS 29.518 — Access and Mobility Management Services](https://www.3gpp.org/dynareport/29518.htm)
- [3GPP TS 29.244 — PFCP](https://www.3gpp.org/dynareport/29244.htm)

Do not treat a secondary mirror, old Release PDF, implementation README or code symbol as a substitute for the pinned normative TS version.
