# CA Skill Compiler v2.1.0 — official release record

**Product:** `v2.1.0`  
**Language:** `connectionagent.io/v1alpha5`  
**Source:** `b67fbb56c34d15b8d5a0903c8a7ca93d608f8f65`  
**Status:** FROZEN / RELEASED (2026-11-23)  
**Release:** https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0  
**Evidence:** https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980  
**Release gate:** [#55](https://github.com/odango-chan/ca-skill-compiler/issues/55)

## Changes

v2.1.0 introduces a separate Development Build API (`connectionagent.io/compiler-development-build-request/v1alpha1` and matching result/manifest, Interface Package and Dependency Lock) with the single Go Compiler Core.

The Development output is deterministic, statically linked, and always `NOT_ADMITTED`. It contains an explicit selected Capability requirement list, complete locked interface identity, and separate Host Admission / Runtime Execution requirements. Build/Verify never grant deployment execution or claim Owner/Host trust.

The five Development v1alpha1 machine schemas are under `contracts/development/v1alpha1/`. Their exact bytes are FROZEN with v2.1.0 at the source SHA above. Schema/DTO parity and negative wire-case tests run in the product gate.

The v2.0.2-frozen v1alpha5 Build/Verify API and artifact bytes are retained. Their embedded generator identity stays `v2.0.2` as an artifact contract constant, while Development manifests use `v2.1.0`. A consumer must not infer the installed CLI binary version from the frozen v1alpha5 artifact's generator field.

## Product release acceptance

Gate on the exact source SHA using `go test ./... -count=1`, `go vet ./...`, CGO-free native and wasm builds, CLI E2E, v2.0.2 frozen golden baseline, Development contract/negative/Verify tests, real Chromium parity (including Development artifact byte equality, input reordering and rejection diagnostics), and `git diff --check`. A successful job must upload and re-download all release assets and verify `SHA256SUMS`.

Published assets: four native binaries, standalone Go/WASM, a runnable static web Worker bundle with `skillc.wasm`, `skillc-worker.js` and matching `wasm_exec.js`, a Development v1alpha1 contract archive and `SHA256SUMS`.

This does not qualify an Owner package, Deployment Grant, Provider, Host, Runtime, 5GC E2E, or MAS Procedure. Real MAS author-source offline checks remain independent in #53.

**Release VERIFIED:** the exact-source workflow passed, created the annotated `v2.1.0` tag, published all eight expected assets, then independently downloaded them and verified all SHA256SUMS. The tag resolves to `b67fbb56c34d15b8d5a0903c8a7ca93d608f8f65`.

This record is a product-delivery assertion only; DEVELOPMENT is still `NOT_ADMITTED`.
