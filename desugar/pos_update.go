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
	"ballerina/ast"
	"ballerina/tools/diagnostics"
)

type posUpdateVisitor struct {
	pos diagnostics.Location
}

func (v *posUpdateVisitor) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		return nil
	}
	if diagnostics.IsLocationEmpty(node.GetPosition()) {
		node.SetPosition(v.pos)
	}
	return v
}

func (v *posUpdateVisitor) VisitTypeData(typeData *ast.TypeData) ast.Visitor {
	return v
}

func setPositionIfMissing(root ast.BLangNode, pos diagnostics.Location) {
	ast.Walk(&posUpdateVisitor{pos: pos}, root)
}
