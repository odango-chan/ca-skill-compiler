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
