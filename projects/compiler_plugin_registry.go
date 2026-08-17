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

package projects

import (
	"fmt"
	"slices"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/compilerplugin"
	"github.com/ballerina-nutcracker/ballerina/compilerpluginregistry"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

type compilerPluginDeclaration struct {
	after    compilerplugin.Stage
	function string
}

type compilerPluginKey struct {
	org      string
	pkg      string
	function string
	after    compilerplugin.Stage
}

type compilerPluginRegistry struct {
	plugins map[compilerPluginKey]compilerplugin.CompilerPlugin
}

var (
	linkedCompilerPluginRegistryOnce sync.Once
	linkedCompilerPluginRegistry     *compilerPluginRegistry
)

func getLinkedCompilerPluginRegistry() *compilerPluginRegistry {
	linkedCompilerPluginRegistryOnce.Do(func() {
		linkedCompilerPluginRegistry = newCompilerPluginRegistry()
	})
	return linkedCompilerPluginRegistry
}

func newCompilerPluginRegistry() *compilerPluginRegistry {
	registry := &compilerPluginRegistry{plugins: make(map[compilerPluginKey]compilerplugin.CompilerPlugin)}
	compilerpluginregistry.RegisterPlugins(registry.register)
	return registry
}

func (r *compilerPluginRegistry) register(org, pkg, function string, plugin compilerplugin.CompilerPlugin) {
	key := compilerPluginKey{org: org, pkg: pkg, function: function, after: plugin.After}
	if _, exists := r.plugins[key]; exists {
		panic(fmt.Sprintf("duplicate statically linked compiler plugin %s/%s:%s", org, pkg, function))
	}
	if plugin.PackageTransformer == nil {
		panic(fmt.Sprintf("compiler plugin %s/%s:%s has a nil package transformer", org, pkg, function))
	}
	r.plugins[key] = plugin
}

func (r *compilerPluginRegistry) lookup(org, pkg string, declaration compilerPluginDeclaration) (compilerplugin.CompilerPlugin, bool) {
	plugin, ok := r.plugins[compilerPluginKey{
		org: org, pkg: pkg, function: declaration.function, after: declaration.after,
	}]
	return plugin, ok
}

type moduleIdentity struct {
	org        string
	moduleName string
}

type compilerPluginProvider struct {
	org      string
	pkg      string
	exported semantics.PackageIdentifier
	plugins  []linkedCompilerPlugin
}

type linkedCompilerPlugin struct {
	declaration compilerPluginDeclaration
	plugin      compilerplugin.CompilerPlugin
}

type compilerPluginResolver struct {
	moduleOwners map[moduleIdentity]string
	providers    map[string]compilerPluginProvider
}

// newCompilerPluginResolver resolves the compiler plugins of every package
// imported by a module of modules that runs compiler plugins. A provider that
// cannot be resolved is reported once, at its first import in modules order,
// through the module declaring that import.
func newCompilerPluginResolver(modules []*moduleContext) *compilerPluginResolver {
	resolver := &compilerPluginResolver{
		moduleOwners: make(map[moduleIdentity]string),
		providers:    make(map[string]compilerPluginProvider),
	}
	packageModules := make(map[string]*moduleContext)
	for _, module := range modules {
		descriptor := module.getDescriptor()
		key := modulePackageKey(module)
		resolver.moduleOwners[moduleIdentity{
			org: descriptor.Org().Value(), moduleName: descriptor.Name().String(),
		}] = key
		if _, exists := packageModules[key]; !exists {
			packageModules[key] = module
		}
	}
	registry := getLinkedCompilerPluginRegistry()
	visited := make(map[string]struct{})
	for _, module := range modules {
		if !runsCompilerPlugins(module) {
			continue
		}
		ownKey := modulePackageKey(module)
		for _, imported := range module.explicitImports {
			key, ok := resolver.moduleOwners[moduleIdentity{org: imported.org, moduleName: imported.moduleName}]
			if !ok || key == ownKey {
				continue
			}
			if _, seen := visited[key]; seen {
				continue
			}
			visited[key] = struct{}{}
			provider, ok := resolveCompilerPluginProvider(registry, packageModules[key], module, imported)
			if ok {
				resolver.providers[key] = provider
			}
		}
	}
	return resolver
}

// resolveCompilerPluginProvider links the CompilerPlugin.toml declarations of
// the package owning providerModule. It reports failures through importer at
// site and returns false for them, and returns false without reporting when
// the package has no CompilerPlugin.toml.
func resolveCompilerPluginProvider(
	registry *compilerPluginRegistry, providerModule, importer *moduleContext, site moduleImport,
) (compilerPluginProvider, bool) {
	pkgDescriptor := providerModule.getDescriptor().PackageDescriptor()
	key := modulePackageKey(providerModule)
	pkg := providerModule.project.CurrentPackage()
	if pkg == nil || !pkg.packageCtx.getDescriptor().Equals(pkgDescriptor) {
		importer.compilerCtx.InternalError(
			fmt.Sprintf("cannot locate package %s declaring module %s", key, providerModule.getDescriptor().Name().String()),
			site.position,
		)
		return compilerPluginProvider{}, false
	}
	manifestCtx := pkg.packageCtx.getCompilerPluginTomlContext()
	if manifestCtx == nil {
		return compilerPluginProvider{}, false
	}
	declarations, err := parseCompilerPluginManifest(manifestCtx.content)
	if err != nil {
		importer.compilerCtx.SemanticError(fmt.Sprintf("invalid CompilerPlugin.toml for %s: %v", key, err), site.position)
		return compilerPluginProvider{}, false
	}
	provider := compilerPluginProvider{
		org:      pkgDescriptor.Org().Value(),
		pkg:      pkgDescriptor.Name().Value(),
		exported: semantics.PackageIdentifier{OrgName: pkgDescriptor.Org().Value(), ModuleName: pkgDescriptor.Name().Value()},
		plugins:  make([]linkedCompilerPlugin, 0, len(declarations)),
	}
	for _, declaration := range declarations {
		plugin, ok := registry.lookup(provider.org, provider.pkg, declaration)
		if !ok {
			importer.compilerCtx.SemanticError(fmt.Sprintf(
				"compiler plugin implementation not linked for %s:%s at %s", key, declaration.function, declaration.after,
			), site.position)
			return compilerPluginProvider{}, false
		}
		provider.plugins = append(provider.plugins, linkedCompilerPlugin{declaration: declaration, plugin: plugin})
	}
	return provider, true
}

func runsCompilerPlugins(module *moduleContext) bool {
	return module.getCompilationState() == moduleCompilationStateLoadedFromSources &&
		module.cfg != nil && module.bLangPkg != nil
}

func modulePackageKey(module *moduleContext) string {
	pkgDescriptor := module.getDescriptor().PackageDescriptor()
	return providerKey(pkgDescriptor.Org().Value(), pkgDescriptor.Name().Value())
}

func providerKey(org, pkg string) string {
	return org + "/" + pkg
}

type resolvedCompilerPlugin struct {
	provider    compilerPluginProvider
	declaration compilerPluginDeclaration
	plugin      compilerplugin.CompilerPlugin
	position    moduleImport
}

// pluginsFor returns the compiler plugins activated by the explicit imports of
// module, ordered by provider package and then by declaration order.
func (r *compilerPluginResolver) pluginsFor(module *moduleContext) []resolvedCompilerPlugin {
	providers := make(map[string]moduleImport)
	ownKey := modulePackageKey(module)
	for _, imported := range module.explicitImports {
		key, ok := r.moduleOwners[moduleIdentity{org: imported.org, moduleName: imported.moduleName}]
		if !ok || key == ownKey {
			continue
		}
		if _, exists := providers[key]; !exists {
			providers[key] = imported
		}
	}
	providerKeys := make([]string, 0, len(providers))
	for key := range providers {
		providerKeys = append(providerKeys, key)
	}
	slices.Sort(providerKeys)

	var result []resolvedCompilerPlugin
	for _, key := range providerKeys {
		provider, ok := r.providers[key]
		if !ok {
			continue
		}
		for _, linked := range provider.plugins {
			result = append(result, resolvedCompilerPlugin{
				provider: provider, declaration: linked.declaration, plugin: linked.plugin, position: providers[key],
			})
		}
	}
	return result
}

func parseCompilerPluginManifest(content string) ([]compilerPluginDeclaration, error) {
	entries, err := compilerplugin.ParseManifest(content)
	if err != nil {
		return nil, err
	}
	declarations := make([]compilerPluginDeclaration, 0, len(entries))
	for _, entry := range entries {
		declarations = append(declarations, compilerPluginDeclaration{after: entry.After, function: entry.Function})
	}
	return declarations, nil
}
