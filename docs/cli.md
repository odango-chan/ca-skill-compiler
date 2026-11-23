# skillc CLI — v2.1.0

[中文](cli.zh-CN.md) · [Documentation](README.md) · [Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0)

**Product:** v2.1.0 (RELEASED). **Only language target:** `connectionagent.io/v1alpha5`. The CLI is a thin adapter over the single Go Compiler Core and never performs network acquisition, tool execution, package publishing or Host admission.

## Commands and contract

| Command | Input | Result |
|---|---|---|
| `skillc check PROJECT` or `--request F` | Frozen v1alpha5 BuildRequest / project loader | Static diagnostics, no artifact |
| `skillc inspect PROJECT` or `--request F` | Frozen v1alpha5 BuildRequest | Read-only DependencyInspection |
| `skillc build PROJECT --out DIR` or `--request F --out DIR` | Frozen v1alpha5 BuildRequest | Deterministic frozen Artifact; automatically verified |
| `skillc build-development --request F --out DIR` | Development v1alpha1 BuildRequest | Deterministic `NOT_ADMITTED` Artifact |
| `skillc verify-development --request F --result R` | Same Development request + complete BuildResult bytes | Strict rebuild-and-byte comparison |
| `skillc version` / `skillc help` | None | Version/API details / CLI help |

The Development commands accept only `--request` plus their respective `--out` or `--result` (and optional `--json`), no positional project path and no `--skill`. The normal `check`/`inspect`/`build` path accepts repeated `--skill NAME` selectors, and `--json` output; `--skill` cannot override explicit nonempty request `targets`.

**Exit codes:** `0` success; `1` compilation, artifact or verification failure; `2` malformed input/usage/I/O error.

## Project input (frozen v1alpha5 path)

```text
PROJECT/
  skills/...                     or files/skills/...
  snapshots/<dependencyId>/...   immutable dependency byte sets
  trusted.json                   local bindings / Owner / capability / evidence config
```

`trusted.json` is a local convenience description, not a public wire contract. The project loader computes the dependency snapshot digests from bytes; no missing Owner/binding/digest can be inferred. Use `--request` for machine clients.

```bash
skillc check web/parity-success
skillc inspect web/parity-success --json
skillc build web/parity-success --out ./dist/frozen-artifact
skillc build --request request.json --out ./dist/frozen-from-json --json
```

Frozen request contract: [BuildRequest v1alpha5](../contracts/v1alpha5/build-request.schema.json). Successful Artifact is self-verified before writing; nonempty/unsafe/symlinked destinations fail closed. The frozen output's `generator.version=v2.0.2` remains unchanged in v2.1.0 by contract.

## Development v1alpha1 input and output

`build-development` receives exactly the CA-SL `files[]`, `targets[]`, exact immutable Interface Package `snapshots[]`, and a `lock` with declared package dependencies. It **does not** accept trusted Owner attestations, live Evidence, Provider data or deployment credentials. Omitted required fields, unknown JSON keys and broken closure are rejected.

```bash
mkdir -p ./dist
skillc build-development --request docs/design/fixtures/registration-development-request.json --out ./dist/development --json > ./dist/development-result.json
skillc verify-development --request docs/design/fixtures/registration-development-request.json --result ./dist/development-result.json --json
```

The complete result JSON from `build-development --json` (including Base64 Artifact file bytes) is required for `verify-development`; a standalone manifest is not enough. Output files include:

```text
manifest.json
host-admission-requirements.json
runtime-execution-requirements.json
skills/<skill>/compiled/procedure.json
skills/<skill>/compiled/ingress-links.json
```

The manifest contains `capabilityRequirements[]` and `executionStatus=NOT_ADMITTED`. A successful `BUILT` Artifact is **never a deployment authorization or Host admission decision**. All five published Development JSON Schema contracts: [Development v1alpha1](../contracts/development/v1alpha1/README.md).

## Build and CI

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
```

The Chromium command needs Node and a real Chrome/Chromium browser; CI installs the browser. v1alpha4 historical CLI docs are [archived here](cli-v1alpha4.md) and do **not** describe the v2.1.0 CLI.
