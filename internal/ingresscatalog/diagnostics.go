package ingresscatalog

import (
	"sort"
	"strings"
)

// Ingress-owned static diagnostic identities from
// connectionagent.io/ingress-diagnostics/v1alpha1 (candidate/DRAFT).
const (
	RegistryIDIngressDiagnostics = "connectionagent.io/ingress-diagnostics/v1alpha1"

	PhaseCStatic  = "C_STATIC"
	SeverityError = "ERROR"

	CodeIngressCatalogInvalid          = "INGRESS_CATALOG_INVALID"
	CodeIngressExportIDDuplicate       = "INGRESS_EXPORT_ID_DUPLICATE"
	CodeIngressEventSelectorDuplicate  = "INGRESS_EVENT_SELECTOR_DUPLICATE"
	CodeIngressProtocolArtifactInvalid = "INGRESS_PROTOCOL_ARTIFACT_INVALID"
	// CodeIngressCorrelationInvalid is the Ingress-owned identity for an invalid
	// correlation or stale-delivery declaration. WP3 uses it for export-side
	// correlation/arming declaration failures; Skill-side failures keep the
	// CA-SL WAIT_CORRELATION_BINDING_INVALID identity.
	CodeIngressCorrelationInvalid = "INGRESS_CORRELATION_INVALID"

	// Reused globally-unique diagnostic identities that this slice needs. They
	// are not Ingress-owned, so they carry no Ingress messageKey/registryId.
	CodeYAMLParseError               = "YAML_PARSE_ERROR"
	CodeDependencyIdentityUnverified = "DEPENDENCY_IDENTITY_UNVERIFIED"
	CodeSchemaMissing                = "SCHEMA_MISSING"
	CodeSchemaInvalid                = "SCHEMA_INVALID"
)

// ingressMessageKeys and ingressMessageTemplates are normative strings from the
// machine-readable Ingress diagnostic registry.
var ingressMessageKeys = map[string]string{
	CodeIngressCatalogInvalid:          "ingress.catalog.invalid",
	CodeIngressExportIDDuplicate:       "ingress.export.id.duplicate",
	CodeIngressEventSelectorDuplicate:  "ingress.event.selector.duplicate",
	CodeIngressProtocolArtifactInvalid: "ingress.protocol.artifact.invalid",
	CodeIngressCorrelationInvalid:      "ingress.correlation.invalid",
}

var ingressMessageTemplates = map[string]string{
	CodeIngressCatalogInvalid:          "IngressCatalog at '{catalogPath}' is structurally invalid or declares an unsupported apiVersion.",
	CodeIngressExportIDDuplicate:       "Ingress Operation export id '{exportId}' is duplicated in this trusted Catalog.",
	CodeIngressEventSelectorDuplicate:  "Ingress event selector '{source}/{service}/{operation}' is declared by more than one export.",
	CodeIngressProtocolArtifactInvalid: "Ingress export '{exportId}' does not match the pinned protocol artifact metadata at '{fieldPath}'.",
	CodeIngressCorrelationInvalid:      "Ingress export '{exportId}' has an invalid correlation or stale-delivery declaration at '{fieldPath}'.",
}

// IngressMessageKeyForCode returns the stable message key of one Ingress-owned
// diagnostic code.
func IngressMessageKeyForCode(code string) (string, bool) {
	key, ok := ingressMessageKeys[code]
	return key, ok
}

// RenderIngressMessage renders the canonical message template of one
// Ingress-owned diagnostic code with its exact "{name}" placeholders. The second
// result is false for a code that this registry does not own.
func RenderIngressMessage(code string, pairs ...string) (string, bool) {
	template, ok := ingressMessageTemplates[code]
	if !ok {
		return "", false
	}
	return renderMessage(template, pairs...), true
}

// diagnosticPrecedence follows #62 §13: parse/prerequisite failures come first,
// then catalog structure, export identity, protocol artifact and Schema closure.
var diagnosticPrecedence = map[string]int{
	CodeYAMLParseError:                 0,
	CodeDependencyIdentityUnverified:   5,
	CodeIngressCatalogInvalid:          10,
	CodeIngressExportIDDuplicate:       20,
	CodeIngressEventSelectorDuplicate:  21,
	CodeIngressProtocolArtifactInvalid: 30,
	CodeSchemaMissing:                  60,
	CodeSchemaInvalid:                  61,
}

// SourceLocation is the logical diagnostic source location required by the
// registry: dependency alias, snapshot path and RFC 6901 JSON Pointer.
type SourceLocation struct {
	DependencyAlias string `json:"dependencyAlias"`
	SnapshotPath    string `json:"snapshotPath"`
	JSONPointer     string `json:"jsonPointer"`
}

// Diagnostic is the package-local logical Compiler diagnostic record for the
// Ingress slice. registryId is provenance, not a second identity namespace;
// Ingress-owned codes always carry the Ingress registry and C_STATIC/ERROR.
type Diagnostic struct {
	Code             string           `json:"code"`
	Phase            string           `json:"phase"`
	Severity         string           `json:"severity"`
	RegistryID       string           `json:"registryId,omitempty"`
	MessageKey       string           `json:"messageKey,omitempty"`
	Message          string           `json:"message"`
	SourceLocation   SourceLocation   `json:"sourceLocation"`
	RelatedLocations []SourceLocation `json:"relatedLocations,omitempty"`
}

func ingressDiagnostic(code, message, dependencyAlias, snapshotPath, pointer string) Diagnostic {
	diagnostic := Diagnostic{
		Code:       code,
		Phase:      PhaseCStatic,
		Severity:   SeverityError,
		RegistryID: RegistryIDIngressDiagnostics,
		MessageKey: ingressMessageKeys[code],
		Message:    message,
		SourceLocation: SourceLocation{
			DependencyAlias: dependencyAlias,
			SnapshotPath:    snapshotPath,
			JSONPointer:     pointer,
		},
	}
	return diagnostic
}

func reusedDiagnostic(code, message, dependencyAlias, snapshotPath, pointer string) Diagnostic {
	return Diagnostic{
		Code:     code,
		Phase:    PhaseCStatic,
		Severity: SeverityError,
		Message:  message,
		SourceLocation: SourceLocation{
			DependencyAlias: dependencyAlias,
			SnapshotPath:    snapshotPath,
			JSONPointer:     pointer,
		},
	}
}

func diagnosticPrecedenceOf(code string) int {
	if precedence, ok := diagnosticPrecedence[code]; ok {
		return precedence
	}
	return 1000
}

// renderMessage fills the registry's canonical message template placeholders.
// The templates use "{name}" placeholders, not Go format verbs.
func renderMessage(template string, pairs ...string) string {
	return strings.NewReplacer(pairs...).Replace(template)
}

// sortDiagnostics applies the #62 §13 stable ordering: precedence, then
// dependency alias, snapshot path, JSON Pointer and Code as UTF-8 byte strings.
func sortDiagnostics(diagnostics []Diagnostic) {
	sort.SliceStable(diagnostics, func(left, right int) bool {
		leftDiagnostic, rightDiagnostic := diagnostics[left], diagnostics[right]
		leftPrecedence := diagnosticPrecedenceOf(leftDiagnostic.Code)
		rightPrecedence := diagnosticPrecedenceOf(rightDiagnostic.Code)
		if leftPrecedence != rightPrecedence {
			return leftPrecedence < rightPrecedence
		}
		if leftDiagnostic.SourceLocation.DependencyAlias != rightDiagnostic.SourceLocation.DependencyAlias {
			return leftDiagnostic.SourceLocation.DependencyAlias < rightDiagnostic.SourceLocation.DependencyAlias
		}
		if leftDiagnostic.SourceLocation.SnapshotPath != rightDiagnostic.SourceLocation.SnapshotPath {
			return leftDiagnostic.SourceLocation.SnapshotPath < rightDiagnostic.SourceLocation.SnapshotPath
		}
		if leftDiagnostic.SourceLocation.JSONPointer != rightDiagnostic.SourceLocation.JSONPointer {
			return leftDiagnostic.SourceLocation.JSONPointer < rightDiagnostic.SourceLocation.JSONPointer
		}
		return leftDiagnostic.Code < rightDiagnostic.Code
	})
}
