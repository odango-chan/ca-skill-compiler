# CA Skill Compiler v2.0.1 — YAML sibling-depth hotfix

**Release type:** patch; no change to CA-SL language target, static validation contract or artifact format.

- Product: `skillc v2.0.1`.
- Language: `connectionagent.io/v1alpha5` only.
- BuildRequest/BuildResult/DependencyInspection and Artifact API: unchanged `/v1alpha5`.
- Existing `v1.0.0` and `v2.0.0` tag, source and release assets remain immutable.

## Fixed

Compiler v2.0.0 erroneously treated goccy/go-yaml's token `Position.IndentLevel` as semantic AST nesting depth. This could incorrectly reject **flat** CA-SL Procedures with many sibling Nodes:

```text
YAML_PARSE_ERROR
skills/establish-pdu-session-mas/assurance/procedure.yaml:260:11
YAML indentation depth exceeds 64
```

The fix removes only the unreliable token-indentation preflight guard. The normalized YAML AST converter and expanded-AST validator still reject genuinely deep nesting over 64; flow nesting, alias expansion, token limits and document size checks are unchanged.

## Regression acceptance

- `TestParseAllowsManyShallowSiblingProcedureNodes`: 80 sibling Nodes parse successfully.
- `TestParseRejectsGenuinelyDeepBlockNesting`: truly deep YAML still fails closed.
- Compiler PR [#44](https://github.com/odango-chan/ca-skill-compiler/pull/44): full Go suite, vet, native and Go/WASM build, CLI E2E, Chromium Worker parity and diff check passed.
- [ca-skills PR #72](https://github.com/odango-chan/ca-skills/pull/72) checked the real MAS source with the merged patch and corrected 14 candidate Capability IDs. `YAML_PARSE_ERROR` and `IDENTIFIER_INVALID` disappeared. `DEPENDENCY_SNAPSHOT_MISSING`, `EVIDENCE_GUARANTEE_LINK_INVALID` and `PREDICATE_EVIDENCE_UNSATISFIABLE` remain correct unresolved trusted Owner/Evidence failures.

## Not included

- No MAS ToolCatalog, Owner Evidence, trusted deployment credentials or CapabilityContracts are published by this release.
- No Runtime/Temporal/Host/Provider protocol change.
- No ability to build the official MAS Skill without its complete immutable external dependency closure.
- No v1alpha4 compatibility or changed APIs.

## Official release

Publication requires a green source-validation and release workflow, builds all Native/WASM binaries from the exact event commit, publishes a new annotated `v2.0.1` tag and a GitHub Release with asset checksums, and independently re-downloads/verifies all assets. No prior tag or release is rewritten.
