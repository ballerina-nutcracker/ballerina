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

func lexicalItems(c *cursor, keywords []string) []protocol.CompletionItem {
	set := newItemSet()
	expectedType, hasExpectedType := expectedTypeAt(c)
	compatible := make(map[string]bool)
	for _, keyword := range keywords {
		set.add(keywordItem(keyword))
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
		if hasExpectedType && semtypes.IsSubtype(semtypes.ContextFrom(c.sm.Context().GetTypeEnv()), symbolType(c, ref), expectedType) {
			compatible[name] = true
		}
	})
	if !hasExpectedType {
		return set.items()
	}
	return set.itemsRanked(func(item protocol.CompletionItem) bool {
		return compatible[item.Label]
	})
}

func symbolType(c *cursor, ref model.SymbolRef) semtypes.SemType {
	ty := c.sm.Context().SymbolType(ref)
	if !semtypes.IsZero(ty) {
		return ty
	}
	location := c.sm.Context().SymbolLocation(ref)
	for _, node := range c.chain {
		variable, ok := node.(*ast.BLangVariable)
		if !ok || variable.Name == nil || !sameLocation(variable.Name.GetPosition(), location) {
			continue
		}
		ty := variable.GetAssociatedType()
		if !semtypes.IsZero(ty) {
			return ty
		}
		return variable.GetDeterminedType()
	}
	return semtypes.SemType{}
}

func sameLocation(left, right ast.Location) bool {
	return left.FileIndex() == right.FileIndex() && left.StartOffset() == right.StartOffset() && left.EndOffset() == right.EndOffset()
}

func walkScope(c *cursor, visit func(model.SymbolRef)) {
	seenSpaces := make(map[*model.SymbolSpace]bool)
	visitSpace := func(space *model.SymbolSpace) {
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
			if locationHasUsableOffsets(loc) && loc.StartOffset() > c.req.Offset {
				continue
			}
			visit(ref)
		}
	}
	addScope := func(scope model.Scope) model.Scope {
		switch scope := scope.(type) {
		case *model.BlockScope:
			visitSpace(scope.Main)
			return scope.Parent
		case *model.FunctionScope:
			visitSpace(scope.Main)
			return scope.Parent
		case *model.ModuleScope:
			visitSpace(scope.Main)
			return nil
		case *model.PackageScope:
			for _, space := range scope.MainSpaces {
				visitSpace(space)
			}
			if scope.Virtual != nil {
				visitSpace(scope.Virtual.Main)
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
