package compiler

// Unified static semantic pipeline.
//
// CA-SL v1alpha5 inherits the released v1alpha4 normative semantics except
// where Appendix L adds or replaces them. Every inherited obligation therefore
// lives in this target-neutral pipeline: this file runs all inherited
// Compiler-owned static checks for one Skill package and returns the validated
// state the artifact assembler needs. The v1alpha5 Wait/Ingress passes compose
// with this pipeline on the same normalized parse result; they are not a second
// language implementation.

import (
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"strings"
)

// artifactInput is the validated per-Skill state the artifact assembler needs.
// It is produced only by validateSkillPackage, so assembly never re-derives
// identity, Owner linking or Script Schemas from unvalidated input.
type artifactInput struct {
	skill        *SkillAST
	identity     dependencyIdentityResult
	ownerLinking ownerLinkingResult
	scripts      scriptSchemaSet
}

// validationOptions carries the one cross-pass input the inherited pipeline
// cannot derive by itself.
type validationOptions struct {
	// additionalUsedAliases marks Contract dependency aliases that are consumed
	// by v1alpha5 Node-entry Wait constructs (the Wait consuming an
	// IngressCatalog export) so the inherited UNUSED_DEPENDENCY warning is not
	// raised for a genuinely used dependency.
	additionalUsedAliases map[string]bool
}

// validateSkillPackage runs the inherited Skill package, dependency identity,
// Owner, Tool, Capability, Schema, CEL, Predicate, Script, Procedure and
// assurance static checks for one Skill package.
//
// It returns the diagnostics emitted by result-producing checks plus the
// validated artifact input (identity, Owner linking and Script Schemas) for the
// artifact assembler. Structural diagnostics are emitted through errorAt so
// their source locations and ordering stay stable.
func validateSkillPackage(
	req BuildRequest,
	name string,
	files map[string][]byte,
	parsedFiles map[string]*yaml.Node,
	opts validationOptions,
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

	// Contract, Procedure, PredicateSet and local Schema checks share the exact
	// normalized nodes parsed once by the caller.
	for _, required := range []string{contractPath, procedurePath, predicatesPath} {
		if _, exists := parsedFiles[required]; !exists {
			errorAt(required, nil, "SOURCE_ROOT_INVALID", "Required CA-SL author source is missing")
			parsedFiles[required] = nil
		}
	}

	sourceAST := parseSkillAST(name, parsedFiles)
	identity := validateDependencyIdentityWithUsed(req, sourceAST, opts.additionalUsedAliases)
	// Derive the Python Script ABI and input Schemas once per Skill so Procedure
	// argument typing and Predicate Script checks share them.
	scriptSchemas := validateScriptABI(req, identity, sourceAST, files, procedurePath, predicatesPath, errorAt)
	contractAST := sourceAST.Contract
	input := artifactInput{skill: sourceAST, identity: identity, scripts: scriptSchemas}
	if contractAST != nil && contractAST.Raw != nil {
		contract := contractAST.Raw
		checkAllowedFields(contract, contractPath, contractFields, errorAt)
		if statement := scalar(contractAST.Goal); statement == "" {
			errorAt(contractPath, contract, "SOURCE_ROOT_INVALID", "Contract requires goal")
		} else if !validExportID(statement) {
			errorAt(contractPath, contractAST.Goal, "IDENTIFIER_INVALID", "Contract goal must satisfy the frozen GoalId (DottedId) grammar")
		}
		validateContractLimits(contractPath, contractAST, errorAt)
		if gate := contractAST.Precondition; gate != nil {
			if gate.Kind != yaml.ScalarNode || gate.Tag != "!!str" || !contractGatePattern.MatchString(gate.Value) {
				errorAt(contractPath, gate, "ENTRY_PRECONDITION_REF_INVALID", "precondition is one scalar node.precondition, not a list")
			}
		}
		// Every Skill-owned Schema root participates in the frozen G.4 allowlist
		// before any generic Schema engine may consume it.
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
				validateRequestSchemaRoot(schemaPath, parsedFiles[schemaPath], errorAt)
			}
		}
		if context := contractAST.ContextSchema; context != nil {
			ref := scalar(context)
			if !validQualifiedSchemaReference(ref) {
				errorAt(contractPath, context, "QUALIFIED_REFERENCE_INVALID",
					"Contract.contextSchema must be an exact Alias::absolute-Schema-URI reference")
			}
		}
		// Prove Alias -> snapshot digest -> trusted binding -> Stable Package ID
		// for the selected Skill only.
		diagnostics = append(diagnostics, identity.Diagnostics...)
		// Index the verified snapshots' exact callable/Schema exports and prove
		// trusted Owner attribution for the references the Skill uses.
		ownerLinking := validateOwnerLinking(req, sourceAST, identity)
		diagnostics = append(diagnostics, ownerLinking.Diagnostics...)
		// Bind exact callable membership and Effect against the trusted
		// deployment CapabilityContracts input. A Skill Grant is not proof of
		// authorization; missing trusted deployment input fails closed.
		capabilityLinking := validateCapabilityLinking(req, sourceAST, ownerLinking)
		diagnostics = append(diagnostics, capabilityLinking.Diagnostics...)
		// Resolve the exact Schema reference closure and enforce the
		// Request/Context root shape. Unresolvable references never guess.
		schemaDiagnostics := validateSchemaClosure(req, sourceAST, identity, ownerLinking)
		diagnostics = append(diagnostics, schemaDiagnostics...)
		// Reachability and global Precondition dominance.
		procedureDiagnostics := validateProcedureGraph(contractAST, sourceAST.Procedure, ownerLinking, contractPath, procedurePath)
		diagnostics = append(diagnostics, procedureDiagnostics...)
		sourcePathDiagnostics := validateSourcePathSchemas(req, sourceAST, identity, contractPath, procedurePath)
		diagnostics = append(diagnostics, sourcePathDiagnostics...)
		producerSourcePathDiagnostics := validateProducerSourcePaths(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, producerSourcePathDiagnostics...)
		argumentKeyDiagnostics := validateArgumentKeys(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, argumentKeyDiagnostics...)
		argumentValueDiagnostics := validateArgumentValueTypes(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, argumentValueDiagnostics...)
		scriptValueDiagnostics := validateScriptValueTypes(req, sourceAST, identity, ownerLinking, scriptSchemas.Inputs, procedurePath)
		diagnostics = append(diagnostics, scriptValueDiagnostics...)
		predicateArgumentDiagnostics := validatePredicateArgumentTypes(req, sourceAST, identity, ownerLinking, procedurePath)
		diagnostics = append(diagnostics, predicateArgumentDiagnostics...)
		mutationContractDiagnostics := validateMutationContracts(req, sourceAST, identity)
		diagnostics = append(diagnostics, mutationContractDiagnostics...)
		validateReconcileContracts(req, sourceAST, identity, ownerLinking, procedurePath, errorAt)
		validateRetryMutationFamilies(req, sourceAST, identity, ownerLinking, procedurePath, errorAt)
		validateRouteKeyEligibility(sourceAST, ownerLinking, procedurePath, errorAt)
		validateEvidenceDescriptors(req, sourceAST, identity, ownerLinking, procedurePath, errorAt)
		validatePredicateEvidence(req, sourceAST, ownerLinking, procedurePath, errorAt)
		validateMutationPositiveContracts(req, sourceAST, identity, ownerLinking, capabilityLinking, procedurePath, errorAt)
		input.ownerLinking = ownerLinking
	}
	if sourceAST.Predicates != nil && sourceAST.Predicates.Raw != nil {
		checkPredicates(predicatesPath, sourceAST.Predicates, errorAt)
		checkPredicateCEL(predicatesPath, sourceAST.Predicates, errorAt)
		validateEvidenceRequirements(predicatesPath, sourceAST.Predicates, errorAt)
		validateIndependenceGroups(predicatesPath, sourceAST.Predicates, errorAt)
		validatePredicateScriptEvaluators(sourceAST, files, predicatesPath, errorAt)
		validateInlineArgumentExpressions(sourceAST, procedurePath, errorAt)
		validateScriptArguments(sourceAST, scriptSchemas.Inputs, procedurePath, errorAt)
	}
	if sourceAST.Procedure != nil && sourceAST.Procedure.Raw != nil {
		checkProcedure(procedurePath, sourceAST.Procedure, contractAST, errorAt)
		validateProcedureStageRoutes(procedurePath, sourceAST.Procedure, errorAt)
		validateScriptPaths(sourceAST, files, procedurePath, errorAt)
		validateRetryContracts(sourceAST, procedurePath, errorAt)
		validateRetryReachability(sourceAST, procedurePath, errorAt)
		validateQualifiedCallableReferences(sourceAST, procedurePath, errorAt)
		validateSourcePaths(procedurePath, sourceAST, errorAt)
	}
	return diagnostics, input
}
