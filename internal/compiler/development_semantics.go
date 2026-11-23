package compiler

// Development Build semantic integration.
//
// The single v1alpha5 pipeline runs unchanged for every statically decidable
// rule. This file adds the Development-profile branches that consume the
// package-scoped interface scope: capability membership from CapabilityCatalog
// exports, Evidence Descriptor linking from EvidenceDescriptor exports, and
// typed Host Admission requirement recording for external pre-dispatch facts and
// typed Runtime Execution requirement recording for execution-time guarantees.
//
// Evidence Descriptor absence or malformation is a Build failure: the frozen
// language semantics require the static contract. Only external trust facts
// (Owner authority, domain authority, independence authorization) become Host
// Admission requirements; real commit proof and real reconciliation become
// Runtime Execution requirements.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type developmentSemantics struct {
	env                    *developmentEnvironment
	scope                  *developmentLinkScope
	synthetic              []syntheticDependencyAlias
	collector              *developmentRequirementCollector
	linkRecords            []DevelopmentLinkRecord
	capabilityRequirements []DevelopmentCapabilityRequirement
}

func newDevelopmentSemantics(env *developmentEnvironment, scope *developmentLinkScope, synthetic []syntheticDependencyAlias, collector *developmentRequirementCollector) *developmentSemantics {
	return &developmentSemantics{env: env, scope: scope, synthetic: synthetic, collector: collector}
}

// recordCapabilityRequirements appends the Skill's static Capability selections
// to the manifest and records the external Catalog Owner authority fact.
//
// The catalog Owner identity comes from the locked Package manifest publisher,
// which is the same Owner attribution every other Interface Package export uses.
// The fact that a real publisher controls that identity is not established by
// Build, so it stays an OWNER_TRUST Host Admission requirement; the artifact
// itself stays NOT_ADMITTED. No Deployment Grant is recorded or implied.
func (semantics *developmentSemantics) recordCapabilityRequirements(skill *SkillAST, requirements []DevelopmentCapabilityRequirement) {
	if skill == nil {
		return
	}
	for _, requirement := range requirements {
		semantics.capabilityRequirements = append(semantics.capabilityRequirements, requirement)
		semantics.collector.add(developmentRequirementKey{
			Kind: DevelopmentRequirementOwnerTrust,
			Location: DevelopmentRequirementLocation{
				Skill: requirement.Skill, Path: requirement.Declaration.Path,
				Node: requirement.CapabilityID, Stage: "capabilities",
				Selector: requirement.Declaration.Pointer,
			},
			Subjects: []DevelopmentRequirementSubject{{
				Key: requirement.Catalog.Key, ID: requirement.Catalog.ExportID,
				SourcePath: requirement.Catalog.SourcePath, SchemaIDs: []string{},
			}},
		})
	}
}

// validateDevelopmentEvidence runs the frozen Evidence Descriptor linking against
// the package-scoped EvidenceDescriptor exports. Supplied descriptors are fully
// validated; a missing or malformed descriptor fails the Build. Only the trust
// predicates that require an external authority are recorded as Host Admission
// requirements.
func (semantics *developmentSemantics) validateDevelopmentEvidence(skill *SkillAST, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Predicates == nil {
		return
	}
	indexes := semantics.evidenceIndexes(skill)
	// A supplied descriptor document must never be silently dropped. Scope
	// indexing already rejected malformed entries; this scan keeps the frozen
	// behaviour for every indexed descriptor.
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
	// A descriptor may declare sourceIdentity and factIdentity at the same time.
	// The two external trust facts are recorded independently and never
	// overwrite each other.
	trustDeferred := map[string]map[string]bool{}
	deferTrust := func(alias, exportID, reason string) {
		reference := alias + "::" + exportID
		if trustDeferred[reference] == nil {
			trustDeferred[reference] = map[string]bool{}
		}
		trustDeferred[reference][reason] = true
	}
	independenceGroups := make([]*DevelopmentIndependenceGroup, 0)
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
					validateEvidenceDescriptorReference(alias, exportID, indexes, nil, add, func(reason string) {
						deferTrust(alias, exportID, reason)
					})
				}
			}
		}
		for index, group := range sequence(child(predicate.Evidence, "independence")) {
			group = unalias(group)
			if group == nil {
				continue
			}
			if recorded := validateDevelopmentIndependenceGroup(group, indexes, semantics.env.rootByAlias, predicateID, index, add); recorded != nil {
				independenceGroups = append(independenceGroups, recorded)
			}
		}
	}
	validateHostRecoveryDualProducer(skill, dependencyIdentityResult{}, ownerLinking, indexes, procedurePath, add)
	semantics.recordEvidenceTrustRequirements(skill, indexes, trustDeferred)
	semantics.recordIndependenceRequirements(skill, independenceGroups)
}

// validateDevelopmentIndependenceGroup performs every statically decidable
// independence check and returns the exact group identity, members, Source
// Domains and Domain Pairs. Only the external IndependentDomains authorization
// is deferred to Host Admission. A structurally impossible group (fewer than two
// members, ambiguous Descriptor, missing Source Identity, fewer than two distinct
// declared domains) fails the Build.
func validateDevelopmentIndependenceGroup(group *yaml.Node, indexes map[string]snapshotExportIndex, rootByAlias map[string]DevelopmentPackageKey, predicateID string, groupIndex int, add func(string, *yaml.Node, string, string)) *DevelopmentIndependenceGroup {
	type memberRef struct {
		alias  string
		export string
	}
	members := make([]memberRef, 0)
	for _, member := range sequence(child(group, "members")) {
		member = unalias(member)
		if member == nil {
			continue
		}
		alias, exportID, ok := splitQualifiedReference(unalias(child(member, "guarantee")))
		if !ok {
			continue
		}
		members = append(members, memberRef{alias: alias, export: exportID})
	}
	if len(members) < 2 {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
			"Evidence independence group must contain at least two (parameter, guarantee) pairs")
		return nil
	}
	recordedMembers := make([]DevelopmentIndependenceMember, 0, len(members))
	seenMember := map[string]bool{}
	for _, member := range members {
		key, keyExists := rootByAlias[member.alias]
		if !keyExists {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence member references an unverified dependency alias: "+member.alias)
			return nil
		}
		index, ok := indexes[member.alias]
		if !ok {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence member references an unverified dependency alias: "+member.alias)
			return nil
		}
		descriptors := index.evidence[member.export]
		if len(descriptors) != 1 {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence member does not resolve exactly once: "+member.alias+"::"+member.export)
			return nil
		}
		descriptor := descriptors[0]
		source := descriptor.SourceIdentity
		if descriptor.ShapeProblem != "" || source == nil || source.Domain == "" || !strings.HasPrefix(source.Path, "record.") {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence requires a statically valid sourceIdentity (domain + record.* path) for "+member.alias+"::"+member.export)
			return nil
		}
		identity := member.alias + "\x00" + member.export + "\x00" + developmentKeyString(key)
		if seenMember[identity] {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence member duplicates another member for "+member.alias+"::"+member.export)
			return nil
		}
		seenMember[identity] = true
		recordedMembers = append(recordedMembers, DevelopmentIndependenceMember{
			DependencyAlias: member.alias,
			Key:             key,
			ExportID:        member.export,
			SourcePath:      descriptor.SnapshotPath,
			Domain:          source.Domain,
		})
	}
	domains := map[string]bool{}
	for _, member := range recordedMembers {
		domains[member.Domain] = true
	}
	if len(domains) < 2 {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
			"Evidence independence group requires at least two distinct declared source domains")
		return nil
	}
	domainList := make([]string, 0, len(domains))
	for domain := range domains {
		domainList = append(domainList, domain)
	}
	sort.Strings(domainList)
	pairs := make([][]string, 0, len(domainList)*(len(domainList)-1)/2)
	for i := 0; i < len(domainList); i++ {
		for j := i + 1; j < len(domainList); j++ {
			pairs = append(pairs, []string{domainList[i], domainList[j]})
		}
	}
	return &DevelopmentIndependenceGroup{GroupIndex: groupIndex, Predicate: predicateID, Members: recordedMembers, DomainPairs: pairs}
}

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
			case "EvidenceDescriptor":
				if descriptor, exists := semantics.scope.evidence[key][export.id]; exists {
					index.evidence[export.id] = append(index.evidence[export.id], descriptor)
				}
			}
		}
		indexes[alias] = index
	}
	return indexes
}

// recordOwnerTrustRequirements records Host Admission owner-identity
// requirements for used cross-package exports. The declared Owner remains the
// interface identity used by the static checks.
//
// An EvidenceDescriptor export is a formal cross-package export like any other:
// its declared `authority` is the Owner identity and it therefore requires its
// own OWNER_TRUST. EVIDENCE_DOMAIN_TRUST and EVIDENCE_FACT_TRUST authorize the
// external source domain and CommonFact instead; neither replaces the Owner
// identity check. The Subject keeps the exact locked snapshot path, and the
// location is the real Predicate Evidence JSON Pointer rather than a derived
// "somewhere under evidence" position.
func (semantics *developmentSemantics) recordOwnerTrustRequirements(skill *SkillAST, ownerLinking ownerLinkingResult) {
	if skill == nil {
		return
	}
	locations := developmentReferenceLocations(skill)
	for _, export := range ownerLinking.Exports {
		switch export.ExportKind {
		case "operation", "observation", "schema", "evidenceGuarantee":
		default:
			continue
		}
		key, exists := semantics.env.rootByAlias[export.DependencyAlias]
		if !exists {
			continue
		}
		locationKey := export.DependencyAlias + "\x00" + export.ExportKind + "\x00" + export.ExportID
		candidates := locations[locationKey]
		if len(candidates) == 0 {
			if export.ExportKind == "evidenceGuarantee" {
				// A Descriptor without a real Predicate Evidence position cannot
				// produce an exactly locatable requirement. The static Evidence
				// pass has already rejected the reference, so the Build fails and
				// no Artifact is produced.
				continue
			}
			candidates = []DevelopmentRequirementLocation{{
				Skill: skill.Name, Path: export.SourcePath, Node: "", Stage: "", Selector: "",
			}}
		}
		semantics.linkRecords = append(semantics.linkRecords, DevelopmentLinkRecord{
			Skill: skill.Name, Key: key, ExportKind: export.ExportKind, ExportID: export.ExportID,
			Owner: export.TrustedOwner, SourcePath: export.SourcePath, SchemaIDs: developmentExportSchemaIDs(export),
		})
		for _, location := range candidates {
			semantics.collector.add(developmentRequirementKey{
				Kind:     DevelopmentRequirementOwnerTrust,
				Location: location,
				Subjects: []DevelopmentRequirementSubject{{
					Key: key, ID: export.ExportID, SourcePath: export.SourcePath,
					SchemaIDs: developmentExportSchemaIDs(export),
				}},
			})
		}
	}
}

func developmentExportSchemaIDs(export linkedExport) []string {
	ids := []string{export.InputSchemaID, export.QueryContextSchemaID, export.OutputSchemaID, export.ReceiptSchemaID}
	return dedupeSortedStrings(ids)
}

// recordEvidenceTrustRequirements records the external trust facts discovered
// while the static Descriptor checks ran. A descriptor that declares both
// sourceIdentity and factIdentity yields both requirements. Requirement presence
// never substitutes for a static check: every supplied descriptor was fully
// validated above.
func (semantics *developmentSemantics) recordEvidenceTrustRequirements(skill *SkillAST, indexes map[string]snapshotExportIndex, trustDeferred map[string]map[string]bool) {
	locations := developmentReferenceLocations(skill)
	for _, reference := range sortedStringMapKeys(trustDeferred) {
		alias, exportID, ok := strings.Cut(reference, "::")
		if !ok {
			continue
		}
		key, exists := semantics.env.rootByAlias[alias]
		if !exists {
			continue
		}
		descriptor := indexes[alias].evidence[exportID]
		sourcePath := ""
		if len(descriptor) == 1 {
			sourcePath = descriptor[0].SnapshotPath
		}
		candidates := locations[alias+"\x00evidenceGuarantee\x00"+exportID]
		if len(candidates) == 0 {
			candidates = []DevelopmentRequirementLocation{{
				Skill: skill.Name, Path: "skills/" + skill.Name + "/assurance/predicates.yaml", Node: exportID, Stage: "evidence", Selector: "",
			}}
		}
		for _, reason := range sortedStringMapKeys(trustDeferred[reference]) {
			kind := ""
			switch reason {
			case "sourceIdentity":
				kind = DevelopmentRequirementEvidenceDomain
			case "factIdentity":
				kind = DevelopmentRequirementEvidenceFact
			default:
				continue
			}
			for _, location := range candidates {
				semantics.collector.add(developmentRequirementKey{
					Kind:     kind,
					Location: location,
					Subjects: []DevelopmentRequirementSubject{{
						Key: key, ID: exportID, SourcePath: sourcePath, SchemaIDs: []string{},
					}},
				})
			}
		}
	}
}

// recordIndependenceRequirements records every statically valid Independence
// Group as its own Host Admission requirement. The requirement carries the exact
// group source position, every member Descriptor, every Source Domain, the
// Domain Pairs that need external authorization and every exact Interface
// Package key. No group is collapsed into a single boolean and no representative
// root is chosen.
func (semantics *developmentSemantics) recordIndependenceRequirements(skill *SkillAST, groups []*DevelopmentIndependenceGroup) {
	for _, group := range groups {
		subjects := make([]DevelopmentRequirementSubject, 0, len(group.Members))
		for _, member := range group.Members {
			subjects = append(subjects, DevelopmentRequirementSubject{
				Key: member.Key, ID: member.ExportID, SourcePath: member.SourcePath, SchemaIDs: []string{},
			})
		}
		detail := normalizeDevelopmentIndependenceGroup(group)
		semantics.collector.add(developmentRequirementKey{
			Kind: DevelopmentRequirementIndependence,
			Location: DevelopmentRequirementLocation{
				Skill: skill.Name, Path: "skills/" + skill.Name + "/assurance/predicates.yaml",
				Node: detail.Predicate, Stage: "evidence",
				Selector: "/predicates/" + v1alpha5JSONPointerToken(detail.Predicate) + "/evidence/independence/" + itoa(detail.GroupIndex),
			},
			Subjects:     subjects,
			Independence: detail,
		})
	}
}

// recordMutationRequirements records the external Mutation facts. The static
// Mutation, Commit-Proof and Reconcile contracts were already validated by the
// inherited pipeline.
func (semantics *developmentSemantics) recordMutationRequirements(skill *SkillAST, ownerLinking ownerLinkingResult, procedurePath string) {
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
		subject := DevelopmentRequirementSubject{
			Key: callable.packageKey, ID: callable.exportID, SourcePath: callable.path,
			SchemaIDs: dedupeSortedStrings([]string{callable.receiptSchemaID, callable.mutation.CommitProofSchemaID}),
		}
		location := DevelopmentRequirementLocation{
			Skill: skill.Name, Path: procedurePath, Node: nodeID, Stage: "", Selector: "/nodes/" + nodeID + "/operation",
		}
		semantics.collector.add(developmentRequirementKey{
			Kind: DevelopmentRequirementCommitProof, Location: location,
			Subjects: []DevelopmentRequirementSubject{subject},
		})
		if callable.reconcile != "" {
			subjects := []DevelopmentRequirementSubject{subject}
			if reconcile := semantics.scope.declaredCallable(callable.packageKey, callable.reconcile); reconcile != nil {
				subjects = append(subjects, DevelopmentRequirementSubject{
					Key: reconcile.packageKey, ID: reconcile.exportID, SourcePath: reconcile.path,
					SchemaIDs: dedupeSortedStrings([]string{callable.mutation.ReconcileProofSchemaID, callable.mutation.RecoverableSchemaID}),
				})
			}
			semantics.collector.add(developmentRequirementKey{
				Kind: DevelopmentRequirementReconcile, Location: location, Subjects: subjects,
			})
		}
	}
}

// developmentReferenceLocations maps alias + export kind + export ID to the exact
// author source positions that reference it.
func developmentReferenceLocations(skill *SkillAST) map[string][]DevelopmentRequirementLocation {
	locations := map[string][]DevelopmentRequirementLocation{}
	if skill == nil {
		return locations
	}
	contractPath := "skills/" + skill.Name + "/assurance/contract.yaml"
	procedurePath := "skills/" + skill.Name + "/assurance/procedure.yaml"
	predicatesPath := "skills/" + skill.Name + "/assurance/predicates.yaml"
	add := func(alias, kind, id string, location DevelopmentRequirementLocation) {
		key := alias + "\x00" + kind + "\x00" + id
		locations[key] = append(locations[key], location)
	}
	if skill.Contract != nil {
		if alias, id, ok := splitQualifiedReference(skill.Contract.ContextSchema); ok {
			add(alias, "schema", id, DevelopmentRequirementLocation{
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
					add(alias, "operation", id, DevelopmentRequirementLocation{
						Skill: skill.Name, Path: procedurePath, Node: nodeID, Stage: "", Selector: "/nodes/" + nodeID + "/operation/tool",
					})
				}
			}
			if node.Observation != nil {
				if alias, id, ok := splitQualifiedReference(node.Observation.Tool); ok {
					add(alias, "observation", id, DevelopmentRequirementLocation{
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
						add(alias, "schema", id, DevelopmentRequirementLocation{
							Skill: skill.Name, Path: predicatesPath, Node: predicateID, Stage: "parameters",
							Selector: "/predicates/" + predicateID + "/parameters/" + itoa(index),
						})
					}
				}
			}
			for _, reference := range collectEvidenceGuaranteeReferences(predicateID, predicate.Evidence, predicatesPath) {
				add(reference.alias, "evidenceGuarantee", reference.exportID, DevelopmentRequirementLocation{
					Skill: skill.Name, Path: predicatesPath, Node: predicateID, Stage: "evidence", Selector: reference.selector,
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
			add(reference.alias, "schema", reference.exportID, DevelopmentRequirementLocation{
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

func (semantics *developmentSemantics) extendedIdentity(skill *SkillAST) dependencyIdentityResult {
	req, _ := semantics.env.legacyRequest()
	identity := validateDependencyIdentityWithUsed(req, skill, nil)
	if identity.Closure == nil {
		identity.Closure = []string{}
	}
	return semantics.env.extendIdentity(identity, semantics.synthetic)
}
