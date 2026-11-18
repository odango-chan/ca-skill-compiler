package compiler

// CA-SL v1alpha5 Node-entry Wait parser (Appendix L.1, L.1.1, L.11).
//
// The amendment keeps Wait as an entry control on an ordinary business Node:
// it is not a fifth business stage and not a separate Wait Node kind. This file
// only owns the lexical/structural obligations that are decidable from the
// authored Procedure source. Trigger/correlation linking (WP3), wait.output
// SourcePath (WP4), Procedure graph edges and dominance (WP5) and response
// boundary reachability/Producer mapping (WP6) are later passes.

import (
	"fmt"
	"regexp"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// v1alpha5WaitDurationPattern is the exact Appendix L.1.1 lexical subset:
//
//	FixedDuration   = "P", (DayComponent, ["T", TimeComponents]
//	                         | "T", TimeComponents)
//	TimeComponents  = HourComponent, [MinuteComponent], [SecondComponent]
//	                | MinuteComponent, [SecondComponent]
//	                | SecondComponent
//	DayComponent    = PositiveInteger, "D"
//	HourComponent   = PositiveInteger, "H"
//	MinuteComponent = PositiveInteger, "M"
//	SecondComponent = PositiveInteger, "S"
//	PositiveInteger = NonZeroDigit, { Digit }
//
// Go's regexp is leftmost-first with greedy quantifiers, so the optional
// components cannot be reordered or partially matched: a trailing mismatch
// fails the anchored match. validV1Alpha5FixedDuration additionally rejects a
// bare "P", a bare "...T" and the empty time part.
var v1alpha5WaitDurationPattern = regexp.MustCompile(`^P([1-9][0-9]*D)?(T([1-9][0-9]*H)?([1-9][0-9]*M)?([1-9][0-9]*S)?)?$`)

// v1alpha5NodeFields is the exact v1alpha5 Procedure Node field set after the
// L.1 amendment: the Wait entry control plus the four inherited business
// stages. No await/polling/onTimeout/payloadSchema/EventCatalog field exists.
var v1alpha5NodeFields = map[string]bool{
	"wait": true, "precondition": true, "operation": true,
	"observation": true, "postcondition": true,
}

var v1alpha5WaitFields = map[string]bool{
	"event": true, "timeout": true, "response": true, "routes": true,
}

var v1alpha5EventFields = map[string]bool{
	"source": true, "service": true, "operation": true,
}

var v1alpha5ResponseFields = map[string]bool{"after": true}

var v1alpha5WaitBusinessStages = []string{"precondition", "operation", "observation", "postcondition"}

// V1Alpha5WaitAST is the parsed Node-entry Wait declaration. Raw points into
// the single normalized YAML parse result owned by the Skill AST.
type V1Alpha5WaitAST struct {
	Raw      *yaml.Node
	Span     SourceSpan
	Event    V1Alpha5IngressEventAST
	Timeout  string
	Response *V1Alpha5WaitResponseAST
	Routes   V1Alpha5WaitRoutesAST
}

// V1Alpha5IngressEventAST is the three-part exact Ingress Operation selector.
// Values are YAML strings compared exactly against a trusted export; no
// case-folding, normalization or free-form filter is applied.
type V1Alpha5IngressEventAST struct {
	Raw       *yaml.Node
	Span      SourceSpan
	Source    string
	Service   string
	Operation string
}

// V1Alpha5WaitResponseAST is the optional paired-response boundary metadata.
// After is empty when the author omits it; the default boundary is then the
// Wait-bearing Node's own completion (Appendix L.1 WaitResponse = map(after?:
// NodeId), L.8.1 default-boundary rule).
type V1Alpha5WaitResponseAST struct {
	Raw   *yaml.Node
	Span  SourceSpan
	After string
}

// V1Alpha5WaitRoutesAST is the mandatory single TIMEOUT Wait-control edge.
type V1Alpha5WaitRoutesAST struct {
	Raw     *yaml.Node
	Span    SourceSpan
	Timeout string
}

// parseV1Alpha5WaitNode parses the Wait declaration of one Procedure Node. It
// also enforces the Node field set and the "wait does not count as a business
// stage" rule for wait-bearing Nodes. It returns (nil, nil) when the Node has
// no wait field.
func parseV1Alpha5WaitNode(path, nodeID string, node *yaml.Node) (*V1Alpha5WaitAST, []Diagnostic) {
	var diagnostics []Diagnostic
	add := func(at *yaml.Node, code, message string) {
		line, column := 0, 0
		if at != nil {
			line, column = at.Line, at.Column
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Phase:    "C_STATIC",
			Code:     code,
			Path:     limitDiagnosticPath(path),
			Line:     line,
			Column:   column,
			Message:  limitDiagnosticMessage(message),
		})
	}

	node = unalias(node)
	if node == nil {
		return nil, nil
	}
	if node.Kind != yaml.MappingNode {
		add(node, "SOURCE_ROOT_INVALID", "Procedure Node must be a mapping")
		return nil, diagnostics
	}

	v1alpha5CheckUnknownFields(node, v1alpha5NodeFields, add)

	waitNode := child(node, "wait")
	if waitNode == nil {
		return nil, diagnostics
	}

	// Wait must accompany at least one of the four existing business stages.
	// A Wait-only Node is invalid; the normal event edge is implicit.
	if !v1alpha5HasBusinessStage(node) {
		add(waitNode, "WAIT_NODE_STAGE_REQUIRED", v1alpha5WaitNodeStageMessage(nodeID))
	}

	waitNode = unalias(waitNode)
	if waitNode == nil || waitNode.Kind != yaml.MappingNode {
		add(waitNode, "WAIT_DECLARATION_INVALID", v1alpha5WaitDeclarationMessage(nodeID, "wait"))
		return nil, diagnostics
	}

	v1alpha5CheckUnknownFields(waitNode, v1alpha5WaitFields, add)

	result := &V1Alpha5WaitAST{Raw: waitNode, Span: sourceSpan(path, waitNode)}

	// wait.event: required mapping with exactly source/service/operation, each a
	// non-empty YAML !!str compared exactly.
	eventNode := unalias(child(waitNode, "event"))
	if eventNode == nil || eventNode.Kind != yaml.MappingNode {
		add(v1alpha5FallbackNode(eventNode, waitNode), "WAIT_DECLARATION_INVALID",
			v1alpha5WaitDeclarationMessage(nodeID, "wait.event"))
	} else {
		v1alpha5CheckUnknownFields(eventNode, v1alpha5EventFields, add)
		result.Event = V1Alpha5IngressEventAST{Raw: eventNode, Span: sourceSpan(path, eventNode)}
		for _, field := range []string{"source", "service", "operation"} {
			valueNode := child(eventNode, field)
			value, ok := v1alpha5NonEmptyString(valueNode)
			if !ok {
				add(v1alpha5FallbackNode(valueNode, eventNode), "WAIT_DECLARATION_INVALID",
					v1alpha5WaitDeclarationMessage(nodeID, "wait.event."+field))
				continue
			}
			switch field {
			case "source":
				result.Event.Source = value
			case "service":
				result.Event.Service = value
			case "operation":
				result.Event.Operation = value
			}
		}
	}

	// wait.timeout: required YAML string in the exact fixed-duration subset.
	// The deployment maximum is a Host admission concern and is deliberately
	// not applied here; large positive components remain lexical values.
	timeoutNode := child(waitNode, "timeout")
	timeout, timeoutIsString := v1alpha5String(timeoutNode)
	if !timeoutIsString || !validV1Alpha5FixedDuration(timeout) {
		add(v1alpha5FallbackNode(timeoutNode, waitNode), "WAIT_TIMEOUT_INVALID", v1alpha5WaitTimeoutMessage(nodeID))
	} else {
		result.Timeout = timeout
	}

	// wait.response: optional. Appendix L.1 declares the field as
	// WaitResponse = map(after?: NodeId); L.8.1 makes the Wait-bearing Node's
	// own completion the default response boundary when "after" is absent, so a
	// present response mapping WITHOUT "after" is valid. Only a present "after"
	// is validated as a NodeId that is neither a terminal nor "context".
	// Reachability and the post-dominance proof belong to WP6.
	//
	// A present response value that is not a mapping is a Wait object shape
	// violation (L.11 -> WAIT_DECLARATION_INVALID), not a response-boundary
	// semantics failure.
	if responseNode := unalias(child(waitNode, "response")); responseNode != nil {
		if responseNode.Kind != yaml.MappingNode {
			add(responseNode, "WAIT_DECLARATION_INVALID", v1alpha5WaitDeclarationMessage(nodeID, "wait.response"))
		} else {
			v1alpha5CheckUnknownFields(responseNode, v1alpha5ResponseFields, add)
			result.Response = &V1Alpha5WaitResponseAST{Raw: responseNode, Span: sourceSpan(path, responseNode)}
			if afterNode := child(responseNode, "after"); afterNode != nil {
				after, afterIsString := v1alpha5String(afterNode)
				if !afterIsString || !validV1Alpha5ResponseBoundaryNodeID(after) {
					add(afterNode, "WAIT_RESPONSE_BOUNDARY_INVALID", v1alpha5WaitResponseMessage(nodeID))
				} else {
					result.Response.After = after
				}
			}
		}
	}

	// wait.routes: required mapping with exactly one key TIMEOUT. EVENT_RECEIVED
	// and any other author-declared route key are invalid; the event edge is
	// implicit. Undeclared NodeId targets are CONTROL_FLOW_DANGLING (WP5).
	routesNode := unalias(child(waitNode, "routes"))
	if routesNode == nil || routesNode.Kind != yaml.MappingNode {
		add(v1alpha5FallbackNode(routesNode, waitNode), "WAIT_TIMEOUT_ROUTE_INVALID", v1alpha5WaitTimeoutRouteMessage(nodeID))
	} else {
		result.Routes = V1Alpha5WaitRoutesAST{Raw: routesNode, Span: sourceSpan(path, routesNode)}
		keys := mapKeys(routesNode)
		if len(keys) != 1 || keys[0] != "TIMEOUT" {
			// Anchor an author-declared extra route key when one exists, else the
			// routes mapping (the missing-TIMEOUT parent).
			add(v1alpha5FallbackNode(v1alpha5FirstExtraRouteKeyNode(routesNode), routesNode),
				"WAIT_TIMEOUT_ROUTE_INVALID", v1alpha5WaitTimeoutRouteMessage(nodeID))
		}
		if timeoutTargetNode := child(routesNode, "TIMEOUT"); timeoutTargetNode != nil {
			target, ok := v1alpha5String(timeoutTargetNode)
			if !ok || !validV1Alpha5WaitTimeoutTarget(target) {
				add(timeoutTargetNode, "WAIT_TIMEOUT_ROUTE_INVALID", v1alpha5WaitTimeoutRouteMessage(nodeID))
			} else {
				result.Routes.Timeout = target
			}
		}
	}

	return result, diagnostics
}

func v1alpha5HasBusinessStage(node *yaml.Node) bool {
	for _, stage := range v1alpha5WaitBusinessStages {
		if child(node, stage) != nil {
			return true
		}
	}
	return false
}

func v1alpha5FirstExtraRouteKeyNode(routes *yaml.Node) *yaml.Node {
	routes = unalias(routes)
	if routes == nil || routes.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(routes.Content); i += 2 {
		key := unalias(routes.Content[i])
		if key != nil && key.Value != "TIMEOUT" {
			return key
		}
	}
	return nil
}

// v1alpha5CheckUnknownFields reports every mapping key outside the exact
// allowed set as UNKNOWN_FIELD anchored at the key node (normal unknown-field
// rule, reused from the inherited grammar).
func v1alpha5CheckUnknownFields(node *yaml.Node, allowed map[string]bool, add func(*yaml.Node, string, string)) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := unalias(node.Content[i])
		if key == nil {
			continue
		}
		if key.Kind == yaml.ScalarNode && allowed[key.Value] {
			continue
		}
		add(key, "UNKNOWN_FIELD", "Unsupported CA-SL author field "+key.Value)
	}
}

func v1alpha5FallbackNode(node, fallback *yaml.Node) *yaml.Node {
	if node != nil {
		return node
	}
	return fallback
}

func v1alpha5String(node *yaml.Node) (string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", false
	}
	return node.Value, true
}

func v1alpha5NonEmptyString(node *yaml.Node) (string, bool) {
	value, ok := v1alpha5String(node)
	return value, ok && value != ""
}

// validV1Alpha5FixedDuration implements the exact fixed-duration lexical
// subset in v1alpha5 Appendix L.1.1. It deliberately does not convert to
// time.Duration: arbitrarily large positive integer components remain lexical
// values and the deployment-specific maximum is checked by Host admission.
func validV1Alpha5FixedDuration(value string) bool {
	matches := v1alpha5WaitDurationPattern.FindStringSubmatch(value)
	if matches == nil {
		return false
	}
	hasDay := matches[1] != ""
	hasT := matches[2] != ""
	hasTime := matches[3] != "" || matches[4] != "" || matches[5] != ""
	if !hasDay && !hasT {
		return false
	}
	if hasT && !hasTime {
		return false
	}
	return true
}

// validV1Alpha5ResponseBoundaryNodeID accepts the inherited v1alpha4 NodeId
// grammar only. Terminals and "context" are not NodeIds for this purpose.
func validV1Alpha5ResponseBoundaryNodeID(value string) bool {
	if value == "" || value == "context" || reservedTerminals[value] {
		return false
	}
	return nodeIDPattern.MatchString(value)
}

// validV1Alpha5WaitTimeoutTarget accepts another declared NodeId or exactly
// the three permitted stopping terminals. SUCCEEDED (and every other terminal)
// is forbidden: a timeout cannot prove the business Goal.
func validV1Alpha5WaitTimeoutTarget(target string) bool {
	switch target {
	case "FAILED", "UNRESOLVED", "SAFE_STOPPED":
		return true
	case "SUCCEEDED", "context", "":
		return false
	default:
		return nodeIDPattern.MatchString(target) && !reservedTerminals[target]
	}
}

func v1alpha5WaitDeclarationMessage(nodeID, fieldPath string) string {
	return fmt.Sprintf("Node '%s' has an invalid wait declaration at '%s'.", nodeID, fieldPath)
}

func v1alpha5WaitTimeoutMessage(nodeID string) string {
	return fmt.Sprintf("Node '%s' has an invalid wait.timeout; expected a positive fixed ISO-8601 duration in the v1alpha5 subset.", nodeID)
}

func v1alpha5WaitTimeoutRouteMessage(nodeID string) string {
	return fmt.Sprintf("Node '%s' wait.routes must contain exactly TIMEOUT with a valid non-success target.", nodeID)
}

func v1alpha5WaitNodeStageMessage(nodeID string) string {
	return fmt.Sprintf("Node '%s' declares wait but has no business stage.", nodeID)
}

func v1alpha5WaitResponseMessage(nodeID string) string {
	return fmt.Sprintf("Node '%s' wait.response.after does not identify one reachable response boundary on every accepted-event path.", nodeID)
}
