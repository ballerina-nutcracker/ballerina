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
	"ballerina-lang-go/ast"
	"ballerina-lang-go/decimal"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/values"
)

// materializeConstantRef replaces a reference to a folded constant with a
// literal carrying the folded value.
func materializeConstantRef(cx *functionContext, ref *ast.BLangSimpleVarRef) ast.BLangExpression {
	constSym, ok := cx.getSymbol(ref.Symbol()).(*model.ConstantValueSymbol)
	if !ok {
		return nil
	}
	value, ok := constSym.ConstantValue()
	if !ok {
		cx.pkgCtx.compilerCtx.InternalError("constant value is not folded", ref.GetPosition())
		return nil
	}
	ty := ref.GetDeterminedType()
	if semtypes.IsZero(ty) {
		cx.pkgCtx.compilerCtx.InternalError("constant reference type is not resolved", ref.GetPosition())
		return nil
	}
	return constantValueLiteral(value, ref.GetPosition(), ty)
}

func constantValueLiteral(value values.BalValue, pos diagnostics.Location, ty semtypes.SemType) ast.BLangExpression {
	var expr ast.BLangExpression
	var lit *ast.BLangLiteral
	info := constantValueLiteralInfo(value)
	if info.numeric {
		numeric := &ast.BLangNumericLiteral{}
		expr = numeric
		lit = &numeric.BLangLiteral
	} else {
		plain := &ast.BLangLiteral{}
		expr = plain
		lit = plain
	}

	lit.SetValue(value)
	lit.SetOriginalValue(values.String(value, make(map[uintptr]bool)))
	lit.SetIsConstant(true)
	lit.SetDeterminedType(ty)
	lit.SetPosition(pos)
	if info.hasTag {
		bt := &ast.BTypeBasic{}
		bt.BTypeSetTag(info.tag)
		lit.SetValueType(bt)
	}
	return expr
}

type constantLiteralInfo struct {
	numeric bool
	tag     ast.TypeTags
	hasTag  bool
}

func constantValueLiteralInfo(value values.BalValue) constantLiteralInfo {
	switch value.(type) {
	case nil:
		return constantLiteralInfo{tag: ast.TypeTags_NIL, hasTag: true}
	case bool:
		return constantLiteralInfo{tag: ast.TypeTags_BOOLEAN, hasTag: true}
	case int, int64, int32, int16, int8:
		return constantLiteralInfo{numeric: true, tag: ast.TypeTags_INT, hasTag: true}
	case byte:
		return constantLiteralInfo{numeric: true, tag: ast.TypeTags_BYTE, hasTag: true}
	case float64, float32:
		return constantLiteralInfo{numeric: true, tag: ast.TypeTags_FLOAT, hasTag: true}
	case *decimal.Decimal:
		return constantLiteralInfo{numeric: true, tag: ast.TypeTags_DECIMAL, hasTag: true}
	case string, *string:
		return constantLiteralInfo{tag: ast.TypeTags_STRING, hasTag: true}
	default:
		return constantLiteralInfo{}
	}
}
