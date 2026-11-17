package compiler

// WP10: Compiler-side Artifact assembly from validated v1alpha4 state.
//
// This assembles the deterministic virtual file tree (author source bytes
// unchanged, generated Script Output Schemas, one artifact manifest) through the
// frozen internal/artifact package. It is not yet wired into the successful
// Build path because Build intentionally remains fail-closed at the residual
// semantic boundary; the assembly itself is exercised directly by tests.

import (
	"fmt"
	"sort"

	"github.com/odango-chan/ca-skill-compiler/internal/artifact"
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

// CompilerVersion is the generator semantic version recorded in Artifact
// manifests. It is the Compiler product version and is independent of the
// frozen CA-SL language target (connectionagent.io/v1alpha4). The immutable build
// commit is not detected here, so the optional manifest commit field stays
// absent rather than being fabricated.
const CompilerVersion = "v1.0.0"

func generateV1Alpha4Artifact(req BuildRequest, skillName string, skill *V1Alpha4SkillAST, identity dependencyIdentityResult, ownerLinking ownerLinkingResult, scripts scriptSchemaSet) ([]artifact.File, error) {
	if skill == nil || skill.Contract == nil {
		return nil, fmt.Errorf("Skill %s has no Contract", skillName)
	}
	goal := scalar(skill.Contract.Goal)
	if goal == "" {
		return nil, fmt.Errorf("Skill %s has no Contract goal", skillName)
	}
	prefix := "skills/" + skillName + "/"
	sourceFiles := make([]artifact.File, 0)
	for _, file := range req.Files {
		if len(file.Path) <= len(prefix) || file.Path[:len(prefix)] != prefix {
			continue
		}
		sourceFiles = append(sourceFiles, artifact.File{Path: file.Path, Bytes: file.Bytes})
	}
	if len(sourceFiles) == 0 {
		return nil, fmt.Errorf("Skill %s has no source files", skillName)
	}

	scriptPaths := referencedScriptPaths(skill)
	generated := make([]artifact.File, 0, len(scriptPaths))
	for _, scriptPath := range scriptPaths {
		schema, ok := scripts.Outputs[prefix+scriptPath]
		if !ok {
			return nil, fmt.Errorf("no derived Output Schema for referenced Script %s", scriptPath)
		}
		file, err := artifact.ScriptOutputSchemaFile(skillName, scriptPath, schema)
		if err != nil {
			return nil, err
		}
		generated = append(generated, file)
	}

	closure := make([]artifact.DependencyClosureEntry, 0)
	exportsByAlias := map[string][]artifact.ExportRef{}
	order := make([]string, 0)
	for _, export := range ownerLinking.Exports {
		if export.DependencyAlias == "" || export.ExportID == "" {
			continue
		}
		if _, seen := exportsByAlias[export.DependencyAlias]; !seen {
			order = append(order, export.DependencyAlias)
		}
		exportsByAlias[export.DependencyAlias] = append(exportsByAlias[export.DependencyAlias], artifact.ExportRef{
			ExportKind: export.ExportKind,
			ExportID:   export.ExportID,
			Owner:      export.TrustedOwner,
		})
	}
	sort.Strings(order)
	for _, alias := range order {
		binding, ok := identity.Bindings[alias]
		if !ok || binding.StablePackageID == "" {
			return nil, fmt.Errorf("dependency closure alias %s has no trusted Stable Package ID", alias)
		}
		closure = append(closure, artifact.DependencyClosureEntry{
			DependencyID:    alias,
			StablePackageID: binding.StablePackageID,
			Exports:         exportsByAlias[alias],
		})
	}

	return artifact.Build(artifact.Input{
		SkillName:        skillName,
		Goal:             goal,
		GeneratorVersion: CompilerVersion,
		SourceFiles:      sourceFiles,
		GeneratedSchemas: generated,
		Dependency:       closure,
	})
}

// referencedScriptPaths returns the normalized script-relative paths referenced
// by Procedure operations and Predicate script evaluators, de-duplicated.
func referencedScriptPaths(skill *V1Alpha4SkillAST) []string {
	seen := map[string]bool{}
	paths := make([]string, 0)
	add := func(script string) {
		if !validLocalScriptPath(script) || seen[script] {
			return
		}
		seen[script] = true
		paths = append(paths, script)
	}
	if skill.Procedure != nil {
		for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
			node := skill.Procedure.ByID[nodeID]
			if node == nil || node.Operation == nil || node.Operation.Script == nil {
				continue
			}
			add(scalar(unalias(node.Operation.Script)))
		}
	}
	if skill.Predicates != nil {
		for _, predicateID := range sortedStringMapKeys(skill.Predicates.Definitions) {
			predicate := skill.Predicates.Definitions[predicateID]
			if predicate == nil || predicate.Evaluator == nil {
				continue
			}
			evaluator := unalias(predicate.Evaluator)
			if evaluator == nil || evaluator.Kind != yaml.MappingNode || scalar(child(evaluator, "type")) != "script" {
				continue
			}
			add(scalar(unalias(child(evaluator, "script"))))
		}
	}
	return paths
}
