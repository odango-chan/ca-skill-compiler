# Skill Compiler v0 status

Updated: 2026-11-11. **Scope frozen: breaking-only new Script/Procedure/Library contracts.**

| Item | Design | Implementation | Acceptance |
| --- | --- | --- | --- |
| New repo, project `skills/lib/output` and Studio ABI | ACCEPTED | bootstrap source present | NOT_RUN (real compiler integration) |
| Dual real ca-agent-demo Skills & original Catalog | PINNED | 80 original Git blobs copied byte-identically; MAS proposed lib copy | source-copy PASS; compiler NOT_RUN |
| Script Contract: external Producer input, strict own Output | ACCEPTED ([spec](design/script-contract-v0.md)) | NOT_IMPLEMENTED | ST-01..07 NOT_RUN |
| Procedure binding, nested step scope & diagnostics | ACCEPTED ([spec](design/procedure-link-v0.md)) | PARTIAL prototype generic scan | G2/G3 NOT_RUN |
| Library manifest + inspect API | ACCEPTED | Tool YAML importer PARTIAL, manifest NOT_IMPLEMENTED | G1 NOT_RUN |
| Go Core, Native CLI, source path safety | ACCEPTED | prototype PARTIAL | G0/G5 NOT_RUN |
| Native/WASM parity, Chromium Pages | ACCEPTED | WASM/Worker stub present | G6/G7 NOT_RUN |
| Raw real Skill v0 migration (no compatibility) | ACCEPTED | NOT_STARTED | G8 NOT_RUN |
| Consumer Host/Runtime adapter to new output | Out of Compiler repo | NOT_STARTED | NOT_RUN |

**Critical source facts**: existing Core misidentifies nested `procedure` references; current path validator rejects valid `lib/manifest.yaml`; existing Go tests/build have not been run in this session. The actual ca-agent-demo ToolCatalog has 12 Operations and MAS proposed has 16; no Library demonstrates Runtime READY. The full authoritative Host Context/Observation schema is not yet present and must not be invented.

**No old Script compatibility required**: the frozen ca-agent-demo source is *migration input*, not a legacy build PASS target. The positive v0 synthetic sample is `examples/script-contract-project` and the multi-Skill basic sample is `examples/minimal-project`.

Next: implement [WP0–WP7](workstreams/local-implementation-goal.md) and verify [G0–G9](workstreams/acceptance-matrix-v0.md) using real local logs. [Draft PR #2](https://github.com/odango-chan/ca-skill-compiler/pull/2) stays Draft; never claim build/Runtime PASS before executing tests.
