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

package desugar

import (
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// materializeConstantRef replaces a reference to a folded constant with a
// literal carrying the folded value.
func materializeConstantRef(cx *functionContext, ref *ast.BLangVarRef) ast.BLangExpression {
	constSym, ok := cx.getSymbol(ref.Symbol()).(*model.ConstantValueSymbol)
	if !ok {
		return nil
	}
	value := constSym.ConstantValue()
	ty := ref.GetDeterminedType()
	if semtypes.IsZero(ty) {
		cx.pkgCtx.compilerCtx.InternalError("constant reference type is not resolved", ref.GetPosition())
		return nil
	}
	return constantValueLiteral(value, ref.GetPosition(), ty)
}

func constantValueLiteral(value values.BalValue, pos diagnostics.Location, ty semtypes.SemType) ast.BLangExpression {
	info := constantValueLiteralInfo(value)
	originalValue := values.String(value, make(map[uintptr]bool))
	var expr ast.BLangExpression
	var lit *ast.BLangLiteral
	if info.numeric {
		numeric := ast.NewBLangNumericLiteral(pos, info.kind, value, originalValue, true)
		expr = numeric
		lit = &numeric.BLangLiteral
	} else {
		lit = ast.NewBLangLiteral(pos, info.kind, value, originalValue, true)
		expr = lit
	}
	lit.SetDeterminedType(ty)
	return expr
}

type constantLiteralInfo struct {
	numeric bool
	kind    ast.LiteralKind
}

func constantValueLiteralInfo(value values.BalValue) constantLiteralInfo {
	switch value.(type) {
	case nil:
		return constantLiteralInfo{kind: ast.LiteralKindNil}
	case bool:
		return constantLiteralInfo{kind: ast.LiteralKindBoolean}
	case int, int64, int32, int16, int8:
		return constantLiteralInfo{numeric: true, kind: ast.LiteralKindInt}
	case byte:
		return constantLiteralInfo{numeric: true, kind: ast.LiteralKindByte}
	case float64, float32:
		return constantLiteralInfo{numeric: true, kind: ast.LiteralKindFloat}
	case *decimal.Decimal:
		return constantLiteralInfo{numeric: true, kind: ast.LiteralKindDecimal}
	case string, *string:
		return constantLiteralInfo{kind: ast.LiteralKindString}
	default:
		return constantLiteralInfo{}
	}
}
