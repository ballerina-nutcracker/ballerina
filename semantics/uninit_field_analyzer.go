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
	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
)

type initState struct {
	initialized map[string]bool
}

func newInitState() *initState {
	return &initState{initialized: make(map[string]bool)}
}

func (s *initState) clone() *initState {
	c := newInitState()
	for k, v := range s.initialized {
		c.initialized[k] = v
	}
	return c
}

func (s *initState) markInitialized(name string) {
	s.initialized[name] = true
}

func (s *initState) isInitialized(name string) bool {
	return s.initialized[name]
}

func mergeInitStates(s1, s2 *initState) *initState {
	if len(s1.initialized) == 0 {
		return s2.clone()
	}
	if len(s2.initialized) == 0 {
		return s1.clone()
	}
	result := newInitState()
	allNames := make(map[string]bool)
	for name := range s1.initialized {
		allNames[name] = true
	}
	for name := range s2.initialized {
		allNames[name] = true
	}
	for name := range allNames {
		result.initialized[name] = s1.initialized[name] && s2.initialized[name]
	}
	return result
}

type namedBlockState struct {
	entry *initState
	exit  *initState
}

// extractAssignedName returns the name being assigned by a CFG node, or "" if not relevant.
type extractAssignedName func(node ast.Node) string

// uninitAnalyzer performs dataflow analysis on a function's CFG to check whether
// a set of named variables are initialized on all code paths.
type uninitAnalyzer struct {
	fcfg            *functionCFG
	states          map[int]*namedBlockState
	names           []string
	extractAssigned extractAssignedName
}

func newUninitAnalyzer(fcfg *functionCFG, names []string, extract extractAssignedName) *uninitAnalyzer {
	a := &uninitAnalyzer{
		fcfg:            fcfg,
		states:          make(map[int]*namedBlockState),
		names:           names,
		extractAssigned: extract,
	}
	for i := range fcfg.bbs {
		a.states[i] = &namedBlockState{
			entry: newInitState(),
			exit:  newInitState(),
		}
	}
	return a
}

func (a *uninitAnalyzer) analyze() {
	if len(a.fcfg.bbs) == 0 {
		return
	}
	for _, i := range a.fcfg.topoOrder {
		bb := &a.fcfg.bbs[i]
		entry := a.mergePredecessors(bb)
		if i == 0 {
			for _, name := range a.names {
				if !entry.isInitialized(name) {
					entry.initialized[name] = false
				}
			}
		}
		a.states[i].entry = entry
		exit := a.analyzeBlock(bb, entry.clone())
		a.states[i].exit = exit
	}
}

func (a *uninitAnalyzer) mergePredecessors(bb *basicBlock) *initState {
	backedgeSet := make(map[int]bool, len(bb.backedgeParents))
	for _, p := range bb.backedgeParents {
		backedgeSet[p] = true
	}
	var result *initState
	for _, parentID := range bb.parents {
		if backedgeSet[parentID] {
			continue
		}
		if result == nil {
			result = a.states[parentID].exit.clone()
		} else {
			result = mergeInitStates(result, a.states[parentID].exit)
		}
	}
	if result == nil {
		result = newInitState()
	}
	return result
}

func (a *uninitAnalyzer) analyzeBlock(bb *basicBlock, state *initState) *initState {
	for _, node := range bb.nodes {
		if name := a.extractAssigned(node); name != "" {
			state.markInitialized(name)
		}
	}
	return state
}

// uninitializedNames returns the names that are not initialized on all terminal paths.
func (a *uninitAnalyzer) uninitializedNames() []string {
	var result []string
	for _, name := range a.names {
		if !a.isInitializedAtAllTerminals(name) {
			result = append(result, name)
		}
	}
	return result
}

func (a *uninitAnalyzer) isInitializedAtAllTerminals(name string) bool {
	for _, bb := range a.fcfg.bbs {
		if !bb.isTerminal() || !bb.isReachable() {
			continue
		}
		if !a.states[bb.id].exit.isInitialized(name) {
			return false
		}
	}
	return true
}

// extractFieldAssignment returns the field name for self.<field> assignments.
func extractFieldAssignment(node ast.Node) string {
	if assignment, ok := node.(*ast.BLangAssignment); ok {
		if fieldAccess, ok := assignment.VarRef.(*ast.BLangFieldBaseAccess); ok {
			if isSelfFieldAccess(fieldAccess) {
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
			if varRef, ok := assignment.VarRef.(*ast.BLangSimpleVarRef); ok {
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
		classDef := &pkg.ClassDefinitions[i]
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
