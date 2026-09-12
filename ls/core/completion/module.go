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

// moduleHandler offers module-level keywords plus lexically-visible module
// symbols when the cursor sits directly among a module's top-level nodes
// (not inside any narrower construct) -- migrated from classify()'s
// kindModule case verbatim.
var moduleHandler = handler{
	name:  "module",
	match: matchModule,
	build: buildModule,
}

func matchModule(c *cursor) (any, bool) {
	node, ok := c.winner()
	if !ok {
		return nil, false
	}
	_, ok = node.(*ast.BLangPackage)
	return nil, ok
}

func buildModule(c *cursor, _ any) []protocol.CompletionItem {
	return lexicalItems(c, moduleKeywords)
}
