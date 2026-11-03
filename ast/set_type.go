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

package ast

import (
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// SetType sets the type of an expression or action.
//
// The exhaustive type switch is intentional: gochecksumtype checks it against every BLangActionOrExpression
// implementer, so adding an expression or action kind without deciding how its type is set fails lint.
func SetType(node BLangActionOrExpression, ty semtypes.SemType) {
	switch n := node.(type) {
	case *bLangExpressionBase: // also covers AbstractExpression (type alias)
		n.Ty = ty
	case *bLangActionBase:
		n.Ty = ty
	case *BLangValueExpressionBase:
		n.Ty = ty
	case *BLangVariableReferenceBase:
		n.Ty = ty
	case *bLangAccessExpressionBase:
		n.Ty = ty
	case *BLangWorkerSendReceiveExprBase:
		n.Ty = ty
	case *BLangWorkerSendExprBase:
		n.Ty = ty
	case *BLangLiteral:
		n.Ty = ty
	case *BLangNumericLiteral:
		n.Ty = ty
	case *BLangTemplateExpr:
		n.Ty = ty
	case *BLangVarRef:
		n.Ty = ty
	case *BLangConstRef:
		n.Ty = ty
	case *BLangFieldBaseAccess:
		n.Ty = ty
	case *BLangIndexBasedAccess:
		n.Ty = ty
	case *BLangAnnotAccessExpr:
		n.Ty = ty
	case *BLangListConstructorExpr:
		n.Ty = ty
	case *BLangMappingConstructorExpr:
		n.Ty = ty
	case *BLangErrorConstructorExpr:
		n.Ty = ty
	case *BLangNewExpression:
		n.Ty = ty
	case *BLangBinaryExpr:
		n.Ty = ty
	case *BLangUnaryExpr:
		n.Ty = ty
	case *BLangTernaryExpr:
		n.Ty = ty
	case *BLangNilConditionalExpr:
		n.Ty = ty
	case *BLangGroupExpr:
		n.Ty = ty
	case *BLangTypeConversionExpr:
		n.Ty = ty
	case *BLangTypeTestExpr:
		n.Ty = ty
	case *BLangTypedescExpr:
		n.Ty = ty
	case *BLangInferredTypedescDefault:
		n.Ty = ty
	case *BLangCheckedExpr:
		n.Ty = ty
	case *BLangCheckPanickedExpr:
		n.Ty = ty
	case *BLangTrapExpr:
		n.Ty = ty
	case *BLangInvocation:
		n.Ty = ty
	case *BLangNamedArgsExpression:
		n.Ty = ty
	case *BLangDefaultArg:
		n.Ty = ty
	case *BLangRemoteMethodCallAction:
		n.Ty = ty
	case *BLangClientResourceAccessAction:
		n.Ty = ty
	case *BLangStartAction:
		n.Ty = ty
	case *BLangSingleWaitAction:
		n.Ty = ty
	case *BLangMultipleWaitAction:
		n.Ty = ty
	case *BLangAlternateWaitAction:
		n.Ty = ty
	case *BLangLambdaFunction:
		n.Ty = ty
	case *BLangArrowFunction:
		n.Ty = ty
	case *BLangQueryExpr:
		n.Ty = ty
	case *BLangXMLElementLiteral:
		n.Ty = ty
	case *BLangXMLTextLiteral:
		n.Ty = ty
	case *BLangXMLCommentLiteral:
		n.Ty = ty
	case *BLangXMLPILiteral:
		n.Ty = ty
	case *BLangXMLSequenceLiteral:
		n.Ty = ty
	case *BLangXMLTemplateExpr:
		n.Ty = ty
	case *BLangXMLStepExpression:
		n.Ty = ty
	case *BLangXMLFilterExpression:
		n.Ty = ty
	case *BLangWorkerReceive:
		n.Ty = ty
	case *BLangAlternateWorkerReceive:
		n.Ty = ty
	case *BLangCommitExpr:
		n.Ty = ty
	case *BLangWildCardBindingPattern:
		n.Ty = ty
	case *BLangBadExprOrAction:
		if !semtypes.IsNever(ty) {
			panic("SetType: a bad expression can only have type never")
		}
	default:
		panic(fmt.Sprintf("SetType: unexpected node %T", node))
	}
}
