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
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/analysis/scope"
	semcommon "github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// validateListConstructor validates members against the inherent list type
// type resolution selected, and that unfilled fixed members have fillers.
func validateListConstructor(cx scope.Context, e *ast.BLangListConstructorExpr, expected semtypes.SemType) (scope.Facts, bool) {
	lat := e.AtomicType
	members := scope.AllFacts
	memberIndex := 0
	restMember := false
	for i, member := range e.Exprs {
		memberExpected := lat.MemberAtInnerVal(memberIndex)
		if restMember || e.IsSpreadMember(i) {
			memberExpected = lat.Rest()
		}
		if e.IsSpreadMember(i) {
			listDefn := semtypes.NewListDefinition()
			memberExpected = listDefn.Define(cx.CompilerCtx().GetTypeEnv(), nil, semtypes.ListRest(memberExpected))
		}
		memberFacts, ok := validateActionOrExpr(cx, member, memberExpected)
		if !ok {
			return scope.NoFacts, false
		}
		members = scope.Meet(members, memberFacts)
		switch {
		case e.IsSpreadMember(i):
			restMember = true
			memberIndex = lat.FixedLength()
		case !restMember:
			memberIndex++
		}
	}
	f := scope.Either(scope.FactsOfType(cx, e), members)
	tyCtx := cx.TyCtx()
	for i := memberIndex; i < lat.FixedLength(); i++ {
		memberTy := lat.MemberAtInnerVal(i)
		if _, ok := semtypes.FillerValue(tyCtx, memberTy); !ok {
			cx.SemanticErr(semcommon.FormatMissingFillerMessage(tyCtx, i, memberTy), e.GetPosition())
			return f, false
		}
	}
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// validateMappingConstructor validates field values against the inherent
// mapping type type resolution selected: keys are unique, identifier keys
// name a field of a record type, and required fields are present.
func validateMappingConstructor(cx scope.Context, e *ast.BLangMappingConstructorExpr, expected semtypes.SemType) (scope.Facts, bool) {
	mat := e.SelectedAtomicType
	fieldNames := mat.FieldNames()
	declared := make(map[string]struct{}, len(fieldNames))
	for _, name := range fieldNames {
		declared[name] = struct{}{}
	}
	present := make(map[string]struct{}, len(e.Fields)+len(e.FieldDefaults))
	for _, fd := range e.FieldDefaults {
		present[fd.FieldName] = struct{}{}
	}
	seen := make(map[string]struct{}, len(e.Fields))
	values := scope.AllFacts
	for _, field := range e.Fields {
		var kv *ast.BLangMappingKeyValueField
		switch field := field.(type) {
		case *ast.BLangMappingKeyValueField:
			kv = field
		default:
			cx.InternalErr("unexpected mapping field", field.GetPosition())
			return scope.NoFacts, false
		}
		var valueExpected semtypes.SemType
		if kv.Key.Kind == ast.MappingKeyComputed {
			if _, ok := validateActionOrExpr(cx, kv.Key.Expr, semtypes.String); !ok {
				return scope.NoFacts, false
			}
			valueExpected = semtypes.MappingMemberTypeInnerVal(cx.TyCtx(), e.Type(),
				semtypes.Intersect(kv.Key.Expr.Type(), semtypes.String))
		} else {
			keyName, ok := semcommon.MappingKeyName(cx.CompilerCtx(), kv.Key)
			if !ok {
				return scope.NoFacts, false
			}
			if _, exists := seen[keyName]; exists {
				cx.SemanticErr(strings.Join([]string{"duplicate key '", keyName, "' in mapping constructor"}, ""), kv.Key.GetPosition())
				return scope.NoFacts, false
			}
			seen[keyName] = struct{}{}
			_, isDeclared := declared[keyName]
			if kv.Key.Kind == ast.MappingKeyIdentifier && len(fieldNames) > 0 && !isDeclared {
				cx.SemanticErr(strings.Join([]string{"identifier '", keyName, "' cannot be used as a key for a rest field; use a string literal instead"}, ""), kv.Key.GetPosition())
				return scope.NoFacts, false
			}
			present[keyName] = struct{}{}
			valueExpected = mat.FieldInnerVal(keyName)
			if e.IsReadonly(keyName) {
				valueExpected = semtypes.Intersect(valueExpected, semtypes.ValReadonly)
			}
		}
		valueFacts, ok := validateActionOrExpr(cx, kv.ValueExpr, valueExpected)
		if !ok {
			return scope.NoFacts, false
		}
		values = scope.Meet(values, valueFacts)
	}
	f := scope.Either(scope.FactsOfType(cx, e), values)
	for _, name := range fieldNames {
		if _, exists := present[name]; !exists && !mat.IsOptional(cx.TyCtx(), name) {
			cx.SemanticErr(strings.Join([]string{"missing non-defaultable required record field '", name, "'"}, ""), e.GetPosition())
			return f, false
		}
	}
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}

// validateErrorConstructor validates the positional and named arguments of
// an error constructor against the error's detail type.
func validateErrorConstructor(cx scope.Context, e *ast.BLangErrorConstructorExpr, expected semtypes.SemType) (scope.Facts, bool) {
	if e.ErrorTypeRef != nil {
		if !dispatchTypeDesc(cx, e.ErrorTypeRef) {
			return scope.NoFacts, false
		}
	}
	argCount := len(e.PositionalArgs)
	if argCount < 1 || argCount > 2 {
		cx.SemanticErr("error constructor must have at least 1 and at most 2 positional arguments", e.GetPosition())
		return scope.NoFacts, false
	}
	if _, ok := validateActionOrExpr(cx, e.PositionalArgs[0], semtypes.String); !ok {
		return scope.NoFacts, false
	}
	tyCtx := cx.TyCtx()
	detailTy, ok := semtypes.ErrorDetailType(tyCtx, e.Type())
	if !ok {
		cx.UnimplementedErr("error detail type not supported", e.GetPosition())
		return scope.NoFacts, false
	}
	seen := make([]string, 0, len(e.NamedArgs))
	provided := make([]semtypes.Field, 0, len(e.NamedArgs))
	cloneable := semtypes.CreateCloneable(tyCtx)
	for i := range e.NamedArgs {
		name := e.NamedArgs[i].Name.GetValue()
		if slices.Contains(seen, name) {
			cx.SemanticErr(strings.Join([]string{"duplicate named argument '", name, "' in error constructor"}, ""), e.NamedArgs[i].GetPosition())
			return scope.NoFacts, false
		}
		seen = append(seen, name)
		fieldTy := semtypes.MappingMemberTypeInnerValProj(tyCtx, detailTy, semtypes.StringConst(name))
		if _, ok := validateActionOrExpr(cx, e.NamedArgs[i].Expr, fieldTy); !ok {
			return scope.NoFacts, false
		}
		argTy := e.NamedArgs[i].Expr.Type()
		if !semtypes.IsSubtype(tyCtx, argTy, cloneable) {
			cx.SemanticErr("named arguments must be subtypes of cloneable", e.NamedArgs[i].GetPosition())
			return scope.NoFacts, false
		}
		provided = append(provided, semtypes.FieldFrom(name, argTy, false, false))
	}
	providedDefn := semtypes.NewMappingDefinition()
	providedTy := semtypes.Intersect(providedDefn.Define(tyCtx.Env(), provided, semtypes.Never), semtypes.ValReadonly)
	if !semtypes.IsSubtype(tyCtx, providedTy, detailTy) {
		cx.SemanticErr("error detail arguments are incompatible with error detail type", e.GetPosition())
		return scope.NoFacts, false
	}
	if argCount == 2 {
		if _, ok := validateActionOrExpr(cx, e.PositionalArgs[1], semtypes.Union(semtypes.Error, semtypes.Nil)); !ok {
			return scope.NoFacts, false
		}
	}
	f := scope.FactsOfType(cx, e)
	if !validateResolvedType(cx, e, expected) {
		return f, false
	}
	return f, true
}
