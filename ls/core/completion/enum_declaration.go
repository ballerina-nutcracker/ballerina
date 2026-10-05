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
)

var enumDeclarationHandler = handler{
	name:  "enum-declaration",
	match: matchEnumDeclaration,
	build: buildEnumDeclaration,
}

func matchEnumDeclaration(c *cursor) (any, bool) {
	declaration, ok := enumDeclarationAt(c)
	if !ok || enumMemberConstantAt(c, declaration) {
		return nil, false
	}
	return nil, true
}

func buildEnumDeclaration(*cursor, any) []protocol.CompletionItem {
	return emptyItems
}

func enumDeclarationAt(c *cursor) (*ast.BLangTypeDefinition, bool) {
	for i := len(c.chain) - 1; i >= 0; i-- {
		declaration, ok := c.chain[i].(*ast.BLangTypeDefinition)
		if !ok || !declaration.IsEnum() || !enumDeclarationContains(declaration, c.req.Offset) {
			continue
		}
		return declaration, true
	}
	return nil, false
}

func enumDeclarationContains(declaration *ast.BLangTypeDefinition, offset int) bool {
	if declaration == nil || declaration.Name == nil {
		return false
	}
	declarationPosition := declaration.GetPosition()
	namePosition := declaration.Name.GetPosition()
	if !locationHasUsableOffsets(declarationPosition) || !locationHasUsableOffsets(namePosition) {
		return false
	}
	return namePosition.EndOffset() < offset && offset < declarationPosition.EndOffset()
}

func enumMemberConstantAt(c *cursor, declaration *ast.BLangTypeDefinition) bool {
	if declaration == nil {
		return false
	}
	declarationPosition := declaration.GetPosition()
	fileIndex := c.sm.Context().DiagnosticEnv().FileIndex(c.req.URI.Path())
	for _, constant := range c.sm.PackageNode().Constants {
		if !constant.Flags().Has(model.FlagEnumMember) {
			continue
		}
		constantPosition := constant.GetPosition()
		if constantPosition.FileIndex() != fileIndex || !locationContains(declarationPosition, constantPosition.StartOffset()) ||
			!locationContains(declarationPosition, constantPosition.EndOffset()) {
			continue
		}
		if locationContains(constantPosition, c.req.Offset) {
			return true
		}
	}
	return false
}
