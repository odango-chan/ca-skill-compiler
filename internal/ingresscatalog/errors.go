package ingresscatalog

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, machine-readable reason for a Schema Closure Digest v1
// or trusted-resolution-context rejection. The catalogue admission layer maps
// these to the diagnostic codes required by the Ingress diagnostic registry.
type ErrorCode string

const (
	// Trusted resolution context construction.
	ErrSnapshotBaseURI     ErrorCode = "INVALID_SNAPSHOT_BASE_URI"
	ErrSnapshotPath        ErrorCode = "INVALID_SNAPSHOT_PATH"
	ErrDocumentURI         ErrorCode = "INVALID_DOCUMENT_URI"
	ErrDocumentURIMismatch ErrorCode = "DOCUMENT_URI_MISMATCH"
	ErrArtifactBinding     ErrorCode = "ARTIFACT_BINDING_CONFLICT"
	ErrArtifactDigest      ErrorCode = "ARTIFACT_DIGEST_MISMATCH"
	ErrArtifactParse       ErrorCode = "ARTIFACT_PARSE_ERROR"
	ErrArtifactDocument    ErrorCode = "ARTIFACT_PARSED_DOCUMENT_MISMATCH"
	ErrUnsupportedDialect  ErrorCode = "UNSUPPORTED_DIALECT"
	ErrUnsupportedValue    ErrorCode = "UNSUPPORTED_JSON_VALUE"
	ErrIDAlias             ErrorCode = "INVALID_ID_ALIAS"
	ErrAmbiguousAlias      ErrorCode = "AMBIGUOUS_RESOURCE_ALIAS"
	ErrSchemaDocument      ErrorCode = "SCHEMA_DOCUMENT_INVALID"

	// Root and reference resolution.
	ErrRootResolution ErrorCode = "ROOT_SCHEMA_UNRESOLVED"
	ErrRefResolution  ErrorCode = "REF_UNRESOLVED"
	ErrMissingPointer ErrorCode = "JSON_POINTER_MISSING"
	ErrInvalidPointer ErrorCode = "INVALID_JSON_POINTER"
	ErrInvalidURI     ErrorCode = "INVALID_URI_REFERENCE"
	ErrNonSchema      ErrorCode = "NON_SCHEMA_TARGET"
	ErrBooleanSchema  ErrorCode = "BOOLEAN_SCHEMA_UNSUPPORTED"
	ErrUnsupportedRef ErrorCode = "UNSUPPORTED_REFERENCE_KEYWORD"

	// Closure/manifest construction.
	ErrClosureDigest ErrorCode = "CLOSURE_DIGEST_FAILED"
)

// Error is a structured rejection. Detail is not part of any wire contract but
// is stable enough for local tests and operator-facing messages.
type Error struct {
	Code        ErrorCode
	ArtifactRef string
	Pointer     string
	Detail      string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	location := e.ArtifactRef
	if e.Pointer != "" {
		location += "#" + e.Pointer
	}
	detail := e.Detail
	if detail == "" {
		detail = string(e.Code)
	}
	if location != "" {
		return fmt.Sprintf("%s: %s: %s", e.Code, location, detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, detail)
}

func newError(code ErrorCode, detail string) *Error {
	return &Error{Code: code, Detail: detail}
}

func newArtifactError(code ErrorCode, artifactRef, pointer, detail string) *Error {
	return &Error{Code: code, ArtifactRef: artifactRef, Pointer: pointer, Detail: detail}
}

// CodeOf returns the stable code carried by err, if it is a structured
// package error. It returns ("", false) for other errors.
func CodeOf(err error) (ErrorCode, bool) {
	var structured *Error
	if errors.As(err, &structured) {
		return structured.Code, true
	}
	return "", false
}
