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
	"math"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	balCommon "github.com/ballerina-nutcracker/ballerina/common"
)

type nodeBuilderMode uint8

const (
	nodeBuilderModeStrict nodeBuilderMode = iota
	nodeBuilderModeRecover
)

type nodeBuilder struct {
	PackageID            *model.PackageID
	anonTypeNameSuffixes []string // Stack for anonymous type name suffixes
	currentCompUnit      *ast.BLangCompilationUnit
	cx                   *context.CompilerContext
	mode                 nodeBuilderMode
}

func (n *nodeBuilder) de() *diagnostics.DiagnosticEnv {
	return n.cx.DiagnosticEnv()
}

func newNodeBuilder(cx *context.CompilerContext) *nodeBuilder {
	return newNodeBuilderWithMode(cx, nodeBuilderModeStrict)
}

func newRecoveringNodeBuilder(cx *context.CompilerContext) *nodeBuilder {
	return newNodeBuilderWithMode(cx, nodeBuilderModeRecover)
}

func newNodeBuilderWithMode(cx *context.CompilerContext, mode nodeBuilderMode) *nodeBuilder {
	nodeBuilder := &nodeBuilder{
		cx:        cx,
		PackageID: cx.GetDefaultPackage(),
		mode:      mode,
	}
	return nodeBuilder
}

func (n *nodeBuilder) transformSyntaxNode(node st.Node) ast.BLangNode {
	switch t := node.(type) {
	case *st.ModulePart:
		return n.transformModulePart(t)
	case *st.FunctionDefinition:
		return n.transformFunctionDefinition(t)
	case *st.ImportDeclarationNode:
		return n.transformImportDeclaration(t)
	case *st.ListenerDeclarationNode:
		return n.transformListenerDeclaration(t)
	case *st.TypeDefinitionNode:
		return n.transformTypeDefinition(t)
	case *st.ServiceDeclarationNode:
		return n.transformServiceDeclaration(t)
	case *st.AssignmentStatementNode:
		return n.transformAssignmentStatement(t)
	case *st.CompoundAssignmentStatementNode:
		return n.transformCompoundAssignmentStatement(t)
	case *st.VariableDeclarationNode:
		return n.transformVariableDeclaration(t)
	case *st.BlockStatementNode:
		return n.transformBlockStatement(t)
	case *st.BreakStatementNode:
		return n.transformBreakStatement(t)
	case *st.FailStatementNode:
		return n.transformFailStatement(t)
	case *st.ExpressionStatementNode:
		return n.transformExpressionStatement(t)
	case *st.ContinueStatementNode:
		return n.transformContinueStatement(t)
	case *st.ExternalFunctionBodyNode:
		return n.transformExternalFunctionBody(t)
	case *st.IfElseStatementNode:
		return n.transformIfElseStatement(t)
	case *st.ElseBlockNode:
		return n.transformElseBlock(t)
	case *st.WhileStatementNode:
		return n.transformWhileStatement(t)
	case *st.PanicStatementNode:
		return n.transformPanicStatement(t)
	case *st.ReturnStatementNode:
		return n.transformReturnStatement(t)
	case *st.LocalTypeDefinitionStatementNode:
		return n.transformLocalTypeDefinitionStatement(t)
	case *st.LockStatementNode:
		return n.transformLockStatement(t)
	case *st.ForkStatementNode:
		return n.transformForkStatement(t)
	case *st.ForEachStatementNode:
		return n.transformForEachStatement(t)
	case *st.BinaryExpressionNode:
		return n.transformBinaryExpression(t)
	case *st.BracedExpressionNode:
		return n.transformBracedExpression(t)
	case *st.CheckExpressionNode:
		return n.transformCheckExpression(t)
	case *st.FieldAccessExpressionNode:
		return n.transformFieldAccessExpression(t)
	case *st.FunctionCallExpressionNode:
		return n.transformFunctionCallExpression(t)
	case *st.MethodCallExpressionNode:
		return n.transformMethodCallExpression(t)
	case *st.MappingConstructorExpressionNode:
		return n.transformMappingConstructorExpression(t)
	case *st.IndexedExpressionNode:
		return n.transformIndexedExpression(t)
	case *st.TypeofExpressionNode:
		return n.transformTypeofExpression(t)
	case *st.UnaryExpressionNode:
		return n.transformUnaryExpression(t)
	case *st.ComputedNameFieldNode:
		return n.transformComputedNameField(t)
	case *st.ConstantDeclarationNode:
		return n.transformConstantDeclaration(t)
	case *st.DefaultableParameterNode:
		return n.transformDefaultableParameter(t)
	case *st.RequiredParameterNode:
		return n.transformRequiredParameter(t)
	case *st.IncludedRecordParameterNode:
		return n.transformIncludedRecordParameter(t)
	case *st.RestParameterNode:
		return n.transformRestParameter(t)
	case *st.ImportOrgNameNode:
		return n.transformImportOrgName(t)
	case *st.ImportPrefixNode:
		return n.transformImportPrefix(t)
	case *st.SpecificFieldNode:
		return n.transformSpecificField(t)
	case *st.SpreadFieldNode:
		return n.transformSpreadField(t)
	case *st.NamedArgumentNode:
		return n.transformNamedArgument(t)
	case *st.PositionalArgumentNode:
		return n.transformPositionalArgument(t)
	case *st.RestArgumentNode:
		return n.transformRestArgument(t)
	case *st.InferredTypedescDefaultNode:
		return n.transformInferredTypedescDefault(t)
	case *st.ObjectTypeDescriptorNode:
		return n.transformObjectTypeDescriptor(t)
	case *st.ObjectConstructorExpressionNode:
		return n.transformObjectConstructorExpression(t)
	case *st.RecordTypeDescriptorNode:
		return n.transformRecordTypeDescriptor(t)
	case *st.ReturnTypeDescriptorNode:
		return n.transformReturnTypeDescriptor(t)
	case *st.NilTypeDescriptorNode:
		return n.transformNilTypeDescriptor(t)
	case *st.OptionalTypeDescriptorNode:
		return n.transformOptionalTypeDescriptor(t)
	case *st.ObjectFieldNode:
		return n.transformObjectField(t)
	case *st.RecordFieldNode:
		return n.transformRecordField(t)
	case *st.RecordFieldWithDefaultValueNode:
		return n.transformRecordFieldWithDefaultValue(t)
	case *st.RecordRestDescriptorNode:
		return n.transformRecordRestDescriptor(t)
	case *st.TypeReferenceNode:
		return n.transformTypeReference(t)
	case *st.AnnotationNode:
		return n.transformAnnotation(t)
	case *st.MetadataNode:
		return n.transformMetadata(t)
	case *st.ModuleVariableDeclarationNode:
		return n.transformModuleVariableDeclaration(t)
	case *st.TypeTestExpressionNode:
		return n.transformTypeTestExpression(t)
	case *st.RemoteMethodCallActionNode:
		return n.transformRemoteMethodCallAction(t)
	case *st.MapTypeDescriptorNode:
		return n.transformMapTypeDescriptor(t)
	case *st.NilLiteralNode:
		return n.transformNilLiteral(t)
	case *st.AnnotationDeclarationNode:
		return n.transformAnnotationDeclaration(t)
	case *st.AnnotationAttachPointNode:
		return n.transformAnnotationAttachPoint(t)
	case *st.XMLNamespaceDeclarationNode:
		return n.transformXMLNamespaceDeclaration(t)
	case *st.ModuleXMLNamespaceDeclarationNode:
		return n.transformModuleXMLNamespaceDeclaration(t)
	case *st.FunctionBodyBlockNode:
		return n.transformFunctionBodyBlock(t)
	case *st.NamedWorkerDeclarationNode:
		return n.transformNamedWorkerDeclaration(t)
	case *st.NamedWorkerDeclarator:
		return n.transformNamedWorkerDeclarator(t)
	case *st.BasicLiteralNode:
		return n.transformBasicLiteral(t)
	case *st.SimpleNameReferenceNode:
		return n.transformSimpleNameReference(t)
	case *st.QualifiedNameReferenceNode:
		return n.transformQualifiedNameReference(t)
	case *st.BuiltinSimpleNameReferenceNode:
		return n.transformBuiltinSimpleNameReference(t)
	case *st.TrapExpressionNode:
		return n.transformTrapExpression(t)
	case *st.ListConstructorExpressionNode:
		return n.transformListConstructorExpression(t)
	case *st.TypeCastExpressionNode:
		return n.transformTypeCastExpression(t)
	case *st.TypeCastParamNode:
		return n.transformTypeCastParam(t)
	case *st.UnionTypeDescriptorNode:
		return n.transformUnionTypeDescriptor(t)
	case *st.TableConstructorExpressionNode:
		return n.transformTableConstructorExpression(t)
	case *st.KeySpecifierNode:
		return n.transformKeySpecifier(t)
	case *st.StreamTypeDescriptorNode:
		return n.transformStreamTypeDescriptor(t)
	case *st.StreamTypeParamsNode:
		return n.transformStreamTypeParams(t)
	case *st.LetExpressionNode:
		return n.transformLetExpression(t)
	case *st.LetVariableDeclarationNode:
		return n.transformLetVariableDeclaration(t)
	case *st.TemplateExpressionNode:
		return n.transformTemplateExpression(t)
	case *st.XMLElementNode:
		return n.transformXMLElement(t)
	case *st.XMLStartTagNode:
		return n.transformXMLStartTag(t)
	case *st.XMLEndTagNode:
		return n.transformXMLEndTag(t)
	case *st.XMLSimpleNameNode:
		return n.transformXMLSimpleName(t)
	case *st.XMLQualifiedNameNode:
		return n.transformXMLQualifiedName(t)
	case *st.XMLEmptyElementNode:
		return n.transformXMLEmptyElement(t)
	case *st.InterpolationNode:
		return n.transformInterpolation(t)
	case *st.XMLTextNode:
		return n.transformXMLText(t)
	case *st.XMLAttributeNode:
		return n.transformXMLAttribute(t)
	case *st.XMLAttributeValue:
		return n.transformXMLAttributeValue(t)
	case *st.XMLComment:
		return n.transformXMLComment(t)
	case *st.XMLCDATANode:
		return n.transformXMLCDATA(t)
	case *st.XMLProcessingInstruction:
		return n.transformXMLProcessingInstruction(t)
	case *st.TableTypeDescriptorNode:
		return n.transformTableTypeDescriptor(t)
	case *st.TypeParameterNode:
		return n.transformTypeParameter(t)
	case *st.KeyTypeConstraintNode:
		return n.transformKeyTypeConstraint(t)
	case *st.FunctionTypeDescriptorNode:
		return n.transformFunctionTypeDescriptor(t)
	case *st.FunctionSignatureNode:
		return n.transformFunctionSignature(t)
	case *st.ExplicitAnonymousFunctionExpressionNode:
		return n.transformExplicitAnonymousFunctionExpression(t)
	case *st.ExpressionFunctionBodyNode:
		return n.transformExpressionFunctionBody(t)
	case *st.TupleTypeDescriptorNode:
		return n.transformTupleTypeDescriptor(t)
	case *st.ParenthesisedTypeDescriptorNode:
		return n.transformParenthesisedTypeDescriptor(t)
	case *st.ExplicitNewExpressionNode:
		return n.transformExplicitNewExpression(t)
	case *st.ImplicitNewExpressionNode:
		return n.transformImplicitNewExpression(t)
	case *st.ParenthesizedArgList:
		return n.transformParenthesizedArgList(t)
	case *st.QueryConstructTypeNode:
		return n.transformQueryConstructType(t)
	case *st.FromClauseNode:
		return n.transformFromClause(t)
	case *st.WhereClauseNode:
		return n.transformWhereClause(t)
	case *st.LetClauseNode:
		return n.transformLetClause(t)
	case *st.JoinClauseNode:
		return n.transformJoinClause(t)
	case *st.OnClauseNode:
		return n.transformOnClause(t)
	case *st.LimitClauseNode:
		return n.transformLimitClause(t)
	case *st.OnConflictClauseNode:
		return n.transformOnConflictClause(t)
	case *st.QueryPipelineNode:
		return n.transformQueryPipeline(t)
	case *st.SelectClauseNode:
		return n.transformSelectClause(t)
	case *st.CollectClauseNode:
		return n.transformCollectClause(t)
	case *st.QueryExpressionNode:
		return n.transformQueryExpression(t)
	case *st.QueryActionNode:
		return n.transformQueryAction(t)
	case *st.IntersectionTypeDescriptorNode:
		return n.transformIntersectionTypeDescriptor(t)
	case *st.ImplicitAnonymousFunctionParameters:
		return n.transformImplicitAnonymousFunctionParameters(t)
	case *st.ImplicitAnonymousFunctionExpressionNode:
		return n.transformImplicitAnonymousFunctionExpression(t)
	case *st.StartActionNode:
		return n.transformStartAction(t)
	case *st.FlushActionNode:
		return n.transformFlushAction(t)
	case *st.SingletonTypeDescriptorNode:
		return n.transformSingletonTypeDescriptor(t)
	case *st.MethodDeclarationNode:
		return n.transformMethodDeclaration(t)
	case *st.TypedBindingPatternNode:
		return n.transformTypedBindingPattern(t)
	case *st.CaptureBindingPatternNode:
		return n.transformCaptureBindingPattern(t)
	case *st.WildcardBindingPatternNode:
		return n.transformWildcardBindingPattern(t)
	case *st.ListBindingPatternNode:
		return n.transformListBindingPattern(t)
	case *st.MappingBindingPatternNode:
		return n.transformMappingBindingPattern(t)
	case *st.FieldBindingPatternFullNode:
		return n.transformFieldBindingPatternFull(t)
	case *st.FieldBindingPatternVarnameNode:
		return n.transformFieldBindingPatternVarname(t)
	case *st.RestBindingPatternNode:
		return n.transformRestBindingPattern(t)
	case *st.ErrorBindingPatternNode:
		return n.transformErrorBindingPattern(t)
	case *st.NamedArgBindingPatternNode:
		return n.transformNamedArgBindingPattern(t)
	case *st.AsyncSendActionNode:
		return n.transformAsyncSendAction(t)
	case *st.SyncSendActionNode:
		return n.transformSyncSendAction(t)
	case *st.ReceiveActionNode:
		return n.transformReceiveAction(t)
	case *st.ReceiveFieldsNode:
		return n.transformReceiveFields(t)
	case *st.AlternateReceiveNode:
		return n.transformAlternateReceive(t)
	case *st.RestDescriptorNode:
		return n.transformRestDescriptor(t)
	case *st.DoubleGTTokenNode:
		return n.transformDoubleGTToken(t)
	case *st.TrippleGTTokenNode:
		return n.transformTrippleGTToken(t)
	case *st.WaitActionNode:
		return n.transformWaitAction(t)
	case *st.WaitFieldsListNode:
		return n.transformWaitFieldsList(t)
	case *st.WaitFieldNode:
		return n.transformWaitField(t)
	case *st.AnnotAccessExpressionNode:
		return n.transformAnnotAccessExpression(t)
	case *st.OptionalFieldAccessExpressionNode:
		return n.transformOptionalFieldAccessExpression(t)
	case *st.ConditionalExpressionNode:
		return n.transformConditionalExpression(t)
	case *st.EnumDeclarationNode:
		return n.transformEnumDeclaration(t)
	case *st.EnumMemberNode:
		return n.transformEnumMember(t)
	case *st.ArrayTypeDescriptorNode:
		return n.transformArrayTypeDescriptor(t)
	case *st.ArrayDimensionNode:
		return n.transformArrayDimension(t)
	case *st.TransactionStatementNode:
		return n.transformTransactionStatement(t)
	case *st.RollbackStatementNode:
		return n.transformRollbackStatement(t)
	case *st.RetryStatementNode:
		return n.transformRetryStatement(t)
	case *st.CommitActionNode:
		return n.transformCommitAction(t)
	case *st.TransactionalExpressionNode:
		return n.transformTransactionalExpression(t)
	case *st.ByteArrayLiteralNode:
		return n.transformByteArrayLiteral(t)
	case *st.XMLFilterExpressionNode:
		return n.transformXMLFilterExpression(t)
	case *st.XMLStepExpressionNode:
		return n.transformXMLStepExpression(t)
	case *st.XMLNamePatternChainingNode:
		return n.transformXMLNamePatternChaining(t)
	case *st.XMLStepIndexedExtendNode:
		return n.transformXMLStepIndexedExtend(t)
	case *st.XMLStepMethodCallExtendNode:
		return n.transformXMLStepMethodCallExtend(t)
	case *st.XMLAtomicNamePatternNode:
		return n.transformXMLAtomicNamePattern(t)
	case *st.TypeReferenceTypeDescNode:
		return n.transformTypeReferenceTypeDesc(t)
	case *st.MatchStatementNode:
		return n.transformMatchStatement(t)
	case *st.MatchClauseNode:
		return n.transformMatchClause(t)
	case *st.MatchGuardNode:
		return n.transformMatchGuard(t)
	case *st.DistinctTypeDescriptorNode:
		return n.transformDistinctTypeDescriptor(t)
	case *st.ListMatchPatternNode:
		return n.transformListMatchPattern(t)
	case *st.RestMatchPatternNode:
		return n.transformRestMatchPattern(t)
	case *st.MappingMatchPatternNode:
		return n.transformMappingMatchPattern(t)
	case *st.FieldMatchPatternNode:
		return n.transformFieldMatchPattern(t)
	case *st.ErrorMatchPatternNode:
		return n.transformErrorMatchPattern(t)
	case *st.NamedArgMatchPatternNode:
		return n.transformNamedArgMatchPattern(t)
	case *st.OrderByClauseNode:
		return n.transformOrderByClause(t)
	case *st.OrderKeyNode:
		return n.transformOrderKey(t)
	case *st.GroupByClauseNode:
		return n.transformGroupByClause(t)
	case *st.GroupingKeyVarDeclarationNode:
		return n.transformGroupingKeyVarDeclaration(t)
	case *st.OnFailClauseNode:
		return n.transformOnFailClause(t)
	case *st.DoStatementNode:
		return n.transformDoStatement(t)
	case *st.ClassDefinitionNode:
		return n.transformClassDefinition(t)
	case *st.ResourcePathParameterNode:
		return n.transformResourcePathParameter(t)
	case *st.RequiredExpressionNode:
		return n.transformRequiredExpression(t)
	case *st.ErrorConstructorExpressionNode:
		return n.transformErrorConstructorExpression(t)
	case *st.ParameterizedTypeDescriptorNode:
		return n.transformParameterizedTypeDescriptor(t)
	case *st.SpreadMemberNode:
		return n.transformSpreadMember(t)
	case *st.ClientResourceAccessActionNode:
		return n.transformClientResourceAccessAction(t)
	case *st.ComputedResourceAccessSegmentNode:
		return n.transformComputedResourceAccessSegment(t)
	case *st.ResourceAccessRestSegmentNode:
		return n.transformResourceAccessRestSegment(t)
	case *st.ReSequenceNode:
		return n.transformReSequence(t)
	case *st.ReAtomQuantifierNode:
		return n.transformReAtomQuantifier(t)
	case *st.ReAtomCharOrEscapeNode:
		return n.transformReAtomCharOrEscape(t)
	case *st.ReQuoteEscapeNode:
		return n.transformReQuoteEscape(t)
	case *st.ReSimpleCharClassEscapeNode:
		return n.transformReSimpleCharClassEscape(t)
	case *st.ReUnicodePropertyEscapeNode:
		return n.transformReUnicodePropertyEscape(t)
	case *st.ReUnicodeScriptNode:
		return n.transformReUnicodeScript(t)
	case *st.ReUnicodeGeneralCategoryNode:
		return n.transformReUnicodeGeneralCategory(t)
	case *st.ReCharacterClassNode:
		return n.transformReCharacterClass(t)
	case *st.ReCharSetRangeWithReCharSetNode:
		return n.transformReCharSetRangeWithReCharSet(t)
	case *st.ReCharSetRangeNode:
		return n.transformReCharSetRange(t)
	case *st.ReCharSetAtomWithReCharSetNoDashNode:
		return n.transformReCharSetAtomWithReCharSetNoDash(t)
	case *st.ReCharSetRangeNoDashWithReCharSetNode:
		return n.transformReCharSetRangeNoDashWithReCharSet(t)
	case *st.ReCharSetRangeNoDashNode:
		return n.transformReCharSetRangeNoDash(t)
	case *st.ReCharSetAtomNoDashWithReCharSetNoDashNode:
		return n.transformReCharSetAtomNoDashWithReCharSetNoDash(t)
	case *st.ReCapturingGroupsNode:
		return n.transformReCapturingGroups(t)
	case *st.ReFlagExpressionNode:
		return n.transformReFlagExpression(t)
	case *st.ReFlagsOnOffNode:
		return n.transformReFlagsOnOff(t)
	case *st.ReFlagsNode:
		return n.transformReFlags(t)
	case *st.ReAssertionNode:
		return n.transformReAssertion(t)
	case *st.ReQuantifierNode:
		return n.transformReQuantifier(t)
	case *st.ReBracedQuantifierNode:
		return n.transformReBracedQuantifier(t)
	case *st.MemberTypeDescriptorNode:
		return n.transformMemberTypeDescriptor(t)
	case *st.ReceiveFieldNode:
		return n.transformReceiveField(t)
	case *st.NaturalExpressionNode:
		return n.transformNaturalExpression(t)
	case *st.IdentifierToken:
		return n.transformIdentifierToken(t)
	case st.Token:
		return n.transformToken(t)
	default:
		panic("transformSyntaxNode: unsupported node type")
	}
}

func getFileName(node st.Node) string {
	st := node.SyntaxTree()
	return st.FilePath()
}

func innermostDiagnosticNodes(node st.Node) []st.Node {
	if !node.HasDiagnostics() {
		return nil
	}

	var nodes []st.Node
	if nt, ok := node.(st.NonTerminalNode); ok {
		for child := range nt.ChildNodes() {
			if child != nil && child.HasDiagnostics() {
				nodes = append(nodes, innermostDiagnosticNodes(child)...)
			}
		}
	}
	if len(nodes) > 0 {
		return nodes
	}
	return []st.Node{node}
}

func diagnosticMessage(diagnostic st.STNodeDiagnostic) string {
	return strings.ReplaceAll(strings.TrimPrefix(diagnostic.DiagnosticCode().MessageKey(), "error."), ".", " ")
}

func (n *nodeBuilder) getPosition(node st.Node) diagnostics.Location {
	textRange := node.TextRange()
	if n.mode == nodeBuilderModeRecover {
		textRange = node.TextRangeWithMinutiae()
	}
	return n.location(node, textRange)
}

func (n *nodeBuilder) getRecoveryPosition(node st.Node) diagnostics.Location {
	return n.location(node, node.TextRangeWithMinutiae())
}

func (n *nodeBuilder) location(node st.Node, textRange st.TextRange) diagnostics.Location {
	return diagnostics.NewLocation(n.de(), getFileName(node), textRange.StartOffset, textRange.EndOffset)
}

func (n *nodeBuilder) getPositionRange(startNode st.Node, endNode st.Node) diagnostics.Location {
	startRange := startNode.TextRange()
	endRange := endNode.TextRange()
	return diagnostics.NewLocation(n.de(), getFileName(startNode), startRange.StartOffset, endRange.EndOffset)
}

func (n *nodeBuilder) getPositionWithoutMetadata(node st.Node) diagnostics.Location {
	pos := n.getPosition(node)
	return diagnostics.NewLocation(n.de(), getFileName(node), metadataExcludedStartOffset(node, pos.StartOffset()), pos.EndOffset())
}

func metadataExcludedStartOffset(node st.Node, defaultStartOffset int) int {
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
		return secondChild.TextRange().StartOffset
	}
	return defaultStartOffset
}

// getDocumentationString extracts the documentation string from metadata
func getDocumentationString(metadata *st.MetadataNode) st.Node {
	return metadata.DocumentationString()
}

func (n *nodeBuilder) populateMetadata(metadata *st.MetadataNode, target ast.AnnotatableNode) {
	if metadata == nil || metadata.IsMissing() {
		return
	}
	if docString := getDocumentationString(metadata); docString != nil && !docString.IsMissing() {
		documentation := n.createMarkdownDocumentationAttachment(docString)
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
	n.addAnnotationAttachments(metadata.Annotations(), target)
}

func (n *nodeBuilder) addAnnotationAttachments(annotations st.NodeList[*st.AnnotationNode], target ast.AnnotatableNode) {
	for annotation := range annotations.Iterator() {
		target.AddAnnotationAttachment(*n.transformAnnotation(annotation).(*ast.BLangAnnotationAttachment))
	}
}

func (n *nodeBuilder) createTrueLiteral(pos diagnostics.Location) *ast.BLangLiteral {
	return ast.NewBLangLiteral(pos, ast.LiteralKindBoolean, true, "true", false)
}

// createMarkdownDocumentationAttachment creates a BLangMarkdownDocumentation from a documentation string node
func (n *nodeBuilder) createMarkdownDocumentationAttachment(docStringNode st.Node) *ast.BLangMarkdownDocumentation {
	if docStringNode == nil || docStringNode.IsMissing() {
		return nil
	}

	markdownDocumentationNode, ok := docStringNode.(*st.MarkdownDocumentationNode)
	if !ok {
		return nil
	}

	doc := &ast.BLangMarkdownDocumentation{}
	documentationLines := []ast.BLangMarkdownDocumentationLine{}
	parameters := []ast.BLangMarkdownParameterDocumentation{}
	references := []ast.BLangMarkdownReferenceDocumentation{}

	docLineList := markdownDocumentationNode.DocumentationLines()

	var bLangParaDoc *ast.BLangMarkdownParameterDocumentation
	var bLangReturnParaDoc *ast.BLangMarkdownReturnParameterDocumentation
	var bLangDeprecationDoc *ast.BLangMarkDownDeprecationDocumentation
	var bLangDeprecatedParaDoc *ast.BLangMarkDownDeprecatedParametersDocumentation

	for i := 0; i < docLineList.Size(); i++ {
		singleDocLine := docLineList.Get(i)
		switch singleDocLine.Kind() {
		case st.MARKDOWN_DOCUMENTATION_LINE, st.MARKDOWN_REFERENCE_DOCUMENTATION_LINE:
			docLineNode := singleDocLine.(*st.MarkdownDocumentationLineNode)
			docElements := docLineNode.DocumentElements()
			docText := n.addReferencesAndReturnDocumentationText(&references, docElements)

			if bLangDeprecationDoc != nil {
				bLangDeprecationDoc.DeprecationDocumentationLines = append(bLangDeprecationDoc.DeprecationDocumentationLines, docText)
			} else if bLangReturnParaDoc != nil {
				bLangReturnParaDoc.ReturnParameterDocumentationLines = append(bLangReturnParaDoc.ReturnParameterDocumentationLines, docText)
			} else if bLangParaDoc != nil {
				bLangParaDoc.ParameterDocumentationLines = append(bLangParaDoc.ParameterDocumentationLines, docText)
			} else {
				bLangDocLine := ast.BLangMarkdownDocumentationLine{}
				bLangDocLine.Text = docText
				bLangDocLine.SetPosition(n.getPosition(docLineNode))
				documentationLines = append(documentationLines, bLangDocLine)
			}
		case st.MARKDOWN_PARAMETER_DOCUMENTATION_LINE:
			if bLangParaDoc != nil {
				if bLangDeprecatedParaDoc != nil {
					bLangDeprecatedParaDoc.Parameters = append(bLangDeprecatedParaDoc.Parameters, *bLangParaDoc)
				} else if bLangDeprecationDoc != nil {
					bLangDeprecatedParaDoc = &ast.BLangMarkDownDeprecatedParametersDocumentation{}
					bLangDeprecatedParaDoc.Parameters = append(bLangDeprecatedParaDoc.Parameters, *bLangParaDoc)
					bLangDeprecationDoc = nil
				} else {
					parameters = append(parameters, *bLangParaDoc)
				}
			}

			bLangParaDoc = &ast.BLangMarkdownParameterDocumentation{}
			parameterDocLineNode := singleDocLine.(*st.MarkdownParameterDocumentationLineNode)

			paraName := &ast.BLangIdentifier{}
			parameterName := parameterDocLineNode.ParameterName()
			parameterNameValue := ""
			if parameterName != nil && !parameterName.IsMissing() {
				parameterNameValue = unescapeUnicodeCodepoints(parameterName.Text())
			}
			paraName.OriginalValue = parameterNameValue
			if n.stringStartsWithSingleQuote(parameterNameValue) {
				parameterNameValue = parameterNameValue[1:]
			}
			paraName.Value = parameterNameValue
			bLangParaDoc.ParameterName = paraName
			paraDocElements := parameterDocLineNode.DocumentElements()
			paraDocText := n.addReferencesAndReturnDocumentationText(&references, paraDocElements)

			bLangParaDoc.ParameterDocumentationLines = append(bLangParaDoc.ParameterDocumentationLines, paraDocText)
			bLangParaDoc.SetPosition(n.getPosition(parameterName))
		case st.MARKDOWN_RETURN_PARAMETER_DOCUMENTATION_LINE:
			bLangReturnParaDoc = &ast.BLangMarkdownReturnParameterDocumentation{}
			returnParaDocLineNode := singleDocLine.(*st.MarkdownParameterDocumentationLineNode)

			returnParaDocElements := returnParaDocLineNode.DocumentElements()
			returnParaDocText := n.addReferencesAndReturnDocumentationText(&references, returnParaDocElements)

			bLangReturnParaDoc.ReturnParameterDocumentationLines = append(bLangReturnParaDoc.ReturnParameterDocumentationLines, returnParaDocText)
			bLangReturnParaDoc.SetPosition(n.getPosition(returnParaDocLineNode))
			doc.ReturnParameter = bLangReturnParaDoc
		case st.MARKDOWN_DEPRECATION_DOCUMENTATION_LINE:
			bLangDeprecationDoc = &ast.BLangMarkDownDeprecationDocumentation{}
			deprecationDocLineNode := singleDocLine.(*st.MarkdownDocumentationLineNode)

			docElements := deprecationDocLineNode.DocumentElements()
			var lineText string
			if docElements.Size() > 0 {
				firstElement := docElements.Get(0)
				if token, ok := firstElement.(st.Token); ok {
					lineText = token.Text()
				}
			}
			bLangDeprecationDoc.AddDeprecationLine("# " + lineText)
			bLangDeprecationDoc.SetPosition(n.getPosition(deprecationDocLineNode))
		case st.MARKDOWN_CODE_BLOCK:
			codeBlockNode := singleDocLine.(*st.MarkdownCodeBlockNode)
			n.transformCodeBlock(&documentationLines, codeBlockNode)
		default:
		}
	}

	if bLangParaDoc != nil {
		if bLangDeprecatedParaDoc != nil {
			bLangDeprecatedParaDoc.Parameters = append(bLangDeprecatedParaDoc.Parameters, *bLangParaDoc)
		} else if bLangDeprecationDoc != nil {
			bLangDeprecatedParaDoc = &ast.BLangMarkDownDeprecatedParametersDocumentation{}
			bLangDeprecatedParaDoc.Parameters = append(bLangDeprecatedParaDoc.Parameters, *bLangParaDoc)
			bLangDeprecationDoc = nil
		} else {
			parameters = append(parameters, *bLangParaDoc)
		}
	}

	doc.DocumentationLines = documentationLines
	doc.Parameters = parameters
	doc.References = references
	doc.DeprecationDocumentation = bLangDeprecationDoc
	doc.DeprecatedParametersDocumentation = bLangDeprecatedParaDoc
	doc.SetPosition(n.getPosition(markdownDocumentationNode))
	return doc
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

// createIdentifierFromToken creates an identifier from a token, handling missing tokens and validation
func createIdentifierFromToken(pos diagnostics.Location, token st.Token) ast.BLangIdentifier {
	return createIdentifierFromTokenInternal(pos, token, false)
}

// createIdentifierFromTokenInternal creates an identifier from a token with XML handling option
func createIdentifierFromTokenInternal(pos diagnostics.Location, token st.Token, isXML bool) ast.BLangIdentifier {
	if token == nil {
		// Return empty identifier for nil token
		return createIdentifier(pos, nil, nil)
	}

	const IDENTIFIER_LITERAL_PREFIX = "'"
	identifierName := token.Text()

	// Handle missing tokens or empty identifier literal prefix
	if token.IsMissing() || identifierName == IDENTIFIER_LITERAL_PREFIX {
		panic("unimplemented")
	} else if !isXML && (identifierName == "_" || identifierName == IDENTIFIER_LITERAL_PREFIX+"_") {
		panic("unimplemented")
	}

	return createIdentifier(pos, &identifierName, &identifierName)
}

func (n *nodeBuilder) createIgnoreIdentifier(node st.Node) ast.BLangIdentifier {
	pos := n.getPosition(node)
	ignoreValue := string(model.IGNORE)
	identifier := createIdentifier(pos, &ignoreValue, &ignoreValue)
	return identifier
}

// getNextAnonymousTypeKey generates the next anonymous type key
// Placeholder function - to be implemented
func (n *nodeBuilder) getNextAnonymousTypeKey(packageID *model.PackageID, suffixes []string) string {
	return n.cx.GetNextAnonymousTypeKey(packageID)
}

// createTypeNode creates a type node from a syntax tree node
// This delegates to the appropriate Transform method based on the node type
func (n *nodeBuilder) createTypeNode(typeNode st.Node) ast.TypeDescriptor {
	result, err := n.createTypeNodeInner(typeNode)
	if err == nil {
		return result
	}
	if n.mode == nodeBuilderModeRecover {
		return n.badTypeNode(typeNode)
	}
	panic(err)
}

func (n *nodeBuilder) createTypeNodeInner(typeNode st.Node) (ast.TypeDescriptor, error) {
	if typeNode == nil {
		return nil, fmt.Errorf("createTypeNode: typeNode is nil")
	}
	if typeNode, ok := typeNode.(*st.BuiltinSimpleNameReferenceNode); ok {
		return n.createBuiltInTypeNode(typeNode), nil
	}
	kind := typeNode.Kind()
	switch kind {
	case st.NIL_TYPE_DESC:
		return n.createBuiltInTypeNode(typeNode), nil
	case st.QUALIFIED_NAME_REFERENCE, st.IDENTIFIER_TOKEN:
		bLUserDefinedType := ast.BLangUserDefinedType{}
		nameRefence := n.createBLangNameReference(typeNode)
		pkgAlias, pkgOK := nameRefence[0].(*ast.BLangIdentifier)
		typeName, nameOK := nameRefence[1].(*ast.BLangIdentifier)
		if !pkgOK || !nameOK {
			return nil, fmt.Errorf("invalid user-defined type name")
		}
		bLUserDefinedType.PkgAlias = *pkgAlias
		bLUserDefinedType.TypeName = *typeName
		bLUserDefinedType.SetPosition(n.getPosition(typeNode))
		return &bLUserDefinedType, nil
	case st.SIMPLE_NAME_REFERENCE:
		nameReferenceNode := typeNode.(*st.SimpleNameReferenceNode)
		return n.createTypeNodeInner(nameReferenceNode.Name())
	default:
		result, ok := n.transformSyntaxNode(typeNode).(ast.BType)
		if !ok {
			return nil, fmt.Errorf("syntax node %T is not a type descriptor", typeNode)
		}
		return result, nil
	}
}

// isDeclaredWithVar checks if a type node is declared with var
func isDeclaredWithVar(typeNode st.Node) bool {
	if typeNode == nil || typeNode.Kind() == st.VAR_TYPE_DESC {
		return true
	}
	return false
}

func (n *nodeBuilder) createSimpleVarInner(name st.Token, typeName st.Node, initializer st.Node, visibilityQualifier st.Token, annotations st.NodeList[*st.AnnotationNode], flags model.Flag) *ast.BLangVariable {
	var namePos diagnostics.Location
	if name != nil {
		namePos = n.getPosition(name)
	}
	identifier := n.createIdentifierNodeFromToken(namePos, name)
	isDeclaredWithVar := isDeclaredWithVar(typeName)
	var typeNode ast.BType
	if !isDeclaredWithVar {
		typeNode = n.createTypeNode(typeName).(ast.BType)
	}
	var expr ast.BLangActionOrExpression
	if initializer != nil {
		expr = n.createExpression(initializer)
	}
	if visibilityQualifier != nil && visibilityQualifier.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	variable := ast.NewBLangVariable(namePos, identifier, typeNode, expr, isDeclaredWithVar, flags)
	n.addAnnotationAttachments(annotations, variable)
	return variable
}

func (n *nodeBuilder) createBuiltInTypeNode(typeNode st.Node) ast.TypeDescriptor {
	var typeText string
	if typeNode.Kind() == st.NIL_TYPE_DESC {
		typeText = "()"
	} else if simpleNameRef, ok := typeNode.(*st.BuiltinSimpleNameReferenceNode); ok {
		if simpleNameRef.Kind() == st.VAR_TYPE_DESC {
			return nil
		} else if simpleNameRef.Name().IsMissing() {
			name := getNextMissingNodeName(n.PackageID)
			identifier := createIdentifier(n.getPosition(simpleNameRef.Name()), &name, &name)
			pkgAlias := ast.BLangIdentifier{}
			return createUserDefinedType(n.getPosition(typeNode), pkgAlias, identifier)
		}
		typeText = simpleNameRef.Name().Text()
	} else {
		// TODO: Remove this once map<string> returns Nodes for `map`
		if token, ok := typeNode.(st.Token); ok {
			typeText = token.Text()
		} else {
			panic("createBuiltInTypeNode: unexpected node type")
		}
	}

	typeKind := stringToTypeKind(typeText)

	kind := typeNode.Kind()
	switch kind {
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
		return &valueType
	default:
		builtInValueType := ast.BLangBuiltInRefTypeNode{}
		builtInValueType.TypeKind = typeKind
		builtInValueType.SetPosition(n.getPosition(typeNode))
		return &builtInValueType
	}
}

func setIdentifierValue(identifier ast.IdentifierNode, value string) {
	if identifier, ok := identifier.(*ast.BLangIdentifier); ok {
		identifier.Value = value
	}
	// We ignore immutable identifiers such as BLangBadIdentifier.
}

func (n *nodeBuilder) createIdentifierNodeFromToken(pos diagnostics.Location, token st.Token) ast.IdentifierNode {
	if token == nil {
		if n.mode == nodeBuilderModeRecover {
			return n.badIdentifier(token)
		}
		panic("missing identifier token")
	}
	if token.IsMissing() || isUnsupportedIdentifierToken(token) {
		if n.mode == nodeBuilderModeRecover {
			return n.badIdentifier(token)
		}
		panic("invalid identifier")
	}
	identifier := createIdentifierFromToken(pos, token)
	return &identifier
}

func isUnsupportedIdentifierToken(token st.Token) bool {
	return token.Text() == "'" || token.Text() == "_" || token.Text() == "'_"
}

func (n *nodeBuilder) createBLangNameReference(node st.Node) [2]ast.IdentifierNode {
	switch node.Kind() {
	case st.QUALIFIED_NAME_REFERENCE:
		iNode := node.(*st.QualifiedNameReferenceNode)
		modulePrefix := iNode.ModulePrefix()
		identifier := iNode.Identifier()
		pkgAlias := n.createIdentifierNodeFromToken(n.getPosition(modulePrefix), modulePrefix)
		namePos := n.getPosition(identifier)
		name := n.createIdentifierNodeFromToken(namePos, identifier)
		return [...]ast.IdentifierNode{pkgAlias, name}
	case st.ERROR_TYPE_DESC:
		builtinNode := node.(*st.BuiltinSimpleNameReferenceNode)
		node = builtinNode.Name()
		// Fall through to default handling
	case st.NEW_KEYWORD, st.IDENTIFIER_TOKEN, st.ERROR_KEYWORD:
		// Break and fall through to default handling
	case st.SIMPLE_NAME_REFERENCE:
		fallthrough
	default:
		simpleNode := node.(*st.SimpleNameReferenceNode)
		node = simpleNode.Name()
	}

	// Default case: node should be a Token at this point
	iToken := node.(st.Token)

	emptyStr := ""
	pkgAlias := createIdentifier(diagnostics.NewBuiltinLocation(), &emptyStr, &emptyStr)
	name := n.createIdentifierNodeFromToken(n.getPosition(iToken), iToken)
	return [...]ast.IdentifierNode{&pkgAlias, name}
}

// isFunctionCallAsync checks if a function call expression is async
func (n *nodeBuilder) isFunctionCallAsync(functionCallBLangExpression *st.FunctionCallExpressionNode) bool {
	parent := functionCallBLangExpression.Parent()
	if parent == nil {
		panic("isFunctionCallAsync: parent is nil")
	}
	return parent.Kind() == st.START_ACTION
}

// createBLangInvocation creates a BLangInvocation from a name node and arguments
func (n *nodeBuilder) createBLangInvocation(nameNode st.Node, arguments st.NodeList[st.FunctionArgumentNode], position diagnostics.Location, isAsync bool) *ast.BLangInvocation {
	var bLInvocation ast.BLangInvocation
	if isAsync {
		panic("unimplemented")
	} else {
		bLInvocation = ast.BLangInvocation{}
	}

	nameReference := n.createBLangNameReference(nameNode)
	bLInvocation.PkgAlias = nameReference[0]
	bLInvocation.Name = nameReference[1]

	var args []ast.BLangExpression
	for arg := range arguments.Iterator() {
		args = append(args, n.createExpression(arg))
	}
	bLInvocation.ArgExprs = args
	bLInvocation.SetPosition(position)
	return &bLInvocation
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

// createSimpleLiteral creates a simple literal from a node
func (n *nodeBuilder) createSimpleLiteral(literal st.Node) ast.LiteralNode {
	return n.createSimpleLiteralInner(literal)
}

// getIntegerLiteral parses integer literals (decimal/hex)
func (n *nodeBuilder) getIntegerLiteral(literal st.Node, textValue string) any {
	basicLiteralNode := literal.(*st.BasicLiteralNode)
	literalTokenKind := basicLiteralNode.LiteralToken().Kind()
	switch literalTokenKind {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN:
		if textValue[0] == '0' && len(textValue) > 1 {
			n.cx.SyntaxError("invalid integer literal: leading zero", n.getPosition(literal))
		}
		return parseLong(textValue, textValue, 10)
	case st.HEX_INTEGER_LITERAL_TOKEN:
		processedNodeValue := strings.ToLower(textValue)
		processedNodeValue = strings.ReplaceAll(processedNodeValue, "0x", "")
		return parseLong(textValue, processedNodeValue, 16)
	}
	return nil
}

// parseLong parses a long integer value
func parseLong(originalNodeValue, processedNodeValue string, radix int) any {
	val, err := strconv.ParseInt(processedNodeValue, radix, 64)
	if err != nil {
		fVal, fErr := strconv.ParseFloat(processedNodeValue, 64)
		if fErr != nil {
			panic("Unimplemented")
		}
		if math.IsInf(fVal, 0) {
			return originalNodeValue
		}
		return fVal
	}
	return val
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

// getHexNodeValue processes hex floating point values
func getHexNodeValue(value string) string {
	if !strings.Contains(value, "p") && !strings.Contains(value, "P") {
		value = value + "p0"
	}
	return value
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

// isNumericLiteral checks if syntax kind is numeric literal
func isNumericLiteral(kind st.SyntaxKind) bool {
	return kind == st.NUMERIC_LITERAL
}

// createSimpleLiteralInner creates a simple literal from a node
func (n *nodeBuilder) createSimpleLiteralInner(literal st.Node) ast.LiteralNode {
	kind := literal.Kind()
	literalKind := ast.LiteralKindNone
	var value any = nil
	var originalValue *string = nil

	var textValue string
	if basicLiteralNode, ok := literal.(*st.BasicLiteralNode); ok {
		textValue = basicLiteralNode.LiteralToken().Text()
	} else if token, ok := literal.(st.Token); ok {
		textValue = token.Text()
	} else {
		textValue = ""
	}

	// TODO: Verify all types, only string type tested
	if kind == st.NUMERIC_LITERAL {
		basicLiteralNode := literal.(*st.BasicLiteralNode)
		literalTokenKind := basicLiteralNode.LiteralToken().Kind()
		switch literalTokenKind {
		case st.DECIMAL_INTEGER_LITERAL_TOKEN, st.HEX_INTEGER_LITERAL_TOKEN:
			literalKind = ast.LiteralKindInt
			value = n.getIntegerLiteral(literal, textValue)
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
			value = getHexNodeValue(textValue)
			originalValue = &textValue
		}
		numericLiteral := ast.NewBLangNumericLiteral(
			n.getPosition(literal), literalKind, value, *originalValue, false,
		)
		return &numericLiteral.BLangLiteral
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
	} else if kind == st.BYTE_ARRAY_LITERAL {
		return n.transformSyntaxNode(literal).(ast.LiteralNode)
	}
	return ast.NewBLangLiteral(n.getPosition(literal), literalKind, value, *originalValue, false)
}

func (n *nodeBuilder) transformModulePart(modulePartNode *st.ModulePart) ast.BLangNode {
	compilationUnit := ast.BLangCompilationUnit{}
	n.currentCompUnit = &compilationUnit
	defer func() { n.currentCompUnit = nil }()
	compilationUnit.SetPackageID(n.PackageID)
	pos := n.getPosition(modulePartNode)

	if modulePartNode.HasDiagnostics() {
		n.syntaxError(modulePartNode)
	}

	// Generate import declarations
	imports := modulePartNode.Imports()
	for importDecl := range imports.Iterator() {
		if importDecl.HasDiagnostics() {
			if n.mode == nodeBuilderModeRecover {
				compilationUnit.AddTopLevelNode(n.badTopLevel(importDecl))
			}
			continue
		}
		node, err := n.transformImportTopLevel(importDecl)
		if err != nil {
			if n.mode == nodeBuilderModeRecover {
				node = n.badTopLevel(importDecl)
			} else {
				panic(err)
			}
		}
		compilationUnit.AddTopLevelNode(node)
	}

	// Generate other module-level declarations
	members := modulePartNode.Members()
	for member := range members.Iterator() {
		// Dispatch to transformSyntaxNode which handles all node types
		var memberNode st.Node = member
		if memberNode.HasDiagnostics() {
			if n.mode != nodeBuilderModeRecover {
				continue
			}
			if memberNode.Kind() != st.FUNCTION_DEFINITION {
				compilationUnit.AddTopLevelNode(n.badTopLevel(memberNode))
				continue
			}
		}
		node, err := n.transformTopLevel(memberNode)
		if err != nil {
			panic(err)
		}
		compilationUnit.AddTopLevelNode(node)
	}

	// Create diagnostic location
	fileName := ""
	if !diagnostics.IsLocationEmpty(pos) {
		fileName = n.de().FileName(pos)
	}

	newLocation := diagnostics.NewLocation(n.de(), fileName, 0, 0)
	compilationUnit.SetPosition(newLocation)
	compilationUnit.SetPackageID(n.PackageID)

	return &compilationUnit
}

func functionQualifierFlags(qualifierList st.NodeList[st.Token]) model.Flag {
	var flags model.Flag
	for qualifier := range qualifierList.Iterator() {
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
		}
	}
	return flags
}

func (n *nodeBuilder) populateFuncSignature(data *ast.InvokableData, funcSignature *st.FunctionSignatureNode) {
	data.ParamListPosition = diagnostics.NewBuiltinLocation()
	openParen := funcSignature.OpenParenToken()
	closeParen := funcSignature.CloseParenToken()
	if openParen != nil && closeParen != nil && !openParen.IsMissing() && !closeParen.IsMissing() {
		data.ParamListPosition = n.getPositionRange(openParen, closeParen)
	}
	parameters := funcSignature.Parameters()
	for param := range parameters.Iterator() {
		paramNode := n.transformSyntaxNode(param).(*ast.BLangVariable)
		if _, isRestParam := param.(*st.RestParameterNode); isRestParam {
			data.RestParam = paramNode
		} else {
			data.RequiredParams = append(data.RequiredParams, *paramNode)
		}
	}

	retTypeDescNode := funcSignature.ReturnTypeDesc()
	if retTypeDescNode == nil {
		nilReturnType := &ast.BLangValueType{TypeKind: ast.TypeKindNil}
		nilReturnType.SetPosition(diagnostics.NewBuiltinLocation())
		data.ReturnTypeDescriptor = nilReturnType
		return
	}
	if returnsKeyword := retTypeDescNode.ReturnsKeyword(); returnsKeyword != nil && !returnsKeyword.IsMissing() {
		data.Flags |= model.FlagExplicitReturnTypeDescriptor
	}
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, "return")
	data.ReturnTypeDescriptor = n.createTypeNode(retTypeDescNode.Type()).(ast.BType)
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	annotations := retTypeDescNode.Annotations()
	for annotation := range annotations.Iterator() {
		data.ReturnAnnotationAttachments = append(
			data.ReturnAnnotationAttachments,
			*n.transformAnnotation(annotation).(*ast.BLangAnnotationAttachment),
		)
	}
}

func (n *nodeBuilder) transformFunctionDefinition(funcDefNode *st.FunctionDefinition) ast.BLangNode {
	// Check for resource functions - panic for now
	relativeResourcePath := funcDefNode.RelativeResourcePath()
	hasResourcePath := relativeResourcePath.Size() > 0
	if hasResourcePath {
		panic("transformFunctionDefinition: resource functions not yet supported")
	}

	// Create function node
	bLFunction := n.createFunctionNode(funcDefNode.FunctionName(), funcDefNode.QualifierList(), funcDefNode.FunctionSignature(), funcDefNode.FunctionBody())
	bLFunction.SetPosition(n.getPositionWithoutMetadata(funcDefNode))

	metadata := funcDefNode.Metadata()
	n.populateMetadata(metadata, bLFunction)

	return bLFunction
}

func (n *nodeBuilder) createFunctionNode(funcName *st.IdentifierToken, qualifierList st.NodeList[st.Token], funcSignature *st.FunctionSignatureNode, funcBody st.FunctionBodyNode) *ast.BLangFunction {
	name := n.createIdentifierNodeFromToken(n.getPosition(funcName), funcName)
	data := ast.InvokableData{Name: name, Flags: functionQualifierFlags(qualifierList)}
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, name.GetValue())
	defer func() {
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	}()
	n.populateFuncSignature(&data, funcSignature)
	if funcBody == nil {
		data.Flags |= model.FlagInterface
	} else {
		data.Body = n.transformSyntaxNode(funcBody).(ast.FunctionBodyNode)
		if _, ok := data.Body.(*ast.BLangExternFunctionBody); ok {
			data.Flags |= model.FlagNative
		}
	}
	return ast.NewBLangFunction(data)
}

func (n *nodeBuilder) transformImportTopLevel(importDecl *st.ImportDeclarationNode) (ast.TopLevelNode, error) {
	transformedNode := n.transformImportDeclaration(importDecl)
	bLangImport, ok := transformedNode.(*ast.BLangImportPackage)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-import node %T", importDecl, transformedNode)
	}
	return bLangImport, nil
}

func (n *nodeBuilder) transformTopLevel(node st.Node) (ast.TopLevelNode, error) {
	result, err := n.transformTopLevelInner(node)
	if err == nil {
		return result, nil
	}
	if n.mode == nodeBuilderModeRecover {
		return n.badTopLevel(node), nil
	}
	return nil, err
}

func (n *nodeBuilder) transformTopLevelInner(node st.Node) (ast.TopLevelNode, error) {
	transformedNode := n.transformSyntaxNode(node)
	topLevel, ok := transformedNode.(ast.TopLevelNode)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-top-level node %T", node, transformedNode)
	}
	return topLevel, nil
}

func (n *nodeBuilder) transformImportDeclaration(importDeclarationNode *st.ImportDeclarationNode) ast.BLangNode {
	// 1. Extract org name (optional)
	orgNameNode := importDeclarationNode.OrgName()
	var orgNameToken st.Token
	if orgNameNode != nil && !orgNameNode.IsMissing() {
		orgNameToken = orgNameNode.OrgName()
	}

	// 2. Extract prefix node (optional)
	prefixNode := importDeclarationNode.Prefix()

	// 3. Get position for entire import declaration
	position := n.getPosition(importDeclarationNode)

	// 4. Process module name components
	var pkgNameComps []ast.BLangIdentifier
	moduleNames := importDeclarationNode.ModuleName()
	for name := range moduleNames.Iterator() {
		namePos := n.getPosition(name)
		nameText := name.Text()
		identifier := createIdentifier(namePos, &nameText, &nameText)
		pkgNameComps = append(pkgNameComps, identifier)
	}

	// 5. Create BLangImportPackage node
	importDcl := &ast.BLangImportPackage{}
	importDcl.SetPosition(position)
	importDcl.PkgNameComps = pkgNameComps

	// 6. Set org name (create identifier even if token is nil)
	var orgNamePos diagnostics.Location
	if orgNameNode != nil && !orgNameNode.IsMissing() {
		orgNamePos = n.getPosition(orgNameNode)
	}
	var orgNameStr *string
	if orgNameToken != nil {
		text := orgNameToken.Text()
		orgNameStr = &text
	}
	orgIdentifier := createIdentifier(orgNamePos, orgNameStr, orgNameStr)
	importDcl.OrgName = &orgIdentifier

	// 7. Set version (always empty for import declarations)
	emptyVersion := createIdentifier(diagnostics.NewBuiltinLocation(), nil, nil)
	importDcl.Version = &emptyVersion

	// 8. Handle alias/prefix
	if prefixNode == nil || prefixNode.IsMissing() {
		// No prefix: use last package name component as alias
		lastPkgComp := &pkgNameComps[len(pkgNameComps)-1]
		importDcl.Alias = lastPkgComp
		return importDcl
	}

	// Prefix exists - check if it's underscore or regular alias
	prefix := prefixNode.Prefix()
	prefixPos := n.getPosition(prefix)

	if prefix.Kind() == st.UNDERSCORE_KEYWORD {
		// Create ignore identifier for underscore
		aliasIdent := n.createIgnoreIdentifier(prefix)
		importDcl.Alias = &aliasIdent
	} else {
		// Use prefix token as alias
		prefixText := prefix.Text()
		aliasIdent := createIdentifier(prefixPos, &prefixText, &prefixText)
		importDcl.Alias = &aliasIdent
	}

	return importDcl
}

func (n *nodeBuilder) transformListenerDeclaration(listenerDeclarationNode *st.ListenerDeclarationNode) ast.BLangNode {
	metadata := listenerDeclarationNode.Metadata()

	pos := n.getPositionWithoutMetadata(listenerDeclarationNode)
	nameToken := listenerDeclarationNode.VariableName()
	namePos := n.getPosition(nameToken)
	identifier := createIdentifierFromToken(namePos, nameToken)

	typeDesc := listenerDeclarationNode.TypeDescriptor()
	isDeclaredWithVar := typeDesc == nil || typeDesc.IsMissing()
	var typeNode ast.BType
	if !isDeclaredWithVar {
		typeNode = n.createTypeNode(typeDesc).(ast.BType)
	}
	var expr ast.BLangActionOrExpression
	if initializer := listenerDeclarationNode.Initializer(); initializer != nil {
		expr = n.createExpression(initializer)
	}
	flags := model.FlagFinal | model.FlagListener
	if visQual := listenerDeclarationNode.VisibilityQualifier(); visQual != nil && visQual.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	bLSimpleVar := ast.NewBLangVariable(pos, &identifier, typeNode, expr, isDeclaredWithVar, flags)
	bLSimpleVar.SetPosition(pos)

	if metadata != nil && !metadata.IsMissing() {
		if annotations := metadata.Annotations(); annotations.Size() > 0 {
			panic("transformListenerDeclaration: annotations not yet supported")
		}
		bLSimpleVar.MarkdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}

	return bLSimpleVar
}

func isAllowedDistinctTypeDescriptor(kind st.SyntaxKind) bool {
	switch kind {
	case st.OBJECT_TYPE_DESC, st.ERROR_TYPE_DESC, st.SIMPLE_NAME_REFERENCE, st.QUALIFIED_NAME_REFERENCE, st.IDENTIFIER_TOKEN:
		return true
	default:
		return false
	}
}

func (n *nodeBuilder) transformTypeDefinition(typeDefinitionNode *st.TypeDefinitionNode) ast.BLangNode {
	identifierNode := createIdentifierFromToken(n.getPosition(typeDefinitionNode.TypeName()), typeDefinitionNode.TypeName())
	var flags model.Flag
	if visibility := typeDefinitionNode.VisibilityQualifier(); visibility != nil && visibility.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	typeDescriptorNode := typeDefinitionNode.TypeDescriptor()
	if _, ok := typeDescriptorNode.(*st.DistinctTypeDescriptorNode); ok {
		flags |= model.FlagDistinct
	}
	var documentation *ast.BLangMarkdownDocumentation
	if metadata := typeDefinitionNode.Metadata(); metadata != nil && !metadata.IsMissing() {
		documentation = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}
	typeDef := ast.NewBLangTypeDefinitionWithData(n.getPositionWithoutMetadata(typeDefinitionNode), &identifierNode, ast.TypeData{}, documentation, flags)
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, typeDef.Name.GetValue())

	if distinctTypeDescriptorNode, ok := typeDescriptorNode.(*st.DistinctTypeDescriptorNode); ok {
		innerTypeDescriptorNode := distinctTypeDescriptorNode.TypeDescriptor()
		if innerTypeDescriptorNode == nil || !isAllowedDistinctTypeDescriptor(innerTypeDescriptorNode.Kind()) {
			n.cx.SyntaxError("only object and error types can be distinct", n.getPosition(distinctTypeDescriptorNode))
			neverType := &ast.BLangValueType{TypeKind: ast.TypeKindNever}
			neverType.SetPosition(n.getPosition(distinctTypeDescriptorNode))
			typeDef.SetTypeData(ast.TypeData{TypeDescriptor: neverType})
			n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
			return typeDef
		}
		typeDescriptorNode = innerTypeDescriptorNode
	}
	typeData := ast.TypeData{
		TypeDescriptor: n.createTypeNode(typeDescriptorNode),
	}
	typeDef.SetTypeData(typeData)

	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]

	typeDef.SetPosition(n.getPositionWithoutMetadata(typeDefinitionNode))

	n.populateMetadata(typeDefinitionNode.Metadata(), typeDef)

	return typeDef
}

func (n *nodeBuilder) transformServiceDeclaration(serviceDeclarationNode *st.ServiceDeclarationNode) ast.BLangNode {
	metadata := serviceDeclarationNode.Metadata()

	service := ast.NewBLangServiceWithFlags(serviceQualifierFlags(serviceDeclarationNode))
	service.SetPosition(n.getPositionWithoutMetadata(serviceDeclarationNode))

	if metadata != nil && !metadata.IsMissing() {
		if annotations := metadata.Annotations(); annotations.Size() > 0 {
			panic("transformServiceDeclaration: annotations not yet supported")
		}
		service.MarkdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}

	if typeDesc := serviceDeclarationNode.TypeDescriptor(); typeDesc != nil && !typeDesc.IsMissing() {
		service.SetTypeData(ast.TypeData{TypeDescriptor: n.createTypeNode(typeDesc)})
	}

	n.populateServiceAttachPoint(&service, serviceDeclarationNode)
	n.populateServiceAttachedExprs(&service, serviceDeclarationNode)

	members := n.collectClassDefnMembers(serviceDeclarationNode.Members())
	service.Fields = members.Fields
	service.Methods = members.Methods
	service.InitFunction = members.InitFunction
	service.ResourceMethods = members.ResourceMethods
	for _, each := range members.UnresolvedInclusions {
		// Parser should catch these
		n.cx.InternalError("unexpected inclusions in service decl", each.GetPosition())
	}

	return &service
}

// populateServiceQualifiers reads the user-controllable qualifiers from the
// service declaration. The `service` flag is already set by NewBLangService.
func serviceQualifierFlags(node *st.ServiceDeclarationNode) model.Flag {
	var flags model.Flag
	quals := node.Qualifiers()
	for qual := range quals.Iterator() {
		if qual.Kind() == st.ISOLATED_KEYWORD {
			flags |= model.FlagIsolated
		}
	}
	return flags
}

func (n *nodeBuilder) populateServiceAttachPoint(service *ast.BLangService, node *st.ServiceDeclarationNode) {
	paths := node.AbsoluteResourcePath()
	if node.HasDiagnostics() {
		return
	}
	if paths.Size() > 0 {
		service.AbsoluteResourcePath = []ast.BLangIdentifier{}
	}
	for i := 0; i < paths.Size(); i++ {
		seg := paths.Get(i)
		if seg.Kind() == st.STRING_LITERAL {
			service.AttachPointLiteral = n.createSimpleLiteral(seg).(*ast.BLangLiteral) //nolint:forcetypeassert // string literals always create BLangLiteral nodes
			continue
		}
		tok, ok := seg.(st.Token)
		if !ok {
			n.cx.InternalError("unexpected node in service attach point", n.getPosition(seg))
			continue
		}
		switch tok.Kind() {
		case st.IDENTIFIER_TOKEN:
			ident := createIdentifierFromToken(n.getPosition(tok), tok)
			service.AbsoluteResourcePath = append(service.AbsoluteResourcePath, ident)
		case st.SLASH_TOKEN:
			// Slash tokens between segments are ignored.
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected token in service attach point: %v", tok.Kind()), n.getPosition(tok))
		}
	}
}

func (n *nodeBuilder) populateServiceAttachedExprs(service *ast.BLangService, node *st.ServiceDeclarationNode) {
	exprs := node.Expressions()
	if exprs.Size() > 0 {
		service.AttachedExprsPosition = n.getPositionRange(exprs.Get(0), exprs.Get(exprs.Size()-1))
	}
	for i := 0; i < exprs.Size(); i += 2 {
		service.AttachedExprs = append(service.AttachedExprs, n.createExpression(exprs.Get(i)))
	}
}

type classDefnMembers struct {
	Fields               []*ast.BLangVariable
	Methods              map[string]*ast.BLangFunction
	InitFunction         *ast.BLangFunction
	ResourceMethods      []*ast.BLangResourceMethod
	UnresolvedInclusions []*ast.BLangUserDefinedType
}

func newClassDefnMembers() classDefnMembers {
	return classDefnMembers{Methods: map[string]*ast.BLangFunction{}}
}

func (n *nodeBuilder) collectClassDefnMembers(memberNodes st.NodeList[st.Node]) classDefnMembers {
	members := newClassDefnMembers()
	for i := 0; i < memberNodes.Size(); i++ {
		member := memberNodes.Get(i)
		switch member.Kind() {
		case st.OBJECT_FIELD:
			field := n.transformClassField(member.(*st.ObjectFieldNode))
			members.Fields = append(members.Fields, field)
		case st.FUNCTION_DEFINITION, st.OBJECT_METHOD_DEFINITION:
			n.addCollectedMethod(&members, member.(*st.FunctionDefinition))
		case st.RESOURCE_ACCESSOR_DEFINITION:
			rm := n.createResourceMethodNode(member.(*st.FunctionDefinition))
			members.ResourceMethods = append(members.ResourceMethods, rm)
		case st.TYPE_REFERENCE:
			typeRef := member.(*st.TypeReferenceNode)
			members.UnresolvedInclusions = append(members.UnresolvedInclusions, n.createTypeNode(typeRef.TypeName()).(*ast.BLangUserDefinedType))
		default:
			panic("collectClassDefnMembers: unsupported member kind")
		}
	}
	return members
}

func (n *nodeBuilder) addCollectedMethod(members *classDefnMembers, funcDef *st.FunctionDefinition) {
	bLFunction := n.createFunctionNode(funcDef.FunctionName(), funcDef.QualifierList(), funcDef.FunctionSignature(), funcDef.FunctionBody())
	bLFunction.SetPosition(n.getPositionWithoutMetadata(funcDef))
	bLFunction.SetAttached()
	n.populateMetadata(funcDef.Metadata(), bLFunction)

	funcName := bLFunction.Name.GetValue()
	if model.Name(funcName) == model.USER_DEFINED_INIT_SUFFIX {
		if members.InitFunction != nil {
			n.cx.SyntaxError("redeclared symbol 'init'", bLFunction.GetPosition())
			return
		}
		members.InitFunction = bLFunction
		return
	}
	if bLFunction.IsRemote() {
		funcName = model.RemoteMethodName(funcName)
		setIdentifierValue(bLFunction.Name, funcName)
	}
	if _, exists := members.Methods[funcName]; exists {
		n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(funcName)+"'", bLFunction.GetPosition())
		return
	}
	members.Methods[funcName] = bLFunction
}

func (n *nodeBuilder) transformAssignmentStatement(assignmentStatementNode *st.AssignmentStatementNode) ast.BLangNode {
	lhsKind := assignmentStatementNode.VarRef().Kind()
	switch lhsKind {
	case st.LIST_BINDING_PATTERN, st.MAPPING_BINDING_PATTERN, st.ERROR_BINDING_PATTERN:
		panic("unimplemented")
	default:
		break
	}

	lhsExpr := ast.NewBLangAssignmentLExpr(n.createExpression(assignmentStatementNode.VarRef()), false)
	return ast.NewBLangAssignment(
		n.getPosition(assignmentStatementNode),
		lhsExpr,
		n.createActionOrExpression(assignmentStatementNode.Expression()),
	)
}

func (n *nodeBuilder) transformCompoundAssignmentStatement(compoundAssignmentStmtNode *st.CompoundAssignmentStatementNode) ast.BLangNode {
	lhsExpr := ast.NewBLangAssignmentLExpr(n.createExpression(compoundAssignmentStmtNode.LhsExpression()), true)
	return ast.NewBLangCompoundAssignment(
		n.getPosition(compoundAssignmentStmtNode),
		lhsExpr,
		n.createActionOrExpression(compoundAssignmentStmtNode.RhsExpression()),
		model.OperatorKindValueFrom(compoundAssignmentStmtNode.BinaryOperator().Text()),
	)
}

func (n *nodeBuilder) transformVariableDeclaration(variableDeclarationNode *st.VariableDeclarationNode) ast.BLangNode {
	varNode := n.createBLangVarDef(
		n.getPosition(variableDeclarationNode),
		variableDeclarationNode.TypedBindingPattern(),
		variableDeclarationNode.Initializer(),
		variableDeclarationNode.FinalKeyword(),
	)
	annotations := variableDeclarationNode.Annotations()
	n.addAnnotationAttachments(annotations, varNode.Var)

	return varNode
}

func (n *nodeBuilder) createBLangVarDef(location diagnostics.Location, typedBindingPattern *st.TypedBindingPatternNode, initializer st.ExpressionNode, finalKeyword st.Token) *ast.BLangVariableDef {
	bindingPattern := typedBindingPattern.BindingPattern()

	variable := n.getBLangVariableNode(bindingPattern, location)

	var qualifiers []st.Token
	if finalKeyword != nil {
		qualifiers = append(qualifiers, finalKeyword) //nolint:staticcheck,ineffassign // qualifierList creation not yet implemented
	}
	// qualifierList := st.CreateNodeListWithFacade(qualifiers)

	switch bindingPattern.Kind() {
	case st.CAPTURE_BINDING_PATTERN, st.WILDCARD_BINDING_PATTERN:
		var expr ast.BLangActionOrExpression
		if initializer != nil {
			expr = n.createActionOrExpression(initializer)
		}
		typeDesc := typedBindingPattern.TypeDescriptor()
		isDeclaredWithVar := isDeclaredWithVar(typeDesc)
		var typeNode ast.BType
		if !isDeclaredWithVar {
			typeNode = n.createTypeNode(typeDesc).(ast.BType)
		}
		var flags model.Flag
		if finalKeyword != nil {
			flags |= model.FlagFinal
		}
		variable = ast.NewBLangVariable(location, variable.Name, typeNode, expr, isDeclaredWithVar, flags)
		variable.SetPosition(location)
		bLVarDef := &ast.BLangVariableDef{Var: variable}
		bLVarDef.SetPosition(location)
		return bLVarDef

	case st.MAPPING_BINDING_PATTERN:
		panic("MAPPING_BINDING_PATTERN unimplemented")

	case st.LIST_BINDING_PATTERN:
		panic("LIST_BINDING_PATTERN unimplemented")

	case st.ERROR_BINDING_PATTERN:
		panic("ERROR_BINDING_PATTERN unimplemented")

	default:
		panic("Syntax kind is not a valid binding pattern")
	}
}

func (n *nodeBuilder) transformBlockStatement(blockStatementNode *st.BlockStatementNode) ast.BLangNode {
	bLBlockStmt := ast.BLangBlockStmt{}
	bLBlockStmt.Stmts = n.generateBLangStatements(blockStatementNode.Statements(), blockStatementNode)
	bLBlockStmt.SetPosition(n.getPosition(blockStatementNode))
	return &bLBlockStmt
}

func (n *nodeBuilder) generateBLangStatements(statementNodes st.NodeList[st.StatementNode], endNode st.Node) []ast.StatementNode {
	statements := []ast.StatementNode{}
	return *n.generateAndAddBLangStatements(statementNodes, &statements, 0, endNode)
}

func (n *nodeBuilder) transformStatement(statement st.StatementNode) ast.StatementNode {
	result, err := n.transformStatementInner(statement)
	if err == nil {
		return result
	}
	if n.mode == nodeBuilderModeRecover {
		return n.badStmt(statement)
	}
	panic(err)
}

func (n *nodeBuilder) transformStatementInner(statement st.StatementNode) (ast.StatementNode, error) {
	if statement == nil {
		return nil, fmt.Errorf("statement is nil")
	}
	// TODO: Ideally we should have a switch that handles all possible stmt nodes instead.
	transformedNode := n.transformSyntaxNode(statement)
	stmt, ok := transformedNode.(ast.StatementNode)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-statement node %T", statement, transformedNode)
	}
	return stmt, nil
}

func (n *nodeBuilder) generateAndAddBLangStatements(statementNodes st.NodeList[st.StatementNode], statements *[]ast.StatementNode, startPosition int, endNode st.Node) *[]ast.StatementNode {
	lastStmtIndex := statementNodes.Size() - 1
	for j := startPosition; j < statementNodes.Size(); j++ {
		currentStatement := statementNodes.Get(j)
		// TODO: Remove this check once statements are non null guaranteed
		if currentStatement == nil {
			continue
		}
		if currentStatement.HasDiagnostics() && n.mode != nodeBuilderModeRecover {
			continue
		}
		if currentStatement.Kind() == st.FORK_STATEMENT {
			forkStmt := currentStatement.(*st.ForkStatementNode)
			n.generateForkStatements(statements, forkStmt)
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

func (n *nodeBuilder) transformBreakStatement(breakStatementNode *st.BreakStatementNode) ast.BLangNode {
	bLBreak := &ast.BLangBreak{}
	bLBreak.SetPosition(n.getPosition(breakStatementNode))
	return bLBreak
}

func (n *nodeBuilder) transformFailStatement(failStatementNode *st.FailStatementNode) ast.BLangNode {
	panic("transformFailStatement unimplemented")
}

func (n *nodeBuilder) transformExpressionStatement(expressionStatement *st.ExpressionStatementNode) ast.BLangNode {
	bLExpressionStmt := ast.BLangExpressionStmt{}
	bLExpressionStmt.Expr = n.createActionOrExpression(expressionStatement.Expression())
	bLExpressionStmt.SetPosition(n.getPosition(expressionStatement))
	return &bLExpressionStmt
}

// createSpecificFieldNameLiteral builds a string-literal expression for a
// non-computed mapping-constructor key. The field name is a static identifier
// or string literal, not a runtime expression, so it must not be represented
// as a var-ref.
func (n *nodeBuilder) createSpecificFieldNameLiteral(fieldName st.Node) ast.BLangExpression {
	if basicLit, ok := fieldName.(*st.BasicLiteralNode); ok {
		return n.createSimpleLiteral(basicLit).(ast.BLangExpression)
	}
	nameRef := n.createBLangNameReference(fieldName)
	name := nameRef[1].GetValue()
	return ast.NewBLangLiteral(n.getPosition(fieldName), ast.LiteralKindString, name, name, false)
}

func (n *nodeBuilder) createExpression(expressionNode st.Node) ast.BLangExpression {
	result, err := n.createExpressionInner(expressionNode)
	if err == nil {
		return result
	}
	if n.mode == nodeBuilderModeRecover {
		return n.badExprOrAction(expressionNode)
	}
	panic(err)
}

func (n *nodeBuilder) createExpressionInner(expressionNode st.Node) (ast.BLangExpression, error) {
	actionOrExpr, err := n.createActionOrExpressionInner(expressionNode)
	if err != nil {
		return nil, err
	}
	expr, ok := actionOrExpr.(ast.BLangExpression)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-expression node %T", expressionNode, actionOrExpr)
	}
	return expr, nil
}

// createActionOrExpression creates an action or expression node from a syntax tree node
func (n *nodeBuilder) createActionOrExpression(actionOrExpression st.Node) ast.BLangActionOrExpression {
	result, err := n.createActionOrExpressionInner(actionOrExpression)
	if err == nil {
		return result
	}
	if n.mode == nodeBuilderModeRecover {
		return n.badExprOrAction(actionOrExpression)
	}
	panic(err)
}

func (n *nodeBuilder) createActionOrExpressionInner(actionOrExpression st.Node) (ast.BLangActionOrExpression, error) {
	if actionOrExpression == nil {
		return nil, fmt.Errorf("missing action or expression")
	}
	if isSimpleLiteral(actionOrExpression.Kind()) {
		result, ok := n.createSimpleLiteral(actionOrExpression).(ast.BLangActionOrExpression)
		if !ok {
			return nil, fmt.Errorf("syntax node %T transformed to non-action-or-expression node", actionOrExpression)
		}
		return result, nil
	}
	if actionOrExpression.Kind() == st.SIMPLE_NAME_REFERENCE ||
		actionOrExpression.Kind() == st.QUALIFIED_NAME_REFERENCE ||
		actionOrExpression.Kind() == st.IDENTIFIER_TOKEN {
		nameReference := n.createBLangNameReference(actionOrExpression)
		bLVarRef := ast.BLangVarRef{}
		bLVarRef.SetPosition(n.getPosition(actionOrExpression))
		bLVarRef.PkgAlias = nameReference[0]
		bLVarRef.VariableName = nameReference[1]
		return &bLVarRef, nil
	}
	if actionOrExpression.Kind() == st.BRACED_EXPRESSION {
		group := ast.BLangGroupExpr{}
		expr, ok := n.transformSyntaxNode(actionOrExpression).(ast.BLangExpression)
		if !ok {
			return nil, fmt.Errorf("braced syntax node %T transformed to non-expression node", actionOrExpression)
		}
		group.Expression = expr
		group.SetPosition(n.getPosition(actionOrExpression))
		return &group, nil
	}
	if isType(actionOrExpression.Kind()) {
		return ast.NewBLangTypedescExpr(
			n.getPosition(actionOrExpression),
			n.createTypeNode(actionOrExpression),
		), nil
	}
	transformedNode := n.transformSyntaxNode(actionOrExpression)
	result, ok := transformedNode.(ast.BLangActionOrExpression)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-action-or-expression node %T", actionOrExpression, transformedNode)
	}
	return result, nil
}

func (n *nodeBuilder) transformContinueStatement(continueStatementNode *st.ContinueStatementNode) ast.BLangNode {
	blContinue := &ast.BLangContinue{}
	blContinue.SetPosition(n.getPosition(continueStatementNode))
	return blContinue
}

func (n *nodeBuilder) transformExternalFunctionBody(externalFunctionBodyNode *st.ExternalFunctionBodyNode) ast.BLangNode {
	body := &ast.BLangExternFunctionBody{}
	body.SetPosition(n.getPosition(externalFunctionBodyNode))
	return body
}

func (n *nodeBuilder) transformIfElseStatement(ifElseStatementNode *st.IfElseStatementNode) ast.BLangNode {
	var elseStmt ast.StatementNode
	if ifElseStatementNode.ElseBody() != nil {
		elseNode := ifElseStatementNode.ElseBody().(*st.ElseBlockNode)
		elseStmt = n.transformSyntaxNode(elseNode.ElseBody()).(ast.StatementNode)
	}
	return ast.NewBLangIf(
		n.getPosition(ifElseStatementNode),
		n.createExpression(ifElseStatementNode.Condition()),
		n.transformBlockStatement(ifElseStatementNode.IfBody()).(*ast.BLangBlockStmt),
		elseStmt,
	)
}

func (n *nodeBuilder) transformElseBlock(elseBlockNode *st.ElseBlockNode) ast.BLangNode {
	panic("transformElseBlock unimplemented")
}

func (n *nodeBuilder) transformWhileStatement(whileStatementNode *st.WhileStatementNode) ast.BLangNode {
	body := n.transformBlockStatement(whileStatementNode.WhileBody()).(*ast.BLangBlockStmt)
	body.SetPosition(n.getPosition(whileStatementNode.WhileBody()))
	var onFailClause *ast.BLangOnFailClause
	if whileStatementNode.OnFailClause() != nil {
		onFailClause = n.transformOnFailClause(whileStatementNode.OnFailClause()).(*ast.BLangOnFailClause)
	}
	return ast.NewBLangWhile(
		n.getPosition(whileStatementNode),
		n.createExpression(whileStatementNode.Condition()),
		body,
		onFailClause,
	)
}

func (n *nodeBuilder) transformPanicStatement(panicStatementNode *st.PanicStatementNode) ast.BLangNode {
	bLPanic := &ast.BLangPanic{}
	bLPanic.SetPosition(n.getPosition(panicStatementNode))
	bLPanic.Expr = n.createExpression(panicStatementNode.Expression())
	return bLPanic
}

func (n *nodeBuilder) transformReturnStatement(returnStatementNode *st.ReturnStatementNode) ast.BLangNode {
	pos := n.getPosition(returnStatementNode)
	var expr ast.BLangActionOrExpression
	if returnStatementNode.Expression() != nil {
		expr = n.createActionOrExpression(returnStatementNode.Expression())
	} else {
		expr = ast.NewBLangLiteral(pos, ast.LiteralKindNil, nil, "", false)
	}
	return ast.NewBLangReturn(pos, expr)
}

func (n *nodeBuilder) transformLocalTypeDefinitionStatement(localTypeDefinitionStatementNode *st.LocalTypeDefinitionStatementNode) ast.BLangNode {
	panic("transformLocalTypeDefinitionStatement unimplemented")
}

func (n *nodeBuilder) transformLockStatement(lockStatementNode *st.LockStatementNode) ast.BLangNode {
	if lockStatementNode.OnFailClause() != nil {
		n.cx.Unimplemented("on-fail clause on lock is not yet supported", n.getPosition(lockStatementNode.OnFailClause()))
	}
	bLLock := &ast.BLangLock{}
	bLLock.SetPosition(n.getPosition(lockStatementNode))
	bLBlockStmt := n.transformBlockStatement(lockStatementNode.BlockStatement()).(*ast.BLangBlockStmt)
	bLBlockStmt.SetPosition(n.getPosition(lockStatementNode.BlockStatement()))
	bLLock.Body = *bLBlockStmt
	return bLLock
}

func (n *nodeBuilder) transformForkStatement(forkStatementNode *st.ForkStatementNode) ast.BLangNode {
	panic("transformForkStatement unimplemented")
}

func (n *nodeBuilder) transformForEachStatement(forEachStatementNode *st.ForEachStatementNode) ast.BLangNode {
	varDef := n.createBLangVarDef(
		n.getPosition(forEachStatementNode.TypedBindingPattern()),
		forEachStatementNode.TypedBindingPattern(),
		nil,
		nil,
	)
	body := n.transformBlockStatement(forEachStatementNode.BlockStatement()).(*ast.BLangBlockStmt)
	body.SetPosition(n.getPosition(forEachStatementNode.BlockStatement()))
	var onFailClause *ast.BLangOnFailClause
	if forEachStatementNode.OnFailClause() != nil {
		onFailClause = n.transformOnFailClause(forEachStatementNode.OnFailClause()).(*ast.BLangOnFailClause)
	}
	return ast.NewBLangForeach(
		n.getPosition(forEachStatementNode),
		varDef,
		n.createExpression(forEachStatementNode.ActionOrExpressionNode()),
		body,
		onFailClause,
	)
}

func (n *nodeBuilder) transformBinaryExpression(binaryBLangExpression *st.BinaryExpressionNode) ast.BLangNode {
	if binaryBLangExpression.Operator().Kind() == st.ELVIS_TOKEN {
		panic("transformBinaryExpression: elvis operator not supported")
	}

	bLBinaryExpr := ast.BLangBinaryExpr{}
	bLBinaryExpr.SetPosition(n.getPosition(binaryBLangExpression))
	bLBinaryExpr.LhsExpr = n.createExpression(binaryBLangExpression.LhsExpr())
	bLBinaryExpr.RhsExpr = n.createExpression(binaryBLangExpression.RhsExpr())
	if binaryBLangExpression.Operator() == nil {
		n.cx.InternalError("binary expression is missing an operator token", bLBinaryExpr.GetPosition())
		return &bLBinaryExpr
	}
	bLBinaryExpr.OpKind = model.OperatorKindValueFrom(binaryBLangExpression.Operator().Text())
	return &bLBinaryExpr
}

func (n *nodeBuilder) transformBracedExpression(bracedBLangExpression *st.BracedExpressionNode) ast.BLangNode {
	return n.createExpression(bracedBLangExpression.Expression())
}

func (n *nodeBuilder) transformCheckExpression(checkBLangExpression *st.CheckExpressionNode) ast.BLangNode {
	pos := n.getPosition(checkBLangExpression)
	// we are deviating from the spec here (https://ballerina.io/spec/lang/master/#section_6.33) check is only suppose
	// to work with expression but jBallerina also allow remote method calls (which is an action)
	expr := n.createActionOrExpression(checkBLangExpression.Expression())
	if checkBLangExpression.CheckKeyword().Kind() == st.CHECK_KEYWORD {
		checkedExpr := &ast.BLangCheckedExpr{}
		checkedExpr.SetPosition(pos)
		checkedExpr.Expr = expr
		return checkedExpr
	}
	checkPanickedExpr := &ast.BLangCheckPanickedExpr{}
	checkPanickedExpr.SetPosition(pos)
	checkPanickedExpr.Expr = expr
	return checkPanickedExpr
}

func (n *nodeBuilder) transformFieldAccessExpression(fieldAccessBLangExpression *st.FieldAccessExpressionNode) ast.BLangNode {
	fieldName := fieldAccessBLangExpression.FieldName()
	if fieldName.Kind() == st.QUALIFIED_NAME_REFERENCE {
		panic("transformFieldAccessExpression: QUALIFIED_NAME_REFERENCE unsupported")
	}

	bLFieldBasedAccess := &ast.BLangFieldBaseAccess{}
	simpleNameRef := fieldName.(*st.SimpleNameReferenceNode)
	bLFieldBasedAccess.Field = n.createIdentifierNodeFromToken(n.getPosition(fieldAccessBLangExpression.FieldName()), simpleNameRef.Name())

	containerExpr := fieldAccessBLangExpression.Expression()
	if containerExpr.Kind() == st.BRACED_EXPRESSION {
		bracedExpr := containerExpr.(*st.BracedExpressionNode)
		bLFieldBasedAccess.Expr = n.createExpression(bracedExpr.Expression())
	} else {
		bLFieldBasedAccess.Expr = n.createExpression(containerExpr)
	}

	bLFieldBasedAccess.SetPosition(n.getPosition(fieldAccessBLangExpression))
	return bLFieldBasedAccess
}

func (n *nodeBuilder) transformFunctionCallExpression(functionCallBLangExpression *st.FunctionCallExpressionNode) ast.BLangNode {
	return n.createBLangInvocation(
		functionCallBLangExpression.FunctionName(),
		functionCallBLangExpression.Arguments(),
		n.getPosition(functionCallBLangExpression),
		n.isFunctionCallAsync(functionCallBLangExpression))
}

func (n *nodeBuilder) transformMethodCallExpression(methodCallBLangExpression *st.MethodCallExpressionNode) ast.BLangNode {
	bLInvocation := n.createBLangInvocation(methodCallBLangExpression.MethodName(),
		methodCallBLangExpression.Arguments(),
		n.getPosition(methodCallBLangExpression), false)
	bLInvocation.Expr = n.createExpression(methodCallBLangExpression.Expression())
	return bLInvocation
}

func (n *nodeBuilder) transformMappingConstructorExpression(mappingConstructorBLangExpression *st.MappingConstructorExpressionNode) ast.BLangNode {
	mappingConstructor := &ast.BLangMappingConstructorExpr{
		Fields: make([]ast.MappingField, 0),
	}
	fields := mappingConstructorBLangExpression.FieldNodes()
	for i := 0; i < fields.Size(); i += 2 {
		field := fields.Get(i)
		switch field.Kind() {
		case st.SPREAD_FIELD:
			panic("mapping constructor spread field not implemented")
		case st.COMPUTED_NAME_FIELD:
			computedNameField := field.(*st.ComputedNameFieldNode)
			keyExpr := n.createExpression(computedNameField.FieldNameExpr())
			key := &ast.BLangMappingKey{
				Expr: keyExpr,
				Kind: ast.MappingKeyComputed,
			}
			key.SetPosition(n.getPosition(computedNameField.FieldNameExpr()))
			keyValueField := &ast.BLangMappingKeyValueField{
				Key:       key,
				ValueExpr: n.createExpression(computedNameField.ValueExpr()),
			}
			keyValueField.SetPosition(n.getPosition(computedNameField))
			mappingConstructor.Fields = append(mappingConstructor.Fields, keyValueField)
		case st.SPECIFIC_FIELD:
			specificField := field.(*st.SpecificFieldNode)
			if specificField.ValueExpr() == nil {
				panic("mapping constructor var-name field not implemented")
			}
			_, isStringLit := specificField.FieldName().(*st.BasicLiteralNode)
			keyKind := ast.MappingKeyIdentifier
			if isStringLit {
				keyKind = ast.MappingKeyStringLiteral
			}
			key := &ast.BLangMappingKey{
				Expr: n.createSpecificFieldNameLiteral(specificField.FieldName()),
				Kind: keyKind,
			}
			key.SetPosition(n.getPosition(specificField.FieldName()))
			keyValueField := &ast.BLangMappingKeyValueField{
				Key:       key,
				ValueExpr: n.createExpression(specificField.ValueExpr()),
				Readonly:  specificField.ReadonlyKeyword() != nil,
			}
			keyValueField.SetPosition(n.getPosition(specificField))
			mappingConstructor.Fields = append(mappingConstructor.Fields, keyValueField)
		default:
			panic(fmt.Sprintf("unexpected mapping field kind: %v", field.Kind()))
		}
	}
	mappingConstructor.SetPosition(n.getPosition(mappingConstructorBLangExpression))
	return mappingConstructor
}

func (n *nodeBuilder) transformIndexedExpression(indexedBLangExpression *st.IndexedExpressionNode) ast.BLangNode {
	indexBasedAccess := &ast.BLangIndexBasedAccess{}
	indexBasedAccess.SetPosition(n.getPosition(indexedBLangExpression))
	keys := indexedBLangExpression.KeyExpression()
	if keys.Size() == 0 {
		panic("missing key expression in member access expression")
	} else if keys.Size() == 1 {
		indexBasedAccess.IndexExpr = n.createExpression(keys.Get(0))
	} else {
		listConstructorExpr := &ast.BLangListConstructorExpr{}
		listConstructorExpr.SetPosition(n.getPositionRange(keys.Get(0), keys.Get(keys.Size()-1)))
		exprs := make([]ast.BLangExpression, 0, keys.Size())
		for i := 0; i < keys.Size(); i++ {
			exprs = append(exprs, n.createExpression(keys.Get(i)))
		}
		listConstructorExpr.Exprs = exprs
		indexBasedAccess.IndexExpr = listConstructorExpr
	}

	indexBasedAccess.Expr = n.createExpression(indexedBLangExpression.ContainerExpression())
	return indexBasedAccess
}

func (n *nodeBuilder) transformTypeofExpression(typeofBLangExpression *st.TypeofExpressionNode) ast.BLangNode {
	panic("transformTypeofExpression unimplemented")
}

func (n *nodeBuilder) transformUnaryExpression(unaryBLangExpression *st.UnaryExpressionNode) ast.BLangNode {
	pos := n.getPosition(unaryBLangExpression)
	operator := model.OperatorKindValueFrom(unaryBLangExpression.UnaryOperator().Text())
	expr := n.createExpression(unaryBLangExpression.Expression())
	if operator == model.OperatorKind_SUB {
		if lit, ok := expr.(*ast.BLangLiteral); ok && foldNegativeIntLiteral(lit) {
			lit.SetPosition(pos)
			return lit
		}
	}
	return ast.NewBLangUnaryExpr(pos, operator, expr)
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

func (n *nodeBuilder) transformComputedNameField(computedNameFieldNode *st.ComputedNameFieldNode) ast.BLangNode {
	panic("transformComputedNameField unimplemented")
}

func (n *nodeBuilder) transformConstantDeclaration(constantDeclarationNode *st.ConstantDeclarationNode) ast.BLangNode {
	nameIdentifier := createIdentifierFromToken(
		n.getPosition(constantDeclarationNode.VariableName()),
		constantDeclarationNode.VariableName(),
	)
	var typeNode ast.BType
	if typeDescriptor := constantDeclarationNode.TypeDescriptor(); typeDescriptor != nil {
		typeNode = n.createTypeNode(typeDescriptor).(ast.BType)
	}
	var flags model.Flag
	if visibility := constantDeclarationNode.VisibilityQualifier(); visibility != nil && visibility.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	constantNode := ast.NewBLangVariable(
		n.getPositionWithoutMetadata(constantDeclarationNode),
		&nameIdentifier,
		typeNode,
		n.createExpression(constantDeclarationNode.Initializer()),
		false,
		flags|model.FlagConstant,
	)
	constantNode.SetPosition(n.getPositionWithoutMetadata(constantDeclarationNode))
	n.populateMetadata(constantDeclarationNode.Metadata(), constantNode)
	return constantNode
}

func (n *nodeBuilder) transformDefaultableParameter(defaultableParameterNode *st.DefaultableParameterNode) ast.BLangNode {
	paramName := defaultableParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarInner(paramName, defaultableParameterNode.TypeName(), defaultableParameterNode.Expression(), nil, defaultableParameterNode.Annotations(), model.FlagDefaultableParam)

	simpleVar.SetPosition(n.getPosition(defaultableParameterNode))

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	return simpleVar
}

func (n *nodeBuilder) createSimpleVarWithTokenNodeNodeList(name st.Token, typeName st.Node, annotations st.NodeList[*st.AnnotationNode], flags model.Flag) *ast.BLangVariable {
	if name != nil {
		return n.createSimpleVarInner(name, typeName, nil, nil, annotations, flags)
	}
	return n.createSimpleVarInner(nil, typeName, nil, nil, annotations, flags)
}

func (n *nodeBuilder) transformRequiredParameter(requiredParameterNode *st.RequiredParameterNode) ast.BLangNode {
	paramName := requiredParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, requiredParameterNode.TypeName(), requiredParameterNode.Annotations(), model.FlagRequiredParam)

	simpleVar.SetPosition(n.getPosition(requiredParameterNode))

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		// Param doesn't have a name and also is not a missing node
		// Therefore, assigning the built-in location
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	return simpleVar
}

func (n *nodeBuilder) transformIncludedRecordParameter(includedRecordParameterNode *st.IncludedRecordParameterNode) ast.BLangNode {
	paramName := includedRecordParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, includedRecordParameterNode.TypeName(), includedRecordParameterNode.Annotations(), model.FlagRequiredParam|model.FlagIncluded)

	simpleVar.SetPosition(n.getPosition(includedRecordParameterNode))

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	return simpleVar
}

func (n *nodeBuilder) transformRestParameter(restParameterNode *st.RestParameterNode) ast.BLangNode {
	paramName := restParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, restParameterNode.TypeName(), restParameterNode.Annotations(), model.FlagRestParam)

	simpleVar.SetPosition(n.getPosition(restParameterNode))

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	return simpleVar
}

func (n *nodeBuilder) transformImportOrgName(importOrgNameNode *st.ImportOrgNameNode) ast.BLangNode {
	panic("transformImportOrgName unimplemented")
}

func (n *nodeBuilder) transformImportPrefix(importPrefixNode *st.ImportPrefixNode) ast.BLangNode {
	panic("transformImportPrefix unimplemented")
}

func (n *nodeBuilder) transformSpecificField(specificFieldNode *st.SpecificFieldNode) ast.BLangNode {
	panic("transformSpecificField unimplemented")
}

func (n *nodeBuilder) transformSpreadField(spreadFieldNode *st.SpreadFieldNode) ast.BLangNode {
	panic("transformSpreadField unimplemented")
}

func (n *nodeBuilder) transformNamedArgument(namedArgumentNode *st.NamedArgumentNode) ast.BLangNode {
	nameToken := namedArgumentNode.ArgumentName().Name()
	return ast.NewBLangNamedArgsExpression(
		n.getPosition(namedArgumentNode),
		n.createIdentifierNodeFromToken(n.getPosition(nameToken), nameToken),
		n.createExpression(namedArgumentNode.Expression()),
	)
}

func (n *nodeBuilder) transformPositionalArgument(positionalArgumentNode *st.PositionalArgumentNode) ast.BLangNode {
	return n.createExpression(positionalArgumentNode.Expression())
}

func (n *nodeBuilder) transformRestArgument(restArgumentNode *st.RestArgumentNode) ast.BLangNode {
	panic("transformRestArgument unimplemented")
}

func (n *nodeBuilder) transformInferredTypedescDefault(inferredTypedescDefaultNode *st.InferredTypedescDefaultNode) ast.BLangNode {
	node := &ast.BLangInferredTypedescDefault{}
	node.SetPosition(n.getPosition(inferredTypedescDefaultNode))
	return node
}

func (n *nodeBuilder) transformObjectTypeDescriptor(objectTypeDescriptorNode *st.ObjectTypeDescriptorNode) ast.BLangNode {
	objectType := ast.NewBLangObjectType(n.getPosition(objectTypeDescriptorNode))

	// Process object type qualifiers (client/service/isolated)
	qualifiers := objectTypeDescriptorNode.ObjectTypeQualifiers()
	for q := range qualifiers.Iterator() {
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
		}
	}

	// Process members
	members := objectTypeDescriptorNode.Members()
	for i := 0; i < members.Size(); i++ {
		member := members.Get(i)
		switch member.Kind() {
		case st.OBJECT_FIELD:
			objectField := member.(*st.ObjectFieldNode)
			fieldName, _ := normalizedIdentifierValue(objectField.FieldName().Text())
			vis := objectField.VisibilityQualifier()
			bField := ast.NewBObjectField(
				n.getPosition(objectField),
				fieldName,
				n.createTypeNode(objectField.TypeName()).(ast.BType),
				vis != nil && vis.Kind() == st.PUBLIC_KEYWORD,
			)
			n.populateMetadata(objectField.Metadata(), bField)
			if objectType.AddMember(bField) {
				n.cx.SyntaxError("redeclared symbol '"+fieldName+"'", bField.GetPosition())
			}
		case st.METHOD_DECLARATION:
			methodDecl := member.(*st.MethodDeclarationNode)
			methodName, _ := normalizedIdentifierValue(methodDecl.MethodName().Text())
			methodKind := ast.ObjectMemberKindMethod
			isPublic := false
			isIsolated := false
			isTransactional := false
			methodQualifiers := methodDecl.QualifierList()
			for q := range methodQualifiers.Iterator() {
				switch q.Kind() {
				case st.PUBLIC_KEYWORD:
					isPublic = true
				case st.REMOTE_KEYWORD:
					methodKind = ast.ObjectMemberKindRemoteMethod
				case st.RESOURCE_KEYWORD:
					methodKind = ast.ObjectMemberKindResourceMethod
				case st.ISOLATED_KEYWORD:
					isIsolated = true
				case st.TRANSACTIONAL_KEYWORD:
					isTransactional = true
				}
			}
			if methodKind == ast.ObjectMemberKindRemoteMethod {
				methodName = model.RemoteMethodName(methodName)
			}

			var functionFlags model.Flag
			if isIsolated {
				functionFlags |= model.FlagIsolated
			}
			if isTransactional {
				functionFlags |= model.FlagTransactional
			}
			funcSig := methodDecl.MethodSignature()
			if funcSig != nil {
				if retTypeDesc := funcSig.ReturnTypeDesc(); retTypeDesc != nil {
					returnsKeyword := retTypeDesc.ReturnsKeyword()
					if returnsKeyword != nil && !returnsKeyword.IsMissing() {
						functionFlags |= model.FlagExplicitReturnTypeDescriptor
					}
				}
			}
			bMethod := ast.NewBMethodDecl(n.getPosition(methodDecl), methodName, methodKind, isPublic, functionFlags)

			// Build function type from method signature
			if funcSig != nil {
				bMethod.ParamListPos = diagnostics.NewBuiltinLocation()
				openParen := funcSig.OpenParenToken()
				closeParen := funcSig.CloseParenToken()
				if openParen != nil && closeParen != nil && !openParen.IsMissing() && !closeParen.IsMissing() {
					bMethod.ParamListPos = n.getPositionRange(openParen, closeParen)
				}

				// Process parameters
				params := funcSig.Parameters()
				for param := range params.Iterator() {
					ftParam := n.createFunctionTypeParam(param)
					if _, isRest := param.(*st.RestParameterNode); isRest {
						bMethod.RestParam = ftParam
					} else {
						bMethod.RequiredParams = append(bMethod.RequiredParams, ftParam)
					}
				}

				// Process return type
				if retTypeDesc := funcSig.ReturnTypeDesc(); retTypeDesc != nil {
					bMethod.ReturnTypeDescriptor = n.createTypeNode(retTypeDesc.Type()).(ast.BType)
				} else {
					nilRet := &ast.BLangValueType{TypeKind: ast.TypeKindNil}
					nilRet.SetPosition(diagnostics.NewBuiltinLocation())
					bMethod.ReturnTypeDescriptor = nilRet
				}
			}

			if objectType.AddMember(bMethod) {
				n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(bMethod.Name())+"'", bMethod.GetPosition())
			}
		case st.TYPE_REFERENCE:
			typeRef := member.(*st.TypeReferenceNode)
			objectType.AddUnresolvedInclusion(n.createTypeNode(typeRef.TypeName()).(*ast.BLangUserDefinedType))
		default:
			panic("unexpected member kind in object type descriptor")
		}
	}

	return objectType
}

func (n *nodeBuilder) transformObjectConstructorExpression(objectConstructorBLangExpression *st.ObjectConstructorExpressionNode) ast.BLangNode {
	panic("transformObjectConstructorExpression unimplemented")
}

func (n *nodeBuilder) transformRecordTypeDescriptor(recordTypeDescriptorNode *st.RecordTypeDescriptorNode) ast.BLangNode {
	recordType := &ast.BLangRecordType{}
	fields := recordTypeDescriptorNode.Fields()
	for i := 0; i < fields.Size(); i++ {
		field := fields.Get(i)
		switch field.Kind() {
		case st.RECORD_FIELD:
			recordField := field.(*st.RecordFieldNode)
			fieldName, _ := normalizedIdentifierValue(recordField.FieldName().Text())
			var flags model.Flag
			if recordField.ReadonlyKeyword() != nil {
				flags |= model.FlagReadonly
			}
			if recordField.QuestionMarkToken() != nil {
				flags |= model.FlagOptional
			}
			bField := ast.NewBField(
				n.getPosition(recordField),
				model.Name(fieldName),
				n.createTypeNode(recordField.TypeName()).(ast.BType),
				nil,
				flags,
			)
			n.populateMetadata(recordField.Metadata(), &bField)
			recordType.AddField(fieldName, bField)
		case st.RECORD_FIELD_WITH_DEFAULT_VALUE:
			recordFieldDV := field.(*st.RecordFieldWithDefaultValueNode)
			fieldName, _ := normalizedIdentifierValue(recordFieldDV.FieldName().Text())
			var flags model.Flag
			if recordFieldDV.ReadonlyKeyword() != nil {
				flags |= model.FlagReadonly
			}
			bField := ast.NewBField(
				n.getPosition(recordFieldDV),
				model.Name(fieldName),
				n.createTypeNode(recordFieldDV.TypeName()).(ast.BType),
				n.createExpression(recordFieldDV.Expression()),
				flags,
			)
			n.populateMetadata(recordFieldDV.Metadata(), &bField)
			recordType.AddField(fieldName, bField)
		case st.TYPE_REFERENCE:
			typeRef := field.(*st.TypeReferenceNode)
			recordType.TypeInclusions = append(recordType.TypeInclusions, n.createTypeNode(typeRef.TypeName()).(ast.BType))
		default:
			panic("unexpected field kind in record type descriptor")
		}
	}
	if restDesc := recordTypeDescriptorNode.RecordRestDescriptor(); restDesc != nil {
		recordType.RestType = n.createTypeNode(restDesc.TypeName()).(ast.BType)
	}
	recordType.IsOpen = recordTypeDescriptorNode.BodyStartDelimiter().Kind() == st.OPEN_BRACE_TOKEN
	recordType.SetPosition(n.getPosition(recordTypeDescriptorNode))
	return recordType
}

func (n *nodeBuilder) transformReturnTypeDescriptor(returnTypeDescriptorNode *st.ReturnTypeDescriptorNode) ast.BLangNode {
	panic("transformReturnTypeDescriptor unimplemented")
}

func (n *nodeBuilder) transformNilTypeDescriptor(nilTypeDescriptorNode *st.NilTypeDescriptorNode) ast.BLangNode {
	panic("transformNilTypeDescriptor unimplemented")
}

func (n *nodeBuilder) transformOptionalTypeDescriptor(optionalTypeDescriptorNode *st.OptionalTypeDescriptorNode) ast.BLangNode {
	nilType := &ast.BLangValueType{TypeKind: ast.TypeKindNil}
	nilType.SetPosition(n.getPosition(optionalTypeDescriptorNode.QuestionMarkToken()))
	return ast.NewBLangUnionTypeNode(
		n.getPosition(optionalTypeDescriptorNode),
		ast.TypeData{TypeDescriptor: n.createTypeNode(optionalTypeDescriptorNode.TypeDescriptor())},
		ast.TypeData{TypeDescriptor: nilType},
	)
}

func (n *nodeBuilder) transformObjectField(objectFieldNode *st.ObjectFieldNode) ast.BLangNode {
	panic("transformObjectField unimplemented")
}

func (n *nodeBuilder) transformRecordField(recordFieldNode *st.RecordFieldNode) ast.BLangNode {
	panic("transformRecordField unimplemented")
}

func (n *nodeBuilder) transformRecordFieldWithDefaultValue(recordFieldWithDefaultValueNode *st.RecordFieldWithDefaultValueNode) ast.BLangNode {
	panic("transformRecordFieldWithDefaultValue unimplemented")
}

func (n *nodeBuilder) transformRecordRestDescriptor(recordRestDescriptorNode *st.RecordRestDescriptorNode) ast.BLangNode {
	panic("transformRecordRestDescriptor unimplemented")
}

func (n *nodeBuilder) transformTypeReference(typeReferenceNode *st.TypeReferenceNode) ast.BLangNode {
	panic("transformTypeReference unimplemented")
}

func (n *nodeBuilder) transformAnnotation(annotationNode *st.AnnotationNode) ast.BLangNode {
	annotation := &ast.BLangAnnotationAttachment{}
	annotation.SetPosition(n.getPosition(annotationNode))
	nameReference := n.createBLangNameReference(annotationNode.AnnotReference())
	annotation.PkgAlias = nameReference[0]
	annotation.AnnotationName = nameReference[1]
	if value := annotationNode.AnnotValue(); value != nil && !value.IsMissing() {
		annotation.Expr = n.createExpression(value)
		annotation.HasValue = true
	} else {
		annotation.Expr = n.createTrueLiteral(annotation.GetPosition())
	}
	return annotation
}

func (n *nodeBuilder) transformMetadata(metadataNode *st.MetadataNode) ast.BLangNode {
	docString := getDocumentationString(metadataNode)
	if docString == nil || docString.IsMissing() {
		return nil
	}
	return n.createMarkdownDocumentationAttachment(docString)
}

func (n *nodeBuilder) transformModuleVariableDeclaration(moduleVariableDeclarationNode *st.ModuleVariableDeclarationNode) ast.BLangNode {
	typedBindingPattern := moduleVariableDeclarationNode.TypedBindingPattern()
	pos := n.getPositionWithoutMetadata(moduleVariableDeclarationNode)
	nameNode := n.getBLangVariableNode(typedBindingPattern.BindingPattern(), pos)

	typeDesc := typedBindingPattern.TypeDescriptor()
	isDeclaredWithVar := typeDesc != nil && isDeclaredWithVar(typeDesc)
	var typeNode ast.BType
	if typeDesc != nil && !isDeclaredWithVar {
		typeNode = n.createTypeNode(typeDesc).(ast.BType)
	}
	var expr ast.BLangActionOrExpression
	if initializer := moduleVariableDeclarationNode.Initializer(); initializer != nil {
		expr = n.createExpression(initializer)
	}
	flags := n.moduleVariableFlags(moduleVariableDeclarationNode, pos)
	simpleVar := ast.NewBLangVariable(pos, nameNode.Name, typeNode, expr, isDeclaredWithVar, flags)
	simpleVar.SetPosition(pos)

	if simpleVar.IsDeclaredWithVar && simpleVar.TypeNode() == nil && simpleVar.Expr == nil {
		n.cx.SyntaxError("var-declared module variable must have an initializer expression for type inference", pos)
		return simpleVar
	}
	n.populateMetadata(moduleVariableDeclarationNode.Metadata(), simpleVar)
	return simpleVar
}

func (n *nodeBuilder) moduleVariableFlags(node *st.ModuleVariableDeclarationNode, pos diagnostics.Location) model.Flag {
	var flags model.Flag
	if visibility := node.VisibilityQualifier(); visibility != nil && visibility.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	qualifiers := node.Qualifiers()
	for qualifier := range qualifiers.Iterator() {
		switch qualifier.Kind() {
		case st.FINAL_KEYWORD:
			flags |= model.FlagFinal
		case st.ISOLATED_KEYWORD:
			flags |= model.FlagIsolated
		case st.CONFIGURABLE_KEYWORD:
			n.cx.Unimplemented("configurable module variables are not supported yet", pos)
		}
	}
	return flags
}

func (n *nodeBuilder) transformTypeTestExpression(typeTestBLangExpression *st.TypeTestExpressionNode) ast.BLangNode {
	pos := n.getPosition(typeTestBLangExpression)
	typeTestExpr := ast.NewBLangTypeTestExpr(
		pos,
		n.createExpression(typeTestBLangExpression.Expression()),
		ast.TypeData{TypeDescriptor: n.createTypeNode(typeTestBLangExpression.TypeDescriptor())},
		typeTestBLangExpression.IsKeyword().Kind() == st.NOT_IS_KEYWORD,
	)
	return typeTestExpr
}

func (n *nodeBuilder) transformRemoteMethodCallAction(remoteMethodCallActionNode *st.RemoteMethodCallActionNode) ast.BLangNode {
	inv := n.createBLangInvocation(remoteMethodCallActionNode.MethodName(),
		remoteMethodCallActionNode.Arguments(),
		n.getPosition(remoteMethodCallActionNode), false)
	action := ast.NewBLangRemoteMethodCallAction(
		inv,
		n.createExpression(remoteMethodCallActionNode.Expression()),
		n.getPosition(remoteMethodCallActionNode),
	)
	return action
}

func (n *nodeBuilder) transformMapTypeDescriptor(mapTypeDescriptorNode *st.MapTypeDescriptorNode) ast.BLangNode {
	refType := &ast.BLangBuiltInRefTypeNode{
		TypeKind: ast.TypeKindMap,
	}
	refType.SetPosition(n.getPosition(mapTypeDescriptorNode))

	mapTypeParamsNode := mapTypeDescriptorNode.MapTypeParamsNode()
	if mapTypeParamsNode == nil || mapTypeParamsNode.TypeNode() == nil {
		panic("map type requires type parameter")
	}
	constraint := n.createTypeNode(mapTypeParamsNode.TypeNode())

	constrainedType := &ast.BLangConstrainedType{
		Type:       ast.TypeData{TypeDescriptor: refType},
		Constraint: ast.TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(refType.GetPosition())
	return constrainedType
}

func (n *nodeBuilder) transformNilLiteral(nilLiteralNode *st.NilLiteralNode) ast.BLangNode {
	panic("transformNilLiteral unimplemented")
}

func (n *nodeBuilder) transformAnnotationDeclaration(annotationDeclarationNode *st.AnnotationDeclarationNode) ast.BLangNode {
	name := createIdentifierFromToken(n.getPosition(annotationDeclarationNode.AnnotationTag()), annotationDeclarationNode.AnnotationTag())
	var flags model.Flag
	if visibility := annotationDeclarationNode.VisibilityQualifier(); visibility != nil && visibility.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	if constKeyword := annotationDeclarationNode.ConstKeyword(); constKeyword != nil && !constKeyword.IsMissing() {
		flags |= model.FlagConstant
	}
	var typeDescriptor ast.TypeDescriptor
	if typeDesc := annotationDeclarationNode.TypeDescriptor(); typeDesc != nil && !typeDesc.IsMissing() {
		typeDescriptor = n.createTypeNode(typeDesc)
	}
	annotation := ast.NewBLangAnnotation(
		n.getPositionWithoutMetadata(annotationDeclarationNode),
		&name,
		typeDescriptor,
		flags,
	)
	attachPoints := annotationDeclarationNode.AttachPoints()
	for attachPoint := range attachPoints.Iterator() {
		if attachPoint, ok := attachPoint.(*st.AnnotationAttachPointNode); ok {
			annotation.AddAttachPoint(n.createAnnotationAttachPoint(attachPoint))
		}
	}
	n.populateMetadata(annotationDeclarationNode.Metadata(), annotation)
	return annotation
}

func (n *nodeBuilder) transformAnnotationAttachPoint(annotationAttachPointNode *st.AnnotationAttachPointNode) ast.BLangNode {
	n.createAnnotationAttachPoint(annotationAttachPointNode)
	return nil
}

func (n *nodeBuilder) createAnnotationAttachPoint(annotationAttachPointNode *st.AnnotationAttachPointNode) ast.AttachPoint {
	parts := []string{}
	identifiers := annotationAttachPointNode.Identifiers()
	for i := 0; i < identifiers.Size(); i++ {
		parts = append(parts, identifiers.Get(i).Text())
	}
	point, ok := annotationAttachPointFromParts(parts)
	if !ok {
		n.cx.SyntaxError("unknown annotation attach point '"+strings.Join(parts, " ")+"'", n.getPosition(annotationAttachPointNode))
	}
	return ast.AttachPoint{
		Point:  point,
		Source: annotationAttachPointNode.SourceKeyword() != nil,
	}
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
	st.Node
	Namespaceuri() st.ExpressionNode
	NamespacePrefix() *st.IdentifierToken
}

func (n *nodeBuilder) transformXMLNamespaceDeclarationNode(node xmlNamespaceDeclarationNode) ast.BLangNode {
	var namespaceURI ast.BLangExpression
	if uriNode := node.Namespaceuri(); uriNode != nil {
		namespaceURI = n.createExpression(uriNode)
	}
	var prefix *ast.BLangIdentifier
	if prefixTok := node.NamespacePrefix(); prefixTok != nil {
		identifier := createIdentifierFromToken(n.getPosition(prefixTok), prefixTok)
		prefix = &identifier
	}
	return ast.NewBLangXMLNS(n.getPosition(node), namespaceURI, prefix)
}

func (n *nodeBuilder) transformXMLNamespaceDeclaration(xMLNamespaceDeclarationNode *st.XMLNamespaceDeclarationNode) ast.BLangNode {
	return n.transformXMLNamespaceDeclarationNode(xMLNamespaceDeclarationNode)
}

func (n *nodeBuilder) transformModuleXMLNamespaceDeclaration(moduleXMLNamespaceDeclarationNode *st.ModuleXMLNamespaceDeclarationNode) ast.BLangNode {
	return n.transformXMLNamespaceDeclarationNode(moduleXMLNamespaceDeclarationNode)
}

func (n *nodeBuilder) transformFunctionBodyBlock(functionBodyBlockNode *st.FunctionBodyBlockNode) ast.BLangNode {
	bLFuncBody := &ast.BLangBlockFunctionBody{}
	statements := []ast.StatementNode{}
	stmtList := statements
	namedWorkerDeclarator := functionBodyBlockNode.NamedWorkerDeclarator()
	if namedWorkerDeclarator != nil {
		panic("unimplemented")
	}

	n.generateAndAddBLangStatements(functionBodyBlockNode.Statements(), &stmtList, 0, functionBodyBlockNode)

	bLFuncBody.Stmts = stmtList
	bLFuncBody.SetPosition(n.getPosition(functionBodyBlockNode))
	return bLFuncBody
}

func (n *nodeBuilder) generateForkStatements(statements *[]ast.StatementNode, forkStatementNode *st.ForkStatementNode) {
	panic("generateForkStatements unimplemented")
}

func (n *nodeBuilder) transformNamedWorkerDeclaration(namedWorkerDeclarationNode *st.NamedWorkerDeclarationNode) ast.BLangNode {
	panic("transformNamedWorkerDeclaration unimplemented")
}

func (n *nodeBuilder) transformNamedWorkerDeclarator(namedWorkerDeclarator *st.NamedWorkerDeclarator) ast.BLangNode {
	panic("transformNamedWorkerDeclarator unimplemented")
}

func (n *nodeBuilder) transformBasicLiteral(basicLiteralNode *st.BasicLiteralNode) ast.BLangNode {
	panic("transformBasicLiteral unimplemented")
}

func (n *nodeBuilder) transformSimpleNameReference(simpleNameReferenceNode *st.SimpleNameReferenceNode) ast.BLangNode {
	panic("transformSimpleNameReference unimplemented")
}

func (n *nodeBuilder) transformQualifiedNameReference(qualifiedNameReferenceNode *st.QualifiedNameReferenceNode) ast.BLangNode {
	nameReference := n.createBLangNameReference(qualifiedNameReferenceNode)
	bLVarRef := &ast.BLangVarRef{}
	bLVarRef.SetPosition(n.getPosition(qualifiedNameReferenceNode))
	bLVarRef.PkgAlias = nameReference[0]
	bLVarRef.VariableName = nameReference[1]
	return bLVarRef
}

func (n *nodeBuilder) transformBuiltinSimpleNameReference(builtinSimpleNameReferenceNode *st.BuiltinSimpleNameReferenceNode) ast.BLangNode {
	panic("transformBuiltinSimpleNameReference unimplemented")
}

func (n *nodeBuilder) transformTrapExpression(trapBLangExpression *st.TrapExpressionNode) ast.BLangNode {
	pos := n.getPosition(trapBLangExpression)
	expr := n.createExpression(trapBLangExpression.Expression())
	trapExpr := &ast.BLangTrapExpr{}
	trapExpr.SetPosition(pos)
	trapExpr.Expr = expr
	return trapExpr
}

func (n *nodeBuilder) transformListConstructorExpression(listConstructorBLangExpression *st.ListConstructorExpressionNode) ast.BLangNode {
	var exprs []ast.BLangExpression
	var spreadMembers []bool
	expressions := listConstructorBLangExpression.Expressions()
	for i := 0; i < expressions.Size(); i += 2 {
		listMember := expressions.Get(i)
		isSpread := listMember.Kind() == st.SPREAD_MEMBER
		if isSpread {
			listMember = listMember.(*st.SpreadMemberNode).Expression()
		}
		exprs = append(exprs, n.createExpression(listMember))
		spreadMembers = append(spreadMembers, isSpread)
	}
	return ast.NewBLangListConstructorExpr(n.getPosition(listConstructorBLangExpression), exprs, spreadMembers)
}

func (n *nodeBuilder) transformTypeCastExpression(typeCastBLangExpression *st.TypeCastExpressionNode) ast.BLangNode {
	typeConversionNode := &ast.BLangTypeConversionExpr{}
	typeConversionNode.SetPosition(n.getPosition(typeCastBLangExpression))
	typeCastParamNode := typeCastBLangExpression.TypeCastParam()
	if typeCastParamNode != nil && typeCastParamNode.Type() != nil {
		typeConversionNode.TypeDescriptor = n.createTypeNode(typeCastParamNode.Type()).(ast.BType)
	} else {
		panic("type cast param node type is not present")
	}
	typeConversionNode.Expression = n.createExpression(typeCastBLangExpression.Expression())
	annotations := typeCastParamNode.Annotations()
	if annotations.Size() > 0 {
		panic("annotations not yet implemented")
	}
	return typeConversionNode
}

func (n *nodeBuilder) transformTypeCastParam(typeCastParamNode *st.TypeCastParamNode) ast.BLangNode {
	panic("transformTypeCastParam unimplemented")
}

func (n *nodeBuilder) transformUnionTypeDescriptor(unionTypeDescriptorNode *st.UnionTypeDescriptorNode) ast.BLangNode {
	return ast.NewBLangUnionTypeNode(
		n.getPosition(unionTypeDescriptorNode),
		ast.TypeData{TypeDescriptor: n.createTypeNode(unionTypeDescriptorNode.LeftTypeDesc())},
		ast.TypeData{TypeDescriptor: n.createTypeNode(unionTypeDescriptorNode.RightTypeDesc())},
	)
}

func (n *nodeBuilder) transformTableConstructorExpression(tableConstructorBLangExpression *st.TableConstructorExpressionNode) ast.BLangNode {
	panic("transformTableConstructorExpression unimplemented")
}

func (n *nodeBuilder) transformKeySpecifier(keySpecifierNode *st.KeySpecifierNode) ast.BLangNode {
	panic("transformKeySpecifier unimplemented")
}

func (n *nodeBuilder) transformStreamTypeDescriptor(streamTypeDescriptorNode *st.StreamTypeDescriptorNode) ast.BLangNode {
	position := n.getPosition(streamTypeDescriptorNode)
	paramsNode := streamTypeDescriptorNode.StreamTypeParamsNode()
	if paramsNode == nil {
		refType := &ast.BLangBuiltInRefTypeNode{
			TypeKind: ast.TypeKindStream,
		}
		refType.SetPosition(position)
		return refType
	}
	params, ok := paramsNode.(*st.StreamTypeParamsNode)
	if !ok {
		n.cx.InternalError("unexpected stream type params node", position)
		return nil
	}
	valueDesc := params.LeftTypeDescNode()
	completionDesc := params.RightTypeDescNode()
	if valueDesc == nil || completionDesc == nil {
		n.cx.InternalError("stream<...> requires both value and completion type parameters", position)
		return nil
	}
	streamType := ast.NewBLangStreamType(ast.TypeData{TypeDescriptor: n.createTypeNode(valueDesc)}, ast.TypeData{TypeDescriptor: n.createTypeNode(completionDesc)})
	streamType.SetPosition(position)
	return streamType
}

func (n *nodeBuilder) transformStreamTypeParams(streamTypeParamsNode *st.StreamTypeParamsNode) ast.BLangNode {
	panic("transformStreamTypeParams unimplemented")
}

func (n *nodeBuilder) transformLetExpression(letBLangExpression *st.LetExpressionNode) ast.BLangNode {
	panic("transformLetExpression unimplemented")
}

func (n *nodeBuilder) transformLetVariableDeclaration(letVariableDeclarationNode *st.LetVariableDeclarationNode) ast.BLangNode {
	varDef := n.createBLangVarDef(
		n.getPosition(letVariableDeclarationNode),
		letVariableDeclarationNode.TypedBindingPattern(),
		letVariableDeclarationNode.Expression(),
		nil,
	)
	annotations := letVariableDeclarationNode.Annotations()
	if annotations.Size() > 0 {
		panic("annotations not yet supported")
	}
	return varDef
}

func (n *nodeBuilder) transformTemplateExpression(templateBLangExpression *st.TemplateExpressionNode) ast.BLangNode {
	typeToken := templateBLangExpression.Type()
	pos := n.getPosition(templateBLangExpression)
	if typeToken == nil {
		n.cx.Unimplemented("raw templates not supported", pos)
		return nil
	}
	switch typeToken.Text() {
	case "string":
		return n.buildStringTemplateExpr(templateBLangExpression, pos)
	case "xml":
		return n.buildXMLTemplateExpr(templateBLangExpression, pos)
	default:
		n.cx.Unimplemented("unsupported template expression kind", pos)
		return nil
	}
}

func (n *nodeBuilder) buildXMLTemplateExpr(templateBLangExpression *st.TemplateExpressionNode, pos diagnostics.Location) ast.BLangNode {
	if !xmlTemplateHasInterpolation(templateBLangExpression.Content()) {
		// If we don't have interpolations we build a literal as an optimization
		return n.buildXMLSequenceLiteral(templateBLangExpression, pos)
	}

	tpl := &ast.BLangXMLTemplateExpr{}
	tpl.SetPosition(pos)
	tpl.Kind = ast.TemplateExprKindXML
	for tok, diag := range n.flattenXMLTemplateContent(templateBLangExpression.Content(), ast.XMLTemplateInsertionKindContent) {
		if diag != nil {
			n.reportXMLTemplateDiagnostic(diag)
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
	return tpl
}

func (n *nodeBuilder) buildXMLSequenceLiteral(templateBLangExpression *st.TemplateExpressionNode, pos diagnostics.Location) ast.BLangNode {
	var children []ast.BLangExpression
	content := templateBLangExpression.Content()
	for child := range content.Iterator() {
		bl := n.transformSyntaxNode(child)
		if bl == nil {
			n.cx.InternalError("xml template child did not produce BLangNode", n.getPosition(child))
			return nil
		}
		expr, ok := bl.(ast.BLangExpression)
		if !ok {
			n.cx.InternalError("xml template child did not produce BLangExpression", n.getPosition(child))
			return nil
		}
		children = append(children, expr)
	}
	if len(children) == 1 {
		return children[0]
	}
	seq := &ast.BLangXMLSequenceLiteral{}
	seq.SetPosition(pos)
	seq.Children = children
	return seq
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
	Internal bool
}

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

func (n *nodeBuilder) flattenXMLTemplateNodeWithNamespace(
	node st.Node,
	kind ast.XMLTemplateInsertionKind,
	namespaceInsertion *ast.XMLTemplateNamespaceInsertion,
	yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool,
) bool {
	switch x := node.(type) {
	case st.Token:
		return yield(newXMLTemplateTextToken(x.Text()), nil)
	case *st.InterpolationNode:
		expr := n.createActionOrExpression(x.Expression())
		be, ok := expr.(ast.BLangExpression)
		if !ok {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation did not produce BLangExpression",
				Position: n.getPosition(x),
				Internal: true,
			})
		}
		return yield(newXMLTemplateInsertionToken(be, kind), nil)
	case *st.XMLTextNode:
		if c := x.Content(); c != nil {
			return yield(newXMLTemplateTextToken(c.Text()), nil)
		}
		return true
	case *st.XMLElementNode:
		return n.flattenXMLTemplateElement(x, namespaceInsertion, yield)
	case *st.XMLEmptyElementNode:
		return n.flattenXMLTemplateEmptyElement(x, namespaceInsertion, yield)
	case *st.XMLComment:
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml comment",
				Position: n.getPosition(ins),
			})
		}
		return yield(newXMLTemplateTextToken(st.ToSourceCode(x.InternalNode())), nil)
	case *st.XMLProcessingInstruction:
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml processing instruction",
				Position: n.getPosition(ins),
			})
		}
		return n.flattenXMLTemplatePI(x, yield)
	case *st.XMLCDATANode:
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml CDATA section",
				Position: n.getPosition(ins),
			})
		}
		return yield(newXMLTemplateTextToken(st.ToSourceCode(x.InternalNode())), nil)
	default:
		return yield(newXMLTemplateTextToken(st.ToSourceCode(node.InternalNode())), nil)
	}
}

func (n *nodeBuilder) flattenXMLTemplateElement(
	x *st.XMLElementNode,
	parentNamespaceInsertion *ast.XMLTemplateNamespaceInsertion,
	yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool,
) bool {
	start := x.StartTag()
	if start == nil {
		return true
	}
	attrs := start.Attributes()
	name := n.xmlNameToString(start.Name())
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
	if !n.flattenXMLTemplateAttributes(attrs, yield) {
		return false
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
	return yield(newXMLTemplateTextToken("</"+name+">"), nil)
}

func (n *nodeBuilder) flattenXMLTemplateEmptyElement(
	x *st.XMLEmptyElementNode,
	parentNamespaceInsertion *ast.XMLTemplateNamespaceInsertion,
	yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool,
) bool {
	name := n.xmlNameToString(x.Name())
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
	return yield(newXMLTemplateTextToken("/>"), nil)
}

func (n *nodeBuilder) flattenXMLTemplatePI(x *st.XMLProcessingInstruction, yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) bool {
	if !yield(newXMLTemplateTextToken("<?"), nil) {
		return false
	}
	if !yield(newXMLTemplateTextToken(n.xmlNameToString(x.Target())), nil) {
		return false
	}
	var dataText strings.Builder
	data := x.Data()
	for child := range data.Iterator() {
		if tok, ok := child.(st.Token); ok {
			dataText.WriteString(tok.Text())
		}
	}
	if data := strings.TrimSpace(dataText.String()); data != "" {
		if !yield(newXMLTemplateTextToken(" "), nil) {
			return false
		}
		if !yield(newXMLTemplateTextToken(data), nil) {
			return false
		}
	}
	return yield(newXMLTemplateTextToken("?>"), nil)
}

func (n *nodeBuilder) flattenXMLTemplateAttributes(attrs st.NodeList[*st.XMLAttributeNode], yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) bool {
	for attr := range attrs.Iterator() {
		name := n.xmlNameToString(attr.AttributeName())
		if !yield(newXMLTemplateTextToken(" "+name+"="), nil) {
			return false
		}
		if value := attr.Value(); value != nil {
			if !n.flattenXMLTemplateAttributeValue(name, value, yield) {
				return false
			}
		}
	}
	return true
}

func (n *nodeBuilder) flattenXMLTemplateAttributeValue(
	name string,
	value *st.XMLAttributeValue,
	yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool,
) bool {
	startQuote := "\""
	if q := value.StartQuote(); q != nil && q.Text() != "" {
		startQuote = q.Text()
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
			if !yield(newXMLTemplateTextToken(tok.Text()), nil) {
				return false
			}
		}
	}
	return yield(newXMLTemplateTextToken(endQuote), nil)
}

func (n *nodeBuilder) reportXMLTemplateDiagnostic(diag *xmlTemplateDiagnostic) {
	if diag.Internal {
		n.cx.InternalError(diag.Message, diag.Position)
		return
	}
	n.cx.SemanticError(diag.Message, diag.Position)
}

func (n *nodeBuilder) collectXMLTemplateNamespaceInsertion(node st.Node) ast.XMLTemplateNamespaceInsertion {
	insn := ast.XMLTemplateNamespaceInsertion{
		UsedPrefixes: map[string]struct{}{},
	}
	n.collectXMLTemplateNamespaceRefs(node, nil, &insn)
	return insn
}

func (n *nodeBuilder) collectXMLTemplateNamespaceRefs(node st.Node, scopes []map[string]struct{}, insn *ast.XMLTemplateNamespaceInsertion) {
	switch x := node.(type) {
	case *st.XMLElementNode:
		start := x.StartTag()
		if start == nil {
			return
		}
		childScopes := appendXMLTemplateNamespaceScope(scopes, n.collectInlineXMLTemplatePrefixes(start.Attributes()))
		n.recordXMLTemplateNameRef(n.xmlNameToString(start.Name()), true, childScopes, insn)
		n.collectXMLTemplateAttributeNamespaceRefs(start.Attributes(), childScopes, insn)
		content := x.Content()
		for child := range content.Iterator() {
			n.collectXMLTemplateNamespaceRefs(child, childScopes, insn)
		}
	case *st.XMLEmptyElementNode:
		childScopes := appendXMLTemplateNamespaceScope(scopes, n.collectInlineXMLTemplatePrefixes(x.Attributes()))
		n.recordXMLTemplateNameRef(n.xmlNameToString(x.Name()), true, childScopes, insn)
		n.collectXMLTemplateAttributeNamespaceRefs(x.Attributes(), childScopes, insn)
	}
}

func (n *nodeBuilder) collectXMLTemplateAttributeNamespaceRefs(
	attrs st.NodeList[*st.XMLAttributeNode],
	scopes []map[string]struct{},
	insn *ast.XMLTemplateNamespaceInsertion,
) {
	for attr := range attrs.Iterator() {
		name := n.xmlNameToString(attr.AttributeName())
		if isXMLTemplateXMLNSName(name) {
			continue
		}
		n.recordXMLTemplateNameRef(name, false, scopes, insn)
	}
}

func (n *nodeBuilder) recordXMLTemplateNameRef(name string, isElement bool, scopes []map[string]struct{}, insn *ast.XMLTemplateNamespaceInsertion) {
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

func (n *nodeBuilder) collectInlineXMLTemplatePrefixes(attrs st.NodeList[*st.XMLAttributeNode]) map[string]struct{} {
	prefixes := map[string]struct{}{}
	for attr := range attrs.Iterator() {
		name := n.xmlNameToString(attr.AttributeName())
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

func (n *nodeBuilder) buildStringTemplateExpr(node *st.TemplateExpressionNode, pos diagnostics.Location) ast.BLangNode {
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
			if c.Kind() != st.TEMPLATE_STRING {
				n.cx.InternalError(fmt.Sprintf("unexpected token kind in string template: %v", c.Kind()), n.getPosition(c))
				continue
			}
			strs = append(strs, c.Text())
			lastStr = true
		case *st.InterpolationNode:
			if !lastStr {
				strs = append(strs, "")
			}
			expr := n.createActionOrExpression(c.Expression())
			be, ok := expr.(ast.BLangExpression)
			if !ok {
				n.cx.InternalError("interpolation did not produce BLangExpression", n.getPosition(c))
				return nil
			}
			insertions = append(insertions, be)
			lastStr = false
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected node in string template: %T", c), n.getPosition(child))
		}
	}
	if !lastStr {
		strs = append(strs, "")
	}
	tpl := &ast.BLangTemplateExpr{Kind: ast.TemplateExprKindString, Strings: strs, Insertions: insertions}
	tpl.SetPosition(pos)
	return tpl
}

func (n *nodeBuilder) xmlNameToString(name st.XMLNameNode) string {
	pos := n.getPosition(name)
	switch name := name.(type) {
	case *st.XMLSimpleNameNode:
		tok := name.Name()
		if tok == nil {
			n.cx.InternalError("xml simple name missing identifier token", pos)
			return ""
		}
		return tok.Text()
	case *st.XMLQualifiedNameNode:
		// TODO: we will a have to revisit this when we support namespaces
		prefixNode := name.Prefix()
		localNode := name.Name()
		if prefixNode == nil || localNode == nil {
			n.cx.InternalError("xml qualified name missing prefix or local part", pos)
			return ""
		}
		prefixTok := prefixNode.Name()
		localTok := localNode.Name()
		if prefixTok == nil || localTok == nil {
			n.cx.InternalError("xml qualified name component missing identifier token", pos)
			return ""
		}
		return prefixTok.Text() + ":" + localTok.Text()
	}
	n.cx.InternalError(fmt.Sprintf("unexpected xml name kind: %T", name), pos)
	return ""
}

func (n *nodeBuilder) xmlAttributes(attrs st.NodeList[*st.XMLAttributeNode]) []ast.BLangXMLAttribute {
	out := make([]ast.BLangXMLAttribute, 0, attrs.Size())
	for attrNode := range attrs.Iterator() {
		attr := n.transformXMLAttribute(attrNode).(*ast.BLangXMLAttribute)
		out = append(out, *attr)
	}
	return out
}

func (n *nodeBuilder) transformXMLElement(xMLElementNode *st.XMLElementNode) ast.BLangNode {
	elem := &ast.BLangXMLElementLiteral{}
	elem.SetPosition(n.getPosition(xMLElementNode))
	if start := xMLElementNode.StartTag(); start != nil {
		elem.Name = n.xmlNameToString(start.Name())
		elem.Attrs = n.xmlAttributes(start.Attributes())
	}
	var children []ast.BLangExpression
	content := xMLElementNode.Content()
	for child := range content.Iterator() {
		bl := n.transformSyntaxNode(child)
		if bl == nil {
			continue
		}
		expr, ok := bl.(ast.BLangExpression)
		if !ok {
			n.cx.InternalError("xml element child did not produce BLangExpression", n.getPosition(child))
			continue
		}
		children = append(children, expr)
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
	return elem
}

func (n *nodeBuilder) transformXMLStartTag(xMLStartTagNode *st.XMLStartTagNode) ast.BLangNode {
	panic("transformXMLStartTag unimplemented")
}

func (n *nodeBuilder) transformXMLEndTag(xMLEndTagNode *st.XMLEndTagNode) ast.BLangNode {
	panic("transformXMLEndTag unimplemented")
}

func (n *nodeBuilder) transformXMLSimpleName(xMLSimpleNameNode *st.XMLSimpleNameNode) ast.BLangNode {
	panic("transformXMLSimpleName unimplemented")
}

func (n *nodeBuilder) transformXMLQualifiedName(xMLQualifiedNameNode *st.XMLQualifiedNameNode) ast.BLangNode {
	panic("transformXMLQualifiedName unimplemented")
}

func (n *nodeBuilder) transformXMLEmptyElement(xMLEmptyElementNode *st.XMLEmptyElementNode) ast.BLangNode {
	elem := &ast.BLangXMLElementLiteral{}
	elem.SetPosition(n.getPosition(xMLEmptyElementNode))
	elem.Name = n.xmlNameToString(xMLEmptyElementNode.Name())
	elem.Attrs = n.xmlAttributes(xMLEmptyElementNode.Attributes())
	return elem
}

func (n *nodeBuilder) transformInterpolation(interpolationNode *st.InterpolationNode) ast.BLangNode {
	n.cx.Unimplemented("xml interpolation not yet supported", n.getPosition(interpolationNode))
	return nil
}

func (n *nodeBuilder) transformXMLText(xMLTextNode *st.XMLTextNode) ast.BLangNode {
	text := &ast.BLangXMLTextLiteral{}
	text.SetPosition(n.getPosition(xMLTextNode))
	if c := xMLTextNode.Content(); c != nil {
		text.Body = c.Text()
	}
	return text
}

func (n *nodeBuilder) transformXMLAttribute(xMLAttributeNode *st.XMLAttributeNode) ast.BLangNode {
	attr := &ast.BLangXMLAttribute{}
	attr.SetPosition(n.getPosition(xMLAttributeNode))
	attr.Name = n.xmlNameToString(xMLAttributeNode.AttributeName())
	if valueNode := xMLAttributeNode.Value(); valueNode != nil {
		if transformed := n.transformXMLAttributeValue(valueNode); transformed != nil {
			if expr, ok := transformed.(ast.BLangExpression); ok {
				attr.Value = expr
			}
		}
	}
	return attr
}

func (n *nodeBuilder) transformXMLAttributeValue(xMLAttributeValue *st.XMLAttributeValue) ast.BLangNode {
	var b strings.Builder
	items := xMLAttributeValue.Value()
	for child := range items.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			n.cx.Unimplemented("xml attribute value interpolation not yet supported", n.getPosition(child))
			return nil
		}
		b.WriteString(tok.Text())
	}
	text := b.String()
	return ast.NewBLangLiteral(n.getPosition(xMLAttributeValue), ast.LiteralKindString, text, text, false)
}

func (n *nodeBuilder) transformXMLComment(xMLComment *st.XMLComment) ast.BLangNode {
	c := &ast.BLangXMLCommentLiteral{}
	c.SetPosition(n.getPosition(xMLComment))
	var b strings.Builder
	content := xMLComment.Content()
	for child := range content.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			n.cx.Unimplemented("xml interpolation in comment not yet supported", n.getPosition(child))
			continue
		}
		b.WriteString(tok.Text())
	}
	c.Body = b.String()
	return c
}

func (n *nodeBuilder) transformXMLCDATA(xMLCDATANode *st.XMLCDATANode) ast.BLangNode {
	n.cx.Unimplemented("xml CDATA not yet supported", n.getPosition(xMLCDATANode))
	return nil
}

func (n *nodeBuilder) transformXMLProcessingInstruction(xMLProcessingInstruction *st.XMLProcessingInstruction) ast.BLangNode {
	pi := &ast.BLangXMLPILiteral{}
	pi.SetPosition(n.getPosition(xMLProcessingInstruction))
	pi.Target = n.xmlNameToString(xMLProcessingInstruction.Target())
	var b strings.Builder
	data := xMLProcessingInstruction.Data()
	for child := range data.Iterator() {
		tok, ok := child.(st.Token)
		if !ok {
			n.cx.Unimplemented("xml interpolation in processing instruction not yet supported", n.getPosition(child))
			continue
		}
		b.WriteString(tok.Text())
	}
	pi.Data = b.String()
	return pi
}

func (n *nodeBuilder) transformTableTypeDescriptor(tableTypeDescriptorNode *st.TableTypeDescriptorNode) ast.BLangNode {
	panic("transformTableTypeDescriptor unimplemented")
}

func (n *nodeBuilder) transformTypeParameter(typeParameterNode *st.TypeParameterNode) ast.BLangNode {
	return n.createTypeNode(typeParameterNode.TypeNode()).(ast.BLangNode)
}

func (n *nodeBuilder) transformKeyTypeConstraint(keyTypeConstraintNode *st.KeyTypeConstraintNode) ast.BLangNode {
	panic("transformKeyTypeConstraint unimplemented")
}

func (n *nodeBuilder) transformFunctionTypeDescriptor(functionTypeDescriptorNode *st.FunctionTypeDescriptorNode) ast.BLangNode {
	var flags model.Flag
	qualifierList := functionTypeDescriptorNode.QualifierList()
	for token := range qualifierList.Iterator() {
		switch token.Kind() {
		case st.ISOLATED_KEYWORD:
			flags |= model.FlagIsolated
		case st.TRANSACTIONAL_KEYWORD:
			flags |= model.FlagTransactional
		}
	}

	var requiredParams []*ast.BLangFunctionTypeParam
	var restParam *ast.BLangFunctionTypeParam
	var returnType ast.BType
	paramListPos := diagnostics.NewBuiltinLocation()
	if funcSignature := functionTypeDescriptorNode.FunctionSignature(); funcSignature != nil {
		openParen := funcSignature.OpenParenToken()
		closeParen := funcSignature.CloseParenToken()
		if openParen != nil && closeParen != nil && !openParen.IsMissing() && !closeParen.IsMissing() {
			paramListPos = n.getPositionRange(openParen, closeParen)
		}
		parameters := funcSignature.Parameters()
		for param := range parameters.Iterator() {
			ftParam := n.createFunctionTypeParam(param)
			if _, isRestParam := param.(*st.RestParameterNode); isRestParam {
				restParam = ftParam
			} else {
				requiredParams = append(requiredParams, ftParam)
			}
		}
		if retNode := funcSignature.ReturnTypeDesc(); retNode != nil {
			if returnsKeyword := retNode.ReturnsKeyword(); returnsKeyword != nil && !returnsKeyword.IsMissing() {
				flags |= model.FlagExplicitReturnTypeDescriptor
			}
			returnType = n.createTypeNode(retNode.Type()).(ast.BType)
		} else {
			returnType = &ast.BLangValueType{TypeKind: ast.TypeKindNil}
			returnType.SetPosition(diagnostics.NewBuiltinLocation())
		}
	} else {
		flags |= model.FlagAnyFunction
	}
	return ast.NewBLangFunctionType(
		n.getPosition(functionTypeDescriptorNode),
		requiredParams,
		restParam,
		returnType,
		paramListPos,
		flags,
	)
}

type typedParameterNode interface {
	st.ParameterNode
	ParamName() st.Token
	TypeName() st.Node
	Annotations() st.NodeList[*st.AnnotationNode]
}

func (n *nodeBuilder) createFunctionTypeParam(param st.ParameterNode) *ast.BLangFunctionTypeParam {
	typedParam, ok := param.(typedParameterNode)
	if !ok {
		panic("createFunctionTypeParam: unsupported parameter type")
	}
	paramName := typedParam.ParamName()
	typeName := typedParam.TypeName()
	annotations := typedParam.Annotations()

	ftParam := ast.BLangFunctionTypeParam{}
	ftParam.SetPosition(n.getPosition(param))

	if paramName != nil {
		name := createIdentifierFromToken(n.getPosition(paramName), paramName)
		name.SetPosition(n.getPosition(paramName))
		ftParam.Name = &name
	}

	ftParam.TypeDesc = n.createTypeNode(typeName).(ast.BType)

	switch p := param.(type) {
	case *st.DefaultableParameterNode:
		defaultExpr := p.Expression()
		ftParam.InitExpr = n.createExpression(defaultExpr)
	case *st.IncludedRecordParameterNode:
		ftParam.SetIncludedRecordParam()
	}

	if annotations.Size() > 0 {
		panic("function type param annotations not yet supported")
	}

	return &ftParam
}

func (n *nodeBuilder) transformFunctionSignature(functionSignatureNode *st.FunctionSignatureNode) ast.BLangNode {
	panic("transformFunctionSignature unimplemented")
}

func (n *nodeBuilder) transformExplicitAnonymousFunctionExpression(anonFuncExprNode *st.ExplicitAnonymousFunctionExpressionNode) ast.BLangNode {
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	data := ast.InvokableData{
		Position: n.getPosition(anonFuncExprNode),
		Name:     &ident,
		Body:     n.transformSyntaxNode(anonFuncExprNode.FunctionBody()).(ast.FunctionBodyNode),
		Flags:    model.FlagLambda | model.FlagAnonymous | functionQualifierFlags(anonFuncExprNode.QualifierList()),
	}
	n.populateFuncSignature(&data, anonFuncExprNode.FunctionSignature())
	bLFunction := ast.NewBLangFunction(data)
	lambdaFunc := &ast.BLangLambdaFunction{Function: bLFunction}
	lambdaFunc.SetPosition(bLFunction.GetPosition())
	return lambdaFunc
}

func (n *nodeBuilder) transformExpressionFunctionBody(expressionFunctionBodyNode *st.ExpressionFunctionBodyNode) ast.BLangNode {
	exprBody := &ast.BLangExprFunctionBody{}
	exprBody.Expr = n.createExpression(expressionFunctionBodyNode.Expression())
	exprBody.SetPosition(n.getPosition(expressionFunctionBodyNode))
	return exprBody
}

func (n *nodeBuilder) transformTupleTypeDescriptor(tupleTypeDescriptorNode *st.TupleTypeDescriptorNode) ast.BLangNode {
	tupleTypeNode := &ast.BLangTupleTypeNode{
		Members: make([]ast.BLangMemberTypeDesc, 0),
	}

	types := tupleTypeDescriptorNode.MemberTypeDesc()
	for i := 0; i < types.Size(); i += 2 {
		node := types.Get(i)
		if node.Kind() == st.REST_TYPE {
			restDescriptor := node.(*st.RestDescriptorNode)
			tupleTypeNode.Rest = n.createTypeNode(restDescriptor.TypeDescriptor()).(ast.BType)
		} else {
			memberNode := node.(*st.MemberTypeDescriptorNode)
			member := ast.BLangMemberTypeDesc{
				TypeDesc: n.createTypeNode(memberNode.TypeDescriptor()),
			}
			member.SetPosition(n.getPosition(memberNode))
			tupleTypeNode.Members = append(tupleTypeNode.Members, member)
		}
	}
	tupleTypeNode.SetPosition(n.getPosition(tupleTypeDescriptorNode))
	return tupleTypeNode
}

func (n *nodeBuilder) transformParenthesisedTypeDescriptor(parenthesisedTypeDescriptorNode *st.ParenthesisedTypeDescriptorNode) ast.BLangNode {
	return n.createTypeNode(parenthesisedTypeDescriptorNode.Typedesc()).(ast.BLangNode)
}

func (n *nodeBuilder) transformExplicitNewExpression(explicitNewBLangExpression *st.ExplicitNewExpressionNode) ast.BLangNode {
	typeInit := &ast.BLangNewExpression{}
	typeInit.SetPosition(n.getPosition(explicitNewBLangExpression))
	typeInit.TypeDescriptor = n.createTypeNode(explicitNewBLangExpression.TypeDescriptor()).(ast.BType)
	if argList := explicitNewBLangExpression.ParenthesizedArgList(); argList != nil {
		args := argList.Arguments()
		for arg := range args.Iterator() {
			typeInit.ArgsExprs = append(typeInit.ArgsExprs, n.createExpression(arg))
		}
	}
	return typeInit
}

func (n *nodeBuilder) transformImplicitNewExpression(implicitNewBLangExpression *st.ImplicitNewExpressionNode) ast.BLangNode {
	typeInit := &ast.BLangNewExpression{}
	typeInit.SetPosition(n.getPosition(implicitNewBLangExpression))
	if argList := implicitNewBLangExpression.ParenthesizedArgList(); argList != nil {
		args := argList.Arguments()
		for arg := range args.Iterator() {
			typeInit.ArgsExprs = append(typeInit.ArgsExprs, n.createExpression(arg))
		}
	}
	return typeInit
}

func (n *nodeBuilder) transformParenthesizedArgList(parenthesizedArgList *st.ParenthesizedArgList) ast.BLangNode {
	panic("transformParenthesizedArgList unimplemented")
}

func (n *nodeBuilder) transformQueryConstructType(queryConstructTypeNode *st.QueryConstructTypeNode) ast.BLangNode {
	keyword := queryConstructTypeNode.Keyword()
	identifier := &ast.BLangIdentifier{Value: keyword.Text()}
	identifier.SetPosition(n.getPosition(queryConstructTypeNode))
	return identifier
}

func (n *nodeBuilder) transformFromClause(fromClauseNode *st.FromClauseNode) ast.BLangNode {
	bindingPatternNode := fromClauseNode.TypedBindingPattern()
	return ast.NewBLangFromClause(
		n.getPosition(fromClauseNode),
		n.createExpression(fromClauseNode.Expression()),
		n.createBLangVarDef(n.getPosition(bindingPatternNode), bindingPatternNode, nil, nil),
		isDeclaredWithVar(bindingPatternNode.TypeDescriptor()),
	)
}

func (n *nodeBuilder) transformWhereClause(whereClauseNode *st.WhereClauseNode) ast.BLangNode {
	whereClause := &ast.BLangWhereClause{}
	whereClause.SetPosition(n.getPosition(whereClauseNode))
	whereClause.Expression = n.createExpression(whereClauseNode.Expression())
	return whereClause
}

func (n *nodeBuilder) transformLetClause(letClauseNode *st.LetClauseNode) ast.BLangNode {
	letClause := &ast.BLangLetClause{}
	letClause.SetPosition(n.getPosition(letClauseNode))
	letVarDeclarations := letClauseNode.LetVarDeclarations()
	letClause.LetVarDeclarations = make([]ast.BLangVariableDef, 0, letVarDeclarations.Size())
	for letVar := range letVarDeclarations.Iterator() {
		varDef := n.transformLetVariableDeclaration(letVar).(*ast.BLangVariableDef)
		letClause.LetVarDeclarations = append(letClause.LetVarDeclarations, *varDef)
	}
	return letClause
}

func (n *nodeBuilder) transformJoinClause(joinClauseNode *st.JoinClauseNode) ast.BLangNode {
	bindingPatternNode := joinClauseNode.TypedBindingPattern()
	var onClause *ast.BLangOnClause
	if onClauseNode := joinClauseNode.JoinOnCondition(); onClauseNode != nil {
		onClause = n.transformOnClause(onClauseNode).(*ast.BLangOnClause)
	}
	return ast.NewBLangJoinClause(
		n.getPosition(joinClauseNode),
		n.createExpression(joinClauseNode.Expression()),
		n.createBLangVarDef(n.getPosition(bindingPatternNode), bindingPatternNode, nil, nil),
		isDeclaredWithVar(bindingPatternNode.TypeDescriptor()),
		joinClauseNode.OuterKeyword() != nil,
		onClause,
	)
}

func (n *nodeBuilder) transformOnClause(onClauseNode *st.OnClauseNode) ast.BLangNode {
	return ast.NewBLangOnClause(
		n.getPosition(onClauseNode),
		n.createExpression(onClauseNode.OnExpression()),
		n.createExpression(onClauseNode.EqualsExpression()),
	)
}

func (n *nodeBuilder) transformLimitClause(limitClauseNode *st.LimitClauseNode) ast.BLangNode {
	return ast.NewBLangLimitClause(n.getPosition(limitClauseNode), n.createExpression(limitClauseNode.Expression()))
}

func (n *nodeBuilder) transformOnConflictClause(onConflictClauseNode *st.OnConflictClauseNode) ast.BLangNode {
	return ast.NewBLangOnConflictClause(n.getPosition(onConflictClauseNode), n.createExpression(onConflictClauseNode.Expression()))
}

func (n *nodeBuilder) transformQueryPipeline(queryPipelineNode *st.QueryPipelineNode) ast.BLangNode {
	panic("transformQueryPipeline unimplemented")
}

func (n *nodeBuilder) transformSelectClause(selectClauseNode *st.SelectClauseNode) ast.BLangNode {
	return ast.NewBLangSelectClause(n.getPosition(selectClauseNode), n.createExpression(selectClauseNode.Expression()))
}

func (n *nodeBuilder) transformCollectClause(collectClauseNode *st.CollectClauseNode) ast.BLangNode {
	return ast.NewBLangCollectClause(
		n.getPosition(collectClauseNode),
		n.createExpression(collectClauseNode.Expression()),
		&balCommon.UnorderedSet[string]{},
	)
}

func (n *nodeBuilder) transformQueryExpression(queryBLangExpression *st.QueryExpressionNode) ast.BLangNode {
	queryExpr := &ast.BLangQueryExpr{}
	queryExpr.SetPosition(n.getPosition(queryBLangExpression))

	if constructType := queryBLangExpression.QueryConstructType(); constructType != nil {
		switch constructType.Keyword().Text() {
		case ast.TypeKindMap.String():
			queryExpr.QueryConstructType = ast.TypeKindMap
		default:
			n.cx.Unimplemented("only map query construct type is supported for now", n.getPosition(constructType))
		}
	}

	queryPipeline := queryBLangExpression.QueryPipeline()
	if queryPipeline == nil || queryPipeline.FromClause() == nil {
		return queryExpr
	}

	fromClause := n.transformSyntaxNode(queryPipeline.FromClause())
	queryExpr.AddQueryClause(fromClause)

	intermediateClauses := queryPipeline.IntermediateClauses()
	for i := 0; i < intermediateClauses.Size(); i++ {
		clause := intermediateClauses.Get(i)
		switch clause.Kind() {
		case st.FROM_CLAUSE, st.JOIN_CLAUSE, st.LET_CLAUSE, st.WHERE_CLAUSE,
			st.GROUP_BY_CLAUSE, st.LIMIT_CLAUSE, st.ORDER_BY_CLAUSE:
			queryExpr.AddQueryClause(n.transformSyntaxNode(clause))
		default:
			n.cx.Unimplemented("only from + join + let + where + group by + order by + limit + select/collect query clauses are supported for now", n.getPosition(clause))
		}
	}

	resultClause := queryBLangExpression.ResultClause()
	if resultClause != nil && (resultClause.Kind() == st.SELECT_CLAUSE || resultClause.Kind() == st.COLLECT_CLAUSE) {
		queryExpr.AddQueryClause(n.transformSyntaxNode(resultClause))
	} else if resultClause != nil {
		n.cx.Unimplemented("only select/collect result clauses are supported for now", n.getPosition(resultClause))
	}

	if queryBLangExpression.OnConflictClause() != nil {
		queryExpr.AddQueryClause(n.transformSyntaxNode(queryBLangExpression.OnConflictClause()))
	}

	return queryExpr
}

func (n *nodeBuilder) transformQueryAction(queryActionNode *st.QueryActionNode) ast.BLangNode {
	panic("transformQueryAction unimplemented")
}

func (n *nodeBuilder) transformIntersectionTypeDescriptor(intersectionTypeDescriptorNode *st.IntersectionTypeDescriptorNode) ast.BLangNode {
	return ast.NewBLangIntersectionTypeNode(
		n.getPosition(intersectionTypeDescriptorNode),
		ast.TypeData{TypeDescriptor: n.createTypeNode(intersectionTypeDescriptorNode.LeftTypeDesc())},
		ast.TypeData{TypeDescriptor: n.createTypeNode(intersectionTypeDescriptorNode.RightTypeDesc())},
	)
}

func (n *nodeBuilder) transformImplicitAnonymousFunctionParameters(implicitAnonymousFunctionParameters *st.ImplicitAnonymousFunctionParameters) ast.BLangNode {
	panic("transformImplicitAnonymousFunctionParameters unimplemented")
}

func (n *nodeBuilder) transformImplicitAnonymousFunctionExpression(node *st.ImplicitAnonymousFunctionExpressionNode) ast.BLangNode {
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	fn := ast.NewBLangFunction(ast.InvokableData{
		Position: n.getPosition(node),
		Name:     &ident,
		Flags:    model.FlagLambda | model.FlagAnonymous,
	})

	var paramNodes []*st.SimpleNameReferenceNode
	switch params := node.Params().(type) {
	case *st.SimpleNameReferenceNode:
		paramNodes = append(paramNodes, params)
	case *st.ImplicitAnonymousFunctionParameters:
		parameters := params.Parameters()
		for param := range parameters.Iterator() {
			paramNodes = append(paramNodes, param)
		}
	default:
		n.cx.SyntaxError("invalid parameter list in inferred anonymous function expression", n.getPosition(node.Params()))
	}
	fn.RequiredParams = make([]ast.BLangVariable, len(paramNodes))
	for i, param := range paramNodes {
		paramName := param.Name()
		paramPos := n.getPosition(paramName)
		ident := createIdentifier(paramPos, nil, nil)
		if paramName != nil && !paramName.IsMissing() {
			paramValue := paramName.Text()
			if paramValue == "_" || paramValue == "'_" {
				n.cx.SyntaxError("'_' cannot be used as an identifier", paramPos)
			}
			ident = createIdentifier(paramPos, &paramValue, &paramValue)
		}
		fn.RequiredParams[i].Name = &ident
		fn.RequiredParams[i].SetPosition(n.getPosition(param))
		fn.RequiredParams[i].SetRequiredParam()
	}
	fn.Body = &ast.BLangExprFunctionBody{
		Expr: n.createExpression(node.Expression()),
	}
	fn.Body.(*ast.BLangExprFunctionBody).SetPosition(n.getPosition(node.Expression()))

	lambda := &ast.BLangLambdaFunction{Function: fn}
	lambda.SetInferredParams()
	lambda.SetPosition(fn.GetPosition())
	return lambda
}

func (n *nodeBuilder) transformStartAction(startActionNode *st.StartActionNode) ast.BLangNode {
	panic("transformStartAction unimplemented")
}

func (n *nodeBuilder) transformFlushAction(flushActionNode *st.FlushActionNode) ast.BLangNode {
	panic("transformFlushAction unimplemented")
}

func (n *nodeBuilder) transformSingletonTypeDescriptor(singletonTypeDescriptorNode *st.SingletonTypeDescriptorNode) ast.BLangNode {
	bLFiniteTypeNode := &ast.BLangFiniteTypeNode{}
	bLFiniteTypeNode.SetPosition(n.getPosition(singletonTypeDescriptorNode))
	bLFiniteTypeNode.ValueSpace = append(bLFiniteTypeNode.ValueSpace, n.createExpression(singletonTypeDescriptorNode.SimpleContExprNode()))
	return bLFiniteTypeNode
}

func (n *nodeBuilder) transformMethodDeclaration(methodDeclarationNode *st.MethodDeclarationNode) ast.BLangNode {
	panic("transformMethodDeclaration unimplemented")
}

func (n *nodeBuilder) transformTypedBindingPattern(typedBindingPatternNode *st.TypedBindingPatternNode) ast.BLangNode {
	panic("transformTypedBindingPattern unimplemented")
}

func (n *nodeBuilder) transformCaptureBindingPattern(captureBindingPatternNode *st.CaptureBindingPatternNode) ast.BLangNode {
	panic("transformCaptureBindingPattern unimplemented")
}

func (n *nodeBuilder) transformWildcardBindingPattern(wildcardBindingPatternNode *st.WildcardBindingPatternNode) ast.BLangNode {
	bLWildCardBindingPattern := &ast.BLangWildCardBindingPattern{}
	bLWildCardBindingPattern.SetPosition(n.getPosition(wildcardBindingPatternNode))
	return bLWildCardBindingPattern
}

func (n *nodeBuilder) transformListBindingPattern(listBindingPatternNode *st.ListBindingPatternNode) ast.BLangNode {
	panic("transformListBindingPattern unimplemented")
}

func (n *nodeBuilder) transformMappingBindingPattern(mappingBindingPatternNode *st.MappingBindingPatternNode) ast.BLangNode {
	panic("transformMappingBindingPattern unimplemented")
}

func (n *nodeBuilder) transformFieldBindingPatternFull(fieldBindingPatternFullNode *st.FieldBindingPatternFullNode) ast.BLangNode {
	panic("transformFieldBindingPatternFull unimplemented")
}

func (n *nodeBuilder) transformFieldBindingPatternVarname(fieldBindingPatternVarnameNode *st.FieldBindingPatternVarnameNode) ast.BLangNode {
	panic("transformFieldBindingPatternVarname unimplemented")
}

func (n *nodeBuilder) transformRestBindingPattern(restBindingPatternNode *st.RestBindingPatternNode) ast.BLangNode {
	panic("transformRestBindingPattern unimplemented")
}

func (n *nodeBuilder) transformErrorBindingPattern(errorBindingPatternNode *st.ErrorBindingPatternNode) ast.BLangNode {
	panic("transformErrorBindingPattern unimplemented")
}

func (n *nodeBuilder) transformNamedArgBindingPattern(namedArgBindingPatternNode *st.NamedArgBindingPatternNode) ast.BLangNode {
	panic("transformNamedArgBindingPattern unimplemented")
}

func (n *nodeBuilder) transformAsyncSendAction(asyncSendActionNode *st.AsyncSendActionNode) ast.BLangNode {
	panic("transformAsyncSendAction unimplemented")
}

func (n *nodeBuilder) transformSyncSendAction(syncSendActionNode *st.SyncSendActionNode) ast.BLangNode {
	panic("transformSyncSendAction unimplemented")
}

func (n *nodeBuilder) transformReceiveAction(receiveActionNode *st.ReceiveActionNode) ast.BLangNode {
	panic("transformReceiveAction unimplemented")
}

func (n *nodeBuilder) transformReceiveFields(receiveFieldsNode *st.ReceiveFieldsNode) ast.BLangNode {
	panic("transformReceiveFields unimplemented")
}

func (n *nodeBuilder) transformAlternateReceive(alternateReceiveNode *st.AlternateReceiveNode) ast.BLangNode {
	panic("transformAlternateReceive unimplemented")
}

func (n *nodeBuilder) transformRestDescriptor(restDescriptorNode *st.RestDescriptorNode) ast.BLangNode {
	panic("transformRestDescriptor unimplemented")
}

func (n *nodeBuilder) transformDoubleGTToken(doubleGTTokenNode *st.DoubleGTTokenNode) ast.BLangNode {
	panic("transformDoubleGTToken unimplemented")
}

func (n *nodeBuilder) transformTrippleGTToken(trippleGTTokenNode *st.TrippleGTTokenNode) ast.BLangNode {
	panic("transformTrippleGTToken unimplemented")
}

func (n *nodeBuilder) transformWaitAction(waitActionNode *st.WaitActionNode) ast.BLangNode {
	panic("transformWaitAction unimplemented")
}

func (n *nodeBuilder) transformWaitFieldsList(waitFieldsListNode *st.WaitFieldsListNode) ast.BLangNode {
	panic("transformWaitFieldsList unimplemented")
}

func (n *nodeBuilder) transformWaitField(waitFieldNode *st.WaitFieldNode) ast.BLangNode {
	panic("transformWaitField unimplemented")
}

func (n *nodeBuilder) transformAnnotAccessExpression(annotAccessBLangExpression *st.AnnotAccessExpressionNode) ast.BLangNode {
	expr := &ast.BLangAnnotAccessExpr{}
	expr.Expr = n.createExpression(annotAccessBLangExpression.Expression())
	nameReference := n.createBLangNameReference(annotAccessBLangExpression.AnnotTagReference())
	expr.PkgAlias = nameReference[0]
	expr.AnnotationName = nameReference[1]
	expr.SetPosition(n.getPosition(annotAccessBLangExpression))
	return expr
}

func (n *nodeBuilder) transformOptionalFieldAccessExpression(optionalFieldAccessBLangExpression *st.OptionalFieldAccessExpressionNode) ast.BLangNode {
	fieldName := optionalFieldAccessBLangExpression.FieldName()
	if fieldName.Kind() == st.QUALIFIED_NAME_REFERENCE {
		panic("transformOptionalFieldAccessExpression: QUALIFIED_NAME_REFERENCE expected")
	}

	containerExpr := optionalFieldAccessBLangExpression.Expression()
	var expr ast.BLangExpression
	if containerExpr.Kind() == st.BRACED_EXPRESSION {
		expr = n.createExpression(containerExpr.(*st.BracedExpressionNode).Expression())
	} else {
		expr = n.createExpression(containerExpr)
	}
	simpleNameRef := fieldName.(*st.SimpleNameReferenceNode)
	return ast.NewBLangFieldBaseAccess(
		n.getPosition(optionalFieldAccessBLangExpression),
		expr,
		n.createIdentifierNodeFromToken(n.getPosition(fieldName), simpleNameRef.Name()),
		true,
	)
}

func (n *nodeBuilder) transformConditionalExpression(conditionalBLangExpression *st.ConditionalExpressionNode) ast.BLangNode {
	panic("transformConditionalExpression unimplemented")
}

func (n *nodeBuilder) transformEnumDeclaration(enumDeclarationNode *st.EnumDeclarationNode) ast.BLangNode {
	publicQualifier := false
	qualifier := enumDeclarationNode.Qualifier()
	if qualifier != nil && qualifier.Kind() == st.PUBLIC_KEYWORD {
		publicQualifier = true
	}

	memberNodes := enumDeclarationNode.EnumMemberList()
	memberTypeNodes := make([]ast.TypeDescriptor, 0)
	for memberNode := range memberNodes.Iterator() {
		if memberNode.Kind() != st.ENUM_MEMBER {
			continue
		}
		enumMember := memberNode.(*st.EnumMemberNode)
		if enumMember.Identifier() == nil || enumMember.Identifier().IsMissing() {
			n.cx.InternalError("missing enum member identifier", n.getPosition(enumMember))
			continue
		}
		constantNode := n.transformEnumMemberWithVisibility(enumMember, publicQualifier)
		if n.currentCompUnit == nil {
			n.cx.InternalError("enum constants can only be added at module level", n.getPosition(enumMember))
			continue
		}
		n.currentCompUnit.AddTopLevelNode(constantNode)
		memberTypeNodes = append(memberTypeNodes, n.createTypeNode(enumMember.Identifier()))
	}

	identifierPos := n.getPosition(enumDeclarationNode.Identifier())
	identifier := createIdentifierFromToken(identifierPos, enumDeclarationNode.Identifier())
	var documentation *ast.BLangMarkdownDocumentation
	if metadata := enumDeclarationNode.Metadata(); metadata != nil && !metadata.IsMissing() {
		documentation = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}
	var flags model.Flag
	if publicQualifier {
		flags |= model.FlagPublic
	}
	typeDef := ast.NewBLangTypeDefinitionWithData(n.getPositionWithoutMetadata(enumDeclarationNode), &identifier, ast.TypeData{}, documentation, flags)
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

	return typeDef
}

func (n *nodeBuilder) transformEnumMember(enumMemberNode *st.EnumMemberNode) ast.BLangNode {
	return n.transformEnumMemberWithVisibility(enumMemberNode, false)
}

func (n *nodeBuilder) transformEnumMemberWithVisibility(enumMemberNode *st.EnumMemberNode, publicQualifier bool) *ast.BLangVariable {
	identifier := createIdentifierFromToken(n.getPosition(enumMemberNode.Identifier()), enumMemberNode.Identifier())
	var expr ast.BLangExpression
	if exprNode := enumMemberNode.ConstExprNode(); exprNode != nil {
		expr = n.createExpression(exprNode)
	} else {
		expr = n.createSimpleLiteral(enumMemberNode.Identifier()).(ast.BLangExpression)
	}
	stringType := &ast.BLangValueType{TypeKind: ast.TypeKindString}
	stringType.SetPosition(diagnostics.NewBuiltinLocation())
	var flags model.Flag
	if publicQualifier {
		flags |= model.FlagPublic
	}
	constantNode := ast.NewBLangVariable(n.getPositionWithoutMetadata(enumMemberNode), &identifier, stringType, expr, false, flags|model.FlagConstant)
	constantNode.SetPosition(n.getPositionWithoutMetadata(enumMemberNode))
	metadata := enumMemberNode.Metadata()
	if metadata != nil && !metadata.IsMissing() {
		constantNode.MarkdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}
	return constantNode
}

func (n *nodeBuilder) transformArrayTypeDescriptor(arrayTypeDescriptorNode *st.ArrayTypeDescriptorNode) ast.BLangNode {
	position := n.getPosition(arrayTypeDescriptorNode)
	dimensionNodes := arrayTypeDescriptorNode.Dimensions()
	dimensionSize := dimensionNodes.Size()
	var sizes []ast.BLangExpression

	for i := 0; i < dimensionSize; i++ {
		dimensionNode := dimensionNodes.Get(i)
		if dimensionNode.ArrayLength() == nil {
			sizes = append(sizes, nil)
		} else {
			sizes = append(sizes, n.createExpression(dimensionNode.ArrayLength()))
		}
	}
	dimensionSize = len(sizes)

	arrayTypeNode := &ast.BLangArrayType{}
	arrayTypeNode.SetPosition(position)
	arrayTypeNode.Elemtype = ast.TypeData{
		TypeDescriptor: n.createTypeNode(arrayTypeDescriptorNode.MemberTypeDesc()),
	}
	arrayTypeNode.Dimensions = dimensionSize
	arrayTypeNode.Sizes = sizes
	return arrayTypeNode
}

func (n *nodeBuilder) transformArrayDimension(arrayDimensionNode *st.ArrayDimensionNode) ast.BLangNode {
	panic("transformArrayDimension unimplemented")
}

func (n *nodeBuilder) transformTransactionStatement(transactionStatementNode *st.TransactionStatementNode) ast.BLangNode {
	panic("transformTransactionStatement unimplemented")
}

func (n *nodeBuilder) transformRollbackStatement(rollbackStatementNode *st.RollbackStatementNode) ast.BLangNode {
	panic("transformRollbackStatement unimplemented")
}

func (n *nodeBuilder) transformRetryStatement(retryStatementNode *st.RetryStatementNode) ast.BLangNode {
	panic("transformRetryStatement unimplemented")
}

func (n *nodeBuilder) transformCommitAction(commitActionNode *st.CommitActionNode) ast.BLangNode {
	panic("transformCommitAction unimplemented")
}

func (n *nodeBuilder) transformTransactionalExpression(transactionalBLangExpression *st.TransactionalExpressionNode) ast.BLangNode {
	panic("transformTransactionalExpression unimplemented")
}

func (n *nodeBuilder) transformByteArrayLiteral(byteArrayLiteralNode *st.ByteArrayLiteralNode) ast.BLangNode {
	panic("transformByteArrayLiteral unimplemented")
}

func (n *nodeBuilder) transformXMLFilterExpression(xMLFilterBLangExpression *st.XMLFilterExpressionNode) ast.BLangNode {
	panic("transformXMLFilterExpression unimplemented")
}

func (n *nodeBuilder) transformXMLStepExpression(xMLStepBLangExpression *st.XMLStepExpressionNode) ast.BLangNode {
	panic("transformXMLStepExpression unimplemented")
}

func (n *nodeBuilder) transformXMLNamePatternChaining(xMLNamePatternChainingNode *st.XMLNamePatternChainingNode) ast.BLangNode {
	panic("transformXMLNamePatternChaining unimplemented")
}

func (n *nodeBuilder) transformXMLStepIndexedExtend(xMLStepIndexedExtendNode *st.XMLStepIndexedExtendNode) ast.BLangNode {
	panic("transformXMLStepIndexedExtend unimplemented")
}

func (n *nodeBuilder) transformXMLStepMethodCallExtend(xMLStepMethodCallExtendNode *st.XMLStepMethodCallExtendNode) ast.BLangNode {
	panic("transformXMLStepMethodCallExtend unimplemented")
}

func (n *nodeBuilder) transformXMLAtomicNamePattern(xMLAtomicNamePatternNode *st.XMLAtomicNamePatternNode) ast.BLangNode {
	panic("transformXMLAtomicNamePattern unimplemented")
}

func (n *nodeBuilder) transformTypeReferenceTypeDesc(typeReferenceTypeDescNode *st.TypeReferenceTypeDescNode) ast.BLangNode {
	panic("transformTypeReferenceTypeDesc unimplemented")
}

func (n *nodeBuilder) transformMatchStatement(matchStatementNode *st.MatchStatementNode) ast.BLangNode {
	matchStatement := &ast.BLangMatchStatement{}
	matchStmtExpr := n.createExpression(matchStatementNode.Condition())
	matchStatement.Expr = matchStmtExpr

	matchClauses := matchStatementNode.MatchClauses()
	for matchClauseNode := range matchClauses.Iterator() {
		bLangMatchClause := &ast.BLangMatchClause{}
		bLangMatchClause.SetPosition(n.getPosition(matchClauseNode))

		// Handle match guard
		if matchClauseNode.MatchGuard() != nil {
			matchGuardNode := matchClauseNode.MatchGuard()
			bLangMatchClause.Guard = n.createExpression(matchGuardNode.Expression())
		}

		// Handle match patterns
		matchPatterns := matchClauseNode.MatchPatterns()
		for matchPattern := range matchPatterns.Iterator() {
			bLangMatchPattern := n.transformMatchPattern(matchPattern)
			if bLangMatchPattern != nil {
				bLangMatchClause.Patterns = append(bLangMatchClause.Patterns, bLangMatchPattern)
			}
		}

		// Handle block statement
		bLangMatchClause.Body = *n.transformBlockStatement(matchClauseNode.BlockStatement()).(*ast.BLangBlockStmt)

		matchStatement.MatchClauses = append(matchStatement.MatchClauses, *bLangMatchClause)
	}

	matchStatement.SetPosition(n.getPosition(matchStatementNode))
	return matchStatement
}

func (n *nodeBuilder) transformMatchPattern(matchPattern st.Node) ast.BLangMatchPattern {
	matchPatternPos := n.getPosition(matchPattern)
	kind := matchPattern.Kind()

	switch kind {
	case st.SIMPLE_NAME_REFERENCE:
		nameRef := matchPattern.(*st.SimpleNameReferenceNode)
		if nameRef.Name().Text() == "_" {
			bLangWildCard := &ast.BLangWildCardMatchPattern{}
			bLangWildCard.SetPosition(matchPatternPos)
			return bLangWildCard
		}
		bLangConstPattern := &ast.BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.SetPosition(matchPatternPos)
		return bLangConstPattern

	case st.IDENTIFIER_TOKEN:
		idToken := matchPattern.(st.Token)
		if idToken.Text() == "_" {
			bLangWildCard := &ast.BLangWildCardMatchPattern{}
			bLangWildCard.SetPosition(matchPatternPos)
			return bLangWildCard
		}
		bLangConstPattern := &ast.BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.SetPosition(matchPatternPos)
		return bLangConstPattern

	case st.NUMERIC_LITERAL,
		st.STRING_LITERAL,
		st.QUALIFIED_NAME_REFERENCE,
		st.NULL_LITERAL,
		st.NIL_LITERAL,
		st.BOOLEAN_LITERAL,
		st.UNARY_EXPRESSION:
		bLangConstPattern := &ast.BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.SetPosition(matchPatternPos)
		return bLangConstPattern

	case st.PIPE_TOKEN, st.COMMA_TOKEN:
		// Skip separator tokens in match pattern lists
		return nil

	default:
		n.cx.InternalError(fmt.Sprintf("unexpected match pattern kind: %v", kind), matchPatternPos)
		return nil
	}
}

func (n *nodeBuilder) transformMatchClause(matchClauseNode *st.MatchClauseNode) ast.BLangNode {
	panic("transformMatchClause unimplemented")
}

func (n *nodeBuilder) transformMatchGuard(matchGuardNode *st.MatchGuardNode) ast.BLangNode {
	panic("transformMatchGuard unimplemented")
}

func (n *nodeBuilder) transformDistinctTypeDescriptor(distinctTypeDescriptorNode *st.DistinctTypeDescriptorNode) ast.BLangNode {
	n.cx.Unimplemented("anonymous distinct types not supported", n.getPosition(distinctTypeDescriptorNode))
	neverType := &ast.BLangValueType{TypeKind: ast.TypeKindNever}
	neverType.SetPosition(n.getPosition(distinctTypeDescriptorNode))
	return neverType
}

func (n *nodeBuilder) transformListMatchPattern(listMatchPatternNode *st.ListMatchPatternNode) ast.BLangNode {
	panic("transformListMatchPattern unimplemented")
}

func (n *nodeBuilder) transformRestMatchPattern(restMatchPatternNode *st.RestMatchPatternNode) ast.BLangNode {
	panic("transformRestMatchPattern unimplemented")
}

func (n *nodeBuilder) transformMappingMatchPattern(mappingMatchPatternNode *st.MappingMatchPatternNode) ast.BLangNode {
	panic("transformMappingMatchPattern unimplemented")
}

func (n *nodeBuilder) transformFieldMatchPattern(fieldMatchPatternNode *st.FieldMatchPatternNode) ast.BLangNode {
	panic("transformFieldMatchPattern unimplemented")
}

func (n *nodeBuilder) transformErrorMatchPattern(errorMatchPatternNode *st.ErrorMatchPatternNode) ast.BLangNode {
	panic("transformErrorMatchPattern unimplemented")
}

func (n *nodeBuilder) transformNamedArgMatchPattern(namedArgMatchPatternNode *st.NamedArgMatchPatternNode) ast.BLangNode {
	panic("transformNamedArgMatchPattern unimplemented")
}

// Helper functions for markdown documentation transformation

func (n *nodeBuilder) addReferencesAndReturnDocumentationText(references *[]ast.BLangMarkdownReferenceDocumentation, docElements st.NodeList[st.Node]) string {
	var docText strings.Builder
	for i := 0; i < docElements.Size(); i++ {
		element := docElements.Get(i)
		if element.Kind() == st.BALLERINA_NAME_REFERENCE {
			bLangRefDoc := &ast.BLangMarkdownReferenceDocumentation{}
			balNameRefNode := element.(*st.BallerinaNameReferenceNode)

			bLangRefDoc.SetPosition(n.getPosition(balNameRefNode))

			startBacktick := balNameRefNode.StartBacktick()
			backtickContent := balNameRefNode.NameReference()
			endBacktick := balNameRefNode.EndBacktick()

			contentString := ""
			if backtickContent != nil && !backtickContent.IsMissing() {
				// Use InternalNode() to get STNode and convert to source code
				contentString = st.ToSourceCode(backtickContent.InternalNode())
			}
			bLangRefDoc.ReferenceName = contentString
			bLangRefDoc.Type = ast.DocumentationReferenceType("BACKTICK_CONTENT")

			referenceType := balNameRefNode.ReferenceType()
			if referenceType != nil && !referenceType.IsMissing() {
				refTypeText := referenceType.Text()
				bLangRefDoc.Type = n.stringToRefType(refTypeText)
				docText.WriteString(refTypeText)
			}

			n.transformDocumentationBacktickContent(backtickContent, bLangRefDoc)

			if startBacktick != nil && !startBacktick.IsMissing() {
				docText.WriteString(startBacktick.Text())
			}
			docText.WriteString(contentString)
			if endBacktick != nil && !endBacktick.IsMissing() {
				docText.WriteString(endBacktick.Text())
			}
			*references = append(*references, *bLangRefDoc)
		} else if element.Kind() == st.DOCUMENTATION_DESCRIPTION {
			if token, ok := element.(st.Token); ok {
				docText.WriteString(token.Text())
			}
		} else if element.Kind() == st.INLINE_CODE_REFERENCE {
			inlineCodeRefNode := element.(*st.InlineCodeReferenceNode)
			if startBacktick := inlineCodeRefNode.StartBacktick(); startBacktick != nil && !startBacktick.IsMissing() {
				docText.WriteString(startBacktick.Text())
			}
			if codeRef := inlineCodeRefNode.CodeReference(); codeRef != nil && !codeRef.IsMissing() {
				docText.WriteString(codeRef.Text())
			}
			if endBacktick := inlineCodeRefNode.EndBacktick(); endBacktick != nil && !endBacktick.IsMissing() {
				docText.WriteString(endBacktick.Text())
			}
		}
	}

	return n.trimLeftAtMostOne(docText.String())
}

func (n *nodeBuilder) transformDocumentationBacktickContent(backtickContent st.Node, bLangRefDoc *ast.BLangMarkdownReferenceDocumentation) {
	switch backtickContent.Kind() {
	case st.CODE_CONTENT:
		// reaching here means ballerina name reference is syntactically invalid.
		// therefore, set hasParserWarnings to true. so that,
		// doc analyzer will avoid further checks on this.
		bLangRefDoc.HasParserWarnings = true
	case st.QUALIFIED_NAME_REFERENCE:
		qualifiedRef := backtickContent.(*st.QualifiedNameReferenceNode)
		modulePrefix := qualifiedRef.ModulePrefix()
		identifier := qualifiedRef.Identifier()
		if modulePrefix != nil && !modulePrefix.IsMissing() {
			bLangRefDoc.Qualifier = modulePrefix.Text()
		}
		if identifier != nil && !identifier.IsMissing() {
			bLangRefDoc.Identifier = identifier.Text()
		}
	case st.SIMPLE_NAME_REFERENCE:
		simpleRef := backtickContent.(*st.SimpleNameReferenceNode)
		name := simpleRef.Name()
		if name != nil && !name.IsMissing() {
			bLangRefDoc.Identifier = name.Text()
		}
	case st.FUNCTION_CALL:
		funcCallExpr := backtickContent.(*st.FunctionCallExpressionNode)
		funcName := funcCallExpr.FunctionName()
		if funcName.Kind() == st.QUALIFIED_NAME_REFERENCE {
			qualifiedRef := funcName.(*st.QualifiedNameReferenceNode)
			modulePrefix := qualifiedRef.ModulePrefix()
			identifier := qualifiedRef.Identifier()
			if modulePrefix != nil && !modulePrefix.IsMissing() {
				bLangRefDoc.Qualifier = modulePrefix.Text()
			}
			if identifier != nil && !identifier.IsMissing() {
				bLangRefDoc.Identifier = identifier.Text()
			}
		} else {
			simpleRef := funcName.(*st.SimpleNameReferenceNode)
			name := simpleRef.Name()
			if name != nil && !name.IsMissing() {
				bLangRefDoc.Identifier = name.Text()
			}
		}
	case st.METHOD_CALL:
		methodCallExprNode := backtickContent.(*st.MethodCallExpressionNode)
		methodName := methodCallExprNode.MethodName()
		if simpleRef, ok := methodName.(*st.SimpleNameReferenceNode); ok {
			name := simpleRef.Name()
			if name != nil && !name.IsMissing() {
				bLangRefDoc.Identifier = name.Text()
			}
		}
		refName := methodCallExprNode.Expression()
		if refName.Kind() == st.QUALIFIED_NAME_REFERENCE {
			qualifiedRef := refName.(*st.QualifiedNameReferenceNode)
			identifier := qualifiedRef.Identifier()
			if identifier != nil && !identifier.IsMissing() {
				bLangRefDoc.TypeName = identifier.Text()
			}
			modulePrefix := qualifiedRef.ModulePrefix()
			if modulePrefix != nil && !modulePrefix.IsMissing() {
				bLangRefDoc.Qualifier = modulePrefix.Text()
			}
		} else if refName.Kind() == st.SIMPLE_NAME_REFERENCE {
			simpleRef := refName.(*st.SimpleNameReferenceNode)
			name := simpleRef.Name()
			if name != nil && !name.IsMissing() {
				bLangRefDoc.TypeName = name.Text()
			}
		}
	default:
		// ignore other cases
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
}

func (n *nodeBuilder) transformCodeBlock(documentationLines *[]ast.BLangMarkdownDocumentationLine, codeBlockNode *st.MarkdownCodeBlockNode) {
	bLangDocLine := ast.BLangMarkdownDocumentationLine{}

	var docText strings.Builder

	langAttribute := codeBlockNode.LangAttribute()
	startBacktick := codeBlockNode.StartBacktick()
	if langAttribute != nil && !langAttribute.IsMissing() {
		if startBacktick != nil && !startBacktick.IsMissing() {
			docText.WriteString(startBacktick.Text())
		}
		docText.WriteString(langAttribute.Text())
	} else {
		if startBacktick != nil && !startBacktick.IsMissing() {
			docText.WriteString(startBacktick.Text())
		}
	}

	codeLines := codeBlockNode.CodeLines()
	for i := 0; i < codeLines.Size(); i++ {
		codeLine := codeLines.Get(i)
		codeDescription := codeLine.CodeDescription()
		if codeDescription != nil && !codeDescription.IsMissing() {
			docText.WriteString(codeDescription.Text())
		}
	}

	endBacktick := codeBlockNode.EndBacktick()
	if endBacktick != nil && !endBacktick.IsMissing() {
		docText.WriteString(endBacktick.Text())
	}

	bLangDocLine.Text = docText.String()
	bLangDocLine.SetPosition(n.getPosition(codeBlockNode.StartLineHashToken()))
	*documentationLines = append(*documentationLines, bLangDocLine)
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

func (n *nodeBuilder) transformOrderByClause(orderByClauseNode *st.OrderByClauseNode) ast.BLangNode {
	orderByClause := &ast.BLangOrderByClause{}
	orderByClause.SetPosition(n.getPosition(orderByClauseNode))

	orderKeys := orderByClauseNode.OrderKey()
	orderByClause.OrderByKeyList = make([]ast.BLangOrderKey, 0, orderKeys.Size())
	for orderKey := range orderKeys.Iterator() {
		keyNode, ok := n.transformOrderKey(orderKey).(*ast.BLangOrderKey)
		if !ok {
			panic("expected BLangOrderKey")
		}
		orderByClause.OrderByKeyList = append(orderByClause.OrderByKeyList, *keyNode)
	}
	return orderByClause
}

func (n *nodeBuilder) transformOrderKey(orderKeyNode *st.OrderKeyNode) ast.BLangNode {
	orderKey := &ast.BLangOrderKey{}
	orderKey.SetPosition(n.getPosition(orderKeyNode))
	orderKey.Expression = n.createExpression(orderKeyNode.Expression())
	if dir := orderKeyNode.OrderDirection(); dir != nil && dir.Kind() == st.DESCENDING_KEYWORD {
		orderKey.IsDescending = true
	} else {
		orderKey.IsDescending = false
	}
	return orderKey
}

func (n *nodeBuilder) transformGroupByClause(groupByClauseNode *st.GroupByClauseNode) ast.BLangNode {
	groupByClause := &ast.BLangGroupByClause{
		NonGroupingKeys: &balCommon.UnorderedSet[string]{},
	}
	groupByClause.SetPosition(n.getPosition(groupByClauseNode))

	groupingKeys := groupByClauseNode.GroupingKey()
	for node := range groupingKeys.Iterator() {
		if node.Kind() == st.COMMA_TOKEN {
			continue
		}
		var groupingKey *ast.BLangGroupingKey
		if node.Kind() == st.SIMPLE_NAME_REFERENCE || node.Kind() == st.IDENTIFIER_TOKEN {
			varRef, ok := n.createExpression(node).(*ast.BLangVarRef)
			if !ok {
				panic("expected grouping key variable reference to be a simple variable reference")
			}
			groupingKey = ast.NewBLangGroupingKeyWithVariableRef(n.getPosition(node), varRef)
		} else {
			keyNode, ok := n.transformGroupingKeyVarDeclaration(node.(*st.GroupingKeyVarDeclarationNode)).(*ast.BLangGroupingKey)
			if !ok {
				panic("expected grouping key declaration to produce a BLangGroupingKey")
			}
			groupingKey = keyNode
		}
		groupByClause.AddGroupingKey(groupingKey)
	}
	return groupByClause
}

func (n *nodeBuilder) transformGroupingKeyVarDeclaration(groupingKeyVarDeclarationNode *st.GroupingKeyVarDeclarationNode) ast.BLangNode {
	pos := n.getPosition(groupingKeyVarDeclarationNode)
	groupingKey := &ast.BLangGroupingKey{}
	groupingKey.SetPosition(pos)

	simpleVar := n.getBLangVariableNode(groupingKeyVarDeclarationNode.SimpleBindingPattern(), pos)
	simpleVar.SetPosition(pos)
	simpleVar.SetInitialExpression(n.createExpression(groupingKeyVarDeclarationNode.Expression()))

	typeDesc := groupingKeyVarDeclarationNode.TypeDescriptor()
	declaredWithVar := isDeclaredWithVar(typeDesc)
	var typeNode ast.BType
	if !declaredWithVar {
		typeNode = n.createTypeNode(typeDesc).(ast.BType)
	}
	simpleVar = ast.NewBLangVariable(pos, simpleVar.Name, typeNode, simpleVar.Expr, declaredWithVar, 0)
	simpleVar.SetPosition(pos)
	varDef := &ast.BLangVariableDef{Var: simpleVar}
	varDef.SetPosition(pos)
	groupingKey = ast.NewBLangGroupingKeyWithVariableDef(pos, varDef)
	return groupingKey
}

func (n *nodeBuilder) transformOnFailClause(onFailClauseNode *st.OnFailClauseNode) ast.BLangNode {
	panic("transformOnFailClause unimplemented")
}

func (n *nodeBuilder) transformDoStatement(doStatementNode *st.DoStatementNode) ast.BLangNode {
	panic("transformDoStatement unimplemented")
}

func (n *nodeBuilder) transformClassDefinition(classDefinitionNode *st.ClassDefinitionNode) ast.BLangNode {
	flags := classQualifierFlags(classDefinitionNode.ClassTypeQualifiers())
	if visibility := classDefinitionNode.VisibilityQualifier(); visibility != nil && visibility.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	blangClass := ast.NewBLangClassDefinitionWithFlags(flags)
	blangClass.SetPosition(n.getPositionWithoutMetadata(classDefinitionNode))

	n.populateMetadata(classDefinitionNode.Metadata(), &blangClass)

	// Set name
	nameIdentifier := createIdentifierFromToken(n.getPosition(classDefinitionNode.ClassName()), classDefinitionNode.ClassName())
	blangClass.Name = &nameIdentifier

	members := n.collectClassDefnMembers(classDefinitionNode.Members())
	blangClass.Fields = members.Fields
	blangClass.Methods = members.Methods
	blangClass.InitFunction = members.InitFunction
	blangClass.ResourceMethods = members.ResourceMethods
	for _, inclusion := range members.UnresolvedInclusions {
		blangClass.AddUnresolvedInclusion(inclusion)
	}

	return &blangClass
}

func classQualifierFlags(qualifiers st.NodeList[st.Token]) model.Flag {
	var flags model.Flag
	for qualifier := range qualifiers.Iterator() {
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
		}
	}
	return flags
}

func (n *nodeBuilder) transformClassField(objectField *st.ObjectFieldNode) *ast.BLangVariable {
	identifier := createIdentifierFromToken(n.getPosition(objectField.FieldName()), objectField.FieldName())
	var flags model.Flag
	if vis := objectField.VisibilityQualifier(); vis != nil && vis.Kind() == st.PUBLIC_KEYWORD {
		flags |= model.FlagPublic
	}
	qualifiers := objectField.QualifierList()
	for qualifier := range qualifiers.Iterator() {
		if qualifier.Kind() == st.FINAL_KEYWORD {
			flags |= model.FlagFinal
		}
	}
	var expr ast.BLangActionOrExpression
	if expression := objectField.Expression(); expression != nil {
		expr = n.createExpression(expression)
	}
	variable := ast.NewBLangVariable(
		n.getPosition(objectField),
		&identifier,
		n.createTypeNode(objectField.TypeName()).(ast.BType),
		expr,
		false,
		flags,
	)
	variable.SetPosition(n.getPosition(objectField))
	n.populateMetadata(objectField.Metadata(), variable)
	return variable
}

func (n *nodeBuilder) transformResourcePathParameter(resourcePathParameterNode *st.ResourcePathParameterNode) ast.BLangNode {
	seg := &ast.BLangResourcePathSegment{}
	switch resourcePathParameterNode.Kind() {
	case st.RESOURCE_PATH_SEGMENT_PARAM:
		seg.Kind = ast.ResourcePathSegmentParam
	case st.RESOURCE_PATH_REST_PARAM:
		seg.Kind = ast.ResourcePathSegmentParamRest
	default:
		n.cx.InternalError(fmt.Sprintf("unexpected resource path parameter node kind: %v", resourcePathParameterNode.Kind()), n.getPosition(resourcePathParameterNode))
	}
	seg.SetPosition(n.getPosition(resourcePathParameterNode))
	nameTok := resourcePathParameterNode.ParamName()
	if nameTok != nil && !nameTok.IsMissing() {
		seg.Name = createIdentifierFromToken(n.getPosition(nameTok), nameTok).Value
	}
	if td := resourcePathParameterNode.TypeDescriptor(); td != nil {
		seg.ParamType = n.createTypeNode(td).(ast.BType)
	}
	return seg
}

func (n *nodeBuilder) createResourceMethodNode(funcDef *st.FunctionDefinition) *ast.BLangResourceMethod {
	name := n.createIdentifierNodeFromToken(n.getPosition(funcDef.FunctionName()), funcDef.FunctionName())
	data := ast.InvokableData{
		Position: n.getPositionWithoutMetadata(funcDef),
		Name:     name,
		Flags:    functionQualifierFlags(funcDef.QualifierList()) | model.FlagAttached | model.FlagResource,
	}
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, name.GetValue())
	n.populateFuncSignature(&data, funcDef.FunctionSignature())
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	if body := funcDef.FunctionBody(); body == nil {
		data.Flags |= model.FlagInterface
	} else {
		data.Body = n.transformSyntaxNode(body).(ast.FunctionBodyNode)
		if _, ok := data.Body.(*ast.BLangExternFunctionBody); ok {
			data.Flags |= model.FlagNative
		}
	}
	rm := ast.NewBLangResourceMethod(data, n.createResourcePathSegments(funcDef.RelativeResourcePath()))
	n.populateMetadata(funcDef.Metadata(), rm)
	return rm
}

func (n *nodeBuilder) createResourcePathSegments(pathNodes st.NodeList[st.Node]) []ast.BLangResourcePathSegment {
	var segments []ast.BLangResourcePathSegment
	for node := range pathNodes.Iterator() {
		switch node.Kind() {
		case st.SLASH_TOKEN:
			continue
		case st.DOT_TOKEN:
			continue
		case st.IDENTIFIER_TOKEN:
			tok := node.(st.Token)
			seg := ast.BLangResourcePathSegment{Kind: ast.ResourcePathSegmentName, Name: tok.Text()}
			seg.SetPosition(n.getPosition(node))
			segments = append(segments, seg)
		case st.RESOURCE_PATH_SEGMENT_PARAM, st.RESOURCE_PATH_REST_PARAM:
			param := node.(*st.ResourcePathParameterNode)
			segments = append(segments, *n.transformResourcePathParameter(param).(*ast.BLangResourcePathSegment))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource path node kind: %v", node.Kind()), n.getPosition(node))
		}
	}
	return segments
}

func (n *nodeBuilder) transformRequiredExpression(requiredBLangExpression *st.RequiredExpressionNode) ast.BLangNode {
	panic("transformRequiredExpression unimplemented")
}

func (n *nodeBuilder) transformErrorConstructorExpression(errorConstructorBLangExpression *st.ErrorConstructorExpressionNode) ast.BLangNode {
	result := &ast.BLangErrorConstructorExpr{}
	result.SetPosition(n.getPosition(errorConstructorBLangExpression))

	typeRefNode := errorConstructorBLangExpression.TypeReference()
	if typeRefNode != nil {
		typeDesc := n.createTypeNode(typeRefNode)
		if userDefinedType, ok := typeDesc.(*ast.BLangUserDefinedType); ok {
			result.ErrorTypeRef = userDefinedType
		} else {
			n.cx.InternalError("error type reference must be a user-defined type", result.GetPosition())
		}
	}

	arguments := errorConstructorBLangExpression.Arguments()
	positionalArgs := make([]ast.BLangExpression, 0)
	namedArgs := make([]ast.BLangNamedArgsExpression, 0)

	for arg := range arguments.Iterator() {
		switch arg.Kind() {
		case st.POSITIONAL_ARG:
			posArg := arg.(*st.PositionalArgumentNode)
			expr := n.createExpression(posArg.Expression())
			positionalArgs = append(positionalArgs, expr)

		case st.NAMED_ARG:
			namedArgNode := arg.(*st.NamedArgumentNode)
			namedArg := n.transformNamedArgument(namedArgNode).(*ast.BLangNamedArgsExpression)
			namedArgs = append(namedArgs, *namedArg)
		case st.REST_ARG:
			n.cx.InternalError("rest arguments not supported in error constructor", n.getPosition(arg))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected argument kind: %v", arg.Kind()), n.getPosition(arg))
		}
	}

	result.PositionalArgs = positionalArgs
	result.NamedArgs = namedArgs

	return result
}

func (n *nodeBuilder) transformParameterizedTypeDescriptor(parameterizedTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) ast.BLangNode {
	switch parameterizedTypeDescriptorNode.Kind() {
	case st.ERROR_TYPE_DESC:
		return n.transformErrorTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.TYPEDESC_TYPE_DESC:
		return n.transformTypedescTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.XML_TYPE_DESC:
		return n.transformXMLTypeDescriptor(parameterizedTypeDescriptorNode)
	}
	panic("transformParameterizedTypeDescriptor supported only for error, typedesc and xml type descriptors")
}

func (n *nodeBuilder) transformTypedescTypeDescriptor(node *st.ParameterizedTypeDescriptorNode) ast.BLangNode {
	typeParamNode := node.TypeParamNode()
	if typeParamNode == nil {
		valueType := &ast.BLangValueType{}
		valueType.SetPosition(n.getPosition(node))
		valueType.TypeKind = ast.TypeKindTypeDesc
		return valueType
	}
	constrainedType := &ast.BLangConstrainedType{}
	constrainedType.SetPosition(n.getPosition(node))
	base := &ast.BLangValueType{}
	base.SetPosition(n.getPosition(node))
	base.TypeKind = ast.TypeKindTypeDesc
	constrainedType.Type = ast.TypeData{TypeDescriptor: base}
	constraint := typeParamNode.TypeNode()
	if constraint == nil {
		constrainedType.Constraint = ast.TypeData{TypeDescriptor: n.createTypeNode(typeParamNode)}
	} else {
		constrainedType.Constraint = ast.TypeData{TypeDescriptor: n.createTypeNode(constraint)}
	}
	return constrainedType
}

func (n *nodeBuilder) transformXMLTypeDescriptor(parameterizedTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) ast.BLangNode {
	pos := n.getPosition(parameterizedTypeDescriptorNode)
	typeParamNode := parameterizedTypeDescriptorNode.TypeParamNode()
	if typeParamNode == nil {
		valueType := &ast.BLangValueType{}
		valueType.SetPosition(pos)
		valueType.TypeKind = ast.TypeKindXML
		return valueType
	}
	refType := &ast.BLangBuiltInRefTypeNode{
		TypeKind: ast.TypeKindXML,
	}
	refType.SetPosition(pos)
	constraint := n.createTypeNode(typeParamNode.TypeNode())
	constrainedType := &ast.BLangConstrainedType{
		Type:       ast.TypeData{TypeDescriptor: refType},
		Constraint: ast.TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(pos)
	return constrainedType
}

func (n *nodeBuilder) transformErrorTypeDescriptor(errorTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) ast.BLangNode {
	var detailType ast.TypeData
	if typeParamNode := errorTypeDescriptorNode.TypeParamNode(); typeParamNode != nil {
		detailType.TypeDescriptor = n.createTypeNode(typeParamNode)
	}
	return ast.NewBLangErrorTypeNode(
		n.getPosition(errorTypeDescriptorNode),
		detailType,
		errorTypeDescriptorNode.Parent().Kind() == st.DISTINCT_TYPE_DESC,
	)
}

func (n *nodeBuilder) transformSpreadMember(spreadMemberNode *st.SpreadMemberNode) ast.BLangNode {
	return n.createExpression(spreadMemberNode.Expression()).(ast.BLangNode)
}

func (n *nodeBuilder) transformClientResourceAccessAction(node *st.ClientResourceAccessActionNode) ast.BLangNode {
	action := &ast.BLangClientResourceAccessAction{}
	action.SetPosition(n.getPosition(node))
	action.Expr = n.createExpression(node.Expression())
	action.MethodName = "get"
	if methodName := node.MethodName(); methodName != nil {
		nameTok := methodName.Name()
		if nameTok == nil || nameTok.IsMissing() {
			n.cx.InternalError("missing method name token in resource access action", action.GetPosition())
		} else {
			action.MethodName = nameTok.Text()
		}
	}
	nameID := &ast.BLangIdentifier{Value: action.MethodName}
	nameID.SetPosition(action.GetPosition())
	action.Name = nameID
	action.Path = n.createResourceAccessSegments(node.ResourceAccessPath())
	if args := node.Arguments(); args != nil {
		var argExprs []ast.BLangExpression
		argList := args.Arguments()
		for arg := range argList.Iterator() {
			argExprs = append(argExprs, n.createExpression(arg))
		}
		action.ArgExprs = argExprs
	}
	return action
}

func (n *nodeBuilder) createResourceAccessSegments(pathNodes st.NodeList[st.Node]) []ast.BLangResourceAccessSegment {
	var segments []ast.BLangResourceAccessSegment
	for node := range pathNodes.Iterator() {
		switch node.Kind() {
		case st.SLASH_TOKEN, st.DOT_TOKEN:
			continue
		case st.IDENTIFIER_TOKEN:
			tok := node.(st.Token)
			seg := ast.BLangResourceAccessSegment{Kind: ast.ResourceAccessSegmentName, Name: tok.Text()}
			seg.SetPosition(n.getPosition(node))
			segments = append(segments, seg)
		case st.COMPUTED_RESOURCE_ACCESS_SEGMENT:
			computed := node.(*st.ComputedResourceAccessSegmentNode)
			segments = append(segments, *n.transformComputedResourceAccessSegment(computed).(*ast.BLangResourceAccessSegment))
		case st.RESOURCE_ACCESS_REST_SEGMENT:
			n.cx.Unimplemented("resource access rest segments are not yet supported", n.getPosition(node))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource access segment kind: %v", node.Kind()), n.getPosition(node))
		}
	}
	return segments
}

func (n *nodeBuilder) transformComputedResourceAccessSegment(node *st.ComputedResourceAccessSegmentNode) ast.BLangNode {
	seg := &ast.BLangResourceAccessSegment{Kind: ast.ResourceAccessSegmentComputed}
	seg.SetPosition(n.getPosition(node))
	seg.Expr = n.createExpression(node.Expression())
	return seg
}

func (n *nodeBuilder) transformResourceAccessRestSegment(resourceAccessRestSegmentNode *st.ResourceAccessRestSegmentNode) ast.BLangNode {
	panic("transformResourceAccessRestSegment unimplemented")
}

func (n *nodeBuilder) transformReSequence(reSequenceNode *st.ReSequenceNode) ast.BLangNode {
	panic("transformReSequence unimplemented")
}

func (n *nodeBuilder) transformReAtomQuantifier(reAtomQuantifierNode *st.ReAtomQuantifierNode) ast.BLangNode {
	panic("transformReAtomQuantifier unimplemented")
}

func (n *nodeBuilder) transformReAtomCharOrEscape(reAtomCharOrEscapeNode *st.ReAtomCharOrEscapeNode) ast.BLangNode {
	panic("transformReAtomCharOrEscape unimplemented")
}

func (n *nodeBuilder) transformReQuoteEscape(reQuoteEscapeNode *st.ReQuoteEscapeNode) ast.BLangNode {
	panic("transformReQuoteEscape unimplemented")
}

func (n *nodeBuilder) transformReSimpleCharClassEscape(reSimpleCharClassEscapeNode *st.ReSimpleCharClassEscapeNode) ast.BLangNode {
	panic("transformReSimpleCharClassEscape unimplemented")
}

func (n *nodeBuilder) transformReUnicodePropertyEscape(reUnicodePropertyEscapeNode *st.ReUnicodePropertyEscapeNode) ast.BLangNode {
	panic("transformReUnicodePropertyEscape unimplemented")
}

func (n *nodeBuilder) transformReUnicodeScript(reUnicodeScriptNode *st.ReUnicodeScriptNode) ast.BLangNode {
	panic("transformReUnicodeScript unimplemented")
}

func (n *nodeBuilder) transformReUnicodeGeneralCategory(reUnicodeGeneralCategoryNode *st.ReUnicodeGeneralCategoryNode) ast.BLangNode {
	panic("transformReUnicodeGeneralCategory unimplemented")
}

func (n *nodeBuilder) transformReCharacterClass(reCharacterClassNode *st.ReCharacterClassNode) ast.BLangNode {
	panic("transformReCharacterClass unimplemented")
}

func (n *nodeBuilder) transformReCharSetRangeWithReCharSet(reCharSetRangeWithReCharSetNode *st.ReCharSetRangeWithReCharSetNode) ast.BLangNode {
	panic("transformReCharSetRangeWithReCharSet unimplemented")
}

func (n *nodeBuilder) transformReCharSetRange(reCharSetRangeNode *st.ReCharSetRangeNode) ast.BLangNode {
	panic("transformReCharSetRange unimplemented")
}

func (n *nodeBuilder) transformReCharSetAtomWithReCharSetNoDash(reCharSetAtomWithReCharSetNoDashNode *st.ReCharSetAtomWithReCharSetNoDashNode) ast.BLangNode {
	panic("transformReCharSetAtomWithReCharSetNoDash unimplemented")
}

func (n *nodeBuilder) transformReCharSetRangeNoDashWithReCharSet(reCharSetRangeNoDashWithReCharSetNode *st.ReCharSetRangeNoDashWithReCharSetNode) ast.BLangNode {
	panic("transformReCharSetRangeNoDashWithReCharSet unimplemented")
}

func (n *nodeBuilder) transformReCharSetRangeNoDash(reCharSetRangeNoDashNode *st.ReCharSetRangeNoDashNode) ast.BLangNode {
	panic("transformReCharSetRangeNoDash unimplemented")
}

func (n *nodeBuilder) transformReCharSetAtomNoDashWithReCharSetNoDash(reCharSetAtomNoDashWithReCharSetNoDashNode *st.ReCharSetAtomNoDashWithReCharSetNoDashNode) ast.BLangNode {
	panic("transformReCharSetAtomNoDashWithReCharSetNoDash unimplemented")
}

func (n *nodeBuilder) transformReCapturingGroups(reCapturingGroupsNode *st.ReCapturingGroupsNode) ast.BLangNode {
	panic("transformReCapturingGroups unimplemented")
}

func (n *nodeBuilder) transformReFlagExpression(reFlagBLangExpression *st.ReFlagExpressionNode) ast.BLangNode {
	panic("transformReFlagExpression unimplemented")
}

func (n *nodeBuilder) transformReFlagsOnOff(reFlagsOnOffNode *st.ReFlagsOnOffNode) ast.BLangNode {
	panic("transformReFlagsOnOff unimplemented")
}

func (n *nodeBuilder) transformReFlags(reFlagsNode *st.ReFlagsNode) ast.BLangNode {
	panic("transformReFlags unimplemented")
}

func (n *nodeBuilder) transformReAssertion(reAssertionNode *st.ReAssertionNode) ast.BLangNode {
	panic("transformReAssertion unimplemented")
}

func (n *nodeBuilder) transformReQuantifier(reQuantifierNode *st.ReQuantifierNode) ast.BLangNode {
	panic("transformReQuantifier unimplemented")
}

func (n *nodeBuilder) transformReBracedQuantifier(reBracedQuantifierNode *st.ReBracedQuantifierNode) ast.BLangNode {
	panic("transformReBracedQuantifier unimplemented")
}

func (n *nodeBuilder) transformMemberTypeDescriptor(memberTypeDescriptorNode *st.MemberTypeDescriptorNode) ast.BLangNode {
	panic("transformMemberTypeDescriptor unimplemented")
}

func (n *nodeBuilder) transformReceiveField(receiveFieldNode *st.ReceiveFieldNode) ast.BLangNode {
	panic("transformReceiveField unimplemented")
}

func (n *nodeBuilder) transformNaturalExpression(naturalBLangExpression *st.NaturalExpressionNode) ast.BLangNode {
	panic("transformNaturalExpression unimplemented")
}

func (n *nodeBuilder) transformToken(token st.Token) ast.BLangNode {
	kind := token.Kind()
	switch kind {
	case st.XML_TEXT_CONTENT, st.TEMPLATE_STRING, st.CLOSE_BRACE_TOKEN, st.PROMPT_CONTENT:
		return n.createSimpleLiteral(token).(ast.BLangNode)
	default:
		if isTokenInRegExp(kind) {
			return n.createSimpleLiteral(token).(ast.BLangNode)
		}
		panic("transformToken: Syntax kind is not supported: " + kind.StrValue())
	}
}

func (n *nodeBuilder) transformIdentifierToken(identifier *st.IdentifierToken) ast.BLangNode {
	panic("transformIdentifierToken unimplemented")
}

func stringToTypeKind(typeText string) ast.TypeKind {
	switch typeText {
	case "int":
		return ast.TypeKindInt
	case "byte":
		return ast.TypeKindByte
	case "float":
		return ast.TypeKindFloat
	case "decimal":
		return ast.TypeKindDecimal
	case "boolean":
		return ast.TypeKindBoolean
	case "string":
		return ast.TypeKindString
	case "json":
		return ast.TypeKindJSON
	case "xml":
		return ast.TypeKindXML
	case "stream":
		return ast.TypeKindStream
	case "table":
		return ast.TypeKindTable
	case "any":
		return ast.TypeKindAny
	case "anydata":
		return ast.TypeKindAnyData
	case "map":
		return ast.TypeKindMap
	case "future":
		return ast.TypeKindFuture
	case "typedesc":
		return ast.TypeKindTypeDesc
	case "error":
		return ast.TypeKindError
	case "()", "null":
		return ast.TypeKindNil
	case "never":
		return ast.TypeKindNever
	case "channel":
		return ast.TypeKindChannel
	case "service":
		return ast.TypeKindService
	case "handle":
		return ast.TypeKindHandle
	case "readonly":
		return ast.TypeKindReadOnly
	case "function":
		return ast.TypeKindFunction
	default:
		panic("stringToTypeKind: invalid type name: " + typeText)
	}
}

func createUserDefinedType(pos diagnostics.Location, pkgAlias ast.BLangIdentifier, typeName ast.BLangIdentifier) ast.TypeDescriptor {
	userDefinedType := ast.BLangUserDefinedType{}
	userDefinedType.SetPosition(pos)
	userDefinedType.PkgAlias = pkgAlias
	userDefinedType.TypeName = typeName
	return &userDefinedType
}

func getNextMissingNodeName(pkgID *model.PackageID) string {
	panic("getNextMissingNodeName unimplemented")
}

func (n *nodeBuilder) getBLangVariableNode(bindingPattern st.BindingPatternNode, varPos diagnostics.Location) *ast.BLangVariable {
	var varName st.Token
	switch bindingPattern.Kind() {
	case st.WILDCARD_BINDING_PATTERN:
		ignore := n.createIgnoreIdentifier(bindingPattern)
		simpleVar := ast.NewBLangVariable(varPos, &ignore, nil, nil, false, 0)
		simpleVar.SetPosition(varPos)
		return simpleVar
	case st.MAPPING_BINDING_PATTERN, st.LIST_BINDING_PATTERN, st.ERROR_BINDING_PATTERN, st.REST_BINDING_PATTERN:
		panic("unimplemented")
	case st.CAPTURE_BINDING_PATTERN:
		fallthrough
	default:
		captureBindingPattern := bindingPattern.(*st.CaptureBindingPatternNode)
		varName = captureBindingPattern.VariableName()
	}

	simpleVar := ast.NewBLangVariable(
		varPos,
		n.createIdentifierNodeFromToken(n.getPosition(varName), varName),
		nil,
		nil,
		false,
		0,
	)
	simpleVar.SetPosition(varPos)
	return simpleVar
}

func (n *nodeBuilder) badTopLevel(node st.Node) *ast.BLangBadTopLevelNode {
	bad := &ast.BLangBadTopLevelNode{}
	bad.SetPosition(n.getRecoveryPosition(node))
	return bad
}

func (n *nodeBuilder) badStmt(node st.Node) *ast.BLangBadStmt {
	bad := &ast.BLangBadStmt{}
	bad.SetPosition(n.getRecoveryPosition(node))
	return bad
}

func (n *nodeBuilder) badExprOrAction(node st.Node) *ast.BLangBadExprOrAction {
	bad := &ast.BLangBadExprOrAction{}
	if node != nil {
		bad.SetPosition(n.getRecoveryPosition(node))
	} else {
		bad.SetPosition(diagnostics.NewBuiltinLocation())
	}
	return bad
}

func (n *nodeBuilder) badTypeNode(node st.Node) *ast.BLangBadTypeNode {
	bad := &ast.BLangBadTypeNode{}
	if node != nil {
		bad.SetPosition(n.getRecoveryPosition(node))
	} else {
		bad.SetPosition(diagnostics.NewBuiltinLocation())
	}
	return bad
}

func (n *nodeBuilder) badIdentifier(token st.Token) *ast.BLangBadIdentifier {
	if token == nil {
		return ast.NewBLangBadIdentifier(diagnostics.NewBuiltinLocation(), "", "", false)
	}
	value, isLiteral := normalizedIdentifierValue(token.Text())
	return ast.NewBLangBadIdentifier(n.getRecoveryPosition(token), value, token.Text(), isLiteral)
}

func (n *nodeBuilder) syntaxError(node st.Node) {
	diagnosticNodes := innermostDiagnosticNodes(node)
	if len(diagnosticNodes) == 0 {
		return
	}
	for _, diagnosticNode := range diagnosticNodes {
		deep := st.FindDeepestDiagnosticSTNode(diagnosticNode.InternalNode())
		if deep == nil || len(deep.Diagnostics()) == 0 {
			continue
		}
		for _, diagnostic := range deep.Diagnostics() {
			n.cx.SyntaxError(diagnosticMessage(diagnostic), n.getPosition(diagnosticNode))
		}
	}
}
