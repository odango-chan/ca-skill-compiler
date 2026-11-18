package compiler

// BuildRequest schema/decoder contract regression.
//
// The frozen v1alpha5 BuildRequest schema and the public strict JSON decoder must
// describe exactly the same property set. Artifact and Schema bindings are legal
// only inside trustedBuildContext; a retired top-level artifactBinding or
// schemaBinding must be rejected by both the schema and the decoder. A v1alpha4
// API or language identity must keep failing closed.

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const buildRequestContractPath = "../../contracts/v1alpha5/build-request.schema.json"

func compileBuildRequestContract(t *testing.T) *jsonschema.Schema {
	t.Helper()
	raw, err := os.ReadFile(buildRequestContractPath)
	if err != nil {
		t.Fatalf("read schema %s: %v", buildRequestContractPath, err)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode schema %s: %v", buildRequestContractPath, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("urn:test:build-request:v1", value); err != nil {
		t.Fatalf("register schema %s: %v", buildRequestContractPath, err)
	}
	schema, err := compiler.Compile("urn:test:build-request:v1")
	if err != nil {
		t.Fatalf("compile schema %s: %v", buildRequestContractPath, err)
	}
	return schema
}

func loadBuildRequestSchemaObject(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(buildRequestContractPath)
	if err != nil {
		t.Fatalf("read schema %s: %v", buildRequestContractPath, err)
	}
	var schemaObject map[string]any
	if err := json.Unmarshal(raw, &schemaObject); err != nil {
		t.Fatalf("decode schema %s: %v", buildRequestContractPath, err)
	}
	return schemaObject
}

// validateSchema returns the schema violation instead of failing the test so the
// rejection direction can be asserted as well as the acceptance direction.
func validateSchema(t *testing.T, schema *jsonschema.Schema, value any) error {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal schema instance: %v", err)
	}
	var instance any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&instance); err != nil {
		t.Fatalf("decode schema instance: %v", err)
	}
	return schema.Validate(instance)
}

func buildRequestJSONObject(t *testing.T, req BuildRequest) map[string]any {
	t.Helper()
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal BuildRequest: %v", err)
	}
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		t.Fatalf("decode BuildRequest JSON object: %v", err)
	}
	return object
}

func encodeJSONValue(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON value: %v", err)
	}
	return raw
}

// TestBuildRequestSchemaAndDecoderAcceptFrozenShape proves the legal v1alpha5
// shape, including the only legal binding paths, through the real schema and the
// public strict decoder.
func TestBuildRequestSchemaAndDecoderAcceptFrozenShape(t *testing.T) {
	req := wp3BuildRequest(t, wp3FixtureOptions{})
	if req.TrustedBuildContext == nil {
		t.Fatal("fixture must carry a trustedBuildContext")
	}
	if len(req.TrustedBuildContext.ArtifactBindings) == 0 {
		t.Fatal("fixture must exercise trustedBuildContext.artifactBindings")
	}
	if len(req.TrustedBuildContext.SchemaBindings) == 0 {
		t.Fatal("fixture must exercise trustedBuildContext.schemaBindings")
	}

	schema := compileBuildRequestContract(t)
	if err := validateSchema(t, schema, req); err != nil {
		t.Fatalf("legal v1alpha5 BuildRequest violates the frozen schema: %v", err)
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal BuildRequest: %v", err)
	}
	decoded, err := DecodeBuildRequestJSON(raw)
	if err != nil {
		t.Fatalf("legal v1alpha5 BuildRequest rejected by the public decoder: %v", err)
	}
	if got, want := len(decoded.TrustedBuildContext.ArtifactBindings), len(req.TrustedBuildContext.ArtifactBindings); got != want {
		t.Fatalf("decoder lost artifactBindings: got %d, want %d", got, want)
	}
	if got, want := len(decoded.TrustedBuildContext.SchemaBindings), len(req.TrustedBuildContext.SchemaBindings); got != want {
		t.Fatalf("decoder lost schemaBindings: got %d, want %d", got, want)
	}
	if diagnostics := Check(decoded); hasErrorDiagnostic(diagnostics) {
		t.Fatalf("legal v1alpha5 BuildRequest must pass Check: %+v", diagnostics)
	}
}

// TestTrustedBindingArraysAreLegalInsideTrustedBuildContext validates the two
// binding arrays as the only legal path, independent of the rest of the fixture.
func TestTrustedBindingArraysAreLegalInsideTrustedBuildContext(t *testing.T) {
	full := buildRequestJSONObject(t, wp3BuildRequest(t, wp3FixtureOptions{}))
	fullContext, ok := full["trustedBuildContext"].(map[string]any)
	if !ok {
		t.Fatal("fixture JSON must carry trustedBuildContext")
	}

	schema := compileBuildRequestContract(t)
	for _, name := range []string{"artifactBindings", "schemaBindings"} {
		t.Run(name, func(t *testing.T) {
			bindings, ok := fullContext[name].([]any)
			if !ok || len(bindings) == 0 {
				t.Fatalf("fixture JSON must carry a nonempty %s array", name)
			}
			request := map[string]any{
				"apiVersion":           BuildRequestAPIVersion,
				"languageTarget":       V1Alpha5Target,
				"files":                full["files"],
				"resolvedDependencies": []any{},
				"trustedBuildContext":  map[string]any{name: bindings},
			}
			if err := validateSchema(t, schema, request); err != nil {
				t.Fatalf("trustedBuildContext.%s must be schema-valid: %v", name, err)
			}
			decoded, err := DecodeBuildRequestJSON(encodeJSONValue(t, request))
			if err != nil {
				t.Fatalf("trustedBuildContext.%s must be decoder-valid: %v", name, err)
			}
			switch name {
			case "artifactBindings":
				if len(decoded.TrustedBuildContext.ArtifactBindings) != len(bindings) {
					t.Fatalf("artifactBindings not decoded: %+v", decoded.TrustedBuildContext)
				}
			case "schemaBindings":
				if len(decoded.TrustedBuildContext.SchemaBindings) != len(bindings) {
					t.Fatalf("schemaBindings not decoded: %+v", decoded.TrustedBuildContext)
				}
			}
		})
	}
}

// TestBuildRequestRejectsRetiredTopLevelBindings proves that the retired
// top-level artifactBinding / schemaBinding properties are rejected by both the
// frozen schema and the public decoder, while their $defs shapes stay valid
// values only at the nested path.
func TestBuildRequestRejectsRetiredTopLevelBindings(t *testing.T) {
	schema := compileBuildRequestContract(t)
	base := buildRequestJSONObject(t, wp3BuildRequest(t, wp3FixtureOptions{}))

	retired := map[string]any{
		"artifactBinding": map[string]any{
			"artifactRef":  "urn:example:artifact:update-sm-context",
			"dependencyId": "tools",
			"snapshotPath": "artifacts/openapi.json",
			"dialect":      "openapi-3.0",
		},
		"schemaBinding": map[string]any{
			"exportId":    "urn:example:export:update-request",
			"artifactRef": "urn:example:artifact:update-request",
			"pointer":     "#",
		},
	}
	for name, value := range retired {
		t.Run(name, func(t *testing.T) {
			object := make(map[string]any, len(base)+1)
			for key, entry := range base {
				object[key] = entry
			}
			object[name] = value
			if err := validateSchema(t, schema, object); err == nil {
				t.Fatalf("schema accepted the retired top-level %s property", name)
			}
			if _, err := DecodeBuildRequestJSON(encodeJSONValue(t, object)); err == nil {
				t.Fatalf("strict decoder accepted the retired top-level %s property", name)
			}
		})
	}
}

// TestBuildRequestRejectsUnknownTopLevelProperty keeps additionalProperties
// false enforced on both sides, including the plural top-level spelling that
// would otherwise look like the legal nested array.
func TestBuildRequestRejectsUnknownTopLevelProperty(t *testing.T) {
	schema := compileBuildRequestContract(t)
	base := buildRequestJSONObject(t, wp3BuildRequest(t, wp3FixtureOptions{}))

	for name, value := range map[string]any{
		"artifactBindings": []any{},
		"schemaBindings":   []any{},
		"unknownProperty":  true,
	} {
		t.Run(name, func(t *testing.T) {
			object := make(map[string]any, len(base)+1)
			for key, entry := range base {
				object[key] = entry
			}
			object[name] = value
			if err := validateSchema(t, schema, object); err == nil {
				t.Fatalf("schema accepted unknown top-level property %q", name)
			}
			if _, err := DecodeBuildRequestJSON(encodeJSONValue(t, object)); err == nil {
				t.Fatalf("strict decoder accepted unknown top-level property %q", name)
			}
		})
	}
}

// TestBuildRequestRetiredLanguageIdentityFailsClosed keeps the v1alpha4 API and
// language identities rejected by the schema and fail-closed by the public Build
// path. The decoder itself is a structural transport decoder: it must not rewrite
// or sniff the identity on the way in.
func TestBuildRequestRetiredLanguageIdentityFailsClosed(t *testing.T) {
	schema := compileBuildRequestContract(t)
	base := buildRequestJSONObject(t, wp3BuildRequest(t, wp3FixtureOptions{}))

	cases := []struct {
		name     string
		property string
		value    string
		wantCode string
	}{
		{
			name:     "v1alpha4 apiVersion",
			property: "apiVersion",
			value:    "connectionagent.io/compiler-build-request/v1alpha4",
			wantCode: "REQUEST_INVALID",
		},
		{
			name:     "v1alpha4 languageTarget",
			property: "languageTarget",
			value:    "connectionagent.io/v1alpha4",
			wantCode: "LANGUAGE_TARGET_UNSUPPORTED",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			object := make(map[string]any, len(base))
			for key, entry := range base {
				object[key] = entry
			}
			object[tc.property] = tc.value
			if err := validateSchema(t, schema, object); err == nil {
				t.Fatalf("schema accepted retired identity %q", tc.value)
			}

			req, err := DecodeBuildRequestJSON(encodeJSONValue(t, object))
			if err != nil {
				return // structural rejection is also fail-closed
			}
			switch tc.property {
			case "apiVersion":
				if req.APIVersion != tc.value {
					t.Fatalf("decoder rewrote apiVersion: got %q", req.APIVersion)
				}
			case "languageTarget":
				if req.LanguageTarget != tc.value {
					t.Fatalf("decoder rewrote languageTarget: got %q", req.LanguageTarget)
				}
			}
			diagnostics := Check(req)
			if !hasCode(diagnostics, tc.wantCode) {
				t.Fatalf("retired identity must fail closed with %s: %+v", tc.wantCode, diagnostics)
			}
			result := Build(req)
			if result.Success {
				t.Fatal("retired identity must never build successfully")
			}
			if len(result.OutputFiles) != 0 {
				t.Fatalf("retired identity must not emit artifact files: %+v", result.OutputFiles)
			}
		})
	}
}

// TestBuildRequestSchemaPropertiesMatchPublicDTOs is the field-level review
// guard: every schema object must describe exactly the JSON properties the public
// Go DTO decodes, so no property can be schema-allowed while the strict decoder
// rejects it (or vice versa).
func TestBuildRequestSchemaPropertiesMatchPublicDTOs(t *testing.T) {
	root := loadBuildRequestSchemaObject(t)

	defs, ok := root["$defs"].(map[string]any)
	if !ok {
		t.Fatal("schema must carry $defs")
	}
	for _, name := range []string{"artifactBinding", "schemaBinding"} {
		if _, ok := defs[name].(map[string]any); !ok {
			t.Fatalf("$defs.%s must remain defined even though the top-level property is retired", name)
		}
	}

	schemaNode := func(t *testing.T, name string) map[string]any {
		t.Helper()
		node, ok := defs[name].(map[string]any)
		if !ok {
			t.Fatalf("$defs.%s must be an object", name)
		}
		return node
	}

	cases := []struct {
		label    string
		node     map[string]any
		goType   reflect.Type
		closedTo bool
	}{
		{"BuildRequest", root, reflect.TypeOf(BuildRequest{}), true},
		{"trustedBuildContext", schemaNode(t, "trustedBuildContext"), reflect.TypeOf(TrustedBuildContext{}), true},
		{"bytesFile", schemaNode(t, "bytesFile"), reflect.TypeOf(File{}), true},
		{"sourceFile", schemaNode(t, "sourceFile"), reflect.TypeOf(File{}), true},
		{"resolvedDependency", schemaNode(t, "resolvedDependency"), reflect.TypeOf(ResolvedDependency{}), true},
		{"exportOwner", schemaNode(t, "exportOwner"), reflect.TypeOf(ExportOwner{}), true},
		{"dependencyBinding", schemaNode(t, "dependencyBinding"), reflect.TypeOf(TrustedDependencyBinding{}), true},
		{"capabilityBundle", schemaNode(t, "capabilityBundle"), reflect.TypeOf(CapabilityContractsBundle{}), true},
		{"deploymentAuthorization", schemaNode(t, "deploymentAuthorization"), reflect.TypeOf(DeploymentAuthorization{}), true},
		{"evidenceBundle", schemaNode(t, "evidenceBundle"), reflect.TypeOf(EvidenceTrustBindingsBundle{}), true},
		{"artifactBinding", schemaNode(t, "artifactBinding"), reflect.TypeOf(TrustedArtifactBinding{}), true},
		{"schemaBinding", schemaNode(t, "schemaBinding"), reflect.TypeOf(TrustedSchemaBinding{}), true},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			if tc.closedTo {
				if closed, ok := tc.node["additionalProperties"].(bool); !ok || closed {
					t.Fatalf("%s must keep additionalProperties: false", tc.label)
				}
			}
			properties, ok := tc.node["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s must declare properties", tc.label)
			}
			var schemaNames []string
			for name := range properties {
				schemaNames = append(schemaNames, name)
			}
			sort.Strings(schemaNames)
			goNames := jsonFieldNames(tc.goType)
			if !reflect.DeepEqual(schemaNames, goNames) {
				t.Fatalf("%s schema properties %v do not match the public DTO JSON fields %v", tc.label, schemaNames, goNames)
			}
		})
	}
}

// jsonFieldNames returns the sorted JSON property names of a public DTO type.
func jsonFieldNames(typ reflect.Type) []string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	var names []string
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
