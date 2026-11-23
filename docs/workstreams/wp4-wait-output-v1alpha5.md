> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP4 — v1alpha5 `wait.output` typed Producer SourcePath

**Status:** implemented on branch `feat/compiler-v1alpha5-wp4-wait-output-20261117`
(stacked on the corrected WP3 branch `feat/compiler-v1alpha5-wp3-trigger-binding-20261117`).
Candidate/DRAFT contract sources; not a language or product qualification claim.
Build remains fail-closed because WP6/WP7/WP8 are not implemented.

**Authority**

- ca-skills #61 candidate single-file spec: Appendix **L.2.3** (`wait.output`),
  **L.2.4** author example, **L.11/L.11.2** diagnostic reuse rules, **L.12**
  EW-012.
- ca-skills #62 Integration Ingress Operation contract: normalized request Schema
  and Schema Closure Digest v1 (already linked/verified by WP3).
- Goal doc WP4 row: “绑定 normalized request Schema；JSON Pointer、类型赋值、
  路径可用性及分支敏感性”.
- Dependent stacked work: WP1 IngressCatalog/Schema Closure (`internal/ingresscatalog`),
  WP2 v1alpha5 Wait AST, WP5 Procedure graph (dominance API), WP3 exact Ingress
  link and trigger/correlation binding (linked export contract + normalized
  `RootRef`).

## 1. Implemented semantics

### 1.1 Producer form (`<nodeId>.wait.output[.<property>…]`)

The WP4 pass runs after the WP3 binding pass over the same parsed Skill. Argument
positions are exactly the inherited SourcePath positions: named Tool
Operation/Observation argument values and positional PredicateCall arguments. The
inherited scalar classification decides SourcePath vs literal — a quoted string or
a YAML `bool`/`null`/`int`/`float` is a literal; only a plain `!!str` is treated as
a SourcePath (this mirrors `validateV1Alpha4SourcePaths`; no second grammar
engine is introduced).

Only paths whose second segment is literally `wait` are handled by this pass; the
remaining inherited SourcePath grammar/typing/availability rules for
`<node>.operation.output` / `<node>.observation.output` / `context.*` are not
re-implemented here (see §4).

For `X.wait.output…`:

- every dotted segment must be an inherited Identifier (`[A-Za-z][A-Za-z0-9_-]*`);
- `X` must be a **declared Wait-bearing Node** of the same Procedure;
- the third segment must be exactly `output`;
- otherwise `SOURCE_PATH_INVALID` anchored at the consuming argument value node.

`<node>.wait.output` is **not** a v1alpha4 Producer form: the frozen v1alpha4
grammar keeps rejecting it with `SOURCE_PATH_INVALID`, and the v1alpha4 pipeline
does not run any WP4 code.

### 1.2 Binding to the exact normalized request Schema

For the linked Wait Node, WP3 has already proved exactly one trusted Ingress
Operation export and verified its `request.normalizedSchemaRef` through the
trusted v1alpha5 `schemaBindings` and a valid Schema Closure Digest v1 closure.
WP4 consumes that same contract (it is passed forward as
`v1alpha5WaitBindingSource`; trusted discovery is not re-done):

- bare `X.wait.output` → producer type is the resolved normalized request object
  Schema root (`ResolutionContext.ResolveRoot`);
- `X.wait.output.p.q` → the dotted segments are converted to a canonical JSON
  Pointer (`#/p/q`) and resolved with `ResolutionContext.LookupSchemaPath` — the
  same exact `$ref`-following trusted Schema lookup WP3 uses for correlation
  paths. A property the normalized Schema does not declare exactly is
  `SOURCE_PATH_INVALID` (“undeclared Wait output Producer property”, L.11.2).

The value is ordinary typed input data: it is not injected into global Context,
not treated as an Operation/Observation Result and not treated as Evidence.
Transport headers, correlation keys, `expectedIngressRef`, `deliveryRef`, response
handles and lifecycle metadata stay private unless the public normalized request
Schema declares a field as business input.

### 1.3 Availability / branch sensitivity (EW-012)

Availability is proven with the **WP5 Procedure graph** and its cached dominator
sets:

- A Wait-bearing Node has a Wait entry-control vertex (`N@entry`) and a business
  vertex (`N`). The business vertex is reachable only through the implicit
  `EVENT_RECEIVED` control edge; the authored `TIMEOUT` edge leaves the entry
  vertex for the timeout target and never passes the business vertex.
- A consumer may read `X.wait.output` exactly when the Wait business vertex `X`
  **dominates** the consumer Node’s business vertex: every path from the Procedure
  start then passes event acceptance. A Node dominates itself, so any declared
  business stage of the Wait-bearing Node (including its Precondition) may read
  the value.
- A timeout handler, or any consumer that is also reachable through a path that
  bypasses event acceptance, is not dominated → `SOURCE_NOT_AVAILABLE`, anchored
  at the consuming argument value node.
- A consumer in a Node unreachable from the start has no dominator set and is
  likewise rejected (consistent with the inherited v1alpha4 producer dominance
  rule).
- After invalidation/cancellation/timeout the Runtime never re-populates the
  value; the Compiler rule covers the static side (no consumer on the TIMEOUT or
  bypass path).

The producer form/Schema check is skipped when the WP3 prerequisite failed (its
own `WAIT_INGRESS_EXPORT_NOT_FOUND` / `DEPENDENCY_*` / `SCHEMA_*` diagnostic is
emitted, no cascade). The pure producer-form and availability rules still apply
because they do not depend on the export contract.

### 1.4 Type assignment (conservative F.5)

Where the consumer’s expected type is statically known, WP4 reuses the existing
G.4/F.5 assignability engine:

- Operation consumers: the ToolCatalog Operation export’s `inputSchema` property
  for the argument key, resolved through the trusted v1alpha5 Schema bindings with
  `LookupSchemaPath`;
- Observation consumers: the `queryContextSchema` property likewise;
- consumer Tool resolution is best-effort and silent, reusing the inherited WP3A
  `buildSnapshotExportIndex` / `attributeExportOwner` machinery (no second
  ToolCatalog parser). An unresolved/ambiguous consumer Tool or an unbound Schema
  leaves the expected type unknown and produces no diagnostic from this pass —
  consumer Tool linking is a later work package, not a WP4 rule;
- `f5Assignable(source wait.output Schema, target argument Schema)`:
  `assignableUnresolved` (Schema not fully provable, e.g. an unresolved nested
  `$ref`) is conservative non-typing, not a mismatch; a proven incompatibility is
  `INPUT_TYPE_MISMATCH`, or `INPUT_NULLABILITY_MISMATCH` for the inherited
  nullability reason. No Wait-specific code is invented (L.11.2).
- Predicate parameter typing is **deferred**: the v1alpha5 WP2 AST does not parse
  Predicate declarations (only the Procedure), and the candidate v1alpha5 sources
  do not yet define how a Predicate parameter Schema identity binds to the
  v1alpha5 `TrustedBuildContext.schemaBindings`; implementing it would require a
  separate Predicate-registry work package rather than a guess. Predicate read
  positions still get producer-form, Schema-binding and availability validation.

## 2. Diagnostics implemented

Only inherited/registered identities are used; no Wait-specific code is added.

| Code | Phase/Severity | Trigger in this slice | Anchor |
|---|---|---|---|
| `SOURCE_PATH_INVALID` | C_STATIC / error | not a declared Wait-bearing root; malformed `<nodeId>.wait.[output]` form; invalid Identifier segment; undeclared normalized-request property path | consuming argument value |
| `SOURCE_NOT_AVAILABLE` | C_STATIC / error | Wait business vertex does not dominate the consumer Node (TIMEOUT handler, bypass path, unreachable consumer) | consuming argument value |
| `INPUT_TYPE_MISMATCH` | C_STATIC / error | proven F.5 incompatibility between the `wait.output` Schema and the statically known consumer argument Schema | consuming argument value |
| `INPUT_NULLABILITY_MISMATCH` | C_STATIC / error | inherited F.5 nullability reason | consuming argument value |

## 3. Clause → executable test traceability

Tests live in `internal/compiler/v1alpha5_output_test.go`; the trusted linking
fixtures are reused from `v1alpha5_binding_test.go` (same package) and extended
additively with a consumer Operation/Observation Input/QueryContext Schema. All
fixtures are clearly labelled synthetic; digests are derived from the fixture
bytes by the same WP1 Schema Closure Digest v1 code.

| Clause / case | Test |
|---|---|
| L.2.3 same-Node business-stage consumption (EW-001 static side) | `TestV1Alpha5WP4PositiveSameNodeRead` |
| L.2.4 same-Node Precondition consumption after event acceptance | `TestV1Alpha5WP4SameNodePreconditionRead` |
| inherited SourcePath positions include positional Predicate arguments | `TestV1Alpha5WP4PredicateArgumentPositions` |
| L.2.3 downstream dominated consumption + JSON Pointer property path | `TestV1Alpha5WP4PositiveDownstreamDominatedRead` |
| L.2.3/EW-012 timeout handler must not read the output | `TestV1Alpha5WP4TimeoutHandlerReadIsUnavailable` |
| L.2.3 path that does not accept the event (bypass) | `TestV1Alpha5WP4BypassPathIsUnavailable` |
| L.11.2 undeclared Wait output property | `TestV1Alpha5WP4UnknownProperty` |
| producer form: only a declared Wait-bearing Node | `TestV1Alpha5WP4NonWaitProducerRoot` |
| inherited SourcePath segment grammar | `TestV1Alpha5WP4InvalidSegment` |
| L.11.2 `INPUT_TYPE_MISMATCH` for Operation Input Schema | `TestV1Alpha5WP4TypeMismatch` |
| L.11.2 `INPUT_TYPE_MISMATCH` for Observation QueryContext Schema | `TestV1Alpha5WP4ObservationTypeMismatch` |
| determinism | `TestV1Alpha5WP4Determinism` |
| v1alpha4 keeps rejecting wait as a Producer source | `TestV1Alpha5WP4V1Alpha4RejectsWaitProducerSourcePath` |
| end-to-end public `Check` positive with the complete trusted fixture | `TestV1Alpha5WP4PositiveSameNodeRead` (also `TestCheckV1Alpha5AcceptsValidProcedure`) |
| Build stays fail-closed after WP4 | `TestBuildV1Alpha5FailsClosedAfterASTValidation` |

## 4. Deliberate scope decisions and deferred work

- **No second SourcePath parser**: the pass interprets only the `wait` Producer
  form. The inherited v1alpha4 SourcePath grammar for
  `<node>.operation.output` / `<node>.observation.output` / `context.*` and its
  producer/context typing is unchanged and not duplicated here. A malformed plain
  `<node>.operation…` path keeps whatever behaviour the inherited pipeline has.
- **Predicate parameter typing deferred** (see §1.4) — no parsed v1alpha5
  Predicate registry exists to bind parameter Schema identities.
- **Required-ness of a pointer path is not enforced**: `LookupSchemaPath` proves
  exact declaration and returns the resolved property Schema, but does not expose
  the parent `required` set. A normalized request property that is declared but
  not guaranteed present is not yet `SOURCE_NOT_AVAILABLE`; that requires an
  additive trusted-lookup projection and is deferred rather than approximated.
- **Normalization mapping completeness** (source→normalized field coverage and
  transform validity) remains Ingress-owned
  (`INGRESS_NORMALIZATION_INVALID`) and is not re-checked here; WP3 already proves
  the normalized Schema Closure.
- **Full Invocation Context / request typing** (Host Context Schema) remains
  deferred; this slice only types `wait.output` consumers with a statically known
  ToolCatalog Input/QueryContext Schema.
- **Response boundary and typed response mapping remain WP6**
  (`WAIT_RESPONSE_BOUNDARY_INVALID`, `WAIT_RESPONSE_MAPPING_INVALID`); WP4 does not
  treat `wait.output` as a response boundary Producer.
- **EVENT_RECEIVED/TIMEOUT are control edges**, never Stage Outcomes, Evidence or
  Goal proofs (WP5). WP4 never turns event acceptance into a Goal proof.
- **Real Owner/Integration artifacts and 5GC profile pins** remain outside the
  compiler repository; every fixture here is synthetic and is not an Owner
  publication or interoperability claim.
- Conformance mapping (WP7), artifact assembly / WASM / Chromium parity (WP8) are
  other work packages.

## 5. Exact commands and results

Environment: Windows, `go version go1.26.6 windows/amd64`, run 2026-11-17
(Asia/Shanghai). All commands from the repository worktree root.

```powershell
$env:GOWORK='off'; go test ./... -count=1
# ok  cmd/skillc 4.530s, cmd/skillc-wasm 3.526s, internal/artifact 0.673s,
#     internal/compiler 3.223s, internal/compiler/yamlast 0.566s,
#     internal/ingresscatalog 0.924s, internal/parity 0.494s,
#     internal/projectinput 3.541s — all PASS, exit 0

$env:GOWORK='off'; go vet ./...
# no output, exit 0

$env:CGO_ENABLED='0'; $env:GOWORK='off'; go build ./...
# no output, exit 0

$env:GOWORK='off'; go test ./internal/compiler -count=1 -run 'V1Alpha5' -v
# 80 top-level tests PASS, 0 FAIL; ok ... 0.550s

git diff --check
# no output, exit 0
```

All prior WP tests (v1alpha4, WP1, WP2, WP5, WP3) stay green. No existing test
assertion needed to be changed: the WP3 positive fixture already reads
`wait-sm-context.wait.output` in the same Node and remains zero-diagnostic, and
the WP2/WP5 fixtures call the AST/graph layers directly.
