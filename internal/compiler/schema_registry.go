package compiler

// WP4B: exact Schema registry, local $ref closure and San v6.0.3 adapter.
//
// The selected San engine is used only after the CA-SL G.4 allowlist has accepted
// a Schema. Resources are registered in memory; the loader rejects every remote
// or ambient fetch. This slice resolves the participating Schema closure and
// enforces the §11.5 / G.6 Request/Context root shape. F.5 assignability and CEL
// are separate WP4 slices.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const maxSchemaReferenceDepth = 64

type schemaResource struct {
	identity string
	external bool
	alias    string
	owner    string
	path     string
	node     *yaml.Node
}

type schemaRegistry struct {
	external map[string][]schemaResource
	local    map[string]schemaResource
}

func buildV1Alpha4SchemaRegistry(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult) schemaRegistry {
	registry := schemaRegistry{external: map[string][]schemaResource{}, local: map[string]schemaResource{}}
	if skill == nil {
		return registry
	}
	base := "skills/" + skill.Name + "/"
	for _, filename := range sortedStringMapKeys(skill.Files) {
		if !strings.HasPrefix(filename, base+"schemas/") || !(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
			continue
		}
		registry.local[filename] = schemaResource{identity: filename, path: filename, node: skill.Files[filename]}
	}
	snapshots := make(map[string]ResolvedDependency, len(identity.Closure))
	for _, alias := range identity.Closure {
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots[alias] = dependency
			}
		}
	}
	for _, alias := range sortedStringMapKeys(snapshots) {
		binding := identity.Bindings[alias]
		snapshot := snapshots[alias]
		files := append([]File(nil), snapshot.Files...)
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		for _, file := range files {
			if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") && !strings.HasSuffix(file.Path, ".json") {
				continue
			}
			root, err := yaml.Parse(file.Bytes)
			if err != nil {
				continue
			}
			export, ok := parseSchemaExport(root, file.Path)
			if !ok {
				continue
			}
			owner, ownershipDiagnostics := attributeExportOwner(export, binding, nil)
			if len(ownershipDiagnostics) > 0 || owner == "" {
				continue
			}
			resource := schemaResource{identity: export.ExportID, external: true, alias: alias, owner: owner, path: file.Path, node: root}
			registry.external[export.ExportID] = append(registry.external[export.ExportID], resource)
		}
	}
	return registry
}

// resolveReference resolves a frozen Schema $ref identity. It never guesses a
// target from a fragment, file name, prefix or declaration order.
func (registry schemaRegistry) resolveReference(ref string) (schemaResource, string) {
	alias := ""
	target := ref
	if index := strings.Index(ref, "::"); index > 0 {
		alias = ref[:index]
		target = ref[index+2:]
	}
	resources := registry.external[target]
	if alias != "" {
		filtered := make([]schemaResource, 0, len(resources))
		for _, resource := range resources {
			if resource.alias == alias {
				filtered = append(filtered, resource)
			}
		}
		resources = filtered
	}
	if len(resources) == 0 {
		return schemaResource{}, "missing"
	}
	if len(resources) > 1 {
		return schemaResource{}, "ambiguous"
	}
	return resources[0], ""
}

type schemaClosureResult struct {
	diagnostics []Diagnostic
	validated   map[string]bool
	visiting    map[string]bool
}

func validateV1Alpha4SchemaClosure(req BuildRequest, skill *V1Alpha4SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult) []Diagnostic {
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return []Diagnostic{}
	}
	registry := buildV1Alpha4SchemaRegistry(req, skill, identity)
	result := &schemaClosureResult{diagnostics: []Diagnostic{}, validated: map[string]bool{}, visiting: map[string]bool{}}
	contractPath := "skills/" + skill.Name + "/assurance/contract.yaml"

	if contract := skill.Contract; contract != nil {
		if request := contract.RequestSchema; request != nil {
			ref := scalar(request)
			schemaPath := "skills/" + skill.Name + "/" + ref
			if resource, ok := registry.local[schemaPath]; ok {
				result.validated[resource.identity] = true
				registry.walkSchemaClosure(resource, result)
			}
		}
		if context := contract.ContextSchema; context != nil {
			ref := scalar(context)
			resource, problem := registry.resolveReference(ref)
			switch problem {
			case "":
				registry.walkSchemaClosure(resource, result)
				diagnostics := validateContextSchemaRoot(resource, registry, expectedContextSchemaURI(skill))
				result.diagnostics = append(result.diagnostics, diagnostics...)
			default:
				// WP3B already diagnosed the missing/ambiguous external Schema
				// reference identity at this exact Skill source position.
				_ = contractPath
			}
		}
	}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind != "schema" {
			continue
		}
		resource, problem := registry.resolveReference(export.DependencyAlias + "::" + export.ExportID)
		switch problem {
		case "missing":
			result.diagnostics = append(result.diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "SCHEMA_MISSING", "", nil,
				"Referenced Schema $id is missing from the verified dependency closure: "+export.ExportID))
		case "ambiguous":
			result.diagnostics = append(result.diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "EXPORT_ID_AMBIGUOUS", "", nil,
				"Referenced Schema $id is ambiguous in the verified dependency closure: "+export.ExportID))
		default:
			registry.walkSchemaClosure(resource, result)
		}
	}
	if len(result.diagnostics) == 0 {
		result.diagnostics = append(result.diagnostics, compileSchemaClosure(registry, result.validated)...)
	}
	sortDiagnostics(result.diagnostics)
	return result.diagnostics
}

// walkSchemaClosure recursively validates the referenced Schema closure: G.4
// allowlist, exact reference resolution and acyclicity.
func (registry schemaRegistry) walkSchemaClosure(resource schemaResource, result *schemaClosureResult) {
	key := schemaResourceKey(resource)
	if result.validated[key] {
		return
	}
	if result.visiting[key] {
		result.diagnostics = append(result.diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "SCHEMA_INVALID", resourceDiagnosticPath(resource), nil,
			"Schema reference closure contains a cycle"))
		return
	}
	result.visiting[key] = true
	defer delete(result.visiting, key)
	result.validated[key] = true
	for _, issue := range validateG4Schema(resource.node) {
		result.diagnostics = append(result.diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "SCHEMA_INVALID", resourceDiagnosticPath(resource), resourceDiagnosticNode(resource, issue), issue.message))
	}
	for _, refNode := range schemaRefNodes(resource.node) {
		ref := scalar(refNode)
		target, problem := registry.resolveReference(ref)
		switch problem {
		case "missing":
			if resource.external || !strings.Contains(ref, "::") {
				result.diagnostics = append(result.diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "SCHEMA_MISSING", resourceDiagnosticPath(resource), refNode,
					"Schema $ref does not resolve to a trusted Schema $id: "+ref))
			}
		case "ambiguous":
			if resource.external || !strings.Contains(ref, "::") {
				result.diagnostics = append(result.diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "EXPORT_ID_AMBIGUOUS", resourceDiagnosticPath(resource), refNode,
					"Schema $ref resolves to more than one trusted Schema $id: "+ref))
			}
		default:
			registry.walkSchemaClosure(target, result)
		}
	}
}

func resourceDiagnosticPath(resource schemaResource) string {
	if resource.external {
		return ""
	}
	return resource.path
}

func resourceDiagnosticNode(resource schemaResource, issue g4Issue) *yaml.Node {
	if resource.external {
		return nil
	}
	return issue.node
}

func schemaResourceKey(resource schemaResource) string {
	if resource.external {
		return "external\x00" + resource.alias + "\x00" + resource.identity
	}
	return "local\x00" + resource.identity
}

func schemaRefNodes(root *yaml.Node) []*yaml.Node {
	nodes := make([]*yaml.Node, 0)
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		node = unalias(node)
		if node == nil {
			return
		}
		switch node.Kind {
		case yaml.MappingNode:
			for i := 0; i+1 < len(node.Content); i += 2 {
				key := unalias(node.Content[i])
				value := unalias(node.Content[i+1])
				if key != nil && key.Value == "$ref" {
					nodes = append(nodes, value)
				}
				walk(value)
			}
		case yaml.SequenceNode:
			for _, item := range node.Content {
				walk(item)
			}
		}
	}
	walk(root)
	return nodes
}

// resolveSchemaRoot follows the permitted root $ref chain to the effective root.
func (registry schemaRegistry) resolveSchemaRoot(resource schemaResource) (schemaResource, string) {
	seen := map[string]bool{}
	for depth := 0; depth < maxSchemaReferenceDepth; depth++ {
		ref := child(resource.node, "$ref")
		if ref == nil {
			return resource, ""
		}
		target, problem := registry.resolveReference(scalar(ref))
		if problem != "" {
			return schemaResource{}, problem
		}
		key := schemaResourceKey(target)
		if seen[key] {
			return schemaResource{}, "cycle"
		}
		seen[key] = true
		resource = target
	}
	return schemaResource{}, "cycle"
}

func validateContextSchemaRoot(resource schemaResource, registry schemaRegistry, expectedURI string) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	path := resourceDiagnosticPath(resource)
	effective, problem := registry.resolveSchemaRoot(resource)
	switch problem {
	case "missing":
		return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "SCHEMA_MISSING", path, nil, "Context Schema root $ref cannot be resolved in the trusted closure"))
	case "ambiguous":
		return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "EXPORT_ID_AMBIGUOUS", path, nil, "Context Schema root $ref is ambiguous in the trusted closure"))
	case "cycle":
		return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "SCHEMA_INVALID", path, nil, "Context Schema root $ref chain is cyclic"))
	}
	if expectedURI != "" {
		if actual := scalar(child(effective.node, "$id")); actual != expectedURI {
			return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "CONTEXT_SCHEMA_MISMATCH", path, nil,
				"Context Schema root $id does not match the Contract contextSchema identity"))
		}
	}
	if scalar(child(effective.node, "type")) != "object" {
		return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "SCHEMA_INVALID", path, nil, "Context Schema effective root must be an object Schema"))
	}
	if scalar(child(effective.node, "additionalProperties")) != "false" {
		return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "CONTEXT_REQUEST_COLLISION", path, nil, "Host Context root must be closed with additionalProperties: false"))
	}
	if propertyNames := child(effective.node, "properties"); propertyNames != nil && propertyNames.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(propertyNames.Content); i += 2 {
			if scalar(propertyNames.Content[i]) == "request" {
				return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "CONTEXT_REQUEST_COLLISION", path, nil, "Host Context root must not declare the reserved property request"))
			}
		}
	}
	for _, name := range sequence(child(effective.node, "required")) {
		if scalar(name) == "request" {
			return append(diagnostics, dependencyDiagnostic("error", "C_STATIC", "CONTEXT_REQUEST_COLLISION", path, nil, "Host Context root must not require the reserved property request"))
		}
	}
	return diagnostics
}

// schemaLoader denies every ambient resource fetch; only explicitly registered
// in-memory resources may be compiled.
type schemaLoader struct{}

func (schemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("Schema resource loading is disabled: %s", url)
}

func compileSchemaClosure(registry schemaRegistry, validated map[string]bool) []Diagnostic {
	resources := map[string]*yaml.Node{}
	for _, resourcesByID := range registry.external {
		if len(resourcesByID) != 1 {
			continue
		}
		resource := resourcesByID[0]
		if !validated[schemaResourceKey(resource)] {
			continue
		}
		resources[resource.identity] = resource.node
	}
	if len(resources) == 0 {
		return []Diagnostic{}
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(schemaLoader{})
	ids := make([]string, 0, len(resources))
	for id := range resources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		value, ok := yamlNodeToJSONValue(resources[id])
		if !ok {
			return []Diagnostic{dependencyDiagnostic("error", "C_STATIC", "SCHEMA_INVALID", "", nil, "Schema document is not a JSON value")}
		}
		if err := compiler.AddResource(id, value); err != nil {
			return []Diagnostic{dependencyDiagnostic("error", "C_STATIC", "SCHEMA_INVALID", "", nil, "Schema registration failed: "+err.Error())}
		}
	}
	diagnostics := make([]Diagnostic, 0)
	for _, id := range ids {
		if _, err := compiler.Compile(id); err != nil {
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "SCHEMA_INVALID", "", nil,
				"Schema reference closure is not resolvable: "+err.Error()))
		}
	}
	return diagnostics
}

func yamlNodeToJSONValue(node *yaml.Node) (any, bool) {
	node = unalias(node)
	if node == nil {
		return nil, false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			return node.Value, true
		case "!!int", "!!float":
			return json.Number(node.Value), true
		case "!!bool":
			return node.Value == "true", true
		case "!!null":
			return nil, true
		default:
			return nil, false
		}
	case yaml.SequenceNode:
		items := make([]any, 0, len(node.Content))
		for _, item := range node.Content {
			value, ok := yamlNodeToJSONValue(item)
			if !ok {
				return nil, false
			}
			items = append(items, value)
		}
		return items, true
	case yaml.MappingNode:
		object := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := scalar(unalias(node.Content[i]))
			if key == "" {
				return nil, false
			}
			value, ok := yamlNodeToJSONValue(node.Content[i+1])
			if !ok {
				return nil, false
			}
			object[key] = value
		}
		return object, true
	}
	return nil, false
}

// expectedContextSchemaURI returns the absolute Schema URI part of the frozen
// Contract contextSchema QualifiedSchemaId (alias::uri), or empty when undeclared
// or malformed (malformed identities are diagnosed by the contract slice).
func expectedContextSchemaURI(skill *V1Alpha4SkillAST) string {
	if skill == nil || skill.Contract == nil || skill.Contract.ContextSchema == nil {
		return ""
	}
	reference := scalar(unalias(skill.Contract.ContextSchema))
	index := strings.Index(reference, "::")
	if index <= 0 || index+2 >= len(reference) {
		return ""
	}
	return reference[index+2:]
}
