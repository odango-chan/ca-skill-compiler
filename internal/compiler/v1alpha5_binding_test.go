package compiler

// WP3: v1alpha5 Ingress exact link and trigger/correlation binding tests.
//
// All fixtures are SYNTHETIC static-compiler fixtures. They are not real Owner
// artifacts, deployment profiles or 5GC/3GPP publication claims. The Ingress
// schemas/digests are derived from the exact fixture bytes by the same WP1 Schema
// Closure Digest v1 implementation the production path uses.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"github.com/odango-chan/ca-skill-compiler/internal/ingresscatalog"
)

const (
	wp3SnapshotBaseURI = "https://snapshots.invalid/example/build/"

	wp3ProcedurePath = "skills/example/assurance/procedure.yaml"
	wp3ContractPath  = "skills/example/assurance/contract.yaml"

	wp3IntegrationDependency = "integration"
	wp3ToolsDependency       = "tools"

	wp3IntegrationPackageID = "urn:example:package:smf-integration"
	wp3ToolsPackageID       = "urn:example:package:amf-client-tools"

	wp3IntegrationOwner = "example.smf-integration-owner"
	wp3ToolsOwner       = "example.amf-tool-owner"

	wp3IngressExportID = "nsmf-pdusession.update-sm-context"
	wp3TriggerExportID = "amf.communication.n1n2_message_transfer"

	wp3CatalogPath = "ingress/catalog.yaml"

	wp3OpenAPIArtifactRef    = "urn:example:openapi:update-sm-context"
	wp3RequestArtifactRef    = "urn:example:schema:update-request"
	wp3NormalizedArtifactRef = "urn:example:schema:update-normalized"
	wp3ArmInputArtifactRef   = "urn:example:schema:update-arm-input"
	wp3TransferInputRef      = "urn:example:schema:amf-n1n2-transfer-input"

	wp3OpenAPIPath       = "presentation/openapi/update-sm-context.json"
	wp3RequestSchemaPath = "presentation/schemas/update-request.json"
	wp3NormalizedPath    = "presentation/schemas/update-normalized.json"
	wp3ArmInputPath      = "presentation/schemas/update-arm-input.json"
	wp3TransferInputPath = "presentation/schemas/transfer-input.json"

	wp3ToolsCatalogPath = "catalogs/toolcatalog.yaml"

	// The v1alpha5 fixture is a complete Skill package: the inherited static
	// semantics require SKILL.md, a Contract with a Goal and declared
	// dependencies/capabilities, a PredicateSet, package Schemas and exact
	// trusted ToolCatalog/Capability artifacts. The Wait-specific assertions are
	// unchanged; this only makes the base fixture a full-language package.
	wp3SkillMarkdownPath = "skills/example/SKILL.md"
	wp3PredicatesPath    = "skills/example/assurance/predicates.yaml"

	wp3SkillMarkdown = "# SYNTHETIC v1alpha5 full-language fixture Skill\n"

	wp3Predicates = `predicates:
  session.goal_satisfied:
    parameters: []
    evaluator:
      type: cel
      expression: "true"
  session.trigger_required:
    parameters: []
    evaluator:
      type: cel
      expression: "true"
  session.wait_output_ready:
    parameters:
      - request: integration::urn:example:schema:update-normalized
    evaluator:
      type: cel
      expression: "true"
`

	wp3TransferOutputRef  = "urn:example:schema:amf-n1n2-transfer-output"
	wp3TransferOutputPath = "presentation/schemas/transfer-output.json"
	wp3SkillContextRef    = "urn:example:schema:skill-context"
	wp3SkillContextPath   = "presentation/schemas/skill-context.json"
	wp3UpdateInputRef     = "urn:example:schema:smf-update-input"
	wp3UpdateInputPath    = "presentation/schemas/smf-update-input.json"
	wp3UpdateOutputRef    = "urn:example:schema:smf-update-output"
	wp3UpdateOutputPath   = "presentation/schemas/smf-update-output.json"
	wp3TimeoutInputRef    = "urn:example:schema:smf-timeout-input"
	wp3TimeoutInputPath   = "presentation/schemas/smf-timeout-input.json"
	wp3TimeoutOutputRef   = "urn:example:schema:smf-timeout-output"
	wp3TimeoutOutputPath  = "presentation/schemas/smf-timeout-output.json"

	wp3CapabilityContractsPath = "capability-contracts.yaml"

	wp3DeploymentID     = "urn:example:deployment:skill-tools"
	wp3PublisherID      = "urn:example:publisher:skill-tools"
	wp3AuthorizationRef = "urn:example:authorization:skill-tools"

	wp3TransferOutputSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:amf-n1n2-transfer-output","type":"object","properties":{"smContextRef":{"type":"string"}}}`
	wp3SkillContextSchema   = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:skill-context","type":"object","additionalProperties":false,"properties":{"smContextRef":{"type":"string"},"ueRef":{"type":"string"}},"required":["smContextRef"]}`
	wp3UpdateInputSchema    = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:smf-update-input","type":"object","properties":{"request":{"type":"object"},"smContextRef":{"type":"string"}}}`
	wp3UpdateOutputSchema   = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:smf-update-output","type":"object","properties":{"smContextRef":{"type":"string"}}}`
	wp3TimeoutInputSchema   = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:smf-timeout-input","type":"object","properties":{"request":{"type":"object"}}}`
	wp3TimeoutOutputSchema  = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:smf-timeout-output","type":"object","properties":{}}`

	wp3CapabilityContracts = `# SYNTHETIC TEST FIXTURE ONLY. Not a real deployment profile.
apiVersion: connectionagent.io/capability-contracts/v1alpha4
kind: CapabilityContracts
metadata:
  name: example-deployment
spec:
  capabilities:
    - id: example.skill-tools
      owner: deployment
      callables:
        - owner: example.amf-tool-owner
          id: amf.communication.n1n2_message_transfer
          kind: operation
          effect: READ
          scopePolicyRef: deployment.skill-tools
        - owner: example.amf-tool-owner
          id: smf.session.update_from_amf
          kind: operation
          effect: READ
          scopePolicyRef: deployment.skill-tools
        - owner: example.amf-tool-owner
          id: smf.update_timeout.handle
          kind: operation
          effect: READ
          scopePolicyRef: deployment.skill-tools
`

	wp3OpenAPIDocument = `{"openapi":"3.0.0","info":{"title":"Synthetic TS 29.502 UpdateSMContext","version":"19.5.0"},"paths":{"/sm-contexts/{smContextRef}/modify":{"post":{"operationId":"UpdateSmContext","parameters":[{"name":"smContextRef","in":"path","required":true,"schema":{"type":"string"}}],"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Request"}}}},"responses":{"204":{"description":"no content"}}}}},"components":{"schemas":{"Request":{"type":"object","properties":{"updateData":{"type":"object"}}}}}}`

	wp3RequestSchema       = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/example/update-request","type":"object","properties":{"updateData":{"type":"object"}}}`
	wp3NormalizedSchema    = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:update-normalized","type":"object","properties":{"smContextRef":{"type":"string"},"updateData":{"type":"object"}},"required":["smContextRef"]}`
	wp3ArmInputSchema      = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/example/update-arm-input","type":"object","properties":{"smContextRef":{"type":"string"}},"required":["smContextRef"]}`
	wp3TransferInputSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.invalid/example/transfer-input","type":"object","properties":{"smContextRef":{"type":"string"}}}`

	wp3ToolCatalog = `apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example.amf-tool-owner
spec:
  contracts:
    - kind: operation
      id: amf.communication.n1n2_message_transfer
      effect: READ
      inputSchema:
        $ref: urn:example:schema:amf-n1n2-transfer-input
      outputSchema:
        $ref: urn:example:schema:amf-n1n2-transfer-output
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
    - kind: operation
      id: smf.session.update_from_amf
      effect: READ
      inputSchema:
        $ref: urn:example:schema:smf-update-input
      outputSchema:
        $ref: urn:example:schema:smf-update-output
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
    - kind: operation
      id: smf.update_timeout.handle
      effect: READ
      inputSchema:
        $ref: urn:example:schema:smf-timeout-input
      outputSchema:
        $ref: urn:example:schema:smf-timeout-output
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
`

	wp3PositiveProcedure = `start: trigger-transfer
nodes:
  trigger-transfer:
    operation:
      tool: tools::amf.communication.n1n2_message_transfer
      arguments:
        smContextRef: "sm-context-placeholder"
      routes:
        SUCCEEDED: wait-sm-context
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  wait-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: tools::smf.session.update_from_amf
      arguments:
        request: wait-sm-context.wait.output
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  handle-update-timeout:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: UNRESOLVED
        UNKNOWN: SAFE_STOPPED
`

	// wp3WaitFreeProcedure is a complete Wait-free v1alpha5 Procedure: it proves
	// the inherited Procedure/assurance path compiles without any Appendix L Wait.
	wp3WaitFreeProcedure = `start: complete-without-wait
nodes:
  complete-without-wait:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`

	// wp3WaitFreeContract declares only the dependencies a Wait-free Skill uses.
	wp3WaitFreeContract = `goal: session.goal_satisfied
dependencies:
  tools: urn:example:package:amf-client-tools
  integration: urn:example:package:smf-integration
capabilities:
  - example.skill-tools
`

	// wp3StartWaitProcedure is the start-Node Wait variant: the Wait-bearing Node
	// is the Procedure start, so the export must declare start arming.
	wp3StartWaitProcedure = `start: wait-sm-context
nodes:
  wait-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: tools::smf.session.update_from_amf
      arguments:
        request: wait-sm-context.wait.output
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  handle-update-timeout:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: UNRESOLVED
        UNKNOWN: SAFE_STOPPED
`

	// wp3BypassTriggerProcedure has two incoming paths to the Wait-bearing Node;
	// one path bypasses the triggering Operation, so no single trigger dominates
	// every reachable entry path.
	wp3BypassTriggerProcedure = `start: choose
nodes:
  choose:
    precondition:
      predicates:
        - predicate: session.trigger_required
          arguments: []
      routes:
        PASS: trigger-transfer
        FAIL: wait-sm-context
        UNKNOWN: UNRESOLVED
  trigger-transfer:
    operation:
      tool: tools::amf.communication.n1n2_message_transfer
      arguments:
        smContextRef: "sm-context-placeholder"
      routes:
        SUCCEEDED: wait-sm-context
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  wait-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: tools::smf.session.update_from_amf
      arguments:
        request: wait-sm-context.wait.output
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  handle-update-timeout:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: UNRESOLVED
        UNKNOWN: SAFE_STOPPED
`

	wp3Contract = `goal: session.goal_satisfied
dependencies:
  tools: urn:example:package:amf-client-tools
  integration: urn:example:package:smf-integration
capabilities:
  - example.skill-tools
`
)

// wp3FixtureOptions lets each negative test change exactly one declaration.
type wp3FixtureOptions struct {
	procedure       string
	contract        string
	catalogText     func(openAPIDigest, requestDigest, normalizedDigest string) string
	extraFiles      []File // extra files in the integration dependency snapshot
	extraOwners     []ExportOwner
	extraOwnersFor  string
	toolsCatalog    string
	extraToolsFiles []File
	mutateContext   func(*TrustedBuildContext)
}

func wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest string) string {
	return fmt.Sprintf(`# SYNTHETIC TEST FIXTURE ONLY. Not a real Owner artifact or deployment profile.
apiVersion: connectionagent.io/ingresscatalog/v1alpha1
kind: IngressCatalog
metadata:
  owner: example.smf-integration-owner
spec:
  operations:
    - id: nsmf-pdusession.update-sm-context
      adapter: integration-adapter::nsmf-pdusession
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      protocol:
        specification: 3GPP TS 29.502
        release: "19.5.0"
        apiUriVersion: v1
        openapi:
          artifactRef: urn:example:openapi:update-sm-context
          artifactDigest: %s
          operationId: UpdateSmContext
          method: POST
          path: /sm-contexts/{smContextRef}/modify
      security:
        authenticatedPeerType: AMF
        authenticationProfileRef: deployment-auth::synthetic-amf-peer
        authorizationPolicyRef: deployment-auth::synthetic-update-sm-context
      request:
        acceptedMediaTypes:
          - application/json
        sourceSchemas:
          - mediaType: application/json
            schemaRef: urn:example:schema:update-request
            schemaDigest: %s
        normalizedSchemaRef: urn:example:schema:update-normalized
        normalizedSchemaDigest: %s
        normalization:
          representations:
            - mediaType: application/json
              mappings:
                - source:
                    kind: json-body
                    pointer: ""
                  target: /updateData
                  transform: identity
                - source:
                    kind: path
                    name: smContextRef
                  target: /smContextRef
                  transform: identity
        maxBytes: 4194304
      correlation:
        armInputSchemaRef: urn:example:schema:update-arm-input
        keyFields:
          - requestPath: /smContextRef
            dimension: session
            comparison: exact
        duplicate:
          identitySource: canonical-request-digest
          scope: expected-ingress
        staleDelivery:
          strategy: single-flight-tombstone
          resourceKey: /smContextRef
          maximumLateArrival: PT30S
          clockSkewAllowance: PT5S
          tombstoneRetention: PT60S
          basisRef: urn:example:synthetic-guarantee
      arming:
        start:
          allowed: false
        triggers:
          - operation:
              packageId: urn:example:package:amf-client-tools
              owner: example.amf-tool-owner
              kind: operation
              id: tools::amf.communication.n1n2_message_transfer
            bindings:
              - requestKey: /smContextRef
                bindingPath: /smContextRef
                source:
                  kind: operation-argument
                  path: /smContextRef
      response:
        mode: none
`, openAPIDigest, requestDigest, normalizedDigest)
}

func wp3Digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// wp3Artifacts returns the exact trusted artifacts shared by the fixture and the
// recomputed Schema Closure digests used inside the synthetic IngressCatalog.
func wp3Artifacts(t *testing.T) ([]ingresscatalog.SnapshotArtifact, map[string]string) {
	t.Helper()
	artifacts := []ingresscatalog.SnapshotArtifact{
		ingresscatalog.NewSnapshotArtifactFromRaw(wp3OpenAPIArtifactRef, wp3OpenAPIPath, ingresscatalog.DialectOpenAPI30, []byte(wp3OpenAPIDocument)),
		ingresscatalog.NewSnapshotArtifactFromRaw(wp3RequestArtifactRef, wp3RequestSchemaPath, ingresscatalog.DialectJSONSchema202012, []byte(wp3RequestSchema)),
		ingresscatalog.NewSnapshotArtifactFromRaw(wp3NormalizedArtifactRef, wp3NormalizedPath, ingresscatalog.DialectJSONSchema202012, []byte(wp3NormalizedSchema)),
		ingresscatalog.NewSnapshotArtifactFromRaw(wp3ArmInputArtifactRef, wp3ArmInputPath, ingresscatalog.DialectJSONSchema202012, []byte(wp3ArmInputSchema)),
		ingresscatalog.NewSnapshotArtifactFromRaw(wp3TransferInputRef, wp3TransferInputPath, ingresscatalog.DialectJSONSchema202012, []byte(wp3TransferInputSchema)),
	}
	resolution, err := ingresscatalog.NewResolutionContext(wp3SnapshotBaseURI, artifacts)
	if err != nil {
		t.Fatalf("build synthetic resolution context: %v", err)
	}
	digests := map[string]string{}
	for _, root := range []struct{ ref, artifact string }{
		{wp3RequestArtifactRef, wp3RequestArtifactRef},
		{wp3NormalizedArtifactRef, wp3NormalizedArtifactRef},
	} {
		closure, err := resolution.ComputeSchemaClosure(ingresscatalog.RootRef{ArtifactRef: root.artifact, Pointer: "#"})
		if err != nil {
			t.Fatalf("compute synthetic schema closure %s: %v", root.ref, err)
		}
		digests[root.ref] = closure.Digest
	}
	return artifacts, digests
}

// wp3BuildRequest builds the complete synthetic v1alpha5 BuildRequest. Exactly
// one declaration can be varied per test through wp3FixtureOptions.
// wp3CapabilityContractsFor derives the synthetic trusted CapabilityContracts
// bundle from the exact ToolCatalog fixture under test, so every callable in the
// fixture is covered by the single granted capability. This is still a
// SYNTHETIC fixture, not a real deployment profile.
func wp3CapabilityContractsFor(t *testing.T, catalogText string) []byte {
	t.Helper()
	root, err := yaml.Parse([]byte(catalogText))
	if err != nil {
		t.Fatalf("parse synthetic ToolCatalog fixture: %v", err)
	}
	owner := scalar(child(child(root, "metadata"), "owner"))
	contracts := sequence(child(child(root, "spec"), "contracts"))
	var bundle strings.Builder
	bundle.WriteString(`# SYNTHETIC TEST FIXTURE ONLY. Not a real deployment profile.
apiVersion: connectionagent.io/capability-contracts/v1alpha4
kind: CapabilityContracts
metadata:
  name: example-deployment
spec:
  capabilities:
    - id: example.skill-tools
      owner: deployment
      callables:
`)
	for _, contract := range contracts {
		kind := scalar(child(contract, "kind"))
		id := scalar(child(contract, "id"))
		effect := scalar(child(contract, "effect"))
		if kind == "" || id == "" || effect == "" {
			continue
		}
		fmt.Fprintf(&bundle,
			"        - owner: %s\n          id: %s\n          kind: %s\n          effect: %s\n          scopePolicyRef: deployment.skill-tools\n",
			owner, id, kind, effect)
	}
	return []byte(bundle.String())
}

func wp3BuildRequest(t *testing.T, options wp3FixtureOptions) BuildRequest {
	t.Helper()
	if options.procedure == "" {
		options.procedure = wp3PositiveProcedure
	}
	if options.contract == "" {
		options.contract = wp3Contract
	}
	if options.catalogText == nil {
		options.catalogText = wp3DefaultCatalogText
	}
	if options.toolsCatalog == "" {
		options.toolsCatalog = wp3ToolCatalog
	}
	_, digests := wp3Artifacts(t)
	catalogBytes := []byte(options.catalogText(
		wp3Digest([]byte(wp3OpenAPIDocument)), digests[wp3RequestArtifactRef], digests[wp3NormalizedArtifactRef]))

	integrationFiles := []File{
		{Path: wp3CatalogPath, Bytes: catalogBytes},
		{Path: wp3OpenAPIPath, Bytes: []byte(wp3OpenAPIDocument)},
		{Path: wp3RequestSchemaPath, Bytes: []byte(wp3RequestSchema)},
		{Path: wp3NormalizedPath, Bytes: []byte(wp3NormalizedSchema)},
		{Path: wp3ArmInputPath, Bytes: []byte(wp3ArmInputSchema)},
	}
	integrationFiles = append(integrationFiles, options.extraFiles...)
	toolsFiles := []File{
		{Path: wp3ToolsCatalogPath, Bytes: []byte(options.toolsCatalog)},
		{Path: wp3TransferInputPath, Bytes: []byte(wp3TransferInputSchema)},
		{Path: wp3TransferOutputPath, Bytes: []byte(wp3TransferOutputSchema)},
		{Path: wp3UpdateInputPath, Bytes: []byte(wp3UpdateInputSchema)},
		{Path: wp3UpdateOutputPath, Bytes: []byte(wp3UpdateOutputSchema)},
		{Path: wp3TimeoutInputPath, Bytes: []byte(wp3TimeoutInputSchema)},
		{Path: wp3TimeoutOutputPath, Bytes: []byte(wp3TimeoutOutputSchema)},
	}
	toolsFiles = append(toolsFiles, options.extraToolsFiles...)

	capabilityFiles := []File{{Path: wp3CapabilityContractsPath, Bytes: wp3CapabilityContractsFor(t, options.toolsCatalog)}}
	capabilityDigest, err := ComputeDependencySnapshotDigest(capabilityFiles)
	if err != nil {
		t.Fatalf("capability contracts digest: %v", err)
	}

	integrationDigest, err := ComputeDependencySnapshotDigest(integrationFiles)
	if err != nil {
		t.Fatalf("integration snapshot digest: %v", err)
	}
	toolsDigest, err := ComputeDependencySnapshotDigest(toolsFiles)
	if err != nil {
		t.Fatalf("tools snapshot digest: %v", err)
	}

	context := &TrustedBuildContext{
		DependencyBindings: []TrustedDependencyBinding{			{
				DependencyID:     wp3ToolsDependency,
				Digest:           toolsDigest,
				StablePackageID:  wp3ToolsPackageID,
				AuthorizedOwners: []string{wp3ToolsOwner},
			},
			{
				DependencyID:     wp3IntegrationDependency,
				Digest:           integrationDigest,
				StablePackageID:  wp3IntegrationPackageID,
				AuthorizedOwners: []string{wp3IntegrationOwner},
				ExportOwners: []ExportOwner{{
					ExportKind: v1alpha5ExportKindIngressOperation,
					ExportID:   wp3IngressExportID,
					Owner:      wp3IntegrationOwner,
				}},
			},
		},
		SnapshotBaseURI: wp3SnapshotBaseURI,
		CapabilityContracts: &CapabilityContractsBundle{
			DeploymentID: wp3DeploymentID,
			PublisherID:  wp3PublisherID,
			Digest:       capabilityDigest,
			Files:        capabilityFiles,
		},
		DeploymentAuthorizations: &[]DeploymentAuthorization{{
			DeploymentID:     wp3DeploymentID,
			PublisherID:      wp3PublisherID,
			AuthorizationRef: wp3AuthorizationRef,
		}},
		ArtifactBindings: []TrustedArtifactBinding{
			{ArtifactRef: wp3OpenAPIArtifactRef, DependencyID: wp3IntegrationDependency, SnapshotPath: wp3OpenAPIPath, Dialect: "openapi-3.0"},
			{ArtifactRef: wp3RequestArtifactRef, DependencyID: wp3IntegrationDependency, SnapshotPath: wp3RequestSchemaPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3NormalizedArtifactRef, DependencyID: wp3IntegrationDependency, SnapshotPath: wp3NormalizedPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3ArmInputArtifactRef, DependencyID: wp3IntegrationDependency, SnapshotPath: wp3ArmInputPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3TransferInputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp3TransferInputPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3TransferOutputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp3TransferOutputPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3UpdateInputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp3UpdateInputPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3UpdateOutputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp3UpdateOutputPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3TimeoutInputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp3TimeoutInputPath, Dialect: "json-schema-2020-12"},
			{ArtifactRef: wp3TimeoutOutputRef, DependencyID: wp3ToolsDependency, SnapshotPath: wp3TimeoutOutputPath, Dialect: "json-schema-2020-12"},
		},
		SchemaBindings: []TrustedSchemaBinding{
			{ExportID: wp3RequestArtifactRef, ArtifactRef: wp3RequestArtifactRef, Pointer: "#"},
			{ExportID: wp3NormalizedArtifactRef, ArtifactRef: wp3NormalizedArtifactRef, Pointer: "#"},
			{ExportID: wp3ArmInputArtifactRef, ArtifactRef: wp3ArmInputArtifactRef, Pointer: "#"},
			{ExportID: wp3TransferInputRef, ArtifactRef: wp3TransferInputRef, Pointer: "#"},
			{ExportID: wp3TransferOutputRef, ArtifactRef: wp3TransferOutputRef, Pointer: "#"},
			{ExportID: wp3UpdateInputRef, ArtifactRef: wp3UpdateInputRef, Pointer: "#"},
			{ExportID: wp3UpdateOutputRef, ArtifactRef: wp3UpdateOutputRef, Pointer: "#"},
			{ExportID: wp3TimeoutInputRef, ArtifactRef: wp3TimeoutInputRef, Pointer: "#"},
			{ExportID: wp3TimeoutOutputRef, ArtifactRef: wp3TimeoutOutputRef, Pointer: "#"},
		},
	}
	if len(options.extraOwners) > 0 {
		target := wp3IntegrationDependency
		if options.extraOwnersFor != "" {
			target = options.extraOwnersFor
		}
		for index := range context.DependencyBindings {
			if context.DependencyBindings[index].DependencyID == target {
				context.DependencyBindings[index].ExportOwners = append(context.DependencyBindings[index].ExportOwners, options.extraOwners...)
			}
		}
	}
	if options.mutateContext != nil {
		options.mutateContext(context)
	}

	return BuildRequest{
		APIVersion:     BuildRequestAPIVersionV1Alpha5,
		LanguageTarget: V1Alpha5Target,
		Files: []File{
			{Path: wp3ProcedurePath, Bytes: []byte(options.procedure)},
			{Path: wp3ContractPath, Bytes: []byte(options.contract)},
			{Path: wp3SkillMarkdownPath, Bytes: []byte(wp3SkillMarkdown)},
			{Path: wp3PredicatesPath, Bytes: []byte(wp3Predicates)},
		},
		ResolvedDependencies: []ResolvedDependency{
			{DependencyID: wp3ToolsDependency, Digest: toolsDigest, Files: toolsFiles},
			{DependencyID: wp3IntegrationDependency, Digest: integrationDigest, Files: integrationFiles},
		},
		TrustedBuildContext: context,
	}
}

func wp3Check(t *testing.T, options wp3FixtureOptions) []Diagnostic {
	t.Helper()
	return Check(wp3BuildRequest(t, options))
}

func assertWP3Diagnostics(t *testing.T, diagnostics []Diagnostic, wantCodes ...string) {
	t.Helper()
	got := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		got = append(got, diagnostic.Code)
		if diagnostic.Severity != "error" || diagnostic.Phase != "C_STATIC" {
			t.Fatalf("%s identity = %s/%s, want error/C_STATIC", diagnostic.Code, diagnostic.Severity, diagnostic.Phase)
		}
	}
	if len(got) != len(wantCodes) {
		t.Fatalf("diagnostic codes = %v, want exactly %v (%#v)", got, wantCodes, diagnostics)
	}
	for index := range wantCodes {
		if got[index] != wantCodes[index] {
			t.Fatalf("diagnostic codes = %v, want %v (%#v)", got, wantCodes, diagnostics)
		}
	}
}

// TestV1Alpha5WP3PositiveTrustedIngressLink is the full positive path: Wait +
// IngressCatalog + ToolCatalog + OpenAPI/Schema artifacts + exact trigger
// dominance => zero diagnostics through the public Check entry.
func TestV1Alpha5WP3PositiveTrustedIngressLink(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{})
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero WP3 diagnostics, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP3SelectorNotFound covers L.3.1 zero-match resolution.
func TestV1Alpha5WP3SelectorNotFound(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "operation: UpdateSMContext", "operation: UpdateSmContext", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "WAIT_INGRESS_EXPORT_NOT_FOUND")
	if diagnostics[0].Path != wp3ProcedurePath {
		t.Fatalf("selector diagnostic path = %q, want %q", diagnostics[0].Path, wp3ProcedurePath)
	}
	if !strings.Contains(diagnostics[0].Message, "AMF/Nsmf_PDUSession/UpdateSMContext") {
		t.Fatalf("selector diagnostic must render the exact selector: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP3SelectorAmbiguous covers L.3.1 multi-match resolution across two
// trusted dependency snapshots.
func TestV1Alpha5WP3SelectorAmbiguous(t *testing.T) {
	_, digests := wp3Artifacts(t)
	secondCatalog := wp3DefaultCatalogText(
		wp3Digest([]byte(wp3OpenAPIDocument)), digests[wp3RequestArtifactRef], digests[wp3NormalizedArtifactRef])
	// The second trusted Catalog declares the same exact event selector but a
	// distinct trusted Ingress Operation export identity.
	secondCatalog = strings.Replace(secondCatalog, "id: nsmf-pdusession.update-sm-context", "id: nsmf-pdusession.update-sm-context.second", 1)
	request := wp3BuildRequest(t, wp3FixtureOptions{
		extraFiles:     []File{{Path: "ingress/catalog-second.yaml", Bytes: []byte(secondCatalog)}},
		extraOwners:    []ExportOwner{{ExportKind: v1alpha5ExportKindIngressOperation, ExportID: "nsmf-pdusession.update-sm-context.second", Owner: wp3IntegrationOwner}},
		extraOwnersFor: wp3IntegrationDependency,
	})
	diagnostics := Check(request)
	assertWP3Diagnostics(t, diagnostics, "WAIT_INGRESS_EXPORT_AMBIGUOUS")
}

// TestV1Alpha5WP3UntrustedCatalogSuppressed proves that an untrusted Catalog
// produces no trusted export and no Ingress diagnostics: the Wait selector is
// simply not found among trusted exports.
func TestV1Alpha5WP3UntrustedCatalogSuppressed(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{
		mutateContext: func(context *TrustedBuildContext) {
			for index := range context.DependencyBindings {
				if context.DependencyBindings[index].DependencyID == wp3IntegrationDependency {
					// Owner is authorized, but the exact ingressOperation export-owner
					// record is missing: the Catalog is untrusted.
					context.DependencyBindings[index].ExportOwners = nil
				}
			}
		},
	})
	diagnostics := Check(request)
	assertWP3Diagnostics(t, diagnostics, "WAIT_INGRESS_EXPORT_NOT_FOUND")
}

// TestV1Alpha5WP3UntrustedOwnerSuppressed proves that a Catalog whose Owner is
// not authorized cannot produce trusted exports.
func TestV1Alpha5WP3UntrustedOwnerSuppressed(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{
		mutateContext: func(context *TrustedBuildContext) {
			for index := range context.DependencyBindings {
				if context.DependencyBindings[index].DependencyID == wp3IntegrationDependency {
					context.DependencyBindings[index].AuthorizedOwners = []string{"example.other-owner"}
				}
			}
		},
	})
	diagnostics := Check(request)
	assertWP3Diagnostics(t, diagnostics, "DEPENDENCY_OWNER_MISMATCH")
}

// TestV1Alpha5WP3SchemaClosureDigestMismatch covers a declared schemaDigest that
// does not match the Schema Closure Digest recomputed from raw bytes. The
// prerequisite failure suppresses dependent selector diagnostics.
func TestV1Alpha5WP3SchemaClosureDigestMismatch(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, normalizedDigest,
				"sha256:0000000000000000000000000000000000000000000000000000000000000000", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "SCHEMA_INVALID")
}

// TestV1Alpha5WP3TriggerOperationMissing covers the exact Package ID + Owner +
// kind=operation + id trigger link with a missing export.
func TestV1Alpha5WP3TriggerOperationMissing(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "tools::amf.communication.n1n2_message_transfer",
				"tools::amf.communication.not_published", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "EXPORT_ID_MISSING")
}

// TestV1Alpha5WP3TriggerOperationAmbiguous covers a duplicate ToolCatalog
// Operation export identity in the trusted snapshot.
func TestV1Alpha5WP3TriggerOperationAmbiguous(t *testing.T) {
	ambiguousCatalog := wp3ToolCatalog + `    - kind: operation
      id: amf.communication.n1n2_message_transfer
      effect: READ
      inputSchema:
        $ref: urn:example:schema:amf-n1n2-transfer-input
      outputSchema:
        $ref: urn:example:schema:amf-n1n2-transfer-output
      ownerOutcomes:
        - SUCCEEDED
`
	diagnostics := wp3Check(t, wp3FixtureOptions{toolsCatalog: ambiguousCatalog})
	assertWP3Diagnostics(t, diagnostics, "CAPABILITY_MEMBERSHIP_DUPLICATE", "EXPORT_ID_AMBIGUOUS")
}

// TestV1Alpha5WP3TriggerOwnerMismatch proves the trusted Owner is part of the
// exact trigger identity.
func TestV1Alpha5WP3TriggerOwnerMismatch(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "owner: example.amf-tool-owner", "owner: example.untrusted-tool-owner", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "EXPORT_ID_MISSING")
}

// TestV1Alpha5WP3MissingArmInputSchema covers a declared armInputSchemaRef that
// the trusted v1alpha5 Schema bindings do not provide.
func TestV1Alpha5WP3MissingArmInputSchema(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "armInputSchemaRef: urn:example:schema:update-arm-input",
				"armInputSchemaRef: urn:example:schema:not-bound", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "SCHEMA_MISSING")
}

// TestV1Alpha5WP3MissingKeyFieldPath covers keyFields[].requestPath that does not
// exist in the normalized request Schema.
func TestV1Alpha5WP3MissingKeyFieldPath(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "requestPath: /smContextRef\n            dimension: session",
				"requestPath: /smContextRefMissing\n            dimension: session", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_CORRELATION_INVALID")
}

// TestV1Alpha5WP3NonScalarKeyField covers a correlation key field that resolves
// to a non-scalar Schema.
func TestV1Alpha5WP3NonScalarKeyField(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "requestPath: /smContextRef\n            dimension: session",
				"requestPath: /updateData\n            dimension: session", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_CORRELATION_INVALID")
}

// TestV1Alpha5WP3BindingPathMissing covers an arming bindingPath that does not
// exist in the typed armInputSchema.
func TestV1Alpha5WP3BindingPathMissing(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "bindingPath: /smContextRef", "bindingPath: /notInArmInputSchema", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_CORRELATION_INVALID")
}

// TestV1Alpha5WP3TriggerNotDominatingOnOnePath covers the per-incoming-path
// dominance rule from Appendix L.4.2/#62 §6.3.
func TestV1Alpha5WP3TriggerNotDominatingOnOnePath(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{procedure: wp3BypassTriggerProcedure})
	assertWP3Diagnostics(t, diagnostics, "WAIT_CORRELATION_BINDING_INVALID")
	if diagnostics[0].Path != wp3ProcedurePath {
		t.Fatalf("correlation diagnostic path = %q, want %q", diagnostics[0].Path, wp3ProcedurePath)
	}
	if !strings.Contains(diagnostics[0].Message, "Node 'wait-sm-context' cannot derive one exact correlation binding") {
		t.Fatalf("correlation diagnostic must use the L.11.1 template: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP3StartWaitWithoutArmingStart covers the start-Node arming rule.
func TestV1Alpha5WP3StartWaitWithoutArmingStart(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{procedure: wp3StartWaitProcedure})
	assertWP3Diagnostics(t, diagnostics, "WAIT_CORRELATION_BINDING_INVALID")
}

// TestV1Alpha5WP3StartWaitWithCompleteStartArming covers the positive start-Node
// Wait path: start arming allowed with a complete Invocation Context binding.
func TestV1Alpha5WP3StartWaitWithCompleteStartArming(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{
		procedure: wp3StartWaitProcedure,
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			text = strings.Replace(text, `        start:
          allowed: false
        triggers:`, `        start:
          allowed: true
          bindings:
            - requestKey: /smContextRef
              bindingPath: /smContextRef
              source:
                kind: invocation-context
                path: /invocation/context/smContextRef
        triggers:`, 1)
			return text
		},
	})
	diagnostics := Check(request)
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero diagnostics for complete start arming, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP3IncompleteStartArming covers an allowed start arming whose
// declared bindings do not cover every correlation key field.
func TestV1Alpha5WP3IncompleteStartArming(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		procedure: wp3StartWaitProcedure,
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, `        start:
          allowed: false
        triggers:`, `        start:
          allowed: true
          bindings:
            - requestKey: /notTheCorrelationKey
              bindingPath: /smContextRef
              source:
                kind: invocation-context
                path: /invocation/context/other
        triggers:`, 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_CORRELATION_INVALID")
}

// TestV1Alpha5WP3RetentionWindowInvalid covers the strict §7.3 retention
// ordering: tombstoneRetention must strictly exceed the sum.
func TestV1Alpha5WP3RetentionWindowInvalid(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "tombstoneRetention: PT60S", "tombstoneRetention: PT35S", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_CORRELATION_INVALID")
}

// TestV1Alpha5WP3TrustedContextBindingValidation covers malformed v1alpha5-only
// trusted-context bindings at the strict envelope (no linking cascade).
func TestV1Alpha5WP3TrustedContextBindingValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*TrustedBuildContext)
	}{
		{"missing-snapshot-base-uri", func(context *TrustedBuildContext) { context.SnapshotBaseURI = "" }},
		{"unknown-artifact-dependency", func(context *TrustedBuildContext) {
			context.ArtifactBindings[0].DependencyID = "missing-dependency"
		}},
		{"missing-artifact-snapshot-path", func(context *TrustedBuildContext) {
			context.ArtifactBindings[0].SnapshotPath = "presentation/openapi/missing.json"
		}},
		{"invalid-artifact-dialect", func(context *TrustedBuildContext) {
			context.ArtifactBindings[0].Dialect = "openapi-2.0"
		}},
		{"unknown-schema-artifact-ref", func(context *TrustedBuildContext) {
			context.SchemaBindings[0].ArtifactRef = "urn:example:artifact:not-bound"
		}},
		{"non-canonical-schema-pointer", func(context *TrustedBuildContext) {
			context.SchemaBindings[0].Pointer = "/schema"
		}},
		{"duplicate-artifact-ref", func(context *TrustedBuildContext) {
			context.ArtifactBindings[0].ArtifactRef = context.ArtifactBindings[1].ArtifactRef
		}},
		{"duplicate-schema-export", func(context *TrustedBuildContext) {
			context.SchemaBindings[1].ExportID = context.SchemaBindings[0].ExportID
		}},
	}
	base := wp3BuildRequest(t, wp3FixtureOptions{})
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := base
			context := *base.TrustedBuildContext
			context.ArtifactBindings = append([]TrustedArtifactBinding(nil), context.ArtifactBindings...)
			context.SchemaBindings = append([]TrustedSchemaBinding(nil), context.SchemaBindings...)
			testCase.mutate(&context)
			request.TrustedBuildContext = &context
			diagnostics := Check(request)
			assertWP3Diagnostics(t, diagnostics, "REQUEST_INVALID")
		})
	}
}

// TestV1Alpha5WP3TriggerArgumentsMissing covers the pre-dispatch
// materialization rule: the declared correlation argument must exist in the
// triggering Node's materialized operation arguments.
func TestV1Alpha5WP3TriggerArgumentsMissing(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		`      arguments:
        smContextRef: "sm-context-placeholder"
      routes:
        SUCCEEDED: wait-sm-context`,
		`      arguments: {}
      routes:
        SUCCEEDED: wait-sm-context`, 1)
	diagnostics := wp3Check(t, wp3FixtureOptions{procedure: procedure})
	assertWP3Diagnostics(t, diagnostics, "WAIT_CORRELATION_BINDING_INVALID")
}

// TestV1Alpha5WP3IngressExportOwnerMismatch covers IN-004: the trusted
// ingressOperation export-owner record Owner differs from the Catalog Owner.
func TestV1Alpha5WP3IngressExportOwnerMismatch(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{mutateContext: func(context *TrustedBuildContext) {
		for index := range context.DependencyBindings {
			if context.DependencyBindings[index].DependencyID != wp3IntegrationDependency {
				continue
			}
			for ownerIndex := range context.DependencyBindings[index].ExportOwners {
				context.DependencyBindings[index].ExportOwners[ownerIndex].Owner = "example.other-owner"
			}
		}
	}})
	assertWP3Diagnostics(t, Check(request), "DEPENDENCY_OWNER_MISMATCH")
}

// TestV1Alpha5WP3IngressExportOwnerRecordMissing covers IN-004: the trusted
// binding declares ingressOperation exports but the Catalog Operation has no
// exact export-owner record.
func TestV1Alpha5WP3IngressExportOwnerRecordMissing(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{mutateContext: func(context *TrustedBuildContext) {
		for index := range context.DependencyBindings {
			if context.DependencyBindings[index].DependencyID != wp3IntegrationDependency {
				continue
			}
			context.DependencyBindings[index].ExportOwners = []ExportOwner{{
				ExportKind: v1alpha5ExportKindIngressOperation,
				ExportID:   "nsmf-pdusession.other-export",
				Owner:      wp3IntegrationOwner,
			}}
		}
	}})
	assertWP3Diagnostics(t, Check(request), "DEPENDENCY_EXPORT_OWNER_UNVERIFIED")
}

// TestV1Alpha5EW031EntryPreconditionBypass covers EW-031: a global Precondition
// gate that a reachable success path bypasses is rejected with the inherited
// ENTRY_PRECONDITION_BYPASSED identity. Wait entry control is not a fifth
// business stage and never establishes the gate.
func TestV1Alpha5EW031EntryPreconditionBypass(t *testing.T) {
	contract := `goal: session.goal_satisfied
precondition: gate.precondition
dependencies:
  tools: urn:example:package:amf-client-tools
  integration: urn:example:package:smf-integration
capabilities:
  - example.skill-tools
`
	procedure := `start: entry
nodes:
  entry:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: choose
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  choose:
    precondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: gate
        FAIL: shortcut
        UNKNOWN: UNRESOLVED
  gate:
    precondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: final
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
  shortcut:
    operation:
      tool: tools::smf.update_timeout.handle
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        UNKNOWN: SAFE_STOPPED
  final:
    postcondition:
      predicates:
        - predicate: session.goal_satisfied
          arguments: []
      routes:
        PASS: SUCCEEDED
        FAIL: SAFE_STOPPED
        UNKNOWN: UNRESOLVED
`
	assertWP3Diagnostics(t, wp3Check(t, wp3FixtureOptions{procedure: procedure, contract: contract}), "ENTRY_PRECONDITION_BYPASSED")
}

// TestV1Alpha5WP3TriggerSelfOutputUnavailable covers IN-019: a trigger argument
// that depends on the triggering Operation's own not-yet-produced output emits
// the inherited SOURCE_NOT_AVAILABLE identity; the dependent correlation binding
// is also rejected.
func TestV1Alpha5WP3TriggerSelfOutputUnavailable(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure,
		`        smContextRef: "sm-context-placeholder"`,
		`        smContextRef: trigger-transfer.operation.output.smContextRef`, 1)
	diagnostics := wp3Check(t, wp3FixtureOptions{procedure: procedure})
	assertWP3Diagnostics(t, diagnostics, "SOURCE_NOT_AVAILABLE", "WAIT_CORRELATION_BINDING_INVALID")
}

// TestV1Alpha5WP3WireDiscriminatorDuplicateIdentityInvalid covers IN-046/IN-060:
// an authenticated-header-parameter wire discriminator may only double as the
// stale-delivery discriminator when the duplicate identity source is the same
// 3gpp-Sbi-Request-Info idempotency-key.
func TestV1Alpha5WP3WireDiscriminatorDuplicateIdentityInvalid(t *testing.T) {
	staleBlock := `        staleDelivery:
          strategy: single-flight-tombstone
          resourceKey: /smContextRef
          maximumLateArrival: PT30S
          clockSkewAllowance: PT5S
          tombstoneRetention: PT60S
          basisRef: urn:example:synthetic-guarantee`
	wireBlock := `        staleDelivery:
          strategy: wire-discriminator
          discriminator:
            kind: authenticated-header-parameter
            header: 3gpp-Sbi-Request-Info
            parameter: idempotency-key
          maximumLateArrival: PT30S
          clockSkewAllowance: PT5S
          discriminatorRetention: PT60S
          basisRef: urn:example:synthetic-guarantee`
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, staleBlock, wireBlock, 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_CORRELATION_INVALID")
}

// TestV1Alpha5WP3DependencyIdentityFailureSuppressesLinking covers the
// prerequisite suppression rule: an unverifiable dependency snapshot produces its
// existing DEPENDENCY_* diagnostic and no dependent Ingress/Wait diagnostics.
func TestV1Alpha5WP3DependencyIdentityFailureSuppressesLinking(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{})
	request.TrustedBuildContext.DependencyBindings[1].Digest =
		"sha256:1111111111111111111111111111111111111111111111111111111111111111"
	diagnostics := Check(request)
	assertWP3Diagnostics(t, diagnostics, "DEPENDENCY_IDENTITY_UNVERIFIED")
}

// TestV1Alpha5WP3V1Alpha4RejectsV1Alpha5OnlyTrustedFields keeps the frozen
// v1alpha4 API closed: the shared TrustedBuildContext may carry the new fields,
// but the v1alpha4 envelope validation explicitly rejects them.
func TestV1Alpha5WP3V1Alpha4RejectsV1Alpha5OnlyTrustedFields(t *testing.T) {
	request := BuildRequest{
		APIVersion:           BuildRequestAPIVersion,
		LanguageTarget:       V1Alpha4Target,
		ResolvedDependencies: []ResolvedDependency{},
		Files: []File{
			{Path: "skills/example/SKILL.md", Bytes: []byte("# example\n")},
		},
		TrustedBuildContext: &TrustedBuildContext{
			SnapshotBaseURI: "https://snapshots.invalid/example/build/",
			SchemaBindings:  []TrustedSchemaBinding{{ExportID: "urn:example:schema:x", ArtifactRef: "urn:example:artifact:x", Pointer: "#"}},
		},
	}
	diagnostics := Check(request)
	if !hasV1Alpha5Diagnostic(diagnostics, "REQUEST_INVALID") {
		t.Fatalf("v1alpha4 must reject v1alpha5-only trusted fields, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP3IngressNormalizationCoverage drives the Ingress-owned request
// normalization static semantics through the public v1alpha5 Check path: a
// required normalized property that no mapping provides must fail closed with
// INGRESS_NORMALIZATION_INVALID.
func TestV1Alpha5WP3IngressNormalizationCoverage(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text,
				`                - source:
                    kind: path
                    name: smContextRef
                  target: /smContextRef
                  transform: identity
`, "", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_NORMALIZATION_INVALID")
	if !strings.Contains(diagnostics[0].Message, "request normalization mapping") {
		t.Fatalf("normalization diagnostic must use the registry template: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP3IngressPeerProfileReferenceInvalid drives the Ingress-owned
// peer profile reference format rule through the public v1alpha5 Check path.
func TestV1Alpha5WP3IngressPeerProfileReferenceInvalid(t *testing.T) {
	diagnostics := wp3Check(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text,
				"authenticationProfileRef: deployment-auth::synthetic-amf-peer",
				"authenticationProfileRef: malformed-reference", 1)
		},
	})
	assertWP3Diagnostics(t, diagnostics, "INGRESS_PEER_PROFILE_REFERENCE_INVALID")
}

// TestV1Alpha5WaitWithoutIngressExportIsNotFound proves the L.3.1 rule for a
// fully admitted v1alpha5 Skill whose trusted Ingress scope contains no matching
// Operation export: the Wait selector has zero trusted Ingress Operation exports
// and Check must emit the exact not-found diagnostic instead of silently
// accepting the Wait. It asserts the exact code, phase, severity, path, line and
// column.
func TestV1Alpha5WaitWithoutIngressExportIsNotFound(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{
		catalogText: func(openAPIDigest, requestDigest, normalizedDigest string) string {
			text := wp3DefaultCatalogText(openAPIDigest, requestDigest, normalizedDigest)
			return strings.Replace(text, "operation: UpdateSMContext", "operation: UpdateSmContext", 1)
		},
	})
	diagnostics := Check(request)
	assertWP3Diagnostics(t, diagnostics, "WAIT_INGRESS_EXPORT_NOT_FOUND")
	diagnostic := diagnostics[0]
	if diagnostic.Severity != "error" || diagnostic.Phase != "C_STATIC" {
		t.Fatalf("diagnostic identity = %s/%s, want error/C_STATIC", diagnostic.Severity, diagnostic.Phase)
	}
	if diagnostic.Path != wp3ProcedurePath {
		t.Fatalf("diagnostic path = %q, want %q", diagnostic.Path, wp3ProcedurePath)
	}
	if diagnostic.Line <= 0 || diagnostic.Column <= 0 {
		t.Fatalf("diagnostic must be anchored at the wait.event selector: %#v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, "Node 'wait-sm-context' wait selector 'AMF/Nsmf_PDUSession/UpdateSMContext' does not resolve to a trusted Ingress Operation export.") {
		t.Fatalf("diagnostic message must use the L.11.1 template: %q", diagnostic.Message)
	}
}
