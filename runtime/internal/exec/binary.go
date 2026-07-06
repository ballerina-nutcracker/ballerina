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

package exec

import (
	"fmt"
	"math"

	"ballerina-lang-go/bir"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

func execBinaryOpAdd(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	switch v1 := op1.(type) {
	case int64:
		v2 := op2.(int64)
		if v1 > 0 && v2 > 0 && v1 > math.MaxInt64-v2 {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		if v1 < 0 && v2 < 0 && v1 < math.MinInt64-v2 {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1+v2)
	case float64:
		v2 := op2.(float64)
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1+v2)
	case string:
		v2 := op2.(string)
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1+v2)
	case *decimal.Decimal:
		v2 := op2.(*decimal.Decimal)
		setOperandValue(ctx, binaryOp.LhsOp, frame, decimalArith(v1.Add, v2))
	case values.XMLValue:
		v2 := op2.(values.XMLValue)
		setOperandValue(ctx, binaryOp.LhsOp, frame, values.NewXMLConcatSequence(v1, v2))
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported type combination: %T + %T", op1, op2)))
	}
}

func execBinaryOpSub(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	switch v1 := op1.(type) {
	case int64:
		v2 := op2.(int64)
		if v2 > 0 && v1 < math.MinInt64+v2 {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		if v2 < 0 && v1 > math.MaxInt64+v2 {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1-v2)
	case float64:
		v2 := op2.(float64)
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1-v2)
	case *decimal.Decimal:
		v2 := op2.(*decimal.Decimal)
		setOperandValue(ctx, binaryOp.LhsOp, frame, decimalArith(v1.Sub, v2))
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported type combination: %T - %T", op1, op2)))
	}
}

func execBinaryOpMul(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	op1, op2 = promoteMultiplicativeOperands(op1, op2, true)
	switch v1 := op1.(type) {
	case int64:
		v2 := op2.(int64)
		result := v1 * v2
		if v1 != 0 && v2 != 0 && ((v1 == math.MinInt64 && v2 == -1) || (v1 == -1 && v2 == math.MinInt64) || result/v2 != v1) {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		setOperandValue(ctx, binaryOp.LhsOp, frame, result)
	case float64:
		v2 := op2.(float64)
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1*v2)
	case *decimal.Decimal:
		v2 := op2.(*decimal.Decimal)
		setOperandValue(ctx, binaryOp.LhsOp, frame, decimalArith(v1.Mul, v2))
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported type combination: %T * %T", op1, op2)))
	}
}

func execBinaryOpDiv(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	op1, op2 = promoteMultiplicativeOperands(op1, op2, false)
	switch v1 := op1.(type) {
	case int64:
		v2 := op2.(int64)
		if v2 == 0 {
			panic(values.NewErrorWithMessage("divide by zero"))
		}
		if v1 == math.MinInt64 && v2 == -1 {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1/v2)
	case float64:
		v2 := op2.(float64)
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1/v2)
	case *decimal.Decimal:
		v2 := op2.(*decimal.Decimal)
		setOperandValue(ctx, binaryOp.LhsOp, frame, decimalArith(v1.Quo, v2))
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported type combination: %T / %T", op1, op2)))
	}
}

func execBinaryOpMod(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	op1, op2 = promoteMultiplicativeOperands(op1, op2, false)
	switch v1 := op1.(type) {
	case int64:
		v2 := op2.(int64)
		if v2 == 0 {
			panic(values.NewErrorWithMessage("divide by zero"))
		}
		setOperandValue(ctx, binaryOp.LhsOp, frame, v1%v2)
	case float64:
		v2 := op2.(float64)
		setOperandValue(ctx, binaryOp.LhsOp, frame, math.Mod(v1, v2))
	case *decimal.Decimal:
		v2 := op2.(*decimal.Decimal)
		setOperandValue(ctx, binaryOp.LhsOp, frame, decimalArith(v1.Rem, v2))
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported type combination: %T %% %T", op1, op2)))
	}
}

func execBinaryOpEqual(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	setOperandValue(ctx, binaryOp.LhsOp, frame, values.DeepEquals(op1, op2))
}

func execBinaryOpNotEqual(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	setOperandValue(ctx, binaryOp.LhsOp, frame, !values.DeepEquals(op1, op2))
}

// promoteMultiplicativeOperands implements the implicit numeric conversion permitted
// for multiplicative expressions with mixed numeric operands: when the second operand
// is int, it is converted to the type of the first operand; when the operation is
// multiplication and the first operand is int, it is converted to the type of the
// second operand. The conversion always succeeds for int -> float / decimal.
func promoteMultiplicativeOperands(op1, op2 values.BalValue, isMul bool) (values.BalValue, values.BalValue) {
	switch v2 := op2.(type) {
	case int64:
		switch v1 := op1.(type) {
		case int64:
			return v1, v2
		case float64:
			return v1, float64(v2)
		case *decimal.Decimal:
			return v1, decimal.FromInt64(v2)
		default:
			panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported numeric type: %T", op1)))
		}
	}
	if !isMul {
		return op1, op2
	}
	if v1, ok := op1.(int64); ok {
		switch op2.(type) {
		case float64:
			return float64(v1), op2
		case *decimal.Decimal:
			return decimal.FromInt64(v1), op2
		default:
			panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported numeric type: %T", op2)))
		}
	}
	return op1, op2
}

// decimalArith invokes a decimal arithmetic method (Add/Sub/Mul/Quo/Rem) and
// converts a typed decimal error into a Ballerina runtime error panic.
func decimalArith(op func(*decimal.Decimal) (*decimal.Decimal, *decimal.Error), b *decimal.Decimal) *decimal.Decimal {
	out, err := op(b)
	if err != nil {
		panic(values.NewErrorWithMessage(err.Error()))
	}
	return out
}

func execBinaryOpGT(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	r := values.Compare(op1, op2)
	setOperandValue(ctx, binaryOp.LhsOp, frame, r == values.CmpGT)
}

func execBinaryOpGTE(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	r := values.Compare(op1, op2)
	setOperandValue(ctx, binaryOp.LhsOp, frame, r == values.CmpGT || r == values.CmpEQ)
}

func execBinaryOpLT(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	r := values.Compare(op1, op2)
	setOperandValue(ctx, binaryOp.LhsOp, frame, r == values.CmpLT)
}

func execBinaryOpLTE(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	r := values.Compare(op1, op2)
	setOperandValue(ctx, binaryOp.LhsOp, frame, r == values.CmpLT || r == values.CmpEQ)
}

func execBinaryOpAnd(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	setOperandValue(ctx, binaryOp.LhsOp, frame, op1.(bool) && op2.(bool))
}

func execBinaryOpOr(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	setOperandValue(ctx, binaryOp.LhsOp, frame, op1.(bool) || op2.(bool))
}

func execBinaryOpRefEqual(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	setOperandValue(ctx, binaryOp.LhsOp, frame, refEqual(op1, op2))
}

func execBinaryOpRefNotEqual(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	setOperandValue(ctx, binaryOp.LhsOp, frame, !refEqual(op1, op2))
}

func refEqual(op1, op2 values.BalValue) bool {
	if op1 == nil && op2 == nil {
		return true
	}
	if op1 == nil || op2 == nil {
		return false
	}
	if xml1, ok := op1.(values.XMLValue); ok {
		xml2, ok := op2.(values.XMLValue)
		return ok && xmlRefEqual(xml1, xml2)
	}
	if f1, ok := op1.(*values.Function); ok {
		if f2, ok := op2.(*values.Function); ok {
			return f1.LookupKey == f2.LookupKey
		}
		return false
	}
	if d1, ok := op1.(*decimal.Decimal); ok {
		d2, ok := op2.(*decimal.Decimal)
		if !ok {
			return false
		}
		return d1.ExactEqual(d2)
	}
	if f1, ok := op1.(float64); ok {
		f2, ok := op2.(float64)
		return ok && values.FloatExactEqual(f1, f2)
	}
	return op1 == op2
}

func xmlRefEqual(op1, op2 values.XMLValue) bool {
	switch v1 := op1.(type) {
	case *values.XMLElement:
		v2, ok := op2.(*values.XMLElement)
		return ok && v1 == v2
	case *values.XMLProcessingInstruction:
		v2, ok := op2.(*values.XMLProcessingInstruction)
		return ok && v1 == v2
	case *values.XMLComment:
		v2, ok := op2.(*values.XMLComment)
		return ok && v1 == v2
	case *values.XMLSequence:
		v2, ok := op2.(*values.XMLSequence)
		return ok && xmlSequenceRefEqual(v1, v2)
	case *values.XMLText:
		_, ok := op2.(*values.XMLText)
		return ok && values.DeepEquals(op1, op2)
	default:
		panic(fmt.Sprintf("internal error: unsupported XML value type %T", op1))
	}
}

func xmlSequenceRefEqual(op1, op2 *values.XMLSequence) bool {
	if len(op1.Children) != len(op2.Children) {
		return false
	}
	if len(op1.Children) == 1 {
		return values.DeepEquals(op1.Children[0], op2.Children[0])
	}
	for i := range op1.Children {
		if !refEqual(op1.Children[i], op2.Children[i]) {
			return false
		}
	}
	return true
}

func execBinaryOpBitwiseAnd(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	execBinaryOpBitwise(ctx, binaryOp, frame, func(a, b int64) int64 { return a & b })
}

func execBinaryOpBitwiseOr(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	execBinaryOpBitwise(ctx, binaryOp, frame, func(a, b int64) int64 { return a | b })
}

func execBinaryOpBitwiseXor(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	execBinaryOpBitwise(ctx, binaryOp, frame, func(a, b int64) int64 { return a ^ b })
}

func execBinaryOpBitwiseLeftShift(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	execBinaryOpBitwise(ctx, binaryOp, frame, func(a, b int64) int64 { return a << uint(b&shiftAmountMask) })
}

func execBinaryOpBitwiseRightShift(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	execBinaryOpBitwise(ctx, binaryOp, frame, func(a, b int64) int64 { return a >> uint(b&shiftAmountMask) })
}

func execBinaryOpBitwiseUnsignedRightShift(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) {
	execBinaryOpBitwise(ctx, binaryOp, frame, func(a, b int64) int64 { return int64(uint64(a) >> uint(b&shiftAmountMask)) })
}

const shiftAmountMask = 0x3F

func execUnaryOpNot(ctx *extern.Context, unaryOp *bir.UnaryOp, frame *Frame) {
	op := getOperandValue(ctx, unaryOp.RhsOp, frame)
	setOperandValue(ctx, unaryOp.LhsOp, frame, !op.(bool))
}

func execUnaryOpNegate(ctx *extern.Context, unaryOp *bir.UnaryOp, frame *Frame) {
	op := getOperandValue(ctx, unaryOp.RhsOp, frame)
	switch v := op.(type) {
	case int64:
		if v == math.MinInt64 {
			panic(values.NewErrorWithMessage("arithmetic overflow"))
		}
		setOperandValue(ctx, unaryOp.LhsOp, frame, -v)
	case float64:
		setOperandValue(ctx, unaryOp.LhsOp, frame, -v)
	case *decimal.Decimal:
		setOperandValue(ctx, unaryOp.LhsOp, frame, v.Neg())
	default:
		panic(values.NewErrorWithMessage(fmt.Sprintf("unsupported type: %T (expected int64, float64, or *decimal.Decimal)", op)))
	}
}

func execUnaryOpBitwiseComplement(ctx *extern.Context, unaryOp *bir.UnaryOp, frame *Frame) {
	op := getOperandValue(ctx, unaryOp.RhsOp, frame)
	v := op.(int64)
	setOperandValue(ctx, unaryOp.LhsOp, frame, ^v)
}

func execBinaryOpBitwise(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame, bitOp func(a, b int64) int64) {
	op1, op2 := getBinaryRhsValues(ctx, binaryOp, frame)
	v1 := op1.(int64)
	v2 := op2.(int64)
	setOperandValue(ctx, binaryOp.LhsOp, frame, bitOp(v1, v2))
}

func getBinaryRhsValues(ctx *extern.Context, binaryOp *bir.BinaryOp, frame *Frame) (op1, op2 values.BalValue) {
	return getOperandValue(ctx, &binaryOp.RhsOp1, frame), getOperandValue(ctx, &binaryOp.RhsOp2, frame)
}
