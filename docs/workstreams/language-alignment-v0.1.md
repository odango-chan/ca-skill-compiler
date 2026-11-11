# Skill Language Alignment · Compiler v0.1

Updated: 2026-11-11. This is the implementation-side source comparison for PR #4 and the v0.1 implementation branch. It records current upstream identities and separates implemented structure from unresolved language and Owner contracts. A passing synthetic fixture is not a real-Skill or Runtime acceptance claim.

## Pinned upstream inputs

| Repository | Ref used | Content checked |
|---|---|---|
| `odango-chan/ca-skills` | `main` at `6742d4cae65c482accdf74c3cd1daf8350856d52` | `docs/skill-language-spec.md`, `docs/authoring-model.md`, `docs/project-layout.md`, `docs/current-5gc-skill-baseline.md`, `spec/skill-language-tokens.yaml` |
| `odango-chan/ca-agent-demo` | `main` at `4395982fa6efb97714b97901603b2dfb7018c4d0` | `skills/establish-pdu-session/**`, `skills/establish-pdu-session-mas/**`, `tools/contracts/createdsession.tool.yaml`, provider contract files |
| Compiler design | PR #4 head `159b5396135fecdf4675cf296325956c824a5279` | `docs/design/compiler-contract-v0.1.md`, `contracts/v0.1/*.schema.json` |

Both sibling repositories were inspected from fetched `origin/main` snapshots. Their worktrees were left untouched. The current-source test tree uses logical paths `skills/**` and `lib/tools/createdsession.tool.yaml`; this does not require a physical `lib/` in BuildRequest.

## Normative source comparison

| Area | Authoritative source | Current implementation status |
|---|---|---|
| Skill metadata | `ca-skills/docs/skill-language-spec.md` §3.1; standard `SKILL.md` frontmatter `name`, `description` | PARTIAL: frontmatter identity and directory matching are parsed; package contract and Markdown body are not yet a fully exposed AST. |
| Skill contract | Same spec §3.2 | PARTIAL: document identity, recognized top-level fields, request/context imports, resource lists, and predicate groups are checked; all nested field constraints and references are not yet complete. |
| Procedure DSL | Same spec §§3.3–3.5 and `authoring-model.md` §§3–6 | PARTIAL: explicit AST for Procedure/node/step structure feeds producer-output linking; call/script/procedure/verify/reconcile/terminal and node pre/post phases are recognized. Outcome-domain validation, exhaustive field constraints, and some cross-scope links remain. Current unnamed `verify` steps and their `with` forms are accepted. |
| Script contract | Same spec §3.8 versus this task's strict Compiler contract and `docs/design/script-contract-v0.md` | CONFLICT: upstream §3.8 documents legacy `connectionagent-tool` `inputSchema`/`outputSchema` metadata and annotated input/output classes. The compiler task requires producer-owned Procedure input and a strict nonempty output `TypedDict`, with no metadata I/O-schema fallback. The implementation follows the explicit strict Compiler requirement; old Scripts remain migration diagnostics. Metadata is never used to make an invalid output type compile. |
| Predicate | Same spec §3.6 | PARTIAL: PredicateSet, evaluator fields, Observation `ref/scope/freshness`, and exact Predicate IDs are structurally checked. CEL is not evaluated. Script evaluator `ref` is checked against `allowedResources.localTools`, then receives `LOCAL_TOOL_RESOLUTION_UNSPECIFIED` because the upstream symbol model does not freeze its mapping to a Script source. A verify `with` without a Predicate consumer Schema also receives an explicit missing-Schema error. |
| ToolCatalog | PR #4 `contracts/v0.1/tool-catalog.schema.json` and latest Demo `tools/contracts/createdsession.tool.yaml` | PARTIAL: exact ToolCatalog identity/metadata/operation fields, unique Operation IDs, declared Effect, local Schema refs and YAML aliases are checked. Catalog presence says nothing about implementation or readiness. MAS's design draft is not imported as an approved catalog. |
| Schema ownership | Skill Language spec §§3.5, 3.6, 3.9; explicit request/Host/Observation import paths in Compiler Procedure contract | PARTIAL: producer schemas are resolved from request, Tool, Script and Procedure data when supplied. No real Host `subject`/`facts` or complete Observation-owner schema set is present in the latest Demo input; missing owner contracts stay errors. |
| Cases | Skill Language spec §3.7 | PARTIAL: current `SkillTestCase` envelope, owner/name identity, known spec fields, classes and purpose are checked. Case context is retained as author fixture material; it is not treated as authoritative Host evidence or execution evidence. |
| Tool / Operation / Capability | Skill Language spec §3.9 | PARTIAL: exact Operation IDs link to ToolCatalog. Capability registry import is optional and absent from the current Project. Never infer Capability/Operation equivalence or deployment readiness. |
| Semantic tokens | `ca-skills/spec/skill-language-tokens.yaml` | NOT IMPLEMENTED: token registry was audited for position-sensitive syntax categories. LSP and editor token services are outside this Compiler implementation. |
| Build API | PR #4 `compiler-contract-v0.1.md` and request/result machine schemas | PARTIAL: existing Go/WASM shared API retains `{files, targets?}` and deterministic result structures; validation must be completed for every wire limit and output rule. Caller supplies all bytes; there is no source acquisition. |
| Studio ABI | `docs/studio-integration.md` plus PR #4 contract | PARTIAL: existing Worker transports base64 file bytes to the same Go Core. Browser parity and v0.1 API documentation require final rerun/update. No Studio repository is changed here. |

## Source-derived semantics and open decisions

- Preconditions are ordered, side-effect-free preparation. Postconditions are ordered business-result checks; both stop at the first non-PASS result and require explicit PASS/FAIL/UNKNOWN routes. These phases must remain attached to their owning node.
- Procedure `call` is a scalar exact Operation ID. `verify` is an exact Predicate ID. `reconcile` targets the original mutation node. Do not add a `dependency:` wrapper or infer an operation from an implementation name.
- `context.subject`, `context.request`, `context.facts`, `context.constraints`, `loop.*` where an iteration contract exists, `input.*`, node outputs, and named phase-step outputs are distinct namespaces. No implicit previous-node inputs are inferred.
- The authoring guide documents `tool`, child `procedure` and `for` forms but the current 5GC baseline does not demonstrate them. They must have explicit semantics and tests before the compiler claims support; otherwise report them as unsupported instead of silently skipping them.
- `SkillContract.spec.dependencies` is not in the current Skill Language field list. Dependency closure is derived only from explicit supported cross-Skill Procedure references. The v0-only `dependencies` fallback has been removed from current synthetic fixtures.
- The current PDU source contains 36 Cases and legacy Script declarations. The MAS Skill declares Operations absent from the approved 12-Operation catalog; its `design/tool-contract.yaml` is a draft. These are source/Owner gaps, not permission to weaken linking.

## Current real-source baseline

The reproducible snapshot is [examples/ca-agent-demo-current-main](../../examples/ca-agent-demo-current-main), pinned to the Demo `main` commit above. Its 79 Skill files and one approved ToolCatalog were verified against upstream blob IDs and have a SHA-256 manifest. The older `examples/ca-agent-demo-project` remains untouched as a separate pre-migration reference.

Under the current implementation, the conventional Skill reports 122 diagnostics: 93 `EXTERNAL_SCHEMA_MISSING` (83 Host/Observation source gaps plus 10 Predicate invocation consumer Schemas), 9 unknown step producers, 7 binding incompatibilities, 4 `BINDING_SOURCE_UNAVAILABLE` node outputs not proven available on every route, 3 unsupported legacy Script outputs, 3 unsupported binding Schemas, 2 `LOCAL_TOOL_RESOLUTION_UNSPECIFIED`, and 1 Tool output-field mismatch. It parses current frontmatter, Contracts, all 36 Cases, PredicateSet, and node phases without syntax diagnostics. MAS reports 126 diagnostics: 51 missing external Schemas, 43 unresolved producers downstream of unavailable Operations, 16 absent Operations, and 16 reconcile operations without approved reconcile output contracts. Its current artifact envelopes parse without metadata/Case/Predicate syntax diagnostics. These counts are implementation snapshots and will be refreshed as parser/linker behavior changes.

Neither result proves qualification, execution, Runtime readiness, or live 5GC behavior. The current positive case is synthetic and must stay labeled synthetic.
