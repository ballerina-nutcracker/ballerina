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

// ClassFieldDescriptor returns the descriptor of a field declared in a class
// or service body, whose type is ty.
func ClassFieldDescriptor(field *ast.BLangVariable, ty semtypes.SemType) model.FieldDescriptor {
	var flags model.FieldDescriptorFlag
	if field.IsFinal() {
		flags |= model.FieldDescriptorFinal
	}
	if field.Expr != nil {
		flags |= model.FieldDescriptorHasDefault
	}
	fd := model.NewFieldDescriptor(field.Name.GetValue(), flags, field.IsPublic())
	fd.SetMemberType(ty)
	return fd
}

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

func IterableType(ctx *context.CompilerContext) (semtypes.SemType, bool) {
	ref, ok := ctx.LangLibDistinctTypeSymbol("lang.object", "Iterable")
	if !ok {
		return semtypes.SemType{}, false
	}
	return ctx.SymbolType(ref), true
}

// DeclaredNodes returns the nodes that declare a symbol. A declaration with a bad name declares none, so it has
// no type.
func DeclaredNodes[N ast.NodeWithSymbol](nodes []N) []N {
	return slices.DeleteFunc(slices.Clone(nodes), func(node N) bool { return !ast.SymbolIsSet(node) })
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

// MappingKeyName returns the field name of a literal or identifier key. A
// computed key has no static name; its value is only known at runtime.
func MappingKeyName(ctx *context.CompilerContext, key *ast.BLangMappingKey) (string, bool) {
	if key.Kind == ast.MappingKeyComputed {
		ctx.InternalError("computed mapping key has no static name", key.GetPosition())
		return "", false
	}
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		return expr.Value.(string), true
	case *ast.BLangVarRef:
		return expr.VariableName.GetValue(), true
	default:
		ctx.InternalError(fmt.Sprintf("unexpected record key expression type: %T", key.Expr), key.GetPosition())
		return "", false
	}
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
