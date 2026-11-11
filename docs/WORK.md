# Skill Compiler v0 implementation status

Updated: 2026-11-11. Implementation branch: `feat/compiler-v0-complete`, implementation HEAD `a4596d9005e58794910270a314b50f4a48c5eab7`, based on `origin/feat/compiler-v0-bootstrap` at `5f2a36b`. Draft [PR #3](https://github.com/odango-chan/ca-skill-compiler/pull/3) targets `feat/compiler-v0-bootstrap`. This report records observed local evidence; it does not claim a complete Compiler v0.

## Work packages

| WP | Status | Evidence / remaining work |
| --- | --- | --- |
| WP0 Go build and base tests | IMPLEMENTED | `go mod tidy`, `GOWORK=off go test ./... -count=1`, and `GOWORK=off go vet ./...` pass. `validPath` accepts `lib/manifest.yaml`; duplicate YAML keys and bounded alias traversal are checked; Native input size limits are checked before file reads. The enclosing `/home/z00587616/go.work` references absent sibling repos, so Go commands in this environment require `GOWORK=off`. |
| WP1 Procedure parser and linker | PARTIAL | Parses action positions rather than recursively scanning YAML keys; checks string-typed Procedure/node/input identifiers and action values, entry/start/branch targets, nonempty procedure/node mappings, nonempty outcome branch maps and names, Skill and local Procedure cycles, local/explicit-self/cross-Skill Procedure references, callee input names/types, input-source cycles, nested step action fields and scopes, and Predicate IDs and source structure. Contract condition groups/entries and PredicateSet root fields and Predicate evaluator/observation fields are shape-checked before reference resolution. Rejects incomplete explicit Procedure references and unresolved `with` on nested verify steps. Reconcile requires an earlier Tool-call target with declared `reconcileOutputSchema`, which is linked as the reconcile node's output producer. Full grammar and outcome semantics still need coverage. |
| WP2 Tool Library | IMPLEMENTED | Imports both pinned YAML catalogs, including aliases: 12 traditional + 16 MAS candidate Operations. Parses `lib/manifest.yaml`; `InspectLibraries` exposes schemas/effect/source maturity. MAS remains `proposed`. Duplicate-operation, in-memory project-local JSON Pointer resolution, remote/cyclic reference rejection, transitive Library packaging, read-only inspection, recursive validation of supported Schema structure, and Draft 2020 structural constraint checks pass. Broader JSON Schema keyword coverage remains. |
| WP3 Producer binding and Script output | PARTIAL | Resolves every declared Procedure input against its producer, including unused inputs, plus request, explicit Host/Observation references, Tool/node/reconcile output, and separate nested step scopes; Procedure input and call-site `from` references must be nonempty YAML strings; unsupported producer, consumer, and literal binding Schema features report `BINDING_UNSUPPORTED` rather than being mislabeled as type mismatches; checks producer/consumer object and JSON literal schemas, including declared/closed `additionalProperties`; rejects missing Host contracts and unknown producer fields. Numeric literal bindings enforce integer-to-number compatibility, exclusive bounds and exact decimal `multipleOf`; producer compatibility checks those constraints conservatively, combining regular and exclusive lower/upper bounds as an intersection. Project-local JSON Pointer refs are expanded from input files. Generates Draft 2020-12 schemas for statically parsed nonempty TypedDict outputs with primitive, Literal, list, NotRequired, Annotated, and nested TypedDict fields. Python raw regex strings, quoted punctuation, and literal constraint keys are tested. String/comment contents are masked before recognizing imports, `TypedDict` declarations and `main`. Duplicate declarations/fields/constraints and non-field TypedDict statements are rejected. No Python runs. Full JSON Schema keyword/constraint coverage, exhaustive dataflow scopes, and invocation projection reporting remain. |
| WP4 Package and Native CLI | PARTIAL | `check`, whole-project `build`, `--skill`, dependency closure, deterministic ordering, referenced read-only Tool/Schema inclusion, and generated Output schemas work. A real synthetic build wrote a 7-file tree. File/directory source-to-generated collisions are rejected. More hostile path and schema cases remain. |
| WP5 WASM and browser interface | PARTIAL | WASM exposes `check`, `build`, and `inspectLibraries`; Worker handles all three. Chromium ran the Script Contract project with an `Annotated` raw regex through check/build/inspect and compared every Native/WASM output path, order, and byte, including the generated Script Output Schema. For missing-target and invalid-binding failures, the browser and Native Go Core are checked against the same shared diagnostic vectors. Broader parity vectors remain. |
| WP6 Real Skill migration review | PARTIAL | Pinned sources were not edited. Current targeted strict checks report 124 diagnostics for PDU and 85 for MAS. Both lack authoritative `context.subject` and `context.facts` Owner schemas; PDU's three legacy Scripts and step forms need migration, and MAS reconcile references lack declared Tool outputs while its catalog remains `proposed`. Per-Skill evidence is in [migration-review-v0.md](workstreams/migration-review-v0.md). A separate migrated project was not created because doing so without Owner schemas would misrepresent the migration. |
| WP7 Final acceptance and PR | PARTIAL | Local test/build/browser evidence, updated status report, pushed implementation branch, and open Draft PR #3 are present. Full G2–G6 acceptance remains incomplete. |

## Acceptance gates

| Gate | Result | Observed evidence |
| --- | --- | --- |
| G0 Base startup | PASS | `GOWORK=off go mod tidy`; `GOWORK=off go test ./... -count=1`; `GOWORK=off go vet ./...`; Native build all pass. |
| G1 Library | PASS | Go test imports the pinned 12 + 16 catalogs and checks the 16 proposed status labels; YAML anchors/aliases, manifest status, and valid/invalid Draft 2020 structural constraints are covered. |
| G2 Graph | PARTIAL | Positive cross-Skill dependency closure, targeted builds, missing target/dependency, Skill dependency cycles, local Procedure recursion, explicit self-Skill references, callee required inputs, reconcile target ordering and output-schema links, string identifier/action requirements, malformed action and Procedure container shapes, unresolved nested verify bindings, contract Predicate group/entry shape, Predicate evaluator/observation field shape, empty node names, empty outcome maps/names, PredicateSet unknown root fields, Predicate source shape, and separate pre/postcondition scopes are tested. Reversing Request.Files order preserves full Build results and Check diagnostics. Full grammar and outcome semantics remain. |
| G3 Binding | PARTIAL | Synthetic Tool output to Script and cross-Skill Procedure input bindings pass; integer-to-number compatibility, combined regular/exclusive numeric bounds, and rejection of non-string `from` sources and explicit diagnostics for unsupported producer/consumer/literal binding Schemas are covered. Negative tests reject unknown producer fields, missing callee inputs, mismatched types, cyclic Procedure input sources, missing Host schemas, and malformed literals. Full JSON Schema keyword coverage and all source scopes remain incomplete. |
| G4 Script | PARTIAL | Strict Output generation, raw regex strings, quoted punctuation, literal constraint keys, ignored declarations inside multiline strings, and rejection of broad `dict` are tested; comprehensive type/constraint and unsupported syntax tests remain. |
| G5 Package | PARTIAL | Native CLI emitted source, imported Tool contract, and generated schema into a fresh 7-file tree. Referenced Library Schema dependencies are included; generated file/directory collisions are tested. More hostile path cases remain. |
| G6 Native/WASM parity | PARTIAL | Browser and Native Core processed the same Script Contract project, including Tool output → Script input binding and generated output schema; output paths, order, and every byte matched. Go tests compare Native diagnostics to the same shared golden vectors used by the browser for full serialized `TARGET_NOT_FOUND` and two binding diagnostics. Broader diagnostic vectors remain. |
| G7 Browser | PASS | Chromium loaded `wasm_exec.js`, `skillc.wasm`, and Worker; check/build/inspect succeeded and a missing-target diagnostic returned in-band. |
| G8 Real migration | PARTIAL | PDU: 124 expected migration/owner diagnostics (83 `EXTERNAL_SCHEMA_MISSING`, 18 `PROCEDURE_UNSUPPORTED`, 9 `BINDING_TYPE_MISMATCH`, 9 `STEP_SCOPE_UNKNOWN`, 3 `SCRIPT_TYPE_UNSUPPORTED`, 1 `SOURCE_FIELD_UNKNOWN`, 1 `BINDING_UNSUPPORTED`). MAS: 85 (51 `EXTERNAL_SCHEMA_MISSING`, 18 `BINDING_TYPE_MISMATCH`, 16 `PROCEDURE_UNSUPPORTED` for reconcile outputs absent from proposed Tools). Procedure input sources are now validated even when no node consumes them. Per-Skill gaps: [migration-review-v0.md](workstreams/migration-review-v0.md). No original snapshot changed. |
| G9 Handoff | PASS | Docs updated; branch pushed; open Draft [PR #3](https://github.com/odango-chan/ca-skill-compiler/pull/3) targets `feat/compiler-v0-bootstrap`. |

## Commands observed

All Go module commands below used `GOWORK=off` because the parent workspace references three absent sibling repositories.

~~~text
GOWORK=off go mod tidy                                            PASS
GOWORK=off go test ./... -count=1                                PASS
GOWORK=off go vet ./...                                          PASS
GOWORK=off go build -o dist/skillc ./cmd/skillc                  PASS
GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ...   PASS
GOWORK=off go run ./cmd/skillc check ./examples/minimal-project  PASS
GOWORK=off go run ./cmd/skillc check ./examples/script-contract-project PASS
GOWORK=off go run ./cmd/skillc build ./examples/script-contract-project --out /tmp/skillc-synthetic-output PASS (7 files)
GOWORK=off go run ./cmd/skillc build ./examples/script-contract-project --skill normalize --out /tmp/skillc-v0-targeted-normalize PASS (7 files)
Chromium static Worker smoke test                                 PASS
  (Script Contract fixture with raw regex; generated Schema bytes and output paths/order match Native; exact target and binding diagnostic vectors)
WASM artifact size                                               5,647,409 bytes
Chromium smoke page result                                       PASS (`data-test=pass`; rerun with unsupported-Schema diagnostic changes)
Targeted pinned reference check (PDU)                              124 migration/owner diagnostics
Targeted pinned reference check (MAS)                              85 migration/owner diagnostics
GOWORK=off go run ./cmd/skillc check ./examples/ca-agent-demo-project --skill establish-pdu-session EXPECTED NONZERO (124 diagnostics)
GOWORK=off go run ./cmd/skillc check ./examples/ca-agent-demo-project --skill establish-pdu-session-mas EXPECTED NONZERO (85 diagnostics)
git diff --check                                                  PASS
~~~

Raw-source checks intentionally return nonzero. They do not establish runtime/5GC readiness. Missing authoritative Owner Schema inputs are: `context.subject`, `context.facts`, and any referenced Observation schema. Do not substitute deployment routing configuration or infer these contracts from `requiredFacts`.

The final result remains **PARTIAL** until the remaining parser/linker and binding coverage and broader Native/WASM parity audit are complete. Draft PR #3 is open for review of this implementation slice.
