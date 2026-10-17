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

// Package scope holds the scope contexts, rule sets and facts of the semantic
// analyzer (stage 6). The descent in package analysis pushes a context per
// semantic scope and runs the active rule sets at each node through the
// Apply* helpers; it cannot reach a parent context, call a rule or fabricate
// facts.
package scope

import (
	"slices"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	semcommon "github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// Context is the innermost scope of the descent. Its queries are answered by
// the package context at the root of the chain.
type Context interface {
	CompilerCtx() *context.CompilerContext
	TyCtx() semtypes.Context
	ModuleVar(model.SymbolRef) (VarDecl, bool)
	SemanticErr(msg string, loc diagnostics.Location)
	InternalErr(msg string, loc diagnostics.Location)
	UnimplementedErr(msg string, loc diagnostics.Location)
	parent() Context
	rules() []ruleSet
}

// ScopeID identifies a closure within a context chain; it is the number of
// closures from the package down to and including it.
type ScopeID uint16

// ClosureKind is the kind of an expression closure.
type ClosureKind uint8

const (
	DefaultParam ClosureKind = iota
	RecordFieldDefault
)

type functionKind uint8

const (
	fnFunction functionKind = iota
	fnModuleInit
	fnMethod
	fnLambda
	fnExpressionClosure
)

// invokableSignatureNode is implemented by functions, methods, lambdas'
// functions and resource methods.
type invokableSignatureNode interface {
	ast.BLangNode
	Symbol() model.SymbolRef
	IsIsolated() bool
	IsNative() bool
	GetParameters() []ast.BLangVariable
	GetRestParam() *ast.BLangVariable
	Scope() model.Scope
}

type (
	contextBase struct {
		outer Context
	}

	ruleBase struct {
		active []ruleSet
	}

	packageContext struct {
		ruleBase
		compilerCtx *context.CompilerContext
		tyCtx       semtypes.Context
		moduleVars  map[model.SymbolRef]VarDecl
	}

	functionContext struct {
		contextBase
		ruleBase
		class    ClassInfo
		retTy    semtypes.SemType
		locals   map[model.SymbolRef]VarDecl
		symbol   model.SymbolRef
		self     model.SymbolRef
		id       ScopeID
		kind     functionKind
		isolated bool
	}

	lockContext struct {
		contextBase
		ruleBase
		lock       *ast.BLangLock
		locals     map[model.SymbolRef]struct{}
		restricted model.SymbolRef
	}

	isolatedExprContext struct {
		contextBase
		ruleBase
	}
)

var (
	_ Context = (*packageContext)(nil)
	_ Context = (*functionContext)(nil)
	_ Context = (*lockContext)(nil)
	_ Context = (*isolatedExprContext)(nil)
)

func (c contextBase) parent() Context { return c.outer }

func (r ruleBase) rules() []ruleSet { return r.active }

func (c contextBase) CompilerCtx() *context.CompilerContext { return c.outer.CompilerCtx() }

func (c contextBase) TyCtx() semtypes.Context { return c.outer.TyCtx() }

func (c contextBase) ModuleVar(sym model.SymbolRef) (VarDecl, bool) { return c.outer.ModuleVar(sym) }

func (c contextBase) SemanticErr(msg string, loc diagnostics.Location) { c.outer.SemanticErr(msg, loc) }

func (c contextBase) InternalErr(msg string, loc diagnostics.Location) { c.outer.InternalErr(msg, loc) }

func (c contextBase) UnimplementedErr(msg string, loc diagnostics.Location) {
	c.outer.UnimplementedErr(msg, loc)
}

func (c *packageContext) parent() Context { return nil }

func (c *packageContext) CompilerCtx() *context.CompilerContext { return c.compilerCtx }

func (c *packageContext) TyCtx() semtypes.Context { return c.tyCtx }

func (c *packageContext) ModuleVar(sym model.SymbolRef) (VarDecl, bool) {
	md, ok := c.moduleVars[c.compilerCtx.UnnarrowedSymbol(sym)]
	return md, ok
}

func (c *packageContext) SemanticErr(msg string, loc diagnostics.Location) {
	c.compilerCtx.SemanticError(msg, loc)
}

func (c *packageContext) InternalErr(msg string, loc diagnostics.Location) {
	c.compilerCtx.InternalError(msg, loc)
}

func (c *packageContext) UnimplementedErr(msg string, loc diagnostics.Location) {
	c.compilerCtx.Unimplemented(msg, loc)
}

// NewPackage builds the root context of a package. It reads the module-level
// variable and constant declarations and the imported public value symbols
// to build their metadata, and reports nothing.
func NewPackage(ctx *context.CompilerContext, tyCtx semtypes.Context, pkg *ast.BLangPackage, imported map[string]model.ExportedSymbolSpace) Context {
	moduleVars := make(map[model.SymbolRef]VarDecl)
	for _, v := range semcommon.DeclaredNodes(pkg.GlobalVars) {
		moduleVars[v.Symbol()] = newVarDecl(ctx.SymbolType(v.Symbol()), v.IsFinal(), v.IsConfigurable(),
			v.Flags().Has(model.FlagIsolated), v.Expr == nil)
	}
	for _, c := range semcommon.DeclaredNodes(pkg.Constants) {
		moduleVars[c.Symbol()] = newVarDecl(ctx.SymbolType(c.Symbol()), true, c.IsConfigurable(),
			c.Flags().Has(model.FlagIsolated), false)
	}
	for _, space := range imported {
		for ref := range space.PublicMainSymbols() {
			md, ok := ctx.ValueSymbolMetadata(ref)
			if !ok || md.Parameter {
				continue
			}
			moduleVars[ref] = newVarDecl(ctx.SymbolType(ref), md.Final || md.Const, md.Configurable, md.Isolated, false)
		}
	}
	return &packageContext{
		ruleBase:    ruleBase{active: closureRules(false)},
		compilerCtx: ctx,
		tyCtx:       tyCtx,
		moduleVars:  moduleVars,
	}
}

// NewFunction pushes the closure of a module-level function.
func NewFunction(parent Context, node invokableSignatureNode) Context {
	fc := newFunctionContext(parent, node)
	if parent.CompilerCtx().SymbolName(node.Symbol()) == "init" {
		fc.kind = fnModuleInit
	}
	fc.isolated = node.IsIsolated() && !node.IsNative()
	return finishFunctionContext(fc, node)
}

// NewMethod pushes the closure of a method or resource method of class, a
// class or service.
func NewMethod(parent Context, node invokableSignatureNode, class ClassInfo) Context {
	fc := newFunctionContext(parent, node)
	fc.kind = fnMethod
	fc.isolated = node.IsIsolated() && !node.IsNative()
	fc.class = class
	self, ok := node.Scope().GetSymbol("self")
	if !ok {
		parent.InternalErr("method missing 'self' symbol", node.GetPosition())
	}
	fc.self = self
	return finishFunctionContext(fc, node)
}

// NewLambda pushes the closure of a lambda expression's function. Its class
// and self are inherited from the enclosing closure, and it is isolated when
// the lambda symbol's typed signature is isolated or the enclosing closure is.
func NewLambda(parent Context, node invokableSignatureNode) Context {
	fc := newFunctionContext(parent, node)
	fc.kind = fnLambda
	enclosing, hasEnclosing := EnclosingFunction(parent)
	sig, hasSig := parent.CompilerCtx().FunctionTypedSignature(node.Symbol())
	fc.isolated = (hasSig && sig.Flags&model.FuncSymbolFlagIsolated != 0) || (hasEnclosing && enclosing.IsIsolated())
	if hasEnclosing {
		fc.class, fc.self = enclosing.class, enclosing.self
	}
	return finishFunctionContext(fc, node)
}

func newFunctionContext(parent Context, node invokableSignatureNode) *functionContext {
	fc := &functionContext{
		contextBase: contextBase{outer: parent},
		locals:      make(map[model.SymbolRef]VarDecl),
		symbol:      node.Symbol(),
		id:          nextScopeID(parent),
	}
	if sig, ok := parent.CompilerCtx().FunctionTypedSignature(node.Symbol()); ok {
		fc.retTy = sig.ReturnType
	}
	return fc
}

func finishFunctionContext(fc *functionContext, node invokableSignatureNode) Context {
	seedParameterLocals(fc, node)
	fc.active = closureRules(fc.isolated)
	return fc
}

// NewExpressionClosure pushes the closure of a default-parameter expression
// (parent is the owning invokable's context) or of a record-field default.
func NewExpressionClosure(parent Context, kind ClosureKind) Context {
	enclosing, hasEnclosing := EnclosingFunction(parent)
	fc := &functionContext{
		contextBase: contextBase{outer: parent},
		locals:      make(map[model.SymbolRef]VarDecl),
		id:          nextScopeID(parent),
		kind:        fnExpressionClosure,
	}
	if hasEnclosing {
		fc.class, fc.self = enclosing.class, enclosing.self
	}
	switch kind {
	case DefaultParam:
		fc.isolated = hasEnclosing && enclosing.IsIsolated()
	case RecordFieldDefault:
		fc.isolated = true
	}
	fc.active = closureRules(fc.isolated)
	return fc
}

func closureRules(isolated bool) []ruleSet {
	if isolated {
		return []ruleSet{outsideLockRules(), isolationRules()}
	}
	return []ruleSet{outsideLockRules()}
}

// NewLock pushes the scope of a lock statement's body. Inside it the
// outside-lock checks do not apply and the three lock rule sets do.
func NewLock(parent Context, lock *ast.BLangLock) Context {
	common.Assert(func() bool { _, nested := EnclosingLock(parent); return !nested })
	return &lockContext{
		contextBase: contextBase{outer: parent},
		ruleBase: ruleBase{active: slices.Concat(withoutKind(parent.rules(), outsideLock),
			[]ruleSet{lockCandidateRules(), lockTransferRules(), lockCallRules()})},
		lock:   lock,
		locals: make(map[model.SymbolRef]struct{}),
	}
}

// NewIsolatedExpr pushes the scope of an expression whose type is a subtype
// of Isolated inside a lock: a reference to a variable outside the lock is
// allowed within it, so the transfer rules do not apply.
func NewIsolatedExpr(parent Context, expr ast.BLangExpression) Context {
	common.Assert(func() bool {
		_, inLock := EnclosingLock(parent)
		tyCtx := parent.TyCtx()
		return inLock && semtypes.IsSubtype(tyCtx, expr.Type(), semtypes.CreateIsolated(tyCtx))
	})
	return &isolatedExprContext{
		contextBase: contextBase{outer: parent},
		ruleBase:    ruleBase{active: withoutKind(parent.rules(), lockTransfer)},
	}
}

func withoutKind(rules []ruleSet, kind ruleKind) []ruleSet {
	return slices.DeleteFunc(slices.Clone(rules), func(r ruleSet) bool { return r.kind == kind })
}

func seedParameterLocals(fc *functionContext, node invokableSignatureNode) {
	params := node.GetParameters()
	for i := range params {
		if ast.SymbolIsSet(&params[i]) {
			defineParameter(fc, params[i].Symbol())
		}
	}
	if rest := node.GetRestParam(); rest != nil && ast.SymbolIsSet(rest) {
		defineParameter(fc, rest.Symbol())
	}
	if rm, ok := node.(*ast.BLangResourceMethod); ok {
		for i := range rm.ResourcePath {
			seg := &rm.ResourcePath[i]
			if seg.Kind == ast.ResourcePathSegmentName || seg.Name == "" {
				continue
			}
			if ref, ok := node.Scope().GetSymbol(seg.Name); ok {
				defineParameter(fc, ref)
			}
		}
	}
	if fc.kind == fnMethod {
		defineParameter(fc, fc.self)
	}
}

func defineParameter(fc *functionContext, sym model.SymbolRef) {
	common.Assert(func() bool { _, defined := fc.locals[sym]; return !defined })
	fc.locals[sym] = newVarDecl(fc.CompilerCtx().SymbolType(sym), true, false, false, false)
}
