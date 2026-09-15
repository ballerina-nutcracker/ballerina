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
	goast "go/ast"
	"go/token"
	"slices"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/common/tomlparser"
	"github.com/ballerina-nutcracker/ballerina/compilerplugin"
	"github.com/ballerina-nutcracker/ballerina/compilerpluginregistry"
	"github.com/ballerina-nutcracker/ballerina/semantics"
)

const afterSemanticsName = "after-semantics"

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
	org          string
	pkg          string
	exported     semantics.PackageIdentifier
	declarations []compilerPluginDeclaration
	manifestErr  error
}

type compilerPluginResolver struct {
	registry     *compilerPluginRegistry
	moduleOwners map[moduleIdentity]string
	providers    map[string]compilerPluginProvider
	rootPackage  PackageDescriptor
	injected     []compilerplugin.InjectedPlugin
}

func newCompilerPluginResolver(
	modules []*moduleContext,
	rootPackage PackageDescriptor,
	injected []compilerplugin.InjectedPlugin,
) *compilerPluginResolver {
	resolver := &compilerPluginResolver{
		registry:     getLinkedCompilerPluginRegistry(),
		moduleOwners: make(map[moduleIdentity]string),
		providers:    make(map[string]compilerPluginProvider),
		rootPackage:  rootPackage,
		injected:     injected,
	}
	for _, module := range modules {
		descriptor := module.getDescriptor()
		pkgDescriptor := descriptor.PackageDescriptor()
		key := providerKey(pkgDescriptor.Org().Value(), pkgDescriptor.Name().Value())
		resolver.moduleOwners[moduleIdentity{
			org: descriptor.Org().Value(), moduleName: descriptor.Name().String(),
		}] = key
		if _, exists := resolver.providers[key]; exists {
			continue
		}
		var declarations []compilerPluginDeclaration
		var manifestErr error
		// The manifest has to come from the package that owns this module. The
		// module list spans every package in the resolution, and
		// project.CurrentPackage() is only the module's own package for
		// single-package projects (a workspace project reports its first
		// member's package for every module it holds).
		pkgCtx := owningPackageContext(module.project, pkgDescriptor)
		if pkgCtx == nil {
			manifestErr = fmt.Errorf("cannot locate package %s declaring module %s",
				key, descriptor.Name().String())
		} else if manifestCtx := pkgCtx.getCompilerPluginTomlContext(); manifestCtx != nil {
			declarations, manifestErr = parseCompilerPluginManifest(manifestCtx.content)
		}
		resolver.providers[key] = compilerPluginProvider{
			org:          pkgDescriptor.Org().Value(),
			pkg:          pkgDescriptor.Name().Value(),
			declarations: declarations,
			manifestErr:  manifestErr,
			exported: semantics.PackageIdentifier{
				OrgName: pkgDescriptor.Org().Value(), ModuleName: pkgDescriptor.Name().Value(),
			},
		}
	}
	return resolver
}

// owningPackageContext returns the context of the package that declares the
// module being resolved. project.CurrentPackage() cannot be used directly: it
// is the module's own package only for single-package projects, while a
// workspace project reports its first member's package for every module it
// holds.
func owningPackageContext(project Project, pkgDescriptor PackageDescriptor) *packageContext {
	if workspace, isWorkspace := project.(*WorkspaceProject); isWorkspace {
		for _, member := range workspace.Projects() {
			if pkg := member.CurrentPackage(); pkg != nil &&
				pkg.packageCtx.getDescriptor().Equals(pkgDescriptor) {
				return pkg.packageCtx
			}
		}
		return nil
	}
	pkg := project.CurrentPackage()
	if pkg == nil || !pkg.packageCtx.getDescriptor().Equals(pkgDescriptor) {
		return nil
	}
	return pkg.packageCtx
}

func providerKey(org, pkg string) string {
	return org + "/" + pkg
}

type resolvedCompilerPlugin struct {
	provider    compilerPluginProvider
	declaration compilerPluginDeclaration
	plugin      compilerplugin.CompilerPlugin
	position    moduleImport
	injected    bool
}

// injectedPluginFunctionName stands in for the manifest-declared Go function
// name in diagnostics about host-injected plugins, which have none.
const injectedPluginFunctionName = "<injected>"

func (r *compilerPluginResolver) pluginsFor(module *moduleContext) ([]resolvedCompilerPlugin, error) {
	providers := make(map[string]moduleImport)
	modulePackage := module.getDescriptor().PackageDescriptor()
	modulePackageKey := providerKey(modulePackage.Org().Value(), modulePackage.Name().Value())
	for _, imported := range module.explicitImports {
		key, ok := r.moduleOwners[moduleIdentity{org: imported.org, moduleName: imported.moduleName}]
		if !ok || key == modulePackageKey {
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
		if provider.manifestErr != nil {
			return nil, fmt.Errorf("invalid CompilerPlugin.toml for %s: %w", key, provider.manifestErr)
		}
		for _, declaration := range provider.declarations {
			plugin, ok := r.registry.lookup(provider.org, provider.pkg, declaration)
			if !ok {
				return nil, fmt.Errorf(
					"compiler plugin implementation not linked for %s:%s at %s",
					key, declaration.function, compilerPluginStageName(declaration.after),
				)
			}
			result = append(result, resolvedCompilerPlugin{
				provider: provider, declaration: declaration, plugin: plugin, position: providers[key],
			})
		}
	}
	return append(result, r.injectedPluginsFor(module, modulePackage)...), nil
}

// injectedPluginsFor returns the host-injected plugins that apply to module:
// those whose provider package is explicitly imported by a root-package module.
func (r *compilerPluginResolver) injectedPluginsFor(
	module *moduleContext, modulePackage PackageDescriptor,
) []resolvedCompilerPlugin {
	if len(r.injected) == 0 || !modulePackage.Equals(r.rootPackage) {
		return nil
	}
	var result []resolvedCompilerPlugin
	for _, injected := range r.injected {
		key := providerKey(injected.Provider.Org, injected.Provider.Package)
		imported, ok := r.explicitImportOf(module, key)
		if !ok {
			continue
		}
		provider, ok := r.providers[key]
		if !ok {
			continue
		}
		result = append(result, resolvedCompilerPlugin{
			provider: provider,
			declaration: compilerPluginDeclaration{
				after: injected.Plugin.After, function: injectedPluginFunctionName,
			},
			plugin:   injected.Plugin,
			position: imported,
			injected: true,
		})
	}
	return result
}

// explicitImportOf reports whether module explicitly imports any module owned
// by the package providerKey identifies. The import is matched through
// moduleOwners rather than by module name, because compilerplugin.Provider
// names a package: importing a non-default module such as
// ballerina/test.helpers still activates the ballerina/test provider.
func (r *compilerPluginResolver) explicitImportOf(
	module *moduleContext, providerKey string,
) (moduleImport, bool) {
	for _, imported := range module.explicitImports {
		owner, known := r.moduleOwners[moduleIdentity{org: imported.org, moduleName: imported.moduleName}]
		if known && owner == providerKey {
			return imported, true
		}
	}
	return moduleImport{}, false
}

func parseCompilerPluginManifest(content string) ([]compilerPluginDeclaration, error) {
	doc, err := tomlparser.ReadString(content)
	if err != nil {
		return nil, err
	}
	tables, ok := doc.GetTables("plugin")
	if !ok || len(tables) == 0 {
		return nil, fmt.Errorf("CompilerPlugin.toml must contain at least one [[plugin]] entry")
	}
	declarations := make([]compilerPluginDeclaration, 0, len(tables))
	seen := make(map[string]struct{}, len(tables))
	for i, table := range tables {
		stageName, ok := table.GetString("stage")
		if !ok {
			return nil, fmt.Errorf("plugin entry %d must define a string stage", i+1)
		}
		if stageName != afterSemanticsName {
			return nil, fmt.Errorf("plugin entry %d has unsupported stage %q", i+1, stageName)
		}
		function, ok := table.GetString("function")
		if !ok {
			return nil, fmt.Errorf("plugin entry %d must define a string function", i+1)
		}
		if !token.IsIdentifier(function) || !goast.IsExported(function) {
			return nil, fmt.Errorf("plugin entry %d has invalid exported Go function %q", i+1, function)
		}
		key := stageName + "\x00" + function
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate plugin declaration for %s at %s", function, stageName)
		}
		seen[key] = struct{}{}
		declarations = append(declarations, compilerPluginDeclaration{
			after: compilerplugin.AfterSemantics, function: function,
		})
	}
	return declarations, nil
}

func compilerPluginStageName(stage compilerplugin.Stage) string {
	switch stage {
	case compilerplugin.AfterSemantics:
		return afterSemanticsName
	default:
		return fmt.Sprintf("unknown-stage-%d", stage)
	}
}
