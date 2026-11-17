package compiler

// WP7b: v1alpha5 DependencyInspection ingress-inventory tests.
//
// All fixtures are the SYNTHETIC static-compiler fixtures shared with WP3/WP6.
// They are not real Owner artifacts, deployment profiles or 5GC/3GPP
// publication claims. The v1alpha5 inspection is snapshot-scoped and read-only.

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func v1alpha5InspectionDiagnosticCodes(inspection DependencyInspection) []string {
	codes := make([]string, 0, len(inspection.Diagnostics))
	for _, diagnostic := range inspection.Diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

// TestV1Alpha5DependencyInspectionPositiveIngressInventory covers the trusted
// positive path: exactly the declared trusted Ingress Operation export with
// exact projected fields, no callables mixing and no diagnostics.
func TestV1Alpha5DependencyInspectionPositiveIngressInventory(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{})
	inspection := InspectDependencies(request)

	if inspection.APIVersion != DependencyInspectionAPIVersionV1Alpha5 {
		t.Fatalf("v1alpha5 inspection apiVersion = %q, want %q",
			inspection.APIVersion, DependencyInspectionAPIVersionV1Alpha5)
	}
	if len(inspection.Diagnostics) != 0 {
		t.Fatalf("positive v1alpha5 inspection must emit no diagnostics, got %#v", inspection.Diagnostics)
	}
	if len(inspection.Callables) != 0 || len(inspection.Schemas) != 0 {
		t.Fatalf("v1alpha5 ingress inventory must not be mixed into callables/schemas: %#v", inspection)
	}
	if len(inspection.IngressOperations) != 1 {
		t.Fatalf("expected exactly one trusted Ingress Operation, got %#v", inspection.IngressOperations)
	}

	_, digests := wp3Artifacts(t)
	want := InspectedIngressOperation{
		DependencyID:           wp3IntegrationDependency,
		Owner:                  wp3IntegrationOwner,
		Kind:                   v1alpha5ExportKindIngressOperation,
		ID:                     wp3IngressExportID,
		SourcePath:             wp3CatalogPath,
		EventSource:            "AMF",
		EventService:           "Nsmf_PDUSession",
		EventOperation:         "UpdateSMContext",
		ProtocolSpecification:  "3GPP TS 29.502",
		ProtocolRelease:        "19.5.0",
		ProtocolAPIVersion:     "v1",
		OperationID:            "UpdateSmContext",
		Method:                 "POST",
		Path:                   "/sm-contexts/{smContextRef}/modify",
		ArtifactRef:            wp3OpenAPIArtifactRef,
		ArtifactDigest:         wp3Digest([]byte(wp3OpenAPIDocument)),
		NormalizedSchemaRef:    wp3NormalizedArtifactRef,
		NormalizedSchemaDigest: digests[wp3NormalizedArtifactRef],
	}
	if !reflect.DeepEqual(inspection.IngressOperations[0], want) {
		t.Fatalf("ingress operation projection =\n%#v\nwant\n%#v",
			inspection.IngressOperations[0], want)
	}

	// The JSON projection keeps a separate ingressOperations array and never
	// projects an ingress operation as a callable.
	encoded, err := json.Marshal(inspection)
	if err != nil {
		t.Fatalf("marshal v1alpha5 inspection: %v", err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("unmarshal v1alpha5 inspection: %v", err)
	}
	if _, ok := document["ingressOperations"]; !ok {
		t.Fatalf("v1alpha5 inspection JSON lacks ingressOperations: %s", encoded)
	}
	var callables []InspectedCallable
	if err := json.Unmarshal(document["callables"], &callables); err != nil {
		t.Fatalf("decode callables: %v", err)
	}
	if len(callables) != 0 {
		t.Fatalf("ingress operations must not be mixed into callables: %s", encoded)
	}
}

// TestV1Alpha5DependencyInspectionUntrustedCatalogsSuppressed covers the #62
// section 13 trusted-only discovery rule: an untrusted Catalog Owner or a missing
// exact ingressOperation export-owner record suppresses the whole Catalog without
// fabricating an ingress inventory or a Wait/Ingress alias diagnostic.
func TestV1Alpha5DependencyInspectionUntrustedCatalogsSuppressed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TrustedBuildContext)
	}{
		{
			name: "unauthorized-catalog-owner",
			mutate: func(context *TrustedBuildContext) {
				for index := range context.DependencyBindings {
					if context.DependencyBindings[index].DependencyID == wp3IntegrationDependency {
						context.DependencyBindings[index].AuthorizedOwners = []string{"example.other-owner"}
					}
				}
			},
		},
		{
			name: "missing-ingress-export-owner-record",
			mutate: func(context *TrustedBuildContext) {
				for index := range context.DependencyBindings {
					if context.DependencyBindings[index].DependencyID == wp3IntegrationDependency {
						context.DependencyBindings[index].ExportOwners = nil
					}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := wp3BuildRequest(t, wp3FixtureOptions{mutateContext: test.mutate})
			inspection := InspectDependencies(request)
			if inspection.APIVersion != DependencyInspectionAPIVersionV1Alpha5 {
				t.Fatalf("apiVersion = %q", inspection.APIVersion)
			}
			if len(inspection.IngressOperations) != 0 {
				t.Fatalf("untrusted Catalog must produce no ingress inventory: %#v",
					inspection.IngressOperations)
			}
			for _, diagnostic := range inspection.Diagnostics {
				if strings.HasPrefix(diagnostic.Code, "WAIT_") ||
					strings.HasPrefix(diagnostic.Code, "INGRESS_") {
					t.Fatalf("untrusted Catalog must not fabricate WAIT_*/INGRESS_* diagnostics: %#v",
						diagnostic)
				}
			}
		})
	}
}

// TestV1Alpha5DependencyInspectionDependencyIdentityPrerequisite asserts that an
// unverifiable snapshot/binding is a prerequisite diagnostic with no ingress
// inventory (no guessed discovery).
func TestV1Alpha5DependencyInspectionDependencyIdentityPrerequisite(t *testing.T) {
	request := wp3BuildRequest(t, wp3FixtureOptions{mutateContext: func(context *TrustedBuildContext) {
		for index := range context.DependencyBindings {
			if context.DependencyBindings[index].DependencyID == wp3IntegrationDependency {
				context.DependencyBindings[index].Digest =
					"sha256:1111111111111111111111111111111111111111111111111111111111111111"
			}
		}
	}})
	inspection := InspectDependencies(request)
	if len(inspection.IngressOperations) != 0 {
		t.Fatalf("dependency identity failure must suppress ingress inventory: %#v",
			inspection.IngressOperations)
	}
	found := false
	for _, diagnostic := range inspection.Diagnostics {
		if diagnostic.Code == "DEPENDENCY_IDENTITY_UNVERIFIED" {
			found = true
			if diagnostic.Phase != "C_STATIC" || diagnostic.Severity != "error" {
				t.Fatalf("unexpected dependency diagnostic identity: %#v", diagnostic)
			}
		}
	}
	if !found {
		t.Fatalf("expected DEPENDENCY_IDENTITY_UNVERIFIED, got %#v", inspection.Diagnostics)
	}
}

// TestV1Alpha5DependencyInspectionDeterministic covers deterministic ordering of
// multiple trusted ingress operations and byte-identical repeated inspection.
func TestV1Alpha5DependencyInspectionDeterministic(t *testing.T) {
	_, digests := wp3Artifacts(t)
	secondCatalog := wp3DefaultCatalogText(
		wp3Digest([]byte(wp3OpenAPIDocument)), digests[wp3RequestArtifactRef], digests[wp3NormalizedArtifactRef])
	secondCatalog = strings.Replace(secondCatalog,
		"id: nsmf-pdusession.update-sm-context",
		"id: nsmf-pdusession.update-sm-context.second", 1)
	secondCatalog = strings.Replace(secondCatalog,
		"operation: UpdateSMContext\n      timeout",
		"operation: UpdateSMContext.second\n      timeout", 1)
	request := wp3BuildRequest(t, wp3FixtureOptions{
		extraFiles:     []File{{Path: "ingress/catalog-second.yaml", Bytes: []byte(secondCatalog)}},
		extraOwners:    []ExportOwner{{ExportKind: v1alpha5ExportKindIngressOperation, ExportID: "nsmf-pdusession.update-sm-context.second", Owner: wp3IntegrationOwner}},
		extraOwnersFor: wp3IntegrationDependency,
	})
	first := InspectDependencies(request)
	second := InspectDependencies(request)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("v1alpha5 inspection must be deterministic:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if len(first.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", first.Diagnostics)
	}
	if len(first.IngressOperations) != 2 {
		t.Fatalf("expected two trusted ingress operations, got %#v", first.IngressOperations)
	}
	for index := 1; index < len(first.IngressOperations); index++ {
		left := first.IngressOperations[index-1]
		right := first.IngressOperations[index]
		if left.DependencyID > right.DependencyID ||
			(left.DependencyID == right.DependencyID && left.SourcePath > right.SourcePath) ||
			(left.DependencyID == right.DependencyID && left.SourcePath == right.SourcePath && left.ID > right.ID) {
			t.Fatalf("ingress operations are not deterministically sorted: %#v",
				first.IngressOperations)
		}
	}
}

// TestV1Alpha5DependencyInspectionAuthorOnlyRequest covers the author-only
// request shape used by the CLI: no dependency snapshots, no diagnostics and an
// empty (not fabricated) ingress inventory.
func TestV1Alpha5DependencyInspectionAuthorOnlyRequest(t *testing.T) {
	request := v1alpha5TestProcedureRequest(validV1Alpha5WaitProcedure())
	inspection := InspectDependencies(request)
	if inspection.APIVersion != DependencyInspectionAPIVersionV1Alpha5 {
		t.Fatalf("apiVersion = %q", inspection.APIVersion)
	}
	if len(inspection.Diagnostics) != 0 {
		t.Fatalf("author-only v1alpha5 inspection must emit no diagnostics, got %#v",
			inspection.Diagnostics)
	}
	if len(inspection.Callables) != 0 || len(inspection.Schemas) != 0 ||
		len(inspection.IngressOperations) != 0 {
		t.Fatalf("author-only v1alpha5 inspection must not fabricate inventory: %#v", inspection)
	}
}

// TestV1Alpha4DependencyInspectionUnchanged keeps the frozen v1alpha4 inspection
// JSON projection byte-compatible: the new ingress field is omitted and the
// existing callables/schemas projection is intact.
func TestV1Alpha4DependencyInspectionUnchanged(t *testing.T) {
	inspection := InspectDependencies(wp3dInspectRequest(t, wp3aBaseFixture(t)))
	if inspection.APIVersion != DependencyInspectionAPIVersion {
		t.Fatalf("v1alpha4 inspection apiVersion = %q, want %q",
			inspection.APIVersion, DependencyInspectionAPIVersion)
	}
	if len(inspection.Callables) == 0 || len(inspection.Schemas) == 0 {
		t.Fatalf("v1alpha4 inspection lost its callables/schemas projection: %#v", inspection)
	}
	if len(inspection.IngressOperations) != 0 {
		t.Fatalf("v1alpha4 inspection must not expose ingress operations: %#v",
			inspection.IngressOperations)
	}
	encoded, err := json.Marshal(inspection)
	if err != nil {
		t.Fatalf("marshal v1alpha4 inspection: %v", err)
	}
	if strings.Contains(string(encoded), "ingressOperations") {
		t.Fatalf("v1alpha4 inspection JSON gained the v1alpha5-only field: %s", encoded)
	}
}
