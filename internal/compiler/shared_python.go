package compiler

// Shared Python TypedDict projection types used by the active v1alpha4 Script
// ABI derivation.

type pyField struct{ name, typ string }

type pyClass struct{ fields []pyField }
