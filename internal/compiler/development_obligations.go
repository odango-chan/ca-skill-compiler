package compiler

// Development Build assurance obligations.
//
// The Development profile performs every statically decidable language, type,
// control-flow and interface check first. Only facts that require an external
// authority are deferred:
//
//	OWNER_TRUST            declared Owner identity of a used cross-package export
//	EVIDENCE_GUARANTEE     Predicate evidence guarantee and its Producer paths
//	EVIDENCE_INDEPENDENCE  independence group members at a consumption point
//	COMMIT_PROOF           Mutation commit-proof authority
//	RECONCILE              Mutation Reconcile authority and recovery binding
//
// A missing external material produces an UNRESOLVED obligation. A supplied
// Evidence Descriptor that statically matches produces DESCRIPTOR_PRESENT. A
// malformed, unrelated or contradictory descriptor fails the build; the checks
// that are statically decidable are never filtered out by error code.

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// DevelopmentObligationLocation is the exact consumption position.
type DevelopmentObligationLocation struct {
	Skill    string `json:"skill"`
	Path     string `json:"path"`
	Node     string `json:"node"`
	Stage    string `json:"stage"`
	Selector string `json:"selector"`
}

// DevelopmentObligationSubject is one package-scoped interface identity.
type DevelopmentObligationSubject struct {
	Key        DevelopmentPackageKey `json:"key"`
	ID         string                `json:"id"`
	SourcePath string                `json:"sourcePath"`
	SchemaIDs  []string              `json:"schemaIds"`
}

// DevelopmentObligation is one typed assurance obligation.
type DevelopmentObligation struct {
	ObligationID     string                         `json:"obligationId"`
	Rule             string                         `json:"rule"`
	Location         DevelopmentObligationLocation  `json:"location"`
	Subjects         []DevelopmentObligationSubject `json:"subjects"`
	Status           string                         `json:"status"`
	DescriptorDigest string                         `json:"descriptorDigest,omitempty"`
}

type developmentObligationKey struct {
	Rule     string                         `json:"rule"`
	Location DevelopmentObligationLocation  `json:"location"`
	Subjects []DevelopmentObligationSubject `json:"subjects"`
}

type developmentObligationCollector struct {
	byID            map[string]DevelopmentObligation
	usedDescriptors map[string]bool
}

func newDevelopmentObligationCollector() *developmentObligationCollector {
	return &developmentObligationCollector{byID: map[string]DevelopmentObligation{}, usedDescriptors: map[string]bool{}}
}

func (collector *developmentObligationCollector) add(key developmentObligationKey, status, descriptorDigest string) {
	key.Subjects = normalizeDevelopmentSubjects(key.Subjects)
	canonical, err := canonicalizeValue(key)
	if err != nil {
		return
	}
	sum := sha256.Sum256(append([]byte(developmentObligationDigestDomain), canonical...))
	obligationID := "sha256:" + hex.EncodeToString(sum[:])
	obligation := DevelopmentObligation{
		ObligationID: obligationID, Rule: key.Rule, Location: key.Location,
		Subjects: key.Subjects, Status: status, DescriptorDigest: descriptorDigest,
	}
	if existing, exists := collector.byID[obligationID]; exists {
		// A present descriptor always dominates an unresolved duplicate at the
		// same identity; otherwise the first record wins.
		if existing.Status == DevelopmentObligationDescriptor {
			return
		}
		if status != DevelopmentObligationDescriptor {
			return
		}
	}
	collector.byID[obligationID] = obligation
}

func (collector *developmentObligationCollector) markDescriptorUsed(identity string) {
	collector.usedDescriptors[identity] = true
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

func normalizeDevelopmentSubjects(subjects []DevelopmentObligationSubject) []DevelopmentObligationSubject {
	seen := map[string]DevelopmentObligationSubject{}
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
	normalized := make([]DevelopmentObligationSubject, 0, len(keys))
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

func (collector *developmentObligationCollector) obligations() []DevelopmentObligation {
	ids := make([]string, 0, len(collector.byID))
	for id := range collector.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	obligations := make([]DevelopmentObligation, 0, len(ids))
	for _, id := range ids {
		obligations = append(obligations, collector.byID[id])
	}
	return obligations
}

// developmentAssuranceIndex indexes the optional Assurance Export descriptors by
// package key and export ID. The same export ID must resolve exactly once across
// every supplied descriptor file.
type developmentAssuranceIndex struct {
	byKey map[DevelopmentPackageKey]map[string][]evidenceDescriptor
	// digestByKey is the Assurance Export digest that carries each package.
	digestByKey map[DevelopmentPackageKey]string
	pathByKey   map[DevelopmentPackageKey]map[string]string
}

func buildDevelopmentAssuranceIndex(env *developmentEnvironment) (*developmentAssuranceIndex, []Diagnostic) {
	index := &developmentAssuranceIndex{
		byKey:       map[DevelopmentPackageKey]map[string][]evidenceDescriptor{},
		digestByKey: map[DevelopmentPackageKey]string{},
		pathByKey:   map[DevelopmentPackageKey]map[string]string{},
	}
	var diagnostics []Diagnostic
	add := func(code, path, message string) {
		diagnostics = append(diagnostics, developmentDiagnostic(code, path, message))
	}
	for _, key := range sortedDevelopmentKeys(env.packages) {
		assurance, exists := env.assuranceByKey[key]
		if !exists {
			continue
		}
		index.digestByKey[key] = assurance.Digest
		index.byKey[key] = map[string][]evidenceDescriptor{}
		index.pathByKey[key] = map[string]string{}
		files := append([]File(nil), assurance.Files...)
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		for _, file := range files {
			if !strings.HasSuffix(file.Path, ".yaml") && !strings.HasSuffix(file.Path, ".yml") && !strings.HasSuffix(file.Path, ".json") {
				continue
			}
			root, err := yaml.Parse(file.Bytes)
			if err != nil {
				add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "Assurance Export file is not valid YAML/JSON")
				continue
			}
			descriptors, ok := parseEvidenceDescriptors(root)
			if !ok {
				add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "Assurance Export file does not carry Evidence Descriptor entries")
				continue
			}
			for _, exportID := range sortedStringMapKeys(descriptors) {
				for position := range descriptors[exportID] {
					descriptors[exportID][position].SnapshotPath = file.Path
				}
				if len(index.byKey[key][exportID]) > 0 {
					add(developmentAssuranceInvalid, key.PackageID+"/"+file.Path, "Evidence Descriptor export is declared more than once in the Assurance Export: "+exportID)
					continue
				}
				index.byKey[key][exportID] = descriptors[exportID]
				index.pathByKey[key][exportID] = file.Path
			}
		}
	}
	sortDiagnostics(diagnostics)
	return index, diagnostics
}

// developmentEvidenceStatus returns whether one guarantee reference has a
// supplied descriptor and the Assurance Export digest that carries it.
func (index *developmentAssuranceIndex) lookup(key DevelopmentPackageKey, exportID string) (bool, string) {
	descriptors, exists := index.byKey[key][exportID]
	if !exists || len(descriptors) != 1 {
		return false, ""
	}
	return true, index.digestByKey[key]
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
		key, exists := scope.env.rootByAlias[export.DependencyAlias]
		if !exists {
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
		_ = key
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
