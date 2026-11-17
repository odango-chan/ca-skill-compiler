# Compiler Pipeline v1alpha4 (implementation map)

**Status:** Implementation traceability document. It maps the frozen pipeline
stages of [Compiler Contract §5](compiler-contract-v1alpha4.md) to the code and
executable evidence that implement them in this repository. It is descriptive
only; the contract and the frozen CA-SL specification remain the authority.

The active pipeline is `validateV1Alpha4` in
[`internal/compiler/v1alpha4.go`](../../internal/compiler/v1alpha4.go). `Check`,
`Build` and `InspectDependencies` enter through
[`internal/compiler/api_v1alpha4.go`](../../internal/compiler/api_v1alpha4.go);
`Check` and `Build` run the same semantic pipeline, and `Build` additionally
assembles the artifact only when no `C_STATIC` error exists.

## Stage map

| # | Contract stage | Primary implementation | Executable evidence |
|---|---|---|---|
| 1 | BuildRequest envelope, limits, path normalization, strict JSON | `api_v1alpha4.go` (`DecodeBuildRequestJSON`, `validateBuildRequest`), `shared_limits.go`, `contract_limits.go` | `api_v1alpha4_test.go`, `contract_limits_test.go`, `cmd/skillc/e2e_test.go`, `cmd/skillc-wasm/request_size_test.go` |
| 2 | Source inventory, Skill package roots, author files | `v1alpha4.go` (root/target selection, `SKILL.md`, required files), `legacy_syntax.go` | `v1alpha4_test.go`, `source_path_test.go`, `e_early_test.go` |
| 3 | Safe YAML parse (single parser, positions) | `internal/compiler/yamlast/parser.go`, `shared_yaml.go`, `v4ParseYAML` | `yamlast/parser_test.go`, `single_parser_test.go`, `determinism_test.go` |
| 4 | Author AST (Contract / Procedure / PredicateSet) | `ast_v1alpha4.go`, `v1alpha4.go` (`v4Allowed`, `v4CheckPredicates`, `v4CheckProcedure`) | `v1alpha4_test.go`, `contract_shape_test.go`, `predicate_parameter_test.go`, `procedure_stages_test.go` |
| 5 | Snapshot integrity and digest | `dependency_snapshot_digest.go`, `dependency_identity.go` | `dependency_identity_test.go`, `determinism_test.go` |
| 6 | Trust binding, Owner attribution, deployment bundles | `dependency_identity.go`, `owner_linking.go`, `evidence_linking.go`, `api_v1alpha4.go` (envelope) | `dependency_identity_test.go`, `owner_linking_test.go`, `e_mid_test.go`, `e_route_test.go` |
| 7 | Contract index (exact callable/Schema identities) | `callable_refs.go`, `schema_registry.go` | `callable_refs_test.go`, `schema_registry_test.go`, `predicate_call_id_test.go` |
| 8 | Capability linking and Effect | `capability_linking.go` | `capability_linking_test.go`, `mutation_contracts_test.go` |
| 9 | Schema closure, G.4 allowlist, F.5 assignability | `schema_g4.go`, `schema_registry.go`, `schema_validation.go`, `schema_assignability.go` | `schema_g4_test.go`, `schema_registry_test.go`, `schema_assignability_test.go`, `context_schema_*_test.go` |
| 10 | Procedure graph, stages, routes, dominance, retry/reconcile | `procedure_graph.go`, `procedure_stages.go`, `route_eligibility.go`, `retry_contracts.go`, `retry_family.go`, `retry_reachability.go`, `mutation_contracts.go`, `reconcile_contracts.go` | `procedure_graph_test.go`, `procedure_stages_test.go`, `route_*_test.go`, `retry_*_test.go`, `mutation_contracts_test.go`, `reconcile_contracts_test.go` |
| 11 | SourcePath / dataflow / F.5 argument typing | `source_path.go`, `source_path_paths.go`, `source_path_producer.go`, `source_path_schema.go`, `argument_typing.go`, `argument_value_typing.go`, `predicate_argument_typing.go`, `script_arguments.go` | `source_path_*_test.go`, `argument_*_test.go`, `predicate_argument_typing_test.go`, `script_arguments_test.go` |
| 12 | Evidence requirements, independence, static satisfiability | `evidence_linking.go`, `evidence_independence.go`, `predicate_evidence_static.go` | `evidence_linking_test.go`, `predicate_evidence_static_test.go`, `e_final2_test.go` |
| 13 | Python CST and Script ABI (never executes Python) | `internal/pythoncst/pythoncst.go`, `script_abi.go`, `script_abi_validate.go`, `script_paths.go`, `script_value_typing.go`, `script_key_access.go`, `script_optional_keys.go`, `shared_python.go`, `shared_python_typeddict.go` | `script_abi_test.go`, `script_*_test.go`, `cmd/skillc/e2e_test.go` |
| 14 | Deterministic result, diagnostics, artifact assembly | `v1alpha4.go` (`hasErrorDiagnostic`, `failedBuildResult`), `artifact_generate.go`, `artifact_pipeline.go`, `artifact_verify_json.go`, `internal/artifact` | `artifact_generate_test.go`, `artifact_pipeline_test.go`, `artifact_verify_test.go`, `determinism_test.go`, `cmd/skillc/e2e_test.go` |

## Cross-cutting gates

| Gate | Implementation | Evidence |
|---|---|---|
| Deterministic diagnostic ordering | `sortDiagnostics` in `api_v1alpha4.go`; per-stage sorted key iteration | `determinism_test.go`, `diagnostic_inventory_test.go` |
| Frozen diagnostic identity only | Diagnostic literals are checked against the frozen Appendix H index | `diagnostic_inventory_test.go` |
| One YAML parser | `internal/compiler/yamlast` is the only YAML entry point | `single_parser_test.go` |
| Single Python parser (no execution) | `internal/pythoncst` wraps gositter; no interpreter | `script_abi_test.go` |
| Legacy syntax rejection | `legacy_syntax.go` recognizes selected v0.2 and Wait/Await field names so v1alpha4 reports the frozen unsupported-syntax diagnostic instead of a generic unknown-field error | `legacy_syntax_test.go` |
| Native / Go-WASM / Chromium consistency | `cmd/skillc-wasm`, `web/skillc-worker.js`, `scripts/chromium-parity.mjs` | `cmd/skillc-wasm/request_size_test.go`, `internal/parity/chromium_test.go` (opt-in) |
| Artifact integrity before write | `compiler.VerifyBuildResult` → `internal/artifact.Verify` | `artifact_verify_test.go`, `cmd/skillc/e2e_test.go` |

## Known fail-closed stages

The following normative branches are implemented as fail-closed because their
external trusted publication formats are not frozen; they are not silently
accepted. See [Known Limitations](../known-limitations-v1alpha4.md).

- Evidence Descriptor content, Producer/Scope/Correlation/Freshness trusted
  relations (WP7 positive path).
- Identity-input trusted publication provenance.
- Cross-snapshot Reconcile association.

The stage map identifies implementation and test evidence for the static checks that are present; it does not mean every E-case is a whole-case PASS or that every positive Build path is enabled. Feature-scoped Evidence, Mutation/Reconcile and Retry Build gates remain conservative where complete static obligations or trusted external contracts are unavailable. See [E-coverage](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md#4-e-coverage) for per-case dispositions and [Known Limitations](../known-limitations-v1alpha4.md) for the exact fail-closed boundaries.
