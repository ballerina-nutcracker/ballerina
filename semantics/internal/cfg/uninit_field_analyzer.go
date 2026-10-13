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

package cfg

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common/bitset"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
)

// extractAssignedName returns the name being assigned by a CFG node, or "" if not relevant.
type extractAssignedName func(node ast.Node) string

// uninitAnalyzer performs dataflow analysis on a function's CFG to check whether
// a set of named variables are initialized on all code paths.
type uninitAnalyzer struct {
	fcfg            *functionCFG
	names           []string
	nameIdx         *bitset.Context[string]
	namePos         []bitset.Pos // parallel to names
	declared        bitset.BitSet
	terminalInit    bitset.BitSet // intersection of init over reachable, non-panicking terminal exits
	sawTerminal     bool
	states          *initStateStore
	extractAssigned extractAssignedName
}

func newUninitAnalyzer(fcfg *functionCFG, names []string, extract extractAssignedName) *uninitAnalyzer {
	nameIdx := bitset.NewContext[string]()
	namePos := make([]bitset.Pos, len(names))
	for i, name := range names {
		namePos[i] = nameIdx.Intern(name)
	}
	declared := nameIdx.New()
	for _, p := range namePos {
		declared.Set(p)
	}
	return &uninitAnalyzer{
		fcfg:         fcfg,
		names:        names,
		nameIdx:      nameIdx,
		namePos:      namePos,
		declared:     declared,
		terminalInit: nameIdx.New(),
		sawTerminal:  false,
		states: newInitStateStore(fcfg, func() *initState {
			return &initState{tracked: nameIdx.New(), init: nameIdx.New()}
		}),
		extractAssigned: extract,
	}
}

func (a *uninitAnalyzer) analyze() {
	if len(a.fcfg.bbs) == 0 {
		return
	}
	for _, i := range a.fcfg.topoOrder {
		bb := &a.fcfg.bbs[i]
		state := a.states.entryState(bb)
		if i == 0 {
			state.tracked.Or(&a.declared)
		}
		for _, node := range bb.nodes {
			if name := a.extractAssigned(node); name != "" {
				state.markInitialized(a.nameIdx.Intern(name))
			}
		}
		a.recordTerminal(bb, state)
		a.states.setExit(bb, state)
	}
}

// recordTerminal folds the exit state of a reachable, non-panicking terminal block into terminalInit.
func (a *uninitAnalyzer) recordTerminal(bb *basicBlock, state *initState) {
	if !bb.isTerminal() || !bb.isReachable() || bb.panics {
		return
	}
	if a.sawTerminal {
		a.terminalInit.And(&state.init)
		return
	}
	a.terminalInit.CopyFrom(&state.init)
	a.sawTerminal = true
}

// uninitializedNames returns the names that are not initialized on all terminal paths that do not panic.
func (a *uninitAnalyzer) uninitializedNames() []string {
	var result []string
	for i, name := range a.names {
		if a.sawTerminal && !a.terminalInit.Has(a.namePos[i]) {
			result = append(result, name)
		}
	}
	return result
}

// extractFieldAssignment returns the field name for self.<field> assignments.
func extractFieldAssignment(node ast.Node) string {
	if assignment, ok := node.(*ast.BLangAssignment); ok {
		if fieldAccess, ok := assignment.VarRef.(*ast.BLangFieldBaseAccess); ok {
			if common.IsSelfFieldAccess(fieldAccess) {
				return fieldAccess.Field.GetValue()
			}
		}
	}
	return ""
}

func analyzeUninitializedGlobalVars(ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG) {
	globalSymbols := make(map[model.SymbolRef]bool)
	var varsNeedingInit []string
	for i := range pkg.GlobalVars {
		if pkg.GlobalVars[i].Expr == nil {
			varsNeedingInit = append(varsNeedingInit, pkg.GlobalVars[i].Name.GetValue())
			globalSymbols[pkg.GlobalVars[i].Symbol()] = true
		}
	}
	if len(varsNeedingInit) == 0 {
		return
	}
	if pkg.InitFunction == nil {
		for _, name := range varsNeedingInit {
			for i := range pkg.GlobalVars {
				if pkg.GlobalVars[i].Name.GetValue() == name {
					ctx.SemanticError("variable '"+name+"' is not initialized", pkg.GlobalVars[i].Name.GetPosition())
					break
				}
			}
		}
		return
	}
	fnCfg, ok := cfg.lookupFunctionCfg(pkg.InitFunction.Symbol())
	if !ok {
		ctx.InternalError("init function CFG not found", pkg.InitFunction.GetPosition())
		return
	}
	// Use a symbol-aware extractor that only matches assignments to the actual
	// global variables, not to shadowed locals with the same name.
	extractor := func(node ast.Node) string {
		if assignment, ok := node.(*ast.BLangAssignment); ok {
			if varRef, ok := assignment.VarRef.(*ast.BLangVarRef); ok {
				if globalSymbols[varRef.Symbol()] {
					return varRef.VariableName.GetValue()
				}
			}
		}
		return ""
	}
	analyzer := newUninitAnalyzer(&fnCfg, varsNeedingInit, extractor)
	analyzer.analyze()
	for _, name := range analyzer.uninitializedNames() {
		for i := range pkg.GlobalVars {
			if pkg.GlobalVars[i].Name.GetValue() == name {
				ctx.SemanticError("variable '"+name+"' is not initialized", pkg.GlobalVars[i].Name.GetPosition())
				break
			}
		}
	}
}

func analyzeUninitializedFields(ctx *context.CompilerContext, pkg *ast.BLangPackage, cfg *PackageCFG) {
	for i := range pkg.ClassDefinitions {
		classDef := pkg.ClassDefinitions[i]
		var fieldsNeedingInit []string
		for _, field := range classDef.Fields {
			if field.GetInitialExpression() == nil {
				fieldsNeedingInit = append(fieldsNeedingInit, field.GetName().GetValue())
			}
		}
		if len(fieldsNeedingInit) == 0 {
			continue
		}
		if classDef.InitFunction == nil {
			for _, name := range fieldsNeedingInit {
				for _, field := range classDef.Fields {
					if field.GetName().GetValue() == name {
						ctx.SemanticError("field '"+name+"' is not initialized", field.GetPosition())
						break
					}
				}
			}
			continue
		}
		fnCfg, ok := cfg.lookupFunctionCfg(classDef.InitFunction.Symbol())
		if !ok {
			ctx.InternalError("init function CFG not found", classDef.InitFunction.GetPosition())
			continue
		}
		analyzer := newUninitAnalyzer(&fnCfg, fieldsNeedingInit, extractFieldAssignment)
		analyzer.analyze()
		for _, name := range analyzer.uninitializedNames() {
			for _, field := range classDef.Fields {
				if field.GetName().GetValue() == name {
					ctx.SemanticError("field '"+name+"' is not initialized", field.GetPosition())
					break
				}
			}
		}
	}
}
