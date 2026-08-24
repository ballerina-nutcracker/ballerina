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

// Package langlib provides a test-only helper that compiles the migrated lang
// library bundles into a caller-supplied compiler context, mirroring what the
// project API does via package resolution. Hand-rolled compile drivers (which
// bypass the project API) use it to obtain the implicit-imports map.
package langlib

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/lib/langlibs"
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/nodebuilder"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

type bundledLib struct {
	org       string
	nameComps []string
	// implicitID, when set, is the key under which the lib is exposed in the
	// implicit-imports map (used without an import statement). When empty the
	// lib requires an explicit import and is exposed via publicSymbols instead.
	implicitID string
	srcFS      fs.FS  // embedded bundle filesystem the source is read from
	balPath    string // path within srcFS
	version    string
}

var migratedLangLibs = []bundledLib{
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "__internal"},
		implicitID: "lang.__internal",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.__internal/0.0.1/any/lang.__internal.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "int"},
		implicitID: "lang.int",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.int/0.0.1/any/lang.int.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "boolean"},
		implicitID: "lang.boolean",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.boolean/0.0.1/any/lang.boolean.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "decimal"},
		implicitID: "lang.decimal",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.decimal/0.0.1/any/lang.decimal.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "float"},
		implicitID: "lang.float",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.float/0.0.1/any/lang.float.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "error"},
		implicitID: "lang.error",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.error/0.0.1/any/lang.error.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "string"},
		implicitID: "lang.string",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.string/0.0.1/any/lang.string.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "value"},
		implicitID: "lang.value",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.value/0.0.1/any/lang.value.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "xml"},
		implicitID: "lang.xml",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.xml/0.0.1/any/lang.xml.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "float"},
		implicitID: "lang.float",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.float/0.0.1/any/lang.float.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "array"},
		implicitID: "lang.array",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.array/0.0.1/any/lang.array.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "map"},
		implicitID: "lang.map",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.map/0.0.1/any/lang.map.bal",
		version:    "0.0.1",
	},
	{
		org:        "ballerina",
		nameComps:  []string{"lang", "object"},
		implicitID: "lang.object",
		srcFS:      langlibs.FS,
		balPath:    "ballerina/lang.object/0.0.1/any/lang.object.bal",
		version:    "0.0.1",
	},
	{
		org:       "ballerina",
		nameComps: []string{"lang", "runtime"},
		srcFS:     langlibs.FS,
		balPath:   "ballerina/lang.runtime/0.0.1/any/lang.runtime.bal",
		version:   "0.0.1",
	},
}

var bundledStdlibs = []bundledLib{
	{
		org:       "ballerina",
		nameComps: []string{"io"},
		srcFS:     stdlibs.FS,
		balPath:   "ballerina/io/0.0.1/go1.26/io.bal",
		version:   "0.0.1",
	},
}

type Symbols struct {
	ImplicitImports map[string]model.ExportedSymbolSpace
	PublicSymbols   map[semantics.PackageIdentifier]model.ExportedSymbolSpace
}

func Build(cx *context.CompilerContext, publicSymbols map[semantics.PackageIdentifier]model.ExportedSymbolSpace) (*Symbols, error) {
	implicitImports := make(map[string]model.ExportedSymbolSpace)
	if publicSymbols == nil {
		publicSymbols = make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace)
	}

	cache := make(map[string]model.ExportedSymbolSpace)
	for _, lib := range migratedLangLibs {
		pkgID := semantics.PackageIdentifier{OrgName: lib.org, ModuleName: strings.Join(lib.nameComps, ".")}
		space, ok := publicSymbols[pkgID]
		if !ok {
			var err error
			space, err = compileBundledLib(cx, cache, lib)
			if err != nil {
				return nil, err
			}
			publicSymbols[pkgID] = space
		}
		if lib.implicitID != "" {
			implicitImports[lib.implicitID] = space
		}
	}
	for _, lib := range bundledStdlibs {
		pkgID := semantics.PackageIdentifier{OrgName: lib.org, ModuleName: strings.Join(lib.nameComps, ".")}
		if _, ok := publicSymbols[pkgID]; !ok {
			space, err := compileBundledLib(cx, cache, lib)
			if err != nil {
				return nil, err
			}
			publicSymbols[pkgID] = space
		}
	}

	return &Symbols{ImplicitImports: implicitImports, PublicSymbols: publicSymbols}, nil
}

// compileBundledLib compiles a single bundled library's source into cx and
// returns its exported symbol space, reusing a previous compilation in the same
// build if present.
func compileBundledLib(cx *context.CompilerContext, cache map[string]model.ExportedSymbolSpace, lib bundledLib) (model.ExportedSymbolSpace, error) {
	if cached, ok := cache[lib.balPath]; ok {
		return cached, nil
	}
	content, err := fs.ReadFile(lib.srcFS, lib.balPath)
	if err != nil {
		return model.ExportedSymbolSpace{}, fmt.Errorf("langlib: read %s: %w", lib.balPath, err)
	}

	cx.DiagnosticEnv().RegisterFile(lib.balPath, text.NewStringTextDocument(string(content)))
	syntaxTree, err := parser.GetSyntaxTree(cx, lib.balPath, string(content))
	if err != nil {
		return model.ExportedSymbolSpace{}, fmt.Errorf("langlib: parse %s: %w", lib.implicitID, err)
	}
	cu := nodebuilder.GetCompilationUnit(cx, syntaxTree)
	if cu == nil {
		return model.ExportedSymbolSpace{}, fmt.Errorf("langlib: AST generation failed for %s", lib.implicitID)
	}
	nameComps := make([]model.Name, len(lib.nameComps))
	for i, c := range lib.nameComps {
		nameComps[i] = model.Name(c)
	}
	pkgID := cx.NewPackageID(model.Name(lib.org), nameComps, model.Name(lib.version))
	cu.SetPackageID(pkgID)
	compilationUnits := []*ast.BLangCompilationUnit{cu}

	// Bundled libraries do not import other modules.
	pkgScope, exported, imported := semantics.ResolveSymbols(
		cx,
		*pkgID,
		compilationUnits,
		make(map[string]model.ExportedSymbolSpace),
		make(map[semantics.PackageIdentifier]model.ExportedSymbolSpace),
		lib.org,
	)
	pkg := nodebuilder.ToPackageFromCompilationUnits(cx, compilationUnits)
	if cx.HasErrors() {
		return model.ExportedSymbolSpace{}, fmt.Errorf("langlib: package assembly failed for %s", lib.implicitID)
	}
	pkg.PackageID = pkgID
	pkg.Scope = pkgScope
	pkg.Imports = nil
	semantics.ResolvePublicNodeTypes(cx, pkg, imported)
	cache[lib.balPath] = exported
	return exported, nil
}
