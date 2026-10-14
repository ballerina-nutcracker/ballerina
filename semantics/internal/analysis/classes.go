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
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// validateClass validates a class's members and, for a non-isolated class,
// each inclusion; an error in one does not abandon the others.
func validateClass(cx scope.Context, c *ast.BLangClassDefinition) {
	info := scope.NewClassInfo(cx, c.Name.GetValue(), c.IsIsolated(), c.Fields,
		memberFieldDescriptors(cx.CompilerCtx().SymbolMembers(c.Symbol())), c.InitFunction)
	validateClassInner(cx, info, c.Fields, c.InitFunction, common.MethodsInResolutionOrder(c.Methods), c.ResourceMethods)
	if !c.IsIsolated() {
		for i := range c.Inclusions {
			validateInclusionNotIsolated(cx, c.Inclusions[i], c.InclusionPositions[i])
		}
	}
}

// validateInclusionNotIsolated: a non-isolated object may not include an
// isolated object type.
func validateInclusionNotIsolated(cx scope.Context, inclusion model.SymbolRef, pos diagnostics.Location) {
	if semtypes.IsIsolatedObject(cx.TyCtx(), cx.CompilerCtx().SymbolType(inclusion)) {
		cx.SemanticErr("cannot include isolated object type in non-isolated object", pos)
	}
}

// validateService validates the attached listener expressions, then the
// members and, for a non-isolated service, each inclusion; an error in a
// listener does not abandon the members.
func validateService(cx scope.Context, svc *ast.BLangService) {
	for i := range svc.AttachedExprs {
		validateServiceListener(cx, svc, i)
	}
	info := scope.NewClassInfo(cx, "", svc.IsIsolated(), svc.Fields, serviceFieldDescriptors(cx, svc), svc.InitFunction)
	validateClassInner(cx, info, svc.Fields, svc.InitFunction, common.MethodsInResolutionOrder(svc.Methods), svc.ResourceMethods)
	if !svc.IsIsolated() {
		for i := range svc.Inclusions {
			validateInclusionNotIsolated(cx, svc.Inclusions[i], svc.InclusionPositions[i])
		}
	}
}

func memberFieldDescriptors(members []model.InclusionMember) []*model.FieldDescriptor {
	var fields []*model.FieldDescriptor
	for _, member := range members {
		if field, ok := member.(*model.FieldDescriptor); ok {
			fields = append(fields, field)
		}
	}
	return fields
}

// serviceFieldDescriptors describes the fields of svc. A service cannot
// include types, so its fields are the ones it declares.
func serviceFieldDescriptors(cx scope.Context, svc *ast.BLangService) []*model.FieldDescriptor {
	fieldDescriptors := make([]*model.FieldDescriptor, len(svc.Fields))
	for i, field := range svc.Fields {
		fd := common.ClassFieldDescriptor(field, cx.CompilerCtx().SymbolType(field.Symbol()))
		fieldDescriptors[i] = &fd
	}
	return fieldDescriptors
}

// validateServiceListener validates an attached expression and checks that it
// is a listener that supports the service's type and attach point.
func validateServiceListener(cx scope.Context, svc *ast.BLangService, i int) {
	expr := svc.AttachedExprs[i]
	if _, ok := validateActionOrExpr(cx, expr, semtypes.SemType{}); !ok {
		return
	}
	serviceTy := svc.GetTypeData().Type
	attachPointTy := svc.AttachPointType
	if semtypes.IsZero(serviceTy) || semtypes.IsZero(attachPointTy) {
		cx.InternalErr("service types not resolved", svc.GetPosition())
		return
	}
	tyCtx := cx.TyCtx()
	listenerTy := semtypes.Diff(expr.Type(), semtypes.Error)
	if semtypes.IsNever(listenerTy) {
		cx.SemanticErr("expression in 'on' clause is not a listener", expr.GetPosition())
		return
	}
	listenerServiceTy, listenerAttachPointTy, ok := common.ListenerTypes(tyCtx, listenerTy, common.ListenerAttachPointBound(tyCtx))
	if !ok {
		cx.SemanticErr("expression in 'on' clause is not a listener", expr.GetPosition())
		return
	}
	if !semtypes.IsSubtype(tyCtx, serviceTy, listenerServiceTy) {
		cx.SemanticErr("service type is not supported by listener", expr.GetPosition())
		return
	}
	if !semtypes.IsSubtype(tyCtx, attachPointTy, listenerAttachPointTy) {
		cx.SemanticErr("attach point is not supported by listener", expr.GetPosition())
		return
	}
	if svc.AbsoluteResourcePath != nil {
		segments := make([]string, len(svc.AbsoluteResourcePath))
		for i := range svc.AbsoluteResourcePath {
			segments[i] = svc.AbsoluteResourcePath[i].Value
		}
		if !hasSingleApplicableAttachPointListType(tyCtx, segments, listenerAttachPointTy) {
			cx.SemanticErr("attach point does not have a single applicable listener list type", expr.GetPosition())
		}
	}
}

func hasSingleApplicableAttachPointListType(tyCtx semtypes.Context, segments []string, attachPointTy semtypes.SemType) bool {
	members := make([]semtypes.ListMemberInfo, len(segments))
	for i, segment := range segments {
		members[i] = semtypes.ListMemberInfo{Index: i, ValueType: semtypes.StringConst(segment)}
	}
	found := false
	for _, alt := range semtypes.ListAlternatives(tyCtx, semtypes.Intersect(attachPointTy, semtypes.List)) {
		if !semtypes.ListAlternativeAllowsMembers(tyCtx, alt, members) {
			continue
		}
		if found {
			return false
		}
		found = true
	}
	return found
}

// validateClassInner validates the members of a class or service body; a
// member that reports is abandoned and the next one runs.
func validateClassInner(cx scope.Context, info scope.ClassInfo, fields []*ast.BLangVariable, initFn *ast.BLangFunction,
	methods []common.NamedClassMethod, resourceMethods []*ast.BLangResourceMethod) {
	for _, field := range fields {
		validateVariable(cx, field, cx.CompilerCtx().SymbolType(field.Symbol()))
	}
	if initFn != nil {
		validateInvokableNode(scope.NewMethod(cx, initFn, info), initFn)
	}
	for _, named := range methods {
		validateInvokableNode(scope.NewMethod(cx, named.Method, info), named.Method)
	}
	for _, rm := range resourceMethods {
		validateInvokableNode(scope.NewMethod(cx, rm, info), rm)
	}
	if info.IsIsolated() {
		for _, field := range fields {
			validateIsolatedClassField(cx, field)
		}
	}
	for i := 1; i < len(resourceMethods); i++ {
		validateUniqueResourceMethod(cx, resourceMethods, i)
	}
}

// validateIsolatedClassField: a public field of an isolated object must be
// final and of a readonly type.
func validateIsolatedClassField(cx scope.Context, field *ast.BLangVariable) {
	if field.IsPublic() && !scope.IsImmutableField(cx, field) {
		cx.SemanticErr("public field of an isolated object must be \"final\" and have a type that is a subtype of \"readonly\"", field.GetPosition())
	}
}

// validateUniqueResourceMethod: rms[i] may not have the same method name and
// path type as an earlier resource method.
func validateUniqueResourceMethod(cx scope.Context, rms []*ast.BLangResourceMethod, i int) {
	ctx := cx.CompilerCtx()
	later, ok := ctx.GetSymbol(rms[i].Symbol()).(model.ResourceMethodSymbol)
	if !ok {
		cx.InternalErr("expected resource method symbol", rms[i].GetPosition())
		return
	}
	for j := 0; j < i; j++ {
		earlier, ok := ctx.GetSymbol(rms[j].Symbol()).(model.ResourceMethodSymbol)
		if !ok {
			cx.InternalErr("expected resource method symbol", rms[j].GetPosition())
			return
		}
		if later.MethodName() == earlier.MethodName() && semtypes.IsSameType(cx.TyCtx(), later.PathListType(), earlier.PathListType()) {
			cx.SemanticErr(strings.Join([]string{"duplicate resource method '", later.MethodName(), "'"}, ""), rms[i].GetPosition())
			return
		}
	}
}
