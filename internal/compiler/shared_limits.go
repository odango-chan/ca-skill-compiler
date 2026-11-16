package compiler

// Shared limits, diagnostics-size guards and sorted-key helpers used by the
// active v1alpha4 Core (historically defined in the legacy analysis pipeline).

import (
	"sort"
	"unicode/utf8"
)

const (
	maxFiles    = 512
	maxFileSize = 2 << 20
	maxTotal    = 16 << 20
)

func limitDiagnosticPath(filename string) string {
	if utf8.RuneCountInString(filename) <= 240 {
		return filename
	}
	runes := []rune(filename)
	return string(runes[:240])
}

func limitDiagnosticMessage(message string) string {
	if utf8.RuneCountInString(message) <= 8192 {
		return message
	}
	runes := []rune(message)
	return string(runes[:8192])
}

func sortedStringMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
