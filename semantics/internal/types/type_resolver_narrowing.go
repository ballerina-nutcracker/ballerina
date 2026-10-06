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

package types

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
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
	// assignmentPositions are the assignments that created this unnarrowing entry, merged
	// from every side; loop arms report them for variables narrowed outside the loop.
	assignmentPositions []diagnostics.Location
	// defaultType is used for unreachable branches (e.g. false branch of constant true)
	// see https://github.com/ballerina-platform/ballerina-spec/issues/1029
	defaultType semtypes.SemType
	// captures is set on an entry that records an effective capture rather than a
	// variable binding. The bindings in the group cannot acquire a conditional
	// refinement while this entry applies.
	captures model.CaptureGroupRef
}

// isCaptureEntry reports whether this entry records a capture rather than a
// binding. Such an entry has no symbol, so binding walks skip it.
func (b *binding) isCaptureEntry() bool {
	return b.captures != 0
}

// addCaptureGroup extends chain with group's capture restrictions. An empty
// group has no effect. Neither input is mutated.
func addCaptureGroup(chain *binding, group model.CaptureGroupRef) *binding {
	if group == 0 {
		return chain
	}
	return &binding{prev: chain, captures: group}
}

func (b *binding) hasFlag(flag bindingFlags) bool {
	return b.flags&flag != 0
}

func (b *binding) isUnnarrowing() bool {
	return b.ref == b.narrowedSymbol
}

type expressionEffect struct {
	ifTrue  *binding
	ifFalse *binding
}

type expressionResult struct {
	ty                semtypes.SemType
	effect            expressionEffect
	functionSignature model.FunctionSignatureRef
	symbol            model.SymbolRef    // resolved callee of an invocation or action, narrowed symbol of a var/const ref; empty otherwise
	lax               bool               // field access resolved as lax access
	members           []expressionResult // member results of a list or mapping constructor, in source order
	spreadMembers     []bool             // spread classification of a list constructor's members
}

type statementEffect struct {
	binding *binding
	// nonCompletion indicates the statement is return/panic etc which spec treats narrowed type as never
	nonCompletion bool
}

// canNarrow reports whether a conditional refinement of ref applies at this
// point. Constants, final bindings and parameters cannot be reassigned, so they
// stay eligible. A mutable module binding is never eligible, because any call can
// reassign it. A mutable local stops being eligible once a capture entry that
// contains it is on the chain. A narrowed copy inherits the flags of the symbol
// it copies, so eligibility is read from the original declaration.
func canNarrow(t typeResolver, chain *binding, ref model.SymbolRef) bool {
	if isStableDeclaration(t, ref) {
		return true
	}
	ref = t.unnarrowedSymbol(ref)
	cx := t.compilerContext()
	if metadata, _ := cx.ValueSymbolMetadata(ref); metadata.TopLevel {
		return false
	}
	for c := chain; c != nil; c = c.prev {
		if c.isCaptureEntry() && cx.CaptureGroupContains(c.captures, ref) {
			return false
		}
	}
	return true
}

// isStableDeclaration reports whether ref names a declaration nothing can
// reassign, so neither a capture nor the module policy can invalidate a
// refinement of it.
func isStableDeclaration(t typeResolver, ref model.SymbolRef) bool {
	metadata, isValue := t.compilerContext().ValueSymbolMetadata(t.unnarrowedSymbol(ref))
	if !isValue {
		// A non-value symbol, such as a module function named in
		// `foo is function () returns int`, is a constant, so capture semantics
		// can be safely ignored for it.
		return true
	}
	return metadata.Const || metadata.Final || metadata.Parameter
}

// lookupBinding returns the effective symbol for a base symbol at the current
// point, and whether that symbol is a substitution for the base one. A refinement
// the eligibility policy rejects is not reported: the base symbol is effective
// instead. Unconditional query aggregation is not a conditional refinement and is
// never suppressed this way.
func lookupBinding(t typeResolver, chain *binding, ref model.SymbolRef) (model.SymbolRef, bool) {
	effective, isNarrowed := lookupBindingInner(t, chain, ref, false)
	if !isNarrowed || lookupQueryAggregatedBinding(chain, ref) {
		return effective, isNarrowed
	}
	if !canNarrow(t, chain, ref) {
		return t.unnarrowedSymbol(ref), false
	}
	return effective, isNarrowed
}

// lookupQueryAggregatedBinding reports unconditional query aggregation, which is
// not a conditional refinement, so the eligibility policy never applies to it and
// it needs no resolver.
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
	if chain.isCaptureEntry() {
		return lookupQueryAggregatedBindingInner(chain.prev, ref, crossedBoundary)
	}
	if chain.ref == ref || chain.narrowedSymbol == ref {
		return !crossedBoundary && !chain.isUnnarrowing() && chain.hasFlag(bindingFlagQueryAggregated)
	}
	return lookupQueryAggregatedBindingInner(chain.prev, ref, crossedBoundary)
}

func lookupBindingInner(t typeResolver, chain *binding, ref model.SymbolRef, crossedBoundary bool) (model.SymbolRef, bool) {
	if chain == nil {
		return ref, false
	}
	if chain.hasFlag(bindingFlagFunctionBoundary) {
		return lookupBindingInner(t, chain.prev, ref, true)
	}
	if chain.isCaptureEntry() {
		if !isStableDeclaration(t, ref) && t.compilerContext().CaptureGroupContains(chain.captures, t.unnarrowedSymbol(ref)) {
			// A capture between here and the refinement invalidates it.
			return t.unnarrowedSymbol(ref), false
		}
		return lookupBindingInner(t, chain.prev, ref, crossedBoundary)
	}
	if chain.ref == ref {
		isNarrowed := !chain.isUnnarrowing()
		if crossedBoundary && isNarrowed && chain.hasFlag(bindingFlagQueryAggregated) {
			// Aggregation keeps its function-boundary restriction; ordinary
			// refinements are governed by eligibility instead.
			return ref, false
		}
		return chain.narrowedSymbol, isNarrowed
	}
	return lookupBindingInner(t, chain.prev, ref, crossedBoundary)
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
	_, isNarrowed := lookupBinding(t, chain, symbol)
	if !isNarrowed {
		return statementEffect{chain, false}
	}
	var positions []diagnostics.Location
	if !diagnostics.IsLocationEmpty(pos) {
		positions = []diagnostics.Location{pos}
	}
	chain = &binding{
		ref:                 symbol,
		narrowedSymbol:      symbol,
		prev:                chain,
		assignmentPositions: positions,
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
			if c.hasFlag(bindingFlagFunctionBoundary) || c.isCaptureEntry() {
				continue
			}
			if seen[c.ref] {
				continue
			}
			seen[c.ref] = true
			if len(c.assignmentPositions) == 0 {
				continue
			}
			if _, isNarrowed := lookupBinding(t, loopEntry, c.ref); !isNarrowed {
				continue
			}
			for _, pos := range c.assignmentPositions {
				t.semanticError("cannot assign to a variable narrowed outside the enclosing loop", pos)
			}
		}
	}
}

// chainDepth counts the entries above a chain's nil tail.
func chainDepth(c *binding) int {
	depth := 0
	for ; c != nil; c = c.prev {
		depth++
	}
	return depth
}

// commonAncestor returns the deepest chain the two arguments share by pointer
// identity. Every pair of chains merged by the resolver descends from the
// expression or statement's incoming chain, so this is that shared history.
// Chains with no shared entry meet at their nil tail, so the result is nil.
func commonAncestor(c1, c2 *binding) *binding {
	d1, d2 := chainDepth(c1), chainDepth(c2)
	for d1 > d2 {
		c1 = c1.prev
		d1--
	}
	for d2 > d1 {
		c2 = c2.prev
		d2--
	}
	for c1 != c2 {
		c1 = c1.prev
		c2 = c2.prev
	}
	return c1
}

// prefixBinding is what one side of a merge contributes for a single symbol.
type prefixBinding struct {
	ty    semtypes.SemType
	flags bindingFlags
	// assignmentPositions is the provenance of an assignment-introduced entry,
	// which the loop diagnostic reads and a merge must therefore not drop.
	assignmentPositions []diagnostics.Location
}

// accumPrefix collects the refinements and capture groups chain introduces above
// ancestor, and returns the unreachable-branch default type if it has one.
//
// A merged prefix never contains a function boundary marker: a lambda body's
// chain stays inside resolveLambdaFunctionExpr, resolveInferredLambdaFunctionExpr
// and resolveDefaultExpression, each of which returns the chain the construction
// leaves behind rather than the body's own.
func accumPrefix(t typeResolver, chain, ancestor *binding, accum map[model.SymbolRef]prefixBinding, groups *[]model.CaptureGroupRef) semtypes.SemType {
	var accumDefault semtypes.SemType
	for c := chain; c != ancestor; c = c.prev {
		if c.isCaptureEntry() {
			*groups = append(*groups, c.captures)
			continue
		}
		if !semtypes.IsZero(c.defaultType) {
			if semtypes.IsZero(accumDefault) {
				accumDefault = c.defaultType
			}
			continue
		}
		if _, seen := accum[c.ref]; seen {
			continue
		}
		accum[c.ref] = prefixBinding{
			ty:                  t.symbolType(c.narrowedSymbol),
			flags:               c.flags,
			assignmentPositions: c.assignmentPositions,
		}
	}
	return accumDefault
}

// ancestorBinding is the contribution of a side that introduces no refinement of
// its own: the effective binding already present in the shared history.
func ancestorBinding(t typeResolver, ancestor *binding, ref model.SymbolRef) prefixBinding {
	effective, _ := lookupBinding(t, ancestor, ref)
	var flags bindingFlags
	if lookupQueryAggregatedBinding(ancestor, ref) {
		flags = bindingFlagQueryAggregated
	}
	return prefixBinding{ty: t.symbolType(effective), flags: flags}
}

// mergeChains combines two chains with mergeOp. The merged refinements are consed
// onto the chains' shared history, so function boundaries, query aggregation and
// assignment provenance below the divergence point survive the merge.
func mergeChains(t typeResolver, c1 *binding, c2 *binding, mergeOp func(semtypes.SemType, semtypes.SemType) semtypes.SemType) *binding {
	if c1 == c2 {
		return c1
	}
	ancestor := commonAncestor(c1, c2)
	m1 := make(map[model.SymbolRef]prefixBinding)
	var g1, g2 []model.CaptureGroupRef
	d1 := accumPrefix(t, c1, ancestor, m1, &g1)
	m2 := make(map[model.SymbolRef]prefixBinding)
	d2 := accumPrefix(t, c2, ancestor, m2, &g2)
	type bindingPair struct{ b1, b2 prefixBinding }
	pairs := make(map[model.SymbolRef]bindingPair, len(m1)+len(m2))
	for ref, b1 := range m1 {
		b2, ok := m2[ref]
		if !ok {
			b2 = otherSideBinding(t, ancestor, ref, d2)
		}
		pairs[ref] = bindingPair{b1, b2}
	}
	for ref, b2 := range m2 {
		if _, ok := m1[ref]; ok {
			continue
		}
		pairs[ref] = bindingPair{otherSideBinding(t, ancestor, ref, d1), b2}
	}
	// An unreachable-branch default applies to every refinement the shared history
	// carries, so those have to be materialized before the merge drops the default.
	if !semtypes.IsZero(d1) || !semtypes.IsZero(d2) {
		for _, ref := range ancestorRefs(ancestor) {
			if _, ok := pairs[ref]; ok {
				continue
			}
			pairs[ref] = bindingPair{otherSideBinding(t, ancestor, ref, d1), otherSideBinding(t, ancestor, ref, d2)}
		}
	}
	// A capture on a path that reaches the join is still in effect after it. A path
	// marked unreachable is not such a path, so its captures stay behind.
	result := ancestor
	if semtypes.IsZero(d1) {
		for _, group := range g1 {
			result = addCaptureGroup(result, group)
		}
	}
	if semtypes.IsZero(d2) {
		for _, group := range g2 {
			result = addCaptureGroup(result, group)
		}
	}
	for ref, pair := range pairs {
		ty := mergeOp(pair.b1.ty, pair.b2.ty)
		// Only a refinement both sides agree is an unconditional aggregation stays
		// one. Boundary markers are never recreated by a merge.
		flags := pair.b1.flags & pair.b2.flags & bindingFlagQueryAggregated
		inherited := ancestorBinding(t, ancestor, ref)
		if flags == inherited.flags && semtypes.IsSameType(t.typeContext(), ty, inherited.ty) {
			// The merge changed nothing for this symbol, so the shared history keeps
			// its entry along with its aggregation flag and assignment provenance.
			continue
		}
		result = &binding{
			ref:                 ref,
			narrowedSymbol:      narrowSymbol(t, ref, ty),
			prev:                result,
			flags:               flags,
			assignmentPositions: mergedAssignmentPositions(pair.b1, pair.b2),
		}
	}
	return result
}

// ancestorRefs lists the symbols refined by the shared history, stopping at a
// function boundary so that an inherited refinement is not lifted across it.
func ancestorRefs(ancestor *binding) []model.SymbolRef {
	var refs []model.SymbolRef
	seen := make(map[model.SymbolRef]struct{})
	for c := ancestor; c != nil; c = c.prev {
		if c.hasFlag(bindingFlagFunctionBoundary) {
			break
		}
		if c.isCaptureEntry() || !semtypes.IsZero(c.defaultType) {
			continue
		}
		if _, ok := seen[c.ref]; ok {
			continue
		}
		seen[c.ref] = struct{}{}
		refs = append(refs, c.ref)
	}
	return refs
}

// mergedAssignmentPositions keeps the assignment positions of both sides of a
// merge, so the loop diagnostic reports every assignment that reaches the join.
func mergedAssignmentPositions(b1, b2 prefixBinding) []diagnostics.Location {
	if len(b1.assignmentPositions) == 0 {
		return b2.assignmentPositions
	}
	if len(b2.assignmentPositions) == 0 {
		return b1.assignmentPositions
	}
	positions := make([]diagnostics.Location, 0, len(b1.assignmentPositions)+len(b2.assignmentPositions))
	positions = append(positions, b1.assignmentPositions...)
	return append(positions, b2.assignmentPositions...)
}

// otherSideBinding is the contribution of the side that does not refine ref: the
// unreachable-branch default when that side has one, else the shared history.
func otherSideBinding(t typeResolver, ancestor *binding, ref model.SymbolRef, sideDefault semtypes.SemType) prefixBinding {
	inherited := ancestorBinding(t, ancestor, ref)
	if !semtypes.IsZero(sideDefault) {
		return prefixBinding{ty: sideDefault, flags: inherited.flags}
	}
	return inherited
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

// sequentialChain is the chain that continues after an expression evaluated for
// its value rather than tested as a condition. Both outcomes of that expression
// are possible, so they join the same way a variable initializer joins them.
func sequentialChain(t typeResolver, effect expressionEffect) *binding {
	return mergeChains(t, effect.ifTrue, effect.ifFalse, semtypes.Union)
}

// singletonResultEffect represents the outcome a constant-valued condition cannot
// take. The impossible outcome keeps the expression's incoming chain: no
// reachable use reads it, and importing the produced effects there would make an
// already reported unreachable region report again.
func singletonResultEffect(chain *binding, ty semtypes.SemType) (expressionEffect, bool) {
	if semtypes.IsZero(ty) {
		return expressionEffect{}, false
	}
	if isSingletonBool(ty, true) {
		return expressionEffect{ifTrue: chain, ifFalse: &binding{defaultType: semtypes.Never, prev: chain}}, true
	} else if isSingletonBool(ty, false) {
		return expressionEffect{ifTrue: &binding{defaultType: semtypes.Never, prev: chain}, ifFalse: chain}, true
	}
	return expressionEffect{}, false
}

func defaultExpressionEffect(chain *binding) expressionEffect {
	return expressionEffect{ifTrue: chain, ifFalse: chain}
}

func defaultStmtEffect(chain *binding) statementEffect {
	return statementEffect{binding: chain, nonCompletion: false}
}

func varRefExp(t typeResolver, chain *binding, expr ast.BLangActionOrExpression) (model.SymbolRef, bool) {
	baseSymbol, isVarRef := varRefExpInner(expr)
	if !isVarRef {
		return baseSymbol, false
	}
	narrowedSym, isNarrowed := lookupBinding(t, chain, baseSymbol)
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
	case *ast.BLangVarRef:
		return expr.Symbol(), true
	case *ast.BLangConstRef:
		return expr.Symbol(), true
	default:
		return model.SymbolRef{}, false
	}
}

// withSingletonEffect replaces the effect of a result whose type is a boolean singleton: the branch that cannot be
// taken narrows every symbol to never.
func withSingletonEffect(chain *binding, result expressionResult) expressionResult {
	singletonEffect, isSingleton := singletonResultEffect(chain, result.ty)
	if !isSingleton {
		return result
	}
	if isSingletonBool(result.ty, true) {
		singletonEffect.ifTrue = result.effect.ifTrue
	} else {
		singletonEffect.ifFalse = result.effect.ifFalse
	}
	result.effect = singletonEffect
	return result
}

// withRawSymbol records the committed callee of an already resolved invocation in result.
func withRawSymbol(result expressionResult, symbol model.Symbol) expressionResult {
	if callee, ok := symbol.(*model.SymbolRef); ok {
		result.symbol = *callee
	}
	return result
}
