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
	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
)

type bindingFlags uint8

const (
	bindingFlagFunctionBoundary bindingFlags = 1 << iota
	bindingFlagQueryAggregated
)

type binding struct {
	// ref is the underlying symbol we are narrowing. This is never a narrowed symbol
	ref            model.SymbolRef
	narrowedSymbol model.SymbolRef
	prev           *binding
	flags          bindingFlags
	// assignmentPos is set on unnarrowing entries created by an assignment
	// statement. The loop arms use it to report assignments to variables
	// narrowed outside the loop whose effect reaches the loop top via the
	// body's natural completion or a continue path.
	assignmentPos diagnostics.Location
	// defaultType is used for unreachable branches (e.g. false branch of constant true)
	// see https://github.com/ballerina-platform/ballerina-spec/issues/1029
	defaultType semtypes.SemType
}

func (b *binding) hasFlag(flag bindingFlags) bool {
	return b.flags&flag != 0
}

func (b *binding) isUnnarrowing() bool {
	return b.ref == b.narrowedSymbol
}

// isAssignment reports whether this binding entry was produced by an
// assignment statement (carrying that statement's source position).
func (b *binding) isAssignment() bool {
	return !diagnostics.IsLocationEmpty(b.assignmentPos)
}

type expressionEffect struct {
	ifTrue  *binding
	ifFalse *binding
}

type statementEffect struct {
	binding *binding
	// nonCompletion indicates the statement is return/panic etc which spec treats narrowed type as never
	nonCompletion bool
}

// lookupBinding returns the effective symbol for a given base symbol at the current point.
// Returns (effectiveSymbol, isNarrowed, isCaptured).
// isCaptured is true when a narrowed variable was found beyond a function boundary marker,
// meaning it's captured by a closure. In that case, the unnarrowed base symbol is returned.
func lookupBinding(chain *binding, ref model.SymbolRef) (model.SymbolRef, bool, bool) {
	return lookupBindingInner(chain, ref, false)
}

func lookupQueryAggregatedBinding(chain *binding, ref model.SymbolRef) bool {
	return lookupQueryAggregatedBindingInner(chain, ref, false)
}

func lookupQueryAggregatedBindingInner(chain *binding, ref model.SymbolRef, crossedBoundary bool) bool {
	if chain == nil {
		return false
	}
	if chain.hasFlag(bindingFlagFunctionBoundary) {
		return lookupQueryAggregatedBindingInner(chain.prev, ref, true)
	}
	if chain.ref == ref || chain.narrowedSymbol == ref {
		return !crossedBoundary && !chain.isUnnarrowing() && chain.hasFlag(bindingFlagQueryAggregated)
	}
	return lookupQueryAggregatedBindingInner(chain.prev, ref, crossedBoundary)
}

func lookupBindingInner(chain *binding, ref model.SymbolRef, crossedBoundary bool) (model.SymbolRef, bool, bool) {
	if chain == nil {
		return ref, false, false
	}
	if chain.hasFlag(bindingFlagFunctionBoundary) {
		return lookupBindingInner(chain.prev, ref, true)
	}
	if chain.ref == ref {
		isNarrowed := !chain.isUnnarrowing()
		if crossedBoundary && isNarrowed {
			// Captured narrowed variable — return unnarrowed base symbol
			return ref, false, true
		}
		return chain.narrowedSymbol, isNarrowed, false
	}
	return lookupBindingInner(chain.prev, ref, crossedBoundary)
}

func narrowSymbol(t typeResolver, underlying model.SymbolRef, ty semtypes.SemType) model.SymbolRef {
	narrowedSymbol := t.createNarrowedSymbol(underlying)
	t.setSymbolType(narrowedSymbol, ty)
	return narrowedSymbol
}

func unnarrowSymbol(t typeResolver, chain *binding, symbol model.SymbolRef) statementEffect {
	return unnarrowSymbolAt(t, chain, symbol, diagnostics.Location{})
}

// unnarrowSymbolAt is unnarrowSymbol but records the position of the
// assignment that triggered the unnarrowing. Loop arms use this position to
// report assignments to variables narrowed outside the loop whose effect
// reaches the loop top.
func unnarrowSymbolAt(t typeResolver, chain *binding, symbol model.SymbolRef, pos diagnostics.Location) statementEffect {
	_, isNarrowed, isCaptured := lookupBinding(chain, symbol)
	if isCaptured {
		t.trackCapturedVar(symbol)
	}
	if !isNarrowed {
		return statementEffect{chain, false}
	}
	chain = &binding{
		ref:            symbol,
		narrowedSymbol: symbol,
		prev:           chain,
		assignmentPos:  pos,
	}
	return statementEffect{chain, false}
}

// reportOutsideLoopAssignments walks chains that flow back to the top of an
// enclosing loop (the body's natural completion and every continue path) and
// emits a semantic error for each assignment-introduced unnarrowing entry
// whose target is narrowed in the loop's entry chain. The walk stops at
// loopEntry: anything below it belongs to the surrounding scope.
func reportOutsideLoopAssignments(t typeResolver, chains []*binding, loopEntry *binding) {
	for _, chain := range chains {
		seen := make(map[model.SymbolRef]bool)
		for c := chain; c != nil && c != loopEntry; c = c.prev {
			if c.hasFlag(bindingFlagFunctionBoundary) {
				continue
			}
			if seen[c.ref] {
				continue
			}
			seen[c.ref] = true
			if !c.isAssignment() {
				continue
			}
			if _, isNarrowed, _ := lookupBinding(loopEntry, c.ref); isNarrowed {
				t.semanticError("cannot assign to a variable narrowed outside the enclosing loop", c.assignmentPos)
			}
		}
	}
}

func accumNarrowedTypes(t typeResolver, chain *binding, accum map[model.SymbolRef]semtypes.SemType, accumDefault semtypes.SemType) semtypes.SemType {
	if chain == nil {
		return accumDefault
	}
	if chain.hasFlag(bindingFlagFunctionBoundary) {
		// This is just a marker move to the next one
		return accumNarrowedTypes(t, chain.prev, accum, accumDefault)
	}
	if semtypes.IsZero(chain.defaultType) {
		ref := chain.ref
		_, hasTy := accum[ref]
		if !hasTy {
			accum[ref] = t.symbolType(chain.narrowedSymbol)
		}
	} else if semtypes.IsZero(accumDefault) {
		accumDefault = chain.defaultType
	}
	return accumNarrowedTypes(t, chain.prev, accum, accumDefault)
}

func mergeChains(t typeResolver, c1 *binding, c2 *binding, mergeOp func(semtypes.SemType, semtypes.SemType) semtypes.SemType) *binding {
	m1 := make(map[model.SymbolRef]semtypes.SemType)
	d1 := accumNarrowedTypes(t, c1, m1, semtypes.SemType{})
	m2 := make(map[model.SymbolRef]semtypes.SemType)
	d2 := accumNarrowedTypes(t, c2, m2, semtypes.SemType{})
	type typePair struct{ ty1, ty2 semtypes.SemType }
	pairs := make(map[model.SymbolRef]typePair)
	for s, ty1 := range m1 {
		ty2, ok := m2[s]
		if !ok {
			if !semtypes.IsZero(d2) {
				ty2 = d2
			} else {
				ty2 = t.symbolType(s)
			}
		}
		pairs[s] = typePair{ty1, ty2}
	}
	for s, ty2 := range m2 {
		if _, ok := m1[s]; !ok {
			if !semtypes.IsZero(d1) {
				pairs[s] = typePair{d1, ty2}
			} else {
				pairs[s] = typePair{t.symbolType(s), ty2}
			}
		}
	}
	var result *binding
	for s, p := range pairs {
		ty := mergeOp(p.ty1, p.ty2)
		sym := narrowSymbol(t, s, ty)
		result = &binding{
			ref:            s,
			narrowedSymbol: sym,
			prev:           result,
		}
	}
	return result
}

func mergeStatementEffects(t typeResolver, s1, s2 statementEffect) statementEffect {
	if s1.nonCompletion {
		return s2
	}
	if s2.nonCompletion {
		return s1
	}
	combined := mergeChains(t, s1.binding, s2.binding, semtypes.Union)
	return statementEffect{combined, false}
}

func diff(c1, c2 *binding) *binding {
	if c1 == c2 {
		return nil
	}
	result := &binding{ref: c1.ref, narrowedSymbol: c1.narrowedSymbol, flags: c1.flags, defaultType: c1.defaultType}
	cur := result
	parent := c1.prev
	for parent != nil && parent != c2 {
		cur.prev = &binding{ref: parent.ref, narrowedSymbol: parent.narrowedSymbol, flags: parent.flags, defaultType: parent.defaultType}
		cur = cur.prev
		parent = parent.prev
	}
	return result
}

func singletonExprEffect(chain *binding, expr ast.BLangActionOrExpression) (expressionEffect, bool) {
	return singletonResultEffect(chain, expr.GetDeterminedType())
}

func singletonResultEffect(chain *binding, ty semtypes.SemType) (expressionEffect, bool) {
	if semtypes.IsZero(ty) {
		return expressionEffect{}, false
	}
	if isSingletonBool(ty, true) {
		return expressionEffect{ifTrue: chain, ifFalse: &binding{defaultType: semtypes.NEVER, prev: chain}}, true
	} else if isSingletonBool(ty, false) {
		return expressionEffect{ifTrue: &binding{defaultType: semtypes.NEVER, prev: chain}, ifFalse: chain}, true
	}
	return expressionEffect{}, false
}

func defaultExpressionEffect(chain *binding) expressionEffect {
	return expressionEffect{ifTrue: chain, ifFalse: chain}
}

func defaultStmtEffect(chain *binding) statementEffect {
	return statementEffect{binding: chain, nonCompletion: false}
}

func varRefExp(chain *binding, expr ast.BLangActionOrExpression) (model.SymbolRef, bool) {
	baseSymbol, isVarRef := varRefExpInner(expr)
	if !isVarRef {
		return baseSymbol, false
	}
	narrowedSym, isNarrowed, _ := lookupBinding(chain, baseSymbol)
	if isNarrowed {
		return narrowedSym, true
	}
	return baseSymbol, true
}

func varRefExpInner(expr ast.BLangActionOrExpression) (model.SymbolRef, bool) {
	if expr == nil {
		return model.SymbolRef{}, false
	}
	switch expr := expr.(type) {
	case *ast.BLangSimpleVarRef:
		return expr.Symbol(), true
	case *ast.BLangLocalVarRef:
		return expr.Symbol(), true
	case *ast.BLangConstRef:
		return expr.Symbol(), true
	default:
		return model.SymbolRef{}, false
	}
}
