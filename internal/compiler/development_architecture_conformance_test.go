package compiler

import (
	"encoding/json"
	"strings"
	"testing"
)

// All Development wire properties are explicitly required, even when the
// semantic value is an empty array (targets: [] means select all Skills).
func TestDevelopmentRequestRequiredProperties(t *testing.T) {
	raw, err := json.Marshal(loadDevelopmentFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"apiVersion", "languageTarget", "profile", "files", "targets", "lock", "snapshots"} {
		t.Run(field, func(t *testing.T) {
			without := make(map[string]json.RawMessage, len(object))
			for k, v := range object {
				if k != field {
					without[k] = v
				}
			}
			payload, err := json.Marshal(without)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeDevelopmentBuildRequestJSON(payload); err == nil || !strings.Contains(err.Error(), field) {
				t.Fatalf("omitted required %q must fail precisely: %v", field, err)
			}
		})
	}
}

func TestDevelopmentPackageManifestRequiresDependencies(t *testing.T) {
	request := modifyDevelopmentManifest(t, loadDevelopmentFixture(t), "urn:example:ca:host-context",
		func(manifest map[string]any) { delete(manifest, "dependencies") })
	result := BuildDevelopment(request)
	if result.Success || !hasDevelopmentCode(result.Diagnostics, developmentSnapshotInvalid) {
		t.Fatalf("package.json missing dependencies must be rejected: %#v", result.Diagnostics)
	}
	if len(result.OutputFiles) != 0 || result.ArtifactDigest != "" {
		t.Fatal("invalid package manifest produced an Artifact")
	}
}

func TestDevelopmentSchemaBindingNeedsDirectDependency(t *testing.T) {
	owner := DevelopmentPackageKey{PackageID: "urn:test:owner", Version: "1.0.0", SnapshotDigest: "sha256:" + strings.Repeat("1", 64)}
	unrelated := DevelopmentPackageKey{PackageID: "urn:test:unrelated", Version: "1.0.0", SnapshotDigest: "sha256:" + strings.Repeat("2", 64)}
	export := developmentExport{
		kind: "Schema", id: "urn:test:local", path: "local.json",
		artifactBindings: []developmentArtifactBinding{{artifactRef: "urn:test:artifact", dependencyID: "self", snapshotPath: "local.json"}},
		schemaBindings: []developmentSchemaBinding{{exportID: "urn:test:unrelated-schema", artifactRef: "urn:test:artifact", pointer: "#"}},
	}
	env := &developmentEnvironment{
		packages: map[DevelopmentPackageKey]*developmentPackage{
			owner: {key: owner, filesByName: map[string][]byte{"local.json": []byte("{}")}, manifest: developmentPackageManifest{exports: []developmentExport{export}}},
			unrelated: {key: unrelated, manifest: developmentPackageManifest{exports: []developmentExport{{kind: "Schema", id: "urn:test:unrelated-schema", path: "schema.json"}}}},
		},
		dependencyGraph: map[DevelopmentPackageKey]map[string]DevelopmentPackageKey{
			owner: {}, unrelated: {},
		},
	}
	problems := validateDevelopmentBindingTargets(env)
	if len(problems) == 0 || !strings.Contains(strings.Join(problems, " | "), "direct dependency") {
		t.Fatalf("unrelated Schema export in Lock Closure must not satisfy a binding: %v", problems)
	}
	env.dependencyGraph[owner]["types"] = unrelated
	if problems := validateDevelopmentBindingTargets(env); len(problems) != 0 {
		t.Fatalf("Schema in a declared direct dependency must link: %v", problems)
	}
}


func TestDevelopmentPackageManifestWireShapeIsStrict(t *testing.T) {
	key := DevelopmentPackageKey{
		PackageID: "urn:test:manifest", Version: "1.0.0",
		SnapshotDigest: "sha256:" + strings.Repeat("1", 64),
	}
	baseline := map[string]any{
		"apiVersion": InterfacePackageAPIVersion,
		"packageId": key.PackageID, "version": key.Version,
		"languageTarget": V1Alpha5Target, "publisher": "test-owner",
		"dependencies": []any{}, "exports": []any{},
		"files": []any{map[string]any{"path":"example.json","sha256":"sha256:"+strings.Repeat("2",64)}},
	}
	tests := []struct {
		name string
		change func(map[string]any)
	}{
		{"dependency-extra",func(m map[string]any) {
			m["dependencies"]=[]any{map[string]any{
				"alias":"types","packageId":"urn:test:types","version":"1.0.0","unexpected":true,
			}}
		}},
		{"file-extra",func(m map[string]any) {
			m["files"]=[]any{map[string]any{"path":"example.json",
				"sha256":"sha256:"+strings.Repeat("2",64),"unexpected":true}}
		}},
		{"artifact-binding-extra",func(m map[string]any) {
			m["exports"]=[]any{map[string]any{"kind":"Schema","id":"urn:test:schema",
				"path":"example.json","artifactBindings":[]any{map[string]any{
					"artifactRef":"urn:test:artifact","dependencyId":"self",
					"snapshotPath":"example.json","dialect":"json-schema","unexpected":true,
				}}}}
		}},
		{"schema-binding-extra",func(m map[string]any) {
			m["exports"]=[]any{map[string]any{"kind":"Schema","id":"urn:test:schema",
				"path":"example.json","schemaBindings":[]any{map[string]any{
					"exportId":"urn:test:schema","artifactRef":"urn:test:artifact",
					"pointer":"#","unexpected":true,
				}}}}
		}},
	}
	for _,tc:=range tests {
		t.Run(tc.name,func(t *testing.T) {
			manifest:=map[string]any{}
			for k,v:=range baseline { manifest[k]=v }
			tc.change(manifest)
			raw,err:=json.Marshal(manifest)
			if err!=nil {t.Fatal(err)}
			_,problems:=parseDevelopmentPackageManifest(raw,key)
			if !strings.Contains(strings.Join(problems," | "),"unsupported field: unexpected") {
				t.Fatalf("unexpected extension must fail closed: %v",problems)
			}
		})
	}
	for name,raw:=range map[string][]byte{
		"yaml-only": []byte("apiVersion: connectionagent.io/interface-package/v1alpha1\n"),
		"duplicate": []byte(`{"apiVersion":"one","apiVersion":"two"}`),
		"trailing": []byte(`{"apiVersion":"one"} {"apiVersion":"two"}`),
	} {
		t.Run(name,func(t *testing.T) {
			_,problems:=parseDevelopmentPackageManifest(raw,key)
			if len(problems)==0 || !strings.Contains(strings.Join(problems," | "),"strict JSON") {
				t.Fatalf("%s must fail as invalid package JSON: %v",name,problems)
			}
		})
	}
}
