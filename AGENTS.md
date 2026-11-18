# CA Skill Compiler — Repository Instructions

- **Current product release:** CA Skill Compiler v1.0.0
- **Active language target:** released CA-SL v1alpha5 (`connectionagent.io/v1alpha5`), which inherits the released v1alpha4 normative semantics
- **Published product release commit (immutable):** `252fd6c6dab15fb92806426aeb19d49a36b08d5e`

## 1. Authority

Follow these sources in order:

1. The released CA-SL v1alpha5 single-file specification at [ca-skills@0abfd0e](https://github.com/odango-chan/ca-skills/blob/0abfd0e1ab99f481a4ae01f3f47efc54783dd5e5/docs/skill-language-spec.v1alpha5.zh-CN.md). It inherits the released v1alpha4 semantics.
2. The released IngressCatalog v1alpha1 logical contract and Ingress diagnostic registry at [ca-skills@9412be9](https://github.com/odango-chan/ca-skills/blob/9412be9894ef36acf846c757a5870dc0c48b33ac/docs/design/integration-ingress-operation-contract-vnext.zh-CN.md).
3. Compiler-owned v1alpha5 contracts in `docs/design/` and machine contracts in `contracts/v1alpha5/`; v1alpha4 contracts remain historical traceability only.
4. The conformance register, diagnostic index and acceptance records under `docs/workstreams/`, `tests/conformance/` and `docs/COMPILER-V1ALPHA5-INTEGRATION-ACCEPTANCE.md`.
5. Current implementation, tests and release/maintenance records.

Historical v0.1/v0.2 documents and fixtures are not current authority. Do not add compatibility, translation, version sniffing or fallback behavior.

## 2. Release and version boundary

- v1.0.0 is published and its tag/release assets are immutable.
- Product version and language version are independent.
- CA-SL v1alpha5 is the active language target; v1alpha4 is a frozen historical target with no new language semantics.
- v1.0.0 / v1alpha4 remains a historical release and is not rewritten; the active line implements the released v1alpha5 Wait/Ingress semantics through the unified pipeline.
- Keep Compiler implementation status, local verification and real Owner/product qualification distinct. The release has local acceptance evidence; real 5GC/MAS qualification remains NOT_QUALIFIED.

## 3. Compiler ownership and trust boundaries

Use one deterministic, stateless Go Core shared by Native and Go/WASM. Compiler consumes source bytes, immutable dependency snapshots and trusted build context. It does not own dependency acquisition, package management, Git/HTTP/MCP access, credentials, deployment authorization decisions, Runtime admission, Tool/Python execution, Qualification or live 5GC operations.

Use the approved pinned dependencies and the single normalized YAML AST. Do not add parallel YAML, Schema, CEL, Python or JCS implementations. Python is parsed statically and never executed.

Host/Runtime remains responsible for Invocation, event ingress, Wait lifecycle, dispatch, authorization revalidation, mutation settlement, retry admission, resource responsibility and Trace. Do not move those responsibilities into Compiler.

## 4. Build and fail-closed boundaries

Check and Build share the same static validation pipeline. Build emits a complete deterministic artifact only when validation succeeds. The earlier v1alpha4 feature-scoped fail-closed gates are resolved: Evidence Descriptor linking, Mutation/Reconcile and retry assurance are implemented by the unified v1alpha5 pipeline. See [v1alpha5 Known Limitations](docs/known-limitations-v1alpha5.md).

A successful Build is not a Runtime authorization token or product qualification.

## 5. Code and documentation hygiene

- Keep implementation files separated by responsibility; avoid broad file consolidation.
- Keep source locations and diagnostic identities aligned with the frozen Appendix H.
- Keep tests beside the behavior they prove. Positive Build tests must exercise the public Build path, not only helper functions.
- Preserve historical specifications, release notes, migration inputs and test fixtures when useful for traceability; label them as historical rather than presenting them as current examples.
- Do not edit the v1.0.0 Git tag or published release assets.
- Remove obsolete implementation code from the active tree when it is unreachable and superseded; Git history preserves it.

## 6. Change workflow

- Work on a dedicated branch from current `main`.
- Open a PR to `main`; keep it Draft while local verification has not been executed or review blockers remain.
- Never push directly to `main`, force-push, or merge without explicit approval.
- Do not claim tests, CI or Chromium acceptance passed unless actually executed and their exact results are recorded.
- GitHub Actions was unavailable for the v1.0.0 release run; local acceptance commands and results are recorded in [Local Verification](docs/local-verification-v1alpha4.md).

## 7. Reproducible local gates

From the repository root:

```bash
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
CGO_ENABLED=0 GOWORK=off go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm GOWORK=off go build -o dist/skillc.wasm ./cmd/skillc-wasm
GOWORK=off go test ./cmd/skillc -count=1 -run TestCLIEndToEnd -v
CHROMIUM_PARITY=1 GOWORK=off go test ./internal/parity -count=1 -run TestChromiumWorkerParity -v
git diff --check
```

Record the actual Go, Node and Chromium versions used. A skipped or unavailable gate is NOT_RUN/BLOCKED, never PASS.
