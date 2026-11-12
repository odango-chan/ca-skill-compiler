package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike"
	js "github.com/santhosh-tekuri/jsonschema/v6"
)

type Candidate struct{}

func (Candidate) Name() string { return "santhosh-tekuri/jsonschema/v6@v6.0.3" }

func (Candidate) Compile(documents map[string]any, rootURI string, assertFormat bool, regexMode string) (spike.Compiled, error) {
	registry := spike.NewRegistry(documents)
	root, _, err := registry.Resolve(rootURI)
	if err != nil {
		return spike.Compiled{Trace: registry.Trace}, err
	}
	c := js.NewCompiler()
	c.DefaultDraft(js.Draft2020)
	c.UseLoader(registryLoader{registry})
	if assertFormat {
		c.AssertFormat()
	}
	if regexMode == "regexp2" {
		c.UseRegexpEngine(compileRegexp2)
	}
	if err := c.AddResource(rootURI, root); err != nil {
		return spike.Compiled{Trace: registry.Trace}, err
	}
	schema, err := c.Compile(rootURI)
	if err != nil {
		return spike.Compiled{Trace: registry.Trace}, err
	}
	return spike.Compiled{Validator: validator{schema}, Trace: registry.Trace}, nil
}

type registryLoader struct{ registry *spike.Registry }

func (l registryLoader) Load(uri string) (any, error) {
	doc, _, err := l.registry.Resolve(uri)
	return doc, err
}

type validator struct{ schema *js.Schema }

func (v validator) Validate(value any) error { return v.schema.Validate(value) }

type regexp2Pattern struct {
	re      *regexp2.Regexp
	pattern string
}

func (r regexp2Pattern) String() string { return r.pattern }
func (r regexp2Pattern) MatchString(s string) bool {
	ok, err := r.re.MatchString(s)
	return err == nil && ok
}

func compileRegexp2(pattern string) (js.Regexp, error) {
	// regexp2 understands the short Unicode general-category name but not the
	// ECMAScript long alias used by the official suite. This explicit alias
	// normalization preserves the defined property set; it is deliberately
	// narrow and is not a general ECMA-262 translation layer.
	normalized := strings.ReplaceAll(pattern, `\p{Letter}`, `\p{L}`)
	normalized = strings.ReplaceAll(normalized, `\P{Letter}`, `\P{L}`)
	re, err := regexp2.Compile(normalized, regexp2.ECMAScript|regexp2.Unicode)
	if err != nil {
		return nil, fmt.Errorf("regexp2: %w", err)
	}
	return regexp2Pattern{re: re, pattern: pattern}, nil
}

type re2Pattern struct{ re *regexp.Regexp }

func (r re2Pattern) String() string            { return r.re.String() }
func (r re2Pattern) MatchString(s string) bool { return r.re.MatchString(s) }

func CompileRE2(pattern string) (js.Regexp, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re2Pattern{re}, nil
}

func ExactJSON(doc any) ([]byte, error) { return json.Marshal(doc) }
