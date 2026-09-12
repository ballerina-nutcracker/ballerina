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
	"unicode"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/model"
)

type cursor struct {
	req   Request
	sm    compile.SealedModule
	chain []ast.BLangNode

	winnerNode    ast.BLangNode
	winnerMatched bool
}

func newCursor(req Request, sm compile.SealedModule) *cursor {
	fileIndex := sm.Context().DiagnosticEnv().FileIndex(req.URI.Path())
	c := &cursor{req: req, sm: sm, chain: nodeChainAtOffset(sm.PackageNode(), req.Offset, fileIndex)}
	c.winnerNode, c.winnerMatched = nearestChainMatch(c)
	return c
}

// winner returns the nearest (tail-to-head) chain node claimed by one of the
// dispatch seam's migrated node-shape checks: field-access/invocation/
// import/module-qualified-reference (the excluded-context family), block, or
// module. Handlers for those families check this instead of independently
// rescanning the whole chain themselves, which preserves classify()'s
// original "nearest node wins" semantics: e.g. a block passed as a lambda
// argument to a call must still win over the enclosing BLangInvocation, even
// though both are present in the chain.
func (c *cursor) winner() (ast.BLangNode, bool) {
	return c.winnerNode, c.winnerMatched
}

// nearestChainMatch is classify()'s original single-pass walk, unchanged,
// now returning the winning node instead of setting a cursorKind field.
func nearestChainMatch(c *cursor) (ast.BLangNode, bool) {
	for i := len(c.chain) - 1; i >= 0; i-- {
		switch node := c.chain[i].(type) {
		case *ast.BLangFieldBaseAccess, *ast.BLangInvocation, *ast.BLangImportPackage:
			if locationContains(node.GetPosition(), c.req.Offset) {
				return c.chain[i], true
			}
		case *ast.BLangVarRef:
			if node.PkgAlias != nil && node.PkgAlias.GetValue() != "" && locationContains(node.GetPosition(), c.req.Offset) {
				return c.chain[i], true
			}
		case *ast.BLangBlockStmt, *ast.BLangBlockFunctionBody:
			if i+1 == len(c.chain) || isBadStatement(c.chain[i+1]) {
				return c.chain[i], true
			}
		case *ast.BLangPackage:
			if i+1 == len(c.chain) || isBadTopLevel(c.chain[i+1]) {
				return c.chain[i], true
			}
		}
	}
	return nil, false
}

func isBadStatement(node ast.BLangNode) bool {
	_, ok := node.(*ast.BLangBadStmt)
	return ok
}

func isBadTopLevel(node ast.BLangNode) bool {
	_, ok := node.(*ast.BLangBadTopLevelNode)
	return ok
}

func nodeChainAtOffset(pkg *ast.BLangPackage, offset, fileIndex int) []ast.BLangNode {
	finder := &chainFinder{offset: offset, fileIndex: fileIndex}
	ast.Walk(finder, pkg)
	return finder.chain
}

type chainFinder struct {
	offset    int
	fileIndex int
	stack     []ast.BLangNode
	chain     []ast.BLangNode
}

func (f *chainFinder) Visit(node ast.BLangNode) ast.Visitor {
	if node == nil {
		if len(f.stack) > 0 {
			f.stack = f.stack[:len(f.stack)-1]
		}
		return f
	}
	loc := node.GetPosition()
	if locationHasUsableOffsets(loc) && (!locationContains(loc, f.offset) || loc.FileIndex() != f.fileIndex) {
		return nil
	}
	f.stack = append(f.stack, node)
	if len(f.chain) == 0 || locationHasUsableOffsets(loc) {
		f.chain = append(f.chain[:0], f.stack...)
	}
	return f
}

func (f *chainFinder) VisitTypeData(*ast.TypeData) ast.Visitor { return f }

func locationHasUsableOffsets(loc ast.Location) bool {
	return loc.StartOffset() >= 0 && loc.EndOffset() >= 0 && (loc.StartOffset() != 0 || loc.EndOffset() != 0)
}

func locationContains(loc ast.Location, offset int) bool {
	return loc.StartOffset() >= 0 && loc.EndOffset() >= 0 && loc.StartOffset() <= offset && offset <= loc.EndOffset()
}

func nearestScope(chain []ast.BLangNode) model.Scope {
	for i := len(chain) - 1; i >= 0; i-- {
		if scoped, ok := chain[i].(ast.NodeWithScope); ok && scoped.Scope() != nil {
			return scoped.Scope()
		}
	}
	return nil
}

func identifierPrefixAtOffset(text string, offset int) string {
	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:start])
		if (r == utf8.RuneError && size == 0) || !(r == '_' || r == '\'' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
			break
		}
		start -= size
	}
	return text[start:offset]
}
