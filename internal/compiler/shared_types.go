package compiler

// Shared public request/response and source-location types used by the active
// v1alpha5 Compiler Core.

import yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"

// File carries original bytes and a logical path. The containing BuildRequest
// field defines whether the path is Skill-rooted or snapshot-relative.
type File struct {
	Path  string `json:"path"`
	Bytes []byte `json:"bytes"`
}

// SourceSpan is the source location retained by the parsed Skill AST.
type SourceSpan struct {
	Path   string
	Line   int
	Column int
}

func sourceSpan(path string, node *yaml.Node) SourceSpan {
	span := SourceSpan{Path: path}
	if node != nil {
		node = unalias(node)
		span.Line, span.Column = node.Line, node.Column
	}
	return span
}
