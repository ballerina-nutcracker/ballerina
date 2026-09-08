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

package types

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semantics/internal/common"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const notConstantExpressionMessage = "expression is not a constant expression"

// isConstantExpression answers whether expr is structurally a constant
// expression (spec §6.4), returning the first subexpression that is not. It
// emits no diagnostic: a non-constant value on a non-const annotation is legal,
// so that caller needs the answer without one.
func isConstantExpression(t typeResolver, expr ast.BLangExpression) (ast.BLangExpression, bool) {
	switch e := expr.(type) {
	case *ast.BLangLiteral, *ast.BLangNumericLiteral, *ast.BLangConstRef:
		return nil, true
	case *ast.BLangVarRef:
		if vs, ok := t.getSymbol(e.Symbol()).(model.ValueSymbol); ok && vs.IsConst() {
			return nil, true
		}
		return expr, false
	case *ast.BLangUnaryExpr:
		return isConstantExpression(t, e.Expr)
	case *ast.BLangTypeConversionExpr:
		return isConstantExpression(t, e.Expression)
	case *ast.BLangGroupExpr:
		return isConstantExpression(t, e.Expression)
	case *ast.BLangBinaryExpr:
		if offender, ok := isConstantExpression(t, e.LhsExpr); !ok {
			return offender, false
		}
		return isConstantExpression(t, e.RhsExpr)
	case *ast.BLangTernaryExpr:
		if offender, ok := isConstantExpression(t, e.Condition); !ok {
			return offender, false
		}
		if offender, ok := isConstantExpression(t, e.ThenExpr); !ok {
			return offender, false
		}
		return isConstantExpression(t, e.ElseExpr)
	case *ast.BLangNilConditionalExpr:
		if offender, ok := isConstantExpression(t, e.LhsExpr); !ok {
			return offender, false
		}
		return isConstantExpression(t, e.RhsExpr)
	case *ast.BLangListConstructorExpr:
		for _, member := range e.Exprs {
			if offender, ok := isConstantExpression(t, member); !ok {
				return offender, false
			}
		}
		return nil, true
	case *ast.BLangMappingConstructorExpr:
		for _, field := range e.Fields {
			kv, isKeyValue := field.(*ast.BLangMappingKeyValueField)
			if !isKeyValue {
				continue
			}
			if kv.Key != nil && kv.Key.Kind == ast.MappingKeyComputed {
				if offender, ok := isConstantExpression(t, kv.Key.Expr); !ok {
					return offender, false
				}
			}
			if offender, ok := isConstantExpression(t, kv.ValueExpr); !ok {
				return offender, false
			}
		}
		return nil, true
	case *ast.BLangTemplateExpr:
		for _, insertion := range e.Insertions {
			if offender, ok := isConstantExpression(t, insertion); !ok {
				return offender, false
			}
		}
		return nil, true
	case *ast.BLangXMLTemplateExpr:
		for _, insertion := range e.Insertions {
			if offender, ok := isConstantExpression(t, insertion); !ok {
				return offender, false
			}
		}
		return nil, true
	default:
		return expr, false
	}
}

type constantExpressionEvaluator struct {
	resolver    typeResolver
	unsupported *unsupportedConstantForm
}

// unsupportedConstantForm is a constant expression form we cannot fold yet. It
// is not a fault in the program, so whether it is an error depends on the
// caller: a non-const annotation evaluates such a value at runtime instead.
type unsupportedConstantForm struct {
	message string
	loc     ast.Location
}

func (u *unsupportedConstantForm) report(t typeResolver) {
	t.unimplemented(u.message, u.loc)
}

// foldConstant evaluates a constant expression at compile time (spec §6.4). It
// is called only on an expression isConstantExpression has accepted, so a
// failure is never structural: it reports its own diagnostic at the failing
// subexpression and returns false. A nil value with ok is a successful fold of
// (), which is why failure is signalled by the bool alone.
func foldConstant(t typeResolver, expr ast.BLangExpression) (values.BalValue, bool) {
	value, unsupported, ok := tryFoldConstant(t, expr)
	if unsupported != nil {
		unsupported.report(t)
	}
	return value, ok
}

// tryFoldConstant is foldConstant for a caller that has a fallback when expr
// uses a form we cannot fold yet: that failure is returned as unsupported
// rather than reported. Every other failure is reported as in foldConstant.
func tryFoldConstant(t typeResolver, expr ast.BLangExpression) (value values.BalValue,
	unsupported *unsupportedConstantForm, ok bool,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.internalError(fmt.Sprintf("constant expression evaluation panicked: %v", recovered), expr.GetPosition())
			value, unsupported, ok = nil, nil, false
		}
	}()
	evaluator := constantExpressionEvaluator{resolver: t}
	value, ok = evaluator.evaluate(expr)
	return value, evaluator.unsupported, ok
}

func (e *constantExpressionEvaluator) semanticFailure(message string, loc ast.Location) (values.BalValue, bool) {
	e.resolver.semanticError(message, loc)
	return nil, false
}

func (e *constantExpressionEvaluator) unsupportedFailure(message string, loc ast.Location) (values.BalValue, bool) {
	e.unsupported = &unsupportedConstantForm{message: message, loc: loc}
	return nil, false
}

func (e *constantExpressionEvaluator) internalFailure(message string, loc ast.Location) (values.BalValue, bool) {
	e.resolver.internalError(message, loc)
	return nil, false
}

func (e *constantExpressionEvaluator) evaluate(expr ast.BLangExpression) (values.BalValue, bool) {
	switch expr := expr.(type) {
	case *ast.BLangLiteral:
		return expr.Value, true
	case *ast.BLangNumericLiteral:
		return expr.Value, true
	case *ast.BLangGroupExpr:
		return e.evaluate(expr.Expression)
	case *ast.BLangVarRef:
		return e.evaluateConstantReference(expr.Symbol(), expr.GetPosition())
	case *ast.BLangConstRef:
		return e.evaluateConstantReference(expr.Symbol(), expr.GetPosition())
	case *ast.BLangMappingConstructorExpr:
		return e.evaluateMappingConstructor(expr)
	case *ast.BLangListConstructorExpr:
		return e.evaluateListConstructor(expr)
	case *ast.BLangUnaryExpr:
		return e.evaluateUnaryExpression(expr)
	case *ast.BLangTernaryExpr:
		return e.evaluateTernaryExpression(expr)
	case *ast.BLangNilConditionalExpr:
		return e.evaluateNilConditionalExpression(expr)
	case *ast.BLangBinaryExpr:
		return e.evaluateBinaryExpression(expr)
	case *ast.BLangTypeConversionExpr:
		return e.evaluateTypeConversion(expr)
	case *ast.BLangTemplateExpr:
		return e.evaluateStringTemplate(expr)
	case *ast.BLangXMLTemplateExpr:
		return e.unsupportedFailure("constant xml template not implemented", expr.GetPosition())
	default:
		return e.internalFailure(fmt.Sprintf("unexpected constant expression %T", expr), expr.GetPosition())
	}
}

func (e *constantExpressionEvaluator) evaluateConstantReference(ref model.SymbolRef, loc ast.Location) (values.BalValue, bool) {
	ref = e.resolver.unnarrowedSymbol(ref)
	sym, ok := e.resolver.getSymbol(ref).(*model.ConstantValueSymbol)
	if !ok {
		return e.semanticFailure(notConstantExpressionMessage, loc)
	}
	return sym.ConstantValue(), true
}

func (e *constantExpressionEvaluator) evaluateMappingConstructor(expr *ast.BLangMappingConstructorExpr) (values.BalValue, bool) {
	entries := make([]values.MapEntry, 0, len(expr.Fields))
	for _, field := range expr.Fields {
		kv, ok := field.(*ast.BLangMappingKeyValueField)
		if !ok {
			return e.unsupportedFailure("constant mapping spread field not implemented", field.GetPosition())
		}
		key, ok := e.constantMappingKey(kv)
		if !ok {
			return nil, false
		}
		value, ok := e.evaluate(kv.ValueExpr)
		if !ok {
			return nil, false
		}
		entries = append(entries, values.MapEntry{Key: key, Value: value})
	}

	return e.readonlyMappingValue(entries, expr.GetPosition())
}

// The type of a constant is the intersection of readonly and the singleton type
// containing just the shape of its value (spec §8.8). readonlyListShapeType and
// readonlyMappingShapeType build that type from the evaluated members, so it is
// a single list or mapping atom rather than an intersection: both the BIR
// deserializer and the inherent type checks on the value require an atomic type.
func readonlyListShapeType(t typeResolver, members []values.BalValue) (semtypes.SemType,
	*semtypes.ListAtomicType, error,
) {
	memberTypes := make([]semtypes.SemType, len(members))
	for i, member := range members {
		memberTypes[i] = values.SemTypeForValue(member)
	}
	ld := semtypes.NewListDefinition()
	ty := ld.Define(t.typeEnv(), memberTypes, semtypes.ListMutability(semtypes.CellMutabilityNone))
	atomic := semtypes.ToListAtomicType(t.typeEnv(), ty)
	if atomic == nil {
		return semtypes.SemType{}, nil, fmt.Errorf("constant list type is not atomic")
	}
	return ty, atomic, nil
}

func readonlyMappingShapeType(t typeResolver, entries []values.MapEntry) (semtypes.SemType,
	*semtypes.MappingAtomicType, error,
) {
	fields := make([]semtypes.Field, 0, len(entries))
	for _, entry := range entries {
		fields = append(fields, semtypes.FieldFrom(entry.Key, values.SemTypeForValue(entry.Value), true, false))
	}
	md := semtypes.NewMappingDefinition()
	ty := md.Define(t.typeEnv(), fields, semtypes.Never, semtypes.MappingMutability(semtypes.CellMutabilityNone))
	atomic := semtypes.ToMappingAtomicType(t.typeContext(), ty)
	if atomic == nil {
		return semtypes.SemType{}, nil, fmt.Errorf("constant mapping type is not atomic")
	}
	return ty, atomic, nil
}

func (e *constantExpressionEvaluator) constantMappingKey(field *ast.BLangMappingKeyValueField) (string, bool) {
	key := field.Key
	if key == nil || key.Expr == nil {
		e.resolver.internalError("constant mapping field has no key", field.GetPosition())
		return "", false
	}
	if key.Kind == ast.MappingKeyComputed {
		e.unsupportedFailure("constant computed mapping key not implemented", key.GetPosition())
		return "", false
	}
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		if value, ok := expr.Value.(string); ok {
			return value, true
		}
	case *ast.BLangVarRef:
		return expr.VariableName.GetValue(), true
	}
	e.resolver.internalError(fmt.Sprintf("unexpected constant mapping key expression %T", key.Expr), key.GetPosition())
	return "", false
}

func (e *constantExpressionEvaluator) evaluateListConstructor(expr *ast.BLangListConstructorExpr) (values.BalValue, bool) {
	initial := make([]values.BalValue, 0, max(len(expr.Exprs), expr.AtomicType.FixedLength()))
	for i, member := range expr.Exprs {
		value, ok := e.evaluate(member)
		if !ok {
			return nil, false
		}
		if expr.IsSpreadMember(i) {
			list, ok := value.(*values.List)
			if !ok {
				return e.semanticFailure("constant list spread member must be a list", member.GetPosition())
			}
			for j := 0; j < list.Len(); j++ {
				initial = append(initial, list.Get(j))
			}
			continue
		}
		initial = append(initial, value)
	}
	for i := len(initial); i < expr.AtomicType.FixedLength(); i++ {
		filler, ok := e.constantFillerValue(expr.AtomicType.MemberAtInnerVal(i), i, expr.GetPosition())
		if !ok {
			return nil, false
		}
		initial = append(initial, filler)
	}
	return e.readonlyListValue(initial, expr.GetPosition())
}

// constantFillerValue builds the constant form of the filler value for the
// omitted member at index i.
func (e *constantExpressionEvaluator) constantFillerValue(memberTy semtypes.SemType, i int,
	loc ast.Location,
) (values.BalValue, bool) {
	cx := e.resolver.typeContext()
	filler, ok := semtypes.FillerValue(cx, memberTy)
	if !ok {
		return e.semanticFailure(common.FormatMissingFillerMessage(cx, i, memberTy), loc)
	}
	return e.readonlyFillerValue(filler, loc)
}

// readonlyFillerValue builds the constant value described by filler:
// recursively readonly, with the singleton shape type of its contents.
func (e *constantExpressionEvaluator) readonlyFillerValue(filler semtypes.Filler,
	loc ast.Location,
) (values.BalValue, bool) {
	switch filler := filler.(type) {
	case semtypes.SingleValueFiller:
		return filler.Value, true
	case semtypes.MappingFiller:
		return e.readonlyMappingValue(nil, loc)
	case semtypes.ListFiller:
		members := make([]values.BalValue, len(filler.Members))
		for i, memberFiller := range filler.Members {
			member, ok := e.readonlyFillerValue(memberFiller, loc)
			if !ok {
				return nil, false
			}
			members[i] = member
		}
		return e.readonlyListValue(members, loc)
	case semtypes.XMLFiller:
		// birgen.materializeFiller does not construct xml fillers either, and
		// BIR has no constant encoding for an xml value.
		return e.unsupportedFailure("constant xml filler value not implemented", loc)
	case semtypes.TableFiller:
		return e.unimplementedFillerFailure(filler.Type, loc)
	case semtypes.ObjectFiller:
		return e.unimplementedFillerFailure(filler.Type, loc)
	case semtypes.StreamFiller:
		return e.unimplementedFillerFailure(filler.Type, loc)
	default:
		return e.internalFailure(fmt.Sprintf("unexpected filler kind %T", filler), loc)
	}
}

func (e *constantExpressionEvaluator) unimplementedFillerFailure(ty semtypes.SemType,
	loc ast.Location,
) (values.BalValue, bool) {
	return e.unsupportedFailure(fmt.Sprintf("constant filler value for type '%s' not implemented",
		semtypes.ToString(e.resolver.typeContext(), ty)), loc)
}

// readonlyListValue builds the constant list holding members, which must
// already be constant values.
func (e *constantExpressionEvaluator) readonlyListValue(members []values.BalValue,
	loc ast.Location,
) (values.BalValue, bool) {
	ty, atomic, err := readonlyListShapeType(e.resolver, members)
	if err != nil {
		return e.internalFailure(err.Error(), loc)
	}
	return values.NewList(ty, atomic, true, nil, len(members), members), true
}

// readonlyMappingValue builds the constant mapping holding entries, whose
// values must already be constant values.
func (e *constantExpressionEvaluator) readonlyMappingValue(entries []values.MapEntry,
	loc ast.Location,
) (values.BalValue, bool) {
	ty, atomic, err := readonlyMappingShapeType(e.resolver, entries)
	if err != nil {
		return e.internalFailure(err.Error(), loc)
	}
	return values.NewMap(ty, atomic, true, entries), true
}

func (e *constantExpressionEvaluator) evaluateUnaryExpression(expr *ast.BLangUnaryExpr) (values.BalValue, bool) {
	value, ok := e.evaluate(expr.Expr)
	if !ok {
		return nil, false
	}
	if value == nil && expr.Operator != model.OperatorKind_NOT {
		return nil, true
	}

	switch expr.Operator {
	case model.OperatorKind_ADD:
		return value, true
	case model.OperatorKind_SUB:
		switch value := value.(type) {
		case int64:
			if value == math.MinInt64 {
				return e.semanticFailure("integer overflow", expr.GetPosition())
			}
			return -value, true
		case float64:
			return -value, true
		case *decimal.Decimal:
			return value.Neg(), true
		}
	case model.OperatorKind_BITWISE_COMPLEMENT:
		if value, ok := value.(int64); ok {
			return ^value, true
		}
	case model.OperatorKind_NOT:
		if value, ok := value.(bool); ok {
			return !value, true
		}
	default:
		// Not a unary operator; falls through to the internal error below.
	}
	return e.internalFailure(fmt.Sprintf("unsupported constant unary operation %s on %T", expr.Operator, value),
		expr.GetPosition())
}

func (e *constantExpressionEvaluator) evaluateTernaryExpression(expr *ast.BLangTernaryExpr) (values.BalValue, bool) {
	condition, ok := e.evaluate(expr.Condition)
	if !ok {
		return nil, false
	}
	conditionValue, isBoolean := condition.(bool)
	if !isBoolean {
		return e.internalFailure(fmt.Sprintf("constant conditional operand has type %T", condition),
			expr.Condition.GetPosition())
	}
	if conditionValue {
		return e.evaluate(expr.ThenExpr)
	}
	return e.evaluate(expr.ElseExpr)
}

func (e *constantExpressionEvaluator) evaluateNilConditionalExpression(expr *ast.BLangNilConditionalExpr) (values.BalValue, bool) {
	lhs, ok := e.evaluate(expr.LhsExpr)
	if !ok {
		return nil, false
	}
	if lhs != nil {
		return lhs, true
	}
	return e.evaluate(expr.RhsExpr)
}

func (e *constantExpressionEvaluator) evaluateBinaryExpression(expr *ast.BLangBinaryExpr) (values.BalValue, bool) {
	lhs, ok := e.evaluate(expr.LhsExpr)
	if !ok {
		return nil, false
	}
	switch expr.OpKind {
	case model.OperatorKind_AND:
		value, isBoolean := lhs.(bool)
		if !isBoolean {
			return e.internalFailure(fmt.Sprintf("constant logical operand has type %T", lhs), expr.LhsExpr.GetPosition())
		}
		if !value {
			return false, true
		}
	case model.OperatorKind_OR:
		value, isBoolean := lhs.(bool)
		if !isBoolean {
			return e.internalFailure(fmt.Sprintf("constant logical operand has type %T", lhs), expr.LhsExpr.GetPosition())
		}
		if value {
			return true, true
		}
	default:
		// Other operators require both operands.
	}

	rhs, ok := e.evaluate(expr.RhsExpr)
	if !ok {
		return nil, false
	}
	if lhs == nil || rhs == nil {
		if isNilLiftedConstantOperator(expr.OpKind) {
			return nil, true
		}
	}

	switch expr.OpKind {
	case model.OperatorKind_ADD, model.OperatorKind_SUB, model.OperatorKind_MUL,
		model.OperatorKind_DIV, model.OperatorKind_MOD:
		value, err := constantArithmetic(expr.OpKind, lhs, rhs)
		if err != nil {
			return e.semanticFailure(err.Error(), expr.GetPosition())
		}
		return value, true
	case model.OperatorKind_AND:
		return lhs.(bool) && rhs.(bool), true
	case model.OperatorKind_OR:
		return lhs.(bool) || rhs.(bool), true
	case model.OperatorKind_EQUAL, model.OperatorKind_EQUALS:
		return values.DeepEquals(lhs, rhs), true
	case model.OperatorKind_NOT_EQUAL:
		return !values.DeepEquals(lhs, rhs), true
	case model.OperatorKind_REF_EQUAL:
		return constantExactEqual(lhs, rhs), true
	case model.OperatorKind_REF_NOT_EQUAL:
		return !constantExactEqual(lhs, rhs), true
	case model.OperatorKind_GREATER_THAN:
		return values.Compare(lhs, rhs) == values.CmpGT, true
	case model.OperatorKind_GREATER_EQUAL:
		result := values.Compare(lhs, rhs)
		return result == values.CmpGT || result == values.CmpEQ, true
	case model.OperatorKind_LESS_THAN:
		return values.Compare(lhs, rhs) == values.CmpLT, true
	case model.OperatorKind_LESS_EQUAL:
		result := values.Compare(lhs, rhs)
		return result == values.CmpLT || result == values.CmpEQ, true
	case model.OperatorKind_BITWISE_AND:
		return lhs.(int64) & rhs.(int64), true
	case model.OperatorKind_BITWISE_OR:
		return lhs.(int64) | rhs.(int64), true
	case model.OperatorKind_BITWISE_XOR:
		return lhs.(int64) ^ rhs.(int64), true
	case model.OperatorKind_BITWISE_LEFT_SHIFT:
		return lhs.(int64) << uint(rhs.(int64)&0x3F), true
	case model.OperatorKind_BITWISE_RIGHT_SHIFT:
		return lhs.(int64) >> uint(rhs.(int64)&0x3F), true
	case model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return int64(uint64(lhs.(int64)) >> uint(rhs.(int64)&0x3F)), true
	default:
		return e.internalFailure(fmt.Sprintf("unsupported constant binary operator %s", expr.OpKind), expr.GetPosition())
	}
}

func constantArithmetic(op model.OperatorKind, lhs, rhs values.BalValue) (values.BalValue, error) {
	switch op {
	case model.OperatorKind_ADD:
		return constantAdd(lhs, rhs)
	case model.OperatorKind_SUB:
		return constantSub(lhs, rhs)
	case model.OperatorKind_MUL:
		return constantMul(lhs, rhs)
	case model.OperatorKind_DIV:
		return constantDiv(lhs, rhs)
	case model.OperatorKind_MOD:
		return constantMod(lhs, rhs)
	default:
		return nil, fmt.Errorf("unsupported constant arithmetic operator %s", op)
	}
}

func isNilLiftedConstantOperator(op model.OperatorKind) bool {
	switch op {
	case model.OperatorKind_ADD, model.OperatorKind_SUB, model.OperatorKind_MUL, model.OperatorKind_DIV, model.OperatorKind_MOD,
		model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR,
		model.OperatorKind_BITWISE_LEFT_SHIFT, model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return true
	default:
		return false
	}
}

func constantAdd(lhs, rhs values.BalValue) (values.BalValue, error) {
	switch lhs := lhs.(type) {
	case int64:
		rhs := rhs.(int64)
		if lhs > 0 && rhs > 0 && lhs > math.MaxInt64-rhs ||
			lhs < 0 && rhs < 0 && lhs < math.MinInt64-rhs {
			return nil, fmt.Errorf("integer overflow")
		}
		return lhs + rhs, nil
	case float64:
		return lhs + rhs.(float64), nil
	case string:
		return lhs + rhs.(string), nil
	case *decimal.Decimal:
		return constantDecimalOperation(lhs.Add, rhs.(*decimal.Decimal))
	default:
		return nil, fmt.Errorf("unsupported constant addition for %T and %T", lhs, rhs)
	}
}

func constantSub(lhs, rhs values.BalValue) (values.BalValue, error) {
	switch lhs := lhs.(type) {
	case int64:
		rhs := rhs.(int64)
		if rhs > 0 && lhs < math.MinInt64+rhs ||
			rhs < 0 && lhs > math.MaxInt64+rhs {
			return nil, fmt.Errorf("integer overflow")
		}
		return lhs - rhs, nil
	case float64:
		return lhs - rhs.(float64), nil
	case *decimal.Decimal:
		return constantDecimalOperation(lhs.Sub, rhs.(*decimal.Decimal))
	default:
		return nil, fmt.Errorf("unsupported constant subtraction for %T and %T", lhs, rhs)
	}
}

func constantMul(lhs, rhs values.BalValue) (values.BalValue, error) {
	lhs, rhs, err := promoteConstantMultiplicativeOperands(lhs, rhs)
	if err != nil {
		return nil, err
	}
	switch lhs := lhs.(type) {
	case int64:
		rhs := rhs.(int64)
		result := lhs * rhs
		if lhs != 0 && rhs != 0 &&
			((lhs == math.MinInt64 && rhs == -1) || (lhs == -1 && rhs == math.MinInt64) || result/rhs != lhs) {
			return nil, fmt.Errorf("integer overflow")
		}
		return result, nil
	case float64:
		return lhs * rhs.(float64), nil
	case *decimal.Decimal:
		return constantDecimalOperation(lhs.Mul, rhs.(*decimal.Decimal))
	default:
		return nil, fmt.Errorf("unsupported constant multiplication for %T and %T", lhs, rhs)
	}
}

func constantDiv(lhs, rhs values.BalValue) (values.BalValue, error) {
	lhs, rhs, err := promoteConstantMultiplicativeOperands(lhs, rhs)
	if err != nil {
		return nil, err
	}
	switch lhs := lhs.(type) {
	case int64:
		rhs := rhs.(int64)
		if rhs == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		if lhs == math.MinInt64 && rhs == -1 {
			return nil, fmt.Errorf("integer overflow")
		}
		return lhs / rhs, nil
	case float64:
		return lhs / rhs.(float64), nil
	case *decimal.Decimal:
		return constantDecimalOperation(lhs.Quo, rhs.(*decimal.Decimal))
	default:
		return nil, fmt.Errorf("unsupported constant division for %T and %T", lhs, rhs)
	}
}

func constantMod(lhs, rhs values.BalValue) (values.BalValue, error) {
	lhs, rhs, err := promoteConstantMultiplicativeOperands(lhs, rhs)
	if err != nil {
		return nil, err
	}
	switch lhs := lhs.(type) {
	case int64:
		rhs := rhs.(int64)
		if rhs == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return lhs % rhs, nil
	case float64:
		return math.Mod(lhs, rhs.(float64)), nil
	case *decimal.Decimal:
		return constantDecimalOperation(lhs.Rem, rhs.(*decimal.Decimal))
	default:
		return nil, fmt.Errorf("unsupported constant remainder for %T and %T", lhs, rhs)
	}
}

func promoteConstantMultiplicativeOperands(lhs, rhs values.BalValue) (values.BalValue, values.BalValue, error) {
	if rhsInt, ok := rhs.(int64); ok {
		switch lhs := lhs.(type) {
		case int64:
			return lhs, rhsInt, nil
		case float64:
			return lhs, float64(rhsInt), nil
		case *decimal.Decimal:
			return lhs, decimal.FromInt64(rhsInt), nil
		default:
			return nil, nil, fmt.Errorf("unsupported constant numeric type %T", lhs)
		}
	}
	if lhsInt, ok := lhs.(int64); ok {
		switch rhs.(type) {
		case float64:
			return float64(lhsInt), rhs, nil
		case *decimal.Decimal:
			return decimal.FromInt64(lhsInt), rhs, nil
		}
	}
	return lhs, rhs, nil
}

func constantDecimalOperation(op func(*decimal.Decimal) (*decimal.Decimal, *decimal.Error), rhs *decimal.Decimal) (values.BalValue, error) {
	result, err := op(rhs)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return result, nil
}

func constantExactEqual(lhs, rhs values.BalValue) bool {
	if lhs == nil || rhs == nil {
		return lhs == nil && rhs == nil
	}
	switch lhs := lhs.(type) {
	case int64:
		rhs, ok := rhs.(int64)
		return ok && lhs == rhs
	case float64:
		rhs, ok := rhs.(float64)
		return ok && values.FloatExactEqual(lhs, rhs)
	case string:
		rhs, ok := rhs.(string)
		return ok && lhs == rhs
	case bool:
		rhs, ok := rhs.(bool)
		return ok && lhs == rhs
	case *decimal.Decimal:
		rhs, ok := rhs.(*decimal.Decimal)
		return ok && lhs.ExactEqual(rhs)
	case *values.List:
		rhs, ok := rhs.(*values.List)
		return ok && lhs == rhs
	case *values.Map:
		rhs, ok := rhs.(*values.Map)
		return ok && lhs == rhs
	default:
		return false
	}
}

func (e *constantExpressionEvaluator) evaluateTypeConversion(expr *ast.BLangTypeConversionExpr) (values.BalValue, bool) {
	value, ok := e.evaluate(expr.Expression)
	if !ok {
		return nil, false
	}
	targetType := expr.TypeDescriptor.GetDeterminedType()
	converted, err := values.CastValue(e.resolver.typeContext(), value, targetType)
	if err != nil {
		return e.semanticFailure(constantCastDiagnostic(value, targetType, err).Error(), expr.GetPosition())
	}
	return converted, true
}

func constantCastDiagnostic(value values.BalValue, targetType semtypes.SemType, err error) error {
	var decimalErr *decimal.Error
	if semtypes.IsSubtypeSimple(targetType, semtypes.Decimal) && errors.As(err, &decimalErr) {
		return decimalErr
	}
	if errors.Is(err, values.ErrBadTypeCast) {
		return fmt.Errorf("converted constant does not belong to target type")
	}
	return constantConversionError(value, targetType)
}

func constantConversionError(value values.BalValue, targetType semtypes.SemType) error {
	target := "target type"
	switch {
	case semtypes.IsSubtypeSimple(targetType, semtypes.Int):
		target = "int"
	case semtypes.IsSubtypeSimple(targetType, semtypes.Float):
		target = "float"
	case semtypes.IsSubtypeSimple(targetType, semtypes.Decimal):
		target = "decimal"
	}
	switch v := value.(type) {
	case bool:
		return fmt.Errorf("bool cannot be converted to %s", target)
	case float64:
		return fmt.Errorf("float value %s cannot be converted to %s", values.FormatFloat(v), target)
	case *decimal.Decimal:
		return fmt.Errorf("decimal value cannot be converted to %s", target)
	default:
		return fmt.Errorf("%T cannot be converted to %s", value, target)
	}
}

func (e *constantExpressionEvaluator) evaluateStringTemplate(expr *ast.BLangTemplateExpr) (values.BalValue, bool) {
	if expr.Kind != ast.TemplateExprKindString || len(expr.Strings) != len(expr.Insertions)+1 {
		return e.internalFailure(fmt.Sprintf("unexpected constant template expression kind %v", expr.Kind),
			expr.GetPosition())
	}
	var result strings.Builder
	for i, insertion := range expr.Insertions {
		result.WriteString(expr.Strings[i])
		value, ok := e.evaluate(insertion)
		if !ok {
			return nil, false
		}
		result.WriteString(values.String(value, nil))
	}
	result.WriteString(expr.Strings[len(expr.Strings)-1])
	return result.String(), true
}
