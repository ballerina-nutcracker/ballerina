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

package semantics

import (
	"sync"

	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
)

func AnalyzeCFG(ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG) {
	var wg sync.WaitGroup
	wg.Go(func() { analyzeReachability(ctx, cfg) })
	wg.Go(func() { analyzeExplicitReturn(ctx, pkg, cfg) })
	wg.Go(func() { analyzeUninitializedVars(ctx, pkg, cfg) })
	wg.Go(func() { analyzeUninitializedFields(ctx, pkg, cfg) })
	wg.Go(func() { analyzeUninitializedGlobalVars(ctx, pkg, cfg) })
	wg.Wait()
}

// analyzeReachability checks for unreachable code in all functions.
// This is now a private function called by AnalyzeCFG.
func analyzeReachability(ctx *context.CompilerContext, cfg *PackageCFG) {
	var wg sync.WaitGroup
	for _, fcfg := range cfg.allFunctionCfgs {
		wg.Add(1)
		go func(fcfg *functionCFG) {
			defer wg.Done()
			for _, bb := range fcfg.bbs {
				if !bb.isReachable() {
					for _, node := range bb.nodes {
						ctx.SemanticError("unreachable code", node.GetPosition())
					}
				}
			}
		}(fcfg)
	}
	wg.Wait()
}

// analyzeExplicitReturn validates that functions with non-nil return types
// have explicit return statements.
// This is now a private function called by AnalyzeCFG.
func analyzeExplicitReturn(ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG) {
	var wg sync.WaitGroup
	spawn := func(n invokableNode) {
		wg.Go(func() { analyzeInvokableExplicitReturn(ctx, n, cfg) })
	}
	for i := range pkg.Functions {
		spawn(&pkg.Functions[i])
	}
	spawnObjectMembers := func(methods map[string]*ast.BLangFunction, resourceMethods []*ast.BLangResourceMethod) {
		for _, method := range methods {
			spawn(method)
		}
		for _, rm := range resourceMethods {
			spawn(rm)
		}
	}
	for i := range pkg.ClassDefinitions {
		c := &pkg.ClassDefinitions[i]
		spawnObjectMembers(c.Methods, c.ResourceMethods)
	}
	for i := range pkg.Services {
		s := &pkg.Services[i]
		spawnObjectMembers(s.Methods, s.ResourceMethods)
	}
	wg.Wait()
}

type invokableNode interface {
	ast.BLangNode
	IsNative() bool
	Symbol() model.SymbolRef
}

func analyzeInvokableExplicitReturn(ctx *context.CompilerContext, fn invokableNode, cfg *PackageCFG) {
	if fn.IsNative() {
		return
	}
	sym := ctx.GetSymbol(fn.Symbol()).(model.FunctionSymbol)
	retType := sym.Signature().ReturnType
	if semtypes.ContainsBasicType(retType, semtypes.NIL) {
		return
	}

	fnCfg, ok := cfg.lookupFunctionCfg(fn.Symbol())
	if !ok {
		return
	}
	if semtypes.IsNever(retType) {
		analyzeFunctionNeverReturn(ctx, fn, fnCfg)
		return
	}

	for _, bb := range fnCfg.bbs {
		if !bb.isTerminal() || !bb.isReachable() {
			continue
		}
		if terminalBlockHasReturnOrPanic(bb) {
			continue
		}
		pos := positionForMissingReturn(bb, fn)
		ctx.SemanticError("missing return statement", pos)
	}
}

func analyzeFunctionNeverReturn(ctx *context.CompilerContext, fn invokableNode, fnCfg functionCFG) {
	for _, bb := range fnCfg.bbs {
		if !bb.isTerminal() || !bb.isReachable() {
			continue
		}
		if terminalBlockHasPanic(bb) {
			continue
		}
		ctx.SemanticError("expected panic", positionForMissingReturn(bb, fn))
	}
}

func terminalBlockHasReturnOrPanic(bb basicBlock) bool {
	if len(bb.nodes) == 0 {
		return false
	}
	last := bb.nodes[len(bb.nodes)-1]
	switch last.(type) {
	case *ast.BLangReturn, *ast.BLangPanic:
		return true
	case *ast.BLangExpressionStmt:
		// The only other way a reachable block becomes terminal is via a
		// `check`/`checkpanic` expression statement whose operand is
		// statically a subtype of error (see analyzeStatement in
		// control_flow_analyzer.go).
		return true
	default:
		return false
	}
}

func terminalBlockHasPanic(bb basicBlock) bool {
	if len(bb.nodes) == 0 {
		return false
	}
	_, ok := bb.nodes[len(bb.nodes)-1].(*ast.BLangPanic)
	return ok
}

func positionForMissingReturn(bb basicBlock, fn ast.BLangNode) diagnostics.Location {
	if len(bb.nodes) > 0 {
		return bb.nodes[len(bb.nodes)-1].GetPosition()
	}
	return fn.GetPosition()
}
