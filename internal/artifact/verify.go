package artifact

// Artifact integrity verification for the frozen Compiler Artifact Contract.
// It validates a produced virtual tree without writing to disk: unique paths,
// one canonical manifest per Skill, every manifest entry present, no unlisted
// files, and generated Script Output Schema identities matching their paths.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gowebpki/jcs"
)

// Verify checks one produced artifact tree for structural and identity integrity.
func Verify(files []File) error {
	if len(files) == 0 {
		return fmt.Errorf("artifact tree is empty")
	}
	byPath := map[string][]byte{}
	for _, file := range files {
		if file.Path == "" {
			return fmt.Errorf("artifact contains an empty file path")
		}
		if _, exists := byPath[file.Path]; exists {
			return fmt.Errorf("artifact contains duplicate file path %q", file.Path)
		}
		if !strings.HasPrefix(file.Path, "skills/") {
			return fmt.Errorf("artifact file path %q is outside a Skill tree", file.Path)
		}
		byPath[file.Path] = file.Bytes
	}
	manifestPaths := make([]string, 0)
	for filePath := range byPath {
		if strings.HasSuffix(filePath, "/generated/artifact-manifest.json") {
			manifestPaths = append(manifestPaths, filePath)
		}
	}
	if len(manifestPaths) == 0 {
		return fmt.Errorf("artifact contains no artifact manifest")
	}
	sort.Strings(manifestPaths)
	listed := map[string]bool{}
	for _, manifestPath := range manifestPaths {
		skill, ok := skillOf(manifestPath)
		if !ok {
			return fmt.Errorf("manifest path %q is not inside a Skill tree", manifestPath)
		}
		raw := byPath[manifestPath]
		canonical, err := jcs.Transform(raw)
		if err != nil {
			return fmt.Errorf("manifest %q is not valid JSON: %w", manifestPath, err)
		}
		if !bytes.Equal(canonical, raw) {
			return fmt.Errorf("manifest %q is not canonical JCS", manifestPath)
		}
		var manifest manifest
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return fmt.Errorf("manifest %q cannot be decoded: %w", manifestPath, err)
		}
		if manifest.APIVersion != APIVersion || manifest.LanguageTarget != LanguageTarget || manifest.Generator.Name != GeneratorName {
			return fmt.Errorf("manifest %q has invalid apiVersion/languageTarget/generator", manifestPath)
		}
		if manifest.Skill.Name != skill {
			return fmt.Errorf("manifest %q names Skill %q but lives under %q", manifestPath, manifest.Skill.Name, skill)
		}
		if len(manifest.SourceFiles) == 0 {
			return fmt.Errorf("manifest %q must list source files", manifestPath)
		}
		prefix := "skills/" + skill + "/"
		listedBySkill := map[string]bool{}
		for _, entry := range append(append([]string{}, manifest.SourceFiles...), manifest.GeneratedFiles...) {
			if !strings.HasPrefix(entry, prefix) {
				return fmt.Errorf("manifest %q lists %q outside its Skill tree", manifestPath, entry)
			}
			if listedBySkill[entry] {
				return fmt.Errorf("manifest %q lists %q twice", manifestPath, entry)
			}
			listedBySkill[entry] = true
			if _, exists := byPath[entry]; !exists {
				return fmt.Errorf("manifest %q lists missing file %q", manifestPath, entry)
			}
			listed[entry] = true
		}
		for _, entry := range manifest.GeneratedFiles {
			if err := verifyGeneratedSchema(skill, entry, byPath[entry]); err != nil {
				return err
			}
		}
		for filePath := range byPath {
			if filePath == manifestPath || !strings.HasPrefix(filePath, prefix) {
				continue
			}
			if !listedBySkill[filePath] {
				return fmt.Errorf("artifact file %q is not listed by manifest %q", filePath, manifestPath)
			}
		}
	}
	for filePath := range byPath {
		if strings.HasSuffix(filePath, "/generated/artifact-manifest.json") {
			continue
		}
		if !listed[filePath] {
			return fmt.Errorf("artifact file %q is not listed by any manifest", filePath)
		}
	}
	return nil
}

func skillOf(manifestPath string) (string, bool) {
	trimmed := strings.TrimPrefix(manifestPath, "skills/")
	index := strings.Index(trimmed, "/")
	if index <= 0 {
		return "", false
	}
	return trimmed[:index], true
}

func verifyGeneratedSchema(skill, filePath string, raw []byte) error {
	prefix := "skills/" + skill + "/schemas/generated/"
	if !strings.HasPrefix(filePath, prefix) || !strings.HasSuffix(filePath, ".output.schema.json") {
		return nil // other generated files are format-checked by their own slices
	}
	var schema map[string]json.RawMessage
	if err := json.Unmarshal(raw, &schema); err != nil {
		return fmt.Errorf("generated Schema %q is not valid JSON: %w", filePath, err)
	}
	var id string
	if rawID, ok := schema["$id"]; ok {
		if err := json.Unmarshal(rawID, &id); err != nil {
			return fmt.Errorf("generated Schema %q has a non-string $id", filePath)
		}
	}
	relative := strings.TrimSuffix(strings.TrimPrefix(filePath, prefix), ".output.schema.json") + ".py"
	expected := "urn:connectionagent:skill:" + encodeRFC3986(skill) + ":script-output:" + encodeRFC3986(relative)
	if id != expected {
		return fmt.Errorf("generated Schema %q has $id %q, want %q", filePath, id, expected)
	}
	if scalarSchemaDialect(schema) != Draft2020 {
		return fmt.Errorf("generated Schema %q must declare the frozen Draft 2020-12 dialect", filePath)
	}
	return nil
}

func scalarSchemaDialect(schema map[string]json.RawMessage) string {
	raw, ok := schema["$schema"]
	if !ok {
		return ""
	}
	var dialect string
	if err := json.Unmarshal(raw, &dialect); err != nil {
		return ""
	}
	return dialect
}
