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

package common

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
)

// NamedTypeRef returns the symbol of the named type typeDesc refers to, looking through arrays and
// optional types, or the zero symbol if there is none.
func NamedTypeRef(typeDesc ast.TypeDescriptor) model.SymbolRef {
	switch n := typeDesc.(type) {
	case *ast.BLangUserDefinedType:
		if ast.SymbolIsSet(n) {
			return n.Symbol()
		}
	case *ast.BLangArrayType:
		return NamedTypeRef(n.Elemtype.TypeDescriptor)
	case *ast.BLangUnionTypeNode:
		return namedTypeRefBeside(n.Lhs().TypeDescriptor, n.Rhs().TypeDescriptor, ast.TypeKindNil)
	case *ast.BLangIntersectionTypeNode:
		return namedTypeRefBeside(n.Lhs().TypeDescriptor, n.Rhs().TypeDescriptor, ast.TypeKindReadOnly)
	}
	return model.SymbolRef{}
}

// namedTypeRefBeside returns the named type of whichever of lhs and rhs sits beside a value type of
// kind, as in `T?` or `readonly & T`.
func namedTypeRefBeside(lhs, rhs ast.TypeDescriptor, kind ast.TypeKind) model.SymbolRef {
	if isValueTypeNode(lhs, kind) {
		return NamedTypeRef(rhs)
	}
	if isValueTypeNode(rhs, kind) {
		return NamedTypeRef(lhs)
	}
	return model.SymbolRef{}
}

func isValueTypeNode(typeDesc ast.TypeDescriptor, kind ast.TypeKind) bool {
	valueType, ok := typeDesc.(*ast.BLangValueType)
	return ok && valueType.TypeKind == kind
}

// NamedTypes skips references that are not types, such as the typedesc parameter `t` in a
// dependently typed `returns t|error`.
func NamedTypes(cx *context.CompilerContext, typeDesc ast.TypeDescriptor) []model.SymbolRef {
	return appendNamedTypes(cx, nil, typeDesc)
}

func appendNamedTypes(cx *context.CompilerContext, namedTypes []model.SymbolRef, typeDesc ast.TypeDescriptor) []model.SymbolRef {
	switch n := typeDesc.(type) {
	case *ast.BLangReturnTypeDescriptor:
		return appendNamedTypes(cx, namedTypes, n.TypeDescriptor)
	case *ast.BLangUnionTypeNode:
		namedTypes = appendNamedTypes(cx, namedTypes, n.Lhs().TypeDescriptor)
		return appendNamedTypes(cx, namedTypes, n.Rhs().TypeDescriptor)
	}
	if ref := NamedTypeRef(typeDesc); !ref.IsEmpty() && cx.SymbolKind(ref) == model.SymbolKindType {
		return append(namedTypes, ref)
	}
	return namedTypes
}
