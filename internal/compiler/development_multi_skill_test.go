package compiler

// Multi-Skill Dependency Lock tests.
//
// One Dependency Lock may serve several selected Skills. Each Skill declares only
// its own roots; the required Root set is the union of those declarations. It is
// never required that one Skill declare another Skill's dependencies.

import (
	"sort"
	"strings"
	"testing"
)

const developmentSecondSkill = "verify-context-only"

// addDevelopmentSecondSkill appends a minimal second Skill that declares only the
// Host Context root. The Skill compiles to a single Predicate-only Node, so it
// needs no Callable dependency at all.
func addDevelopmentSecondSkill(t *testing.T, request DevelopmentBuildRequest, dependencies string) DevelopmentBuildRequest {
	t.Helper()
	request = cloneDevelopmentRequest(t, request)
	base := "skills/" + developmentSecondSkill + "/"
	request.Files = append(request.Files,
		File{Path: base + "SKILL.md", Bytes: []byte("---\nname: " + developmentSecondSkill + "\ndescription: Verify a Host Context only\n---\n")},
		File{Path: base + "assurance/contract.yaml", Bytes: []byte("goal: ue.context_only\n" +
			"contextSchema: ca-context::urn:example:ca:invocation-context:v1\n" +
			"dependencies:\n" + dependencies)},
		File{Path: base + "assurance/predicates.yaml", Bytes: []byte("predicates:\n" +
			"  example.context-present:\n" +
			"    parameters: []\n" +
			"    evaluator:\n" +
			"      type: cel\n" +
			"      expression: \"true\"\n")},
		File{Path: base + "assurance/procedure.yaml", Bytes: []byte("start: check-context\n" +
			"nodes:\n" +
			"  check-context:\n" +
			"    postcondition:\n" +
			"      predicates:\n" +
			"        - predicate: example.context-present\n" +
			"          arguments: []\n" +
			"      routes:\n" +
			"        PASS: SUCCEEDED\n" +
			"        FAIL: FAILED\n" +
			"        UNKNOWN: UNRESOLVED\n")},
	)
	sort.Slice(request.Files, func(i, j int) bool { return request.Files[i].Path < request.Files[j].Path })
	request.Targets = append(request.Targets, developmentSecondSkill)
	sort.Strings(request.Targets)
	return request
}

// replaceDevelopmentContract rewrites a Skill Contract file.
func replaceDevelopmentContract(t *testing.T, request DevelopmentBuildRequest, skillName, contract string) DevelopmentBuildRequest {
	t.Helper()
	return replaceDevelopmentSkillFileBytes(t, request, "skills/"+skillName+"/assurance/contract.yaml", []byte(contract))
}

// TestDevelopmentMultiSkillSharedLockPasses proves the positive case: two Skills
// declare different roots and share one Lock. Neither Skill has to declare the
// other's dependency, and the union covers every Lock root.
func TestDevelopmentMultiSkillSharedLockPasses(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:host-context\n")
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("two Skills sharing one Lock must build: %#v", result.Diagnostics)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
	// The Artifact carries one compiled Procedure per selected Skill.
	paths := []string{}
	for _, file := range result.OutputFiles {
		if strings.HasSuffix(file.Path, "/compiled/procedure.json") {
			paths = append(paths, file.Path)
		}
	}
	sort.Strings(paths)
	want := []string{
		"skills/" + developmentSecondSkill + "/compiled/procedure.json",
		"skills/verify-ue-registration/compiled/procedure.json",
	}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("compiled Procedure paths = %v, want %v", paths, want)
	}
}

// TestDevelopmentMultiSkillMissingRootFails proves both Skills' roots are
// required: dropping the tools root fails even though the second Skill never
// declares it.
func TestDevelopmentMultiSkillMissingRootFails(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:host-context\n")
	request = cloneDevelopmentRequest(t, request)
	roots := request.Lock.Roots[:0]
	for _, root := range request.Lock.Roots {
		if root.Alias == "ca" {
			continue
		}
		roots = append(roots, root)
	}
	request.Lock.Roots = roots
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("a Lock missing a root declared by a selected Skill must fail")
	}
	if !hasDevelopmentCode(result.Diagnostics, developmentLockInvalid) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, developmentLockInvalid)
	}
}

// TestDevelopmentMultiSkillWrongPackageFails proves an alias must resolve to the
// exact Stable Package ID the Skill declares.
func TestDevelopmentMultiSkillWrongPackageFails(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:tools\n")
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("an alias bound to the wrong Stable Package ID must fail")
	}
	if !hasDevelopmentCode(result.Diagnostics, developmentLockInvalid) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, developmentLockInvalid)
	}
}

// TestDevelopmentMultiSkillExtraRootFails proves a Lock root no selected Skill
// declares is rejected instead of being silently compiled in.
func TestDevelopmentMultiSkillExtraRootFails(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:host-context\n")
	request = cloneDevelopmentRequest(t, request)
	var typesKey DevelopmentPackageKey
	for _, snapshot := range request.Snapshots {
		if snapshot.Key.PackageID == "urn:example:ca:types" {
			typesKey = snapshot.Key
		}
	}
	request.Lock.Roots = append(request.Lock.Roots, DevelopmentLockRoot{Alias: "extra", Key: typesKey})
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("a Lock root no selected Skill declares must fail")
	}
	if !hasDevelopmentCode(result.Diagnostics, developmentLockInvalid) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, developmentLockInvalid)
	}
}

// TestDevelopmentMultiSkillAliasIdentityConflictFails proves two Skills cannot
// bind the same alias name to two different Package identities.
func TestDevelopmentMultiSkillAliasIdentityConflictFails(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca: urn:example:ca:types\n")
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("one alias bound to two Stable Package IDs across Skills must fail")
	}
	if !hasDevelopmentCode(result.Diagnostics, developmentLockInvalid) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, developmentLockInvalid)
	}
}

// TestDevelopmentMultiSkillBrokenClosureFails proves the Lock Closure must equal
// the transitive closure of the union of roots: removing a transitive edge fails
// even when every root itself is present.
func TestDevelopmentMultiSkillBrokenClosureFails(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:host-context\n")
	request = cloneDevelopmentRequest(t, request)
	for index := range request.Lock.Packages {
		if request.Lock.Packages[index].Key.PackageID != "urn:example:ca:tools" {
			continue
		}
		request.Lock.Packages[index].Dependencies = nil
	}
	result := BuildDevelopment(request)
	if result.Success {
		t.Fatal("a Lock Closure missing a transitive dependency must fail")
	}
	if !hasDevelopmentCode(result.Diagnostics, developmentLockInvalid) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, developmentLockInvalid)
	}
}

// TestDevelopmentMultiSkillReorderingInvariant proves the input order of a
// multi-Skill request does not change a single artifact byte.
func TestDevelopmentMultiSkillReorderingInvariant(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:host-context\n")
	base := BuildDevelopment(request)
	if !base.Success {
		t.Fatalf("base build failed: %#v", base.Diagnostics)
	}
	permuted := cloneDevelopmentRequest(t, request)
	developmentReverseFiles(permuted.Files)
	for left, right := 0, len(permuted.Targets)-1; left < right; left, right = left+1, right-1 {
		permuted.Targets[left], permuted.Targets[right] = permuted.Targets[right], permuted.Targets[left]
	}
	for index := range permuted.Snapshots {
		developmentReverseFiles(permuted.Snapshots[index].Files)
	}
	reverseSnapshots(permuted.Snapshots)
	reverseRoots(permuted.Lock.Roots)
	reversePackages(permuted.Lock.Packages)
	for index := range permuted.Lock.Packages {
		reverseEdges(permuted.Lock.Packages[index].Dependencies)
	}
	result := BuildDevelopment(permuted)
	if !result.Success {
		t.Fatalf("permuted build failed: %#v", result.Diagnostics)
	}
	if result.ArtifactDigest != base.ArtifactDigest {
		t.Fatalf("reordered multi-Skill inputs changed artifactDigest: %s != %s", result.ArtifactDigest, base.ArtifactDigest)
	}
	assertDevelopmentFilesEqual(t, base, result)
}

// TestDevelopmentSingleSkillRootsUnchanged keeps the single-Skill behavior
// explicit: the original fixture, which declares every root, still builds.
func TestDevelopmentSingleSkillRootsUnchanged(t *testing.T) {
	result := BuildDevelopment(loadDevelopmentFixture(t))
	if !result.Success {
		t.Fatalf("single-Skill fixture must still build: %#v", result.Diagnostics)
	}
}


// TestDevelopmentMultiSkillSharedPackageWithDifferentAliases proves the Lock
// models alias -> Package key, not a bijection between aliases and Packages.
// Different Skills may use different aliases for exactly the same immutable
// Interface Package snapshot without silently losing one of the bindings.
func TestDevelopmentMultiSkillSharedPackageWithDifferentAliases(t *testing.T) {
	request := addDevelopmentSecondSkill(t, loadDevelopmentFixture(t),
		"  ca-context: urn:example:ca:host-context\n  provider: urn:example:ca:tools\n")
	request = replaceDevelopmentContract(t, request, developmentSecondSkill,
		"goal: ue.context_only\n"+
			"contextSchema: ca-context::urn:example:ca:invocation-context:v1\n"+
			"dependencies:\n"+
			"  ca-context: urn:example:ca:host-context\n"+
			"  provider: urn:example:ca:tools\n"+
			"capabilities:\n"+
			"  - example.registration-read\n")
	request = replaceDevelopmentSkillFileBytes(t, request,
		"skills/"+developmentSecondSkill+"/assurance/procedure.yaml", []byte(
			"start: inspect-context\n"+
				"nodes:\n"+
				"  inspect-context:\n"+
				"    observation:\n"+
				"      tool: provider::ue.registration.current\n"+
				"      arguments:\n"+
				"        ueRef: context.ueRef\n"+
				"      routes:\n"+
				"        SUCCEEDED: check-context\n"+
				"        FAILED: FAILED\n"+
				"        UNKNOWN: UNRESOLVED\n"+
				"  check-context:\n"+
				"    postcondition:\n"+
				"      predicates:\n"+
				"        - predicate: example.context-present\n"+
				"          arguments: []\n"+
				"      routes:\n"+
				"        PASS: SUCCEEDED\n"+
				"        FAIL: FAILED\n"+
				"        UNKNOWN: UNRESOLVED\n"))
	request = cloneDevelopmentRequest(t, request)
	for _, root := range request.Lock.Roots {
		if root.Alias == "ca" {
			request.Lock.Roots = append(request.Lock.Roots,
				DevelopmentLockRoot{Alias: "provider", Key: root.Key})
			break
		}
	}
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("different used Root aliases for the same Package must build: %#v", result.Diagnostics)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
	permuted := cloneDevelopmentRequest(t, request)
	reverseRoots(permuted.Lock.Roots)
	again := BuildDevelopment(permuted)
	if !again.Success || again.ArtifactDigest != result.ArtifactDigest {
		t.Fatalf("Root order changed a shared-package build: %#v", again.Diagnostics)
	}
	assertDevelopmentFilesEqual(t, result, again)
}

// TestDevelopmentSyntheticAliasNeverOverwritesRoot makes the legacy projection
// collision-safe when an author uses the reserved-looking alias locked_0.
func TestDevelopmentSyntheticAliasNeverOverwritesRoot(t *testing.T) {
	request := cloneDevelopmentRequest(t, loadDevelopmentFixture(t))
	for index := range request.Lock.Roots {
		if request.Lock.Roots[index].Alias == "ca" {
			request.Lock.Roots[index].Alias = "locked_0"
		}
	}
	for index := range request.Files {
		content := string(request.Files[index].Bytes)
		if strings.HasSuffix(request.Files[index].Path, "/assurance/contract.yaml") {
			content = strings.ReplaceAll(content, "  ca: ", "  locked_0: ")
		}
		content = strings.ReplaceAll(content, "ca::", "locked_0::")
		request.Files[index].Bytes = []byte(content)
	}
	result := BuildDevelopment(request)
	if !result.Success {
		t.Fatalf("declared locked_0 root must not collide with a generated alias: %#v", result.Diagnostics)
	}
	if err := VerifyDevelopment(request, result); err != nil {
		t.Fatalf("VerifyDevelopment: %v", err)
	}
}
