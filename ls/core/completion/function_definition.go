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
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

var functionDefinitionHandler = handler{
	syntaxOnly: true,
	name:       "function-definition",
	match:      matchFunctionDefinition,
	build:      buildFunctionDefinition,
}

func matchFunctionDefinition(c *cursor) (any, bool) {
	node, ok := moduleLevelRecoveredDeclarationAt(c)
	if !ok {
		return nil, false
	}
	loc := node.GetPosition()
	if !locationHasUsableOffsets(loc) || loc.StartOffset() > c.req.Offset || c.req.Offset > len(c.req.Text) {
		return nil, false
	}
	tokens := significantTokens(typeDefinitionTokens(c.req.Text[loc.StartOffset():c.req.Offset]))
	return nil, functionKeywordMissingAfterQualifiers(tokens, c.req.Offset-loc.StartOffset())
}

func moduleLevelRecoveredDeclarationAt(c *cursor) (ast.BLangNode, bool) {
	for i := len(c.chain) - 1; i > 0; i-- {
		node := c.chain[i]
		switch node := node.(type) {
		case *ast.BLangBadTopLevelNode:
		case *ast.BLangVariable:
			if _, badType := node.TypeNode().(*ast.BLangBadTypeNode); !badType {
				continue
			}
		default:
			continue
		}
		_, inModule := c.chain[i-1].(*ast.BLangPackage)
		return node, inModule && locationContains(node.GetPosition(), c.req.Offset)
	}
	return nil, false
}

func significantTokens(tokens []typeDefinitionToken) []typeDefinitionToken {
	significant := make([]typeDefinitionToken, 0, len(tokens))
	for _, token := range tokens {
		if strings.TrimSpace(token.text) != "" {
			significant = append(significant, token)
		}
	}
	return significant
}

func functionKeywordMissingAfterQualifiers(tokens []typeDefinitionToken, cursorEnd int) bool {
	index := 0
	if index < len(tokens) && tokens[index].text == "public" {
		index++
	}
	isolated, transactional := false, false
	for ; index < len(tokens); index++ {
		switch tokens[index].text {
		case "isolated":
			isolated = true
			continue
		case "transactional":
			transactional = true
			continue
		}
		break
	}
	if !isolated || !transactional || index != len(tokens)-1 {
		return false
	}
	name := tokens[index]
	return name.word && name.text != "function" && name.end == cursorEnd
}

func buildFunctionDefinition(*cursor, any) []protocol.CompletionItem {
	snippetSort := neutralRelevance.add(1).sortText()
	keywordSort := neutralRelevance.sortText()
	return []protocol.CompletionItem{
		snippetItem(kwFunction, keywordSort),
		snippetItem(defFunction, snippetSort),
		snippetItem(defExpressionBodiedFunction, snippetSort),
	}
}
