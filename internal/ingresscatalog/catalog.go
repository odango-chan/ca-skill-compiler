package ingresscatalog

import (
	"sort"
	"strconv"
	"strings"
)

const (
	IngressCatalogAPIVersion = "connectionagent.io/ingresscatalog/v1alpha1"
	IngressCatalogKind       = "IngressCatalog"

	openAPIArtifactRefField    = "/protocol/openapi/artifactRef"
	openAPIArtifactDigestField = "/protocol/openapi/artifactDigest"
	openAPIOperationIDField    = "/protocol/openapi/operationId"
	openAPIMethodField         = "/protocol/openapi/method"
	openAPIPathField           = "/protocol/openapi/path"
)

// CatalogInput is one IngressCatalog source file from a trusted dependency
// snapshot.
type CatalogInput struct {
	Bytes           []byte
	DependencyAlias string
	SnapshotPath    string
}

// TrustedSnapshot is the caller-declared immutable dependency context used to
// resolve protocol artifacts and Schema Closure roots. No acquisition, network
// retrieval or fallback lookup happens inside this package.
type TrustedSnapshot struct {
	Context       *ResolutionContext
	SchemaExports map[string]RootRef
}

// NewTrustedSnapshot builds a trusted snapshot view. The resolution context and
// the export binding map must come from the same trusted dependency snapshot.
func NewTrustedSnapshot(context *ResolutionContext, schemaExports map[string]RootRef) *TrustedSnapshot {
	copied := make(map[string]RootRef, len(schemaExports))
	for exportRef, root := range schemaExports {
		copied[exportRef] = root
	}
	return &TrustedSnapshot{Context: context, SchemaExports: copied}
}

// EventSelector is the exact wait.event selector tuple.
type EventSelector struct {
	Source    string `json:"source"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
}

// OpenAPIProtocol is the pinned OpenAPI protocol identity declared by an export.
type OpenAPIProtocol struct {
	ArtifactRef    string `json:"artifactRef"`
	ArtifactDigest string `json:"artifactDigest"`
	OperationID    string `json:"operationId"`
	Method         string `json:"method"`
	Path           string `json:"path"`
}

// Protocol is the protocol metadata projection of one export.
type Protocol struct {
	Specification string          `json:"specification"`
	Release       string          `json:"release"`
	APIVersion    string          `json:"apiUriVersion"`
	OpenAPI       OpenAPIProtocol `json:"openapi"`
}

// SchemaBinding is one declared Schema reference and its declared closure digest.
type SchemaBinding struct {
	SchemaRef    string `json:"schemaRef"`
	SchemaDigest string `json:"schemaDigest"`
}

// RequestProjection is the typed request projection of one export.
type RequestProjection struct {
	AcceptedMediaTypes []string        `json:"acceptedMediaTypes"`
	SourceSchemas      []SchemaBinding `json:"sourceSchemas"`
	Normalized         SchemaBinding   `json:"normalized"`
	MaxBytes           int64           `json:"maxBytes"`
}

// CorrelationKeyProjection is one declared normalized-request correlation key
// field (#62 §6.1). requestPath is an RFC 6901 JSON Pointer into the normalized
// request Schema; comparison is only ever the exact comparison.
type CorrelationKeyProjection struct {
	RequestPath string `json:"requestPath"`
	Dimension   string `json:"dimension"`
	Comparison  string `json:"comparison"`
}

// DuplicateIdentityProjection is the declared protocol duplicate identity source
// and its authoritative uniqueness scope (#62 §7.4).
type DuplicateIdentityProjection struct {
	IdentitySource string `json:"identitySource"`
	Scope          string `json:"scope"`
}

// StaleDeliveryProjection is the declared generation/stale-delivery strategy
// projection (#62 §7.2/§7.3). Only the fields needed by static linking are
// projected; the retention ordering rules remain a later validation slice.
type StaleDeliveryProjection struct {
	Strategy               string `json:"strategy"`
	ResourceKey            string `json:"resourceKey"`
	MaximumLateArrival     string `json:"maximumLateArrival"`
	ClockSkewAllowance     string `json:"clockSkewAllowance"`
	TombstoneRetention     string `json:"tombstoneRetention"`
	DiscriminatorRetention string `json:"discriminatorRetention"`
	BasisRef               string `json:"basisRef"`
	DiscriminatorKind      string `json:"discriminatorKind"`
	DiscriminatorPath      string `json:"discriminatorPath"`
}

// CorrelationProjection is the typed correlation declaration of one export.
type CorrelationProjection struct {
	ArmInputSchemaRef string                      `json:"armInputSchemaRef"`
	KeyFields         []CorrelationKeyProjection  `json:"keyFields"`
	Duplicate         DuplicateIdentityProjection `json:"duplicate"`
	StaleDelivery     StaleDeliveryProjection     `json:"staleDelivery"`
}

// ArmingBindingProjection is one exact arming binding (#62 §6.2). sourceKind is
// invocation-context for start arming and operation-argument for trigger arming;
// bindingPath is a JSON Pointer into the armInputSchema.
type ArmingBindingProjection struct {
	RequestKey  string `json:"requestKey"`
	BindingPath string `json:"bindingPath"`
	SourceKind  string `json:"sourceKind"`
	SourcePath  string `json:"sourcePath"`
}

// TriggerOperationProjection is the exact trusted ToolCatalog Operation identity
// of one declared trigger (#62 §6.2/L.4.2).
type TriggerOperationProjection struct {
	PackageID string `json:"packageId"`
	Owner     string `json:"owner"`
	Kind      string `json:"kind"`
	ID        string `json:"id"`
}

// TriggerProjection is one declared triggering Operation and its exact
// correlation bindings.
type TriggerProjection struct {
	Operation TriggerOperationProjection `json:"operation"`
	Bindings  []ArmingBindingProjection  `json:"bindings"`
}

// StartArmingProjection is the declared start-Node arming mode (#62 §6.2 A).
type StartArmingProjection struct {
	Allowed  bool                      `json:"allowed"`
	Bindings []ArmingBindingProjection `json:"bindings"`
}

// ArmingProjection is the declared Wait arming contract of one export.
type ArmingProjection struct {
	Start    StartArmingProjection `json:"start"`
	Triggers []TriggerProjection   `json:"triggers"`
}

// ResponsePartProjection is one declared multipart response part (#62 §8.3).
type ResponsePartProjection struct {
	Name         string `json:"name"`
	MediaType    string `json:"mediaType"`
	Kind         string `json:"kind"`
	Required     bool   `json:"required"`
	SchemaRef    string `json:"schemaRef,omitempty"`
	SchemaDigest string `json:"schemaDigest,omitempty"`
	MaxBytes     int64  `json:"maxBytes"`
}

// ResponseRepresentationProjection is one declared protocol response
// representation. It is projected for the WP6 response-boundary slice.
type ResponseRepresentationProjection struct {
	ID                   string                   `json:"id"`
	MediaType            string                   `json:"mediaType"`
	ResponseSchemaRef    string                   `json:"responseSchemaRef"`
	ResponseSchemaDigest string                   `json:"responseSchemaDigest"`
	MaxBytes             int64                    `json:"maxBytes"`
	RootPart             string                   `json:"rootPart,omitempty"`
	Parts                []ResponsePartProjection `json:"parts,omitempty"`
	MinBinaryParts       int64                    `json:"minBinaryParts,omitempty"`
	MaxBinaryParts       int64                    `json:"maxBinaryParts,omitempty"`
}

// ResponseDispositionProjection is one declared protocol response disposition.
type ResponseDispositionProjection struct {
	ID              string                             `json:"id"`
	Status          int64                              `json:"status"`
	Body            string                             `json:"body"`
	Representations []ResponseRepresentationProjection `json:"representations"`
}

// ResponseTargetProjection is one closed-set response mapping target (#62 §8.3).
// ContentIDPart/ContentIDPointer are populated only for multipart-binary targets:
// they carry the contentIdField JSON Pointer into the JSON root part that binds
// the Integration-generated wire Content-ID.
type ResponseTargetProjection struct {
	Kind             string `json:"kind"`
	Part             string `json:"part,omitempty"`
	Pointer          string `json:"pointer,omitempty"`
	ContentIDPart    string `json:"contentIdPart,omitempty"`
	ContentIDPointer string `json:"contentIdPointer,omitempty"`
}

// ResponseMappingProjection is one declared typed result-to-protocol-response
// mapping (#62 §8.3).
type ResponseMappingProjection struct {
	DispositionID    string                   `json:"dispositionId"`
	RepresentationID string                   `json:"representationId"`
	From             string                   `json:"from"`
	Target           ResponseTargetProjection `json:"target"`
	Transform        string                   `json:"transform"`
}

// ResponseSelectorCaseProjection is one declared dispositionSelector case.
type ResponseSelectorCaseProjection struct {
	Value            any    `json:"value"`
	DispositionID    string `json:"dispositionId"`
	RepresentationID string `json:"representationId,omitempty"`
}

// ResponseResultProjection is the declared paired-response result Producer and
// its typed result-to-protocol-response mapping.
type ResponseResultProjection struct {
	Producer                 string                           `json:"producer"`
	ResponseResultSchemaRef  string                           `json:"responseResultSchemaRef"`
	DispositionSelectorPath  string                           `json:"dispositionSelectorSourcePath"`
	DispositionSelectorCases []ResponseSelectorCaseProjection `json:"dispositionSelectorCases,omitempty"`
	Mappings                 []ResponseMappingProjection      `json:"mappings,omitempty"`
}

// ResponseBinaryContentProjection is the declared immutable typed binary-content
// reference lifecycle used by multipart/related responses (#62 §8.3/§8.5).
type ResponseBinaryContentProjection struct {
	ReferenceSchemaRef string `json:"referenceSchemaRef"`
	DigestAlgorithm    string `json:"digestAlgorithm"`
	Immutable          bool   `json:"immutable"`
	Access             string `json:"access"`
	Retention          string `json:"retention"`
	MaxBytes           int64  `json:"maxBytes"`
}

// ResponseDeliveryRecoveryProjection is the declared durable delivery-recovery
// contract projection (#62 §8.4/§8.5).
type ResponseDeliveryRecoveryProjection struct {
	Mode                   string `json:"mode"`
	RequestIdentity        string `json:"requestIdentity"`
	LostChannelDisposition string `json:"lostChannelDisposition"`
}

// ResponseProjection is the declared protocol response contract projection.
type ResponseProjection struct {
	Mode                  string                             `json:"mode"`
	DeadlinePolicyRef     string                             `json:"deadlinePolicyRef"`
	Result                ResponseResultProjection           `json:"result"`
	Dispositions          []ResponseDispositionProjection    `json:"dispositions"`
	EarlyFailureResponses map[string]string                  `json:"earlyFailureResponses"`
	BinaryContent         *ResponseBinaryContentProjection   `json:"binaryContent,omitempty"`
	DeliveryRecovery      ResponseDeliveryRecoveryProjection `json:"deliveryRecovery"`
}

// Operation is the deterministic projection of one IngressCatalog export for
// later WP3 trigger/correlation linking. Correlation, Arming and Response were
// added additively for WP3; they never change the WP1 projection semantics of the
// id/adapter/event/protocol/request fields.
type Operation struct {
	ID          string                `json:"id"`
	Adapter     string                `json:"adapter"`
	Event       EventSelector         `json:"event"`
	Protocol    Protocol              `json:"protocol"`
	Request     RequestProjection     `json:"request"`
	Correlation CorrelationProjection `json:"correlation"`
	Arming      ArmingProjection      `json:"arming"`
	Response    ResponseProjection    `json:"response"`
}

// Catalog is the deterministic typed projection of one admitted IngressCatalog.
type Catalog struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Owner      string      `json:"owner"`
	Operations []Operation `json:"operations"`
}

// OperationByID returns the unique operation with the exact id.
func (c *Catalog) OperationByID(id string) (Operation, bool) {
	if c == nil {
		return Operation{}, false
	}
	for _, operation := range c.Operations {
		if operation.ID == id {
			return operation, true
		}
	}
	return Operation{}, false
}

// DeclaredSchemaBinding is one declared (schemaRef, schemaDigest) pair with its
// exact catalog source locations. It is exposed for WP3+ linking and for the
// traceability documentation.
type DeclaredSchemaBinding struct {
	OperationIndex int
	Role           string
	SchemaRef      string
	SchemaDigest   string
	SchemaRefField string // canonical JSON Pointer to the schemaRef field
	DigestField    string // canonical JSON Pointer to the schemaDigest field
}

// VerifiedSchemaClosure is one successfully verified declared Schema Closure
// Digest.
type VerifiedSchemaClosure struct {
	SchemaRef      string
	Root           RootRef
	DeclaredDigest string
	ComputedDigest string
	NodeCount      int
}

// Admission is the result of IngressCatalog v1alpha1 static admission.
type Admission struct {
	Catalog                *Catalog
	Diagnostics            []Diagnostic
	DeclaredSchemaBindings []DeclaredSchemaBinding
	VerifiedSchemaClosures []VerifiedSchemaClosure
}

// Valid reports whether admission produced no blocking diagnostic.
func (a *Admission) Valid() bool {
	return a != nil && len(a.Diagnostics) == 0
}

// AdmitCatalog performs YAML parsing, structural validation against the copied
// v1alpha1 structural schema and the semantic checks implemented by this slice:
// export identity uniqueness, exact event-selector uniqueness, trusted protocol
// artifact binding, and declared Schema Closure Digest verification.
func AdmitCatalog(input CatalogInput, snapshot *TrustedSnapshot) *Admission {
	admission := &Admission{}
	parsed, err := parseStrictJSONValue(input.Bytes)
	if err != nil {
		admission.Diagnostics = append(admission.Diagnostics, reusedDiagnostic(
			CodeYAMLParseError,
			"IngressCatalog could not be parsed as YAML: "+err.Error(),
			input.DependencyAlias, input.SnapshotPath, "#"))
		return admission
	}
	issues, err := validateCatalogStructure(parsed)
	if err != nil {
		admission.Diagnostics = append(admission.Diagnostics, ingressDiagnostic(
			CodeIngressCatalogInvalid,
			renderMessage(ingressMessageTemplates[CodeIngressCatalogInvalid], "{catalogPath}", input.SnapshotPath),
			input.DependencyAlias, input.SnapshotPath, "#"))
		sortDiagnostics(admission.Diagnostics)
		return admission
	}
	if len(issues) > 0 {
		message := renderMessage(ingressMessageTemplates[CodeIngressCatalogInvalid], "{catalogPath}", input.SnapshotPath)
		for _, issue := range issues {
			admission.Diagnostics = append(admission.Diagnostics, ingressDiagnostic(
				CodeIngressCatalogInvalid, message,
				input.DependencyAlias, input.SnapshotPath, issue.pointer))
		}
		sortDiagnostics(admission.Diagnostics)
		return admission
	}

	root, ok := parsed.(map[string]any)
	if !ok {
		admission.Diagnostics = append(admission.Diagnostics, ingressDiagnostic(
			CodeIngressCatalogInvalid,
			renderMessage(ingressMessageTemplates[CodeIngressCatalogInvalid], "{catalogPath}", input.SnapshotPath),
			input.DependencyAlias, input.SnapshotPath, "#"))
		return admission
	}
	admission.Catalog = projectCatalog(root)
	admission.DeclaredSchemaBindings = collectDeclaredSchemaBindings(root)

	admission.Diagnostics = append(admission.Diagnostics, checkExportIDUniqueness(admission.Catalog, input)...)
	admission.Diagnostics = append(admission.Diagnostics, checkEventSelectorUniqueness(admission.Catalog, input)...)
	admission.Diagnostics = append(admission.Diagnostics, checkProtocolArtifacts(admission.Catalog, snapshot, input)...)

	verified, schemaDiagnostics := verifySchemaClosures(admission.DeclaredSchemaBindings, snapshot, input)
	admission.VerifiedSchemaClosures = verified
	admission.Diagnostics = append(admission.Diagnostics, schemaDiagnostics...)

	sortDiagnostics(admission.Diagnostics)
	return admission
}

func operationPointer(index int, suffix string) string {
	return "#/spec/operations/" + strconv.Itoa(index) + suffix
}

func checkExportIDUniqueness(catalog *Catalog, input CatalogInput) []Diagnostic {
	firstIndex := make(map[string]int, len(catalog.Operations))
	diagnostics := make([]Diagnostic, 0)
	for index, operation := range catalog.Operations {
		if earlier, exists := firstIndex[operation.ID]; exists {
			diagnostic := ingressDiagnostic(
				CodeIngressExportIDDuplicate,
				renderMessage(ingressMessageTemplates[CodeIngressExportIDDuplicate], "{exportId}", operation.ID),
				input.DependencyAlias, input.SnapshotPath, operationPointer(index, "/id"))
			diagnostic.RelatedLocations = []SourceLocation{{
				DependencyAlias: input.DependencyAlias,
				SnapshotPath:    input.SnapshotPath,
				JSONPointer:     operationPointer(earlier, "/id"),
			}}
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		firstIndex[operation.ID] = index
	}
	return diagnostics
}

func checkEventSelectorUniqueness(catalog *Catalog, input CatalogInput) []Diagnostic {
	type occurrence struct{ index int }
	first := make(map[string]occurrence, len(catalog.Operations))
	diagnostics := make([]Diagnostic, 0)
	for index, operation := range catalog.Operations {
		selector := operation.Event
		key := selector.Source + "\x00" + selector.Service + "\x00" + selector.Operation
		if earlier, exists := first[key]; exists {
			diagnostic := ingressDiagnostic(
				CodeIngressEventSelectorDuplicate,
				renderMessage(ingressMessageTemplates[CodeIngressEventSelectorDuplicate],
					"{source}", selector.Source, "{service}", selector.Service, "{operation}", selector.Operation),
				input.DependencyAlias, input.SnapshotPath, operationPointer(index, "/event"))
			diagnostic.RelatedLocations = []SourceLocation{{
				DependencyAlias: input.DependencyAlias,
				SnapshotPath:    input.SnapshotPath,
				JSONPointer:     operationPointer(earlier.index, "/event"),
			}}
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		first[key] = occurrence{index: index}
	}
	return diagnostics
}

type openAPIOperationMatch struct {
	path    string
	method  string
	pointer string
}

func findOpenAPIOperations(document map[string]any, operationID string) []openAPIOperationMatch {
	matches := make([]openAPIOperationMatch, 0, 1)
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return matches
	}
	pathNames := make([]string, 0, len(paths))
	for pathName := range paths {
		pathNames = append(pathNames, pathName)
	}
	sort.Strings(pathNames)
	for _, pathName := range pathNames {
		pathItem, ok := paths[pathName].(map[string]any)
		if !ok {
			continue
		}
		for _, method := range openAPIMethods {
			operation, ok := pathItem[method].(map[string]any)
			if !ok {
				continue
			}
			declared, _ := operation["operationId"].(string)
			if declared != operationID {
				continue
			}
			pointer := canonicalPointer(jsonPointer{"paths", pathName, method})
			matches = append(matches, openAPIOperationMatch{path: pathName, method: strings.ToUpper(method), pointer: pointer})
		}
	}
	return matches
}

func checkProtocolArtifacts(catalog *Catalog, snapshot *TrustedSnapshot, input CatalogInput) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	for index, operation := range catalog.Operations {
		openAPI := operation.Protocol.OpenAPI
		refPointer := operationPointer(index, openAPIArtifactRefField)
		digestPointer := operationPointer(index, openAPIArtifactDigestField)
		if snapshot == nil || snapshot.Context == nil {
			diagnostics = append(diagnostics, reusedDiagnostic(
				CodeDependencyIdentityUnverified,
				"OpenAPI artifact '"+openAPI.ArtifactRef+"' cannot be verified without a trusted dependency snapshot.",
				input.DependencyAlias, input.SnapshotPath, refPointer))
			continue
		}
		artifact, found := snapshot.Context.Artifact(openAPI.ArtifactRef)
		if !found {
			diagnostics = append(diagnostics, reusedDiagnostic(
				CodeDependencyIdentityUnverified,
				"OpenAPI artifact '"+openAPI.ArtifactRef+"' is not present in the trusted dependency snapshot.",
				input.DependencyAlias, input.SnapshotPath, refPointer))
			continue
		}
		if artifact.ArtifactDigest != openAPI.ArtifactDigest {
			diagnostics = append(diagnostics, reusedDiagnostic(
				CodeDependencyIdentityUnverified,
				"OpenAPI artifact '"+openAPI.ArtifactRef+"' raw-byte digest does not match the declared artifactDigest.",
				input.DependencyAlias, input.SnapshotPath, digestPointer))
			continue
		}
		document, isObject := artifact.Document.(map[string]any)
		if !isObject {
			diagnostics = append(diagnostics, protocolDiagnostic(operation, input, index,
				operationPointer(index, "/protocol/openapi")))
			continue
		}
		if version, _ := document["openapi"].(string); !strings.HasPrefix(version, "3.0.") {
			diagnostics = append(diagnostics, protocolDiagnostic(operation, input, index,
				operationPointer(index, "/protocol/openapi")))
			continue
		}
		matches := findOpenAPIOperations(document, openAPI.OperationID)
		switch len(matches) {
		case 0:
			diagnostics = append(diagnostics, protocolDiagnostic(operation, input, index,
				operationPointer(index, openAPIOperationIDField)))
			continue
		case 1:
		default:
			diagnostics = append(diagnostics, protocolDiagnostic(operation, input, index,
				operationPointer(index, openAPIOperationIDField)))
			continue
		}
		match := matches[0]
		related := SourceLocation{
			DependencyAlias: artifact.ArtifactRef,
			SnapshotPath:    artifact.SnapshotPath,
			JSONPointer:     match.pointer,
		}
		if match.method != strings.ToUpper(openAPI.Method) {
			diagnostic := protocolDiagnostic(operation, input, index, operationPointer(index, openAPIMethodField))
			diagnostic.RelatedLocations = []SourceLocation{related}
			diagnostics = append(diagnostics, diagnostic)
		}
		if match.path != openAPI.Path {
			diagnostic := protocolDiagnostic(operation, input, index, operationPointer(index, openAPIPathField))
			diagnostic.RelatedLocations = []SourceLocation{related}
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics
}

func protocolDiagnostic(operation Operation, input CatalogInput, index int, pointer string) Diagnostic {
	fieldPath := pointer
	diagnostic := ingressDiagnostic(
		CodeIngressProtocolArtifactInvalid,
		renderMessage(ingressMessageTemplates[CodeIngressProtocolArtifactInvalid], "{exportId}", operation.ID, "{fieldPath}", fieldPath),
		input.DependencyAlias, input.SnapshotPath, pointer)
	return diagnostic
}

func verifySchemaClosures(bindings []DeclaredSchemaBinding, snapshot *TrustedSnapshot, input CatalogInput) ([]VerifiedSchemaClosure, []Diagnostic) {
	verified := make([]VerifiedSchemaClosure, 0, len(bindings))
	diagnostics := make([]Diagnostic, 0)
	computed := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		if snapshot == nil || snapshot.Context == nil {
			diagnostics = append(diagnostics, reusedDiagnostic(
				CodeSchemaMissing,
				"Schema '"+binding.SchemaRef+"' cannot be resolved without a trusted dependency snapshot.",
				input.DependencyAlias, input.SnapshotPath, binding.SchemaRefField))
			continue
		}
		root, found := snapshot.SchemaExports[binding.SchemaRef]
		if !found {
			diagnostics = append(diagnostics, reusedDiagnostic(
				CodeSchemaMissing,
				"Schema '"+binding.SchemaRef+"' is not declared by the trusted dependency snapshot.",
				input.DependencyAlias, input.SnapshotPath, binding.SchemaRefField))
			continue
		}
		cacheKey := root.ArtifactRef + "\x00" + root.Pointer
		computedDigest, cached := computed[cacheKey]
		var result *ClosureResult
		if !cached {
			computedResult, err := snapshot.Context.ComputeSchemaClosure(root)
			if err != nil {
				diagnostics = append(diagnostics, reusedDiagnostic(
					CodeSchemaInvalid,
					"Schema '"+binding.SchemaRef+"' is invalid for Schema Closure Digest v1: "+err.Error(),
					input.DependencyAlias, input.SnapshotPath, binding.SchemaRefField))
				continue
			}
			result = computedResult
			computedDigest = result.Digest
			computed[cacheKey] = computedDigest
		}
		if computedDigest != binding.SchemaDigest {
			diagnostics = append(diagnostics, reusedDiagnostic(
				CodeSchemaInvalid,
				"Declared schemaDigest for '"+binding.SchemaRef+"' does not match the computed Schema Closure Digest.",
				input.DependencyAlias, input.SnapshotPath, binding.DigestField))
			continue
		}
		nodeCount := 0
		if result != nil {
			nodeCount = len(result.Nodes)
		} else {
			nodeCount = cachedNodeCount(snapshot.Context, root)
		}
		verified = append(verified, VerifiedSchemaClosure{
			SchemaRef:      binding.SchemaRef,
			Root:           root,
			DeclaredDigest: binding.SchemaDigest,
			ComputedDigest: computedDigest,
			NodeCount:      nodeCount,
		})
	}
	return verified, diagnostics
}

// cachedNodeCount recomputes only the node count for repeated bindings. It is
// called at most once per distinct closure because of the digest cache.
func cachedNodeCount(context *ResolutionContext, root RootRef) int {
	result, err := context.ComputeSchemaClosure(root)
	if err != nil {
		return 0
	}
	return len(result.Nodes)
}

func collectDeclaredSchemaBindings(root map[string]any) []DeclaredSchemaBinding {
	bindings := make([]DeclaredSchemaBinding, 0, 4)
	operations := arrayValue(root, "spec", "operations")
	for index, operationValue := range operations {
		operation, ok := operationValue.(map[string]any)
		if !ok {
			continue
		}
		request, _ := operation["request"].(map[string]any)
		if request != nil {
			for sourceIndex, sourceValue := range arrayField(request, "sourceSchemas") {
				source, ok := sourceValue.(map[string]any)
				if !ok {
					continue
				}
				appendSchemaBinding(&bindings, index, "request.sourceSchema",
					operationPointer(index, "/request/sourceSchemas/"+strconv.Itoa(sourceIndex)+"/schemaRef"),
					operationPointer(index, "/request/sourceSchemas/"+strconv.Itoa(sourceIndex)+"/schemaDigest"),
					stringField(source, "schemaRef"), stringField(source, "schemaDigest"))
			}
			appendSchemaBinding(&bindings, index, "request.normalizedSchema",
				operationPointer(index, "/request/normalizedSchemaRef"),
				operationPointer(index, "/request/normalizedSchemaDigest"),
				stringField(request, "normalizedSchemaRef"), stringField(request, "normalizedSchemaDigest"))
			if normalization, ok := request["normalization"].(map[string]any); ok {
				for representationIndex, representationValue := range arrayField(normalization, "representations") {
					representation, ok := representationValue.(map[string]any)
					if !ok {
						continue
					}
					base := "/request/normalization/representations/" + strconv.Itoa(representationIndex)
					for partIndex, partValue := range arrayField(representation, "parts") {
						part, ok := partValue.(map[string]any)
						if !ok || stringField(part, "kind") != "json" {
							continue
						}
						partPath := base + "/parts/" + strconv.Itoa(partIndex)
						appendSchemaBinding(&bindings, index, "request.normalization.jsonPart",
							operationPointer(index, partPath+"/schemaRef"),
							operationPointer(index, partPath+"/schemaDigest"),
							stringField(part, "schemaRef"), stringField(part, "schemaDigest"))
					}
				}
			}
		}
		response, _ := operation["response"].(map[string]any)
		if response == nil {
			continue
		}
		for dispositionIndex, dispositionValue := range arrayField(response, "dispositions") {
			disposition, ok := dispositionValue.(map[string]any)
			if !ok {
				continue
			}
			for representationIndex, representationValue := range arrayField(disposition, "representations") {
				representation, ok := representationValue.(map[string]any)
				if !ok {
					continue
				}
				representationPath := "/response/dispositions/" + strconv.Itoa(dispositionIndex) +
					"/representations/" + strconv.Itoa(representationIndex)
				appendSchemaBinding(&bindings, index, "response.representation",
					operationPointer(index, representationPath+"/responseSchemaRef"),
					operationPointer(index, representationPath+"/responseSchemaDigest"),
					stringField(representation, "responseSchemaRef"), stringField(representation, "responseSchemaDigest"))
				for partIndex, partValue := range arrayField(representation, "parts") {
					part, ok := partValue.(map[string]any)
					if !ok || stringField(part, "kind") != "json" {
						continue
					}
					partPath := representationPath + "/parts/" + strconv.Itoa(partIndex)
					appendSchemaBinding(&bindings, index, "response.jsonPart",
						operationPointer(index, partPath+"/schemaRef"),
						operationPointer(index, partPath+"/schemaDigest"),
						stringField(part, "schemaRef"), stringField(part, "schemaDigest"))
				}
			}
		}
	}
	return bindings
}

func appendSchemaBinding(bindings *[]DeclaredSchemaBinding, operationIndex int, role, refField, digestField, ref, digest string) {
	if ref == "" || digest == "" {
		return
	}
	*bindings = append(*bindings, DeclaredSchemaBinding{
		OperationIndex: operationIndex,
		Role:           role,
		SchemaRef:      ref,
		SchemaDigest:   digest,
		SchemaRefField: refField,
		DigestField:    digestField,
	})
}

func projectCatalog(root map[string]any) *Catalog {
	catalog := &Catalog{
		APIVersion: stringField(root, "apiVersion"),
		Kind:       stringField(root, "kind"),
		Owner:      stringField(objectField(root, "metadata"), "owner"),
	}
	for _, operationValue := range arrayValue(root, "spec", "operations") {
		operationMap, ok := operationValue.(map[string]any)
		if !ok {
			continue
		}
		operation := Operation{
			ID:      stringField(operationMap, "id"),
			Adapter: stringField(operationMap, "adapter"),
			Event: EventSelector{
				Source:    stringField(objectField(operationMap, "event"), "source"),
				Service:   stringField(objectField(operationMap, "event"), "service"),
				Operation: stringField(objectField(operationMap, "event"), "operation"),
			},
		}
		protocolMap := objectField(operationMap, "protocol")
		operation.Protocol = Protocol{
			Specification: stringField(protocolMap, "specification"),
			Release:       stringField(protocolMap, "release"),
			APIVersion:    stringField(protocolMap, "apiUriVersion"),
			OpenAPI: OpenAPIProtocol{
				ArtifactRef:    stringField(objectField(protocolMap, "openapi"), "artifactRef"),
				ArtifactDigest: stringField(objectField(protocolMap, "openapi"), "artifactDigest"),
				OperationID:    stringField(objectField(protocolMap, "openapi"), "operationId"),
				Method:         stringField(objectField(protocolMap, "openapi"), "method"),
				Path:           stringField(objectField(protocolMap, "openapi"), "path"),
			},
		}
		requestMap := objectField(operationMap, "request")
		operation.Request = RequestProjection{
			MaxBytes: integerField(requestMap, "maxBytes"),
			Normalized: SchemaBinding{
				SchemaRef:    stringField(requestMap, "normalizedSchemaRef"),
				SchemaDigest: stringField(requestMap, "normalizedSchemaDigest"),
			},
		}
		for _, mediaType := range arrayField(requestMap, "acceptedMediaTypes") {
			if text, ok := mediaType.(string); ok {
				operation.Request.AcceptedMediaTypes = append(operation.Request.AcceptedMediaTypes, text)
			}
		}
		for _, sourceValue := range arrayField(requestMap, "sourceSchemas") {
			source, ok := sourceValue.(map[string]any)
			if !ok {
				continue
			}
			operation.Request.SourceSchemas = append(operation.Request.SourceSchemas, SchemaBinding{
				SchemaRef:    stringField(source, "schemaRef"),
				SchemaDigest: stringField(source, "schemaDigest"),
			})
		}
		operation.Correlation = projectCorrelation(objectField(operationMap, "correlation"))
		operation.Arming = projectArming(objectField(operationMap, "arming"))
		operation.Response = projectResponse(objectField(operationMap, "response"))
		catalog.Operations = append(catalog.Operations, operation)
	}
	return catalog
}

// projectCorrelation projects the additive correlation declaration. Absent or
// malformed sub-objects project to their zero value; structural validity is
// owned by catalog admission, not by this projection.
func projectCorrelation(correlation map[string]any) CorrelationProjection {
	projection := CorrelationProjection{}
	if correlation == nil {
		return projection
	}
	projection.ArmInputSchemaRef = stringField(correlation, "armInputSchemaRef")
	for _, keyValue := range arrayField(correlation, "keyFields") {
		key, ok := keyValue.(map[string]any)
		if !ok {
			continue
		}
		projection.KeyFields = append(projection.KeyFields, CorrelationKeyProjection{
			RequestPath: stringField(key, "requestPath"),
			Dimension:   stringField(key, "dimension"),
			Comparison:  stringField(key, "comparison"),
		})
	}
	duplicate := objectField(correlation, "duplicate")
	projection.Duplicate = DuplicateIdentityProjection{
		IdentitySource: stringField(duplicate, "identitySource"),
		Scope:          stringField(duplicate, "scope"),
	}
	stale := objectField(correlation, "staleDelivery")
	projection.StaleDelivery = StaleDeliveryProjection{
		Strategy:               stringField(stale, "strategy"),
		ResourceKey:            stringField(stale, "resourceKey"),
		MaximumLateArrival:     stringField(stale, "maximumLateArrival"),
		ClockSkewAllowance:     stringField(stale, "clockSkewAllowance"),
		TombstoneRetention:     stringField(stale, "tombstoneRetention"),
		DiscriminatorRetention: stringField(stale, "discriminatorRetention"),
		BasisRef:               stringField(stale, "basisRef"),
	}
	discriminator := objectField(stale, "discriminator")
	projection.StaleDelivery.DiscriminatorKind = stringField(discriminator, "kind")
	projection.StaleDelivery.DiscriminatorPath = stringField(discriminator, "path")
	return projection
}

// projectArming projects the additive arming declaration (start mode plus the
// exact trigger set and their bindings).
func projectArming(arming map[string]any) ArmingProjection {
	projection := ArmingProjection{}
	if arming == nil {
		return projection
	}
	start := objectField(arming, "start")
	projection.Start.Allowed = booleanField(start, "allowed")
	projection.Start.Bindings = projectArmingBindings(arrayField(start, "bindings"))
	for _, triggerValue := range arrayField(arming, "triggers") {
		triggerMap, ok := triggerValue.(map[string]any)
		if !ok {
			continue
		}
		operation := objectField(triggerMap, "operation")
		projection.Triggers = append(projection.Triggers, TriggerProjection{
			Operation: TriggerOperationProjection{
				PackageID: stringField(operation, "packageId"),
				Owner:     stringField(operation, "owner"),
				Kind:      stringField(operation, "kind"),
				ID:        stringField(operation, "id"),
			},
			Bindings: projectArmingBindings(arrayField(triggerMap, "bindings")),
		})
	}
	return projection
}

func projectArmingBindings(values []any) []ArmingBindingProjection {
	bindings := make([]ArmingBindingProjection, 0, len(values))
	for _, value := range values {
		binding, ok := value.(map[string]any)
		if !ok {
			continue
		}
		bindings = append(bindings, ArmingBindingProjection{
			RequestKey:  stringField(binding, "requestKey"),
			BindingPath: stringField(binding, "bindingPath"),
			SourceKind:  stringField(objectField(binding, "source"), "kind"),
			SourcePath:  stringField(objectField(binding, "source"), "path"),
		})
	}
	return bindings
}

// projectResponse projects the additive paired-response contract used by the
// WP6 response-boundary slice. Absent or malformed sub-objects project to their
// zero value; structural validity is owned by catalog admission, not by this
// projection.
func projectResponse(response map[string]any) ResponseProjection {
	projection := ResponseProjection{}
	if response == nil {
		return projection
	}
	projection.Mode = stringField(response, "mode")
	projection.DeadlinePolicyRef = stringField(response, "deadlinePolicyRef")
	result := objectField(response, "result")
	selector := objectField(result, "dispositionSelector")
	projection.Result = ResponseResultProjection{
		Producer:                stringField(result, "producer"),
		ResponseResultSchemaRef: stringField(result, "responseResultSchemaRef"),
		DispositionSelectorPath: stringField(selector, "sourcePath"),
	}
	for _, caseValue := range arrayField(selector, "cases") {
		caseMap, ok := caseValue.(map[string]any)
		if !ok {
			continue
		}
		projection.Result.DispositionSelectorCases = append(projection.Result.DispositionSelectorCases,
			ResponseSelectorCaseProjection{
				Value:            caseMap["value"],
				DispositionID:    stringField(caseMap, "dispositionId"),
				RepresentationID: stringField(caseMap, "representationId"),
			})
	}
	for _, mappingValue := range arrayField(result, "mappings") {
		mappingMap, ok := mappingValue.(map[string]any)
		if !ok {
			continue
		}
		target := objectField(mappingMap, "target")
		contentIDField := objectField(target, "contentIdField")
		projection.Result.Mappings = append(projection.Result.Mappings, ResponseMappingProjection{
			DispositionID:    stringField(mappingMap, "dispositionId"),
			RepresentationID: stringField(mappingMap, "representationId"),
			From:             stringField(mappingMap, "from"),
			Transform:        stringField(mappingMap, "transform"),
			Target: ResponseTargetProjection{
				Kind:             stringField(target, "kind"),
				Part:             stringField(target, "part"),
				Pointer:          stringField(target, "pointer"),
				ContentIDPart:    stringField(contentIDField, "part"),
				ContentIDPointer: stringField(contentIDField, "pointer"),
			},
		})
	}
	for _, dispositionValue := range arrayField(response, "dispositions") {
		dispositionMap, ok := dispositionValue.(map[string]any)
		if !ok {
			continue
		}
		disposition := ResponseDispositionProjection{
			ID:     stringField(dispositionMap, "id"),
			Status: integerField(dispositionMap, "status"),
			Body:   stringField(dispositionMap, "body"),
		}
		for _, representationValue := range arrayField(dispositionMap, "representations") {
			representation, ok := representationValue.(map[string]any)
			if !ok {
				continue
			}
			projected := ResponseRepresentationProjection{
				ID:                   stringField(representation, "id"),
				MediaType:            stringField(representation, "mediaType"),
				ResponseSchemaRef:    stringField(representation, "responseSchemaRef"),
				ResponseSchemaDigest: stringField(representation, "responseSchemaDigest"),
				MaxBytes:             integerField(representation, "maxBytes"),
				RootPart:             stringField(representation, "rootPart"),
				MinBinaryParts:       integerField(representation, "minBinaryParts"),
				MaxBinaryParts:       integerField(representation, "maxBinaryParts"),
			}
			for _, partValue := range arrayField(representation, "parts") {
				partMap, ok := partValue.(map[string]any)
				if !ok {
					continue
				}
				projected.Parts = append(projected.Parts, ResponsePartProjection{
					Name:         stringField(partMap, "name"),
					MediaType:    stringField(partMap, "mediaType"),
					Kind:         stringField(partMap, "kind"),
					Required:     booleanField(partMap, "required"),
					SchemaRef:    stringField(partMap, "schemaRef"),
					SchemaDigest: stringField(partMap, "schemaDigest"),
					MaxBytes:     integerField(partMap, "maxBytes"),
				})
			}
			disposition.Representations = append(disposition.Representations, projected)
		}
		projection.Dispositions = append(projection.Dispositions, disposition)
	}
	if earlyFailures := objectField(response, "earlyFailureResponses"); earlyFailures != nil {
		projection.EarlyFailureResponses = make(map[string]string, len(earlyFailures))
		for outcome, value := range earlyFailures {
			entry, _ := value.(map[string]any)
			projection.EarlyFailureResponses[outcome] = stringField(entry, "responseRef")
		}
	}
	if binaryContent := objectField(response, "binaryContent"); binaryContent != nil {
		projection.BinaryContent = &ResponseBinaryContentProjection{
			ReferenceSchemaRef: stringField(binaryContent, "referenceSchemaRef"),
			DigestAlgorithm:    stringField(binaryContent, "digestAlgorithm"),
			Immutable:          booleanField(binaryContent, "immutable"),
			Access:             stringField(binaryContent, "access"),
			Retention:          stringField(binaryContent, "retention"),
			MaxBytes:           integerField(binaryContent, "maxBytes"),
		}
	}
	deliveryRecovery := objectField(response, "deliveryRecovery")
	projection.DeliveryRecovery = ResponseDeliveryRecoveryProjection{
		Mode:                   stringField(deliveryRecovery, "mode"),
		RequestIdentity:        stringField(deliveryRecovery, "requestIdentity"),
		LostChannelDisposition: stringField(deliveryRecovery, "lostChannelDisposition"),
	}
	return projection
}

func objectField(object map[string]any, field string) map[string]any {
	if object == nil {
		return nil
	}
	child, _ := object[field].(map[string]any)
	return child
}

func arrayField(object map[string]any, field string) []any {
	if object == nil {
		return nil
	}
	child, _ := object[field].([]any)
	return child
}

func arrayValue(root map[string]any, fields ...string) []any {
	current := root
	for index, field := range fields {
		if index == len(fields)-1 {
			return arrayField(current, field)
		}
		current = objectField(current, field)
		if current == nil {
			return nil
		}
	}
	return nil
}

func stringField(object map[string]any, field string) string {
	if object == nil {
		return ""
	}
	text, _ := object[field].(string)
	return text
}

func integerField(object map[string]any, field string) int64 {
	if object == nil {
		return 0
	}
	switch value := object[field].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	}
	return 0
}

func booleanField(object map[string]any, field string) bool {
	if object == nil {
		return false
	}
	value, _ := object[field].(bool)
	return value
}
