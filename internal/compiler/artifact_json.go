package compiler

// Deterministic JSON serialization for the Compiler artifact path.
//
// RFC 8785 JSON Canonicalization Scheme (JCS) is applied through the pinned OSS
// implementation, so the same BuildRequest always produces byte-identical
// artifact documents on every host platform. There are no timestamps, random
// values or environment probes in the output.

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
)

// canonicalizeValue serializes a Go value as compact JSON and applies JCS.
func canonicalizeValue(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return canonicalizeJSON(bytes.TrimSpace(buffer.Bytes()))
}

// canonicalizeJSON applies RFC 8785 JCS to valid UTF-8 JSON bytes.
func canonicalizeJSON(data []byte) ([]byte, error) {
	canonical, err := jcs.Transform(data)
	if err != nil {
		return nil, err
	}
	if !json.Valid(canonical) {
		return nil, fmt.Errorf("JCS produced invalid JSON")
	}
	return canonical, nil
}
