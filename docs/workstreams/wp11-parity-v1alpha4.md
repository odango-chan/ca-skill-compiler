# WP11 Native / WASM / Chromium Worker parity (v1alpha4)

**Status:** COMPLETED — merged via PR #24 and included in CA Skill Compiler v1.0.0; retained as a historical implementation record.

> **已完成 / COMPLETED — MERGED INTO v1.0.0.** 本文记录 WP 实施时的任务边界、测试和决策。对应实现已通过 [PR #24](https://github.com/odango-chan/ca-skill-compiler/pull/24) 合并并纳入 v1.0.0。原分支名、Draft 审查状态和停止条件是历史信息；本文保留为实现证据，不是待执行任务。当前状态见[最终验收记录](../COMPILER-V1ALPHA4-FINAL-ACCEPTANCE.md)和[已知限制](../known-limitations-v1alpha4.md)。

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
