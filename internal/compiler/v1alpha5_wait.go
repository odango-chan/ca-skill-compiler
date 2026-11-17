package compiler

import (
	"regexp"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const V1Alpha5Target = "connectionagent.io/v1alpha5"

var v1alpha5WaitDurationPattern = regexp.MustCompile(`^P([1-9][0-9]*D)?(T([1-9][0-9]*H)?([1-9][0-9]*M)?([1-9][0-9]*S)?)?$`)

type V1Alpha5WaitAST struct {
	Raw      *yaml.Node
	Event    V1Alpha5IngressEventAST
	Timeout  string
	Response *V1Alpha5WaitResponseAST
	Routes   V1Alpha5WaitRoutesAST
}

type V1Alpha5IngressEventAST struct {
	Raw       *yaml.Node
	Source    string
	Service   string
	Operation string
}

type V1Alpha5WaitResponseAST struct {
	Raw   *yaml.Node
	After string
}

type V1Alpha5WaitRoutesAST struct {
	Raw     *yaml.Node
	Timeout string
}

// parseV1Alpha5WaitNode parses only the Node-entry Wait amendment. It is kept
// separate from the v1alpha4 parser so the released target never accepts the
// new field by accident. Graph/link validation is intentionally a later pass.
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
	if node == nil || node.Kind != yaml.MappingNode {
		add(node, "SOURCE_ROOT_INVALID", "Procedure Node must be a mapping")
		return nil, diagnostics
	}
	waitNode := child(node, "wait")
	if waitNode == nil {
		return nil, diagnostics
	}

	allowedNodeFields := map[string]bool{
		"wait": true, "precondition": true, "operation": true,
		"observation": true, "postcondition": true,
	}
	v1alpha5CheckUnknownFields(node, allowedNodeFields, add)

	if child(node, "precondition") == nil &&
		child(node, "operation") == nil &&
		child(node, "observation") == nil &&
		child(node, "postcondition") == nil {
		add(waitNode, "WAIT_NODE_STAGE_REQUIRED", "Wait-bearing Node must declare at least one business stage")
	}

	waitNode = unalias(waitNode)
	if waitNode == nil || waitNode.Kind != yaml.MappingNode {
		add(waitNode, "WAIT_DECLARATION_INVALID", "Node "+nodeID+" wait must be a mapping")
		return nil, diagnostics
	}
	v1alpha5CheckUnknownFields(waitNode, map[string]bool{
		"event": true, "timeout": true, "response": true, "routes": true,
	}, add)

	result := &V1Alpha5WaitAST{Raw: waitNode}

	event := child(waitNode, "event")
	event = unalias(event)
	if event == nil || event.Kind != yaml.MappingNode {
		add(event, "WAIT_DECLARATION_INVALID", "Node "+nodeID+" wait.event must be a mapping")
	} else {
		v1alpha5CheckUnknownFields(event, map[string]bool{
			"source": true, "service": true, "operation": true,
		}, add)
		result.Event = V1Alpha5IngressEventAST{Raw: event}
		for _, field := range []string{"source", "service", "operation"} {
			valueNode := child(event, field)
			value, ok := v1alpha5String(valueNode)
			if !ok || value == "" {
				add(valueNode, "WAIT_DECLARATION_INVALID", "Node "+nodeID+" wait.event."+field+" must be a non-empty YAML string")
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

	timeoutNode := child(waitNode, "timeout")
	timeout, timeoutIsString := v1alpha5String(timeoutNode)
	if !timeoutIsString || !validV1Alpha5FixedDuration(timeout) {
		add(timeoutNode, "WAIT_TIMEOUT_INVALID", "Node "+nodeID+" wait.timeout must be a positive fixed duration in the v1alpha5 ISO-8601 subset")
	} else {
		result.Timeout = timeout
	}

	responseNode := child(waitNode, "response")
	if responseNode != nil {
		responseNode = unalias(responseNode)
		if responseNode.Kind != yaml.MappingNode {
			add(responseNode, "WAIT_RESPONSE_BOUNDARY_INVALID", "Node "+nodeID+" wait.response must be a mapping")
		} else {
			v1alpha5CheckUnknownFields(responseNode, map[string]bool{"after": true}, add)
			result.Response = &V1Alpha5WaitResponseAST{Raw: responseNode}
			if afterNode := child(responseNode, "after"); afterNode != nil {
				after, ok := v1alpha5String(afterNode)
				if !ok || !v4NodeID.MatchString(after) || v4Terminals[after] || after == "context" {
					add(afterNode, "WAIT_RESPONSE_BOUNDARY_INVALID", "Node "+nodeID+" wait.response.after must be a NodeId")
				} else {
					result.Response.After = after
				}
			}
		}
	}

	routesNode := child(waitNode, "routes")
	routesNode = unalias(routesNode)
	if routesNode == nil || routesNode.Kind != yaml.MappingNode {
		add(routesNode, "WAIT_TIMEOUT_ROUTE_INVALID", "Node "+nodeID+" wait.routes must be a mapping containing exactly TIMEOUT")
	} else {
		keys := mapKeys(routesNode)
		if len(keys) != 1 || keys[0] != "TIMEOUT" {
			add(routesNode, "WAIT_TIMEOUT_ROUTE_INVALID", "Node "+nodeID+" wait.routes must contain exactly the TIMEOUT key")
		}
		if timeoutTargetNode := child(routesNode, "TIMEOUT"); timeoutTargetNode != nil {
			target, ok := v1alpha5String(timeoutTargetNode)
			if !ok || !validV1Alpha5WaitTimeoutTarget(target) {
				add(timeoutTargetNode, "WAIT_TIMEOUT_ROUTE_INVALID", "Node "+nodeID+" wait.routes.TIMEOUT must target a declared Node or FAILED, UNRESOLVED, SAFE_STOPPED")
			} else {
				result.Routes = V1Alpha5WaitRoutesAST{Raw: routesNode, Timeout: target}
			}
		}
	}

	return result, diagnostics
}

func v1alpha5CheckUnknownFields(node *yaml.Node, allowed map[string]bool, add func(*yaml.Node, string, string)) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := unalias(node.Content[i])
		if key == nil || key.Kind != yaml.ScalarNode || key.Tag != "!!str" || allowed[key.Value] {
			continue
		}
		add(key, "UNKNOWN_FIELD", "Unsupported v1alpha5 author field "+key.Value)
	}
}

func v1alpha5String(node *yaml.Node) (string, bool) {
	node = unalias(node)
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", false
	}
	return node.Value, true
}

// validV1Alpha5FixedDuration implements the exact fixed-duration lexical subset
// in v1alpha5 Appendix L.1.1. It deliberately does not convert to time.Duration:
// arbitrarily large positive integer components remain lexical values and the
// deployment-specific maximum is checked by Host admission, not Compiler.
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

func validV1Alpha5WaitTimeoutTarget(target string) bool {
	switch target {
	case "FAILED", "UNRESOLVED", "SAFE_STOPPED":
		return true
	case "SUCCEEDED", "context", "":
		return false
	default:
		return v4NodeID.MatchString(target) && !v4Terminals[target]
	}
}
