package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	js "github.com/santhosh-tekuri/jsonschema/v6"
)

type validationSnapshot struct {
	SchemaURL    string               `json:"schemaURL"`
	InstancePath []string             `json:"instancePath"`
	ErrorKind    string               `json:"errorKind"`
	KeywordPath  []string             `json:"keywordPath"`
	Localized    string               `json:"localized"`
	Causes       []validationSnapshot `json:"causes,omitempty"`
}

func snapshotValidationError(err error) (validationSnapshot, error) {
	var validationErr *js.ValidationError
	if !errors.As(err, &validationErr) {
		return validationSnapshot{}, fmt.Errorf("expected structured *jsonschema.ValidationError, got %T", err)
	}
	snapshot := validationSnapshot{
		SchemaURL:    validationErr.SchemaURL,
		InstancePath: append([]string(nil), validationErr.InstanceLocation...),
		ErrorKind:    fmt.Sprintf("%T", validationErr.ErrorKind),
		KeywordPath:  append([]string(nil), validationErr.ErrorKind.KeywordPath()...),
		Localized:    validationErr.Error(),
	}
	for _, cause := range validationErr.Causes {
		child, err := snapshotValidationError(cause)
		if err != nil {
			return validationSnapshot{}, err
		}
		snapshot.Causes = append(snapshot.Causes, child)
	}
	return snapshot, nil
}

func TestValidationErrorHasStableStructuredLocations(t *testing.T) {
	const root = "https://spike.invalid/diagnostics/person.yaml"
	documents := map[string]any{root: map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"age": map[string]any{"type": "integer"},
		},
		"required": []any{"age"},
	}}
	instance := map[string]any{"age": "young"}

	var want []byte
	for run := 0; run < 3; run++ {
		compiled, err := (Candidate{}).Compile(documents, root, false, "")
		if err != nil {
			t.Fatalf("compile run %d: %v", run, err)
		}
		err = compiled.Validator.Validate(instance)
		if err == nil {
			t.Fatalf("run %d: invalid instance passed", run)
		}
		snapshot, err := snapshotValidationError(err)
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		leaf := snapshot
		for len(leaf.KeywordPath) == 0 && len(leaf.Causes) > 0 {
			leaf = leaf.Causes[0]
		}
		if snapshot.SchemaURL != root+"#" || leaf.SchemaURL != root+"#/properties/age" || len(leaf.InstancePath) != 1 || leaf.InstancePath[0] != "age" || len(leaf.KeywordPath) == 0 {
			t.Fatalf("run %d: structured paths are incomplete: %+v", run, snapshot)
		}
		got, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			want = got
		} else if string(got) != string(want) {
			t.Fatalf("run %d changed structured error:\n got %s\nwant %s", run, got, want)
		}
	}
}
