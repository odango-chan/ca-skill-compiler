package compiler

import (
	"fmt"
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type linkedCallable struct {
	Inventory        InspectedCallable
	Raw              *yaml.Node
	SchemaReferences []dependencySchemaReference
	ReconcileCallable string
}

type linkedSchema struct {
	Inventory InspectedSchema
	Raw       *yaml.Node
	Anchors   map[string]*yaml.Node
}

type linkedDependency struct {
	DependencyID string
	PackageID    string
	Callables    map[string]linkedCallable
	Schemas      map[string]linkedSchema
	ReconcileOnly map[string]bool
}

type dependencyLinkResult struct {
	Snapshots   map[string]*linkedDependency
	Diagnostics []Diagnostic
	Callables   []InspectedCallable
	Schemas     []InspectedSchema
}

type callableUse struct {
	Alias      string
	ID         string
	Kind       string
	Owner      string
	Effect     string
	SourcePath string
	Node       *yaml.Node
}

type schemaUse struct {
	Alias string
	ID    string
	Node  *yaml.Node
}

type qualifiedReference struct {
	Alias    string
	Identity string
	Fragment string
}

func inspectDependencies(req BuildRequest) DependencyInspection {
	diagnostics := validateBuildRequest(req)
	if len(diagnostics) != 0 {
		return DependencyInspection{
			APIVersion:  DependencyInspectionAPIVersion,
			Diagnostics: diagnostics,
			Callables:   []InspectedCallable{},
			Schemas:     []InspectedSchema{},
		}
	}
	var bindings []TrustedDependencyBinding
	if req.TrustedBuildContext != nil {
		bindings = req.TrustedBuildContext.DependencyBindings
	}
	linked := linkDependencySnapshots(req.ResolvedDependencies, bindings, nil)
	sortDiagnostics(linked.Diagnostics)
	if linked.Diagnostics == nil {
		linked.Diagnostics = []Diagnostic{}
	}
	if linked.Callables == nil {
		linked.Callables = []InspectedCallable{}
	}
	if linked.Schemas == nil {
		linked.Schemas = []InspectedSchema{}
	}
	return DependencyInspection{
		APIVersion:  DependencyInspectionAPIVersion,
		Diagnostics: linked.Diagnostics,
		Callables:   linked.Callables,
		Schemas:     linked.Schemas,
	}
}

func linkSelectedSkillDependencies(source *V1Alpha4SkillAST, req BuildRequest, add func(string, *yaml.Node, string, string)) (dependencyLinkResult, []string, []callableUse) {
	result := dependencyLinkResult{Snapshots: make(map[string]*linkedDependency)}
	if source == nil || source.Contract == nil {
		return result, nil, nil
	}

	declared := make(map[string]string)
	dependencyNode := unalias(source.Contract.Dependencies)
	if dependencyNode != nil {
		if dependencyNode.Kind != yaml.MappingNode {
			add("", dependencyNode, "DEPENDENCY_REQUIREMENT_INVALID", "Contract.dependencies must be a mapping from Dependency Alias to Stable Package ID")
		} else {
			for i := 0; i+1 < len(dependencyNode.Content); i += 2 {
				key := unalias(dependencyNode.Content[i])
				value := unalias(dependencyNode.Content[i+1])
				if key == nil || value == nil || key.Kind != yaml.ScalarNode || key.Tag != "!!str" ||
					!validDependencyID(key.Value) || value.Kind != yaml.ScalarNode || value.Tag != "!!str" || !validStablePackageID(value.Value) {
					add("", dependencyNode.Content[i], "DEPENDENCY_REQUIREMENT_INVALID", "Each Contract.dependencies entry must map a valid Alias to an absolute Stable Package ID")
					continue
				}
				if _, exists := declared[key.Value]; exists {
					add("", key, "DEPENDENCY_DUPLICATE", "Duplicate Dependency Alias in Contract.dependencies: "+key.Value)
					continue
				}
				declared[key.Value] = value.Value
			}
		}
	}

	used := make(map[string]bool)
	var callableRefs []callableUse
	var schemaRefs []schemaUse
	collectAlias := func(alias string, node *yaml.Node) bool {
		if !validDependencyID(alias) {
			add("", node, "QUALIFIED_REFERENCE_INVALID", "Qualified reference contains an invalid Dependency Alias")
			return false
		}
		if _, exists := declared[alias]; !exists {
			add("", node, "DEPENDENCY_ALIAS_MISSING", "Qualified reference uses an Alias not declared by Contract.dependencies: "+alias)
			return false
		}
		used[alias] = true
		return true
	}

	if source.Contract.ContextSchema != nil {
		ref, ok := parseQualifiedSchemaReference(scalar(source.Contract.ContextSchema))
		if !ok {
			add("", source.Contract.ContextSchema, "QUALIFIED_REFERENCE_INVALID", "Contract.contextSchema must be an exact Alias::absolute-Schema-ID reference")
		} else if collectAlias(ref.Alias, source.Contract.ContextSchema) {
			schemaRefs = append(schemaRefs, schemaUse{Alias: ref.Alias, ID: ref.Identity, Node: source.Contract.ContextSchema})
		}
	}

	for _, nodeID := range sortedStringMapKeys(source.Procedure.ByID) {
		node := source.Procedure.ByID[nodeID]
		if node.Operation != nil && node.Operation.Tool != nil {
			ref, ok := parseQualifiedCallableReference(scalar(node.Operation.Tool))
			if !ok {
				add("", node.Operation.Tool, "QUALIFIED_REFERENCE_INVALID", "operation.tool must be an exact Alias::ExportId reference")
			} else if collectAlias(ref.Alias, node.Operation.Tool) {
				callableRefs = append(callableRefs, callableUse{Alias: ref.Alias, ID: ref.Identity, Kind: "operation", Node: node.Operation.Tool})
			}
		}
		if node.Observation != nil && node.Observation.Tool != nil {
			ref, ok := parseQualifiedCallableReference(scalar(node.Observation.Tool))
			if !ok {
				add("", node.Observation.Tool, "QUALIFIED_REFERENCE_INVALID", "observation.tool must be an exact Alias::ExportId reference")
			} else if collectAlias(ref.Alias, node.Observation.Tool) {
				callableRefs = append(callableRefs, callableUse{Alias: ref.Alias, ID: ref.Identity, Kind: "observation", Node: node.Observation.Tool})
			}
		}
	}

	if source.Predicates != nil {
		for _, predicateID := range sortedStringMapKeys(source.Predicates.Definitions) {
			definition := source.Predicates.Definitions[predicateID]
			for _, parameter := range sequence(definition.Parameters) {
				if parameter == nil || parameter.Kind != yaml.MappingNode || len(parameter.Content) != 2 {
					continue
				}
				typeNode := unalias(parameter.Content[1])
				typeName := scalar(typeNode)
				if isPredicatePrimitive(typeName) {
					continue
				}
				ref, ok := parseQualifiedSchemaReference(typeName)
				if !ok {
					add("", typeNode, "QUALIFIED_REFERENCE_INVALID", "Predicate parameter type must be a supported primitive or exact Alias::Schema-ID reference")
					continue
				}
				if collectAlias(ref.Alias, typeNode) {
					schemaRefs = append(schemaRefs, schemaUse{Alias: ref.Alias, ID: ref.Identity, Node: typeNode})
				}
			}
			// Evidence Guarantee exports have no frozen physical descriptor format yet.
			// Their Alias is still part of the Skill dependency closure; WP7 owns descriptor linking.
			requires := child(definition.Raw, "evidence")
			for _, guaranteeList := range mappingValues(child(requires, "requires")) {
				for _, guaranteeNode := range sequence(guaranteeList) {
					ref, ok := parseQualifiedCallableReference(scalar(guaranteeNode))
					if !ok {
						add("", guaranteeNode, "QUALIFIED_REFERENCE_INVALID", "Evidence guarantee reference must be an exact Alias::ExportId")
						continue
					}
					if collectAlias(ref.Alias, guaranteeNode) {
						add("", guaranteeNode, "EVIDENCE_GUARANTEE_LINK_INVALID", "Evidence Guarantee descriptor linking is deferred until the actual Owner-published descriptor format is available")
					}
				}
			}
		}
	}

	var selected []ResolvedDependency
	expectedPackages := make(map[string]string)
	for _, alias := range sortedStringMapKeys(used) {
		expectedPackages[alias] = declared[alias]
		found := false
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				selected = append(selected, dependency)
				found = true
			}
		}
		if !found {
			add("", nil, "DEPENDENCY_SNAPSHOT_MISSING", "Contract uses Dependency Alias without a matching immutable snapshot: "+alias)
		}
	}
	for _, alias := range sortedStringMapKeys(declared) {
		if !used[alias] {
			add("", source.Contract.Dependencies, "UNUSED_DEPENDENCY", "Declared dependency is not referenced by the selected Skill source: "+alias)
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "warning", Phase: "C_WARNING", Code: "UNUSED_DEPENDENCY",
				Path: "", Message: "Declared dependency is not referenced by the selected Skill source: " + alias,
			})
		}
	}

	var bindings []TrustedDependencyBinding
	if req.TrustedBuildContext != nil {
		bindings = req.TrustedBuildContext.DependencyBindings
	}
	linked := linkDependencySnapshots(selected, bindings, expectedPackages)
	result.Snapshots = linked.Snapshots
	result.Diagnostics = append(result.Diagnostics, linked.Diagnostics...)

	for i := range callableRefs {
		ref := &callableRefs[i]
		snapshot := result.Snapshots[ref.Alias]
		if snapshot == nil {
			continue
		}
		callable, exists := snapshot.Callables[ref.ID]
		if !exists {
			add("", ref.Node, "EXPORT_ID_MISSING", "Callable ExportId does not exist in the exact trusted dependency snapshot: "+ref.Alias+"::"+ref.ID)
			continue
		}
		if callable.Inventory.Kind != ref.Kind {
			add("", ref.Node, "EXPORT_ID_MISSING", "Callable ExportId exists but its kind does not match the Procedure stage: "+ref.Alias+"::"+ref.ID)
			continue
		}
		if snapshot.ReconcileOnly[ref.ID] {
			add("", ref.Node, "RECONCILE_CONTRACT_INVALID", "A dedicated Reconcile READ callable cannot be invoked as an ordinary operation.tool")
			continue
		}
		ref.Owner = callable.Inventory.Owner
		ref.Effect = callable.Inventory.Effect
		ref.SourcePath = callable.Inventory.SourcePath
	}
	for _, ref := range schemaRefs {
		snapshot := result.Snapshots[ref.Alias]
		if snapshot == nil {
			continue
		}
		if _, exists := snapshot.Schemas[ref.ID]; !exists {
			add("", ref.Node, "SCHEMA_MISSING", "Exact Schema ID does not exist in the trusted dependency snapshot: "+ref.Alias+"::"+ref.ID)
		}
	}
	return result, sortedStringMapKeys(used), callableRefs
}

func linkDependencySnapshots(dependencies []ResolvedDependency, bindings []TrustedDependencyBinding, expectedPackages map[string]string) dependencyLinkResult {
	result := dependencyLinkResult{Snapshots: make(map[string]*linkedDependency)}
	for _, dependency := range dependencies {
		var binding *TrustedDependencyBinding
		for i := range bindings {
			candidate := &bindings[i]
			if candidate.DependencyID == dependency.DependencyID && candidate.Digest == dependency.Digest {
				binding = candidate
				break
			}
		}
		if binding == nil {
			result.Diagnostics = append(result.Diagnostics, linkDiagnostic("DEPENDENCY_IDENTITY_UNVERIFIED", "", 0, 0, "No trusted identity binding matches the exact dependencyId and measured snapshot digest: "+dependency.DependencyID))
			continue
		}
		if expected, required := expectedPackages[dependency.DependencyID]; required && binding.StablePackageID != expected {
			result.Diagnostics = append(result.Diagnostics, linkDiagnostic("DEPENDENCY_ID_MISMATCH", "", 0, 0, "Trusted Stable Package ID differs from Contract.dependencies for Alias: "+dependency.DependencyID))
			continue
		}
		linked, diagnostics, valid := parseDependencySnapshot(dependency, *binding)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !valid {
			continue
		}
		result.Snapshots[dependency.DependencyID] = linked
	}
	for alias, snapshot := range result.Snapshots {
		diagnostics, valid := validateDependencySchemaReferences(snapshot)
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if !valid {
			delete(result.Snapshots, alias)
		}
	}
	for _, snapshot := range result.Snapshots {
		for _, callable := range snapshot.Callables {
			result.Callables = append(result.Callables, callable.Inventory)
		}
		for _, schema := range snapshot.Schemas {
			result.Schemas = append(result.Schemas, schema.Inventory)
		}
	}
	sortInspectedCallables(result.Callables)
	sortInspectedSchemas(result.Schemas)
	return result
}

func linkDiagnostic(code, path string, line, column int, message string) Diagnostic {
	return Diagnostic{
		Severity: "error", Phase: "C_STATIC", Code: code,
		Path: limitDiagnosticPath(path), Line: line, Column: column,
		Message: limitDiagnosticMessage(message),
	}
}

func parseQualifiedCallableReference(value string) (qualifiedReference, bool) {
	parts := strings.Split(value, "::")
	if len(parts) != 2 || !validDependencyID(parts[0]) || !validExportID(parts[1]) {
		return qualifiedReference{}, false
	}
	return qualifiedReference{Alias: parts[0], Identity: parts[1]}, true
}

func parseQualifiedSchemaReference(value string) (qualifiedReference, bool) {
	index := strings.Index(value, "::")
	if index <= 0 || index+2 >= len(value) {
		return qualifiedReference{}, false
	}
	alias := value[:index]
	identity := value[index+2:]
	if !validDependencyID(alias) || !validSchemaReferenceURI(identity) {
		return qualifiedReference{}, false
	}
	return qualifiedReference{Alias: alias, Identity: identity}, true
}

func validExportID(value string) bool {
	if value == "" || len(value) > maxExportIDRunes {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" || !asciiLetter(part[0]) {
			return false
		}
		for i := 1; i < len(part); i++ {
			c := part[i]
			if !asciiLetter(c) && !(c >= '0' && c <= '9') && c != '_' && c != '-' {
				return false
			}
		}
	}
	return true
}

func validStablePackageID(value string) bool {
	return validAbsoluteURI(value) && !strings.Contains(value, "#")
}

func validSchemaReferenceURI(value string) bool {
	base := strings.SplitN(value, "#", 2)[0]
	return validAbsoluteURI(base)
}

func isPredicatePrimitive(value string) bool {
	switch value {
	case "string", "integer", "number", "boolean":
		return true
	default:
		return false
	}
}

func sortedStringMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortInspectedCallables(values []InspectedCallable) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		for _, pair := range [][2]string{{a.DependencyID, b.DependencyID}, {a.Owner, b.Owner}, {a.Kind, b.Kind}, {a.ID, b.ID}, {a.SourcePath, b.SourcePath}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
}

func sortInspectedSchemas(values []InspectedSchema) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		for _, pair := range [][2]string{{a.DependencyID, b.DependencyID}, {a.Owner, b.Owner}, {a.ID, b.ID}, {a.SourcePath, b.SourcePath}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
}

func mappingValues(node *yaml.Node) []*yaml.Node {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	values := make([]*yaml.Node, 0, len(node.Content)/2)
	for i := 1; i < len(node.Content); i += 2 {
		values = append(values, unalias(node.Content[i]))
	}
	return values
}

func sortedDiagnosticsCopy(values []Diagnostic) []Diagnostic {
	copyValues := append([]Diagnostic(nil), values...)
	sortDiagnostics(copyValues)
	return copyValues
}

func dependencyErrorMessage(alias, detail string) string {
	return fmt.Sprintf("Dependency Alias %s: %s", alias, detail)
}
