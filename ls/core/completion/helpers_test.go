// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Shared-helper unit tests for the two helpers this ticket ships but wires
// to no handler yet (no invocation- or import-family handler exists to
// drive them through a real LSP request): functionSignatureLabel and
// moduleContentSymbols. expectedTypeCompatible (the third shared helper) is
// already exercised transitively by the existing typed-initializer
// completion fixtures via lexicalItems, so it gets no separate unit test
// here.
package completion

import (
	"testing"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func TestFunctionSignatureLabel(t *testing.T) {
	tests := []struct {
		name string
		sig  model.UntypedFunctionSignature
		want string
	}{
		{
			name: "no params",
			sig:  model.NewUntypedFunctionSignature(nil, false),
			want: "()",
		},
		{
			name: "fixed params",
			sig: model.NewUntypedFunctionSignature([]model.Param{
				{Name: "a"},
				{Name: "b"},
			}, false),
			want: "(a, b)",
		},
		{
			name: "rest param",
			sig: model.NewUntypedFunctionSignature([]model.Param{
				{Name: "a"},
				{Name: "rest", Flag: model.ParamFlagRestParam},
			}, true),
			want: "(a, ...rest)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := functionSignatureLabel(test.sig); got != test.want {
				t.Errorf("functionSignatureLabel(%+v) = %q, want %q", test.sig, got, test.want)
			}
		})
	}
}

func TestModuleContentSymbols(t *testing.T) {
	org, pkg, version := model.Name("testorg"), model.Name("testpkg"), model.Name("0.1.0")
	pkgID := model.PackageID{OrgName: &org, PkgName: &pkg, Version: &version}
	main := model.NewSymbolSpaceInner(pkgID, 0)
	visible := model.NewVariableSymbol("println", true, false, false, diagnostics.Location{})
	hidden := model.NewVariableSymbol("hidden", false, false, false, diagnostics.Location{})
	main.AddSymbol("println", &visible)
	main.AddSymbol("hidden", &hidden)
	exported := model.NewExportedSymbolSpaces([]*model.SymbolSpace{main}, nil)

	moduleScope := &model.ModuleScope{
		Main:   model.NewSymbolSpaceInner(pkgID, 1),
		Prefix: map[string]model.ExportedSymbolSpace{"io": exported},
	}

	t.Run("found at module scope", func(t *testing.T) {
		space, ok := moduleContentSymbols(moduleScope, "io")
		if !ok {
			t.Fatal("moduleContentSymbols(_, \"io\") ok = false, want true")
		}
		if _, ok := space.GetSymbol("println"); !ok {
			t.Error("expected space to contain public symbol \"println\"")
		}
		if _, ok := space.GetSymbol("hidden"); ok {
			t.Error("expected space to exclude non-public symbol \"hidden\"")
		}
	})

	t.Run("unbound prefix", func(t *testing.T) {
		if _, ok := moduleContentSymbols(moduleScope, "nope"); ok {
			t.Error("moduleContentSymbols(_, \"nope\") ok = true, want false")
		}
	})

	t.Run("walks outward through block scopes", func(t *testing.T) {
		fnScope := &model.FunctionScope{BlockScopeBase: model.BlockScopeBase{Parent: moduleScope, Main: model.NewSymbolSpaceInner(pkgID, 2)}}
		blockScope := &model.BlockScope{BlockScopeBase: model.BlockScopeBase{Parent: fnScope, Main: model.NewSymbolSpaceInner(pkgID, 3)}}

		space, ok := moduleContentSymbols(blockScope, "io")
		if !ok {
			t.Fatal("moduleContentSymbols(blockScope, \"io\") ok = false, want true")
		}
		if _, ok := space.GetSymbol("println"); !ok {
			t.Error("expected space walked from a nested block scope to still contain \"println\"")
		}
	})

	t.Run("nil scope", func(t *testing.T) {
		if _, ok := moduleContentSymbols(nil, "io"); ok {
			t.Error("moduleContentSymbols(nil, \"io\") ok = true, want false")
		}
	})

	t.Run("package scope delegates to virtual module scope", func(t *testing.T) {
		pkgScope := &model.PackageScope{Virtual: moduleScope}
		space, ok := moduleContentSymbols(pkgScope, "io")
		if !ok {
			t.Fatal("moduleContentSymbols(pkgScope, \"io\") ok = false, want true")
		}
		if _, ok := space.GetSymbol("println"); !ok {
			t.Error("expected space reached via PackageScope.Virtual to still contain \"println\"")
		}
	})

	t.Run("package scope with nil virtual", func(t *testing.T) {
		if _, ok := moduleContentSymbols(&model.PackageScope{}, "io"); ok {
			t.Error("moduleContentSymbols(&model.PackageScope{}, \"io\") ok = true, want false")
		}
	})
}
