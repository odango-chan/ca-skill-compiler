package compiler

// WP6: CA-SL v1alpha5 response-boundary and typed response-mapping tests.
//
// All fixtures are SYNTHETIC static-compiler fixtures. They are not real Owner
// artifacts, deployment profiles or 5GC/3GPP publication claims. The Ingress
// response Schemas, responseResultSchema and Producer Output Schema are synthetic
// JSON Schema documents whose Schema Closure Digests are derived from the exact
// fixture bytes by the same WP1 Schema Closure Digest v1 implementation the
// production path uses.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/odango-chan/ca-skill-compiler/internal/ingresscatalog"
)

const (
	wp6ResponseBodyRef       = "urn:example:schema:wp6-response-body"
	wp6MultipartRootRef      = "urn:example:schema:wp6-multipart-root"
	wp6BinaryRefRef          = "urn:example:schema:wp6-binary-ref"
	wp6ResponseResultRef     = "urn:example:schema:wp6-response-result"
	wp6SelectorResultRef     = "urn:example:schema:wp6-selector-result"
	wp6ProducerOutputRef     = "urn:example:schema:wp6-producer-output"
	wp6ProducerOutputBadRef  = "urn:example:schema:wp6-producer-output-bad"
	wp6ProducerOutputSelRef  = "urn:example:schema:wp6-producer-output-selector"
	wp6ProducerInputRef      = "urn:example:schema:wp6-producer-input"
	wp6ProducerOperationID   = "smf.session.response-producer"
	wp6ResponseProducerTool  = "tools::" + wp6ProducerOperationID
	wp6ResponseBodyPath      = "presentation/schemas/wp6-response-body.json"
	wp6MultipartRootPath     = "presentation/schemas/wp6-multipart-root.json"
	wp6BinaryRefPath         = "presentation/schemas/wp6-binary-ref.json"
	wp6ResponseResultPath    = "presentation/schemas/wp6-response-result.json"
	wp6SelectorResultPath    = "presentation/schemas/wp6-selector-result.json"
	wp6ProducerOutputPath    = "presentation/schemas/wp6-producer-output.json"
	wp6ProducerOutputBadPath = "presentation/schemas/wp6-producer-output-bad.json"
	wp6ProducerOutputSelPath = "presentation/schemas/wp6-producer-output-selector.json"
	wp6ProducerInputPath     = "presentation/schemas/wp6-producer-input.json"
	wp6ResponseStartArming   = `          allowed: true
          bindings:
            - requestKey: /smContextRef
              bindingPath: /smContextRef
              source:
                kind: invocation-context
                path: /invocation/context/smContextRef`
	wp6ProcedurePathForAssert = "skills/example/assurance/procedure.yaml"
)

// wp6SchemaTexts are the synthetic trusted schemas shared by the WP6 fixtures.
var wp6SchemaTexts = map[string]string{
	wp6ResponseBodyRef:      `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-response-body","type":"object","required":["n4Action","smContextRef"],"properties":{"n4Action":{"type":"string"},"smContextRef":{"type":"string"}}}`,
	wp6MultipartRootRef:     `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-multipart-root","type":"object","required":["n4Action","n2SmInfoContentId"],"properties":{"n4Action":{"type":"string"},"n2SmInfoContentId":{"type":"string"}}}`,
	wp6BinaryRefRef:         `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-binary-ref","type":"object","required":["contentId","mediaType","length","sha256"],"properties":{"contentId":{"type":"string"},"mediaType":{"type":"string"},"length":{"type":"integer"},"sha256":{"type":"string"}}}`,
	wp6ResponseResultRef:    `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-response-result","type":"object","required":["n4Action"],"properties":{"n4Action":{"type":"string"},"smContextRef":{"type":"string"},"n2SmInfo":{"type":"object"}}}`,
	wp6SelectorResultRef:    `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-selector-result","type":"object","required":["responseRepresentation"],"properties":{"responseRepresentation":{"type":"string","enum":["updated","no-content"]}}}`,
	wp6ProducerOutputRef:    `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-producer-output","type":"object","required":["n4Action","smContextRef"],"properties":{"n4Action":{"type":"string"},"smContextRef":{"type":"string"},"n2SmInfo":{"type":"object"}}}`,
	wp6ProducerOutputBadRef: `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-producer-output-bad","type":"object","required":["n4Action","smContextRef"],"properties":{"n4Action":{"type":"integer"},"smContextRef":{"type":"string"},"n2SmInfo":{"type":"object"}}}`,
	wp6ProducerOutputSelRef: `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-producer-output-selector","type":"object","required":["responseRepresentation"],"properties":{"responseRepresentation":{"type":"string","enum":["updated","no-content"]}}}`,
	wp6ProducerInputRef:     `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"urn:example:schema:wp6-producer-input","type":"object","properties":{"request":{"type":"object"}}}`,
}

// wp6ProducerOutputSelRef is the typed Producer output used by the
// multi-bodyless disposition selector fixture.

type wp6Digests struct {
	OpenAPI       string
	Request       string
	Normalized    string
	Body          string
	MultipartRoot string
	BinaryRef     string
	Result        string
	Selector      string
	Producer      string
	ProducerBad   string
	ProducerSel   string
}

func wp6SchemaDigest(t *testing.T, schemaRef string, raw []byte) string {
	t.Helper()
	resolution, err := ingresscatalog.NewResolutionContext(wp3SnapshotBaseURI, []ingresscatalog.SnapshotArtifact{
		ingresscatalog.NewSnapshotArtifactFromRaw(schemaRef, "schemas/synthetic.json", ingresscatalog.DialectJSONSchema202012, raw),
	})
	if err != nil {
		t.Fatalf("build wp6 digest resolution context: %v", err)
	}
	closure, err := resolution.ComputeSchemaClosure(ingresscatalog.RootRef{ArtifactRef: schemaRef, Pointer: "#"})
	if err != nil {
		t.Fatalf("compute wp6 schema closure %s: %v", schemaRef, err)
	}
	return closure.Digest
}

func wp6DigestsFor(t *testing.T) wp6Digests {
	t.Helper()
	_, digests := wp3Artifacts(t)
	selectorProducer := wp6SchemaTexts[wp6ProducerOutputSelRef]
	return wp6Digests{
		OpenAPI:       wp3Digest([]byte(wp3OpenAPIDocument)),
		Request:       digests[wp3RequestArtifactRef],
		Normalized:    digests[wp3NormalizedArtifactRef],
		Body:          wp6SchemaDigest(t, wp6ResponseBodyRef, []byte(wp6SchemaTexts[wp6ResponseBodyRef])),
		MultipartRoot: wp6SchemaDigest(t, wp6MultipartRootRef, []byte(wp6SchemaTexts[wp6MultipartRootRef])),
		BinaryRef:     wp6SchemaDigest(t, wp6BinaryRefRef, []byte(wp6SchemaTexts[wp6BinaryRefRef])),
		Result:        wp6SchemaDigest(t, wp6ResponseResultRef, []byte(wp6SchemaTexts[wp6ResponseResultRef])),
		Selector:      wp6SchemaDigest(t, wp6SelectorResultRef, []byte(wp6SchemaTexts[wp6SelectorResultRef])),
		Producer:      wp6SchemaDigest(t, wp6ProducerOutputRef, []byte(wp6SchemaTexts[wp6ProducerOutputRef])),
		ProducerBad:   wp6SchemaDigest(t, wp6ProducerOutputBadRef, []byte(wp6SchemaTexts[wp6ProducerOutputBadRef])),
		ProducerSel:   wp6SchemaDigest(t, wp6ProducerOutputSelRef, []byte(selectorProducer)),
	}
}

// wp6CatalogText renders the synthetic trusted IngressCatalog with the exact
// synthetic digests. responseBlock and startBlock are fully indented YAML
// fragments supplied by the individual fixture.
func wp6CatalogText(d wp6Digests, responseBlock, startBlock string) string {
	return `# SYNTHETIC TEST FIXTURE ONLY. Not a real Owner artifact or deployment profile.
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
          artifactDigest: ` + d.OpenAPI + `
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
            schemaDigest: ` + d.Request + `
        normalizedSchemaRef: urn:example:schema:update-normalized
        normalizedSchemaDigest: ` + d.Normalized + `
        normalization:
          representations:
            - mediaType: application/json
              mappings:
                - source:
                    kind: json-body
                    pointer: ""
                  target: /updateData
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
` + startBlock + `
        triggers: []
      response:
` + responseBlock + `
`
}

func wp6ToolsCatalog(outputSchemaRef string) string {
	return `# SYNTHETIC TEST FIXTURE ONLY. Not a real Owner artifact.
apiVersion: connectionagent.io/toolcatalog/v1alpha4
kind: ToolCatalog
metadata:
  owner: example.amf-tool-owner
spec:
  contracts:
    - kind: operation
      id: ` + wp6ProducerOperationID + `
      effect: READ
      inputSchema:
        $ref: ` + wp6ProducerInputRef + `
      outputSchema:
        $ref: ` + outputSchemaRef + `
      ownerOutcomes:
        - SUCCEEDED
        - FAILED
`
}

func wp6ResponseDeliveryRecoveryBlock() string {
	return `        deliveryRecovery:
          mode: protocol-replay
          requestIdentity: wp6-request-identity
          lostChannelDisposition: explicit-failure-response`
}

// wp6JSONBodyResponseBlock is the complete single body-bearing JSON
// representation with a typed Producer and full protocol-field mapping.
func wp6JSONBodyResponseBlock(d wp6Digests) string {
	return `        mode: paired
        deadlinePolicyRef: deployment-policy::wp6-response-deadline
` + wp6ResponseDeliveryRecoveryBlock() + `
        dispositions:
          - id: updated
            status: 200
            body: required
            representations:
              - id: json
                mediaType: application/json
                responseSchemaRef: ` + wp6ResponseBodyRef + `
                responseSchemaDigest: ` + d.Body + `
                maxBytes: 65536
        result:
          producer: operation
          responseResultSchemaRef: ` + wp6ResponseResultRef + `
          mappings:
            - dispositionId: updated
              representationId: json
              from: /n4Action
              target:
                kind: json-body
                pointer: /n4Action
              transform: identity
            - dispositionId: updated
              representationId: json
              from: /smContextRef
              target:
                kind: json-body
                pointer: /smContextRef
              transform: identity
` + wp6EarlyFailureBlock("SAFE_STOPPED", "UNRESOLVED")
}

// wp6JSONBodyResponseBlockWithoutEarlyFailures is the same contract without the
// required early-failure response policy.
func wp6JSONBodyResponseBlockWithoutEarlyFailures(d wp6Digests) string {
	block := wp6JSONBodyResponseBlock(d)
	return strings.Replace(block, wp6EarlyFailureBlock("SAFE_STOPPED", "UNRESOLVED"),
		"        # earlyFailureResponses deliberately omitted by the negative fixture\n", 1)
}

func wp6EarlyFailureBlock(outcomes ...string) string {
	block := "        earlyFailureResponses:\n"
	for _, outcome := range outcomes {
		block += "          " + outcome + ":\n            responseRef: urn:example:response-ref:wp6-" + strings.ToLower(outcome) + "\n"
	}
	return block
}

// wp6JSONBodyResponseBlockMissingMapping omits one required protocol field
// mapping, so the declared coverage is incomplete.
func wp6JSONBodyResponseBlockMissingMapping(d wp6Digests) string {
	block := wp6JSONBodyResponseBlock(d)
	return strings.Replace(block, `            - dispositionId: updated
              representationId: json
              from: /smContextRef
              target:
                kind: json-body
                pointer: /smContextRef
              transform: identity
`, "", 1)
}

// wp6MultipartResponseBlockMissingBinaryMapping omits the binary part mapping.
func wp6MultipartResponseBlockMissingBinaryMapping(d wp6Digests) string {
	block := wp6MultipartResponseBlock(d)
	return strings.Replace(block, `            - dispositionId: multipart-updated
              representationId: multipart
              from: /n2SmInfo
              target:
                kind: multipart-binary
                part: n2SmInfo
                contentIdField:
                  part: jsonData
                  pointer: /n2SmInfoContentId
              transform: binary-content-ref
`, "", 1)
}

// wp6SingleBodylessResponseBlock is the single fixed bodyless disposition case
// (L.8.2 exemption: no typed response-result Producer is required).
func wp6SingleBodylessResponseBlock(d wp6Digests) string {
	return `        mode: paired
        deadlinePolicyRef: deployment-policy::wp6-response-deadline
` + wp6ResponseDeliveryRecoveryBlock() + `
        dispositions:
          - id: no-content
            status: 204
            body: none
` + wp6EarlyFailureBlock("SAFE_STOPPED", "UNRESOLVED")
}

// wp6MultipartResponseBlock is the complete multipart/related body-bearing
// representation: JSON root part plus one binary part, typed result mappings,
// Content-ID field and immutable binary-content lifecycle.
func wp6MultipartResponseBlock(d wp6Digests) string {
	return `        mode: paired
        deadlinePolicyRef: deployment-policy::wp6-response-deadline
` + wp6ResponseDeliveryRecoveryBlock() + `
        dispositions:
          - id: multipart-updated
            status: 200
            body: required
            representations:
              - id: multipart
                mediaType: multipart/related
                responseSchemaRef: ` + wp6MultipartRootRef + `
                responseSchemaDigest: ` + d.MultipartRoot + `
                maxBytes: 65536
                rootPart: jsonData
                parts:
                  - name: jsonData
                    mediaType: application/json
                    kind: json
                    required: true
                    schemaRef: ` + wp6MultipartRootRef + `
                    schemaDigest: ` + d.MultipartRoot + `
                    maxBytes: 32768
                  - name: n2SmInfo
                    mediaType: application/octet-stream
                    kind: binary
                    required: false
                    maxBytes: 4096
                minBinaryParts: 1
                maxBinaryParts: 1
        result:
          producer: operation
          responseResultSchemaRef: ` + wp6ResponseResultRef + `
          mappings:
            - dispositionId: multipart-updated
              representationId: multipart
              from: /n4Action
              target:
                kind: multipart-json
                part: jsonData
                pointer: /n4Action
              transform: identity
            - dispositionId: multipart-updated
              representationId: multipart
              from: /n2SmInfo
              target:
                kind: multipart-binary
                part: n2SmInfo
                contentIdField:
                  part: jsonData
                  pointer: /n2SmInfoContentId
              transform: binary-content-ref
        binaryContent:
          referenceSchemaRef: ` + wp6BinaryRefRef + `
          digestAlgorithm: sha-256
          immutable: true
          access: integration-response-only
          retention: PT1H
          maxBytes: 8192
` + wp6EarlyFailureBlock("SAFE_STOPPED", "UNRESOLVED")
}

// wp6MultiBodylessSelectorResponseBlock declares two bodyless dispositions and a
// dispositionSelector. The valid variant covers both combinations exactly.
func wp6MultiBodylessSelectorResponseBlock(d wp6Digests) string {
	return `        mode: paired
        deadlinePolicyRef: deployment-policy::wp6-response-deadline
` + wp6ResponseDeliveryRecoveryBlock() + `
        dispositions:
          - id: updated
            status: 200
            body: none
          - id: no-content
            status: 204
            body: none
        result:
          producer: operation
          responseResultSchemaRef: ` + wp6SelectorResultRef + `
          dispositionSelector:
            sourcePath: /responseRepresentation
            cases:
              - value: updated
                dispositionId: updated
              - value: no-content
                dispositionId: no-content
` + wp6EarlyFailureBlock("SAFE_STOPPED", "UNRESOLVED")
}

// wp6BadSelectorResponseBlock has a structurally valid but semantically
// incomplete dispositionSelector: one case is not an enum value and the
// no-content combination is uncovered.
func wp6BadSelectorResponseBlock(d wp6Digests) string {
	block := wp6MultiBodylessSelectorResponseBlock(d)
	return strings.Replace(block,
		"              - value: no-content\n                dispositionId: no-content",
		"              - value: other\n                dispositionId: no-content", 1)
}

// wp6StartWaitProcedure is the base start-Node Wait procedure: the Wait-bearing
// Node is the Procedure start and its operation is the response Producer.
const wp6StartWaitProcedure = `start: update-sm-context
nodes:
  update-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
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
        OUTCOME_UNKNOWN: SAFE_STOPPED
`

// wp6NamedBoundaryProcedure declares an explicit response boundary later than
// the Wait-bearing Node. Every normal path completes the boundary Node.
const wp6NamedBoundaryProcedure = `start: update-sm-context
nodes:
  update-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: prepare-response
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: prepare-response
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  prepare-response:
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
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
        OUTCOME_UNKNOWN: SAFE_STOPPED
`

// wp6SuccessBeforeBoundaryProcedure has one accepted path that reaches SUCCEEDED
// through the respected fail route chain without completing the selected
// boundary Node.
const wp6SuccessBeforeBoundaryProcedure = `start: update-sm-context
nodes:
  update-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: prepare-response
      routes:
        TIMEOUT: handle-update-timeout
    precondition:
      predicates:
        - predicate: session.response_required
          arguments: []
      routes:
        FAIL: final
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: prepare-response
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  prepare-response:
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
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
        OUTCOME_UNKNOWN: SAFE_STOPPED
`

// wp6UnreachableBoundaryProcedure names a declared but unreachable boundary Node.
const wp6UnreachableBoundaryProcedure = `start: update-sm-context
nodes:
  update-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: orphan-response
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: final
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  orphan-response:
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: SAFE_STOPPED
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
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
        OUTCOME_UNKNOWN: SAFE_STOPPED
`

// wp6BoundaryWithoutProducerProcedure names a boundary Node that does not declare
// the export's Producer stage.
const wp6BoundaryWithoutProducerProcedure = `start: update-sm-context
nodes:
  update-sm-context:
    wait:
      event:
        source: AMF
        service: Nsmf_PDUSession
        operation: UpdateSMContext
      timeout: PT30S
      response:
        after: prepare-response
      routes:
        TIMEOUT: handle-update-timeout
    operation:
      tool: ` + wp6ResponseProducerTool + `
      arguments: {}
      routes:
        SUCCEEDED: prepare-response
        FAILED: SAFE_STOPPED
        OUTCOME_UNKNOWN: SAFE_STOPPED
  prepare-response:
    postcondition:
      predicates:
        - predicate: session.response_ready
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
        OUTCOME_UNKNOWN: SAFE_STOPPED
`

type wp6FixtureOptions struct {
	procedure   string
	response    func(wp6Digests) string
	toolsOutput string
}

func wp6BuildRequest(t *testing.T, options wp6FixtureOptions) BuildRequest {
	t.Helper()
	if options.procedure == "" {
		options.procedure = wp6StartWaitProcedure
	}
	if options.response == nil {
		options.response = wp6JSONBodyResponseBlock
	}
	toolsOutputRef := wp6ProducerOutputRef
	if options.toolsOutput != "" {
		toolsOutputRef = options.toolsOutput
	}
	digests := wp6DigestsFor(t)
	request := wp3BuildRequest(t, wp3FixtureOptions{
		procedure: options.procedure,
		catalogText: func(_, _, _ string) string {
			return wp6CatalogText(digests, options.response(digests), wp6ResponseStartArming)
		},
		toolsCatalog: wp6ToolsCatalog(toolsOutputRef),
		extraFiles: []File{
			{Path: wp6ResponseBodyPath, Bytes: []byte(wp6SchemaTexts[wp6ResponseBodyRef])},
			{Path: wp6MultipartRootPath, Bytes: []byte(wp6SchemaTexts[wp6MultipartRootRef])},
			{Path: wp6BinaryRefPath, Bytes: []byte(wp6SchemaTexts[wp6BinaryRefRef])},
			{Path: wp6ResponseResultPath, Bytes: []byte(wp6SchemaTexts[wp6ResponseResultRef])},
			{Path: wp6SelectorResultPath, Bytes: []byte(wp6SchemaTexts[wp6SelectorResultRef])},
		},
		extraToolsFiles: []File{
			{Path: wp6ProducerOutputPath, Bytes: []byte(wp6SchemaTexts[wp6ProducerOutputRef])},
			{Path: wp6ProducerOutputBadPath, Bytes: []byte(wp6SchemaTexts[wp6ProducerOutputBadRef])},
			{Path: wp6ProducerOutputSelPath, Bytes: []byte(wp6SchemaTexts[wp6ProducerOutputSelRef])},
			{Path: wp6ProducerInputPath, Bytes: []byte(wp6SchemaTexts[wp6ProducerInputRef])},
		},
		mutateContext: func(context *TrustedBuildContext) {
			for _, binding := range []struct{ ref, path string }{
				{wp6ResponseBodyRef, wp6ResponseBodyPath},
				{wp6MultipartRootRef, wp6MultipartRootPath},
				{wp6BinaryRefRef, wp6BinaryRefPath},
				{wp6ResponseResultRef, wp6ResponseResultPath},
				{wp6SelectorResultRef, wp6SelectorResultPath},
			} {
				context.ArtifactBindings = append(context.ArtifactBindings, TrustedArtifactBinding{
					ArtifactRef: binding.ref, DependencyID: wp3IntegrationDependency,
					SnapshotPath: binding.path, Dialect: "json-schema-2020-12",
				})
				context.SchemaBindings = append(context.SchemaBindings, TrustedSchemaBinding{
					ExportID: binding.ref, ArtifactRef: binding.ref, Pointer: "#",
				})
			}
			for _, binding := range []struct{ ref, path string }{
				{wp6ProducerOutputRef, wp6ProducerOutputPath},
				{wp6ProducerOutputBadRef, wp6ProducerOutputBadPath},
				{wp6ProducerOutputSelRef, wp6ProducerOutputSelPath},
				{wp6ProducerInputRef, wp6ProducerInputPath},
			} {
				context.ArtifactBindings = append(context.ArtifactBindings, TrustedArtifactBinding{
					ArtifactRef: binding.ref, DependencyID: wp3ToolsDependency,
					SnapshotPath: binding.path, Dialect: "json-schema-2020-12",
				})
				context.SchemaBindings = append(context.SchemaBindings, TrustedSchemaBinding{
					ExportID: binding.ref, ArtifactRef: binding.ref, Pointer: "#",
				})
			}
		},
	})
	return request
}

func wp6Check(t *testing.T, options wp6FixtureOptions) []Diagnostic {
	t.Helper()
	return Check(wp6BuildRequest(t, options))
}

func wp6DiagnosticCodes(diagnostics []Diagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

func wp6AssertCodes(t *testing.T, diagnostics []Diagnostic, wantCodes ...string) {
	t.Helper()
	got := wp6DiagnosticCodes(diagnostics)
	if len(got) != len(wantCodes) {
		t.Fatalf("diagnostic codes = %v, want exactly %v (%#v)", got, wantCodes, diagnostics)
	}
	for index := range wantCodes {
		if got[index] != wantCodes[index] {
			t.Fatalf("diagnostic codes = %v, want %v (%#v)", got, wantCodes, diagnostics)
		}
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != "error" || diagnostic.Phase != "C_STATIC" {
			t.Fatalf("%s identity = %s/%s, want error/C_STATIC", diagnostic.Code, diagnostic.Severity, diagnostic.Phase)
		}
	}
}

// TestV1Alpha5WP6PositiveSingleBodylessDisposition covers the L.8.2 single fixed
// bodyless disposition exemption: paired response, default boundary at the
// Wait-bearing Node, no typed Producer required, all early terminals mapped.
func TestV1Alpha5WP6PositiveSingleBodylessDisposition(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{response: wp6SingleBodylessResponseBlock})
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero WP6 diagnostics for the single bodyless disposition, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP6PositiveJSONBodyProducer covers a paired JSON body-bearing
// response with an exact Producer, complete required-field mapping and early
// failure policy.
func TestV1Alpha5WP6PositiveJSONBodyProducer(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{})
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero WP6 diagnostics for the JSON body response, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP6PositiveDeclaredBoundary covers explicit wait.response.after
// boundary selection and post-dominance.
func TestV1Alpha5WP6PositiveDeclaredBoundary(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{
		procedure: wp6NamedBoundaryProcedure,
		response: func(d wp6Digests) string {
			return strings.Replace(wp6JSONBodyResponseBlock(d), "producer: operation", "producer: operation", 1)
		},
	})
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero WP6 diagnostics for the declared boundary, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP6PositiveMultipartResponse covers the multipart/related body
// bearing case: JSON root part, binary part mapping, Content-ID field and
// immutable binary-content lifecycle.
func TestV1Alpha5WP6PositiveMultipartResponse(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{response: wp6MultipartResponseBlock})
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero WP6 diagnostics for the multipart response, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP6PositiveMultiBodylessSelector covers the dispositionSelector
// coverage rule for multiple bodyless dispositions.
func TestV1Alpha5WP6PositiveMultiBodylessSelector(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{
		response:    wp6MultiBodylessSelectorResponseBlock,
		toolsOutput: wp6ProducerOutputSelRef,
	})
	// The selector-typed Producer output Schema is bound with its own identity by
	// the fixture helper; if it is not, the only expected diagnostic is the
	// missing Producer Schema.
	if len(diagnostics) != 0 {
		t.Fatalf("expected zero WP6 diagnostics for the multi-bodyless selector, got %#v", diagnostics)
	}
}

// TestV1Alpha5WP6WaitResponseOnUnpairedExport covers L.8.1: authoring
// wait.response is invalid when the linked export declares response.mode: none.
func TestV1Alpha5WP6WaitResponseOnUnpairedExport(t *testing.T) {
	procedure := strings.Replace(wp3PositiveProcedure, `      routes:
        TIMEOUT: handle-update-timeout`, `      response: {}
      routes:
        TIMEOUT: handle-update-timeout`, 1)
	diagnostics := wp3Check(t, wp3FixtureOptions{procedure: procedure})
	wp6AssertCodes(t, diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID")
	if !strings.Contains(diagnostics[0].Message,
		"Node 'wait-sm-context' wait.response.after does not identify one reachable response boundary on every accepted-event path.") {
		t.Fatalf("boundary diagnostic must use the L.11.1 template: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP6ExportRequiresLaterBoundary covers L.8.1: the export's Producer
// stage is not declared by the Wait-bearing Node, so wait.response.after is
// required and its omission is invalid.
func TestV1Alpha5WP6ExportRequiresLaterBoundary(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{
		response: func(d wp6Digests) string {
			return strings.Replace(wp6JSONBodyResponseBlock(d), "producer: operation", "producer: observation", 1)
		},
	})
	wp6AssertCodes(t, diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID")
	if diagnostics[0].Path != wp6ProcedurePathForAssert {
		t.Fatalf("boundary diagnostic path = %q, want %q", diagnostics[0].Path, wp6ProcedurePathForAssert)
	}
}

// TestV1Alpha5WP6BoundaryNodeWithoutProducerStage covers L.8.2: a declared
// boundary Node that does not declare the export's Producer stage is a mapping
// failure.
func TestV1Alpha5WP6BoundaryNodeWithoutProducerStage(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{procedure: wp6BoundaryWithoutProducerProcedure})
	wp6AssertCodes(t, diagnostics, "WAIT_RESPONSE_MAPPING_INVALID")
	if !strings.Contains(diagnostics[0].Message,
		"Node 'update-sm-context' response boundary has no valid typed Producer-to-response mapping.") {
		t.Fatalf("mapping diagnostic must use the L.11.1 template: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP6SuccessBeforeBoundary covers EW-019/L.8.1: an accepted path that
// reaches SUCCEEDED before the selected response boundary is invalid.
func TestV1Alpha5WP6SuccessBeforeBoundary(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{procedure: wp6SuccessBeforeBoundaryProcedure})
	wp6AssertCodes(t, diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID")
}

// TestV1Alpha5WP6UnreachableBoundary covers EW-030/L.11: response.after names a
// declared but unreachable Node.
func TestV1Alpha5WP6UnreachableBoundary(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{procedure: wp6UnreachableBoundaryProcedure})
	wp6AssertCodes(t, diagnostics, "WAIT_RESPONSE_BOUNDARY_INVALID")
}

// TestV1Alpha5WP6EarlyFailureMappingMissing covers L.8.1/§8.4: an accepted early
// terminal path before the response boundary requires an explicit trusted
// early-failure response mapping. The defect is Ingress-owned.
func TestV1Alpha5WP6EarlyFailureMappingMissing(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{
		response: wp6JSONBodyResponseBlockWithoutEarlyFailures,
	})
	wp6AssertCodes(t, diagnostics, "INGRESS_RESPONSE_CONTRACT_INVALID")
	if diagnostics[0].Path != wp3CatalogPath {
		t.Fatalf("Ingress contract diagnostic path = %q, want %q", diagnostics[0].Path, wp3CatalogPath)
	}
	if !strings.Contains(diagnostics[0].Message,
		"Ingress export 'nsmf-pdusession.update-sm-context' has an invalid paired-response contract at '/response/earlyFailureResponses'.") {
		t.Fatalf("Ingress contract diagnostic must use the registry template: %q", diagnostics[0].Message)
	}
}

// TestV1Alpha5WP6ProducerOutputSchemaIncompatible covers L.8.2/L.11.2: an
// incompatible Producer Output Schema reuses INPUT_TYPE_MISMATCH, not
// WAIT_RESPONSE_MAPPING_INVALID.
func TestV1Alpha5WP6ProducerOutputSchemaIncompatible(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{toolsOutput: wp6ProducerOutputBadRef})
	wp6AssertCodes(t, diagnostics, "INPUT_TYPE_MISMATCH")
}

// TestV1Alpha5WP6ProducerOutputSchemaUnavailable covers L.11.2: an unavailable
// Producer Output Schema reuses SOURCE_NOT_AVAILABLE.
func TestV1Alpha5WP6ProducerOutputSchemaUnavailable(t *testing.T) {
	request := wp6BuildRequest(t, wp6FixtureOptions{})
	// Remove the trusted Schema binding for the Producer Output Schema.
	filtered := make([]TrustedSchemaBinding, 0, len(request.TrustedBuildContext.SchemaBindings))
	for _, binding := range request.TrustedBuildContext.SchemaBindings {
		if binding.ExportID == wp6ProducerOutputRef {
			continue
		}
		filtered = append(filtered, binding)
	}
	request.TrustedBuildContext.SchemaBindings = filtered
	diagnostics := Check(request)
	wp6AssertCodes(t, diagnostics, "SOURCE_NOT_AVAILABLE")
}

// TestV1Alpha5WP6DispositionSelectorCoverageInvalid covers §8.3: the selector
// must cover every allowed disposition/representation combination with closed
// enum values. The defect is Ingress-owned.
func TestV1Alpha5WP6DispositionSelectorCoverageInvalid(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{
		response:    wp6BadSelectorResponseBlock,
		toolsOutput: wp6ProducerOutputSelRef,
	})
	wp6AssertCodes(t, diagnostics, "INGRESS_RESPONSE_CONTRACT_INVALID")
}

// TestV1Alpha5WP6MappingCoverageIncomplete covers §8.3: a body-bearing
// representation must supply every required protocol response field. The defect
// is Ingress-owned.
func TestV1Alpha5WP6MappingCoverageIncomplete(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{response: wp6JSONBodyResponseBlockMissingMapping})
	wp6AssertCodes(t, diagnostics, "INGRESS_RESPONSE_MAPPING_INVALID")
}

// TestV1Alpha5WP6MultipartBinaryPartUnmapped covers §8.3: every declared binary
// part must have exactly one mapping, and removing that mapping also leaves the
// Integration-generated Content-ID root field without a declared producer. Both
// defects are Ingress-owned.
func TestV1Alpha5WP6MultipartBinaryPartUnmapped(t *testing.T) {
	diagnostics := wp6Check(t, wp6FixtureOptions{response: wp6MultipartResponseBlockMissingBinaryMapping})
	wp6AssertCodes(t, diagnostics,
		"INGRESS_RESPONSE_MAPPING_INVALID", "INGRESS_RESPONSE_MAPPING_INVALID")
}

// TestV1Alpha5WP6Deterministic proves that repeated checks of the same request
// produce byte-identical diagnostic sequences.
func TestV1Alpha5WP6Deterministic(t *testing.T) {
	first := wp6Check(t, wp6FixtureOptions{procedure: wp6SuccessBeforeBoundaryProcedure})
	second := wp6Check(t, wp6FixtureOptions{procedure: wp6SuccessBeforeBoundaryProcedure})
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("WP6 diagnostics are not deterministic:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if len(first) == 0 {
		t.Fatalf("determinism fixture must produce at least one diagnostic")
	}
}

// TestV1Alpha5WP6BuildRemainsFailClosed proves that Build stays fail-closed after
// the WP6 static pass succeeds.
func TestV1Alpha5WP6BuildRemainsFailClosed(t *testing.T) {
	request := wp6BuildRequest(t, wp6FixtureOptions{})
	if diagnostics := Check(request); len(diagnostics) != 0 {
		t.Fatalf("positive WP6 fixture must pass Check, got %#v", diagnostics)
	}
	result := Build(request)
	if result.Success {
		t.Fatalf("WP6 static pass must not open the v1alpha5 Build gate")
	}
	if !hasV1Alpha5Diagnostic(result.Diagnostics, "CAPABILITY_INPUT_UNTRUSTED") {
		t.Fatalf("Build must stay fail-closed through the residual gate, got %#v", result.Diagnostics)
	}
}
