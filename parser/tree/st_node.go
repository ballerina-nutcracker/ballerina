// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package tree

import (
	"fmt"
	"reflect"
	"strings"

	"ballerina/parser/common"
	"ballerina/tools/diagnostics"
)

// This represent green nodes in the syntax tree. Green nodes satisfy fallowing properties:
//   1. Immutable (except via `Replace`)
//   2. No parent reference
//   3. Know width but not position
//   4. We build them bottom up
// All green nodes satisfy STNode interface.
// All the actual nodes are generated in `st_node_gen.go` using `tree-gen` tool.
// We need these to be very memory efficient but not necessarily fast to build. We build these once but hold them
// for very long time.
//   We need to be careful about not holding lot of "medium" sized nodes for each token.
// As package name suggests, these nodes represent an internal implementation details of the parser and should not be
// exposed

type STNode interface {
	Kind() common.SyntaxKind
	Diagnostics() []STNodeDiagnostic
	Width() uint16
	WidthWithLeadingMinutiae() uint16
	WidthWithTrailingMinutiae() uint16
	WidthWithMinutiae() uint16
	Flags() uint8
	BucketCount() int
	HasDiagnostics() bool
	IsMissing() bool
	ChildBuckets() []STNode
	ChildInBucket(bucket int) STNode
	setDiagnostics(diagnostics []STNodeDiagnostic)
	addChildren(children ...STNode)
	updateDiagnostics(children []STNode)
	updateWidth(children []STNode)
	CreateFacade(position int, parent NonTerminalNode) Node
}

type STToken interface {
	STNode
	Text() string
	HasTrailingNewLine() bool
	LeadingMinutiae() STNode
	TrailingMinutiae() STNode
	ModifyWith(leadingMinutiae STNode, trailingMinutiae STNode) STToken
}

// Actual "base" types for AST nodes. We generate most of the actual nodes in st_node_gen.go.
//
//go:generate ../../tree-gen -config ../nodes.json -type st-node -template ../../compiler-tools/tree-gen/templates/st-node.go.tmpl -output st_node_gen.go -util-template ../../compiler-tools/tree-gen/templates/st-node-util.go.tmpl -util-output st_node_util_gen.go
type (
	STTokenBase struct {
		diagnostics      []STNodeDiagnostic
		leadingMinutiae  STNode
		trailingMinutiae STNode
		kind             common.SyntaxKind
		width            uint16
		flags            uint8
	}

	STMissingToken struct {
		STTokenBase
	}

	STLiteralValueToken struct {
		STTokenBase
		// Ideally we don't need width here and instead calculate it on the go
		text string
	}

	// TODO: can this be based on STTokenBase as well?
	STInvalidTokenMinutiaeNode struct {
		STNodeBase
		token STToken
	}

	STInvalidToken struct {
		STTokenBase
		tokenText string
	}

	STIdentifierToken struct {
		STTokenBase
		text string
	}

	STMinutiae struct {
		STNodeBase
		text string
	}

	STInvalidNodeMinutiae struct {
		STMinutiae
		invalidNode STNode
	}

	STNodeList struct {
		STNodeBase
		children []STNode
	}

	STNodeBase struct {
		kind                      common.SyntaxKind
		diagnostics               []STNodeDiagnostic
		width                     uint16
		widthWithLeadingMinutiae  uint16
		widthWithTrailingMinutiae uint16
		widthWithMinutiae         uint16
		flags                     uint8
	}

	STNodeDiagnostic struct {
		code diagnostics.DiagnosticCode
		args []any
	}
)

func (n *STLiteralValueToken) BucketCount() int {
	return 1
}

func (n *STLiteralValueToken) ChildInBucket(bucket int) STNode {
	switch bucket {
	case 0:
		// token is STToken, which implements STNode
		return n
	default:
		panic("invalid bucket index")
	}
}

func (n *STLiteralValueToken) CreateFacade(position int, parent NonTerminalNode) Node {
	return &LiteralValueToken{
		TokenBase: TokenBase{
			NodeBase: NodeBase{
				internalNode: n,
				position:     position,
				parent:       parent,
			},
		},
	}
}

func (n *STMinutiae) Width() uint16 {
	return uint16(len(n.text))
}

func (n *STMinutiae) WidthWithLeadingMinutiae() uint16 {
	return n.Width()
}

func (n *STMinutiae) WidthWithTrailingMinutiae() uint16 {
	return n.Width()
}

func (n *STMinutiae) WidthWithMinutiae() uint16 {
	return n.Width()
}

func (n *STMinutiae) CreateFacade(position int, parent NonTerminalNode) Node {
	panic("unsupported operation")
}

func (n *STInvalidNodeMinutiae) CreateFacade(position int, parent NonTerminalNode) Node {
	panic("unsupported operation")
}

func (n *STInvalidTokenMinutiaeNode) BucketCount() int {
	return 1
}

func (n *STInvalidTokenMinutiaeNode) ChildInBucket(bucket int) STNode {
	switch bucket {
	case 0:
		return n.token
	default:
		panic("invalid bucket index")
	}
}

func (n *STInvalidTokenMinutiaeNode) ChildBuckets() []STNode {
	return []STNode{n.token}
}

func (n *STInvalidTokenMinutiaeNode) CreateFacade(position int, parent NonTerminalNode) Node {
	panic("unsupported operation")
}

func (t *STNodeBase) BucketCount() int {
	panic("BucketCount is not supported for STNodeBase")
}

func (t *STNodeBase) ChildBuckets() []STNode {
	panic("ChildBuckets is not supported for STNodeBase")
}

func (t *STNodeBase) ChildInBucket(bucket int) STNode {
	panic("ChildInBucket is not supported for STNodeBase")
}

func (t *STNodeBase) CreateFacade(position int, parent NonTerminalNode) Node {
	panic("CreateFacade is not supported for STNodeBase")
}

// ModifyWith implementations for all token types
// Port from Java STToken.java:129-131 and similar methods in each token class

func (t *STTokenBase) ModifyWith(leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	// FIXME: add lenghts
	return createNodeAndAddChildren(&STTokenBase{
		kind:             t.kind,
		width:            t.width,
		diagnostics:      t.diagnostics,
		flags:            t.flags,
		leadingMinutiae:  leadingMinutiae,
		trailingMinutiae: trailingMinutiae,
	}, leadingMinutiae, trailingMinutiae).(*STTokenBase)
}

func (t *STMissingToken) ModifyWith(leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	return createNodeAndAddChildren(&STMissingToken{
		STTokenBase: STTokenBase{
			kind:             t.kind,
			width:            t.width,
			diagnostics:      t.diagnostics,
			flags:            t.flags,
			leadingMinutiae:  leadingMinutiae,
			trailingMinutiae: trailingMinutiae,
		},
	}, leadingMinutiae, trailingMinutiae).(*STMissingToken)
}

func (t *STMissingToken) CreateFacade(position int, parent NonTerminalNode) Node {
	if t.kind == common.IDENTIFIER_TOKEN {
		return &IdentifierToken{
			TokenBase: TokenBase{
				NodeBase: NodeBase{
					internalNode: t,
					position:     position,
					parent:       parent,
				},
			},
		}
	} else {
		return &TokenBase{
			NodeBase: NodeBase{
				internalNode: t,
				position:     position,
				parent:       parent,
			},
		}
	}
}

func (t *STLiteralValueToken) ModifyWith(leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	return createNodeAndAddChildren(&STLiteralValueToken{
		STTokenBase: STTokenBase{
			kind:             t.kind,
			width:            t.width,
			diagnostics:      t.diagnostics,
			flags:            t.flags,
			leadingMinutiae:  leadingMinutiae,
			trailingMinutiae: trailingMinutiae,
		},
		text: t.text,
	}, leadingMinutiae, trailingMinutiae).(*STLiteralValueToken)
}

func (t *STInvalidToken) ModifyWith(leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	return createNodeAndAddChildren(&STInvalidToken{
		STTokenBase: STTokenBase{
			kind:             t.kind,
			width:            t.width,
			diagnostics:      t.diagnostics,
			flags:            t.flags,
			leadingMinutiae:  leadingMinutiae,
			trailingMinutiae: trailingMinutiae,
		},
		tokenText: t.tokenText,
	}, leadingMinutiae, trailingMinutiae).(*STInvalidToken)
}

func (t *STInvalidToken) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TokenBase{
		NodeBase: NodeBase{
			internalNode: t,
			position:     position,
			parent:       parent,
		},
	}
}

func (t *STIdentifierToken) ModifyWith(leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	// STIdentifierToken embeds STToken interface, need to extract the base
	base := t.STTokenBase
	return createNodeAndAddChildren(&STIdentifierToken{
		STTokenBase: STTokenBase{
			kind:             base.kind,
			width:            base.width,
			diagnostics:      base.diagnostics,
			flags:            base.flags,
			leadingMinutiae:  leadingMinutiae,
			trailingMinutiae: trailingMinutiae,
		},
		text: t.text,
	}, leadingMinutiae, trailingMinutiae).(*STIdentifierToken)
}

func (s STNodeList) IsEmpty() bool {
	return s.BucketCount() == 0
}

func (s *STNodeList) ChildBuckets() []STNode {
	return s.children
}

func (s *STNodeList) ChildInBucket(bucket int) STNode {
	return s.children[bucket]
}

func (s *STNodeList) CreateFacade(position int, parent NonTerminalNode) Node {
	return &ExternalTreeNodeList{
		NonTerminalNodeBase: NonTerminalNodeBase{
			NodeBase: NodeBase{
				internalNode: s,
				position:     position,
				parent:       parent,
			},
			childBuckets: make([]Node, s.BucketCount()),
		},
	}
}

// Shared common nodes
var (
	emptyNodeList = &STNodeList{
		STNodeBase: STNodeBase{
			kind: common.LIST,
		},
	}
)

// Methods for creating STNodes
func CreateInvalidTokenMinutiaeNode(token STToken) *STInvalidTokenMinutiaeNode {
	return createNodeAndAddChildren(&STInvalidTokenMinutiaeNode{
		STNodeBase: STNodeBase{
			kind: common.INVALID_TOKEN_MINUTIAE_NODE,
		},
		token: token,
	}, token).(*STInvalidTokenMinutiaeNode)
}

func CreateLiteralValueToken(kind common.SyntaxKind,
	text string,
	leadingTrivia STNode,
	trailingTrivia STNode,
) STToken {
	return CreateLiteralValueTokenWithDiagnostics(kind, text, leadingTrivia, trailingTrivia, nil)
}

func CreateLiteralValueTokenWithDiagnostics(kind common.SyntaxKind,
	text string,
	leadingTrivia STNode,
	trailingTrivia STNode,
	diagnostics []STNodeDiagnostic,
) STToken {
	return createNodeAndAddChildren(&STLiteralValueToken{
		STTokenBase: STTokenBase{
			kind:             kind,
			width:            uint16(len(text)),
			leadingMinutiae:  leadingTrivia,
			trailingMinutiae: trailingTrivia,
			diagnostics:      diagnostics,
		},
		text: text,
	}, leadingTrivia, trailingTrivia).(*STLiteralValueToken)
}

func CreateDiagnostic(code diagnostics.DiagnosticCode, args ...any) STNodeDiagnostic {
	return STNodeDiagnostic{
		code: code,
		args: args,
	}
}

func CreateTokenFrom(kind common.SyntaxKind, leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	return CreateTokenWithDiagnostics(kind, leadingMinutiae, trailingMinutiae, nil)
}

func CreateTokenWithDiagnostics(kind common.SyntaxKind, leadingMinutiae STNode, trailingMinutiae STNode, diagnostics []STNodeDiagnostic) STToken {
	width := uint16(len(kind.StrValue()))
	return createNodeAndAddChildren(&STTokenBase{
		kind:             kind,
		width:            width,
		leadingMinutiae:  leadingMinutiae,
		trailingMinutiae: trailingMinutiae,
		diagnostics:      diagnostics,
	}, leadingMinutiae, trailingMinutiae).(*STTokenBase)
}

// FIXME: remove this
func CreateNodeListFromNodes(nodes ...STNode) STNode {
	return CreateNodeList(nodes...)
}

func CreateNodeList(nodes ...STNode) STNode {
	// Return shared empty instance for empty lists
	if len(nodes) == 0 {
		return emptyNodeList
	}
	return createNodeAndAddChildren(&STNodeList{
		STNodeBase: STNodeBase{
			kind: common.LIST,
		},
		children: nodes,
	}, nodes...)
}

func CreateEmptyNodeList() STNode {
	return emptyNodeList
}

func CreateMinutiae(kind common.SyntaxKind, text string) *STMinutiae {
	return &STMinutiae{
		STNodeBase: STNodeBase{
			kind: kind,
		},
		text: text,
	}
}

func CreateInvalidToken(tokenText string) *STInvalidToken {
	return &STInvalidToken{
		STTokenBase: STTokenBase{
			kind:             common.INVALID_TOKEN,
			width:            uint16(len(tokenText)),
			leadingMinutiae:  CreateEmptyNodeList(),
			trailingMinutiae: CreateEmptyNodeList(),
		},
		tokenText: tokenText,
	}
}

func CreateInvalidNodeMinutiae(invalidToken STToken) STNode {
	invalidNode := CreateInvalidTokenMinutiaeNode(invalidToken)
	return createNodeAndAddChildren(&STInvalidNodeMinutiae{
		STMinutiae: STMinutiae{
			STNodeBase: STNodeBase{
				kind: common.INVALID_NODE_MINUTIAE,
			},
			text: ToSourceCode(invalidNode),
		},
		invalidNode: invalidNode,
	}, invalidNode).(*STInvalidNodeMinutiae)
}

func CreateIdentifierToken(text string, leadingTrivia STNode, trailingTrivia STNode) STToken {
	return createNodeAndAddChildren(&STIdentifierToken{
		STTokenBase: STTokenBase{
			kind:             common.IDENTIFIER_TOKEN,
			width:            uint16(len(text)),
			leadingMinutiae:  leadingTrivia,
			trailingMinutiae: trailingTrivia,
		},
		text: text,
	}, leadingTrivia, trailingTrivia).(*STIdentifierToken)
}

func CreateIdentifierTokenWithDiagnostics(text string, leadingTrivia STNode, trailingTrivia STNode, diagnostics []STNodeDiagnostic) STToken {
	token := CreateIdentifierToken(text, leadingTrivia, trailingTrivia)
	token.setDiagnostics(diagnostics)
	return token
}

// findToken searches for a token in the child buckets.
// Direction determines iteration order: forward searches from first to last child,
// backward searches from last to first child.
func findToken(n STNode, dir direction) STToken {
	start, end, step := 0, n.BucketCount(), 1
	if dir == backward {
		start, end, step = n.BucketCount()-1, -1, -1
	}

	for i := start; i != end; i += step {
		child := n.ChildInBucket(i)
		if isToken(child) {
			token, ok := child.(STToken)
			if ok {
				return token
			}
			panic("expected STToken")
		}
		if (!IsSTNodePresent(child) || IsSTNodeList(child)) && child.BucketCount() == 0 {
			continue
		}
		var token STToken
		if dir == forward {
			token = FirstToken(child)
		} else {
			token = LastToken(child)
		}
		if IsSTNodePresent(token) {
			return token
		}
	}
	if dir == forward {
		panic("failed to find first token")
	}
	panic("failed to find last token")
}

func FirstToken(n STNode) STToken {
	token, ok := n.(STToken)
	if ok {
		return token
	}
	return findToken(n, forward)
}

func LastToken(n STNode) STToken {
	token, ok := n.(STToken)
	if ok {
		return token
	}
	return findToken(n, backward)
}

// childSTNodesForDiagnosticSearch returns child ST nodes to traverse when looking for
// syntax diagnostics. It includes token minutiae and invalid-node minutiae, which are
// not reachable via ChildBuckets() alone.
func childSTNodesForDiagnosticSearch(st STNode) []STNode {
	if st == nil || !IsSTNodePresent(st) {
		return nil
	}
	switch n := st.(type) {
	case *STNodeList:
		out := make([]STNode, 0, n.Size())
		for i := 0; i < n.Size(); i++ {
			out = append(out, n.Get(i))
		}
		return out
	case *STInvalidNodeMinutiae:
		return []STNode{n.invalidNode}
	case STToken:
		var out []STNode
		if lm := n.LeadingMinutiae(); IsSTNodePresent(lm) {
			out = append(out, lm)
		}
		if tm := n.TrailingMinutiae(); IsSTNodePresent(tm) {
			out = append(out, tm)
		}
		return out
	case *STMinutiae:
		return nil
	default:
		return n.ChildBuckets()
	}
}

func FindDeepestDiagnosticSTNode(st STNode) STNode {
	if st == nil || !IsSTNodePresent(st) {
		return nil
	}
	for _, ch := range childSTNodesForDiagnosticSearch(st) {
		if d := FindDeepestDiagnosticSTNode(ch); d != nil {
			return d
		}
	}
	if len(st.Diagnostics()) > 0 {
		return st
	}
	return nil
}

func ToSourceCode(n STNode) string {
	builder := strings.Builder{}
	writeTo(n, &builder)
	return builder.String()
}

func writeTo(n STNode, builder *strings.Builder) {
	tok, ok := n.(STToken)
	if ok {
		writeTo(tok.LeadingMinutiae(), builder)
		builder.WriteString(tok.Text())
		writeTo(tok.TrailingMinutiae(), builder)
		return
	}
	for _, child := range n.ChildBuckets() {
		if IsSTNodePresent(child) {
			writeTo(child, builder)
		}
	}
}

func (n *STNodeBase) setDiagnostics(diagnostics []STNodeDiagnostic) {
	n.diagnostics = diagnostics
	if len(diagnostics) > 0 {
		n.flags = n.flags | HAS_DIAGNOSTIC
	} else {
		n.flags = n.flags &^ HAS_DIAGNOSTIC
	}
}

func (n *STNodeBase) copy() *STNodeBase {
	diagnosticsCopy := make([]STNodeDiagnostic, len(n.diagnostics))
	copy(diagnosticsCopy, n.diagnostics)

	return &STNodeBase{
		kind:                      n.kind,
		diagnostics:               diagnosticsCopy,
		width:                     n.width,
		widthWithLeadingMinutiae:  n.widthWithLeadingMinutiae,
		widthWithTrailingMinutiae: n.widthWithTrailingMinutiae,
		widthWithMinutiae:         n.widthWithMinutiae,
		flags:                     n.flags,
	}
}

func (n STNodeBase) Kind() common.SyntaxKind {
	return n.kind
}

func (n STNodeBase) Diagnostics() []STNodeDiagnostic {
	return n.diagnostics
}

func (n STNodeBase) Width() uint16 {
	return n.width
}

func (n STNodeBase) WidthWithLeadingMinutiae() uint16 {
	return n.widthWithLeadingMinutiae
}

func (n STNodeBase) WidthWithTrailingMinutiae() uint16 {
	return n.widthWithTrailingMinutiae
}

func (n STNodeBase) WidthWithMinutiae() uint16 {
	return n.widthWithMinutiae
}

func (n STNodeBase) Flags() uint8 {
	return n.flags
}

func (n STNodeBase) HasDiagnostics() bool {
	return isFlagSet(n.flags, HAS_DIAGNOSTIC)
}

func (n STNodeBase) IsMissing() bool {
	return isFlagSet(n.flags, IS_MISSING)
}

func Tokens(n STNode) []STToken {
	token, ok := n.(STToken)
	if ok {
		return []STToken{token}
	}
	tokens := make([]STToken, 0, n.BucketCount())
	for _, child := range n.ChildBuckets() {
		if IsSTNodePresent(child) {
			tokens = append(tokens, Tokens(child)...)
		}
	}
	return tokens
}

func (n *STNodeBase) addChildren(children ...STNode) {
	n.updateDiagnostics(children)
	n.updateWidth(children)
}

func (n *STNodeBase) updateDiagnostics(children []STNode) {
	for _, child := range children {
		if !IsSTNodePresent(child) {
			continue
		}
		if child.HasDiagnostics() {
			n.flags = n.flags | HAS_DIAGNOSTIC
			return
		}
	}
}

func (n *STNodeBase) updateWidth(children []STNode) {
	firstChildIndex := n.getFirstChildIndex(children)
	if firstChildIndex == -1 {
		return
	}
	lastChildIndex := n.getLastChildIndex(children)
	if lastChildIndex == -1 {
		return
	}
	firstChild := children[firstChildIndex]
	lastChild := children[lastChildIndex]
	if firstChildIndex == lastChildIndex {
		n.width = firstChild.Width()
		n.widthWithLeadingMinutiae = firstChild.WidthWithLeadingMinutiae()
		n.widthWithTrailingMinutiae = firstChild.WidthWithTrailingMinutiae()
		n.widthWithMinutiae = firstChild.WidthWithMinutiae()
		return
	}
	n.width = firstChild.WidthWithTrailingMinutiae() + lastChild.WidthWithLeadingMinutiae()
	n.widthWithLeadingMinutiae = firstChild.WidthWithMinutiae() + lastChild.WidthWithLeadingMinutiae()
	n.widthWithTrailingMinutiae = firstChild.WidthWithTrailingMinutiae() + lastChild.WidthWithMinutiae()
	n.widthWithMinutiae = firstChild.WidthWithMinutiae() + lastChild.WidthWithMinutiae()
	n.updateWidthWithIndices(children, firstChildIndex, lastChildIndex)
}

func (n *STNodeBase) updateWidthWithIndices(children []STNode, firstChildIndex int, lastChildIndex int) {
	for i := firstChildIndex + 1; i < lastChildIndex; i++ {
		child := children[i]
		if !IsSTNodePresent(child) {
			continue
		}
		n.width += child.WidthWithMinutiae()
		n.widthWithLeadingMinutiae += child.WidthWithMinutiae()
		n.widthWithTrailingMinutiae += child.WidthWithMinutiae()
		n.widthWithMinutiae += child.WidthWithMinutiae()
	}
}

func (n STNodeBase) getFirstChildIndex(children []STNode) int {
	for i, child := range children {
		if IsSTNodePresent(child) && child.WidthWithMinutiae() != 0 {
			return i
		}
	}
	return -1
}

func (n STNodeBase) getLastChildIndex(children []STNode) int {
	for i := len(children) - 1; i >= 0; i-- {
		if IsSTNodePresent(children[i]) && children[i].WidthWithMinutiae() != 0 {
			return i
		}
	}
	return -1
}

func (n STTokenBase) Kind() common.SyntaxKind {
	return n.kind
}

func (n STTokenBase) LeadingMinutiae() STNode {
	return n.leadingMinutiae
}

func (n STTokenBase) TrailingMinutiae() STNode {
	return n.trailingMinutiae
}

func (n *STTokenBase) addChildren(children ...STNode) {
	// if (len(children) >= 1) {
	// 	leadingMinutiae := children[0]
	// 	n.widthWithLeadingMinutiae =  n.width + leadingMinutiae.Width()
	// 	n.widthWithMinutiae =  n.width + leadingMinutiae.Width()
	// }
	// if (len(children) >= 2) {
	// 	trailingMinutiae := children[len(children)-1]
	// 	n.widthWithTrailingMinutiae =  n.width + trailingMinutiae.Width()
	// 	n.widthWithMinutiae =  n.width + trailingMinutiae.Width()
	// }
	n.updateDiagnostics(children)
}

func (n *STTokenBase) updateDiagnostics(children []STNode) {
	for _, child := range children {
		if !IsSTNodePresent(child) {
			continue
		}
		if child.HasDiagnostics() {
			n.flags = n.flags | HAS_DIAGNOSTIC
			return
		}
	}
}

func (n STTokenBase) updateWidth(children []STNode) {
	panic("updateWidth is not supported for STToken")
}

func (n *STTokenBase) Diagnostics() []STNodeDiagnostic {
	return n.diagnostics
}

func (n *STTokenBase) Width() uint16 {
	return n.width
}

func (n *STTokenBase) WidthWithLeadingMinutiae() uint16 {
	return n.width + n.leadingMinutiae.Width()
}

func (n *STTokenBase) WidthWithTrailingMinutiae() uint16 {
	return n.width + n.trailingMinutiae.Width()
}

func (n *STTokenBase) WidthWithMinutiae() uint16 {
	return n.width + n.leadingMinutiae.Width() + n.trailingMinutiae.Width()
}

func (n *STTokenBase) Flags() uint8 {
	return n.flags
}

func (n *STTokenBase) BucketCount() int {
	return 0
}

func (n *STTokenBase) ChildBuckets() []STNode {
	return nil
}

func (n *STTokenBase) HasDiagnostics() bool {
	return isFlagSet(n.flags, HAS_DIAGNOSTIC)
}

func (n *STTokenBase) ChildInBucket(bucket int) STNode {
	panic("ChildInBucket is not supported for STToken")
}

func (n *STTokenBase) IsMissing() bool {
	return isFlagSet(n.flags, IS_MISSING)
}

func (n *STTokenBase) Tokens() []STToken {
	return []STToken{n}
}

func (t *STTokenBase) Text() string {
	return t.kind.StrValue()
}

func (t *STTokenBase) CreateFacade(position int, parent NonTerminalNode) Node {
	return &TokenBase{
		NodeBase: NodeBase{
			internalNode: t,
			position:     position,
			parent:       parent,
		},
	}
}

func (t *STIdentifierToken) Text() string {
	return t.text
}

func (t *STIdentifierToken) CreateFacade(position int, parent NonTerminalNode) Node {
	return &IdentifierToken{
		TokenBase: TokenBase{
			NodeBase: NodeBase{
				internalNode: t,
				position:     position,
				parent:       parent,
			},
		},
	}
}

func (t *STLiteralValueToken) Text() string {
	return t.text
}

func (t *STInvalidToken) Text() string {
	return t.tokenText
}

func (t *STTokenBase) HasTrailingNewLine() bool {
	stNodeList := t.trailingMinutiae.(*STNodeList)
	for i := 0; i < stNodeList.Size(); i++ {
		if stNodeList.Get(i).Kind() == common.END_OF_LINE_MINUTIAE {
			return true
		}
	}
	return false
}

func (t *STTokenBase) setDiagnostics(diagnostics []STNodeDiagnostic) {
	t.diagnostics = diagnostics
	if len(diagnostics) > 0 {
		t.flags = t.flags | HAS_DIAGNOSTIC
	} else {
		t.flags = t.flags &^ HAS_DIAGNOSTIC
	}
}

func (s *STNodeList) Get(i int) STNode {
	rangeCheck(i, s.Size())
	return s.children[i]
}

func (s *STNodeList) Size() int {
	return len(s.children)
}

func (s *STNodeList) add(node STNode) *STNodeList {
	s.children = append(s.children, node)
	s.updateWidth(s.children)
	s.updateDiagnostics(s.children)
	return s
}

// AddAllAt inserts nodes at specified index (prepend when index=0)
// Port from Java STNodeList.java:69-78 (Java addAll(int, Collection))
func (s *STNodeList) AddAllAt(index int, nodes []STNode) *STNodeList {
	// Create new array with increased capacity
	newLength := s.Size() + len(nodes)
	newBuckets := make([]STNode, newLength)

	// Copy elements before insertion point
	copy(newBuckets[0:index], s.children[0:index])

	// Insert new nodes
	copy(newBuckets[index:index+len(nodes)], nodes)

	// Copy remaining elements after insertion point
	copy(newBuckets[index+len(nodes):], s.children[index:s.Size()])

	return createNodeAndAddChildren(&STNodeList{STNodeBase: *s.copy(), children: newBuckets}, newBuckets...).(*STNodeList)
}

// AddAll appends nodes to end of list
// Port from Java STNodeList.java:80-88 (Java addAll(Collection))
func (s *STNodeList) AddAll(nodes []STNode) *STNodeList {
	newLength := s.Size() + len(nodes)
	newBuckets := make([]STNode, newLength)

	// Copy existing elements
	copy(newBuckets[0:s.Size()], s.children[0:s.Size()])

	// Append new nodes
	copy(newBuckets[s.Size():], nodes)

	return createNodeAndAddChildren(&STNodeList{STNodeBase: *s.copy(), children: newBuckets}, newBuckets...).(*STNodeList)
}

func (s *STNodeList) BucketCount() int {
	return s.Size()
}

func (s STNodeDiagnostic) DiagnosticCode() diagnostics.DiagnosticCode {
	return s.code
}

// Modification methods
func Replace(current STNode, target STNode, replacement STNode) STNode {
	// TODO: this is doing value comparison which is super expensive, need to think of a better way to do this
	_, result := replaceInner(current, target, replacement)
	return result
}

func replaceAll(current []STNode, target STNode, replacement STNode) (bool, []STNode) {
	modified := false
	var result []STNode
	for _, each := range current {
		m, e := replaceInner(each, target, replacement)
		modified = modified || m
		result = append(result, e)

	}
	return modified, result
}

func AddSyntaxDiagnostic[T STNode](node T, diagnostic STNodeDiagnostic) T {
	return AddSyntaxDiagnostics(node, []STNodeDiagnostic{diagnostic})
}

func AddSyntaxDiagnostics[T STNode](node T, diagnostics []STNodeDiagnostic) T {
	if len(diagnostics) == 0 {
		return node
	}

	oldDiagnostics := node.Diagnostics()
	if len(oldDiagnostics) == 0 {
		return modifyWithDiagnostics(node, diagnostics)
	}

	// Merge all diagnostics
	newDiagnostics := make([]STNodeDiagnostic, len(oldDiagnostics))
	copy(newDiagnostics, oldDiagnostics)
	newDiagnostics = append(newDiagnostics, diagnostics...)
	return modifyWithDiagnostics(node, newDiagnostics)
}

func modifyWithDiagnostics[T STNode](base T, diagnostics []STNodeDiagnostic) T {
	// Get the underlying value from the interface
	baseValue := reflect.ValueOf(base)
	// TODO: think of a better way to do this. We need to avoid mutating via the pointer
	if baseValue.Kind() == reflect.Pointer {
		// If it's a pointer, create a new instance and copy the struct
		elemType := baseValue.Elem().Type()
		newValue := reflect.New(elemType)
		// Copy all fields from the original to the new instance
		newValue.Elem().Set(baseValue.Elem())
		// Get the new instance as STNode and set diagnostics
		// We can't directly assert to T (type parameter), so we assert to STNode first
		newNode := newValue.Interface().(STNode)
		newNode.setDiagnostics(diagnostics)
		// Convert back to T - this is safe because T is constrained to STNode
		return any(newNode).(T)
	}
	panic("expected pointer")
}

// Utility functions
func rangeCheck(index, size int) {
	if index >= size || index < 0 {
		panic(fmt.Sprintf("index out of bounds: %d, size: %d", index, size))
	}
}

// getLiteralTokenName returns the name for literal tokens used in S-expression format
func getLiteralTokenName(kind common.SyntaxKind) string {
	switch kind {
	case common.IDENTIFIER_TOKEN:
		return "ident"
	case common.STRING_LITERAL_TOKEN:
		return "string"
	case common.DECIMAL_INTEGER_LITERAL_TOKEN:
		return "int"
	case common.HEX_INTEGER_LITERAL_TOKEN:
		return "hexInt"
	case common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
		return "float"
	case common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		return "hexFloat"
	case common.XML_TEXT_CONTENT:
		return "xmlText"
	case common.TEMPLATE_STRING:
		return "templateString"
	case common.PROMPT_CONTENT:
		return "promptContent"
	default:
		return ""
	}
}

// kindStrValue returns a string representation of the SyntaxKind.
// If StrValue() returns a non-empty string, it uses that.
// Otherwise, it returns the Go constant name for known values, or falls back to the tag number.
func kindStrValue(kind common.SyntaxKind) string {
	strVal := kind.StrValue()
	if strVal != "" {
		return strVal
	}

	// For values where StrValue() returns empty, return the constant name
	switch kind {
	case common.INVALID:
		return "INVALID"
	case common.MODULE_PART:
		return "MODULE_PART"
	case common.EOF_TOKEN:
		return "EOF_TOKEN"
	case common.LIST:
		return "LIST"
	case common.NONE:
		return "NONE"
	default:
		// Fall back to tag number for unknown values
		return fmt.Sprintf("%d", kind.Tag())
	}
}

// toSexpr converts an STNode to S-expression format: (Kind width flags (diagnostics) *children)
func ToSexpr(node STNode) string {
	return toSexprIndented(node, 0)
}

// toSexprIndented converts an STNode to S-expression format with indentation
func toSexprIndented(node STNode, indentLevel int) string {
	if !IsSTNodePresent(node) {
		return "nil"
	}

	var builder strings.Builder
	nextIndent := strings.Repeat("  ", indentLevel+1)

	// Start the S-expression: (
	builder.WriteString("(")

	// Kind
	kind := node.Kind()
	kindStr := kindStrValue(kind)
	// Special case for literal tokens
	literalName := getLiteralTokenName(kind)
	if literalName != "" {
		var text string
		// Try STIdentifierToken first
		if identToken, ok := node.(*STIdentifierToken); ok {
			text = identToken.text
		} else if literalToken, ok := node.(*STLiteralValueToken); ok {
			// Try STLiteralValueToken for other literal tokens
			text = literalToken.text
		} else {
			// Can this happen?
			text = "UNKNOWN"
		}
		kindStr = fmt.Sprintf("%s, \"%s\"", literalName, text)
	}
	builder.WriteString(kindStr)

	// Width
	builder.WriteString(" ")
	fmt.Fprintf(&builder, "%d", node.Width())

	// Flags (hex format)
	builder.WriteString(" ")
	fmt.Fprintf(&builder, "0x%02x", node.Flags())

	// Diagnostics
	builder.WriteString(" (")
	diagnostics := node.Diagnostics()
	for i, diag := range diagnostics {
		if i > 0 {
			builder.WriteString(" ")
		}
		// Format diagnostic as: DiagnosticId arg1 arg2 ...
		builder.WriteString(diag.code.DiagnosticId())
		for _, arg := range diag.args {
			builder.WriteString(" ")
			fmt.Fprintf(&builder, "%v", arg)
		}
	}
	builder.WriteString(")")

	// Children
	children := node.ChildBuckets()
	// Add each child on a new line with indentation
	for _, child := range children {
		builder.WriteString("\n")
		builder.WriteString(nextIndent)
		builder.WriteString(toSexprIndented(child, indentLevel+1))
	}

	builder.WriteString(")")
	return builder.String()
}

type direction uint8

const (
	forward direction = iota
	backward
)

// Flag constants for STNode
const (
	HAS_DIAGNOSTIC uint8 = 1 << 1 // 0x02
	IS_MISSING     uint8 = 1 << 2 // 0x04
)

// isFlagSet checks whether the given flag is set in the given flags.
func isFlagSet(flags uint8, flag uint8) bool {
	return (flags & flag) != 0
}

func IsSTNodeList(child STNode) bool {
	return child.Kind() == common.LIST
}

func IsSTNodePresent(child STNode) bool {
	return child != nil
}

func isToken(node STNode) bool {
	_, ok := node.(STToken)
	return ok
}

// CloneWithLeadingInvalidNodeMinutiae clones a node/token with invalid node as leading minutiae
// Port from Java SyntaxErrors.java:457-465 (STNode) and 476-484 (STToken)
func CloneWithLeadingInvalidNodeMinutiae(toClone STNode, invalidNode STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) STNode {
	// Check if toClone is a token - if so, handle directly (Java line 476-484)
	if token, ok := toClone.(STToken); ok {
		minutiaeList := convertInvalidNodeToMinutiae(invalidNode, diagnosticCode, args...)
		leadingMinutiae := token.LeadingMinutiae().(*STNodeList)
		leadingMinutiae = leadingMinutiae.AddAllAt(0, minutiaeList) // Prepend at index 0
		return token.ModifyWith(leadingMinutiae, token.TrailingMinutiae())
	}

	// Otherwise it's a non-token STNode (Java line 457-465)
	firstToken := FirstToken(toClone)
	firstTokenWithInvalidNodeMinutiae := CloneWithLeadingInvalidNodeMinutiae(
		firstToken, invalidNode, diagnosticCode, args...).(STToken)
	return Replace(toClone, firstToken, firstTokenWithInvalidNodeMinutiae)
}

// CloneWithLeadingInvalidNodeMinutiaeWithoutDiagnostics clones without adding diagnostics
// Port from Java SyntaxErrors.java:444-446
func CloneWithLeadingInvalidNodeMinutiaeWithoutDiagnostics(toClone STNode, invalidNode STNode) STNode {
	return CloneWithLeadingInvalidNodeMinutiae(toClone, invalidNode, nil)
}

func CreateMissingTokenWithDiagnosticsFromParserRules(expectedKind common.SyntaxKind, currentCtx common.ParserRuleContext) STToken {
	return CreateMissingTokenWithDiagnostics(expectedKind, currentCtx.GetErrorCode())
}

func CreateMissingTokenWithDiagnostics(expectedKind common.SyntaxKind, diagnosticCode diagnostics.DiagnosticCode) STToken {
	diagnosticList := []STNodeDiagnostic{CreateDiagnosticWithArgs(diagnosticCode)}
	return CreateMissingToken(expectedKind, diagnosticList)
}

func CreateDiagnosticWithArgs(diagnosticCode diagnostics.DiagnosticCode, args ...any) STNodeDiagnostic {
	return STNodeDiagnostic{
		code: diagnosticCode,
		args: args,
	}
}

func CreateMissingToken(expectedKind common.SyntaxKind, diagnosticList []STNodeDiagnostic) STToken {
	flags := uint8(0)
	if len(diagnosticList) > 0 {
		flags = flags | HAS_DIAGNOSTIC
	}
	flags = flags | IS_MISSING
	return &STMissingToken{
		STTokenBase: STTokenBase{
			kind:             expectedKind,
			leadingMinutiae:  CreateEmptyNodeList(),
			trailingMinutiae: CreateEmptyNodeList(),
			flags:            flags,
			diagnostics:      diagnosticList,
		},
	}
}

func AddDiagnostic[T STNode](node T, diagnosticCode diagnostics.DiagnosticCode, args ...any) T {
	return AddSyntaxDiagnostic(node, CreateDiagnostic(diagnosticCode, args...))
}

func CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(toClone STNode, invalidNode STNode) STNode {
	return CloneWithTrailingInvalidNodeMinutiae(toClone, invalidNode, nil, nil)
}

// CloneWithTrailingInvalidNodeMinutiae clones a node/token with invalid node as trailing minutiae
// Port from Java SyntaxErrors.java:506-514 (STNode) and 525-533 (STToken)
func CloneWithTrailingInvalidNodeMinutiae(toClone STNode, invalidNode STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) STNode {
	// Check if toClone is a token - if so, handle directly (Java line 525-533)
	if token, ok := toClone.(STToken); ok {
		minutiaeList := convertInvalidNodeToMinutiae(invalidNode, diagnosticCode, args...)
		trailingMinutiae := token.TrailingMinutiae().(*STNodeList)
		trailingMinutiae = trailingMinutiae.AddAll(minutiaeList) // Append to end
		return token.ModifyWith(token.LeadingMinutiae(), trailingMinutiae)
	}

	// Otherwise it's a non-token STNode (Java line 506-514)
	lastToken := LastToken(toClone)
	lastTokenWithInvalidNodeMinutiae := CloneWithTrailingInvalidNodeMinutiae(
		lastToken, invalidNode, diagnosticCode, args...).(STToken)
	return Replace(toClone, lastToken, lastTokenWithInvalidNodeMinutiae)
}

func ToToken(node STNode) STToken {
	tok, ok := node.(STToken)
	if !ok {
		panic("node is not a STToken")
	}
	return tok
}

// addMinutiaeToList adds minutiae node(s) to a list
// If minutiae is a STNodeList, adds all elements; otherwise adds single node
// Port from Java SyntaxErrors.java:579-592
func addMinutiaeToList(list []STNode, minutiae STNode) []STNode {
	if !IsSTNodeList(minutiae) {
		return append(list, minutiae)
	}

	minutiaeList := minutiae.(*STNodeList)
	for i := 0; i < minutiaeList.Size(); i++ {
		element := minutiaeList.Get(i)
		if IsSTNodePresent(element) {
			list = append(list, element)
		}
	}
	return list
}

// convertInvalidNodeToMinutiae converts an invalid node to a list of minutiae nodes
// Port from Java SyntaxErrors.java:557-577
func convertInvalidNodeToMinutiae(invalidNode STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) []STNode {
	minutiaeList := []STNode{}
	tokens := Tokens(invalidNode)

	for _, token := range tokens {
		// Add leading minutiae
		minutiaeList = addMinutiaeToList(minutiaeList, token.LeadingMinutiae())

		if !token.IsMissing() {
			// Add diagnostic to first invalid token only
			if diagnosticCode != nil {
				token = AddDiagnostic(token, diagnosticCode, args...)
				diagnosticCode = nil // Only add to first token
			}

			// Create token with no minutiae and wrap as invalid node minutiae
			tokenWithNoMinutiae := token.ModifyWith(CreateEmptyNodeList(), CreateEmptyNodeList())
			minutiaeList = append(minutiaeList, CreateInvalidNodeMinutiae(tokenWithNoMinutiae))
		}

		// Add trailing minutiae
		minutiaeList = addMinutiaeToList(minutiaeList, token.TrailingMinutiae())
	}

	return minutiaeList
}

func CreateEmptyNode() STNode {
	return nil
}

func CreateMarkdownDocumentationLineNode(kind common.SyntaxKind, hashToken STNode, documentElements STNode) STNode {
	return createNodeAndAddChildren(&STMarkdownDocumentationLineNode{
		STDocumentationNode: &STNodeBase{
			kind: kind,
		},
		HashToken:        hashToken,
		DocumentElements: documentElements,
	}, hashToken, documentElements)
}

func CreateMarkdownDocumentationNode(documentationLines STNode) STNode {
	return createNodeAndAddChildren(&STMarkdownDocumentationNode{
		STDocumentationNode: &STNodeBase{
			kind: common.MARKDOWN_DOCUMENTATION,
		},
		DocumentationLines: documentationLines,
	}, documentationLines)
}

func CreateMarkdownParameterDocumentationLineNode(kind common.SyntaxKind, hashToken STNode, plusToken STNode, parameterName STNode, minusToken STNode, documentElements STNode) STNode {
	return createNodeAndAddChildren(&STMarkdownParameterDocumentationLineNode{
		STDocumentationNode: &STNodeBase{
			kind: kind,
		},
		HashToken:        hashToken,
		PlusToken:        plusToken,
		ParameterName:    parameterName,
		MinusToken:       minusToken,
		DocumentElements: documentElements,
	}, hashToken, plusToken, parameterName, minusToken, documentElements)
}

func CreateBallerinaNameReferenceNode(referenceType STNode, startBacktick STNode, nameReference STNode, endBacktick STNode) STNode {
	return createNodeAndAddChildren(&STBallerinaNameReferenceNode{
		STDocumentationNode: &STNodeBase{
			kind: common.BALLERINA_NAME_REFERENCE,
		},
		ReferenceType: referenceType,
		StartBacktick: startBacktick,
		NameReference: nameReference,
		EndBacktick:   endBacktick,
	}, referenceType, startBacktick, nameReference, endBacktick)
}

func CreateInlineCodeReferenceNode(startBacktick STNode, codeReference STNode, endBacktick STNode) STNode {
	return createNodeAndAddChildren(&STInlineCodeReferenceNode{
		STDocumentationNode: &STNodeBase{
			kind: common.INLINE_CODE_REFERENCE,
		},
		StartBacktick: startBacktick,
		CodeReference: codeReference,
		EndBacktick:   endBacktick,
	}, startBacktick, codeReference, endBacktick)
}

func CreateMarkdownCodeBlockNode(startLineHashToken STNode, startBacktick STNode, langAttribute STNode, codeLines STNode, endLineHashToken STNode, endBacktick STNode) STNode {
	return createNodeAndAddChildren(&STMarkdownCodeBlockNode{
		STDocumentationNode: &STNodeBase{
			kind: common.MARKDOWN_CODE_BLOCK,
		},
		StartLineHashToken: startLineHashToken,
		StartBacktick:      startBacktick,
		LangAttribute:      langAttribute,
		CodeLines:          codeLines,
		EndLineHashToken:   endLineHashToken,
		EndBacktick:        endBacktick,
	}, startLineHashToken, startBacktick, langAttribute, codeLines, endLineHashToken, endBacktick)
}

func CreateMarkdownCodeLineNode(hashToken STNode, codeDescription STNode) STNode {
	return createNodeAndAddChildren(&STMarkdownCodeLineNode{
		STDocumentationNode: &STNodeBase{
			kind: common.MARKDOWN_CODE_LINE,
		},
		HashToken:       hashToken,
		CodeDescription: codeDescription,
	}, hashToken, codeDescription)
}

func CreateModulePartNode(imports STNode, members STNode, eofToken STNode) STNode {
	return createNodeAndAddChildren(&STModulePart{
		STNode: &STNodeBase{
			kind: common.MODULE_PART,
		},
		Imports:  imports,
		Members:  members,
		EofToken: eofToken,
	}, imports, members, eofToken)
}

func CreateMetadataNode(documentationString STNode, annotations STNode) STNode {
	return createNodeAndAddChildren(&STMetadataNode{
		STNode: &STNodeBase{
			kind: common.METADATA,
		},
		DocumentationString: documentationString,
		Annotations:         annotations,
	}, documentationString, annotations)
}

func CreateAmbiguousCollectionNode(kind common.SyntaxKind, collectionStartToken STNode, members []STNode, collectionEndToken STNode) *STAmbiguousCollectionNode {
	children := make([]STNode, 0, len(members)+2)
	children = append(children, collectionStartToken)
	children = append(children, members...)
	children = append(children, collectionEndToken)
	return createNodeAndAddChildren(&STAmbiguousCollectionNode{
		STNodeBase: STNodeBase{
			kind: kind,
		},
		CollectionStartToken: collectionStartToken,
		CollectionEndToken:   collectionEndToken,
		Members:              members,
	}, children...).(*STAmbiguousCollectionNode)
}

func createNodeAndAddChildren(base STNode, children ...STNode) STNode {
	base.addChildren(children...)
	return base
}

func UpdateAllNodesInNodeListWithDiagnostic(nodeList *STNodeList, diagnosticCode diagnostics.DiagnosticCode) STNode {
	newList := make([]STNode, nodeList.Size())
	for i := 0; i < nodeList.Size(); i++ {
		newList[i] = AddDiagnostic(nodeList.Get(i), diagnosticCode)
	}
	return CreateNodeList(newList...)
}

func CreateImportOrgNameNode(orgName STNode, slashToken STNode) STNode {
	return createNodeAndAddChildren(&STImportOrgNameNode{
		STNode: &STNodeBase{
			kind: common.IMPORT_ORG_NAME,
		},
		OrgName:    orgName,
		SlashToken: slashToken,
	}, orgName, slashToken)
}

func CreateImportDeclarationNode(importKeyword STNode, orgName STNode, moduleName STNode, prefix STNode, semicolon STNode) STNode {
	return createNodeAndAddChildren(&STImportDeclarationNode{
		STNode: &STNodeBase{
			kind: common.IMPORT_DECLARATION,
		},
		ImportKeyword: importKeyword,
		OrgName:       orgName,
		ModuleName:    moduleName,
		Prefix:        prefix,
		Semicolon:     semicolon,
	}, importKeyword, orgName, moduleName, prefix, semicolon)
}

// FIXME:
func CreateBuiltinSimpleNameReferenceNode(kind common.SyntaxKind, name STNode) STNode {
	return createNodeAndAddChildren(&STBuiltinSimpleNameReferenceNode{
		STNameReferenceNode: &STNodeBase{
			kind: kind,
		},
		Name: name,
	}, name)
}

func CreateImportPrefixNode(asKeyword STNode, prefix STNode) STNode {
	return createNodeAndAddChildren(&STImportPrefixNode{
		STNode: &STNodeBase{
			kind: common.IMPORT_PREFIX,
		},
		AsKeyword: asKeyword,
		Prefix:    prefix,
	}, asKeyword, prefix)
}

func CreateSpecificFieldNode(readonlyKeyword STNode, fieldName STNode, colon STNode, valueExpr STNode) STNode {
	return createNodeAndAddChildren(&STSpecificFieldNode{
		STMappingFieldNode: &STNodeBase{
			kind: common.SPECIFIC_FIELD,
		},
		ReadonlyKeyword: readonlyKeyword,
		FieldName:       fieldName,
		Colon:           colon,
		ValueExpr:       valueExpr,
	}, readonlyKeyword, fieldName, colon, valueExpr)
}

func CreateCaptureBindingPatternNode(variableName STNode) STNode {
	return createNodeAndAddChildren(&STCaptureBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.CAPTURE_BINDING_PATTERN,
		},
		VariableName: variableName,
	}, variableName)
}

func CreateTypedBindingPatternNode(typeDescriptor STNode, bindingPattern STNode) STNode {
	return createNodeAndAddChildren(&STTypedBindingPatternNode{
		STNode: &STNodeBase{
			kind: common.TYPED_BINDING_PATTERN,
		},
		TypeDescriptor: typeDescriptor,
		BindingPattern: bindingPattern,
	}, typeDescriptor, bindingPattern)
}

func CreateSimpleNameReferenceNode(name STNode) STNode {
	return createNodeAndAddChildren(&STSimpleNameReferenceNode{
		STNameReferenceNode: &STNodeBase{
			kind: common.SIMPLE_NAME_REFERENCE,
		},
		Name: name,
	}, name)
}

func CreateFieldBindingPatternFullNode(variableName STNode, colon STNode, bindingPattern STNode) STNode {
	return createNodeAndAddChildren(&STFieldBindingPatternFullNode{
		STFieldBindingPatternNode: &STNodeBase{
			kind: common.FIELD_BINDING_PATTERN,
		},
		VariableName:   variableName,
		Colon:          colon,
		BindingPattern: bindingPattern,
	}, variableName, colon, bindingPattern)
}

func CreateRestBindingPatternNode(ellipsisToken STNode, variableName STNode) STNode {
	return createNodeAndAddChildren(&STRestBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.REST_BINDING_PATTERN,
		},
		EllipsisToken: ellipsisToken,
		VariableName:  variableName,
	}, ellipsisToken, variableName)
}

func CreateSpreadFieldNode(ellipsis STNode, valueExpr STNode) STNode {
	return createNodeAndAddChildren(&STSpreadFieldNode{
		STMappingFieldNode: &STNodeBase{
			kind: common.SPREAD_FIELD,
		},
		Ellipsis:  ellipsis,
		ValueExpr: valueExpr,
	}, ellipsis, valueExpr)
}

func CreateSingletonTypeDescriptorNode(simpleContExprNode STNode) STNode {
	return createNodeAndAddChildren(&STSingletonTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.SINGLETON_TYPE_DESC,
		},
		SimpleContExprNode: simpleContExprNode,
	}, simpleContExprNode)
}

func CreateListConstructorExpressionNode(openBracket STNode, expressions STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STListConstructorExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.LIST_CONSTRUCTOR,
		},
		OpenBracket:  openBracket,
		Expressions:  expressions,
		CloseBracket: closeBracket,
	}, openBracket, expressions, closeBracket)
}

func CreateListBindingPatternNode(openBracket STNode, bindingPatterns STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STListBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.LIST_BINDING_PATTERN,
		},
		OpenBracket:     openBracket,
		BindingPatterns: bindingPatterns,
		CloseBracket:    closeBracket,
	}, openBracket, bindingPatterns, closeBracket)
}

func CreateInferredTypedescDefaultNode(ltToken STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STInferredTypedescDefaultNode{
		STExpressionNode: &STNodeBase{
			kind: common.INFERRED_TYPEDESC_DEFAULT,
		},
		LtToken: ltToken,
		GtToken: gtToken,
	}, ltToken, gtToken)
}

func CreateBinaryExpressionNode(kind common.SyntaxKind, lhsExpr STNode, operator STNode, rhsExpr STNode) STNode {
	return createNodeAndAddChildren(&STBinaryExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: kind,
		},
		LhsExpr:  lhsExpr,
		Operator: operator,
		RhsExpr:  rhsExpr,
	}, lhsExpr, operator, rhsExpr)
}

func CreateFieldAccessExpressionNode(expression STNode, dotToken STNode, fieldName STNode) STNode {
	return createNodeAndAddChildren(&STFieldAccessExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.FIELD_ACCESS,
		},
		Expression: expression,
		DotToken:   dotToken,
		FieldName:  fieldName,
	}, expression, dotToken, fieldName)
}

func CreateIndexedExpressionNode(containerExpression STNode, openBracket STNode, keyExpression STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STIndexedExpressionNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.INDEXED_EXPRESSION,
		},
		ContainerExpression: containerExpression,
		OpenBracket:         openBracket,
		KeyExpression:       keyExpression,
		CloseBracket:        closeBracket,
	}, containerExpression, openBracket, keyExpression, closeBracket)
}

func CreateTypeTestExpressionNode(expression STNode, isKeyword STNode, typeDescriptor STNode) STNode {
	return createNodeAndAddChildren(&STTypeTestExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.TYPE_TEST_EXPRESSION,
		},
		Expression:     expression,
		IsKeyword:      isKeyword,
		TypeDescriptor: typeDescriptor,
	}, expression, isKeyword, typeDescriptor)
}

func CreateConditionalExpressionNode(lhsExpression STNode, questionMarkToken STNode, middleExpression STNode, colonToken STNode, endExpression STNode) STNode {
	return createNodeAndAddChildren(&STConditionalExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.CONDITIONAL_EXPRESSION,
		},
		LhsExpression:     lhsExpression,
		QuestionMarkToken: questionMarkToken,
		MiddleExpression:  middleExpression,
		ColonToken:        colonToken,
		EndExpression:     endExpression,
	}, lhsExpression, questionMarkToken, middleExpression, colonToken, endExpression)
}

func CreateRemoteMethodCallActionNode(expression STNode, rightArrowToken STNode, methodName STNode, openParenToken STNode, arguments STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STRemoteMethodCallActionNode{
		STActionNode: &STNodeBase{
			kind: common.REMOTE_METHOD_CALL_ACTION,
		},
		Expression:      expression,
		RightArrowToken: rightArrowToken,
		MethodName:      methodName,
		OpenParenToken:  openParenToken,
		Arguments:       arguments,
		CloseParenToken: closeParenToken,
	}, expression, rightArrowToken, methodName, openParenToken, arguments, closeParenToken)
}

func CreateAsyncSendActionNode(expression STNode, rightArrowToken STNode, peerWorker STNode) STNode {
	return createNodeAndAddChildren(&STAsyncSendActionNode{
		STActionNode: &STNodeBase{
			kind: common.ASYNC_SEND_ACTION,
		},
		Expression:      expression,
		RightArrowToken: rightArrowToken,
		PeerWorker:      peerWorker,
	}, expression, rightArrowToken, peerWorker)
}

func CreateFunctionCallExpressionNode(functionName STNode, openParenToken STNode, arguments STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STFunctionCallExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.FUNCTION_CALL,
		},
		FunctionName:    functionName,
		OpenParenToken:  openParenToken,
		Arguments:       arguments,
		CloseParenToken: closeParenToken,
	}, functionName, openParenToken, arguments, closeParenToken)
}

func CreateArrayTypeDescriptorNode(memberTypeDesc STNode, dimensions STNode) STNode {
	return createNodeAndAddChildren(&STArrayTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.ARRAY_TYPE_DESC,
		},
		MemberTypeDesc: memberTypeDesc,
		Dimensions:     dimensions,
	}, memberTypeDesc, dimensions)
}

func CreateOptionalTypeDescriptorNode(typeDescriptor STNode, questionMarkToken STNode) STNode {
	return createNodeAndAddChildren(&STOptionalTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.OPTIONAL_TYPE_DESC,
		},
		TypeDescriptor:    typeDescriptor,
		QuestionMarkToken: questionMarkToken,
	}, typeDescriptor, questionMarkToken)
}

func CreateMemberTypeDescriptorNode(annotations STNode, typeDescriptor STNode) STNode {
	return createNodeAndAddChildren(&STMemberTypeDescriptorNode{
		STNode: &STNodeBase{
			kind: common.MEMBER_TYPE_DESC,
		},
		Annotations:    annotations,
		TypeDescriptor: typeDescriptor,
	}, annotations, typeDescriptor)
}

func CreateNilTypeDescriptorNode(openParenToken STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STNilTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.NIL_TYPE_DESC,
		},
		OpenParenToken:  openParenToken,
		CloseParenToken: closeParenToken,
	}, openParenToken, closeParenToken)
}

func CreateParenthesisedTypeDescriptorNode(openParenToken STNode, typedesc STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STParenthesisedTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.PARENTHESISED_TYPE_DESC,
		},
		OpenParenToken:  openParenToken,
		Typedesc:        typedesc,
		CloseParenToken: closeParenToken,
	}, openParenToken, typedesc, closeParenToken)
}

func CreateTupleTypeDescriptorNode(openBracket STNode, memberTypeDesc STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STTupleTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.TUPLE_TYPE_DESC,
		},
		OpenBracketToken:  openBracket,
		MemberTypeDesc:    memberTypeDesc,
		CloseBracketToken: closeBracket,
	}, openBracket, memberTypeDesc, closeBracket)
}

func CreateMappingBindingPatternNode(openBrace STNode, fieldBindingPatterns STNode, closeBrace STNode) STNode {
	return createNodeAndAddChildren(&STMappingBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.MAPPING_BINDING_PATTERN,
		},
		OpenBrace:            openBrace,
		FieldBindingPatterns: fieldBindingPatterns,
		CloseBrace:           closeBrace,
	}, openBrace, fieldBindingPatterns, closeBrace)
}

func CreateFieldBindingPatternVarnameNode(variableName STNode) STNode {
	return createNodeAndAddChildren(&STFieldBindingPatternVarnameNode{
		STFieldBindingPatternNode: &STNodeBase{
			kind: common.FIELD_BINDING_PATTERN,
		},
		VariableName: variableName,
	}, variableName)
}

func CreateErrorBindingPatternNode(errorKeyword STNode, typeReference STNode, openParenthesis STNode, argListBindingPatterns STNode, closeParenthesis STNode) STNode {
	return createNodeAndAddChildren(&STErrorBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.ERROR_BINDING_PATTERN,
		},
		ErrorKeyword:           errorKeyword,
		TypeReference:          typeReference,
		OpenParenthesis:        openParenthesis,
		ArgListBindingPatterns: argListBindingPatterns,
		CloseParenthesis:       closeParenthesis,
	}, errorKeyword, typeReference, openParenthesis, argListBindingPatterns, closeParenthesis)
}

func CreateNamedArgBindingPatternNode(argName STNode, equalsToken STNode, bindingPattern STNode) STNode {
	return createNodeAndAddChildren(&STNamedArgBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.NAMED_ARG_BINDING_PATTERN,
		},
		ArgName:        argName,
		EqualsToken:    equalsToken,
		BindingPattern: bindingPattern,
	}, argName, equalsToken, bindingPattern)
}

func CreateRestParameterNode(annotations STNode, typeName STNode, ellipsisToken STNode, paramName STNode) STNode {
	return createNodeAndAddChildren(&STRestParameterNode{
		STParameterNode: &STNodeBase{
			kind: common.REST_PARAM,
		},
		Annotations:   annotations,
		TypeName:      typeName,
		EllipsisToken: ellipsisToken,
		ParamName:     paramName,
	}, annotations, typeName, ellipsisToken, paramName)
}

func CreateIncludedRecordParameterNode(annotations STNode, asteriskToken STNode, typeName STNode, paramName STNode) STNode {
	return createNodeAndAddChildren(&STIncludedRecordParameterNode{
		STParameterNode: &STNodeBase{
			kind: common.INCLUDED_RECORD_PARAM,
		},
		Annotations:   annotations,
		AsteriskToken: asteriskToken,
		TypeName:      typeName,
		ParamName:     paramName,
	}, annotations, asteriskToken, typeName, paramName)
}

func CreateRequiredParameterNode(annotations STNode, typeName STNode, paramName STNode) STNode {
	return createNodeAndAddChildren(&STRequiredParameterNode{
		STParameterNode: &STNodeBase{
			kind: common.REQUIRED_PARAM,
		},
		Annotations: annotations,
		TypeName:    typeName,
		ParamName:   paramName,
	}, annotations, typeName, paramName)
}

func CreateDefaultableParameterNode(annotations STNode, typeName STNode, paramName STNode, equalsToken STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STDefaultableParameterNode{
		STParameterNode: &STNodeBase{
			kind: common.DEFAULTABLE_PARAM,
		},
		Annotations: annotations,
		TypeName:    typeName,
		ParamName:   paramName,
		EqualsToken: equalsToken,
		Expression:  expression,
	}, annotations, typeName, paramName, equalsToken, expression)
}

func CreateReturnTypeDescriptorNode(returnsKeyword STNode, annotations STNode, typeNode STNode) STNode {
	return createNodeAndAddChildren(&STReturnTypeDescriptorNode{
		STNode: &STNodeBase{
			kind: common.RETURN_TYPE_DESCRIPTOR,
		},
		ReturnsKeyword: returnsKeyword,
		Annotations:    annotations,
		Type:           typeNode,
	}, returnsKeyword, annotations, typeNode)
}

func CreateFunctionDefinitionNode(kind common.SyntaxKind, metadata STNode, qualifierList STNode, functionKeyword STNode, functionName STNode, relativeResourcePath STNode, functionSignature STNode, functionBody STNode) STNode {
	return createNodeAndAddChildren(&STFunctionDefinition{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: kind,
		},
		Metadata:             metadata,
		QualifierList:        qualifierList,
		FunctionKeyword:      functionKeyword,
		FunctionName:         functionName,
		RelativeResourcePath: relativeResourcePath,
		FunctionSignature:    functionSignature,
		FunctionBody:         functionBody,
	}, metadata, qualifierList, functionKeyword, functionName, relativeResourcePath, functionSignature, functionBody)
}

func CreateMethodDeclarationNode(kind common.SyntaxKind, metadata STNode, qualifierList STNode, functionKeyword STNode, methodName STNode, relativeResourcePath STNode, methodSignature STNode, semicolon STNode) STNode {
	return createNodeAndAddChildren(&STMethodDeclarationNode{
		STNode: &STNodeBase{
			kind: kind,
		},
		Metadata:             metadata,
		QualifierList:        qualifierList,
		FunctionKeyword:      functionKeyword,
		MethodName:           methodName,
		RelativeResourcePath: relativeResourcePath,
		MethodSignature:      methodSignature,
		Semicolon:            semicolon,
	}, metadata, qualifierList, functionKeyword, methodName, relativeResourcePath, methodSignature, semicolon)
}

func CreateFunctionSignatureNode(openParenToken STNode, parameters STNode, closeParenToken STNode, returnTypeDesc STNode) STNode {
	return createNodeAndAddChildren(&STFunctionSignatureNode{
		STNode: &STNodeBase{
			kind: common.FUNCTION_SIGNATURE,
		},
		OpenParenToken:  openParenToken,
		Parameters:      parameters,
		CloseParenToken: closeParenToken,
		ReturnTypeDesc:  returnTypeDesc,
	}, openParenToken, parameters, closeParenToken, returnTypeDesc)
}

func CreateFunctionTypeDescriptorNode(qualifierList STNode, functionKeyword STNode, functionSignature STNode) STNode {
	return createNodeAndAddChildren(&STFunctionTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.FUNCTION_TYPE_DESC,
		},
		QualifierList:     qualifierList,
		FunctionKeyword:   functionKeyword,
		FunctionSignature: functionSignature,
	}, qualifierList, functionKeyword, functionSignature)
}

func CreateDistinctTypeDescriptorNode(distinctKeyword STNode, typeDescriptor STNode) STNode {
	return createNodeAndAddChildren(&STDistinctTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.DISTINCT_TYPE_DESC,
		},
		DistinctKeyword: distinctKeyword,
		TypeDescriptor:  typeDescriptor,
	}, distinctKeyword, typeDescriptor)
}

func CreateNamedWorkerDeclarator(workerInitStatements STNode, namedWorkerDeclarations STNode) STNode {
	return createNodeAndAddChildren(&STNamedWorkerDeclarator{
		STNode: &STNodeBase{
			kind: common.NAMED_WORKER_DECLARATOR,
		},
		WorkerInitStatements:    workerInitStatements,
		NamedWorkerDeclarations: namedWorkerDeclarations,
	}, workerInitStatements, namedWorkerDeclarations)
}

func CreateFunctionBodyBlockNode(openBraceToken STNode, namedWorkerDeclarator STNode, statements STNode, closeBraceToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STFunctionBodyBlockNode{
		STFunctionBodyNode: &STNodeBase{
			kind: common.FUNCTION_BODY_BLOCK,
		},
		OpenBraceToken:        openBraceToken,
		NamedWorkerDeclarator: namedWorkerDeclarator,
		Statements:            statements,
		CloseBraceToken:       closeBraceToken,
		SemicolonToken:        semicolonToken,
	}, openBraceToken, namedWorkerDeclarator, statements, closeBraceToken, semicolonToken)
}

func CreateExternalFunctionBodyNode(equalsToken STNode, annotations STNode, externalKeyword STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STExternalFunctionBodyNode{
		STFunctionBodyNode: &STNodeBase{
			kind: common.EXTERNAL_FUNCTION_BODY,
		},
		EqualsToken:     equalsToken,
		Annotations:     annotations,
		ExternalKeyword: externalKeyword,
		SemicolonToken:  semicolonToken,
	}, equalsToken, annotations, externalKeyword, semicolonToken)
}

func CreateRecordTypeDescriptorNode(recordKeyword STNode, bodyStartDelimiter STNode, fields STNode, recordRestDescriptor STNode, bodyEndDelimiter STNode) STNode {
	return createNodeAndAddChildren(&STRecordTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.RECORD_TYPE_DESC,
		},
		RecordKeyword:        recordKeyword,
		BodyStartDelimiter:   bodyStartDelimiter,
		Fields:               fields,
		RecordRestDescriptor: recordRestDescriptor,
		BodyEndDelimiter:     bodyEndDelimiter,
	}, recordKeyword, bodyStartDelimiter, fields, recordRestDescriptor, bodyEndDelimiter)
}

func CreateClassDefinitionNode(metadata STNode, visibilityQualifier STNode, classTypeQualifiers STNode, classKeyword STNode, className STNode, openBrace STNode, members STNode, closeBrace STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STClassDefinitionNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.CLASS_DEFINITION,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		ClassTypeQualifiers: classTypeQualifiers,
		ClassKeyword:        classKeyword,
		ClassName:           className,
		OpenBrace:           openBrace,
		Members:             members,
		CloseBrace:          closeBrace,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, classTypeQualifiers, classKeyword, className, openBrace, members, closeBrace, semicolonToken)
}

func CreateTypeDefinitionNode(metadata STNode, visibilityQualifier STNode, typeKeyword STNode, typeName STNode, typeDescriptor STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STTypeDefinitionNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.TYPE_DEFINITION,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		TypeKeyword:         typeKeyword,
		TypeName:            typeName,
		TypeDescriptor:      typeDescriptor,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, typeKeyword, typeName, typeDescriptor, semicolonToken)
}

func CreateTypeReferenceNode(asteriskToken STNode, typeName STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STTypeReferenceNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.TYPE_REFERENCE,
		},
		AsteriskToken:  asteriskToken,
		TypeName:       typeName,
		SemicolonToken: semicolonToken,
	}, asteriskToken, typeName, semicolonToken)
}

func CreateQualifiedNameReferenceNode(modulePrefix STNode, colon STNode, identifier STNode) STNode {
	return createNodeAndAddChildren(&STQualifiedNameReferenceNode{
		STNameReferenceNode: &STNodeBase{
			kind: common.QUALIFIED_NAME_REFERENCE,
		},
		ModulePrefix: modulePrefix,
		Colon:        colon,
		Identifier:   identifier,
	}, modulePrefix, colon, identifier)
}

func CreateObjectFieldNode(metadata STNode, visibilityQualifier STNode, qualifierList STNode, typeName STNode, fieldName STNode, equalsToken STNode, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STObjectFieldNode{
		STNode: &STNodeBase{
			kind: common.OBJECT_FIELD,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		QualifierList:       qualifierList,
		TypeName:            typeName,
		FieldName:           fieldName,
		EqualsToken:         equalsToken,
		Expression:          expression,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, qualifierList, typeName, fieldName, equalsToken, expression, semicolonToken)
}

func CreateWhereClauseNode(whereKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STWhereClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.WHERE_CLAUSE,
		},
		WhereKeyword: whereKeyword,
		Expression:   expression,
	}, whereKeyword, expression)
}

func CreateModuleVariableDeclarationNode(metadata STNode, visibilityQualifier STNode, qualifiers STNode, typedBindingPattern STNode, equalsToken STNode, initializer STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STModuleVariableDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.MODULE_VAR_DECL,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		Qualifiers:          qualifiers,
		TypedBindingPattern: typedBindingPattern,
		EqualsToken:         equalsToken,
		Initializer:         initializer,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, qualifiers, typedBindingPattern, equalsToken, initializer, semicolonToken)
}

func CreateRequiredExpressionNode(questionMarkToken STNode) STNode {
	return createNodeAndAddChildren(&STRequiredExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.REQUIRED_EXPRESSION,
		},
		QuestionMarkToken: questionMarkToken,
	}, questionMarkToken)
}

func CreateVariableDeclarationNode(annotations STNode, finalKeyword STNode, typedBindingPattern STNode, equalsToken STNode, initializer STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STVariableDeclarationNode{
		STStatementNode: &STNodeBase{
			kind: common.LOCAL_VAR_DECL,
		},
		Annotations:         annotations,
		FinalKeyword:        finalKeyword,
		TypedBindingPattern: typedBindingPattern,
		EqualsToken:         equalsToken,
		Initializer:         initializer,
		SemicolonToken:      semicolonToken,
	}, annotations, finalKeyword, typedBindingPattern, equalsToken, initializer, semicolonToken)
}

func CreateRecordRestDescriptorNode(typeName STNode, ellipsisToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STRecordRestDescriptorNode{
		STNode: &STNodeBase{
			kind: common.RECORD_REST_TYPE,
		},
		TypeName:       typeName,
		EllipsisToken:  ellipsisToken,
		SemicolonToken: semicolonToken,
	}, typeName, ellipsisToken, semicolonToken)
}

func CreateRecordFieldNode(metadata STNode, readonlyKeyword STNode, typeName STNode, fieldName STNode, questionMarkToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STRecordFieldNode{
		STNode: &STNodeBase{
			kind: common.RECORD_FIELD,
		},
		Metadata:          metadata,
		ReadonlyKeyword:   readonlyKeyword,
		TypeName:          typeName,
		FieldName:         fieldName,
		QuestionMarkToken: questionMarkToken,
		SemicolonToken:    semicolonToken,
	}, metadata, readonlyKeyword, typeName, fieldName, questionMarkToken, semicolonToken)
}

func CreateRecordFieldWithDefaultValueNode(metadata STNode, readonlyKeyword STNode, typeName STNode, fieldName STNode, equalsToken STNode, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STRecordFieldWithDefaultValueNode{
		STNode: &STNodeBase{
			kind: common.RECORD_FIELD_WITH_DEFAULT_VALUE,
		},
		Metadata:        metadata,
		ReadonlyKeyword: readonlyKeyword,
		TypeName:        typeName,
		FieldName:       fieldName,
		EqualsToken:     equalsToken,
		Expression:      expression,
		SemicolonToken:  semicolonToken,
	}, metadata, readonlyKeyword, typeName, fieldName, equalsToken, expression, semicolonToken)
}

func CreateAssignmentStatementNode(varRef STNode, equalsToken STNode, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STAssignmentStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.ASSIGNMENT_STATEMENT,
		},
		VarRef:         varRef,
		EqualsToken:    equalsToken,
		Expression:     expression,
		SemicolonToken: semicolonToken,
	}, varRef, equalsToken, expression, semicolonToken)
}

func CreateNaturalExpressionNode(constKeyword STNode, naturalKeyword STNode, parenthesizedArgList STNode, openBraceToken STNode, prompt STNode, closeBraceToken STNode) STNode {
	return createNodeAndAddChildren(&STNaturalExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.NATURAL_EXPRESSION,
		},
		ConstKeyword:         constKeyword,
		NaturalKeyword:       naturalKeyword,
		ParenthesizedArgList: parenthesizedArgList,
		OpenBraceToken:       openBraceToken,
		Prompt:               prompt,
		CloseBraceToken:      closeBraceToken,
	}, constKeyword, naturalKeyword, parenthesizedArgList, openBraceToken, prompt, closeBraceToken)
}

func CreateObjectConstructorExpressionNode(annotations STNode, objectTypeQualifiers STNode, objectKeyword STNode, typeReference STNode, openBraceToken STNode, members STNode, closeBraceToken STNode) STNode {
	return createNodeAndAddChildren(&STObjectConstructorExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.OBJECT_CONSTRUCTOR,
		},
		Annotations:          annotations,
		ObjectTypeQualifiers: objectTypeQualifiers,
		ObjectKeyword:        objectKeyword,
		TypeReference:        typeReference,
		OpenBraceToken:       openBraceToken,
		Members:              members,
		CloseBraceToken:      closeBraceToken,
	}, annotations, objectTypeQualifiers, objectKeyword, typeReference, openBraceToken, members, closeBraceToken)
}

func CreateExplicitNewExpressionNode(newKeyword STNode, typeDescriptor STNode, parenthesizedArgList STNode) STNode {
	return createNodeAndAddChildren(&STExplicitNewExpressionNode{
		STNewExpressionNode: &STNodeBase{
			kind: common.EXPLICIT_NEW_EXPRESSION,
		},
		NewKeyword:           newKeyword,
		TypeDescriptor:       typeDescriptor,
		ParenthesizedArgList: parenthesizedArgList,
	}, newKeyword, typeDescriptor, parenthesizedArgList)
}

func CreateImplicitNewExpressionNode(newKeyword STNode, parenthesizedArgList STNode) STNode {
	return createNodeAndAddChildren(&STImplicitNewExpressionNode{
		STNewExpressionNode: &STNodeBase{
			kind: common.IMPLICIT_NEW_EXPRESSION,
		},
		NewKeyword:           newKeyword,
		ParenthesizedArgList: parenthesizedArgList,
	}, newKeyword, parenthesizedArgList)
}

func CreateParenthesizedArgList(openParenToken STNode, arguments STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STParenthesizedArgList{
		STNode: &STNodeBase{
			kind: common.PARENTHESIZED_ARG_LIST,
		},
		OpenParenToken:  openParenToken,
		Arguments:       arguments,
		CloseParenToken: closeParenToken,
	}, openParenToken, arguments, closeParenToken)
}

func CreateMethodCallExpressionNode(expression STNode, dotToken STNode, methodName STNode, openParenToken STNode, arguments STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STMethodCallExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.METHOD_CALL,
		},
		Expression:      expression,
		DotToken:        dotToken,
		MethodName:      methodName,
		OpenParenToken:  openParenToken,
		Arguments:       arguments,
		CloseParenToken: closeParenToken,
	}, expression, dotToken, methodName, openParenToken, arguments, closeParenToken)
}

func CreateBasicLiteralNode(kind common.SyntaxKind, literalToken STNode) STNode {
	return createNodeAndAddChildren(&STBasicLiteralNode{
		STExpressionNode: &STNodeBase{
			kind: kind,
		},
		LiteralToken: literalToken,
	}, literalToken)
}

func CreateErrorConstructorExpressionNode(errorKeyword STNode, typeReference STNode, openParenToken STNode, arguments STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STErrorConstructorExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.ERROR_CONSTRUCTOR,
		},
		ErrorKeyword:    errorKeyword,
		TypeReference:   typeReference,
		OpenParenToken:  openParenToken,
		Arguments:       arguments,
		CloseParenToken: closeParenToken,
	}, errorKeyword, typeReference, openParenToken, arguments, closeParenToken)
}

func CreateXMLStepExpressionNode(expression STNode, xmlStepStart STNode, xmlStepExtend STNode) STNode {
	return createNodeAndAddChildren(&STXMLStepExpressionNode{
		STXMLNavigateExpressionNode: &STNodeBase{
			kind: common.XML_STEP_EXPRESSION,
		},
		Expression:    expression,
		XmlStepStart:  xmlStepStart,
		XmlStepExtend: xmlStepExtend,
	}, expression, xmlStepStart, xmlStepExtend)
}

func CreateBracedExpressionNode(kind common.SyntaxKind, openParen STNode, expression STNode, closeParen STNode) STNode {
	return createNodeAndAddChildren(&STBracedExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: kind,
		},
		OpenParen:  openParen,
		Expression: expression,
		CloseParen: closeParen,
	}, openParen, expression, closeParen)
}

func CreateNilLiteralNode(openParenToken STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STNilLiteralNode{
		STExpressionNode: &STNodeBase{
			kind: common.NIL_LITERAL,
		},
		OpenParenToken:  openParenToken,
		CloseParenToken: closeParenToken,
	}, openParenToken, closeParenToken)
}

func CreateResourcePathParameterNode(kind common.SyntaxKind, openBracketToken STNode, annotations STNode, typeDescriptor STNode, ellipsisToken STNode, paramName STNode, closeBracketToken STNode) STNode {
	return createNodeAndAddChildren(&STResourcePathParameterNode{
		STNode: &STNodeBase{
			kind: kind,
		},
		OpenBracketToken:  openBracketToken,
		Annotations:       annotations,
		TypeDescriptor:    typeDescriptor,
		EllipsisToken:     ellipsisToken,
		ParamName:         paramName,
		CloseBracketToken: closeBracketToken,
	}, openBracketToken, annotations, typeDescriptor, ellipsisToken, paramName, closeBracketToken)
}

func CreateObjectTypeDescriptorNode(objectTypeQualifiers STNode, objectKeyword STNode, openBrace STNode, members STNode, closeBrace STNode) STNode {
	return createNodeAndAddChildren(&STObjectTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.OBJECT_TYPE_DESC,
		},
		ObjectTypeQualifiers: objectTypeQualifiers,
		ObjectKeyword:        objectKeyword,
		OpenBrace:            openBrace,
		Members:              members,
		CloseBrace:           closeBrace,
	}, objectTypeQualifiers, objectKeyword, openBrace, members, closeBrace)
}

func CreatePositionalArgumentNode(expression STNode) STNode {
	return createNodeAndAddChildren(&STPositionalArgumentNode{
		STFunctionArgumentNode: &STNodeBase{
			kind: common.POSITIONAL_ARG,
		},
		Expression: expression,
	}, expression)
}

func CreateRestArgumentNode(ellipsis STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STRestArgumentNode{
		STFunctionArgumentNode: &STNodeBase{
			kind: common.REST_ARG,
		},
		Ellipsis:   ellipsis,
		Expression: expression,
	}, ellipsis, expression)
}

func CreateNamedArgumentNode(argumentName STNode, equalsToken STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STNamedArgumentNode{
		STFunctionArgumentNode: &STNodeBase{
			kind: common.NAMED_ARG,
		},
		ArgumentName: argumentName,
		EqualsToken:  equalsToken,
		Expression:   expression,
	}, argumentName, equalsToken, expression)
}

func CreateIfElseStatementNode(ifKeyword STNode, condition STNode, ifBody STNode, elseBody STNode) STNode {
	return createNodeAndAddChildren(&STIfElseStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.IF_ELSE_STATEMENT,
		},
		IfKeyword: ifKeyword,
		Condition: condition,
		IfBody:    ifBody,
		ElseBody:  elseBody,
	}, ifKeyword, condition, ifBody, elseBody)
}

func CreateTypeCastExpressionNode(ltToken STNode, typeCastParam STNode, gtToken STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STTypeCastExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.TYPE_CAST_EXPRESSION,
		},
		LtToken:       ltToken,
		TypeCastParam: typeCastParam,
		GtToken:       gtToken,
		Expression:    expression,
	}, ltToken, typeCastParam, gtToken, expression)
}

func CreateSpreadMemberNode(ellipsis STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STSpreadMemberNode{
		STNode: &STNodeBase{
			kind: common.SPREAD_MEMBER,
		},
		Ellipsis:   ellipsis,
		Expression: expression,
	}, ellipsis, expression)
}

func CreateForEachStatementNode(forEachKeyword STNode, typedBindingPattern STNode, inKeyword STNode, actionOrExpressionNode STNode, blockStatement STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STForEachStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.FOREACH_STATEMENT,
		},
		ForEachKeyword:         forEachKeyword,
		TypedBindingPattern:    typedBindingPattern,
		InKeyword:              inKeyword,
		ActionOrExpressionNode: actionOrExpressionNode,
		BlockStatement:         blockStatement,
		OnFailClause:           onFailClause,
	}, forEachKeyword, typedBindingPattern, inKeyword, actionOrExpressionNode, blockStatement, onFailClause)
}

func CreateTrapExpressionNode(kind common.SyntaxKind, trapKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STTrapExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: kind,
		},
		TrapKeyword: trapKeyword,
		Expression:  expression,
	}, trapKeyword, expression)
}

func CreateForkStatementNode(forkKeyword STNode, openBraceToken STNode, namedWorkerDeclarations STNode, closeBraceToken STNode) STNode {
	return createNodeAndAddChildren(&STForkStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.FORK_STATEMENT,
		},
		ForkKeyword:             forkKeyword,
		OpenBraceToken:          openBraceToken,
		NamedWorkerDeclarations: namedWorkerDeclarations,
		CloseBraceToken:         closeBraceToken,
	}, forkKeyword, openBraceToken, namedWorkerDeclarations, closeBraceToken)
}

func CreateUnionTypeDescriptorNode(leftTypeDesc STNode, pipeToken STNode, rightTypeDesc STNode) STNode {
	return createNodeAndAddChildren(&STUnionTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.UNION_TYPE_DESC,
		},
		LeftTypeDesc:  leftTypeDesc,
		PipeToken:     pipeToken,
		RightTypeDesc: rightTypeDesc,
	}, leftTypeDesc, pipeToken, rightTypeDesc)
}

func CreateLockStatementNode(lockKeyword STNode, blockStatement STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STLockStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.LOCK_STATEMENT,
		},
		LockKeyword:    lockKeyword,
		BlockStatement: blockStatement,
		OnFailClause:   onFailClause,
	}, lockKeyword, blockStatement, onFailClause)
}

func CreateNamedWorkerDeclarationNode(annotations STNode, transactionalKeyword STNode, workerKeyword STNode, workerName STNode, returnTypeDesc STNode, workerBody STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STNamedWorkerDeclarationNode{
		STNode: &STNodeBase{
			kind: common.NAMED_WORKER_DECLARATION,
		},
		Annotations:          annotations,
		TransactionalKeyword: transactionalKeyword,
		WorkerKeyword:        workerKeyword,
		WorkerName:           workerName,
		ReturnTypeDesc:       returnTypeDesc,
		WorkerBody:           workerBody,
		OnFailClause:         onFailClause,
	}, annotations, transactionalKeyword, workerKeyword, workerName, returnTypeDesc, workerBody, onFailClause)
}

func CreateModuleXMLNamespaceDeclarationNode(xmlnsKeyword STNode, namespaceuri STNode, asKeyword STNode, namespacePrefix STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STModuleXMLNamespaceDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.MODULE_XML_NAMESPACE_DECLARATION,
		},
		XmlnsKeyword:    xmlnsKeyword,
		Namespaceuri:    namespaceuri,
		AsKeyword:       asKeyword,
		NamespacePrefix: namespacePrefix,
		SemicolonToken:  semicolonToken,
	}, xmlnsKeyword, namespaceuri, asKeyword, namespacePrefix, semicolonToken)
}

func CreateXMLNamespaceDeclarationNode(xmlnsKeyword STNode, namespaceuri STNode, asKeyword STNode, namespacePrefix STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STXMLNamespaceDeclarationNode{
		STStatementNode: &STNodeBase{
			kind: common.XML_NAMESPACE_DECLARATION,
		},
		XmlnsKeyword:    xmlnsKeyword,
		Namespaceuri:    namespaceuri,
		AsKeyword:       asKeyword,
		NamespacePrefix: namespacePrefix,
		SemicolonToken:  semicolonToken,
	}, xmlnsKeyword, namespaceuri, asKeyword, namespacePrefix, semicolonToken)
}

func CreateAnnotationAttachPointNode(sourceKeyword STNode, identifiers STNode) STNode {
	return createNodeAndAddChildren(&STAnnotationAttachPointNode{
		STNode: &STNodeBase{
			kind: common.ANNOTATION_ATTACH_POINT,
		},
		SourceKeyword: sourceKeyword,
		Identifiers:   identifiers,
	}, sourceKeyword, identifiers)
}

func CreateAnnotationDeclarationNode(metadata STNode, visibilityQualifier STNode, constKeyword STNode, annotationKeyword STNode, typeDescriptor STNode, annotationTag STNode, onKeyword STNode, attachPoints STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STAnnotationDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.ANNOTATION_DECLARATION,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		ConstKeyword:        constKeyword,
		AnnotationKeyword:   annotationKeyword,
		TypeDescriptor:      typeDescriptor,
		AnnotationTag:       annotationTag,
		OnKeyword:           onKeyword,
		AttachPoints:        attachPoints,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, constKeyword, annotationKeyword, typeDescriptor, annotationTag, onKeyword, attachPoints, semicolonToken)
}

func CreateParameterizedTypeDescriptorNode(kind common.SyntaxKind, keywordToken STNode, typeParamNode STNode) STNode {
	return createNodeAndAddChildren(&STParameterizedTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: kind,
		},
		KeywordToken:  keywordToken,
		TypeParamNode: typeParamNode,
	}, keywordToken, typeParamNode)
}

func CreateMapTypeDescriptorNode(mapKeywordToken STNode, mapTypeParamsNode STNode) STNode {
	return createNodeAndAddChildren(&STMapTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.MAP_TYPE_DESC,
		},
		MapKeywordToken:   mapKeywordToken,
		MapTypeParamsNode: mapTypeParamsNode,
	}, mapKeywordToken, mapTypeParamsNode)
}

func CreateClientResourceAccessActionNode(expression STNode, rightArrowToken STNode, slashToken STNode, resourceAccessPath STNode, dotToken STNode, methodName STNode, arguments STNode) STNode {
	return createNodeAndAddChildren(&STClientResourceAccessActionNode{
		STActionNode: &STNodeBase{
			kind: common.CLIENT_RESOURCE_ACCESS_ACTION,
		},
		Expression:         expression,
		RightArrowToken:    rightArrowToken,
		SlashToken:         slashToken,
		ResourceAccessPath: resourceAccessPath,
		DotToken:           dotToken,
		MethodName:         methodName,
		Arguments:          arguments,
	}, expression, rightArrowToken, slashToken, resourceAccessPath, dotToken, methodName, arguments)
}

func CreateResourceAccessRestSegmentNode(openBracketToken STNode, ellipsisToken STNode, expression STNode, closeBracketToken STNode) STNode {
	return createNodeAndAddChildren(&STResourceAccessRestSegmentNode{
		STNode: &STNodeBase{
			kind: common.RESOURCE_ACCESS_REST_SEGMENT,
		},
		OpenBracketToken:  openBracketToken,
		EllipsisToken:     ellipsisToken,
		Expression:        expression,
		CloseBracketToken: closeBracketToken,
	}, openBracketToken, ellipsisToken, expression, closeBracketToken)
}

func CreateComputedResourceAccessSegmentNode(openBracketToken STNode, expression STNode, closeBracketToken STNode) STNode {
	return createNodeAndAddChildren(&STComputedResourceAccessSegmentNode{
		STNode: &STNodeBase{
			kind: common.COMPUTED_RESOURCE_ACCESS_SEGMENT,
		},
		OpenBracketToken:  openBracketToken,
		Expression:        expression,
		CloseBracketToken: closeBracketToken,
	}, openBracketToken, expression, closeBracketToken)
}

func CreateExpressionStatementNode(kind common.SyntaxKind, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STExpressionStatementNode{
		STStatementNode: &STNodeBase{
			kind: kind,
		},
		Expression:     expression,
		SemicolonToken: semicolonToken,
	}, expression, semicolonToken)
}

func CreateLocalTypeDefinitionStatementNode(annotations STNode, typeKeyword STNode, typeName STNode, typeDescriptor STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STLocalTypeDefinitionStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.LOCAL_TYPE_DEFINITION_STATEMENT,
		},
		Annotations:    annotations,
		TypeKeyword:    typeKeyword,
		TypeName:       typeName,
		TypeDescriptor: typeDescriptor,
		SemicolonToken: semicolonToken,
	}, annotations, typeKeyword, typeName, typeDescriptor, semicolonToken)
}

func CreateAnnotationNode(atToken STNode, annotReference STNode, annotValue STNode) STNode {
	return createNodeAndAddChildren(&STAnnotationNode{
		STNode: &STNodeBase{
			kind: common.ANNOTATION,
		},
		AtToken:        atToken,
		AnnotReference: annotReference,
		AnnotValue:     annotValue,
	}, atToken, annotReference, annotValue)
}

func CreateArrayDimensionNode(openBracket STNode, arrayLength STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STArrayDimensionNode{
		STNode: &STNodeBase{
			kind: common.ARRAY_DIMENSION,
		},
		OpenBracket:  openBracket,
		ArrayLength:  arrayLength,
		CloseBracket: closeBracket,
	}, openBracket, arrayLength, closeBracket)
}

func CreateUnaryExpressionNode(unaryOperator STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STUnaryExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.UNARY_EXPRESSION,
		},
		UnaryOperator: unaryOperator,
		Expression:    expression,
	}, unaryOperator, expression)
}

func CreateTypeofExpressionNode(typeofKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STTypeofExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.TYPEOF_EXPRESSION,
		},
		TypeofKeyword: typeofKeyword,
		Expression:    expression,
	}, typeofKeyword, expression)
}

func CreateConstantDeclarationNode(metadata STNode, visibilityQualifier STNode, constKeyword STNode, typeDescriptor STNode, variableName STNode, equalsToken STNode, initializer STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STConstantDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.CONST_DECLARATION,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		ConstKeyword:        constKeyword,
		TypeDescriptor:      typeDescriptor,
		VariableName:        variableName,
		EqualsToken:         equalsToken,
		Initializer:         initializer,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, constKeyword, typeDescriptor, variableName, equalsToken, initializer, semicolonToken)
}

func CreateListenerDeclarationNode(metadata STNode, visibilityQualifier STNode, listenerKeyword STNode, typeDescriptor STNode, variableName STNode, equalsToken STNode, initializer STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STListenerDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.LISTENER_DECLARATION,
		},
		Metadata:            metadata,
		VisibilityQualifier: visibilityQualifier,
		ListenerKeyword:     listenerKeyword,
		TypeDescriptor:      typeDescriptor,
		VariableName:        variableName,
		EqualsToken:         equalsToken,
		Initializer:         initializer,
		SemicolonToken:      semicolonToken,
	}, metadata, visibilityQualifier, listenerKeyword, typeDescriptor, variableName, equalsToken, initializer, semicolonToken)
}

func CreateServiceDeclarationNode(metadata STNode, qualifiers STNode, serviceKeyword STNode, typeDescriptor STNode, absoluteResourcePath STNode, onKeyword STNode, expressions STNode, openBraceToken STNode, members STNode, closeBraceToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STServiceDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.SERVICE_DECLARATION,
		},
		Metadata:             metadata,
		Qualifiers:           qualifiers,
		ServiceKeyword:       serviceKeyword,
		TypeDescriptor:       typeDescriptor,
		AbsoluteResourcePath: absoluteResourcePath,
		OnKeyword:            onKeyword,
		Expressions:          expressions,
		OpenBraceToken:       openBraceToken,
		Members:              members,
		CloseBraceToken:      closeBraceToken,
		SemicolonToken:       semicolonToken,
	}, metadata, qualifiers, serviceKeyword, typeDescriptor, absoluteResourcePath, onKeyword, expressions, openBraceToken, members, closeBraceToken, semicolonToken)
}

func CreateCompoundAssignmentStatementNode(lhsExpression STNode, binaryOperator STNode, equalsToken STNode, rhsExpression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STCompoundAssignmentStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.COMPOUND_ASSIGNMENT_STATEMENT,
		},
		LhsExpression:  lhsExpression,
		BinaryOperator: binaryOperator,
		EqualsToken:    equalsToken,
		RhsExpression:  rhsExpression,
		SemicolonToken: semicolonToken,
	}, lhsExpression, binaryOperator, equalsToken, rhsExpression, semicolonToken)
}

func CreateComputedNameFieldNode(openBracket STNode, fieldNameExpr STNode, closeBracket STNode, colonToken STNode, valueExpr STNode) STNode {
	return createNodeAndAddChildren(&STComputedNameFieldNode{
		STMappingFieldNode: &STNodeBase{
			kind: common.COMPUTED_NAME_FIELD,
		},
		OpenBracket:   openBracket,
		FieldNameExpr: fieldNameExpr,
		CloseBracket:  closeBracket,
		ColonToken:    colonToken,
		ValueExpr:     valueExpr,
	}, openBracket, fieldNameExpr, closeBracket, colonToken, valueExpr)
}

func CreateContinueStatementNode(continueToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STContinueStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.CONTINUE_STATEMENT,
		},
		ContinueToken:  continueToken,
		SemicolonToken: semicolonToken,
	}, continueToken, semicolonToken)
}

func CreateFailStatementNode(failKeyword STNode, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STFailStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.FAIL_STATEMENT,
		},
		FailKeyword:    failKeyword,
		Expression:     expression,
		SemicolonToken: semicolonToken,
	}, failKeyword, expression, semicolonToken)
}

func CreateBreakStatementNode(breakToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STBreakStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.BREAK_STATEMENT,
		},
		BreakToken:     breakToken,
		SemicolonToken: semicolonToken,
	}, breakToken, semicolonToken)
}

func CreateCheckExpressionNode(kind common.SyntaxKind, checkKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STCheckExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: kind,
		},
		CheckKeyword: checkKeyword,
		Expression:   expression,
	}, checkKeyword, expression)
}

func CreateBlockStatementNode(openBraceToken STNode, statements STNode, closeBraceToken STNode) STNode {
	return createNodeAndAddChildren(&STBlockStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.BLOCK_STATEMENT,
		},
		OpenBraceToken:  openBraceToken,
		Statements:      statements,
		CloseBraceToken: closeBraceToken,
	}, openBraceToken, statements, closeBraceToken)
}

func CreateElseBlockNode(elseKeyword STNode, elseBody STNode) STNode {
	return createNodeAndAddChildren(&STElseBlockNode{
		STNode: &STNodeBase{
			kind: common.ELSE_BLOCK,
		},
		ElseKeyword: elseKeyword,
		ElseBody:    elseBody,
	}, elseKeyword, elseBody)
}

func CreateDoStatementNode(doKeyword STNode, blockStatement STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STDoStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.DO_STATEMENT,
		},
		DoKeyword:      doKeyword,
		BlockStatement: blockStatement,
		OnFailClause:   onFailClause,
	}, doKeyword, blockStatement, onFailClause)
}

func CreateWhileStatementNode(whileKeyword STNode, condition STNode, whileBody STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STWhileStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.WHILE_STATEMENT,
		},
		WhileKeyword: whileKeyword,
		Condition:    condition,
		WhileBody:    whileBody,
		OnFailClause: onFailClause,
	}, whileKeyword, condition, whileBody, onFailClause)
}

func CreatePanicStatementNode(panicKeyword STNode, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STPanicStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.PANIC_STATEMENT,
		},
		PanicKeyword:   panicKeyword,
		Expression:     expression,
		SemicolonToken: semicolonToken,
	}, panicKeyword, expression, semicolonToken)
}

func CreateReturnStatementNode(returnKeyword STNode, expression STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STReturnStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.RETURN_STATEMENT,
		},
		ReturnKeyword:  returnKeyword,
		Expression:     expression,
		SemicolonToken: semicolonToken,
	}, returnKeyword, expression, semicolonToken)
}

func CreateMappingConstructorExpressionNode(openBrace STNode, fields STNode, closeBrace STNode) STNode {
	return createNodeAndAddChildren(&STMappingConstructorExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.MAPPING_CONSTRUCTOR,
		},
		OpenBrace:  openBrace,
		Fields:     fields,
		CloseBrace: closeBrace,
	}, openBrace, fields, closeBrace)
}

func CreateTypeCastParamNode(annotations STNode, typeNode STNode) STNode {
	return createNodeAndAddChildren(&STTypeCastParamNode{
		STNode: &STNodeBase{
			kind: common.TYPE_CAST_PARAM,
		},
		Annotations: annotations,
		Type:        typeNode,
	}, annotations, typeNode)
}

func CreateTableConstructorExpressionNode(tableKeyword STNode, keySpecifier STNode, openBracket STNode, rows STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STTableConstructorExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.TABLE_CONSTRUCTOR,
		},
		TableKeyword: tableKeyword,
		KeySpecifier: keySpecifier,
		OpenBracket:  openBracket,
		Rows:         rows,
		CloseBracket: closeBracket,
	}, tableKeyword, keySpecifier, openBracket, rows, closeBracket)
}

func CreateKeySpecifierNode(keyKeyword STNode, openParenToken STNode, fieldNames STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STKeySpecifierNode{
		STNode: &STNodeBase{
			kind: common.KEY_SPECIFIER,
		},
		KeyKeyword:      keyKeyword,
		OpenParenToken:  openParenToken,
		FieldNames:      fieldNames,
		CloseParenToken: closeParenToken,
	}, keyKeyword, openParenToken, fieldNames, closeParenToken)
}

func CreateStreamTypeDescriptorNode(streamKeywordToken STNode, streamTypeParamsNode STNode) STNode {
	return createNodeAndAddChildren(&STStreamTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.STREAM_TYPE_DESC,
		},
		StreamKeywordToken:   streamKeywordToken,
		StreamTypeParamsNode: streamTypeParamsNode,
	}, streamKeywordToken, streamTypeParamsNode)
}

func CreateStreamTypeParamsNode(ltToken STNode, leftTypeDescNode STNode, commaToken STNode, rightTypeDescNode STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STStreamTypeParamsNode{
		STNode: &STNodeBase{
			kind: common.STREAM_TYPE_PARAMS,
		},
		LtToken:           ltToken,
		LeftTypeDescNode:  leftTypeDescNode,
		CommaToken:        commaToken,
		RightTypeDescNode: rightTypeDescNode,
		GtToken:           gtToken,
	}, ltToken, leftTypeDescNode, commaToken, rightTypeDescNode, gtToken)
}

func CreateLetExpressionNode(letKeyword STNode, letVarDeclarations STNode, inKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STLetExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.LET_EXPRESSION,
		},
		LetKeyword:         letKeyword,
		LetVarDeclarations: letVarDeclarations,
		InKeyword:          inKeyword,
		Expression:         expression,
	}, letKeyword, letVarDeclarations, inKeyword, expression)
}

func CreateLetVariableDeclarationNode(annotations STNode, typedBindingPattern STNode, equalsToken STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STLetVariableDeclarationNode{
		STNode: &STNodeBase{
			kind: common.LET_VAR_DECL,
		},
		Annotations:         annotations,
		TypedBindingPattern: typedBindingPattern,
		EqualsToken:         equalsToken,
		Expression:          expression,
	}, annotations, typedBindingPattern, equalsToken, expression)
}

func CreateTemplateExpressionNode(kind common.SyntaxKind, typeNode STNode, startBacktick STNode, content STNode, endBacktick STNode) STNode {
	return createNodeAndAddChildren(&STTemplateExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: kind,
		},
		Type:          typeNode,
		StartBacktick: startBacktick,
		Content:       content,
		EndBacktick:   endBacktick,
	}, typeNode, startBacktick, content, endBacktick)
}

func CreateWaitActionNode(waitKeyword STNode, waitFutureExpr STNode) STNode {
	return createNodeAndAddChildren(&STWaitActionNode{
		STActionNode: &STNodeBase{
			kind: common.WAIT_ACTION,
		},
		WaitKeyword:    waitKeyword,
		WaitFutureExpr: waitFutureExpr,
	}, waitKeyword, waitFutureExpr)
}

func CreateAlternateReceiveNode(workers STNode) STNode {
	return createNodeAndAddChildren(&STAlternateReceiveNode{
		STNode: &STNodeBase{
			kind: common.ALTERNATE_RECEIVE,
		},
		Workers: workers,
	}, workers)
}

func CreateWaitFieldsListNode(openBrace STNode, waitFields STNode, closeBrace STNode) STNode {
	return createNodeAndAddChildren(&STWaitFieldsListNode{
		STNode: &STNodeBase{
			kind: common.WAIT_FIELDS_LIST,
		},
		OpenBrace:  openBrace,
		WaitFields: waitFields,
		CloseBrace: closeBrace,
	}, openBrace, waitFields, closeBrace)
}

func CreateWaitFieldNode(fieldName STNode, colon STNode, waitFutureExpr STNode) STNode {
	return createNodeAndAddChildren(&STWaitFieldNode{
		STNode: &STNodeBase{
			kind: common.WAIT_FIELD,
		},
		FieldName:      fieldName,
		Colon:          colon,
		WaitFutureExpr: waitFutureExpr,
	}, fieldName, colon, waitFutureExpr)
}

func CreateToken(kind common.SyntaxKind, leadingMinutiae STNode, trailingMinutiae STNode) STToken {
	// FIXME: remove this
	return CreateTokenFrom(kind, leadingMinutiae, trailingMinutiae)
}

// ConditionalExprResolver
func GetSimpleNameRefNode(modulePrefixIdentifier STNode) STNode {
	identifier := ToToken(modulePrefixIdentifier)
	var syntaxToken STToken
	text := identifier.Text()
	switch text {
	case "boolean":
		syntaxToken = CreateTokenFrom(common.BOOLEAN_KEYWORD,
			identifier.LeadingMinutiae(), identifier.TrailingMinutiae())
	case "decimal":
		syntaxToken = CreateTokenFrom(common.DECIMAL_KEYWORD,
			identifier.LeadingMinutiae(), identifier.TrailingMinutiae())
	case "float":
		syntaxToken = CreateTokenFrom(common.FLOAT_KEYWORD,
			identifier.LeadingMinutiae(), identifier.TrailingMinutiae())
	case "int":
		syntaxToken = CreateTokenFrom(common.INT_KEYWORD,
			identifier.LeadingMinutiae(), identifier.TrailingMinutiae())
	case "string":
		syntaxToken = CreateTokenFrom(common.STRING_KEYWORD,
			identifier.LeadingMinutiae(), identifier.TrailingMinutiae())
	default:
		return CreateSimpleNameReferenceNode(identifier)
	}
	return CreateBuiltinSimpleNameReferenceNode(syntaxToken.Kind(), syntaxToken)
}

func GetQualifiedNameRefNode(parentNode STNode, leftMost bool) STNode {
	if parentNode == nil {
		return nil
	}
	if parentNode.Kind() == common.LIST {
		listNode, ok := parentNode.(*STNodeList)
		if !ok {
			panic("expected STNodeList")
		}
		if listNode.IsEmpty() {
			return nil
		}
	}
	if parentNode.Kind() == common.QUALIFIED_NAME_REFERENCE {
		qualifiedNameRefNode, ok := parentNode.(*STQualifiedNameReferenceNode)
		if !ok {
			panic("expected STQualifiedNameReferenceNode")
		}
		modulePrefix := qualifiedNameRefNode.ModulePrefix
		if IsValidSimpleNameRef(ToToken(modulePrefix)) {
			return parentNode
		} else {
			return nil
		}
	}
	var firstOrLastChild STNode
	if leftMost {
		firstOrLastChild = parentNode.ChildInBucket(0)
	} else {
		firstOrLastChild = parentNode.ChildInBucket(parentNode.BucketCount() - 1)
	}
	if IsNonTerminalNode(firstOrLastChild) {
		return GetQualifiedNameRefNode(firstOrLastChild, leftMost)
	}
	return nil
}

func IsValidSimpleNameRef(modulePrefixIdentifier STToken) bool {
	switch modulePrefixIdentifier.Text() {
	case "error", "future", "map", "object", "stream", "table", "transaction", "typedesc", "xml":
		return false
	default:
		return true
	}
}

// SyntaxUtils
func IsNonTerminalNode(node STNode) bool {
	return !IsToken(node)
}

func IsToken(node STNode) bool {
	_, ok := node.(STToken)
	return ok
}

// ========== Query Expression Methods (STNodeFactory.java:1467-1606) ==========

// From STNodeFactory.java:1467-1474
func CreateQueryConstructTypeNode(keyword STNode, keySpecifier STNode) STNode {
	return createNodeAndAddChildren(&STQueryConstructTypeNode{
		STNode: &STNodeBase{
			kind: common.QUERY_CONSTRUCT_TYPE,
		},
		Keyword:      keyword,
		KeySpecifier: keySpecifier,
	}, keyword, keySpecifier)
}

// From STNodeFactory.java:1476-1487
func CreateFromClauseNode(fromKeyword STNode, typedBindingPattern STNode, inKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STFromClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.FROM_CLAUSE,
		},
		FromKeyword:         fromKeyword,
		TypedBindingPattern: typedBindingPattern,
		InKeyword:           inKeyword,
		Expression:          expression,
	}, fromKeyword, typedBindingPattern, inKeyword, expression)
}

// From STNodeFactory.java:1498-1505
func CreateLetClauseNode(letKeyword STNode, letVarDeclarations STNode) STNode {
	return createNodeAndAddChildren(&STLetClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.LET_CLAUSE,
		},
		LetKeyword:         letKeyword,
		LetVarDeclarations: letVarDeclarations,
	}, letKeyword, letVarDeclarations)
}

// From STNodeFactory.java:1507-1522
func CreateJoinClauseNode(outerKeyword STNode, joinKeyword STNode, typedBindingPattern STNode, inKeyword STNode, expression STNode, joinOnCondition STNode) STNode {
	return createNodeAndAddChildren(&STJoinClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.JOIN_CLAUSE,
		},
		OuterKeyword:        outerKeyword,
		JoinKeyword:         joinKeyword,
		TypedBindingPattern: typedBindingPattern,
		InKeyword:           inKeyword,
		Expression:          expression,
		JoinOnCondition:     joinOnCondition,
	}, outerKeyword, joinKeyword, typedBindingPattern, inKeyword, expression, joinOnCondition)
}

// From STNodeFactory.java:1524-1535
func CreateOnClauseNode(onKeyword STNode, onExpression STNode, equalsKeyword STNode, equalsExpression STNode) STNode {
	return createNodeAndAddChildren(&STOnClauseNode{
		STClauseNode: &STNodeBase{
			kind: common.ON_CLAUSE,
		},
		OnKeyword:        onKeyword,
		OnExpression:     onExpression,
		EqualsKeyword:    equalsKeyword,
		EqualsExpression: equalsExpression,
	}, onKeyword, onExpression, equalsKeyword, equalsExpression)
}

// From STNodeFactory.java:1537-1544
func CreateLimitClauseNode(limitKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STLimitClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.LIMIT_CLAUSE,
		},
		LimitKeyword: limitKeyword,
		Expression:   expression,
	}, limitKeyword, expression)
}

// From STNodeFactory.java:1546-1555
func CreateOnConflictClauseNode(onKeyword STNode, conflictKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STOnConflictClauseNode{
		STClauseNode: &STNodeBase{
			kind: common.ON_CONFLICT_CLAUSE,
		},
		OnKeyword:       onKeyword,
		ConflictKeyword: conflictKeyword,
		Expression:      expression,
	}, onKeyword, conflictKeyword, expression)
}

// From STNodeFactory.java:1557-1564
func CreateQueryPipelineNode(fromClause STNode, intermediateClauses STNode) STNode {
	return createNodeAndAddChildren(&STQueryPipelineNode{
		STNode: &STNodeBase{
			kind: common.QUERY_PIPELINE,
		},
		FromClause:          fromClause,
		IntermediateClauses: intermediateClauses,
	}, fromClause, intermediateClauses)
}

// From STNodeFactory.java:1566-1573
func CreateSelectClauseNode(selectKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STSelectClauseNode{
		STClauseNode: &STNodeBase{
			kind: common.SELECT_CLAUSE,
		},
		SelectKeyword: selectKeyword,
		Expression:    expression,
	}, selectKeyword, expression)
}

// From STNodeFactory.java:1575-1582
func CreateCollectClauseNode(collectKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STCollectClauseNode{
		STClauseNode: &STNodeBase{
			kind: common.COLLECT_CLAUSE,
		},
		CollectKeyword: collectKeyword,
		Expression:     expression,
	}, collectKeyword, expression)
}

// From STNodeFactory.java:1584-1595
func CreateQueryExpressionNode(queryConstructType STNode, queryPipeline STNode, resultClause STNode, onConflictClause STNode) STNode {
	return createNodeAndAddChildren(&STQueryExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.QUERY_EXPRESSION,
		},
		QueryConstructType: queryConstructType,
		QueryPipeline:      queryPipeline,
		ResultClause:       resultClause,
		OnConflictClause:   onConflictClause,
	}, queryConstructType, queryPipeline, resultClause, onConflictClause)
}

// From STNodeFactory.java:1597-1606
func CreateQueryActionNode(queryPipeline STNode, doKeyword STNode, blockStatement STNode) STNode {
	return createNodeAndAddChildren(&STQueryActionNode{
		STActionNode: &STNodeBase{
			kind: common.QUERY_ACTION,
		},
		QueryPipeline:  queryPipeline,
		DoKeyword:      doKeyword,
		BlockStatement: blockStatement,
	}, queryPipeline, doKeyword, blockStatement)
}

// From STNodeFactory.java:2325-2334
func CreateOrderByClauseNode(orderKeyword STNode, byKeyword STNode, orderKey STNode) STNode {
	return createNodeAndAddChildren(&STOrderByClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.ORDER_BY_CLAUSE,
		},
		OrderKeyword: orderKeyword,
		ByKeyword:    byKeyword,
		OrderKey:     orderKey,
	}, orderKeyword, byKeyword, orderKey)
}

// From STNodeFactory.java:2336-2343
func CreateOrderKeyNode(expression STNode, orderDirection STNode) STNode {
	return createNodeAndAddChildren(&STOrderKeyNode{
		STNode: &STNodeBase{
			kind: common.ORDER_KEY,
		},
		Expression:     expression,
		OrderDirection: orderDirection,
	}, expression, orderDirection)
}

// From STNodeFactory.java:2345-2354
func CreateGroupByClauseNode(groupKeyword STNode, byKeyword STNode, groupingKey STNode) STNode {
	return createNodeAndAddChildren(&STGroupByClauseNode{
		STIntermediateClauseNode: &STNodeBase{
			kind: common.GROUP_BY_CLAUSE,
		},
		GroupKeyword: groupKeyword,
		ByKeyword:    byKeyword,
		GroupingKey:  groupingKey,
	}, groupKeyword, byKeyword, groupingKey)
}

// From STNodeFactory.java:2356-2367
func CreateGroupingKeyVarDeclarationNode(typeDescriptor STNode, simpleBindingPattern STNode, equalsToken STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STGroupingKeyVarDeclarationNode{
		STNode: &STNodeBase{
			kind: common.GROUPING_KEY_VAR_DECLARATION,
		},
		TypeDescriptor:       typeDescriptor,
		SimpleBindingPattern: simpleBindingPattern,
		EqualsToken:          equalsToken,
		Expression:           expression,
	}, typeDescriptor, simpleBindingPattern, equalsToken, expression)
}

// ========== Match Statement & Pattern Methods (STNodeFactory.java:2122-2238) ==========

// From STNodeFactory.java:2122-2137
func CreateMatchStatementNode(matchKeyword STNode, condition STNode, openBrace STNode, matchClauses STNode, closeBrace STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STMatchStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.MATCH_STATEMENT,
		},
		MatchKeyword: matchKeyword,
		Condition:    condition,
		OpenBrace:    openBrace,
		MatchClauses: matchClauses,
		CloseBrace:   closeBrace,
		OnFailClause: onFailClause,
	}, matchKeyword, condition, openBrace, matchClauses, closeBrace, onFailClause)
}

// From STNodeFactory.java:2139-2150
func CreateMatchClauseNode(matchPatterns STNode, matchGuard STNode, rightDoubleArrow STNode, blockStatement STNode) STNode {
	return createNodeAndAddChildren(&STMatchClauseNode{
		STNode: &STNodeBase{
			kind: common.MATCH_CLAUSE,
		},
		MatchPatterns:    matchPatterns,
		MatchGuard:       matchGuard,
		RightDoubleArrow: rightDoubleArrow,
		BlockStatement:   blockStatement,
	}, matchPatterns, matchGuard, rightDoubleArrow, blockStatement)
}

// From STNodeFactory.java:2152-2159
func CreateMatchGuardNode(ifKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STMatchGuardNode{
		STNode: &STNodeBase{
			kind: common.MATCH_GUARD,
		},
		IfKeyword:  ifKeyword,
		Expression: expression,
	}, ifKeyword, expression)
}

// From STNodeFactory.java:2170-2179
func CreateListMatchPatternNode(openBracket STNode, matchPatterns STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STListMatchPatternNode{
		STNode: &STNodeBase{
			kind: common.LIST_MATCH_PATTERN,
		},
		OpenBracket:   openBracket,
		MatchPatterns: matchPatterns,
		CloseBracket:  closeBracket,
	}, openBracket, matchPatterns, closeBracket)
}

// From STNodeFactory.java:2181-2190
func CreateRestMatchPatternNode(ellipsisToken STNode, varKeywordToken STNode, variableName STNode) STNode {
	return createNodeAndAddChildren(&STRestMatchPatternNode{
		STNode: &STNodeBase{
			kind: common.REST_MATCH_PATTERN,
		},
		EllipsisToken:   ellipsisToken,
		VarKeywordToken: varKeywordToken,
		VariableName:    variableName,
	}, ellipsisToken, varKeywordToken, variableName)
}

// From STNodeFactory.java:2192-2201
func CreateMappingMatchPatternNode(openBraceToken STNode, fieldMatchPatterns STNode, closeBraceToken STNode) STNode {
	return createNodeAndAddChildren(&STMappingMatchPatternNode{
		STNode: &STNodeBase{
			kind: common.MAPPING_MATCH_PATTERN,
		},
		OpenBraceToken:     openBraceToken,
		FieldMatchPatterns: fieldMatchPatterns,
		CloseBraceToken:    closeBraceToken,
	}, openBraceToken, fieldMatchPatterns, closeBraceToken)
}

// From STNodeFactory.java:2203-2212
func CreateFieldMatchPatternNode(fieldNameNode STNode, colonToken STNode, matchPattern STNode) STNode {
	return createNodeAndAddChildren(&STFieldMatchPatternNode{
		STNode: &STNodeBase{
			kind: common.FIELD_MATCH_PATTERN,
		},
		FieldNameNode: fieldNameNode,
		ColonToken:    colonToken,
		MatchPattern:  matchPattern,
	}, fieldNameNode, colonToken, matchPattern)
}

// From STNodeFactory.java:2214-2227
func CreateErrorMatchPatternNode(errorKeyword STNode, typeReference STNode, openParenthesisToken STNode, argListMatchPatternNode STNode, closeParenthesisToken STNode) STNode {
	return createNodeAndAddChildren(&STErrorMatchPatternNode{
		STNode: &STNodeBase{
			kind: common.ERROR_MATCH_PATTERN,
		},
		ErrorKeyword:            errorKeyword,
		TypeReference:           typeReference,
		OpenParenthesisToken:    openParenthesisToken,
		ArgListMatchPatternNode: argListMatchPatternNode,
		CloseParenthesisToken:   closeParenthesisToken,
	}, errorKeyword, typeReference, openParenthesisToken, argListMatchPatternNode, closeParenthesisToken)
}

// From STNodeFactory.java:2229-2238
func CreateNamedArgMatchPatternNode(identifier STNode, equalToken STNode, matchPattern STNode) STNode {
	return createNodeAndAddChildren(&STNamedArgMatchPatternNode{
		STNode: &STNodeBase{
			kind: common.NAMED_ARG_MATCH_PATTERN,
		},
		Identifier:   identifier,
		EqualToken:   equalToken,
		MatchPattern: matchPattern,
	}, identifier, equalToken, matchPattern)
}

// ========== Worker Communication Methods (STNodeFactory.java:1641-1827, 1836-1843, 2747-2756) ==========

// From STNodeFactory.java:1641-1650
func CreateStartActionNode(annotations STNode, startKeyword STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STStartActionNode{
		STExpressionNode: &STNodeBase{
			kind: common.START_ACTION,
		},
		Annotations:  annotations,
		StartKeyword: startKeyword,
		Expression:   expression,
	}, annotations, startKeyword, expression)
}

// From STNodeFactory.java:1652-1659
func CreateFlushActionNode(flushKeyword STNode, peerWorker STNode) STNode {
	return createNodeAndAddChildren(&STFlushActionNode{
		STExpressionNode: &STNodeBase{
			kind: common.FLUSH_ACTION,
		},
		FlushKeyword: flushKeyword,
		PeerWorker:   peerWorker,
	}, flushKeyword, peerWorker)
}

// From STNodeFactory.java:1798-1807
func CreateSyncSendActionNode(expression STNode, syncSendToken STNode, peerWorker STNode) STNode {
	return createNodeAndAddChildren(&STSyncSendActionNode{
		STActionNode: &STNodeBase{
			kind: common.SYNC_SEND_ACTION,
		},
		Expression:    expression,
		SyncSendToken: syncSendToken,
		PeerWorker:    peerWorker,
	}, expression, syncSendToken, peerWorker)
}

// From STNodeFactory.java:1809-1816
func CreateReceiveActionNode(leftArrow STNode, receiveWorkers STNode) STNode {
	return createNodeAndAddChildren(&STReceiveActionNode{
		STActionNode: &STNodeBase{
			kind: common.RECEIVE_ACTION,
		},
		LeftArrow:      leftArrow,
		ReceiveWorkers: receiveWorkers,
	}, leftArrow, receiveWorkers)
}

// From STNodeFactory.java:1818-1827
func CreateReceiveFieldsNode(openBrace STNode, receiveFields STNode, closeBrace STNode) STNode {
	return createNodeAndAddChildren(&STReceiveFieldsNode{
		STNode: &STNodeBase{
			kind: common.RECEIVE_FIELDS,
		},
		OpenBrace:     openBrace,
		ReceiveFields: receiveFields,
		CloseBrace:    closeBrace,
	}, openBrace, receiveFields, closeBrace)
}

// From STNodeFactory.java:1836-1843
func CreateRestDescriptorNode(typeDescriptor STNode, ellipsisToken STNode) STNode {
	return createNodeAndAddChildren(&STRestDescriptorNode{
		STNode: &STNodeBase{
			kind: common.REST_TYPE,
		},
		TypeDescriptor: typeDescriptor,
		EllipsisToken:  ellipsisToken,
	}, typeDescriptor, ellipsisToken)
}

// From STNodeFactory.java:2747-2756
func CreateReceiveFieldNode(fieldName STNode, colon STNode, peerWorker STNode) STNode {
	return createNodeAndAddChildren(&STReceiveFieldNode{
		STNode: &STNodeBase{
			kind: common.RECEIVE_FIELD,
		},
		FieldName:  fieldName,
		Colon:      colon,
		PeerWorker: peerWorker,
	}, fieldName, colon, peerWorker)
}

// ========== Transaction Support Methods (STNodeFactory.java:1987-2036, 2369-2380) ==========

// From STNodeFactory.java:1987-1996
func CreateTransactionStatementNode(transactionKeyword STNode, blockStatement STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STTransactionStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.TRANSACTION_STATEMENT,
		},
		TransactionKeyword: transactionKeyword,
		BlockStatement:     blockStatement,
		OnFailClause:       onFailClause,
	}, transactionKeyword, blockStatement, onFailClause)
}

// From STNodeFactory.java:1998-2007
func CreateRollbackStatementNode(rollbackKeyword STNode, expression STNode, semicolon STNode) STNode {
	return createNodeAndAddChildren(&STRollbackStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.ROLLBACK_STATEMENT,
		},
		RollbackKeyword: rollbackKeyword,
		Expression:      expression,
		Semicolon:       semicolon,
	}, rollbackKeyword, expression, semicolon)
}

// From STNodeFactory.java:2009-2022
func CreateRetryStatementNode(retryKeyword STNode, typeParameter STNode, arguments STNode, retryBody STNode, onFailClause STNode) STNode {
	return createNodeAndAddChildren(&STRetryStatementNode{
		STStatementNode: &STNodeBase{
			kind: common.RETRY_STATEMENT,
		},
		RetryKeyword:  retryKeyword,
		TypeParameter: typeParameter,
		Arguments:     arguments,
		RetryBody:     retryBody,
		OnFailClause:  onFailClause,
	}, retryKeyword, typeParameter, arguments, retryBody, onFailClause)
}

// From STNodeFactory.java:2024-2029
func CreateCommitActionNode(commitKeyword STNode) STNode {
	return createNodeAndAddChildren(&STCommitActionNode{
		STActionNode: &STNodeBase{
			kind: common.COMMIT_ACTION,
		},
		CommitKeyword: commitKeyword,
	}, commitKeyword)
}

// From STNodeFactory.java:2031-2036
func CreateTransactionalExpressionNode(transactionalKeyword STNode) STNode {
	return createNodeAndAddChildren(&STTransactionalExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.TRANSACTIONAL_EXPRESSION,
		},
		TransactionalKeyword: transactionalKeyword,
	}, transactionalKeyword)
}

// From STNodeFactory.java:2369-2380
func CreateOnFailClauseNode(onKeyword STNode, failKeyword STNode, typedBindingPattern STNode, blockStatement STNode) STNode {
	return createNodeAndAddChildren(&STOnFailClauseNode{
		STClauseNode: &STNodeBase{
			kind: common.ON_FAIL_CLAUSE,
		},
		OnKeyword:           onKeyword,
		FailKeyword:         failKeyword,
		TypedBindingPattern: typedBindingPattern,
		BlockStatement:      blockStatement,
	}, onKeyword, failKeyword, typedBindingPattern, blockStatement)
}

// ========== Anonymous Function Methods (STNodeFactory.java:1388-1412, 1619-1639) ==========

// From STNodeFactory.java:1619-1628
func CreateImplicitAnonymousFunctionParameters(openParenToken STNode, parameters STNode, closeParenToken STNode) STNode {
	return createNodeAndAddChildren(&STImplicitAnonymousFunctionParameters{
		STNode: &STNodeBase{
			kind: common.INFER_PARAM_LIST,
		},
		OpenParenToken:  openParenToken,
		Parameters:      parameters,
		CloseParenToken: closeParenToken,
	}, openParenToken, parameters, closeParenToken)
}

// From STNodeFactory.java:1630-1639
func CreateImplicitAnonymousFunctionExpressionNode(params STNode, rightDoubleArrow STNode, expression STNode) STNode {
	return createNodeAndAddChildren(&STImplicitAnonymousFunctionExpressionNode{
		STAnonymousFunctionExpressionNode: &STNodeBase{
			kind: common.IMPLICIT_ANONYMOUS_FUNCTION_EXPRESSION,
		},
		Params:           params,
		RightDoubleArrow: rightDoubleArrow,
		Expression:       expression,
	}, params, rightDoubleArrow, expression)
}

// From STNodeFactory.java:1388-1401
func CreateExplicitAnonymousFunctionExpressionNode(annotations STNode, qualifierList STNode, functionKeyword STNode, functionSignature STNode, functionBody STNode) STNode {
	return createNodeAndAddChildren(&STExplicitAnonymousFunctionExpressionNode{
		STAnonymousFunctionExpressionNode: &STNodeBase{
			kind: common.EXPLICIT_ANONYMOUS_FUNCTION_EXPRESSION,
		},
		Annotations:       annotations,
		QualifierList:     qualifierList,
		FunctionKeyword:   functionKeyword,
		FunctionSignature: functionSignature,
		FunctionBody:      functionBody,
	}, annotations, qualifierList, functionKeyword, functionSignature, functionBody)
}

// From STNodeFactory.java:1403-1412
func CreateExpressionFunctionBodyNode(rightDoubleArrow STNode, expression STNode, semicolon STNode) STNode {
	return createNodeAndAddChildren(&STExpressionFunctionBodyNode{
		STFunctionBodyNode: &STNodeBase{
			kind: common.EXPRESSION_FUNCTION_BODY,
		},
		RightDoubleArrow: rightDoubleArrow,
		Expression:       expression,
		Semicolon:        semicolon,
	}, rightDoubleArrow, expression, semicolon)
}

// ========== Type Descriptor Methods (STNodeFactory.java:1333-1362, 1608-1617) ==========

// From STNodeFactory.java:1608-1617
func CreateIntersectionTypeDescriptorNode(leftTypeDesc STNode, bitwiseAndToken STNode, rightTypeDesc STNode) STNode {
	return createNodeAndAddChildren(&STIntersectionTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.INTERSECTION_TYPE_DESC,
		},
		LeftTypeDesc:    leftTypeDesc,
		BitwiseAndToken: bitwiseAndToken,
		RightTypeDesc:   rightTypeDesc,
	}, leftTypeDesc, bitwiseAndToken, rightTypeDesc)
}

// From STNodeFactory.java:1333-1342
func CreateTableTypeDescriptorNode(tableKeywordToken STNode, rowTypeParameterNode STNode, keyConstraintNode STNode) STNode {
	return createNodeAndAddChildren(&STTableTypeDescriptorNode{
		STTypeDescriptorNode: &STNodeBase{
			kind: common.TABLE_TYPE_DESC,
		},
		TableKeywordToken:    tableKeywordToken,
		RowTypeParameterNode: rowTypeParameterNode,
		KeyConstraintNode:    keyConstraintNode,
	}, tableKeywordToken, rowTypeParameterNode, keyConstraintNode)
}

// From STNodeFactory.java:1344-1353
func CreateTypeParameterNode(ltToken STNode, typeNode STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STTypeParameterNode{
		STNode: &STNodeBase{
			kind: common.TYPE_PARAMETER,
		},
		LtToken:  ltToken,
		TypeNode: typeNode,
		GtToken:  gtToken,
	}, ltToken, typeNode, gtToken)
}

// From STNodeFactory.java:1355-1362
func CreateKeyTypeConstraintNode(keyKeywordToken STNode, typeParameterNode STNode) STNode {
	return createNodeAndAddChildren(&STKeyTypeConstraintNode{
		STNode: &STNodeBase{
			kind: common.KEY_TYPE_CONSTRAINT,
		},
		KeyKeywordToken:   keyKeywordToken,
		TypeParameterNode: typeParameterNode,
	}, keyKeywordToken, typeParameterNode)
}

// ========== XML & Template Methods (STNodeFactory.java:2051-2113) ==========

// From STNodeFactory.java:2051-2058
func CreateXMLFilterExpressionNode(expression STNode, xmlPatternChain STNode) STNode {
	return createNodeAndAddChildren(&STXMLFilterExpressionNode{
		STXMLNavigateExpressionNode: &STNodeBase{
			kind: common.XML_FILTER_EXPRESSION,
		},
		Expression:      expression,
		XmlPatternChain: xmlPatternChain,
	}, expression, xmlPatternChain)
}

// From STNodeFactory.java:2071-2080
func CreateXMLNamePatternChainingNode(startToken STNode, xmlNamePattern STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STXMLNamePatternChainingNode{
		STNode: &STNodeBase{
			kind: common.XML_NAME_PATTERN_CHAIN,
		},
		StartToken:     startToken,
		XmlNamePattern: xmlNamePattern,
		GtToken:        gtToken,
	}, startToken, xmlNamePattern, gtToken)
}

// From STNodeFactory.java:2082-2091
func CreateXMLStepIndexedExtendNode(openBracket STNode, expression STNode, closeBracket STNode) STNode {
	return createNodeAndAddChildren(&STXMLStepIndexedExtendNode{
		STNode: &STNodeBase{
			kind: common.XML_STEP_INDEXED_EXTEND,
		},
		OpenBracket:  openBracket,
		Expression:   expression,
		CloseBracket: closeBracket,
	}, openBracket, expression, closeBracket)
}

// From STNodeFactory.java:2093-2102
func CreateXMLStepMethodCallExtendNode(dotToken STNode, methodName STNode, parenthesizedArgList STNode) STNode {
	return createNodeAndAddChildren(&STXMLStepMethodCallExtendNode{
		STNode: &STNodeBase{
			kind: common.XML_STEP_METHOD_CALL_EXTEND,
		},
		DotToken:             dotToken,
		MethodName:           methodName,
		ParenthesizedArgList: parenthesizedArgList,
	}, dotToken, methodName, parenthesizedArgList)
}

// From STNodeFactory.java:2104-2113
func CreateXMLAtomicNamePatternNode(prefix STNode, colon STNode, name STNode) STNode {
	return createNodeAndAddChildren(&STXMLAtomicNamePatternNode{
		STNode: &STNodeBase{
			kind: common.XML_ATOMIC_NAME_PATTERN,
		},
		Prefix: prefix,
		Colon:  colon,
		Name:   name,
	}, prefix, colon, name)
}

// ========== Expression Extensions Methods (STNodeFactory.java:1258-1267, 1896-1916, 2038-2049) ==========

// From STNodeFactory.java:1896-1905
func CreateAnnotAccessExpressionNode(expression STNode, annotChainingToken STNode, annotTagReference STNode) STNode {
	return createNodeAndAddChildren(&STAnnotAccessExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.ANNOT_ACCESS,
		},
		Expression:         expression,
		AnnotChainingToken: annotChainingToken,
		AnnotTagReference:  annotTagReference,
	}, expression, annotChainingToken, annotTagReference)
}

// From STNodeFactory.java:1907-1916
func CreateOptionalFieldAccessExpressionNode(expression STNode, optionalChainingToken STNode, fieldName STNode) STNode {
	return createNodeAndAddChildren(&STOptionalFieldAccessExpressionNode{
		STExpressionNode: &STNodeBase{
			kind: common.OPTIONAL_FIELD_ACCESS,
		},
		Expression:            expression,
		OptionalChainingToken: optionalChainingToken,
		FieldName:             fieldName,
	}, expression, optionalChainingToken, fieldName)
}

// From STNodeFactory.java:2038-2049
func CreateByteArrayLiteralNode(typeNode STNode, startBacktick STNode, content STNode, endBacktick STNode) STNode {
	return createNodeAndAddChildren(&STByteArrayLiteralNode{
		STExpressionNode: &STNodeBase{
			kind: common.BYTE_ARRAY_LITERAL,
		},
		Type:          typeNode,
		StartBacktick: startBacktick,
		Content:       content,
		EndBacktick:   endBacktick,
	}, typeNode, startBacktick, content, endBacktick)
}

// From STNodeFactory.java:1258-1267
func CreateInterpolationNode(interpolationStartToken STNode, expression STNode, interpolationEndToken STNode) STNode {
	return createNodeAndAddChildren(&STInterpolationNode{
		STXMLItemNode: &STNodeBase{
			kind: common.INTERPOLATION,
		},
		InterpolationStartToken: interpolationStartToken,
		Expression:              expression,
		InterpolationEndToken:   interpolationEndToken,
	}, interpolationStartToken, expression, interpolationEndToken)
}

// ========== XML Template Factories (STNodeFactory XML element/attribute/text/comment/PI/CDATA) ==========

func CreateXMLElementNode(startTag STNode, content STNode, endTag STNode) STNode {
	return createNodeAndAddChildren(&STXMLElementNode{
		STXMLItemNode: &STNodeBase{
			kind: common.XML_ELEMENT,
		},
		StartTag: startTag,
		Content:  content,
		EndTag:   endTag,
	}, startTag, content, endTag)
}

func CreateXMLStartTagNode(ltToken STNode, name STNode, attributes STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STXMLStartTagNode{
		STXMLElementTagNode: &STNodeBase{
			kind: common.XML_ELEMENT_START_TAG,
		},
		LtToken:    ltToken,
		Name:       name,
		Attributes: attributes,
		GtToken:    gtToken,
	}, ltToken, name, attributes, gtToken)
}

func CreateXMLEndTagNode(ltToken STNode, slashToken STNode, name STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STXMLEndTagNode{
		STXMLElementTagNode: &STNodeBase{
			kind: common.XML_ELEMENT_END_TAG,
		},
		LtToken:    ltToken,
		SlashToken: slashToken,
		Name:       name,
		GtToken:    gtToken,
	}, ltToken, slashToken, name, gtToken)
}

func CreateXMLEmptyElementNode(ltToken STNode, name STNode, attributes STNode, slashToken STNode, gtToken STNode) STNode {
	return createNodeAndAddChildren(&STXMLEmptyElementNode{
		STXMLItemNode: &STNodeBase{
			kind: common.XML_EMPTY_ELEMENT,
		},
		LtToken:    ltToken,
		Name:       name,
		Attributes: attributes,
		SlashToken: slashToken,
		GtToken:    gtToken,
	}, ltToken, name, attributes, slashToken, gtToken)
}

func CreateXMLSimpleNameNode(name STNode) STNode {
	return createNodeAndAddChildren(&STXMLSimpleNameNode{
		STXMLNameNode: &STNodeBase{
			kind: common.XML_SIMPLE_NAME,
		},
		Name: name,
	}, name)
}

func CreateXMLQualifiedNameNode(prefix STNode, colon STNode, name STNode) STNode {
	return createNodeAndAddChildren(&STXMLQualifiedNameNode{
		STXMLNameNode: &STNodeBase{
			kind: common.XML_QUALIFIED_NAME,
		},
		Prefix: prefix,
		Colon:  colon,
		Name:   name,
	}, prefix, colon, name)
}

func CreateXMLAttributeNode(attributeName STNode, equalToken STNode, value STNode) STNode {
	return createNodeAndAddChildren(&STXMLAttributeNode{
		STNode: &STNodeBase{
			kind: common.XML_ATTRIBUTE,
		},
		AttributeName: attributeName,
		EqualToken:    equalToken,
		Value:         value,
	}, attributeName, equalToken, value)
}

func CreateXMLAttributeValue(startQuote STNode, value STNode, endQuote STNode) STNode {
	return createNodeAndAddChildren(&STXMLAttributeValue{
		STNode: &STNodeBase{
			kind: common.XML_ATTRIBUTE_VALUE,
		},
		StartQuote: startQuote,
		Value:      value,
		EndQuote:   endQuote,
	}, startQuote, value, endQuote)
}

func CreateXMLTextNode(content STNode) STNode {
	return createNodeAndAddChildren(&STXMLTextNode{
		STXMLItemNode: &STNodeBase{
			kind: common.XML_TEXT,
		},
		Content: content,
	}, content)
}

func CreateXMLComment(commentStart STNode, content STNode, commentEnd STNode) STNode {
	return createNodeAndAddChildren(&STXMLComment{
		STXMLItemNode: &STNodeBase{
			kind: common.XML_COMMENT,
		},
		CommentStart: commentStart,
		Content:      content,
		CommentEnd:   commentEnd,
	}, commentStart, content, commentEnd)
}

func CreateXMLProcessingInstruction(piStart STNode, target STNode, data STNode, piEnd STNode) STNode {
	return createNodeAndAddChildren(&STXMLProcessingInstruction{
		STXMLItemNode: &STNodeBase{
			kind: common.XML_PI,
		},
		PiStart: piStart,
		Target:  target,
		Data:    data,
		PiEnd:   piEnd,
	}, piStart, target, data, piEnd)
}

func CreateXMLCDATANode(cdataStart STNode, content STNode, cdataEnd STNode) STNode {
	return createNodeAndAddChildren(&STXMLCDATANode{
		STXMLItemNode: &STNodeBase{
			kind: common.XML_CDATA,
		},
		CdataStart: cdataStart,
		Content:    content,
		CdataEnd:   cdataEnd,
	}, cdataStart, content, cdataEnd)
}

// ========== Binding Pattern Methods (STNodeFactory.java:1705-1710) ==========

// From STNodeFactory.java:1705-1710
func CreateWildcardBindingPatternNode(underscoreToken STNode) STNode {
	return createNodeAndAddChildren(&STWildcardBindingPatternNode{
		STBindingPatternNode: &STNodeBase{
			kind: common.WILDCARD_BINDING_PATTERN,
		},
		UnderscoreToken: underscoreToken,
	}, underscoreToken)
}

// ========== Enum Support Methods (STNodeFactory.java:1933-1965) ==========

// From STNodeFactory.java:1933-1952
func CreateEnumDeclarationNode(metadata STNode, qualifier STNode, enumKeywordToken STNode, identifier STNode, openBraceToken STNode, enumMemberList STNode, closeBraceToken STNode, semicolonToken STNode) STNode {
	return createNodeAndAddChildren(&STEnumDeclarationNode{
		STModuleMemberDeclarationNode: &STNodeBase{
			kind: common.ENUM_DECLARATION,
		},
		Metadata:         metadata,
		Qualifier:        qualifier,
		EnumKeywordToken: enumKeywordToken,
		Identifier:       identifier,
		OpenBraceToken:   openBraceToken,
		EnumMemberList:   enumMemberList,
		CloseBraceToken:  closeBraceToken,
		SemicolonToken:   semicolonToken,
	}, metadata, qualifier, enumKeywordToken, identifier, openBraceToken, enumMemberList, closeBraceToken, semicolonToken)
}

// From STNodeFactory.java:1954-1965
func CreateEnumMemberNode(metadata STNode, identifier STNode, equalToken STNode, constExprNode STNode) STNode {
	return createNodeAndAddChildren(&STEnumMemberNode{
		STNode: &STNodeBase{
			kind: common.ENUM_MEMBER,
		},
		Metadata:      metadata,
		Identifier:    identifier,
		EqualToken:    equalToken,
		ConstExprNode: constExprNode,
	}, metadata, identifier, equalToken, constExprNode)
}

// TODO: think how to special case this so it can also be generated
type STAmbiguousCollectionNode struct {
	STNodeBase

	CollectionStartToken STNode

	Members []STNode

	CollectionEndToken STNode
}

func (n *STAmbiguousCollectionNode) Kind() common.SyntaxKind {
	return n.STNodeBase.Kind()
}

func (n *STAmbiguousCollectionNode) BucketCount() int {
	return 3
}

func (n *STAmbiguousCollectionNode) ChildInBucket(bucket int) STNode {
	switch bucket {

	case 0:
		return n.CollectionStartToken

	case 1:
		return CreateNodeList(n.Members...)
	case 2:
		return n.CollectionEndToken

	default:
		panic("invalid bucket index")
	}
}

func (n *STAmbiguousCollectionNode) ChildBuckets() []STNode {
	return []STNode{
		n.CollectionStartToken,
		CreateNodeList(n.Members...),
		n.CollectionEndToken,
	}
}

var _ STNode = &STAmbiguousCollectionNode{}

func (n *STAmbiguousCollectionNode) CreateFacade(position int, parent NonTerminalNode) Node {
	panic("unsupported operation")
}

func CreateUnlinkedFacade[T STNode, E Node](node T) E {
	return node.CreateFacade(0, nil).(E)
}
