package compiler

// Shared Python TypedDict projection types used by the v1alpha5 Script
// ABI derivation.

type pyField struct{ name, typ string }

type pyClass struct{ fields []pyField }
