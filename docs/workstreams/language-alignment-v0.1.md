# Skill Language Alignment · Compiler v0.1

Updated: 2026-11-11. This implementation report follows the narrowed Compiler contract in PR #4. ca-skills v0.2 is the only official Skill source and language authority. A synthetic PASS is not official-Skill acceptance or Runtime evidence.

## Pinned sources

| Source | Ref | Use |
|---|---|---|
| `odango-chan/ca-skills` | `codex/semantic-token-contract-v01` at `eeae62badf29f90eee257c7ef2ab504211bf7e30` | Skill Language v0.2, Script Contract, Schema Ownership, Authoring Model, official PDU and MAS Skills, approved ToolCatalog |
| Compiler design | PR #4 head `69b67c572910c4222e657758ff155269e57e60f5` | Narrow Build API and current language/linking contract |
| `odango-chan/ca-agent-demo` | Not used as a Skill source tree | May help locate Tool Owner gaps or refresh business behavior; its Skills do not define Compiler grammar |

The isolated v0.2 design-branch snapshot is [examples/ca-skills-v0.2](../../examples/ca-skills-v0.2). It has 61 Skill source files and one approved ToolCatalog, verified with a SHA-256 manifest. The snapshot includes no Host/Observation Owner Schemas. It is not the same revision as current `ca-skills/main`. `examples/ca-agent-demo-project` remains the separate, unmodified pre-migration v0 fixture.

## Current upstream divergence (verified 2026-11-11)

Read-only `git ls-remote` and fetch verified these upstream tips:

| Repository/ref | Current tip | Relationship to the v0.1 implementation contract |
|---|---|---|
| `ca-skills` `main` | `6742d4cae65c482accdf74c3cd1daf8350856d52` | Default branch calls its language specification v0.1 and labels it draft/source-derived. It includes refreshed current Skill sources and changed language semantics. |
| `ca-skills` `codex/semantic-token-contract-v01` | `eeae62badf29f90eee257c7ef2ab504211bf7e30` | This is the exact v0.2 source contract pinned by design PR #4 and the implementation's `examples/ca-skills-v0.2` fixture. |
| `ca-agent-demo` `main` | `4395982fa6efb97714b97901603b2dfb7018c4d0` | Matches the source revision named by current `ca-skills/main`'s `current-5gc-skill-baseline.md`; its ToolCatalog blob is `05a34f38b9974742c9e2d85e0d432cf616ce01e8`, matching the checked catalog bytes (SHA-256 `c99ffe9c0ccbf5e65da19f689b5af47580a2975c4734954c18af4805641d7f6f`). |

The `ca-skills/main` language documents differ materially from the PR #4 v0.2 contract. Main describes `allowedResources.localTools`, identity-bearing local Script metadata, and a broader Capability/Operation/Local Tool vocabulary; PR #4 explicitly says `localTools` does not exist and requires `main(input: dict) -> Output` with a strict nonempty Output TypedDict. The default-branch language spec is marked v0.1 Draft and source-derived, while PR #4 explicitly pins the separate v0.2 branch. This implementation remains contractually based on PR #4; accepting current `main` as normative would require a deliberate contract migration, not an implicit parser fallback.

For an independent static review, the two current Skill directories were read from `ca-skills/main` into a temporary directory and paired with the exact ToolCatalog from `ca-agent-demo/main`; neither upstream working tree was modified. Current implementation results are PDU 119 diagnostics (`93 EXTERNAL_SCHEMA_MISSING`, `9 STEP_SCOPE_UNKNOWN`, `7 BINDING_TYPE_MISMATCH`, `3 BINDING_UNSUPPORTED`, `3 SCRIPT_TYPE_UNSUPPORTED`, `2 PREDICATE_SOURCE_INCOMPLETE`, `1 CONTRACT_SPEC`, `1 SOURCE_FIELD_UNKNOWN`) and MAS 127 (`51 EXTERNAL_SCHEMA_MISSING`, `43 BINDING_UNRESOLVED`, `16 PROCEDURE_UNSUPPORTED`, `16 LIB_OPERATION_UNKNOWN`, `1 CONTRACT_SPEC`). This review confirms that latest default-branch source is not accepted by the PR #4 contract: PDU reports `localTools`, `skill.*` Predicate refs, and three legacy-metadata Scripts; MAS reports `localTools`. Remaining findings include Owner Schemas, source/link gaps and absent MAS Operations. It does not claim qualification or runtime acceptance.

## Contract-to-implementation status

| Area | Current status | Evidence / open work |
|---|---|---|
| BuildRequest / Result | PARTIAL | Shared Go/WASM Core accepts `{files, targets?}`, enforces path/size limits, sorts outputs and returns no partial output on errors. No physical `lib/` directory is needed. Complete every response limit and current machine-schema check. |
| SkillContract / metadata | PARTIAL | Standard frontmatter, document envelopes, identities, request/context schema paths and known nested fields are parsed. Remaining source field/type constraints require focused coverage. |
| Procedure graph | PARTIAL | Typed AST covers current actions, bindings, phases, explicit cross-Skill Procedure refs and dependency closure. Declared `on` edges drive node-output dominance; named phase-step outputs obey phase order. Terminal/outcome domains, cyclic flow, and complete nested Procedure input availability remain open. |
| PredicateSet | PARTIAL | Exact Predicate IDs, CEL/script evaluator forms and Observation alias/ref/scope/freshness are parsed. Script `ref` is a package-relative path and its Output Schema is generated. Predicate Script context requires request plus explicit subject/facts/observations/evidence Owner Schemas. CEL is not evaluated. |
| Script Contract | PARTIAL | Strict `main(input: dict) -> Output` and nonempty local TypedDict, nested/functional TypedDict, `Literal`, `NotRequired`, lists and supported literal `Annotated` constraints are handled without executing Python. Procedure input is per-call `with`; Predicate input is the Host-created context. Exhaustive syntax/constraint rejection remains. |
| Schema / ToolCatalog | PARTIAL | Exact approved ToolCatalog Operation IDs, effects, schemas, local references and YAML aliases are linked. There is no `lib/manifest.yaml`, `sourceStatus`, MAS proposed catalog or lifecycle state. Full Draft 2020-12 semantic coverage remains open. |
| Producer binding | PARTIAL | Request, context, Tool, Script, Procedure and named phase-step sources are checked against supplied schemas where available. Missing Owner schemas and the broad UDM nested Output remain explicit. Predicate `with` is projected through the declared subject schema. Exhaustive branch-sensitive alias analysis remains. |
| Build output | PARTIAL | Deterministic source, required imported contracts and generated Script Output schemas are packaged. All targets and selected dependency closure need current parity coverage. |
| Native/WASM/browser | PARTIAL | Shared Core tests and Chromium smoke compare generated paths/order/bytes and diagnostic vectors on synthetic input. Broader official-source and diagnostic parity remain. |
| Official source review | PARTIAL | Snapshot parses current frontmatter, Contracts, Procedures, PredicateSets, Cases and Script Output forms without those syntax diagnostics. PDU and MAS still fail on explicit Owner, Tool and binding gaps described below. |

## Owner boundaries and unresolved inputs

- Skill owns `spec.inputs.requestSchema` and Script Output `TypedDict` declarations.
- Tool owners supply Operation input/output/reconcile Schemas under logical `lib/**`.
- Host owns Subject, Facts and evidence/provenance Schemas; Observation owners supply observation payload Schemas.
- `context.requiredFacts` is only a list of fact IDs; it never stands in for a Facts Schema.
- Procedure Script input is projected per call site from `with`. Predicate Script input uses the Host-created Evaluation Context (`subject`, validated `request`, `facts`, declared `observations`, `evidence`).
- No schema is inferred from Python body access, Cases, request shape guesses or deployment configuration.

The pinned official PDU Contract references Host subject, facts and evidence plus Observation context schemas that are not included. The approved `sbi.udm.sm_data.get` Tool Output Schema declares `result` only as an unconstrained object, while the official selection Script consumes nested subscription fields; those fields stay unresolved. The official MAS Skill references 16 `ca.pdu_session.*` Operations absent from the approved 12-Operation catalog. Its design draft is not an approved ToolCatalog.

## Current static source baseline

From the current compiler on the pinned snapshot, targeted check results are nonzero by design:

- PDU: 115 diagnostics (97 `EXTERNAL_SCHEMA_MISSING`, 9 `BINDING_UNSUPPORTED`, 8 `BINDING_TYPE_MISMATCH`, 1 `SOURCE_FIELD_UNKNOWN`). Its current language artifacts and all five Script Output declarations parse; no `SKILL_METADATA_INVALID`, `CONTRACT_SPEC`, `CASE_INVALID`, `PREDICATE_SOURCE_INCOMPLETE`, `PROCEDURE_UNSUPPORTED`, `SCRIPT_TYPE_UNSUPPORTED` or unavailable-node-output diagnostic is present.
- MAS: 133 diagnostics (51 `EXTERNAL_SCHEMA_MISSING`, 43 `BINDING_UNRESOLVED`, 16 `LIB_OPERATION_UNKNOWN`, 16 `PROCEDURE_UNSUPPORTED` for absent reconcile outputs, 7 `TOOL_NOT_DECLARED`). Its current artifact envelopes parse without metadata/Case/Predicate syntax errors.

These counts are local static evidence only. They do not establish execution, Qualification, Runtime readiness, or live 5GC behavior. Keep this report refreshed whenever parser/linker behavior changes.
