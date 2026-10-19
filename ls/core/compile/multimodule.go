// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package compile

import (
	stdcontext "context"
	"sort"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// registeredNames retains registration keys per DiagnosticEnv for the process
// lifetime, including environments replaced by reloads.
var (
	registeredNamesMu sync.Mutex
	registeredNames   = make(map[*diagnostics.DiagnosticEnv]map[string]bool)
)

// alreadyRegistered reports whether name has already been registered with
// env by a previous ensureParsed call (this or an earlier generation).
func alreadyRegistered(env *diagnostics.DiagnosticEnv, name string) bool {
	registeredNamesMu.Lock()
	defer registeredNamesMu.Unlock()
	return registeredNames[env][name]
}

// markRegistered records that name has been registered with env.
func markRegistered(env *diagnostics.DiagnosticEnv, name string) {
	registeredNamesMu.Lock()
	defer registeredNamesMu.Unlock()
	names, ok := registeredNames[env]
	if !ok {
		names = make(map[string]bool)
		registeredNames[env] = names
	}
	names[name] = true
}

// migratedLangLibs lists the bundled libraries loaded for implicit imports.
var migratedLangLibs = []string{
	"lang.__internal", "lang.int", "lang.boolean", "lang.decimal", "lang.error",
	"lang.string", "lang.value", "lang.xml", "lang.float", "lang.array",
	"lang.map", "lang.object",
}

// seedMigratedLangLibs adds available bundled-library symbols to implicitImports.
func seedMigratedLangLibs(implicitImports map[string]model.ExportedSymbolSpace, publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace) {
	for _, name := range migratedLangLibs {
		if space, ok := publicSymbols[semantics.PackageIdentifier{OrgName: "ballerina", ModuleName: name}]; ok {
			implicitImports[name] = space
		}
	}
}

// packageIdentifierFor returns the module's organization/name symbol-space key.
func packageIdentifierFor(module *projects.Module) semantics.PackageIdentifier {
	desc := module.Descriptor()
	return semantics.PackageIdentifier{OrgName: desc.Org().Value(), ModuleName: desc.Name().String()}
}

// packageDriver runs dependency-ordered Phase 1 before sequential Phase 2.
// External dependencies stop at top-level type resolution. Unchanged modules
// outside resetScope reuse prior drivers; stale, when set, checks supersession.
type packageDriver struct {
	pkg *projects.Package
	env *context.CompilerEnvironment

	drivers       map[projects.ModuleID]*moduleDriver
	publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace
	// externalProjections retains dependencies that reached symbol resolution,
	// including diagnosed modules, for feature queries rather than resolution.
	externalProjections   map[semantics.PackageIdentifier]ExternalModuleProjection
	externalMaterializing map[semantics.PackageIdentifier]bool
	externalSymbolsReady  bool
	dependencySnapshots   map[projects.ModuleID]moduleGenSnapshot
	phase1Errored         map[projects.ModuleID]bool

	openText   textProvider
	priorState *packageGenState
	stale      func() bool

	resetScopeReady bool
	resetScope      map[projects.ModuleID]bool
	directlyDirty   map[projects.ModuleID]bool
	textHashes      map[projects.ModuleID]string
	fingerprints    map[projects.ModuleID]fingerprint
	reverseDeps     map[projects.ModuleID][]projects.ModuleID

	order []*projects.Module
}

// newPackageDriver shares pkg's compiler environment and creates run-local
// maps. openText, priorState, and stale may be nil.
func newPackageDriver(pkg *projects.Package, openText textProvider, priorState *packageGenState, stale func() bool) *packageDriver {
	return &packageDriver{
		pkg:                   pkg,
		env:                   pkg.Project().Environment().CompilerEnvironment(),
		drivers:               make(map[projects.ModuleID]*moduleDriver),
		publicSymbols:         make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace),
		externalProjections:   make(map[semantics.PackageIdentifier]ExternalModuleProjection),
		externalMaterializing: make(map[semantics.PackageIdentifier]bool),
		dependencySnapshots:   make(map[projects.ModuleID]moduleGenSnapshot),
		phase1Errored:         make(map[projects.ModuleID]bool),
		openText:              openText,
		priorState:            priorState,
		stale:                 stale,
	}
}

// aborted reports whether the run's supersession check is set and true.
func (pd *packageDriver) aborted() bool {
	return pd.stale != nil && pd.stale()
}

// topoModules caches the dependency-ordered modules found in pkg.
func (pd *packageDriver) topoModules() []*projects.Module {
	if pd.order != nil {
		return pd.order
	}
	descs := pd.pkg.Resolution().ModuleDependencyGraph().ToTopologicallySortedList()
	modules := make([]*projects.Module, 0, len(descs))
	for _, desc := range descs {
		module := pd.pkg.ModuleByName(desc.Name())
		if module == nil {
			continue
		}
		modules = append(modules, module)
	}
	pd.order = modules
	return modules
}

// driverFor returns this generation's moduleDriver for module, creating one
// (with a fresh CompilerContext over the shared environment) on first use.
func (pd *packageDriver) driverFor(module *projects.Module) *moduleDriver {
	id := module.ModuleID()
	if d, ok := pd.drivers[id]; ok {
		return d
	}
	d := newModuleDriver(pd.env, pd.openText, pd.stale)
	pd.drivers[id] = d
	return d
}

// resolutionInputFor supplies the run's published symbols and implicit imports.
func (pd *packageDriver) resolutionInputFor(module *projects.Module) moduleResolutionInput {
	implicit := make(map[string]model.ExportedSymbolSpace)
	seedMigratedLangLibs(implicit, pd.publicSymbols)
	return newModuleResolutionInput(module.Descriptor().Org().Value(), implicit, pd.publicSymbols)
}

// loadImplicitLangLibs materializes bundled libraries before local resolution,
// excluding the root package itself.
func (pd *packageDriver) loadImplicitLangLibs() {
	root := pd.pkg.Descriptor()
	for _, name := range migratedLangLibs {
		if pd.aborted() {
			return
		}
		if root.Org().Value() == "ballerina" && root.Name().Value() == name {
			continue
		}
		descriptor, err := projects.NewPackageDescriptorFromStrings("ballerina", name, "0.0.1")
		if err != nil {
			continue
		}
		pd.loadResolvedDependency(descriptor)
	}
}

func (pd *packageDriver) loadResolvedDependencySymbols() {
	if pd.externalSymbolsReady {
		return
	}
	pd.externalSymbolsReady = true
	dependencies := pd.pkg.Resolution().ResolvedDependencies()
	identifiers := make([]string, 0, len(dependencies))
	for identifier := range dependencies {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)
	for _, identifier := range identifiers {
		if pd.aborted() {
			return
		}
		descriptor := dependencies[identifier]
		if descriptor == nil {
			continue
		}
		pd.loadResolvedDependency(*descriptor)
	}
}

// loadResolvedDependency materializes a dependency's Phase-1 symbols and
// completion projection. Resolution failures and panics return without
// aborting the local cycle.
func (pd *packageDriver) loadResolvedDependency(descriptor projects.PackageDescriptor) {
	packageID := semantics.PackageIdentifier{OrgName: descriptor.Org().Value(), ModuleName: descriptor.Name().String()}
	if pd.externalMaterializing[packageID] {
		return
	}
	pd.externalMaterializing[packageID] = true
	defer func() {
		if recover() != nil {
			return
		}
	}()

	resolver := pd.pkg.Project().Environment().PackageResolver()
	options := pd.pkg.Project().Environment().ResolutionOptions()
	responses := resolver.ResolvePackages(stdcontext.Background(), []projects.ResolutionRequest{projects.NewResolutionRequest(descriptor)}, options)
	if len(responses) != 1 || !responses[0].IsResolved() || responses[0].Package() == nil {
		return
	}
	dependencyPackage := responses[0].Package()
	dependencyDriver := newPackageDriver(dependencyPackage, nil, pd.dependencyPriorState(dependencyPackage), pd.stale)
	for identifier, symbols := range pd.publicSymbols {
		dependencyDriver.publicSymbols[identifier] = symbols
	}
	dependencyDriver.externalMaterializing = pd.externalMaterializing
	dependencyDriver.loadResolvedDependencySymbols()
	if pd.aborted() || dependencyDriver.aborted() {
		return
	}
	dependencyDriver.runPhase1(dependencyDriver.topoModules())
	if pd.aborted() || dependencyDriver.aborted() {
		return
	}
	for id, snapshot := range dependencyDriver.snapshotForCommit() {
		pd.dependencySnapshots[id] = snapshot
	}
	for _, module := range dependencyDriver.topoModules() {
		driver, ok := dependencyDriver.drivers[module.ModuleID()]
		if !ok || driver.currentStage() < stageSymbolResolved {
			continue
		}
		identifier := packageIdentifierFor(module)
		pd.externalProjections[identifier] = ExternalModuleProjection{PackageNode: driver.pkgNode, Symbols: driver.exported}
		if driver.symbolResolutionUsable {
			pd.publicSymbols[identifier] = driver.exported
		}
	}
}

// dependencyPriorState selects prior snapshots by pkg's module IDs.
func (pd *packageDriver) dependencyPriorState(pkg *projects.Package) *packageGenState {
	if pd.priorState == nil {
		return nil
	}
	modules := make(map[projects.ModuleID]moduleGenSnapshot)
	for _, id := range pkg.ModuleIDs() {
		if snapshot, ok := pd.priorState.modules[id]; ok {
			modules[id] = snapshot
		}
	}
	return &packageGenState{modules: modules}
}

// dependencyErrored reports whether a direct same-package dependency failed
// Phase 1.
func (pd *packageDriver) dependencyErrored(module *projects.Module, depGraph *projects.DependencyGraph[projects.ModuleDescriptor]) bool {
	for _, depDesc := range depGraph.DirectDependencies(module.Descriptor()) {
		depModule := pd.pkg.ModuleByName(depDesc.Name())
		if depModule == nil {
			continue // external dependency: not tracked by phase1Errored
		}
		if pd.phase1Errored[depModule.ModuleID()] {
			return true
		}
	}
	return false
}

// ensureResetScope caches document hashes and the modules to rebuild. Without
// prior state it resets every supplied module; otherwise it uses a topological
// suffix for a complete module list or a dependent closure for a partial list.
func (pd *packageDriver) ensureResetScope(modules []*projects.Module, depGraph *projects.DependencyGraph[projects.ModuleDescriptor]) {
	if pd.resetScopeReady {
		return
	}
	pd.resetScopeReady = true

	pd.textHashes = make(map[projects.ModuleID]string, len(modules))
	for _, m := range modules {
		pd.textHashes[m.ModuleID()] = moduleTextHash(m, pd.openText)
	}

	if pd.priorState == nil {
		pd.resetScope = allModuleIDs(modules)
		pd.directlyDirty = pd.resetScope
		return
	}

	dirty := make(map[projects.ModuleID]bool)
	for _, m := range modules {
		id := m.ModuleID()
		prev, ok := pd.priorState.modules[id]
		if !ok || prev.textHash != pd.textHashes[id] {
			dirty[id] = true
		}
	}
	pd.directlyDirty = dirty

	if len(dirty) == 0 {
		pd.resetScope = make(map[projects.ModuleID]bool)
		return
	}
	if len(modules) == len(pd.pkg.ModuleIDs()) {
		pd.resetScope = topoTailScope(modules, dirty)
	} else {
		pd.reverseDeps = reverseDependents(pd.pkg, depGraph)
		pd.resetScope = dependentClosure(dirty, pd.reverseDeps)
	}
	if pd.reverseDeps == nil {
		pd.reverseDeps = reverseDependents(pd.pkg, depGraph)
	}
}

// adoptCarriedForward restores a prior driver and its Phase-1 status, publishing
// exports only for a non-errored snapshot. It returns false if no snapshot exists.
func (pd *packageDriver) adoptCarriedForward(module *projects.Module) bool {
	if pd.priorState == nil {
		return false
	}
	id := module.ModuleID()
	snap, ok := pd.priorState.modules[id]
	if !ok {
		return false
	}
	pd.drivers[id] = snap.driver
	if snap.errored {
		pd.phase1Errored[id] = true
		return true
	}
	pd.publicSymbols[packageIdentifierFor(module)] = snap.exported
	return true
}

// recordFingerprintAndPrune records module's public-API fingerprint. If unchanged,
// it removes direct dependents from resetScope unless they are directly dirty.
func (pd *packageDriver) recordFingerprintAndPrune(module *projects.Module, d *moduleDriver) {
	id := module.ModuleID()
	fp := computeFingerprint(d.diagnosticContext(), d.exported)
	if pd.fingerprints == nil {
		pd.fingerprints = make(map[projects.ModuleID]fingerprint)
	}
	pd.fingerprints[id] = fp

	if pd.priorState == nil {
		return
	}
	prev, ok := pd.priorState.modules[id]
	if !ok || !fp.equal(prev.fingerprint) {
		return
	}
	if pd.reverseDeps == nil {
		return
	}
	for _, dependent := range pd.reverseDeps[id] {
		if pd.directlyDirty[dependent] {
			continue
		}
		delete(pd.resetScope, dependent)
	}
}

// runPhase1Module adopts prior state or advances through top-level type
// resolution. A failed direct dependency skips the module. Usable symbols
// are published before top-level type resolution; a later error leaves them
// published but marks the module errored for dependent gating.
func (pd *packageDriver) runPhase1Module(module *projects.Module, depGraph *projects.DependencyGraph[projects.ModuleDescriptor]) {
	id := module.ModuleID()
	if !pd.resetScope[id] && pd.adoptCarriedForward(module) {
		return
	}
	if pd.dependencyErrored(module, depGraph) {
		pd.phase1Errored[id] = true
		return
	}

	d := pd.driverFor(module)
	input := pd.resolutionInputFor(module)

	d.advanceTo(stageSymbolResolved, module, input)
	if d.currentStage() < stageSymbolResolved || !d.symbolResolutionUsable {
		pd.markPhase1Errored(module, d, input)
		return
	}
	pd.publicSymbols[packageIdentifierFor(module)] = d.exported

	d.advanceTo(stageTopLevelTypeResolved, module, input)
	if d.currentStage() < stageTopLevelTypeResolved || !d.topLevelTypeResolutionUsable {
		pd.markPhase1Errored(module, d, input)
		return
	}
	pd.recordFingerprintAndPrune(module, d)
}

// markPhase1Errored marks module for dependent gating and attempts to finish
// its top-level type resolution. Already-published symbols are retained.
func (pd *packageDriver) markPhase1Errored(module *projects.Module, d *moduleDriver, input moduleResolutionInput) {
	pd.phase1Errored[module.ModuleID()] = true
	d.advanceTo(stageTopLevelTypeResolved, module, input)
}

// runPhase1 advances modules in the supplied order, checking supersession
// between modules.
func (pd *packageDriver) runPhase1(modules []*projects.Module) {
	depGraph := pd.pkg.Resolution().ModuleDependencyGraph()
	pd.ensureResetScope(modules, depGraph)
	for _, module := range modules {
		if pd.aborted() {
			return
		}
		pd.runPhase1Module(module, depGraph)
	}
}

// phase2Eligible returns a driver that reached top-level type resolution,
// including one with diagnostics.
func (pd *packageDriver) phase2Eligible(module *projects.Module) (*moduleDriver, bool) {
	id := module.ModuleID()
	d, ok := pd.drivers[id]
	if !ok || d.currentStage() < stageTopLevelTypeResolved {
		return nil, false
	}
	return d, true
}

// advanceAll runs Phase 1 package-wide, then advances eligible modules to
// target sequentially. Callers must check aborted before committing results.
func (pd *packageDriver) advanceAll(target moduleStage) {
	pd.loadImplicitLangLibs()
	pd.loadResolvedDependencySymbols()
	modules := pd.topoModules()
	pd.runPhase1(modules)
	if pd.aborted() {
		return
	}
	for _, module := range modules {
		if pd.aborted() {
			return
		}
		d, ok := pd.phase2Eligible(module)
		if !ok {
			continue
		}
		d.advanceTo(target, module, pd.resolutionInputFor(module))
	}
}

// advanceModule runs Phase 1 through targetModule's topological prefix, then
// advances only targetModule to target.
func (pd *packageDriver) advanceModule(target moduleStage, targetModule *projects.Module) {
	pd.loadImplicitLangLibs()
	pd.loadResolvedDependencySymbols()
	depGraph := pd.pkg.Resolution().ModuleDependencyGraph()
	targetID := targetModule.ModuleID()
	for _, module := range pd.topoModules() {
		if pd.aborted() {
			return
		}
		pd.runPhase1Module(module, depGraph)
		if module.ModuleID() == targetID {
			break
		}
	}
	d, ok := pd.phase2Eligible(targetModule)
	if !ok {
		return
	}
	d.advanceTo(target, targetModule, pd.resolutionInputFor(targetModule))
}

// allDiagnostics collects publishable diagnostics in local module dependency
// order, including diagnostics from carried-forward drivers.
func (pd *packageDriver) allDiagnostics() []diagnostics.Diagnostic {
	var all []diagnostics.Diagnostic
	for _, module := range pd.topoModules() {
		d, ok := pd.drivers[module.ModuleID()]
		if !ok {
			continue
		}
		all = append(all, d.publishableDiagnostics()...)
	}
	return all
}

// snapshotForCommit collects reached local and external module state, retaining
// prior fingerprints for adopted modules. Callers must reject aborted runs.
func (pd *packageDriver) snapshotForCommit() map[projects.ModuleID]moduleGenSnapshot {
	out := make(map[projects.ModuleID]moduleGenSnapshot, len(pd.dependencySnapshots)+len(pd.drivers))
	for id, snapshot := range pd.dependencySnapshots {
		out[id] = snapshot
	}
	for _, module := range pd.topoModules() {
		id := module.ModuleID()
		d, ok := pd.drivers[id]
		if !ok {
			continue
		}
		snap := moduleGenSnapshot{
			driver:              d,
			errored:             pd.phase1Errored[id],
			textHash:            pd.textHashes[id],
			externalProjections: pd.externalProjections,
		}
		if !snap.errored {
			snap.exported = pd.publicSymbols[packageIdentifierFor(module)]
		}
		if pd.resetScope[id] {
			snap.fingerprint = pd.fingerprints[id]
		} else if pd.priorState != nil {
			if prev, ok := pd.priorState.modules[id]; ok {
				snap.fingerprint = prev.fingerprint
			}
		}
		out[id] = snap
	}
	return out
}
