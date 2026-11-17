package compiler

// Shared Python TypedDict projection types used by the active v1alpha4 Script
// ABI derivation (the regex-era legacy parser is build-tagged out).

type pyField struct{ name, typ string }

type pyClass struct{ fields []pyField }
