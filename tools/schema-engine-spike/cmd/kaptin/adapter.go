package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	js "github.com/kaptinlin/jsonschema"
	"github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
)

type Candidate struct{}

func (Candidate) Name() string { return "kaptinlin/jsonschema@v0.9.10" }

func (Candidate) Compile(documents map[string]any, rootURI string, assertFormat bool, _ string) (spike.Compiled, error) {
	registry := spike.NewRegistry(documents)
	_, source, err := registry.Resolve(rootURI)
	if err != nil {
		return spike.Compiled{Trace: registry.Trace}, err
	}
	c := js.NewCompiler()
	// NewCompiler pre-registers HTTP and HTTPS loaders. Replace the entire map
	// before compilation so every URI scheme can only consult our in-memory set.
	c.Loaders = make(map[string]func(string) (io.ReadCloser, error))
	c.SetDefaultBaseURI(rootURI)
	for _, scheme := range []string{"", "mem", "http", "https", "file", "urn", "ftp", "gopher", "custom"} {
		s := scheme
		c.RegisterLoader(s, func(uri string) (io.ReadCloser, error) {
			_, b, err := registry.Resolve(uri)
			if err != nil {
				return nil, err
			}
			return io.NopCloser(bytes.NewReader(b)), nil
		})
	}
	if assertFormat {
		c.SetAssertFormat(true)
	}
	schema, err := c.Compile(source, rootURI)
	if err != nil {
		return spike.Compiled{Trace: registry.Trace}, err
	}
	return spike.Compiled{Validator: validator{schema}, Trace: registry.Trace}, nil
}

type validator struct{ schema *js.Schema }

func (v validator) Validate(value any) error {
	result := v.schema.Validate(value)
	if result.IsValid() {
		return nil
	}
	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("validation failed; error encoding result: %w", err)
	}
	return fmt.Errorf("validation failed: %s", b)
}
