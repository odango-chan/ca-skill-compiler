package compiler

// CA-SL v1alpha5 WP6: paired-response boundary, post-dominance and typed
// response-result Producer mapping.
//
// Authoritative clauses: ca-skills #61 Appendix L.8.1/L.8.2/L.8.3, L.11/L.11.1
// and L.11.2 plus the #62 Integration Ingress Operation contract §7/§8 and the
// #62 Ingress diagnostic registry (INGRESS_RESPONSE_CONTRACT_INVALID,
// INGRESS_RESPONSE_MAPPING_INVALID, INGRESS_BINARY_CONTENT_LIFECYCLE_INVALID).
//
// The pass is deliberately exact and fail-closed:
//
//   - an authoring wait.response is invalid when the linked trusted export
//     declares response.mode: none;
//   - for a paired export the response boundary is the Node named by
//     wait.response.after when present, else the Wait-bearing Node; it must be
//     reachable from the event-accepted edge;
//   - every event-accepted path must reach the boundary Node's completion, or
//     terminate earlier in FAILED/UNRESOLVED/SAFE_STOPPED with an explicit
//     trusted earlyFailureResponses mapping; a path that reaches SUCCEEDED before
//     the boundary is invalid (WAIT_RESPONSE_BOUNDARY_INVALID);
//   - the response-result Producer is derived from the export's
//     response.result.producer stage type, never from a Skill-specific NodeId in
//     the reusable export; the boundary Node must declare that stage, its exact
//     ToolCatalog output Schema must be available on every normal path and must be
//     assignable to responseResultSchema (INPUT_TYPE_MISMATCH / SOURCE_NOT_AVAILABLE
//     reuse, not WAIT_RESPONSE_MAPPING_INVALID);
//   - Ingress-owned response contract/representation/mapping/binary-content
//     defects are reported with the Ingress-owned INGRESS_RESPONSE_* identities
//     and suppress the dependent Skill-side response diagnostics.
//
// Downstream scope is deliberately not faked: the response deadline policy and
// durable delivery-recovery contract are projected and their static presence is
// enforced by IngressCatalog admission, but their resolution against trusted
// deployment policy artifacts belongs to Host Admission/Integration (§8.4/§8.5),
// and actual wire serialization/validation belongs to the Integration adapter.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"github.com/odango-chan/ca-skill-compiler/internal/ingresscatalog"
)

// v1alpha5ResponseTerminals is the deterministic early-failure terminal order.
var v1alpha5ResponseTerminals = []string{"FAILED", "UNRESOLVED", "SAFE_STOPPED"}

// v1alpha5ResponseContract is the validated Ingress response contract of one
// trusted export. It carries only facts proven by the Ingress-side checks; the
// Skill-side boundary/Producer rules consume it.
type v1alpha5ResponseContract struct {
	Paired           bool
	RequiresProducer bool
	RequiresMappings bool
	Producer         string
	ResultRoot       ingresscatalog.RootRef
	HasResultRoot    bool
	EarlyFailures    map[string]bool
}

// v1alpha5ResponseCache caches the once-per-export Ingress response contract
// validation so multiple Wait-bearing Nodes linking the same export do not repeat
// (or duplicate) Ingress-owned diagnostics.
type v1alpha5ResponseCache struct {
	contracts map[string]*v1alpha5ResponseContract
}

func newV1Alpha5ResponseCache() *v1alpha5ResponseCache {
	return &v1alpha5ResponseCache{
		contracts: map[string]*v1alpha5ResponseContract{},
	}
}

func (cache *v1alpha5ResponseCache) contractFor(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext) (*v1alpha5ResponseContract, []Diagnostic) {
	key := v1alpha5ExportContractKey(export)
	if contract, cached := cache.contracts[key]; cached {
		return contract, nil
	}
	contract, diagnostics := buildV1Alpha5ResponseContract(export, link)
	cache.contracts[key] = contract
	return contract, diagnostics
}

// v1alpha5IngressResponseDiagnostic builds one Ingress-owned response diagnostic
// with its registry message template. The source location is the exact trusted
// dependency snapshot path; the canonical field path is rendered into the
// normative message.
func v1alpha5IngressResponseDiagnostic(export v1alpha5TrustedIngressExport, code, fieldPath string) Diagnostic {
	message, _ := ingresscatalog.RenderIngressMessage(code,
		"{exportId}", export.Operation.ID, "{fieldPath}", fieldPath)
	return Diagnostic{
		Severity: "error",
		Phase:    "C_STATIC",
		Code:     code,
		Path:     limitDiagnosticPath(export.SnapshotPath),
		Message:  limitDiagnosticMessage(message),
	}
}

func v1alpha5ResponseBoundaryMessage(nodeID string) string {
	return fmt.Sprintf("Node '%s' wait.response.after does not identify one reachable response boundary on every accepted-event path.", nodeID)
}

func v1alpha5ResponseMappingMessage(nodeID string) string {
	return fmt.Sprintf("Node '%s' response boundary has no valid typed Producer-to-response mapping.", nodeID)
}

// v1alpha5ResponseAnchor returns the diagnostic anchor for a response-boundary
// failure: the exact response.after value when authored, else the Wait object.
func v1alpha5ResponseAnchor(wait *V1Alpha5WaitAST) *yaml.Node {
	if wait == nil {
		return nil
	}
	if wait.Response != nil && wait.Response.Raw != nil {
		if after := child(wait.Response.Raw, "after"); after != nil {
			return after
		}
		return wait.Response.Raw
	}
	return wait.Raw
}

// v1alpha5ValidateWaitResponse is the WP6 pass for one Wait-bearing Node. It runs
// after the WP3 export link/correlation contract succeeded, so the trusted export
// and its provenance are already proven.
func v1alpha5ValidateWaitResponse(procedurePath, nodeID string, wait *V1Alpha5WaitAST, graph *V1Alpha5ProcedureGraph, skill *V1Alpha5SkillAST, contract *v1alpha5ExportContract, link *v1alpha5LinkContext, scope []v1alpha5Dependency, triggerIndexes map[string]*v1alpha5SnapshotTriggerIndex, cache *v1alpha5ResponseCache) []Diagnostic {
	if wait == nil || contract == nil || contract.Export.Operation.ID == "" {
		return nil
	}
	var diagnostics []Diagnostic
	export := contract.Export
	response := export.Operation.Response
	authorDeclared := wait.Response != nil
	boundaryInvalid := func() Diagnostic {
		return v1alpha5StaticDiagnostic(procedurePath, v1alpha5ResponseAnchor(wait),
			"WAIT_RESPONSE_BOUNDARY_INVALID", v1alpha5ResponseBoundaryMessage(nodeID))
	}
	mappingInvalid := func() Diagnostic {
		return v1alpha5StaticDiagnostic(procedurePath, v1alpha5ResponseAnchor(wait),
			"WAIT_RESPONSE_MAPPING_INVALID", v1alpha5ResponseMappingMessage(nodeID))
	}

	// L.8.1: an authoring wait.response is invalid when the export declares no
	// paired protocol response.
	if response.Mode != "paired" {
		if authorDeclared {
			return []Diagnostic{boundaryInvalid()}
		}
		return nil
	}

	responseContract, contractDiagnostics := cache.contractFor(export, link)
	diagnostics = append(diagnostics, contractDiagnostics...)
	if responseContract == nil {
		// Ingress response contract is a prerequisite failure; dependent
		// Skill-side response diagnostics are suppressed (#62 §13).
		return diagnostics
	}

	// Boundary selection (L.8.1 / #62 §8.2): the Node named by
	// wait.response.after when present, else the Wait-bearing Node's own
	// completion.
	boundaryNodeID := nodeID
	if wait.Response != nil && wait.Response.After != "" {
		boundaryNodeID = wait.Response.After
	}
	if graph.Vertex(boundaryNodeID) == nil {
		// An unknown response.after Node is already diagnosed by the WP5 graph
		// pass with the same identity; do not duplicate it.
		return diagnostics
	}
	anchor := nodeID
	reachable := graph.ReachableFrom(anchor)
	if !reachable[boundaryNodeID] {
		diagnostics = append(diagnostics, boundaryInvalid())
		return diagnostics
	}

	// Producer binding: the boundary Node must declare the exact stage type the
	// export names, and the response SourcePath is derived as
	// "<boundary-node>.<producer>.output".
	if responseContract.RequiresProducer {
		producerStage := responseContract.Producer
		if producerStage != "operation" && producerStage != "observation" {
			// IngressCatalog admission already rejects an invalid producer enum;
			// treat a projected empty/invalid value as an Ingress contract defect.
			diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
				ingresscatalog.CodeIngressResponseContractInvalid, "/response/result/producer"))
			return diagnostics
		}
		if v1alpha5StageNode(skill, boundaryNodeID, producerStage) == nil {
			if boundaryNodeID == nodeID {
				// The export requires a response boundary later than the
				// Wait-bearing Node, so omitting wait.response.after is invalid.
				diagnostics = append(diagnostics, boundaryInvalid())
			} else {
				// IN-030: the selected boundary Node does not declare the exact
				// export-declared Producer stage, so it is not a valid response
				// boundary. EW-021 additionally records that no typed
				// Producer-to-response mapping can be satisfied at that Node.
				diagnostics = append(diagnostics, boundaryInvalid(), mappingInvalid())
			}
			return diagnostics
		}
		diagnostics = append(diagnostics, v1alpha5ValidateResponseProducer(procedurePath, nodeID,
			boundaryNodeID, producerStage, skill, link, scope, triggerIndexes, responseContract)...)
	}

	// Post-dominance and Producer availability proof. The boundary is committed
	// only when the path completes the boundary Node's last declared business
	// stage on its normal Outcome; any earlier exit terminates before the
	// boundary.
	postDominates := graph.PostDominates(boundaryNodeID, anchor)
	bypass, incomplete := v1alpha5ResponsePathStates(graph, anchor, boundaryNodeID)
	if bypass["SUCCEEDED"] {
		diagnostics = append(diagnostics, boundaryInvalid())
	}
	if !postDominates && len(bypass) == 0 && !incomplete {
		// Defensive: a boundary that does not post-dominate the event-accepted
		// edge, while no bypass terminal can be exhibited, cannot be proven.
		diagnostics = append(diagnostics, boundaryInvalid())
	}
	var unmapped []string
	for _, terminal := range v1alpha5ResponseTerminals {
		if !bypass[terminal] {
			continue
		}
		if responseContract.EarlyFailures[terminal] {
			continue
		}
		unmapped = append(unmapped, terminal)
	}
	if len(unmapped) > 0 {
		// The early-failure response policy is Ingress-owned (#62 §8.4); a
		// permitted early terminal path without an explicit trusted mapping is an
		// invalid paired-response contract, not a Skill-side mapping defect.
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressResponseContractInvalid, "/response/earlyFailureResponses"))
	}
	return diagnostics
}

// v1alpha5StageNode returns the raw business stage mapping of a declared Node.
func v1alpha5StageNode(skill *V1Alpha5SkillAST, nodeID, stage string) *yaml.Node {
	if skill == nil || skill.Procedure == nil {
		return nil
	}
	node := skill.Procedure.ByID[nodeID]
	if node == nil {
		return nil
	}
	return v1alpha5ProcedureStage(node, stage)
}

// v1alpha5ResponsePathStates returns the set of terminals reachable from the
// event-accepted edge before the boundary Node's completion on its final normal
// Outcome. It also reports whether a path can leave the boundary Node without
// ever reaching a terminal (incomplete procedure).
//
// The traversal state is (vertex, completed): completed becomes true only when
// the boundary Node's last declared business stage routes its normal Outcome
// (Appendix L.1 stage auto-advance semantics: earlier stages auto-advance on the
// normal Outcome, so leaving through the last stage proves every declared stage,
// including the response Producer stage, executed).
func v1alpha5ResponsePathStates(graph *V1Alpha5ProcedureGraph, anchor, boundaryNodeID string) (map[string]bool, bool) {
	bypass := map[string]bool{}
	if graph == nil {
		return bypass, false
	}
	boundary := graph.Vertex(boundaryNodeID)
	if boundary == nil || boundary.Node == nil {
		return bypass, false
	}
	lastStage := ""
	for _, stage := range v1alpha5WaitBusinessStages {
		if v1alpha5ProcedureStage(boundary.Node, stage) != nil {
			lastStage = stage
		}
	}
	type state struct {
		vertex    string
		completed bool
	}
	seen := map[state]bool{}
	queue := []state{{vertex: anchor}}
	incomplete := false
	terminals := map[string]bool{"FAILED": true, "UNRESOLVED": true, "SAFE_STOPPED": true, "SUCCEEDED": true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		if terminals[current.vertex] {
			if !current.completed {
				bypass[current.vertex] = true
			}
			continue
		}
		edges := graph.EdgesFrom(current.vertex)
		if current.vertex == boundaryNodeID && !current.completed {
			if len(edges) == 0 {
				incomplete = true
				continue
			}
			for _, edge := range edges {
				completed := lastStage != "" && edge.Kind == V1Alpha5EdgeStageRoute &&
					edge.Stage == lastStage && edge.Outcome == v4StageNormalOutcome[lastStage]
				queue = append(queue, state{vertex: edge.To, completed: completed})
			}
			continue
		}
		if len(edges) == 0 {
			if !current.completed {
				incomplete = true
			}
			continue
		}
		for _, edge := range edges {
			queue = append(queue, state{vertex: edge.To, completed: current.completed})
		}
	}
	return bypass, incomplete
}

// v1alpha5ValidateResponseProducer proves the exact typed Producer binding:
// response SourcePath "<boundary-node>.<producer>.output" resolves through the
// trusted ToolCatalog export to a trusted Schema binding, and its Output Schema is
// assignable to the export's responseResultSchema. Unavailable/unresolved paths
// reuse SOURCE_NOT_AVAILABLE / SCHEMA_*; ordinary Schema incompatibility reuses
// INPUT_TYPE_MISMATCH (Appendix L.11.2).
func v1alpha5ValidateResponseProducer(procedurePath, waitNodeID, boundaryNodeID, producerStage string, skill *V1Alpha5SkillAST, link *v1alpha5LinkContext, scope []v1alpha5Dependency, triggerIndexes map[string]*v1alpha5SnapshotTriggerIndex, responseContract *v1alpha5ResponseContract) []Diagnostic {
	stageNode := v1alpha5StageNode(skill, boundaryNodeID, producerStage)
	if stageNode == nil {
		return []Diagnostic{v1alpha5StaticDiagnostic(procedurePath, nil, "WAIT_RESPONSE_MAPPING_INVALID",
			v1alpha5ResponseMappingMessage(waitNodeID))}
	}
	toolNode := child(stageNode, "tool")
	unavailable := func(message string) []Diagnostic {
		return []Diagnostic{v1alpha5StaticDiagnostic(procedurePath, v1alpha5FallbackNode(toolNode, stageNode),
			"SOURCE_NOT_AVAILABLE", message)}
	}
	alias, exportID, ok := splitQualifiedReference(toolNode)
	if !ok {
		return unavailable("Response boundary Producer '" + boundaryNodeID + "." + producerStage + ".output' has no exact qualified Tool reference.")
	}
	var dependency v1alpha5Dependency
	found := false
	for _, candidate := range scope {
		if candidate.Alias == alias {
			dependency, found = candidate, true
			break
		}
	}
	if !found {
		return unavailable("Response boundary Producer '" + boundaryNodeID + "." + producerStage + ".output' does not resolve in a declared trusted dependency snapshot.")
	}
	cached, exists := triggerIndexes[dependency.Alias]
	if !exists {
		index, _, indexDiagnostics := buildSnapshotExportIndex(dependency.Snapshot)
		cached = &v1alpha5SnapshotTriggerIndex{index: index, diagnostics: indexDiagnostics}
		triggerIndexes[dependency.Alias] = cached
	}
	records := cached.index.callables[exportID]
	if len(records) != 1 || records[0].ExportKind != producerStage {
		return unavailable("Response boundary Producer '" + boundaryNodeID + "." + producerStage + ".output' does not resolve to exactly one trusted ToolCatalog " + producerStage + " export.")
	}
	record := records[0]
	owner, ownershipDiagnostics := attributeExportOwner(record, dependency.Binding, nil)
	if len(ownershipDiagnostics) > 0 || owner == "" {
		return unavailable("Response boundary Producer '" + boundaryNodeID + "." + producerStage + ".output' does not resolve to a trusted Owner-attributed ToolCatalog export.")
	}
	if record.OutputSchemaID == "" {
		return unavailable("Response boundary Producer '" + boundaryNodeID + "." + producerStage + ".output' declares no Output Schema.")
	}
	outputRoot, bound := link.schemaRoots[record.OutputSchemaID]
	if !bound {
		return unavailable("Response boundary Producer Output Schema '" + record.OutputSchemaID + "' is not declared by the trusted schema bindings.")
	}
	if _, err := link.resolution.ComputeSchemaClosure(outputRoot); err != nil {
		return []Diagnostic{v1alpha5SchemaDiagnostic(record.SourcePath, "SCHEMA_INVALID",
			"Response boundary Producer Output Schema '"+record.OutputSchemaID+"' is invalid for Schema Closure Digest v1: "+err.Error())}
	}
	outputNode, err := v1alpha5SchemaRootNode(link.resolution, outputRoot)
	if err != nil {
		return []Diagnostic{v1alpha5SchemaDiagnostic(record.SourcePath, "SCHEMA_INVALID",
			"Response boundary Producer Output Schema '"+record.OutputSchemaID+"' cannot be materialized: "+err.Error())}
	}
	resultNode, err := v1alpha5SchemaRootNode(link.resolution, responseContract.ResultRoot)
	if err != nil {
		return []Diagnostic{v1alpha5SchemaDiagnostic(link.snapshotPathForRoot(responseContract.ResultRoot), "SCHEMA_INVALID",
			"responseResultSchema cannot be materialized: "+err.Error())}
	}
	reason, ok := schemaAssignable(outputNode, resultNode)
	if !ok || reason != assignableOK {
		return []Diagnostic{v1alpha5StaticDiagnostic(procedurePath, v1alpha5FallbackNode(toolNode, stageNode),
			"INPUT_TYPE_MISMATCH", "Response boundary Producer Output Schema is not assignable to the export responseResultSchema (reason "+string(reason)+").")}
	}
	return nil
}

// v1alpha5SchemaRootNode materializes one trusted Schema root through the single
// normalized YAML AST used by the rest of the compiler. JSON is the deterministic
// wire form of the already-validated trusted Schema value; no second Schema
// implementation is introduced.
func v1alpha5SchemaRootNode(resolution *ingresscatalog.ResolutionContext, root ingresscatalog.RootRef) (*yaml.Node, error) {
	resolved, err := resolution.ResolveRoot(root)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(resolved.Schema)
	if err != nil {
		return nil, err
	}
	return yaml.Parse(raw)
}

// v1alpha5SchemaRootPath returns the trusted snapshot path of a Schema artifact
// root for diagnostics. It is empty when the artifact is not projected.
func (link *v1alpha5LinkContext) snapshotPathForRoot(root ingresscatalog.RootRef) string {
	if link == nil {
		return ""
	}
	for _, context := range link.artifactPaths {
		if context.artifactRef == root.ArtifactRef {
			return context.snapshotPath
		}
	}
	return ""
}

// v1alpha5SchemaArtifactPath is one artifact-ref to dependency-snapshot-path
// mapping used only for Ingress diagnostic source locations.
type v1alpha5SchemaArtifactPath struct {
	artifactRef  string
	snapshotPath string
}

// buildV1Alpha5ResponseContract validates the Ingress-owned paired-response
// contract and representation/mapping completeness (#62 §8.3/§8.4). Any failure
// returns a nil contract so dependent Skill-side diagnostics are suppressed.
func buildV1Alpha5ResponseContract(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext) (*v1alpha5ResponseContract, []Diagnostic) {
	var diagnostics []Diagnostic
	response := export.Operation.Response
	contract := &v1alpha5ResponseContract{
		Paired:        response.Mode == "paired",
		EarlyFailures: map[string]bool{},
	}
	for outcome, responseRef := range response.EarlyFailureResponses {
		if responseRef != "" {
			contract.EarlyFailures[outcome] = true
		}
	}
	if !contract.Paired {
		return contract, nil
	}
	invalidContract := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressResponseContractInvalid, fieldPath))
	}
	invalidMapping := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressResponseMappingInvalid, fieldPath))
	}
	invalidBinary := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressBinaryContentLifecycleInvalid, fieldPath))
	}

	if len(response.Dispositions) == 0 {
		invalidContract("/response/dispositions")
		return nil, diagnostics
	}
	bodyBearing := false
	seenStatus := map[int64]bool{}
	seenDisposition := map[string]bool{}
	for index, disposition := range response.Dispositions {
		base := "/response/dispositions/" + itoa(index)
		if disposition.ID == "" || seenDisposition[disposition.ID] {
			invalidContract(base + "/id")
		}
		seenDisposition[disposition.ID] = true
		if seenStatus[disposition.Status] {
			// #62 §8.3: a status may have only one disposition; additional media
			// types are representations of the same disposition.
			invalidContract(base + "/status")
		}
		seenStatus[disposition.Status] = true
		switch disposition.Body {
		case "required":
			bodyBearing = true
			if len(disposition.Representations) == 0 {
				invalidContract(base + "/representations")
			}
		case "none":
			if len(disposition.Representations) != 0 {
				invalidContract(base + "/representations")
			}
		default:
			invalidContract(base + "/body")
		}
	}
	contract.RequiresProducer = bodyBearing || len(response.Dispositions) > 1
	contract.RequiresMappings = bodyBearing
	contract.Producer = response.Result.Producer
	if contract.RequiresProducer && (response.Result.Producer == "" || response.Result.ResponseResultSchemaRef == "") {
		invalidContract("/response/result")
		return nil, diagnostics
	}
	if response.Result.ResponseResultSchemaRef != "" {
		root, bound := link.schemaRoots[response.Result.ResponseResultSchemaRef]
		if !bound {
			diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
				"responseResultSchema '"+response.Result.ResponseResultSchemaRef+"' is not declared by the trusted v1alpha5 schema bindings."))
			return nil, diagnostics
		}
		if _, err := link.resolution.ComputeSchemaClosure(root); err != nil {
			diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_INVALID",
				"responseResultSchema '"+response.Result.ResponseResultSchemaRef+"' is invalid for Schema Closure Digest v1: "+err.Error()))
			return nil, diagnostics
		}
		contract.ResultRoot = root
		contract.HasResultRoot = true
	}

	// Representative Schema roots for target/coverage checks. IngressCatalog
	// admission already verified the declared Schema Closure digests, so a missing
	// binding here is a prerequisite Schema failure.
	representationRoot := func(representation ingresscatalog.ResponseRepresentationProjection) (ingresscatalog.RootRef, bool, bool) {
		root, bound := link.schemaRoots[representation.ResponseSchemaRef]
		if !bound {
			diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
				"Response representation Schema '"+representation.ResponseSchemaRef+"' is not declared by the trusted v1alpha5 schema bindings."))
			return ingresscatalog.RootRef{}, true, false
		}
		if _, err := link.resolution.ComputeSchemaClosure(root); err != nil {
			diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_INVALID",
				"Response representation Schema '"+representation.ResponseSchemaRef+"' is invalid for Schema Closure Digest v1: "+err.Error()))
			return ingresscatalog.RootRef{}, true, false
		}
		return root, true, true
	}

	// Binary-content lifecycle declaration (#62 §8.3/§8.5). Only declared when a
	// multipart/related representation exists.
	multipart := false
	for _, disposition := range response.Dispositions {
		for _, representation := range disposition.Representations {
			if representation.MediaType == "multipart/related" {
				multipart = true
			}
		}
	}
	if multipart {
		if response.BinaryContent == nil {
			invalidBinary("/response/binaryContent")
		} else {
			binary := response.BinaryContent
			if binary.ReferenceSchemaRef == "" {
				invalidBinary("/response/binaryContent/referenceSchemaRef")
			} else if _, bound := link.schemaRoots[binary.ReferenceSchemaRef]; !bound {
				diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
					"response.binaryContent.referenceSchemaRef '"+binary.ReferenceSchemaRef+"' is not declared by the trusted v1alpha5 schema bindings."))
			}
			if binary.DigestAlgorithm != "sha-256" {
				invalidBinary("/response/binaryContent/digestAlgorithm")
			}
			if !binary.Immutable {
				invalidBinary("/response/binaryContent/immutable")
			}
			if binary.Access != "integration-response-only" {
				invalidBinary("/response/binaryContent/access")
			}
			if binary.Retention == "" {
				invalidBinary("/response/binaryContent/retention")
			}
			if binary.MaxBytes <= 0 {
				invalidBinary("/response/binaryContent/maxBytes")
			}
		}
	}

	// Representation structural/semantic rules that the structural schema cannot
	// express across dispositions and parts.
	for dispositionIndex, disposition := range response.Dispositions {
		for representationIndex, representation := range disposition.Representations {
			base := "/response/dispositions/" + itoa(dispositionIndex) + "/representations/" + itoa(representationIndex)
			root, schemaChecked, ok := representationRoot(representation)
			if schemaChecked && !ok {
				continue
			}
			switch representation.MediaType {
			case "application/json":
				if len(representation.Parts) != 0 || representation.RootPart != "" {
					invalidMapping(base + "/parts")
				}
				_ = root
			case "multipart/related":
				if representation.RootPart == "" {
					invalidMapping(base + "/rootPart")
					continue
				}
				jsonParts := 0
				binaryParts := 0
				rootPartFound := false
				seenPartName := map[string]bool{}
				for partIndex, part := range representation.Parts {
					partBase := base + "/parts/" + itoa(partIndex)
					if part.Name == "" || seenPartName[part.Name] {
						invalidMapping(partBase + "/name")
					}
					seenPartName[part.Name] = true
					if part.MaxBytes <= 0 || part.MaxBytes > representation.MaxBytes {
						invalidMapping(partBase + "/maxBytes")
					}
					switch part.Kind {
					case "json":
						jsonParts++
						if part.SchemaRef == "" {
							invalidMapping(partBase + "/schemaRef")
							continue
						}
						if _, bound := link.schemaRoots[part.SchemaRef]; !bound {
							diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
								"multipart JSON part Schema '"+part.SchemaRef+"' is not declared by the trusted v1alpha5 schema bindings."))
						}
						if part.Name == representation.RootPart {
							rootPartFound = true
						}
					case "binary":
						binaryParts++
						if response.BinaryContent != nil && part.MaxBytes > response.BinaryContent.MaxBytes {
							invalidBinary(partBase + "/maxBytes")
						}
					default:
						invalidMapping(partBase + "/kind")
					}
				}
				if jsonParts != 1 || !rootPartFound {
					invalidMapping(base + "/rootPart")
				}
				// #62 §8.3: the root JSON part Schema must be consistent with the
				// representation responseSchemaRef/digest.
				if rootPart, exists := v1alpha5FindPart(representation, representation.RootPart); exists {
					if rootPart.SchemaRef != representation.ResponseSchemaRef ||
						rootPart.SchemaDigest != representation.ResponseSchemaDigest {
						invalidMapping(base + "/rootPart")
					}
				}
				if representation.MinBinaryParts < 1 || representation.MaxBinaryParts < representation.MinBinaryParts ||
					representation.MinBinaryParts > int64(binaryParts) || representation.MaxBinaryParts > int64(binaryParts) {
					invalidMapping(base + "/minBinaryParts")
				}
			default:
				invalidMapping(base + "/mediaType")
			}
		}
	}

	// dispositionSelector coverage (#62 §8.3): required when multiple
	// dispositions or any body-bearing disposition has multiple representations.
	selectorRequired := len(response.Dispositions) > 1
	for _, disposition := range response.Dispositions {
		if len(disposition.Representations) > 1 {
			selectorRequired = true
		}
	}
	if selectorRequired {
		diagnostics = append(diagnostics, v1alpha5ValidateResponseSelector(export, link, response, contract)...)
	}

	// Typed result-to-response mappings.
	diagnostics = append(diagnostics, v1alpha5ValidateResponseMappings(export, link, response, contract)...)

	sortDiagnostics(diagnostics)
	if hasErrorDiagnostic(diagnostics) {
		return nil, diagnostics
	}
	return contract, diagnostics
}

// v1alpha5ValidateResponseSelector proves the closed-enum dispositionSelector
// coverage: the source property is a closed enum in responseResultSchema, every
// case value is one enum value, and every allowed disposition/representation
// combination is covered exactly once.
func v1alpha5ValidateResponseSelector(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext, response ingresscatalog.ResponseProjection, contract *v1alpha5ResponseContract) []Diagnostic {
	var diagnostics []Diagnostic
	invalid := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressResponseContractInvalid, fieldPath))
	}
	if !contract.HasResultRoot {
		invalid("/response/result")
		return diagnostics
	}
	selectorPath := response.Result.DispositionSelectorPath
	pointer, ok := v1alpha5DataPointer(selectorPath)
	if !ok {
		invalid("/response/result/dispositionSelector/sourcePath")
		return diagnostics
	}
	result, err := link.resolution.LookupSchemaPath(contract.ResultRoot, pointer)
	if err != nil {
		invalid("/response/result/dispositionSelector/sourcePath")
		return diagnostics
	}
	selectorSchema, ok := result.Schema.(map[string]any)
	if !ok {
		invalid("/response/result/dispositionSelector/sourcePath")
		return diagnostics
	}
	enumValues, ok := selectorSchema["enum"].([]any)
	if !ok || len(enumValues) < 2 {
		invalid("/response/result/dispositionSelector/sourcePath")
		return diagnostics
	}
	enumSet := map[string]bool{}
	for _, value := range enumValues {
		if !v1alpha5JSONScalar(value) {
			invalid("/response/result/dispositionSelector/sourcePath")
			return diagnostics
		}
		enumSet[v1alpha5JSONValueKey(value)] = true
	}
	if len(enumSet) != len(enumValues) {
		invalid("/response/result/dispositionSelector/sourcePath")
		return diagnostics
	}

	// Allowed disposition/representation combinations.
	allowed := map[string]bool{}
	for _, disposition := range response.Dispositions {
		if disposition.Body == "none" {
			allowed[disposition.ID+"\x00"] = true
			continue
		}
		for _, representation := range disposition.Representations {
			allowed[disposition.ID+"\x00"+representation.ID] = true
		}
	}
	covered := map[string]bool{}
	seenValue := map[string]bool{}
	for index, selectorCase := range response.Result.DispositionSelectorCases {
		base := "/response/result/dispositionSelector/cases/" + itoa(index)
		if !v1alpha5JSONScalar(selectorCase.Value) {
			invalid(base + "/value")
			continue
		}
		key := v1alpha5JSONValueKey(selectorCase.Value)
		if !enumSet[key] || seenValue[key] {
			invalid(base + "/value")
		}
		seenValue[key] = true
		dispositionID := selectorCase.DispositionID
		ip, exists := v1alpha5FindDisposition(response.Dispositions, dispositionID)
		if !exists {
			invalid(base + "/dispositionId")
			continue
		}
		if ip.Body == "none" {
			if selectorCase.RepresentationID != "" {
				invalid(base + "/representationId")
			}
			combo := dispositionID + "\x00"
			if covered[combo] {
				invalid(base)
			}
			covered[combo] = true
			continue
		}
		if selectorCase.RepresentationID == "" {
			invalid(base + "/representationId")
			continue
		}
		if _, exists := v1alpha5FindRepresentation(*ip, selectorCase.RepresentationID); !exists {
			invalid(base + "/representationId")
			continue
		}
		combo := dispositionID + "\x00" + selectorCase.RepresentationID
		if covered[combo] {
			invalid(base)
		}
		covered[combo] = true
	}
	for combo := range allowed {
		if !covered[combo] {
			invalid("/response/result/dispositionSelector/cases")
			break
		}
	}
	if len(covered) != len(allowed) {
		invalid("/response/result/dispositionSelector/cases")
	}
	return diagnostics
}

// v1alpha5ValidateResponseMappings validates every declared typed mapping: the
// from Pointer exists in responseResultSchema, the target Pointer exists in the
// exact protocol response Schema, multipart part rules hold, and every body-bearing
// representation supplies every required protocol response field.
func v1alpha5ValidateResponseMappings(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext, response ingresscatalog.ResponseProjection, contract *v1alpha5ResponseContract) []Diagnostic {
	var diagnostics []Diagnostic
	invalid := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressResponseMappingInvalid, fieldPath))
	}
	if !contract.RequiresMappings {
		return diagnostics
	}
	mappings := response.Result.Mappings
	if len(mappings) == 0 {
		invalid("/response/result/mappings")
		return diagnostics
	}
	if !contract.HasResultRoot {
		return diagnostics
	}
	seenTarget := map[string]bool{}
	for index, mapping := range mappings {
		base := "/response/result/mappings/" + itoa(index)
		disposition, exists := v1alpha5FindDisposition(response.Dispositions, mapping.DispositionID)
		if !exists {
			invalid(base + "/dispositionId")
			continue
		}
		representation, exists := v1alpha5FindRepresentation(*disposition, mapping.RepresentationID)
		if !exists {
			invalid(base + "/representationId")
			continue
		}
		if disposition.Body != "required" {
			invalid(base + "/dispositionId")
			continue
		}
		fromPointer, ok := v1alpha5DataPointer(mapping.From)
		if !ok {
			invalid(base + "/from")
			continue
		}
		from, err := link.resolution.LookupSchemaPath(contract.ResultRoot, fromPointer)
		if err != nil {
			invalid(base + "/from")
			continue
		}
		targetRoot, targetPointer, partOK := v1alpha5MappingTargetSchema(link, *representation, mapping)
		if !partOK {
			invalid(base + "/target")
			continue
		}
		if _, err := link.resolution.LookupSchemaPath(targetRoot, targetPointer); err != nil {
			invalid(base + "/target")
			continue
		}
		// Conservative value-type compatibility between the typed result field and
		// the protocol response field when both declare a JSON type; full wire
		// validation remains the Integration adapter's responsibility. A
		// multipart-binary mapping reads a typed binary content reference and
		// writes the Integration-generated Content-ID string, so no direct JSON
		// type comparison applies to its contentIdField.
		if mapping.Target.Kind != "multipart-binary" {
			if targetResult, err := link.resolution.LookupSchemaPath(targetRoot, targetPointer); err == nil {
				if !v1alpha5JSONTypesCompatible(from.Schema, targetResult.Schema) {
					invalid(base + "/from")
					continue
				}
			}
		}
		targetKey := mapping.DispositionID + "\x00" + mapping.RepresentationID + "\x00" +
			mapping.Target.Kind + "\x00" + mapping.Target.Part + "\x00" + mapping.Target.Pointer
		if seenTarget[targetKey] {
			invalid(base + "/target")
			continue
		}
		seenTarget[targetKey] = true
	}

	// Per-representation coverage.
	for dispositionIndex, disposition := range response.Dispositions {
		if disposition.Body != "required" {
			continue
		}
		for representationIndex, representation := range disposition.Representations {
			base := "/response/dispositions/" + itoa(dispositionIndex) + "/representations/" + itoa(representationIndex)
			diagnostics = append(diagnostics, v1alpha5ValidateResponseRepresentationCoverage(export, link,
				disposition, representation, base, mappings)...)
		}
	}
	return diagnostics
}

// v1alpha5ValidateResponseRepresentationCoverage proves that every required
// protocol response field of one body-bearing representation is supplied by the
// declared mappings. For multipart/related it also proves the root JSON part and
// binary part mapping rules (#62 §8.3).
func v1alpha5ValidateResponseRepresentationCoverage(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext, disposition ingresscatalog.ResponseDispositionProjection, representation ingresscatalog.ResponseRepresentationProjection, base string, mappings []ingresscatalog.ResponseMappingProjection) []Diagnostic {
	var diagnostics []Diagnostic
	missing := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressResponseDiagnostic(export,
			ingresscatalog.CodeIngressResponseMappingInvalid, fieldPath))
	}
	var targetRoot ingresscatalog.RootRef
	expectedKind := ""
	expectedPart := ""
	switch representation.MediaType {
	case "application/json":
		expectedKind = "json-body"
		if root, bound := link.schemaRoots[representation.ResponseSchemaRef]; bound {
			targetRoot = root
		} else {
			return diagnostics
		}
	case "multipart/related":
		expectedKind = "multipart-json"
		expectedPart = representation.RootPart
		if part, exists := v1alpha5FindPart(representation, representation.RootPart); exists && part.Kind == "json" {
			if root, bound := link.schemaRoots[part.SchemaRef]; bound {
				targetRoot = root
			} else {
				return diagnostics
			}
		} else {
			missing(base + "/rootPart")
			return diagnostics
		}
	default:
		return diagnostics
	}
	required, err := v1alpha5RequiredLeafPaths(link, targetRoot)
	if err != nil {
		missing(base + "/responseSchemaRef")
		return diagnostics
	}
	// The multipart Content-ID property in the JSON root part is generated by
	// Integration at serialization time (#62 §8.3) and is therefore not supplied
	// by a response.result mapping; exclude exactly those declared contentIdField
	// pointers from the required-field coverage obligation.
	integrationGenerated := map[string]bool{}
	if representation.MediaType == "multipart/related" {
		for _, mapping := range mappings {
			if mapping.DispositionID != disposition.ID || mapping.RepresentationID != representation.ID {
				continue
			}
			if mapping.Target.Kind == "multipart-binary" {
				integrationGenerated[mapping.Target.ContentIDPointer] = true
			}
		}
	}
	covered := make(map[string]bool)
	for _, mapping := range mappings {
		if mapping.DispositionID != disposition.ID || mapping.RepresentationID != representation.ID {
			continue
		}
		if mapping.Target.Kind != expectedKind {
			continue
		}
		if expectedPart != "" && mapping.Target.Part != expectedPart {
			continue
		}
		covered[mapping.Target.Pointer] = true
	}
	for _, fieldPath := range required {
		if integrationGenerated[fieldPath] {
			continue
		}
		supplied := false
		for pointer := range covered {
			if v1alpha5JSONPointerCovers(pointer, fieldPath) {
				supplied = true
				break
			}
		}
		if !supplied {
			missing(base + "/responseSchemaRef")
			break
		}
	}
	if representation.MediaType != "multipart/related" {
		return diagnostics
	}
	// Binary part mapping rules: exactly one mapping per declared binary part, no
	// mapping for an undeclared part, unique Content-ID targets inside the JSON
	// root part.
	binaryParts := map[string]bool{}
	for _, part := range representation.Parts {
		if part.Kind == "binary" {
			binaryParts[part.Name] = true
		}
	}
	mappedBinaryParts := map[string]int{}
	contentIDFields := map[string]bool{}
	for _, mapping := range mappings {
		if mapping.DispositionID != disposition.ID || mapping.RepresentationID != representation.ID {
			continue
		}
		if mapping.Target.Kind != "multipart-binary" {
			continue
		}
		if !binaryParts[mapping.Target.Part] {
			missing(base + "/parts")
			continue
		}
		mappedBinaryParts[mapping.Target.Part]++
		contentIDField := mapping.Target.ContentIDPart + "\x00" + mapping.Target.ContentIDPointer
		if mapping.Target.ContentIDPart != representation.RootPart || !v1alpha5ValidContentIDField(link, representation, mapping) {
			missing("/response/result/mappings")
			continue
		}
		if contentIDFields[contentIDField] {
			missing("/response/result/mappings")
			continue
		}
		contentIDFields[contentIDField] = true
	}
	for part := range binaryParts {
		if mappedBinaryParts[part] != 1 {
			missing(base + "/parts")
			break
		}
	}
	return diagnostics
}

// v1alpha5ValidContentIDField proves that the declared contentIdField pointer
// resolves to a string property of the multipart JSON root part Schema
// (#62 §8.3: RefToBinaryData.contentId).
func v1alpha5ValidContentIDField(link *v1alpha5LinkContext, representation ingresscatalog.ResponseRepresentationProjection, mapping ingresscatalog.ResponseMappingProjection) bool {
	part, exists := v1alpha5FindPart(representation, representation.RootPart)
	if !exists {
		return false
	}
	root, bound := link.schemaRoots[part.SchemaRef]
	if !bound {
		return false
	}
	pointer, ok := v1alpha5DataPointer(mapping.Target.ContentIDPointer)
	if !ok {
		return false
	}
	result, err := link.resolution.LookupSchemaPath(root, pointer)
	if err != nil {
		return false
	}
	schema, ok := result.Schema.(map[string]any)
	if !ok {
		return false
	}
	declared, _ := schema["type"].(string)
	return declared == "string"
}

// v1alpha5MappingTargetSchema returns the exact protocol response Schema root and
// JSON Pointer a mapping target writes, together with the part/media-type
// compatibility result.
func v1alpha5MappingTargetSchema(link *v1alpha5LinkContext, representation ingresscatalog.ResponseRepresentationProjection, mapping ingresscatalog.ResponseMappingProjection) (ingresscatalog.RootRef, string, bool) {
	switch mapping.Target.Kind {
	case "json-body":
		if representation.MediaType != "application/json" {
			return ingresscatalog.RootRef{}, "", false
		}
		pointer, ok := v1alpha5DataPointer(mapping.Target.Pointer)
		if !ok {
			return ingresscatalog.RootRef{}, "", false
		}
		root, bound := link.schemaRoots[representation.ResponseSchemaRef]
		return root, pointer, bound
	case "multipart-json":
		if representation.MediaType != "multipart/related" {
			return ingresscatalog.RootRef{}, "", false
		}
		pointer, ok := v1alpha5DataPointer(mapping.Target.Pointer)
		if !ok {
			return ingresscatalog.RootRef{}, "", false
		}
		part, exists := v1alpha5FindPart(representation, mapping.Target.Part)
		if !exists || part.Kind != "json" || mapping.Target.Part != representation.RootPart {
			return ingresscatalog.RootRef{}, "", false
		}
		root, bound := link.schemaRoots[part.SchemaRef]
		return root, pointer, bound
	case "multipart-binary":
		if representation.MediaType != "multipart/related" {
			return ingresscatalog.RootRef{}, "", false
		}
		part, exists := v1alpha5FindPart(representation, mapping.Target.Part)
		if !exists || part.Kind != "binary" {
			return ingresscatalog.RootRef{}, "", false
		}
		// The binary target has no JSON pointer; the contentIdField is validated
		// separately. Use the JSON root part as the Schema root for the pointer.
		rootPart, exists := v1alpha5FindPart(representation, representation.RootPart)
		if !exists || rootPart.Kind != "json" {
			return ingresscatalog.RootRef{}, "", false
		}
		contentPointer, ok := v1alpha5DataPointer(mapping.Target.ContentIDPointer)
		if !ok {
			return ingresscatalog.RootRef{}, "", false
		}
		root, bound := link.schemaRoots[rootPart.SchemaRef]
		return root, contentPointer, bound
	}
	return ingresscatalog.RootRef{}, "", false
}

// v1alpha5JSONTypesCompatible is a conservative JSON type compatibility check for
// one typed-result field and its protocol response target field. Unresolved
// Schemas (no declared type) are not rejected here; full wire validation is
// Integration-adapter owned.
func v1alpha5JSONTypesCompatible(source, target any) bool {
	sourceTypes, sourceOK := v1alpha5JSONSchemaTypes(source)
	targetTypes, targetOK := v1alpha5JSONSchemaTypes(target)
	if !sourceOK || !targetOK {
		return true
	}
	for sourceType := range sourceTypes {
		if targetTypes[sourceType] {
			continue
		}
		if sourceType == "integer" && targetTypes["number"] {
			continue
		}
		return false
	}
	return true
}

func v1alpha5JSONSchemaTypes(schema any) (map[string]bool, bool) {
	object, ok := schema.(map[string]any)
	if !ok {
		return nil, false
	}
	types := map[string]bool{}
	switch declared := object["type"].(type) {
	case string:
		types[declared] = true
	case []any:
		for _, value := range declared {
			text, ok := value.(string)
			if !ok {
				return nil, false
			}
			types[text] = true
		}
	default:
		return nil, false
	}
	return types, len(types) > 0
}

// v1alpha5RequiredLeafPaths returns the exact JSON Pointers of every required
// protocol response field. Required object properties with required children are
// expanded recursively; a mapping that writes the whole object subtree also
// supplies them (v1alpha5JSONPointerCovers).
func v1alpha5RequiredLeafPaths(link *v1alpha5LinkContext, root ingresscatalog.RootRef) ([]string, error) {
	resolved, err := link.resolution.ResolveRoot(root)
	if err != nil {
		return nil, err
	}
	var paths []string
	if err := v1alpha5WalkRequiredLeaves(link, root, resolved.Schema, "", 0, &paths); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func v1alpha5WalkRequiredLeaves(link *v1alpha5LinkContext, root ingresscatalog.RootRef, schema any, path string, depth int, paths *[]string) error {
	if depth > 64 {
		return fmt.Errorf("required-field expansion exceeds 64 levels")
	}
	schema, root, err := v1alpha5ResolveSchemaRef(link, root, schema)
	if err != nil {
		return err
	}
	object, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	required, _ := object["required"].([]any)
	if len(required) == 0 {
		if path != "" {
			*paths = append(*paths, path)
		}
		return nil
	}
	properties, _ := object["properties"].(map[string]any)
	for _, value := range required {
		name, ok := value.(string)
		if !ok {
			return fmt.Errorf("required entry is not a string")
		}
		childPath := path + "/" + v1alpha5JSONPointerToken(name)
		childSchema, exists := properties[name]
		if !exists {
			return fmt.Errorf("required property %q is not declared", name)
		}
		if err := v1alpha5WalkRequiredLeaves(link, root, childSchema, childPath, depth+1, paths); err != nil {
			return err
		}
	}
	return nil
}

// v1alpha5ResolveSchemaRef follows exact $ref indirections against the trusted
// snapshot. Local fragment references are resolved against the same trusted
// artifact root; an absolute $ref is resolved only when the trusted schema
// bindings declare exactly that Schema identity. Anything else is an unresolved
// Schema error, never a guess.
func v1alpha5ResolveSchemaRef(link *v1alpha5LinkContext, root ingresscatalog.RootRef, schema any) (any, ingresscatalog.RootRef, error) {
	for hops := 0; hops < 64; hops++ {
		object, ok := schema.(map[string]any)
		if !ok {
			return schema, root, nil
		}
		rawRef, exists := object["$ref"]
		if !exists {
			return schema, root, nil
		}
		ref, ok := rawRef.(string)
		if !ok {
			return nil, root, fmt.Errorf("$ref is not a string")
		}
		if strings.HasPrefix(ref, "#") {
			root = ingresscatalog.RootRef{ArtifactRef: root.ArtifactRef, Pointer: "#" + ref[1:]}
		} else if bound, exists := link.schemaRoots[ref]; exists {
			root = bound
		} else {
			return nil, root, fmt.Errorf("absolute $ref %q is not a declared trusted Schema identity", ref)
		}
		resolved, err := link.resolution.ResolveRoot(root)
		if err != nil {
			return nil, root, err
		}
		schema = resolved.Schema
	}
	return nil, root, fmt.Errorf("$ref chain exceeds 64 hops")
}

// v1alpha5FindDisposition resolves an exact disposition id.
func v1alpha5FindDisposition(dispositions []ingresscatalog.ResponseDispositionProjection, id string) (*ingresscatalog.ResponseDispositionProjection, bool) {
	for index := range dispositions {
		if dispositions[index].ID == id {
			return &dispositions[index], true
		}
	}
	return nil, false
}

// v1alpha5FindRepresentation resolves an exact representation id.
func v1alpha5FindRepresentation(disposition ingresscatalog.ResponseDispositionProjection, id string) (*ingresscatalog.ResponseRepresentationProjection, bool) {
	for index := range disposition.Representations {
		if disposition.Representations[index].ID == id {
			return &disposition.Representations[index], true
		}
	}
	return nil, false
}

// v1alpha5FindPart resolves an exact multipart part name.
func v1alpha5FindPart(representation ingresscatalog.ResponseRepresentationProjection, name string) (*ingresscatalog.ResponsePartProjection, bool) {
	for index := range representation.Parts {
		if representation.Parts[index].Name == name {
			return &representation.Parts[index], true
		}
	}
	return nil, false
}

// v1alpha5JSONPointerCovers reports whether writing one mapping target pointer
// supplies a required field path (exact match or whole-subtree write).
func v1alpha5JSONPointerCovers(written, required string) bool {
	if written == required {
		return true
	}
	return strings.HasPrefix(required, written+"/")
}

// v1alpha5JSONValueKey returns one deterministic comparable key for a JSON scalar
// selector value.
func v1alpha5JSONValueKey(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%T:%v", value, value)
	}
	return string(raw)
}
