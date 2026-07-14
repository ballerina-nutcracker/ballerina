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
	"regexp"
	"strconv"
	"strings"

	"ballerina-lang-go/context"
	"ballerina-lang-go/model"
	"ballerina-lang-go/parser/common"
	"ballerina-lang-go/parser/tree"
	"ballerina-lang-go/tools/diagnostics"

	balCommon "ballerina-lang-go/common"
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

type NodeBuilder struct {
	PackageID            *model.PackageID
	anonTypeNameSuffixes []string // Stack for anonymous type name suffixes
	additionalStatements []StatementNode
	currentCompUnit      *BLangCompilationUnit
	CurrentCompUnitName  string
	isInLocalContext     bool
	isInFiniteContext    bool
	constantSet          map[string]string // Track declared constants to detect redeclarations
	cx                   *context.CompilerContext
	types                typeTable
}

func (n *NodeBuilder) de() *diagnostics.DiagnosticEnv {
	return n.cx.DiagnosticEnv()
}

// NewNodeBuilder creates and initializes a new NodeBuilder instance
func NewNodeBuilder(cx *context.CompilerContext) *NodeBuilder {
	nodeBuilder := &NodeBuilder{
		constantSet: make(map[string]string),
		cx:          cx,
		PackageID:   cx.GetDefaultPackage(),
		types:       newTypeTable(),
	}
	return nodeBuilder
}

var _ tree.NodeTransformer[BLangNode] = &NodeBuilder{}

const (
	OPEN_ARRAY_INDICATOR     = -1
	INFERRED_ARRAY_INDICATOR = -2
)

func (n *NodeBuilder) TransformSyntaxNode(node tree.Node) BLangNode {
	switch t := node.(type) {
	case *tree.ModulePart:
		return n.TransformModulePart(t)
	case *tree.FunctionDefinition:
		return n.TransformFunctionDefinition(t)
	case *tree.ImportDeclarationNode:
		return n.TransformImportDeclaration(t)
	case *tree.ListenerDeclarationNode:
		return n.TransformListenerDeclaration(t)
	case *tree.TypeDefinitionNode:
		return n.TransformTypeDefinition(t)
	case *tree.ServiceDeclarationNode:
		return n.TransformServiceDeclaration(t)
	case *tree.AssignmentStatementNode:
		return n.TransformAssignmentStatement(t)
	case *tree.CompoundAssignmentStatementNode:
		return n.TransformCompoundAssignmentStatement(t)
	case *tree.VariableDeclarationNode:
		return n.TransformVariableDeclaration(t)
	case *tree.BlockStatementNode:
		return n.TransformBlockStatement(t)
	case *tree.BreakStatementNode:
		return n.TransformBreakStatement(t)
	case *tree.FailStatementNode:
		return n.TransformFailStatement(t)
	case *tree.ExpressionStatementNode:
		return n.TransformExpressionStatement(t)
	case *tree.ContinueStatementNode:
		return n.TransformContinueStatement(t)
	case *tree.ExternalFunctionBodyNode:
		return n.TransformExternalFunctionBody(t)
	case *tree.IfElseStatementNode:
		return n.TransformIfElseStatement(t)
	case *tree.ElseBlockNode:
		return n.TransformElseBlock(t)
	case *tree.WhileStatementNode:
		return n.TransformWhileStatement(t)
	case *tree.PanicStatementNode:
		return n.TransformPanicStatement(t)
	case *tree.ReturnStatementNode:
		return n.TransformReturnStatement(t)
	case *tree.LocalTypeDefinitionStatementNode:
		return n.TransformLocalTypeDefinitionStatement(t)
	case *tree.LockStatementNode:
		return n.TransformLockStatement(t)
	case *tree.ForkStatementNode:
		return n.TransformForkStatement(t)
	case *tree.ForEachStatementNode:
		return n.TransformForEachStatement(t)
	case *tree.BinaryExpressionNode:
		return n.TransformBinaryExpression(t)
	case *tree.BracedExpressionNode:
		return n.TransformBracedExpression(t)
	case *tree.CheckExpressionNode:
		return n.TransformCheckExpression(t)
	case *tree.FieldAccessExpressionNode:
		return n.TransformFieldAccessExpression(t)
	case *tree.FunctionCallExpressionNode:
		return n.TransformFunctionCallExpression(t)
	case *tree.MethodCallExpressionNode:
		return n.TransformMethodCallExpression(t)
	case *tree.MappingConstructorExpressionNode:
		return n.TransformMappingConstructorExpression(t)
	case *tree.IndexedExpressionNode:
		return n.TransformIndexedExpression(t)
	case *tree.TypeofExpressionNode:
		return n.TransformTypeofExpression(t)
	case *tree.UnaryExpressionNode:
		return n.TransformUnaryExpression(t)
	case *tree.ComputedNameFieldNode:
		return n.TransformComputedNameField(t)
	case *tree.ConstantDeclarationNode:
		return n.TransformConstantDeclaration(t)
	case *tree.DefaultableParameterNode:
		return n.TransformDefaultableParameter(t)
	case *tree.RequiredParameterNode:
		return n.TransformRequiredParameter(t)
	case *tree.IncludedRecordParameterNode:
		return n.TransformIncludedRecordParameter(t)
	case *tree.RestParameterNode:
		return n.TransformRestParameter(t)
	case *tree.ImportOrgNameNode:
		return n.TransformImportOrgName(t)
	case *tree.ImportPrefixNode:
		return n.TransformImportPrefix(t)
	case *tree.SpecificFieldNode:
		return n.TransformSpecificField(t)
	case *tree.SpreadFieldNode:
		return n.TransformSpreadField(t)
	case *tree.NamedArgumentNode:
		return n.TransformNamedArgument(t)
	case *tree.PositionalArgumentNode:
		return n.TransformPositionalArgument(t)
	case *tree.RestArgumentNode:
		return n.TransformRestArgument(t)
	case *tree.InferredTypedescDefaultNode:
		return n.TransformInferredTypedescDefault(t)
	case *tree.ObjectTypeDescriptorNode:
		return n.TransformObjectTypeDescriptor(t)
	case *tree.ObjectConstructorExpressionNode:
		return n.TransformObjectConstructorExpression(t)
	case *tree.RecordTypeDescriptorNode:
		return n.TransformRecordTypeDescriptor(t)
	case *tree.ReturnTypeDescriptorNode:
		return n.TransformReturnTypeDescriptor(t)
	case *tree.NilTypeDescriptorNode:
		return n.TransformNilTypeDescriptor(t)
	case *tree.OptionalTypeDescriptorNode:
		return n.TransformOptionalTypeDescriptor(t)
	case *tree.ObjectFieldNode:
		return n.TransformObjectField(t)
	case *tree.RecordFieldNode:
		return n.TransformRecordField(t)
	case *tree.RecordFieldWithDefaultValueNode:
		return n.TransformRecordFieldWithDefaultValue(t)
	case *tree.RecordRestDescriptorNode:
		return n.TransformRecordRestDescriptor(t)
	case *tree.TypeReferenceNode:
		return n.TransformTypeReference(t)
	case *tree.AnnotationNode:
		return n.TransformAnnotation(t)
	case *tree.MetadataNode:
		return n.TransformMetadata(t)
	case *tree.ModuleVariableDeclarationNode:
		return n.TransformModuleVariableDeclaration(t)
	case *tree.TypeTestExpressionNode:
		return n.TransformTypeTestExpression(t)
	case *tree.RemoteMethodCallActionNode:
		return n.TransformRemoteMethodCallAction(t)
	case *tree.MapTypeDescriptorNode:
		return n.TransformMapTypeDescriptor(t)
	case *tree.NilLiteralNode:
		return n.TransformNilLiteral(t)
	case *tree.AnnotationDeclarationNode:
		return n.TransformAnnotationDeclaration(t)
	case *tree.AnnotationAttachPointNode:
		return n.TransformAnnotationAttachPoint(t)
	case *tree.XMLNamespaceDeclarationNode:
		return n.TransformXMLNamespaceDeclaration(t)
	case *tree.ModuleXMLNamespaceDeclarationNode:
		return n.TransformModuleXMLNamespaceDeclaration(t)
	case *tree.FunctionBodyBlockNode:
		return n.TransformFunctionBodyBlock(t)
	case *tree.NamedWorkerDeclarationNode:
		return n.TransformNamedWorkerDeclaration(t)
	case *tree.NamedWorkerDeclarator:
		return n.TransformNamedWorkerDeclarator(t)
	case *tree.BasicLiteralNode:
		return n.TransformBasicLiteral(t)
	case *tree.SimpleNameReferenceNode:
		return n.TransformSimpleNameReference(t)
	case *tree.QualifiedNameReferenceNode:
		return n.TransformQualifiedNameReference(t)
	case *tree.BuiltinSimpleNameReferenceNode:
		return n.TransformBuiltinSimpleNameReference(t)
	case *tree.TrapExpressionNode:
		return n.TransformTrapExpression(t)
	case *tree.ListConstructorExpressionNode:
		return n.TransformListConstructorExpression(t)
	case *tree.TypeCastExpressionNode:
		return n.TransformTypeCastExpression(t)
	case *tree.TypeCastParamNode:
		return n.TransformTypeCastParam(t)
	case *tree.UnionTypeDescriptorNode:
		return n.TransformUnionTypeDescriptor(t)
	case *tree.TableConstructorExpressionNode:
		return n.TransformTableConstructorExpression(t)
	case *tree.KeySpecifierNode:
		return n.TransformKeySpecifier(t)
	case *tree.StreamTypeDescriptorNode:
		return n.TransformStreamTypeDescriptor(t)
	case *tree.StreamTypeParamsNode:
		return n.TransformStreamTypeParams(t)
	case *tree.LetExpressionNode:
		return n.TransformLetExpression(t)
	case *tree.LetVariableDeclarationNode:
		return n.TransformLetVariableDeclaration(t)
	case *tree.TemplateExpressionNode:
		return n.TransformTemplateExpression(t)
	case *tree.XMLElementNode:
		return n.TransformXMLElement(t)
	case *tree.XMLStartTagNode:
		return n.TransformXMLStartTag(t)
	case *tree.XMLEndTagNode:
		return n.TransformXMLEndTag(t)
	case *tree.XMLSimpleNameNode:
		return n.TransformXMLSimpleName(t)
	case *tree.XMLQualifiedNameNode:
		return n.TransformXMLQualifiedName(t)
	case *tree.XMLEmptyElementNode:
		return n.TransformXMLEmptyElement(t)
	case *tree.InterpolationNode:
		return n.TransformInterpolation(t)
	case *tree.XMLTextNode:
		return n.TransformXMLText(t)
	case *tree.XMLAttributeNode:
		return n.TransformXMLAttribute(t)
	case *tree.XMLAttributeValue:
		return n.TransformXMLAttributeValue(t)
	case *tree.XMLComment:
		return n.TransformXMLComment(t)
	case *tree.XMLCDATANode:
		return n.TransformXMLCDATA(t)
	case *tree.XMLProcessingInstruction:
		return n.TransformXMLProcessingInstruction(t)
	case *tree.TableTypeDescriptorNode:
		return n.TransformTableTypeDescriptor(t)
	case *tree.TypeParameterNode:
		return n.TransformTypeParameter(t)
	case *tree.KeyTypeConstraintNode:
		return n.TransformKeyTypeConstraint(t)
	case *tree.FunctionTypeDescriptorNode:
		return n.TransformFunctionTypeDescriptor(t)
	case *tree.FunctionSignatureNode:
		return n.TransformFunctionSignature(t)
	case *tree.ExplicitAnonymousFunctionExpressionNode:
		return n.TransformExplicitAnonymousFunctionExpression(t)
	case *tree.ExpressionFunctionBodyNode:
		return n.TransformExpressionFunctionBody(t)
	case *tree.TupleTypeDescriptorNode:
		return n.TransformTupleTypeDescriptor(t)
	case *tree.ParenthesisedTypeDescriptorNode:
		return n.TransformParenthesisedTypeDescriptor(t)
	case *tree.ExplicitNewExpressionNode:
		return n.TransformExplicitNewExpression(t)
	case *tree.ImplicitNewExpressionNode:
		return n.TransformImplicitNewExpression(t)
	case *tree.ParenthesizedArgList:
		return n.TransformParenthesizedArgList(t)
	case *tree.QueryConstructTypeNode:
		return n.TransformQueryConstructType(t)
	case *tree.FromClauseNode:
		return n.TransformFromClause(t)
	case *tree.WhereClauseNode:
		return n.TransformWhereClause(t)
	case *tree.LetClauseNode:
		return n.TransformLetClause(t)
	case *tree.JoinClauseNode:
		return n.TransformJoinClause(t)
	case *tree.OnClauseNode:
		return n.TransformOnClause(t)
	case *tree.LimitClauseNode:
		return n.TransformLimitClause(t)
	case *tree.OnConflictClauseNode:
		return n.TransformOnConflictClause(t)
	case *tree.QueryPipelineNode:
		return n.TransformQueryPipeline(t)
	case *tree.SelectClauseNode:
		return n.TransformSelectClause(t)
	case *tree.CollectClauseNode:
		return n.TransformCollectClause(t)
	case *tree.QueryExpressionNode:
		return n.TransformQueryExpression(t)
	case *tree.QueryActionNode:
		return n.TransformQueryAction(t)
	case *tree.IntersectionTypeDescriptorNode:
		return n.TransformIntersectionTypeDescriptor(t)
	case *tree.ImplicitAnonymousFunctionParameters:
		return n.TransformImplicitAnonymousFunctionParameters(t)
	case *tree.ImplicitAnonymousFunctionExpressionNode:
		return n.TransformImplicitAnonymousFunctionExpression(t)
	case *tree.StartActionNode:
		return n.TransformStartAction(t)
	case *tree.FlushActionNode:
		return n.TransformFlushAction(t)
	case *tree.SingletonTypeDescriptorNode:
		return n.TransformSingletonTypeDescriptor(t)
	case *tree.MethodDeclarationNode:
		return n.TransformMethodDeclaration(t)
	case *tree.TypedBindingPatternNode:
		return n.TransformTypedBindingPattern(t)
	case *tree.CaptureBindingPatternNode:
		return n.TransformCaptureBindingPattern(t)
	case *tree.WildcardBindingPatternNode:
		return n.TransformWildcardBindingPattern(t)
	case *tree.ListBindingPatternNode:
		return n.TransformListBindingPattern(t)
	case *tree.MappingBindingPatternNode:
		return n.TransformMappingBindingPattern(t)
	case *tree.FieldBindingPatternFullNode:
		return n.TransformFieldBindingPatternFull(t)
	case *tree.FieldBindingPatternVarnameNode:
		return n.TransformFieldBindingPatternVarname(t)
	case *tree.RestBindingPatternNode:
		return n.TransformRestBindingPattern(t)
	case *tree.ErrorBindingPatternNode:
		return n.TransformErrorBindingPattern(t)
	case *tree.NamedArgBindingPatternNode:
		return n.TransformNamedArgBindingPattern(t)
	case *tree.AsyncSendActionNode:
		return n.TransformAsyncSendAction(t)
	case *tree.SyncSendActionNode:
		return n.TransformSyncSendAction(t)
	case *tree.ReceiveActionNode:
		return n.TransformReceiveAction(t)
	case *tree.ReceiveFieldsNode:
		return n.TransformReceiveFields(t)
	case *tree.AlternateReceiveNode:
		return n.TransformAlternateReceive(t)
	case *tree.RestDescriptorNode:
		return n.TransformRestDescriptor(t)
	case *tree.DoubleGTTokenNode:
		return n.TransformDoubleGTToken(t)
	case *tree.TrippleGTTokenNode:
		return n.TransformTrippleGTToken(t)
	case *tree.WaitActionNode:
		return n.TransformWaitAction(t)
	case *tree.WaitFieldsListNode:
		return n.TransformWaitFieldsList(t)
	case *tree.WaitFieldNode:
		return n.TransformWaitField(t)
	case *tree.AnnotAccessExpressionNode:
		return n.TransformAnnotAccessExpression(t)
	case *tree.OptionalFieldAccessExpressionNode:
		return n.TransformOptionalFieldAccessExpression(t)
	case *tree.ConditionalExpressionNode:
		return n.TransformConditionalExpression(t)
	case *tree.EnumDeclarationNode:
		return n.TransformEnumDeclaration(t)
	case *tree.EnumMemberNode:
		return n.TransformEnumMember(t)
	case *tree.ArrayTypeDescriptorNode:
		return n.TransformArrayTypeDescriptor(t)
	case *tree.ArrayDimensionNode:
		return n.TransformArrayDimension(t)
	case *tree.TransactionStatementNode:
		return n.TransformTransactionStatement(t)
	case *tree.RollbackStatementNode:
		return n.TransformRollbackStatement(t)
	case *tree.RetryStatementNode:
		return n.TransformRetryStatement(t)
	case *tree.CommitActionNode:
		return n.TransformCommitAction(t)
	case *tree.TransactionalExpressionNode:
		return n.TransformTransactionalExpression(t)
	case *tree.ByteArrayLiteralNode:
		return n.TransformByteArrayLiteral(t)
	case *tree.XMLFilterExpressionNode:
		return n.TransformXMLFilterExpression(t)
	case *tree.XMLStepExpressionNode:
		return n.TransformXMLStepExpression(t)
	case *tree.XMLNamePatternChainingNode:
		return n.TransformXMLNamePatternChaining(t)
	case *tree.XMLStepIndexedExtendNode:
		return n.TransformXMLStepIndexedExtend(t)
	case *tree.XMLStepMethodCallExtendNode:
		return n.TransformXMLStepMethodCallExtend(t)
	case *tree.XMLAtomicNamePatternNode:
		return n.TransformXMLAtomicNamePattern(t)
	case *tree.TypeReferenceTypeDescNode:
		return n.TransformTypeReferenceTypeDesc(t)
	case *tree.MatchStatementNode:
		return n.TransformMatchStatement(t)
	case *tree.MatchClauseNode:
		return n.TransformMatchClause(t)
	case *tree.MatchGuardNode:
		return n.TransformMatchGuard(t)
	case *tree.DistinctTypeDescriptorNode:
		return n.TransformDistinctTypeDescriptor(t)
	case *tree.ListMatchPatternNode:
		return n.TransformListMatchPattern(t)
	case *tree.RestMatchPatternNode:
		return n.TransformRestMatchPattern(t)
	case *tree.MappingMatchPatternNode:
		return n.TransformMappingMatchPattern(t)
	case *tree.FieldMatchPatternNode:
		return n.TransformFieldMatchPattern(t)
	case *tree.ErrorMatchPatternNode:
		return n.TransformErrorMatchPattern(t)
	case *tree.NamedArgMatchPatternNode:
		return n.TransformNamedArgMatchPattern(t)
	case *tree.OrderByClauseNode:
		return n.TransformOrderByClause(t)
	case *tree.OrderKeyNode:
		return n.TransformOrderKey(t)
	case *tree.GroupByClauseNode:
		return n.TransformGroupByClause(t)
	case *tree.GroupingKeyVarDeclarationNode:
		return n.TransformGroupingKeyVarDeclaration(t)
	case *tree.OnFailClauseNode:
		return n.TransformOnFailClause(t)
	case *tree.DoStatementNode:
		return n.TransformDoStatement(t)
	case *tree.ClassDefinitionNode:
		return n.TransformClassDefinition(t)
	case *tree.ResourcePathParameterNode:
		return n.TransformResourcePathParameter(t)
	case *tree.RequiredExpressionNode:
		return n.TransformRequiredExpression(t)
	case *tree.ErrorConstructorExpressionNode:
		return n.TransformErrorConstructorExpression(t)
	case *tree.ParameterizedTypeDescriptorNode:
		return n.TransformParameterizedTypeDescriptor(t)
	case *tree.SpreadMemberNode:
		return n.TransformSpreadMember(t)
	case *tree.ClientResourceAccessActionNode:
		return n.TransformClientResourceAccessAction(t)
	case *tree.ComputedResourceAccessSegmentNode:
		return n.TransformComputedResourceAccessSegment(t)
	case *tree.ResourceAccessRestSegmentNode:
		return n.TransformResourceAccessRestSegment(t)
	case *tree.ReSequenceNode:
		return n.TransformReSequence(t)
	case *tree.ReAtomQuantifierNode:
		return n.TransformReAtomQuantifier(t)
	case *tree.ReAtomCharOrEscapeNode:
		return n.TransformReAtomCharOrEscape(t)
	case *tree.ReQuoteEscapeNode:
		return n.TransformReQuoteEscape(t)
	case *tree.ReSimpleCharClassEscapeNode:
		return n.TransformReSimpleCharClassEscape(t)
	case *tree.ReUnicodePropertyEscapeNode:
		return n.TransformReUnicodePropertyEscape(t)
	case *tree.ReUnicodeScriptNode:
		return n.TransformReUnicodeScript(t)
	case *tree.ReUnicodeGeneralCategoryNode:
		return n.TransformReUnicodeGeneralCategory(t)
	case *tree.ReCharacterClassNode:
		return n.TransformReCharacterClass(t)
	case *tree.ReCharSetRangeWithReCharSetNode:
		return n.TransformReCharSetRangeWithReCharSet(t)
	case *tree.ReCharSetRangeNode:
		return n.TransformReCharSetRange(t)
	case *tree.ReCharSetAtomWithReCharSetNoDashNode:
		return n.TransformReCharSetAtomWithReCharSetNoDash(t)
	case *tree.ReCharSetRangeNoDashWithReCharSetNode:
		return n.TransformReCharSetRangeNoDashWithReCharSet(t)
	case *tree.ReCharSetRangeNoDashNode:
		return n.TransformReCharSetRangeNoDash(t)
	case *tree.ReCharSetAtomNoDashWithReCharSetNoDashNode:
		return n.TransformReCharSetAtomNoDashWithReCharSetNoDash(t)
	case *tree.ReCapturingGroupsNode:
		return n.TransformReCapturingGroups(t)
	case *tree.ReFlagExpressionNode:
		return n.TransformReFlagExpression(t)
	case *tree.ReFlagsOnOffNode:
		return n.TransformReFlagsOnOff(t)
	case *tree.ReFlagsNode:
		return n.TransformReFlags(t)
	case *tree.ReAssertionNode:
		return n.TransformReAssertion(t)
	case *tree.ReQuantifierNode:
		return n.TransformReQuantifier(t)
	case *tree.ReBracedQuantifierNode:
		return n.TransformReBracedQuantifier(t)
	case *tree.MemberTypeDescriptorNode:
		return n.TransformMemberTypeDescriptor(t)
	case *tree.ReceiveFieldNode:
		return n.TransformReceiveField(t)
	case *tree.NaturalExpressionNode:
		return n.TransformNaturalExpression(t)
	case *tree.IdentifierToken:
		return n.TransformIdentifierToken(t)
	case tree.Token:
		return n.TransformToken(t)
	default:
		panic("TransformSyntaxNode: unsupported node type")
	}
}

func getFileName(node tree.Node) string {
	st := node.SyntaxTree()
	return st.FilePath()
}

func innermostDiagnosticNodes(node tree.Node) []tree.Node {
	if !node.HasDiagnostics() {
		return nil
	}

	var nodes []tree.Node
	if nt, ok := node.(tree.NonTerminalNode); ok {
		for child := range nt.ChildNodes() {
			if child != nil && child.HasDiagnostics() {
				nodes = append(nodes, innermostDiagnosticNodes(child)...)
			}
		}
	}
	if len(nodes) > 0 {
		return nodes
	}
	return []tree.Node{node}
}

func diagnosticMessage(node tree.Node) string {
	deep := tree.FindDeepestDiagnosticSTNode(node.InternalNode())
	if deep == nil || len(deep.Diagnostics()) == 0 {
		return "syntax error"
	}
	return strings.ReplaceAll(strings.TrimPrefix(deep.Diagnostics()[0].DiagnosticCode().MessageKey(), "error."), ".", " ")
}

func getPosition(de *diagnostics.DiagnosticEnv, node tree.Node) diagnostics.Location {
	textRange := node.TextRange()
	fileName := getFileName(node)
	return diagnostics.NewLocation(de, fileName, textRange.StartOffset, textRange.EndOffset)
}

func getPositionRange(de *diagnostics.DiagnosticEnv, startNode tree.Node, endNode tree.Node) diagnostics.Location {
	startRange := startNode.TextRange()
	endRange := endNode.TextRange()
	fileName := getFileName(startNode)
	return diagnostics.NewLocation(de, fileName, startRange.StartOffset, endRange.EndOffset)
}

func getPositionWithoutMetadata(de *diagnostics.DiagnosticEnv, node tree.Node) diagnostics.Location {
	nodeTextRange := node.TextRange()
	nonTerminalNode := node.(tree.NonTerminalNode)

	startOffset := nodeTextRange.StartOffset

	var firstChild, secondChild tree.Node
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

	if firstChild != nil && firstChild.Kind() == common.METADATA && secondChild != nil {
		startOffset = secondChild.TextRange().StartOffset
	}

	fileName := getFileName(node)
	return diagnostics.NewLocation(de, fileName, startOffset, nodeTextRange.EndOffset)
}

// getDocumentationString extracts the documentation string from metadata
func getDocumentationString(metadata *tree.MetadataNode) tree.Node {
	return metadata.DocumentationString()
}

func (n *NodeBuilder) populateMetadata(metadata *tree.MetadataNode, target AnnotatableNode) {
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

func (n *NodeBuilder) addAnnotationAttachments(annotations tree.NodeList[*tree.AnnotationNode], target AnnotatableNode) {
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
func (n *NodeBuilder) createMarkdownDocumentationAttachment(docStringNode tree.Node) *BLangMarkdownDocumentation {
	if docStringNode == nil || docStringNode.IsMissing() {
		return nil
	}

	markdownDocumentationNode, ok := docStringNode.(*tree.MarkdownDocumentationNode)
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
		case common.MARKDOWN_DOCUMENTATION_LINE, common.MARKDOWN_REFERENCE_DOCUMENTATION_LINE:
			docLineNode := singleDocLine.(*tree.MarkdownDocumentationLineNode)
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
				bLangDocLine.pos = getPosition(n.de(), docLineNode)
				documentationLines = append(documentationLines, bLangDocLine)
			}
		case common.MARKDOWN_PARAMETER_DOCUMENTATION_LINE:
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
			parameterDocLineNode := singleDocLine.(*tree.MarkdownParameterDocumentationLineNode)

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
			bLangParaDoc.pos = getPosition(n.de(), parameterName)
		case common.MARKDOWN_RETURN_PARAMETER_DOCUMENTATION_LINE:
			bLangReturnParaDoc = &BLangMarkdownReturnParameterDocumentation{}
			returnParaDocLineNode := singleDocLine.(*tree.MarkdownParameterDocumentationLineNode)

			returnParaDocElements := returnParaDocLineNode.DocumentElements()
			returnParaDocText := n.addReferencesAndReturnDocumentationText(&references, returnParaDocElements)

			bLangReturnParaDoc.ReturnParameterDocumentationLines = append(bLangReturnParaDoc.ReturnParameterDocumentationLines, returnParaDocText)
			bLangReturnParaDoc.pos = getPosition(n.de(), returnParaDocLineNode)
			doc.ReturnParameter = bLangReturnParaDoc
		case common.MARKDOWN_DEPRECATION_DOCUMENTATION_LINE:
			bLangDeprecationDoc = &BLangMarkDownDeprecationDocumentation{}
			deprecationDocLineNode := singleDocLine.(*tree.MarkdownDocumentationLineNode)

			docElements := deprecationDocLineNode.DocumentElements()
			var lineText string
			if docElements.Size() > 0 {
				firstElement := docElements.Get(0)
				if token, ok := firstElement.(tree.Token); ok {
					lineText = token.Text()
				}
			}
			bLangDeprecationDoc.AddDeprecationLine("# " + lineText)
			bLangDeprecationDoc.pos = getPosition(n.de(), deprecationDocLineNode)
		case common.MARKDOWN_CODE_BLOCK:
			codeBlockNode := singleDocLine.(*tree.MarkdownCodeBlockNode)
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
	doc.pos = getPosition(n.de(), markdownDocumentationNode)
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
func createIdentifierFromToken(pos diagnostics.Location, token tree.Token) BLangIdentifier {
	return createIdentifierFromTokenInternal(pos, token, false)
}

// createIdentifierFromTokenInternal creates an identifier from a token with XML handling option
func createIdentifierFromTokenInternal(pos diagnostics.Location, token tree.Token, isXML bool) BLangIdentifier {
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

func createIgnoreIdentifier(de *diagnostics.DiagnosticEnv, node tree.Node) BLangIdentifier {
	pos := getPosition(de, node)
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
func (n *NodeBuilder) createTypeNode(typeNode tree.Node) TypeDescriptor {
	if typeNode == nil {
		panic("createTypeNode: typeNode is nil")
	}
	if typeNode, ok := typeNode.(*tree.BuiltinSimpleNameReferenceNode); ok {
		return n.createBuiltInTypeNode(typeNode)
	}
	kind := typeNode.Kind()
	switch kind {
	case common.NIL_TYPE_DESC:
		return n.createBuiltInTypeNode(typeNode)
	case common.QUALIFIED_NAME_REFERENCE, common.IDENTIFIER_TOKEN:
		bLUserDefinedType := BLangUserDefinedType{}
		nameRefence := n.createBLangNameReference(typeNode)
		bLUserDefinedType.PkgAlias = nameRefence[0]
		bLUserDefinedType.TypeName = nameRefence[1]
		bLUserDefinedType.pos = getPosition(n.de(), typeNode)
		return &bLUserDefinedType
	case common.SIMPLE_NAME_REFERENCE:
		nameReferenceNode := typeNode.(*tree.SimpleNameReferenceNode)
		return n.createTypeNode(nameReferenceNode.Name())
	default:
		return n.TransformSyntaxNode(typeNode).(BType)
	}
}

// isDeclaredWithVar checks if a type node is declared with var
func isDeclaredWithVar(typeNode tree.Node) bool {
	if typeNode == nil || typeNode.Kind() == common.VAR_TYPE_DESC {
		return true
	}
	return false
}

func (n *NodeBuilder) createSimpleVarInner(name tree.Token, typeName tree.Node, initializer tree.Node, visibilityQualifier tree.Token, annotations tree.NodeList[*tree.AnnotationNode]) *BLangSimpleVariable {
	bLSimpleVar := createSimpleVariableNode()

	var namePos diagnostics.Location
	if name != nil {
		namePos = getPosition(n.de(), name)
	}
	identifier := createIdentifierFromToken(namePos, name)
	identifier.pos = namePos
	bLSimpleVar.SetName(&identifier)

	if isDeclaredWithVar(typeName) {
		bLSimpleVar.IsDeclaredWithVar = true
	} else {
		bLSimpleVar.SetTypeNode(n.createTypeNode(typeName).(BType))
	}

	if visibilityQualifier != nil {
		if visibilityQualifier.Kind() == common.PRIVATE_KEYWORD {
			bLSimpleVar.SetPrivate()
		} else if visibilityQualifier.Kind() == common.PUBLIC_KEYWORD {
			bLSimpleVar.SetPublic()
		}
	}

	if initializer != nil {
		bLSimpleVar.SetInitialExpression(n.createExpression(initializer))
	}

	n.addAnnotationAttachments(annotations, bLSimpleVar)

	return bLSimpleVar
}

func (n *NodeBuilder) createBuiltInTypeNode(typeNode tree.Node) TypeDescriptor {
	var typeText string
	if typeNode.Kind() == common.NIL_TYPE_DESC {
		typeText = "()"
	} else if simpleNameRef, ok := typeNode.(*tree.BuiltinSimpleNameReferenceNode); ok {
		if simpleNameRef.Kind() == common.VAR_TYPE_DESC {
			return nil
		} else if simpleNameRef.Name().IsMissing() {
			name := getNextMissingNodeName(n.PackageID)
			identifier := createIdentifier(getPosition(n.de(), simpleNameRef.Name()), &name, &name)
			pkgAlias := BLangIdentifier{}
			return createUserDefinedType(getPosition(n.de(), typeNode), pkgAlias, identifier)
		}
		typeText = simpleNameRef.Name().Text()
	} else {
		// TODO: Remove this once map<string> returns Nodes for `map`
		if token, ok := typeNode.(tree.Token); ok {
			typeText = token.Text()
		} else {
			panic("createBuiltInTypeNode: unexpected node type")
		}
	}

	typeKind := stringToTypeKind(typeText)

	kind := typeNode.Kind()
	switch kind {
	case common.BOOLEAN_TYPE_DESC,
		common.INT_TYPE_DESC,
		common.BYTE_TYPE_DESC,
		common.FLOAT_TYPE_DESC,
		common.DECIMAL_TYPE_DESC,
		common.STRING_TYPE_DESC,
		common.ANY_TYPE_DESC,
		common.NIL_TYPE_DESC,
		common.HANDLE_TYPE_DESC,
		common.ANYDATA_TYPE_DESC,
		common.READONLY_TYPE_DESC,
		common.NEVER_TYPE_DESC:
		valueType := BLangValueType{}
		valueType.TypeKind = typeKind
		valueType.pos = getPosition(n.de(), typeNode)
		return &valueType
	default:
		builtInValueType := BLangBuiltInRefTypeNode{}
		builtInValueType.TypeKind = typeKind
		builtInValueType.pos = getPosition(n.de(), typeNode)
		return &builtInValueType
	}
}

func (n *NodeBuilder) createBLangNameReference(node tree.Node) [2]BLangIdentifier {
	switch node.Kind() {
	case common.QUALIFIED_NAME_REFERENCE:
		iNode := node.(*tree.QualifiedNameReferenceNode)
		modulePrefix := iNode.ModulePrefix()
		identifier := iNode.Identifier()
		pkgAlias := createIdentifierFromToken(getPosition(n.de(), modulePrefix), modulePrefix)
		namePos := getPosition(n.de(), identifier)
		name := createIdentifierFromToken(namePos, identifier)
		return [...]BLangIdentifier{pkgAlias, name}
	case common.ERROR_TYPE_DESC:
		builtinNode := node.(*tree.BuiltinSimpleNameReferenceNode)
		node = builtinNode.Name()
		// Fall through to default handling
	case common.NEW_KEYWORD, common.IDENTIFIER_TOKEN, common.ERROR_KEYWORD:
		// Break and fall through to default handling
	case common.SIMPLE_NAME_REFERENCE:
		fallthrough
	default:
		simpleNode := node.(*tree.SimpleNameReferenceNode)
		node = simpleNode.Name()
	}

	// Default case: node should be a Token at this point
	iToken := node.(tree.Token)

	emptyStr := ""
	pkgAlias := createIdentifier(diagnostics.NewBuiltinLocation(), &emptyStr, &emptyStr)
	name := createIdentifierFromToken(getPosition(n.de(), iToken), iToken)
	return [...]BLangIdentifier{pkgAlias, name}
}

// isFunctionCallAsync checks if a function call expression is async
func (n *NodeBuilder) isFunctionCallAsync(functionCallBLangExpression *tree.FunctionCallExpressionNode) bool {
	parent := functionCallBLangExpression.Parent()
	if parent == nil {
		panic("isFunctionCallAsync: parent is nil")
	}
	return parent.Kind() == common.START_ACTION
}

// createBLangInvocation creates a BLangInvocation from a name node and arguments
func (n *NodeBuilder) createBLangInvocation(nameNode tree.Node, arguments tree.NodeList[tree.FunctionArgumentNode], position diagnostics.Location, isAsync bool) *BLangInvocation {
	var bLInvocation BLangInvocation
	if isAsync {
		panic("unimplemented")
	} else {
		bLInvocation = BLangInvocation{}
	}

	nameReference := n.createBLangNameReference(nameNode)
	bLInvocation.PkgAlias = &nameReference[0]
	bLInvocation.Name = &nameReference[1]

	var args []BLangExpression
	for arg := range arguments.Iterator() {
		args = append(args, n.createExpression(arg))
	}
	bLInvocation.ArgExprs = args
	bLInvocation.pos = position
	return &bLInvocation
}

// isSimpleLiteral checks if the syntax kind is a simple literal
func isSimpleLiteral(syntaxKind common.SyntaxKind) bool {
	switch syntaxKind {
	case common.STRING_LITERAL, common.NUMERIC_LITERAL, common.BOOLEAN_LITERAL, common.NIL_LITERAL, common.NULL_LITERAL:
		return true
	default:
		return false
	}
}

// isType checks if the syntax kind is a type descriptor
func isType(nodeKind common.SyntaxKind) bool {
	switch nodeKind {
	case common.RECORD_TYPE_DESC,
		common.OBJECT_TYPE_DESC,
		common.NIL_TYPE_DESC,
		common.OPTIONAL_TYPE_DESC,
		common.ARRAY_TYPE_DESC,
		common.INT_TYPE_DESC,
		common.BYTE_TYPE_DESC,
		common.FLOAT_TYPE_DESC,
		common.DECIMAL_TYPE_DESC,
		common.STRING_TYPE_DESC,
		common.BOOLEAN_TYPE_DESC,
		common.XML_TYPE_DESC,
		common.JSON_TYPE_DESC,
		common.HANDLE_TYPE_DESC,
		common.ANY_TYPE_DESC,
		common.ANYDATA_TYPE_DESC,
		common.NEVER_TYPE_DESC,
		common.VAR_TYPE_DESC,
		common.SERVICE_TYPE_DESC,
		common.MAP_TYPE_DESC,
		common.UNION_TYPE_DESC,
		common.ERROR_TYPE_DESC,
		common.STREAM_TYPE_DESC,
		common.TABLE_TYPE_DESC,
		common.FUNCTION_TYPE_DESC,
		common.TUPLE_TYPE_DESC,
		common.PARENTHESISED_TYPE_DESC,
		common.READONLY_TYPE_DESC,
		common.DISTINCT_TYPE_DESC,
		common.INTERSECTION_TYPE_DESC,
		common.SINGLETON_TYPE_DESC,
		common.TYPE_REFERENCE_TYPE_DESC:
		return true
	default:
		return false
	}
}

// createSimpleLiteral creates a simple literal from a node
func (n *NodeBuilder) createSimpleLiteral(literal tree.Node) LiteralNode {
	return n.createSimpleLiteralInner(literal, n.isInFiniteContext)
}

// getIntegerLiteral parses integer literals (decimal/hex)
func getIntegerLiteral(cx *context.CompilerContext, literal tree.Node, textValue string) any {
	basicLiteralNode := literal.(*tree.BasicLiteralNode)
	literalTokenKind := basicLiteralNode.LiteralToken().Kind()
	switch literalTokenKind {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN:
		if textValue[0] == '0' && len(textValue) > 1 {
			cx.SyntaxError("invalid integer literal: leading zero", getPosition(cx.DiagnosticEnv(), literal))
		}
		return parseLong(textValue, textValue, 10)
	case common.HEX_INTEGER_LITERAL_TOKEN:
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
func isTokenInRegExp(kind common.SyntaxKind) bool {
	switch kind {
	case common.RE_LITERAL_CHAR,
		common.RE_CONTROL_ESCAPE,
		common.RE_NUMERIC_ESCAPE,
		common.RE_SIMPLE_CHAR_CLASS_CODE,
		common.RE_PROPERTY,
		common.RE_UNICODE_SCRIPT_START,
		common.RE_UNICODE_PROPERTY_VALUE,
		common.RE_UNICODE_GENERAL_CATEGORY_START,
		common.RE_UNICODE_GENERAL_CATEGORY_NAME,
		common.RE_FLAGS_VALUE,
		common.DIGIT,
		common.ASTERISK_TOKEN,
		common.PLUS_TOKEN,
		common.QUESTION_MARK_TOKEN,
		common.DOT_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.OPEN_BRACKET_TOKEN,
		common.CLOSE_BRACKET_TOKEN,
		common.OPEN_PAREN_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.DOLLAR_TOKEN,
		common.BITWISE_XOR_TOKEN,
		common.COLON_TOKEN,
		common.BACK_SLASH_TOKEN,
		common.MINUS_TOKEN,
		common.ESCAPED_MINUS_TOKEN,
		common.PIPE_TOKEN,
		common.COMMA_TOKEN:
		return true
	default:
		return false
	}
}

// isNumericLiteral checks if syntax kind is numeric literal
func isNumericLiteral(kind common.SyntaxKind) bool {
	return kind == common.NUMERIC_LITERAL
}

// createSimpleLiteralInner creates a simple literal from a node
func (n *NodeBuilder) createSimpleLiteralInner(literal tree.Node, isFiniteType bool) LiteralNode {
	var bLiteral LiteralNode
	kind := literal.Kind()
	var typeTag TypeTags = -1
	var value any = nil
	var originalValue *string = nil

	var textValue string
	if basicLiteralNode, ok := literal.(*tree.BasicLiteralNode); ok {
		textValue = basicLiteralNode.LiteralToken().Text()
	} else if token, ok := literal.(tree.Token); ok {
		textValue = token.Text()
	} else {
		textValue = ""
	}

	// TODO: Verify all types, only string type tested
	if kind == common.NUMERIC_LITERAL {
		basicLiteralNode := literal.(*tree.BasicLiteralNode)
		literalTokenKind := basicLiteralNode.LiteralToken().Kind()
		switch literalTokenKind {
		case common.DECIMAL_INTEGER_LITERAL_TOKEN, common.HEX_INTEGER_LITERAL_TOKEN:
			typeTag = TypeTags_INT
			value = getIntegerLiteral(n.cx, literal, textValue)
			originalValue = &textValue
			// TODO: can we fix below?
			if literalTokenKind == common.HEX_INTEGER_LITERAL_TOKEN && withinByteRange(value) {
				typeTag = TypeTags_BYTE
			}
		case common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			// TODO: Check effect of mapping negative(-) numbers as unary-expr
			if balCommon.IsDecimalDiscriminated(textValue) {
				typeTag = TypeTags_DECIMAL
			} else {
				typeTag = TypeTags_FLOAT
			}
			if isFiniteType {
				// Remove f, d, and + suffixes
				value = regexp.MustCompile("[fd+]").ReplaceAllString(textValue, "")
				originalValue = new(strings.ReplaceAll(textValue, "+", ""))
			} else {
				value = textValue
				originalValue = &textValue
			}
		default:
			// TODO: Check effect of mapping negative(-) numbers as unary-expr
			typeTag = TypeTags_FLOAT
			value = getHexNodeValue(textValue)
			originalValue = &textValue
		}
		numericLiteral := &BLangNumericLiteral{}
		numericLiteral.pos = getPosition(n.de(), literal)
		numericLiteral.SetValueType(n.types.getTypeFromTag(typeTag).(BType))
		numericLiteral.Value = value
		numericLiteral.OriginalValue = *originalValue
		return &numericLiteral.BLangLiteral
	} else if kind == common.BOOLEAN_LITERAL {
		typeTag = TypeTags_BOOLEAN
		value = strings.ToLower(textValue) == "true"
		originalValue = &textValue
		bLiteral = &BLangLiteral{}
	} else if kind == common.STRING_LITERAL || kind == common.XML_TEXT_CONTENT ||
		kind == common.TEMPLATE_STRING || kind == common.IDENTIFIER_TOKEN ||
		kind == common.PROMPT_CONTENT || isTokenInRegExp(kind) {
		text := textValue
		if kind == common.STRING_LITERAL {
			if len(text) > 1 && text[len(text)-1] == '"' {
				text = text[1 : len(text)-1]
			} else {
				// Missing end quote case
				text = text[1:]
			}
		}

		const identifierLiteralPrefix = "'"
		if kind == common.IDENTIFIER_TOKEN && strings.HasPrefix(text, identifierLiteralPrefix) {
			text = text[1:]
		}

		if kind != common.TEMPLATE_STRING && kind != common.XML_TEXT_CONTENT &&
			kind != common.PROMPT_CONTENT && !isTokenInRegExp(kind) {
			pos := getPosition(n.de(), literal)
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
	} else if kind == common.NIL_LITERAL {
		typeTag = TypeTags_NIL
		value = nil
		originalValue = new(string(model.NIL_VALUE))
		bLiteral = &BLangLiteral{}
	} else if kind == common.NULL_LITERAL {
		originalValue = new("null")
		typeTag = TypeTags_NIL
		bLiteral = &BLangLiteral{}
	} else if kind == common.BINARY_EXPRESSION { // Should be base16 and base64
		typeTag = TypeTags_BYTE_ARRAY
		value = textValue
		originalValue = &textValue

		// If numeric literal create a numeric literal expression; otherwise create a literal expression
		if isNumericLiteral(kind) {
			bLiteral = &BLangNumericLiteral{}
		} else {
			bLiteral = &BLangLiteral{}
		}
	} else if kind == common.BYTE_ARRAY_LITERAL {
		return n.TransformSyntaxNode(literal).(LiteralNode)
	}
	bLangNode := bLiteral.(BLangNode)
	bLangNode.SetPosition(getPosition(n.de(), literal))
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

func (n *NodeBuilder) TransformModulePart(modulePartNode *tree.ModulePart) BLangNode {
	compilationUnit := BLangCompilationUnit{}
	n.currentCompUnit = &compilationUnit
	defer func() { n.currentCompUnit = nil }()
	compilationUnit.Name = n.CurrentCompUnitName
	compilationUnit.packageID = n.PackageID
	pos := getPosition(n.de(), modulePartNode)
	compUnit := createIdentifier(pos, &n.CurrentCompUnitName, &n.CurrentCompUnitName)

	if modulePartNode.HasDiagnostics() {
		n.reportSyntaxDiagnostics(modulePartNode)
	}

	// Generate import declarations
	imports := modulePartNode.Imports()
	for importDecl := range imports.Iterator() {
		if importDecl.HasDiagnostics() {
			continue
		}
		bLangImport := n.TransformImportDeclaration(importDecl).(*BLangImportPackage)
		bLangImport.CompUnit = &compUnit
		compilationUnit.AddTopLevelNode(bLangImport)
	}

	// Generate other module-level declarations
	members := modulePartNode.Members()
	for member := range members.Iterator() {
		if member.HasDiagnostics() {
			continue
		}
		// Dispatch to TransformSyntaxNode which handles all node types
		var memberNode tree.Node = member
		if memberNode.HasDiagnostics() {
			n.reportSyntaxDiagnostics(memberNode)
			continue
		}
		transformedNode := n.TransformSyntaxNode(memberNode)
		node := transformedNode.(TopLevelNode)
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

func setFunctionQualifiers(bLFunction *BLangFunction, qualifierList tree.NodeList[tree.Token]) {
	setFunctionQualifiersOnBase(&bLFunction.bLangInvokableNodeBase, qualifierList)
}

func setFunctionQualifiersOnBase(base *bLangInvokableNodeBase, qualifierList tree.NodeList[tree.Token]) {
	for qualifier := range qualifierList.Iterator() {
		kind := qualifier.Kind()

		switch kind {
		case common.PUBLIC_KEYWORD:
			base.SetPublic()
		case common.PRIVATE_KEYWORD:
			// private is the default
		case common.REMOTE_KEYWORD:
			base.SetRemote()
		case common.TRANSACTIONAL_KEYWORD:
			base.SetTransactional()
		case common.RESOURCE_KEYWORD:
			base.SetResource()
		case common.ISOLATED_KEYWORD:
			base.SetIsolated()
		default:
			// Skip unknown qualifiers
			continue
		}
	}
}

func (n *NodeBuilder) populateFuncSignature(bLFunction *BLangFunction, funcSignature *tree.FunctionSignatureNode) {
	n.populateFuncSignatureOnBase(&bLFunction.bLangInvokableNodeBase, funcSignature)
}

func (n *NodeBuilder) populateFuncSignatureOnBase(bLFunction *bLangInvokableNodeBase, funcSignature *tree.FunctionSignatureNode) {
	// Set Parameters
	parameters := funcSignature.Parameters()
	for param := range parameters.Iterator() {
		// Transform parameter using TransformSyntaxNode
		paramNode := n.TransformSyntaxNode(param).(SimpleVariableNode)

		// Special handling for rest parameters
		if _, isRestParam := param.(*tree.RestParameterNode); isRestParam {
			bLFunction.SetRestParameter(paramNode)
			continue
		}

		// Add to parameters list (all non-rest parameters)
		bLFunction.AddParameter(paramNode)
	}

	// Set Return Type
	retTypeDescNode := funcSignature.ReturnTypeDesc()
	if retTypeDescNode != nil {
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

func (n *NodeBuilder) TransformFunctionDefinition(funcDefNode *tree.FunctionDefinition) BLangNode {
	// Check for resource functions - panic for now
	relativeResourcePath := funcDefNode.RelativeResourcePath()
	hasResourcePath := relativeResourcePath.Size() > 0
	if hasResourcePath {
		panic("TransformFunctionDefinition: resource functions not yet supported")
	}

	// Create function node
	bLFunction := n.createFunctionNode(funcDefNode.FunctionName(), funcDefNode.QualifierList(), funcDefNode.FunctionSignature(), funcDefNode.FunctionBody())
	bLFunction.pos = getPositionWithoutMetadata(n.de(), funcDefNode)

	metadata := funcDefNode.Metadata()
	n.populateMetadata(metadata, bLFunction)

	return bLFunction
}

func (n *NodeBuilder) createFunctionNode(funcName *tree.IdentifierToken, qualifierList tree.NodeList[tree.Token], funcSignature *tree.FunctionSignatureNode, funcBody tree.FunctionBodyNode) *BLangFunction {
	blFunction := BLangFunction{}
	name := createIdentifierFromTokenInternal(getPosition(n.de(), funcName), funcName, false)
	n.populateFunctionNode(name, qualifierList, funcSignature, funcBody, &blFunction)
	return &blFunction
}

func (n *NodeBuilder) populateFunctionNode(name BLangIdentifier, qualifierList tree.NodeList[tree.Token], funcSignature *tree.FunctionSignatureNode, funcBody tree.FunctionBodyNode, blFunction *BLangFunction) {
	// Set function name
	blFunction.Name = name
	// Set method qualifiers
	setFunctionQualifiers(blFunction, qualifierList)
	// Set function signature
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, name.Value)
	n.populateFuncSignature(blFunction, funcSignature)
	n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]

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

func (n *NodeBuilder) TransformImportDeclaration(importDeclarationNode *tree.ImportDeclarationNode) BLangNode {
	// 1. Extract org name (optional)
	orgNameNode := importDeclarationNode.OrgName()
	var orgNameToken tree.Token
	if orgNameNode != nil && !orgNameNode.IsMissing() {
		orgNameToken = orgNameNode.OrgName()
	}

	// 2. Extract prefix node (optional)
	prefixNode := importDeclarationNode.Prefix()

	// 3. Get position for entire import declaration
	position := getPosition(n.de(), importDeclarationNode)

	// 4. Process module name components
	var pkgNameComps []BLangIdentifier
	moduleNames := importDeclarationNode.ModuleName()
	for name := range moduleNames.Iterator() {
		namePos := getPosition(n.de(), name)
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
		orgNamePos = getPosition(n.de(), orgNameNode)
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
	prefixPos := getPosition(n.de(), prefix)

	if prefix.Kind() == common.UNDERSCORE_KEYWORD {
		// Create ignore identifier for underscore
		aliasIdent := createIgnoreIdentifier(n.de(), prefix)
		importDcl.Alias = &aliasIdent
	} else {
		// Use prefix token as alias
		prefixText := prefix.Text()
		aliasIdent := createIdentifier(prefixPos, &prefixText, &prefixText)
		importDcl.Alias = &aliasIdent
	}

	return importDcl
}

func (n *NodeBuilder) TransformListenerDeclaration(listenerDeclarationNode *tree.ListenerDeclarationNode) BLangNode {
	metadata := listenerDeclarationNode.Metadata()

	pos := getPositionWithoutMetadata(n.de(), listenerDeclarationNode)
	nameToken := listenerDeclarationNode.VariableName()
	namePos := getPosition(n.de(), nameToken)
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

	if visQual := listenerDeclarationNode.VisibilityQualifier(); visQual != nil && visQual.Kind() == common.PUBLIC_KEYWORD {
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

func isAllowedDistinctTypeDescriptor(kind common.SyntaxKind) bool {
	switch kind {
	case common.OBJECT_TYPE_DESC, common.ERROR_TYPE_DESC, common.SIMPLE_NAME_REFERENCE, common.QUALIFIED_NAME_REFERENCE, common.IDENTIFIER_TOKEN:
		return true
	default:
		return false
	}
}

func (n *NodeBuilder) TransformTypeDefinition(typeDefinitionNode *tree.TypeDefinitionNode) BLangNode {
	typeDef := NewBLangTypeDefinition()

	identifierNode := createIdentifierFromToken(getPosition(n.de(), typeDefinitionNode.TypeName()), typeDefinitionNode.TypeName())
	typeDef.Name = &identifierNode

	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, typeDef.Name.GetValue())

	typeDescriptorNode := typeDefinitionNode.TypeDescriptor()
	if distinctTypeDescriptorNode, ok := typeDescriptorNode.(*tree.DistinctTypeDescriptorNode); ok {
		innerTypeDescriptorNode := distinctTypeDescriptorNode.TypeDescriptor()
		if innerTypeDescriptorNode == nil || !isAllowedDistinctTypeDescriptor(innerTypeDescriptorNode.Kind()) {
			n.cx.SyntaxError("only object and error types can be distinct", getPosition(n.de(), distinctTypeDescriptorNode))
			neverType := &BLangValueType{TypeKind: TypeKind_NEVER}
			neverType.pos = getPosition(n.de(), distinctTypeDescriptorNode)
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
	if visibilityQualifier != nil && visibilityQualifier.Kind() == common.PUBLIC_KEYWORD {
		typeDef.SetPublic()
	}

	typeDef.pos = getPositionWithoutMetadata(n.de(), typeDefinitionNode)

	n.populateMetadata(typeDefinitionNode.Metadata(), typeDef)

	return typeDef
}

func (n *NodeBuilder) TransformServiceDeclaration(serviceDeclarationNode *tree.ServiceDeclarationNode) BLangNode {
	metadata := serviceDeclarationNode.Metadata()

	service := NewBLangService()
	service.pos = getPositionWithoutMetadata(n.de(), serviceDeclarationNode)

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
func (n *NodeBuilder) populateServiceQualifiers(service *BLangService, node *tree.ServiceDeclarationNode) {
	quals := node.Qualifiers()
	for qual := range quals.Iterator() {
		if qual.Kind() == common.ISOLATED_KEYWORD {
			service.SetIsolated()
		}
	}
}

func (n *NodeBuilder) populateServiceAttachPoint(service *BLangService, node *tree.ServiceDeclarationNode) {
	paths := node.AbsoluteResourcePath()
	if node.HasDiagnostics() {
		n.reportSyntaxDiagnostics(node)
		return
	}
	for i := 0; i < paths.Size(); i++ {
		seg := paths.Get(i)
		tok, ok := seg.(tree.Token)
		if !ok {
			n.cx.InternalError("unexpected node in service attach point", getPosition(n.de(), seg))
			continue
		}
		switch tok.Kind() {
		case common.STRING_LITERAL:
			lit, ok := n.createExpression(tok).(*BLangLiteral)
			if !ok {
				n.cx.InternalError("invalid service attach point literal", getPosition(n.de(), tok))
				continue
			}
			if _, isString := lit.GetValue().(string); !isString {
				n.cx.InternalError("service attach point literal must be a string", getPosition(n.de(), tok))
				continue
			}
			service.AttachPointLiteral = lit
		case common.IDENTIFIER_TOKEN:
			ident := createIdentifierFromToken(getPosition(n.de(), tok), tok)
			service.AbsoluteResourcePath = append(service.AbsoluteResourcePath, ident)
		case common.SLASH_TOKEN:
			// Slash tokens between segments are ignored.
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected token in service attach point: %v", tok.Kind()), getPosition(n.de(), tok))
		}
	}
}

func (n *NodeBuilder) populateServiceAttachedExprs(service *BLangService, node *tree.ServiceDeclarationNode) {
	exprs := node.Expressions()
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

func (n *NodeBuilder) collectClassDefnMembers(memberNodes tree.NodeList[tree.Node]) classDefnMembers {
	members := newClassDefnMembers()
	for i := 0; i < memberNodes.Size(); i++ {
		member := memberNodes.Get(i)
		switch member.Kind() {
		case common.OBJECT_FIELD:
			field := n.transformClassField(member.(*tree.ObjectFieldNode))
			members.Fields = append(members.Fields, field)
		case common.FUNCTION_DEFINITION, common.OBJECT_METHOD_DEFINITION:
			n.addCollectedMethod(&members, member.(*tree.FunctionDefinition))
		case common.RESOURCE_ACCESSOR_DEFINITION:
			rm := n.createResourceMethodNode(member.(*tree.FunctionDefinition))
			members.ResourceMethods = append(members.ResourceMethods, rm)
		case common.TYPE_REFERENCE:
			typeRef := member.(*tree.TypeReferenceNode)
			members.UnresolvedInclusions = append(members.UnresolvedInclusions, n.createTypeNode(typeRef.TypeName()).(*BLangUserDefinedType))
		default:
			panic("collectClassDefnMembers: unsupported member kind")
		}
	}
	return members
}

func (n *NodeBuilder) addCollectedMethod(members *classDefnMembers, funcDef *tree.FunctionDefinition) {
	bLFunction := n.createFunctionNode(funcDef.FunctionName(), funcDef.QualifierList(), funcDef.FunctionSignature(), funcDef.FunctionBody())
	bLFunction.pos = getPositionWithoutMetadata(n.de(), funcDef)
	bLFunction.SetAttached()
	n.populateMetadata(funcDef.Metadata(), bLFunction)

	funcName := bLFunction.Name.Value
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
		bLFunction.Name.Value = funcName
	}
	if _, exists := members.Methods[funcName]; exists {
		n.cx.SyntaxError("redeclared symbol '"+model.StripRemotePrefix(funcName)+"'", bLFunction.pos)
		return
	}
	members.Methods[funcName] = bLFunction
}

func (n *NodeBuilder) TransformAssignmentStatement(assignmentStatementNode *tree.AssignmentStatementNode) BLangNode {
	lhsKind := assignmentStatementNode.VarRef().Kind()
	switch lhsKind {
	case common.LIST_BINDING_PATTERN, common.MAPPING_BINDING_PATTERN, common.ERROR_BINDING_PATTERN:
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
	bLAssignment.pos = getPosition(n.de(), assignmentStatementNode)
	bLAssignment.VarRef = lhsExpr.(LExpr)
	return bLAssignment
}

func (n *NodeBuilder) TransformCompoundAssignmentStatement(compoundAssignmentStmtNode *tree.CompoundAssignmentStatementNode) BLangNode {
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
	BLangNode(bLCompAssignment).SetPosition(getPosition(n.de(), compoundAssignmentStmtNode))
	bLCompAssignment.OpKind = model.OperatorKindValueFrom(compoundAssignmentStmtNode.BinaryOperator().Text())
	return bLCompAssignment
}

func (n *NodeBuilder) TransformVariableDeclaration(variableDeclarationNode *tree.VariableDeclarationNode) BLangNode {
	varNode := n.createBLangVarDef(
		getPosition(n.de(), variableDeclarationNode),
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

func (n *NodeBuilder) createBLangVarDef(location diagnostics.Location, typedBindingPattern *tree.TypedBindingPatternNode, initializer tree.ExpressionNode, finalKeyword tree.Token) VariableDefinitionNode {
	bindingPattern := typedBindingPattern.BindingPattern()

	variable := n.getBLangVariableNode(bindingPattern, location)

	var qualifiers []tree.Token
	if finalKeyword != nil {
		qualifiers = append(qualifiers, finalKeyword) //nolint:staticcheck,ineffassign // qualifierList creation not yet implemented
	}
	// qualifierList := tree.CreateNodeListWithFacade(qualifiers)

	switch bindingPattern.Kind() {
	case common.CAPTURE_BINDING_PATTERN, common.WILDCARD_BINDING_PATTERN:
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

	case common.MAPPING_BINDING_PATTERN:
		panic("MAPPING_BINDING_PATTERN unimplemented")

	case common.LIST_BINDING_PATTERN:
		panic("LIST_BINDING_PATTERN unimplemented")

	case common.ERROR_BINDING_PATTERN:
		panic("ERROR_BINDING_PATTERN unimplemented")

	default:
		panic("Syntax kind is not a valid binding pattern")
	}
}

func (n *NodeBuilder) TransformBlockStatement(blockStatementNode *tree.BlockStatementNode) BLangNode {
	bLBlockStmt := BLangBlockStmt{}
	n.isInLocalContext = true
	bLBlockStmt.Stmts = n.generateBLangStatements(blockStatementNode.Statements(), blockStatementNode)
	n.isInLocalContext = false
	bLBlockStmt.pos = getPosition(n.de(), blockStatementNode)
	return &bLBlockStmt
}

func (n *NodeBuilder) generateBLangStatements(statementNodes tree.NodeList[tree.StatementNode], endNode tree.Node) []StatementNode {
	statements := []StatementNode{}
	return *n.generateAndAddBLangStatements(statementNodes, &statements, 0, endNode)
}

func (n *NodeBuilder) generateAndAddBLangStatements(statementNodes tree.NodeList[tree.StatementNode], statements *[]StatementNode, startPosition int, endNode tree.Node) *[]StatementNode {
	lastStmtIndex := statementNodes.Size() - 1
	for j := startPosition; j < statementNodes.Size(); j++ {
		currentStatement := statementNodes.Get(j)
		// TODO: Remove this check once statements are non null guaranteed
		if currentStatement == nil {
			continue
		}
		if currentStatement.HasDiagnostics() {
			continue
		}
		if currentStatement.Kind() == common.FORK_STATEMENT {
			forkStmt := currentStatement.(*tree.ForkStatementNode)
			n.generateForkStatements(statements, forkStmt)
			continue
		}
		// If there is an `if` statement without an `else`, all the statements following that `if` statement
		// are added to a new block statement.
		if ifElseStmt, ok := currentStatement.(*tree.IfElseStatementNode); ok && ifElseStmt.ElseBody() == nil {
			*statements = append(*statements, n.TransformSyntaxNode(currentStatement).(StatementNode))
			if j == lastStmtIndex {
				// Add an empty block statement if there are no statements following the `if` statement.
				emptyBlock := &BLangBlockStmt{}
				emptyBlock.pos = getPositionRange(n.de(), currentStatement, endNode)
				*statements = append(*statements, emptyBlock)
				break
			}
			bLBlockStmt := &BLangBlockStmt{}
			nextStmtIndex := j + 1
			n.isInLocalContext = true
			n.generateAndAddBLangStatements(statementNodes, &bLBlockStmt.Stmts, nextStmtIndex, endNode)
			n.isInLocalContext = false
			if nextStmtIndex <= lastStmtIndex {
				bLBlockStmt.pos = getPositionRange(n.de(), statementNodes.Get(nextStmtIndex), endNode)
			}
			*statements = append(*statements, bLBlockStmt)
			break
		} else {
			*statements = append(*statements, n.TransformSyntaxNode(currentStatement).(StatementNode))
		}
	}
	return statements
}

func (n *NodeBuilder) TransformBreakStatement(breakStatementNode *tree.BreakStatementNode) BLangNode {
	bLBreak := &BLangBreak{}
	bLBreak.pos = getPosition(n.de(), breakStatementNode)
	return bLBreak
}

func (n *NodeBuilder) TransformFailStatement(failStatementNode *tree.FailStatementNode) BLangNode {
	panic("TransformFailStatement unimplemented")
}

func (n *NodeBuilder) TransformExpressionStatement(expressionStatement *tree.ExpressionStatementNode) BLangNode {
	bLExpressionStmt := BLangExpressionStmt{}
	bLExpressionStmt.Expr = n.createActionOrExpression(expressionStatement.Expression())
	bLExpressionStmt.pos = getPosition(n.de(), expressionStatement)
	return &bLExpressionStmt
}

// createSpecificFieldNameLiteral builds a string-literal expression for a
// non-computed mapping-constructor key. The field name is a static identifier
// or string literal, not a runtime expression, so it must not be represented
// as a var-ref.
func (n *NodeBuilder) createSpecificFieldNameLiteral(fieldName tree.Node) BLangExpression {
	if basicLit, ok := fieldName.(*tree.BasicLiteralNode); ok {
		return n.createSimpleLiteral(basicLit).(BLangExpression)
	}
	nameRef := n.createBLangNameReference(fieldName)
	name := nameRef[1].GetValue()
	pos := getPosition(n.de(), fieldName)
	lit := &BLangLiteral{}
	lit.SetPosition(pos)
	bType := &BTypeBasic{}
	bType.BTypeSetTag(TypeTags_STRING)
	lit.SetValueType(bType)
	lit.SetValue(name)
	lit.SetOriginalValue(name)
	return lit
}

func (n *NodeBuilder) createExpression(expressionNode tree.Node) BLangExpression {
	return n.createActionOrExpression(expressionNode).(BLangExpression) //nolint:forcetypeassert // only called where expressions are expected, not actions
}

// createActionOrExpression creates an action or expression node from a syntax tree node
func (n *NodeBuilder) createActionOrExpression(actionOrExpression tree.Node) BLangActionOrExpression {
	if isSimpleLiteral(actionOrExpression.Kind()) {
		return n.createSimpleLiteral(actionOrExpression).(BLangActionOrExpression)
	} else if actionOrExpression.Kind() == common.SIMPLE_NAME_REFERENCE ||
		actionOrExpression.Kind() == common.QUALIFIED_NAME_REFERENCE ||
		actionOrExpression.Kind() == common.IDENTIFIER_TOKEN {
		nameReference := n.createBLangNameReference(actionOrExpression)
		bLVarRef := BLangSimpleVarRef{}
		bLVarRef.pos = getPosition(n.de(), actionOrExpression)
		bLVarRef.PkgAlias = new(createIdentifier(nameReference[0].GetPosition(), new(nameReference[0].GetValue()), new(nameReference[0].GetValue())))
		bLVarRef.VariableName = new(createIdentifier(nameReference[1].GetPosition(), new(nameReference[1].GetValue()), new(nameReference[1].GetValue())))
		return &bLVarRef

	} else if actionOrExpression.Kind() == common.BRACED_EXPRESSION {
		group := BLangGroupExpr{}
		group.Expression = n.TransformSyntaxNode(actionOrExpression).(BLangExpression)
		group.pos = getPosition(n.de(), actionOrExpression)
		return &group
	} else if isType(actionOrExpression.Kind()) {
		typeAccessExpr := BLangTypedescExpr{}
		typeAccessExpr.pos = getPosition(n.de(), actionOrExpression)
		typeAccessExpr.typeDescriptor = n.createTypeNode(actionOrExpression)
		return &typeAccessExpr
	} else {
		return n.TransformSyntaxNode(actionOrExpression).(BLangActionOrExpression)
	}
}

func (n *NodeBuilder) TransformContinueStatement(continueStatementNode *tree.ContinueStatementNode) BLangNode {
	blContinue := &BLangContinue{}
	blContinue.pos = getPosition(n.de(), continueStatementNode)
	return blContinue
}

func (n *NodeBuilder) TransformExternalFunctionBody(externalFunctionBodyNode *tree.ExternalFunctionBodyNode) BLangNode {
	body := &BLangExternFunctionBody{}
	body.pos = getPosition(n.de(), externalFunctionBodyNode)
	return body
}

func (n *NodeBuilder) TransformIfElseStatement(ifElseStatementNode *tree.IfElseStatementNode) BLangNode {
	bLIf := BLangIf{}
	bLIf.pos = getPosition(n.de(), ifElseStatementNode)
	bLIf.SetCondition(n.createExpression(ifElseStatementNode.Condition()))
	bLIf.SetBody(n.TransformBlockStatement(ifElseStatementNode.IfBody()).(*BLangBlockStmt))
	if ifElseStatementNode.ElseBody() != nil {
		elseNode := ifElseStatementNode.ElseBody().(*tree.ElseBlockNode)
		bLIf.SetElseStatement(n.TransformSyntaxNode(elseNode.ElseBody()).(StatementNode))
	}
	return &bLIf
}

func (n *NodeBuilder) TransformElseBlock(elseBlockNode *tree.ElseBlockNode) BLangNode {
	panic("TransformElseBlock unimplemented")
}

func (n *NodeBuilder) TransformWhileStatement(whileStatementNode *tree.WhileStatementNode) BLangNode {
	bLWhile := &BLangWhile{}
	bLWhile.SetCondition(n.createExpression(whileStatementNode.Condition()))
	bLWhile.pos = getPosition(n.de(), whileStatementNode)

	bLBlockStmt := n.TransformBlockStatement(whileStatementNode.WhileBody()).(*BLangBlockStmt)
	bLBlockStmt.pos = getPosition(n.de(), whileStatementNode.WhileBody())
	bLWhile.SetBody(bLBlockStmt)
	if whileStatementNode.OnFailClause() != nil {
		onFailClauseNode := whileStatementNode.OnFailClause()
		bLWhile.SetOnFailClause(n.TransformOnFailClause(onFailClauseNode).(*BLangOnFailClause))
	} else {
		bLWhile.OnFailClause.pos = diagnostics.NewBuiltinLocation()
	}
	return bLWhile
}

func (n *NodeBuilder) TransformPanicStatement(panicStatementNode *tree.PanicStatementNode) BLangNode {
	bLPanic := &BLangPanic{}
	bLPanic.pos = getPosition(n.de(), panicStatementNode)
	bLPanic.Expr = n.createExpression(panicStatementNode.Expression())
	return bLPanic
}

func (n *NodeBuilder) TransformReturnStatement(returnStatementNode *tree.ReturnStatementNode) BLangNode {
	bLReturn := &BLangReturn{}
	bLReturn.pos = getPosition(n.de(), returnStatementNode)
	if returnStatementNode.Expression() != nil {
		bLReturn.SetActionOrExpression(n.createActionOrExpression(returnStatementNode.Expression()))
	} else {
		nilLiteral := &BLangLiteral{}
		nilLiteral.pos = getPosition(n.de(), returnStatementNode)
		nilLiteral.Value = nil
		nilLiteral.SetValueType(n.types.getTypeFromTag(TypeTags_NIL).(BType))
		bLReturn.SetActionOrExpression(nilLiteral)
	}

	return bLReturn
}

func (n *NodeBuilder) TransformLocalTypeDefinitionStatement(localTypeDefinitionStatementNode *tree.LocalTypeDefinitionStatementNode) BLangNode {
	panic("TransformLocalTypeDefinitionStatement unimplemented")
}

func (n *NodeBuilder) TransformLockStatement(lockStatementNode *tree.LockStatementNode) BLangNode {
	if lockStatementNode.OnFailClause() != nil {
		n.cx.Unimplemented("on-fail clause on lock is not yet supported", getPosition(n.de(), lockStatementNode.OnFailClause()))
	}
	bLLock := &BLangLock{}
	bLLock.pos = getPosition(n.de(), lockStatementNode)
	bLBlockStmt := n.TransformBlockStatement(lockStatementNode.BlockStatement()).(*BLangBlockStmt)
	bLBlockStmt.pos = getPosition(n.de(), lockStatementNode.BlockStatement())
	bLLock.Body = *bLBlockStmt
	return bLLock
}

func (n *NodeBuilder) TransformForkStatement(forkStatementNode *tree.ForkStatementNode) BLangNode {
	panic("TransformForkStatement unimplemented")
}

func (n *NodeBuilder) TransformForEachStatement(forEachStatementNode *tree.ForEachStatementNode) BLangNode {
	bLForeach := &BLangForeach{}
	bLForeach.pos = getPosition(n.de(), forEachStatementNode)

	varDef := n.createBLangVarDef(
		getPosition(n.de(), forEachStatementNode.TypedBindingPattern()),
		forEachStatementNode.TypedBindingPattern(),
		nil,
		nil,
	).(*BLangSimpleVariableDef)
	bLForeach.VariableDef = varDef
	bLForeach.IsDeclaredWithVar = varDef.Var.IsDeclaredWithVar

	bLForeach.Collection = n.createExpression(forEachStatementNode.ActionOrExpressionNode())

	body := n.TransformBlockStatement(forEachStatementNode.BlockStatement()).(*BLangBlockStmt)
	body.pos = getPosition(n.de(), forEachStatementNode.BlockStatement())
	bLForeach.Body = *body

	if forEachStatementNode.OnFailClause() != nil {
		bLForeach.SetOnFailClause(
			n.TransformOnFailClause(forEachStatementNode.OnFailClause()).(*BLangOnFailClause),
		)
	}
	return bLForeach
}

func (n *NodeBuilder) TransformBinaryExpression(binaryBLangExpression *tree.BinaryExpressionNode) BLangNode {
	if binaryBLangExpression.Operator().Kind() == common.ELVIS_TOKEN {
		panic("TransformBinaryExpression: elvis operator not supported")
	}

	bLBinaryExpr := BLangBinaryExpr{}
	bLBinaryExpr.pos = getPosition(n.de(), binaryBLangExpression)
	bLBinaryExpr.LhsExpr = n.createExpression(binaryBLangExpression.LhsExpr())
	bLBinaryExpr.RhsExpr = n.createExpression(binaryBLangExpression.RhsExpr())
	if binaryBLangExpression.Operator() == nil {
		n.cx.InternalError("binary expression is missing an operator token", bLBinaryExpr.pos)
		return &bLBinaryExpr
	}
	bLBinaryExpr.OpKind = model.OperatorKindValueFrom(binaryBLangExpression.Operator().Text())
	return &bLBinaryExpr
}

func (n *NodeBuilder) TransformBracedExpression(bracedBLangExpression *tree.BracedExpressionNode) BLangNode {
	return n.createExpression(bracedBLangExpression.Expression())
}

func (n *NodeBuilder) TransformCheckExpression(checkBLangExpression *tree.CheckExpressionNode) BLangNode {
	pos := getPosition(n.de(), checkBLangExpression)
	// we are deviating from the spec here (https://ballerina.io/spec/lang/master/#section_6.33) check is only suppose
	// to work with expression but jBallerina also allow remote method calls (which is an action)
	expr := n.createActionOrExpression(checkBLangExpression.Expression())
	if checkBLangExpression.CheckKeyword().Kind() == common.CHECK_KEYWORD {
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

func (n *NodeBuilder) TransformFieldAccessExpression(fieldAccessBLangExpression *tree.FieldAccessExpressionNode) BLangNode {
	fieldName := fieldAccessBLangExpression.FieldName()
	if fieldName.Kind() == common.QUALIFIED_NAME_REFERENCE {
		panic("TransformFieldAccessExpression: QUALIFIED_NAME_REFERENCE unsupported")
	}

	bLFieldBasedAccess := &BLangFieldBaseAccess{}
	simpleNameRef := fieldName.(*tree.SimpleNameReferenceNode)
	bLFieldBasedAccess.Field = createIdentifierFromToken(getPosition(n.de(), fieldAccessBLangExpression.FieldName()), simpleNameRef.Name())

	containerExpr := fieldAccessBLangExpression.Expression()
	if containerExpr.Kind() == common.BRACED_EXPRESSION {
		bracedExpr := containerExpr.(*tree.BracedExpressionNode)
		bLFieldBasedAccess.Expr = n.createExpression(bracedExpr.Expression())
	} else {
		bLFieldBasedAccess.Expr = n.createExpression(containerExpr)
	}

	bLFieldBasedAccess.pos = getPosition(n.de(), fieldAccessBLangExpression)
	return bLFieldBasedAccess
}

func (n *NodeBuilder) TransformFunctionCallExpression(functionCallBLangExpression *tree.FunctionCallExpressionNode) BLangNode {
	return n.createBLangInvocation(
		functionCallBLangExpression.FunctionName(),
		functionCallBLangExpression.Arguments(),
		getPosition(n.de(), functionCallBLangExpression),
		n.isFunctionCallAsync(functionCallBLangExpression))
}

func (n *NodeBuilder) TransformMethodCallExpression(methodCallBLangExpression *tree.MethodCallExpressionNode) BLangNode {
	bLInvocation := n.createBLangInvocation(methodCallBLangExpression.MethodName(),
		methodCallBLangExpression.Arguments(),
		getPosition(n.de(), methodCallBLangExpression), false)
	bLInvocation.Expr = n.createExpression(methodCallBLangExpression.Expression())
	return bLInvocation
}

func (n *NodeBuilder) TransformMappingConstructorExpression(mappingConstructorBLangExpression *tree.MappingConstructorExpressionNode) BLangNode {
	mappingConstructor := &BLangMappingConstructorExpr{
		Fields: make([]MappingField, 0),
	}
	fields := mappingConstructorBLangExpression.FieldNodes()
	for i := 0; i < fields.Size(); i += 2 {
		field := fields.Get(i)
		switch field.Kind() {
		case common.SPREAD_FIELD:
			panic("mapping constructor spread field not implemented")
		case common.COMPUTED_NAME_FIELD:
			computedNameField := field.(*tree.ComputedNameFieldNode)
			keyExpr := n.createExpression(computedNameField.FieldNameExpr())
			key := &BLangMappingKey{
				Expr: keyExpr,
				Kind: MappingKeyComputed,
			}
			key.SetPosition(getPosition(n.de(), computedNameField.FieldNameExpr()))
			keyValueField := &BLangMappingKeyValueField{
				Key:       key,
				ValueExpr: n.createExpression(computedNameField.ValueExpr()),
			}
			keyValueField.SetPosition(getPosition(n.de(), computedNameField))
			mappingConstructor.Fields = append(mappingConstructor.Fields, keyValueField)
		case common.SPECIFIC_FIELD:
			specificField := field.(*tree.SpecificFieldNode)
			if specificField.ValueExpr() == nil {
				panic("mapping constructor var-name field not implemented")
			}
			_, isStringLit := specificField.FieldName().(*tree.BasicLiteralNode)
			keyKind := MappingKeyIdentifier
			if isStringLit {
				keyKind = MappingKeyStringLiteral
			}
			key := &BLangMappingKey{
				Expr: n.createSpecificFieldNameLiteral(specificField.FieldName()),
				Kind: keyKind,
			}
			key.SetPosition(getPosition(n.de(), specificField.FieldName()))
			keyValueField := &BLangMappingKeyValueField{
				Key:       key,
				ValueExpr: n.createExpression(specificField.ValueExpr()),
				Readonly:  specificField.ReadonlyKeyword() != nil,
			}
			keyValueField.SetPosition(getPosition(n.de(), specificField))
			mappingConstructor.Fields = append(mappingConstructor.Fields, keyValueField)
		default:
			panic(fmt.Sprintf("unexpected mapping field kind: %v", field.Kind()))
		}
	}
	mappingConstructor.SetPosition(getPosition(n.de(), mappingConstructorBLangExpression))
	return mappingConstructor
}

func (n *NodeBuilder) TransformIndexedExpression(indexedBLangExpression *tree.IndexedExpressionNode) BLangNode {
	indexBasedAccess := &BLangIndexBasedAccess{}
	indexBasedAccess.pos = getPosition(n.de(), indexedBLangExpression)
	keys := indexedBLangExpression.KeyExpression()
	if keys.Size() == 0 {
		panic("missing key expression in member access expression")
	} else if keys.Size() == 1 {
		indexBasedAccess.IndexExpr = n.createExpression(keys.Get(0))
	} else {
		listConstructorExpr := &BLangListConstructorExpr{}
		listConstructorExpr.pos = getPositionRange(n.de(), keys.Get(0), keys.Get(keys.Size()-1))
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

func (n *NodeBuilder) TransformTypeofExpression(typeofBLangExpression *tree.TypeofExpressionNode) BLangNode {
	panic("TransformTypeofExpression unimplemented")
}

func (n *NodeBuilder) TransformUnaryExpression(unaryBLangExpression *tree.UnaryExpressionNode) BLangNode {
	pos := getPosition(n.de(), unaryBLangExpression)
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

func (n *NodeBuilder) TransformComputedNameField(computedNameFieldNode *tree.ComputedNameFieldNode) BLangNode {
	panic("TransformComputedNameField unimplemented")
}

func (n *NodeBuilder) TransformConstantDeclaration(constantDeclarationNode *tree.ConstantDeclarationNode) BLangNode {
	// Line 940: BLangConstant constantNode = (BLangConstant) TreeBuilder.createConstantNode();
	constantNode := createConstantNode()

	pos := getPositionWithoutMetadata(n.de(), constantDeclarationNode)

	identifierPos := getPosition(n.de(), constantDeclarationNode.VariableName())

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
	if visibilityQualifier != nil && visibilityQualifier.Kind() == common.PUBLIC_KEYWORD {
		constantNode.SetPublic()
	}

	constantName := constantNode.Name.GetValue()

	if initializedValue, exists := n.constantSet[constantName]; exists {
		if initializedValue != "" {
			n.cx.SemanticError(
				fmt.Sprintf("symbol '%s' is already initialized with '%s'", constantName, initializedValue),
				constantNode.Name.GetPosition(),
			)
		} else {
			n.cx.SemanticError(
				fmt.Sprintf("symbol '%s' is already initialized", constantName),
				constantNode.Name.GetPosition(),
			)
		}
	} else {
		n.constantSet[constantName] = getConstantInitValue(constantNode.Expr)
	}

	return constantNode
}

func (n *NodeBuilder) TransformDefaultableParameter(defaultableParameterNode *tree.DefaultableParameterNode) BLangNode {
	paramName := defaultableParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarInner(paramName, defaultableParameterNode.TypeName(), defaultableParameterNode.Expression(), nil, defaultableParameterNode.Annotations())

	simpleVar.pos = getPosition(n.de(), defaultableParameterNode)

	if paramName != nil {
		simpleVar.Name.pos = getPosition(n.de(), paramName)
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.pos) {
		simpleVar.Name.pos = diagnostics.NewBuiltinLocation()
	}

	simpleVar.SetDefaultableParam()

	return simpleVar
}

func (n *NodeBuilder) createSimpleVarWithTokenNodeNodeList(name tree.Token, typeName tree.Node, annotations tree.NodeList[*tree.AnnotationNode]) *BLangSimpleVariable {
	if name != nil {
		return n.createSimpleVarInner(name, typeName, nil, nil, annotations)
	}
	return n.createSimpleVarInner(nil, typeName, nil, nil, annotations)
}

func (n *NodeBuilder) TransformRequiredParameter(requiredParameterNode *tree.RequiredParameterNode) BLangNode {
	paramName := requiredParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, requiredParameterNode.TypeName(), requiredParameterNode.Annotations())

	simpleVar.pos = getPosition(n.de(), requiredParameterNode)

	if paramName != nil {
		simpleVar.Name.pos = getPosition(n.de(), paramName)
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.pos) {
		// Param doesn't have a name and also is not a missing node
		// Therefore, assigning the built-in location
		simpleVar.Name.pos = diagnostics.NewBuiltinLocation()
	}

	simpleVar.SetRequiredParam()

	return simpleVar
}

func (n *NodeBuilder) TransformIncludedRecordParameter(includedRecordParameterNode *tree.IncludedRecordParameterNode) BLangNode {
	paramName := includedRecordParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, includedRecordParameterNode.TypeName(), includedRecordParameterNode.Annotations())

	simpleVar.pos = getPosition(n.de(), includedRecordParameterNode)

	if paramName != nil {
		simpleVar.Name.pos = getPosition(n.de(), paramName)
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.pos) {
		simpleVar.Name.pos = diagnostics.NewBuiltinLocation()
	}

	simpleVar.SetRequiredParam()
	simpleVar.SetIncludedRecordParam()

	return simpleVar
}

func (n *NodeBuilder) TransformRestParameter(restParameterNode *tree.RestParameterNode) BLangNode {
	paramName := restParameterNode.ParamName()

	if paramName != nil {
		n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, paramName.Text())
	}

	simpleVar := n.createSimpleVarWithTokenNodeNodeList(paramName, restParameterNode.TypeName(), restParameterNode.Annotations())

	simpleVar.pos = getPosition(n.de(), restParameterNode)

	if paramName != nil {
		simpleVar.Name.pos = getPosition(n.de(), paramName)
		n.anonTypeNameSuffixes = n.anonTypeNameSuffixes[:len(n.anonTypeNameSuffixes)-1]
	} else if diagnostics.IsLocationEmpty(simpleVar.Name.pos) {
		simpleVar.Name.pos = diagnostics.NewBuiltinLocation()
	}

	simpleVar.SetRestParam()

	return simpleVar
}

func (n *NodeBuilder) TransformImportOrgName(importOrgNameNode *tree.ImportOrgNameNode) BLangNode {
	panic("TransformImportOrgName unimplemented")
}

func (n *NodeBuilder) TransformImportPrefix(importPrefixNode *tree.ImportPrefixNode) BLangNode {
	panic("TransformImportPrefix unimplemented")
}

func (n *NodeBuilder) TransformSpecificField(specificFieldNode *tree.SpecificFieldNode) BLangNode {
	panic("TransformSpecificField unimplemented")
}

func (n *NodeBuilder) TransformSpreadField(spreadFieldNode *tree.SpreadFieldNode) BLangNode {
	panic("TransformSpreadField unimplemented")
}

func (n *NodeBuilder) TransformNamedArgument(namedArgumentNode *tree.NamedArgumentNode) BLangNode {
	namedArg := &BLangNamedArgsExpression{}
	namedArg.pos = getPosition(n.de(), namedArgumentNode)
	namedArg.Name = createIdentifierFromToken(getPosition(n.de(), namedArgumentNode.ArgumentName()), namedArgumentNode.ArgumentName().Name())
	namedArg.Expr = n.createExpression(namedArgumentNode.Expression())
	return namedArg
}

func (n *NodeBuilder) TransformPositionalArgument(positionalArgumentNode *tree.PositionalArgumentNode) BLangNode {
	return n.createExpression(positionalArgumentNode.Expression())
}

func (n *NodeBuilder) TransformRestArgument(restArgumentNode *tree.RestArgumentNode) BLangNode {
	panic("TransformRestArgument unimplemented")
}

func (n *NodeBuilder) TransformInferredTypedescDefault(inferredTypedescDefaultNode *tree.InferredTypedescDefaultNode) BLangNode {
	node := &BLangInferredTypedescDefault{}
	node.pos = getPosition(n.de(), inferredTypedescDefaultNode)
	return node
}

func (n *NodeBuilder) TransformObjectTypeDescriptor(objectTypeDescriptorNode *tree.ObjectTypeDescriptorNode) BLangNode {
	objectType := &BLangObjectType{members: make(map[string]ObjectMember)}

	// Process object type qualifiers (client/service/isolated)
	qualifiers := objectTypeDescriptorNode.ObjectTypeQualifiers()
	for q := range qualifiers.Iterator() {
		switch q.Kind() {
		case common.CLIENT_KEYWORD:
			objectType.NetworkQuals = ObjectNetworkQualsClient
		case common.SERVICE_KEYWORD:
			objectType.NetworkQuals = ObjectNetworkQualsService
		case common.ISOLATED_KEYWORD:
			objectType.Isolated = true
		case common.READONLY_KEYWORD:
			// https://github.com/ballerina-platform/ballerina-lang-go/issues/537",
			n.cx.Unimplemented("readonly object type descriptors are not implemented", getPosition(n.de(), q))
		}
	}

	// Process members
	members := objectTypeDescriptorNode.Members()
	for i := 0; i < members.Size(); i++ {
		member := members.Get(i)
		switch member.Kind() {
		case common.OBJECT_FIELD:
			objectField := member.(*tree.ObjectFieldNode)
			fieldName, _ := normalizedIdentifierValue(objectField.FieldName().Text())
			bField := &BObjectField{
				Ty: n.createTypeNode(objectField.TypeName()).(BType),
			}
			bField.name = fieldName
			bField.pos = getPosition(n.de(), objectField)
			if vis := objectField.VisibilityQualifier(); vis != nil && vis.Kind() == common.PUBLIC_KEYWORD {
				bField.flags |= model.FlagPublic
			}
			n.populateMetadata(objectField.Metadata(), bField)
			if objectType.AddMember(bField) {
				n.cx.SyntaxError("redeclared symbol '"+fieldName+"'", bField.pos)
			}
		case common.METHOD_DECLARATION:
			methodDecl := member.(*tree.MethodDeclarationNode)
			methodName, _ := normalizedIdentifierValue(methodDecl.MethodName().Text())
			bMethod := &BMethodDecl{}
			bMethod.name = methodName
			bMethod.pos = getPosition(n.de(), methodDecl)
			bMethod.memberKind = ObjectMemberKindMethod

			// Process visibility and method kind from qualifier list
			methodQuals := methodDecl.QualifierList()
			for q := range methodQuals.Iterator() {
				switch q.Kind() {
				case common.PUBLIC_KEYWORD:
					bMethod.flags |= model.FlagPublic
				case common.REMOTE_KEYWORD:
					bMethod.memberKind = ObjectMemberKindRemoteMethod
				case common.RESOURCE_KEYWORD:
					bMethod.memberKind = ObjectMemberKindResourceMethod
				case common.ISOLATED_KEYWORD:
					bMethod.SetIsolated()
				case common.TRANSACTIONAL_KEYWORD:
					bMethod.SetTransactional()
				}
			}

			if bMethod.memberKind == ObjectMemberKindRemoteMethod {
				bMethod.name = model.RemoteMethodName(bMethod.name)
			}

			// Build function type from method signature
			funcSig := methodDecl.MethodSignature()
			if funcSig != nil {
				// Process parameters
				params := funcSig.Parameters()
				for param := range params.Iterator() {
					ftParam := n.createFunctionTypeParam(param)
					if _, isRest := param.(*tree.RestParameterNode); isRest {
						bMethod.RestParam = &ftParam
					} else {
						bMethod.RequiredParams = append(bMethod.RequiredParams, ftParam)
					}
				}

				// Process return type
				if retTypeDesc := funcSig.ReturnTypeDesc(); retTypeDesc != nil {
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
		case common.TYPE_REFERENCE:
			typeRef := member.(*tree.TypeReferenceNode)
			objectType.unresolvedInclusions = append(objectType.unresolvedInclusions, n.createTypeNode(typeRef.TypeName()).(*BLangUserDefinedType))
		default:
			panic("unexpected member kind in object type descriptor")
		}
	}

	objectType.pos = getPosition(n.de(), objectTypeDescriptorNode)
	return objectType
}

func (n *NodeBuilder) TransformObjectConstructorExpression(objectConstructorBLangExpression *tree.ObjectConstructorExpressionNode) BLangNode {
	panic("TransformObjectConstructorExpression unimplemented")
}

func (n *NodeBuilder) TransformRecordTypeDescriptor(recordTypeDescriptorNode *tree.RecordTypeDescriptorNode) BLangNode {
	recordType := &BLangRecordType{}
	fields := recordTypeDescriptorNode.Fields()
	for i := 0; i < fields.Size(); i++ {
		field := fields.Get(i)
		switch field.Kind() {
		case common.RECORD_FIELD:
			recordField := field.(*tree.RecordFieldNode)
			fieldName := recordField.FieldName().Text()
			bField := BField{
				Name: model.Name(fieldName),
				Type: n.createTypeNode(recordField.TypeName()).(BType),
			}
			bField.pos = getPosition(n.de(), recordField)
			if recordField.ReadonlyKeyword() != nil {
				bField.SetReadonly()
			}
			if recordField.QuestionMarkToken() != nil {
				bField.SetOptional()
			}
			n.populateMetadata(recordField.Metadata(), &bField)
			recordType.AddField(fieldName, bField)
		case common.RECORD_FIELD_WITH_DEFAULT_VALUE:
			recordFieldDV := field.(*tree.RecordFieldWithDefaultValueNode)
			fieldName := recordFieldDV.FieldName().Text()
			bField := BField{
				Name:        model.Name(fieldName),
				Type:        n.createTypeNode(recordFieldDV.TypeName()).(BType),
				DefaultExpr: n.createExpression(recordFieldDV.Expression()),
			}
			bField.pos = getPosition(n.de(), recordFieldDV)
			if recordFieldDV.ReadonlyKeyword() != nil {
				bField.SetReadonly()
			}
			n.populateMetadata(recordFieldDV.Metadata(), &bField)
			recordType.AddField(fieldName, bField)
		case common.TYPE_REFERENCE:
			typeRef := field.(*tree.TypeReferenceNode)
			recordType.TypeInclusions = append(recordType.TypeInclusions, n.createTypeNode(typeRef.TypeName()).(BType))
		default:
			panic("unexpected field kind in record type descriptor")
		}
	}
	if restDesc := recordTypeDescriptorNode.RecordRestDescriptor(); restDesc != nil {
		recordType.RestType = n.createTypeNode(restDesc.TypeName()).(BType)
	}
	recordType.IsOpen = recordTypeDescriptorNode.BodyStartDelimiter().Kind() == common.OPEN_BRACE_TOKEN
	recordType.pos = getPosition(n.de(), recordTypeDescriptorNode)
	return recordType
}

func (n *NodeBuilder) TransformReturnTypeDescriptor(returnTypeDescriptorNode *tree.ReturnTypeDescriptorNode) BLangNode {
	panic("TransformReturnTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformNilTypeDescriptor(nilTypeDescriptorNode *tree.NilTypeDescriptorNode) BLangNode {
	panic("TransformNilTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformOptionalTypeDescriptor(optionalTypeDescriptorNode *tree.OptionalTypeDescriptorNode) BLangNode {
	typeDesc := optionalTypeDescriptorNode.TypeDescriptor()
	nilType := &BLangValueType{TypeKind: TypeKind_NIL}
	nilType.pos = getPosition(n.de(), optionalTypeDescriptorNode.QuestionMarkToken())
	bLUnionType := &BLangUnionTypeNode{
		lhs: TypeData{
			TypeDescriptor: n.createTypeNode(typeDesc),
		},
		rhs: TypeData{
			TypeDescriptor: nilType,
		},
	}
	bLUnionType.pos = getPosition(n.de(), optionalTypeDescriptorNode)
	return bLUnionType
}

func (n *NodeBuilder) TransformObjectField(objectFieldNode *tree.ObjectFieldNode) BLangNode {
	panic("TransformObjectField unimplemented")
}

func (n *NodeBuilder) TransformRecordField(recordFieldNode *tree.RecordFieldNode) BLangNode {
	panic("TransformRecordField unimplemented")
}

func (n *NodeBuilder) TransformRecordFieldWithDefaultValue(recordFieldWithDefaultValueNode *tree.RecordFieldWithDefaultValueNode) BLangNode {
	panic("TransformRecordFieldWithDefaultValue unimplemented")
}

func (n *NodeBuilder) TransformRecordRestDescriptor(recordRestDescriptorNode *tree.RecordRestDescriptorNode) BLangNode {
	panic("TransformRecordRestDescriptor unimplemented")
}

func (n *NodeBuilder) TransformTypeReference(typeReferenceNode *tree.TypeReferenceNode) BLangNode {
	panic("TransformTypeReference unimplemented")
}

func (n *NodeBuilder) TransformAnnotation(annotationNode *tree.AnnotationNode) BLangNode {
	annotation := &BLangAnnotationAttachment{}
	annotation.SetPosition(getPosition(n.de(), annotationNode))
	nameReference := n.createBLangNameReference(annotationNode.AnnotReference())
	annotation.PkgAlias = &nameReference[0]
	annotation.AnnotationName = &nameReference[1]
	if value := annotationNode.AnnotValue(); value != nil && !value.IsMissing() {
		annotation.Expr = n.createExpression(value)
		annotation.HasValue = true
	} else {
		annotation.Expr = n.createTrueLiteral(annotation.GetPosition())
	}
	return annotation
}

func (n *NodeBuilder) TransformMetadata(metadataNode *tree.MetadataNode) BLangNode {
	docString := getDocumentationString(metadataNode)
	if docString == nil || docString.IsMissing() {
		return nil
	}
	return n.createMarkdownDocumentationAttachment(docString)
}

func (n *NodeBuilder) TransformModuleVariableDeclaration(moduleVariableDeclarationNode *tree.ModuleVariableDeclarationNode) BLangNode {
	typedBindingPattern := moduleVariableDeclarationNode.TypedBindingPattern()
	bindingPattern := typedBindingPattern.BindingPattern()
	pos := getPositionWithoutMetadata(n.de(), moduleVariableDeclarationNode)

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

func (n *NodeBuilder) populateModuleVariableVisibilityAndQualifiers(node *tree.ModuleVariableDeclarationNode, simpleVar *BLangSimpleVariable) {
	visibilityQualifier := node.VisibilityQualifier()
	if visibilityQualifier != nil && visibilityQualifier.Kind() == common.PUBLIC_KEYWORD {
		simpleVar.SetPublic()
	}

	qualifiers := node.Qualifiers()
	for i := 0; i < qualifiers.Size(); i++ {
		qualifier := qualifiers.Get(i)
		switch qualifier.Kind() {
		case common.FINAL_KEYWORD:
			simpleVar.SetFinal()
		case common.ISOLATED_KEYWORD:
			simpleVar.SetIsolated()
		case common.CONFIGURABLE_KEYWORD:
			n.cx.Unimplemented("configurable module variables are not supported yet", simpleVar.pos)
		}
	}
}

func (n *NodeBuilder) TransformTypeTestExpression(typeTestBLangExpression *tree.TypeTestExpressionNode) BLangNode {
	typeTestExpr := &BLangTypeTestExpr{}
	typeTestExpr.isNegation = typeTestBLangExpression.IsKeyword().Kind() == common.NOT_IS_KEYWORD
	typeTestExpr.Expr = n.createExpression(typeTestBLangExpression.Expression())
	typeTestExpr.Type = TypeData{TypeDescriptor: n.createTypeNode(typeTestBLangExpression.TypeDescriptor())}
	typeTestExpr.SetPosition(getPosition(n.de(), typeTestBLangExpression))
	return typeTestExpr
}

func (n *NodeBuilder) TransformRemoteMethodCallAction(remoteMethodCallActionNode *tree.RemoteMethodCallActionNode) BLangNode {
	inv := n.createBLangInvocation(remoteMethodCallActionNode.MethodName(),
		remoteMethodCallActionNode.Arguments(),
		getPosition(n.de(), remoteMethodCallActionNode), false)
	action := &BLangRemoteMethodCallAction{}
	action.bLangInvocationBase = inv.bLangInvocationBase
	action.Expr = n.createExpression(remoteMethodCallActionNode.Expression())
	action.pos = getPosition(n.de(), remoteMethodCallActionNode)
	return action
}

func (n *NodeBuilder) TransformMapTypeDescriptor(mapTypeDescriptorNode *tree.MapTypeDescriptorNode) BLangNode {
	refType := &BLangBuiltInRefTypeNode{
		TypeKind: TypeKind_MAP,
	}
	refType.SetPosition(getPosition(n.de(), mapTypeDescriptorNode))

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

func (n *NodeBuilder) TransformNilLiteral(nilLiteralNode *tree.NilLiteralNode) BLangNode {
	panic("TransformNilLiteral unimplemented")
}

func (n *NodeBuilder) TransformAnnotationDeclaration(annotationDeclarationNode *tree.AnnotationDeclarationNode) BLangNode {
	annotation := &BLangAnnotation{}
	annotation.SetPosition(getPositionWithoutMetadata(n.de(), annotationDeclarationNode))
	name := createIdentifierFromToken(getPosition(n.de(), annotationDeclarationNode.AnnotationTag()), annotationDeclarationNode.AnnotationTag())
	annotation.Name = &name
	if visibility := annotationDeclarationNode.VisibilityQualifier(); visibility != nil && visibility.Kind() == common.PUBLIC_KEYWORD {
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
		if attachPoint, ok := attachPoint.(*tree.AnnotationAttachPointNode); ok {
			annotation.AddAttachPoint(n.createAnnotationAttachPoint(attachPoint))
		}
	}
	n.populateMetadata(annotationDeclarationNode.Metadata(), annotation)
	return annotation
}

func (n *NodeBuilder) TransformAnnotationAttachPoint(annotationAttachPointNode *tree.AnnotationAttachPointNode) BLangNode {
	n.createAnnotationAttachPoint(annotationAttachPointNode)
	return nil
}

func (n *NodeBuilder) createAnnotationAttachPoint(annotationAttachPointNode *tree.AnnotationAttachPointNode) AttachPoint {
	parts := []string{}
	identifiers := annotationAttachPointNode.Identifiers()
	for i := 0; i < identifiers.Size(); i++ {
		parts = append(parts, identifiers.Get(i).Text())
	}
	point, ok := annotationAttachPointFromParts(parts)
	if !ok {
		n.cx.SyntaxError("unknown annotation attach point '"+strings.Join(parts, " ")+"'", getPosition(n.de(), annotationAttachPointNode))
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
	tree.Node
	Namespaceuri() tree.ExpressionNode
	NamespacePrefix() *tree.IdentifierToken
}

func (n *NodeBuilder) transformXMLNamespaceDeclaration(node xmlNamespaceDeclarationNode) BLangNode {
	pos := getPosition(n.de(), node)
	xmlns := &BLangXMLNS{}
	xmlns.SetPosition(pos)
	n.populateXMLNS(xmlns, pos, node.Namespaceuri(), node.NamespacePrefix())
	return xmlns
}

func (n *NodeBuilder) TransformXMLNamespaceDeclaration(xMLNamespaceDeclarationNode *tree.XMLNamespaceDeclarationNode) BLangNode {
	return n.transformXMLNamespaceDeclaration(xMLNamespaceDeclarationNode)
}

func (n *NodeBuilder) TransformModuleXMLNamespaceDeclaration(moduleXMLNamespaceDeclarationNode *tree.ModuleXMLNamespaceDeclarationNode) BLangNode {
	return n.transformXMLNamespaceDeclaration(moduleXMLNamespaceDeclarationNode)
}

func (n *NodeBuilder) populateXMLNS(target *BLangXMLNS, pos diagnostics.Location, uriNode tree.ExpressionNode, prefixTok *tree.IdentifierToken) {
	if uriNode != nil {
		target.SetNamespaceURI(n.createExpression(uriNode))
	}
	if prefixTok != nil {
		prefixIdent := createIdentifierFromToken(getPosition(n.de(), prefixTok), prefixTok)
		target.SetPrefix(&prefixIdent)
	}
}

func (n *NodeBuilder) TransformFunctionBodyBlock(functionBodyBlockNode *tree.FunctionBodyBlockNode) BLangNode {
	bLFuncBody := &BLangBlockFunctionBody{}
	n.isInLocalContext = true
	statements := []StatementNode{}
	stmtList := statements
	namedWorkerDeclarator := functionBodyBlockNode.NamedWorkerDeclarator()
	if namedWorkerDeclarator != nil {
		panic("unimplemented")
	}

	n.generateAndAddBLangStatements(functionBodyBlockNode.Statements(), &stmtList, 0, functionBodyBlockNode)

	bLFuncBody.Stmts = stmtList
	bLFuncBody.pos = getPosition(n.de(), functionBodyBlockNode)
	n.isInLocalContext = false
	return bLFuncBody
}

func (n *NodeBuilder) generateForkStatements(statements *[]StatementNode, forkStatementNode *tree.ForkStatementNode) {
	panic("generateForkStatements unimplemented")
}

func (n *NodeBuilder) TransformNamedWorkerDeclaration(namedWorkerDeclarationNode *tree.NamedWorkerDeclarationNode) BLangNode {
	panic("TransformNamedWorkerDeclaration unimplemented")
}

func (n *NodeBuilder) TransformNamedWorkerDeclarator(namedWorkerDeclarator *tree.NamedWorkerDeclarator) BLangNode {
	panic("TransformNamedWorkerDeclarator unimplemented")
}

func (n *NodeBuilder) TransformBasicLiteral(basicLiteralNode *tree.BasicLiteralNode) BLangNode {
	panic("TransformBasicLiteral unimplemented")
}

func (n *NodeBuilder) TransformSimpleNameReference(simpleNameReferenceNode *tree.SimpleNameReferenceNode) BLangNode {
	panic("TransformSimpleNameReference unimplemented")
}

func (n *NodeBuilder) TransformQualifiedNameReference(qualifiedNameReferenceNode *tree.QualifiedNameReferenceNode) BLangNode {
	panic("TransformQualifiedNameReference unimplemented")
}

func (n *NodeBuilder) TransformBuiltinSimpleNameReference(builtinSimpleNameReferenceNode *tree.BuiltinSimpleNameReferenceNode) BLangNode {
	panic("TransformBuiltinSimpleNameReference unimplemented")
}

func (n *NodeBuilder) TransformTrapExpression(trapBLangExpression *tree.TrapExpressionNode) BLangNode {
	pos := getPosition(n.de(), trapBLangExpression)
	expr := n.createExpression(trapBLangExpression.Expression())
	trapExpr := &BLangTrapExpr{}
	trapExpr.pos = pos
	trapExpr.Expr = expr
	return trapExpr
}

func (n *NodeBuilder) TransformListConstructorExpression(listConstructorBLangExpression *tree.ListConstructorExpressionNode) BLangNode {
	argExprList := make([]BLangExpression, 0)
	spreadMemberIndexes := make([]int, 0)
	listConstructorExpr := &BLangListConstructorExpr{}

	expressions := listConstructorBLangExpression.Expressions()
	for i := 0; i < expressions.Size(); i += 2 {
		listMember := expressions.Get(i)
		var memberExpr BLangExpression
		if listMember.Kind() == common.SPREAD_MEMBER {
			spreadMember := listMember.(*tree.SpreadMemberNode)
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
	listConstructorExpr.pos = getPosition(n.de(), listConstructorBLangExpression)
	return listConstructorExpr
}

func (n *NodeBuilder) TransformTypeCastExpression(typeCastBLangExpression *tree.TypeCastExpressionNode) BLangNode {
	typeConversionNode := &BLangTypeConversionExpr{}
	typeConversionNode.SetPosition(getPosition(n.de(), typeCastBLangExpression))
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

func (n *NodeBuilder) TransformTypeCastParam(typeCastParamNode *tree.TypeCastParamNode) BLangNode {
	panic("TransformTypeCastParam unimplemented")
}

func (n *NodeBuilder) TransformUnionTypeDescriptor(unionTypeDescriptorNode *tree.UnionTypeDescriptorNode) BLangNode {
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
	bLUnionType.pos = getPosition(n.de(), unionTypeDescriptorNode)
	return bLUnionType
}

func (n *NodeBuilder) TransformTableConstructorExpression(tableConstructorBLangExpression *tree.TableConstructorExpressionNode) BLangNode {
	panic("TransformTableConstructorExpression unimplemented")
}

func (n *NodeBuilder) TransformKeySpecifier(keySpecifierNode *tree.KeySpecifierNode) BLangNode {
	panic("TransformKeySpecifier unimplemented")
}

func (n *NodeBuilder) TransformStreamTypeDescriptor(streamTypeDescriptorNode *tree.StreamTypeDescriptorNode) BLangNode {
	position := getPosition(n.de(), streamTypeDescriptorNode)
	paramsNode := streamTypeDescriptorNode.StreamTypeParamsNode()
	if paramsNode == nil {
		refType := &BLangBuiltInRefTypeNode{
			TypeKind: TypeKind_STREAM,
		}
		refType.SetPosition(position)
		return refType
	}
	params, ok := paramsNode.(*tree.StreamTypeParamsNode)
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

func (n *NodeBuilder) TransformStreamTypeParams(streamTypeParamsNode *tree.StreamTypeParamsNode) BLangNode {
	panic("TransformStreamTypeParams unimplemented")
}

func (n *NodeBuilder) TransformLetExpression(letBLangExpression *tree.LetExpressionNode) BLangNode {
	panic("TransformLetExpression unimplemented")
}

func (n *NodeBuilder) TransformLetVariableDeclaration(letVariableDeclarationNode *tree.LetVariableDeclarationNode) BLangNode {
	varDef := n.createBLangVarDef(
		getPosition(n.de(), letVariableDeclarationNode),
		letVariableDeclarationNode.TypedBindingPattern(),
		letVariableDeclarationNode.Expression(),
		nil,
	)
	annotations := letVariableDeclarationNode.Annotations()
	if annotations.Size() > 0 {
		panic("annotations not yet supported")
	}
	return varDef.(BLangNode)
}

func (n *NodeBuilder) TransformTemplateExpression(templateBLangExpression *tree.TemplateExpressionNode) BLangNode {
	typeToken := templateBLangExpression.Type()
	pos := getPosition(n.de(), templateBLangExpression)
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

func (n *NodeBuilder) buildXMLTemplateExpr(templateBLangExpression *tree.TemplateExpressionNode, pos diagnostics.Location) BLangNode {
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

func (n *NodeBuilder) buildXMLSequenceLiteral(templateBLangExpression *tree.TemplateExpressionNode, pos diagnostics.Location) BLangNode {
	var children []BLangExpression
	content := templateBLangExpression.Content()
	for child := range content.Iterator() {
		bl := n.TransformSyntaxNode(child)
		if bl == nil {
			n.cx.InternalError("xml template child did not produce BLangNode", getPosition(n.de(), child))
			return nil
		}
		expr, ok := bl.(BLangExpression)
		if !ok {
			n.cx.InternalError("xml template child did not produce BLangExpression", getPosition(n.de(), child))
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

func xmlTemplateHasInterpolation(content tree.NodeList[tree.Node]) bool {
	for child := range content.Iterator() {
		if xmlNodeHasInterpolation(child) {
			return true
		}
	}
	return false
}

func xmlNodeHasInterpolation(node tree.Node) bool {
	return firstXMLInterpolation(node) != nil
}

func firstXMLInterpolation(node tree.Node) *tree.InterpolationNode {
	switch x := node.(type) {
	case *tree.InterpolationNode:
		return x
	case *tree.XMLElementNode:
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
	case *tree.XMLEmptyElementNode:
		attrs := x.Attributes()
		for attr := range attrs.Iterator() {
			if value := attr.Value(); value != nil {
				if ins := firstXMLInterpolation(value); ins != nil {
					return ins
				}
			}
		}
	case *tree.XMLAttributeValue:
		value := x.Value()
		for child := range value.Iterator() {
			if ins := firstXMLInterpolation(child); ins != nil {
				return ins
			}
		}
	case *tree.XMLComment:
		content := x.Content()
		for child := range content.Iterator() {
			if ins, ok := child.(*tree.InterpolationNode); ok {
				return ins
			}
		}
	case *tree.XMLProcessingInstruction:
		data := x.Data()
		for child := range data.Iterator() {
			if ins, ok := child.(*tree.InterpolationNode); ok {
				return ins
			}
		}
	case *tree.XMLCDATANode:
		content := x.Content()
		for child := range content.Iterator() {
			if ins, ok := child.(*tree.InterpolationNode); ok {
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

func (n *NodeBuilder) flattenXMLTemplateContent(content tree.NodeList[tree.Node], kind XMLTemplateInsertionKind) iter.Seq2[xmlTemplateToken, *xmlTemplateDiagnostic] {
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
	node tree.Node,
	kind XMLTemplateInsertionKind,
	namespaceInsertion *XMLTemplateNamespaceInsertion,
	yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool,
) bool {
	switch x := node.(type) {
	case tree.Token:
		return yield(newXMLTemplateTextToken(x.Text()), nil)
	case *tree.InterpolationNode:
		expr := n.createActionOrExpression(x.Expression())
		be, ok := expr.(BLangExpression)
		if !ok {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation did not produce BLangExpression",
				Position: getPosition(n.de(), x),
				Internal: true,
			})
		}
		return yield(newXMLTemplateInsertionToken(be, kind), nil)
	case *tree.XMLTextNode:
		if c := x.Content(); c != nil {
			return yield(newXMLTemplateTextToken(c.Text()), nil)
		}
		return true
	case *tree.XMLElementNode:
		return n.flattenXMLTemplateElement(x, namespaceInsertion, yield)
	case *tree.XMLEmptyElementNode:
		return n.flattenXMLTemplateEmptyElement(x, namespaceInsertion, yield)
	case *tree.XMLComment:
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml comment",
				Position: getPosition(n.de(), ins),
			})
		}
		return yield(newXMLTemplateTextToken(tree.ToSourceCode(x.InternalNode())), nil)
	case *tree.XMLProcessingInstruction:
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml processing instruction",
				Position: getPosition(n.de(), ins),
			})
		}
		return n.flattenXMLTemplatePI(x, yield)
	case *tree.XMLCDATANode:
		if ins := firstXMLInterpolation(x); ins != nil {
			return yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
				Message:  "interpolation is not allowed in xml CDATA section",
				Position: getPosition(n.de(), ins),
			})
		}
		return yield(newXMLTemplateTextToken(tree.ToSourceCode(x.InternalNode())), nil)
	default:
		return yield(newXMLTemplateTextToken(tree.ToSourceCode(node.InternalNode())), nil)
	}
}

func (n *NodeBuilder) flattenXMLTemplateElement(
	x *tree.XMLElementNode,
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
	x *tree.XMLEmptyElementNode,
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

func (n *NodeBuilder) flattenXMLTemplatePI(x *tree.XMLProcessingInstruction, yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) bool {
	if !yield(newXMLTemplateTextToken("<?"), nil) {
		return false
	}
	if !yield(newXMLTemplateTextToken(n.xmlNameToString(x.Target())), nil) {
		return false
	}
	var dataText strings.Builder
	data := x.Data()
	for child := range data.Iterator() {
		if tok, ok := child.(tree.Token); ok {
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

func (n *NodeBuilder) flattenXMLTemplateAttributes(attrs tree.NodeList[*tree.XMLAttributeNode], yield func(xmlTemplateToken, *xmlTemplateDiagnostic) bool) bool {
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
	value *tree.XMLAttributeValue,
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
		if ins, ok := child.(*tree.InterpolationNode); ok {
			if isXMLNS {
				if !yield(xmlTemplateToken{}, &xmlTemplateDiagnostic{
					Message:  "interpolation is not allowed in xml xmlns attribute value",
					Position: getPosition(n.de(), child),
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
		if tok, ok := child.(tree.Token); ok {
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

func (n *NodeBuilder) collectXMLTemplateNamespaceInsertion(node tree.Node) XMLTemplateNamespaceInsertion {
	insn := XMLTemplateNamespaceInsertion{
		UsedPrefixes: map[string]struct{}{},
	}
	n.collectXMLTemplateNamespaceRefs(node, nil, &insn)
	return insn
}

func (n *NodeBuilder) collectXMLTemplateNamespaceRefs(node tree.Node, scopes []map[string]struct{}, insn *XMLTemplateNamespaceInsertion) {
	switch x := node.(type) {
	case *tree.XMLElementNode:
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
	case *tree.XMLEmptyElementNode:
		childScopes := appendXMLTemplateNamespaceScope(scopes, n.collectInlineXMLTemplatePrefixes(x.Attributes()))
		n.recordXMLTemplateNameRef(n.xmlNameToString(x.Name()), true, childScopes, insn)
		n.collectXMLTemplateAttributeNamespaceRefs(x.Attributes(), childScopes, insn)
	}
}

func (n *NodeBuilder) collectXMLTemplateAttributeNamespaceRefs(
	attrs tree.NodeList[*tree.XMLAttributeNode],
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

func (n *NodeBuilder) collectInlineXMLTemplatePrefixes(attrs tree.NodeList[*tree.XMLAttributeNode]) map[string]struct{} {
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

func (n *NodeBuilder) buildStringTemplateExpr(node *tree.TemplateExpressionNode, pos diagnostics.Location) BLangNode {
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
		case tree.Token:
			if c.Kind() != common.TEMPLATE_STRING {
				n.cx.InternalError(fmt.Sprintf("unexpected token kind in string template: %v", c.Kind()), getPosition(n.de(), c))
				continue
			}
			strs = append(strs, c.Text())
			lastStr = true
		case *tree.InterpolationNode:
			if !lastStr {
				strs = append(strs, "")
			}
			expr := n.createActionOrExpression(c.Expression())
			be, ok := expr.(BLangExpression)
			if !ok {
				n.cx.InternalError("interpolation did not produce BLangExpression", getPosition(n.de(), c))
				return nil
			}
			insertions = append(insertions, be)
			lastStr = false
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected node in string template: %T", c), getPosition(n.de(), child))
		}
	}
	if !lastStr {
		strs = append(strs, "")
	}
	tpl := &BLangTemplateExpr{Kind: TemplateExprKindString, Strings: strs, Insertions: insertions}
	tpl.SetPosition(pos)
	return tpl
}

func (n *NodeBuilder) xmlNameToString(name tree.XMLNameNode) string {
	pos := getPosition(n.de(), name)
	switch name := name.(type) {
	case *tree.XMLSimpleNameNode:
		tok := name.Name()
		if tok == nil {
			n.cx.InternalError("xml simple name missing identifier token", pos)
			return ""
		}
		return tok.Text()
	case *tree.XMLQualifiedNameNode:
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

func (n *NodeBuilder) xmlAttributes(attrs tree.NodeList[*tree.XMLAttributeNode]) []BLangXMLAttribute {
	out := make([]BLangXMLAttribute, 0, attrs.Size())
	for attrNode := range attrs.Iterator() {
		attr := n.TransformXMLAttribute(attrNode).(*BLangXMLAttribute)
		out = append(out, *attr)
	}
	return out
}

func (n *NodeBuilder) TransformXMLElement(xMLElementNode *tree.XMLElementNode) BLangNode {
	elem := &BLangXMLElementLiteral{}
	elem.pos = getPosition(n.de(), xMLElementNode)
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
			n.cx.InternalError("xml element child did not produce BLangExpression", getPosition(n.de(), child))
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

func (n *NodeBuilder) TransformXMLStartTag(xMLStartTagNode *tree.XMLStartTagNode) BLangNode {
	panic("TransformXMLStartTag unimplemented")
}

func (n *NodeBuilder) TransformXMLEndTag(xMLEndTagNode *tree.XMLEndTagNode) BLangNode {
	panic("TransformXMLEndTag unimplemented")
}

func (n *NodeBuilder) TransformXMLSimpleName(xMLSimpleNameNode *tree.XMLSimpleNameNode) BLangNode {
	panic("TransformXMLSimpleName unimplemented")
}

func (n *NodeBuilder) TransformXMLQualifiedName(xMLQualifiedNameNode *tree.XMLQualifiedNameNode) BLangNode {
	panic("TransformXMLQualifiedName unimplemented")
}

func (n *NodeBuilder) TransformXMLEmptyElement(xMLEmptyElementNode *tree.XMLEmptyElementNode) BLangNode {
	elem := &BLangXMLElementLiteral{}
	elem.pos = getPosition(n.de(), xMLEmptyElementNode)
	elem.Name = n.xmlNameToString(xMLEmptyElementNode.Name())
	elem.Attrs = n.xmlAttributes(xMLEmptyElementNode.Attributes())
	return elem
}

func (n *NodeBuilder) TransformInterpolation(interpolationNode *tree.InterpolationNode) BLangNode {
	n.cx.Unimplemented("xml interpolation not yet supported", getPosition(n.de(), interpolationNode))
	return nil
}

func (n *NodeBuilder) TransformXMLText(xMLTextNode *tree.XMLTextNode) BLangNode {
	text := &BLangXMLTextLiteral{}
	text.pos = getPosition(n.de(), xMLTextNode)
	if c := xMLTextNode.Content(); c != nil {
		text.Body = c.Text()
	}
	return text
}

func (n *NodeBuilder) TransformXMLAttribute(xMLAttributeNode *tree.XMLAttributeNode) BLangNode {
	attr := &BLangXMLAttribute{}
	attr.pos = getPosition(n.de(), xMLAttributeNode)
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

func (n *NodeBuilder) TransformXMLAttributeValue(xMLAttributeValue *tree.XMLAttributeValue) BLangNode {
	var b strings.Builder
	items := xMLAttributeValue.Value()
	for child := range items.Iterator() {
		tok, ok := child.(tree.Token)
		if !ok {
			n.cx.Unimplemented("xml attribute value interpolation not yet supported", getPosition(n.de(), child))
			return nil
		}
		b.WriteString(tok.Text())
	}
	text := b.String()
	lit := &BLangLiteral{}
	lit.pos = getPosition(n.de(), xMLAttributeValue)
	lit.SetValueType(n.types.getTypeFromTag(TypeTags_STRING).(BType))
	lit.Value = text
	lit.OriginalValue = text
	return lit
}

func (n *NodeBuilder) TransformXMLComment(xMLComment *tree.XMLComment) BLangNode {
	c := &BLangXMLCommentLiteral{}
	c.pos = getPosition(n.de(), xMLComment)
	var b strings.Builder
	content := xMLComment.Content()
	for child := range content.Iterator() {
		tok, ok := child.(tree.Token)
		if !ok {
			n.cx.Unimplemented("xml interpolation in comment not yet supported", getPosition(n.de(), child))
			continue
		}
		b.WriteString(tok.Text())
	}
	c.Body = b.String()
	return c
}

func (n *NodeBuilder) TransformXMLCDATA(xMLCDATANode *tree.XMLCDATANode) BLangNode {
	n.cx.Unimplemented("xml CDATA not yet supported", getPosition(n.de(), xMLCDATANode))
	return nil
}

func (n *NodeBuilder) TransformXMLProcessingInstruction(xMLProcessingInstruction *tree.XMLProcessingInstruction) BLangNode {
	pi := &BLangXMLPILiteral{}
	pi.pos = getPosition(n.de(), xMLProcessingInstruction)
	pi.Target = n.xmlNameToString(xMLProcessingInstruction.Target())
	var b strings.Builder
	data := xMLProcessingInstruction.Data()
	for child := range data.Iterator() {
		tok, ok := child.(tree.Token)
		if !ok {
			n.cx.Unimplemented("xml interpolation in processing instruction not yet supported", getPosition(n.de(), child))
			continue
		}
		b.WriteString(tok.Text())
	}
	pi.Data = b.String()
	return pi
}

func (n *NodeBuilder) TransformTableTypeDescriptor(tableTypeDescriptorNode *tree.TableTypeDescriptorNode) BLangNode {
	panic("TransformTableTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformTypeParameter(typeParameterNode *tree.TypeParameterNode) BLangNode {
	return n.createTypeNode(typeParameterNode.TypeNode()).(BLangNode)
}

func (n *NodeBuilder) TransformKeyTypeConstraint(keyTypeConstraintNode *tree.KeyTypeConstraintNode) BLangNode {
	panic("TransformKeyTypeConstraint unimplemented")
}

func (n *NodeBuilder) TransformFunctionTypeDescriptor(functionTypeDescriptorNode *tree.FunctionTypeDescriptorNode) BLangNode {
	funcType := &BLangFunctionType{}
	funcType.pos = getPosition(n.de(), functionTypeDescriptorNode)

	if funcSignature := functionTypeDescriptorNode.FunctionSignature(); funcSignature != nil {
		// Set Parameters
		parameters := funcSignature.Parameters()
		for param := range parameters.Iterator() {
			ftParam := n.createFunctionTypeParam(param)
			if _, isRestParam := param.(*tree.RestParameterNode); isRestParam {
				funcType.RestParam = &ftParam
			} else {
				funcType.RequiredParams = append(funcType.RequiredParams, ftParam)
			}
		}

		// Set Return Type
		if retNode := funcSignature.ReturnTypeDesc(); retNode != nil {
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
		case common.ISOLATED_KEYWORD:
			funcType.SetIsolated()
		case common.TRANSACTIONAL_KEYWORD:
			funcType.SetTransactional()
		}
	}

	return funcType
}

type typedParameterNode interface {
	tree.ParameterNode
	ParamName() tree.Token
	TypeName() tree.Node
	Annotations() tree.NodeList[*tree.AnnotationNode]
}

func (n *NodeBuilder) createFunctionTypeParam(param tree.ParameterNode) BLangFunctionTypeParam {
	typedParam, ok := param.(typedParameterNode)
	if !ok {
		panic("createFunctionTypeParam: unsupported parameter type")
	}
	paramName := typedParam.ParamName()
	typeName := typedParam.TypeName()
	annotations := typedParam.Annotations()

	ftParam := BLangFunctionTypeParam{}
	ftParam.pos = getPosition(n.de(), param)

	if paramName != nil {
		name := createIdentifierFromToken(getPosition(n.de(), paramName), paramName)
		name.pos = getPosition(n.de(), paramName)
		ftParam.Name = &name
	}

	ftParam.TypeDesc = n.createTypeNode(typeName).(BType)

	if dp, ok := param.(*tree.DefaultableParameterNode); ok {
		defaultExpr := dp.Expression()
		ftParam.InitExpr = n.createExpression(defaultExpr)
	}

	if annotations.Size() > 0 {
		panic("function type param annotations not yet supported")
	}

	return ftParam
}

func (n *NodeBuilder) TransformFunctionSignature(functionSignatureNode *tree.FunctionSignatureNode) BLangNode {
	panic("TransformFunctionSignature unimplemented")
}

func (n *NodeBuilder) TransformExplicitAnonymousFunctionExpression(anonFuncExprNode *tree.ExplicitAnonymousFunctionExpressionNode) BLangNode {
	bLFunction := &BLangFunction{}
	name := n.cx.GetNextAnonymousFunctionKey(n.PackageID)
	ident := createIdentifier(diagnostics.NewBuiltinLocation(), &name, &name)
	bLFunction.Name = ident
	n.populateFuncSignature(bLFunction, anonFuncExprNode.FunctionSignature())
	body := n.TransformSyntaxNode(anonFuncExprNode.FunctionBody()).(FunctionBodyNode)
	bLFunction.Body = body
	bLFunction.pos = getPosition(n.de(), anonFuncExprNode)
	bLFunction.SetAnonymous()
	setFunctionQualifiers(bLFunction, anonFuncExprNode.QualifierList())

	lambdaFunc := &BLangLambdaFunction{Function: bLFunction}
	lambdaFunc.pos = bLFunction.pos
	return lambdaFunc
}

func (n *NodeBuilder) TransformExpressionFunctionBody(expressionFunctionBodyNode *tree.ExpressionFunctionBodyNode) BLangNode {
	exprBody := &BLangExprFunctionBody{}
	exprBody.Expr = n.createExpression(expressionFunctionBodyNode.Expression())
	exprBody.pos = getPosition(n.de(), expressionFunctionBodyNode)
	return exprBody
}

func (n *NodeBuilder) TransformTupleTypeDescriptor(tupleTypeDescriptorNode *tree.TupleTypeDescriptorNode) BLangNode {
	tupleTypeNode := &BLangTupleTypeNode{
		Members: make([]BLangMemberTypeDesc, 0),
	}

	types := tupleTypeDescriptorNode.MemberTypeDesc()
	for i := 0; i < types.Size(); i += 2 {
		node := types.Get(i)
		if node.Kind() == common.REST_TYPE {
			restDescriptor := node.(*tree.RestDescriptorNode)
			tupleTypeNode.Rest = n.createTypeNode(restDescriptor.TypeDescriptor()).(BType)
		} else {
			memberNode := node.(*tree.MemberTypeDescriptorNode)
			member := BLangMemberTypeDesc{
				TypeDesc: n.createTypeNode(memberNode.TypeDescriptor()),
			}
			member.pos = getPosition(n.de(), memberNode)
			tupleTypeNode.Members = append(tupleTypeNode.Members, member)
		}
	}
	tupleTypeNode.pos = getPosition(n.de(), tupleTypeDescriptorNode)
	return tupleTypeNode
}

func (n *NodeBuilder) TransformParenthesisedTypeDescriptor(parenthesisedTypeDescriptorNode *tree.ParenthesisedTypeDescriptorNode) BLangNode {
	return n.createTypeNode(parenthesisedTypeDescriptorNode.Typedesc()).(BLangNode)
}

func (n *NodeBuilder) TransformExplicitNewExpression(explicitNewBLangExpression *tree.ExplicitNewExpressionNode) BLangNode {
	typeInit := &BLangNewExpression{}
	typeInit.pos = getPosition(n.de(), explicitNewBLangExpression)
	typeInit.TypeDescriptor = n.createTypeNode(explicitNewBLangExpression.TypeDescriptor()).(BType)
	if argList := explicitNewBLangExpression.ParenthesizedArgList(); argList != nil {
		args := argList.Arguments()
		for arg := range args.Iterator() {
			typeInit.ArgsExprs = append(typeInit.ArgsExprs, n.createExpression(arg))
		}
	}
	return typeInit
}

func (n *NodeBuilder) TransformImplicitNewExpression(implicitNewBLangExpression *tree.ImplicitNewExpressionNode) BLangNode {
	typeInit := &BLangNewExpression{}
	typeInit.pos = getPosition(n.de(), implicitNewBLangExpression)
	if argList := implicitNewBLangExpression.ParenthesizedArgList(); argList != nil {
		args := argList.Arguments()
		for arg := range args.Iterator() {
			typeInit.ArgsExprs = append(typeInit.ArgsExprs, n.createExpression(arg))
		}
	}
	return typeInit
}

func (n *NodeBuilder) TransformParenthesizedArgList(parenthesizedArgList *tree.ParenthesizedArgList) BLangNode {
	panic("TransformParenthesizedArgList unimplemented")
}

func (n *NodeBuilder) TransformQueryConstructType(queryConstructTypeNode *tree.QueryConstructTypeNode) BLangNode {
	keyword := queryConstructTypeNode.Keyword()
	return &BLangIdentifier{
		Value: keyword.Text(),
		bLangNodeBase: bLangNodeBase{
			pos: getPosition(n.de(), queryConstructTypeNode),
		},
	}
}

func (n *NodeBuilder) TransformFromClause(fromClauseNode *tree.FromClauseNode) BLangNode {
	fromClause := &BLangFromClause{}
	fromClause.pos = getPosition(n.de(), fromClauseNode)
	fromClause.SetCollection(n.createExpression(fromClauseNode.Expression()))
	bindingPatternNode := fromClauseNode.TypedBindingPattern()
	fromClause.SetVariableDefinitionNode(n.createBLangVarDef(getPosition(n.de(), bindingPatternNode), bindingPatternNode,
		nil, nil))
	fromClause.IsDeclaredWithVarFlag = isDeclaredWithVar(bindingPatternNode.TypeDescriptor())
	return fromClause
}

func (n *NodeBuilder) TransformWhereClause(whereClauseNode *tree.WhereClauseNode) BLangNode {
	whereClause := &BLangWhereClause{}
	whereClause.pos = getPosition(n.de(), whereClauseNode)
	whereClause.Expression = n.createExpression(whereClauseNode.Expression())
	return whereClause
}

func (n *NodeBuilder) TransformLetClause(letClauseNode *tree.LetClauseNode) BLangNode {
	letClause := &BLangLetClause{}
	letClause.pos = getPosition(n.de(), letClauseNode)
	letVarDeclarations := letClauseNode.LetVarDeclarations()
	letClause.LetVarDeclarations = make([]BLangSimpleVariableDef, 0, letVarDeclarations.Size())
	for letVar := range letVarDeclarations.Iterator() {
		varDef := n.TransformLetVariableDeclaration(letVar).(*BLangSimpleVariableDef)
		letClause.LetVarDeclarations = append(letClause.LetVarDeclarations, *varDef)
	}
	return letClause
}

func (n *NodeBuilder) TransformJoinClause(joinClauseNode *tree.JoinClauseNode) BLangNode {
	joinClause := &BLangJoinClause{}
	joinClause.pos = getPosition(n.de(), joinClauseNode)
	joinClause.SetCollection(n.createExpression(joinClauseNode.Expression()))
	bindingPatternNode := joinClauseNode.TypedBindingPattern()
	joinClause.SetVariableDefinitionNode(
		n.createBLangVarDef(getPosition(n.de(), bindingPatternNode), bindingPatternNode, nil, nil),
	)
	joinClause.IsDeclaredWithVarFlag = isDeclaredWithVar(bindingPatternNode.TypeDescriptor())
	joinClause.IsOuterJoinFlag = joinClauseNode.OuterKeyword() != nil
	if onClauseNode := joinClauseNode.JoinOnCondition(); onClauseNode != nil {
		joinClause.OnClause = *n.TransformOnClause(onClauseNode).(*BLangOnClause)
	}
	return joinClause
}

func (n *NodeBuilder) TransformOnClause(onClauseNode *tree.OnClauseNode) BLangNode {
	onClause := &BLangOnClause{}
	onClause.pos = getPosition(n.de(), onClauseNode)
	onClause.SetOnExpression(n.createExpression(onClauseNode.OnExpression()))
	onClause.SetEqualsExpression(n.createExpression(onClauseNode.EqualsExpression()))
	return onClause
}

func (n *NodeBuilder) TransformLimitClause(limitClauseNode *tree.LimitClauseNode) BLangNode {
	limitClause := &BLangLimitClause{}
	limitClause.pos = getPosition(n.de(), limitClauseNode)
	limitClause.SetExpression(n.createExpression(limitClauseNode.Expression()))
	return limitClause
}

func (n *NodeBuilder) TransformOnConflictClause(onConflictClauseNode *tree.OnConflictClauseNode) BLangNode {
	onConflictClause := &BLangOnConflictClause{}
	onConflictClause.pos = getPosition(n.de(), onConflictClauseNode)
	onConflictClause.SetExpression(n.createExpression(onConflictClauseNode.Expression()))
	return onConflictClause
}

func (n *NodeBuilder) TransformQueryPipeline(queryPipelineNode *tree.QueryPipelineNode) BLangNode {
	panic("TransformQueryPipeline unimplemented")
}

func (n *NodeBuilder) TransformSelectClause(selectClauseNode *tree.SelectClauseNode) BLangNode {
	selectClause := &BLangSelectClause{}
	selectClause.pos = getPosition(n.de(), selectClauseNode)
	selectClause.SetExpression(n.createExpression(selectClauseNode.Expression()))
	return selectClause
}

func (n *NodeBuilder) TransformCollectClause(collectClauseNode *tree.CollectClauseNode) BLangNode {
	collectClause := &BLangCollectClause{
		NonGroupingKeys: &balCommon.UnorderedSet[string]{},
	}
	collectClause.pos = getPosition(n.de(), collectClauseNode)
	collectClause.SetExpression(n.createExpression(collectClauseNode.Expression()))
	return collectClause
}

func (n *NodeBuilder) TransformQueryExpression(queryBLangExpression *tree.QueryExpressionNode) BLangNode {
	queryExpr := &BLangQueryExpr{}
	queryExpr.pos = getPosition(n.de(), queryBLangExpression)

	if constructType := queryBLangExpression.QueryConstructType(); constructType != nil {
		switch constructType.Keyword().Text() {
		case string(TypeKind_MAP):
			queryExpr.QueryConstructType = TypeKind_MAP
		default:
			n.cx.Unimplemented("only map query construct type is supported for now", getPosition(n.de(), constructType))
		}
	}

	queryPipeline := queryBLangExpression.QueryPipeline()
	if queryPipeline == nil || queryPipeline.FromClause() == nil {
		return queryExpr
	}

	fromClause := n.TransformSyntaxNode(queryPipeline.FromClause())
	queryExpr.AddQueryClause(fromClause)

	intermediateClauses := queryPipeline.IntermediateClauses()
	for i := 0; i < intermediateClauses.Size(); i++ {
		clause := intermediateClauses.Get(i)
		switch clause.Kind() {
		case common.FROM_CLAUSE, common.JOIN_CLAUSE, common.LET_CLAUSE, common.WHERE_CLAUSE,
			common.GROUP_BY_CLAUSE, common.LIMIT_CLAUSE, common.ORDER_BY_CLAUSE:
			queryExpr.AddQueryClause(n.TransformSyntaxNode(clause))
		default:
			n.cx.Unimplemented("only from + join + let + where + group by + order by + limit + select/collect query clauses are supported for now", getPosition(n.de(), clause))
		}
	}

	resultClause := queryBLangExpression.ResultClause()
	if resultClause != nil && (resultClause.Kind() == common.SELECT_CLAUSE || resultClause.Kind() == common.COLLECT_CLAUSE) {
		queryExpr.AddQueryClause(n.TransformSyntaxNode(resultClause))
	} else if resultClause != nil {
		n.cx.Unimplemented("only select/collect result clauses are supported for now", getPosition(n.de(), resultClause))
	}

	if queryBLangExpression.OnConflictClause() != nil {
		queryExpr.AddQueryClause(n.TransformSyntaxNode(queryBLangExpression.OnConflictClause()))
	}

	return queryExpr
}

func (n *NodeBuilder) TransformQueryAction(queryActionNode *tree.QueryActionNode) BLangNode {
	panic("TransformQueryAction unimplemented")
}

func (n *NodeBuilder) TransformIntersectionTypeDescriptor(intersectionTypeDescriptorNode *tree.IntersectionTypeDescriptorNode) BLangNode {
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
	bLIntersectionType.pos = getPosition(n.de(), intersectionTypeDescriptorNode)
	return bLIntersectionType
}

func (n *NodeBuilder) TransformImplicitAnonymousFunctionParameters(implicitAnonymousFunctionParameters *tree.ImplicitAnonymousFunctionParameters) BLangNode {
	panic("TransformImplicitAnonymousFunctionParameters unimplemented")
}

func (n *NodeBuilder) TransformImplicitAnonymousFunctionExpression(implicitAnonymousFunctionBLangExpression *tree.ImplicitAnonymousFunctionExpressionNode) BLangNode {
	panic("TransformImplicitAnonymousFunctionExpression unimplemented")
}

func (n *NodeBuilder) TransformStartAction(startActionNode *tree.StartActionNode) BLangNode {
	panic("TransformStartAction unimplemented")
}

func (n *NodeBuilder) TransformFlushAction(flushActionNode *tree.FlushActionNode) BLangNode {
	panic("TransformFlushAction unimplemented")
}

func (n *NodeBuilder) TransformSingletonTypeDescriptor(singletonTypeDescriptorNode *tree.SingletonTypeDescriptorNode) BLangNode {
	bLFiniteTypeNode := &BLangFiniteTypeNode{}
	bLFiniteTypeNode.pos = getPosition(n.de(), singletonTypeDescriptorNode)
	bLFiniteTypeNode.ValueSpace = append(bLFiniteTypeNode.ValueSpace, n.createExpression(singletonTypeDescriptorNode.SimpleContExprNode()))
	return bLFiniteTypeNode
}

func (n *NodeBuilder) TransformMethodDeclaration(methodDeclarationNode *tree.MethodDeclarationNode) BLangNode {
	panic("TransformMethodDeclaration unimplemented")
}

func (n *NodeBuilder) TransformTypedBindingPattern(typedBindingPatternNode *tree.TypedBindingPatternNode) BLangNode {
	panic("TransformTypedBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformCaptureBindingPattern(captureBindingPatternNode *tree.CaptureBindingPatternNode) BLangNode {
	panic("TransformCaptureBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformWildcardBindingPattern(wildcardBindingPatternNode *tree.WildcardBindingPatternNode) BLangNode {
	bLWildCardBindingPattern := &BLangWildCardBindingPattern{}
	bLWildCardBindingPattern.pos = getPosition(n.de(), wildcardBindingPatternNode)
	return bLWildCardBindingPattern
}

func (n *NodeBuilder) TransformListBindingPattern(listBindingPatternNode *tree.ListBindingPatternNode) BLangNode {
	panic("TransformListBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformMappingBindingPattern(mappingBindingPatternNode *tree.MappingBindingPatternNode) BLangNode {
	panic("TransformMappingBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformFieldBindingPatternFull(fieldBindingPatternFullNode *tree.FieldBindingPatternFullNode) BLangNode {
	panic("TransformFieldBindingPatternFull unimplemented")
}

func (n *NodeBuilder) TransformFieldBindingPatternVarname(fieldBindingPatternVarnameNode *tree.FieldBindingPatternVarnameNode) BLangNode {
	panic("TransformFieldBindingPatternVarname unimplemented")
}

func (n *NodeBuilder) TransformRestBindingPattern(restBindingPatternNode *tree.RestBindingPatternNode) BLangNode {
	panic("TransformRestBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformErrorBindingPattern(errorBindingPatternNode *tree.ErrorBindingPatternNode) BLangNode {
	panic("TransformErrorBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformNamedArgBindingPattern(namedArgBindingPatternNode *tree.NamedArgBindingPatternNode) BLangNode {
	panic("TransformNamedArgBindingPattern unimplemented")
}

func (n *NodeBuilder) TransformAsyncSendAction(asyncSendActionNode *tree.AsyncSendActionNode) BLangNode {
	panic("TransformAsyncSendAction unimplemented")
}

func (n *NodeBuilder) TransformSyncSendAction(syncSendActionNode *tree.SyncSendActionNode) BLangNode {
	panic("TransformSyncSendAction unimplemented")
}

func (n *NodeBuilder) TransformReceiveAction(receiveActionNode *tree.ReceiveActionNode) BLangNode {
	panic("TransformReceiveAction unimplemented")
}

func (n *NodeBuilder) TransformReceiveFields(receiveFieldsNode *tree.ReceiveFieldsNode) BLangNode {
	panic("TransformReceiveFields unimplemented")
}

func (n *NodeBuilder) TransformAlternateReceive(alternateReceiveNode *tree.AlternateReceiveNode) BLangNode {
	panic("TransformAlternateReceive unimplemented")
}

func (n *NodeBuilder) TransformRestDescriptor(restDescriptorNode *tree.RestDescriptorNode) BLangNode {
	panic("TransformRestDescriptor unimplemented")
}

func (n *NodeBuilder) TransformDoubleGTToken(doubleGTTokenNode *tree.DoubleGTTokenNode) BLangNode {
	panic("TransformDoubleGTToken unimplemented")
}

func (n *NodeBuilder) TransformTrippleGTToken(trippleGTTokenNode *tree.TrippleGTTokenNode) BLangNode {
	panic("TransformTrippleGTToken unimplemented")
}

func (n *NodeBuilder) TransformWaitAction(waitActionNode *tree.WaitActionNode) BLangNode {
	panic("TransformWaitAction unimplemented")
}

func (n *NodeBuilder) TransformWaitFieldsList(waitFieldsListNode *tree.WaitFieldsListNode) BLangNode {
	panic("TransformWaitFieldsList unimplemented")
}

func (n *NodeBuilder) TransformWaitField(waitFieldNode *tree.WaitFieldNode) BLangNode {
	panic("TransformWaitField unimplemented")
}

func (n *NodeBuilder) TransformAnnotAccessExpression(annotAccessBLangExpression *tree.AnnotAccessExpressionNode) BLangNode {
	expr := &BLangAnnotAccessExpr{}
	expr.Expr = n.createExpression(annotAccessBLangExpression.Expression())
	nameReference := n.createBLangNameReference(annotAccessBLangExpression.AnnotTagReference())
	expr.PkgAlias = &nameReference[0]
	expr.AnnotationName = &nameReference[1]
	expr.SetPosition(getPosition(n.de(), annotAccessBLangExpression))
	return expr
}

func (n *NodeBuilder) TransformOptionalFieldAccessExpression(optionalFieldAccessBLangExpression *tree.OptionalFieldAccessExpressionNode) BLangNode {
	fieldName := optionalFieldAccessBLangExpression.FieldName()
	if fieldName.Kind() == common.QUALIFIED_NAME_REFERENCE {
		// @cleanup we should replace all these panics with proper internal errors. Need to problem is with return value
		// this should be detected by parser
		panic("TransformOptionalFieldAccessExpression: QUALIFIED_NAME_REFERENCE expected")
	}

	bLFieldBasedAccess := &BLangFieldBaseAccess{}
	bLFieldBasedAccess.SetOptionalAccess()
	simpleNameRef := fieldName.(*tree.SimpleNameReferenceNode)
	bLFieldBasedAccess.Field = createIdentifierFromToken(getPosition(n.de(), optionalFieldAccessBLangExpression.FieldName()), simpleNameRef.Name())

	containerExpr := optionalFieldAccessBLangExpression.Expression()
	if containerExpr.Kind() == common.BRACED_EXPRESSION {
		bracedExpr := containerExpr.(*tree.BracedExpressionNode)
		bLFieldBasedAccess.Expr = n.createExpression(bracedExpr.Expression())
	} else {
		bLFieldBasedAccess.Expr = n.createExpression(containerExpr)
	}

	bLFieldBasedAccess.pos = getPosition(n.de(), optionalFieldAccessBLangExpression)
	return bLFieldBasedAccess
}

func (n *NodeBuilder) TransformConditionalExpression(conditionalBLangExpression *tree.ConditionalExpressionNode) BLangNode {
	panic("TransformConditionalExpression unimplemented")
}

func (n *NodeBuilder) TransformEnumDeclaration(enumDeclarationNode *tree.EnumDeclarationNode) BLangNode {
	publicQualifier := false
	qualifier := enumDeclarationNode.Qualifier()
	if qualifier != nil && qualifier.Kind() == common.PUBLIC_KEYWORD {
		publicQualifier = true
	}

	memberNodes := enumDeclarationNode.EnumMemberList()
	memberTypeNodes := make([]TypeDescriptor, 0)
	for memberNode := range memberNodes.Iterator() {
		if memberNode.Kind() != common.ENUM_MEMBER {
			continue
		}
		enumMember := memberNode.(*tree.EnumMemberNode)
		if enumMember.Identifier() == nil || enumMember.Identifier().IsMissing() {
			n.cx.InternalError("missing enum member identifier", getPosition(n.de(), enumMember))
			continue
		}
		constantNode, redeclared := n.transformEnumMember(enumMember, publicQualifier)
		if redeclared {
			continue
		}
		if n.currentCompUnit == nil {
			n.cx.InternalError("enum constants can only be added at module level", getPosition(n.de(), enumMember))
			continue
		}
		n.currentCompUnit.AddTopLevelNode(constantNode)
		memberTypeNodes = append(memberTypeNodes, n.createTypeNode(enumMember.Identifier()))
	}

	typeDef := NewBLangTypeDefinition()
	typeDef.pos = getPositionWithoutMetadata(n.de(), enumDeclarationNode)
	if publicQualifier {
		typeDef.SetPublic()
	}

	identifierPos := getPosition(n.de(), enumDeclarationNode.Identifier())
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

func (n *NodeBuilder) TransformEnumMember(enumMemberNode *tree.EnumMemberNode) BLangNode {
	constantNode, _ := n.transformEnumMember(enumMemberNode, false)
	return constantNode
}

func (n *NodeBuilder) transformEnumMember(enumMemberNode *tree.EnumMemberNode, publicQualifier bool) (*BLangConstant, bool) {
	constantNode := createConstantNode()
	constantNode.pos = getPositionWithoutMetadata(n.de(), enumMemberNode)
	if publicQualifier {
		constantNode.SetPublic()
	}

	identifierPos := getPosition(n.de(), enumMemberNode.Identifier())
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

	constantName := constantNode.Name.GetValue()
	if _, exists := n.constantSet[constantName]; exists {
		n.cx.SemanticError("redeclared symbol '"+constantName+"'", constantNode.Name.GetPosition())
		return nil, true
	} else {
		n.constantSet[constantName] = getConstantInitValue(constantNode.Expr)
	}

	return constantNode, false
}

func (n *NodeBuilder) TransformArrayTypeDescriptor(arrayTypeDescriptorNode *tree.ArrayTypeDescriptorNode) BLangNode {
	position := getPosition(n.de(), arrayTypeDescriptorNode)
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

func (n *NodeBuilder) TransformArrayDimension(arrayDimensionNode *tree.ArrayDimensionNode) BLangNode {
	panic("TransformArrayDimension unimplemented")
}

func (n *NodeBuilder) TransformTransactionStatement(transactionStatementNode *tree.TransactionStatementNode) BLangNode {
	panic("TransformTransactionStatement unimplemented")
}

func (n *NodeBuilder) TransformRollbackStatement(rollbackStatementNode *tree.RollbackStatementNode) BLangNode {
	panic("TransformRollbackStatement unimplemented")
}

func (n *NodeBuilder) TransformRetryStatement(retryStatementNode *tree.RetryStatementNode) BLangNode {
	panic("TransformRetryStatement unimplemented")
}

func (n *NodeBuilder) TransformCommitAction(commitActionNode *tree.CommitActionNode) BLangNode {
	panic("TransformCommitAction unimplemented")
}

func (n *NodeBuilder) TransformTransactionalExpression(transactionalBLangExpression *tree.TransactionalExpressionNode) BLangNode {
	panic("TransformTransactionalExpression unimplemented")
}

func (n *NodeBuilder) TransformByteArrayLiteral(byteArrayLiteralNode *tree.ByteArrayLiteralNode) BLangNode {
	panic("TransformByteArrayLiteral unimplemented")
}

func (n *NodeBuilder) TransformXMLFilterExpression(xMLFilterBLangExpression *tree.XMLFilterExpressionNode) BLangNode {
	panic("TransformXMLFilterExpression unimplemented")
}

func (n *NodeBuilder) TransformXMLStepExpression(xMLStepBLangExpression *tree.XMLStepExpressionNode) BLangNode {
	panic("TransformXMLStepExpression unimplemented")
}

func (n *NodeBuilder) TransformXMLNamePatternChaining(xMLNamePatternChainingNode *tree.XMLNamePatternChainingNode) BLangNode {
	panic("TransformXMLNamePatternChaining unimplemented")
}

func (n *NodeBuilder) TransformXMLStepIndexedExtend(xMLStepIndexedExtendNode *tree.XMLStepIndexedExtendNode) BLangNode {
	panic("TransformXMLStepIndexedExtend unimplemented")
}

func (n *NodeBuilder) TransformXMLStepMethodCallExtend(xMLStepMethodCallExtendNode *tree.XMLStepMethodCallExtendNode) BLangNode {
	panic("TransformXMLStepMethodCallExtend unimplemented")
}

func (n *NodeBuilder) TransformXMLAtomicNamePattern(xMLAtomicNamePatternNode *tree.XMLAtomicNamePatternNode) BLangNode {
	panic("TransformXMLAtomicNamePattern unimplemented")
}

func (n *NodeBuilder) TransformTypeReferenceTypeDesc(typeReferenceTypeDescNode *tree.TypeReferenceTypeDescNode) BLangNode {
	panic("TransformTypeReferenceTypeDesc unimplemented")
}

func (n *NodeBuilder) TransformMatchStatement(matchStatementNode *tree.MatchStatementNode) BLangNode {
	matchStatement := &BLangMatchStatement{}
	matchStmtExpr := n.createExpression(matchStatementNode.Condition())
	matchStatement.Expr = matchStmtExpr

	matchClauses := matchStatementNode.MatchClauses()
	for matchClauseNode := range matchClauses.Iterator() {
		bLangMatchClause := &BLangMatchClause{}
		bLangMatchClause.pos = getPosition(n.de(), matchClauseNode)

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

	matchStatement.pos = getPosition(n.de(), matchStatementNode)
	return matchStatement
}

func (n *NodeBuilder) transformMatchPattern(matchPattern tree.Node, matchStmtExpr BLangExpression) BLangMatchPattern {
	matchPatternPos := getPosition(n.de(), matchPattern)
	kind := matchPattern.Kind()

	switch kind {
	case common.SIMPLE_NAME_REFERENCE:
		nameRef := matchPattern.(*tree.SimpleNameReferenceNode)
		if nameRef.Name().Text() == "_" {
			bLangWildCard := &BLangWildCardMatchPattern{}
			bLangWildCard.pos = matchPatternPos
			return bLangWildCard
		}
		bLangConstPattern := &BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.pos = matchPatternPos
		return bLangConstPattern

	case common.IDENTIFIER_TOKEN:
		idToken := matchPattern.(tree.Token)
		if idToken.Text() == "_" {
			bLangWildCard := &BLangWildCardMatchPattern{}
			bLangWildCard.pos = matchPatternPos
			return bLangWildCard
		}
		bLangConstPattern := &BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.pos = matchPatternPos
		return bLangConstPattern

	case common.NUMERIC_LITERAL,
		common.STRING_LITERAL,
		common.QUALIFIED_NAME_REFERENCE,
		common.NULL_LITERAL,
		common.NIL_LITERAL,
		common.BOOLEAN_LITERAL,
		common.UNARY_EXPRESSION:
		bLangConstPattern := &BLangConstPattern{}
		bLangConstPattern.Expr = n.createExpression(matchPattern)
		bLangConstPattern.pos = matchPatternPos
		return bLangConstPattern

	case common.PIPE_TOKEN, common.COMMA_TOKEN:
		// Skip separator tokens in match pattern lists
		return nil

	default:
		n.cx.InternalError(fmt.Sprintf("unexpected match pattern kind: %v", kind), matchPatternPos)
		return nil
	}
}

func (n *NodeBuilder) TransformMatchClause(matchClauseNode *tree.MatchClauseNode) BLangNode {
	panic("TransformMatchClause unimplemented")
}

func (n *NodeBuilder) TransformMatchGuard(matchGuardNode *tree.MatchGuardNode) BLangNode {
	panic("TransformMatchGuard unimplemented")
}

func (n *NodeBuilder) TransformDistinctTypeDescriptor(distinctTypeDescriptorNode *tree.DistinctTypeDescriptorNode) BLangNode {
	n.cx.Unimplemented("anonymous distinct types not supported", getPosition(n.de(), distinctTypeDescriptorNode))
	neverType := &BLangValueType{TypeKind: TypeKind_NEVER}
	neverType.pos = getPosition(n.de(), distinctTypeDescriptorNode)
	return neverType
}

func (n *NodeBuilder) TransformListMatchPattern(listMatchPatternNode *tree.ListMatchPatternNode) BLangNode {
	panic("TransformListMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformRestMatchPattern(restMatchPatternNode *tree.RestMatchPatternNode) BLangNode {
	panic("TransformRestMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformMappingMatchPattern(mappingMatchPatternNode *tree.MappingMatchPatternNode) BLangNode {
	panic("TransformMappingMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformFieldMatchPattern(fieldMatchPatternNode *tree.FieldMatchPatternNode) BLangNode {
	panic("TransformFieldMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformErrorMatchPattern(errorMatchPatternNode *tree.ErrorMatchPatternNode) BLangNode {
	panic("TransformErrorMatchPattern unimplemented")
}

func (n *NodeBuilder) TransformNamedArgMatchPattern(namedArgMatchPatternNode *tree.NamedArgMatchPatternNode) BLangNode {
	panic("TransformNamedArgMatchPattern unimplemented")
}

// Helper functions for markdown documentation transformation

func (n *NodeBuilder) addReferencesAndReturnDocumentationText(references *[]BLangMarkdownReferenceDocumentation, docElements tree.NodeList[tree.Node]) string {
	var docText strings.Builder
	for i := 0; i < docElements.Size(); i++ {
		element := docElements.Get(i)
		if element.Kind() == common.BALLERINA_NAME_REFERENCE {
			bLangRefDoc := &BLangMarkdownReferenceDocumentation{}
			balNameRefNode := element.(*tree.BallerinaNameReferenceNode)

			bLangRefDoc.pos = getPosition(n.de(), balNameRefNode)

			startBacktick := balNameRefNode.StartBacktick()
			backtickContent := balNameRefNode.NameReference()
			endBacktick := balNameRefNode.EndBacktick()

			contentString := ""
			if backtickContent != nil && !backtickContent.IsMissing() {
				// Use InternalNode() to get STNode and convert to source code
				contentString = tree.ToSourceCode(backtickContent.InternalNode())
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
		} else if element.Kind() == common.DOCUMENTATION_DESCRIPTION {
			if token, ok := element.(tree.Token); ok {
				docText.WriteString(token.Text())
			}
		} else if element.Kind() == common.INLINE_CODE_REFERENCE {
			inlineCodeRefNode := element.(*tree.InlineCodeReferenceNode)
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

func (n *NodeBuilder) transformDocumentationBacktickContent(backtickContent tree.Node, bLangRefDoc *BLangMarkdownReferenceDocumentation) {
	switch backtickContent.Kind() {
	case common.CODE_CONTENT:
		// reaching here means ballerina name reference is syntactically invalid.
		// therefore, set hasParserWarnings to true. so that,
		// doc analyzer will avoid further checks on this.
		bLangRefDoc.HasParserWarnings = true
	case common.QUALIFIED_NAME_REFERENCE:
		qualifiedRef := backtickContent.(*tree.QualifiedNameReferenceNode)
		modulePrefix := qualifiedRef.ModulePrefix()
		identifier := qualifiedRef.Identifier()
		if modulePrefix != nil && !modulePrefix.IsMissing() {
			bLangRefDoc.Qualifier = modulePrefix.Text()
		}
		if identifier != nil && !identifier.IsMissing() {
			bLangRefDoc.Identifier = identifier.Text()
		}
	case common.SIMPLE_NAME_REFERENCE:
		simpleRef := backtickContent.(*tree.SimpleNameReferenceNode)
		name := simpleRef.Name()
		if name != nil && !name.IsMissing() {
			bLangRefDoc.Identifier = name.Text()
		}
	case common.FUNCTION_CALL:
		funcCallExpr := backtickContent.(*tree.FunctionCallExpressionNode)
		funcName := funcCallExpr.FunctionName()
		if funcName.Kind() == common.QUALIFIED_NAME_REFERENCE {
			qualifiedRef := funcName.(*tree.QualifiedNameReferenceNode)
			modulePrefix := qualifiedRef.ModulePrefix()
			identifier := qualifiedRef.Identifier()
			if modulePrefix != nil && !modulePrefix.IsMissing() {
				bLangRefDoc.Qualifier = modulePrefix.Text()
			}
			if identifier != nil && !identifier.IsMissing() {
				bLangRefDoc.Identifier = identifier.Text()
			}
		} else {
			simpleRef := funcName.(*tree.SimpleNameReferenceNode)
			name := simpleRef.Name()
			if name != nil && !name.IsMissing() {
				bLangRefDoc.Identifier = name.Text()
			}
		}
	case common.METHOD_CALL:
		methodCallExprNode := backtickContent.(*tree.MethodCallExpressionNode)
		methodName := methodCallExprNode.MethodName()
		if simpleRef, ok := methodName.(*tree.SimpleNameReferenceNode); ok {
			name := simpleRef.Name()
			if name != nil && !name.IsMissing() {
				bLangRefDoc.Identifier = name.Text()
			}
		}
		refName := methodCallExprNode.Expression()
		if refName.Kind() == common.QUALIFIED_NAME_REFERENCE {
			qualifiedRef := refName.(*tree.QualifiedNameReferenceNode)
			identifier := qualifiedRef.Identifier()
			if identifier != nil && !identifier.IsMissing() {
				bLangRefDoc.TypeName = identifier.Text()
			}
			modulePrefix := qualifiedRef.ModulePrefix()
			if modulePrefix != nil && !modulePrefix.IsMissing() {
				bLangRefDoc.Qualifier = modulePrefix.Text()
			}
		} else if refName.Kind() == common.SIMPLE_NAME_REFERENCE {
			simpleRef := refName.(*tree.SimpleNameReferenceNode)
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

func (n *NodeBuilder) transformCodeBlock(documentationLines *[]BLangMarkdownDocumentationLine, codeBlockNode *tree.MarkdownCodeBlockNode) {
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
	bLangDocLine.pos = getPosition(n.de(), codeBlockNode.StartLineHashToken())
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

func (n *NodeBuilder) TransformOrderByClause(orderByClauseNode *tree.OrderByClauseNode) BLangNode {
	orderByClause := &BLangOrderByClause{}
	orderByClause.pos = getPosition(n.de(), orderByClauseNode)

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

func (n *NodeBuilder) TransformOrderKey(orderKeyNode *tree.OrderKeyNode) BLangNode {
	orderKey := &BLangOrderKey{}
	orderKey.pos = getPosition(n.de(), orderKeyNode)
	orderKey.Expression = n.createExpression(orderKeyNode.Expression())
	if dir := orderKeyNode.OrderDirection(); dir != nil && dir.Kind() == common.DESCENDING_KEYWORD {
		orderKey.IsDescending = true
	} else {
		orderKey.IsDescending = false
	}
	return orderKey
}

func (n *NodeBuilder) TransformGroupByClause(groupByClauseNode *tree.GroupByClauseNode) BLangNode {
	groupByClause := &BLangGroupByClause{
		NonGroupingKeys: &balCommon.UnorderedSet[string]{},
	}
	groupByClause.pos = getPosition(n.de(), groupByClauseNode)

	groupingKeys := groupByClauseNode.GroupingKey()
	for node := range groupingKeys.Iterator() {
		if node.Kind() == common.COMMA_TOKEN {
			continue
		}
		groupingKey := &BLangGroupingKey{}
		groupingKey.pos = getPosition(n.de(), node)
		if node.Kind() == common.SIMPLE_NAME_REFERENCE || node.Kind() == common.IDENTIFIER_TOKEN {
			varRef, ok := n.createExpression(node).(*BLangSimpleVarRef)
			if !ok {
				panic("expected grouping key variable reference to be a simple variable reference")
			}
			groupingKey.SetGroupingKey(varRef)
		} else {
			keyNode, ok := n.TransformGroupingKeyVarDeclaration(node.(*tree.GroupingKeyVarDeclarationNode)).(*BLangGroupingKey)
			if !ok {
				panic("expected grouping key declaration to produce a BLangGroupingKey")
			}
			groupingKey = keyNode
		}
		groupByClause.AddGroupingKey(groupingKey)
	}
	return groupByClause
}

func (n *NodeBuilder) TransformGroupingKeyVarDeclaration(groupingKeyVarDeclarationNode *tree.GroupingKeyVarDeclarationNode) BLangNode {
	pos := getPosition(n.de(), groupingKeyVarDeclarationNode)
	groupingKey := &BLangGroupingKey{}
	groupingKey.pos = pos

	variableNode := n.getBLangVariableNode(groupingKeyVarDeclarationNode.SimpleBindingPattern(), pos)
	simpleVar, ok := variableNode.(*BLangSimpleVariable)
	if !ok {
		panic("expected grouping key declaration to create a simple variable reference")
	}
	simpleVar.SetPosition(pos)
	simpleVar.SetInitialExpression(n.createExpression(groupingKeyVarDeclarationNode.Expression()))

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

func (n *NodeBuilder) TransformOnFailClause(onFailClauseNode *tree.OnFailClauseNode) BLangNode {
	panic("TransformOnFailClause unimplemented")
}

func (n *NodeBuilder) TransformDoStatement(doStatementNode *tree.DoStatementNode) BLangNode {
	panic("TransformDoStatement unimplemented")
}

func (n *NodeBuilder) TransformClassDefinition(classDefinitionNode *tree.ClassDefinitionNode) BLangNode {
	blangClass := NewBLangClassDefinition()
	blangClass.pos = getPositionWithoutMetadata(n.de(), classDefinitionNode)

	n.populateMetadata(classDefinitionNode.Metadata(), &blangClass)

	// Set name
	nameIdentifier := createIdentifierFromToken(getPosition(n.de(), classDefinitionNode.ClassName()), classDefinitionNode.ClassName())
	blangClass.Name = &nameIdentifier

	// Handle visibility qualifier
	if visQual := classDefinitionNode.VisibilityQualifier(); visQual != nil {
		if visQual.Kind() == common.PUBLIC_KEYWORD {
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

func (n *NodeBuilder) setClassQualifiers(blangClass *BLangClassDefinition, qualifiers tree.NodeList[tree.Token]) {
	for qualifier := range qualifiers.Iterator() {
		switch qualifier.Kind() {
		case common.DISTINCT_KEYWORD:
			blangClass.SetDistinct()
		case common.CLIENT_KEYWORD:
			blangClass.SetClient()
		case common.READONLY_KEYWORD:
			blangClass.SetReadonly()
		case common.SERVICE_KEYWORD:
			blangClass.SetService()
		case common.ISOLATED_KEYWORD:
			blangClass.SetIsolated()
		}
	}
}

func (n *NodeBuilder) transformClassField(objectField *tree.ObjectFieldNode) *BLangSimpleVariable {
	bLSimpleVar := createSimpleVariableNode()
	identifier := createIdentifierFromToken(getPosition(n.de(), objectField.FieldName()), objectField.FieldName())
	bLSimpleVar.SetName(&identifier)
	bLSimpleVar.pos = getPosition(n.de(), objectField)
	bLSimpleVar.SetTypeNode(n.createTypeNode(objectField.TypeName()).(BType))

	if vis := objectField.VisibilityQualifier(); vis != nil {
		if vis.Kind() == common.PUBLIC_KEYWORD {
			bLSimpleVar.SetPublic()
		} else if vis.Kind() == common.PRIVATE_KEYWORD {
			bLSimpleVar.SetPrivate()
		}
	}

	qualifiers := objectField.QualifierList()
	for qualifier := range qualifiers.Iterator() {
		if qualifier.Kind() == common.FINAL_KEYWORD {
			bLSimpleVar.SetFinal()
		}
	}

	if expr := objectField.Expression(); expr != nil {
		bLSimpleVar.SetInitialExpression(n.createExpression(expr))
	}

	n.populateMetadata(objectField.Metadata(), bLSimpleVar)
	return bLSimpleVar
}

func (n *NodeBuilder) TransformResourcePathParameter(resourcePathParameterNode *tree.ResourcePathParameterNode) BLangNode {
	seg := &BLangResourcePathSegment{}
	switch resourcePathParameterNode.Kind() {
	case common.RESOURCE_PATH_SEGMENT_PARAM:
		seg.Kind = ResourcePathSegmentParam
	case common.RESOURCE_PATH_REST_PARAM:
		seg.Kind = ResourcePathSegmentParamRest
	default:
		n.cx.InternalError(fmt.Sprintf("unexpected resource path parameter node kind: %v", resourcePathParameterNode.Kind()), getPosition(n.de(), resourcePathParameterNode))
	}
	seg.pos = getPosition(n.de(), resourcePathParameterNode)
	nameTok := resourcePathParameterNode.ParamName()
	if nameTok != nil && !nameTok.IsMissing() {
		seg.Name = createIdentifierFromToken(getPosition(n.de(), nameTok), nameTok).Value
	}
	if td := resourcePathParameterNode.TypeDescriptor(); td != nil {
		seg.ParamType = n.createTypeNode(td).(BType)
	}
	return seg
}

func (n *NodeBuilder) createResourceMethodNode(funcDef *tree.FunctionDefinition) *BLangResourceMethod {
	rm := &BLangResourceMethod{}
	rm.pos = getPositionWithoutMetadata(n.de(), funcDef)
	rm.Name = createIdentifierFromTokenInternal(getPosition(n.de(), funcDef.FunctionName()), funcDef.FunctionName(), false)
	setFunctionQualifiersOnBase(&rm.bLangInvokableNodeBase, funcDef.QualifierList())
	rm.SetAttached()
	rm.SetResource()
	n.anonTypeNameSuffixes = append(n.anonTypeNameSuffixes, rm.Name.Value)
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

func (n *NodeBuilder) createResourcePathSegments(pathNodes tree.NodeList[tree.Node]) []BLangResourcePathSegment {
	var segments []BLangResourcePathSegment
	for node := range pathNodes.Iterator() {
		switch node.Kind() {
		case common.SLASH_TOKEN:
			continue
		case common.DOT_TOKEN:
			continue
		case common.IDENTIFIER_TOKEN:
			tok := node.(tree.Token)
			seg := BLangResourcePathSegment{Kind: ResourcePathSegmentName, Name: tok.Text()}
			seg.pos = getPosition(n.de(), node)
			segments = append(segments, seg)
		case common.RESOURCE_PATH_SEGMENT_PARAM, common.RESOURCE_PATH_REST_PARAM:
			param := node.(*tree.ResourcePathParameterNode)
			segments = append(segments, *n.TransformResourcePathParameter(param).(*BLangResourcePathSegment))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource path node kind: %v", node.Kind()), getPosition(n.de(), node))
		}
	}
	return segments
}

func (n *NodeBuilder) TransformRequiredExpression(requiredBLangExpression *tree.RequiredExpressionNode) BLangNode {
	panic("TransformRequiredExpression unimplemented")
}

func (n *NodeBuilder) TransformErrorConstructorExpression(errorConstructorBLangExpression *tree.ErrorConstructorExpressionNode) BLangNode {
	result := &BLangErrorConstructorExpr{}
	result.pos = getPosition(n.de(), errorConstructorBLangExpression)

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
		case common.POSITIONAL_ARG:
			posArg := arg.(*tree.PositionalArgumentNode)
			expr := n.createExpression(posArg.Expression())
			positionalArgs = append(positionalArgs, expr)

		case common.NAMED_ARG:
			namedArgNode := arg.(*tree.NamedArgumentNode)
			namedArg := n.TransformNamedArgument(namedArgNode).(*BLangNamedArgsExpression)
			namedArgs = append(namedArgs, *namedArg)
		case common.REST_ARG:
			n.cx.InternalError("rest arguments not supported in error constructor", getPosition(n.de(), arg))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected argument kind: %v", arg.Kind()), getPosition(n.de(), arg))
		}
	}

	result.PositionalArgs = positionalArgs
	result.NamedArgs = namedArgs

	return result
}

func (n *NodeBuilder) TransformParameterizedTypeDescriptor(parameterizedTypeDescriptorNode *tree.ParameterizedTypeDescriptorNode) BLangNode {
	switch parameterizedTypeDescriptorNode.Kind() {
	case common.ERROR_TYPE_DESC:
		return n.transformErrorTypeDescriptor(parameterizedTypeDescriptorNode)
	case common.TYPEDESC_TYPE_DESC:
		return n.transformTypedescTypeDescriptor(parameterizedTypeDescriptorNode)
	case common.XML_TYPE_DESC:
		return n.transformXMLTypeDescriptor(parameterizedTypeDescriptorNode)
	}
	panic("TransformParameterizedTypeDescriptor supported only for error, typedesc and xml type descriptors")
}

func (n *NodeBuilder) transformTypedescTypeDescriptor(node *tree.ParameterizedTypeDescriptorNode) BLangNode {
	typeParamNode := node.TypeParamNode()
	if typeParamNode == nil {
		valueType := &BLangValueType{}
		valueType.pos = getPosition(n.de(), node)
		valueType.TypeKind = TypeKind_TYPEDESC
		return valueType
	}
	constrainedType := &BLangConstrainedType{}
	constrainedType.pos = getPosition(n.de(), node)
	base := &BLangValueType{}
	base.pos = getPosition(n.de(), node)
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

func (n *NodeBuilder) transformXMLTypeDescriptor(parameterizedTypeDescriptorNode *tree.ParameterizedTypeDescriptorNode) BLangNode {
	pos := getPosition(n.de(), parameterizedTypeDescriptorNode)
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

func (n *NodeBuilder) transformErrorTypeDescriptor(errorTypeDescriptorNode *tree.ParameterizedTypeDescriptorNode) BLangNode {
	errorType := &BLangErrorTypeNode{}
	errorType.pos = getPosition(n.de(), errorTypeDescriptorNode)

	// Handle optional type parameter
	typeParamNode := errorTypeDescriptorNode.TypeParamNode()
	if typeParamNode != nil {
		errorType.DetailType = TypeData{
			TypeDescriptor: n.createTypeNode(typeParamNode),
		}
	}

	// Check if this is a distinct error type
	parent := errorTypeDescriptorNode.Parent()
	if parent.Kind() == common.DISTINCT_TYPE_DESC {
		errorType.SetDistinct()
	}

	return errorType
}

func (n *NodeBuilder) TransformSpreadMember(spreadMemberNode *tree.SpreadMemberNode) BLangNode {
	return n.createExpression(spreadMemberNode.Expression()).(BLangNode)
}

func (n *NodeBuilder) TransformClientResourceAccessAction(node *tree.ClientResourceAccessActionNode) BLangNode {
	action := &BLangClientResourceAccessAction{}
	action.pos = getPosition(n.de(), node)
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

func (n *NodeBuilder) createResourceAccessSegments(pathNodes tree.NodeList[tree.Node]) []BLangResourceAccessSegment {
	var segments []BLangResourceAccessSegment
	for node := range pathNodes.Iterator() {
		switch node.Kind() {
		case common.SLASH_TOKEN, common.DOT_TOKEN:
			continue
		case common.IDENTIFIER_TOKEN:
			tok := node.(tree.Token)
			seg := BLangResourceAccessSegment{Kind: ResourceAccessSegmentName, Name: tok.Text()}
			seg.pos = getPosition(n.de(), node)
			segments = append(segments, seg)
		case common.COMPUTED_RESOURCE_ACCESS_SEGMENT:
			computed := node.(*tree.ComputedResourceAccessSegmentNode)
			segments = append(segments, *n.TransformComputedResourceAccessSegment(computed).(*BLangResourceAccessSegment))
		case common.RESOURCE_ACCESS_REST_SEGMENT:
			n.cx.Unimplemented("resource access rest segments are not yet supported", getPosition(n.de(), node))
		default:
			n.cx.InternalError(fmt.Sprintf("unexpected resource access segment kind: %v", node.Kind()), getPosition(n.de(), node))
		}
	}
	return segments
}

func (n *NodeBuilder) TransformComputedResourceAccessSegment(node *tree.ComputedResourceAccessSegmentNode) BLangNode {
	seg := &BLangResourceAccessSegment{Kind: ResourceAccessSegmentComputed}
	seg.pos = getPosition(n.de(), node)
	seg.Expr = n.createExpression(node.Expression())
	return seg
}

func (n *NodeBuilder) TransformResourceAccessRestSegment(resourceAccessRestSegmentNode *tree.ResourceAccessRestSegmentNode) BLangNode {
	panic("TransformResourceAccessRestSegment unimplemented")
}

func (n *NodeBuilder) TransformReSequence(reSequenceNode *tree.ReSequenceNode) BLangNode {
	panic("TransformReSequence unimplemented")
}

func (n *NodeBuilder) TransformReAtomQuantifier(reAtomQuantifierNode *tree.ReAtomQuantifierNode) BLangNode {
	panic("TransformReAtomQuantifier unimplemented")
}

func (n *NodeBuilder) TransformReAtomCharOrEscape(reAtomCharOrEscapeNode *tree.ReAtomCharOrEscapeNode) BLangNode {
	panic("TransformReAtomCharOrEscape unimplemented")
}

func (n *NodeBuilder) TransformReQuoteEscape(reQuoteEscapeNode *tree.ReQuoteEscapeNode) BLangNode {
	panic("TransformReQuoteEscape unimplemented")
}

func (n *NodeBuilder) TransformReSimpleCharClassEscape(reSimpleCharClassEscapeNode *tree.ReSimpleCharClassEscapeNode) BLangNode {
	panic("TransformReSimpleCharClassEscape unimplemented")
}

func (n *NodeBuilder) TransformReUnicodePropertyEscape(reUnicodePropertyEscapeNode *tree.ReUnicodePropertyEscapeNode) BLangNode {
	panic("TransformReUnicodePropertyEscape unimplemented")
}

func (n *NodeBuilder) TransformReUnicodeScript(reUnicodeScriptNode *tree.ReUnicodeScriptNode) BLangNode {
	panic("TransformReUnicodeScript unimplemented")
}

func (n *NodeBuilder) TransformReUnicodeGeneralCategory(reUnicodeGeneralCategoryNode *tree.ReUnicodeGeneralCategoryNode) BLangNode {
	panic("TransformReUnicodeGeneralCategory unimplemented")
}

func (n *NodeBuilder) TransformReCharacterClass(reCharacterClassNode *tree.ReCharacterClassNode) BLangNode {
	panic("TransformReCharacterClass unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRangeWithReCharSet(reCharSetRangeWithReCharSetNode *tree.ReCharSetRangeWithReCharSetNode) BLangNode {
	panic("TransformReCharSetRangeWithReCharSet unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRange(reCharSetRangeNode *tree.ReCharSetRangeNode) BLangNode {
	panic("TransformReCharSetRange unimplemented")
}

func (n *NodeBuilder) TransformReCharSetAtomWithReCharSetNoDash(reCharSetAtomWithReCharSetNoDashNode *tree.ReCharSetAtomWithReCharSetNoDashNode) BLangNode {
	panic("TransformReCharSetAtomWithReCharSetNoDash unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRangeNoDashWithReCharSet(reCharSetRangeNoDashWithReCharSetNode *tree.ReCharSetRangeNoDashWithReCharSetNode) BLangNode {
	panic("TransformReCharSetRangeNoDashWithReCharSet unimplemented")
}

func (n *NodeBuilder) TransformReCharSetRangeNoDash(reCharSetRangeNoDashNode *tree.ReCharSetRangeNoDashNode) BLangNode {
	panic("TransformReCharSetRangeNoDash unimplemented")
}

func (n *NodeBuilder) TransformReCharSetAtomNoDashWithReCharSetNoDash(reCharSetAtomNoDashWithReCharSetNoDashNode *tree.ReCharSetAtomNoDashWithReCharSetNoDashNode) BLangNode {
	panic("TransformReCharSetAtomNoDashWithReCharSetNoDash unimplemented")
}

func (n *NodeBuilder) TransformReCapturingGroups(reCapturingGroupsNode *tree.ReCapturingGroupsNode) BLangNode {
	panic("TransformReCapturingGroups unimplemented")
}

func (n *NodeBuilder) TransformReFlagExpression(reFlagBLangExpression *tree.ReFlagExpressionNode) BLangNode {
	panic("TransformReFlagExpression unimplemented")
}

func (n *NodeBuilder) TransformReFlagsOnOff(reFlagsOnOffNode *tree.ReFlagsOnOffNode) BLangNode {
	panic("TransformReFlagsOnOff unimplemented")
}

func (n *NodeBuilder) TransformReFlags(reFlagsNode *tree.ReFlagsNode) BLangNode {
	panic("TransformReFlags unimplemented")
}

func (n *NodeBuilder) TransformReAssertion(reAssertionNode *tree.ReAssertionNode) BLangNode {
	panic("TransformReAssertion unimplemented")
}

func (n *NodeBuilder) TransformReQuantifier(reQuantifierNode *tree.ReQuantifierNode) BLangNode {
	panic("TransformReQuantifier unimplemented")
}

func (n *NodeBuilder) TransformReBracedQuantifier(reBracedQuantifierNode *tree.ReBracedQuantifierNode) BLangNode {
	panic("TransformReBracedQuantifier unimplemented")
}

func (n *NodeBuilder) TransformMemberTypeDescriptor(memberTypeDescriptorNode *tree.MemberTypeDescriptorNode) BLangNode {
	panic("TransformMemberTypeDescriptor unimplemented")
}

func (n *NodeBuilder) TransformReceiveField(receiveFieldNode *tree.ReceiveFieldNode) BLangNode {
	panic("TransformReceiveField unimplemented")
}

func (n *NodeBuilder) TransformNaturalExpression(naturalBLangExpression *tree.NaturalExpressionNode) BLangNode {
	panic("TransformNaturalExpression unimplemented")
}

func (n *NodeBuilder) TransformToken(token tree.Token) BLangNode {
	kind := token.Kind()
	switch kind {
	case common.XML_TEXT_CONTENT, common.TEMPLATE_STRING, common.CLOSE_BRACE_TOKEN, common.PROMPT_CONTENT:
		return n.createSimpleLiteral(token).(BLangNode)
	default:
		if isTokenInRegExp(kind) {
			return n.createSimpleLiteral(token).(BLangNode)
		}
		panic("TransformToken: Syntax kind is not supported: " + kind.StrValue())
	}
}

func (n *NodeBuilder) TransformIdentifierToken(identifier *tree.IdentifierToken) BLangNode {
	panic("TransformIdentifierToken unimplemented")
}

func getConstantInitValue(expr BLangActionOrExpression) string {
	type constantValue interface {
		GetValue() any
		GetOriginalValue() string
	}
	if cv, ok := expr.(constantValue); ok {
		if v := cv.GetValue(); v != nil {
			return fmt.Sprintf("%v", v)
		}
		return cv.GetOriginalValue()
	}
	return ""
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

func (n *NodeBuilder) getBLangVariableNode(bindingPattern tree.BindingPatternNode, varPos diagnostics.Location) VariableNode {
	var varName tree.Token
	switch bindingPattern.Kind() {
	case common.WILDCARD_BINDING_PATTERN:
		ignore := createIgnoreIdentifier(n.de(), bindingPattern)
		simpleVar := createSimpleVariableNode()
		simpleVar.SetName(&ignore)
		simpleVar.pos = varPos
		return simpleVar
	case common.MAPPING_BINDING_PATTERN, common.LIST_BINDING_PATTERN, common.ERROR_BINDING_PATTERN, common.REST_BINDING_PATTERN:
		panic("unimplemented")
	case common.CAPTURE_BINDING_PATTERN:
		fallthrough
	default:
		captureBindingPattern := bindingPattern.(*tree.CaptureBindingPatternNode)
		varName = captureBindingPattern.VariableName()
	}

	return createSimpleVariableNodeWithLocationTokenLocation(varPos, varName, getPosition(n.de(), varName))
}

func (n *NodeBuilder) reportSyntaxDiagnostics(node tree.Node) {
	diagnostics := innermostDiagnosticNodes(node)
	if len(diagnostics) == 0 {
		return
	}
	for _, diagnostic := range diagnostics {
		n.cx.SyntaxError(diagnosticMessage(diagnostic), getPosition(n.de(), diagnostic))
	}
}
