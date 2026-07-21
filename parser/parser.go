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
package parser

import (
	"slices"
	"strings"

	debugcommon "ballerina-lang-go/common"
	"ballerina-lang-go/context"
	"ballerina-lang-go/parser/common"
	tree "ballerina-lang-go/parser/tree"
	"ballerina-lang-go/tools/diagnostics"
	"ballerina-lang-go/tools/text"
)

type OperatorPrecedence uint8

const (
	OPERATOR_PRECEDENCE_MEMBER_ACCESS     OperatorPrecedence = iota //  x.k, x.@a, f(x), x.f(y), x[y], x?.k, x.<y>, x/<y>, x/**/<y>, x/*xml-step-extend
	OPERATOR_PRECEDENCE_UNARY                                       //  (+x), (-x), (~x), (!x), (<T>x), (typeof x),
	OPERATOR_PRECEDENCE_EXPRESSION_ACTION                           //  Expression that can also be an action. eg: (check x), (checkpanic x). Same as unary.
	OPERATOR_PRECEDENCE_MULTIPLICATIVE                              //  (x * y), (x / y), (x % y)
	OPERATOR_PRECEDENCE_ADDITIVE                                    //  (x + y), (x - y)
	OPERATOR_PRECEDENCE_SHIFT                                       //  (x << y), (x >> y), (x >>> y)
	OPERATOR_PRECEDENCE_RANGE                                       //  (x ... y), (x ..< y)
	OPERATOR_PRECEDENCE_BINARY_COMPARE                              //  (x < y), (x > y), (x <= y), (x >= y), (x is y)
	OPERATOR_PRECEDENCE_EQUALITY                                    //  (x == y), (x != y), (x == y), (x === y), (x !== y)
	OPERATOR_PRECEDENCE_BITWISE_AND                                 //  (x & y)
	OPERATOR_PRECEDENCE_BITWISE_XOR                                 //  (x ^ y)
	OPERATOR_PRECEDENCE_BITWISE_OR                                  //  (x | y)
	OPERATOR_PRECEDENCE_LOGICAL_AND                                 //  (x && y)
	OPERATOR_PRECEDENCE_LOGICAL_OR                                  //  (x || y)
	OPERATOR_PRECEDENCE_ELVIS_CONDITIONAL                           //  x ?: y
	OPERATOR_PRECEDENCE_CONDITIONAL                                 //  x ? y : z

	OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET //  (x) => y

	//  Actions cannot reside inside expressions (excluding query-action-or-expr), hence they have the lowest
	//  precedence.
	OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION //  (x -> y()),
	OPERATOR_PRECEDENCE_ACTION             //  (start x), ...
	OPERATOR_PRECEDENCE_TRAP               //  (trap x)

	// A query-action-or-expr or a query-action can have actions in certain clauses.
	OPERATOR_PRECEDENCE_QUERY //  from x, select x, where x

	OPERATOR_PRECEDENCE_DEFAULT //  (start x), ...
)

const DEFAULT_OP_PRECEDENCE OperatorPrecedence = OPERATOR_PRECEDENCE_DEFAULT

func (o *OperatorPrecedence) isHigherThanOrEqual(other OperatorPrecedence, allowActions bool) bool {
	if allowActions {
		if (*o == OPERATOR_PRECEDENCE_EXPRESSION_ACTION) && (other == OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION) {
			return false
		}
	}
	return uint8(*o) <= uint8(other)
}

type TypePrecedence uint8

func (t *TypePrecedence) isHigherThanOrEqual(other TypePrecedence) bool {
	return uint8(*t) <= uint8(other)
}

const (
	TYPE_PRECEDENCE_DISTINCT          TypePrecedence = iota // distinct T
	TYPE_PRECEDENCE_ARRAY_OR_OPTIONAL                       // T[], T?
	TYPE_PRECEDENCE_INTERSECTION                            // T1 & T2
	TYPE_PRECEDENCE_UNION                                   // T1 | T2
	TYPE_PRECEDENCE_DEFAULT                                 // function(args) returns T
)

type Action uint8

const (
	ACTION_INSERT Action = iota
	ACTION_REMOVE
	ACTION_KEEP
)

type ParserErrorHandler interface {
	SwitchContext(context common.ParserRuleContext)
	GetParentContext() common.ParserRuleContext
	EndContext()
	StartContext(context common.ParserRuleContext)
	Recover(currentCtx common.ParserRuleContext, token tree.STToken, isCompletion bool) *Solution
	GetContextStack() []common.ParserRuleContext
	GetGrandParentContext() common.ParserRuleContext
	ConsumeInvalidToken() tree.STToken
}

type invalidNodeInfo struct {
	node           tree.STNode
	diagnosticCode diagnostics.DiagnosticCode
	args           []any
}

type abstractParser struct {
	errorHandler         ParserErrorHandler
	tokenReader          *TokenReader
	invalidNodeInfoStack []invalidNodeInfo
	insertedToken        tree.STToken
}

func NewInvalidNodeInfoFromInvalidNodeDiagnosticCodeArgs(invalidNode tree.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) invalidNodeInfo {
	this := invalidNodeInfo{}
	this.node = invalidNode
	this.diagnosticCode = diagnosticCode
	this.args = args
	return this
}

func NewAbstractParserFromTokenReaderErrorHandler(tokenReader *TokenReader, errorHandler ParserErrorHandler) abstractParser {
	this := abstractParser{}
	this.invalidNodeInfoStack = make([]invalidNodeInfo, 0)
	this.insertedToken = nil
	// Default field initializations

	this.tokenReader = tokenReader
	this.errorHandler = errorHandler
	return this
}

func NewAbstractParserFromTokenReader(tokenReader *TokenReader) abstractParser {
	this := abstractParser{}
	this.invalidNodeInfoStack = make([]invalidNodeInfo, 0)
	this.insertedToken = nil
	// Default field initializations

	this.tokenReader = tokenReader
	this.errorHandler = nil
	return this
}

func (a *abstractParser) peek() tree.STToken {
	if a.insertedToken != nil {
		return a.insertedToken
	}
	return a.tokenReader.Peek()
}

func (a *abstractParser) peekN(n int) tree.STToken {
	if a.insertedToken == nil {
		return a.tokenReader.PeekN(n)
	}
	if n == 1 {
		return a.insertedToken
	}
	if n > 0 {
		n = (n - 1)
	}
	return a.tokenReader.PeekN(n)
}

func (a *abstractParser) consume() tree.STToken {
	if a.insertedToken != nil {
		nextToken := a.insertedToken
		a.insertedToken = nil
		return a.consumeWithInvalidNodesWithToken(nextToken)
	}
	if len(a.invalidNodeInfoStack) == 0 {
		return a.tokenReader.Read()
	}
	return a.consumeWithInvalidNodes()
}

func (a *abstractParser) consumeWithInvalidNodes() tree.STToken {
	token := a.tokenReader.Read()
	return a.consumeWithInvalidNodesWithToken(token)
}

func (a *abstractParser) consumeWithInvalidNodesWithToken(token tree.STToken) tree.STToken {
	newToken := token
	for len(a.invalidNodeInfoStack) > 0 {
		invalidNodeInfo := a.invalidNodeInfoStack[len(a.invalidNodeInfoStack)-1]
		a.invalidNodeInfoStack = a.invalidNodeInfoStack[:len(a.invalidNodeInfoStack)-1]
		newToken = tree.ToToken(tree.CloneWithLeadingInvalidNodeMinutiae(newToken, invalidNodeInfo.node,
			invalidNodeInfo.diagnosticCode, invalidNodeInfo.args))
	}
	return newToken
}

func (a *abstractParser) recover(token tree.STToken, currentCtx common.ParserRuleContext, isCompletion bool) *Solution {
	isCompletion = isCompletion || token.Kind() == common.EOF_TOKEN
	sol := a.errorHandler.Recover(currentCtx, token, isCompletion)
	switch sol.Action {
	case ACTION_REMOVE:
		a.insertedToken = nil
		a.addInvalidTokenToNextToken(sol.RemovedToken)
	case ACTION_INSERT:
		a.insertedToken = tree.ToToken(sol.RecoveredNode)
	}
	return sol
}

func (a *abstractParser) insertToken(kind common.SyntaxKind, context common.ParserRuleContext) {
	a.insertedToken = tree.CreateMissingTokenWithDiagnosticsFromParserRules(kind, context)
}

func (a *abstractParser) removeInsertedToken() {
	a.insertedToken = nil
}

func (a *abstractParser) isInvalidNodeStackEmpty() bool {
	return len(a.invalidNodeInfoStack) == 0
}

func (a *abstractParser) startContext(context common.ParserRuleContext) {
	a.errorHandler.StartContext(context)
}

func (a *abstractParser) endContext() {
	a.errorHandler.EndContext()
}

func (a *abstractParser) getCurrentContext() common.ParserRuleContext {
	return a.errorHandler.GetParentContext()
}

func (a *abstractParser) switchContext(context common.ParserRuleContext) {
	a.errorHandler.SwitchContext(context)
}

func (a *abstractParser) getNextNextToken() tree.STToken {
	return a.peekN(2)
}

func (a *abstractParser) isNodeListEmpty(node tree.STNode) bool {
	nodeList, ok := node.(*tree.STNodeList)
	if !ok {
		panic("node is not a STNodeList")
	}
	return nodeList.IsEmpty()
}

func (a *abstractParser) cloneWithDiagnosticIfListEmpty(nodeList tree.STNode, target tree.STNode, diagnosticCode diagnostics.DiagnosticCode) tree.STNode {
	if a.isNodeListEmpty(nodeList) {
		return tree.AddDiagnostic(target, diagnosticCode)
	}
	return target
}

func (a *abstractParser) updateLastNodeInListWithInvalidNode(nodeList []tree.STNode, invalidParam tree.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) []tree.STNode {
	prevNode := nodeList[len(nodeList)-1]
	nodeList = nodeList[:len(nodeList)-1]
	newNode := tree.CloneWithTrailingInvalidNodeMinutiae(prevNode, invalidParam, diagnosticCode, args)
	nodeList = append(nodeList, newNode)
	return nodeList
}

func (a *abstractParser) updateFirstNodeInListWithLeadingInvalidNode(nodeList []tree.STNode, invalidParam tree.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) []tree.STNode {
	return a.updateANodeInListWithLeadingInvalidNode(nodeList, 0, invalidParam, diagnosticCode, args)
}

func (a *abstractParser) updateANodeInListWithLeadingInvalidNode(nodeList []tree.STNode, indexOfTheNode int, invalidParam tree.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) []tree.STNode {
	node := nodeList[indexOfTheNode]
	newNode := tree.CloneWithLeadingInvalidNodeMinutiae(node, invalidParam, diagnosticCode, args)
	nodeList[indexOfTheNode] = newNode
	return nodeList
}

func (a *abstractParser) invalidateRestAndAddToTrailingMinutiae(node tree.STNode) tree.STNode {
	node = a.addInvalidNodeStackToTrailingMinutiae(node)
	for a.peek().Kind() != common.EOF_TOKEN {
		invalidToken := a.consume()
		node = tree.CloneWithTrailingInvalidNodeMinutiae(node, invalidToken, &common.ERROR_INVALID_TOKEN, invalidToken.Text())
	}
	return node
}

func (a *abstractParser) addInvalidNodeStackToTrailingMinutiae(node tree.STNode) tree.STNode {
	for len(a.invalidNodeInfoStack) != 0 {
		invalidNodeInfo := a.invalidNodeInfoStack[len(a.invalidNodeInfoStack)-1]
		a.invalidNodeInfoStack = a.invalidNodeInfoStack[:len(a.invalidNodeInfoStack)-1]
		node = tree.CloneWithTrailingInvalidNodeMinutiae(node, invalidNodeInfo.node, invalidNodeInfo.diagnosticCode, invalidNodeInfo.args)
	}
	return node
}

func (a *abstractParser) addInvalidNodeToNextToken(invalidNode tree.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) {
	a.invalidNodeInfoStack = append(a.invalidNodeInfoStack, invalidNodeInfo{node: invalidNode, diagnosticCode: diagnosticCode, args: args})
}

func (a *abstractParser) addInvalidTokenToNextToken(invalidNode tree.STToken) {
	a.invalidNodeInfoStack = append(a.invalidNodeInfoStack, invalidNodeInfo{node: invalidNode, diagnosticCode: &common.ERROR_INVALID_TOKEN, args: []any{invalidNode.Text()}})
}

type BallerinaParser struct {
	abstractParser
}

func NewBallerinaParserFromTokenReader(tokenReader *TokenReader) BallerinaParser {
	this := BallerinaParser{}
	// Default field initializations

	this.abstractParser = abstractParser{
		tokenReader:          tokenReader,
		invalidNodeInfoStack: make([]invalidNodeInfo, 0),
		insertedToken:        nil,
	}
	errorHandler := NewBallerinaParserErrorHandlerFromTokenReader(this.tokenReader)
	this.errorHandler = &errorHandler
	return this
}

func isParameterizedTypeToken(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.TYPEDESC_KEYWORD, common.FUTURE_KEYWORD, common.XML_KEYWORD, common.ERROR_KEYWORD:
		return true
	default:
		return false
	}
}

func CreateBuiltinSimpleNameReference(token tree.STNode) tree.STNode {
	typeKind := getBuiltinTypeSyntaxKind(token.Kind())
	return tree.CreateBuiltinSimpleNameReferenceNode(typeKind, token)
}

func isCompoundBinaryOperator(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.SLASH_TOKEN,
		common.ASTERISK_TOKEN,
		common.BITWISE_AND_TOKEN,
		common.BITWISE_XOR_TOKEN,
		common.PIPE_TOKEN,
		common.DOUBLE_LT_TOKEN,
		common.DOUBLE_GT_TOKEN,
		common.TRIPPLE_GT_TOKEN:
		return true
	default:
		return false
	}
}

func isTypeStartingToken(nextTokenKind common.SyntaxKind, nextNextToken tree.STToken) bool {
	switch nextTokenKind {
	case common.IDENTIFIER_TOKEN,
		common.SERVICE_KEYWORD,
		common.RECORD_KEYWORD,
		common.OBJECT_KEYWORD,
		common.ABSTRACT_KEYWORD,
		common.CLIENT_KEYWORD,
		common.OPEN_PAREN_TOKEN,
		common.MAP_KEYWORD,
		common.STREAM_KEYWORD,
		common.TABLE_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.OPEN_BRACKET_TOKEN,
		common.DISTINCT_KEYWORD,
		common.ISOLATED_KEYWORD,
		common.TRANSACTIONAL_KEYWORD,
		common.TRANSACTION_KEYWORD,
		common.NATURAL_KEYWORD:
		return true
	default:
		if isParameterizedTypeToken(nextTokenKind) {
			return true
		}
		if isSingletonTypeDescStart(nextTokenKind, nextNextToken) {
			return true
		}
		return isSimpleType(nextTokenKind)
	}
}

func isSimpleType(nodeKind common.SyntaxKind) bool {
	switch nodeKind {
	case common.INT_KEYWORD,
		common.FLOAT_KEYWORD,
		common.DECIMAL_KEYWORD,
		common.BOOLEAN_KEYWORD,
		common.STRING_KEYWORD,
		common.BYTE_KEYWORD,
		common.JSON_KEYWORD,
		common.HANDLE_KEYWORD,
		common.ANY_KEYWORD,
		common.ANYDATA_KEYWORD,
		common.NEVER_KEYWORD,
		common.VAR_KEYWORD,
		common.READONLY_KEYWORD:
		return true
	default:
		return false
	}
}

func isPredeclaredPrefix(nodeKind common.SyntaxKind) bool {
	switch nodeKind {
	case common.BOOLEAN_KEYWORD,
		common.DECIMAL_KEYWORD,
		common.ERROR_KEYWORD,
		common.FLOAT_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.FUTURE_KEYWORD,
		common.INT_KEYWORD,
		common.MAP_KEYWORD,
		common.NATURAL_KEYWORD,
		common.OBJECT_KEYWORD,
		common.STREAM_KEYWORD,
		common.STRING_KEYWORD,
		common.TABLE_KEYWORD,
		common.TRANSACTION_KEYWORD,
		common.TYPEDESC_KEYWORD,
		common.XML_KEYWORD:
		return true
	default:
		return false
	}
}

func getBuiltinTypeSyntaxKind(typeKeyword common.SyntaxKind) common.SyntaxKind {
	switch typeKeyword {
	case common.INT_KEYWORD:
		return common.INT_TYPE_DESC
	case common.FLOAT_KEYWORD:
		return common.FLOAT_TYPE_DESC
	case common.DECIMAL_KEYWORD:
		return common.DECIMAL_TYPE_DESC
	case common.BOOLEAN_KEYWORD:
		return common.BOOLEAN_TYPE_DESC
	case common.STRING_KEYWORD:
		return common.STRING_TYPE_DESC
	case common.BYTE_KEYWORD:
		return common.BYTE_TYPE_DESC
	case common.JSON_KEYWORD:
		return common.JSON_TYPE_DESC
	case common.HANDLE_KEYWORD:
		return common.HANDLE_TYPE_DESC
	case common.ANY_KEYWORD:
		return common.ANY_TYPE_DESC
	case common.ANYDATA_KEYWORD:
		return common.ANYDATA_TYPE_DESC
	case common.NEVER_KEYWORD:
		return common.NEVER_TYPE_DESC
	case common.VAR_KEYWORD:
		return common.VAR_TYPE_DESC
	case common.READONLY_KEYWORD:
		return common.READONLY_TYPE_DESC
	default:
		panic(typeKeyword.StrValue() + "is not a built-in type")
	}
}

func isKeyKeyword(token tree.STToken) bool {
	return ((token.Kind() == common.IDENTIFIER_TOKEN) && KEY == token.Text())
}

func isNaturalKeyword(token tree.STToken) bool {
	return ((token.Kind() == common.IDENTIFIER_TOKEN) && NATURAL == (token.Text()))
}

func isEndOfLetVarDeclarations(nextToken tree.STToken, nextNextToken tree.STToken) bool {
	tokenKind := nextToken.Kind()
	switch tokenKind {
	case common.COMMA_TOKEN, common.AT_TOKEN:
		return false
	case common.IN_KEYWORD:
		return true
	default:
		return (isGroupOrCollectKeyword(nextToken) || (!isTypeStartingToken(tokenKind, nextNextToken)))
	}
}

func isGroupOrCollectKeyword(nextToken tree.STToken) bool {
	return (isKeywordMatch(common.COLLECT_KEYWORD, nextToken) || isKeywordMatch(common.GROUP_KEYWORD, nextToken))
}

func isKeywordMatch(syntaxKind common.SyntaxKind, token tree.STToken) bool {
	return ((token.Kind() == common.IDENTIFIER_TOKEN) && syntaxKind.StrValue() == (token.Text()))
}

func isSingletonTypeDescStart(tokenKind common.SyntaxKind, nextNextToken tree.STToken) bool {
	switch tokenKind {
	case common.STRING_LITERAL_TOKEN,
		common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.NULL_KEYWORD:
		return true
	case common.PLUS_TOKEN, common.MINUS_TOKEN:
		return isIntOrFloat(nextNextToken)
	default:
		return false
	}
}

func isIntOrFloat(token tree.STToken) bool {
	switch token.Kind() {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		return true
	default:
		return false
	}
}

func isValidBase16LiteralContent(content string) bool {
	hexDigitCount := 0
	charArray := []byte(content)
	for _, c := range charArray {
		switch c {
		case TAB,
			NEWLINE,
			CARRIAGE_RETURN,
			SPACE:
		default:
			if isHexDigit(c) {
				hexDigitCount++
			} else {
				return false
			}
		}
	}
	return ((hexDigitCount % 2) == 0)
}

func isValidBase64LiteralContent(content string) bool {
	charArray := []byte(content)
	base64CharCount := 0
	paddingCharCount := 0
	for _, c := range charArray {
		switch c {
		case TAB,
			NEWLINE,
			CARRIAGE_RETURN,
			SPACE:
		case EQUAL:
			paddingCharCount++
		default:
			if isBase64Char(c) {
				if paddingCharCount == 0 {
					base64CharCount++
				} else {
					return false
				}
			} else {
				return false
			}
		}
	}
	if paddingCharCount > 2 {
		return false
	} else if paddingCharCount == 0 {
		return ((base64CharCount % 4) == 0)
	} else {
		return base64CharCount%4 == 4-paddingCharCount
	}
}

func isBase64Char(c byte) bool {
	if ('a' <= c) && (c <= 'z') {
		return true
	}
	if ('A' <= c) && (c <= 'Z') {
		return true
	}
	if (c == '+') || (c == '/') {
		return true
	}
	return isDigit(c)
}

func isHexDigit(c byte) bool {
	if ('a' <= c) && (c <= 'f') {
		return true
	}
	if ('A' <= c) && (c <= 'F') {
		return true
	}
	return isDigit(c)
}

func isDigit(c byte) bool {
	return (('0' <= c) && (c <= '9'))
}

func (b *BallerinaParser) Parse() tree.STNode {
	ast := b.parseCompUnit()
	debugcommon.DebugWriteLazy(debugcommon.DUMP_ST, func() string { return tree.GenerateJSON(ast) })
	return ast
}

func (b *BallerinaParser) ParseImports() tree.STNode {
	ast := b.parseCompUnitImports()
	debugcommon.DebugWriteLazy(debugcommon.DUMP_ST, func() string { return tree.GenerateJSON(ast) })
	return ast
}

func (b *BallerinaParser) ParseAsStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	stmt := b.parseStatement()
	if (stmt == nil) || b.validateStatement(stmt) {
		stmt = b.createMissingSimpleVarDecl(false)
		stmt = b.invalidateRestAndAddToTrailingMinutiae(stmt)
		return stmt
	}
	if stmt.Kind() == common.NAMED_WORKER_DECLARATION {
		b.addInvalidNodeToNextToken(stmt, &common.ERROR_NAMED_WORKER_NOT_ALLOWED_HERE)
		stmt = b.createMissingSimpleVarDecl(false)
		stmt = b.invalidateRestAndAddToTrailingMinutiae(stmt)
		return stmt
	}
	stmt = b.invalidateRestAndAddToTrailingMinutiae(stmt)
	return stmt
}

func (b *BallerinaParser) ParseAsBlockStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	b.startContext(common.PARSER_RULE_CONTEXT_WHILE_BLOCK)
	blockStmtNode := b.parseBlockNode()
	blockStmtNode = b.invalidateRestAndAddToTrailingMinutiae(blockStmtNode)
	return blockStmtNode
}

func (b *BallerinaParser) ParseAsStatements() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	stmtsNode := b.parseStatements()
	stmtNodeList, ok := stmtsNode.(*tree.STNodeList)
	if !ok {
		panic("stmtsNode is not a STNodeList")
	}
	var stmts []tree.STNode
	for i := 0; i < (stmtNodeList.Size() - 1); i++ {
		stmts = append(stmts, stmtNodeList.Get(i))
	}
	var lastStmt tree.STNode
	if stmtNodeList.Size() == 0 {
		lastStmt = b.createMissingSimpleVarDecl(false)
	} else {
		lastStmt = stmtNodeList.Get(stmtNodeList.Size() - 1)
	}
	lastStmt = b.invalidateRestAndAddToTrailingMinutiae(lastStmt)
	stmts = append(stmts, lastStmt)
	return tree.CreateNodeList(stmts...)
}

func (b *BallerinaParser) ParseAsExpression() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	expr := b.parseExpression()
	expr = b.invalidateRestAndAddToTrailingMinutiae(expr)
	return expr
}

func (b *BallerinaParser) ParseAsActionOrExpression() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	actionOrExpr := b.parseActionOrExpression()
	actionOrExpr = b.invalidateRestAndAddToTrailingMinutiae(actionOrExpr)
	return actionOrExpr
}

func (b *BallerinaParser) ParseAsModuleMemberDeclaration() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	topLevelNode := b.parseTopLevelNode()
	if topLevelNode == nil {
		topLevelNode = b.createMissingSimpleVarDecl(true)
	}
	if topLevelNode.Kind() == common.IMPORT_DECLARATION {
		temp := topLevelNode
		topLevelNode = b.createMissingSimpleVarDecl(true)
		topLevelNode = tree.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(topLevelNode, temp)
	}
	topLevelNode = b.invalidateRestAndAddToTrailingMinutiae(topLevelNode)
	return topLevelNode
}

func (b *BallerinaParser) ParseAsImportDeclaration() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	importDecl := b.parseImportDecl()
	importDecl = b.invalidateRestAndAddToTrailingMinutiae(importDecl)
	return importDecl
}

func (b *BallerinaParser) ParseAsTypeDescriptor() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION)
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF)
	typeDesc = b.invalidateRestAndAddToTrailingMinutiae(typeDesc)
	return typeDesc
}

func (b *BallerinaParser) ParseAsBindingPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	bindingPattern := b.parseBindingPattern()
	bindingPattern = b.invalidateRestAndAddToTrailingMinutiae(bindingPattern)
	return bindingPattern
}

func (b *BallerinaParser) ParseAsFunctionBodyBlock() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	funcBodyBlock := b.parseFunctionBodyBlock(false)
	funcBodyBlock = b.invalidateRestAndAddToTrailingMinutiae(funcBodyBlock)
	return funcBodyBlock
}

func (b *BallerinaParser) ParseAsObjectMember() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_SERVICE_DECL)
	b.startContext(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)
	objectMember := b.parseObjectMember(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)
	if objectMember == nil {
		objectMember = b.createMissingSimpleObjectField()
	}
	objectMember = b.invalidateRestAndAddToTrailingMinutiae(objectMember)
	return objectMember
}

func (b *BallerinaParser) ParseAsIntermediateClause(allowActions bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	b.startContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
	var intermediateClause tree.STNode
	if !b.isEndOfIntermediateClause(b.peek().Kind()) {
		intermediateClause = b.parseIntermediateClause(true, allowActions)
	}
	if intermediateClause == nil {
		intermediateClause = b.createMissingWhereClause()
	}
	if intermediateClause.Kind() == common.SELECT_CLAUSE {
		temp := intermediateClause
		intermediateClause = b.createMissingWhereClause()
		intermediateClause = tree.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(intermediateClause, temp)
	}
	intermediateClause = b.invalidateRestAndAddToTrailingMinutiae(intermediateClause)
	return intermediateClause
}

func (b *BallerinaParser) ParseAsLetVarDeclaration(allowActions bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	b.switchContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
	b.switchContext(common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL)
	letVarDeclaration := b.parseLetVarDecl(common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL, true, allowActions)
	letVarDeclaration = b.invalidateRestAndAddToTrailingMinutiae(letVarDeclaration)
	return letVarDeclaration
}

func (b *BallerinaParser) ParseAsAnnotation() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATIONS)
	annotation := b.parseAnnotation()
	annotation = b.invalidateRestAndAddToTrailingMinutiae(annotation)
	return annotation
}

func (b *BallerinaParser) ParseAsMarkdownDocumentation() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	markdownDoc := b.parseMarkdownDocumentation()
	if tree.ToSourceCode(markdownDoc) == "" {
		missingHash := tree.CreateMissingTokenWithDiagnostics(common.HASH_TOKEN,
			&common.WARNING_MISSING_HASH_TOKEN)
		docLine := tree.CreateMarkdownDocumentationLineNode(common.MARKDOWN_DOCUMENTATION_LINE,
			missingHash, tree.CreateEmptyNodeList())
		markdownDoc = tree.CreateMarkdownDocumentationNode(tree.CreateNodeListFromNodes(docLine))
	}
	markdownDoc = b.invalidateRestAndAddToTrailingMinutiae(markdownDoc)
	return markdownDoc
}

func (b *BallerinaParser) ParseWithContext(context common.ParserRuleContext) tree.STNode {
	switch context {
	case common.PARSER_RULE_CONTEXT_COMP_UNIT:
		return b.parseCompUnit()
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE:
		b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
		return b.parseTopLevelNode()
	case common.PARSER_RULE_CONTEXT_STATEMENT:
		b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
		b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
		return b.parseStatement()
	case common.PARSER_RULE_CONTEXT_EXPRESSION:
		b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
		b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return b.parseExpression()
	default:
		panic("Cannot start parsing from: " + context.String())
	}
}

func (b *BallerinaParser) parseCompUnit() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	var otherDecls []tree.STNode
	var importDecls []tree.STNode
	processImports := true
	token := b.peek()
	for token.Kind() != common.EOF_TOKEN {
		decl := b.parseTopLevelNode()
		if decl == nil {
			break
		}
		if decl.Kind() == common.IMPORT_DECLARATION {
			if processImports {
				importDecls = append(importDecls, decl)
			} else {
				b.updateLastNodeInListWithInvalidNode(otherDecls, decl,
					&common.ERROR_IMPORT_DECLARATION_AFTER_OTHER_DECLARATIONS)
			}
		} else {
			if processImports {
				processImports = false
			}
			otherDecls = append(otherDecls, decl)
		}
		token = b.peek()
	}
	eof := b.consume()
	b.endContext()
	return tree.CreateModulePartNode(tree.CreateNodeList(importDecls...), tree.CreateNodeList(otherDecls...), eof)
}

func (b *BallerinaParser) parseCompUnitImports() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	var importDecls []tree.STNode
	for b.peek().Kind() == common.IMPORT_KEYWORD {
		importDecls = append(importDecls, b.parseImportDecl())
	}
	b.endContext()
	return tree.CreateModulePartNode(
		tree.CreateNodeList(importDecls...),
		tree.CreateEmptyNodeList(),
		tree.CreateMissingToken(common.EOF_TOKEN, nil),
	)
}

func (b *BallerinaParser) parseTopLevelNode() tree.STNode {
	nextToken := b.peek()
	var metadata tree.STNode
	switch nextToken.Kind() {
	case common.EOF_TOKEN:
		return nil
	case common.DOCUMENTATION_STRING, common.AT_TOKEN:
		metadata = b.parseMetaData()
		return b.parseTopLevelNodeWithMetadata(metadata)
	case common.IMPORT_KEYWORD,
		common.FINAL_KEYWORD,
		common.PUBLIC_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.TYPE_KEYWORD,
		common.LISTENER_KEYWORD,
		common.CONST_KEYWORD,
		common.ANNOTATION_KEYWORD,
		common.XMLNS_KEYWORD,
		common.ENUM_KEYWORD,
		common.CLASS_KEYWORD,
		common.TRANSACTIONAL_KEYWORD,
		common.ISOLATED_KEYWORD,
		common.DISTINCT_KEYWORD,
		common.CLIENT_KEYWORD,
		common.READONLY_KEYWORD,
		common.CONFIGURABLE_KEYWORD,
		common.SERVICE_KEYWORD:
		metadata = tree.CreateEmptyNode()
	case common.RESOURCE_KEYWORD, common.REMOTE_KEYWORD:
		b.reportInvalidQualifier(b.consume())
		return b.parseTopLevelNode()
	case common.IDENTIFIER_TOKEN:
		if b.isModuleVarDeclStart(1) || nextToken.IsMissing() {
			return b.parseModuleVarDecl(tree.CreateEmptyNode())
		}
		fallthrough
	default:
		if isTypeStartingToken(nextToken.Kind(), b.getNextNextToken()) && (nextToken.Kind() != common.IDENTIFIER_TOKEN) {
			metadata = tree.CreateEmptyNode()
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE)
		if solution.Action == ACTION_KEEP {
			metadata = tree.CreateEmptyNode()
			break
		}
		return b.parseTopLevelNode()
	}
	return b.parseTopLevelNodeWithMetadata(metadata)
}

func (b *BallerinaParser) parseTopLevelNodeWithMetadata(metadata tree.STNode) tree.STNode {
	nextToken := b.peek()
	var publicQualifier tree.STNode
	switch nextToken.Kind() {
	case common.EOF_TOKEN:
		if metadata != nil {
			metadaNode, ok := metadata.(*tree.STMetadataNode)
			if !ok {
				panic("metadata is not a STMetadataNode")
			}
			metadata = b.addMetadataNotAttachedDiagnostic(*metadaNode)
			return b.createMissingSimpleVarDeclInner(metadata, true)
		}
		return nil
	case common.PUBLIC_KEYWORD:
		publicQualifier = b.consume()
	case common.FUNCTION_KEYWORD,
		common.TYPE_KEYWORD,
		common.LISTENER_KEYWORD,
		common.CONST_KEYWORD,
		common.FINAL_KEYWORD,
		common.IMPORT_KEYWORD,
		common.ANNOTATION_KEYWORD,
		common.XMLNS_KEYWORD,
		common.ENUM_KEYWORD,
		common.CLASS_KEYWORD,
		common.TRANSACTIONAL_KEYWORD,
		common.ISOLATED_KEYWORD,
		common.DISTINCT_KEYWORD,
		common.CLIENT_KEYWORD,
		common.READONLY_KEYWORD,
		common.SERVICE_KEYWORD,
		common.CONFIGURABLE_KEYWORD:
		break
	case common.RESOURCE_KEYWORD, common.REMOTE_KEYWORD:
		b.reportInvalidQualifier(b.consume())
		return b.parseTopLevelNodeWithMetadata(metadata)
	case common.IDENTIFIER_TOKEN:
		if b.isModuleVarDeclStart(1) {
			return b.parseModuleVarDecl(metadata)
		}
		fallthrough
	default:
		if b.isTypeStartingToken(nextToken.Kind()) && (nextToken.Kind() != common.IDENTIFIER_TOKEN) {
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA)
		if solution.Action == ACTION_KEEP {
			publicQualifier = tree.CreateEmptyNode()
			break
		}
		return b.parseTopLevelNodeWithMetadata(metadata)
	}
	return b.parseTopLevelNodeWithQualifiers(metadata, publicQualifier)
}

func (b *BallerinaParser) addMetadataNotAttachedDiagnostic(metadata tree.STMetadataNode) tree.STNode {
	docString := metadata.DocumentationString
	if docString != nil {
		docString = tree.AddDiagnostic(docString, &common.ERROR_DOCUMENTATION_NOT_ATTACHED_TO_A_CONSTRUCT)
	}
	annotList, ok := metadata.Annotations.(*tree.STNodeList)
	if !ok {
		panic("annotations is not a STNodeList")
	}
	annotations := b.addAnnotNotAttachedDiagnostic(annotList)
	return tree.CreateMetadataNode(docString, annotations)
}

func (b *BallerinaParser) addAnnotNotAttachedDiagnostic(annotList *tree.STNodeList) tree.STNode {
	annotations := tree.UpdateAllNodesInNodeListWithDiagnostic(annotList, &common.ERROR_ANNOTATION_NOT_ATTACHED_TO_A_CONSTRUCT)
	return annotations
}

func (b *BallerinaParser) isModuleVarDeclStart(lookahead int) bool {
	nextToken := b.peekN(lookahead + 1)
	switch nextToken.Kind() {
	case common.EQUAL_TOKEN, // Scenario: foo = . Even though this is not valid, consider this as a var-decl and
		// continue;
		common.OPEN_BRACKET_TOKEN,  // Scenario foo[] (Array type descriptor with custom type)
		common.QUESTION_MARK_TOKEN, // Scenario foo? (Optional type descriptor with custom type)
		common.PIPE_TOKEN,          // Scenario foo | (Union type descriptor with custom type)
		common.BITWISE_AND_TOKEN,   // Scenario foo & (Intersection type descriptor with custom type)
		common.OPEN_BRACE_TOKEN,    // Scenario foo{} (mapping-binding-pattern)
		common.ERROR_KEYWORD,       // Scenario foo error (error-binding-pattern)
		common.EOF_TOKEN:
		return true
	case common.IDENTIFIER_TOKEN:
		switch b.peekN(lookahead + 2).Kind() {
		case common.EQUAL_TOKEN,
			// Scenario: foo bar =
			common.SEMICOLON_TOKEN,
			// Scenario: foo bar;
			common.EOF_TOKEN:
			return true
		default:
			return false
		}
	case common.COLON_TOKEN:
		if lookahead > 1 {
			return false
		}
		switch b.peekN(lookahead + 2).Kind() {
		case common.IDENTIFIER_TOKEN:
			return b.isModuleVarDeclStart(lookahead + 2)
		case common.EOF_TOKEN:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func (b *BallerinaParser) parseImportDecl() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_IMPORT_DECL)
	b.tokenReader.StartMode(PARSER_MODE_IMPORT_MODE)
	importKeyword := b.parseImportKeyword()
	identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME)
	importDecl := b.parseImportDeclWithIdentifier(importKeyword, identifier)
	b.tokenReader.EndMode()
	b.endContext()
	return importDecl
}

func (b *BallerinaParser) parseImportKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IMPORT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD)
		return b.parseImportKeyword()
	}
}

func (b *BallerinaParser) parseIdentifier(currentCtx common.ParserRuleContext) tree.STNode {
	token := b.peek()
	if token.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else if token.Kind() == common.MAP_KEYWORD {
		mapKeyword := b.consume()
		return tree.CreateIdentifierTokenWithDiagnostics(mapKeyword.Text(), mapKeyword.LeadingMinutiae(), mapKeyword.TrailingMinutiae(),
			mapKeyword.Diagnostics())
	} else {
		b.recoverWithBlockContext(token, currentCtx)
		return b.parseIdentifier(currentCtx)
	}
}

func (b *BallerinaParser) parseImportDeclWithIdentifier(importKeyword tree.STNode, identifier tree.STNode) tree.STNode {
	nextToken := b.peek()
	var orgName tree.STNode
	var moduleName tree.STNode
	var alias tree.STNode
	switch nextToken.Kind() {
	case common.SLASH_TOKEN:
		slash := b.parseSlashToken()
		orgName = tree.CreateImportOrgNameNode(identifier, slash)
		moduleName = b.parseModuleName()
		alias = b.parseImportPrefixDecl()
	case common.DOT_TOKEN, common.AS_KEYWORD:
		orgName = tree.CreateEmptyNode()
		moduleName = b.parseModuleNameInner(identifier)
		alias = b.parseImportPrefixDecl()
	case common.SEMICOLON_TOKEN:
		orgName = tree.CreateEmptyNode()
		moduleName = b.parseModuleNameInner(identifier)
		alias = tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS)
		return b.parseImportDeclWithIdentifier(importKeyword, identifier)
	}
	semicolon := b.parseSemicolon()
	return tree.CreateImportDeclarationNode(importKeyword, orgName, moduleName, alias, semicolon)
}

func (b *BallerinaParser) parseSlashToken() tree.STToken {
	token := b.peek()
	if token.Kind() == common.SLASH_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SLASH)
		return b.parseSlashToken()
	}
}

func (b *BallerinaParser) parseDotToken() tree.STNode {
	token := b.peek()
	if token.Kind() == common.DOT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_DOT)
		return b.parseDotToken()
	}
}

func (b *BallerinaParser) parseModuleName() tree.STNode {
	moduleNameStart := b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME)
	return b.parseModuleNameInner(moduleNameStart)
}

func (b *BallerinaParser) parseModuleNameInner(moduleNameStart tree.STNode) tree.STNode {
	var moduleNameParts []tree.STNode
	moduleNameParts = append(moduleNameParts, moduleNameStart)
	nextToken := b.peek()
	for !b.isEndOfImportDecl(nextToken) {
		moduleNameSeparator := b.parseModuleNameRhs()
		if moduleNameSeparator == nil {
			break
		}

		moduleNameParts = append(moduleNameParts, moduleNameSeparator)
		moduleNameParts = append(moduleNameParts, b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME))
		nextToken = b.peek()
	}
	return tree.CreateNodeList(moduleNameParts...)
}

func (b *BallerinaParser) parseModuleNameRhs() tree.STNode {
	switch b.peek().Kind() {
	case common.DOT_TOKEN:
		return b.consume()
	case common.AS_KEYWORD, common.SEMICOLON_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME)
		return b.parseModuleNameRhs()
	}
}

func (b *BallerinaParser) isEndOfImportDecl(nextToken tree.STToken) bool {
	switch nextToken.Kind() {
	case common.SEMICOLON_TOKEN,
		common.PUBLIC_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.TYPE_KEYWORD,
		common.ABSTRACT_KEYWORD,
		common.CONST_KEYWORD,
		common.EOF_TOKEN,
		common.SERVICE_KEYWORD,
		common.IMPORT_KEYWORD,
		common.FINAL_KEYWORD,
		common.TRANSACTIONAL_KEYWORD,
		common.ISOLATED_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseDecimalIntLiteral(context common.ParserRuleContext) tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.DECIMAL_INTEGER_LITERAL_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), context)
		return b.parseDecimalIntLiteral(context)
	}
}

func (b *BallerinaParser) parseImportPrefixDecl() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.AS_KEYWORD:
		asKeyword := b.parseAsKeyword()
		prefix := b.parseImportPrefix()
		return tree.CreateImportPrefixNode(asKeyword, prefix)
	case common.SEMICOLON_TOKEN:
		return tree.CreateEmptyNode()
	default:
		if b.isEndOfImportDecl(nextToken) {
			return tree.CreateEmptyNode()
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL)
		return b.parseImportPrefixDecl()
	}
}

func (b *BallerinaParser) parseAsKeyword() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.AS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_AS_KEYWORD)
		return b.parseAsKeyword()
	}
}

func (b *BallerinaParser) parseImportPrefix() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.IDENTIFIER_TOKEN {
		identifier := b.consume()
		if b.isUnderscoreToken(identifier) {
			return b.getUnderscoreKeyword(identifier)
		}
		return identifier
	} else if isPredeclaredPrefix(nextToken.Kind()) {
		preDeclaredPrefix := b.consume()
		return tree.CreateIdentifierToken(preDeclaredPrefix.Text(), preDeclaredPrefix.LeadingMinutiae(),
			preDeclaredPrefix.TrailingMinutiae())
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_IMPORT_PREFIX)
		return b.parseImportPrefix()
	}
}

func (b *BallerinaParser) parseTopLevelNodeWithQualifiers(metadata, publicQualifier tree.STNode) tree.STNode {
	res, _ := b.parseTopLevelNodeInner(metadata, publicQualifier, nil)
	return res
}

func (b *BallerinaParser) parseTopLevelNodeInner(metadata, publicQualifier tree.STNode, qualifiers []tree.STNode) (tree.STNode, []tree.STNode) {
	qualifiers = b.parseTopLevelQualifiers(qualifiers)
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.EOF_TOKEN:
		return b.createMissingSimpleVarDeclInnerWithQualifiers(metadata, publicQualifier, qualifiers, true), qualifiers
	case common.FUNCTION_KEYWORD:
		return b.parseFuncDefOrFuncTypeDesc(metadata, publicQualifier, qualifiers, false, false), qualifiers
	case common.TYPE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseModuleTypeDefinition(metadata, publicQualifier), qualifiers
	case common.CLASS_KEYWORD:
		return b.parseClassDefinition(metadata, publicQualifier, qualifiers), qualifiers
	case common.LISTENER_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseListenerDeclaration(metadata, publicQualifier), qualifiers
	case common.CONST_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseConstantDeclaration(metadata, publicQualifier), qualifiers
	case common.ANNOTATION_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		constKeyword := tree.CreateEmptyNode()
		return b.parseAnnotationDeclaration(metadata, publicQualifier, constKeyword), qualifiers
	case common.IMPORT_KEYWORD:
		b.reportInvalidMetaData(metadata, "import declaration")
		b.reportInvalidQualifier(publicQualifier)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseImportDecl(), qualifiers
	case common.XMLNS_KEYWORD:
		b.reportInvalidMetaData(metadata, "XML namespace declaration")
		b.reportInvalidQualifier(publicQualifier)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseXMLNamespaceDeclaration(true), qualifiers
	case common.ENUM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseEnumDeclaration(metadata, publicQualifier), qualifiers
	case common.RESOURCE_KEYWORD, common.REMOTE_KEYWORD:
		b.reportInvalidQualifier(b.consume())
		return b.parseTopLevelNodeInner(metadata, publicQualifier, qualifiers)
	case common.IDENTIFIER_TOKEN:
		if b.isModuleVarDeclStart(1) {
			return b.parseModuleVarDeclInner(metadata, publicQualifier, qualifiers)
		}
		fallthrough
	default:
		if b.isPossibleServiceDecl(qualifiers) {
			return b.parseServiceDeclOrVarDecl(metadata, publicQualifier, qualifiers), qualifiers
		}
		if b.isTypeStartingToken(nextToken.Kind()) && (nextToken.Kind() != common.IDENTIFIER_TOKEN) {
			return b.parseModuleVarDeclInner(metadata, publicQualifier, qualifiers)
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER)
		if solution.Action == ACTION_KEEP {
			return b.parseModuleVarDeclInner(metadata, publicQualifier, qualifiers)
		}
		return b.parseTopLevelNodeInner(metadata, publicQualifier, qualifiers)
	}
}

func (b *BallerinaParser) parseModuleVarDecl(metadata tree.STNode) tree.STNode {
	var emptyList []tree.STNode
	publicQualifier := tree.CreateEmptyNode()
	res, _ := b.parseVariableDeclInner(metadata, publicQualifier, emptyList, emptyList, true)
	return res
}

func (b *BallerinaParser) parseModuleVarDeclInner(metadata tree.STNode, publicQualifier tree.STNode, topLevelQualifiers []tree.STNode) (tree.STNode, []tree.STNode) {
	varDeclQuals, topLevelQualifiers := b.extractVarDeclQualifiers(topLevelQualifiers, true)
	res, _ := b.parseVariableDeclInner(metadata, publicQualifier, varDeclQuals, topLevelQualifiers, true)
	return res, topLevelQualifiers
}

func (b *BallerinaParser) extractVarDeclQualifiers(qualifiers []tree.STNode, isModuleVar bool) ([]tree.STNode, []tree.STNode) {
	var varDeclQualList []tree.STNode
	initialListSize := len(qualifiers)
	configurableQualIndex := (-1)
	i := 0
	for ; (i < 2) && (i < initialListSize); i++ {
		qualifierKind := qualifiers[0].Kind()
		if (!b.isSyntaxKindInList(varDeclQualList, qualifierKind)) && b.isModuleVarDeclQualifier(qualifierKind) {
			varDeclQualList = append(varDeclQualList, qualifiers[0])
			qualifiers = qualifiers[1:]
			if qualifierKind == common.CONFIGURABLE_KEYWORD {
				configurableQualIndex = i
			}
			continue
		}
		break
	}
	if isModuleVar && (configurableQualIndex > (-1)) {
		configurableQual := varDeclQualList[configurableQualIndex]
		i := 0
		for ; i < len(varDeclQualList); i++ {
			if i < configurableQualIndex {
				invalidQual := tree.ToToken(varDeclQualList[i])
				configurableQual = tree.CloneWithLeadingInvalidNodeMinutiae(configurableQual, invalidQual,
					b.getInvalidQualifierError(invalidQual.Kind()), (invalidQual).Text())
			} else if i > configurableQualIndex {
				invalidQual := tree.ToToken(varDeclQualList[i])
				configurableQual = tree.CloneWithTrailingInvalidNodeMinutiae(configurableQual, invalidQual,
					b.getInvalidQualifierError(invalidQual.Kind()), (invalidQual).Text())
			}
		}
		varDeclQualList = []tree.STNode{configurableQual}
	}
	return varDeclQualList, qualifiers
}

func (b *BallerinaParser) getInvalidQualifierError(qualifierKind common.SyntaxKind) *common.DiagnosticErrorCode {
	if qualifierKind == common.FINAL_KEYWORD {
		return &common.ERROR_CONFIGURABLE_VAR_IMPLICITLY_FINAL
	}
	return &common.ERROR_QUALIFIER_NOT_ALLOWED
}

func (b *BallerinaParser) isModuleVarDeclQualifier(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.FINAL_KEYWORD, common.ISOLATED_KEYWORD, common.CONFIGURABLE_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) reportInvalidQualifier(qualifier tree.STNode) {
	if (qualifier != nil) && (qualifier.Kind() != common.NONE) {
		b.addInvalidNodeToNextToken(qualifier, &common.ERROR_INVALID_QUALIFIER,
			tree.ToToken(qualifier).Text())
	}
}

func (b *BallerinaParser) reportInvalidMetaData(metadata tree.STNode, constructName string) {
	if (metadata != nil) && (metadata.Kind() != common.NONE) {
		b.addInvalidNodeToNextToken(metadata, &common.ERROR_INVALID_METADATA, constructName)
	}
}

func (b *BallerinaParser) reportInvalidQualifierList(qualifiers []tree.STNode) {
	for _, qual := range qualifiers {
		b.addInvalidNodeToNextToken(qual, &common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qual).Text())
	}
}

func (b *BallerinaParser) reportInvalidStatementAnnots(annots tree.STNode, qualifiers []tree.STNode) {
	diagnosticErrorCode := common.ERROR_ANNOTATIONS_ATTACHED_TO_STATEMENT
	b.reportInvalidAnnotations(annots, qualifiers, diagnosticErrorCode)
}

func (b *BallerinaParser) reportInvalidExpressionAnnots(annots tree.STNode, qualifiers []tree.STNode) {
	diagnosticErrorCode := common.ERROR_ANNOTATIONS_ATTACHED_TO_EXPRESSION
	b.reportInvalidAnnotations(annots, qualifiers, diagnosticErrorCode)
}

func (b *BallerinaParser) reportInvalidAnnotations(annots tree.STNode, qualifiers []tree.STNode, errorCode common.DiagnosticErrorCode) {
	if b.isNodeListEmpty(annots) {
		return
	}
	if len(qualifiers) == 0 {
		b.addInvalidNodeToNextToken(annots, &errorCode)
	} else {
		b.updateFirstNodeInListWithLeadingInvalidNode(qualifiers, annots, &errorCode)
	}
}

func (b *BallerinaParser) isTopLevelQualifier(tokenKind common.SyntaxKind) bool {
	var nextNextToken tree.STToken
	switch tokenKind {
	case common.FINAL_KEYWORD, // final-qualifier
		common.CONFIGURABLE_KEYWORD:
		return true
	case common.READONLY_KEYWORD:
		nextNextToken = b.getNextNextToken()
		switch nextNextToken.Kind() {
		case common.CLIENT_KEYWORD,
			common.SERVICE_KEYWORD,
			common.DISTINCT_KEYWORD,
			common.ISOLATED_KEYWORD,
			common.CLASS_KEYWORD:
			return true
		default:
			return false
		}
	case common.DISTINCT_KEYWORD:
		nextNextToken = b.getNextNextToken()
		switch nextNextToken.Kind() {
		case common.CLIENT_KEYWORD,
			common.SERVICE_KEYWORD,
			common.READONLY_KEYWORD,
			common.ISOLATED_KEYWORD,
			common.CLASS_KEYWORD:
			return true
		default:
			return false
		}
	default:
		return b.isTypeDescQualifier(tokenKind)
	}
}

func (b *BallerinaParser) isTypeDescQualifier(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.TRANSACTIONAL_KEYWORD, // func-type-dec, func-def
		common.ISOLATED_KEYWORD, // func-type-dec, object-type-desc, func-def, class-def, isolated-final-qual
		common.CLIENT_KEYWORD,   // object-type-desc, class-def
		common.ABSTRACT_KEYWORD, // object-type-desc(outdated)
		common.SERVICE_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isObjectMemberQualifier(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.REMOTE_KEYWORD, // method-def, method-decl
		common.RESOURCE_KEYWORD, // resource-method-def
		common.FINAL_KEYWORD:
		return true
	default:
		return b.isTypeDescQualifier(tokenKind)
	}
}

func (b *BallerinaParser) isExprQualifier(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.TRANSACTIONAL_KEYWORD:
		nextNextToken := b.getNextNextToken()
		switch nextNextToken.Kind() {
		case common.CLIENT_KEYWORD,
			common.ABSTRACT_KEYWORD,
			common.ISOLATED_KEYWORD,
			common.OBJECT_KEYWORD,
			common.FUNCTION_KEYWORD:
			return true
		default:
			return false
		}
	default:
		return b.isTypeDescQualifier(tokenKind)
	}
}

func (b *BallerinaParser) parseTopLevelQualifiers(qualifiers []tree.STNode) []tree.STNode {
	for b.isTopLevelQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *BallerinaParser) parseTypeDescQualifiers(qualifiers []tree.STNode) []tree.STNode {
	for b.isTypeDescQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *BallerinaParser) parseObjectMemberQualifiers(qualifiers []tree.STNode) []tree.STNode {
	for b.isObjectMemberQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *BallerinaParser) parseExprQualifiers(qualifiers []tree.STNode) []tree.STNode {
	for b.isExprQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *BallerinaParser) parseOptionalRelativePath(isObjectMember bool) tree.STNode {
	var resourcePath tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.DOT_TOKEN, common.IDENTIFIER_TOKEN, common.OPEN_BRACKET_TOKEN:
		resourcePath = b.parseRelativeResourcePath()
	case common.OPEN_PAREN_TOKEN:
		return tree.CreateEmptyNodeList()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH)
		return b.parseOptionalRelativePath(isObjectMember)
	}
	if !isObjectMember {
		b.addInvalidNodeToNextToken(resourcePath, &common.ERROR_RESOURCE_PATH_IN_FUNCTION_DEFINITION)
		return tree.CreateEmptyNodeList()
	}
	return resourcePath
}

func (b *BallerinaParser) parseFuncDefOrFuncTypeDesc(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, isObjectMember bool, isObjectTypeDesc bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE)
	functionKeyword := b.parseFunctionKeyword()
	funcDefOrType := b.parseFunctionKeywordRhs(metadata, visibilityQualifier, qualifiers, functionKeyword,
		isObjectMember, isObjectTypeDesc)
	return funcDefOrType
}

func (b *BallerinaParser) parseFunctionDefinition(metadata tree.STNode, visibilityQualifier tree.STNode, resourcePath tree.STNode, qualifiers []tree.STNode, functionKeyword tree.STNode, name tree.STNode, isObjectMember bool, isObjectTypeDesc bool) tree.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	funcSignature := b.parseFuncSignature(false)
	funcDef := b.parseFuncDefOrMethodDeclEnd(metadata, visibilityQualifier, qualifiers, functionKeyword, name,
		resourcePath, funcSignature, isObjectMember, isObjectTypeDesc)
	b.endContext()
	return funcDef
}

func (b *BallerinaParser) parseFuncDefOrFuncTypeDescRhs(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, functionKeyword tree.STNode, name tree.STNode, isObjectMember bool, isObjectTypeDesc bool) tree.STNode {
	switch b.peek().Kind() {
	case common.OPEN_PAREN_TOKEN,
		common.DOT_TOKEN,
		common.IDENTIFIER_TOKEN,
		common.OPEN_BRACKET_TOKEN:
		resourcePath := b.parseOptionalRelativePath(isObjectMember)
		return b.parseFunctionDefinition(metadata, visibilityQualifier, resourcePath, qualifiers, functionKeyword,
			name, isObjectMember, isObjectTypeDesc)
	case common.EQUAL_TOKEN,
		common.SEMICOLON_TOKEN:
		b.endContext()
		extractQualifiersList, qualifiers := b.extractVarDeclOrObjectFieldQualifiers(qualifiers, isObjectMember,
			isObjectTypeDesc)
		typeDesc := b.createFunctionTypeDescriptor(qualifiers, functionKeyword,
			tree.CreateEmptyNode(), false)
		if isObjectMember {
			objectFieldQualNodeList := tree.CreateNodeList(extractQualifiersList...)
			return b.parseObjectFieldRhs(metadata, visibilityQualifier, objectFieldQualNodeList, typeDesc, name,
				isObjectTypeDesc)
		}
		b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		funcTypeName := tree.CreateSimpleNameReferenceNode(name)
		refNode, ok := funcTypeName.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("expected STSimpleNameReferenceNode")
		}
		bindingPattern := b.createCaptureOrWildcardBP(refNode.Name)
		typedBindingPattern := tree.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
		res, _ := b.parseVarDeclRhsInner(metadata, visibilityQualifier, extractQualifiersList, typedBindingPattern, true)
		return res
	default:
		token := b.peek()
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS)
		return b.parseFuncDefOrFuncTypeDescRhs(metadata, visibilityQualifier, qualifiers, functionKeyword, name,
			isObjectMember, isObjectTypeDesc)
	}
}

func (b *BallerinaParser) parseFunctionKeywordRhs(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, functionKeyword tree.STNode, isObjectMember bool, isObjectTypeDesc bool) tree.STNode {
	switch b.peek().Kind() {
	case common.IDENTIFIER_TOKEN:
		name := b.consume()
		return b.parseFuncDefOrFuncTypeDescRhs(metadata, visibilityQualifier, qualifiers, functionKeyword, name,
			isObjectMember, isObjectTypeDesc)
	case common.OPEN_PAREN_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		b.startContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC)
		funcSignature := b.parseFuncSignature(true)
		b.endContext()
		b.endContext()
		return b.parseFunctionTypeDescRhs(metadata, visibilityQualifier, qualifiers, functionKeyword,
			funcSignature, isObjectMember, isObjectTypeDesc)
	default:
		token := b.peek()
		if b.isValidTypeContinuationToken(token) || b.isBindingPatternsStartToken(token.Kind()) {
			return b.parseVarDeclWithFunctionType(metadata, visibilityQualifier, qualifiers, functionKeyword,
				tree.CreateEmptyNode(), isObjectMember,
				isObjectTypeDesc, false)
		}
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS)
		return b.parseFunctionKeywordRhs(metadata, visibilityQualifier, qualifiers, functionKeyword,
			isObjectMember, isObjectTypeDesc)
	}
}

func (b *BallerinaParser) isBindingPatternsStartToken(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.IDENTIFIER_TOKEN,
		common.OPEN_BRACKET_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.ERROR_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseFuncDefOrMethodDeclEnd(metadata tree.STNode, visibilityQualifier tree.STNode, qualifierList []tree.STNode, functionKeyword tree.STNode, name tree.STNode, resourcePath tree.STNode, funcSignature tree.STNode, isObjectMember bool, isObjectTypeDesc bool) tree.STNode {
	if !isObjectMember {
		return b.createFunctionDefinition(metadata, visibilityQualifier, qualifierList, functionKeyword, name,
			funcSignature)
	}
	hasResourcePath := (!b.isNodeListEmpty(resourcePath))
	hasResourceQual := b.isSyntaxKindInList(qualifierList, common.RESOURCE_KEYWORD)
	if hasResourceQual && (!hasResourcePath) {
		var relativePath []tree.STNode
		relativePath = append(relativePath, tree.CreateMissingToken(common.DOT_TOKEN, nil))
		resourcePath = tree.CreateNodeList(relativePath...)
		var errorCode common.DiagnosticErrorCode
		if isObjectTypeDesc {
			errorCode = common.ERROR_MISSING_RESOURCE_PATH_IN_RESOURCE_ACCESSOR_DECLARATION
		} else {
			errorCode = common.ERROR_MISSING_RESOURCE_PATH_IN_RESOURCE_ACCESSOR_DEFINITION
		}
		name = tree.AddDiagnostic(name, &errorCode)
		hasResourcePath = true
	}
	if hasResourcePath {
		return b.createResourceAccessorDefnOrDecl(metadata, visibilityQualifier, qualifierList, functionKeyword, name,
			resourcePath, funcSignature, isObjectTypeDesc)
	}
	if isObjectTypeDesc {
		return b.createMethodDeclaration(metadata, visibilityQualifier, qualifierList, functionKeyword, name,
			funcSignature)
	} else {
		return b.createMethodDefinition(metadata, visibilityQualifier, qualifierList, functionKeyword, name,
			funcSignature)
	}
}

func (b *BallerinaParser) createFunctionDefinition(metadata tree.STNode, visibilityQualifier tree.STNode, qualifierList []tree.STNode, functionKeyword tree.STNode, name tree.STNode, funcSignature tree.STNode) tree.STNode {
	var validatedList []tree.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = tree.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	if visibilityQualifier != nil {
		validatedList = append([]tree.STNode{visibilityQualifier}, validatedList...)
	}
	qualifiers := tree.CreateNodeList(validatedList...)
	resourcePath := tree.CreateEmptyNodeList()
	body := b.parseFunctionBody()
	return tree.CreateFunctionDefinitionNode(common.FUNCTION_DEFINITION, metadata, qualifiers,
		functionKeyword, name, resourcePath, funcSignature, body)
}

func (b *BallerinaParser) createMethodDefinition(metadata tree.STNode, visibilityQualifier tree.STNode, qualifierList []tree.STNode, functionKeyword tree.STNode, name tree.STNode, funcSignature tree.STNode) tree.STNode {
	var validatedList []tree.STNode
	hasRemoteQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
			continue
		}
		if qualifier.Kind() == common.REMOTE_KEYWORD {
			hasRemoteQual = true
			validatedList = append(validatedList, qualifier)
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = tree.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	if visibilityQualifier != nil {
		if hasRemoteQual {
			b.updateFirstNodeInListWithLeadingInvalidNode(validatedList, visibilityQualifier,
				&common.ERROR_REMOTE_METHOD_HAS_A_VISIBILITY_QUALIFIER)
		} else {
			validatedList = append([]tree.STNode{visibilityQualifier}, validatedList...)
		}
	}
	qualifiers := tree.CreateNodeList(validatedList...)
	resourcePath := tree.CreateEmptyNodeList()
	body := b.parseFunctionBody()
	return tree.CreateFunctionDefinitionNode(common.OBJECT_METHOD_DEFINITION, metadata, qualifiers,
		functionKeyword, name, resourcePath, funcSignature, body)
}

func (b *BallerinaParser) createMethodDeclaration(metadata tree.STNode, visibilityQualifier tree.STNode, qualifierList []tree.STNode, functionKeyword tree.STNode, name tree.STNode, funcSignature tree.STNode) tree.STNode {
	var validatedList []tree.STNode
	hasRemoteQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
			continue
		}
		if qualifier.Kind() == common.REMOTE_KEYWORD {
			hasRemoteQual = true
			validatedList = append(validatedList, qualifier)
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = tree.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	if visibilityQualifier != nil {
		if hasRemoteQual {
			b.updateFirstNodeInListWithLeadingInvalidNode(validatedList, visibilityQualifier,
				&common.ERROR_REMOTE_METHOD_HAS_A_VISIBILITY_QUALIFIER)
		} else {
			validatedList = append([]tree.STNode{visibilityQualifier}, validatedList...)
		}
	}
	qualifiers := tree.CreateNodeList(validatedList...)
	resourcePath := tree.CreateEmptyNodeList()
	semicolon := b.parseSemicolon()
	return tree.CreateMethodDeclarationNode(common.METHOD_DECLARATION, metadata, qualifiers,
		functionKeyword, name, resourcePath, funcSignature, semicolon)
}

func (b *BallerinaParser) createResourceAccessorDefnOrDecl(metadata tree.STNode, visibilityQualifier tree.STNode, qualifierList []tree.STNode, functionKeyword tree.STNode, name tree.STNode, resourcePath tree.STNode, funcSignature tree.STNode, isObjectTypeDesc bool) tree.STNode {
	var validatedList []tree.STNode
	hasResourceQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
			continue
		}
		if qualifier.Kind() == common.RESOURCE_KEYWORD {
			hasResourceQual = true
			validatedList = append(validatedList, qualifier)
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = tree.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	if !hasResourceQual {
		validatedList = append(validatedList, tree.CreateMissingToken(common.RESOURCE_KEYWORD, nil))
		functionKeyword = tree.AddDiagnostic(functionKeyword, &common.ERROR_MISSING_RESOURCE_KEYWORD)
	}
	if visibilityQualifier != nil {
		b.updateFirstNodeInListWithLeadingInvalidNode(validatedList, visibilityQualifier,
			&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(visibilityQualifier).Text())
	}
	qualifiers := tree.CreateNodeList(validatedList...)
	if isObjectTypeDesc {
		semicolon := b.parseSemicolon()
		return tree.CreateMethodDeclarationNode(common.RESOURCE_ACCESSOR_DECLARATION, metadata,
			qualifiers, functionKeyword, name, resourcePath, funcSignature, semicolon)
	} else {
		body := b.parseFunctionBody()
		return tree.CreateFunctionDefinitionNode(common.RESOURCE_ACCESSOR_DEFINITION, metadata,
			qualifiers, functionKeyword, name, resourcePath, funcSignature, body)
	}
}

func (b *BallerinaParser) parseFuncSignature(isParamNameOptional bool) tree.STNode {
	openParenthesis := b.parseOpenParenthesis()
	parameters := b.parseParamList(isParamNameOptional)
	closeParenthesis := b.parseCloseParenthesis()
	b.endContext()
	returnTypeDesc := b.parseFuncReturnTypeDescriptor(isParamNameOptional)
	return tree.CreateFunctionSignatureNode(openParenthesis, parameters, closeParenthesis, returnTypeDesc)
}

func (b *BallerinaParser) parseFunctionTypeDescRhs(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, functionKeyword tree.STNode, funcSignature tree.STNode, isObjectMember bool, isObjectTypeDesc bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACE_TOKEN, common.EQUAL_TOKEN:
		break
	case common.SEMICOLON_TOKEN, common.IDENTIFIER_TOKEN, common.OPEN_BRACKET_TOKEN:
		fallthrough
	default:
		return b.parseVarDeclWithFunctionType(metadata, visibilityQualifier, qualifiers, functionKeyword,
			funcSignature, isObjectMember, isObjectTypeDesc, true)
	}
	b.switchContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	name := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_FUNCTION_NAME)
	fnSig, ok := funcSignature.(*tree.STFunctionSignatureNode)
	if !ok {
		panic("expected STFunctionSignatureNode")
	}
	funcSignature = b.validateAndGetFuncParams(*fnSig)
	resourcePath := tree.CreateEmptyNodeList()
	funcDef := b.parseFuncDefOrMethodDeclEnd(metadata, visibilityQualifier, qualifiers, functionKeyword,
		name, resourcePath, funcSignature, isObjectMember, isObjectTypeDesc)
	b.endContext()
	return funcDef
}

func (b *BallerinaParser) extractVarDeclOrObjectFieldQualifiers(qualifierList []tree.STNode, isObjectMember bool, isObjectTypeDesc bool) ([]tree.STNode, []tree.STNode) {
	if isObjectMember {
		return b.extractObjectFieldQualifiers(qualifierList, isObjectTypeDesc)
	}
	return b.extractVarDeclQualifiers(qualifierList, false)
}

func (b *BallerinaParser) createFunctionTypeDescriptor(qualifierList []tree.STNode, functionKeyword tree.STNode, funcSignature tree.STNode, hasFuncSignature bool) tree.STNode {
	nodes := b.createFuncTypeQualNodeList(qualifierList, functionKeyword, hasFuncSignature)
	qualifierNodeList := nodes[0]
	functionKeyword = nodes[1]
	return tree.CreateFunctionTypeDescriptorNode(qualifierNodeList, functionKeyword, funcSignature)
}

func (b *BallerinaParser) parseVarDeclWithFunctionType(metadata tree.STNode, visibilityQualifier tree.STNode, qualifierList []tree.STNode, functionKeyword tree.STNode, funcSignature tree.STNode, isObjectMember bool, isObjectTypeDesc bool, hasFuncSignature bool) tree.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	extractQualifiersList, qualifierList := b.extractVarDeclOrObjectFieldQualifiers(qualifierList, isObjectMember,
		isObjectTypeDesc)
	typeDesc := b.createFunctionTypeDescriptor(qualifierList, functionKeyword, funcSignature, hasFuncSignature)
	typeDesc = b.parseComplexTypeDescriptor(typeDesc,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	if isObjectMember {
		b.endContext()
		objectFieldQualNodeList := tree.CreateNodeList(extractQualifiersList...)
		fieldName := b.parseVariableName()
		return b.parseObjectFieldRhs(metadata, visibilityQualifier, objectFieldQualNodeList, typeDesc, fieldName,
			isObjectTypeDesc)
	}
	typedBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	res, _ := b.parseVarDeclRhsInner(metadata, visibilityQualifier, extractQualifiersList, typedBindingPattern, true)
	return res
}

func (b *BallerinaParser) validateAndGetFuncParams(signature tree.STFunctionSignatureNode) tree.STNode {
	parameters := signature.Parameters
	paramCount := parameters.BucketCount()
	index := 0
	for ; index < paramCount; index++ {
		param := parameters.ChildInBucket(index)
		switch param.Kind() {
		case common.REQUIRED_PARAM:
			requiredParam, ok := param.(*tree.STRequiredParameterNode)
			if !ok {
				panic("expected STRequiredParameterNode")
			}
			if b.isEmpty(requiredParam.ParamName) {
				break
			}
			continue
		case common.DEFAULTABLE_PARAM:
			defaultableParam, ok := param.(*tree.STDefaultableParameterNode)
			if !ok {
				panic("expected STDefaultableParameterNode")
			}
			if b.isEmpty(defaultableParam.ParamName) {
				break
			}
			continue
		case common.REST_PARAM:
			restParam, ok := param.(*tree.STRestParameterNode)
			if !ok {
				panic("STRestParameterNode")
			}
			if b.isEmpty(restParam.ParamName) {
				break
			}
			continue
		default:
			continue
		}
		break
	}
	if index == paramCount {
		return &signature
	}
	updatedParams := b.getUpdatedParamList(parameters, index)
	return tree.CreateFunctionSignatureNode(signature.OpenParenToken, updatedParams,
		signature.CloseParenToken, signature.ReturnTypeDesc)
}

func (b *BallerinaParser) getUpdatedParamList(parameters tree.STNode, index int) tree.STNode {
	paramCount := parameters.BucketCount()
	newIndex := 0
	var newParams []tree.STNode
	for ; newIndex < index; newIndex++ {
		newParams = append(newParams, parameters.ChildInBucket(index))
	}
	for ; newIndex < paramCount; newIndex++ {
		param := parameters.ChildInBucket(newIndex)
		paramName := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		switch param.Kind() {
		case common.REQUIRED_PARAM:
			requiredParam, ok := param.(*tree.STRequiredParameterNode)
			if !ok {
				panic("expected STRequiredParameterNode")
			}
			if b.isEmpty(requiredParam.ParamName) {
				param = tree.CreateRequiredParameterNode(requiredParam.Annotations,
					requiredParam.TypeName, paramName)
			}
		case common.DEFAULTABLE_PARAM:
			defaultableParam, ok := param.(*tree.STDefaultableParameterNode)
			if !ok {
				panic("expected STDefaultableParameterNode")
			}
			if b.isEmpty(defaultableParam.ParamName) {
				param = tree.CreateDefaultableParameterNode(defaultableParam.Annotations, defaultableParam.TypeName,
					paramName, defaultableParam.EqualsToken, defaultableParam.Expression)
			}
		case common.REST_PARAM:
			restParam, ok := param.(*tree.STRestParameterNode)
			if !ok {
				panic("expected STRestParameterNode")
			}
			if b.isEmpty(restParam.ParamName) {
				param = tree.CreateRestParameterNode(restParam.Annotations, restParam.TypeName,
					restParam.EllipsisToken, paramName)
			}
		default:
		}
		newParams = append(newParams, param)
	}
	return tree.CreateNodeList(newParams...)
}

func (b *BallerinaParser) isEmpty(node tree.STNode) bool {
	return (!tree.IsSTNodePresent(node))
}

func (b *BallerinaParser) parseFunctionKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FUNCTION_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD)
		return b.parseFunctionKeyword()
	}
}

func (b *BallerinaParser) parseFunctionName() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNC_NAME)
		return b.parseFunctionName()
	}
}

func (b *BallerinaParser) parseArgListOpenParenthesis() tree.STNode {
	return b.parseOpenParenthesisInner(common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN)
}

func (b *BallerinaParser) parseOpenParenthesis() tree.STNode {
	return b.parseOpenParenthesisInner(common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS)
}

func (b *BallerinaParser) parseOpenParenthesisInner(ctx common.ParserRuleContext) tree.STNode {
	token := b.peek()
	if token.Kind() == common.OPEN_PAREN_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, ctx)
		return b.parseOpenParenthesisInner(ctx)
	}
}

func (b *BallerinaParser) parseArgListCloseParenthesis() tree.STNode {
	return b.parseCloseParenthesisInner(common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN)
}

func (b *BallerinaParser) parseCloseParenthesis() tree.STNode {
	return b.parseCloseParenthesisInner(common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS)
}

func (b *BallerinaParser) parseCloseParenthesisInner(ctx common.ParserRuleContext) tree.STNode {
	token := b.peek()
	if token.Kind() == common.CLOSE_PAREN_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, ctx)
		return b.parseCloseParenthesisInner(ctx)
	}
}

func (b *BallerinaParser) parseParamList(isParamNameOptional bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_PARAM_LIST)
	token := b.peek()
	if b.isEndOfParametersList(token.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	var paramsList []tree.STNode
	b.startContext(common.PARSER_RULE_CONTEXT_REQUIRED_PARAM)
	firstParam := b.parseParameterInner(common.REQUIRED_PARAM, isParamNameOptional)
	prevParamKind := firstParam.Kind()
	paramsList = append(paramsList, firstParam)
	paramOrderErrorPresent := false
	token = b.peek()
	for !b.isEndOfParametersList(token.Kind()) {
		paramEnd := b.parseParameterRhs()
		if paramEnd == nil {
			break
		}
		b.endContext()
		if prevParamKind == common.DEFAULTABLE_PARAM {
			b.startContext(common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM)
		} else {
			b.startContext(common.PARSER_RULE_CONTEXT_REQUIRED_PARAM)
		}
		param := b.parseParameterInner(prevParamKind, isParamNameOptional)
		if paramOrderErrorPresent {
			b.updateLastNodeInListWithInvalidNode(paramsList, paramEnd, nil)
			b.updateLastNodeInListWithInvalidNode(paramsList, param, nil)
		} else {
			paramOrderError := b.validateParamOrder(param, prevParamKind)
			if paramOrderError == nil {
				paramsList = append(paramsList, paramEnd)
				paramsList = append(paramsList, param)
			} else {
				paramOrderErrorPresent = true
				b.updateLastNodeInListWithInvalidNode(paramsList, paramEnd, nil)
				b.updateLastNodeInListWithInvalidNode(paramsList, param, paramOrderError)
			}
		}
		prevParamKind = param.Kind()
		token = b.peek()
	}
	b.endContext()
	return tree.CreateNodeList(paramsList...)
}

func (b *BallerinaParser) validateParamOrder(param tree.STNode, prevParamKind common.SyntaxKind) diagnostics.DiagnosticCode {
	if prevParamKind == common.REST_PARAM {
		return &common.ERROR_PARAMETER_AFTER_THE_REST_PARAMETER
	} else if (prevParamKind == common.DEFAULTABLE_PARAM) && (param.Kind() == common.REQUIRED_PARAM) {
		return &common.ERROR_REQUIRED_PARAMETER_AFTER_THE_DEFAULTABLE_PARAMETER
	}
	return nil
}

func (b *BallerinaParser) isSyntaxKindInList(nodeList []tree.STNode, kind common.SyntaxKind) bool {
	for _, node := range nodeList {
		if node.Kind() == kind {
			return true
		}
	}
	return false
}

func (b *BallerinaParser) isPossibleServiceDecl(nodeList []tree.STNode) bool {
	if len(nodeList) == 0 {
		return false
	}
	firstElement := nodeList[0]
	switch firstElement.Kind() {
	case common.SERVICE_KEYWORD:
		return true
	case common.ISOLATED_KEYWORD:
		return ((len(nodeList) > 1) && (nodeList[1].Kind() == common.SERVICE_KEYWORD))
	default:
		return false
	}
}

func (b *BallerinaParser) parseParameterRhs() tree.STNode {
	return b.parseParameterRhsInner(b.peek().Kind())
}

func (b *BallerinaParser) parseParameterRhsInner(tokenKind common.SyntaxKind) tree.STNode {
	switch tokenKind {
	case common.COMMA_TOKEN:
		return b.consume()
	case common.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_PARAM_END)
		return b.parseParameterRhs()
	}
}

func (b *BallerinaParser) parseParameter(annots tree.STNode, prevParamKind common.SyntaxKind, isParamNameOptional bool) tree.STNode {
	var inclusionSymbol tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ASTERISK_TOKEN:
		inclusionSymbol = b.consume()
	case common.IDENTIFIER_TOKEN:
		inclusionSymbol = tree.CreateEmptyNode()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			inclusionSymbol = tree.CreateEmptyNode()
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION)
		if solution.Action == ACTION_KEEP {
			inclusionSymbol = tree.CreateEmptyNodeList()
			break
		}
		return b.parseParameter(annots, prevParamKind, isParamNameOptional)
	}
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	return b.parseAfterParamType(prevParamKind, annots, inclusionSymbol, ty, isParamNameOptional)
}

func (b *BallerinaParser) parseParameterInner(prevParamKind common.SyntaxKind, isParamNameOptional bool) tree.STNode {
	var annots tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.AT_TOKEN:
		annots = b.parseOptionalAnnotations()
	case common.ASTERISK_TOKEN, common.IDENTIFIER_TOKEN:
		annots = tree.CreateEmptyNodeList()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			annots = tree.CreateEmptyNodeList()
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PARAMETER_START)
		if solution.Action == ACTION_KEEP {
			annots = tree.CreateEmptyNodeList()
			break
		}
		return b.parseParameterInner(prevParamKind, isParamNameOptional)
	}
	return b.parseParameter(annots, prevParamKind, isParamNameOptional)
}

func (b *BallerinaParser) parseAfterParamType(prevParamKind common.SyntaxKind, annots tree.STNode, inclusionSymbol tree.STNode, ty tree.STNode, isParamNameOptional bool) tree.STNode {
	var paramName tree.STNode
	token := b.peek()
	switch token.Kind() {
	case common.ELLIPSIS_TOKEN:
		if inclusionSymbol != nil {
			ty = tree.CloneWithLeadingInvalidNodeMinutiae(ty, inclusionSymbol,
				&common.REST_PARAMETER_CANNOT_BE_INCLUDED_RECORD_PARAMETER)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_REST_PARAM)
		ellipsis := b.parseEllipsis()
		if isParamNameOptional && (b.peek().Kind() != common.IDENTIFIER_TOKEN) {
			paramName = tree.CreateEmptyNode()
		} else {
			paramName = b.parseVariableName()
		}
		return tree.CreateRestParameterNode(annots, ty, ellipsis, paramName)
	case common.IDENTIFIER_TOKEN:
		paramName = b.parseVariableName()
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	case common.EQUAL_TOKEN:
		if !isParamNameOptional {
			break
		}
		paramName = tree.CreateEmptyNode()
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	default:
		if !isParamNameOptional {
			break
		}
		paramName = tree.CreateEmptyNode()
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE)
	return b.parseAfterParamType(prevParamKind, annots, inclusionSymbol, ty, false)
}

func (b *BallerinaParser) parseEllipsis() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ELLIPSIS_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ELLIPSIS)
		return b.parseEllipsis()
	}
}

func (b *BallerinaParser) parseParameterRhsWithAnnots(prevParamKind common.SyntaxKind, annots tree.STNode, inclusionSymbol tree.STNode, ty tree.STNode, paramName tree.STNode) tree.STNode {
	nextToken := b.peek()
	if b.isEndOfParameter(nextToken.Kind()) {
		if inclusionSymbol != nil {
			return tree.CreateIncludedRecordParameterNode(annots, inclusionSymbol, ty, paramName)
		} else {
			return tree.CreateRequiredParameterNode(annots, ty, paramName)
		}
	} else if nextToken.Kind() == common.EQUAL_TOKEN {
		if prevParamKind == common.REQUIRED_PARAM {
			b.switchContext(common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM)
		}
		equal := b.parseAssignOp()
		expr := b.parseInferredTypeDescDefaultOrExpression()
		if inclusionSymbol != nil {
			ty = tree.CloneWithLeadingInvalidNodeMinutiae(ty, inclusionSymbol,
				&common.ERROR_DEFAULTABLE_PARAMETER_CANNOT_BE_INCLUDED_RECORD_PARAMETER)
		}
		return tree.CreateDefaultableParameterNode(annots, ty, paramName, equal, expr)
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_PARAMETER_NAME_RHS)
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	}
}

func (b *BallerinaParser) parseComma() tree.STNode {
	token := b.peek()
	if token.Kind() == common.COMMA_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COMMA)
		return b.parseComma()
	}
}

func (b *BallerinaParser) parseFuncReturnTypeDescriptor(isFuncTypeDesc bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACE_TOKEN,
		common.EQUAL_TOKEN:
		return tree.CreateEmptyNode()
	case common.RETURNS_KEYWORD:
		break
	case common.IDENTIFIER_TOKEN:
		if (!isFuncTypeDesc) || b.isSafeMissingReturnsParse() {
			break
		}
		fallthrough
	default:
		nextNextToken := b.getNextNextToken()
		if nextNextToken.Kind() == common.RETURNS_KEYWORD {
			break
		}
		return tree.CreateEmptyNode()
	}
	returnsKeyword := b.parseReturnsKeyword()
	annot := b.parseOptionalAnnotations()
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC)
	return tree.CreateReturnTypeDescriptorNode(returnsKeyword, annot, ty)
}

func (b *BallerinaParser) isSafeMissingReturnsParse() bool {
	for _, context := range b.errorHandler.GetContextStack() {
		if !b.isSafeMissingReturnsParseCtx(context) {
			return false
		}
	}
	return true
}

func (b *BallerinaParser) isSafeMissingReturnsParseCtx(ctx common.ParserRuleContext) bool {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STARTED_WITH_DENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM,
		common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT:
		return false
	default:
		return true
	}
}

func (b *BallerinaParser) parseReturnsKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.RETURNS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD)
		return b.parseReturnsKeyword()
	}
}

func (b *BallerinaParser) parseTypeDescriptor(context common.ParserRuleContext) tree.STNode {
	return b.parseTypeDescriptorWithinContext(nil, context, false, false, TYPE_PRECEDENCE_DEFAULT)
}

func (b *BallerinaParser) parseTypeDescriptorWithPrecedence(context common.ParserRuleContext, precedence TypePrecedence) tree.STNode {
	return b.parseTypeDescriptorWithinContext(nil, context, false, false, precedence)
}

func (b *BallerinaParser) parseTypeDescriptorWithQualifier(qualifiers []tree.STNode, context common.ParserRuleContext) tree.STNode {
	return b.parseTypeDescriptorWithinContext(qualifiers, context, false, false, TYPE_PRECEDENCE_DEFAULT)
}

func (b *BallerinaParser) parseTypeDescriptorInExpression(isInConditionalExpr bool) tree.STNode {
	return b.parseTypeDescriptorWithinContext(nil, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION, false, isInConditionalExpr,
		TYPE_PRECEDENCE_DEFAULT)
}

func (b *BallerinaParser) parseTypeDescriptorWithoutQualifiers(context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence TypePrecedence) tree.STNode {
	return b.parseTypeDescriptorWithinContext(nil, context, isTypedBindingPattern, isInConditionalExpr, precedence)
}

func (b *BallerinaParser) parseTypeDescriptorWithinContext(qualifiers []tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence TypePrecedence) tree.STNode {
	b.startContext(context)
	typeDesc := b.parseTypeDescriptorInner(qualifiers, context, isTypedBindingPattern, isInConditionalExpr,
		precedence)
	b.endContext()
	return typeDesc
}

func (b *BallerinaParser) parseTypeDescriptorInner(qualifiers []tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence TypePrecedence) tree.STNode {
	typeDesc := b.parseTypeDescriptorInternal(qualifiers, context, isInConditionalExpr)
	if ((typeDesc.Kind() == common.VAR_TYPE_DESC) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY) {
		var missingToken tree.STNode
		missingToken = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		missingToken = tree.CloneWithLeadingInvalidNodeMinutiae(missingToken, typeDesc,
			&common.ERROR_INVALID_USAGE_OF_VAR)
		typeDesc = tree.CreateSimpleNameReferenceNode(missingToken.(tree.STToken))
	}
	return b.parseComplexTypeDescriptorInternal(typeDesc, context, isTypedBindingPattern, precedence)
}

func (b *BallerinaParser) parseComplexTypeDescriptor(typeDesc tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool) tree.STNode {
	b.startContext(context)
	complexTypeDesc := b.parseComplexTypeDescriptorInternal(typeDesc, context, isTypedBindingPattern,
		TYPE_PRECEDENCE_DEFAULT)
	b.endContext()
	return complexTypeDesc
}

func (b *BallerinaParser) parseComplexTypeDescriptorInternal(typeDesc tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, precedence TypePrecedence) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.QUESTION_MARK_TOKEN:
		if precedence.isHigherThanOrEqual(TYPE_PRECEDENCE_ARRAY_OR_OPTIONAL) {
			return typeDesc
		}
		isPossibleOptionalType := true
		nextNextToken := b.getNextNextToken()
		if ((context == common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION) && (!b.isValidTypeContinuationToken(nextNextToken))) && b.isValidExprStart(nextNextToken.Kind()) {
			if nextNextToken.Kind() == common.OPEN_BRACE_TOKEN {
				grandParentCtx := b.errorHandler.GetGrandParentContext()
				isPossibleOptionalType = ((grandParentCtx == common.PARSER_RULE_CONTEXT_IF_BLOCK) || (grandParentCtx == common.PARSER_RULE_CONTEXT_WHILE_BLOCK))
			} else {
				isPossibleOptionalType = false
			}
		}
		if !isPossibleOptionalType {
			return typeDesc
		}
		optionalTypeDes := b.parseOptionalTypeDescriptor(typeDesc)
		return b.parseComplexTypeDescriptorInternal(optionalTypeDes, context, isTypedBindingPattern, precedence)
	case common.OPEN_BRACKET_TOKEN:
		if isTypedBindingPattern {
			return typeDesc
		}
		if precedence.isHigherThanOrEqual(TYPE_PRECEDENCE_ARRAY_OR_OPTIONAL) {
			return typeDesc
		}
		arrayTypeDesc := b.parseArrayTypeDescriptor(typeDesc)
		return b.parseComplexTypeDescriptorInternal(arrayTypeDesc, context, false, precedence)
	case common.PIPE_TOKEN:
		if precedence.isHigherThanOrEqual(TYPE_PRECEDENCE_UNION) {
			return typeDesc
		}
		newTypeDesc := b.parseUnionTypeDescriptor(typeDesc, context, isTypedBindingPattern)
		return b.parseComplexTypeDescriptorInternal(newTypeDesc, context, isTypedBindingPattern, precedence)
	case common.BITWISE_AND_TOKEN:
		if precedence.isHigherThanOrEqual(TYPE_PRECEDENCE_INTERSECTION) {
			return typeDesc
		}
		newTypeDesc := b.parseIntersectionTypeDescriptor(typeDesc, context, isTypedBindingPattern)
		return b.parseComplexTypeDescriptorInternal(newTypeDesc, context, isTypedBindingPattern, precedence)
	default:
		return typeDesc
	}
}

func (b *BallerinaParser) isValidTypeContinuationToken(token tree.STToken) bool {
	switch token.Kind() {
	case common.QUESTION_MARK_TOKEN, common.OPEN_BRACKET_TOKEN, common.PIPE_TOKEN, common.BITWISE_AND_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) validateForUsageOfVar(typeDesc tree.STNode) tree.STNode {
	if typeDesc.Kind() != common.VAR_TYPE_DESC {
		return typeDesc
	}
	var missingToken tree.STNode
	missingToken = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
	missingToken = tree.CloneWithLeadingInvalidNodeMinutiae(missingToken, typeDesc,
		&common.ERROR_INVALID_USAGE_OF_VAR)
	return tree.CreateSimpleNameReferenceNode(missingToken)
}

func (b *BallerinaParser) parseTypeDescriptorInternal(qualifiers []tree.STNode, context common.ParserRuleContext, isInConditionalExpr bool) tree.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	if b.isQualifiedIdentifierPredeclaredPrefix(nextToken.Kind()) {
		return b.parseQualifiedTypeRefOrTypeDesc(qualifiers, isInConditionalExpr)
	}
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTypeReferenceInner(isInConditionalExpr)
	case common.RECORD_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRecordTypeDescriptor()
	case common.OBJECT_KEYWORD:
		objectTypeQualifiers := b.createObjectTypeQualNodeList(qualifiers)
		return b.parseObjectTypeDescriptor(b.consume(), objectTypeQualifiers)
	case common.OPEN_PAREN_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseNilOrParenthesisedTypeDesc()
	case common.MAP_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMapTypeDescriptor(b.consume())
	case common.STREAM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStreamTypeDescriptor(b.consume())
	case common.TABLE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTableTypeDescriptor(b.consume())
	case common.FUNCTION_KEYWORD:
		return b.parseFunctionTypeDesc(qualifiers)
	case common.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTupleTypeDesc()
	case common.DISTINCT_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		distinctKeyword := b.consume()
		return b.parseDistinctTypeDesc(distinctKeyword, context)
	case common.TRANSACTION_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseQualifiedIdentWithTransactionPrefix(context)
	default:
		if isParameterizedTypeToken(nextToken.Kind()) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseParameterizedTypeDescriptor(b.consume())
		}
		if isSingletonTypeDescStart(nextToken.Kind(), b.getNextNextToken()) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseSingletonTypeDesc()
		}
		if isSimpleType(nextToken.Kind()) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseSimpleTypeDescriptor()
		}
	}
	recoveryCtx := b.getTypeDescRecoveryCtx(qualifiers)
	solution := b.recoverWithBlockContext(b.peek(), recoveryCtx)
	if solution.Action == ACTION_KEEP {
		b.reportInvalidQualifierList(qualifiers)
		return b.parseSingletonTypeDesc()
	}
	return b.parseTypeDescriptorInternal(qualifiers, context, isInConditionalExpr)
}

func (b *BallerinaParser) parseTypeDescriptorInternalWithPrecedence(qualifiers []tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence TypePrecedence) tree.STNode {
	typeDesc := b.parseTypeDescriptorInternal(qualifiers, context, isInConditionalExpr)

	// var is parsed as a built-in simple type. However, since var is not allowed everywhere,
	// validate it here. This is done to give better error messages.
	if ((typeDesc.Kind() == common.VAR_TYPE_DESC) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY) {
		var missingToken tree.STNode
		missingToken = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		missingToken = tree.CloneWithLeadingInvalidNodeMinutiae(missingToken, typeDesc,
			&common.ERROR_INVALID_USAGE_OF_VAR)
		typeDesc = tree.CreateSimpleNameReferenceNode(missingToken.(tree.STToken))
	}

	return b.parseComplexTypeDescriptorInternal(typeDesc, context, isTypedBindingPattern, precedence)
}

func (b *BallerinaParser) getTypeDescRecoveryCtx(qualifiers []tree.STNode) common.ParserRuleContext {
	if len(qualifiers) == 0 {
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	}
	lastQualifier := b.getLastNodeInList(qualifiers)
	switch lastQualifier.Kind() {
	case common.ISOLATED_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED
	case common.TRANSACTIONAL_KEYWORD:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC
	default:
		return common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR
	}
}

func (b *BallerinaParser) parseQualifiedIdentWithTransactionPrefix(context common.ParserRuleContext) tree.STNode {
	transactionKeyword := b.consume()
	identifier := tree.CreateIdentifierToken(transactionKeyword.Text(),
		transactionKeyword.LeadingMinutiae(), transactionKeyword.TrailingMinutiae())
	colon := tree.CreateMissingTokenWithDiagnostics(common.COLON_TOKEN,
		&common.ERROR_MISSING_COLON_TOKEN)
	varOrFuncName := b.parseIdentifier(context)
	return b.createQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
}

func (b *BallerinaParser) parseQualifiedTypeRefOrTypeDesc(qualifiers []tree.STNode, isInConditionalExpr bool) tree.STNode {
	preDeclaredPrefix := b.consume()
	nextNextToken := b.getNextNextToken()
	if (preDeclaredPrefix.Kind() == common.TRANSACTION_KEYWORD) || (nextNextToken.Kind() == common.IDENTIFIER_TOKEN) {
		b.reportInvalidQualifierList(qualifiers)
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	var context common.ParserRuleContext
	switch preDeclaredPrefix.Kind() {
	case common.MAP_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF
	case common.OBJECT_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF
	case common.STREAM_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF
	case common.TABLE_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF
	default:
		if isParameterizedTypeToken(preDeclaredPrefix.Kind()) {
			context = common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF
		} else {
			context = common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF
		}
	}
	solution := b.recoverWithBlockContext(b.peek(), context)
	if solution.Action == ACTION_KEEP {
		b.reportInvalidQualifierList(qualifiers)
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	return b.parseTypeDescStartWithPredeclPrefix(preDeclaredPrefix, qualifiers)
}

func (b *BallerinaParser) parseTypeDescStartWithPredeclPrefix(preDeclaredPrefix tree.STToken, qualifiers []tree.STNode) tree.STNode {
	switch preDeclaredPrefix.Kind() {
	case common.MAP_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMapTypeDescriptor(preDeclaredPrefix)
	case common.OBJECT_KEYWORD:
		objectTypeQualifiers := b.createObjectTypeQualNodeList(qualifiers)
		return b.parseObjectTypeDescriptor(preDeclaredPrefix, objectTypeQualifiers)
	case common.STREAM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStreamTypeDescriptor(preDeclaredPrefix)
	case common.TABLE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTableTypeDescriptor(preDeclaredPrefix)
	default:
		if isParameterizedTypeToken(preDeclaredPrefix.Kind()) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseParameterizedTypeDescriptor(preDeclaredPrefix)
		}
		return CreateBuiltinSimpleNameReference(preDeclaredPrefix)
	}
}

func (b *BallerinaParser) parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix tree.STToken, isInConditionalExpr bool) tree.STNode {
	identifier := tree.CreateIdentifierToken(preDeclaredPrefix.Text(),
		preDeclaredPrefix.LeadingMinutiae(), preDeclaredPrefix.TrailingMinutiae())
	return b.parseQualifiedIdentifierNode(identifier, isInConditionalExpr)
}

func (b *BallerinaParser) parseDistinctTypeDesc(distinctKeyword tree.STNode, context common.ParserRuleContext) tree.STNode {
	typeDesc := b.parseTypeDescriptorWithPrecedence(context, TYPE_PRECEDENCE_DISTINCT)
	return tree.CreateDistinctTypeDescriptorNode(distinctKeyword, typeDesc)
}

func (b *BallerinaParser) parseNilOrParenthesisedTypeDesc() tree.STNode {
	openParen := b.parseOpenParenthesis()
	return b.parseNilOrParenthesisedTypeDescRhs(openParen)
}

func (b *BallerinaParser) parseNilOrParenthesisedTypeDescRhs(openParen tree.STNode) tree.STNode {
	var closeParen tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.CLOSE_PAREN_TOKEN:
		closeParen = b.parseCloseParenthesis()
		return tree.CreateNilTypeDescriptorNode(openParen, closeParen)
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			typedesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS)
			closeParen = b.parseCloseParenthesis()
			return tree.CreateParenthesisedTypeDescriptorNode(openParen, typedesc, closeParen)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS)
		return b.parseNilOrParenthesisedTypeDescRhs(openParen)
	}
}

func (b *BallerinaParser) parseSimpleTypeInTerminalExpr() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION)
	simpleTypeDescriptor := b.parseSimpleTypeDescriptor()
	b.endContext()
	return simpleTypeDescriptor
}

func (b *BallerinaParser) parseSimpleTypeDescriptor() tree.STNode {
	nextToken := b.peek()
	if isSimpleType(nextToken.Kind()) {
		token := b.consume()
		return CreateBuiltinSimpleNameReference(token)
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR)
		return b.parseSimpleTypeDescriptor()
	}
}

func (b *BallerinaParser) parseFunctionBody() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.EQUAL_TOKEN:
		return b.parseExternalFunctionBody()
	case common.OPEN_BRACE_TOKEN:
		return b.parseFunctionBodyBlock(false)
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		return b.parseExpressionFuncBody(false, false)
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNC_BODY)
		return b.parseFunctionBody()
	}
}

func (b *BallerinaParser) parseFunctionBodyBlock(isAnonFunc bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	openBrace := b.parseOpenBrace()
	token := b.peek()
	firstStmtList := make([]tree.STNode, 0)
	workers := make([]tree.STNode, 0)
	secondStmtList := make([]tree.STNode, 0)
	currentCtx := common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_INIT
	hasNamedWorkers := false
	for !b.isEndOfFuncBodyBlock(token.Kind(), isAnonFunc) {
		stmt := b.parseStatement()
		if stmt == nil {
			break
		}
		if b.validateStatement(stmt) {
			continue
		}
		switch currentCtx {
		case common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_INIT:
			if stmt.Kind() != common.NAMED_WORKER_DECLARATION {
				firstStmtList = append(firstStmtList, stmt)
				break
			}
			currentCtx = common.PARSER_RULE_CONTEXT_NAMED_WORKERS
			hasNamedWorkers = true
			fallthrough
		case common.PARSER_RULE_CONTEXT_NAMED_WORKERS:
			if stmt.Kind() == common.NAMED_WORKER_DECLARATION {
				workers = append(workers, stmt)
				break
			}
			currentCtx = common.PARSER_RULE_CONTEXT_DEFAULT_WORKER
			fallthrough
		case common.PARSER_RULE_CONTEXT_DEFAULT_WORKER:
			fallthrough
		default:
			if stmt.Kind() == common.NAMED_WORKER_DECLARATION {
				b.updateLastNodeInListWithInvalidNode(secondStmtList, stmt,
					&common.ERROR_NAMED_WORKER_NOT_ALLOWED_HERE)
				break
			}
			secondStmtList = append(secondStmtList, stmt)
		}
		token = b.peek()
	}
	var namedWorkersList tree.STNode
	var statements tree.STNode
	if hasNamedWorkers {
		workerInitStatements := tree.CreateNodeList(firstStmtList...)
		namedWorkers := tree.CreateNodeList(workers...)
		namedWorkersList = tree.CreateNamedWorkerDeclarator(workerInitStatements, namedWorkers)
		statements = tree.CreateNodeList(secondStmtList...)
	} else {
		namedWorkersList = tree.CreateEmptyNode()
		statements = tree.CreateNodeList(firstStmtList...)
	}
	closeBrace := b.parseCloseBrace()
	var semicolon tree.STNode
	if isAnonFunc {
		semicolon = tree.CreateEmptyNode()
	} else {
		semicolon = b.parseOptionalSemicolon()
	}
	b.endContext()
	return tree.CreateFunctionBodyBlockNode(openBrace, namedWorkersList, statements, closeBrace,
		semicolon)
}

func (b *BallerinaParser) isEndOfFuncBodyBlock(nextTokenKind common.SyntaxKind, isAnonFunc bool) bool {
	if isAnonFunc {
		switch nextTokenKind {
		case common.CLOSE_BRACE_TOKEN, common.CLOSE_PAREN_TOKEN, common.CLOSE_BRACKET_TOKEN,
			common.OPEN_BRACE_TOKEN, common.SEMICOLON_TOKEN, common.COMMA_TOKEN,
			common.PUBLIC_KEYWORD, common.EOF_TOKEN, common.EQUAL_TOKEN, common.BACKTICK_TOKEN:
			return true
		default:
			break
		}
	}
	return b.isEndOfStatements()
}

func (b *BallerinaParser) isEndOfRecordTypeNode(_ common.SyntaxKind) bool {
	return b.isEndOfModuleLevelNode(1)
}

func (b *BallerinaParser) isEndOfObjectTypeNode() bool {
	return b.isEndOfModuleLevelNodeInner(1, true)
}

func (b *BallerinaParser) isEndOfStatements() bool {
	switch b.peek().Kind() {
	case common.RESOURCE_KEYWORD:
		return true
	default:
		return b.isEndOfModuleLevelNode(1)
	}
}

func (b *BallerinaParser) isEndOfModuleLevelNode(peekIndex int) bool {
	return b.isEndOfModuleLevelNodeInner(peekIndex, false)
}

func (b *BallerinaParser) isEndOfModuleLevelNodeInner(peekIndex int, isObject bool) bool {
	switch b.peekN(peekIndex).Kind() {
	case common.EOF_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.CLOSE_BRACE_PIPE_TOKEN,
		common.IMPORT_KEYWORD,
		common.ANNOTATION_KEYWORD,
		common.LISTENER_KEYWORD,
		common.CLASS_KEYWORD:
		return true
	case common.SERVICE_KEYWORD:
		return b.isServiceDeclStart(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER, 1)
	case common.PUBLIC_KEYWORD:
		return ((!isObject) && b.isEndOfModuleLevelNodeInner(peekIndex+1, false))
	case common.FUNCTION_KEYWORD:
		if isObject {
			return false
		}
		return ((b.peekN(peekIndex+1).Kind() == common.IDENTIFIER_TOKEN) && (b.peekN(peekIndex+2).Kind() == common.OPEN_PAREN_TOKEN))
	default:
		return false
	}
}

func (b *BallerinaParser) isEndOfParameter(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.CLOSE_PAREN_TOKEN,
		common.CLOSE_BRACKET_TOKEN,
		common.SEMICOLON_TOKEN,
		common.COMMA_TOKEN,
		common.RETURNS_KEYWORD,
		common.TYPE_KEYWORD,
		common.IF_KEYWORD,
		common.WHILE_KEYWORD,
		common.DO_KEYWORD,
		common.AT_TOKEN:
		return true
	default:
		return b.isEndOfModuleLevelNode(1)
	}
}

func (b *BallerinaParser) isEndOfParametersList(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.CLOSE_PAREN_TOKEN,
		common.SEMICOLON_TOKEN,
		common.RETURNS_KEYWORD,
		common.TYPE_KEYWORD,
		common.IF_KEYWORD,
		common.WHILE_KEYWORD,
		common.DO_KEYWORD,
		common.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return b.isEndOfModuleLevelNode(1)
	}
}

func (b *BallerinaParser) parseStatementStartIdentifier() tree.STNode {
	return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME)
}

func (b *BallerinaParser) parseVariableName() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_VARIABLE_NAME)
		return b.parseVariableName()
	}
}

func (b *BallerinaParser) parseOpenBrace() tree.STNode {
	token := b.peek()
	if token.Kind() == common.OPEN_BRACE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OPEN_BRACE)
		return b.parseOpenBrace()
	}
}

func (b *BallerinaParser) parseCloseBrace() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CLOSE_BRACE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSE_BRACE)
		return b.parseCloseBrace()
	}
}

func (b *BallerinaParser) parseExternalFunctionBody() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY)
	assign := b.parseAssignOp()
	return b.parseExternalFuncBodyRhs(assign)
}

func (b *BallerinaParser) parseExternalFuncBodyRhs(assign tree.STNode) tree.STNode {
	var annotation tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.AT_TOKEN:
		annotation = b.parseAnnotations()
	case common.EXTERNAL_KEYWORD:
		annotation = tree.CreateEmptyNodeList()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS)
		return b.parseExternalFuncBodyRhs(assign)
	}
	externalKeyword := b.parseExternalKeyword()
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateExternalFunctionBodyNode(assign, annotation, externalKeyword, semicolon)
}

func (b *BallerinaParser) parseSemicolon() tree.STNode {
	token := b.peek()
	if token.Kind() == common.SEMICOLON_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SEMICOLON)
		return b.parseSemicolon()
	}
}

func (b *BallerinaParser) parseOptionalSemicolon() tree.STNode {
	token := b.peek()
	if token.Kind() == common.SEMICOLON_TOKEN {
		return b.consume()
	}
	return tree.CreateEmptyNode()
}

func (b *BallerinaParser) parseExternalKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.EXTERNAL_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD)
		return b.parseExternalKeyword()
	}
}

func (b *BallerinaParser) parseAssignOp() tree.STNode {
	token := b.peek()
	if token.Kind() == common.EQUAL_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ASSIGN_OP)
		return b.parseAssignOp()
	}
}

func (b *BallerinaParser) parseBinaryOperator() tree.STNode {
	token := b.peek()
	if b.isBinaryOperator(token.Kind()) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR)
		return b.parseBinaryOperator()
	}
}

func (b *BallerinaParser) isBinaryOperator(kind common.SyntaxKind) bool {
	switch kind {
	case common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.SLASH_TOKEN,
		common.ASTERISK_TOKEN,
		common.GT_TOKEN,
		common.LT_TOKEN,
		common.DOUBLE_EQUAL_TOKEN,
		common.TRIPPLE_EQUAL_TOKEN,
		common.LT_EQUAL_TOKEN,
		common.GT_EQUAL_TOKEN,
		common.NOT_EQUAL_TOKEN,
		common.NOT_DOUBLE_EQUAL_TOKEN,
		common.BITWISE_AND_TOKEN,
		common.BITWISE_XOR_TOKEN,
		common.PIPE_TOKEN,
		common.LOGICAL_AND_TOKEN,
		common.LOGICAL_OR_TOKEN,
		common.PERCENT_TOKEN,
		common.DOUBLE_LT_TOKEN,
		common.DOUBLE_GT_TOKEN,
		common.TRIPPLE_GT_TOKEN,
		common.ELLIPSIS_TOKEN,
		common.DOUBLE_DOT_LT_TOKEN,
		common.ELVIS_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) getOpPrecedence(binaryOpKind common.SyntaxKind) OperatorPrecedence {
	switch binaryOpKind {
	case common.ASTERISK_TOKEN, // multiplication
		common.SLASH_TOKEN, // division
		common.PERCENT_TOKEN:
		return OPERATOR_PRECEDENCE_MULTIPLICATIVE
	case common.PLUS_TOKEN, common.MINUS_TOKEN:
		return OPERATOR_PRECEDENCE_ADDITIVE
	case common.GT_TOKEN,
		common.LT_TOKEN,
		common.GT_EQUAL_TOKEN,
		common.LT_EQUAL_TOKEN,
		common.IS_KEYWORD,
		common.NOT_IS_KEYWORD:
		return OPERATOR_PRECEDENCE_BINARY_COMPARE
	case common.DOT_TOKEN,
		common.OPEN_BRACKET_TOKEN,
		common.OPEN_PAREN_TOKEN,
		common.ANNOT_CHAINING_TOKEN,
		common.OPTIONAL_CHAINING_TOKEN,
		common.DOT_LT_TOKEN,
		common.SLASH_LT_TOKEN,
		common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		common.SLASH_ASTERISK_TOKEN:
		return OPERATOR_PRECEDENCE_MEMBER_ACCESS
	case common.DOUBLE_EQUAL_TOKEN,
		common.TRIPPLE_EQUAL_TOKEN,
		common.NOT_EQUAL_TOKEN,
		common.NOT_DOUBLE_EQUAL_TOKEN:
		return OPERATOR_PRECEDENCE_EQUALITY
	case common.BITWISE_AND_TOKEN:
		return OPERATOR_PRECEDENCE_BITWISE_AND
	case common.BITWISE_XOR_TOKEN:
		return OPERATOR_PRECEDENCE_BITWISE_XOR
	case common.PIPE_TOKEN:
		return OPERATOR_PRECEDENCE_BITWISE_OR
	case common.LOGICAL_AND_TOKEN:
		return OPERATOR_PRECEDENCE_LOGICAL_AND
	case common.LOGICAL_OR_TOKEN:
		return OPERATOR_PRECEDENCE_LOGICAL_OR
	case common.RIGHT_ARROW_TOKEN:
		return OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		return OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET
	case common.SYNC_SEND_TOKEN:
		return OPERATOR_PRECEDENCE_ACTION
	case common.DOUBLE_LT_TOKEN,
		common.DOUBLE_GT_TOKEN,
		common.TRIPPLE_GT_TOKEN:
		return OPERATOR_PRECEDENCE_SHIFT
	case common.ELLIPSIS_TOKEN,
		common.DOUBLE_DOT_LT_TOKEN:
		return OPERATOR_PRECEDENCE_RANGE
	case common.ELVIS_TOKEN:
		return OPERATOR_PRECEDENCE_ELVIS_CONDITIONAL
	case common.QUESTION_MARK_TOKEN, common.COLON_TOKEN:
		return OPERATOR_PRECEDENCE_CONDITIONAL
	default:
		panic("Unsupported binary operator '" + binaryOpKind.StrValue() + "'")
	}
}

func (b *BallerinaParser) getBinaryOperatorKindToInsert(opPrecedenceLevel OperatorPrecedence) common.SyntaxKind {
	switch opPrecedenceLevel {
	case OPERATOR_PRECEDENCE_MULTIPLICATIVE:
		return common.ASTERISK_TOKEN
	case OPERATOR_PRECEDENCE_DEFAULT,
		OPERATOR_PRECEDENCE_UNARY,
		OPERATOR_PRECEDENCE_ACTION,
		OPERATOR_PRECEDENCE_EXPRESSION_ACTION,
		OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION,
		OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET,
		OPERATOR_PRECEDENCE_QUERY,
		OPERATOR_PRECEDENCE_TRAP,
		OPERATOR_PRECEDENCE_ADDITIVE:
		return common.PLUS_TOKEN
	case OPERATOR_PRECEDENCE_SHIFT:
		return common.DOUBLE_LT_TOKEN
	case OPERATOR_PRECEDENCE_RANGE:
		return common.ELLIPSIS_TOKEN
	case OPERATOR_PRECEDENCE_BINARY_COMPARE:
		return common.LT_TOKEN
	case OPERATOR_PRECEDENCE_EQUALITY:
		return common.DOUBLE_EQUAL_TOKEN
	case OPERATOR_PRECEDENCE_BITWISE_AND:
		return common.BITWISE_AND_TOKEN
	case OPERATOR_PRECEDENCE_BITWISE_XOR:
		return common.BITWISE_XOR_TOKEN
	case OPERATOR_PRECEDENCE_BITWISE_OR:
		return common.PIPE_TOKEN
	case OPERATOR_PRECEDENCE_LOGICAL_AND:
		return common.LOGICAL_AND_TOKEN
	case OPERATOR_PRECEDENCE_LOGICAL_OR:
		return common.LOGICAL_OR_TOKEN
	case OPERATOR_PRECEDENCE_ELVIS_CONDITIONAL:
		return common.ELVIS_TOKEN
	default:
		panic(
			"Unsupported operator precedence level")
	}
}

func (b *BallerinaParser) getMissingBinaryOperatorContext(opPrecedenceLevel OperatorPrecedence) common.ParserRuleContext {
	switch opPrecedenceLevel {
	case OPERATOR_PRECEDENCE_MULTIPLICATIVE:
		return common.PARSER_RULE_CONTEXT_ASTERISK
	case OPERATOR_PRECEDENCE_DEFAULT,
		OPERATOR_PRECEDENCE_UNARY,
		OPERATOR_PRECEDENCE_ACTION,
		OPERATOR_PRECEDENCE_EXPRESSION_ACTION,
		OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION,
		OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET,
		OPERATOR_PRECEDENCE_QUERY,
		OPERATOR_PRECEDENCE_TRAP,
		OPERATOR_PRECEDENCE_ADDITIVE:
		return common.PARSER_RULE_CONTEXT_PLUS_TOKEN
	case OPERATOR_PRECEDENCE_SHIFT:
		return common.PARSER_RULE_CONTEXT_DOUBLE_LT
	case OPERATOR_PRECEDENCE_RANGE:
		return common.PARSER_RULE_CONTEXT_ELLIPSIS
	case OPERATOR_PRECEDENCE_BINARY_COMPARE:
		return common.PARSER_RULE_CONTEXT_LT_TOKEN
	case OPERATOR_PRECEDENCE_EQUALITY:
		return common.PARSER_RULE_CONTEXT_DOUBLE_EQUAL
	case BITWISE_AND:
		return common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR
	case BITWISE_XOR:
		return common.PARSER_RULE_CONTEXT_BITWISE_XOR
	case OPERATOR_PRECEDENCE_BITWISE_OR:
		return common.PARSER_RULE_CONTEXT_PIPE
	case OPERATOR_PRECEDENCE_LOGICAL_AND:
		return common.PARSER_RULE_CONTEXT_LOGICAL_AND
	case OPERATOR_PRECEDENCE_LOGICAL_OR:
		return common.PARSER_RULE_CONTEXT_LOGICAL_OR
	case OPERATOR_PRECEDENCE_ELVIS_CONDITIONAL:
		return common.PARSER_RULE_CONTEXT_ELVIS
	default:
		panic(
			"Unsupported operator precedence level")
	}
}

func (b *BallerinaParser) parseModuleTypeDefinition(metadata tree.STNode, qualifier tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION)
	typeKeyword := b.parseTypeKeyword()
	typeName := b.parseTypeName()
	typeDescriptor := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF)
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateTypeDefinitionNode(metadata, qualifier, typeKeyword, typeName, typeDescriptor,
		semicolon)
}

func (b *BallerinaParser) parseClassDefinition(metadata tree.STNode, qualifier tree.STNode, qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION)
	classTypeQualifiers := b.createClassTypeQualNodeList(qualifiers)
	classKeyword := b.parseClassKeyword()
	className := b.parseClassName()
	openBrace := b.parseOpenBrace()
	classMembers := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_CLASS_MEMBER)
	closeBrace := b.parseCloseBrace()
	semicolon := b.parseOptionalSemicolon()
	b.endContext()
	return tree.CreateClassDefinitionNode(metadata, qualifier, classTypeQualifiers, classKeyword,
		className, openBrace, classMembers, closeBrace, semicolon)
}

func (b *BallerinaParser) isClassTypeQual(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.READONLY_KEYWORD, common.DISTINCT_KEYWORD, common.ISOLATED_KEYWORD:
		return true
	default:
		return b.isObjectNetworkQual(tokenKind)
	}
}

func (b *BallerinaParser) isObjectTypeQual(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.ISOLATED_KEYWORD:
		return true
	default:
		return b.isObjectNetworkQual(tokenKind)
	}
}

func (b *BallerinaParser) isObjectNetworkQual(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.SERVICE_KEYWORD, common.CLIENT_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) createClassTypeQualNodeList(qualifierList []tree.STNode) tree.STNode {
	var validatedList []tree.STNode
	hasNetworkQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
			continue
		}
		if b.isObjectNetworkQual(qualifier.Kind()) {
			if hasNetworkQual {
				b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
					&common.ERROR_MORE_THAN_ONE_OBJECT_NETWORK_QUALIFIERS)
			} else {
				validatedList = append(validatedList, qualifier)
				hasNetworkQual = true
			}
			continue
		}
		if b.isClassTypeQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	return tree.CreateNodeList(validatedList...)
}

func (b *BallerinaParser) createObjectTypeQualNodeList(qualifierList []tree.STNode) tree.STNode {
	var validatedList []tree.STNode
	hasNetworkQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
			continue
		}
		if b.isObjectNetworkQual(qualifier.Kind()) {
			if hasNetworkQual {
				b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
					&common.ERROR_MORE_THAN_ONE_OBJECT_NETWORK_QUALIFIERS)
			} else {
				validatedList = append(validatedList, qualifier)
				hasNetworkQual = true
			}
			continue
		}
		if b.isObjectTypeQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	return tree.CreateNodeList(validatedList...)
}

func (b *BallerinaParser) parseClassKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CLASS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLASS_KEYWORD)
		return b.parseClassKeyword()
	}
}

func (b *BallerinaParser) parseTypeKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.TYPE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TYPE_KEYWORD)
		return b.parseTypeKeyword()
	}
}

func (b *BallerinaParser) parseTypeName() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TYPE_NAME)
		return b.parseTypeName()
	}
}

func (b *BallerinaParser) parseClassName() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLASS_NAME)
		return b.parseClassName()
	}
}

func (b *BallerinaParser) parseRecordTypeDescriptor() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR)
	recordKeyword := b.parseRecordKeyword()
	bodyStartDelimiter := b.parseRecordBodyStartDelimiter()
	var recordFields []tree.STNode
	token := b.peek()
	recordRestDescriptor := tree.CreateEmptyNode()
	for !b.isEndOfRecordTypeNode(token.Kind()) {
		field := b.parseFieldOrRestDescriptor()
		if field == nil {
			break
		}
		token = b.peek()
		if (field.Kind() == common.RECORD_REST_TYPE) && (bodyStartDelimiter.Kind() == common.OPEN_BRACE_TOKEN) {
			if len(recordFields) == 0 {
				bodyStartDelimiter = tree.CloneWithTrailingInvalidNodeMinutiae(bodyStartDelimiter, field,
					&common.ERROR_INCLUSIVE_RECORD_TYPE_CANNOT_CONTAIN_REST_FIELD)
			} else {
				b.updateLastNodeInListWithInvalidNode(recordFields, field,
					&common.ERROR_INCLUSIVE_RECORD_TYPE_CANNOT_CONTAIN_REST_FIELD)
			}
			continue
		} else if field.Kind() == common.RECORD_REST_TYPE {
			recordRestDescriptor = field
			for !b.isEndOfRecordTypeNode(token.Kind()) {
				invalidField := b.parseFieldOrRestDescriptor()
				if invalidField == nil {
					break
				}
				recordRestDescriptor = tree.CloneWithTrailingInvalidNodeMinutiae(recordRestDescriptor,
					invalidField, &common.ERROR_MORE_RECORD_FIELDS_AFTER_REST_FIELD)
				token = b.peek()
			}
			break
		}
		recordFields = append(recordFields, field)
	}
	fields := tree.CreateNodeList(recordFields...)
	bodyEndDelimiter := b.parseRecordBodyCloseDelimiter(bodyStartDelimiter.Kind())
	b.endContext()
	return tree.CreateRecordTypeDescriptorNode(recordKeyword, bodyStartDelimiter, fields,
		recordRestDescriptor, bodyEndDelimiter)
}

func (b *BallerinaParser) parseRecordBodyStartDelimiter() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACE_PIPE_TOKEN:
		return b.parseClosedRecordBodyStart()
	case common.OPEN_BRACE_TOKEN:
		return b.parseOpenBrace()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RECORD_BODY_START)
		return b.parseRecordBodyStartDelimiter()
	}
}

func (b *BallerinaParser) parseClosedRecordBodyStart() tree.STNode {
	token := b.peek()
	if token.Kind() == common.OPEN_BRACE_PIPE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START)
		return b.parseClosedRecordBodyStart()
	}
}

func (b *BallerinaParser) parseRecordBodyCloseDelimiter(startingDelimeter common.SyntaxKind) tree.STNode {
	if startingDelimeter == common.OPEN_BRACE_PIPE_TOKEN {
		return b.parseClosedRecordBodyEnd()
	}
	return b.parseCloseBrace()
}

func (b *BallerinaParser) parseClosedRecordBodyEnd() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CLOSE_BRACE_PIPE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END)
		return b.parseClosedRecordBodyEnd()
	}
}

func (b *BallerinaParser) parseRecordKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.RECORD_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RECORD_KEYWORD)
		return b.parseRecordKeyword()
	}
}

func (b *BallerinaParser) parseFieldOrRestDescriptor() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.CLOSE_BRACE_TOKEN,
		common.CLOSE_BRACE_PIPE_TOKEN:
		return nil
	case common.ASTERISK_TOKEN:
		b.startContext(common.PARSER_RULE_CONTEXT_RECORD_FIELD)
		asterisk := b.consume()
		ty := b.parseTypeReferenceInTypeInclusion()
		semicolonToken := b.parseSemicolon()
		b.endContext()
		return tree.CreateTypeReferenceNode(asterisk, ty, semicolonToken)
	case common.DOCUMENTATION_STRING,
		common.AT_TOKEN:
		return b.parseRecordField()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseRecordField()
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END)
		return b.parseFieldOrRestDescriptor()
	}
}

func (b *BallerinaParser) parseRecordField() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RECORD_FIELD)
	metadata := b.parseMetaData()
	fieldOrRestDesc := b.parseRecordFieldInner(b.peek(), metadata)
	b.endContext()
	return fieldOrRestDesc
}

func (b *BallerinaParser) parseRecordFieldInner(nextToken tree.STToken, metadata tree.STNode) tree.STNode {
	if nextToken.Kind() != common.READONLY_KEYWORD {
		ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD)
		return b.parseFieldOrRestDescriptorRhs(metadata, ty)
	}
	var ty tree.STNode
	var readOnlyQualifier tree.STNode
	readOnlyQualifier = b.parseReadonlyKeyword()
	nextToken = b.peek()
	if nextToken.Kind() == common.IDENTIFIER_TOKEN {
		fieldNameOrTypeDesc := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_RECORD_FIELD_NAME_OR_TYPE_NAME)
		if fieldNameOrTypeDesc.Kind() == common.QUALIFIED_NAME_REFERENCE {
			ty = fieldNameOrTypeDesc
		} else {
			nextToken = b.peek()
			switch nextToken.Kind() {
			case common.SEMICOLON_TOKEN, common.EQUAL_TOKEN:
				ty = CreateBuiltinSimpleNameReference(readOnlyQualifier)
				readOnlyQualifier = tree.CreateEmptyNode()
				nameNode, ok := fieldNameOrTypeDesc.(*tree.STSimpleNameReferenceNode)
				if !ok {
					panic("expected STSimpleNameReferenceNode")
				}
				fieldName := nameNode.Name
				return b.parseFieldDescriptorRhs(metadata, readOnlyQualifier, ty, fieldName)
			default:
				ty = b.parseComplexTypeDescriptor(fieldNameOrTypeDesc,
					common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD, false)
			}
		}
	} else if nextToken.Kind() == common.ELLIPSIS_TOKEN {
		ty = CreateBuiltinSimpleNameReference(readOnlyQualifier)
		return b.parseFieldOrRestDescriptorRhs(metadata, ty)
	} else if b.isTypeStartingToken(nextToken.Kind()) {
		ty = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD)
	} else {
		readOnlyQualifier = CreateBuiltinSimpleNameReference(readOnlyQualifier)
		ty = b.parseComplexTypeDescriptor(readOnlyQualifier, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD, false)
		readOnlyQualifier = tree.CreateEmptyNode()
	}
	return b.parseIndividualRecordField(metadata, readOnlyQualifier, ty)
}

func (b *BallerinaParser) parseIndividualRecordField(metadata tree.STNode, readOnlyQualifier tree.STNode, ty tree.STNode) tree.STNode {
	fieldName := b.parseVariableName()
	return b.parseFieldDescriptorRhs(metadata, readOnlyQualifier, ty, fieldName)
}

func (b *BallerinaParser) parseTypeReferenceInTypeInclusion() tree.STNode {
	typeReference := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION)
	if typeReference.Kind() == common.SIMPLE_NAME_REFERENCE {
		if typeReference.HasDiagnostics() {
			emptyNameReference := tree.CreateSimpleNameReferenceNode(tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN, &common.ERROR_MISSING_IDENTIFIER))
			return emptyNameReference
		}
		return typeReference
	}
	if typeReference.Kind() == common.QUALIFIED_NAME_REFERENCE {
		return typeReference
	}
	emptyNameReference := tree.CreateSimpleNameReferenceNode(tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil))
	emptyNameReference = tree.CloneWithTrailingInvalidNodeMinutiae(emptyNameReference, typeReference,
		&common.ERROR_ONLY_TYPE_REFERENCE_ALLOWED_AS_TYPE_INCLUSIONS)
	return emptyNameReference
}

func (b *BallerinaParser) parseTypeReference() tree.STNode {
	return b.parseTypeReferenceInner(false)
}

func (b *BallerinaParser) parseTypeReferenceInner(isInConditionalExpr bool) tree.STNode {
	return b.parseQualifiedIdentifierInner(common.PARSER_RULE_CONTEXT_TYPE_REFERENCE, isInConditionalExpr)
}

func (b *BallerinaParser) parseQualifiedIdentifier(currentCtx common.ParserRuleContext) tree.STNode {
	return b.parseQualifiedIdentifierInner(currentCtx, false)
}

func (b *BallerinaParser) parseQualifiedIdentifierInner(currentCtx common.ParserRuleContext, isInConditionalExpr bool) tree.STNode {
	token := b.peek()
	var typeRefOrPkgRef tree.STNode
	if token.Kind() == common.IDENTIFIER_TOKEN {
		typeRefOrPkgRef = b.consume()
	} else if b.isQualifiedIdentifierPredeclaredPrefix(token.Kind()) {
		preDeclaredPrefix := b.consume()
		typeRefOrPkgRef = tree.CreateIdentifierToken(preDeclaredPrefix.Text(),
			preDeclaredPrefix.LeadingMinutiae(), preDeclaredPrefix.TrailingMinutiae())
	} else {
		b.recover(token, currentCtx, false)
		if b.peek().Kind() != common.IDENTIFIER_TOKEN {
			b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
			return b.parseQualifiedIdentifierInner(currentCtx, isInConditionalExpr)
		}
		typeRefOrPkgRef = b.consume()
	}
	return b.parseQualifiedIdentifierNode(typeRefOrPkgRef, isInConditionalExpr)
}

func (b *BallerinaParser) parseQualifiedIdentifierNode(identifier tree.STNode, isInConditionalExpr bool) tree.STNode {
	nextToken := b.peekN(1)
	if nextToken.Kind() != common.COLON_TOKEN {
		return tree.CreateSimpleNameReferenceNode(identifier)
	}
	if isInConditionalExpr && (b.hasTrailingMinutiae(identifier) || b.hasTrailingMinutiae(nextToken)) {
		return tree.GetSimpleNameRefNode(identifier)
	}
	nextNextToken := b.peekN(2)
	switch nextNextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		colon := b.consume()
		varOrFuncName := b.consume()
		return b.createQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
	case common.COLON_TOKEN:
		b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
		return b.parseQualifiedIdentifierNode(identifier, isInConditionalExpr)
	default:
		if (nextNextToken.Kind() == common.MAP_KEYWORD) && (b.peekN(3).Kind() != common.LT_TOKEN) {
			colon := b.consume()
			mapKeyword := b.consume()
			refName := tree.CreateIdentifierTokenWithDiagnostics(mapKeyword.Text(),
				mapKeyword.LeadingMinutiae(), mapKeyword.TrailingMinutiae(), mapKeyword.Diagnostics())
			return b.createQualifiedNameReferenceNode(identifier, colon, refName)
		}
		if isInConditionalExpr {
			return tree.GetSimpleNameRefNode(identifier)
		}
		colon := b.consume()
		varOrFuncName := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_IDENTIFIER)
		return b.createQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
	}
}

func (b *BallerinaParser) createQualifiedNameReferenceNode(identifier tree.STNode, colon tree.STNode, varOrFuncName tree.STNode) tree.STNode {
	if b.hasTrailingMinutiae(identifier) || b.hasTrailingMinutiae(colon) {
		colon = tree.AddDiagnostic(colon,
			&common.ERROR_INTERVENING_WHITESPACES_ARE_NOT_ALLOWED)
	}
	return tree.CreateQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
}

func (b *BallerinaParser) parseFieldOrRestDescriptorRhs(metadata tree.STNode, ty tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ELLIPSIS_TOKEN:
		b.reportInvalidMetaData(metadata, "record rest descriptor")
		ellipsis := b.parseEllipsis()
		semicolonToken := b.parseSemicolon()
		return tree.CreateRecordRestDescriptorNode(ty, ellipsis, semicolonToken)
	case common.IDENTIFIER_TOKEN:
		readonlyQualifier := tree.CreateEmptyNode()
		return b.parseIndividualRecordField(metadata, readonlyQualifier, ty)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS)
		return b.parseFieldOrRestDescriptorRhs(metadata, ty)
	}
}

func (b *BallerinaParser) parseFieldDescriptorRhs(metadata tree.STNode, readonlyQualifier tree.STNode, ty tree.STNode, fieldName tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.SEMICOLON_TOKEN:
		questionMarkToken := tree.CreateEmptyNode()
		semicolonToken := b.parseSemicolon()
		return tree.CreateRecordFieldNode(metadata, readonlyQualifier, ty, fieldName,
			questionMarkToken, semicolonToken)
	case common.QUESTION_MARK_TOKEN:
		questionMarkToken := b.parseQuestionMark()
		semicolonToken := b.parseSemicolon()
		return tree.CreateRecordFieldNode(metadata, readonlyQualifier, ty, fieldName,
			questionMarkToken, semicolonToken)
	case common.EQUAL_TOKEN:
		equalsToken := b.parseAssignOp()
		expression := b.parseExpression()
		semicolonToken := b.parseSemicolon()
		return tree.CreateRecordFieldWithDefaultValueNode(metadata, readonlyQualifier, ty, fieldName,
			equalsToken, expression, semicolonToken)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS)
		return b.parseFieldDescriptorRhs(metadata, readonlyQualifier, ty, fieldName)
	}
}

func (b *BallerinaParser) parseQuestionMark() tree.STNode {
	token := b.peek()
	if token.Kind() == common.QUESTION_MARK_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_QUESTION_MARK)
		return b.parseQuestionMark()
	}
}

func (b *BallerinaParser) parseStatements() tree.STNode {
	res, _ := b.parseStatementsInner(nil)
	return res
}

func (b *BallerinaParser) parseStatementsInner(stmts []tree.STNode) (tree.STNode, []tree.STNode) {
	for !b.isEndOfStatements() {
		stmt := b.parseStatement()
		if stmt == nil {
			break
		}
		if stmt.Kind() == common.NAMED_WORKER_DECLARATION {
			b.addInvalidNodeToNextToken(stmt, &common.ERROR_NAMED_WORKER_NOT_ALLOWED_HERE)
			continue
		}
		if b.validateStatement(stmt) {
			continue
		}
		stmts = append(stmts, stmt)
	}
	return tree.CreateNodeList(stmts...), stmts
}

func (b *BallerinaParser) parseStatement() tree.STNode {
	nextToken := b.peek()
	annots := tree.CreateEmptyNodeList()
	switch nextToken.Kind() {
	case common.CLOSE_BRACE_TOKEN, common.EOF_TOKEN:
		return nil
	case common.SEMICOLON_TOKEN:
		b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
		return b.parseStatement()
	case common.AT_TOKEN:
		annots = b.parseOptionalAnnotations()
	default:
		if b.isStatementStartingToken(nextToken.Kind()) {
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STATEMENT)
		if solution.Action == ACTION_KEEP {
			break
		}
		return b.parseStatement()
	}
	return b.parseStatementWithAnnotataions(annots)
}

func (b *BallerinaParser) validateStatement(statement tree.STNode) bool {
	switch statement.Kind() {
	case common.LOCAL_TYPE_DEFINITION_STATEMENT:
		b.addInvalidNodeToNextToken(statement, &common.ERROR_LOCAL_TYPE_DEFINITION_NOT_ALLOWED)
		return true
	case common.CONST_DECLARATION:
		b.addInvalidNodeToNextToken(statement, &common.ERROR_LOCAL_CONST_DECL_NOT_ALLOWED)
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) getAnnotations(nullbaleAnnot tree.STNode) tree.STNode {
	if nullbaleAnnot != nil {
		return nullbaleAnnot
	}
	return tree.CreateEmptyNodeList()
}

func (b *BallerinaParser) parseStatementWithAnnotataions(annots tree.STNode) tree.STNode {
	result, _ := b.parseStatementInner(annots, nil)
	return result
}

func (b *BallerinaParser) parseStatementInner(annots tree.STNode, qualifiers []tree.STNode) (tree.STNode, []tree.STNode) {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		return b.parseStmtStartsWithTypeOrExpr(b.getAnnotations(annots), qualifiers), qualifiers
	}
	switch nextToken.Kind() {
	case common.CLOSE_BRACE_TOKEN,
		common.EOF_TOKEN:
		publicQualifier := tree.CreateEmptyNode()
		return b.createMissingSimpleVarDeclInnerWithQualifiers(b.getAnnotations(annots), publicQualifier, qualifiers, false), qualifiers
	case common.SEMICOLON_TOKEN:
		b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
		return b.parseStatementInner(annots, qualifiers)
	case common.FINAL_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		finalKeyword := b.consume()
		return b.parseVariableDecl(b.getAnnotations(annots), finalKeyword), qualifiers
	case common.IF_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseIfElseBlock(), qualifiers
	case common.WHILE_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseWhileStatement(), qualifiers
	case common.DO_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseDoStatement(), qualifiers
	case common.PANIC_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parsePanicStatement(), qualifiers
	case common.CONTINUE_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseContinueStatement(), qualifiers
	case common.BREAK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseBreakStatement(), qualifiers
	case common.RETURN_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseReturnStatement(), qualifiers
	case common.FAIL_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseFailStatement(), qualifiers
	case common.TYPE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseLocalTypeDefinitionStatement(b.getAnnotations(annots)), qualifiers
	case common.CONST_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseConstantDeclaration(annots, tree.CreateEmptyNode()), qualifiers
	case common.LOCK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseLockStatement(), qualifiers
	case common.OPEN_BRACE_TOKEN:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStatementStartsWithOpenBrace(), qualifiers
	case common.WORKER_KEYWORD:
		return b.parseNamedWorkerDeclaration(b.getAnnotations(annots), qualifiers), qualifiers
	case common.FORK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseForkStatement(), qualifiers
	case common.FOREACH_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseForEachStatement(), qualifiers
	case common.START_KEYWORD,
		common.CHECK_KEYWORD,
		common.CHECKPANIC_KEYWORD,
		common.TRAP_KEYWORD,
		common.FLUSH_KEYWORD,
		common.LEFT_ARROW_TOKEN,
		common.WAIT_KEYWORD,
		common.FROM_KEYWORD,
		common.COMMIT_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseExpressionStatement(b.getAnnotations(annots)), qualifiers
	case common.XMLNS_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseXMLNamespaceDeclaration(false), qualifiers
	case common.TRANSACTION_KEYWORD:
		return b.parseTransactionStmtOrVarDecl(annots, qualifiers, b.consume())
	case common.RETRY_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRetryStatement(), qualifiers
	case common.ROLLBACK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRollbackStatement(), qualifiers
	case common.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStatementStartsWithOpenBracket(b.getAnnotations(annots), false), qualifiers
	case common.FUNCTION_KEYWORD,
		common.OPEN_PAREN_TOKEN,
		common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.STRING_KEYWORD,
		common.XML_KEYWORD:
		return b.parseStmtStartsWithTypeOrExpr(b.getAnnotations(annots), qualifiers), qualifiers
	case common.MATCH_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMatchStatement(), qualifiers
	case common.ERROR_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseErrorTypeDescOrErrorBP(b.getAnnotations(annots)), qualifiers
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseStatementStartWithExpr(b.getAnnotations(annots)), qualifiers
		}
		if b.isTypeStartingToken(nextToken.Kind()) {
			publicQualifier := tree.CreateEmptyNode()
			res, _ := b.parseVariableDeclInner(b.getAnnotations(annots), publicQualifier, nil, qualifiers,
				false)
			return res, qualifiers
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS)
		if solution.Action == ACTION_KEEP {
			b.reportInvalidQualifierList(qualifiers)
			finalKeyword := tree.CreateEmptyNode()
			return b.parseVariableDecl(b.getAnnotations(annots), finalKeyword), qualifiers
		}
		return b.parseStatementInner(annots, qualifiers)
	}
}

func (b *BallerinaParser) parseVariableDecl(annots tree.STNode, finalKeyword tree.STNode) tree.STNode {
	var typeDescQualifiers []tree.STNode
	var varDecQualifiers []tree.STNode
	if finalKeyword != nil {
		varDecQualifiers = append(varDecQualifiers, finalKeyword)
	}
	publicQualifier := tree.CreateEmptyNode()
	res, _ := b.parseVariableDeclInner(annots, publicQualifier, varDecQualifiers, typeDescQualifiers, false)
	return res
}

// Return result, and modified varDeclQuals
func (b *BallerinaParser) parseVariableDeclInner(annots tree.STNode, publicQualifier tree.STNode, varDeclQuals []tree.STNode, typeDescQualifiers []tree.STNode, isModuleVar bool) (tree.STNode, []tree.STNode) {
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	typeBindingPattern := b.parseTypedBindingPatternInner(typeDescQualifiers,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	return b.parseVarDeclRhsInner(annots, publicQualifier, varDeclQuals, typeBindingPattern, isModuleVar)
}

// Return result, and modified qualifiers
func (b *BallerinaParser) parseVarDeclTypeDescRhs(typeDesc tree.STNode, metadata tree.STNode, qualifiers []tree.STNode, isTypedBindingPattern bool, isModuleVar bool) (tree.STNode, []tree.STNode) {
	publicQualifier := tree.CreateEmptyNode()
	return b.parseVarDeclTypeDescRhsInner(typeDesc, metadata, publicQualifier, qualifiers, isTypedBindingPattern,
		isModuleVar)
}

// Return result, and modified qualifiers
func (b *BallerinaParser) parseVarDeclTypeDescRhsInner(typeDesc tree.STNode, metadata tree.STNode, publicQual tree.STNode, qualifiers []tree.STNode, isTypedBindingPattern bool, isModuleVar bool) (tree.STNode, []tree.STNode) {
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	typeDesc = b.parseComplexTypeDescriptor(typeDesc,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, isTypedBindingPattern)
	typedBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	return b.parseVarDeclRhsInner(metadata, publicQual, qualifiers, typedBindingPattern, isModuleVar)
}

// Return result, and modified varDeclQuals
func (b *BallerinaParser) parseVarDeclRhs(metadata tree.STNode, varDeclQuals []tree.STNode, typedBindingPattern tree.STNode, isModuleVar bool) (tree.STNode, []tree.STNode) {
	publicQualifier := tree.CreateEmptyNode()
	return b.parseVarDeclRhsInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, isModuleVar)
}

// Return result, and modified varDeclQuals
func (b *BallerinaParser) parseVarDeclRhsInner(metadata tree.STNode, publicQualifier tree.STNode, varDeclQuals []tree.STNode, typedBindingPattern tree.STNode, isModuleVar bool) (tree.STNode, []tree.STNode) {
	var assign tree.STNode
	var expr tree.STNode
	var semicolon tree.STNode
	hasVarInit := false
	isConfigurable := isModuleVar && b.isSyntaxKindInList(varDeclQuals, common.CONFIGURABLE_KEYWORD)

	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.EQUAL_TOKEN:
		assign = b.parseAssignOp()
		if isModuleVar {
			if isConfigurable {
				expr = b.parseConfigurableVarDeclRhs()
			} else {
				expr = b.parseExpression()
			}
		} else {
			expr = b.parseActionOrExpression()
		}
		semicolon = b.parseSemicolon()
		hasVarInit = true
	case common.SEMICOLON_TOKEN:
		assign = tree.CreateEmptyNode()
		expr = tree.CreateEmptyNode()
		semicolon = b.parseSemicolon()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS)
		return b.parseVarDeclRhsInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, isModuleVar)
	}
	b.endContext()
	if !hasVarInit {
		typedBindingPatternNode, ok := typedBindingPattern.(*tree.STTypedBindingPatternNode)
		if !ok {
			panic("expected STTypedBindingPatternNode")
		}
		bindingPatternKind := typedBindingPatternNode.BindingPattern.Kind()
		if bindingPatternKind != common.CAPTURE_BINDING_PATTERN {
			assign = tree.CreateMissingTokenWithDiagnostics(common.EQUAL_TOKEN,
				&common.ERROR_VARIABLE_DECL_HAVING_BP_MUST_BE_INITIALIZED)
			identifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
			expr = tree.CreateSimpleNameReferenceNode(identifier)
		}
	}
	if isModuleVar {
		return b.createModuleVarDeclaration(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign,
			expr, semicolon, isConfigurable, hasVarInit)
	}
	var finalKeyword tree.STNode
	if len(varDeclQuals) == 0 {
		finalKeyword = tree.CreateEmptyNode()
	} else {
		finalKeyword = varDeclQuals[0]
	}
	if metadata.Kind() != common.LIST {
		panic("assertion failed")
	}
	return tree.CreateVariableDeclarationNode(metadata, finalKeyword, typedBindingPattern, assign,
		expr, semicolon), varDeclQuals
}

func (b *BallerinaParser) parseConfigurableVarDeclRhs() tree.STNode {
	var expr tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.QUESTION_MARK_TOKEN:
		expr = tree.CreateRequiredExpressionNode(b.consume())
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			expr = b.parseExpression()
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_CONFIG_VAR_DECL_RHS)
		return b.parseConfigurableVarDeclRhs()
	}
	return expr
}

func (b *BallerinaParser) createModuleVarDeclaration(metadata tree.STNode, publicQualifier tree.STNode, varDeclQuals []tree.STNode, typedBindingPattern tree.STNode, assign tree.STNode, expr tree.STNode, semicolon tree.STNode, isConfigurable bool, hasVarInit bool) (tree.STNode, []tree.STNode) {
	if hasVarInit || len(varDeclQuals) == 0 {
		return b.createModuleVarDeclarationInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign,
			expr, semicolon), varDeclQuals
	}
	if isConfigurable {
		return b.createConfigurableModuleVarDeclWithMissingInitializer(metadata, publicQualifier, varDeclQuals,
			typedBindingPattern, semicolon), varDeclQuals
	}
	lastQualifier := b.getLastNodeInList(varDeclQuals)
	if lastQualifier.Kind() == common.ISOLATED_KEYWORD {
		lastQualifier = varDeclQuals[len(varDeclQuals)-1]
		varDeclQuals = varDeclQuals[:len(varDeclQuals)-1]
		typedBindingPattern = b.modifyTypedBindingPatternWithIsolatedQualifier(typedBindingPattern, lastQualifier)
	}
	return b.createModuleVarDeclarationInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign, expr,
		semicolon), varDeclQuals
}

func (b *BallerinaParser) createConfigurableModuleVarDeclWithMissingInitializer(metadata tree.STNode, publicQualifier tree.STNode, varDeclQuals []tree.STNode, typedBindingPattern tree.STNode, semicolon tree.STNode) tree.STNode {
	var assign tree.STNode
	assign = tree.CreateMissingToken(common.EQUAL_TOKEN, nil)
	assign = tree.AddDiagnostic(assign,
		&common.ERROR_CONFIGURABLE_VARIABLE_MUST_BE_INITIALIZED_OR_REQUIRED)
	questionMarkToken := tree.CreateMissingToken(common.QUESTION_MARK_TOKEN, nil)
	expr := tree.CreateRequiredExpressionNode(questionMarkToken)
	return b.createModuleVarDeclarationInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign, expr,
		semicolon)
}

func (b *BallerinaParser) createModuleVarDeclarationInner(metadata tree.STNode, publicQualifier tree.STNode, varDeclQuals []tree.STNode, typedBindingPattern tree.STNode, assign tree.STNode, expr tree.STNode, semicolon tree.STNode) tree.STNode {
	if publicQualifier != nil {
		typedBindingPatternNode, ok := typedBindingPattern.(*tree.STTypedBindingPatternNode)
		if !ok {
			panic("expected STTypedBindingPatternNode")
		}
		if typedBindingPatternNode.TypeDescriptor.Kind() == common.VAR_TYPE_DESC {
			if len(varDeclQuals) != 0 {
				b.updateFirstNodeInListWithLeadingInvalidNode(varDeclQuals, publicQualifier,
					&common.ERROR_VARIABLE_DECLARED_WITH_VAR_CANNOT_BE_PUBLIC)
			} else {
				typedBindingPattern = tree.CloneWithLeadingInvalidNodeMinutiae(typedBindingPattern,
					publicQualifier, &common.ERROR_VARIABLE_DECLARED_WITH_VAR_CANNOT_BE_PUBLIC)
			}
			publicQualifier = tree.CreateEmptyNode()
		} else if b.isSyntaxKindInList(varDeclQuals, common.ISOLATED_KEYWORD) {
			b.updateFirstNodeInListWithLeadingInvalidNode(varDeclQuals, publicQualifier,
				&common.ERROR_ISOLATED_VAR_CANNOT_BE_DECLARED_AS_PUBLIC)
			publicQualifier = tree.CreateEmptyNode()
		}
	}
	varDeclQualifiersNode := tree.CreateNodeList(varDeclQuals...)
	return tree.CreateModuleVariableDeclarationNode(metadata, publicQualifier, varDeclQualifiersNode,
		typedBindingPattern, assign, expr, semicolon)
}

func (b *BallerinaParser) createMissingSimpleVarDecl(isModuleVar bool) tree.STNode {
	var metadata tree.STNode
	if isModuleVar {
		metadata = tree.CreateEmptyNode()
	} else {
		metadata = tree.CreateEmptyNodeList()
	}
	return b.createMissingSimpleVarDeclInner(metadata, isModuleVar)
}

func (b *BallerinaParser) createMissingSimpleVarDeclInner(metadata tree.STNode, isModuleVar bool) tree.STNode {
	publicQualifier := tree.CreateEmptyNode()
	return b.createMissingSimpleVarDeclInnerWithQualifiers(metadata, publicQualifier, nil, isModuleVar)
}

func (b *BallerinaParser) createMissingSimpleVarDeclInnerWithQualifiers(metadata tree.STNode, publicQualifier tree.STNode, qualifiers []tree.STNode, isModuleVar bool) tree.STNode {
	emptyNode := tree.CreateEmptyNode()
	simpleTypeDescIdentifier := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_TYPE_DESC)
	identifier := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_VARIABLE_NAME)
	simpleNameRef := tree.CreateSimpleNameReferenceNode(simpleTypeDescIdentifier)
	semicolon := tree.CreateMissingTokenWithDiagnostics(common.SEMICOLON_TOKEN,
		&common.ERROR_MISSING_SEMICOLON_TOKEN)
	captureBP := tree.CreateCaptureBindingPatternNode(identifier)
	typedBindingPattern := tree.CreateTypedBindingPatternNode(simpleNameRef, captureBP)
	if isModuleVar {
		varDeclQuals, qualifiers := b.extractVarDeclQualifiers(qualifiers, true)
		typedBindingPattern = b.modifyNodeWithInvalidTokenList(qualifiers, typedBindingPattern)
		if b.isSyntaxKindInList(varDeclQuals, common.CONFIGURABLE_KEYWORD) {
			return b.createConfigurableModuleVarDeclWithMissingInitializer(metadata, publicQualifier, varDeclQuals,
				typedBindingPattern, semicolon)
		}
		varDeclQualNodeList := tree.CreateNodeList(varDeclQuals...)
		return tree.CreateModuleVariableDeclarationNode(metadata, publicQualifier, varDeclQualNodeList,
			typedBindingPattern, emptyNode, emptyNode, semicolon)
	}
	typedBindingPattern = b.modifyNodeWithInvalidTokenList(qualifiers, typedBindingPattern)
	return tree.CreateVariableDeclarationNode(metadata, emptyNode, typedBindingPattern, emptyNode,
		emptyNode, semicolon)
}

func (b *BallerinaParser) createMissingWhereClause() tree.STNode {
	whereKeyword := tree.CreateMissingTokenWithDiagnostics(common.WHERE_KEYWORD,
		&common.ERROR_MISSING_WHERE_KEYWORD)
	missingIdentifier := tree.CreateMissingTokenWithDiagnostics(
		common.IDENTIFIER_TOKEN, &common.ERROR_MISSING_EXPRESSION)
	missingExpr := tree.CreateSimpleNameReferenceNode(missingIdentifier)
	return tree.CreateWhereClauseNode(whereKeyword, missingExpr)
}

func (b *BallerinaParser) createMissingSimpleObjectFieldInner(metadata tree.STNode, qualifiers []tree.STNode, isObjectTypeDesc bool) (tree.STNode, []tree.STNode) {
	emptyNode := tree.CreateEmptyNode()
	simpleTypeDescIdentifier := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_TYPE_DESC)
	simpleNameRef := tree.CreateSimpleNameReferenceNode(simpleTypeDescIdentifier)
	identifier := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_FIELD_NAME)
	semicolon := tree.CreateMissingTokenWithDiagnostics(common.SEMICOLON_TOKEN,
		&common.ERROR_MISSING_SEMICOLON_TOKEN)
	objectFieldQualifiers, qualifiers := b.extractObjectFieldQualifiers(qualifiers, isObjectTypeDesc)
	objectFieldQualNodeList := tree.CreateNodeList(objectFieldQualifiers...)
	simpleNameRef = b.modifyNodeWithInvalidTokenList(qualifiers, simpleNameRef)
	metadataNode, ok := metadata.(*tree.STMetadataNode)
	if !ok {
		panic("expected STMetadataNode")
	}
	if metadata != nil {
		metadata = b.addMetadataNotAttachedDiagnostic(*metadataNode)
	}
	return tree.CreateObjectFieldNode(metadata, emptyNode, objectFieldQualNodeList,
		simpleNameRef, identifier, emptyNode, emptyNode, semicolon), qualifiers
}

func (b *BallerinaParser) createMissingSimpleObjectField() tree.STNode {
	metadata := tree.CreateEmptyNode()
	res, _ := b.createMissingSimpleObjectFieldInner(metadata, nil, false)
	return res
}

func (b *BallerinaParser) modifyNodeWithInvalidTokenList(qualifiers []tree.STNode, node tree.STNode) tree.STNode {
	i := (len(qualifiers) - 1)
	for ; i >= 0; i-- {
		qualifier := qualifiers[i]
		node = tree.CloneWithLeadingInvalidNodeMinutiae(node, qualifier, nil)
	}
	return node
}

func (b *BallerinaParser) modifyTypedBindingPatternWithIsolatedQualifier(typedBindingPattern tree.STNode, isolatedQualifier tree.STNode) tree.STNode {
	typedBindingPatternNode, ok := typedBindingPattern.(*tree.STTypedBindingPatternNode)
	if !ok {
		panic("expected STTypedBindingPatternNode")
	}
	typeDescriptor := typedBindingPatternNode.TypeDescriptor
	bindingPattern := typedBindingPatternNode.BindingPattern
	switch typeDescriptor.Kind() {
	case common.OBJECT_TYPE_DESC:
		typeDescriptor = b.modifyObjectTypeDescWithALeadingQualifier(typeDescriptor, isolatedQualifier)
	case common.FUNCTION_TYPE_DESC:
		typeDescriptor = b.modifyFuncTypeDescWithALeadingQualifier(typeDescriptor, isolatedQualifier)
	default:
		typeDescriptor = tree.CloneWithLeadingInvalidNodeMinutiae(typeDescriptor, isolatedQualifier,
			&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(isolatedQualifier).Text())
	}
	return tree.CreateTypedBindingPatternNode(typeDescriptor, bindingPattern)
}

func (b *BallerinaParser) modifyObjectTypeDescWithALeadingQualifier(objectTypeDesc tree.STNode, newQualifier tree.STNode) tree.STNode {
	objectTypeDescriptorNode, ok := objectTypeDesc.(*tree.STObjectTypeDescriptorNode)
	if !ok {
		panic("expected STObjectTypeDescriptorNode")
	}

	qualifierList, ok := objectTypeDescriptorNode.ObjectTypeQualifiers.(*tree.STNodeList)
	if !ok {
		panic("expected STNodeList")
	}
	newObjectTypeQualifiers := b.modifyNodeListWithALeadingQualifier(qualifierList, newQualifier)
	return tree.CreateObjectTypeDescriptorNode(newObjectTypeQualifiers, objectTypeDescriptorNode.ObjectKeyword,
		objectTypeDescriptorNode.OpenBrace, objectTypeDescriptorNode.Members,
		objectTypeDescriptorNode.CloseBrace)
}

func (b *BallerinaParser) modifyFuncTypeDescWithALeadingQualifier(funcTypeDesc tree.STNode, newQualifier tree.STNode) tree.STNode {
	funcTypeDescriptorNode, ok := funcTypeDesc.(*tree.STFunctionTypeDescriptorNode)
	if !ok {
		panic("expected STFunctionTypeDescriptorNode")
	}
	qualifierList := funcTypeDescriptorNode.QualifierList
	newfuncTypeQualifiers := b.modifyNodeListWithALeadingQualifier(qualifierList, newQualifier)
	return tree.CreateFunctionTypeDescriptorNode(newfuncTypeQualifiers, funcTypeDescriptorNode.FunctionKeyword,
		funcTypeDescriptorNode.FunctionSignature)
}

func (b *BallerinaParser) modifyNodeListWithALeadingQualifier(qualifiers tree.STNode, newQualifier tree.STNode) tree.STNode {
	var newQualifierList []tree.STNode
	newQualifierList = append(newQualifierList, newQualifier)
	qualifierNodeList, ok := qualifiers.(*tree.STNodeList)
	if !ok {
		panic("expected STNodeList")
	}
	i := 0
	for ; i < qualifierNodeList.Size(); i++ {
		qualifier := qualifierNodeList.Get(i)
		if qualifier.Kind() == newQualifier.Kind() {
			b.updateLastNodeInListWithInvalidNode(newQualifierList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(qualifier).Text())
		} else {
			newQualifierList = append(newQualifierList, qualifier)
		}
	}
	return tree.CreateNodeList(newQualifierList...)
}

func (b *BallerinaParser) parseAssignmentStmtRhs(lvExpr tree.STNode) tree.STNode {
	assign := b.parseAssignOp()
	expr := b.parseActionOrExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	if lvExpr.Kind() == common.ERROR_CONSTRUCTOR {
		errConstructor, ok := lvExpr.(*tree.STErrorConstructorExpressionNode)
		if !ok {
			panic("expected STErrorConstructorExpressionNode")
		}
		if b.isPossibleErrorBindingPattern(*errConstructor) {
			lvExpr = b.getBindingPattern(lvExpr, false)
		}
	}
	if b.isWildcardBP(lvExpr) {
		lvExpr = b.getWildcardBindingPattern(lvExpr)
	}
	lvExprValid := b.isValidLVExpr(lvExpr)
	if !lvExprValid {
		identifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		simpleNameRef := tree.CreateSimpleNameReferenceNode(identifier)
		lvExpr = tree.CloneWithLeadingInvalidNodeMinutiae(simpleNameRef, lvExpr,
			&common.ERROR_INVALID_EXPR_IN_ASSIGNMENT_LHS)
	}
	return tree.CreateAssignmentStatementNode(lvExpr, assign, expr, semicolon)
}

func (b *BallerinaParser) parseExpression() tree.STNode {
	return b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_DEFAULT, true, false)
}

func (b *BallerinaParser) parseActionOrExpression() tree.STNode {
	return b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_DEFAULT, true, true)
}

func (b *BallerinaParser) parseActionOrExpressionInLhs(annots tree.STNode) tree.STNode {
	return b.parseExpressionInner(OPERATOR_PRECEDENCE_DEFAULT, annots, false, true, false)
}

func (b *BallerinaParser) parseExpressionPossibleRhsExpr(isRhsExpr bool) tree.STNode {
	return b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_DEFAULT, isRhsExpr, false)
}

func (b *BallerinaParser) isValidLVExpr(expression tree.STNode) bool {
	switch expression.Kind() {
	case common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE,
		common.LIST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.ERROR_BINDING_PATTERN,
		common.WILDCARD_BINDING_PATTERN:
		return true
	case common.FIELD_ACCESS:
		fieldAccessExpressionNode, ok := expression.(*tree.STFieldAccessExpressionNode)
		if !ok {
			panic("expected STFieldAccessExpressionNode")
		}
		return b.isValidLVMemberExpr(fieldAccessExpressionNode.Expression)
	case common.INDEXED_EXPRESSION:
		indexedExpressionNode, ok := expression.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("expected STIndexedExpressionNode")
		}
		return b.isValidLVMemberExpr(indexedExpressionNode.ContainerExpression)
	default:
		_, ok := expression.(*tree.STMissingToken)
		return ok
	}
}

func (b *BallerinaParser) isValidLVMemberExpr(expression tree.STNode) bool {
	switch expression.Kind() {
	case common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE:
		return true
	case common.FIELD_ACCESS:
		fieldAccessExpressionNode, ok := expression.(*tree.STFieldAccessExpressionNode)
		if !ok {
			panic("expected STFieldAccessExpressionNode")
		}
		return b.isValidLVMemberExpr(fieldAccessExpressionNode.Expression)
	case common.INDEXED_EXPRESSION:
		indexedExpressionNode, ok := expression.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("expected STIndexedExpressionNode")
		}
		return b.isValidLVMemberExpr(indexedExpressionNode.ContainerExpression)
	case common.BRACED_EXPRESSION:
		bracedExpressionNode, ok := expression.(*tree.STBracedExpressionNode)
		if !ok {
			panic("expected STBracedExpressionNode")
		}
		return b.isValidLVMemberExpr(bracedExpressionNode.Expression)
	default:
		_, ok := expression.(*tree.STMissingToken)
		return ok
	}
}

func (b *BallerinaParser) parseExpressionWithPrecedence(precedenceLevel OperatorPrecedence, isRhsExpr bool, allowActions bool) tree.STNode {
	return b.parseExpressionWithConditional(precedenceLevel, isRhsExpr, allowActions, false)
}

func (b *BallerinaParser) parseExpressionWithConditional(precedenceLevel OperatorPrecedence, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	return b.parseExpressionWithMatchGuard(precedenceLevel, isRhsExpr, allowActions, false, isInConditionalExpr)
}

func (b *BallerinaParser) parseExpressionWithMatchGuard(precedenceLevel OperatorPrecedence, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) tree.STNode {
	expr := b.parseTerminalExpression(isRhsExpr, allowActions, isInConditionalExpr)
	return b.parseExpressionRhsInner(precedenceLevel, expr, isRhsExpr, allowActions, isInMatchGuard, isInConditionalExpr)
}

func (b *BallerinaParser) invalidateActionAndGetMissingExpr(node tree.STNode) tree.STNode {
	var identifier tree.STNode
	identifier = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
	identifier = tree.CloneWithTrailingInvalidNodeMinutiae(identifier, node, &common.ERROR_EXPRESSION_EXPECTED_ACTION_FOUND)
	return tree.CreateSimpleNameReferenceNode(identifier)
}

func (b *BallerinaParser) parseExpressionInner(precedenceLevel OperatorPrecedence, annots tree.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	expr := b.parseTerminalExpressionWithAnnotations(annots, isRhsExpr, allowActions, isInConditionalExpr)
	return b.parseExpressionRhsInner(precedenceLevel, expr, isRhsExpr, allowActions, false, isInConditionalExpr)
}

func (b *BallerinaParser) parseTerminalExpression(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	annots := tree.CreateEmptyNodeList()
	if b.peek().Kind() == common.AT_TOKEN {
		annots = b.parseOptionalAnnotations()
	}
	return b.parseTerminalExpressionWithAnnotations(annots, isRhsExpr, allowActions, isInConditionalExpr)
}

func (b *BallerinaParser) parseTerminalExpressionWithAnnotations(annots tree.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	return b.parseTerminalExpressionInner(annots, nil, isRhsExpr, allowActions, isInConditionalExpr)
}

func (b *BallerinaParser) parseTerminalExpressionInner(annots tree.STNode, qualifiers []tree.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	qualifiers = b.parseExprQualifiers(qualifiers)
	nextToken := b.peek()
	annotNodeList := annots.(*tree.STNodeList)
	if (!annotNodeList.IsEmpty()) && (!b.isAnnotAllowedExprStart(nextToken)) {
		annots = b.addAnnotNotAttachedDiagnostic(annotNodeList)
		qualifierNodeList := b.createObjectTypeQualNodeList(qualifiers)
		return b.createMissingObjectConstructor(annots, qualifierNodeList)
	}
	b.validateExprAnnotsAndQualifiers(nextToken, annots, qualifiers)
	if b.isQualifiedIdentifierPredeclaredPrefix(nextToken.Kind()) {
		return b.parseQualifiedIdentifierOrExpression(isInConditionalExpr, isRhsExpr, allowActions)
	}
	switch nextToken.Kind() {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		return b.parseBasicLiteral()
	case common.OPEN_PAREN_TOKEN:
		return b.parseBracedExpression(isRhsExpr, allowActions)
	case common.CHECK_KEYWORD,
		common.CHECKPANIC_KEYWORD:
		return b.parseCheckExpression(isRhsExpr, allowActions, isInConditionalExpr)
	case common.OPEN_BRACE_TOKEN:
		return b.parseMappingConstructorExpr()
	case common.TYPEOF_KEYWORD:
		return b.parseTypeofExpression(isRhsExpr, isInConditionalExpr)
	case common.PLUS_TOKEN, common.MINUS_TOKEN, common.NEGATION_TOKEN, common.EXCLAMATION_MARK_TOKEN:
		return b.parseUnaryExpression(isRhsExpr, isInConditionalExpr)
	case common.TRAP_KEYWORD:
		return b.parseTrapExpression(isRhsExpr, allowActions, isInConditionalExpr)
	case common.OPEN_BRACKET_TOKEN:
		return b.parseListConstructorExpr()
	case common.LT_TOKEN:
		return b.parseTypeCastExpr(isRhsExpr, allowActions, isInConditionalExpr)
	case common.TABLE_KEYWORD, common.STREAM_KEYWORD, common.FROM_KEYWORD, common.MAP_KEYWORD:
		return b.parseTableConstructorOrQuery(isRhsExpr, allowActions)
	case common.ERROR_KEYWORD:
		return b.parseErrorConstructorExpr(b.consume())
	case common.LET_KEYWORD:
		return b.parseLetExpression(isRhsExpr, isInConditionalExpr)
	case common.BACKTICK_TOKEN:
		return b.parseTemplateExpression()
	case common.OBJECT_KEYWORD:
		return b.parseObjectConstructorExpression(annots, qualifiers)
	case common.XML_KEYWORD:
		return b.parseXMLTemplateExpression()
	case common.RE_KEYWORD:
		return b.parseRegExpTemplateExpression()
	case common.STRING_KEYWORD:
		nextNextToken := b.getNextNextToken()
		if nextNextToken.Kind() == common.BACKTICK_TOKEN {
			return b.parseStringTemplateExpression()
		}
		return b.parseSimpleTypeInTerminalExpr()
	case common.FUNCTION_KEYWORD:
		return b.parseExplicitFunctionExpression(annots, qualifiers, isRhsExpr)
	case common.NEW_KEYWORD:
		return b.parseNewExpression()
	case common.START_KEYWORD:
		return b.parseStartAction(annots)
	case common.FLUSH_KEYWORD:
		return b.parseFlushAction()
	case common.LEFT_ARROW_TOKEN:
		return b.parseReceiveAction()
	case common.WAIT_KEYWORD:
		return b.parseWaitAction()
	case common.COMMIT_KEYWORD:
		return b.parseCommitAction()
	case common.TRANSACTIONAL_KEYWORD:
		return b.parseTransactionalExpression()
	case common.BASE16_KEYWORD,
		common.BASE64_KEYWORD:
		return b.parseByteArrayLiteral()
	case common.TRANSACTION_KEYWORD:
		return b.parseQualifiedIdentWithTransactionPrefix(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
	case common.IDENTIFIER_TOKEN:
		if b.isNaturalKeyword(nextToken) && (b.getNextNextToken().Kind() == common.OPEN_BRACE_TOKEN) {
			return b.parseNaturalExpression()
		}
		return b.parseQualifiedIdentifierInner(common.PARSER_RULE_CONTEXT_VARIABLE_REF, isInConditionalExpr)
	case common.CONST_KEYWORD:
		if b.isNaturalKeyword(b.getNextNextToken()) {
			return b.parseNaturalExpression()
		}
		fallthrough
	default:
		if b.isSimpleTypeInExpression(nextToken.Kind()) {
			return b.parseSimpleTypeInTerminalExpr()
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION)
		return b.parseTerminalExpressionInner(annots, qualifiers, isRhsExpr, allowActions, isInConditionalExpr)
	}
}

func (b *BallerinaParser) parseNaturalExpression() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION)
	var optionalConstKeyword tree.STNode
	if b.peek().Kind() == common.CONST_KEYWORD {
		optionalConstKeyword = b.consume()
	} else {
		optionalConstKeyword = tree.CreateEmptyNode()
	}
	naturalKeyword := b.parseNaturalKeyword()
	optionalParenthesizedArgList := b.parseOptionalParenthesizedArgList()
	return b.parseNaturalExprBody(optionalConstKeyword, naturalKeyword, optionalParenthesizedArgList)
}

func (b *BallerinaParser) parseNaturalExprBody(optionalConstKeyword tree.STNode, naturalKeyword tree.STNode, optionalParenthesizedArgList tree.STNode) tree.STNode {
	openBrace := b.parseOpenBrace()
	if openBrace.IsMissing() {
		b.endContext()
		return b.createMissingNaturalExpressionNode(optionalConstKeyword, naturalKeyword,
			optionalParenthesizedArgList)
	}
	b.tokenReader.StartMode(PARSER_MODE_PROMPT)
	prompt := b.parsePromptContent()
	closeBrace := b.parseCloseBrace()
	if b.tokenReader.GetCurrentMode() == PARSER_MODE_PROMPT {
		b.tokenReader.EndMode()
	}
	b.endContext()
	return tree.CreateNaturalExpressionNode(optionalConstKeyword, naturalKeyword,
		optionalParenthesizedArgList, openBrace, prompt, closeBrace)
}

func (b *BallerinaParser) createMissingNaturalExpressionNode(optionalConstKeyword tree.STNode, naturalKeyword tree.STNode, optionalParenthesizedArgList tree.STNode) tree.STNode {
	openBrace := tree.CreateMissingToken(common.OPEN_BRACE_TOKEN, nil)
	closeBrace := tree.CreateMissingToken(common.CLOSE_BRACE_TOKEN, nil)
	prompt := tree.CreateEmptyNodeList()
	naturalExpr := tree.CreateNaturalExpressionNode(optionalConstKeyword, naturalKeyword,
		optionalParenthesizedArgList, openBrace, prompt, closeBrace)
	naturalExpr = tree.AddDiagnostic(naturalExpr, &common.ERROR_MISSING_NATURAL_PROMPT_BLOCK)
	return naturalExpr
}

func (b *BallerinaParser) parseOptionalParenthesizedArgList() tree.STNode {
	if b.peek().Kind() == common.OPEN_PAREN_TOKEN {
		return b.parseParenthesizedArgList()
	}
	return tree.CreateEmptyNode()
}

func (b *BallerinaParser) parsePromptContent() tree.STNode {
	var items []tree.STNode
	nextToken := b.peek()
	for !b.isEndOfPromptContent(nextToken.Kind()) {
		contentItem := b.parsePromptItem()
		items = append(items, contentItem)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(items...)
}

func (b *BallerinaParser) isEndOfPromptContent(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN, common.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parsePromptItem() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.INTERPOLATION_START_TOKEN {
		return b.parseInterpolation()
	}
	if nextToken.Kind() != common.PROMPT_CONTENT {
		nextToken = b.consume()
		return tree.CreateLiteralValueTokenWithDiagnostics(common.PROMPT_CONTENT,
			nextToken.Text(), nextToken.LeadingMinutiae(), nextToken.TrailingMinutiae(),
			nextToken.Diagnostics())
	}
	return b.consume()
}

func (b *BallerinaParser) createMissingObjectConstructor(annots tree.STNode, qualifierNodeList tree.STNode) tree.STNode {
	objectKeyword := tree.CreateMissingToken(common.OBJECT_KEYWORD, nil)
	openBrace := tree.CreateMissingToken(common.OPEN_BRACE_TOKEN, nil)
	closeBrace := tree.CreateMissingToken(common.CLOSE_BRACE_TOKEN, nil)
	objConstructor := tree.CreateObjectConstructorExpressionNode(annots, qualifierNodeList,
		objectKeyword, tree.CreateEmptyNode(), openBrace, tree.CreateEmptyNodeList(),
		closeBrace)
	objConstructor = tree.AddDiagnostic(objConstructor,
		&common.ERROR_MISSING_OBJECT_CONSTRUCTOR_EXPRESSION)
	return objConstructor
}

func (b *BallerinaParser) parseQualifiedIdentifierOrExpression(isInConditionalExpr bool, isRhsExpr bool, allowActions bool) tree.STNode {
	preDeclaredPrefix := b.consume()
	nextNextToken := b.getNextNextToken()
	if (nextNextToken.Kind() == common.IDENTIFIER_TOKEN) && (!isKeyKeyword(nextNextToken)) {
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	var context common.ParserRuleContext
	switch preDeclaredPrefix.Kind() {
	case common.TABLE_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF
	case common.STREAM_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF
	case common.ERROR_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF
	default:
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	solution := b.recoverWithBlockContext(b.peek(), context)
	if solution.Action == ACTION_KEEP {
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	if preDeclaredPrefix.Kind() == common.ERROR_KEYWORD {
		return b.parseErrorConstructorExpr(preDeclaredPrefix)
	}
	b.startContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION)
	var tableOrQuery tree.STNode
	if preDeclaredPrefix.Kind() == common.STREAM_KEYWORD {
		queryConstructType := b.parseQueryConstructType(preDeclaredPrefix, nil)
		tableOrQuery = b.parseQueryExprRhs(queryConstructType, isRhsExpr, allowActions)
	} else {
		tableOrQuery = b.parseTableConstructorOrQueryWithKeyword(preDeclaredPrefix, isRhsExpr, allowActions)
	}
	b.endContext()
	return tableOrQuery
}

func (b *BallerinaParser) validateExprAnnotsAndQualifiers(nextToken tree.STToken, annots tree.STNode, qualifiers []tree.STNode) {
	switch nextToken.Kind() {
	case common.START_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
	case common.FUNCTION_KEYWORD, common.OBJECT_KEYWORD, common.AT_TOKEN:
		break
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			b.reportInvalidExpressionAnnots(annots, qualifiers)
			b.reportInvalidQualifierList(qualifiers)
		}
	}
}

func (b *BallerinaParser) isAnnotAllowedExprStart(nextToken tree.STToken) bool {
	switch nextToken.Kind() {
	case common.START_KEYWORD, common.FUNCTION_KEYWORD, common.OBJECT_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isValidExprStart(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.IDENTIFIER_TOKEN,
		common.OPEN_PAREN_TOKEN,
		common.CHECK_KEYWORD,
		common.CHECKPANIC_KEYWORD,
		common.OPEN_BRACE_TOKEN,
		common.TYPEOF_KEYWORD,
		common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.NEGATION_TOKEN,
		common.EXCLAMATION_MARK_TOKEN,
		common.TRAP_KEYWORD,
		common.OPEN_BRACKET_TOKEN,
		common.LT_TOKEN,
		common.TABLE_KEYWORD,
		common.STREAM_KEYWORD,
		common.FROM_KEYWORD,
		common.ERROR_KEYWORD,
		common.LET_KEYWORD,
		common.BACKTICK_TOKEN,
		common.XML_KEYWORD,
		common.RE_KEYWORD,
		common.STRING_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.AT_TOKEN,
		common.NEW_KEYWORD,
		common.START_KEYWORD,
		common.FLUSH_KEYWORD,
		common.LEFT_ARROW_TOKEN,
		common.WAIT_KEYWORD,
		common.COMMIT_KEYWORD,
		common.SERVICE_KEYWORD,
		common.BASE16_KEYWORD,
		common.BASE64_KEYWORD,
		common.ISOLATED_KEYWORD,
		common.TRANSACTIONAL_KEYWORD,
		common.CLIENT_KEYWORD,
		common.NATURAL_KEYWORD,
		common.OBJECT_KEYWORD:
		return true
	default:
		if isPredeclaredPrefix(tokenKind) {
			return true
		}
		return b.isSimpleTypeInExpression(tokenKind)
	}
}

func (b *BallerinaParser) parseNewExpression() tree.STNode {
	newKeyword := b.parseNewKeyword()
	return b.parseNewKeywordRhs(newKeyword)
}

func (b *BallerinaParser) parseNewKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.NEW_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_NEW_KEYWORD)
		return b.parseNewKeyword()
	}
}

func (b *BallerinaParser) parseNewKeywordRhs(newKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.OPEN_PAREN_TOKEN {
		return b.parseImplicitNewExpr(newKeyword)
	}
	if b.isClassDescriptorStartToken(nextToken.Kind()) {
		return b.parseExplicitNewExpr(newKeyword)
	}
	return b.createImplicitNewExpr(newKeyword, tree.CreateEmptyNode())
}

func (b *BallerinaParser) isClassDescriptorStartToken(tokenKind common.SyntaxKind) bool {
	return ((tokenKind == common.STREAM_KEYWORD) || b.isPredeclaredIdentifier(tokenKind))
}

func (b *BallerinaParser) parseExplicitNewExpr(newKeyword tree.STNode) tree.STNode {
	typeDescriptor := b.parseClassDescriptor()
	parenthesizedArgsList := b.parseParenthesizedArgList()
	return tree.CreateExplicitNewExpressionNode(newKeyword, typeDescriptor, parenthesizedArgsList)
}

func (b *BallerinaParser) parseClassDescriptor() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR)
	var classDescriptor tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.STREAM_KEYWORD:
		classDescriptor = b.parseStreamTypeDescriptor(b.consume())
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			classDescriptor = b.parseTypeReference()
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR)
		return b.parseClassDescriptor()
	}
	b.endContext()
	return classDescriptor
}

func (b *BallerinaParser) parseImplicitNewExpr(newKeyword tree.STNode) tree.STNode {
	parenthesizedArgList := b.parseParenthesizedArgList()
	return b.createImplicitNewExpr(newKeyword, parenthesizedArgList)
}

func (b *BallerinaParser) createImplicitNewExpr(newKeyword tree.STNode, parenthesizedArgList tree.STNode) tree.STNode {
	return tree.CreateImplicitNewExpressionNode(newKeyword, parenthesizedArgList)
}

func (b *BallerinaParser) parseParenthesizedArgList() tree.STNode {
	openParan := b.parseArgListOpenParenthesis()
	arguments := b.parseArgsList()
	closeParan := b.parseArgListCloseParenthesis()
	return tree.CreateParenthesizedArgList(openParan, arguments, closeParan)
}

func (b *BallerinaParser) parseExpressionRhs(precedenceLevel OperatorPrecedence, lhsExpr tree.STNode, isRhsExpr bool, allowActions bool) tree.STNode {
	return b.parseExpressionRhsInner(precedenceLevel, lhsExpr, isRhsExpr, allowActions, false, false)
}

func (b *BallerinaParser) parseExpressionRhsInner(currentPrecedenceLevel OperatorPrecedence, lhsExpr tree.STNode, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) tree.STNode {
	actionOrExpression := b.parseExpressionRhsInternal(currentPrecedenceLevel, lhsExpr, isRhsExpr, allowActions,
		isInMatchGuard, isInConditionalExpr)
	if ((!allowActions) && b.isAction(actionOrExpression)) && (actionOrExpression.Kind() != common.BRACED_ACTION) {
		actionOrExpression = b.invalidateActionAndGetMissingExpr(actionOrExpression)
	}
	return actionOrExpression
}

func (b *BallerinaParser) parseExpressionRhsInternal(currentPrecedenceLevel OperatorPrecedence, lhsExpr tree.STNode, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) tree.STNode {
	nextToken := b.peek()
	if b.isAction(lhsExpr) || b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard) {
		return lhsExpr
	}
	nextTokenKind := nextToken.Kind()
	if !b.isValidExprRhsStart(nextTokenKind, lhsExpr.Kind()) {
		return b.recoverExpressionRhs(currentPrecedenceLevel, lhsExpr, isRhsExpr, allowActions, isInMatchGuard,
			isInConditionalExpr)
	}
	if (nextTokenKind == common.GT_TOKEN) && (b.peekN(2).Kind() == common.GT_TOKEN) {
		if b.peekN(3).Kind() == common.GT_TOKEN {
			nextTokenKind = common.TRIPPLE_GT_TOKEN
		} else {
			nextTokenKind = common.DOUBLE_GT_TOKEN
		}
	}
	nextOperatorPrecedence := b.getOpPrecedence(nextTokenKind)
	if currentPrecedenceLevel.isHigherThanOrEqual(nextOperatorPrecedence, allowActions) {
		return lhsExpr
	}
	var newLhsExpr tree.STNode
	var operator tree.STNode
	switch nextTokenKind {
	case common.OPEN_PAREN_TOKEN:
		newLhsExpr = b.parseFuncCallOrNaturalExpr(lhsExpr)
	case common.OPEN_BRACKET_TOKEN:
		newLhsExpr = b.parseMemberAccessExpr(lhsExpr, isRhsExpr)
	case common.DOT_TOKEN:
		newLhsExpr = b.parseFieldAccessOrMethodCall(lhsExpr, isInConditionalExpr)
	case common.IS_KEYWORD,
		common.NOT_IS_KEYWORD:
		newLhsExpr = b.parseTypeTestExpression(lhsExpr, isInConditionalExpr)
	case common.RIGHT_ARROW_TOKEN:
		newLhsExpr = b.parseRemoteMethodCallOrClientResourceAccessOrAsyncSendAction(lhsExpr, isRhsExpr,
			isInMatchGuard)
	case common.SYNC_SEND_TOKEN:
		newLhsExpr = b.parseSyncSendAction(lhsExpr)
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		newLhsExpr = b.parseImplicitAnonFuncWithParams(lhsExpr, isRhsExpr)
	case common.ANNOT_CHAINING_TOKEN:
		newLhsExpr = b.parseAnnotAccessExpression(lhsExpr, isInConditionalExpr)
	case common.OPTIONAL_CHAINING_TOKEN:
		newLhsExpr = b.parseOptionalFieldAccessExpression(lhsExpr, isInConditionalExpr)
	case common.QUESTION_MARK_TOKEN:
		newLhsExpr = b.parseConditionalExpression(lhsExpr, isInConditionalExpr)
	case common.DOT_LT_TOKEN:
		newLhsExpr = b.parseXMLFilterExpression(lhsExpr)
	case common.SLASH_LT_TOKEN,
		common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		common.SLASH_ASTERISK_TOKEN:
		newLhsExpr = b.parseXMLStepExpression(lhsExpr)
	default:
		if (nextTokenKind == common.SLASH_TOKEN) && (b.peekN(2).Kind() == common.LT_TOKEN) {
			expectedNodeType := b.getExpectedNodeKind(3)
			if expectedNodeType == common.XML_STEP_EXPRESSION {
				newLhsExpr = b.createXMLStepExpression(lhsExpr)
				break
			}
		}
		switch nextTokenKind {
		case common.DOUBLE_GT_TOKEN:
			operator = b.parseSignedRightShiftToken()
		case common.TRIPPLE_GT_TOKEN:
			operator = b.parseUnsignedRightShiftToken()
		default:
			operator = b.parseBinaryOperator()
		}
		rhsExpr := b.parseExpressionWithConditional(nextOperatorPrecedence, isRhsExpr, false, isInConditionalExpr)
		newLhsExpr = tree.CreateBinaryExpressionNode(common.BINARY_EXPRESSION, lhsExpr, operator,
			rhsExpr)
	}
	return b.parseExpressionRhsInternal(currentPrecedenceLevel, newLhsExpr, isRhsExpr, allowActions, isInMatchGuard,
		isInConditionalExpr)
}

func (b *BallerinaParser) recoverExpressionRhs(currentPrecedenceLevel OperatorPrecedence, lhsExpr tree.STNode, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) tree.STNode {
	token := b.peek()
	lhsExprKind := lhsExpr.Kind()
	var solution *Solution
	if (lhsExprKind == common.QUALIFIED_NAME_REFERENCE) || (lhsExprKind == common.SIMPLE_NAME_REFERENCE) {
		solution = b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS)
	} else {
		solution = b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS)
	}
	if solution.Action == ACTION_REMOVE {
		return b.parseExpressionRhsInner(currentPrecedenceLevel, lhsExpr, isRhsExpr, allowActions, isInMatchGuard,
			isInConditionalExpr)
	}
	if solution.Ctx == common.PARSER_RULE_CONTEXT_BINARY_OPERATOR {
		binaryOpKind := b.getBinaryOperatorKindToInsert(currentPrecedenceLevel)
		binaryOpContext := b.getMissingBinaryOperatorContext(currentPrecedenceLevel)
		b.insertToken(binaryOpKind, binaryOpContext)
	}
	return b.parseExpressionRhsInternal(currentPrecedenceLevel, lhsExpr, isRhsExpr, allowActions, isInMatchGuard,
		isInConditionalExpr)
}

func (b *BallerinaParser) createXMLStepExpression(lhsExpr tree.STNode) tree.STNode {
	var newLhsExpr tree.STNode
	slashToken := b.parseSlashToken()
	ltToken := b.parseLTToken()
	var slashLT tree.STNode
	if b.hasTrailingMinutiae(slashToken) || b.hasLeadingMinutiae(ltToken) {
		var diagnostics []tree.STNodeDiagnostic
		diagnostics = append(diagnostics, tree.CreateDiagnostic(&common.ERROR_INVALID_WHITESPACE_IN_SLASH_LT_TOKEN))
		slashLT = tree.CreateMissingToken(common.SLASH_LT_TOKEN, diagnostics)
		slashLT = tree.CloneWithLeadingInvalidNodeMinutiae(slashLT, slashToken, nil)
		slashLT = tree.CloneWithLeadingInvalidNodeMinutiae(slashLT, ltToken, nil)
	} else {
		slashLT = tree.CreateToken(common.SLASH_LT_TOKEN, slashToken.LeadingMinutiae(),
			ltToken.TrailingMinutiae())
	}
	namePattern := b.parseXMLNamePatternChain(slashLT)
	xmlStepExtends := b.parseXMLStepExtends()
	newLhsExpr = tree.CreateXMLStepExpressionNode(lhsExpr, namePattern, xmlStepExtends)
	return newLhsExpr
}

func (b *BallerinaParser) getExpectedNodeKind(lookahead int) common.SyntaxKind {
	nextToken := b.peekN(lookahead)
	switch nextToken.Kind() {
	case common.ASTERISK_TOKEN:
		return common.XML_STEP_EXPRESSION
	case common.GT_TOKEN:
		break
	case common.PIPE_TOKEN:
		return b.getExpectedNodeKind(lookahead + 1)
	case common.IDENTIFIER_TOKEN:
		nextToken = b.peekN(lookahead + 1)
		switch nextToken.Kind() {
		case common.GT_TOKEN:
			break
		case common.PIPE_TOKEN:
			return b.getExpectedNodeKind(lookahead + 1)
		case common.COLON_TOKEN:
			nextToken = b.peekN(lookahead + 1)
			switch nextToken.Kind() {
			case common.ASTERISK_TOKEN, common.GT_TOKEN:
				return common.XML_STEP_EXPRESSION
			case common.IDENTIFIER_TOKEN:
				nextToken = b.peekN(lookahead + 1)
				if nextToken.Kind() == common.PIPE_TOKEN {
					return b.getExpectedNodeKind(lookahead + 1)
				}
			default:
				return common.TYPE_CAST_EXPRESSION
			}
		default:
			return common.TYPE_CAST_EXPRESSION
		}
	default:
		return common.TYPE_CAST_EXPRESSION
	}
	nextToken = b.peekN(lookahead + 1)
	switch nextToken.Kind() {
	case common.OPEN_BRACKET_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.FROM_KEYWORD,
		common.LET_KEYWORD:
		return common.XML_STEP_EXPRESSION
	default:
		if b.isValidExpressionStart(nextToken.Kind(), lookahead) {
			break
		}
		return common.XML_STEP_EXPRESSION
	}
	return common.TYPE_CAST_EXPRESSION
}

func (b *BallerinaParser) hasTrailingMinutiae(node tree.STNode) bool {
	return (node.WidthWithTrailingMinutiae() > node.Width())
}

func (b *BallerinaParser) hasLeadingMinutiae(node tree.STNode) bool {
	return (node.WidthWithLeadingMinutiae() > node.Width())
}

func (b *BallerinaParser) isValidExprRhsStart(tokenKind common.SyntaxKind, precedingNodeKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.OPEN_PAREN_TOKEN:
		return ((precedingNodeKind == common.QUALIFIED_NAME_REFERENCE) || (precedingNodeKind == common.SIMPLE_NAME_REFERENCE))
	case common.DOT_TOKEN,
		common.OPEN_BRACKET_TOKEN,
		common.IS_KEYWORD,
		common.RIGHT_ARROW_TOKEN,
		common.RIGHT_DOUBLE_ARROW_TOKEN,
		common.SYNC_SEND_TOKEN,
		common.ANNOT_CHAINING_TOKEN,
		common.OPTIONAL_CHAINING_TOKEN,
		common.COLON_TOKEN,
		common.DOT_LT_TOKEN,
		common.SLASH_LT_TOKEN,
		common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		common.SLASH_ASTERISK_TOKEN,
		common.NOT_IS_KEYWORD:
		return true
	case common.QUESTION_MARK_TOKEN:
		return ((b.getNextNextToken().Kind() != common.EQUAL_TOKEN) && (b.peekN(3).Kind() != common.EQUAL_TOKEN))
	default:
		return b.isBinaryOperator(tokenKind)
	}
}

func (b *BallerinaParser) parseMemberAccessExpr(lhsExpr tree.STNode, isRhsExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR)
	openBracket := b.parseOpenBracket()
	keyExpr := b.parseMemberAccessKeyExprs(isRhsExpr)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	if isRhsExpr {
		listKeyExprNode, ok := keyExpr.(*tree.STNodeList)
		if !ok {
			panic("expected STNodeList")
		}
		if listKeyExprNode.IsEmpty() {
			missingVarRef := tree.CreateSimpleNameReferenceNode(tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil))
			keyExpr = tree.CreateNodeList(missingVarRef)
			closeBracket = tree.AddDiagnostic(closeBracket,
				&common.ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR)
		}
	}
	return tree.CreateIndexedExpressionNode(lhsExpr, openBracket, keyExpr, closeBracket)
}

func (b *BallerinaParser) parseMemberAccessKeyExprs(isRhsExpr bool) tree.STNode {
	var exprList []tree.STNode
	var keyExpr tree.STNode
	var keyExprEnd tree.STNode
	for !b.isEndOfTypeList(b.peek().Kind()) {
		keyExpr = b.parseKeyExpr(isRhsExpr)
		exprList = append(exprList, keyExpr)
		keyExprEnd = b.parseMemberAccessKeyExprEnd()
		if keyExprEnd == nil {
			break
		}
		exprList = append(exprList, keyExprEnd)
	}
	return tree.CreateNodeList(exprList...)
}

func (b *BallerinaParser) parseKeyExpr(isRhsExpr bool) tree.STNode {
	if (!isRhsExpr) && (b.peek().Kind() == common.ASTERISK_TOKEN) {
		return tree.CreateBasicLiteralNode(common.ASTERISK_LITERAL, b.consume())
	}
	return b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_DEFAULT, isRhsExpr, false)
}

func (b *BallerinaParser) parseMemberAccessKeyExprEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END)
		return b.parseMemberAccessKeyExprEnd()
	}
}

func (b *BallerinaParser) parseCloseBracket() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CLOSE_BRACKET_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET)
		return b.parseCloseBracket()
	}
}

func (b *BallerinaParser) parseFieldAccessOrMethodCall(lhsExpr tree.STNode, isInConditionalExpr bool) tree.STNode {
	dotToken := b.parseDotToken()
	if b.isSpecialMethodName(b.peek()) {
		methodName := b.getKeywordAsSimpleNameRef()
		openParen := b.parseArgListOpenParenthesis()
		args := b.parseArgsList()
		closeParen := b.parseArgListCloseParenthesis()
		return tree.CreateMethodCallExpressionNode(lhsExpr, dotToken, methodName, openParen, args,
			closeParen)
	}
	fieldOrMethodName := b.parseFieldAccessIdentifier(isInConditionalExpr)
	if fieldOrMethodName.Kind() == common.QUALIFIED_NAME_REFERENCE {
		return tree.CreateFieldAccessExpressionNode(lhsExpr, dotToken, fieldOrMethodName)
	}
	nextToken := b.peek()
	if nextToken.Kind() == common.OPEN_PAREN_TOKEN {
		openParen := b.parseArgListOpenParenthesis()
		args := b.parseArgsList()
		closeParen := b.parseArgListCloseParenthesis()
		return tree.CreateMethodCallExpressionNode(lhsExpr, dotToken, fieldOrMethodName, openParen, args,
			closeParen)
	}
	return tree.CreateFieldAccessExpressionNode(lhsExpr, dotToken, fieldOrMethodName)
}

func (b *BallerinaParser) getKeywordAsSimpleNameRef() tree.STNode {
	mapKeyword := b.consume()
	var methodName tree.STNode
	methodName = tree.CreateIdentifierTokenWithDiagnostics(mapKeyword.Text(), mapKeyword.LeadingMinutiae(),
		mapKeyword.TrailingMinutiae(), mapKeyword.Diagnostics())
	methodName = tree.CreateSimpleNameReferenceNode(methodName)
	return methodName
}

func (b *BallerinaParser) parseBracedExpression(isRhsExpr bool, allowActions bool) tree.STNode {
	openParen := b.parseOpenParenthesis()
	if b.peek().Kind() == common.CLOSE_PAREN_TOKEN {
		return tree.CreateNilLiteralNode(openParen, b.consume())
	}
	b.startContext(common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS)
	var expr tree.STNode
	if allowActions {
		expr = b.parseExpressionWithPrecedence(DEFAULT_OP_PRECEDENCE, isRhsExpr, true)
	} else {
		expr = b.parseExpressionWithPrecedence(DEFAULT_OP_PRECEDENCE, isRhsExpr, false)
	}
	return b.parseBracedExprOrAnonFuncParamRhs(openParen, expr, isRhsExpr)
}

func (b *BallerinaParser) parseBracedExprOrAnonFuncParamRhs(openParen tree.STNode, expr tree.STNode, isRhsExpr bool) tree.STNode {
	nextToken := b.peek()
	if expr.Kind() == common.SIMPLE_NAME_REFERENCE {
		switch nextToken.Kind() {
		case common.CLOSE_PAREN_TOKEN:
			break
		case common.COMMA_TOKEN:
			return b.parseImplicitAnonFuncWithOpenParenAndFirstParam(openParen, expr, isRhsExpr)
		default:
			b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS)
			return b.parseBracedExprOrAnonFuncParamRhs(openParen, expr, isRhsExpr)
		}
	}
	closeParen := b.parseCloseParenthesis()
	b.endContext()
	if b.isAction(expr) {
		return tree.CreateBracedExpressionNode(common.BRACED_ACTION, openParen, expr, closeParen)
	}
	return tree.CreateBracedExpressionNode(common.BRACED_EXPRESSION, openParen, expr, closeParen)
}

func (b *BallerinaParser) isAction(node tree.STNode) bool {
	switch node.Kind() {
	case common.REMOTE_METHOD_CALL_ACTION,
		common.BRACED_ACTION,
		common.CHECK_ACTION,
		common.START_ACTION,
		common.TRAP_ACTION,
		common.FLUSH_ACTION,
		common.ASYNC_SEND_ACTION,
		common.SYNC_SEND_ACTION,
		common.RECEIVE_ACTION,
		common.WAIT_ACTION,
		common.QUERY_ACTION,
		common.COMMIT_ACTION,
		common.CLIENT_RESOURCE_ACCESS_ACTION:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isEndOfActionOrExpression(nextToken tree.STToken, isRhsExpr bool, isInMatchGuard bool) bool {
	tokenKind := nextToken.Kind()
	if !isRhsExpr {
		if b.isCompoundAssignment(tokenKind) {
			return true
		}
		if isInMatchGuard && (tokenKind == common.RIGHT_DOUBLE_ARROW_TOKEN) {
			return true
		}
	}
	switch tokenKind {
	case common.EOF_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.CLOSE_BRACKET_TOKEN,
		common.SEMICOLON_TOKEN,
		common.COMMA_TOKEN,
		common.PUBLIC_KEYWORD,
		common.CONST_KEYWORD,
		common.LISTENER_KEYWORD,
		common.RESOURCE_KEYWORD,
		common.EQUAL_TOKEN,
		common.DOCUMENTATION_STRING,
		common.AT_TOKEN,
		common.AS_KEYWORD,
		common.IN_KEYWORD,
		common.FROM_KEYWORD,
		common.WHERE_KEYWORD,
		common.LET_KEYWORD,
		common.SELECT_KEYWORD,
		common.DO_KEYWORD,
		common.COLON_TOKEN,
		common.ON_KEYWORD,
		common.CONFLICT_KEYWORD,
		common.LIMIT_KEYWORD,
		common.JOIN_KEYWORD,
		common.OUTER_KEYWORD,
		common.ORDER_KEYWORD,
		common.BY_KEYWORD,
		common.ASCENDING_KEYWORD,
		common.DESCENDING_KEYWORD,
		common.EQUALS_KEYWORD,
		common.TYPE_KEYWORD:
		return true
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		return isInMatchGuard
	case common.IDENTIFIER_TOKEN:
		return isGroupOrCollectKeyword(nextToken)
	default:
		return isSimpleType(tokenKind)
	}
}

func (b *BallerinaParser) parseBasicLiteral() tree.STNode {
	literalToken := b.consume()
	return b.parseBasicLiteralInner(literalToken)
}

func (b *BallerinaParser) parseBasicLiteralInner(literalToken tree.STNode) tree.STNode {
	var nodeKind common.SyntaxKind
	switch literalToken.Kind() {
	case common.NULL_KEYWORD:
		nodeKind = common.NULL_LITERAL
	case common.TRUE_KEYWORD, common.FALSE_KEYWORD:
		nodeKind = common.BOOLEAN_LITERAL
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		nodeKind = common.NUMERIC_LITERAL
	case common.STRING_LITERAL_TOKEN:
		nodeKind = common.STRING_LITERAL
	case common.ASTERISK_TOKEN:
		nodeKind = common.ASTERISK_LITERAL
	default:
		nodeKind = literalToken.Kind()
	}
	return tree.CreateBasicLiteralNode(nodeKind, literalToken)
}

func (b *BallerinaParser) parseFuncCallOrNaturalExpr(identifier tree.STNode) tree.STNode {
	openParen := b.parseArgListOpenParenthesis()
	args := b.parseArgsList()
	closeParen := b.parseArgListCloseParenthesis()
	if (b.peek().Kind() == common.OPEN_BRACE_TOKEN) && b.isNaturalKeyword(identifier) {
		nameRef, ok := identifier.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("expected STSimpleNameReferenceNode")
		}
		return b.parseNaturalExpressionInner(*nameRef, openParen, args, closeParen)
	}
	return tree.CreateFunctionCallExpressionNode(identifier, openParen, args, closeParen)
}

func (b *BallerinaParser) parseNaturalExpressionInner(nameRef tree.STSimpleNameReferenceNode, openParen tree.STNode, args tree.STNode, closeParen tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION)
	optionalConstKeyword := tree.CreateEmptyNode()
	naturalKeyword := b.getNaturalKeyword(tree.ToToken(nameRef.Name))
	parenthesizedArgList := tree.CreateParenthesizedArgList(openParen, args, closeParen)
	return b.parseNaturalExprBody(optionalConstKeyword, naturalKeyword, parenthesizedArgList)
}

func (b *BallerinaParser) parseErrorBindingPatternOrErrorConstructor() tree.STNode {
	return b.parseErrorConstructorExprAmbiguous(true)
}

func (b *BallerinaParser) parseErrorConstructorExpr(errorKeyword tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR)
	return b.parseErrorConstructorExprInner(errorKeyword, false)
}

func (b *BallerinaParser) parseErrorConstructorExprAmbiguous(isAmbiguous bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR)
	errorKeyword := b.parseErrorKeyword()
	return b.parseErrorConstructorExprInner(errorKeyword, isAmbiguous)
}

func (b *BallerinaParser) parseErrorConstructorExprInner(errorKeyword tree.STNode, isAmbiguous bool) tree.STNode {
	typeReference := b.parseErrorTypeReference()
	openParen := b.parseArgListOpenParenthesis()
	functionArgs := b.parseArgsList()
	var errorArgs tree.STNode
	if isAmbiguous {
		errorArgs = functionArgs
	} else {
		errorArgs = b.getErrorArgList(functionArgs)
	}
	closeParen := b.parseArgListCloseParenthesis()
	b.endContext()
	openParen = b.cloneWithDiagnosticIfListEmpty(errorArgs, openParen,
		&common.ERROR_MISSING_ARG_WITHIN_PARENTHESIS)
	return tree.CreateErrorConstructorExpressionNode(errorKeyword, typeReference, openParen, errorArgs,
		closeParen)
}

func (b *BallerinaParser) parseErrorTypeReference() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		return tree.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			return b.parseTypeReference()
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS)
		return b.parseErrorTypeReference()
	}
}

func (b *BallerinaParser) getErrorArgList(functionArgs tree.STNode) tree.STNode {
	argList, ok := functionArgs.(*tree.STNodeList)
	if !ok {
		panic("expected *tree.STNodeList")
	}
	if argList.IsEmpty() {
		return argList
	}
	var errorArgList []tree.STNode
	arg := argList.Get(0)
	switch arg.Kind() {
	case common.POSITIONAL_ARG:
		errorArgList = append(errorArgList, arg)
	case common.NAMED_ARG:
		arg = tree.AddDiagnostic(arg,
			&common.ERROR_MISSING_ERROR_MESSAGE_IN_ERROR_CONSTRUCTOR)
		errorArgList = append(errorArgList, arg)
	default:
		arg = tree.AddDiagnostic(arg,
			&common.ERROR_MISSING_ERROR_MESSAGE_IN_ERROR_CONSTRUCTOR)
		arg = tree.AddDiagnostic(arg, &common.ERROR_REST_ARG_IN_ERROR_CONSTRUCTOR)
		errorArgList = append(errorArgList, arg)
	}
	diagnosticErrorCode := &common.ERROR_REST_ARG_IN_ERROR_CONSTRUCTOR
	hasPositionalArg := false
	var leadingComma tree.STNode
	i := 1
	for ; i < argList.Size(); i = i + 2 {
		leadingComma = argList.Get(i)
		arg = argList.Get(i + 1)
		if arg.Kind() == common.NAMED_ARG {
			errorArgList = append(errorArgList, leadingComma, arg)
			continue
		}
		if arg.Kind() == common.POSITIONAL_ARG {
			if !hasPositionalArg {
				errorArgList = append(errorArgList, leadingComma, arg)
				hasPositionalArg = true
				continue
			}
			diagnosticErrorCode = &common.ERROR_ADDITIONAL_POSITIONAL_ARG_IN_ERROR_CONSTRUCTOR
		}
		b.updateLastNodeInListWithInvalidNode(errorArgList, leadingComma, nil)
		b.updateLastNodeInListWithInvalidNode(errorArgList, arg, diagnosticErrorCode)
	}
	return tree.CreateNodeList(errorArgList...)
}

func (b *BallerinaParser) parseArgsList() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ARG_LIST)
	token := b.peek()
	if b.isEndOfParametersList(token.Kind()) {
		args := tree.CreateEmptyNodeList()
		b.endContext()
		return args
	}
	firstArg := b.parseArgument()
	argsList := b.parseArgList(firstArg)
	b.endContext()
	return argsList
}

func (b *BallerinaParser) parseArgList(firstArg tree.STNode) tree.STNode {
	var argsList []tree.STNode
	argsList = append(argsList, firstArg)
	lastValidArgKind := firstArg.Kind()
	nextToken := b.peek()
	for !b.isEndOfParametersList(nextToken.Kind()) {
		argEnd := b.parseArgEnd()
		if argEnd == nil {
			break
		}
		curArg := b.parseArgument()
		errorCode := b.validateArgumentOrder(lastValidArgKind, curArg.Kind())
		if errorCode == nil {
			argsList = append(argsList, argEnd, curArg)
			lastValidArgKind = curArg.Kind()
		} else if errorCode == &common.ERROR_NAMED_ARG_FOLLOWED_BY_POSITIONAL_ARG {
			posArg, ok := curArg.(*tree.STPositionalArgumentNode)
			if !ok {
				panic("parseArgList: expected STPositionalArgumentNode")
			}
			if posArg.Expression.Kind() == common.SIMPLE_NAME_REFERENCE {
				missingEqual := tree.CreateMissingToken(common.EQUAL_TOKEN, nil)
				missingIdentifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
				nameRef := tree.CreateSimpleNameReferenceNode(missingIdentifier)
				expr := posArg.Expression
				simpleNameExpr, ok := expr.(*tree.STSimpleNameReferenceNode)
				if !ok {
					panic("parseArgList: expected STSimpleNameReferenceNode")
				}
				if simpleNameExpr.Name.IsMissing() {
					errorCode = &common.ERROR_MISSING_NAMED_ARG
					expr = nameRef
				}
				curArg = tree.CreateNamedArgumentNode(expr, missingEqual, nameRef)
				curArg = tree.AddDiagnostic(curArg, errorCode)
				argsList = append(argsList, argEnd, curArg)
			} else {
				argsList = b.updateLastNodeInListWithInvalidNode(argsList, argEnd, nil)
				argsList = b.updateLastNodeInListWithInvalidNode(argsList, curArg, errorCode)
			}
		} else {
			argsList = b.updateLastNodeInListWithInvalidNode(argsList, argEnd, nil)
			argsList = b.updateLastNodeInListWithInvalidNode(argsList, curArg, errorCode)
		}
		nextToken = b.peek()
	}
	return tree.CreateNodeList(argsList...)
}

func (b *BallerinaParser) validateArgumentOrder(prevArgKind common.SyntaxKind, curArgKind common.SyntaxKind) *common.DiagnosticErrorCode {
	var errorCode *common.DiagnosticErrorCode
	switch prevArgKind {
	case common.POSITIONAL_ARG:
		// Positional args can be followed by any type of arg - no error
		errorCode = nil
	case common.NAMED_ARG:
		// Named args cannot be followed by positional args
		if curArgKind == common.POSITIONAL_ARG {
			errorCode = &common.ERROR_NAMED_ARG_FOLLOWED_BY_POSITIONAL_ARG
		}
	case common.REST_ARG:
		errorCode = &common.ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG
	default:
		panic("Invalid common.SyntaxKind in an argument")
	}
	return errorCode
}

func (b *BallerinaParser) parseArgEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ARG_END)
		return b.parseArgEnd()
	}
}

func (b *BallerinaParser) parseArgument() tree.STNode {
	var arg tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ELLIPSIS_TOKEN:
		ellipsis := b.consume()
		expr := b.parseExpression()
		arg = tree.CreateRestArgumentNode(ellipsis, expr)
	case common.IDENTIFIER_TOKEN:
		arg = b.parseNamedOrPositionalArg()
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			expr := b.parseExpression()
			arg = tree.CreatePositionalArgumentNode(expr)
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ARG_START)
		return b.parseArgument()
	}
	return arg
}

func (b *BallerinaParser) parseNamedOrPositionalArg() tree.STNode {
	argNameOrExpr := b.parseTerminalExpression(true, false, false)
	secondToken := b.peek()
	switch secondToken.Kind() {
	case common.EQUAL_TOKEN:
		if argNameOrExpr.Kind() != common.SIMPLE_NAME_REFERENCE {
			break
		}
		equal := b.parseAssignOp()
		valExpr := b.parseExpression()
		return tree.CreateNamedArgumentNode(argNameOrExpr, equal, valExpr)
	case common.COMMA_TOKEN, common.CLOSE_PAREN_TOKEN:
		return tree.CreatePositionalArgumentNode(argNameOrExpr)
	}
	argNameOrExpr = b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, argNameOrExpr, true, false)
	return tree.CreatePositionalArgumentNode(argNameOrExpr)
}

func (b *BallerinaParser) parseObjectTypeDescriptor(objectKeyword tree.STNode, objectTypeQualifiers tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR)
	openBrace := b.parseOpenBrace()
	objectMemberDescriptors := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return tree.CreateObjectTypeDescriptorNode(objectTypeQualifiers, objectKeyword, openBrace,
		objectMemberDescriptors, closeBrace)
}

func (b *BallerinaParser) parseObjectConstructorExpression(annots tree.STNode, qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR)
	objectTypeQualifier := b.createObjectTypeQualNodeList(qualifiers)
	objectKeyword := b.parseObjectKeyword()
	typeReference := b.parseObjectConstructorTypeReference()
	openBrace := b.parseOpenBrace()
	objectMembers := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return tree.CreateObjectConstructorExpressionNode(annots,
		objectTypeQualifier, objectKeyword, typeReference, openBrace, objectMembers, closeBrace)
}

func (b *BallerinaParser) parseObjectConstructorTypeReference() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACE_TOKEN:
		return tree.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			return b.parseTypeReference()
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF)
		return b.parseObjectConstructorTypeReference()
	}
}

func (b *BallerinaParser) isPredeclaredIdentifier(tokenKind common.SyntaxKind) bool {
	return ((tokenKind == common.IDENTIFIER_TOKEN) || b.isQualifiedIdentifierPredeclaredPrefix(tokenKind))
}

func (b *BallerinaParser) parseObjectKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.OBJECT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD)
		return b.parseObjectKeyword()
	}
}

func (b *BallerinaParser) parseObjectMembers(context common.ParserRuleContext) tree.STNode {
	var objectMembers []tree.STNode
	for !b.isEndOfObjectTypeNode() {
		b.startContext(context)
		member := b.parseObjectMember(context)
		b.endContext()
		if member == nil {
			break
		}
		if (context == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER) && (member.Kind() == common.TYPE_REFERENCE) {
			b.addInvalidNodeToNextToken(member, &common.ERROR_TYPE_INCLUSION_IN_OBJECT_CONSTRUCTOR)
		} else {
			objectMembers = append(objectMembers, member)
		}
	}
	return tree.CreateNodeList(objectMembers...)
}

func (b *BallerinaParser) parseObjectMember(context common.ParserRuleContext) tree.STNode {
	var metadata tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.EOF_TOKEN,
		common.CLOSE_BRACE_TOKEN:
		return nil
	case common.ASTERISK_TOKEN,
		common.PUBLIC_KEYWORD,
		common.PRIVATE_KEYWORD,
		common.FINAL_KEYWORD,
		common.REMOTE_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.TRANSACTIONAL_KEYWORD,
		common.ISOLATED_KEYWORD,
		common.RESOURCE_KEYWORD:
		metadata = tree.CreateEmptyNode()
	case common.DOCUMENTATION_STRING,
		common.AT_TOKEN:
		metadata = b.parseMetaData()
	case common.RETURN_KEYWORD:
		b.addInvalidNodeToNextToken(b.consume(), &common.ERROR_INVALID_TOKEN)
		return b.parseObjectMember(context)
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			metadata = tree.CreateEmptyNode()
			break
		}
		var recoveryCtx common.ParserRuleContext
		if context == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER {
			recoveryCtx = common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START
		} else {
			recoveryCtx = common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START
		}
		solution := b.recoverWithBlockContext(b.peek(), recoveryCtx)
		if solution.Action == ACTION_KEEP {
			metadata = tree.CreateEmptyNode()
			break
		}
		return b.parseObjectMember(context)
	}
	return b.parseObjectMemberWithoutMeta(metadata, context)
}

func (b *BallerinaParser) parseObjectMemberWithoutMeta(metadata tree.STNode, context common.ParserRuleContext) tree.STNode {
	isObjectTypeDesc := (context == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER)
	var recoveryCtx common.ParserRuleContext
	if context == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER {
		recoveryCtx = common.PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META
	} else {
		recoveryCtx = common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META
	}
	res, _ := b.parseObjectMemberWithoutMetaInner(metadata, nil, recoveryCtx, isObjectTypeDesc)
	return res
}

func (b *BallerinaParser) parseObjectMemberWithoutMetaInner(metadata tree.STNode, qualifiers []tree.STNode, recoveryCtx common.ParserRuleContext, isObjectTypeDesc bool) (tree.STNode, []tree.STNode) {
	qualifiers = b.parseObjectMemberQualifiers(qualifiers)
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.EOF_TOKEN,
		common.CLOSE_BRACE_TOKEN:
		if (metadata != nil) || (len(qualifiers) > 0) {
			return b.createMissingSimpleObjectFieldInner(metadata, qualifiers, isObjectTypeDesc)
		}
		return nil, nil
	case common.PUBLIC_KEYWORD,
		common.PRIVATE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		var visibilityQualifier tree.STNode
		visibilityQualifier = b.consume()
		if isObjectTypeDesc && (visibilityQualifier.Kind() == common.PRIVATE_KEYWORD) {
			b.addInvalidNodeToNextToken(visibilityQualifier,
				&common.ERROR_PRIVATE_QUALIFIER_IN_OBJECT_MEMBER_DESCRIPTOR)
			visibilityQualifier = tree.CreateEmptyNode()
		}
		return b.parseObjectMethodOrField(metadata, visibilityQualifier, isObjectTypeDesc), qualifiers
	case common.FUNCTION_KEYWORD:
		visibilityQualifier := tree.CreateEmptyNode()
		return b.parseObjectMethodOrFuncTypeDesc(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc), qualifiers
	case common.ASTERISK_TOKEN:
		b.reportInvalidMetaData(metadata, "object ty inclusion")
		b.reportInvalidQualifierList(qualifiers)
		asterisk := b.consume()
		ty := b.parseTypeReferenceInTypeInclusion()
		semicolonToken := b.parseSemicolon()
		return tree.CreateTypeReferenceNode(asterisk, ty, semicolonToken), qualifiers
	case common.IDENTIFIER_TOKEN:
		if b.isObjectFieldStart() || nextToken.IsMissing() {
			return b.parseObjectField(metadata, tree.CreateEmptyNode(), qualifiers, isObjectTypeDesc)
		}
		if b.isObjectMethodStart(b.getNextNextToken()) {
			b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
			return b.parseObjectMemberWithoutMetaInner(metadata, qualifiers, recoveryCtx, isObjectTypeDesc)
		}
		fallthrough
	default:
		if b.isTypeStartingToken(nextToken.Kind()) && (nextToken.Kind() != common.IDENTIFIER_TOKEN) {
			return b.parseObjectField(metadata, tree.CreateEmptyNode(), qualifiers, isObjectTypeDesc)
		}
		solution := b.recoverWithBlockContext(b.peek(), recoveryCtx)
		if solution.Action == ACTION_KEEP {
			return b.parseObjectField(metadata, tree.CreateEmptyNode(), qualifiers, isObjectTypeDesc)
		}
		return b.parseObjectMemberWithoutMetaInner(metadata, qualifiers, recoveryCtx, isObjectTypeDesc)
	}
}

func (b *BallerinaParser) isObjectFieldStart() bool {
	nextNextToken := b.getNextNextToken()
	switch nextNextToken.Kind() {
	case common.ERROR_KEYWORD, // error-binding-pattern not allowed in fields
		common.OPEN_BRACE_TOKEN:
		return false
	case common.CLOSE_BRACE_TOKEN:
		return true
	default:
		return b.isModuleVarDeclStart(1)
	}
}

func (b *BallerinaParser) isObjectMethodStart(token tree.STToken) bool {
	switch token.Kind() {
	case common.FUNCTION_KEYWORD,
		common.REMOTE_KEYWORD,
		common.RESOURCE_KEYWORD,
		common.ISOLATED_KEYWORD,
		common.TRANSACTIONAL_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseObjectMethodOrField(metadata tree.STNode, visibilityQualifier tree.STNode, isObjectTypeDesc bool) tree.STNode {
	result, _ := b.parseObjectMethodOrFieldInner(metadata, visibilityQualifier, nil, isObjectTypeDesc)
	return result
}

func (b *BallerinaParser) parseObjectMethodOrFieldInner(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, isObjectTypeDesc bool) (tree.STNode, []tree.STNode) {
	qualifiers = b.parseObjectMemberQualifiers(qualifiers)
	nextToken := b.peekN(1)
	nextNextToken := b.peekN(2)
	switch nextToken.Kind() {
	case common.FUNCTION_KEYWORD:
		return b.parseObjectMethodOrFuncTypeDesc(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc), qualifiers
	case common.IDENTIFIER_TOKEN:
		if nextNextToken.Kind() != common.OPEN_PAREN_TOKEN {
			return b.parseObjectField(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc)
		}
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseObjectField(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc)
		}
	}
	b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY)
	return b.parseObjectMethodOrFieldInner(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc)
}

func (b *BallerinaParser) parseObjectField(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, isObjectTypeDesc bool) (tree.STNode, []tree.STNode) {
	objectFieldQualifiers, qualifiers := b.extractObjectFieldQualifiers(qualifiers, isObjectTypeDesc)
	objectFieldQualNodeList := tree.CreateNodeList(objectFieldQualifiers...)
	ty := b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	fieldName := b.parseVariableName()
	return b.parseObjectFieldRhs(metadata, visibilityQualifier, objectFieldQualNodeList, ty, fieldName,
		isObjectTypeDesc), qualifiers
}

func (b *BallerinaParser) extractObjectFieldQualifiers(qualifiers []tree.STNode, isObjectTypeDesc bool) ([]tree.STNode, []tree.STNode) {
	var objectFieldQualifiers []tree.STNode
	if len(qualifiers) != 0 && (!isObjectTypeDesc) {
		firstQualifier := qualifiers[0]
		if firstQualifier.Kind() == common.FINAL_KEYWORD {
			objectFieldQualifiers = append(objectFieldQualifiers, qualifiers[0])
			qualifiers = qualifiers[1:]
		}
	}
	return objectFieldQualifiers, qualifiers
}

func (b *BallerinaParser) parseObjectFieldRhs(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers tree.STNode, ty tree.STNode, fieldName tree.STNode, isObjectTypeDesc bool) tree.STNode {
	nextToken := b.peek()
	var equalsToken tree.STNode
	var expression tree.STNode
	var semicolonToken tree.STNode
	switch nextToken.Kind() {
	case common.SEMICOLON_TOKEN:
		equalsToken = tree.CreateEmptyNode()
		expression = tree.CreateEmptyNode()
		semicolonToken = b.parseSemicolon()
	case common.EQUAL_TOKEN:
		equalsToken = b.parseAssignOp()
		expression = b.parseExpression()
		semicolonToken = b.parseSemicolon()
		if isObjectTypeDesc {
			fieldName = tree.CloneWithTrailingInvalidNodeMinutiae(fieldName, equalsToken,
				&common.ERROR_FIELD_INITIALIZATION_NOT_ALLOWED_IN_OBJECT_TYPE)
			fieldName = tree.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(fieldName, expression)
			equalsToken = tree.CreateEmptyNode()
			expression = tree.CreateEmptyNode()
		}
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS)
		return b.parseObjectFieldRhs(metadata, visibilityQualifier, qualifiers, ty, fieldName,
			isObjectTypeDesc)
	}
	return tree.CreateObjectFieldNode(metadata, visibilityQualifier, qualifiers, ty, fieldName,
		equalsToken, expression, semicolonToken)
}

func (b *BallerinaParser) parseObjectMethodOrFuncTypeDesc(metadata tree.STNode, visibilityQualifier tree.STNode, qualifiers []tree.STNode, isObjectTypeDesc bool) tree.STNode {
	return b.parseFuncDefOrFuncTypeDesc(metadata, visibilityQualifier, qualifiers, true, isObjectTypeDesc)
}

func (b *BallerinaParser) parseRelativeResourcePath() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH)
	var pathElementList []tree.STNode
	nextToken := b.peek()
	if nextToken.Kind() == common.DOT_TOKEN {
		pathElementList = append(pathElementList, b.consume())
		b.endContext()
		return tree.CreateNodeList(pathElementList...)
	}
	pathSegment := b.parseResourcePathSegment(true)
	pathElementList = append(pathElementList, pathSegment)
	var leadingSlash tree.STNode
	for !b.isEndRelativeResourcePath(nextToken.Kind()) {
		leadingSlash = b.parseRelativeResourcePathEnd()
		if leadingSlash == nil {
			break
		}
		pathElementList = append(pathElementList, leadingSlash)
		pathSegment = b.parseResourcePathSegment(false)
		pathElementList = append(pathElementList, pathSegment)
		nextToken = b.peek()
	}
	b.endContext()
	return b.createResourcePathNodeList(pathElementList)
}

func (b *BallerinaParser) isEndRelativeResourcePath(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.EOF_TOKEN, common.OPEN_PAREN_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) createResourcePathNodeList(pathElementList []tree.STNode) tree.STNode {
	if len(pathElementList) == 0 {
		return tree.CreateEmptyNodeList()
	}
	var validatedList []tree.STNode
	firstElement := pathElementList[0]
	validatedList = append(validatedList, firstElement)
	hasRestPram := (firstElement.Kind() == common.RESOURCE_PATH_REST_PARAM)
	i := 1
	for ; i < len(pathElementList); i = i + 2 {
		leadingSlash := pathElementList[i]
		pathSegment := pathElementList[i+1]
		if hasRestPram {
			b.updateLastNodeInListWithInvalidNode(validatedList, leadingSlash, nil)
			b.updateLastNodeInListWithInvalidNode(validatedList, pathSegment,
				&common.ERROR_RESOURCE_PATH_SEGMENT_NOT_ALLOWED_AFTER_REST_PARAM)
			continue
		}
		hasRestPram = (pathSegment.Kind() == common.RESOURCE_PATH_REST_PARAM)
		validatedList = append(validatedList, leadingSlash)
		validatedList = append(validatedList, pathSegment)
	}
	return tree.CreateNodeList(validatedList...)
}

func (b *BallerinaParser) parseResourcePathSegment(isFirstSegment bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		if ((isFirstSegment && nextToken.IsMissing()) && b.isInvalidNodeStackEmpty()) && (b.getNextNextToken().Kind() == common.SLASH_TOKEN) {
			b.removeInsertedToken()
			return tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
				&common.ERROR_RESOURCE_PATH_CANNOT_BEGIN_WITH_SLASH)
		}
		return b.consume()
	case common.OPEN_BRACKET_TOKEN:
		return b.parseResourcePathParameter()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT)
		return b.parseResourcePathSegment(isFirstSegment)
	}
}

func (b *BallerinaParser) parseResourcePathParameter() tree.STNode {
	openBracket := b.parseOpenBracket()
	annots := b.parseOptionalAnnotations()
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM)
	ellipsis := b.parseOptionalEllipsis()
	paramName := b.parseOptionalPathParamName()
	closeBracket := b.parseCloseBracket()
	var pathPramKind common.SyntaxKind
	if ellipsis == nil {
		pathPramKind = common.RESOURCE_PATH_SEGMENT_PARAM
	} else {
		pathPramKind = common.RESOURCE_PATH_REST_PARAM
	}
	return tree.CreateResourcePathParameterNode(pathPramKind, openBracket, annots, ty, ellipsis,
		paramName, closeBracket)
}

func (b *BallerinaParser) parseOptionalPathParamName() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		return b.consume()
	case common.CLOSE_BRACKET_TOKEN:
		return tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME)
		return b.parseOptionalPathParamName()
	}
}

func (b *BallerinaParser) parseOptionalEllipsis() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ELLIPSIS_TOKEN:
		return b.consume()
	case common.IDENTIFIER_TOKEN, common.CLOSE_BRACKET_TOKEN:
		return tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS)
		return b.parseOptionalEllipsis()
	}
}

func (b *BallerinaParser) parseRelativeResourcePathEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN, common.EOF_TOKEN:
		return nil
	case common.SLASH_TOKEN:
		return b.consume()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END)
		return b.parseRelativeResourcePathEnd()
	}
}

func (b *BallerinaParser) parseIfElseBlock() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_IF_BLOCK)
	ifKeyword := b.parseIfKeyword()
	condition := b.parseExpression()
	ifBody := b.parseBlockNode()
	b.endContext()
	elseBody := b.parseElseBlock()
	return tree.CreateIfElseStatementNode(ifKeyword, condition, ifBody, elseBody)
}

func (b *BallerinaParser) parseIfKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IF_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IF_KEYWORD)
		return b.parseIfKeyword()
	}
}

func (b *BallerinaParser) parseElseKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ELSE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ELSE_KEYWORD)
		return b.parseElseKeyword()
	}
}

func (b *BallerinaParser) parseBlockNode() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
	openBrace := b.parseOpenBrace()
	stmts := b.parseStatements()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return tree.CreateBlockStatementNode(openBrace, stmts, closeBrace)
}

func (b *BallerinaParser) parseElseBlock() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() != common.ELSE_KEYWORD {
		return tree.CreateEmptyNode()
	}
	elseKeyword := b.parseElseKeyword()
	elseBody := b.parseElseBody()
	return tree.CreateElseBlockNode(elseKeyword, elseBody)
}

func (b *BallerinaParser) parseElseBody() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IF_KEYWORD:
		return b.parseIfElseBlock()
	case common.OPEN_BRACE_TOKEN:
		return b.parseBlockNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ELSE_BODY)
		return b.parseElseBody()
	}
}

func (b *BallerinaParser) parseDoStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_DO_BLOCK)
	doKeyword := b.parseDoKeyword()
	doBody := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateDoStatementNode(doKeyword, doBody, onFailClause)
}

func (b *BallerinaParser) parseWhileStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_WHILE_BLOCK)
	whileKeyword := b.parseWhileKeyword()
	condition := b.parseExpression()
	whileBody := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateWhileStatementNode(whileKeyword, condition, whileBody, onFailClause)
}

func (b *BallerinaParser) parseWhileKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.WHILE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_WHILE_KEYWORD)
		return b.parseWhileKeyword()
	}
}

func (b *BallerinaParser) parsePanicStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_PANIC_STMT)
	panicKeyword := b.parsePanicKeyword()
	expression := b.parseExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreatePanicStatementNode(panicKeyword, expression, semicolon)
}

func (b *BallerinaParser) parsePanicKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.PANIC_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PANIC_KEYWORD)
		return b.parsePanicKeyword()
	}
}

func (b *BallerinaParser) parseCheckExpression(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	checkingKeyword := b.parseCheckingKeyword()
	expr := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_EXPRESSION_ACTION, isRhsExpr, allowActions, isInConditionalExpr)
	if b.isAction(expr) {
		return tree.CreateCheckExpressionNode(common.CHECK_ACTION, checkingKeyword, expr)
	} else {
		return tree.CreateCheckExpressionNode(common.CHECK_EXPRESSION, checkingKeyword, expr)
	}
}

func (b *BallerinaParser) parseCheckingKeyword() tree.STNode {
	token := b.peek()
	if (token.Kind() == common.CHECK_KEYWORD) || (token.Kind() == common.CHECKPANIC_KEYWORD) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD)
		return b.parseCheckingKeyword()
	}
}

func (b *BallerinaParser) parseContinueStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT)
	continueKeyword := b.parseContinueKeyword()
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateContinueStatementNode(continueKeyword, semicolon)
}

func (b *BallerinaParser) parseContinueKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CONTINUE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD)
		return b.parseContinueKeyword()
	}
}

func (b *BallerinaParser) parseFailStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FAIL_STATEMENT)
	failKeyword := b.parseFailKeyword()
	expr := b.parseExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateFailStatementNode(failKeyword, expr, semicolon)
}

func (b *BallerinaParser) parseFailKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FAIL_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FAIL_KEYWORD)
		return b.parseFailKeyword()
	}
}

func (b *BallerinaParser) parseReturnStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RETURN_STMT)
	returnKeyword := b.parseReturnKeyword()
	returnRhs := b.parseReturnStatementRhs(returnKeyword)
	b.endContext()
	return returnRhs
}

func (b *BallerinaParser) parseReturnKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.RETURN_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RETURN_KEYWORD)
		return b.parseReturnKeyword()
	}
}

func (b *BallerinaParser) parseBreakStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BREAK_STATEMENT)
	breakKeyword := b.parseBreakKeyword()
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateBreakStatementNode(breakKeyword, semicolon)
}

func (b *BallerinaParser) parseBreakKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.BREAK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BREAK_KEYWORD)
		return b.parseBreakKeyword()
	}
}

func (b *BallerinaParser) parseReturnStatementRhs(returnKeyword tree.STNode) tree.STNode {
	var expr tree.STNode
	token := b.peek()
	switch token.Kind() {
	case common.SEMICOLON_TOKEN:
		expr = tree.CreateEmptyNode()
	default:
		expr = b.parseActionOrExpression()
	}
	semicolon := b.parseSemicolon()
	return tree.CreateReturnStatementNode(returnKeyword, expr, semicolon)
}

func (b *BallerinaParser) parseMappingConstructorExpr() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
	openBrace := b.parseOpenBrace()
	fields := b.parseMappingConstructorFields()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return tree.CreateMappingConstructorExpressionNode(openBrace, fields, closeBrace)
}

func (b *BallerinaParser) parseMappingConstructorFields() tree.STNode {
	nextToken := b.peek()
	if b.isEndOfMappingConstructor(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	var fields []tree.STNode
	field := b.parseMappingField(common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD)
	if field != nil {
		fields = append(fields, field)
	}
	return b.finishParseMappingConstructorFields(fields)
}

func (b *BallerinaParser) finishParseMappingConstructorFields(fields []tree.STNode) tree.STNode {
	var nextToken tree.STToken
	var mappingFieldEnd tree.STNode
	nextToken = b.peek()
	for !b.isEndOfMappingConstructor(nextToken.Kind()) {
		mappingFieldEnd = b.parseMappingFieldEnd()
		if mappingFieldEnd == nil {
			break
		}
		fields = append(fields, mappingFieldEnd)
		field := b.parseMappingField(common.PARSER_RULE_CONTEXT_MAPPING_FIELD)
		fields = append(fields, field)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(fields...)
}

func (b *BallerinaParser) parseMappingFieldEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END)
		return b.parseMappingFieldEnd()
	}
}

func (b *BallerinaParser) isEndOfMappingConstructor(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.IDENTIFIER_TOKEN, common.READONLY_KEYWORD:
		return false
	case common.EOF_TOKEN,
		common.DOCUMENTATION_STRING,
		common.AT_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.SEMICOLON_TOKEN,
		common.PUBLIC_KEYWORD,
		common.PRIVATE_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.RETURNS_KEYWORD,
		common.SERVICE_KEYWORD,
		common.TYPE_KEYWORD,
		common.LISTENER_KEYWORD,
		common.CONST_KEYWORD,
		common.FINAL_KEYWORD,
		common.RESOURCE_KEYWORD:
		return true
	default:
		return isSimpleType(tokenKind)
	}
}

func (b *BallerinaParser) parseMappingField(fieldContext common.ParserRuleContext) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		readonlyKeyword := tree.CreateEmptyNode()
		return b.parseSpecificFieldWithOptionalValue(readonlyKeyword)
	case common.STRING_LITERAL_TOKEN:
		readonlyKeyword := tree.CreateEmptyNode()
		return b.parseQualifiedSpecificField(readonlyKeyword)
	case common.READONLY_KEYWORD:
		readonlyKeyword := b.parseReadonlyKeyword()
		return b.parseSpecificField(readonlyKeyword)
	case common.OPEN_BRACKET_TOKEN:
		return b.parseComputedField()
	case common.ELLIPSIS_TOKEN:
		ellipsis := b.parseEllipsis()
		expr := b.parseExpression()
		return tree.CreateSpreadFieldNode(ellipsis, expr)
	case common.CLOSE_BRACE_TOKEN:
		if fieldContext == common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD {
			return nil
		}
		fallthrough
	default:
		b.recoverWithBlockContext(nextToken, fieldContext)
		return b.parseMappingField(fieldContext)
	}
}

func (b *BallerinaParser) parseSpecificField(readonlyKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.STRING_LITERAL_TOKEN:
		return b.parseQualifiedSpecificField(readonlyKeyword)
	case common.IDENTIFIER_TOKEN:
		return b.parseSpecificFieldWithOptionalValue(readonlyKeyword)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD)
		return b.parseSpecificField(readonlyKeyword)
	}
}

func (b *BallerinaParser) parseQualifiedSpecificField(readonlyKeyword tree.STNode) tree.STNode {
	key := b.parseStringLiteral()
	colon := b.parseColon()
	valueExpr := b.parseExpression()
	return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
}

func (b *BallerinaParser) parseSpecificFieldWithOptionalValue(readonlyKeyword tree.STNode) tree.STNode {
	key := b.parseIdentifier(common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME)
	return b.parseSpecificFieldRhs(readonlyKeyword, key)
}

func (b *BallerinaParser) parseSpecificFieldRhs(readonlyKeyword tree.STNode, key tree.STNode) tree.STNode {
	var colon tree.STNode
	var valueExpr tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COLON_TOKEN:
		colon = b.parseColon()
		valueExpr = b.parseExpression()
	case common.COMMA_TOKEN:
		colon = tree.CreateEmptyNode()
		valueExpr = tree.CreateEmptyNode()
	default:
		if b.isEndOfMappingConstructor(nextToken.Kind()) {
			colon = tree.CreateEmptyNode()
			valueExpr = tree.CreateEmptyNode()
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS)
		return b.parseSpecificFieldRhs(readonlyKeyword, key)
	}
	return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
}

func (b *BallerinaParser) parseStringLiteral() tree.STNode {
	token := b.peek()
	var stringLiteral tree.STNode
	if token.Kind() == common.STRING_LITERAL_TOKEN {
		stringLiteral = b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN)
		return b.parseStringLiteral()
	}
	return b.parseBasicLiteralInner(stringLiteral)
}

func (b *BallerinaParser) parseColon() tree.STNode {
	token := b.peek()
	if token.Kind() == common.COLON_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COLON)
		return b.parseColon()
	}
}

func (b *BallerinaParser) parseReadonlyKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.READONLY_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_READONLY_KEYWORD)
		return b.parseReadonlyKeyword()
	}
}

func (b *BallerinaParser) parseComputedField() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME)
	openBracket := b.parseOpenBracket()
	fieldNameExpr := b.parseExpression()
	closeBracket := b.parseCloseBracket()
	b.endContext()
	colon := b.parseColon()
	valueExpr := b.parseExpression()
	return tree.CreateComputedNameFieldNode(openBracket, fieldNameExpr, closeBracket, colon, valueExpr)
}

func (b *BallerinaParser) parseOpenBracket() tree.STNode {
	token := b.peek()
	if token.Kind() == common.OPEN_BRACKET_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OPEN_BRACKET)
		return b.parseOpenBracket()
	}
}

func (b *BallerinaParser) parseCompoundAssignmentStmtRhs(lvExpr tree.STNode) tree.STNode {
	binaryOperator := b.parseCompoundBinaryOperator()
	equalsToken := b.parseAssignOp()
	expr := b.parseActionOrExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	lvExprValid := b.isValidLVExpr(lvExpr)
	if !lvExprValid {
		identifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		simpleNameRef := tree.CreateSimpleNameReferenceNode(identifier)
		lvExpr = tree.CloneWithLeadingInvalidNodeMinutiae(simpleNameRef, lvExpr,
			&common.ERROR_INVALID_EXPR_IN_COMPOUND_ASSIGNMENT_LHS)
	}
	return tree.CreateCompoundAssignmentStatementNode(lvExpr, binaryOperator, equalsToken, expr,
		semicolon)
}

func (b *BallerinaParser) parseCompoundBinaryOperator() tree.STNode {
	token := b.peek()
	if b.isCompoundAssignment(token.Kind()) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR)
		return b.parseCompoundBinaryOperator()
	}
}

func (b *BallerinaParser) parseServiceDeclOrVarDecl(metadata tree.STNode, publicQualifier tree.STNode, qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_SERVICE_DECL)
	serviceDeclQualList, qualifiers := b.extractServiceDeclQualifiers(qualifiers)
	serviceKeyword, qualifiers := b.extractServiceKeyword(qualifiers)
	typeDesc := b.parseServiceDeclTypeDescriptor(qualifiers)
	if (typeDesc != nil) && (typeDesc.Kind() == common.OBJECT_TYPE_DESC) {
		return b.finishParseServiceDeclOrVarDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword,
			typeDesc)
	} else {
		return b.parseServiceDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword, typeDesc)
	}
}

func (b *BallerinaParser) finishParseServiceDeclOrVarDecl(metadata tree.STNode, publicQualifier tree.STNode, serviceDeclQualList []tree.STNode, serviceKeyword tree.STNode, typeDesc tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.SLASH_TOKEN, common.ON_KEYWORD:
		return b.parseServiceDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword, typeDesc)
	case common.OPEN_BRACKET_TOKEN,
		common.IDENTIFIER_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.ERROR_KEYWORD:
		b.endContext()
		typeDesc = b.modifyObjectTypeDescWithALeadingQualifier(typeDesc, serviceKeyword)
		if len(serviceDeclQualList) != 0 {
			isolatedQualifier := serviceDeclQualList[0]
			typeDesc = b.modifyObjectTypeDescWithALeadingQualifier(typeDesc, isolatedQualifier)
		}
		res, _ := b.parseVarDeclTypeDescRhsInner(typeDesc, metadata, publicQualifier, nil, true, true)
		return res
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SERVICE_DECL_OR_VAR_DECL)
		return b.finishParseServiceDeclOrVarDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword,
			typeDesc)
	}
}

func (b *BallerinaParser) extractServiceDeclQualifiers(qualifierList []tree.STNode) ([]tree.STNode, []tree.STNode) {
	var validatedList []tree.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if qualifier.Kind() == common.SERVICE_KEYWORD {
			qualifierList = qualifierList[i:]
			break
		}
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, tree.ToToken(tree.ToToken(qualifier)).Text())
			continue
		}
		if qualifier.Kind() == common.ISOLATED_KEYWORD {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				tree.ToToken(tree.ToToken(qualifier)).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(tree.ToToken(qualifier)).Text())
		}
	}
	return validatedList, qualifierList
}

func (b *BallerinaParser) extractServiceKeyword(qualifierList []tree.STNode) (tree.STNode, []tree.STNode) {
	if len(qualifierList) == 0 {
		panic("assertion failed")
	}
	serviceKeyword := qualifierList[0]
	qualifierList = qualifierList[1:]
	if serviceKeyword.Kind() != common.SERVICE_KEYWORD {
		panic("assertion failed")
	}
	return serviceKeyword, qualifierList
}

func (b *BallerinaParser) parseServiceDecl(metadata tree.STNode, publicQualifier tree.STNode, qualList []tree.STNode, serviceKeyword tree.STNode, serviceType tree.STNode) tree.STNode {
	if publicQualifier != nil {
		if len(qualList) != 0 {
			b.updateFirstNodeInListWithLeadingInvalidNode(qualList, publicQualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED)
		} else {
			serviceKeyword = tree.CloneWithLeadingInvalidNodeMinutiae(serviceKeyword, publicQualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED)
		}
	}
	qualNodeList := tree.CreateNodeList(qualList...)
	resourcePath := b.parseOptionalAbsolutePathOrStringLiteral()
	onKeyword := b.parseOnKeyword()
	expressionList := b.parseListeners()
	openBrace := b.parseOpenBrace()
	objectMembers := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)
	closeBrace := b.parseCloseBrace()
	semicolon := b.parseOptionalSemicolon()
	onKeyword = b.cloneWithDiagnosticIfListEmpty(expressionList, onKeyword, &common.ERROR_MISSING_EXPRESSION)
	b.endContext()
	return tree.CreateServiceDeclarationNode(metadata, qualNodeList, serviceKeyword, serviceType,
		resourcePath, onKeyword, expressionList, openBrace, objectMembers, closeBrace, semicolon)
}

func (b *BallerinaParser) parseServiceDeclTypeDescriptor(qualifiers []tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.SLASH_TOKEN,
		common.ON_KEYWORD,
		common.STRING_LITERAL_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return tree.CreateEmptyNode()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE)
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE)
		return b.parseServiceDeclTypeDescriptor(qualifiers)
	}
}

func (b *BallerinaParser) parseOptionalAbsolutePathOrStringLiteral() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.SLASH_TOKEN:
		return b.parseAbsoluteResourcePath()
	case common.STRING_LITERAL_TOKEN:
		stringLiteralToken := b.consume()
		stringLiteralNode := b.parseBasicLiteralInner(stringLiteralToken)
		return tree.CreateNodeList(stringLiteralNode)
	case common.ON_KEYWORD:
		return tree.CreateEmptyNodeList()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH)
		return b.parseOptionalAbsolutePathOrStringLiteral()
	}
}

func (b *BallerinaParser) parseAbsoluteResourcePath() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH)
	var identifierList []tree.STNode
	nextToken := b.peek()
	var leadingSlash tree.STNode
	isInitialSlash := true
	for !b.isEndAbsoluteResourcePath(nextToken.Kind()) {
		leadingSlash = b.parseAbsoluteResourcePathEnd(isInitialSlash)
		if leadingSlash == nil {
			break
		}
		identifierList = append(identifierList, leadingSlash)
		nextToken = b.peek()
		if isInitialSlash && (nextToken.Kind() == common.ON_KEYWORD) {
			break
		}
		isInitialSlash = false
		leadingSlash = b.parseIdentifier(common.PARSER_RULE_CONTEXT_IDENTIFIER)
		identifierList = append(identifierList, leadingSlash)
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateNodeList(identifierList...)
}

func (b *BallerinaParser) isEndAbsoluteResourcePath(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.EOF_TOKEN, common.ON_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseAbsoluteResourcePathEnd(isInitialSlash bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ON_KEYWORD, common.EOF_TOKEN:
		return nil
	case common.SLASH_TOKEN:
		return b.consume()
	default:
		var context common.ParserRuleContext
		if isInitialSlash {
			context = common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH
		} else {
			context = common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END
		}
		b.recoverWithBlockContext(nextToken, context)
		return b.parseAbsoluteResourcePathEnd(isInitialSlash)
	}
}

// MIGRATION-NOTE: this is used only recursively in Ballerina parser as well, left as is for now.
func (b *BallerinaParser) parseServiceKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.SERVICE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD)
		return b.parseServiceKeyword()
	}
}

func (b *BallerinaParser) isCompoundAssignment(tokenKind common.SyntaxKind) bool {
	return (isCompoundBinaryOperator(tokenKind) && (b.getNextNextToken().Kind() == common.EQUAL_TOKEN))
}

func (b *BallerinaParser) parseOnKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ON_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ON_KEYWORD)
		return b.parseOnKeyword()
	}
}

func (b *BallerinaParser) parseListeners() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LISTENERS_LIST)
	var listeners []tree.STNode
	nextToken := b.peek()
	if b.isEndOfListeners(nextToken.Kind()) {
		b.endContext()
		return tree.CreateEmptyNodeList()
	}
	expr := b.parseExpression()
	listeners = append(listeners, expr)
	var listenersMemberEnd tree.STNode
	for !b.isEndOfListeners(b.peek().Kind()) {
		listenersMemberEnd = b.parseListenersMemberEnd()
		if listenersMemberEnd == nil {
			break
		}
		listeners = append(listeners, listenersMemberEnd)
		expr = b.parseExpression()
		listeners = append(listeners, expr)
	}
	b.endContext()
	return tree.CreateNodeList(listeners...)
}

func (b *BallerinaParser) isEndOfListeners(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.OPEN_BRACE_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseListenersMemberEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.OPEN_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END)
		return b.parseListenersMemberEnd()
	}
}

func (b *BallerinaParser) isServiceDeclStart(currentContext common.ParserRuleContext, lookahead int) bool {
	switch b.peekN(lookahead + 1).Kind() {
	case common.IDENTIFIER_TOKEN:
		tokenAfterIdentifier := b.peekN(lookahead + 2).Kind()
		switch tokenAfterIdentifier {
		case common.ON_KEYWORD,
			// service foo on ...
			common.OPEN_BRACE_TOKEN:
			return true
		case common.EQUAL_TOKEN,
			// service foo = ...
			common.SEMICOLON_TOKEN,
			// service foo;
			common.QUESTION_MARK_TOKEN:
			return false
		default:
			return false
		}
	case common.ON_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseListenerDeclaration(metadata tree.STNode, qualifier tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LISTENER_DECL)
	listenerKeyword := b.parseListenerKeyword()
	if b.peek().Kind() == common.IDENTIFIER_TOKEN {
		listenerDecl := b.parseConstantOrListenerDeclWithOptionalType(metadata, qualifier, listenerKeyword, true)
		b.endContext()
		return listenerDecl
	}
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	variableName := b.parseVariableName()
	equalsToken := b.parseAssignOp()
	initializer := b.parseExpression()
	semicolonToken := b.parseSemicolon()
	b.endContext()
	return tree.CreateListenerDeclarationNode(metadata, qualifier, listenerKeyword, typeDesc, variableName,
		equalsToken, initializer, semicolonToken)
}

func (b *BallerinaParser) parseListenerKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.LISTENER_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD)
		return b.parseListenerKeyword()
	}
}

func (b *BallerinaParser) parseConstantDeclaration(metadata tree.STNode, qualifier tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONSTANT_DECL)
	constKeyword := b.parseConstantKeyword()
	return b.parseConstDecl(metadata, qualifier, constKeyword)
}

func (b *BallerinaParser) parseConstDecl(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ANNOTATION_KEYWORD:
		b.endContext()
		return b.parseAnnotationDeclaration(metadata, qualifier, constKeyword)
	case common.IDENTIFIER_TOKEN:
		constantDecl := b.parseConstantOrListenerDeclWithOptionalType(metadata, qualifier, constKeyword, false)
		b.endContext()
		return constantDecl
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_CONST_DECL_TYPE)
		return b.parseConstDecl(metadata, qualifier, constKeyword)
	}
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	variableName := b.parseVariableName()
	equalsToken := b.parseAssignOp()
	initializer := b.parseExpression()
	semicolonToken := b.parseSemicolon()
	b.endContext()
	return tree.CreateConstantDeclarationNode(metadata, qualifier, constKeyword, typeDesc, variableName,
		equalsToken, initializer, semicolonToken)
}

func (b *BallerinaParser) parseConstantOrListenerDeclWithOptionalType(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode, isListener bool) tree.STNode {
	varNameOrTypeName := b.parseStatementStartIdentifier()
	return b.parseConstantOrListenerDeclRhs(metadata, qualifier, constKeyword, varNameOrTypeName, isListener)
}

func (b *BallerinaParser) parseConstantOrListenerDeclRhs(metadata tree.STNode, qualifier tree.STNode, keyword tree.STNode, typeOrVarName tree.STNode, isListener bool) tree.STNode {
	if typeOrVarName.Kind() == common.QUALIFIED_NAME_REFERENCE {
		ty := typeOrVarName
		variableName := b.parseVariableName()
		return b.parseListenerOrConstRhs(metadata, qualifier, keyword, isListener, ty, variableName)
	}
	var ty tree.STNode
	var variableName tree.STNode
	switch b.peek().Kind() {
	case common.IDENTIFIER_TOKEN:
		ty = typeOrVarName
		variableName = b.parseVariableName()
	case common.EQUAL_TOKEN:
		simpleNameNode, ok := typeOrVarName.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("parseConstantOrListenerDeclRhs: expected STSimpleNameReferenceNode")
		}
		variableName = simpleNameNode.Name
		ty = tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_CONST_DECL_RHS)
		return b.parseConstantOrListenerDeclRhs(metadata, qualifier, keyword, typeOrVarName, isListener)
	}
	return b.parseListenerOrConstRhs(metadata, qualifier, keyword, isListener, ty, variableName)
}

func (b *BallerinaParser) parseListenerOrConstRhs(metadata tree.STNode, qualifier tree.STNode, keyword tree.STNode, isListener bool, ty tree.STNode, variableName tree.STNode) tree.STNode {
	equalsToken := b.parseAssignOp()
	initializer := b.parseExpression()
	semicolonToken := b.parseSemicolon()
	if isListener {
		return tree.CreateListenerDeclarationNode(metadata, qualifier, keyword, ty, variableName,
			equalsToken, initializer, semicolonToken)
	}
	return tree.CreateConstantDeclarationNode(metadata, qualifier, keyword, ty, variableName,
		equalsToken, initializer, semicolonToken)
}

func (b *BallerinaParser) parseConstantKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CONST_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CONST_KEYWORD)
		return b.parseConstantKeyword()
	}
}

func (b *BallerinaParser) parseTypeofExpression(isRhsExpr bool, isInConditionalExpr bool) tree.STNode {
	typeofKeyword := b.parseTypeofKeyword()
	expr := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_UNARY, isRhsExpr, false, isInConditionalExpr)
	return tree.CreateTypeofExpressionNode(typeofKeyword, expr)
}

func (b *BallerinaParser) parseTypeofKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.TYPEOF_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD)
		return b.parseTypeofKeyword()
	}
}

func (b *BallerinaParser) parseOptionalTypeDescriptor(typeDescriptorNode tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR)
	questionMarkToken := b.parseQuestionMark()
	b.endContext()
	return b.createOptionalTypeDesc(typeDescriptorNode, questionMarkToken)
}

func (b *BallerinaParser) createOptionalTypeDesc(typeDescNode tree.STNode, questionMarkToken tree.STNode) tree.STNode {
	if typeDescNode.Kind() == common.UNION_TYPE_DESC {
		unionTypeDesc, ok := typeDescNode.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected tree.STUnionTypeDescriptorNode")
		}
		middleTypeDesc := b.createOptionalTypeDesc(unionTypeDesc.RightTypeDesc, questionMarkToken)
		typeDescNode = b.mergeTypesWithUnion(unionTypeDesc.LeftTypeDesc, unionTypeDesc.PipeToken, middleTypeDesc)
	} else if typeDescNode.Kind() == common.INTERSECTION_TYPE_DESC {
		intersectionTypeDesc, ok := typeDescNode.(*tree.STIntersectionTypeDescriptorNode)
		if !ok {
			panic("expected tree.STIntersectionTypeDescriptorNode")
		}
		middleTypeDesc := b.createOptionalTypeDesc(intersectionTypeDesc.RightTypeDesc, questionMarkToken)
		typeDescNode = b.mergeTypesWithIntersection(intersectionTypeDesc.LeftTypeDesc,
			intersectionTypeDesc.BitwiseAndToken, middleTypeDesc)
	} else {
		typeDescNode = b.validateForUsageOfVar(typeDescNode)
		typeDescNode = tree.CreateOptionalTypeDescriptorNode(typeDescNode, questionMarkToken)
	}
	return typeDescNode
}

func (b *BallerinaParser) parseUnaryExpression(isRhsExpr bool, isInConditionalExpr bool) tree.STNode {
	unaryOperator := b.parseUnaryOperator()
	expr := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_UNARY, isRhsExpr, false, isInConditionalExpr)
	return tree.CreateUnaryExpressionNode(unaryOperator, expr)
}

func (b *BallerinaParser) parseUnaryOperator() tree.STNode {
	token := b.peek()
	if b.isUnaryOperator(token.Kind()) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_UNARY_OPERATOR)
		return b.parseUnaryOperator()
	}
}

func (b *BallerinaParser) isUnaryOperator(kind common.SyntaxKind) bool {
	switch kind {
	case common.PLUS_TOKEN, common.MINUS_TOKEN, common.NEGATION_TOKEN, common.EXCLAMATION_MARK_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseArrayTypeDescriptor(memberTypeDesc tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR)
	openBracketToken := b.parseOpenBracket()
	arrayLengthNode := b.parseArrayLength()
	closeBracketToken := b.parseCloseBracket()
	b.endContext()
	return b.createArrayTypeDesc(memberTypeDesc, openBracketToken, arrayLengthNode, closeBracketToken)
}

func (b *BallerinaParser) createArrayTypeDesc(memberTypeDesc tree.STNode, openBracketToken tree.STNode, arrayLengthNode tree.STNode, closeBracketToken tree.STNode) tree.STNode {
	memberTypeDesc = b.validateForUsageOfVar(memberTypeDesc)
	if arrayLengthNode != nil {
		switch arrayLengthNode.Kind() {
		case common.ASTERISK_LITERAL,
			common.SIMPLE_NAME_REFERENCE,
			common.QUALIFIED_NAME_REFERENCE:
			break
		case common.NUMERIC_LITERAL:
			numericLiteralKind := arrayLengthNode.ChildInBucket(0).Kind()
			if (numericLiteralKind == common.DECIMAL_INTEGER_LITERAL_TOKEN) || (numericLiteralKind == common.HEX_INTEGER_LITERAL_TOKEN) {
				break
			}
		default:
			openBracketToken = tree.CloneWithTrailingInvalidNodeMinutiae(openBracketToken,
				arrayLengthNode, &common.ERROR_INVALID_ARRAY_LENGTH)
			arrayLengthNode = tree.CreateEmptyNode()
		}
	}
	var arrayDimensions []tree.STNode
	if memberTypeDesc.Kind() == common.ARRAY_TYPE_DESC {
		innerArrayType, ok := memberTypeDesc.(*tree.STArrayTypeDescriptorNode)
		if !ok {
			panic("expected tree.STArrayTypeDescriptorNode")
		}
		innerArrayDimensions := innerArrayType.Dimensions
		dimensionCount := innerArrayDimensions.BucketCount()
		i := 0
		for ; i < dimensionCount; i++ {
			arrayDimensions = append(arrayDimensions, innerArrayDimensions.ChildInBucket(i))
		}
		memberTypeDesc = innerArrayType.MemberTypeDesc
	}
	arrayDimension := tree.CreateArrayDimensionNode(openBracketToken, arrayLengthNode,
		closeBracketToken)
	arrayDimensions = append(arrayDimensions, arrayDimension)
	arrayDimensionNodeList := tree.CreateNodeList(arrayDimensions...)
	return tree.CreateArrayTypeDescriptorNode(memberTypeDesc, arrayDimensionNodeList)
}

func (b *BallerinaParser) parseArrayLength() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.ASTERISK_TOKEN:
		return b.parseBasicLiteral()
	case common.CLOSE_BRACKET_TOKEN:
		return tree.CreateEmptyNode()
	case common.IDENTIFIER_TOKEN:
		return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_ARRAY_LENGTH)
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ARRAY_LENGTH)
		return b.parseArrayLength()
	}
}

func (b *BallerinaParser) parseOptionalAnnotations() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATIONS)
	var annotList []tree.STNode
	nextToken := b.peek()
	for nextToken.Kind() == common.AT_TOKEN {
		annotList = append(annotList, b.parseAnnotation())
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateNodeList(annotList...)
}

func (b *BallerinaParser) parseAnnotations() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATIONS)
	var annotList []tree.STNode
	annotList = append(annotList, b.parseAnnotation())
	for b.peek().Kind() == common.AT_TOKEN {
		annotList = append(annotList, b.parseAnnotation())
	}
	b.endContext()
	return tree.CreateNodeList(annotList...)
}

func (b *BallerinaParser) parseAnnotation() tree.STNode {
	atToken := b.parseAtToken()
	var annotReference tree.STNode
	if b.isPredeclaredIdentifier(b.peek().Kind()) {
		annotReference = b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE)
	} else {
		annotReference = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		annotReference = tree.CreateSimpleNameReferenceNode(annotReference)
	}
	var annotValue tree.STNode
	if b.peek().Kind() == common.OPEN_BRACE_TOKEN {
		annotValue = b.parseMappingConstructorExpr()
	} else {
		annotValue = tree.CreateEmptyNode()
	}
	return tree.CreateAnnotationNode(atToken, annotReference, annotValue)
}

func (b *BallerinaParser) parseAtToken() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.AT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_AT)
		return b.parseAtToken()
	}
}

func (b *BallerinaParser) parseMetaData() tree.STNode {
	var docString tree.STNode
	var annotations tree.STNode
	switch b.peek().Kind() {
	case common.DOCUMENTATION_STRING:
		docString = b.parseMarkdownDocumentation()
		annotations = b.parseOptionalAnnotations()
	case common.AT_TOKEN:
		docString = tree.CreateEmptyNode()
		annotations = b.parseOptionalAnnotations()
	default:
		return tree.CreateEmptyNode()
	}
	return b.createMetadata(docString, annotations)
}

func (b *BallerinaParser) createMetadata(docString tree.STNode, annotations tree.STNode) tree.STNode {
	if (annotations == nil) && (docString == nil) {
		return tree.CreateEmptyNode()
	} else {
		return tree.CreateMetadataNode(docString, annotations)
	}
}

func (b *BallerinaParser) parseTypeTestExpression(lhsExpr tree.STNode, isInConditionalExpr bool) tree.STNode {
	isOrNotIsKeyword := b.parseIsOrNotIsKeyword()
	typeDescriptor := b.parseTypeDescriptorInExpression(isInConditionalExpr)
	return tree.CreateTypeTestExpressionNode(lhsExpr, isOrNotIsKeyword, typeDescriptor)
}

func (b *BallerinaParser) parseIsOrNotIsKeyword() tree.STNode {
	token := b.peek()
	if (token.Kind() == common.IS_KEYWORD) || (token.Kind() == common.NOT_IS_KEYWORD) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IS_KEYWORD)
		return b.parseIsOrNotIsKeyword()
	}
}

func (b *BallerinaParser) parseLocalTypeDefinitionStatement(annots tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LOCAL_TYPE_DEFINITION_STMT)
	typeKeyword := b.parseTypeKeyword()
	typeName := b.parseTypeName()
	typeDescriptor := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF)
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateLocalTypeDefinitionStatementNode(annots, typeKeyword, typeName, typeDescriptor,
		semicolon)
}

func (b *BallerinaParser) parseExpressionStatement(annots tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
	expression := b.parseActionOrExpressionInLhs(annots)
	return b.getExpressionAsStatement(expression)
}

func (b *BallerinaParser) parseStatementStartWithExpr(annots tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	expr := b.parseActionOrExpressionInLhs(annots)
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *BallerinaParser) parseStatementStartWithExprRhs(expression tree.STNode) tree.STNode {
	nextTokenKind := b.peek().Kind()
	if b.isAction(expression) || (nextTokenKind == common.SEMICOLON_TOKEN) {
		return b.getExpressionAsStatement(expression)
	}
	switch nextTokenKind {
	case common.EQUAL_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
		return b.parseAssignmentStmtRhs(expression)
	case common.IDENTIFIER_TOKEN:
		fallthrough
	default:
		if b.isCompoundAssignment(nextTokenKind) {
			return b.parseCompoundAssignmentStmtRhs(expression)
		}
		var context common.ParserRuleContext
		if b.isPossibleExpressionStatement(expression) {
			context = common.PARSER_RULE_CONTEXT_EXPR_STMT_RHS
		} else {
			context = common.PARSER_RULE_CONTEXT_STMT_START_WITH_EXPR_RHS
		}
		b.recoverWithBlockContext(b.peek(), context)
		return b.parseStatementStartWithExprRhs(expression)
	}
}

func (b *BallerinaParser) isPossibleExpressionStatement(expression tree.STNode) bool {
	switch expression.Kind() {
	case common.METHOD_CALL,
		common.FUNCTION_CALL,
		common.CHECK_EXPRESSION,
		common.REMOTE_METHOD_CALL_ACTION,
		common.CHECK_ACTION,
		common.BRACED_ACTION,
		common.START_ACTION,
		common.TRAP_ACTION,
		common.FLUSH_ACTION,
		common.ASYNC_SEND_ACTION,
		common.SYNC_SEND_ACTION,
		common.RECEIVE_ACTION,
		common.WAIT_ACTION,
		common.QUERY_ACTION,
		common.COMMIT_ACTION:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) getExpressionAsStatement(expression tree.STNode) tree.STNode {
	switch expression.Kind() {
	case common.METHOD_CALL,
		common.FUNCTION_CALL:
		return b.parseCallStatement(expression)
	case common.CHECK_EXPRESSION:
		return b.parseCheckStatement(expression)
	case common.REMOTE_METHOD_CALL_ACTION,
		common.CHECK_ACTION,
		common.BRACED_ACTION,
		common.START_ACTION,
		common.TRAP_ACTION,
		common.FLUSH_ACTION,
		common.ASYNC_SEND_ACTION,
		common.SYNC_SEND_ACTION,
		common.RECEIVE_ACTION,
		common.WAIT_ACTION,
		common.QUERY_ACTION,
		common.COMMIT_ACTION,
		common.CLIENT_RESOURCE_ACCESS_ACTION:
		return b.parseActionStatement(expression)
	default:
		semicolon := b.parseSemicolon()
		b.endContext()
		expression = b.getExpression(expression)
		exprStmt := tree.CreateExpressionStatementNode(common.INVALID_EXPRESSION_STATEMENT,
			expression, semicolon)
		exprStmt = tree.AddDiagnostic(exprStmt, &common.ERROR_INVALID_EXPRESSION_STATEMENT)
		return exprStmt
	}
}

func (b *BallerinaParser) parseArrayTypeDescriptorNode(indexedExpr tree.STIndexedExpressionNode) tree.STNode {
	memberTypeDesc := b.getTypeDescFromExpr(indexedExpr.ContainerExpression)
	lengthExprs, ok := indexedExpr.KeyExpression.(*tree.STNodeList)
	if !ok {
		panic("expected tree.STNodeList")
	}
	if lengthExprs.IsEmpty() {
		return b.createArrayTypeDesc(memberTypeDesc, indexedExpr.OpenBracket, tree.CreateEmptyNode(),
			indexedExpr.CloseBracket)
	}
	lengthExpr := lengthExprs.Get(0)
	switch lengthExpr.Kind() {
	case common.SIMPLE_NAME_REFERENCE:
		nameRef, ok := lengthExpr.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("expected tree.STSimpleNameReferenceNode")
		}
		if nameRef.Name.IsMissing() {
			return b.createArrayTypeDesc(memberTypeDesc, indexedExpr.OpenBracket, tree.CreateEmptyNode(),
				indexedExpr.CloseBracket)
		}
	case common.ASTERISK_LITERAL,
		common.QUALIFIED_NAME_REFERENCE:
		break
	case common.NUMERIC_LITERAL:
		innerChildKind := lengthExpr.ChildInBucket(0).Kind()
		if (innerChildKind == common.DECIMAL_INTEGER_LITERAL_TOKEN) || (innerChildKind == common.HEX_INTEGER_LITERAL_TOKEN) {
			break
		}
	default:
		newOpenBracketWithDiagnostics := tree.CloneWithTrailingInvalidNodeMinutiae(
			indexedExpr.OpenBracket, lengthExpr, &common.ERROR_INVALID_ARRAY_LENGTH)
		replacedNode := tree.Replace(&indexedExpr, indexedExpr.OpenBracket, newOpenBracketWithDiagnostics)
		newIndexedExpr, ok := replacedNode.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("expected STIndexedExpressionNode")
		}
		indexedExpr = *newIndexedExpr
		lengthExpr = tree.CreateEmptyNode()
	}
	return b.createArrayTypeDesc(memberTypeDesc, indexedExpr.OpenBracket, lengthExpr, indexedExpr.CloseBracket)
}

func (b *BallerinaParser) parseCallStatement(expression tree.STNode) tree.STNode {
	return b.parseCallStatementOrCheckStatement(expression)
}

func (b *BallerinaParser) parseCheckStatement(expression tree.STNode) tree.STNode {
	return b.parseCallStatementOrCheckStatement(expression)
}

func (b *BallerinaParser) parseCallStatementOrCheckStatement(expression tree.STNode) tree.STNode {
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateExpressionStatementNode(common.CALL_STATEMENT, expression, semicolon)
}

func (b *BallerinaParser) parseActionStatement(action tree.STNode) tree.STNode {
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateExpressionStatementNode(common.ACTION_STATEMENT, action, semicolon)
}

func (b *BallerinaParser) parseClientResourceAccessAction(expression tree.STNode, rightArrow tree.STNode, slashToken tree.STNode, isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION)
	resourceAccessPath := b.parseOptionalResourceAccessPath(isRhsExpr, isInMatchGuard)
	resourceAccessMethodDot := b.parseOptionalResourceAccessMethodDot(isRhsExpr, isInMatchGuard)
	resourceAccessMethodName := tree.CreateEmptyNode()
	if resourceAccessMethodDot != nil {
		resourceAccessMethodName = tree.CreateSimpleNameReferenceNode(b.parseFunctionName())
	}
	resourceMethodCallArgList := b.parseOptionalResourceAccessActionArgList(isRhsExpr, isInMatchGuard)
	b.endContext()
	return tree.CreateClientResourceAccessActionNode(expression, rightArrow, slashToken,
		resourceAccessPath, resourceAccessMethodDot, resourceAccessMethodName, resourceMethodCallArgList)
}

func (b *BallerinaParser) parseOptionalResourceAccessPath(isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	resourceAccessPath := tree.CreateEmptyNodeList()
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN,
		common.OPEN_BRACKET_TOKEN:
		resourceAccessPath = b.parseResourceAccessPath(isRhsExpr, isInMatchGuard)
	case common.DOT_TOKEN,
		common.OPEN_PAREN_TOKEN:
		break
	default:
		if b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard) {
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH)
		return b.parseOptionalResourceAccessPath(isRhsExpr, isInMatchGuard)
	}
	return resourceAccessPath
}

func (b *BallerinaParser) parseOptionalResourceAccessMethodDot(isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	dotToken := tree.CreateEmptyNode()
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.DOT_TOKEN:
		dotToken = b.consume()
	case common.OPEN_PAREN_TOKEN:
		break
	default:
		if b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard) {
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD)
		return b.parseOptionalResourceAccessMethodDot(isRhsExpr, isInMatchGuard)
	}
	return dotToken
}

func (b *BallerinaParser) parseOptionalResourceAccessActionArgList(isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	argList := tree.CreateEmptyNode()
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		argList = b.parseParenthesizedArgList()
	default:
		if b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard) {
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST)
		return b.parseOptionalResourceAccessActionArgList(isRhsExpr, isInMatchGuard)
	}
	return argList
}

func (b *BallerinaParser) parseResourceAccessPath(isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	var pathSegmentList []tree.STNode
	pathSegment := b.parseResourceAccessSegment()
	pathSegmentList = append(pathSegmentList, pathSegment)
	var leadingSlash tree.STNode
	previousPathSegmentNode := pathSegment
	for !b.isEndOfResourceAccessPathSegments(b.peek(), isRhsExpr, isInMatchGuard) {
		leadingSlash = b.parseResourceAccessSegmentRhs(isRhsExpr, isInMatchGuard)
		if leadingSlash == nil {
			break
		}
		pathSegment = b.parseResourceAccessSegment()
		if previousPathSegmentNode.Kind() == common.RESOURCE_ACCESS_REST_SEGMENT {
			b.updateLastNodeInListWithInvalidNode(pathSegmentList, leadingSlash, nil)
			b.updateLastNodeInListWithInvalidNode(pathSegmentList, pathSegment,
				&common.RESOURCE_ACCESS_SEGMENT_IS_NOT_ALLOWED_AFTER_REST_SEGMENT)
		} else {
			pathSegmentList = append(pathSegmentList, leadingSlash)
			pathSegmentList = append(pathSegmentList, pathSegment)
			previousPathSegmentNode = pathSegment
		}
	}
	return tree.CreateNodeList(pathSegmentList...)
}

func (b *BallerinaParser) parseResourceAccessSegment() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		return b.consume()
	case common.OPEN_BRACKET_TOKEN:
		return b.parseComputedOrResourceAccessRestSegment(b.consume())
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT)
		return b.parseResourceAccessSegment()
	}
}

func (b *BallerinaParser) parseComputedOrResourceAccessRestSegment(openBracket tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ELLIPSIS_TOKEN:
		ellipsisToken := b.consume()
		expression := b.parseExpression()
		closeBracketToken := b.parseCloseBracket()
		return tree.CreateResourceAccessRestSegmentNode(openBracket, ellipsisToken,
			expression, closeBracketToken)
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			expression := b.parseExpression()
			closeBracketToken := b.parseCloseBracket()
			return tree.CreateComputedResourceAccessSegmentNode(openBracket, expression,
				closeBracketToken)
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT)
		return b.parseComputedOrResourceAccessRestSegment(openBracket)
	}
}

func (b *BallerinaParser) parseResourceAccessSegmentRhs(isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.SLASH_TOKEN:
		return b.consume()
	default:
		if b.isEndOfResourceAccessPathSegments(nextToken, isRhsExpr, isInMatchGuard) {
			return nil
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS)
		return b.parseResourceAccessSegmentRhs(isRhsExpr, isInMatchGuard)
	}
}

func (b *BallerinaParser) isEndOfResourceAccessPathSegments(nextToken tree.STToken, isRhsExpr bool, isInMatchGuard bool) bool {
	switch nextToken.Kind() {
	case common.DOT_TOKEN, common.OPEN_PAREN_TOKEN:
		return true
	default:
		return b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard)
	}
}

func (b *BallerinaParser) parseRemoteMethodCallOrClientResourceAccessOrAsyncSendAction(expression tree.STNode, isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	rightArrow := b.parseRightArrow()
	return b.parseClientResourceAccessOrAsyncSendActionRhs(expression, rightArrow, isRhsExpr, isInMatchGuard)
}

func (b *BallerinaParser) parseClientResourceAccessOrAsyncSendActionRhs(expression tree.STNode, rightArrow tree.STNode, isRhsExpr bool, isInMatchGuard bool) tree.STNode {
	var name tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.FUNCTION_KEYWORD:
		functionKeyword := b.consume()
		name = tree.CreateSimpleNameReferenceNode(functionKeyword)
		return b.parseAsyncSendAction(expression, rightArrow, name)
	case common.CONTINUE_KEYWORD,
		common.COMMIT_KEYWORD:
		name = b.getKeywordAsSimpleNameRef()
	case common.SLASH_TOKEN:
		slashToken := b.consume()
		return b.parseClientResourceAccessAction(expression, rightArrow, slashToken, isRhsExpr, isInMatchGuard)
	default:
		if nextToken.Kind() == common.IDENTIFIER_TOKEN {
			nextNextToken := b.getNextNextToken()
			if ((nextNextToken.Kind() == common.OPEN_PAREN_TOKEN) || b.isEndOfActionOrExpression(nextNextToken, isRhsExpr, isInMatchGuard)) || nextToken.IsMissing() {
				name = tree.CreateSimpleNameReferenceNode(b.parseFunctionName())
				break
			}
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS)
		if solution.Action == ACTION_KEEP {
			name = tree.CreateSimpleNameReferenceNode(b.parseFunctionName())
			break
		}
		return b.parseClientResourceAccessOrAsyncSendActionRhs(expression, rightArrow, isRhsExpr, isInMatchGuard)
	}
	return b.parseRemoteCallOrAsyncSendEnd(expression, rightArrow, name)
}

func (b *BallerinaParser) parseRemoteCallOrAsyncSendEnd(expression tree.STNode, rightArrow tree.STNode, name tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		return b.parseRemoteMethodCallAction(expression, rightArrow, name)
	case common.SEMICOLON_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.COMMA_TOKEN,
		common.FROM_KEYWORD,
		common.JOIN_KEYWORD,
		common.ON_KEYWORD,
		common.LET_KEYWORD,
		common.WHERE_KEYWORD,
		common.ORDER_KEYWORD,
		common.LIMIT_KEYWORD,
		common.SELECT_KEYWORD:
		return b.parseAsyncSendAction(expression, rightArrow, name)
	default:
		if isGroupOrCollectKeyword(nextToken) {
			return b.parseAsyncSendAction(expression, rightArrow, name)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END)
		return b.parseRemoteCallOrAsyncSendEnd(expression, rightArrow, name)
	}
}

func (b *BallerinaParser) parseAsyncSendAction(expression tree.STNode, rightArrow tree.STNode, peerWorker tree.STNode) tree.STNode {
	return tree.CreateAsyncSendActionNode(expression, rightArrow, peerWorker)
}

func (b *BallerinaParser) parseRemoteMethodCallAction(expression tree.STNode, rightArrow tree.STNode, name tree.STNode) tree.STNode {
	openParenToken := b.parseArgListOpenParenthesis()
	arguments := b.parseArgsList()
	closeParenToken := b.parseArgListCloseParenthesis()
	return tree.CreateRemoteMethodCallActionNode(expression, rightArrow, name, openParenToken, arguments,
		closeParenToken)
}

func (b *BallerinaParser) parseRightArrow() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.RIGHT_ARROW_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RIGHT_ARROW)
		return b.parseRightArrow()
	}
}

func (b *BallerinaParser) parseMapTypeDescriptor(mapKeyword tree.STNode) tree.STNode {
	typeParameter := b.parseTypeParameter()
	return tree.CreateMapTypeDescriptorNode(mapKeyword, typeParameter)
}

func (b *BallerinaParser) parseParameterizedTypeDescriptor(keywordToken tree.STNode) tree.STNode {
	var typeParamNode tree.STNode
	nextToken := b.peek()
	if nextToken.Kind() == common.LT_TOKEN {
		typeParamNode = b.parseTypeParameter()
	} else {
		typeParamNode = tree.CreateEmptyNode()
	}
	parameterizedTypeDescKind := b.getParameterizedTypeDescKind(keywordToken)
	return tree.CreateParameterizedTypeDescriptorNode(parameterizedTypeDescKind, keywordToken,
		typeParamNode)
}

func (b *BallerinaParser) getParameterizedTypeDescKind(keywordToken tree.STNode) common.SyntaxKind {
	switch keywordToken.Kind() {
	case common.TYPEDESC_KEYWORD:
		return common.TYPEDESC_TYPE_DESC
	case common.FUTURE_KEYWORD:
		return common.FUTURE_TYPE_DESC
	case common.XML_KEYWORD:
		return common.XML_TYPE_DESC
	default:
		return common.ERROR_TYPE_DESC
	}
}

func (b *BallerinaParser) parseGTToken() tree.STToken {
	nextToken := b.peek()
	if nextToken.Kind() == common.GT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_GT)
		return b.parseGTToken()
	}
}

func (b *BallerinaParser) parseLTToken() tree.STToken {
	nextToken := b.peek()
	if nextToken.Kind() == common.LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_LT)
		return b.parseLTToken()
	}
}

func (b *BallerinaParser) parseNilLiteral() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NIL_LITERAL)
	openParenthesisToken := b.parseOpenParenthesis()
	closeParenthesisToken := b.parseCloseParenthesis()
	b.endContext()
	return tree.CreateNilLiteralNode(openParenthesisToken, closeParenthesisToken)
}

func (b *BallerinaParser) parseAnnotationDeclaration(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATION_DECL)
	annotationKeyword := b.parseAnnotationKeyword()
	annotDecl := b.parseAnnotationDeclFromType(metadata, qualifier, constKeyword, annotationKeyword)
	b.endContext()
	return annotDecl
}

func (b *BallerinaParser) parseAnnotationKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ANNOTATION_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD)
		return b.parseAnnotationKeyword()
	}
}

func (b *BallerinaParser) parseAnnotationDeclFromType(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode, annotationKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		return b.parseAnnotationDeclWithOptionalType(metadata, qualifier, constKeyword, annotationKeyword)
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE)
		return b.parseAnnotationDeclFromType(metadata, qualifier, constKeyword, annotationKeyword)
	}
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL)
	annotTag := b.parseAnnotationTag()
	return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword, typeDesc,
		annotTag)
}

func (b *BallerinaParser) parseAnnotationTag() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANNOTATION_TAG)
		return b.parseAnnotationTag()
	}
}

func (b *BallerinaParser) parseAnnotationDeclWithOptionalType(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode, annotationKeyword tree.STNode) tree.STNode {
	typeDescOrAnnotTag := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE)
	if typeDescOrAnnotTag.Kind() == common.QUALIFIED_NAME_REFERENCE {
		annotTag := b.parseAnnotationTag()
		return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword,
			typeDescOrAnnotTag, annotTag)
	}
	nextToken := b.peek()
	if (nextToken.Kind() == common.IDENTIFIER_TOKEN) || b.isValidTypeContinuationToken(nextToken) {
		typeDesc := b.parseComplexTypeDescriptor(typeDescOrAnnotTag,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL, false)
		annotTag := b.parseAnnotationTag()
		return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword, typeDesc,
			annotTag)
	}
	simplenameNode, ok := typeDescOrAnnotTag.(*tree.STSimpleNameReferenceNode)
	if !ok {
		panic("parseAnnotationDeclWithOptionalType: expected STSimpleNameReferenceNode")
	}
	annotTag := simplenameNode.Name
	return b.parseAnnotationDeclRhs(metadata, qualifier, constKeyword, annotationKeyword, annotTag)
}

func (b *BallerinaParser) parseAnnotationDeclRhs(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode, annotationKeyword tree.STNode, typeDescOrAnnotTag tree.STNode) tree.STNode {
	nextToken := b.peek()
	var typeDesc tree.STNode
	var annotTag tree.STNode
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		typeDesc = typeDescOrAnnotTag
		annotTag = b.parseAnnotationTag()
	case common.SEMICOLON_TOKEN,
		common.ON_KEYWORD:
		typeDesc = tree.CreateEmptyNode()
		annotTag = typeDescOrAnnotTag
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANNOT_DECL_RHS)
		return b.parseAnnotationDeclRhs(metadata, qualifier, constKeyword, annotationKeyword, typeDescOrAnnotTag)
	}
	return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword, typeDesc,
		annotTag)
}

func (b *BallerinaParser) parseAnnotationDeclAttachPoints(metadata tree.STNode, qualifier tree.STNode, constKeyword tree.STNode, annotationKeyword tree.STNode, typeDesc tree.STNode, annotTag tree.STNode) tree.STNode {
	var onKeyword tree.STNode
	var attachPoints tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.SEMICOLON_TOKEN:
		onKeyword = tree.CreateEmptyNode()
		attachPoints = tree.CreateEmptyNodeList()
	case common.ON_KEYWORD:
		onKeyword = b.parseOnKeyword()
		attachPoints = b.parseAnnotationAttachPoints()
		onKeyword = b.cloneWithDiagnosticIfListEmpty(attachPoints, onKeyword,
			&common.ERROR_MISSING_ANNOTATION_ATTACH_POINT)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS)
		return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword, typeDesc,
			annotTag)
	}
	semicolonToken := b.parseSemicolon()
	return tree.CreateAnnotationDeclarationNode(metadata, qualifier, constKeyword, annotationKeyword,
		typeDesc, annotTag, onKeyword, attachPoints, semicolonToken)
}

func (b *BallerinaParser) parseAnnotationAttachPoints() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST)
	var attachPoints []tree.STNode
	nextToken := b.peek()
	if b.isEndAnnotAttachPointList(nextToken.Kind()) {
		b.endContext()
		return tree.CreateEmptyNodeList()
	}
	attachPoint := b.parseAnnotationAttachPoint()
	attachPoints = append(attachPoints, attachPoint)
	nextToken = b.peek()
	var leadingComma tree.STNode
	for !b.isEndAnnotAttachPointList(nextToken.Kind()) {
		leadingComma = b.parseAttachPointEnd()
		if leadingComma == nil {
			break
		}
		attachPoints = append(attachPoints, leadingComma)
		attachPoint = b.parseAnnotationAttachPoint()
		if attachPoint == nil {
			missingAttachPointIdent := tree.CreateMissingToken(common.TYPE_KEYWORD, nil)
			identList := tree.CreateNodeList(missingAttachPointIdent)
			attachPoint = tree.CreateAnnotationAttachPointNode(tree.CreateEmptyNode(), identList)
			attachPoint = tree.AddDiagnostic(attachPoint,
				&common.ERROR_MISSING_ANNOTATION_ATTACH_POINT)
			attachPoints = append(attachPoints, attachPoint)
			break
		}
		attachPoints = append(attachPoints, attachPoint)
		nextToken = b.peek()
	}
	if (tree.LastToken(attachPoint).IsMissing() && (b.tokenReader.Peek().Kind() == common.IDENTIFIER_TOKEN)) && (!b.tokenReader.Head().HasTrailingNewLine()) {
		nextNonVirtualToken := b.tokenReader.Read()
		b.updateLastNodeInListWithInvalidNode(attachPoints, nextNonVirtualToken,
			&common.ERROR_INVALID_TOKEN, nextNonVirtualToken.Text())
	}
	b.endContext()
	return tree.CreateNodeList(attachPoints...)
}

func (b *BallerinaParser) parseAttachPointEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.SEMICOLON_TOKEN:
		return nil
	case common.COMMA_TOKEN:
		return b.consume()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ATTACH_POINT_END)
		return b.parseAttachPointEnd()
	}
}

func (b *BallerinaParser) isEndAnnotAttachPointList(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.EOF_TOKEN, common.SEMICOLON_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseAnnotationAttachPoint() tree.STNode {
	switch b.peek().Kind() {
	case common.EOF_TOKEN:
		return nil
	case common.ANNOTATION_KEYWORD,
		common.EXTERNAL_KEYWORD,
		common.VAR_KEYWORD,
		common.CONST_KEYWORD,
		common.LISTENER_KEYWORD,
		common.WORKER_KEYWORD,
		common.SOURCE_KEYWORD:
		sourceKeyword := b.parseSourceKeyword()
		return b.parseAttachPointIdent(sourceKeyword)
	case common.OBJECT_KEYWORD,
		common.TYPE_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.PARAMETER_KEYWORD,
		common.RETURN_KEYWORD,
		common.SERVICE_KEYWORD,
		common.FIELD_KEYWORD,
		common.RECORD_KEYWORD,
		common.CLASS_KEYWORD:
		sourceKeyword := tree.CreateEmptyNode()
		firstIdent := b.consume()
		return b.parseDualAttachPointIdent(sourceKeyword, firstIdent)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ATTACH_POINT)
		return b.parseAnnotationAttachPoint()
	}
}

func (b *BallerinaParser) parseSourceKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.SOURCE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD)
		return b.parseSourceKeyword()
	}
}

func (b *BallerinaParser) parseAttachPointIdent(sourceKeyword tree.STNode) tree.STNode {
	switch b.peek().Kind() {
	case common.ANNOTATION_KEYWORD,
		common.EXTERNAL_KEYWORD,
		common.VAR_KEYWORD,
		common.CONST_KEYWORD,
		common.LISTENER_KEYWORD,
		common.WORKER_KEYWORD:
		firstIdent := b.consume()
		identList := tree.CreateNodeList(firstIdent)
		return tree.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	case common.OBJECT_KEYWORD,
		common.RESOURCE_KEYWORD,
		common.RECORD_KEYWORD,
		common.TYPE_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.PARAMETER_KEYWORD,
		common.RETURN_KEYWORD,
		common.SERVICE_KEYWORD,
		common.FIELD_KEYWORD,
		common.CLASS_KEYWORD:
		firstIdent := b.consume()
		return b.parseDualAttachPointIdent(sourceKeyword, firstIdent)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT)
		return b.parseAttachPointIdent(sourceKeyword)
	}
}

func (b *BallerinaParser) parseDualAttachPointIdent(sourceKeyword tree.STNode, firstIdent tree.STNode) tree.STNode {
	var secondIdent tree.STNode
	switch firstIdent.Kind() {
	case common.OBJECT_KEYWORD:
		secondIdent = b.parseIdentAfterObjectIdent()
	case common.RESOURCE_KEYWORD:
		secondIdent = b.parseFunctionIdent()
	case common.RECORD_KEYWORD:
		secondIdent = b.parseFieldIdent()
	case common.SERVICE_KEYWORD:
		return b.parseServiceAttachPoint(sourceKeyword, firstIdent)
	case common.TYPE_KEYWORD, common.FUNCTION_KEYWORD, common.PARAMETER_KEYWORD,
		common.RETURN_KEYWORD, common.FIELD_KEYWORD, common.CLASS_KEYWORD:
		fallthrough
	default:
		identList := tree.CreateNodeList(firstIdent)
		return tree.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	}
	identList := tree.CreateNodeList(firstIdent, secondIdent)
	return tree.CreateAnnotationAttachPointNode(sourceKeyword, identList)
}

func (b *BallerinaParser) parseRemoteIdent() tree.STNode {
	token := b.peek()
	if token.Kind() == common.REMOTE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_REMOTE_IDENT)
		return b.parseRemoteIdent()
	}
}

func (b *BallerinaParser) parseServiceAttachPoint(sourceKeyword tree.STNode, firstIdent tree.STNode) tree.STNode {
	var identList tree.STNode
	token := b.peek()
	switch token.Kind() {
	case common.REMOTE_KEYWORD:
		secondIdent := b.parseRemoteIdent()
		thirdIdent := b.parseFunctionIdent()
		identList = tree.CreateNodeList(firstIdent, secondIdent, thirdIdent)
		return tree.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	case common.COMMA_TOKEN,
		common.SEMICOLON_TOKEN:
		identList = tree.CreateNodeList(firstIdent)
		return tree.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS)
		return b.parseServiceAttachPoint(sourceKeyword, firstIdent)
	}
}

func (b *BallerinaParser) parseIdentAfterObjectIdent() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.FUNCTION_KEYWORD, common.FIELD_KEYWORD:
		return b.consume()
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT)
		return b.parseIdentAfterObjectIdent()
	}
}

func (b *BallerinaParser) parseFunctionIdent() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FUNCTION_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNCTION_IDENT)
		return b.parseFunctionIdent()
	}
}

func (b *BallerinaParser) parseFieldIdent() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FIELD_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FIELD_IDENT)
		return b.parseFieldIdent()
	}
}

func (b *BallerinaParser) parseXMLNamespaceDeclaration(isModuleVar bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION)
	xmlnsKeyword := b.parseXMLNSKeyword()
	namespaceUri := b.parseSimpleConstExpr()
	for !b.isValidXMLNameSpaceURI(namespaceUri) {
		xmlnsKeyword = tree.CloneWithTrailingInvalidNodeMinutiae(xmlnsKeyword, namespaceUri,
			&common.ERROR_INVALID_XML_NAMESPACE_URI)
		namespaceUri = b.parseSimpleConstExpr()
	}
	xmlnsDecl := b.parseXMLDeclRhs(xmlnsKeyword, namespaceUri, isModuleVar)
	b.endContext()
	return xmlnsDecl
}

func (b *BallerinaParser) parseXMLNSKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.XMLNS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD)
		return b.parseXMLNSKeyword()
	}
}

func (b *BallerinaParser) isValidXMLNameSpaceURI(expr tree.STNode) bool {
	switch expr.Kind() {
	case common.STRING_LITERAL, common.QUALIFIED_NAME_REFERENCE, common.SIMPLE_NAME_REFERENCE:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseSimpleConstExpr() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION)
	expr := b.parseSimpleConstExprInternal()
	b.endContext()
	return expr
}

func (b *BallerinaParser) parseSimpleConstExprInternal() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.STRING_LITERAL_TOKEN,
		common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.NULL_KEYWORD:
		return b.parseBasicLiteral()
	case common.PLUS_TOKEN, common.MINUS_TOKEN:
		return b.parseSignedIntOrFloat()
	case common.OPEN_PAREN_TOKEN:
		return b.parseNilLiteral()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START)
		return b.parseSimpleConstExprInternal()
	}
}

func (b *BallerinaParser) parseXMLDeclRhs(xmlnsKeyword tree.STNode, namespaceUri tree.STNode, isModuleVar bool) tree.STNode {
	asKeyword := tree.CreateEmptyNode()
	namespacePrefix := tree.CreateEmptyNode()
	switch b.peek().Kind() {
	case common.AS_KEYWORD:
		asKeyword = b.parseAsKeyword()
		namespacePrefix = b.parseNamespacePrefix()
	case common.SEMICOLON_TOKEN:
		break
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL)
		return b.parseXMLDeclRhs(xmlnsKeyword, namespaceUri, isModuleVar)
	}
	semicolon := b.parseSemicolon()
	if isModuleVar {
		return tree.CreateModuleXMLNamespaceDeclarationNode(xmlnsKeyword, namespaceUri, asKeyword,
			namespacePrefix, semicolon)
	}
	return tree.CreateXMLNamespaceDeclarationNode(xmlnsKeyword, namespaceUri, asKeyword, namespacePrefix,
		semicolon)
}

func (b *BallerinaParser) parseNamespacePrefix() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_NAMESPACE_PREFIX)
		return b.parseNamespacePrefix()
	}
}

func (b *BallerinaParser) parseNamedWorkerDeclaration(annots tree.STNode, qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL)
	transactionalKeyword := b.getTransactionalKeyword(qualifiers)
	workerKeyword := b.parseWorkerKeyword()
	workerName := b.parseWorkerName()
	returnTypeDesc := b.parseReturnTypeDescriptor()
	workerBody := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateNamedWorkerDeclarationNode(annots, transactionalKeyword, workerKeyword, workerName,
		returnTypeDesc, workerBody, onFailClause)
}

func (b *BallerinaParser) getTransactionalKeyword(qualifierList []tree.STNode) tree.STNode {
	var validatedList []tree.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			qualifierToken, ok := qualifier.(tree.STToken)
			if !ok {
				panic("expected STToken")
			}
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, qualifierToken.Text())
		} else if qualifier.Kind() == common.TRANSACTIONAL_KEYWORD {
			validatedList = append(validatedList, qualifier)
		} else if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	var transactionalKeyword tree.STNode
	if len(validatedList) == 0 {
		transactionalKeyword = tree.CreateEmptyNode()
	} else {
		transactionalKeyword = validatedList[0]
	}
	return transactionalKeyword
}

func (b *BallerinaParser) parseReturnTypeDescriptor() tree.STNode {
	token := b.peek()
	if token.Kind() != common.RETURNS_KEYWORD {
		return tree.CreateEmptyNode()
	}
	returnsKeyword := b.consume()
	annot := b.parseOptionalAnnotations()
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC)
	return tree.CreateReturnTypeDescriptorNode(returnsKeyword, annot, ty)
}

func (b *BallerinaParser) parseWorkerKeyword() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.WORKER_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WORKER_KEYWORD)
		return b.parseWorkerKeyword()
	}
}

func (b *BallerinaParser) parseWorkerName() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WORKER_NAME)
		return b.parseWorkerName()
	}
}

func (b *BallerinaParser) parseLockStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LOCK_STMT)
	lockKeyword := b.parseLockKeyword()
	blockStatement := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateLockStatementNode(lockKeyword, blockStatement, onFailClause)
}

func (b *BallerinaParser) parseLockKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.LOCK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LOCK_KEYWORD)
		return b.parseLockKeyword()
	}
}

func (b *BallerinaParser) parseUnionTypeDescriptor(leftTypeDesc tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool) tree.STNode {
	pipeToken := b.consume()
	rightTypeDesc := b.parseTypeDescriptorInternalWithPrecedence(nil, context, isTypedBindingPattern, false,
		TYPE_PRECEDENCE_UNION)
	return b.mergeTypesWithUnion(leftTypeDesc, pipeToken, rightTypeDesc)
}

func (b *BallerinaParser) createUnionTypeDesc(leftTypeDesc tree.STNode, pipeToken tree.STNode, rightTypeDesc tree.STNode) tree.STNode {
	leftTypeDesc = b.validateForUsageOfVar(leftTypeDesc)
	rightTypeDesc = b.validateForUsageOfVar(rightTypeDesc)
	return tree.CreateUnionTypeDescriptorNode(leftTypeDesc, pipeToken, rightTypeDesc)
}

func (b *BallerinaParser) parsePipeToken() tree.STNode {
	token := b.peek()
	if token.Kind() == common.PIPE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PIPE)
		return b.parsePipeToken()
	}
}

func (b *BallerinaParser) isTypeStartingToken(nodeKind common.SyntaxKind) bool {
	return isTypeStartingToken(nodeKind, b.getNextNextToken())
}

func (b *BallerinaParser) isSimpleTypeInExpression(nodeKind common.SyntaxKind) bool {
	switch nodeKind {
	case common.VAR_KEYWORD, common.READONLY_KEYWORD:
		return false
	default:
		return isSimpleType(nodeKind)
	}
}

func (b *BallerinaParser) isQualifiedIdentifierPredeclaredPrefix(nodeKind common.SyntaxKind) bool {
	return (isPredeclaredPrefix(nodeKind) && (b.getNextNextToken().Kind() == common.COLON_TOKEN))
}

func (b *BallerinaParser) parseForkKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FORK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FORK_KEYWORD)
		return b.parseForkKeyword()
	}
}

func (b *BallerinaParser) parseForkStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FORK_STMT)
	forkKeyword := b.parseForkKeyword()
	openBrace := b.parseOpenBrace()
	var workers []tree.STNode
	for !b.isEndOfStatements() {
		stmt := b.parseStatement()
		if stmt == nil {
			break
		}
		if b.validateStatement(stmt) {
			continue
		}
		switch stmt.Kind() {
		case common.NAMED_WORKER_DECLARATION:
			workers = append(workers, stmt)
		default:
			if len(workers) == 0 {
				openBrace = tree.CloneWithTrailingInvalidNodeMinutiae(openBrace, stmt,
					&common.ERROR_ONLY_NAMED_WORKERS_ALLOWED_HERE)
			} else {
				b.updateLastNodeInListWithInvalidNode(workers, stmt,
					&common.ERROR_ONLY_NAMED_WORKERS_ALLOWED_HERE)
			}
		}
	}
	namedWorkerDeclarations := tree.CreateNodeList(workers...)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	forkStmt := tree.CreateForkStatementNode(forkKeyword, openBrace, namedWorkerDeclarations, closeBrace)
	if b.isNodeListEmpty(namedWorkerDeclarations) {
		return tree.AddDiagnostic(forkStmt,
			&common.ERROR_MISSING_NAMED_WORKER_DECLARATION_IN_FORK_STMT)
	}
	return forkStmt
}

func (b *BallerinaParser) parseTrapExpression(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	trapKeyword := b.parseTrapKeyword()
	expr := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_TRAP, isRhsExpr, allowActions, isInConditionalExpr)
	if b.isAction(expr) {
		return tree.CreateTrapExpressionNode(common.TRAP_ACTION, trapKeyword, expr)
	}
	return tree.CreateTrapExpressionNode(common.TRAP_EXPRESSION, trapKeyword, expr)
}

func (b *BallerinaParser) parseTrapKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.TRAP_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TRAP_KEYWORD)
		return b.parseTrapKeyword()
	}
}

func (b *BallerinaParser) parseListConstructorExpr() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR)
	openBracket := b.parseOpenBracket()
	listMembers := b.parseListMembers()
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return tree.CreateListConstructorExpressionNode(openBracket, listMembers, closeBracket)
}

func (b *BallerinaParser) parseListMembers() tree.STNode {
	var listMembers []tree.STNode
	if b.isEndOfListConstructor(b.peek().Kind()) {
		return tree.CreateEmptyNodeList()
	}
	listMember := b.parseListMember()
	listMembers = append(listMembers, listMember)
	return b.parseListMembersInner(listMembers)
}

func (b *BallerinaParser) parseListMembersInner(listMembers []tree.STNode) tree.STNode {
	var listConstructorMemberEnd tree.STNode
	for !b.isEndOfListConstructor(b.peek().Kind()) {
		listConstructorMemberEnd = b.parseListConstructorMemberEnd()
		if listConstructorMemberEnd == nil {
			break
		}
		listMembers = append(listMembers, listConstructorMemberEnd)
		listMember := b.parseListMember()
		listMembers = append(listMembers, listMember)
	}
	return tree.CreateNodeList(listMembers...)
}

func (b *BallerinaParser) parseListMember() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.ELLIPSIS_TOKEN {
		return b.parseSpreadMember()
	} else {
		return b.parseExpression()
	}
}

func (b *BallerinaParser) parseSpreadMember() tree.STNode {
	ellipsis := b.parseEllipsis()
	expr := b.parseExpression()
	return tree.CreateSpreadMemberNode(ellipsis, expr)
}

func (b *BallerinaParser) isEndOfListConstructor(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACKET_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseListConstructorMemberEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return b.consume()
	case common.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END)
		return b.parseListConstructorMemberEnd()
	}
}

func (b *BallerinaParser) parseForEachStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FOREACH_STMT)
	forEachKeyword := b.parseForEachKeyword()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_FOREACH_STMT)
	inKeyword := b.parseInKeyword()
	actionOrExpr := b.parseActionOrExpression()
	blockStatement := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateForEachStatementNode(forEachKeyword, typedBindingPattern, inKeyword, actionOrExpr,
		blockStatement, onFailClause)
}

func (b *BallerinaParser) parseForEachKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FOREACH_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD)
		return b.parseForEachKeyword()
	}
}

func (b *BallerinaParser) parseInKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.IN_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IN_KEYWORD)
		return b.parseInKeyword()
	}
}

func (b *BallerinaParser) parseTypeCastExpr(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TYPE_CAST)
	ltToken := b.parseLTToken()
	return b.parseTypeCastExprInner(ltToken, isRhsExpr, allowActions, isInConditionalExpr)
}

func (b *BallerinaParser) parseTypeCastExprInner(ltToken tree.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) tree.STNode {
	typeCastParam := b.parseTypeCastParam()
	gtToken := b.parseGTToken()
	b.endContext()
	expression := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_EXPRESSION_ACTION, isRhsExpr, allowActions, isInConditionalExpr)
	return tree.CreateTypeCastExpressionNode(ltToken, typeCastParam, gtToken, expression)
}

func (b *BallerinaParser) parseTypeCastParam() tree.STNode {
	var annot tree.STNode
	var ty tree.STNode
	token := b.peek()
	switch token.Kind() {
	case common.AT_TOKEN:
		annot = b.parseOptionalAnnotations()
		token = b.peek()
		if b.isTypeStartingToken(token.Kind()) {
			ty = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS)
		} else {
			ty = tree.CreateEmptyNode()
		}
	default:
		annot = tree.CreateEmptyNode()
		ty = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS)
	}
	return tree.CreateTypeCastParamNode(b.getAnnotations(annot), ty)
}

func (b *BallerinaParser) parseTableConstructorExprRhs(tableKeyword tree.STNode, keySpecifier tree.STNode) tree.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR)
	openBracket := b.parseOpenBracket()
	rowList := b.parseRowList()
	closeBracket := b.parseCloseBracket()
	return tree.CreateTableConstructorExpressionNode(tableKeyword, keySpecifier, openBracket, rowList,
		closeBracket)
}

func (b *BallerinaParser) parseTableKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.TABLE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TABLE_KEYWORD)
		return b.parseTableKeyword()
	}
}

func (b *BallerinaParser) parseRowList() tree.STNode {
	nextToken := b.peek()
	if b.isEndOfTableRowList(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	var mappings []tree.STNode
	mapExpr := b.parseMappingConstructorExpr()
	mappings = append(mappings, mapExpr)
	nextToken = b.peek()
	var rowEnd tree.STNode
	for !b.isEndOfTableRowList(nextToken.Kind()) {
		rowEnd = b.parseTableRowEnd()
		if rowEnd == nil {
			break
		}
		mappings = append(mappings, rowEnd)
		mapExpr = b.parseMappingConstructorExpr()
		mappings = append(mappings, mapExpr)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(mappings...)
}

func (b *BallerinaParser) isEndOfTableRowList(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACKET_TOKEN:
		return true
	case common.COMMA_TOKEN, common.OPEN_BRACE_TOKEN:
		return false
	default:
		return b.isEndOfMappingConstructor(tokenKind)
	}
}

func (b *BallerinaParser) parseTableRowEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACKET_TOKEN, common.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_ROW_END)
		return b.parseTableRowEnd()
	}
}

func (b *BallerinaParser) parseKeySpecifier() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_KEY_SPECIFIER)
	keyKeyword := b.parseKeyKeyword()
	openParen := b.parseOpenParenthesis()
	fieldNames := b.parseFieldNames()
	closeParen := b.parseCloseParenthesis()
	b.endContext()
	return tree.CreateKeySpecifierNode(keyKeyword, openParen, fieldNames, closeParen)
}

func (b *BallerinaParser) parseKeyKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.KEY_KEYWORD {
		return b.consume()
	}
	if isKeyKeyword(token) {
		return b.getKeyKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_KEY_KEYWORD)
	return b.parseKeyKeyword()
}

func (b *BallerinaParser) getKeyKeyword(token tree.STToken) tree.STNode {
	return tree.CreateTokenWithDiagnostics(common.KEY_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *BallerinaParser) getUnderscoreKeyword(token tree.STToken) tree.STToken {
	return tree.CreateTokenWithDiagnostics(common.UNDERSCORE_KEYWORD, token.LeadingMinutiae(),
		token.TrailingMinutiae(), token.Diagnostics())
}

func (b *BallerinaParser) parseNaturalKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.NATURAL_KEYWORD {
		return b.consume()
	}
	if b.isNaturalKeyword(token) {
		return b.getNaturalKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD)
	return b.parseNaturalKeyword()
}

func (b *BallerinaParser) isNaturalKeyword(node tree.STNode) bool {
	token, isToken := node.(tree.STToken)
	if isToken {
		return isNaturalKeyword(token)
	}
	if node.Kind() != common.SIMPLE_NAME_REFERENCE {
		return false
	}
	simpleNameNode, ok := node.(*tree.STSimpleNameReferenceNode)
	if !ok {
		panic("isNaturalKeyword: expected STSimpleNameReferenceNode")
	}
	nameToken, ok := simpleNameNode.Name.(tree.STToken)
	if !ok {
		panic("isNaturalKeyword: expected STToken")
	}
	return isNaturalKeyword(nameToken)
}

func (b *BallerinaParser) getNaturalKeyword(token tree.STToken) tree.STNode {
	return tree.CreateTokenWithDiagnostics(common.NATURAL_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *BallerinaParser) parseFieldNames() tree.STNode {
	nextToken := b.peek()
	if b.isEndOfFieldNamesList(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	var fieldNames []tree.STNode
	fieldName := b.parseVariableName()
	fieldNames = append(fieldNames, fieldName)
	nextToken = b.peek()
	var leadingComma tree.STNode
	for !b.isEndOfFieldNamesList(nextToken.Kind()) {
		leadingComma = b.parseComma()
		fieldNames = append(fieldNames, leadingComma)
		fieldName = b.parseVariableName()
		fieldNames = append(fieldNames, fieldName)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(fieldNames...)
}

func (b *BallerinaParser) isEndOfFieldNamesList(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.COMMA_TOKEN, common.IDENTIFIER_TOKEN:
		return false
	default:
		return true
	}
}

func (b *BallerinaParser) parseErrorKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ERROR_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ERROR_KEYWORD)
		return b.parseErrorKeyword()
	}
}

func (b *BallerinaParser) parseStreamTypeDescriptor(streamKeywordToken tree.STNode) tree.STNode {
	var streamTypeParamsNode tree.STNode
	nextToken := b.peek()
	if nextToken.Kind() == common.LT_TOKEN {
		streamTypeParamsNode = b.parseStreamTypeParamsNode()
	} else {
		streamTypeParamsNode = tree.CreateEmptyNode()
	}
	return tree.CreateStreamTypeDescriptorNode(streamKeywordToken, streamTypeParamsNode)
}

func (b *BallerinaParser) parseStreamTypeParamsNode() tree.STNode {
	ltToken := b.parseLTToken()
	b.startContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC)
	leftTypeDescNode := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC)
	streamTypedesc := b.parseStreamTypeParamsNodeInner(ltToken, leftTypeDescNode)
	b.endContext()
	return streamTypedesc
}

func (b *BallerinaParser) parseStreamTypeParamsNodeInner(ltToken tree.STNode, leftTypeDescNode tree.STNode) tree.STNode {
	var commaToken tree.STNode
	var rightTypeDescNode tree.STNode
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		commaToken = b.parseComma()
		rightTypeDescNode = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC)
	case common.GT_TOKEN:
		commaToken = tree.CreateEmptyNode()
		rightTypeDescNode = tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS)
		return b.parseStreamTypeParamsNodeInner(ltToken, leftTypeDescNode)
	}
	gtToken := b.parseGTToken()
	return tree.CreateStreamTypeParamsNode(ltToken, leftTypeDescNode, commaToken, rightTypeDescNode,
		gtToken)
}

func (b *BallerinaParser) parseLetExpression(isRhsExpr bool, isInConditionalExpr bool) tree.STNode {
	letKeyword := b.parseLetKeyword()
	letVarDeclarations := b.parseLetVarDeclarations(common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL, isRhsExpr, false)
	inKeyword := b.parseInKeyword()
	letKeyword = b.cloneWithDiagnosticIfListEmpty(letVarDeclarations, letKeyword,
		&common.ERROR_MISSING_LET_VARIABLE_DECLARATION)
	expression := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION, isRhsExpr, false,
		isInConditionalExpr)
	return tree.CreateLetExpressionNode(letKeyword, letVarDeclarations, inKeyword, expression)
}

func (b *BallerinaParser) parseLetKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.LET_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LET_KEYWORD)
		return b.parseLetKeyword()
	}
}

func (b *BallerinaParser) parseLetVarDeclarations(context common.ParserRuleContext, isRhsExpr bool, allowActions bool) tree.STNode {
	b.startContext(context)
	var varDecls []tree.STNode
	nextToken := b.peek()
	if isEndOfLetVarDeclarations(nextToken, b.getNextNextToken()) {
		b.endContext()
		return tree.CreateEmptyNodeList()
	}
	varDec := b.parseLetVarDecl(context, isRhsExpr, allowActions)
	varDecls = append(varDecls, varDec)
	nextToken = b.peek()
	var leadingComma tree.STNode
	for !isEndOfLetVarDeclarations(nextToken, b.getNextNextToken()) {
		leadingComma = b.parseComma()
		varDecls = append(varDecls, leadingComma)
		varDec = b.parseLetVarDecl(context, isRhsExpr, allowActions)
		varDecls = append(varDecls, varDec)
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateNodeList(varDecls...)
}

func (b *BallerinaParser) parseLetVarDecl(context common.ParserRuleContext, isRhsExpr bool, allowActions bool) tree.STNode {
	annot := b.parseOptionalAnnotations()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL)
	assign := b.parseAssignOp()
	var expression tree.STNode
	if context == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
		expression = b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, allowActions)
	} else {
		expression = b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET, isRhsExpr, false)
	}
	return tree.CreateLetVariableDeclarationNode(annot, typedBindingPattern, assign, expression)
}

func (b *BallerinaParser) parseTemplateExpression() tree.STNode {
	ty := tree.CreateEmptyNode()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	content := b.parseTemplateContent()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	return tree.CreateTemplateExpressionNode(common.RAW_TEMPLATE_EXPRESSION, ty, startingBackTick,
		content, endingBackTick)
}

func (b *BallerinaParser) parseTemplateContent() tree.STNode {
	var items []tree.STNode
	nextToken := b.peek()
	for !b.isEndOfBacktickContent(nextToken.Kind()) {
		contentItem := b.parseTemplateItem()
		items = append(items, contentItem)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(items...)
}

func (b *BallerinaParser) isEndOfBacktickContent(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN, common.BACKTICK_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseTemplateItem() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.INTERPOLATION_START_TOKEN {
		return b.parseInterpolation()
	}
	if nextToken.Kind() != common.TEMPLATE_STRING {
		nextToken = b.consume()
		return tree.CreateLiteralValueTokenWithDiagnostics(common.TEMPLATE_STRING,
			nextToken.Text(), nextToken.LeadingMinutiae(), nextToken.TrailingMinutiae(),
			nextToken.Diagnostics())
	}
	return b.consume()
}

func (b *BallerinaParser) parseStringTemplateExpression() tree.STNode {
	ty := b.parseStringKeyword()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	content := b.parseTemplateContent()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return tree.CreateTemplateExpressionNode(common.STRING_TEMPLATE_EXPRESSION, ty, startingBackTick,
		content, endingBackTick)
}

func (b *BallerinaParser) parseStringKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.STRING_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STRING_KEYWORD)
		return b.parseStringKeyword()
	}
}

func (b *BallerinaParser) parseXMLTemplateExpression() tree.STNode {
	xmlKeyword := b.parseXMLKeyword()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	if startingBackTick.IsMissing() {
		return b.createMissingTemplateExpressionNode(xmlKeyword, common.XML_TEMPLATE_EXPRESSION)
	}
	content := b.parseTemplateContentAsXML()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return tree.CreateTemplateExpressionNode(common.XML_TEMPLATE_EXPRESSION, xmlKeyword,
		startingBackTick, content, endingBackTick)
}

func (b *BallerinaParser) parseXMLKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.XML_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XML_KEYWORD)
		return b.parseXMLKeyword()
	}
}

func (b *BallerinaParser) parseTemplateContentAsXML() tree.STNode {
	var expressions []tree.STNode
	var xmlStringBuilder strings.Builder
	nextToken := b.peek()
	for !b.isEndOfBacktickContent(nextToken.Kind()) {
		contentItem := b.parseTemplateItem()
		if contentItem.Kind() == common.TEMPLATE_STRING {
			contentToken, ok := contentItem.(tree.STToken)
			if !ok {
				panic("parseTemplateContentAsXML: expected STToken")
			}
			xmlStringBuilder.WriteString(contentToken.Text())
		} else {
			xmlStringBuilder.WriteString("${}")
			expressions = append(expressions, contentItem) //nolint:staticcheck // TODO
		}
		nextToken = b.peek()
	}
	charReader := text.CharReaderFromText(xmlStringBuilder.String())
	xl := newXMLLexer(charReader)
	tr := CreateTokenReader(xl)
	xp := newXMLParser(tr, expressions)
	return xp.Parse()
}

func (b *BallerinaParser) parseRegExpTemplateExpression() tree.STNode {
	reKeyword := b.consume()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	if startingBackTick.IsMissing() {
		return b.createMissingTemplateExpressionNode(reKeyword, common.REGEX_TEMPLATE_EXPRESSION)
	}
	content := b.parseTemplateContentAsRegExp()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return tree.CreateTemplateExpressionNode(common.REGEX_TEMPLATE_EXPRESSION, reKeyword,
		startingBackTick, content, endingBackTick)
}

func (b *BallerinaParser) createMissingTemplateExpressionNode(reKeyword tree.STNode, kind common.SyntaxKind) tree.STNode {
	startingBackTick := tree.CreateMissingToken(common.BACKTICK_TOKEN, nil)
	endingBackTick := tree.CreateMissingToken(common.BACKTICK_TOKEN, nil)
	content := tree.CreateEmptyNodeList()
	templateExpr := tree.CreateTemplateExpressionNode(kind, reKeyword, startingBackTick, content, endingBackTick)
	templateExpr = tree.AddDiagnostic(templateExpr, &common.ERROR_MISSING_BACKTICK_STRING)
	return templateExpr
}

func (b *BallerinaParser) parseTemplateContentAsRegExp() tree.STNode {
	b.tokenReader.StartMode(PARSER_MODE_REGEXP)
	panic("Regexp parser not implemented")
	// expressions := make([]interface{}, 0)
	// regExpStringBuilder := nil
	// nextToken := this.peek()
	// for !this.isEndOfBacktickContent(nextToken.Kind()) {
	// 	contentItem := this.parseTemplateItem()
	// 	if contentItem.Kind() == common.TEMPLATE_STRING {
	// 		contentToken, ok := contentItem.(STToken)
	// 		if !ok {
	// 			panic("parseTemplateContentAsRegExp: expected STToken")
	// 		}
	// 		this.regExpStringBuilder.append(contentToken.text())
	// 	} else {
	// 		this.regExpStringBuilder.append("${}")
	// 		this.expressions.add(contentItem)
	// 	}
	// 	nextToken = this.peek()
	// }
	// this.this.tokenReader.endMode()
	// charReader := this.CharReader.from(regExpStringBuilder.toString())
	// tokenReader := nil
	// regExpParser := nil
	// return this.regExpParser.parse()
}

func (b *BallerinaParser) parseInterpolation() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_INTERPOLATION)
	interpolStart := b.parseInterpolationStart()
	expr := b.parseExpression()
	for !b.isEndOfInterpolation() {
		nextToken := b.consume()
		expr = tree.CloneWithTrailingInvalidNodeMinutiae(expr, nextToken,
			&common.ERROR_INVALID_TOKEN, nextToken.Text())
	}
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return tree.CreateInterpolationNode(interpolStart, expr, closeBrace)
}

func (b *BallerinaParser) isEndOfInterpolation() bool {
	nextTokenKind := b.peek().Kind()
	switch nextTokenKind {
	case common.EOF_TOKEN, common.BACKTICK_TOKEN:
		return true
	default:
		currentLexerMode := b.tokenReader.GetCurrentMode()
		return (((nextTokenKind == common.CLOSE_BRACE_TOKEN) && (currentLexerMode != PARSER_MODE_INTERPOLATION)) && (currentLexerMode != PARSER_MODE_INTERPOLATION_BRACED_CONTENT))
	}
}

func (b *BallerinaParser) parseInterpolationStart() tree.STNode {
	token := b.peek()
	if token.Kind() == common.INTERPOLATION_START_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN)
		return b.parseInterpolationStart()
	}
}

func (b *BallerinaParser) parseBacktickToken(ctx common.ParserRuleContext) tree.STNode {
	token := b.peek()
	if token.Kind() == common.BACKTICK_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, ctx)
		return b.parseBacktickToken(ctx)
	}
}

func (b *BallerinaParser) parseTableTypeDescriptor(tableKeywordToken tree.STNode) tree.STNode {
	rowTypeParameterNode := b.parseRowTypeParameter()
	var keyConstraintNode tree.STNode
	nextToken := b.peek()
	if isKeyKeyword(nextToken) {
		keyKeywordToken := b.getKeyKeyword(b.consume())
		keyConstraintNode = b.parseKeyConstraint(keyKeywordToken)
	} else {
		keyConstraintNode = tree.CreateEmptyNode()
	}
	return tree.CreateTableTypeDescriptorNode(tableKeywordToken, rowTypeParameterNode, keyConstraintNode)
}

func (b *BallerinaParser) parseRowTypeParameter() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM)
	rowTypeParameterNode := b.parseTypeParameter()
	b.endContext()
	return rowTypeParameterNode
}

func (b *BallerinaParser) parseTypeParameter() tree.STNode {
	ltToken := b.parseLTToken()
	typeNode := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS)
	gtToken := b.parseGTToken()
	return tree.CreateTypeParameterNode(ltToken, typeNode, gtToken)
}

func (b *BallerinaParser) parseKeyConstraint(keyKeywordToken tree.STNode) tree.STNode {
	switch b.peek().Kind() {
	case common.OPEN_PAREN_TOKEN:
		return b.parseKeySpecifierWithKeyKeywordToken(keyKeywordToken)
	case common.LT_TOKEN:
		return b.parseKeyTypeConstraint(keyKeywordToken)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS)
		return b.parseKeyConstraint(keyKeywordToken)
	}
}

func (b *BallerinaParser) parseKeySpecifierWithKeyKeywordToken(keyKeywordToken tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_KEY_SPECIFIER)
	openParenToken := b.parseOpenParenthesis()
	fieldNamesNode := b.parseFieldNames()
	closeParenToken := b.parseCloseParenthesis()
	b.endContext()
	return tree.CreateKeySpecifierNode(keyKeywordToken, openParenToken, fieldNamesNode, closeParenToken)
}

func (b *BallerinaParser) parseKeyTypeConstraint(keyKeywordToken tree.STNode) tree.STNode {
	typeParameterNode := b.parseTypeParameter()
	return tree.CreateKeyTypeConstraintNode(keyKeywordToken, typeParameterNode)
}

func (b *BallerinaParser) parseFunctionTypeDesc(qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC)
	functionKeyword := b.parseFunctionKeyword()
	hasFuncSignature := false
	signature := tree.CreateEmptyNode()
	if (b.peek().Kind() == common.OPEN_PAREN_TOKEN) || b.isSyntaxKindInList(qualifiers, common.TRANSACTIONAL_KEYWORD) {
		signature = b.parseFuncSignature(true)
		hasFuncSignature = true
	}
	nodes := b.createFuncTypeQualNodeList(qualifiers, functionKeyword, hasFuncSignature)
	qualifierList := nodes[0]
	functionKeyword = nodes[1]
	b.endContext()
	return tree.CreateFunctionTypeDescriptorNode(qualifierList, functionKeyword, signature)
}

func (b *BallerinaParser) getLastNodeInList(nodeList []tree.STNode) tree.STNode {
	return nodeList[len(nodeList)-1]
}

func (b *BallerinaParser) createFuncTypeQualNodeList(qualifierList []tree.STNode, functionKeyword tree.STNode, hasFuncSignature bool) []tree.STNode {
	var validatedList []tree.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			qualifierToken, ok := qualifier.(tree.STToken)
			if !ok {
				panic("createFuncTypeQualNodeList: expected STToken")
			}
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, qualifierToken.Text())
		} else if hasFuncSignature && b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
		} else if qualifier.Kind() == common.ISOLATED_KEYWORD {
			validatedList = append(validatedList, qualifier)
		} else if len(qualifierList) == nextIndex {
			functionKeyword = tree.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, tree.ToToken(qualifier).Text())
		}
	}
	nodeList := tree.CreateNodeList(validatedList...)
	return []tree.STNode{nodeList, functionKeyword}
}

func (b *BallerinaParser) isRegularFuncQual(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.ISOLATED_KEYWORD, common.TRANSACTIONAL_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseExplicitFunctionExpression(annots tree.STNode, qualifiers []tree.STNode, isRhsExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION)
	funcKeyword := b.parseFunctionKeyword()
	nodes := b.createFuncTypeQualNodeList(qualifiers, funcKeyword, true)
	qualifierList := nodes[0]
	funcKeyword = nodes[1]
	funcSignature := b.parseFuncSignature(false)
	funcBody := b.parseAnonFuncBody(isRhsExpr)
	return tree.CreateExplicitAnonymousFunctionExpressionNode(annots, qualifierList, funcKeyword,
		funcSignature, funcBody)
}

func (b *BallerinaParser) parseAnonFuncBody(isRhsExpr bool) tree.STNode {
	switch b.peek().Kind() {
	case common.OPEN_BRACE_TOKEN,
		common.EOF_TOKEN:
		body := b.parseFunctionBodyBlock(true)
		b.endContext()
		return body
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		b.endContext()
		return b.parseExpressionFuncBody(true, isRhsExpr)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY)
		return b.parseAnonFuncBody(isRhsExpr)
	}
}

func (b *BallerinaParser) parseExpressionFuncBody(isAnon bool, isRhsExpr bool) tree.STNode {
	rightDoubleArrow := b.parseDoubleRightArrow()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION, isRhsExpr, false)
	var semiColon tree.STNode
	if isAnon {
		semiColon = tree.CreateEmptyNode()
	} else {
		semiColon = b.parseSemicolon()
	}
	return tree.CreateExpressionFunctionBodyNode(rightDoubleArrow, expression, semiColon)
}

func (b *BallerinaParser) parseDoubleRightArrow() tree.STNode {
	token := b.peek()
	if token.Kind() == common.RIGHT_DOUBLE_ARROW_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START)
		return b.parseDoubleRightArrow()
	}
}

func (b *BallerinaParser) parseImplicitAnonFuncWithParams(params tree.STNode, isRhsExpr bool) tree.STNode {
	switch params.Kind() {
	case common.SIMPLE_NAME_REFERENCE, common.INFER_PARAM_LIST:
		break
	case common.BRACED_EXPRESSION:
		bracedExpr, ok := params.(*tree.STBracedExpressionNode)
		if !ok {
			panic("parseImplicitAnonFunc: expected STBracedExpressionNode")
		}
		params = b.getAnonFuncParam(*bracedExpr)
	case common.NIL_LITERAL:
		nilLiteralNode, ok := params.(*tree.STNilLiteralNode)
		if !ok {
			panic("expected STNilLiteralNode")
		}
		params = tree.CreateImplicitAnonymousFunctionParameters(nilLiteralNode.OpenParenToken,
			tree.CreateNodeList(), nilLiteralNode.CloseParenToken)
	default:
		var syntheticParam tree.STNode
		syntheticParam = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		syntheticParam = tree.CloneWithLeadingInvalidNodeMinutiae(syntheticParam, params,
			&common.ERROR_INVALID_PARAM_LIST_IN_INFER_ANONYMOUS_FUNCTION_EXPR)
		params = tree.CreateSimpleNameReferenceNode(syntheticParam)
	}
	rightDoubleArrow := b.parseDoubleRightArrow()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_REMOTE_CALL_ACTION, isRhsExpr, false)
	return tree.CreateImplicitAnonymousFunctionExpressionNode(params, rightDoubleArrow, expression)
}

func (b *BallerinaParser) getAnonFuncParam(bracedExpression tree.STBracedExpressionNode) tree.STNode {
	var paramList []tree.STNode
	innerExpression := bracedExpression.Expression
	openParen := bracedExpression.OpenParen
	if innerExpression.Kind() == common.SIMPLE_NAME_REFERENCE {
		paramList = append(paramList, innerExpression)
	} else {
		openParen = tree.CloneWithTrailingInvalidNodeMinutiae(openParen, innerExpression,
			&common.ERROR_INVALID_PARAM_LIST_IN_INFER_ANONYMOUS_FUNCTION_EXPR)
	}
	return tree.CreateImplicitAnonymousFunctionParameters(openParen,
		tree.CreateNodeList(paramList...), bracedExpression.CloseParen)
}

func (b *BallerinaParser) parseImplicitAnonFuncWithOpenParenAndFirstParam(openParen tree.STNode, firstParam tree.STNode, isRhsExpr bool) tree.STNode {
	var paramList []tree.STNode
	paramList = append(paramList, firstParam)
	nextToken := b.peek()
	var paramEnd tree.STNode
	var param tree.STNode
	for !b.isEndOfAnonFuncParametersList(nextToken.Kind()) {
		paramEnd = b.parseImplicitAnonFuncParamEnd()
		if paramEnd == nil {
			break
		}
		paramList = append(paramList, paramEnd)
		param = b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM)
		param = tree.CreateSimpleNameReferenceNode(param)
		paramList = append(paramList, param)
		nextToken = b.peek()
	}
	params := tree.CreateNodeList(paramList...)
	closeParen := b.parseCloseParenthesis()
	b.endContext()
	inferedParams := tree.CreateImplicitAnonymousFunctionParameters(openParen, params, closeParen)
	return b.parseImplicitAnonFuncWithParams(inferedParams, isRhsExpr)
}

func (b *BallerinaParser) parseImplicitAnonFuncParamEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS)
		return b.parseImplicitAnonFuncParamEnd()
	}
}

func (b *BallerinaParser) isEndOfAnonFuncParametersList(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.EOF_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.CLOSE_BRACKET_TOKEN,
		common.SEMICOLON_TOKEN,
		common.RETURNS_KEYWORD,
		common.TYPE_KEYWORD,
		common.LISTENER_KEYWORD,
		common.IF_KEYWORD,
		common.WHILE_KEYWORD,
		common.DO_KEYWORD,
		common.OPEN_BRACE_TOKEN,
		common.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseTupleTypeDesc() tree.STNode {
	openBracket := b.parseOpenBracket()
	b.startContext(common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS)
	memberTypeDesc := b.parseTupleMemberTypeDescList()
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return tree.CreateTupleTypeDescriptorNode(openBracket, memberTypeDesc, closeBracket)
}

func (b *BallerinaParser) parseTupleMemberTypeDescList() tree.STNode {
	var typeDescList []tree.STNode
	nextToken := b.peek()
	if b.isEndOfTypeList(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	typeDesc := b.parseTupleMember()
	res, _ := b.parseTupleTypeMembers(typeDesc, typeDescList)
	return res
}

func (b *BallerinaParser) parseTupleTypeMembers(firstMember tree.STNode, memberList []tree.STNode) (tree.STNode, []tree.STNode) {
	var tupleMemberRhs tree.STNode
	for !b.isEndOfTypeList(b.peek().Kind()) {
		if firstMember.Kind() == common.REST_TYPE {
			firstMember = b.invalidateTypeDescAfterRestDesc(firstMember)
			break
		}
		tupleMemberRhs = b.parseTupleMemberRhs()
		if tupleMemberRhs == nil {
			break
		}
		memberList = append(memberList, firstMember)
		memberList = append(memberList, tupleMemberRhs)
		firstMember = b.parseTupleMember()
	}
	memberList = append(memberList, firstMember)
	return tree.CreateNodeList(memberList...), memberList
}

func (b *BallerinaParser) parseTupleMember() tree.STNode {
	annot := b.parseOptionalAnnotations()
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	return b.createMemberOrRestNode(annot, typeDesc)
}

func (b *BallerinaParser) createMemberOrRestNode(annot tree.STNode, typeDesc tree.STNode) tree.STNode {
	tupleMemberRhs := b.parseTypeDescInTupleRhs()
	if tupleMemberRhs != nil {
		annotList, ok := annot.(*tree.STNodeList)
		if !ok {
			panic("createMemberOrRestNode: expected tree.STNodeList")
		}
		if !annotList.IsEmpty() {
			typeDesc = tree.CloneWithLeadingInvalidNodeMinutiae(typeDesc, annot,
				&common.ERROR_ANNOTATIONS_NOT_ALLOWED_FOR_TUPLE_REST_DESCRIPTOR)
		}
		return tree.CreateRestDescriptorNode(typeDesc, tupleMemberRhs)
	}
	return tree.CreateMemberTypeDescriptorNode(annot, typeDesc)
}

func (b *BallerinaParser) invalidateTypeDescAfterRestDesc(restDescriptor tree.STNode) tree.STNode {
	for !b.isEndOfTypeList(b.peek().Kind()) {
		tupleMemberRhs := b.parseTupleMemberRhs()
		if tupleMemberRhs == nil {
			break
		}
		restDescriptor = tree.CloneWithTrailingInvalidNodeMinutiae(restDescriptor, tupleMemberRhs, nil)
		restDescriptor = tree.CloneWithTrailingInvalidNodeMinutiae(restDescriptor, b.parseTupleMember(),
			&common.ERROR_TYPE_DESC_AFTER_REST_DESCRIPTOR)
	}
	return restDescriptor
}

func (b *BallerinaParser) parseTupleMemberRhs() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS)
		return b.parseTupleMemberRhs()
	}
}

func (b *BallerinaParser) parseTypeDescInTupleRhs() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN, common.CLOSE_BRACKET_TOKEN:
		return nil
	case common.ELLIPSIS_TOKEN:
		return b.parseEllipsis()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS)
		return b.parseTypeDescInTupleRhs()
	}
}

func (b *BallerinaParser) isEndOfTypeList(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.CLOSE_BRACKET_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.EOF_TOKEN,
		common.EQUAL_TOKEN,
		common.SEMICOLON_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseTableConstructorOrQuery(isRhsExpr bool, allowActions bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION)
	tableOrQueryExpr := b.parseTableConstructorOrQueryInner(isRhsExpr, allowActions)
	b.endContext()
	return tableOrQueryExpr
}

func (b *BallerinaParser) parseTableConstructorOrQueryInner(isRhsExpr bool, allowActions bool) tree.STNode {
	var queryConstructType tree.STNode
	switch b.peek().Kind() {
	case common.FROM_KEYWORD:
		queryConstructType = tree.CreateEmptyNode()
		return b.parseQueryExprRhs(queryConstructType, isRhsExpr, allowActions)
	case common.TABLE_KEYWORD:
		tableKeyword := b.parseTableKeyword()
		return b.parseTableConstructorOrQueryWithKeyword(tableKeyword, isRhsExpr, allowActions)
	case common.STREAM_KEYWORD,
		common.MAP_KEYWORD:
		streamOrMapKeyword := b.consume()
		keySpecifier := tree.CreateEmptyNode()
		queryConstructType = b.parseQueryConstructType(streamOrMapKeyword, keySpecifier)
		return b.parseQueryExprRhs(queryConstructType, isRhsExpr, allowActions)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START)
		return b.parseTableConstructorOrQueryInner(isRhsExpr, allowActions)
	}
}

func (b *BallerinaParser) parseTableConstructorOrQueryWithKeyword(tableKeyword tree.STNode, isRhsExpr bool, allowActions bool) tree.STNode {
	var keySpecifier tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACKET_TOKEN:
		keySpecifier = tree.CreateEmptyNode()
		return b.parseTableConstructorExprRhs(tableKeyword, keySpecifier)
	case common.KEY_KEYWORD:
		keySpecifier = b.parseKeySpecifier()
		return b.parseTableConstructorOrQueryRhs(tableKeyword, keySpecifier, isRhsExpr, allowActions)
	case common.IDENTIFIER_TOKEN:
		if isKeyKeyword(nextToken) {
			keySpecifier = b.parseKeySpecifier()
			return b.parseTableConstructorOrQueryRhs(tableKeyword, keySpecifier, isRhsExpr, allowActions)
		}
	default:
		break
	}
	b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS)
	return b.parseTableConstructorOrQueryWithKeyword(tableKeyword, isRhsExpr, allowActions)
}

func (b *BallerinaParser) parseTableConstructorOrQueryRhs(tableKeyword tree.STNode, keySpecifier tree.STNode, isRhsExpr bool, allowActions bool) tree.STNode {
	switch b.peek().Kind() {
	case common.FROM_KEYWORD:
		return b.parseQueryExprRhs(b.parseQueryConstructType(tableKeyword, keySpecifier), isRhsExpr, allowActions)
	case common.OPEN_BRACKET_TOKEN:
		return b.parseTableConstructorExprRhs(tableKeyword, keySpecifier)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS)
		return b.parseTableConstructorOrQueryRhs(tableKeyword, keySpecifier, isRhsExpr, allowActions)
	}
}

func (b *BallerinaParser) parseQueryConstructType(keyword tree.STNode, keySpecifier tree.STNode) tree.STNode {
	return tree.CreateQueryConstructTypeNode(keyword, keySpecifier)
}

func (b *BallerinaParser) parseQueryExprRhs(queryConstructType tree.STNode, isRhsExpr bool, allowActions bool) tree.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
	fromClause := b.parseFromClause(isRhsExpr, allowActions)
	var clauses []tree.STNode
	var intermediateClause tree.STNode
	var selectClause tree.STNode
	var collectClause tree.STNode
	for !b.isEndOfIntermediateClause(b.peek().Kind()) {
		intermediateClause = b.parseIntermediateClause(isRhsExpr, allowActions)
		if intermediateClause == nil {
			break
		}

		// If there are more clauses after select clause they are add as invalid nodes to the select clause
		if selectClause != nil {
			selectClause = tree.CloneWithTrailingInvalidNodeMinutiae(selectClause, intermediateClause,
				&common.ERROR_MORE_CLAUSES_AFTER_SELECT_CLAUSE)
			continue
		} else if collectClause != nil {
			collectClause = tree.CloneWithTrailingInvalidNodeMinutiae(collectClause, intermediateClause,
				&common.ERROR_MORE_CLAUSES_AFTER_COLLECT_CLAUSE)
			continue
		}
		if intermediateClause.Kind() == common.SELECT_CLAUSE {
			selectClause = intermediateClause
		} else if intermediateClause.Kind() == common.COLLECT_CLAUSE {
			collectClause = intermediateClause
		} else {
			clauses = append(clauses, intermediateClause)
			continue
		}
		if b.isNestedQueryExpr() || (!b.isValidIntermediateQueryStart(b.peek())) {
			// Break the loop for,
			// 1. nested query expressions as remaining clauses belong to the parent.
			// 2. next token not being an intermediate-clause start as that token could belong to the parent node.
			break
		}
	}
	if (b.peek().Kind() == common.DO_KEYWORD) && ((!b.isNestedQueryExpr()) || ((selectClause == nil) && (collectClause == nil))) {
		intermediateClauses := tree.CreateNodeList(clauses...)
		queryPipeline := tree.CreateQueryPipelineNode(fromClause, intermediateClauses)
		return b.parseQueryAction(queryConstructType, queryPipeline, selectClause, collectClause)
	}
	if (selectClause == nil) && (collectClause == nil) {
		selectKeyword := tree.CreateMissingToken(common.SELECT_KEYWORD, nil)
		expr := tree.CreateSimpleNameReferenceNode(tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil))
		selectClause = tree.CreateSelectClauseNode(selectKeyword, expr)

		// Now we need to attach the diagnostic to the last intermediate clause.
		// If there are no intermediate clauses, then attach to the from clause.
		if len(clauses) == 0 {
			fromClause = tree.AddDiagnostic(fromClause, &common.ERROR_MISSING_SELECT_CLAUSE)
		} else {
			lastIndex := (len(clauses) - 1)
			intClauseWithDiagnostic := tree.AddDiagnostic(clauses[lastIndex],
				&common.ERROR_MISSING_SELECT_CLAUSE)
			clauses[lastIndex] = intClauseWithDiagnostic
		}
	}
	intermediateClauses := tree.CreateNodeList(clauses...)
	queryPipeline := tree.CreateQueryPipelineNode(fromClause, intermediateClauses)
	onConflictClause := b.parseOnConflictClause(isRhsExpr)
	var clause tree.STNode
	if selectClause == nil {
		clause = collectClause
	} else {
		clause = selectClause
	}
	return tree.CreateQueryExpressionNode(queryConstructType, queryPipeline,
		clause, onConflictClause)
}

func (b *BallerinaParser) isNestedQueryExpr() bool {
	contextStack := b.errorHandler.GetContextStack()
	count := 0
	for _, ctx := range contextStack {
		if ctx == common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION {
			count++
		}
		if count > 1 {
			return true
		}
	}
	return false
}

func (b *BallerinaParser) isValidIntermediateQueryStart(token tree.STToken) bool {
	switch token.Kind() {
	case common.FROM_KEYWORD,
		common.WHERE_KEYWORD,
		common.LET_KEYWORD,
		common.SELECT_KEYWORD,
		common.JOIN_KEYWORD,
		common.OUTER_KEYWORD,
		common.ORDER_KEYWORD,
		common.BY_KEYWORD,
		common.ASCENDING_KEYWORD,
		common.DESCENDING_KEYWORD,
		common.LIMIT_KEYWORD:
		return true
	case common.IDENTIFIER_TOKEN:
		return isGroupOrCollectKeyword(token)
	default:
		return false
	}
}

func (b *BallerinaParser) parseIntermediateClause(isRhsExpr bool, allowActions bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.FROM_KEYWORD:
		return b.parseFromClause(isRhsExpr, allowActions)
	case common.WHERE_KEYWORD:
		return b.parseWhereClause(isRhsExpr)
	case common.LET_KEYWORD:
		return b.parseLetClause(isRhsExpr, allowActions)
	case common.SELECT_KEYWORD:
		return b.parseSelectClause(isRhsExpr, allowActions)
	case common.JOIN_KEYWORD, common.OUTER_KEYWORD:
		return b.parseJoinClause(isRhsExpr)
	case common.ORDER_KEYWORD,
		common.ASCENDING_KEYWORD,
		common.DESCENDING_KEYWORD:
		return b.parseOrderByClause(isRhsExpr)
	case common.LIMIT_KEYWORD:
		return b.parseLimitClause(isRhsExpr)
	case common.DO_KEYWORD,
		common.SEMICOLON_TOKEN,
		common.ON_KEYWORD,
		common.CONFLICT_KEYWORD:
		return nil
	default:
		if isKeywordMatch(common.COLLECT_KEYWORD, nextToken) {
			return b.parseCollectClause(isRhsExpr)
		}
		if isKeywordMatch(common.GROUP_KEYWORD, nextToken) {
			return b.parseGroupByClause(isRhsExpr)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS)
		return b.parseIntermediateClause(isRhsExpr, allowActions)
	}
}

func (b *BallerinaParser) parseCollectClause(isRhsExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE)
	collectKeyword := b.parseCollectKeyword()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	b.endContext()
	return tree.CreateCollectClauseNode(collectKeyword, expression)
}

func (b *BallerinaParser) parseCollectKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.COLLECT_KEYWORD {
		return b.consume()
	}
	if isKeywordMatch(common.COLLECT_KEYWORD, token) {
		return b.getCollectKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD)
	return b.parseCollectKeyword()
}

func (b *BallerinaParser) getCollectKeyword(token tree.STToken) tree.STNode {
	return tree.CreateTokenWithDiagnostics(common.COLLECT_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *BallerinaParser) parseJoinKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.JOIN_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_JOIN_KEYWORD)
		return b.parseJoinKeyword()
	}
}

func (b *BallerinaParser) parseEqualsKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.EQUALS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD)
		return b.parseEqualsKeyword()
	}
}

func (b *BallerinaParser) isEndOfIntermediateClause(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.CLOSE_BRACE_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.CLOSE_BRACKET_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.SEMICOLON_TOKEN,
		common.PUBLIC_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.EOF_TOKEN,
		common.RESOURCE_KEYWORD,
		common.LISTENER_KEYWORD,
		common.DOCUMENTATION_STRING,
		common.PRIVATE_KEYWORD,
		common.RETURNS_KEYWORD,
		common.SERVICE_KEYWORD,
		common.TYPE_KEYWORD,
		common.CONST_KEYWORD,
		common.FINAL_KEYWORD,
		common.DO_KEYWORD,
		common.ON_KEYWORD,
		common.CONFLICT_KEYWORD:
		return true
	default:
		return b.isValidExprRhsStart(tokenKind, common.NONE)
	}
}

func (b *BallerinaParser) parseFromClause(isRhsExpr bool, allowActions bool) tree.STNode {
	fromKeyword := b.parseFromKeyword()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_FROM_CLAUSE)
	inKeyword := b.parseInKeyword()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, allowActions)
	return tree.CreateFromClauseNode(fromKeyword, typedBindingPattern, inKeyword, expression)
}

func (b *BallerinaParser) parseFromKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FROM_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FROM_KEYWORD)
		return b.parseFromKeyword()
	}
}

func (b *BallerinaParser) parseWhereClause(isRhsExpr bool) tree.STNode {
	whereKeyword := b.parseWhereKeyword()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	return tree.CreateWhereClauseNode(whereKeyword, expression)
}

func (b *BallerinaParser) parseWhereKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.WHERE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_WHERE_KEYWORD)
		return b.parseWhereKeyword()
	}
}

func (b *BallerinaParser) parseLimitKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.LIMIT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD)
		return b.parseLimitKeyword()
	}
}

func (b *BallerinaParser) parseLetClause(isRhsExpr bool, allowActions bool) tree.STNode {
	letKeyword := b.parseLetKeyword()
	letVarDeclarations := b.parseLetVarDeclarations(common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL, isRhsExpr,
		allowActions)
	letKeyword = b.cloneWithDiagnosticIfListEmpty(letVarDeclarations, letKeyword,
		&common.ERROR_MISSING_LET_VARIABLE_DECLARATION)
	return tree.CreateLetClauseNode(letKeyword, letVarDeclarations)
}

func (b *BallerinaParser) parseGroupByClause(isRhsExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE)
	groupKeyword := b.parseGroupKeyword()
	byKeyword := b.parseByKeyword()
	groupingKeys := b.parseGroupingKeyList(isRhsExpr)
	byKeyword = b.cloneWithDiagnosticIfListEmpty(groupingKeys, byKeyword,
		&common.ERROR_MISSING_GROUPING_KEY)
	b.endContext()
	return tree.CreateGroupByClauseNode(groupKeyword, byKeyword, groupingKeys)
}

func (b *BallerinaParser) parseGroupKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.GROUP_KEYWORD {
		return b.consume()
	}
	if isKeywordMatch(common.GROUP_KEYWORD, token) {
		return b.getGroupKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_GROUP_KEYWORD)
	return b.parseGroupKeyword()
}

func (b *BallerinaParser) getGroupKeyword(token tree.STToken) tree.STNode {
	return tree.CreateTokenWithDiagnostics(common.GROUP_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *BallerinaParser) parseOrderKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ORDER_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ORDER_KEYWORD)
		return b.parseOrderKeyword()
	}
}

func (b *BallerinaParser) parseByKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.BY_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BY_KEYWORD)
		return b.parseByKeyword()
	}
}

func (b *BallerinaParser) parseOrderByClause(isRhsExpr bool) tree.STNode {
	orderKeyword := b.parseOrderKeyword()
	byKeyword := b.parseByKeyword()
	orderKeys := b.parseOrderKeyList(isRhsExpr)
	byKeyword = b.cloneWithDiagnosticIfListEmpty(orderKeys, byKeyword, &common.ERROR_MISSING_ORDER_KEY)
	return tree.CreateOrderByClauseNode(orderKeyword, byKeyword, orderKeys)
}

func (b *BallerinaParser) parseGroupingKeyList(isRhsExpr bool) tree.STNode {
	var groupingKeys []tree.STNode
	nextToken := b.peek()
	if b.isEndOfGroupByKeyListElement(nextToken) {
		return tree.CreateEmptyNodeList()
	}
	groupingKey := b.parseGroupingKey(isRhsExpr)
	groupingKeys = append(groupingKeys, groupingKey)
	nextToken = b.peek()
	var groupingKeyListMemberEnd tree.STNode
	for !b.isEndOfGroupByKeyListElement(nextToken) {
		groupingKeyListMemberEnd = b.parseGroupingKeyListMemberEnd()
		if groupingKeyListMemberEnd == nil {
			break
		}
		groupingKeys = append(groupingKeys, groupingKeyListMemberEnd)
		groupingKey = b.parseGroupingKey(isRhsExpr)
		groupingKeys = append(groupingKeys, groupingKey)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(groupingKeys...)
}

func (b *BallerinaParser) parseOrderKeyList(isRhsExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST)
	var orderKeys []tree.STNode
	nextToken := b.peek()
	if b.isEndOfOrderKeys(nextToken) {
		b.endContext()
		return tree.CreateEmptyNodeList()
	}
	orderKey := b.parseOrderKey(isRhsExpr)
	orderKeys = append(orderKeys, orderKey)
	nextToken = b.peek()
	var orderKeyListMemberEnd tree.STNode
	for !b.isEndOfOrderKeys(nextToken) {
		orderKeyListMemberEnd = b.parseOrderKeyListMemberEnd()
		if orderKeyListMemberEnd == nil {
			break
		}
		orderKeys = append(orderKeys, orderKeyListMemberEnd)
		orderKey = b.parseOrderKey(isRhsExpr)
		orderKeys = append(orderKeys, orderKey)
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateNodeList(orderKeys...)
}

func (b *BallerinaParser) isEndOfGroupByKeyListElement(nextToken tree.STToken) bool {
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return false
	case common.EOF_TOKEN:
		return true
	default:
		return b.isQueryClauseStartToken(nextToken)
	}
}

func (b *BallerinaParser) isEndOfOrderKeys(nextToken tree.STToken) bool {
	switch nextToken.Kind() {
	case common.COMMA_TOKEN,
		common.ASCENDING_KEYWORD,
		common.DESCENDING_KEYWORD:
		return false
	case common.SEMICOLON_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return b.isQueryClauseStartToken(nextToken)
	}
}

func (b *BallerinaParser) isQueryClauseStartToken(nextToken tree.STToken) bool {
	switch nextToken.Kind() {
	case common.SELECT_KEYWORD,
		common.LET_KEYWORD,
		common.WHERE_KEYWORD,
		common.OUTER_KEYWORD,
		common.JOIN_KEYWORD,
		common.ORDER_KEYWORD,
		common.DO_KEYWORD,
		common.FROM_KEYWORD,
		common.LIMIT_KEYWORD:
		return true
	case common.IDENTIFIER_TOKEN:
		return isGroupOrCollectKeyword(nextToken)
	default:
		return false
	}
}

func (b *BallerinaParser) parseGroupingKeyListMemberEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return b.consume()
	case common.EOF_TOKEN:
		return nil
	default:
		if b.isQueryClauseStartToken(nextToken) {
			return nil
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END)
		return b.parseGroupingKeyListMemberEnd()
	}
}

func (b *BallerinaParser) parseOrderKeyListMemberEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.EOF_TOKEN:
		return nil
	default:
		if b.isQueryClauseStartToken(nextToken) {
			return nil
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END)
		return b.parseOrderKeyListMemberEnd()
	}
}

func (b *BallerinaParser) parseGroupingKeyVariableDeclaration(isRhsExpr bool) tree.STNode {
	groupingKeyElementTypeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY)
	b.startContext(common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER)
	groupingKeySimpleBP := b.createCaptureOrWildcardBP(b.parseVariableName())
	b.endContext()
	equalsToken := b.parseAssignOp()
	groupingKeyExpression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	return tree.CreateGroupingKeyVarDeclarationNode(groupingKeyElementTypeDesc, groupingKeySimpleBP,
		equalsToken, groupingKeyExpression)
}

func (b *BallerinaParser) parseGroupingKey(isRhsExpr bool) tree.STNode {
	nextToken := b.peek()
	nextTokenKind := nextToken.Kind()
	if (nextTokenKind == common.IDENTIFIER_TOKEN) && (!b.isPossibleGroupingKeyVarDeclaration()) {
		return tree.CreateSimpleNameReferenceNode(b.parseVariableName())
	} else if isTypeStartingToken(nextTokenKind, nextToken) {
		return b.parseGroupingKeyVariableDeclaration(isRhsExpr)
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT)
	return b.parseGroupingKey(isRhsExpr)
}

func (b *BallerinaParser) isPossibleGroupingKeyVarDeclaration() bool {
	nextNextTokenKind := b.getNextNextToken().Kind()
	return ((nextNextTokenKind == common.EQUAL_TOKEN) || ((nextNextTokenKind == common.IDENTIFIER_TOKEN) && (b.peekN(3).Kind() == common.EQUAL_TOKEN)))
}

func (b *BallerinaParser) parseOrderKey(isRhsExpr bool) tree.STNode {
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	var orderDirection tree.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ASCENDING_KEYWORD, common.DESCENDING_KEYWORD:
		orderDirection = b.consume()
	default:
		orderDirection = tree.CreateEmptyNode()
	}
	return tree.CreateOrderKeyNode(expression, orderDirection)
}

func (b *BallerinaParser) parseSelectClause(isRhsExpr bool, allowActions bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_SELECT_CLAUSE)
	selectKeyword := b.parseSelectKeyword()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, allowActions)
	b.endContext()
	return tree.CreateSelectClauseNode(selectKeyword, expression)
}

func (b *BallerinaParser) parseSelectKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.SELECT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SELECT_KEYWORD)
		return b.parseSelectKeyword()
	}
}

func (b *BallerinaParser) parseOnConflictClause(isRhsExpr bool) tree.STNode {
	nextToken := b.peek()
	if (nextToken.Kind() != common.ON_KEYWORD) && (nextToken.Kind() != common.CONFLICT_KEYWORD) {
		return tree.CreateEmptyNode()
	}
	b.startContext(common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE)
	onKeyword := b.parseOnKeyword()
	conflictKeyword := b.parseConflictKeyword()
	b.endContext()
	expr := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	return tree.CreateOnConflictClauseNode(onKeyword, conflictKeyword, expr)
}

func (b *BallerinaParser) parseConflictKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.CONFLICT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD)
		return b.parseConflictKeyword()
	}
}

func (b *BallerinaParser) parseLimitClause(isRhsExpr bool) tree.STNode {
	limitKeyword := b.parseLimitKeyword()
	expr := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	return tree.CreateLimitClauseNode(limitKeyword, expr)
}

func (b *BallerinaParser) parseJoinClause(isRhsExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_JOIN_CLAUSE)
	var outerKeyword tree.STNode
	nextToken := b.peek()
	if nextToken.Kind() == common.OUTER_KEYWORD {
		outerKeyword = b.consume()
	} else {
		outerKeyword = tree.CreateEmptyNode()
	}
	joinKeyword := b.parseJoinKeyword()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_JOIN_CLAUSE)
	inKeyword := b.parseInKeyword()
	expression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	b.endContext()
	onCondition := b.parseOnClause(isRhsExpr)
	return tree.CreateJoinClauseNode(outerKeyword, joinKeyword, typedBindingPattern, inKeyword, expression,
		onCondition)
}

func (b *BallerinaParser) parseOnClause(isRhsExpr bool) tree.STNode {
	nextToken := b.peek()
	if b.isQueryClauseStartToken(nextToken) {
		return b.createMissingOnClauseNode()
	}
	b.startContext(common.PARSER_RULE_CONTEXT_ON_CLAUSE)
	onKeyword := b.parseOnKeyword()
	onExpression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	equalsKeyword := b.parseEqualsKeyword()
	b.endContext()
	equalsExpression := b.parseExpressionWithPrecedence(OPERATOR_PRECEDENCE_QUERY, isRhsExpr, false)
	return tree.CreateOnClauseNode(onKeyword, onExpression, equalsKeyword, equalsExpression)
}

func (b *BallerinaParser) createMissingOnClauseNode() tree.STNode {
	onKeyword := tree.CreateMissingTokenWithDiagnostics(common.ON_KEYWORD,
		&common.ERROR_MISSING_ON_KEYWORD)
	identifier := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_IDENTIFIER)
	equalsKeyword := tree.CreateMissingTokenWithDiagnostics(common.EQUALS_KEYWORD,
		&common.ERROR_MISSING_EQUALS_KEYWORD)
	onExpression := tree.CreateSimpleNameReferenceNode(identifier)
	equalsExpression := tree.CreateSimpleNameReferenceNode(identifier)
	return tree.CreateOnClauseNode(onKeyword, onExpression, equalsKeyword, equalsExpression)
}

func (b *BallerinaParser) parseStartAction(annots tree.STNode) tree.STNode {
	startKeyword := b.parseStartKeyword()
	expr := b.parseActionOrExpression()
	switch expr.Kind() {
	case common.FUNCTION_CALL,
		common.METHOD_CALL,
		common.REMOTE_METHOD_CALL_ACTION:
		break
	case common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE,
		common.FIELD_ACCESS,
		common.ASYNC_SEND_ACTION:
		expr = b.generateValidExprForStartAction(expr)
	default:
		startKeyword = tree.CloneWithTrailingInvalidNodeMinutiae(startKeyword, expr,
			&common.ERROR_INVALID_EXPRESSION_IN_START_ACTION)
		var funcName tree.STNode
		funcName = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		funcName = tree.CreateSimpleNameReferenceNode(funcName)
		openParenToken := tree.CreateMissingToken(common.OPEN_PAREN_TOKEN, nil)
		closeParenToken := tree.CreateMissingToken(common.CLOSE_PAREN_TOKEN, nil)
		expr = tree.CreateFunctionCallExpressionNode(funcName, openParenToken,
			tree.CreateEmptyNodeList(), closeParenToken)
	}
	return tree.CreateStartActionNode(b.getAnnotations(annots), startKeyword, expr)
}

func (b *BallerinaParser) generateValidExprForStartAction(expr tree.STNode) tree.STNode {
	openParenToken := tree.CreateMissingTokenWithDiagnostics(common.OPEN_PAREN_TOKEN,
		&common.ERROR_MISSING_OPEN_PAREN_TOKEN)
	arguments := tree.CreateEmptyNodeList()
	closeParenToken := tree.CreateMissingTokenWithDiagnostics(common.CLOSE_PAREN_TOKEN,
		&common.ERROR_MISSING_CLOSE_PAREN_TOKEN)
	switch expr.Kind() {
	case common.FIELD_ACCESS:
		fieldAccessExpr, ok := expr.(*tree.STFieldAccessExpressionNode)
		if !ok {
			panic("expected STFieldAccessExpressionNode")
		}
		return tree.CreateMethodCallExpressionNode(fieldAccessExpr.Expression,
			fieldAccessExpr.DotToken, fieldAccessExpr.FieldName, openParenToken, arguments,
			closeParenToken)
	case common.ASYNC_SEND_ACTION:
		asyncSendAction, ok := expr.(*tree.STAsyncSendActionNode)
		if !ok {
			panic("expected STAsyncSendActionNode")
		}
		return tree.CreateRemoteMethodCallActionNode(asyncSendAction.Expression,
			asyncSendAction.RightArrowToken, asyncSendAction.PeerWorker, openParenToken, arguments,
			closeParenToken)
	default:
		return tree.CreateFunctionCallExpressionNode(expr, openParenToken, arguments, closeParenToken)
	}
}

func (b *BallerinaParser) parseStartKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.START_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_START_KEYWORD)
		return b.parseStartKeyword()
	}
}

func (b *BallerinaParser) parseFlushAction() tree.STNode {
	flushKeyword := b.parseFlushKeyword()
	peerWorker := b.parseOptionalPeerWorkerName()
	return tree.CreateFlushActionNode(flushKeyword, peerWorker)
}

func (b *BallerinaParser) parseFlushKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.FLUSH_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD)
		return b.parseFlushKeyword()
	}
}

func (b *BallerinaParser) parseOptionalPeerWorkerName() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.IDENTIFIER_TOKEN, common.FUNCTION_KEYWORD:
		return tree.CreateSimpleNameReferenceNode(b.consume())
	default:
		return tree.CreateEmptyNode()
	}
}

func (b *BallerinaParser) parseIntersectionTypeDescriptor(leftTypeDesc tree.STNode, context common.ParserRuleContext, isTypedBindingPattern bool) tree.STNode {
	bitwiseAndToken := b.consume()
	rightTypeDesc := b.parseTypeDescriptorInternalWithPrecedence(nil, context, isTypedBindingPattern, false,
		TYPE_PRECEDENCE_INTERSECTION)
	return b.mergeTypesWithIntersection(leftTypeDesc, bitwiseAndToken, rightTypeDesc)
}

func (b *BallerinaParser) createIntersectionTypeDesc(leftTypeDesc tree.STNode, bitwiseAndToken tree.STNode, rightTypeDesc tree.STNode) tree.STNode {
	leftTypeDesc = b.validateForUsageOfVar(leftTypeDesc)
	rightTypeDesc = b.validateForUsageOfVar(rightTypeDesc)
	return tree.CreateIntersectionTypeDescriptorNode(leftTypeDesc, bitwiseAndToken, rightTypeDesc)
}

func (b *BallerinaParser) parseSingletonTypeDesc() tree.STNode {
	simpleContExpr := b.parseSimpleConstExpr()
	return tree.CreateSingletonTypeDescriptorNode(simpleContExpr)
}

func (b *BallerinaParser) parseSignedIntOrFloat() tree.STNode {
	operator := b.parseUnaryOperator()
	var literal tree.STNode
	nextToken := b.peek()

	switch nextToken.Kind() {

	case common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		literal = b.parseBasicLiteral()
	default:
		literal = tree.CreateBasicLiteralNode(common.NUMERIC_LITERAL,
			b.parseDecimalIntLiteral(common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN))
	}
	return tree.CreateUnaryExpressionNode(operator, literal)
}

func (b *BallerinaParser) isValidExpressionStart(nextTokenKind common.SyntaxKind, nextTokenIndex int) bool {
	nextTokenIndex++
	switch nextTokenKind {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		nextNextTokenKind := b.peekN(nextTokenIndex).Kind()
		if (nextNextTokenKind == common.PIPE_TOKEN) || (nextNextTokenKind == common.BITWISE_AND_TOKEN) {
			nextTokenIndex++
			return b.isValidExpressionStart(b.peekN(nextTokenIndex).Kind(), nextTokenIndex)
		}
		return ((((nextNextTokenKind == common.SEMICOLON_TOKEN) || (nextNextTokenKind == common.COMMA_TOKEN)) || (nextNextTokenKind == common.CLOSE_BRACKET_TOKEN)) || b.isValidExprRhsStart(nextNextTokenKind, common.SIMPLE_NAME_REFERENCE))
	case common.IDENTIFIER_TOKEN:
		return b.isValidExprRhsStart(b.peekN(nextTokenIndex).Kind(), common.SIMPLE_NAME_REFERENCE)
	case common.OPEN_PAREN_TOKEN, common.CHECK_KEYWORD, common.CHECKPANIC_KEYWORD, common.OPEN_BRACE_TOKEN,
		common.TYPEOF_KEYWORD, common.NEGATION_TOKEN, common.EXCLAMATION_MARK_TOKEN, common.TRAP_KEYWORD,
		common.OPEN_BRACKET_TOKEN, common.LT_TOKEN, common.FROM_KEYWORD, common.LET_KEYWORD,
		common.BACKTICK_TOKEN, common.NEW_KEYWORD, common.LEFT_ARROW_TOKEN, common.FUNCTION_KEYWORD,
		common.TRANSACTIONAL_KEYWORD, common.ISOLATED_KEYWORD, common.BASE16_KEYWORD, common.BASE64_KEYWORD,
		common.NATURAL_KEYWORD:
		return true
	case common.PLUS_TOKEN, common.MINUS_TOKEN:
		return b.isValidExpressionStart(b.peekN(nextTokenIndex).Kind(), nextTokenIndex)
	case common.TABLE_KEYWORD, common.MAP_KEYWORD:
		return (b.peekN(nextTokenIndex).Kind() == common.FROM_KEYWORD)
	case common.STREAM_KEYWORD:
		nextNextToken := b.peekN(nextTokenIndex)
		return (((nextNextToken.Kind() == common.KEY_KEYWORD) || (nextNextToken.Kind() == common.OPEN_BRACKET_TOKEN)) || (nextNextToken.Kind() == common.FROM_KEYWORD))
	case common.ERROR_KEYWORD:
		return (b.peekN(nextTokenIndex).Kind() == common.OPEN_PAREN_TOKEN)
	case common.XML_KEYWORD, common.STRING_KEYWORD, common.RE_KEYWORD:
		return (b.peekN(nextTokenIndex).Kind() == common.BACKTICK_TOKEN)
	case common.START_KEYWORD,
		common.FLUSH_KEYWORD,
		common.WAIT_KEYWORD:
		fallthrough
	default:
		return false
	}
}

func (b *BallerinaParser) parseSyncSendAction(expression tree.STNode) tree.STNode {
	syncSendToken := b.parseSyncSendToken()
	peerWorker := b.parsePeerWorkerName()
	return tree.CreateSyncSendActionNode(expression, syncSendToken, peerWorker)
}

func (b *BallerinaParser) parsePeerWorkerName() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.IDENTIFIER_TOKEN, common.FUNCTION_KEYWORD:
		return tree.CreateSimpleNameReferenceNode(b.consume())
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME)
		return b.parsePeerWorkerName()
	}
}

func (b *BallerinaParser) parseSyncSendToken() tree.STNode {
	token := b.peek()
	if token.Kind() == common.SYNC_SEND_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN)
		return b.parseSyncSendToken()
	}
}

func (b *BallerinaParser) parseReceiveAction() tree.STNode {
	leftArrow := b.parseLeftArrowToken()
	receiveWorkers := b.parseReceiveWorkers()
	return tree.CreateReceiveActionNode(leftArrow, receiveWorkers)
}

func (b *BallerinaParser) parseReceiveWorkers() tree.STNode {
	switch b.peek().Kind() {
	case common.FUNCTION_KEYWORD, common.IDENTIFIER_TOKEN:
		return b.parseSingleOrAlternateReceiveWorkers()
	case common.OPEN_BRACE_TOKEN:
		return b.parseMultipleReceiveWorkers()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS)
		return b.parseReceiveWorkers()
	}
}

func (b *BallerinaParser) parseSingleOrAlternateReceiveWorkers() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER)
	var workers []tree.STNode
	peerWorker := b.parsePeerWorkerName()
	workers = append(workers, peerWorker)
	nextToken := b.peek()
	if nextToken.Kind() != common.PIPE_TOKEN {
		b.endContext()
		return peerWorker
	}
	for nextToken.Kind() == common.PIPE_TOKEN {
		pipeToken := b.consume()
		workers = append(workers, pipeToken)
		peerWorker = b.parsePeerWorkerName()
		workers = append(workers, peerWorker)
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateAlternateReceiveNode(tree.CreateNodeList(workers...))
}

func (b *BallerinaParser) parseMultipleReceiveWorkers() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS)
	openBrace := b.parseOpenBrace()
	receiveFields := b.parseReceiveFields()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	openBrace = b.cloneWithDiagnosticIfListEmpty(receiveFields, openBrace,
		&common.ERROR_MISSING_RECEIVE_FIELD_IN_RECEIVE_ACTION)
	return tree.CreateReceiveFieldsNode(openBrace, receiveFields, closeBrace)
}

func (b *BallerinaParser) parseReceiveFields() tree.STNode {
	var receiveFields []tree.STNode
	nextToken := b.peek()
	if b.isEndOfReceiveFields(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	receiveField := b.parseReceiveField()
	receiveFields = append(receiveFields, receiveField)
	nextToken = b.peek()
	var recieveFieldEnd tree.STNode
	for !b.isEndOfReceiveFields(nextToken.Kind()) {
		recieveFieldEnd = b.parseReceiveFieldEnd()
		if recieveFieldEnd == nil {
			break
		}
		receiveFields = append(receiveFields, recieveFieldEnd)
		receiveField = b.parseReceiveField()
		receiveFields = append(receiveFields, receiveField)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(receiveFields...)
}

func (b *BallerinaParser) isEndOfReceiveFields(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseReceiveFieldEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END)
		return b.parseReceiveFieldEnd()
	}
}

func (b *BallerinaParser) parseReceiveField() tree.STNode {
	switch b.peek().Kind() {
	case common.FUNCTION_KEYWORD:
		functionKeyword := b.consume()
		return tree.CreateSimpleNameReferenceNode(functionKeyword)
	case common.IDENTIFIER_TOKEN:
		identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME)
		return b.createReceiveField(identifier)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECEIVE_FIELD)
		return b.parseReceiveField()
	}
}

func (b *BallerinaParser) createReceiveField(identifier tree.STNode) tree.STNode {
	if b.peek().Kind() != common.COLON_TOKEN {
		return tree.CreateSimpleNameReferenceNode(identifier)
	}
	identifier = tree.CreateSimpleNameReferenceNode(identifier)
	colon := b.parseColon()
	peerWorker := b.parsePeerWorkerName()
	return tree.CreateReceiveFieldNode(identifier, colon, peerWorker)
}

func (b *BallerinaParser) parseLeftArrowToken() tree.STNode {
	token := b.peek()
	if token.Kind() == common.LEFT_ARROW_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN)
		return b.parseLeftArrowToken()
	}
}

func (b *BallerinaParser) parseSignedRightShiftToken() tree.STNode {
	firstToken := b.consume()
	if firstToken.Kind() == common.DOUBLE_GT_TOKEN {
		return firstToken
	}
	endLGToken := b.consume()
	var doubleGTToken tree.STNode
	doubleGTToken = tree.CreateToken(common.DOUBLE_GT_TOKEN, firstToken.LeadingMinutiae(),
		endLGToken.TrailingMinutiae())
	if b.hasTrailingMinutiae(firstToken) {
		doubleGTToken = tree.AddDiagnostic(doubleGTToken,
			&common.ERROR_NO_WHITESPACES_ALLOWED_IN_RIGHT_SHIFT_OP)
	}
	return doubleGTToken
}

func (b *BallerinaParser) parseUnsignedRightShiftToken() tree.STNode {
	firstToken := b.consume()
	if firstToken.Kind() == common.TRIPPLE_GT_TOKEN {
		return firstToken
	}
	middleGTToken := b.consume()
	endLGToken := b.consume()
	var unsignedRightShiftToken tree.STNode
	unsignedRightShiftToken = tree.CreateToken(common.TRIPPLE_GT_TOKEN,
		firstToken.LeadingMinutiae(), endLGToken.TrailingMinutiae())
	validOpenGTToken := (!b.hasTrailingMinutiae(firstToken))
	validMiddleGTToken := (!b.hasTrailingMinutiae(middleGTToken))
	if validOpenGTToken && validMiddleGTToken {
		return unsignedRightShiftToken
	}
	unsignedRightShiftToken = tree.AddDiagnostic(unsignedRightShiftToken,
		&common.ERROR_NO_WHITESPACES_ALLOWED_IN_UNSIGNED_RIGHT_SHIFT_OP)
	return unsignedRightShiftToken
}

func (b *BallerinaParser) parseWaitAction() tree.STNode {
	waitKeyword := b.parseWaitKeyword()
	if b.peek().Kind() == common.OPEN_BRACE_TOKEN {
		return b.parseMultiWaitAction(waitKeyword)
	}
	return b.parseSingleOrAlternateWaitAction(waitKeyword)
}

func (b *BallerinaParser) parseWaitKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.WAIT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_WAIT_KEYWORD)
		return b.parseWaitKeyword()
	}
}

func (b *BallerinaParser) parseSingleOrAlternateWaitAction(waitKeyword tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS)
	nextToken := b.peek()
	if b.isEndOfWaitFutureExprList(nextToken.Kind()) {
		b.endContext()
		waitFutureExprs := tree.CreateSimpleNameReferenceNode(tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil))
		waitFutureExprs = tree.AddDiagnostic(waitFutureExprs,
			&common.ERROR_MISSING_WAIT_FUTURE_EXPRESSION)
		return tree.CreateWaitActionNode(waitKeyword, waitFutureExprs)
	}
	var waitFutureExprList []tree.STNode
	waitField := b.parseWaitFutureExpr()
	waitFutureExprList = append(waitFutureExprList, waitField)
	nextToken = b.peek()
	var waitFutureExprEnd tree.STNode
	for !b.isEndOfWaitFutureExprList(nextToken.Kind()) {
		waitFutureExprEnd = b.parseWaitFutureExprEnd()
		if waitFutureExprEnd == nil {
			break
		}
		waitFutureExprList = append(waitFutureExprList, waitFutureExprEnd)
		waitField = b.parseWaitFutureExpr()
		waitFutureExprList = append(waitFutureExprList, waitField)
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateWaitActionNode(waitKeyword, waitFutureExprList[0])
}

func (b *BallerinaParser) isEndOfWaitFutureExprList(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACE_TOKEN, common.SEMICOLON_TOKEN, common.OPEN_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseWaitFutureExpr() tree.STNode {
	waitFutureExpr := b.parseActionOrExpression()
	if waitFutureExpr.Kind() == common.MAPPING_CONSTRUCTOR {
		waitFutureExpr = tree.AddDiagnostic(waitFutureExpr,
			&common.ERROR_MAPPING_CONSTRUCTOR_EXPR_AS_A_WAIT_EXPR)
	} else if b.isAction(waitFutureExpr) {
		waitFutureExpr = tree.AddDiagnostic(waitFutureExpr, &common.ERROR_ACTION_AS_A_WAIT_EXPR)
	}
	return waitFutureExpr
}

func (b *BallerinaParser) parseWaitFutureExprEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.PIPE_TOKEN:
		return b.parsePipeToken()
	default:
		if b.isEndOfWaitFutureExprList(nextToken.Kind()) || (!b.isValidExpressionStart(nextToken.Kind(), 1)) {
			return nil
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END)
		return b.parseWaitFutureExprEnd()
	}
}

func (b *BallerinaParser) parseMultiWaitAction(waitKeyword tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS)
	openBrace := b.parseOpenBrace()
	waitFields := b.parseWaitFields()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	openBrace = b.cloneWithDiagnosticIfListEmpty(waitFields, openBrace,
		&common.ERROR_MISSING_WAIT_FIELD_IN_WAIT_ACTION)
	waitFieldsNode := tree.CreateWaitFieldsListNode(openBrace, waitFields, closeBrace)
	return tree.CreateWaitActionNode(waitKeyword, waitFieldsNode)
}

func (b *BallerinaParser) parseWaitFields() tree.STNode {
	var waitFields []tree.STNode
	nextToken := b.peek()
	if b.isEndOfWaitFields(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	waitField := b.parseWaitField()
	waitFields = append(waitFields, waitField)
	nextToken = b.peek()
	var waitFieldEnd tree.STNode
	for !b.isEndOfWaitFields(nextToken.Kind()) {
		waitFieldEnd = b.parseWaitFieldEnd()
		if waitFieldEnd == nil {
			break
		}
		waitFields = append(waitFields, waitFieldEnd)
		waitField = b.parseWaitField()
		waitFields = append(waitFields, waitField)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(waitFields...)
}

func (b *BallerinaParser) isEndOfWaitFields(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseWaitFieldEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WAIT_FIELD_END)
		return b.parseWaitFieldEnd()
	}
}

func (b *BallerinaParser) parseWaitField() tree.STNode {
	switch b.peek().Kind() {
	case common.IDENTIFIER_TOKEN:
		identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME)
		identifier = tree.CreateSimpleNameReferenceNode(identifier)
		return b.createQualifiedWaitField(identifier)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME)
		return b.parseWaitField()
	}
}

func (b *BallerinaParser) createQualifiedWaitField(identifier tree.STNode) tree.STNode {
	if b.peek().Kind() != common.COLON_TOKEN {
		return identifier
	}
	colon := b.parseColon()
	waitFutureExpr := b.parseWaitFutureExpr()
	return tree.CreateWaitFieldNode(identifier, colon, waitFutureExpr)
}

func (b *BallerinaParser) parseAnnotAccessExpression(lhsExpr tree.STNode, isInConditionalExpr bool) tree.STNode {
	annotAccessToken := b.parseAnnotChainingToken()
	annotTagReference := b.parseFieldAccessIdentifier(isInConditionalExpr)
	return tree.CreateAnnotAccessExpressionNode(lhsExpr, annotAccessToken, annotTagReference)
}

func (b *BallerinaParser) parseAnnotChainingToken() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ANNOT_CHAINING_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN)
		return b.parseAnnotChainingToken()
	}
}

func (b *BallerinaParser) parseFieldAccessIdentifier(isInConditionalExpr bool) tree.STNode {
	nextToken := b.peek()
	if !b.isPredeclaredIdentifier(nextToken.Kind()) {
		var identifier tree.STNode = tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_IDENTIFIER)
		return b.parseQualifiedIdentifierNode(identifier, isInConditionalExpr)
	}
	return b.parseQualifiedIdentifierInner(common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER, isInConditionalExpr)
}

func (b *BallerinaParser) parseQueryAction(queryConstructType tree.STNode, queryPipeline tree.STNode, selectClause tree.STNode, collectClause tree.STNode) tree.STNode {
	if queryConstructType != nil {
		queryPipeline = tree.CloneWithLeadingInvalidNodeMinutiae(queryPipeline, queryConstructType,
			&common.ERROR_QUERY_CONSTRUCT_TYPE_IN_QUERY_ACTION)
	}
	if selectClause != nil {
		queryPipeline = tree.CloneWithTrailingInvalidNodeMinutiae(queryPipeline, selectClause,
			&common.ERROR_SELECT_CLAUSE_IN_QUERY_ACTION)
	}
	if collectClause != nil {
		queryPipeline = tree.CloneWithTrailingInvalidNodeMinutiae(queryPipeline, collectClause,
			&common.ERROR_COLLECT_CLAUSE_IN_QUERY_ACTION)
	}
	b.startContext(common.PARSER_RULE_CONTEXT_DO_CLAUSE)
	doKeyword := b.parseDoKeyword()
	blockStmt := b.parseBlockNode()
	b.endContext()
	return tree.CreateQueryActionNode(queryPipeline, doKeyword, blockStmt)
}

func (b *BallerinaParser) parseDoKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.DO_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_DO_KEYWORD)
		return b.parseDoKeyword()
	}
}

func (b *BallerinaParser) parseOptionalFieldAccessExpression(lhsExpr tree.STNode, isInConditionalExpr bool) tree.STNode {
	optionalFieldAccessToken := b.parseOptionalChainingToken()
	fieldName := b.parseFieldAccessIdentifier(isInConditionalExpr)
	return tree.CreateOptionalFieldAccessExpressionNode(lhsExpr, optionalFieldAccessToken, fieldName)
}

func (b *BallerinaParser) parseOptionalChainingToken() tree.STNode {
	token := b.peek()
	if token.Kind() == common.OPTIONAL_CHAINING_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN)
		return b.parseOptionalChainingToken()
	}
}

func (b *BallerinaParser) parseConditionalExpression(lhsExpr tree.STNode, isInConditionalExpr bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION)
	questionMark := b.parseQuestionMark()
	middleExpr := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET, true, false, true)
	if b.peek().Kind() != common.COLON_TOKEN {
		if middleExpr.Kind() == common.CONDITIONAL_EXPRESSION {
			innerConditionalExpr, ok := middleExpr.(*tree.STConditionalExpressionNode)
			if !ok {
				panic("expected STConditionalExpressionNode")
			}
			innerMiddleExpr := innerConditionalExpr.MiddleExpression
			rightMostQNameRef := tree.GetQualifiedNameRefNode(innerMiddleExpr, false)
			if rightMostQNameRef != nil {
				middleExpr = b.generateConditionalExprForRightMost(innerConditionalExpr.LhsExpression,
					innerConditionalExpr.QuestionMarkToken, innerMiddleExpr, rightMostQNameRef)
				b.endContext()
				return tree.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr,
					innerConditionalExpr.ColonToken, innerConditionalExpr.EndExpression)
			}
			leftMostQNameRef := tree.GetQualifiedNameRefNode(innerMiddleExpr, true)
			if leftMostQNameRef != nil {
				middleExpr = b.generateConditionalExprForLeftMost(innerConditionalExpr.LhsExpression,
					innerConditionalExpr.QuestionMarkToken, innerMiddleExpr, leftMostQNameRef)
				b.endContext()
				return tree.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr,
					innerConditionalExpr.ColonToken, innerConditionalExpr.EndExpression)
			}
		}
		rightMostQNameRef := tree.GetQualifiedNameRefNode(middleExpr, false)
		if rightMostQNameRef != nil {
			b.endContext()
			return b.generateConditionalExprForRightMost(lhsExpr, questionMark, middleExpr, rightMostQNameRef)
		}
		leftMostQNameRef := tree.GetQualifiedNameRefNode(middleExpr, true)
		if leftMostQNameRef != nil {
			b.endContext()
			return b.generateConditionalExprForLeftMost(lhsExpr, questionMark, middleExpr, leftMostQNameRef)
		}
	}
	return b.parseConditionalExprRhs(lhsExpr, questionMark, middleExpr, isInConditionalExpr)
}

func (b *BallerinaParser) generateConditionalExprForRightMost(lhsExpr tree.STNode, questionMark tree.STNode, middleExpr tree.STNode, rightMostQualifiedNameRef tree.STNode) tree.STNode {
	qualifiedNameRef, ok := rightMostQualifiedNameRef.(*tree.STQualifiedNameReferenceNode)
	if !ok {
		panic("expected STQualifiedNameReferenceNode")
	}
	endExpr := tree.CreateSimpleNameReferenceNode(qualifiedNameRef.Identifier)
	simpleNameRef := tree.GetSimpleNameRefNode(qualifiedNameRef.ModulePrefix)
	middleExpr = tree.Replace(middleExpr, rightMostQualifiedNameRef, simpleNameRef)
	return tree.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr, qualifiedNameRef.Colon,
		endExpr)
}

func (b *BallerinaParser) generateConditionalExprForLeftMost(lhsExpr tree.STNode, questionMark tree.STNode, middleExpr tree.STNode, leftMostQualifiedNameRef tree.STNode) tree.STNode {
	qualifiedNameRef, ok := leftMostQualifiedNameRef.(*tree.STQualifiedNameReferenceNode)
	if !ok {
		panic("expected STQualifiedNameReferenceNode")
	}
	simpleNameRef := tree.CreateSimpleNameReferenceNode(qualifiedNameRef.Identifier)
	endExpr := tree.Replace(middleExpr, leftMostQualifiedNameRef, simpleNameRef)
	middleExpr = tree.GetSimpleNameRefNode(qualifiedNameRef.ModulePrefix)
	return tree.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr, qualifiedNameRef.Colon,
		endExpr)
}

func (b *BallerinaParser) parseConditionalExprRhs(lhsExpr tree.STNode, questionMark tree.STNode, middleExpr tree.STNode, isInConditionalExpr bool) tree.STNode {
	colon := b.parseColon()
	b.endContext()
	endExpr := b.parseExpressionWithConditional(OPERATOR_PRECEDENCE_ANON_FUNC_OR_LET, true, false,
		isInConditionalExpr)
	return tree.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr, colon, endExpr)
}

func (b *BallerinaParser) parseEnumDeclaration(metadata tree.STNode, qualifier tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION)
	enumKeywordToken := b.parseEnumKeyword()
	identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_MODULE_ENUM_NAME)
	openBraceToken := b.parseOpenBrace()
	enumMemberList := b.parseEnumMemberList()
	closeBraceToken := b.parseCloseBrace()
	semicolon := b.parseOptionalSemicolon()
	b.endContext()
	enumDecl := tree.CreateEnumDeclarationNode(metadata, qualifier, enumKeywordToken, identifier,
		openBraceToken, enumMemberList, closeBraceToken, semicolon)
	return b.cloneWithDiagnosticIfListEmpty(enumMemberList, enumDecl,
		&common.ERROR_MISSING_ENUM_MEMBER)
}

func (b *BallerinaParser) parseEnumKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ENUM_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ENUM_KEYWORD)
		return b.parseEnumKeyword()
	}
}

func (b *BallerinaParser) parseEnumMemberList() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST)
	if b.peek().Kind() == common.CLOSE_BRACE_TOKEN {
		return tree.CreateEmptyNodeList()
	}
	var enumMemberList []tree.STNode
	enumMember := b.parseEnumMember()
	var enumMemberRhs tree.STNode
	for b.peek().Kind() != common.CLOSE_BRACE_TOKEN {
		enumMemberRhs = b.parseEnumMemberEnd()
		if enumMemberRhs == nil {
			break
		}
		enumMemberList = append(enumMemberList, enumMember)
		enumMemberList = append(enumMemberList, enumMemberRhs)
		enumMember = b.parseEnumMember()
	}
	enumMemberList = append(enumMemberList, enumMember)
	b.endContext()
	return tree.CreateNodeList(enumMemberList...)
}

func (b *BallerinaParser) parseEnumMember() tree.STNode {
	var metadata tree.STNode
	switch b.peek().Kind() {
	case common.DOCUMENTATION_STRING, common.AT_TOKEN:
		metadata = b.parseMetaData()
	default:
		metadata = tree.CreateEmptyNode()
	}
	identifierNode := b.parseIdentifier(common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME)
	return b.parseEnumMemberRhs(metadata, identifierNode)
}

func (b *BallerinaParser) parseEnumMemberRhs(metadata tree.STNode, identifierNode tree.STNode) tree.STNode {
	var equalToken tree.STNode
	var constExprNode tree.STNode
	switch b.peek().Kind() {
	case common.EQUAL_TOKEN:
		equalToken = b.parseAssignOp()
		constExprNode = b.parseExpression()
	case common.COMMA_TOKEN, common.CLOSE_BRACE_TOKEN:
		equalToken = tree.CreateEmptyNode()
		constExprNode = tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS)
		return b.parseEnumMemberRhs(metadata, identifierNode)
	}
	return tree.CreateEnumMemberNode(metadata, identifierNode, equalToken, constExprNode)
}

func (b *BallerinaParser) parseEnumMemberEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END)
		return b.parseEnumMemberEnd()
	}
}

func (b *BallerinaParser) parseTransactionStmtOrVarDecl(annots tree.STNode, qualifiers []tree.STNode, transactionKeyword tree.STToken) (tree.STNode, []tree.STNode) {
	switch b.peek().Kind() {
	case common.OPEN_BRACE_TOKEN:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTransactionStatement(transactionKeyword), qualifiers
	case common.COLON_TOKEN:
		if b.getNextNextToken().Kind() == common.IDENTIFIER_TOKEN {
			typeDesc := b.parseQualifiedIdentifierWithPredeclPrefix(transactionKeyword, false)
			return b.parseVarDeclTypeDescRhs(typeDesc, annots, qualifiers, true, false)
		}
		fallthrough
	default:
		solution := b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF)
		if (solution.Action == ACTION_KEEP) || ((solution.Action == ACTION_INSERT) && (solution.TokenKind == common.COLON_TOKEN)) {
			typeDesc := b.parseQualifiedIdentifierWithPredeclPrefix(transactionKeyword, false)
			return b.parseVarDeclTypeDescRhs(typeDesc, annots, qualifiers, true, false)
		}
		return b.parseTransactionStmtOrVarDecl(annots, qualifiers, transactionKeyword)
	}
}

func (b *BallerinaParser) parseTransactionStatement(transactionKeyword tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TRANSACTION_STMT)
	blockStmt := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateTransactionStatementNode(transactionKeyword, blockStmt, onFailClause)
}

func (b *BallerinaParser) parseCommitAction() tree.STNode {
	commitKeyword := b.parseCommitKeyword()
	return tree.CreateCommitActionNode(commitKeyword)
}

func (b *BallerinaParser) parseCommitKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.COMMIT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD)
		return b.parseCommitKeyword()
	}
}

func (b *BallerinaParser) parseRetryStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RETRY_STMT)
	retryKeyword := b.parseRetryKeyword()
	retryStmt := b.parseRetryKeywordRhs(retryKeyword)
	return retryStmt
}

func (b *BallerinaParser) parseRetryKeywordRhs(retryKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.LT_TOKEN:
		return b.parseRetryTypeParamRhs(retryKeyword, b.parseTypeParameter())
	case common.OPEN_PAREN_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.TRANSACTION_KEYWORD:
		return b.parseRetryTypeParamRhs(retryKeyword, tree.CreateEmptyNode())
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS)
		return b.parseRetryKeywordRhs(retryKeyword)
	}
}

func (b *BallerinaParser) parseRetryTypeParamRhs(retryKeyword tree.STNode, typeParam tree.STNode) tree.STNode {
	var args tree.STNode
	switch b.peek().Kind() {
	case common.OPEN_PAREN_TOKEN:
		args = b.parseParenthesizedArgList()
	case common.OPEN_BRACE_TOKEN,
		common.TRANSACTION_KEYWORD:
		args = tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS)
		return b.parseRetryTypeParamRhs(retryKeyword, typeParam)
	}
	blockStmt := b.parseRetryBody()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateRetryStatementNode(retryKeyword, typeParam, args, blockStmt, onFailClause)
}

func (b *BallerinaParser) parseRetryBody() tree.STNode {
	switch b.peek().Kind() {
	case common.OPEN_BRACE_TOKEN:
		return b.parseBlockNode()
	case common.TRANSACTION_KEYWORD:
		return b.parseTransactionStatement(b.consume())
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RETRY_BODY)
		return b.parseRetryBody()
	}
}

func (b *BallerinaParser) parseOptionalOnFailClause() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.ON_KEYWORD {
		return b.parseOnFailClause()
	}
	if b.isEndOfRegularCompoundStmt(nextToken.Kind()) {
		return tree.CreateEmptyNode()
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS)
	return b.parseOptionalOnFailClause()
}

func (b *BallerinaParser) isEndOfRegularCompoundStmt(nodeKind common.SyntaxKind) bool {
	switch nodeKind {
	case common.CLOSE_BRACE_TOKEN, common.SEMICOLON_TOKEN, common.AT_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return b.isStatementStartingToken(nodeKind)
	}
}

func (b *BallerinaParser) isStatementStartingToken(nodeKind common.SyntaxKind) bool {
	switch nodeKind {
	case common.FINAL_KEYWORD, common.IF_KEYWORD, common.WHILE_KEYWORD, common.DO_KEYWORD,
		common.PANIC_KEYWORD, common.CONTINUE_KEYWORD, common.BREAK_KEYWORD, common.RETURN_KEYWORD,
		common.LOCK_KEYWORD, common.OPEN_BRACE_TOKEN, common.FORK_KEYWORD, common.FOREACH_KEYWORD,
		common.XMLNS_KEYWORD, common.TRANSACTION_KEYWORD, common.RETRY_KEYWORD, common.ROLLBACK_KEYWORD,
		common.MATCH_KEYWORD, common.FAIL_KEYWORD, common.CHECK_KEYWORD, common.CHECKPANIC_KEYWORD,
		common.TRAP_KEYWORD, common.START_KEYWORD, common.FLUSH_KEYWORD, common.LEFT_ARROW_TOKEN,
		common.WAIT_KEYWORD, common.COMMIT_KEYWORD, common.WORKER_KEYWORD, common.TYPE_KEYWORD,
		common.CONST_KEYWORD:
		return true
	default:
		if b.isTypeStartingToken(nodeKind) {
			return true
		}
		if b.isValidExpressionStart(nodeKind, 1) {
			return true
		}
		return false
	}
}

func (b *BallerinaParser) parseOnFailClause() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE)
	onKeyword := b.parseOnKeyword()
	failKeyword := b.parseFailKeyword()
	typedBindingPattern := b.parseOnfailOptionalBP()
	blockStatement := b.parseBlockNode()
	b.endContext()
	return tree.CreateOnFailClauseNode(onKeyword, failKeyword, typedBindingPattern,
		blockStatement)
}

func (b *BallerinaParser) parseOnfailOptionalBP() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.OPEN_BRACE_TOKEN {
		return tree.CreateEmptyNode()
	} else if b.isTypeStartingToken(nextToken.Kind()) {
		return b.parseTypedBindingPattern()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN)
		return b.parseOnfailOptionalBP()
	}
}

func (b *BallerinaParser) parseTypedBindingPattern() tree.STNode {
	typeDescriptor := b.parseTypeDescriptorWithoutQualifiers(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true, false, TYPE_PRECEDENCE_DEFAULT)
	bindingPattern := b.parseBindingPattern()
	return tree.CreateTypedBindingPatternNode(typeDescriptor, bindingPattern)
}

func (b *BallerinaParser) parseRetryKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.RETRY_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RETRY_KEYWORD)
		return b.parseRetryKeyword()
	}
}

func (b *BallerinaParser) parseRollbackStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ROLLBACK_STMT)
	rollbackKeyword := b.parseRollbackKeyword()
	var expression tree.STNode
	if b.peek().Kind() == common.SEMICOLON_TOKEN {
		expression = tree.CreateEmptyNode()
	} else {
		expression = b.parseExpression()
	}
	semicolon := b.parseSemicolon()
	b.endContext()
	return tree.CreateRollbackStatementNode(rollbackKeyword, expression, semicolon)
}

func (b *BallerinaParser) parseRollbackKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.ROLLBACK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD)
		return b.parseRollbackKeyword()
	}
}

func (b *BallerinaParser) parseTransactionalExpression() tree.STNode {
	transactionalKeyword := b.parseTransactionalKeyword()
	return tree.CreateTransactionalExpressionNode(transactionalKeyword)
}

func (b *BallerinaParser) parseTransactionalKeyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.TRANSACTIONAL_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD)
		return b.parseTransactionalKeyword()
	}
}

func (b *BallerinaParser) parseByteArrayLiteral() tree.STNode {
	var ty tree.STNode
	if b.peek().Kind() == common.BASE16_KEYWORD {
		ty = b.parseBase16Keyword()
	} else {
		ty = b.parseBase64Keyword()
	}
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	if startingBackTick.IsMissing() {
		startingBackTick = tree.CreateMissingToken(common.BACKTICK_TOKEN, nil)
		endingBackTick := tree.CreateMissingToken(common.BACKTICK_TOKEN, nil)
		content := tree.CreateEmptyNode()
		byteArrayLiteral := tree.CreateByteArrayLiteralNode(ty, startingBackTick, content, endingBackTick)
		byteArrayLiteral = tree.AddDiagnostic(byteArrayLiteral, &common.ERROR_MISSING_BYTE_ARRAY_CONTENT)
		return byteArrayLiteral
	}
	content := b.parseByteArrayContent()
	return b.parseByteArrayLiteralWithContent(ty, startingBackTick, content)
}

func (b *BallerinaParser) parseByteArrayLiteralWithContent(typeKeyword tree.STNode, startingBackTick tree.STNode, byteArrayContent tree.STNode) tree.STNode {
	content := tree.CreateEmptyNode()
	newStartingBackTick := startingBackTick
	items, ok := byteArrayContent.(*tree.STNodeList)
	if !ok {
		panic("byteArrayContent is not a STNodeList")
	}
	if items.Size() == 1 {
		item := items.Get(0)
		if (typeKeyword.Kind() == common.BASE16_KEYWORD) && (!isValidBase16LiteralContent(tree.ToSourceCode(item))) {
			newStartingBackTick = tree.CloneWithTrailingInvalidNodeMinutiae(startingBackTick, item,
				&common.ERROR_INVALID_BASE16_CONTENT_IN_BYTE_ARRAY_LITERAL)
		} else if (typeKeyword.Kind() == common.BASE64_KEYWORD) && (!isValidBase64LiteralContent(tree.ToSourceCode(item))) {
			newStartingBackTick = tree.CloneWithTrailingInvalidNodeMinutiae(startingBackTick, item,
				&common.ERROR_INVALID_BASE64_CONTENT_IN_BYTE_ARRAY_LITERAL)
		} else if item.Kind() != common.TEMPLATE_STRING {
			newStartingBackTick = tree.CloneWithTrailingInvalidNodeMinutiae(startingBackTick, item,
				&common.ERROR_INVALID_CONTENT_IN_BYTE_ARRAY_LITERAL)
		} else {
			content = item
		}
	} else if items.Size() > 1 {
		clonedStartingBackTick := startingBackTick
		for index := 0; index < items.Size(); index++ {
			item := items.Get(index)
			clonedStartingBackTick = tree.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(clonedStartingBackTick, item)
		}
		newStartingBackTick = tree.AddDiagnostic(clonedStartingBackTick,
			&common.ERROR_INVALID_CONTENT_IN_BYTE_ARRAY_LITERAL)
	}
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return tree.CreateByteArrayLiteralNode(typeKeyword, newStartingBackTick, content, endingBackTick)
}

func (b *BallerinaParser) parseBase16Keyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.BASE16_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BASE16_KEYWORD)
		return b.parseBase16Keyword()
	}
}

func (b *BallerinaParser) parseBase64Keyword() tree.STNode {
	token := b.peek()
	if token.Kind() == common.BASE64_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BASE64_KEYWORD)
		return b.parseBase64Keyword()
	}
}

func (b *BallerinaParser) parseByteArrayContent() tree.STNode {
	nextToken := b.peek()
	var items []tree.STNode
	for !b.isEndOfBacktickContent(nextToken.Kind()) {
		content := b.parseTemplateItem()
		items = append(items, content)
		nextToken = b.peek()
	}
	return tree.CreateNodeList(items...)
}

func (b *BallerinaParser) parseXMLFilterExpression(lhsExpr tree.STNode) tree.STNode {
	xmlNamePatternChain := b.parseXMLFilterExpressionRhs()
	return tree.CreateXMLFilterExpressionNode(lhsExpr, xmlNamePatternChain)
}

func (b *BallerinaParser) parseXMLFilterExpressionRhs() tree.STNode {
	dotLTToken := b.parseDotLTToken()
	return b.parseXMLNamePatternChain(dotLTToken)
}

func (b *BallerinaParser) parseXMLNamePatternChain(startToken tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN)
	xmlNamePattern := b.parseXMLNamePattern()
	gtToken := b.parseGTToken()
	b.endContext()
	startToken = b.cloneWithDiagnosticIfListEmpty(xmlNamePattern, startToken,
		&common.ERROR_MISSING_XML_ATOMIC_NAME_PATTERN)
	return tree.CreateXMLNamePatternChainingNode(startToken, xmlNamePattern, gtToken)
}

func (b *BallerinaParser) parseXMLStepExtends() tree.STNode {
	nextToken := b.peek()
	if b.isEndOfXMLStepExtend(nextToken.Kind()) {
		return tree.CreateEmptyNodeList()
	}
	var xmlStepExtendList []tree.STNode
	b.startContext(common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS)
	var stepExtension tree.STNode
	for !b.isEndOfXMLStepExtend(nextToken.Kind()) {
		if nextToken.Kind() == common.DOT_TOKEN {
			stepExtension = b.parseXMLStepMethodCallExtend()
		} else if nextToken.Kind() == common.DOT_LT_TOKEN {
			stepExtension = b.parseXMLFilterExpressionRhs()
		} else {
			stepExtension = b.parseXMLIndexedStepExtend()
		}
		xmlStepExtendList = append(xmlStepExtendList, stepExtension)
		nextToken = b.peek()
	}
	b.endContext()
	return tree.CreateNodeList(xmlStepExtendList...)
}

func (b *BallerinaParser) parseXMLIndexedStepExtend() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR)
	openBracket := b.parseOpenBracket()
	keyExpr := b.parseKeyExpr(true)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return tree.CreateXMLStepIndexedExtendNode(openBracket, keyExpr, closeBracket)
}

func (b *BallerinaParser) parseXMLStepMethodCallExtend() tree.STNode {
	dotToken := b.parseDotToken()
	methodName := b.parseMethodName()
	parenthesizedArgsList := b.parseParenthesizedArgList()
	return tree.CreateXMLStepMethodCallExtendNode(dotToken, methodName, parenthesizedArgsList)
}

func (b *BallerinaParser) parseMethodName() tree.STNode {
	if b.isSpecialMethodName(b.peek()) {
		return b.getKeywordAsSimpleNameRef()
	}
	return tree.CreateSimpleNameReferenceNode(b.parseIdentifier(common.PARSER_RULE_CONTEXT_IDENTIFIER))
}

func (b *BallerinaParser) parseDotLTToken() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.DOT_LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN)
		return b.parseDotLTToken()
	}
}

func (b *BallerinaParser) parseXMLNamePattern() tree.STNode {
	var xmlAtomicNamePatternList []tree.STNode
	nextToken := b.peek()
	if b.isEndOfXMLNamePattern(nextToken.Kind()) {
		return tree.CreateNodeList(xmlAtomicNamePatternList...)
	}
	xmlAtomicNamePattern := b.parseXMLAtomicNamePattern()
	xmlAtomicNamePatternList = append(xmlAtomicNamePatternList, xmlAtomicNamePattern)
	var separator tree.STNode
	for !b.isEndOfXMLNamePattern(b.peek().Kind()) {
		separator = b.parseXMLNamePatternSeparator()
		if separator == nil {
			break
		}
		xmlAtomicNamePatternList = append(xmlAtomicNamePatternList, separator)
		xmlAtomicNamePattern = b.parseXMLAtomicNamePattern()
		xmlAtomicNamePatternList = append(xmlAtomicNamePatternList, xmlAtomicNamePattern)
	}
	return tree.CreateNodeList(xmlAtomicNamePatternList...)
}

func (b *BallerinaParser) isEndOfXMLNamePattern(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.GT_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isEndOfXMLStepExtend(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.OPEN_BRACKET_TOKEN, common.DOT_LT_TOKEN:
		return false
	case common.DOT_TOKEN:
		return b.peekN(3).Kind() != common.OPEN_PAREN_TOKEN
	default:
		return true
	}
}

func (b *BallerinaParser) parseXMLNamePatternSeparator() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.PIPE_TOKEN:
		return b.consume()
	case common.GT_TOKEN, common.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS)
		return b.parseXMLNamePatternSeparator()
	}
}

func (b *BallerinaParser) parseXMLAtomicNamePattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN)
	atomicNamePattern := b.parseXMLAtomicNamePatternBody()
	b.endContext()
	return atomicNamePattern
}

func (b *BallerinaParser) parseXMLAtomicNamePatternBody() tree.STNode {
	token := b.peek()
	var identifier tree.STNode
	switch token.Kind() {
	case common.ASTERISK_TOKEN:
		return b.consume()
	case common.IDENTIFIER_TOKEN:
		identifier = b.consume()
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START)
		return b.parseXMLAtomicNamePatternBody()
	}
	return b.parseXMLAtomicNameIdentifier(identifier)
}

func (b *BallerinaParser) parseXMLAtomicNameIdentifier(identifier tree.STNode) tree.STNode {
	token := b.peek()
	if token.Kind() == common.COLON_TOKEN {
		colon := b.consume()
		nextToken := b.peek()
		if (nextToken.Kind() == common.IDENTIFIER_TOKEN) || (nextToken.Kind() == common.ASTERISK_TOKEN) {
			endToken := b.consume()
			return tree.CreateXMLAtomicNamePatternNode(identifier, colon, endToken)
		}
	}
	return tree.CreateSimpleNameReferenceNode(identifier)
}

func (b *BallerinaParser) parseXMLStepExpression(lhsExpr tree.STNode) tree.STNode {
	xmlStepStart := b.parseXMLStepStart()
	xmlStepExtends := b.parseXMLStepExtends()
	return tree.CreateXMLStepExpressionNode(lhsExpr, xmlStepStart, xmlStepExtends)
}

func (b *BallerinaParser) parseXMLStepStart() tree.STNode {
	token := b.peek()
	var startToken tree.STNode
	switch token.Kind() {
	case common.SLASH_ASTERISK_TOKEN:
		return b.consume()
	case common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
		startToken = b.parseDoubleSlashDoubleAsteriskLTToken()
	case common.SLASH_LT_TOKEN:
	default:
		startToken = b.parseSlashLTToken()
	}
	return b.parseXMLNamePatternChain(startToken)
}

func (b *BallerinaParser) parseSlashLTToken() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.SLASH_LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN)
		return b.parseSlashLTToken()
	}
}

func (b *BallerinaParser) parseDoubleSlashDoubleAsteriskLTToken() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN)
		return b.parseDoubleSlashDoubleAsteriskLTToken()
	}
}

func (b *BallerinaParser) parseMatchStatement() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MATCH_STMT)
	matchKeyword := b.parseMatchKeyword()
	actionOrExpr := b.parseActionOrExpression()
	b.startContext(common.PARSER_RULE_CONTEXT_MATCH_BODY)
	openBrace := b.parseOpenBrace()
	var matchClausesList []tree.STNode
	for !b.isEndOfMatchClauses(b.peek().Kind()) {
		clause := b.parseMatchClause()
		matchClausesList = append(matchClausesList, clause)
	}
	matchClauses := tree.CreateNodeList(matchClausesList...)
	if b.isNodeListEmpty(matchClauses) {
		openBrace = tree.AddDiagnostic(openBrace,
			&common.ERROR_MATCH_STATEMENT_SHOULD_HAVE_ONE_OR_MORE_MATCH_CLAUSES)
	}
	closeBrace := b.parseCloseBrace()
	b.endContext()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return tree.CreateMatchStatementNode(matchKeyword, actionOrExpr, openBrace, matchClauses, closeBrace,
		onFailClause)
}

func (b *BallerinaParser) parseMatchKeyword() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.MATCH_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MATCH_KEYWORD)
		return b.parseMatchKeyword()
	}
}

func (b *BallerinaParser) isEndOfMatchClauses(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACE_TOKEN, common.TYPE_KEYWORD:
		return true
	default:
		return b.isEndOfStatements()
	}
}

func (b *BallerinaParser) parseMatchClause() tree.STNode {
	matchPatterns := b.parseMatchPatternList()
	matchGuard := b.parseMatchGuard()
	rightDoubleArrow := b.parseDoubleRightArrow()
	blockStmt := b.parseBlockNode()
	if b.isNodeListEmpty(matchPatterns) {
		identifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		constantPattern := tree.CreateSimpleNameReferenceNode(identifier)
		matchPatterns = tree.CreateNodeList(constantPattern)
		errorCode := &common.ERROR_MISSING_MATCH_PATTERN
		if matchGuard != nil {
			matchGuard = tree.AddDiagnostic(matchGuard, errorCode)
		} else {
			rightDoubleArrow = tree.AddDiagnostic(rightDoubleArrow, errorCode)
		}
	}
	return tree.CreateMatchClauseNode(matchPatterns, matchGuard, rightDoubleArrow, blockStmt)
}

func (b *BallerinaParser) parseMatchGuard() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IF_KEYWORD:
		ifKeyword := b.parseIfKeyword()
		expr := b.parseExpressionWithMatchGuard(DEFAULT_OP_PRECEDENCE, true, false, true, false)
		return tree.CreateMatchGuardNode(ifKeyword, expr)
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		return tree.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD)
		return b.parseMatchGuard()
	}
}

func (b *BallerinaParser) parseMatchPatternList() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MATCH_PATTERN)
	var matchClauses []tree.STNode
	for !b.isEndOfMatchPattern(b.peek().Kind()) {
		clause := b.parseMatchPattern()
		if clause == nil {
			break
		}
		matchClauses = append(matchClauses, clause)
		seperator := b.parseMatchPatternListMemberRhs()
		if seperator == nil {
			break
		}
		matchClauses = append(matchClauses, seperator)
	}
	b.endContext()
	return tree.CreateNodeList(matchClauses...)
}

func (b *BallerinaParser) isEndOfMatchPattern(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.PIPE_TOKEN, common.IF_KEYWORD, common.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseMatchPattern() tree.STNode {
	nextToken := b.peek()
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		typeRefOrConstExpr := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_MATCH_PATTERN)
		return b.parseErrorMatchPatternOrConsPattern(typeRefOrConstExpr)
	}
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN:
		return b.parseSimpleConstExpr()
	case common.VAR_KEYWORD:
		return b.parseVarTypedBindingPattern()
	case common.OPEN_BRACKET_TOKEN:
		return b.parseListMatchPattern()
	case common.OPEN_BRACE_TOKEN:
		return b.parseMappingMatchPattern()
	case common.ERROR_KEYWORD:
		return b.parseErrorMatchPattern()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START)
		return b.parseMatchPattern()
	}
}

func (b *BallerinaParser) parseMatchPatternListMemberRhs() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.PIPE_TOKEN:
		return b.parsePipeToken()
	case common.IF_KEYWORD, common.RIGHT_DOUBLE_ARROW_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS)
		return b.parseMatchPatternListMemberRhs()
	}
}

func (b *BallerinaParser) parseVarTypedBindingPattern() tree.STNode {
	varKeyword := b.parseVarKeyword()
	varTypeDesc := CreateBuiltinSimpleNameReference(varKeyword)
	bindingPattern := b.parseBindingPattern()
	return tree.CreateTypedBindingPatternNode(varTypeDesc, bindingPattern)
}

func (b *BallerinaParser) parseVarKeyword() tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.VAR_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_VAR_KEYWORD)
		return b.parseVarKeyword()
	}
}

func (b *BallerinaParser) parseListMatchPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN)
	openBracketToken := b.parseOpenBracket()
	var matchPatternList []tree.STNode
	var listMatchPatternMemberRhs tree.STNode
	isEndOfFields := false
	for !b.IsEndOfListMatchPattern() {
		listMatchPatternMember := b.parseListMatchPatternMember()
		matchPatternList = append(matchPatternList, listMatchPatternMember)
		listMatchPatternMemberRhs = b.parseListMatchPatternMemberRhs()
		if listMatchPatternMember.Kind() == common.REST_MATCH_PATTERN {
			isEndOfFields = true
			break
		}
		if listMatchPatternMemberRhs != nil {
			matchPatternList = append(matchPatternList, listMatchPatternMemberRhs)
		} else {
			break
		}
	}
	for isEndOfFields && (listMatchPatternMemberRhs != nil) {
		b.updateLastNodeInListWithInvalidNode(matchPatternList, listMatchPatternMemberRhs, nil)
		if b.peek().Kind() == common.CLOSE_BRACKET_TOKEN {
			break
		}
		invalidField := b.parseListMatchPatternMember()
		b.updateLastNodeInListWithInvalidNode(matchPatternList, invalidField,
			&common.ERROR_MATCH_PATTERN_AFTER_REST_MATCH_PATTERN)
		listMatchPatternMemberRhs = b.parseListMatchPatternMemberRhs()
	}
	matchPatternListNode := tree.CreateNodeList(matchPatternList...)
	closeBracketToken := b.parseCloseBracket()
	b.endContext()
	return tree.CreateListMatchPatternNode(openBracketToken, matchPatternListNode, closeBracketToken)
}

func (b *BallerinaParser) IsEndOfListMatchPattern() bool {
	switch b.peek().Kind() {
	case common.CLOSE_BRACKET_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseListMatchPatternMember() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.ELLIPSIS_TOKEN:
		return b.parseRestMatchPattern()
	default:
		return b.parseMatchPattern()
	}
}

func (b *BallerinaParser) parseRestMatchPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN)
	ellipsisToken := b.parseEllipsis()
	varKeywordToken := b.parseVarKeyword()
	variableName := b.parseVariableName()
	b.endContext()
	simpleNameReferenceNode, ok := tree.CreateSimpleNameReferenceNode(variableName).(*tree.STSimpleNameReferenceNode)
	if !ok {
		panic("expected STSimpleNameReferenceNode")
	}
	return tree.CreateRestMatchPatternNode(ellipsisToken, varKeywordToken, simpleNameReferenceNode)
}

func (b *BallerinaParser) parseListMatchPatternMemberRhs() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACKET_TOKEN, common.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS)
		return b.parseListMatchPatternMemberRhs()
	}
}

func (b *BallerinaParser) parseMappingMatchPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN)
	openBraceToken := b.parseOpenBrace()
	fieldMatchPatterns := b.parseFieldMatchPatternList()
	closeBraceToken := b.parseCloseBrace()
	b.endContext()
	return tree.CreateMappingMatchPatternNode(openBraceToken, fieldMatchPatterns, closeBraceToken)
}

func (b *BallerinaParser) parseFieldMatchPatternList() tree.STNode {
	var fieldMatchPatterns []tree.STNode
	fieldMatchPatternMember := b.parseFieldMatchPatternMember()
	if fieldMatchPatternMember == nil {
		return tree.CreateEmptyNodeList()
	}
	fieldMatchPatterns = append(fieldMatchPatterns, fieldMatchPatternMember)
	if fieldMatchPatternMember.Kind() == common.REST_MATCH_PATTERN {
		b.invalidateExtraFieldMatchPatterns(fieldMatchPatterns)
		return tree.CreateNodeList(fieldMatchPatterns...)
	}
	return b.parseFieldMatchPatternListWithPatterns(fieldMatchPatterns)
}

func (b *BallerinaParser) parseFieldMatchPatternListWithPatterns(fieldMatchPatterns []tree.STNode) tree.STNode {
	for !b.IsEndOfMappingMatchPattern() {
		fieldMatchPatternRhs := b.parseFieldMatchPatternRhs()
		if fieldMatchPatternRhs == nil {
			break
		}
		fieldMatchPatterns = append(fieldMatchPatterns, fieldMatchPatternRhs)
		fieldMatchPatternMember := b.parseFieldMatchPatternMember()
		if fieldMatchPatternMember == nil {
			fieldMatchPatternMember = b.createMissingFieldMatchPattern()
		}
		fieldMatchPatterns = append(fieldMatchPatterns, fieldMatchPatternMember)
		if fieldMatchPatternMember.Kind() == common.REST_MATCH_PATTERN {
			b.invalidateExtraFieldMatchPatterns(fieldMatchPatterns)
			break
		}
	}
	return tree.CreateNodeList(fieldMatchPatterns...)
}

func (b *BallerinaParser) createMissingFieldMatchPattern() tree.STNode {
	fieldName := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
	colon := tree.CreateMissingToken(common.COLON_TOKEN, nil)
	identifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
	matchPattern := tree.CreateSimpleNameReferenceNode(identifier)
	fieldMatchPatternMember := tree.CreateFieldMatchPatternNode(fieldName, colon, matchPattern)
	fieldMatchPatternMember = tree.AddDiagnostic(fieldMatchPatternMember,
		&common.ERROR_MISSING_FIELD_MATCH_PATTERN_MEMBER)
	return fieldMatchPatternMember
}

func (b *BallerinaParser) invalidateExtraFieldMatchPatterns(fieldMatchPatterns []tree.STNode) {
	for !b.IsEndOfMappingMatchPattern() {
		fieldMatchPatternRhs := b.parseFieldMatchPatternRhs()
		if fieldMatchPatternRhs == nil {
			break
		}
		fieldMatchPatternMember := b.parseFieldMatchPatternMember()
		if fieldMatchPatternMember == nil {
			rhsToken, ok := fieldMatchPatternRhs.(tree.STToken)
			if !ok {
				panic("invalidateExtraFieldMatchPatterns: expected STToken")
			}
			b.updateLastNodeInListWithInvalidNode(fieldMatchPatterns, fieldMatchPatternRhs,
				&common.ERROR_INVALID_TOKEN, rhsToken.Text())
		} else {
			b.updateLastNodeInListWithInvalidNode(fieldMatchPatterns, fieldMatchPatternRhs, nil)
			b.updateLastNodeInListWithInvalidNode(fieldMatchPatterns, fieldMatchPatternMember,
				&common.ERROR_MATCH_PATTERN_AFTER_REST_MATCH_PATTERN)
		}
	}
}

func (b *BallerinaParser) parseFieldMatchPatternMember() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		return b.ParseFieldMatchPattern()
	case common.ELLIPSIS_TOKEN:
		return b.parseRestMatchPattern()
	case common.CLOSE_BRACE_TOKEN, common.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START)
		return b.parseFieldMatchPatternMember()
	}
}

func (b *BallerinaParser) ParseFieldMatchPattern() tree.STNode {
	fieldNameNode := b.parseVariableName()
	colonToken := b.parseColon()
	matchPattern := b.parseMatchPattern()
	return tree.CreateFieldMatchPatternNode(fieldNameNode, colonToken, matchPattern)
}

func (b *BallerinaParser) IsEndOfMappingMatchPattern() bool {
	switch b.peek().Kind() {
	case common.CLOSE_BRACE_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseFieldMatchPatternRhs() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACE_TOKEN, common.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS)
		return b.parseFieldMatchPatternRhs()
	}
}

func (b *BallerinaParser) parseErrorMatchPatternOrConsPattern(typeRefOrConstExpr tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		errorKeyword := tree.CreateMissingTokenWithDiagnostics(common.ERROR_KEYWORD,
			common.PARSER_RULE_CONTEXT_ERROR_KEYWORD.GetErrorCode())
		b.startContext(common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN)
		return b.parseErrorMatchPatternWithErrorKeywordAndTypeRef(errorKeyword, typeRefOrConstExpr)
	default:
		if b.isMatchPatternEnd(b.peek().Kind()) {
			return typeRefOrConstExpr
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN)
		return b.parseErrorMatchPatternOrConsPattern(typeRefOrConstExpr)
	}
}

func (b *BallerinaParser) isMatchPatternEnd(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.RIGHT_DOUBLE_ARROW_TOKEN,
		common.COMMA_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.CLOSE_BRACKET_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.PIPE_TOKEN,
		common.IF_KEYWORD,
		common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseErrorMatchPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN)
	errorKeyword := b.consume()
	return b.parseErrorMatchPatternWithErrorKeyword(errorKeyword)
}

func (b *BallerinaParser) parseErrorMatchPatternWithErrorKeyword(errorKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	var typeRef tree.STNode
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		typeRef = tree.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			typeRef = b.parseTypeReference()
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS)
		return b.parseErrorMatchPatternWithErrorKeyword(errorKeyword)
	}
	return b.parseErrorMatchPatternWithErrorKeywordAndTypeRef(errorKeyword, typeRef)
}

func (b *BallerinaParser) parseErrorMatchPatternWithErrorKeywordAndTypeRef(errorKeyword tree.STNode, typeRef tree.STNode) tree.STNode {
	openParenthesisToken := b.parseOpenParenthesis()
	argListMatchPatternNode := b.parseErrorArgListMatchPatterns()
	closeParenthesisToken := b.parseCloseParenthesis()
	b.endContext()
	return tree.CreateErrorMatchPatternNode(errorKeyword, typeRef, openParenthesisToken,
		argListMatchPatternNode, closeParenthesisToken)
}

func (b *BallerinaParser) parseErrorArgListMatchPatterns() tree.STNode {
	var argListMatchPatterns []tree.STNode
	if b.isEndOfErrorFieldMatchPatterns() {
		return tree.CreateNodeList(argListMatchPatterns...)
	}
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG)
	firstArg := b.parseErrorArgListMatchPattern(common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START)
	b.endContext()
	if b.isSimpleMatchPattern(firstArg.Kind()) {
		argListMatchPatterns = append(argListMatchPatterns, firstArg)
		argEnd := b.parseErrorArgListMatchPatternEnd(common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END)
		if argEnd != nil {
			secondArg := b.parseErrorArgListMatchPattern(common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS)
			if b.isValidSecondArgMatchPattern(secondArg.Kind()) {
				argListMatchPatterns = append(argListMatchPatterns, argEnd)
				argListMatchPatterns = append(argListMatchPatterns, secondArg)
			} else {
				b.updateLastNodeInListWithInvalidNode(argListMatchPatterns, argEnd, nil)
				b.updateLastNodeInListWithInvalidNode(argListMatchPatterns, secondArg,
					&common.ERROR_MATCH_PATTERN_NOT_ALLOWED)
			}
		}
	} else {
		if (firstArg.Kind() != common.NAMED_ARG_MATCH_PATTERN) && (firstArg.Kind() != common.REST_MATCH_PATTERN) {
			b.addInvalidNodeToNextToken(firstArg, &common.ERROR_MATCH_PATTERN_NOT_ALLOWED)
		} else {
			argListMatchPatterns = append(argListMatchPatterns, firstArg)
		}
	}
	argListMatchPatterns = b.parseErrorFieldMatchPatterns(argListMatchPatterns)
	return tree.CreateNodeList(argListMatchPatterns...)
}

func (b *BallerinaParser) isSimpleMatchPattern(matchPatternKind common.SyntaxKind) bool {
	switch matchPatternKind {
	case common.IDENTIFIER_TOKEN,
		common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE,
		common.NUMERIC_LITERAL,
		common.STRING_LITERAL,
		common.NULL_LITERAL,
		common.NIL_LITERAL,
		common.BOOLEAN_LITERAL,
		common.TYPED_BINDING_PATTERN,
		common.UNARY_EXPRESSION:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isValidSecondArgMatchPattern(syntaxKind common.SyntaxKind) bool {
	switch syntaxKind {
	case common.ERROR_MATCH_PATTERN,
		common.NAMED_ARG_MATCH_PATTERN,
		common.REST_MATCH_PATTERN:
		return true
	default:
		return b.isSimpleMatchPattern(syntaxKind)
	}
}

// Return modified argListMatchPatterns
func (b *BallerinaParser) parseErrorFieldMatchPatterns(argListMatchPatterns []tree.STNode) []tree.STNode {
	lastValidArgKind := common.NAMED_ARG_MATCH_PATTERN
	for !b.isEndOfErrorFieldMatchPatterns() {
		argEnd := b.parseErrorArgListMatchPatternEnd(common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS)
		if argEnd == nil {
			break
		}
		currentArg := b.parseErrorArgListMatchPattern(common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN)
		errorCode := b.validateErrorFieldMatchPatternOrder(lastValidArgKind, currentArg.Kind())
		if errorCode == nil {
			argListMatchPatterns = append(argListMatchPatterns, argEnd)
			argListMatchPatterns = append(argListMatchPatterns, currentArg)
			lastValidArgKind = currentArg.Kind()
		} else if len(argListMatchPatterns) == 0 {
			b.addInvalidNodeToNextToken(argEnd, nil)
			b.addInvalidNodeToNextToken(currentArg, errorCode)
		} else {
			argListMatchPatterns = b.updateLastNodeInListWithInvalidNode(argListMatchPatterns, argEnd, nil)
			argListMatchPatterns = b.updateLastNodeInListWithInvalidNode(argListMatchPatterns, currentArg, errorCode)
		}
	}
	return argListMatchPatterns
}

func (b *BallerinaParser) isEndOfErrorFieldMatchPatterns() bool {
	return b.isEndOfErrorFieldBindingPatterns()
}

func (b *BallerinaParser) parseErrorArgListMatchPatternEnd(currentCtx common.ParserRuleContext) tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.consume()
	case common.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), currentCtx)
		return b.parseErrorArgListMatchPatternEnd(currentCtx)
	}
}

func (b *BallerinaParser) parseErrorArgListMatchPattern(context common.ParserRuleContext) tree.STNode {
	nextToken := b.peek()
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		return b.parseNamedArgOrSimpleMatchPattern()
	}
	switch nextToken.Kind() {
	case common.ELLIPSIS_TOKEN:
		return b.parseRestMatchPattern()
	case common.OPEN_PAREN_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.OPEN_BRACKET_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.ERROR_KEYWORD:
		return b.parseMatchPattern()
	case common.VAR_KEYWORD:
		varType := CreateBuiltinSimpleNameReference(b.consume())
		variableName := b.createCaptureOrWildcardBP(b.parseVariableName())
		return tree.CreateTypedBindingPatternNode(varType, variableName)
	case common.CLOSE_PAREN_TOKEN:
		return tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_MATCH_PATTERN)
	default:
		b.recoverWithBlockContext(nextToken, context)
		return b.parseErrorArgListMatchPattern(context)
	}
}

func (b *BallerinaParser) parseNamedArgOrSimpleMatchPattern() tree.STNode {
	constRefExpr := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_MATCH_PATTERN)
	if (constRefExpr.Kind() == common.QUALIFIED_NAME_REFERENCE) || (b.peek().Kind() != common.EQUAL_TOKEN) {
		return constRefExpr
	}
	simpleNameNode, ok := constRefExpr.(*tree.STSimpleNameReferenceNode)
	if !ok {
		panic("parseNamedArgOrSimpleMatchPattern: expected STSimpleNameReferenceNode")
	}
	return b.parseNamedArgMatchPattern(simpleNameNode.Name)
}

func (b *BallerinaParser) parseNamedArgMatchPattern(identifier tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN)
	equalToken := b.parseAssignOp()
	matchPattern := b.parseMatchPattern()
	b.endContext()
	return tree.CreateNamedArgMatchPatternNode(identifier, equalToken, matchPattern)
}

func (b *BallerinaParser) validateErrorFieldMatchPatternOrder(prevArgKind common.SyntaxKind, currentArgKind common.SyntaxKind) *common.DiagnosticErrorCode {
	switch currentArgKind {
	case common.NAMED_ARG_MATCH_PATTERN,
		common.REST_MATCH_PATTERN:
		if prevArgKind == common.REST_MATCH_PATTERN {
			return &common.ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG
		}
		return nil
	default:
		return &common.ERROR_MATCH_PATTERN_NOT_ALLOWED
	}
}

func (b *BallerinaParser) parseMarkdownDocumentation() tree.STNode {
	markdownDocLineList := make([]tree.STNode, 0)
	nextToken := b.peek()
	for nextToken.Kind() == common.DOCUMENTATION_STRING {
		documentationString := b.consume()
		parsedDocLines := b.parseDocumentationString(documentationString)
		markdownDocLineList = b.appendParsedDocumentationLines(markdownDocLineList, parsedDocLines)
		nextToken = b.peek()
	}
	markdownDocLines := tree.CreateNodeList(markdownDocLineList...)
	return tree.CreateMarkdownDocumentationNode(markdownDocLines)
}

func (b *BallerinaParser) parseDocumentationString(documentationStringToken tree.STToken) tree.STNode {
	leadingTriviaList := b.getLeadingTriviaList(documentationStringToken.LeadingMinutiae())
	diagnostics := documentationStringToken.Diagnostics()

	charReader := text.CharReaderFromText(documentationStringToken.Text())
	documentationLexer := newDocumentationLexer(charReader, leadingTriviaList, diagnostics)
	tokenReader := CreateTokenReader(documentationLexer)
	documentationParser := NewDocumentationParser(tokenReader)

	return documentationParser.Parse()
}

func (b *BallerinaParser) getLeadingTriviaList(leadingMinutiaeNode tree.STNode) []tree.STNode {
	leadingTriviaList := make([]tree.STNode, 0)
	bucketCount := leadingMinutiaeNode.BucketCount()
	i := 0
	for ; i < bucketCount; i++ {
		leadingTriviaList = append(leadingTriviaList, leadingMinutiaeNode.ChildInBucket(i))
	}
	return leadingTriviaList
}

func (b *BallerinaParser) appendParsedDocumentationLines(markdownDocLineList []tree.STNode, parsedDocLines tree.STNode) []tree.STNode {
	bucketCount := parsedDocLines.BucketCount()
	for i := range bucketCount {
		markdownDocLine := parsedDocLines.ChildInBucket(i)
		markdownDocLineList = append(markdownDocLineList, markdownDocLine)
	}
	return markdownDocLineList
}

func (b *BallerinaParser) parseStmtStartsWithTypeOrExpr(annots tree.STNode, qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	typeOrExpr := b.parseTypedBindingPatternOrExprWithQualifiers(qualifiers, true)
	return b.parseStmtStartsWithTypedBPOrExprRhs(annots, typeOrExpr)
}

func (b *BallerinaParser) parseStmtStartsWithTypedBPOrExprRhs(annots tree.STNode, typedBindingPatternOrExpr tree.STNode) tree.STNode {
	if typedBindingPatternOrExpr.Kind() == common.TYPED_BINDING_PATTERN {
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		res, _ := b.parseVarDeclRhs(annots, nil, typedBindingPatternOrExpr, false)
		return res
	}
	expr := b.getExpression(typedBindingPatternOrExpr)
	expr = b.getExpression(b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, expr, false, true))
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *BallerinaParser) parseTypedBindingPatternOrExpr(allowAssignment bool) tree.STNode {
	typeDescQualifiers := make([]tree.STNode, 0)
	return b.parseTypedBindingPatternOrExprWithQualifiers(typeDescQualifiers, allowAssignment)
}

func (b *BallerinaParser) parseTypedBindingPatternOrExprWithQualifiers(qualifiers []tree.STNode, allowAssignment bool) tree.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	var typeOrExpr tree.STNode
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME)
		return b.parseTypedBindingPatternOrExprRhs(typeOrExpr, allowAssignment)
	}
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTypedBPOrExprStartsWithOpenParenthesis()
	case common.FUNCTION_KEYWORD:
		return b.parseAnonFuncExprOrTypedBPWithFuncType(qualifiers)
	case common.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseTupleTypeDescOrListConstructor(tree.CreateEmptyNodeList())
		return b.parseTypedBindingPatternOrExprRhs(typeOrExpr, allowAssignment)
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		basicLiteral := b.parseBasicLiteral()
		return b.parseTypedBindingPatternOrExprRhs(basicLiteral, allowAssignment)
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseActionOrExpressionInLhs(tree.CreateEmptyNodeList())
		}
		return b.parseTypedBindingPatternInner(qualifiers, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	}
}

func (b *BallerinaParser) parseTypedBindingPatternOrExprRhs(typeOrExpr tree.STNode, allowAssignment bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.PIPE_TOKEN, common.BITWISE_AND_TOKEN:
		nextNextToken := b.peekN(2)
		if nextNextToken.Kind() == common.EQUAL_TOKEN {
			return typeOrExpr
		}
		pipeOrAndToken := b.parseBinaryOperator()
		rhsTypedBPOrExpr := b.parseTypedBindingPatternOrExpr(allowAssignment)
		if rhsTypedBPOrExpr.Kind() == common.TYPED_BINDING_PATTERN {
			typedBP, ok := rhsTypedBPOrExpr.(*tree.STTypedBindingPatternNode)
			if !ok {
				panic("expected STTypedBindingPatternNode")
			}
			typeOrExpr = b.getTypeDescFromExpr(typeOrExpr)
			newTypeDesc := b.mergeTypes(typeOrExpr, pipeOrAndToken, typedBP.TypeDescriptor)
			return tree.CreateTypedBindingPatternNode(newTypeDesc, typedBP.BindingPattern)
		}
		if b.peek().Kind() == common.EQUAL_TOKEN {
			return b.createCaptureBPWithMissingVarName(typeOrExpr, pipeOrAndToken, rhsTypedBPOrExpr)
		}
		return tree.CreateBinaryExpressionNode(common.BINARY_EXPRESSION, typeOrExpr,
			pipeOrAndToken, rhsTypedBPOrExpr)
	case common.SEMICOLON_TOKEN:
		if b.isExpression(typeOrExpr.Kind()) {
			return typeOrExpr
		}
		if b.isDefiniteTypeDesc(typeOrExpr.Kind()) || (!b.isAllBasicLiterals(typeOrExpr)) {
			typeDesc := b.getTypeDescFromExpr(typeOrExpr)
			return b.parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc)
		}
		return typeOrExpr
	case common.IDENTIFIER_TOKEN, common.QUESTION_MARK_TOKEN:
		if b.isAmbiguous(typeOrExpr) || b.isDefiniteTypeDesc(typeOrExpr.Kind()) {
			typeDesc := b.getTypeDescFromExpr(typeOrExpr)
			return b.parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc)
		}
		return typeOrExpr
	case common.EQUAL_TOKEN:
		return typeOrExpr
	case common.OPEN_BRACKET_TOKEN:
		return b.parseTypedBindingPatternOrMemberAccess(typeOrExpr, false, allowAssignment,
			common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	case common.OPEN_BRACE_TOKEN, common.ERROR_KEYWORD:
		typeDesc := b.getTypeDescFromExpr(typeOrExpr)
		return b.parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc)
	default:
		if b.isCompoundAssignment(nextToken.Kind()) {
			return typeOrExpr
		}
		if b.isValidExprRhsStart(nextToken.Kind(), typeOrExpr.Kind()) {
			return typeOrExpr
		}
		token := b.peek()
		typeOrExprKind := typeOrExpr.Kind()
		if (typeOrExprKind == common.QUALIFIED_NAME_REFERENCE) || (typeOrExprKind == common.SIMPLE_NAME_REFERENCE) {
			b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS)
		} else {
			b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS)
		}
		return b.parseTypedBindingPatternOrExprRhs(typeOrExpr, allowAssignment)
	}
}

func (b *BallerinaParser) createCaptureBPWithMissingVarName(lhsType tree.STNode, separatorToken tree.STNode, rhsType tree.STNode) tree.STNode {
	lhsType = b.getTypeDescFromExpr(lhsType)
	rhsType = b.getTypeDescFromExpr(rhsType)
	newTypeDesc := b.mergeTypes(lhsType, separatorToken, rhsType)
	identifier := tree.CreateMissingTokenWithDiagnosticsFromParserRules(common.IDENTIFIER_TOKEN,
		common.PARSER_RULE_CONTEXT_VARIABLE_NAME)
	captureBP := tree.CreateCaptureBindingPatternNode(identifier)
	return tree.CreateTypedBindingPatternNode(newTypeDesc, captureBP)
}

func (b *BallerinaParser) parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc tree.STNode) tree.STNode {
	typeDesc = b.parseComplexTypeDescriptor(typeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	return b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
}

func (b *BallerinaParser) parseTypedBPOrExprStartsWithOpenParenthesis() tree.STNode {
	exprOrTypeDesc := b.parseTypedDescOrExprStartsWithOpenParenthesis()
	if b.isDefiniteTypeDesc(exprOrTypeDesc.Kind()) {
		return b.parseTypeBindingPatternStartsWithAmbiguousNode(exprOrTypeDesc)
	}
	return b.parseTypedBindingPatternOrExprRhs(exprOrTypeDesc, false)
}

func (b *BallerinaParser) isDefiniteTypeDesc(kind common.SyntaxKind) bool {
	return ((kind.CompareTo(common.RECORD_TYPE_DESC) >= 0) && (kind.CompareTo(common.FUTURE_TYPE_DESC) <= 0))
}

func (b *BallerinaParser) isDefiniteExpr(kind common.SyntaxKind) bool {
	if (kind == common.QUALIFIED_NAME_REFERENCE) || (kind == common.SIMPLE_NAME_REFERENCE) {
		return false
	}
	return ((kind.CompareTo(common.BINARY_EXPRESSION) >= 0) && (kind.CompareTo(common.ERROR_CONSTRUCTOR) <= 0))
}

func (b *BallerinaParser) isDefiniteAction(kind common.SyntaxKind) bool {
	return ((kind.CompareTo(common.REMOTE_METHOD_CALL_ACTION) >= 0) && (kind.CompareTo(common.CLIENT_RESOURCE_ACCESS_ACTION) <= 0))
}

func (b *BallerinaParser) parseTypedDescOrExprStartsWithOpenParenthesis() tree.STNode {
	openParen := b.parseOpenParenthesis()
	nextToken := b.peek()
	if nextToken.Kind() == common.CLOSE_PAREN_TOKEN {
		closeParen := b.parseCloseParenthesis()
		return b.parseTypeOrExprStartWithEmptyParenthesis(openParen, closeParen)
	}
	typeOrExpr := b.parseTypeDescOrExpr()
	if b.isAction(typeOrExpr) {
		closeParen := b.parseCloseParenthesis()
		return tree.CreateBracedExpressionNode(common.BRACED_ACTION, openParen, typeOrExpr,
			closeParen)
	}
	if b.isExpression(typeOrExpr.Kind()) {
		b.startContext(common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS)
		return b.parseBracedExprOrAnonFuncParamRhs(openParen, typeOrExpr, false)
	}
	typeDescNode := b.getTypeDescFromExpr(typeOrExpr)
	typeDescNode = b.parseComplexTypeDescriptor(typeDescNode, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS, false)
	closeParen := b.parseCloseParenthesis()
	return tree.CreateParenthesisedTypeDescriptorNode(openParen, typeDescNode, closeParen)
}

func (b *BallerinaParser) parseTypeDescOrExpr() tree.STNode {
	return b.parseTypeDescOrExprWithQualifiers(nil)
}

func (b *BallerinaParser) parseTypeDescOrExprWithQualifiers(qualifiers []tree.STNode) tree.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	var typeOrExpr tree.STNode
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseTypedDescOrExprStartsWithOpenParenthesis()
	case common.FUNCTION_KEYWORD:
		typeOrExpr = b.parseAnonFuncExprOrFuncTypeDesc(qualifiers)
	case common.IDENTIFIER_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME)
		return b.parseTypeDescOrExprRhs(typeOrExpr)
	case common.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseTupleTypeDescOrListConstructor(tree.CreateEmptyNodeList())
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.NULL_KEYWORD,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		basicLiteral := b.parseBasicLiteral()
		return b.parseTypeDescOrExprRhs(basicLiteral)
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseActionOrExpressionInLhs(tree.CreateEmptyNodeList())
		}
		return b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
	}
	if b.isDefiniteTypeDesc(typeOrExpr.Kind()) {
		return b.parseComplexTypeDescriptor(typeOrExpr, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	}
	return b.parseTypeDescOrExprRhs(typeOrExpr)
}

func (b *BallerinaParser) isExpression(kind common.SyntaxKind) bool {
	switch kind {
	case common.NUMERIC_LITERAL,
		common.STRING_LITERAL_TOKEN,
		common.NIL_LITERAL,
		common.NULL_LITERAL,
		common.BOOLEAN_LITERAL:
		return true
	default:
		return ((kind.CompareTo(common.BINARY_EXPRESSION) >= 0) && (kind.CompareTo(common.ERROR_CONSTRUCTOR) <= 0))
	}
}

func (b *BallerinaParser) parseTypeOrExprStartWithEmptyParenthesis(openParen tree.STNode, closeParen tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.RIGHT_DOUBLE_ARROW_TOKEN:
		params := tree.CreateEmptyNodeList()
		anonFuncParam := tree.CreateImplicitAnonymousFunctionParameters(openParen, params, closeParen)
		return b.parseImplicitAnonFuncWithParams(anonFuncParam, false)
	default:
		return tree.CreateNilLiteralNode(openParen, closeParen)
	}
}

func (b *BallerinaParser) parseAnonFuncExprOrTypedBPWithFuncType(qualifiers []tree.STNode) tree.STNode {
	exprOrTypeDesc := b.parseAnonFuncExprOrFuncTypeDesc(qualifiers)
	if b.isAction(exprOrTypeDesc) || b.isExpression(exprOrTypeDesc.Kind()) {
		return exprOrTypeDesc
	}
	return b.parseTypedBindingPatternTypeRhs(exprOrTypeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
}

func (b *BallerinaParser) parseAnonFuncExprOrFuncTypeDesc(qualifiers []tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC)
	var qualifierList tree.STNode
	functionKeyword := b.parseFunctionKeyword()
	var funcSignature tree.STNode
	if b.peek().Kind() == common.OPEN_PAREN_TOKEN {
		funcSignature = b.parseFuncSignature(true)
		nodes := b.createFuncTypeQualNodeList(qualifiers, functionKeyword, true)
		qualifierList = nodes[0]
		functionKeyword = nodes[1]
		b.endContext()
		return b.parseAnonFuncExprOrFuncTypeDescWithComponents(qualifierList, functionKeyword, funcSignature)
	}
	funcSignature = tree.CreateEmptyNode()
	nodes := b.createFuncTypeQualNodeList(qualifiers, functionKeyword, false)
	qualifierList = nodes[0]
	functionKeyword = nodes[1]
	funcTypeDesc := tree.CreateFunctionTypeDescriptorNode(qualifierList, functionKeyword,
		funcSignature)
	if b.getCurrentContext() != common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	}
	return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
}

func (b *BallerinaParser) parseAnonFuncExprOrFuncTypeDescWithComponents(qualifierList tree.STNode, functionKeyword tree.STNode, funcSignature tree.STNode) tree.STNode {
	currentCtx := b.getCurrentContext()
	switch b.peek().Kind() {
	case common.OPEN_BRACE_TOKEN, common.RIGHT_DOUBLE_ARROW_TOKEN:
		if currentCtx != common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
			b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		}
		b.startContext(common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION)
		funcSignatureNode, ok := funcSignature.(*tree.STFunctionSignatureNode)
		if !ok {
			panic("parseAnonFuncExprOrFuncTypeDescWithComponents: expected STFunctionSignatureNode")
		}
		funcSignature = b.validateAndGetFuncParams(*funcSignatureNode)
		funcBody := b.parseAnonFuncBody(false)
		annots := tree.CreateEmptyNodeList()
		anonFunc := tree.CreateExplicitAnonymousFunctionExpressionNode(annots, qualifierList,
			functionKeyword, funcSignature, funcBody)
		return b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, anonFunc, false, true)
	case common.IDENTIFIER_TOKEN:
		fallthrough
	default:
		funcTypeDesc := tree.CreateFunctionTypeDescriptorNode(qualifierList, functionKeyword,
			funcSignature)
		if currentCtx != common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
			b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
			return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
				true)
		}
		return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
	}
}

func (b *BallerinaParser) parseTypeDescOrExprRhs(typeOrExpr tree.STNode) tree.STNode {
	nextToken := b.peek()
	var typeDesc tree.STNode
	switch nextToken.Kind() {
	case common.PIPE_TOKEN,
		common.BITWISE_AND_TOKEN:
		nextNextToken := b.peekN(2)
		if nextNextToken.Kind() == common.EQUAL_TOKEN {
			return typeOrExpr
		}
		pipeOrAndToken := b.parseBinaryOperator()
		rhsTypeDescOrExpr := b.parseTypeDescOrExpr()
		if b.isExpression(rhsTypeDescOrExpr.Kind()) {
			return tree.CreateBinaryExpressionNode(common.BINARY_EXPRESSION, typeOrExpr,
				pipeOrAndToken, rhsTypeDescOrExpr)
		}
		typeDesc = b.getTypeDescFromExpr(typeOrExpr)
		rhsTypeDescOrExpr = b.getTypeDescFromExpr(rhsTypeDescOrExpr)
		return b.mergeTypes(typeDesc, pipeOrAndToken, rhsTypeDescOrExpr)
	case common.IDENTIFIER_TOKEN,
		common.QUESTION_MARK_TOKEN:
		typeDesc = b.parseComplexTypeDescriptor(b.getTypeDescFromExpr(typeOrExpr),
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, false)
		return typeDesc
	case common.SEMICOLON_TOKEN:
		return b.getTypeDescFromExpr(typeOrExpr)
	case common.EQUAL_TOKEN, common.CLOSE_PAREN_TOKEN, common.CLOSE_BRACE_TOKEN, common.CLOSE_BRACKET_TOKEN, common.EOF_TOKEN, common.COMMA_TOKEN:
		return typeOrExpr
	case common.OPEN_BRACKET_TOKEN:
		return b.parseTypedBindingPatternOrMemberAccess(typeOrExpr, false, true,
			common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	case common.ELLIPSIS_TOKEN:
		ellipsis := b.parseEllipsis()
		typeOrExpr = b.getTypeDescFromExpr(typeOrExpr)
		return tree.CreateRestDescriptorNode(typeOrExpr, ellipsis)
	default:
		if b.isCompoundAssignment(nextToken.Kind()) {
			return typeOrExpr
		}
		if b.isValidExprRhsStart(nextToken.Kind(), typeOrExpr.Kind()) {
			return b.parseExpressionRhsInner(DEFAULT_OP_PRECEDENCE, typeOrExpr, false, false, false, false)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS)
		return b.parseTypeDescOrExprRhs(typeOrExpr)
	}
}

func (b *BallerinaParser) isAmbiguous(node tree.STNode) bool {
	switch node.Kind() {
	case common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE,
		common.NIL_LITERAL,
		common.NULL_LITERAL,
		common.NUMERIC_LITERAL,
		common.STRING_LITERAL,
		common.BOOLEAN_LITERAL,
		common.BRACKETED_LIST:
		return true
	case common.BINARY_EXPRESSION:
		binaryExpr, ok := node.(*tree.STBinaryExpressionNode)
		if !ok {
			panic("expected STBinaryExpressionNode")
		}
		if binaryExpr.Operator.Kind() != common.PIPE_TOKEN {
			return false
		}
		return (b.isAmbiguous(binaryExpr.LhsExpr) && b.isAmbiguous(binaryExpr.RhsExpr))
	case common.BRACED_EXPRESSION:
		bracedExpr, ok := node.(*tree.STBracedExpressionNode)
		if !ok {
			panic("isAmbiguous: expected STBracedExpressionNode")
		}
		return b.isAmbiguous(bracedExpr.Expression)
	case common.INDEXED_EXPRESSION:
		indexExpr, ok := node.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("expected STIndexedExpressionNode")
		}
		if !b.isAmbiguous(indexExpr.ContainerExpression) {
			return false
		}
		keys := indexExpr.KeyExpression
		i := 0
		for ; i < keys.BucketCount(); i++ {
			item := keys.ChildInBucket(i)
			if item.Kind() == common.COMMA_TOKEN {
				continue
			}
			if !b.isAmbiguous(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isAllBasicLiterals(node tree.STNode) bool {
	switch node.Kind() {
	case common.NIL_LITERAL, common.NULL_LITERAL, common.NUMERIC_LITERAL, common.STRING_LITERAL, common.BOOLEAN_LITERAL:
		return true
	case common.BINARY_EXPRESSION:
		binaryExpr, ok := node.(*tree.STBinaryExpressionNode)
		if !ok {
			panic("expected STBinaryExpressionNode")
		}
		if binaryExpr.Operator.Kind() != common.PIPE_TOKEN {
			return false
		}
		return (b.isAmbiguous(binaryExpr.LhsExpr) && b.isAmbiguous(binaryExpr.RhsExpr))
	case common.BRACED_EXPRESSION:
		bracedExpr, ok := node.(*tree.STBracedExpressionNode)
		if !ok {
			panic("isAllBasicLiterals: expected STBracedExpressionNode")
		}
		return b.isAmbiguous(bracedExpr.Expression)
	case common.BRACKETED_LIST:
		list, ok := node.(*tree.STAmbiguousCollectionNode)
		if !ok {
			panic("expected STAmbiguousCollectionNode")
		}
		for _, member := range list.Members {
			if member.Kind() == common.COMMA_TOKEN {
				continue
			}
			if !b.isAllBasicLiterals(member) {
				return false
			}
		}
		return true
	case common.UNARY_EXPRESSION:
		unaryExpr, ok := node.(*tree.STUnaryExpressionNode)
		if !ok {
			panic("expected STUnaryExpressionNode")
		}
		if (unaryExpr.UnaryOperator.Kind() != common.PLUS_TOKEN) && (unaryExpr.UnaryOperator.Kind() != common.MINUS_TOKEN) {
			return false
		}
		return b.isNumericLiteral(unaryExpr.Expression)
	default:
		return false
	}
}

func (b *BallerinaParser) isNumericLiteral(node tree.STNode) bool {
	switch node.Kind() {
	case common.NUMERIC_LITERAL:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseBindingPattern() tree.STNode {
	switch b.peek().Kind() {
	case common.OPEN_BRACKET_TOKEN:
		return b.parseListBindingPattern()
	case common.IDENTIFIER_TOKEN:
		return b.parseBindingPatternStartsWithIdentifier()
	case common.OPEN_BRACE_TOKEN:
		return b.parseMappingBindingPattern()
	case common.ERROR_KEYWORD:
		return b.parseErrorBindingPattern()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_BINDING_PATTERN)
		return b.parseBindingPattern()
	}
}

func (b *BallerinaParser) parseBindingPatternStartsWithIdentifier() tree.STNode {
	argNameOrBindingPattern := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER)
	secondToken := b.peek()
	if secondToken.Kind() == common.OPEN_PAREN_TOKEN {
		b.startContext(common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN)
		errorKeyword := tree.CreateMissingTokenWithDiagnostics(common.ERROR_KEYWORD,
			common.PARSER_RULE_CONTEXT_ERROR_KEYWORD.GetErrorCode())
		return b.parseErrorBindingPatternWithTypeRef(errorKeyword, argNameOrBindingPattern)
	}
	if argNameOrBindingPattern.Kind() != common.SIMPLE_NAME_REFERENCE {
		var identifier tree.STNode
		identifier = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		identifier = tree.CloneWithLeadingInvalidNodeMinutiae(identifier, argNameOrBindingPattern,
			&common.ERROR_FIELD_BP_INSIDE_LIST_BP)
		return tree.CreateCaptureBindingPatternNode(identifier)
	}
	simpleNameNode, ok := argNameOrBindingPattern.(*tree.STSimpleNameReferenceNode)
	if !ok {
		panic("parseBindingPatternStartsWithIdentifier: expected STSimpleNameReferenceNode")
	}
	return b.createCaptureOrWildcardBP(simpleNameNode.Name)
}

func (b *BallerinaParser) createCaptureOrWildcardBP(varName tree.STNode) tree.STNode {
	var bindingPattern tree.STNode
	if b.isWildcardBP(varName) {
		bindingPattern = b.getWildcardBindingPattern(varName)
	} else {
		bindingPattern = tree.CreateCaptureBindingPatternNode(varName)
	}
	return bindingPattern
}

func (b *BallerinaParser) parseListBindingPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN)
	openBracket := b.parseOpenBracket()
	listBindingPattern, _ := b.parseListBindingPatternWithOpenBracket(openBracket, nil)
	b.endContext()
	return listBindingPattern
}

func (b *BallerinaParser) parseListBindingPatternWithOpenBracket(openBracket tree.STNode, bindingPatternsList []tree.STNode) (tree.STNode, []tree.STNode) {
	if b.isEndOfListBindingPattern(b.peek().Kind()) && len(bindingPatternsList) == 0 {
		closeBracket := b.parseCloseBracket()
		bindingPatternsNode := tree.CreateNodeList(bindingPatternsList...)
		return tree.CreateListBindingPatternNode(openBracket, bindingPatternsNode, closeBracket), bindingPatternsList
	}
	listBindingPatternMember := b.parseListBindingPatternMember()
	bindingPatternsList = append(bindingPatternsList, listBindingPatternMember)
	listBindingPattern, bindingPatternsList := b.parseListBindingPatternWithFirstMember(openBracket, listBindingPatternMember, bindingPatternsList)
	return listBindingPattern, bindingPatternsList
}

func (b *BallerinaParser) parseListBindingPatternWithFirstMember(openBracket tree.STNode, firstMember tree.STNode, bindingPatterns []tree.STNode) (tree.STNode, []tree.STNode) {
	member := firstMember
	token := b.peek()
	var listBindingPatternRhs tree.STNode
	for (!b.isEndOfListBindingPattern(token.Kind())) && (member.Kind() != common.REST_BINDING_PATTERN) {
		listBindingPatternRhs = b.parseListBindingPatternMemberRhs()
		if listBindingPatternRhs == nil {
			break
		}
		bindingPatterns = append(bindingPatterns, listBindingPatternRhs)
		member = b.parseListBindingPatternMember()
		bindingPatterns = append(bindingPatterns, member)
		token = b.peek()
	}
	closeBracket := b.parseCloseBracket()
	bindingPatternsNode := tree.CreateNodeList(bindingPatterns...)
	return tree.CreateListBindingPatternNode(openBracket, bindingPatternsNode, closeBracket), bindingPatterns
}

func (b *BallerinaParser) parseListBindingPatternMemberRhs() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END)
		return b.parseListBindingPatternMemberRhs()
	}
}

func (b *BallerinaParser) isEndOfListBindingPattern(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.CLOSE_BRACKET_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseListBindingPatternMember() tree.STNode {
	switch b.peek().Kind() {
	case common.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	case common.OPEN_BRACKET_TOKEN,
		common.IDENTIFIER_TOKEN,
		common.OPEN_BRACE_TOKEN,
		common.ERROR_KEYWORD:
		return b.parseBindingPattern()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER)
		return b.parseListBindingPatternMember()
	}
}

func (b *BallerinaParser) parseRestBindingPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN)
	ellipsis := b.parseEllipsis()
	varName := b.parseVariableName()
	b.endContext()
	simpleNameReferenceNode, ok := tree.CreateSimpleNameReferenceNode(varName).(*tree.STSimpleNameReferenceNode)
	if !ok {
		panic("expected STSimpleNameReferenceNode")
	}
	return tree.CreateRestBindingPatternNode(ellipsis, simpleNameReferenceNode)
}

func (b *BallerinaParser) parseTypedBindingPatternWithContext(context common.ParserRuleContext) tree.STNode {
	return b.parseTypedBindingPatternInner(nil, context)
}

func (b *BallerinaParser) parseTypedBindingPatternInner(qualifiers []tree.STNode, context common.ParserRuleContext) tree.STNode {
	typeDesc := b.parseTypeDescriptorWithinContext(qualifiers,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true, false, TYPE_PRECEDENCE_DEFAULT)
	typeBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, context)
	return typeBindingPattern
}

func (b *BallerinaParser) parseMappingBindingPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN)
	openBrace := b.parseOpenBrace()
	token := b.peek()
	if b.isEndOfMappingBindingPattern(token.Kind()) {
		closeBrace := b.parseCloseBrace()
		bindingPatternsNode := tree.CreateEmptyNodeList()
		b.endContext()
		return tree.CreateMappingBindingPatternNode(openBrace, bindingPatternsNode, closeBrace)
	}
	var bindingPatterns []tree.STNode
	prevMember := b.parseMappingBindingPatternMember()
	if prevMember.Kind() != common.REST_BINDING_PATTERN {
		bindingPatterns = append(bindingPatterns, prevMember)
	}
	res, _ := b.parseMappingBindingPatternInner(openBrace, bindingPatterns, prevMember)
	return res
}

func (b *BallerinaParser) parseMappingBindingPatternInner(openBrace tree.STNode, bindingPatterns []tree.STNode, prevMember tree.STNode) (tree.STNode, []tree.STNode) {
	token := b.peek()
	var mappingBindingPatternRhs tree.STNode
	for (!b.isEndOfMappingBindingPattern(token.Kind())) && (prevMember.Kind() != common.REST_BINDING_PATTERN) {
		mappingBindingPatternRhs = b.parseMappingBindingPatternEnd()
		if mappingBindingPatternRhs == nil {
			break
		}
		bindingPatterns = append(bindingPatterns, mappingBindingPatternRhs)
		prevMember = b.parseMappingBindingPatternMember()
		if prevMember.Kind() == common.REST_BINDING_PATTERN {
			break
		}
		bindingPatterns = append(bindingPatterns, prevMember)
		token = b.peek()
	}
	if prevMember.Kind() == common.REST_BINDING_PATTERN {
		bindingPatterns = append(bindingPatterns, prevMember)
	}
	closeBrace := b.parseCloseBrace()
	bindingPatternsNode := tree.CreateNodeList(bindingPatterns...)
	b.endContext()
	return tree.CreateMappingBindingPatternNode(openBrace, bindingPatternsNode, closeBrace), bindingPatterns
}

func (b *BallerinaParser) parseMappingBindingPatternMember() tree.STNode {
	token := b.peek()
	switch token.Kind() {
	case common.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	default:
		return b.parseFieldBindingPattern()
	}
}

func (b *BallerinaParser) parseMappingBindingPatternEnd() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END)
		return b.parseMappingBindingPatternEnd()
	}
}

func (b *BallerinaParser) parseFieldBindingPattern() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME)
		simpleNameReference := tree.CreateSimpleNameReferenceNode(identifier)
		return b.parseFieldBindingPatternWithName(simpleNameReference)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME)
		return b.parseFieldBindingPattern()
	}
}

func (b *BallerinaParser) parseFieldBindingPatternWithName(simpleNameReference tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.COMMA_TOKEN, common.CLOSE_BRACE_TOKEN:
		return tree.CreateFieldBindingPatternVarnameNode(simpleNameReference)
	case common.COLON_TOKEN:
		colon := b.parseColon()
		bindingPattern := b.parseBindingPattern()
		return tree.CreateFieldBindingPatternFullNode(simpleNameReference, colon, bindingPattern)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END)
		return b.parseFieldBindingPatternWithName(simpleNameReference)
	}
}

func (b *BallerinaParser) isEndOfMappingBindingPattern(nextTokenKind common.SyntaxKind) bool {
	return ((nextTokenKind == common.CLOSE_BRACE_TOKEN) || b.isEndOfModuleLevelNode(1))
}

func (b *BallerinaParser) parseErrorTypeDescOrErrorBP(annots tree.STNode) tree.STNode {
	nextNextToken := b.peekN(2)
	switch nextNextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		return b.parseAsErrorBindingPattern()
	case common.LT_TOKEN:
		return b.parseAsErrorTypeDesc(annots)
	case common.IDENTIFIER_TOKEN:
		nextNextNextTokenKind := b.peekN(3).Kind()
		if (nextNextNextTokenKind == common.COLON_TOKEN) || (nextNextNextTokenKind == common.OPEN_PAREN_TOKEN) {
			return b.parseAsErrorBindingPattern()
		}
		fallthrough
	default:
		return b.parseAsErrorTypeDesc(annots)
	}
}

func (b *BallerinaParser) parseAsErrorBindingPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
	return b.parseAssignmentStmtRhs(b.parseErrorBindingPattern())
}

func (b *BallerinaParser) parseAsErrorTypeDesc(annots tree.STNode) tree.STNode {
	finalKeyword := tree.CreateEmptyNode()
	return b.parseVariableDecl(b.getAnnotations(annots), finalKeyword)
}

func (b *BallerinaParser) parseErrorBindingPattern() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN)
	errorKeyword := b.parseErrorKeyword()
	return b.parseErrorBindingPatternWithKeyword(errorKeyword)
}

func (b *BallerinaParser) parseErrorBindingPatternWithKeyword(errorKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	var typeRef tree.STNode
	switch nextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		typeRef = tree.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			typeRef = b.parseTypeReference()
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS)
		return b.parseErrorBindingPatternWithKeyword(errorKeyword)
	}
	return b.parseErrorBindingPatternWithTypeRef(errorKeyword, typeRef)
}

func (b *BallerinaParser) parseErrorBindingPatternWithTypeRef(errorKeyword tree.STNode, typeRef tree.STNode) tree.STNode {
	openParenthesis := b.parseOpenParenthesis()
	argListBindingPatterns := b.parseErrorArgListBindingPatterns()
	closeParenthesis := b.parseCloseParenthesis()
	b.endContext()
	return tree.CreateErrorBindingPatternNode(errorKeyword, typeRef, openParenthesis,
		argListBindingPatterns, closeParenthesis)
}

func (b *BallerinaParser) parseErrorArgListBindingPatterns() tree.STNode {
	var argListBindingPatterns []tree.STNode
	if b.isEndOfErrorFieldBindingPatterns() {
		return tree.CreateNodeList(argListBindingPatterns...)
	}
	return b.parseErrorArgListBindingPatternsWithList(argListBindingPatterns)
}

func (b *BallerinaParser) parseErrorArgListBindingPatternsWithList(argListBindingPatterns []tree.STNode) tree.STNode {
	firstArg := b.parseErrorArgListBindingPattern(common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START, true)
	if firstArg == nil {
		return tree.CreateNodeList(argListBindingPatterns...)
	}
	switch firstArg.Kind() {
	case common.CAPTURE_BINDING_PATTERN, common.WILDCARD_BINDING_PATTERN:
		argListBindingPatterns = append(argListBindingPatterns, firstArg)
		return b.parseErrorArgListBPWithoutErrorMsg(argListBindingPatterns)
	case common.ERROR_BINDING_PATTERN:
		missingIdentifier := tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
		missingErrorMsgBP := tree.CreateCaptureBindingPatternNode(missingIdentifier)
		missingErrorMsgBP = tree.AddDiagnostic(missingErrorMsgBP,
			&common.ERROR_MISSING_ERROR_MESSAGE_BINDING_PATTERN)
		missingComma := tree.CreateMissingTokenWithDiagnostics(common.COMMA_TOKEN,
			&common.ERROR_MISSING_COMMA_TOKEN)
		argListBindingPatterns = append(argListBindingPatterns, missingErrorMsgBP)
		argListBindingPatterns = append(argListBindingPatterns, missingComma)
		argListBindingPatterns = append(argListBindingPatterns, firstArg)
		return b.parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns, firstArg.Kind())
	case common.NAMED_ARG_BINDING_PATTERN, common.REST_BINDING_PATTERN:
		argListBindingPatterns = append(argListBindingPatterns, firstArg)
		return b.parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns, firstArg.Kind())
	default:
		b.addInvalidNodeToNextToken(firstArg, &common.ERROR_BINDING_PATTERN_NOT_ALLOWED)
		return b.parseErrorArgListBindingPatternsWithList(argListBindingPatterns)
	}
}

func (b *BallerinaParser) parseErrorArgListBPWithoutErrorMsg(argListBindingPatterns []tree.STNode) tree.STNode {
	argEnd := b.parseErrorArgsBindingPatternEnd(common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END)
	if argEnd == nil {
		// null marks the end of args
		return tree.CreateNodeList(argListBindingPatterns...)
	}
	secondArg := b.parseErrorArgListBindingPattern(common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS, false)
	if secondArg == nil { // depending on the recovery context we will not get null here
		panic("assertion failed")
	}
	switch secondArg.Kind() {
	case common.CAPTURE_BINDING_PATTERN, common.WILDCARD_BINDING_PATTERN, common.ERROR_BINDING_PATTERN, common.REST_BINDING_PATTERN, common.NAMED_ARG_BINDING_PATTERN:
		argListBindingPatterns = append(argListBindingPatterns, argEnd)
		argListBindingPatterns = append(argListBindingPatterns, secondArg)
		return b.parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns, secondArg.Kind())
	default:
		// we reach here for list and mapping binding patterns
		// mark them as invalid and re-parse the second arg.
		b.updateLastNodeInListWithInvalidNode(argListBindingPatterns, argEnd, nil)
		b.updateLastNodeInListWithInvalidNode(argListBindingPatterns, secondArg,
			&common.ERROR_BINDING_PATTERN_NOT_ALLOWED)
		return b.parseErrorArgListBPWithoutErrorMsg(argListBindingPatterns)
	}
}

func (b *BallerinaParser) parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns []tree.STNode, lastValidArgKind common.SyntaxKind) tree.STNode {
	for !b.isEndOfErrorFieldBindingPatterns() {
		argEnd := b.parseErrorArgsBindingPatternEnd(common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END)
		if argEnd == nil {
			// null marks the end of args
			break
		}
		currentArg := b.parseErrorArgListBindingPattern(common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN, false)
		if currentArg == nil { // depending on the recovery context we will not get null here
			panic("assertion failed")
		}
		errorCode := b.validateErrorFieldBindingPatternOrder(lastValidArgKind, currentArg.Kind())
		if errorCode == nil {
			argListBindingPatterns = append(argListBindingPatterns, argEnd)
			argListBindingPatterns = append(argListBindingPatterns, currentArg)
			lastValidArgKind = currentArg.Kind()
		} else if len(argListBindingPatterns) == 0 {
			b.addInvalidNodeToNextToken(argEnd, nil)
			b.addInvalidNodeToNextToken(currentArg, errorCode)
		} else {
			b.updateLastNodeInListWithInvalidNode(argListBindingPatterns, argEnd, nil)
			b.updateLastNodeInListWithInvalidNode(argListBindingPatterns, currentArg, errorCode)
		}
	}
	return tree.CreateNodeList(argListBindingPatterns...)
}

func (b *BallerinaParser) isEndOfErrorFieldBindingPatterns() bool {
	nextTokenKind := b.peek().Kind()
	switch nextTokenKind {
	case common.CLOSE_PAREN_TOKEN, common.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseErrorArgsBindingPatternEnd(currentCtx common.ParserRuleContext) tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), currentCtx)
		return b.parseErrorArgsBindingPatternEnd(currentCtx)
	}
}

func (b *BallerinaParser) parseErrorArgListBindingPattern(context common.ParserRuleContext, isFirstArg bool) tree.STNode {
	switch b.peek().Kind() {
	case common.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	case common.IDENTIFIER_TOKEN:
		argNameOrSimpleBindingPattern := b.consume()
		return b.parseNamedOrSimpleArgBindingPattern(argNameOrSimpleBindingPattern)
	case common.OPEN_BRACKET_TOKEN, common.OPEN_BRACE_TOKEN, common.ERROR_KEYWORD:
		return b.parseBindingPattern()
	case common.CLOSE_PAREN_TOKEN:
		if isFirstArg {
			return nil
		}
		fallthrough
	default:
		b.recoverWithBlockContext(b.peek(), context)
		return b.parseErrorArgListBindingPattern(context, isFirstArg)
	}
}

func (b *BallerinaParser) parseNamedOrSimpleArgBindingPattern(argNameOrSimpleBindingPattern tree.STNode) tree.STNode {
	secondToken := b.peek()
	switch secondToken.Kind() {
	case common.EQUAL_TOKEN:
		equal := b.consume()
		bindingPattern := b.parseBindingPattern()
		return tree.CreateNamedArgBindingPatternNode(argNameOrSimpleBindingPattern,
			equal, bindingPattern)
	case common.COMMA_TOKEN, common.CLOSE_PAREN_TOKEN:
		fallthrough
	default:
		return b.createCaptureOrWildcardBP(argNameOrSimpleBindingPattern)
	}
}

func (b *BallerinaParser) validateErrorFieldBindingPatternOrder(prevArgKind common.SyntaxKind, currentArgKind common.SyntaxKind) *common.DiagnosticErrorCode {
	switch currentArgKind {
	case common.NAMED_ARG_BINDING_PATTERN,
		common.REST_BINDING_PATTERN:
		if prevArgKind == common.REST_BINDING_PATTERN {
			return &common.ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG
		}
		return nil
	default:
		return &common.ERROR_BINDING_PATTERN_NOT_ALLOWED
	}
}

func (b *BallerinaParser) parseTypedBindingPatternTypeRhs(typeDesc tree.STNode, context common.ParserRuleContext) tree.STNode {
	return b.parseTypedBindingPatternTypeRhsWithRoot(typeDesc, context, true)
}

func (b *BallerinaParser) parseTypedBindingPatternTypeRhsWithRoot(typeDesc tree.STNode, context common.ParserRuleContext, isRoot bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN, common.OPEN_BRACE_TOKEN, common.ERROR_KEYWORD:
		bindingPattern := b.parseBindingPattern()
		return tree.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
	case common.OPEN_BRACKET_TOKEN:
		typedBindingPattern := b.parseTypedBindingPatternOrMemberAccess(typeDesc, true, true, context)
		if typedBindingPattern.Kind() != common.TYPED_BINDING_PATTERN {
			panic("assertion failed")
		}
		return typedBindingPattern
	case common.CLOSE_PAREN_TOKEN, common.COMMA_TOKEN, common.CLOSE_BRACKET_TOKEN, common.CLOSE_BRACE_TOKEN:
		if !isRoot {
			return typeDesc
		}
		fallthrough
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS)
		return b.parseTypedBindingPatternTypeRhsWithRoot(typeDesc, context, isRoot)
	}
}

func (b *BallerinaParser) parseTypedBindingPatternOrMemberAccess(typeDescOrExpr tree.STNode, isTypedBindingPattern bool, allowAssignment bool, context common.ParserRuleContext) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	if b.isBracketedListEnd(b.peek().Kind()) {
		return b.parseAsArrayTypeDesc(typeDescOrExpr, openBracket, tree.CreateEmptyNode(), context)
	}
	member := b.parseBracketedListMember(isTypedBindingPattern)
	currentNodeType := b.getBracketedListNodeType(member, isTypedBindingPattern)
	switch currentNodeType {
	case common.ARRAY_TYPE_DESC:
		typedBindingPattern := b.parseAsArrayTypeDesc(typeDescOrExpr, openBracket, member, context)
		return typedBindingPattern
	case common.LIST_BINDING_PATTERN:
		bindingPattern, _ := b.parseAsListBindingPatternWithMemberAndRoot(openBracket, nil, member, false)
		typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		return tree.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
	case common.INDEXED_EXPRESSION:
		return b.parseAsMemberAccessExpr(typeDescOrExpr, openBracket, member)
	case common.ARRAY_TYPE_DESC_OR_MEMBER_ACCESS:
		break
	case common.NONE:
		fallthrough
	default:
		memberEnd := b.parseBracketedListMemberEnd()
		if memberEnd != nil {
			var memberList []tree.STNode
			memberList = append(memberList, b.getBindingPattern(member, true))
			memberList = append(memberList, memberEnd)
			bindingPattern, memberList := b.parseAsListBindingPattern(openBracket, memberList) //nolint:staticcheck,ineffassign // memberList will be used when list binding pattern is fully implemented
			typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
			return tree.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
		}
	}
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return b.parseTypedBindingPatternOrMemberAccessRhs(typeDescOrExpr, openBracket, member, closeBracket,
		isTypedBindingPattern, allowAssignment, context)
}

func (b *BallerinaParser) parseAsMemberAccessExpr(typeNameOrExpr tree.STNode, openBracket tree.STNode, member tree.STNode) tree.STNode {
	member = b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, member, false, true)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	keyExpr := tree.CreateNodeList(member)
	memberAccessExpr := tree.CreateIndexedExpressionNode(typeNameOrExpr, openBracket, keyExpr, closeBracket)
	return b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, memberAccessExpr, false, false)
}

func (b *BallerinaParser) isBracketedListEnd(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACKET_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseBracketedListMember(isTypedBindingPattern bool) tree.STNode {
	nextToken := b.peek()

	switch nextToken.Kind() {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN, common.HEX_INTEGER_LITERAL_TOKEN, common.ASTERISK_TOKEN, common.STRING_LITERAL_TOKEN:
		return b.parseBasicLiteral()
	case common.CLOSE_BRACKET_TOKEN:
		return tree.CreateEmptyNode()
	case common.OPEN_BRACE_TOKEN, common.ERROR_KEYWORD, common.ELLIPSIS_TOKEN, common.OPEN_BRACKET_TOKEN:
		return b.parseStatementStartBracketedListMember()
	case common.IDENTIFIER_TOKEN:
		if isTypedBindingPattern {
			return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		}
	default:
		if ((!isTypedBindingPattern) && b.isValidExpressionStart(nextToken.Kind(), 1)) || b.isQualifiedIdentifierPredeclaredPrefix(nextToken.Kind()) {
			break
		}
		var recoverContext common.ParserRuleContext
		if isTypedBindingPattern {
			recoverContext = common.PARSER_RULE_CONTEXT_LIST_BINDING_MEMBER_OR_ARRAY_LENGTH
		} else {
			recoverContext = common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER
		}
		b.recoverWithBlockContext(b.peek(), recoverContext)
		return b.parseBracketedListMember(isTypedBindingPattern)
	}
	expr := b.parseExpression()
	if b.isWildcardBP(expr) {
		return b.getWildcardBindingPattern(expr)
	}

	// we don't know which one
	return expr
}

func (b *BallerinaParser) parseAsArrayTypeDesc(typeDesc tree.STNode, openBracket tree.STNode, member tree.STNode, context common.ParserRuleContext) tree.STNode {
	typeDesc = b.getTypeDescFromExpr(typeDesc)
	b.switchContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
	b.startContext(common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	b.endContext()
	return b.parseTypedBindingPatternOrMemberAccessRhs(typeDesc, openBracket, member, closeBracket, true, true,
		context)
}

func (b *BallerinaParser) parseBracketedListMemberEnd() tree.STNode {
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return b.parseComma()
	case common.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END)
		return b.parseBracketedListMemberEnd()
	}
}

func (b *BallerinaParser) parseTypedBindingPatternOrMemberAccessRhs(typeDescOrExpr tree.STNode, openBracket tree.STNode, member tree.STNode, closeBracket tree.STNode, isTypedBindingPattern bool, allowAssignment bool, context common.ParserRuleContext) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN, common.OPEN_BRACE_TOKEN, common.ERROR_KEYWORD:
		typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
		return b.parseTypedBindingPatternTypeRhs(arrayTypeDesc, context)
	case common.OPEN_BRACKET_TOKEN:
		if isTypedBindingPattern {
			typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
			arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
			return b.parseTypedBindingPatternTypeRhs(arrayTypeDesc, context)
		}
		keyExpr := b.getKeyExpr(member)
		expr := tree.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr, closeBracket)
		return b.parseTypedBindingPatternOrMemberAccess(expr, false, allowAssignment, context)
	case common.QUESTION_MARK_TOKEN:
		typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
		typeDesc = b.parseComplexTypeDescriptor(arrayTypeDesc,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
		return b.parseTypedBindingPatternTypeRhs(typeDesc, context)
	case common.PIPE_TOKEN, common.BITWISE_AND_TOKEN:
		if (!isTypedBindingPattern) && allowAssignment && (b.peekN(2).Kind() == common.EQUAL_TOKEN) && b.isValidLVExpr(typeDescOrExpr) {
			keyExpr := b.getKeyExpr(member)
			typeDescOrExpr = b.getExpression(typeDescOrExpr)
			return tree.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr, closeBracket)
		}
		return b.parseComplexTypeDescInTypedBPOrExprRhs(typeDescOrExpr, openBracket, member, closeBracket,
			isTypedBindingPattern)
	case common.IN_KEYWORD:
		if ((context != common.PARSER_RULE_CONTEXT_FOREACH_STMT) && (context != common.PARSER_RULE_CONTEXT_FROM_CLAUSE)) && (context != common.PARSER_RULE_CONTEXT_JOIN_CLAUSE) {
			break
		}
		return b.createTypedBindingPattern(typeDescOrExpr, openBracket, member, closeBracket)
	case common.EQUAL_TOKEN:
		if (context == common.PARSER_RULE_CONTEXT_FOREACH_STMT) || (context == common.PARSER_RULE_CONTEXT_FROM_CLAUSE) {
			break
		}
		if (isTypedBindingPattern || (!allowAssignment)) || (!b.isValidLVExpr(typeDescOrExpr)) {
			return b.createTypedBindingPattern(typeDescOrExpr, openBracket, member, closeBracket)
		}
		keyExpr := b.getKeyExpr(member)
		typeDescOrExpr = b.getExpression(typeDescOrExpr)
		return tree.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr, closeBracket)
	case common.SEMICOLON_TOKEN:
		if (context == common.PARSER_RULE_CONTEXT_FOREACH_STMT) || (context == common.PARSER_RULE_CONTEXT_FROM_CLAUSE) {
			break
		}
		return b.createTypedBindingPattern(typeDescOrExpr, openBracket, member, closeBracket)
	case common.CLOSE_BRACE_TOKEN, common.COMMA_TOKEN:
		if context == common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT {
			keyExpr := b.getKeyExpr(member)
			return tree.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr,
				closeBracket)
		}
		return nil
	default:
		if (!isTypedBindingPattern) && b.isValidExprRhsStart(nextToken.Kind(), closeBracket.Kind()) {
			keyExpr := b.getKeyExpr(member)
			typeDescOrExpr = b.getExpression(typeDescOrExpr)
			return tree.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr,
				closeBracket)
		}
	}
	recoveryCtx := common.PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS
	if isTypedBindingPattern {
		recoveryCtx = common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS
	}
	b.recoverWithBlockContext(b.peek(), recoveryCtx)
	return b.parseTypedBindingPatternOrMemberAccessRhs(typeDescOrExpr, openBracket, member, closeBracket,
		isTypedBindingPattern, allowAssignment, context)
}

func (b *BallerinaParser) getKeyExpr(member tree.STNode) tree.STNode {
	if member == nil {
		keyIdentifier := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR)
		missingVarRef := tree.CreateSimpleNameReferenceNode(keyIdentifier)
		return tree.CreateNodeList(missingVarRef)
	}
	return tree.CreateNodeList(member)
}

func (b *BallerinaParser) createTypedBindingPattern(typeDescOrExpr tree.STNode, openBracket tree.STNode, member tree.STNode, closeBracket tree.STNode) tree.STNode {
	bindingPatterns := tree.CreateEmptyNodeList()
	if !b.isEmpty(member) {
		memberKind := member.Kind()
		if (memberKind == common.NUMERIC_LITERAL) || (memberKind == common.ASTERISK_LITERAL) {
			typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
			arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
			identifierToken := tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
				&common.ERROR_MISSING_VARIABLE_NAME)
			variableName := tree.CreateCaptureBindingPatternNode(identifierToken)
			return tree.CreateTypedBindingPatternNode(arrayTypeDesc, variableName)
		}
		bindingPattern := b.getBindingPattern(member, true)
		bindingPatterns = tree.CreateNodeList(bindingPattern)
	}
	bindingPattern := tree.CreateListBindingPatternNode(openBracket, bindingPatterns, closeBracket)
	typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
	return tree.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
}

func (b *BallerinaParser) parseComplexTypeDescInTypedBPOrExprRhs(typeDescOrExpr tree.STNode, openBracket tree.STNode, member tree.STNode, closeBracket tree.STNode, isTypedBindingPattern bool) tree.STNode {
	pipeOrAndToken := b.parseUnionOrIntersectionToken()
	typedBindingPatternOrExpr := b.parseTypedBindingPatternOrExpr(false)
	if typedBindingPatternOrExpr.Kind() == common.TYPED_BINDING_PATTERN {
		lhsTypeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		lhsTypeDesc = b.getArrayTypeDesc(openBracket, member, closeBracket, lhsTypeDesc)
		rhsTypedBindingPattern, ok := typedBindingPatternOrExpr.(*tree.STTypedBindingPatternNode)
		if !ok {
			panic("expected *tree.STTypedBindingPatternNode")
		}
		rhsTypeDesc := rhsTypedBindingPattern.TypeDescriptor
		newTypeDesc := b.mergeTypes(lhsTypeDesc, pipeOrAndToken, rhsTypeDesc)
		return tree.CreateTypedBindingPatternNode(newTypeDesc, rhsTypedBindingPattern.BindingPattern)
	}
	if isTypedBindingPattern {
		lhsTypeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		lhsTypeDesc = b.getArrayTypeDesc(openBracket, member, closeBracket, lhsTypeDesc)
		return b.createCaptureBPWithMissingVarName(lhsTypeDesc, pipeOrAndToken, typedBindingPatternOrExpr)
	}
	keyExpr := b.getExpression(member)
	containerExpr := b.getExpression(typeDescOrExpr)
	lhsExpr := tree.CreateIndexedExpressionNode(containerExpr, openBracket, keyExpr, closeBracket)
	return tree.CreateBinaryExpressionNode(common.BINARY_EXPRESSION, lhsExpr, pipeOrAndToken,
		typedBindingPatternOrExpr)
}

func (b *BallerinaParser) mergeTypes(lhsTypeDesc tree.STNode, pipeOrAndToken tree.STNode, rhsTypeDesc tree.STNode) tree.STNode {
	if pipeOrAndToken.Kind() == common.PIPE_TOKEN {
		return b.mergeTypesWithUnion(lhsTypeDesc, pipeOrAndToken, rhsTypeDesc)
	} else {
		return b.mergeTypesWithIntersection(lhsTypeDesc, pipeOrAndToken, rhsTypeDesc)
	}
}

func (b *BallerinaParser) mergeTypesWithUnion(lhsTypeDesc tree.STNode, pipeToken tree.STNode, rhsTypeDesc tree.STNode) tree.STNode {
	if rhsTypeDesc.Kind() == common.UNION_TYPE_DESC {
		rhsUnionTypeDesc, ok := rhsTypeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		return b.replaceLeftMostUnionWithAUnion(lhsTypeDesc, pipeToken, rhsUnionTypeDesc)
	} else {
		return b.createUnionTypeDesc(lhsTypeDesc, pipeToken, rhsTypeDesc)
	}
}

func (b *BallerinaParser) mergeTypesWithIntersection(lhsTypeDesc tree.STNode, bitwiseAndToken tree.STNode, rhsTypeDesc tree.STNode) tree.STNode {
	if lhsTypeDesc.Kind() == common.UNION_TYPE_DESC {
		lhsUnionTypeDesc, ok := lhsTypeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		if rhsTypeDesc.Kind() == common.INTERSECTION_TYPE_DESC {
			rhsIntSecTypeDesc, ok := rhsTypeDesc.(*tree.STIntersectionTypeDescriptorNode)
			if !ok {
				panic("expected *tree.STIntersectionTypeDescriptorNode")
			}
			rhsTypeDesc = b.replaceLeftMostIntersectionWithAIntersection(lhsUnionTypeDesc.RightTypeDesc,
				bitwiseAndToken, rhsIntSecTypeDesc)
			return b.createUnionTypeDesc(lhsUnionTypeDesc.LeftTypeDesc, lhsUnionTypeDesc.PipeToken, rhsTypeDesc)
		} else if rhsTypeDesc.Kind() == common.UNION_TYPE_DESC {
			rhsUnionTypeDesc, ok := rhsTypeDesc.(*tree.STUnionTypeDescriptorNode)
			if !ok {
				panic("expected *tree.STUnionTypeDescriptorNode")
			}
			//nolint:staticcheck // rhsTypeDesc reassigned but not yet used in return path
			rhsTypeDesc = b.replaceLeftMostUnionWithAIntersection(lhsUnionTypeDesc.RightTypeDesc,
				bitwiseAndToken, rhsUnionTypeDesc)
			return b.replaceLeftMostUnionWithAUnion(lhsUnionTypeDesc.LeftTypeDesc,
				lhsUnionTypeDesc.PipeToken, rhsUnionTypeDesc)
		}
	}
	if rhsTypeDesc.Kind() == common.UNION_TYPE_DESC {
		rhsUnionTypeDesc, ok := rhsTypeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		return b.replaceLeftMostUnionWithAIntersection(lhsTypeDesc, bitwiseAndToken, rhsUnionTypeDesc)
	} else if rhsTypeDesc.Kind() == common.INTERSECTION_TYPE_DESC {
		rhsIntSecTypeDesc, ok := rhsTypeDesc.(*tree.STIntersectionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STIntersectionTypeDescriptorNode")
		}
		return b.replaceLeftMostIntersectionWithAIntersection(lhsTypeDesc, bitwiseAndToken, rhsIntSecTypeDesc)
	}
	return b.createIntersectionTypeDesc(lhsTypeDesc, bitwiseAndToken, rhsTypeDesc)
}

func (b *BallerinaParser) replaceLeftMostUnionWithAUnion(typeDesc tree.STNode, pipeToken tree.STNode, unionTypeDesc *tree.STUnionTypeDescriptorNode) tree.STNode {
	leftTypeDesc := unionTypeDesc.LeftTypeDesc
	if leftTypeDesc.Kind() == common.UNION_TYPE_DESC {
		leftUnionTypeDesc, ok := leftTypeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		newLeftTypeDesc := b.replaceLeftMostUnionWithAUnion(typeDesc, pipeToken, leftUnionTypeDesc)
		return tree.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	leftTypeDesc = b.createUnionTypeDesc(typeDesc, pipeToken, leftTypeDesc)
	return tree.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, leftTypeDesc)
}

func (b *BallerinaParser) replaceLeftMostUnionWithAIntersection(typeDesc tree.STNode, bitwiseAndToken tree.STNode, unionTypeDesc *tree.STUnionTypeDescriptorNode) tree.STNode {
	leftTypeDesc := unionTypeDesc.LeftTypeDesc
	if leftTypeDesc.Kind() == common.UNION_TYPE_DESC {
		leftUnionTypeDesc, ok := leftTypeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		newLeftTypeDesc := b.replaceLeftMostUnionWithAIntersection(typeDesc, bitwiseAndToken, leftUnionTypeDesc)
		return tree.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	if leftTypeDesc.Kind() == common.INTERSECTION_TYPE_DESC {
		leftIntersectionTypeDesc, ok := leftTypeDesc.(*tree.STIntersectionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STIntersectionTypeDescriptorNode")
		}
		newLeftTypeDesc := b.replaceLeftMostIntersectionWithAIntersection(typeDesc, bitwiseAndToken, leftIntersectionTypeDesc)
		return tree.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	leftTypeDesc = b.createIntersectionTypeDesc(typeDesc, bitwiseAndToken, leftTypeDesc)
	return tree.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, leftTypeDesc)
}

func (b *BallerinaParser) replaceLeftMostIntersectionWithAIntersection(typeDesc tree.STNode, bitwiseAndToken tree.STNode, intersectionTypeDesc *tree.STIntersectionTypeDescriptorNode) tree.STNode {
	leftTypeDesc := intersectionTypeDesc.LeftTypeDesc
	if leftTypeDesc.Kind() == common.INTERSECTION_TYPE_DESC {
		leftIntersectionTypeDesc, ok := leftTypeDesc.(*tree.STIntersectionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STIntersectionTypeDescriptorNode")
		}
		newLeftTypeDesc := b.replaceLeftMostIntersectionWithAIntersection(typeDesc, bitwiseAndToken, leftIntersectionTypeDesc)
		return tree.Replace(intersectionTypeDesc, intersectionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	leftTypeDesc = b.createIntersectionTypeDesc(typeDesc, bitwiseAndToken, leftTypeDesc)
	return tree.Replace(intersectionTypeDesc, intersectionTypeDesc.LeftTypeDesc, leftTypeDesc)
}

func (b *BallerinaParser) getArrayTypeDesc(openBracket tree.STNode, member tree.STNode, closeBracket tree.STNode, lhsTypeDesc tree.STNode) tree.STNode {
	if lhsTypeDesc.Kind() == common.UNION_TYPE_DESC {
		unionTypeDesc, ok := lhsTypeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		middleTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, unionTypeDesc.RightTypeDesc)
		lhsTypeDesc = b.mergeTypesWithUnion(unionTypeDesc.LeftTypeDesc, unionTypeDesc.PipeToken, middleTypeDesc)
	} else if lhsTypeDesc.Kind() == common.INTERSECTION_TYPE_DESC {
		intersectionTypeDesc, ok := lhsTypeDesc.(*tree.STIntersectionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STIntersectionTypeDescriptorNode")
		}
		middleTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, intersectionTypeDesc.RightTypeDesc)
		lhsTypeDesc = b.mergeTypesWithIntersection(intersectionTypeDesc.LeftTypeDesc,
			intersectionTypeDesc.BitwiseAndToken, middleTypeDesc)
	} else {
		lhsTypeDesc = b.createArrayTypeDesc(lhsTypeDesc, openBracket, member, closeBracket)
	}
	return lhsTypeDesc
}

func (b *BallerinaParser) parseUnionOrIntersectionToken() tree.STNode {
	token := b.peek()
	if (token.Kind() == common.PIPE_TOKEN) || (token.Kind() == common.BITWISE_AND_TOKEN) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN)
		return b.parseUnionOrIntersectionToken()
	}
}

func (b *BallerinaParser) getBracketedListNodeType(memberNode tree.STNode, isTypedBindingPattern bool) common.SyntaxKind {
	if b.isEmpty(memberNode) {
		return common.NONE
	}
	if b.isDefiniteTypeDesc(memberNode.Kind()) {
		return common.TUPLE_TYPE_DESC
	}
	switch memberNode.Kind() {
	case common.ASTERISK_LITERAL:
		return common.ARRAY_TYPE_DESC
	case common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.REST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.WILDCARD_BINDING_PATTERN:
		return common.LIST_BINDING_PATTERN
	case common.QUALIFIED_NAME_REFERENCE,
		common.REST_TYPE:
		return common.TUPLE_TYPE_DESC
	case common.NUMERIC_LITERAL:
		if isTypedBindingPattern {
			return common.ARRAY_TYPE_DESC
		}
		return common.ARRAY_TYPE_DESC_OR_MEMBER_ACCESS
	case common.SIMPLE_NAME_REFERENCE,
		common.BRACKETED_LIST,
		common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return common.NONE
	case common.ERROR_CONSTRUCTOR:
		if isTypedBindingPattern {
			return common.LIST_BINDING_PATTERN
		}
		errorCtorNode, ok := memberNode.(*tree.STErrorConstructorExpressionNode)
		if !ok {
			panic("getBracketedListNodeType: expected STErrorConstructorExpressionNode")
		}
		if b.isPossibleErrorBindingPattern(*errorCtorNode) {
			return common.NONE
		}
		return common.INDEXED_EXPRESSION
	default:
		if isTypedBindingPattern {
			return common.NONE
		}
		return common.INDEXED_EXPRESSION
	}
}

func (b *BallerinaParser) parseStatementStartsWithOpenBracket(annots tree.STNode, possibleMappingField bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT)
	return b.parseStatementStartsWithOpenBracketWithRoot(annots, true, possibleMappingField)
}

func (b *BallerinaParser) parseMemberBracketedList() tree.STNode {
	annots := tree.CreateEmptyNodeList()
	return b.parseStatementStartsWithOpenBracketWithRoot(annots, false, false)
}

func (b *BallerinaParser) parseStatementStartsWithOpenBracketWithRoot(annots tree.STNode, isRoot bool, possibleMappingField bool) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	var memberList []tree.STNode
	for !b.isBracketedListEnd(b.peek().Kind()) {
		member := b.parseStatementStartBracketedListMember()
		currentNodeType := b.getStmtStartBracketedListType(member)
		switch currentNodeType {
		case common.TUPLE_TYPE_DESC:
			member = b.parseComplexTypeDescriptor(member, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
			member = b.createMemberOrRestNode(tree.CreateEmptyNodeList(), member)
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot)
		case common.MEMBER_TYPE_DESC, common.REST_TYPE:
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot)
		case common.LIST_BINDING_PATTERN:
			res, _ := b.parseAsListBindingPatternWithMemberAndRoot(openBracket, memberList, member, isRoot)
			return res
		case common.LIST_CONSTRUCTOR:
			res, _ := b.parseAsListConstructor(openBracket, memberList, member, isRoot)
			return res
		case common.LIST_BP_OR_LIST_CONSTRUCTOR:
			res, _ := b.parseAsListBindingPatternOrListConstructor(openBracket, memberList, member, isRoot)
			return res
		case common.TUPLE_TYPE_DESC_OR_LIST_CONST:
			res, _ := b.parseAsTupleTypeDescOrListConstructor(annots, openBracket, memberList, member, isRoot)
			return res
		case common.NONE:
			fallthrough
		default:
			memberList = append(memberList, member)
		}
		memberEnd := b.parseBracketedListMemberEnd()
		if memberEnd == nil {
			break
		}
		memberList = append(memberList, memberEnd)
	}
	closeBracket := b.parseCloseBracket()
	bracketedList := b.parseStatementStartBracketedListRhs(annots, openBracket, memberList, closeBracket,
		isRoot, possibleMappingField)
	return bracketedList
}

func (b *BallerinaParser) parseStatementStartBracketedListMember() tree.STNode {
	return b.parseStatementStartBracketedListMemberWithQualifiers(nil)
}

func (b *BallerinaParser) parseStatementStartBracketedListMemberWithQualifiers(qualifiers []tree.STNode) tree.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMemberBracketedList()
	case common.IDENTIFIER_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		identifier := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		if b.isWildcardBP(identifier) {
			simpleNameNode, ok := identifier.(*tree.STSimpleNameReferenceNode)
			if !ok {
				panic("parseStatementStartBracketedListMember: expected STSimpleNameReferenceNode")
			}
			varName := simpleNameNode.Name
			return b.getWildcardBindingPattern(varName)
		}
		nextToken = b.peek()
		if nextToken.Kind() == common.ELLIPSIS_TOKEN {
			ellipsis := b.parseEllipsis()
			return tree.CreateRestDescriptorNode(identifier, ellipsis)
		}
		if (nextToken.Kind() != common.OPEN_BRACKET_TOKEN) && b.isValidTypeContinuationToken(nextToken) {
			return b.parseComplexTypeDescriptor(identifier, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
		}
		return b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, identifier, false, true)
	case common.OPEN_BRACE_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMappingBindingPatterOrMappingConstructor()
	case common.ERROR_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		nextNextToken := b.getNextNextToken()
		if (nextNextToken.Kind() == common.OPEN_PAREN_TOKEN) || (nextNextToken.Kind() == common.IDENTIFIER_TOKEN) {
			return b.parseErrorBindingPatternOrErrorConstructor()
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case common.ELLIPSIS_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRestBindingOrSpreadMember()
	case common.XML_KEYWORD, common.STRING_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		if b.getNextNextToken().Kind() == common.BACKTICK_TOKEN {
			return b.parseExpressionPossibleRhsExpr(false)
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case common.TABLE_KEYWORD, common.STREAM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		if b.getNextNextToken().Kind() == common.LT_TOKEN {
			return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		return b.parseExpressionPossibleRhsExpr(false)
	case common.OPEN_PAREN_TOKEN:
		return b.parseTypeDescOrExprWithQualifiers(qualifiers)
	case common.FUNCTION_KEYWORD:
		return b.parseAnonFuncExprOrFuncTypeDesc(qualifiers)
	case common.AT_TOKEN:
		return b.parseTupleMember()
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseExpressionPossibleRhsExpr(false)
		}
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER)
		return b.parseStatementStartBracketedListMemberWithQualifiers(qualifiers)
	}
}

func (b *BallerinaParser) parseRestBindingOrSpreadMember() tree.STNode {
	ellipsis := b.parseEllipsis()
	expr := b.parseExpression()
	if expr.Kind() == common.SIMPLE_NAME_REFERENCE {
		return tree.CreateRestBindingPatternNode(ellipsis, expr)
	} else {
		return tree.CreateSpreadMemberNode(ellipsis, expr)
	}
}

// return result and modified memberList
func (b *BallerinaParser) parseAsTupleTypeDescOrListConstructor(annots tree.STNode, openBracket tree.STNode, memberList []tree.STNode, member tree.STNode, isRoot bool) (tree.STNode, []tree.STNode) {
	memberList = append(memberList, member)
	memberEnd := b.parseBracketedListMemberEnd()
	var tupleTypeDescOrListCons tree.STNode
	if memberEnd == nil {
		closeBracket := b.parseCloseBracket()
		tupleTypeDescOrListCons = b.parseTupleTypeDescOrListConstructorRhs(openBracket, memberList, closeBracket, isRoot)
	} else {
		memberList = append(memberList, memberEnd)
		tupleTypeDescOrListCons, memberList = b.parseTupleTypeDescOrListConstructorWithBracketAndMembers(annots, openBracket, memberList, isRoot)
	}
	return tupleTypeDescOrListCons, memberList
}

func (b *BallerinaParser) parseTupleTypeDescOrListConstructor(annots tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	var memberList []tree.STNode
	result, _ := b.parseTupleTypeDescOrListConstructorWithBracketAndMembers(annots, openBracket, memberList, false)
	return result
}

func (b *BallerinaParser) parseTupleTypeDescOrListConstructorWithBracketAndMembers(annots tree.STNode, openBracket tree.STNode, memberList []tree.STNode, isRoot bool) (tree.STNode, []tree.STNode) {
	nextToken := b.peek()
	for !b.isBracketedListEnd(nextToken.Kind()) {
		member := b.parseTupleTypeDescOrListConstructorMember(annots)
		currentNodeType := b.getParsingNodeTypeOfTupleTypeOrListCons(member)
		switch currentNodeType {
		case common.LIST_CONSTRUCTOR:
			return b.parseAsListConstructor(openBracket, memberList, member, isRoot)
		case common.REST_TYPE, common.MEMBER_TYPE_DESC:
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot), memberList
		case common.TUPLE_TYPE_DESC:
			member = b.parseComplexTypeDescriptor(member, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
			member = b.createMemberOrRestNode(tree.CreateEmptyNodeList(), member)
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot), memberList
		case common.TUPLE_TYPE_DESC_OR_LIST_CONST:
			fallthrough
		default:
			memberList = append(memberList, member)
		}
		memberEnd := b.parseBracketedListMemberEnd()
		if memberEnd == nil {
			break
		}
		memberList = append(memberList, memberEnd)
		nextToken = b.peek()
	}
	closeBracket := b.parseCloseBracket()
	return b.parseTupleTypeDescOrListConstructorRhs(openBracket, memberList, closeBracket, isRoot), memberList
}

func (b *BallerinaParser) parseTupleTypeDescOrListConstructorMember(annots tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACKET_TOKEN:
		return b.parseTupleTypeDescOrListConstructor(annots)
	case common.IDENTIFIER_TOKEN:
		identifier := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		if b.peek().Kind() == common.ELLIPSIS_TOKEN {
			ellipsis := b.parseEllipsis()
			return tree.CreateRestDescriptorNode(identifier, ellipsis)
		}
		return b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, identifier, false, false)
	case common.OPEN_BRACE_TOKEN:
		return b.parseMappingConstructorExpr()
	case common.ERROR_KEYWORD:
		nextNextToken := b.getNextNextToken()
		if (nextNextToken.Kind() == common.OPEN_PAREN_TOKEN) || (nextNextToken.Kind() == common.IDENTIFIER_TOKEN) {
			return b.parseErrorConstructorExprAmbiguous(false)
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case common.XML_KEYWORD, common.STRING_KEYWORD:
		if b.getNextNextToken().Kind() == common.BACKTICK_TOKEN {
			return b.parseExpressionPossibleRhsExpr(false)
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case common.TABLE_KEYWORD, common.STREAM_KEYWORD:
		if b.getNextNextToken().Kind() == common.LT_TOKEN {
			return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		return b.parseExpressionPossibleRhsExpr(false)
	case common.OPEN_PAREN_TOKEN:
		return b.parseTypeDescOrExpr()
	case common.AT_TOKEN:
		return b.parseTupleMember()
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			return b.parseExpressionPossibleRhsExpr(false)
		}
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER)
		return b.parseTupleTypeDescOrListConstructorMember(annots)
	}
}

func (b *BallerinaParser) getParsingNodeTypeOfTupleTypeOrListCons(memberNode tree.STNode) common.SyntaxKind {
	return b.getStmtStartBracketedListType(memberNode)
}

func (b *BallerinaParser) parseTupleTypeDescOrListConstructorRhs(openBracket tree.STNode, members []tree.STNode, closeBracket tree.STNode, isRoot bool) tree.STNode {
	var tupleTypeOrListConst tree.STNode
	switch b.peek().Kind() {
	case common.COMMA_TOKEN, common.CLOSE_BRACE_TOKEN, common.CLOSE_BRACKET_TOKEN, common.PIPE_TOKEN, common.BITWISE_AND_TOKEN:
		if !isRoot {
			b.endContext()
			return tree.CreateAmbiguousCollectionNode(common.TUPLE_TYPE_DESC_OR_LIST_CONST, openBracket, members, closeBracket)
		}
	default:
		if b.isValidExprRhsStart(b.peek().Kind(), closeBracket.Kind()) || (isRoot && (b.peek().Kind() == common.EQUAL_TOKEN)) {
			members = b.getExpressionList(members, false)
			memberExpressions := tree.CreateNodeList(members...)
			tupleTypeOrListConst = tree.CreateListConstructorExpressionNode(openBracket,
				memberExpressions, closeBracket)
			break
		}
		memberTypeDescs := tree.CreateNodeList(b.getTupleMemberList(members)...)
		tupleTypeDesc := tree.CreateTupleTypeDescriptorNode(openBracket, memberTypeDescs, closeBracket)
		tupleTypeOrListConst = b.parseComplexTypeDescriptor(tupleTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
	}
	b.endContext()
	if !isRoot {
		return tupleTypeOrListConst
	}
	annots := tree.CreateEmptyNodeList()
	return b.parseStmtStartsWithTupleTypeOrExprRhs(annots, tupleTypeOrListConst, true)
}

func (b *BallerinaParser) parseStmtStartsWithTupleTypeOrExprRhs(annots tree.STNode, tupleTypeOrListConst tree.STNode, isRoot bool) tree.STNode {
	if (tupleTypeOrListConst.Kind().CompareTo(common.RECORD_TYPE_DESC) >= 0) && (tupleTypeOrListConst.Kind().CompareTo(common.TYPEDESC_TYPE_DESC) <= 0) {
		typedBindingPattern := b.parseTypedBindingPatternTypeRhsWithRoot(tupleTypeOrListConst, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, isRoot)
		if !isRoot {
			return typedBindingPattern
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		res, _ := b.parseVarDeclRhs(annots, nil, typedBindingPattern, false)
		return res
	}
	expr := b.getExpression(tupleTypeOrListConst)
	expr = b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, expr, false, true)
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *BallerinaParser) parseAsTupleTypeDesc(annots tree.STNode, openBracket tree.STNode, memberList []tree.STNode, member tree.STNode, isRoot bool) tree.STNode {
	memberList = b.getTupleMemberList(memberList)
	b.startContext(common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS)
	tupleTypeMembers, memberList := b.parseTupleTypeMembers(member, memberList) //nolint:staticcheck,ineffassign // memberList will be used when tuple rest descriptor is fully implemented
	closeBracket := b.parseCloseBracket()
	b.endContext()
	tupleType := tree.CreateTupleTypeDescriptorNode(openBracket, tupleTypeMembers, closeBracket)
	typeDesc := b.parseComplexTypeDescriptor(tupleType, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	b.endContext()
	if !isRoot {
		return typeDesc
	}
	typedBindingPattern := b.parseTypedBindingPatternTypeRhsWithRoot(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, true)
	b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	res, _ := b.parseVarDeclRhs(annots, nil, typedBindingPattern, false)
	return res
}

func (b *BallerinaParser) parseAsListBindingPatternWithMemberAndRoot(openBracket tree.STNode, memberList []tree.STNode, member tree.STNode, isRoot bool) (tree.STNode, []tree.STNode) {
	memberList = b.getBindingPatternsList(memberList, true)
	memberList = append(memberList, b.getBindingPattern(member, true))
	b.switchContext(common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN)
	listBindingPattern, memberList := b.parseListBindingPatternWithFirstMember(openBracket, member, memberList)
	b.endContext()
	if !isRoot {
		return listBindingPattern, memberList
	}
	return b.parseAssignmentStmtRhs(listBindingPattern), memberList
}

func (b *BallerinaParser) parseAsListBindingPattern(openBracket tree.STNode, memberList []tree.STNode) (tree.STNode, []tree.STNode) {
	memberList = b.getBindingPatternsList(memberList, true)
	b.switchContext(common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN)
	listBindingPattern, memberList := b.parseListBindingPatternWithOpenBracket(openBracket, memberList)
	b.endContext()
	return listBindingPattern, memberList
}

func (b *BallerinaParser) parseAsListBindingPatternOrListConstructor(openBracket tree.STNode, memberList []tree.STNode, member tree.STNode, isRoot bool) (tree.STNode, []tree.STNode) {
	memberList = append(memberList, member)
	memberEnd := b.parseBracketedListMemberEnd()
	var listBindingPatternOrListCons tree.STNode
	if memberEnd == nil {
		closeBracket := b.parseCloseBracket()
		listBindingPatternOrListCons = b.parseListBindingPatternOrListConstructorWithCloseBracket(openBracket, memberList, closeBracket, isRoot)
	} else {
		memberList = append(memberList, memberEnd)
		listBindingPatternOrListCons, memberList = b.parseListBindingPatternOrListConstructorInner(openBracket, memberList, isRoot)
	}
	return listBindingPatternOrListCons, memberList
}

func (b *BallerinaParser) getStmtStartBracketedListType(memberNode tree.STNode) common.SyntaxKind {
	if (memberNode.Kind().CompareTo(common.RECORD_TYPE_DESC) >= 0) && (memberNode.Kind().CompareTo(common.FUTURE_TYPE_DESC) <= 0) {
		return common.TUPLE_TYPE_DESC
	}
	switch memberNode.Kind() {
	case common.WILDCARD_BINDING_PATTERN,
		common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.ERROR_BINDING_PATTERN:
		return common.LIST_BINDING_PATTERN
	case common.QUALIFIED_NAME_REFERENCE:
		return common.TUPLE_TYPE_DESC
	case common.LIST_CONSTRUCTOR,
		common.MAPPING_CONSTRUCTOR,
		common.SPREAD_MEMBER:
		return common.LIST_CONSTRUCTOR
	case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		common.REST_BINDING_PATTERN:
		return common.LIST_BP_OR_LIST_CONSTRUCTOR
	case common.SIMPLE_NAME_REFERENCE, // member is a simple type-ref/var-ref
		common.BRACKETED_LIST:
		return common.NONE
	case common.ERROR_CONSTRUCTOR:
		errorCtorNode, ok := memberNode.(*tree.STErrorConstructorExpressionNode)
		if !ok {
			panic("getStmtStartBracketedListType: expected STErrorConstructorExpressionNode")
		}
		if b.isPossibleErrorBindingPattern(*errorCtorNode) {
			return common.NONE
		}
		return common.LIST_CONSTRUCTOR
	case common.INDEXED_EXPRESSION:
		return common.TUPLE_TYPE_DESC_OR_LIST_CONST
	case common.MEMBER_TYPE_DESC:
		return common.MEMBER_TYPE_DESC
	case common.REST_TYPE:
		return common.REST_TYPE
	default:
		if (b.isExpression(memberNode.Kind()) && (!b.isAllBasicLiterals(memberNode))) && (!b.isAmbiguous(memberNode)) {
			return common.LIST_CONSTRUCTOR
		}
		return common.NONE
	}
}

func (b *BallerinaParser) isPossibleErrorBindingPattern(errorConstructor tree.STErrorConstructorExpressionNode) bool {
	args := errorConstructor.Arguments
	size := args.BucketCount()
	i := 0
	for ; i < size; i++ {
		arg := args.ChildInBucket(i)
		if ((arg.Kind() != common.NAMED_ARG) && (arg.Kind() != common.POSITIONAL_ARG)) && (arg.Kind() != common.REST_ARG) {
			continue
		}
		functionArg := arg
		if !b.isPosibleArgBindingPattern(functionArg) {
			return false
		}
	}
	return true
}

func (b *BallerinaParser) isPosibleArgBindingPattern(arg tree.STFunctionArgumentNode) bool {
	switch arg.Kind() {
	case common.POSITIONAL_ARG:
		positionalArg, ok := arg.(*tree.STPositionalArgumentNode)
		if !ok {
			panic("isPosibleArgBindingPattern: expected STPositionalArgumentNode")
		}
		return b.isPosibleBindingPattern(positionalArg.Expression)
	case common.NAMED_ARG:
		namedArg, ok := arg.(*tree.STNamedArgumentNode)
		if !ok {
			panic("isPosibleArgBindingPattern: expected STNamedArgumentNode")
		}
		return b.isPosibleBindingPattern(namedArg.Expression)
	case common.REST_ARG:
		restArg, ok := arg.(*tree.STRestArgumentNode)
		if !ok {
			panic("isPosibleArgBindingPattern: expected STRestArgumentNode")
		}
		return (restArg.Expression.Kind() == common.SIMPLE_NAME_REFERENCE)
	default:
		return false
	}
}

func (b *BallerinaParser) isPosibleBindingPattern(node tree.STNode) bool {
	switch node.Kind() {
	case common.SIMPLE_NAME_REFERENCE:
		return true
	case common.LIST_CONSTRUCTOR:
		listConstructor, ok := node.(*tree.STListConstructorExpressionNode)
		if !ok {
			panic("isPosibleBindingPattern: expected STListConstructorExpressionNode")
		}
		i := 0
		for ; i < listConstructor.BucketCount(); i++ {
			expr := listConstructor.ChildInBucket(i)
			if !b.isPosibleBindingPattern(expr) {
				return false
			}
		}
		return true
	case common.MAPPING_CONSTRUCTOR:
		mappingConstructor, ok := node.(*tree.STMappingConstructorExpressionNode)
		if !ok {
			panic("isPosibleBindingPattern: expected STMappingConstructorExpressionNode")
		}
		i := 0
		for ; i < mappingConstructor.BucketCount(); i++ {
			expr := mappingConstructor.ChildInBucket(i)
			if !b.isPosibleBindingPattern(expr) {
				return false
			}
		}
		return true
	case common.SPECIFIC_FIELD:
		specificField, ok := node.(*tree.STSpecificFieldNode)
		if !ok {
			panic("isPosibleBindingPattern: expected STSpecificFieldNode")
		}
		if specificField.ReadonlyKeyword != nil {
			return false
		}
		if specificField.ValueExpr == nil {
			return true
		}
		return b.isPosibleBindingPattern(specificField.ValueExpr)
	case common.ERROR_CONSTRUCTOR:
		errorCtorNode, ok := node.(*tree.STErrorConstructorExpressionNode)
		if !ok {
			panic("isPosibleBindingPattern: expected STErrorConstructorExpressionNode")
		}
		return b.isPossibleErrorBindingPattern(*errorCtorNode)
	default:
		return false
	}
}

// return result, and modified memberList
func (b *BallerinaParser) parseStatementStartBracketedListRhs(annots tree.STNode, openBracket tree.STNode, members []tree.STNode, closeBracket tree.STNode, isRoot bool, possibleMappingField bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.EQUAL_TOKEN:
		if !isRoot {
			b.endContext()
			return tree.CreateAmbiguousCollectionNode(common.BRACKETED_LIST, openBracket, members, closeBracket)
		}
		memberBindingPatterns := tree.CreateNodeList(b.getBindingPatternsList(members, true)...)
		listBindingPattern := tree.CreateListBindingPatternNode(openBracket,
			memberBindingPatterns, closeBracket)
		b.endContext() // end tuple typ-desc
		b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
		return b.parseAssignmentStmtRhs(listBindingPattern)
	case common.IDENTIFIER_TOKEN, common.OPEN_BRACE_TOKEN:
		if !isRoot {
			b.endContext()
			return tree.CreateAmbiguousCollectionNode(common.BRACKETED_LIST, openBracket, members, closeBracket)
		}
		if len(members) == 0 {
			openBracket = tree.AddDiagnostic(openBracket, &common.ERROR_MISSING_TUPLE_MEMBER)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		b.startContext(common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS)
		memberTypeDescs := tree.CreateNodeList(b.getTupleMemberList(members)...)
		tupleTypeDesc := tree.CreateTupleTypeDescriptorNode(openBracket, memberTypeDescs, closeBracket)
		b.endContext() // end tuple typ-desc
		typeDesc := b.parseComplexTypeDescriptor(tupleTypeDesc,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
		b.endContext() // end binding pattern
		typedBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, typedBindingPattern)
	case common.OPEN_BRACKET_TOKEN:
		// [a, ..][..
		// definitely not binding pattern. Can be type-desc or list-constructor
		if !isRoot {
			// if this is a member, treat as type-desc.
			// TODO: handle expression case.
			memberTypeDescs := tree.CreateNodeList(b.getTupleMemberList(members)...)
			tupleTypeDesc := tree.CreateTupleTypeDescriptorNode(openBracket, memberTypeDescs, closeBracket)
			b.endContext()
			typeDesc := b.parseComplexTypeDescriptor(tupleTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
			return typeDesc
		}
		list := tree.CreateAmbiguousCollectionNode(common.BRACKETED_LIST, openBracket, members, closeBracket)
		b.endContext()
		tpbOrExpr := b.parseTypedBindingPatternOrExprRhs(list, true)
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, tpbOrExpr)
	case common.COLON_TOKEN: // "{[a]:" could be a computed-name-field in mapping-constructor
		if possibleMappingField && (len(members) == 1) {
			b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
			colon := b.parseColon()
			fieldNameExpr := b.getExpression(members[0])
			valueExpr := b.parseExpression()
			return tree.CreateComputedNameFieldNode(openBracket, fieldNameExpr, closeBracket, colon,
				valueExpr)
		}
		// fall through
		fallthrough
	default:
		b.endContext()
		if !isRoot {
			return tree.CreateAmbiguousCollectionNode(common.BRACKETED_LIST, openBracket, members, closeBracket)
		}
		list := tree.CreateAmbiguousCollectionNode(common.BRACKETED_LIST, openBracket, members, closeBracket)
		exprOrTPB := b.parseTypedBindingPatternOrExprRhs(list, false)
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, exprOrTPB)
	}
}

func (b *BallerinaParser) isWildcardBP(node tree.STNode) bool {
	switch node.Kind() {
	case common.SIMPLE_NAME_REFERENCE:
		simpleNameNode, ok := node.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("isWildcardBP: expected STSimpleNameReferenceNode")
		}
		nameToken, ok := simpleNameNode.Name.(tree.STToken)
		if !ok {
			panic("isWildcardBP: expected STToken")
		}
		return b.isUnderscoreToken(nameToken)
	case common.IDENTIFIER_TOKEN:
		identifierToken, ok := node.(tree.STToken)
		if !ok {
			panic("isWildcardBP: expected STToken")
		}
		return b.isUnderscoreToken(identifierToken)
	default:
		return false
	}
}

func (b *BallerinaParser) isUnderscoreToken(token tree.STToken) bool {
	return token.Text() == "_"
}

func (b *BallerinaParser) getWildcardBindingPattern(identifier tree.STNode) tree.STNode {
	var underscore tree.STNode
	switch identifier.Kind() {
	case common.SIMPLE_NAME_REFERENCE:
		simpleNameNode, ok := identifier.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("getWildcardBindingPattern: expected STSimpleNameReferenceNode")
		}
		varName := simpleNameNode.Name
		nameToken, ok := varName.(tree.STToken)
		if !ok {
			panic("getWildcardBindingPattern: expected STToken")
		}
		underscore = b.getUnderscoreKeyword(nameToken)
		return tree.CreateWildcardBindingPatternNode(underscore)
	case common.IDENTIFIER_TOKEN:
		identifierToken, ok := identifier.(tree.STToken)
		if !ok {
			panic("getWildcardBindingPattern: expected STToken")
		}
		underscore = b.getUnderscoreKeyword(identifierToken)
		return tree.CreateWildcardBindingPatternNode(underscore)
	default:
		panic("getWildcardBindingPattern: expected SIMPLE_NAME_REFERENCE or IDENTIFIER_TOKEN")
	}
}

func (b *BallerinaParser) parseStatementStartsWithOpenBrace() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	openBrace := b.parseOpenBrace()
	if b.peek().Kind() == common.CLOSE_BRACE_TOKEN {
		closeBrace := b.parseCloseBrace()
		switch b.peek().Kind() {
		case common.EQUAL_TOKEN:
			b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
			fields := tree.CreateEmptyNodeList()
			bindingPattern := tree.CreateMappingBindingPatternNode(openBrace, fields,
				closeBrace)
			return b.parseAssignmentStmtRhs(bindingPattern)
		case common.RIGHT_ARROW_TOKEN, common.SYNC_SEND_TOKEN:
			b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
			fields := tree.CreateEmptyNodeList()
			expr := tree.CreateMappingConstructorExpressionNode(openBrace, fields, closeBrace)
			expr = b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, expr, false, true)
			return b.parseStatementStartWithExprRhs(expr)
		default:
			statements := tree.CreateEmptyNodeList()
			b.endContext()
			return tree.CreateBlockStatementNode(openBrace, statements, closeBrace)
		}
	}
	member := b.parseStatementStartingBracedListFirstMember(openBrace.IsMissing())
	nodeType := b.getBracedListType(member)
	var stmt tree.STNode
	switch nodeType {
	case common.MAPPING_BINDING_PATTERN:
		return b.parseStmtAsMappingBindingPatternStart(openBrace, member)
	case common.MAPPING_CONSTRUCTOR:
		return b.parseStmtAsMappingConstructorStart(openBrace, member)
	case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return b.parseStmtAsMappingBPOrMappingConsStart(openBrace, member)
	case common.BLOCK_STATEMENT:
		closeBrace := b.parseCloseBrace()
		stmt = tree.CreateBlockStatementNode(openBrace, member, closeBrace)
		b.endContext()
		return stmt
	default:
		var stmts []tree.STNode
		stmts = append(stmts, member)
		statements, stmts := b.parseStatementsInner(stmts) //nolint:staticcheck,ineffassign // stmts will be used for error recovery
		closeBrace := b.parseCloseBrace()
		b.endContext()
		return tree.CreateBlockStatementNode(openBrace, statements, closeBrace)
	}
}

func (b *BallerinaParser) parseStmtAsMappingBindingPatternStart(openBrace tree.STNode, firstMappingField tree.STNode) tree.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN)
	var bindingPatterns []tree.STNode
	if firstMappingField.Kind() != common.REST_BINDING_PATTERN {
		bindingPatterns = append(bindingPatterns, b.getBindingPattern(firstMappingField, false))
	}
	mappingBP, _ := b.parseMappingBindingPatternInner(openBrace, bindingPatterns, firstMappingField)
	return b.parseAssignmentStmtRhs(mappingBP)
}

func (b *BallerinaParser) parseStmtAsMappingConstructorStart(openBrace tree.STNode, firstMember tree.STNode) tree.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
	mappingCons, _ := b.parseAsMappingConstructor(openBrace, nil, firstMember)
	expr := b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, mappingCons, false, true)
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *BallerinaParser) parseAsMappingConstructor(openBrace tree.STNode, members []tree.STNode, member tree.STNode) (tree.STNode, []tree.STNode) {
	members = append(members, member)
	members = b.getExpressionList(members, true)
	b.switchContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
	fields := b.finishParseMappingConstructorFields(members)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return tree.CreateMappingConstructorExpressionNode(openBrace, fields, closeBrace), members
}

func (b *BallerinaParser) parseStmtAsMappingBPOrMappingConsStart(openBrace tree.STNode, member tree.STNode) tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR)
	var members []tree.STNode
	members = append(members, member)
	var bpOrConstructor tree.STNode
	memberEnd := b.parseMappingFieldEnd()
	if memberEnd == nil {
		closeBrace := b.parseCloseBrace()
		bpOrConstructor = b.parseMappingBindingPatternOrMappingConstructorWithCloseBrace(openBrace, members, closeBrace)
	} else {
		members = append(members, memberEnd)
		bpOrConstructor, members = b.parseMappingBindingPatternOrMappingConstructor(openBrace, members) //nolint:staticcheck,ineffassign // members will be used when mapping binding pattern is fully implemented
	}
	switch bpOrConstructor.Kind() {
	case common.MAPPING_CONSTRUCTOR:
		b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		expr := b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, bpOrConstructor, false, true)
		return b.parseStatementStartWithExprRhs(expr)
	case common.MAPPING_BINDING_PATTERN:
		b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
		bindingPattern := b.getBindingPattern(bpOrConstructor, false)
		return b.parseAssignmentStmtRhs(bindingPattern)
	case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		fallthrough
	default:
		if b.peek().Kind() == common.EQUAL_TOKEN {
			b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
			bindingPattern := b.getBindingPattern(bpOrConstructor, false)
			return b.parseAssignmentStmtRhs(bindingPattern)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		expr := b.getExpression(bpOrConstructor)
		expr = b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, expr, false, true)
		return b.parseStatementStartWithExprRhs(expr)
	}
}

func (b *BallerinaParser) parseStatementStartingBracedListFirstMember(isOpenBraceMissing bool) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.READONLY_KEYWORD:
		readonlyKeyword := b.parseReadonlyKeyword()
		return b.bracedListMemberStartsWithReadonly(readonlyKeyword)
	case common.IDENTIFIER_TOKEN:
		readonlyKeyword := tree.CreateEmptyNode()
		return b.parseIdentifierRhsInStmtStartingBrace(readonlyKeyword)
	case common.STRING_LITERAL_TOKEN:
		key := b.parseStringLiteral()
		if b.peek().Kind() == common.COLON_TOKEN {
			readonlyKeyword := tree.CreateEmptyNode()
			colon := b.parseColon()
			valueExpr := b.parseExpression()
			return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
		expr := b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, key, false, true)
		return b.parseStatementStartWithExprRhs(expr)
	case common.OPEN_BRACKET_TOKEN:
		annots := tree.CreateEmptyNodeList()
		return b.parseStatementStartsWithOpenBracket(annots, true)
	case common.OPEN_BRACE_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		return b.parseStatementStartsWithOpenBrace()
	case common.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	default:
		if isOpenBraceMissing {
			readonlyKeyword := tree.CreateEmptyNode()
			return b.parseIdentifierRhsInStmtStartingBrace(readonlyKeyword)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		return b.parseStatements()
	}
}

func (b *BallerinaParser) bracedListMemberStartsWithReadonly(readonlyKeyword tree.STNode) tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.IDENTIFIER_TOKEN:
		return b.parseIdentifierRhsInStmtStartingBrace(readonlyKeyword)
	case common.STRING_LITERAL_TOKEN:
		if b.peekN(2).Kind() == common.COLON_TOKEN {
			key := b.parseStringLiteral()
			colon := b.parseColon()
			valueExpr := b.parseExpression()
			return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
		}
		fallthrough
	default:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		typeDesc := CreateBuiltinSimpleNameReference(readonlyKeyword)
		res, _ := b.parseVarDeclTypeDescRhs(typeDesc, tree.CreateEmptyNodeList(), nil,
			true, false)
		return res
	}
}

func (b *BallerinaParser) parseIdentifierRhsInStmtStartingBrace(readonlyKeyword tree.STNode) tree.STNode {
	identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
	switch b.peek().Kind() {
	case common.COMMA_TOKEN, common.CLOSE_BRACE_TOKEN:
		colon := tree.CreateEmptyNode()
		value := tree.CreateEmptyNode()
		return tree.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, value)
	case common.COLON_TOKEN:
		colon := b.parseColon()
		if !b.isEmpty(readonlyKeyword) {
			value := b.parseExpression()
			return tree.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, value)
		}
		switch b.peek().Kind() {
		case common.OPEN_BRACKET_TOKEN:
			bindingPatternOrExpr := b.parseListBindingPatternOrListConstructor()
			return b.getMappingField(identifier, colon, bindingPatternOrExpr)
		case common.OPEN_BRACE_TOKEN:
			bindingPatternOrExpr := b.parseMappingBindingPatterOrMappingConstructor()
			return b.getMappingField(identifier, colon, bindingPatternOrExpr)
		case common.ERROR_KEYWORD:
			bindingPatternOrExpr := b.parseErrorBindingPatternOrErrorConstructor()
			return b.getMappingField(identifier, colon, bindingPatternOrExpr)
		case common.IDENTIFIER_TOKEN:
			return b.parseQualifiedIdentifierRhsInStmtStartBrace(identifier, colon)
		default:
			expr := b.parseExpression()
			return b.getMappingField(identifier, colon, expr)
		}
	default:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		if !b.isEmpty(readonlyKeyword) {
			b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
			bindingPattern := tree.CreateCaptureBindingPatternNode(identifier)
			typedBindingPattern := tree.CreateTypedBindingPatternNode(readonlyKeyword, bindingPattern)
			annots := tree.CreateEmptyNodeList()
			res, _ := b.parseVarDeclRhs(annots, nil, typedBindingPattern, false)
			return res
		}
		b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
		qualifiedIdentifier := b.parseQualifiedIdentifierNode(identifier, false)
		expr := b.parseTypedBindingPatternOrExprRhs(qualifiedIdentifier, true)
		annots := tree.CreateEmptyNodeList()
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, expr)
	}
}

func (b *BallerinaParser) parseQualifiedIdentifierRhsInStmtStartBrace(identifier tree.STNode, colon tree.STNode) tree.STNode {
	secondIdentifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
	secondNameRef := tree.CreateSimpleNameReferenceNode(secondIdentifier)
	if b.isWildcardBP(secondIdentifier) {
		wildcardBP := b.getWildcardBindingPattern(secondIdentifier)
		nameRef := tree.CreateSimpleNameReferenceNode(identifier)
		return tree.CreateFieldBindingPatternFullNode(nameRef, colon, wildcardBP)
	}
	qualifiedNameRef := b.createQualifiedNameReferenceNode(identifier, colon, secondIdentifier)
	switch b.peek().Kind() {
	case common.COMMA_TOKEN:
		return tree.CreateSpecificFieldNode(tree.CreateEmptyNode(), identifier, colon,
			secondNameRef)
	case common.OPEN_BRACE_TOKEN, common.IDENTIFIER_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		typeBindingPattern := b.parseTypedBindingPatternTypeRhs(qualifiedNameRef, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		annots := tree.CreateEmptyNodeList()
		res, _ := b.parseVarDeclRhs(annots, nil, typeBindingPattern, false)
		return res
	case common.OPEN_BRACKET_TOKEN:
		return b.parseMemberRhsInStmtStartWithBrace(identifier, colon, secondIdentifier, secondNameRef)
	case common.QUESTION_MARK_TOKEN:
		typeDesc := b.parseComplexTypeDescriptor(qualifiedNameRef,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
		typeBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		annots := tree.CreateEmptyNodeList()
		res, _ := b.parseVarDeclRhs(annots, nil, typeBindingPattern, false)
		return res
	case common.EQUAL_TOKEN, common.SEMICOLON_TOKEN:
		return b.parseStatementStartWithExprRhs(qualifiedNameRef)
	case common.PIPE_TOKEN, common.BITWISE_AND_TOKEN:
		fallthrough
	default:
		return b.parseMemberWithExprInRhs(identifier, colon, secondIdentifier, secondNameRef)
	}
}

func (b *BallerinaParser) getBracedListType(member tree.STNode) common.SyntaxKind {
	switch member.Kind() {
	case common.FIELD_BINDING_PATTERN,
		common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.WILDCARD_BINDING_PATTERN:
		return common.MAPPING_BINDING_PATTERN
	case common.SPECIFIC_FIELD:
		specificFieldNode, ok := member.(*tree.STSpecificFieldNode)
		if !ok {
			panic("getBracedListType: expected STSpecificFieldNode")
		}
		expr := specificFieldNode.ValueExpr
		if expr == nil {
			return common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
		}
		switch expr.Kind() {
		case common.SIMPLE_NAME_REFERENCE,
			common.LIST_BP_OR_LIST_CONSTRUCTOR,
			common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
			return common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
		case common.ERROR_BINDING_PATTERN:
			return common.MAPPING_BINDING_PATTERN
		case common.ERROR_CONSTRUCTOR:
			errorCtorNode, ok := expr.(*tree.STErrorConstructorExpressionNode)
			if !ok {
				panic("getBracedListType: expected STErrorConstructorExpressionNode")
			}
			if b.isPossibleErrorBindingPattern(*errorCtorNode) {
				return common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
			}
			return common.MAPPING_CONSTRUCTOR
		default:
			return common.MAPPING_CONSTRUCTOR
		}
	case common.SPREAD_FIELD,
		common.COMPUTED_NAME_FIELD:
		return common.MAPPING_CONSTRUCTOR
	case common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE,
		common.LIST_BP_OR_LIST_CONSTRUCTOR,
		common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		common.REST_BINDING_PATTERN:
		return common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
	case common.LIST:
		return common.BLOCK_STATEMENT
	default:
		return common.NONE
	}
}

func (b *BallerinaParser) parseMappingBindingPatterOrMappingConstructor() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR)
	openBrace := b.parseOpenBrace()
	res, _ := b.parseMappingBindingPatternOrMappingConstructor(openBrace, nil)
	return res
}

func (b *BallerinaParser) isBracedListEnd(nextTokenKind common.SyntaxKind) bool {
	switch nextTokenKind {
	case common.EOF_TOKEN, common.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) parseMappingBindingPatternOrMappingConstructor(openBrace tree.STNode, memberList []tree.STNode) (tree.STNode, []tree.STNode) {
	nextToken := b.peek()
	for !b.isBracedListEnd(nextToken.Kind()) {
		member := b.parseMappingBindingPatterOrMappingConstructorMember()
		currentNodeType := b.getTypeOfMappingBPOrMappingCons(member)
		switch currentNodeType {
		case common.MAPPING_CONSTRUCTOR:
			return b.parseAsMappingConstructor(openBrace, memberList, member)
		case common.MAPPING_BINDING_PATTERN:
			return b.parseAsMappingBindingPattern(openBrace, memberList, member)
		case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
			fallthrough
		default:
			memberList = append(memberList, member)
		}
		memberEnd := b.parseMappingFieldEnd()
		if memberEnd == nil {
			break
		}
		memberList = append(memberList, memberEnd)
		nextToken = b.peek()
	}
	closeBrace := b.parseCloseBrace()
	return b.parseMappingBindingPatternOrMappingConstructorWithCloseBrace(openBrace, memberList, closeBrace), memberList
}

func (b *BallerinaParser) parseMappingBindingPatterOrMappingConstructorMember() tree.STNode {
	switch b.peek().Kind() {
	case common.IDENTIFIER_TOKEN:
		key := b.parseIdentifier(common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME)
		return b.parseMappingFieldRhs(key)
	case common.STRING_LITERAL_TOKEN:
		readonlyKeyword := tree.CreateEmptyNode()
		key := b.parseStringLiteral()
		colon := b.parseColon()
		valueExpr := b.parseExpression()
		return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
	case common.OPEN_BRACKET_TOKEN:
		return b.parseComputedField()
	case common.ELLIPSIS_TOKEN:
		ellipsis := b.parseEllipsis()
		expr := b.parseExpression()
		if expr.Kind() == common.SIMPLE_NAME_REFERENCE {
			return tree.CreateRestBindingPatternNode(ellipsis, expr)
		}
		return tree.CreateSpreadFieldNode(ellipsis, expr)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER)
		return b.parseMappingBindingPatterOrMappingConstructorMember()
	}
}

func (b *BallerinaParser) parseMappingFieldRhs(key tree.STNode) tree.STNode {
	var colon tree.STNode
	var valueExpr tree.STNode
	switch b.peek().Kind() {
	case common.COLON_TOKEN:
		colon = b.parseColon()
		return b.parseMappingFieldValue(key, colon)
	case common.COMMA_TOKEN, common.CLOSE_BRACE_TOKEN:
		readonlyKeyword := tree.CreateEmptyNode()
		colon = tree.CreateEmptyNode()
		valueExpr = tree.CreateEmptyNode()
		return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
	default:
		token := b.peek()
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END)
		readonlyKeyword := tree.CreateEmptyNode()
		return b.parseSpecificFieldRhs(readonlyKeyword, key)
	}
}

func (b *BallerinaParser) parseMappingFieldValue(key tree.STNode, colon tree.STNode) tree.STNode {
	var expr tree.STNode
	switch b.peek().Kind() {
	case common.IDENTIFIER_TOKEN:
		expr = b.parseExpression()
	case common.OPEN_BRACKET_TOKEN:
		expr = b.parseListBindingPatternOrListConstructor()
	case common.OPEN_BRACE_TOKEN:
		expr = b.parseMappingBindingPatterOrMappingConstructor()
	default:
		expr = b.parseExpression()
	}
	if b.isBindingPattern(expr.Kind()) {
		key = tree.CreateSimpleNameReferenceNode(key)
		return tree.CreateFieldBindingPatternFullNode(key, colon, expr)
	}
	readonlyKeyword := tree.CreateEmptyNode()
	return tree.CreateSpecificFieldNode(readonlyKeyword, key, colon, expr)
}

func (b *BallerinaParser) isBindingPattern(kind common.SyntaxKind) bool {
	switch kind {
	case common.FIELD_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.WILDCARD_BINDING_PATTERN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) getTypeOfMappingBPOrMappingCons(memberNode tree.STNode) common.SyntaxKind {
	switch memberNode.Kind() {
	case common.FIELD_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.WILDCARD_BINDING_PATTERN:
		return common.MAPPING_BINDING_PATTERN
	case common.SPECIFIC_FIELD:
		specificFieldNode, ok := memberNode.(*tree.STSpecificFieldNode)
		if !ok {
			panic("getTypeOfMappingBPOrMappingCons: expected STSpecificFieldNode")
		}
		expr := specificFieldNode.ValueExpr
		if (((expr == nil) || (expr.Kind() == common.SIMPLE_NAME_REFERENCE)) || (expr.Kind() == common.LIST_BP_OR_LIST_CONSTRUCTOR)) || (expr.Kind() == common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR) {
			return common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
		}
		return common.MAPPING_CONSTRUCTOR
	case common.SPREAD_FIELD,
		common.COMPUTED_NAME_FIELD:
		return common.MAPPING_CONSTRUCTOR
	case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR, common.SIMPLE_NAME_REFERENCE, common.QUALIFIED_NAME_REFERENCE, common.LIST_BP_OR_LIST_CONSTRUCTOR, common.REST_BINDING_PATTERN:
		fallthrough
	default:
		return common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
	}
}

func (b *BallerinaParser) parseMappingBindingPatternOrMappingConstructorWithCloseBrace(openBrace tree.STNode, members []tree.STNode, closeBrace tree.STNode) tree.STNode {
	b.endContext()
	return tree.CreateAmbiguousCollectionNode(common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR, openBrace, members, closeBrace)
}

func (b *BallerinaParser) parseAsMappingBindingPattern(openBrace tree.STNode, members []tree.STNode, member tree.STNode) (tree.STNode, []tree.STNode) {
	members = append(members, member)
	members = b.getBindingPatternsList(members, false)
	b.switchContext(common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN)
	return b.parseMappingBindingPatternInner(openBrace, members, member)
}

func (b *BallerinaParser) parseListBindingPatternOrListConstructor() tree.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	res, _ := b.parseListBindingPatternOrListConstructorInner(openBracket, nil, false)
	return res
}

// return result, and modified memberList
func (b *BallerinaParser) parseListBindingPatternOrListConstructorInner(openBracket tree.STNode, memberList []tree.STNode, isRoot bool) (tree.STNode, []tree.STNode) {
	nextToken := b.peek()
	for !b.isBracketedListEnd(nextToken.Kind()) {
		member := b.parseListBindingPatternOrListConstructorMember()
		currentNodeType := b.getParsingNodeTypeOfListBPOrListCons(member)
		switch currentNodeType {
		case common.LIST_CONSTRUCTOR:
			return b.parseAsListConstructor(openBracket, memberList, member, isRoot)
		case common.LIST_BINDING_PATTERN:
			return b.parseAsListBindingPatternWithMemberAndRoot(openBracket, memberList, member, isRoot)
		case common.LIST_BP_OR_LIST_CONSTRUCTOR:
			fallthrough
		default:
			memberList = append(memberList, member)
		}
		memberEnd := b.parseBracketedListMemberEnd()
		if memberEnd == nil {
			break
		}
		memberList = append(memberList, memberEnd)
		nextToken = b.peek()
	}
	closeBracket := b.parseCloseBracket()
	return b.parseListBindingPatternOrListConstructorWithCloseBracket(openBracket, memberList, closeBracket, isRoot), memberList
}

func (b *BallerinaParser) parseListBindingPatternOrListConstructorMember() tree.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case common.OPEN_BRACKET_TOKEN:
		return b.parseListBindingPatternOrListConstructor()
	case common.IDENTIFIER_TOKEN:
		identifier := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		if b.isWildcardBP(identifier) {
			return b.getWildcardBindingPattern(identifier)
		}
		return b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, identifier, false, false)
	case common.OPEN_BRACE_TOKEN:
		return b.parseMappingBindingPatterOrMappingConstructor()
	case common.ELLIPSIS_TOKEN:
		return b.parseRestBindingOrSpreadMember()
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			return b.parseExpression()
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER)
		return b.parseListBindingPatternOrListConstructorMember()
	}
}

func (b *BallerinaParser) getParsingNodeTypeOfListBPOrListCons(memberNode tree.STNode) common.SyntaxKind {
	switch memberNode.Kind() {
	case common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.WILDCARD_BINDING_PATTERN:
		return common.LIST_BINDING_PATTERN
	case common.SIMPLE_NAME_REFERENCE, // member is a simple type-ref/var-ref
		common.LIST_BP_OR_LIST_CONSTRUCTOR, // member is again ambiguous
		common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		common.REST_BINDING_PATTERN:
		return common.LIST_BP_OR_LIST_CONSTRUCTOR
	default:
		return common.LIST_CONSTRUCTOR
	}
}

// Return res and modified memberList
func (b *BallerinaParser) parseAsListConstructor(openBracket tree.STNode, memberList []tree.STNode, member tree.STNode, isRoot bool) (tree.STNode, []tree.STNode) {
	memberList = append(memberList, member)
	memberList = b.getExpressionList(memberList, false)
	b.switchContext(common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR)
	listMembers := b.parseListMembersInner(memberList)
	closeBracket := b.parseCloseBracket()
	listConstructor := tree.CreateListConstructorExpressionNode(openBracket, listMembers, closeBracket)
	b.endContext()
	expr := b.parseExpressionRhs(OPERATOR_PRECEDENCE_DEFAULT, listConstructor, false, true)
	if !isRoot {
		return expr, memberList
	}
	return b.parseStatementStartWithExprRhs(expr), memberList
}

func (b *BallerinaParser) parseListBindingPatternOrListConstructorWithCloseBracket(openBracket tree.STNode, members []tree.STNode, closeBracket tree.STNode, isRoot bool) tree.STNode {
	var lbpOrListCons tree.STNode
	switch b.peek().Kind() {
	case common.COMMA_TOKEN,
		common.CLOSE_BRACE_TOKEN,
		common.CLOSE_BRACKET_TOKEN:
		if !isRoot {
			b.endContext()
			return tree.CreateAmbiguousCollectionNode(common.LIST_BP_OR_LIST_CONSTRUCTOR, openBracket, members, closeBracket)
		}
		fallthrough
	default:
		nextTokenKind := b.peek().Kind()
		if b.isValidExprRhsStart(nextTokenKind, closeBracket.Kind()) || ((nextTokenKind == common.SEMICOLON_TOKEN) && isRoot) {
			members = b.getExpressionList(members, false)
			memberExpressions := tree.CreateNodeList(members...)
			lbpOrListCons = tree.CreateListConstructorExpressionNode(openBracket, memberExpressions,
				closeBracket)
			lbpOrListCons = b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, lbpOrListCons, false, true)
			break
		}
		members = b.getBindingPatternsList(members, true)
		bindingPatternsNode := tree.CreateNodeList(members...)
		lbpOrListCons = tree.CreateListBindingPatternNode(openBracket, bindingPatternsNode,
			closeBracket)
	}
	b.endContext()
	if !isRoot {
		return lbpOrListCons
	}
	if lbpOrListCons.Kind() == common.LIST_BINDING_PATTERN {
		return b.parseAssignmentStmtRhs(lbpOrListCons)
	} else {
		return b.parseStatementStartWithExprRhs(lbpOrListCons)
	}
}

func (b *BallerinaParser) parseMemberRhsInStmtStartWithBrace(identifier tree.STNode, colon tree.STNode, secondIdentifier tree.STNode, secondNameRef tree.STNode) tree.STNode {
	typedBPOrExpr := b.parseTypedBindingPatternOrMemberAccess(secondNameRef, false, true, common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	if b.isExpression(typedBPOrExpr.Kind()) {
		return b.parseMemberWithExprInRhs(identifier, colon, secondIdentifier, typedBPOrExpr)
	}
	b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	varDeclQualifiers := []tree.STNode{}
	annots := tree.CreateEmptyNodeList()
	typedBP, ok := typedBPOrExpr.(*tree.STTypedBindingPatternNode)
	if !ok {
		panic("expected STTypedBindingPatternNode")
	}
	qualifiedNameRef := b.createQualifiedNameReferenceNode(identifier, colon, secondIdentifier)
	newTypeDesc := b.mergeQualifiedNameWithTypeDesc(qualifiedNameRef, typedBP.TypeDescriptor)
	newTypeBP := tree.CreateTypedBindingPatternNode(newTypeDesc, typedBP.BindingPattern)
	publicQualifier := tree.CreateEmptyNode()
	res, _ := b.parseVarDeclRhsInner(annots, publicQualifier, varDeclQualifiers, newTypeBP, false)
	return res
}

func (b *BallerinaParser) parseMemberWithExprInRhs(identifier tree.STNode, colon tree.STNode, secondIdentifier tree.STNode, memberAccessExpr tree.STNode) tree.STNode {
	expr := b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, memberAccessExpr, false, true)
	switch b.peek().Kind() {
	case common.COMMA_TOKEN, common.CLOSE_BRACE_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		readonlyKeyword := tree.CreateEmptyNode()
		return tree.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, expr)
	case common.EQUAL_TOKEN, common.SEMICOLON_TOKEN:
		fallthrough
	default:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		qualifiedName := b.createQualifiedNameReferenceNode(identifier, colon, secondIdentifier)
		updatedExpr := b.mergeQualifiedNameWithExpr(qualifiedName, expr)
		return b.parseStatementStartWithExprRhs(updatedExpr)
	}
}

func (b *BallerinaParser) parseInferredTypeDescDefaultOrExpression() tree.STNode {
	nextToken := b.peek()
	nextTokenKind := nextToken.Kind()
	if nextTokenKind == common.LT_TOKEN {
		return b.parseInferredTypeDescDefaultOrExpressionInner(b.consume())
	}
	if b.isValidExprStart(nextTokenKind) {
		return b.parseExpression()
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START)
	return b.parseInferredTypeDescDefaultOrExpression()
}

func (b *BallerinaParser) parseInferredTypeDescDefaultOrExpressionInner(ltToken tree.STToken) tree.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == common.GT_TOKEN {
		return tree.CreateInferredTypedescDefaultNode(ltToken, b.consume())
	}
	if b.isTypeStartingToken(nextToken.Kind()) || (nextToken.Kind() == common.AT_TOKEN) {
		b.startContext(common.PARSER_RULE_CONTEXT_TYPE_CAST)
		expr := b.parseTypeCastExprInner(ltToken, true, false, false)
		return b.parseExpressionRhs(DEFAULT_OP_PRECEDENCE, expr, true, false)
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END)
	return b.parseInferredTypeDescDefaultOrExpressionInner(ltToken)
}

func (b *BallerinaParser) mergeQualifiedNameWithExpr(qualifiedName tree.STNode, exprOrAction tree.STNode) tree.STNode {
	switch exprOrAction.Kind() {
	case common.SIMPLE_NAME_REFERENCE:
		return qualifiedName
	case common.BINARY_EXPRESSION:
		binaryExpr, ok := exprOrAction.(*tree.STBinaryExpressionNode)
		if !ok {
			panic("expected STBinaryExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, binaryExpr.LhsExpr)
		return tree.CreateBinaryExpressionNode(binaryExpr.Kind(), newLhsExpr, binaryExpr.Operator,
			binaryExpr.RhsExpr)
	case common.FIELD_ACCESS:
		fieldAccess, ok := exprOrAction.(*tree.STFieldAccessExpressionNode)
		if !ok {
			panic("expected STFieldAccessExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, fieldAccess.Expression)
		return tree.CreateFieldAccessExpressionNode(newLhsExpr, fieldAccess.DotToken,
			fieldAccess.FieldName)
	case common.INDEXED_EXPRESSION:
		memberAccess, ok := exprOrAction.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("expected STIndexedExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, memberAccess.ContainerExpression)
		return tree.CreateIndexedExpressionNode(newLhsExpr, memberAccess.OpenBracket,
			memberAccess.KeyExpression, memberAccess.CloseBracket)
	case common.TYPE_TEST_EXPRESSION:
		typeTest, ok := exprOrAction.(*tree.STTypeTestExpressionNode)
		if !ok {
			panic("expected STTypeTestExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, typeTest.Expression)
		return tree.CreateTypeTestExpressionNode(newLhsExpr, typeTest.IsKeyword,
			typeTest.TypeDescriptor)
	case common.ANNOT_ACCESS:
		annotAccess, ok := exprOrAction.(*tree.STAnnotAccessExpressionNode)
		if !ok {
			panic("expected STAnnotAccessExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, annotAccess.Expression)
		return tree.CreateFieldAccessExpressionNode(newLhsExpr, annotAccess.AnnotChainingToken,
			annotAccess.AnnotTagReference)
	case common.OPTIONAL_FIELD_ACCESS:
		optionalFieldAccess, ok := exprOrAction.(*tree.STOptionalFieldAccessExpressionNode)
		if !ok {
			panic("expected STOptionalFieldAccessExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, optionalFieldAccess.Expression)
		return tree.CreateFieldAccessExpressionNode(newLhsExpr,
			optionalFieldAccess.OptionalChainingToken, optionalFieldAccess.FieldName)
	case common.CONDITIONAL_EXPRESSION:
		conditionalExpr, ok := exprOrAction.(*tree.STConditionalExpressionNode)
		if !ok {
			panic("expected STConditionalExpressionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, conditionalExpr.LhsExpression)
		return tree.CreateConditionalExpressionNode(newLhsExpr, conditionalExpr.QuestionMarkToken,
			conditionalExpr.MiddleExpression, conditionalExpr.ColonToken, conditionalExpr.EndExpression)
	case common.REMOTE_METHOD_CALL_ACTION:
		remoteCall, ok := exprOrAction.(*tree.STRemoteMethodCallActionNode)
		if !ok {
			panic("expected STRemoteMethodCallActionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, remoteCall.Expression)
		return tree.CreateRemoteMethodCallActionNode(newLhsExpr, remoteCall.RightArrowToken,
			remoteCall.MethodName, remoteCall.OpenParenToken, remoteCall.Arguments,
			remoteCall.CloseParenToken)
	case common.ASYNC_SEND_ACTION:
		asyncSend, ok := exprOrAction.(*tree.STAsyncSendActionNode)
		if !ok {
			panic("expected STAsyncSendActionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, asyncSend.Expression)
		return tree.CreateAsyncSendActionNode(newLhsExpr, asyncSend.RightArrowToken,
			asyncSend.PeerWorker)
	case common.SYNC_SEND_ACTION:
		syncSend, ok := exprOrAction.(*tree.STSyncSendActionNode)
		if !ok {
			panic("expected STSyncSendActionNode")
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, syncSend.Expression)
		return tree.CreateAsyncSendActionNode(newLhsExpr, syncSend.SyncSendToken, syncSend.PeerWorker)
	case common.FUNCTION_CALL:
		funcCall, ok := exprOrAction.(*tree.STFunctionCallExpressionNode)
		if !ok {
			panic("expected STFunctionCallExpressionNode")
		}
		return tree.CreateFunctionCallExpressionNode(qualifiedName, funcCall.OpenParenToken,
			funcCall.Arguments, funcCall.CloseParenToken)
	default:
		return exprOrAction
	}
}

func (b *BallerinaParser) mergeQualifiedNameWithTypeDesc(qualifiedName tree.STNode, typeDesc tree.STNode) tree.STNode {
	switch typeDesc.Kind() {
	case common.SIMPLE_NAME_REFERENCE:
		return qualifiedName
	case common.ARRAY_TYPE_DESC:
		arrayTypeDesc, ok := typeDesc.(*tree.STArrayTypeDescriptorNode)
		if !ok {
			panic("expected STArrayTypeDescriptorNode")
		}
		newMemberType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, arrayTypeDesc.MemberTypeDesc)
		return tree.CreateArrayTypeDescriptorNode(newMemberType, arrayTypeDesc.Dimensions)
	case common.UNION_TYPE_DESC:
		unionTypeDesc, ok := typeDesc.(*tree.STUnionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STUnionTypeDescriptorNode")
		}
		newlhsType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, unionTypeDesc.LeftTypeDesc)
		return b.mergeTypesWithUnion(newlhsType, unionTypeDesc.PipeToken, unionTypeDesc.RightTypeDesc)
	case common.INTERSECTION_TYPE_DESC:
		intersectionTypeDesc, ok := typeDesc.(*tree.STIntersectionTypeDescriptorNode)
		if !ok {
			panic("expected *tree.STIntersectionTypeDescriptorNode")
		}
		newlhsType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, intersectionTypeDesc.LeftTypeDesc)
		return b.mergeTypesWithIntersection(newlhsType, intersectionTypeDesc.BitwiseAndToken,
			intersectionTypeDesc.RightTypeDesc)
	case common.OPTIONAL_TYPE_DESC:
		optionalType, ok := typeDesc.(*tree.STOptionalTypeDescriptorNode)
		if !ok {
			panic("expected STOptionalTypeDescriptorNode")
		}
		newMemberType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, optionalType.TypeDescriptor)
		return tree.CreateOptionalTypeDescriptorNode(newMemberType, optionalType.QuestionMarkToken)
	default:
		return typeDesc
	}
}

func (b *BallerinaParser) getTupleMemberList(ambiguousList []tree.STNode) []tree.STNode {
	var tupleMemberList []tree.STNode
	for _, item := range ambiguousList {
		if item.Kind() == common.COMMA_TOKEN {
			tupleMemberList = append(tupleMemberList, item)
		} else {
			tupleMemberList = append(tupleMemberList,
				tree.CreateMemberTypeDescriptorNode(tree.CreateEmptyNodeList(),
					b.getTypeDescFromExpr(item)))
		}
	}
	return tupleMemberList
}

func (b *BallerinaParser) getTypeDescFromExpr(expression tree.STNode) tree.STNode {
	if b.isDefiniteTypeDesc(expression.Kind()) || (expression.Kind() == common.COMMA_TOKEN) {
		return expression
	}
	switch expression.Kind() {
	case common.INDEXED_EXPRESSION:
		indexedExpr, ok := expression.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("getTypeDescFromExpr: expected STIndexedExpressionNode")
		}
		return b.parseArrayTypeDescriptorNode(*indexedExpr)
	case common.NUMERIC_LITERAL,
		common.BOOLEAN_LITERAL,
		common.STRING_LITERAL,
		common.NULL_LITERAL,
		common.UNARY_EXPRESSION:
		return tree.CreateSingletonTypeDescriptorNode(expression)
	case common.TYPE_REFERENCE_TYPE_DESC:
		typeRefNode, ok := expression.(*tree.STTypeReferenceTypeDescNode)
		if !ok {
			panic("getTypeDescFromExpr: expected STTypeReferenceTypeDescNode")
		}
		return typeRefNode.TypeRef
	case common.BRACED_EXPRESSION:
		bracedExpr, ok := expression.(*tree.STBracedExpressionNode)
		if !ok {
			panic("expected STBracedExpressionNode")
		}
		typeDesc := b.getTypeDescFromExpr(bracedExpr.Expression)
		return tree.CreateParenthesisedTypeDescriptorNode(bracedExpr.OpenParen, typeDesc,
			bracedExpr.CloseParen)
	case common.NIL_LITERAL:
		nilLiteral, ok := expression.(*tree.STNilLiteralNode)
		if !ok {
			panic("expected STNilLiteralNode")
		}
		return tree.CreateNilTypeDescriptorNode(nilLiteral.OpenParenToken, nilLiteral.CloseParenToken)
	case common.BRACKETED_LIST,
		common.LIST_BP_OR_LIST_CONSTRUCTOR,
		common.TUPLE_TYPE_DESC_OR_LIST_CONST:
		innerList, ok := expression.(*tree.STAmbiguousCollectionNode)
		if !ok {
			panic("expected STAmbiguousCollectionNode")
		}
		memberTypeDescs := tree.CreateNodeList(b.getTupleMemberList(innerList.Members)...)
		return tree.CreateTupleTypeDescriptorNode(innerList.CollectionStartToken, memberTypeDescs,
			innerList.CollectionEndToken)
	case common.BINARY_EXPRESSION:
		binaryExpr, ok := expression.(*tree.STBinaryExpressionNode)
		if !ok {
			panic("expected STBinaryExpressionNode")
		}
		switch binaryExpr.Operator.Kind() {
		case common.PIPE_TOKEN,
			common.BITWISE_AND_TOKEN:
			lhsTypeDesc := b.getTypeDescFromExpr(binaryExpr.LhsExpr)
			rhsTypeDesc := b.getTypeDescFromExpr(binaryExpr.RhsExpr)
			return b.mergeTypes(lhsTypeDesc, binaryExpr.Operator, rhsTypeDesc)
		default:
			break
		}
		return expression
	case common.SIMPLE_NAME_REFERENCE,
		common.QUALIFIED_NAME_REFERENCE:
		return expression
	default:
		var simpleTypeDescIdentifier tree.STNode
		simpleTypeDescIdentifier = tree.CreateMissingTokenWithDiagnostics(
			common.IDENTIFIER_TOKEN, &common.ERROR_MISSING_TYPE_DESC)
		simpleTypeDescIdentifier = tree.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(simpleTypeDescIdentifier,
			expression)
		return tree.CreateSimpleNameReferenceNode(simpleTypeDescIdentifier)
	}
}

func (b *BallerinaParser) getBindingPatternsList(ambibuousList []tree.STNode, isListBP bool) []tree.STNode {
	var bindingPatterns []tree.STNode
	for _, item := range ambibuousList {
		bindingPatterns = append(bindingPatterns, b.getBindingPattern(item, isListBP))
	}
	return bindingPatterns
}

func (b *BallerinaParser) getBindingPattern(ambiguousNode tree.STNode, isListBP bool) tree.STNode {
	errorCode := common.ERROR_INVALID_BINDING_PATTERN
	if b.isEmpty(ambiguousNode) {
		return nil
	}
	switch ambiguousNode.Kind() {
	case common.WILDCARD_BINDING_PATTERN,
		common.CAPTURE_BINDING_PATTERN,
		common.LIST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN,
		common.ERROR_BINDING_PATTERN,
		common.REST_BINDING_PATTERN,
		common.FIELD_BINDING_PATTERN,
		common.NAMED_ARG_BINDING_PATTERN,
		common.COMMA_TOKEN:
		return ambiguousNode
	case common.SIMPLE_NAME_REFERENCE:
		simpleNameNode, ok := ambiguousNode.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("getBindingPattern: expected STSimpleNameReferenceNode")
		}
		varName := simpleNameNode.Name
		return b.createCaptureOrWildcardBP(varName)
	case common.QUALIFIED_NAME_REFERENCE:
		if isListBP {
			errorCode = common.ERROR_FIELD_BP_INSIDE_LIST_BP
			break
		}
		qualifiedName, ok := ambiguousNode.(*tree.STQualifiedNameReferenceNode)
		if !ok {
			panic("expected STQualifiedNameReferenceNode")
		}
		fieldName := tree.CreateSimpleNameReferenceNode(qualifiedName.ModulePrefix)
		return tree.CreateFieldBindingPatternFullNode(fieldName, qualifiedName.Colon,
			b.createCaptureOrWildcardBP(qualifiedName.Identifier))
	case common.BRACKETED_LIST,
		common.LIST_BP_OR_LIST_CONSTRUCTOR:
		innerList, ok := ambiguousNode.(*tree.STAmbiguousCollectionNode)
		if !ok {
			panic("expected STAmbiguousCollectionNode")
		}
		memberBindingPatterns := tree.CreateNodeList(b.getBindingPatternsList(innerList.Members, true)...)
		return tree.CreateListBindingPatternNode(innerList.CollectionStartToken, memberBindingPatterns,
			innerList.CollectionEndToken)
	case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		innerList, ok := ambiguousNode.(*tree.STAmbiguousCollectionNode)
		if !ok {
			panic("expected STAmbiguousCollectionNode")
		}
		var bindingPatterns []tree.STNode
		i := 0
		for ; i < len(innerList.Members); i++ {
			bp := b.getBindingPattern(innerList.Members[i], false)
			bindingPatterns = append(bindingPatterns, bp)
			if bp.Kind() == common.REST_BINDING_PATTERN {
				break
			}
		}
		memberBindingPatterns := tree.CreateNodeList(bindingPatterns...)
		return tree.CreateMappingBindingPatternNode(innerList.CollectionStartToken,
			memberBindingPatterns, innerList.CollectionEndToken)
	case common.SPECIFIC_FIELD:
		field, ok := ambiguousNode.(*tree.STSpecificFieldNode)
		if !ok {
			panic("expected STSpecificFieldNode")
		}
		fieldName := tree.CreateSimpleNameReferenceNode(field.FieldName)
		if field.ValueExpr == nil {
			return tree.CreateFieldBindingPatternVarnameNode(fieldName)
		}
		return tree.CreateFieldBindingPatternFullNode(fieldName, field.Colon,
			b.getBindingPattern(field.ValueExpr, false))
	case common.ERROR_CONSTRUCTOR:
		errorCons, ok := ambiguousNode.(*tree.STErrorConstructorExpressionNode)
		if !ok {
			panic("expected STErrorConstructorExpressionNode")
		}
		args := errorCons.Arguments
		size := args.BucketCount()
		var bindingPatterns []tree.STNode
		i := 0
		for ; i < size; i++ {
			arg := args.ChildInBucket(i)
			bindingPatterns = append(bindingPatterns, b.getBindingPattern(arg, false))
		}
		argListBindingPatterns := tree.CreateNodeList(bindingPatterns...)
		return tree.CreateErrorBindingPatternNode(errorCons.ErrorKeyword, errorCons.TypeReference,
			errorCons.OpenParenToken, argListBindingPatterns, errorCons.CloseParenToken)
	case common.POSITIONAL_ARG:
		positionalArg, ok := ambiguousNode.(*tree.STPositionalArgumentNode)
		if !ok {
			panic("expected STPositionalArgumentNode")
		}
		return b.getBindingPattern(positionalArg.Expression, false)
	case common.NAMED_ARG:
		namedArg, nameOk := ambiguousNode.(*tree.STNamedArgumentNode)
		if !nameOk {
			panic("exprected STNamedArgumentNode")
		}
		argNameNode, ok := namedArg.ArgumentName.(*tree.STSimpleNameReferenceNode)
		if !ok {
			panic("getBindingPattern: expected STSimpleNameReferenceNode for named argument")
		}
		bindingPatternArgName := argNameNode.Name
		return tree.CreateNamedArgBindingPatternNode(bindingPatternArgName, namedArg.EqualsToken,
			b.getBindingPattern(namedArg.Expression, false))
	case common.REST_ARG:
		restArg, ok := ambiguousNode.(*tree.STRestArgumentNode)
		if !ok {
			panic("expected STRestArgumentNode")
		}
		return tree.CreateRestBindingPatternNode(restArg.Ellipsis, restArg.Expression)
	}
	var identifier tree.STNode
	identifier = tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil)
	identifier = tree.CloneWithLeadingInvalidNodeMinutiae(identifier, ambiguousNode, &errorCode)
	return tree.CreateCaptureBindingPatternNode(identifier)
}

func (b *BallerinaParser) getExpressionList(ambibuousList []tree.STNode, isMappingConstructor bool) []tree.STNode {
	var exprList []tree.STNode
	for _, item := range ambibuousList {
		exprList = append(exprList, b.getExpressionInner(item, isMappingConstructor))
	}
	return exprList
}

func (b *BallerinaParser) getExpression(ambiguousNode tree.STNode) tree.STNode {
	return b.getExpressionInner(ambiguousNode, false)
}

func (b *BallerinaParser) getExpressionInner(ambiguousNode tree.STNode, isInMappingConstructor bool) tree.STNode {
	if ((b.isEmpty(ambiguousNode) || (b.isDefiniteExpr(ambiguousNode.Kind()) && (ambiguousNode.Kind() != common.INDEXED_EXPRESSION))) || b.isDefiniteAction(ambiguousNode.Kind())) || (ambiguousNode.Kind() == common.COMMA_TOKEN) {
		return ambiguousNode
	}
	switch ambiguousNode.Kind() {
	case common.BRACKETED_LIST, common.LIST_BP_OR_LIST_CONSTRUCTOR, common.TUPLE_TYPE_DESC_OR_LIST_CONST:
		innerList, ok := ambiguousNode.(*tree.STAmbiguousCollectionNode)
		if !ok {
			panic("getExpressionInner: expected STAmbiguousCollectionNode")
		}
		memberExprs := tree.CreateNodeList(b.getExpressionList(innerList.Members, false)...)
		return tree.CreateListConstructorExpressionNode(innerList.CollectionStartToken, memberExprs,
			innerList.CollectionEndToken)

	case common.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		innerList, ok := ambiguousNode.(*tree.STAmbiguousCollectionNode)
		if !ok {
			panic("getExpressionInner: expected STAmbiguousCollectionNode")
		}
		var fieldList []tree.STNode
		i := 0
		for ; i < len(innerList.Members); i++ {
			field := innerList.Members[i]
			var fieldNode tree.STNode
			if field.Kind() == common.QUALIFIED_NAME_REFERENCE {
				qualifiedNameRefNode, ok := field.(*tree.STQualifiedNameReferenceNode)
				if !ok {
					panic("getExpressionInner: expected STQualifiedNameReferenceNode")
				}
				readOnlyKeyword := tree.CreateEmptyNode()
				fieldName := qualifiedNameRefNode.ModulePrefix
				colon := qualifiedNameRefNode.Colon
				valueExpr := b.getExpression(qualifiedNameRefNode.Identifier)
				fieldNode = tree.CreateSpecificFieldNode(readOnlyKeyword, fieldName, colon, valueExpr)
			} else {
				fieldNode = b.getExpressionInner(field, true)
			}
			fieldList = append(fieldList, fieldNode)
		}
		fields := tree.CreateNodeList(fieldList...)
		return tree.CreateMappingConstructorExpressionNode(innerList.CollectionStartToken, fields,

			innerList.CollectionEndToken)

	case common.REST_BINDING_PATTERN:
		restBindingPattern, ok := ambiguousNode.(*tree.STRestBindingPatternNode)
		if !ok {
			panic("getExpressionInner: expected STRestBindingPatternNode")
		}
		if isInMappingConstructor {
			return tree.CreateSpreadFieldNode(restBindingPattern.EllipsisToken,
				restBindingPattern.VariableName)
		}

		return tree.CreateSpreadMemberNode(restBindingPattern.EllipsisToken,

			restBindingPattern.VariableName)

	case common.SPECIFIC_FIELD:
		field, ok := ambiguousNode.(*tree.STSpecificFieldNode)
		if !ok {
			panic("getExpressionInner: expected STSpecificFieldNode")
		}
		return tree.CreateSpecificFieldNode(field.ReadonlyKeyword, field.FieldName, field.Colon,

			b.getExpression(field.ValueExpr))

	case common.ERROR_CONSTRUCTOR:
		errorCons, ok := ambiguousNode.(*tree.STErrorConstructorExpressionNode)
		if !ok {
			panic("getExpressionInner: expected STErrorConstructorExpressionNode")
		}
		errorArgs := b.getErrorArgList(errorCons.Arguments)
		return tree.CreateErrorConstructorExpressionNode(errorCons.ErrorKeyword,
			errorCons.TypeReference, errorCons.OpenParenToken, errorArgs, errorCons.CloseParenToken)

	case common.IDENTIFIER_TOKEN:
		return tree.CreateSimpleNameReferenceNode(ambiguousNode)
	case common.INDEXED_EXPRESSION:
		indexedExpressionNode, ok := ambiguousNode.(*tree.STIndexedExpressionNode)
		if !ok {
			panic("getExpressionInner: expected STIndexedExpressionNode")
		}
		keys, ok := indexedExpressionNode.KeyExpression.(*tree.STNodeList)
		if !ok {
			panic("getExpressionInner: expected STNodeList")
		}
		if !keys.IsEmpty() {
			return ambiguousNode
		}
		lhsExpr := indexedExpressionNode.ContainerExpression
		openBracket := indexedExpressionNode.OpenBracket
		closeBracket := indexedExpressionNode.CloseBracket
		missingVarRef := tree.CreateSimpleNameReferenceNode(tree.CreateMissingToken(common.IDENTIFIER_TOKEN, nil))
		keyExpr := tree.CreateNodeList(missingVarRef)
		closeBracket = tree.AddDiagnostic(closeBracket,
			&common.ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR)
		return tree.CreateIndexedExpressionNode(lhsExpr, openBracket, keyExpr, closeBracket)
	case common.SIMPLE_NAME_REFERENCE, common.QUALIFIED_NAME_REFERENCE, common.COMPUTED_NAME_FIELD, common.SPREAD_FIELD, common.SPREAD_MEMBER:
		return ambiguousNode
	default:
		var simpleVarRef tree.STNode
		simpleVarRef = tree.CreateMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_EXPRESSION)
		simpleVarRef = tree.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(simpleVarRef, ambiguousNode)
		return tree.CreateSimpleNameReferenceNode(simpleVarRef)
	}
}

func (b *BallerinaParser) getMappingField(identifier tree.STNode, colon tree.STNode, bindingPatternOrExpr tree.STNode) tree.STNode {
	simpleNameRef := tree.CreateSimpleNameReferenceNode(identifier)
	switch bindingPatternOrExpr.Kind() {
	case common.LIST_BINDING_PATTERN,
		common.MAPPING_BINDING_PATTERN:
		return tree.CreateFieldBindingPatternFullNode(simpleNameRef, colon, bindingPatternOrExpr)
	case common.LIST_CONSTRUCTOR, common.MAPPING_CONSTRUCTOR:
		readonlyKeyword := tree.CreateEmptyNode()
		return tree.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, bindingPatternOrExpr)
	default:
		readonlyKeyword := tree.CreateEmptyNode()
		return tree.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, bindingPatternOrExpr)
	}
}

func (b *BallerinaParser) recoverWithBlockContext(nextToken tree.STToken, currentCtx common.ParserRuleContext) *Solution {
	if b.isInsideABlock(nextToken) {
		return b.recover(nextToken, currentCtx, true)
	} else {
		return b.recover(nextToken, currentCtx, false)
	}
}

func (b *BallerinaParser) isInsideABlock(nextToken tree.STToken) bool {
	if nextToken.Kind() != common.CLOSE_BRACE_TOKEN {
		return false
	}
	return slices.ContainsFunc(b.errorHandler.GetContextStack(), b.isBlockContext)
}

func (b *BallerinaParser) isBlockContext(ctx common.ParserRuleContext) bool {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER,
		common.PARSER_RULE_CONTEXT_BLOCK_STMT,
		common.PARSER_RULE_CONTEXT_MATCH_BODY,
		common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_FORK_STMT,
		common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS,
		common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS,
		common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION:
		return true
	default:
		return false
	}
}

func (b *BallerinaParser) isSpecialMethodName(token tree.STToken) bool {
	return (((token.Kind() == common.MAP_KEYWORD) || (token.Kind() == common.START_KEYWORD)) || (token.Kind() == common.JOIN_KEYWORD))
}

// GetSyntaxTree parses content into a syntax tree, attributing it to fileName
// (used for diagnostics and the syntax tree's text document).
func GetSyntaxTree(ctx *context.CompilerContext, fileName string, content string) (*tree.SyntaxTree, error) {
	return getSyntaxTree(ctx, fileName, content, false)
}

func GetImportSyntaxTree(ctx *context.CompilerContext, fileName string, content string) (*tree.SyntaxTree, error) {
	return getSyntaxTree(ctx, fileName, content, true)
}

func getSyntaxTree(ctx *context.CompilerContext, fileName string, content string, importsOnly bool) (*tree.SyntaxTree, error) {
	reader := text.CharReaderFromText(content)
	lexer := NewLexer(reader)
	tokenReader := CreateTokenReader(lexer)
	ballerinaParser := NewBallerinaParserFromTokenReader(tokenReader)
	var rootNode *tree.STModulePart
	if importsOnly {
		rootNode = ballerinaParser.ParseImports().(*tree.STModulePart)
	} else {
		rootNode = ballerinaParser.Parse().(*tree.STModulePart)
	}
	moduleNode := tree.CreateUnlinkedFacade[*tree.STModulePart, *tree.ModulePart](rootNode)
	syntaxTree := tree.NewSyntaxTreeFromNodeTextDocument(moduleNode, nil, fileName, false)
	if syntaxTree.HasDiagnostics() {
		ctx.SyntaxError("syntax error at", diagnostics.Location{})
	}
	return &syntaxTree, nil
}
