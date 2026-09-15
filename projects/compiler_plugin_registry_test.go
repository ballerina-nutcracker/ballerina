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
	"errors"
	"os"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/compilerplugin"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func TestCompilerPluginActivationRequiresExplicitImport(t *testing.T) {
	version, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	appDescriptor := NewModuleDescriptorForDefaultModule(NewPackageDescriptor(
		NewPackageOrg("acme"), NewPackageName("app"), version,
	))
	httpProvider := compilerPluginProvider{
		org: "ballerina", pkg: "http",
		declarations: []compilerPluginDeclaration{{
			after: compilerplugin.AfterSemantics, function: "ValidateService",
		}},
	}
	resolver := &compilerPluginResolver{
		registry: newCompilerPluginRegistry(),
		moduleOwners: map[moduleIdentity]string{
			{org: "ballerina", moduleName: "http"}: "ballerina/http",
			{org: "acme", moduleName: "helper"}:    "acme/helper",
		},
		providers: map[string]compilerPluginProvider{"ballerina/http": httpProvider},
	}

	direct := &moduleContext{
		moduleDescriptor: appDescriptor,
		explicitImports:  []moduleImport{{org: "ballerina", moduleName: "http"}},
	}
	plugins, err := resolver.pluginsFor(direct)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 || plugins[0].declaration.function != "ValidateService" {
		t.Fatalf("unexpected direct-import plugins: %#v", plugins)
	}

	transitiveOnly := &moduleContext{
		moduleDescriptor: appDescriptor,
		explicitImports:  []moduleImport{{org: "acme", moduleName: "helper"}},
	}
	plugins, err = resolver.pluginsFor(transitiveOnly)
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Fatalf("transitive import activated plugins: %#v", plugins)
	}
}

func TestCompilerPluginResolverRejectsUnlinkedDeclaration(t *testing.T) {
	version, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	appDescriptor := NewModuleDescriptorForDefaultModule(NewPackageDescriptor(
		NewPackageOrg("acme"), NewPackageName("app"), version,
	))
	resolver := &compilerPluginResolver{
		registry: newCompilerPluginRegistry(),
		moduleOwners: map[moduleIdentity]string{
			{org: "acme", moduleName: "provider"}: "acme/provider",
		},
		providers: map[string]compilerPluginProvider{
			"acme/provider": {
				org: "acme", pkg: "provider",
				declarations: []compilerPluginDeclaration{{
					after: compilerplugin.AfterSemantics, function: "Missing",
				}},
			},
		},
	}
	module := &moduleContext{
		moduleDescriptor: appDescriptor,
		explicitImports:  []moduleImport{{org: "acme", moduleName: "provider"}},
	}

	if _, err := resolver.pluginsFor(module); err == nil {
		t.Fatal("expected an unlinked compiler plugin error")
	}

	typeEnv := semtypes.CreateTypeEnv()
	compilerEnv := compilercontext.NewCompilerEnvironment(typeEnv, false)
	module.cfg = &semantics.PackageCFG{}
	module.bLangPkg = &ast.BLangPackage{}
	module.compilerCtx = compilercontext.NewCompilerContext(compilerEnv)
	compilation := &PackageCompilation{compilerPluginManager: resolver}
	compilation.runCompilerPlugins(module)
	diagnostics := module.compilerCtx.Diagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(diagnostics))
	}
	if got := diagnostics[0].DiagnosticInfo().Code(); got != "SEMANTIC_ERROR" {
		t.Fatalf("diagnostic code = %q, want SEMANTIC_ERROR", got)
	}
}

func TestCompilerPluginResultsAreThreadedInDeclarationOrder(t *testing.T) {
	initial := &ast.BLangPackage{}
	firstResult := &ast.BLangPackage{}
	secondResult := &ast.BLangPackage{}
	plugins := []testCompilerPlugin{
		{name: "First", transformer: func(_ *compilercontext.CompilerContext, _ model.ExportedSymbolSpace, pkg *ast.BLangPackage) (*ast.BLangPackage, error) {
			if pkg != initial {
				t.Fatalf("first transformer received %p, want %p", pkg, initial)
			}
			return firstResult, nil
		}},
		{name: "Second", transformer: func(_ *compilercontext.CompilerContext, _ model.ExportedSymbolSpace, pkg *ast.BLangPackage) (*ast.BLangPackage, error) {
			if pkg != firstResult {
				t.Fatalf("second transformer received %p, want %p", pkg, firstResult)
			}
			return secondResult, nil
		}},
	}
	compilation, module := newCompilerPluginTestCompilation(t, initial, plugins)

	compilation.runCompilerPlugins(module)

	if module.bLangPkg != secondResult {
		t.Fatalf("final package = %p, want %p", module.bLangPkg, secondResult)
	}
	if module.compilerCtx.HasErrors() {
		t.Fatalf("unexpected diagnostics: %#v", module.compilerCtx.Diagnostics())
	}
}

func TestCompilerPluginFailuresBecomeFatalDiagnostics(t *testing.T) {
	tests := []struct {
		name        string
		transformer compilerplugin.PackageTransformer
	}{
		{name: "returned error", transformer: func(_ *compilercontext.CompilerContext, _ model.ExportedSymbolSpace, _ *ast.BLangPackage) (*ast.BLangPackage, error) {
			return nil, errors.New("plugin failure")
		}},
		{name: "nil package", transformer: func(_ *compilercontext.CompilerContext, _ model.ExportedSymbolSpace, _ *ast.BLangPackage) (*ast.BLangPackage, error) {
			return nil, nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := &ast.BLangPackage{}
			compilation, module := newCompilerPluginTestCompilation(t, initial, []testCompilerPlugin{{
				name: "Fail", transformer: tt.transformer,
			}})

			compilation.runCompilerPlugins(module)

			if !module.compilerCtx.HasErrors() {
				t.Fatal("expected a fatal compiler diagnostic")
			}
			if module.bLangPkg != initial {
				t.Fatal("failed transformer replaced the package")
			}
		})
	}
}

type testCompilerPlugin struct {
	name        string
	transformer compilerplugin.PackageTransformer
}

func newCompilerPluginTestCompilation(
	t *testing.T,
	initial *ast.BLangPackage,
	plugins []testCompilerPlugin,
) (*PackageCompilation, *moduleContext) {
	t.Helper()
	typeEnv := semtypes.CreateTypeEnv()
	compilerEnv := compilercontext.NewCompilerEnvironment(typeEnv, false)
	env := NewEnvironment(os.DirFS(t.TempDir()), compilerEnv)
	project := newBuildProjectWithEnv("", NewBuildOptions(), env)
	compilerCtx := compilercontext.NewCompilerContext(compilerEnv)

	version, err := NewPackageVersionFromString("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	appDescriptor := NewModuleDescriptorForDefaultModule(NewPackageDescriptor(
		NewPackageOrg("acme"), NewPackageName("app"), version,
	))
	manifest := ""
	registry := newCompilerPluginRegistry()
	for _, plugin := range plugins {
		manifest += "[[plugin]]\nstage = \"after-semantics\"\nfunction = \"" + plugin.name + "\"\n"
		registry.register("acme", "provider", plugin.name, compilerplugin.CompilerPlugin{
			After:              compilerplugin.AfterSemantics,
			PackageTransformer: plugin.transformer,
		})
	}
	providerID := semantics.PackageIdentifier{OrgName: "acme", ModuleName: "provider"}
	env.publicSymbols[providerID] = model.ExportedSymbolSpace{}
	resolver := &compilerPluginResolver{
		registry: registry,
		moduleOwners: map[moduleIdentity]string{
			{org: "acme", moduleName: "provider"}: "acme/provider",
		},
		providers: map[string]compilerPluginProvider{
			"acme/provider": {
				org: "acme", pkg: "provider", exported: providerID,
				declarations: mustParseCompilerPluginManifest(t, manifest),
			},
		},
	}
	module := &moduleContext{
		moduleDescriptor: appDescriptor,
		explicitImports:  []moduleImport{{org: "acme", moduleName: "provider"}},
		cfg:              &semantics.PackageCFG{},
		bLangPkg:         initial,
		compilerCtx:      compilerCtx,
	}
	compilation := &PackageCompilation{
		rootPackageContext:    &packageContext{project: project},
		compilerPluginManager: resolver,
	}
	return compilation, module
}

func mustParseCompilerPluginManifest(t *testing.T, content string) []compilerPluginDeclaration {
	t.Helper()
	declarations, err := parseCompilerPluginManifest(content)
	if err != nil {
		t.Fatal(err)
	}
	return declarations
}
