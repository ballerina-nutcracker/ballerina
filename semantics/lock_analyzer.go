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
	"fmt"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
)

// enclosingClassOf walks the analyzer parent chain and returns the
// enclosingClassBody of the nearest functionAnalyzer that has one set.
// The boolean is false if no such analyzer exists (e.g. the lock is
// inside a free function).
func enclosingClassOf(a analyzer) (*enclosingClassBody, bool) {
	for cur := a; cur != nil; cur = cur.parentAnalyzer() {
		fa, ok := cur.(*functionAnalyzer)
		if !ok {
			continue
		}
		if fa.enclosingClass != nil {
			return fa.enclosingClass, true
		}
	}
	return nil, false
}

// findRestrictedVariable walks body and identifies the unique restricted
// variable referenced inside it. A restricted variable is one of:
//
//   - a module-level value symbol declared with the `isolated` qualifier; or
//   - a non-immutable field of an isolated class accessed via `self` from a
//     method of that class.
//
// Returns the lock key, the restricted symbol and whether one was found.
func findRestrictedVariable(a analyzer, body *ast.BLangBlockStmt) (key string, sym model.SymbolRef, ok bool) {
	ok = true
	everyNode(a, body, func(_ analyzer, inner ast.BLangNode) bool {
		switch n := inner.(type) {
		case *ast.BLangLambdaFunction, *ast.BLangFunction:
			return false
		case *ast.BLangSimpleVarRef:
			unnarrowed := a.ctx().UnnarrowedSymbol(n.Symbol())
			switch s := a.ctx().GetSymbol(unnarrowed).(type) {
			case model.ValueSymbol:
				if s.IsIsolated() {
					if !sym.IsEmpty() {
						if unnarrowed != sym {
							a.semanticErr("more than one restricted variable referenced in lock statement", n.GetPosition())
							ok = false
						}
					} else {
						sym = unnarrowed
						key = moduleVarLockKey(a.ctx().SymbolPackage(unnarrowed), s.Name())
					}
				}
			case model.FunctionSymbol:
				// Function value reference; never a restricted variable.
			default:
				a.internalErr(fmt.Sprintf("unexpected symbol kind in variable reference: %T", s), n.GetPosition())
			}
		case *ast.BLangFieldBaseAccess:
			if k, ref, isIsolatedFieldAccess := selfFieldLockEntry(a, n); isIsolatedFieldAccess {
				if !sym.IsEmpty() {
					if ref != sym {
						a.semanticErr("more than one restricted variable referenced in lock statement", n.GetPosition())
						ok = false
					}
				} else {
					sym = ref
					key = k
				}
			}
		}
		return true
	})
	return key, sym, ok
}

// selfFieldLockEntry returns the (lock key, field's symbol ref, ok) for
// `self.fieldName` accesses qualifying as restricted-variable references.
func selfFieldLockEntry(a analyzer, access *ast.BLangFieldBaseAccess) (string, model.SymbolRef, bool) {
	if !isSelfFieldAccess(access) {
		return "", model.SymbolRef{}, false
	}
	cls, ok := enclosingClassOf(a)
	if !ok {
		a.internalErr("failed to find enclosing class definition", access.GetPosition())
		return "", model.SymbolRef{}, false
	}
	if !cls.isolated {
		return "", model.SymbolRef{}, false
	}
	fieldName := access.Field.GetValue()
	for _, f := range cls.fields {
		field := f.(*ast.BLangSimpleVariable)
		if field.Name.GetValue() != fieldName {
			continue
		}
		if isImmutableField(a.tyCtx(), field) {
			return "", model.SymbolRef{}, false
		}
		pkg := a.ctx().SymbolPackage(field.Symbol())
		return classBodyFieldLockKey(pkg, cls, field.Symbol(), fieldName), field.Symbol(), true
	}
	return "", model.SymbolRef{}, false
}

func moduleVarLockKey(pkg model.PackageIdentifier, name string) string {
	return pkg.Organization + "/" + pkg.Package + ":" + name
}

func classFieldLockKey(pkg model.PackageIdentifier, className, fieldName string) string {
	return pkg.Organization + "/" + pkg.Package + ":" + className + "." + fieldName
}

// classBodyFieldLockKey returns a per-field lock key for either a class or a
// service. For classes it uses the class's module-level name; for services
// (which have no name) it derives a unique key from the field's SymbolRef,
// which is already unique within the package.
func classBodyFieldLockKey(pkg model.PackageIdentifier, cls *enclosingClassBody, fieldRef model.SymbolRef, fieldName string) string {
	if cls.name != "" {
		return classFieldLockKey(pkg, cls.name, fieldName)
	}
	return fmt.Sprintf("%s/%s:$service.%d.%d.%s", pkg.Organization, pkg.Package, fieldRef.SpaceIndex, fieldRef.Index, fieldName)
}

// validateLockStmt determines the restricted variable and validate semantics of a lock statment.
func validateLockStmt(a analyzer, lock *ast.BLangLock) bool {
	if !resolveRestricted(a, lock) {
		return false
	}
	if !validateLockInvocations(a, &lock.Body) {
		return false
	}
	return validateLockBody(a, lock)
}

// resolveRestricted determines the lock's restricted variable, caching
// the result on the AST node. A non-zero RestrictedSymbol is the sentinel
// for "already resolved"; subsequent calls reuse it without recomputing.
func resolveRestricted(a analyzer, lock *ast.BLangLock) bool {
	if !lock.RestrictedSymbol.IsEmpty() {
		return true
	}
	key, sym, ok := findRestrictedVariable(a, &lock.Body)
	lock.LockKey, lock.RestrictedSymbol = key, sym
	return ok
}

// validateLockInvocations enforces all invocations within lock statement must be to isolated functions.
func validateLockInvocations(a analyzer, body *ast.BLangBlockStmt) bool {
	tyCtx := a.tyCtx()
	isolatedFn := semtypes.CreateIsolatedFn(tyCtx)
	ok := true
	everyNode(a, body, func(_ analyzer, inner ast.BLangNode) bool {
		switch n := inner.(type) {
		case *ast.BLangLambdaFunction, *ast.BLangFunction:
			return false
		case *ast.BLangInvocation:
			if loc, invalid := isolatedInvocationViolation(a, n); invalid {
				a.semanticErr("invocation of a non-isolated function inside lock statement", loc)
				ok = false
			}
		case *ast.BLangRemoteMethodCallAction:
			fnSym := n.MethodSymbol()
			if !semtypes.IsSubtype(tyCtx, a.ctx().SymbolType(fnSym), isolatedFn) {
				a.semanticErr("invocation of a non-isolated function inside lock statement", n.GetPosition())
				ok = false
			}
		}
		return true
	})
	return ok
}

func isolatedInvocationViolation(a analyzer, n *ast.BLangInvocation) (diagnostics.Location, bool) {
	return isolatedInvocationViolationInner(a.ctx(), a.tyCtx(), n)
}

func isolatedInvocationViolationInner(ctx *context.CompilerContext, tyCtx semtypes.Context, n *ast.BLangInvocation) (diagnostics.Location, bool) {
	if ast.IsStreamOperation(n) {
		return diagnostics.Location{}, false
	}
	isolatedFn := semtypes.CreateIsolatedFn(tyCtx)
	fnRef := n.Symbol()
	if !semtypes.IsSubtype(tyCtx, ctx.SymbolType(fnRef), isolatedFn) {
		return n.GetPosition(), true
	}

	resolvedSymbol := ctx.GetSymbol(fnRef)
	resolvedFn, ok := resolvedSymbol.(model.FunctionSymbol)
	if !ok {
		if _, isFunctionValue := resolvedSymbol.(model.ValueSymbol); isFunctionValue {
			return diagnostics.Location{}, false
		}
		ctx.InternalError(fmt.Sprintf("unexpected symbol kind in invocation: %T", resolvedSymbol), n.GetPosition())
		return diagnostics.Location{}, false
	}
	originalRef := fnRef
	if mono, ok := resolvedFn.(model.MonomorphicFunctionSymbol); ok {
		originalRef = mono.PolymorphicSymbol()
	}
	opaque, ok := ctx.GetSymbol(originalRef).(*model.OpaqueFunctionSymbol)
	if !ok {
		return diagnostics.Location{}, false
	}

	paramNames := resolvedFn.ParamNames()
	for i, arg := range n.CallArgs() {
		paramIndex := i
		argExpr := arg
		if named, ok := arg.(*ast.BLangNamedArgsExpression); ok {
			paramIndex = paramIndexOf(paramNames, named.Name.GetValue())
			argExpr = named.Expr
		}
		if paramIndex < 0 || !opaque.IsIsolatedParam(paramIndex) {
			continue
		}
		if !semtypes.IsSubtype(tyCtx, argExpr.GetDeterminedType(), isolatedFn) {
			return arg.GetPosition(), true
		}
	}
	return diagnostics.Location{}, false
}

// validateLockBody validates transfer in and out conditions.
// transfer in:
//   - expression fallowing return must be isolated expression
//   - assignement to variable defined outside of lock body is allowed only if lhs is a just a variable name and rhs is isolated expression.
//     Or the targe must be the restricted-variable
//
// transfer out:
//   - Variable references to variables not define inside the lock statement is allowed only if the refernce occures inside an isolated expression.
//     Or the referred variable must be the restricted-variable
func validateLockBody(a analyzer, lock *ast.BLangLock) bool {
	cls, _ := enclosingClassOf(a)
	v := &lockBodyVisitor{enclosingClass: cls, a: a, lock: lock, locals: make(map[model.SymbolRef]struct{}), ok: true}
	ast.Walk(v, &lock.Body)
	return v.ok
}

type lockBodyVisitor struct {
	a              analyzer
	enclosingClass *enclosingClassBody
	lock           *ast.BLangLock
	locals         map[model.SymbolRef]struct{}
	ok             bool
}

func (v *lockBodyVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor { return v }

func (v *lockBodyVisitor) Visit(n ast.BLangNode) ast.Visitor {
	if n == nil {
		return v
	}
	switch node := n.(type) {
	case *ast.BLangLambdaFunction, *ast.BLangFunction:
		return nil
	case *ast.BLangSimpleVariableDef:
		v.locals[node.Var.Symbol()] = struct{}{}
	case *ast.BLangAssignment:
		v.checkAssignment(node.VarRef, node.Expr, node.GetPosition())
	case *ast.BLangCompoundAssignment:
		v.checkAssignment(node.VarRef.(ast.BLangExpression), node.Expr, node.GetPosition())
		return v
	case *ast.BLangReturn:
		if node.Expr != nil && !isIsolatedExpression(v.a, node.Expr.(ast.BLangExpression)) {
			v.a.semanticErr("access of mutable variable", node.Expr.(ast.BLangNode).GetPosition())
			v.ok = false
		}
	case ast.BLangExpression:
		if v.containsTransferInRef(node) {
			if !isIsolatedExpression(v.a, node) {
				v.a.semanticErr("access of mutable variable", node.GetPosition())
				v.ok = false
			}
			return nil
		}
	}
	return v
}

// checkAssignment validates transfer in for assignment.
func (v *lockBodyVisitor) checkAssignment(lhs ast.BLangExpression, rhs ast.BLangActionOrExpression, pos diagnostics.Location) {
	// If the LHS targets the restricted variable, no check.
	if v.assignsRestricted(lhs) {
		return
	}
	// If the LHS is a local declared inside the lock body, no check.
	lhsRef, ok := exprRef(v.enclosingFields(), lhs)
	if !ok {
		v.ok = false
		v.a.internalErr("failed to find variable symbol", lhs.GetPosition())
		return
	}
	lhsRef = v.a.ctx().UnnarrowedSymbol(lhsRef)
	if _, isLocal := v.locals[lhsRef]; isLocal {
		return
	}

	// LHS must be a plain variable reference, and RHS must be an
	// isolated expression.
	if _, ok := lhs.(*ast.BLangSimpleVarRef); !ok {
		v.a.semanticErr("assignment in a lock statement to a target defined outside the lock must use a plain variable name on the left-hand side", pos)
		v.ok = false
		return
	}
	expr, ok := rhs.(ast.BLangExpression)
	if !ok {
		return
	}
	if !isIsolatedExpression(v.a, expr) {
		v.a.semanticErr("access of mutable variable", rhs.GetPosition())
		v.ok = false
	}
}

// assignsRestricted reports whether lhs targets the lock's restricted variable.
// findRestrictedVariable now returns a uniform SymbolRef in both module-var
// and self-field cases, so this is a single ref-equality test on the LHS
// base symbol.
func (v *lockBodyVisitor) assignsRestricted(lhs ast.BLangExpression) bool {
	lhsRef, ok := exprRef(v.enclosingFields(), lhs)
	if !ok {
		v.ok = false
		v.a.internalErr("failed to find variable symbol", lhs.GetPosition())
		// avoid continuing the validation
		return true
	}
	lhsRef = v.a.ctx().UnnarrowedSymbol(lhsRef)
	return v.lock.RestrictedSymbol == lhsRef
}

// containsTransferInRef reports whether expr's subtree contains any reference
// to a variable defined outside the lock body that isn't the restricted
// variable or `self`.
func (v *lockBodyVisitor) containsTransferInRef(expr ast.BLangExpression) bool {
	found := false
	everyNode(v.a, expr, func(_ analyzer, n ast.BLangNode) bool {
		ref, ok := n.(*ast.BLangSimpleVarRef)
		if !ok {
			return true
		}
		if ref.VariableName.GetValue() == "self" {
			return true
		}
		unnarrowed := v.a.ctx().UnnarrowedSymbol(ref.Symbol())
		if _, isLocal := v.locals[unnarrowed]; isLocal {
			return true
		}
		if unnarrowed == v.lock.RestrictedSymbol {
			return true
		}
		found = true
		return false
	})
	return found
}

type varDeclMetadata struct {
	Type         semtypes.SemType
	Final        bool
	Configurable bool
	Isolated     bool
}

// buildModuleVarMetadata is invoked once per Analyze call to build the
// SymbolRef -> varDeclMetadata snapshot used by isolation analysis. Walks
// the current package's module-level variables and constants, plus the
// exported value symbols (variables and constants) of every imported
// package.
func (sa *SemanticAnalyzer) buildModuleVarMetadata() map[model.SymbolRef]varDeclMetadata {
	out := make(map[model.SymbolRef]varDeclMetadata)
	for i := range sa.pkg.GlobalVars {
		v := &sa.pkg.GlobalVars[i]
		out[v.Symbol()] = varDeclMetadata{
			Type:         v.GetDeterminedType(),
			Final:        v.IsFinal(),
			Configurable: v.IsConfigurable(),
			Isolated:     v.Flags().Has(model.FlagIsolated),
		}
	}
	for i := range sa.pkg.Constants {
		c := &sa.pkg.Constants[i]
		out[c.Symbol()] = varDeclMetadata{
			Type:         c.GetDeterminedType(),
			Final:        true,
			Configurable: c.IsConfigurable(),
			Isolated:     c.Flags().Has(model.FlagIsolated),
		}
	}
	for _, space := range sa.importedSymbols {
		for ref := range space.PublicMainSymbols() {
			metadata, ok := sa.compilerCtx.ValueSymbolMetadata(ref)
			if !ok || metadata.Parameter {
				continue
			}
			out[ref] = varDeclMetadata{
				Type:         sa.compilerCtx.SymbolType(ref),
				Final:        metadata.Final || metadata.Const,
				Configurable: metadata.Configurable,
				Isolated:     metadata.Isolated,
			}
		}
	}
	return out
}

// isIsolatedExpression reports whether expr is an "isolated expression" under
// the spec.
//
//  1. If the expression's static type is a subtype of `Isolated`
//     (`readonly | isolated object {}`), it is isolated.
//  2. Otherwise, list/mapping constructors, type conversion, check, and trap
//     expressions are isolated iff every immediate value child is itself an
//     isolated expression.
//  3. All other expressions are not isolated.
func isIsolatedExpression(a analyzer, expr ast.BLangExpression) bool {
	if expr == nil {
		a.ctx().InternalError("nil expression in isolation check", diagnostics.Location{})
		return false
	}
	tyCtx := a.tyCtx()
	ty := expr.GetDeterminedType()
	if semtypes.IsSubtype(tyCtx, ty, semtypes.CreateIsolated(tyCtx)) {
		return true
	}
	switch e := expr.(type) {
	case *ast.BLangGroupExpr:
		return isIsolatedExpression(a, e.Expression)
	case *ast.BLangListConstructorExpr:
		for _, m := range e.Exprs {
			if !isIsolatedExpression(a, m) {
				return false
			}
		}
		return true
	case *ast.BLangMappingConstructorExpr:
		for _, f := range e.Fields {
			kv, ok := f.(*ast.BLangMappingKeyValueField)
			if !ok {
				a.ctx().InternalError(fmt.Sprintf("unexpected mapping field kind %T", f), f.GetPosition())
				return false
			}
			if !isIsolatedExpression(a, kv.ValueExpr) {
				return false
			}
		}
		return true
	case *ast.BLangTypeConversionExpr:
		return isIsolatedExpression(a, e.Expression)
	case *ast.BLangCheckedExpr:
		return isIsolatedExpression(a, e.Expr.(ast.BLangExpression))
	case *ast.BLangCheckPanickedExpr:
		return isIsolatedExpression(a, e.Expr.(ast.BLangExpression))
	case *ast.BLangTrapExpr:
		return isIsolatedExpression(a, e.Expr)
	}
	return false
}

func checkIsolatedModuleVarOutsideLock(a analyzer, ref *ast.BLangSimpleVarRef) {
	unnarrowed := a.ctx().UnnarrowedSymbol(ref.Symbol())
	sym, ok := a.ctx().GetSymbol(unnarrowed).(model.ValueSymbol)
	if !ok || !sym.IsIsolated() {
		return
	}
	if enclosingLockAnalyzer(a) == nil {
		a.semanticErr(
			"access of an isolated variable must be inside a lock statement",
			ref.GetPosition(),
		)
	}
}

// checkIsolatedFieldOutsideLock is the field-access twin of
// checkIsolatedModuleVarOutsideLock. It rejects every `self.f`
// reference where `f` is a non-final field of an isolated class and
// the reference's enclosing closure has no lock analyzer in scope.
// Because enclosingLockAnalyzer stops at the nearest functionAnalyzer
// (lambdas push their own), this also rejects captures of `self.f`
// inside lambdas constructed within a lock body.
func checkIsolatedFieldOutsideLock(a analyzer, access *ast.BLangFieldBaseAccess) {
	_, _, ok := selfFieldLockEntry(a, access)
	if !ok {
		return
	}
	if inInitFunction(a) {
		return
	}
	if enclosingLockAnalyzer(a) == nil {
		a.semanticErr(
			"mutable field access within isolated object must be inside a lock statement",
			access.GetPosition(),
		)
	}
}

// inInitFunction reports whether `a` is inside the body of the
// enclosing class's `init` method (still within the same closure —
// crossing into a lambda detaches us from init). Mutations of
// `self.f` inside init are exempt because the constructed object is
// not observable to other strands until init returns.
func inInitFunction(a analyzer) bool {
	for cur := a; cur != nil; cur = cur.parentAnalyzer() {
		fa, ok := cur.(*functionAnalyzer)
		if !ok {
			continue
		}
		return fa.enclosingClass != nil && fa.enclosingClass.initFn == fa.function
	}
	return false
}

// validateModuleLevelIsolatedDecls enforces that the initializer of
// every module-level `isolated` declaration is itself an isolated
// expression.
func (sa *SemanticAnalyzer) validateModuleLevelIsolatedDecls(pkg *ast.BLangPackage) {
	check := func(expr ast.BLangExpression, sym model.SymbolRef) {
		vs, ok := sa.ctx().GetSymbol(sym).(model.ValueSymbol)
		if !ok || !vs.IsIsolated() {
			return
		}
		if !isIsolatedExpression(sa, expr) {
			sa.semanticErr(
				"initializer of an isolated variable must be an isolated expression",
				expr.GetPosition(),
			)
		}
	}
	for i := range pkg.GlobalVars {
		v := &pkg.GlobalVars[i]
		if v.Expr == nil {
			continue
		}
		check(v.Expr.(ast.BLangExpression), v.Symbol())
	}
	for i := range pkg.Constants {
		c := &pkg.Constants[i]
		if c.Expr == nil {
			continue
		}
		check(c.Expr.(ast.BLangExpression), c.Symbol())
	}
}

// validateIsolatedFunction validates the body of an isolated function under
// the rules described on isIsolatedFunctionInner, using the enclosing
// function-analyzer's locals scope (seeded with parameters) so that inner
// closures resolving past it land on the capture branch of checkRead.
func validateIsolatedFunction(a analyzer, fn invokableSignatureNode) {
	if fn.GetBody() == nil {
		if !fn.IsNative() {
			a.ctx().InternalError("non-native function with nil body", fn.GetPosition())
		}
		return
	}
	isIsolatedFunctionInner(a, fn.GetBody().(ast.BLangNode), enclosingFunctionLocals(a))
}

// isIsolatedFunctionInner walks an arbitrary node treating it as the body of
// an isolated closure. `scope` is the lexical scope chain visible at the
// entry node; pass nil when there is no enclosing function frame
// (record-field defaults, default-parameter expressions). In that case a
// fresh fn-boundary scope with no parent is created so local definitions
// inside the walked node have somewhere to live; captures of outer
// parameters are handled by checkRead's fall-through branch on a missing
// scope lookup.
//
// Rules enforced:
//
//  1. Calls (invocations / remote method calls / class init via `new`) must
//     resolve to isolated functions. Stream operations are unconditionally
//     treated as isolated.
//  2. A reference to a module-level variable is allowed iff the variable
//     declaration is final or configurable, not declared `isolated`, AND its
//     type is a subtype of `Isolated`.
//  3. Assignment whose LHS is rooted in a module-level variable is rejected
//     unless they are isolated. (we detect that they are in lock statements in normal semantic analysis)
//  4. Captures of outer-scope locally-declared variables (including
//     parameters of an enclosing function) must be effectively final and
//     have a type that is a subtype of `Isolated`. Lambdas push a new
//     function-boundary frame so refs that resolve past that frame land on
//     the capture branch of checkRead; non-lambda closures (record-field
//     defaults, default-parameter expressions) reach the same rule via the
//     fall-through branch on a missing scope lookup.
func isIsolatedFunctionInner(a analyzer, node ast.BLangNode, scope *localScope) {
	if scope == nil {
		scope = newLocalScope(nil, true)
	}
	v := &isolatedFnVisitor{a: a, scope: scope}
	ast.Walk(v, node)
}

// localScope is a lexical scope frame in a parent-linked chain. A frame
// marked fnBoundary starts a new function frame: lookups that pass through
// such a frame are captures of an enclosing function's locals.
type localScope struct {
	parent     *localScope
	fnBoundary bool
	vars       map[model.SymbolRef]varDeclMetadata
}

func newLocalScope(parent *localScope, fnBoundary bool) *localScope {
	return &localScope{
		parent:     parent,
		fnBoundary: fnBoundary,
		vars:       map[model.SymbolRef]varDeclMetadata{},
	}
}

func (s *localScope) define(sym model.SymbolRef, md varDeclMetadata) {
	s.vars[sym] = md
}

// lookup walks the scope chain. The third return is true iff at least one
// function-boundary frame was crossed before the symbol was found, i.e. the
// reference is a capture of an enclosing function's local.
func (s *localScope) lookup(sym model.SymbolRef) (varDeclMetadata, bool, bool) {
	crossed := false
	for cur := s; cur != nil; cur = cur.parent {
		if md, ok := cur.vars[sym]; ok {
			return md, true, crossed
		}
		if cur.fnBoundary {
			crossed = true
		}
	}
	return varDeclMetadata{}, false, false
}

type isolatedFnVisitor struct {
	a     analyzer
	scope *localScope
}

func (visitor *isolatedFnVisitor) Visit(n ast.BLangNode) ast.Visitor {
	if n == nil {
		return visitor
	}
	a := visitor.a
	tyCtx := a.tyCtx()
	isolatedFn := semtypes.CreateIsolatedFn(tyCtx)
	switch node := n.(type) {
	case *ast.BLangSimpleVariableDef:
		v := node.Var
		visitor.scope.define(v.Symbol(), varDeclMetadata{
			Type:  v.GetDeterminedType(),
			Final: v.IsFinal(),
		})
		return visitor
	case *ast.BLangInvocation:
		if loc, invalid := isolatedInvocationViolation(a, node); invalid {
			a.semanticErr("invocation of a non-isolated function", loc)
		}
		return visitor
	case *ast.BLangRemoteMethodCallAction:
		fnSym := node.MethodSymbol()
		if !semtypes.IsSubtype(tyCtx, a.ctx().SymbolType(fnSym), isolatedFn) {
			a.semanticErr("invocation of a non-isolated function", n.GetPosition())
		}
		return visitor
	case *ast.BLangNewExpression:
		if !checkIsolatedNew(visitor.a, node) {
			a.semanticErr("non isolated initialization", n.GetPosition())
		}
		return visitor
	case *ast.BLangLambdaFunction:
		visitor.walkLambda(node)
		return nil
	case *ast.BLangLock:
		visitor.walkLock(node)
		return nil
	case *ast.BLangSimpleVarRef:
		visitor.checkRead(node)
		return visitor
	}
	return visitor
}

func (visitor *isolatedFnVisitor) walkLambda(node *ast.BLangLambdaFunction) {
	fn := node.Function
	validateIsolatedCapture(visitor.a, visitor.scope, fn.GetBody().(ast.BLangNode))
	inner := newLocalScope(visitor.scope, true)
	for _, param := range fn.RequiredParams {
		sym := param.Symbol()
		inner.define(sym, varDeclMetadata{Type: visitor.a.ctx().SymbolType(sym), Final: true})
	}
	if fn.RestParam != nil {
		sym := fn.RestParam.Symbol()
		inner.define(sym, varDeclMetadata{Type: visitor.a.ctx().SymbolType(sym), Final: true})
	}
	prev := visitor.scope
	visitor.scope = inner
	defer func() { visitor.scope = prev }()
	ast.Walk(visitor, fn.GetBody().(ast.BLangNode))
}

// validateIsolatedCapture validates that every reference inside `body` to a
// variable visible in `outer` is final and has a type that is a subtype of
// `Isolated`. Used at every closure boundary (nested lambdas, record-field
// defaults, default-param exprs).
func validateIsolatedCapture(a analyzer, outer *localScope, body ast.BLangNode) {
	if outer == nil {
		return
	}
	tyCtx := a.tyCtx()
	isolated := semtypes.CreateIsolated(tyCtx)
	v := &captureVisitor{a: a, outer: outer, isolated: isolated}
	ast.Walk(v, body)
}

type captureVisitor struct {
	a        analyzer
	outer    *localScope
	isolated semtypes.SemType
}

func (v *captureVisitor) Visit(n ast.BLangNode) ast.Visitor {
	if n == nil {
		return v
	}
	if ref, ok := n.(*ast.BLangSimpleVarRef); ok {
		unnarrowed := v.a.ctx().UnnarrowedSymbol(ref.Symbol())
		if md, found, _ := v.outer.lookup(unnarrowed); found {
			if !md.Final || !semtypes.IsSubtype(v.a.tyCtx(), md.Type, v.isolated) {
				v.a.semanticErr("invalid capture of mutable variable in isolated lambda", ref.GetPosition())
			}
		}
	}
	return v
}

func (v *captureVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor { return v }

// walkLock pushes a non-function scope for the lock body. The lock's
// restricted variable is treated as a local within the body — actual
// read/write constraints on it are enforced separately by validateLockStmt.
// Restricted-variable resolution goes through resolveRestricted so it
// happens at most once per lock regardless of which pass reaches it first.
func (visitor *isolatedFnVisitor) walkLock(node *ast.BLangLock) {
	inner := newLocalScope(visitor.scope, false)
	if resolveRestricted(visitor.a, node) && !node.RestrictedSymbol.IsEmpty() {
		inner.define(node.RestrictedSymbol, varDeclMetadata{})
	}
	prev := visitor.scope
	visitor.scope = inner
	defer func() { visitor.scope = prev }()
	ast.Walk(visitor, &node.Body)
}

func (visitor *isolatedFnVisitor) VisitTypeData(_ *ast.TypeData) ast.Visitor { return visitor }

func checkIsolatedNew(a analyzer, expr *ast.BLangNewExpression) bool {
	return checkIsolatedNewWithContext(a.ctx(), a.tyCtx(), expr)
}

func checkIsolatedNewWithContext(ctx *context.CompilerContext, tyCtx semtypes.Context, expr *ast.BLangNewExpression) bool {
	if ast.IsStreamNewExpression(expr) {
		return true
	}
	classTy := ctx.SymbolType(expr.ClassSymbol)
	initTy := semtypes.ObjectMemberType(tyCtx, semtypes.StringConst("init"), classTy)
	if semtypes.IsZero(initTy) || !semtypes.IsSubtype(tyCtx, initTy, semtypes.CreateIsolatedFn(tyCtx)) {
		return false
	}
	return true
}

// Non isolated module level variables can be read under 2 conditions
// 1. Identifier is declared final or configurable but not isolated
// 2. Variable is a subtype of isolated
func (visitor *isolatedFnVisitor) checkRead(ref *ast.BLangSimpleVarRef) {
	tyCtx := visitor.a.tyCtx()
	unnarrowed := visitor.a.ctx().UnnarrowedSymbol(ref.Symbol())
	if _, ok, _ := visitor.scope.lookup(unnarrowed); ok {
		// local declaration
		return
	}
	if md, ok := visitor.a.moduleVarMetadata(unnarrowed); ok {
		isolated := semtypes.CreateIsolated(tyCtx)
		if (md.Final || md.Configurable) && !md.Isolated &&
			semtypes.IsSubtype(tyCtx, md.Type, isolated) {
			return
		}
		visitor.a.semanticErr("access of mutable variable", ref.GetPosition())
		return
	}
	if ref.VariableName.GetValue() == "self" {
		return
	}
}

func (v *lockBodyVisitor) enclosingFields() []ast.SimpleVariableNode {
	if v.enclosingClass == nil {
		return nil
	}
	return v.enclosingClass.fields
}

func exprRef(enclosingFields []ast.SimpleVariableNode, expr ast.BLangExpression) (model.SymbolRef, bool) {
	switch expr := expr.(type) {
	case *ast.BLangSimpleVarRef:
		return expr.Symbol(), true
	case *ast.BLangFieldBaseAccess:
		if isSelfFieldAccess(expr) {
			fieldName := expr.Field.GetValue()
			for _, f := range enclosingFields {
				field := f.(*ast.BLangSimpleVariable)
				if field.Name.GetValue() != fieldName {
					continue
				}
				return field.Symbol(), true
			}
		} else {
			return exprRef(enclosingFields, expr.Expr)
		}
	case *ast.BLangIndexBasedAccess:
		return exprRef(enclosingFields, expr.Expr)
	}
	return model.SymbolRef{}, false
}
