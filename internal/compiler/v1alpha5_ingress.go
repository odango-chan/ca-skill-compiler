package compiler

// CA-SL v1alpha5 WP3: trusted dependency scope, trusted Ingress Operation
// discovery and exact Wait event-selector linking.
//
// Authoritative clauses: ca-skills #61 Appendix L.3.1/L.3.2/L.4.1/L.4.2/L.11 and
// the #62 Integration Ingress Operation contract §6/§7/§10/§12/§13 plus the
// versioned Ingress diagnostic registry.
//
// The linking slice never guesses: no name search, no newest-export selection, no
// untrusted catalog use and no fallback EventCatalog. An IngressCatalog snapshot
// produces trusted exports only when the verified TrustedDependencyBinding
// declares the exact ingressOperation export identity with the matching Owner.
// Every other catalog is suppressed without Ingress diagnostics, as required by
// #62 §13 prerequisite suppression.

import (
	"fmt"
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"github.com/odango-chan/ca-skill-compiler/internal/ingresscatalog"
)

const (
	// v1alpha5IngressCatalogAPIVersion / Kind identify one package-scoped
	// Integration Ingress Operation catalog inside a dependency snapshot.
	v1alpha5IngressCatalogAPIVersion   = "connectionagent.io/ingresscatalog/v1alpha1"
	v1alpha5IngressCatalogKind         = "IngressCatalog"
	v1alpha5ExportKindIngressOperation = "ingressOperation"
)

// v1alpha5StaticDiagnostic builds one C_STATIC error diagnostic anchored at an
// optional normalized YAML node.
func v1alpha5StaticDiagnostic(path string, node *yaml.Node, code, message string) Diagnostic {
	line, column := 0, 0
	if node != nil {
		line, column = node.Line, node.Column
	}
	return Diagnostic{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     code,
		Path:     limitDiagnosticPath(path),
		Line:     line,
		Column:   column,
		Message:  limitDiagnosticMessage(message),
	}
}

// v1alpha5LinkContext holds the validated trusted artifact/Schema context used by
// the WP3 linking passes. resolution is nil when the request declares no
// v1alpha5 artifact bindings.
type v1alpha5LinkContext struct {
	resolution  *ingresscatalog.ResolutionContext
	schemaRoots map[string]ingresscatalog.RootRef
}

// v1alpha5Dependency is one verified Contract dependency snapshot. PackageID is
// the trusted Stable Package ID from the matching TrustedDependencyBinding.
type v1alpha5Dependency struct {
	Alias     string
	PackageID string
	Snapshot  ResolvedDependency
	Binding   TrustedDependencyBinding
}

// v1alpha5TrustedIngressExport is one trusted Ingress Operation export together
// with its exact dependency/Catalog provenance.
type v1alpha5TrustedIngressExport struct {
	DependencyAlias string
	CatalogOwner    string
	SnapshotPath    string
	Operation       ingresscatalog.Operation
}

// validateV1Alpha5DependencyContext is the strict v1alpha5 envelope admission for
// the dependency snapshot set and the trusted build context. It validates the
// shapes needed to link trusted exports; the v1alpha4 envelope validation is
// untouched and explicitly rejects the v1alpha5-only context fields.
func validateV1Alpha5DependencyContext(req BuildRequest) []Diagnostic {
	var diagnostics []Diagnostic
	add := func(code, path, message string) {
		diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(path, nil, code, message))
	}

	if len(req.ResolvedDependencies) > maxResolvedDependencies {
		add("INPUT_LIMIT", "", "BuildRequest.resolvedDependencies exceeds the dependency-snapshot count limit")
	}
	dependencyFiles := make(map[string]map[string]bool, len(req.ResolvedDependencies))
	seenDependency := make(map[string]bool, len(req.ResolvedDependencies))
	totalBytes := 0
	for _, dependency := range req.ResolvedDependencies {
		if !validDependencyID(dependency.DependencyID) {
			add("DEPENDENCY_REQUIREMENT_INVALID", "", "ResolvedDependency.dependencyId must be a valid dependency alias")
			continue
		}
		if seenDependency[dependency.DependencyID] {
			add("DEPENDENCY_DUPLICATE", "", "Duplicate resolved dependencyId: "+dependency.DependencyID)
			continue
		}
		seenDependency[dependency.DependencyID] = true
		if len(dependency.Files) == 0 {
			add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Resolved dependency snapshot must contain at least one file: "+dependency.DependencyID)
		}
		paths := make(map[string]bool, len(dependency.Files))
		filesValid := true
		seenPath := make(map[string]bool, len(dependency.Files))
		seenFolded := make(map[string]bool, len(dependency.Files))
		for _, file := range dependency.Files {
			if file.Bytes == nil {
				add("REQUEST_INVALID", file.Path, "File.bytes is required and must be a base64 string on the JSON wire")
				filesValid = false
			}
			if !validRelativeSnapshotPath(file.Path) {
				add("PACKAGE_PATH_INVALID", file.Path, "Dependency snapshot file path must be a normalized UTF-8 POSIX relative path")
				filesValid = false
			}
			if seenPath[file.Path] {
				add("PACKAGE_PATH_INVALID", file.Path, "Duplicate file path within dependency "+dependency.DependencyID)
				filesValid = false
			} else if seenFolded[strings.ToLower(file.Path)] {
				add("PACKAGE_PATH_INVALID", file.Path, "Case-folded file path collision within dependency "+dependency.DependencyID)
				filesValid = false
			}
			seenPath[file.Path] = true
			seenFolded[strings.ToLower(file.Path)] = true
			if len(file.Bytes) > maxFileSize {
				add("INPUT_LIMIT", file.Path, "File exceeds the 2 MiB raw byte limit")
				filesValid = false
			}
			totalBytes += len(file.Bytes)
			paths[file.Path] = true
		}
		dependencyFiles[dependency.DependencyID] = paths
		if !validDigest(dependency.Digest) {
			add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Resolved dependency has an invalid SHA-256 digest: "+dependency.DependencyID)
		} else if filesValid && len(dependency.Files) > 0 {
			actual, err := ComputeDependencySnapshotDigest(dependency.Files)
			if err != nil || actual != dependency.Digest {
				add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Resolved dependency snapshot digest does not match its original file bytes: "+dependency.DependencyID)
			}
		}
	}
	if totalBytes > maxTotal {
		add("INPUT_LIMIT", "", "BuildRequest dependency byte total exceeds the 16 MiB aggregate raw byte limit")
	}

	ctx := req.TrustedBuildContext
	if ctx == nil {
		sortDiagnostics(diagnostics)
		return sortedV1Alpha5Diagnostics(diagnostics)
	}
	if len(ctx.DependencyBindings) > maxResolvedDependencies || len(ctx.ArtifactBindings) > maxFiles || len(ctx.SchemaBindings) > maxFiles {
		add("INPUT_LIMIT", "", "TrustedBuildContext exceeds the binding or authorization count limit")
	}
	seenBinding := make(map[string]bool, len(ctx.DependencyBindings))
	for _, binding := range ctx.DependencyBindings {
		if !validDependencyID(binding.DependencyID) || !validDigest(binding.Digest) || !validURIIdentity(binding.StablePackageID) || len(binding.AuthorizedOwners) == 0 || len(binding.AuthorizedOwners) > 128 {
			add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Trusted dependency binding is incomplete or malformed: "+binding.DependencyID)
		}
		if seenBinding[binding.DependencyID] {
			add("DEPENDENCY_DUPLICATE", "", "Duplicate trusted dependency binding: "+binding.DependencyID)
		}
		seenBinding[binding.DependencyID] = true
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == binding.DependencyID && binding.Digest != dependency.Digest {
				add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Trusted dependency binding digest differs from the measured snapshot: "+binding.DependencyID)
			}
		}
		owners := make(map[string]bool, len(binding.AuthorizedOwners))
		for _, owner := range binding.AuthorizedOwners {
			if owner == "" || owners[owner] {
				add("DEPENDENCY_IDENTITY_UNVERIFIED", "", "Trusted dependency binding contains an invalid or duplicate authorized Owner: "+binding.DependencyID)
			}
			owners[owner] = true
		}
		if len(binding.ExportOwners) > 512 {
			add("INPUT_LIMIT", "", "Trusted dependency binding exceeds 512 export-owner records")
		}
		exports := make(map[string]bool, len(binding.ExportOwners))
		for _, export := range binding.ExportOwners {
			key := export.ExportKind + "\x00" + export.ExportID
			if !validExportKindV1Alpha5(export.ExportKind) || export.ExportID == "" || utf8RuneCount(export.ExportID) > maxExportIDRunes || export.Owner == "" || utf8RuneCount(export.Owner) > maxOwnerIDRunes || exports[key] {
				add("DEPENDENCY_EXPORT_OWNER_UNVERIFIED", "", "Trusted dependency binding contains an invalid or ambiguous export-owner record: "+binding.DependencyID)
			}
			exports[key] = true
		}
	}
	validateV1Alpha5ArtifactBindings(ctx, dependencyFiles, add)

	sortDiagnostics(diagnostics)
	return sortedV1Alpha5Diagnostics(diagnostics)
}

// validExportKindV1Alpha5 is the v1alpha5 trusted export-kind set: the inherited
// kinds plus the package-scoped ingressOperation export kind. The v1alpha4 set is
// deliberately not widened.
func validExportKindV1Alpha5(value string) bool {
	return validExportKind(value) || value == v1alpha5ExportKindIngressOperation
}

func utf8RuneCount(value string) int {
	return len([]rune(value))
}

// validateV1Alpha5ArtifactBindings validates the v1alpha5-only trusted artifact
// and Schema bindings exactly: unique artifact identities, existing dependency
// and snapshot path, declared dialect enum, existing artifactRef for every Schema
// binding, canonical Schema root pointer and a required snapshot base URI
// whenever any binding is present. No digest is accepted from the caller.
func validateV1Alpha5ArtifactBindings(ctx *TrustedBuildContext, dependencyFiles map[string]map[string]bool, add func(code, path, message string)) {
	if ctx == nil {
		return
	}
	artifactRefs := make(map[string]bool, len(ctx.ArtifactBindings))
	boundPaths := make(map[string]bool, len(ctx.ArtifactBindings))
	for _, binding := range ctx.ArtifactBindings {
		if binding.ArtifactRef == "" || !validURIIdentity(binding.ArtifactRef) {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings artifactRef must be a valid absolute URI identity")
		}
		if artifactRefs[binding.ArtifactRef] {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings contains a duplicate artifactRef: "+binding.ArtifactRef)
		}
		artifactRefs[binding.ArtifactRef] = true
		if !validDependencyID(binding.DependencyID) {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings dependencyId is not a valid dependency alias")
		}
		paths, exists := dependencyFiles[binding.DependencyID]
		if !exists {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings references a dependency that is not a declared resolved dependency: "+binding.DependencyID)
		} else if !paths[binding.SnapshotPath] {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings snapshotPath does not exist in dependency "+binding.DependencyID+": "+binding.SnapshotPath)
		}
		if binding.Dialect != string(ingresscatalog.DialectOpenAPI30) && binding.Dialect != string(ingresscatalog.DialectJSONSchema202012) {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings dialect must be openapi-3.0 or json-schema-2020-12")
		}
		pathKey := binding.DependencyID + "\x00" + binding.SnapshotPath
		if boundPaths[pathKey] {
			add("REQUEST_INVALID", "", "TrustedBuildContext.artifactBindings binds the same dependency snapshot path more than once")
		}
		boundPaths[pathKey] = true
	}
	exportIDs := make(map[string]bool, len(ctx.SchemaBindings))
	for _, binding := range ctx.SchemaBindings {
		if !validStablePackageID(binding.ExportID) {
			add("REQUEST_INVALID", "", "TrustedBuildContext.schemaBindings exportId must be a valid absolute Schema identity")
		}
		if exportIDs[binding.ExportID] {
			add("REQUEST_INVALID", "", "TrustedBuildContext.schemaBindings contains a duplicate exportId: "+binding.ExportID)
		}
		exportIDs[binding.ExportID] = true
		if !artifactRefs[binding.ArtifactRef] {
			add("REQUEST_INVALID", "", "TrustedBuildContext.schemaBindings artifactRef does not identify a declared trusted artifact: "+binding.ArtifactRef)
		}
		if !validCanonicalRootPointer(binding.Pointer) {
			add("REQUEST_INVALID", "", "TrustedBuildContext.schemaBindings pointer must be a canonical JSON Pointer root '#', '#/...'")
		}
	}
	if len(ctx.ArtifactBindings)+len(ctx.SchemaBindings) > 0 && ctx.SnapshotBaseURI == "" {
		add("REQUEST_INVALID", "", "TrustedBuildContext.snapshotBaseURI is required when trusted artifact or Schema bindings are present")
	}
	if ctx.SnapshotBaseURI != "" && !validURIText(ctx.SnapshotBaseURI) {
		add("REQUEST_INVALID", "", "TrustedBuildContext.snapshotBaseURI must be a non-empty URI without whitespace or control characters")
	}
}

// validCanonicalRootPointer accepts exactly "#" or "#/..." with valid RFC 6901
// escaping. Percent-encoding is not part of a canonical artifact pointer.
func validCanonicalRootPointer(pointer string) bool {
	if pointer == "#" {
		return true
	}
	if !strings.HasPrefix(pointer, "#/") {
		return false
	}
	for _, token := range strings.Split(pointer[2:], "/") {
		for index := 0; index < len(token); index++ {
			if token[index] != '~' {
				continue
			}
			if index+1 >= len(token) || (token[index+1] != '0' && token[index+1] != '1') {
				return false
			}
			index++
		}
	}
	return true
}

func validURIText(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		c := value[index]
		if c <= 0x20 || c == 0x7f || c == '\\' {
			return false
		}
	}
	return true
}

// buildV1Alpha5LinkContext builds the trusted artifact/Schema resolution context
// from the exact dependency snapshot bytes. WP1 recomputes every artifact digest
// from those raw bytes; nothing here trusts a caller-supplied digest.
func buildV1Alpha5LinkContext(req BuildRequest) (*v1alpha5LinkContext, []Diagnostic) {
	context := &v1alpha5LinkContext{schemaRoots: map[string]ingresscatalog.RootRef{}}
	ctx := req.TrustedBuildContext
	if ctx == nil || (len(ctx.ArtifactBindings) == 0 && len(ctx.SchemaBindings) == 0) {
		return context, nil
	}
	byDependency := make(map[string]ResolvedDependency, len(req.ResolvedDependencies))
	for _, dependency := range req.ResolvedDependencies {
		if _, exists := byDependency[dependency.DependencyID]; !exists {
			byDependency[dependency.DependencyID] = dependency
		}
	}
	var diagnostics []Diagnostic
	artifacts := make([]ingresscatalog.SnapshotArtifact, 0, len(ctx.ArtifactBindings))
	for _, binding := range ctx.ArtifactBindings {
		dependency, exists := byDependency[binding.DependencyID]
		if !exists {
			continue
		}
		raw, found := v1alpha5DependencyFileBytes(dependency, binding.SnapshotPath)
		if !found {
			continue
		}
		artifacts = append(artifacts, ingresscatalog.NewSnapshotArtifactFromRaw(
			binding.ArtifactRef, binding.SnapshotPath, ingresscatalog.Dialect(binding.Dialect), raw))
	}
	resolution, err := ingresscatalog.NewResolutionContext(ctx.SnapshotBaseURI, artifacts)
	if err != nil {
		code := "DEPENDENCY_IDENTITY_UNVERIFIED"
		if errorCode, ok := ingresscatalog.CodeOf(err); ok && errorCode == ingresscatalog.ErrSnapshotBaseURI {
			code = "REQUEST_INVALID"
		}
		diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, code,
			"Trusted v1alpha5 artifact/Schema context is invalid: "+err.Error()))
		return context, diagnostics
	}
	context.resolution = resolution
	for _, binding := range ctx.SchemaBindings {
		context.schemaRoots[binding.ExportID] = ingresscatalog.RootRef{ArtifactRef: binding.ArtifactRef, Pointer: binding.Pointer}
	}
	return context, diagnostics
}

func v1alpha5DependencyFileBytes(dependency ResolvedDependency, snapshotPath string) ([]byte, bool) {
	for _, file := range dependency.Files {
		if file.Path == snapshotPath {
			return file.Bytes, true
		}
	}
	return nil, false
}

// v1alpha5UsedDependencyAliases returns the Dependency Aliases referenced by the
// Procedure's operation/observation tool values. It only inspects the exact
// qualified-reference positions; no arbitrary YAML string is searched.
func v1alpha5UsedDependencyAliases(skill *V1Alpha5SkillAST) map[string]bool {
	used := map[string]bool{}
	if skill == nil || skill.Procedure == nil {
		return used
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		for _, stage := range []*yaml.Node{node.Operation, node.Observation} {
			if alias, _, ok := splitQualifiedReference(child(stage, "tool")); ok {
				used[alias] = true
			}
		}
	}
	return used
}

// collectV1Alpha5DependencyScope verifies the Contract-declared dependency
// snapshots and their trusted Stable Package IDs. Any identity failure is a
// prerequisite failure: the caller suppresses the dependent Ingress/Skill
// diagnostics instead of cascading.
func collectV1Alpha5DependencyScope(req BuildRequest, declared map[string]declaredDependency, used map[string]bool) ([]v1alpha5Dependency, []Diagnostic) {
	var diagnostics []Diagnostic
	scope := make([]v1alpha5Dependency, 0, len(declared))
	aliases := make([]string, 0, len(declared))
	for alias := range declared {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		entry := declared[alias]
		if !entry.requirement {
			// The Contract dependency value itself was already rejected.
			continue
		}
		var snapshots []ResolvedDependency
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots = append(snapshots, dependency)
			}
		}
		if len(snapshots) == 0 {
			if used[alias] {
				diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, "DEPENDENCY_SNAPSHOT_MISSING",
					"No supplied immutable DependencySnapshot has the used Dependency Alias as its dependencyId: "+alias))
			}
			continue
		}
		if len(snapshots) > 1 {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, "DEPENDENCY_DUPLICATE",
				"Multiple DependencySnapshots claim the same Dependency Alias: "+alias))
			continue
		}
		snapshot := snapshots[0]
		measuredDigest, err := ComputeDependencySnapshotDigest(snapshot.Files)
		if err != nil || measuredDigest != snapshot.Digest {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, "DEPENDENCY_IDENTITY_UNVERIFIED",
				"Used DependencySnapshot bytes do not match the supplied digest: "+alias))
			continue
		}
		var bindings []TrustedDependencyBinding
		if req.TrustedBuildContext != nil {
			for _, binding := range req.TrustedBuildContext.DependencyBindings {
				if binding.DependencyID == alias && binding.Digest == measuredDigest {
					bindings = append(bindings, binding)
				}
			}
		}
		if len(bindings) == 0 {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, "DEPENDENCY_IDENTITY_UNVERIFIED",
				"No trusted dependency binding matches the used Dependency Alias and measured snapshot digest: "+alias))
			continue
		}
		if len(bindings) > 1 {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, "DEPENDENCY_DUPLICATE",
				"Multiple trusted dependency bindings match the same Alias and measured snapshot digest: "+alias))
			continue
		}
		binding := bindings[0]
		if binding.StablePackageID != entry.packageID {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil, "DEPENDENCY_ID_MISMATCH",
				"Trusted Stable Package ID differs from the Contract dependency value for Alias "+alias))
			continue
		}
		scope = append(scope, v1alpha5Dependency{
			Alias: alias, PackageID: binding.StablePackageID, Snapshot: snapshot, Binding: binding,
		})
	}
	return scope, diagnostics
}

// discoverV1Alpha5IngressExports admits only the IngressCatalogs whose exact
// ingressOperation export identities are declared by the verified trusted
// dependency binding with the matching Owner. Untrusted catalogs are suppressed
// completely: they produce neither exports nor Ingress diagnostics.
func discoverV1Alpha5IngressExports(scope []v1alpha5Dependency, link *v1alpha5LinkContext) ([]v1alpha5TrustedIngressExport, []Diagnostic) {
	var exports []v1alpha5TrustedIngressExport
	var diagnostics []Diagnostic
	trustedSnapshot := ingresscatalog.NewTrustedSnapshot(link.resolution, link.schemaRoots)
	for _, dependency := range scope {
		files := append([]File(nil), dependency.Snapshot.Files...)
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		for _, file := range files {
			if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") && !strings.HasSuffix(file.Path, ".json") {
				continue
			}
			root, err := yaml.Parse(file.Bytes)
			if err != nil {
				continue
			}
			if scalar(child(root, "apiVersion")) != v1alpha5IngressCatalogAPIVersion || scalar(child(root, "kind")) != v1alpha5IngressCatalogKind {
				continue
			}
			owner := scalar(child(child(root, "metadata"), "owner"))
			operationIDs := v1alpha5CatalogOperationIDs(root)
			if !v1alpha5CatalogTrusted(dependency.Binding, owner, operationIDs) {
				continue
			}
			admission := ingresscatalog.AdmitCatalog(ingresscatalog.CatalogInput{
				Bytes:           file.Bytes,
				DependencyAlias: dependency.Alias,
				SnapshotPath:    file.Path,
			}, trustedSnapshot)
			diagnostics = append(diagnostics, v1alpha5IngressAdmissionDiagnostics(admission.Diagnostics)...)
			if !admission.Valid() || admission.Catalog == nil {
				continue
			}
			for _, operation := range admission.Catalog.Operations {
				exports = append(exports, v1alpha5TrustedIngressExport{
					DependencyAlias: dependency.Alias,
					CatalogOwner:    owner,
					SnapshotPath:    file.Path,
					Operation:       operation,
				})
			}
		}
	}
	sort.SliceStable(exports, func(i, j int) bool {
		left, right := exports[i], exports[j]
		if left.DependencyAlias != right.DependencyAlias {
			return left.DependencyAlias < right.DependencyAlias
		}
		if left.SnapshotPath != right.SnapshotPath {
			return left.SnapshotPath < right.SnapshotPath
		}
		return left.Operation.ID < right.Operation.ID
	})
	return exports, diagnostics
}

func v1alpha5CatalogOperationIDs(root *yaml.Node) []string {
	ids := make([]string, 0)
	for _, operation := range sequence(child(child(root, "spec"), "operations")) {
		id := scalar(child(operation, "id"))
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// v1alpha5CatalogTrusted applies the trusted-only discovery rule: the Catalog
// Owner must be an authorized Owner of the verified snapshot binding, and every
// declared Operation must carry an exact trusted ingressOperation export-owner
// record with the same Owner. A single unverified Operation makes the whole
// Catalog untrusted.
func v1alpha5CatalogTrusted(binding TrustedDependencyBinding, owner string, operationIDs []string) bool {
	if owner == "" || len(operationIDs) == 0 {
		return false
	}
	authorized := false
	for _, candidate := range binding.AuthorizedOwners {
		if candidate == owner {
			authorized = true
			break
		}
	}
	if !authorized {
		return false
	}
	for _, operationID := range operationIDs {
		trusted := false
		for _, export := range binding.ExportOwners {
			if export.ExportKind == v1alpha5ExportKindIngressOperation && export.ExportID == operationID && export.Owner == owner {
				trusted = true
				break
			}
		}
		if !trusted {
			return false
		}
	}
	return true
}

// v1alpha5IngressAdmissionDiagnostics converts the WP1 package-local Ingress
// diagnostics into compiler diagnostics. The primary source location is the
// exact dependency snapshot path; the dependency alias and JSON Pointer remain
// available through the WP1 Admission projection until the WP7 diagnostic
// envelope slice carries the full registry record.
func v1alpha5IngressAdmissionDiagnostics(diagnostics []ingresscatalog.Diagnostic) []Diagnostic {
	converted := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		converted = append(converted, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     diagnostic.Code,
			Path:     limitDiagnosticPath(diagnostic.SourceLocation.SnapshotPath),
			Message:  limitDiagnosticMessage(diagnostic.Message),
		})
	}
	return converted
}

// v1alpha5IngressCorrelationDiagnostic builds the Ingress-owned
// INGRESS_CORRELATION_INVALID diagnostic for an invalid export-side correlation
// or stale-delivery declaration.
func v1alpha5IngressCorrelationDiagnostic(export v1alpha5TrustedIngressExport, fieldPath string) Diagnostic {
	message, _ := ingresscatalog.RenderIngressMessage(ingresscatalog.CodeIngressCorrelationInvalid,
		"{exportId}", export.Operation.ID, "{fieldPath}", fieldPath)
	return Diagnostic{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     ingresscatalog.CodeIngressCorrelationInvalid,
		Path:     limitDiagnosticPath(export.SnapshotPath),
		Message:  limitDiagnosticMessage(message),
	}
}

// v1alpha5SelectorString renders the exact three-part selector for the normative
// L.11.1 message placeholder. No case-folding or escaping is applied.
func v1alpha5SelectorString(event V1Alpha5IngressEventAST) string {
	return event.Source + "/" + event.Service + "/" + event.Operation
}

// resolveV1Alpha5IngressExport resolves the exact wait.event selector against the
// trusted export set. It returns the unique export, or the exact not-found /
// ambiguous classification.
func resolveV1Alpha5IngressExport(exports []v1alpha5TrustedIngressExport, event V1Alpha5IngressEventAST) (v1alpha5TrustedIngressExport, bool, bool) {
	var matches []v1alpha5TrustedIngressExport
	for _, export := range exports {
		if export.Operation.Event.Source == event.Source &&
			export.Operation.Event.Service == event.Service &&
			export.Operation.Event.Operation == event.Operation {
			matches = append(matches, export)
		}
	}
	switch len(matches) {
	case 0:
		return v1alpha5TrustedIngressExport{}, false, false
	case 1:
		return matches[0], false, true
	default:
		return v1alpha5TrustedIngressExport{}, true, false
	}
}

// v1alpha5WaitSelectorDiagnostics reports the exact L.11.1 not-found/ambiguous
// diagnostic for a Wait node whose selector does not resolve to exactly one
// trusted Ingress Operation export.
func v1alpha5WaitSelectorDiagnostics(procedurePath string, nodeID string, wait *V1Alpha5WaitAST, ambiguous bool) []Diagnostic {
	selector := v1alpha5SelectorString(wait.Event)
	code := "WAIT_INGRESS_EXPORT_NOT_FOUND"
	message := fmt.Sprintf("Node '%s' wait selector '%s' does not resolve to a trusted Ingress Operation export.", nodeID, selector)
	if ambiguous {
		code = "WAIT_INGRESS_EXPORT_AMBIGUOUS"
		message = fmt.Sprintf("Node '%s' wait selector '%s' resolves to more than one trusted Ingress Operation export.", nodeID, selector)
	}
	return []Diagnostic{v1alpha5StaticDiagnostic(procedurePath, wait.Event.Raw, code, message)}
}
