package compiler

// WP10 Build wiring: successful Build assembles the deterministic Artifact tree.
//
// The released Mutation/Reconcile/retry positive-path obligations and the
// Evidence Descriptor linking obligations are enforced by the inherited static
// pipeline: the earlier feature-scoped fail-closed gates are removed. Remaining
// limitations (the §17.1 Host-recovery dual-Producer static slice and the
// deferred Ingress response-contract cases) are recorded in
// docs/known-limitations-v1alpha5.md.

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
// Skill cannot be assembled (currently only a missing validated Contract state).
// Semantic obligations are enforced by the static pipeline before assembly.
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
