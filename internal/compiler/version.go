package compiler

// CompilerVersion is the product version of the CA Skill Compiler. It is
// independent of the CA-SL language version and of the Compiler API versions:
// the language target stays connectionagent.io/v1alpha5 while the product
// version advances per release.
//
// The v2.0.x product line is the single-version v1alpha5 Compiler. v1alpha4
// BuildRequest/CLI input is rejected, which is a breaking public-API change
// relative to the v1.0.0 product release.
const CompilerVersion = "v2.0.2"
