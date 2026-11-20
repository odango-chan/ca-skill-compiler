package compiler

// Development Build offline interface linking.
//
// Every package in the admitted lock closure is indexed only through its
// package.json export declarations. The linker resolves CallableCatalog, Schema,
// CapabilityCatalog and IngressCatalog surfaces, proves that every Skill
// reference names a declared export whose payload identity matches, validates
// capability membership through the declared direct dependency aliases, and walks
// the complete Schema $ref closure across the locked packages.
//
// It never searches an undeclared path, a different version or an unrelated
// package to fill a missing identity.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type developmentCapabilityMember struct {
	publisher  string
	alias      string
	callableID string
	kind       string
	effect     string
}

type developmentCapability struct {
	id         string
	owner      string
	packageKey DevelopmentPackageKey
	members    []developmentCapabilityMember
}

type developmentSchemaResource struct {
	packageKey DevelopmentPackageKey
	publisher  string
	path       string
	id         string
	node       *yaml.Node
}

type developmentCallable struct {
	packageKey      DevelopmentPackageKey
	publisher       string
	exportID        string
	kind            string
	effect          string
	path            string
	inputSchemaID   string
	querySchemaID   string
	outputSchemaID  string
	receiptSchemaID string
	provides        []evidenceProvide
	mutation        mutationContractFields
	reconcile       string
	mutationProblem string
	ownerOutcomes   []string
}

type developmentLinkScope struct {
	env            *developmentEnvironment
	schemas        map[string][]developmentSchemaResource
	callables      map[string][]developmentCallable
	capabilities   map[string][]developmentCapability
	packageAliases map[DevelopmentPackageKey]map[string]DevelopmentPackageKey
	exportKeys     map[DevelopmentPackageKey]map[string]developmentExport
}

// buildDevelopmentLinkScope indexes the exact declared export surface of every
// locked package. Duplicate identities across the scope are rejected because a
// reference could not resolve to one exact interface.
func buildDevelopmentLinkScope(env *developmentEnvironment) (*developmentLinkScope, []Diagnostic) {
	scope := &developmentLinkScope{
		env:            env,
		schemas:        map[string][]developmentSchemaResource{},
		callables:      map[string][]developmentCallable{},
		capabilities:   map[string][]developmentCapability{},
		packageAliases: map[DevelopmentPackageKey]map[string]DevelopmentPackageKey{},
		exportKeys:     map[DevelopmentPackageKey]map[string]developmentExport{},
	}
	var diagnostics []Diagnostic
	add := func(code, path, message string) {
		diagnostics = append(diagnostics, developmentDiagnostic(code, path, message))
	}
	for _, key := range sortedDevelopmentKeys(env.packages) {
		pkg := env.packages[key]
		aliases := env.dependencyGraph[key]
		scope.packageAliases[key] = aliases
		exports := map[string]developmentExport{}
		for _, export := range pkg.manifest.exports {
			exports[export.kind+"\x00"+export.id] = export
		}
		scope.exportKeys[key] = exports
		for _, export := range pkg.manifest.exports {
			raw := pkg.filesByName[export.path]
			switch export.kind {
			case "CallableCatalog":
				root, err := yaml.Parse(raw)
				if err != nil {
					add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, "declared CallableCatalog export is not valid YAML/JSON")
					continue
				}
				catalog, ok := parseToolCatalog(root, export.path)
				if !ok {
					add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, "declared CallableCatalog export is not a supported ToolCatalog")
					continue
				}
				for _, callable := range catalog.callables {
					if callable.ExportID != export.id {
						continue
					}
					record := developmentCallable{
						packageKey:      key,
						publisher:       pkg.manifest.publisher,
						exportID:        callable.ExportID,
						kind:            callable.ExportKind,
						effect:          callable.Effect,
						path:            export.path,
						inputSchemaID:   callable.InputSchemaID,
						querySchemaID:   callable.QueryContextSchemaID,
						outputSchemaID:  callable.OutputSchemaID,
						receiptSchemaID: callable.ReceiptSchemaID,
						provides:        callable.EvidenceProvides,
						mutation:        callable.MutationFields,
						reconcile:       callable.ReconcileTarget,
						mutationProblem: callable.MutationProblem,
						ownerOutcomes:   callable.OwnerOutcomes,
					}
					scope.callables[callable.ExportID] = append(scope.callables[callable.ExportID], record)
				}
			case "Schema":
				root, err := yaml.Parse(raw)
				if err != nil {
					add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, "declared Schema export is not valid YAML/JSON")
					continue
				}
				record, ok := parseSchemaExport(root, export.path)
				if !ok || record.ExportID != export.id {
					add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, "declared Schema export does not carry the declared $id")
					continue
				}
				scope.schemas[export.id] = append(scope.schemas[export.id], developmentSchemaResource{
					packageKey: key, publisher: pkg.manifest.publisher, path: export.path, id: export.id, node: root,
				})
			case "CapabilityCatalog":
				root, err := yaml.Parse(raw)
				if err != nil {
					add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, "declared CapabilityCatalog export is not valid YAML/JSON")
					continue
				}
				capabilities, problems := parseDevelopmentCapabilityDocument(root, export.path)
				for _, problem := range problems {
					add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, problem)
				}
				for _, capability := range capabilities {
					if capability.id != export.id {
						continue
					}
					record := developmentCapability{id: capability.id, owner: pkg.manifest.publisher, packageKey: key}
					for _, member := range capability.members {
						resolved, memberProblems := scope.resolveCapabilityMember(key, pkg, member)
						for _, problem := range memberProblems {
							add(developmentSnapshotInvalid, key.PackageID+"/"+export.path, problem)
						}
						if resolved != nil {
							record.members = append(record.members, *resolved)
						}
					}
					scope.capabilities[capability.id] = append(scope.capabilities[capability.id], record)
				}
			}
		}
	}
	// Ambiguous export identities are not resolvable by the Compiler.
	for _, id := range sortedStringMapKeys(scope.schemas) {
		if len(scope.schemas[id]) > 1 {
			add(developmentLinkInvalidCode(), "", "Schema $id is exported more than once in the locked closure: "+id)
		}
	}
	for _, id := range sortedStringMapKeys(scope.callables) {
		if len(scope.callables[id]) > 1 {
			add(developmentLinkInvalidCode(), "", "Callable ExportId is exported more than once in the locked closure: "+id)
		}
	}
	for _, id := range sortedStringMapKeys(scope.capabilities) {
		if len(scope.capabilities[id]) > 1 {
			add(developmentLinkInvalidCode(), "", "CapabilityId is exported more than once in the locked closure: "+id)
		}
	}
	sortDiagnostics(diagnostics)
	return scope, diagnostics
}

func developmentLinkInvalidCode() string {
	return developmentLockInvalid
}

// parseDevelopmentCapabilityDocument parses the package-scoped CapabilityCatalog
// logical payload: a capabilities list whose members are either same-package
// callables or an exact direct dependency alias plus a callable ID.
type developmentCapabilityDocument struct {
	id      string
	owner   string
	members []developmentCapabilityMemberRef
}

type developmentCapabilityMemberRef struct {
	alias      string
	callableID string
}

func parseDevelopmentCapabilityDocument(root *yaml.Node, path string) ([]developmentCapabilityDocument, []string) {
	var capabilities []developmentCapabilityDocument
	var problems []string
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, []string{"CapabilityCatalog document must be a mapping with a capabilities list"}
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if name := scalar(unalias(root.Content[i])); name != "capabilities" {
			problems = append(problems, "CapabilityCatalog document contains an unsupported field: "+name)
		}
	}
	list := unalias(child(root, "capabilities"))
	if list == nil || list.Kind != yaml.SequenceNode || len(list.Content) == 0 {
		return nil, []string{"CapabilityCatalog capabilities must be a non-empty ordered list"}
	}
	seenID := map[string]bool{}
	for index, node := range list.Content {
		node = unalias(node)
		label := "capabilities[" + itoa(index) + "]"
		if node == nil || node.Kind != yaml.MappingNode {
			problems = append(problems, label+" must be a mapping with id and members")
			continue
		}
		capability := developmentCapabilityDocument{id: scalar(child(node, "id"))}
		if !validExportID(capability.id) {
			problems = append(problems, label+".id must be a CapabilityId")
			continue
		}
		if seenID[capability.id] {
			problems = append(problems, label+".id is duplicated in one CapabilityCatalog document")
			continue
		}
		seenID[capability.id] = true
		members := unalias(child(node, "members"))
		if members == nil || members.Kind != yaml.SequenceNode || len(members.Content) == 0 {
			problems = append(problems, label+".members must be a non-empty ordered list")
			continue
		}
		seenMember := map[string]bool{}
		for memberIndex, memberNode := range members.Content {
			memberNode = unalias(memberNode)
			memberLabel := label + ".members[" + itoa(memberIndex) + "]"
			if memberNode == nil || memberNode.Kind != yaml.MappingNode {
				problems = append(problems, memberLabel+" must be a mapping with callableId and optional dependencyAlias")
				continue
			}
			for i := 0; i+1 < len(memberNode.Content); i += 2 {
				name := scalar(unalias(memberNode.Content[i]))
				if name != "callableId" && name != "dependencyAlias" {
					problems = append(problems, memberLabel+" contains an unsupported field: "+name)
				}
			}
			member := developmentCapabilityMemberRef{
				alias:      scalar(child(memberNode, "dependencyAlias")),
				callableID: scalar(child(memberNode, "callableId")),
			}
			if !validExportID(member.callableID) {
				problems = append(problems, memberLabel+".callableId must be an ExportId")
				continue
			}
			if member.alias != "" && !validDependencyID(member.alias) {
				problems = append(problems, memberLabel+".dependencyAlias must be a declared direct dependency alias")
				continue
			}
			key := member.alias + "\x00" + member.callableID
			if seenMember[key] {
				problems = append(problems, memberLabel+" duplicates the same callable member")
				continue
			}
			seenMember[key] = true
			capability.members = append(capability.members, member)
		}
		capabilities = append(capabilities, capability)
	}
	return capabilities, problems
}

// resolveCapabilityMember resolves one declared CapabilityCatalog member to the
// exact package and Callable export it names.
func (scope *developmentLinkScope) resolveCapabilityMember(ownerKey DevelopmentPackageKey, owner *developmentPackage, member developmentCapabilityMemberRef) (*developmentCapabilityMember, []string) {
	targetKey := ownerKey
	if member.alias != "" {
		declared, exists := scope.packageAliases[ownerKey][member.alias]
		if !exists {
			return nil, []string{"Capability member uses a dependency alias that the package manifest does not declare: " + member.alias}
		}
		targetKey = declared
	}
	target, exists := scope.env.packages[targetKey]
	if !exists {
		return nil, []string{"Capability member resolves to a package outside the locked closure: " + member.callableID}
	}
	callable := scope.declaredCallable(targetKey, member.callableID)
	if callable == nil {
		return nil, []string{"Capability member does not resolve to a declared Callable export: " + member.callableID}
	}
	return &developmentCapabilityMember{
		publisher:  target.manifest.publisher,
		alias:      member.alias,
		callableID: callable.exportID,
		kind:       callable.kind,
		effect:     callable.effect,
	}, nil
}

func targetCallablePackage(scope *developmentLinkScope, member developmentCapabilityMemberRef) string {
	records := scope.callables[member.callableID]
	if len(records) == 1 {
		return records[0].packageKey.PackageID
	}
	return ""
}

func (scope *developmentLinkScope) declaresCallable(key DevelopmentPackageKey, id string) bool {
	_, exists := scope.exportKeys[key]["CallableCatalog\x00"+id]
	return exists
}

func (scope *developmentLinkScope) declaredCallable(key DevelopmentPackageKey, id string) *developmentCallable {
	if !scope.declaresCallable(key, id) {
		return nil
	}
	for _, record := range scope.callables[id] {
		if record.packageKey == key {
			copyRecord := record
			return &copyRecord
		}
	}
	return nil
}

// validateDevelopmentReferenceSurface proves every qualified Skill reference is
// a declared export of the exact locked root package and that the export's
// payload identity matches. It deliberately runs before the inherited pipeline so
// an undeclared export cannot be linked merely because it exists in a file.
func validateDevelopmentReferenceSurface(env *developmentEnvironment, skill *SkillAST, scope *developmentLinkScope) []Diagnostic {
	if skill == nil {
		return []Diagnostic{}
	}
	var diagnostics []Diagnostic
	add := func(code, path, message string) {
		diagnostics = append(diagnostics, developmentDiagnostic(code, path, message))
	}
	for _, reference := range collectDependencyReferences(skill) {
		key, exists := env.rootByAlias[reference.alias]
		if !exists {
			// The inherited dependency-identity pass reports the exact alias
			// failure; do not cascade a second identity here.
			continue
		}
		kind, id := reference.exportKind, reference.exportID
		switch kind {
		case "operation", "observation":
			if !scope.declaresCallable(key, id) {
				add("EXPORT_ID_MISSING", reference.path, "Skill reference does not name a declared CallableCatalog export: "+reference.alias+"::"+id)
				continue
			}
			callable := scope.declaredCallable(key, id)
			if callable == nil || callable.kind != kind {
				add("EXPORT_ID_MISSING", reference.path, "Declared callable export has a different kind than the Skill reference: "+reference.alias+"::"+id)
				continue
			}
			if kind == "observation" && callable.effect != "READ" {
				add("EFFECT_MISMATCH", reference.path, "Observation reference does not name a READ callable: "+reference.alias+"::"+id)
			}
		case "schema":
			if !scope.declaresSchema(key, id) {
				add("SCHEMA_MISSING", reference.path, "Skill reference does not name a declared Schema export: "+reference.alias+"::"+id)
			}
		case "evidenceGuarantee":
			// Evidence descriptors are supplied only by the optional Assurance
			// Export and are checked by the deferred-obligation pass.
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

func (scope *developmentLinkScope) declaresSchema(key DevelopmentPackageKey, id string) bool {
	_, exists := scope.exportKeys[key]["Schema\x00"+id]
	return exists
}

// validateDevelopmentSchemaClosure walks every Schema root that the selected
// Skill and its linked callables depend on, resolving each $ref exactly once
// across the locked closure. Missing or ambiguous references fail the build.
func validateDevelopmentSchemaClosure(env *developmentEnvironment, skill *SkillAST, scope *developmentLinkScope) []Diagnostic {
	if skill == nil || skill.Contract == nil {
		return []Diagnostic{}
	}
	var diagnostics []Diagnostic
	add := func(path, message string) {
		diagnostics = append(diagnostics, developmentDiagnostic("SCHEMA_MISSING", path, message))
	}
	visited := map[DevelopmentPackageKey]map[string]bool{}
	var walk func(resource developmentSchemaResource, chain []string)
	walk = func(resource developmentSchemaResource, chain []string) {
		if visited[resource.packageKey] == nil {
			visited[resource.packageKey] = map[string]bool{}
		}
		if visited[resource.packageKey][resource.id] {
			return
		}
		visited[resource.packageKey][resource.id] = true
		if len(chain) > maxSchemaReferenceDepth {
			add(resource.path, "Schema $ref closure exceeds the maximum depth at "+resource.id)
			return
		}
		for _, ref := range collectSchemaRefs(resource.node) {
			target, problem := scope.resolveDevelopmentSchemaRef(resource, ref)
			switch problem {
			case "":
				walk(target, append(chain, resource.id))
			case "local":
				// Same-document fragment or relative pointer: the shared Schema
				// engine validates the local closure.
			default:
				add(resource.path, "Schema $ref "+ref+" in "+resource.id+" is "+problem+" in the locked closure")
			}
		}
	}
	contract := skill.Contract
	if context := contract.ContextSchema; context != nil {
		alias, id, ok := splitQualifiedReference(unalias(context))
		if ok {
			if key, exists := env.rootByAlias[alias]; exists {
				if resource, found := scope.schemaByPackage(key, id); found {
					walk(resource, nil)
				}
			}
		}
	}
	if request := contract.RequestSchema; request != nil {
		ref := scalar(unalias(request))
		schemaPath := "skills/" + skill.Name + "/" + ref
		if node, exists := skill.Files[schemaPath]; exists {
			walk(developmentSchemaResource{packageKey: DevelopmentPackageKey{}, publisher: "", path: schemaPath, id: schemaPath, node: node}, nil)
		}
	}
	// Every declared callable used by the Skill must have its complete input,
	// query, output, receipt, commit-proof, reconcile and recoverable Schema
	// closure present.
	for _, export := range usedDevelopmentCallables(skill, env, scope) {
		for _, id := range []string{export.inputSchemaID, export.querySchemaID, export.outputSchemaID, export.receiptSchemaID,
			export.mutation.CommitProofSchemaID, export.mutation.ReconcileProofSchemaID, export.mutation.RecoverableSchemaID} {
			if id == "" {
				continue
			}
			if !scope.hasSchema(id) {
				add("", "Callable "+export.exportID+" references a Schema that is missing from the locked closure: "+id)
			}
		}
		// Walk every schema this callable references so nested $refs are proven.
		for _, id := range []string{export.inputSchemaID, export.querySchemaID, export.outputSchemaID, export.receiptSchemaID,
			export.mutation.CommitProofSchemaID, export.mutation.ReconcileProofSchemaID, export.mutation.RecoverableSchemaID} {
			if id == "" {
				continue
			}
			for _, resource := range scope.schemas[id] {
				walk(resource, nil)
			}
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

func (scope *developmentLinkScope) schemaByPackage(key DevelopmentPackageKey, id string) (developmentSchemaResource, bool) {
	for _, resource := range scope.schemas[id] {
		if resource.packageKey == key {
			return resource, true
		}
	}
	return developmentSchemaResource{}, false
}

func (scope *developmentLinkScope) hasSchema(id string) bool {
	return len(scope.schemas[id]) == 1
}

func (scope *developmentLinkScope) resolveDevelopmentSchemaRef(owner developmentSchemaResource, ref string) (developmentSchemaResource, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return developmentSchemaResource{}, "empty"
	}
	if strings.HasPrefix(ref, "#") {
		return developmentSchemaResource{}, "local"
	}
	alias := ""
	target := ref
	if index := strings.Index(ref, "::"); index > 0 {
		alias = ref[:index]
		target = ref[index+2:]
	}
	if alias != "" {
		key, exists := scope.env.rootByAlias[alias]
		if !exists {
			return developmentSchemaResource{}, "missing"
		}
		resource, found := scope.schemaByPackage(key, target)
		if !found {
			return developmentSchemaResource{}, "missing"
		}
		return resource, ""
	}
	// An absolute Schema URI resolves within the owner package first, then in the
	// owner's locked dependency closure. It never falls back to an unrelated root.
	if resource, found := scope.schemaByPackage(owner.packageKey, target); found {
		return resource, ""
	}
	if owner.packageKey.PackageID != "" {
		for _, alias := range sortedStringMapKeys(scope.packageAliases[owner.packageKey]) {
			if resource, found := scope.schemaByPackage(scope.packageAliases[owner.packageKey][alias], target); found {
				return resource, ""
			}
		}
	}
	if len(scope.schemas[target]) == 1 && owner.packageKey.PackageID == "" {
		return scope.schemas[target][0], ""
	}
	if len(scope.schemas[target]) > 1 {
		return developmentSchemaResource{}, "ambiguous"
	}
	return developmentSchemaResource{}, "missing"
}

func collectSchemaRefs(root *yaml.Node) []string {
	var refs []string
	seen := map[string]bool{}
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
				if key != nil && key.Value == "$ref" && value != nil && value.Kind == yaml.ScalarNode {
					ref := value.Value
					if !seen[ref] {
						seen[ref] = true
						refs = append(refs, ref)
					}
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
	sort.Strings(refs)
	return refs
}

func usedDevelopmentCallables(skill *SkillAST, env *developmentEnvironment, scope *developmentLinkScope) []developmentCallable {
	if skill == nil {
		return nil
	}
	seen := map[string]bool{}
	var callables []developmentCallable
	for _, reference := range collectDependencyReferences(skill) {
		if reference.exportKind != "operation" && reference.exportKind != "observation" {
			continue
		}
		key, exists := env.rootByAlias[reference.alias]
		if !exists {
			continue
		}
		callable := scope.declaredCallable(key, reference.exportID)
		if callable == nil {
			continue
		}
		identity := developmentKeyString(key) + "\x00" + callable.exportID
		if seen[identity] {
			continue
		}
		seen[identity] = true
		callables = append(callables, *callable)
	}
	sort.Slice(callables, func(i, j int) bool {
		if callables[i].exportID != callables[j].exportID {
			return callables[i].exportID < callables[j].exportID
		}
		return developmentKeyString(callables[i].packageKey) < developmentKeyString(callables[j].packageKey)
	})
	return callables
}

// developmentLegacyRequest converts the admitted Development scope into the
// internal single-Core request shape. Root aliases keep their Contract alias;
// locked transitive packages become deterministic internal aliases so the shared
// Schema registry can resolve the complete closure. Declared package identity is
// bound as the interface identity only; trust is never asserted.
func (env *developmentEnvironment) legacyRequest() (BuildRequest, []syntheticDependencyAlias) {
	req := BuildRequest{
		APIVersion:     BuildRequestAPIVersion,
		LanguageTarget: V1Alpha5Target,
		Files:          append([]File(nil), env.sourceFiles...),
		Targets:        append([]string(nil), env.targets...),
	}
	context := &TrustedBuildContext{}
	roots := map[DevelopmentPackageKey]string{}
	for _, root := range env.roots {
		roots[root.Key] = root.Alias
	}
	keys := sortedDevelopmentKeys(env.packages)
	synthetic := make([]syntheticDependencyAlias, 0)
	index := 0
	for _, key := range keys {
		if !env.reachable[key] {
			continue
		}
		pkg := env.packages[key]
		alias := ""
		if rootAlias, exists := roots[key]; exists {
			alias = rootAlias
		} else {
			alias = newSyntheticDependencyAlias(index)
			index++
			synthetic = append(synthetic, syntheticDependencyAlias{Alias: alias, Key: key})
		}
		req.ResolvedDependencies = append(req.ResolvedDependencies, ResolvedDependency{
			DependencyID: alias,
			Digest:       key.SnapshotDigest,
			Files:        append([]File(nil), pkg.files...),
		})
		binding := TrustedDependencyBinding{
			DependencyID:     alias,
			Digest:           key.SnapshotDigest,
			StablePackageID:  key.PackageID,
			AuthorizedOwners: []string{pkg.manifest.publisher},
		}
		for _, export := range pkg.manifest.exports {
			switch export.kind {
			case "CallableCatalog":
				for _, callable := range env.callablesByExport(key, export.id) {
					binding.ExportOwners = append(binding.ExportOwners, ExportOwner{
						ExportKind: callable.kind, ExportID: callable.exportID, Owner: pkg.manifest.publisher,
					})
				}
			case "Schema":
				binding.ExportOwners = append(binding.ExportOwners, ExportOwner{
					ExportKind: "schema", ExportID: export.id, Owner: pkg.manifest.publisher,
				})
			case "IngressCatalog":
				binding.ExportOwners = append(binding.ExportOwners, ExportOwner{
					ExportKind: "ingressOperation", ExportID: export.id, Owner: pkg.manifest.publisher,
				})
			}
		}
		sort.Slice(binding.ExportOwners, func(i, j int) bool {
			a, b := binding.ExportOwners[i], binding.ExportOwners[j]
			if a.ExportKind != b.ExportKind {
				return a.ExportKind < b.ExportKind
			}
			if a.ExportID != b.ExportID {
				return a.ExportID < b.ExportID
			}
			return a.Owner < b.Owner
		})
		context.DependencyBindings = append(context.DependencyBindings, binding)
		// Manifest-declared artifact/Schema bindings of an IngressCatalog export
		// are carried through to the frozen WP1/WP5/WP6 link context. "self"
		// means the export's own package; a direct dependency alias is resolved
		// through the package's lock edges.
		for _, export := range pkg.manifest.exports {
			if export.kind != "IngressCatalog" {
				continue
			}
			for _, artifact := range export.artifactBindings {
				target := alias
				if artifact.dependencyID != "self" {
					if edge, exists := env.dependencyGraph[key][artifact.dependencyID]; exists {
						if edgeAlias, ok := legacyAliasForPackage(roots, synthetic, edge); ok {
							target = edgeAlias
						}
					}
				}
				context.ArtifactBindings = append(context.ArtifactBindings, TrustedArtifactBinding{
					ArtifactRef: artifact.artifactRef, DependencyID: target,
					SnapshotPath: artifact.snapshotPath, Dialect: artifact.dialect,
				})
			}
			for _, schema := range export.schemaBindings {
				context.SchemaBindings = append(context.SchemaBindings, TrustedSchemaBinding{
					ExportID: schema.exportID, ArtifactRef: schema.artifactRef, Pointer: schema.pointer,
				})
			}
		}
	}
	if len(context.ArtifactBindings) > 0 {
		context.SnapshotBaseURI = "https://development.invalid/snapshots/"
	}
	req.TrustedBuildContext = context
	return req, synthetic
}

type syntheticDependencyAlias struct {
	Alias string
	Key   DevelopmentPackageKey
}

func newSyntheticDependencyAlias(index int) string {
	const digits = "0123456789"
	value := "locked"
	if index == 0 {
		return value + "_0"
	}
	suffix := ""
	for index > 0 {
		suffix = string(digits[index%10]) + suffix
		index /= 10
	}
	return value + "_" + suffix
}

func legacyAliasForPackage(roots map[DevelopmentPackageKey]string, synthetic []syntheticDependencyAlias, key DevelopmentPackageKey) (string, bool) {
	if alias, exists := roots[key]; exists {
		return alias, true
	}
	for _, alias := range synthetic {
		if alias.Key == key {
			return alias.Alias, true
		}
	}
	return "", false
}

// callablesByExport lists the declared callables of one package/export id.
func (env *developmentEnvironment) callablesByExport(key DevelopmentPackageKey, id string) []developmentCallable {
	pkg := env.packages[key]
	if pkg == nil {
		return nil
	}
	for _, export := range pkg.manifest.exports {
		if export.kind != "CallableCatalog" || export.id != id {
			continue
		}
		raw, exists := pkg.filesByName[export.path]
		if !exists {
			return nil
		}
		root, err := yaml.Parse(raw)
		if err != nil {
			return nil
		}
		catalog, ok := parseToolCatalog(root, export.path)
		if !ok {
			return nil
		}
		var callables []developmentCallable
		for _, callable := range catalog.callables {
			if callable.ExportID != id {
				continue
			}
			callables = append(callables, developmentCallable{
				packageKey: key, publisher: pkg.manifest.publisher, exportID: callable.ExportID,
				kind: callable.ExportKind, effect: callable.Effect, path: export.path,
				inputSchemaID: callable.InputSchemaID, querySchemaID: callable.QueryContextSchemaID,
				outputSchemaID: callable.OutputSchemaID, receiptSchemaID: callable.ReceiptSchemaID,
				provides: callable.EvidenceProvides, mutation: callable.MutationFields,
				reconcile: callable.ReconcileTarget, mutationProblem: callable.MutationProblem,
			})
		}
		return callables
	}
	return nil
}

// extendIdentity appends the locked transitive packages to the validated
// dependency closure so the shared Schema registry can resolve cross-package
// Schema references. The extra aliases are internal and never exposed.
func (env *developmentEnvironment) extendIdentity(identity dependencyIdentityResult, synthetic []syntheticDependencyAlias) dependencyIdentityResult {
	for _, entry := range synthetic {
		identity.Closure = append(identity.Closure, entry.Alias)
		identity.Identities = append(identity.Identities, dependencyIdentity{
			Alias:           entry.Alias,
			MeasuredDigest:  entry.Key.SnapshotDigest,
			StablePackageID: entry.Key.PackageID,
		})
		pkg := env.packages[entry.Key]
		binding := TrustedDependencyBinding{
			DependencyID:     entry.Alias,
			Digest:           entry.Key.SnapshotDigest,
			StablePackageID:  entry.Key.PackageID,
			AuthorizedOwners: []string{pkg.manifest.publisher},
		}
		for _, export := range pkg.manifest.exports {
			switch export.kind {
			case "Schema":
				binding.ExportOwners = append(binding.ExportOwners, ExportOwner{ExportKind: "schema", ExportID: export.id, Owner: pkg.manifest.publisher})
			case "CallableCatalog":
				for _, callable := range env.callablesByExport(entry.Key, export.id) {
					binding.ExportOwners = append(binding.ExportOwners, ExportOwner{ExportKind: callable.kind, ExportID: callable.exportID, Owner: pkg.manifest.publisher})
				}
			case "IngressCatalog":
				binding.ExportOwners = append(binding.ExportOwners, ExportOwner{ExportKind: "ingressOperation", ExportID: export.id, Owner: pkg.manifest.publisher})
			}
		}
		identity.Bindings[entry.Alias] = binding
	}
	sort.Strings(identity.Closure)
	return identity
}
