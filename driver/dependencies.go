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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package driver

import (
	stdcontext "context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// DependencyRequest names one package a DependencyResolver is asked to supply.
// Descriptor.Version may be empty when the importing source pinned no version, and
// ModuleName and Repository are hints: the module the response must contain and the
// repository the manifest asked it be looked up in.
type DependencyRequest struct {
	Descriptor PackageDescriptor
	ModuleName string
	Repository string
	Resolution ResolutionOptions
}

// DependencyResolver supplies the sources of a single dependency package.
// Returning every result as its zero value (no fsys, root, sources or diagnostics, and
// a nil error) means "not found", which callers treat as unresolved rather than fatal;
// any other partially populated response is an error.
type DependencyResolver interface {
	Resolve(ctx stdcontext.Context, request DependencyRequest) (
		fsys fs.FS,
		packageRoot string,
		sources *PackageSources,
		discoveryDiagnostics []diagnostics.Diagnostic,
		err error,
	)
}

var implicitPackages = []string{
	"lang.__internal", "lang.int", "lang.boolean", "lang.decimal", "lang.error", "lang.float",
	"lang.string", "lang.value", "lang.xml", "lang.array", "lang.map", "lang.object", "lang.runtime",
}

type selectedPackage struct {
	descriptor PackageDescriptor
	parsed     *ParsedPackage
	manifest   Manifest
	implicit   bool
}

type moduleGraph struct {
	nodes []ModuleDescriptor
	edges map[ModuleDescriptor][]ModuleDescriptor
}

var errDependencyDiagnostic = errors.New("driver: dependency diagnostic stopped resolution")

// ResolveDependencies resolves the whole dependency closure of parsed and returns it as a
// single ParsedPackage whose modules are in compilation order. Resolution halted by an error
// diagnostic is not an error for the caller: the original parsed package is returned unchanged
// and the diagnostics are left on ctx.
func ResolveDependencies(ctx *Context, parsed *ParsedPackage, resolver DependencyResolver) (*ParsedPackage, error) {
	result, err := resolveDependencies(ctx, parsed, resolver)
	if errors.Is(err, errDependencyDiagnostic) {
		return parsed, nil
	}
	return result, err
}

// resolveDependencies does the work of ResolveDependencies without swallowing
// errDependencyDiagnostic. It selects the root package, then every implicit lang package, then
// walks imports and manifest dependencies breadth-first, recording each dependency as a module
// graph edge, and returns the selection flattened into topological order. A dependency cycle is
// reported as a package diagnostic rather than an error.
func resolveDependencies(ctx *Context, parsed *ParsedPackage, resolver DependencyResolver) (*ParsedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if parsed == nil || resolver == nil {
		return nil, errors.New("driver: nil parsed package or dependency resolver")
	}
	rootData, ok := parsed.packages[parsed.Root]
	if !ok {
		return nil, errors.New("driver: parsed package is missing root manifest metadata")
	}
	selected := make(map[string]*selectedPackage)
	root := &selectedPackage{descriptor: parsed.Root, parsed: parsed, manifest: cloneManifest(rootData.manifest)}
	for _, dependency := range root.manifest.Dependencies {
		if implicitPackage(dependency.Org, dependency.Name) && dependency.Version != "0.0.1" {
			return nil, fmt.Errorf("driver: implicit package %s/%s requires version 0.0.1, got %s",
				dependency.Org, dependency.Name, dependency.Version)
		}
	}
	selected[packageKey(parsed.Root)] = root
	selection := []*selectedPackage{root}
	graph := moduleGraph{edges: make(map[ModuleDescriptor][]ModuleDescriptor)}
	for _, module := range parsed.Modules {
		graph.addNode(module.ID)
	}

	implicitSelected := make([]*selectedPackage, 0, len(implicitPackages))
	for _, name := range implicitPackages {
		descriptor := PackageDescriptor{Org: "ballerina", Name: name, Version: "0.0.1"}
		if packageKey(descriptor) == packageKey(parsed.Root) {
			if module := findModule(parsed, name); module != nil {
				module.implicitName = name
			}
			continue
		}
		dependency, unresolved, err := resolvePackage(ctx, resolver, DependencyRequest{
			Descriptor: descriptor, Resolution: root.manifest.Resolution,
		}, true)
		if err != nil {
			return nil, err
		}
		if unresolved {
			return nil, fmt.Errorf("driver: required implicit package %s/%s:%s was not resolved", descriptor.Org, descriptor.Name, descriptor.Version)
		}
		dependency.implicit = true
		selected[packageKey(descriptor)] = dependency
		selection = append(selection, dependency)
		implicitSelected = append(implicitSelected, dependency)
		for _, module := range dependency.parsed.Modules {
			if module.ID.Name == name {
				module.implicitName = name
			}
			graph.addNode(module.ID)
		}
	}
	for _, from := range parsed.Modules {
		for _, dependency := range implicitSelected {
			for _, to := range dependency.parsed.Modules {
				graph.addEdge(from.ID, to.ID)
			}
		}
	}

	queue := []*selectedPackage{root}
	queue = append(queue, implicitSelected...)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := queue[0]
		queue = queue[1:]
		for _, module := range current.parsed.Modules {
			for _, document := range orderedParsedDocuments(module) {
				for _, imported := range importsFromDocument(document.document) {
					resolvedModule, newlySelected, err := resolveImport(ctx, resolver, root.manifest.Resolution,
						root.manifest, current.manifest, current, module, document, imported, selected)
					if err != nil {
						return nil, err
					}
					if resolvedModule == nil {
						continue
					}
					graph.addEdge(module.ID, resolvedModule.ID)
					if newlySelected != nil {
						selection = append(selection, newlySelected)
						queue = append(queue, newlySelected)
						for _, selectedModule := range newlySelected.parsed.Modules {
							graph.addNode(selectedModule.ID)
						}
					}
				}
			}
		}
		if current != root {
			for _, dependency := range current.manifest.Dependencies {
				target, newlySelected, err := resolveManifestDependency(ctx, resolver, root.manifest.Resolution,
					root.manifest, current, dependency, selected)
				if err != nil {
					return nil, err
				}
				if target == nil {
					continue
				}
				for _, from := range current.parsed.Modules {
					for _, to := range target.parsed.Modules {
						graph.addEdge(from.ID, to.ID)
					}
				}
				if newlySelected != nil {
					selection = append(selection, newlySelected)
					queue = append(queue, newlySelected)
					for _, module := range newlySelected.parsed.Modules {
						graph.addNode(module.ID)
					}
				}
			}
		}
	}
	for _, pkg := range selection {
		defaultModule := findModule(pkg.parsed, pkg.descriptor.Name)
		if defaultModule == nil {
			continue
		}
		for _, module := range pkg.parsed.Modules {
			if module != defaultModule {
				graph.addEdge(defaultModule.ID, module.ID)
			}
		}
	}

	ordered, cycle := graph.topological()
	if cycle {
		ctx.addPackageDiagnostics(parsed.Root, []diagnostics.Diagnostic{packageDiagnostic("cyclic package or module dependency")})
	}
	byDescriptor := make(map[ModuleDescriptor]*ParsedModule)
	packages := make(map[PackageDescriptor]packageData)
	for _, pkg := range selection {
		packages[pkg.descriptor] = packageData{manifest: cloneManifest(pkg.manifest)}
		for _, module := range pkg.parsed.Modules {
			byDescriptor[module.ID] = module
		}
	}
	modules := make([]*ParsedModule, 0, len(ordered))
	for _, descriptor := range ordered {
		if module := byDescriptor[descriptor]; module != nil {
			modules = append(modules, module)
		}
	}
	ids := make([]ModuleDescriptor, len(modules))
	for i, module := range modules {
		ids[i] = module.ID
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx.setTopologicalModules(ids)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &ParsedPackage{Root: parsed.Root, Modules: modules, packages: packages}, nil
}

type parsedDocumentRef struct {
	document *ParsedDocument
	test     bool
	index    int
}

// orderedParsedDocuments lists a module's documents in diagnostic order: the production
// documents in declaration order, then the test documents, each tagged with its index within
// its own source set.
func orderedParsedDocuments(module *ParsedModule) []parsedDocumentRef {
	result := make([]parsedDocumentRef, 0, len(module.Documents)+len(module.TestDocuments))
	for i, document := range module.Documents {
		result = append(result, parsedDocumentRef{document: document, index: i})
	}
	for i, document := range module.TestDocuments {
		result = append(result, parsedDocumentRef{document: document, test: true, index: i})
	}
	return result
}

type sourceImport struct {
	org, module string
	start, end  int
}

// importsFromDocument extracts the import declarations of a document with quoted-identifier
// prefixes stripped and dotted module names rejoined, keeping each declaration's text range so
// an unresolved import can be reported at its source position. It returns nil when the syntax
// tree root is not a module part.
func importsFromDocument(document *ParsedDocument) []sourceImport {
	root, ok := document.SyntaxTree.RootNode.(*st.ModulePart)
	if !ok {
		return nil
	}
	result := make([]sourceImport, 0)
	imports := root.Imports()
	for declaration := range imports.Iterator() {
		org := ""
		if orgNode := declaration.OrgName(); orgNode != nil {
			org = strings.TrimPrefix(orgNode.OrgName().Text(), "'")
		}
		moduleName := declaration.ModuleName()
		parts := make([]string, 0, moduleName.Size())
		for identifier := range moduleName.Iterator() {
			parts = append(parts, strings.TrimPrefix(identifier.Text(), "'"))
		}
		rangeValue := declaration.TextRange()
		result = append(result, sourceImport{org: org, module: strings.Join(parts, "."), start: rangeValue.StartOffset, end: rangeValue.EndOffset})
	}
	return result
}

// resolveImport resolves one import declaration to the module that satisfies it, selecting a new
// package if needed. An import with no org, or with the importing package's own org, is first
// looked for inside the current package; otherwise the full module name and its first dotted
// segment are tried as package names in turn, versioned by the current manifest and repository-
// overridden by the root manifest. It returns a nil module with no error after recording an
// unresolved-import diagnostic when no candidate resolves; the second result is non-nil only for
// a package newly added to selected.
func resolveImport(ctx *Context, resolver DependencyResolver, options ResolutionOptions, rootManifest, currentManifest Manifest, current *selectedPackage,
	module *ParsedModule, document parsedDocumentRef, imported sourceImport, selected map[string]*selectedPackage,
) (*ParsedModule, *selectedPackage, error) {
	org := imported.org
	if org == "" {
		org = current.descriptor.Org
	}
	if org == current.descriptor.Org {
		if target := findModule(current.parsed, imported.module); target != nil {
			return target, nil, nil
		}
	}
	candidates := []string{imported.module}
	if index := strings.IndexByte(imported.module, '.'); index > 0 {
		candidates = append(candidates, imported.module[:index])
	}
	for _, candidate := range candidates {
		descriptor := PackageDescriptor{Org: org, Name: candidate}
		repository := ""
		if dependency, ok := manifestDependency(currentManifest, org, candidate); ok {
			descriptor.Version = dependency.Version
			repository = dependency.Repository
		}
		if dependency, ok := manifestDependency(rootManifest, org, candidate); ok && dependency.Repository != "" {
			repository = dependency.Repository
		}
		if implicitPackage(org, candidate) && descriptor.Version != "" && descriptor.Version != "0.0.1" {
			return nil, nil, fmt.Errorf("driver: implicit package %s/%s requires version 0.0.1, got %s", org, candidate, descriptor.Version)
		}
		if existing := selected[packageKey(descriptor)]; existing != nil {
			if existing.implicit && descriptor.Version != "" && descriptor.Version != "0.0.1" {
				return nil, nil, fmt.Errorf("driver: conflicting implicit package version %s", descriptor.Version)
			}
			if target := findModule(existing.parsed, imported.module); target != nil {
				return target, nil, nil
			}
			return nil, nil, fmt.Errorf("driver: selected package %s/%s does not contain required module %s", org, candidate, imported.module)
		}
		resolved, unresolved, err := resolvePackage(ctx, resolver, DependencyRequest{Descriptor: descriptor,
			ModuleName: imported.module, Repository: repository, Resolution: options}, false)
		if err != nil {
			return nil, nil, err
		}
		if unresolved {
			continue
		}
		if implicitPackage(org, candidate) && resolved.descriptor.Version != "0.0.1" {
			return nil, nil, fmt.Errorf("driver: implicit package %s/%s resolved to version %s", org, candidate, resolved.descriptor.Version)
		}
		selected[packageKey(resolved.descriptor)] = resolved
		return findModule(resolved.parsed, imported.module), resolved, nil
	}
	addUnresolvedImportDiagnostic(ctx, module, document, imported, org)
	return nil, nil, nil
}

// resolveManifestDependency selects the package for one manifest dependency entry, reusing the
// already selected package for that org and name if there is one. The repository from the root
// manifest wins over the entry's own. An unresolvable dependency is reported as a package
// diagnostic on current and returns a nil target with no error; the second result is non-nil only
// when the package was newly selected.
func resolveManifestDependency(ctx *Context, resolver DependencyResolver, options ResolutionOptions,
	rootManifest Manifest, current *selectedPackage, dependency Dependency, selected map[string]*selectedPackage,
) (*selectedPackage, *selectedPackage, error) {
	descriptor := PackageDescriptor{Org: dependency.Org, Name: dependency.Name, Version: dependency.Version}
	if implicitPackage(dependency.Org, dependency.Name) && dependency.Version != "0.0.1" {
		return nil, nil, fmt.Errorf("driver: implicit package %s/%s requires version 0.0.1, got %s", dependency.Org, dependency.Name, dependency.Version)
	}
	if existing := selected[packageKey(descriptor)]; existing != nil {
		return existing, nil, nil
	}
	repository := dependency.Repository
	if rootDependency, ok := manifestDependency(rootManifest, dependency.Org, dependency.Name); ok && rootDependency.Repository != "" {
		repository = rootDependency.Repository
	}
	resolved, unresolved, err := resolvePackage(ctx, resolver, DependencyRequest{Descriptor: descriptor,
		Repository: repository, Resolution: options}, false)
	if err != nil {
		return nil, nil, err
	}
	if unresolved {
		ctx.addPackageDiagnostics(current.descriptor, []diagnostics.Diagnostic{packageDiagnostic("cannot resolve dependency " + dependency.Org + "/" + dependency.Name + ":" + dependency.Version)})
		return nil, nil, nil
	}
	if implicitPackage(dependency.Org, dependency.Name) && resolved.descriptor.Version != "0.0.1" {
		return nil, nil, fmt.Errorf("driver: conflicting implicit package version %s", resolved.descriptor.Version)
	}
	selected[packageKey(resolved.descriptor)] = resolved
	return resolved, resolved, nil
}

// addUnresolvedImportDiagnostic records an error diagnostic against the import declaration's own
// text range in the document that contains it, attributed to the import resolution stage.
func addUnresolvedImportDiagnostic(ctx *Context, module *ParsedModule, document parsedDocumentRef, imported sourceImport, org string) {
	identity := diagnosticIdentity(module.dependency, module.ID, document.document.SourcePath)
	location := diagnostics.NewLocationForIdentity(ctx.DiagnosticEnv(), identity, imported.start, imported.end)
	ctx.addModuleDiagnostics(module.ID, "Import Resolution", document.test, document.index,
		[]diagnostics.Diagnostic{sourceDiagnostic("Unknown import: "+org+"/"+imported.module, location)})
}

// resolvePackage asks the resolver for one package, validates the response against the request,
// and parses it. The boolean result is true when the resolver reported the package as not found;
// a response whose descriptor, version or module set contradicts the request, or that is only
// partly populated, is an error. Discovery diagnostics are recorded on ctx and an error among
// them stops resolution with errDependencyDiagnostic.
func resolvePackage(ctx *Context, resolver DependencyResolver, request DependencyRequest, implicit bool) (*selectedPackage, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	fsys, root, sources, discoveryDiagnostics, err := resolver.Resolve(ctx.ctx, request)
	if err != nil {
		return nil, false, err
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	empty := fsys == nil && root == "" && sources == nil && len(discoveryDiagnostics) == 0
	if empty {
		return nil, true, nil
	}
	if fsys == nil || root == "" || sources == nil {
		return nil, false, errors.New("driver: partially populated dependency resolver response")
	}
	if err := validateParseInputs(root, sources); err != nil {
		return nil, false, fmt.Errorf("driver: invalid dependency resolver response: %w", err)
	}
	if sources.Descriptor.Org != request.Descriptor.Org || sources.Descriptor.Name != request.Descriptor.Name {
		return nil, false, fmt.Errorf("driver: resolver descriptor mismatch: requested %s/%s, got %s/%s", request.Descriptor.Org, request.Descriptor.Name, sources.Descriptor.Org, sources.Descriptor.Name)
	}
	if request.Descriptor.Version != "" && sources.Descriptor.Version != request.Descriptor.Version {
		return nil, false, fmt.Errorf("driver: resolver version mismatch: requested %s, got %s", request.Descriptor.Version, sources.Descriptor.Version)
	}
	if request.ModuleName != "" && findModuleSources(sources, request.ModuleName) == nil {
		return nil, false, fmt.Errorf("driver: resolver response does not contain required module %s", request.ModuleName)
	}
	ctx.addPackageDiagnostics(sources.Descriptor, discoveryDiagnostics)
	if hasErrors(discoveryDiagnostics) {
		return nil, false, errDependencyDiagnostic
	}
	parsed, err := parsePackage(ctx, fsys, root, sources, parser.DebugOptions{}, true)
	if err != nil {
		return nil, false, err
	}
	return &selectedPackage{descriptor: sources.Descriptor, parsed: parsed, manifest: cloneManifest(sources.Manifest), implicit: implicit}, false, nil
}

// manifestDependency returns the manifest's dependency entry for an org and name, and whether
// one was declared.
func manifestDependency(manifest Manifest, org, name string) (Dependency, bool) {
	for _, dependency := range manifest.Dependencies {
		if dependency.Org == org && dependency.Name == name {
			return dependency, true
		}
	}
	return Dependency{}, false
}

// findModule returns the module of parsed whose unqualified name is name, or nil if the package
// has no such module.
func findModule(parsed *ParsedPackage, name string) *ParsedModule {
	for _, module := range parsed.Modules {
		if module.ID.Name == name {
			return module
		}
	}
	return nil
}

// findModuleSources returns the unparsed sources of the module named name, or nil if the
// resolver response has no such module.
func findModuleSources(sources *PackageSources, name string) *ModuleSources {
	for _, module := range sources.Modules {
		if module.ID.Name == name {
			return module
		}
	}
	return nil
}

// packageKey identifies a package by org and name only, deliberately omitting the version, so
// that at most one version of any org/name pair can ever be selected.
func packageKey(descriptor PackageDescriptor) string {
	return descriptor.Org + "\x00" + descriptor.Name
}

// implicitPackage reports whether org and name denote one of the ballerina lang packages that
// every module implicitly depends on, and which are therefore pinned to version 0.0.1.
func implicitPackage(org, name string) bool {
	if org != "ballerina" {
		return false
	}
	for _, value := range implicitPackages {
		if value == name {
			return true
		}
	}
	return false
}

// sourceDiagnostic builds an UNRESOLVED_IMPORT error diagnostic anchored at a source location.
func sourceDiagnostic(message string, location diagnostics.Location) diagnostics.Diagnostic {
	code := "UNRESOLVED_IMPORT"
	return diagnostics.NewDefaultDiagnostic(diagnostics.NewDiagnosticInfo(&code, message, diagnostics.Error), location, nil)
}

// packageDiagnostic builds a DEPENDENCY_RESOLUTION error diagnostic with no source location, for
// failures that belong to a package as a whole rather than to any one document.
func packageDiagnostic(message string) diagnostics.Diagnostic {
	code := "DEPENDENCY_RESOLUTION"
	return diagnostics.NewDefaultDiagnostic(diagnostics.NewDiagnosticInfo(&code, message, diagnostics.Error), diagnostics.Location{}, nil)
}

// addNode appends module to the node list unless it is already present, preserving insertion
// order so that the topological sort is deterministic.
func (g *moduleGraph) addNode(module ModuleDescriptor) {
	for _, existing := range g.nodes {
		if existing == module {
			return
		}
	}
	g.nodes = append(g.nodes, module)
}

// addEdge records that from depends on to, adding both endpoints as nodes and ignoring a
// duplicate edge. Self edges are stored and later show up as a cycle.
func (g *moduleGraph) addEdge(from, to ModuleDescriptor) {
	g.addNode(from)
	g.addNode(to)
	for _, existing := range g.edges[from] {
		if existing == to {
			return
		}
	}
	g.edges[from] = append(g.edges[from], to)
}

// topological returns the nodes in depth-first post-order, so every module appears after the
// modules it depends on, and reports whether a cycle was found. On a cycle the order is
// best-effort rather than valid: duplicates are dropped and any node the traversal never emitted
// is appended at the end, so the result always contains each node exactly once.
func (g *moduleGraph) topological() ([]ModuleDescriptor, bool) {
	state := make(map[ModuleDescriptor]uint8)
	result := make([]ModuleDescriptor, 0, len(g.nodes))
	cycle := false
	var visit func(ModuleDescriptor)
	visit = func(module ModuleDescriptor) {
		if state[module] == 2 {
			return
		}
		if state[module] == 1 {
			cycle = true
			return
		}
		state[module] = 1
		for _, dependency := range g.edges[module] {
			visit(dependency)
		}
		state[module] = 2
		result = append(result, module)
	}
	for _, module := range g.nodes {
		visit(module)
	}
	if cycle {
		seen := make(map[ModuleDescriptor]bool)
		unique := result[:0]
		for _, module := range result {
			if !seen[module] {
				seen[module] = true
				unique = append(unique, module)
			}
		}
		for _, module := range g.nodes {
			if !seen[module] {
				unique = append(unique, module)
			}
		}
		result = unique
	}
	return result, cycle
}
