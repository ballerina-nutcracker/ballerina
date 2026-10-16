// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
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

// Package nodebuilder provides APIs to convert syntax tree in to abstract syntax tree
package nodebuilder

import (
	"fmt"
	"iter"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	balCommon "github.com/ballerina-nutcracker/ballerina/common"
)

type rangeFn func(st.Node) st.TextRange

type nodeBuilder struct {
	PackageID            *model.PackageID
	anonTypeNameSuffixes []string // Stack for anonymous type name suffixes
	currentCompUnit      *ast.BLangCompilationUnit
	cx                   *context.CompilerContext
	rangeOf              rangeFn
}

func (n *nodeBuilder) de() *diagnostics.DiagnosticEnv {
	return n.cx.DiagnosticEnv()
}

func (n *nodeBuilder) position(node st.Node) diagnostics.Location {
	if node == nil {
		return diagnostics.NewBuiltinLocation()
	}
	return n.getPosition(node)
}

func (n *nodeBuilder) internalError(message string, node st.Node) {
	n.cx.InternalError(message, n.position(node))
}

func (n *nodeBuilder) unimplemented(message string, node st.Node) {
	if node != nil && node.HasDiagnostics() {
		found := false
		for diagnosticNode, position := range st.DiagnosticSTNodes(node.InternalNode(), node.Position()) {
			found = true
			for _, diagnostic := range diagnosticNode.Diagnostics() {
				textRange := diagnostic.TextRange(diagnosticNode, position)
				location := diagnostics.NewLocation(n.de(), getFileName(node), textRange.StartOffset, textRange.EndOffset)
				n.cx.SyntaxError(diagnosticMessage(diagnostic), location)
			}
		}
		if !found {
			n.internalError("unsupported node has diagnostics but none were found", node)
		}
		return
	}
	n.cx.Unimplemented(message, n.position(node))
}

func newNodeBuilder(cx *context.CompilerContext, rangeOf rangeFn) *nodeBuilder {
	nodeBuilder := &nodeBuilder{
		cx:        cx,
		PackageID: cx.GetDefaultPackage(),
		rangeOf:   rangeOf,
	}
	return nodeBuilder
}

// expectToken checks a token whose kind and presence are fixed by its slot. A missing keyword or punctuation
// token is reported but accepted, so the enclosing node is still built.
func (n *nodeBuilder) expectToken(tok st.Token) bool {
	if isRecoverableMissingToken(tok) {
		n.reportMissingToken(tok)
		return true
	}
	return n.expectSignificantToken(tok)
}

func isRecoverableMissingToken(tok st.Token) bool {
	return tok != nil && tok.IsMissing() && isKeywordOrPunctuation(tok.Kind())
}

func (n *nodeBuilder) reportMissingToken(tok st.Token) {
	if tok.HasDiagnostics() {
		n.reportTokenDiagnostics(tok)
		return
	}
	n.cx.SyntaxError(fmt.Sprintf("expected '%s'", tokenText(tok.Kind())), n.getPosition(tok))
}

// expectSignificantToken checks a token whose kind, text or presence decides the AST that is built; any failure,
// including a missing token, fails the enclosing node.
func (n *nodeBuilder) expectSignificantToken(tok st.Token) bool {
	if tok == nil {
		n.internalError("missing required token", nil)
		return false
	}
	if tok.HasDiagnostics() {
		n.reportTokenDiagnostics(tok)
		return false
	}
	if tok.IsMissing() {
		n.cx.SyntaxError(fmt.Sprintf("expected '%s'", tokenText(tok.Kind())), n.getPosition(tok))
		return false
	}
	return true
}

func (n *nodeBuilder) expectOptionalToken(tok st.Token) bool {
	return tok == nil || n.expectToken(tok)
}

func (n *nodeBuilder) expectOptionalSignificantToken(tok st.Token) bool {
	return tok == nil || n.expectSignificantToken(tok)
}

func (n *nodeBuilder) expectNode(node st.NonTerminalNode) bool {
	nodeDiagnostics := node.InternalNode().Diagnostics()
	if len(nodeDiagnostics) == 0 {
		return true
	}
	for _, diagnostic := range nodeDiagnostics {
		n.cx.SyntaxError(diagnosticMessage(diagnostic), n.getPosition(node))
	}
	return false
}

func (n *nodeBuilder) reportTokenDiagnostics(tok st.Token) {
	deepest := st.FindDeepestDiagnosticSTNode(tok.InternalNode())
	if deepest == nil {
		n.internalError("token has diagnostics but none were found", tok)
		return
	}
	for _, diagnostic := range deepest.Diagnostics() {
		n.cx.SyntaxError(diagnosticMessage(diagnostic), n.getPosition(tok))
	}
}

// rejectNode reports a node of a kind its slot does not accept; the node's own diagnostics, which usually explain
// why it is there, take precedence over the internal error.
func (n *nodeBuilder) rejectNode(node st.Node, message string) {
	switch node := node.(type) {
	case nil:
		n.internalError("missing required node", nil)
		return
	case st.Token:
		if !n.expectSignificantToken(node) {
			return
		}
	case st.NonTerminalNode:
		if !n.expectNode(node) {
			return
		}
	}
	n.internalError(message, node)
}

func tokenText(kind st.SyntaxKind) string {
	if text := kind.StrValue(); text != "" {
		return text
	}
	return kind.String()
}

func (n *nodeBuilder) expectSignificantTokens(tokens st.NodeList[st.Token]) bool {
	for tok := range tokens.Iterator() {
		if !n.expectSignificantToken(tok) {
			return false
		}
	}
	return true
}

// missingNode positions a bad node standing in for a required node that the syntax tree lacks; the caller has
// already reported the internal error.
func missingNode[T interface{ SetPosition(diagnostics.Location) }](bad T) T {
	bad.SetPosition(diagnostics.NewBuiltinLocation())
	return bad
}

func (n *nodeBuilder) transformTopLevel(node st.Node) ast.TopLevelNode {
	topLevel, ok := n.transformTopLevelKind(node)
	if !ok {
		if node == nil {
			return ast.NewBLangBadTopLevelNode(diagnostics.NewBuiltinLocation(), ast.BadTopLevelNodeMissing, nil)
		}
		return n.badTopLevel(node)
	}
	return topLevel
}

func (n *nodeBuilder) transformTopLevelKind(node st.Node) (ast.TopLevelNode, bool) {
	switch node := node.(type) {
	case *st.ImportDeclarationNode:
		return n.transformImportDeclaration(node)
	case *st.FunctionDefinition:
		return n.transformFunctionDefinition(node)
	case *st.ListenerDeclarationNode:
		return n.transformListenerDeclaration(node)
	case *st.TypeDefinitionNode:
		return n.transformTypeDefinition(node)
	case *st.ServiceDeclarationNode:
		return n.transformServiceDeclaration(node)
	case *st.ConstantDeclarationNode:
		return n.transformConstantDeclaration(node)
	case *st.ModuleVariableDeclarationNode:
		return n.transformModuleVariableDeclaration(node)
	case *st.AnnotationDeclarationNode:
		return n.transformAnnotationDeclaration(node)
	case *st.ModuleXMLNamespaceDeclarationNode:
		return n.transformXMLNamespaceDeclarationNode(node)
	case *st.EnumDeclarationNode:
		return n.transformEnumDeclaration(node)
	case *st.ClassDefinitionNode:
		return n.transformClassDefinition(node)
	default:
		n.rejectNode(node, fmt.Sprintf("transformTopLevelKind: unsupported node type %T", node))
		return nil, false
	}
}

func (n *nodeBuilder) transformStatement(node st.StatementNode) ast.StatementNode {
	stmt, ok := n.transformStatementKind(node)
	if !ok {
		if node == nil {
			return ast.NewBLangBadStmt(diagnostics.NewBuiltinLocation(), ast.BadStmtMissing, nil)
		}
		return n.badStmt(node)
	}
	return stmt
}

func (n *nodeBuilder) transformStatementKind(node st.StatementNode) (ast.StatementNode, bool) {
	switch node := node.(type) {
	case *st.AssignmentStatementNode:
		return n.transformAssignmentStatement(node)
	case *st.CompoundAssignmentStatementNode:
		return n.transformCompoundAssignmentStatement(node)
	case *st.VariableDeclarationNode:
		return n.transformVariableDeclaration(node)
	case *st.BlockStatementNode:
		return n.transformBlockStatement(node)
	case *st.BreakStatementNode:
		return n.transformBreakStatement(node)
	case *st.FailStatementNode:
		return n.transformFailStatement(node)
	case *st.ExpressionStatementNode:
		return n.transformExpressionStatement(node)
	case *st.ContinueStatementNode:
		return n.transformContinueStatement(node)
	case *st.IfElseStatementNode:
		return n.transformIfElseStatement(node)
	case *st.WhileStatementNode:
		return n.transformWhileStatement(node)
	case *st.PanicStatementNode:
		return n.transformPanicStatement(node)
	case *st.ReturnStatementNode:
		return n.transformReturnStatement(node)
	case *st.LocalTypeDefinitionStatementNode:
		return n.transformLocalTypeDefinitionStatement(node)
	case *st.LockStatementNode:
		return n.transformLockStatement(node)
	case *st.ForkStatementNode:
		return n.transformForkStatement(node)
	case *st.ForEachStatementNode:
		return n.transformForEachStatement(node)
	case *st.XMLNamespaceDeclarationNode:
		return n.transformXMLNamespaceDeclarationNode(node)
	case *st.TransactionStatementNode:
		return n.transformTransactionStatement(node)
	case *st.RollbackStatementNode:
		return n.transformRollbackStatement(node)
	case *st.RetryStatementNode:
		return n.transformRetryStatement(node)
	case *st.MatchStatementNode:
		return n.transformMatchStatement(node)
	case *st.DoStatementNode:
		return n.transformDoStatement(node)
	default:
		n.rejectNode(node, fmt.Sprintf("transformStatementKind: unsupported node type %T", node))
		return nil, false
	}
}

func (n *nodeBuilder) transformActionOrExpr(node st.Node) ast.BLangActionOrExpression {
	actionOrExpr, ok := n.transformActionOrExprKind(node)
	if !ok {
		if node == nil {
			return ast.NewBLangBadExprOrAction(diagnostics.NewBuiltinLocation(), ast.BadExprMissing, nil)
		}
		return n.badExprOrAction(node)
	}
	return actionOrExpr
}

func (n *nodeBuilder) transformActionOrExprKind(node st.Node) (ast.BLangActionOrExpression, bool) {
	switch node := node.(type) {
	case *st.RemoteMethodCallActionNode:
		return n.transformRemoteMethodCallAction(node)
	case *st.ClientResourceAccessActionNode:
		return n.transformClientResourceAccessAction(node)
	case *st.StartActionNode:
		return n.transformStartAction(node)
	case *st.WaitActionNode:
		return n.transformWaitAction(node)
	case *st.QueryActionNode:
		return n.transformQueryAction(node)
	case *st.AsyncSendActionNode:
		return n.transformAsyncSendAction(node)
	case *st.SyncSendActionNode:
		return n.transformSyncSendAction(node)
	case *st.ReceiveActionNode:
		return n.transformReceiveAction(node)
	case *st.CommitActionNode:
		return n.transformCommitAction(node)
	case *st.FlushActionNode:
		return n.transformFlushAction(node)
	case *st.BracedExpressionNode:
		if node.Kind() == st.BRACED_ACTION {
			return n.transformBracedAction(node)
		}
		return n.transformExprKind(node)
	default:
		return n.transformExprKind(node)
	}
}

func (n *nodeBuilder) transformExpr(node st.Node) ast.BLangExpression {
	expr, ok := n.transformExprKind(node)
	if !ok {
		if node == nil {
			return ast.NewBLangBadExprOrAction(diagnostics.NewBuiltinLocation(), ast.BadExprMissing, nil)
		}
		return n.badExprOrAction(node)
	}
	return expr
}

func (n *nodeBuilder) transformExprKind(node st.Node) (ast.BLangExpression, bool) {
	if node == nil {
		n.internalError("missing action or expression", nil)
		return nil, false
	}
	kind := node.Kind()
	switch {
	case isSimpleLiteral(kind):
		return n.createSimpleLiteral(node)
	case kind == st.SIMPLE_NAME_REFERENCE || kind == st.QUALIFIED_NAME_REFERENCE || kind == st.IDENTIFIER_TOKEN:
		return n.createVarRef(node)
	case isType(kind):
		return n.createTypedescExpr(node)
	}
	switch node := node.(type) {
	case *st.BracedExpressionNode:
		if node.Kind() == st.BRACED_ACTION {
			n.rejectNode(node, "transformExprKind: braced action is not an expression")
			return nil, false
		}
		return n.transformBracedExpression(node)
	case *st.BinaryExpressionNode:
		return n.transformBinaryExpression(node)
	case *st.CheckExpressionNode:
		return n.transformCheckExpression(node)
	case *st.FieldAccessExpressionNode:
		return n.transformFieldAccessExpression(node)
	case *st.FunctionCallExpressionNode:
		return n.transformFunctionCallExpression(node)
	case *st.MethodCallExpressionNode:
		return n.transformMethodCallExpression(node)
	case *st.MappingConstructorExpressionNode:
		return n.transformMappingConstructorExpression(node)
	case *st.IndexedExpressionNode:
		return n.transformIndexedExpression(node)
	case *st.TypeofExpressionNode:
		return n.transformTypeofExpression(node)
	case *st.UnaryExpressionNode:
		return n.transformUnaryExpression(node)
	case *st.InferredTypedescDefaultNode:
		return n.transformInferredTypedescDefault(node)
	case *st.ObjectConstructorExpressionNode:
		return n.transformObjectConstructorExpression(node)
	case *st.TypeTestExpressionNode:
		return n.transformTypeTestExpression(node)
	case *st.TrapExpressionNode:
		return n.transformTrapExpression(node)
	case *st.ListConstructorExpressionNode:
		return n.transformListConstructorExpression(node)
	case *st.TypeCastExpressionNode:
		return n.transformTypeCastExpression(node)
	case *st.TableConstructorExpressionNode:
		return n.transformTableConstructorExpression(node)
	case *st.LetExpressionNode:
		return n.transformLetExpression(node)
	case *st.TemplateExpressionNode:
		return n.transformTemplateExpression(node)
	case *st.ExplicitAnonymousFunctionExpressionNode:
		return n.transformExplicitAnonymousFunctionExpression(node)
	case *st.ImplicitAnonymousFunctionExpressionNode:
		return n.transformImplicitAnonymousFunctionExpression(node)
	case *st.ExplicitNewExpressionNode:
		return n.transformExplicitNewExpression(node)
	case *st.ImplicitNewExpressionNode:
		return n.transformImplicitNewExpression(node)
	case *st.QueryExpressionNode:
		return n.transformQueryExpression(node)
	case *st.AnnotAccessExpressionNode:
		return n.transformAnnotAccessExpression(node)
	case *st.OptionalFieldAccessExpressionNode:
		return n.transformOptionalFieldAccessExpression(node)
	case *st.ConditionalExpressionNode:
		return n.transformConditionalExpression(node)
	case *st.TransactionalExpressionNode:
		return n.transformTransactionalExpression(node)
	case *st.ByteArrayLiteralNode:
		return n.transformByteArrayLiteral(node)
	case *st.XMLFilterExpressionNode:
		return n.transformXMLFilterExpression(node)
	case *st.XMLStepExpressionNode:
		return n.transformXMLStepExpression(node)
	case *st.RequiredExpressionNode:
		return n.transformRequiredExpression(node)
	case *st.ErrorConstructorExpressionNode:
		return n.transformErrorConstructorExpression(node)
	case *st.NaturalExpressionNode:
		return n.transformNaturalExpression(node)
	case *st.PositionalArgumentNode:
		return n.transformPositionalArgument(node)
	case *st.NamedArgumentNode:
		return n.transformNamedArgument(node)
	case *st.RestArgumentNode:
		return n.transformRestArgument(node)
	case *st.WildcardBindingPatternNode:
		return n.transformWildcardBindingPattern(node)
	case *st.BasicLiteralNode:
		return n.transformBasicLiteral(node)
	default:
		n.rejectNode(node, fmt.Sprintf("transformExprKind: unsupported node type %T", node))
		return nil, false
	}
}

func (n *nodeBuilder) transformType(node st.Node) ast.BType {
	typeNode, ok := n.transformTypeInner(node)
	if !ok {
		if node == nil {
			return ast.NewBLangBadTypeNode(diagnostics.NewBuiltinLocation(), ast.BadTypeMissing, nil)
		}
		return n.badTypeNode(node)
	}
	return typeNode
}

func (n *nodeBuilder) transformTypeInner(node st.Node) (ast.BType, bool) {
	switch node := node.(type) {
	case *st.BuiltinSimpleNameReferenceNode:
		return n.createBuiltInTypeNode(node)
	case *st.NilTypeDescriptorNode:
		return n.transformNilTypeDescriptor(node)
	case *st.QualifiedNameReferenceNode:
		return n.transformQualifiedTypeReference(node)
	case *st.SimpleNameReferenceNode:
		return n.transformSimpleTypeReference(node)
	case *st.ObjectTypeDescriptorNode:
		return n.transformObjectTypeDescriptor(node)
	case *st.RecordTypeDescriptorNode:
		return n.transformRecordTypeDescriptor(node)
	case *st.OptionalTypeDescriptorNode:
		return n.transformOptionalTypeDescriptor(node)
	case *st.MapTypeDescriptorNode:
		return n.transformMapTypeDescriptor(node)
	case *st.UnionTypeDescriptorNode:
		return n.transformUnionTypeDescriptor(node)
	case *st.StreamTypeDescriptorNode:
		return n.transformStreamTypeDescriptor(node)
	case *st.TableTypeDescriptorNode:
		return n.transformTableTypeDescriptor(node)
	case *st.FunctionTypeDescriptorNode:
		return n.transformFunctionTypeDescriptor(node)
	case *st.TupleTypeDescriptorNode:
		return n.transformTupleTypeDescriptor(node)
	case *st.ParenthesisedTypeDescriptorNode:
		return n.transformParenthesisedTypeDescriptor(node)
	case *st.IntersectionTypeDescriptorNode:
		return n.transformIntersectionTypeDescriptor(node)
	case *st.SingletonTypeDescriptorNode:
		return n.transformSingletonTypeDescriptor(node)
	case *st.ArrayTypeDescriptorNode:
		return n.transformArrayTypeDescriptor(node)
	case *st.TypeReferenceTypeDescNode:
		return n.transformTypeReferenceTypeDesc(node)
	case *st.DistinctTypeDescriptorNode:
		return n.transformDistinctTypeDescriptor(node)
	case *st.ParameterizedTypeDescriptorNode:
		return n.transformParameterizedTypeDescriptor(node)
	case st.Token:
		if node.Kind() == st.IDENTIFIER_TOKEN {
			return n.createUserDefinedTypeFromName(node)
		}
		return n.createBuiltInTypeNodeFromToken(node)
	default:
		n.rejectNode(node, fmt.Sprintf("transformTypeInner: unsupported node type %T", node))
		return nil, false
	}
}

const underscoreKeywordMessage = "'_' is a keyword, and may not be used as an identifier"

func (n *nodeBuilder) transformIdentifier(token st.Token) ast.IdentifierNode {
	return n.transformIdentifierWith(token, nil, underscoreKeywordMessage)
}

// transformIdentifierWith is transformIdentifier with mapValue (when non-nil) applied to the value of a
// successfully built identifier and underscoreMessage reported for `_`.
func (n *nodeBuilder) transformIdentifierWith(token st.Token, mapValue func(string) string, underscoreMessage string) ast.IdentifierNode {
	identifier, ok := n.createBLangIdentifierWith(token, underscoreMessage)
	if !ok {
		return n.recoverIdentifier(token)
	}
	if mapValue != nil {
		identifier.Value = mapValue(identifier.Value)
	}
	return &identifier
}

func (n *nodeBuilder) transformModulePrefixIdentifier(token st.Token) ast.IdentifierNode {
	identifier, ok := n.createBLangModulePrefixIdentifier(token)
	if !ok {
		return n.recoverIdentifier(token)
	}
	return &identifier
}

// recoverIdentifier is the bad identifier the identifier recovery points substitute for a failed token.
func (n *nodeBuilder) recoverIdentifier(token st.Token) *ast.BLangBadIdentifier {
	if token == nil {
		return missingNode(&ast.BLangBadIdentifier{})
	}
	return n.badIdentifier(token)
}

// createBLangIdentifier builds an identifier for a value-typed BLangIdentifier slot, which cannot hold a bad
// identifier; any failure unwinds to the enclosing recovery point.
func (n *nodeBuilder) createBLangIdentifier(token st.Token) (ast.BLangIdentifier, bool) {
	return n.createBLangIdentifierWith(token, underscoreKeywordMessage)
}

func (n *nodeBuilder) createBLangIdentifierWith(token st.Token, underscoreMessage string) (ast.BLangIdentifier, bool) {
	if !n.expectToken(token) {
		return ast.BLangIdentifier{}, false
	}
	pos := n.getPosition(token)
	if token.Text() == "'" {
		n.cx.InternalError("invalid identifier", pos)
		return ast.BLangIdentifier{}, false
	}
	identifierName := token.Text()
	if identifierName == "_" || identifierName == "'_" {
		n.cx.SyntaxError(underscoreMessage, pos)
		return ast.BLangIdentifier{}, false
	}
	return createIdentifier(pos, &identifierName, &identifierName), true
}

func (n *nodeBuilder) createBLangModulePrefixIdentifier(token st.Token) (ast.BLangIdentifier, bool) {
	if token != nil && token.Text() == "'_" {
		if !n.expectToken(token) {
			return ast.BLangIdentifier{}, false
		}
		identifierName := token.Text()
		return createIdentifier(n.getPosition(token), &identifierName, &identifierName), true
	}
	return n.createBLangIdentifier(token)
}

func getFileName(node st.Node) string {
	st := node.SyntaxTree()
	return st.FilePath()
}

func diagnosticMessage(diagnostic st.STNodeDiagnostic) string {
	return strings.ReplaceAll(strings.TrimPrefix(diagnostic.DiagnosticCode().MessageKey(), "error."), ".", " ")
}

func (n *nodeBuilder) getPosition(node st.Node) diagnostics.Location {
	textRange := n.rangeOf(node)
	return diagnostics.NewLocation(n.de(), getFileName(node), textRange.StartOffset, textRange.EndOffset)
}

func (n *nodeBuilder) getPositionRange(startNode st.Node, endNode st.Node) diagnostics.Location {
	startRange := n.rangeOf(startNode)
	endRange := n.rangeOf(endNode)
	return diagnostics.NewLocation(n.de(), getFileName(startNode), startRange.StartOffset, endRange.EndOffset)
}

func (n *nodeBuilder) getPositionWithoutMetadata(node st.Node) diagnostics.Location {
	pos := n.getPosition(node)
	return diagnostics.NewLocation(n.de(), getFileName(node), metadataExcludedStartOffset(n.rangeOf, node, pos.StartOffset()), pos.EndOffset())
}

func metadataExcludedStartOffset(rangeOf rangeFn, node st.Node, defaultStartOffset int) int {
	nonTerminalNode := node.(st.NonTerminalNode)

	var firstChild, secondChild st.Node
	childIndex := 0
	for child := range nonTerminalNode.ChildNodes() {
		if childIndex == 0 {
			firstChild = child
			childIndex++
		} else if childIndex == 1 {
			secondChild = child
			break
		}
	}

	if firstChild != nil && firstChild.Kind() == st.METADATA && secondChild != nil {
		return rangeOf(secondChild).StartOffset
	}
	return defaultStartOffset
}

// asToken converts a possibly-nil identifier token pointer into a Token interface that is nil when absent.
func asToken(token *st.IdentifierToken) st.Token {
	if token == nil {
		return nil
	}
	return token
}

type metadataParts struct {
	documentation *ast.BLangMarkdownDocumentation
	annotations   []ast.BLangAnnotationAttachment
}

func (n *nodeBuilder) transformMetadata(metadata *st.MetadataNode) (metadataParts, bool) {
	var parts metadataParts
	if metadata == nil {
		return parts, true
	}
	if !n.expectNode(metadata) {
		return parts, false
	}
	if docString := metadata.DocumentationString(); docString != nil {
		documentation, ok := n.createMarkdownDocumentationAttachment(docString)
		if !ok {
			return parts, false
		}
		parts.documentation = documentation
	}
	annotations, ok := n.transformAnnotations(metadata.Annotations())
	if !ok {
		return parts, false
	}
	parts.annotations = annotations
	return parts, true
}

func applyMetadata(parts metadataParts, target ast.AnnotatableNode) {
	if parts.documentation != nil {
		setDocumentation(target, parts.documentation)
	}
	addAnnotationAttachments(parts.annotations, target)
}

func setDocumentation(target ast.AnnotatableNode, documentation *ast.BLangMarkdownDocumentation) {
	switch target := target.(type) {
	case *ast.BLangAnnotation:
		target.MarkdownDocumentationAttachment = documentation
	case *ast.BLangClassDefinition:
		target.MarkdownDocumentationAttachment = documentation
	case *ast.BLangService:
		target.MarkdownDocumentationAttachment = documentation
	case *ast.BLangVariable:
		target.MarkdownDocumentationAttachment = documentation
	case *ast.BLangFunction:
		target.MarkdownDocumentationAttachment = documentation
	case *ast.BLangResourceMethod:
		target.MarkdownDocumentationAttachment = documentation
	case *ast.BLangMemberTypeDesc:
		target.MarkdownDocumentationAttachment = documentation
	}
}

func (n *nodeBuilder) transformAnnotations(annotations st.NodeList[*st.AnnotationNode]) ([]ast.BLangAnnotationAttachment, bool) {
	var attachments []ast.BLangAnnotationAttachment
	for annotation := range annotations.Iterator() {
		attachment, ok := n.transformAnnotation(annotation)
		if !ok {
			return nil, false
		}
		attachments = append(attachments, *attachment)
	}
	return attachments, true
}

func addAnnotationAttachments(attachments []ast.BLangAnnotationAttachment, target ast.AnnotatableNode) {
	for _, attachment := range attachments {
		target.AddAnnotationAttachment(attachment)
	}
}

func (n *nodeBuilder) createTrueLiteral(pos diagnostics.Location) *ast.BLangLiteral {
	return ast.NewBLangLiteral(pos, ast.LiteralKindBoolean, true, "true", false)
}

type markdownDocumentationBuilder struct {
	doc                    *ast.BLangMarkdownDocumentation
	documentationLines     []ast.BLangMarkdownDocumentationLine
	parameters             []ast.BLangMarkdownParameterDocumentation
	references             []ast.BLangMarkdownReferenceDocumentation
	bLangParaDoc           *ast.BLangMarkdownParameterDocumentation
	bLangReturnParaDoc     *ast.BLangMarkdownReturnParameterDocumentation
	bLangDeprecationDoc    *ast.BLangMarkDownDeprecationDocumentation
	bLangDeprecatedParaDoc *ast.BLangMarkDownDeprecatedParametersDocumentation
}

func (b *markdownDocumentationBuilder) flushParameterDoc() {
	if b.bLangParaDoc == nil {
		return
	}
	if b.bLangDeprecatedParaDoc != nil {
		b.bLangDeprecatedParaDoc.Parameters = append(b.bLangDeprecatedParaDoc.Parameters, *b.bLangParaDoc)
	} else if b.bLangDeprecationDoc != nil {
		b.bLangDeprecatedParaDoc = &ast.BLangMarkDownDeprecatedParametersDocumentation{}
		b.bLangDeprecatedParaDoc.Parameters = append(b.bLangDeprecatedParaDoc.Parameters, *b.bLangParaDoc)
		b.bLangDeprecationDoc = nil
	} else {
		b.parameters = append(b.parameters, *b.bLangParaDoc)
	}
}

// createMarkdownDocumentationAttachment creates a BLangMarkdownDocumentation from a documentation string node
func (n *nodeBuilder) createMarkdownDocumentationAttachment(docStringNode st.Node) (*ast.BLangMarkdownDocumentation, bool) {
	markdownDocumentationNode, ok := docStringNode.(*st.MarkdownDocumentationNode)
	if !ok {
		n.internalError(fmt.Sprintf("unexpected documentation node %T", docStringNode), docStringNode)
		return nil, false
	}
	if !n.expectNode(markdownDocumentationNode) {
		return nil, false
	}

	b := &markdownDocumentationBuilder{
		doc:                &ast.BLangMarkdownDocumentation{},
		documentationLines: []ast.BLangMarkdownDocumentationLine{},
		parameters:         []ast.BLangMarkdownParameterDocumentation{},
		references:         []ast.BLangMarkdownReferenceDocumentation{},
	}
	docLineList := markdownDocumentationNode.DocumentationLines()
	for singleDocLine := range docLineList.Iterator() {
		if !n.addMarkdownDocumentationLine(b, singleDocLine) {
			return nil, false
		}
	}
	b.flushParameterDoc()

	doc := b.doc
	doc.DocumentationLines = b.documentationLines
	doc.Parameters = b.parameters
	doc.References = b.references
	doc.DeprecationDocumentation = b.bLangDeprecationDoc
	doc.DeprecatedParametersDocumentation = b.bLangDeprecatedParaDoc
	doc.SetPosition(n.getPosition(markdownDocumentationNode))
	return doc, true
}

func (n *nodeBuilder) addMarkdownDocumentationLine(b *markdownDocumentationBuilder, singleDocLine st.Node) bool {
	switch singleDocLine.Kind() {
	case st.MARKDOWN_DOCUMENTATION_LINE, st.MARKDOWN_REFERENCE_DOCUMENTATION_LINE:
		docLineNode := singleDocLine.(*st.MarkdownDocumentationLineNode)
		if !n.expectNode(docLineNode) || !n.expectToken(docLineNode.HashToken()) {
			return false
		}
		docText, ok := n.addReferencesAndReturnDocumentationText(&b.references, docLineNode.DocumentElements())
		if !ok {
			return false
		}
		if b.bLangDeprecationDoc != nil {
			b.bLangDeprecationDoc.DeprecationDocumentationLines = append(b.bLangDeprecationDoc.DeprecationDocumentationLines, docText)
		} else if b.bLangReturnParaDoc != nil {
			b.bLangReturnParaDoc.ReturnParameterDocumentationLines = append(b.bLangReturnParaDoc.ReturnParameterDocumentationLines, docText)
		} else if b.bLangParaDoc != nil {
			b.bLangParaDoc.ParameterDocumentationLines = append(b.bLangParaDoc.ParameterDocumentationLines, docText)
		} else {
			bLangDocLine := ast.BLangMarkdownDocumentationLine{}
			bLangDocLine.Text = docText
			bLangDocLine.SetPosition(n.getPosition(docLineNode))
			b.documentationLines = append(b.documentationLines, bLangDocLine)
		}
	case st.MARKDOWN_PARAMETER_DOCUMENTATION_LINE:
		b.flushParameterDoc()
		parameterDocLineNode := singleDocLine.(*st.MarkdownParameterDocumentationLineNode)
		if !n.expectNode(parameterDocLineNode) || !n.expectToken(parameterDocLineNode.HashToken()) ||
			!n.expectToken(parameterDocLineNode.PlusToken()) {
			return false
		}
		parameterName := parameterDocLineNode.ParameterName()
		if !n.expectToken(parameterName) || !n.expectToken(parameterDocLineNode.MinusToken()) {
			return false
		}
		b.bLangParaDoc = &ast.BLangMarkdownParameterDocumentation{}
		paraName := &ast.BLangIdentifier{}
		parameterNameValue := unescapeUnicodeCodepoints(parameterName.Text())
		paraName.OriginalValue = parameterNameValue
		if n.stringStartsWithSingleQuote(parameterNameValue) {
			parameterNameValue = parameterNameValue[1:]
		}
		paraName.Value = parameterNameValue
		b.bLangParaDoc.ParameterName = paraName
		paraDocText, ok := n.addReferencesAndReturnDocumentationText(&b.references, parameterDocLineNode.DocumentElements())
		if !ok {
			return false
		}
		b.bLangParaDoc.ParameterDocumentationLines = append(b.bLangParaDoc.ParameterDocumentationLines, paraDocText)
		b.bLangParaDoc.SetPosition(n.getPosition(parameterName))
	case st.MARKDOWN_RETURN_PARAMETER_DOCUMENTATION_LINE:
		returnParaDocLineNode := singleDocLine.(*st.MarkdownParameterDocumentationLineNode)
		if !n.expectNode(returnParaDocLineNode) || !n.expectToken(returnParaDocLineNode.HashToken()) ||
			!n.expectToken(returnParaDocLineNode.PlusToken()) || !n.expectToken(returnParaDocLineNode.ParameterName()) ||
			!n.expectToken(returnParaDocLineNode.MinusToken()) {
			return false
		}
		b.bLangReturnParaDoc = &ast.BLangMarkdownReturnParameterDocumentation{}
		returnParaDocText, ok := n.addReferencesAndReturnDocumentationText(&b.references, returnParaDocLineNode.DocumentElements())
		if !ok {
			return false
		}
		b.bLangReturnParaDoc.ReturnParameterDocumentationLines = append(b.bLangReturnParaDoc.ReturnParameterDocumentationLines, returnParaDocText)
		b.bLangReturnParaDoc.SetPosition(n.getPosition(returnParaDocLineNode))
		b.doc.ReturnParameter = b.bLangReturnParaDoc
	case st.MARKDOWN_DEPRECATION_DOCUMENTATION_LINE:
		deprecationDocLineNode := singleDocLine.(*st.MarkdownDocumentationLineNode)
		if !n.expectNode(deprecationDocLineNode) || !n.expectToken(deprecationDocLineNode.HashToken()) {
			return false
		}
		b.bLangDeprecationDoc = &ast.BLangMarkDownDeprecationDocumentation{}
		lineText, ok := n.deprecationLineText(deprecationDocLineNode.DocumentElements())
		if !ok {
			return false
		}
		b.bLangDeprecationDoc.AddDeprecationLine("# " + lineText)
		b.bLangDeprecationDoc.SetPosition(n.getPosition(deprecationDocLineNode))
	case st.MARKDOWN_CODE_BLOCK:
		codeBlockNode := singleDocLine.(*st.MarkdownCodeBlockNode)
		return n.transformCodeBlock(&b.documentationLines, codeBlockNode)
	default:
		if tok, ok := singleDocLine.(st.Token); ok {
			return n.expectSignificantToken(tok)
		}
		if nonTerminal, ok := singleDocLine.(st.NonTerminalNode); ok {
			return n.expectNode(nonTerminal)
		}
	}
	return true
}

func (n *nodeBuilder) deprecationLineText(docElements st.NodeList[st.Node]) (string, bool) {
	var lineText string
	i := -1
	for element := range docElements.Iterator() {
		i++
		token, ok := element.(st.Token)
		if !ok {
			if nonTerminal, ok := element.(st.NonTerminalNode); ok && !n.expectNode(nonTerminal) {
				return "", false
			}
			continue
		}
		if !n.expectSignificantToken(token) {
			return "", false
		}
		if i == 0 {
			lineText = token.Text()
		}
	}
	return lineText, true
}

func createIdentifier(pos diagnostics.Location, value, originalValue *string) ast.BLangIdentifier {
	if value == nil {
		return ast.NewBLangIdentifier(pos, "", "")
	}
	identifierValue, _ := normalizedIdentifierValue(*value)
	return ast.NewBLangIdentifier(pos, identifierValue, *originalValue)
}

func normalizedIdentifierValue(value string) (string, bool) {
	const IDENTIFIER_LITERAL_PREFIX = "'"
	if len(value) > 0 && value[0:1] == IDENTIFIER_LITERAL_PREFIX {
		return value[1:], true
	}
	return value, false
}

func (n *nodeBuilder) createIgnoreIdentifier(node st.Node) ast.BLangIdentifier {
	pos := n.getPosition(node)
	ignoreValue := string(model.IGNORE)
	identifier := createIdentifier(pos, &ignoreValue, &ignoreValue)
	return identifier
}

func (n *nodeBuilder) transformQualifiedTypeReference(node *st.QualifiedNameReferenceNode) (*ast.BLangUserDefinedType, bool) {
	if !n.expectNode(node) {
		return nil, false
	}
	pkgAlias, ok := n.createBLangModulePrefixIdentifier(node.ModulePrefix())
	if !ok || !n.expectColon(node.Colon()) {
		return nil, false
	}
	typeName, ok := n.createBLangIdentifier(asToken(node.Identifier()))
	if !ok {
		return nil, false
	}
	return createUserDefinedType(n.getPosition(node), pkgAlias, typeName), true
}

func (n *nodeBuilder) transformSimpleTypeReference(node *st.SimpleNameReferenceNode) (*ast.BLangUserDefinedType, bool) {
	if !n.expectNode(node) {
		return nil, false
	}
	return n.createUserDefinedTypeFromName(node.Name())
}

func (n *nodeBuilder) createUserDefinedTypeFromName(name st.Token) (*ast.BLangUserDefinedType, bool) {
	typeName, ok := n.createBLangIdentifier(name)
	if !ok {
		return nil, false
	}
	emptyStr := ""
	pkgAlias := createIdentifier(diagnostics.NewBuiltinLocation(), &emptyStr, &emptyStr)
	return createUserDefinedType(n.getPosition(name), pkgAlias, typeName), true
}

// transformTypeReferenceName transforms a name reference that must resolve to a user-defined type, such as a
// type inclusion or an error constructor's type reference.
func (n *nodeBuilder) transformTypeReferenceName(node st.Node) (*ast.BLangUserDefinedType, bool) {
	switch node := node.(type) {
	case *st.QualifiedNameReferenceNode:
		return n.transformQualifiedTypeReference(node)
	case *st.SimpleNameReferenceNode:
		return n.transformSimpleTypeReference(node)
	default:
		n.internalError(fmt.Sprintf("type reference must be a user-defined type, found %T", node), node)
		return nil, false
	}
}

func (n *nodeBuilder) expectColon(colon st.Node) bool {
	if token, ok := colon.(st.Token); ok {
		return n.expectToken(token)
	}
	if colon == nil {
		return n.expectToken(nil)
	}
	n.internalError(fmt.Sprintf("unexpected colon node %T", colon), colon)
	return false
}

// isDeclaredWithVar checks if a type node is declared with var
func isDeclaredWithVar(typeNode st.Node) bool {
	if typeNode == nil || typeNode.Kind() == st.VAR_TYPE_DESC {
		return true
	}
	return false
}

// transformVarTypeKeyword checks the `var` type keyword of a declaration that infers its type.
func (n *nodeBuilder) transformVarTypeKeyword(typeNode st.Node) bool {
	switch typeNode := typeNode.(type) {
	case nil:
		return true
	case *st.BuiltinSimpleNameReferenceNode:
		return n.expectNode(typeNode) && n.expectSignificantToken(typeNode.Name())
	case st.Token:
		return n.expectSignificantToken(typeNode)
	default:
		n.internalError(fmt.Sprintf("unexpected var type node %T", typeNode), typeNode)
		return false
	}
}

func (n *nodeBuilder) createBuiltInTypeNode(typeNode *st.BuiltinSimpleNameReferenceNode) (ast.BType, bool) {
	if !n.expectNode(typeNode) {
		return nil, false
	}
	if typeNode.Kind() == st.VAR_TYPE_DESC {
		n.internalError("createBuiltInTypeNode: unexpected var type descriptor", typeNode)
		return nil, false
	}
	name := typeNode.Name()
	if !n.expectToken(name) {
		return nil, false
	}
	return n.createBuiltInType(typeNode, name.Text())
}

func (n *nodeBuilder) transformNilTypeDescriptor(nilTypeDescriptorNode *st.NilTypeDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(nilTypeDescriptorNode) || !n.expectToken(nilTypeDescriptorNode.OpenParenToken()) ||
		!n.expectToken(nilTypeDescriptorNode.CloseParenToken()) {
		return nil, false
	}
	return n.createBuiltInType(nilTypeDescriptorNode, "()")
}

// createBuiltInTypeNodeFromToken handles built-in type keywords that appear as bare tokens (e.g. `map`).
func (n *nodeBuilder) createBuiltInTypeNodeFromToken(token st.Token) (ast.BType, bool) {
	if !n.expectSignificantToken(token) {
		return nil, false
	}
	return n.createBuiltInType(token, token.Text())
}

func (n *nodeBuilder) createBuiltInType(typeNode st.Node, typeText string) (ast.BType, bool) {
	typeKind, ok := n.stringToTypeKind(typeText, typeNode)
	if !ok {
		return nil, false
	}
	switch typeNode.Kind() {
	case st.BOOLEAN_TYPE_DESC,
		st.INT_TYPE_DESC,
		st.BYTE_TYPE_DESC,
		st.FLOAT_TYPE_DESC,
		st.DECIMAL_TYPE_DESC,
		st.STRING_TYPE_DESC,
		st.ANY_TYPE_DESC,
		st.NIL_TYPE_DESC,
		st.HANDLE_TYPE_DESC,
		st.ANYDATA_TYPE_DESC,
		st.READONLY_TYPE_DESC,
		st.NEVER_TYPE_DESC:
		valueType := ast.BLangValueType{}
		valueType.TypeKind = typeKind
		valueType.SetPosition(n.getPosition(typeNode))
		return &valueType, true
	default:
		builtInValueType := ast.BLangBuiltInRefTypeNode{}
		builtInValueType.TypeKind = typeKind
		builtInValueType.SetPosition(n.getPosition(typeNode))
		return &builtInValueType, true
	}
}

// remoteMethodName mangles the name of an attached remote method; `init` keeps its name so that it is still
// recognized as the initializer.
func remoteMethodName(name string) string {
	if model.Name(name) == model.USER_DEFINED_INIT_SUFFIX {
		return name
	}
	return model.RemoteMethodName(name)
}

// identifierTokenValue returns the normalized value an identifier token spells, or "" when it is absent.
func identifierTokenValue(token st.Token) string {
	if token == nil {
		return ""
	}
	value, _ := normalizedIdentifierValue(token.Text())
	return value
}

// createBLangNameReference transforms a name reference into its package alias and name identifiers.
func (n *nodeBuilder) createBLangNameReference(node st.Node) (ast.IdentifierNode, ast.IdentifierNode, bool) {
	var nameToken st.Token
	switch node := node.(type) {
	case *st.QualifiedNameReferenceNode:
		if !n.expectNode(node) {
			return nil, nil, false
		}
		pkgAlias := n.transformModulePrefixIdentifier(node.ModulePrefix())
		if !n.expectColon(node.Colon()) {
			return nil, nil, false
		}
		name := n.transformIdentifier(asToken(node.Identifier()))
		return pkgAlias, name, true
	case *st.BuiltinSimpleNameReferenceNode:
		if !n.expectNode(node) {
			return nil, nil, false
		}
		nameToken = node.Name()
	case *st.SimpleNameReferenceNode:
		if !n.expectNode(node) {
			return nil, nil, false
		}
		nameToken = node.Name()
	case st.Token:
		nameToken = node
	default:
		n.internalError(fmt.Sprintf("unexpected name reference node %T", node), node)
		return nil, nil, false
	}

	emptyStr := ""
	pkgAlias := createIdentifier(diagnostics.NewBuiltinLocation(), &emptyStr, &emptyStr)
	name := n.transformIdentifier(nameToken)
	return &pkgAlias, name, true
}

func (n *nodeBuilder) createVarRef(node st.Node) (*ast.BLangVarRef, bool) {
	pkgAlias, name, ok := n.createBLangNameReference(node)
	if !ok {
		return nil, false
	}
	bLVarRef := &ast.BLangVarRef{}
	bLVarRef.SetPosition(n.getPosition(node))
	bLVarRef.PkgAlias = pkgAlias
	bLVarRef.VariableName = name
	return bLVarRef, true
}

func (n *nodeBuilder) createTypedescExpr(node st.Node) (*ast.BLangTypedescExpr, bool) {
	return ast.NewBLangTypedescExpr(n.getPosition(node), n.transformType(node)), true
}

// transformArguments transforms a separated list of call arguments.
func (n *nodeBuilder) transformArguments(arguments st.NodeList[st.FunctionArgumentNode]) ([]ast.BLangExpression, bool) {
	var args []ast.BLangExpression
	for arg, separator := range arguments.SeparatedElements() {
		args = append(args, n.transformExpr(arg))
		if !n.expectOptionalToken(separator) {
			return nil, false
		}
	}
	return args, true
}

func (n *nodeBuilder) transformParenthesizedArgList(argList *st.ParenthesizedArgList) ([]ast.BLangExpression, bool) {
	if !n.expectNode(argList) || !n.expectToken(argList.OpenParenToken()) {
		return nil, false
	}
	args, ok := n.transformArguments(argList.Arguments())
	if !ok || !n.expectToken(argList.CloseParenToken()) {
		return nil, false
	}
	return args, true
}

// createBLangInvocation creates a BLangInvocation from a name node and parenthesized arguments
func (n *nodeBuilder) createBLangInvocation(nameNode st.Node, openParen st.Token, arguments st.NodeList[st.FunctionArgumentNode], closeParen st.Token, position diagnostics.Location) (*ast.BLangInvocation, bool) {
	bLInvocation := ast.BLangInvocation{}
	pkgAlias, name, ok := n.createBLangNameReference(nameNode)
	if !ok || !n.expectToken(openParen) {
		return nil, false
	}
	bLInvocation.PkgAlias = pkgAlias
	bLInvocation.Name = name
	args, ok := n.transformArguments(arguments)
	if !ok || !n.expectToken(closeParen) {
		return nil, false
	}
	bLInvocation.ArgExprs = args
	bLInvocation.SetPosition(position)
	return &bLInvocation, true
}

// isSimpleLiteral checks if the syntax kind is a simple literal
func isSimpleLiteral(syntaxKind st.SyntaxKind) bool {
	switch syntaxKind {
	case st.STRING_LITERAL, st.NUMERIC_LITERAL, st.BOOLEAN_LITERAL, st.NIL_LITERAL, st.NULL_LITERAL:
		return true
	default:
		return false
	}
}

// isType checks if the syntax kind is a type descriptor
func isType(nodeKind st.SyntaxKind) bool {
	switch nodeKind {
	case st.RECORD_TYPE_DESC,
		st.OBJECT_TYPE_DESC,
		st.NIL_TYPE_DESC,
		st.OPTIONAL_TYPE_DESC,
		st.ARRAY_TYPE_DESC,
		st.INT_TYPE_DESC,
		st.BYTE_TYPE_DESC,
		st.FLOAT_TYPE_DESC,
		st.DECIMAL_TYPE_DESC,
		st.STRING_TYPE_DESC,
		st.BOOLEAN_TYPE_DESC,
		st.XML_TYPE_DESC,
		st.JSON_TYPE_DESC,
		st.HANDLE_TYPE_DESC,
		st.ANY_TYPE_DESC,
		st.ANYDATA_TYPE_DESC,
		st.NEVER_TYPE_DESC,
		st.VAR_TYPE_DESC,
		st.SERVICE_TYPE_DESC,
		st.MAP_TYPE_DESC,
		st.UNION_TYPE_DESC,
		st.ERROR_TYPE_DESC,
		st.STREAM_TYPE_DESC,
		st.TABLE_TYPE_DESC,
		st.FUNCTION_TYPE_DESC,
		st.TUPLE_TYPE_DESC,
		st.PARENTHESISED_TYPE_DESC,
		st.READONLY_TYPE_DESC,
		st.DISTINCT_TYPE_DESC,
		st.INTERSECTION_TYPE_DESC,
		st.SINGLETON_TYPE_DESC,
		st.TYPE_REFERENCE_TYPE_DESC:
		return true
	default:
		return false
	}
}

// getIntegerLiteral parses integer literals (decimal/hex)
func (n *nodeBuilder) getIntegerLiteral(literal st.Node, literalTokenKind st.SyntaxKind, textValue string) (any, bool) {
	switch literalTokenKind {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN, st.HEX_INTEGER_LITERAL_TOKEN:
	default:
		n.internalError("unexpected integer literal token kind", literal)
		return nil, false
	}
	value, ok := balCommon.ParseIntLiteral(textValue)
	if !ok {
		n.internalError("failed to parse numeric literal", literal)
		return nil, false
	}
	return value, true
}

// withinByteRange checks if integer is in byte range (0-255)
func withinByteRange(value any) bool {
	switch v := value.(type) {
	case int64:
		return v <= 255 && v >= 0
	case int:
		return v <= 255 && v >= 0
	default:
		return false
	}
}

// isTokenInRegExp checks if token is in regexp context
func isTokenInRegExp(kind st.SyntaxKind) bool {
	switch kind {
	case st.RE_LITERAL_CHAR,
		st.RE_CONTROL_ESCAPE,
		st.RE_NUMERIC_ESCAPE,
		st.RE_SIMPLE_CHAR_CLASS_CODE,
		st.RE_PROPERTY,
		st.RE_UNICODE_SCRIPT_START,
		st.RE_UNICODE_PROPERTY_VALUE,
		st.RE_UNICODE_GENERAL_CATEGORY_START,
		st.RE_UNICODE_GENERAL_CATEGORY_NAME,
		st.RE_FLAGS_VALUE,
		st.DIGIT,
		st.ASTERISK_TOKEN,
		st.PLUS_TOKEN,
		st.QUESTION_MARK_TOKEN,
		st.DOT_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.OPEN_BRACKET_TOKEN,
		st.CLOSE_BRACKET_TOKEN,
		st.OPEN_PAREN_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.DOLLAR_TOKEN,
		st.BITWISE_XOR_TOKEN,
		st.COLON_TOKEN,
		st.BACK_SLASH_TOKEN,
		st.MINUS_TOKEN,
		st.ESCAPED_MINUS_TOKEN,
		st.PIPE_TOKEN,
		st.COMMA_TOKEN:
		return true
	default:
		return false
	}
}

// literalText checks the tokens of a literal node and returns the literal's source text and token kind.
func (n *nodeBuilder) literalText(literal st.Node) (string, st.SyntaxKind, bool) {
	switch literal := literal.(type) {
	case *st.BasicLiteralNode:
		literalToken := literal.LiteralToken()
		if !n.expectNode(literal) || !n.expectToken(literalToken) {
			return "", 0, false
		}
		return literalToken.Text(), literalToken.Kind(), true
	case *st.NilLiteralNode:
		if !n.expectNode(literal) || !n.expectToken(literal.OpenParenToken()) || !n.expectToken(literal.CloseParenToken()) {
			return "", 0, false
		}
		return "", st.NIL_LITERAL, true
	case st.Token:
		if !n.expectSignificantToken(literal) {
			return "", 0, false
		}
		return literal.Text(), literal.Kind(), true
	default:
		n.internalError(fmt.Sprintf("unexpected literal node %T", literal), literal)
		return "", 0, false
	}
}

// createSimpleLiteral creates a simple literal from a node
func (n *nodeBuilder) createSimpleLiteral(literal st.Node) (*ast.BLangLiteral, bool) {
	textValue, literalTokenKind, ok := n.literalText(literal)
	if !ok {
		return nil, false
	}
	return n.createSimpleLiteralFromText(literal, textValue, literalTokenKind)
}

// createSimpleLiteralFromText builds a simple literal from the already-checked text of a literal node.
func (n *nodeBuilder) createSimpleLiteralFromText(literal st.Node, textValue string, literalTokenKind st.SyntaxKind) (*ast.BLangLiteral, bool) {
	kind := literal.Kind()
	var literalKind ast.LiteralKind
	var value any
	var originalValue *string
	var ok bool

	// TODO: Verify all types, only string type tested
	if kind == st.NUMERIC_LITERAL {
		switch literalTokenKind {
		case st.DECIMAL_INTEGER_LITERAL_TOKEN, st.HEX_INTEGER_LITERAL_TOKEN:
			literalKind = ast.LiteralKindInt
			value, ok = n.getIntegerLiteral(literal, literalTokenKind, textValue)
			if !ok {
				return nil, false
			}
			originalValue = &textValue
			// TODO: can we fix below?
			if literalTokenKind == st.HEX_INTEGER_LITERAL_TOKEN && withinByteRange(value) {
				literalKind = ast.LiteralKindByte
			}
		case st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			// TODO: Check effect of mapping negative(-) numbers as unary-expr
			if balCommon.IsDecimalDiscriminated(textValue) {
				literalKind = ast.LiteralKindDecimal
			} else {
				literalKind = ast.LiteralKindFloat
			}
			value = textValue
			originalValue = &textValue
		default:
			// TODO: Check effect of mapping negative(-) numbers as unary-expr
			literalKind = ast.LiteralKindFloat
			value = balCommon.NormalizeHexFloatLiteral(textValue)
			originalValue = &textValue
		}
		numericLiteral := ast.NewBLangNumericLiteral(
			n.getPosition(literal), literalKind, value, *originalValue, false,
		)
		return &numericLiteral.BLangLiteral, true
	} else if kind == st.BOOLEAN_LITERAL {
		literalKind = ast.LiteralKindBoolean
		value = strings.ToLower(textValue) == "true"
		originalValue = &textValue
	} else if kind == st.STRING_LITERAL || kind == st.XML_TEXT_CONTENT ||
		kind == st.TEMPLATE_STRING || kind == st.IDENTIFIER_TOKEN ||
		kind == st.PROMPT_CONTENT || isTokenInRegExp(kind) {
		text := textValue
		if kind == st.STRING_LITERAL {
			if len(text) > 1 && text[len(text)-1] == '"' {
				text = text[1 : len(text)-1]
			} else {
				// Missing end quote case
				text = text[1:]
			}
		}

		const identifierLiteralPrefix = "'"
		if kind == st.IDENTIFIER_TOKEN && strings.HasPrefix(text, identifierLiteralPrefix) {
			text = text[1:]
		}

		if kind != st.TEMPLATE_STRING && kind != st.XML_TEXT_CONTENT &&
			kind != st.PROMPT_CONTENT && !isTokenInRegExp(kind) {
			pos := n.getPosition(literal)
			validateUnicodePoints(text, pos)

			// Try to unescape, but handle errors gracefully
			// We may reach here when the string literal has syntax diagnostics.
			// Therefore mock the compiler with an empty string on error.
			text = unescapeBallerinaString(text)
		}

		literalKind = ast.LiteralKindString
		value = text
		originalValue = &textValue
	} else if kind == st.NIL_LITERAL {
		literalKind = ast.LiteralKindNil
		value = nil
		originalValue = new(string(model.NIL_VALUE))
	} else if kind == st.NULL_LITERAL {
		originalValue = new("null")
		literalKind = ast.LiteralKindNil
	} else if kind == st.BINARY_EXPRESSION { // Should be base16 and base64
		literalKind = ast.LiteralKindByteArray
		value = textValue
		originalValue = &textValue
	} else {
		n.internalError(fmt.Sprintf("unexpected simple literal kind: %v", kind), literal)
		return nil, false
	}
	return ast.NewBLangLiteral(n.getPosition(literal), literalKind, value, *originalValue, false), true
}

func (n *nodeBuilder) transformModulePart(modulePartNode *st.ModulePart) *ast.BLangCompilationUnit {
	compilationUnit := ast.BLangCompilationUnit{}
	n.currentCompUnit = &compilationUnit
	defer func() { n.currentCompUnit = nil }()
	compilationUnit.SetPackageID(n.PackageID)
	pos := n.getPosition(modulePartNode)

	// The module part is above every recovery point, so its own diagnostics are reported and building continues.
	n.expectNode(modulePartNode)

	imports := modulePartNode.Imports()
	for importDecl := range imports.Iterator() {
		compilationUnit.AddTopLevelNode(n.transformTopLevel(importDecl))
	}

	members := modulePartNode.Members()
	for member := range members.Iterator() {
		compilationUnit.AddTopLevelNode(n.transformTopLevel(member))
	}

	if eofToken := modulePartNode.EofToken(); !n.expectToken(eofToken) {
		if eofToken == nil {
			compilationUnit.AddTopLevelNode(ast.NewBLangBadTopLevelNode(diagnostics.NewBuiltinLocation(), ast.BadTopLevelNodeMissing, nil))
		} else {
			compilationUnit.AddTopLevelNode(n.badTopLevel(eofToken))
		}
	}

	fileName := ""
	if !diagnostics.IsLocationEmpty(pos) {
		fileName = n.de().FileName(pos)
	}

	newLocation := diagnostics.NewLocation(n.de(), fileName, 0, 0)
	compilationUnit.SetPosition(newLocation)
	compilationUnit.SetPackageID(n.PackageID)

	return &compilationUnit
}

func (n *nodeBuilder) functionQualifierFlags(qualifierList st.NodeList[st.Token]) (model.Flag, bool) {
	var flags model.Flag
	for qualifier := range qualifierList.Iterator() {
		if !n.expectSignificantToken(qualifier) {
			return 0, false
		}
		switch qualifier.Kind() {
		case st.PUBLIC_KEYWORD:
			flags |= model.FlagPublic
		case st.REMOTE_KEYWORD:
			flags |= model.FlagRemote
		case st.TRANSACTIONAL_KEYWORD:
			flags |= model.FlagTransactional
		case st.RESOURCE_KEYWORD:
			flags |= model.FlagResource
		case st.ISOLATED_KEYWORD:
			flags |= model.FlagIsolated
		case st.PRIVATE_KEYWORD:
			flags |= model.FlagPrivate
		default:
			n.internalError("unexpected function qualifier", qualifier)
			return 0, false
		}
	}
	return flags, true
}

func (n *nodeBuilder) populateFuncSignature(data *ast.InvokableData, funcSignature *st.FunctionSignatureNode) bool {
	if !n.expectNode(funcSignature) {
		return false
	}
	data.ParamListPosition = diagnostics.NewBuiltinLocation()
	openParen := funcSignature.OpenParenToken()
	closeParen := funcSignature.CloseParenToken()
	if !n.expectToken(openParen) {
		return false
	}
	parameters := funcSignature.Parameters()
	for param, separator := range parameters.SeparatedElements() {
		paramNode, ok := n.transformParameter(param)
		if !ok || !n.expectOptionalToken(separator) {
			return false
		}
		if _, isRestParam := param.(*st.RestParameterNode); isRestParam {
			data.RestParam = paramNode
		} else {
			data.RequiredParams = append(data.RequiredParams, *paramNode)
		}
	}
	if !n.expectToken(closeParen) {
		return false
	}
	data.ParamListPosition = n.getPositionRange(openParen, closeParen)

	retTypeDescNode := funcSignature.ReturnTypeDesc()
	if retTypeDescNode == nil {
		nilReturnType := &ast.BLangValueType{TypeKind: ast.TypeKindNil}
		nilReturnType.SetPosition(diagnostics.NewBuiltinLocation())
		data.ReturnTypeDescriptor = nilReturnType
		return true
	}
	if !n.expectNode(retTypeDescNode) {
		return false
	}
	returnsKeyword := retTypeDescNode.ReturnsKeyword()
	if !n.expectToken(returnsKeyword) {
		return false
	}
	data.Flags |= model.FlagExplicitReturnTypeDescriptor
	annotations, ok := n.transformAnnotations(retTypeDescNode.Annotations())
	if !ok {
		return false
	}
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, "return")
	data.ReturnTypeDescriptor = n.transformType(retTypeDescNode.Type())
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	data.ReturnAnnotationAttachments = append(data.ReturnAnnotationAttachments, annotations...)
	return true
}

func (n *nodeBuilder) transformFunctionDefinition(funcDefNode *st.FunctionDefinition) (*ast.BLangFunction, bool) {
	if !n.expectNode(funcDefNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(funcDefNode.Metadata())
	if !ok {
		return nil, false
	}
	bLFunction, ok := n.createFunctionNode(funcDefNode, false)
	if !ok {
		return nil, false
	}
	bLFunction.SetPosition(n.getPositionWithoutMetadata(funcDefNode))
	applyMetadata(metadata, bLFunction)
	return bLFunction, true
}

// createFunctionNode builds a function from its definition; the names of attached remote methods are mangled
// with model.RemoteMethodName.
func (n *nodeBuilder) createFunctionNode(funcDefNode *st.FunctionDefinition, attached bool) (*ast.BLangFunction, bool) {
	flags, ok := n.functionQualifierFlags(funcDefNode.QualifierList())
	if !ok || !n.expectToken(funcDefNode.FunctionKeyword()) {
		return nil, false
	}
	nameToken := asToken(funcDefNode.FunctionName())
	var mapName func(string) string
	if attached && flags&model.FlagRemote != 0 {
		mapName = remoteMethodName
	}
	name := n.transformIdentifierWith(nameToken, mapName, underscoreKeywordMessage)
	if relativeResourcePath := funcDefNode.RelativeResourcePath(); relativeResourcePath.Size() > 0 {
		n.unimplemented("transformFunctionDefinition: resource functions not yet supported", funcDefNode)
		return nil, false
	}
	data := ast.InvokableData{Name: name, Flags: flags}
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, identifierTokenValue(nameToken))
	defer func() {
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	}()
	if !n.populateFuncSignature(&data, funcDefNode.FunctionSignature()) {
		return nil, false
	}
	if !n.populateFunctionBody(&data, funcDefNode.FunctionBody()) {
		return nil, false
	}
	return ast.NewBLangFunction(data), true
}

func (n *nodeBuilder) populateFunctionBody(data *ast.InvokableData, funcBody st.FunctionBodyNode) bool {
	if funcBody == nil {
		data.Flags |= model.FlagInterface
		return true
	}
	body, ok := n.transformFunctionBody(funcBody)
	if !ok {
		return false
	}
	data.Body = body
	if funcBody.Kind() == st.EXTERNAL_FUNCTION_BODY {
		data.Flags |= model.FlagNative
	}
	return true
}

func (n *nodeBuilder) transformFunctionBody(funcBody st.FunctionBodyNode) (ast.FunctionBodyNode, bool) {
	switch funcBody := funcBody.(type) {
	case *st.FunctionBodyBlockNode:
		return n.transformFunctionBodyBlock(funcBody)
	case *st.ExternalFunctionBodyNode:
		return n.transformExternalFunctionBody(funcBody)
	case *st.ExpressionFunctionBodyNode:
		return n.transformExpressionFunctionBody(funcBody)
	default:
		n.internalError(fmt.Sprintf("transformFunctionBody: unsupported node type %T", funcBody), funcBody)
		return nil, false
	}
}

func (n *nodeBuilder) transformParameter(param st.Node) (*ast.BLangVariable, bool) {
	paramNode, parts, ok := paramPartsOf(param)
	if !ok {
		n.internalError(fmt.Sprintf("transformParameter: unsupported node type %T", param), param)
		return nil, false
	}
	return n.createParameter(paramNode, parts)
}

func (n *nodeBuilder) transformImportDeclaration(importDeclarationNode *st.ImportDeclarationNode) (*ast.BLangImportPackage, bool) {
	if !n.expectNode(importDeclarationNode) || !n.expectToken(importDeclarationNode.ImportKeyword()) {
		return nil, false
	}
	orgNameNode := importDeclarationNode.OrgName()
	var orgNameToken st.Token
	if orgNameNode != nil {
		orgNameToken = orgNameNode.OrgName()
		if !n.expectNode(orgNameNode) || !n.expectToken(orgNameToken) || !n.expectToken(orgNameNode.SlashToken()) {
			return nil, false
		}
	}

	position := n.getPosition(importDeclarationNode)

	var pkgNameComps []ast.BLangIdentifier
	moduleNames := importDeclarationNode.ModuleName()
	for name, separator := range moduleNames.SeparatedElements() {
		if !n.expectToken(name) {
			return nil, false
		}
		nameText := name.Text()
		pkgNameComps = append(pkgNameComps, createIdentifier(n.getPosition(name), &nameText, &nameText))
		if !n.expectOptionalToken(separator) {
			return nil, false
		}
	}

	importDcl := &ast.BLangImportPackage{}
	importDcl.SetPosition(position)
	importDcl.PkgNameComps = pkgNameComps

	var orgNamePos diagnostics.Location
	if orgNameNode != nil {
		orgNamePos = n.getPosition(orgNameNode)
	}
	var orgNameStr *string
	if orgNameToken != nil {
		text := orgNameToken.Text()
		orgNameStr = &text
	}
	orgIdentifier := createIdentifier(orgNamePos, orgNameStr, orgNameStr)
	importDcl.OrgName = &orgIdentifier

	emptyVersion := createIdentifier(diagnostics.NewBuiltinLocation(), nil, nil)
	importDcl.Version = &emptyVersion

	prefixNode := importDeclarationNode.Prefix()
	if prefixNode == nil {
		if len(pkgNameComps) == 0 {
			n.internalError("import declaration has no module name", importDeclarationNode)
			return nil, false
		}
		lastPkgComp := &pkgNameComps[len(pkgNameComps)-1]
		importDcl.Alias = lastPkgComp
	} else {
		prefix := prefixNode.Prefix()
		if !n.expectNode(prefixNode) || !n.expectToken(prefixNode.AsKeyword()) || !n.expectToken(prefix) {
			return nil, false
		}
		if prefix.Kind() == st.UNDERSCORE_KEYWORD {
			aliasIdent := n.createIgnoreIdentifier(prefix)
			importDcl.Alias = &aliasIdent
		} else {
			prefixText := prefix.Text()
			aliasIdent := createIdentifier(n.getPosition(prefix), &prefixText, &prefixText)
			importDcl.Alias = &aliasIdent
		}
	}

	if !n.expectToken(importDeclarationNode.Semicolon()) {
		return nil, false
	}
	return importDcl, true
}

func publicFlag(visibilityQualifier st.Token) model.Flag {
	if visibilityQualifier != nil && visibilityQualifier.Kind() == st.PUBLIC_KEYWORD {
		return model.FlagPublic
	}
	return 0
}

func (n *nodeBuilder) transformListenerDeclaration(listenerDeclarationNode *st.ListenerDeclarationNode) (*ast.BLangVariable, bool) {
	if !n.expectNode(listenerDeclarationNode) {
		return nil, false
	}
	metadataNode := listenerDeclarationNode.Metadata()
	metadata, ok := n.transformMetadata(metadataNode)
	if !ok {
		return nil, false
	}
	if len(metadata.annotations) > 0 {
		annotations := metadataNode.Annotations()
		n.unimplemented("transformListenerDeclaration: annotations not yet supported", annotations.Get(0))
		return nil, false
	}
	visQual := listenerDeclarationNode.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(visQual) || !n.expectToken(listenerDeclarationNode.ListenerKeyword()) {
		return nil, false
	}

	pos := n.getPositionWithoutMetadata(listenerDeclarationNode)
	typeDesc := listenerDeclarationNode.TypeDescriptor()
	isDeclaredWithVar := typeDesc == nil
	var typeNode ast.BType
	if !isDeclaredWithVar {
		typeNode = n.transformType(typeDesc)
	}
	identifier := n.transformIdentifier(listenerDeclarationNode.VariableName())
	if !n.expectToken(listenerDeclarationNode.EqualsToken()) {
		return nil, false
	}
	initializer := listenerDeclarationNode.Initializer()
	if initializer == nil {
		n.internalError("listener declaration has no initializer", listenerDeclarationNode)
		return nil, false
	}
	expr := n.transformExpr(initializer)
	if !n.expectToken(listenerDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	flags := model.FlagFinal | model.FlagListener | publicFlag(visQual)
	bLSimpleVar := ast.NewBLangVariable(pos, identifier, typeNode, expr, isDeclaredWithVar, flags)
	bLSimpleVar.SetPosition(pos)
	bLSimpleVar.MarkdownDocumentationAttachment = metadata.documentation
	return bLSimpleVar, true
}

func isAllowedDistinctTypeDescriptor(kind st.SyntaxKind) bool {
	switch kind {
	case st.OBJECT_TYPE_DESC, st.ERROR_TYPE_DESC, st.SIMPLE_NAME_REFERENCE, st.QUALIFIED_NAME_REFERENCE, st.IDENTIFIER_TOKEN:
		return true
	default:
		return false
	}
}

func (n *nodeBuilder) transformTypeDefinition(typeDefinitionNode *st.TypeDefinitionNode) (*ast.BLangTypeDefinition, bool) {
	if !n.expectNode(typeDefinitionNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(typeDefinitionNode.Metadata())
	if !ok {
		return nil, false
	}
	visibility := typeDefinitionNode.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(visibility) || !n.expectToken(typeDefinitionNode.TypeKeyword()) {
		return nil, false
	}
	identifierNode := n.transformIdentifier(typeDefinitionNode.TypeName())
	flags := publicFlag(visibility)
	typeDescriptorNode := typeDefinitionNode.TypeDescriptor()
	distinctTypeDescriptorNode, isDistinct := typeDescriptorNode.(*st.DistinctTypeDescriptorNode)
	if isDistinct {
		flags |= model.FlagDistinct
	}
	typeDef := ast.NewBLangTypeDefinitionWithData(n.getPositionWithoutMetadata(typeDefinitionNode), identifierNode, ast.TypeData{}, metadata.documentation, flags)
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, typeDef.Name.GetValue())
	typeData, ok := n.transformTypeDefinitionDescriptor(typeDescriptorNode, distinctTypeDescriptorNode)
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	if !ok || !n.expectToken(typeDefinitionNode.SemicolonToken()) {
		return nil, false
	}
	typeDef.SetTypeData(typeData)
	typeDef.SetPosition(n.getPositionWithoutMetadata(typeDefinitionNode))
	addAnnotationAttachments(metadata.annotations, typeDef)
	return typeDef, true
}

func (n *nodeBuilder) transformTypeDefinitionDescriptor(typeDescriptorNode st.Node, distinctTypeDescriptorNode *st.DistinctTypeDescriptorNode) (ast.TypeData, bool) {
	if distinctTypeDescriptorNode == nil {
		return ast.TypeData{TypeDescriptor: n.transformType(typeDescriptorNode)}, true
	}
	if !n.expectNode(distinctTypeDescriptorNode) || !n.expectToken(distinctTypeDescriptorNode.DistinctKeyword()) {
		return ast.TypeData{}, false
	}
	innerTypeDescriptorNode := distinctTypeDescriptorNode.TypeDescriptor()
	if innerTypeDescriptorNode == nil || !isAllowedDistinctTypeDescriptor(innerTypeDescriptorNode.Kind()) {
		if innerTypeDescriptorNode != nil {
			n.transformType(innerTypeDescriptorNode)
		}
		n.cx.SyntaxError("only object and error types can be distinct", n.getPosition(distinctTypeDescriptorNode))
		neverType := &ast.BLangValueType{TypeKind: ast.TypeKindNever}
		neverType.SetPosition(n.getPosition(distinctTypeDescriptorNode))
		return ast.TypeData{TypeDescriptor: neverType}, true
	}
	return ast.TypeData{TypeDescriptor: n.transformType(innerTypeDescriptorNode)}, true
}

func (n *nodeBuilder) transformServiceDeclaration(serviceDeclarationNode *st.ServiceDeclarationNode) (*ast.BLangService, bool) {
	if !n.expectNode(serviceDeclarationNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(serviceDeclarationNode.Metadata())
	if !ok {
		return nil, false
	}
	flags, ok := n.serviceQualifierFlags(serviceDeclarationNode)
	if !ok || !n.expectToken(serviceDeclarationNode.ServiceKeyword()) {
		return nil, false
	}
	service := ast.NewBLangServiceWithFlags(flags)
	service.SetPosition(n.getPositionWithoutMetadata(serviceDeclarationNode))
	applyMetadata(metadata, &service)

	if typeDesc := serviceDeclarationNode.TypeDescriptor(); typeDesc != nil {
		service.SetTypeData(ast.TypeData{TypeDescriptor: n.transformType(typeDesc)})
	}

	if !n.populateServiceAttachPoint(&service, serviceDeclarationNode) ||
		!n.expectToken(serviceDeclarationNode.OnKeyword()) ||
		!n.populateServiceAttachedExprs(&service, serviceDeclarationNode) ||
		!n.expectToken(serviceDeclarationNode.OpenBraceToken()) {
		return nil, false
	}

	members := n.collectClassDefnMembers(serviceDeclarationNode.Members())
	if !n.expectToken(serviceDeclarationNode.CloseBraceToken()) ||
		!n.expectOptionalToken(serviceDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	service.Fields = members.Fields
	service.Methods = members.Methods
	service.InitFunction = members.InitFunction
	service.ResourceMethods = members.ResourceMethods
	service.BadTopLevelNodes = members.BadTopLevelNodes
	if len(members.UnresolvedInclusions) > 0 {
		// Parser should catch these
		n.cx.InternalError("unexpected inclusions in service decl", members.UnresolvedInclusions[0].GetPosition())
		return nil, false
	}

	return &service, true
}

// serviceQualifierFlags reads the user-controllable qualifiers from the
// service declaration. The `service` flag is already set by NewBLangService.
func (n *nodeBuilder) serviceQualifierFlags(node *st.ServiceDeclarationNode) (model.Flag, bool) {
	var flags model.Flag
	quals := node.Qualifiers()
	for qual := range quals.Iterator() {
		if !n.expectSignificantToken(qual) {
			return 0, false
		}
		if qual.Kind() == st.ISOLATED_KEYWORD {
			flags |= model.FlagIsolated
		}
	}
	return flags, true
}

func (n *nodeBuilder) populateServiceAttachPoint(service *ast.BLangService, node *st.ServiceDeclarationNode) bool {
	paths := node.AbsoluteResourcePath()
	if paths.Size() > 0 {
		service.AbsoluteResourcePath = []ast.BLangIdentifier{}
	}
	for seg := range paths.Iterator() {
		if seg.Kind() == st.STRING_LITERAL {
			literal, ok := n.createSimpleLiteral(seg)
			if !ok {
				return false
			}
			service.AttachPointLiteral = literal
			continue
		}
		tok, ok := seg.(st.Token)
		if !ok {
			n.cx.InternalError("unexpected node in service attach point", n.getPosition(seg))
			return false
		}
		if !n.expectSignificantToken(tok) {
			return false
		}
		switch tok.Kind() {
		case st.IDENTIFIER_TOKEN:
			ident, ok := n.createBLangIdentifier(tok)
			if !ok {
				return false
			}
			service.AbsoluteResourcePath = append(service.AbsoluteResourcePath, ident)
		case st.SLASH_TOKEN:
			// Slash tokens between segments are ignored.
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected token in service attach point: %v", tok.Kind()), n.getPosition(tok))
			return false
		}
	}
	return true
}

func (n *nodeBuilder) populateServiceAttachedExprs(service *ast.BLangService, node *st.ServiceDeclarationNode) bool {
	exprs := node.Expressions()
	if exprs.Size() > 0 {
		service.AttachedExprsPosition = n.getPositionRange(exprs.Get(0), exprs.Get(exprs.Size()-1))
	}
	for expr, separator := range exprs.SeparatedElements() {
		service.AttachedExprs = append(service.AttachedExprs, n.transformExpr(expr))
		if !n.expectOptionalToken(separator) {
			return false
		}
	}
	return true
}

type classDefnMembers struct {
	Fields               []*ast.BLangVariable
	Methods              map[string]*ast.BLangFunction
	InitFunction         *ast.BLangFunction
	ResourceMethods      []*ast.BLangResourceMethod
	UnresolvedInclusions []*ast.BLangUserDefinedType
	BadTopLevelNodes     []*ast.BLangBadTopLevelNode
}

func newClassDefnMembers() classDefnMembers {
	return classDefnMembers{Methods: map[string]*ast.BLangFunction{}}
}

// collectClassDefnMembers is the member recovery point: a member that fails to build is replaced by a bad member.
func (n *nodeBuilder) collectClassDefnMembers(memberNodes st.NodeList[st.Node]) classDefnMembers {
	members := newClassDefnMembers()
	for member := range memberNodes.Iterator() {
		if !n.addCollectedMember(&members, member) {
			members.BadTopLevelNodes = append(members.BadTopLevelNodes, n.badTopLevel(member))
		}
	}
	return members
}

func (n *nodeBuilder) addCollectedMember(members *classDefnMembers, member st.Node) bool {
	switch member := member.(type) {
	case *st.ObjectFieldNode:
		field, ok := n.transformClassField(member)
		if !ok {
			return false
		}
		members.Fields = append(members.Fields, field)
		return true
	case *st.FunctionDefinition:
		if member.Kind() == st.RESOURCE_ACCESSOR_DEFINITION {
			return n.addCollectedResourceMethod(members, member)
		}
		return n.addCollectedMethod(members, member)
	case *st.TypeReferenceNode:
		inclusion, ok := n.transformTypeReference(member)
		if !ok {
			return false
		}
		members.UnresolvedInclusions = append(members.UnresolvedInclusions, inclusion)
		return true
	default:
		n.rejectNode(member, "collectClassDefnMembers: unsupported member kind")
		return false
	}
}

func (n *nodeBuilder) transformTypeReference(typeReferenceNode *st.TypeReferenceNode) (*ast.BLangUserDefinedType, bool) {
	if !n.expectNode(typeReferenceNode) || !n.expectToken(typeReferenceNode.AsteriskToken()) {
		return nil, false
	}
	typeRef, ok := n.transformTypeReferenceName(typeReferenceNode.TypeName())
	if !ok || !n.expectToken(typeReferenceNode.SemicolonToken()) {
		return nil, false
	}
	return typeRef, true
}

func (n *nodeBuilder) addCollectedResourceMethod(members *classDefnMembers, funcDef *st.FunctionDefinition) bool {
	rm, ok := n.createResourceMethodNode(funcDef)
	if !ok {
		return false
	}
	members.ResourceMethods = append(members.ResourceMethods, rm)
	return true
}

func (n *nodeBuilder) addCollectedMethod(members *classDefnMembers, funcDef *st.FunctionDefinition) bool {
	if !n.expectNode(funcDef) {
		return false
	}
	metadata, ok := n.transformMetadata(funcDef.Metadata())
	if !ok {
		return false
	}
	bLFunction, ok := n.createFunctionNode(funcDef, true)
	if !ok {
		return false
	}
	bLFunction.SetPosition(n.getPositionWithoutMetadata(funcDef))
	bLFunction.SetAttached()
	applyMetadata(metadata, bLFunction)

	funcName := bLFunction.Name.GetValue()
	if model.Name(funcName) == model.USER_DEFINED_INIT_SUFFIX {
		if members.InitFunction != nil {
			n.cx.SyntaxError("redeclared symbol 'init'", bLFunction.GetPosition())
			return true
		}
		members.InitFunction = bLFunction
		return true
	}
	if _, exists := members.Methods[funcName]; exists {
		n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(funcName)+"'", bLFunction.GetPosition())
		return true
	}
	members.Methods[funcName] = bLFunction
	return true
}

func (n *nodeBuilder) transformConstantDeclaration(constantDeclarationNode *st.ConstantDeclarationNode) (*ast.BLangVariable, bool) {
	if !n.expectNode(constantDeclarationNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(constantDeclarationNode.Metadata())
	if !ok {
		return nil, false
	}
	visibility := constantDeclarationNode.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(visibility) || !n.expectToken(constantDeclarationNode.ConstKeyword()) {
		return nil, false
	}
	var typeNode ast.BType
	if typeDescriptor := constantDeclarationNode.TypeDescriptor(); typeDescriptor != nil {
		typeNode = n.transformType(typeDescriptor)
	}
	nameIdentifier := n.transformIdentifier(constantDeclarationNode.VariableName())
	if !n.expectToken(constantDeclarationNode.EqualsToken()) {
		return nil, false
	}
	initializer := n.transformExpr(constantDeclarationNode.Initializer())
	if !n.expectToken(constantDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	constantNode := ast.NewBLangVariable(
		n.getPositionWithoutMetadata(constantDeclarationNode),
		nameIdentifier,
		typeNode,
		initializer,
		false,
		publicFlag(visibility)|model.FlagConstant,
	)
	constantNode.SetPosition(n.getPositionWithoutMetadata(constantDeclarationNode))
	applyMetadata(metadata, constantNode)
	return constantNode, true
}

type paramParts struct {
	annotations st.NodeList[*st.AnnotationNode]
	asterisk    st.Token
	typeName    st.Node
	ellipsis    st.Token
	paramName   st.Token
	equals      st.Token
	defaultExpr st.Node
	flags       model.Flag
	kind        st.SyntaxKind
}

// expectParamTokenBefore checks the `*` of an included record parameter. The parser leaves the slot empty when
// it could not parse one, and then reports the error inside the parameter's type.
func (n *nodeBuilder) expectParamTokenBefore(parts paramParts) bool {
	return parts.kind != st.INCLUDED_RECORD_PARAM || n.expectOptionalSignificantToken(parts.asterisk)
}

// expectParamTokenAfter checks the token that follows the type of a rest parameter.
func (n *nodeBuilder) expectParamTokenAfter(parts paramParts) bool {
	return parts.kind != st.REST_PARAM || n.expectToken(parts.ellipsis)
}

// transformParamDefault checks the `=` of a defaultable parameter and transforms its default expression.
func (n *nodeBuilder) transformParamDefault(paramNode st.Node, parts paramParts) (ast.BLangExpression, bool) {
	if parts.kind != st.DEFAULTABLE_PARAM {
		return nil, true
	}
	if !n.expectToken(parts.equals) {
		return nil, false
	}
	if parts.defaultExpr == nil {
		n.internalError("defaultable parameter has no default expression", paramNode)
		return nil, false
	}
	return n.transformExpr(parts.defaultExpr), true
}

func (n *nodeBuilder) createParameter(paramNode st.NonTerminalNode, parts paramParts) (*ast.BLangVariable, bool) {
	if !n.expectNode(paramNode) {
		return nil, false
	}
	attachments, ok := n.transformAnnotations(parts.annotations)
	if !ok || !n.expectParamTokenBefore(parts) {
		return nil, false
	}
	paramName := parts.paramName
	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
		defer func() {
			n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
		}()
	}
	declaredWithVar := isDeclaredWithVar(parts.typeName)
	var typeNode ast.BType
	if !declaredWithVar {
		typeNode = n.transformType(parts.typeName)
	} else if !n.transformVarTypeKeyword(parts.typeName) {
		return nil, false
	}
	if !n.expectParamTokenAfter(parts) {
		return nil, false
	}
	identifier := n.transformIdentifier(paramName)
	defaultExpr, ok := n.transformParamDefault(paramNode, parts)
	if !ok {
		return nil, false
	}
	var expr ast.BLangActionOrExpression
	if defaultExpr != nil {
		expr = defaultExpr
	}
	var namePos diagnostics.Location
	if paramName != nil {
		namePos = n.getPosition(paramName)
	}
	simpleVar := ast.NewBLangVariable(namePos, identifier, typeNode, expr, declaredWithVar, parts.flags)
	addAnnotationAttachments(attachments, simpleVar)
	simpleVar.SetPosition(n.getPosition(paramNode))
	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		// Param doesn't have a name and also is not a missing node
		// Therefore, assigning the built-in location
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}
	return simpleVar, true
}

func (n *nodeBuilder) transformModuleVariableDeclaration(moduleVariableDeclarationNode *st.ModuleVariableDeclarationNode) (*ast.BLangVariable, bool) {
	if !n.expectNode(moduleVariableDeclarationNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(moduleVariableDeclarationNode.Metadata())
	if !ok {
		return nil, false
	}
	pos := n.getPositionWithoutMetadata(moduleVariableDeclarationNode)
	flags, ok := n.moduleVariableFlags(moduleVariableDeclarationNode, pos)
	if !ok {
		return nil, false
	}
	typedBindingPattern := moduleVariableDeclarationNode.TypedBindingPattern()
	if !n.expectNode(typedBindingPattern) {
		return nil, false
	}
	typeDesc := typedBindingPattern.TypeDescriptor()
	isDeclaredWithVar := typeDesc != nil && isDeclaredWithVar(typeDesc)
	var typeNode ast.BType
	if typeDesc != nil && !isDeclaredWithVar {
		typeNode = n.transformType(typeDesc)
	} else if !n.transformVarTypeKeyword(typeDesc) {
		return nil, false
	}
	nameNode, ok := n.getBLangVariableNode(typedBindingPattern.BindingPattern(), pos)
	if !ok || !n.expectOptionalToken(moduleVariableDeclarationNode.EqualsToken()) {
		return nil, false
	}
	var expr ast.BLangActionOrExpression
	if initializer := moduleVariableDeclarationNode.Initializer(); initializer != nil {
		expr = n.transformExpr(initializer)
	}
	if !n.expectToken(moduleVariableDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	simpleVar := ast.NewBLangVariable(pos, nameNode.Name, typeNode, expr, isDeclaredWithVar, flags)
	simpleVar.SetPosition(pos)

	if simpleVar.IsDeclaredWithVar && simpleVar.TypeNode() == nil && simpleVar.Expr == nil {
		n.cx.SyntaxError("var-declared module variable must have an initializer expression for type inference", pos)
		return simpleVar, true
	}
	applyMetadata(metadata, simpleVar)
	return simpleVar, true
}

func (n *nodeBuilder) moduleVariableFlags(node *st.ModuleVariableDeclarationNode, pos diagnostics.Location) (model.Flag, bool) {
	visibility := node.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(visibility) {
		return 0, false
	}
	flags := publicFlag(visibility)
	qualifiers := node.Qualifiers()
	for qualifier := range qualifiers.Iterator() {
		if !n.expectSignificantToken(qualifier) {
			return 0, false
		}
		switch qualifier.Kind() {
		case st.FINAL_KEYWORD:
			flags |= model.FlagFinal
		case st.ISOLATED_KEYWORD:
			flags |= model.FlagIsolated
		case st.CONFIGURABLE_KEYWORD:
			n.cx.Unimplemented("configurable module variables are not supported yet", pos)
			return 0, false
		default:
			n.internalError("unexpected module variable qualifier", qualifier)
			return 0, false
		}
	}
	return flags, true
}

func (n *nodeBuilder) transformAnnotationDeclaration(annotationDeclarationNode *st.AnnotationDeclarationNode) (*ast.BLangAnnotation, bool) {
	if !n.expectNode(annotationDeclarationNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(annotationDeclarationNode.Metadata())
	if !ok {
		return nil, false
	}
	visibility := annotationDeclarationNode.VisibilityQualifier()
	constKeyword := annotationDeclarationNode.ConstKeyword()
	if !n.expectOptionalSignificantToken(visibility) || !n.expectOptionalSignificantToken(constKeyword) ||
		!n.expectToken(annotationDeclarationNode.AnnotationKeyword()) {
		return nil, false
	}
	flags := publicFlag(visibility)
	if constKeyword != nil {
		flags |= model.FlagConstant
	}
	var typeDescriptor ast.TypeDescriptor
	if typeDesc := annotationDeclarationNode.TypeDescriptor(); typeDesc != nil {
		typeDescriptor = n.transformType(typeDesc)
	}
	name := n.transformIdentifier(annotationDeclarationNode.AnnotationTag())
	if !n.expectOptionalToken(annotationDeclarationNode.OnKeyword()) {
		return nil, false
	}
	annotation := ast.NewBLangAnnotation(
		n.getPositionWithoutMetadata(annotationDeclarationNode),
		name,
		typeDescriptor,
		flags,
	)
	attachPoints := annotationDeclarationNode.AttachPoints()
	for attachPoint, separator := range attachPoints.SeparatedElements() {
		attachPointNode, isAttachPoint := attachPoint.(*st.AnnotationAttachPointNode)
		if !isAttachPoint {
			n.internalError(fmt.Sprintf("unexpected annotation attach point node %T", attachPoint), attachPoint)
			return nil, false
		}
		point, ok := n.createAnnotationAttachPoint(attachPointNode)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		annotation.AddAttachPoint(point)
	}
	if !n.expectToken(annotationDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	applyMetadata(metadata, annotation)
	return annotation, true
}

func (n *nodeBuilder) createAnnotationAttachPoint(annotationAttachPointNode *st.AnnotationAttachPointNode) (ast.AttachPoint, bool) {
	sourceKeyword := annotationAttachPointNode.SourceKeyword()
	if !n.expectNode(annotationAttachPointNode) || !n.expectOptionalSignificantToken(sourceKeyword) {
		return ast.AttachPoint{}, false
	}
	parts := []string{}
	identifiers := annotationAttachPointNode.Identifiers()
	for identifier := range identifiers.Iterator() {
		if !n.expectToken(identifier) {
			return ast.AttachPoint{}, false
		}
		parts = append(parts, identifier.Text())
	}
	point, ok := annotationAttachPointFromParts(parts)
	if !ok {
		n.cx.SyntaxError("unknown annotation attach point '"+strings.Join(parts, " ")+"'", n.getPosition(annotationAttachPointNode))
	}
	return ast.AttachPoint{
		Point:  point,
		Source: sourceKeyword != nil,
	}, true
}

// annotationAttachPointFromParts maps the space-separated source spelling of an
// annotation attach point to its Point. This is the inverse of Point.String(),
// but keyed on the spelled-out source form (e.g. "object function"), which
// differs from the canonical key (e.g. "objectfunction").
func annotationAttachPointFromParts(parts []string) (ast.Point, bool) {
	switch strings.Join(parts, " ") {
	case "type":
		return ast.PointType, true
	case "object":
		return ast.PointObject, true
	case "function":
		return ast.PointFunction, true
	case "object function":
		return ast.PointObjectMethod, true
	case "service remote function":
		return ast.PointServiceRemote, true
	case "parameter":
		return ast.PointParameter, true
	case "return":
		return ast.PointReturn, true
	case "service":
		return ast.PointService, true
	case "field":
		return ast.PointField, true
	case "object field":
		return ast.PointObjectField, true
	case "record field":
		return ast.PointRecordField, true
	case "listener":
		return ast.PointListener, true
	case "annotation":
		return ast.PointAnnotation, true
	case "external":
		return ast.PointExternal, true
	case "var":
		return ast.PointVar, true
	case "const":
		return ast.PointConst, true
	case "worker":
		return ast.PointWorker, true
	case "class":
		return ast.PointClass, true
	default:
		return 0, false
	}
}

type xmlNamespaceDeclarationNode interface {
	st.NonTerminalNode
	XmlnsKeyword() st.Token
	Namespaceuri() st.ExpressionNode
	AsKeyword() st.Token
	NamespacePrefix() *st.IdentifierToken
	SemicolonToken() st.Token
}

func (n *nodeBuilder) transformXMLNamespaceDeclarationNode(node xmlNamespaceDeclarationNode) (*ast.BLangXMLNS, bool) {
	if !n.expectNode(node) || !n.expectToken(node.XmlnsKeyword()) {
		return nil, false
	}
	uriNode := node.Namespaceuri()
	if uriNode == nil {
		n.internalError("xml namespace declaration has no namespace uri", node)
		return nil, false
	}
	namespaceURI := n.transformExpr(uriNode)
	if !n.expectOptionalToken(node.AsKeyword()) {
		return nil, false
	}
	var prefix *ast.BLangIdentifier
	if prefixTok := asToken(node.NamespacePrefix()); prefixTok != nil {
		identifier, ok := n.createBLangIdentifier(prefixTok)
		if !ok {
			return nil, false
		}
		prefix = &identifier
	}
	if !n.expectToken(node.SemicolonToken()) {
		return nil, false
	}
	return ast.NewBLangXMLNS(n.getPosition(node), namespaceURI, prefix), true
}

func (n *nodeBuilder) transformEnumDeclaration(enumDeclarationNode *st.EnumDeclarationNode) (*ast.BLangTypeDefinition, bool) {
	if !n.expectNode(enumDeclarationNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(enumDeclarationNode.Metadata())
	if !ok {
		return nil, false
	}
	qualifier := enumDeclarationNode.Qualifier()
	if !n.expectOptionalSignificantToken(qualifier) || !n.expectToken(enumDeclarationNode.EnumKeywordToken()) {
		return nil, false
	}
	publicQualifier := qualifier != nil && qualifier.Kind() == st.PUBLIC_KEYWORD
	identifier := n.transformIdentifier(asToken(enumDeclarationNode.Identifier()))
	if !n.expectToken(enumDeclarationNode.OpenBraceToken()) {
		return nil, false
	}

	memberNodes := enumDeclarationNode.EnumMemberList()
	memberTypeNodes := make([]ast.TypeDescriptor, 0)
	var constants []*ast.BLangVariable
	for memberNode, separator := range memberNodes.SeparatedElements() {
		enumMember, isEnumMember := memberNode.(*st.EnumMemberNode)
		if !isEnumMember {
			n.internalError(fmt.Sprintf("unexpected enum member node %T", memberNode), memberNode)
			return nil, false
		}
		constantNode, memberType, ok := n.transformEnumMember(enumMember, publicQualifier)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		constants = append(constants, constantNode)
		memberTypeNodes = append(memberTypeNodes, memberType)
	}
	if !n.expectToken(enumDeclarationNode.CloseBraceToken()) || !n.expectOptionalToken(enumDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	if n.currentCompUnit == nil {
		n.cx.InternalError("enum constants can only be added at module level", n.getPosition(enumDeclarationNode))
		return nil, false
	}
	for _, constantNode := range constants {
		n.currentCompUnit.AddTopLevelNode(constantNode)
	}

	flags := model.FlagEnum
	if publicQualifier {
		flags |= model.FlagPublic
	}
	typeDef := ast.NewBLangTypeDefinitionWithData(n.getPositionWithoutMetadata(enumDeclarationNode), identifier, ast.TypeData{}, metadata.documentation, flags)
	typeDef.SetPosition(n.getPositionWithoutMetadata(enumDeclarationNode))

	if len(memberTypeNodes) > 0 {
		current := memberTypeNodes[0]
		for i := 1; i < len(memberTypeNodes); i++ {
			current = ast.NewBLangUnionTypeNode(
				typeDef.GetPosition(),
				ast.TypeData{TypeDescriptor: current},
				ast.TypeData{TypeDescriptor: memberTypeNodes[i]},
			)
		}
		typeDef.SetTypeData(ast.TypeData{TypeDescriptor: current})
	} else {
		neverType := &ast.BLangValueType{TypeKind: ast.TypeKindNever}
		neverType.SetPosition(diagnostics.NewBuiltinLocation())
		typeDef.SetTypeData(ast.TypeData{TypeDescriptor: neverType})
		n.cx.SyntaxError("missing enum member", typeDef.Name.GetPosition())
	}

	return typeDef, true
}

// transformEnumMember transforms an enum member into its constant and the singleton type it contributes to the
// enum's union type.
func (n *nodeBuilder) transformEnumMember(enumMemberNode *st.EnumMemberNode, publicQualifier bool) (*ast.BLangVariable, ast.TypeDescriptor, bool) {
	if !n.expectNode(enumMemberNode) {
		return nil, nil, false
	}
	metadata, ok := n.transformMetadata(enumMemberNode.Metadata())
	if !ok {
		return nil, nil, false
	}
	identifierToken := asToken(enumMemberNode.Identifier())
	if identifierToken == nil {
		n.internalError("missing enum member identifier", enumMemberNode)
		return nil, nil, false
	}
	identifier := n.transformIdentifier(identifierToken)
	if !n.expectOptionalToken(enumMemberNode.EqualToken()) {
		return nil, nil, false
	}
	var expr ast.BLangExpression
	if exprNode := enumMemberNode.ConstExprNode(); exprNode != nil {
		expr = n.transformExpr(exprNode)
	} else {
		literal, ok := n.createSimpleLiteralFromText(identifierToken, identifierToken.Text(), identifierToken.Kind())
		if !ok {
			return nil, nil, false
		}
		expr = literal
	}
	memberTypeName := identifierToken.Text()
	emptyStr := ""
	memberType := createUserDefinedType(
		n.getPosition(identifierToken),
		createIdentifier(diagnostics.NewBuiltinLocation(), &emptyStr, &emptyStr),
		createIdentifier(n.getPosition(identifierToken), &memberTypeName, &memberTypeName),
	)
	stringType := &ast.BLangValueType{TypeKind: ast.TypeKindString}
	stringType.SetPosition(diagnostics.NewBuiltinLocation())
	var flags model.Flag
	if publicQualifier {
		flags |= model.FlagPublic
	}
	constantNode := ast.NewBLangVariable(n.getPositionWithoutMetadata(enumMemberNode), identifier, stringType, expr, false, flags|model.FlagConstant|model.FlagEnumMember)
	constantNode.SetPosition(n.getPositionWithoutMetadata(enumMemberNode))
	constantNode.MarkdownDocumentationAttachment = metadata.documentation
	return constantNode, memberType, true
}

func (n *nodeBuilder) transformClassDefinition(classDefinitionNode *st.ClassDefinitionNode) (*ast.BLangClassDefinition, bool) {
	if !n.expectNode(classDefinitionNode) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(classDefinitionNode.Metadata())
	if !ok {
		return nil, false
	}
	visibility := classDefinitionNode.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(visibility) {
		return nil, false
	}
	flags, ok := n.classQualifierFlags(classDefinitionNode.ClassTypeQualifiers())
	if !ok || !n.expectToken(classDefinitionNode.ClassKeyword()) {
		return nil, false
	}
	blangClass := ast.NewBLangClassDefinitionWithFlags(flags | publicFlag(visibility))
	blangClass.SetPosition(n.getPositionWithoutMetadata(classDefinitionNode))
	applyMetadata(metadata, &blangClass)
	blangClass.Name = n.transformIdentifier(classDefinitionNode.ClassName())
	if !n.expectToken(classDefinitionNode.OpenBrace()) {
		return nil, false
	}

	members := n.collectClassDefnMembers(classDefinitionNode.Members())
	if !n.expectToken(classDefinitionNode.CloseBrace()) || !n.expectOptionalToken(classDefinitionNode.SemicolonToken()) {
		return nil, false
	}
	blangClass.Fields = members.Fields
	blangClass.Methods = members.Methods
	blangClass.InitFunction = members.InitFunction
	blangClass.ResourceMethods = members.ResourceMethods
	blangClass.BadTopLevelNodes = members.BadTopLevelNodes
	for _, inclusion := range members.UnresolvedInclusions {
		blangClass.AddUnresolvedInclusion(inclusion)
	}

	return &blangClass, true
}

func (n *nodeBuilder) classQualifierFlags(qualifiers st.NodeList[st.Token]) (model.Flag, bool) {
	var flags model.Flag
	for qualifier := range qualifiers.Iterator() {
		if !n.expectSignificantToken(qualifier) {
			return 0, false
		}
		switch qualifier.Kind() {
		case st.DISTINCT_KEYWORD:
			flags |= model.FlagDistinct
		case st.CLIENT_KEYWORD:
			flags |= model.FlagClient
		case st.READONLY_KEYWORD:
			flags |= model.FlagReadonly
		case st.SERVICE_KEYWORD:
			flags |= model.FlagService
		case st.ISOLATED_KEYWORD:
			flags |= model.FlagIsolated
		default:
			n.internalError("unexpected class qualifier", qualifier)
			return 0, false
		}
	}
	return flags, true
}

func (n *nodeBuilder) transformClassField(objectField *st.ObjectFieldNode) (*ast.BLangVariable, bool) {
	if !n.expectNode(objectField) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(objectField.Metadata())
	if !ok {
		return nil, false
	}
	vis := objectField.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(vis) {
		return nil, false
	}
	var flags model.Flag
	if vis != nil && vis.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	} else if vis != nil && vis.Kind() == st.PRIVATE_KEYWORD {
		flags |= model.FlagPrivate
	}
	qualifiers := objectField.QualifierList()
	for qualifier := range qualifiers.Iterator() {
		if !n.expectSignificantToken(qualifier) {
			return nil, false
		}
		if qualifier.Kind() == st.FINAL_KEYWORD {
			flags |= model.FlagFinal
		}
	}
	typeNode := n.transformType(objectField.TypeName())
	identifier := n.transformIdentifier(objectField.FieldName())
	if !n.expectOptionalToken(objectField.EqualsToken()) {
		return nil, false
	}
	var expr ast.BLangActionOrExpression
	if expression := objectField.Expression(); expression != nil {
		expr = n.transformExpr(expression)
	}
	if !n.expectToken(objectField.SemicolonToken()) {
		return nil, false
	}
	variable := ast.NewBLangVariable(n.getPosition(objectField), identifier, typeNode, expr, false, flags)
	variable.SetPosition(n.getPosition(objectField))
	applyMetadata(metadata, variable)
	return variable, true
}

func (n *nodeBuilder) transformResourcePathParameter(resourcePathParameterNode *st.ResourcePathParameterNode) (*ast.BLangResourcePathSegment, bool) {
	if !n.expectNode(resourcePathParameterNode) || !n.expectToken(resourcePathParameterNode.OpenBracketToken()) {
		return nil, false
	}
	seg := &ast.BLangResourcePathSegment{}
	switch resourcePathParameterNode.Kind() {
	case st.RESOURCE_PATH_SEGMENT_PARAM:
		seg.Kind = ast.ResourcePathSegmentParam
	case st.RESOURCE_PATH_REST_PARAM:
		seg.Kind = ast.ResourcePathSegmentParamRest
	default:
		n.cx.InternalError(fmt.Sprintf("unexpected resource path parameter node kind: %v", resourcePathParameterNode.Kind()), n.getPosition(resourcePathParameterNode))
		return nil, false
	}
	seg.SetPosition(n.getPosition(resourcePathParameterNode))
	if _, ok := n.transformAnnotations(resourcePathParameterNode.Annotations()); !ok {
		return nil, false
	}
	td := resourcePathParameterNode.TypeDescriptor()
	if td == nil {
		n.internalError("resource path parameter has no type descriptor", resourcePathParameterNode)
		return nil, false
	}
	seg.ParamType = n.transformType(td)
	if !n.expectOptionalSignificantToken(resourcePathParameterNode.EllipsisToken()) {
		return nil, false
	}
	if nameTok := resourcePathParameterNode.ParamName(); nameTok != nil {
		name, ok := n.createBLangIdentifier(nameTok)
		if !ok {
			return nil, false
		}
		seg.Name = name.Value
	}
	if !n.expectToken(resourcePathParameterNode.CloseBracketToken()) {
		return nil, false
	}
	return seg, true
}

func (n *nodeBuilder) createResourceMethodNode(funcDef *st.FunctionDefinition) (*ast.BLangResourceMethod, bool) {
	if !n.expectNode(funcDef) {
		return nil, false
	}
	metadata, ok := n.transformMetadata(funcDef.Metadata())
	if !ok {
		return nil, false
	}
	flags, ok := n.functionQualifierFlags(funcDef.QualifierList())
	if !ok || !n.expectToken(funcDef.FunctionKeyword()) {
		return nil, false
	}
	name := n.transformIdentifier(asToken(funcDef.FunctionName()))
	segments, ok := n.createResourcePathSegments(funcDef.RelativeResourcePath())
	if !ok {
		return nil, false
	}
	data := ast.InvokableData{
		Position: n.getPositionWithoutMetadata(funcDef),
		Name:     name,
		Flags:    flags | model.FlagAttached | model.FlagResource,
	}
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, name.GetValue())
	ok = n.populateFuncSignature(&data, funcDef.FunctionSignature())
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	if !ok || !n.populateFunctionBody(&data, funcDef.FunctionBody()) {
		return nil, false
	}
	rm := ast.NewBLangResourceMethod(data, segments)
	applyMetadata(metadata, rm)
	return rm, true
}

func (n *nodeBuilder) createResourcePathSegments(pathNodes st.NodeList[st.Node]) ([]ast.BLangResourcePathSegment, bool) {
	var segments []ast.BLangResourcePathSegment
	for node := range pathNodes.Iterator() {
		switch node := node.(type) {
		case *st.ResourcePathParameterNode:
			param, ok := n.transformResourcePathParameter(node)
			if !ok {
				return nil, false
			}
			segments = append(segments, *param)
		case st.Token:
			if !n.expectSignificantToken(node) {
				return nil, false
			}
			switch node.Kind() {
			case st.SLASH_TOKEN, st.DOT_TOKEN:
			case st.IDENTIFIER_TOKEN:
				seg := ast.BLangResourcePathSegment{Kind: ast.ResourcePathSegmentName, Name: node.Text()}
				seg.SetPosition(n.getPosition(node))
				segments = append(segments, seg)
			default:
				n.cx.InternalError(fmt.Sprintf("unexpected resource path node kind: %v", node.Kind()), n.getPosition(node))
				return nil, false
			}
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource path node kind: %v", node.Kind()), n.getPosition(node))
			return nil, false
		}
	}
	return segments, true
}

func (n *nodeBuilder) getBLangVariableNode(bindingPattern st.BindingPatternNode, varPos diagnostics.Location) (*ast.BLangVariable, bool) {
	switch bindingPattern := bindingPattern.(type) {
	case *st.WildcardBindingPatternNode:
		if !n.expectNode(bindingPattern) || !n.expectToken(bindingPattern.UnderscoreToken()) {
			return nil, false
		}
		ignore := n.createIgnoreIdentifier(bindingPattern)
		simpleVar := ast.NewBLangVariable(varPos, &ignore, nil, nil, false, 0)
		simpleVar.SetPosition(varPos)
		return simpleVar, true
	case *st.CaptureBindingPatternNode:
		if !n.expectNode(bindingPattern) {
			return nil, false
		}
		simpleVar := ast.NewBLangVariable(varPos, n.transformIdentifier(bindingPattern.VariableName()), nil, nil, false, 0)
		simpleVar.SetPosition(varPos)
		return simpleVar, true
	case *st.MappingBindingPatternNode, *st.ListBindingPatternNode, *st.ErrorBindingPatternNode, *st.RestBindingPatternNode:
		if !n.expectNode(bindingPattern) {
			return nil, false
		}
		n.unimplemented("binding pattern is not implemented", bindingPattern)
		return nil, false
	default:
		n.internalError("syntax kind is not a valid binding pattern", bindingPattern)
		return nil, false
	}
}

func (n *nodeBuilder) transformAssignmentStatement(assignmentStatementNode *st.AssignmentStatementNode) (*ast.BLangAssignment, bool) {
	if !n.expectNode(assignmentStatementNode) {
		return nil, false
	}
	varRef := assignmentStatementNode.VarRef()
	if kind := varRef.Kind(); kind == st.LIST_BINDING_PATTERN || kind == st.MAPPING_BINDING_PATTERN || kind == st.ERROR_BINDING_PATTERN {
		if nonTerminal, ok := varRef.(st.NonTerminalNode); ok && !n.expectNode(nonTerminal) {
			return nil, false
		}
		if !n.expectSignificantToken(assignmentStatementNode.EqualsToken()) {
			return nil, false
		}
		n.unimplemented("binding-pattern assignment is not implemented", varRef)
		return nil, false
	}

	lhsExpr := ast.NewBLangAssignmentLExpr(n.transformExpr(varRef), false)
	if !n.expectToken(assignmentStatementNode.EqualsToken()) {
		return nil, false
	}
	rhsExpr := n.transformActionOrExpr(assignmentStatementNode.Expression())
	if !n.expectToken(assignmentStatementNode.SemicolonToken()) {
		return nil, false
	}
	return ast.NewBLangAssignment(n.getPosition(assignmentStatementNode), lhsExpr, rhsExpr), true
}

func (n *nodeBuilder) transformCompoundAssignmentStatement(compoundAssignmentStmtNode *st.CompoundAssignmentStatementNode) (*ast.BLangCompoundAssignment, bool) {
	if !n.expectNode(compoundAssignmentStmtNode) {
		return nil, false
	}
	lhsExpr := ast.NewBLangAssignmentLExpr(n.transformExpr(compoundAssignmentStmtNode.LhsExpression()), true)
	binaryOperator := compoundAssignmentStmtNode.BinaryOperator()
	if !n.expectSignificantToken(binaryOperator) || !n.expectToken(compoundAssignmentStmtNode.EqualsToken()) {
		return nil, false
	}
	rhsExpr := n.transformActionOrExpr(compoundAssignmentStmtNode.RhsExpression())
	if !n.expectToken(compoundAssignmentStmtNode.SemicolonToken()) {
		return nil, false
	}
	return ast.NewBLangCompoundAssignment(
		n.getPosition(compoundAssignmentStmtNode),
		lhsExpr,
		rhsExpr,
		model.OperatorKindValueFrom(binaryOperator.Text()),
	), true
}

func (n *nodeBuilder) transformVariableDeclaration(variableDeclarationNode *st.VariableDeclarationNode) (*ast.BLangVariableDef, bool) {
	if !n.expectNode(variableDeclarationNode) {
		return nil, false
	}
	attachments, ok := n.transformAnnotations(variableDeclarationNode.Annotations())
	if !ok {
		return nil, false
	}
	finalKeyword := variableDeclarationNode.FinalKeyword()
	if !n.expectOptionalSignificantToken(finalKeyword) {
		return nil, false
	}
	varNode, ok := n.createBLangVarDef(
		n.getPosition(variableDeclarationNode),
		variableDeclarationNode.TypedBindingPattern(),
		variableDeclarationNode.EqualsToken(),
		variableDeclarationNode.Initializer(),
		false,
		finalKeyword,
	)
	if !ok || !n.expectToken(variableDeclarationNode.SemicolonToken()) {
		return nil, false
	}
	addAnnotationAttachments(attachments, varNode.Var)
	return varNode, true
}

// transformVarDefInitializer checks the `=` of a variable definition and transforms its initializer; both are
// required when initializerRequired is set.
func (n *nodeBuilder) transformVarDefInitializer(equalsToken st.Token, initializer st.ExpressionNode, initializerRequired bool) (ast.BLangActionOrExpression, bool) {
	if initializerRequired {
		if !n.expectToken(equalsToken) {
			return nil, false
		}
		if initializer == nil {
			n.internalError("variable definition has no initializer", nil)
			return nil, false
		}
	} else if !n.expectOptionalToken(equalsToken) {
		return nil, false
	}
	if initializer == nil {
		return nil, true
	}
	return n.transformActionOrExpr(initializer), true
}

func (n *nodeBuilder) createBLangVarDef(location diagnostics.Location, typedBindingPattern *st.TypedBindingPatternNode, equalsToken st.Token, initializer st.ExpressionNode, initializerRequired bool, finalKeyword st.Token) (*ast.BLangVariableDef, bool) {
	if !n.expectNode(typedBindingPattern) {
		return nil, false
	}
	typeDesc := typedBindingPattern.TypeDescriptor()
	declaredWithVar := isDeclaredWithVar(typeDesc)
	var typeNode ast.BType
	if !declaredWithVar {
		typeNode = n.transformType(typeDesc)
	} else if !n.transformVarTypeKeyword(typeDesc) {
		return nil, false
	}
	variable, ok := n.getBLangVariableNode(typedBindingPattern.BindingPattern(), location)
	if !ok {
		return nil, false
	}
	expr, ok := n.transformVarDefInitializer(equalsToken, initializer, initializerRequired)
	if !ok {
		return nil, false
	}
	var flags model.Flag
	if finalKeyword != nil {
		flags |= model.FlagFinal
	}
	variable = ast.NewBLangVariable(location, variable.Name, typeNode, expr, declaredWithVar, flags)
	variable.SetPosition(location)
	bLVarDef := &ast.BLangVariableDef{Var: variable}
	bLVarDef.SetPosition(location)
	return bLVarDef, true
}

func (n *nodeBuilder) transformBlockStatement(blockStatementNode *st.BlockStatementNode) (*ast.BLangBlockStmt, bool) {
	if !n.expectNode(blockStatementNode) || !n.expectToken(blockStatementNode.OpenBraceToken()) {
		return nil, false
	}
	bLBlockStmt := ast.BLangBlockStmt{}
	bLBlockStmt.Stmts = n.generateBLangStatements(blockStatementNode.Statements(), blockStatementNode)
	if !n.expectToken(blockStatementNode.CloseBraceToken()) {
		return nil, false
	}
	bLBlockStmt.SetPosition(n.getPosition(blockStatementNode))
	return &bLBlockStmt, true
}

func (n *nodeBuilder) generateBLangStatements(statementNodes st.NodeList[st.StatementNode], endNode st.Node) []ast.StatementNode {
	statements := []ast.StatementNode{}
	return *n.generateAndAddBLangStatements(statementNodes, &statements, 0, endNode)
}

func (n *nodeBuilder) generateAndAddBLangStatements(statementNodes st.NodeList[st.StatementNode], statements *[]ast.StatementNode, startPosition int, endNode st.Node) *[]ast.StatementNode {
	lastStmtIndex := statementNodes.Size() - 1
	for j := startPosition; j < statementNodes.Size(); j++ {
		currentStatement := statementNodes.Get(j)
		// TODO: Remove this check once statements are non null guaranteed
		if currentStatement == nil {
			continue
		}
		// If there is an `if` statement without an `else`, all the statements following that `if` statement
		// are added to a new block statement.
		if ifElseStmt, ok := currentStatement.(*st.IfElseStatementNode); ok && ifElseStmt.ElseBody() == nil {
			*statements = append(*statements, n.transformStatement(currentStatement))
			if j == lastStmtIndex {
				// Add an empty block statement if there are no statements following the `if` statement.
				emptyBlock := &ast.BLangBlockStmt{}
				emptyBlock.SetPosition(n.getPositionRange(currentStatement, endNode))
				*statements = append(*statements, emptyBlock)
				break
			}
			bLBlockStmt := &ast.BLangBlockStmt{}
			nextStmtIndex := j + 1
			n.generateAndAddBLangStatements(statementNodes, &bLBlockStmt.Stmts, nextStmtIndex, endNode)
			if nextStmtIndex <= lastStmtIndex {
				bLBlockStmt.SetPosition(n.getPositionRange(statementNodes.Get(nextStmtIndex), endNode))
			}
			*statements = append(*statements, bLBlockStmt)
			break
		} else {
			*statements = append(*statements, n.transformStatement(currentStatement))
		}
	}
	return statements
}

func (n *nodeBuilder) transformBreakStatement(breakStatementNode *st.BreakStatementNode) (*ast.BLangBreak, bool) {
	if !n.expectNode(breakStatementNode) || !n.expectToken(breakStatementNode.BreakToken()) ||
		!n.expectToken(breakStatementNode.SemicolonToken()) {
		return nil, false
	}
	bLBreak := &ast.BLangBreak{}
	bLBreak.SetPosition(n.getPosition(breakStatementNode))
	return bLBreak, true
}

func (n *nodeBuilder) transformFailStatement(failStatementNode *st.FailStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(failStatementNode) {
		return nil, false
	}
	n.unimplemented("transformFailStatement unimplemented", failStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformExpressionStatement(expressionStatement *st.ExpressionStatementNode) (*ast.BLangExpressionStmt, bool) {
	if !n.expectNode(expressionStatement) {
		return nil, false
	}
	bLExpressionStmt := ast.BLangExpressionStmt{}
	bLExpressionStmt.Expr = n.transformActionOrExpr(expressionStatement.Expression())
	if !n.expectToken(expressionStatement.SemicolonToken()) {
		return nil, false
	}
	bLExpressionStmt.SetPosition(n.getPosition(expressionStatement))
	return &bLExpressionStmt, true
}

func (n *nodeBuilder) transformContinueStatement(continueStatementNode *st.ContinueStatementNode) (*ast.BLangContinue, bool) {
	if !n.expectNode(continueStatementNode) || !n.expectToken(continueStatementNode.ContinueToken()) ||
		!n.expectToken(continueStatementNode.SemicolonToken()) {
		return nil, false
	}
	blContinue := &ast.BLangContinue{}
	blContinue.SetPosition(n.getPosition(continueStatementNode))
	return blContinue, true
}

func (n *nodeBuilder) transformExternalFunctionBody(externalFunctionBodyNode *st.ExternalFunctionBodyNode) (*ast.BLangExternFunctionBody, bool) {
	if !n.expectNode(externalFunctionBodyNode) || !n.expectToken(externalFunctionBodyNode.EqualsToken()) {
		return nil, false
	}
	if _, ok := n.transformAnnotations(externalFunctionBodyNode.Annotations()); !ok {
		return nil, false
	}
	if !n.expectToken(externalFunctionBodyNode.ExternalKeyword()) || !n.expectToken(externalFunctionBodyNode.SemicolonToken()) {
		return nil, false
	}
	body := &ast.BLangExternFunctionBody{}
	body.SetPosition(n.getPosition(externalFunctionBodyNode))
	return body, true
}

func (n *nodeBuilder) transformIfElseStatement(ifElseStatementNode *st.IfElseStatementNode) (*ast.BLangIf, bool) {
	if !n.expectNode(ifElseStatementNode) || !n.expectToken(ifElseStatementNode.IfKeyword()) {
		return nil, false
	}
	condition := n.transformExpr(ifElseStatementNode.Condition())
	body, ok := n.transformBlockStatement(ifElseStatementNode.IfBody())
	if !ok {
		return nil, false
	}
	var elseStmt ast.StatementNode
	if elseBody := ifElseStatementNode.ElseBody(); elseBody != nil {
		elseStmt, ok = n.transformElseBlock(elseBody)
		if !ok {
			return nil, false
		}
	}
	return ast.NewBLangIf(n.getPosition(ifElseStatementNode), condition, body, elseStmt), true
}

func (n *nodeBuilder) transformElseBlock(elseBody st.Node) (ast.StatementNode, bool) {
	elseBlockNode, ok := elseBody.(*st.ElseBlockNode)
	if !ok {
		n.internalError(fmt.Sprintf("unexpected else block node %T", elseBody), elseBody)
		return nil, false
	}
	if !n.expectNode(elseBlockNode) || !n.expectToken(elseBlockNode.ElseKeyword()) {
		return nil, false
	}
	switch body := elseBlockNode.ElseBody().(type) {
	case *st.IfElseStatementNode:
		return n.transformIfElseStatement(body)
	case *st.BlockStatementNode:
		return n.transformBlockStatement(body)
	default:
		n.internalError(fmt.Sprintf("unexpected else body node %T", body), body)
		return nil, false
	}
}

func (n *nodeBuilder) transformWhileStatement(whileStatementNode *st.WhileStatementNode) (*ast.BLangWhile, bool) {
	if !n.expectNode(whileStatementNode) || !n.expectToken(whileStatementNode.WhileKeyword()) {
		return nil, false
	}
	condition := n.transformExpr(whileStatementNode.Condition())
	body, ok := n.transformBlockStatement(whileStatementNode.WhileBody())
	if !ok {
		return nil, false
	}
	if onFailNode := whileStatementNode.OnFailClause(); onFailNode != nil {
		n.transformOnFailClause(onFailNode)
		return nil, false
	}
	return ast.NewBLangWhile(n.getPosition(whileStatementNode), condition, body, nil), true
}

func (n *nodeBuilder) transformPanicStatement(panicStatementNode *st.PanicStatementNode) (*ast.BLangPanic, bool) {
	if !n.expectNode(panicStatementNode) || !n.expectToken(panicStatementNode.PanicKeyword()) {
		return nil, false
	}
	bLPanic := &ast.BLangPanic{}
	bLPanic.SetPosition(n.getPosition(panicStatementNode))
	bLPanic.Expr = n.transformExpr(panicStatementNode.Expression())
	if !n.expectToken(panicStatementNode.SemicolonToken()) {
		return nil, false
	}
	return bLPanic, true
}

func (n *nodeBuilder) transformReturnStatement(returnStatementNode *st.ReturnStatementNode) (*ast.BLangReturn, bool) {
	if !n.expectNode(returnStatementNode) || !n.expectToken(returnStatementNode.ReturnKeyword()) {
		return nil, false
	}
	pos := n.getPosition(returnStatementNode)
	var expr ast.BLangActionOrExpression
	if returnStatementNode.Expression() != nil {
		expr = n.transformActionOrExpr(returnStatementNode.Expression())
	} else {
		expr = ast.NewBLangLiteral(pos, ast.LiteralKindNil, nil, "", false)
	}
	if !n.expectToken(returnStatementNode.SemicolonToken()) {
		return nil, false
	}
	return ast.NewBLangReturn(pos, expr), true
}

func (n *nodeBuilder) transformLocalTypeDefinitionStatement(localTypeDefinitionStatementNode *st.LocalTypeDefinitionStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(localTypeDefinitionStatementNode) {
		return nil, false
	}
	n.unimplemented("transformLocalTypeDefinitionStatement unimplemented", localTypeDefinitionStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformLockStatement(lockStatementNode *st.LockStatementNode) (*ast.BLangLock, bool) {
	if !n.expectNode(lockStatementNode) || !n.expectToken(lockStatementNode.LockKeyword()) {
		return nil, false
	}
	bLBlockStmt, ok := n.transformBlockStatement(lockStatementNode.BlockStatement())
	if !ok {
		return nil, false
	}
	if onFailClause := lockStatementNode.OnFailClause(); onFailClause != nil {
		if !n.expectNode(onFailClause) {
			return nil, false
		}
		n.cx.Unimplemented("on-fail clause on lock is not yet supported", n.getPosition(onFailClause))
		return nil, false
	}
	bLLock := &ast.BLangLock{}
	bLLock.SetPosition(n.getPosition(lockStatementNode))
	bLLock.Body = *bLBlockStmt
	return bLLock, true
}

func (n *nodeBuilder) transformForkStatement(forkStatementNode *st.ForkStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(forkStatementNode) {
		return nil, false
	}
	n.unimplemented("transformForkStatement unimplemented", forkStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformForEachStatement(forEachStatementNode *st.ForEachStatementNode) (*ast.BLangForeach, bool) {
	if !n.expectNode(forEachStatementNode) || !n.expectToken(forEachStatementNode.ForEachKeyword()) {
		return nil, false
	}
	typedBindingPattern := forEachStatementNode.TypedBindingPattern()
	varDef, ok := n.createBLangVarDef(n.getPosition(typedBindingPattern), typedBindingPattern, nil, nil, false, nil)
	if !ok || !n.expectToken(forEachStatementNode.InKeyword()) {
		return nil, false
	}
	collection := n.transformActionOrExpr(forEachStatementNode.ActionOrExpressionNode())
	body, ok := n.transformBlockStatement(forEachStatementNode.BlockStatement())
	if !ok {
		return nil, false
	}
	if onFailNode := forEachStatementNode.OnFailClause(); onFailNode != nil {
		n.transformOnFailClause(onFailNode)
		return nil, false
	}
	return ast.NewBLangForeach(n.getPosition(forEachStatementNode), varDef, collection, body, nil), true
}

// transformOnFailClause reports the on-fail clause, which is not implemented yet; callers fail after it.
func (n *nodeBuilder) transformOnFailClause(onFailClauseNode *st.OnFailClauseNode) {
	if !n.expectNode(onFailClauseNode) {
		return
	}
	n.unimplemented("transformOnFailClause unimplemented", onFailClauseNode)
}

func (n *nodeBuilder) transformDoStatement(doStatementNode *st.DoStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(doStatementNode) {
		return nil, false
	}
	n.unimplemented("transformDoStatement unimplemented", doStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformTransactionStatement(transactionStatementNode *st.TransactionStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(transactionStatementNode) {
		return nil, false
	}
	n.unimplemented("transformTransactionStatement unimplemented", transactionStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformRollbackStatement(rollbackStatementNode *st.RollbackStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(rollbackStatementNode) {
		return nil, false
	}
	n.unimplemented("transformRollbackStatement unimplemented", rollbackStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformRetryStatement(retryStatementNode *st.RetryStatementNode) (ast.StatementNode, bool) {
	if !n.expectNode(retryStatementNode) {
		return nil, false
	}
	n.unimplemented("transformRetryStatement unimplemented", retryStatementNode)
	return nil, false
}

func (n *nodeBuilder) transformMatchStatement(matchStatementNode *st.MatchStatementNode) (*ast.BLangMatchStatement, bool) {
	if !n.expectNode(matchStatementNode) || !n.expectToken(matchStatementNode.MatchKeyword()) {
		return nil, false
	}
	matchStatement := &ast.BLangMatchStatement{}
	matchStatement.Expr = n.transformActionOrExpr(matchStatementNode.Condition())
	if !n.expectToken(matchStatementNode.OpenBrace()) {
		return nil, false
	}
	matchClauses := matchStatementNode.MatchClauses()
	for matchClauseNode := range matchClauses.Iterator() {
		matchClause, ok := n.transformMatchClause(matchClauseNode)
		if !ok {
			return nil, false
		}
		matchStatement.MatchClauses = append(matchStatement.MatchClauses, *matchClause)
	}
	if !n.expectToken(matchStatementNode.CloseBrace()) {
		return nil, false
	}
	if onFailNode := matchStatementNode.OnFailClause(); onFailNode != nil {
		n.transformOnFailClause(onFailNode)
		return nil, false
	}
	matchStatement.SetPosition(n.getPosition(matchStatementNode))
	return matchStatement, true
}

func (n *nodeBuilder) transformMatchClause(matchClauseNode *st.MatchClauseNode) (*ast.BLangMatchClause, bool) {
	if !n.expectNode(matchClauseNode) {
		return nil, false
	}
	bLangMatchClause := &ast.BLangMatchClause{}
	bLangMatchClause.SetPosition(n.getPosition(matchClauseNode))

	matchPatterns := matchClauseNode.MatchPatterns()
	for matchPattern, separator := range matchPatterns.SeparatedElements() {
		bLangMatchPattern, ok := n.transformMatchPattern(matchPattern)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		bLangMatchClause.Patterns = append(bLangMatchClause.Patterns, bLangMatchPattern)
	}

	if matchGuardNode := matchClauseNode.MatchGuard(); matchGuardNode != nil {
		if !n.expectNode(matchGuardNode) || !n.expectToken(matchGuardNode.IfKeyword()) {
			return nil, false
		}
		bLangMatchClause.Guard = n.transformExpr(matchGuardNode.Expression())
	}

	if !n.expectToken(matchClauseNode.RightDoubleArrow()) {
		return nil, false
	}
	body, ok := n.transformBlockStatement(matchClauseNode.BlockStatement())
	if !ok {
		return nil, false
	}
	bLangMatchClause.Body = *body
	return bLangMatchClause, true
}

func (n *nodeBuilder) transformMatchPattern(matchPattern st.Node) (ast.BLangMatchPattern, bool) {
	matchPatternPos := n.getPosition(matchPattern)
	switch matchPattern.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		nameRef := matchPattern.(*st.SimpleNameReferenceNode)
		if name := nameRef.Name(); name != nil && name.Text() == "_" {
			if !n.expectNode(nameRef) || !n.expectToken(name) {
				return nil, false
			}
			return n.createWildCardMatchPattern(matchPatternPos), true
		}
		return n.createConstMatchPattern(matchPattern, matchPatternPos), true

	case st.IDENTIFIER_TOKEN:
		idToken := matchPattern.(st.Token)
		if idToken.Text() == "_" {
			if !n.expectToken(idToken) {
				return nil, false
			}
			return n.createWildCardMatchPattern(matchPatternPos), true
		}
		return n.createConstMatchPattern(matchPattern, matchPatternPos), true

	case st.NUMERIC_LITERAL,
		st.STRING_LITERAL,
		st.QUALIFIED_NAME_REFERENCE,
		st.NULL_LITERAL,
		st.NIL_LITERAL,
		st.BOOLEAN_LITERAL,
		st.UNARY_EXPRESSION:
		return n.createConstMatchPattern(matchPattern, matchPatternPos), true

	case st.LIST_MATCH_PATTERN, st.MAPPING_MATCH_PATTERN, st.ERROR_MATCH_PATTERN, st.TYPED_BINDING_PATTERN:
		n.unimplemented(fmt.Sprintf("match pattern kind %v is not implemented", matchPattern.Kind()), matchPattern)
		return nil, false

	default:
		n.rejectNode(matchPattern, fmt.Sprintf("unexpected match pattern kind: %v", matchPattern.Kind()))
		return nil, false
	}
}

func (n *nodeBuilder) createWildCardMatchPattern(pos diagnostics.Location) *ast.BLangWildCardMatchPattern {
	bLangWildCard := &ast.BLangWildCardMatchPattern{}
	bLangWildCard.SetPosition(pos)
	return bLangWildCard
}

func (n *nodeBuilder) createConstMatchPattern(matchPattern st.Node, pos diagnostics.Location) *ast.BLangConstPattern {
	bLangConstPattern := &ast.BLangConstPattern{}
	bLangConstPattern.Expr = n.transformExpr(matchPattern)
	bLangConstPattern.SetPosition(pos)
	return bLangConstPattern
}

func (n *nodeBuilder) transformFunctionBodyBlock(functionBodyBlockNode *st.FunctionBodyBlockNode) (*ast.BLangBlockFunctionBody, bool) {
	if !n.expectNode(functionBodyBlockNode) || !n.expectToken(functionBodyBlockNode.OpenBraceToken()) {
		return nil, false
	}
	if namedWorkerDeclarator := functionBodyBlockNode.NamedWorkerDeclarator(); namedWorkerDeclarator != nil {
		if !n.expectNode(namedWorkerDeclarator) {
			return nil, false
		}
		n.unimplemented("named worker declarators are not implemented", namedWorkerDeclarator)
		return nil, false
	}

	bLFuncBody := &ast.BLangBlockFunctionBody{}
	statements := []ast.StatementNode{}
	n.generateAndAddBLangStatements(functionBodyBlockNode.Statements(), &statements, 0, functionBodyBlockNode)
	if !n.expectToken(functionBodyBlockNode.CloseBraceToken()) || !n.expectOptionalToken(functionBodyBlockNode.SemicolonToken()) {
		return nil, false
	}
	bLFuncBody.Stmts = statements
	bLFuncBody.SetPosition(n.getPosition(functionBodyBlockNode))
	return bLFuncBody, true
}

// createSpecificFieldNameLiteral builds a string-literal expression for a
// non-computed mapping-constructor key. The field name is a static identifier
// or string literal, not a runtime expression, so it must not be represented
// as a var-ref.
func (n *nodeBuilder) createSpecificFieldNameLiteral(fieldName st.Node) (*ast.BLangLiteral, bool) {
	var nameToken st.Token
	switch fieldName := fieldName.(type) {
	case *st.BasicLiteralNode:
		return n.createSimpleLiteral(fieldName)
	case *st.SimpleNameReferenceNode:
		if !n.expectNode(fieldName) {
			return nil, false
		}
		nameToken = fieldName.Name()
	case st.Token:
		nameToken = fieldName
	default:
		n.internalError(fmt.Sprintf("unexpected mapping field name node %T", fieldName), fieldName)
		return nil, false
	}
	identifier, ok := n.createBLangIdentifier(nameToken)
	if !ok {
		return nil, false
	}
	return fieldNameLiteral(n.getPosition(fieldName), identifier.Value), true
}

func fieldNameLiteral(pos diagnostics.Location, name string) *ast.BLangLiteral {
	return ast.NewBLangLiteral(pos, ast.LiteralKindString, name, name, false)
}

// shorthandFieldName returns the field name of a `{x}` shorthand field from tokens already checked by the
// transformation of its value.
func (n *nodeBuilder) shorthandFieldName(fieldName st.Node) (string, bool) {
	switch fieldName := fieldName.(type) {
	case *st.SimpleNameReferenceNode:
		return identifierTokenValue(fieldName.Name()), true
	case st.Token:
		return identifierTokenValue(fieldName), true
	default:
		n.internalError(fmt.Sprintf("unexpected mapping field name node %T", fieldName), fieldName)
		return "", false
	}
}

func (n *nodeBuilder) transformBracedExpression(bracedExpression *st.BracedExpressionNode) (*ast.BLangGroupExpr, bool) {
	expr, ok := n.transformBracedInnerExpression(bracedExpression)
	if !ok {
		return nil, false
	}
	group := ast.BLangGroupExpr{}
	group.Expression = expr
	group.SetPosition(n.getPosition(bracedExpression))
	return &group, true
}

// transformBracedAction transforms a parenthesized action into the action it encloses.
func (n *nodeBuilder) transformBracedAction(bracedAction *st.BracedExpressionNode) (ast.BLangActionOrExpression, bool) {
	return transformBraced(n, bracedAction, n.transformActionOrExpr)
}

// transformBracedInnerExpression checks the parentheses of a braced expression and transforms the expression
// they enclose, without wrapping it in a group expression.
func (n *nodeBuilder) transformBracedInnerExpression(bracedExpression *st.BracedExpressionNode) (ast.BLangExpression, bool) {
	return transformBraced(n, bracedExpression, n.transformExpr)
}

func transformBraced[T any](n *nodeBuilder, braced *st.BracedExpressionNode, transformInner func(st.Node) T) (T, bool) {
	var zero T
	if !n.expectNode(braced) || !n.expectToken(braced.OpenParen()) {
		return zero, false
	}
	inner := transformInner(braced.Expression())
	if !n.expectToken(braced.CloseParen()) {
		return zero, false
	}
	return inner, true
}

// transformAccessContainer transforms the container of a field access, dropping any enclosing parentheses.
func (n *nodeBuilder) transformAccessContainer(containerExpr st.Node) (ast.BLangExpression, bool) {
	if bracedExpr, ok := containerExpr.(*st.BracedExpressionNode); ok {
		return n.transformBracedInnerExpression(bracedExpr)
	}
	return n.transformExpr(containerExpr), true
}

func (n *nodeBuilder) transformBinaryExpression(binaryExpression *st.BinaryExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(binaryExpression) {
		return nil, false
	}
	lhsExpr := n.transformExpr(binaryExpression.LhsExpr())
	operator := binaryExpression.Operator()
	if operator == nil {
		n.cx.InternalError("binary expression is missing an operator token", n.getPosition(binaryExpression))
		return nil, false
	}
	if !n.expectSignificantToken(operator) {
		return nil, false
	}
	rhsExpr := n.transformExpr(binaryExpression.RhsExpr())
	if operator.Kind() == st.ELVIS_TOKEN {
		return ast.NewBLangNilConditionalExpr(n.getPosition(binaryExpression), lhsExpr, rhsExpr), true
	}

	bLBinaryExpr := ast.BLangBinaryExpr{}
	bLBinaryExpr.SetPosition(n.getPosition(binaryExpression))
	bLBinaryExpr.LhsExpr = lhsExpr
	bLBinaryExpr.RhsExpr = rhsExpr
	bLBinaryExpr.OpKind = model.OperatorKindValueFrom(operator.Text())
	return &bLBinaryExpr, true
}

func (n *nodeBuilder) transformCheckExpression(checkExpression *st.CheckExpressionNode) (ast.BLangExpression, bool) {
	checkKeyword := checkExpression.CheckKeyword()
	if !n.expectNode(checkExpression) || !n.expectSignificantToken(checkKeyword) {
		return nil, false
	}
	pos := n.getPosition(checkExpression)
	// we are deviating from the spec here (https://ballerina.io/spec/lang/master/#section_6.33) check is only suppose
	// to work with expression but jBallerina also allow remote method calls (which is an action)
	expr := n.transformActionOrExpr(checkExpression.Expression())
	if checkKeyword.Kind() == st.CHECK_KEYWORD {
		checkedExpr := &ast.BLangCheckedExpr{}
		checkedExpr.SetPosition(pos)
		checkedExpr.Expr = expr
		return checkedExpr, true
	}
	checkPanickedExpr := &ast.BLangCheckPanickedExpr{}
	checkPanickedExpr.SetPosition(pos)
	checkPanickedExpr.Expr = expr
	return checkPanickedExpr, true
}

// transformFieldName transforms the field name of a field access, which must be a simple name reference.
func (n *nodeBuilder) transformFieldName(fieldName st.Node) (ast.IdentifierNode, bool) {
	simpleNameRef, ok := fieldName.(*st.SimpleNameReferenceNode)
	if !ok {
		n.internalError(fmt.Sprintf("unexpected field name node %T", fieldName), fieldName)
		return nil, false
	}
	if !n.expectNode(simpleNameRef) {
		return nil, false
	}
	return n.transformIdentifier(simpleNameRef.Name()), true
}

func (n *nodeBuilder) unsupportedQualifiedFieldName(message string, node *st.QualifiedNameReferenceNode) {
	if !n.expectNode(node) {
		return
	}
	if _, ok := n.createBLangModulePrefixIdentifier(node.ModulePrefix()); !ok || !n.expectColon(node.Colon()) {
		return
	}
	if _, ok := n.createBLangIdentifier(asToken(node.Identifier())); !ok {
		return
	}
	n.unimplemented(message, node)
}

func (n *nodeBuilder) transformFieldAccessExpression(fieldAccessExpression *st.FieldAccessExpressionNode) (*ast.BLangFieldBaseAccess, bool) {
	if !n.expectNode(fieldAccessExpression) {
		return nil, false
	}
	expr, ok := n.transformAccessContainer(fieldAccessExpression.Expression())
	if !ok || !n.expectToken(fieldAccessExpression.DotToken()) {
		return nil, false
	}
	fieldName := fieldAccessExpression.FieldName()
	if qualified, ok := fieldName.(*st.QualifiedNameReferenceNode); ok {
		n.unsupportedQualifiedFieldName("transformFieldAccessExpression: QUALIFIED_NAME_REFERENCE unsupported", qualified)
		return nil, false
	}
	field, ok := n.transformFieldName(fieldName)
	if !ok {
		return nil, false
	}
	bLFieldBasedAccess := &ast.BLangFieldBaseAccess{}
	bLFieldBasedAccess.Field = field
	bLFieldBasedAccess.Expr = expr
	bLFieldBasedAccess.SetPosition(n.getPosition(fieldAccessExpression))
	return bLFieldBasedAccess, true
}

func (n *nodeBuilder) transformFunctionCallExpression(functionCallExpression *st.FunctionCallExpressionNode) (*ast.BLangInvocation, bool) {
	if !n.expectNode(functionCallExpression) {
		return nil, false
	}
	return n.createBLangInvocation(
		functionCallExpression.FunctionName(),
		functionCallExpression.OpenParenToken(),
		functionCallExpression.Arguments(),
		functionCallExpression.CloseParenToken(),
		n.getPosition(functionCallExpression))
}

func (n *nodeBuilder) transformMethodCallExpression(methodCallExpression *st.MethodCallExpressionNode) (*ast.BLangInvocation, bool) {
	if !n.expectNode(methodCallExpression) {
		return nil, false
	}
	expr := n.transformExpr(methodCallExpression.Expression())
	if !n.expectToken(methodCallExpression.DotToken()) {
		return nil, false
	}
	bLInvocation, ok := n.createBLangInvocation(
		methodCallExpression.MethodName(),
		methodCallExpression.OpenParenToken(),
		methodCallExpression.Arguments(),
		methodCallExpression.CloseParenToken(),
		n.getPosition(methodCallExpression))
	if !ok {
		return nil, false
	}
	bLInvocation.Expr = expr
	return bLInvocation, true
}

func (n *nodeBuilder) transformMappingConstructorExpression(mappingConstructorExpression *st.MappingConstructorExpressionNode) (*ast.BLangMappingConstructorExpr, bool) {
	if !n.expectNode(mappingConstructorExpression) || !n.expectToken(mappingConstructorExpression.OpenBrace()) {
		return nil, false
	}
	mappingConstructor := &ast.BLangMappingConstructorExpr{
		Fields: make([]ast.MappingField, 0),
	}
	fields := mappingConstructorExpression.FieldNodes()
	for field, separator := range fields.SeparatedElements() {
		if !n.addMappingField(mappingConstructor, field) || !n.expectOptionalToken(separator) {
			return nil, false
		}
	}
	if !n.expectToken(mappingConstructorExpression.CloseBrace()) {
		return nil, false
	}
	mappingConstructor.SetPosition(n.getPosition(mappingConstructorExpression))
	return mappingConstructor, true
}

func (n *nodeBuilder) addMappingField(mappingConstructor *ast.BLangMappingConstructorExpr, field st.Node) bool {
	switch field := field.(type) {
	case *st.SpreadFieldNode:
		if !n.expectNode(field) {
			return false
		}
		n.unimplemented("mapping constructor spread field not implemented", field)
		return false
	case *st.ComputedNameFieldNode:
		keyValueField, ok := n.transformComputedNameField(field)
		if !ok {
			return false
		}
		mappingConstructor.Fields = append(mappingConstructor.Fields, keyValueField)
		return true
	case *st.SpecificFieldNode:
		return n.addSpecificField(mappingConstructor, field)
	default:
		n.internalError(fmt.Sprintf("unexpected mapping field kind: %v", field.Kind()), field)
		return false
	}
}

func (n *nodeBuilder) transformComputedNameField(computedNameField *st.ComputedNameFieldNode) (*ast.BLangMappingKeyValueField, bool) {
	if !n.expectNode(computedNameField) || !n.expectToken(computedNameField.OpenBracket()) {
		return nil, false
	}
	keyExpr := n.transformExpr(computedNameField.FieldNameExpr())
	if !n.expectToken(computedNameField.CloseBracket()) || !n.expectToken(computedNameField.ColonToken()) {
		return nil, false
	}
	key := &ast.BLangMappingKey{
		Expr: keyExpr,
		Kind: ast.MappingKeyComputed,
	}
	key.SetPosition(n.getPosition(computedNameField.FieldNameExpr()))
	keyValueField := &ast.BLangMappingKeyValueField{
		Key:       key,
		ValueExpr: n.transformExpr(computedNameField.ValueExpr()),
	}
	keyValueField.SetPosition(n.getPosition(computedNameField))
	return keyValueField, true
}

func (n *nodeBuilder) addSpecificField(mappingConstructor *ast.BLangMappingConstructorExpr, specificField *st.SpecificFieldNode) bool {
	readonlyKeyword := specificField.ReadonlyKeyword()
	if !n.expectNode(specificField) || !n.expectOptionalSignificantToken(readonlyKeyword) {
		return false
	}
	fieldName := specificField.FieldName()
	var keyLiteral *ast.BLangLiteral
	var value ast.BLangExpression
	if valueExpr := specificField.ValueExpr(); valueExpr == nil {
		value = n.transformExpr(fieldName)
		name, ok := n.shorthandFieldName(fieldName)
		if !ok || !n.expectOptionalToken(specificField.Colon()) {
			return false
		}
		keyLiteral = fieldNameLiteral(n.getPosition(fieldName), name)
	} else {
		var ok bool
		keyLiteral, ok = n.createSpecificFieldNameLiteral(fieldName)
		if !ok || !n.expectOptionalToken(specificField.Colon()) {
			return false
		}
		value = n.transformExpr(valueExpr)
	}
	keyKind := ast.MappingKeyIdentifier
	if _, isStringLit := fieldName.(*st.BasicLiteralNode); isStringLit {
		keyKind = ast.MappingKeyStringLiteral
	}
	key := &ast.BLangMappingKey{
		Expr: keyLiteral,
		Kind: keyKind,
	}
	key.SetPosition(n.getPosition(fieldName))
	keyValueField := &ast.BLangMappingKeyValueField{
		Key:       key,
		ValueExpr: value,
	}
	keyValueField.SetPosition(n.getPosition(specificField))
	mappingConstructor.Fields = append(mappingConstructor.Fields, keyValueField)
	if readonlyKeyword != nil {
		keyName, ok := keyLiteral.GetValue().(string)
		if !ok {
			n.internalError("mapping constructor field name is not a string", fieldName)
			return false
		}
		mappingConstructor.ReadonlyFields = append(mappingConstructor.ReadonlyFields, keyName)
	}
	return true
}

func (n *nodeBuilder) transformIndexedExpression(indexedExpression *st.IndexedExpressionNode) (*ast.BLangIndexBasedAccess, bool) {
	if !n.expectNode(indexedExpression) {
		return nil, false
	}
	indexBasedAccess := &ast.BLangIndexBasedAccess{}
	indexBasedAccess.SetPosition(n.getPosition(indexedExpression))
	indexBasedAccess.Expr = n.transformExpr(indexedExpression.ContainerExpression())
	if !n.expectToken(indexedExpression.OpenBracket()) {
		return nil, false
	}
	keys := indexedExpression.KeyExpression()
	var exprs []ast.BLangExpression
	for key, separator := range keys.SeparatedElements() {
		exprs = append(exprs, n.transformExpr(key))
		if !n.expectOptionalToken(separator) {
			return nil, false
		}
	}
	if !n.expectToken(indexedExpression.CloseBracket()) {
		return nil, false
	}
	switch len(exprs) {
	case 0:
		n.internalError("missing key expression in member access expression", indexedExpression)
		return nil, false
	case 1:
		indexBasedAccess.IndexExpr = exprs[0]
	default:
		listConstructorExpr := &ast.BLangListConstructorExpr{}
		listConstructorExpr.SetPosition(n.getPositionRange(keys.Get(0), keys.Get(keys.Size()-1)))
		listConstructorExpr.Exprs = exprs
		indexBasedAccess.IndexExpr = listConstructorExpr
	}
	return indexBasedAccess, true
}

func (n *nodeBuilder) transformTypeofExpression(typeofExpression *st.TypeofExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(typeofExpression) {
		return nil, false
	}
	n.unimplemented("transformTypeofExpression unimplemented", typeofExpression)
	return nil, false
}

func (n *nodeBuilder) transformUnaryExpression(unaryExpression *st.UnaryExpressionNode) (ast.BLangExpression, bool) {
	unaryOperator := unaryExpression.UnaryOperator()
	if !n.expectNode(unaryExpression) || !n.expectSignificantToken(unaryOperator) {
		return nil, false
	}
	pos := n.getPosition(unaryExpression)
	operator := model.OperatorKindValueFrom(unaryOperator.Text())
	operand := unaryExpression.Expression()
	if operand.Kind() == st.NUMERIC_LITERAL {
		lit, ok := n.createSimpleLiteral(operand)
		if !ok {
			return nil, false
		}
		if operator == model.OperatorKind_SUB && foldNegativeIntLiteral(lit) {
			lit.SetPosition(pos)
			return lit, true
		}
		return ast.NewBLangUnaryExpr(pos, operator, lit), true
	}
	return ast.NewBLangUnaryExpr(pos, operator, n.transformExpr(operand)), true
}

// foldNegativeIntLiteral folds `-N` into a single int literal when `N` is an
// integer literal whose positive value overflows int64 but the negated value
// fits (e.g. `-9223372036854775808`). Without this fold, `N` is parsed as a
// float (losing precision) and later coerced back to int, corrupting the
// value used at runtime (e.g. for `<decimal>-9223372036854775808`).
func foldNegativeIntLiteral(lit *ast.BLangLiteral) bool {
	if lit.GetLiteralKind() != ast.LiteralKindInt {
		return false
	}
	if _, isFloat := lit.GetValue().(float64); !isFloat {
		return false
	}
	raw := lit.OriginalValue
	base := 10
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		raw = raw[2:]
		base = 16
	}
	parsed, err := strconv.ParseInt("-"+raw, base, 64)
	if err != nil {
		return false
	}
	lit.SetValue(parsed)
	lit.OriginalValue = "-" + lit.OriginalValue
	return true
}

func (n *nodeBuilder) transformInferredTypedescDefault(inferredTypedescDefaultNode *st.InferredTypedescDefaultNode) (*ast.BLangInferredTypedescDefault, bool) {
	if !n.expectNode(inferredTypedescDefaultNode) || !n.expectToken(inferredTypedescDefaultNode.LtToken()) ||
		!n.expectToken(inferredTypedescDefaultNode.GtToken()) {
		return nil, false
	}
	node := &ast.BLangInferredTypedescDefault{}
	node.SetPosition(n.getPosition(inferredTypedescDefaultNode))
	return node, true
}

func (n *nodeBuilder) transformObjectConstructorExpression(objectConstructorExpression *st.ObjectConstructorExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(objectConstructorExpression) {
		return nil, false
	}
	n.unimplemented("transformObjectConstructorExpression unimplemented", objectConstructorExpression)
	return nil, false
}

func (n *nodeBuilder) transformTypeTestExpression(typeTestExpression *st.TypeTestExpressionNode) (*ast.BLangTypeTestExpr, bool) {
	if !n.expectNode(typeTestExpression) {
		return nil, false
	}
	expr := n.transformExpr(typeTestExpression.Expression())
	isKeyword := typeTestExpression.IsKeyword()
	if !n.expectSignificantToken(isKeyword) {
		return nil, false
	}
	return ast.NewBLangTypeTestExpr(
		n.getPosition(typeTestExpression),
		expr,
		ast.TypeData{TypeDescriptor: n.transformType(typeTestExpression.TypeDescriptor())},
		isKeyword.Kind() == st.NOT_IS_KEYWORD,
	), true
}

func (n *nodeBuilder) transformRemoteMethodCallAction(remoteMethodCallActionNode *st.RemoteMethodCallActionNode) (*ast.BLangRemoteMethodCallAction, bool) {
	if !n.expectNode(remoteMethodCallActionNode) {
		return nil, false
	}
	expr := n.transformExpr(remoteMethodCallActionNode.Expression())
	if !n.expectToken(remoteMethodCallActionNode.RightArrowToken()) {
		return nil, false
	}
	inv, ok := n.createBLangInvocation(remoteMethodCallActionNode.MethodName(),
		remoteMethodCallActionNode.OpenParenToken(),
		remoteMethodCallActionNode.Arguments(),
		remoteMethodCallActionNode.CloseParenToken(),
		n.getPosition(remoteMethodCallActionNode))
	if !ok {
		return nil, false
	}
	return ast.NewBLangRemoteMethodCallAction(inv, expr, n.getPosition(remoteMethodCallActionNode)), true
}

func (n *nodeBuilder) transformTrapExpression(trapExpression *st.TrapExpressionNode) (*ast.BLangTrapExpr, bool) {
	if !n.expectNode(trapExpression) || !n.expectToken(trapExpression.TrapKeyword()) {
		return nil, false
	}
	trapExpr := &ast.BLangTrapExpr{}
	trapExpr.SetPosition(n.getPosition(trapExpression))
	trapExpr.Expr = n.transformActionOrExpr(trapExpression.Expression())
	return trapExpr, true
}

func (n *nodeBuilder) transformListConstructorExpression(listConstructorExpression *st.ListConstructorExpressionNode) (*ast.BLangListConstructorExpr, bool) {
	if !n.expectNode(listConstructorExpression) || !n.expectToken(listConstructorExpression.OpenBracket()) {
		return nil, false
	}
	var exprs []ast.BLangExpression
	var spreadMembers []bool
	expressions := listConstructorExpression.Expressions()
	for listMember, separator := range expressions.SeparatedElements() {
		expr, isSpread, ok := n.transformListMember(listMember)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		exprs = append(exprs, expr)
		spreadMembers = append(spreadMembers, isSpread)
	}
	if !n.expectToken(listConstructorExpression.CloseBracket()) {
		return nil, false
	}
	return ast.NewBLangListConstructorExpr(n.getPosition(listConstructorExpression), exprs, spreadMembers), true
}

func (n *nodeBuilder) transformListMember(listMember st.Node) (ast.BLangExpression, bool, bool) {
	spreadMember, isSpread := listMember.(*st.SpreadMemberNode)
	if !isSpread {
		return n.transformExpr(listMember), false, true
	}
	if !n.expectNode(spreadMember) || !n.expectToken(spreadMember.Ellipsis()) {
		return nil, false, false
	}
	return n.transformExpr(spreadMember.Expression()), true, true
}

func (n *nodeBuilder) transformTypeCastExpression(typeCastExpression *st.TypeCastExpressionNode) (*ast.BLangTypeConversionExpr, bool) {
	if !n.expectNode(typeCastExpression) || !n.expectToken(typeCastExpression.LtToken()) {
		return nil, false
	}
	typeConversionNode := &ast.BLangTypeConversionExpr{}
	typeConversionNode.SetPosition(n.getPosition(typeCastExpression))
	typeCastParamNode := typeCastExpression.TypeCastParam()
	if !n.expectNode(typeCastParamNode) {
		return nil, false
	}
	annotations := typeCastParamNode.Annotations()
	if annotations.Size() > 0 {
		n.unimplemented("type cast annotations are not yet implemented", annotations.Get(0))
		return nil, false
	}
	if typeCastParamNode.Type() == nil {
		n.internalError("type cast param node type is not present", typeCastExpression)
		return nil, false
	}
	typeConversionNode.TypeDescriptor = n.transformType(typeCastParamNode.Type())
	if !n.expectToken(typeCastExpression.GtToken()) {
		return nil, false
	}
	typeConversionNode.Expression = n.transformExpr(typeCastExpression.Expression())
	return typeConversionNode, true
}

func (n *nodeBuilder) transformTableConstructorExpression(tableConstructorExpression *st.TableConstructorExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(tableConstructorExpression) {
		return nil, false
	}
	n.unimplemented("transformTableConstructorExpression unimplemented", tableConstructorExpression)
	return nil, false
}

func (n *nodeBuilder) transformLetExpression(letExpression *st.LetExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(letExpression) {
		return nil, false
	}
	n.unimplemented("transformLetExpression unimplemented", letExpression)
	return nil, false
}

func (n *nodeBuilder) transformLetVariableDeclaration(letVariableDeclarationNode *st.LetVariableDeclarationNode) (*ast.BLangVariableDef, bool) {
	if !n.expectNode(letVariableDeclarationNode) {
		return nil, false
	}
	annotations := letVariableDeclarationNode.Annotations()
	if annotations.Size() > 0 {
		n.unimplemented("let variable annotations are not yet supported", annotations.Get(0))
		return nil, false
	}
	return n.createBLangVarDef(
		n.getPosition(letVariableDeclarationNode),
		letVariableDeclarationNode.TypedBindingPattern(),
		letVariableDeclarationNode.EqualsToken(),
		letVariableDeclarationNode.Expression(),
		true,
		nil,
	)
}

func (n *nodeBuilder) transformTemplateExpression(templateExpression *st.TemplateExpressionNode) (ast.BLangExpression, bool) {
	typeToken := templateExpression.Type()
	if !n.expectNode(templateExpression) || !n.expectOptionalSignificantToken(typeToken) {
		return nil, false
	}
	pos := n.getPosition(templateExpression)
	if typeToken == nil {
		n.cx.Unimplemented("raw templates not supported", pos)
		return nil, false
	}
	var expr ast.BLangExpression
	var ok bool
	switch typeToken.Text() {
	case "string":
		if !n.expectToken(templateExpression.StartBacktick()) {
			return nil, false
		}
		expr, ok = n.buildStringTemplateExpr(templateExpression, pos)
	case "xml":
		if !n.expectToken(templateExpression.StartBacktick()) {
			return nil, false
		}
		expr, ok = n.buildXMLTemplateExpr(templateExpression, pos)
	default:
		n.cx.Unimplemented("unsupported template expression kind", pos)
		return nil, false
	}
	if !ok || !n.expectToken(templateExpression.EndBacktick()) {
		return nil, false
	}
	return expr, true
}

// transformInterpolation checks the delimiters of an interpolation and transforms its expression.
func (n *nodeBuilder) transformInterpolation(interpolationNode *st.InterpolationNode) (ast.BLangExpression, bool) {
	if !n.expectNode(interpolationNode) || !n.expectToken(interpolationNode.InterpolationStartToken()) {
		return nil, false
	}
	expr := n.transformExpr(interpolationNode.Expression())
	if !n.expectToken(interpolationNode.InterpolationEndToken()) {
		return nil, false
	}
	return expr, true
}

func (n *nodeBuilder) buildXMLTemplateExpr(templateExpression *st.TemplateExpressionNode, pos diagnostics.Location) (ast.BLangExpression, bool) {
	if !xmlTemplateHasInterpolation(templateExpression.Content()) {
		// If we don't have interpolations we build a literal as an optimization
		return n.buildXMLSequenceLiteral(templateExpression, pos)
	}

	tpl := &ast.BLangXMLTemplateExpr{}
	tpl.SetPosition(pos)
	tpl.Kind = ast.TemplateExprKindXML
	for tok, diag := range n.flattenXMLTemplateContent(templateExpression.Content(), ast.XMLTemplateInsertionKindContent) {
		if diag != nil {
			if !n.reportXMLTemplateDiagnostic(diag) {
				return nil, false
			}
			continue
		}
		switch tok.Kind {
		case xmlTemplateTokenKindText:
			tpl.Strings = append(tpl.Strings, tok.Text)
			tpl.NamespaceInsertions = append(tpl.NamespaceInsertions, tok.NamespaceInsertions)
		case xmlTemplateTokenKindInsertion:
			tpl.Insertions = append(tpl.Insertions, tok.Insertion)
			tpl.InsertionKinds = append(tpl.InsertionKinds, tok.InsertionKind)
		}
	}
	return tpl, true
}

func (n *nodeBuilder) buildXMLSequenceLiteral(templateExpression *st.TemplateExpressionNode, pos diagnostics.Location) (ast.BLangExpression, bool) {
	var children []ast.BLangExpression
	content := templateExpression.Content()
	for child := range content.Iterator() {
		expr, ok := n.transformXMLItem(child)
		if !ok {
			return nil, false
		}
		children = append(children, expr)
	}
	if len(children) == 1 {
		return children[0], true
	}
	seq := &ast.BLangXMLSequenceLiteral{}
	seq.SetPosition(pos)
	seq.Children = children
	return seq, true
}

func (n *nodeBuilder) transformXMLItem(node st.Node) (ast.BLangExpression, bool) {
	switch node := node.(type) {
	case *st.XMLElementNode:
		return n.transformXMLElement(node)
	case *st.XMLEmptyElementNode:
		return n.transformXMLEmptyElement(node)
	case *st.XMLTextNode:
		return n.transformXMLText(node)
	case *st.XMLComment:
		return n.transformXMLComment(node)
	case *st.XMLProcessingInstruction:
		return n.transformXMLProcessingInstruction(node)
	case *st.XMLCDATANode:
		if !n.expectNode(node) {
			return nil, false
		}
		n.cx.Unimplemented("xml CDATA not yet supported", n.getPosition(node))
		return nil, false
	case *st.InterpolationNode:
		if !n.expectNode(node) {
			return nil, false
		}
		n.cx.Unimplemented("xml interpolation not yet supported", n.getPosition(node))
		return nil, false
	case st.Token:
		return n.transformXMLToken(node)
	default:
		n.internalError(fmt.Sprintf("unexpected xml item node %T", node), node)
		return nil, false
	}
}

func (n *nodeBuilder) transformXMLToken(token st.Token) (ast.BLangExpression, bool) {
	kind := token.Kind()
	if kind == st.XML_TEXT_CONTENT || kind == st.TEMPLATE_STRING || kind == st.CLOSE_BRACE_TOKEN ||
		kind == st.PROMPT_CONTENT || isTokenInRegExp(kind) {
		return n.createSimpleLiteral(token)
	}
	n.internalError("transformToken: syntax kind is not supported: "+kind.StrValue(), token)
	return nil, false
}

func xmlTemplateHasInterpolation(content st.NodeList[st.Node]) bool {
	for child := range content.Iterator() {
		if xmlNodeHasInterpolation(child) {
			return true
		}
	}
	return false
}

func xmlNodeHasInterpolation(node st.Node) bool {
	return firstXMLInterpolation(node) != nil
}

func firstXMLInterpolation(node st.Node) *st.InterpolationNode {
	switch x := node.(type) {
	case *st.InterpolationNode:
		return x
	case *st.XMLElementNode:
		content := x.Content()
		for child := range content.Iterator() {
			if ins := firstXMLInterpolation(child); ins != nil {
				return ins
			}
		}
		if start := x.StartTag(); start != nil {
			attrs := start.Attributes()
			for attr := range attrs.Iterator() {
				if value := attr.Value(); value != nil {
					if ins := firstXMLInterpolation(value); ins != nil {
						return ins
					}
				}
			}
		}
	case *st.XMLEmptyElementNode:
		attrs := x.Attributes()
		for attr := range attrs.Iterator() {
			if value := attr.Value(); value != nil {
				if ins := firstXMLInterpolation(value); ins != nil {
					return ins
				}
			}
		}
	case *st.XMLAttributeValue:
		value := x.Value()
		for child := range value.Iterator() {
			if ins := firstXMLInterpolation(child); ins != nil {
				return ins
			}
		}
	case *st.XMLComment:
		content := x.Content()
		for child := range content.Iterator() {
			if ins, ok := child.(*st.InterpolationNode); ok {
				return ins
			}
		}
	case *st.XMLProcessingInstruction:
		data := x.Data()
		for child := range data.Iterator() {
			if ins, ok := child.(*st.InterpolationNode); ok {
				return ins
			}
		}
	case *st.XMLCDATANode:
		content := x.Content()
		for child := range content.Iterator() {
			if ins, ok := child.(*st.InterpolationNode); ok {
				return ins
			}
		}
	}
	return nil
}

type xmlTemplateTokenKind uint8

const (
	xmlTemplateTokenKindText xmlTemplateTokenKind = iota
	xmlTemplateTokenKindInsertion
)

type xmlTemplateToken struct {
	Kind                xmlTemplateTokenKind
	Text                string
	NamespaceInsertions []ast.XMLTemplateNamespaceInsertion
	Insertion           ast.BLangExpression
	InsertionKind       ast.XMLTemplateInsertionKind
}

func newXMLTemplateTextToken(value string, insertions ...ast.XMLTemplateNamespaceInsertion) xmlTemplateToken {
	return xmlTemplateToken{Kind: xmlTemplateTokenKindText, Text: value, NamespaceInsertions: insertions}
}

func newXMLTemplateInsertionToken(expr ast.BLangExpression, kind ast.XMLTemplateInsertionKind) xmlTemplateToken {
	return xmlTemplateToken{Kind: xmlTemplateTokenKindInsertion, Insertion: expr, InsertionKind: kind}
}

type xmlTemplateTextAccumulator struct {
	text                strings.Builder
	namespaceInsertions []ast.XMLTemplateNamespaceInsertion
}

func appendXMLTemplateText(current *xmlTemplateTextAccumulator, tok xmlTemplateToken) *xmlTemplateTextAccumulator {
	if current == nil {
		current = &xmlTemplateTextAccumulator{}
	}
	baseOffset := current.text.Len()
	current.text.WriteString(tok.Text)
	for _, insn := range tok.NamespaceInsertions {
		insn.Offset += baseOffset
		current.namespaceInsertions = append(current.namespaceInsertions, insn)
	}
	return current
}

func isTemplateAccumEmtpy(t *xmlTemplateTextAccumulator) bool {
	return t == nil || t.text.Len() == 0
}

func xmlTemplateAccumToken(t *xmlTemplateTextAccumulator) xmlTemplateToken {
	if t == nil {
		return newXMLTemplateTextToken("")
	}
	return newXMLTemplateTextToken(t.text.String(), t.namespaceInsertions...)
}

type xmlTemplateDiagnostic struct {
	Message  string
	Position diagnostics.Location
	// Reported marks a failure that has already been reported, such as a syntax error.
	Reported bool
}

var xmlTemplateFailure = &xmlTemplateDiagnostic{Reported: true}

func (n *nodeBuilder) flattenXMLTemplateContent(content st.NodeList[st.Node], kind ast.XMLTemplateInsertionKind) iter.Seq2[xmlTemplateToken, *xmlTemplateDiagnostic] {
	return func(yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) {
		var current *xmlTemplateTextAccumulator
		rawYield := func(tok xmlTemplateToken, diag *xmlTemplateDiagnostic) bool {
			if diag != nil {
				if !isTemplateAccumEmtpy(current) && !yield(xmlTemplateAccumToken(current), nil) {
					return false
				}
				current = nil
				return yield(tok, diag)
			}
			switch tok.Kind {
			case xmlTemplateTokenKindText:
				current = appendXMLTemplateText(current, tok)
				return true
			case xmlTemplateTokenKindInsertion:
				if !yield(xmlTemplateAccumToken(current), nil) {
					return false
				}
				current = nil
				return yield(tok, nil)
			default:
				return true
			}
		}
		for child := range content.Iterator() {
			if !n.flattenXMLTemplateNodeWithNamespace(child, kind, nil, rawYield) {
				return
			}
		}
		yield(xmlTemplateAccumToken(current), nil)
	}
}

type xmlTemplateYield = func(xmlTemplateToken, *xmlTemplateDiagnostic) bool

func (n *nodeBuilder) yieldXMLTemplateText(token st.Token, yield xmlTemplateYield) bool {
	if !n.expectSignificantToken(token) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	return yield(newXMLTemplateTextToken(token.Text()), nil)
}

// expectVerbatimNodes checks nodes that are copied verbatim (or rejected) rather than transformed: tokens,
// interpolations and the own diagnostics of other nodes.
func (n *nodeBuilder) expectVerbatimNodes(nodes ...st.Node) bool {
	for _, node := range nodes {
		switch node := node.(type) {
		case nil:
		case st.Token:
			if !n.expectToken(node) {
				return false
			}
		case *st.InterpolationNode:
			if _, ok := n.transformInterpolation(node); !ok {
				return false
			}
		case st.NonTerminalNode:
			if !n.expectNode(node) {
				return false
			}
		}
	}
	return true
}

func nodeListItems(list st.NodeList[st.Node]) []st.Node {
	var items []st.Node
	for item := range list.Iterator() {
		items = append(items, item)
	}
	return items
}

func (n *nodeBuilder) flattenXMLTemplateNodeWithNamespace(
	node st.Node,
	kind ast.XMLTemplateInsertionKind,
	namespaceInsertion *ast.XMLTemplateNamespaceInsertion,
	yield xmlTemplateYield,
) bool {
	switch x := node.(type) {
	case st.Token:
		return n.yieldXMLTemplateText(x, yield)
	case *st.InterpolationNode:
		expr, ok := n.transformInterpolation(x)
		if !ok {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		return yield(newXMLTemplateInsertionToken(expr, kind), nil)
	case *st.XMLTextNode:
		if !n.expectNode(x) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		return n.yieldXMLTemplateText(x.Content(), yield)
	case *st.XMLElementNode:
		return n.flattenXMLTemplateElement(x, namespaceInsertion, yield)
	case *st.XMLEmptyElementNode:
		return n.flattenXMLTemplateEmptyElement(x, namespaceInsertion, yield)
	case *st.XMLComment:
		if !n.expectNode(x) || !n.expectVerbatimNodes(x.CommentStart()) || !n.expectVerbatimNodes(nodeListItems(x.Content())...) ||
			!n.expectVerbatimNodes(x.CommentEnd()) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml comment",
				Position: n.getPosition(ins),
			})
		}
		return yield(newXMLTemplateTextToken(st.ToSourceCode(x.InternalNode())), nil)
	case *st.XMLProcessingInstruction:
		return n.flattenXMLTemplatePI(x, yield)
	case *st.XMLCDATANode:
		if !n.expectNode(x) || !n.expectVerbatimNodes(x.CdataStart()) || !n.expectVerbatimNodes(nodeListItems(x.Content())...) ||
			!n.expectVerbatimNodes(x.CdataEnd()) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml CDATA section",
				Position: n.getPosition(ins),
			})
		}
		return yield(newXMLTemplateTextToken(st.ToSourceCode(x.InternalNode())), nil)
	default:
		if !n.expectVerbatimNodes(node) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		return yield(newXMLTemplateTextToken(st.ToSourceCode(node.InternalNode())), nil)
	}
}

func (n *nodeBuilder) flattenXMLTemplateElement(
	x *st.XMLElementNode,
	parentNamespaceInsertion *ast.XMLTemplateNamespaceInsertion,
	yield xmlTemplateYield,
) bool {
	if !n.expectNode(x) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	start := x.StartTag()
	if start == nil {
		n.internalError("xml element has no start tag", x)
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	if !n.expectNode(start) || !n.expectToken(start.LtToken()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	name, ok := n.xmlNameToString(start.Name())
	if !ok {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	namespaceInsertion := parentNamespaceInsertion
	if namespaceInsertion == nil {
		insn := n.collectXMLTemplateNamespaceInsertion(x)
		namespaceInsertion = &insn
	}
	startText := "<" + name
	if parentNamespaceInsertion == nil {
		namespaceInsertion.Offset = len(startText)
		if !yield(newXMLTemplateTextToken(startText, *namespaceInsertion), nil) {
			return false
		}
	} else if !yield(newXMLTemplateTextToken(startText), nil) {
		return false
	}
	if !n.flattenXMLTemplateAttributes(start.Attributes(), yield) {
		return false
	}
	if !n.expectToken(start.GtToken()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	if !yield(newXMLTemplateTextToken(">"), nil) {
		return false
	}
	content := x.Content()
	for child := range content.Iterator() {
		if !n.flattenXMLTemplateNodeWithNamespace(child, ast.XMLTemplateInsertionKindContent, namespaceInsertion, yield) {
			return false
		}
	}
	if !n.expectXMLEndTag(x) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	return yield(newXMLTemplateTextToken("</"+name+">"), nil)
}

func (n *nodeBuilder) expectXMLEndTag(element *st.XMLElementNode) bool {
	endTag := element.EndTag()
	if endTag == nil {
		n.internalError("xml element has no end tag", element)
		return false
	}
	if !n.expectNode(endTag) || !n.expectToken(endTag.LtToken()) || !n.expectToken(endTag.SlashToken()) {
		return false
	}
	if _, _, ok := n.xmlName(endTag.Name()); !ok {
		return false
	}
	return n.expectToken(endTag.GtToken())
}

func (n *nodeBuilder) flattenXMLTemplateEmptyElement(
	x *st.XMLEmptyElementNode,
	parentNamespaceInsertion *ast.XMLTemplateNamespaceInsertion,
	yield xmlTemplateYield,
) bool {
	if !n.expectNode(x) || !n.expectToken(x.LtToken()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	name, ok := n.xmlNameToString(x.Name())
	if !ok {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	namespaceInsertion := parentNamespaceInsertion
	if namespaceInsertion == nil {
		insn := n.collectXMLTemplateNamespaceInsertion(x)
		namespaceInsertion = &insn
	}
	startText := "<" + name
	if parentNamespaceInsertion == nil {
		namespaceInsertion.Offset = len(startText)
		if !yield(newXMLTemplateTextToken(startText, *namespaceInsertion), nil) {
			return false
		}
	} else if !yield(newXMLTemplateTextToken(startText), nil) {
		return false
	}
	if !n.flattenXMLTemplateAttributes(x.Attributes(), yield) {
		return false
	}
	if !n.expectToken(x.SlashToken()) || !n.expectToken(x.GtToken()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	return yield(newXMLTemplateTextToken("/>"), nil)
}

func (n *nodeBuilder) flattenXMLTemplatePI(x *st.XMLProcessingInstruction, yield xmlTemplateYield) bool {
	if !n.expectNode(x) || !n.expectToken(x.PiStart()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	target, ok := n.xmlNameToString(x.Target())
	if !ok {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	data, ok := n.xmlProcessingInstructionData(x)
	if !ok || !n.expectToken(x.PiEnd()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	if ins := firstXMLInterpolation(x); ins != nil {
		return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
			Message:  "interpolation is not allowed in xml processing instruction",
			Position: n.getPosition(ins),
		})
	}
	if !yield(newXMLTemplateTextToken("<?"), nil) {
		return false
	}
	if !yield(newXMLTemplateTextToken(target), nil) {
		return false
	}
	if data := strings.TrimSpace(data); data != "" {
		if !yield(newXMLTemplateTextToken(" "), nil) {
			return false
		}
		if !yield(newXMLTemplateTextToken(data), nil) {
			return false
		}
	}
	return yield(newXMLTemplateTextToken("?>"), nil)
}

func (n *nodeBuilder) xmlProcessingInstructionData(x *st.XMLProcessingInstruction) (string, bool) {
	var dataText strings.Builder
	data := x.Data()
	for child := range data.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			if !n.expectVerbatimNodes(child) {
				return "", false
			}
			continue
		}
		if !n.expectSignificantToken(tok) {
			return "", false
		}
		dataText.WriteString(tok.Text())
	}
	return dataText.String(), true
}

func (n *nodeBuilder) flattenXMLTemplateAttributes(attrs st.NodeList[*st.XMLAttributeNode], yield xmlTemplateYield) bool {
	for attr := range attrs.Iterator() {
		if !n.expectNode(attr) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		name, ok := n.xmlNameToString(attr.AttributeName())
		if !ok || !n.expectToken(attr.EqualToken()) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		if !yield(newXMLTemplateTextToken(" "+name+"="), nil) {
			return false
		}
		value := attr.Value()
		if value == nil {
			n.internalError("xml attribute has no value", attr)
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
		if !n.flattenXMLTemplateAttributeValue(name, value, yield) {
			return false
		}
	}
	return true
}

func (n *nodeBuilder) flattenXMLTemplateAttributeValue(
	name string,
	value *st.XMLAttributeValue,
	yield xmlTemplateYield,
) bool {
	startQuoteToken := value.StartQuote()
	if !n.expectNode(value) || !n.expectToken(startQuoteToken) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	startQuote := "\""
	if startQuoteToken != nil && startQuoteToken.Text() != "" {
		startQuote = startQuoteToken.Text()
	}
	endQuote := startQuote
	if q := value.EndQuote(); q != nil && q.Text() != "" {
		endQuote = q.Text()
	}
	if !yield(newXMLTemplateTextToken(startQuote), nil) {
		return false
	}
	isXMLNS := isXMLTemplateXMLNSName(name)
	items := value.Value()
	for child := range items.Iterator() {
		if ins, ok := child.(*st.InterpolationNode); ok {
			if isXMLNS {
				if _, ok := n.transformInterpolation(ins); !ok {
					return yield(xmlTemplateToken{}, xmlTemplateFailure)
				}
				if !yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
					Message:  "interpolation is not allowed in xml xmlns attribute value",
					Position: n.getPosition(child),
				}) {
					return false
				}
				continue
			}
			if !n.flattenXMLTemplateNodeWithNamespace(ins, ast.XMLTemplateInsertionKindAttribute, nil, yield) {
				return false
			}
			continue
		}
		if tok, ok := child.(st.Token); ok {
			if !n.yieldXMLTemplateText(tok, yield) {
				return false
			}
			continue
		}
		if !n.expectVerbatimNodes(child) {
			return yield(xmlTemplateToken{}, xmlTemplateFailure)
		}
	}
	if !n.expectToken(value.EndQuote()) {
		return yield(xmlTemplateToken{}, xmlTemplateFailure)
	}
	return yield(newXMLTemplateTextToken(endQuote), nil)
}

// reportXMLTemplateDiagnostic reports a diagnostic raised while flattening an xml template and returns whether
// building the template can continue.
func (n *nodeBuilder) reportXMLTemplateDiagnostic(diag *xmlTemplateDiagnostic) bool {
	if diag.Reported {
		return false
	}
	n.cx.SemanticError(diag.Message, diag.Position)
	return true
}

func (n *nodeBuilder) collectXMLTemplateNamespaceInsertion(node st.Node) ast.XMLTemplateNamespaceInsertion {
	insn := ast.XMLTemplateNamespaceInsertion{
		UsedPrefixes: map[string]struct{}{},
	}
	collectXMLTemplateNamespaceRefs(node, nil, &insn)
	return insn
}

func collectXMLTemplateNamespaceRefs(node st.Node, scopes []map[string]struct{}, insn *ast.XMLTemplateNamespaceInsertion) {
	switch x := node.(type) {
	case *st.XMLElementNode:
		start := x.StartTag()
		if start == nil {
			return
		}
		childScopes := appendXMLTemplateNamespaceScope(scopes, collectInlineXMLTemplatePrefixes(start.Attributes()))
		recordXMLTemplateNameRef(xmlNameText(start.Name()), true, childScopes, insn)
		collectXMLTemplateAttributeNamespaceRefs(start.Attributes(), childScopes, insn)
		content := x.Content()
		for child := range content.Iterator() {
			collectXMLTemplateNamespaceRefs(child, childScopes, insn)
		}
	case *st.XMLEmptyElementNode:
		childScopes := appendXMLTemplateNamespaceScope(scopes, collectInlineXMLTemplatePrefixes(x.Attributes()))
		recordXMLTemplateNameRef(xmlNameText(x.Name()), true, childScopes, insn)
		collectXMLTemplateAttributeNamespaceRefs(x.Attributes(), childScopes, insn)
	}
}

func collectXMLTemplateAttributeNamespaceRefs(
	attrs st.NodeList[*st.XMLAttributeNode],
	scopes []map[string]struct{},
	insn *ast.XMLTemplateNamespaceInsertion,
) {
	for attr := range attrs.Iterator() {
		name := xmlNameText(attr.AttributeName())
		if isXMLTemplateXMLNSName(name) {
			continue
		}
		recordXMLTemplateNameRef(name, false, scopes, insn)
	}
}

func recordXMLTemplateNameRef(name string, isElement bool, scopes []map[string]struct{}, insn *ast.XMLTemplateNamespaceInsertion) {
	prefix, _ := splitXMLTemplateName(name)
	if prefix == "xmlns" {
		return
	}
	if prefix != "" {
		if isXMLTemplatePrefixInScope(prefix, scopes) {
			return
		}
		insn.UsedPrefixes[prefix] = struct{}{}
		return
	}
	if isElement && !isXMLTemplatePrefixInScope("", scopes) {
		insn.NeedsDefaultNS = true
	}
}

func collectInlineXMLTemplatePrefixes(attrs st.NodeList[*st.XMLAttributeNode]) map[string]struct{} {
	prefixes := map[string]struct{}{}
	for attr := range attrs.Iterator() {
		name := xmlNameText(attr.AttributeName())
		if !isXMLTemplateXMLNSName(name) {
			continue
		}
		_, local := splitXMLTemplateName(name)
		if name == "xmlns" {
			prefixes[""] = struct{}{}
		} else {
			prefixes[local] = struct{}{}
		}
	}
	return prefixes
}

func appendXMLTemplateNamespaceScope(scopes []map[string]struct{}, scope map[string]struct{}) []map[string]struct{} {
	if len(scope) == 0 {
		return scopes
	}
	out := make([]map[string]struct{}, 0, len(scopes)+1)
	out = append(out, scopes...)
	out = append(out, scope)
	return out
}

func isXMLTemplatePrefixInScope(prefix string, scopes []map[string]struct{}) bool {
	for i := len(scopes) - 1; i >= 0; i-- {
		if _, ok := scopes[i][prefix]; ok {
			return true
		}
	}
	return false
}

func splitXMLTemplateName(name string) (string, string) {
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	return "", name
}

func isXMLTemplateXMLNSName(name string) bool {
	prefix, local := splitXMLTemplateName(name)
	return name == "xmlns" || prefix == "xmlns" && local != ""
}

func (n *nodeBuilder) buildStringTemplateExpr(node *st.TemplateExpressionNode, pos diagnostics.Location) (ast.BLangExpression, bool) {
	// We maintain fallowing 2 invariants
	// 1. First and last elements are always strings
	// 2. Between any two expressions there is a string
	// For this we will add empty strings. This is meant to reducing the number of branchings needed in runtime
	var strs []string
	var insertions []ast.BLangExpression
	content := node.Content()
	lastStr := false
	for child := range content.Iterator() {
		switch c := child.(type) {
		case st.Token:
			if !n.expectSignificantToken(c) {
				return nil, false
			}
			if c.Kind() != st.TEMPLATE_STRING {
				n.cx.InternalError(fmt.Sprintf("unexpected token kind in string template: %v", c.Kind()), n.getPosition(c))
				return nil, false
			}
			strs = append(strs, c.Text())
			lastStr = true
		case *st.InterpolationNode:
			if !lastStr {
				strs = append(strs, "")
			}
			expr, ok := n.transformInterpolation(c)
			if !ok {
				return nil, false
			}
			insertions = append(insertions, expr)
			lastStr = false
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected node in string template: %T", c), n.getPosition(child))
			return nil, false
		}
	}
	if !lastStr {
		strs = append(strs, "")
	}
	tpl := &ast.BLangTemplateExpr{Kind: ast.TemplateExprKindString, Strings: strs, Insertions: insertions}
	tpl.SetPosition(pos)
	return tpl, true
}

// xmlNameText reads an xml name without checking it; it is used by analyses that run ahead of the checked
// transformation.
func xmlNameText(name st.XMLNameNode) string {
	switch name := name.(type) {
	case *st.XMLSimpleNameNode:
		if tok := name.Name(); tok != nil {
			return tok.Text()
		}
	case *st.XMLQualifiedNameNode:
		prefixNode, localNode := name.Prefix(), name.Name()
		if prefixNode == nil || localNode == nil || prefixNode.Name() == nil || localNode.Name() == nil {
			return ""
		}
		return prefixNode.Name().Text() + ":" + localNode.Name().Text()
	}
	return ""
}

func (n *nodeBuilder) xmlNameToString(name st.XMLNameNode) (string, bool) {
	prefix, localName, ok := n.xmlName(name)
	if !ok {
		return "", false
	}
	if prefix == "" {
		return localName, true
	}
	return prefix + ":" + localName, true
}

func (n *nodeBuilder) xmlName(name st.XMLNameNode) (string, string, bool) {
	pos := n.getPosition(name)
	switch name := name.(type) {
	case *st.XMLSimpleNameNode:
		localName, ok := n.xmlSimpleName(name)
		return "", localName, ok
	case *st.XMLQualifiedNameNode:
		prefixNode := name.Prefix()
		localNode := name.Name()
		if prefixNode == nil || localNode == nil {
			n.cx.InternalError("xml qualified name missing prefix or local part", pos)
			return "", "", false
		}
		if !n.expectNode(name) {
			return "", "", false
		}
		prefix, ok := n.xmlSimpleName(prefixNode)
		if !ok || !n.expectToken(name.Colon()) {
			return "", "", false
		}
		localName, ok := n.xmlSimpleName(localNode)
		if !ok {
			return "", "", false
		}
		return prefix, localName, true
	}
	n.cx.InternalError(fmt.Sprintf("unexpected xml name kind: %T", name), pos)
	return "", "", false
}

func (n *nodeBuilder) xmlSimpleName(name *st.XMLSimpleNameNode) (string, bool) {
	tok := name.Name()
	if tok == nil {
		n.cx.InternalError("xml simple name missing identifier token", n.getPosition(name))
		return "", false
	}
	if !n.expectNode(name) || !n.expectToken(tok) {
		return "", false
	}
	return tok.Text(), true
}

func (n *nodeBuilder) xmlAttributes(attrs st.NodeList[*st.XMLAttributeNode]) ([]ast.BLangXMLAttribute, bool) {
	out := make([]ast.BLangXMLAttribute, 0, attrs.Size())
	for attrNode := range attrs.Iterator() {
		attr, ok := n.transformXMLAttribute(attrNode)
		if !ok {
			return nil, false
		}
		out = append(out, *attr)
	}
	return out, true
}

func (n *nodeBuilder) transformXMLElement(xMLElementNode *st.XMLElementNode) (*ast.BLangXMLElementLiteral, bool) {
	if !n.expectNode(xMLElementNode) {
		return nil, false
	}
	elem := &ast.BLangXMLElementLiteral{}
	elem.SetPosition(n.getPosition(xMLElementNode))
	start := xMLElementNode.StartTag()
	if start == nil {
		n.internalError("xml element has no start tag", xMLElementNode)
		return nil, false
	}
	if !n.expectNode(start) || !n.expectToken(start.LtToken()) {
		return nil, false
	}
	var ok bool
	elem.Prefix, elem.LocalName, ok = n.xmlName(start.Name())
	if !ok {
		return nil, false
	}
	elem.Attrs, ok = n.xmlAttributes(start.Attributes())
	if !ok || !n.expectToken(start.GtToken()) {
		return nil, false
	}
	var children []ast.BLangExpression
	content := xMLElementNode.Content()
	for child := range content.Iterator() {
		expr, ok := n.transformXMLItem(child)
		if !ok {
			return nil, false
		}
		children = append(children, expr)
	}
	if !n.expectXMLEndTag(xMLElementNode) {
		return nil, false
	}
	switch len(children) {
	case 0:
	case 1:
		elem.Content = children[0]
	default:
		seq := &ast.BLangXMLSequenceLiteral{}
		seq.SetPosition(elem.GetPosition())
		seq.Children = children
		elem.Content = seq
	}
	return elem, true
}

func (n *nodeBuilder) transformXMLEmptyElement(xMLEmptyElementNode *st.XMLEmptyElementNode) (*ast.BLangXMLElementLiteral, bool) {
	if !n.expectNode(xMLEmptyElementNode) || !n.expectToken(xMLEmptyElementNode.LtToken()) {
		return nil, false
	}
	elem := &ast.BLangXMLElementLiteral{}
	elem.SetPosition(n.getPosition(xMLEmptyElementNode))
	var ok bool
	elem.Prefix, elem.LocalName, ok = n.xmlName(xMLEmptyElementNode.Name())
	if !ok {
		return nil, false
	}
	elem.Attrs, ok = n.xmlAttributes(xMLEmptyElementNode.Attributes())
	if !ok || !n.expectToken(xMLEmptyElementNode.SlashToken()) || !n.expectToken(xMLEmptyElementNode.GtToken()) {
		return nil, false
	}
	return elem, true
}

func (n *nodeBuilder) transformXMLText(xMLTextNode *st.XMLTextNode) (*ast.BLangXMLTextLiteral, bool) {
	if !n.expectNode(xMLTextNode) {
		return nil, false
	}
	text := &ast.BLangXMLTextLiteral{}
	text.SetPosition(n.getPosition(xMLTextNode))
	content := xMLTextNode.Content()
	if !n.expectSignificantToken(content) {
		return nil, false
	}
	text.Body = content.Text()
	return text, true
}

func (n *nodeBuilder) transformXMLAttribute(xMLAttributeNode *st.XMLAttributeNode) (*ast.BLangXMLAttribute, bool) {
	if !n.expectNode(xMLAttributeNode) {
		return nil, false
	}
	attr := &ast.BLangXMLAttribute{}
	attr.SetPosition(n.getPosition(xMLAttributeNode))
	name, ok := n.xmlNameToString(xMLAttributeNode.AttributeName())
	if !ok || !n.expectToken(xMLAttributeNode.EqualToken()) {
		return nil, false
	}
	attr.Name = name
	valueNode := xMLAttributeNode.Value()
	if valueNode == nil {
		n.internalError("xml attribute has no value", xMLAttributeNode)
		return nil, false
	}
	value, ok := n.transformXMLAttributeValue(valueNode)
	if !ok {
		return nil, false
	}
	attr.Value = value
	return attr, true
}

func (n *nodeBuilder) transformXMLAttributeValue(xMLAttributeValue *st.XMLAttributeValue) (*ast.BLangLiteral, bool) {
	if !n.expectNode(xMLAttributeValue) || !n.expectToken(xMLAttributeValue.StartQuote()) {
		return nil, false
	}
	var b strings.Builder
	items := xMLAttributeValue.Value()
	for child := range items.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			n.cx.Unimplemented("xml attribute value interpolation not yet supported", n.getPosition(child))
			return nil, false
		}
		if !n.expectSignificantToken(tok) {
			return nil, false
		}
		b.WriteString(tok.Text())
	}
	if !n.expectToken(xMLAttributeValue.EndQuote()) {
		return nil, false
	}
	text := b.String()
	return ast.NewBLangLiteral(n.getPosition(xMLAttributeValue), ast.LiteralKindString, text, text, false), true
}

func (n *nodeBuilder) transformXMLComment(xMLComment *st.XMLComment) (*ast.BLangXMLCommentLiteral, bool) {
	if !n.expectNode(xMLComment) || !n.expectToken(xMLComment.CommentStart()) {
		return nil, false
	}
	c := &ast.BLangXMLCommentLiteral{}
	c.SetPosition(n.getPosition(xMLComment))
	var b strings.Builder
	content := xMLComment.Content()
	for child := range content.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			n.cx.Unimplemented("xml interpolation in comment not yet supported", n.getPosition(child))
			return nil, false
		}
		if !n.expectToken(tok) {
			return nil, false
		}
		b.WriteString(tok.Text())
	}
	if !n.expectToken(xMLComment.CommentEnd()) {
		return nil, false
	}
	c.Body = b.String()
	return c, true
}

func (n *nodeBuilder) transformXMLProcessingInstruction(xMLProcessingInstruction *st.XMLProcessingInstruction) (*ast.BLangXMLPILiteral, bool) {
	if !n.expectNode(xMLProcessingInstruction) || !n.expectToken(xMLProcessingInstruction.PiStart()) {
		return nil, false
	}
	pi := &ast.BLangXMLPILiteral{}
	pi.SetPosition(n.getPosition(xMLProcessingInstruction))
	target, ok := n.xmlNameToString(xMLProcessingInstruction.Target())
	if !ok {
		return nil, false
	}
	pi.Target = target
	var b strings.Builder
	data := xMLProcessingInstruction.Data()
	for child := range data.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			n.cx.Unimplemented("xml interpolation in processing instruction not yet supported", n.getPosition(child))
			return nil, false
		}
		if !n.expectToken(tok) {
			return nil, false
		}
		b.WriteString(tok.Text())
	}
	if !n.expectToken(xMLProcessingInstruction.PiEnd()) {
		return nil, false
	}
	pi.Data = b.String()
	return pi, true
}

func (n *nodeBuilder) transformPositionalArgument(positionalArgumentNode *st.PositionalArgumentNode) (ast.BLangExpression, bool) {
	if !n.expectNode(positionalArgumentNode) {
		return nil, false
	}
	return n.transformExpr(positionalArgumentNode.Expression()), true
}

func (n *nodeBuilder) transformNamedArgument(namedArgumentNode *st.NamedArgumentNode) (*ast.BLangNamedArgsExpression, bool) {
	argumentName := namedArgumentNode.ArgumentName()
	if !n.expectNode(namedArgumentNode) || !n.expectNode(argumentName) {
		return nil, false
	}
	name := n.transformIdentifier(argumentName.Name())
	if !n.expectToken(namedArgumentNode.EqualsToken()) {
		return nil, false
	}
	return ast.NewBLangNamedArgsExpression(
		n.getPosition(namedArgumentNode),
		name,
		n.transformExpr(namedArgumentNode.Expression()),
	), true
}

func (n *nodeBuilder) transformWildcardBindingPattern(wildcardBindingPatternNode *st.WildcardBindingPatternNode) (*ast.BLangWildCardBindingPattern, bool) {
	if !n.expectNode(wildcardBindingPatternNode) || !n.expectToken(wildcardBindingPatternNode.UnderscoreToken()) {
		return nil, false
	}
	bLWildCardBindingPattern := &ast.BLangWildCardBindingPattern{}
	bLWildCardBindingPattern.SetPosition(n.getPosition(wildcardBindingPatternNode))
	return bLWildCardBindingPattern, true
}

func (n *nodeBuilder) transformBasicLiteral(basicLiteralNode *st.BasicLiteralNode) (ast.BLangExpression, bool) {
	if !n.expectNode(basicLiteralNode) {
		return nil, false
	}
	n.unimplemented("transformBasicLiteral unimplemented", basicLiteralNode)
	return nil, false
}

func (n *nodeBuilder) transformRestArgument(restArgumentNode *st.RestArgumentNode) (ast.BLangExpression, bool) {
	if !n.expectNode(restArgumentNode) {
		return nil, false
	}
	n.unimplemented("transformRestArgument unimplemented", restArgumentNode)
	return nil, false
}

func (n *nodeBuilder) transformExplicitAnonymousFunctionExpression(anonFuncExprNode *st.ExplicitAnonymousFunctionExpressionNode) (*ast.BLangLambdaFunction, bool) {
	if !n.expectNode(anonFuncExprNode) {
		return nil, false
	}
	if _, ok := n.transformAnnotations(anonFuncExprNode.Annotations()); !ok {
		return nil, false
	}
	qualifierFlags, ok := n.functionQualifierFlags(anonFuncExprNode.QualifierList())
	if !ok || !n.expectToken(anonFuncExprNode.FunctionKeyword()) {
		return nil, false
	}
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	data := ast.InvokableData{
		Position: n.getPosition(anonFuncExprNode),
		Name:     &ident,
		Flags:    model.FlagLambda | model.FlagAnonymous | qualifierFlags,
	}
	if !n.populateFuncSignature(&data, anonFuncExprNode.FunctionSignature()) {
		return nil, false
	}
	body, ok := n.transformFunctionBody(anonFuncExprNode.FunctionBody())
	if !ok {
		return nil, false
	}
	data.Body = body
	bLFunction := ast.NewBLangFunction(data)
	lambdaFunc := &ast.BLangLambdaFunction{Function: bLFunction}
	lambdaFunc.SetPosition(bLFunction.GetPosition())
	return lambdaFunc, true
}

func (n *nodeBuilder) transformExpressionFunctionBody(expressionFunctionBodyNode *st.ExpressionFunctionBodyNode) (*ast.BLangExprFunctionBody, bool) {
	if !n.expectNode(expressionFunctionBodyNode) || !n.expectToken(expressionFunctionBodyNode.RightDoubleArrow()) {
		return nil, false
	}
	exprBody := &ast.BLangExprFunctionBody{}
	exprBody.Expr = n.transformExpr(expressionFunctionBodyNode.Expression())
	if !n.expectOptionalToken(expressionFunctionBodyNode.Semicolon()) {
		return nil, false
	}
	exprBody.SetPosition(n.getPosition(expressionFunctionBodyNode))
	return exprBody, true
}

func (n *nodeBuilder) transformExplicitNewExpression(explicitNewExpression *st.ExplicitNewExpressionNode) (*ast.BLangNewExpression, bool) {
	if !n.expectNode(explicitNewExpression) || !n.expectToken(explicitNewExpression.NewKeyword()) {
		return nil, false
	}
	typeInit := &ast.BLangNewExpression{}
	typeInit.SetPosition(n.getPosition(explicitNewExpression))
	typeInit.TypeDescriptor = n.transformType(explicitNewExpression.TypeDescriptor())
	argList := explicitNewExpression.ParenthesizedArgList()
	if argList == nil {
		n.internalError("explicit new expression has no argument list", explicitNewExpression)
		return nil, false
	}
	args, ok := n.transformParenthesizedArgList(argList)
	if !ok {
		return nil, false
	}
	typeInit.ArgsExprs = args
	return typeInit, true
}

func (n *nodeBuilder) transformImplicitNewExpression(implicitNewExpression *st.ImplicitNewExpressionNode) (*ast.BLangNewExpression, bool) {
	if !n.expectNode(implicitNewExpression) || !n.expectToken(implicitNewExpression.NewKeyword()) {
		return nil, false
	}
	typeInit := &ast.BLangNewExpression{}
	typeInit.SetPosition(n.getPosition(implicitNewExpression))
	if argList := implicitNewExpression.ParenthesizedArgList(); argList != nil {
		args, ok := n.transformParenthesizedArgList(argList)
		if !ok {
			return nil, false
		}
		typeInit.ArgsExprs = args
	}
	return typeInit, true
}

func (n *nodeBuilder) expectKeySpecifier(keySpecifier *st.KeySpecifierNode) bool {
	if keySpecifier == nil {
		return true
	}
	if !n.expectNode(keySpecifier) || !n.expectToken(keySpecifier.KeyKeyword()) || !n.expectToken(keySpecifier.OpenParenToken()) {
		return false
	}
	fieldNames := keySpecifier.FieldNames()
	for fieldName, separator := range fieldNames.SeparatedElements() {
		if !n.expectToken(fieldName) || !n.expectOptionalToken(separator) {
			return false
		}
	}
	return n.expectToken(keySpecifier.CloseParenToken())
}

func (n *nodeBuilder) transformFromClause(fromClauseNode *st.FromClauseNode) (*ast.BLangFromClause, bool) {
	if !n.expectNode(fromClauseNode) || !n.expectToken(fromClauseNode.FromKeyword()) {
		return nil, false
	}
	bindingPatternNode := fromClauseNode.TypedBindingPattern()
	varDef, ok := n.createBLangVarDef(n.getPosition(bindingPatternNode), bindingPatternNode, nil, nil, false, nil)
	if !ok || !n.expectToken(fromClauseNode.InKeyword()) {
		return nil, false
	}
	return ast.NewBLangFromClause(
		n.getPosition(fromClauseNode),
		n.transformExpr(fromClauseNode.Expression()),
		varDef,
		isDeclaredWithVar(bindingPatternNode.TypeDescriptor()),
	), true
}

func (n *nodeBuilder) transformWhereClause(whereClauseNode *st.WhereClauseNode) (*ast.BLangWhereClause, bool) {
	if !n.expectNode(whereClauseNode) || !n.expectToken(whereClauseNode.WhereKeyword()) {
		return nil, false
	}
	whereClause := &ast.BLangWhereClause{}
	whereClause.SetPosition(n.getPosition(whereClauseNode))
	whereClause.Expression = n.transformExpr(whereClauseNode.Expression())
	return whereClause, true
}

func (n *nodeBuilder) transformLetClause(letClauseNode *st.LetClauseNode) (*ast.BLangLetClause, bool) {
	if !n.expectNode(letClauseNode) || !n.expectToken(letClauseNode.LetKeyword()) {
		return nil, false
	}
	letClause := &ast.BLangLetClause{}
	letClause.SetPosition(n.getPosition(letClauseNode))
	letVarDeclarations := letClauseNode.LetVarDeclarations()
	letClause.LetVarDeclarations = make([]ast.BLangVariableDef, 0, letVarDeclarations.Size())
	for letVar, separator := range letVarDeclarations.SeparatedElements() {
		varDef, ok := n.transformLetVariableDeclaration(letVar)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		letClause.LetVarDeclarations = append(letClause.LetVarDeclarations, *varDef)
	}
	return letClause, true
}

func (n *nodeBuilder) transformJoinClause(joinClauseNode *st.JoinClauseNode) (*ast.BLangJoinClause, bool) {
	outerKeyword := joinClauseNode.OuterKeyword()
	if !n.expectNode(joinClauseNode) || !n.expectOptionalSignificantToken(outerKeyword) || !n.expectToken(joinClauseNode.JoinKeyword()) {
		return nil, false
	}
	bindingPatternNode := joinClauseNode.TypedBindingPattern()
	varDef, ok := n.createBLangVarDef(n.getPosition(bindingPatternNode), bindingPatternNode, nil, nil, false, nil)
	if !ok || !n.expectToken(joinClauseNode.InKeyword()) {
		return nil, false
	}
	expr := n.transformExpr(joinClauseNode.Expression())
	onClauseNode := joinClauseNode.JoinOnCondition()
	if onClauseNode == nil {
		n.internalError("join clause has no on condition", joinClauseNode)
		return nil, false
	}
	onClause, ok := n.transformOnClause(onClauseNode)
	if !ok {
		return nil, false
	}
	return ast.NewBLangJoinClause(
		n.getPosition(joinClauseNode),
		expr,
		varDef,
		isDeclaredWithVar(bindingPatternNode.TypeDescriptor()),
		outerKeyword != nil,
		onClause,
	), true
}

func (n *nodeBuilder) transformOnClause(onClauseNode *st.OnClauseNode) (*ast.BLangOnClause, bool) {
	if !n.expectNode(onClauseNode) || !n.expectToken(onClauseNode.OnKeyword()) {
		return nil, false
	}
	onExpr := n.transformExpr(onClauseNode.OnExpression())
	if !n.expectToken(onClauseNode.EqualsKeyword()) {
		return nil, false
	}
	return ast.NewBLangOnClause(n.getPosition(onClauseNode), onExpr, n.transformExpr(onClauseNode.EqualsExpression())), true
}

func (n *nodeBuilder) transformLimitClause(limitClauseNode *st.LimitClauseNode) (*ast.BLangLimitClause, bool) {
	if !n.expectNode(limitClauseNode) || !n.expectToken(limitClauseNode.LimitKeyword()) {
		return nil, false
	}
	return ast.NewBLangLimitClause(n.getPosition(limitClauseNode), n.transformExpr(limitClauseNode.Expression())), true
}

func (n *nodeBuilder) transformOnConflictClause(onConflictClauseNode *st.OnConflictClauseNode) (*ast.BLangOnConflictClause, bool) {
	if !n.expectNode(onConflictClauseNode) || !n.expectToken(onConflictClauseNode.OnKeyword()) ||
		!n.expectToken(onConflictClauseNode.ConflictKeyword()) {
		return nil, false
	}
	return ast.NewBLangOnConflictClause(n.getPosition(onConflictClauseNode), n.transformExpr(onConflictClauseNode.Expression())), true
}

func (n *nodeBuilder) transformSelectClause(selectClauseNode *st.SelectClauseNode) (*ast.BLangSelectClause, bool) {
	if !n.expectNode(selectClauseNode) || !n.expectToken(selectClauseNode.SelectKeyword()) {
		return nil, false
	}
	return ast.NewBLangSelectClause(n.getPosition(selectClauseNode), n.transformExpr(selectClauseNode.Expression())), true
}

func (n *nodeBuilder) transformCollectClause(collectClauseNode *st.CollectClauseNode) (*ast.BLangCollectClause, bool) {
	if !n.expectNode(collectClauseNode) || !n.expectToken(collectClauseNode.CollectKeyword()) {
		return nil, false
	}
	return ast.NewBLangCollectClause(
		n.getPosition(collectClauseNode),
		n.transformExpr(collectClauseNode.Expression()),
		&balCommon.UnorderedSet[string]{},
	), true
}

func (n *nodeBuilder) transformQueryExpression(queryExpression *st.QueryExpressionNode) (*ast.BLangQueryExpr, bool) {
	if !n.expectNode(queryExpression) {
		return nil, false
	}
	queryExpr := &ast.BLangQueryExpr{}
	queryExpr.SetPosition(n.getPosition(queryExpression))

	if constructType := queryExpression.QueryConstructType(); constructType != nil {
		keyword := constructType.Keyword()
		if !n.expectNode(constructType) || !n.expectSignificantToken(keyword) || !n.expectKeySpecifier(constructType.KeySpecifier()) {
			return nil, false
		}
		switch keyword.Text() {
		case ast.TypeKindMap.String():
			queryExpr.QueryConstructType = ast.TypeKindMap
		default:
			n.cx.Unimplemented("only map query construct type is supported for now", n.getPosition(constructType))
			return nil, false
		}
	}

	queryPipeline := queryExpression.QueryPipeline()
	if queryPipeline == nil || queryPipeline.FromClause() == nil {
		n.internalError("query expression has no from clause", queryExpression)
		return nil, false
	}
	if !n.expectNode(queryPipeline) {
		return nil, false
	}
	fromClause, ok := n.transformFromClause(queryPipeline.FromClause())
	if !ok {
		return nil, false
	}
	queryExpr.AddQueryClause(fromClause)

	intermediateClauses := queryPipeline.IntermediateClauses()
	for clause := range intermediateClauses.Iterator() {
		transformed, ok := n.transformIntermediateClause(clause)
		if !ok {
			return nil, false
		}
		queryExpr.AddQueryClause(transformed)
	}

	resultClause := queryExpression.ResultClause()
	if resultClause == nil {
		n.internalError("query expression has no result clause", queryExpression)
		return nil, false
	}
	transformedResult, ok := n.transformResultClause(resultClause)
	if !ok {
		return nil, false
	}
	queryExpr.AddQueryClause(transformedResult)

	if onConflictClauseNode := queryExpression.OnConflictClause(); onConflictClauseNode != nil {
		onConflictClause, ok := n.transformOnConflictClause(onConflictClauseNode)
		if !ok {
			return nil, false
		}
		queryExpr.AddQueryClause(onConflictClause)
	}

	return queryExpr, true
}

func (n *nodeBuilder) transformIntermediateClause(clause st.Node) (ast.BLangNode, bool) {
	switch clause := clause.(type) {
	case *st.FromClauseNode:
		return n.transformFromClause(clause)
	case *st.JoinClauseNode:
		return n.transformJoinClause(clause)
	case *st.LetClauseNode:
		return n.transformLetClause(clause)
	case *st.WhereClauseNode:
		return n.transformWhereClause(clause)
	case *st.GroupByClauseNode:
		return n.transformGroupByClause(clause)
	case *st.LimitClauseNode:
		return n.transformLimitClause(clause)
	case *st.OrderByClauseNode:
		return n.transformOrderByClause(clause)
	default:
		n.cx.Unimplemented("only from + join + let + where + group by + order by + limit + select/collect query clauses are supported for now", n.getPosition(clause))
		return nil, false
	}
}

func (n *nodeBuilder) transformResultClause(clause st.Node) (ast.BLangNode, bool) {
	switch clause := clause.(type) {
	case *st.SelectClauseNode:
		return n.transformSelectClause(clause)
	case *st.CollectClauseNode:
		return n.transformCollectClause(clause)
	default:
		n.cx.Unimplemented("only select/collect result clauses are supported for now", n.getPosition(clause))
		return nil, false
	}
}

func (n *nodeBuilder) transformQueryAction(queryActionNode *st.QueryActionNode) (ast.BLangActionOrExpression, bool) {
	if !n.expectNode(queryActionNode) {
		return nil, false
	}
	n.unimplemented("transformQueryAction unimplemented", queryActionNode)
	return nil, false
}

func (n *nodeBuilder) transformOrderByClause(orderByClauseNode *st.OrderByClauseNode) (*ast.BLangOrderByClause, bool) {
	if !n.expectNode(orderByClauseNode) || !n.expectToken(orderByClauseNode.OrderKeyword()) ||
		!n.expectToken(orderByClauseNode.ByKeyword()) {
		return nil, false
	}
	orderByClause := &ast.BLangOrderByClause{}
	orderByClause.SetPosition(n.getPosition(orderByClauseNode))
	orderKeys := orderByClauseNode.OrderKey()
	orderByClause.OrderByKeyList = make([]ast.BLangOrderKey, 0, orderKeys.Size())
	for orderKey, separator := range orderKeys.SeparatedElements() {
		keyNode, ok := n.transformOrderKey(orderKey)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		orderByClause.OrderByKeyList = append(orderByClause.OrderByKeyList, *keyNode)
	}
	return orderByClause, true
}

func (n *nodeBuilder) transformOrderKey(orderKeyNode *st.OrderKeyNode) (*ast.BLangOrderKey, bool) {
	if !n.expectNode(orderKeyNode) {
		return nil, false
	}
	orderKey := &ast.BLangOrderKey{}
	orderKey.SetPosition(n.getPosition(orderKeyNode))
	orderKey.Expression = n.transformExpr(orderKeyNode.Expression())
	dir := orderKeyNode.OrderDirection()
	if !n.expectOptionalSignificantToken(dir) {
		return nil, false
	}
	orderKey.IsDescending = dir != nil && dir.Kind() == st.DESCENDING_KEYWORD
	return orderKey, true
}

func (n *nodeBuilder) transformGroupByClause(groupByClauseNode *st.GroupByClauseNode) (*ast.BLangGroupByClause, bool) {
	if !n.expectNode(groupByClauseNode) || !n.expectToken(groupByClauseNode.GroupKeyword()) ||
		!n.expectToken(groupByClauseNode.ByKeyword()) {
		return nil, false
	}
	groupByClause := &ast.BLangGroupByClause{
		NonGroupingKeys: &balCommon.UnorderedSet[string]{},
	}
	groupByClause.SetPosition(n.getPosition(groupByClauseNode))
	groupingKeys := groupByClauseNode.GroupingKey()
	for node, separator := range groupingKeys.SeparatedElements() {
		groupingKey, ok := n.transformGroupingKey(node)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		groupByClause.AddGroupingKey(groupingKey)
	}
	return groupByClause, true
}

func (n *nodeBuilder) transformGroupingKey(node st.Node) (*ast.BLangGroupingKey, bool) {
	switch node := node.(type) {
	case *st.GroupingKeyVarDeclarationNode:
		return n.transformGroupingKeyVarDeclaration(node)
	case *st.SimpleNameReferenceNode, st.Token:
		varRef, ok := n.createVarRef(node)
		if !ok {
			return nil, false
		}
		return ast.NewBLangGroupingKeyWithVariableRef(n.getPosition(node), varRef), true
	default:
		n.internalError("expected grouping key variable reference to be a simple variable reference", node)
		return nil, false
	}
}

func (n *nodeBuilder) transformGroupingKeyVarDeclaration(groupingKeyVarDeclarationNode *st.GroupingKeyVarDeclarationNode) (*ast.BLangGroupingKey, bool) {
	if !n.expectNode(groupingKeyVarDeclarationNode) {
		return nil, false
	}
	pos := n.getPosition(groupingKeyVarDeclarationNode)
	typeDesc := groupingKeyVarDeclarationNode.TypeDescriptor()
	declaredWithVar := isDeclaredWithVar(typeDesc)
	var typeNode ast.BType
	if !declaredWithVar {
		typeNode = n.transformType(typeDesc)
	} else if !n.transformVarTypeKeyword(typeDesc) {
		return nil, false
	}
	simpleVar, ok := n.getBLangVariableNode(groupingKeyVarDeclarationNode.SimpleBindingPattern(), pos)
	if !ok || !n.expectToken(groupingKeyVarDeclarationNode.EqualsToken()) {
		return nil, false
	}
	expr := n.transformExpr(groupingKeyVarDeclarationNode.Expression())
	simpleVar = ast.NewBLangVariable(pos, simpleVar.Name, typeNode, expr, declaredWithVar, 0)
	simpleVar.SetPosition(pos)
	varDef := &ast.BLangVariableDef{Var: simpleVar}
	varDef.SetPosition(pos)
	return ast.NewBLangGroupingKeyWithVariableDef(pos, varDef), true
}

func (n *nodeBuilder) transformImplicitAnonymousFunctionExpression(node *st.ImplicitAnonymousFunctionExpressionNode) (*ast.BLangLambdaFunction, bool) {
	if !n.expectNode(node) {
		return nil, false
	}
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	fn := ast.NewBLangFunction(ast.InvokableData{
		Position: n.getPosition(node),
		Name:     &ident,
		Flags:    model.FlagLambda | model.FlagAnonymous,
	})

	params, ok := n.implicitAnonymousFunctionParams(node.Params())
	if !ok {
		return nil, false
	}
	fn.RequiredParams = make([]ast.BLangVariable, len(params))
	for i, param := range params {
		fn.RequiredParams[i].Name = param.name
		fn.RequiredParams[i].SetPosition(n.getPosition(param.node))
		fn.RequiredParams[i].SetRequiredParam()
	}
	if !n.expectToken(node.RightDoubleArrow()) {
		return nil, false
	}
	body := &ast.BLangExprFunctionBody{
		Expr: n.transformExpr(node.Expression()),
	}
	body.SetPosition(n.getPosition(node.Expression()))
	fn.Body = body

	lambda := &ast.BLangLambdaFunction{Function: fn}
	lambda.SetInferredParams()
	lambda.SetPosition(fn.GetPosition())
	return lambda, true
}

type inferredParam struct {
	node *st.SimpleNameReferenceNode
	name ast.IdentifierNode
}

func (n *nodeBuilder) implicitAnonymousFunctionParams(params st.Node) ([]inferredParam, bool) {
	switch params := params.(type) {
	case *st.SimpleNameReferenceNode:
		param, ok := n.transformInferredParam(params)
		if !ok {
			return nil, false
		}
		return []inferredParam{param}, true
	case *st.ImplicitAnonymousFunctionParameters:
		if !n.expectNode(params) || !n.expectToken(params.OpenParenToken()) {
			return nil, false
		}
		var result []inferredParam
		parameters := params.Parameters()
		for paramNode, separator := range parameters.SeparatedElements() {
			param, ok := n.transformInferredParam(paramNode)
			if !ok || !n.expectOptionalToken(separator) {
				return nil, false
			}
			result = append(result, param)
		}
		if !n.expectToken(params.CloseParenToken()) {
			return nil, false
		}
		return result, true
	default:
		if !n.expectVerbatimNodes(params) {
			return nil, false
		}
		n.cx.SyntaxError("invalid parameter list in inferred anonymous function expression", n.getPosition(params))
		return nil, true
	}
}

func (n *nodeBuilder) transformInferredParam(paramNode *st.SimpleNameReferenceNode) (inferredParam, bool) {
	if !n.expectNode(paramNode) {
		return inferredParam{}, false
	}
	return inferredParam{node: paramNode, name: n.transformIdentifierWith(paramNode.Name(), nil, "'_' cannot be used as an identifier")}, true
}

func (n *nodeBuilder) transformStartAction(startActionNode *st.StartActionNode) (*ast.BLangStartAction, bool) {
	if !n.expectNode(startActionNode) {
		return nil, false
	}
	annotations := startActionNode.Annotations()
	if annotations.Size() > 0 {
		n.cx.Unimplemented("annotations on start actions are not yet supported", n.getPosition(annotations.Get(0)))
		return nil, false
	}
	if !n.expectToken(startActionNode.StartKeyword()) {
		return nil, false
	}
	call, ok := n.transformStartCall(startActionNode.Expression())
	if !ok {
		return nil, false
	}
	action := &ast.BLangStartAction{Call: call}
	action.SetPosition(n.getPosition(startActionNode))
	return action, true
}

func (n *nodeBuilder) transformStartCall(call st.Node) (ast.Invocable, bool) {
	switch call := call.(type) {
	case *st.FunctionCallExpressionNode:
		return n.transformFunctionCallExpression(call)
	case *st.MethodCallExpressionNode:
		return n.transformMethodCallExpression(call)
	case *st.RemoteMethodCallActionNode:
		return n.transformRemoteMethodCallAction(call)
	case *st.ClientResourceAccessActionNode:
		return n.transformClientResourceAccessAction(call)
	default:
		n.cx.InternalError("start action operand is not invocable", n.getPosition(call))
		return nil, false
	}
}

func (n *nodeBuilder) transformFlushAction(flushActionNode *st.FlushActionNode) (ast.BLangActionOrExpression, bool) {
	if !n.expectNode(flushActionNode) {
		return nil, false
	}
	n.unimplemented("transformFlushAction unimplemented", flushActionNode)
	return nil, false
}

func (n *nodeBuilder) transformAsyncSendAction(asyncSendActionNode *st.AsyncSendActionNode) (ast.BLangActionOrExpression, bool) {
	if !n.expectNode(asyncSendActionNode) {
		return nil, false
	}
	n.unimplemented("transformAsyncSendAction unimplemented", asyncSendActionNode)
	return nil, false
}

func (n *nodeBuilder) transformSyncSendAction(syncSendActionNode *st.SyncSendActionNode) (ast.BLangActionOrExpression, bool) {
	if !n.expectNode(syncSendActionNode) {
		return nil, false
	}
	n.unimplemented("transformSyncSendAction unimplemented", syncSendActionNode)
	return nil, false
}

func (n *nodeBuilder) transformReceiveAction(receiveActionNode *st.ReceiveActionNode) (ast.BLangActionOrExpression, bool) {
	if !n.expectNode(receiveActionNode) {
		return nil, false
	}
	n.unimplemented("transformReceiveAction unimplemented", receiveActionNode)
	return nil, false
}

func (n *nodeBuilder) transformCommitAction(commitActionNode *st.CommitActionNode) (ast.BLangActionOrExpression, bool) {
	if !n.expectNode(commitActionNode) {
		return nil, false
	}
	n.unimplemented("transformCommitAction unimplemented", commitActionNode)
	return nil, false
}

func (n *nodeBuilder) transformWaitAction(waitActionNode *st.WaitActionNode) (ast.BLangAction, bool) {
	if !n.expectNode(waitActionNode) || !n.expectToken(waitActionNode.WaitKeyword()) {
		return nil, false
	}
	switch futureExpr := waitActionNode.WaitFutureExpr().(type) {
	case *st.ExternalTreeNodeList:
		futureExprs := make([]ast.BLangExpression, 0)
		for child := range futureExpr.ChildNodes() {
			if tok, ok := child.(st.Token); ok {
				if !n.expectToken(tok) {
					return nil, false
				}
				continue
			}
			futureExprs = append(futureExprs, n.transformExpr(child))
		}
		action := &ast.BLangAlternateWaitAction{FutureExprs: futureExprs}
		action.SetPosition(n.getPosition(waitActionNode))
		return action, true
	case *st.WaitFieldsListNode:
		return n.transformWaitFieldsList(waitActionNode, futureExpr)
	default:
		action := &ast.BLangSingleWaitAction{FutureExpr: n.transformExpr(futureExpr)}
		action.SetPosition(n.getPosition(waitActionNode))
		return action, true
	}
}

func (n *nodeBuilder) transformWaitFieldsList(waitActionNode *st.WaitActionNode, waitFieldsListNode *st.WaitFieldsListNode) (*ast.BLangMultipleWaitAction, bool) {
	if !n.expectNode(waitFieldsListNode) || !n.expectToken(waitFieldsListNode.OpenBrace()) {
		return nil, false
	}
	fields := waitFieldsListNode.WaitFields()
	futureExprs := make([]ast.BLangExpression, 0, (fields.Size()+1)/2)
	fieldNames := make([]string, 0, (fields.Size()+1)/2)
	for field, separator := range fields.SeparatedElements() {
		fieldName, expr, ok := n.transformWaitField(field)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		futureExprs = append(futureExprs, expr)
		fieldNames = append(fieldNames, fieldName)
	}
	if !n.expectToken(waitFieldsListNode.CloseBrace()) {
		return nil, false
	}
	action := &ast.BLangMultipleWaitAction{FutureExprs: futureExprs, FieldNames: fieldNames}
	action.SetPosition(n.getPosition(waitActionNode))
	return action, true
}

func (n *nodeBuilder) transformWaitField(field st.Node) (string, ast.BLangExpression, bool) {
	switch field := field.(type) {
	case *st.SimpleNameReferenceNode:
		return identifierTokenValue(field.Name()), n.transformExpr(field), true
	case *st.WaitFieldNode:
		nameRef := field.FieldName()
		if !n.expectNode(field) || !n.expectNode(nameRef) {
			return "", nil, false
		}
		identifier, ok := n.createBLangIdentifier(nameRef.Name())
		if !ok || !n.expectToken(field.Colon()) {
			return "", nil, false
		}
		return identifier.Value, n.transformExpr(field.WaitFutureExpr()), true
	default:
		n.cx.InternalError("unexpected multiple wait field", n.getPosition(field))
		return "", nil, false
	}
}

func (n *nodeBuilder) transformAnnotAccessExpression(annotAccessExpression *st.AnnotAccessExpressionNode) (*ast.BLangAnnotAccessExpr, bool) {
	if !n.expectNode(annotAccessExpression) {
		return nil, false
	}
	expr := &ast.BLangAnnotAccessExpr{}
	expr.Expr = n.transformExpr(annotAccessExpression.Expression())
	if !n.expectToken(annotAccessExpression.AnnotChainingToken()) {
		return nil, false
	}
	pkgAlias, name, ok := n.createBLangNameReference(annotAccessExpression.AnnotTagReference())
	if !ok {
		return nil, false
	}
	expr.PkgAlias = pkgAlias
	expr.AnnotationName = name
	expr.SetPosition(n.getPosition(annotAccessExpression))
	return expr, true
}

func (n *nodeBuilder) transformOptionalFieldAccessExpression(optionalFieldAccessExpression *st.OptionalFieldAccessExpressionNode) (*ast.BLangFieldBaseAccess, bool) {
	if !n.expectNode(optionalFieldAccessExpression) {
		return nil, false
	}
	expr, ok := n.transformAccessContainer(optionalFieldAccessExpression.Expression())
	if !ok || !n.expectToken(optionalFieldAccessExpression.OptionalChainingToken()) {
		return nil, false
	}
	fieldName := optionalFieldAccessExpression.FieldName()
	if qualified, ok := fieldName.(*st.QualifiedNameReferenceNode); ok {
		n.unsupportedQualifiedFieldName("transformOptionalFieldAccessExpression: QUALIFIED_NAME_REFERENCE unsupported", qualified)
		return nil, false
	}
	field, ok := n.transformFieldName(fieldName)
	if !ok {
		return nil, false
	}
	return ast.NewBLangFieldBaseAccess(n.getPosition(optionalFieldAccessExpression), expr, field, true), true
}

func (n *nodeBuilder) transformConditionalExpression(conditionalExpression *st.ConditionalExpressionNode) (*ast.BLangTernaryExpr, bool) {
	if !n.expectNode(conditionalExpression) {
		return nil, false
	}
	lhsExpr := n.transformExpr(conditionalExpression.LhsExpression())
	if !n.expectToken(conditionalExpression.QuestionMarkToken()) {
		return nil, false
	}
	middleExpr := n.transformExpr(conditionalExpression.MiddleExpression())
	if !n.expectToken(conditionalExpression.ColonToken()) {
		return nil, false
	}
	return ast.NewBLangTernaryExpr(
		n.getPosition(conditionalExpression),
		lhsExpr,
		middleExpr,
		n.transformExpr(conditionalExpression.EndExpression()),
	), true
}

func (n *nodeBuilder) transformTransactionalExpression(transactionalExpression *st.TransactionalExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(transactionalExpression) {
		return nil, false
	}
	n.unimplemented("transformTransactionalExpression unimplemented", transactionalExpression)
	return nil, false
}

func (n *nodeBuilder) transformByteArrayLiteral(byteArrayLiteralNode *st.ByteArrayLiteralNode) (ast.BLangExpression, bool) {
	if !n.expectNode(byteArrayLiteralNode) {
		return nil, false
	}
	n.unimplemented("transformByteArrayLiteral unimplemented", byteArrayLiteralNode)
	return nil, false
}

func (n *nodeBuilder) transformRequiredExpression(requiredExpression *st.RequiredExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(requiredExpression) {
		return nil, false
	}
	n.unimplemented("transformRequiredExpression unimplemented", requiredExpression)
	return nil, false
}

func (n *nodeBuilder) transformNaturalExpression(naturalExpression *st.NaturalExpressionNode) (ast.BLangExpression, bool) {
	if !n.expectNode(naturalExpression) {
		return nil, false
	}
	n.unimplemented("transformNaturalExpression unimplemented", naturalExpression)
	return nil, false
}

func (n *nodeBuilder) transformXMLFilterExpression(node *st.XMLFilterExpressionNode) (*ast.BLangXMLFilterExpression, bool) {
	if !n.expectNode(node) {
		return nil, false
	}
	expression := n.transformExpr(node.Expression())
	patterns, ok := n.createXMLNamePatterns(node.XmlPatternChain())
	if !ok {
		return nil, false
	}
	expr := &ast.BLangXMLFilterExpression{
		Expression:  expression,
		NamePattern: patterns,
	}
	expr.SetPosition(n.getPosition(node))
	return expr, true
}

func (n *nodeBuilder) createXMLNamePatterns(chain *st.XMLNamePatternChainingNode) ([]ast.BLangAtomicNamePattern, bool) {
	if !n.expectNode(chain) || !n.expectToken(chain.StartToken()) {
		return nil, false
	}
	patterns := make([]ast.BLangAtomicNamePattern, 0)
	patternNodes := chain.XmlNamePattern()
	for node, separator := range patternNodes.SeparatedElements() {
		pattern, ok := n.createXMLAtomicNamePattern(node)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		patterns = append(patterns, pattern)
	}
	if !n.expectToken(chain.GtToken()) {
		return nil, false
	}
	return patterns, true
}

func (n *nodeBuilder) transformXMLIdentifier(token st.Token) ast.IdentifierNode {
	return n.transformIdentifierWith(token, unescapeIdentifier, underscoreKeywordMessage)
}

func (n *nodeBuilder) createXMLAtomicNamePattern(node st.Node) (ast.BLangAtomicNamePattern, bool) {
	switch node := node.(type) {
	case st.Token:
		if !n.expectSignificantToken(node) {
			return ast.BLangAtomicNamePattern{}, false
		}
		if node.Kind() != st.ASTERISK_TOKEN {
			n.internalError("unexpected XML name pattern token", node)
			return ast.BLangAtomicNamePattern{}, false
		}
		return ast.BLangAtomicNamePattern{Kind: ast.NamePatternKindWildCard}, true
	case *st.SimpleNameReferenceNode:
		if !n.expectNode(node) {
			return ast.BLangAtomicNamePattern{}, false
		}
		return ast.BLangAtomicNamePattern{
			Kind:       ast.NamePatternKindIdentifier,
			Identifier: n.transformXMLIdentifier(node.Name()),
		}, true
	case *st.XMLAtomicNamePatternNode:
		if !n.expectNode(node) {
			return ast.BLangAtomicNamePattern{}, false
		}
		namespacePrefix := n.transformXMLIdentifier(node.Prefix())
		if !n.expectToken(node.Colon()) {
			return ast.BLangAtomicNamePattern{}, false
		}
		kind := ast.NamePatternKindQualifiedIdentifier
		var identifier ast.IdentifierNode
		if name := node.Name(); name.Kind() == st.ASTERISK_TOKEN {
			if !n.expectToken(name) {
				return ast.BLangAtomicNamePattern{}, false
			}
			kind = ast.NamePatternKindPrefix
		} else {
			identifier = n.transformXMLIdentifier(name)
		}
		return ast.BLangAtomicNamePattern{
			Kind:            kind,
			Identifier:      identifier,
			NamespacePrefix: namespacePrefix,
		}, true
	default:
		n.internalError(fmt.Sprintf("unexpected XML name pattern node %T", node), node)
		return ast.BLangAtomicNamePattern{}, false
	}
}

func (n *nodeBuilder) transformXMLStepExpression(node *st.XMLStepExpressionNode) (*ast.BLangXMLStepExpression, bool) {
	if !n.expectNode(node) {
		return nil, false
	}
	expr := &ast.BLangXMLStepExpression{
		Expression: n.transformExpr(node.Expression()),
		Extensions: make([]ast.XMLStepExtend, 0),
	}
	expr.SetPosition(n.getPosition(node))

	switch start := node.XmlStepStart().(type) {
	case st.Token:
		if !n.expectSignificantToken(start) {
			return nil, false
		}
		if start.Kind() != st.SLASH_ASTERISK_TOKEN {
			n.internalError("unexpected XML step start token", start)
			return nil, false
		}
		expr.Start = &ast.BLangXMLStepStart{Kind: ast.XMLStepStartKindAllChildren}
		expr.Start.SetPosition(n.getPosition(start))
	case *st.XMLNamePatternChainingNode:
		kind := ast.XMLStepStartKindElementChildren
		if start.StartToken().Kind() == st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN {
			kind = ast.XMLStepStartKindElementDescendants
		}
		patterns, ok := n.createXMLNamePatterns(start)
		if !ok {
			return nil, false
		}
		expr.Start = &ast.BLangXMLStepStart{
			Kind:        kind,
			NamePattern: patterns,
		}
		expr.Start.SetPosition(n.getPosition(start))
	default:
		n.internalError(fmt.Sprintf("unexpected XML step start node %T", start), node.XmlStepStart())
		return nil, false
	}

	extensions := node.XmlStepExtend()
	for extension := range extensions.Iterator() {
		transformed, ok := n.transformXMLStepExtend(extension)
		if !ok {
			return nil, false
		}
		expr.Extensions = append(expr.Extensions, transformed)
	}
	return expr, true
}

func (n *nodeBuilder) transformXMLStepExtend(extension st.Node) (ast.XMLStepExtend, bool) {
	switch extension := extension.(type) {
	case *st.XMLNamePatternChainingNode:
		patterns, ok := n.createXMLNamePatterns(extension)
		if !ok {
			return nil, false
		}
		filter := &ast.BLangXMLStepFilterExtend{NamePattern: patterns}
		filter.SetPosition(n.getPosition(extension))
		return filter, true
	case *st.XMLStepIndexedExtendNode:
		return n.transformXMLStepIndexedExtend(extension)
	case *st.XMLStepMethodCallExtendNode:
		return n.transformXMLStepMethodCallExtend(extension)
	default:
		n.internalError(fmt.Sprintf("unexpected XML step extension node %T", extension), extension)
		return nil, false
	}
}

func (n *nodeBuilder) transformXMLStepIndexedExtend(node *st.XMLStepIndexedExtendNode) (*ast.BLangXMLStepIndexExtend, bool) {
	if !n.expectNode(node) || !n.expectToken(node.OpenBracket()) {
		return nil, false
	}
	extension := &ast.BLangXMLStepIndexExtend{Expression: n.transformExpr(node.Expression())}
	if !n.expectToken(node.CloseBracket()) {
		return nil, false
	}
	extension.SetPosition(n.getPosition(node))
	return extension, true
}

func (n *nodeBuilder) transformXMLStepMethodCallExtend(node *st.XMLStepMethodCallExtendNode) (*ast.BLangXMLStepMethodCallExtend, bool) {
	if !n.expectNode(node) || !n.expectToken(node.DotToken()) {
		return nil, false
	}
	pkgAlias, name, ok := n.createBLangNameReference(node.MethodName())
	if !ok {
		return nil, false
	}
	args, ok := n.transformParenthesizedArgList(node.ParenthesizedArgList())
	if !ok {
		return nil, false
	}
	invocation := &ast.BLangInvocation{}
	invocation.PkgAlias = pkgAlias
	invocation.Name = name
	invocation.ArgExprs = args
	invocation.SetPosition(n.getPosition(node))
	extension := &ast.BLangXMLStepMethodCallExtend{Invocation: invocation}
	extension.SetPosition(n.getPosition(node))
	return extension, true
}

func (n *nodeBuilder) transformErrorConstructorExpression(errorConstructorExpression *st.ErrorConstructorExpressionNode) (*ast.BLangErrorConstructorExpr, bool) {
	if !n.expectNode(errorConstructorExpression) || !n.expectToken(errorConstructorExpression.ErrorKeyword()) {
		return nil, false
	}
	result := &ast.BLangErrorConstructorExpr{}
	result.SetPosition(n.getPosition(errorConstructorExpression))

	if typeRefNode := errorConstructorExpression.TypeReference(); typeRefNode != nil {
		userDefinedType, ok := n.transformTypeReferenceName(typeRefNode)
		if !ok {
			return nil, false
		}
		result.ErrorTypeRef = userDefinedType
	}
	if !n.expectToken(errorConstructorExpression.OpenParenToken()) {
		return nil, false
	}

	positionalArgs := make([]ast.BLangExpression, 0)
	namedArgs := make([]ast.BLangNamedArgsExpression, 0)
	arguments := errorConstructorExpression.Arguments()
	for arg, separator := range arguments.SeparatedElements() {
		switch arg := arg.(type) {
		case *st.PositionalArgumentNode:
			expr, ok := n.transformPositionalArgument(arg)
			if !ok {
				return nil, false
			}
			positionalArgs = append(positionalArgs, expr)
		case *st.NamedArgumentNode:
			namedArg, ok := n.transformNamedArgument(arg)
			if !ok {
				return nil, false
			}
			namedArgs = append(namedArgs, *namedArg)
		case *st.RestArgumentNode:
			n.cx.InternalError("rest arguments not supported in error constructor", n.getPosition(arg))
			return nil, false
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected argument kind: %v", arg.Kind()), n.getPosition(arg))
			return nil, false
		}
		if !n.expectOptionalToken(separator) {
			return nil, false
		}
	}
	if !n.expectToken(errorConstructorExpression.CloseParenToken()) {
		return nil, false
	}

	result.PositionalArgs = positionalArgs
	result.NamedArgs = namedArgs
	return result, true
}

func (n *nodeBuilder) transformClientResourceAccessAction(node *st.ClientResourceAccessActionNode) (*ast.BLangClientResourceAccessAction, bool) {
	if !n.expectNode(node) {
		return nil, false
	}
	action := &ast.BLangClientResourceAccessAction{}
	action.SetPosition(n.getPosition(node))
	action.Expr = n.transformExpr(node.Expression())
	if !n.expectToken(node.RightArrowToken()) || !n.expectToken(node.SlashToken()) {
		return nil, false
	}
	path, ok := n.createResourceAccessSegments(node.ResourceAccessPath())
	if !ok || !n.expectOptionalToken(node.DotToken()) {
		return nil, false
	}
	action.Path = path
	action.MethodName = "get"
	if methodName := node.MethodName(); methodName != nil {
		nameTok := methodName.Name()
		if !n.expectNode(methodName) || !n.expectToken(nameTok) {
			return nil, false
		}
		action.MethodName = nameTok.Text()
	}
	nameID := &ast.BLangIdentifier{Value: action.MethodName}
	nameID.SetPosition(action.GetPosition())
	action.Name = nameID
	if argList := node.Arguments(); argList != nil {
		args, ok := n.transformParenthesizedArgList(argList)
		if !ok {
			return nil, false
		}
		action.ArgExprs = args
	}
	return action, true
}

func (n *nodeBuilder) createResourceAccessSegments(pathNodes st.NodeList[st.Node]) ([]ast.BLangResourceAccessSegment, bool) {
	var segments []ast.BLangResourceAccessSegment
	for node, separator := range pathNodes.SeparatedElements() {
		segment, ok := n.createResourceAccessSegment(node)
		if !ok || !n.expectOptionalToken(separator) {
			return nil, false
		}
		segments = append(segments, segment)
	}
	return segments, true
}

func (n *nodeBuilder) createResourceAccessSegment(node st.Node) (ast.BLangResourceAccessSegment, bool) {
	switch node := node.(type) {
	case *st.ComputedResourceAccessSegmentNode:
		segment, ok := n.transformComputedResourceAccessSegment(node)
		if !ok {
			return ast.BLangResourceAccessSegment{}, false
		}
		return *segment, true
	case *st.ResourceAccessRestSegmentNode:
		if !n.expectNode(node) {
			return ast.BLangResourceAccessSegment{}, false
		}
		n.cx.Unimplemented("resource access rest segments are not yet supported", n.getPosition(node))
		return ast.BLangResourceAccessSegment{}, false
	case st.Token:
		if !n.expectSignificantToken(node) {
			return ast.BLangResourceAccessSegment{}, false
		}
		if node.Kind() != st.IDENTIFIER_TOKEN {
			n.cx.InternalError(fmt.Sprintf("unexpected resource access segment kind: %v", node.Kind()), n.getPosition(node))
			return ast.BLangResourceAccessSegment{}, false
		}
		seg := ast.BLangResourceAccessSegment{Kind: ast.ResourceAccessSegmentName, Name: node.Text()}
		seg.SetPosition(n.getPosition(node))
		return seg, true
	default:
		n.cx.InternalError(fmt.Sprintf("unexpected resource access segment kind: %v", node.Kind()), n.getPosition(node))
		return ast.BLangResourceAccessSegment{}, false
	}
}

func (n *nodeBuilder) transformComputedResourceAccessSegment(node *st.ComputedResourceAccessSegmentNode) (*ast.BLangResourceAccessSegment, bool) {
	if !n.expectNode(node) || !n.expectToken(node.OpenBracketToken()) {
		return nil, false
	}
	seg := &ast.BLangResourceAccessSegment{Kind: ast.ResourceAccessSegmentComputed}
	seg.SetPosition(n.getPosition(node))
	seg.Expr = n.transformExpr(node.Expression())
	if !n.expectToken(node.CloseBracketToken()) {
		return nil, false
	}
	return seg, true
}

func (n *nodeBuilder) objectTypeQualifiers(objectType *ast.BLangObjectType, qualifiers st.NodeList[st.Token]) bool {
	for q := range qualifiers.Iterator() {
		if !n.expectSignificantToken(q) {
			return false
		}
		switch q.Kind() {
		case st.CLIENT_KEYWORD:
			objectType.NetworkQuals = ast.ObjectNetworkQualsClient
		case st.SERVICE_KEYWORD:
			objectType.NetworkQuals = ast.ObjectNetworkQualsService
		case st.ISOLATED_KEYWORD:
			objectType.Isolated = true
		case st.READONLY_KEYWORD:
			// https://github.com/ballerina-nutcracker/ballerina/issues/537",
			n.cx.Unimplemented("readonly object type descriptors are not implemented", n.getPosition(q))
			return false
		default:
			n.internalError("unexpected object type qualifier", q)
			return false
		}
	}
	return true
}

func (n *nodeBuilder) transformObjectTypeDescriptor(objectTypeDescriptorNode *st.ObjectTypeDescriptorNode) (*ast.BLangObjectType, bool) {
	if !n.expectNode(objectTypeDescriptorNode) {
		return nil, false
	}
	objectType := ast.NewBLangObjectType(n.getPosition(objectTypeDescriptorNode))
	if !n.objectTypeQualifiers(objectType, objectTypeDescriptorNode.ObjectTypeQualifiers()) ||
		!n.expectToken(objectTypeDescriptorNode.ObjectKeyword()) || !n.expectToken(objectTypeDescriptorNode.OpenBrace()) {
		return nil, false
	}

	members := objectTypeDescriptorNode.Members()
	for member := range members.Iterator() {
		var ok bool
		switch member := member.(type) {
		case *st.ObjectFieldNode:
			ok = n.addObjectTypeField(objectType, member)
		case *st.MethodDeclarationNode:
			ok = n.addObjectTypeMethod(objectType, member)
		case *st.TypeReferenceNode:
			var inclusion *ast.BLangUserDefinedType
			inclusion, ok = n.transformTypeReference(member)
			if ok {
				objectType.AddUnresolvedInclusion(inclusion)
			}
		default:
			n.rejectNode(member, "unexpected member kind in object type descriptor")
		}
		if !ok {
			objectType.BadMembers = append(objectType.BadMembers, n.badTopLevel(member))
		}
	}

	if !n.expectToken(objectTypeDescriptorNode.CloseBrace()) {
		return nil, false
	}
	return objectType, true
}

func (n *nodeBuilder) addObjectTypeField(objectType *ast.BLangObjectType, objectField *st.ObjectFieldNode) bool {
	if !n.expectNode(objectField) {
		return false
	}
	metadata, ok := n.transformMetadata(objectField.Metadata())
	if !ok {
		return false
	}
	vis := objectField.VisibilityQualifier()
	if !n.expectOptionalSignificantToken(vis) || !n.expectSignificantTokens(objectField.QualifierList()) {
		return false
	}
	fieldType := n.transformType(objectField.TypeName())
	fieldNameToken := objectField.FieldName()
	if !n.expectToken(fieldNameToken) || !n.expectOptionalToken(objectField.EqualsToken()) {
		return false
	}
	if expression := objectField.Expression(); expression != nil {
		n.transformExpr(expression)
	}
	if !n.expectToken(objectField.SemicolonToken()) {
		return false
	}
	fieldName, _ := normalizedIdentifierValue(fieldNameToken.Text())
	bField := ast.NewBObjectField(
		n.getPosition(objectField),
		fieldName,
		fieldType,
		vis != nil && vis.Kind() == st.PUBLIC_KEYWORD,
	)
	applyMetadata(metadata, bField)
	if objectType.AddMember(bField) {
		n.cx.SyntaxError("redeclared symbol '"+fieldName+"'", bField.GetPosition())
	}
	return true
}

type methodQualifiers struct {
	kind          ast.ObjectMemberKind
	isPublic      bool
	functionFlags model.Flag
}

func (n *nodeBuilder) objectMethodQualifiers(qualifiers st.NodeList[st.Token]) (methodQualifiers, bool) {
	result := methodQualifiers{kind: ast.ObjectMemberKindMethod}
	for q := range qualifiers.Iterator() {
		if !n.expectSignificantToken(q) {
			return result, false
		}
		switch q.Kind() {
		case st.PUBLIC_KEYWORD:
			result.isPublic = true
		case st.REMOTE_KEYWORD:
			result.kind = ast.ObjectMemberKindRemoteMethod
		case st.RESOURCE_KEYWORD:
			result.kind = ast.ObjectMemberKindResourceMethod
		case st.ISOLATED_KEYWORD:
			result.functionFlags |= model.FlagIsolated
		case st.TRANSACTIONAL_KEYWORD:
			result.functionFlags |= model.FlagTransactional
		case st.PRIVATE_KEYWORD:
			// Private methods do not require a flag.
		default:
			n.internalError("unexpected object method qualifier", q)
			return result, false
		}
	}
	return result, true
}

func (n *nodeBuilder) addObjectTypeMethod(objectType *ast.BLangObjectType, methodDecl *st.MethodDeclarationNode) bool {
	if !n.expectNode(methodDecl) {
		return false
	}
	if _, ok := n.transformMetadata(methodDecl.Metadata()); !ok {
		return false
	}
	qualifiers, ok := n.objectMethodQualifiers(methodDecl.QualifierList())
	if !ok || !n.expectToken(methodDecl.FunctionKeyword()) {
		return false
	}
	if methodDecl.Kind() != st.METHOD_DECLARATION {
		n.internalError("unexpected member kind in object type descriptor", methodDecl)
		return false
	}
	methodNameToken := asToken(methodDecl.MethodName())
	if !n.expectToken(methodNameToken) {
		return false
	}
	if _, ok := n.createResourcePathSegments(methodDecl.RelativeResourcePath()); !ok {
		return false
	}
	methodName, _ := normalizedIdentifierValue(methodNameToken.Text())
	if qualifiers.kind == ast.ObjectMemberKindRemoteMethod {
		methodName = model.RemoteMethodName(methodName)
	}

	functionFlags := qualifiers.functionFlags
	funcSig := methodDecl.MethodSignature()
	if funcSig == nil {
		n.internalError("method declaration has no signature", methodDecl)
		return false
	}
	signature, ok := n.transformFunctionTypeSignature(funcSig)
	if !ok {
		return false
	}
	if signature.explicitReturnType {
		functionFlags |= model.FlagExplicitReturnTypeDescriptor
	}
	if !n.expectToken(methodDecl.Semicolon()) {
		return false
	}
	bMethod := ast.NewBMethodDecl(n.getPosition(methodDecl), methodName, qualifiers.kind, qualifiers.isPublic, functionFlags)
	bMethod.ParamListPos = signature.paramListPos
	bMethod.RequiredParams = signature.requiredParams
	bMethod.RestParam = signature.restParam
	bMethod.ReturnTypeDescriptor = signature.returnType

	if objectType.AddMember(bMethod) {
		n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(bMethod.Name())+"'", bMethod.GetPosition())
	}
	return true
}

type functionTypeSignature struct {
	requiredParams     []*ast.BLangFunctionTypeParam
	restParam          *ast.BLangFunctionTypeParam
	returnType         ast.BType
	paramListPos       diagnostics.Location
	explicitReturnType bool
}

func newFunctionTypeSignature() functionTypeSignature {
	return functionTypeSignature{paramListPos: diagnostics.NewBuiltinLocation()}
}

func (n *nodeBuilder) transformFunctionTypeSignature(funcSignature *st.FunctionSignatureNode) (functionTypeSignature, bool) {
	signature := newFunctionTypeSignature()
	openParen := funcSignature.OpenParenToken()
	closeParen := funcSignature.CloseParenToken()
	if !n.expectNode(funcSignature) || !n.expectToken(openParen) {
		return signature, false
	}
	parameters := funcSignature.Parameters()
	for param, separator := range parameters.SeparatedElements() {
		ftParam, ok := n.createFunctionTypeParam(param)
		if !ok || !n.expectOptionalToken(separator) {
			return signature, false
		}
		if _, isRestParam := param.(*st.RestParameterNode); isRestParam {
			signature.restParam = ftParam
		} else {
			signature.requiredParams = append(signature.requiredParams, ftParam)
		}
	}
	if !n.expectToken(closeParen) {
		return signature, false
	}
	signature.paramListPos = n.getPositionRange(openParen, closeParen)
	retNode := funcSignature.ReturnTypeDesc()
	if retNode == nil {
		returnType := &ast.BLangValueType{TypeKind: ast.TypeKindNil}
		returnType.SetPosition(diagnostics.NewBuiltinLocation())
		signature.returnType = returnType
		return signature, true
	}
	returnsKeyword := retNode.ReturnsKeyword()
	if !n.expectNode(retNode) || !n.expectToken(returnsKeyword) {
		return signature, false
	}
	signature.explicitReturnType = true
	if _, ok := n.transformAnnotations(retNode.Annotations()); !ok {
		return signature, false
	}
	signature.returnType = n.transformType(retNode.Type())
	return signature, true
}

func (n *nodeBuilder) transformRecordTypeDescriptor(recordTypeDescriptorNode *st.RecordTypeDescriptorNode) (*ast.BLangRecordType, bool) {
	bodyStartDelimiter := recordTypeDescriptorNode.BodyStartDelimiter()
	if !n.expectNode(recordTypeDescriptorNode) || !n.expectToken(recordTypeDescriptorNode.RecordKeyword()) ||
		!n.expectSignificantToken(bodyStartDelimiter) {
		return nil, false
	}
	recordType := &ast.BLangRecordType{}
	fields := recordTypeDescriptorNode.Fields()
	for field := range fields.Iterator() {
		switch field := field.(type) {
		case *st.RecordFieldNode:
			bField, ok := n.transformRecordField(field)
			if !ok {
				return nil, false
			}
			recordType.AddField(string(bField.Name), bField)
		case *st.RecordFieldWithDefaultValueNode:
			bField, ok := n.transformRecordFieldWithDefaultValue(field)
			if !ok {
				return nil, false
			}
			recordType.AddField(string(bField.Name), bField)
		case *st.TypeReferenceNode:
			inclusion, ok := n.transformRecordTypeInclusion(field)
			if !ok {
				return nil, false
			}
			recordType.TypeInclusions = append(recordType.TypeInclusions, inclusion)
		default:
			n.internalError("unexpected field kind in record type descriptor", field)
			return nil, false
		}
	}
	if restDesc := recordTypeDescriptorNode.RecordRestDescriptor(); restDesc != nil {
		restType, ok := n.transformRecordRestDescriptor(restDesc)
		if !ok {
			return nil, false
		}
		recordType.RestType = restType
	}
	if !n.expectToken(recordTypeDescriptorNode.BodyEndDelimiter()) {
		return nil, false
	}
	recordType.IsOpen = bodyStartDelimiter.Kind() == st.OPEN_BRACE_TOKEN
	recordType.SetPosition(n.getPosition(recordTypeDescriptorNode))
	return recordType, true
}

func (n *nodeBuilder) transformRecordTypeInclusion(typeReference *st.TypeReferenceNode) (ast.BType, bool) {
	if !n.expectNode(typeReference) || !n.expectToken(typeReference.AsteriskToken()) {
		return nil, false
	}
	inclusion := n.transformType(typeReference.TypeName())
	if !n.expectToken(typeReference.SemicolonToken()) {
		return nil, false
	}
	return inclusion, true
}

func (n *nodeBuilder) transformRecordRestDescriptor(restDesc *st.RecordRestDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(restDesc) {
		return nil, false
	}
	restType := n.transformType(restDesc.TypeName())
	if !n.expectToken(restDesc.EllipsisToken()) || !n.expectToken(restDesc.SemicolonToken()) {
		return nil, false
	}
	return restType, true
}

func (n *nodeBuilder) transformRecordField(recordField *st.RecordFieldNode) (ast.BField, bool) {
	if !n.expectNode(recordField) {
		return ast.BField{}, false
	}
	metadata, ok := n.transformMetadata(recordField.Metadata())
	if !ok {
		return ast.BField{}, false
	}
	readonlyKeyword := recordField.ReadonlyKeyword()
	if !n.expectOptionalSignificantToken(readonlyKeyword) {
		return ast.BField{}, false
	}
	fieldType := n.transformType(recordField.TypeName())
	fieldNameToken := recordField.FieldName()
	questionMarkToken := recordField.QuestionMarkToken()
	if !n.expectToken(fieldNameToken) || !n.expectOptionalSignificantToken(questionMarkToken) || !n.expectToken(recordField.SemicolonToken()) {
		return ast.BField{}, false
	}
	fieldName, _ := normalizedIdentifierValue(fieldNameToken.Text())
	var flags model.Flag
	if readonlyKeyword != nil {
		flags |= model.FlagReadonly
	}
	if questionMarkToken != nil {
		flags |= model.FlagOptional
	}
	bField := ast.NewBField(n.getPosition(recordField), model.Name(fieldName), fieldType, nil, flags)
	applyMetadata(metadata, &bField)
	return bField, true
}

func (n *nodeBuilder) transformRecordFieldWithDefaultValue(recordFieldDV *st.RecordFieldWithDefaultValueNode) (ast.BField, bool) {
	if !n.expectNode(recordFieldDV) {
		return ast.BField{}, false
	}
	metadata, ok := n.transformMetadata(recordFieldDV.Metadata())
	if !ok {
		return ast.BField{}, false
	}
	readonlyKeyword := recordFieldDV.ReadonlyKeyword()
	if !n.expectOptionalSignificantToken(readonlyKeyword) {
		return ast.BField{}, false
	}
	fieldType := n.transformType(recordFieldDV.TypeName())
	fieldNameToken := recordFieldDV.FieldName()
	if !n.expectToken(fieldNameToken) || !n.expectToken(recordFieldDV.EqualsToken()) {
		return ast.BField{}, false
	}
	defaultExpr := n.transformExpr(recordFieldDV.Expression())
	if !n.expectToken(recordFieldDV.SemicolonToken()) {
		return ast.BField{}, false
	}
	fieldName, _ := normalizedIdentifierValue(fieldNameToken.Text())
	var flags model.Flag
	if readonlyKeyword != nil {
		flags |= model.FlagReadonly
	}
	bField := ast.NewBField(n.getPosition(recordFieldDV), model.Name(fieldName), fieldType, defaultExpr, flags)
	applyMetadata(metadata, &bField)
	return bField, true
}

func (n *nodeBuilder) transformOptionalTypeDescriptor(optionalTypeDescriptorNode *st.OptionalTypeDescriptorNode) (*ast.BLangUnionTypeNode, bool) {
	if !n.expectNode(optionalTypeDescriptorNode) {
		return nil, false
	}
	typeDescriptor := n.transformType(optionalTypeDescriptorNode.TypeDescriptor())
	questionMarkToken := optionalTypeDescriptorNode.QuestionMarkToken()
	if !n.expectToken(questionMarkToken) {
		return nil, false
	}
	nilType := &ast.BLangValueType{TypeKind: ast.TypeKindNil}
	nilType.SetPosition(n.getPosition(questionMarkToken))
	return ast.NewBLangUnionTypeNode(
		n.getPosition(optionalTypeDescriptorNode),
		ast.TypeData{TypeDescriptor: typeDescriptor},
		ast.TypeData{TypeDescriptor: nilType},
	), true
}

func (n *nodeBuilder) transformMapTypeDescriptor(mapTypeDescriptorNode *st.MapTypeDescriptorNode) (*ast.BLangConstrainedType, bool) {
	if !n.expectNode(mapTypeDescriptorNode) || !n.expectToken(mapTypeDescriptorNode.MapKeywordToken()) {
		return nil, false
	}
	refType := &ast.BLangBuiltInRefTypeNode{
		TypeKind: ast.TypeKindMap,
	}
	refType.SetPosition(n.getPosition(mapTypeDescriptorNode))

	mapTypeParamsNode := mapTypeDescriptorNode.MapTypeParamsNode()
	if mapTypeParamsNode == nil || mapTypeParamsNode.TypeNode() == nil {
		n.internalError("map type requires type parameter", mapTypeDescriptorNode)
		return nil, false
	}
	constraint, ok := n.transformTypeParameter(mapTypeParamsNode)
	if !ok {
		return nil, false
	}
	constrainedType := &ast.BLangConstrainedType{
		BaseType:   ast.TypeData{TypeDescriptor: refType},
		Constraint: ast.TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(refType.GetPosition())
	return constrainedType, true
}

// transformTypeParameter checks the angle brackets of a type parameter and transforms the type they enclose.
func (n *nodeBuilder) transformTypeParameter(typeParameterNode *st.TypeParameterNode) (ast.BType, bool) {
	if !n.expectNode(typeParameterNode) || !n.expectToken(typeParameterNode.LtToken()) {
		return nil, false
	}
	typeNode := typeParameterNode.TypeNode()
	if typeNode == nil {
		n.internalError("createTypeNode: typeNode is nil", typeParameterNode)
		return nil, false
	}
	constraint := n.transformType(typeNode)
	if !n.expectToken(typeParameterNode.GtToken()) {
		return nil, false
	}
	return constraint, true
}

func (n *nodeBuilder) transformUnionTypeDescriptor(unionTypeDescriptorNode *st.UnionTypeDescriptorNode) (*ast.BLangUnionTypeNode, bool) {
	if !n.expectNode(unionTypeDescriptorNode) {
		return nil, false
	}
	lhs := n.transformType(unionTypeDescriptorNode.LeftTypeDesc())
	if !n.expectToken(unionTypeDescriptorNode.PipeToken()) {
		return nil, false
	}
	return ast.NewBLangUnionTypeNode(
		n.getPosition(unionTypeDescriptorNode),
		ast.TypeData{TypeDescriptor: lhs},
		ast.TypeData{TypeDescriptor: n.transformType(unionTypeDescriptorNode.RightTypeDesc())},
	), true
}

func (n *nodeBuilder) transformIntersectionTypeDescriptor(intersectionTypeDescriptorNode *st.IntersectionTypeDescriptorNode) (*ast.BLangIntersectionTypeNode, bool) {
	if !n.expectNode(intersectionTypeDescriptorNode) {
		return nil, false
	}
	lhs := n.transformType(intersectionTypeDescriptorNode.LeftTypeDesc())
	if !n.expectToken(intersectionTypeDescriptorNode.BitwiseAndToken()) {
		return nil, false
	}
	return ast.NewBLangIntersectionTypeNode(
		n.getPosition(intersectionTypeDescriptorNode),
		ast.TypeData{TypeDescriptor: lhs},
		ast.TypeData{TypeDescriptor: n.transformType(intersectionTypeDescriptorNode.RightTypeDesc())},
	), true
}

func (n *nodeBuilder) transformStreamTypeDescriptor(streamTypeDescriptorNode *st.StreamTypeDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(streamTypeDescriptorNode) || !n.expectToken(streamTypeDescriptorNode.StreamKeywordToken()) {
		return nil, false
	}
	position := n.getPosition(streamTypeDescriptorNode)
	paramsNode := streamTypeDescriptorNode.StreamTypeParamsNode()
	if paramsNode == nil {
		refType := &ast.BLangBuiltInRefTypeNode{
			TypeKind: ast.TypeKindStream,
		}
		refType.SetPosition(position)
		return refType, true
	}
	params, ok := paramsNode.(*st.StreamTypeParamsNode)
	if !ok {
		n.cx.InternalError("unexpected stream type params node", position)
		return nil, false
	}
	if !n.expectNode(params) || !n.expectToken(params.LtToken()) {
		return nil, false
	}
	valueDesc := params.LeftTypeDescNode()
	if valueDesc == nil {
		n.cx.InternalError("stream<...> requires a value type parameter", position)
		return nil, false
	}
	valueType := ast.TypeData{TypeDescriptor: n.transformType(valueDesc)}
	if !n.expectOptionalToken(params.CommaToken()) {
		return nil, false
	}
	var completionType ast.TypeData
	if completionDesc := params.RightTypeDescNode(); completionDesc != nil {
		completionType.TypeDescriptor = n.transformType(completionDesc)
	}
	if !n.expectToken(params.GtToken()) {
		return nil, false
	}
	streamType := ast.NewBLangStreamType(valueType, completionType)
	streamType.SetPosition(position)
	return streamType, true
}

func (n *nodeBuilder) transformTableTypeDescriptor(tableTypeDescriptorNode *st.TableTypeDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(tableTypeDescriptorNode) {
		return nil, false
	}
	n.unimplemented("transformTableTypeDescriptor unimplemented", tableTypeDescriptorNode)
	return nil, false
}

func (n *nodeBuilder) transformFunctionTypeDescriptor(functionTypeDescriptorNode *st.FunctionTypeDescriptorNode) (*ast.BLangFunctionType, bool) {
	if !n.expectNode(functionTypeDescriptorNode) {
		return nil, false
	}
	var flags model.Flag
	qualifierList := functionTypeDescriptorNode.QualifierList()
	for token := range qualifierList.Iterator() {
		if !n.expectSignificantToken(token) {
			return nil, false
		}
		switch token.Kind() {
		case st.ISOLATED_KEYWORD:
			flags |= model.FlagIsolated
		case st.TRANSACTIONAL_KEYWORD:
			flags |= model.FlagTransactional
		default:
			n.internalError("unexpected function type qualifier", token)
			return nil, false
		}
	}
	if !n.expectToken(functionTypeDescriptorNode.FunctionKeyword()) {
		return nil, false
	}

	signature := newFunctionTypeSignature()
	if funcSignature := functionTypeDescriptorNode.FunctionSignature(); funcSignature != nil {
		var ok bool
		signature, ok = n.transformFunctionTypeSignature(funcSignature)
		if !ok {
			return nil, false
		}
		if signature.explicitReturnType {
			flags |= model.FlagExplicitReturnTypeDescriptor
		}
	} else {
		flags |= model.FlagAnyFunction
	}
	return ast.NewBLangFunctionType(
		n.getPosition(functionTypeDescriptorNode),
		signature.requiredParams,
		signature.restParam,
		signature.returnType,
		signature.paramListPos,
		flags,
	), true
}

func paramPartsOf(param st.Node) (st.NonTerminalNode, paramParts, bool) {
	switch param := param.(type) {
	case *st.RequiredParameterNode:
		return param, paramParts{
			annotations: param.Annotations(),
			typeName:    param.TypeName(),
			paramName:   param.ParamName(),
			flags:       model.FlagRequiredParam,
			kind:        st.REQUIRED_PARAM,
		}, true
	case *st.DefaultableParameterNode:
		return param, paramParts{
			annotations: param.Annotations(),
			typeName:    param.TypeName(),
			paramName:   param.ParamName(),
			equals:      param.EqualsToken(),
			defaultExpr: param.Expression(),
			flags:       model.FlagDefaultableParam,
			kind:        st.DEFAULTABLE_PARAM,
		}, true
	case *st.IncludedRecordParameterNode:
		return param, paramParts{
			annotations: param.Annotations(),
			asterisk:    param.AsteriskToken(),
			typeName:    param.TypeName(),
			paramName:   param.ParamName(),
			flags:       model.FlagRequiredParam | model.FlagIncluded,
			kind:        st.INCLUDED_RECORD_PARAM,
		}, true
	case *st.RestParameterNode:
		return param, paramParts{
			annotations: param.Annotations(),
			typeName:    param.TypeName(),
			ellipsis:    param.EllipsisToken(),
			paramName:   param.ParamName(),
			flags:       model.FlagRestParam,
			kind:        st.REST_PARAM,
		}, true
	default:
		return nil, paramParts{}, false
	}
}

func (n *nodeBuilder) createFunctionTypeParam(param st.Node) (*ast.BLangFunctionTypeParam, bool) {
	paramNode, parts, ok := paramPartsOf(param)
	if !ok {
		n.internalError("createFunctionTypeParam: unsupported parameter type", param)
		return nil, false
	}
	if !n.expectNode(paramNode) {
		return nil, false
	}
	if parts.annotations.Size() > 0 {
		n.unimplemented("function type param annotations not yet supported", parts.annotations.Get(0))
		return nil, false
	}
	if !n.expectParamTokenBefore(parts) {
		return nil, false
	}
	ftParam := ast.BLangFunctionTypeParam{}
	ftParam.SetPosition(n.getPosition(param))
	ftParam.TypeDesc = n.transformType(parts.typeName)
	if !n.expectParamTokenAfter(parts) {
		return nil, false
	}
	if parts.paramName != nil {
		ftParam.Name = n.transformIdentifier(parts.paramName)
	}
	defaultExpr, ok := n.transformParamDefault(paramNode, parts)
	if !ok {
		return nil, false
	}
	if defaultExpr != nil {
		ftParam.InitExpr = defaultExpr
	}
	if parts.flags&model.FlagIncluded != 0 {
		ftParam.SetIncludedRecordParam()
	}
	return &ftParam, true
}

func (n *nodeBuilder) transformTupleTypeDescriptor(tupleTypeDescriptorNode *st.TupleTypeDescriptorNode) (*ast.BLangTupleTypeNode, bool) {
	if !n.expectNode(tupleTypeDescriptorNode) || !n.expectToken(tupleTypeDescriptorNode.OpenBracketToken()) {
		return nil, false
	}
	tupleTypeNode := &ast.BLangTupleTypeNode{
		Members: make([]ast.BLangMemberTypeDesc, 0),
	}
	types := tupleTypeDescriptorNode.MemberTypeDesc()
	for node, separator := range types.SeparatedElements() {
		if !n.addTupleMember(tupleTypeNode, node) || !n.expectOptionalToken(separator) {
			return nil, false
		}
	}
	if !n.expectToken(tupleTypeDescriptorNode.CloseBracketToken()) {
		return nil, false
	}
	tupleTypeNode.SetPosition(n.getPosition(tupleTypeDescriptorNode))
	return tupleTypeNode, true
}

func (n *nodeBuilder) addTupleMember(tupleTypeNode *ast.BLangTupleTypeNode, node st.Node) bool {
	switch node := node.(type) {
	case *st.RestDescriptorNode:
		if !n.expectNode(node) {
			return false
		}
		tupleTypeNode.Rest = n.transformType(node.TypeDescriptor())
		return n.expectToken(node.EllipsisToken())
	case *st.MemberTypeDescriptorNode:
		if !n.expectNode(node) {
			return false
		}
		if _, ok := n.transformAnnotations(node.Annotations()); !ok {
			return false
		}
		member := ast.BLangMemberTypeDesc{
			TypeDesc: n.transformType(node.TypeDescriptor()),
		}
		member.SetPosition(n.getPosition(node))
		tupleTypeNode.Members = append(tupleTypeNode.Members, member)
		return true
	default:
		n.internalError(fmt.Sprintf("unexpected tuple member node %T", node), node)
		return false
	}
}

func (n *nodeBuilder) transformParenthesisedTypeDescriptor(parenthesisedTypeDescriptorNode *st.ParenthesisedTypeDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(parenthesisedTypeDescriptorNode) || !n.expectToken(parenthesisedTypeDescriptorNode.OpenParenToken()) {
		return nil, false
	}
	typeNode := n.transformType(parenthesisedTypeDescriptorNode.Typedesc())
	if !n.expectToken(parenthesisedTypeDescriptorNode.CloseParenToken()) {
		return nil, false
	}
	return typeNode, true
}

func (n *nodeBuilder) transformSingletonTypeDescriptor(singletonTypeDescriptorNode *st.SingletonTypeDescriptorNode) (*ast.BLangFiniteTypeNode, bool) {
	if !n.expectNode(singletonTypeDescriptorNode) {
		return nil, false
	}
	bLFiniteTypeNode := &ast.BLangFiniteTypeNode{}
	bLFiniteTypeNode.SetPosition(n.getPosition(singletonTypeDescriptorNode))
	bLFiniteTypeNode.ValueSpace = append(bLFiniteTypeNode.ValueSpace, n.transformExpr(singletonTypeDescriptorNode.SimpleContExprNode()))
	return bLFiniteTypeNode, true
}

func (n *nodeBuilder) transformArrayTypeDescriptor(arrayTypeDescriptorNode *st.ArrayTypeDescriptorNode) (*ast.BLangArrayType, bool) {
	if !n.expectNode(arrayTypeDescriptorNode) {
		return nil, false
	}
	elemType := n.transformType(arrayTypeDescriptorNode.MemberTypeDesc())
	var sizes []ast.BLangExpression
	dimensionNodes := arrayTypeDescriptorNode.Dimensions()
	for dimensionNode := range dimensionNodes.Iterator() {
		if !n.expectNode(dimensionNode) || !n.expectToken(dimensionNode.OpenBracket()) {
			return nil, false
		}
		if arrayLength := dimensionNode.ArrayLength(); arrayLength == nil {
			sizes = append(sizes, nil)
		} else {
			sizes = append(sizes, n.transformExpr(arrayLength))
		}
		if !n.expectToken(dimensionNode.CloseBracket()) {
			return nil, false
		}
	}

	arrayTypeNode := &ast.BLangArrayType{}
	arrayTypeNode.SetPosition(n.getPosition(arrayTypeDescriptorNode))
	arrayTypeNode.Elemtype = ast.TypeData{TypeDescriptor: elemType}
	arrayTypeNode.Dimensions = len(sizes)
	arrayTypeNode.Sizes = sizes
	return arrayTypeNode, true
}

func (n *nodeBuilder) transformTypeReferenceTypeDesc(typeReferenceTypeDescNode *st.TypeReferenceTypeDescNode) (ast.BType, bool) {
	if !n.expectNode(typeReferenceTypeDescNode) {
		return nil, false
	}
	n.unimplemented("transformTypeReferenceTypeDesc unimplemented", typeReferenceTypeDescNode)
	return nil, false
}

func (n *nodeBuilder) transformDistinctTypeDescriptor(distinctTypeDescriptorNode *st.DistinctTypeDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(distinctTypeDescriptorNode) {
		return nil, false
	}
	n.cx.Unimplemented("anonymous distinct types not supported", n.getPosition(distinctTypeDescriptorNode))
	return nil, false
}

func (n *nodeBuilder) transformParameterizedTypeDescriptor(parameterizedTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) (ast.BType, bool) {
	if !n.expectNode(parameterizedTypeDescriptorNode) || !n.expectToken(parameterizedTypeDescriptorNode.KeywordToken()) {
		return nil, false
	}
	switch parameterizedTypeDescriptorNode.Kind() {
	case st.ERROR_TYPE_DESC:
		return n.transformErrorTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.TYPEDESC_TYPE_DESC:
		return n.transformTypedescTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.FUTURE_TYPE_DESC:
		return n.transformFutureTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.XML_TYPE_DESC:
		return n.transformXMLTypeDescriptor(parameterizedTypeDescriptorNode)
	default:
		n.internalError("transformParameterizedTypeDescriptor supported only for error, typedesc, future and xml type descriptors", parameterizedTypeDescriptorNode)
		return nil, false
	}
}

func (n *nodeBuilder) transformFutureTypeDescriptor(node *st.ParameterizedTypeDescriptorNode) (ast.BType, bool) {
	pos := n.getPosition(node)
	base := &ast.BLangBuiltInRefTypeNode{TypeKind: ast.TypeKindFuture}
	base.SetPosition(pos)
	typeParamNode := node.TypeParamNode()
	if typeParamNode == nil {
		return base, true
	}
	constraint, ok := n.transformTypeParameter(typeParamNode)
	if !ok {
		return nil, false
	}
	constrainedType := &ast.BLangConstrainedType{
		BaseType:   ast.TypeData{TypeDescriptor: base},
		Constraint: ast.TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(pos)
	return constrainedType, true
}

func (n *nodeBuilder) transformTypedescTypeDescriptor(node *st.ParameterizedTypeDescriptorNode) (ast.BType, bool) {
	typeParamNode := node.TypeParamNode()
	if typeParamNode == nil {
		valueType := &ast.BLangValueType{}
		valueType.SetPosition(n.getPosition(node))
		valueType.TypeKind = ast.TypeKindTypeDesc
		return valueType, true
	}
	constraint, ok := n.transformTypeParameter(typeParamNode)
	if !ok {
		return nil, false
	}
	constrainedType := &ast.BLangConstrainedType{}
	constrainedType.SetPosition(n.getPosition(node))
	base := &ast.BLangValueType{}
	base.SetPosition(n.getPosition(node))
	base.TypeKind = ast.TypeKindTypeDesc
	constrainedType.BaseType = ast.TypeData{TypeDescriptor: base}
	constrainedType.Constraint = ast.TypeData{TypeDescriptor: constraint}
	return constrainedType, true
}

func (n *nodeBuilder) transformXMLTypeDescriptor(parameterizedTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) (ast.BType, bool) {
	pos := n.getPosition(parameterizedTypeDescriptorNode)
	typeParamNode := parameterizedTypeDescriptorNode.TypeParamNode()
	if typeParamNode == nil {
		valueType := &ast.BLangValueType{}
		valueType.SetPosition(pos)
		valueType.TypeKind = ast.TypeKindXML
		return valueType, true
	}
	constraint, ok := n.transformTypeParameter(typeParamNode)
	if !ok {
		return nil, false
	}
	refType := &ast.BLangBuiltInRefTypeNode{
		TypeKind: ast.TypeKindXML,
	}
	refType.SetPosition(pos)
	constrainedType := &ast.BLangConstrainedType{
		BaseType:   ast.TypeData{TypeDescriptor: refType},
		Constraint: ast.TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(pos)
	return constrainedType, true
}

func (n *nodeBuilder) transformErrorTypeDescriptor(errorTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) (ast.BType, bool) {
	var detailType ast.TypeData
	if typeParamNode := errorTypeDescriptorNode.TypeParamNode(); typeParamNode != nil {
		constraint, ok := n.transformTypeParameter(typeParamNode)
		if !ok {
			return nil, false
		}
		detailType.TypeDescriptor = constraint
	}
	return ast.NewBLangErrorTypeNode(
		n.getPosition(errorTypeDescriptorNode),
		detailType,
		errorTypeDescriptorNode.Parent().Kind() == st.DISTINCT_TYPE_DESC,
	), true
}

func (n *nodeBuilder) transformAnnotation(annotationNode *st.AnnotationNode) (*ast.BLangAnnotationAttachment, bool) {
	if !n.expectNode(annotationNode) || !n.expectToken(annotationNode.AtToken()) {
		return nil, false
	}
	annotation := &ast.BLangAnnotationAttachment{}
	annotation.SetPosition(n.getPosition(annotationNode))
	pkgAlias, name, ok := n.createBLangNameReference(annotationNode.AnnotReference())
	if !ok {
		return nil, false
	}
	annotation.PkgAlias = pkgAlias
	annotation.AnnotationName = name
	if value := annotationNode.AnnotValue(); value != nil {
		annotation.Expr = n.transformExpr(value)
		annotation.HasValue = true
	} else {
		annotation.Expr = n.createTrueLiteral(annotation.GetPosition())
	}
	return annotation, true
}

func (n *nodeBuilder) addReferencesAndReturnDocumentationText(references *[]ast.BLangMarkdownReferenceDocumentation, docElements st.NodeList[st.Node]) (string, bool) {
	var docText strings.Builder
	for element := range docElements.Iterator() {
		switch element := element.(type) {
		case *st.BallerinaNameReferenceNode:
			bLangRefDoc, text, ok := n.transformBallerinaNameReference(element)
			if !ok {
				return "", false
			}
			docText.WriteString(text)
			*references = append(*references, *bLangRefDoc)
		case *st.InlineCodeReferenceNode:
			text, ok := n.inlineCodeReferenceText(element)
			if !ok {
				return "", false
			}
			docText.WriteString(text)
		case st.Token:
			if !n.expectSignificantToken(element) {
				return "", false
			}
			if element.Kind() == st.DOCUMENTATION_DESCRIPTION {
				docText.WriteString(element.Text())
			}
		case st.NonTerminalNode:
			if !n.expectNode(element) {
				return "", false
			}
		}
	}
	return n.trimLeftAtMostOne(docText.String()), true
}

func (n *nodeBuilder) transformBallerinaNameReference(balNameRefNode *st.BallerinaNameReferenceNode) (*ast.BLangMarkdownReferenceDocumentation, string, bool) {
	referenceType := balNameRefNode.ReferenceType()
	startBacktick := balNameRefNode.StartBacktick()
	backtickContent := balNameRefNode.NameReference()
	endBacktick := balNameRefNode.EndBacktick()
	if !n.expectNode(balNameRefNode) || !n.expectOptionalToken(referenceType) || !n.expectToken(startBacktick) {
		return nil, "", false
	}

	bLangRefDoc := &ast.BLangMarkdownReferenceDocumentation{}
	bLangRefDoc.SetPosition(n.getPosition(balNameRefNode))
	var docText strings.Builder

	if backtickContent == nil {
		n.internalError("documentation reference has no name reference", balNameRefNode)
		return nil, "", false
	}
	bLangRefDoc.Type = ast.DocumentationReferenceType("BACKTICK_CONTENT")
	if referenceType != nil {
		refTypeText := referenceType.Text()
		bLangRefDoc.Type = n.stringToRefType(refTypeText)
		docText.WriteString(refTypeText)
	}
	if !n.transformDocumentationBacktickContent(backtickContent, bLangRefDoc) {
		return nil, "", false
	}
	contentString := st.ToSourceCode(backtickContent.InternalNode())
	bLangRefDoc.ReferenceName = contentString
	if !n.expectToken(endBacktick) {
		return nil, "", false
	}

	docText.WriteString(startBacktick.Text())
	docText.WriteString(contentString)
	docText.WriteString(endBacktick.Text())
	return bLangRefDoc, docText.String(), true
}

func (n *nodeBuilder) inlineCodeReferenceText(inlineCodeRefNode *st.InlineCodeReferenceNode) (string, bool) {
	startBacktick := inlineCodeRefNode.StartBacktick()
	codeRef := inlineCodeRefNode.CodeReference()
	endBacktick := inlineCodeRefNode.EndBacktick()
	if !n.expectNode(inlineCodeRefNode) || !n.expectToken(startBacktick) || !n.expectToken(codeRef) || !n.expectToken(endBacktick) {
		return "", false
	}
	var docText strings.Builder
	docText.WriteString(startBacktick.Text())
	docText.WriteString(codeRef.Text())
	docText.WriteString(endBacktick.Text())
	return docText.String(), true
}

// documentationNameReference checks a name reference inside a documentation backtick and returns its qualifier
// and identifier text.
func (n *nodeBuilder) documentationNameReference(nameRef st.Node) (string, string, bool) {
	switch nameRef := nameRef.(type) {
	case *st.QualifiedNameReferenceNode:
		modulePrefix := nameRef.ModulePrefix()
		identifier := asToken(nameRef.Identifier())
		if !n.expectNode(nameRef) || !n.expectToken(modulePrefix) || !n.expectColon(nameRef.Colon()) || !n.expectToken(identifier) {
			return "", "", false
		}
		return modulePrefix.Text(), identifier.Text(), true
	case *st.SimpleNameReferenceNode:
		name := nameRef.Name()
		if !n.expectNode(nameRef) || !n.expectToken(name) {
			return "", "", false
		}
		return "", name.Text(), true
	case st.Token:
		return "", "", n.expectToken(nameRef)
	case st.NonTerminalNode:
		return "", "", n.expectNode(nameRef)
	default:
		return "", "", true
	}
}

// expectDocumentationArguments checks the arguments of a call written inside a documentation reference; the
// arguments are not part of the documentation AST.
func (n *nodeBuilder) expectDocumentationArguments(arguments st.NodeList[st.FunctionArgumentNode]) bool {
	_, ok := n.transformArguments(arguments)
	return ok
}

func (n *nodeBuilder) transformDocumentationBacktickContent(backtickContent st.Node, bLangRefDoc *ast.BLangMarkdownReferenceDocumentation) bool {
	switch backtickContent.Kind() {
	case st.CODE_CONTENT:
		// reaching here means ballerina name reference is syntactically invalid.
		// therefore, set hasParserWarnings to true. so that,
		// doc analyzer will avoid further checks on this.
		if tok, ok := backtickContent.(st.Token); ok && !n.expectSignificantToken(tok) {
			return false
		}
		bLangRefDoc.HasParserWarnings = true
	case st.QUALIFIED_NAME_REFERENCE, st.SIMPLE_NAME_REFERENCE:
		qualifier, identifier, ok := n.documentationNameReference(backtickContent)
		if !ok {
			return false
		}
		bLangRefDoc.Qualifier = qualifier
		bLangRefDoc.Identifier = identifier
	case st.FUNCTION_CALL:
		funcCallExpr := backtickContent.(*st.FunctionCallExpressionNode)
		if !n.expectNode(funcCallExpr) {
			return false
		}
		qualifier, identifier, ok := n.documentationNameReference(funcCallExpr.FunctionName())
		if !ok || !n.expectToken(funcCallExpr.OpenParenToken()) || !n.expectDocumentationArguments(funcCallExpr.Arguments()) ||
			!n.expectToken(funcCallExpr.CloseParenToken()) {
			return false
		}
		bLangRefDoc.Qualifier = qualifier
		bLangRefDoc.Identifier = identifier
	case st.METHOD_CALL:
		methodCallExprNode := backtickContent.(*st.MethodCallExpressionNode)
		if !n.expectNode(methodCallExprNode) {
			return false
		}
		typeQualifier, typeName, ok := n.documentationNameReference(methodCallExprNode.Expression())
		if !ok || !n.expectToken(methodCallExprNode.DotToken()) {
			return false
		}
		_, methodName, ok := n.documentationNameReference(methodCallExprNode.MethodName())
		if !ok || !n.expectToken(methodCallExprNode.OpenParenToken()) || !n.expectDocumentationArguments(methodCallExprNode.Arguments()) ||
			!n.expectToken(methodCallExprNode.CloseParenToken()) {
			return false
		}
		bLangRefDoc.Identifier = methodName
		bLangRefDoc.TypeName = typeName
		bLangRefDoc.Qualifier = typeQualifier
	default:
		if tok, ok := backtickContent.(st.Token); ok {
			return n.expectSignificantToken(tok)
		}
		if nonTerminal, ok := backtickContent.(st.NonTerminalNode); ok {
			return n.expectNode(nonTerminal)
		}
	}

	// Process identifier and qualifier - unescape and remove single quote prefix if present
	if bLangRefDoc.Identifier != "" {
		bLangRefDoc.Identifier = unescapeUnicodeCodepoints(bLangRefDoc.Identifier)
		if n.stringStartsWithSingleQuote(bLangRefDoc.Identifier) {
			bLangRefDoc.Identifier = bLangRefDoc.Identifier[1:]
		}
	}
	if bLangRefDoc.Qualifier != "" {
		bLangRefDoc.Qualifier = unescapeUnicodeCodepoints(bLangRefDoc.Qualifier)
		if n.stringStartsWithSingleQuote(bLangRefDoc.Qualifier) {
			bLangRefDoc.Qualifier = bLangRefDoc.Qualifier[1:]
		}
	}
	return true
}

func (n *nodeBuilder) transformCodeBlock(documentationLines *[]ast.BLangMarkdownDocumentationLine, codeBlockNode *st.MarkdownCodeBlockNode) bool {
	startLineHashToken := codeBlockNode.StartLineHashToken()
	startBacktick := codeBlockNode.StartBacktick()
	langAttribute := codeBlockNode.LangAttribute()
	if !n.expectNode(codeBlockNode) || !n.expectToken(startLineHashToken) || !n.expectSignificantToken(startBacktick) ||
		!n.expectOptionalSignificantToken(langAttribute) {
		return false
	}

	var docText strings.Builder
	docText.WriteString(startBacktick.Text())
	if langAttribute != nil {
		docText.WriteString(langAttribute.Text())
	}

	codeLines := codeBlockNode.CodeLines()
	for codeLine := range codeLines.Iterator() {
		codeDescription := codeLine.CodeDescription()
		if !n.expectNode(codeLine) || !n.expectToken(codeLine.HashToken()) || !n.expectToken(codeDescription) {
			return false
		}
		docText.WriteString(codeDescription.Text())
	}

	endBacktick := codeBlockNode.EndBacktick()
	if !n.expectToken(codeBlockNode.EndLineHashToken()) || !n.expectToken(endBacktick) {
		return false
	}
	docText.WriteString(endBacktick.Text())

	bLangDocLine := ast.BLangMarkdownDocumentationLine{}
	bLangDocLine.Text = docText.String()
	bLangDocLine.SetPosition(n.getPosition(startLineHashToken))
	*documentationLines = append(*documentationLines, bLangDocLine)
	return true
}

func (n *nodeBuilder) stringToRefType(refTypeName string) ast.DocumentationReferenceType {
	switch refTypeName {
	case "type":
		return ast.DocumentationReferenceType("TYPE")
	case "service":
		return ast.DocumentationReferenceType("SERVICE")
	case "variable":
		return ast.DocumentationReferenceType("VARIABLE")
	case "var":
		return ast.DocumentationReferenceType("VAR")
	case "annotation":
		return ast.DocumentationReferenceType("ANNOTATION")
	case "module":
		return ast.DocumentationReferenceType("MODULE")
	case "function":
		return ast.DocumentationReferenceType("FUNCTION")
	case "parameter":
		return ast.DocumentationReferenceType("PARAMETER")
	case "const":
		return ast.DocumentationReferenceType("CONST")
	default:
		return ast.DocumentationReferenceType("BACKTICK_CONTENT")
	}
}

func (n *nodeBuilder) stringStartsWithSingleQuote(s string) bool {
	return len(s) > 0 && s[0] == '\''
}

func (n *nodeBuilder) trimLeftAtMostOne(text string) string {
	countToStrip := 0
	if len(text) > 0 && (text[0] == ' ' || text[0] == '\t' || text[0] == '\n' || text[0] == '\r') {
		countToStrip = 1
	}
	if countToStrip > 0 && len(text) > countToStrip {
		return text[countToStrip:]
	}
	return text
}

func (n *nodeBuilder) stringToTypeKind(typeText string, node st.Node) (ast.TypeKind, bool) {
	switch typeText {
	case "int":
		return ast.TypeKindInt, true
	case "byte":
		return ast.TypeKindByte, true
	case "float":
		return ast.TypeKindFloat, true
	case "decimal":
		return ast.TypeKindDecimal, true
	case "boolean":
		return ast.TypeKindBoolean, true
	case "string":
		return ast.TypeKindString, true
	case "json":
		return ast.TypeKindJSON, true
	case "xml":
		return ast.TypeKindXML, true
	case "stream":
		return ast.TypeKindStream, true
	case "table":
		return ast.TypeKindTable, true
	case "any":
		return ast.TypeKindAny, true
	case "anydata":
		return ast.TypeKindAnyData, true
	case "map":
		return ast.TypeKindMap, true
	case "future":
		return ast.TypeKindFuture, true
	case "typedesc":
		return ast.TypeKindTypeDesc, true
	case "error":
		return ast.TypeKindError, true
	case "()", "null":
		return ast.TypeKindNil, true
	case "never":
		return ast.TypeKindNever, true
	case "channel":
		return ast.TypeKindChannel, true
	case "service":
		return ast.TypeKindService, true
	case "handle":
		return ast.TypeKindHandle, true
	case "readonly":
		return ast.TypeKindReadOnly, true
	case "function":
		return ast.TypeKindFunction, true
	default:
		n.internalError("stringToTypeKind: invalid type name: "+typeText, node)
		return ast.TypeKindNever, false
	}
}

func createUserDefinedType(pos diagnostics.Location, pkgAlias ast.BLangIdentifier, typeName ast.BLangIdentifier) *ast.BLangUserDefinedType {
	userDefinedType := ast.BLangUserDefinedType{}
	userDefinedType.SetPosition(pos)
	userDefinedType.PkgAlias = pkgAlias
	userDefinedType.TypeName = typeName
	return &userDefinedType
}

func (n *nodeBuilder) badTopLevel(node st.Node) *ast.BLangBadTopLevelNode {
	return ast.NewBLangBadTopLevelNode(n.getPosition(node), n.badTopLevelKind(node), n.recoveredTokens(node))
}

func (n *nodeBuilder) badTopLevelKind(node st.Node) ast.BadTopLevelNodeKind {
	switch node.Kind() {
	case st.IMPORT_DECLARATION:
		return ast.BadTopLevelNodeImport
	case st.FUNCTION_DEFINITION, st.OBJECT_METHOD_DEFINITION, st.RESOURCE_ACCESSOR_DEFINITION, st.METHOD_DECLARATION,
		st.RESOURCE_ACCESSOR_DECLARATION:
		return ast.BadTopLevelNodeFunction
	case st.TYPE_DEFINITION:
		return ast.BadTopLevelNodeTypeDefinition
	case st.ENUM_DECLARATION:
		return ast.BadTopLevelNodeEnum
	case st.CONST_DECLARATION:
		return ast.BadTopLevelNodeConstant
	case st.MODULE_VAR_DECL:
		return ast.BadTopLevelNodeVariable
	case st.LISTENER_DECLARATION:
		return ast.BadTopLevelNodeListener
	case st.CLASS_DEFINITION:
		return ast.BadTopLevelNodeClass
	case st.SERVICE_DECLARATION:
		return ast.BadTopLevelNodeService
	case st.ANNOTATION_DECLARATION:
		return ast.BadTopLevelNodeAnnotation
	case st.MODULE_XML_NAMESPACE_DECLARATION:
		return ast.BadTopLevelNodeXMLNamespace
	case st.OBJECT_FIELD:
		return ast.BadTopLevelNodeField
	case st.TYPE_REFERENCE:
		return ast.BadTopLevelNodeTypeInclusion
	case st.EOF_TOKEN:
		return ast.BadTopLevelNodeTrailingTokens
	default:
		n.internalError(fmt.Sprintf("badTopLevelKind: unclassified node kind %s", node.Kind()), node)
		return ast.BadTopLevelNodeMissing
	}
}

func (n *nodeBuilder) badStmt(node st.Node) *ast.BLangBadStmt {
	return ast.NewBLangBadStmt(n.getPosition(node), n.badStmtKind(node), n.recoveredTokens(node))
}

func (n *nodeBuilder) badStmtKind(node st.Node) ast.BadStmtKind {
	switch node.(type) {
	case *st.AssignmentStatementNode:
		return ast.BadStmtAssignment
	case *st.CompoundAssignmentStatementNode:
		return ast.BadStmtCompoundAssignment
	case *st.VariableDeclarationNode:
		return ast.BadStmtVariableDeclaration
	case *st.BlockStatementNode:
		return ast.BadStmtBlock
	case *st.BreakStatementNode:
		return ast.BadStmtBreak
	case *st.FailStatementNode:
		return ast.BadStmtFail
	case *st.ExpressionStatementNode:
		return ast.BadStmtExpression
	case *st.ContinueStatementNode:
		return ast.BadStmtContinue
	case *st.IfElseStatementNode:
		return ast.BadStmtIfElse
	case *st.WhileStatementNode:
		return ast.BadStmtWhile
	case *st.PanicStatementNode:
		return ast.BadStmtPanic
	case *st.ReturnStatementNode:
		return ast.BadStmtReturn
	case *st.LocalTypeDefinitionStatementNode:
		return ast.BadStmtLocalTypeDefinition
	case *st.LockStatementNode:
		return ast.BadStmtLock
	case *st.ForkStatementNode:
		return ast.BadStmtFork
	case *st.ForEachStatementNode:
		return ast.BadStmtForEach
	case *st.XMLNamespaceDeclarationNode:
		return ast.BadStmtXMLNamespace
	case *st.TransactionStatementNode:
		return ast.BadStmtTransaction
	case *st.RollbackStatementNode:
		return ast.BadStmtRollback
	case *st.RetryStatementNode:
		return ast.BadStmtRetry
	case *st.MatchStatementNode:
		return ast.BadStmtMatch
	case *st.DoStatementNode:
		return ast.BadStmtDo
	default:
		n.internalError(fmt.Sprintf("badStmtKind: unclassified node type %T", node), node)
		return ast.BadStmtMissing
	}
}

func (n *nodeBuilder) badExprOrAction(node st.Node) *ast.BLangBadExprOrAction {
	return ast.NewBLangBadExprOrAction(n.getPosition(node), n.badExprKind(node), n.recoveredTokens(node))
}

func (n *nodeBuilder) badExprKind(node st.Node) ast.BadExprKind {
	switch node := node.(type) {
	case *st.RemoteMethodCallActionNode:
		return ast.BadExprRemoteMethodCallAction
	case *st.ClientResourceAccessActionNode:
		return ast.BadExprClientResourceAccessAction
	case *st.StartActionNode:
		return ast.BadExprStartAction
	case *st.WaitActionNode:
		return ast.BadExprWaitAction
	case *st.QueryActionNode:
		return ast.BadExprQueryAction
	case *st.AsyncSendActionNode:
		return ast.BadExprAsyncSendAction
	case *st.SyncSendActionNode:
		return ast.BadExprSyncSendAction
	case *st.ReceiveActionNode:
		return ast.BadExprReceiveAction
	case *st.CommitActionNode:
		return ast.BadExprCommitAction
	case *st.FlushActionNode:
		return ast.BadExprFlushAction
	case *st.BracedExpressionNode:
		if node.Kind() == st.BRACED_ACTION {
			return ast.BadExprBracedAction
		}
	}
	kind := node.Kind()
	switch {
	case isSimpleLiteral(kind):
		return ast.BadExprSimpleLiteral
	case kind == st.SIMPLE_NAME_REFERENCE || kind == st.QUALIFIED_NAME_REFERENCE || kind == st.IDENTIFIER_TOKEN:
		return ast.BadExprVarRef
	case isType(kind):
		return ast.BadExprTypedesc
	}
	switch node.(type) {
	case *st.BracedExpressionNode:
		return ast.BadExprBraced
	case *st.BinaryExpressionNode:
		return ast.BadExprBinary
	case *st.CheckExpressionNode:
		return ast.BadExprCheck
	case *st.FieldAccessExpressionNode:
		return ast.BadExprFieldAccess
	case *st.FunctionCallExpressionNode:
		return ast.BadExprFunctionCall
	case *st.MethodCallExpressionNode:
		return ast.BadExprMethodCall
	case *st.MappingConstructorExpressionNode:
		return ast.BadExprMappingConstructor
	case *st.IndexedExpressionNode:
		return ast.BadExprIndexed
	case *st.TypeofExpressionNode:
		return ast.BadExprTypeof
	case *st.UnaryExpressionNode:
		return ast.BadExprUnary
	case *st.InferredTypedescDefaultNode:
		return ast.BadExprInferredTypedescDefault
	case *st.ObjectConstructorExpressionNode:
		return ast.BadExprObjectConstructor
	case *st.TypeTestExpressionNode:
		return ast.BadExprTypeTest
	case *st.TrapExpressionNode:
		return ast.BadExprTrap
	case *st.ListConstructorExpressionNode:
		return ast.BadExprListConstructor
	case *st.TypeCastExpressionNode:
		return ast.BadExprTypeCast
	case *st.TableConstructorExpressionNode:
		return ast.BadExprTableConstructor
	case *st.LetExpressionNode:
		return ast.BadExprLet
	case *st.TemplateExpressionNode:
		return ast.BadExprTemplate
	case *st.ExplicitAnonymousFunctionExpressionNode:
		return ast.BadExprExplicitAnonymousFunction
	case *st.ImplicitAnonymousFunctionExpressionNode:
		return ast.BadExprImplicitAnonymousFunction
	case *st.ExplicitNewExpressionNode:
		return ast.BadExprExplicitNew
	case *st.ImplicitNewExpressionNode:
		return ast.BadExprImplicitNew
	case *st.QueryExpressionNode:
		return ast.BadExprQuery
	case *st.AnnotAccessExpressionNode:
		return ast.BadExprAnnotAccess
	case *st.OptionalFieldAccessExpressionNode:
		return ast.BadExprOptionalFieldAccess
	case *st.ConditionalExpressionNode:
		return ast.BadExprConditional
	case *st.TransactionalExpressionNode:
		return ast.BadExprTransactional
	case *st.ByteArrayLiteralNode:
		return ast.BadExprByteArrayLiteral
	case *st.XMLFilterExpressionNode:
		return ast.BadExprXMLFilter
	case *st.XMLStepExpressionNode:
		return ast.BadExprXMLStep
	case *st.RequiredExpressionNode:
		return ast.BadExprRequired
	case *st.ErrorConstructorExpressionNode:
		return ast.BadExprErrorConstructor
	case *st.NaturalExpressionNode:
		return ast.BadExprNatural
	case *st.PositionalArgumentNode:
		return ast.BadExprPositionalArgument
	case *st.NamedArgumentNode:
		return ast.BadExprNamedArgument
	case *st.RestArgumentNode:
		return ast.BadExprRestArgument
	case *st.WildcardBindingPatternNode:
		return ast.BadExprWildcardBindingPattern
	case *st.BasicLiteralNode:
		return ast.BadExprBasicLiteral
	default:
		n.internalError(fmt.Sprintf("badExprKind: unclassified node type %T", node), node)
		return ast.BadExprMissing
	}
}

func (n *nodeBuilder) badTypeNode(node st.Node) *ast.BLangBadTypeNode {
	return ast.NewBLangBadTypeNode(n.getPosition(node), n.badTypeKind(node), n.recoveredTokens(node))
}

func (n *nodeBuilder) badTypeKind(node st.Node) ast.BadTypeKind {
	switch node := node.(type) {
	case *st.BuiltinSimpleNameReferenceNode:
		return ast.BadTypeBuiltin
	case *st.NilTypeDescriptorNode:
		return ast.BadTypeNil
	case *st.QualifiedNameReferenceNode:
		return ast.BadTypeQualifiedReference
	case *st.SimpleNameReferenceNode:
		return ast.BadTypeSimpleReference
	case *st.ObjectTypeDescriptorNode:
		return ast.BadTypeObject
	case *st.RecordTypeDescriptorNode:
		return ast.BadTypeRecord
	case *st.OptionalTypeDescriptorNode:
		return ast.BadTypeOptional
	case *st.MapTypeDescriptorNode:
		return ast.BadTypeMap
	case *st.UnionTypeDescriptorNode:
		return ast.BadTypeUnion
	case *st.StreamTypeDescriptorNode:
		return ast.BadTypeStream
	case *st.TableTypeDescriptorNode:
		return ast.BadTypeTable
	case *st.FunctionTypeDescriptorNode:
		return ast.BadTypeFunction
	case *st.TupleTypeDescriptorNode:
		return ast.BadTypeTuple
	case *st.ParenthesisedTypeDescriptorNode:
		return ast.BadTypeParenthesised
	case *st.IntersectionTypeDescriptorNode:
		return ast.BadTypeIntersection
	case *st.SingletonTypeDescriptorNode:
		return ast.BadTypeSingleton
	case *st.ArrayTypeDescriptorNode:
		return ast.BadTypeArray
	case *st.TypeReferenceTypeDescNode:
		return ast.BadTypeTypeReference
	case *st.DistinctTypeDescriptorNode:
		return ast.BadTypeDistinct
	case *st.ParameterizedTypeDescriptorNode:
		return ast.BadTypeParameterized
	case st.Token:
		if node.Kind() == st.IDENTIFIER_TOKEN {
			return ast.BadTypeSimpleReference
		}
		return ast.BadTypeBuiltin
	default:
		n.internalError(fmt.Sprintf("badTypeKind: unclassified node type %T", node), node)
		return ast.BadTypeMissing
	}
}

// recoveredTokens collects, in source order, the keyword and punctuation tokens present in the source span of node.
// Token positions never include minutiae, whichever range the builder uses for nodes.
func (n *nodeBuilder) recoveredTokens(node st.Node) []ast.RecoveredToken {
	var tokens []ast.RecoveredToken
	var visit func(st.Node)
	visit = func(node st.Node) {
		switch node := node.(type) {
		case nil:
		case st.Token:
			if node.IsMissing() || !isKeywordOrPunctuation(node.Kind()) {
				return
			}
			textRange := node.TextRange()
			pos := diagnostics.NewLocation(n.de(), getFileName(node), textRange.StartOffset, textRange.EndOffset)
			tokens = append(tokens, ast.NewRecoveredToken(pos, node.Kind().StrValue()))
		case st.NonTerminalNode:
			for child := range node.ChildNodes() {
				visit(child)
			}
		}
	}
	visit(node)
	return tokens
}

func isKeywordOrPunctuation(kind st.SyntaxKind) bool {
	return kind < st.TYPE_DOC_REFERENCE_TOKEN
}

func (n *nodeBuilder) badIdentifier(token st.Token) *ast.BLangBadIdentifier {
	value, isLiteral := normalizedIdentifierValue(token.Text())
	return ast.NewBLangBadIdentifier(n.getPosition(token), value, token.Text(), isLiteral)
}
