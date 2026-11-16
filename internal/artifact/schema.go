package artifact

// Generated Script Output Schema file construction: exact Draft 2020-12 URI,
// stable $id and JCS bytes without insignificant whitespace or trailing newline.

import "fmt"

// ScriptOutputSchemaFile turns one derived Script output Schema value into its
// canonical generated Artifact file. The derived value must already be a
// CA-SL G.4-supported Schema object; this function only adds the frozen dialect
// and identity and canonicalizes the bytes.
func ScriptOutputSchemaFile(skill, scriptPath string, schema map[string]any) (File, error) {
	if schema == nil {
		return File{}, fmt.Errorf("Script output Schema is required for %s", scriptPath)
	}
	if _, exists := schema["type"]; !exists {
		return File{}, fmt.Errorf("Script output Schema for %s must declare a type", scriptPath)
	}
	path, err := ScriptOutputSchemaPath(skill, scriptPath)
	if err != nil {
		return File{}, err
	}
	id, err := ScriptOutputSchemaID(skill, scriptPath)
	if err != nil {
		return File{}, err
	}
	value := make(map[string]any, len(schema)+2)
	for key, entry := range schema {
		value[key] = entry
	}
	value["$schema"] = Draft2020
	value["$id"] = id
	canonical, err := CanonicalizeValue(value)
	if err != nil {
		return File{}, fmt.Errorf("canonicalize generated Schema %s: %w", path, err)
	}
	return File{Path: path, Bytes: canonical}, nil
}
