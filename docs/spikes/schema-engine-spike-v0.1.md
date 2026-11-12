# JSON Schema Engine library spike v0.1

**Status:** Spike complete; no production migration performed.
**Decision:** **CONDITIONAL ADOPT** `github.com/santhosh-tekuri/jsonschema/v6@v6.0.3`, with the tested in-memory loader and `regexp2` configuration.
**Baseline:** PR #5 checkpoint `49753ce345f8cf99654566995dd3b5aff070f844`.
**Design:** PR #6 at `751e018d4b34c2f7cdbedc625f3a3f7843180c01`.
**Spike branch:** `spike/schema-engine-library-v0.1-20261112`.

## PR #8 review remediation (2026-11-12)

- Added boundary tests against the pinned suite's Compatibility algorithm,
  including ordered disjunction ranges, unknown versions, future drafts, and
  Required/Optional/Skipped accounting invariants. The runner now separately
  counts compile failures and validation failures.
- Strengthened the closed-loader evidence with exact resolved URI and coded
  registry-denial assertions, HTTP and HTTPS loopback zero-request checks, a
  valid file-schema canary, a positive registered-relative-reference case,
  and explicit path-traversal, missing-resource, unknown-scheme, and
  custom-scheme denials.
- Removed the Kaptinlin executable module and its dependency checksums from the
  Spike harness. Its historical suite totals and failure categories remain
  below for comparison; the old implementation is no longer independently
  rerunnable from this branch.
- Re-ran Native and Node WASM, the pinned official suite, and three actual
  Chromium Worker runs on the current San-only artifacts. The detailed counts
  and measurements below supersede the earlier run's values.

This report records an isolated candidate evaluation. It does not authorize a
production dependency change or migration.

## Decision

San v6.0.3 is the candidate to carry into a separately reviewed migration plan.
It compiles and validates the shared Draft 2020-12 corpus, preserves YAML number
precision through validation, denies external references through an
instrumented in-memory loader, and passes Native, Node WASM, and real Chromium
Worker runs on the existing Go 1.23 baseline. Its structured validation errors
also expose stable schema and instance locations across repeated runs.

The decision remains conditional because the spike does not establish all
production acceptance conditions:

1. The selected `regexp2` ECMAScript/Unicode mode passes all required official
   suite cases and the regex samples from current contracts. Six Optional
   cases across two schema groups still fail to compile due to unsupported
   ECMAScript regex features, and 131 Optional cases fail format/assertion
   validation. This is not a claim of complete ECMA-262 compatibility. Keep the current contract
   pattern subset explicit and re-evaluate if it expands.
2. The library supplies JSON Schema and instance paths, but the spike's YAML
   conversion does not retain a source map back to YAML line and column. The
   production adapter must preserve that mapping before integration.
3. Optional format/assertion behavior remains incomplete: 131 optional
   validation failures and 6 optional regex compilation failures are retained
   in the latest suite result. Current catalog `format` uses annotation
   behavior; any future assertion policy needs its own acceptance decision and
   tests.
4. The stripped spike executable is 6.70 MB WASM, with a 2.29 MB increase over
   the same-toolchain no-engine harness. There is no accepted production size
   budget yet, and this is not a full Compiler binary delta.
5. The error-shape test proves repeatable structured locations and keyword
   paths, but Compiler diagnostic-code mapping and a full dependency/security
   review remain migration work.

The Kaptinlin result is retained for comparison, not continued. Its recorded
official required suite has 17 failures, and its deleted nested module required
Go 1.27.0. No Go toolchain or production module version was upgraded for this
spike. Corvus was not evaluated.

## Candidate comparison

| Candidate | Toolchain / license | Recorded compatibility | Decision |
| --- | --- | --- | --- |
| [`santhosh-tekuri/jsonschema/v6`](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6) v6.0.3 | Go 1.23.0; Apache-2.0. Custom regex mode adds `regexp2` v1.11.0 (MIT). | 25/25 shared vectors; 8/8 exact denied refs; required suite 1,319 pass / 0 fail / 475 skipped with custom regex; Native, Node WASM, and 3 current Chromium runs pass. | **CONDITIONAL ADOPT** with the conditions above. |
| [`kaptinlin/jsonschema`](https://github.com/kaptinlin/jsonschema) v0.9.10 | Deleted nested module required Go 1.27.0; MIT. | Historical results: 25/25 shared vectors and 8/8 denied refs; required suite 1,302 pass / 17 fail / 475 skipped; optional suite 862 pass / 107 fail / 126 skipped. Implementation removed; historical runs are not reproducible from this branch. | Do not continue or select. |

San's custom regex path is the only configuration of these candidates that
passed the required official suite in the recorded corpus. The extra regex
engine is included in the measured footprint.

## Isolation and inputs

- All spike implementation and dependencies are under
  `tools/schema-engine-spike/`; the production `go.mod` and `go.sum`,
  `internal/compiler/**`, `BuildRequest`, `BuildResult`, CLI, WASM API, Skill
  sources, ToolCatalog, PR #5 branch, and `main` were not changed by the spike.
- The spike branch starts at the exact PR #5 checkpoint above. PR #6 remained a
  separate design branch at the reviewed commit above.
- Authored fixtures remain YAML. `yaml.v3` nodes are converted to JSON values;
  numbers become `json.Number`, not `float64`.
- The 12-operation approved catalog input is
  `examples/ca-agent-demo-project/lib/tools/createdsession.tool.yaml`, SHA-256
  `c99ffe9c0ccbf5e65da19f689b5af47580a2975c4734954c18af4805641d7f6f`.
  Tests compile its 12 Operations and all 6 `x-common` fragments (34 schema
  roots in total) with both San regex configurations.
- The official [JSON-Schema-Test-Suite](https://github.com/json-schema-org/JSON-Schema-Test-Suite)
  is pinned at commit
  `38dea932b02986f496345bf007108277777630bf`. The lock and fetch script are in
  `tools/schema-engine-spike/fixtures/JSON-Schema-Test-Suite.lock` and
  `tools/schema-engine-spike/scripts/fetch-json-schema-test-suite.sh`.
- San module: Go `1.23.0`, library `v6.0.3`, optional custom regex dependency
  `github.com/dlclark/regexp2@v1.11.0`. License files reviewed for the engine
  (Apache-2.0), regexp2 (MIT), and `golang.org/x/text@v0.14.0` (MIT). This is a
  license/dependency inventory, not a vulnerability scan. The linked Go module
  set from `go list -deps -buildvcs=false` is San, regexp2, x/text, yaml.v3,
  and the two local spike modules.
- Kaptinlin historical candidate: `v0.9.10`, `go 1.27.0`, license MIT. Its
  nested module, tests, and dependency files were removed from the Spike branch;
  the Compiler toolchain was not changed.
- Runtime environment for the final San measurements: Linux amd64, Go 1.23.0,
  Node `v24.22.1`, Chromium `156.0.8078.79`. Browser runs used a new DevTools
  target in the already-running Chromium process and closed only that target.

## WP1 — shared Draft 2020-12 semantics

The shared YAML fixture contains 25 cases. San passed **25/25** in Native,
Node WASM, and each of the three Chromium Worker repetitions. The fixture
covers:

- Boolean `true` and `false` schemas.
- `$defs`, local and cross-document `$ref`, `$ref` sibling behavior, nested
  references, relative `$id` resolution, and recursive references.
- `allOf`, `anyOf`, `oneOf`, `not`, conditional `if`/`then`/`else`,
  `additionalProperties`, `prefixItems`, and `items`.
- `const`, `enum`, exact numeric constraints, format annotation, explicit
  format assertion, and unknown formats remaining annotations.

The exact fixture is `tools/schema-engine-spike/fixtures/core-vectors.yaml`.
The same YAML-to-JSON conversion and fixture corpus are used for both candidate
adapters. The secondary candidate's existing Native and Node WASM run also
passed 25/25; its prior Chromium smoke passed the same shared vectors.

## WP2 — YAML precision and JSON boundary

San passed the shared exact-number cases through YAML AST conversion and engine
validation:

| Case | Evidence |
| --- | --- |
| Integer above 2^53 | `9007199254740993` accepted as the exact `const`; adjacent `9007199254740992` rejected |
| `uint64` maximum | `18446744073709551615` preserved and matched exactly |
| Large negative integer | `-9223372036854775809` preserved and distinguished from its adjacent value |
| Decimals | `0.1` and `0.3` preserved; `multipleOf: 0.1` accepts `0.3` and rejects `0.31` |
| Scientific notation | `1e-7` and `3e-7` preserved; exact `multipleOf` behavior verified |
| Exclusive bounds | Bounds above 2^53 distinguish the interior integer from both endpoints |

`yamljson_test.go` also verifies exact `json.Number` output, YAML integer-base
normalization, acyclic aliases, and rejection of duplicate keys, non-string
keys, timestamps, custom tags, NaN, infinity, and alias cycles. The same
converter feeds the candidate engine; `encoding/json` marshaling retains
`json.Number` lexemes. No lossy `float64` conversion is used in the fixture
path.

## WP3 — reference security

San passed **8/8** denied-reference cases in Native, Node WASM, and each of
three Chromium Worker runs. The spike asserts each canonical resolved URI is
present in both the registry lookup and a `SPIKE_REGISTRY_DENIED` record, and
that the compile error contains that code and exact URI. A generic compile
failure or an unrelated registry lookup cannot satisfy the test.

| Attempt | Result |
| --- | --- |
| HTTP / HTTPS remote references | Exact resolved URI denied; separate loopback HTTP and HTTPS servers each receive zero requests |
| Absolute filesystem path / `file:` URI | Denied by registry; a valid JSON schema on disk is used as a file-loader canary |
| Path traversal | `../../outside.json` resolves to the exact escaped URI and is denied; no filesystem loader is installed |
| Relative reference | A positive `../shared/child.json` resolves to the exact in-memory URI and validates with that schema |
| Unknown and custom schemes | Exact URI denied by the registry |
| Missing in-memory resource | Exact URI denied with the registry miss code |

The successful semantic vectors separately exercise local `$defs`,
cross-document references, nesting, `$id` base resolution, and recursive
references. Each attempted resource lookup is recorded by the registry.
The HTTPS check uses a live TLS loopback server with the same zero-request
assertion. The file canary contains a syntactically valid schema, so successful
compilation or an error without the registry denial marker fails the test.
These checks verify the loader boundary itself and do not infer safety from an
arbitrary compile failure.

## WP4 — regular expressions

Eight focused cases distinguish the default Go RE2 path from the custom path:
Unicode literals, Unicode property escapes and aliases, astral Unicode, look
ahead, lookbehind, backreferences, and lookaround under `patternProperties`.

| San configuration | Focused samples conforming | Official required failures |
| --- | ---: | ---: |
| Default RE2 | 3/8 | 9, all `pattern.json` / `patternProperties.json` Unicode property cases |
| `regexp2.ECMAScript | regexp2.Unicode` | 8/8 | 0 |

The custom engine applies only two explicit normalizations:
`\p{Letter}` → `\p{L}` and `\P{Letter}` → `\P{L}`. This handles the suite's
long Unicode category alias while leaving other patterns unchanged; it is not
a general regex translator.

The current checked contract patterns are simpler ASCII expressions, including
the catalog digest and response-artifact patterns, and the pinned PDU Skill's
DNN/MCC/MNC patterns. A San test validates positive and negative examples for
those patterns under both default RE2 and the custom engine. This evidence is
limited to those patterns; it does not establish arbitrary ECMA-262 behavior.

## WP5 — Native, Node WASM, and Chromium Worker

San Native and `GOOS=js GOARCH=wasm` builds and runs passed. Native timings are
three fresh-process runs, Node WASM timings use five fresh instances in one
Node process, and Chromium measurements use three browser runs. Compile and
validation samples come from the 25 shared vectors; validation timings are per
vector batch, which may contain multiple instances.

| Metric | Native Go 1.23.0 | Node WASM Go 1.23.0 | Chromium Worker (3 runs) |
| --- | ---: | ---: | ---: |
| First schema compile, median (range) | 0.039 ms (0.036–0.078) | 3.992 ms (single smoke sample) | Included in `goRun` |
| Later per-schema compile, median (range) | 0.064 ms (0.019–0.340) | 0.345 ms (0.131–7.363) | Included in `goRun` |
| Validation batch, median (range) | 0.010 ms (0.001–0.224) | 0.057 ms (0.012–1.224) | Included in `goRun` |
| Full harness, median (range) | 6.699 ms (6.330–8.042) | 29.705 ms (23.061–70.458) | `goRun` 318.4 ms (264.7–344.1) |
| WASM instantiate or fetch + instantiate, median (range) | n/a | 23.193 ms (19.673–50.502) | 136.2 ms (99.6–162.2) |
| Whole browser worker elapsed, median (range) | n/a | n/a | 459.6 ms (447.4–509.3) |
| Peak memory | 8,084 KiB process RSS | WASM linear memory 19,595,264 → 28,311,552 bytes | WASM linear memory 19,595,264 → 28,311,552 bytes |

Native performance aggregates three fresh processes. The Node timing measured
five fresh WASM instances in one Node process; its `go.run` median was 82.313 ms
(57.030–174.353), including the slower first runtime invocation. Node process
RSS deltas varied with GC and are not used as a memory claim. All three current
Chromium runs passed 25 semantic vectors and 8 security cases in HeadlessChrome
156.0.8078.79. The worker reports fetch plus instantiation separately from Go
execution. The runtime glue matched Go 1.23.0 `wasm_exec.js` SHA-256
`45ce9dfe7211247544ab6f4268eb8cb5b6f3d5ae602dc3b51447b7eada99c229`.

The size comparison uses a same-toolchain no-engine **spike harness** under
`cmd/skeleton`, with the same shared runner but a stub validator. These are
stripped (`-trimpath -ldflags='-s -w'`) executable sizes; the delta includes
the engine, adapter, and custom regex path. It is not a production Compiler
size estimate.

| Artifact | Size | No-engine harness | Delta |
| --- | ---: | ---: | ---: |
| Native San executable | 3,756,184 bytes | 2,527,384 bytes | +1,228,800 bytes (+48.6%) |
| San WASM executable | 6,701,949 bytes | 4,407,571 bytes | +2,294,378 bytes (+52.0%) |

## WP6 — official Draft 2020-12 suite

### Compatibility semantics and accounting

`compatibleRelease` follows the pinned `validation/README.md` pseudocode. An
unqualified release is a lower bound, `<=` an upper bound, and `=` an exact
release. Comma-separated, ordered constraints describe disjoint ranges (for
example, `=3,7,<=2020` means draft-03 plus draft-07 through 2020-12); evaluation
stops at the first upper boundary containing the release under test. Empty
metadata is universal. Unknown or malformed release tokens are rejected.
Drafts 2027/2028 and placeholder 9999 are not selected for the 2020 run.

`suite_review_test.go` covers the requested boundaries (`7`, `2019`, `<=2019`,
`2020`, `=2020`, `6,<=2019`, `9999`, unknown versions), disjunctions and their
boundaries (`=3,7,<=2020`, `<=3,7`), and future releases. Its fake candidate
exercises compile and validation failures in both classes, tagged Optional
groups, skipped groups, and incompatible-group no-compile behavior. It asserts
that each class's `total = passed + failed + skipped` and `failed = compile
failures + validation failures`.

The runner used the pinned suite's `validation/tests` corpus (86 JSON group
files), compatibility metadata, and `UseNumber`. It classified
`format-assertion` groups, explicitly optional groups, and
`tests/optional/` as Optional; incompatible groups were counted as Skipped;
proposals were excluded. Of 689 total groups, 492 were compatible and 197
were skipped by the Draft 2020-12 compatibility filter.
Across 2,889 cases, 2,288 were evaluated and 601 skipped; the class totals and
group totals reconcile in both regex configurations.

### San v6.0.3

| Regex mode | Class | Total | Passed | Failed | Skipped | Compile failures | Validation failures |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Default RE2 | Required | 1,794 | 1,310 | 9 | 475 | 9 | 0 |
| Default RE2 | Optional | 1,095 | 805 | 164 | 126 | 18 | 146 |
| `regexp2` custom | Required | 1,794 | 1,319 | 0 | 475 | 0 | 0 |
| `regexp2` custom | Optional | 1,095 | 832 | 137 | 126 | 6 | 131 |

With the custom regex engine, 6 Optional cases fail schema compilation for
unsupported ECMAScript regex constructs. The other 131 are validation
mismatches: 126 built-in format cases and 5 optional regex/format-assertion
cases, including duration, email, hostname/IDN, URI/IRI, and URI-template
behavior. Draft 2020-12 defaults `format` to annotation; assertion-tagged tests
are deliberately reported separately as Optional. All class totals reconcile:
`passed + failed + skipped = total`, and compile plus validation failures equal
failed cases. The required class has no failures in the custom regex run.

### Kaptinlin v0.9.10 — retained prior result, not rerun

| Class | Total | Passed | Failed | Skipped |
| --- | ---: | ---: | ---: | ---: |
| Required | 1,794 | 1,302 | 17 | 475 |
| Optional | 1,095 | 862 | 107 | 126 |

Its 17 required failures were 9 `content.json` cases, 1 `dynamicRef.json`, 6
empty-`enum` cases, and 1 `vocabulary.json` case. This result and the earlier
Native/Node WASM/Chromium smoke evidence are retained from the previous
candidate run. No Kaptinlin validation was started after the instruction to
continue only San.

## Error structure and mapping

The San adapter returns the library's `*jsonschema.ValidationError`, whose
tree exposes `SchemaURL`, `InstanceLocation`, `ErrorKind.KeywordPath()`, and
nested causes. A new test compiles and validates the same invalid instance three
times and compares a canonical snapshot of those fields; the schema URI,
instance path, keyword path, and nested cause structure remained identical.
This makes deterministic Compiler diagnostic mapping feasible. The spike does
not yet assign Compiler diagnostic codes or map JSON Pointer locations back to
YAML source line and column.

## Reproduction commands

Run from the repository root. The nested modules use the existing Go 1.23.0
toolchain; no Compiler toolchain upgrade is required.

```sh
# Fetch the pinned official suite (the script prints the suite root).
tools/schema-engine-spike/scripts/fetch-json-schema-test-suite.sh /tmp/JSON-Schema-Test-Suite-spike

# Unit and adapter tests.
GOTOOLCHAIN=go1.23.0 GOWORK=off go -C tools/schema-engine-spike test ./... -count=1
GOTOOLCHAIN=go1.23.0 GOWORK=off go -C tools/schema-engine-spike/cmd/san test ./... -count=1

# Three Native timing / RSS repetitions; writes JSON and /usr/bin/time output.
tools/schema-engine-spike/scripts/measure-native.sh /tmp/schema-engine-spike-native-measurements

# Build the Native executable used by the official suite runner.
GOTOOLCHAIN=go1.23.0 GOWORK=off go -C tools/schema-engine-spike/cmd/san build -buildvcs=false -trimpath -ldflags='-s -w' -o /tmp/schema-spike-san.native

# Build and run WASM under Node with the matching Go 1.23.0 runtime glue.
GOTOOLCHAIN=go1.23.0 GOWORK=off GOOS=js GOARCH=wasm go -C tools/schema-engine-spike/cmd/san build -buildvcs=false -trimpath -ldflags='-s -w' -o /tmp/schema-spike-san.wasm
node tools/schema-engine-spike/scripts/run-wasm-node.cjs /tmp/wasm_exec.js /tmp/schema-spike-san.wasm

# Build the same-toolchain no-engine size baselines.
GOTOOLCHAIN=go1.23.0 GOWORK=off go -C tools/schema-engine-spike/cmd/skeleton build -buildvcs=false -trimpath -ldflags='-s -w' -o /tmp/schema-spike-baseline.native
GOTOOLCHAIN=go1.23.0 GOWORK=off GOOS=js GOARCH=wasm go -C tools/schema-engine-spike/cmd/skeleton build -buildvcs=false -trimpath -ldflags='-s -w' -o /tmp/schema-spike-baseline.wasm

# Full pinned suite, default RE2 and custom regexp2 runs.
SPIKE_SUITE_ROOT=/tmp/JSON-Schema-Test-Suite-spike SPIKE_SUITE_REGEX2=1 /tmp/schema-spike-san.native

# Stage the browser assets, then repeat the actual Chromium Worker smoke three times.
cp tools/schema-engine-spike/web/index.html /tmp/schema-spike-web/index.html
cp tools/schema-engine-spike/web/worker.js /tmp/schema-spike-web/worker.js
cp /tmp/schema-spike-san.wasm /tmp/schema-spike-web/san.wasm
cp /tmp/wasm_exec.js /tmp/schema-spike-web/wasm_exec_san.js
for run in 1 2 3; do SPIKE_WEB_ROOT=/tmp/schema-spike-web node tools/schema-engine-spike/scripts/chromium-smoke.cjs; done
```

`/tmp/wasm_exec.js` in these commands is Go 1.23.0's official
`misc/wasm/wasm_exec.js` from
`https://raw.githubusercontent.com/golang/go/go1.23.0/misc/wasm/wasm_exec.js`.
The Chromium harness defaults to the San candidate; Kaptinlin is not rerun by
the acceptance commands above.

The candidate/API references are the upstream
[San repository](https://github.com/santhosh-tekuri/jsonschema), its
[v6 package documentation](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6),
the [Kaptinlin repository](https://github.com/kaptinlin/jsonschema), and the
[JSON Schema Draft 2020-12 specification](https://json-schema.org/draft/2020-12).

## Stop condition

The requested spike report and independent reproducible evidence are complete.
No production Compiler change, production dependency, PR #5 edit, PR merge, or
migration was made. Stop after opening the independent Spike PR and wait for
review.
