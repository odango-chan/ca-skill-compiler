package compiler

// Package-scoped Schema resolver tests.
//
// The Development profile resolves every $ref in the scope of the document that
// declares it: a Skill-owned document uses the Skill Contract aliases, and a
// Package-owned Schema uses that Package's own declared direct dependencies. A
// Schema that merely exists somewhere in the Lock Closure is never a legal
// target.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"testing"
)

type developmentTestPackage struct {
	ID        string
	Publisher string
	Files     map[string][]byte // relative path -> payload (package.json excluded)
	Exports   []map[string]any  // declared exports
	Deps      map[string]string // alias -> packageId
}

// addDevelopmentTestGraph appends a set of Interface Packages, their lock
// packages and the requested roots to an existing Development request. The
// helper computes package manifests and snapshot digests exactly like a real
// Resolver would.
func addDevelopmentTestGraph(t *testing.T, request DevelopmentBuildRequest, packages []developmentTestPackage, roots map[string]string) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	keyByID := map[string]DevelopmentPackageKey{}
	for _, snapshot := range request.Snapshots {
		keyByID[snapshot.Key.PackageID] = snapshot.Key
	}
	snapshots := make([]DevelopmentSnapshot, 0, len(packages))
	for _, pkg := range packages {
		paths := make([]string, 0, len(pkg.Files))
		for path := range pkg.Files {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		files := make([]File, 0, len(paths)+1)
		fileEntries := make([]any, 0, len(paths))
		for _, path := range paths {
			files = append(files, File{Path: path, Bytes: pkg.Files[path]})
			sum := sha256.Sum256(pkg.Files[path])
			fileEntries = append(fileEntries, map[string]any{"path": path, "sha256": "sha256:" + hex.EncodeToString(sum[:])})
		}
		dependencyEntries := make([]any, 0, len(pkg.Deps))
		for _, alias := range sortedStringMapKeys(pkg.Deps) {
			dependencyEntries = append(dependencyEntries, map[string]any{
				"alias": alias, "packageId": pkg.Deps[alias], "version": "1.0.0",
			})
		}
		exportEntries := make([]any, 0, len(pkg.Exports))
		for _, export := range pkg.Exports {
			exportEntries = append(exportEntries, export)
		}
		manifest := map[string]any{
			"apiVersion": InterfacePackageAPIVersion, "packageId": pkg.ID,
			"version": "1.0.0", "languageTarget": V1Alpha5Target, "publisher": pkg.Publisher,
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
		key := DevelopmentPackageKey{PackageID: pkg.ID, Version: "1.0.0", SnapshotDigest: digest}
		keyByID[pkg.ID] = key
		request.Snapshots = append(request.Snapshots, DevelopmentSnapshot{Key: key, Files: files})
		snapshots = append(snapshots, request.Snapshots[len(request.Snapshots)-1])
	}
	for _, pkg := range packages {
		entry := DevelopmentLockPackage{Key: keyByID[pkg.ID]}
		for _, alias := range sortedStringMapKeys(pkg.Deps) {
			key, exists := keyByID[pkg.Deps[alias]]
			if !exists {
				t.Fatalf("test package %s depends on unknown package %s", pkg.ID, pkg.Deps[alias])
			}
			entry.Dependencies = append(entry.Dependencies, DevelopmentLockEdge{Alias: alias, Key: key})
		}
		request.Lock.Packages = append(request.Lock.Packages, entry)
	}
	for _, alias := range sortedStringMapKeys(roots) {
		key, exists := keyByID[roots[alias]]
		if !exists {
			t.Fatalf("test root %s resolves to unknown package %s", alias, roots[alias])
		}
		request.Lock.Roots = append(request.Lock.Roots, DevelopmentLockRoot{Alias: alias, Key: key})
	}
	return request
}

// enableDevelopmentRequestSchemaRoot adds the test Package chain to the fixture
// Skill: the Skill declares dependency alias "a" and a Skill-owned Request Schema
// that $refs A. A then references B through A's own alias, and B references C
// through B's own alias.
func enableDevelopmentRequestSchemaRoot(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
	t.Helper()
	contractPath := "skills/verify-ue-registration/assurance/contract.yaml"
	// The Skill declares one more root alias and a Skill-owned Request Schema
	// whose $ref is a Skill Root Alias reference.
	contract := "goal: ue.registration_verified\n" +
		"contextSchema: ca-context::urn:example:ca:invocation-context:v1\n" +
		"requestSchema: schemas/request.yaml\n" +
		"dependencies:\n" +
		"  ca-context: urn:example:ca:host-context\n" +
		"  ca: urn:example:ca:tools\n" +
		"  a: urn:example:test:a\n" +
		"capabilities:\n" +
		"  - example.registration-read\n"
	request = replaceDevelopmentSkillFileBytes(t, request, contractPath, []byte(contract))
	request.Files = append(request.Files, File{
		Path: "skills/verify-ue-registration/schemas/request.yaml",
		Bytes: []byte("$schema: https://json-schema.org/draft/2020-12/schema\n" +
			"$ref: a::urn:example:test:a-root:v1\n"),
	})
	sort.Slice(request.Files, func(i, j int) bool { return request.Files[i].Path < request.Files[j].Path })
	return request
}

func developmentSchemaChainPackages(aInnerRef string) []developmentTestPackage {
	return []developmentTestPackage{
		{
			ID: "urn:example:test:a", Publisher: "example-a-owner",
			Files: map[string][]byte{
				"a-root.schema.yaml": []byte("$schema: https://json-schema.org/draft/2020-12/schema\n" +
					"$id: urn:example:test:a-root:v1\n" +
					"type: object\nadditionalProperties: false\nproperties:\n  inner:\n    $ref: " + aInnerRef + "\n"),
			},
			Exports: []map[string]any{{"kind": "Schema", "id": "urn:example:test:a-root:v1", "path": "a-root.schema.yaml"}},
			Deps:    map[string]string{"b": "urn:example:test:b"},
		},
		{
			ID: "urn:example:test:b", Publisher: "example-b-owner",
			Files: map[string][]byte{
				"b-inner.schema.yaml": []byte("$schema: https://json-schema.org/draft/2020-12/schema\n" +
					"$id: urn:example:test:b-inner:v1\n" +
					"type: object\nadditionalProperties: false\nproperties:\n  leaf:\n    $ref: c::urn:example:test:c-leaf:v1\n"),
			},
			Exports: []map[string]any{{"kind": "Schema", "id": "urn:example:test:b-inner:v1", "path": "b-inner.schema.yaml"}},
			Deps:    map[string]string{"c": "urn:example:test:c"},
		},
		{
			ID: "urn:example:test:c", Publisher: "example-c-owner",
			Files: map[string][]byte{
				"c-leaf.schema.yaml": []byte("$schema: https://json-schema.org/draft/2020-12/schema\n" +
					"$id: urn:example:test:c-leaf:v1\ntype: string\nminLength: 1\n"),
			},
			Exports: []map[string]any{{"kind": "Schema", "id": "urn:example:test:c-leaf:v1", "path": "c-leaf.schema.yaml"}},
			Deps:    map[string]string{},
		},
	}
}

// TestDevelopmentSchemaResolverPackageChain proves the positive A -> B -> C
// case: every Package resolves its own alias against its own declared direct
// dependency, and the Skill Root Alias only applies to Skill-owned documents.
func TestDevelopmentSchemaResolverPackageChain(t *testing.T) {
	request := enableDevelopmentRequestSchemaRoot(t, loadDevelopmentFixture(t))
	request = addDevelopmentTestGraph(t, request, developmentSchemaChainPackages("b::urn:example:test:b-inner:v1"),
		map[string]string{"a": "urn:example:test:a"})
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("A -> B -> C Package-scoped Schema chain must build: %#v", result.Diagnostics)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}

// TestDevelopmentSchemaResolverRejectsUndeclaredGlobalID proves the negative
// case the review required: Package C is present in the Lock Closure, but
// Package A never declares a dependency on C, so A's global $id reference to C's
// Schema must fail even though the Schema is locked and unique.
func TestDevelopmentSchemaResolverRejectsUndeclaredGlobalID(t *testing.T) {
	// B still declares C, so C stays in the closure; A references C globally
	// without declaring a dependency on it.
	request := enableDevelopmentRequestSchemaRoot(t, loadDevelopmentFixture(t))
	request = addDevelopmentTestGraph(t, request, developmentSchemaChainPackages("urn:example:test:c-leaf:v1"),
		map[string]string{"a": "urn:example:test:a"})
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("Package A referenced a Schema outside its declared dependency closure and the Build succeeded")
	}
	if !hasDevelopmentCode(result.Diagnostics, "SCHEMA_MISSING") {
		t.Fatalf("diagnostics = %#v, want SCHEMA_MISSING", result.Diagnostics)
	}
}

// TestDevelopmentSchemaResolverRejectsUndeclaredAlias proves that an alias that
// the referencing Package does not declare cannot borrow a Skill Root Alias.
func TestDevelopmentSchemaResolverRejectsUndeclaredAlias(t *testing.T) {
	request := enableDevelopmentRequestSchemaRoot(t, loadDevelopmentFixture(t))
	request = addDevelopmentTestGraph(t, request, developmentSchemaChainPackages("missing-alias::urn:example:test:c-leaf:v1"),
		map[string]string{"a": "urn:example:test:a"})
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("an undeclared dependency alias inside a Package Schema must fail the Build")
	}
	if !hasDevelopmentCode(result.Diagnostics, "SCHEMA_MISSING") {
		t.Fatalf("diagnostics = %#v, want SCHEMA_MISSING", result.Diagnostics)
	}
}

// TestDevelopmentSchemaResolverRejectsWrongPackage proves that a declared alias
// only authorizes the exact Package it is bound to: pointing alias c at a
// Package that does not export the Schema fails.
func TestDevelopmentSchemaResolverRejectsWrongPackage(t *testing.T) {
	packages := developmentSchemaChainPackages("b::urn:example:test:b-inner:v1")
	// B declares a "d" alias bound to the types Package and references the leaf
	// Schema through it; the types Package does not export that Schema. C stays
	// reachable through the real "c" edge.
	packages[1].Files["b-inner.schema.yaml"] = []byte("$schema: https://json-schema.org/draft/2020-12/schema\n" +
		"$id: urn:example:test:b-inner:v1\n" +
		"type: object\nadditionalProperties: false\nproperties:\n  leaf:\n    $ref: d::urn:example:test:c-leaf:v1\n")
	packages[1].Deps = map[string]string{"c": "urn:example:test:c", "d": "urn:example:ca:types"}
	request := enableDevelopmentRequestSchemaRoot(t, loadDevelopmentFixture(t))
	request = addDevelopmentTestGraph(t, request, packages, map[string]string{"a": "urn:example:test:a"})
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("an alias bound to a Package that does not export the Schema must fail the Build")
	}
	if !hasDevelopmentCode(result.Diagnostics, "SCHEMA_MISSING") {
		t.Fatalf("diagnostics = %#v, want SCHEMA_MISSING", result.Diagnostics)
	}
}

// TestDevelopmentSchemaResolverRejectsIllegalRelativeRef proves a relative path
// inside a Package Schema is not silently interpreted against the global lock.
func TestDevelopmentSchemaResolverRejectsIllegalRelativeRef(t *testing.T) {
	request := enableDevelopmentRequestSchemaRoot(t, loadDevelopmentFixture(t))
	request = addDevelopmentTestGraph(t, request, developmentSchemaChainPackages("../c-leaf.schema.yaml"),
		map[string]string{"a": "urn:example:test:a"})
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("an illegal relative Schema $ref must fail the Build")
	}
	if !hasDevelopmentCode(result.Diagnostics, "SCHEMA_MISSING") {
		t.Fatalf("diagnostics = %#v, want SCHEMA_MISSING", result.Diagnostics)
	}
}

// TestDevelopmentSchemaResolverRejectsAmbiguousIdentity proves a Schema identity
// exported by more than one Package stays ambiguous instead of being guessed.
func TestDevelopmentSchemaResolverRejectsAmbiguousIdentity(t *testing.T) {
	packages := developmentSchemaChainPackages("b::urn:example:test:b-inner:v1")
	packages[2].Files["c-leaf-copy.schema.yaml"] = []byte("$schema: https://json-schema.org/draft/2020-12/schema\n" +
		"$id: urn:example:test:c-leaf:v1\ntype: string\n")
	packages[2].Exports = append(packages[2].Exports, map[string]any{"kind": "Schema", "id": "urn:example:test:c-leaf:v1", "path": "c-leaf-copy.schema.yaml"})
	request := enableDevelopmentRequestSchemaRoot(t, loadDevelopmentFixture(t))
	request = addDevelopmentTestGraph(t, request, packages, map[string]string{"a": "urn:example:test:a"})
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("a Schema identity exported twice must fail the Build")
	}
}
