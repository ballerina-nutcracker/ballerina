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
	"fmt"
	"maps"
	"slices"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func IsSelfFieldAccess(n *ast.BLangFieldBaseAccess) bool {
	varRef, ok := n.Expr.(*ast.BLangVarRef)
	return ok && varRef.VariableName.GetValue() == "self"
}

// AssignmentTargetError returns the reason a symbol of the given kind cannot be assigned to, if any.
func AssignmentTargetError(kind model.SymbolKind) (string, bool) {
	switch kind {
	case model.SymbolKindConstant:
		return "cannot assign to constant", true
	case model.SymbolKindParemeter:
		return "cannot assign to parameter", true
	case model.SymbolKindFunction:
		return "cannot assign to function", true
	case model.SymbolKindType:
		return "cannot assign to type", true
	default:
		return "", false
	}
}

func IterableType(ctx *context.CompilerContext, symbolType func(model.SymbolRef) semtypes.SemType) (semtypes.SemType, bool) {
	ref, ok := ctx.LangLibDistinctTypeSymbol("lang.object", "Iterable")
	if !ok {
		return semtypes.SemType{}, false
	}
	return symbolType(ref), true
}

type NamedClassMethod struct {
	Name   string
	Method *ast.BLangFunction
}

func MethodsInResolutionOrder(methods map[string]*ast.BLangFunction) []NamedClassMethod {
	names := slices.Sorted(maps.Keys(methods))
	result := make([]NamedClassMethod, len(names))
	for i, name := range names {
		result[i] = NamedClassMethod{Name: name, Method: methods[name]}
	}
	return result
}

func IsNumericType(ctx semtypes.Context, ty semtypes.SemType) bool {
	return semtypes.IsSubtype(ctx, ty, semtypes.Number)
}

func MapQuerySelectExpectedType(env semtypes.Env) semtypes.SemType {
	return mapQuerySelectExpectedTypeWithValue(env, semtypes.Union(semtypes.Any, semtypes.Error))
}

func QuerySelectExpectedType(ctx semtypes.Context, env semtypes.Env, queryConstructType ast.TypeKind, expectedType semtypes.SemType) semtypes.SemType {
	switch queryConstructType {
	case ast.TypeKindNone:
		return listQuerySelectExpectedType(ctx, expectedType)
	case ast.TypeKindMap:
		return mapQuerySelectExpectedTypeFromQueryExpectedType(ctx, env, expectedType)
	default:
		return semtypes.SemType{}
	}
}

func listQuerySelectExpectedType(ctx semtypes.Context, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return semtypes.SemType{}
	}
	listTy := semtypes.Intersect(expectedType, semtypes.List)
	if semtypes.IsEmpty(ctx, listTy) {
		return semtypes.SemType{}
	}
	memberTypes := semtypes.ListAllMemberTypesInner(ctx, listTy)
	result := semtypes.Never
	for _, memberTy := range memberTypes.Types {
		result = semtypes.Union(result, memberTy)
	}
	if semtypes.IsEmpty(ctx, result) {
		return semtypes.SemType{}
	}
	return result
}

func mapQuerySelectExpectedTypeFromQueryExpectedType(ctx semtypes.Context, env semtypes.Env, expectedType semtypes.SemType) semtypes.SemType {
	if semtypes.IsZero(expectedType) {
		return semtypes.SemType{}
	}
	mappingTy := semtypes.Intersect(expectedType, semtypes.Mapping)
	if semtypes.IsEmpty(ctx, mappingTy) {
		return semtypes.SemType{}
	}
	valueTy := semtypes.MappingMemberTypeInnerValProj(ctx, mappingTy, semtypes.String)
	if semtypes.IsSubtype(ctx, semtypes.CreateAnydata(ctx), valueTy) {
		return semtypes.SemType{}
	}
	return mapQuerySelectExpectedTypeWithValue(env, valueTy)
}

func mapQuerySelectExpectedTypeWithValue(env semtypes.Env, valueTy semtypes.SemType) semtypes.SemType {
	ld := semtypes.NewListDefinition()
	return ld.Define(env, []semtypes.SemType{semtypes.String, valueTy})
}

func MappingKeyName(ctx *context.CompilerContext, key *ast.BLangMappingKey) (string, bool) {
	if lit, ok := key.Expr.(*ast.BLangLiteral); ok && key.Kind != ast.MappingKeyComputed {
		return lit.Value.(string), true
	}
	ctx.InternalError(fmt.Sprintf("mapping key is not a field name literal: %T", key.Expr), key.GetPosition())
	return "", false
}

// ComputedMappingFieldType returns the type inherentType requires for the value of a computed name
// field whose key has type keyTy. A key that is not a string is an error reported on the key; the
// value is then resolved as if the key were any string so that no follow-on error is reported.
func ComputedMappingFieldType(cx semtypes.Context, inherentType, keyTy semtypes.SemType) semtypes.SemType {
	if !semtypes.IsSubtype(cx, keyTy, semtypes.String) {
		keyTy = semtypes.String
	}
	return semtypes.MappingMemberTypeInnerValProj(cx, inherentType, keyTy)
}

func FormatIncompatibleTypeMessage(ctx semtypes.Context, expectedType, actualType semtypes.SemType) string {
	return fmt.Sprintf("incompatible type: expected %s, got %s", semtypes.ToString(ctx, expectedType), semtypes.ToString(ctx, actualType))
}

func FormatMissingFillerMessage(ctx semtypes.Context, index int, memberType semtypes.SemType) string {
	return fmt.Sprintf("missing required member at index %d: type '%s' has no filler value", index, semtypes.ToString(ctx, memberType))
}

var TemplateInsertionAllowedTypes = semtypes.Diff(semtypes.SimpleOrString, semtypes.Nil)

func XMLTemplateInsertionAllowedTypes(kind ast.XMLTemplateInsertionKind) semtypes.SemType {
	if kind == ast.XMLTemplateInsertionKindContent {
		return semtypes.Union(TemplateInsertionAllowedTypes, semtypes.XML)
	}
	return TemplateInsertionAllowedTypes
}
