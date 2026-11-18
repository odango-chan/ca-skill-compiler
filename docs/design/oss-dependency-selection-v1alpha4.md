> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# Open-source Dependency Selection — Compiler v1alpha4

- **Status:** ADOPTED IN CA SKILL COMPILER v1.0.0; LOCAL ACCEPTANCE PASS. Product qualification remains separate.
- **Decision date:** 2026-11-16
- **Language authority:** CA-SL v1alpha4 at [ca-skills@7f84393](https://github.com/odango-chan/ca-skills/tree/7f84393ee8c395e4d8c39a74cb5000a1f896588d).

This ADR selects the third-party engines used by the v1alpha4 Compiler. It does not change CA-SL semantics, claim implementation conformance, or qualify a real Tool Owner publication. Exact dependency versions are pinned; floating `latest` references are prohibited.

## 1. Decision summary

| Responsibility | Selected dependency | Pinned version | License | Minimum Go |
|---|---|---:|---|---:|
| YAML syntax, tokens and AST | [goccy/go-yaml](https://github.com/goccy/go-yaml/tree/v1.19.2) | `v1.19.2` | MIT | 1.21 |
| JSON Schema Draft 2020-12 engine | [santhosh-tekuri/jsonschema](https://github.com/santhosh-tekuri/jsonschema/tree/v6.0.3) | `v6.0.3` | Apache-2.0 | 1.21 |
| CEL parser and type checker | [cel-go](https://github.com/cel-expr/cel-go/tree/v0.33.2) | `v0.33.2` | Apache-2.0 / BSD-3-Clause | 1.23 |
| Python concrete-syntax parser | [gositter](https://github.com/dobbo-ca/gositter/tree/v0.2.0), importing only `grammars/python` | `v0.2.0` | MIT; Python grammar MIT | 1.25 |
| RFC 8785 JSON canonicalization | [gowebpki/jcs](https://github.com/gowebpki/jcs/tree/v1.0.2) | `v1.0.2` | Apache-2.0 | 1.21 |
| JSON, hashing, paths, sorting, UTF-8 and regular expressions | Go standard library | Go toolchain pin below | Go license | Go toolchain pin below |

**Toolchain:** Go 1.25 is the minimum required by the selected dependency set and is declared in `go.mod`. The v1.0.0 release acceptance was executed with Go 1.26.6 on Windows/amd64; that is the actual recorded release toolchain. Future runs MUST record the exact toolchain used. Compiler builds MUST remain CGo-free and pass both Native and `GOOS=js GOARCH=wasm` builds.

The selected set is intentionally small. Do not add a second YAML parser, a second JSON Schema validator, a second CEL implementation, a Python interpreter, a general-purpose regex engine, or another canonical JSON implementation.

## 2. Boundary rule: libraries parse; CA-SL defines meaning

Third-party libraries provide mature syntax parsing, generic validation and canonical serialization. They do not define the accepted CA-SL language.

The Compiler MUST keep the following CA-SL-owned layers:

- YAML AST to the exact v1alpha4 Contract / Procedure / PredicateSet source model, including unknown-field rejection, identifier rules, source locations, bounded alias expansion and the frozen scalar subset.
- The exact G.4 JSON Schema keyword allowlist, exact reference closure, F.5 conservative Source ⊆ Target assignability and all dependency identity checks.
- The exact v1alpha4 CEL AST whitelist, parameter environment, optional/nullability checks and static boolean result requirement.
- The strict Script ABI, TypedDict and annotation analysis, call-site input derivation, generated Output Schema and source mapping.
- The deterministic artifact tree, source-byte preservation, generated identifiers and failure atomicity.

A library accepting a syntax form or Schema keyword MUST NOT make that form legal in CA-SL. A library's permissive default, fallback, coercion or network loader MUST NOT weaken a frozen contract.

## 3. YAML parser

**Decision:** use `github.com/goccy/go-yaml v1.19.2` as the only YAML parser.

The parser package exposes a lexer/parser and AST with source tokens; duplicate mapping keys are rejected by default. The Compiler MUST use the AST/token path rather than immediately decoding source into generic Go maps. This preserves source positions and allows the adapter to validate exact scalar and mapping structure.

Adapter requirements:

1. Parse exactly one YAML document for each CA-SL YAML source file.
2. Reject duplicate mapping keys, custom tags, malformed anchors/aliases and unsupported document directives.
3. Enforce the frozen depth, alias-expansion and file-size limits before recursively traversing or expanding nodes.
4. Enforce the CA-SL YAML 1.2 scalar subset from token/tag/value information. Do not rely on a library's implicit scalar coercion to define CA-SL semantics.
5. Preserve line and column information in diagnostics.
6. Never re-encode or normalize author YAML. Build artifacts preserve original source bytes exactly.

The YAML 1.2, scalar and duplicate-key fixtures in the pinned conformance plan are mandatory acceptance gates. If the selected parser cannot satisfy them through its public AST/token API, stop and review the dependency choice; do not build a second YAML grammar or silently retain a parallel `yaml.v3` parser.

WP2 implements this adapter in `internal/compiler/yamlast` and removes `gopkg.in/yaml.v3` from `go.mod`, `go.sum` and all production parse paths. The old parser is not retained as a fallback. No dual-parser compatibility mode is permitted.

## 4. JSON Schema engine

**Decision:** use `github.com/santhosh-tekuri/jsonschema/v6 v6.0.3` for generic Draft 2020-12 Schema compilation and instance validation.

The library supports Draft 2020-12 and permits resources to be registered in memory. The Compiler adapter MUST:

- register only the exact, already-verified local Schema resource closure;
- use exact `$id` identities and reject missing, duplicate, cyclic, remote, fragment and otherwise unsupported references before or during compilation;
- provide a deny-by-default loader; no HTTP, HTTPS, filesystem or other ambient resource retrieval is allowed;
- require the explicit Draft 2020-12 dialect URI on every participating data-Schema root;
- validate only Schemas that have first passed the CA-SL G.4 allowlist;
- keep Compiler/API and Owner-contract meta-schemas separate from G.4 data Schemas;
- disable format and content assertions and never enable unsupported vocabularies;
- return structured validation errors which the adapter maps to the frozen diagnostic identities.

The library is a generic Schema engine, not the CA-SL type system. It MUST NOT be used to infer compatibility through coercion, defaults, permissive unions or a keyword outside G.4. Source ⊆ Target assignability remains a Compiler-owned conservative proof.

The selected module declares `regexp2` for its own upstream test suite. The Compiler MUST NOT configure or import `regexp2` as a production regex engine. G.4 excludes `pattern` and `format`; no custom regex implementation is required for the frozen data-Schema profile. The upstream test-only module requirement is not a reason to enable those keywords.

PR #8's San v6.0.3 spike remains technical evidence, not production conformance evidence. The production adapter still requires the exact G.4 positive/negative fixtures, local-reference security tests, Native/WASM parity and resource-bound tests.

## 5. CEL parser and type checker

**Decision:** use the canonical module path `cel.dev/cel-go v0.33.2`.

The Compiler uses cel-go to parse and statically check expressions against a type environment derived from the owning Predicate's declared ordered parameters and their exact linked Schemas. The Compiler does not evaluate a Predicate expression against live data.

The adapter MUST enforce the frozen v1alpha4 subset before accepting an expression:

- declared parameters and deterministic scalar constants;
- static field dot access and parentheses;
- boolean `!`, `&&`, `||`, equality / inequality and same-numeric-type ordering;
- `has(x.field)`;
- `size(x)` for string / array;
- statically boolean result.

Macros, comprehensions, loops, array indexing, object construction, dynamic member names, reflection, time, randomness, external data, undeclared functions and host extension functions are rejected. Disable macro expansion for the accepted environment and inspect the parsed/check AST against the explicit whitelist; do not infer acceptance from a successful generic CEL compile alone.

The adapter MUST preserve the frozen int64 / finite binary64 distinction and safe-integer binding range. A type-check error, overflow, non-finite value or unsupported expression is never converted to PASS. Runtime evaluation and UNKNOWN handling remain Host/Runtime responsibilities.

## 6. Python Script parser

**Decision:** use `github.com/dobbo-ca/gositter v0.2.0` and import only its Python grammar package, `github.com/dobbo-ca/gositter/grammars/python`.

gositter is a pure-Go port of the Tree-sitter v0.25.0 parser runtime. Its Python grammar is generated from `tree-sitter-python v0.25.0`; both the runtime and Python grammar are MIT licensed. The selected packages do not require CGo or a CPython interpreter.

The parser returns a Tree-sitter concrete syntax tree (CST), not CPython's abstract syntax tree. The Compiler MUST place it behind a small internal adapter and derive a Compiler-owned, constrained semantic projection for the frozen Script ABI. It MUST NOT expose gositter node types as public Compiler API.

Adapter requirements:

- inspect named nodes and exact grammar fields for the supported `main`, TypedDict, field annotation, Literal, Required / NotRequired and Annotated forms;
- reject syntax outside the CA-SL F.6 subset, unsupported declarations, dynamic annotations, ambiguous imports and malformed source;
- preserve byte offsets and map them to the existing Script diagnostic path / line / column contract;
- use the parser's bounded-progress API plus existing source-size and AST-depth limits for hostile or pathological input;
- never import, execute or evaluate user Python;
- never infer a Schema from sample values, comments, function bodies or runtime output.

The upstream gositter API is v0.x. All usage MUST be isolated behind the internal adapter so a future parser replacement does not affect Script ABI or Compiler API. Before WP9 is qualified, its Python fixtures MUST pass Native and WASM builds, malformed-input/resource tests and a pinned parity comparison with the Tree-sitter C runtime in a test-only oracle job. CGo is permitted only in that isolated oracle job, never in the Compiler binary or WASM build.

Only the Python grammar package may be imported. Do not import an aggregate grammar registry or unrelated language packages.

The Host Python interpreter version remains a Host/Runtime deployment choice and MUST NOT be inferred or pinned by the Compiler.

## 7. JCS and JSON

**Decision:** use `github.com/gowebpki/jcs v1.0.2` for RFC 8785 canonicalization.

The adapter serializes the Compiler-owned generated JSON value, canonicalizes the resulting UTF-8 JSON bytes through the library, and returns the canonical bytes. It MUST reject invalid JSON / I-JSON input and propagate canonicalization errors. It MUST NOT implement property sorting, UTF-16 key ordering, ECMAScript number formatting or string escaping itself.

JCS applies only to generated JSON Schemas and Artifact Manifests as specified by the Artifact Contract. It does not apply to author source bytes, the BuildRequest transport, or Dependency Snapshot Digest framing. The frozen binary digest algorithm and its golden vectors remain unchanged.

Required tests include the RFC 8785 vectors, UTF-16 property ordering, numeric edge cases, Unicode / escaping, duplicate properties, invalid UTF-8, non-I-JSON numbers, deterministic bytes and Native/WASM equality.

Go `encoding/json` remains the JSON transport and in-memory value parser where appropriate; `crypto/sha256`, `sort`, `path`, `bytes`, `io`, `unicode/utf8` and `regexp` remain standard-library facilities. No hand-written JCS or JSON parser is permitted.

## 8. Native, WASM, licensing and resource gates

Every selected dependency is pure Go and the production Compiler MUST build with `CGO_ENABLED=0`. The following are mandatory before a dependency-consuming WP is marked complete:

1. exact module versions are pinned in `go.mod` / `go.sum`; no floating branch or `latest`;
2. `GOWORK=off go test ./... -count=1`, `go vet ./...` and Native build pass;
3. `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build` passes;
4. actual Chromium Worker execution passes for the assigned fixtures;
5. Native and WASM diagnostics, locations, dependency closure, output paths/order and output bytes match;
6. duplicate-key, alias/depth, Schema-reference, CEL-depth and Python-parser resource bounds are tested;
7. third-party notices and a dependency/license inventory are retained with the release artifact.

The selected direct dependency licenses are MIT, Apache-2.0 and BSD-3-Clause. No GPL, AGPL, SSPL, unknown or non-redistributable dependency is accepted into the production Compiler dependency graph. Transitive dependencies MUST be reviewed whenever a pinned version changes.

## 9. Version and compatibility policy

- Pin the exact versions in §1. Upgrades require a reviewed ADR amendment, upstream changelog review, license review and the complete Native/WASM conformance gates.
- Do not keep two implementations active as fallback paths.
- Do not parse or translate legacy v0.1 / v0.2 source, old `call / with / target` syntax, old ToolCatalog / CapabilityContracts envelopes or old Script metadata.
- Do not add a compatibility mode, version sniffing, implicit conversion or legacy default branch.
- Remove superseded parser / validator code as each selected adapter replaces it; historical implementations remain available in Git history only.
- This ADR is generic Compiler infrastructure. It does not add 5GC / SMF business rules, MAS behavior, Tool Owner contracts, Host context or Runtime behavior to Compiler core.

## 10. Adoption status

All selected dependencies are integrated in the v1.0.0 Compiler release:

- goccy/go-yaml v1.19.2 is the single YAML parser/AST source;
- San v6.0.3 is integrated behind the restricted G.4 Schema profile;
- cel-go v0.33.2 provides the parser/type-checker behind the frozen CEL subset;
- gositter v0.2.0 provides the Python CST parser; user Python is never executed;
- gowebpki/jcs v1.0.2 provides RFC 8785 canonicalization.

The release commit passed the recorded local Go test, vet, CGo-free Native build, Go/WASM build, CLI E2E and real Chromium Worker parity gates. These are implementation/local verification results, not real Owner or product qualification. External publication-dependent paths and WP13 real 5GC/MAS qualification remain NOT_QUALIFIED; see [Known Limitations](../known-limitations-v1alpha4.md).
