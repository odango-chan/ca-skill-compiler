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
	RegistryLookups      []string         `json:"registryLookups"`
	DeniedReferences     []string         `json:"deniedReferences"`
	RegistryDenials      []RegistryDenial `json:"registryDenials"`
	DeniedExternal       []string         `json:"deniedExternalReferences"`
	ExternalAccesses     int              `json:"externalAccesses"`
	ResourcesPassedInMem int              `json:"resourcesPassedInMemory"`
}

const RegistryDeniedCode = "SPIKE_REGISTRY_DENIED"

type RegistryDenial struct {
	URI  string `json:"uri"`
	Code string `json:"code"`
}

type RegistryDeniedError struct{ URI string }

func (e RegistryDeniedError) Error() string {
	return fmt.Sprintf("%s: resource is not registered in the closed in-memory registry: %s", RegistryDeniedCode, e.URI)
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
		if parsed != nil && isExternalScheme(parsed.Scheme) {
			r.Trace.DeniedExternal = append(r.Trace.DeniedExternal, uri)
		}
		r.Trace.RegistryDenials = append(r.Trace.RegistryDenials, RegistryDenial{URI: uri, Code: RegistryDeniedCode})
		return nil, nil, RegistryDeniedError{URI: uri}
	}
	r.Trace.ResourcesPassedInMem++
	b, err := json.Marshal(doc)
	return doc, b, err
}

func isExternalScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
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
