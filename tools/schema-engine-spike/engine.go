package spike

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type Validator interface {
	Validate(any) error
}

type Compiled struct {
	Validator Validator
	Trace     *Trace
}

type Trace struct {
	RegistryLookups      []string `json:"registryLookups"`
	DeniedReferences     []string `json:"deniedReferences"`
	DeniedExternal       []string `json:"deniedExternalReferences"`
	ExternalAccesses     int      `json:"externalAccesses"`
	ResourcesPassedInMem int      `json:"resourcesPassedInMemory"`
}

type Registry struct {
	Documents map[string]any
	Trace     *Trace
}

func NewRegistry(documents map[string]any) *Registry {
	return &Registry{Documents: documents, Trace: &Trace{}}
}

func (r *Registry) Resolve(uri string) (any, []byte, error) {
	r.Trace.RegistryLookups = append(r.Trace.RegistryLookups, uri)
	doc, ok := r.Documents[uri]
	if !ok {
		parsed, _ := url.Parse(uri)
		if parsed != nil && isExternalScheme(strings.ToLower(parsed.Scheme)) {
			r.Trace.ExternalAccesses++
		}
		r.Trace.DeniedReferences = append(r.Trace.DeniedReferences, uri)
		if parsed != nil && (parsed.Scheme == "http" || parsed.Scheme == "https" || parsed.Scheme == "file") {
			r.Trace.DeniedExternal = append(r.Trace.DeniedExternal, uri)
		}
		return nil, nil, fmt.Errorf("resource denied by in-memory registry: %s", uri)
	}
	r.Trace.ResourcesPassedInMem++
	b, err := json.Marshal(doc)
	return doc, b, err
}

func isExternalScheme(scheme string) bool {
	switch scheme {
	case "http", "https", "file":
		return true
	default:
		return false
	}
}

type Candidate interface {
	Name() string
	Compile(documents map[string]any, rootURI string, assertFormat bool, regexMode string) (Compiled, error)
}
