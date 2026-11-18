package ingresscatalog

// Pinned OpenAPI response-shape cross-check.
//
// The released IngressCatalog v1alpha1 contract §8.3 requires the declared
// paired response representations to agree with the pinned protocol artifact:
// when the operation's pinned OpenAPI response declares a multipart/related
// representation, the export may not declare only JSON (or a bodyless)
// representation, because binary parts can never be dropped or stringified.
// This file performs exactly that static cross-check. Missing/invalid artifacts
// remain the protocol-artifact identity diagnostics, not this one.

import (
	"sort"
	"strings"
)

func checkPinnedResponseShapes(catalog *Catalog, snapshot *TrustedSnapshot, input CatalogInput) []Diagnostic {
	if catalog == nil || snapshot == nil || snapshot.Context == nil {
		return nil
	}
	var diagnostics []Diagnostic
	for index, operation := range catalog.Operations {
		if operation.Protocol.OpenAPI.ArtifactRef == "" {
			continue
		}
		view, ok := snapshot.Context.Artifact(operation.Protocol.OpenAPI.ArtifactRef)
		if !ok {
			continue
		}
		document, ok := view.Document.(map[string]any)
		if !ok {
			continue
		}
		mediaTypes, ok := pinnedResponseMediaTypes(document, operation)
		if !ok {
			continue
		}
		openAPIMultipart := false
		for _, mediaType := range mediaTypes {
			if strings.HasPrefix(mediaType, "multipart/") {
				openAPIMultipart = true
			}
		}
		if !openAPIMultipart {
			continue
		}
		if declaredMultipartRepresentation(operation) {
			continue
		}
		diagnostics = append(diagnostics, ingressDiagnostic(
			CodeIngressResponseMappingInvalid,
			renderMessage(ingressMessageTemplates[CodeIngressResponseMappingInvalid],
				"{exportId}", operation.ID, "{fieldPath}", "/response/dispositions"),
			input.DependencyAlias, input.SnapshotPath, operationPointer(index, "/response/dispositions")))
	}
	return diagnostics
}

// pinnedResponseMediaTypes returns the exact response content media types
// declared by the pinned OpenAPI operation for the Catalog operation's path and
// method. The lookup is an exact path-template/method match; no heuristic route
// matching happens here.
func pinnedResponseMediaTypes(document map[string]any, operation Operation) ([]string, bool) {
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return nil, false
	}
	pathItem, ok := paths[operation.Protocol.OpenAPI.Path].(map[string]any)
	if !ok {
		return nil, false
	}
	method := strings.ToLower(operation.Protocol.OpenAPI.Method)
	action, ok := pathItem[method].(map[string]any)
	if !ok {
		return nil, false
	}
	responses, ok := action["responses"].(map[string]any)
	if !ok {
		return nil, false
	}
	set := make(map[string]bool)
	for _, responseValue := range responses {
		response, ok := responseValue.(map[string]any)
		if !ok {
			continue
		}
		content, ok := response["content"].(map[string]any)
		if !ok {
			continue
		}
		for mediaType := range content {
			set[mediaType] = true
		}
	}
	mediaTypes := make([]string, 0, len(set))
	for mediaType := range set {
		mediaTypes = append(mediaTypes, mediaType)
	}
	sort.Strings(mediaTypes)
	return mediaTypes, true
}

func declaredMultipartRepresentation(operation Operation) bool {
	for _, disposition := range operation.Response.Dispositions {
		for _, representation := range disposition.Representations {
			if strings.HasPrefix(representation.MediaType, "multipart/") {
				return true
			}
		}
	}
	return false
}
