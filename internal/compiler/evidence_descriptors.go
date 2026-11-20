package compiler

// Evidence Descriptor static linking (v1alpha4 搂17 / Appendix F.3 / G.3.1,
// inherited by v1alpha5).
//
// The released language fixes the Evidence Descriptor logical DSL through the
// E.3 example and F.3 prose: a trusted dependency snapshot may publish an
// "evidence-guarantees" document whose top-level keys are guarantee export IDs
// and whose values carry authority/fact/producers/scope/correlation/factValue/
// selection/freshness (and optional sourceIdentity/factIdentity). The Compiler
// resolves every Predicate guarantee reference to exactly one such descriptor,
// matches every declared Producer to the exact trusted ToolCatalog callable and
// its provides.path, validates the static reference roots and selection/freshness
// fields, and verifies independence/fact identity against the trusted deployment
// EvidenceTrustBindings input.
//
// Runtime record selection, receipts and actual values remain Runtime/Host
// responsibilities.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type evidenceProducer struct {
	Owner   string
	Kind    string
	ID      string
	Outcome string
	Path    string
}

type evidenceBinding struct {
	Name   string
	Record string
	Equals string
}

type evidenceSelection struct {
	Mode    string
	OrderBy string
}

type evidenceFreshness struct {
	Timestamp string
	MaxAgeMs  int64
	HasMaxAge bool
}

type evidenceSourceIdentity struct {
	Domain string
	Path   string
}

type evidenceFactIdentity struct {
	Namespace  string
	Fact       string
	ValuePath  string
	ScopePaths []evidenceBinding
	// Owners is populated only for trusted CommonFact deployment bindings; a
	// published descriptor never declares its own Owner authorization.
	Owners []string
}

type evidenceDescriptor struct {
	ExportID string
	// ShapeProblem is the first structural problem found in the trusted
	// descriptor document. A malformed container or binding must never be
	// silently dropped: the linked guarantee is rejected instead.
	ShapeProblem   string
	Owner          string
	Authority      string
	Fact           string
	Producers      []evidenceProducer
	Scope          []evidenceBinding
	Correlation    []evidenceBinding
	FactValue      string
	Selection      evidenceSelection
	Freshness      evidenceFreshness
	SourceIdentity *evidenceSourceIdentity
	FactIdentity   *evidenceFactIdentity
	SnapshotPath   string
}

// parseEvidenceDescriptors detects and parses one evidence-guarantees document.
// A descriptor document is a top-level mapping (without apiVersion/kind) whose
// entries carry the F.3 logical fields. Recognition is deliberately tolerant so
// that a malformed entry is still indexed and rejected with an explicit
// diagnostic instead of disappearing from the trusted export set.
func parseEvidenceDescriptors(root *yaml.Node) (map[string][]evidenceDescriptor, bool) {
	root = unalias(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, false
	}
	if child(root, "apiVersion") != nil || child(root, "kind") != nil {
		return nil, false
	}
	detected := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		value := unalias(root.Content[i+1])
		if value == nil || value.Kind != yaml.MappingNode {
			continue
		}
		if child(value, "authority") != nil || child(value, "producers") != nil {
			detected = true
			break
		}
	}
	if !detected {
		return nil, false
	}
	descriptors := map[string][]evidenceDescriptor{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := unalias(root.Content[i])
		value := unalias(root.Content[i+1])
		if key == nil || key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			// An unaddressable descriptor entry makes the trusted export set
			// ambiguous; dropping it would fail open for every later reference.
			return nil, false
		}
		exportID := key.Value
		if exportID == "" {
			return nil, false
		}
		if value == nil || value.Kind != yaml.MappingNode {
			descriptors[exportID] = append(descriptors[exportID], evidenceDescriptor{
				ExportID:     exportID,
				ShapeProblem: "Evidence Descriptor entry must be a mapping",
			})
			continue
		}
		descriptors[exportID] = append(descriptors[exportID], parseEvidenceDescriptor(exportID, value))
	}
	return descriptors, true
}

func parseEvidenceDescriptor(exportID string, node *yaml.Node) evidenceDescriptor {
	descriptor := evidenceDescriptor{ExportID: exportID}
	problem := func(message string) {
		if descriptor.ShapeProblem == "" {
			descriptor.ShapeProblem = message
		}
	}
	setString := func(field string, required bool) string {
		raw := unalias(child(node, field))
		if raw == nil {
			if required {
				problem(field + " is required")
			}
			return ""
		}
		if raw.Kind != yaml.ScalarNode || raw.Tag != "!!str" {
			problem(field + " must be a string")
			return ""
		}
		if required && raw.Value == "" {
			problem(field + " must be a non-empty string")
		}
		return raw.Value
	}

	descriptor.Authority = setString("authority", true)
	descriptor.Owner = descriptor.Authority
	descriptor.Fact = setString("fact", true)
	if descriptor.Fact != "" && !validExportID(descriptor.Fact) {
		problem("fact must be a DottedId")
	}

	producersNode := unalias(child(node, "producers"))
	if producersNode == nil {
		problem("producers is required")
	} else if producersNode.Kind != yaml.SequenceNode {
		problem("producers must be an ordered list")
	} else if len(producersNode.Content) == 0 {
		problem("producers must contain at least one Producer")
	} else {
		for index, producerNode := range producersNode.Content {
			producerNode = unalias(producerNode)
			if producerNode == nil || producerNode.Kind != yaml.MappingNode {
				problem("producers[" + itoa(index) + "] must be a mapping with owner/kind/id/outcome/path")
				continue
			}
			producer := evidenceProducer{
				Owner:   evidenceStringField(producerNode, "owner", "producers["+itoa(index)+"].owner", problem),
				Kind:    evidenceStringField(producerNode, "kind", "producers["+itoa(index)+"].kind", problem),
				ID:      evidenceStringField(producerNode, "id", "producers["+itoa(index)+"].id", problem),
				Outcome: evidenceStringField(producerNode, "outcome", "producers["+itoa(index)+"].outcome", problem),
				Path:    evidenceStringField(producerNode, "path", "producers["+itoa(index)+"].path", problem),
			}
			if producer.Kind != "" && producer.Kind != "operation" && producer.Kind != "observation" {
				problem("producers[" + itoa(index) + "].kind must be operation or observation")
			}
			if producer.Outcome != "" && producer.Outcome != "SUCCEEDED" {
				problem("producers[" + itoa(index) + "].outcome must be SUCCEEDED")
			}
			if producer.Path != "" && !validDottedIdentifierPath(producer.Path) {
				problem("producers[" + itoa(index) + "].path must be a static dotted field path")
			}
			descriptor.Producers = append(descriptor.Producers, producer)
		}
	}

	descriptor.Scope, descriptor.Correlation = nil, nil
	if bindings, problemText := parseEvidenceBindings(node, "scope"); problemText != "" {
		problem(problemText)
	} else {
		descriptor.Scope = bindings
	}
	if bindings, problemText := parseEvidenceBindings(node, "correlation"); problemText != "" {
		problem(problemText)
	} else {
		descriptor.Correlation = bindings
	}

	descriptor.FactValue = setString("factValue", true)
	if descriptor.FactValue != "" && !validRecordRelativePath(descriptor.FactValue) {
		problem("factValue must be a static record-relative field path")
	}

	selectionNode := unalias(child(node, "selection"))
	if selectionNode == nil {
		problem("selection is required")
	} else if selectionNode.Kind != yaml.MappingNode {
		problem("selection must be a mapping with mode and optional orderBy")
	} else {
		descriptor.Selection.Mode = evidenceStringField(selectionNode, "mode", "selection.mode", problem)
		descriptor.Selection.OrderBy = evidenceStringField(selectionNode, "orderBy", "selection.orderBy", problem)
		switch descriptor.Selection.Mode {
		case "exact":
		case "latest":
			if descriptor.Selection.OrderBy == "" {
				problem("selection.mode latest requires orderBy")
			}
		default:
			if descriptor.Selection.Mode != "" {
				problem("selection.mode must be exact or latest")
			}
		}
	}

	freshnessNode := unalias(child(node, "freshness"))
	if freshnessNode == nil {
		problem("freshness is required")
	} else if freshnessNode.Kind != yaml.MappingNode {
		problem("freshness must be a mapping with timestamp and maxAgeMs")
	} else {
		descriptor.Freshness.Timestamp = evidenceStringField(freshnessNode, "timestamp", "freshness.timestamp", problem)
		if descriptor.Freshness.Timestamp != "" && !validRecordRelativePath(descriptor.Freshness.Timestamp) {
			problem("freshness.timestamp must be a static record-relative field path")
		}
		maxAgeNode := unalias(child(freshnessNode, "maxAgeMs"))
		if maxAgeNode == nil {
			problem("freshness.maxAgeMs is required")
		} else if value, ok := evidencePositiveInteger(maxAgeNode); ok {
			descriptor.Freshness.MaxAgeMs = value
			descriptor.Freshness.HasMaxAge = true
		} else {
			problem("freshness.maxAgeMs must be a positive integer")
		}
	}

	if sourceNode := unalias(child(node, "sourceIdentity")); sourceNode != nil {
		if sourceNode.Kind != yaml.MappingNode {
			problem("sourceIdentity must be a mapping with domain and path")
		} else {
			identity := &evidenceSourceIdentity{
				Domain: evidenceStringField(sourceNode, "domain", "sourceIdentity.domain", problem),
				Path:   evidenceStringField(sourceNode, "path", "sourceIdentity.path", problem),
			}
			if identity.Domain != "" && !validAbsoluteURI(identity.Domain) {
				problem("sourceIdentity.domain must be an absolute URI")
			}
			if identity.Path != "" && !strings.HasPrefix(identity.Path, "record.") {
				problem("sourceIdentity.path must be a record.* static path")
			}
			descriptor.SourceIdentity = identity
		}
	}

	if factIdentityNode := unalias(child(node, "factIdentity")); factIdentityNode != nil {
		if factIdentityNode.Kind != yaml.MappingNode {
			problem("factIdentity must be a mapping with namespace, fact, scopePaths and valuePath")
		} else {
			identity := &evidenceFactIdentity{
				Namespace: evidenceStringField(factIdentityNode, "namespace", "factIdentity.namespace", problem),
				Fact:      evidenceStringField(factIdentityNode, "fact", "factIdentity.fact", problem),
				ValuePath: evidenceStringField(factIdentityNode, "valuePath", "factIdentity.valuePath", problem),
			}
			if identity.Namespace != "" && !validAbsoluteURI(identity.Namespace) {
				problem("factIdentity.namespace must be an absolute URI")
			}
			if identity.Fact != "" && !validExportID(identity.Fact) {
				problem("factIdentity.fact must be a DottedId")
			}
			if identity.ValuePath != "" && !strings.HasPrefix(identity.ValuePath, "record.") {
				problem("factIdentity.valuePath must be a record.* static path")
			}
			scopePaths := unalias(child(factIdentityNode, "scopePaths"))
			if scopePaths == nil {
				problem("factIdentity.scopePaths is required")
			} else if scopePaths.Kind != yaml.MappingNode {
				problem("factIdentity.scopePaths must be a mapping of dimension to record.* path")
			} else {
				for i := 0; i+1 < len(scopePaths.Content); i += 2 {
					name := scalar(unalias(scopePaths.Content[i]))
					if name == "" || !validDependencyID(name) {
						problem("factIdentity.scopePaths keys must be Identifiers")
						continue
					}
					value := unalias(scopePaths.Content[i+1])
					if value == nil || value.Kind != yaml.ScalarNode || value.Tag != "!!str" || !strings.HasPrefix(value.Value, "record.") {
						problem("factIdentity.scopePaths." + name + " must be a record.* static path")
						continue
					}
					identity.ScopePaths = append(identity.ScopePaths, evidenceBinding{Name: name, Record: value.Value})
				}
			}
			descriptor.FactIdentity = identity
		}
	}
	return descriptor
}

func evidenceStringField(node *yaml.Node, field, label string, problem func(string)) string {
	raw := unalias(child(node, field))
	if raw == nil {
		problem(label + " is required")
		return ""
	}
	if raw.Kind != yaml.ScalarNode || raw.Tag != "!!str" {
		problem(label + " must be a string")
		return ""
	}
	if raw.Value == "" {
		problem(label + " must be a non-empty string")
	}
	return raw.Value
}

func parseEvidenceBindings(node *yaml.Node, field string) ([]evidenceBinding, string) {
	container := unalias(child(node, field))
	if container == nil {
		return nil, field + " is required"
	}
	if container.Kind != yaml.MappingNode {
		return nil, field + " must be a mapping of named equality bindings"
	}
	if len(container.Content) == 0 {
		return nil, field + " must declare at least one equality binding"
	}
	bindings := make([]evidenceBinding, 0, len(container.Content)/2)
	for i := 0; i+1 < len(container.Content); i += 2 {
		nameNode := unalias(container.Content[i])
		name := scalar(nameNode)
		if name == "" || !validDependencyID(name) {
			return nil, field + " binding names must be Identifiers"
		}
		value := unalias(container.Content[i+1])
		if value == nil || value.Kind != yaml.MappingNode {
			return nil, field + "." + name + " must be a mapping with record and equals"
		}
		recordNode := unalias(child(value, "record"))
		equalsNode := unalias(child(value, "equals"))
		if recordNode == nil || recordNode.Kind != yaml.ScalarNode || recordNode.Tag != "!!str" {
			return nil, field + "." + name + ".record must be a non-empty string"
		}
		if equalsNode == nil || equalsNode.Kind != yaml.ScalarNode || equalsNode.Tag != "!!str" {
			return nil, field + "." + name + ".equals must be a non-empty string"
		}
		record := strings.TrimPrefix(recordNode.Value, "record.")
		if record == "" || !validDottedIdentifierPath(record) {
			return nil, field + "." + name + ".record must be a static record-relative field path"
		}
		if !validEvidenceReferencePath(equalsNode.Value) {
			return nil, field + "." + name + ".equals must be an exact query/arguments/execution/record reference path"
		}
		bindings = append(bindings, evidenceBinding{Name: name, Record: record, Equals: equalsNode.Value})
	}
	return bindings, ""
}

func validDottedIdentifierPath(value string) bool {
	if value == "" {
		return false
	}
	for _, segment := range strings.Split(value, ".") {
		if !validDependencyID(segment) {
			return false
		}
	}
	return true
}

func validRecordRelativePath(value string) bool {
	return validDottedIdentifierPath(strings.TrimPrefix(value, "record."))
}

func evidencePositiveInteger(node *yaml.Node) (int64, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!int" {
		return 0, false
	}
	value, ok := parseInt(node.Value)
	if !ok || value <= 0 {
		return 0, false
	}
	return int64(value), true
}

// validateEvidenceDescriptors is the positive-path Evidence Descriptor
// linking gate.
func validateEvidenceDescriptors(req BuildRequest, skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Predicates == nil {
		return
	}
	snapshots := map[string]ResolvedDependency{}
	for _, alias := range identity.Closure {
		for _, dependency := range req.ResolvedDependencies {
			if dependency.DependencyID == alias {
				snapshots[alias] = dependency
			}
		}
	}
	indexes := map[string]snapshotExportIndex{}
	for alias, snapshot := range snapshots {
		index, _, _ := buildSnapshotExportIndex(snapshot)
		indexes[alias] = index
	}
	trusted, trustedProblem := parseTrustedEvidenceBindings(req)
	if trustedProblem != "" {
		add(procedurePath, nil, "EVIDENCE_GUARANTEE_LINK_INVALID", trustedProblem)
		return
	}
	// A trusted Evidence Descriptor document is a structured trusted input: a
	// malformed entry/container must fail the build explicitly instead of being
	// silently dropped by the indexer, even before (or without) a Skill reference.
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
	referenced := map[string]bool{}
	for _, predicateID := range sortedStringMapKeys(skill.Predicates.Definitions) {
		predicate := skill.Predicates.Definitions[predicateID]
		if predicate == nil || predicate.Evidence == nil {
			continue
		}
		requires := unalias(child(predicate.Evidence, "requires"))
		if requires != nil && requires.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(requires.Content); i += 2 {
				for _, guarantee := range sequence(requires.Content[i+1]) {
					if alias, exportID, ok := splitQualifiedReference(guarantee); ok {
						referenced[alias+"::"+exportID] = true
						validateEvidenceDescriptorReference(alias, exportID, indexes, trusted, add, nil)
					}
				}
			}
		}
		for _, group := range sequence(child(predicate.Evidence, "independence")) {
			group = unalias(group)
			if group == nil {
				continue
			}
			for _, member := range sequence(child(group, "members")) {
				member = unalias(member)
				if member == nil {
					continue
				}
				if guarantee := unalias(child(member, "guarantee")); guarantee != nil {
					if alias, exportID, ok := splitQualifiedReference(guarantee); ok {
						validateEvidenceDescriptorReference(alias, exportID, indexes, trusted, add, nil)
					}
				}
			}
			validateEvidenceIndependenceGroup(group, indexes, trusted, add)
		}
	}
	_ = referenced
	validateHostRecoveryDualProducer(skill, identity, ownerLinking, indexes, procedurePath, add)
}

// validateHostRecoveryDualProducer implements 搂17.1: when a reachable Predicate
// consumes <original-mutation-node>.operation.output and that Mutation publishes
// a trusted recoverable output through its dedicated Reconcile READ, every
// required guarantee for that argument must declare both the normal Producer and
// the Reconcile READ Producer. Without a recoverable-output promise the second
// Producer is not required.
func validateHostRecoveryDualProducer(skill *SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, indexes map[string]snapshotExportIndex, procedurePath string, add func(string, *yaml.Node, string, string)) {
	if skill == nil || skill.Procedure == nil || skill.Predicates == nil {
		return
	}
	linked := map[string]linkedExport{}
	ownerByExport := map[string]string{}
	for _, export := range ownerLinking.Exports {
		linked[export.DependencyAlias+"::"+export.ExportID] = export
		owner := export.TrustedOwner
		if owner == "" {
			owner = export.SelfDeclaredOwner
		}
		ownerByExport[export.DependencyAlias+"::"+export.ExportID] = owner
	}
	graph := buildProcedureGraph(skill.Procedure)
	start := scalar(skill.Procedure.Start)
	reachable := reachableProcedureNodes(graph, start)
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		if !reachable[nodeID] {
			continue
		}
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		for _, stage := range []*PredicateStageAST{node.Precondition, node.Postcondition} {
			if stage == nil {
				continue
			}
			for _, call := range stage.Predicates {
				if call == nil {
					continue
				}
				predicate := skill.Predicates.Definitions[scalar(unalias(call.Predicate))]
				if predicate == nil || predicate.Evidence == nil {
					continue
				}
				requires := unalias(child(predicate.Evidence, "requires"))
				if requires == nil || requires.Kind != yaml.MappingNode {
					continue
				}
				parameterIndex := predicateParameterIndex(predicate.Parameters)
				arguments := sequence(call.Arguments)
				for i := 0; i+1 < len(requires.Content); i += 2 {
					parameter := scalar(unalias(requires.Content[i]))
					index, ok := parameterIndex[parameter]
					if !ok || index >= len(arguments) {
						continue
					}
					argument := unalias(arguments[index])
					producer, ok := recoverableMutationProducer(argument, skill, linked)
					if !ok {
						continue
					}
					alias := producer.DependencyAlias
					targets := indexes[alias].callables[producer.MutationFields.ReconcileCallable]
					if len(targets) != 1 {
						continue
					}
					reconcile := targets[0]
					reconcileOwner := ownerByExport[alias+"::"+reconcile.ExportID]
					if reconcileOwner == "" {
						reconcileOwner = reconcile.TrustedOwner
					}
					if reconcileOwner == "" {
						reconcileOwner = reconcile.SelfDeclaredOwner
					}
					normalOwner := ownerByExport[alias+"::"+producer.ExportID]
					if normalOwner == "" {
						normalOwner = producer.SelfDeclaredOwner
					}
					for _, guarantee := range sequence(requires.Content[i+1]) {
						guaranteeAlias, guaranteeID, ok := splitQualifiedReference(guarantee)
						if !ok || guaranteeAlias != alias {
							continue
						}
						descriptors := indexes[alias].evidence[guaranteeID]
						if len(descriptors) != 1 {
							continue
						}
						hasNormal := evidenceDescriptorDeclaresProducer(descriptors[0], producer.ExportID, "operation", normalOwner)
						hasRecovery := evidenceDescriptorDeclaresProducer(descriptors[0], reconcile.ExportID, "operation", reconcileOwner)
						if !hasNormal || !hasRecovery {
							add(procedurePath, argument, "EVIDENCE_GUARANTEE_LINK_INVALID",
								"Host-recoverable Mutation output requires the Evidence guarantee to declare both the normal Producer and the Reconcile READ Producer: "+guaranteeID)
						}
					}
				}
			}
		}
	}
}

// recoverableMutationProducer resolves <node>.operation.output to a linked
// MUTATION export that publishes a trusted recoverable output.
func recoverableMutationProducer(argument *yaml.Node, skill *SkillAST, linked map[string]linkedExport) (linkedExport, bool) {
	argument = unalias(argument)
	if argument == nil || argument.Kind != yaml.ScalarNode || argument.Tag != "!!str" || argument.Style != 0 {
		return linkedExport{}, false
	}
	segments := strings.Split(argument.Value, ".")
	if len(segments) < 3 || segments[1] != "operation" || segments[2] != "output" {
		return linkedExport{}, false
	}
	node := skill.Procedure.ByID[segments[0]]
	if node == nil || node.Operation == nil {
		return linkedExport{}, false
	}
	export, ok := linked[scalar(unalias(node.Operation.Tool))]
	if !ok || export.Effect != "MUTATION" {
		return linkedExport{}, false
	}
	if export.MutationFields.ReconcileCallable == "" || export.MutationFields.RecoverableSchemaID == "" {
		return linkedExport{}, false
	}
	return export, true
}

func evidenceDescriptorDeclaresProducer(descriptor evidenceDescriptor, exportID, kind, owner string) bool {
	for _, producer := range descriptor.Producers {
		if producer.ID == exportID && producer.Kind == kind && producer.Outcome == "SUCCEEDED" && (owner == "" || producer.Owner == owner) {
			return true
		}
	}
	return false
}

func validateEvidenceDescriptorReference(alias, exportID string, indexes map[string]snapshotExportIndex, trusted *trustedEvidenceBindings, add func(string, *yaml.Node, string, string), deferTrust func(string)) {
	index, ok := indexes[alias]
	if !ok {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID", "Evidence guarantee references an unverified dependency alias: "+alias)
		return
	}
	descriptors := index.evidence[exportID]
	if len(descriptors) != 1 {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
			"Evidence Descriptor export must resolve exactly once in the trusted snapshot: "+alias+"::"+exportID)
		return
	}
	descriptor := descriptors[0]
	if descriptor.ShapeProblem != "" {
		// Already reported as a malformed trusted descriptor document.
		return
	}
	if problem := evidenceDescriptorProblem(descriptor); problem != "" {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID", "Evidence Descriptor "+alias+"::"+exportID+": "+problem)
		return
	}
	for _, producer := range descriptor.Producers {
		records := index.callables[producer.ID]
		if len(records) != 1 {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence Descriptor producer does not resolve exactly once in the trusted snapshot: "+producer.Owner+"/"+producer.ID)
			return
		}
		callable := records[0]
		if callable.ExportKind != producer.Kind || callable.TrustedOwner != "" && callable.TrustedOwner != producer.Owner || callable.SelfDeclaredOwner != "" && callable.SelfDeclaredOwner != producer.Owner {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence Descriptor producer identity does not match the trusted ToolCatalog callable: "+producer.ID)
			return
		}
		if producer.Outcome != "SUCCEEDED" {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence Descriptor producer outcome must be SUCCEEDED: "+producer.ID)
			return
		}
		paths := 0
		for _, provide := range callable.EvidenceProvides {
			if provide.ID != exportID {
				continue
			}
			paths++
			if provide.Path != producer.Path {
				add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
					"Evidence Descriptor producer path does not match provides.path for "+exportID)
				return
			}
		}
		if paths != 1 {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence guarantee must be provided exactly once by its trusted ToolCatalog callable: "+exportID)
			return
		}
	}
	if descriptor.SourceIdentity != nil {
		if !validAbsoluteURI(descriptor.SourceIdentity.Domain) || !strings.HasPrefix(descriptor.SourceIdentity.Path, "record.") {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence Descriptor sourceIdentity must declare an absolute domain and a record.* path: "+exportID)
		}
		if trusted == nil {
			if deferTrust != nil {
				deferTrust("sourceIdentity")
			}
		} else if !trusted.hasDomain(descriptor.SourceIdentity.Domain) {
			if deferTrust != nil {
				deferTrust("sourceIdentity")
			} else {
				add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
					"Evidence Descriptor sourceIdentity domain is not authorized by the trusted IndependentDomains input: "+exportID)
			}
		}
	}
	if descriptor.FactIdentity != nil {
		if !trusted.hasCommonFact(descriptor.FactIdentity.Namespace, descriptor.FactIdentity.Fact, descriptor.Authority) {
			if deferTrust != nil {
				deferTrust("factIdentity")
			} else {
				add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
					"Evidence Descriptor factIdentity is not authorized by the trusted CommonFact deployment binding: "+exportID)
			}
		}
	}
}

func evidenceDescriptorProblem(descriptor evidenceDescriptor) string {
	if descriptor.ShapeProblem != "" {
		return descriptor.ShapeProblem
	}
	if descriptor.Authority == "" {
		return "authority is required"
	}
	if !validExportID(descriptor.Fact) {
		return "fact must be a DottedId"
	}
	if len(descriptor.Producers) == 0 {
		return "at least one producer is required"
	}
	if descriptor.FactValue == "" || strings.HasPrefix(descriptor.FactValue, ".") {
		return "factValue must be a static record path"
	}
	if descriptor.Selection.Mode != "exact" && descriptor.Selection.Mode != "latest" {
		return "selection.mode must be exact or latest"
	}
	if descriptor.Selection.Mode == "latest" && descriptor.Selection.OrderBy == "" {
		return "selection.mode latest requires orderBy"
	}
	if descriptor.Freshness.Timestamp == "" || !descriptor.Freshness.HasMaxAge || descriptor.Freshness.MaxAgeMs <= 0 {
		return "freshness requires a timestamp and a positive integer maxAgeMs"
	}
	producerKinds := map[string]bool{}
	for _, producer := range descriptor.Producers {
		producerKinds[producer.Kind] = true
	}
	for _, binding := range append(append([]evidenceBinding{}, descriptor.Scope...), descriptor.Correlation...) {
		if binding.Record == "" || binding.Equals == "" {
			return "scope/correlation bindings require record and equals paths"
		}
		if !validEvidenceReferencePath(binding.Equals) {
			return "scope/correlation equals path must use a trusted reference root: " + binding.Equals
		}
		switch strings.Split(binding.Equals, ".")[0] {
		case "query":
			if !producerKinds["observation"] {
				return "query reference root requires an Observation producer"
			}
		case "arguments":
			if !producerKinds["operation"] {
				return "arguments reference root requires an Operation producer"
			}
		}
	}
	return ""
}

// validEvidenceReferencePath validates the F.3 abstract reference roots:
// query (Observation materialized arguments), arguments (Operation materialized
// arguments) and execution (Runtime-assigned invocation identities).
func validEvidenceReferencePath(path string) bool {
	segments := strings.Split(path, ".")
	if len(segments) < 2 {
		return false
	}
	switch segments[0] {
	case "query", "arguments", "execution", "record":
	default:
		return false
	}
	for _, segment := range segments[1:] {
		if !validDependencyID(segment) {
			return false
		}
	}
	return true
}

func validateEvidenceIndependenceGroup(group *yaml.Node, indexes map[string]snapshotExportIndex, trusted *trustedEvidenceBindings, add func(string, *yaml.Node, string, string)) {
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
		return
	}
	domains := map[string]bool{}
	for _, member := range members {
		index, ok := indexes[member.alias]
		if !ok {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence member references an unverified dependency alias: "+member.alias)
			continue
		}
		descriptors := index.evidence[member.export]
		if len(descriptors) != 1 {
			continue // already reported by the reference check
		}
		source := descriptors[0].SourceIdentity
		if source == nil || source.Domain == "" || !strings.HasPrefix(source.Path, "record.") {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence requires an authorized sourceIdentity (domain + record.* path) for "+member.export)
			continue
		}
		if trusted != nil && !trusted.hasDomain(source.Domain) {
			add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
				"Evidence independence source domain is not authorized by the trusted IndependentDomains input: "+source.Domain)
			continue
		}
		domains[source.Domain] = true
	}
	if len(domains) < 2 {
		return
	}
	if trusted == nil {
		add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
			"Evidence independence across distinct source domains requires a trusted IndependentDomains deployment input")
		return
	}
	domainList := make([]string, 0, len(domains))
	for domain := range domains {
		domainList = append(domainList, domain)
	}
	sort.Strings(domainList)
	for i := 0; i < len(domainList); i++ {
		for j := i + 1; j < len(domainList); j++ {
			if !trusted.independent(domainList[i], domainList[j]) {
				add("", nil, "EVIDENCE_GUARANTEE_LINK_INVALID",
					"IndependentDomains does not authorize the evidence source domain pair: "+domainList[i]+" / "+domainList[j])
			}
		}
	}
}

// trustedEvidenceBindings is the parsed trusted deployment relation input.
type trustedEvidenceBindings struct {
	DeploymentID string
	PublisherID  string
	Domains      map[string]bool
	Pairs        map[string]bool
	CommonFacts  []evidenceFactIdentity
}

func parseTrustedEvidenceBindings(req BuildRequest) (*trustedEvidenceBindings, string) {
	ctx := req.TrustedBuildContext
	if ctx == nil || ctx.EvidenceTrustBindings == nil {
		return nil, ""
	}
	bundle := ctx.EvidenceTrustBindings
	if bundle.DeploymentID == "" || bundle.PublisherID == "" || bundle.SnapshotDigest == "" {
		return nil, "EvidenceTrustBindings bundle is incomplete"
	}
	result := &trustedEvidenceBindings{DeploymentID: bundle.DeploymentID, PublisherID: bundle.PublisherID, Domains: map[string]bool{}, Pairs: map[string]bool{}}
	seen := false
	for _, file := range bundle.Files {
		root, err := yaml.Parse(file.Bytes)
		if err != nil {
			return nil, "EvidenceTrustBindings file is not valid YAML: " + file.Path
		}
		root = unalias(root)
		if root == nil || scalar(child(root, "kind")) != "EvidenceTrustBindings" {
			return nil, "EvidenceTrustBindings file must declare kind EvidenceTrustBindings: " + file.Path
		}
		if scalar(child(root, "deploymentId")) != bundle.DeploymentID || scalar(child(root, "publisherId")) != bundle.PublisherID {
			return nil, "EvidenceTrustBindings file deployment/publisher identity differs from its trusted bundle: " + file.Path
		}
		seen = true
		for _, domainsNode := range sequence(child(root, "independentDomains")) {
			domainsNode = unalias(domainsNode)
			if domainsNode == nil {
				continue
			}
			domains := map[string]bool{}
			for _, domain := range sequence(child(domainsNode, "domains")) {
				value := scalar(unalias(domain))
				if !validAbsoluteURI(value) || domains[value] {
					return nil, "IndependentDomains must declare unique absolute domain URIs"
				}
				domains[value] = true
				result.Domains[value] = true
			}
			if len(domains) < 2 {
				return nil, "IndependentDomains must declare at least two domains"
			}
			for _, pair := range sequence(child(domainsNode, "independentPairs")) {
				entries := sequence(pair)
				if len(entries) != 2 {
					return nil, "IndependentDomains pairs must contain exactly two domains"
				}
				left, right := scalar(unalias(entries[0])), scalar(unalias(entries[1]))
				if left == right || !domains[left] || !domains[right] {
					return nil, "IndependentDomains pair must contain two distinct declared domains"
				}
				if right < left {
					left, right = right, left
				}
				key := left + "\x00" + right
				if result.Pairs[key] {
					return nil, "IndependentDomains pair is duplicated"
				}
				result.Pairs[key] = true
			}
		}
		for _, factNode := range sequence(child(root, "commonFacts")) {
			factNode = unalias(factNode)
			if factNode == nil {
				continue
			}
			identity := evidenceFactIdentity{
				Namespace: scalar(child(factNode, "namespace")),
				Fact:      scalar(child(factNode, "fact")),
				ValuePath: scalar(child(factNode, "comparableValueSchemaId")),
			}
			if !validAbsoluteURI(identity.Namespace) || !validExportID(identity.Fact) || identity.ValuePath == "" {
				return nil, "CommonFact binding must declare namespace, fact and comparableValueSchemaId"
			}
			for _, owner := range sequence(child(factNode, "owners")) {
				identity.Owners = append(identity.Owners, scalar(unalias(owner)))
			}
			if len(identity.Owners) < 2 {
				return nil, "CommonFact binding must authorize at least two Owners"
			}
			result.CommonFacts = append(result.CommonFacts, identity)
		}
	}
	if !seen {
		return nil, "EvidenceTrustBindings bundle contains no relation document"
	}
	return result, ""
}

func (trusted *trustedEvidenceBindings) hasCommonFact(namespace, fact, owner string) bool {
	if trusted == nil {
		return false
	}
	for _, binding := range trusted.CommonFacts {
		if binding.Namespace != namespace || binding.Fact != fact {
			continue
		}
		for _, candidate := range binding.Owners {
			if candidate == owner {
				return true
			}
		}
	}
	return false
}

func (trusted *trustedEvidenceBindings) hasDomain(domain string) bool {
	return trusted != nil && trusted.Domains[domain]
}

func (trusted *trustedEvidenceBindings) independent(left, right string) bool {
	if trusted == nil {
		return false
	}
	if right < left {
		left, right = right, left
	}
	return trusted.Pairs[left+"\x00"+right]
}

func evidenceLetter(r rune) bool { return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') }

func validAbsoluteURI(value string) bool {
	colon := strings.IndexByte(value, ':')
	if colon <= 0 {
		return false
	}
	if !evidenceLetter(rune(value[0])) {
		return false
	}
	for i := 1; i < colon; i++ {
		c := value[i]
		if !evidenceLetter(rune(c)) && !(c >= '0' && c <= '9') && c != '+' && c != '.' && c != '-' {
			return false
		}
	}
	return colon+1 < len(value)
}



