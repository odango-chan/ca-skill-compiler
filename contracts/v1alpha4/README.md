# Compiler v1alpha4 machine contracts

All files in this directory are versioned Compiler / imported-contract schemas. They do not change CA-SL language semantics.

- build-request.schema.json — source files, resolved DependencySnapshots and trusted build context.
- build-result.schema.json — diagnostics, used dependency aliases and generated output files.
- dependency-inspection.schema.json — read-only projection of callable and Schema exports for Studio.
- tool-catalog.schema.json — exact ToolCatalog v1alpha4 imported Owner contract.
- capability-contracts.schema.json — exact CapabilityContracts v1alpha4 deployment input.
- evidence-trust-bindings.schema.json — BuildRequest v1alpha4-owned embedded transport document for trusted IndependentDomains and common-fact relations; not an Evidence Descriptor API version.
- artifact-manifest.schema.json — generated per-Skill Compiler Artifact manifest.
- dependency-snapshot-digest-vectors.json — cross-language golden vectors.

## Version and trust rules

BuildRequest apiVersion, languageTarget, ToolCatalog apiVersion, CapabilityContracts apiVersion and Artifact apiVersion are independent and exact-match. No v1alpha3 / v0 fallback exists.

The BuildRequest trustedBuildContext is supplied by a trusted caller. A Skill file, dependency README or arbitrary imported document cannot self-assert deployment authorization, package identity or Owner ownership. The browser Compiler is not a security boundary; Host/Runtime revalidates authorization at use time.

Schema documents are Draft 2020-12 API contracts. The schemas imported by CA-SL source are a separate matter: they must follow the stricter CA-SL G.4 keyword whitelist.
