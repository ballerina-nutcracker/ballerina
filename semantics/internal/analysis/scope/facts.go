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

package scope

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

const (
	factIsolated uint8 = 1 << iota
	factIsolatedTransferred
	factOperandsIsolated
	factCalleeIsolated
	factAll = factIsolated | factIsolatedTransferred | factOperandsIsolated | factCalleeIsolated
)

// Facts are properties of an expression synthesized bottom-up while it is
// validated. They are synthesized only under a context that demands them
// (NewLock, NewIsolatedExpr, NewStartCall, NewIsolatedVarInit); elsewhere
// they carry no meaning and must not be read.
type Facts struct{ bits uint8 }

// NoFacts holds no property.
var NoFacts Facts

// AllFacts holds every property; it is the identity of Meet.
var AllFacts = Facts{bits: factAll}

// FactsOfType holds Isolated and IsolatedTransferred iff e's determined type
// is a subtype of Isolated, or NoFacts where no consumer demands facts; e is
// an expression or an action.
func FactsOfType(cx Context, e ast.BLangActionOrExpression) Facts {
	if !cx.factsNeeded() {
		return NoFacts
	}
	if isolatedType(cx, e) {
		return Facts{bits: factIsolated | factIsolatedTransferred}
	}
	return NoFacts
}

// IsolatedWhateverOperands reports whether e is an isolated expression
// whatever its subexpressions: its type is a subtype of Isolated, or it is a
// call to lang.value:clone.
func IsolatedWhateverOperands(cx Context, e ast.BLangActionOrExpression) bool {
	// Spec §6.15: lang.value:clone calls are always isolated, regardless of their arguments.
	// https://ballerina.io/spec/lang/master/#function_call
	if inv, ok := e.(*ast.BLangInvocation); ok && !ast.IsStreamOperation(inv) && isValueCloneCall(cx, inv) {
		return true
	}
	return isolatedType(cx, e)
}

func isolatedType(cx Context, e ast.BLangActionOrExpression) bool {
	tyCtx := cx.TyCtx()
	return semtypes.IsSubtype(tyCtx, e.Type(), semtypes.CreateIsolated(tyCtx))
}

func isValueCloneCall(cx Context, inv *ast.BLangInvocation) bool {
	ctx := cx.CompilerCtx()
	mono, ok := ctx.GetSymbol(inv.ResolvedSymbol()).(model.MonomorphicFunctionSymbol)
	if !ok {
		return false
	}
	poly := mono.PolymorphicSymbol()
	fn, ok := ctx.GetSymbol(poly).(*model.OpaqueFunctionSymbol)
	if !ok || fn.OpaqueID() != model.OpaqueFnValueClone {
		return false
	}
	pkg := ctx.SymbolPackage(poly)
	return pkg.Organization == "ballerina" && pkg.Package == "lang.value"
}

// Meet holds the properties both a and b hold.
func Meet(a, b Facts) Facts { return Facts{bits: a.bits & b.bits} }

// Either holds the properties a or b holds.
func Either(a, b Facts) Facts { return Facts{bits: a.bits | b.bits} }

// CallFacts are the facts of call given the fold of its operands, or NoFacts
// where no consumer demands facts.
func CallFacts(cx Context, call ast.Invocable, operands Facts) Facts {
	if !cx.factsNeeded() {
		return NoFacts
	}
	if inv, ok := call.(*ast.BLangInvocation); ok {
		return invocationFacts(cx, inv, operands)
	}
	return actionFacts(cx, call, operands)
}

// StartFacts are the facts of a start action given the facts of its call: it
// is isolated when its callee and operands are.
func StartFacts(call Facts) Facts {
	if call.calleeIsolated() && call.operandsIsolated() {
		return AllFacts
	}
	return NoFacts
}

// invocationFacts holds Isolated and IsolatedTransferred iff inv is isolated
// whatever its operands, plus whether its operands (the fold of its receiver
// and arguments) are isolated and its callee is an isolated function.
func invocationFacts(cx Context, inv *ast.BLangInvocation, operands Facts) Facts {
	var f Facts
	if IsolatedWhateverOperands(cx, inv) {
		f.bits |= factIsolated | factIsolatedTransferred
	}
	if operands.Isolated() {
		f.bits |= factOperandsIsolated
	}
	if ast.IsStreamOperation(inv) || calleeIsolated(cx, inv.ResolvedSymbol()) {
		f.bits |= factCalleeIsolated
	}
	return f
}

// actionFacts are the facts of a remote method call or client resource
// access: the type rule as FactsOfType, or else isolated iff its callee is an
// isolated function and transferred iff in addition every operand is
// isolated.
func actionFacts(cx Context, call ast.Invocable, operands Facts) Facts {
	var f Facts
	if operands.Isolated() {
		f.bits |= factOperandsIsolated
	}
	if calleeIsolated(cx, call.ResolvedSymbol()) {
		f.bits |= factCalleeIsolated | factIsolated
		if operands.Isolated() {
			f.bits |= factIsolatedTransferred
		}
	}
	return Either(FactsOfType(cx, call), f)
}

func (f Facts) Isolated() bool            { return f.bits&factIsolated != 0 }
func (f Facts) IsolatedTransferred() bool { return f.bits&factIsolatedTransferred != 0 }
func (f Facts) operandsIsolated() bool    { return f.bits&factOperandsIsolated != 0 }
func (f Facts) calleeIsolated() bool      { return f.bits&factCalleeIsolated != 0 }

func calleeIsolated(cx Context, callee model.SymbolRef) bool {
	tyCtx := cx.TyCtx()
	return semtypes.IsSubtype(tyCtx, cx.CompilerCtx().SymbolType(callee), semtypes.CreateIsolatedFn(tyCtx))
}
