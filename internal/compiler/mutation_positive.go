package compiler

// Positive-path static assurance for trusted Mutation / Reconcile contracts.
//
// This implements the released v1alpha4 §9.2.1, §20.2/§20.3 and Appendix
// G.2/G.2.2 obligations that the earlier identity slice only partially checked:
//
//   - commit-proof / failure-class / reconcile proof paths use the exact
//     receipt.<Identifier>(.<Identifier>)* anchor over the *declaring callable's*
//     receiptSchema root, never the proof Schema root;
//   - the extracted proof value must be assignable to the declared proof Schema,
//     and that proof Schema must carry the minimal closed object with
//     originalInvocationId (non-empty string) and commitState
//     [COMMITTED, NOT_COMMITTED];
//   - failure classes must map to declared Owner Outcomes, with FAILED proving
//     no side effect and no retry authorization, and NOT_COMMITTED granting retry
//     only together with noSideEffect;
//   - the dedicated Reconcile READ must be a same-Owner READ operation with an
//     exact originalInvocationId input contract, its own receipt proof path,
//     output.<...> recoverable path equal to the declared Schema identity, and a
//     granted Capability;
//   - the recoverable output Schema must be assignable to the original Mutation
//     output Schema.
//
// Runtime settlement, actual receipt values, retry admission and Reconcile
// dispatch remain Host responsibilities.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

func validateV1Alpha4MutationPositiveContracts(
	req BuildRequest,
	skill *V1Alpha4SkillAST,
	identity dependencyIdentityResult,
	ownerLinking ownerLinkingResult,
	capabilityLinking capabilityLinkingResult,
	procedurePath string,
	add func(string, *yaml.Node, string, string),
) {
	if skill == nil || skill.Contract == nil {
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
	registry := buildV1Alpha4SchemaRegistry(req, skill, identity)
	// Positive-path assurance applies to the Mutation exports the selected Skill
	// actually links; unrelated dependency content is not part of this closure.
	usedMutations := map[string]bool{}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" && export.Effect == "MUTATION" {
			usedMutations[export.DependencyAlias+"::"+export.ExportID] = true
		}
	}
	trustedOwner := map[string]string{}
	for _, export := range ownerLinking.Exports {
		trustedOwner[export.DependencyAlias+"\x00"+export.ExportKind+"\x00"+export.ExportID] = export.TrustedOwner
	}
	report := func(message string) {
		add(procedurePath, nil, "RECONCILE_CONTRACT_INVALID", message)
	}
	for _, alias := range sortedStringMapKeys(snapshots) {
		index, _, _ := buildSnapshotExportIndex(snapshots[alias])
		for _, exportID := range sortedStringMapKeys(index.callables) {
			for _, callable := range index.callables[exportID] {
				if callable.Effect != "MUTATION" || callable.MutationProblem != "" {
					continue
				}
				if !usedMutations[alias+"::"+callable.ExportID] {
					continue
				}
				validateMutationReceiptContract(alias, callable, index, registry, identity, trustedOwner, capabilityLinking, report)
			}
		}
	}
}

func validateMutationReceiptContract(
	alias string,
	callable linkedExport,
	index snapshotExportIndex,
	registry schemaRegistry,
	identity dependencyIdentityResult,
	trustedOwner map[string]string,
	capabilityLinking capabilityLinkingResult,
	report func(string),
) {
	fields := callable.MutationFields
	receiptSchema, ok := resolveSchemaIdentity(registry, alias, fields.ReceiptSchemaID)
	if !ok {
		report("Mutation receiptSchema is not resolvable in the trusted snapshot: " + fields.ReceiptSchemaID)
		return
	}
	// commitProof: receipt anchor -> receiptSchema root -> proof Schema.
	commitProofSchema, ok := resolveSchemaIdentity(registry, alias, fields.CommitProofSchemaID)
	if !ok {
		report("Mutation commitProof Schema is not resolvable: " + fields.CommitProofSchemaID)
		return
	}
	if !minimalCommitProofSchema(commitProofSchema) {
		report("Mutation commitProof Schema must be a closed object with originalInvocationId (non-empty string) and commitState enum [COMMITTED, NOT_COMMITTED]: " + fields.CommitProofSchemaID)
		return
	}
	proofValue, ok := resolveReceiptPath(receiptSchema, fields.CommitProofPath, "receipt")
	if !ok {
		report("Mutation commitProof.path must be an exact receipt.<Identifier>(.<Identifier>)* path declared by receiptSchema: " + fields.CommitProofPath)
		return
	}
	if _, assignable := schemaAssignable(derefSchema(registry, alias, proofValue), commitProofSchema); !assignable {
		report("Mutation commitProof value is not assignable to its proof Schema: " + fields.CommitProofSchemaID)
	}
	// failureClassPath / failureClasses.
	if len(fields.FailureClasses) > 0 {
		if fields.FailureClassPath == "" {
			report("Mutation failureClasses require failureClassPath")
		} else if leaf, ok := resolveReceiptPath(receiptSchema, fields.FailureClassPath, "receipt"); !ok {
			report("Mutation failureClassPath must be an exact receipt.<Identifier>(.<Identifier>)* path declared by receiptSchema: " + fields.FailureClassPath)
		} else if !schemaIsString(leaf) {
			report("Mutation failureClassPath must resolve to a string failure-class id: " + fields.FailureClassPath)
		}
	}
	seenFailureClasses := map[string]bool{}
	for _, class := range fields.FailureClasses {
		switch {
		case class.ID == "":
			report("Mutation failureClasses[].id is required")
			continue
		case seenFailureClasses[class.ID]:
			report("Mutation failureClasses[].id is duplicated: " + class.ID)
			continue
		}
		seenFailureClasses[class.ID] = true
		if class.Outcome != "FAILED" && class.Outcome != "NOT_COMMITTED" {
			report("Mutation failureClasses[].outcome must be FAILED or NOT_COMMITTED: " + class.ID)
			continue
		}
		if !containsString(callable.OwnerOutcomes, class.Outcome) {
			report("Mutation failureClass outcome is not a declared Owner Outcome: " + class.ID)
			continue
		}
		if class.Outcome == "FAILED" && (!class.NoSideEffect || class.Retryable) {
			report("Mutation FAILED failureClass must prove no residual side effect and must not grant retry: " + class.ID)
			continue
		}
		if class.Outcome == "NOT_COMMITTED" && class.Retryable && !class.NoSideEffect {
			report("Mutation NOT_COMMITTED retry requires noSideEffect=true: " + class.ID)
			continue
		}
		classProofSchema, ok := resolveSchemaIdentity(registry, alias, class.ProofSchemaID)
		if !ok {
			report("Mutation failureClass proofSchema is not resolvable: " + class.ID)
			continue
		}
		if !minimalCommitProofSchema(classProofSchema) {
			report("Mutation failureClass proofSchema does not carry the minimal originalInvocationId/commitState proof: " + class.ID)
			continue
		}
		leaf, ok := resolveReceiptPath(receiptSchema, class.ProofPath, "receipt")
		if !ok {
			report("Mutation failureClass proofPath must be an exact receipt.<Identifier>(.<Identifier>)* path declared by receiptSchema: " + class.ID)
			continue
		}
		if _, assignable := schemaAssignable(derefSchema(registry, alias, leaf), classProofSchema); !assignable {
			report("Mutation failureClass proof value is not assignable to its proofSchema: " + class.ID)
		}
	}
	if fields.ReconcileCallable == "" {
		return
	}
	validateMutationReconcileContract(alias, callable, index, registry, identity, trustedOwner, capabilityLinking, report)
}

func validateMutationReconcileContract(
	alias string,
	callable linkedExport,
	index snapshotExportIndex,
	registry schemaRegistry,
	identity dependencyIdentityResult,
	trustedOwner map[string]string,
	capabilityLinking capabilityLinkingResult,
	report func(string),
) {
	fields := callable.MutationFields
	targets := index.callables[fields.ReconcileCallable]
	if len(targets) != 1 {
		report("Mutation Reconcile READ callable must resolve to exactly one same-snapshot operation: " + fields.ReconcileCallable)
		return
	}
	target := targets[0]
	if target.ExportKind != "operation" || target.Effect != "READ" {
		report("Mutation Reconcile callable must be a READ operation: " + fields.ReconcileCallable)
		return
	}
	mutationOwner := trustedOwner[alias+"\x00operation\x00"+callable.ExportID]
	if mutationOwner == "" {
		mutationOwner = callable.TrustedOwner
	}
	if mutationOwner == "" {
		mutationOwner = callable.SelfDeclaredOwner
	}
	targetOwner := trustedOwner[alias+"\x00operation\x00"+target.ExportID]
	if targetOwner == "" {
		targetOwner = target.TrustedOwner
	}
	if targetOwner == "" {
		targetOwner = target.SelfDeclaredOwner
	}
	authorized := map[string]bool{}
	for _, owner := range identity.Bindings[alias].AuthorizedOwners {
		authorized[owner] = true
	}
	if mutationOwner == "" || targetOwner == "" || !authorized[targetOwner] || mutationOwner != targetOwner {
		report("Mutation Reconcile READ must belong to the same trusted Owner as the Mutation: " + fields.ReconcileCallable)
		return
	}
	if !containsString(target.OwnerOutcomes, "SUCCEEDED") || containsString(target.OwnerOutcomes, "NOT_COMMITTED") {
		report("Reconcile READ ownerOutcomes must contain SUCCEEDED and must not declare Runtime-reserved outcomes: " + fields.ReconcileCallable)
	}
	if !reconcileInputContract(registry, alias, target) {
		report("Reconcile READ inputSchema must be a closed object whose only required/allowed property is originalInvocationId (non-empty string): " + fields.ReconcileCallable)
	}
	// Reconcile proof path resolves in the Reconcile callable's own receiptSchema.
	reconcileReceipt, ok := resolveSchemaIdentity(registry, alias, target.ReceiptSchemaID)
	if !ok {
		report("Reconcile READ must publish a resolvable receiptSchema: " + fields.ReconcileCallable)
	} else {
		proofSchema, ok := resolveSchemaIdentity(registry, alias, fields.ReconcileProofSchemaID)
		if !ok {
			report("Mutation Reconcile proofSchema is not resolvable: " + fields.ReconcileProofSchemaID)
		} else if !minimalCommitProofSchema(proofSchema) {
			report("Mutation Reconcile proofSchema does not carry the minimal originalInvocationId/commitState proof: " + fields.ReconcileProofSchemaID)
		} else if leaf, ok := resolveReceiptPath(reconcileReceipt, fields.ReconcileProofPath, "receipt"); !ok {
			report("Mutation Reconcile proofPath must be an exact receipt.<Identifier>(.<Identifier>)* path declared by the Reconcile READ receiptSchema: " + fields.ReconcileProofPath)
		} else if _, assignable := schemaAssignable(derefSchema(registry, alias, leaf), proofSchema); !assignable {
			report("Mutation Reconcile proof value is not assignable to its proofSchema: " + fields.ReconcileProofSchemaID)
		}
	}
	if len(capabilityLinking.Capabilities) > 0 && !capabilityCoversReconcile(capabilityLinking, targetOwner, target) {
		report("Reconcile READ callable must be covered by at least one granted Capability: " + fields.ReconcileCallable)
	}
	if fields.RecoverableSchemaID == "" {
		return
	}
	recoverableSchema, ok := resolveSchemaIdentity(registry, alias, fields.RecoverableSchemaID)
	if !ok {
		report("Mutation recoverableOutputSchema is not resolvable: " + fields.RecoverableSchemaID)
		return
	}
	outputSchema, ok := resolveSchemaIdentity(registry, alias, target.OutputSchemaID)
	if !ok {
		report("Reconcile READ outputSchema is not resolvable: " + target.OutputSchemaID)
		return
	}
	leaf, ok := resolveReceiptPath(outputSchema, fields.RecoverablePath, "output")
	if !ok {
		report("Mutation recoverableOutputPath must be an exact output.<Identifier>(.<Identifier>)* path declared by the Reconcile READ outputSchema: " + fields.RecoverablePath)
		return
	}
	if ref, isRef := schemaRefString(leaf); !isRef || ref != fields.RecoverableSchemaID {
		report("Mutation recoverableOutputPath must reference exactly recoverableOutputSchema: " + fields.RecoverablePath)
		return
	}
	originalOutput, ok := resolveSchemaIdentity(registry, alias, callable.OutputSchemaID)
	if !ok {
		report("Original Mutation outputSchema is not resolvable: " + callable.OutputSchemaID)
		return
	}
	if _, assignable := schemaAssignable(recoverableSchema, originalOutput); !assignable {
		report("Recoverable output Schema is not assignable to the original Mutation output Schema: " + fields.RecoverableSchemaID)
		return
	}
}

// reconcileInputContract proves the exact closed originalInvocationId input.
func reconcileInputContract(registry schemaRegistry, alias string, target linkedExport) bool {
	input, ok := resolveSchemaIdentity(registry, alias, target.InputSchemaID)
	if !ok || input == nil || input.Kind != yaml.MappingNode {
		return false
	}
	if scalar(child(input, "type")) != "object" {
		return false
	}
	if scalar(child(input, "additionalProperties")) != "false" {
		return false
	}
	required := sequence(child(input, "required"))
	if len(required) != 1 || scalar(required[0]) != "originalInvocationId" {
		return false
	}
	properties := child(input, "properties")
	if properties == nil || len(properties.Content) != 2 || scalar(unalias(properties.Content[0])) != "originalInvocationId" {
		return false
	}
	property := unalias(properties.Content[1])
	if scalar(child(property, "type")) != "string" {
		return false
	}
	minimum, ok := intValue(child(property, "minLength"))
	return ok && minimum >= 1
}

// capabilityCoversReconcile proves the dedicated READ is a member of at least
// one Capability granted by the Skill Contract with the exact Owner/kind/Effect.
func capabilityCoversReconcile(linking capabilityLinkingResult, owner string, target linkedExport) bool {
	granted := map[string]bool{}
	for _, id := range linking.Grants {
		granted[id] = true
	}
	for _, capability := range linking.Capabilities {
		if !granted[capability.id] {
			continue
		}
		for _, member := range capability.members {
			if member.owner == owner && member.callableID == target.ExportID &&
				member.kind == "operation" && member.effect == "READ" {
				return true
			}
		}
	}
	return false
}

// resolveSchemaIdentity resolves a trusted Schema reference and follows the root
// $ref chain, returning the effective root Schema node.
func resolveSchemaIdentity(registry schemaRegistry, alias, schemaID string) (*yaml.Node, bool) {
	if schemaID == "" {
		return nil, false
	}
	resource, problem := registry.resolveReference(alias + "::" + schemaID)
	if problem != "" {
		return nil, false
	}
	resolved, problem := registry.resolveSchemaRoot(resource)
	if problem != "" {
		return nil, false
	}
	node := unalias(resolved.node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, false
	}
	return node, true
}

// derefSchema follows local $ref targets inside the same trusted snapshot so the
// extracted proof value is compared by its effective Schema identity.
func derefSchema(registry schemaRegistry, alias string, node *yaml.Node) *yaml.Node {
	for hops := 0; hops < 16; hops++ {
		ref, ok := schemaRefString(node)
		if !ok {
			return unalias(node)
		}
		resource, problem := registry.resolveReference(alias + "::" + ref)
		if problem != "" {
			return unalias(node)
		}
		resolved, problem := registry.resolveSchemaRoot(resource)
		if problem != "" {
			return unalias(node)
		}
		node = resolved.node
	}
	return unalias(node)
}

func resolveReceiptPath(root *yaml.Node, path, anchor string) (*yaml.Node, bool) {
	segments := splitDottedPath(path)
	if len(segments) < 2 || segments[0] != anchor {
		return nil, false
	}
	node := root
	for _, segment := range segments[1:] {
		if !validDependencyID(segment) {
			return nil, false
		}
		properties := child(node, "properties")
		if properties == nil {
			return nil, false
		}
		node = unalias(child(properties, segment))
		if node == nil {
			return nil, false
		}
	}
	return node, true
}

// minimalCommitProofSchema is the G.2.2 minimal closed commit proof object.
func minimalCommitProofSchema(node *yaml.Node) bool {
	if node == nil || scalar(child(node, "type")) != "object" {
		return false
	}
	additional := child(node, "additionalProperties")
	if additional == nil || !(additional.Kind == yaml.ScalarNode && additional.Value == "false") {
		return false
	}
	required := map[string]bool{}
	for _, entry := range sequence(child(node, "required")) {
		required[scalar(entry)] = true
	}
	if !required["originalInvocationId"] || !required["commitState"] {
		return false
	}
	properties := child(node, "properties")
	invocation := unalias(child(properties, "originalInvocationId"))
	if !schemaIsNonEmptyString(invocation) {
		return false
	}
	commitState := unalias(child(properties, "commitState"))
	if scalar(child(commitState, "type")) != "string" {
		return false
	}
	values := map[string]bool{}
	for _, entry := range sequence(child(commitState, "enum")) {
		values[scalar(entry)] = true
	}
	return values["COMMITTED"] && values["NOT_COMMITTED"] && len(values) == 2
}

func schemaIsString(node *yaml.Node) bool {
	return scalar(child(node, "type")) == "string"
}

func schemaIsNonEmptyString(node *yaml.Node) bool {
	if !schemaIsString(node) {
		return false
	}
	if minimum, ok := intValue(child(node, "minLength")); ok && minimum >= 1 {
		return true
	}
	if pattern := scalar(child(node, "pattern")); pattern != "" {
		return true
	}
	for _, entry := range sequence(child(node, "enum")) {
		if scalar(entry) != "" {
			return true
		}
	}
	return false
}

func schemaRefString(node *yaml.Node) (string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return "", false
	}
	ref := scalar(child(node, "$ref"))
	return ref, ref != ""
}

func intValue(node *yaml.Node) (int, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode {
		return 0, false
	}
	value, ok := parseInt(node.Value)
	return value, ok
}

func parseInt(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}
	value := 0
	for _, digit := range raw {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		value = value*10 + int(digit-'0')
		if value > 1<<30 {
			return 0, false
		}
	}
	return value, true
}
