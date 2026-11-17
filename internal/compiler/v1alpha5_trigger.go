package compiler

// CA-SL v1alpha5 WP3: exact triggering Operation and correlation-binding
// validation for every Wait-bearing Node.
//
// Authoritative clauses: ca-skills #61 Appendix L.4.1/L.4.2/L.11/L.11.1 and the
// #62 Integration Ingress Operation contract §6/§7/§10/§12/§13.
//
// The pass is deliberately exact and fail-closed:
//
//   - each declared trigger identity resolves to exactly one trusted ToolCatalog
//     Operation export by exact Package ID + trusted Owner + kind=operation +
//     exact Operation ID; missing/ambiguous -> EXPORT_ID_MISSING /
//     EXPORT_ID_AMBIGUOUS (never translated to an Ingress Wait code);
//   - the Ingress export's own correlation/stale-delivery declaration is
//     validated and reported with its Ingress-owned INGRESS_CORRELATION_INVALID
//     identity; a failure suppresses the dependent Skill-path diagnostics;
//   - the typed correlation contract is derived through the trusted v1alpha5
//     Schema bindings and the Schema Closure Digest v1 machinery;
//   - for every reachable incoming path to a Wait-bearing Node, exactly one
//     declared trigger must dominate that path and must materialize every
//     correlation argument before dispatch; otherwise the Skill-side
//     WAIT_CORRELATION_BINDING_INVALID identity applies.

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	yaml "github.com/odango-chan/ca-skill-compiler/internal/compiler/yamlast"
	"github.com/odango-chan/ca-skill-compiler/internal/ingresscatalog"
)

// v1alpha5ResolvedTrigger is one declared trigger whose exact trusted ToolCatalog
// Operation export and Input Schema identity have been proven.
type v1alpha5ResolvedTrigger struct {
	Declared      ingresscatalog.TriggerProjection
	PackageID     string
	ExportID      string
	Owner         string
	InputSchemaID string
}

// v1alpha5ExportContract is the fully validated correlation contract of one
// trusted Ingress Operation export.
type v1alpha5ExportContract struct {
	Export        v1alpha5TrustedIngressExport
	Normalized    ingresscatalog.RootRef
	ArmInput      ingresscatalog.RootRef
	KeyFields     []ingresscatalog.CorrelationKeyProjection
	StartAllowed  bool
	StartBindings []ingresscatalog.ArmingBindingProjection
	Triggers      []v1alpha5ResolvedTrigger
}

type v1alpha5ContractCache struct {
	contracts   map[string]*v1alpha5ExportContract
	diagnostics map[string][]Diagnostic
}

func newV1Alpha5ContractCache() *v1alpha5ContractCache {
	return &v1alpha5ContractCache{
		contracts:   map[string]*v1alpha5ExportContract{},
		diagnostics: map[string][]Diagnostic{},
	}
}

func v1alpha5ExportContractKey(export v1alpha5TrustedIngressExport) string {
	return export.DependencyAlias + "\x00" + export.SnapshotPath + "\x00" + export.Operation.ID
}

// validateV1Alpha5WaitBindings is the WP3 linking pass over one selected v1alpha5
// Skill. It runs after the WP5 Procedure graph validation and returns only real
// static diagnostics; Build remains fail-closed for the unimplemented later work
// packages.
func validateV1Alpha5WaitBindings(req BuildRequest, skill *V1Alpha5SkillAST, procedurePath string, parsedFiles map[string]*yaml.Node) []Diagnostic {
	waitNodeIDs := make([]string, 0)
	if skill == nil || skill.Procedure == nil {
		return nil
	}
	for _, nodeID := range sortedStringMapKeys(skill.Procedure.ByID) {
		if node := skill.Procedure.ByID[nodeID]; node != nil && node.Wait != nil {
			waitNodeIDs = append(waitNodeIDs, nodeID)
		}
	}
	// WP3 linking needs the immutable dependency snapshot set in which the
	// Integration package is declared. A v1alpha5 request without any dependency
	// snapshot cannot declare an Integration dependency; WP2-only fixtures keep
	// their exact previous behavior and Build stays fail-closed.
	if len(waitNodeIDs) == 0 || len(req.ResolvedDependencies) == 0 {
		return nil
	}

	var diagnostics []Diagnostic
	contractPath := "skills/" + skill.Name + "/assurance/contract.yaml"
	declared := map[string]declaredDependency{}
	if contractNode := parsedFiles[contractPath]; contractNode != nil {
		declared = collectContractDependencies(contractPath, child(contractNode, "dependencies"), &diagnostics)
	}

	link, linkDiagnostics := buildV1Alpha5LinkContext(req)
	diagnostics = append(diagnostics, linkDiagnostics...)
	if hasErrorDiagnostic(linkDiagnostics) {
		// Trusted artifact/Schema prerequisite failed: suppress all dependent
		// Ingress and Skill-path diagnostics.
		return diagnostics
	}

	scope, scopeDiagnostics := collectV1Alpha5DependencyScope(req, declared, v1alpha5UsedDependencyAliases(skill))
	diagnostics = append(diagnostics, scopeDiagnostics...)
	if hasErrorDiagnostic(scopeDiagnostics) {
		return diagnostics
	}
	scopeByAlias := make(map[string]v1alpha5Dependency, len(scope))
	for _, dependency := range scope {
		scopeByAlias[dependency.Alias] = dependency
	}

	exports, catalogDiagnostics := discoverV1Alpha5IngressExports(scope, link)
	diagnostics = append(diagnostics, catalogDiagnostics...)
	if hasErrorDiagnostic(catalogDiagnostics) {
		// A trusted Catalog failed admission (protocol artifact, Schema Closure or
		// catalog structure). Its exports are not usable; suppress the dependent
		// selector/trigger diagnostics instead of cascading (#62 §13).
		return diagnostics
	}

	graph := buildV1Alpha5ProcedureGraph(skill.Procedure)
	cache := newV1Alpha5ContractCache()
	triggerIndexes := make(map[string]*v1alpha5SnapshotTriggerIndex)
	reachable := graph.ReachableVertices()

	for _, nodeID := range waitNodeIDs {
		wait := skill.Procedure.ByID[nodeID].Wait
		if wait.Event.Source == "" || wait.Event.Service == "" || wait.Event.Operation == "" {
			// Incomplete Wait selectors are already diagnosed by the WP2 parser;
			// no dependent resolution diagnostic is inferred here.
			continue
		}
		export, ambiguous, found := resolveV1Alpha5IngressExport(exports, wait.Event)
		if !found {
			diagnostics = append(diagnostics, v1alpha5WaitSelectorDiagnostics(procedurePath, nodeID, wait, ambiguous)...)
			continue
		}
		contract, contractDiagnostics := cache.contractFor(export, link, scope, triggerIndexes)
		diagnostics = append(diagnostics, contractDiagnostics...)
		if contract == nil {
			continue
		}
		diagnostics = append(diagnostics, v1alpha5ValidateWaitEntryPaths(procedurePath, nodeID, wait, graph, reachable, skill, contract, scopeByAlias)...)
	}
	return diagnostics
}

func (cache *v1alpha5ContractCache) contractFor(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext, scope []v1alpha5Dependency, triggerIndexes map[string]*v1alpha5SnapshotTriggerIndex) (*v1alpha5ExportContract, []Diagnostic) {
	key := v1alpha5ExportContractKey(export)
	if contract, cached := cache.contracts[key]; cached {
		return contract, cache.diagnostics[key]
	}
	contract, diagnostics := buildV1Alpha5ExportContract(export, link, scope, triggerIndexes)
	cache.contracts[key] = contract
	cache.diagnostics[key] = diagnostics
	return contract, diagnostics
}

// v1alpha5SchemaDiagnostic reports a missing or invalid trusted Schema with its
// existing compiler/dependency diagnostic identity.
func v1alpha5SchemaDiagnostic(snapshotPath, code, message string) Diagnostic {
	return v1alpha5StaticDiagnostic(snapshotPath, nil, code, message)
}

// buildV1Alpha5ExportContract validates the Ingress export's correlation and
// arming declaration against the trusted v1alpha5 Schema/Schema Closure context.
// A non-nil diagnostics result is a prerequisite failure: the caller must not
// derive Skill-path correlation diagnostics from an invalid contract.
func buildV1Alpha5ExportContract(export v1alpha5TrustedIngressExport, link *v1alpha5LinkContext, scope []v1alpha5Dependency, triggerIndexes map[string]*v1alpha5SnapshotTriggerIndex) (*v1alpha5ExportContract, []Diagnostic) {
	var diagnostics []Diagnostic
	invalid := func(fieldPath string) {
		diagnostics = append(diagnostics, v1alpha5IngressCorrelationDiagnostic(export, fieldPath))
	}

	if link.resolution == nil {
		diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "DEPENDENCY_IDENTITY_UNVERIFIED",
			"Ingress export '"+export.Operation.ID+"' cannot be linked without a trusted v1alpha5 artifact/Schema resolution context."))
		return nil, diagnostics
	}

	normalizedRef := export.Operation.Request.Normalized.SchemaRef
	normalizedRoot, ok := link.schemaRoots[normalizedRef]
	if !ok {
		diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
			"Normalized request Schema '"+normalizedRef+"' is not declared by the trusted v1alpha5 schema bindings."))
		return nil, diagnostics
	}
	if _, err := link.resolution.ComputeSchemaClosure(normalizedRoot); err != nil {
		diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_INVALID",
			"Normalized request Schema '"+normalizedRef+"' is invalid for Schema Closure Digest v1: "+err.Error()))
		return nil, diagnostics
	}

	armInputRef := export.Operation.Correlation.ArmInputSchemaRef
	armInputRoot, ok := link.schemaRoots[armInputRef]
	if !ok {
		diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
			"Correlation armInputSchema '"+armInputRef+"' is not declared by the trusted v1alpha5 schema bindings."))
		return nil, diagnostics
	}
	if _, err := link.resolution.ComputeSchemaClosure(armInputRoot); err != nil {
		diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_INVALID",
			"Correlation armInputSchema '"+armInputRef+"' is invalid for Schema Closure Digest v1: "+err.Error()))
		return nil, diagnostics
	}

	contract := &v1alpha5ExportContract{
		Export:       export,
		Normalized:   normalizedRoot,
		ArmInput:     armInputRoot,
		KeyFields:    export.Operation.Correlation.KeyFields,
		StartAllowed: export.Operation.Arming.Start.Allowed,
		StartBindings: append([]ingresscatalog.ArmingBindingProjection(nil),
			export.Operation.Arming.Start.Bindings...),
	}

	if len(contract.KeyFields) == 0 {
		invalid("/correlation/keyFields")
		return nil, diagnostics
	}
	seenKeyField := make(map[string]bool, len(contract.KeyFields))
	keyFieldsValid := true
	for index, keyField := range contract.KeyFields {
		fieldPath := "/correlation/keyFields/" + strconv.Itoa(index)
		if keyField.Comparison != "exact" {
			invalid(fieldPath + "/comparison")
			keyFieldsValid = false
			continue
		}
		if seenKeyField[keyField.RequestPath] {
			invalid(fieldPath + "/requestPath")
			keyFieldsValid = false
			continue
		}
		seenKeyField[keyField.RequestPath] = true
		dataPointer, ok := v1alpha5DataPointer(keyField.RequestPath)
		if !ok {
			invalid(fieldPath + "/requestPath")
			keyFieldsValid = false
			continue
		}
		result, err := link.resolution.LookupSchemaPath(normalizedRoot, dataPointer)
		if err != nil {
			invalid(fieldPath + "/requestPath")
			keyFieldsValid = false
			continue
		}
		if !v1alpha5SchemaIsScalar(result.Schema) {
			invalid(fieldPath + "/requestPath")
			keyFieldsValid = false
		}
	}
	diagnostics = append(diagnostics, v1alpha5ValidateStaleDelivery(export, normalizedRoot, link)...)

	// Binding validation depends on a complete and valid key-field set; a failed
	// key-field prerequisite suppresses the dependent binding diagnostics.
	if keyFieldsValid && contract.StartAllowed {
		diagnostics = append(diagnostics, v1alpha5ValidateArmingBindings(export, contract.KeyFields,
			contract.StartBindings, "/arming/start/bindings", "invocation-context", armInputRoot, link)...)
	}
	for index, trigger := range export.Operation.Arming.Triggers {
		fieldPath := "/arming/triggers/" + strconv.Itoa(index)
		if trigger.Operation.Kind != "operation" {
			invalid(fieldPath + "/operation/kind")
			continue
		}
		resolved, triggerDiagnostics := v1alpha5ResolveTriggerOperation(export, trigger, link, scope, triggerIndexes)
		diagnostics = append(diagnostics, triggerDiagnostics...)
		if resolved == nil {
			continue
		}
		contract.Triggers = append(contract.Triggers, *resolved)
		if keyFieldsValid {
			diagnostics = append(diagnostics, v1alpha5ValidateArmingBindings(export, contract.KeyFields,
				trigger.Bindings, fieldPath+"/bindings", "operation-argument", armInputRoot, link)...)
		}
	}
	if hasErrorDiagnostic(diagnostics) {
		return nil, diagnostics
	}
	return contract, diagnostics
}

// v1alpha5ValidateStaleDelivery enforces the #62 §7.2/§7.3 static safety rules
// that the structural Catalog schema cannot express: the declared strategy, the
// exact fixed-duration retention window and the strict retention ordering.
func v1alpha5ValidateStaleDelivery(export v1alpha5TrustedIngressExport, normalizedRoot ingresscatalog.RootRef, link *v1alpha5LinkContext) []Diagnostic {
	stale := export.Operation.Correlation.StaleDelivery
	invalid := func(fieldPath string) []Diagnostic {
		return []Diagnostic{v1alpha5IngressCorrelationDiagnostic(export, fieldPath)}
	}
	switch stale.Strategy {
	case "single-flight-tombstone":
		if _, ok := v1alpha5DataPointer(stale.ResourceKey); !ok {
			return invalid("/correlation/staleDelivery/resourceKey")
		}
		resourcePointer, _ := v1alpha5DataPointer(stale.ResourceKey)
		result, err := link.resolution.LookupSchemaPath(normalizedRoot, resourcePointer)
		if err != nil || !v1alpha5SchemaIsScalar(result.Schema) {
			return invalid("/correlation/staleDelivery/resourceKey")
		}
		if stale.BasisRef == "" {
			return invalid("/correlation/staleDelivery/basisRef")
		}
		return v1alpha5ValidateRetentionWindow(export,
			stale.MaximumLateArrival, stale.ClockSkewAllowance, stale.TombstoneRetention,
			"/correlation/staleDelivery/maximumLateArrival",
			"/correlation/staleDelivery/clockSkewAllowance",
			"/correlation/staleDelivery/tombstoneRetention")
	case "wire-discriminator":
		if stale.BasisRef == "" {
			return invalid("/correlation/staleDelivery/basisRef")
		}
		switch stale.DiscriminatorKind {
		case "normalized-request-path":
			discriminatorPointer, ok := v1alpha5DataPointer(stale.DiscriminatorPath)
			if !ok {
				return invalid("/correlation/staleDelivery/discriminator/path")
			}
			result, err := link.resolution.LookupSchemaPath(normalizedRoot, discriminatorPointer)
			if err != nil || !v1alpha5SchemaIsScalar(result.Schema) {
				return invalid("/correlation/staleDelivery/discriminator/path")
			}
		case "authenticated-header-parameter":
		default:
			return invalid("/correlation/staleDelivery/discriminator/kind")
		}
		return v1alpha5ValidateRetentionWindow(export,
			stale.MaximumLateArrival, stale.ClockSkewAllowance, stale.DiscriminatorRetention,
			"/correlation/staleDelivery/maximumLateArrival",
			"/correlation/staleDelivery/clockSkewAllowance",
			"/correlation/staleDelivery/discriminatorRetention")
	default:
		return invalid("/correlation/staleDelivery/strategy")
	}
}

// v1alpha5ValidateRetentionWindow enforces that every declared window is a
// positive fixed ISO-8601 duration in the v1alpha5 lexical subset and that the
// retention strictly exceeds maximumLateArrival + clockSkewAllowance (#62
// §7.2/§7.3: equality is not accepted).
func v1alpha5ValidateRetentionWindow(export v1alpha5TrustedIngressExport, maximumLateArrival, clockSkewAllowance, retention, maximumPath, skewPath, retentionPath string) []Diagnostic {
	maximum, ok := v1alpha5DurationSeconds(maximumLateArrival)
	if !ok {
		return []Diagnostic{v1alpha5IngressCorrelationDiagnostic(export, maximumPath)}
	}
	skew, ok := v1alpha5DurationSeconds(clockSkewAllowance)
	if !ok {
		return []Diagnostic{v1alpha5IngressCorrelationDiagnostic(export, skewPath)}
	}
	retained, ok := v1alpha5DurationSeconds(retention)
	if !ok {
		return []Diagnostic{v1alpha5IngressCorrelationDiagnostic(export, retentionPath)}
	}
	threshold := new(big.Int).Add(maximum, skew)
	if retained.Cmp(threshold) <= 0 {
		return []Diagnostic{v1alpha5IngressCorrelationDiagnostic(export, retentionPath)}
	}
	return nil
}

// v1alpha5DurationSeconds converts one accepted fixed-duration lexical value into
// exact integer seconds. Arbitrarily large positive components are supported; no
// deployment maximum is applied here (that is Host admission).
func v1alpha5DurationSeconds(value string) (*big.Int, bool) {
	if !validV1Alpha5FixedDuration(value) {
		return nil, false
	}
	matches := v1alpha5WaitDurationPattern.FindStringSubmatch(value)
	units := []struct {
		index  int
		suffix string
		factor int64
	}{{1, "D", 86400}, {3, "H", 3600}, {4, "M", 60}, {5, "S", 1}}
	total := new(big.Int)
	for _, unit := range units {
		if matches[unit.index] == "" {
			continue
		}
		componentText := strings.TrimSuffix(matches[unit.index], unit.suffix)
		component, ok := new(big.Int).SetString(componentText, 10)
		if !ok {
			return nil, false
		}
		component.Mul(component, big.NewInt(unit.factor))
		total.Add(total, component)
	}
	return total, true
}

// v1alpha5ValidateArmingBindings proves that every correlation key field has
// exactly one exact arming binding, that the binding target exists in the typed
// armInputSchema as a scalar, and that the declared binding source kind is the
// exact mode-appropriate kind.
func v1alpha5ValidateArmingBindings(export v1alpha5TrustedIngressExport, keyFields []ingresscatalog.CorrelationKeyProjection, bindings []ingresscatalog.ArmingBindingProjection, fieldPath, sourceKind string, armInputRoot ingresscatalog.RootRef, link *v1alpha5LinkContext) []Diagnostic {
	invalid := func(path string) []Diagnostic {
		return []Diagnostic{v1alpha5IngressCorrelationDiagnostic(export, path)}
	}
	bindingFor := make(map[string]int, len(bindings))
	for index, binding := range bindings {
		if _, exists := bindingFor[binding.RequestKey]; exists {
			return invalid(fieldPath + "/" + strconv.Itoa(index) + "/requestKey")
		}
		bindingFor[binding.RequestKey] = index
	}
	allowed := make(map[string]bool, len(keyFields))
	for index, keyField := range keyFields {
		allowed[keyField.RequestPath] = true
		if _, covered := bindingFor[keyField.RequestPath]; !covered {
			return invalid(fieldPath + "/" + strconv.Itoa(index))
		}
	}
	for index, binding := range bindings {
		if !allowed[binding.RequestKey] {
			return invalid(fieldPath + "/" + strconv.Itoa(index) + "/requestKey")
		}
		if binding.SourceKind != sourceKind {
			return invalid(fieldPath + "/" + strconv.Itoa(index) + "/source/kind")
		}
		if _, ok := v1alpha5DataPointer(binding.SourcePath); !ok {
			return invalid(fieldPath + "/" + strconv.Itoa(index) + "/source/path")
		}
		bindingPointer, ok := v1alpha5DataPointer(binding.BindingPath)
		if !ok {
			return invalid(fieldPath + "/" + strconv.Itoa(index) + "/bindingPath")
		}
		result, err := link.resolution.LookupSchemaPath(armInputRoot, bindingPointer)
		if err != nil || !v1alpha5SchemaIsScalar(result.Schema) {
			return invalid(fieldPath + "/" + strconv.Itoa(index) + "/bindingPath")
		}
	}
	return nil
}

// v1alpha5SnapshotTriggerIndex caches the ToolCatalog/Schema export index of one
// verified dependency snapshot for trigger Operation resolution.
type v1alpha5SnapshotTriggerIndex struct {
	index       snapshotExportIndex
	diagnostics []Diagnostic
}

// v1alpha5ResolveTriggerOperation resolves one declared trigger identity to
// exactly one trusted ToolCatalog Operation export using the inherited v1alpha4
// export index/Owner-attribution machinery. No second ToolCatalog parser is
// introduced.
func v1alpha5ResolveTriggerOperation(export v1alpha5TrustedIngressExport, trigger ingresscatalog.TriggerProjection, link *v1alpha5LinkContext, scope []v1alpha5Dependency, triggerIndexes map[string]*v1alpha5SnapshotTriggerIndex) (*v1alpha5ResolvedTrigger, []Diagnostic) {
	var diagnostics []Diagnostic
	missing := func(message string) {
		diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(export.SnapshotPath, nil, "EXPORT_ID_MISSING", message))
	}
	ambiguous := func(message string) {
		diagnostics = append(diagnostics, v1alpha5StaticDiagnostic(export.SnapshotPath, nil, "EXPORT_ID_AMBIGUOUS", message))
	}

	var dependencies []v1alpha5Dependency
	for _, dependency := range scope {
		if dependency.PackageID == trigger.Operation.PackageID {
			dependencies = append(dependencies, dependency)
		}
	}
	switch len(dependencies) {
	case 0:
		missing("Triggering Operation '" + trigger.Operation.ID + "' does not resolve in a declared trusted dependency snapshot with Package ID '" + trigger.Operation.PackageID + "'.")
		return nil, diagnostics
	case 1:
	default:
		ambiguous("Triggering Operation '" + trigger.Operation.ID + "' is claimed by more than one trusted dependency snapshot with Package ID '" + trigger.Operation.PackageID + "'.")
		return nil, diagnostics
	}
	dependency := dependencies[0]

	exportID, exportIDOK := v1alpha5TriggerExportID(dependency.Alias, trigger.Operation.ID)
	if !exportIDOK {
		missing("Triggering Operation identity '" + trigger.Operation.ID + "' has no exact trustable ToolCatalog Operation export ID.")
		return nil, diagnostics
	}

	cached, exists := triggerIndexes[dependency.Alias]
	if !exists {
		index, _, indexDiagnostics := buildSnapshotExportIndex(dependency.Snapshot)
		cached = &v1alpha5SnapshotTriggerIndex{index: index, diagnostics: indexDiagnostics}
		triggerIndexes[dependency.Alias] = cached
	}
	diagnostics = append(diagnostics, cached.diagnostics...)

	records := cached.index.callables[exportID]
	if len(records) == 0 {
		missing("Triggering Operation '" + trigger.Operation.ID + "' does not exist exactly once as a ToolCatalog export in the trusted dependency snapshot of Package ID '" + trigger.Operation.PackageID + "'.")
		return nil, diagnostics
	}
	if len(records) > 1 {
		// The duplicate identity is already reported by buildSnapshotExportIndex.
		return nil, diagnostics
	}
	matched := records[0]
	if matched.ExportKind != "operation" {
		missing("Triggering Operation '" + trigger.Operation.ID + "' does not resolve to a ToolCatalog Operation export.")
		return nil, diagnostics
	}
	owner, ownershipDiagnostics := attributeExportOwner(matched, dependency.Binding, nil)
	diagnostics = append(diagnostics, ownershipDiagnostics...)
	if len(ownershipDiagnostics) > 0 {
		return nil, diagnostics
	}
	if owner != trigger.Operation.Owner {
		missing("Triggering Operation '" + trigger.Operation.ID + "' is not owned by the declared trusted Owner '" + trigger.Operation.Owner + "'.")
		return nil, diagnostics
	}
	if _, bound := link.schemaRoots[matched.InputSchemaID]; !bound {
		diagnostics = append(diagnostics, v1alpha5SchemaDiagnostic(export.SnapshotPath, "SCHEMA_MISSING",
			"Triggering Operation '"+trigger.Operation.ID+"' Input Schema '"+matched.InputSchemaID+"' is not declared by the trusted v1alpha5 schema bindings."))
		return nil, diagnostics
	}
	return &v1alpha5ResolvedTrigger{
		Declared:      trigger,
		PackageID:     dependency.PackageID,
		ExportID:      exportID,
		Owner:         owner,
		InputSchemaID: matched.InputSchemaID,
	}, diagnostics
}

// v1alpha5TriggerExportID maps one declared Trigger Operation ID to the exact
// package-local ToolCatalog Operation export ID. The candidate contract shows
// both the package-local export ID and the alias-qualified author reference
// ("<dependencyAlias>::<exportID>"); both are matched exactly and the qualified
// form must name the already-matched dependency alias. No other form is guessed.
func v1alpha5TriggerExportID(alias, declaredID string) (string, bool) {
	if validExportID(declaredID) {
		return declaredID, true
	}
	index := strings.Index(declaredID, "::")
	if index <= 0 {
		return "", false
	}
	if declaredID[:index] != alias {
		return "", false
	}
	exportID := declaredID[index+2:]
	if !validExportID(exportID) {
		return "", false
	}
	return exportID, true
}

// v1alpha5ValidateWaitEntryPaths enforces the per-incoming-path trigger
// dominance and pre-dispatch materialization obligations (Appendix L.4.2, #62
// §6.3). It reports at most one WAIT_CORRELATION_BINDING_INVALID per Wait Node,
// anchored at wait.event.
func v1alpha5ValidateWaitEntryPaths(procedurePath, nodeID string, wait *V1Alpha5WaitAST, graph *V1Alpha5ProcedureGraph, reachable map[string]bool, skill *V1Alpha5SkillAST, contract *v1alpha5ExportContract, scopeByAlias map[string]v1alpha5Dependency) []Diagnostic {
	invalid := func() []Diagnostic {
		return []Diagnostic{v1alpha5StaticDiagnostic(procedurePath, wait.Event.Raw, "WAIT_CORRELATION_BINDING_INVALID",
			fmt.Sprintf("Node '%s' cannot derive one exact correlation binding on every reachable entry path.", nodeID))}
	}
	if nodeID == graph.StartNode() {
		// A start-Node Wait is armed from the already-admitted Invocation Context.
		if !contract.StartAllowed {
			return invalid()
		}
		return nil
	}
	entryID := graph.WaitEntryVertex(nodeID)
	if entryID == "" {
		return nil
	}
	predecessors := make([]string, 0)
	seen := make(map[string]bool)
	for _, edge := range graph.EdgesTo(entryID) {
		if seen[edge.From] || !reachable[edge.From] {
			continue
		}
		seen[edge.From] = true
		predecessors = append(predecessors, edge.From)
	}
	sort.Strings(predecessors)
	for _, predecessor := range predecessors {
		triggerNodeID, triggers, ok := v1alpha5DominatingTrigger(skill, graph, predecessor, contract, scopeByAlias)
		if !ok {
			return invalid()
		}
		triggerNode := skill.Procedure.ByID[triggerNodeID]
		for _, trigger := range triggers {
			if !v1alpha5TriggerBindingsMaterializable(triggerNode, trigger) {
				return invalid()
			}
		}
	}
	return nil
}

// v1alpha5DominatingTrigger returns the unique declared-trigger Operation Node
// that dominates one incoming path predecessor, and the exact declared triggers
// its tool reference matches. A path with no dominating trigger or with more
// than one dominating trigger cannot derive one exact binding.
func v1alpha5DominatingTrigger(skill *V1Alpha5SkillAST, graph *V1Alpha5ProcedureGraph, predecessor string, contract *v1alpha5ExportContract, scopeByAlias map[string]v1alpha5Dependency) (string, []v1alpha5ResolvedTrigger, bool) {
	if skill == nil || skill.Procedure == nil || len(contract.Triggers) == 0 {
		return "", nil, false
	}
	candidates := make([]string, 0, 1)
	for _, candidateID := range sortedStringMapKeys(skill.Procedure.ByID) {
		node := skill.Procedure.ByID[candidateID]
		if node == nil || node.Operation == nil {
			continue
		}
		if _, matches := v1alpha5NodeTriggerMatches(node, contract, scopeByAlias); len(matches) == 0 {
			continue
		}
		if graph.Dominates(candidateID, predecessor) {
			candidates = append(candidates, candidateID)
		}
	}
	if len(candidates) != 1 {
		return "", nil, false
	}
	_, triggers := v1alpha5NodeTriggerMatches(skill.Procedure.ByID[candidates[0]], contract, scopeByAlias)
	return candidates[0], triggers, true
}

// v1alpha5NodeTriggerMatches reports the declared triggers whose exact trusted
// Package ID and Operation export identity match this Node's operation tool
// reference.
func v1alpha5NodeTriggerMatches(node *V1Alpha5ProcedureNodeAST, contract *v1alpha5ExportContract, scopeByAlias map[string]v1alpha5Dependency) (string, []v1alpha5ResolvedTrigger) {
	alias, exportID, ok := splitQualifiedReference(child(node.Operation, "tool"))
	if !ok {
		return "", nil
	}
	dependency, exists := scopeByAlias[alias]
	if !exists {
		return "", nil
	}
	var matches []v1alpha5ResolvedTrigger
	for _, trigger := range contract.Triggers {
		if trigger.PackageID == dependency.PackageID && trigger.ExportID == exportID {
			matches = append(matches, trigger)
		}
	}
	return alias, matches
}

// v1alpha5TriggerBindingsMaterializable proves that every declared trigger
// binding value is materialized by the triggering Node's operation arguments
// before dispatch and does not depend on the trigger Operation's own output,
// which is unavailable until after the external action may have caused the event
// (Appendix L.4.2).
func v1alpha5TriggerBindingsMaterializable(triggerNode *V1Alpha5ProcedureNodeAST, trigger v1alpha5ResolvedTrigger) bool {
	if triggerNode == nil {
		return false
	}
	arguments := child(triggerNode.Operation, "arguments")
	for _, binding := range trigger.Declared.Bindings {
		value, ok := v1alpha5ArgumentValue(arguments, binding.SourcePath)
		if !ok {
			return false
		}
		outputPrefix := triggerNode.ID + ".operation.output"
		if value == outputPrefix || strings.HasPrefix(value, outputPrefix+".") {
			return false
		}
	}
	return true
}

// v1alpha5ArgumentValue resolves one arming source path against the materialized
// arguments mapping of one Operation stage and returns the argument expression.
func v1alpha5ArgumentValue(arguments *yaml.Node, sourcePath string) (string, bool) {
	pointer, ok := v1alpha5DataPointer(sourcePath)
	if !ok {
		return "", false
	}
	tokens := strings.Split(strings.TrimPrefix(pointer, "#/"), "/")
	current := arguments
	for _, token := range tokens {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		current = child(current, token)
		if current == nil {
			return "", false
		}
	}
	value, isString := v1alpha5String(current)
	if !isString {
		return "", false
	}
	return value, true
}

// v1alpha5DataPointer converts one Catalog JSON Pointer field value ("/a/b") into
// the canonical artifact/Schema pointer form ("#/a/b") used by the trusted Schema
// lookup. Both forms are the same RFC 6901 pointer; only the leading marker
// differs.
func v1alpha5DataPointer(path string) (string, bool) {
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", false
	}
	pointer := "#" + path
	if !validCanonicalRootPointer(pointer) {
		return "", false
	}
	return pointer, true
}

// v1alpha5SchemaIsScalar reports whether one resolved Schema Object describes a
// stably comparable scalar correlation value: exactly one of the JSON scalar
// types string/integer/number/boolean, or an enum/const made only of scalars when
// no type is declared. Multi-type Schemas are rejected because the comparison
// type would be ambiguous.
func v1alpha5SchemaIsScalar(schema any) bool {
	object, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	if rawType, exists := object["type"]; exists {
		name, isString := rawType.(string)
		if !isString {
			return false
		}
		switch name {
		case "string", "integer", "number", "boolean":
			return true
		default:
			return false
		}
	}
	if rawEnum, exists := object["enum"]; exists {
		values, isArray := rawEnum.([]any)
		if !isArray || len(values) == 0 {
			return false
		}
		for _, value := range values {
			if !v1alpha5JSONScalar(value) {
				return false
			}
		}
		return true
	}
	if value, exists := object["const"]; exists {
		return v1alpha5JSONScalar(value)
	}
	return false
}

func v1alpha5JSONScalar(value any) bool {
	switch value.(type) {
	case string, bool, int64, float64:
		return true
	default:
		return false
	}
}
