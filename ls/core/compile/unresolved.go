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
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// The compiler's type resolution assumes symbol resolution resolved every
// symbol, and panics on an empty one inside a goroutine the LS cannot
// recover. The LS still wants to type-resolve erroneous modules, so after
// symbol resolution it binds every unresolved variable reference (a
// half-typed name, or the parser's placeholder for a missing expression) to a
// stand-in symbol of type never, and keeps the module out of type resolution
// only if some other kind of symbol is still unresolved.

// bindUnresolvedReferences binds every unresolved variable reference in pkg
// to one fresh stand-in symbol of type never. never is a subtype of every
// type, so the stand-in adds as few new type errors as possible.
func bindUnresolvedReferences(ctx *context.CompilerContext, pkgID model.PackageID, pkg *ast.BLangPackage) {
	binder := &unresolvedReferenceBinder{bind: func() model.SymbolRef {
		return newStandInSymbol(ctx, pkgID)
	}}
	ast.Walk(binder, pkg)
}

func newStandInSymbol(ctx *context.CompilerContext, pkgID model.PackageID) model.SymbolRef {
	space := ctx.NewSymbolSpace(pkgID)
	symbol := model.NewVariableSymbol("", false, false, false, diagnostics.NewBuiltinLocation())
	ref := space.RefAt(space.AppendSymbol(&symbol))
	ctx.SetSymbolType(ref, semtypes.Never)
	return ref
}

type unresolvedReferenceBinder struct {
	bind    func() model.SymbolRef
	standIn model.SymbolRef
	bound   bool
}

func (b *unresolvedReferenceBinder) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if ref, ok := node.(*ast.BLangVarRef); ok && !ast.SymbolIsSet(ref) {
		ref.SetSymbol(b.standInSymbol())
	}
	return b
}

func (b *unresolvedReferenceBinder) VisitTypeData(*ast.TypeData) ast.Visitor { return b }

func (b *unresolvedReferenceBinder) standInSymbol() model.SymbolRef {
	if !b.bound {
		b.standIn = b.bind()
		b.bound = true
	}
	return b.standIn
}

// hasUnresolvedSymbols reports whether any node of pkg still has an empty
// symbol, i.e. one bindUnresolvedReferences could not bind.
func hasUnresolvedSymbols(pkg *ast.BLangPackage) bool {
	finder := &unresolvedSymbolFinder{}
	ast.Walk(finder, pkg)
	return finder.found
}

type unresolvedSymbolFinder struct {
	found bool
}

func (f *unresolvedSymbolFinder) Visit(node ast.BLangNode) ast.Visitor {
	if f.found || node == nil {
		return nil
	}
	if _, ok := node.(ast.BLangBadNode); ok {
		return nil
	}
	if symbolUnresolved(node) {
		f.found = true
		return nil
	}
	return f
}

func (f *unresolvedSymbolFinder) VisitTypeData(*ast.TypeData) ast.Visitor { return f }

// symbolUnresolved reports whether node carries a symbol that symbol
// resolution left empty. An invocation's Symbol() panics before type
// resolution when its RawSymbol is still a deferred method symbol (resolved
// by type resolution itself), so invocations are checked via RawSymbol.
func symbolUnresolved(node ast.BLangNode) bool {
	if invocation, ok := node.(*ast.BLangInvocation); ok {
		if invocation.RawSymbol == nil {
			return true
		}
		ref, ok := invocation.RawSymbol.(*model.SymbolRef)
		return ok && ref.IsEmpty()
	}
	withSymbol, ok := node.(ast.NodeWithSymbol)
	return ok && !ast.SymbolIsSet(withSymbol)
}
