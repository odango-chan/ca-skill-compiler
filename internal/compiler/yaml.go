//go:build legacy_v01

package compiler

import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

// parseYAML is the single source parser used by the legacy analysis helpers
// and the v1alpha4 Core. Both consume the same normalized AST adapter.
func (a *analysis) parseYAML(filename string) *yaml.Node {
	root, err := yaml.Parse(a.files[filename])
	if err != nil {
		line, _ := yaml.ParseErrorLocation(err)
		if line == 0 {
			line = 1
		}
		code := "YAML_PARSE"
		if yaml.IsRootError(err) {
			code = "YAML_ROOT"
		}
		if yaml.IsLimitError(err) {
			code = "YAML_LIMIT"
		}
		a.errorAt(filename, line, code, err.Error())
		return nil
	}
	return root
}
