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
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

func lexicalItems(c *cursor, extras []protocol.CompletionItem) []protocol.CompletionItem {
	set := newItemSet()
	for _, item := range extras {
		set.add(item)
	}
	walkScope(c, func(ref model.SymbolRef) {
		kind := c.sm.Context().SymbolKind(ref)
		if !lexicalSymbolKind(kind) {
			return
		}
		name := c.sm.Context().SymbolName(ref)
		item := protocol.CompletionItem{
			Label:      name,
			Kind:       protocol.NewOptional(completionItemKind(kind)),
			InsertText: protocol.NewOptional(name),
		}
		if detail := symbolDetail(c, ref); detail != "" {
			item.Detail = protocol.NewOptional(detail)
		}
		set.add(item)
	})
	return set.items()
}

func sameLocation(left, right ast.Location) bool {
	return left.FileIndex() == right.FileIndex() && left.StartOffset() == right.StartOffset() && left.EndOffset() == right.EndOffset()
}

func walkScope(c *cursor, visit func(model.SymbolRef)) {
	seenSpaces := make(map[*model.SymbolSpace]bool)
	visitSpace := func(space *model.SymbolSpace, declarationOrdered bool) {
		if space == nil || seenSpaces[space] {
			return
		}
		seenSpaces[space] = true
		for ref := range space.Symbols() {
			name := c.sm.Context().SymbolName(ref)
			if name == "" || isGeneratedName(name) {
				continue
			}
			loc := c.sm.Context().SymbolLocation(ref)
			if (declarationOrdered && declaredAfterCursor(c, loc)) || declaresVariableAtCursor(c, loc) {
				continue
			}
			visit(ref)
		}
	}
	addScope := func(scope model.Scope) model.Scope {
		switch scope := scope.(type) {
		case *model.BlockScope:
			visitSpace(scope.Main, true)
			return scope.Parent
		case *model.FunctionScope:
			visitSpace(scope.Main, true)
			return scope.Parent
		case *model.ModuleScope:
			visitSpace(scope.Main, false)
			return nil
		case *model.PackageScope:
			for _, space := range scope.MainSpaces {
				visitSpace(space, false)
			}
			if scope.Virtual != nil {
				visitSpace(scope.Virtual.Main, false)
			}
		}
		return nil
	}
	scope := nearestScope(c.chain)
	if scope == nil {
		scope = c.sm.PackageNode().Scope
	}
	for scope != nil {
		scope = addScope(scope)
	}
	if pkgScope := c.sm.PackageNode().Scope; pkgScope != nil {
		addScope(pkgScope)
	}
}

func declaredAfterCursor(c *cursor, loc ast.Location) bool {
	return locationHasUsableOffsets(loc) && loc.StartOffset() > c.req.Offset
}

// A variable is not in scope in its own declaration, including its initializer. A foreach variable is not in
// scope in the foreach collection.
func declaresVariableAtCursor(c *cursor, loc ast.Location) bool {
	for _, node := range c.chain {
		var variable *ast.BLangVariable
		switch node := node.(type) {
		case *ast.BLangVariable:
			variable = node
		case *ast.BLangForeach:
			if node.VariableDef != nil && node.Collection != nil && locationContains(node.Collection.GetPosition(), c.req.Offset) {
				variable = node.VariableDef.Var
			}
		}
		if variable != nil && variable.Name != nil && sameLocation(variable.Name.GetPosition(), loc) {
			return true
		}
	}
	return false
}

func lexicalSymbolKind(kind model.SymbolKind) bool {
	return kind == model.SymbolKindVariable || kind == model.SymbolKindParemeter ||
		kind == model.SymbolKindConstant || kind == model.SymbolKindFunction || kind == model.SymbolKindType
}

func symbolDetail(c *cursor, ref model.SymbolRef) string {
	ty := c.sm.Context().SymbolType(ref)
	if semtypes.IsZero(ty) {
		return ""
	}
	return semtypes.ToString(semtypes.ContextFrom(c.sm.Context().GetTypeEnv()), ty)
}
