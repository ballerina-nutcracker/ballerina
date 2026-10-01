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

package desugar

import (
	"maps"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common/constants"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// desugarWorkerRegion lowers the worker region and the trailing statements of
// a block function body with named workers to existing language constructs.
// The emitted sequence is:
//
//	handle $latch = createLatch();
//	future<T> $default; future<T1> $w1; ... future<Tn> $wn;
//	var $cd = function () returns T { waitOnLatch($latch); <trailing stmts> };
//	var $c1 = function () returns T1 { waitOnLatch($latch); <body 1> }; ...
//	$default = start $cd(); $w1 = start $c1(); ... $wn = start $cn();
//	openLatch($latch);
//	return wait $default;
//
// The only state shared between workers is the future slot table: a worker body
// may reference a sibling declared after it, so every slot is allocated before
// any body is lowered. Every closure is constructed before the first start, and
// the latch is opened only after every slot has been assigned.
func desugarWorkerRegion(cx *functionContext, body *ast.BLangBlockFunctionBody) []ast.StatementNode {
	workers := body.Workers
	pos := workers[0].GetPosition()

	latchDef, latchRef := assignToLocal(cx, createLatchInvocation(cx, pos), pos)
	stmts := make([]ast.StatementNode, 0, 3*len(workers)+6)
	stmts = append(stmts, latchDef)

	cx.workerFutureSlots = maps.Clone(cx.workerFutureSlots)
	if cx.workerFutureSlots == nil {
		cx.workerFutureSlots = make(map[model.SymbolRef]*ast.BLangVarRef, len(workers)+1)
	}
	defaultDef, defaultSlot := declareWorkerFutureSlot(cx, body.DefaultWorker, pos)
	stmts = append(stmts, defaultDef)
	cx.workerFutureSlots[body.DefaultWorker] = defaultSlot
	slotRefs := make([]*ast.BLangVarRef, len(workers))
	for i, worker := range workers {
		def, ref := declareWorkerFutureSlot(cx, worker.Symbol(), pos)
		stmts = append(stmts, def)
		slotRefs[i] = ref
		cx.workerFutureSlots[worker.Symbol()] = ref
	}

	defaultClosureDef, defaultStart := desugarDefaultWorker(cx, body, latchRef, defaultSlot)
	stmts = append(stmts, defaultClosureDef)
	starts := make([]ast.StatementNode, 0, len(workers)+1)
	starts = append(starts, defaultStart)
	for i, worker := range workers {
		closureDef, start := desugarNamedWorker(cx, worker, latchRef, slotRefs[i])
		stmts = append(stmts, closureDef)
		starts = append(starts, start)
	}

	stmts = append(stmts, starts...)
	stmts = append(stmts, expressionStatement(openLatchInvocation(cx, latchRef, pos), pos))
	return append(stmts, returnDefaultWorkerResult(cx, body, defaultSlot))
}

// desugarNamedWorker lowers one worker into the statement that binds its
// closure and the statement that starts it and publishes the resulting future
// into slot. The two are returned separately because every closure must be
// bound before the first worker is started.
func desugarNamedWorker(
	cx *functionContext,
	worker *ast.BLangNamedWorkerDeclaration,
	latch *ast.BLangVarRef,
	slot *ast.BLangVarRef,
) (closureDef ast.StatementNode, start ast.StatementNode) {
	closure := createWorkerClosure(cx, worker, latch)
	closureDef, closureRef := assignToLocal(cx, closure, worker.GetPosition())
	return closureDef, createWorkerStart(cx, worker.Symbol(), closureRef, slot, worker.GetPosition())
}

// desugarDefaultWorker lowers the trailing statements of a body with named
// workers into the default worker's closure, and returns the statement that
// binds it and the one that starts it.
func desugarDefaultWorker(
	cx *functionContext,
	body *ast.BLangBlockFunctionBody,
	latch *ast.BLangVarRef,
	slot *ast.BLangVarRef,
) (closureDef ast.StatementNode, start ast.StatementNode) {
	pos := defaultWorkerPosition(body)
	returnTy := workerReturnType(cx, body.DefaultWorker)
	isolated := cx.isIsolated

	latchWait := expressionStatement(waitOnLatchInvocation(cx, latch, pos), pos)
	// The statements are lowered in this context rather than as a nested
	// function; BIR gen resolves the variables they capture lexically.
	closureBody := &ast.BLangBlockFunctionBody{
		Stmts: append([]ast.StatementNode{latchWait}, walkStatementList(cx, body.Stmts)...),
	}
	closureBody.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(closureBody, pos)

	name := workerClosureName(cx, constants.DefaultWorkerName)
	closureTy := workerClosureType(cx, returnTy, isolated)
	symRef := addWorkerClosureSymbol(cx, name, pos, closureTy, returnTy, isolated)
	var returnTypeDescriptor ast.BType
	if cx.returnTypeDescriptor != nil {
		returnTypeDescriptor = cx.returnTypeDescriptor.TypeDescriptor
	}
	fn := newWorkerClosureFunction(name, symRef, closureBody, returnTypeDescriptor, cx.currentScope(), isolated, pos)

	lambda := &ast.BLangLambdaFunction{Function: fn}
	lambda.SetDeterminedType(closureTy)
	setPositionIfMissing(lambda, pos)
	closureDef, closureRef := assignToLocal(cx, lambda, pos)
	// Starting over the trailing statements keeps the stack trace of a panic
	// in them unchanged.
	return closureDef, createWorkerStart(cx, body.DefaultWorker, closureRef, slot, pos)
}

// defaultWorkerPosition spans the trailing statements of a body, or is the
// body's closing brace when there are none.
func defaultWorkerPosition(body *ast.BLangBlockFunctionBody) diagnostics.Location {
	if len(body.Stmts) == 0 {
		return diagnostics.EndLocation(body.GetPosition())
	}
	first := body.Stmts[0].GetPosition()
	last := body.Stmts[len(body.Stmts)-1].GetPosition()
	return diagnostics.SpanLocations(first, last)
}

// returnDefaultWorkerResult makes the function return what its default worker
// terminated with.
func returnDefaultWorkerResult(
	cx *functionContext,
	body *ast.BLangBlockFunctionBody,
	slot *ast.BLangVarRef,
) ast.StatementNode {
	pos := diagnostics.EndLocation(body.GetPosition())
	slotRef := copyVarRef(slot)
	slotRef.SetPosition(pos)
	wait := &ast.BLangSingleWaitAction{FutureExpr: slotRef}
	wait.SetDeterminedType(workerReturnType(cx, body.DefaultWorker))
	setPositionIfMissing(wait, pos)
	ret := &ast.BLangReturn{Expr: wait}
	ret.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(ret, pos)
	return ret
}

// declareWorkerFutureSlot declares the hidden future slot of a worker without
// initialising it. The slot is assigned by the generated start action.
func declareWorkerFutureSlot(
	cx *functionContext,
	worker model.SymbolRef,
	pos diagnostics.Location,
) (ast.StatementNode, *ast.BLangVarRef) {
	futureTy := cx.symbolType(worker)
	name, symRef := cx.addDesugardSymbol(futureTy, model.SymbolKindVariable, pos)

	slotVar := &ast.BLangVariable{Name: newIdentifier(name)}
	slotVar.Name.SetDeterminedType(semtypes.Never)
	slotVar.SetDeterminedType(semtypes.Never)
	slotVar.SetSymbol(symRef)
	varDef := &ast.BLangVariableDef{}
	varDef.SetVariable(slotVar)
	varDef.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(varDef, pos)

	varRef := &ast.BLangVarRef{VariableName: slotVar.Name}
	varRef.SetSymbol(symRef)
	varRef.SetDeterminedType(futureTy)
	setPositionIfMissing(varRef, pos)
	return varDef, varRef
}

// createWorkerClosure turns a source worker into an anonymous function whose
// body awaits the startup latch before executing any source statement, so it
// cannot observe an unassigned sibling slot.
func createWorkerClosure(
	cx *functionContext,
	worker *ast.BLangNamedWorkerDeclaration,
	latch *ast.BLangVarRef,
) ast.BLangExpression {
	pos := worker.GetPosition()
	returnTy := workerReturnType(cx, worker.Symbol())
	isolated := cx.isIsolated

	workerBody := worker.Body
	latchWait := expressionStatement(waitOnLatchInvocation(cx, latch, pos), pos)
	workerBody.Stmts = append([]ast.StatementNode{latchWait}, workerBody.Stmts...)

	closureTy := workerClosureType(cx, returnTy, isolated)
	name := workerClosureName(cx, worker.Name)
	symRef := addWorkerClosureSymbol(cx, name, pos, closureTy, returnTy, isolated)
	fn := newWorkerClosureFunction(name, symRef, workerBody, worker.ReturnType.TypeDescriptor, worker.Scope(), isolated, pos)

	lambda := &ast.BLangLambdaFunction{Function: desugarNestedFunction(cx, fn)}
	lambda.SetDeterminedType(closureTy)
	setPositionIfMissing(lambda, pos)
	return lambda
}

func newWorkerClosureFunction(
	name string,
	symRef model.SymbolRef,
	body *ast.BLangBlockFunctionBody,
	returnTypeDescriptor ast.BType,
	scope model.Scope,
	isolated bool,
	pos diagnostics.Location,
) *ast.BLangFunction {
	flags := model.FlagLambda | model.FlagAnonymous
	if isolated {
		flags |= model.FlagIsolated
	}
	nameNode := newIdentifier(name)
	nameNode.SetDeterminedType(semtypes.Never)
	fn := ast.NewBLangFunction(ast.InvokableData{
		Position:             pos,
		Name:                 nameNode,
		ReturnTypeDescriptor: returnTypeDescriptor,
		Body:                 body,
		Flags:                flags,
	})
	fn.SetSymbol(symRef)
	fn.SetScope(scope)
	fn.SetDeterminedType(semtypes.Never)
	return fn
}

// createWorkerStart starts a worker closure and publishes the resulting future
// into the worker's hidden slot. Scheduling metadata is set explicitly here
// because semantic analysis, which computes it for source start actions, has
// already run.
func createWorkerStart(
	cx *functionContext,
	worker model.SymbolRef,
	closure *ast.BLangVarRef,
	slot *ast.BLangVarRef,
	pos diagnostics.Location,
) ast.StatementNode {
	invocation := &ast.BLangInvocation{}
	invocation.Name = closure.VariableName
	invocation.SetSymbol(closure.Symbol())
	invocation.SetDeterminedType(workerReturnType(cx, worker))
	setPositionIfMissing(invocation, pos)

	start := &ast.BLangStartAction{Call: invocation, IsIsolated: cx.isIsolated}
	start.SetDeterminedType(cx.symbolType(worker))
	setPositionIfMissing(start, pos)

	assignment := &ast.BLangAssignment{VarRef: copyVarRef(slot), Expr: start}
	assignment.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(assignment, pos)
	return assignment
}

// workerFutureSlotRef replaces a value reference to a worker with a load of the
// worker's hidden future slot. It applies outside waits as well as within them;
// wait nodes keep their already-resolved source type.
func workerFutureSlotRef(cx *functionContext, reference *ast.BLangVarRef) ast.BLangExpression {
	slot, ok := cx.workerFutureSlots[reference.Symbol()]
	if !ok {
		// Unreachable: every worker slot is allocated before any worker body or
		// trailing default-worker statement is lowered, and the reference-count
		// rule keeps a worker name out of scopes the slot table does not cover.
		cx.internalError("no future slot for worker reference", reference.GetPosition())
		return reference
	}
	rewritten := copyVarRef(slot)
	rewritten.SetPosition(reference.GetPosition())
	return rewritten
}

func workerReturnType(cx *functionContext, worker model.SymbolRef) semtypes.SemType {
	return semtypes.FutureEventualType(cx.typeCtx(), cx.symbolType(worker))
}

func workerClosureType(cx *functionContext, returnTy semtypes.SemType, isolated bool) semtypes.SemType {
	env := cx.typeEnv()
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.Define(env, nil, semtypes.ListRest(semtypes.Never),
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	fnDefn := semtypes.NewFunctionDefinition()
	return fnDefn.Define(env, paramListTy, returnTy,
		semtypes.FunctionQualifiersFrom(env, isolated, false))
}

// workerClosureName is the name of the generated function a worker's body is
// lowered into. Generated functions share one package-wide lookup-key
// namespace, so the worker's source name is qualified with its owner; a worker
// name is unique within its owner, and lambdas are already uniquely named. The
// default worker's name can't clash with a source name.
func workerClosureName(cx *functionContext, workerName string) string {
	return constants.WorkerClosurePrefix + cx.uniqueNamePrefix + ":" + workerName
}

// addWorkerClosureSymbol declares the generated closure for a worker.
func addWorkerClosureSymbol(
	cx *functionContext,
	name string,
	pos diagnostics.Location,
	closureTy semtypes.SemType,
	returnTy semtypes.SemType,
	isolated bool,
) model.SymbolRef {
	signature := model.TypedFunctionSignature{ReturnType: returnTy, RestParamType: semtypes.Never}
	if isolated {
		signature.Flags |= model.FuncSymbolFlagIsolated
	}
	symbol := model.NewFunctionSymbol(name, signature, false, pos)
	symbol.SetType(closureTy)
	cx.currentScope().AddSymbol(name, symbol)
	ref, _ := cx.currentScope().GetSymbol(name)
	return ref
}

func createLatchInvocation(cx *functionContext, pos diagnostics.Location) ast.BLangExpression {
	return createLangInternalInvocation(cx, "createLatch", semtypes.Handle, nil, pos)
}

func waitOnLatchInvocation(
	cx *functionContext,
	latch *ast.BLangVarRef,
	pos diagnostics.Location,
) ast.BLangExpression {
	return createLangInternalInvocation(cx, "waitOnLatch", semtypes.Nil,
		[]ast.BLangExpression{copyVarRef(latch)}, pos)
}

func openLatchInvocation(
	cx *functionContext,
	latch *ast.BLangVarRef,
	pos diagnostics.Location,
) ast.BLangExpression {
	return createLangInternalInvocation(cx, "openLatch", semtypes.Nil,
		[]ast.BLangExpression{copyVarRef(latch)}, pos)
}

func copyVarRef(ref *ast.BLangVarRef) *ast.BLangVarRef {
	copied := &ast.BLangVarRef{VariableName: ref.VariableName}
	copied.SetSymbol(ref.Symbol())
	copied.SetDeterminedType(ref.GetDeterminedType())
	copied.SetPosition(ref.GetPosition())
	return copied
}

func expressionStatement(expr ast.BLangExpression, pos diagnostics.Location) ast.StatementNode {
	stmt := &ast.BLangExpressionStmt{Expr: expr}
	stmt.SetDeterminedType(semtypes.Never)
	setPositionIfMissing(stmt, pos)
	return stmt
}
