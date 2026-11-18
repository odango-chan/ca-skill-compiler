> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP11 Native / WASM / Chromium Worker parity (v1alpha4)

**Status:** Chromium Worker parity harness implemented and locally executed — **PASS** for the checked fixture projection.
This is Worker/transport parity evidence, not a claim that the whole Compiler or Studio integration is qualified.

## Harness

`scripts/chromium-parity.mjs`:

1. builds the CGo-free Go/WASM compiler (`GOOS=js GOARCH=wasm`) into `.parity-web/skillc.wasm` and copies the pinned toolchain `wasm_exec.js`;
2. builds a `BuildRequest` from `web/parity-project` (base64 file bytes, frozen v1alpha4 wire shape);
3. computes the Native Go reference with `cmd/parity-replay` (`Check`, `Build`, `InspectDependencies`);
4. serves the page over an ephemeral localhost port, launches **real headless Chromium** (`--headless=new`) with the DevTools protocol;
5. runs the page, which loads `web/skillc-worker.js` (the same Worker transport Skill Studio uses) in a real Web Worker, waiting for the Go/WASM callbacks;
6. compares Native vs browser projections for `check`, `build.success`, `build.diagnostics`, `build.dependencyClosure`, `build.outputFiles[].path` and `inspect`.

Exit code 0 only if every projection is byte-identical after stable key ordering.

## Reproduce

```text
GOTOOLCHAIN=go1.27.4 node scripts/chromium-parity.mjs
```

Local run (Windows, Chrome/158.0.8127.37):

```text
ok: check projection identical
ok: build.success projection identical
ok: build.diagnostics projection identical
ok: build.dependencyClosure projection identical
ok: build.outputFiles.paths projection identical
ok: inspect projection identical
PASS: Native and real Chromium Worker projections are identical
```

Set `CHROME_PATH` when Chrome is installed elsewhere. The harness is not part of `go test` because it requires a real browser.

## Transport fix included

`web/skillc-worker.js` previously waited for and called `skillcInspectLibraries`, which the Go/WASM entry point no longer exposes (`skillcInspectDependencies` is the v1alpha4 callback). The Worker now waits for and calls the correct callback, so the browser transport no longer hangs or silently misroutes `inspect`.

## Honest limits

- Parity is established for the checked fixture (a v1alpha4 Skill whose validation currently fails closed on missing dependency snapshots). It proves the browser executes the same Core and produces identical diagnostics/results for the same request; it does not yet cover a successful Build with Artifact bytes.
- Artifact output byte parity in the browser becomes testable once a successful Build path exists (WP10 wiring).
- `cmd/parity-replay` is tooling only; no Compiler core semantics were added for it.


## Successful-Build byte parity (added)

The harness now also runs a second phase against web/parity-success/ (fixture Skill + real snapshot files + capability bundle + 	rusted.json bindings). cmd/parity-replay builds the trusted BuildRequest, computes snapshot digests, and emits the exact request JSON; the browser runs Build through the Worker and every generated artifact file byte is compared by SHA-256 against Native Go:

`	ext
ok: success.build.outputFiles.paths projection identical
ok: successful-Build artifact byte parity (5 files)
ok: studio-success.artifactPaths projection identical
`
