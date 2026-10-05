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
	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
)

// dumpCursorAST logs the pretty-printed AST rooted at the cursor's nearest
// chain node (the tightest enclosing node with a real position -- see
// nodeChainAtOffset), with the node dispatch() actually matched on
// (c.winnerNode) bracketed by >>> <<<. Only WithASTDebugLogging call sites
// reach here, so this never runs on the hot path by default.
func dumpCursorAST(logger *observability.Logger, c *cursor) {
	if len(c.chain) == 0 {
		return
	}
	printer := &ast.PrettyPrinter{
		ShowNodeLocations: true,
		DiagnosticEnv:     c.sm.Context().DiagnosticEnv(),
	}
	logger.Info("completion cursor AST",
		"offset", c.req.Offset,
		"winner_matched", c.winnerMatched,
		"ast", printer.Print(c.chain[0]))
}
