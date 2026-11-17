# WP5 — v1alpha5 Procedure Graph with Wait entry control vertices

**Status:** candidate implementation on
`feat/compiler-v1alpha5-wp5-procedure-graph-20261117` (Draft PR, stacked on
WP2 PR #30 / branch `feat/compiler-v1alpha5-wait-ast-20261117`). This is static
Compiler source admission for a **candidate** language, not language
publication, product qualification, Host/Runtime implementation or 5GC/MAS
qualification. v1alpha4 remains the only published authoring contract.

**Language target:** `connectionagent.io/v1alpha5` (ca-skills #61 candidate).

## 1. Scope delivered

| Area | Delivered |
|---|---|
| Graph vertices | Every authored Node; a distinct Wait entry-control vertex (`<nodeId>@entry`) for every Wait-bearing Node; the four terminals `SUCCEEDED`/`FAILED`/`SAFE_STOPPED`/`UNRESOLVED` |
| Wait entry control | Every inter-node route edge that targets a Wait-bearing Node targets that Node's entry vertex, so the Wait gate runs on every incoming path (L.2.1) |
| Implicit event edge | Entry vertex → the Node's own first business stage as an `EVENT_RECEIVED` control edge; never a Stage Outcome, Predicate PASS, Evidence or Goal proof (L.2.1, L.11) |
| TIMEOUT edge | Entry vertex → `wait.routes.TIMEOUT` target as a `TIMEOUT` control edge (L.2.2) |
| Business-stage edges | Inherited v1alpha4 stage outcome → route semantics; no new route key |
| Start | The Procedure start vertex; the entry vertex when the start Node carries a Wait |
| Route identity | Undeclared business-stage route target and grammatically valid undeclared TIMEOUT target → inherited `CONTROL_FLOW_DANGLING`; unknown `response.after` target → `WAIT_RESPONSE_BOUNDARY_INVALID` (L.11) |
| DAG | Cycle through Wait entry vertices and `EVENT_RECEIVED`/`TIMEOUT` edges → inherited `CONTROL_FLOW_CYCLE` |
| Goal proof | Only a reachable final Postcondition PASS route directly targeting `SUCCEEDED` proves the Goal; `FINAL_POSTCONDITION_MISSING` / `SUCCESS_BYPASSES_FINAL_POSTCONDITION` reused; event receipt alone proves nothing |
| Graph API | Reachability, dominator and post-dominator computation with `Dominates`/`PostDominates` queries, deterministic and reusable by WP4/WP6 |
| Pipeline | Graph validation wired into `CheckV1Alpha5` after the WP2 AST parse; Build stays fail-closed |
| v1alpha4 | Unchanged code and behavior; no v1alpha4 file modified |

## 2. Files

| File | Responsibility |
|---|---|
| `internal/compiler/v1alpha5_graph.go` | v1alpha5 Procedure graph construction, Wait entry vertices, route-target/DAG/final-success validation, deterministic dominator/post-dominator API |
| `internal/compiler/v1alpha5_graph_test.go` | Graph structure, entry dominance, route identity, DAG, Goal proof, dominator algorithm, multi-Wait, determinism, public Check |
| `internal/compiler/v1alpha5.go` | Pipeline wiring: graph validation runs after `parseV1Alpha5SkillAST` for each selected Skill; residual fail-closed message refreshed |
| `docs/workstreams/wp5-procedure-graph-v1alpha5.md` | This traceability record |

No WP2 file was rewritten. `v1alpha5_ast.go` and `v1alpha5_wait.go` are consumed
as-is; no additive accessor was required (the Wait AST already exposes
`Routes.Timeout`, `Routes.Raw`, `Response.After` and `Response.Raw`). The only
change outside the two new files is the pipeline wiring and its comment/message
in `v1alpha5.go`.

## 3. Graph design

### 3.1 Vertex and edge model

```text
Node without Wait:  [node]
Node with Wait:     [node@entry] --EVENT_RECEIVED--> [node]
                             \--TIMEOUT--> [target node/entry | terminal]

route from any business stage -> target Node
    target Node without Wait -> [target]
    target Node with Wait    -> [target@entry]   (Wait must run on every path)
```

- `<nodeId>@entry` cannot collide with an authored NodeId because `@` is outside
  the inherited NodeId grammar (`^[A-Za-z][A-Za-z0-9_-]*$`).
- The business-stage vertex of a Wait-bearing Node is reached only by its own
  `EVENT_RECEIVED` edge. All incoming route edges target the entry vertex, so the
  entry vertex dominates the Node's business stages structurally, not merely by
  convention.
- `EVENT_RECEIVED` and `TIMEOUT` are control edges with `Kind` distinct from
  `STAGE_ROUTE`; they carry no stage/Outcome fields. `IsGoalProofEdge` accepts
  only a reachable `STAGE_ROUTE` from `postcondition` with key `PASS` targeting
  `SUCCEEDED`.
- Declared Node ids that are not legal, non-reserved NodeIds (e.g. a declared
  `SAFE_STOPPED`) are omitted from the graph because the parser already reports
  `NODE_ID_RESERVED`; this also prevents a reserved id from colliding with a
  terminal vertex.
- Invalid grammar values (e.g. a missing/invalid TIMEOUT target) are already
  diagnosed by WP2 and are not re-reported. The graph adds edges only for
  resolved targets. An undeclared but grammatically valid **business-route or
  TIMEOUT NodeId** target is diagnosed as `CONTROL_FLOW_DANGLING`; an undeclared
  **`response.after`** target is diagnosed as `WAIT_RESPONSE_BOUNDARY_INVALID`
  (Appendix L.11), not as a generic graph-target error.

### 3.2 Validation rules (inherited / Appendix L.11 identities only)

| Rule | Diagnostic | Anchor |
|---|---|---|
| Business-stage route value is neither a terminal nor a declared Node | `CONTROL_FLOW_DANGLING` | route value node |
| Grammatically valid TIMEOUT NodeId target is undeclared | `CONTROL_FLOW_DANGLING` | `TIMEOUT` value node |
| `response.after` is a grammatically valid NodeId but not a declared Node | `WAIT_RESPONSE_BOUNDARY_INVALID` (WP5 declaration check; reachability/post-dominance of a declared boundary is WP6) | `after` value node |
| Start-reachable cycle including Wait entry vertices and `EVENT_RECEIVED`/`TIMEOUT` edges | `CONTROL_FLOW_CYCLE` | Procedure root |
| A reachable `SUCCEEDED` target exists but no reachable final Postcondition PASS proof does | `FINAL_POSTCONDITION_MISSING` | Procedure root |
| A legal Goal proof exists and another reachable non-final stage routes `SUCCEEDED` | `SUCCESS_BYPASSES_FINAL_POSTCONDITION` | route value node |

All diagnostics are `Severity=error`, `Phase=C_STATIC`, build-blocking, and are
ordered by the shared `sortDiagnostics` (deterministic). No new diagnostic Code
is introduced; `WAIT_TIMEOUT_ROUTE_INVALID` (WP2) remains the identity for a
missing/extra Wait route key or a forbidden TIMEOUT target such as `SUCCEEDED`.

### 3.3 Reachability, dominators and post-dominators (WP4/WP6 API)

`V1Alpha5ProcedureGraph` exposes:

- `ReachableFrom(id)` / `ReachableVertices()`;
- `Dominators()` / `DominatorSet(vertex)` / `Dominates(dominator, vertex)`;
- `PostDominators()` / `PostDominatorSet(vertex)` /
  `PostDominates(postDominator, vertex)`;
- `Vertices()`, `Edges()`, `EdgesFrom`, `EdgesTo`, `Start`, `StartNode`,
  `WaitEntryVertex`, `HasCycleFromStart`, `IsGoalProofEdge`.

Dominators are the standard iterative fixed point over the start-reachable
graph. Post-dominators are computed by adding an internal virtual exit connected
from all terminal vertices and computing dominators on the reversed graph; the
virtual exit is never exposed. A vertex `v` post-dominates `u` iff every path
from `u` to any terminal vertex passes through `v`. Vertices that cannot reach a
terminal are excluded from post-dominator sets. Sets are cached once and all
order-dependent construction is sorted, so results are deterministic.

This API is the intended input for WP4 (`wait.output` availability on every path
dominated by the event-accepted edge) and WP6 (response-boundary
post-dominance). Neither WP4 nor WP6 logic is implemented here.

### 3.4 `response.after` identity split

Appendix L.11 assigns `WAIT_RESPONSE_BOUNDARY_INVALID` to a
`response.after` that is unknown/unreachable, bypassed by a normal accepted
path, preceded by a success terminal, or omitted when the export requires a later
boundary. Accordingly:

- **unknown / not a declared Node** → `WAIT_RESPONSE_BOUNDARY_INVALID` /
  `C_STATIC` / `error`, anchored at the `after` value node. This WP5 declaration
  check is implemented in `v1alpha5_graph.go`.
- **declared Node, but unreachable / not post-dominating every accepted-event
  path** → still `WAIT_RESPONSE_BOUNDARY_INVALID`, but the reachability and
  post-dominance proof is **WP6**; WP5 exposes the dominator/post-dominator API
  WP6 needs and does not claim that proof.
- **absence** of `response`/`after` → default boundary is the Wait-bearing Node's
  own completion (L.8.1); the export-required later boundary is WP6.

`CONTROL_FLOW_DANGLING` remains the identity for an undeclared business-stage
route target and for a grammatically valid undeclared TIMEOUT NodeId target
(L.2.2 explicitly assigns that case to the existing graph diagnostic).

## 4. Clause → test traceability

### 4.1 Appendix L.2.1 / L.2.2 / L.2.3

| Clause rule | WP5 evidence |
|---|---|
| L.2.1 Wait is entry control, evaluated after control flow selects the Node and before its first business stage | `TestV1Alpha5GraphWaitEntryVerticesAndControlEdges`, `TestV1Alpha5GraphWaitEntryDominance` |
| L.2.1 Event acceptance enters the Node's first declared business stage and resumes that same Node | `TestV1Alpha5GraphWaitEntryVerticesAndControlEdges` (entry → business vertex `EVENT_RECEIVED`), `TestV1Alpha5GraphWaitBearingStartNode` |
| L.2.1 Event receipt is implicit; authors must not declare `EVENT_RECEIVED` | Modeled as an implicit graph edge; author route key rejected by WP2 `WAIT_TIMEOUT_ROUTE_INVALID` (`TestV1Alpha5WaitDiagnosticAnchors/EVENT_RECEIVED_extra_route_key_anchors_at_key`) |
| L.2.1 Event receipt is not a Stage Outcome / Predicate PASS / Evidence / Goal proof | `TestV1Alpha5GraphEventReceivedIsNotGoalProof` |
| L.2.1 A Wait-bearing Node is otherwise an ordinary business Node | Entry + business vertex model; positive diagnostics tests |
| L.2.2 TIMEOUT is mandatory and must be a declared Node or exactly `FAILED`/`UNRESOLVED`/`SAFE_STOPPED`; `SUCCEEDED` forbidden | Declaration: `TestV1Alpha5GraphDanglingTimeoutTarget`; forbidden success target: WP2 `WAIT_TIMEOUT_ROUTE_INVALID` tests |
| L.2.2 Undeclared (but grammatically valid) TIMEOUT target uses the existing `CONTROL_FLOW_DANGLING` | `TestV1Alpha5GraphDanglingTimeoutTarget` (exact line/column) |
| L.2.2 No other Wait route key is permitted | WP2 parser behavior, unchanged |
| L.2.2 The timeout branch may not make the Run succeed directly; only a final Postcondition PASS edge may target `SUCCEEDED` | `TestV1Alpha5GraphEventReceivedIsNotGoalProof`, `TestV1Alpha5GraphFinalSuccessRequiresPostconditionPASS` |
| L.2.3 `wait.output` availability is path-sensitive (after event acceptance, not on TIMEOUT/bypass) | Graph dominance/post-dominance API implemented and unit-tested (`TestV1Alpha5GraphDominatorAndPostDominatorAlgorithm`); actual SourcePath availability is WP4 |
| L.2.3 A timeout handler must not read the timed-out Wait's output | TIMEOUT edge does not pass through the business vertex: `TestV1Alpha5GraphWaitEntryDominance`; SourcePath enforcement is WP4 |
| L.2.4 Author example shape (Wait entry + timeout handler + downstream convergence) | `v1alpha5MultiWaitProcedure` / `TestV1Alpha5GraphMultiWaitProcedure` |

### 4.2 Appendix L.11 diagnostics and L.12 EW cases

| Item | WP5 status |
|---|---|
| L.11 reuse of `CONTROL_FLOW_DANGLING`, `CONTROL_FLOW_CYCLE`, `FINAL_POSTCONDITION_MISSING`, `SUCCESS_BYPASSES_FINAL_POSTCONDITION`; no invented code; `WAIT_RESPONSE_BOUNDARY_INVALID` (L.11) for an unknown `response.after` | Implemented; asserted across the graph tests |
| EW-012 (`wait.output` referenced on a timeout/bypass path → `SOURCE_NOT_AVAILABLE`) | Graph dependency and dominance API ready; SourcePath rule deferred to WP4 — **NOT_IMPLEMENTED in WP5** |
| EW-013 (Wait Node has no business stage → `WAIT_NODE_STAGE_REQUIRED`) | Delivered by WP2: `TestV1Alpha5WaitDiagnosticAnchors/wait-only_node_anchors_at_wait_mapping` |
| EW-014 (TIMEOUT route targets `SUCCEEDED` → `WAIT_TIMEOUT_ROUTE_INVALID`) | Delivered by WP2: `TestV1Alpha5WaitDiagnosticAnchors/TIMEOUT_to_SUCCEEDED_anchors_at_value`; WP5 graph never creates a TIMEOUT edge to `SUCCEEDED` (`TestV1Alpha5GraphEventReceivedIsNotGoalProof` edge-kind assertion) |
| EW-016 (one incoming path cannot derive correlation before trigger → `WAIT_CORRELATION_BINDING_INVALID`) | Deferred to WP3; WP5 supplies the per-incoming-path graph (all incoming edges target the entry vertex) — **NOT_IMPLEMENTED in WP5** |
| EW-019 (`response.after` bypassed by a normal accepted path → `WAIT_RESPONSE_BOUNDARY_INVALID`) | Deferred to WP6; WP5 provides `Dominates`/`PostDominates` — **NOT_IMPLEMENTED in WP5** |
| EW-030 (`response.after` names an unknown or unreachable Node) | Declaration half implemented as `WAIT_RESPONSE_BOUNDARY_INVALID` (`TestV1Alpha5GraphUnknownResponseAfterTarget`); declared-but-unreachable/post-dominance half deferred to WP6 — **PARTIAL** |
| EW-033 (Wait routes mapping or TIMEOUT key missing → `WAIT_TIMEOUT_ROUTE_INVALID`) | Delivered by WP2: `TestV1Alpha5WaitMissingRequiredSections` |
| EW-034 (author declares `EVENT_RECEIVED` or another key under `wait.routes`) | Delivered by WP2 test; WP5 models the normal event edge as implicit control (`TestV1Alpha5GraphWaitEntryVerticesAndControlEdges`) |

### 4.3 Graph-specific tests

| Test | Proves |
|---|---|
| `TestV1Alpha5GraphWaitEntryVerticesAndControlEdges` | Vertex/edge model, entry vertex ids, terminal vertices, all incoming route edges target the entry vertex |
| `TestV1Alpha5GraphWaitEntryDominance` | Entry dominates the Node; business vertex entered only by `EVENT_RECEIVED`; TIMEOUT branch cannot reach the Node or its stages |
| `TestV1Alpha5GraphWaitBearingStartNode` | Wait-bearing start Node starts at its entry vertex |
| `TestV1Alpha5GraphPositiveProcedureHasZeroDiagnostics` | Positive single-Wait, multi-Wait, Wait-start Procedures through the public `Check` path |
| `TestV1Alpha5GraphTimeoutEdgeCycle` | `CONTROL_FLOW_CYCLE` for a TIMEOUT edge re-entering an earlier Node and for a self-target |
| `TestV1Alpha5GraphDanglingTimeoutTarget` / `.../BusinessRouteTarget` | `CONTROL_FLOW_DANGLING` at the exact offending value with `C_STATIC`/`error` |
| `TestV1Alpha5GraphUnknownResponseAfterTarget` / `TestV1Alpha5GraphDeclaredResponseAfterTargetIsAccepted` | Unknown `response.after` → `WAIT_RESPONSE_BOUNDARY_INVALID` (not `CONTROL_FLOW_DANGLING`); a declared boundary Node is accepted (declaration-only check) |
| `TestV1Alpha5GraphFinalSuccessRequiresPostconditionPASS` | Missing final proof, legal proof, bypass, and unreachable illegal success |
| `TestV1Alpha5GraphEventReceivedIsNotGoalProof` | Control edges are never Goal proofs; only final Postcondition PASS counts |
| `TestV1Alpha5GraphDominatorAndPostDominatorAlgorithm` | Dominator/post-dominator unit semantics, multi-path joins, virtual exit not exposed, unreachable vertices |
| `TestV1Alpha5GraphMultiWaitProcedure` | Two sequential Wait entries, both dominated by their entry vertices, second entered through its entry vertex |
| `TestV1Alpha5GraphDiagnosticsAreDeterministic` | Byte-identical repeated `Check` diagnostics and repeated graph construction |
| `TestV1Alpha5GraphUnreachableWaitNodeIsNotDiagnosedOnItsOwn` | Reachability is exposed but WP5 invents no `UNREACHABLE_NODE` diagnostic |

## 5. Local verification (executed)

Environment: Windows, Go `go1.26.6 windows/amd64`, `GOWORK=off`,
`CGO_ENABLED=0` where stated. Node/Chromium parity is not required for this WP
(no Native/WASM/Worker adapter change).

| Command | Result |
|---|---|
| `GOWORK=off CGO_ENABLED=0 go test ./... -count=1` | PASS |
| `GOWORK=off go vet ./...` | PASS |
| `GOWORK=off CGO_ENABLED=0 go build ./...` | PASS |
| `GOWORK=off CGO_ENABLED=0 go test ./internal/compiler -count=1 -run 'V1Alpha5' -v` | PASS |
| `gofmt -l internal/compiler/v1alpha5_graph.go internal/compiler/v1alpha5_graph_test.go internal/compiler/v1alpha5.go` | PASS (no files listed) |
| `git diff --check` | PASS (no whitespace errors) |

The focused run includes the existing WP2 suites and the new WP5 graph suite; no
WP2 test expectation was weakened. `Build` remains fail-closed through the
internal residual gate for WP3–WP8.

CI: no GitHub Actions runner result was observed in this session; CI is
**NOT_RUN / CI_UNAVAILABLE**, never claimed as PASS. All claims above are local
execution evidence only.

## 6. Deferred scope (explicit)

| Item | Owner / next step |
|---|---|
| Trigger Operation binding and per-incoming-path correlation proof (`WAIT_CORRELATION_BINDING_INVALID`, typed `armInputSchema`) | WP3 |
| `wait.output` Producer SourcePath, normalized request Schema, path-sensitive availability (`SOURCE_NOT_AVAILABLE`/`SOURCE_PATH_INVALID`, EW-012) using the graph dominance API | WP4 |
| `response.after` reachability and post-dominance for a *declared* boundary Node, boundary Producer/`responseResultSchema`, response dispositions (`WAIT_RESPONSE_BOUNDARY_INVALID` reachability cases, `WAIT_RESPONSE_MAPPING_INVALID`, EW-019/EW-021/EW-030 reachability) | WP6 |
| Full inherited v1alpha4 stage route cardinality/eligibility matrix for v1alpha5 (`ROUTE_OUTCOME_MISSING`, `ROUTE_NORMAL_OUTCOME_NOT_ALLOWED`, route-key eligibility) | Later v1alpha5 slice (not in WP5 scope) |
| `UNREACHABLE_NODE` warning for v1alpha5 | Deliberately not emitted in WP5; reachability is exposed via the graph API |
| EW-001–EW-036 executable fixture/diagnostic-index/messageKey mapping | WP7 |
| Native/WASM/Chromium worker parity for the v1alpha5 graph diagnostics | WP8 |
| Host maximum Wait duration admission (`WAIT_DURATION_LIMIT_EXCEEDED`) | Host/Runtime contract |
