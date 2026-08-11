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

package ast

import (
	"fmt"
	"iter"
	"math"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"

	balCommon "github.com/ballerina-nutcracker/ballerina/common"
)

type typeTable struct {
	booleanType *BTypeBasic
	intType     *BTypeBasic
	nilType     *BTypeBasic
	stringType  *BTypeBasic
	floatType   *BTypeBasic
	decimalType *BTypeBasic
	byteType    *BTypeBasic
}

func newTypeTable() typeTable {
	return typeTable{
		booleanType: &BTypeBasic{tag: TypeTags_BOOLEAN, flags: model.FlagReadonly},
		intType:     &BTypeBasic{tag: TypeTags_INT, flags: model.FlagReadonly},
		nilType:     &BTypeBasic{tag: TypeTags_NIL, flags: model.FlagReadonly},
		stringType:  &BTypeBasic{tag: TypeTags_STRING, flags: model.FlagReadonly},
		floatType:   &BTypeBasic{tag: TypeTags_FLOAT, flags: model.FlagReadonly},
		decimalType: &BTypeBasic{tag: TypeTags_DECIMAL, flags: model.FlagReadonly},
		byteType:    &BTypeBasic{tag: TypeTags_BYTE, flags: model.FlagReadonly},
	}
}

func (t *typeTable) getTypeFromTag(tag TypeTags) TypeDescriptor {
	switch tag {
	case TypeTags_BOOLEAN:
		return t.booleanType
	case TypeTags_INT:
		return t.intType
	case TypeTags_NIL:
		return t.nilType
	case TypeTags_STRING:
		return t.stringType
	case TypeTags_FLOAT:
		return t.floatType
	case TypeTags_DECIMAL:
		return t.decimalType
	case TypeTags_BYTE:
		return t.byteType
	default:
		panic("not implemented")
	}
}

type NodeBuilderMode uint8

const (
	NodeBuilderModeStrict NodeBuilderMode = iota
	NodeBuilderModeRecover
)

type NodeBuilder struct {
	PackageID            *model.PackageID
	anonTypeNameSuffixes []string // Stack for anonymous type name suffixes
	currentCompUnit      *BLangCompilationUnit
	cx                   *context.CompilerContext
	types                typeTable
	mode                 NodeBuilderMode
}

func (n *NodeBuilder) de() *diagnostics.DiagnosticEnv {
	return n.cx.DiagnosticEnv()
}

// NewNodeBuilder creates and initializes a new NodeBuilder instance
func NewNodeBuilder(cx *context.CompilerContext) *NodeBuilder {
	return newNodeBuilder(cx, NodeBuilderModeStrict)
}

func NewRecoveringNodeBuilder(cx *context.CompilerContext) *NodeBuilder {
	return newNodeBuilder(cx, NodeBuilderModeRecover)
}

func newNodeBuilder(cx *context.CompilerContext, mode NodeBuilderMode) *NodeBuilder {
	nodeBuilder := &NodeBuilder{
		cx:        cx,
		PackageID: cx.GetDefaultPackage(),
		types:     newTypeTable(),
		mode:      mode,
	}
	return nodeBuilder
}

var _ st.NodeTransformer[BLangNode] = &NodeBuilder{}

const (
	OPEN_ARRAY_INDICATOR     = -1
	INFERRED_ARRAY_INDICATOR = -2
)

func (n *NodeBuilder) TransformSyntaxNode(node st.Node) BLangNode {
	switch t := node.(type) {
	case *st.ModulePart:
		return n.TransformModulePart(t)
	case *st.FunctionDefinition:
		return n.TransformFunctionDefinition(t)
	case *st.ImportDeclarationNode:
		return n.TransformImportDeclaration(t)
	case *st.ListenerDeclarationNode:
		return n.TransformListenerDeclaration(t)
	case *st.TypeDefinitionNode:
		return n.TransformTypeDefinition(t)
	case *st.ServiceDeclarationNode:
		return n.TransformServiceDeclaration(t)
	case *st.AssignmentStatementNode:
		return n.TransformAssignmentStatement(t)
	case *st.CompoundAssignmentStatementNode:
		return n.TransformCompoundAssignmentStatement(t)
	case *st.VariableDeclarationNode:
		return n.TransformVariableDeclaration(t)
	case *st.BlockStatementNode:
		return n.TransformBlockStatement(t)
	case *st.BreakStatementNode:
		return n.TransformBreakStatement(t)
	case *st.FailStatementNode:
		return n.TransformFailStatement(t)
	case *st.ExpressionStatementNode:
		return n.TransformExpressionStatement(t)
	case *st.ContinueStatementNode:
		return n.TransformContinueStatement(t)
	case *st.ExternalFunctionBodyNode:
		return n.TransformExternalFunctionBody(t)
	case *st.IfElseStatementNode:
		return n.TransformIfElseStatement(t)
	case *st.ElseBlockNode:
		return n.TransformElseBlock(t)
	case *st.WhileStatementNode:
		return n.TransformWhileStatement(t)
	case *st.PanicStatementNode:
		return n.TransformPanicStatement(t)
	case *st.ReturnStatementNode:
		return n.TransformReturnStatement(t)
	case *st.LocalTypeDefinitionStatementNode:
		return n.TransformLocalTypeDefinitionStatement(t)
	case *st.LockStatementNode:
		return n.TransformLockStatement(t)
	case *st.ForkStatementNode:
		return n.TransformForkStatement(t)
	case *st.ForEachStatementNode:
		return n.TransformForEachStatement(t)
	case *st.BinaryExpressionNode:
		return n.TransformBinaryExpression(t)
	case *st.BracedExpressionNode:
		return n.TransformBracedExpression(t)
	case *st.CheckExpressionNode:
		return n.TransformCheckExpression(t)
	case *st.FieldAccessExpressionNode:
		return n.TransformFieldAccessExpression(t)
	case *st.FunctionCallExpressionNode:
		return n.TransformFunctionCallExpression(t)
	case *st.MethodCallExpressionNode:
		return n.TransformMethodCallExpression(t)
	case *st.MappingConstructorExpressionNode:
		return n.TransformMappingConstructorExpression(t)
	case *st.IndexedExpressionNode:
		return n.TransformIndexedExpression(t)
	case *st.TypeofExpressionNode:
		return n.TransformTypeofExpression(t)
	case *st.UnaryExpressionNode:
		return n.TransformUnaryExpression(t)
	case *st.ComputedNameFieldNode:
		return n.TransformComputedNameField(t)
	case *st.ConstantDeclarationNode:
		return n.TransformConstantDeclaration(t)
	case *st.DefaultableParameterNode:
		return n.TransformDefaultableParameter(t)
	case *st.RequiredParameterNode:
		return n.TransformRequiredParameter(t)
	case *st.IncludedRecordParameterNode:
		return n.TransformIncludedRecordParameter(t)
	case *st.RestParameterNode:
		return n.TransformRestParameter(t)
	case *st.ImportOrgNameNode:
		return n.TransformImportOrgName(t)
	case *st.ImportPrefixNode:
		return n.TransformImportPrefix(t)
	case *st.SpecificFieldNode:
		return n.TransformSpecificField(t)
	case *st.SpreadFieldNode:
		return n.TransformSpreadField(t)
	case *st.NamedArgumentNode:
		return n.TransformNamedArgument(t)
	case *st.PositionalArgumentNode:
		return n.TransformPositionalArgument(t)
	case *st.RestArgumentNode:
		return n.TransformRestArgument(t)
	case *st.InferredTypedescDefaultNode:
		return n.TransformInferredTypedescDefault(t)
	case *st.ObjectTypeDescriptorNode:
		return n.TransformObjectTypeDescriptor(t)
	case *st.ObjectConstructorExpressionNode:
		return n.TransformObjectConstructorExpression(t)
	case *st.RecordTypeDescriptorNode:
		return n.TransformRecordTypeDescriptor(t)
	case *st.ReturnTypeDescriptorNode:
		return n.TransformReturnTypeDescriptor(t)
	case *st.NilTypeDescriptorNode:
		return n.TransformNilTypeDescriptor(t)
	case *st.OptionalTypeDescriptorNode:
		return n.TransformOptionalTypeDescriptor(t)
	case *st.ObjectFieldNode:
		return n.TransformObjectField(t)
	case *st.RecordFieldNode:
		return n.TransformRecordField(t)
	case *st.RecordFieldWithDefaultValueNode:
		return n.TransformRecordFieldWithDefaultValue(t)
	case *st.RecordRestDescriptorNode:
		return n.TransformRecordRestDescriptor(t)
	case *st.TypeReferenceNode:
		return n.TransformTypeReference(t)
	case *st.AnnotationNode:
		return n.TransformAnnotation(t)
	case *st.MetadataNode:
		return n.TransformMetadata(t)
	case *st.ModuleVariableDeclarationNode:
		return n.TransformModuleVariableDeclaration(t)
	case *st.TypeTestExpressionNode:
		return n.TransformTypeTestExpression(t)
	case *st.RemoteMethodCallActionNode:
		return n.TransformRemoteMethodCallAction(t)
	case *st.MapTypeDescriptorNode:
		return n.TransformMapTypeDescriptor(t)
	case *st.NilLiteralNode:
		return n.TransformNilLiteral(t)
	case *st.AnnotationDeclarationNode:
		return n.TransformAnnotationDeclaration(t)
	case *st.AnnotationAttachPointNode:
		return n.TransformAnnotationAttachPoint(t)
	case *st.XMLNamespaceDeclarationNode:
		return n.TransformXMLNamespaceDeclaration(t)
	case *st.ModuleXMLNamespaceDeclarationNode:
		return n.TransformModuleXMLNamespaceDeclaration(t)
	case *st.FunctionBodyBlockNode:
		return n.TransformFunctionBodyBlock(t)
	case *st.NamedWorkerDeclarationNode:
		return n.TransformNamedWorkerDeclaration(t)
	case *st.NamedWorkerDeclarator:
		return n.TransformNamedWorkerDeclarator(t)
	case *st.BasicLiteralNode:
		return n.TransformBasicLiteral(t)
	case *st.SimpleNameReferenceNode:
		return n.TransformSimpleNameReference(t)
	case *st.QualifiedNameReferenceNode:
		return n.TransformQualifiedNameReference(t)
	case *st.BuiltinSimpleNameReferenceNode:
		return n.TransformBuiltinSimpleNameReference(t)
	case *st.TrapExpressionNode:
		return n.TransformTrapExpression(t)
	case *st.ListConstructorExpressionNode:
		return n.TransformListConstructorExpression(t)
	case *st.TypeCastExpressionNode:
		return n.TransformTypeCastExpression(t)
	case *st.TypeCastParamNode:
		return n.TransformTypeCastParam(t)
	case *st.UnionTypeDescriptorNode:
		return n.TransformUnionTypeDescriptor(t)
	case *st.TableConstructorExpressionNode:
		return n.TransformTableConstructorExpression(t)
	case *st.KeySpecifierNode:
		return n.TransformKeySpecifier(t)
	case *st.StreamTypeDescriptorNode:
		return n.TransformStreamTypeDescriptor(t)
	case *st.StreamTypeParamsNode:
		return n.TransformStreamTypeParams(t)
	case *st.LetExpressionNode:
		return n.TransformLetExpression(t)
	case *st.LetVariableDeclarationNode:
		return n.TransformLetVariableDeclaration(t)
	case *st.TemplateExpressionNode:
		return n.TransformTemplateExpression(t)
	case *st.XMLElementNode:
		return n.TransformXMLElement(t)
	case *st.XMLStartTagNode:
		return n.TransformXMLStartTag(t)
	case *st.XMLEndTagNode:
		return n.TransformXMLEndTag(t)
	case *st.XMLSimpleNameNode:
		return n.TransformXMLSimpleName(t)
	case *st.XMLQualifiedNameNode:
		return n.TransformXMLQualifiedName(t)
	case *st.XMLEmptyElementNode:
		return n.TransformXMLEmptyElement(t)
	case *st.InterpolationNode:
		return n.TransformInterpolation(t)
	case *st.XMLTextNode:
		return n.TransformXMLText(t)
	case *st.XMLAttributeNode:
		return n.TransformXMLAttribute(t)
	case *st.XMLAttributeValue:
		return n.TransformXMLAttributeValue(t)
	case *st.XMLComment:
		return n.TransformXMLComment(t)
	case *st.XMLCDATANode:
		return n.TransformXMLCDATA(t)
	case *st.XMLProcessingInstruction:
		return n.TransformXMLProcessingInstruction(t)
	case *st.TableTypeDescriptorNode:
		return n.TransformTableTypeDescriptor(t)
	case *st.TypeParameterNode:
		return n.TransformTypeParameter(t)
	case *st.KeyTypeConstraintNode:
		return n.TransformKeyTypeConstraint(t)
	case *st.FunctionTypeDescriptorNode:
		return n.TransformFunctionTypeDescriptor(t)
	case *st.FunctionSignatureNode:
		return n.TransformFunctionSignature(t)
	case *st.ExplicitAnonymousFunctionExpressionNode:
		return n.TransformExplicitAnonymousFunctionExpression(t)
	case *st.ExpressionFunctionBodyNode:
		return n.TransformExpressionFunctionBody(t)
	case *st.TupleTypeDescriptorNode:
		return n.TransformTupleTypeDescriptor(t)
	case *st.ParenthesisedTypeDescriptorNode:
		return n.TransformParenthesisedTypeDescriptor(t)
	case *st.ExplicitNewExpressionNode:
		return n.TransformExplicitNewExpression(t)
	case *st.ImplicitNewExpressionNode:
		return n.TransformImplicitNewExpression(t)
	case *st.ParenthesizedArgList:
		return n.TransformParenthesizedArgList(t)
	case *st.QueryConstructTypeNode:
		return n.TransformQueryConstructType(t)
	case *st.FromClauseNode:
		return n.TransformFromClause(t)
	case *st.WhereClauseNode:
		return n.TransformWhereClause(t)
	case *st.LetClauseNode:
		return n.TransformLetClause(t)
	case *st.JoinClauseNode:
		return n.TransformJoinClause(t)
	case *st.OnClauseNode:
		return n.TransformOnClause(t)
	case *st.LimitClauseNode:
		return n.TransformLimitClause(t)
	case *st.OnConflictClauseNode:
		return n.TransformOnConflictClause(t)
	case *st.QueryPipelineNode:
		return n.TransformQueryPipeline(t)
	case *st.SelectClauseNode:
		return n.TransformSelectClause(t)
	case *st.CollectClauseNode:
		return n.TransformCollectClause(t)
	case *st.QueryExpressionNode:
		return n.TransformQueryExpression(t)
	case *st.QueryActionNode:
		return n.TransformQueryAction(t)
	case *st.IntersectionTypeDescriptorNode:
		return n.TransformIntersectionTypeDescriptor(t)
	case *st.ImplicitAnonymousFunctionParameters:
		return n.TransformImplicitAnonymousFunctionParameters(t)
	case *st.ImplicitAnonymousFunctionExpressionNode:
		return n.TransformImplicitAnonymousFunctionExpression(t)
	case *st.StartActionNode:
		return n.TransformStartAction(t)
	case *st.FlushActionNode:
		return n.TransformFlushAction(t)
	case *st.SingletonTypeDescriptorNode:
		return n.TransformSingletonTypeDescriptor(t)
	case *st.MethodDeclarationNode:
		return n.TransformMethodDeclaration(t)
	case *st.TypedBindingPatternNode:
		return n.TransformTypedBindingPattern(t)
	case *st.CaptureBindingPatternNode:
		return n.TransformCaptureBindingPattern(t)
	case *st.WildcardBindingPatternNode:
		return n.TransformWildcardBindingPattern(t)
	case *st.ListBindingPatternNode:
		return n.TransformListBindingPattern(t)
	case *st.MappingBindingPatternNode:
		return n.TransformMappingBindingPattern(t)
	case *st.FieldBindingPatternFullNode:
		return n.TransformFieldBindingPatternFull(t)
	case *st.FieldBindingPatternVarnameNode:
		return n.TransformFieldBindingPatternVarname(t)
	case *st.RestBindingPatternNode:
		return n.TransformRestBindingPattern(t)
	case *st.ErrorBindingPatternNode:
		return n.TransformErrorBindingPattern(t)
	case *st.NamedArgBindingPatternNode:
		return n.TransformNamedArgBindingPattern(t)
	case *st.AsyncSendActionNode:
		return n.TransformAsyncSendAction(t)
	case *st.SyncSendActionNode:
		return n.TransformSyncSendAction(t)
	case *st.ReceiveActionNode:
		return n.TransformReceiveAction(t)
	case *st.ReceiveFieldsNode:
		return n.TransformReceiveFields(t)
	case *st.AlternateReceiveNode:
		return n.TransformAlternateReceive(t)
	case *st.RestDescriptorNode:
		return n.TransformRestDescriptor(t)
	case *st.DoubleGTTokenNode:
		return n.TransformDoubleGTToken(t)
	case *st.TrippleGTTokenNode:
		return n.TransformTrippleGTToken(t)
	case *st.WaitActionNode:
		return n.TransformWaitAction(t)
	case *st.WaitFieldsListNode:
		return n.TransformWaitFieldsList(t)
	case *st.WaitFieldNode:
		return n.TransformWaitField(t)
	case *st.AnnotAccessExpressionNode:
		return n.TransformAnnotAccessExpression(t)
	case *st.OptionalFieldAccessExpressionNode:
		return n.TransformOptionalFieldAccessExpression(t)
	case *st.ConditionalExpressionNode:
		return n.TransformConditionalExpression(t)
	case *st.EnumDeclarationNode:
		return n.TransformEnumDeclaration(t)
	case *st.EnumMemberNode:
		return n.TransformEnumMember(t)
	case *st.ArrayTypeDescriptorNode:
		return n.TransformArrayTypeDescriptor(t)
	case *st.ArrayDimensionNode:
		return n.TransformArrayDimension(t)
	case *st.TransactionStatementNode:
		return n.TransformTransactionStatement(t)
	case *st.RollbackStatementNode:
		return n.TransformRollbackStatement(t)
	case *st.RetryStatementNode:
		return n.TransformRetryStatement(t)
	case *st.CommitActionNode:
		return n.TransformCommitAction(t)
	case *st.TransactionalExpressionNode:
		return n.TransformTransactionalExpression(t)
	case *st.ByteArrayLiteralNode:
		return n.TransformByteArrayLiteral(t)
	case *st.XMLFilterExpressionNode:
		return n.TransformXMLFilterExpression(t)
	case *st.XMLStepExpressionNode:
		return n.TransformXMLStepExpression(t)
	case *st.XMLNamePatternChainingNode:
		return n.TransformXMLNamePatternChaining(t)
	case *st.XMLStepIndexedExtendNode:
		return n.TransformXMLStepIndexedExtend(t)
	case *st.XMLStepMethodCallExtendNode:
		return n.TransformXMLStepMethodCallExtend(t)
	case *st.XMLAtomicNamePatternNode:
		return n.TransformXMLAtomicNamePattern(t)
	case *st.TypeReferenceTypeDescNode:
		return n.TransformTypeReferenceTypeDesc(t)
	case *st.MatchStatementNode:
		return n.TransformMatchStatement(t)
	case *st.MatchClauseNode:
		return n.TransformMatchClause(t)
	case *st.MatchGuardNode:
		return n.TransformMatchGuard(t)
	case *st.DistinctTypeDescriptorNode:
		return n.TransformDistinctTypeDescriptor(t)
	case *st.ListMatchPatternNode:
		return n.TransformListMatchPattern(t)
	case *st.RestMatchPatternNode:
		return n.TransformRestMatchPattern(t)
	case *st.MappingMatchPatternNode:
		return n.TransformMappingMatchPattern(t)
	case *st.FieldMatchPatternNode:
		return n.TransformFieldMatchPattern(t)
	case *st.ErrorMatchPatternNode:
		return n.TransformErrorMatchPattern(t)
	case *st.NamedArgMatchPatternNode:
		return n.TransformNamedArgMatchPattern(t)
	case *st.OrderByClauseNode:
		return n.TransformOrderByClause(t)
	case *st.OrderKeyNode:
		return n.TransformOrderKey(t)
	case *st.GroupByClauseNode:
		return n.TransformGroupByClause(t)
	case *st.GroupingKeyVarDeclarationNode:
		return n.TransformGroupingKeyVarDeclaration(t)
	case *st.OnFailClauseNode:
		return n.TransformOnFailClause(t)
	case *st.DoStatementNode:
		return n.TransformDoStatement(t)
	case *st.ClassDefinitionNode:
		return n.TransformClassDefinition(t)
	case *st.ResourcePathParameterNode:
		return n.TransformResourcePathParameter(t)
	case *st.RequiredExpressionNode:
		return n.TransformRequiredExpression(t)
	case *st.ErrorConstructorExpressionNode:
		return n.TransformErrorConstructorExpression(t)
	case *st.ParameterizedTypeDescriptorNode:
		return n.TransformParameterizedTypeDescriptor(t)
	case *st.SpreadMemberNode:
		return n.TransformSpreadMember(t)
	case *st.ClientResourceAccessActionNode:
		return n.TransformClientResourceAccessAction(t)
	case *st.ComputedResourceAccessSegmentNode:
		return n.TransformComputedResourceAccessSegment(t)
	case *st.ResourceAccessRestSegmentNode:
		return n.TransformResourceAccessRestSegment(t)
	case *st.ReSequenceNode:
		return n.TransformReSequence(t)
	case *st.ReAtomQuantifierNode:
		return n.TransformReAtomQuantifier(t)
	case *st.ReAtomCharOrEscapeNode:
		return n.TransformReAtomCharOrEscape(t)
	case *st.ReQuoteEscapeNode:
		return n.TransformReQuoteEscape(t)
	case *st.ReSimpleCharClassEscapeNode:
		return n.TransformReSimpleCharClassEscape(t)
	case *st.ReUnicodePropertyEscapeNode:
		return n.TransformReUnicodePropertyEscape(t)
	case *st.ReUnicodeScriptNode:
		return n.TransformReUnicodeScript(t)
	case *st.ReUnicodeGeneralCategoryNode:
		return n.TransformReUnicodeGeneralCategory(t)
	case *st.ReCharacterClassNode:
		return n.TransformReCharacterClass(t)
	case *st.ReCharSetRangeWithReCharSetNode:
		return n.TransformReCharSetRangeWithReCharSet(t)
	case *st.ReCharSetRangeNode:
		return n.TransformReCharSetRange(t)
	case *st.ReCharSetAtomWithReCharSetNoDashNode:
		return n.TransformReCharSetAtomWithReCharSetNoDash(t)
	case *st.ReCharSetRangeNoDashWithReCharSetNode:
		return n.TransformReCharSetRangeNoDashWithReCharSet(t)
	case *st.ReCharSetRangeNoDashNode:
		return n.TransformReCharSetRangeNoDash(t)
	case *st.ReCharSetAtomNoDashWithReCharSetNoDashNode:
		return n.TransformReCharSetAtomNoDashWithReCharSetNoDash(t)
	case *st.ReCapturingGroupsNode:
		return n.TransformReCapturingGroups(t)
	case *st.ReFlagExpressionNode:
		return n.TransformReFlagExpression(t)
	case *st.ReFlagsOnOffNode:
		return n.TransformReFlagsOnOff(t)
	case *st.ReFlagsNode:
		return n.TransformReFlags(t)
	case *st.ReAssertionNode:
		return n.TransformReAssertion(t)
	case *st.ReQuantifierNode:
		return n.TransformReQuantifier(t)
	case *st.ReBracedQuantifierNode:
		return n.TransformReBracedQuantifier(t)
	case *st.MemberTypeDescriptorNode:
		return n.TransformMemberTypeDescriptor(t)
	case *st.ReceiveFieldNode:
		return n.TransformReceiveField(t)
	case *st.NaturalExpressionNode:
		return n.TransformNaturalExpression(t)
	case *st.IdentifierToken:
		return n.TransformIdentifierToken(t)
	case st.Token:
		return n.TransformToken(t)
	default:
		panic("TransformSyntaxNode: unsupported node type")
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

func (n *NodeBuilder) getPosition(node st.Node) diagnostics.Location {
	textRange := node.TextRange()
	if n.mode == NodeBuilderModeRecover {
		textRange = node.TextRangeWithMinutiae()
	}
	return n.location(node, textRange)
}

func (n *NodeBuilder) getRecoveryPosition(node st.Node) diagnostics.Location {
	return n.location(node, node.TextRangeWithMinutiae())
}

func (n *NodeBuilder) location(node st.Node, textRange st.TextRange) diagnostics.Location {
	return diagnostics.NewLocation(n.de(), getFileName(node), textRange.StartOffset, textRange.EndOffset)
}

func (n *NodeBuilder) getPositionRange(startNode st.Node, endNode st.Node) diagnostics.Location {
	startRange := startNode.TextRange()
	endRange := endNode.TextRange()
	return diagnostics.NewLocation(n.de(), getFileName(startNode), startRange.StartOffset, endRange.EndOffset)
}

func (n *NodeBuilder) getPositionWithoutMetadata(node st.Node) diagnostics.Location {
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

func (n *NodeBuilder) populateMetadata(metadata *st.MetadataNode, target AnnotatableNode) {
	if metadata == nil || metadata.IsMissing() {
		return
	}
	if docTarget, ok := target.(DocumentableNode); ok {
		docString := getDocumentationString(metadata)
		if docString != nil && !docString.IsMissing() {
			docTarget.SetMarkdownDocumentationAttachment(n.createMarkdownDocumentationAttachment(docString))
		}
	}
	n.addAnnotationAttachments(metadata.Annotations(), target)
}

func (n *NodeBuilder) addAnnotationAttachments(annotations st.NodeList[*st.AnnotationNode], target AnnotatableNode) {
	for annotation := range annotations.Iterator() {
		target.AddAnnotationAttachment(n.TransformAnnotation(annotation).(*BLangAnnotationAttachment))
	}
}

func (n *NodeBuilder) createTrueLiteral(pos diagnostics.Location) *BLangLiteral {
	literal := &BLangLiteral{}
	literal.SetValueType(n.types.booleanType)
	literal.SetValue(true)
	literal.SetOriginalValue("true")
	literal.SetPosition(pos)
	return literal
}

// createMarkdownDocumentationAttachment creates a BLangMarkdownDocumentation from a documentation string node
func (n *NodeBuilder) createMarkdownDocumentationAttachment(docStringNode st.Node) *BLangMarkdownDocumentation {
	if docStringNode == nil || docStringNode.IsMissing() {
		return nil
	}

	markdownDocumentationNode, ok := docStringNode.(*st.MarkdownDocumentationNode)
	if !ok {
		return nil
	}

	doc := &BLangMarkdownDocumentation{}
	documentationLines := []BLangMarkdownDocumentationLine{}
	parameters := []BLangMarkdownParameterDocumentation{}
	references := []BLangMarkdownReferenceDocumentation{}

	docLineList := markdownDocumentationNode.DocumentationLines()

	var bLangParaDoc *BLangMarkdownParameterDocumentation
	var bLangReturnParaDoc *BLangMarkdownReturnParameterDocumentation
	var bLangDeprecationDoc *BLangMarkDownDeprecationDocumentation
	var bLangDeprecatedParaDoc *BLangMarkDownDeprecatedParametersDocumentation

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
				bLangDocLine := BLangMarkdownDocumentationLine{}
				bLangDocLine.Text = docText
				bLangDocLine.pos = n.getPosition(docLineNode)
				documentationLines = append(documentationLines, bLangDocLine)
			}
		case st.MARKDOWN_PARAMETER_DOCUMENTATION_LINE:
			if bLangParaDoc != nil {
				if bLangDeprecatedParaDoc != nil {
					bLangDeprecatedParaDoc.Parameters = append(bLangDeprecatedParaDoc.Parameters, *bLangParaDoc)
				} else if bLangDeprecationDoc != nil {
					bLangDeprecatedParaDoc = &BLangMarkDownDeprecatedParametersDocumentation{}
					bLangDeprecatedParaDoc.Parameters = append(bLangDeprecatedParaDoc.Parameters, *bLangParaDoc)
					bLangDeprecationDoc = nil
				} else {
					parameters = append(parameters, *bLangParaDoc)
				}
			}

			bLangParaDoc = &BLangMarkdownParameterDocumentation{}
			parameterDocLineNode := singleDocLine.(*st.MarkdownParameterDocumentationLineNode)

			paraName := &BLangIdentifier{}
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
			bLangParaDoc.pos = n.getPosition(parameterName)
		case st.MARKDOWN_RETURN_PARAMETER_DOCUMENTATION_LINE:
			bLangReturnParaDoc = &BLangMarkdownReturnParameterDocumentation{}
			returnParaDocLineNode := singleDocLine.(*st.MarkdownParameterDocumentationLineNode)

			returnParaDocElements := returnParaDocLineNode.DocumentElements()
			returnParaDocText := n.addReferencesAndReturnDocumentationText(&references, returnParaDocElements)

			bLangReturnParaDoc.ReturnParameterDocumentationLines = append(bLangReturnParaDoc.ReturnParameterDocumentationLines, returnParaDocText)
			bLangReturnParaDoc.pos = n.getPosition(returnParaDocLineNode)
			doc.ReturnParameter = bLangReturnParaDoc
		case st.MARKDOWN_DEPRECATION_DOCUMENTATION_LINE:
			bLangDeprecationDoc = &BLangMarkDownDeprecationDocumentation{}
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
			bLangDeprecationDoc.pos = n.getPosition(deprecationDocLineNode)
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
			bLangDeprecatedParaDoc = &BLangMarkDownDeprecatedParametersDocumentation{}
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
	doc.pos = n.getPosition(markdownDocumentationNode)
	return doc
}

func createIdentifier(pos diagnostics.Location, value, originalValue *string) BLangIdentifier {
	bLIdentifer := BLangIdentifier{}
	bLIdentifer.pos = pos
	if value == nil {
		return bLIdentifer
	}
	identifierValue, isLiteral := normalizedIdentifierValue(*value)
	bLIdentifer.SetValue(identifierValue)
	bLIdentifer.SetLiteral(isLiteral)
	bLIdentifer.SetOriginalValue(*originalValue)
	return bLIdentifer
}

func normalizedIdentifierValue(value string) (string, bool) {
	const IDENTIFIER_LITERAL_PREFIX = "'"
	if len(value) > 0 && value[0:1] == IDENTIFIER_LITERAL_PREFIX {
		return value[1:], true
	}
	return value, false
}

// createIdentifierFromToken creates an identifier from a token, handling missing tokens and validation
func createIdentifierFromToken(pos diagnostics.Location, token st.Token) BLangIdentifier {
	return createIdentifierFromTokenInternal(pos, token, false)
}

// createIdentifierFromTokenInternal creates an identifier from a token with XML handling option
func createIdentifierFromTokenInternal(pos diagnostics.Location, token st.Token, isXML bool) BLangIdentifier {
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

func (n *NodeBuilder) createIgnoreIdentifier(node st.Node) BLangIdentifier {
	pos := n.getPosition(node)
	ignoreValue := string(model.IGNORE)
	identifier := createIdentifier(pos, &ignoreValue, &ignoreValue)
	return identifier
}

// getNextAnonymousTypeKey generates the next anonymous type key
// Placeholder function - to be implemented
func (n *NodeBuilder) getNextAnonymousTypeKey(packageID *model.PackageID, suffixes []string) string {
	return n.cx.GetNextAnonymousTypeKey(packageID)
}

// createTypeNode creates a type node from a syntax tree node
// This delegates to the appropriate Transform method based on the node type
func (n *NodeBuilder) createTypeNode(typeNode st.Node) TypeDescriptor {
	result, err := n.createTypeNodeInner(typeNode)
	if err == nil {
		return result
	}
	if n.mode == NodeBuilderModeRecover {
		return n.badTypeNode(typeNode)
	}
	panic(err)
}

func (n *NodeBuilder) createTypeNodeInner(typeNode st.Node) (TypeDescriptor, error) {
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
		bLUserDefinedType := BLangUserDefinedType{}
		nameRefence := n.createBLangNameReference(typeNode)
		pkgAlias, pkgOK := nameRefence[0].(*BLangIdentifier)
		typeName, nameOK := nameRefence[1].(*BLangIdentifier)
		if !pkgOK || !nameOK {
			return nil, fmt.Errorf("invalid user-defined type name")
		}
		bLUserDefinedType.PkgAlias = *pkgAlias
		bLUserDefinedType.TypeName = *typeName
		bLUserDefinedType.pos = n.getPosition(typeNode)
		return &bLUserDefinedType, nil
	case st.SIMPLE_NAME_REFERENCE:
		nameReferenceNode := typeNode.(*st.SimpleNameReferenceNode)
		return n.createTypeNodeInner(nameReferenceNode.Name())
	default:
		result, ok := n.TransformSyntaxNode(typeNode).(BType)
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

func (n *NodeBuilder) createSimpleVarInner(name st.Token, typeName st.Node, initializer st.Node, visibilityQualifier st.Token, annotations st.NodeList[*st.AnnotationNode]) *BLangSimpleVariable {
	bLSimpleVar := createSimpleVariableNode()

	var namePos diagnostics.Location
	if name != nil {
		namePos = n.getPosition(name)
	}
	identifier := n.createIdentifierNodeFromToken(namePos, name)
	bLSimpleVar.SetName(identifier)

	if isDeclaredWithVar(typeName) {
		bLSimpleVar.IsDeclaredWithVar = true
	} else {
		bLSimpleVar.SetTypeNode(n.createTypeNode(typeName).(BType))
	}

	if visibilityQualifier != nil {
		if visibilityQualifier.Kind() == st.PRIVATE_KEYWORD {
			bLSimpleVar.SetPrivate()
		} else if visibilityQualifier.Kind() == st.PUBLIC_KEYWORD {
			bLSimpleVar.SetPublic()
		}
	}

	if initializer != nil {
		bLSimpleVar.SetInitialExpression(n.createExpression(initializer))
	}

	n.addAnnotationAttachments(annotations, bLSimpleVar)

	return bLSimpleVar
}

func (n *NodeBuilder) createBuiltInTypeNode(typeNode st.Node) TypeDescriptor {
	var typeText string
	if typeNode.Kind() == st.NIL_TYPE_DESC {
		typeText = "()"
	} else if simpleNameRef, ok := typeNode.(*st.BuiltinSimpleNameReferenceNode); ok {
		if simpleNameRef.Kind() == st.VAR_TYPE_DESC {
			return nil
		} else if simpleNameRef.Name().IsMissing() {
			name := getNextMissingNodeName(n.PackageID)
			identifier := createIdentifier(n.getPosition(simpleNameRef.Name()), &name, &name)
			pkgAlias := BLangIdentifier{}
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
		valueType := BLangValueType{}
		valueType.TypeKind = typeKind
		valueType.pos = n.getPosition(typeNode)
		return &valueType
	default:
		builtInValueType := BLangBuiltInRefTypeNode{}
		builtInValueType.TypeKind = typeKind
		builtInValueType.pos = n.getPosition(typeNode)
		return &builtInValueType
	}
}

type mutableIdentifier interface {
	IdentifierNode
	SetValue(string)
}

func setIdentifierValue(identifier IdentifierNode, value string) {
	if identifier, ok := any(identifier).(mutableIdentifier); ok {
		identifier.SetValue(value)
	}
	// We ignore immuatable identifiers such as BadIdentifier (not sure if this can be called for them)
}

func (n *NodeBuilder) createIdentifierNodeFromToken(pos diagnostics.Location, token st.Token) IdentifierNode {
	if token == nil {
		if n.mode == NodeBuilderModeRecover {
			return n.badIdentifier(token)
		}
		panic("missing identifier token")
	}
	if token.IsMissing() || isUnsupportedIdentifierToken(token) {
		if n.mode == NodeBuilderModeRecover {
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

func (n *NodeBuilder) createBLangNameReference(node st.Node) [2]IdentifierNode {
	switch node.Kind() {
	case st.QUALIFIED_NAME_REFERENCE:
		iNode := node.(*st.QualifiedNameReferenceNode)
		modulePrefix := iNode.ModulePrefix()
		identifier := iNode.Identifier()
		pkgAlias := n.createIdentifierNodeFromToken(n.getPosition(modulePrefix), modulePrefix)
		namePos := n.getPosition(identifier)
		name := n.createIdentifierNodeFromToken(namePos, identifier)
		return [...]IdentifierNode{pkgAlias, name}
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
	return [...]IdentifierNode{&pkgAlias, name}
}

// isFunctionCallAsync checks if a function call expression is async
func (n *NodeBuilder) isFunctionCallAsync(functionCallBLangExpression *st.FunctionCallExpressionNode) bool {
	parent := functionCallBLangExpression.Parent()
	if parent == nil {
		panic("isFunctionCallAsync: parent is nil")
	}
	return parent.Kind() == st.START_ACTION
}

// createBLangInvocation creates a BLangInvocation from a name node and arguments
func (n *NodeBuilder) createBLangInvocation(nameNode st.Node, arguments st.NodeList[st.FunctionArgumentNode], position diagnostics.Location, isAsync bool) *BLangInvocation {
	var bLInvocation BLangInvocation
	if isAsync {
		panic("unimplemented")
	} else {
		bLInvocation = BLangInvocation{}
	}

	nameReference := n.createBLangNameReference(nameNode)
	bLInvocation.PkgAlias = nameReference[0]
	bLInvocation.Name = nameReference[1]

	var args []BLangExpression
	for arg := range arguments.Iterator() {
		args = append(args, n.createExpression(arg))
	}
	bLInvocation.ArgExprs = args
	bLInvocation.pos = position
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
func (n *NodeBuilder) createSimpleLiteral(literal st.Node) LiteralNode {
	return n.createSimpleLiteralInner(literal)
}

// getIntegerLiteral parses integer literals (decimal/hex)
func (n *NodeBuilder) getIntegerLiteral(literal st.Node, textValue string) any {
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
func (n *NodeBuilder) createSimpleLiteralInner(literal st.Node) LiteralNode {
	var bLiteral LiteralNode
	kind := literal.Kind()
	var typeTag TypeTags = -1
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
			typeTag = TypeTags_INT
			value = n.getIntegerLiteral(literal, textValue)
			originalValue = &textValue
			// TODO: can we fix below?
			if literalTokenKind == st.HEX_INTEGER_LITERAL_TOKEN && withinByteRange(value) {
				typeTag = TypeTags_BYTE
			}
		case st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			// TODO: Check effect of mapping negative(-) numbers as unary-expr
			if balCommon.IsDecimalDiscriminated(textValue) {
				typeTag = TypeTags_DECIMAL
			} else {
				typeTag = TypeTags_FLOAT
			}
			value = textValue
			originalValue = &textValue
		default:
			// TODO: Check effect of mapping negative(-) numbers as unary-expr
			typeTag = TypeTags_FLOAT
			value = getHexNodeValue(textValue)
			originalValue = &textValue
		}
		numericLiteral := &BLangNumericLiteral{}
		numericLiteral.pos = n.getPosition(literal)
		numericLiteral.SetValueType(n.types.getTypeFromTag(typeTag).(BType))
		numericLiteral.Value = value
		numericLiteral.OriginalValue = *originalValue
		return &numericLiteral.BLangLiteral
	} else if kind == st.BOOLEAN_LITERAL {
		typeTag = TypeTags_BOOLEAN
		value = strings.ToLower(textValue) == "true"
		originalValue = &textValue
		bLiteral = &BLangLiteral{}
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

		typeTag = TypeTags_STRING
		value = text
		originalValue = &textValue
		bLiteral = &BLangLiteral{}
	} else if kind == st.NIL_LITERAL {
		typeTag = TypeTags_NIL
		value = nil
		originalValue = new(string(model.NIL_VALUE))
		bLiteral = &BLangLiteral{}
	} else if kind == st.NULL_LITERAL {
		originalValue = new("null")
		typeTag = TypeTags_NIL
		bLiteral = &BLangLiteral{}
	} else if kind == st.BINARY_EXPRESSION { // Should be base16 and base64
		typeTag = TypeTags_BYTE_ARRAY
		value = textValue
		originalValue = &textValue

		// If numeric literal create a numeric literal expression; otherwise create a literal expression
		if isNumericLiteral(kind) {
			bLiteral = &BLangNumericLiteral{}
		} else {
			bLiteral = &BLangLiteral{}
		}
	} else if kind == st.BYTE_ARRAY_LITERAL {
		return n.TransformSyntaxNode(literal).(LiteralNode)
	}
	bLangNode := bLiteral.(BLangNode)
	bLangNode.SetPosition(n.getPosition(literal))
	bType := n.types.getTypeFromTag(typeTag).(BType)
	bType.BTypeSetTag(typeTag)
	switch bl := bLiteral.(type) {
	case *BLangLiteral:
		bl.SetValueType(bType)
	case *BLangNumericLiteral:
		bl.SetValueType(bType)
	}
	bLiteral.SetValue(value)
	bLiteral.SetOriginalValue(*originalValue)
	return bLiteral
}

func (n *NodeBuilder) TransformModulePart(modulePartNode *st.ModulePart) BLangNode {
	compilationUnit := BLangCompilationUnit{}
	n.currentCompUnit = &compilationUnit
	defer func() { n.currentCompUnit = nil }()
	compilationUnit.packageID = n.PackageID
	pos := n.getPosition(modulePartNode)

	if modulePartNode.HasDiagnostics() {
		n.syntaxError(modulePartNode)
	}

	// Generate import declarations
	imports := modulePartNode.Imports()
	for importDecl := range imports.Iterator() {
		if importDecl.HasDiagnostics() {
			if n.mode == NodeBuilderModeRecover {
				compilationUnit.AddTopLevelNode(n.badTopLevel(importDecl))
			}
			continue
		}
		node, err := n.transformImportTopLevel(importDecl)
		if err != nil {
			if n.mode == NodeBuilderModeRecover {
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
		// Dispatch to TransformSyntaxNode which handles all node types
		var memberNode st.Node = member
		if memberNode.HasDiagnostics() {
			if n.mode != NodeBuilderModeRecover {
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
	compilationUnit.pos = newLocation
	compilationUnit.packageID = n.PackageID

	return &compilationUnit
}

func setFunctionQualifiers(bLFunction *BLangFunction, qualifierList st.NodeList[st.Token]) {
	setFunctionQualifiersOnBase(&bLFunction.bLangInvokableNodeBase, qualifierList)
}

func setFunctionQualifiersOnBase(base *bLangInvokableNodeBase, qualifierList st.NodeList[st.Token]) {
	for qualifier := range qualifierList.Iterator() {
		kind := qualifier.Kind()

		switch kind {
		case st.PUBLIC_KEYWORD:
			base.SetPublic()
		case st.PRIVATE_KEYWORD:
			// private is the default
		case st.REMOTE_KEYWORD:
			base.SetRemote()
		case st.TRANSACTIONAL_KEYWORD:
			base.SetTransactional()
		case st.RESOURCE_KEYWORD:
			base.SetResource()
		case st.ISOLATED_KEYWORD:
			base.SetIsolated()
		default:
			// Skip unknown qualifiers
			continue
		}
	}
}

func (n *NodeBuilder) populateFuncSignature(bLFunction *BLangFunction, funcSignature *st.FunctionSignatureNode) {
	n.populateFuncSignatureOnBase(&bLFunction.bLangInvokableNodeBase, funcSignature)
}

func (n *NodeBuilder) populateFuncSignatureOnBase(bLFunction *bLangInvokableNodeBase, funcSignature *st.FunctionSignatureNode) {
	bLFunction.ParamListPos = diagnostics.NewBuiltinLocation()
	openParen := funcSignature.OpenParenToken()
	closeParen := funcSignature.CloseParenToken()
	if openParen != nil && closeParen != nil && !openParen.IsMissing() && !closeParen.IsMissing() {
		bLFunction.ParamListPos = n.getPositionRange(openParen, closeParen)
	}

	// Set Parameters
	parameters := funcSignature.Parameters()
	for param := range parameters.Iterator() {
		// Transform parameter using TransformSyntaxNode
		paramNode := n.TransformSyntaxNode(param).(SimpleVariableNode)

		// Special handling for rest parameters
		if _, isRestParam := param.(*st.RestParameterNode); isRestParam {
			bLFunction.SetRestParameter(paramNode)
			continue
		}

		// Add to parameters list (all non-rest parameters)
		bLFunction.AddParameter(paramNode)
	}

	// Set Return Type
	retTypeDescNode := funcSignature.ReturnTypeDesc()
	if retTypeDescNode != nil {
		returnsKeyword := retTypeDescNode.ReturnsKeyword()
		if returnsKeyword != nil && !returnsKeyword.IsMissing() {
			bLFunction.SetExplicitReturnTypeDescriptor()
		}

		// Get the type child from the return type descriptor
		typeNode := retTypeDescNode.Type()

		// Push "return" onto the anonymous type name suffixes stack
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, "return")

		// Create the type node from the type child
		bLFunction.SetReturnTypeDescriptor(n.createTypeNode(typeNode))

		// Pop "return" from the stack
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
		annots := retTypeDescNode.Annotations()
		n.addAnnotationAttachments(annots, bLFunction.ReturnTypeDescriptorNode())
	} else {
		// Default return type is nil when not specified
		nilReturnType := &BLangValueType{TypeKind: TypeKind_NIL}
		nilReturnType.pos = diagnostics.NewBuiltinLocation()
		bLFunction.SetReturnTypeDescriptor(nilReturnType)
	}
}

func (n *NodeBuilder) TransformFunctionDefinition(funcDefNode *st.FunctionDefinition) BLangNode {
	// Check for resource functions - panic for now
	relativeResourcePath := funcDefNode.RelativeResourcePath()
	hasResourcePath := relativeResourcePath.Size() > 0
	if hasResourcePath {
		panic("TransformFunctionDefinition: resource functions not yet supported")
	}

	// Create function node
	bLFunction := n.createFunctionNode(funcDefNode.FunctionName(), funcDefNode.QualifierList(), funcDefNode.FunctionSignature(), funcDefNode.FunctionBody())
	bLFunction.pos = n.getPositionWithoutMetadata(funcDefNode)

	metadata := funcDefNode.Metadata()
	n.populateMetadata(metadata, bLFunction)

	return bLFunction
}

func (n *NodeBuilder) createFunctionNode(funcName *st.IdentifierToken, qualifierList st.NodeList[st.Token], funcSignature *st.FunctionSignatureNode, funcBody st.FunctionBodyNode) *BLangFunction {
	blFunction := BLangFunction{}
	name := n.createIdentifierNodeFromToken(n.getPosition(funcName), funcName)
	n.populateFunctionNode(name, qualifierList, funcSignature, funcBody, &blFunction)
	return &blFunction
}

func (n *NodeBuilder) populateFunctionNode(name IdentifierNode, qualifierList st.NodeList[st.Token], funcSignature *st.FunctionSignatureNode, funcBody st.FunctionBodyNode, blFunction *BLangFunction) {
	// Set function name
	blFunction.Name = name
	// Set method qualifiers
	setFunctionQualifiers(blFunction, qualifierList)
	// Set function signature
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, name.GetValue())
	defer func() {
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	}()
	n.populateFuncSignature(blFunction, funcSignature)

	// Set the function body
	if funcBody == nil {
		blFunction.Body = nil
		blFunction.SetInterface()
	} else {
		body := n.TransformSyntaxNode(funcBody).(FunctionBodyNode)
		blFunction.Body = body
		if _, ok := body.(*BLangExternFunctionBody); ok {
			blFunction.SetNative()
		}
	}
}

func (n *NodeBuilder) transformImportTopLevel(importDecl *st.ImportDeclarationNode) (TopLevelNode, error) {
	transformedNode := n.TransformImportDeclaration(importDecl)
	bLangImport, ok := transformedNode.(*BLangImportPackage)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-import node %T", importDecl, transformedNode)
	}
	return bLangImport, nil
}

func (n *NodeBuilder) transformTopLevel(node st.Node) (TopLevelNode, error) {
	result, err := n.transformTopLevelInner(node)
	if err == nil {
		return result, nil
	}
	if n.mode == NodeBuilderModeRecover {
		return n.badTopLevel(node), nil
	}
	return nil, err
}

func (n *NodeBuilder) transformTopLevelInner(node st.Node) (TopLevelNode, error) {
	transformedNode := n.TransformSyntaxNode(node)
	topLevel, ok := transformedNode.(TopLevelNode)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-top-level node %T", node, transformedNode)
	}
	return topLevel, nil
}

func (n *NodeBuilder) TransformImportDeclaration(importDeclarationNode *st.ImportDeclarationNode) BLangNode {
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
	var pkgNameComps []BLangIdentifier
	moduleNames := importDeclarationNode.ModuleName()
	for name := range moduleNames.Iterator() {
		namePos := n.getPosition(name)
		nameText := name.Text()
		identifier := createIdentifier(namePos, &nameText, &nameText)
		pkgNameComps = append(pkgNameComps, identifier)
	}

	// 5. Create BLangImportPackage node
	importDcl := &BLangImportPackage{}
	importDcl.pos = position
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

func (n *NodeBuilder) TransformListenerDeclaration(listenerDeclarationNode *st.ListenerDeclarationNode) BLangNode {
	metadata := listenerDeclarationNode.Metadata()

	pos := n.getPositionWithoutMetadata(listenerDeclarationNode)
	nameToken := listenerDeclarationNode.VariableName()
	namePos := n.getPosition(nameToken)
	identifier := createIdentifierFromToken(namePos, nameToken)

	bLSimpleVar := createSimpleVariableNode()
	bLSimpleVar.SetName(&identifier)
	bLSimpleVar.pos = pos

	typeDesc := listenerDeclarationNode.TypeDescriptor()
	if typeDesc != nil && !typeDesc.IsMissing() {
		bLSimpleVar.SetTypeNode(n.createTypeNode(typeDesc).(BType))
	} else {
		bLSimpleVar.IsDeclaredWithVar = true
	}

	if initializer := listenerDeclarationNode.Initializer(); initializer != nil {
		bLSimpleVar.SetInitialExpression(n.createExpression(initializer))
	}

	if visQual := listenerDeclarationNode.VisibilityQualifier(); visQual != nil && visQual.Kind() == st.PUBLIC_KEYWORD {
		bLSimpleVar.SetPublic()
	}

	if metadata != nil && !metadata.IsMissing() {
		if annotations := metadata.Annotations(); annotations.Size() > 0 {
			panic("TransformListenerDeclaration: annotations not yet supported")
		}
		bLSimpleVar.MarkdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}

	// Listeners are final (the binding cannot be reassigned).
	bLSimpleVar.SetFinal()
	bLSimpleVar.SetListener()
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

func (n *NodeBuilder) TransformTypeDefinition(typeDefinitionNode *st.TypeDefinitionNode) BLangNode {
	typeDef := NewBLangTypeDefinition()

	identifierNode := createIdentifierFromToken(n.getPosition(typeDefinitionNode.TypeName()), typeDefinitionNode.TypeName())
	typeDef.Name = &identifierNode

	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, typeDef.Name.GetValue())

	typeDescriptorNode := typeDefinitionNode.TypeDescriptor()
	if distinctTypeDescriptorNode, ok := typeDescriptorNode.(*st.DistinctTypeDescriptorNode); ok {
		innerTypeDescriptorNode := distinctTypeDescriptorNode.TypeDescriptor()
		if innerTypeDescriptorNode == nil || !isAllowedDistinctTypeDescriptor(innerTypeDescriptorNode.Kind()) {
			n.cx.SyntaxError("only object and error types can be distinct", n.getPosition(distinctTypeDescriptorNode))
			neverType := &BLangValueType{TypeKind: TypeKind_NEVER}
			neverType.pos = n.getPosition(distinctTypeDescriptorNode)
			typeDef.SetTypeData(TypeData{TypeDescriptor: neverType})
			n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
			return typeDef
		}
		typeDescriptorNode = innerTypeDescriptorNode
		typeDef.SetDistinct()
	}
	typeData := TypeData{
		TypeDescriptor: n.createTypeNode(typeDescriptorNode),
	}
	typeDef.SetTypeData(typeData)

	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]

	visibilityQualifier := typeDefinitionNode.VisibilityQualifier()
	if visibilityQualifier != nil && visibilityQualifier.Kind() == st.PUBLIC_KEYWORD {
		typeDef.SetPublic()
	}

	typeDef.pos = n.getPositionWithoutMetadata(typeDefinitionNode)

	n.populateMetadata(typeDefinitionNode.Metadata(), typeDef)

	return typeDef
}

func (n *NodeBuilder) TransformServiceDeclaration(serviceDeclarationNode *st.ServiceDeclarationNode) BLangNode {
	metadata := serviceDeclarationNode.Metadata()

	service := NewBLangService()
	service.pos = n.getPositionWithoutMetadata(serviceDeclarationNode)

	if metadata != nil && !metadata.IsMissing() {
		if annotations := metadata.Annotations(); annotations.Size() > 0 {
			panic("TransformServiceDeclaration: annotations not yet supported")
		}
		service.MarkdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(getDocumentationString(metadata))
	}

	if typeDesc := serviceDeclarationNode.TypeDescriptor(); typeDesc != nil && !typeDesc.IsMissing() {
		service.SetTypeData(TypeData{TypeDescriptor: n.createTypeNode(typeDesc)})
	}

	n.populateServiceQualifiers(&service, serviceDeclarationNode)
	n.populateServiceAttachPoint(&service, serviceDeclarationNode)
	n.populateServiceAttachedExprs(&service, serviceDeclarationNode)

	members := n.collectClassDefnMembers(serviceDeclarationNode.Members())
	service.Fields = members.Fields
	service.Methods = members.Methods
	service.InitFunction = members.InitFunction
	service.ResourceMethods = members.ResourceMethods
	for _, each := range members.UnresolvedInclusions {
		// Parser should catch these
		n.cx.InternalError("unexpected inclusions in service decl", each.pos)
	}

	return &service
}

// populateServiceQualifiers reads the user-controllable qualifiers from the
// service declaration. The `service` flag is already set by NewBLangService.
func (n *NodeBuilder) populateServiceQualifiers(service *BLangService, node *st.ServiceDeclarationNode) {
	quals := node.Qualifiers()
	for qual := range quals.Iterator() {
		if qual.Kind() == st.ISOLATED_KEYWORD {
			service.SetIsolated()
		}
	}
}

func (n *NodeBuilder) populateServiceAttachPoint(service *BLangService, node *st.ServiceDeclarationNode) {
	paths := node.AbsoluteResourcePath()
	if node.HasDiagnostics() {
		return
	}
	if paths.Size() > 0 {
		service.AbsoluteResourcePath = []BLangIdentifier{}
	}
	for i := 0; i < paths.Size(); i++ {
		seg := paths.Get(i)
		if seg.Kind() == st.STRING_LITERAL {
			service.AttachPointLiteral = n.createSimpleLiteral(seg).(*BLangLiteral) //nolint:forcetypeassert // string literals always create BLangLiteral nodes
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

func (n *NodeBuilder) populateServiceAttachedExprs(service *BLangService, node *st.ServiceDeclarationNode) {
	exprs := node.Expressions()
	if exprs.Size() > 0 {
		service.AttachedExprsPosition = n.getPositionRange(exprs.Get(0), exprs.Get(exprs.Size()-1))
	}
	for i := 0; i < exprs.Size(); i += 2 {
		service.AttachedExprs = append(service.AttachedExprs, n.createExpression(exprs.Get(i)))
	}
}

type classDefnMembers struct {
	Fields               []SimpleVariableNode
	Methods              map[string]*BLangFunction
	InitFunction         *BLangFunction
	ResourceMethods      []*BLangResourceMethod
	UnresolvedInclusions []*BLangUserDefinedType
}

func newClassDefnMembers() classDefnMembers {
	return classDefnMembers{Methods: map[string]*BLangFunction{}}
}

func (n *NodeBuilder) collectClassDefnMembers(memberNodes st.NodeList[st.Node]) classDefnMembers {
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
			members.UnresolvedInclusions = append(members.UnresolvedInclusions, n.createTypeNode(typeRef.TypeName()).(*BLangUserDefinedType))
		default:
			panic("collectClassDefnMembers: unsupported member kind")
		}
	}
	return members
}

func (n *NodeBuilder) addCollectedMethod(members *classDefnMembers, funcDef *st.FunctionDefinition) {
	bLFunction := n.createFunctionNode(funcDef.FunctionName(), funcDef.QualifierList(), funcDef.FunctionSignature(), funcDef.FunctionBody())
	bLFunction.pos = n.getPositionWithoutMetadata(funcDef)
	bLFunction.SetAttached()
	n.populateMetadata(funcDef.Metadata(), bLFunction)

	funcName := bLFunction.Name.GetValue()
	if model.Name(funcName) == model.USER_DEFINED_INIT_SUFFIX {
		if members.InitFunction != nil {
			n.cx.SyntaxError("redeclared symbol 'init'", bLFunction.pos)
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
		n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(funcName)+"'", bLFunction.pos)
		return
	}
	members.Methods[funcName] = bLFunction
}

func (n *NodeBuilder) TransformAssignmentStatement(assignmentStatementNode *st.AssignmentStatementNode) BLangNode {
	lhsKind := assignmentStatementNode.VarRef().Kind()
	switch lhsKind {
	case st.LIST_BINDING_PATTERN, st.MAPPING_BINDING_PATTERN, st.ERROR_BINDING_PATTERN:
		panic("unimplemented")
	default:
		break
	}

	bLAssignment := &BLangAssignment{}
	lhsExpr := n.createExpression(assignmentStatementNode.VarRef())
	switch lhsExpr := lhsExpr.(type) {
	case *BLangFieldBaseAccess:
		lhsExpr.SetLexpr()
	case *BLangIndexBasedAccess:
		lhsExpr.SetLexpr()
	}
	bLAssignment.SetActionOrExpression(n.createActionOrExpression(assignmentStatementNode.Expression()))
	bLAssignment.pos = n.getPosition(assignmentStatementNode)
	bLAssignment.VarRef = lhsExpr.(LExpr)
	return bLAssignment
}

func (n *NodeBuilder) TransformCompoundAssignmentStatement(compoundAssignmentStmtNode *st.CompoundAssignmentStatementNode) BLangNode {
	bLCompAssignment := &BLangCompoundAssignment{}
	bLCompAssignment.SetActionOrExpression(n.createActionOrExpression(compoundAssignmentStmtNode.RhsExpression()))
	lhsExpr := n.createExpression(compoundAssignmentStmtNode.LhsExpression())
	switch lhsExpr := lhsExpr.(type) {
	case *BLangFieldBaseAccess:
		lhsExpr.SetLexpr()
		lhsExpr.SetCompoundAssignmentLValue()
	case *BLangIndexBasedAccess:
		lhsExpr.SetLexpr()
		lhsExpr.SetCompoundAssignmentLValue()
	}
	bLCompAssignment.SetVariable(lhsExpr.(LExpr))
	BLangNode(bLCompAssignment).SetPosition(n.getPosition(compoundAssignmentStmtNode))
	bLCompAssignment.OpKind = model.OperatorKindValueFrom(compoundAssignmentStmtNode.BinaryOperator().Text())
	return bLCompAssignment
}

func (n *NodeBuilder) TransformVariableDeclaration(variableDeclarationNode *st.VariableDeclarationNode) BLangNode {
	varNode := n.createBLangVarDef(
		n.getPosition(variableDeclarationNode),
		variableDeclarationNode.TypedBindingPattern(),
		variableDeclarationNode.Initializer(),
		variableDeclarationNode.FinalKeyword(),
	)
	annotations := variableDeclarationNode.Annotations()
	if simpleVarDef, ok := varNode.(*BLangSimpleVariableDef); ok {
		n.addAnnotationAttachments(annotations, simpleVarDef.Var)
	}

	return varNode.(BLangNode)
}

func (n *NodeBuilder) createBLangVarDef(location diagnostics.Location, typedBindingPattern *st.TypedBindingPatternNode, initializer st.ExpressionNode, finalKeyword st.Token) VariableDefinitionNode {
	bindingPattern := typedBindingPattern.BindingPattern()

	variable := n.getBLangVariableNode(bindingPattern, location)

	var qualifiers []st.Token
	if finalKeyword != nil {
		qualifiers = append(qualifiers, finalKeyword) //nolint:staticcheck,ineffassign // qualifierList creation not yet implemented
	}
	// qualifierList := st.CreateNodeListWithFacade(qualifiers)

	switch bindingPattern.Kind() {
	case st.CAPTURE_BINDING_PATTERN, st.WILDCARD_BINDING_PATTERN:
		variable := variable.(*BLangSimpleVariable)
		bLVarDef := &BLangSimpleVariableDef{}

		bLVarDef.pos = location
		variable.SetPosition(location)

		var expr BLangActionOrExpression
		if initializer != nil {
			expr = n.createActionOrExpression(initializer)
		}
		variable.SetInitialExpression(expr)

		bLVarDef.SetVariable(variable)

		if finalKeyword != nil {
			variable.SetFinal()
		}

		typeDesc := typedBindingPattern.TypeDescriptor()
		isDeclaredWithVar := isDeclaredWithVar(typeDesc)
		variable.SetIsDeclaredWithVar(isDeclaredWithVar)
		if !isDeclaredWithVar {
			variable.SetTypeNode(n.createTypeNode(typeDesc).(BType))
		}

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

func (n *NodeBuilder) TransformBlockStatement(blockStatementNode *st.BlockStatementNode) BLangNode {
	bLBlockStmt := BLangBlockStmt{}
	bLBlockStmt.Stmts = n.generateBLangStatements(blockStatementNode.Statements(), blockStatementNode)
	bLBlockStmt.pos = n.getPosition(blockStatementNode)
	return &bLBlockStmt
}

func (n *NodeBuilder) generateBLangStatements(statementNodes st.NodeList[st.StatementNode], endNode st.Node) []StatementNode {
	statements := []StatementNode{}
	return *n.generateAndAddBLangStatements(statementNodes, &statements, 0, endNode)
}

func (n *NodeBuilder) transformStatement(statement st.StatementNode) StatementNode {
	result, err := n.transformStatementInner(statement)
	if err == nil {
		return result
	}
	if n.mode == NodeBuilderModeRecover {
		return n.badStmt(statement)
	}
	panic(err)
}

func (n *NodeBuilder) transformStatementInner(statement st.StatementNode) (StatementNode, error) {
	if statement == nil {
		return nil, fmt.Errorf("statement is nil")
	}
	// TODO: Ideally we should have a switch that handles all possible stmt nodes instead.
	transformedNode := n.TransformSyntaxNode(statement)
	stmt, ok := transformedNode.(StatementNode)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-statement node %T", statement, transformedNode)
	}
	return stmt, nil
}

func (n *NodeBuilder) generateAndAddBLangStatements(statementNodes st.NodeList[st.StatementNode], statements *[]StatementNode, startPosition int, endNode st.Node) *[]StatementNode {
	lastStmtIndex := statementNodes.Size() - 1
	for j := startPosition; j < statementNodes.Size(); j++ {
		currentStatement := statementNodes.Get(j)
		// TODO: Remove this check once statements are non null guaranteed
		if currentStatement == nil {
			continue
		}
		if currentStatement.HasDiagnostics() && n.mode != NodeBuilderModeRecover {
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
				emptyBlock := &BLangBlockStmt{}
				emptyBlock.pos = n.getPositionRange(currentStatement, endNode)
				*statements = append(*statements, emptyBlock)
				break
			}
			bLBlockStmt := &BLangBlockStmt{}
			nextStmtIndex := j + 1
			n.generateAndAddBLangStatements(statementNodes, &bLBlockStmt.Stmts, nextStmtIndex, endNode)
			if nextStmtIndex <= lastStmtIndex {
				bLBlockStmt.pos = n.getPositionRange(statementNodes.Get(nextStmtIndex), endNode)
			}
			*statements = append(*statements, bLBlockStmt)
			break
		} else {
			*statements = append(*statements, n.transformStatement(currentStatement))
		}
	}
	return statements
}

func (n *NodeBuilder) TransformBreakStatement(breakStatementNode *st.BreakStatementNode) BLangNode {
	bLBreak := &BLangBreak{}
	bLBreak.pos = n.getPosition(breakStatementNode)
	return bLBreak
}

func (n *NodeBuilder) TransformFailStatement(failStatementNode *st.FailStatementNode) BLangNode {
	panic("TransformFailStatement unimplemented")
}

func (n *NodeBuilder) TransformExpressionStatement(expressionStatement *st.ExpressionStatementNode) BLangNode {
	bLExpressionStmt := BLangExpressionStmt{}
	bLExpressionStmt.Expr = n.createActionOrExpression(expressionStatement.Expression())
	bLExpressionStmt.pos = n.getPosition(expressionStatement)
	return &bLExpressionStmt
}

// createSpecificFieldNameLiteral builds a string-literal expression for a
// non-computed mapping-constructor key. The field name is a static identifier
// or string literal, not a runtime expression, so it must not be represented
// as a var-ref.
func (n *NodeBuilder) createSpecificFieldNameLiteral(fieldName st.Node) BLangExpression {
	if basicLit, ok := fieldName.(*st.BasicLiteralNode); ok {
		return n.createSimpleLiteral(basicLit).(BLangExpression)
	}
	nameRef := n.createBLangNameReference(fieldName)
	name := nameRef[1].GetValue()
	pos := n.getPosition(fieldName)
	lit := &BLangLiteral{}
	lit.SetPosition(pos)
	bType := &BTypeBasic{}
	bType.BTypeSetTag(TypeTags_STRING)
	lit.SetValueType(bType)
	lit.SetValue(name)
	lit.SetOriginalValue(name)
	return lit
}

func (n *NodeBuilder) createExpression(expressionNode st.Node) BLangExpression {
	result, err := n.createExpressionInner(expressionNode)
	if err == nil {
		return result
	}
	if n.mode == NodeBuilderModeRecover {
		return n.badExprOrAction(expressionNode)
	}
	panic(err)
}

func (n *NodeBuilder) createExpressionInner(expressionNode st.Node) (BLangExpression, error) {
	actionOrExpr, err := n.createActionOrExpressionInner(expressionNode)
	if err != nil {
		return nil, err
	}
	expr, ok := actionOrExpr.(BLangExpression)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-expression node %T", expressionNode, actionOrExpr)
	}
	return expr, nil
}

// createActionOrExpression creates an action or expression node from a syntax tree node
func (n *NodeBuilder) createActionOrExpression(actionOrExpression st.Node) BLangActionOrExpression {
	result, err := n.createActionOrExpressionInner(actionOrExpression)
	if err == nil {
		return result
	}
	if n.mode == NodeBuilderModeRecover {
		return n.badExprOrAction(actionOrExpression)
	}
	panic(err)
}

func (n *NodeBuilder) createActionOrExpressionInner(actionOrExpression st.Node) (BLangActionOrExpression, error) {
	if actionOrExpression == nil {
		return nil, fmt.Errorf("missing action or expression")
	}
	if isSimpleLiteral(actionOrExpression.Kind()) {
		result, ok := n.createSimpleLiteral(actionOrExpression).(BLangActionOrExpression)
		if !ok {
			return nil, fmt.Errorf("syntax node %T transformed to non-action-or-expression node", actionOrExpression)
		}
		return result, nil
	}
	if actionOrExpression.Kind() == st.SIMPLE_NAME_REFERENCE ||
		actionOrExpression.Kind() == st.QUALIFIED_NAME_REFERENCE ||
		actionOrExpression.Kind() == st.IDENTIFIER_TOKEN {
		nameReference := n.createBLangNameReference(actionOrExpression)
		bLVarRef := BLangSimpleVarRef{}
		bLVarRef.pos = n.getPosition(actionOrExpression)
		bLVarRef.PkgAlias = nameReference[0]
		bLVarRef.VariableName = nameReference[1]
		return &bLVarRef, nil
	}
	if actionOrExpression.Kind() == st.BRACED_EXPRESSION {
		bracedExpr := actionOrExpression.(*st.BracedExpressionNode)
		inner, err := n.createActionOrExpressionInner(bracedExpr.Expression())
		if err != nil {
			return nil, err
		}
		if action, ok := inner.(BLangAction); ok {
			return action, nil
		}
		expr, ok := inner.(BLangExpression)
		if !ok {
			return nil, fmt.Errorf("braced syntax node %T transformed to non-expression node %T", actionOrExpression, inner)
		}
		group := BLangGroupExpr{}
		group.Expression = expr
		group.pos = n.getPosition(actionOrExpression)
		return &group, nil
	}
	if isType(actionOrExpression.Kind()) {
		typeAccessExpr := BLangTypedescExpr{}
		typeAccessExpr.pos = n.getPosition(actionOrExpression)
		typeAccessExpr.typeDescriptor = n.createTypeNode(actionOrExpression)
		return &typeAccessExpr, nil
	}
	transformedNode := n.TransformSyntaxNode(actionOrExpression)
	result, ok := transformedNode.(BLangActionOrExpression)
	if !ok {
		return nil, fmt.Errorf("syntax node %T transformed to non-action-or-expression node %T", actionOrExpression, transformedNode)
	}
	return result, nil
}

func (n *NodeBuilder) TransformContinueStatement(continueStatementNode *st.ContinueStatementNode) BLangNode {
	blContinue := &BLangContinue{}
	blContinue.pos = n.getPosition(continueStatementNode)
	return blContinue
}

func (n *NodeBuilder) TransformExternalFunctionBody(externalFunctionBodyNode *st.ExternalFunctionBodyNode) BLangNode {
	body := &BLangExternFunctionBody{}
	body.pos = n.getPosition(externalFunctionBodyNode)
	return body
}

func (n *NodeBuilder) TransformIfElseStatement(ifElseStatementNode *st.IfElseStatementNode) BLangNode {
	bLIf := BLangIf{}
	bLIf.pos = n.getPosition(ifElseStatementNode)
	bLIf.SetCondition(n.createExpression(ifElseStatementNode.Condition()))
	bLIf.SetBody(n.TransformBlockStatement(ifElseStatementNode.IfBody()).(*BLangBlockStmt))
	if ifElseStatementNode.ElseBody() != nil {
		elseNode := ifElseStatementNode.ElseBody().(*st.ElseBlockNode)
		bLIf.SetElseStatement(n.TransformSyntaxNode(elseNode.ElseBody()).(StatementNode))
	}
	return &bLIf
}

func (n *NodeBuilder) TransformElseBlock(elseBlockNode *st.ElseBlockNode) BLangNode {
	panic("TransformElseBlock unimplemented")
}

func (n *NodeBuilder) TransformWhileStatement(whileStatementNode *st.WhileStatementNode) BLangNode {
	bLWhile := &BLangWhile{}
	bLWhile.SetCondition(n.createExpression(whileStatementNode.Condition()))
	bLWhile.pos = n.getPosition(whileStatementNode)

	bLBlockStmt := n.TransformBlockStatement(whileStatementNode.WhileBody()).(*BLangBlockStmt)
	bLBlockStmt.pos = n.getPosition(whileStatementNode.WhileBody())
	bLWhile.SetBody(bLBlockStmt)
	if whileStatementNode.OnFailClause() != nil {
		onFailClauseNode := whileStatementNode.OnFailClause()
		bLWhile.SetOnFailClause(n.TransformOnFailClause(onFailClauseNode).(*BLangOnFailClause))
	} else {
		bLWhile.OnFailClause.pos = diagnostics.NewBuiltinLocation()
	}
	return bLWhile
}

func (n *NodeBuilder) TransformPanicStatement(panicStatementNode *st.PanicStatementNode) BLangNode {
	bLPanic := &BLangPanic{}
	bLPanic.pos = n.getPosition(panicStatementNode)
	bLPanic.Expr = n.createExpression(panicStatementNode.Expression())
	return bLPanic
}

func (n *NodeBuilder) TransformReturnStatement(returnStatementNode *st.ReturnStatementNode) BLangNode {
	bLReturn := &BLangReturn{}
	bLReturn.pos = n.getPosition(returnStatementNode)
	if returnStatementNode.Expression() != nil {
		bLReturn.SetActionOrExpression(n.createActionOrExpression(returnStatementNode.Expression()))
	} else {
		nilLiteral := &BLangLiteral{}
		nilLiteral.pos = n.getPosition(returnStatementNode)
		nilLiteral.Value = nil
		nilLiteral.SetValueType(n.types.getTypeFromTag(TypeTags_NIL).(BType))
		bLReturn.SetActionOrExpression(nilLiteral)
	}

	return bLReturn
}

func (n *NodeBuilder) TransformLocalTypeDefinitionStatement(localTypeDefinitionStatementNode *st.LocalTypeDefinitionStatementNode) BLangNode {
	panic("TransformLocalTypeDefinitionStatement unimplemented")
}

func (n *NodeBuilder) TransformLockStatement(lockStatementNode *st.LockStatementNode) BLangNode {
	if lockStatementNode.OnFailClause() != nil {
		n.cx.Unimplemented("on-fail clause on lock is not yet supported", n.getPosition(lockStatementNode.OnFailClause()))
	}
	bLLock := &BLangLock{}
	bLLock.pos = n.getPosition(lockStatementNode)
	bLBlockStmt := n.TransformBlockStatement(lockStatementNode.BlockStatement()).(*BLangBlockStmt)
	bLBlockStmt.pos = n.getPosition(lockStatementNode.BlockStatement())
	bLLock.Body = *bLBlockStmt
	return bLLock
}

func (n *NodeBuilder) TransformForkStatement(forkStatementNode *st.ForkStatementNode) BLangNode {
	panic("TransformForkStatement unimplemented")
}

func (n *NodeBuilder) TransformForEachStatement(forEachStatementNode *st.ForEachStatementNode) BLangNode {
	bLForeach := &BLangForeach{}
	bLForeach.pos = n.getPosition(forEachStatementNode)

	varDef := n.createBLangVarDef(
		n.getPosition(forEachStatementNode.TypedBindingPattern()),
		forEachStatementNode.TypedBindingPattern(),
		nil,
		nil,
	).(*BLangSimpleVariableDef)
	bLForeach.VariableDef = varDef
	bLForeach.IsDeclaredWithVar = varDef.Var.IsDeclaredWithVar

	bLForeach.Collection = n.createExpression(forEachStatementNode.ActionOrExpressionNode())

	body := n.TransformBlockStatement(forEachStatementNode.BlockStatement()).(*BLangBlockStmt)
	body.pos = n.getPosition(forEachStatementNode.BlockStatement())
	bLForeach.Body = *body

	if forEachStatementNode.OnFailClause() != nil {
		bLForeach.SetOnFailClause(
			n.TransformOnFailClause(forEachStatementNode.OnFailClause()).(*BLangOnFailClause),
		)
	}
	return bLForeach
}

func (n *NodeBuilder) TransformBinaryExpression(binaryBLangExpression *st.BinaryExpressionNode) BLangNode {
	if binaryBLangExpression.Operator().Kind() == st.ELVIS_TOKEN {
		panic("TransformBinaryExpression: elvis operator not supported")
	}

	bLBinaryExpr := BLangBinaryExpr{}
	bLBinaryExpr.pos = n.getPosition(binaryBLangExpression)
	bLBinaryExpr.LhsExpr = n.createExpression(binaryBLangExpression.LhsExpr())
	bLBinaryExpr.RhsExpr = n.createExpression(binaryBLangExpression.RhsExpr())
	if binaryBLangExpression.Operator() == nil {
		n.cx.InternalError("binary expression is missing an operator token", bLBinaryExpr.pos)
		return &bLBinaryExpr
	}
	bLBinaryExpr.OpKind = model.OperatorKindValueFrom(binaryBLangExpression.Operator().Text())
	return &bLBinaryExpr
}

func (n *NodeBuilder) TransformBracedExpression(bracedBLangExpression *st.BracedExpressionNode) BLangNode {
	return n.createActionOrExpression(bracedBLangExpression.Expression()).(BLangNode)
}

func (n *NodeBuilder) TransformCheckExpression(checkBLangExpression *st.CheckExpressionNode) BLangNode {
	pos := n.getPosition(checkBLangExpression)
	// we are deviating from the spec here (https://ballerina.io/spec/lang/master/#section_6.33) check is only suppose
	// to work with expression but jBallerina also allow remote method calls (which is an action)
	expr := n.createActionOrExpression(checkBLangExpression.Expression())
	if checkBLangExpression.CheckKeyword().Kind() == st.CHECK_KEYWORD {
		checkedExpr := &BLangCheckedExpr{}
		checkedExpr.pos = pos
		checkedExpr.Expr = expr
		return checkedExpr
	}
	checkPanickedExpr := &BLangCheckPanickedExpr{}
	checkPanickedExpr.pos = pos
	checkPanickedExpr.Expr = expr
	return checkPanickedExpr
}

func (n *NodeBuilder) TransformFieldAccessExpression(fieldAccessBLangExpression *st.FieldAccessExpressionNode) BLangNode {
	fieldName := fieldAccessBLangExpression.FieldName()
	if fieldName.Kind() == st.QUALIFIED_NAME_REFERENCE {
		panic("TransformFieldAccessExpression: QUALIFIED_NAME_REFERENCE unsupported")
	}

	bLFieldBasedAccess := &BLangFieldBaseAccess{}
	simpleNameRef := fieldName.(*st.SimpleNameReferenceNode)
	bLFieldBasedAccess.Field = n.createIdentifierNodeFromToken(n.getPosition(fieldAccessBLangExpression.FieldName()), simpleNameRef.Name())

	containerExpr := fieldAccessBLangExpression.Expression()
	if containerExpr.Kind() == st.BRACED_EXPRESSION {
		bracedExpr := containerExpr.(*st.BracedExpressionNode)
		bLFieldBasedAccess.Expr = n.createExpression(bracedExpr.Expression())
	} else {
		bLFieldBasedAccess.Expr = n.createExpression(containerExpr)
	}

	bLFieldBasedAccess.pos = n.getPosition(fieldAccessBLangExpression)
	return bLFieldBasedAccess
}

func (n *NodeBuilder) TransformFunctionCallExpression(functionCallBLangExpression *st.FunctionCallExpressionNode) BLangNode {
	return n.createBLangInvocation(
		functionCallBLangExpression.FunctionName(),
		functionCallBLangExpression.Arguments(),
		n.getPosition(functionCallBLangExpression),
		n.isFunctionCallAsync(functionCallBLangExpression))
}

func (n *NodeBuilder) TransformMethodCallExpression(methodCallBLangExpression *st.MethodCallExpressionNode) BLangNode {
	bLInvocation := n.createBLangInvocation(methodCallBLangExpression.MethodName(),
		methodCallBLangExpression.Arguments(),
		n.getPosition(methodCallBLangExpression), false)
	bLInvocation.Expr = n.createExpression(methodCallBLangExpression.Expression())
	return bLInvocation
}

func (n *NodeBuilder) TransformMappingConstructorExpression(mappingConstructorBLangExpression *st.MappingConstructorExpressionNode) BLangNode {
	mappingConstructor := &BLangMappingConstructorExpr{
		Fields: make([]MappingField, 0),
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
			key := &BLangMappingKey{
				Expr: keyExpr,
				Kind: MappingKeyComputed,
			}
			key.SetPosition(n.getPosition(computedNameField.FieldNameExpr()))
			keyValueField := &BLangMappingKeyValueField{
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
			keyKind := MappingKeyIdentifier
			if isStringLit {
				keyKind = MappingKeyStringLiteral
			}
			key := &BLangMappingKey{
				Expr: n.createSpecificFieldNameLiteral(specificField.FieldName()),
				Kind: keyKind,
			}
			key.SetPosition(n.getPosition(specificField.FieldName()))
			keyValueField := &BLangMappingKeyValueField{
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

func (n *NodeBuilder) TransformIndexedExpression(indexedBLangExpression *st.IndexedExpressionNode) BLangNode {
	indexBasedAccess := &BLangIndexBasedAccess{}
	indexBasedAccess.pos = n.getPosition(indexedBLangExpression)
	keys := indexedBLangExpression.KeyExpression()
	if keys.Size() == 0 {
		panic("missing key expression in member access expression")
	} else if keys.Size() == 1 {
		indexBasedAccess.IndexExpr = n.createExpression(keys.Get(0))
	} else {
		listConstructorExpr := &BLangListConstructorExpr{}
		listConstructorExpr.pos = n.getPositionRange(keys.Get(0), keys.Get(keys.Size()-1))
		exprs := make([]BLangExpression, 0, keys.Size())
		for i := 0; i < keys.Size(); i++ {
			exprs = append(exprs, n.createExpression(keys.Get(i)))
		}
		listConstructorExpr.Exprs = exprs
		indexBasedAccess.IndexExpr = listConstructorExpr
	}

	indexBasedAccess.Expr = n.createExpression(indexedBLangExpression.ContainerExpression())
	return indexBasedAccess
}

func (n *NodeBuilder) TransformTypeofExpression(typeofBLangExpression *st.TypeofExpressionNode) BLangNode {
	panic("TransformTypeofExpression unimplemented")
}

func (n *NodeBuilder) TransformUnaryExpression(unaryBLangExpression *st.UnaryExpressionNode) BLangNode {
	pos := n.getPosition(unaryBLangExpression)
	operator := model.OperatorKindValueFrom(unaryBLangExpression.UnaryOperator().Text())
	expr := n.createExpression(unaryBLangExpression.Expression())
	if operator == model.OperatorKind_SUB {
		if lit, ok := expr.(*BLangLiteral); ok && foldNegativeIntLiteral(lit) {
			lit.SetPosition(pos)
			return lit
		}
	}
	return createBLangUnaryExpr(pos, operator, expr)
}

// foldNegativeIntLiteral folds `-N` into a single int literal when `N` is an
// integer literal whose positive value overflows int64 but the negated value
// fits (e.g. `-9223372036854775808`). Without this fold, `N` is parsed as a
// float (losing precision) and later coerced back to int, corrupting the
// value used at runtime (e.g. for `<decimal>-9223372036854775808`).
func foldNegativeIntLiteral(lit *BLangLiteral) bool {
	if lit.GetValueType().BTypeGetTag() != TypeTags_INT {
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

func (n *NodeBuilder) TransformComputedNameField(computedNameFieldNode *st.ComputedNameFieldNode) BLangNode {
	panic("TransformComputedNameField unimplemented")
}

func (n *NodeBuilder) TransformConstantDeclaration(constantDeclarationNode *st.ConstantDeclarationNode) BLangNode {
	// Line 940: BLangConstant constantNode = (BLangConstant) TreeBuilder.createConstantNode();
	constantNode := createConstantNode()

	pos := n.getPositionWithoutMetadata(constantDeclarationNode)

	identifierPos := n.getPosition(constantDeclarationNode.VariableName())

	nameIdentifier := createIdentifierFromToken(identifierPos, constantDeclarationNode.VariableName())
	constantNode.Name = &nameIdentifier

	constantNode.Expr = n.createExpression(constantDeclarationNode.Initializer())

	constantNode.pos = pos

	typeDescriptor := constantDeclarationNode.TypeDescriptor()
	if typeDescriptor != nil {
		constantNode.SetTypeNode(n.createTypeNode(typeDescriptor).(BType))
	}

	n.populateMetadata(constantDeclarationNode.Metadata(), constantNode)

	visibilityQualifier := constantDeclarationNode.VisibilityQualifier()
	if visibilityQualifier != nil && visibilityQualifier.Kind() == st.PUBLIC_KEYWORD {
		constantNode.SetPublic()
	}

	return constantNode
}

func (n *NodeBuilder) TransformDefaultableParameter(defaultableParameterNode *st.DefaultableParameterNode) BLangNode {
	paramName := defaultableParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarInner(paramName, defaultableParameterNode.TypeName(), defaultableParameterNode.Expression(), nil, defaultableParameterNode.Annotations())

	simpleVar.pos = n.getPosition(defaultableParameterNode)

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	simpleVar.SetDefaultableParam()

	return simpleVar
}

func (n *NodeBuilder) createSimpleVarWithTokenNodeNodeList(name st.Token, typeName st.Node, annotations st.NodeList[*st.AnnotationNode]) *BLangSimpleVariable {
	if name != nil {
		return n.createSimpleVarInner(name, typeName, nil, nil, annotations)
	}
	return n.createSimpleVarInner(nil, typeName, nil, nil, annotations)
}

func (n *NodeBuilder) TransformRequiredParameter(requiredParameterNode *st.RequiredParameterNode) BLangNode {
	paramName := requiredParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, requiredParameterNode.TypeName(), requiredParameterNode.Annotations())

	simpleVar.pos = n.getPosition(requiredParameterNode)

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		// Param doesn't have a name and also is not a missing node
		// Therefore, assigning the built-in location
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	simpleVar.SetRequiredParam()

	return simpleVar
}

func (n *NodeBuilder) TransformIncludedRecordParameter(includedRecordParameterNode *st.IncludedRecordParameterNode) BLangNode {
	paramName := includedRecordParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, includedRecordParameterNode.TypeName(), includedRecordParameterNode.Annotations())

	simpleVar.pos = n.getPosition(includedRecordParameterNode)

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	simpleVar.SetRequiredParam()
	simpleVar.SetIncludedRecordParam()

	return simpleVar
}

func (n *NodeBuilder) TransformRestParameter(restParameterNode *st.RestParameterNode) BLangNode {
	paramName := restParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, restParameterNode.TypeName(), restParameterNode.Annotations())

	simpleVar.pos = n.getPosition(restParameterNode)

	if paramName != nil {
		simpleVar.Name.SetPosition(n.getPosition(paramName))
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.GetPosition()) {
		simpleVar.Name.SetPosition(diagnostics.NewBuiltinLocation())
	}

	simpleVar.SetRestParam()

	return simpleVar
}

func (n *NodeBuilder) TransformImportOrgName(importOrgNameNode *st.ImportOrgNameNode) BLangNode {
	panic("TransformImportOrgName unimplemented")
}

func (n *NodeBuilder) TransformImportPrefix(importPrefixNode *st.ImportPrefixNode) BLangNode {
	panic("TransformImportPrefix unimplemented")
}

func (n *NodeBuilder) TransformSpecificField(specificFieldNode *st.SpecificFieldNode) BLangNode {
	panic("TransformSpecificField unimplemented")
}

func (n *NodeBuilder) TransformSpreadField(spreadFieldNode *st.SpreadFieldNode) BLangNode {
	panic("TransformSpreadField unimplemented")
}

func (n *NodeBuilder) TransformNamedArgument(namedArgumentNode *st.NamedArgumentNode) BLangNode {
	namedArg := &BLangNamedArgsExpression{}
	namedArg.pos = n.getPosition(namedArgumentNode)
	nameToken := namedArgumentNode.ArgumentName().Name()
	namedArg.Name = n.createIdentifierNodeFromToken(n.getPosition(nameToken), nameToken)
	namedArg.Expr = n.createExpression(namedArgumentNode.Expression())
	return namedArg
}

func (n *NodeBuilder) TransformPositionalArgument(positionalArgumentNode *st.PositionalArgumentNode) BLangNode {
	return n.createExpression(positionalArgumentNode.Expression())
}

func (n *NodeBuilder) TransformRestArgument(restArgumentNode *st.RestArgumentNode) BLangNode {
	panic("TransformRestArgument unimplemented")
}

func (n *NodeBuilder) TransformInferredTypedescDefault(inferredTypedescDefaultNode *st.InferredTypedescDefaultNode) BLangNode {
	node := &BLangInferredTypedescDefault{}
	node.pos = n.getPosition(inferredTypedescDefaultNode)
	return node
}

func (n *NodeBuilder) TransformObjectTypeDescriptor(objectTypeDescriptorNode *st.ObjectTypeDescriptorNode) BLangNode {
	objectType := &BLangObjectType{members: make(map[string]ObjectMember)}

	// Process object type qualifiers (client/service/isolated)
	qualifiers := objectTypeDescriptorNode.ObjectTypeQualifiers()
	for q := range qualifiers.Iterator() {
		switch q.Kind() {
		case st.CLIENT_KEYWORD:
			objectType.NetworkQuals = ObjectNetworkQualsClient
		case st.SERVICE_KEYWORD:
			objectType.NetworkQuals = ObjectNetworkQualsService
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
			bField := &BObjectField{
				Ty: n.createTypeNode(objectField.TypeName()).(BType),
			}
			bField.name = fieldName
			bField.pos = n.getPosition(objectField)
			if vis := objectField.VisibilityQualifier(); vis != nil && vis.Kind() == st.PUBLIC_KEYWORD {
				bField.flags |= model.FlagPublic
			}
			n.populateMetadata(objectField.Metadata(), bField)
			if objectType.AddMember(bField) {
				n.cx.SyntaxError("redeclared symbol '"+fieldName+"'", bField.pos)
			}
		case st.METHOD_DECLARATION:
			methodDecl := member.(*st.MethodDeclarationNode)
			methodName, _ := normalizedIdentifierValue(methodDecl.MethodName().Text())
			bMethod := &BMethodDecl{}
			bMethod.name = methodName
			bMethod.pos = n.getPosition(methodDecl)
			bMethod.memberKind = ObjectMemberKindMethod

			// Process visibility and method kind from qualifier list
			methodQuals := methodDecl.QualifierList()
			for q := range methodQuals.Iterator() {
				switch q.Kind() {
				case st.PUBLIC_KEYWORD:
					bMethod.flags |= model.FlagPublic
				case st.REMOTE_KEYWORD:
					bMethod.memberKind = ObjectMemberKindRemoteMethod
				case st.RESOURCE_KEYWORD:
					bMethod.memberKind = ObjectMemberKindResourceMethod
				case st.ISOLATED_KEYWORD:
					bMethod.SetIsolated()
				case st.TRANSACTIONAL_KEYWORD:
					bMethod.SetTransactional()
				}
			}

			if bMethod.memberKind == ObjectMemberKindRemoteMethod {
				bMethod.name = model.RemoteMethodName(bMethod.name)
			}

			// Build function type from method signature
			funcSig := methodDecl.MethodSignature()
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
						bMethod.RestParam = &ftParam
					} else {
						bMethod.RequiredParams = append(bMethod.RequiredParams, ftParam)
					}
				}

				// Process return type
				if retTypeDesc := funcSig.ReturnTypeDesc(); retTypeDesc != nil {
					returnsKeyword := retTypeDesc.ReturnsKeyword()
					if returnsKeyword != nil && !returnsKeyword.IsMissing() {
						bMethod.SetExplicitReturnTypeDescriptor()
					}
					bMethod.ReturnTypeDescriptor = n.createTypeNode(retTypeDesc.Type()).(BType)
				} else {
					nilRet := &BLangValueType{TypeKind: TypeKind_NIL}
					nilRet.pos = diagnostics.NewBuiltinLocation()
					bMethod.ReturnTypeDescriptor = nilRet
				}
			}

			if objectType.AddMember(bMethod) {
				n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(bMethod.name)+"'", bMethod.pos)
			}
		case st.TYPE_REFERENCE:
			typeRef := member.(*st.TypeReferenceNode)
			objectType.unresolvedInclusions = append(objectType.unresolvedInclusions, n.createTypeNode(typeRef.TypeName()).(*BLangUserDefinedType))
		default:
			panic("unexpected member kind in object type descriptor")
		}
	}

	objectType.pos = n.getPosition(objectTypeDescriptorNode)
	return objectType
}

func (n *NodeBuilder) TransformObjectConstructorExpression(objectConstructorBLangExpression *st.ObjectConstructorExpressionNode) BLangNode {
	panic("TransformObjectConstructorExpression unimplemented")
}

func (n *NodeBuilder) TransformRecordTypeDescriptor(recordTypeDescriptorNode *st.RecordTypeDescriptorNode) BLangNode {
	recordType := &BLangRecordType{}
	fields := recordTypeDescriptorNode.Fields()
	for i := 0; i < fields.Size(); i++ {
		field := fields.Get(i)
		switch field.Kind() {
		case st.RECORD_FIELD:
			recordField := field.(*st.RecordFieldNode)
			fieldName, _ := normalizedIdentifierValue(recordField.FieldName().Text())
			bField := BField{
				Name: model.Name(fieldName),
				Type: n.createTypeNode(recordField.TypeName()).(BType),
			}
			bField.pos = n.getPosition(recordField)
			if recordField.ReadonlyKeyword() != nil {
				bField.SetReadonly()
			}
			if recordField.QuestionMarkToken() != nil {
				bField.SetOptional()
			}
			n.populateMetadata(recordField.Metadata(), &bField)
			recordType.AddField(fieldName, bField)
		case st.RECORD_FIELD_WITH_DEFAULT_VALUE:
			recordFieldDV := field.(*st.RecordFieldWithDefaultValueNode)
			fieldName, _ := normalizedIdentifierValue(recordFieldDV.FieldName().Text())
			bField := BField{
				Name:        model.Name(fieldName),
				Type:        n.createTypeNode(recordFieldDV.TypeName()).(BType),
				DefaultExpr: n.createExpression(recordFieldDV.Expression()),
			}
			bField.pos = n.getPosition(recordFieldDV)
			if recordFieldDV.ReadonlyKeyword() != nil {
				bField.SetReadonly()
			}
			n.populateMetadata(recordFieldDV.Metadata(), &bField)
			recordType.AddField(fieldName, bField)
		case st.TYPE_REFERENCE:
			typeRef := field.(*st.TypeReferenceNode)
			recordType.TypeInclusions = append(recordType.TypeInclusions, n.createTypeNode(typeRef.TypeName()).(BType))
		default:
			panic("unexpected field kind in record type descriptor")
		}
	}
	if restDesc := recordTypeDescriptorNode.RecordRestDescriptor(); restDesc != nil {
		recordType.RestType = n.createTypeNode(restDesc.TypeName()).(BType)
	}
	recordType.IsOpen = recordTypeDescriptorNode.BodyStartDelimiter().Kind() == st.OPEN_BRACE_TOKEN
	recordType.pos = n.getPosition(recordTypeDescriptorNode)
	return recordType
}

func (n *NodeBuilder) TransformReturnTypeDescriptor(returnTypeDescriptorNode *st.ReturnTypeDescriptorNode) BLangNode {
	panic("TransformReturnTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformNilTypeDescriptor(nilTypeDescriptorNode *st.NilTypeDescriptorNode) BLangNode {
	panic("TransformNilTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformOptionalTypeDescriptor(optionalTypeDescriptorNode *st.OptionalTypeDescriptorNode) BLangNode {
	typeDesc := optionalTypeDescriptorNode.TypeDescriptor()
	nilType := &BLangValueType{TypeKind: TypeKind_NIL}
	nilType.pos = n.getPosition(optionalTypeDescriptorNode.QuestionMarkToken())
	bLUnionType := &BLangUnionTypeNode{
		lhs: TypeData{
			TypeDescriptor: n.createTypeNode(typeDesc),
		},
		rhs: TypeData{
			TypeDescriptor: nilType,
		},
	}
	bLUnionType.pos = n.getPosition(optionalTypeDescriptorNode)
	return bLUnionType
}

func (n *NodeBuilder) TransformObjectField(objectFieldNode *st.ObjectFieldNode) BLangNode {
	panic("TransformObjectField unimplemented")
}

func (n *NodeBuilder) TransformRecordField(recordFieldNode *st.RecordFieldNode) BLangNode {
	panic("TransformRecordField unimplemented")
}

func (n *NodeBuilder) TransformRecordFieldWithDefaultValue(recordFieldWithDefaultValueNode *st.RecordFieldWithDefaultValueNode) BLangNode {
	panic("TransformRecordFieldWithDefaultValue unimplemented")
}

func (n *NodeBuilder) TransformRecordRestDescriptor(recordRestDescriptorNode *st.RecordRestDescriptorNode) BLangNode {
	panic("TransformRecordRestDescriptor unimplemented")
}

func (n *NodeBuilder) TransformTypeReference(typeReferenceNode *st.TypeReferenceNode) BLangNode {
	panic("TransformTypeReference unimplemented")
}

func (n *NodeBuilder) TransformAnnotation(annotationNode *st.AnnotationNode) BLangNode {
	annotation := &BLangAnnotationAttachment{}
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

func (n *NodeBuilder) TransformMetadata(metadataNode *st.MetadataNode) BLangNode {
	docString := getDocumentationString(metadataNode)
	if docString == nil || docString.IsMissing() {
		return nil
	}
	return n.createMarkdownDocumentationAttachment(docString)
}

func (n *NodeBuilder) TransformModuleVariableDeclaration(moduleVariableDeclarationNode *st.ModuleVariableDeclarationNode) BLangNode {
	typedBindingPattern := moduleVariableDeclarationNode.TypedBindingPattern()
	bindingPattern := typedBindingPattern.BindingPattern()
	pos := n.getPositionWithoutMetadata(moduleVariableDeclarationNode)

	variable := n.getBLangVariableNode(bindingPattern, pos)
	simpleVar := variable.(*BLangSimpleVariable)

	typeDesc := typedBindingPattern.TypeDescriptor()
	if typeDesc != nil {
		if isDeclaredWithVar(typeDesc) {
			simpleVar.SetIsDeclaredWithVar(true)
		} else {
			simpleVar.SetTypeNode(n.createTypeNode(typeDesc).(BType))
		}
	}

	initializer := moduleVariableDeclarationNode.Initializer()
	if initializer != nil {
		simpleVar.SetInitialExpression(n.createExpression(initializer))
	}

	if simpleVar.IsDeclaredWithVar && simpleVar.TypeNode() == nil && simpleVar.Expr == nil {
		n.cx.SyntaxError("var-declared module variable must have an initializer expression for type inference", pos)
		return simpleVar
	}

	n.populateModuleVariableVisibilityAndQualifiers(moduleVariableDeclarationNode, simpleVar)
	n.populateMetadata(moduleVariableDeclarationNode.Metadata(), simpleVar)

	simpleVar.pos = pos
	return simpleVar
}

func (n *NodeBuilder) populateModuleVariableVisibilityAndQualifiers(node *st.ModuleVariableDeclarationNode, simpleVar *BLangSimpleVariable) {
	visibilityQualifier := node.VisibilityQualifier()
	if visibilityQualifier != nil && visibilityQualifier.Kind() == st.PUBLIC_KEYWORD {
		simpleVar.SetPublic()
	}

	qualifiers := node.Qualifiers()
	for i := 0; i < qualifiers.Size(); i++ {
		qualifier := qualifiers.Get(i)
		switch qualifier.Kind() {
		case st.FINAL_KEYWORD:
			simpleVar.SetFinal()
		case st.ISOLATED_KEYWORD:
			simpleVar.SetIsolated()
		case st.CONFIGURABLE_KEYWORD:
			n.cx.Unimplemented("configurable module variables are not supported yet", simpleVar.pos)
		}
	}
}

func (n *NodeBuilder) TransformTypeTestExpression(typeTestBLangExpression *st.TypeTestExpressionNode) BLangNode {
	typeTestExpr := &BLangTypeTestExpr{}
	typeTestExpr.isNegation = typeTestBLangExpression.IsKeyword().Kind() == st.NOT_IS_KEYWORD
	typeTestExpr.Expr = n.createExpression(typeTestBLangExpression.Expression())
	typeTestExpr.Type = TypeData{TypeDescriptor: n.createTypeNode(typeTestBLangExpression.TypeDescriptor())}
	typeTestExpr.SetPosition(n.getPosition(typeTestBLangExpression))
	return typeTestExpr
}

func (n *NodeBuilder) TransformRemoteMethodCallAction(remoteMethodCallActionNode *st.RemoteMethodCallActionNode) BLangNode {
	inv := n.createBLangInvocation(remoteMethodCallActionNode.MethodName(),
		remoteMethodCallActionNode.Arguments(),
		n.getPosition(remoteMethodCallActionNode), false)
	action := &BLangRemoteMethodCallAction{}
	action.bLangInvocationBase = inv.bLangInvocationBase
	action.Expr = n.createExpression(remoteMethodCallActionNode.Expression())
	action.pos = n.getPosition(remoteMethodCallActionNode)
	return action
}

func (n *NodeBuilder) TransformMapTypeDescriptor(mapTypeDescriptorNode *st.MapTypeDescriptorNode) BLangNode {
	refType := &BLangBuiltInRefTypeNode{
		TypeKind: TypeKind_MAP,
	}
	refType.SetPosition(n.getPosition(mapTypeDescriptorNode))

	mapTypeParamsNode := mapTypeDescriptorNode.MapTypeParamsNode()
	if mapTypeParamsNode == nil || mapTypeParamsNode.TypeNode() == nil {
		panic("map type requires type parameter")
	}
	constraint := n.createTypeNode(mapTypeParamsNode.TypeNode())

	constrainedType := &BLangConstrainedType{
		Type:       TypeData{TypeDescriptor: refType},
		Constraint: TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(refType.GetPosition())
	return constrainedType
}

func (n *NodeBuilder) TransformNilLiteral(nilLiteralNode *st.NilLiteralNode) BLangNode {
	panic("TransformNilLiteral unimplemented")
}

func (n *NodeBuilder) TransformAnnotationDeclaration(annotationDeclarationNode *st.AnnotationDeclarationNode) BLangNode {
	annotation := &BLangAnnotation{}
	annotation.SetPosition(n.getPositionWithoutMetadata(annotationDeclarationNode))
	name := createIdentifierFromToken(n.getPosition(annotationDeclarationNode.AnnotationTag()), annotationDeclarationNode.AnnotationTag())
	annotation.Name = &name
	if visibility := annotationDeclarationNode.VisibilityQualifier(); visibility != nil && visibility.Kind() == st.PUBLIC_KEYWORD {
		annotation.SetPublic()
	}
	if constKeyword := annotationDeclarationNode.ConstKeyword(); constKeyword != nil && !constKeyword.IsMissing() {
		annotation.SetConst()
	}
	if typeDesc := annotationDeclarationNode.TypeDescriptor(); typeDesc != nil && !typeDesc.IsMissing() {
		annotation.SetTypeDescriptor(n.createTypeNode(typeDesc))
	}
	attachPoints := annotationDeclarationNode.AttachPoints()
	for attachPoint := range attachPoints.Iterator() {
		if attachPoint, ok := attachPoint.(*st.AnnotationAttachPointNode); ok {
			annotation.AddAttachPoint(n.createAnnotationAttachPoint(attachPoint))
		}
	}
	n.populateMetadata(annotationDeclarationNode.Metadata(), annotation)
	return annotation
}

func (n *NodeBuilder) TransformAnnotationAttachPoint(annotationAttachPointNode *st.AnnotationAttachPointNode) BLangNode {
	n.createAnnotationAttachPoint(annotationAttachPointNode)
	return nil
}

func (n *NodeBuilder) createAnnotationAttachPoint(annotationAttachPointNode *st.AnnotationAttachPointNode) AttachPoint {
	parts := []string{}
	identifiers := annotationAttachPointNode.Identifiers()
	for i := 0; i < identifiers.Size(); i++ {
		parts = append(parts, identifiers.Get(i).Text())
	}
	point, ok := annotationAttachPointFromParts(parts)
	if !ok {
		n.cx.SyntaxError("unknown annotation attach point '"+strings.Join(parts, " ")+"'", n.getPosition(annotationAttachPointNode))
	}
	return AttachPoint{
		Point:  point,
		Source: annotationAttachPointNode.SourceKeyword() != nil,
	}
}

// annotationAttachPointFromParts maps the space-separated source spelling of an
// annotation attach point to its Point. This is the inverse of Point.String(),
// but keyed on the spelled-out source form (e.g. "object function"), which
// differs from the canonical key (e.g. "objectfunction").
func annotationAttachPointFromParts(parts []string) (Point, bool) {
	switch strings.Join(parts, " ") {
	case "type":
		return Point_TYPE, true
	case "object":
		return Point_OBJECT, true
	case "function":
		return Point_FUNCTION, true
	case "object function":
		return Point_OBJECT_METHOD, true
	case "service remote function":
		return Point_SERVICE_REMOTE, true
	case "parameter":
		return Point_PARAMETER, true
	case "return":
		return Point_RETURN, true
	case "service":
		return Point_SERVICE, true
	case "field":
		return Point_FIELD, true
	case "object field":
		return Point_OBJECT_FIELD, true
	case "record field":
		return Point_RECORD_FIELD, true
	case "listener":
		return Point_LISTENER, true
	case "annotation":
		return Point_ANNOTATION, true
	case "external":
		return Point_EXTERNAL, true
	case "var":
		return Point_VAR, true
	case "const":
		return Point_CONST, true
	case "worker":
		return Point_WORKER, true
	case "class":
		return Point_CLASS, true
	default:
		return 0, false
	}
}

type xmlNamespaceDeclarationNode interface {
	st.Node
	Namespaceuri() st.ExpressionNode
	NamespacePrefix() *st.IdentifierToken
}

func (n *NodeBuilder) transformXMLNamespaceDeclaration(node xmlNamespaceDeclarationNode) BLangNode {
	pos := n.getPosition(node)
	xmlns := &BLangXMLNS{}
	xmlns.SetPosition(pos)
	n.populateXMLNS(xmlns, pos, node.Namespaceuri(), node.NamespacePrefix())
	return xmlns
}

func (n *NodeBuilder) TransformXMLNamespaceDeclaration(xMLNamespaceDeclarationNode *st.XMLNamespaceDeclarationNode) BLangNode {
	return n.transformXMLNamespaceDeclaration(xMLNamespaceDeclarationNode)
}

func (n *NodeBuilder) TransformModuleXMLNamespaceDeclaration(moduleXMLNamespaceDeclarationNode *st.ModuleXMLNamespaceDeclarationNode) BLangNode {
	return n.transformXMLNamespaceDeclaration(moduleXMLNamespaceDeclarationNode)
}

func (n *NodeBuilder) populateXMLNS(target *BLangXMLNS, pos diagnostics.Location, uriNode st.ExpressionNode, prefixTok *st.IdentifierToken) {
	if uriNode != nil {
		target.SetNamespaceURI(n.createExpression(uriNode))
	}
	if prefixTok != nil {
		prefixIdent := createIdentifierFromToken(n.getPosition(prefixTok), prefixTok)
		target.SetPrefix(&prefixIdent)
	}
}

func (n *NodeBuilder) TransformFunctionBodyBlock(functionBodyBlockNode *st.FunctionBodyBlockNode) BLangNode {
	bLFuncBody := &BLangBlockFunctionBody{}
	statements := []StatementNode{}
	stmtList := statements
	namedWorkerDeclarator := functionBodyBlockNode.NamedWorkerDeclarator()
	if namedWorkerDeclarator != nil {
		panic("unimplemented")
	}

	n.generateAndAddBLangStatements(functionBodyBlockNode.Statements(), &stmtList, 0, functionBodyBlockNode)

	bLFuncBody.Stmts = stmtList
	bLFuncBody.pos = n.getPosition(functionBodyBlockNode)
	return bLFuncBody
}

func (n *NodeBuilder) generateForkStatements(statements *[]StatementNode, forkStatementNode *st.ForkStatementNode) {
	panic("generateForkStatements unimplemented")
}

func (n *NodeBuilder) TransformNamedWorkerDeclaration(namedWorkerDeclarationNode *st.NamedWorkerDeclarationNode) BLangNode {
	panic("TransformNamedWorkerDeclaration unimplemented")
}

func (n *NodeBuilder) TransformNamedWorkerDeclarator(namedWorkerDeclarator *st.NamedWorkerDeclarator) BLangNode {
	panic("TransformNamedWorkerDeclarator unimplemented")
}

func (n *NodeBuilder) TransformBasicLiteral(basicLiteralNode *st.BasicLiteralNode) BLangNode {
	panic("TransformBasicLiteral unimplemented")
}

func (n *NodeBuilder) TransformSimpleNameReference(simpleNameReferenceNode *st.SimpleNameReferenceNode) BLangNode {
	panic("TransformSimpleNameReference unimplemented")
}

func (n *NodeBuilder) TransformQualifiedNameReference(qualifiedNameReferenceNode *st.QualifiedNameReferenceNode) BLangNode {
	nameReference := n.createBLangNameReference(qualifiedNameReferenceNode)
	bLVarRef := &BLangSimpleVarRef{}
	bLVarRef.pos = n.getPosition(qualifiedNameReferenceNode)
	bLVarRef.PkgAlias = nameReference[0]
	bLVarRef.VariableName = nameReference[1]
	return bLVarRef
}

func (n *NodeBuilder) TransformBuiltinSimpleNameReference(builtinSimpleNameReferenceNode *st.BuiltinSimpleNameReferenceNode) BLangNode {
	panic("TransformBuiltinSimpleNameReference unimplemented")
}

func (n *NodeBuilder) TransformTrapExpression(trapBLangExpression *st.TrapExpressionNode) BLangNode {
	pos := n.getPosition(trapBLangExpression)
	expr := n.createActionOrExpression(trapBLangExpression.Expression())
	trapExpr := &BLangTrapExpr{}
	trapExpr.pos = pos
	trapExpr.Expr = expr
	return trapExpr
}

func (n *NodeBuilder) TransformListConstructorExpression(listConstructorBLangExpression *st.ListConstructorExpressionNode) BLangNode {
	argExprList := make([]BLangExpression, 0)
	spreadMemberIndexes := make([]int, 0)
	listConstructorExpr := &BLangListConstructorExpr{}

	expressions := listConstructorBLangExpression.Expressions()
	for i := 0; i < expressions.Size(); i += 2 {
		listMember := expressions.Get(i)
		var memberExpr BLangExpression
		if listMember.Kind() == st.SPREAD_MEMBER {
			spreadMember := listMember.(*st.SpreadMemberNode)
			memberExpr = n.createExpression(spreadMember.Expression())
			spreadMemberIndexes = append(spreadMemberIndexes, len(argExprList))
		} else {
			memberExpr = n.createExpression(listMember)
		}
		argExprList = append(argExprList, memberExpr)
	}

	listConstructorExpr.Exprs = argExprList
	for _, index := range spreadMemberIndexes {
		listConstructorExpr.SetSpreadMember(index)
	}
	listConstructorExpr.pos = n.getPosition(listConstructorBLangExpression)
	return listConstructorExpr
}

func (n *NodeBuilder) TransformTypeCastExpression(typeCastBLangExpression *st.TypeCastExpressionNode) BLangNode {
	typeConversionNode := &BLangTypeConversionExpr{}
	typeConversionNode.SetPosition(n.getPosition(typeCastBLangExpression))
	typeCastParamNode := typeCastBLangExpression.TypeCastParam()
	if typeCastParamNode != nil && typeCastParamNode.Type() != nil {
		typeConversionNode.TypeDescriptor = n.createTypeNode(typeCastParamNode.Type()).(BType)
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

func (n *NodeBuilder) TransformTypeCastParam(typeCastParamNode *st.TypeCastParamNode) BLangNode {
	panic("TransformTypeCastParam unimplemented")
}

func (n *NodeBuilder) TransformUnionTypeDescriptor(unionTypeDescriptorNode *st.UnionTypeDescriptorNode) BLangNode {
	lhs := unionTypeDescriptorNode.LeftTypeDesc()
	rhs := unionTypeDescriptorNode.RightTypeDesc()
	bLUnionType := &BLangUnionTypeNode{
		lhs: TypeData{
			TypeDescriptor: n.createTypeNode(lhs),
		},
		rhs: TypeData{
			TypeDescriptor: n.createTypeNode(rhs),
		},
	}
	bLUnionType.pos = n.getPosition(unionTypeDescriptorNode)
	return bLUnionType
}

func (n *NodeBuilder) TransformTableConstructorExpression(tableConstructorBLangExpression *st.TableConstructorExpressionNode) BLangNode {
	panic("TransformTableConstructorExpression unimplemented")
}

func (n *NodeBuilder) TransformKeySpecifier(keySpecifierNode *st.KeySpecifierNode) BLangNode {
	panic("TransformKeySpecifier unimplemented")
}

func (n *NodeBuilder) TransformStreamTypeDescriptor(streamTypeDescriptorNode *st.StreamTypeDescriptorNode) BLangNode {
	position := n.getPosition(streamTypeDescriptorNode)
	paramsNode := streamTypeDescriptorNode.StreamTypeParamsNode()
	if paramsNode == nil {
		refType := &BLangBuiltInRefTypeNode{
			TypeKind: TypeKind_STREAM,
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
	streamType := NewBLangStreamType(
		TypeData{TypeDescriptor: n.createTypeNode(valueDesc)},
		TypeData{TypeDescriptor: n.createTypeNode(completionDesc)},
	)
	streamType.SetPosition(position)
	return streamType
}

func (n *NodeBuilder) TransformStreamTypeParams(streamTypeParamsNode *st.StreamTypeParamsNode) BLangNode {
	panic("TransformStreamTypeParams unimplemented")
}

func (n *NodeBuilder) TransformLetExpression(letBLangExpression *st.LetExpressionNode) BLangNode {
	panic("TransformLetExpression unimplemented")
}

func (n *NodeBuilder) TransformLetVariableDeclaration(letVariableDeclarationNode *st.LetVariableDeclarationNode) BLangNode {
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
	variableDef := varDef.(*BLangSimpleVariableDef)
	variableDef.Var.SetFinal()
	return varDef.(BLangNode)
}

func (n *NodeBuilder) TransformTemplateExpression(templateBLangExpression *st.TemplateExpressionNode) BLangNode {
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

func (n *NodeBuilder) buildXMLTemplateExpr(templateBLangExpression *st.TemplateExpressionNode, pos diagnostics.Location) BLangNode {
	if !xmlTemplateHasInterpolation(templateBLangExpression.Content()) {
		// If we don't have interpolations we build a literal as an optimization
		return n.buildXMLSequenceLiteral(templateBLangExpression, pos)
	}

	tpl := &BLangXMLTemplateExpr{}
	tpl.SetPosition(pos)
	tpl.Kind = TemplateExprKindXML
	for tok, diag := range n.flattenXMLTemplateContent(templateBLangExpression.Content(), XMLTemplateInsertionKindContent) {
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

func (n *NodeBuilder) buildXMLSequenceLiteral(templateBLangExpression *st.TemplateExpressionNode, pos diagnostics.Location) BLangNode {
	var children []BLangExpression
	content := templateBLangExpression.Content()
	for child := range content.Iterator() {
		bl := n.TransformSyntaxNode(child)
		if bl == nil {
			n.cx.InternalError("xml template child did not produce BLangNode", n.getPosition(child))
			return nil
		}
		expr, ok := bl.(BLangExpression)
		if !ok {
			n.cx.InternalError("xml template child did not produce BLangExpression", n.getPosition(child))
			return nil
		}
		children = append(children, expr)
	}
	if len(children) == 1 {
		return children[0]
	}
	seq := &BLangXMLSequenceLiteral{}
	seq.pos = pos
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
	NamespaceInsertions []XMLTemplateNamespaceInsertion
	Insertion           BLangExpression
	InsertionKind       XMLTemplateInsertionKind
}

func newXMLTemplateTextToken(value string, insertions ...XMLTemplateNamespaceInsertion) xmlTemplateToken {
	return xmlTemplateToken{Kind: xmlTemplateTokenKindText, Text: value, NamespaceInsertions: insertions}
}

func newXMLTemplateInsertionToken(expr BLangExpression, kind XMLTemplateInsertionKind) xmlTemplateToken {
	return xmlTemplateToken{Kind: xmlTemplateTokenKindInsertion, Insertion: expr, InsertionKind: kind}
}

type xmlTemplateTextAccumulator struct {
	text                strings.Builder
	namespaceInsertions []XMLTemplateNamespaceInsertion
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

func (n *NodeBuilder) flattenXMLTemplateContent(content st.NodeList[st.Node], kind XMLTemplateInsertionKind) iter.Seq2[xmlTemplateToken, *xmlTemplateDiagnostic] {
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

func (n *NodeBuilder) flattenXMLTemplateNodeWithNamespace(
	node st.Node,
	kind XMLTemplateInsertionKind,
	namespaceInsertion *XMLTemplateNamespaceInsertion,
	yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool,
) bool {
	switch x := node.(type) {
	case st.Token:
		return yield(newXMLTemplateTextToken(x.Text()), nil)
	case *st.InterpolationNode:
		expr := n.createActionOrExpression(x.Expression())
		be, ok := expr.(BLangExpression)
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

func (n *NodeBuilder) flattenXMLTemplateElement(
	x *st.XMLElementNode,
	parentNamespaceInsertion *XMLTemplateNamespaceInsertion,
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
		if !n.flattenXMLTemplateNodeWithNamespace(child, XMLTemplateInsertionKindContent, namespaceInsertion, yield) {
			return false
		}
	}
	return yield(newXMLTemplateTextToken("</"+name+">"), nil)
}

func (n *NodeBuilder) flattenXMLTemplateEmptyElement(
	x *st.XMLEmptyElementNode,
	parentNamespaceInsertion *XMLTemplateNamespaceInsertion,
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

func (n *NodeBuilder) flattenXMLTemplatePI(x *st.XMLProcessingInstruction, yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) bool {
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

func (n *NodeBuilder) flattenXMLTemplateAttributes(attrs st.NodeList[*st.XMLAttributeNode], yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) bool {
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

func (n *NodeBuilder) flattenXMLTemplateAttributeValue(
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
			if !n.flattenXMLTemplateNodeWithNamespace(ins, XMLTemplateInsertionKindAttribute, nil, yield) {
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

func (n *NodeBuilder) reportXMLTemplateDiagnostic(diag *xmlTemplateDiagnostic) {
	if diag.Internal {
		n.cx.InternalError(diag.Message, diag.Position)
		return
	}
	n.cx.SemanticError(diag.Message, diag.Position)
}

func (n *NodeBuilder) collectXMLTemplateNamespaceInsertion(node st.Node) XMLTemplateNamespaceInsertion {
	insn := XMLTemplateNamespaceInsertion{
		UsedPrefixes: map[string]struct{}{},
	}
	n.collectXMLTemplateNamespaceRefs(node, nil, &insn)
	return insn
}

func (n *NodeBuilder) collectXMLTemplateNamespaceRefs(node st.Node, scopes []map[string]struct{}, insn *XMLTemplateNamespaceInsertion) {
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

func (n *NodeBuilder) collectXMLTemplateAttributeNamespaceRefs(
	attrs st.NodeList[*st.XMLAttributeNode],
	scopes []map[string]struct{},
	insn *XMLTemplateNamespaceInsertion,
) {
	for attr := range attrs.Iterator() {
		name := n.xmlNameToString(attr.AttributeName())
		if isXMLTemplateXMLNSName(name) {
			continue
		}
		n.recordXMLTemplateNameRef(name, false, scopes, insn)
	}
}

func (n *NodeBuilder) recordXMLTemplateNameRef(name string, isElement bool, scopes []map[string]struct{}, insn *XMLTemplateNamespaceInsertion) {
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

func (n *NodeBuilder) collectInlineXMLTemplatePrefixes(attrs st.NodeList[*st.XMLAttributeNode]) map[string]struct{} {
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

func (n *NodeBuilder) buildStringTemplateExpr(node *st.TemplateExpressionNode, pos diagnostics.Location) BLangNode {
	// We maintain fallowing 2 invariants
	// 1. First and last elements are always strings
	// 2. Between any two expressions there is a string
	// For this we will add empty strings. This is meant to reducing the number of branchings needed in runtime
	var strs []string
	var insertions []BLangExpression
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
			be, ok := expr.(BLangExpression)
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
	tpl := &BLangTemplateExpr{Kind: TemplateExprKindString, Strings: strs, Insertions: insertions}
	tpl.SetPosition(pos)
	return tpl
}

func (n *NodeBuilder) xmlNameToString(name st.XMLNameNode) string {
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

func (n *NodeBuilder) xmlAttributes(attrs st.NodeList[*st.XMLAttributeNode]) []BLangXMLAttribute {
	out := make([]BLangXMLAttribute, 0, attrs.Size())
	for attrNode := range attrs.Iterator() {
		attr := n.TransformXMLAttribute(attrNode).(*BLangXMLAttribute)
		out = append(out, *attr)
	}
	return out
}

func (n *NodeBuilder) TransformXMLElement(xMLElementNode *st.XMLElementNode) BLangNode {
	elem := &BLangXMLElementLiteral{}
	elem.pos = n.getPosition(xMLElementNode)
	if start := xMLElementNode.StartTag(); start != nil {
		elem.Name = n.xmlNameToString(start.Name())
		elem.Attrs = n.xmlAttributes(start.Attributes())
	}
	var children []BLangExpression
	content := xMLElementNode.Content()
	for child := range content.Iterator() {
		bl := n.TransformSyntaxNode(child)
		if bl == nil {
			continue
		}
		expr, ok := bl.(BLangExpression)
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
		seq := &BLangXMLSequenceLiteral{}
		seq.pos = elem.pos
		seq.Children = children
		elem.Content = seq
	}
	return elem
}

func (n *NodeBuilder) TransformXMLStartTag(xMLStartTagNode *st.XMLStartTagNode) BLangNode {
	panic("TransformXMLStartTag unimplemented")
}

func (n *NodeBuilder) TransformXMLEndTag(xMLEndTagNode *st.XMLEndTagNode) BLangNode {
	panic("TransformXMLEndTag unimplemented")
}

func (n *NodeBuilder) TransformXMLSimpleName(xMLSimpleNameNode *st.XMLSimpleNameNode) BLangNode {
	panic("TransformXMLSimpleName unimplemented")
}

func (n *NodeBuilder) TransformXMLQualifiedName(xMLQualifiedNameNode *st.XMLQualifiedNameNode) BLangNode {
	panic("TransformXMLQualifiedName unimplemented")
}

func (n *NodeBuilder) TransformXMLEmptyElement(xMLEmptyElementNode *st.XMLEmptyElementNode) BLangNode {
	elem := &BLangXMLElementLiteral{}
	elem.pos = n.getPosition(xMLEmptyElementNode)
	elem.Name = n.xmlNameToString(xMLEmptyElementNode.Name())
	elem.Attrs = n.xmlAttributes(xMLEmptyElementNode.Attributes())
	return elem
}

func (n *NodeBuilder) TransformInterpolation(interpolationNode *st.InterpolationNode) BLangNode {
	n.cx.Unimplemented("xml interpolation not yet supported", n.getPosition(interpolationNode))
	return nil
}

func (n *NodeBuilder) TransformXMLText(xMLTextNode *st.XMLTextNode) BLangNode {
	text := &BLangXMLTextLiteral{}
	text.pos = n.getPosition(xMLTextNode)
	if c := xMLTextNode.Content(); c != nil {
		text.Body = c.Text()
	}
	return text
}

func (n *NodeBuilder) TransformXMLAttribute(xMLAttributeNode *st.XMLAttributeNode) BLangNode {
	attr := &BLangXMLAttribute{}
	attr.pos = n.getPosition(xMLAttributeNode)
	attr.Name = n.xmlNameToString(xMLAttributeNode.AttributeName())
	if valueNode := xMLAttributeNode.Value(); valueNode != nil {
		if transformed := n.TransformXMLAttributeValue(valueNode); transformed != nil {
			if expr, ok := transformed.(BLangExpression); ok {
				attr.Value = expr
			}
		}
	}
	return attr
}

func (n *NodeBuilder) TransformXMLAttributeValue(xMLAttributeValue *st.XMLAttributeValue) BLangNode {
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
	lit := &BLangLiteral{}
	lit.pos = n.getPosition(xMLAttributeValue)
	lit.SetValueType(n.types.getTypeFromTag(TypeTags_STRING).(BType))
	lit.Value = text
	lit.OriginalValue = text
	return lit
}

func (n *NodeBuilder) TransformXMLComment(xMLComment *st.XMLComment) BLangNode {
	c := &BLangXMLCommentLiteral{}
	c.pos = n.getPosition(xMLComment)
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

func (n *NodeBuilder) TransformXMLCDATA(xMLCDATANode *st.XMLCDATANode) BLangNode {
	n.cx.Unimplemented("xml CDATA not yet supported", n.getPosition(xMLCDATANode))
	return nil
}

func (n *NodeBuilder) TransformXMLProcessingInstruction(xMLProcessingInstruction *st.XMLProcessingInstruction) BLangNode {
	pi := &BLangXMLPILiteral{}
	pi.pos = n.getPosition(xMLProcessingInstruction)
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

func (n *NodeBuilder) TransformTableTypeDescriptor(tableTypeDescriptorNode *st.TableTypeDescriptorNode) BLangNode {
	panic("TransformTableTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformTypeParameter(typeParameterNode *st.TypeParameterNode) BLangNode {
	return n.createTypeNode(typeParameterNode.TypeNode()).(BLangNode)
}

func (n *NodeBuilder) TransformKeyTypeConstraint(keyTypeConstraintNode *st.KeyTypeConstraintNode) BLangNode {
	panic("TransformKeyTypeConstraint unimplemented")
}

func (n *NodeBuilder) TransformFunctionTypeDescriptor(functionTypeDescriptorNode *st.FunctionTypeDescriptorNode) BLangNode {
	funcType := &BLangFunctionType{}
	funcType.pos = n.getPosition(functionTypeDescriptorNode)

	if funcSignature := functionTypeDescriptorNode.FunctionSignature(); funcSignature != nil {
		funcType.ParamListPos = diagnostics.NewBuiltinLocation()
		openParen := funcSignature.OpenParenToken()
		closeParen := funcSignature.CloseParenToken()
		if openParen != nil && closeParen != nil && !openParen.IsMissing() && !closeParen.IsMissing() {
			funcType.ParamListPos = n.getPositionRange(openParen, closeParen)
		}

		// Set Parameters
		parameters := funcSignature.Parameters()
		for param := range parameters.Iterator() {
			ftParam := n.createFunctionTypeParam(param)
			if _, isRestParam := param.(*st.RestParameterNode); isRestParam {
				funcType.RestParam = &ftParam
			} else {
				funcType.RequiredParams = append(funcType.RequiredParams, ftParam)
			}
		}

		// Set Return Type
		if retNode := funcSignature.ReturnTypeDesc(); retNode != nil {
			returnsKeyword := retNode.ReturnsKeyword()
			if returnsKeyword != nil && !returnsKeyword.IsMissing() {
				funcType.SetExplicitReturnTypeDescriptor()
			}
			funcType.ReturnTypeDescriptor = n.createTypeNode(retNode.Type()).(BType)
		} else {
			retType := &BLangValueType{TypeKind: TypeKind_NIL}
			retType.pos = diagnostics.NewBuiltinLocation()
			funcType.ReturnTypeDescriptor = retType
		}
	} else {
		funcType.SetAnyFunction()
	}

	qualifierList := functionTypeDescriptorNode.QualifierList()
	for token := range qualifierList.Iterator() {
		switch token.Kind() {
		case st.ISOLATED_KEYWORD:
			funcType.SetIsolated()
		case st.TRANSACTIONAL_KEYWORD:
			funcType.SetTransactional()
		}
	}

	return funcType
}

type typedParameterNode interface {
	st.ParameterNode
	ParamName() st.Token
	TypeName() st.Node
	Annotations() st.NodeList[*st.AnnotationNode]
}

func (n *NodeBuilder) createFunctionTypeParam(param st.ParameterNode) BLangFunctionTypeParam {
	typedParam, ok := param.(typedParameterNode)
	if !ok {
		panic("createFunctionTypeParam: unsupported parameter type")
	}
	paramName := typedParam.ParamName()
	typeName := typedParam.TypeName()
	annotations := typedParam.Annotations()

	ftParam := BLangFunctionTypeParam{}
	ftParam.pos = n.getPosition(param)

	if paramName != nil {
		name := createIdentifierFromToken(n.getPosition(paramName), paramName)
		name.pos = n.getPosition(paramName)
		ftParam.Name = &name
	}

	ftParam.TypeDesc = n.createTypeNode(typeName).(BType)

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

	return ftParam
}

func (n *NodeBuilder) TransformFunctionSignature(functionSignatureNode *st.FunctionSignatureNode) BLangNode {
	panic("TransformFunctionSignature unimplemented")
}

func (n *NodeBuilder) TransformExplicitAnonymousFunctionExpression(anonFuncExprNode *st.ExplicitAnonymousFunctionExpressionNode) BLangNode {
	bLFunction := &BLangFunction{}
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	bLFunction.Name = &ident
	n.populateFuncSignature(bLFunction, anonFuncExprNode.FunctionSignature())
	body := n.TransformSyntaxNode(anonFuncExprNode.FunctionBody()).(FunctionBodyNode)
	bLFunction.Body = body
	bLFunction.pos = n.getPosition(anonFuncExprNode)
	bLFunction.SetAnonymous()
	setFunctionQualifiers(bLFunction, anonFuncExprNode.QualifierList())

	lambdaFunc := &BLangLambdaFunction{Function: bLFunction}
	lambdaFunc.pos = bLFunction.pos
	return lambdaFunc
}

func (n *NodeBuilder) TransformExpressionFunctionBody(expressionFunctionBodyNode *st.ExpressionFunctionBodyNode) BLangNode {
	exprBody := &BLangExprFunctionBody{}
	exprBody.Expr = n.createExpression(expressionFunctionBodyNode.Expression())
	exprBody.pos = n.getPosition(expressionFunctionBodyNode)
	return exprBody
}

func (n *NodeBuilder) TransformTupleTypeDescriptor(tupleTypeDescriptorNode *st.TupleTypeDescriptorNode) BLangNode {
	tupleTypeNode := &BLangTupleTypeNode{
		Members: make([]BLangMemberTypeDesc, 0),
	}

	types := tupleTypeDescriptorNode.MemberTypeDesc()
	for i := 0; i < types.Size(); i += 2 {
		node := types.Get(i)
		if node.Kind() == st.REST_TYPE {
			restDescriptor := node.(*st.RestDescriptorNode)
			tupleTypeNode.Rest = n.createTypeNode(restDescriptor.TypeDescriptor()).(BType)
		} else {
			memberNode := node.(*st.MemberTypeDescriptorNode)
			member := BLangMemberTypeDesc{
				TypeDesc: n.createTypeNode(memberNode.TypeDescriptor()),
			}
			member.pos = n.getPosition(memberNode)
			tupleTypeNode.Members = append(tupleTypeNode.Members, member)
		}
	}
	tupleTypeNode.pos = n.getPosition(tupleTypeDescriptorNode)
	return tupleTypeNode
}

func (n *NodeBuilder) TransformParenthesisedTypeDescriptor(parenthesisedTypeDescriptorNode *st.ParenthesisedTypeDescriptorNode) BLangNode {
	return n.createTypeNode(parenthesisedTypeDescriptorNode.Typedesc()).(BLangNode)
}

func (n *NodeBuilder) TransformExplicitNewExpression(explicitNewBLangExpression *st.ExplicitNewExpressionNode) BLangNode {
	typeInit := &BLangNewExpression{}
	typeInit.pos = n.getPosition(explicitNewBLangExpression)
	typeInit.TypeDescriptor = n.createTypeNode(explicitNewBLangExpression.TypeDescriptor()).(BType)
	if argList := explicitNewBLangExpression.ParenthesizedArgList(); argList != nil {
		args := argList.Arguments()
		for arg := range args.Iterator() {
			typeInit.ArgsExprs = append(typeInit.ArgsExprs, n.createExpression(arg))
		}
	}
	return typeInit
}

func (n *NodeBuilder) TransformImplicitNewExpression(implicitNewBLangExpression *st.ImplicitNewExpressionNode) BLangNode {
	typeInit := &BLangNewExpression{}
	typeInit.pos = n.getPosition(implicitNewBLangExpression)
	if argList := implicitNewBLangExpression.ParenthesizedArgList(); argList != nil {
		args := argList.Arguments()
		for arg := range args.Iterator() {
			typeInit.ArgsExprs = append(typeInit.ArgsExprs, n.createExpression(arg))
		}
	}
	return typeInit
}

func (n *NodeBuilder) TransformParenthesizedArgList(parenthesizedArgList *st.ParenthesizedArgList) BLangNode {
	panic("TransformParenthesizedArgList unimplemented")
}

func (n *NodeBuilder) TransformQueryConstructType(queryConstructTypeNode *st.QueryConstructTypeNode) BLangNode {
	keyword := queryConstructTypeNode.Keyword()
	return &BLangIdentifier{
		Value: keyword.Text(),
		bLangNodeBase: bLangNodeBase{
			pos: n.getPosition(queryConstructTypeNode),
		},
	}
}

func (n *NodeBuilder) TransformFromClause(fromClauseNode *st.FromClauseNode) BLangNode {
	fromClause := &BLangFromClause{}
	fromClause.pos = n.getPosition(fromClauseNode)
	fromClause.SetCollection(n.createActionOrExpression(fromClauseNode.Expression()))
	bindingPatternNode := fromClauseNode.TypedBindingPattern()
	fromClause.SetVariableDefinitionNode(n.createBLangVarDef(n.getPosition(bindingPatternNode), bindingPatternNode,
		nil, nil))
	fromClause.VariableDefinitionNode.Var.SetFinal()
	fromClause.IsDeclaredWithVarFlag = isDeclaredWithVar(bindingPatternNode.TypeDescriptor())
	return fromClause
}

func (n *NodeBuilder) TransformWhereClause(whereClauseNode *st.WhereClauseNode) BLangNode {
	whereClause := &BLangWhereClause{}
	whereClause.pos = n.getPosition(whereClauseNode)
	whereClause.Expression = n.createExpression(whereClauseNode.Expression())
	return whereClause
}

func (n *NodeBuilder) TransformLetClause(letClauseNode *st.LetClauseNode) BLangNode {
	letClause := &BLangLetClause{}
	letClause.pos = n.getPosition(letClauseNode)
	letVarDeclarations := letClauseNode.LetVarDeclarations()
	letClause.LetVarDeclarations = make([]BLangSimpleVariableDef, 0, letVarDeclarations.Size())
	for letVar := range letVarDeclarations.Iterator() {
		varDef := n.TransformLetVariableDeclaration(letVar).(*BLangSimpleVariableDef)
		letClause.LetVarDeclarations = append(letClause.LetVarDeclarations, *varDef)
	}
	return letClause
}

func (n *NodeBuilder) TransformJoinClause(joinClauseNode *st.JoinClauseNode) BLangNode {
	joinClause := &BLangJoinClause{}
	joinClause.pos = n.getPosition(joinClauseNode)
	joinClause.SetCollection(n.createActionOrExpression(joinClauseNode.Expression()))
	bindingPatternNode := joinClauseNode.TypedBindingPattern()
	joinClause.SetVariableDefinitionNode(
		n.createBLangVarDef(n.getPosition(bindingPatternNode), bindingPatternNode, nil, nil),
	)
	joinClause.VariableDefinitionNode.Var.SetFinal()
	joinClause.IsDeclaredWithVarFlag = isDeclaredWithVar(bindingPatternNode.TypeDescriptor())
	joinClause.IsOuterJoinFlag = joinClauseNode.OuterKeyword() != nil
	if onClauseNode := joinClauseNode.JoinOnCondition(); onClauseNode != nil {
		joinClause.OnClause = *n.TransformOnClause(onClauseNode).(*BLangOnClause)
	}
	return joinClause
}

func (n *NodeBuilder) TransformOnClause(onClauseNode *st.OnClauseNode) BLangNode {
	onClause := &BLangOnClause{}
	onClause.pos = n.getPosition(onClauseNode)
	onClause.SetOnExpression(n.createExpression(onClauseNode.OnExpression()))
	onClause.SetEqualsExpression(n.createExpression(onClauseNode.EqualsExpression()))
	return onClause
}

func (n *NodeBuilder) TransformLimitClause(limitClauseNode *st.LimitClauseNode) BLangNode {
	limitClause := &BLangLimitClause{}
	limitClause.pos = n.getPosition(limitClauseNode)
	limitClause.SetExpression(n.createExpression(limitClauseNode.Expression()))
	return limitClause
}

func (n *NodeBuilder) TransformOnConflictClause(onConflictClauseNode *st.OnConflictClauseNode) BLangNode {
	onConflictClause := &BLangOnConflictClause{}
	onConflictClause.pos = n.getPosition(onConflictClauseNode)
	onConflictClause.SetExpression(n.createExpression(onConflictClauseNode.Expression()))
	return onConflictClause
}

func (n *NodeBuilder) TransformQueryPipeline(queryPipelineNode *st.QueryPipelineNode) BLangNode {
	panic("TransformQueryPipeline unimplemented")
}

func (n *NodeBuilder) addQueryPipelineClauses(queryClauseAdder interface{ AddQueryClause(Node) }, queryPipeline *st.QueryPipelineNode) {
	if queryPipeline == nil || queryPipeline.FromClause() == nil {
		return
	}

	queryClauseAdder.AddQueryClause(n.TransformSyntaxNode(queryPipeline.FromClause()))

	intermediateClauses := queryPipeline.IntermediateClauses()
	for i := 0; i < intermediateClauses.Size(); i++ {
		clause := intermediateClauses.Get(i)
		switch clause.Kind() {
		case st.FROM_CLAUSE, st.JOIN_CLAUSE, st.LET_CLAUSE, st.WHERE_CLAUSE,
			st.GROUP_BY_CLAUSE, st.LIMIT_CLAUSE, st.ORDER_BY_CLAUSE:
			queryClauseAdder.AddQueryClause(n.TransformSyntaxNode(clause))
		default:
			n.cx.Unimplemented("only from, join, let, where, group by, order by, and limit query clauses are supported for now", n.getPosition(clause))
		}
	}
}

func (n *NodeBuilder) TransformSelectClause(selectClauseNode *st.SelectClauseNode) BLangNode {
	selectClause := &BLangSelectClause{}
	selectClause.pos = n.getPosition(selectClauseNode)
	selectClause.SetExpression(n.createActionOrExpression(selectClauseNode.Expression()))
	return selectClause
}

func (n *NodeBuilder) TransformCollectClause(collectClauseNode *st.CollectClauseNode) BLangNode {
	collectClause := &BLangCollectClause{
		NonGroupingKeys: &balCommon.UnorderedSet[string]{},
	}
	collectClause.pos = n.getPosition(collectClauseNode)
	collectClause.SetExpression(n.createExpression(collectClauseNode.Expression()))
	return collectClause
}

func (n *NodeBuilder) TransformQueryExpression(queryBLangExpression *st.QueryExpressionNode) BLangNode {
	queryExpr := &BLangQueryExpr{}
	queryExpr.pos = n.getPosition(queryBLangExpression)

	if constructType := queryBLangExpression.QueryConstructType(); constructType != nil {
		switch constructType.Keyword().Text() {
		case string(TypeKind_MAP):
			queryExpr.QueryConstructType = TypeKind_MAP
		default:
			n.cx.Unimplemented("only map query construct type is supported for now", n.getPosition(constructType))
		}
	}

	queryPipeline := queryBLangExpression.QueryPipeline()
	n.addQueryPipelineClauses(queryExpr, queryPipeline)

	resultClause := queryBLangExpression.ResultClause()
	if resultClause != nil && (resultClause.Kind() == st.SELECT_CLAUSE || resultClause.Kind() == st.COLLECT_CLAUSE) {
		queryExpr.AddQueryClause(n.TransformSyntaxNode(resultClause))
	} else if resultClause != nil {
		n.cx.Unimplemented("only select/collect result clauses are supported for now", n.getPosition(resultClause))
	}

	if queryBLangExpression.OnConflictClause() != nil {
		queryExpr.AddQueryClause(n.TransformSyntaxNode(queryBLangExpression.OnConflictClause()))
	}

	return queryExpr
}

func (n *NodeBuilder) TransformQueryAction(queryActionNode *st.QueryActionNode) BLangNode {
	queryAction := &BLangQueryAction{}
	queryAction.pos = n.getPosition(queryActionNode)

	n.addQueryPipelineClauses(queryAction, queryActionNode.QueryPipeline())

	doClause := &BLangDoClause{}
	doClause.pos = n.getPosition(queryActionNode)
	if blockStmt := queryActionNode.BlockStatement(); blockStmt != nil {
		doClause.Body = n.TransformBlockStatement(blockStmt).(*BLangBlockStmt)
	}
	queryAction.SetDoClause(doClause)
	return queryAction
}

func (n *NodeBuilder) TransformIntersectionTypeDescriptor(intersectionTypeDescriptorNode *st.IntersectionTypeDescriptorNode) BLangNode {
	lhs := intersectionTypeDescriptorNode.LeftTypeDesc()
	rhs := intersectionTypeDescriptorNode.RightTypeDesc()
	bLIntersectionType := &BLangIntersectionTypeNode{
		lhs: TypeData{
			TypeDescriptor: n.createTypeNode(lhs),
		},
		rhs: TypeData{
			TypeDescriptor: n.createTypeNode(rhs),
		},
	}
	bLIntersectionType.pos = n.getPosition(intersectionTypeDescriptorNode)
	return bLIntersectionType
}

func (n *NodeBuilder) TransformImplicitAnonymousFunctionParameters(implicitAnonymousFunctionParameters *st.ImplicitAnonymousFunctionParameters) BLangNode {
	panic("TransformImplicitAnonymousFunctionParameters unimplemented")
}

func (n *NodeBuilder) TransformImplicitAnonymousFunctionExpression(node *st.ImplicitAnonymousFunctionExpressionNode) BLangNode {
	fn := &BLangFunction{}
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	fn.Name = &ident
	fn.pos = n.getPosition(node)
	fn.SetAnonymous()

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
	fn.RequiredParams = make([]BLangSimpleVariable, len(paramNodes))
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
		fn.RequiredParams[i].pos = n.getPosition(param)
		fn.RequiredParams[i].SetRequiredParam()
	}
	fn.Body = &BLangExprFunctionBody{
		Expr: n.createExpression(node.Expression()),
	}
	fn.Body.(*BLangExprFunctionBody).pos = n.getPosition(node.Expression())

	lambda := &BLangLambdaFunction{Function: fn}
	lambda.SetInferredParams()
	lambda.pos = fn.pos
	return lambda
}

func (n *NodeBuilder) TransformStartAction(startActionNode *st.StartActionNode) BLangNode {
	panic("TransformStartAction unimplemented")
}

func (n *NodeBuilder) TransformFlushAction(flushActionNode *st.FlushActionNode) BLangNode {
	panic("TransformFlushAction unimplemented")
}

func (n *NodeBuilder) TransformSingletonTypeDescriptor(singletonTypeDescriptorNode *st.SingletonTypeDescriptorNode) BLangNode {
	bLFiniteTypeNode := &BLangFiniteTypeNode{}
	bLFiniteTypeNode.pos = n.getPosition(singletonTypeDescriptorNode)
	bLFiniteTypeNode.ValueSpace = append(bLFiniteTypeNode.ValueSpace, n.createExpression(singletonTypeDescriptorNode.SimpleContExprNode()))
	return bLFiniteTypeNode
}

func (n *NodeBuilder) TransformMethodDeclaration(methodDeclarationNode *st.MethodDeclarationNode) BLangNode {
	panic("TransformMethodDeclaration unimplemented")
}

func (n *NodeBuilder) TransformTypedBindingPattern(typedBindingPatternNode *st.TypedBindingPatternNode) BLangNode {
	panic("TransformTypedBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformCaptureBindingPattern(captureBindingPatternNode *st.CaptureBindingPatternNode) BLangNode {
	panic("TransformCaptureBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformWildcardBindingPattern(wildcardBindingPatternNode *st.WildcardBindingPatternNode) BLangNode {
	bLWildCardBindingPattern := &BLangWildCardBindingPattern{}
	bLWildCardBindingPattern.pos = n.getPosition(wildcardBindingPatternNode)
	return bLWildCardBindingPattern
}

func (n *NodeBuilder) TransformListBindingPattern(listBindingPatternNode *st.ListBindingPatternNode) BLangNode {
	panic("TransformListBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformMappingBindingPattern(mappingBindingPatternNode *st.MappingBindingPatternNode) BLangNode {
	panic("TransformMappingBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformFieldBindingPatternFull(fieldBindingPatternFullNode *st.FieldBindingPatternFullNode) BLangNode {
	panic("TransformFieldBindingPatternFull unimplemented")
}

func (n *NodeBuilder) TransformFieldBindingPatternVarname(fieldBindingPatternVarnameNode *st.FieldBindingPatternVarnameNode) BLangNode {
	panic("TransformFieldBindingPatternVarname unimplemented")
}

func (n *NodeBuilder) TransformRestBindingPattern(restBindingPatternNode *st.RestBindingPatternNode) BLangNode {
	panic("TransformRestBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformErrorBindingPattern(errorBindingPatternNode *st.ErrorBindingPatternNode) BLangNode {
	panic("TransformErrorBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformNamedArgBindingPattern(namedArgBindingPatternNode *st.NamedArgBindingPatternNode) BLangNode {
	panic("TransformNamedArgBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformAsyncSendAction(asyncSendActionNode *st.AsyncSendActionNode) BLangNode {
	panic("TransformAsyncSendAction unimplemented")
}

func (n *NodeBuilder) TransformSyncSendAction(syncSendActionNode *st.SyncSendActionNode) BLangNode {
	panic("TransformSyncSendAction unimplemented")
}

func (n *NodeBuilder) TransformReceiveAction(receiveActionNode *st.ReceiveActionNode) BLangNode {
	panic("TransformReceiveAction unimplemented")
}

func (n *NodeBuilder) TransformReceiveFields(receiveFieldsNode *st.ReceiveFieldsNode) BLangNode {
	panic("TransformReceiveFields unimplemented")
}

func (n *NodeBuilder) TransformAlternateReceive(alternateReceiveNode *st.AlternateReceiveNode) BLangNode {
	panic("TransformAlternateReceive unimplemented")
}

func (n *NodeBuilder) TransformRestDescriptor(restDescriptorNode *st.RestDescriptorNode) BLangNode {
	panic("TransformRestDescriptor unimplemented")
}

func (n *NodeBuilder) TransformDoubleGTToken(doubleGTTokenNode *st.DoubleGTTokenNode) BLangNode {
	panic("TransformDoubleGTToken unimplemented")
}

func (n *NodeBuilder) TransformTrippleGTToken(trippleGTTokenNode *st.TrippleGTTokenNode) BLangNode {
	panic("TransformTrippleGTToken unimplemented")
}

func (n *NodeBuilder) TransformWaitAction(waitActionNode *st.WaitActionNode) BLangNode {
	panic("TransformWaitAction unimplemented")
}

func (n *NodeBuilder) TransformWaitFieldsList(waitFieldsListNode *st.WaitFieldsListNode) BLangNode {
	panic("TransformWaitFieldsList unimplemented")
}

func (n *NodeBuilder) TransformWaitField(waitFieldNode *st.WaitFieldNode) BLangNode {
	panic("TransformWaitField unimplemented")
}

func (n *NodeBuilder) TransformAnnotAccessExpression(annotAccessBLangExpression *st.AnnotAccessExpressionNode) BLangNode {
	expr := &BLangAnnotAccessExpr{}
	expr.Expr = n.createExpression(annotAccessBLangExpression.Expression())
	nameReference := n.createBLangNameReference(annotAccessBLangExpression.AnnotTagReference())
	expr.PkgAlias = nameReference[0]
	expr.AnnotationName = nameReference[1]
	expr.SetPosition(n.getPosition(annotAccessBLangExpression))
	return expr
}

func (n *NodeBuilder) TransformOptionalFieldAccessExpression(optionalFieldAccessBLangExpression *st.OptionalFieldAccessExpressionNode) BLangNode {
	fieldName := optionalFieldAccessBLangExpression.FieldName()
	if fieldName.Kind() == st.QUALIFIED_NAME_REFERENCE {
		// @cleanup we should replace all these panics with proper internal errors. Need to problem is with return value
		// this should be detected by parser
		panic("TransformOptionalFieldAccessExpression: QUALIFIED_NAME_REFERENCE expected")
	}

	bLFieldBasedAccess := &BLangFieldBaseAccess{}
	bLFieldBasedAccess.SetOptionalAccess()
	simpleNameRef := fieldName.(*st.SimpleNameReferenceNode)
	bLFieldBasedAccess.Field = n.createIdentifierNodeFromToken(n.getPosition(optionalFieldAccessBLangExpression.FieldName()), simpleNameRef.Name())

	containerExpr := optionalFieldAccessBLangExpression.Expression()
	if containerExpr.Kind() == st.BRACED_EXPRESSION {
		bracedExpr := containerExpr.(*st.BracedExpressionNode)
		bLFieldBasedAccess.Expr = n.createExpression(bracedExpr.Expression())
	} else {
		bLFieldBasedAccess.Expr = n.createExpression(containerExpr)
	}

	bLFieldBasedAccess.pos = n.getPosition(optionalFieldAccessBLangExpression)
	return bLFieldBasedAccess
}

func (n *NodeBuilder) TransformConditionalExpression(conditionalBLangExpression *st.ConditionalExpressionNode) BLangNode {
	panic("TransformConditionalExpression unimplemented")
}

func (n *NodeBuilder) TransformEnumDeclaration(enumDeclarationNode *st.EnumDeclarationNode) BLangNode {
	publicQualifier := false
	qualifier := enumDeclarationNode.Qualifier()
	if qualifier != nil && qualifier.Kind() == st.PUBLIC_KEYWORD {
		publicQualifier = true
	}

	memberNodes := enumDeclarationNode.EnumMemberList()
	memberTypeNodes := make([]TypeDescriptor, 0)
	for memberNode := range memberNodes.Iterator() {
		if memberNode.Kind() != st.ENUM_MEMBER {
			continue
		}
		enumMember := memberNode.(*st.EnumMemberNode)
		if enumMember.Identifier() == nil || enumMember.Identifier().IsMissing() {
			n.cx.InternalError("missing enum member identifier", n.getPosition(enumMember))
			continue
		}
		constantNode := n.transformEnumMember(enumMember, publicQualifier)
		if n.currentCompUnit == nil {
			n.cx.InternalError("enum constants can only be added at module level", n.getPosition(enumMember))
			continue
		}
		n.currentCompUnit.AddTopLevelNode(constantNode)
		memberTypeNodes = append(memberTypeNodes, n.createTypeNode(enumMember.Identifier()))
	}

	typeDef := NewBLangTypeDefinition()
	typeDef.pos = n.getPositionWithoutMetadata(enumDeclarationNode)
	if publicQualifier {
		typeDef.SetPublic()
	}

	identifierPos := n.getPosition(enumDeclarationNode.Identifier())
	identifier := createIdentifierFromToken(identifierPos, enumDeclarationNode.Identifier())
	typeDef.Name = &identifier

	if len(memberTypeNodes) > 0 {
		current := memberTypeNodes[0]
		for i := 1; i < len(memberTypeNodes); i++ {
			unionType := &BLangUnionTypeNode{
				lhs: TypeData{TypeDescriptor: current},
				rhs: TypeData{TypeDescriptor: memberTypeNodes[i]},
			}
			unionType.pos = typeDef.pos
			current = unionType
		}
		typeDef.SetTypeData(TypeData{TypeDescriptor: current})
	} else {
		neverType := &BLangValueType{TypeKind: TypeKind_NEVER}
		neverType.pos = diagnostics.NewBuiltinLocation()
		typeDef.SetTypeData(TypeData{TypeDescriptor: neverType})
		n.cx.SyntaxError("missing enum member", typeDef.Name.GetPosition())
	}

	metadata := enumDeclarationNode.Metadata()
	if metadata != nil && !metadata.IsMissing() {
		docString := getDocumentationString(metadata)
		typeDef.markdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(docString)
	}

	return typeDef
}

func (n *NodeBuilder) TransformEnumMember(enumMemberNode *st.EnumMemberNode) BLangNode {
	return n.transformEnumMember(enumMemberNode, false)
}

func (n *NodeBuilder) transformEnumMember(enumMemberNode *st.EnumMemberNode, publicQualifier bool) *BLangConstant {
	constantNode := createConstantNode()
	constantNode.pos = n.getPositionWithoutMetadata(enumMemberNode)
	if publicQualifier {
		constantNode.SetPublic()
	}

	identifierPos := n.getPosition(enumMemberNode.Identifier())
	identifier := createIdentifierFromToken(identifierPos, enumMemberNode.Identifier())
	constantNode.Name = &identifier

	if exprNode := enumMemberNode.ConstExprNode(); exprNode != nil {
		constantNode.Expr = n.createExpression(exprNode)
	} else {
		constantNode.Expr = n.createSimpleLiteral(enumMemberNode.Identifier()).(BLangExpression)
	}

	stringType := &BLangValueType{TypeKind: TypeKind_STRING}
	stringType.pos = diagnostics.NewBuiltinLocation()
	constantNode.SetTypeNode(stringType)

	metadata := enumMemberNode.Metadata()
	if metadata != nil && !metadata.IsMissing() {
		docString := getDocumentationString(metadata)
		constantNode.MarkdownDocumentationAttachment = n.createMarkdownDocumentationAttachment(docString)
	}

	return constantNode
}

func (n *NodeBuilder) TransformArrayTypeDescriptor(arrayTypeDescriptorNode *st.ArrayTypeDescriptorNode) BLangNode {
	position := n.getPosition(arrayTypeDescriptorNode)
	dimensionNodes := arrayTypeDescriptorNode.Dimensions()
	dimensionSize := dimensionNodes.Size()
	var sizes []BLangExpression

	for i := 0; i < dimensionSize; i++ {
		dimensionNode := dimensionNodes.Get(i)
		if dimensionNode.ArrayLength() == nil {
			sizes = append(sizes, nil)
		} else {
			sizes = append(sizes, n.createExpression(dimensionNode.ArrayLength()))
		}
	}
	dimensionSize = len(sizes)

	arrayTypeNode := &BLangArrayType{}
	arrayTypeNode.pos = position
	arrayTypeNode.Elemtype = TypeData{
		TypeDescriptor: n.createTypeNode(arrayTypeDescriptorNode.MemberTypeDesc()),
	}
	arrayTypeNode.Dimensions = dimensionSize
	arrayTypeNode.Sizes = sizes
	return arrayTypeNode
}

func (n *NodeBuilder) TransformArrayDimension(arrayDimensionNode *st.ArrayDimensionNode) BLangNode {
	panic("TransformArrayDimension unimplemented")
}

func (n *NodeBuilder) TransformTransactionStatement(transactionStatementNode *st.TransactionStatementNode) BLangNode {
	panic("TransformTransactionStatement unimplemented")
}

func (n *NodeBuilder) TransformRollbackStatement(rollbackStatementNode *st.RollbackStatementNode) BLangNode {
	panic("TransformRollbackStatement unimplemented")
}

func (n *NodeBuilder) TransformRetryStatement(retryStatementNode *st.RetryStatementNode) BLangNode {
	panic("TransformRetryStatement unimplemented")
}

func (n *NodeBuilder) TransformCommitAction(commitActionNode *st.CommitActionNode) BLangNode {
	panic("TransformCommitAction unimplemented")
}

func (n *NodeBuilder) TransformTransactionalExpression(transactionalBLangExpression *st.TransactionalExpressionNode) BLangNode {
	panic("TransformTransactionalExpression unimplemented")
}

func (n *NodeBuilder) TransformByteArrayLiteral(byteArrayLiteralNode *st.ByteArrayLiteralNode) BLangNode {
	panic("TransformByteArrayLiteral unimplemented")
}

func (n *NodeBuilder) TransformXMLFilterExpression(xMLFilterBLangExpression *st.XMLFilterExpressionNode) BLangNode {
	panic("TransformXMLFilterExpression unimplemented")
}

func (n *NodeBuilder) TransformXMLStepExpression(xMLStepBLangExpression *st.XMLStepExpressionNode) BLangNode {
	panic("TransformXMLStepExpression unimplemented")
}

func (n *NodeBuilder) TransformXMLNamePatternChaining(xMLNamePatternChainingNode *st.XMLNamePatternChainingNode) BLangNode {
	panic("TransformXMLNamePatternChaining unimplemented")
}

func (n *NodeBuilder) TransformXMLStepIndexedExtend(xMLStepIndexedExtendNode *st.XMLStepIndexedExtendNode) BLangNode {
	panic("TransformXMLStepIndexedExtend unimplemented")
}

func (n *NodeBuilder) TransformXMLStepMethodCallExtend(xMLStepMethodCallExtendNode *st.XMLStepMethodCallExtendNode) BLangNode {
	panic("TransformXMLStepMethodCallExtend unimplemented")
}

func (n *NodeBuilder) TransformXMLAtomicNamePattern(xMLAtomicNamePatternNode *st.XMLAtomicNamePatternNode) BLangNode {
	panic("TransformXMLAtomicNamePattern unimplemented")
}

func (n *NodeBuilder) TransformTypeReferenceTypeDesc(typeReferenceTypeDescNode *st.TypeReferenceTypeDescNode) BLangNode {
	panic("TransformTypeReferenceTypeDesc unimplemented")
}

func (n *NodeBuilder) TransformMatchStatement(matchStatementNode *st.MatchStatementNode) BLangNode {
	matchStatement := &BLangMatchStatement{}
	matchStmtExpr := n.createExpression(matchStatementNode.Condition())
	matchStatement.Expr = matchStmtExpr

	matchClauses := matchStatementNode.MatchClauses()
	for matchClauseNode := range matchClauses.Iterator() {
		bLangMatchClause := &BLangMatchClause{}
		bLangMatchClause.pos = n.getPosition(matchClauseNode)

		// Handle match guard
		if matchClauseNode.MatchGuard() != nil {
			matchGuardNode := matchClauseNode.MatchGuard()
			bLangMatchClause.Guard = n.createExpression(matchGuardNode.Expression())
		}

		// Handle match patterns
		matchPatterns := matchClauseNode.MatchPatterns()
		for matchPattern := range matchPatterns.Iterator() {
			bLangMatchPattern := n.transformMatchPattern(matchPattern, matchStmtExpr)
			if bLangMatchPattern != nil {
				bLangMatchClause.Patterns = append(bLangMatchClause.Patterns, bLangMatchPattern)
			}
		}

		// Handle block statement
		bLangMatchClause.Body = *n.TransformBlockStatement(matchClauseNode.BlockStatement()).(*BLangBlockStmt)

		matchStatement.MatchClauses = append(matchStatement.MatchClauses, *bLangMatchClause)
	}

	matchStatement.pos = n.getPosition(matchStatementNode)
	return matchStatement
}

func (n *NodeBuilder) transformMatchPattern(matchPattern st.Node, matchStmtExpr BLangExpression) BLangMatchPattern {
	matchPatternPos := n.getPosition(matchPattern)
	kind := matchPattern.Kind()

	switch kind {
	case st.SIMPLE_NAME_REFERENCE:
		nameRef := matchPattern.(*st.SimpleNameReferenceNode)
		if nameRef.Name().Text() == "_" {
			bLangWildCard := &BLangWildCardMatchPattern{}
			bLangWildCard.pos = matchPatternPos
			return bLangWildCard
		}
		bLangConstPattern := &BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.pos = matchPatternPos
		return bLangConstPattern

	case st.IDENTIFIER_TOKEN:
		idToken := matchPattern.(st.Token)
		if idToken.Text() == "_" {
			bLangWildCard := &BLangWildCardMatchPattern{}
			bLangWildCard.pos = matchPatternPos
			return bLangWildCard
		}
		bLangConstPattern := &BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.pos = matchPatternPos
		return bLangConstPattern

	case st.NUMERIC_LITERAL,
		st.STRING_LITERAL,
		st.QUALIFIED_NAME_REFERENCE,
		st.NULL_LITERAL,
		st.NIL_LITERAL,
		st.BOOLEAN_LITERAL,
		st.UNARY_EXPRESSION:
		bLangConstPattern := &BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.pos = matchPatternPos
		return bLangConstPattern

	case st.PIPE_TOKEN, st.COMMA_TOKEN:
		// Skip separator tokens in match pattern lists
		return nil

	default:
		n.cx.InternalError(fmt.Sprintf("unexpected match pattern kind: %v", kind), matchPatternPos)
		return nil
	}
}

func (n *NodeBuilder) TransformMatchClause(matchClauseNode *st.MatchClauseNode) BLangNode {
	panic("TransformMatchClause unimplemented")
}

func (n *NodeBuilder) TransformMatchGuard(matchGuardNode *st.MatchGuardNode) BLangNode {
	panic("TransformMatchGuard unimplemented")
}

func (n *NodeBuilder) TransformDistinctTypeDescriptor(distinctTypeDescriptorNode *st.DistinctTypeDescriptorNode) BLangNode {
	n.cx.Unimplemented("anonymous distinct types not supported", n.getPosition(distinctTypeDescriptorNode))
	neverType := &BLangValueType{TypeKind: TypeKind_NEVER}
	neverType.pos = n.getPosition(distinctTypeDescriptorNode)
	return neverType
}

func (n *NodeBuilder) TransformListMatchPattern(listMatchPatternNode *st.ListMatchPatternNode) BLangNode {
	panic("TransformListMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformRestMatchPattern(restMatchPatternNode *st.RestMatchPatternNode) BLangNode {
	panic("TransformRestMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformMappingMatchPattern(mappingMatchPatternNode *st.MappingMatchPatternNode) BLangNode {
	panic("TransformMappingMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformFieldMatchPattern(fieldMatchPatternNode *st.FieldMatchPatternNode) BLangNode {
	panic("TransformFieldMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformErrorMatchPattern(errorMatchPatternNode *st.ErrorMatchPatternNode) BLangNode {
	panic("TransformErrorMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformNamedArgMatchPattern(namedArgMatchPatternNode *st.NamedArgMatchPatternNode) BLangNode {
	panic("TransformNamedArgMatchPattern unimplemented")
}

// Helper functions for markdown documentation transformation

func (n *NodeBuilder) addReferencesAndReturnDocumentationText(references *[]BLangMarkdownReferenceDocumentation, docElements st.NodeList[st.Node]) string {
	var docText strings.Builder
	for i := 0; i < docElements.Size(); i++ {
		element := docElements.Get(i)
		if element.Kind() == st.BALLERINA_NAME_REFERENCE {
			bLangRefDoc := &BLangMarkdownReferenceDocumentation{}
			balNameRefNode := element.(*st.BallerinaNameReferenceNode)

			bLangRefDoc.pos = n.getPosition(balNameRefNode)

			startBacktick := balNameRefNode.StartBacktick()
			backtickContent := balNameRefNode.NameReference()
			endBacktick := balNameRefNode.EndBacktick()

			contentString := ""
			if backtickContent != nil && !backtickContent.IsMissing() {
				// Use InternalNode() to get STNode and convert to source code
				contentString = st.ToSourceCode(backtickContent.InternalNode())
			}
			bLangRefDoc.ReferenceName = contentString
			bLangRefDoc.Type = DocumentationReferenceType("BACKTICK_CONTENT")

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

func (n *NodeBuilder) transformDocumentationBacktickContent(backtickContent st.Node, bLangRefDoc *BLangMarkdownReferenceDocumentation) {
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

func (n *NodeBuilder) transformCodeBlock(documentationLines *[]BLangMarkdownDocumentationLine, codeBlockNode *st.MarkdownCodeBlockNode) {
	bLangDocLine := BLangMarkdownDocumentationLine{}

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
	bLangDocLine.pos = n.getPosition(codeBlockNode.StartLineHashToken())
	*documentationLines = append(*documentationLines, bLangDocLine)
}

func (n *NodeBuilder) stringToRefType(refTypeName string) DocumentationReferenceType {
	switch refTypeName {
	case "type":
		return DocumentationReferenceType("TYPE")
	case "service":
		return DocumentationReferenceType("SERVICE")
	case "variable":
		return DocumentationReferenceType("VARIABLE")
	case "var":
		return DocumentationReferenceType("VAR")
	case "annotation":
		return DocumentationReferenceType("ANNOTATION")
	case "module":
		return DocumentationReferenceType("MODULE")
	case "function":
		return DocumentationReferenceType("FUNCTION")
	case "parameter":
		return DocumentationReferenceType("PARAMETER")
	case "const":
		return DocumentationReferenceType("CONST")
	default:
		return DocumentationReferenceType("BACKTICK_CONTENT")
	}
}

func (n *NodeBuilder) stringStartsWithSingleQuote(s string) bool {
	return len(s) > 0 && s[0] == '\''
}

func (n *NodeBuilder) trimLeftAtMostOne(text string) string {
	countToStrip := 0
	if len(text) > 0 && (text[0] == ' ' || text[0] == '\t' || text[0] == '\n' || text[0] == '\r') {
		countToStrip = 1
	}
	if countToStrip > 0 && len(text) > countToStrip {
		return text[countToStrip:]
	}
	return text
}

func (n *NodeBuilder) TransformOrderByClause(orderByClauseNode *st.OrderByClauseNode) BLangNode {
	orderByClause := &BLangOrderByClause{}
	orderByClause.pos = n.getPosition(orderByClauseNode)

	orderKeys := orderByClauseNode.OrderKey()
	orderByClause.OrderByKeyList = make([]BLangOrderKey, 0, orderKeys.Size())
	for orderKey := range orderKeys.Iterator() {
		keyNode, ok := n.TransformOrderKey(orderKey).(*BLangOrderKey)
		if !ok {
			panic("expected BLangOrderKey")
		}
		orderByClause.OrderByKeyList = append(orderByClause.OrderByKeyList, *keyNode)
	}
	return orderByClause
}

func (n *NodeBuilder) TransformOrderKey(orderKeyNode *st.OrderKeyNode) BLangNode {
	orderKey := &BLangOrderKey{}
	orderKey.pos = n.getPosition(orderKeyNode)
	orderKey.Expression = n.createExpression(orderKeyNode.Expression())
	if dir := orderKeyNode.OrderDirection(); dir != nil && dir.Kind() == st.DESCENDING_KEYWORD {
		orderKey.IsDescending = true
	} else {
		orderKey.IsDescending = false
	}
	return orderKey
}

func (n *NodeBuilder) TransformGroupByClause(groupByClauseNode *st.GroupByClauseNode) BLangNode {
	groupByClause := &BLangGroupByClause{
		NonGroupingKeys: &balCommon.UnorderedSet[string]{},
	}
	groupByClause.pos = n.getPosition(groupByClauseNode)

	groupingKeys := groupByClauseNode.GroupingKey()
	for node := range groupingKeys.Iterator() {
		if node.Kind() == st.COMMA_TOKEN {
			continue
		}
		groupingKey := &BLangGroupingKey{}
		groupingKey.pos = n.getPosition(node)
		if node.Kind() == st.SIMPLE_NAME_REFERENCE || node.Kind() == st.IDENTIFIER_TOKEN {
			varRef, ok := n.createExpression(node).(*BLangSimpleVarRef)
			if !ok {
				panic("expected grouping key variable reference to be a simple variable reference")
			}
			groupingKey.SetGroupingKey(varRef)
		} else {
			keyNode, ok := n.TransformGroupingKeyVarDeclaration(node.(*st.GroupingKeyVarDeclarationNode)).(*BLangGroupingKey)
			if !ok {
				panic("expected grouping key declaration to produce a BLangGroupingKey")
			}
			groupingKey = keyNode
		}
		groupByClause.AddGroupingKey(groupingKey)
	}
	return groupByClause
}

func (n *NodeBuilder) TransformGroupingKeyVarDeclaration(groupingKeyVarDeclarationNode *st.GroupingKeyVarDeclarationNode) BLangNode {
	pos := n.getPosition(groupingKeyVarDeclarationNode)
	groupingKey := &BLangGroupingKey{}
	groupingKey.pos = pos

	variableNode := n.getBLangVariableNode(groupingKeyVarDeclarationNode.SimpleBindingPattern(), pos)
	simpleVar, ok := variableNode.(*BLangSimpleVariable)
	if !ok {
		panic("expected grouping key declaration to create a simple variable reference")
	}
	simpleVar.SetPosition(pos)
	simpleVar.SetInitialExpression(n.createExpression(groupingKeyVarDeclarationNode.Expression()))
	simpleVar.SetFinal()

	typeDesc := groupingKeyVarDeclarationNode.TypeDescriptor()
	if isDeclaredWithVar(typeDesc) {
		simpleVar.SetIsDeclaredWithVar(true)
	} else {
		simpleVar.SetTypeNode(n.createTypeNode(typeDesc).(BType))
	}

	varDef := &BLangSimpleVariableDef{}
	varDef.pos = pos
	varDef.SetVariable(simpleVar)
	groupingKey.SetGroupingKey(varDef)
	return groupingKey
}

func (n *NodeBuilder) TransformOnFailClause(onFailClauseNode *st.OnFailClauseNode) BLangNode {
	panic("TransformOnFailClause unimplemented")
}

func (n *NodeBuilder) TransformDoStatement(doStatementNode *st.DoStatementNode) BLangNode {
	panic("TransformDoStatement unimplemented")
}

func (n *NodeBuilder) TransformClassDefinition(classDefinitionNode *st.ClassDefinitionNode) BLangNode {
	blangClass := NewBLangClassDefinition()
	blangClass.pos = n.getPositionWithoutMetadata(classDefinitionNode)

	n.populateMetadata(classDefinitionNode.Metadata(), &blangClass)

	// Set name
	nameIdentifier := createIdentifierFromToken(n.getPosition(classDefinitionNode.ClassName()), classDefinitionNode.ClassName())
	blangClass.Name = &nameIdentifier

	// Handle visibility qualifier
	if visQual := classDefinitionNode.VisibilityQualifier(); visQual != nil {
		if visQual.Kind() == st.PUBLIC_KEYWORD {
			blangClass.SetPublic()
		}
	}

	// Handle class type qualifiers
	n.setClassQualifiers(&blangClass, classDefinitionNode.ClassTypeQualifiers())

	members := n.collectClassDefnMembers(classDefinitionNode.Members())
	blangClass.Fields = members.Fields
	blangClass.Methods = members.Methods
	blangClass.InitFunction = members.InitFunction
	blangClass.ResourceMethods = members.ResourceMethods
	blangClass.unresolvedInclusions = members.UnresolvedInclusions

	return &blangClass
}

func (n *NodeBuilder) setClassQualifiers(blangClass *BLangClassDefinition, qualifiers st.NodeList[st.Token]) {
	for qualifier := range qualifiers.Iterator() {
		switch qualifier.Kind() {
		case st.DISTINCT_KEYWORD:
			blangClass.SetDistinct()
		case st.CLIENT_KEYWORD:
			blangClass.SetClient()
		case st.READONLY_KEYWORD:
			blangClass.SetReadonly()
		case st.SERVICE_KEYWORD:
			blangClass.SetService()
		case st.ISOLATED_KEYWORD:
			blangClass.SetIsolated()
		}
	}
}

func (n *NodeBuilder) transformClassField(objectField *st.ObjectFieldNode) *BLangSimpleVariable {
	bLSimpleVar := createSimpleVariableNode()
	identifier := createIdentifierFromToken(n.getPosition(objectField.FieldName()), objectField.FieldName())
	bLSimpleVar.SetName(&identifier)
	bLSimpleVar.pos = n.getPosition(objectField)
	bLSimpleVar.SetTypeNode(n.createTypeNode(objectField.TypeName()).(BType))

	if vis := objectField.VisibilityQualifier(); vis != nil {
		if vis.Kind() == st.PUBLIC_KEYWORD {
			bLSimpleVar.SetPublic()
		} else if vis.Kind() == st.PRIVATE_KEYWORD {
			bLSimpleVar.SetPrivate()
		}
	}

	qualifiers := objectField.QualifierList()
	for qualifier := range qualifiers.Iterator() {
		if qualifier.Kind() == st.FINAL_KEYWORD {
			bLSimpleVar.SetFinal()
		}
	}

	if expr := objectField.Expression(); expr != nil {
		bLSimpleVar.SetInitialExpression(n.createExpression(expr))
	}

	n.populateMetadata(objectField.Metadata(), bLSimpleVar)
	return bLSimpleVar
}

func (n *NodeBuilder) TransformResourcePathParameter(resourcePathParameterNode *st.ResourcePathParameterNode) BLangNode {
	seg := &BLangResourcePathSegment{}
	switch resourcePathParameterNode.Kind() {
	case st.RESOURCE_PATH_SEGMENT_PARAM:
		seg.Kind = ResourcePathSegmentParam
	case st.RESOURCE_PATH_REST_PARAM:
		seg.Kind = ResourcePathSegmentParamRest
	default:
		n.cx.InternalError(fmt.Sprintf("unexpected resource path parameter node kind: %v", resourcePathParameterNode.Kind()), n.getPosition(resourcePathParameterNode))
	}
	seg.pos = n.getPosition(resourcePathParameterNode)
	nameTok := resourcePathParameterNode.ParamName()
	if nameTok != nil && !nameTok.IsMissing() {
		seg.Name = createIdentifierFromToken(n.getPosition(nameTok), nameTok).Value
	}
	if td := resourcePathParameterNode.TypeDescriptor(); td != nil {
		seg.ParamType = n.createTypeNode(td).(BType)
	}
	return seg
}

func (n *NodeBuilder) createResourceMethodNode(funcDef *st.FunctionDefinition) *BLangResourceMethod {
	rm := &BLangResourceMethod{}
	rm.pos = n.getPositionWithoutMetadata(funcDef)
	rm.Name = n.createIdentifierNodeFromToken(n.getPosition(funcDef.FunctionName()), funcDef.FunctionName())
	setFunctionQualifiersOnBase(&rm.bLangInvokableNodeBase, funcDef.QualifierList())
	rm.SetAttached()
	rm.SetResource()
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, rm.Name.GetValue())
	n.populateFuncSignatureOnBase(&rm.bLangInvokableNodeBase, funcDef.FunctionSignature())
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	body := funcDef.FunctionBody()
	if body == nil {
		rm.SetInterface()
	} else {
		bodyNode := n.TransformSyntaxNode(body).(FunctionBodyNode)
		rm.Body = bodyNode
		if _, ok := bodyNode.(*BLangExternFunctionBody); ok {
			rm.SetNative()
		}
	}
	rm.ResourcePath = n.createResourcePathSegments(funcDef.RelativeResourcePath())
	n.populateMetadata(funcDef.Metadata(), rm)
	return rm
}

func (n *NodeBuilder) createResourcePathSegments(pathNodes st.NodeList[st.Node]) []BLangResourcePathSegment {
	var segments []BLangResourcePathSegment
	for node := range pathNodes.Iterator() {
		switch node.Kind() {
		case st.SLASH_TOKEN:
			continue
		case st.DOT_TOKEN:
			continue
		case st.IDENTIFIER_TOKEN:
			tok := node.(st.Token)
			seg := BLangResourcePathSegment{Kind: ResourcePathSegmentName, Name: tok.Text()}
			seg.pos = n.getPosition(node)
			segments = append(segments, seg)
		case st.RESOURCE_PATH_SEGMENT_PARAM, st.RESOURCE_PATH_REST_PARAM:
			param := node.(*st.ResourcePathParameterNode)
			segments = append(segments, *n.TransformResourcePathParameter(param).(*BLangResourcePathSegment))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource path node kind: %v", node.Kind()), n.getPosition(node))
		}
	}
	return segments
}

func (n *NodeBuilder) TransformRequiredExpression(requiredBLangExpression *st.RequiredExpressionNode) BLangNode {
	panic("TransformRequiredExpression unimplemented")
}

func (n *NodeBuilder) TransformErrorConstructorExpression(errorConstructorBLangExpression *st.ErrorConstructorExpressionNode) BLangNode {
	result := &BLangErrorConstructorExpr{}
	result.pos = n.getPosition(errorConstructorBLangExpression)

	typeRefNode := errorConstructorBLangExpression.TypeReference()
	if typeRefNode != nil {
		typeDesc := n.createTypeNode(typeRefNode)
		if userDefinedType, ok := typeDesc.(*BLangUserDefinedType); ok {
			result.ErrorTypeRef = userDefinedType
		} else {
			n.cx.InternalError("error type reference must be a user-defined type", result.pos)
		}
	}

	arguments := errorConstructorBLangExpression.Arguments()
	positionalArgs := make([]BLangExpression, 0)
	namedArgs := make([]BLangNamedArgsExpression, 0)

	for arg := range arguments.Iterator() {
		switch arg.Kind() {
		case st.POSITIONAL_ARG:
			posArg := arg.(*st.PositionalArgumentNode)
			expr := n.createExpression(posArg.Expression())
			positionalArgs = append(positionalArgs, expr)

		case st.NAMED_ARG:
			namedArgNode := arg.(*st.NamedArgumentNode)
			namedArg := n.TransformNamedArgument(namedArgNode).(*BLangNamedArgsExpression)
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

func (n *NodeBuilder) TransformParameterizedTypeDescriptor(parameterizedTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) BLangNode {
	switch parameterizedTypeDescriptorNode.Kind() {
	case st.ERROR_TYPE_DESC:
		return n.transformErrorTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.TYPEDESC_TYPE_DESC:
		return n.transformTypedescTypeDescriptor(parameterizedTypeDescriptorNode)
	case st.XML_TYPE_DESC:
		return n.transformXMLTypeDescriptor(parameterizedTypeDescriptorNode)
	}
	panic("TransformParameterizedTypeDescriptor supported only for error, typedesc and xml type descriptors")
}

func (n *NodeBuilder) transformTypedescTypeDescriptor(node *st.ParameterizedTypeDescriptorNode) BLangNode {
	typeParamNode := node.TypeParamNode()
	if typeParamNode == nil {
		valueType := &BLangValueType{}
		valueType.pos = n.getPosition(node)
		valueType.TypeKind = TypeKind_TYPEDESC
		return valueType
	}
	constrainedType := &BLangConstrainedType{}
	constrainedType.pos = n.getPosition(node)
	base := &BLangValueType{}
	base.pos = n.getPosition(node)
	base.TypeKind = TypeKind_TYPEDESC
	constrainedType.Type = TypeData{TypeDescriptor: base}
	constraint := typeParamNode.TypeNode()
	if constraint == nil {
		constrainedType.Constraint = TypeData{TypeDescriptor: n.createTypeNode(typeParamNode)}
	} else {
		constrainedType.Constraint = TypeData{TypeDescriptor: n.createTypeNode(constraint)}
	}
	return constrainedType
}

func (n *NodeBuilder) transformXMLTypeDescriptor(parameterizedTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) BLangNode {
	pos := n.getPosition(parameterizedTypeDescriptorNode)
	typeParamNode := parameterizedTypeDescriptorNode.TypeParamNode()
	if typeParamNode == nil {
		valueType := &BLangValueType{}
		valueType.pos = pos
		valueType.TypeKind = TypeKind_XML
		return valueType
	}
	refType := &BLangBuiltInRefTypeNode{
		TypeKind: TypeKind_XML,
	}
	refType.SetPosition(pos)
	constraint := n.createTypeNode(typeParamNode.TypeNode())
	constrainedType := &BLangConstrainedType{
		Type:       TypeData{TypeDescriptor: refType},
		Constraint: TypeData{TypeDescriptor: constraint},
	}
	constrainedType.SetPosition(pos)
	return constrainedType
}

func (n *NodeBuilder) transformErrorTypeDescriptor(errorTypeDescriptorNode *st.ParameterizedTypeDescriptorNode) BLangNode {
	errorType := &BLangErrorTypeNode{}
	errorType.pos = n.getPosition(errorTypeDescriptorNode)

	// Handle optional type parameter
	typeParamNode := errorTypeDescriptorNode.TypeParamNode()
	if typeParamNode != nil {
		errorType.DetailType = TypeData{
			TypeDescriptor: n.createTypeNode(typeParamNode),
		}
	}

	// Check if this is a distinct error type
	parent := errorTypeDescriptorNode.Parent()
	if parent.Kind() == st.DISTINCT_TYPE_DESC {
		errorType.SetDistinct()
	}

	return errorType
}

func (n *NodeBuilder) TransformSpreadMember(spreadMemberNode *st.SpreadMemberNode) BLangNode {
	return n.createExpression(spreadMemberNode.Expression()).(BLangNode)
}

func (n *NodeBuilder) TransformClientResourceAccessAction(node *st.ClientResourceAccessActionNode) BLangNode {
	action := &BLangClientResourceAccessAction{}
	action.pos = n.getPosition(node)
	action.Expr = n.createExpression(node.Expression())
	action.MethodName = "get"
	if methodName := node.MethodName(); methodName != nil {
		nameTok := methodName.Name()
		if nameTok == nil || nameTok.IsMissing() {
			n.cx.InternalError("missing method name token in resource access action", action.pos)
		} else {
			action.MethodName = nameTok.Text()
		}
	}
	nameID := &BLangIdentifier{Value: action.MethodName}
	nameID.SetPosition(action.pos)
	action.Name = nameID
	action.Path = n.createResourceAccessSegments(node.ResourceAccessPath())
	if args := node.Arguments(); args != nil {
		var argExprs []BLangExpression
		argList := args.Arguments()
		for arg := range argList.Iterator() {
			argExprs = append(argExprs, n.createExpression(arg))
		}
		action.ArgExprs = argExprs
	}
	return action
}

func (n *NodeBuilder) createResourceAccessSegments(pathNodes st.NodeList[st.Node]) []BLangResourceAccessSegment {
	var segments []BLangResourceAccessSegment
	for node := range pathNodes.Iterator() {
		switch node.Kind() {
		case st.SLASH_TOKEN, st.DOT_TOKEN:
			continue
		case st.IDENTIFIER_TOKEN:
			tok := node.(st.Token)
			seg := BLangResourceAccessSegment{Kind: ResourceAccessSegmentName, Name: tok.Text()}
			seg.pos = n.getPosition(node)
			segments = append(segments, seg)
		case st.COMPUTED_RESOURCE_ACCESS_SEGMENT:
			computed := node.(*st.ComputedResourceAccessSegmentNode)
			segments = append(segments, *n.TransformComputedResourceAccessSegment(computed).(*BLangResourceAccessSegment))
		case st.RESOURCE_ACCESS_REST_SEGMENT:
			n.cx.Unimplemented("resource access rest segments are not yet supported", n.getPosition(node))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource access segment kind: %v", node.Kind()), n.getPosition(node))
		}
	}
	return segments
}

func (n *NodeBuilder) TransformComputedResourceAccessSegment(node *st.ComputedResourceAccessSegmentNode) BLangNode {
	seg := &BLangResourceAccessSegment{Kind: ResourceAccessSegmentComputed}
	seg.pos = n.getPosition(node)
	seg.Expr = n.createExpression(node.Expression())
	return seg
}

func (n *NodeBuilder) TransformResourceAccessRestSegment(resourceAccessRestSegmentNode *st.ResourceAccessRestSegmentNode) BLangNode {
	panic("TransformResourceAccessRestSegment unimplemented")
}

func (n *NodeBuilder) TransformReSequence(reSequenceNode *st.ReSequenceNode) BLangNode {
	panic("TransformReSequence unimplemented")
}

func (n *NodeBuilder) TransformReAtomQuantifier(reAtomQuantifierNode *st.ReAtomQuantifierNode) BLangNode {
	panic("TransformReAtomQuantifier unimplemented")
}

func (n *NodeBuilder) TransformReAtomCharOrEscape(reAtomCharOrEscapeNode *st.ReAtomCharOrEscapeNode) BLangNode {
	panic("TransformReAtomCharOrEscape unimplemented")
}

func (n *NodeBuilder) TransformReQuoteEscape(reQuoteEscapeNode *st.ReQuoteEscapeNode) BLangNode {
	panic("TransformReQuoteEscape unimplemented")
}

func (n *NodeBuilder) TransformReSimpleCharClassEscape(reSimpleCharClassEscapeNode *st.ReSimpleCharClassEscapeNode) BLangNode {
	panic("TransformReSimpleCharClassEscape unimplemented")
}

func (n *NodeBuilder) TransformReUnicodePropertyEscape(reUnicodePropertyEscapeNode *st.ReUnicodePropertyEscapeNode) BLangNode {
	panic("TransformReUnicodePropertyEscape unimplemented")
}

func (n *NodeBuilder) TransformReUnicodeScript(reUnicodeScriptNode *st.ReUnicodeScriptNode) BLangNode {
	panic("TransformReUnicodeScript unimplemented")
}

func (n *NodeBuilder) TransformReUnicodeGeneralCategory(reUnicodeGeneralCategoryNode *st.ReUnicodeGeneralCategoryNode) BLangNode {
	panic("TransformReUnicodeGeneralCategory unimplemented")
}

func (n *NodeBuilder) TransformReCharacterClass(reCharacterClassNode *st.ReCharacterClassNode) BLangNode {
	panic("TransformReCharacterClass unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRangeWithReCharSet(reCharSetRangeWithReCharSetNode *st.ReCharSetRangeWithReCharSetNode) BLangNode {
	panic("TransformReCharSetRangeWithReCharSet unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRange(reCharSetRangeNode *st.ReCharSetRangeNode) BLangNode {
	panic("TransformReCharSetRange unimplemented")
}

func (n *NodeBuilder) TransformReCharSetAtomWithReCharSetNoDash(reCharSetAtomWithReCharSetNoDashNode *st.ReCharSetAtomWithReCharSetNoDashNode) BLangNode {
	panic("TransformReCharSetAtomWithReCharSetNoDash unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRangeNoDashWithReCharSet(reCharSetRangeNoDashWithReCharSetNode *st.ReCharSetRangeNoDashWithReCharSetNode) BLangNode {
	panic("TransformReCharSetRangeNoDashWithReCharSet unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRangeNoDash(reCharSetRangeNoDashNode *st.ReCharSetRangeNoDashNode) BLangNode {
	panic("TransformReCharSetRangeNoDash unimplemented")
}

func (n *NodeBuilder) TransformReCharSetAtomNoDashWithReCharSetNoDash(reCharSetAtomNoDashWithReCharSetNoDashNode *st.ReCharSetAtomNoDashWithReCharSetNoDashNode) BLangNode {
	panic("TransformReCharSetAtomNoDashWithReCharSetNoDash unimplemented")
}

func (n *NodeBuilder) TransformReCapturingGroups(reCapturingGroupsNode *st.ReCapturingGroupsNode) BLangNode {
	panic("TransformReCapturingGroups unimplemented")
}

func (n *NodeBuilder) TransformReFlagExpression(reFlagBLangExpression *st.ReFlagExpressionNode) BLangNode {
	panic("TransformReFlagExpression unimplemented")
}

func (n *NodeBuilder) TransformReFlagsOnOff(reFlagsOnOffNode *st.ReFlagsOnOffNode) BLangNode {
	panic("TransformReFlagsOnOff unimplemented")
}

func (n *NodeBuilder) TransformReFlags(reFlagsNode *st.ReFlagsNode) BLangNode {
	panic("TransformReFlags unimplemented")
}

func (n *NodeBuilder) TransformReAssertion(reAssertionNode *st.ReAssertionNode) BLangNode {
	panic("TransformReAssertion unimplemented")
}

func (n *NodeBuilder) TransformReQuantifier(reQuantifierNode *st.ReQuantifierNode) BLangNode {
	panic("TransformReQuantifier unimplemented")
}

func (n *NodeBuilder) TransformReBracedQuantifier(reBracedQuantifierNode *st.ReBracedQuantifierNode) BLangNode {
	panic("TransformReBracedQuantifier unimplemented")
}

func (n *NodeBuilder) TransformMemberTypeDescriptor(memberTypeDescriptorNode *st.MemberTypeDescriptorNode) BLangNode {
	panic("TransformMemberTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformReceiveField(receiveFieldNode *st.ReceiveFieldNode) BLangNode {
	panic("TransformReceiveField unimplemented")
}

func (n *NodeBuilder) TransformNaturalExpression(naturalBLangExpression *st.NaturalExpressionNode) BLangNode {
	panic("TransformNaturalExpression unimplemented")
}

func (n *NodeBuilder) TransformToken(token st.Token) BLangNode {
	kind := token.Kind()
	switch kind {
	case st.XML_TEXT_CONTENT, st.TEMPLATE_STRING, st.CLOSE_BRACE_TOKEN, st.PROMPT_CONTENT:
		return n.createSimpleLiteral(token).(BLangNode)
	default:
		if isTokenInRegExp(kind) {
			return n.createSimpleLiteral(token).(BLangNode)
		}
		panic("TransformToken: Syntax kind is not supported: " + kind.StrValue())
	}
}

func (n *NodeBuilder) TransformIdentifierToken(identifier *st.IdentifierToken) BLangNode {
	panic("TransformIdentifierToken unimplemented")
}

func stringToTypeKind(typeText string) TypeKind {
	switch typeText {
	case "int":
		return TypeKind_INT
	case "byte":
		return TypeKind_BYTE
	case "float":
		return TypeKind_FLOAT
	case "decimal":
		return TypeKind_DECIMAL
	case "boolean":
		return TypeKind_BOOLEAN
	case "string":
		return TypeKind_STRING
	case "json":
		return TypeKind_JSON
	case "xml":
		return TypeKind_XML
	case "stream":
		return TypeKind_STREAM
	case "table":
		return TypeKind_TABLE
	case "any":
		return TypeKind_ANY
	case "anydata":
		return TypeKind_ANYDATA
	case "map":
		return TypeKind_MAP
	case "future":
		return TypeKind_FUTURE
	case "typedesc":
		return TypeKind_TYPEDESC
	case "error":
		return TypeKind_ERROR
	case "()", "null":
		return TypeKind_NIL
	case "never":
		return TypeKind_NEVER
	case "channel":
		return TypeKind_CHANNEL
	case "service":
		return TypeKind_SERVICE
	case "handle":
		return TypeKind_HANDLE
	case "readonly":
		return TypeKind_READONLY
	case "function":
		return TypeKind_FUNCTION
	default:
		panic("stringToTypeKind: invalid type name: " + typeText)
	}
}

func createUserDefinedType(pos diagnostics.Location, pkgAlias BLangIdentifier, typeName BLangIdentifier) TypeDescriptor {
	userDefinedType := BLangUserDefinedType{}
	userDefinedType.pos = pos
	userDefinedType.PkgAlias = pkgAlias
	userDefinedType.TypeName = typeName
	return &userDefinedType
}

func getNextMissingNodeName(pkgID *model.PackageID) string {
	panic("getNextMissingNodeName unimplemented")
}

func (n *NodeBuilder) getBLangVariableNode(bindingPattern st.BindingPatternNode, varPos diagnostics.Location) VariableNode {
	var varName st.Token
	switch bindingPattern.Kind() {
	case st.WILDCARD_BINDING_PATTERN:
		ignore := n.createIgnoreIdentifier(bindingPattern)
		simpleVar := createSimpleVariableNode()
		simpleVar.SetName(&ignore)
		simpleVar.pos = varPos
		return simpleVar
	case st.MAPPING_BINDING_PATTERN, st.LIST_BINDING_PATTERN, st.ERROR_BINDING_PATTERN, st.REST_BINDING_PATTERN:
		panic("unimplemented")
	case st.CAPTURE_BINDING_PATTERN:
		fallthrough
	default:
		captureBindingPattern := bindingPattern.(*st.CaptureBindingPatternNode)
		varName = captureBindingPattern.VariableName()
	}

	simpleVar := createSimpleVariableNode()
	simpleVar.pos = varPos
	simpleVar.SetName(n.createIdentifierNodeFromToken(n.getPosition(varName), varName))
	return simpleVar
}

func (n *NodeBuilder) badTopLevel(node st.Node) *BLangBadTopLevelNode {
	bad := &BLangBadTopLevelNode{}
	bad.SetPosition(n.getRecoveryPosition(node))
	return bad
}

func (n *NodeBuilder) badStmt(node st.Node) *BLangBadStmt {
	bad := &BLangBadStmt{}
	bad.SetPosition(n.getRecoveryPosition(node))
	return bad
}

func (n *NodeBuilder) badExprOrAction(node st.Node) *BLangBadExprOrAction {
	bad := &BLangBadExprOrAction{}
	if node != nil {
		bad.SetPosition(n.getRecoveryPosition(node))
	} else {
		bad.SetPosition(diagnostics.NewBuiltinLocation())
	}
	return bad
}

func (n *NodeBuilder) badTypeNode(node st.Node) *BLangBadTypeNode {
	bad := &BLangBadTypeNode{}
	if node != nil {
		bad.SetPosition(n.getRecoveryPosition(node))
	} else {
		bad.SetPosition(diagnostics.NewBuiltinLocation())
	}
	return bad
}

func (n *NodeBuilder) badIdentifier(token st.Token) *BLangBadIdentifier {
	bad := &BLangBadIdentifier{}
	if token != nil {
		bad.Value, bad.isLiteral = normalizedIdentifierValue(token.Text())
		bad.OriginalValue = token.Text()
		bad.SetPosition(n.getRecoveryPosition(token))
	} else {
		bad.SetPosition(diagnostics.NewBuiltinLocation())
	}
	return bad
}

func (n *NodeBuilder) syntaxError(node st.Node) {
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
