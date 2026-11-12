package spike

type RegexResult struct {
	Candidate      string `json:"candidate"`
	RegexMode      string `json:"regexMode"`
	ID             string `json:"id"`
	Compiled       bool   `json:"compiled"`
	MatchesEngine  bool   `json:"matchesEngine"`
	ExpectedJS     bool   `json:"expectedEcma262NoFlags"`
	ConformsSample bool   `json:"conformsSample"`
	Error          string `json:"error,omitempty"`
}

func RunRegex(c Candidate, regexMode string) []RegexResult {
	tests := []struct {
		id, pattern, value string
		expect             bool
	}{
		{"unicode-literal", "^café+$", "caféé", true},
		{"unicode-property-escape", `^\p{L}+$`, "é", true},
		{"unicode-long-category-alias", `^\p{Letter}+$`, "é", true},
		{"astral-dot-unicode", `^.$`, "😀", true},
		{"lookahead", `^(?!x).+$`, "abc", true},
		{"lookbehind", `(?<=pre)fix`, "prefix", true},
		{"backreference", `^(a+)\1$`, "aaaa", true},
	}
	results := make([]RegexResult, 0, len(tests)+1)
	for _, test := range tests {
		root := "https://spike.invalid/regex/" + test.id + ".json"
		docs := map[string]any{root: map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"type":    "string",
			"pattern": test.pattern,
		}}
		compiled, err := c.Compile(docs, root, false, regexMode)
		result := RegexResult{Candidate: c.Name(), RegexMode: regexMode, ID: test.id, ExpectedJS: test.expect}
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		result.Compiled = true
		err = compiled.Validator.Validate(test.value)
		result.MatchesEngine = err == nil
		result.ConformsSample = result.MatchesEngine == test.expect
		results = append(results, result)
	}

	root := "https://spike.invalid/regex/pattern-properties.json"
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"patternProperties":    map[string]any{`^(?!x).*$`: map[string]any{"type": "integer"}},
		"additionalProperties": false,
	}
	compiled, err := c.Compile(map[string]any{root: schema}, root, false, regexMode)
	r := RegexResult{Candidate: c.Name(), RegexMode: regexMode, ID: "patternProperties-lookaround", ExpectedJS: true}
	if err != nil {
		r.Error = err.Error()
		results = append(results, r)
		return results
	}
	r.Compiled = true
	r.MatchesEngine = compiled.Validator.Validate(map[string]any{"a": 1}) == nil && compiled.Validator.Validate(map[string]any{"x": 1}) != nil
	r.ConformsSample = r.MatchesEngine
	results = append(results, r)
	return results
}
