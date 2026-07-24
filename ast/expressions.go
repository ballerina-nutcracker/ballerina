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
	"strconv"
	"strings"

	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
	"ballerina/values"
)

type BLangActionOrExpression interface {
	BLangNode
	actionOrExpression()
}

type BLangExpression interface {
	BLangActionOrExpression
	expressionNode()
}

type BLangAction interface {
	BLangActionOrExpression
	actionNode()
}
type Channel struct {
	Sender     string
	Receiver   string
	EventIndex int
}

func (c *Channel) WorkerPairId() string {
	return WorkerPairId(c.Sender, c.Receiver)
}

func (c *Channel) ChannelId() string {
	return c.Sender + "->" + c.Receiver + ":" + strconv.Itoa(c.EventIndex)
}

func WorkerPairId(sender, receiver string) string {
	return sender + "->" + receiver
}

type (
	BLangMarkdownDocumentationLine struct {
		bLangExpressionBase
		Text string
	}
	BLangMarkdownParameterDocumentation struct {
		bLangExpressionBase
		ParameterName               *BLangIdentifier
		ParameterDocumentationLines []string
	}
	BLangMarkdownReturnParameterDocumentation struct {
		bLangExpressionBase
		ReturnParameterDocumentationLines []string
		ReturnType                        *BLangValueType
	}
	BLangMarkDownDeprecationDocumentation struct {
		bLangExpressionBase
		DeprecationDocumentationLines []string
		DeprecationLines              []string
		IsCorrectDeprecationLine      bool
	}
	BLangMarkDownDeprecatedParametersDocumentation struct {
		bLangExpressionBase
		Parameters []BLangMarkdownParameterDocumentation
	}
	bLangExpressionBase struct {
		bLangNodeBase
	}
)

// AbstractExpression expression is there to allow other packages (such as Desugar) to define their
// own ast nodes. All stages after that will need to be aware on how to handle them.
type AbstractExpression = bLangExpressionBase

func (*bLangExpressionBase) actionOrExpression() {}
func (*bLangExpressionBase) expressionNode()     {}

func (b *BLangValueExpressionBase) IsCompoundAssignmentLValue() bool {
	return b.flags.Has(valueExpressionFlagCompoundAssignmentLValue)
}

func (b *BLangValueExpressionBase) SetCompoundAssignmentLValue() {
	b.flags |= valueExpressionFlagCompoundAssignmentLValue
}

func (b *BLangValueExpressionBase) IsLexpr() bool {
	return b.flags.Has(valueExpressionFlagLexpr)
}

func (b *BLangValueExpressionBase) SetLexpr() {
	b.flags |= valueExpressionFlagLexpr
}

func (b *BLangValueExpressionBase) IsOptionalAccess() bool {
	return b.flags.Has(valueExpressionFlagOptionalAccess)
}

func (b *BLangValueExpressionBase) SetOptionalAccess() {
	b.flags |= valueExpressionFlagOptionalAccess
}

func (f BLangValueExpressionFlags) Has(flag BLangValueExpressionFlags) bool {
	return f&flag == flag
}

func (*BLangRemoteMethodCallAction) actionNode()         {}
func (*BLangRemoteMethodCallAction) actionOrExpression() {}

func (*BLangClientResourceAccessAction) actionNode()         {}
func (*BLangClientResourceAccessAction) actionOrExpression() {}

type ResourceAccessSegmentKind uint8

const (
	ResourceAccessSegmentName ResourceAccessSegmentKind = iota
	ResourceAccessSegmentComputed
)

type MappingKeyKind uint8

const (
	MappingKeyStringLiteral MappingKeyKind = iota
	MappingKeyIdentifier
	MappingKeyComputed
)

const (
	valueExpressionFlagCompoundAssignmentLValue BLangValueExpressionFlags = 1 << iota
	valueExpressionFlagLexpr
	valueExpressionFlagOptionalAccess
)

type TemplateExprKind uint8

type XMLTemplateInsertionKind uint8

const (
	TemplateExprKindString TemplateExprKind = iota
	TemplateExprKindXML
	TemplateExprKindRaw
)

const (
	XMLTemplateInsertionKindContent XMLTemplateInsertionKind = iota
	XMLTemplateInsertionKindAttribute
)

type (
	NarrowedTypes struct {
		TrueType  BType
		FalseType BType
	}

	BLangTypeConversionExpr struct {
		bLangExpressionBase
		Expression     BLangExpression
		TypeDescriptor BType
	}

	BLangValueExpressionFlags byte

	BLangValueExpressionBase struct {
		bLangExpressionBase
		flags BLangValueExpressionFlags
	}

	bLangAccessExpressionBase struct {
		BLangValueExpressionBase
		Expr         BLangExpression
		OriginalType BType
	}

	BLangFieldBaseAccess struct {
		bLangAccessExpressionBase
		Field IdentifierNode
		// I think this need a symbol to got to the field definition in type but Expr could be non atomic and
		// this should still work
	}

	BLangAlternateWorkerReceive struct {
		bLangExpressionBase
		workerReceives []BLangWorkerReceive
	}

	BLangAnnotAccessExpr struct {
		bLangExpressionBase
		Expr           BLangExpression
		PkgAlias       IdentifierNode
		AnnotationName IdentifierNode
		symbol         model.SymbolRef
	}

	BLangArrowFunction struct {
		bLangExpressionBase
		Params       []BLangSimpleVariable
		FunctionName *BLangIdentifier
		Body         *BLangExprFunctionBody
		FuncType     BType
	}

	BLangLambdaFunction struct {
		bLangExpressionBase
		Function *BLangFunction
	}

	BLangBinaryExpr struct {
		bLangExpressionBase
		LhsExpr BLangExpression
		RhsExpr BLangExpression
		OpKind  model.OperatorKind
	}
	BLangQueryExpr struct {
		bLangExpressionBase
		QueryClauseList    []BLangNode
		QueryConstructType TypeKind
	}

	BLangCheckedExpr struct {
		bLangExpressionBase
		Expr BLangActionOrExpression
	}

	BLangCheckPanickedExpr struct {
		BLangCheckedExpr
	}

	BLangTrapExpr struct {
		bLangExpressionBase
		Expr BLangExpression
	}

	BLangCommitExpr struct {
		bLangExpressionBase
	}
	BLangVariableReferenceBase struct {
		BLangValueExpressionBase
		symbol model.SymbolRef
	}

	BLangSimpleVarRef struct {
		BLangVariableReferenceBase
		PkgAlias     IdentifierNode
		VariableName IdentifierNode
	}

	BLangLocalVarRef struct {
		BLangSimpleVarRef
	}

	BLangConstRef struct {
		BLangSimpleVarRef
		Value         any
		OriginalValue string
	}
	BLangLiteral struct {
		bLangExpressionBase
		valueType     BType
		Value         any
		OriginalValue string
		IsConstant    bool
	}

	BLangNumericLiteral struct {
		BLangLiteral
		Kind NodeKind
	}
	BLangElvisExpr struct {
		bLangExpressionBase
		LhsExpr BLangExpression
		RhsExpr BLangExpression
	}

	BLangWorkerSendReceiveExprBase struct {
		bLangExpressionBase
		WorkerType       BType
		WorkerIdentifier *BLangIdentifier
		Channel          *Channel
	}

	BLangWorkerReceive struct {
		BLangWorkerSendReceiveExprBase
		Send               WorkerSendExpressionNode
		MatchingSendsError BType
	}

	BLangWorkerSendExprBase struct {
		BLangWorkerSendReceiveExprBase
		Expr                     BLangExpression
		Receive                  *BLangWorkerReceive
		SendType                 BType
		SendTypeWithNoMsgIgnored BType
		NoMessagePossible        bool
	}

	bLangInvocationBase struct {
		Name IdentifierNode
		// RawSymbol holds either a *model.SymbolRef (resolved) or a *deferredMethodSymbol (unresolved).
		// Access via Symbol() after type resolution, or directly for deferred-symbol checks.
		RawSymbol    model.Symbol
		Expr         BLangExpression // receiver (nil for standalone function calls)
		ArgExprs     []BLangExpression
		RequiredArgs []BLangExpression
		RestArgs     []BLangExpression
	}

	BLangInvocation struct {
		bLangExpressionBase
		bLangInvocationBase
		PkgAlias IdentifierNode
		Async    bool
	}

	BLangRemoteMethodCallAction struct {
		bLangNodeBase
		bLangInvocationBase
	}

	BLangResourceAccessSegment struct {
		bLangNodeBase
		Kind ResourceAccessSegmentKind
		Name string
		Expr BLangExpression
	}

	BLangClientResourceAccessAction struct {
		bLangNodeBase
		bLangInvocationBase
		Path       []BLangResourceAccessSegment
		MethodName string
	}

	BLangGroupExpr struct {
		bLangExpressionBase
		Expression BLangExpression
	}

	BLangTypedescExpr struct {
		bLangExpressionBase
		typeDescriptor TypeDescriptor
		// Constraint is the semtype of the type this typedesc denotes — the T in
		// typedesc<T>. BIR lowers the expression to a TypeDesc{Type: Constraint}
		// constant.
		Constraint       semtypes.SemType
		AnnotationValues values.AnnotationValues
	}

	BLangInferredTypedescDefault struct {
		bLangExpressionBase
	}

	BLangUnaryExpr struct {
		bLangExpressionBase
		Expr     BLangExpression
		Operator model.OperatorKind
	}

	BLangIndexBasedAccess struct {
		bLangAccessExpressionBase
		IndexExpr BLangExpression
	}

	BLangListConstructorExpr struct {
		bLangExpressionBase
		Exprs         []BLangExpression
		AtomicType    semtypes.ListAtomicType
		SpreadMembers []bool
	}

	BLangErrorConstructorExpr struct {
		bLangExpressionBase
		ErrorTypeRef   *BLangUserDefinedType
		PositionalArgs []BLangExpression
		NamedArgs      []BLangNamedArgsExpression
	}

	BLangTypeTestExpr struct {
		bLangExpressionBase
		Expr       BLangExpression
		Type       TypeData
		isNegation bool
	}

	BLangMappingKey struct {
		bLangNodeBase
		Expr BLangExpression
		Kind MappingKeyKind
	}

	BLangMappingKeyValueField struct {
		bLangNodeBase
		Key       *BLangMappingKey
		ValueExpr BLangExpression
		Readonly  bool
	}

	BLangMappingConstructorExpr struct {
		bLangExpressionBase
		Fields        []MappingField
		AtomicType    semtypes.MappingAtomicType
		FieldDefaults []model.FieldDefault
	}

	BLangNamedArgsExpression struct {
		bLangExpressionBase
		Name IdentifierNode
		Expr BLangExpression
		// JBallerina has symbols for these as well. Need to think if we need them as well (for go to definition)
	}

	BLangNewExpression struct {
		bLangExpressionBase
		AtomicType     *semtypes.MappingAtomicType
		ClassSymbol    model.SymbolRef
		TypeDescriptor BType
		ArgsExprs      []BLangExpression
	}

	BLangXMLSequenceLiteral struct {
		bLangExpressionBase
		// PR-TODO: this should by a slice of XML stuff
		Children []BLangExpression
	}

	BLangTemplateExpr struct {
		bLangExpressionBase
		Kind       TemplateExprKind
		Strings    []string
		Insertions []BLangExpression
	}

	XMLTemplateNamespaceInsertion struct {
		Offset         int // this is the offest in the string where we need to insert the namespace declarations when we desugar to BLangTemplateExpr
		UsedPrefixes   map[string]struct{}
		NeedsDefaultNS bool
		Namespaces     []model.SymbolRef // Namespaces referred from this node
	}

	BLangXMLTemplateExpr struct {
		BLangTemplateExpr
		InsertionKinds      []XMLTemplateInsertionKind        // This tracks where do we do the insertion for each expression
		NamespaceInsertions [][]XMLTemplateNamespaceInsertion // namespace insertion points for each template string
	}

	BLangXMLElementLiteral struct {
		bLangExpressionBase
		Name       string
		Attrs      []BLangXMLAttribute
		Content    BLangExpression
		Namespaces []model.SymbolRef // Namespaces referred from this node
	}

	BLangXMLAttribute struct {
		bLangExpressionBase
		Name  string
		Value BLangExpression
	}

	BLangXMLPILiteral struct {
		bLangExpressionBase
		Target string
		Data   string
	}

	BLangXMLCommentLiteral struct {
		bLangExpressionBase
		Body string
	}

	BLangXMLTextLiteral struct {
		bLangExpressionBase
		Body string
	}
)

var (
	_ BinaryExpressionNode                                   = &BLangBinaryExpr{}
	_ QueryExpressionNode                                    = &BLangQueryExpr{}
	_ SimpleVariableReferenceNode                            = &BLangSimpleVarRef{}
	_ SimpleVariableReferenceNode                            = &BLangLocalVarRef{}
	_ LiteralNode                                            = &BLangConstRef{}
	_ LiteralNode                                            = &BLangLiteral{}
	_ BLangExpression                                        = &BLangLiteral{}
	_ MappingVarNameFieldNode                                = &BLangConstRef{}
	_ ElvisExpressionNode                                    = &BLangElvisExpr{}
	_ MarkdownDocumentationTextAttributeNode                 = &BLangMarkdownDocumentationLine{}
	_ MarkdownDocumentationParameterAttributeNode            = &BLangMarkdownParameterDocumentation{}
	_ MarkdownDocumentationReturnParameterAttributeNode      = &BLangMarkdownReturnParameterDocumentation{}
	_ MarkDownDocumentationDeprecationAttributeNode          = &BLangMarkDownDeprecationDocumentation{}
	_ MarkDownDocumentationDeprecatedParametersAttributeNode = &BLangMarkDownDeprecatedParametersDocumentation{}
	_ WorkerReceiveNode                                      = &BLangWorkerReceive{}
	_ LambdaFunctionNode                                     = &BLangLambdaFunction{}
	_ InvocationNode                                         = &BLangInvocation{}
	_ BLangExpression                                        = &BLangInvocation{}
	_ BLangAction                                            = &BLangRemoteMethodCallAction{}
	_ BLangAction                                            = &BLangClientResourceAccessAction{}
	_ BLangExpression                                        = &BLangQueryExpr{}
	_ GroupExpressionNode                                    = &BLangGroupExpr{}
	_ TypedescExpressionNode                                 = &BLangTypedescExpr{}
	_ LiteralNode                                            = &BLangNumericLiteral{}
	_ UnaryExpressionNode                                    = &BLangUnaryExpr{}
	_ IndexBasedAccessNode                                   = &BLangIndexBasedAccess{}
	_ ListConstructorExprNode                                = &BLangListConstructorExpr{}
	_ ErrorConstructorExpressionNode                         = &BLangErrorConstructorExpr{}
	_ TypeConversionNode                                     = &BLangTypeConversionExpr{}
	_ BLangExpression                                        = &BLangTypeConversionExpr{}
	_ BLangExpression                                        = &BLangErrorConstructorExpr{}
	_ BLangNode                                              = &BLangErrorConstructorExpr{}
	_ BLangExpression                                        = &BLangTypeTestExpr{}
	_ TypeTestExpressionNode                                 = &BLangTypeTestExpr{}
	_ MappingConstructor                                     = &BLangMappingConstructorExpr{}
	_ MappingKeyValueFieldNode                               = &BLangMappingKeyValueField{}
	_ BLangExpression                                        = &BLangMappingConstructorExpr{}
	_ BLangNode                                              = &BLangMappingConstructorExpr{}
	_ BLangExpression                                        = &BLangNamedArgsExpression{}
	_ NamedArgNode                                           = &BLangNamedArgsExpression{}
	_ TrapNode                                               = &BLangTrapExpr{}
	_ BLangExpression                                        = &BLangTrapExpr{}
	_ BLangExpression                                        = &BLangNewExpression{}
)

var (
	_ BLangNode       = &BLangTypeConversionExpr{}
	_ BLangNode       = &BLangAlternateWorkerReceive{}
	_ BLangNode       = &BLangAnnotAccessExpr{}
	_ BLangNode       = &BLangArrowFunction{}
	_ BLangNode       = &BLangLambdaFunction{}
	_ BLangExpression = &BLangLambdaFunction{}
	_ BLangNode       = &BLangBinaryExpr{}
	_ BLangNode       = &BLangQueryExpr{}
	_ BLangNode       = &BLangCheckedExpr{}
	_ BLangNode       = &BLangCheckPanickedExpr{}
	_ BLangNode       = &BLangCommitExpr{}
	_ BLangNode       = &BLangSimpleVarRef{}
	_ BLangNode       = &BLangLocalVarRef{}
	_ BLangNode       = &BLangConstRef{}
	_ BLangNode       = &BLangLiteral{}
	_ BLangNode       = &BLangNumericLiteral{}
	_ BLangNode       = &BLangElvisExpr{}
	_ BLangNode       = &BLangWorkerReceive{}
	_ BLangNode       = &BLangInvocation{}
	_ BLangNode       = &BLangMarkdownDocumentationLine{}
	_ BLangNode       = &BLangMarkdownParameterDocumentation{}
	_ BLangNode       = &BLangMarkdownReturnParameterDocumentation{}
	_ BLangNode       = &BLangMarkDownDeprecationDocumentation{}
	_ BLangNode       = &BLangMarkDownDeprecatedParametersDocumentation{}
	_ BLangNode       = &BLangGroupExpr{}
	_ BLangNode       = &BLangTypedescExpr{}
	_ BLangNode       = &BLangInferredTypedescDefault{}
	_ BLangExpression = &BLangInferredTypedescDefault{}
	_ BLangNode       = &BLangIndexBasedAccess{}
	_ BLangNode       = &BLangListConstructorExpr{}
	_ BLangNode       = &BLangTypeConversionExpr{}
	_ BLangNode       = &BLangMappingConstructorExpr{}
	_ BLangNode       = &BLangMappingKeyValueField{}
	_ BLangNode       = &BLangTrapExpr{}
	_ BLangNode       = &BLangNewExpression{}
)

var (
	// Assert that concrete types with symbols implement BNodeWithSymbol
	_ BNodeWithSymbol = &BLangSimpleVarRef{}
	_ BNodeWithSymbol = &BLangLocalVarRef{}
	_ BNodeWithSymbol = &BLangConstRef{}
	_ BNodeWithSymbol = &BLangAnnotAccessExpr{}
	_ BNodeWithSymbol = &BLangInvocation{}
)

// Symbol methods for BNodeWithSymbol interface

func (*BLangVariableReferenceBase) isVariableReference() {}

func (*BLangSimpleVarRef) isLExpr()         {}
func (*bLangAccessExpressionBase) isLExpr() {}

func (*BLangCommitExpr) isAction()    {}
func (*BLangWorkerReceive) isAction() {}

func (n *BLangVariableReferenceBase) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangVariableReferenceBase) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

// Symbol returns the resolved SymbolRef for this invocation.
// Panics if the symbol has not been resolved yet (i.e. is a deferred method symbol).
// Only call this after type resolution.
func (n *BLangInvocation) Symbol() model.SymbolRef {
	return *n.RawSymbol.(*model.SymbolRef)
}

func (n *BLangInvocation) SetSymbol(symbolRef model.SymbolRef) {
	n.RawSymbol = &symbolRef
}

func (n *BLangAnnotAccessExpr) Symbol() model.SymbolRef {
	return n.symbol
}

func (n *BLangAnnotAccessExpr) SetSymbol(symbolRef model.SymbolRef) {
	n.symbol = symbolRef
}

func (n *BLangRemoteMethodCallAction) MethodSymbol() model.SymbolRef {
	return *n.RawSymbol.(*model.SymbolRef)
}

func (n *BLangRemoteMethodCallAction) SetMethodSymbol(symbolRef model.SymbolRef) {
	n.RawSymbol = &symbolRef
}

func (n *BLangClientResourceAccessAction) MethodSymbol() model.SymbolRef {
	return *n.RawSymbol.(*model.SymbolRef)
}

func (n *BLangClientResourceAccessAction) SetMethodSymbol(symbolRef model.SymbolRef) {
	n.RawSymbol = &symbolRef
}

func (b *BLangGroupExpr) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangTypedescExpr) GetTypeDescriptor() TypeDescriptor {
	return b.typeDescriptor
}

func (b *BLangTypedescExpr) SetTypeDescriptor(typeDescriptor TypeDescriptor) {
	if typeDescriptor == nil {
		b.typeDescriptor = nil
		return
	}
	b.typeDescriptor = typeDescriptor.(BType)
}

func (b *BLangLiteral) GetValueType() BType {
	return b.valueType
}

func (b *BLangLiteral) SetValueType(bt BType) {
	b.valueType = bt
}

func (b *BLangLambdaFunction) GetFunctionNode() FunctionNode {
	return b.Function
}

func (b *BLangLambdaFunction) SetFunctionNode(functionNode FunctionNode) {
	if fn, ok := functionNode.(*BLangFunction); ok {
		b.Function = fn
	} else {
		panic("functionNode is not a BLangFunction")
	}
}

func (b *BLangAlternateWorkerReceive) ToActionString() string {
	panic("Not implemented")
}

func (b *BLangWorkerReceive) GetWorkerName() *BLangIdentifier {
	return b.WorkerIdentifier
}

func (b *BLangWorkerReceive) SetWorkerName(identifierNode *BLangIdentifier) {
	b.WorkerIdentifier = identifierNode
}

func (b *BLangWorkerReceive) ToActionString() string {
	if b.WorkerIdentifier != nil {
		return fmt.Sprintf(" <- %s", b.WorkerIdentifier.Value)
	}
	return " <- "
}

func (b *BLangBinaryExpr) GetLeftExpression() BLangExpression {
	return b.LhsExpr
}

func (b *BLangBinaryExpr) GetRightExpression() BLangExpression {
	return b.RhsExpr
}

func (b *BLangBinaryExpr) GetOperatorKind() model.OperatorKind {
	return b.OpKind
}

func (b *BLangQueryExpr) GetQueryClauses() []Node {
	result := make([]Node, len(b.QueryClauseList))
	for i := range b.QueryClauseList {
		result[i] = b.QueryClauseList[i]
	}
	return result
}

func (b *BLangQueryExpr) AddQueryClause(queryClause Node) {
	if node, ok := queryClause.(BLangNode); ok {
		b.QueryClauseList = append(b.QueryClauseList, node)
		return
	}
	panic("query clause is not a BLangNode")
}

func (b *BLangCheckedExpr) GetExpression() BLangActionOrExpression {
	return b.Expr
}

func (b *BLangCheckedExpr) GetOperatorKind() model.OperatorKind {
	return model.OperatorKind_CHECK
}

func (b *BLangCheckPanickedExpr) GetOperatorKind() model.OperatorKind {
	return model.OperatorKind_CHECK_PANIC
}

func (b *BLangSimpleVarRef) GetPackageAlias() IdentifierNode {
	return b.PkgAlias
}

func (b *BLangSimpleVarRef) GetVariableName() IdentifierNode {
	return b.VariableName
}

func (b *BLangConstRef) GetValue() any {
	return b.Value
}

func (b *BLangConstRef) SetValue(value any) {
	b.Value = value
}

func (b *BLangConstRef) GetIsConstant() bool {
	return true
}

func (b *BLangConstRef) SetIsConstant(isConstant bool) {
	if !isConstant {
		panic("isConstant is not true")
	}
}

func (b *BLangConstRef) GetOriginalValue() string {
	return b.OriginalValue
}

func (b *BLangConstRef) SetOriginalValue(originalValue string) {
	b.OriginalValue = originalValue
}

func (b *BLangConstRef) IsKeyValueField() bool {
	return false
}

func (b *BLangLiteral) GetValue() any {
	return b.Value
}

func (b *BLangLiteral) GetIsConstant() bool {
	return b.IsConstant
}

func (b *BLangLiteral) SetIsConstant(isConstant bool) {
	b.IsConstant = isConstant
}

func (b *BLangLiteral) SetValue(value any) {
	b.Value = value
}

func (b *BLangLiteral) GetOriginalValue() string {
	return b.OriginalValue
}

func (b *BLangLiteral) SetOriginalValue(originalValue string) {
	b.OriginalValue = originalValue
}

func (b *BLangElvisExpr) GetLeftExpression() BLangExpression {
	return b.LhsExpr
}

func (b *BLangElvisExpr) GetRightExpression() BLangExpression {
	return b.RhsExpr
}

func (b *BLangMarkdownDocumentationLine) GetText() string {
	return b.Text
}

func (b *BLangMarkdownDocumentationLine) SetText(text string) {
	b.Text = text
}

func (b *BLangMarkdownParameterDocumentation) GetParameterName() *BLangIdentifier {
	return b.ParameterName
}

func (b *BLangMarkdownParameterDocumentation) SetParameterName(parameterName *BLangIdentifier) {
	b.ParameterName = parameterName
}

func (b *BLangMarkdownParameterDocumentation) GetParameterDocumentationLines() []string {
	return b.ParameterDocumentationLines
}

func (b *BLangMarkdownParameterDocumentation) AddParameterDocumentationLine(text string) {
	b.ParameterDocumentationLines = append(b.ParameterDocumentationLines, text)
}

func (b *BLangMarkdownParameterDocumentation) GetParameterDocumentation() string {
	return strings.ReplaceAll(strings.Join(b.ParameterDocumentationLines, "\n"), "\r", "")
}

func (b *BLangMarkdownReturnParameterDocumentation) GetReturnParameterDocumentationLines() []string {
	return b.ReturnParameterDocumentationLines
}

func (b *BLangMarkdownReturnParameterDocumentation) AddReturnParameterDocumentationLine(text string) {
	b.ReturnParameterDocumentationLines = append(b.ReturnParameterDocumentationLines, text)
}

func (b *BLangMarkdownReturnParameterDocumentation) GetReturnParameterDocumentation() string {
	return strings.ReplaceAll(strings.Join(b.ReturnParameterDocumentationLines, "\n"), "\r", "")
}

func (b *BLangMarkdownReturnParameterDocumentation) GetReturnType() *BLangValueType {
	return b.ReturnType
}

func (b *BLangMarkdownReturnParameterDocumentation) SetReturnType(ty *BLangValueType) {
	b.ReturnType = ty
}

func (b *BLangMarkDownDeprecationDocumentation) AddDeprecationDocumentationLine(text string) {
	b.DeprecationDocumentationLines = append(b.DeprecationDocumentationLines, text)
}

func (b *BLangMarkDownDeprecationDocumentation) AddDeprecationLine(text string) {
	b.DeprecationLines = append(b.DeprecationLines, text)
}

func (b *BLangMarkDownDeprecationDocumentation) GetDocumentation() string {
	return strings.ReplaceAll(strings.Join(b.DeprecationDocumentationLines, "\n"), "\r", "")
}

func (b *BLangMarkDownDeprecatedParametersDocumentation) AddParameter(parameter MarkdownDocumentationParameterAttributeNode) {
	if param, ok := parameter.(*BLangMarkdownParameterDocumentation); ok {
		b.Parameters = append(b.Parameters, *param)
	} else {
		panic("parameter is not a BLangMarkdownParameterDocumentation")
	}
}

func (b *BLangMarkDownDeprecatedParametersDocumentation) GetParameters() []MarkdownDocumentationParameterAttributeNode {
	result := make([]MarkdownDocumentationParameterAttributeNode, len(b.Parameters))
	for i := range b.Parameters {
		result[i] = &b.Parameters[i]
	}
	return result
}

func (b *BLangWorkerSendExprBase) GetExpr() BLangExpression {
	return b.Expr
}

func (b *BLangWorkerSendExprBase) GetWorkerName() *BLangIdentifier {
	return b.WorkerIdentifier
}

func (b *BLangWorkerSendExprBase) SetWorkerName(identifierNode *BLangIdentifier) {
	b.WorkerIdentifier = identifierNode
}

func (b *bLangInvocationBase) SetRawSymbol(symbol model.Symbol) {
	b.RawSymbol = symbol
}

func (b *bLangInvocationBase) GetName() IdentifierNode {
	return b.Name
}

func (b *bLangInvocationBase) GetArgumentExpressions() []BLangExpression {
	result := make([]BLangExpression, len(b.ArgExprs))
	copy(result, b.ArgExprs)
	return result
}

func (b *bLangInvocationBase) GetRequiredArgs() []BLangExpression {
	result := make([]BLangExpression, len(b.RequiredArgs))
	copy(result, b.RequiredArgs)
	return result
}

func (b *bLangInvocationBase) GetExpression() BLangExpression {
	return b.Expr
}

func (n *bLangInvocationBase) ResolvedSymbol() model.SymbolRef {
	return *n.RawSymbol.(*model.SymbolRef)
}
func (n *bLangInvocationBase) SetResolvedSymbol(ref model.SymbolRef) { n.RawSymbol = &ref }
func (n *bLangInvocationBase) Receiver() BLangExpression             { return n.Expr }
func (n *bLangInvocationBase) SetReceiver(expr BLangExpression)      { n.Expr = expr }
func (n *bLangInvocationBase) CallArgs() []BLangExpression           { return n.ArgExprs }
func (n *bLangInvocationBase) SetCallArgs(args []BLangExpression)    { n.ArgExprs = args }

func (b *BLangInvocation) GetPackageAlias() IdentifierNode {
	return b.PkgAlias
}

func (b *BLangTypeConversionExpr) GetExpression() BLangExpression {
	return b.Expression
}

func (b *BLangTypeConversionExpr) SetExpression(expression BLangExpression) {
	b.Expression = expression
}

func (b *BLangTypeConversionExpr) GetTypeDescriptor() TypeDescriptor {
	if b.TypeDescriptor == nil {
		return nil
	}
	return b.TypeDescriptor
}

func (b *BLangTypeConversionExpr) SetTypeDescriptor(typeDescriptor TypeDescriptor) {
	if typeDescriptor == nil {
		b.TypeDescriptor = nil
		return
	}
	b.TypeDescriptor = typeDescriptor.(BType)
}

func (b *BLangTypeConversionExpr) IsPublic() bool {
	return false
}

func (b *BLangTypeConversionExpr) GetAnnotationAttachments() []AnnotationAttachmentNode {
	panic("not implemented")
}

func (b *BLangTypeConversionExpr) AddAnnotationAttachment(annAttachment AnnotationAttachmentNode) {
	panic("not implemented")
}

func (b *BLangUnaryExpr) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangUnaryExpr) GetOperatorKind() model.OperatorKind {
	return b.Operator
}

func (b *BLangIndexBasedAccess) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangIndexBasedAccess) GetIndex() BLangExpression {
	return b.IndexExpr
}

func (b *BLangFieldBaseAccess) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangFieldBaseAccess) GetFieldName() IdentifierNode {
	return b.Field
}

func (b *BLangListConstructorExpr) GetExpressions() []BLangExpression {
	result := make([]BLangExpression, len(b.Exprs))
	copy(result, b.Exprs)
	return result
}

func (b *BLangListConstructorExpr) SetSpreadMember(index int) {
	if len(b.SpreadMembers) != len(b.Exprs) {
		b.SpreadMembers = make([]bool, len(b.Exprs))
	}
	b.SpreadMembers[index] = true
}

func (b *BLangListConstructorExpr) IsSpreadMember(index int) bool {
	return index >= 0 && index < len(b.SpreadMembers) && b.SpreadMembers[index]
}

func (b *BLangListConstructorExpr) HasSpreadMembers() bool {
	for _, isSpread := range b.SpreadMembers {
		if isSpread {
			return true
		}
	}
	return false
}

func (b *BLangErrorConstructorExpr) GetPositionalArgs() []BLangExpression {
	result := make([]BLangExpression, len(b.PositionalArgs))
	copy(result, b.PositionalArgs)
	return result
}

func (b *BLangErrorConstructorExpr) GetNamedArgs() []NamedArgNode {
	result := make([]NamedArgNode, len(b.NamedArgs))
	for i := range b.NamedArgs {
		result[i] = &b.NamedArgs[i]
	}
	return result
}

func (b *BLangTypeTestExpr) IsNegation() bool {
	return b.isNegation
}

func (b *BLangTypeTestExpr) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangTypeTestExpr) GetType() TypeData {
	return b.Type
}

func (b *BLangMappingKeyValueField) GetKey() BLangExpression {
	if b.Key == nil {
		return nil
	}
	return b.Key.Expr
}

func (b *BLangMappingKeyValueField) GetValue() BLangExpression {
	return b.ValueExpr
}

func (b *BLangMappingKeyValueField) IsKeyValueField() bool {
	return true
}

func (b *BLangMappingConstructorExpr) GetFields() []MappingField {
	return b.Fields
}

func (b *BLangNamedArgsExpression) SetName(name IdentifierNode) {
	b.Name = name
}

func (b *BLangNamedArgsExpression) GetName() IdentifierNode {
	return b.Name
}

func (b *BLangNamedArgsExpression) GetExpression() BLangExpression {
	return b.Expr
}

func (b *BLangNamedArgsExpression) SetExpression(expr BLangExpression) {
	b.Expr = expr
}

func (b *BLangTrapExpr) GetExpression() BLangExpression {
	return b.Expr
}

// IsStreamOperation use to distinguish stream operations from method calls.
func IsStreamOperation(inv interface{ Receiver() BLangExpression }) bool {
	recv := inv.Receiver()
	return recv != nil && semtypes.IsSubtypeSimple(recv.GetDeterminedType(), semtypes.STREAM)
}

// IsStreamNewExpression returns true when the new expression constructs a stream value.
func IsStreamNewExpression(expr *BLangNewExpression) bool {
	return semtypes.IsSubtypeSimple(expr.GetDeterminedType(), semtypes.STREAM)
}

func createBLangUnaryExpr(location diagnostics.Location, operator model.OperatorKind, expr BLangExpression) *BLangUnaryExpr {
	exprNode := &BLangUnaryExpr{}
	exprNode.pos = location
	exprNode.Expr = expr
	exprNode.Operator = operator
	return exprNode
}

var (
	_ BLangExpression = &BLangXMLSequenceLiteral{}
	_ BLangExpression = &BLangTemplateExpr{}
	_ BLangExpression = &BLangXMLTemplateExpr{}
	_ BLangExpression = &BLangXMLElementLiteral{}
	_ BLangExpression = &BLangXMLAttribute{}
	_ BLangExpression = &BLangXMLPILiteral{}
	_ BLangExpression = &BLangXMLCommentLiteral{}
	_ BLangExpression = &BLangXMLTextLiteral{}
	_ BLangNode       = &BLangXMLSequenceLiteral{}
	_ BLangNode       = &BLangTemplateExpr{}
	_ BLangNode       = &BLangXMLTemplateExpr{}
	_ BLangNode       = &BLangXMLElementLiteral{}
	_ BLangNode       = &BLangXMLAttribute{}
	_ BLangNode       = &BLangXMLPILiteral{}
	_ BLangNode       = &BLangXMLCommentLiteral{}
	_ BLangNode       = &BLangXMLTextLiteral{}
)
