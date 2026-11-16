package compiler

// E114 conformance subset: package path variants beyond the traversal/duplicate
// cases already covered by the BuildRequest path tests.

import "testing"

func TestE114PackagePathVariants(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*BuildRequest)
	}{
		{"absolute path", func(req *BuildRequest) { req.Files[0].Path = "/abs/SKILL.md" }},
		{"backslash path", func(req *BuildRequest) { req.Files[0].Path = `skills\demo\SKILL.md` }},
		{"case-folded duplicate", func(req *BuildRequest) {
			req.Files = append(req.Files, File{Path: "skills/demo/skill.md", Bytes: []byte("# other\n")})
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req := validEnvelopeRequest()
			testCase.mutate(&req)
			if got := Check(req); !hasCode(got, "PACKAGE_PATH_INVALID") {
				t.Fatalf("variant %q accepted: %+v", testCase.name, got)
			}
		})
	}
}
