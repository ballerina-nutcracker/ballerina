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

// The field-access/invocation/import families are today's silent
// classify() fallthrough: no completion is offered inside a field access, a
// call, or an import declaration. They're registered here as explicit no-op
// handlers -- match true, build empty -- so a later per-provider ticket has
// an existing entry to replace instead of an implicit "nothing matched"
// fallthrough with no attachment point.
//
// All three ask cursor.winner() (cursor.go), the memoized migration of
// classify()'s original single-pass walk, rather than independently
// rescanning the chain -- see winner()'s doc comment for why that matters.

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

// importHandler also covers a module-qualified BLangVarRef (e.g. the `mod`
// in `mod:name`). classify() grouped that with the import-prefix case since
// both need module-scoped symbol enumeration to build real candidates --
// moduleContentSymbols (module_symbols.go) is that shared lookup, ready for
// whichever per-provider ticket replaces this no-op. The ADR doesn't name
// this fourth node shape explicitly; folding it in here is this migration's
// call, not a design decision.
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
