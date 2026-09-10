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
	"slices"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// HasSpreadField reports whether the constructor contains any spread field.
func HasSpreadField(expr *ast.BLangMappingConstructorExpr) bool {
	return slices.ContainsFunc(expr.Fields, func(f ast.MappingField) bool {
		_, ok := f.(*ast.BLangMappingSpreadField)
		return ok
	})
}

// MappingSpreadContextualType is the mapping type used to contextually type a spread operand of
// a constructor whose inherent type is target. It encompasses every value the target can hold,
// so nested operand expressions get a member context without bypassing the per-key checks. A
// nested constructor operand supplies exactly its own keys, so it is typed against those keys
// alone: an open map would replace its inferred shape with one that can supply any key.
func MappingSpreadContextualType(ctx *context.CompilerContext, env semtypes.Env,
	target *semtypes.MappingAtomicType, operand ast.BLangExpression) semtypes.SemType {
	if fields, ok := mappingConstructorOperandFields(ctx, target, operand); ok {
		return semtypes.MappingOfFieldTypes(env, fields)
	}
	return semtypes.MappingOfMemberType(env, target.AllMemberInnerVal())
}

// mappingConstructorOperandFields are the keys a constructor operand is known to supply, each
// with the type target holds for it. A spread field, a computed key or a repeated key leaves the
// supplied keys unknown; a key target cannot hold at all is left to the per-key checks, which
// report it against the spread field rather than against the operand.
func mappingConstructorOperandFields(ctx *context.CompilerContext, target *semtypes.MappingAtomicType,
	operand ast.BLangExpression) ([]semtypes.MappingFieldInfo, bool) {
	expr, ok := operand.(*ast.BLangMappingConstructorExpr)
	if !ok || HasSpreadField(expr) {
		return nil, false
	}
	seen := map[string]bool{}
	fields := make([]semtypes.MappingFieldInfo, 0, len(expr.Fields))
	for _, f := range expr.Fields {
		field, ok := f.(*ast.BLangMappingKeyValueField)
		if !ok || field.Key.Kind == ast.MappingKeyComputed {
			return nil, false
		}
		name, ok := MappingKeyName(ctx, field.Key)
		if !ok || seen[name] {
			return nil, false
		}
		seen[name] = true
		fieldTy := target.FieldInnerVal(name)
		if semtypes.IsNever(fieldTy) {
			return nil, false
		}
		fields = append(fields, semtypes.MappingFieldInfo{Name: name, Type: fieldTy})
	}
	return fields, true
}
