# Binding Resolver vNext — isolated Go spike

Status: **EXPERIMENT ONLY — NOT COMPILER V0.1 BEHAVIOR**

This package is deliberately outside `internal/compiler`. It does not modify
`internal/compiler/bindings.go`, any JSON Schema/YAML grammar, the paused Schema
Engine worktree, or the running 5GC Agent Host.

## Why this exists

The design in `ca-skills` PR #24 proposes replacing repeated `with/from`
plumbing with a deterministic compiled Binding Plan. The current MAS source has
11 common context fields forwarded into each of 16 calls (176 source mappings).
The existing 5GC Procedure similarly repeats UE/ingress/lifecycle aliases,
although many other resource handles must remain explicitly bound.

This spike tests a **narrow semantic resolution algorithm**:

1. The Context/Operation contract authorities **explicitly supply**
   `SemanticID` and `Type` for producer and consumer fields.
2. A separate, trusted CFG/dataflow analysis supplies `AvailableAt` for each
   producer, meaning it is definitely available on every path to that consumer.
3. The resolver chooses the unique same-semantic, type/nullability-compatible
   and definitely-available producer. It never uses field-name equality alone.
4. Optional inputs without a source remain unbound. Ambiguous or invalid
   required inputs produce fail-closed diagnostics.
5. The result is sorted and inspectable, intended as a **candidate Binding Plan**,
   not executable code.

The semantic identifiers in `resolver_test.go` are **proposed test
annotations**, not fields present in today's ToolCatalog or ca-skills v0.2.
The 16 synthetic consumer IDs reproduce the MAS repeated-field count; these
tests do **not** parse the upstream YAML nor prove whole-Procedure equivalence.

## Local test

Under Go 1.23 (the Compiler's current `go.mod` target):

```sh
go test ./internal/bindingspike -v
```

The original test files were also executed locally as a standalone Go package
with `GO111MODULE=off go test -v`; all positive and negative tests passed.
This is not a claim of remote CI success.

## Deliberate limits

- No actual YAML/JSON Schema ingestion or owner authority linking.
- No automatic construction/proof of CFG dominance, availability or optional
  branch semantics: `AvailableAt` is a **trusted fixture precondition**.
- No explicit author overrides, literal `value`, transforms, shadowing,
  workflow loops or recursion.
- `Type` uses conservative exact primitive matching, not the full Compiler
  Draft 2020-12 schema implication engine.
- No source freshness, Evidence contract, Capability permission, Operation
  Effect or Runtime handling.
- No deterministic artifact serialization/integration into BuildResult.
- The test demonstrates 176 resolvable MAS mappings **if** the shared semantic
  contracts are authored and certified, not that 176 mappings can be deleted
  from the present v0.2 files without changing behavior.
- Do not infer ambiguous `stateRef`/Fabric hold/N4 handle bindings.

## Proposed next integration gates

- Freeze semantic field-ID owner/vocabulary in `ca-skills`, without allowing
  Tool owners to hard-code Host paths.
- Compile the actual 5GC/MAS fixtures (including precise Operation Schema and
  Procedure CFG), compare explicit old bindings with generated results.
- Use current Compiler's schema compatibility and definite-assignment checks;
  do not introduce a second weaker verifier.
- Record source identity/contract digest and exact producer path in IR.
- Add negative cases for branches, loops, missing contract, same-name conflict,
  multiple producer generations, explicit override and version skew.
- Keep PR #24 design-only and this spike Draft until those gates pass.
