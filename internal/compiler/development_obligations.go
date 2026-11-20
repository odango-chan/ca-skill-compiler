package compiler

// Development Build host-admission requirements.
//
// The Compiler owns the frozen CA-SL Evidence language semantics: declaration and
// reference resolution, Descriptor structure, Producer/Consumer identity,
// parameter binding, SourcePath/data flow, Schema typing and the statically
// decidable Independence and Mutation/Commit-Proof/Reconcile constraints. Those
// checks either pass or fail the Build; they are never deferred.
//
// The following facts cannot be established from Compiler inputs and are owned by
// the Owner, an Assurance Authority and the Host/Runtime:
//
//	OWNER_TRUST                  real Owner identity and signing authority
//	EVIDENCE_DOMAIN_TRUST        sourceIdentity domain authority and real source
//	EVIDENCE_INDEPENDENCE_TRUST  IndependentDomains authorization and real source
//	COMMIT_PROOF                 actual Mutation commitment
//	RECONCILE                    actual recovery conclusion
//
// They are recorded as typed Host Admission requirements with status
// HOST_VERIFICATION_REQUIRED. They are not Compiler obligations and never stand in
// for a missing static contract.

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// DevelopmentRequirementLocation is the exact consumption position.
type DevelopmentRequirementLocation struct {
	Skill    string `json:"skill"`
	Path     string `json:"path"`
	Node     string `json:"node"`
	Stage    string `json:"stage"`
	Selector string `json:"selector"`
}

// DevelopmentRequirementSubject is one package-scoped interface identity.
type DevelopmentRequirementSubject struct {
	Key        DevelopmentPackageKey `json:"key"`
	ID         string                `json:"id"`
	SourcePath string                `json:"sourcePath"`
	SchemaIDs  []string              `json:"schemaIds"`
}

// DevelopmentHostRequirement is one external verification requirement.
type DevelopmentHostRequirement struct {
	RequirementID string                          `json:"requirementId"`
	Kind          string                          `json:"kind"`
	Location      DevelopmentRequirementLocation  `json:"location"`
	Subjects      []DevelopmentRequirementSubject `json:"subjects"`
	Status        string                          `json:"status"`
}

// DevelopmentLinkRecord is one validated interface link with its exact package
// key and declared interface Owner.
type DevelopmentLinkRecord struct {
	Skill      string                `json:"skill"`
	Key        DevelopmentPackageKey `json:"package"`
	ExportKind string                `json:"exportKind"`
	ExportID   string                `json:"exportId"`
	Owner      string                `json:"owner"`
	SourcePath string                `json:"sourcePath"`
	SchemaIDs  []string              `json:"schemaIds"`
}

type developmentRequirementKey struct {
	Kind     string                          `json:"kind"`
	Location DevelopmentRequirementLocation  `json:"location"`
	Subjects []DevelopmentRequirementSubject `json:"subjects"`
}

type developmentRequirementCollector struct {
	byID map[string]DevelopmentHostRequirement
}

func newDevelopmentRequirementCollector() *developmentRequirementCollector {
	return &developmentRequirementCollector{byID: map[string]DevelopmentHostRequirement{}}
}

func (collector *developmentRequirementCollector) add(key developmentRequirementKey) {
	key.Subjects = normalizeDevelopmentRequirementSubjects(key.Subjects)
	canonical, err := canonicalizeValue(key)
	if err != nil {
		return
	}
	sum := sha256.Sum256(append([]byte(developmentRequirementDigestDomain), canonical...))
	requirementID := "sha256:" + hex.EncodeToString(sum[:])
	if _, exists := collector.byID[requirementID]; exists {
		return
	}
	collector.byID[requirementID] = DevelopmentHostRequirement{
		RequirementID: requirementID,
		Kind:          key.Kind,
		Location:      key.Location,
		Subjects:      key.Subjects,
		Status:        DevelopmentHostRequirementStatus,
	}
}

func normalizeDevelopmentRequirementSubjects(subjects []DevelopmentRequirementSubject) []DevelopmentRequirementSubject {
	seen := map[string]DevelopmentRequirementSubject{}
	for _, subject := range subjects {
		if subject.SchemaIDs == nil {
			subject.SchemaIDs = []string{}
		}
		subject.SchemaIDs = dedupeSortedStrings(subject.SchemaIDs)
		canonical, err := canonicalizeValue(subject)
		if err != nil {
			continue
		}
		seen[string(canonical)] = subject
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	normalized := make([]DevelopmentRequirementSubject, 0, len(keys))
	for _, key := range keys {
		normalized = append(normalized, seen[key])
	}
	return normalized
}

func dedupeSortedStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (collector *developmentRequirementCollector) requirements() []DevelopmentHostRequirement {
	ids := make([]string, 0, len(collector.byID))
	for id := range collector.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	requirements := make([]DevelopmentHostRequirement, 0, len(ids))
	for _, id := range ids {
		requirements = append(requirements, collector.byID[id])
	}
	return requirements
}

// validateDevelopmentCapabilityLinking links the Skill Contract capabilities to
// the package-scoped CapabilityCatalog exports. Membership is proven through the
// exact package key and Callable ExportId; a cross-package member must use a
// declared direct dependency alias of the catalog's package.
func validateDevelopmentCapabilityLinking(skill *SkillAST, ownerLinking ownerLinkingResult, scope *developmentLinkScope) capabilityLinkingResult {
	result := capabilityLinkingResult{Diagnostics: []Diagnostic{}, Grants: []string{}, Capabilities: []capabilityDefinition{}}
	if skill == nil || skill.Contract == nil || skill.Contract.Raw == nil {
		return result
	}
	grants, grantDiagnostics := parseSkillCapabilityGrants("skills/"+skill.Name+"/assurance/contract.yaml", skill.Contract.Capabilities)
	result.Grants = grants
	result.Diagnostics = append(result.Diagnostics, grantDiagnostics...)
	granted := map[string]bool{}
	for _, grant := range grants {
		granted[grant] = true
	}
	for _, id := range sortedStringMapKeys(scope.capabilities) {
		for _, capability := range scope.capabilities[id] {
			record := capabilityDefinition{id: capability.id, owner: capability.owner}
			for _, member := range capability.members {
				record.members = append(record.members, capabilityMember{
					owner: member.publisher, callableID: member.callableID, kind: member.kind, effect: member.effect,
				})
			}
			result.Capabilities = append(result.Capabilities, record)
		}
	}
	for _, export := range ownerLinking.Exports {
		if export.ExportKind != "operation" && export.ExportKind != "observation" {
			continue
		}
		matching := make([]capabilityDefinition, 0)
		for _, capability := range result.Capabilities {
			for _, member := range capability.members {
				if member.callableID != export.ExportID || member.kind != export.ExportKind {
					continue
				}
				if member.owner != "" && member.owner != export.TrustedOwner {
					continue
				}
				matching = append(matching, capability)
				break
			}
		}
		if len(matching) == 0 {
			result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "CAPABILITY_MAPPING_MISSING", "", nil,
				"No CapabilityCatalog member matches the used callable identity: "+export.ExportID))
			continue
		}
		grantedMatch := false
		for _, capability := range matching {
			if granted[capability.id] {
				grantedMatch = true
				break
			}
		}
		if !grantedMatch {
			result.Diagnostics = append(result.Diagnostics, dependencyDiagnostic(
				"error", "C_STATIC", "CAPABILITY_NOT_GRANTED", "", nil,
				"No matching Capability is granted by the selected Skill for callable: "+export.ExportID))
		}
	}
	sortDiagnostics(result.Diagnostics)
	return result
}
