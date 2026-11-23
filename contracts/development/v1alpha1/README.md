# CA Skill Compiler Development v1alpha1 — FROZEN with v2.1.0

These five JSON Schema (Draft 2020-12) documents describe the Development Build
family introduced by CA Skill Compiler v2.1.0. Their code-defined identities
remain v1alpha1. These exact machine-contract bytes were released and frozen
with tag `v2.1.0` at source commit
`b67fbb56c34d15b8d5a0903c8a7ca93d608f8f65` (Release acceptance run
[#45100406980](https://github.com/odango-chan/ca-skill-compiler/actions/runs/45100406980)).
FROZEN describes the Compiler product contract only; it never implies Host
admission, Owner trust, or product qualification of external systems.

| Schema | Purpose |
| --- | --- |
| `build-request.schema.json` | CA-SL source, exact snapshots, lock, DEVELOPMENT profile |
| `build-result.schema.json` | Deterministic diagnostics, statuses, emitted bytes |
| `artifact-manifest.schema.json` | Linked static identity, capabilities, requirements, file digests |
| `interface-package.schema.json` | Immutable package manifest and declared interface exports |
| `interface-dependency-lock.schema.json` | Resolver-provided exact package dependency graph |

`go test ./internal/compiler -run TestDevelopmentMachine` exercises schema/Go
DTO field parity and real fixture/Artifact validation; the standard
`go test ./...` runs this regression as part of acceptance.

**Boundary:** JavaScript/Go clients must not interpret `NOT_ADMITTED` as a
failure to compile or as permission to execute. Signature verification, actual
Deployment Grants and Runtime Evidence belong to Host/Runtime, not Compiler.
Schema validation checks wire shapes, while the Compiler's code performs the
stronger cross-file, digest, closure, alias and semantic checks.

The v2.1.0 release gates tested these schemas together with the Go decoder,
exact package bytes, Development Artifact, frozen v2.0.2 golden baseline, and
real Chromium Worker parity. The source SHA is the published tag's commit.
Existing v2.0.2 artifacts and machine contracts remain unchanged.
