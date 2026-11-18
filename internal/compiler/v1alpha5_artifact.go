package compiler

// CA-SL v1alpha5 WP8: deterministic successful-Build artifact assembly.
//
// The candidate v1alpha5 target reuses the single Go Core pipeline: request
// admission, Procedure/AST, WP5 graph, WP3 trigger/correlation linking, WP4
// wait.output SourcePath binding and WP6 response-boundary/mapping validation.
// Only after that pipeline returns zero error diagnostics does Build assemble a
// complete deterministic artifact tree. There are no timestamps, random values,
// environment probes or dependency bytes in the output; the same BuildRequest
// always yields byte-identical files.
//
// Layout (candidate, compiler-owned):
//
//	manifest.json
//	skills/<skill>/compiled/procedure.json
//	skills/<skill>/compiled/ingress-links.json
//
// The manifest is canonical JCS and lists every emitted file (except itself)
// with its sha256, the measured dependency snapshot closure identity, per-Skill
// source digests and the linked Ingress Operation / response-boundary metadata
// for every Wait-bearing Node. It deliberately carries no Owner qualification,
// deployment authorization, Runtime admission or Artifact-bytes-of-dependencies
// claim. See docs/design/compiler-artifact-v1alpha5.md and
// contracts/v1alpha5/artifact-manifest.schema.json.
//
// v1alpha4 artifact generation and verification are untouched; this file never
// participates in the v1alpha4 Build path.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/odango-chan/ca-skill-compiler/internal/artifact"
	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
)

const (
	// V1Alpha5ArtifactAPIVersion is the candidate Compiler Artifact API version
	// for the v1alpha5 language target. It is independent of the frozen
	// v1alpha4 artifact contract.
	V1Alpha5ArtifactAPIVersion = "connectionagent.io/compiler-artifact/v1alpha5"
	// V1Alpha5LanguageTarget is the candidate CA-SL language target recorded in
	// the artifact manifest (also exported through V1Alpha5Target).
	V1Alpha5LanguageTarget = V1Alpha5Target
	// V1Alpha5ArtifactManifestPath is the single top-level manifest path. It is
	// the only artifact file outside a skills/<name>/compiled/ tree.
	V1Alpha5ArtifactManifestPath = "manifest.json"
)

const (
	v1alpha5ArtifactGeneratorName = "ca-skill-compiler"
	v1alpha5SourceDigestDomain    = "CA-SKILL-V1ALPHA5-SOURCE-SET\x00v1\x00"
	v1alpha5CompiledDir           = "/compiled/"
	v1alpha5ProcedureFileName     = "procedure.json"
	v1alpha5IngressLinksFileName  = "ingress-links.json"

	// v1alpha5ArtifactValidationStatus is the manifest-level declaration that a
	// successful Build covers the complete released v1alpha5 Compiler-owned static
	// validation scope. It is deliberately not an Owner/Host/Runtime or product
	// qualification claim.
	v1alpha5ArtifactValidationStatus = "FULL_LANGUAGE_VALIDATION"
)

// v1alpha5ManifestValidatedPasses lists the static passes a successful
// candidate Build actually executed and gated on.
var v1alpha5ManifestValidatedPasses = []string{
	"v1alpha5.request-envelope",
	"wp2.node-entry-wait-procedure-ast",
	"wp5.procedure-graph",
	"wp3.ingress-exact-link",
	"wp3.trigger-correlation-binding",
	"wp4.wait-output-sourcepath",
	"wp6.response-boundary",
	"wp1.request-normalization-and-binary-lifecycle",
	"wp1.peer-profile-reference-form",
	"inherited.package-and-dependency-identity",
	"inherited.owner-tool-capability-linking",
	"inherited.schema-cel-predicate-script",
	"inherited.procedure-and-assurance",
	"inherited.evidence-mutation-reconcile-retry-static",
	"inherited.evidence-descriptor-linking",
	"inherited.evidence-host-recovery-dual-producer",
	"artifact.self-verification",
}

// v1alpha5ManifestNotValidated lists the candidate obligations this Build does
// NOT gate. It keeps a successful artifact from being read as full validation;
// the same list is documented in docs/known-limitations-v1alpha5.md.
var v1alpha5ManifestNotValidated = []string{
	"host-runtime-integration-semantics",
	"real-owner-artifact-qualification",
}

type v1alpha5ManifestValidationScope struct {
	Status          string   `json:"status"`
	ValidatedPasses []string `json:"validatedPasses"`
	NotValidated    []string `json:"notValidated"`
}

type v1alpha5ManifestGenerator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type v1alpha5ManifestExportRef struct {
	ExportKind string `json:"exportKind"`
	ExportID   string `json:"exportId"`
	Owner      string `json:"owner"`
}

type v1alpha5ManifestDependency struct {
	DependencyID    string                      `json:"dependencyId"`
	StablePackageID string                      `json:"stablePackageId"`
	Exports         []v1alpha5ManifestExportRef `json:"exports"`
}

type v1alpha5ManifestEvent struct {
	Source    string `json:"source"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
}

type v1alpha5ManifestProtocol struct {
	Specification string `json:"specification"`
	Release       string `json:"release"`
	APIVersion    string `json:"apiUriVersion"`
	OperationID   string `json:"operationId"`
	Method        string `json:"method"`
	Path          string `json:"path"`
}

type v1alpha5ManifestIngress struct {
	DependencyAlias              string                   `json:"dependencyAlias"`
	CatalogOwner                 string                   `json:"catalogOwner"`
	SnapshotPath                 string                   `json:"snapshotPath"`
	OperationID                  string                   `json:"operationId"`
	NormalizedSchemaRef          string                   `json:"normalizedSchemaRef"`
	NormalizedSchemaDigest       string                   `json:"normalizedSchemaClosureDigest"`
	NormalizedSchemaSnapshotPath string                   `json:"normalizedSchemaSnapshotPath,omitempty"`
	Protocol                     v1alpha5ManifestProtocol `json:"protocol"`
}

type v1alpha5ManifestResponse struct {
	Mode                  string   `json:"mode"`
	Declared              bool     `json:"declared"`
	BoundaryNodeID        string   `json:"boundaryNodeId"`
	ProducerStage         string   `json:"producerStage,omitempty"`
	ResultSchemaRef       string   `json:"responseResultSchemaRef,omitempty"`
	ResultSchemaDigest    string   `json:"responseResultSchemaClosureDigest,omitempty"`
	EarlyFailureTerminals []string `json:"earlyFailureTerminals"`
}

type v1alpha5ManifestWaitLink struct {
	NodeID       string                   `json:"nodeId"`
	Event        v1alpha5ManifestEvent    `json:"event"`
	Timeout      string                   `json:"timeout"`
	TimeoutRoute string                   `json:"timeoutRoute"`
	Ingress      v1alpha5ManifestIngress  `json:"ingress"`
	Response     v1alpha5ManifestResponse `json:"response"`
}

type v1alpha5ManifestSkill struct {
	Name          string                     `json:"name"`
	SourceDigest  string                     `json:"sourceDigest"`
	SourceFiles   []string                   `json:"sourceFiles"`
	CompiledFiles []string                   `json:"compiledFiles"`
	WaitLinks     []v1alpha5ManifestWaitLink `json:"waitLinks"`
}

type v1alpha5ManifestOutputFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type v1alpha5ArtifactManifest struct {
	APIVersion        string                         `json:"apiVersion"`
	LanguageTarget    string                         `json:"languageTarget"`
	ValidationScope   v1alpha5ManifestValidationScope `json:"validationScope"`
	Generator         v1alpha5ManifestGenerator      `json:"generator"`
	DependencyClosure []v1alpha5ManifestDependency   `json:"dependencyClosure"`
	Skills            []v1alpha5ManifestSkill        `json:"skills"`
	OutputFiles       []v1alpha5ManifestOutputFile   `json:"outputFiles"`
}

type v1alpha5IngressLinksDocument struct {
	Links []v1alpha5ManifestWaitLink `json:"links"`
}

// v1alpha5ArtifactSkill is the validated per-Skill state used only by artifact
// assembly. It is built by re-running the deterministic static passes after the
// public Build path already established that they produce zero errors.
type v1alpha5ArtifactSkill struct {
	name         string
	sourceFiles  []File
	procedure    *V1Alpha5ProcedureAST
	waitLinks    []v1alpha5ManifestWaitLink
	dependencies []v1alpha5ManifestDependency
}

// assembleV1Alpha5SuccessResult builds the deterministic artifact tree for a
// request whose v1alpha5 static validation returned zero error diagnostics. Any
// assembly inconsistency fails the whole Build atomically.
func assembleV1Alpha5SuccessResult(req BuildRequest) (BuildResult, error) {
	skills, err := collectV1Alpha5ArtifactSkills(req)
	if err != nil {
		return BuildResult{}, err
	}
	files, closure, err := buildV1Alpha5ArtifactFiles(skills)
	if err != nil {
		return BuildResult{}, err
	}
	outputs := make([]File, 0, len(files))
	for _, file := range files {
		outputs = append(outputs, File{Path: file.path, Bytes: file.bytes})
	}
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Path < outputs[j].Path })
	aliases := make([]string, 0, len(closure))
	for _, entry := range closure {
		aliases = append(aliases, entry.DependencyID)
	}
	sort.Strings(aliases)
	return BuildResult{
		APIVersion:        BuildResultAPIVersionV1Alpha5,
		Success:           true,
		Diagnostics:       []Diagnostic{},
		DependencyClosure: aliases,
		OutputFiles:       outputs,
	}, nil
}

// collectV1Alpha5ArtifactSkills re-runs the deterministic static passes for the
// selected Skills and keeps the validated linking state. The public Build path
// only calls it after zero-error validation, so any diagnostic here is an
// internal inconsistency reported as an assembly failure.
func collectV1Alpha5ArtifactSkills(req BuildRequest) ([]v1alpha5ArtifactSkill, error) {
	skillFiles := make(map[string][]File)
	skillNames := make(map[string]bool)
	for _, file := range req.Files {
		parts := strings.Split(file.Path, "/")
		if len(parts) < 2 || parts[0] != "skills" {
			continue
		}
		skillFiles[parts[1]] = append(skillFiles[parts[1]], file)
		skillNames[parts[1]] = true
	}
	if len(skillNames) == 0 {
		return nil, fmt.Errorf("no logical Skill package rooted at skills/<name>")
	}
	selected := make(map[string]bool, len(skillNames))
	if len(req.Targets) == 0 {
		for name := range skillNames {
			selected[name] = true
		}
	} else {
		for _, target := range req.Targets {
			if !skillNames[target] {
				return nil, fmt.Errorf("selected Skill target %q does not exist in BuildRequest.files", target)
			}
			selected[target] = true
		}
	}

	names := sortedStringMapKeys(selected)
	skills := make([]v1alpha5ArtifactSkill, 0, len(names))
	for _, name := range names {
		base := "skills/" + name + "/"
		procedurePath := base + "assurance/procedure.yaml"

		files := append([]File(nil), skillFiles[name]...)
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

		parsedFiles := make(map[string]*yaml.Node)
		for _, file := range files {
			if strings.HasSuffix(file.Path, ".yaml") || strings.HasSuffix(file.Path, ".yml") {
				root, err := yaml.Parse(file.Bytes)
				if err != nil {
					return nil, fmt.Errorf("Skill %s source %s is not valid YAML: %w", name, file.Path, err)
				}
				parsedFiles[file.Path] = root
			}
		}
		procedureRaw, exists := parsedFiles[procedurePath]
		if !exists {
			return nil, fmt.Errorf("Skill %s is missing %s", name, procedurePath)
		}
		skillAST, astDiagnostics := parseV1Alpha5SkillAST(name, parsedFiles)
		if len(astDiagnostics) != 0 {
			return nil, fmt.Errorf("Skill %s AST diagnostics reappeared during assembly: %v", name, diagnosticCodes(astDiagnostics))
		}
		if skillAST == nil || skillAST.Procedure == nil {
			return nil, fmt.Errorf("Skill %s has no parsed v1alpha5 Procedure", name)
		}
		if skillAST.Procedure.Raw != procedureRaw {
			// The Procedure must be the single normalized parse result, not a copy.
			return nil, fmt.Errorf("Skill %s Procedure AST does not reference the normalized parse result", name)
		}

		bindingDiagnostics, source := validateV1Alpha5WaitBindings(req, skillAST, procedurePath, parsedFiles)
		if len(bindingDiagnostics) != 0 {
			return nil, fmt.Errorf("Skill %s linking diagnostics reappeared during assembly: %v", name, diagnosticCodes(bindingDiagnostics))
		}
		if outputDiagnostics := validateV1Alpha5WaitOutputSourcePaths(skillAST, procedurePath, source); len(outputDiagnostics) != 0 {
			return nil, fmt.Errorf("Skill %s wait.output diagnostics reappeared during assembly: %v", name, diagnosticCodes(outputDiagnostics))
		}

		waitLinks := make([]v1alpha5ManifestWaitLink, 0)
		dependencies := make([]v1alpha5ManifestDependency, 0)
		if source != nil {
			if source.link == nil {
				return nil, fmt.Errorf("Skill %s has Wait-bearing Nodes but no trusted linking context", name)
			}
			exportsByAlias := make(map[string][]v1alpha5ManifestExportRef)
			stableByAlias := make(map[string]string)
			for _, nodeID := range sortedStringMapKeys(skillAST.Procedure.ByID) {
				node := skillAST.Procedure.ByID[nodeID]
				if node == nil || node.Wait == nil {
					continue
				}
				contract, ok := source.contracts[nodeID]
				if !ok || contract == nil {
					return nil, fmt.Errorf("Skill %s Wait Node %s has no validated Ingress link", name, nodeID)
				}
				link, err := buildV1Alpha5ManifestWaitLink(source.link, nodeID, node.Wait, contract)
				if err != nil {
					return nil, fmt.Errorf("Skill %s Wait Node %s: %w", name, nodeID, err)
				}
				waitLinks = append(waitLinks, link)

				alias := contract.Export.DependencyAlias
				if alias == "" || contract.Export.SnapshotPath == "" || contract.Export.Operation.ID == "" {
					return nil, fmt.Errorf("Skill %s Wait Node %s has an incomplete Ingress link identity", name, nodeID)
				}
				dependency, ok := source.scopeByAlias[alias]
				if !ok || dependency.PackageID == "" {
					return nil, fmt.Errorf("Skill %s Wait Node %s links dependency alias %q without a verified Stable Package ID", name, nodeID, alias)
				}
				stableByAlias[alias] = dependency.PackageID
				exportsByAlias[alias] = append(exportsByAlias[alias], v1alpha5ManifestExportRef{
					ExportKind: "ingressOperation",
					ExportID:   contract.Export.Operation.ID,
					Owner:      contract.Export.CatalogOwner,
				})
				for _, trigger := range contract.Triggers {
					triggerAlias, err := v1alpha5AliasForPackage(source.scopeByAlias, trigger.PackageID)
					if err != nil {
						return nil, fmt.Errorf("Skill %s Wait Node %s trigger %s: %w", name, nodeID, trigger.ExportID, err)
					}
					stableByAlias[triggerAlias] = source.scopeByAlias[triggerAlias].PackageID
					exportsByAlias[triggerAlias] = append(exportsByAlias[triggerAlias], v1alpha5ManifestExportRef{
						ExportKind: "operation",
						ExportID:   trigger.ExportID,
						Owner:      trigger.Owner,
					})
				}
			}
			aliases := make([]string, 0, len(exportsByAlias))
			for alias := range exportsByAlias {
				aliases = append(aliases, alias)
			}
			sort.Strings(aliases)
			for _, alias := range aliases {
				exports, err := normalizeV1Alpha5ManifestExports(exportsByAlias[alias])
				if err != nil {
					return nil, fmt.Errorf("Skill %s dependency %s: %w", name, alias, err)
				}
				dependencies = append(dependencies, v1alpha5ManifestDependency{
					DependencyID:    alias,
					StablePackageID: stableByAlias[alias],
					Exports:         exports,
				})
			}
		}

		// The artifact dependency closure is the full inherited closure, not just
		// the Wait trigger/Ingress links: ordinary Operation and Observation
		// callables, Capability membership and Schema dependencies are part of the
		// compiled Skill and must be represented exactly once.
		filesByName := make(map[string][]byte, len(files))
		for _, file := range files {
			filesByName[file.Path] = file.Bytes
		}
		_, inherited := validateInheritedSkillPackage(
			req, name, filesByName, parsedFiles,
			inheritedValidationOptions{
				allowWait:             true,
				missingSourceMessage:  "Required v1alpha5 author source is missing",
				additionalUsedAliases: v1alpha5IngressDependencyAliases(req, skillAST, parsedFiles),
			},
			func(string, *yaml.Node, string, string) {},
		)
		present := make(map[string]bool, len(dependencies))
		for _, dependency := range dependencies {
			present[dependency.DependencyID] = true
		}
		for _, alias := range inherited.identity.Closure {
			if present[alias] {
				continue
			}
			exports := make([]v1alpha5ManifestExportRef, 0)
			for _, export := range inherited.ownerLinking.Exports {
				if export.DependencyAlias != alias {
					continue
				}
				if export.ExportKind != "operation" && export.ExportKind != "observation" && export.ExportKind != "schema" {
					continue
				}
				exports = append(exports, v1alpha5ManifestExportRef{
					ExportKind: export.ExportKind,
					ExportID:   export.ExportID,
					Owner:      export.TrustedOwner,
				})
			}
			merged, err := normalizeV1Alpha5ManifestExports(exports)
			if err != nil {
				return nil, fmt.Errorf("Skill %s inherited dependency %s: %w", name, alias, err)
			}
			stablePackageID := ""
			for _, identity := range inherited.identity.Identities {
				if identity.Alias == alias {
					stablePackageID = identity.StablePackageID
					break
				}
			}
			dependencies = append(dependencies, v1alpha5ManifestDependency{
				DependencyID:    alias,
				StablePackageID: stablePackageID,
				Exports:         merged,
			})
			present[alias] = true
		}
		sort.Slice(dependencies, func(i, j int) bool { return dependencies[i].DependencyID < dependencies[j].DependencyID })

		skills = append(skills, v1alpha5ArtifactSkill{
			name:         name,
			sourceFiles:  files,
			procedure:    skillAST.Procedure,
			waitLinks:    waitLinks,
			dependencies: dependencies,
		})
	}
	return skills, nil
}

// buildV1Alpha5ManifestWaitLink projects one validated Wait link into the stable
// manifest shape. Only facts already proven by WP3/WP6 are recorded.
func buildV1Alpha5ManifestWaitLink(link *v1alpha5LinkContext, nodeID string, wait *V1Alpha5WaitAST, contract *v1alpha5ExportContract) (v1alpha5ManifestWaitLink, error) {
	if link == nil || wait == nil || contract == nil {
		return v1alpha5ManifestWaitLink{}, fmt.Errorf("missing validated link state")
	}
	export := contract.Export
	if export.Operation.ID == "" || export.SnapshotPath == "" || export.DependencyAlias == "" ||
		export.CatalogOwner == "" || contract.Normalized.ArtifactRef == "" {
		return v1alpha5ManifestWaitLink{}, fmt.Errorf("incomplete trusted Ingress identity")
	}
	normalizedClosure, err := link.resolution.ComputeSchemaClosure(contract.Normalized)
	if err != nil {
		return v1alpha5ManifestWaitLink{}, fmt.Errorf("normalized request Schema closure: %w", err)
	}
	if normalizedClosure == nil || normalizedClosure.Digest == "" {
		return v1alpha5ManifestWaitLink{}, fmt.Errorf("normalized request Schema closure digest is empty")
	}

	responseLink, err := buildV1Alpha5ManifestResponse(link, nodeID, wait, export)
	if err != nil {
		return v1alpha5ManifestWaitLink{}, err
	}

	return v1alpha5ManifestWaitLink{
		NodeID:       nodeID,
		Event:        v1alpha5ManifestEvent{Source: wait.Event.Source, Service: wait.Event.Service, Operation: wait.Event.Operation},
		Timeout:      wait.Timeout,
		TimeoutRoute: wait.Routes.Timeout,
		Ingress: v1alpha5ManifestIngress{
			DependencyAlias:              export.DependencyAlias,
			CatalogOwner:                 export.CatalogOwner,
			SnapshotPath:                 export.SnapshotPath,
			OperationID:                  export.Operation.ID,
			NormalizedSchemaRef:          contract.Normalized.ArtifactRef,
			NormalizedSchemaDigest:       normalizedClosure.Digest,
			NormalizedSchemaSnapshotPath: link.snapshotPathForRoot(contract.Normalized),
			Protocol: v1alpha5ManifestProtocol{
				Specification: export.Operation.Protocol.Specification,
				Release:       export.Operation.Protocol.Release,
				APIVersion:    export.Operation.Protocol.APIVersion,
				OperationID:   export.Operation.Protocol.OpenAPI.OperationID,
				Method:        export.Operation.Protocol.OpenAPI.Method,
				Path:          export.Operation.Protocol.OpenAPI.Path,
			},
		},
		Response: responseLink,
	}, nil
}

// buildV1Alpha5ManifestResponse records the paired-response boundary facts that
// WP6 proved. The Ingress-owned response contract is re-derived through the same
// deterministic builder; a non-empty diagnostic result is an assembly failure
// because Build only reaches this point after zero-error validation.
func buildV1Alpha5ManifestResponse(link *v1alpha5LinkContext, nodeID string, wait *V1Alpha5WaitAST, export v1alpha5TrustedIngressExport) (v1alpha5ManifestResponse, error) {
	contract, diagnostics := buildV1Alpha5ResponseContract(export, link)
	if len(diagnostics) != 0 || contract == nil {
		return v1alpha5ManifestResponse{}, fmt.Errorf("Ingress response contract diagnostics reappeared: %v", diagnosticCodes(diagnostics))
	}
	mode := export.Operation.Response.Mode
	if mode == "" {
		mode = "none"
	}
	boundary := nodeID
	declared := wait.Response != nil
	if declared && wait.Response != nil && wait.Response.After != "" {
		boundary = wait.Response.After
	}
	response := v1alpha5ManifestResponse{
		Mode:                  mode,
		Declared:              declared,
		BoundaryNodeID:        boundary,
		ProducerStage:         contract.Producer,
		EarlyFailureTerminals: sortedStringMapKeys(contract.EarlyFailures),
	}
	if contract.HasResultRoot {
		closure, err := link.resolution.ComputeSchemaClosure(contract.ResultRoot)
		if err != nil {
			return v1alpha5ManifestResponse{}, fmt.Errorf("response result Schema closure: %w", err)
		}
		response.ResultSchemaRef = contract.ResultRoot.ArtifactRef
		if closure != nil {
			response.ResultSchemaDigest = closure.Digest
		}
	}
	return response, nil
}

func v1alpha5AliasForPackage(scope map[string]v1alpha5Dependency, packageID string) (string, error) {
	if packageID == "" {
		return "", fmt.Errorf("resolved trigger has no Stable Package ID")
	}
	alias := ""
	for candidate, dependency := range scope {
		if dependency.PackageID != packageID {
			continue
		}
		if alias != "" && alias != candidate {
			return "", fmt.Errorf("Stable Package ID %q resolves to multiple dependency aliases", packageID)
		}
		alias = candidate
	}
	if alias == "" {
		return "", fmt.Errorf("Stable Package ID %q is not in the verified dependency scope", packageID)
	}
	return alias, nil
}

func normalizeV1Alpha5ManifestExports(exports []v1alpha5ManifestExportRef) ([]v1alpha5ManifestExportRef, error) {
	seen := make(map[v1alpha5ManifestExportRef]bool, len(exports))
	normalized := make([]v1alpha5ManifestExportRef, 0, len(exports))
	for _, export := range exports {
		if export.ExportKind == "" || export.ExportID == "" || export.Owner == "" {
			return nil, fmt.Errorf("export reference requires exportKind, exportId and owner")
		}
		if seen[export] {
			continue
		}
		seen[export] = true
		normalized = append(normalized, export)
	}
	sort.Slice(normalized, func(i, j int) bool {
		a, b := normalized[i], normalized[j]
		if a.ExportKind != b.ExportKind {
			return a.ExportKind < b.ExportKind
		}
		if a.ExportID != b.ExportID {
			return a.ExportID < b.ExportID
		}
		return a.Owner < b.Owner
	})
	return normalized, nil
}

type v1alpha5ArtifactFile struct {
	path  string
	bytes []byte
}

// buildV1Alpha5ArtifactFiles emits the deterministic file tree and the merged
// dependency-closure identity recorded both in the manifest and BuildResult.
func buildV1Alpha5ArtifactFiles(skills []v1alpha5ArtifactSkill) ([]v1alpha5ArtifactFile, []v1alpha5ManifestDependency, error) {
	manifestSkills := make([]v1alpha5ManifestSkill, 0, len(skills))
	files := make([]v1alpha5ArtifactFile, 0, len(skills)*2)

	closureByAlias := make(map[string]v1alpha5ManifestDependency)
	for _, skill := range skills {
		if skill.procedure == nil || skill.procedure.Raw == nil {
			return nil, nil, fmt.Errorf("Skill %s has no normalized Procedure document", skill.name)
		}
		procedureValue := nodeValue(skill.procedure.Raw)
		if procedureValue == nil {
			return nil, nil, fmt.Errorf("Skill %s Procedure document cannot be converted to JSON", skill.name)
		}
		procedureBytes, err := artifact.CanonicalizeValue(procedureValue)
		if err != nil {
			return nil, nil, fmt.Errorf("Skill %s Procedure document canonicalization failed: %w", skill.name, err)
		}
		links := append([]v1alpha5ManifestWaitLink(nil), skill.waitLinks...)
		if links == nil {
			links = []v1alpha5ManifestWaitLink{}
		}
		linksBytes, err := artifact.CanonicalizeValue(v1alpha5IngressLinksDocument{Links: links})
		if err != nil {
			return nil, nil, fmt.Errorf("Skill %s ingress link document canonicalization failed: %w", skill.name, err)
		}

		prefix := "skills/" + skill.name + "/compiled/"
		procedurePath := prefix + v1alpha5ProcedureFileName
		linksPath := prefix + v1alpha5IngressLinksFileName
		if err := validV1Alpha5ArtifactPath(procedurePath); err != nil {
			return nil, nil, err
		}
		if err := validV1Alpha5ArtifactPath(linksPath); err != nil {
			return nil, nil, err
		}
		files = append(files,
			v1alpha5ArtifactFile{path: procedurePath, bytes: procedureBytes},
			v1alpha5ArtifactFile{path: linksPath, bytes: linksBytes},
		)

		sourceFiles := make([]string, 0, len(skill.sourceFiles))
		for _, file := range skill.sourceFiles {
			sourceFiles = append(sourceFiles, file.Path)
		}
		sort.Strings(sourceFiles)
		if len(sourceFiles) == 0 {
			return nil, nil, fmt.Errorf("Skill %s has no source files", skill.name)
		}
		sourceDigest, err := v1alpha5SourceSetDigest(skill.sourceFiles)
		if err != nil {
			return nil, nil, fmt.Errorf("Skill %s source digest: %w", skill.name, err)
		}
		compiledFiles := []string{procedurePath, linksPath}
		sort.Strings(compiledFiles)
		manifestSkills = append(manifestSkills, v1alpha5ManifestSkill{
			Name:          skill.name,
			SourceDigest:  sourceDigest,
			SourceFiles:   sourceFiles,
			CompiledFiles: compiledFiles,
			WaitLinks:     links,
		})

		for _, dependency := range skill.dependencies {
			existing, ok := closureByAlias[dependency.DependencyID]
			if !ok {
				closureByAlias[dependency.DependencyID] = dependency
				continue
			}
			if existing.StablePackageID != dependency.StablePackageID {
				return nil, nil, fmt.Errorf("dependency alias %s has conflicting Stable Package IDs", dependency.DependencyID)
			}
			merged, err := normalizeV1Alpha5ManifestExports(append(append([]v1alpha5ManifestExportRef{}, existing.Exports...), dependency.Exports...))
			if err != nil {
				return nil, nil, err
			}
			existing.Exports = merged
			closureByAlias[dependency.DependencyID] = existing
		}
	}

	aliases := make([]string, 0, len(closureByAlias))
	for alias := range closureByAlias {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	closure := make([]v1alpha5ManifestDependency, 0, len(aliases))
	for _, alias := range aliases {
		closure = append(closure, closureByAlias[alias])
	}

	outputFiles := make([]v1alpha5ManifestOutputFile, 0, len(files))
	for _, file := range files {
		sum := sha256.Sum256(file.bytes)
		outputFiles = append(outputFiles, v1alpha5ManifestOutputFile{
			Path:   file.path,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(outputFiles, func(i, j int) bool { return outputFiles[i].Path < outputFiles[j].Path })

	manifest := v1alpha5ArtifactManifest{
		APIVersion:     V1Alpha5ArtifactAPIVersion,
		LanguageTarget: V1Alpha5LanguageTarget,
		ValidationScope: v1alpha5ManifestValidationScope{
			Status:          v1alpha5ArtifactValidationStatus,
			ValidatedPasses: append([]string(nil), v1alpha5ManifestValidatedPasses...),
			NotValidated:    append([]string(nil), v1alpha5ManifestNotValidated...),
		},
		Generator:         v1alpha5ManifestGenerator{Name: v1alpha5ArtifactGeneratorName, Version: CompilerVersion},
		DependencyClosure: closure,
		Skills:            manifestSkills,
		OutputFiles:       outputFiles,
	}
	manifestBytes, err := artifact.CanonicalizeValue(manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("v1alpha5 artifact manifest canonicalization failed: %w", err)
	}
	if _, exists := findV1Alpha5ArtifactFile(files, V1Alpha5ArtifactManifestPath); exists {
		return nil, nil, fmt.Errorf("v1alpha5 artifact manifest path collides with an output file")
	}
	files = append(files, v1alpha5ArtifactFile{path: V1Alpha5ArtifactManifestPath, bytes: manifestBytes})
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, closure, nil
}

func findV1Alpha5ArtifactFile(files []v1alpha5ArtifactFile, path string) (v1alpha5ArtifactFile, bool) {
	for _, file := range files {
		if file.path == path {
			return file, true
		}
	}
	return v1alpha5ArtifactFile{}, false
}

func validV1Alpha5ArtifactPath(path string) error {
	if len(path) == 0 || len(path) > 240 {
		return fmt.Errorf("v1alpha5 artifact path must be 1 to 240 bytes: %q", path)
	}
	if !strings.HasPrefix(path, "skills/") || strings.Contains(path, "\\") || strings.Contains(path, "//") ||
		strings.Contains(path, "../") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("v1alpha5 artifact path is not normalized: %q", path)
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("v1alpha5 artifact path has an invalid segment: %q", path)
		}
	}
	return nil
}

// v1alpha5SourceSetDigest hashes the exact author source bytes of one Skill with
// a domain separator and sorted, length-prefixed paths. It excludes request
// ordering, so a permuted BuildRequest produces the same digest.
func v1alpha5SourceSetDigest(files []File) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("source set is empty")
	}
	ordered := append([]File(nil), files...)
	seen := make(map[string]bool, len(ordered))
	for _, file := range ordered {
		if !validSourcePath(file.Path) {
			return "", fmt.Errorf("invalid source path %q", file.Path)
		}
		if seen[file.Path] {
			return "", fmt.Errorf("duplicate source path %q", file.Path)
		}
		seen[file.Path] = true
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	hash := sha256.New()
	_, _ = hash.Write([]byte(v1alpha5SourceDigestDomain))
	var u32 [4]byte
	var u64 [8]byte
	binary.BigEndian.PutUint32(u32[:], uint32(len(ordered)))
	_, _ = hash.Write(u32[:])
	for _, file := range ordered {
		pathBytes := []byte(file.Path)
		binary.BigEndian.PutUint32(u32[:], uint32(len(pathBytes)))
		_, _ = hash.Write(u32[:])
		_, _ = hash.Write(pathBytes)
		binary.BigEndian.PutUint64(u64[:], uint64(len(file.Bytes)))
		_, _ = hash.Write(u64[:])
		_, _ = hash.Write(file.Bytes)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func diagnosticCodes(diagnostics []Diagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	return codes
}
