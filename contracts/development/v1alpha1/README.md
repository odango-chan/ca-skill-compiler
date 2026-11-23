# CA Skill Compiler Development v1alpha1 — v2.1.0 release candidate contracts

These five JSON Schema (Draft 2020-12) documents describe the Development Build
family introduced by CA Skill Compiler v2.1.0. Their code-defined identities
remain v1alpha1; this candidate becomes the frozen product contract surface **only
when v2.1.0 is successfully published from the reviewed release commit**. Before
that release, they remain candidates. This is never a Host admission, Owner trust,
or product qualification claim.

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

The v2.1.0 release gates must test these schemas together with the Go decoder,
exact package bytes, candidate Artifact, frozen v2.0.2 golden baseline, and real
Chromium Worker parity. The source SHA is the tag's commit after publication.
Existing v2.0.2 artifacts and machine contracts remain unchanged.
