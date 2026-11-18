package compiler

// Inherited static semantic pipeline.
//
// CA-SL v1alpha5 inherits the released v1alpha4 normative semantics except
// where Appendix L adds or replaces them. The obligations therefore live in one
// target-neutral static pipeline instead of two divergent implementations: this
// file runs every inherited Compiler-owned static check for one Skill package
// and returns the validated state the target artifact assembler needs.
//
// The v1alpha5 target enables the Appendix L node-entry Wait allowance; the
// historical v1alpha4 target keeps it rejected with the frozen v1alpha4
// diagnostic identity. Everything else is shared.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"strings"
)

// inheritedValidationOptions selects the target-specific parts of the inherited
// semantic pipeline.
type inheritedValidationOptions struct {
	// allowWait admits the Appendix L node-entry Wait declaration. It is only
	// true for the v1alpha5 language target.
	allowWait bool
	// missingSourceMessage is the target-specific diagnostic message for a
	// required author source that is absent from the BuildRequest.
	missingSourceMessage string
}

// validateInheritedSkillPackage runs the inherited Skill package, dependency
// identity, Owner, Tool, Capability, Schema, CEL, Predicate, Script, Procedure
// and assurance static checks for one Skill package.
//
// It returns the diagnostics emitted by result-producing checks plus the
// validated artifact input (identity, Owner linking and Script Schemas) for the
// target-specific artifact assembler. Structural diagnostics are emitted through
// errorAt exactly as the released v1alpha4 pipeline did, so existing diagnostic
// ordering and source locations are preserved.
func validateInheritedSkillPackage(
	req BuildRequest,
	name string,
	files map[string][]byte,
	parsedFiles map[string]*yaml.Node,
	opts inheritedValidationOptions,
	errorAt func(string, *yaml.Node, string, string),
) ([]Diagnostic, artifactInput) {
	var diagnostics []Diagnostic
	base := "skills/" + name + "/"
	contractPath := base + "assurance/contract.yaml"
	procedurePath := base + "assurance/procedure.yaml"
	predicatesPath := base + "assurance/predicates.yaml"
	if files[base+"SKILL.md"] == nil {
		errorAt(base+"SKILL.md", nil, "SOURCE_ROOT_INVALID", "SKILL.md missing")
	}

	// Parse every YAML author file exactly once. Contract, Procedure,
	// PredicateSet and local Schema checks share these normalized nodes.
	if parsedFiles == nil {
		parsedFiles = make(map[string]*yaml.Node)
		for _, filename := range sortedStringMapKeys(files) {
			if strings.HasPrefix(filename, base) && (strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
				parsedFiles[filename] = v4ParseYAML(filename, files[filename], errorAt)
			}
		}
	}
	for _, required := range []string{contractPath, procedurePath, predicatesPath} {
		if _, exists := parsedFiles[required]; !exists {
			errorAt(required, nil, "SOURCE_ROOT_INVALID", opts.missingSourceMessage)
			parsedFiles[required] = nil
		}
	}

	sourceAST := parseV1Alpha4SkillAST(name, parsedFiles)
	identity := validateV1Alpha4DependencyIdentity(req, sourceAST)
	// WP9: derive the Python Script ABI and input Schemas once per Skill so
	// Procedure argument typing and Predicate Script checks share them.
	scriptSchemas := validateV1Alpha4ScriptABI(req, identity, sourceAST, files, procedurePath, predicatesPath, errorAt)
	contractAST := sourceAST.Contract
	input := artifactInput{skill: sourceAST, identity: identity, scripts: scriptSchemas}
	if contractAST != nil && contractAST.Raw != nil {
		contract := contractAST.Raw
		v4Allowed(contract, contractPath, v4ContractFields, errorAt)
		if statement := scalar(contractAST.Goal); statement == "" {
			errorAt(contractPath, contract, "SOURCE_ROOT_INVALID", "Contract requires goal")
		} else if !validExportID(statement) {
			errorAt(contractPath, contractAST.Goal, "IDENTIFIER_INVALID", "Contract goal must satisfy the frozen GoalId (DottedId) grammar")
		}
		validateV1Alpha4ContractLimits(contractPath, contractAST, errorAt)
		if gate := contractAST.Precondition; gate != nil {
			if gate.Kind != yaml.ScalarNode || gate.Tag != "!!str" || !v4ContractGate.MatchString(gate.Value) {
				errorAt(contractPath, gate, "ENTRY_PRECONDITION_REF_INVALID", "precondition is one scalar node.precondition, not a list")
			}
		}
		// WP4A: every Skill-owned Schema root participates in the frozen G.4
		// allowlist before any generic Schema engine may consume it.
		for _, filename := range sortedStringMapKeys(parsedFiles) {
			if !strings.HasPrefix(filename, base+"schemas/") || !(strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml")) {
				continue
			}
			if schemaRoot := parsedFiles[filename]; schemaRoot != nil {
				for _, issue := range validateG4Schema(schemaRoot) {
					errorAt(filename, issue.node, "SCHEMA_INVALID", issue.message)
				}
			}
		}
		if request := contractAST.RequestSchema; request != nil {
			ref := scalar(request)
			schemaPath := base + ref
			if !strings.HasPrefix(ref, "schemas/") || files[schemaPath] == nil || parsedFiles[schemaPath] == nil {
				errorAt(contractPath, request, "REQUEST_SCHEMA_MISSING", "Skill-owned request Schema is missing")
			} else {
				validateV1Alpha4RequestSchemaRoot(schemaPath, parsedFiles[schemaPath], errorAt)
			}
		}
		if context := contractAST.ContextSchema; context != nil {
			ref := scalar(context)
			if !validQualifiedSchemaReference(ref) {
				errorAt(contractPath, context, "QUALIFIED_REFERENCE_INVALID",
					"Contract.contextSchema must be an exact Alias::absolute-Schema-URI reference")
			}
		}
		// WP3A: prove Alias -> snapshot digest -> trusted binding -> Stable
		// Package ID for the selected Skill only.
		diagnostics = append(diagnostics, identity.Diagnostics...)
		// WP3B: index the verified snapshots' exact callable/Schema exports and
		// prove trusted Owner attribution for the references the Skill uses.
		ownerLinking := validateV1Alpha4OwnerLinking(req, sourceAST, identity)
		diagnostics = append(diagnostics, ownerLinking.Diagnostics...)
		// WP3C: bind exact callable membership and Effect against the trusted
		// deployment CapabilityContracts input. A Skill Grant is not proof of
		// authorization; missing trusted deployment input fails closed.
		capabilityLinking := validateV1Alpha4CapabilityLinking(req, sourceAST, ownerLinking)
		diagnostics = append(diagnostics, capabilityLinking.Diagnostics...)
		// WP4B: resolve the exact Schema reference closure and enforce the
		// Request/Context root shape. Unresolvable references never guess.
		schemaDiagnostics := validateV1Alpha4SchemaClosure(req, sourceAST, identity, ownerLinking)
		diagnostics = append(diagnostics, schemaDiagnostics...)
		// WP5: reachability and global Precondition dominance.
		procedureDiagnostics := validateV1Alpha4ProcedureGraph(contractAST, sourceAST.Procedure, ownerLinking, contractPath, procedurePath)
		diagnostics = append(diagnostics, procedureDiagnostics...)
		sourcePathDiagnostics := validateV1Alpha4SourcePathSchemas(req, sourceAST, identity, contractPath, procedurePath)
		diagnostics = append(diagnostics, sourcePathDiagnostics...)
		producerSourcePathDiagnostics := validateV1Alpha4ProducerSourcePaths(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, producerSourcePathDiagnostics...)
		argumentKeyDiagnostics := validateV1Alpha4ArgumentKeys(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, argumentKeyDiagnostics...)
		argumentValueDiagnostics := validateV1Alpha4ArgumentValueTypes(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, argumentValueDiagnostics...)
		scriptValueDiagnostics := validateV1Alpha4ScriptValueTypes(req, sourceAST, identity, ownerLinking, scriptSchemas.Inputs, procedurePath)
		diagnostics = append(diagnostics, scriptValueDiagnostics...)
		predicateArgumentDiagnostics := validateV1Alpha4PredicateArgumentTypes(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, predicateArgumentDiagnostics...)
		mutationContractDiagnostics := validateV1Alpha4MutationContracts(req, sourceAST, identity)
		diagnostics = append(diagnostics, mutationContractDiagnostics...)
		validateV1Alpha4ReconcileContracts(req, sourceAST, identity, ownerLinking, procedurePath, errorAt)
		validateV1Alpha4RetryMutationFamilies(sourceAST, ownerLinking, procedurePath, errorAt)
		validateV1Alpha4RouteKeyEligibility(sourceAST, ownerLinking, procedurePath, errorAt)
		validateV1Alpha4PredicateEvidence(req, sourceAST, ownerLinking, procedurePath, errorAt)
		input.ownerLinking = ownerLinking
	}
	if sourceAST.Predicates != nil && sourceAST.Predicates.Raw != nil {
		v4CheckPredicates(predicatesPath, sourceAST.Predicates, errorAt)
		v4CheckPredicateCEL(predicatesPath, sourceAST.Predicates, errorAt)
		validateV1Alpha4EvidenceRequirements(predicatesPath, sourceAST.Predicates, errorAt)
		validateV1Alpha4IndependenceGroups(predicatesPath, sourceAST.Predicates, errorAt)
		validateV1Alpha4PredicateScriptEvaluators(sourceAST, files, predicatesPath, errorAt)
		validateV1Alpha4InlineArgumentExpressions(sourceAST, procedurePath, errorAt)
		validateV1Alpha4ScriptArguments(sourceAST, scriptSchemas.Inputs, procedurePath, errorAt)
	}
	if sourceAST.Procedure != nil && sourceAST.Procedure.Raw != nil {
		v4CheckProcedure(procedurePath, sourceAST.Procedure, contractAST, errorAt, opts.allowWait)
		validateV1Alpha4ProcedureStageRoutes(procedurePath, sourceAST.Procedure, errorAt)
		validateV1Alpha4ScriptPaths(sourceAST, files, procedurePath, errorAt)
		validateV1Alpha4RetryContracts(sourceAST, procedurePath, errorAt)
		validateV1Alpha4RetryReachability(sourceAST, procedurePath, errorAt)
		validateV1Alpha4QualifiedCallableReferences(sourceAST, procedurePath, errorAt)
		validateV1Alpha4SourcePaths(procedurePath, sourceAST, errorAt)
	}
	return diagnostics, input
}
