# Formal ToolCatalog integration and consistency report

Updated: 2026-11-12. This report covers the Compiler's imported ToolCatalog boundary. The Compiler receives catalog bytes in `BuildRequest`; it does not load this fixture by default, read registrations, invoke adapters, or execute Tools.

## Authoritative source and pinned fixture

The formal Agent-facing contract source is `odango-chan/ca-agent-demo` at `main` commit `4395982fa6efb97714b97901603b2dfb7018c4d0`, file `tools/contracts/createdsession.tool.yaml`.

| Evidence | Value |
|---|---|
| Owner Git blob ID | `05a34f38b9974742c9e2d85e0d432cf616ce01e8` |
| Source file SHA-256 | `c99ffe9c0ccbf5e65da19f689b5af47580a2975c4734954c18af4805641d7f6f` |
| Compiler fixture | `examples/ca-skills-v0.2/lib/tools/createdsession.tool.yaml` |
| Fixture test | `TestCreatedSessionToolCatalogMatchesCaAgentDemoOwnerBaseline` |

The test pins the owner commit and file digest, parses the same fixture through `InspectLibraries`, and compares all 12 Operation IDs, Effects, `inputSchema`, `outputSchema`, and optional `reconcileOutputSchema` values against the YAML decoded from the pinned bytes. This prevents the fixture and its own checksum manifest from drifting together unnoticed. The bytes are imported from the owner contract; Compiler code contains no operation schemas.

## Parsed formal Operations

| Operation ID | Effect | Reconcile Output Schema |
|---|---|---|
| `sbi.udm.sm_data.get` | READ | absent |
| `ca.session.udm_hold.bind` | READ | absent |
| `sbi.udm.sdm_subscription.create` | MUTATION | present |
| `ca.session.pcf_hold.bind` | READ | absent |
| `ca.session.n1n2_hold.bind` | READ | absent |
| `ca.session.pcf_create_inputs.read` | READ | absent |
| `sbi.pcf.sm_policy.create` | MUTATION | present |
| `ca.session.operation_context.read` | READ | absent |
| `ca.session.establishment_response.prepare` | READ | absent |
| `sbi.amf.n1n2.transfer` | MUTATION | present |
| `ca.session.evidence.await` | READ | absent |
| `ca.session.resource_responsibility.transfer` | MUTATION | present |

At the same owner revision, the independent CapabilityContracts file declares these same 12 identities and Effects, with no discrepancy. Its Git blob is `51e50545a0a97bd1ff735e1b82f95a23552c486e`. This is an owner-side consistency cross-check only: CapabilityContracts are not a Compiler input and do not replace or augment ToolCatalog schemas. The runtime registration file maps the same 12 identities to deployment implementations, but registration is not a Compiler input and carries no Schema or Effect authority. Its blob is `2299ab8acb3838b88f9c7fb70e1aad959db170bb`.

The `adapter` field is descriptive ToolCatalog metadata. Compiler does not treat adapter labels as executable symbols, URLs, network endpoints or proof of implementation/readiness.

## Compiler acceptance evidence

| Requirement | Evidence | Status |
|---|---|---|
| Parse all 12 formal Operations and YAML aliases | `TestPinnedOfficialToolCatalogInspection`; official ToolCatalog is the imported fixture | PASS |
| Fixture remains exact to the owner baseline | commit + SHA-256 pin and per-Operation schema comparison test | PASS |
| Unknown Operation fails exact linking | `TestMissingLibraryOperationFails`; shared `unknownOperation` diagnostic golden | PASS |
| Duplicate Operation fails without source-order precedence | `TestDuplicateOperationFails`, `TestNativeDiagnosticResultsMatchBrowserGoldens` | PASS |
| Missing required Tool input fails | `TestToolCallRejectsMissingRequiredCatalogInput` and cross-Skill required-input negative case | PASS |
| Unsupported binding constraints fail closed | `TestUnsupportedDraftKeywordsFailBindingWithExplicitDiagnostic`, `TestPatternImplicationIsUnsupportedInsteadOfReportedAsMismatch` | PASS for covered constraints |
| Reconcile cannot invent an output contract | `TestReconcileRequiresEarlierToolNodeWithDeclaredSchema` | PASS |
| Local `$ref` and document-local fragments | `TestProjectLocalSchemaRefsAreResolvedWithoutFilesystemOrNetwork`, schema-cycle and transitive packaging tests | PASS for covered pointer cases |
| Full Draft 2020-12 binding compatibility | See [language alignment](language-alignment-v0.1.md) | PARTIAL |

The Compiler's schema shape validator recognizes the Draft 2020-12 keywords listed in the machine review below, while the binding prover deliberately supports a smaller subset. Recognizing a keyword's shape never means the compiler proves Producer/Consumer compatibility for it. Unsupported binding assertions must remain diagnostics rather than permissive acceptance.

## Owner boundary files reviewed

At the same ca-agent-demo baseline:

- `tools/contracts/createdsession.tool.yaml` is the Agent-facing Operation contract and schema source.
- `tools/contracts/capabilities/createdsession.capabilities.yaml` independently declares the expected Effects for the same identities. This is not added to BuildRequest.
- `tools/registrations/default.yaml` maps identities to runtime implementation labels. It is excluded from BuildRequest.
- `src/ca_agent_host/runtime/tool_contract.py` describes Runtime schema validation and adapter dispatch; none of that execution behavior is imported into Compiler.
- `tools/README.md` documents the Agent-facing Tool, external API consumer, registration, adapter, and Runtime boundaries.

The owner working tree currently has unrelated untracked `lane.log` and `lane.pid`; it was inspected read-only and left untouched.

## Not accepted as a ToolCatalog

`examples/ca-skills-v0.2/skills/establish-pdu-session-mas/design/tool-contract.yaml` is a NOT_READY proposal. Its 16 Operation references remain unresolved because none is in the approved 12-Operation catalog. The Compiler does not import it or make MAS pass by fabricating Operations.

