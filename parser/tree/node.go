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

//go:generate ../../tree-gen -config ../nodes.json -type node -template ../../compiler-tools/tree-gen/templates/node.go.tmpl -output node_gen.go

import (
	"iter"

	"ballerina/parser/common"
	"ballerina/tools/diagnostics"
)

// This represent red nodes in the syntax tree. Red nodes satisfy fallowing properties:
//   1. Immutable
//   2. Have parent reference
//   3. Know position but not width
//   4. We build them top down
//       -- They are essentially facade nodes for green nodes.
//       -- We rebuild these per keystroke.
// All red nodes satisfy Node interface.
// We need these to be very fast to build but we rebuild the tree per keystroke so no necessarily memory efficient.
//

// TODO: Revisit how we store position information for nodes. This is storing too much information that can be computed
//  no demand and costing us memory.

type Node interface {
	Position() int
	Parent() NonTerminalNode
	Ancestor(filter func(Node) bool) *Node
	Ancestors() []*Node
	TextRange() TextRange
	TextRangeWithMinutiae() TextRange
	Kind() common.SyntaxKind
	Location() NodeLocation
	Diagnostics() iter.Seq[Diagnostic]
	HasDiagnostics() bool
	IsMissing() bool
	SyntaxTree() *SyntaxTree
	SetSyntaxTree(syntaxTree *SyntaxTree)
	LineRange() LineRange
	LeadingMinutiae() MinutiaeList
	TrailingMinutiae() MinutiaeList
	LeadingInvalidTokens() []Token
	TrailingInvalidTokens() []Token
	// TODO: think about how to do nodetransformer
	InternalNode() STNode
	ToSourceCode() string
}

type MinutiaeList struct{}

func (m *MinutiaeList) Iterator() iter.Seq[Minutiae] {
	panic("not implemented")
}

type Minutiae struct {
	internalMinutiae STMinutiae
	token            Token
	position         int
	textRange        TextRange
	lineRange        LineRange
}

type Location interface {
	LineRange() LineRange
	TextRange() TextRange
}

// TODO: get rid of this unwanted indirection (to get this you had a node get location from that)
type NodeLocation struct {
	node Node
}

func (n *NodeLocation) LineRange() LineRange {
	return n.node.LineRange()
}

func (n *NodeLocation) TextRange() TextRange {
	return n.node.TextRange()
}

type Diagnostic interface {
	Location() Location
	DiagnosticInfo() DiagnosticInfo
	Message() string
	Properties() []DiagnosticProperty[any]
}

type DiagnosticProperty[T any] interface {
	Kind() diagnostics.DiagnosticPropertyKind
	Value() T
}

type SyntaxDiagnostic struct {
	nodeDiagnostic STNodeDiagnostic
	location       NodeLocation
	diagnosticInfo *DiagnosticInfo
}

func (sd *SyntaxDiagnostic) Location() Location {
	return &sd.location
}

func (sd *SyntaxDiagnostic) DiagnosticInfo() DiagnosticInfo {
	if sd.diagnosticInfo != nil {
		return *sd.diagnosticInfo
	}
	diagnosticCode := sd.nodeDiagnostic.DiagnosticCode()
	sd.diagnosticInfo = &DiagnosticInfo{
		code:          diagnosticCode.DiagnosticId(),
		messageFormat: diagnosticCode.MessageKey(), severity: DiagnosticSeverity(diagnosticCode.Severity()),
	}
	return *sd.diagnosticInfo
}

type DiagnosticInfo struct {
	code          string
	messageFormat string
	severity      DiagnosticSeverity
}

type DiagnosticSeverity uint8

const (
	Internal DiagnosticSeverity = iota
	Hint
	Info
	Warning
	Error
)

type NodeBase struct {
	internalNode STNode
	// TODO: does this needs to be int?
	position              int
	parent                NonTerminalNode
	syntaxTree            *SyntaxTree
	lineRange             LineRange
	textRange             TextRange
	textRangeWithMinutiae TextRange
}

func NodeFrom(internalNode STNode, position int, parent NonTerminalNode) Node {
	return &NodeBase{
		internalNode: internalNode,
		position:     position,
		parent:       parent,
	}
}

func (n *NodeBase) Kind() common.SyntaxKind {
	return n.internalNode.Kind()
}

func (n *NodeBase) Position() int {
	return n.position
}

func (n *NodeBase) Parent() NonTerminalNode {
	return n.parent
}

func (n *NodeBase) SetSyntaxTree(syntaxTree *SyntaxTree) {
	n.syntaxTree = syntaxTree
}

func (n *NodeBase) Ancestor(filter func(Node) bool) *Node {
	if n.parent == nil {
		return nil
	}
	parent := n.parent
	for parent != nil {
		if filter(parent) {
			var result Node = parent
			return &result
		}
		parentPtr := parent.Parent()
		if parentPtr == nil {
			break
		}
		parent = parentPtr
	}
	return nil
}

func (n *NodeBase) Ancestors() []*Node {
	var ancestors []*Node
	if n.parent == nil {
		return ancestors
	}
	var parent Node = n.parent
	for parent != nil {
		ancestors = append(ancestors, &parent)
		parent = parent.Parent()
	}
	return ancestors
}

func (n *NodeBase) TextRange() TextRange {
	if n.textRange.Length != 0 {
		return n.textRange
	}
	leadingMinutiaeDelta := int(n.internalNode.WidthWithLeadingMinutiae()) - int(n.internalNode.Width())
	positionWithoutLeadingMinutiae := n.position + leadingMinutiaeDelta
	n.textRange = TextRange{
		StartOffset: positionWithoutLeadingMinutiae,
		EndOffset:   positionWithoutLeadingMinutiae + int(n.internalNode.Width()),
		Length:      int(n.internalNode.Width()),
	}
	return n.textRange
}

func (n *NodeBase) TextRangeWithMinutiae() TextRange {
	if n.textRangeWithMinutiae.Length != 0 {
		return n.textRangeWithMinutiae
	}
	n.textRangeWithMinutiae = TextRange{
		StartOffset: n.position,
		EndOffset:   n.position + int(n.internalNode.WidthWithMinutiae()),
		Length:      int(n.internalNode.WidthWithMinutiae()),
	}
	return n.textRangeWithMinutiae
}

func (n *NodeBase) Location() NodeLocation {
	return NodeLocation{node: n}
}

func (n *NodeBase) Diagnostics() iter.Seq[Diagnostic] {
	panic("Diagnostics() should be implemented by child types")
}

func (n *NonTerminalNodeBase) Diagnostics() iter.Seq[Diagnostic] {
	return func(yield func(Diagnostic) bool) {
		if !n.internalNode.HasDiagnostics() {
			return
		}
		for _, ch := range n.Children() {
			for diagnostic := range ch.Diagnostics() {
				if !yield(diagnostic) {
					return
				}
			}
		}
		for _, diagnostic := range n.internalNode.Diagnostics() {
			if !yield(createSyntaxDiagnostic(diagnostic)) {
				return
			}
		}
	}
}

func createSyntaxDiagnostic(diagnostic STNodeDiagnostic) Diagnostic {
	panic("not implemented")
}

func (n *NonTerminalNodeBase) Children() []Node {
	panic("Children() should be implemented by child types")
}

func (n *NodeBase) HasDiagnostics() bool {
	return n.internalNode.HasDiagnostics()
}

func (n *NodeBase) IsMissing() bool {
	return n.internalNode.IsMissing()
}

func (n *NodeBase) SyntaxTree() *SyntaxTree {
	return n.populateSyntaxTree()
}

func (n *NodeBase) LineRange() LineRange {
	if n.lineRange.StartLine.Line != 0 || n.lineRange.EndLine.Line != 0 {
		return n.lineRange
	}

	textDocument := n.SyntaxTree().TextDocument()
	if textDocument == nil {
		return n.lineRange
	}

	textRange := n.TextRange()
	startOffset := textRange.StartOffset
	endOffset := textRange.EndOffset

	startLine, startColumn, err := textDocument.LinePositionFromTextPosition(startOffset)
	if err != nil {
		return n.lineRange
	}
	endLine, endColumn, err := textDocument.LinePositionFromTextPosition(endOffset)
	if err != nil {
		return n.lineRange
	}

	n.lineRange = LineRange{
		StartLine: LinePosition{Line: startLine, Column: startColumn},
		EndLine:   LinePosition{Line: endLine, Column: endColumn},
	}

	return n.lineRange
}

func (n *NodeBase) LeadingMinutiae() MinutiaeList {
	panic("LeadingMinutiae() should be implemented by child types")
}

func (n *NodeBase) TrailingMinutiae() MinutiaeList {
	panic("TrailingMinutiae() should be implemented by child types")
}

func (n *NodeBase) LeadingInvalidTokens() []Token {
	panic("LeadingInvalidTokens() should be implemented by child types")
}

func (n *NodeBase) TrailingInvalidTokens() []Token {
	panic("TrailingInvalidTokens() should be implemented by child types")
}

func (n *NodeBase) InternalNode() STNode {
	return n.internalNode
}

func (n *NodeBase) ToSourceCode() string {
	panic("ToSourceCode() should be implemented by child types")
}

func (n *NodeBase) populateSyntaxTree() *SyntaxTree {
	if n.syntaxTree != nil {
		return n.syntaxTree
	}

	if n.parent == nil {
		// This is a detached node. Create a new SyntaxTree with this node being the root.
		n.syntaxTree = &SyntaxTree{
			RootNode: n,
		}
	} else {
		parent := n.parent
		n.syntaxTree = parent.SyntaxTree()
	}
	return n.syntaxTree
}

type NonTerminalNode interface {
	Node
	bucketCount() int
	ChildNodes() iter.Seq[Node]
	loadNode(childIndex int) Node
	ChildInBucket(bucket int) Node
}
type NonTerminalNodeBase struct {
	NodeBase
	childBuckets []Node
}

func (n *NonTerminalNodeBase) bucketCount() int {
	return n.internalNode.BucketCount()
}

func (n *NonTerminalNodeBase) ChildNodes() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for i := range n.childBuckets {
			if !yield(n.loadNode(i)) {
				return
			}
		}
	}
}

func (n *NonTerminalNodeBase) loadNode(childIndex int) Node {
	return n.ChildInBucket(childIndex)
}

func into[T Node](node Node) T {
	typed, ok := node.(T)
	if !ok {
		panic("failed to cast node to type")
	}
	return typed
}

func (n *NonTerminalNodeBase) getChildPosition(bucket int) int {
	childPos := n.position
	for i := range bucket {
		childNode := n.internalNode.ChildInBucket(i)
		if IsSTNodePresent(childNode) {
			childPos += int(childNode.WidthWithMinutiae())
		}
	}
	return childPos
}

func (n *NonTerminalNodeBase) ChildInBucket(bucket int) Node {
	child := n.childBuckets[bucket]
	if child != nil {
		return child
	}
	internalChild := n.internalNode.ChildInBucket(bucket)
	if !IsSTNodePresent(internalChild) {
		return nil
	}
	child = createFacade[Node](internalChild, n.getChildPosition(bucket), n)
	n.childBuckets[bucket] = child
	return child
}

type Token interface {
	Node
	Text() string
}

type TokenBase struct {
	NodeBase
	leadingMinutiaeList  MinutiaeList
	trailingMinutiaeList MinutiaeList
}

func (t *TokenBase) Text() string {
	stToken, ok := t.internalNode.(STToken)
	if !ok {
		panic("expected STToken")
	}
	return stToken.Text()
}

type LineRange struct {
	// In java version there is fileNmae as well I think we can get this from textDocument
	StartLine LinePosition
	EndLine   LinePosition
}

// TODO: int to match with java, i think a pair of u16 is enough
type LinePosition struct {
	Line   int
	Column int
}
type TextRange struct {
	StartOffset int
	EndOffset   int
	Length      int
}

func createFacade[T Node](node STNode, position int, parent NonTerminalNode) T {
	return node.CreateFacade(position, parent).(T)
}

type NodeList[T Node] struct {
	internalListNode STNodeList
	nonTerminalNode  NonTerminalNode
	size             int
}

func (n *NodeList[T]) Size() int {
	return n.size
}

func nodeListFrom[T Node](nonTerminalNode NonTerminalNode) NodeList[T] {
	size := nonTerminalNode.bucketCount()
	internalListNode, ok := nonTerminalNode.InternalNode().(*STNodeList)
	if !ok {
		panic("expected STNodeList")
	}
	return NodeList[T]{
		internalListNode: *internalListNode,
		nonTerminalNode:  nonTerminalNode,
		size:             size,
	}
}

func (n *NodeList[T]) Get(index int) T {
	if index < 0 || index >= n.size {
		panic("index out of bounds")
	}
	return n.nonTerminalNode.ChildInBucket(index).(T)
}

func (n *NodeList[T]) tryGet(index int) (*T, bool) {
	if index < 0 || index >= n.size {
		panic("index out of bounds")
	}
	if val, ok := n.nonTerminalNode.ChildInBucket(index).(T); ok {
		return &val, true
	}
	return nil, false
}

func (n *NodeList[T]) Iterator() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range n.size {
			if val, ok := n.tryGet(i); ok {
				if !yield(*val) {
					return
				}
			} else {
				continue
			}
		}
	}
}

type DocumentMemberDeclarationNode struct {
	NonTerminalNodeBase
}

type IdentifierToken struct {
	TokenBase
}

type ExternalTreeNodeList struct {
	NonTerminalNodeBase
}

var _ Node = &ExternalTreeNodeList{}

type LiteralValueToken struct {
	TokenBase
}

var _ Node = &LiteralValueToken{}

func CreateNodeListWithFacade[T Node](nodes []T) NodeList[T] {
	var internalNodes []STNode
	for _, node := range nodes {
		internalNodes = append(internalNodes, node.InternalNode())
	}
	stNodeList := CreateNodeList(internalNodes...).(*STNodeList)
	nodeList := NodeList[T]{
		internalListNode: *stNodeList,
		nonTerminalNode:  nil,
		size:             len(nodes),
	}
	return nodeList
}
