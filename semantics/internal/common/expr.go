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

import "github.com/ballerina-nutcracker/ballerina/model"

type OperatorExpression interface {
	GetOperatorKind() model.OperatorKind
}

func IsEqualityExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_EQUAL, model.OperatorKind_EQUALS, model.OperatorKind_NOT_EQUAL, model.OperatorKind_REF_EQUAL, model.OperatorKind_REF_NOT_EQUAL:
		return true
	default:
		return false
	}
}

func IsMultiplicativeExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_MUL, model.OperatorKind_DIV, model.OperatorKind_MOD:
		return true
	default:
		return false
	}
}

func IsRangeExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_CLOSED_RANGE, model.OperatorKind_HALF_OPEN_RANGE:
		return true
	default:
		return false
	}
}

func IsBitWiseExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_BITWISE_AND, model.OperatorKind_BITWISE_OR, model.OperatorKind_BITWISE_XOR:
		return true
	default:
		return false
	}
}

func IsShiftExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_BITWISE_LEFT_SHIFT, model.OperatorKind_BITWISE_RIGHT_SHIFT, model.OperatorKind_BITWISE_UNSIGNED_RIGHT_SHIFT:
		return true
	default:
		return false
	}
}

func IsRelationalExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_LESS_THAN, model.OperatorKind_LESS_EQUAL, model.OperatorKind_GREATER_THAN, model.OperatorKind_GREATER_EQUAL:
		return true
	default:
		return false
	}
}

func IsAdditiveExpr(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_ADD, model.OperatorKind_SUB:
		return true
	default:
		return false
	}
}

func IsLogicalExpression(expr OperatorExpression) bool {
	switch expr.GetOperatorKind() {
	case model.OperatorKind_AND, model.OperatorKind_OR:
		return true
	default:
		return false
	}
}
