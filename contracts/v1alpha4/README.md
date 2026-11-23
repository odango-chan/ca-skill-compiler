> **HISTORICAL / NOT CURRENT INPUT (v2.1.0).** Retained at its original path as frozen older-version or design evidence, not an active BuildRequest, syntax or CLI target. The current Compiler accepts **only CA-SL v1alpha5**; v1alpha4 inherited *semantics* remain implemented, but v1alpha4 API inputs are rejected. See [documentation](../../docs/README.md), [Development contracts](../development/v1alpha1/README.md) and [v2.1.0 Release](https://github.com/odango-chan/ca-skill-compiler/releases/tag/v2.1.0). Statements below about old implementation states are historical. <!-- historical-docs-v2.1.0 -->

---

# Compiler v1alpha4 machine contracts

**Status:** FROZEN Compiler-owned machine-contract schemas for implementation. These schemas do not change CA-SL language semantics.

All files in this directory are versioned Compiler / imported-contract schemas.

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

These files validate Compiler API / imported ToolCatalog / CapabilityContracts / deployment trust documents. Their use of JSON Schema composition keywords does not expand the CA-SL Schema language. The Compiler API / Owner-envelope meta-schemas MUST be validated separately from CA-SL data Schemas. Schemas referenced by Skill Request, Host Context, Tool Input/Output or Evidence data are separate Owner-published schemas and MUST follow the stricter CA-SL G.4 keyword whitelist. No generic Evidence Descriptor schema is invented here; each Evidence Owner must publish and version its real descriptor format.
