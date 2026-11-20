package compiler

// Test-only conversion of the released v1alpha5 trusted fixtures into an exact
// Development request scope. It groups each legacy dependency snapshot into one
// Interface Package, derives package.json exports from the actual payload
// documents, synthesizes a CapabilityCatalog from the deployment bundle and moves
// Evidence Descriptor documents into the optional Assurance Export. The resulting
// request is synthetic and compiler-owned; it does not claim a real Owner
// publication.

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type developmentPackageBuilder struct {
	alias            string
	packageID        string
	publisher        string
	files            map[string][]byte
	exports          []map[string]any
	dependencies     []map[string]any // alias/packageId/version
	capabilities     []map[string]any
	artifactBindings []map[string]any
	schemaBindings   []map[string]any
}

func developmentRequestFromLegacy(t *testing.T, legacy BuildRequest) DevelopmentBuildRequest {
	t.Helper()
	if legacy.TrustedBuildContext == nil {
		t.Fatalf("legacy fixture has no trusted build context")
	}
	// alias -> binding
	byAlias := map[string]TrustedDependencyBinding{}
	for _, binding := range legacy.TrustedBuildContext.DependencyBindings {
		byAlias[binding.DependencyID] = binding
	}
	publisherByOwner := map[string]string{}
	for _, dependency := range legacy.ResolvedDependencies {
		binding, exists := byAlias[dependency.DependencyID]
		if !exists {
			t.Fatalf("legacy dependency %s has no trusted binding", dependency.DependencyID)
		}
		if len(binding.AuthorizedOwners) == 0 {
			t.Fatalf("legacy dependency %s has no authorized owner", dependency.DependencyID)
		}
		publisherByOwner[binding.AuthorizedOwners[0]] = dependency.DependencyID
	}
	builders := map[string]*developmentPackageBuilder{}
	order := make([]string, 0, len(legacy.ResolvedDependencies))
	for _, dependency := range legacy.ResolvedDependencies {
		binding := byAlias[dependency.DependencyID]
		builder := &developmentPackageBuilder{
			alias: dependency.DependencyID, packageID: binding.StablePackageID,
			publisher: binding.AuthorizedOwners[0], files: map[string][]byte{},
		}
		for _, file := range dependency.Files {
			builder.files[file.Path] = append([]byte(nil), file.Bytes...)
		}
		builders[dependency.DependencyID] = builder
		order = append(order, dependency.DependencyID)
	}
	// Derive the declared export surface from the payload identities.
	for _, alias := range order {
		builder := builders[alias]
		paths := make([]string, 0, len(builder.files))
		for path := range builder.files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			root, err := yaml.Parse(builder.files[path])
			if err != nil {
				continue
			}
			if catalog, ok := parseToolCatalog(root, path); ok && catalog != nil {
				for _, callable := range catalog.callables {
					builder.exports = append(builder.exports, map[string]any{
						"kind": "CallableCatalog", "id": callable.ExportID, "path": path,
					})
				}
				continue
			}
			if export, ok := parseSchemaExport(root, path); ok {
				builder.exports = append(builder.exports, map[string]any{
					"kind": "Schema", "id": export.ExportID, "path": path,
				})
				continue
			}
			if scalar(child(root, "kind")) == "IngressCatalog" {
				for _, operation := range sequence(child(child(root, "spec"), "operations")) {
					id := scalar(child(operation, "id"))
					if id == "" {
						continue
					}
					export := map[string]any{"kind": "IngressCatalog", "id": id, "path": path}
					var artifacts, schemas []any
					for _, binding := range legacy.TrustedBuildContext.ArtifactBindings {
						if binding.DependencyID != alias {
							continue
						}
						artifacts = append(artifacts, map[string]any{
							"artifactRef": binding.ArtifactRef, "dependencyId": "self",
							"snapshotPath": binding.SnapshotPath, "dialect": binding.Dialect,
						})
					}
					for _, binding := range legacy.TrustedBuildContext.SchemaBindings {
						schemas = append(schemas, map[string]any{
							"exportId": binding.ExportID, "artifactRef": binding.ArtifactRef, "pointer": binding.Pointer,
						})
					}
					if len(artifacts) > 0 {
						export["artifactBindings"] = artifacts
					}
					if len(schemas) > 0 {
						export["schemaBindings"] = schemas
					}
					builder.exports = append(builder.exports, export)
				}
			}
		}
	}
	// Canonical artifact-root Schema exports come from the trusted schema
	// bindings: their package export ID is the canonical Schema identity, not the
	// file-level $id.
	for _, alias := range order {
		builder := builders[alias]
		artifactPaths := map[string]string{}
		for _, binding := range legacy.TrustedBuildContext.ArtifactBindings {
			if binding.DependencyID != alias {
				continue
			}
			artifactPaths[binding.ArtifactRef] = binding.SnapshotPath
		}
		exported := map[string]bool{}
		for _, export := range builder.exports {
			kind, _ := export["kind"].(string)
			id, _ := export["id"].(string)
			exported[kind+"\x00"+id] = true
		}
		for _, binding := range legacy.TrustedBuildContext.SchemaBindings {
			path, exists := artifactPaths[binding.ArtifactRef]
			if !exists || exported["Schema\x00"+binding.ExportID] {
				continue
			}
			dialect := ""
			for _, artifact := range legacy.TrustedBuildContext.ArtifactBindings {
				if artifact.DependencyID == alias && artifact.ArtifactRef == binding.ArtifactRef {
					dialect = artifact.Dialect
				}
			}
			builder.exports = append(builder.exports, map[string]any{
				"kind": "Schema", "id": binding.ExportID, "path": path,
				"artifactBindings": []any{map[string]any{
					"artifactRef": binding.ArtifactRef, "dependencyId": "self",
					"snapshotPath": path, "dialect": dialect,
				}},
				"schemaBindings": []any{map[string]any{
					"exportId": binding.ExportID, "artifactRef": binding.ArtifactRef, "pointer": binding.Pointer,
				}},
			})
			exported["Schema\x00"+binding.ExportID] = true
		}
	}
	// Synthesize the package-scoped CapabilityCatalog documents from the trusted
	// deployment CapabilityContracts bundle.
	if bundle := legacy.TrustedBuildContext.CapabilityContracts; bundle != nil {
		capabilities, diagnostics := parseCapabilityContracts(bundle.Files)
		if len(diagnostics) > 0 {
			t.Fatalf("legacy capability bundle diagnostics: %+v", diagnostics)
		}
		for _, capability := range capabilities {
			if len(capability.members) == 0 {
				continue
			}
			hostAlias, ok := publisherByOwner[capability.members[0].owner]
			if !ok {
				t.Fatalf("capability %s member owner %s has no package", capability.id, capability.members[0].owner)
			}
			host := builders[hostAlias]
			document := map[string]any{"capabilities": []any{}}
			members := make([]any, 0, len(capability.members))
			for _, member := range capability.members {
				memberAlias, ok := publisherByOwner[member.owner]
				if !ok {
					t.Fatalf("capability member owner %s has no package", member.owner)
				}
				entry := map[string]any{"callableId": member.callableID}
				if memberAlias != hostAlias {
					entry["dependencyAlias"] = memberAlias
					host.dependencies = append(host.dependencies, map[string]any{
						"alias": memberAlias, "packageId": builders[memberAlias].packageID, "version": "1.0.0",
					})
				}
				members = append(members, entry)
			}
			document["capabilities"] = append(document["capabilities"].([]any), map[string]any{
				"id": capability.id, "members": members,
			})
			raw, err := json.MarshalIndent(document, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			file := "capabilities-" + strings.ReplaceAll(capability.id, ".", "-") + ".json"
			host.files[file] = append(raw, '\n')
			host.exports = append(host.exports, map[string]any{"kind": "CapabilityCatalog", "id": capability.id, "path": file})
		}
	}
	// Evidence Descriptor documents are declared Interface Package exports. Their
	// bytes are fixed by the Dependency Lock like every other snapshot file.
	for _, alias := range order {
		builder := builders[alias]
		paths := make([]string, 0, len(builder.files))
		for path := range builder.files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		exported := map[string]bool{}
		for _, export := range builder.exports {
			kind, _ := export["kind"].(string)
			id, _ := export["id"].(string)
			exported[kind+"\x00"+id] = true
		}
		for _, path := range paths {
			root, err := yaml.Parse(builder.files[path])
			if err != nil {
				continue
			}
			descriptors, ok := parseEvidenceDescriptors(root)
			if !ok {
				continue
			}
			for _, exportID := range sortedStringMapKeys(descriptors) {
				if exported["EvidenceDescriptor\x00"+exportID] {
					continue
				}
				builder.exports = append(builder.exports, map[string]any{
					"kind": "EvidenceDescriptor", "id": exportID, "path": path,
				})
				exported["EvidenceDescriptor\x00"+exportID] = true
			}
		}
	}
	// Write package.json manifests and compute the exact snapshot identity.
	request := DevelopmentBuildRequest{
		APIVersion: DevelopmentBuildRequestAPIVersion, LanguageTarget: V1Alpha5Target,
		Profile: DevelopmentProfile, Targets: append([]string(nil), legacy.Targets...),
	}
	for _, file := range legacy.Files {
		request.Files = append(request.Files, File{Path: file.Path, Bytes: append([]byte(nil), file.Bytes...)})
	}
	keys := map[string]DevelopmentPackageKey{}
	for _, alias := range order {
		builder := builders[alias]
		files := make([]File, 0, len(builder.files))
		paths := make([]string, 0, len(builder.files))
		for path := range builder.files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		fileEntries := make([]any, 0, len(paths))
		for _, path := range paths {
			files = append(files, File{Path: path, Bytes: builder.files[path]})
			sum := sha256Hex(builder.files[path])
			fileEntries = append(fileEntries, map[string]any{"path": path, "sha256": sum})
		}
		exportEntries := make([]any, 0, len(builder.exports))
		for _, export := range builder.exports {
			exportEntries = append(exportEntries, export)
		}
		dependencyEntries := make([]any, 0, len(builder.dependencies))
		seenDependency := map[string]bool{}
		for _, dependency := range builder.dependencies {
			aliasName, _ := dependency["alias"].(string)
			if seenDependency[aliasName] {
				continue
			}
			seenDependency[aliasName] = true
			dependencyEntries = append(dependencyEntries, dependency)
		}
		manifest := map[string]any{
			"apiVersion": InterfacePackageAPIVersion, "packageId": builder.packageID,
			"version": "1.0.0", "languageTarget": V1Alpha5Target, "publisher": builder.publisher,
			"dependencies": dependencyEntries, "exports": exportEntries, "files": fileEntries,
		}
		raw, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, File{Path: "package.json", Bytes: append(raw, '\n')})
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		digest, err := ComputeDependencySnapshotDigest(files)
		if err != nil {
			t.Fatal(err)
		}
		key := DevelopmentPackageKey{PackageID: builder.packageID, Version: "1.0.0", SnapshotDigest: digest}
		keys[alias] = key
		request.Snapshots = append(request.Snapshots, DevelopmentSnapshot{Key: key, Files: files})
	}
	// Lock: every legacy alias is a root; package edges mirror declared
	// dependencies.
	request.Lock.APIVersion = InterfaceDependencyLockAPIVersion
	for _, alias := range order {
		builder := builders[alias]
		request.Lock.Roots = append(request.Lock.Roots, DevelopmentLockRoot{Alias: alias, Key: keys[alias]})
		entry := DevelopmentLockPackage{Key: keys[alias]}
		for _, dependency := range builder.dependencies {
			targetAlias, _ := dependency["alias"].(string)
			entry.Dependencies = append(entry.Dependencies, DevelopmentLockEdge{Alias: targetAlias, Key: keys[targetAlias]})
		}
		request.Lock.Packages = append(request.Lock.Packages, entry)
	}
	return request
}
