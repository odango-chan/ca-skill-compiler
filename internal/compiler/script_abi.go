package compiler

// WP9 Python Script ABI: static derivation of the frozen Script interface.
//
// A Script is `main(input: InputTypedDict) -> OutputTypedDict` with strict,
// statically resolvable TypedDict annotations (CA-SL §32.1). The Compiler derives
// the input and output Schema values from those annotations. Author Python is
// parsed through the internal pythoncst adapter and is never executed, imported
// or evaluated; function bodies are not analysed for types or values.
//
// Predicate Scripts receive the Host-injected read-only `PredicateInput` type
// and must use delayed annotations (CA-SL §32.2). The generated input Schema for
// a Predicate Script depends on the owning Predicate parameters and is derived
// separately once predicate linking provides it.

import (
	"fmt"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/pythoncst"
)

type scriptABI struct {
	InputName    string
	OutputName   string
	InputSchema  map[string]any
	OutputSchema map[string]any
}

type pythonModule struct {
	Classes    map[string]pyClass
	HasFuture  bool
	Main       *pythoncst.Node
	MainInput  *pythoncst.Node
	MainOutput *pythoncst.Node
}

func parseScriptModule(source []byte) (*pythonModule, error) {
	root, err := pythoncst.Parse(source)
	if err != nil {
		return nil, err
	}
	module := &pythonModule{Classes: map[string]pyClass{}}
	for _, statement := range root.Children {
		switch statement.Kind {
		case "future_import_statement":
			if strings.Contains(statement.Text(source), "annotations") {
				module.HasFuture = true
			}
		case "import_statement", "import_from_statement", "expression_statement":
			// Imports are declarations; module docstrings are inert. Other
			// executable module statements are not part of the Script interface.
			if statement.Kind == "expression_statement" && !isDocstring(statement) {
				return nil, fmt.Errorf("unsupported module-level executable statement")
			}
		case "class_definition":
			name, class, ok, err := parseTypedDictClass(statement, source)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue // helper classes are inert; only TypedDicts define the ABI
			}
			if _, exists := module.Classes[name]; exists {
				return nil, fmt.Errorf("duplicate TypedDict declaration %s", name)
			}
			module.Classes[name] = class
		case "function_definition":
			if module.Main != nil {
				return nil, fmt.Errorf("Script must declare exactly one main function")
			}
			module.Main = statement
		default:
			return nil, fmt.Errorf("unsupported module-level statement %s", statement.Kind)
		}
	}
	if module.Main == nil {
		return nil, fmt.Errorf("Script must declare main(input: InputTypedDict) -> OutputTypedDict")
	}
	name := module.Main.Field("name")
	if name == nil || name.Text(source) != "main" {
		return nil, fmt.Errorf("the only module-level function must be named main")
	}
	body := module.Main.Field("body")
	if body == nil {
		return nil, fmt.Errorf("main must have a body")
	}
	parameters := module.Main.Field("parameters")
	if parameters == nil || len(parameters.Children) != 1 || parameters.Children[0].Kind != "typed_parameter" {
		return nil, fmt.Errorf("main must take exactly one typed parameter")
	}
	parameter := parameters.Children[0]
	parameterName := ""
	if first := parameter.Children[0]; first != nil && first.Kind == "identifier" {
		parameterName = first.Text(source)
	}
	if parameterName == "" {
		return nil, fmt.Errorf("main parameter must be a plain identifier")
	}
	module.MainInput = parameter.Field("type")
	if module.MainInput == nil {
		return nil, fmt.Errorf("main input parameter must be annotated with its TypedDict type")
	}
	module.MainOutput = module.Main.Field("return_type")
	if module.MainOutput == nil {
		return nil, fmt.Errorf("main must annotate its strict Output TypedDict return type")
	}
	return module, nil
}

func isDocstring(statement *pythoncst.Node) bool {
	if len(statement.Children) != 1 {
		return false
	}
	child := statement.Children[0]
	return child.Kind == "string" || child.Kind == "concatenated_string"
}

func parseTypedDictClass(classNode *pythoncst.Node, source []byte) (string, pyClass, bool, error) {
	nameNode := classNode.Field("name")
	if nameNode == nil || nameNode.Kind != "identifier" {
		return "", pyClass{}, false, fmt.Errorf("class declaration without a name")
	}
	name := nameNode.Text(source)
	superclasses := classNode.Field("superclasses")
	if superclasses == nil {
		return name, pyClass{}, false, nil
	}
	if len(superclasses.Children) != 1 || superclasses.Children[0].Kind != "identifier" || superclasses.Children[0].Text(source) != "TypedDict" {
		if classDeclaresTypedDict(superclasses, source) {
			return "", pyClass{}, false, fmt.Errorf("TypedDict class %s must not use total= or multiple bases", name)
		}
		return name, pyClass{}, false, nil
	}
	body := classNode.Field("body")
	if body == nil {
		return "", pyClass{}, false, fmt.Errorf("TypedDict class %s must have a body", name)
	}
	class := pyClass{}
	seen := map[string]bool{}
	for _, statement := range body.Children {
		if statement.Kind == "expression_statement" && isDocstring(statement) {
			continue
		}
		if statement.Kind == "pass_statement" {
			continue
		}
		assignment := singleChildKind(statement, "assignment")
		if assignment == nil {
			return "", pyClass{}, false, fmt.Errorf("unsupported TypedDict body statement in %s: %s", name, strings.TrimSpace(statement.Text(source)))
		}
		fieldName := assignment.Field("left")
		fieldType := assignment.Field("type")
		if fieldName == nil || fieldName.Kind != "identifier" || fieldType == nil {
			return "", pyClass{}, false, fmt.Errorf("TypedDict field in %s must be a plain `name: type` annotation", name)
		}
		if assignment.Field("right") != nil {
			return "", pyClass{}, false, fmt.Errorf("TypedDict field %s.%s must not have a default value", name, fieldName.Text(source))
		}
		key := fieldName.Text(source)
		if seen[key] {
			return "", pyClass{}, false, fmt.Errorf("duplicate TypedDict field %s.%s", name, key)
		}
		seen[key] = true
		class.fields = append(class.fields, pyField{name: key, typ: strings.TrimSpace(fieldType.Text(source))})
	}
	return name, class, true, nil
}

func classDeclaresTypedDict(superclasses *pythoncst.Node, source []byte) bool {
	for _, child := range superclasses.Children {
		if child.Kind == "keyword_argument" && strings.Contains(child.Text(source), "TypedDict") {
			return false
		}
		if child.Kind == "identifier" && child.Text(source) == "TypedDict" {
			return true
		}
	}
	return false
}

func singleChildKind(node *pythoncst.Node, kind string) *pythoncst.Node {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child.Kind == kind {
			return child
		}
	}
	return nil
}

func deriveProcedureScriptABI(source []byte) (*scriptABI, error) {
	module, err := parseScriptModule(source)
	if err != nil {
		return nil, err
	}
	inputName := strings.TrimSpace(module.MainInput.Text(source))
	outputName := strings.TrimSpace(module.MainOutput.Text(source))
	if _, ok := module.Classes[inputName]; !ok {
		return nil, fmt.Errorf("main input type %s is not a local TypedDict", inputName)
	}
	if _, ok := module.Classes[outputName]; !ok {
		return nil, fmt.Errorf("main return type %s is not a local TypedDict", outputName)
	}
	if err := rejectNotRequired(module.Classes[outputName], outputName); err != nil {
		return nil, err
	}
	inputSchema, err := typedDictSchema(inputName, module.Classes, map[string]bool{})
	if err != nil {
		return nil, fmt.Errorf("input %s: %w", inputName, err)
	}
	outputSchema, err := typedDictSchema(outputName, module.Classes, map[string]bool{})
	if err != nil {
		return nil, fmt.Errorf("output %s: %w", outputName, err)
	}
	return &scriptABI{InputName: inputName, OutputName: outputName, InputSchema: inputSchema, OutputSchema: outputSchema}, nil
}

func derivePredicateScriptABI(source []byte) (*scriptABI, error) {
	module, err := parseScriptModule(source)
	if err != nil {
		return nil, err
	}
	if !module.HasFuture {
		return nil, fmt.Errorf("Predicate Script must use delayed annotations (`from __future__ import annotations`)")
	}
	inputName := strings.TrimSpace(module.MainInput.Text(source))
	if inputName != "PredicateInput" {
		return nil, fmt.Errorf("Predicate Script main input type must be the injected PredicateInput")
	}
	outputName := strings.TrimSpace(module.MainOutput.Text(source))
	if _, ok := module.Classes[outputName]; !ok {
		return nil, fmt.Errorf("main return type %s is not a local TypedDict", outputName)
	}
	if err := rejectNotRequired(module.Classes[outputName], outputName); err != nil {
		return nil, err
	}
	outputSchema, err := typedDictSchema(outputName, module.Classes, map[string]bool{})
	if err != nil {
		return nil, fmt.Errorf("output %s: %w", outputName, err)
	}
	return &scriptABI{InputName: inputName, OutputName: outputName, OutputSchema: outputSchema}, nil
}

func rejectNotRequired(class pyClass, name string) error {
	for _, field := range class.fields {
		if strings.HasPrefix(strings.TrimSpace(field.typ), "NotRequired[") {
			return fmt.Errorf("strict output TypedDict %s must not declare NotRequired field %s", name, field.name)
		}
	}
	return nil
}
