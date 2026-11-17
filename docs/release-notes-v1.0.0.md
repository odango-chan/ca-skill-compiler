# CA Skill Compiler v1.0.0

**Product version:** CA Skill Compiler **v1.0.0**
**Language target:** **CA-SL v1alpha4**, frozen at
`odango-chan/ca-skills@7f84393ee8c395e4d8c39a74cb5000a1f896588d`
(spec blob `b70064c5737b8ee2b3d1435fe496357f44b42773`)
**Artifact generator version:** `v1.0.0` (recorded in the JCS artifact manifest)

The Compiler product version and the CA-SL language version are independent. This
release does not change the frozen v1alpha4 syntax, Compiler-owned contracts, diagnostic
identities or language semantics. It is a formal delivery of the previously verified
v1alpha4 implementation and its reference CLI.

## Features

- **CA-SL v1alpha4 Compiler** — deterministic, stateless Go Core shared by Native Go and
  Go/WASM; no network access, dependency acquisition, Python execution or Runtime dispatch.
- **CLI v1** — `skillc check` / `build` / `inspect` / `version`, project-directory and
  strict BuildRequest JSON (`--request`) input, human-readable diagnostics with
  `path:line:column` and frozen code, `--json` machine output matching the frozen
  Check/BuildResult/DependencyInspection shapes, documented exit codes, fail-closed
  dependency handling.
- **BuildRequest and dependency snapshot digest** — strict JSON envelope, limits, exact
  `sha256:` snapshot digests over raw bytes.
- **YAML AST and exact source positions** — one normalized goccy/go-yaml AST; duplicate
  keys, aliases and structural limits enforced; diagnostics retain original positions.
- **Dependency identity, Owner and Capability linking** — trusted snapshot bindings,
  Stable Package ID checks, export-owner attribution, CapabilityContracts membership,
  Effect and Grant checks; DependencyInspection.
- **Schema Registry / closure / assignability** — G.4 allowlist, exact `$id` closure,
  San adapter, F.5 conservative assignability, CEL subset and Predicate signatures.
- **Procedure graph static analysis** — reachability, edge-exact global Precondition
  dominance, route-key eligibility, Mutation safe-terminal rules, retry/reconcile static
  contracts, reachability-exact final-Postcondition classification.
- **Python CST / TypedDict / Script ABI** — gositter grammar only (never executes Python),
  typed input derivation from the call site, strict TypedDict output, NotRequired guard
  proof, restricted key access.
- **Evidence / Mutation / Reconcile static analysis** — evidence requirement binding,
  independence groups, static Evidence satisfiability, trusted Mutation/Reconcile Schema
  semantics; unfrozen external formats fail closed.
- **Deterministic Artifact / JCS Manifest** — byte-deterministic tree, canonical JCS
  manifest, generated Script Output Schemas, structural verification before write.
- **Native / WASM build** — CGo-free Native and Go/WASM builds from the same Core.
- **Browser Worker and Skill Studio read-only integration** — real headless-Chromium
  parity for check/build/inspect, artifact bytes and read-only Studio presentation.

## Verification

Local acceptance executed on the release commit on Windows (`go1.26.6 windows/amd64`,
node v24.18.0, system Google Chrome headless). GitHub Actions is unavailable for this
account (billing); the complete local gate below is the formal acceptance evidence,
marked `LOCAL_ACCEPTANCE_PASS`. CI is **not** claimed as passing.

| Command | Result |
|---|---|
| `GOWORK=off go test ./... -count=1` | **PASS** (exit 0) |
| `GOWORK=off go vet ./...` | **PASS** (exit 0) |
| `CGO_ENABLED=0 GOWORK=off go build ./...` | **PASS** (exit 0) |
| `CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm` | **PASS** (exit 0) |
| `git diff --check` | **PASS** (exit 0) |
| `GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v` | **PASS** (9 CLI end-to-end tests, real binary) |
| `CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v` | **PASS** (real headless Chromium; check/build/inspect and Studio projections identical, in-Worker artifact verification, successful-Build 5-file SHA-256 byte parity) |

E-coverage (honest, `tests/conformance/v1alpha4/e-coverage.json`):

- 71 `PARTIAL_STATIC_EVIDENCE` — executed static evidence for sub-branches, **not**
  whole-case PASS.
- 8 `BLOCKED_EXTERNAL_CONTRACT` — E38/E46/E68/E73/E87/E101/E102/E109 depend on unfrozen
  external publications.
- 2 `BLOCKED_NORMATIVE_MAPPING` — E71/E74 await language-owner clarification.
- 39 `NOT_APPLICABLE_TO_COMPILER` — Host/Runtime only.
- **0 whole-case PASS claimed.**

## Known Limitations

- External Evidence Descriptor, deployment Scope algebra and identity-input publication
  formats are not frozen; those positive branches continue to **fail closed**. No format
  was invented.
- E71/E74 remain recorded against the language-owner clarification
  ([ca-skills #59](https://github.com/odango-chan/ca-skills/issues/59)); no diagnostic was
  guessed.
- WP13 real 5GC / MAS Qualification is **NOT_QUALIFIED**. No real Owner/Deployment
  publications or runnable MAS baseline were available; synthetic fixtures are not
  product conformance evidence.
- External Owner integration requires later real published artifacts and is not claimed
  by this release.
- GitHub Actions is unavailable; this release uses the local acceptance gate above.
- Pre-v1alpha4 analysis files remain isolated behind the `legacy_v01` build tag; they are
  not reachable from public APIs and are retained only as repository history.
- The non-Windows release binaries are cross-compiled with `CGO_ENABLED=0`; only the
  Windows binary was executed locally, so no runtime test is claimed for other platforms.
  The WASM module was executed in real Chromium.

A successful build is a static compilation result, not a Runtime authorization token.
Compiler product conformance and product qualification are distinct: this release claims
Compiler implementation + local verification, not ConnectionAgent product certification.

## Reproduce

```bash
git clone https://github.com/odango-chan/ca-skill-compiler.git
cd ca-skill-compiler
git checkout v1.0.0

GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o dist/skillc.wasm ./cmd/skillc-wasm
CGO_ENABLED=0 go build -o dist/skillc ./cmd/skillc

./dist/skillc version
./dist/skillc check web/parity-success
./dist/skillc inspect web/parity-success
./dist/skillc build web/parity-success --out /tmp/skillc-out

CHROMIUM_PARITY=1 go test ./internal/parity -run TestChromiumWorkerParity
```
