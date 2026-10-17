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

package analysis

import (
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// invokableNode is a function, method, lambda's function or resource method.
type invokableNode interface {
	ast.BLangNode
	Symbol() model.SymbolRef
	IsIsolated() bool
	IsNative() bool
	GetName() ast.IdentifierNode
	GetParameters() []ast.BLangVariable
	GetRestParam() *ast.BLangVariable
	GetBody() ast.FunctionBodyNode
	GetAnnotationAttachments() []ast.BLangAnnotationAttachment
	ReturnTypeDescriptorNode() *ast.BLangReturnTypeDescriptor
	Scope() model.Scope
}

// validateFunction validates a module-level function, including the main and
// module init rules; an error in those rules does not abandon the function.
func validateFunction(cx scope.Context, fn *ast.BLangFunction) {
	switch fn.Name.GetValue() {
	case "main":
		validateMainIsPublic(cx, fn)
		validateMainReturnType(cx, fn)
	case "init":
		validateInitIsNotPublic(cx, fn)
		validateInitReturnType(cx, fn)
		validateInitHasNoParams(cx, fn)
	}
	validateInvokableNode(scope.NewFunction(cx, fn), fn)
}

func validateMainIsPublic(cx scope.Context, fn *ast.BLangFunction) {
	if !cx.CompilerCtx().GetSymbol(fn.Symbol()).IsPublic() {
		cx.SemanticErr("'main' function must be public", fn.GetPosition())
	}
}

func validateMainReturnType(cx scope.Context, fn *ast.BLangFunction) {
	if sig, ok := cx.CompilerCtx().FunctionTypedSignature(fn.Symbol()); ok && !isNilOrErrorReturn(cx, sig.ReturnType) {
		cx.SemanticErr("'main' function must have return type '()' or  'error?'", fn.GetPosition())
	}
}

func validateInitReturnType(cx scope.Context, fn *ast.BLangFunction) {
	if sig, ok := cx.CompilerCtx().FunctionTypedSignature(fn.Symbol()); ok && !isNilOrErrorReturn(cx, sig.ReturnType) {
		cx.SemanticErr("'init' function must have return type '()' or  'error?'", fn.GetPosition())
	}
}

func validateInitIsNotPublic(cx scope.Context, fn *ast.BLangFunction) {
	if fn.IsPublic() {
		cx.SemanticErr("'init' function cannot be declared as public", fn.GetPosition())
	}
}

func validateInitHasNoParams(cx scope.Context, fn *ast.BLangFunction) {
	if len(fn.RequiredParams) > 0 || fn.RestParam != nil {
		cx.SemanticErr("'init' function cannot have parameters", fn.GetPosition())
	}
}

func isNilOrErrorReturn(cx scope.Context, retTy semtypes.SemType) bool {
	tyCtx := cx.TyCtx()
	return semtypes.IsZero(retTy) || semtypes.IsSameType(tyCtx, retTy, semtypes.Nil) ||
		semtypes.IsSameType(tyCtx, retTy, semtypes.Union(semtypes.Nil, semtypes.Error))
}

// validateInvokableNode validates a function or resource method under its own
// closure: signature rules, annotations, parameters (each default is its own
// closure), return type descriptor and body. An error in the signature does
// not abandon the body.
func validateInvokableNode(cx scope.Context, fn invokableNode) bool {
	signatureOK := validateSignatureHead(cx, fn)
	paramsOK := true
	params := fn.GetParameters()
	for i := range params {
		paramsOK = validateParameter(cx, scope.NewExpressionClosure(cx, scope.DefaultParam), fn, &params[i], i) && paramsOK
	}
	signatureOK = validateSignatureTail(cx, fn) && signatureOK
	bodyOK := true
	if body := fn.GetBody(); body != nil {
		bodyOK = validateBody(cx, body)
	}
	return signatureOK && paramsOK && bodyOK
}

// validateSignatureHead validates the dependent-function and resource return
// type rules and the annotations, which precede the parameters.
func validateSignatureHead(cx scope.Context, fn invokableNode) bool {
	if !validateDependentFunction(cx, fn) {
		return false
	}
	if rm, ok := fn.(*ast.BLangResourceMethod); ok {
		if !validateResourceMethodReturnType(cx, rm) {
			return false
		}
	}
	if !validateAnnotationAttachments(cx, fn.GetAnnotationAttachments()) {
		return false
	}
	return true
}

// validateSignatureTail validates the rest parameter and the return type
// descriptor, which follow the parameters.
func validateSignatureTail(cx scope.Context, fn invokableNode) bool {
	if rest := fn.GetRestParam(); rest != nil {
		if !validateAnnotationAttachments(cx, rest.AnnAttachments) {
			return false
		}
		if typeNode := rest.TypeNode(); typeNode != nil {
			if !dispatchTypeDesc(cx, typeNode) {
				return false
			}
		}
	}
	if rt := fn.ReturnTypeDescriptorNode(); rt != nil {
		if !validateAnnotationAttachments(cx, rt.AnnAttachments) {
			return false
		}
		if rt.TypeDescriptor != nil {
			if !dispatchTypeDesc(cx, rt.TypeDescriptor) {
				return false
			}
		}
	}
	return true
}

// validateDependentFunction: a dependently-typed function must be external and
// the dependent and defined parts of its return type must be disjoint.
func validateDependentFunction(cx scope.Context, fn invokableNode) bool {
	paramTypes, retOp, dependent := cx.CompilerCtx().DependentlyTypedFunctionType(fn.Symbol())
	if dependent {
		if _, external := fn.GetBody().(*ast.BLangExternFunctionBody); !external {
			cx.SemanticErr("dependently-typed function must be external", fn.GetPosition())
			return false
		}
		if !dependentPartsDisjoint(cx.TyCtx(), retOp, paramTypes) {
			cx.SemanticErr("dependently-typed function return type dependent and defined parts must be disjoint", fn.GetPosition())
			return false
		}
	}
	return true
}

// validateParameter validates parameter i: annotations, type descriptor and
// default. An inferred typedesc default `<>` needs a return type that depends
// on the parameter; any other default is validated under closure, its own
// closure, and must be of the parameter's type.
func validateParameter(cx scope.Context, closure scope.Context, fn invokableNode, param *ast.BLangVariable, i int) bool {
	if !validateAnnotationAttachments(cx, param.AnnAttachments) {
		return false
	}
	if typeNode := param.TypeNode(); typeNode != nil {
		if !dispatchTypeDesc(cx, typeNode) {
			return false
		}
	}
	if param.Expr != nil {
		if !validateParameterDefault(cx, closure, fn, param, i) {
			return false
		}
	}
	return true
}

func validateParameterDefault(cx scope.Context, closure scope.Context, fn invokableNode, param *ast.BLangVariable, i int) bool {
	if _, inferred := param.Expr.(*ast.BLangInferredTypedescDefault); inferred {
		_, retOp, dependent := cx.CompilerCtx().DependentlyTypedFunctionType(fn.Symbol())
		if !dependent || !slices.Contains(typeOpRefIndices(retOp), i) {
			cx.SemanticErr("inferred typedesc default '<>' requires the return type to depend on this parameter", param.GetPosition())
			return false
		}
	} else {
		if _, ok := validateActionOrExpr(closure, param.Expr, semtypes.SemType{}); !ok {
			return false
		}
		if !semtypes.IsSubtype(cx.TyCtx(), param.Expr.Type(), cx.CompilerCtx().SymbolType(param.Symbol())) {
			cx.SemanticErr(strings.Join([]string{"incompatible default value for parameter '", param.Name.GetValue(), "'"}, ""), param.Expr.GetPosition())
			return false
		}
	}
	return true
}

func validateResourceMethodReturnType(cx scope.Context, rm *ast.BLangResourceMethod) bool {
	if sig, ok := cx.CompilerCtx().FunctionTypedSignature(rm.Symbol()); ok {
		tyCtx := cx.TyCtx()
		if !semtypes.IsEmpty(tyCtx, semtypes.Intersect(sig.ReturnType, semtypes.Function)) {
			cx.SemanticErr("resource method return type must not include a function type", rm.GetPosition())
			return false
		}
		// A plain object's network qualifier is "client"|"service", so
		// client-ness is tested directly rather than by intersecting with
		// `client object {}`.
		if semtypes.IsClientObject(tyCtx, sig.ReturnType) {
			cx.SemanticErr("resource method return type must not include a client object type", rm.GetPosition())
			return false
		}
	}
	return true
}

// typeOpRefIndices returns the parameter indices op refers to through unions,
// intersections and arrays.
func typeOpRefIndices(op model.TypeOp) []int {
	var indices []int
	pending := []model.TypeOp{op}
	for len(pending) > 0 {
		next := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		switch o := next.(type) {
		case *model.RefTypeOp:
			indices = append(indices, o.Index)
		case *model.BinaryTypeOp:
			pending = append(pending, o.Lhs, o.Rhs)
		case *model.ArrayTypeOp:
			pending = append(pending, o.Element)
		}
	}
	return indices
}

// dependentPartsDisjoint reports whether, in every union of op with exactly
// one side depending on a typedesc parameter, the two sides are disjoint.
func dependentPartsDisjoint(tyCtx semtypes.Context, op model.TypeOp, paramTypes []semtypes.SemType) bool {
	pending := []model.TypeOp{op}
	for len(pending) > 0 {
		next := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		switch o := next.(type) {
		case *model.RefTypeOp, *model.IdentityTypeOp:
		case *model.ArrayTypeOp:
			pending = append(pending, o.Element)
		case *model.BinaryTypeOp:
			pending = append(pending, o.Lhs, o.Rhs)
			lhsDepends, rhsDepends := len(typeOpRefIndices(o.Lhs)) > 0, len(typeOpRefIndices(o.Rhs)) > 0
			if o.Kind == model.TypeOpUnion && lhsDepends != rhsDepends &&
				!semtypes.IsEmpty(tyCtx, semtypes.Intersect(o.Lhs.Apply(tyCtx, paramTypes), o.Rhs.Apply(tyCtx, paramTypes))) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validateBody(cx scope.Context, body ast.FunctionBodyNode) bool {
	switch body := body.(type) {
	case *ast.BLangBlockFunctionBody:
		validateBlock(cx, body.Stmts)
		return true
	case *ast.BLangExprFunctionBody:
		_, ok := validateActionOrExpr(cx, body.Expr, semtypes.SemType{})
		return ok
	case *ast.BLangExternFunctionBody:
		return true
	default:
		cx.InternalErr("unexpected function body", body.GetPosition())
		return false
	}
}
