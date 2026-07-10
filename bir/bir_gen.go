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

package bir

import (
	"fmt"
	"sort"

	"ballerina/ast"
	compilerctx "ballerina/context"
	"ballerina/desugar"
	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
	"ballerina/values"
)

// birLoc converts a diagnostics.Location (byte offsets) to a bir.Location (line/column)
// using a DiagnosticEnv to resolve byte offsets.
func birLoc(de *diagnostics.DiagnosticEnv, pos diagnostics.Location) Location {
	return NewLocation(de.FileName(pos), de.StartLine(pos), de.EndLine(pos), de.StartColumn(pos), de.EndColumn(pos))
}

// Since BLangNodeVisitor is anyway deprecated in jBallerina, we'll try to do this more cleanly

type Context struct {
	CompilerContext *compilerctx.CompilerContext
	packageID       *model.PackageID // Current package ID
	birPkg          *BIRPackage
	typeCtx         semtypes.Context
	// PR-TODO: extract them to memoized types struc
	stringMapTy      semtypes.SemType // Memoized map<string> type
	serviceClassKeys map[*ast.BLangService]string
}

func (c *Context) TypeContext() semtypes.Context {
	return c.typeCtx
}

// functionContext holds the per-function emission state. Functions nest
// (lambdas/closures): `definedIn` is the block the function literal sits in
// (nil at top level) and is the cross-function link a closure walks to
// resolve captured variables. `isClosure` is set when a variable reference
// inside this function resolves into an outer function's frame, and is read
// to emit FPLoad.IsClosure.
type functionContext struct {
	enclosing    context // defining-site block in the enclosing function; nil at top level
	bbs          []*BIRBasicBlock
	errorEntries []BIRErrorEntry
	birCx        *Context
	retVarDcl    *BIRLocalVariableDcl // the function's return variable (frame index 0 of the root frame)
	isClosure    bool                 // true iff captures variables otherwise treated as a simple lambda
}

func (fn *functionContext) addBB() *BIRBasicBlock {
	index := len(fn.bbs)
	bb := BB(index)
	fn.bbs = append(fn.bbs, &bb)
	return &bb
}

func (fn *functionContext) loc(pos diagnostics.Location) Location {
	return birLoc(fn.birCx.CompilerContext.DiagnosticEnv(), pos)
}

// context is one node of the lexical block tree. The whole tree (across
// nested lambdas, via `enclosing`) is the single source of truth for both
// variable resolution and abrupt-exit cleanup. Every block owns a runtime
// frame; `enclosing` stays within one function (nil at the funcBlock) and
// closure lookups cross into the enclosing function via fn.definedIn.
type context interface {
	enclosingBlock() context
	function() *functionContext
	addTempVar(ty semtypes.SemType) *BIROperand
	addLocalVar(name model.Name, ty semtypes.SemType, symbol model.SymbolRef) *BIROperand
	getLocalVar(symRef model.SymbolRef) (*BIROperand, bool)

	numLocals() int

	// Compiler-context proxies.
	compilerContext() *compilerctx.CompilerContext
	symbolType(symRef model.SymbolRef) semtypes.SemType
	getSymbol(symRef model.SymbolRef) model.Symbol
	unnarrowedSymbol(symRef model.SymbolRef) model.SymbolRef
	symbolName(symRef model.SymbolRef) string
	symbolPackage(symRef model.SymbolRef) model.PackageIdentifier
	typeEnv() semtypes.Env
	internalError(message string, pos diagnostics.Location)
	unimplemented(message string, pos diagnostics.Location)
}

// blockContext is the state every block shares, and is itself the node used
// for an ordinary block (if branch / match clause / bare block). funcBlock,
// loopBlock and lockBlock embed it.
type blockContext struct {
	fn        *functionContext
	enclosing context // lexical parent within this function; nil at the funcBlock
	localVars []*BIRLocalVariableDcl
	vars      map[model.SymbolRef]*BIROperand
}

type funcBlock struct{ blockContext }

type loopBlock struct {
	blockContext
	onBreakBB, onContinueBB *BIRBasicBlock
}

type lockBlock struct {
	blockContext
	key string
}

func newBlockContext(parent context) blockContext {
	return blockContext{fn: parent.function(), enclosing: parent, vars: make(map[model.SymbolRef]*BIROperand)}
}

func (c *Context) stringMapType() semtypes.SemType {
	if semtypes.IsZero(c.stringMapTy) {
		md := semtypes.NewMappingDefinition()
		c.stringMapTy = md.DefineMappingTypeWrapped(c.CompilerContext.GetTypeEnv(), nil, semtypes.STRING)
	}
	return c.stringMapTy
}

func (b *blockContext) enclosingBlock() context    { return b.enclosing }
func (b *blockContext) function() *functionContext { return b.fn }

func (b *blockContext) compilerContext() *compilerctx.CompilerContext {
	return b.fn.birCx.CompilerContext
}

func (b *blockContext) symbolType(symRef model.SymbolRef) semtypes.SemType {
	return b.compilerContext().SymbolType(symRef)
}

func (b *blockContext) getSymbol(symRef model.SymbolRef) model.Symbol {
	return b.compilerContext().GetSymbol(symRef)
}

func (b *blockContext) unnarrowedSymbol(symRef model.SymbolRef) model.SymbolRef {
	return b.compilerContext().UnnarrowedSymbol(symRef)
}

func (b *blockContext) symbolName(symRef model.SymbolRef) string {
	return b.compilerContext().SymbolName(symRef)
}

func (b *blockContext) symbolPackage(symRef model.SymbolRef) model.PackageIdentifier {
	return b.compilerContext().SymbolPackage(symRef)
}

func (b *blockContext) typeEnv() semtypes.Env {
	return b.compilerContext().GetTypeEnv()
}

func (b *blockContext) internalError(message string, pos diagnostics.Location) {
	b.compilerContext().InternalError(message, pos)
}

func (b *blockContext) unimplemented(message string, pos diagnostics.Location) {
	b.compilerContext().Unimplemented(message, pos)
}

func (b *blockContext) addLocalVarInner(name model.Name, ty semtypes.SemType) *BIROperand {
	varDcl := &BIRLocalVariableDcl{}
	varDcl.Name = name
	varDcl.Type = ty
	b.localVars = append(b.localVars, varDcl)
	return &BIROperand{VariableDcl: varDcl, Address: RelativeAddress(len(b.localVars) - 1)}
}

func (b *blockContext) addTempVar(ty semtypes.SemType) *BIROperand {
	return b.addLocalVarInner(model.Name(fmt.Sprintf("%%%d", len(b.localVars))), ty)
}

func (b *blockContext) addLocalVar(name model.Name, ty semtypes.SemType, symbol model.SymbolRef) *BIROperand {
	operand := b.addLocalVarInner(name, ty)
	b.vars[symbol] = operand
	return operand
}

func (b *blockContext) getLocalVar(symRef model.SymbolRef) (*BIROperand, bool) {
	op, ok := b.vars[symRef]
	return op, ok
}

// lookupVar resolves a symbol to an operand relative to b.
// return BIROperand, needs to capture from parent fn, found a matching operand
func lookupVar(b context, symRef model.SymbolRef) (*BIROperand, bool, bool) {
	levelsUp := 0
	crossed := false
	curFn := b.function()
	cur := b
	for cur != nil {
		if op, ok := cur.getLocalVar(symRef); ok {
			if levelsUp == 0 {
				return op, crossed, true
			}
			baseIndex := levelsUp
			if op.Address.Mode == AddressingModeAbsolute {
				baseIndex = levelsUp + op.Address.BaseIndex
			}
			return &BIROperand{VariableDcl: op.VariableDcl, Address: absoluteAddress(baseIndex, op.Address.FrameIndex)}, crossed, true
		}
		next := cur.enclosingBlock()
		if next == nil {
			// reached the funcBlock of curFn; cross into the defining function
			next = curFn.enclosing
			if next == nil {
				return nil, false, false
			}
			crossed = true
			curFn = next.function()
		}
		levelsUp++
		cur = next
	}
	return nil, false, false
}

// retVar returns the function's return variable addressed from block b.
func retVar(b context) *BIROperand {
	depth := 0
	cur := b
	for cur.enclosingBlock() != nil {
		depth++
		cur = cur.enclosingBlock()
	}
	retVarDcl := b.function().retVarDcl
	if depth == 0 {
		return &BIROperand{VariableDcl: retVarDcl, Address: RelativeAddress(0)}
	}
	return &BIROperand{VariableDcl: retVarDcl, Address: absoluteAddress(depth, 0)}
}

func (b *blockContext) numLocals() int { return len(b.localVars) }

// unwindInner emits the cleanup for leaving a single block: a PopScopeFrame,
// plus a LockEnd (ending the BB and continuing in a fresh one) when the block
// is a lock body. Shared by unwindLoop and unwindFunction.
func unwindInner(ctx context, curBB *BIRBasicBlock, pos Location) *BIRBasicBlock {
	curBB.Instructions = append(curBB.Instructions, &PopScopeFrame{BIRInstructionBase: BIRInstructionBase{BIRNodeBase: BIRNodeBase{Pos: pos}}})
	if lk, ok := ctx.(*lockBlock); ok {
		next := ctx.function().addBB()
		curBB.Terminator = NewLockEnd(lk.key, next, pos)
		curBB = next
	}
	return curBB
}

// unwindLoop emits the cleanup for a break/continue: it walks the enclosing
// chain from ctx calling unwindInner on each block and stops at (and
// including) the nearest loop block, which it returns so the caller can jump
// to its break/continue target.
func unwindLoop(ctx context, curBB *BIRBasicBlock, pos Location) (*BIRBasicBlock, *loopBlock) {
	for {
		curBB = unwindInner(ctx, curBB, pos)
		if lb, ok := ctx.(*loopBlock); ok {
			return curBB, lb
		}
		ctx = ctx.enclosingBlock()
	}
}

// unwindFunction emits the cleanup for a return/panic: it walks the enclosing
// chain from ctx to the function root calling unwindInner on each block. The
// root (call) frame is not popped — it is released by the Return/Panic.
func unwindFunction(ctx context, curBB *BIRBasicBlock, pos Location) *BIRBasicBlock {
	for ctx.enclosingBlock() != nil {
		curBB = unwindInner(ctx, curBB, pos)
		ctx = ctx.enclosingBlock()
	}
	return curBB
}

// functionRoot returns the function's root block and the number of frames
// between ctx and it.
func functionRoot(ctx context) (context, int) {
	depth := 0
	for ctx.enclosingBlock() != nil {
		ctx = ctx.enclosingBlock()
		depth++
	}
	return ctx, depth
}

// addFunctionTempVar allocates a temp in the function's root (call) frame. It
// returns an operand addressed from ctx (to store into the temp before
// unwinding) and one addressed from the root frame (to read after unwinding
// to the function root, where the root frame is current).
func addFunctionTempVar(ctx context, ty semtypes.SemType) (fromCtx, fromRoot *BIROperand) {
	root, depth := functionRoot(ctx)
	fromRoot = root.addTempVar(ty)
	if depth == 0 {
		return fromRoot, fromRoot
	}
	fromCtx = &BIROperand{VariableDcl: fromRoot.VariableDcl, Address: absoluteAddress(depth, fromRoot.Address.FrameIndex)}
	return fromCtx, fromRoot
}

// emitBlockBody pushes blk's frame, lowers stmts within blk, and on normal
// (fall-through) exit pops the frame, returning the final BB. Returns a nil
// block if control left abruptly (the abrupt-exit statement emitted its own
// unwind).
func emitBlockBody(blk context, bb *BIRBasicBlock, stmts []ast.StatementNode, pos Location) statementEffect {
	// Push this block's frame at the start of bb.
	push := &PushScopeFrame{}
	push.Pos = pos
	bb.Instructions = append(bb.Instructions, push)

	cur := bb
	for _, stmt := range stmts {
		effect := handleStatement(blk, cur, stmt)
		cur = effect.block
		if cur == nil {
			// Abrupt exit: size the frame even though the abrupt-exit
			// statement emitted its own PopScopeFrame.
			push.NumLocals = blk.numLocals()
			return statementEffect{}
		}
	}
	// Normal fall-through exit: size the frame and pop it.
	push.NumLocals = blk.numLocals()
	cur.Instructions = append(cur.Instructions, &PopScopeFrame{BIRInstructionBase: BIRInstructionBase{BIRNodeBase: BIRNodeBase{Pos: pos}}})
	return statementEffect{block: cur}
}

func buildLookupKey(pkg model.PackageIdentifier, qualifiedName string) string {
	return pkg.Organization + "/" + pkg.Package + ":" + qualifiedName
}

func packageIDFromIdentifier(ctx *compilerctx.CompilerContext, pkg model.PackageIdentifier) *model.PackageID {
	return ctx.NewPackageID(model.Name(pkg.Organization), model.CreateNameComps(model.Name(pkg.Package)), model.Name(pkg.Version))
}

func buildFunctionLookupKeyFromSymbol(ctx *Context, symRef model.SymbolRef) string {
	sym := ctx.CompilerContext.GetSymbol(symRef)
	if mono, ok := sym.(model.MonomorphicFunctionSymbol); ok {
		// For monomorphic functions (ex: dependently typed functions), in runtime we dispatch to a single
		// polymorphic function
		origRef := mono.PolymorphicSymbol()
		return buildLookupKey(ctx.CompilerContext.SymbolPackage(origRef), ctx.CompilerContext.SymbolName(origRef))
	}
	return buildLookupKey(ctx.CompilerContext.SymbolPackage(symRef), ctx.CompilerContext.SymbolName(symRef))
}

func buildMethodLookupKeyFromSymbol(ctx *Context, className string, symRef model.SymbolRef) string {
	return buildLookupKey(ctx.CompilerContext.SymbolPackage(symRef), className+"."+ctx.CompilerContext.SymbolName(symRef))
}

func buildGlobalVarLookupKey(pkgId *model.PackageID, name model.Name) string {
	return pkgId.OrgName.Value() + "/" + pkgId.PkgName.Value() + ":" + name.Value()
}

func newContext(compilerCtx *compilerctx.CompilerContext, packageID *model.PackageID, birPkg *BIRPackage) *Context {
	c := &Context{
		CompilerContext:  compilerCtx,
		packageID:        packageID,
		birPkg:           birPkg,
		serviceClassKeys: make(map[*ast.BLangService]string),
		typeCtx:          semtypes.TypeCheckContext(compilerCtx.GetTypeEnv()),
	}
	return c
}

func GenBir(ctx *compilerctx.CompilerContext, ast *ast.BLangPackage) *BIRPackage {
	birPkg := &BIRPackage{}
	birPkg.PackageID = ast.PackageID
	genCtx := newContext(ctx, ast.PackageID, birPkg)
	birPkg.GlobalVars = make(map[string]BIRGlobalVariableDcl)
	for _, globalVar := range ast.GlobalVars {
		addGlobalVar(birPkg, TransformGlobalVariableDcl(genCtx, &globalVar))
	}
	// Constants are never added to the BIR package: a const-expr is evaluated at
	// compile time, so a foldable constant is inlined at its use sites during
	// desugar, and a constant that cannot be folded is a compile-time error
	// (see resolveConstant). Either way no constant survives to BIR generation.
	for i := range ast.ClassDefinitions {
		transformClassDefinition(genCtx, &ast.ClassDefinitions[i], birPkg)
	}
	for i := range ast.Services {
		transformService(genCtx, &ast.Services[i], i, birPkg)
	}
	if ast.InitFunction != nil {
		birPkg.InitFunction = TransformFunction(genCtx, ast.InitFunction)
	}
	for _, function := range ast.Functions {
		if function.IsNative() {
			continue
		}
		birFunc := TransformFunction(genCtx, &function)
		birPkg.Functions = append(birPkg.Functions, *birFunc)
		switch birFunc.Name.Value() {
		case "main":
			birPkg.MainFunction = birFunc
		case desugar.StartFunctionName:
			birPkg.StartFunction = birFunc
		case desugar.GracefulStopFunctionName:
			birPkg.GracefulStopFunction = birFunc
		case desugar.ImmediateStopFunctionName:
			birPkg.ImmediateStopFunction = birFunc
		}
	}
	return birPkg
}

func addGlobalVar(birPkg *BIRPackage, dcl BIRGlobalVariableDcl) {
	birPkg.GlobalVars[dcl.GlobalVarLookupKey] = dcl
}

func TransformGlobalVariableDcl(ctx *Context, ast *ast.BLangSimpleVariable) BIRGlobalVariableDcl {
	name := model.Name(ast.GetName().GetValue())
	dcl := BIRGlobalVariableDcl{}
	dcl.Pos = birLoc(ctx.CompilerContext.DiagnosticEnv(), ast.GetPosition())
	dcl.Name = name
	dcl.PkgId = ctx.packageID
	dcl.Type = ctx.CompilerContext.SymbolType(ast.Symbol())
	dcl.Flags = ast.Flags()
	dcl.GlobalVarLookupKey = buildGlobalVarLookupKey(ctx.packageID, name)
	return dcl
}

func TransformFunction(ctx *Context, astFunc *ast.BLangFunction) *BIRFunction {
	return transformFunctionInner(newFunctionRoot(ctx, nil), astFunc, nil)
}

// newFunctionRoot creates the root block (the call frame) of a function.
// definedIn is the block the function literal sits in (nil at top level),
// used by closures to resolve captured variables.
func newFunctionRoot(ctx *Context, definedIn context) *funcBlock {
	fn := &functionContext{birCx: ctx, enclosing: definedIn}
	return &funcBlock{blockContext{fn: fn, vars: make(map[model.SymbolRef]*BIROperand)}}
}

func transformFunctionInner(root *funcBlock, astFunc *ast.BLangFunction, selfSymbolRef *model.SymbolRef) *BIRFunction {
	symRef := astFunc.Symbol()
	funcName := model.Name(astFunc.GetName().GetValue())
	birFunc := &BIRFunction{}
	birFunc.Pos = root.fn.loc(astFunc.GetPosition())
	birFunc.Name = funcName
	birFunc.OriginalName = funcName
	birFunc.Flags = astFunc.Flags()
	ctx := root.fn.birCx
	birFunc.FunctionLookupKey = buildFunctionLookupKeyFromSymbol(ctx, symRef)
	funcSym := ctx.CompilerContext.GetSymbol(astFunc.Symbol()).(model.FunctionSymbol)
	retOp := root.addLocalVarInner(model.Name("%0"), funcSym.Signature().ReturnType)
	root.fn.retVarDcl = retOp.VariableDcl.(*BIRLocalVariableDcl)
	if selfSymbolRef != nil {
		root.addLocalVar(model.Name("self"), ctx.CompilerContext.SymbolType(*selfSymbolRef), *selfSymbolRef)
	}
	requiredParams := make([]BIRParameter, len(astFunc.RequiredParams))
	for i, param := range astFunc.RequiredParams {
		root.addLocalVar(model.Name(param.GetName().GetValue()), ctx.CompilerContext.SymbolType(param.Symbol()), param.Symbol())
		requiredParams[i] = BIRParameter{
			Name:  model.Name(param.GetName().GetValue()),
			Flags: param.Flags(),
		}
	}
	if astFunc.RestParam != nil {
		restParam := astFunc.RestParam
		ty := ctx.CompilerContext.SymbolType(restParam.Symbol())
		root.addLocalVar(model.Name(restParam.GetName().GetValue()), ty, restParam.Symbol())
		birFunc.RestParams = &BIRParameter{Name: model.Name(restParam.GetName().GetValue())}
	}
	birFunc.RequiredParams = requiredParams
	switch body := astFunc.Body.(type) {
	case *ast.BLangBlockFunctionBody:
		handleBlockFunctionBody(root, body)
	case *ast.BLangExprFunctionBody:
		handleExprFunctionBody(root, body)
	default:
		panic("unexpected function body type")
	}
	for _, bbPtr := range root.fn.bbs {
		birFunc.BasicBlocks = append(birFunc.BasicBlocks, *bbPtr)
	}
	for _, varPtr := range root.localVars {
		birFunc.LocalVars = append(birFunc.LocalVars, *varPtr)
	}
	birFunc.ErrorTable = root.fn.errorEntries
	birFunc.ReturnVariable = root.fn.retVarDcl
	return birFunc
}

func handleBlockFunctionBody(ctx context, ast *ast.BLangBlockFunctionBody) {
	curBB := ctx.function().addBB()
	for _, stmt := range ast.Stmts {
		effect := handleStatement(ctx, curBB, stmt)
		curBB = effect.block
		if curBB == nil {
			return
		}
	}
	// Add implicit return
	curBB.Terminator = NewReturn(ctx.function().loc(ast.GetPosition()))
}

type statementEffect struct {
	block *BIRBasicBlock
}

func handleStatement(ctx context, curBB *BIRBasicBlock, stmt ast.StatementNode) statementEffect {
	switch stmt := stmt.(type) {
	case *ast.BLangExpressionStmt:
		return expressionStatement(ctx, curBB, stmt)
	case *ast.BLangIf:
		return ifStatement(ctx, curBB, stmt)
	case *ast.BLangBlockStmt:
		return blockStatement(ctx, curBB, stmt)
	case *ast.BLangReturn:
		return returnStatement(ctx, curBB, stmt)
	case *ast.BLangSimpleVariableDef:
		return simpleVariableDefinition(ctx, curBB, stmt)
	case *ast.BLangAssignment:
		return assignmentStatement(ctx, curBB, stmt)
	case *ast.BLangCompoundAssignment:
		return compoundAssignment(ctx, curBB, stmt)
	case *ast.BLangWhile:
		return whileStatement(ctx, curBB, stmt)
	case *ast.BLangBreak:
		return breakStatement(ctx, curBB, stmt)
	case *ast.BLangContinue:
		return continueStatement(ctx, curBB, stmt)
	case *ast.BLangPanic:
		return panicStatement(ctx, curBB, stmt)
	case *ast.BLangMatchStatement:
		return matchStatement(ctx, curBB, stmt)
	case *ast.BLangXMLNS:
		// xmlns declarations have no runtime effect.
		return statementEffect{block: curBB}
	case *ast.BLangLock:
		return lockStatement(ctx, curBB, stmt)
	default:
		panic("unexpected statement type")
	}
}

func lockStatement(ctx context, bb *BIRBasicBlock, stmt *ast.BLangLock) statementEffect {
	pos := ctx.function().loc(stmt.GetPosition())
	if stmt.LockKey == "" {
		ctx.internalError("lock statement reached BIR-gen without a lock key", stmt.GetPosition())
	}
	key := stmt.LockKey
	bodyEntry := ctx.function().addBB()
	bb.Terminator = NewLockStart(key, bodyEntry, pos)
	lk := &lockBlock{blockContext: newBlockContext(ctx), key: key}
	bodyEffect := emitBlockBody(lk, bodyEntry, stmt.Body.Stmts, pos)
	afterLock := ctx.function().addBB()
	if bodyEffect.block != nil {
		bodyEffect.block.Terminator = NewLockEnd(key, afterLock, pos)
	}
	return statementEffect{block: afterLock}
}

func compoundAssignment(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangCompoundAssignment) statementEffect {
	pos := ctx.function().loc(stmt.GetPosition())
	if indexRef, ok := stmt.VarRef.(*ast.BLangIndexBasedAccess); ok {
		return compoundAssignmentToMember(ctx, curBB, stmt, indexRef, pos)
	}
	ref := stmt.VarRef
	valueEffect := binaryExpressionInner(ctx, curBB, stmt.OpKind, ref, stmt.Expr, stmt.Expr.GetDeterminedType(), pos)
	return assignmentStatementInner(ctx, ref, valueEffect, pos)
}

// compoundAssignmentToMember handles compound assignment with an index-based access LHS
// (e.g. `x[i] += rhs`). The container reference and index expression must be evaluated
// only once even though the LHS is conceptually both read and written.
func compoundAssignmentToMember(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangCompoundAssignment, ref *ast.BLangIndexBasedAccess, pos Location) statementEffect {
	containerEffect := assignmentContainerReference(ctx, curBB, ref.Expr)
	indexEffect := handleActionOrExpression(ctx, containerEffect.block, ref.IndexExpr)
	curBB = indexEffect.block

	loadKind, storeKind := memberAccessInstructionKinds(ctx.function().birCx.TypeContext(), ref.Expr.GetDeterminedType())

	lhsValue := ctx.addTempVar(ref.GetDeterminedType())
	load := NewFieldAccess(loadKind, lhsValue, indexEffect.result, containerEffect.result, pos)
	curBB.Instructions = append(curBB.Instructions, load)
	lhsEffect := snapshotIfNeeded(ctx, expressionEffect{result: lhsValue, block: curBB}, pos)
	curBB = lhsEffect.block

	rhsEffect := handleActionOrExpression(ctx, curBB, stmt.Expr)
	rhsEffect = snapshotIfNeeded(ctx, rhsEffect, pos)
	curBB = rhsEffect.block

	resultOperand := ctx.addTempVar(ref.GetDeterminedType())
	binaryOp := NewBinaryOp(operatorKindToBinaryInstructionKind(stmt.OpKind), resultOperand, lhsEffect.result, rhsEffect.result, pos)
	curBB.Instructions = append(curBB.Instructions, binaryOp)

	store := NewFieldAccess(storeKind, containerEffect.result, indexEffect.result, resultOperand, pos)
	curBB.Instructions = append(curBB.Instructions, store)
	return statementEffect{
		block: curBB,
	}
}

func memberAccessInstructionKinds(tyCtx semtypes.Context, containerType semtypes.SemType) (loadKind, storeKind InstructionKind) {
	containerType = semtypes.Diff(containerType, semtypes.NIL)
	switch {
	case semtypes.IsSubtype(tyCtx, containerType, semtypes.LIST):
		return INSTRUCTION_KIND_ARRAY_LOAD, INSTRUCTION_KIND_ARRAY_STORE
	case semtypes.IsSubtype(tyCtx, containerType, semtypes.OBJECT):
		return INSTRUCTION_KIND_OBJECT_LOAD, INSTRUCTION_KIND_OBJECT_STORE
	default:
		return INSTRUCTION_KIND_MAP_LOAD, INSTRUCTION_KIND_MAP_STORE
	}
}

func continueStatement(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangContinue) statementEffect {
	pos := ctx.function().loc(stmt.GetPosition())
	curBB, loop := unwindLoop(ctx, curBB, pos)
	curBB.Terminator = NewGoto(loop.onContinueBB, pos)
	// We don't know where to add the next statement so we return nil
	return statementEffect{block: nil}
}

func breakStatement(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangBreak) statementEffect {
	pos := ctx.function().loc(stmt.GetPosition())
	curBB, loop := unwindLoop(ctx, curBB, pos)
	curBB.Terminator = NewGoto(loop.onBreakBB, pos)
	// We don't know where to add the next statement so we return nil
	return statementEffect{block: nil}
}

func whileStatement(ctx context, bb *BIRBasicBlock, stmt *ast.BLangWhile) statementEffect {
	pos := ctx.function().loc(stmt.GetPosition())
	loopHead := ctx.function().addBB()
	// jump to loop head
	bb.Terminator = NewGoto(loopHead, pos)
	// The loop condition is evaluated in the enclosing block's frame.
	condEffect := handleActionOrExpression(ctx, loopHead, stmt.Expr)

	loopBody := ctx.function().addBB()
	loopEnd := ctx.function().addBB()
	// conditionally jump to loop body
	condEffect.block.Terminator = NewBranch(condEffect.result, loopBody, loopEnd, pos)

	// Each iteration gets its own frame; emitBlockBody pushes/pops it.
	loop := &loopBlock{blockContext: newBlockContext(ctx), onBreakBB: loopEnd, onContinueBB: loopHead}
	bodyEffect := emitBlockBody(loop, loopBody, stmt.Body.Stmts, pos)

	// This could happen if the while block always ends return, break or continue
	if bodyEffect.block != nil {
		bodyEffect.block.Terminator = NewGoto(loopHead, pos)
	}
	return statementEffect{
		block: loopEnd,
	}
}

func assignmentStatement(ctx context, bb *BIRBasicBlock, stmt *ast.BLangAssignment) statementEffect {
	valueEffect := handleActionOrExpression(ctx, bb, stmt.Expr)
	return assignmentStatementInner(ctx, stmt.VarRef, valueEffect, ctx.function().loc(stmt.GetPosition()))
}

func assignmentStatementInner(ctx context, ref ast.BLangExpression, valueEffect expressionEffect, pos Location) statementEffect {
	switch varRef := ref.(type) {
	case *ast.BLangIndexBasedAccess:
		return assignToMemberStatement(ctx, varRef, valueEffect, pos)
	case *ast.BLangWildCardBindingPattern:
		return assignToWildcardBindingPattern(ctx, varRef, valueEffect, pos)
	case *ast.BLangSimpleVarRef:
		return assignToSimpleVariable(ctx, varRef, valueEffect, pos)
	default:
		panic("unexpected variable reference type")
	}
}

func assignToWildcardBindingPattern(ctx context, varRef *ast.BLangWildCardBindingPattern, valueEffect expressionEffect, pos Location) statementEffect {
	refEffect := wildcardBindingPattern(ctx, valueEffect.block, varRef)
	currBB := refEffect.block
	mov := NewMove(valueEffect.result, refEffect.result, pos)
	currBB.Instructions = append(currBB.Instructions, mov)
	return statementEffect{
		block: currBB,
	}
}

func assignToSimpleVariable(ctx context, varRef *ast.BLangSimpleVarRef, valueEffect expressionEffect, pos Location) statementEffect {
	refEffect := simpleVariableReference(ctx, valueEffect.block, varRef)
	currBB := refEffect.block
	mov := NewMove(valueEffect.result, refEffect.result, pos)
	currBB.Instructions = append(currBB.Instructions, mov)
	return statementEffect{
		block: currBB,
	}
}

func assignToMemberStatement(ctx context, varRef *ast.BLangIndexBasedAccess, valueEffect expressionEffect, pos Location) statementEffect {
	currBB := valueEffect.block
	containerRefEffect := assignmentContainerReference(ctx, currBB, varRef.Expr)
	currBB = containerRefEffect.block
	indexEffect := handleActionOrExpression(ctx, currBB, varRef.IndexExpr)
	currBB = indexEffect.block
	_, storeKind := memberAccessInstructionKinds(ctx.function().birCx.TypeContext(), varRef.Expr.GetDeterminedType())
	fieldAccess := NewFieldAccess(storeKind, containerRefEffect.result, indexEffect.result, valueEffect.result, pos)
	currBB.Instructions = append(currBB.Instructions, fieldAccess)
	return statementEffect{
		block: currBB,
	}
}

func simpleVariableDefinition(ctx context, bb *BIRBasicBlock, stmt *ast.BLangSimpleVariableDef) statementEffect {
	ty := ctx.symbolType(stmt.Var.Symbol())
	varName := model.Name(stmt.Var.GetName().GetValue())
	if stmt.Var.Expr == nil {
		ctx.addLocalVar(varName, ty, stmt.Var.Symbol())
		// just declare the variable
		return statementEffect{
			block: bb,
		}
	}
	exprResult := handleActionOrExpression(ctx, bb, stmt.Var.Expr)
	curBB := exprResult.block
	lhsOp := ctx.addLocalVar(varName, ty, stmt.Var.Symbol())
	move := NewMove(exprResult.result, lhsOp, ctx.function().loc(stmt.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, move)
	return statementEffect{
		block: curBB,
	}
}

func returnStatement(ctx context, bb *BIRBasicBlock, stmt *ast.BLangReturn) statementEffect {
	curBB := bb
	pos := ctx.function().loc(stmt.GetPosition())
	if stmt.Expr != nil {
		valueEffect := handleActionOrExpression(ctx, curBB, stmt.Expr)
		curBB = valueEffect.block
		mov := NewMove(valueEffect.result, retVar(ctx), pos)
		curBB.Instructions = append(curBB.Instructions, mov)
	}
	curBB = unwindFunction(ctx, curBB, pos)
	curBB.Terminator = NewReturn(pos)
	return statementEffect{}
}

func panicStatement(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangPanic) statementEffect {
	pos := ctx.function().loc(stmt.GetPosition())
	errorEffect := handleActionOrExpression(ctx, curBB, stmt.Expr)
	curBB = errorEffect.block
	// The frame pops emitted by unwindFunction discard the frame holding the
	// error operand, so when there are frames to pop, stash it in a
	// function-level temp (read back from the root frame after unwinding).
	panicOp := errorEffect.result
	if _, depth := functionRoot(ctx); depth > 0 {
		store, fromRoot := addFunctionTempVar(ctx, errorEffect.result.VariableDcl.GetType())
		curBB.Instructions = append(curBB.Instructions, NewMove(errorEffect.result, store, pos))
		panicOp = fromRoot
	}
	curBB = unwindFunction(ctx, curBB, pos)
	curBB.Terminator = NewPanic(panicOp, pos)
	return statementEffect{}
}

func expressionStatement(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangExpressionStmt) statementEffect {
	result := handleActionOrExpression(ctx, curBB, stmt.Expr)
	// We are ignoring the expression result (We can have one for things like call)
	return statementEffect{
		block: result.block,
	}
}

func ifStatement(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangIf) statementEffect {
	cond := handleActionOrExpression(ctx, curBB, stmt.Expr)
	curBB = cond.block
	thenBB := ctx.function().addBB()
	var finalBB *BIRBasicBlock
	thenEffect := blockStatement(ctx, thenBB, &stmt.Body)
	// TODO: refactor this
	if stmt.ElseStmt != nil {
		elseBB := ctx.function().addBB()
		// Add branch to current BB
		curBB.Terminator = NewBranch(cond.result, thenBB, elseBB, ctx.function().loc(stmt.GetPosition()))

		elseEffect := handleStatement(ctx, elseBB, stmt.ElseStmt)
		finalBB = ctx.function().addBB()
		if elseEffect.block != nil {
			elseEffect.block.Terminator = NewGoto(finalBB, ctx.function().loc(stmt.GetPosition()))
		}
	} else {
		finalBB = ctx.function().addBB()
		curBB.Terminator = NewBranch(cond.result, thenBB, finalBB, ctx.function().loc(stmt.GetPosition()))
	}
	// this could be nil if the control flow moved out of the if (ex: break, continue, return, etc)
	if thenEffect.block != nil {
		thenEffect.block.Terminator = NewGoto(finalBB, ctx.function().loc(stmt.GetPosition()))
	}
	return statementEffect{
		block: finalBB,
	}
}

func blockStatement(ctx context, bb *BIRBasicBlock, stmt *ast.BLangBlockStmt) statementEffect {
	child := newBlockContext(ctx)
	return emitBlockBody(&child, bb, stmt.Stmts, ctx.function().loc(stmt.GetPosition()))
}

func matchStatement(ctx context, curBB *BIRBasicBlock, stmt *ast.BLangMatchStatement) statementEffect {
	exprEffect := handleActionOrExpression(ctx, curBB, stmt.Expr)
	curBB = exprEffect.block
	matchOperand := exprEffect.result
	finalBB := ctx.function().addBB()

	for _, clause := range stmt.MatchClauses {
		clauseBodyBB := ctx.function().addBB()

		if isUnconditionalWildcard(&clause) {
			curBB.Terminator = NewGoto(clauseBodyBB, ctx.function().loc(stmt.GetPosition()))
			bodyEffect := blockStatement(ctx, clauseBodyBB, &clause.Body)
			if bodyEffect.block != nil {
				bodyEffect.block.Terminator = NewGoto(finalBB, ctx.function().loc(stmt.GetPosition()))
			}
			continue
		}

		var condOperand *BIROperand
		for _, pattern := range clause.Patterns {
			switch p := pattern.(type) {
			case *ast.BLangConstPattern:
				patternEffect := handleActionOrExpression(ctx, curBB, p.Expr)
				curBB = patternEffect.block
				eqResult := ctx.addTempVar(semtypes.BOOLEAN)
				eqPos := ctx.function().loc(p.Expr.GetPosition())
				binaryOp := NewBinaryOp(INSTRUCTION_KIND_EQUAL, eqResult, matchOperand, patternEffect.result, eqPos)
				curBB.Instructions = append(curBB.Instructions, binaryOp)
				condOperand = orOperands(ctx, curBB, condOperand, eqResult, eqPos)
			case *ast.BLangWildCardMatchPattern:
				// Wildcard in multi-pattern — always matches; but may have guard
				trueOperand := ctx.addTempVar(semtypes.BOOLEAN)
				constLoad := NewConstantLoad(trueOperand, true, ctx.function().loc(p.GetPosition()))
				curBB.Instructions = append(curBB.Instructions, constLoad)
				condOperand = orOperands(ctx, curBB, condOperand, trueOperand, ctx.function().loc(p.GetPosition()))
			default:
				ctx.internalError("unexpected match pattern type", pattern.GetPosition())
			}
		}

		if clause.Guard != nil {
			guardEffect := handleActionOrExpression(ctx, curBB, clause.Guard)
			curBB = guardEffect.block
			condOperand = andOperands(ctx, curBB, condOperand, guardEffect.result, ctx.function().loc(clause.Guard.GetPosition()))
		}

		nextCheckBB := ctx.function().addBB()
		curBB.Terminator = NewBranch(condOperand, clauseBodyBB, nextCheckBB, ctx.function().loc(stmt.GetPosition()))

		bodyEffect := blockStatement(ctx, clauseBodyBB, &clause.Body)
		if bodyEffect.block != nil {
			bodyEffect.block.Terminator = NewGoto(finalBB, ctx.function().loc(stmt.GetPosition()))
		}

		curBB = nextCheckBB
	}

	if !stmt.IsExhaustive {
		curBB.Terminator = NewGoto(finalBB, ctx.function().loc(stmt.GetPosition()))
	}

	return statementEffect{block: finalBB}
}

func isUnconditionalWildcard(clause *ast.BLangMatchClause) bool {
	if clause.Guard != nil {
		return false
	}
	if len(clause.Patterns) != 1 {
		return false
	}
	_, ok := clause.Patterns[0].(*ast.BLangWildCardMatchPattern)
	return ok
}

func orOperands(ctx context, bb *BIRBasicBlock, existing *BIROperand, new *BIROperand, pos Location) *BIROperand {
	if existing == nil {
		return new
	}
	result := ctx.addTempVar(semtypes.BOOLEAN)
	binaryOp := NewBinaryOp(INSTRUCTION_KIND_OR, result, existing, new, pos)
	bb.Instructions = append(bb.Instructions, binaryOp)
	return result
}

func andOperands(ctx context, bb *BIRBasicBlock, existing *BIROperand, new *BIROperand, pos Location) *BIROperand {
	result := ctx.addTempVar(semtypes.BOOLEAN)
	binaryOp := NewBinaryOp(INSTRUCTION_KIND_AND, result, existing, new, pos)
	bb.Instructions = append(bb.Instructions, binaryOp)
	return result
}

func handleExprFunctionBody(ctx context, body *ast.BLangExprFunctionBody) {
	curBB := ctx.function().addBB()
	effect := handleActionOrExpression(ctx, curBB, body.Expr)
	curBB = effect.block
	if curBB != nil {
		retAssign := &Move{}
		retAssign.LhsOp = retVar(ctx)
		retAssign.RhsOp = effect.result
		curBB.Instructions = append(curBB.Instructions, retAssign)
		curBB.Terminator = &Return{}
	}
}

func lambdaFunction(ctx context, curBB *BIRBasicBlock, expr *ast.BLangLambdaFunction) expressionEffect {
	root := newFunctionRoot(ctx.function().birCx, ctx)
	birFunc := transformFunctionInner(root, expr.Function, nil)
	ctx.function().birCx.birPkg.Functions = append(ctx.function().birCx.birPkg.Functions, *birFunc)
	funcType := expr.GetDeterminedType()
	resultOperand := ctx.addTempVar(funcType)
	fpLoad := &FPLoad{}
	fpLoad.Pos = ctx.function().loc(expr.GetPosition())
	fpLoad.FunctionLookupKey = birFunc.FunctionLookupKey
	fpLoad.Type = funcType
	fpLoad.IsClosure = root.fn.isClosure
	fpLoad.LhsOp = resultOperand
	curBB.Instructions = append(curBB.Instructions, fpLoad)
	// If the inner function is a closure, this function also needs parent frame
	// access to maintain the frame chain for nested closures
	if root.fn.isClosure {
		ctx.function().isClosure = true
	}
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

type expressionEffect struct {
	result *BIROperand
	block  *BIRBasicBlock
}

// snapshotIfNeeded stores values without storage identity in a temp var before referencing so that modification in one part
// of an expression dont' affect the other.
func snapshotIfNeeded(ctx context, effect expressionEffect, pos Location) expressionEffect {
	op := effect.result
	if _, isLocal := op.VariableDcl.(*BIRLocalVariableDcl); isLocal && hasNoStorageIdentity(ctx.function().birCx.TypeContext(), op.VariableDcl.GetType()) {
		tempOp := ctx.addTempVar(op.VariableDcl.GetType())
		effect.block.Instructions = append(effect.block.Instructions, NewMove(op, tempOp, pos))
		effect.result = tempOp
	}
	return effect
}

func handleActionOrExpression(ctx context, curBB *BIRBasicBlock, expr ast.BLangActionOrExpression) expressionEffect {
	switch expr := expr.(type) {
	case *ast.BLangInvocation:
		return generateCall(ctx, curBB, expr)
	case *ast.BLangLiteral:
		return literal(ctx, curBB, expr)
	case *ast.BLangNumericLiteral:
		return literal(ctx, curBB, &expr.BLangLiteral)
	case *ast.BLangBinaryExpr:
		return binaryExpression(ctx, curBB, expr)
	case *ast.BLangSimpleVarRef:
		return simpleVariableReference(ctx, curBB, expr)
	case *ast.BLangUnaryExpr:
		return unaryExpression(ctx, curBB, expr)
	case *ast.BLangWildCardBindingPattern:
		return wildcardBindingPattern(ctx, curBB, expr)
	case *ast.BLangGroupExpr:
		return groupExpression(ctx, curBB, expr)
	case *ast.BLangIndexBasedAccess:
		return indexBasedAccess(ctx, curBB, expr)
	case *ast.BLangListConstructorExpr:
		return listConstructorExpression(ctx, curBB, expr)
	case *ast.BLangTypeConversionExpr:
		return typeConversionExpression(ctx, curBB, expr)
	case *ast.BLangTypeTestExpr:
		return typeTestExpression(ctx, curBB, expr)
	case *ast.BLangMappingConstructorExpr:
		return mappingConstructorExpression(ctx, curBB, expr)
	case *ast.BLangAnnotAccessExpr:
		return annotAccessExpression(ctx, curBB, expr)
	case *ast.BLangErrorConstructorExpr:
		return errorConstructorExpression(ctx, curBB, expr)
	case *ast.BLangTrapExpr:
		return trapExpression(ctx, curBB, expr)
	case *ast.BLangNewExpression:
		return newExpression(ctx, curBB, expr)
	case *desugar.BLangServiceInit:
		return serviceInitExpression(ctx, curBB, expr)
	case *ast.BLangLambdaFunction:
		return lambdaFunction(ctx, curBB, expr)
	case *ast.BLangRemoteMethodCallAction:
		return generateCall(ctx, curBB, expr)
	case *ast.BLangClientResourceAccessAction:
		return generateResourceAccessCall(ctx, curBB, expr)
	case *ast.BLangTypedescExpr:
		return typedescExpression(ctx, curBB, expr)
	case *ast.BLangXMLSequenceLiteral:
		return xmlSequenceLiteral(ctx, curBB, expr)
	case *ast.BLangXMLElementLiteral:
		return xmlElementLiteral(ctx, curBB, expr)
	case *ast.BLangXMLPILiteral:
		return xmlPILiteral(ctx, curBB, expr)
	case *ast.BLangXMLCommentLiteral:
		return xmlCommentLiteral(ctx, curBB, expr)
	case *ast.BLangXMLTextLiteral:
		return xmlTextLiteral(ctx, curBB, expr)
	case *ast.BLangTemplateExpr:
		return templateExpression(ctx, curBB, expr)
	default:
		panic(fmt.Sprintf("unexpected expression type: %T", expr))
	}
}

func typedescExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangTypedescExpr) expressionEffect {
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	td := values.NewTypeDesc(expr.Constraint, expr.AnnotationValues)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(resultOperand, td, ctx.function().loc(expr.GetPosition())))
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func annotAccessExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangAnnotAccessExpr) expressionEffect {
	pos := ctx.function().loc(expr.GetPosition())
	receiver := handleActionOrExpression(ctx, curBB, expr.Expr)
	curBB = receiver.block
	symRef := expr.Symbol()
	sym := ctx.getSymbol(symRef)
	keyOp := ctx.addTempVar(semtypes.STRING)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(keyOp, model.AnnotationKey(ctx.symbolPackage(symRef), sym.Name()), pos))
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewBinaryOp(INSTRUCTION_KIND_ANNOT_ACCESS, resultOperand, receiver.result, keyOp, pos))
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func xmlTextLiteral(ctx context, curBB *BIRBasicBlock, expr *ast.BLangXMLTextLiteral) expressionEffect {
	pos := ctx.function().loc(expr.GetPosition())
	bodyOp := ctx.addTempVar(semtypes.STRING)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(bodyOp, expr.Body, pos))
	resultOp := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewXMLTextInstr(resultOp, bodyOp, pos))
	return expressionEffect{result: resultOp, block: curBB}
}

func xmlCommentLiteral(ctx context, curBB *BIRBasicBlock, expr *ast.BLangXMLCommentLiteral) expressionEffect {
	pos := ctx.function().loc(expr.GetPosition())
	bodyOp := ctx.addTempVar(semtypes.STRING)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(bodyOp, expr.Body, pos))
	resultOp := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewXMLCommentInstr(resultOp, bodyOp, pos))
	return expressionEffect{result: resultOp, block: curBB}
}

func xmlPILiteral(ctx context, curBB *BIRBasicBlock, expr *ast.BLangXMLPILiteral) expressionEffect {
	pos := ctx.function().loc(expr.GetPosition())
	targetOp := ctx.addTempVar(semtypes.STRING)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(targetOp, expr.Target, pos))
	dataOp := ctx.addTempVar(semtypes.STRING)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(dataOp, expr.Data, pos))
	resultOp := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewXMLPIInstr(resultOp, targetOp, dataOp, pos))
	return expressionEffect{result: resultOp, block: curBB}
}

func xmlElementLiteral(ctx context, curBB *BIRBasicBlock, expr *ast.BLangXMLElementLiteral) expressionEffect {
	pos := ctx.function().loc(expr.GetPosition())
	nameOp := ctx.addTempVar(semtypes.STRING)
	curBB.Instructions = append(curBB.Instructions, NewConstantLoad(nameOp, expr.Name, pos))
	var contentOp *BIROperand
	if expr.Content != nil {
		eff := handleActionOrExpression(ctx, curBB, expr.Content)
		curBB = eff.block
		contentOp = eff.result
	}
	var attrsOp *BIROperand
	if len(expr.Attrs) > 0 {
		fields := make([]mappingField, 0, len(expr.Attrs))
		for _, attr := range expr.Attrs {
			fields = append(fields, mappingField{key: attr.Name, value: attr.Value})
		}
		attrMapEff := mappingConstructorExpressionInner(ctx, curBB, ctx.function().birCx.stringMapType(), fields, nil, pos)
		curBB = attrMapEff.block
		attrsOp = attrMapEff.result
	}
	var namespacesOp *BIROperand
	if len(expr.Namespaces) > 0 {
		namespacesOp, curBB = buildXMLNamespacesMap(ctx, curBB, expr.Namespaces, pos)
	}
	resultOp := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewXMLElementInstr(resultOp, nameOp, contentOp, attrsOp, namespacesOp, pos))
	return expressionEffect{result: resultOp, block: curBB}
}

type xmlNamespaceDecl struct {
	key string
	uri string
}

func xmlNamespaceDecls(ctx context, refs []model.SymbolRef) []xmlNamespaceDecl {
	decls := make([]xmlNamespaceDecl, 0, len(refs))
	for _, ref := range refs {
		symbol := ctx.getSymbol(ref)
		key, err := model.XMLNamespaceDeclKey(symbol)
		if err != nil {
			ctx.internalError(err.Error(), diagnostics.Location{})
			continue
		}
		uri, err := model.XMLNamespaceURI(symbol)
		if err != nil {
			ctx.internalError(err.Error(), diagnostics.Location{})
			continue
		}
		decls = append(decls, xmlNamespaceDecl{key: key, uri: uri})
	}
	return decls
}

// buildXMLNamespacesMap constructs a string map of XML namespace declarations
// from an element's resolved namespace symbols. Iteration is sorted by emitted
// declaration key for deterministic output.
func buildXMLNamespacesMap(ctx context, curBB *BIRBasicBlock, ns []model.SymbolRef, pos Location) (*BIROperand, *BIRBasicBlock) {
	namespaces := xmlNamespaceDecls(ctx, ns)
	sort.SliceStable(namespaces, func(i, j int) bool {
		return namespaces[i].key < namespaces[j].key
	})
	entries := make([]MappingConstructorEntry, 0, len(namespaces))
	for _, ns := range namespaces {
		keyOp := ctx.addTempVar(semtypes.STRING)
		curBB.Instructions = append(curBB.Instructions, NewConstantLoad(keyOp, ns.key, pos))
		valOp := ctx.addTempVar(semtypes.STRING)
		curBB.Instructions = append(curBB.Instructions, NewConstantLoad(valOp, ns.uri, pos))
		entries = append(entries, &MappingConstructorKeyValueEntry{keyOp: keyOp, valueOp: valOp})
	}
	resultOp := ctx.addTempVar(ctx.function().birCx.stringMapType())
	curBB.Instructions = append(curBB.Instructions, NewMapConstructor(ctx.function().birCx.stringMapType(), resultOp, entries, nil, false, pos))
	return resultOp, curBB
}

func templateExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangTemplateExpr) expressionEffect {
	pos := ctx.function().loc(expr.GetPosition())
	operands := make([]*BIROperand, len(expr.Insertions))
	for i, ins := range expr.Insertions {
		eff := handleActionOrExpression(ctx, curBB, ins)
		curBB = eff.block
		operands[i] = eff.result
	}
	var kind TemplateKind
	switch expr.Kind {
	case ast.TemplateExprKindString:
		kind = TemplateKindString
	case ast.TemplateExprKindXML:
		kind = TemplateKindXML
	default:
		panic(fmt.Sprintf("unsupported template expr kind: %d", expr.Kind))
	}
	resultOp := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewEvalTemplateExpr(kind, expr.Strings, operands, resultOp, pos))
	return expressionEffect{result: resultOp, block: curBB}
}

func xmlSequenceLiteral(ctx context, curBB *BIRBasicBlock, expr *ast.BLangXMLSequenceLiteral) expressionEffect {
	if len(expr.Children) == 1 {
		return handleActionOrExpression(ctx, curBB, expr.Children[0])
	}
	pos := ctx.function().loc(expr.GetPosition())
	var childOps []*BIROperand
	for _, child := range expr.Children {
		eff := handleActionOrExpression(ctx, curBB, child)
		curBB = eff.block
		childOps = append(childOps, eff.result)
	}
	resultOp := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Instructions = append(curBB.Instructions, NewXMLSequenceInstr(resultOp, childOps, pos))
	return expressionEffect{result: resultOp, block: curBB}
}

type mappingField struct {
	key   string
	value ast.BLangExpression
}

func mappingConstructorExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangMappingConstructorExpr) expressionEffect {
	var fields []mappingField
	for _, field := range expr.Fields {
		switch f := field.(type) {
		case *ast.BLangMappingKeyValueField:
			keyName := mappingKeyName(f.Key)
			fields = append(fields, mappingField{key: keyName, value: f.ValueExpr})
		default:
			ctx.unimplemented("non-key-value record field not implemented", expr.GetPosition())
		}
	}
	var defaults []MappingConstructorDefaultEntry
	for _, fd := range expr.FieldDefaults {
		defaults = append(defaults, MappingConstructorDefaultEntry{
			FieldName:         fd.FieldName,
			FunctionLookupKey: buildFunctionLookupKeyFromSymbol(ctx.function().birCx, fd.FnRef),
		})
	}
	return mappingConstructorExpressionInner(ctx, curBB, expr.GetDeterminedType(), fields, defaults, ctx.function().loc(expr.GetPosition()))
}

func mappingKeyName(key *ast.BLangMappingKey) string {
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		return expr.Value.(string)
	case *ast.BLangSimpleVarRef:
		return expr.VariableName.Value
	default:
		panic(fmt.Sprintf("unexpected mapping key expression type: %T", key.Expr))
	}
}

func mappingConstructorExpressionInner(ctx context, curBB *BIRBasicBlock, mapType semtypes.SemType, fields []mappingField, defaults []MappingConstructorDefaultEntry, pos Location) expressionEffect {
	var entries []MappingConstructorEntry
	for _, field := range fields {
		keyOperand := ctx.addTempVar(semtypes.STRING)
		keyLoad := NewConstantLoad(keyOperand, field.key, pos)
		curBB.Instructions = append(curBB.Instructions, keyLoad)

		valueEffect := handleActionOrExpression(ctx, curBB, field.value)
		curBB = valueEffect.block
		entries = append(entries, &MappingConstructorKeyValueEntry{
			keyOp:   keyOperand,
			valueOp: valueEffect.result,
		})
	}
	resultOperand := ctx.addTempVar(mapType)
	isReadonly := semtypes.IsSubtype(ctx.function().birCx.typeCtx, mapType, semtypes.VAL_READONLY)
	newMap := NewMapConstructor(mapType, resultOperand, entries, defaults, isReadonly, pos)
	curBB.Instructions = append(curBB.Instructions, newMap)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func errorConstructorExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangErrorConstructorExpr) expressionEffect {
	// Message is the first positional arg
	msgEffect := handleActionOrExpression(ctx, curBB, expr.PositionalArgs[0])
	curBB = msgEffect.block

	// Cause is the optional second positional arg
	var causeOp *BIROperand
	if len(expr.PositionalArgs) > 1 {
		causeEffect := handleActionOrExpression(ctx, curBB, expr.PositionalArgs[1])
		curBB = causeEffect.block
		causeOp = causeEffect.result
	}

	// Detail from named args
	var detailOp *BIROperand
	if len(expr.NamedArgs) > 0 {
		var fields []mappingField
		for _, namedArg := range expr.NamedArgs {
			fields = append(fields, mappingField{key: namedArg.Name.Value, value: namedArg.Expr})
		}
		detailEffect := mappingConstructorExpressionInner(ctx, curBB, semtypes.MAPPING, fields, nil, ctx.function().loc(expr.GetPosition()))
		curBB = detailEffect.block
		detailOp = detailEffect.result
	}

	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	typeName := ""
	if expr.ErrorTypeRef != nil {
		typeName = expr.ErrorTypeRef.TypeName.Value
	}
	newError := NewErrorConstructor(expr.GetDeterminedType(), typeName, resultOperand, msgEffect.result, causeOp, detailOp, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, newError)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func typeConversionExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangTypeConversionExpr) expressionEffect {
	exprEffect := handleActionOrExpression(ctx, curBB, expr.Expression)
	curBB = exprEffect.block
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	typeCast := NewTypeCast(expr.TypeDescriptor.GetDeterminedType(), resultOperand, exprEffect.result, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, typeCast)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func typeTestExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangTypeTestExpr) expressionEffect {
	exprEffect := handleActionOrExpression(ctx, curBB, expr.Expr)
	curBB = exprEffect.block
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	typeTest := &TypeTest{}
	typeTest.Pos = ctx.function().loc(expr.GetPosition())
	typeTest.LhsOp = resultOperand
	typeTest.RhsOp = exprEffect.result
	typeTest.Type = expr.Type.Type
	typeTest.IsNegation = expr.IsNegation()
	curBB.Instructions = append(curBB.Instructions, typeTest)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

// materializeFiller emits BIR instructions that construct a fresh filler
// value for ty at runtime.
func materializeFiller(ctx context, bb *BIRBasicBlock, ty semtypes.SemType, f semtypes.Filler, pos Location) (*BIROperand, *BIRBasicBlock) {
	tyCx := ctx.function().birCx.typeCtx
	switch f := f.(type) {
	case semtypes.SingleValueFiller:
		operand := ctx.addTempVar(ty)
		bb.Instructions = append(bb.Instructions, NewConstantLoad(operand, f.Value, pos))
		return operand, bb
	case semtypes.MappingFiller:
		operand := ctx.addTempVar(f.Type)
		mapReadonly := semtypes.IsSubtype(tyCx, f.Type, semtypes.VAL_READONLY)
		bb.Instructions = append(bb.Instructions, NewMapConstructor(f.Type, operand, nil, nil, mapReadonly, pos))
		return operand, bb
	case semtypes.ListFiller:
		memberOperands := make([]*BIROperand, len(f.Members))
		for i, memberFiller := range f.Members {
			memberOperands[i], bb = materializeFiller(ctx, bb, f.Atomic.MemberAtInnerVal(i), memberFiller, pos)
		}
		sizeOperand := ctx.addTempVar(semtypes.INT)
		bb.Instructions = append(bb.Instructions, NewConstantLoad(sizeOperand, int64(len(memberOperands)), pos))
		restFiller, _ := values.FillerFactoryFor(tyCx, f.Atomic.Rest())
		operand := ctx.addTempVar(f.Type)
		listReadonly := semtypes.IsSubtype(tyCx, f.Type, semtypes.VAL_READONLY)
		bb.Instructions = append(bb.Instructions, NewArrayConstructor(f.Type, operand, sizeOperand, memberOperands, restFiller, listReadonly, pos))
		return operand, bb
	default:
		panic(fmt.Sprintf("unsupported filler kind %T in BIR generation", f))
	}
}

func listConstructorExpression(ctx context, bb *BIRBasicBlock, expr *ast.BLangListConstructorExpr) expressionEffect {
	initValues := make([]*BIROperand, len(expr.Exprs))
	for i, expr := range expr.Exprs {
		exprEffect := handleActionOrExpression(ctx, bb, expr)
		bb = exprEffect.block
		initValues[i] = exprEffect.result
	}

	lat := expr.AtomicType
	exprPos := ctx.function().loc(expr.GetPosition())
	tyCx := ctx.function().birCx.typeCtx
	for i := len(expr.Exprs); i < lat.Members.FixedLength; i++ {
		ty := lat.MemberAtInnerVal(i)
		filler, ok := semtypes.FillerValue(tyCx, ty)
		if !ok {
			ctx.internalError("no filler value for list member type; semantic analysis should have rejected this", expr.GetPosition())
		}
		var fillerOperand *BIROperand
		fillerOperand, bb = materializeFiller(ctx, bb, ty, filler, exprPos)
		initValues = append(initValues, fillerOperand)
	}
	restFiller, _ := values.FillerFactoryFor(tyCx, lat.Rest())

	sizeOperand := ctx.addTempVar(semtypes.INT)
	constantLoad := NewConstantLoad(sizeOperand, int64(len(initValues)), exprPos)
	bb.Instructions = append(bb.Instructions, constantLoad)

	resultOperand := ctx.addTempVar(semtypes.LIST)
	listTy := expr.GetDeterminedType()
	isReadonly := semtypes.IsSubtype(tyCx, listTy, semtypes.VAL_READONLY)
	newArray := NewArrayConstructor(listTy, resultOperand, sizeOperand, initValues, restFiller, isReadonly, exprPos)
	bb.Instructions = append(bb.Instructions, newArray)
	return expressionEffect{
		result: resultOperand,
		block:  bb,
	}
}

// assignmentContainerReference produces the container reference for an indexed assignment LHS.
// When the container is itself an index-based access on a list or mapping, the inner read
// must be a filling load so that intermediate arrays grow (and fill) and absent map keys
// are populated with a filler value before storing.
func assignmentContainerReference(ctx context, bb *BIRBasicBlock, expr ast.BLangExpression) expressionEffect {
	inner, ok := expr.(*ast.BLangIndexBasedAccess)
	if !ok {
		return handleActionOrExpression(ctx, bb, expr)
	}
	// The container of an indexed lvalue access can show up as a nilable type
	// when it itself comes from another map index (e.g. `m["a"]["b"]` where the
	// inner lookup nominally yields `T?`). After filling, the container is
	// guaranteed non-nil, so we strip `()` before classifying.
	containerType := semtypes.Diff(inner.Expr.GetDeterminedType(), semtypes.NIL)
	tyCtx := ctx.function().birCx.TypeContext()
	var fillingKind InstructionKind
	var filler values.FillerFactory
	switch {
	case semtypes.IsSubtype(tyCtx, containerType, semtypes.LIST):
		fillingKind = INSTRUCTION_KIND_ARRAY_FILLING_LOAD
	case semtypes.IsSubtype(tyCtx, containerType, semtypes.MAPPING):
		fillingKind = INSTRUCTION_KIND_MAP_FILLING_LOAD
		tyCx := semtypes.TypeCheckContext(ctx.typeEnv())
		valueType := semtypes.MappingMemberTypeInnerVal(tyCx, containerType, semtypes.STRING)
		filler, _ = values.FillerFactoryFor(tyCx, valueType)
	default:
		return handleActionOrExpression(ctx, bb, expr)
	}
	resultOperand := ctx.addTempVar(inner.GetDeterminedType())
	indexEffect := handleActionOrExpression(ctx, bb, inner.IndexExpr)
	containerRefEffect := assignmentContainerReference(ctx, indexEffect.block, inner.Expr)
	fieldAccess := NewFieldAccess(fillingKind, resultOperand, indexEffect.result, containerRefEffect.result, ctx.function().loc(inner.GetPosition()))
	fieldAccess.Filler = filler
	containerRefEffect.block.Instructions = append(containerRefEffect.block.Instructions, fieldAccess)
	return expressionEffect{
		result: resultOperand,
		block:  containerRefEffect.block,
	}
}

func indexBasedAccess(ctx context, bb *BIRBasicBlock, expr *ast.BLangIndexBasedAccess) expressionEffect {
	// Assignment is handled in assignmentStatement to this is always a load
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	loadKind, _ := memberAccessInstructionKinds(ctx.function().birCx.TypeContext(), expr.Expr.GetDeterminedType())
	indexEffect := handleActionOrExpression(ctx, bb, expr.IndexExpr)
	containerRefEffect := handleActionOrExpression(ctx, indexEffect.block, expr.Expr)
	currBB := containerRefEffect.block
	fieldAccess := NewFieldAccess(loadKind, resultOperand, indexEffect.result, containerRefEffect.result, ctx.function().loc(expr.GetPosition()))
	currBB.Instructions = append(currBB.Instructions, fieldAccess)
	return expressionEffect{
		result: resultOperand,
		block:  currBB,
	}
}

func groupExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangGroupExpr) expressionEffect {
	return handleActionOrExpression(ctx, curBB, expr.Expression)
}

func wildcardBindingPattern(ctx context, curBB *BIRBasicBlock, expr *ast.BLangWildCardBindingPattern) expressionEffect {
	return expressionEffect{
		result: ctx.addTempVar(semtypes.NEVER),
		block:  curBB,
	}
}

func unaryExpression(ctx context, bb *BIRBasicBlock, expr *ast.BLangUnaryExpr) expressionEffect {
	var kind InstructionKind
	switch expr.Operator {
	case model.OperatorKind_NOT:
		kind = INSTRUCTION_KIND_NOT
	case model.OperatorKind_SUB:
		kind = INSTRUCTION_KIND_NEGATE
	case model.OperatorKind_BITWISE_COMPLEMENT:
		kind = INSTRUCTION_KIND_BITWISE_COMPLEMENT
	default:
		panic("unexpected unary operator kind")
	}
	opEffect := handleActionOrExpression(ctx, bb, expr.Expr)

	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	curBB := opEffect.block
	unaryOp := NewUnaryOp(kind, resultOperand, opEffect.result, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, unaryOp)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

type callable interface {
	ast.BLangActionOrExpression
	ResolvedSymbol() model.SymbolRef
	Receiver() ast.BLangExpression
	CallArgs() []ast.BLangExpression
	GetName() ast.IdentifierNode
}

func generateResourceAccessCall(ctx context, bb *BIRBasicBlock, expr *ast.BLangClientResourceAccessAction) expressionEffect {
	curBB := bb
	recvEffect := handleActionOrExpression(ctx, curBB, expr.Expr)
	curBB = recvEffect.block
	// this should always result in a value
	receiver := *recvEffect.result
	pos := ctx.function().loc(expr.GetPosition())
	var pathSegments []BIROperand
	for i := range expr.Path {
		seg := &expr.Path[i]
		switch seg.Kind {
		case ast.ResourceAccessSegmentName:
			temp := ctx.addTempVar(semtypes.StringConst(seg.Name))
			curBB.Instructions = append(curBB.Instructions, NewConstantLoad(temp, seg.Name, pos))
			pathSegments = append(pathSegments, *temp)
		case ast.ResourceAccessSegmentComputed:
			effect := handleActionOrExpression(ctx, curBB, seg.Expr)
			effect = snapshotIfNeeded(ctx, effect, pos)
			curBB = effect.block
			pathSegments = append(pathSegments, *effect.result)
		}
	}
	var args []BIROperand
	for _, arg := range expr.ArgExprs {
		effect := handleActionOrExpression(ctx, curBB, arg)
		effect = snapshotIfNeeded(ctx, effect, pos)
		curBB = effect.block
		args = append(args, *effect.result)
	}
	thenBB := ctx.function().addBB()
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	curBB.Terminator = NewResourceFunctionCall(receiver, expr.MethodName, pathSegments, args, thenBB, resultOperand, pos)
	return expressionEffect{result: resultOperand, block: thenBB}
}

func generateCall(ctx context, bb *BIRBasicBlock, callable callable) expressionEffect {
	curBB := bb
	if ast.IsStreamOperation(callable) {
		return streamMethodCall(ctx, curBB, callable)
	}
	var args []BIROperand
	isMethodCall := false

	if callable.Receiver() != nil {
		effect := handleActionOrExpression(ctx, curBB, callable.Receiver())
		curBB = effect.block
		args = append(args, *effect.result)
		isMethodCall = true
	}

	for _, arg := range callable.CallArgs() {
		effect := handleActionOrExpression(ctx, curBB, arg)
		effect = snapshotIfNeeded(ctx, effect, ctx.function().loc(callable.GetPosition()))
		curBB = effect.block
		args = append(args, *effect.result)
	}

	thenBB := ctx.function().addBB()
	resultOperand := ctx.addTempVar(callable.GetDeterminedType())
	callName := callable.GetName().GetValue()
	if _, isRemote := callable.(*ast.BLangRemoteMethodCallAction); isRemote {
		callName = model.RemoteMethodName(callName)
	}
	call := NewCall(INSTRUCTION_KIND_CALL, args, model.Name(callName), thenBB, resultOperand, ctx.function().loc(callable.GetPosition()))
	call.IsMethodCall = isMethodCall

	if !isMethodCall {
		symRef := callable.ResolvedSymbol()
		sym := ctx.getSymbol(symRef)
		if sym.Kind() == model.SymbolKindFunction {
			call.FunctionLookupKey = buildFunctionLookupKeyFromSymbol(ctx.function().birCx, symRef)
			call.CalleePkg = packageIDFromIdentifier(ctx.compilerContext(), ctx.symbolPackage(symRef))
		} else {
			call.Kind = INSTRUCTION_KIND_FP_CALL
			unnarrowedRef := ctx.unnarrowedSymbol(symRef)
			if op, crossedFunction, ok := lookupVar(ctx, unnarrowedRef); ok {
				call.FpOperand = op
				ctx.function().isClosure = ctx.function().isClosure || crossedFunction
			}
		}
	}
	curBB.Terminator = call
	return expressionEffect{
		result: resultOperand,
		block:  thenBB,
	}
}

func literal(ctx context, curBB *BIRBasicBlock, expr *ast.BLangLiteral) expressionEffect {
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())
	constantLoad := NewConstantLoad(resultOperand, expr.Value, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, constantLoad)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func operatorKindToBinaryInstructionKind(opKind model.OperatorKind) InstructionKind {
	switch opKind {
	case model.OperatorKind_ADD:
		return INSTRUCTION_KIND_ADD
	case model.OperatorKind_SUB:
		return INSTRUCTION_KIND_SUB
	case model.OperatorKind_MUL:
		return INSTRUCTION_KIND_MUL
	case model.OperatorKind_DIV:
		return INSTRUCTION_KIND_DIV
	case model.OperatorKind_MOD:
		return INSTRUCTION_KIND_MOD
	case model.OperatorKind_EQUAL:
		return INSTRUCTION_KIND_EQUAL
	case model.OperatorKind_NOT_EQUAL:
		return INSTRUCTION_KIND_NOT_EQUAL
	case model.OperatorKind_GREATER_THAN:
		return INSTRUCTION_KIND_GREATER_THAN
	case model.OperatorKind_GREATER_EQUAL:
		return INSTRUCTION_KIND_GREATER_EQUAL
	case model.OperatorKind_LESS_THAN:
		return INSTRUCTION_KIND_LESS_THAN
	case model.OperatorKind_LESS_EQUAL:
		return INSTRUCTION_KIND_LESS_EQUAL
	case model.OperatorKind_REF_EQUAL:
		return INSTRUCTION_KIND_REF_EQUAL
	case model.OperatorKind_REF_NOT_EQUAL:
		return INSTRUCTION_KIND_REF_NOT_EQUAL
	case model.OperatorKind_BITWISE_AND:
		return INSTRUCTION_KIND_BITWISE_AND
	case model.OperatorKind_BITWISE_OR:
		return INSTRUCTION_KIND_BITWISE_OR
	case model.OperatorKind_BITWISE_XOR:
		return INSTRUCTION_KIND_BITWISE_XOR
	case model.OperatorKind_BITWISE_LEFT_SHIFT:
		return INSTRUCTION_KIND_BITWISE_LEFT_SHIFT
	case model.OperatorKind_BITWISE_RIGHT_SHIFT:
		return INSTRUCTION_KIND_BITWISE_RIGHT_SHIFT
	case model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return INSTRUCTION_KIND_BITWISE_UNSIGNED_RIGHT_SHIFT
	default:
		panic("unexpected binary operator kind")
	}
}

func binaryExpressionInner(ctx context, curBB *BIRBasicBlock, opKind model.OperatorKind, lhsExpr ast.BLangExpression, rhsExpr ast.BLangActionOrExpression, resultType semtypes.SemType, pos Location) expressionEffect {
	kind := operatorKindToBinaryInstructionKind(opKind)
	resultOperand := ctx.addTempVar(resultType)
	op1Effect := handleActionOrExpression(ctx, curBB, lhsExpr)
	op1Effect = snapshotIfNeeded(ctx, op1Effect, pos)
	curBB = op1Effect.block
	op2Effect := handleActionOrExpression(ctx, curBB, rhsExpr)
	op2Effect = snapshotIfNeeded(ctx, op2Effect, pos)
	curBB = op2Effect.block
	binaryOp := NewBinaryOp(kind, resultOperand, op1Effect.result, op2Effect.result, pos)
	curBB.Instructions = append(curBB.Instructions, binaryOp)
	return expressionEffect{
		result: resultOperand,
		block:  curBB,
	}
}

func binaryExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangBinaryExpr) expressionEffect {
	switch expr.OpKind {
	case model.OperatorKind_AND:
		return logicalAndExpression(ctx, curBB, expr)
	case model.OperatorKind_OR:
		return logicalOrExpression(ctx, curBB, expr)
	default:
		return binaryExpressionInner(ctx, curBB, expr.OpKind, expr.LhsExpr, expr.RhsExpr, expr.GetDeterminedType(), ctx.function().loc(expr.GetPosition()))
	}
}

func logicalAndExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangBinaryExpr) expressionEffect {
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())

	lhsEffect := handleActionOrExpression(ctx, curBB, expr.LhsExpr)
	curBB = lhsEffect.block

	mov := NewMove(lhsEffect.result, resultOperand, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, mov)

	evalRhsBB := ctx.function().addBB()
	doneBB := ctx.function().addBB()

	curBB.Terminator = NewBranch(lhsEffect.result, evalRhsBB, doneBB, ctx.function().loc(expr.GetPosition()))

	rhsEffect := handleActionOrExpression(ctx, evalRhsBB, expr.RhsExpr)
	rhsBB := rhsEffect.block

	rhsMov := NewMove(rhsEffect.result, resultOperand, ctx.function().loc(expr.GetPosition()))

	rhsBB.Instructions = append(rhsBB.Instructions, rhsMov)
	rhsBB.Terminator = NewGoto(doneBB, ctx.function().loc(expr.GetPosition()))

	return expressionEffect{
		result: resultOperand,
		block:  doneBB,
	}
}

func logicalOrExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangBinaryExpr) expressionEffect {
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())

	lhsEffect := handleActionOrExpression(ctx, curBB, expr.LhsExpr)
	curBB = lhsEffect.block

	mov := NewMove(lhsEffect.result, resultOperand, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, mov)

	evalRhsBB := ctx.function().addBB()
	doneBB := ctx.function().addBB()

	curBB.Terminator = NewBranch(lhsEffect.result, doneBB, evalRhsBB, ctx.function().loc(expr.GetPosition()))

	rhsEffect := handleActionOrExpression(ctx, evalRhsBB, expr.RhsExpr)
	rhsBB := rhsEffect.block

	rhsMov := NewMove(rhsEffect.result, resultOperand, ctx.function().loc(expr.GetPosition()))
	rhsBB.Instructions = append(rhsBB.Instructions, rhsMov)
	rhsBB.Terminator = NewGoto(doneBB, ctx.function().loc(expr.GetPosition()))

	return expressionEffect{
		result: resultOperand,
		block:  doneBB,
	}
}

func simpleVariableReference(ctx context, curBB *BIRBasicBlock, expr *ast.BLangSimpleVarRef) expressionEffect {
	varName := expr.VariableName.GetValue()
	symRef := ctx.unnarrowedSymbol(expr.Symbol())

	if operand, crossedFunction, ok := lookupVar(ctx, symRef); ok {
		ctx.function().isClosure = ctx.function().isClosure || crossedFunction
		return expressionEffect{
			result: operand,
			block:  curBB,
		}
	}

	// Try function lookup
	sym := ctx.getSymbol(symRef)
	if sym.Kind() == model.SymbolKindType {
		resultOperand := ctx.addTempVar(expr.GetDeterminedType())
		td := values.NewTypeDesc(ctx.symbolType(symRef), ctx.compilerContext().SymbolAnnotationValues(symRef))
		curBB.Instructions = append(curBB.Instructions, NewConstantLoad(resultOperand, td, ctx.function().loc(expr.GetPosition())))
		return expressionEffect{
			result: resultOperand,
			block:  curBB,
		}
	}
	if sym.Kind() == model.SymbolKindFunction {
		funcType := ctx.symbolType(symRef)
		lookupKey := buildFunctionLookupKeyFromSymbol(ctx.function().birCx, symRef)
		resultOperand := ctx.addTempVar(funcType)
		fpLoad := NewFPLoad(lookupKey, funcType, resultOperand, ctx.function().loc(expr.GetPosition()))
		curBB.Instructions = append(curBB.Instructions, fpLoad)
		return expressionEffect{
			result: resultOperand,
			block:  curBB,
		}
	}
	if sym.Kind() == model.SymbolKindConstant {
		ctx.internalError("constant reference was not inlined during desugar", expr.GetPosition())
	}

	// Global variable reference
	pkgId := packageIDFromIdentifier(ctx.compilerContext(), ctx.symbolPackage(symRef))
	gv := &BIRGlobalVariableDcl{}
	gv.Name = model.Name(varName)
	gv.PkgId = pkgId
	gv.GlobalVarLookupKey = buildGlobalVarLookupKey(pkgId, gv.Name)
	return expressionEffect{
		result: &BIROperand{VariableDcl: gv},
		block:  curBB,
	}
}

func trapExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangTrapExpr) expressionEffect {
	resultOperand := ctx.addTempVar(expr.GetDeterminedType())

	trapStartBB := ctx.function().addBB()
	curBB.Terminator = NewGoto(trapStartBB, ctx.function().loc(expr.GetPosition()))

	innerEffect := handleActionOrExpression(ctx, trapStartBB, expr.Expr)
	trapEndBB := innerEffect.block

	mov := NewMove(innerEffect.result, resultOperand, ctx.function().loc(expr.GetPosition()))
	trapEndBB.Instructions = append(trapEndBB.Instructions, mov)

	afterTrapBB := ctx.function().addBB()
	trapEndBB.Terminator = NewGoto(afterTrapBB, ctx.function().loc(expr.GetPosition()))

	fn := ctx.function()
	fn.errorEntries = append(fn.errorEntries, BIRErrorEntry{
		Start:   trapStartBB.Number,
		End:     trapEndBB.Number,
		Target:  afterTrapBB.Number,
		ErrorOp: resultOperand,
	})

	return expressionEffect{
		result: resultOperand,
		block:  afterTrapBB,
	}
}

func transformClassDefinition(ctx *Context, class *ast.BLangClassDefinition, birPkg *BIRPackage) {
	className := class.GetName().GetValue()
	classLookupKey := buildLookupKey(ctx.CompilerContext.SymbolPackage(class.Symbol()), ctx.CompilerContext.SymbolName(class.Symbol()))
	methodLookupKey := func(methodName string, symRef model.SymbolRef) string {
		return buildMethodLookupKeyFromSymbol(ctx, className, symRef)
	}
	resourceLookupKey := func(rm *ast.BLangResourceMethod) string {
		return buildFunctionLookupKeyFromSymbol(ctx, rm.Symbol())
	}
	birClassDef := transformClassBody(ctx, class.Scope(), classLookupKey, model.Name(className), class.Fields, class.InitFunction, class.Methods, class.ResourceMethods, methodLookupKey, resourceLookupKey, class.GetPosition())
	birPkg.ClassDefs = append(birPkg.ClassDefs, *birClassDef)
}

func transformService(ctx *Context, svc *ast.BLangService, idx int, birPkg *BIRPackage) {
	className := fmt.Sprintf("$service$%d", idx)
	pkg := model.PackageIdentifierFromID(ctx.packageID)
	classLookupKey := buildLookupKey(pkg, className)
	ctx.serviceClassKeys[svc] = classLookupKey
	methodLookupKey := func(methodName string, _ model.SymbolRef) string {
		return buildLookupKey(pkg, className+"."+methodName)
	}
	resourceLookupKey := func(rm *ast.BLangResourceMethod) string {
		sym := ctx.CompilerContext.GetSymbol(rm.Symbol())
		return buildLookupKey(pkg, className+"."+sym.Name())
	}
	birClassDef := transformClassBody(ctx, svc.Scope(), classLookupKey, model.Name(className), svc.Fields, svc.InitFunction, svc.Methods, svc.ResourceMethods, methodLookupKey, resourceLookupKey, svc.GetPosition())
	birPkg.ClassDefs = append(birPkg.ClassDefs, *birClassDef)
}

func transformClassBody(
	ctx *Context,
	classScope model.Scope,
	classLookupKey string,
	className model.Name,
	fields []ast.SimpleVariableNode,
	initFn *ast.BLangFunction,
	methods map[string]*ast.BLangFunction,
	resourceMethods []*ast.BLangResourceMethod,
	methodLookupKey func(string, model.SymbolRef) string,
	resourceLookupKey func(*ast.BLangResourceMethod) string,
	pos diagnostics.Location,
) *BIRClassDef {
	selfRef, ok := classScope.GetSymbol("self")
	if !ok {
		ctx.CompilerContext.InternalError("self symbol not found in class scope", pos)
	}

	birClassDef := &BIRClassDef{
		Name:      className,
		LookupKey: classLookupKey,
		VTable:    make(map[string]*BIRFunction),
		RTable:    make(map[string][]BIRResourceMethod),
	}

	for _, field := range fields {
		birClassDef.Fields = append(birClassDef.Fields, ObjectField{
			Name: field.GetName().GetValue(),
			Ty:   ctx.CompilerContext.SymbolType(field.Symbol()),
		})
	}

	initFunc := transformFunctionInner(newFunctionRoot(ctx, nil), initFn, &selfRef)
	initFunc.FunctionLookupKey = methodLookupKey("init", initFn.Symbol())
	birClassDef.VTable["init"] = initFunc

	for methodName, method := range methods {
		lookupKey := methodLookupKey(methodName, method.Symbol())
		var fn *BIRFunction
		if method.IsNative() {
			fn = &BIRFunction{
				Name:              model.Name(method.GetName().GetValue()),
				OriginalName:      model.Name(method.GetName().GetValue()),
				Flags:             method.Flags(),
				FunctionLookupKey: lookupKey,
			}
			fn.Pos = birLoc(ctx.CompilerContext.DiagnosticEnv(), method.GetPosition())
		} else {
			fn = transformFunctionInner(newFunctionRoot(ctx, nil), method, &selfRef)
			fn.FunctionLookupKey = lookupKey
		}
		birClassDef.VTable[methodName] = fn
	}

	for _, rm := range resourceMethods {
		lookupKey := resourceLookupKey(rm)
		var fn *BIRFunction
		if rm.IsNative() {
			fn = &BIRFunction{
				Name:              model.Name(ctx.CompilerContext.SymbolName(rm.Symbol())),
				OriginalName:      model.Name(rm.GetName().GetValue()),
				Flags:             rm.Flags(),
				FunctionLookupKey: lookupKey,
			}
			fn.Pos = birLoc(ctx.CompilerContext.DiagnosticEnv(), rm.GetPosition())
		} else {
			fn = transformResourceMethodInner(newFunctionRoot(ctx, nil), rm, &selfRef)
			fn.FunctionLookupKey = lookupKey
		}
		methodName := rm.GetName().GetValue()
		entry := buildResourceMethodEntry(ctx, rm, fn)
		birClassDef.RTable[methodName] = append(birClassDef.RTable[methodName], entry)
	}

	return birClassDef
}

func buildResourceMethodEntry(ctx *Context, rm *ast.BLangResourceMethod, fn *BIRFunction) BIRResourceMethod {
	var pathSegments []ResourcePathSegmentDef
	var restTy = semtypes.NEVER
	for i := range rm.ResourcePath {
		seg := &rm.ResourcePath[i]
		segTy := seg.GetDeterminedType()
		if seg.Kind == ast.ResourcePathSegmentParamRest {
			restTy = segTy
		} else {
			pathSegments = append(pathSegments, ResourcePathSegmentDef{Ty: segTy})
		}
	}
	return BIRResourceMethod{
		PathSegments:  pathSegments,
		RestSegmentTy: restTy,
		Fn:            fn,
	}
}

func transformResourceMethodInner(root *funcBlock, rm *ast.BLangResourceMethod, selfSymbolRef *model.SymbolRef) *BIRFunction {
	symRef := rm.Symbol()
	ctx := root.fn.birCx
	funcName := model.Name(ctx.CompilerContext.SymbolName(symRef))
	birFunc := &BIRFunction{}
	birFunc.Pos = root.fn.loc(rm.GetPosition())
	birFunc.Name = funcName
	birFunc.OriginalName = funcName
	birFunc.Flags = rm.Flags()
	birFunc.FunctionLookupKey = buildFunctionLookupKeyFromSymbol(ctx, symRef)
	funcSym := ctx.CompilerContext.GetSymbol(symRef).(model.FunctionSymbol)
	retOp := root.addLocalVarInner(model.Name("%0"), funcSym.Signature().ReturnType)
	root.fn.retVarDcl = retOp.VariableDcl.(*BIRLocalVariableDcl)
	if selfSymbolRef != nil {
		root.addLocalVar(model.Name("self"), ctx.CompilerContext.SymbolType(*selfSymbolRef), *selfSymbolRef)
	}
	var requiredParams []BIRParameter
	for i := range rm.ResourcePath {
		seg := &rm.ResourcePath[i]
		if seg.Kind == ast.ResourcePathSegmentName || seg.Name == "" {
			continue
		}
		name := seg.Name
		ref, ok := rm.Scope().GetSymbol(name)
		if !ok {
			continue
		}
		root.addLocalVar(model.Name(name), ctx.CompilerContext.SymbolType(ref), ref)
		requiredParams = append(requiredParams, BIRParameter{Name: model.Name(name)})
	}
	for i := range rm.RequiredParams {
		param := &rm.RequiredParams[i]
		root.addLocalVar(model.Name(param.GetName().GetValue()), ctx.CompilerContext.SymbolType(param.Symbol()), param.Symbol())
		requiredParams = append(requiredParams, BIRParameter{
			Name:  model.Name(param.GetName().GetValue()),
			Flags: param.Flags(),
		})
	}
	if rm.RestParam != nil {
		restParam := rm.RestParam
		ty := ctx.CompilerContext.SymbolType(restParam.Symbol())
		root.addLocalVar(model.Name(restParam.GetName().GetValue()), ty, restParam.Symbol())
		birFunc.RestParams = &BIRParameter{Name: model.Name(restParam.GetName().GetValue())}
	}
	birFunc.RequiredParams = requiredParams
	switch body := rm.Body.(type) {
	case *ast.BLangBlockFunctionBody:
		handleBlockFunctionBody(root, body)
	case *ast.BLangExprFunctionBody:
		handleExprFunctionBody(root, body)
	default:
		panic("unexpected function body type")
	}
	for _, bbPtr := range root.fn.bbs {
		birFunc.BasicBlocks = append(birFunc.BasicBlocks, *bbPtr)
	}
	for _, varPtr := range root.localVars {
		birFunc.LocalVars = append(birFunc.LocalVars, *varPtr)
	}
	birFunc.ErrorTable = root.fn.errorEntries
	birFunc.ReturnVariable = root.fn.retVarDcl
	return birFunc
}

func newExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangNewExpression) expressionEffect {
	if semtypes.IsSubtypeSimple(expr.GetDeterminedType(), semtypes.STREAM) {
		return newStreamExpression(ctx, curBB, expr)
	}
	classSymbol := expr.ClassSymbol
	className := ctx.symbolName(classSymbol)
	classLookupKey := buildLookupKey(ctx.symbolPackage(classSymbol), className)
	objectTy := semtypes.Diff(expr.GetDeterminedType(), semtypes.ERROR)
	return emitObjectInit(ctx, curBB, classLookupKey, objectTy, expr.GetDeterminedType(), expr.ArgsExprs, expr.GetPosition())
}

func serviceInitExpression(ctx context, curBB *BIRBasicBlock, expr *desugar.BLangServiceInit) expressionEffect {
	classLookupKey, ok := ctx.function().birCx.serviceClassKeys[expr.Service]
	if !ok {
		// We should have set this when going over the service decl at the begining
		ctx.function().birCx.CompilerContext.InternalError("service class not registered", expr.GetPosition())
	}
	objectTy := semtypes.Diff(expr.GetDeterminedType(), semtypes.ERROR)
	return emitObjectInit(ctx, curBB, classLookupKey, objectTy, expr.GetDeterminedType(), nil, expr.GetPosition())
}

func emitObjectInit(ctx context, curBB *BIRBasicBlock, classLookupKey string, objectTy semtypes.SemType, resultTy semtypes.SemType, argExprs []ast.BLangExpression, pos diagnostics.Location) expressionEffect {
	object := ctx.addTempVar(objectTy)
	newObj := NewObjectConstructor(classLookupKey, object, ctx.function().loc(pos))
	curBB.Instructions = append(curBB.Instructions, newObj)

	var args []BIROperand
	args = append(args, *object)
	for _, arg := range argExprs {
		argEffect := handleActionOrExpression(ctx, curBB, arg)
		curBB = argEffect.block
		args = append(args, *argEffect.result)
	}

	initMethodLookupKey := classLookupKey + ".init"
	initResult := ctx.addTempVar(semtypes.Union(semtypes.NIL, semtypes.ERROR))
	initDoneBB := ctx.function().addBB()
	call := NewCall(INSTRUCTION_KIND_CALL, args, model.Name("init"), initDoneBB, initResult, ctx.function().loc(pos))
	call.IsMethodCall = true
	call.CachedMethodLookupKey = initMethodLookupKey
	curBB.Terminator = call

	result := ctx.addTempVar(resultTy)
	isInitResultNil := ctx.addTempVar(semtypes.BOOLEAN)
	nilCheck := NewTypeTest(semtypes.NIL, isInitResultNil, initResult, ctx.function().loc(pos))
	initDoneBB.Instructions = append(initDoneBB.Instructions, nilCheck)

	assignObjectBB := ctx.function().addBB()
	assignErrorBB := ctx.function().addBB()
	thenBB := ctx.function().addBB()
	initDoneBB.Terminator = NewBranch(isInitResultNil, assignObjectBB, assignErrorBB, ctx.function().loc(pos))

	assignObjectBB.Instructions = append(assignObjectBB.Instructions, NewMove(object, result, ctx.function().loc(pos)))
	assignObjectBB.Terminator = NewGoto(thenBB, ctx.function().loc(pos))

	assignErrorBB.Instructions = append(assignErrorBB.Instructions, NewMove(initResult, result, ctx.function().loc(pos)))
	assignErrorBB.Terminator = NewGoto(thenBB, ctx.function().loc(pos))

	return expressionEffect{
		result: result,
		block:  thenBB,
	}
}

func streamMethodCall(ctx context, curBB *BIRBasicBlock, callable callable) expressionEffect {
	recvEffect := handleActionOrExpression(ctx, curBB, callable.Receiver())
	curBB = recvEffect.block
	result := ctx.addTempVar(callable.GetDeterminedType())
	pos := ctx.function().loc(callable.GetPosition())
	switch callable.GetName().GetValue() {
	case "next":
		curBB.Instructions = append(curBB.Instructions, NewStreamNext(result, recvEffect.result, pos))
	case "close":
		curBB.Instructions = append(curBB.Instructions, NewStreamClose(result, recvEffect.result, pos))
	default:
		ctx.internalError("unexpected stream method: "+callable.GetName().GetValue(), callable.GetPosition())
	}
	return expressionEffect{result: result, block: curBB}
}

func newStreamExpression(ctx context, curBB *BIRBasicBlock, expr *ast.BLangNewExpression) expressionEffect {
	argEffect := handleActionOrExpression(ctx, curBB, expr.ArgsExprs[0])
	curBB = argEffect.block
	result := ctx.addTempVar(expr.GetDeterminedType())
	instr := NewStreamConstructor(expr.GetDeterminedType(), result, argEffect.result, ctx.function().loc(expr.GetPosition()))
	curBB.Instructions = append(curBB.Instructions, instr)
	return expressionEffect{result: result, block: curBB}
}

func appendIfNotNil[T any](slice []T, item *T) []T {
	if item != nil {
		slice = append(slice, *item)
	}
	return slice
}

func hasNoStorageIdentity(tyCtx semtypes.Context, ty semtypes.SemType) bool {
	return semtypes.IsSubtype(tyCtx, ty, semtypes.SIMPLE_BASIC)
}
