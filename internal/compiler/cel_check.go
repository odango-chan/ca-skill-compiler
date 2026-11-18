package compiler

// WP4D: CA-SL v1alpha4 CEL expression subset parsing/type checking.
//
// cel-go is the ADR-selected parser/type checker; it does NOT define the accepted
// CA-SL expression subset. This adapter compiles the expression against the
// Predicate's declared ordered parameters, requires a statically boolean result,
// and walks the parsed tree against the explicit v1alpha4 whitelist. Runtime
// evaluation is never performed.
//
// Documented WP4D limitation: external Schema-typed parameters use a dynamic CEL
// type in this slice; field-level typing from the linked Schema is WP6 work.
// Optional/null guard proofs are also deferred to WP6.

import (
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

var celAllowedFunctions = map[string]bool{
	"_&&_": true, "_||_": true, "!_": true,
	"_==_": true, "_!=_": true,
	"_<_": true, "_<=_": true, "_>_": true, "_>=_": true,
	"has": true, "size": true,
}

// checkCELExpression returns an empty string when the expression is a
// legal v1alpha4 CEL expression with a statically boolean result.
func checkCELExpression(expression string, parameters map[string]string) string {
	if expression == "" {
		return "CEL expression must be a nonempty string"
	}
	options := make([]cel.EnvOption, 0, len(parameters))
	for _, name := range sortedStringMapKeys(parameters) {
		celType := celPrimitiveType(parameters[name])
		options = append(options, cel.Variable(name, celType))
	}
	env, err := cel.NewEnv(options...)
	if err != nil {
		return "CEL environment could not be created: " + err.Error()
	}
	checked, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return "CEL expression is not statically valid: " + issues.Err().Error()
	}
	if checked.OutputType() != cel.BoolType {
		return "CEL expression must have a statically boolean result"
	}
	if problem := celSubsetViolation(checked.NativeRep().Expr()); problem != "" {
		return problem
	}
	return ""
}

func celPrimitiveType(declared string) *cel.Type {
	switch declared {
	case "string":
		return cel.StringType
	case "integer":
		return cel.IntType
	case "number":
		return cel.DoubleType
	case "boolean":
		return cel.BoolType
	default:
		// Qualified external Schema parameter: exact field typing lands in WP6.
		return cel.DynType
	}
}

// celSubsetViolation walks the parsed CEL AST against the frozen v1alpha4 subset.
func celSubsetViolation(expression ast.Expr) string {
	if expression == nil {
		return "CEL expression is empty"
	}
	switch expression.Kind() {
	case ast.LiteralKind:
		return ""
	case ast.IdentKind:
		return ""
	case ast.SelectKind:
		operand := expression.AsSelect().Operand()
		return celSubsetViolation(operand)
	case ast.CallKind:
		call := expression.AsCall()
		if call.IsMemberFunction() {
			return "CEL method calls are not supported in CA-SL v1alpha4"
		}
		if !celAllowedFunctions[call.FunctionName()] {
			return fmt.Sprintf("CEL function %q is not supported in CA-SL v1alpha4", call.FunctionName())
		}
		for _, argument := range call.Args() {
			if problem := celSubsetViolation(argument); problem != "" {
				return problem
			}
		}
		return ""
	default:
		return "CEL construct is outside the CA-SL v1alpha4 expression subset"
	}
}

// checkPredicateCEL validates Predicate parameter signatures and every cel
// evaluator expression. Script evaluators are WP9 scope and are left untouched.
func checkPredicateCEL(file string, source *PredicateSetAST, add func(string, *yaml.Node, string, string)) {
	if source == nil {
		return
	}
	for _, predicateID := range sortedStringMapKeys(source.Definitions) {
		predicate := source.Definitions[predicateID]
		if predicate == nil || predicate.Raw == nil {
			continue
		}
		parameters := map[string]string{}
		seen := map[string]bool{}
		for _, parameter := range sequence(predicate.Parameters) {
			parameter = unalias(parameter)
			if parameter == nil || parameter.Kind != yaml.MappingNode {
				add(file, parameter, "PREDICATE_GROUP_INVALID", "Each parameter is a one-key declaration")
				continue
			}
			for i := 0; i+1 < len(parameter.Content); i += 2 {
				nameNode := unalias(parameter.Content[i])
				typeNode := unalias(parameter.Content[i+1])
				name := scalar(nameNode)
				declared := scalar(typeNode)
				if name == "" || seen[name] {
					add(file, nameNode, "PREDICATE_GROUP_INVALID", "Parameter names must be unique nonempty strings")
					continue
				}
				seen[name] = true
				switch declared {
				case "string", "integer", "number", "boolean":
					parameters[name] = declared
				default:
					if strings.Contains(declared, "::") && validStablePackageID(strings.SplitN(declared, "::", 2)[1]) {
						parameters[name] = "schema"
					} else {
						add(file, typeNode, "PREDICATE_GROUP_INVALID", "Parameter type must be PrimitiveType or QualifiedSchemaId")
					}
				}
			}
		}
		if predicate.Evaluator == nil {
			continue
		}
		if scalar(child(predicate.Evaluator, "type")) != "cel" {
			continue
		}
		expressionNode := child(predicate.Evaluator, "expression")
		if problem := checkCELExpression(scalar(expressionNode), parameters); problem != "" {
			add(file, expressionNode, "CEL_INVALID", problem)
		}
	}
}
