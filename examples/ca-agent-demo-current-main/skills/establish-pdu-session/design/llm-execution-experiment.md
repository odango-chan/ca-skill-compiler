# CreatedSession weak-model execution experiment

**Status: EXPERIMENTAL / DESIGN PROPOSAL**
**Scope: establish-pdu-session only**
**Repository-wide executor rules:** [docs/design/llm-skill-execution.md](../../../docs/design/llm-skill-execution.md)

This file contains only the CreatedSession-specific experiment. Generic LLM executor,
Decision Context, concurrency, Observation mediation and MCP grounding rules are owned by
the repository-wide design.

## 1. Goal

Prove that a weak model can execute a small governed slice of
`establish-pdu-session` from:

- `SKILL.md` business guidance;
- a compact current Decision Context;
- a bounded admitted action surface;
- Harness-mediated business facts;

without reading `assurance/procedure.yaml` as a step-by-step script.

The deterministic Runtime remains unchanged by this experiment.

## 2. Model request used by this experiment

Each model decision must use the repository-wide five-part assembly:

```text
System Prompt
  -> src/ca_agent_demo/agent/host/prompts/llm-skill-executor.md

Skill Guidance
  -> skills/establish-pdu-session/SKILL.md
  -> plus only admitted model references if needed

Decision Context
  -> exact current CreatedSession execution state

Available Actions
  -> current admitted business action surface

Response Contract
  -> Host-owned structured ACT / OBSERVE / COMPLETE / STOP schema
```

Do not add execution-session/lifecycle/producer correlation fields merely so the weak
model can echo them back. The Host retains that envelope and correlates the response to
the exact request.

This PoC should preserve the provenance-bearing request-assembly style already used by
the repository's `ModelRequestAssembly` rather than create an ad-hoc concatenated
prompt path.

### Model-selected business references

`SKILL.md` lists three flat `../references/*.md` knowledge files and their
purpose. The LLM independently decides whether to read one with the optional
Host-provided `read_reference` tool. The Host checks exact-path access and the
pinned Skill identity, but does not choose the reference based on a semantic
trigger. Reading is not a fifth business response or a new Procedure.

This change provides the source package and local read capability. The pure
LLM executor still needs a tool-call/continuation adapter to expose the read
inside an actual model turn and record it in request provenance. Full runnable
mainline and real-model acceptance are NOT_RUN; a missing tool never permits
invented knowledge or filesystem access through ACT/OBSERVE.

## 2. Business slice

Do not start with the complete CreatedSession procedure.

The first slice should contain:

1. one business action whose result is meaningful but not sufficient for final success;
2. one business fact that the model must request because current state is insufficient;
3. one attempted completion that the Harness can reject through Contract closure.

A useful first candidate is around the AMF/RAN boundary:

```text
current session already has:
  usable subscription
  owned SDM subscription
  owned PCF association
  initial N4 resources

weak model:
  -> initiate the admitted N1N2 transfer business action
  -> inspect the resulting current business state
  -> request createdsession.ran.setup_accepted when needed
  -> propose completion too early

Harness:
  -> execute/guard the action
  -> bind the exact transfer producer
  -> resolve the RAN fact through the current execution session
  -> reject COMPLETE while final Contract facts are missing
```

This keeps the experiment focused on the new LLM/Harness boundary rather than UDM/PCF
setup mechanics.

## 3. Required model-facing business knowledge

The model gets the normal CreatedSession `SKILL.md`.

For this slice, its Decision Context should make clear:

- the exact current occurrence is already admitted;
- subscription/policy/initial-N4 prerequisites are currently satisfied;
- N1/N2 delivery has not yet been initiated, or has just completed;
- RAN setup is not assumed from AMF transfer success;
- final completion additionally requires N4 forwarding, UE configuration and managed
  resource responsibility.

Do not expose raw Observation tool arguments or old producer/lifecycle identifiers as
things for the model to choose.

## 4. Case A: ACT -> OBSERVE -> rejected COMPLETE

Expected semantic trajectory:

```text
Decision 1
  weak model selects the N1N2 transfer business action

Harness
  validates current hard guards
  executes the existing governed capability
  records the exact producer/operation reference
  rebuilds current state

Decision 2
  weak model sees transfer initiation is not RAN completion
  requests createdsession.ran.setup_accepted

Harness
  binds the request to the same Skill execution session
  uses the exact transfer/lifecycle provenance
  resolves through the existing Observation backend
  returns current normalized fact/state

Decision 3
  weak model proposes COMPLETE before all final Contract facts are satisfied

Harness
  performs Contract closure
  rejects final success
```

The model must never need to construct the RAN Observation MCP parameters.

## 5. Case B: concurrent correlation

Run two simultaneous execution sessions:

```text
Session A
  UE/session/lifecycle A
  transfer producer A

Session B
  UE/session/lifecycle B
  transfer producer B
```

Both weak-model executions request:

```text
createdsession.ran.setup_accepted
```

Acceptance requires that:

- A resolves only through producer/lifecycle A;
- B resolves only through producer/lifecycle B;
- no latest-per-UE/session fallback is used;
- ambiguity remains unresolved rather than guessed.

## 6. Case C: repeated OBSERVE

Have the weak model ask for the same fact more than once without a relevant world change.

The Harness should reuse a still-fresh result when allowed instead of blindly re-querying
the backend.

Record:

- model decision count;
- OBSERVE count;
- backend Observation query count;
- reused-fact count;
- ambiguous/stale binding count.

This case tests whether a weak model can be tolerated without creating an Observation
polling loop.

## 7. Case D: stale decision before mutation

Build one model request from a current Decision Context, then change a relevant guarded
state before the model's mutation is dispatched.

The Harness must reject or re-evaluate the stale action rather than trusting the model's
earlier snapshot.

The model does not need to echo an internal revision identifier; the Host correlates the
response with the request that produced it.

## 8. Acceptance criteria

The first experiment passes only if:

- a weak model can choose the intended business action from `SKILL.md` + current state;
- it can request the missing RAN business fact without knowing an Observation backend;
- the Harness grounds backend parameters from the exact current execution session;
- concurrent fact requests cannot cross producer/lifecycle boundaries;
- missing, stale or ambiguous evidence never becomes PASS;
- AMF transfer success is not treated as RAN or final success;
- COMPLETE cannot bypass Contract acceptance;
- repeated fact requests do not cause unbounded backend polling;
- no new Skill YAML syntax is required to make the experiment work.

## 9. Explicit non-goals

This experiment does not yet add:

- an LLM Operation DSL;
- a Context Semantic Catalog;
- an Observation Frontier DSL;
- new Skill YAML fields;
- a second LLM workflow graph;
- replacement Observation MCP tools;
- production Runtime compatibility claims.

If the experiment exposes a repeated missing contract, record that need before promoting
new syntax.
