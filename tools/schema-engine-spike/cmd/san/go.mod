module github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike/cmd/san

go 1.23.0

require (
	github.com/dlclark/regexp2 v1.11.0
	github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike v0.0.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
)

require (
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/odango-chan/ca-skill-compiler/tools/schema-engine-spike => ../..
