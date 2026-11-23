package compiler

// CompilerVersion is the product version of the CA Skill Compiler. It is
// independent of the CA-SL language version and of the Compiler API versions:
// the language target stays connectionagent.io/v1alpha5 while the product
// version advances per release.
//
// v2.1.0 adds the independent Development Build contract family without
// changing the previously frozen CA-SL v1alpha5 Build/Verify artifact bytes.
// Product version and frozen artifact-format generator identity are deliberately
// independent; do not substitute CompilerVersion in v1alpha5 manifest creation.
const CompilerVersion = "v2.1.0"

// V1Alpha5FrozenGeneratorVersion preserves byte-for-byte v2.0.2 output for
// the already-released v1alpha5 Build/Verify API, including the generator stamp.
// It is an output contract identity, NOT the version of the installed binary.
// New Development manifests use CompilerVersion (v2.1.0) instead.
const V1Alpha5FrozenGeneratorVersion = "v2.0.2"
