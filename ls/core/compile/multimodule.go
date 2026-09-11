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

// multimodule.go implements packageDriver, which orchestrates moduleDriver
// across every module of a package for one generation, replicating the
// Phase 1 -> Phase 2 barrier projects/package_compilation.go's
// compileModulesInternal implements (its two loops, ~line 113 and ~line 144)
// using only the compiler's public per-stage API and pkg's public accessors —
// this package cannot see moduleContext/PackageCompilation internals. See
// docs/adr/2026-08-28-ls-owned-staged-compilation-pipeline.md's "Phase 1 ->
// Phase 2 barrier" bullet.
//
// Scope: this orchestrates the package's own (same-package, editable)
// modules only. External dependency packages (stdlib, Central, BALA) are not
// driven through packageDriver at all — matching the ADR's "full bypass
// applies only to the LS's own editable source-root modules" — so a module
// that imports an external package (e.g. ballerina/io) will not resolve that
// import through this driver today; this is an existing, not newly
// introduced, gap already present in phase 1's single-module driver (its
// moduleResolutionInput.publicSymbols was always empty), now simply carried
// forward at package granularity. See the phase 2 handoff report for the
// full reasoning.
//
// Phase 2 runs sequentially, not in parallel like
// projects/package_compilation.go's goroutine-per-module loop. A module
// skipped in Phase 1 (dependencyErrored) never reaches stageParsed, and every
// ensureX stage function cascades back down to ensureParsed for a module
// that hasn't reached it yet. Phase 2 excludes Phase-1-skipped modules by
// construction (see phase2Eligible), so this cascade is not reachable today —
// but running Phase 2 sequentially removes the question entirely rather than
// depending on that exclusion staying correct, and keeps this driver free of
// the goroutine/panic-recovery plumbing package_compilation.go needs for its
// parallel Phase 2. There is no performance requirement in this ticket that
// forces parallelism.
//
// Concurrency: two independent packageDriver runs can legitimately execute
// concurrently over the same shared CompilerEnvironment — CompilationService's
// background runCycle and Compile()'s inline extractForURI fallback are
// exactly such a pair, since Compile() has no coordination with the
// background cycle (it only reads the stable-snapshot cache first; on a miss
// it drives inline), so the two can race on the same package the first time a
// root is compiled. This is safe without any lock owned by this package:
// CompilerEnvironment's shared mutable state is guarded field-by-field at its
// own boundary (symbolSpaces via symbolSpacesMu, DiagnosticEnv and
// functionSignatures.Store via their own internal RWMutexes, symbolAnnotations/
// underlyingSymbol as sync.Map, packageInterner via its own rwLock, typeEnv via
// its own internal mutexes) — see context/env.go. The one field that was
// missing this, anonTypeCount/anonFuncCount, is now guarded by
// CompilerEnvironment's own anonCountMu (context/env.go), matching the
// symbolSpacesMu pattern. A whole-driver-run lock here would be the wrong
// layer: CompilerEnvironment is designed to be read/written concurrently by
// its own accessors, so serializing entire packageDriver runs against it
// would only hide bugs in that per-field locking rather than fix anything.
package compile

import (
	"sync"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// registeredNames tracks, per shared DiagnosticEnv, which registration keys
// moduleDriver.ensureParsed has already registered. diagnostics.RegisterFile
// panics on a same-name/different-content re-registration (a genuine name
// collision, by design — see tools/diagnostics/diagnostic_env.go); ensureParsed
// re-runs over the same module on every edit generation with the module's
// current (possibly-changed) content, so it must not call RegisterFile a
// second time for a name it has already registered. This is safe because
// nothing in this package's diagnostic-extraction path (extractByFile,
// extractForURI) reads DiagnosticEnv's stored content back — they resolve
// positions from their own freshly-fetched document text against the
// Location's raw byte offsets, keyed only by env.FileName(loc)'s registration
// key. RegisterFile's role for this driver is therefore only to mint a
// stable fileIndex once per name, via NewLocation's env.FileIndex(fileName);
// re-registering under the same name would never be observed by any reader.
// Entries are never evicted (one per distinct DiagnosticEnv for the process
// lifetime — bounded by the number of source roots ever opened in this
// session), the same accepted growth tradeoff as CompilerEnvironment's other
// per-environment state.
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

// migratedLangLibs mirrors projects/module_context.go's bundledLangLibs name
// list (unexported there) so seedMigratedLangLibs can be reimplemented here
// without reaching into projects internals.
var migratedLangLibs = []string{
	"lang.__internal", "lang.int", "lang.boolean", "lang.decimal", "lang.error",
	"lang.string", "lang.value", "lang.xml", "lang.float", "lang.array",
	"lang.map", "lang.object",
}

// seedMigratedLangLibs adds already-published migrated lang libraries from
// publicSymbols to implicitImports, mirroring
// projects/module_context.go:456-466 (unexported there, reimplemented here
// against packageDriver's own local publicSymbols map instead of
// projects.Environment's private one). A no-op until a lib has been
// compiled — this package's local map only ever contains modules of the
// package packageDriver is driving.
func seedMigratedLangLibs(implicitImports map[string]model.ExportedSymbolSpace, publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace) {
	for _, name := range migratedLangLibs {
		if space, ok := publicSymbols[semantics.PackageIdentifier{OrgName: "ballerina", ModuleName: name}]; ok {
			implicitImports[name] = space
		}
	}
}

// packageIdentifierFor builds the key module_context.go:296-299 publishes a
// module's exported symbol space under, from public ModuleDescriptor
// accessors only.
func packageIdentifierFor(module *projects.Module) semantics.PackageIdentifier {
	desc := module.Descriptor()
	return semantics.PackageIdentifier{OrgName: desc.Org().Value(), ModuleName: desc.Name().String()}
}

// packageDriver drives a package's own modules through the LS stage ladder
// for one generation. One packageDriver corresponds to one generation-advance
// over the whole package (or a single targeted module plus its same-package
// dependencies); it owns one moduleDriver per module reached and one
// publicSymbols map local to this run (never projects.Environment's private
// field — see the ADR).
//
// priorState (design item 1) is the previous committed generation's
// carry-forward state, or nil for a run that never participates in carry-
// forward (extractForURI's single-module path — see gen_state.go's doc
// comment). When non-nil, a module outside the computed resetScope is adopted
// from priorState instead of rebuilt (adoptCarriedForward) — its moduleDriver
// (and thus its diagnostics, still reachable via allDiagnostics) carries
// forward by reference, untouched. stale (design item 5) is checked between
// modules in runPhase1/advanceAll/advanceModule's loops so a superseded
// generation aborts early instead of finishing wasted work; nil means never
// stale (extractForURI has no generation to compare against).
type packageDriver struct {
	pkg *projects.Package
	env *context.CompilerEnvironment

	drivers       map[projects.ModuleID]*moduleDriver
	publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace
	phase1Errored map[projects.ModuleID]bool

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

// newPackageDriver constructs a packageDriver for one generation-advance over
// pkg, sharing pkg's project's CompilerEnvironment (reused across
// generations, per the ADR's re-entrancy rule) but owning fresh per-module
// CompilerContexts and its own local publicSymbols map. openText, priorState
// and stale are ticket 28's additions (see the type doc comment); any/all may
// be nil.
func newPackageDriver(pkg *projects.Package, openText textProvider, priorState *packageGenState, stale func() bool) *packageDriver {
	return &packageDriver{
		pkg:           pkg,
		env:           pkg.Project().Environment().CompilerEnvironment(),
		drivers:       make(map[projects.ModuleID]*moduleDriver),
		publicSymbols: make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace),
		phase1Errored: make(map[projects.ModuleID]bool),
		openText:      openText,
		priorState:    priorState,
		stale:         stale,
	}
}

// aborted reports whether a newer generation has already superseded this run
// (design item 5). Callers check this between modules/stages and abort
// cleanly (no cleanup needed — a packageDriver's state is local to this run
// and is never published unless the caller commits it).
func (pd *packageDriver) aborted() bool {
	return pd.stale != nil && pd.stale()
}

// topoModules returns pkg's own modules in dependency order, using
// pkg.Resolution().ModuleDependencyGraph() (public, independent of
// pkg.Compilation()) -> ToTopologicallySortedList(). A descriptor the graph
// carries for an external dependency (pkg.ModuleByName returns nil for it)
// is skipped — external packages are out of this driver's scope.
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

// resolutionInputFor builds this module's moduleResolutionInput against
// packageDriver's shared, growing publicSymbols map — the same map every
// module in this generation reads from and, on success, publishes into.
func (pd *packageDriver) resolutionInputFor(module *projects.Module) moduleResolutionInput {
	implicit := make(map[string]model.ExportedSymbolSpace)
	seedMigratedLangLibs(implicit, pd.publicSymbols)
	return newModuleResolutionInput(module.Descriptor().Org().Value(), implicit, pd.publicSymbols)
}

// dependencyErrored reports whether any of module's direct same-package
// dependencies already failed Phase 1, mirroring
// projects/package_compilation.go:100-111's dependencyErrored closure
// (unexported there) using the public DependencyGraph.DirectDependencies
// instead of the private descToModule/erroredModules pair it closes over.
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

// ensureResetScope computes design item 1's reset scope, memoized for this
// packageDriver instance. Every module gets a textHash regardless of whether
// it turns out dirty (needed to build the next generation's carry-forward
// state either way). With no priorState (extractForURI's single-module
// path — see the type doc comment), the whole scope is "reset everything",
// matching pre-ticket-28 behavior exactly (every module always rebuilt fresh).
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

// adoptCarriedForward carries module's prior-generation moduleDriver forward
// by reference instead of rebuilding it, populating this generation's
// drivers/publicSymbols/phase1Errored exactly as a fresh Phase 1 run would
// have, so allDiagnostics and dependency resolution both see it as if it had
// just run. Returns false (nothing adopted) if priorState has no snapshot for
// module — the caller then falls through to a full reset instead of silently
// dropping the module.
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

// recordFingerprintAndPrune computes module's public-API fingerprint (design
// item 4) after its Phase 1 succeeds and records it for this generation's
// carry-forward commit. If the fingerprint is unchanged from the prior
// committed generation's, module's direct dependents are pruned back out of
// the reset scope (unless independently dirty from their own edit) — they
// keep referencing the prior generation's state untouched instead of being
// rebuilt just because they transitively followed a reset module in topo
// order.
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

// runPhase1Module advances module through Phase 1 (parse -> symbol
// resolution -> top-level type resolution), mirroring
// projects/module_context.go's resolveTypesAndSymbols publish-then-continue
// ordering exactly: a module is marked errored and skips further Phase 1
// work as soon as HasErrors() is true right after symbol resolution
// (module_context.go:290-294) — before publishing, so a half-built symbol
// space is never published — and publicSymbols is published immediately
// after that check succeeds (module_context.go:296-299), before top-level
// type resolution runs, so a later top-level-type error does not retroactively
// un-publish an already-valid symbol space. If module's own direct
// dependency already errored in this generation, module is skipped entirely
// (never reaches driverFor/ensureParsed), matching
// package_compilation.go:124-127.
//
// A module outside the reset scope is adopted from the prior generation
// (adoptCarriedForward) instead of run through any of this — design item 1.
// On success, its fingerprint is recorded and may prune its own dependents
// out of the reset scope — design item 4.
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
	if d.diagnosticContext().HasErrors() {
		pd.phase1Errored[id] = true
		return
	}
	pd.publicSymbols[packageIdentifierFor(module)] = d.exported

	d.advanceTo(stageTopLevelTypeResolved, module, input)
	if d.diagnosticContext().HasErrors() {
		pd.phase1Errored[id] = true
		return
	}
	pd.recordFingerprintAndPrune(module, d)
}

// runPhase1 advances every module in modules through Phase 1, in topological
// order (sequential — symbol/top-level-type resolution of a module needs its
// dependencies' published symbol spaces). Checks aborted() between modules
// (design item 5).
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

// phase2Eligible reports whether module completed Phase 1 successfully (not
// errored, and reached at least stageTopLevelTypeResolved) and so may
// proceed into Phase 2. A module never reached by Phase 1 (e.g. because an
// earlier caller only drove a prefix of the topological order — see
// advanceModule) is not eligible either.
func (pd *packageDriver) phase2Eligible(module *projects.Module) (*moduleDriver, bool) {
	id := module.ModuleID()
	if pd.phase1Errored[id] {
		return nil, false
	}
	d, ok := pd.drivers[id]
	if !ok || d.currentStage() < stageTopLevelTypeResolved {
		return nil, false
	}
	return d, true
}

// advanceAll drives every module of pkg to target: Phase 1 across the whole
// package, then — only if no module failed Phase 1
// (package_compilation.go:139-143's "stop the compilation pipeline here" gate)
// — Phase 2 for every Phase-1-eligible module, sequentially. Checks
// aborted() between modules (design item 5): a superseded generation stops
// early rather than finishing wasted work. The caller must check aborted()
// itself before relying on or committing this run's results.
func (pd *packageDriver) advanceAll(target moduleStage) {
	modules := pd.topoModules()
	pd.runPhase1(modules)
	if pd.aborted() {
		return
	}
	if len(pd.phase1Errored) > 0 {
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

// advanceModule drives targetModule's same-package dependencies through
// Phase 1 only as far as targetModule itself (topological order, early
// return once targetModule is reached — modules ordered after it are never
// compiled, mirroring ls-ref's prepareSymbolResolution pattern the ADR cites),
// then drives targetModule alone through Phase 2 up to target. It is the
// single-module path Compile's inline fallback (extractForURI) uses instead
// of driving the whole package, so a single-file diagnostic read does not pay
// for every other module's Phase 2. This path never participates in
// carry-forward (see gen_state.go): the caller always constructs pd with a
// nil priorState, so every module here is always rebuilt fresh, matching
// pre-ticket-28 behavior exactly.
func (pd *packageDriver) advanceModule(target moduleStage, targetModule *projects.Module) {
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

// allDiagnostics collects every reached module's diagnostics in topological
// order. Each module's diagnostics live on that module's own CompilerContext:
// a fresh one for a module reset this generation, or the prior generation's
// carried-forward one for a module adoptCarriedForward adopted untouched
// (design item 1) — either way pd.drivers holds exactly one moduleDriver per
// module reached, so this never double-counts within a generation, and an
// unedited module's diagnostics keep surfacing instead of silently vanishing
// once it's no longer reset every generation.
func (pd *packageDriver) allDiagnostics() []diagnostics.Diagnostic {
	var all []diagnostics.Diagnostic
	for _, module := range pd.topoModules() {
		d, ok := pd.drivers[module.ModuleID()]
		if !ok {
			continue
		}
		all = append(all, d.diagnosticContext().Diagnostics()...)
	}
	return all
}

// snapshotForCommit builds this generation's carry-forward state (design
// item 1) for every module this run actually reached (a module skipped
// entirely via dependencyErrored cascade has no driver and is omitted, same
// as allDiagnostics/moduleDiagCount). A carried-forward module's fingerprint
// is copied from the prior generation unchanged (it was never recomputed this
// generation — recordFingerprintAndPrune only runs for a module actually
// reset). The caller (realCompilePackage) must not call this if aborted()
// is true — a superseded run's partial state must never be committed.
func (pd *packageDriver) snapshotForCommit() map[projects.ModuleID]moduleGenSnapshot {
	out := make(map[projects.ModuleID]moduleGenSnapshot, len(pd.drivers))
	for _, module := range pd.topoModules() {
		id := module.ModuleID()
		d, ok := pd.drivers[id]
		if !ok {
			continue
		}
		snap := moduleGenSnapshot{
			driver:   d,
			errored:  pd.phase1Errored[id],
			textHash: pd.textHashes[id],
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
