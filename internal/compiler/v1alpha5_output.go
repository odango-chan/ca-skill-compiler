package compiler

// CA-SL v1alpha5 WP4: typed wait.output Producer SourcePath (Appendix L.2.3,
// L.11/L.11.2, EW-012).
//
// An accepted Node-entry Wait (WP2/WP5) produces one typed value named by the
// SourcePath <nodeId>.wait.output. That value is the exact normalized request
// value of the trusted Ingress Operation export linked by the WP3 pass (#62
// normalized request Schema, Schema Closure Digest v1 verified); it is not raw
// transport data and it is neither global Context nor an Operation/Observation
// Result or Evidence. It becomes available only after the event has been accepted
// for that exact Wait occurrence.
//
// This file only interprets the wait-specific Producer form. The inherited
// v1alpha4 SourcePath machinery is reused rather than forked:
//
//   - the inherited scalar classification decides whether an argument value is a
//     SourcePath (a plain !!str) or a literal (quoted string, bool/null/int/float);
//   - the inherited diagnostics SOURCE_PATH_INVALID / SOURCE_NOT_AVAILABLE and the
//     existing INPUT_TYPE_MISMATCH / INPUT_NULLABILITY_MISMATCH identities are
//     used, never a Wait-specific code (L.11.2);
//   - JSON Pointer property paths resolve through the trusted Schema lookup of the
//     WP1 ResolutionContext (the same machinery WP3 uses for correlation paths);
//   - type assignment reuses the existing G.4/F.5 assignability engine.
//
// Availability is proven with the WP5 Procedure graph. A Wait-bearing Node's
// business vertex is only reachable through the implicit EVENT_RECEIVED control
// edge from its entry-control vertex, while the TIMEOUT edge bypasses the business
// vertex. Therefore a consumer may read the value exactly when that business
// vertex dominates the consumer Node: every path from the Procedure start to the
// consumer then passes event acceptance. The same Node's business stages are
// covered because a vertex dominates itself. A timeout handler, or any consumer
// reachable through a path that does not accept the event, is not dominated and is
// rejected with SOURCE_NOT_AVAILABLE anchored at the consuming argument.

import (
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// v1alpha5WaitOutputConsumer is one argument position that reads a wait.output
// SourcePath. key is the named argument key for operation/observation arguments;
// it is empty for positional PredicateCall arguments.
type v1alpha5WaitOutputConsumer struct {
	nodeID string
	stage  string // operation | observation | precondition | postcondition
	key    string
	value  *yaml.Node
}

// validateV1Alpha5WaitOutputSourcePaths is the WP4 pass over one parsed v1alpha5
// Skill. It runs after the WP3 binding pass and uses its validated linking context
// (nil when a WP3 prerequisite failed; binding-dependent diagnostics are then
// suppressed, but the pure producer-form and availability rules still apply).
func validateV1Alpha5WaitOutputSourcePaths(skill *V1Alpha5SkillAST, procedurePath string, source *v1alpha5WaitBindingSource) []Diagnostic {
	consumers := collectV1Alpha5WaitOutputConsumers(skill)
	if len(consumers) == 0 {
		return nil
	}
	graph := buildV1Alpha5ProcedureGraph(skill.Procedure)
	consumerIndexes := make(map[string]*snapshotExportIndex)
	var diagnostics []Diagnostic
	for _, consumer := range consumers {
		value := consumer.value
		segments := strings.Split(value.Value, ".")
		invalidSegment := ""
		for _, segment := range segments {
			if !validDependencyID(segment) {
				invalidSegment = segment
				break
			}
		}
		if invalidSegment != "" {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(procedurePath, value, "SOURCE_PATH_INVALID",
				"SourcePath segment is not a valid Identifier: "+invalidSegment))
			continue
		}
		waitNodeID := segments[0]
		node := skill.Procedure.ByID[waitNodeID]
		if node == nil || node.Wait == nil {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(procedurePath, value, "SOURCE_PATH_INVALID",
				"Wait output SourcePath root is not a declared Wait-bearing Node: "+waitNodeID))
			continue
		}
		if len(segments) < 3 || segments[2] != "output" {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(procedurePath, value, "SOURCE_PATH_INVALID",
				"Wait output SourcePath must be <nodeId>.wait.output[.<property>...]: "+value.Value))
			continue
		}
		// Event-accepted-path availability. The Wait-node business vertex is only
		// reachable through EVENT_RECEIVED; a TIMEOUT handler or any bypass path is
		// therefore not dominated. The same Node's stages are self-dominated.
		if !graph.Dominates(waitNodeID, consumer.nodeID) {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(procedurePath, value, "SOURCE_NOT_AVAILABLE",
				"Wait output is available only on paths that accept the event at Wait Node '"+waitNodeID+"'; it is not available to Node '"+consumer.nodeID+"': "+value.Value))
			continue
		}
		if source == nil {
			continue
		}
		contract := source.contracts[waitNodeID]
		if contract == nil {
			// The exact linked export/Schema is unavailable (already diagnosed by
			// WP3); dependent Schema/typing diagnostics are suppressed.
			continue
		}
		sourceSchema, problem := v1alpha5WaitOutputSchema(source.link, contract, segments[3:])
		if problem != "" {
			diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(procedurePath, value, "SOURCE_PATH_INVALID", problem))
			continue
		}
		if sourceSchema == nil || consumer.key == "" {
			// Predicate parameter typing has no parsed v1alpha5 Predicate registry
			// yet; availability/Schema binding above still apply.
			continue
		}
		targetSchema := source.consumerArgumentSchema(skill, consumer, consumerIndexes)
		if targetSchema == nil {
			// The consumer's expected type is not statically known; nothing to
			// prove. Missing consumer Tool/linking diagnostics belong to later
			// work packages, so no diagnostic is inferred here.
			continue
		}
		reason, assignable := f5Assignable(sourceSchema, targetSchema)
		if assignable {
			continue
		}
		if reason == assignableUnresolved {
			// The F.5 engine cannot prove compatibility without full closure of the
			// compared Schemas; conservative non-typing, not a mismatch.
			continue
		}
		code := "INPUT_TYPE_MISMATCH"
		if reason == assignableNullabilityMismatch {
			code = "INPUT_NULLABILITY_MISMATCH"
		}
		diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(procedurePath, value, code,
			"Wait output value is not assignable to the Owner argument Schema: "+consumer.key))
	}
	return diagnostics
}

// collectV1Alpha5WaitOutputConsumers gathers every argument position whose plain
// scalar value is a wait.output Producer SourcePath. Named Tool arguments and
// positional PredicateCall arguments are both source positions (the inherited
// SourcePath grammar). Nothing else in the Procedure is scanned.
func collectV1Alpha5WaitOutputConsumers(skill *V1Alpha5SkillAST) []v1alpha5WaitOutputConsumer {
	consumers := make([]v1alpha5WaitOutputConsumer, 0)
	if skill == nil || skill.Procedure == nil {
		return consumers
	}
	appendArguments := func(nodeID, stage string, arguments *yaml.Node) {
		arguments = unalias(arguments)
		if arguments == nil || arguments.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(arguments.Content); i += 2 {
			key := scalar(unalias(arguments.Content[i]))
			value := unalias(arguments.Content[i+1])
			if _, ok := v1alpha5WaitOutputPath(value); !ok {
				continue
			}
			consumers = append(consumers, v1alpha5WaitOutputConsumer{nodeID: nodeID, stage: stage, key: key, value: value})
		}
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[nodeID]
		if node == nil {
			continue
		}
		appendArguments(nodeID, "operation", child(node.Operation, "arguments"))
		appendArguments(nodeID, "observation", child(node.Observation, "arguments"))
		for _, stage := range []struct {
			name string
			raw  *yaml.Node
		}{{"precondition", node.Precondition}, {"postcondition", node.Postcondition}} {
			for _, call := range sequence(child(stage.raw, "predicates")) {
				for _, argument := range sequence(child(call, "arguments")) {
					value := unalias(argument)
					if _, ok := v1alpha5WaitOutputPath(value); !ok {
						continue
					}
					consumers = append(consumers, v1alpha5WaitOutputConsumer{nodeID: nodeID, stage: stage.name, value: value})
				}
			}
		}
	}
	return consumers
}

// v1alpha5WaitOutputPath reports whether one argument scalar is a plain
// wait.output Producer SourcePath and returns its text. The inherited scalar
// classification applies: quoted strings and YAML bool/null/int/float scalars are
// literals; only a plain !!str is a SourcePath.
func v1alpha5WaitOutputPath(node *yaml.Node) (string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", false
	}
	if node.Style == yaml.SingleQuotedStyle || node.Style == yaml.DoubleQuotedStyle {
		return "", false
	}
	segments := strings.Split(node.Value, ".")
	if len(segments) < 2 || segments[1] != "wait" {
		return "", false
	}
	return node.Value, true
}

// v1alpha5WaitOutputSchema resolves the producer type of one wait.output
// SourcePath against the exact linked normalized request Schema root. For a bare
// value the producer type is the normalized request object Schema itself; for a
// property path every segment resolves through the trusted Schema lookup, and a
// property that is not declared exactly is a SOURCE_PATH_INVALID problem
// (L.11.2: undeclared Wait output Producer property).
//
// SourcePath segments are inherited Identifiers ([A-Za-z][A-Za-z0-9_-]*), so a
// segment can never contain the RFC 6901 tilde or slash escapes: the mapping from
// dotted path to canonical pointer needs no escaping beyond joining the segments.
func v1alpha5WaitOutputSchema(link *v1alpha5LinkContext, contract *v1alpha5ExportContract, propertyPath []string) (*yaml.Node, string) {
	if link == nil || link.resolution == nil || contract == nil {
		return nil, ""
	}
	if len(propertyPath) == 0 {
		root, err := link.resolution.ResolveRoot(contract.Normalized)
		if err != nil {
			// WP3 already proved this exact root through Schema Closure Digest v1;
			// a failure here is not a new author defect to report.
			return nil, ""
		}
		return schemaNode(root.Schema), ""
	}
	dataPointer := "#/" + strings.Join(propertyPath, "/")
	result, err := link.resolution.LookupSchemaPath(contract.Normalized, dataPointer)
	if err != nil {
		return nil, "Wait output Schema does not declare the path '" + strings.Join(propertyPath, ".") + "'."
	}
	return schemaNode(result.Schema), ""
}

// consumerArgumentSchema resolves the statically known expected Schema of one
// consumer argument position from the trusted ToolCatalog Operation/Observation
// Input (QueryContext) Schema. It is best-effort and silent: an unresolved
// consumer Tool export or Schema leaves the expected type unknown, and the
// corresponding linking diagnostic is not owned by this pass.
func (source *v1alpha5WaitBindingSource) consumerArgumentSchema(skill *V1Alpha5SkillAST, consumer v1alpha5WaitOutputConsumer, indexes map[string]*snapshotExportIndex) *yaml.Node {
	if skill == nil || skill.Procedure == nil || source == nil || source.link == nil || source.link.resolution == nil {
		return nil
	}
	node := skill.Procedure.ByID[consumer.nodeID]
	if node == nil {
		return nil
	}
	stage := v1alpha5ProcedureStage(node, consumer.stage)
	alias, exportID, ok := splitQualifiedReference(child(stage, "tool"))
	if !ok {
		return nil
	}
	dependency, ok := source.scopeByAlias[alias]
	if !ok {
		return nil
	}
	index, cached := indexes[alias]
	if !cached {
		built, _, _ := buildSnapshotExportIndex(dependency.Snapshot)
		index = &built
		indexes[alias] = index
	}
	records := index.callables[exportID]
	if len(records) != 1 {
		return nil
	}
	record := records[0]
	expectedKind := "operation"
	if consumer.stage == "observation" {
		expectedKind = "observation"
	}
	if record.ExportKind != expectedKind {
		return nil
	}
	if _, ownerDiagnostics := attributeExportOwner(record, dependency.Binding, nil); len(ownerDiagnostics) > 0 {
		return nil
	}
	schemaID := record.InputSchemaID
	if expectedKind == "observation" {
		schemaID = record.QueryContextSchemaID
	}
	if schemaID == "" {
		return nil
	}
	root, ok := source.link.schemaRoots[schemaID]
	if !ok {
		return nil
	}
	argumentPointer := "#/" + v1alpha5JSONPointerToken(consumer.key)
	result, err := source.link.resolution.LookupSchemaPath(root, argumentPointer)
	if err != nil {
		return nil
	}
	return schemaNode(result.Schema)
}

// v1alpha5JSONPointerToken applies RFC 6901 escaping to one pointer token.
func v1alpha5JSONPointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	return strings.ReplaceAll(token, "/", "~1")
}
