package compiler

// Draft Development v1alpha1 machine-contract conformance. A wire schema is
// valuable only when it agrees with the public Go DTOs and real emitted bytes.
// This test never promotes Development artifacts to production admission.
import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const developmentContractDir = "../../contracts/development/v1alpha1/"

func compileDevelopmentMachineContract(t *testing.T, filename string) (*jsonschema.Schema, map[string]any) {
	t.Helper()
	raw, err := os.ReadFile(developmentContractDir + filename)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	var instance any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&instance); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	id := "urn:test:development:" + filename
	if err := compiler.AddResource(id, instance); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(id)
	if err != nil {
		t.Fatal(err)
	}
	return compiled, document
}

func TestDevelopmentMachineSchemasAcceptRealFixtureAndArtifact(t *testing.T) {
	request := loadDevelopmentFixture(t)
	requestSchema, _ := compileDevelopmentMachineContract(t, "build-request.schema.json")
	if err := validateSchema(t, requestSchema, request); err != nil {
		t.Fatalf("valid Development BuildRequest violates public schema: %v", err)
	}
	lockSchema, _ := compileDevelopmentMachineContract(t, "interface-dependency-lock.schema.json")
	if err := validateSchema(t, lockSchema, request.Lock); err != nil {
		t.Fatalf("valid dependency Lock violates public schema: %v", err)
	}
	packageSchema, _ := compileDevelopmentMachineContract(t, "interface-package.schema.json")
	for _, snapshot := range request.Snapshots {
		found := false
		for _, file := range snapshot.Files {
			if file.Path != "package.json" {
				continue
			}
			found = true
			var manifest any
			if err := json.Unmarshal(file.Bytes, &manifest); err != nil {
				t.Fatal(err)
			}
			if err := validateSchema(t, packageSchema, manifest); err != nil {
				t.Fatalf("valid Interface Package %s violates public schema: %v", snapshot.Key.PackageID, err)
			}
		}
		if !found {
			t.Fatalf("missing manifest in %s", snapshot.Key.PackageID)
		}
	}
	build := BuildDevelopment(request)
	if !build.Success {
		t.Fatalf("Development fixture must build: %+v", build.Diagnostics)
	}
	resultSchema, _ := compileDevelopmentMachineContract(t, "build-result.schema.json")
	if err := validateSchema(t, resultSchema, build); err != nil {
		t.Fatalf("valid Development BuildResult violates public schema: %v", err)
	}
	artifactSchema, _ := compileDevelopmentMachineContract(t, "artifact-manifest.schema.json")
	var manifest any
	if err := json.Unmarshal(developmentOutputFile(t, build, "manifest.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if err := validateSchema(t, artifactSchema, manifest); err != nil {
		t.Fatalf("emitted Development Artifact manifest violates public schema: %v", err)
	}
}

func TestDevelopmentSchemaRequiredFieldsAndProfile(t *testing.T) {
	schema, _ := compileDevelopmentMachineContract(t, "build-request.schema.json")
	raw, err := json.Marshal(loadDevelopmentFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var base map[string]any
	if err := json.Unmarshal(raw, &base); err != nil {
		t.Fatal(err)
	}
	for _, property := range []string{"apiVersion", "languageTarget", "profile", "files", "targets", "lock", "snapshots"} {
		t.Run("missing-"+property, func(t *testing.T) {
			candidate := map[string]any{}
			for k, v := range base {
				if k != property {
					candidate[k] = v
				}
			}
			if err := validateSchema(t, schema, candidate); err == nil {
				t.Fatal("machine contract accepted missing required property")
			}
			if _, err := DecodeDevelopmentBuildRequestJSON(encodeJSONValue(t, candidate)); err == nil {
				t.Fatal("Go decoder accepted missing required property")
			}
		})
	}
	t.Run("production-profile", func(t *testing.T) {
		candidate := map[string]any{}
		for k, v := range base {
			candidate[k] = v
		}
		candidate["profile"] = "PRODUCTION"
		if err := validateSchema(t, schema, candidate); err == nil {
			t.Fatal("Development contract accepted PRODUCTION profile")
		}
		result := BuildDevelopment(DevelopmentBuildRequest{
			APIVersion: DevelopmentBuildRequestAPIVersion, LanguageTarget: V1Alpha5Target, Profile: "PRODUCTION",
			Files: loadDevelopmentFixture(t).Files, Targets: loadDevelopmentFixture(t).Targets,
			Lock: loadDevelopmentFixture(t).Lock, Snapshots: loadDevelopmentFixture(t).Snapshots,
		})
		if result.Success {
			t.Fatal("Go Compiler admitted PRODUCTION")
		}
	})
}

func TestDevelopmentMachineContractPropertiesMatchDTOs(t *testing.T) {
	cases := []struct {
		file string
		rootType reflect.Type
		defs map[string]reflect.Type
	}{
		{"build-request.schema.json", reflect.TypeOf(DevelopmentBuildRequest{}), map[string]reflect.Type{
			"file":reflect.TypeOf(File{}), "packageKey":reflect.TypeOf(DevelopmentPackageKey{}),
			"lockRoot":reflect.TypeOf(DevelopmentLockRoot{}), "lockEdge":reflect.TypeOf(DevelopmentLockEdge{}),
			"lockPackage":reflect.TypeOf(DevelopmentLockPackage{}), "lock":reflect.TypeOf(DevelopmentLock{}),
			"snapshot":reflect.TypeOf(DevelopmentSnapshot{}),
		}},
		{"build-result.schema.json", reflect.TypeOf(DevelopmentBuildResult{}), map[string]reflect.Type{
			"file":reflect.TypeOf(File{}), "diagnostic":reflect.TypeOf(Diagnostic{}),
		}},
		{"interface-dependency-lock.schema.json", reflect.TypeOf(DevelopmentLock{}), map[string]reflect.Type{
			"packageKey":reflect.TypeOf(DevelopmentPackageKey{}), "lockRoot":reflect.TypeOf(DevelopmentLockRoot{}),
			"lockEdge":reflect.TypeOf(DevelopmentLockEdge{}), "lockPackage":reflect.TypeOf(DevelopmentLockPackage{}),
		}},
		{"artifact-manifest.schema.json", reflect.TypeOf(developmentArtifactManifest{}), map[string]reflect.Type{
			"packageKey":reflect.TypeOf(DevelopmentPackageKey{}), "lockRoot":reflect.TypeOf(DevelopmentLockRoot{}),
			"lockEdge":reflect.TypeOf(DevelopmentLockEdge{}), "manifestLockPackage":reflect.TypeOf(developmentManifestLockPackage{}),
			"manifestLock":reflect.TypeOf(developmentManifestLock{}), "generator":reflect.TypeOf(developmentManifestGenerator{}),
			"link":reflect.TypeOf(DevelopmentLinkRecord{}), "capRequirement":reflect.TypeOf(DevelopmentCapabilityRequirement{}),
			"capCatalog":reflect.TypeOf(DevelopmentCapabilityCatalogIdentity{}), "capCallable":reflect.TypeOf(DevelopmentCapabilityCallable{}),
			"sourceDecl":reflect.TypeOf(DevelopmentSourceDeclaration{}), "requirements":reflect.TypeOf(developmentManifestRequirementDocuments{}),
			"manifestFile":reflect.TypeOf(developmentManifestFile{}),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			_, document := compileDevelopmentMachineContract(t, tc.file)
			check := func(label string, node map[string]any, typ reflect.Type) {
				t.Helper()
				if node["additionalProperties"] != false {
					t.Fatalf("%s must forbid unknown JSON properties",label)
				}
				props, ok := node["properties"].(map[string]any)
				if !ok {
					t.Fatalf("%s missing properties",label)
				}
				var names []string
				for k := range props {
					names=append(names,k)
				}
				sort.Strings(names)
				if !reflect.DeepEqual(names, jsonFieldNames(typ)) {
					t.Fatalf("%s schema properties %v != Go DTO %v",label,names,jsonFieldNames(typ))
				}
			}
			check("root",document,tc.rootType)
			defs, ok := document["$defs"].(map[string]any)
			if !ok {
				t.Fatal("missing $defs")
			}
			for name,typ := range tc.defs {
				node, ok := defs[name].(map[string]any)
				if !ok {
					t.Fatalf("missing $defs.%s",name)
				}
				check(name,node,typ)
			}
		})
	}
}
