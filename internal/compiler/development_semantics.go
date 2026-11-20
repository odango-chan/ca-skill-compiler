package compiler

// Development Build semantic integration.
//
// The single v1alpha5 pipeline runs unchanged for every statically decidable
// rule. This file only adds the Development-profile branches that consume the
// package-scoped interface scope: capability membership from CapabilityCatalog
// exports, Evidence Descriptor deferral, and typed obligation recording.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type developmentSemantics struct {
	env       *developmentEnvironment
	scope     *developmentLinkScope
	synthetic []syntheticDependencyAlias
	assurance *developmentAssuranceIndex
	collector *developmentObligationCollector

	// deferredEvidence records guarantee references whose external authority is
	// not statically decidable, keyed by root alias + "::" + exportID.
	linkRecords []DevelopmentLinkRecord

	deferredEvidence map[string]bool
	// descriptorPresent records guarantee references with a supplied descriptor
	// that passed every static check, keyed by root alias + "::" + exportID.
	descriptorPresent map[string]bool
}

func newDevelopmentSemantics(env *developmentEnvironment, scope *developmentLinkScope, synthetic []syntheticDependencyAlias, assurance *developmentAssuranceIndex, collector *developmentObligationCollector) *developmentSemantics {
	return &developmentSemantics{
		env: env, scope: scope, synthetic: synthetic, assurance: assurance, collector: collector,
		deferredEvidence:  map[string]bool{},
		descriptorPresent: map[string]bool{},
	}
}

// validateDevelopmentEvidence is the Development-profile Evidence pass. Descriptor
// absence is deferred as an UNRESOLVED obligation; every descriptor that is
// supplied is validated with the frozen positive-path checks.
func (semantics *developmentSemantics) validateDevelopmentEvidence(skill *SkillAST, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Predicates == nil {
		return
	}
	indexes := semantics.evidenceIndexes(skill)
	// A supplied descriptor document must never be silently dropped.
	for _, alias := range sortedStringMapKeys(indexes) {
		for _, exportID := range sortedStringMapKeys(indexes[alias].evidence) {
			for _, descriptor := range indexes[alias].evidence[exportID] {
				if descriptor.ShapeProblem == "" {
					continue
				}
				add(procedurePath, nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
					"Evidence Descriptor "+alias+"::"+exportID+": "+descriptor.ShapeProblem)
				break
			}
		}
	}
	trustDeferred := map[string]bool{}
	deferTrust := func(alias, exportID string) {
		trustDeferred[alias+"::"+exportID] = true
	}
	for _, predicateID := range sortedStringMapKeys(skill.Predicates.Definitions) {
		predicate := skill.Predicates.Definitions[predicateID]
		if predicate == nil || predicate.Evidence == nil {
			continue
		}
		requires := unalias(child(predicate.Evidence, "requires"))
		if requires != nil && requires.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(requires.Content); i += 2 {
				for _, guarantee := range sequence(requires.Content[i+1]) {
					alias, exportID, ok := splitQualifiedReference(guarantee)
					if !ok {
						continue
					}
					semantics.validateDeferredEvidenceReference(alias, exportID, indexes, add, deferTrust)
				}
			}
		}
		for _, group := range sequence(child(predicate.Evidence, "independence")) {
			group = unalias(group)
			if group == nil {
				continue
			}
			missing := false
			for _, member := range sequence(child(group, "members")) {
				member = unalias(member)
				if member == nil {
					continue
				}
				if guarantee := unalias(child(member, "guarantee")); guarantee != nil {
					alias, exportID, ok := splitQualifiedReference(guarantee)
					if !ok {
						continue
					}
					key, exists := semantics.env.rootByAlias[alias]
					if !exists || !semantics.hasDescriptor(key, exportID) {
						missing = true
					}
				}
			}
			if missing {
				// The independence relation cannot be decided without every
				// member descriptor; the obligation records the deferral.
				continue
			}
			validateEvidenceIndependenceGroup(group, indexes, nil, add)
		}
	}
	for reference := range trustDeferred {
		semantics.deferredEvidence[reference] = true
	}
	validateHostRecoveryDualProducer(skill, dependencyIdentityResult{}, ownerLinking, indexes, procedurePath, add)
}

func (semantics *developmentSemantics) hasDescriptor(key DevelopmentPackageKey, exportID string) bool {
	present, _ := semantics.assurance.lookup(key, exportID)
	return present
}

func (semantics *developmentSemantics) validateDeferredEvidenceReference(alias, exportID string, indexes map[string]snapshotExportIndex, add func(string, *yaml.Node, string, string), deferTrust func(string, string)) {
	key, exists := semantics.env.rootByAlias[alias]
	if !exists {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID", "Evidence guarantee references an unverified dependency alias: "+alias)
		return
	}
	present, _ := semantics.assurance.lookup(key, exportID)
	if !present {
		semantics.deferredEvidence[alias+"::"+exportID] = true
		return
	}
	validateEvidenceDescriptorReference(alias, exportID, indexes, nil, add, func() {
		deferTrust(alias, exportID)
	})
	semantics.descriptorPresent[alias+"::"+exportID] = true
}

// evidenceIndexes projects the Development link scope into the frozen Evidence
// indexing shape for every Skill root alias.
func (semantics *developmentSemantics) evidenceIndexes(skill *SkillAST) map[string]snapshotExportIndex {
	indexes := map[string]snapshotExportIndex{}
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return indexes
	}
	dependencyIDs := map[string]bool{}
	for _, reference := range collectDependencyReferences(skill) {
		dependencyIDs[reference.alias] = true
	}
	for alias := range dependencyIDs {
		key, exists := semantics.env.rootByAlias[alias]
		if !exists {
			continue
		}
		index := snapshotExportIndex{callables: map[string][]linkedExport{}, schemas: map[string][]linkedExport{}, evidence: map[string][]evidenceDescriptor{}}
		pkg := semantics.env.packages[key]
		for _, export := range pkg.manifest.exports {
			switch export.kind {
			case "CallableCatalog":
				for _, record := range semantics.scope.callables {
					for _, callable := range record {
						if callable.packageKey != key || callable.exportID != export.id {
							continue
						}
						index.callables[callable.exportID] = append(index.callables[callable.exportID], linkedExport{
							ExportKind: callable.kind, ExportID: callable.exportID, Effect: callable.effect,
							SelfDeclaredOwner: callable.publisher, TrustedOwner: callable.publisher,
							SourcePath: callable.path, InputSchemaID: callable.inputSchemaID,
							QueryContextSchemaID: callable.querySchemaID, OutputSchemaID: callable.outputSchemaID,
							ReceiptSchemaID: callable.receiptSchemaID, EvidenceProvides: callable.provides,
							MutationFields: callable.mutation, ReconcileTarget: callable.reconcile,
							MutationProblem: callable.mutationProblem, OwnerOutcomes: callable.ownerOutcomes,
						})
					}
				}
			case "Schema":
				for _, resource := range semantics.scope.schemas[export.id] {
					if resource.packageKey != key {
						continue
					}
					index.schemas[export.id] = append(index.schemas[export.id], linkedExport{
						ExportKind: "schema", ExportID: export.id, SelfDeclaredOwner: pkg.manifest.publisher,
						TrustedOwner: pkg.manifest.publisher, SourcePath: export.path,
					})
				}
			}
		}
		for exportID, descriptors := range semantics.assurance.byKey[key] {
			index.evidence[exportID] = append(index.evidence[exportID], descriptors...)
		}
		indexes[alias] = index
	}
	return indexes
}

// recordOwnerTrustObligations records one OWNER_TRUST obligation per used
// cross-package export and consumption position. The declared Owner is the
// interface identity; no Compiler input proves its signing authority.
func (semantics *developmentSemantics) recordOwnerTrustObligations(skill *SkillAST, ownerLinking ownerLinkingResult) {
	if skill == nil {
		return
	}
	locations := developmentReferenceLocations(skill)
	for _, export := range ownerLinking.Exports {
		if export.ExportKind != "operation" && export.ExportKind != "observation" && export.ExportKind != "schema" {
			continue
		}
		key, exists := semantics.env.rootByAlias[export.DependencyAlias]
		if !exists {
			continue
		}
		locationKey := export.DependencyAlias + "\x00" + export.ExportKind + "\x00" + export.ExportID
		candidates := locations[locationKey]
		if len(candidates) == 0 {
			candidates = []DevelopmentObligationLocation{{
				Skill: skill.Name, Path: export.SourcePath, Node: "", Stage: "", Selector: "",
			}}
		}
		for _, location := range candidates {
			semantics.linkRecords = append(semantics.linkRecords, DevelopmentLinkRecord{
				Skill: skill.Name, Key: key, ExportKind: export.ExportKind, ExportID: export.ExportID,
				Owner: export.TrustedOwner, SourcePath: export.SourcePath, SchemaIDs: developmentExportSchemaIDs(export),
			})
			semantics.collector.add(developmentObligationKey{
				Rule:     DevelopmentObligationOwnerTrust,
				Location: location,
				Subjects: []DevelopmentObligationSubject{{
					Key: key, ID: export.ExportID, SourcePath: export.SourcePath,
					SchemaIDs: developmentExportSchemaIDs(export),
				}},
			}, DevelopmentObligationUnresolved, "")
		}
	}
}

func developmentExportSchemaIDs(export linkedExport) []string {
	ids := []string{export.InputSchemaID, export.QueryContextSchemaID, export.OutputSchemaID, export.ReceiptSchemaID}
	return dedupeSortedStrings(ids)
}

// recordEvidenceObligations records the deferred guarantee and independence
// obligations at every reachable consumption point, including the exact Producer
// and, where statically established, the Reconcile READ recovery Producer.
func (semantics *developmentSemantics) recordEvidenceObligations(skill *SkillAST, ownerLinking ownerLinkingResult, procedurePath string, indexes map[string]snapshotExportIndex) {
	if skill == nil || skill.Predicates == nil || skill.Procedure == nil {
		return
	}
	callables := map[string]developmentCallable{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind != "operation" && export.ExportKind != "observation" {
			continue
		}
		if key, exists := semantics.env.rootByAlias[export.DependencyAlias]; exists {
			if callable := semantics.scope.declaredCallable(key, export.ExportID); callable != nil {
				callables[export.DependencyAlias+"::"+export.ExportID] = *callable
			}
		}
	}
	registry := buildSchemaRegistry(semantics.legacyRequestForSkill(skill), skill, semantics.extendedIdentity(skill))
	for _, site := range developmentPredicateCallSites(skill) {
		predicate := skill.Predicates.Definitions[site.predicateID]
		if predicate == nil || predicate.Evidence == nil {
			continue
		}
		requires := unalias(child(predicate.Evidence, "requires"))
		if requires != nil && requires.Kind == yaml.MappingNode {
			parameterIndex := predicateParameterIndex(predicate.Parameters)
			for i := 0; i+1 < len(requires.Content); i += 2 {
				parameter := scalar(unalias(requires.Content[i]))
				argumentIndex, ok := parameterIndex[parameter]
				if !ok || argumentIndex >= len(site.arguments) {
					continue
				}
				argument := unalias(site.arguments[argumentIndex])
				for _, guarantee := range sequence(requires.Content[i+1]) {
					alias, exportID, ok := splitQualifiedReference(guarantee)
					if !ok {
						continue
					}
					subjects := semantics.evidenceSubjects(alias, exportID, argument, site, callables, registry)
					status, digest := semantics.evidenceStatus(alias, exportID)
					semantics.collector.add(developmentObligationKey{
						Rule: DevelopmentObligationEvidence,
						Location: DevelopmentObligationLocation{
							Skill: skill.Name, Path: procedurePath, Node: site.nodeID, Stage: site.stage,
							Selector: site.selector + "/arguments/" + itoa(argumentIndex),
						},
						Subjects: subjects,
					}, status, digest)
				}
			}
		}
		for groupIndex, group := range sequence(child(predicate.Evidence, "independence")) {
			group = unalias(group)
			if group == nil {
				continue
			}
			subjects := make([]DevelopmentObligationSubject, 0)
			unresolved := false
			for _, member := range sequence(child(group, "members")) {
				guarantee := unalias(child(member, "guarantee"))
				alias, exportID, ok := splitQualifiedReference(guarantee)
				if !ok {
					continue
				}
				key, exists := semantics.env.rootByAlias[alias]
				if !exists {
					continue
				}
				present, digest := semantics.assurance.lookup(key, exportID)
				if !present {
					unresolved = true
				} else if semantics.deferredEvidence[alias+"::"+exportID] {
					unresolved = true
				} else {
					semantics.collector.markDescriptorUsed(key.PackageID + "@" + key.Version + "\x00" + exportID)
					_ = digest
				}
				subjects = append(subjects, DevelopmentObligationSubject{
					Key: key, ID: exportID, SourcePath: semantics.assurance.pathByKey[key][exportID], SchemaIDs: []string{},
				})
			}
			status := DevelopmentObligationDescriptor
			if unresolved {
				status = DevelopmentObligationUnresolved
			}
			semantics.collector.add(developmentObligationKey{
				Rule: DevelopmentObligationIndepend,
				Location: DevelopmentObligationLocation{
					Skill: skill.Name, Path: procedurePath, Node: site.nodeID, Stage: site.stage,
					Selector: site.selector + "/independence/" + itoa(groupIndex),
				},
				Subjects: subjects,
			}, status, "")
		}
	}
}

func (semantics *developmentSemantics) evidenceStatus(alias, exportID string) (string, string) {
	key, exists := semantics.env.rootByAlias[alias]
	if !exists {
		return DevelopmentObligationUnresolved, ""
	}
	present, digest := semantics.assurance.lookup(key, exportID)
	if !present {
		return DevelopmentObligationUnresolved, ""
	}
	if semantics.deferredEvidence[alias+"::"+exportID] {
		return DevelopmentObligationUnresolved, ""
	}
	semantics.collector.markDescriptorUsed(key.PackageID + "@" + key.Version + "\x00" + exportID)
	return DevelopmentObligationDescriptor, digest
}

func (semantics *developmentSemantics) evidenceSubjects(alias, exportID string, argument *yaml.Node, site developmentPredicateCallSite, callables map[string]developmentCallable, registry schemaRegistry) []DevelopmentObligationSubject {
	subjects := make([]DevelopmentObligationSubject, 0, 2)
	if key, exists := semantics.env.rootByAlias[alias]; exists {
		subjects = append(subjects, DevelopmentObligationSubject{
			Key: key, ID: exportID, SourcePath: semantics.assurance.pathByKey[key][exportID], SchemaIDs: []string{},
		})
	}
	argument = unalias(argument)
	if argument == nil || argument.Kind != yaml.ScalarNode || argument.Tag != "!!str" || argument.Style != 0 {
		return subjects
	}
	segments := strings.Split(argument.Value, ".")
	if len(segments) < 3 {
		return subjects
	}
	nodeID := segments[0]
	producerKey := ""
	producerAlias := ""
	for alias, callable := range callables {
		_ = callable
		if strings.HasPrefix(alias, "") {
			_ = producerKey
		}
	}
	node := site.node
	_ = node
	// Resolve the producer through the same exact linked callable the Procedure
	// references.
	procedureNode := site.skill.Procedure.ByID[nodeID]
	if procedureNode == nil {
		return subjects
	}
	var tool *yaml.Node
	if procedureNode.Operation != nil {
		tool = unalias(procedureNode.Operation.Tool)
	} else if procedureNode.Observation != nil {
		tool = unalias(procedureNode.Observation.Tool)
	}
	producerAlias, producerExportID, ok := splitQualifiedReference(tool)
	if !ok {
		return subjects
	}
	callable, exists := callables[producerAlias+"::"+producerExportID]
	if !exists {
		return subjects
	}
	schemaIDs := dedupeSortedStrings([]string{callable.outputSchemaID, callable.receiptSchemaID})
	subjects = append(subjects, DevelopmentObligationSubject{
		Key: callable.packageKey, ID: callable.exportID, SourcePath: callable.path, SchemaIDs: schemaIDs,
	})
	// A recoverable Mutation output also depends on the dedicated Reconcile READ
	// Producer; record it as a distinct path subject.
	if callable.effect == "MUTATION" && callable.reconcile != "" {
		if reconcile := semantics.scope.declaredCallable(callable.packageKey, callable.reconcile); reconcile != nil {
			subjects = append(subjects, DevelopmentObligationSubject{
				Key: reconcile.packageKey, ID: reconcile.exportID, SourcePath: reconcile.path,
				SchemaIDs: dedupeSortedStrings([]string{reconcile.outputSchemaID, callable.mutation.ReconcileProofSchemaID, callable.mutation.RecoverableSchemaID}),
			})
		}
	}
	_ = producerKey
	_ = producerAlias
	_ = registry
	return subjects
}

// recordMutationObligations records COMMIT_PROOF and RECONCILE obligations for
// every reachable Mutation occurrence. The static Mutation contract checks have
// already run; only the external authority is deferred.
func (semantics *developmentSemantics) recordMutationObligations(skill *SkillAST, ownerLinking ownerLinkingResult, procedurePath string) {
	if skill == nil || skill.Procedure == nil {
		return
	}
	callables := map[string]developmentCallable{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind != "operation" || export.Effect != "MUTATION" {
			continue
		}
		if key, exists := semantics.env.rootByAlias[export.DependencyAlias]; exists {
			if callable := semantics.scope.declaredCallable(key, export.ExportID); callable != nil {
				callables[export.DependencyAlias+"::"+export.ExportID] = *callable
			}
		}
	}
	graph := buildProcedureGraph(skill.Procedure)
	reachable := reachableProcedureNodes(graph, scalar(skill.Procedure.Start))
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		if !reachable[nodeID] {
			continue
		}
		node := skill.Procedure.ByID[nodeID]
		if node == nil || node.Operation == nil {
			continue
		}
		alias, exportID, ok := splitQualifiedReference(unalias(node.Operation.Tool))
		if !ok {
			continue
		}
		callable, exists := callables[alias+"::"+exportID]
		if !exists {
			continue
		}
		subject := DevelopmentObligationSubject{
			Key: callable.packageKey, ID: callable.exportID, SourcePath: callable.path,
			SchemaIDs: dedupeSortedStrings([]string{callable.receiptSchemaID, callable.mutation.CommitProofSchemaID}),
		}
		location := DevelopmentObligationLocation{
			Skill: skill.Name, Path: procedurePath, Node: nodeID, Stage: "", Selector: "/nodes/" + nodeID + "/operation",
		}
		semantics.collector.add(developmentObligationKey{
			Rule: DevelopmentObligationCommitProof, Location: location,
			Subjects: []DevelopmentObligationSubject{subject},
		}, DevelopmentObligationUnresolved, "")
		if callable.reconcile != "" {
			subjects := []DevelopmentObligationSubject{subject}
			if reconcile := semantics.scope.declaredCallable(callable.packageKey, callable.reconcile); reconcile != nil {
				subjects = append(subjects, DevelopmentObligationSubject{
					Key: reconcile.packageKey, ID: reconcile.exportID, SourcePath: reconcile.path,
					SchemaIDs: dedupeSortedStrings([]string{callable.mutation.ReconcileProofSchemaID, callable.mutation.RecoverableSchemaID}),
				})
			}
			semantics.collector.add(developmentObligationKey{
				Rule: DevelopmentObligationReconcile, Location: location, Subjects: subjects,
			}, DevelopmentObligationUnresolved, "")
		}
	}
}

type developmentPredicateCallSite struct {
	skill       *SkillAST
	nodeID      string
	stage       string
	index       int
	predicateID string
	arguments   []*yaml.Node
	selector    string
	node        *ProcedureNodeAST
}

func developmentPredicateCallSites(skill *SkillAST) []developmentPredicateCallSite {
	if skill == nil || skill.Procedure == nil {
		return nil
	}
	graph := buildProcedureGraph(skill.Procedure)
	reachable := reachableProcedureNodes(graph, scalar(skill.Procedure.Start))
	var sites []developmentPredicateCallSite
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		if !reachable[nodeID] {
			continue
		}
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		for _, stage := range []struct {
			name  string
			value *PredicateStageAST
		}{{"precondition", node.Precondition}, {"postcondition", node.Postcondition}} {
			if stage.value == nil {
				continue
			}
			for index, call := range stage.value.Predicates {
				if call == nil {
					continue
				}
				sites = append(sites, developmentPredicateCallSite{
					skill: skill, nodeID: nodeID, stage: stage.name, index: index,
					predicateID: scalar(unalias(call.Predicate)),
					arguments:   sequence(call.Arguments),
					selector:    "/nodes/" + nodeID + "/" + stage.name + "/predicates/" + itoa(index),
					node:        node,
				})
			}
		}
	}
	return sites
}

// developmentReferenceLocations maps alias + export kind + export ID to the exact
// author source positions that reference it.
func developmentReferenceLocations(skill *SkillAST) map[string][]DevelopmentObligationLocation {
	locations := map[string][]DevelopmentObligationLocation{}
	if skill == nil {
		return locations
	}
	contractPath := "skills/" + skill.Name + "/assurance/contract.yaml"
	procedurePath := "skills/" + skill.Name + "/assurance/procedure.yaml"
	predicatesPath := "skills/" + skill.Name + "/assurance/predicates.yaml"
	add := func(alias, kind, id string, location DevelopmentObligationLocation) {
		key := alias + "\x00" + kind + "\x00" + id
		locations[key] = append(locations[key], location)
	}
	if skill.Contract != nil {
		if alias, id, ok := splitQualifiedReference(skill.Contract.ContextSchema); ok {
			add(alias, "schema", id, DevelopmentObligationLocation{
				Skill: skill.Name, Path: contractPath, Node: "", Stage: "", Selector: "/contextSchema",
			})
		}
	}
	if skill.Procedure != nil {
		for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
			node := skill.Procedure.ByID[nodeID]
			if node == nil {
				continue
			}
			if node.Operation != nil {
				if alias, id, ok := splitQualifiedReference(node.Operation.Tool); ok {
					add(alias, "operation", id, DevelopmentObligationLocation{
						Skill: skill.Name, Path: procedurePath, Node: nodeID, Stage: "", Selector: "/nodes/" + nodeID + "/operation/tool",
					})
				}
			}
			if node.Observation != nil {
				if alias, id, ok := splitQualifiedReference(node.Observation.Tool); ok {
					add(alias, "observation", id, DevelopmentObligationLocation{
						Skill: skill.Name, Path: procedurePath, Node: nodeID, Stage: "", Selector: "/nodes/" + nodeID + "/observation/tool",
					})
				}
			}
		}
	}
	if skill.Predicates != nil {
		for _, predicateID := range sortedStringMapKeys(skill.Predicates.Definitions) {
			predicate := skill.Predicates.Definitions[predicateID]
			if predicate == nil {
				continue
			}
			for index, parameter := range sequence(predicate.Parameters) {
				parameter = unalias(parameter)
				if parameter == nil || parameter.Kind != yaml.MappingNode {
					continue
				}
				for i := 0; i+1 < len(parameter.Content); i += 2 {
					value := unalias(parameter.Content[i+1])
					if alias, id, ok := splitQualifiedReference(value); ok {
						add(alias, "schema", id, DevelopmentObligationLocation{
							Skill: skill.Name, Path: predicatesPath, Node: predicateID, Stage: "parameters",
							Selector: "/predicates/" + predicateID + "/parameters/" + itoa(index),
						})
					}
				}
			}
			for _, reference := range collectEvidenceGuaranteeReferences(predicate.Evidence, predicatesPath) {
				add(reference.alias, "evidenceGuarantee", reference.exportID, DevelopmentObligationLocation{
					Skill: skill.Name, Path: predicatesPath, Node: predicateID, Stage: "evidence", Selector: "/predicates/" + predicateID + "/evidence",
				})
			}
		}
	}
	base := "skills/" + skill.Name + "/schemas/"
	for _, filename := range sortedStringMapKeys(skill.Files) {
		if !strings.HasPrefix(filename, base) || !(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
			continue
		}
		for _, reference := range collectSchemaRefReferences(skill.Files[filename], filename) {
			add(reference.alias, "schema", reference.exportID, DevelopmentObligationLocation{
				Skill: skill.Name, Path: filename, Node: "", Stage: "", Selector: "/$ref",
			})
		}
	}
	for key := range locations {
		sort.Slice(locations[key], func(i, j int) bool {
			a, b := locations[key][i], locations[key][j]
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			if a.Node != b.Node {
				return a.Node < b.Node
			}
			if a.Stage != b.Stage {
				return a.Stage < b.Stage
			}
			return a.Selector < b.Selector
		})
	}
	return locations
}

// legacyRequestForSkill and extendedIdentity reconstruct the internal request
// view used only to build the shared Schema registry during obligation recording.
func (semantics *developmentSemantics) legacyRequestForSkill(skill *SkillAST) BuildRequest {
	req, _ := semantics.env.legacyRequest()
	return req
}

func (semantics *developmentSemantics) extendedIdentity(skill *SkillAST) dependencyIdentityResult {
	identity := validateDependencyIdentityWithUsed(semantics.legacyRequestForSkill(skill), skill, nil)
	if identity.Closure == nil {
		identity.Closure = []string{}
	}
	return semantics.env.extendIdentity(identity, semantics.synthetic)
}
