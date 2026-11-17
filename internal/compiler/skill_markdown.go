//go:build legacy_v01

package compiler

import (
	"bytes"
	"regexp"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// parseSkillFrontmatter validates the standard Skill identity block while leaving
// the Markdown body as opaque author guidance.
func (a *analysis) parseSkillFrontmatter(skill *skillEntry) {
	filename := "skills/" + skill.name + "/SKILL.md"
	source := a.files[filename]
	if !bytes.HasPrefix(source, []byte("---\n")) {
		a.errorAt(filename, 1, "SKILL_METADATA_INVALID", "SKILL.md must begin with YAML frontmatter")
		return
	}
	end := bytes.Index(source[4:], []byte("\n---"))
	if end < 0 {
		a.errorAt(filename, 1, "SKILL_METADATA_INVALID", "SKILL.md frontmatter closing delimiter is missing")
		return
	}
	frontmatter := source[4 : 4+end]
	root, err := yaml.Parse(frontmatter)
	if err != nil || root == nil || unalias(root).Kind != yaml.MappingNode {
		line, _ := yaml.ParseErrorLocation(err)
		if line == 0 {
			line = 1
		}
		message := "SKILL.md frontmatter must be one YAML mapping"
		if err != nil {
			message = err.Error()
		}
		a.errorAt(filename, line+1, "SKILL_METADATA_INVALID", message)
		return
	}
	allowed := map[string]bool{"name": true, "description": true, "license": true, "compatibility": true, "allowed-tools": true, "metadata": true}
	for _, key := range mapKeys(root) {
		value := child(root, key)
		if !allowed[key] {
			a.errorAt(filename, value.Line+1, "SKILL_METADATA_UNSUPPORTED", "unsupported Skill frontmatter field: "+key)
		}
		if key != "metadata" && unalias(value).Kind != yaml.ScalarNode {
			a.errorAt(filename, value.Line+1, "SKILL_METADATA_INVALID", "Skill frontmatter "+key+" must be a scalar")
		}
	}
	name := child(root, "name")
	description := child(root, "description")
	if name == nil || name.Kind != yaml.ScalarNode || name.Tag != "!!str" || name.Value == "" || len(name.Value) > 64 || !skillNamePattern.MatchString(name.Value) {
		line := 2
		if name != nil {
			line = name.Line + 1
		}
		a.errorAt(filename, line, "SKILL_METADATA_INVALID", "Skill frontmatter name must be a lowercase hyphenated identifier of at most 64 characters")
	} else if name.Value != skill.name {
		a.errorAt(filename, name.Line+1, "SKILL_NAME_MISMATCH", "Skill frontmatter name must match its package directory")
	}
	if description == nil || description.Kind != yaml.ScalarNode || description.Tag != "!!str" || strings.TrimSpace(description.Value) == "" || len(description.Value) > 1024 {
		line := 2
		if description != nil {
			line = description.Line + 1
		}
		a.errorAt(filename, line, "SKILL_METADATA_INVALID", "Skill frontmatter description must be a nonempty string of at most 1024 characters")
	}
	if metadata := child(root, "metadata"); metadata != nil && unalias(metadata).Kind != yaml.MappingNode {
		a.errorAt(filename, metadata.Line+1, "SKILL_METADATA_INVALID", "Skill frontmatter metadata must be a mapping")
	}
	skill.ast.Metadata = SkillMetadataAST{Name: scalar(name), Description: scalar(description), Span: sourceSpan(filename, root)}
}
