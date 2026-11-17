# v1alpha5 candidate contracts

This directory contains **candidate, DRAFT** machine contracts for the next
Compiler language target (`v1alpha5`). They are intentionally separate from the
frozen v1alpha4 contracts in `contracts/v1alpha4/`, which are **untouched** by
this workstream.

## Provenance

| File | Source | Source revision | Status |
|---|---|---|---|
| `ingresscatalog.schema.json` | `ca-skills` PR #62, `docs/design/schemas/ingresscatalog.v1alpha1.schema.json` | PR head commit `0fc6a0b0633a50afc478d7380accb46819c01afa` | candidate / DRAFT, **not Owner-published** |

The file is a byte-exact copy of the candidate structural schema except for the
removal of a leading UTF-8 byte-order mark, which is not part of the JSON
document. `internal/ingresscatalog/catalog_test.go`
(`TestCatalogStructuralSchemaCopiesAndNoNetwork`) asserts that this copy, the
embedded runtime copy at
`internal/ingresscatalog/schemas/ingresscatalog.schema.json`, and the compiled
schema stay identical.

## Scope of this copy

The schema validates only IngressCatalog structure, field types and local
conditions. It does **not** validate trusted dependency identity, cross-snapshot
uniqueness, Schema Closure resolution, Trigger Operation linking, Procedure
path analysis, deployment profile binding or live protocol behavior. Those
obligations remain with Compiler semantic admission (partially implemented in
`internal/ingresscatalog`) and later work packages.

## Version boundary

- v1alpha4 accepts no IngressCatalog, no `ingressOperation` export kind and no
  Event Wait syntax. Nothing in `contracts/v1alpha4/` or the v1alpha4 language
  contract changes here.
- These candidate contracts are not a release, not a stable API and not a claim
  of Owner publication, deployment qualification or 5GC interoperability.
