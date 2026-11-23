# CA Skill Compiler Development v1alpha1 — draft machine contracts

These five JSON Schema (Draft 2020-12) documents describe the **unreleased**
Development Build family, independent of the already frozen v1alpha5 product
entry. Their code-defined identities remain v1alpha1. They are a review and
regression baseline, **not** a declaration of contract freeze, Host admission,
Owner trust, or product qualification.

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

Before an official v2.1.0 release, the v1alpha1 schemas and Go implementation
must be reviewed and frozen together at an exact source SHA. Existing
v2.0.2 artifacts and machine contracts remain unchanged.
