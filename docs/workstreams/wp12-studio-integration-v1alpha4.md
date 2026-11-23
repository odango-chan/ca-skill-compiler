> **ARCHIVED IMPLEMENTATION RECORD — not a current task authorization (v2.1.0).** Original GOAL/PR/branch/conformance wording is kept for traceability; the current product is [Compiler v2.1.0](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0), supports only CA-SL v1alpha5, and has a separate Development v1alpha1 profile. Current state: [WORK](../WORK.md) · [historical index](README.md). <!-- historical-docs-v2.1.0 -->

---

> **HISTORICAL RECORD.** This document describes a past implementation, release or planning state (CA-SL v1alpha4 or an earlier v1alpha5 work package). It is retained for traceability and is not the current Compiler behaviour. Current status: [v1alpha5 single-version acceptance](../COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md).

# WP12 Skill Studio integration (v1alpha4)

**Status:** Minimal read-only Studio integration surface implemented and exercised in real headless Chromium — **PASS** for the checked fixture.
This is transport/UI-consumption evidence, not a claim of a full Studio product acceptance or that the Compiler is qualified.

## What is implemented

`web/studio.html` + the existing `web/skillc-worker.js` transport:

- consumes the **same unified Compiler Worker result** as WP11 (`check` / `build` / `inspect`); it does not embed a YAML parser, JSON Schema validator or procedure linker;
- displays every diagnostic with `code`, `phase`, `path`, `line`, `column`, `message`, using data attributes that a real Studio can map to source positions;
- presents the Build artifact as a **read-only** file list with a read-only content view (`aria-readonly="true"`); no editing or regeneration path;
- shows the compiler status line honestly (`Build success`, diagnostic count, inspect counts, `NOT_QUALIFIED`).

## Acceptance

Extends `scripts/chromium-parity.mjs`, which now duplicates the Native/Worker parity checks and then opens `studio.html` in real headless Chromium:

```text
ok: studio.diagnostics projection identical
ok: studio.buildSuccess projection identical
ok: studio.artifactPaths projection identical
ok: studio diagnostics/artifact read-only integration identical
PASS: Native and real Chromium Worker projections are identical
```

Assertions: the rendered diagnostic rows equal the Native `build.diagnostics` projection (code/phase/path/line/column), the artifact path list equals the Native Build output paths, the artifact view is read-only, and the page declares no reimplemented parser.

Reproduce: `GOTOOLCHAIN=go1.27.4 node scripts/chromium-parity.mjs`, or the Go entry point
`CHROMIUM_PARITY=1 GOTOOLCHAIN=go1.27.4 go test ./internal/parity -run TestChromiumWorkerParity -v`.

## Honest limits

- The fixture currently exercises the fail-closed Build path; once a successful Build path exists, the same page will list real generated schemas and the manifest without UI changes.
- This is an integration harness proving Studio *consumes* Compiler results; a full Studio product (editor, project management, ZIP export) is out of this repository's Compiler scope.
- No credentials, filesystem handles or executable code cross the Worker boundary.

## Successful-Build presentation (added)

The Chromium acceptance also opens studio.html?request=parity-success-request.json: the page lists the real generated artifact files (manifest + author sources) with the read-only view, and studio-success.buildSuccess/rtifactPaths are asserted against the Native Build result.
