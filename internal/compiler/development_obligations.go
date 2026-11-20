package compiler

// Development Build requirement records and interface-link helpers.
//
// The Compiler owns the frozen CA-SL Evidence language semantics: declaration and
// reference resolution, Descriptor structure, Producer/Consumer identity,
// parameter binding, SourcePath/data flow, Schema typing and the statically
// decidable Independence and Mutation/Commit-Proof/Reconcile constraints. Those
// checks either pass or fail the Build; they are never deferred.
//
// Two disjoint requirement classes describe what the Compiler cannot establish:
//
// Host Admission Requirement (host-admission-requirements.json,
// HOST_VERIFICATION_REQUIRED) records external pre-dispatch facts:
//
//	OWNER_TRUST                  real Owner identity and signing authority
//	EVIDENCE_DOMAIN_TRUST        sourceIdentity domain authority and real source
//	EVIDENCE_FACT_TRUST          CommonFact authorization
//	EVIDENCE_INDEPENDENCE_TRUST  IndependentDomains authorization for each group
//
// Runtime Execution Requirement (runtime-execution-requirements.json,
// RUNTIME_EXECUTION_REQUIRED) records execution-time guarantees that no Build or
// admission step can satisfy:
//
//	COMMIT_PROOF                 actual Mutation commitment and proof verification
//	RECONCILE                    actual exact-original recovery conclusion
//
// Neither class is a Compiler obligation and neither ever stands in for a
// missing static contract.

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

// DevelopmentIndependenceMember is one exact Independence Group member: the
// declared dependency alias, the exact Interface Package key, the guarantee
// export ID, the locked Descriptor snapshot path and the declared Source Domain.
type DevelopmentIndependenceMember struct {
	DependencyAlias string                `json:"dependencyAlias"`
	Key             DevelopmentPackageKey `json:"key"`
	ExportID        string                `json:"exportId"`
	SourcePath      string                `json:"sourcePath"`
	Domain          string                `json:"domain"`
}

// DevelopmentIndependenceGroup is the exact Independence evidence of one
// source-level group. Each group and each Domain Pair is modelled separately:
// no single boolean and no env.roots[0]-style identity stand-in is permitted.
type DevelopmentIndependenceGroup struct {
	// Predicate identifies the source Predicate for the requirement location. It
	// is already carried by location.node and is deliberately not serialized: the
	// contract fixes the independence detail to
	// {groupIndex,members,domainPairs}.
	Predicate   string                          `json:"-"`
	GroupIndex  int                             `json:"groupIndex"`
	Members     []DevelopmentIndependenceMember `json:"members"`
	DomainPairs [][]string                      `json:"domainPairs"`
}

// DevelopmentHostRequirement is one external verification requirement. Host
// Admission requirements and Runtime Execution requirements share the record
// shape but never share an artifact, a status or a satisfaction point.
type DevelopmentHostRequirement struct {
	RequirementID string                          `json:"requirementId"`
	Kind          string                          `json:"kind"`
	Location      DevelopmentRequirementLocation  `json:"location"`
	Subjects      []DevelopmentRequirementSubject `json:"subjects"`
	Independence  *DevelopmentIndependenceGroup   `json:"independence,omitempty"`
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
	Kind         string                          `json:"kind"`
	Location     DevelopmentRequirementLocation  `json:"location"`
	Subjects     []DevelopmentRequirementSubject `json:"subjects"`
	Independence *DevelopmentIndependenceGroup   `json:"independence,omitempty"`
}

type developmentRequirementCollector struct {
	host    map[string]DevelopmentHostRequirement
	runtime map[string]DevelopmentHostRequirement
}

func newDevelopmentRequirementCollector() *developmentRequirementCollector {
	return &developmentRequirementCollector{
		host:    map[string]DevelopmentHostRequirement{},
		runtime: map[string]DevelopmentHostRequirement{},
	}
}

func (collector *developmentRequirementCollector) add(key developmentRequirementKey) {
	normalized := normalizeDevelopmentRequirementKey(key)
	canonical, err := canonicalizeValue(normalized)
	if err != nil {
		return
	}
	sum := sha256.Sum256(append([]byte(developmentRequirementDigestDomain), canonical...))
	requirementID := "sha256:" + hex.EncodeToString(sum[:])
	requirement := DevelopmentHostRequirement{
		RequirementID: requirementID,
		Kind:          normalized.Kind,
		Location:      normalized.Location,
		Subjects:      normalized.Subjects,
		Independence:  normalized.Independence,
		Status:        developmentRequirementStatus(normalized.Kind),
	}
	switch developmentRequirementClass(normalized.Kind) {
	case "host":
		if _, exists := collector.host[requirementID]; exists {
			return
		}
		collector.host[requirementID] = requirement
	case "runtime":
		if _, exists := collector.runtime[requirementID]; exists {
			return
		}
		collector.runtime[requirementID] = requirement
	}
}

// normalizeDevelopmentRequirementKey produces the single canonical requirement
// key used both when recording and when re-deriving an artifact requirement ID.
func normalizeDevelopmentRequirementKey(key developmentRequirementKey) developmentRequirementKey {
	key.Subjects = normalizeDevelopmentRequirementSubjects(key.Subjects)
	key.Independence = normalizeDevelopmentIndependenceGroup(key.Independence)
	return key
}

func normalizeDevelopmentIndependenceGroup(group *DevelopmentIndependenceGroup) *DevelopmentIndependenceGroup {
	if group == nil {
		return nil
	}
	normalized := &DevelopmentIndependenceGroup{GroupIndex: group.GroupIndex, Predicate: group.Predicate}
	seen := map[string]DevelopmentIndependenceMember{}
	for _, member := range group.Members {
		canonical, err := canonicalizeValue(member)
		if err != nil {
			continue
		}
		seen[string(canonical)] = member
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	normalized.Members = make([]DevelopmentIndependenceMember, 0, len(keys))
	for _, key := range keys {
		normalized.Members = append(normalized.Members, seen[key])
	}
	pairs := map[string][]string{}
	for _, pair := range group.DomainPairs {
		if len(pair) != 2 || pair[0] == "" || pair[1] == "" || pair[0] == pair[1] {
			continue
		}
		left, right := pair[0], pair[1]
		if right < left {
			left, right = right, left
		}
		pairs[left+"\x00"+right] = []string{left, right}
	}
	pairKeys := make([]string, 0, len(pairs))
	for key := range pairs {
		pairKeys = append(pairKeys, key)
	}
	sort.Strings(pairKeys)
	normalized.DomainPairs = make([][]string, 0, len(pairKeys))
	for _, key := range pairKeys {
		normalized.DomainPairs = append(normalized.DomainPairs, pairs[key])
	}
	return normalized
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

func (collector *developmentRequirementCollector) hostRequirements() []DevelopmentHostRequirement {
	return sortedDevelopmentRequirements(collector.host)
}

func (collector *developmentRequirementCollector) runtimeRequirements() []DevelopmentHostRequirement {
	return sortedDevelopmentRequirements(collector.runtime)
}

func sortedDevelopmentRequirements(byID map[string]DevelopmentHostRequirement) []DevelopmentHostRequirement {
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	requirements := make([]DevelopmentHostRequirement, 0, len(ids))
	for _, id := range ids {
		requirements = append(requirements, byID[id])
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
