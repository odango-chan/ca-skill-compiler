package ingresscatalog

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The IngressCatalog v1alpha1 structural schema is the exact candidate schema
// from ca-skills PR #62 head 0fc6a0b0633a50afc478d7380accb46819c01afa. A second
// copy lives at contracts/v1alpha1/ingresscatalog.schema.json; a test asserts
// both files are byte-identical. Candidate/DRAFT: not an Owner-published
// contract.
//
//go:embed schemas/ingresscatalog.schema.json
var structuralSchemaBytes []byte

const structuralSchemaURL = "urn:connectionagent:ca:ingresscatalog:schema:v1alpha1"

var (
	structuralSchemaOnce sync.Once
	structuralSchema     *jsonschema.Schema
	structuralSchemaErr  error
)

// offlineLoader fails every load. The structural schema has no external
// references other than the embedded JSON Schema 2020-12 meta-schema, which the
// validator resolves from its own embedded copy, so this proves no network or
// filesystem lookup occurs.
type offlineLoader struct{}

func (offlineLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("network/schema loading is disabled for IngressCatalog validation: %s", url)
}

func compiledCatalogSchema() (*jsonschema.Schema, error) {
	structuralSchemaOnce.Do(func() {
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(structuralSchemaBytes))
		if err != nil {
			structuralSchemaErr = fmt.Errorf("decode embedded IngressCatalog schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		compiler.UseLoader(offlineLoader{})
		compiler.AssertFormat()
		if err := compiler.AddResource(structuralSchemaURL, document); err != nil {
			structuralSchemaErr = fmt.Errorf("add IngressCatalog schema resource: %w", err)
			return
		}
		structuralSchema, structuralSchemaErr = compiler.Compile(structuralSchemaURL)
	})
	return structuralSchema, structuralSchemaErr
}

type validationIssue struct {
	pointer string
	message string
}

func toValidationValue(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
}

// validateCatalogStructure runs the copied structural schema and returns
// deterministic leaf issues with canonical JSON Pointers.
func validateCatalogStructure(value any) ([]validationIssue, error) {
	schema, err := compiledCatalogSchema()
	if err != nil {
		return nil, err
	}
	candidate, err := toValidationValue(value)
	if err != nil {
		return nil, err
	}
	validationErr := schema.Validate(candidate)
	if validationErr == nil {
		return nil, nil
	}
	var structured *jsonschema.ValidationError
	if !errors.As(validationErr, &structured) {
		return nil, validationErr
	}
	issues := make([]validationIssue, 0, 8)
	collectValidationIssues(structured, &issues)
	if len(issues) == 0 {
		issues = append(issues, validationIssue{pointer: "#", message: structured.Error()})
	}
	sort.SliceStable(issues, func(left, right int) bool {
		return issues[left].pointer < issues[right].pointer
	})
	return issues, nil
}

func collectValidationIssues(validation *jsonschema.ValidationError, issues *[]validationIssue) {
	if len(validation.Causes) == 0 {
		pointer := "#"
		if len(validation.InstanceLocation) > 0 {
			pointer = canonicalPointer(jsonPointer(validation.InstanceLocation))
		}
		*issues = append(*issues, validationIssue{pointer: pointer, message: validation.Error()})
		return
	}
	for _, cause := range validation.Causes {
		if cause != nil {
			collectValidationIssues(cause, issues)
		}
	}
}
