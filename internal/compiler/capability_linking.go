package compiler

// WP3C: trusted CapabilityContracts deployment input, exact callable membership,
// Effect consistency and Scope-policy presence.
//
// CapabilityContracts is deployment-side trusted input, independent of ordinary
// DependencySnapshots. This file never infers authorization from Skill source or
// ToolCatalog, and never invents a Scope policy format. The external Scope rule
// content is a separate deployment publication; when it is unavailable the
// compiler can only require the exact trusted reference and must not claim a
// wider authorization.

import (
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const (
	capabilityContractsAPIVersion = "connectionagent.io/capability-contracts/v1alpha4"
	capabilityContractsKind       = "CapabilityContracts"
	maxCapabilityIDRunes          = 240
	maxScopePolicyRefRunes        = 512
)

type capabilityMember struct {
	owner          string
	callableID     string
	kind           string
	effect         string
	scopePolicyRef string
}

type capabilityDefinition struct {
	id      string
	owner   string
	members []capabilityMember
}

type capabilityLinkingResult struct {
	Diagnostics  []Diagnostic
	Grants       []string
	Capabilities []capabilityDefinition
	// Requirements carries the Development-profile static Capability
	// requirements that enter the final Artifact. The frozen v1alpha5 profile
	// leaves it empty: its Artifact contract is unchanged.
	Requirements []DevelopmentCapabilityRequirement
}

func validateCapabilityLinking(req BuildRequest, skill *SkillAST, ownerLinking ownerLinkingResult) capabilityLinkingResult {
	result := capabilityLinkingResult{Diagnostics: []Diagnostic{}, Grants: []string{}, Capabilities: []capabilityDefinition{}}
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return result
	}
	grants, grantDiagnostics := parseSkillCapabilityGrants("skills/"+skill.Name+"/assurance/contract.yaml", skill.Contract.Capabilities)
	result.Grants = grants
	result.Diagnostics = append(result.Diagnostics, grantDiagnostics...)

	callables := make([]linkedExport, 0, len(ownerLinking.Exports))
	for _, export := range ownerLinking.Exports {
		if export.ExportKind == "operation" || export.ExportKind == "observation" {
			if export.TrustedOwner != "" {
				callables = append(callables, export)
			}
		}
	}
	if len(callables) == 0 {
		return result
	}
	var bundle *CapabilityContractsBundle
	var authorizations []DeploymentAuthorization
	if req.TrustedBuildContext != nil {
		bundle = req.TrustedBuildContext.CapabilityContracts
		if req.TrustedBuildContext.DeploymentAuthorizations != nil {
			authorizations = *req.TrustedBuildContext.DeploymentAuthorizations
		}
	}
	if bundle == nil {
		result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "CAPABILITY_INPUT_UNTRUSTED", "", nil,
			"A selected Skill references external callables but no trusted CapabilityContracts deployment input is supplied"))
		return result
	}
	if !deploymentAuthorized(bundle, authorizations) {
		result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "CAPABILITY_INPUT_UNTRUSTED", "", nil,
			"The trusted deployment authorization does not bind the CapabilityContracts deployment and publisher"))
		return result
	}
	capabilities, parseDiagnostics := parseCapabilityContracts(bundle.Files)
	result.Diagnostics = append(result.Diagnostics, parseDiagnostics...)
	if len(capabilities) == 0 {
		result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "CAPABILITY_INPUT_UNTRUSTED", "", nil,
			"The trusted CapabilityContracts artifact set does not contain a valid v1alpha4 CapabilityContracts document"))
		return result
	}
	result.Capabilities = capabilities
	granted := make(map[string]bool, len(grants))
	for _, grant := range grants {
		granted[grant] = true
	}
	for _, callable := range callables {
		result.Diagnostics = append(result.Diagnostics, checkCallableCapability(callable, capabilities, granted)...)
	}
	sortDiagnostics(result.Diagnostics)
	return result
}

func deploymentAuthorized(bundle *CapabilityContractsBundle, authorizations []DeploymentAuthorization) bool {
	for _, authorization := range authorizations {
		if authorization.DeploymentID == bundle.DeploymentID && authorization.PublisherID == bundle.PublisherID {
			return true
		}
	}
	return false
}

func parseSkillCapabilityGrants(contractPath string, node *yaml.Node) ([]string, []Diagnostic) {
	grants := make([]string, 0)
	diagnostics := make([]Diagnostic, 0)
	node = unalias(node)
	if node == nil {
		return grants, diagnostics
	}
	if node.Kind != yaml.SequenceNode {
		// A malformed Grant list grants nothing; every referenced callable then
		// reports the frozen CAPABILITY_NOT_GRANTED condition.
		return grants, diagnostics
	}
	seen := make(map[string]bool, len(node.Content))
	for _, item := range node.Content {
		item = unalias(item)
		value := scalar(item)
		if !validExportID(value) || len([]rune(value)) > maxCapabilityIDRunes {
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "IDENTIFIER_INVALID", contractPath, item,
				"Contract capabilities entry must be a valid CapabilityId (DottedId): "+value))
			continue
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		grants = append(grants, value)
	}
	sort.Strings(grants)
	return grants, diagnostics
}

func parseCapabilityContracts(files []File) ([]capabilityDefinition, []Diagnostic) {
	diagnostics := make([]Diagnostic, 0)
	byID := make(map[string]*capabilityDefinition)
	ordered := make([]File, len(files))
	copy(ordered, files)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, file := range ordered {
		if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") && !strings.HasSuffix(file.Path, ".json") {
			continue
		}
		root, err := yaml.Parse(file.Bytes)
		if err != nil {
			continue
		}
		root = unalias(root)
		if root == nil || root.Kind != yaml.MappingNode {
			continue
		}
		if scalar(child(root, "apiVersion")) != capabilityContractsAPIVersion || scalar(child(root, "kind")) != capabilityContractsKind {
			continue
		}
		for _, capabilityNode := range sequence(child(child(root, "spec"), "capabilities")) {
			capability, ok := parseCapability(capabilityNode)
			if !ok {
				continue
			}
			if existing, exists := byID[capability.id]; exists {
				diagnostics = append(diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "CAPABILITY_DEFINITION_CONFLICT", "", nil,
					"Capability ID is defined more than once in the trusted artifact set: "+capability.id))
				_ = existing
				continue
			}
			byID[capability.id] = &capability
			diagnostics = append(diagnostics, duplicateMembershipDiagnostics(capability)...)
		}
	}
	ids := sortedStringMapKeys(byID)
	capabilities := make([]capabilityDefinition, 0, len(ids))
	for _, id := range ids {
		capabilities = append(capabilities, *byID[id])
	}
	return capabilities, diagnostics
}

func parseCapability(node *yaml.Node) (capabilityDefinition, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return capabilityDefinition{}, false
	}
	id := scalar(child(node, "id"))
	owner := scalar(child(node, "owner"))
	if !validExportID(id) || len([]rune(id)) > maxCapabilityIDRunes || owner == "" || len([]rune(owner)) > maxOwnerIDRunes {
		return capabilityDefinition{}, false
	}
	members := sequence(child(node, "callables"))
	if len(members) == 0 {
		return capabilityDefinition{}, false
	}
	capability := capabilityDefinition{id: id, owner: owner, members: make([]capabilityMember, 0, len(members))}
	for _, memberNode := range members {
		member, ok := parseCapabilityMember(memberNode)
		if !ok {
			continue
		}
		capability.members = append(capability.members, member)
	}
	return capability, true
}

func parseCapabilityMember(node *yaml.Node) (capabilityMember, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return capabilityMember{}, false
	}
	member := capabilityMember{
		owner:      scalar(child(node, "owner")),
		callableID: scalar(child(node, "id")),
		kind:       scalar(child(node, "kind")),
		effect:     scalar(child(node, "effect")),
	}
	if member.owner == "" || len([]rune(member.owner)) > maxOwnerIDRunes ||
		!validExportID(member.callableID) || len([]rune(member.callableID)) > maxExportIDRunes {
		return capabilityMember{}, false
	}
	if member.kind != "operation" && member.kind != "observation" {
		return capabilityMember{}, false
	}
	if member.effect != "READ" && member.effect != "MUTATION" {
		return capabilityMember{}, false
	}
	if node := child(node, "scopePolicyRef"); node != nil {
		member.scopePolicyRef = scalar(node)
		if member.scopePolicyRef == "" || len([]rune(member.scopePolicyRef)) > maxScopePolicyRefRunes {
			return capabilityMember{}, false
		}
	}
	return member, true
}

func duplicateMembershipDiagnostics(capability capabilityDefinition) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	seen := make(map[string]bool, len(capability.members))
	for _, member := range capability.members {
		key := member.owner + "\x00" + member.callableID + "\x00" + member.kind
		if seen[key] {
			diagnostics = append(diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "CAPABILITY_MEMBERSHIP_DUPLICATE", "", nil,
				"Capability declares the same callable member more than once: "+capability.id+" "+member.callableID))
			continue
		}
		seen[key] = true
	}
	return diagnostics
}

func checkCallableCapability(callable linkedExport, capabilities []capabilityDefinition, granted map[string]bool) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	matching := make([]capabilityDefinition, 0)
	for _, capability := range capabilities {
		for _, member := range capability.members {
			if member.owner != callable.TrustedOwner || member.callableID != callable.ExportID || member.kind != callable.ExportKind {
				continue
			}
			matching = append(matching, capability)
			if callable.Effect != "" && member.effect != callable.Effect {
				diagnostics = append(diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "EFFECT_MISMATCH", "", nil,
					"Capability member Effect differs from the trusted ToolCatalog Effect for "+callable.ExportID))
			}
			break
		}
	}
	if len(matching) == 0 {
		diagnostics = append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "CAPABILITY_MAPPING_MISSING", "", nil,
			"No CapabilityContracts member record matches the trusted callable Owner, ID and kind: "+callable.ExportID))
		return diagnostics
	}
	grantedMatch := false
	for _, capability := range matching {
		if !granted[capability.id] {
			continue
		}
		grantedMatch = true
		for _, member := range capability.members {
			if member.owner != callable.TrustedOwner || member.callableID != callable.ExportID || member.kind != callable.ExportKind {
				continue
			}
			if member.scopePolicyRef == "" {
				diagnostics = append(diagnostics, dependencyDiagnostic(
					"error", "C_STATIC", "CAPABILITY_SCOPE_UNRESOLVED", "", nil,
					"Granted Capability member has no trusted resolvable Scope policy reference: "+capability.id+" "+callable.ExportID))
			}
			break
		}
	}
	if !grantedMatch {
		diagnostics = append(diagnostics, dependencyDiagnostic(
			"error", "C_STATIC", "CAPABILITY_NOT_GRANTED", "", nil,
			"No matching Capability is granted by the selected Skill for callable: "+callable.ExportID))
	}
	return diagnostics
}
