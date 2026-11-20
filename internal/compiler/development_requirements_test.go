package compiler

// Development Build final-architecture tests: the two disjoint requirement
// classes, multi-requirement Evidence, exact Independence-group modelling and
// precise IngressCatalog export identity.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// developmentSourceAndFactDescriptor appends both external identities to the
// released Evidence Descriptor used by the Development fixture.
func developmentDescriptorWithIdentities(t *testing.T, includeSource, includeFact bool) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "tests", "conformance", "v1alpha4", "fixtures", "dependency-identity", "base", "snapshots", "ca", "evidence-guarantees.yaml"))
	if err != nil {
		t.Fatalf("read evidence descriptor fixture: %v", err)
	}
	content := append([]byte(nil), raw...)
	if includeSource {
		content = append(content, []byte("  sourceIdentity: {domain: urn:example:authority:one, path: record.source}\n")...)
	}
	if includeFact {
		content = append(content, []byte("  factIdentity: {namespace: urn:example:authority:facts, fact: ue.registration, valuePath: record.factValue, scopePaths: {ue: record.scope}}\n")...)
	}
	return content
}

func buildDevelopmentWithIdentities(t *testing.T, includeSource, includeFact bool) DevelopmentBuildResult {
	t.Helper()
	request := modifyDevelopmentPackageFile(t, loadDevelopmentFixture(t), "urn:example:ca:tools", "evidence-guarantees.yaml",
		developmentDescriptorWithIdentities(t, includeSource, includeFact))
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("statically valid descriptor (source=%v fact=%v) must build: %#v", includeSource, includeFact, result.Diagnostics)
	}
	return result
}

// TestDevelopmentEvidenceTrustRequirementsAreIndependent proves that a single
// Descriptor declaring both sourceIdentity and factIdentity produces both Host
// Admission requirements and that either one alone produces only its own kind.
func TestDevelopmentEvidenceTrustRequirementsAreIndependent(t *testing.T) {
	cases := []struct {
		name         string
		source, fact bool
	}{
		{name: "source only", source: true},
		{name: "fact only", fact: true},
		{name: "both", source: true, fact: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := buildDevelopmentWithIdentities(t, testCase.source, testCase.fact)
			_, hasDomain := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceDomain, "ue.registration-state")
			_, hasFact := developmentRequirementByKind(t, result, DevelopmentRequirementEvidenceFact, "ue.registration-state")
			if hasDomain != testCase.source {
				t.Fatalf("EVIDENCE_DOMAIN_TRUST presence = %v, want %v", hasDomain, testCase.source)
			}
			if hasFact != testCase.fact {
				t.Fatalf("EVIDENCE_FACT_TRUST presence = %v, want %v", hasFact, testCase.fact)
			}
			if testCase.source && testCase.fact {
				// Neither requirement may overwrite the other.
				count := 0
				for _, requirement := range developmentRequirements(t, result, developmentRequirementsFileName) {
					if requirement.Kind == DevelopmentRequirementEvidenceDomain || requirement.Kind == DevelopmentRequirementEvidenceFact {
						for _, subject := range requirement.Subjects {
							if subject.ID == "ue.registration-state" {
								count++
							}
						}
					}
				}
				if count != 2 {
					t.Fatalf("both evidence trust requirements must be retained, got %d", count)
				}
			}
			if err := VerifyDevelopment(loadDevelopmentFixtureWithDescriptor(t, testCase.source, testCase.fact), result); err != nil {
				t.Fatalf("VerifyDevelopment: %v", err)
			}
		})
	}
}

func loadDevelopmentFixtureWithDescriptor(t *testing.T, includeSource, includeFact bool) DevelopmentBuildRequest {
	t.Helper()
	return modifyDevelopmentPackageFile(t, loadDevelopmentFixture(t), "urn:example:ca:tools", "evidence-guarantees.yaml",
		developmentDescriptorWithIdentities(t, includeSource, includeFact))
}

// TestDevelopmentRequirementClassSeparation proves that the Host Admission and
// Runtime Execution documents never carry each other's kinds and that the
// runtime document covers every statically reachable Mutation/Reconcile point.
func TestDevelopmentRequirementClassSeparation(t *testing.T) {
	legacy := recoveryFixture(t, recoveryDescriptor(true))
	request := developmentRequestFromLegacy(t, legacy)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("Mutation Development build failed: %#v", result.Diagnostics)
	}
	host := developmentRequirements(t, result, developmentRequirementsFileName)
	runtime := developmentRequirements(t, result, developmentRuntimeRequirementsFileName)
	if len(runtime) == 0 {
		t.Fatal("runtime-execution requirements document is empty for a Mutation with Reconcile")
	}
	runtimeKinds := map[string]int{}
	for _, requirement := range runtime {
		runtimeKinds[requirement.Kind]++
		if requirement.Kind != DevelopmentRequirementCommitProof && requirement.Kind != DevelopmentRequirementReconcile {
			t.Fatalf("runtime document contains a non-runtime kind: %s", requirement.Kind)
		}
		if requirement.Status != DevelopmentRuntimeRequirementStatus {
			t.Fatalf("runtime requirement status = %q", requirement.Status)
		}
		if requirement.Location.Path == "" || requirement.Location.Node == "" || requirement.Location.Selector == "" {
			t.Fatalf("runtime requirement lacks an exact source location: %+v", requirement.Location)
		}
	}
	if runtimeKinds[DevelopmentRequirementCommitProof] != 1 || runtimeKinds[DevelopmentRequirementReconcile] != 1 {
		t.Fatalf("runtime coverage = %#v, want exactly one COMMIT_PROOF and one RECONCILE", runtimeKinds)
	}
	for _, requirement := range host {
		if requirement.Kind == DevelopmentRequirementCommitProof || requirement.Kind == DevelopmentRequirementReconcile {
			t.Fatalf("host document contains a runtime kind: %s", requirement.Kind)
		}
		if requirement.Status != DevelopmentHostRequirementStatus {
			t.Fatalf("host requirement status = %q", requirement.Status)
		}
	}
	// Reordering the request must not move a requirement between classes or
	// change the artifact.
	permuted := cloneDevelopmentRequest(t, request)
	developmentReverseFiles(permuted.Files)
	reverseSnapshots(permuted.Snapshots)
	rebuilt := BuildDevelopment(permuted)
	if !rebuilt.Success || rebuilt.ArtifactDigest != result.ArtifactDigest {
		t.Fatalf("reordered request changed the Development artifact")
	}
}

// independenceDevelopmentFixture builds a Development request with three
// Evidence Descriptors in distinct Source Domains and two Independence Groups.
func independenceDevelopmentFixture(t *testing.T) DevelopmentBuildRequest {
	t.Helper()
	fixture := wp3aBaseFixture(t)
	fixture.files[wp3aPredicatesPath] = []byte(`predicates:
  example.ue.registered:
    parameters:
      - registration: ca::urn:example:ca:registration:v1
      - context: ca-context::urn:example:ca:invocation-context:v1
    evidence:
      requires:
        registration: [ca::ue.registration-state, ca::ue.registration-freshness, ca::ue.registration-alternate]
        context: [ca-context::ue.context-state]
      independence:
        - members:
            - parameter: registration
              guarantee: ca::ue.registration-state
            - parameter: registration
              guarantee: ca::ue.registration-freshness
            - parameter: registration
              guarantee: ca::ue.registration-alternate
            - parameter: context
              guarantee: ca-context::ue.context-state
        - members:
            - parameter: registration
              guarantee: ca::ue.registration-state
            - parameter: context
              guarantee: ca-context::ue.context-state
    evaluator:
      type: cel
      expression: >-
        registration.state == "REGISTERED"
`)
	fixture.files[wp3aProcedurePath] = []byte(`start: inspect-context
nodes:
  inspect-context:
    observation:
      tool: ca-context::ue.context.observe
      arguments:
        ueRef: context.ueRef
      routes:
        SUCCEEDED: inspect-registration
        FAILED: FAILED
        UNKNOWN: UNRESOLVED
  inspect-registration:
    observation:
      tool: ca::ue.registration.current
      arguments:
        ueRef: context.ueRef
      routes:
        FAILED: FAILED
        UNKNOWN: UNRESOLVED
    postcondition:
      predicates:
        - predicate: example.ue.registered
          arguments:
            - inspect-registration.observation.output
            - inspect-context.observation.output
      routes:
        PASS: SUCCEEDED
        FAIL: FAILED
        UNKNOWN: UNRESOLVED
`)
	// The base descriptor gains domain one; two further descriptor documents
	// declare domains two and three under the same Observation producer.
	base := string(replaceDependencyFileContent(t, &fixture, "ca", "evidence-guarantees.yaml", nil))
	base += "  sourceIdentity: {domain: \"urn:example:record-domain:one\", path: record.sourceRecordId}\n"
	replaceDependencyFileContent(t, &fixture, "ca", "evidence-guarantees.yaml", []byte(base))
	replaceDependencyFileContent(t, &fixture, "ca", "freshness-guarantees.yaml", []byte(`ue.registration-freshness:
  authority: example-ue-owner
  fact: ue.registration-freshness
  producers:
    - owner: example-ue-owner
      kind: observation
      id: ue.registration.current
      outcome: SUCCEEDED
      path: output
  scope:
    subject: {record: scope.subject, equals: query.ueRef}
  correlation:
    invocation: {record: invocationId, equals: execution.invocationId}
  factValue: value
  selection: {mode: latest, orderBy: observedAt}
  freshness: {timestamp: observedAt, maxAgeMs: 10000}
  sourceIdentity: {domain: "urn:example:record-domain:two", path: record.sourceRecordId}
`))
	replaceDependencyFileContent(t, &fixture, "ca", "alternate-guarantees.yaml", []byte(`ue.registration-alternate:
  authority: example-ue-owner
  fact: ue.registration-alternate
  producers:
    - owner: example-ue-owner
      kind: observation
      id: ue.registration.current
      outcome: SUCCEEDED
      path: output
  scope:
    subject: {record: scope.subject, equals: query.ueRef}
  correlation:
    invocation: {record: invocationId, equals: execution.invocationId}
  factValue: value
  selection: {mode: latest, orderBy: observedAt}
  freshness: {timestamp: observedAt, maxAgeMs: 10000}
  sourceIdentity: {domain: "urn:example:record-domain:three", path: record.sourceRecordId}
`))
	catalog := string(replaceDependencyFileContent(t, &fixture, "ca", "tool-catalog.yaml", nil))
	catalog = strings.Replace(catalog,
		"            - guarantee: ue.registration-state\n              path: output\n",
		"            - guarantee: ue.registration-state\n              path: output\n"+
			"            - guarantee: ue.registration-freshness\n              path: output\n"+
			"            - guarantee: ue.registration-alternate\n              path: output\n", 1)
	replaceDependencyFileContent(t, &fixture, "ca", "tool-catalog.yaml", []byte(catalog))
	// A second root package publishes its own Evidence Descriptor and Observation
	// producer, so one Independence Group spans two exact Interface Package keys.
	replaceDependencyFileContent(t, &fixture, "ca-context", "tool-catalog.yaml", []byte(`apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  name: example-context-queries
  owner: example-host-owner
spec:
  contracts:
    - kind: observation
      id: ue.context.observe
      effect: READ
      ownerOutcomes: [SUCCEEDED, FAILED]
      adapter: example-context-query
      queryContextSchema:
        $ref: urn:example:ca:invocation-context:v1
      outputSchema:
        $ref: urn:example:ca:invocation-context:v1
      guarantees:
        evidence:
          provides:
            - guarantee: ue.context-state
              path: output
`))
	replaceDependencyFileContent(t, &fixture, "ca-context", "context-state.yaml", []byte(`ue.context-state:
  authority: example-host-owner
  fact: ue.context
  producers:
    - owner: example-host-owner
      kind: observation
      id: ue.context.observe
      outcome: SUCCEEDED
      path: output
  scope:
    subject: {record: scope.subject, equals: query.ueRef}
  correlation:
    invocation: {record: invocationId, equals: execution.invocationId}
  factValue: value
  selection: {mode: latest, orderBy: observedAt}
  freshness: {timestamp: observedAt, maxAgeMs: 10000}
  sourceIdentity: {domain: "urn:example:record-domain:four", path: record.sourceRecordId}
`))
	request := developmentRequestFromLegacy(t, fixture.request(t))
	// The Development scope declares a direct dependency from the tools package to
	// the host-context package so the cross-package Capability member and the
	// Procedure observation resolve through one exact lock edge.
	hostContextKey := developmentPackageKeyByID(t, request, wp3aHostContextPackageID)
	request = modifyDevelopmentManifest(t, request, wp3aToolsPackageID, func(manifest map[string]any) {
		dependencies, _ := manifest["dependencies"].([]any)
		manifest["dependencies"] = append(dependencies, map[string]any{
			"alias": "ca-context", "packageId": wp3aHostContextPackageID, "version": hostContextKey.Version,
		})
	})
	for index := range request.Lock.Packages {
		if request.Lock.Packages[index].Key.PackageID != wp3aToolsPackageID {
			continue
		}
		request.Lock.Packages[index].Dependencies = append(request.Lock.Packages[index].Dependencies,
			DevelopmentLockEdge{Alias: "ca-context", Key: hostContextKey})
	}
	// The base legacy fixture has no deployment CapabilityContract bundle, so the
	// Development scope is given the one CapabilityCatalog its Contract grants.
	return addDevelopmentExportAndFile(t, request, wp3aToolsPackageID, "capabilities.json",
		[]byte("{\n  \"capabilities\": [\n    {\n      \"id\": \"example.registration-read\",\n      \"members\": [\n        {\"callableId\": \"ue.registration.current\"},\n        {\"dependencyAlias\": \"ca-context\", \"callableId\": \"ue.context.observe\"}\n      ]\n    }\n  ]\n}\n"),
		map[string]any{"kind": "CapabilityCatalog", "id": "example.registration-read", "path": "capabilities.json"})
}

func developmentPackageKeyByID(t *testing.T, request DevelopmentBuildRequest, packageID string) DevelopmentPackageKey {
	t.Helper()
	for _, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID == packageID {
			return snapshot.Key
		}
	}
	t.Fatalf("package %s not found", packageID)
	return DevelopmentPackageKey{}
}

// replaceDependencyFileContent replaces or appends one file inside a resolved
// dependency snapshot and returns the (possibly unchanged) previous bytes.
func replaceDependencyFileContent(t *testing.T, fixture *wp3aFixture, dependencyID, path string, content []byte) []byte {
	t.Helper()
	for index := range fixture.dependencies {
		if fixture.dependencies[index].DependencyID != dependencyID {
			continue
		}
		files := fixture.dependencies[index].Files
		for fileIndex := range files {
			if files[fileIndex].Path != path {
				continue
			}
			previous := append([]byte(nil), files[fileIndex].Bytes...)
			if content != nil {
				files[fileIndex].Bytes = append([]byte(nil), content...)
			}
			return previous
		}
		if content != nil {
			files = append(files, File{Path: path, Bytes: append([]byte(nil), content...)})
			sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
			fixture.dependencies[index].Files = files
		}
		return nil
	}
	t.Fatalf("dependency %s not found", dependencyID)
	return nil
}

// TestDevelopmentIndependenceGroupsAreExact proves that each Independence Group
// is recorded separately with its own source position, members, Source Domains,
// Domain Pairs and exact Interface Package keys, and that reordering inputs is
// deterministic.
func TestDevelopmentIndependenceGroupsAreExact(t *testing.T) {
	request := independenceDevelopmentFixture(t)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("multi-domain independence Development build failed: %#v", result.Diagnostics)
	}
	var groups []DevelopmentHostRequirement
	for _, requirement := range developmentRequirements(t, result, developmentRequirementsFileName) {
		if requirement.Kind == DevelopmentRequirementIndependence {
			groups = append(groups, requirement)
		}
	}
	if len(groups) != 2 {
		t.Fatalf("independence requirements = %d, want 2", len(groups))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Independence.GroupIndex < groups[j].Independence.GroupIndex })
	first, second := groups[0].Independence, groups[1].Independence
	if first.GroupIndex != 0 || second.GroupIndex != 1 {
		t.Fatalf("group indexes = %d,%d", first.GroupIndex, second.GroupIndex)
	}
	if groups[0].Location.Node != "example.ue.registered" || groups[1].Location.Node != "example.ue.registered" {
		t.Fatalf("group predicate identity = %q,%q", groups[0].Location.Node, groups[1].Location.Node)
	}
	if len(first.Members) != 4 || len(first.DomainPairs) != 6 {
		t.Fatalf("group 0 members/pairs = %d/%d, want 4/6", len(first.Members), len(first.DomainPairs))
	}
	if len(second.Members) != 2 || len(second.DomainPairs) != 1 {
		t.Fatalf("group 1 members/pairs = %d/%d, want 2/1", len(second.Members), len(second.DomainPairs))
	}
	domains := map[string]bool{}
	rootKeys := map[string]bool{}
	for _, member := range first.Members {
		if member.Key.PackageID != wp3aToolsPackageID && member.Key.PackageID != wp3aHostContextPackageID {
			t.Fatalf("independence member package key = %q", member.Key.PackageID)
		}
		if member.DependencyAlias == "" || member.ExportID == "" || member.SourcePath == "" {
			t.Fatalf("independence member identity is incomplete: %+v", member)
		}
		domains[member.Domain] = true
		rootKeys[developmentKeyString(member.Key)] = true
	}
	if len(domains) != 4 {
		t.Fatalf("group 0 declared domains = %#v, want 4", domains)
	}
	// Group 1 spans two exact root packages, proving no env.roots[0]-style
	// stand-in is used.
	secondPackageKeys := map[string]bool{}
	for _, member := range second.Members {
		secondPackageKeys[member.Key.PackageID] = true
		domains[member.Domain] = true
		rootKeys[developmentKeyString(member.Key)] = true
	}
	if !secondPackageKeys[wp3aToolsPackageID] || !secondPackageKeys[wp3aHostContextPackageID] {
		t.Fatalf("group 1 member package keys = %#v, want the two distinct roots", secondPackageKeys)
	}
	if len(domains) != 4 || len(rootKeys) != 2 {
		t.Fatalf("independence domains/roots = %d/%d, want 4/2", len(domains), len(rootKeys))
	}
	for _, pair := range first.DomainPairs {
		if len(pair) != 2 || pair[0] >= pair[1] {
			t.Fatalf("domain pair is not canonically ordered: %#v", pair)
		}
	}
	if !strings.Contains(groups[0].Location.Selector, "/evidence/independence/0") ||
		!strings.Contains(groups[0].Location.Selector, "/predicates/example.ue.registered/") {
		t.Fatalf("group 0 selector = %q", groups[0].Location.Selector)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
	// The same group structure with reversed input order must be byte identical.
	permuted := cloneDevelopmentRequest(t, request)
	developmentReverseFiles(permuted.Files)
	reverseSnapshots(permuted.Snapshots)
	reverseRoots(permuted.Lock.Roots)
	reversePackages(permuted.Lock.Packages)
	for index := range permuted.Lock.Packages {
		reverseEdges(permuted.Lock.Packages[index].Dependencies)
	}
	rebuilt := BuildDevelopment(permuted)
	if !rebuilt.Success || rebuilt.ArtifactDigest != result.ArtifactDigest {
		t.Fatalf("reordered independence inputs changed the Development artifact")
	}
}

// TestDevelopmentIndependenceSameDomainFails proves that an Independence Group
// that cannot be independent from its own declarations fails the Build instead of
// producing an external requirement.
func TestDevelopmentIndependenceSameDomainFails(t *testing.T) {
	request := independenceDevelopmentFixture(t)
	// Rewrite all three descriptors to the same declared Source Domain.
	for _, path := range []string{"evidence-guarantees.yaml", "freshness-guarantees.yaml", "alternate-guarantees.yaml"} {
		request = modifyDevelopmentPackageFile(t, request, wp3aToolsPackageID, path,
			bytes.ReplaceAll(mustDevelopmentPackageFile(t, request, wp3aToolsPackageID, path),
				[]byte("urn:example:record-domain:two"), []byte("urn:example:record-domain:one")))
		request = modifyDevelopmentPackageFile(t, request, wp3aToolsPackageID, path,
			bytes.ReplaceAll(mustDevelopmentPackageFile(t, request, wp3aToolsPackageID, path),
				[]byte("urn:example:record-domain:three"), []byte("urn:example:record-domain:one")))
	}
	request = modifyDevelopmentPackageFile(t, request, wp3aHostContextPackageID, "context-state.yaml",
		bytes.ReplaceAll(mustDevelopmentPackageFile(t, request, wp3aHostContextPackageID, "context-state.yaml"),
			[]byte("urn:example:record-domain:four"), []byte("urn:example:record-domain:one")))
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("an Independence Group with one declared Source Domain must fail the Build")
	}
	if !hasDevelopmentCode(result.Diagnostics, "EVIDENCE_GUARANTEE_LINK_INVALID") {
		t.Fatalf("expected EVIDENCE_GUARANTEE_LINK_INVALID, got %#v", result.Diagnostics)
	}
}

// ingressDevelopmentRequest loads the released Wait/Ingress fixture as a
// Development request.
func ingressDevelopmentRequest(t *testing.T) DevelopmentBuildRequest {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "web", "parity-v1alpha5", "trusted-build-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := DecodeBuildRequestJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return developmentRequestFromLegacy(t, legacy)
}

func mutateIngressExport(t *testing.T, request DevelopmentBuildRequest, mutate func(export map[string]any)) DevelopmentBuildRequest {
	t.Helper()
	return modifyDevelopmentManifest(t, request, "urn:example:package:smf-integration", func(manifest map[string]any) {
		exports, _ := manifest["exports"].([]any)
		for _, entry := range exports {
			export, _ := entry.(map[string]any)
			if export["kind"] == "IngressCatalog" {
				mutate(export)
				return
			}
		}
		t.Fatal("fixture has no IngressCatalog export")
	})
}

// TestDevelopmentIngressExportIdentity proves that a package.json IngressCatalog
// export must bind exactly to the catalog payload: id, path, operation identity
// and Owner are all proven, and each negative case fails the Build.
func TestDevelopmentIngressExportIdentity(t *testing.T) {
	base := ingressDevelopmentRequest(t)
	if result := BuildDevelopment(base); !result.Success {
		t.Fatalf("positive Ingress fixture must build: %#v", result.Diagnostics)
	}

	cases := []struct {
		name   string
		mutate func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest
	}{
		{
			name: "wrong export id",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				return mutateIngressExport(t, request, func(export map[string]any) {
					export["id"] = "nsmf-pdusession.unknown"
				})
			},
		},
		{
			name: "wrong export path",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				return mutateIngressExport(t, request, func(export map[string]any) {
					export["path"] = "presentation/openapi/update-sm-context.json"
				})
			},
		},
		{
			name: "missing operation",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				return modifyDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml",
					bytes.Replace(mustDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml"),
						[]byte("id: nsmf-pdusession.update-sm-context"), []byte("id: nsmf-pdusession.other"), 1))
			},
		},
		{
			name: "duplicate operation",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				catalog := mustDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml")
				catalog = bytes.Replace(catalog, []byte("    - id: nsmf-pdusession.update-sm-context"),
					[]byte("    - id: nsmf-pdusession.update-sm-context\n    - id: nsmf-pdusession.update-sm-context"), 1)
				return modifyDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml", catalog)
			},
		},
		{
			name: "wrong owner",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				catalog := mustDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml")
				catalog = bytes.Replace(catalog, []byte("owner: example.smf-integration-owner"), []byte("owner: example.other-owner"), 1)
				return modifyDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml", catalog)
			},
		},
		{
			name: "unexported operation",
			mutate: func(t *testing.T, request DevelopmentBuildRequest) DevelopmentBuildRequest {
				catalog := mustDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml")
				catalog = append(catalog, []byte("    - id: nsmf-pdusession.undeclared\n")...)
				return modifyDevelopmentPackageFile(t, request, "urn:example:package:smf-integration", "ingress/catalog.yaml", catalog)
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := testCase.mutate(t, ingressDevelopmentRequest(t))
			result := BuildDevelopment(request)
			if result.Success {
				t.Fatalf("invalid IngressCatalog export identity %q must fail the Build", testCase.name)
			}
			if len(result.OutputFiles) != 0 {
				t.Fatal("failed Ingress Build produced an artifact")
			}
			if !hasDevelopmentCode(result.Diagnostics, developmentSnapshotInvalid) {
				t.Fatalf("expected %s for invalid Ingress identity, got %#v", developmentSnapshotInvalid, result.Diagnostics)
			}
		})
	}
}

// TestDevelopmentVerifyRejectsRequirementClassMigration proves that moving a
// host requirement into the runtime document, or deleting one from either
// document, fails Verify even after every outer digest is re-sealed.
func TestDevelopmentVerifyRejectsRequirementClassMigration(t *testing.T) {
	legacy := recoveryFixture(t, recoveryDescriptor(true))
	request := developmentRequestFromLegacy(t, legacy)
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("Mutation Development build failed: %#v", result.Diagnostics)
	}
	runtime := developmentRequirements(t, result, developmentRuntimeRequirementsFileName)
	host := developmentRequirements(t, result, developmentRequirementsFileName)

	deletedRuntime := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		files[developmentRuntimeRequirementsFileName] = canonicalTestJSON(t, runtime[1:])
	})
	if err := VerifyDevelopment(request, deletedRuntime); err == nil {
		t.Fatal("VerifyDevelopment accepted a deleted runtime requirement")
	}

	migrated := append(append([]DevelopmentHostRequirement{}, host...), runtime[0])
	sort.Slice(migrated, func(i, j int) bool { return migrated[i].RequirementID < migrated[j].RequirementID })
	tampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		files[developmentRequirementsFileName] = canonicalTestJSON(t, migrated)
		files[developmentRuntimeRequirementsFileName] = canonicalTestJSON(t, runtime[1:])
	})
	if err := VerifyDevelopment(request, tampered); err == nil {
		t.Fatal("VerifyDevelopment accepted a runtime requirement migrated into Host Admission")
	}

	// Editing the manifest's embedded lock and re-sealing every digest fails.
	lockTampered := tamperDevelopmentArtifact(t, result, func(files map[string][]byte) {
		var manifest map[string]any
		if err := json.Unmarshal(files[developmentManifestFileName], &manifest); err != nil {
			t.Fatal(err)
		}
		lock, _ := manifest["lock"].(map[string]any)
		roots, _ := lock["roots"].([]any)
		root, _ := roots[0].(map[string]any)
		root["alias"] = "tampered-root"
		files[developmentManifestFileName] = canonicalTestJSON(t, manifest)
	})
	if err := VerifyDevelopment(request, lockTampered); err == nil {
		t.Fatal("VerifyDevelopment accepted an edited embedded lock")
	}
}

// TestDevelopmentVerifyRejectsSourceChanges proves that changing the original
// Procedure, Descriptor or Requirement inputs makes Verify fail even though the
// presented artifact is internally self-consistent.
func TestDevelopmentVerifyRejectsSourceChanges(t *testing.T) {
	descriptorRequest := loadDevelopmentFixtureWithDescriptor(t, true, true)
	result := BuildDevelopment(descriptorRequest)
	if !result.Success {
		t.Fatalf("fixture build failed: %#v", result.Diagnostics)
	}
	changedDescriptor := cloneDevelopmentRequest(t, descriptorRequest)
	changedDescriptor = modifyDevelopmentPackageFile(t, changedDescriptor, "urn:example:ca:tools", "evidence-guarantees.yaml",
		bytes.ReplaceAll(developmentDescriptorWithIdentities(t, true, true),
			[]byte("urn:example:authority:one"), []byte("urn:example:authority:two")))
	if err := VerifyDevelopment(changedDescriptor, result); err == nil {
		t.Fatal("VerifyDevelopment accepted a changed Evidence Descriptor")
	}

	changedProcedure := cloneDevelopmentRequest(t, descriptorRequest)
	procedurePath := "skills/verify-ue-registration/assurance/procedure.yaml"
	var current []byte
	for _, file := range changedProcedure.Files {
		if file.Path == procedurePath {
			current = file.Bytes
		}
	}
	if current == nil {
		t.Fatal("fixture has no Procedure")
	}
	edited := bytes.ReplaceAll(current, []byte("inspect-registration"), []byte("inspect-registered"))
	for index := range changedProcedure.Files {
		if changedProcedure.Files[index].Path == procedurePath {
			changedProcedure.Files[index].Bytes = edited
		}
	}
	if err := VerifyDevelopment(changedProcedure, result); err == nil {
		t.Fatal("VerifyDevelopment accepted a changed Procedure")
	}
}
