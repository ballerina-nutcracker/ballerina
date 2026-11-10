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
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/ast"
)

// PrettyPrintFallback prints the desugar-only AST nodes when serializing a
// desugared package via ast.PrettyPrinter. Wire it in by setting
// PrettyPrinter.Fallback to this function.
func PrettyPrintFallback(p *ast.PrettyPrinter, node ast.BLangNode) {
	switch n := node.(type) {
	case *BLangServiceInit:
		p.StartNode()
		p.PrintString("service-init")
		p.EndNode()
	case *BLangExpressionThunk:
		p.StartNode()
		p.PrintString("expression-thunk")
		for _, stmt := range n.InitStmts {
			p.PrintInner(stmt.(ast.BLangNode))
		}
		p.PrintInner(n.Expr)
		p.EndNode()
	default:
		panic(fmt.Sprintf("desugar pretty printer: unsupported node %T", n)) //nolint:forbidigo // Every desugar-owned node must have a printer case.
	}
}
