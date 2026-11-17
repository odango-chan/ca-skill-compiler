package compiler

// WP10 Build wiring: successful Build assembles the deterministic Artifact tree.
//
// Build remains feature-scoped fail-closed. Evidence Descriptor positive-path
// relations, complete Mutation/Reconcile semantics and full retry contract
// compatibility/path coverage are not complete; a Skill using those features is
// rejected rather than receiving a partial artifact. Static sub-checks still run
// first, and other fully validated Skills produce a deterministic output tree.

import (
	"sort"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

type artifactInput struct {
	skill        *V1Alpha4SkillAST
	identity     dependencyIdentityResult
	ownerLinking ownerLinkingResult
	scripts      scriptSchemaSet
}

// unsupportedSemanticsReason returns the first deterministic reason why a selected
// Skill crosses a feature-scoped Build gate. Static sub-checks are not a substitute
// for the complete positive-path obligations.
func unsupportedSemanticsReason(inputs map[string]artifactInput) string {
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		input := inputs[name]
		if input.skill == nil || input.skill.Contract == nil {
			return "Skill " + name + " has no validated Contract"
		}
		if input.skill.Predicates != nil {
			for _, predicateID := range sortedStringMapKeys(input.skill.Predicates.Definitions) {
				predicate := input.skill.Predicates.Definitions[predicateID]
				if predicate != nil && predicate.Evidence != nil {
					return "Evidence guarantee linking (WP7) is not implemented; compilation fails closed"
				}
			}
		}
		for _, export := range input.ownerLinking.Exports {
			if export.Effect == "MUTATION" {
				return "deep Mutation/Reconcile semantics (WP8) are not implemented; compilation fails closed"
			}
		}
		if input.skill.Procedure != nil {
			for _, nodeID := range sortedStringMapKeys(input.skill.Procedure.ByID) {
				node := input.skill.Procedure.ByID[nodeID]
				if node == nil || node.Operation == nil {
					continue
				}
				if child(node.Operation.Raw, "retryOf") != nil {
					return "full retry contract semantics (WP8) are not implemented; compilation fails closed"
				}
			}
		}
	}
	return ""
}

// assembleV1Alpha4SuccessResult builds the real successful BuildResult from the
// validated per-Skill state. Any assembly problem fails the whole build
// atomically (no partial files, empty closure).
func assembleV1Alpha4SuccessResult(req BuildRequest, names []string, inputs map[string]artifactInput, diagnostics []Diagnostic, add func(string, *yaml.Node, string, string)) (BuildResult, bool) {
	outputs := make([]File, 0)
	closure := map[string]bool{}
	for _, name := range names {
		input, ok := inputs[name]
		if !ok {
			add("", nil, "CAPABILITY_INPUT_UNTRUSTED", "fail-closed: no validated compiler state for selected Skill "+name)
			return failedBuildResult(diagnostics), false
		}
		files, err := generateV1Alpha4Artifact(req, name, input.skill, input.identity, input.ownerLinking, input.scripts)
		if err != nil {
			add("", nil, "CAPABILITY_INPUT_UNTRUSTED", "fail-closed: artifact assembly failed: "+err.Error())
			return failedBuildResult(diagnostics), false
		}
		for _, file := range files {
			outputs = append(outputs, File{Path: file.Path, Bytes: file.Bytes})
		}
		for _, alias := range input.identity.Closure {
			closure[alias] = true
		}
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	aliases := make([]string, 0, len(closure))
	for alias := range closure {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	sortDiagnostics(diagnostics)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	return BuildResult{
		APIVersion:        BuildResultAPIVersion,
		Success:           true,
		Diagnostics:       diagnostics,
		DependencyClosure: aliases,
		OutputFiles:       outputs,
	}, true
}
