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
	"github.com/ballerina-nutcracker/ballerina/model"
)

func newIdentifier(value string) *ast.BLangIdentifier { return &ast.BLangIdentifier{Value: value} }
func functionNames() {
	fn := &ast.BLangFunction{}
	fn.Name = newIdentifier("$assignedFunction")                                  // want "generated function name must use"
	_ = ast.InvokableData{Name: newIdentifier("$invokable")}                      // want "generated function name must use"
	_ = ast.InvokableData{Name: &ast.BLangIdentifier{Value: "$directIdentifier"}} // want "generated function name must use"
	_ = newIdentifier("$variableIdentifier")
}

type desugaredSymbol struct {
	name string
	kind int
}

var _ = &desugaredSymbol{name: "$method", kind: model.SymbolKindFunction} // want "generated function name must use"
var _ = &desugaredSymbol{name: "$variable", kind: model.SymbolKindVariable}
