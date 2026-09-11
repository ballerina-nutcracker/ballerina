// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package completion

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func expectedTypeAt(c *cursor) (semtypes.SemType, bool) {
	for i := len(c.chain) - 1; i >= 0; i-- {
		variable, ok := c.chain[i].(*ast.BLangVariable)
		if !ok || variable.IsDeclaredWithVar || variable.TypeNode() == nil || variable.Expr == nil {
			continue
		}
		if !isInitializerOffset(c.req.Text, c.req.Offset, variable) {
			continue
		}
		ty := variable.GetAssociatedType()
		if semtypes.IsZero(ty) && !variable.Symbol().IsEmpty() {
			ty = c.sm.Context().SymbolType(variable.Symbol())
		}
		if semtypes.IsZero(ty) {
			ty = variable.GetDeterminedType()
		}
		if semtypes.IsZero(ty) {
			return semtypes.SemType{}, false
		}
		return ty, true
	}
	return semtypes.SemType{}, false
}

func isInitializerOffset(text string, offset int, variable *ast.BLangVariable) bool {
	expressionPosition := variable.Expr.GetPosition()
	variablePosition := variable.GetPosition()
	if expressionPosition.StartOffset() < 0 || expressionPosition.EndOffset() < expressionPosition.StartOffset() ||
		variablePosition.StartOffset() < 0 || expressionPosition.StartOffset() > len(text) ||
		variablePosition.StartOffset() > expressionPosition.StartOffset() {
		return false
	}
	variableStart := variablePosition.StartOffset()
	equalsOffset := strings.IndexByte(text[variableStart:expressionPosition.StartOffset()], '=')
	if equalsOffset < 0 {
		return false
	}
	initializerStart := variableStart + equalsOffset + 1
	return initializerStart <= offset && offset <= expressionPosition.EndOffset()
}
