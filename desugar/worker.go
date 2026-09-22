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
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common/constants"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// desugarWorkerRegion lowers the worker region of a block function body to
// existing language constructs. The emitted sequence is:
//
//	handle $latch = createLatch();
//	future<T1> $w1; ... future<Tn> $wn;
//	var $c1 = function () returns T1 { waitOnLatch($latch); <body 1> }; ...
//	$w1 = start $c1(); ... $wn = start $cn();
//	openLatch($latch);
//
// The only state shared between workers is the future slot table: a worker body
// may reference a sibling declared after it, so every slot is allocated before
// any body is lowered. Everything else a worker needs is passed to
// desugarNamedWorker. Every closure is constructed before the first start, and
// the latch is opened only after every slot has been assigned.
func desugarWorkerRegion(cx *functionContext, body *ast.BLangBlockFunctionBody) []ast.StatementNode {
	workers := body.Workers
	if len(workers) == 0 {
		return nil
	}
	pos := workers[0].GetPosition()

	latchDef, latchRef := assignToLocal(cx, createLatchInvocation(cx, pos), pos)
	stmts := make([]ast.StatementNode, 0, 3*len(workers)+2)
	stmts = append(stmts, latchDef)

	slotRefs := make([]*ast.BLangVarRef, len(workers))
	cx.workerFutureSlots = make(map[model.SymbolRef]*ast.BLangVarRef, len(workers))
	for i, worker := range workers {
		def, ref := declareWorkerFutureSlot(cx, worker, pos)
		stmts = append(stmts, def)
		slotRefs[i] = ref
		cx.workerFutureSlots[worker.Symbol()] = ref
	}

	starts := make([]ast.StatementNode, len(workers))
	for i, worker := range workers {
		closureDef, start := desugarNamedWorker(cx, worker, latchRef, slotRefs[i])
		stmts = append(stmts, closureDef)
		starts[i] = start
	}

	stmts = append(stmts, starts...)
	stmts = append(stmts, expressionStatement(openLatchInvocation(cx, latchRef, pos), pos))
	return stmts
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
	return closureDef, createWorkerStart(cx, worker, closureRef, slot)
}

// declareWorkerFutureSlot declares the hidden future slot of a worker without
// initialising it. The slot is assigned by the generated start action.
func declareWorkerFutureSlot(
	cx *functionContext,
	worker *ast.BLangNamedWorkerDeclaration,
	pos diagnostics.Location,
) (ast.StatementNode, *ast.BLangVarRef) {
	futureTy := cx.symbolType(worker.Symbol())
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
	returnTy := workerReturnType(cx, worker)
	isolated := cx.isIsolated

	workerBody := worker.Body
	latchWait := expressionStatement(waitOnLatchInvocation(cx, latch, pos), pos)
	workerBody.Stmts = append([]ast.StatementNode{latchWait}, workerBody.Stmts...)

	flags := model.FlagLambda | model.FlagAnonymous
	if isolated {
		flags |= model.FlagIsolated
	}
	closureTy := workerClosureType(cx, returnTy, isolated)
	name, symRef := addWorkerClosureSymbol(cx, worker, closureTy, returnTy, isolated)

	nameNode := newIdentifier(name)
	nameNode.SetDeterminedType(semtypes.Never)
	fn := ast.NewBLangFunction(ast.InvokableData{
		Position:             pos,
		Name:                 nameNode,
		ReturnTypeDescriptor: worker.ReturnType.TypeDescriptor,
		Body:                 workerBody,
		Flags:                flags,
	})
	fn.SetSymbol(symRef)
	fn.SetScope(worker.Scope())
	fn.SetDeterminedType(semtypes.Never)

	lambda := &ast.BLangLambdaFunction{Function: desugarNestedFunction(cx, fn)}
	lambda.SetDeterminedType(closureTy)
	setPositionIfMissing(lambda, pos)
	return lambda
}

// createWorkerStart starts a worker closure and publishes the resulting future
// into the worker's hidden slot. Scheduling metadata is set explicitly here
// because semantic analysis, which computes it for source start actions, has
// already run.
func createWorkerStart(
	cx *functionContext,
	worker *ast.BLangNamedWorkerDeclaration,
	closure *ast.BLangVarRef,
	slot *ast.BLangVarRef,
) ast.StatementNode {
	pos := worker.GetPosition()
	invocation := &ast.BLangInvocation{}
	invocation.Name = closure.VariableName
	invocation.SetSymbol(closure.Symbol())
	invocation.SetDeterminedType(workerReturnType(cx, worker))
	setPositionIfMissing(invocation, pos)

	start := &ast.BLangStartAction{Call: invocation, IsIsolated: cx.isIsolated}
	start.SetDeterminedType(cx.symbolType(worker.Symbol()))
	setPositionIfMissing(start, pos)

	assignment := &ast.BLangAssignment{VarRef: slot, Expr: start}
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

func workerReturnType(cx *functionContext, worker *ast.BLangNamedWorkerDeclaration) semtypes.SemType {
	return semtypes.FutureEventualType(cx.typeCtx(), cx.symbolType(worker.Symbol()))
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

// workerClosurePrefix marks a generated named-worker closure. The runtime
// relies on it to keep generated frames out of stack traces
// (`runtime/internal/exec.desugaredFunctionPrefixes`).
const workerClosurePrefix = constants.WorkerClosurePrefix

// workerClosureName is the name of the generated function a worker's body is
// lowered into. Generated functions share one package-wide lookup-key
// namespace, so the worker's source name is qualified with its owner; a worker
// name is unique within its owner, and lambdas are already uniquely named.
func workerClosureName(cx *functionContext, worker *ast.BLangNamedWorkerDeclaration) string {
	return workerClosurePrefix + cx.uniqueNamePrefix + ":" + worker.Name
}

// addWorkerClosureSymbol declares the generated closure for a worker.
func addWorkerClosureSymbol(
	cx *functionContext,
	worker *ast.BLangNamedWorkerDeclaration,
	closureTy semtypes.SemType,
	returnTy semtypes.SemType,
	isolated bool,
) (string, model.SymbolRef) {
	name := workerClosureName(cx, worker)
	signature := model.TypedFunctionSignature{ReturnType: returnTy, RestParamType: semtypes.Never}
	if isolated {
		signature.Flags |= model.FuncSymbolFlagIsolated
	}
	symbol := model.NewFunctionSymbol(name, signature, false, worker.GetPosition())
	symbol.SetType(closureTy)
	cx.currentScope().AddSymbol(name, symbol)
	ref, _ := cx.currentScope().GetSymbol(name)
	return name, ref
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
