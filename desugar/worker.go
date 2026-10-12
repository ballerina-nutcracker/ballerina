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

// desugarBodyWorkerRegion lowers the worker region of a block function body
// to existing language constructs. The emitted sequence is:
//
//	handle $latch = createLatch();
//	handle $m1 = createWorkerMessage(); ...      // one per send
//	future<T> $default; future<T1> $w1; ... future<Tn> $wn;
//	var $cd = function () returns T { waitOnLatch($latch); <default worker stmts> };
//	var $c1 = function () returns T1 { waitOnLatch($latch); <body 1> }; ...
//	$default = start $cd(); $w1 = start $c1(); ... $wn = start $cn();
//	openLatch($latch);
//	return wait $default;
func desugarBodyWorkerRegion(cx *functionContext, body *ast.BLangBlockFunctionBody) []ast.StatementNode {
	region := body.Region
	stmts, openLatch := lowerWorkerRegion(cx, region, region.Workers[0].GetPosition())
	stmts = append(stmts, openLatch)
	return append(stmts, returnDefaultWorkerResult(cx, region.DefaultWorker))
}

// lowerWorkerRegion returns the statements that start the region's workers
// and, separately, the statement that opens its latch, which must come last.
//
// The only state shared between workers is the future slot table: a worker body
// may reference a sibling declared after it, so every slot is allocated before
// any body is lowered. Every closure is constructed before the first start, and
// the latch is opened only after every slot has been assigned.
func lowerWorkerRegion(cx *functionContext, region *ast.WorkerRegion, pos diagnostics.Location) (stmts []ast.StatementNode, openLatch ast.StatementNode) {
	workers := region.Workers
	defaultWorker := region.DefaultWorker

	latchDef, latchRef := assignToLocal(cx, createLangInternalInvocation(cx, "createLatch", semtypes.Handle, nil, pos), pos)
	stmts = make([]ast.StatementNode, 0, 3*len(workers)+6)
	stmts = append(stmts, latchDef)

	// Nested contexts share these maps by reference, so the region's entries
	// go into a copy.
	cx.workerMessages = maps.Clone(cx.workerMessages)
	if cx.workerMessages == nil {
		cx.workerMessages = make(map[model.WorkerMessageRef]*ast.BLangVarRef)
	}
	stmts = append(stmts, declareWorkerMessages(cx, region, pos)...)

	cx.workerFutureSlots = maps.Clone(cx.workerFutureSlots)
	if cx.workerFutureSlots == nil {
		cx.workerFutureSlots = make(map[model.SymbolRef]*ast.BLangVarRef, len(workers)+1)
	}
	if defaultWorker != nil {
		stmts = append(stmts, declareWorkerFutureSlot(cx, defaultWorker.Symbol(), pos))
	}
	for _, worker := range workers {
		stmts = append(stmts, declareWorkerFutureSlot(cx, worker.Symbol(), pos))
	}

	starts := make([]ast.StatementNode, 0, len(workers)+1)
	if defaultWorker != nil {
		closureDefs, start := desugarDefaultWorker(cx, region.NamePrefix, defaultWorker, latchRef)
		stmts = append(stmts, closureDefs...)
		starts = append(starts, start)
	}
	for _, worker := range workers {
		closureDefs, start := desugarNamedWorker(cx, region.NamePrefix, worker, latchRef)
		stmts = append(stmts, closureDefs...)
		starts = append(starts, start)
	}

	stmts = append(stmts, starts...)
	open := createLangInternalInvocation(cx, "openLatch", semtypes.Nil,
		[]ast.BLangExpression{copyVarRef(latchRef)}, pos)
	return stmts, expressionStatement(open, pos)
}

// desugarNamedWorker lowers one worker into the statements that bind its
// closures and the statement that starts it and publishes the resulting future
// into its slot. They are returned separately because every closure must be
// bound before the first worker is started.
func desugarNamedWorker(
	cx *functionContext,
	namePrefix string,
	worker *ast.BLangNamedWorkerDeclaration,
	latch *ast.BLangVarRef,
) (closureDefs []ast.StatementNode, start ast.StatementNode) {
	closureDefs, closure := createWorkerClosure(cx, namePrefix, worker, latch)
	closureDef, closureRef := assignToLocal(cx, closure, worker.GetPosition())
	return append(closureDefs, closureDef), createWorkerStart(cx, worker.Symbol(), closureRef, worker.GetPosition())
}

// desugarDefaultWorker lowers the default worker into its closure, and returns
// the statements that bind its closures and the one that starts it. Unlike a
// named worker, its statements belong to the enclosing function's scope, so
// they are lowered in this context rather than as a nested function; BIR gen
// resolves the variables they capture lexically.
func desugarDefaultWorker(
	cx *functionContext,
	namePrefix string,
	worker *ast.BLangDefaultWorker,
	latch *ast.BLangVarRef,
) (closureDefs []ast.StatementNode, start ast.StatementNode) {
	pos := worker.GetPosition()
	returnTy := workerReturnType(cx, worker.Symbol())
	name := workerClosureName(cx, namePrefix, constants.DefaultWorkerName)
	var returnTypeDescriptor ast.BType
	if cx.returnTypeDescriptor != nil {
		returnTypeDescriptor = cx.returnTypeDescriptor.TypeDescriptor
	}
	sends := asyncSendsOf(worker.Stmts)
	stmts := walkStatementList(cx, worker.Stmts)
	if len(sends) > 0 {
		inner := newWorkerClosure(cx, name+workerBodySuffix, newBlockFunctionBody(stmts, pos), returnTy, returnTypeDescriptor, cx.currentScope(), pos)
		innerDef, innerRef := assignToLocal(cx, inner, pos)
		closureDefs = append(closureDefs, innerDef)
		stmts = withTerminationDelivery(cx, innerRef, returnTy, sends, diagnostics.EndLocation(pos))
	}
	latchWait := expressionStatement(waitOnLatchInvocation(cx, latch, pos), pos)
	closureBody := newBlockFunctionBody(append([]ast.StatementNode{latchWait}, stmts...), pos)
	lambda := newWorkerClosure(cx, name, closureBody, returnTy, returnTypeDescriptor, cx.currentScope(), pos)
	closureDef, closureRef := assignToLocal(cx, lambda, pos)
	// Starting over the default worker's statements keeps the stack trace of a
	// panic in them unchanged.
	return append(closureDefs, closureDef), createWorkerStart(cx, worker.Symbol(), closureRef, pos)
}

// returnDefaultWorkerResult makes the function return what its default worker
// terminated with.
func returnDefaultWorkerResult(cx *functionContext, worker *ast.BLangDefaultWorker) ast.StatementNode {
	pos := diagnostics.EndLocation(worker.GetPosition())
	slotRef := copyVarRef(cx.workerFutureSlots[worker.Symbol()])
	slotRef.SetPosition(pos)
	wait := &ast.BLangSingleWaitAction{FutureExpr: slotRef}
	ast.SetType(wait, workerReturnType(cx, worker.Symbol()))
	setPositionIfMissing(wait, pos)
	ret := &ast.BLangReturn{Expr: wait}
	setPositionIfMissing(ret, pos)
	return ret
}

// declareWorkerFutureSlot declares the hidden future slot of a worker without
// initialising it and records it in cx. The slot is assigned by the generated
// start action.
func declareWorkerFutureSlot(cx *functionContext, worker model.SymbolRef, pos diagnostics.Location) ast.StatementNode {
	futureTy := cx.symbolType(worker)
	name, symRef := cx.addDesugardSymbol(futureTy, model.SymbolKindVariable, pos)

	slotVar := &ast.BLangVariable{Name: newIdentifier(name)}
	slotVar.SetSymbol(symRef)
	varDef := &ast.BLangVariableDef{}
	varDef.SetVariable(slotVar)
	setPositionIfMissing(varDef, pos)

	varRef := &ast.BLangVarRef{VariableName: slotVar.Name}
	varRef.SetSymbol(symRef)
	ast.SetType(varRef, futureTy)
	setPositionIfMissing(varRef, pos)
	cx.workerFutureSlots[worker] = varRef
	return varDef
}

// createWorkerClosure turns a source worker into an anonymous function whose
// body awaits the startup latch before executing any source statement, so it
// cannot observe an unassigned sibling slot. A worker that waits for the
// delivery of its async messages runs its body in a second closure; the
// statements binding it are returned too.
func createWorkerClosure(
	cx *functionContext,
	namePrefix string,
	worker *ast.BLangNamedWorkerDeclaration,
	latch *ast.BLangVarRef,
) ([]ast.StatementNode, ast.BLangExpression) {
	pos := worker.GetPosition()
	returnTy := workerReturnType(cx, worker.Symbol())
	name := workerClosureName(cx, namePrefix, worker.Name)
	returnTypeDescriptor := worker.ReturnType.TypeDescriptor

	workerBody := worker.Body
	latchWait := expressionStatement(waitOnLatchInvocation(cx, latch, pos), pos)
	sends := asyncSendsOf(workerBody.Stmts)
	if len(sends) == 0 {
		workerBody.Stmts = append([]ast.StatementNode{latchWait}, workerBody.Stmts...)
		lambda := newWorkerClosure(cx, name, workerBody, returnTy, returnTypeDescriptor, worker.Scope(), pos)
		lambda.Function = desugarNestedFunction(cx, lambda.Function)
		return nil, lambda
	}
	inner := newWorkerClosure(cx, name+workerBodySuffix, workerBody, returnTy, returnTypeDescriptor, worker.Scope(), pos)
	inner.Function = desugarNestedFunction(cx, inner.Function)
	innerDef, innerRef := assignToLocal(cx, inner, pos)
	terminationPos := diagnostics.EndLocation(workerBody.GetPosition())
	stmts := append([]ast.StatementNode{latchWait}, withTerminationDelivery(cx, innerRef, returnTy, sends, terminationPos)...)
	lambda := newWorkerClosure(cx, name, newBlockFunctionBody(stmts, pos), returnTy, returnTypeDescriptor, cx.currentScope(), pos)
	return []ast.StatementNode{innerDef}, lambda
}

// workerBodySuffix names the hidden function a worker's body runs in when the
// worker waits for the delivery of its async messages before terminating.
const workerBodySuffix = "$body"

// newWorkerClosure builds a generated anonymous function of the current
// function from body as it is; the caller lowers it if needed.
func newWorkerClosure(
	cx *functionContext,
	name string,
	body *ast.BLangBlockFunctionBody,
	returnTy semtypes.SemType,
	returnTypeDescriptor ast.BType,
	scope model.Scope,
	pos diagnostics.Location,
) *ast.BLangLambdaFunction {
	isolated := cx.isIsolated
	closureTy := workerClosureType(cx, returnTy, isolated)
	symRef := addWorkerClosureSymbol(cx, name, pos, closureTy, returnTy, isolated)
	fn := newWorkerClosureFunction(name, symRef, body, returnTypeDescriptor, scope, isolated, pos)
	lambda := &ast.BLangLambdaFunction{Function: fn}
	ast.SetType(lambda, closureTy)
	setPositionIfMissing(lambda, pos)
	return lambda
}

func newBlockFunctionBody(stmts []ast.StatementNode, pos diagnostics.Location) *ast.BLangBlockFunctionBody {
	body := &ast.BLangBlockFunctionBody{Stmts: stmts}
	setPositionIfMissing(body, pos)
	return body
}

// withTerminationDelivery runs a worker's body as the hidden function bound to
// inner, then holds the worker until every async message it sent is received,
// and returns what the body returned. pos is where the worker terminates.
// inner is bound before the workers start, like every worker closure.
//
//	T $r = inner();
//	awaitWorkerMessageDelivery([<async send messages>], [<their receivers' futures>]);
//	return $r;
func withTerminationDelivery(
	cx *functionContext,
	inner *ast.BLangVarRef,
	returnTy semtypes.SemType,
	sends []*ast.BLangWorkerAsyncSendAction,
	pos diagnostics.Location,
) []ast.StatementNode {
	call := &ast.BLangInvocation{}
	call.Name = inner.VariableName
	call.SetSymbol(inner.Symbol())
	ast.SetType(call, returnTy)
	setPositionIfMissing(call, pos)
	resultDef, resultRef := assignToLocal(cx, call, pos)
	ret := &ast.BLangReturn{Expr: copyVarRef(resultRef)}
	setPositionIfMissing(ret, pos)
	return []ast.StatementNode{resultDef, awaitDeliveryStatement(cx, sends, pos), ret}
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
	fn := ast.NewBLangFunction(ast.InvokableData{
		Position:             pos,
		Name:                 nameNode,
		ReturnTypeDescriptor: returnTypeDescriptor,
		Body:                 body,
		Flags:                flags,
	})
	fn.SetSymbol(symRef)
	fn.SetScope(scope)
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
	pos diagnostics.Location,
) ast.StatementNode {
	invocation := &ast.BLangInvocation{}
	invocation.Name = closure.VariableName
	invocation.SetSymbol(closure.Symbol())
	ast.SetType(invocation, workerReturnType(cx, worker))
	setPositionIfMissing(invocation, pos)

	start := &ast.BLangStartAction{Call: invocation, IsIsolated: cx.isIsolated}
	ast.SetType(start, cx.symbolType(worker))
	setPositionIfMissing(start, pos)

	assignment := &ast.BLangAssignment{VarRef: copyVarRef(cx.workerFutureSlots[worker]), Expr: start}
	setPositionIfMissing(assignment, pos)
	return assignment
}

// workerFutureSlotRef replaces a value reference to a worker with a load of the
// worker's hidden future slot. It applies outside waits as well as within them;
// wait nodes keep their already-resolved source type.
func workerFutureSlotRef(cx *functionContext, reference *ast.BLangVarRef) ast.BLangExpression {
	slot, ok := cx.workerFutureSlots[reference.Symbol()]
	if !ok {
		// Sanity check: every worker slot is allocated before any worker body or
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
// namespace, so the worker's source name is qualified with its owner and its
// region's prefix; a worker name is unique within its region, region prefixes
// are unique within their owner, and lambdas are already uniquely named. The
// default worker's name can't clash with a source name.
func workerClosureName(cx *functionContext, namePrefix string, workerName string) string {
	if namePrefix == "" {
		return constants.WorkerClosurePrefix + cx.uniqueNamePrefix + ":" + workerName
	}
	return constants.WorkerClosurePrefix + cx.uniqueNamePrefix + ":" + namePrefix + ":" + workerName
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

func waitOnLatchInvocation(
	cx *functionContext,
	latch *ast.BLangVarRef,
	pos diagnostics.Location,
) ast.BLangExpression {
	return createLangInternalInvocation(cx, "waitOnLatch", semtypes.Nil,
		[]ast.BLangExpression{copyVarRef(latch)}, pos)
}

func copyVarRef(ref *ast.BLangVarRef) *ast.BLangVarRef {
	copied := &ast.BLangVarRef{VariableName: ref.VariableName}
	copied.SetSymbol(ref.Symbol())
	ast.SetType(copied, ref.Type())
	copied.SetPosition(ref.GetPosition())
	return copied
}

func expressionStatement(expr ast.BLangExpression, pos diagnostics.Location) ast.StatementNode {
	stmt := &ast.BLangExpressionStmt{Expr: expr}
	setPositionIfMissing(stmt, pos)
	return stmt
}
