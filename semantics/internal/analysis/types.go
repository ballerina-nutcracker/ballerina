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
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func dispatchTypeDesc(cx scope.Context, td ast.TypeDescriptor) bool {
	switch td := td.(type) {
	case *ast.BLangValueType, *ast.BLangBuiltInRefTypeNode, *ast.BLangUserDefinedType:
		return true
	case *ast.BLangArrayType:
		return validateArrayType(cx, td)
	case *ast.BLangFiniteTypeNode:
		return validateFiniteType(cx, td)
	case *ast.BLangUnionTypeNode:
		return validateUnionType(cx, td)
	case *ast.BLangIntersectionTypeNode:
		return validateIntersectionType(cx, td)
	case *ast.BLangErrorTypeNode:
		return dispatchTypeData(cx, &td.DetailType)
	case *ast.BLangConstrainedType:
		return validateConstrainedType(cx, td)
	case *ast.BLangStreamType:
		return validateStreamType(cx, td)
	case *ast.BLangTupleTypeNode:
		return validateTupleType(cx, td)
	case *ast.BLangRecordType:
		return validateRecordType(cx, td)
	case *ast.BLangObjectType:
		return validateObjectType(cx, td)
	case *ast.BLangFunctionType:
		return validateFunctionType(cx, td)
	default:
		cx.InternalErr("unexpected type descriptor", td.GetPosition())
		return false
	}
}

func dispatchTypeData(cx scope.Context, td *ast.TypeData) bool {
	if td.TypeDescriptor != nil {
		return dispatchTypeDesc(cx, td.TypeDescriptor)
	}
	return true
}

func validateArrayType(cx scope.Context, td *ast.BLangArrayType) bool {
	if !dispatchTypeData(cx, &td.Elemtype) {
		return false
	}
	for _, size := range td.Sizes {
		if size == nil {
			continue
		}
		if _, ok := validateActionOrExpr(cx, size, semtypes.SemType{}); !ok {
			return false
		}
	}
	return true
}

func validateFiniteType(cx scope.Context, td *ast.BLangFiniteTypeNode) bool {
	for _, value := range td.ValueSpace {
		if _, ok := validateActionOrExpr(cx, value, semtypes.SemType{}); !ok {
			return false
		}
	}
	return true
}

func validateUnionType(cx scope.Context, td *ast.BLangUnionTypeNode) bool {
	if !dispatchTypeData(cx, td.Lhs()) {
		return false
	}
	if !dispatchTypeData(cx, td.Rhs()) {
		return false
	}
	return true
}

func validateIntersectionType(cx scope.Context, td *ast.BLangIntersectionTypeNode) bool {
	if !dispatchTypeData(cx, td.Lhs()) {
		return false
	}
	if !dispatchTypeData(cx, td.Rhs()) {
		return false
	}
	return true
}

func validateConstrainedType(cx scope.Context, td *ast.BLangConstrainedType) bool {
	if !dispatchTypeData(cx, &td.BaseType) {
		return false
	}
	if !dispatchTypeData(cx, &td.Constraint) {
		return false
	}
	return true
}

func validateStreamType(cx scope.Context, td *ast.BLangStreamType) bool {
	if !dispatchTypeData(cx, &td.ValueType) {
		return false
	}
	if !dispatchTypeData(cx, &td.CompletionType) {
		return false
	}
	return true
}

func validateTupleType(cx scope.Context, td *ast.BLangTupleTypeNode) bool {
	for i := range td.Members {
		if !validateAnnotationAttachments(cx, td.Members[i].AnnAttachments) {
			return false
		}
		if !dispatchTypeDesc(cx, td.Members[i].TypeDesc) {
			return false
		}
	}
	if td.Rest != nil {
		if !dispatchTypeDesc(cx, td.Rest) {
			return false
		}
	}
	return true
}

// validateRecordType validates each inclusion, each field and the rest type
// independently: an error in one of them does not abandon the others.
func validateRecordType(cx scope.Context, td *ast.BLangRecordType) bool {
	inclusionsOK := true
	for _, inclusion := range td.TypeInclusions {
		inclusionsOK = dispatchTypeDesc(cx, inclusion) && inclusionsOK
	}
	fieldsOK := true
	for _, field := range td.FieldPtrs() {
		fieldsOK = validateRecordField(cx, field) && fieldsOK
	}
	restOK := true
	if td.RestType != nil {
		restOK = dispatchTypeDesc(cx, td.RestType)
	}
	return inclusionsOK && fieldsOK && restOK
}

// validateRecordField validates a record field; its default is a closure
// evaluated at record construction.
func validateRecordField(cx scope.Context, field *ast.BField) bool {
	if !validateAnnotationAttachments(cx, field.GetAnnotationAttachments()) {
		return false
	}
	if !dispatchTypeDesc(cx, field.Type) {
		return false
	}
	if field.Default != nil {
		closure := scope.NewExpressionClosure(cx, scope.RecordFieldDefault)
		if _, ok := validateActionOrExpr(closure, field.Default.Expr, field.Type.Type()); !ok {
			return false
		}
	}
	return true
}

// validateObjectType validates an object type's members; a non-isolated
// object may not include an isolated one.
func validateObjectType(cx scope.Context, td *ast.BLangObjectType) bool {
	membersOK := true
	for member := range td.Members() {
		membersOK = dispatchObjectMember(cx, member) && membersOK
	}
	inclusionsOK := true
	if !td.Isolated {
		for i := range td.Inclusions {
			inclusionsOK = validateObjectInclusion(cx, td, i) && inclusionsOK
		}
	}
	return membersOK && inclusionsOK
}

func validateObjectInclusion(cx scope.Context, td *ast.BLangObjectType, i int) bool {
	if semtypes.IsIsolatedObject(cx.TyCtx(), cx.CompilerCtx().SymbolType(td.Inclusions[i])) {
		cx.SemanticErr("cannot include isolated object type in non-isolated object", td.InclusionPositions[i])
		return false
	}
	return true
}

func dispatchObjectMember(cx scope.Context, member ast.ObjectMember) bool {
	switch member := member.(type) {
	case *ast.BObjectField:
		return validateObjectField(cx, member)
	case *ast.BMethodDecl:
		return validateFunctionType(cx, &member.BLangFunctionType)
	default:
		cx.InternalErr("unexpected object member", member.(ast.BLangNode).GetPosition())
		return false
	}
}

func validateObjectField(cx scope.Context, field *ast.BObjectField) bool {
	if !validateAnnotationAttachments(cx, field.GetAnnotationAttachments()) {
		return false
	}
	return dispatchTypeDesc(cx, field.Ty)
}

func validateFunctionType(cx scope.Context, td *ast.BLangFunctionType) bool {
	for _, param := range td.RequiredParams {
		if !validateFunctionTypeParam(cx, param) {
			return false
		}
	}
	if td.RestParam != nil {
		if !validateFunctionTypeParam(cx, td.RestParam) {
			return false
		}
	}
	if td.ReturnTypeDescriptor != nil {
		if !dispatchTypeDesc(cx, td.ReturnTypeDescriptor) {
			return false
		}
	}
	return true
}

func validateFunctionTypeParam(cx scope.Context, param *ast.BLangFunctionTypeParam) bool {
	if param.TypeDesc != nil {
		if !dispatchTypeDesc(cx, param.TypeDesc) {
			return false
		}
	}
	if param.InitExpr != nil {
		if _, ok := validateActionOrExpr(cx, param.InitExpr, semtypes.SemType{}); !ok {
			return false
		}
	}
	return true
}
