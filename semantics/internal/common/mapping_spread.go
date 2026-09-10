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
	"slices"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// MappingConstructorEntry describes one resolved field of a mapping constructor: either a
// specific field with its decoded key and value type, or a spread with its operand type.
type MappingConstructorEntry struct {
	Spread     bool
	Name       string
	Type       semtypes.SemType
	Shape      semtypes.MappingSpreadShape
	Pos        diagnostics.Location
	IsReadonly bool
}

// MappingConstructorEntries decodes the constructor's fields into the resolved description the
// spread checks work on. Spread operands must already be known to be mappings.
func MappingConstructorEntries(ctx *context.CompilerContext, cx semtypes.Context,
	expr *ast.BLangMappingConstructorExpr) ([]MappingConstructorEntry, bool) {
	entries := make([]MappingConstructorEntry, 0, len(expr.Fields))
	for _, field := range expr.Fields {
		switch f := field.(type) {
		case *ast.BLangMappingKeyValueField:
			name, ok := MappingKeyName(ctx, f.Key)
			if !ok {
				return nil, false
			}
			entries = append(entries, MappingConstructorEntry{
				Name:       name,
				Type:       f.ValueExpr.GetDeterminedType(),
				Pos:        f.GetPosition(),
				IsReadonly: expr.IsReadonly(name),
			})
		case *ast.BLangMappingSpreadField:
			operandTy := f.Expr.GetDeterminedType()
			entries = append(entries, MappingConstructorEntry{
				Spread: true,
				Type:   operandTy,
				Shape:  semtypes.MappingShapeForSpread(cx, operandTy),
				Pos:    f.GetPosition(),
			})
		default:
			ctx.InternalError(fmt.Sprintf("unexpected mapping field kind %T", field), field.GetPosition())
			return nil, false
		}
	}
	return entries, true
}

// HasSpreadField reports whether the constructor contains any spread field.
func HasSpreadField(expr *ast.BLangMappingConstructorExpr) bool {
	return slices.ContainsFunc(expr.Fields, func(f ast.MappingField) bool {
		_, ok := f.(*ast.BLangMappingSpreadField)
		return ok
	})
}

// MappingConstructorDiagnostic is a message attached to the field that caused it.
type MappingConstructorDiagnostic struct {
	Message string
	Pos     diagnostics.Location
}

// CheckMappingConstructorKeys reports statically possible duplicate keys between specific fields
// and spreads, and between pairs of spreads. Textual order does not matter: a key a spread can
// supply conflicts with a specific field written either before or after it.
func CheckMappingConstructorKeys(cx semtypes.Context, entries []MappingConstructorEntry) (MappingConstructorDiagnostic, bool) {
	for i, entry := range entries {
		if !entry.Spread {
			continue
		}
		for j, other := range entries {
			if i == j {
				continue
			}
			if !other.Spread {
				if semtypes.MappingSpreadAllowsKey(cx, entry.Type, other.Name) {
					return MappingConstructorDiagnostic{
						Message: fmt.Sprintf("spread field may duplicate key '%s'", other.Name),
						Pos:     entry.Pos,
					}, false
				}
				continue
			}
			if j < i {
				continue
			}
			name, overlap := semtypes.MappingSpreadsOverlap(cx, entry.Type, other.Type)
			if !overlap {
				continue
			}
			message := "spread fields may supply overlapping keys"
			if name != "" {
				message = fmt.Sprintf("spread fields may duplicate key '%s'", name)
			}
			return MappingConstructorDiagnostic{Message: message, Pos: other.Pos}, false
		}
	}
	return MappingConstructorDiagnostic{}, true
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

// CheckMappingConstructorAgainstTarget validates the resolved constructor entries against the
// selected inherent type: every possible member must be admitted by the corresponding declared
// field or by the target rest, and every required target field must be guaranteed a value.
func CheckMappingConstructorAgainstTarget(cx semtypes.Context, entries []MappingConstructorEntry,
	target *semtypes.MappingAtomicType, defaults []string, pos diagnostics.Location) (MappingConstructorDiagnostic, bool) {
	guaranteed := map[string]bool{}
	for _, name := range defaults {
		guaranteed[name] = true
	}
	targetRest := target.RestInnerVal()
	for _, entry := range entries {
		if !entry.Spread {
			guaranteed[entry.Name] = true
			expected := target.FieldInnerVal(entry.Name)
			if entry.IsReadonly {
				expected = semtypes.Intersect(expected, semtypes.ValReadonly)
			}
			if !semtypes.MappingFieldTypeAllowed(cx, entry.Type, expected) {
				return MappingConstructorDiagnostic{
					Message: fmt.Sprintf("field '%s' is not allowed by the inherent type of the mapping constructor", entry.Name),
					Pos:     entry.Pos,
				}, false
			}
			continue
		}
		if diag, ok := checkSpreadAgainstTarget(cx, entry, target, targetRest, guaranteed); !ok {
			return diag, false
		}
	}
	for _, name := range target.FieldNames() {
		if guaranteed[name] || target.IsOptional(cx, name) {
			continue
		}
		return MappingConstructorDiagnostic{
			Message: fmt.Sprintf("missing non-defaultable required record field '%s'", name),
			Pos:     pos,
		}, false
	}
	return MappingConstructorDiagnostic{}, true
}

func checkSpreadAgainstTarget(cx semtypes.Context, entry MappingConstructorEntry,
	target *semtypes.MappingAtomicType, targetRest semtypes.SemType, guaranteed map[string]bool) (MappingConstructorDiagnostic, bool) {
	shape := entry.Shape
	if !shape.Inhabited {
		return MappingConstructorDiagnostic{}, true
	}
	excluded := map[string]bool{}
	for _, field := range shape.Fields {
		if semtypes.IsNever(field.Type()) {
			excluded[field.Name()] = true
			continue
		}
		expected := target.FieldInnerVal(field.Name())
		if !semtypes.IsSubtype(cx, field.Type(), expected) {
			return MappingConstructorDiagnostic{
				Message: fmt.Sprintf("spread field member '%s' is not allowed by the inherent type of the mapping constructor", field.Name()),
				Pos:     entry.Pos,
			}, false
		}
		if !field.Optional() {
			guaranteed[field.Name()] = true
		}
	}
	if semtypes.IsNever(shape.Rest) {
		return MappingConstructorDiagnostic{}, true
	}
	if !semtypes.IsSubtype(cx, shape.Rest, targetRest) {
		return MappingConstructorDiagnostic{
			Message: "spread field may supply keys the inherent type of the mapping constructor does not allow",
			Pos:     entry.Pos,
		}, false
	}
	// The source rest can also land on a target field the source does not distinguish, so that
	// contribution has to satisfy the declared field type as well.
	for _, name := range target.FieldNames() {
		if excluded[name] || shapeDistinguishes(shape, name) {
			continue
		}
		if !semtypes.IsSubtype(cx, shape.Rest, target.FieldInnerVal(name)) {
			return MappingConstructorDiagnostic{
				Message: fmt.Sprintf("spread field may supply a value for '%s' that the inherent type does not allow", name),
				Pos:     entry.Pos,
			}, false
		}
	}
	return MappingConstructorDiagnostic{}, true
}

func shapeDistinguishes(shape semtypes.MappingSpreadShape, name string) bool {
	return slices.ContainsFunc(shape.Fields, func(f semtypes.Field) bool { return f.Name() == name })
}
