// Package bindingspike experiments with deterministic semantic input binding.
// It is deliberately not wired into Compiler v0.1 or Runtime.
package bindingspike

import (
	"fmt"
	"sort"
)

// Field is a contract-declared semantic value, not a guessed property name.
// SemanticID must be supplied by the corresponding contract authority.
type Field struct {
	Name       string
	SemanticID string
	Type       string
	Nullable   bool
	Required   bool
}

// Producer is an eligible source with explicit semantic identity and control-flow
// evidence. AvailableAt lists the consumer nodes it definitely dominates.
// The spike does NOT construct or verify a Procedure CFG.
type Producer struct {
	ID          string
	Field       Field
	SourcePath  string
	AvailableAt []string
}

// Consumer is one operation invocation and its required input contract.
type Consumer struct {
	NodeID    string
	Operation string
	Inputs    []Field
}

type Binding struct {
	Target     string `json:"target"`
	Source     string `json:"source"`
	ProducerID string `json:"producerId"`
	SemanticID string `json:"semanticId"`
}

type Diagnostic struct {
	Code   string `json:"code"`
	Node   string `json:"node"`
	Input  string `json:"input"`
	Detail string `json:"detail"`
}

type Result struct {
	Bindings    []Binding    `json:"bindings"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Resolve fails closed: a required input is bound only if precisely one
// semantically matching, type-compatible producer is definitely available.
// Explicit mappings and complex JSON Schema implications are deliberately out
// of scope for this isolated automatic-binding prototype.
func Resolve(producers []Producer, consumers []Consumer) Result {
	var result Result
	for _, consumer := range consumers {
		for _, input := range consumer.Inputs {
			if input.SemanticID == "" || input.Type == "" {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{"INVALID_CONTRACT", consumer.NodeID, input.Name, "input lacks semantic ID or type"})
				continue
			}
			candidates := make([]Producer, 0)
			for _, p := range producers {
				if p.Field.SemanticID == input.SemanticID {
					candidates = append(candidates, p)
				}
			}
			if len(candidates) == 0 {
				if input.Required {
					result.Diagnostics = append(result.Diagnostics, Diagnostic{"BINDING_UNRESOLVED", consumer.NodeID, input.Name, "no semantic producer"})
				}
				continue
			}
			compatible := make([]Producer, 0)
			unreachable := 0
			for _, p := range candidates {
				if !compatibleType(p.Field, input) {
					continue
				}
				available := false
				for _, node := range p.AvailableAt {
					if node == consumer.NodeID {
						available = true
						break
					}
				}
				if !available {
					unreachable++
					continue
				}
				compatible = append(compatible, p)
			}
			switch len(compatible) {
			case 1:
				p := compatible[0]
				result.Bindings = append(result.Bindings, Binding{consumer.NodeID + "." + input.Name, p.SourcePath, p.ID, input.SemanticID})
			case 0:
				code, reason := "BINDING_TYPE_MISMATCH", "semantic producers violate type/nullability"
				if unreachable > 0 {
					code, reason = "BINDING_SOURCE_UNAVAILABLE", "producer not definitely available at consumer"
				}
				if input.Required {
					result.Diagnostics = append(result.Diagnostics, Diagnostic{code, consumer.NodeID, input.Name, reason})
				}
			default:
				result.Diagnostics = append(result.Diagnostics, Diagnostic{"BINDING_AMBIGUOUS", consumer.NodeID, input.Name, fmt.Sprintf("%d eligible semantic producers", len(compatible))})
			}
		}
	}
	sort.Slice(result.Bindings, func(i, j int) bool { return result.Bindings[i].Target < result.Bindings[j].Target })
	sort.Slice(result.Diagnostics, func(i, j int) bool {
		a, b := result.Diagnostics[i], result.Diagnostics[j]
		if a.Node != b.Node { return a.Node < b.Node }
		if a.Input != b.Input { return a.Input < b.Input }
		return a.Code < b.Code
	})
	return result
}

func compatibleType(source, target Field) bool {
	if source.Type != target.Type { return false } // conservative: no coercion or widening
	if source.Nullable && !target.Nullable { return false }
	return true
}
