package compiler

// WP7b: v1alpha5 DependencyInspection ingress inventory.
//
// Ingress Operation exports are a separate inventory type (InspectedIngressOperation)
// and are never mixed into InspectedCallable. The v1alpha5 inspection is
// snapshot-scoped like the frozen v1alpha4 one: it validates the supplied
// immutable dependency snapshots and trusted dependency bindings, reuses the WP3
// trusted Ingress discovery (trusted Owner + exact ingressOperation export
// identity + verified Schema Closure digests) and projects only exports whose
// identity and Owner can be established. It never resolves author references,
// evaluates Capability Grants, authorizes invocation or publishes dependency
// bytes/digests.

import (
	"sort"
)

// InspectedIngressOperation is the independent DependencyInspection projection of
// one trusted Ingress Operation export. Kind is always
// "ingressOperation"; callables and ingress operations are distinct JSON
// arrays and must never be mixed.
type InspectedIngressOperation struct {
	DependencyID           string `json:"dependencyId"`
	Owner                  string `json:"owner"`
	Kind                   string `json:"kind"`
	ID                     string `json:"id"`
	SourcePath             string `json:"sourcePath"`
	EventSource            string `json:"eventSource"`
	EventService           string `json:"eventService"`
	EventOperation         string `json:"eventOperation"`
	ProtocolSpecification  string `json:"protocolSpecification,omitempty"`
	ProtocolRelease        string `json:"protocolRelease,omitempty"`
	ProtocolAPIVersion     string `json:"apiUriVersion,omitempty"`
	OperationID            string `json:"operationId"`
	Method                 string `json:"method"`
	Path                   string `json:"path"`
	ArtifactRef            string `json:"artifactRef"`
	ArtifactDigest         string `json:"artifactDigest"`
	NormalizedSchemaRef    string `json:"normalizedSchemaRef"`
	NormalizedSchemaDigest string `json:"normalizedSchemaDigest"`
}

// inspectV1Alpha5Dependencies is the exact-target dispatch target for
// InspectDependencies when BuildRequest.languageTarget is v1alpha5. It returns
// the independent v1alpha5 inspection API version and the ingress inventory; the
// inherited v1alpha4 projection (callables/schemas/diagnostics) is untouched.
func inspectV1Alpha5Dependencies(req BuildRequest) DependencyInspection {
	result := DependencyInspection{
		APIVersion:        DependencyInspectionAPIVersionV1Alpha5,
		Diagnostics:       []Diagnostic{},
		Callables:         []InspectedCallable{},
		Schemas:           []InspectedSchema{},
		IngressOperations: []InspectedIngressOperation{},
	}

	// Same strict v1alpha5 envelope/prerequisite admission as Check/Build.
	diagnostics := validateBuildRequestV1Alpha5(req)
	if len(diagnostics) != 0 {
		sortDiagnostics(diagnostics)
		result.Diagnostics = diagnostics
		return result
	}

	link, linkDiagnostics := buildV1Alpha5LinkContext(req)
	diagnostics = append(diagnostics, linkDiagnostics...)

	scope, scopeDiagnostics := v1alpha5InspectionDependencyScope(req)
	diagnostics = append(diagnostics, scopeDiagnostics...)

	exports, exportDiagnostics := discoverV1Alpha5IngressExports(scope, link)
	diagnostics = append(diagnostics, exportDiagnostics...)

	operations := make([]InspectedIngressOperation, 0, len(exports))
	for _, export := range exports {
		operations = append(operations, inspectedV1Alpha5IngressOperation(export))
	}
	sort.Slice(operations, func(i, j int) bool {
		left, right := operations[i], operations[j]
		if left.DependencyID != right.DependencyID {
			return left.DependencyID < right.DependencyID
		}
		if left.SourcePath != right.SourcePath {
			return left.SourcePath < right.SourcePath
		}
		return left.ID < right.ID
	})

	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	result.Diagnostics = diagnostics
	result.IngressOperations = operations
	return result
}

func inspectedV1Alpha5IngressOperation(export v1alpha5TrustedIngressExport) InspectedIngressOperation {
	operation := export.Operation
	return InspectedIngressOperation{
		DependencyID:           export.DependencyAlias,
		Owner:                  export.CatalogOwner,
		Kind:                   v1alpha5ExportKindIngressOperation,
		ID:                     operation.ID,
		SourcePath:             export.SnapshotPath,
		EventSource:            operation.Event.Source,
		EventService:           operation.Event.Service,
		EventOperation:         operation.Event.Operation,
		ProtocolSpecification:  operation.Protocol.Specification,
		ProtocolRelease:        operation.Protocol.Release,
		ProtocolAPIVersion:     operation.Protocol.APIVersion,
		OperationID:            operation.Protocol.OpenAPI.OperationID,
		Method:                 operation.Protocol.OpenAPI.Method,
		Path:                   operation.Protocol.OpenAPI.Path,
		ArtifactRef:            operation.Protocol.OpenAPI.ArtifactRef,
		ArtifactDigest:         operation.Protocol.OpenAPI.ArtifactDigest,
		NormalizedSchemaRef:    operation.Request.Normalized.SchemaRef,
		NormalizedSchemaDigest: operation.Request.Normalized.SchemaDigest,
	}
}

// v1alpha5InspectionDependencyScope verifies the same immutable snapshot
// prerequisites as the v1alpha5 static path (measured snapshot digest, exactly
// one trusted dependency binding with that digest, non-empty authorized Owners)
// and returns the trusted scope consumed by the shared WP3 ingress discovery.
// It deliberately does not consult Skill Contract references: inspection is
// snapshot-scoped, not selected-Skill-closure-scoped.
func v1alpha5InspectionDependencyScope(req BuildRequest) ([]v1alpha5Dependency, []Diagnostic) {
	var diagnostics []Diagnostic
	snapshots := append([]ResolvedDependency(nil), req.ResolvedDependencies...)
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].DependencyID < snapshots[j].DependencyID })

	scope := make([]v1alpha5Dependency, 0, len(snapshots))
	for _, snapshot := range snapshots {
		measured, err := ComputeDependencySnapshotDigest(snapshot.Files)
		if err != nil || measured != snapshot.Digest {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil,
				"DEPENDENCY_IDENTITY_UNVERIFIED",
				"Supplied DependencySnapshot bytes do not match the declared digest: "+snapshot.DependencyID))
			continue
		}
		var matches []TrustedDependencyBinding
		if req.TrustedBuildContext != nil {
			for _, binding := range req.TrustedBuildContext.DependencyBindings {
				if binding.DependencyID == snapshot.DependencyID && binding.Digest == measured {
					matches = append(matches, binding)
				}
			}
		}
		if len(matches) == 0 {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil,
				"DEPENDENCY_IDENTITY_UNVERIFIED",
				"No trusted dependency binding matches the supplied snapshot and measured digest: "+snapshot.DependencyID))
			continue
		}
		if len(matches) > 1 {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil,
				"DEPENDENCY_DUPLICATE",
				"Multiple trusted dependency bindings match the supplied snapshot and measured digest: "+snapshot.DependencyID))
			continue
		}
		binding := matches[0]
		if len(binding.AuthorizedOwners) == 0 {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic("", nil,
				"DEPENDENCY_IDENTITY_UNVERIFIED",
				"Trusted dependency binding declares no authorized Owner: "+snapshot.DependencyID))
			continue
		}
		scope = append(scope, v1alpha5Dependency{
			Alias:     snapshot.DependencyID,
			PackageID: binding.StablePackageID,
			Snapshot:  snapshot,
			Binding:   binding,
		})
	}
	return scope, diagnostics
}
