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
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

var fieldAccessHandler = handler{
	name:  "field-access",
	match: matchFieldAccess,
	build: noCompletionItems,
}

var invocationHandler = handler{
	name:  "invocation",
	match: matchInvocation,
	build: noCompletionItems,
}

var importHandler = handler{
	name:  "import",
	match: matchImport,
	build: noCompletionItems,
}

func matchFieldAccess(c *cursor) (any, bool) {
	node, ok := c.winner()
	if !ok {
		return nil, false
	}
	_, ok = node.(*ast.BLangFieldBaseAccess)
	return nil, ok
}

func matchInvocation(c *cursor) (any, bool) {
	node, ok := c.winner()
	if !ok {
		return nil, false
	}
	_, ok = node.(*ast.BLangInvocation)
	return nil, ok
}

func matchImport(c *cursor) (any, bool) {
	node, ok := c.winner()
	if !ok {
		return nil, false
	}
	switch node.(type) {
	case *ast.BLangImportPackage, *ast.BLangVarRef:
		return nil, true
	default:
		return nil, false
	}
}

func noCompletionItems(*cursor, any) []protocol.CompletionItem {
	return emptyItems
}
