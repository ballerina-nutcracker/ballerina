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

package semantics

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"ballerina-lang-go/ast"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

var errNotConstantExpression = errors.New("not a constant expression")

type constantExpressionEvaluator struct {
	resolver typeResolver
}

func evaluateConstantExpression(t typeResolver, expr ast.BLangExpression) (values.BalValue, error) {
	evaluator := constantExpressionEvaluator{
		resolver: t,
	}
	return evaluator.evaluate(expr)
}

func (e *constantExpressionEvaluator) evaluate(expr ast.BLangExpression) (values.BalValue, error) {
	switch expr := expr.(type) {
	case *ast.BLangLiteral:
		return expr.Value, nil
	case *ast.BLangNumericLiteral:
		return expr.Value, nil
	case *ast.BLangGroupExpr:
		return e.evaluate(expr.Expression)
	case *ast.BLangSimpleVarRef:
		return e.evaluateConstantReference(expr.Symbol(), expr.GetDeterminedType())
	case *ast.BLangConstRef:
		return e.evaluateConstantReference(expr.Symbol(), expr.GetDeterminedType())
	case *ast.BLangMappingConstructorExpr:
		return e.evaluateMappingConstructor(expr)
	case *ast.BLangListConstructorExpr:
		return e.evaluateListConstructor(expr)
	case *ast.BLangUnaryExpr:
		return e.evaluateUnaryExpression(expr)
	case *ast.BLangBinaryExpr:
		ty := expr.GetDeterminedType()
		if expr.OpKind == model.OperatorKind_ADD && !semtypes.IsZero(ty) && semtypes.IsSubtypeSimple(ty, semtypes.STRING) {
			if value, ok := constantSingleShapeValue(ty); ok {
				return value, nil
			}
		}
		return e.evaluateBinaryExpression(expr)
	case *ast.BLangTypeConversionExpr:
		return e.evaluateTypeConversion(expr)
	case *ast.BLangTemplateExpr:
		if value, ok := constantSingleShapeValue(expr.GetDeterminedType()); ok {
			return value, nil
		}
		return e.evaluateStringTemplate(expr)
	default:
		return nil, fmt.Errorf("%w: %T", errNotConstantExpression, expr)
	}
}

func constantSingleShapeValue(ty semtypes.SemType) (values.BalValue, bool) {
	if semtypes.IsZero(ty) {
		return nil, false
	}
	shape := semtypes.SingleShape(ty)
	if shape.IsEmpty() {
		return nil, false
	}
	value := shape.Get().Value
	switch value.(type) {
	case nil, bool, int64, float64, string, *decimal.Decimal:
		return value, true
	default:
		return nil, false
	}
}

func (e *constantExpressionEvaluator) evaluateConstantReference(ref model.SymbolRef, ty semtypes.SemType) (values.BalValue, error) {
	ref = e.resolver.unnarrowedSymbol(ref)
	if sym, ok := e.resolver.getSymbol(ref).(*model.ConstantValueSymbol); ok {
		return sym.ConstantValue(), nil
	}

	value, ok := constantSingleShapeValue(ty)
	if !ok {
		return nil, errNotConstantExpression
	}
	return value, nil
}

func (e *constantExpressionEvaluator) evaluateMappingConstructor(expr *ast.BLangMappingConstructorExpr) (values.BalValue, error) {
	entries := make([]values.MapEntry, 0, len(expr.Fields))
	for _, field := range expr.Fields {
		kv, ok := field.(*ast.BLangMappingKeyValueField)
		if !ok {
			return nil, errNotConstantExpression
		}
		key, ok := constantMappingKey(kv.Key)
		if !ok {
			return nil, errNotConstantExpression
		}
		value, err := e.evaluate(kv.ValueExpr)
		if err != nil {
			return nil, err
		}
		entries = append(entries, values.MapEntry{Key: key, Value: value})
	}

	ty := expr.GetDeterminedType()
	atomic := semtypes.ToMappingAtomicType(e.resolver.typeContext(), ty)
	if atomic == nil {
		return nil, fmt.Errorf("constant mapping type is not atomic")
	}
	return values.NewMap(ty, atomic, true, entries), nil
}

func constantMappingKey(key *ast.BLangMappingKey) (string, bool) {
	if key == nil || key.Expr == nil {
		return "", false
	}
	switch expr := key.Expr.(type) {
	case *ast.BLangLiteral:
		value, ok := expr.Value.(string)
		return value, ok
	case *ast.BLangSimpleVarRef:
		return expr.VariableName.Value, true
	default:
		return "", false
	}
}

func (e *constantExpressionEvaluator) evaluateListConstructor(expr *ast.BLangListConstructorExpr) (values.BalValue, error) {
	initial := make([]values.BalValue, 0, max(len(expr.Exprs), expr.AtomicType.Members.FixedLength))
	for i, member := range expr.Exprs {
		value, err := e.evaluate(member)
		if err != nil {
			return nil, err
		}
		if expr.IsSpreadMember(i) {
			list, ok := value.(*values.List)
			if !ok {
				return nil, fmt.Errorf("constant list spread member has type %T", value)
			}
			for j := 0; j < list.Len(); j++ {
				initial = append(initial, list.Get(j))
			}
			continue
		}
		initial = append(initial, value)
	}
	for i := len(initial); i < expr.AtomicType.Members.FixedLength; i++ {
		filler, ok := values.FillerFactoryFor(e.resolver.typeContext(), expr.AtomicType.MemberAtInnerVal(i))
		if !ok {
			return nil, fmt.Errorf("constant list member %d has no filler value", i)
		}
		initial = append(initial, filler())
	}
	restFiller, _ := values.FillerFactoryFor(e.resolver.typeContext(), expr.AtomicType.Rest())
	return values.NewList(expr.GetDeterminedType(), &expr.AtomicType, true, restFiller, len(initial), initial), nil
}

func (e *constantExpressionEvaluator) evaluateUnaryExpression(expr *ast.BLangUnaryExpr) (values.BalValue, error) {
	value, err := e.evaluate(expr.Expr)
	if err != nil {
		return nil, err
	}
	if value == nil && expr.Operator != model.OperatorKind_NOT {
		return nil, nil
	}

	switch expr.Operator {
	case model.OperatorKind_ADD:
		return value, nil
	case model.OperatorKind_SUB:
		switch value := value.(type) {
		case int64:
			if value == math.MinInt64 {
				return nil, fmt.Errorf("integer overflow")
			}
			return -value, nil
		case float64:
			return -value, nil
		case *decimal.Decimal:
			return value.Neg(), nil
		}
	case model.OperatorKind_BITWISE_COMPLEMENT:
		if value, ok := value.(int64); ok {
			return ^value, nil
		}
	case model.OperatorKind_NOT:
		if value, ok := value.(bool); ok {
			return !value, nil
		}
	}
	return nil, fmt.Errorf("unsupported constant unary operation %s on %T", expr.Operator, value)
}

func (e *constantExpressionEvaluator) evaluateBinaryExpression(expr *ast.BLangBinaryExpr) (values.BalValue, error) {
	lhs, err := e.evaluate(expr.LhsExpr)
	if err != nil {
		return nil, err
	}
	switch expr.OpKind {
	case model.OperatorKind_AND:
		value, ok := lhs.(bool)
		if !ok {
			return nil, fmt.Errorf("constant logical operand has type %T", lhs)
		}
		if !value {
			return false, nil
		}
	case model.OperatorKind_OR:
		value, ok := lhs.(bool)
		if !ok {
			return nil, fmt.Errorf("constant logical operand has type %T", lhs)
		}
		if value {
			return true, nil
		}
	}

	rhs, err := e.evaluate(expr.RhsExpr)
	if err != nil {
		return nil, err
	}
	if lhs == nil || rhs == nil {
		if isNilLiftedConstantOperator(expr.OpKind) {
			return nil, nil
		}
	}

	switch expr.OpKind {
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
	case model.OperatorKind_AND:
		return lhs.(bool) && rhs.(bool), nil
	case model.OperatorKind_OR:
		return lhs.(bool) || rhs.(bool), nil
	case model.OperatorKind_EQUAL, model.OperatorKind_EQUALS:
		return values.DeepEquals(lhs, rhs), nil
	case model.OperatorKind_NOT_EQUAL:
		return !values.DeepEquals(lhs, rhs), nil
	case model.OperatorKind_REF_EQUAL:
		return constantExactEqual(lhs, rhs), nil
	case model.OperatorKind_REF_NOT_EQUAL:
		return !constantExactEqual(lhs, rhs), nil
	case model.OperatorKind_GREATER_THAN:
		return values.Compare(lhs, rhs) == values.CmpGT, nil
	case model.OperatorKind_GREATER_EQUAL:
		result := values.Compare(lhs, rhs)
		return result == values.CmpGT || result == values.CmpEQ, nil
	case model.OperatorKind_LESS_THAN:
		return values.Compare(lhs, rhs) == values.CmpLT, nil
	case model.OperatorKind_LESS_EQUAL:
		result := values.Compare(lhs, rhs)
		return result == values.CmpLT || result == values.CmpEQ, nil
	case model.OperatorKind_BITWISE_AND:
		return lhs.(int64) & rhs.(int64), nil
	case model.OperatorKind_BITWISE_OR:
		return lhs.(int64) | rhs.(int64), nil
	case model.OperatorKind_BITWISE_XOR:
		return lhs.(int64) ^ rhs.(int64), nil
	case model.OperatorKind_BITWISE_LEFT_SHIFT:
		return lhs.(int64) << uint(rhs.(int64)&0x3F), nil
	case model.OperatorKind_BITWISE_RIGHT_SHIFT:
		return lhs.(int64) >> uint(rhs.(int64)&0x3F), nil
	case model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return int64(uint64(lhs.(int64)) >> uint(rhs.(int64)&0x3F)), nil
	default:
		return nil, fmt.Errorf("%w: binary operator %s", errNotConstantExpression, expr.OpKind)
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

func (e *constantExpressionEvaluator) evaluateTypeConversion(expr *ast.BLangTypeConversionExpr) (values.BalValue, error) {
	value, err := e.evaluate(expr.Expression)
	if err != nil {
		return nil, err
	}
	targetType := expr.TypeDescriptor.GetDeterminedType()
	converted, err := values.CastValue(e.resolver.typeContext(), value, targetType)
	if err != nil {
		return nil, constantCastDiagnostic(value, targetType, err)
	}
	return converted, nil
}

func constantCastDiagnostic(value values.BalValue, targetType semtypes.SemType, err error) error {
	var decimalErr *decimal.Error
	if semtypes.IsSubtypeSimple(targetType, semtypes.DECIMAL) && errors.As(err, &decimalErr) {
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
	case semtypes.IsSubtypeSimple(targetType, semtypes.INT):
		target = "int"
	case semtypes.IsSubtypeSimple(targetType, semtypes.FLOAT):
		target = "float"
	case semtypes.IsSubtypeSimple(targetType, semtypes.DECIMAL):
		target = "decimal"
	}
	switch value.(type) {
	case bool:
		return fmt.Errorf("bool cannot be converted to %s", target)
	case float64:
		return fmt.Errorf("float value cannot be converted to %s", target)
	case *decimal.Decimal:
		return fmt.Errorf("decimal value cannot be converted to %s", target)
	default:
		return fmt.Errorf("%T cannot be converted to %s", value, target)
	}
}

func (e *constantExpressionEvaluator) evaluateStringTemplate(expr *ast.BLangTemplateExpr) (values.BalValue, error) {
	if expr.Kind != ast.TemplateExprKindString || len(expr.Strings) != len(expr.Insertions)+1 {
		return nil, errNotConstantExpression
	}
	var result strings.Builder
	for i, insertion := range expr.Insertions {
		result.WriteString(expr.Strings[i])
		value, err := e.evaluate(insertion)
		if err != nil {
			return nil, err
		}
		result.WriteString(values.String(value, nil))
	}
	result.WriteString(expr.Strings[len(expr.Strings)-1])
	return result.String(), nil
}
