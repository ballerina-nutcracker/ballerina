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

// Package parser parses Ballerina source code into syntax trees.
package parser

import (
	"errors"
	"slices"
	"strings"

	debugcommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

type operatorPrecedence uint8

const (
	operatorPrecedenceMemberAccess     operatorPrecedence = iota //  x.k, x.@a, f(x), x.f(y), x[y], x?.k, x.<y>, x/<y>, x/**/<y>, x/*xml-step-extend
	operatorPrecedenceUnary                                      //  (+x), (-x), (~x), (!x), (<T>x), (typeof x),
	operatorPrecedenceExpressionAction                           //  Expression that can also be an action. eg: (check x), (checkpanic x). Same as unary.
	operatorPrecedenceMultiplicative                             //  (x * y), (x / y), (x % y)
	operatorPrecedenceAdditive                                   //  (x + y), (x - y)
	operatorPrecedenceShift                                      //  (x << y), (x >> y), (x >>> y)
	operatorPrecedenceRange                                      //  (x ... y), (x ..< y)
	operatorPrecedenceBinaryCompare                              //  (x < y), (x > y), (x <= y), (x >= y), (x is y)
	operatorPrecedenceEquality                                   //  (x == y), (x != y), (x == y), (x === y), (x !== y)
	operatorPrecedenceBitwiseAnd                                 //  (x & y)
	operatorPrecedenceBitwiseXor                                 //  (x ^ y)
	operatorPrecedenceBitwiseOr                                  //  (x | y)
	operatorPrecedenceLogicalAnd                                 //  (x && y)
	operatorPrecedenceLogicalOr                                  //  (x || y)
	operatorPrecedenceElvisConditional                           //  x ?: y
	operatorPrecedenceConditional                                //  x ? y : z

	operatorPrecedenceAnonFuncOrLet //  (x) => y

	//  Actions cannot reside inside expressions (excluding query-action-or-expr), hence they have the lowest
	//  precedence.
	operatorPrecedenceRemoteCallAction //  (x -> y()),
	operatorPrecedenceAction           //  (start x), ...
	operatorPrecedenceTrap             //  (trap x)

	// A query-action-or-expr or a query-action can have actions in certain clauses.
	operatorPrecedenceQuery //  from x, select x, where x

	operatorPrecedenceDefault //  (start x), ...
)

const defaultOpPrecedence operatorPrecedence = operatorPrecedenceDefault

func (o *operatorPrecedence) isHigherThanOrEqual(other operatorPrecedence, allowActions bool) bool {
	if allowActions {
		if (*o == operatorPrecedenceExpressionAction) && (other == operatorPrecedenceRemoteCallAction) {
			return false
		}
	}
	return uint8(*o) <= uint8(other)
}

type typePrecedence uint8

func (t *typePrecedence) isHigherThanOrEqual(other typePrecedence) bool {
	return uint8(*t) <= uint8(other)
}

const (
	typePrecedenceDistinct        typePrecedence = iota // distinct T
	typePrecedenceArrayOrOptional                       // T[], T?
	typePrecedenceIntersection                          // T1 & T2
	typePrecedenceUnion                                 // T1 | T2
	typePrecedenceDefault                               // function(args) returns T
)

type action uint8

const (
	actionInsert action = iota
	actionRemove
	actionKeep
)

type parserErrorHandler interface {
	SwitchContext(context common.ParserRuleContext)
	GetParentContext() common.ParserRuleContext
	EndContext()
	StartContext(context common.ParserRuleContext)
	Recover(currentCtx common.ParserRuleContext, token st.STToken, isCompletion bool) *solution
	GetContextStack() []common.ParserRuleContext
	GetGrandParentContext() common.ParserRuleContext
	ConsumeInvalidToken() st.STToken
}

type invalidNodeInfo struct {
	node           st.STNode
	diagnosticCode diagnostics.DiagnosticCode
	args           []any
}

type abstractParser struct {
	errorHandler         parserErrorHandler
	tokenReader          *tokenReader
	invalidNodeInfoStack []invalidNodeInfo
	insertedToken        st.STToken
	ctx                  *context.CompilerContext
	location             diagnostics.Location
}

func (a *abstractParser) internalError(message string) {
	a.ctx.InternalError(message, a.location)
}

func (a *abstractParser) unimplemented(message string) {
	a.ctx.Unimplemented(message, a.location)
}

func newAbstractParserFromTokenReader(tokenReader *tokenReader) abstractParser {
	this := abstractParser{}
	this.invalidNodeInfoStack = make([]invalidNodeInfo, 0)
	this.insertedToken = nil
	// Default field initializations

	this.tokenReader = tokenReader
	this.errorHandler = nil
	this.ctx = tokenReader.ctx
	this.location = tokenReader.location
	return this
}

func (a *abstractParser) peek() st.STToken {
	if a.insertedToken != nil {
		return a.insertedToken
	}
	return a.tokenReader.Peek()
}

func (a *abstractParser) peekN(n int) st.STToken {
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

func (a *abstractParser) consume() st.STToken {
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

func (a *abstractParser) consumeWithInvalidNodes() st.STToken {
	token := a.tokenReader.Read()
	return a.consumeWithInvalidNodesWithToken(token)
}

func (a *abstractParser) consumeWithInvalidNodesWithToken(token st.STToken) st.STToken {
	newToken := token
	for len(a.invalidNodeInfoStack) > 0 {
		invalidNodeInfo := a.invalidNodeInfoStack[len(a.invalidNodeInfoStack)-1]
		a.invalidNodeInfoStack = a.invalidNodeInfoStack[:len(a.invalidNodeInfoStack)-1]
		newToken = st.ToToken(st.CloneWithLeadingInvalidNodeMinutiae(newToken, invalidNodeInfo.node,
			invalidNodeInfo.diagnosticCode, invalidNodeInfo.args))
	}
	return newToken
}

func (a *abstractParser) recover(token st.STToken, currentCtx common.ParserRuleContext, isCompletion bool) *solution {
	isCompletion = isCompletion || token.Kind() == st.EOF_TOKEN
	sol := a.errorHandler.Recover(currentCtx, token, isCompletion)
	if sol == nil {
		return &solution{Action: actionKeep, Ctx: currentCtx, RecoveredNode: failedToken(), TokenKind: st.EOF_TOKEN}
	}
	switch sol.Action {
	case actionRemove:
		a.insertedToken = nil
		a.addInvalidTokenToNextToken(sol.RemovedToken)
	case actionInsert:
		a.insertedToken = st.ToToken(sol.RecoveredNode)
	}
	return sol
}

func (a *abstractParser) insertToken(kind st.SyntaxKind, context common.ParserRuleContext) {
	a.insertedToken = createMissingTokenWithDiagnosticsFromParserRules(kind, context)
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

func (a *abstractParser) getNextNextToken() st.STToken {
	return a.peekN(2)
}

func (a *abstractParser) isNodeListEmpty(node st.STNode) bool {
	nodeList, ok := node.(*st.STNodeList)
	if !ok {
		a.internalError("node is not a STNodeList")
		return false
	}
	return nodeList.IsEmpty()
}

func (a *abstractParser) cloneWithDiagnosticIfListEmpty(nodeList st.STNode, target st.STNode, diagnosticCode diagnostics.DiagnosticCode) st.STNode {
	if a.isNodeListEmpty(nodeList) {
		return st.AddDiagnostic(target, diagnosticCode)
	}
	return target
}

func (a *abstractParser) updateLastNodeInListWithInvalidNode(nodeList []st.STNode, invalidParam st.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) []st.STNode {
	prevNode := nodeList[len(nodeList)-1]
	nodeList = nodeList[:len(nodeList)-1]
	newNode := st.CloneWithTrailingInvalidNodeMinutiae(prevNode, invalidParam, diagnosticCode, args)
	nodeList = append(nodeList, newNode)
	return nodeList
}

func (a *abstractParser) updateFirstNodeInListWithLeadingInvalidNode(nodeList []st.STNode, invalidParam st.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) {
	a.updateANodeInListWithLeadingInvalidNode(nodeList, 0, invalidParam, diagnosticCode, args)
}

func (a *abstractParser) updateANodeInListWithLeadingInvalidNode(nodeList []st.STNode, indexOfTheNode int, invalidParam st.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) {
	node := nodeList[indexOfTheNode]
	newNode := st.CloneWithLeadingInvalidNodeMinutiae(node, invalidParam, diagnosticCode, args)
	nodeList[indexOfTheNode] = newNode
}

func (a *abstractParser) invalidateRestAndAddToTrailingMinutiae(node st.STNode) st.STNode {
	node = a.addInvalidNodeStackToTrailingMinutiae(node)
	for a.peek().Kind() != st.EOF_TOKEN {
		invalidToken := a.consume()
		node = st.CloneWithTrailingInvalidNodeMinutiae(node, invalidToken, &common.ERROR_INVALID_TOKEN, invalidToken.Text())
	}
	return node
}

func (a *abstractParser) addInvalidNodeStackToTrailingMinutiae(node st.STNode) st.STNode {
	for len(a.invalidNodeInfoStack) != 0 {
		invalidNodeInfo := a.invalidNodeInfoStack[len(a.invalidNodeInfoStack)-1]
		a.invalidNodeInfoStack = a.invalidNodeInfoStack[:len(a.invalidNodeInfoStack)-1]
		node = st.CloneWithTrailingInvalidNodeMinutiae(node, invalidNodeInfo.node, invalidNodeInfo.diagnosticCode, invalidNodeInfo.args)
	}
	return node
}

func (a *abstractParser) addInvalidNodeToNextToken(invalidNode st.STNode, diagnosticCode diagnostics.DiagnosticCode, args ...any) {
	a.invalidNodeInfoStack = append(a.invalidNodeInfoStack, invalidNodeInfo{node: invalidNode, diagnosticCode: diagnosticCode, args: args})
}

func (a *abstractParser) addInvalidTokenToNextToken(invalidNode st.STToken) {
	a.invalidNodeInfoStack = append(a.invalidNodeInfoStack, invalidNodeInfo{node: invalidNode, diagnosticCode: &common.ERROR_INVALID_TOKEN, args: []any{invalidNode.Text()}})
}

type ballerinaParser struct {
	abstractParser
}

func newBallerinaParserFromTokenReader(tokenReader *tokenReader) ballerinaParser {
	this := ballerinaParser{}
	// Default field initializations

	this.abstractParser = abstractParser{
		tokenReader:          tokenReader,
		invalidNodeInfoStack: make([]invalidNodeInfo, 0),
		insertedToken:        nil,
		ctx:                  tokenReader.ctx,
		location:             tokenReader.location,
	}
	errorHandler := newBallerinaParserErrorHandlerFromTokenReader(this.tokenReader)
	this.errorHandler = &errorHandler
	return this
}

func isParameterizedTypeToken(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.TYPEDESC_KEYWORD, st.FUTURE_KEYWORD, st.XML_KEYWORD, st.ERROR_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) createBuiltinSimpleNameReference(token st.STNode) st.STNode {
	typeKind := b.getBuiltinTypeSyntaxKind(token.Kind())
	return st.CreateBuiltinSimpleNameReferenceNode(typeKind, token)
}

func isCompoundBinaryOperator(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.SLASH_TOKEN,
		st.ASTERISK_TOKEN,
		st.BITWISE_AND_TOKEN,
		st.BITWISE_XOR_TOKEN,
		st.PIPE_TOKEN,
		st.DOUBLE_LT_TOKEN,
		st.DOUBLE_GT_TOKEN,
		st.TRIPPLE_GT_TOKEN:
		return true
	default:
		return false
	}
}

func isTypeStartingToken(nextTokenKind st.SyntaxKind, nextNextToken st.STToken) bool {
	switch nextTokenKind {
	case st.IDENTIFIER_TOKEN,
		st.SERVICE_KEYWORD,
		st.RECORD_KEYWORD,
		st.OBJECT_KEYWORD,
		st.ABSTRACT_KEYWORD,
		st.CLIENT_KEYWORD,
		st.OPEN_PAREN_TOKEN,
		st.MAP_KEYWORD,
		st.STREAM_KEYWORD,
		st.TABLE_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.OPEN_BRACKET_TOKEN,
		st.DISTINCT_KEYWORD,
		st.ISOLATED_KEYWORD,
		st.TRANSACTIONAL_KEYWORD,
		st.TRANSACTION_KEYWORD,
		st.NATURAL_KEYWORD:
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

func isSimpleType(nodeKind st.SyntaxKind) bool {
	switch nodeKind {
	case st.INT_KEYWORD,
		st.FLOAT_KEYWORD,
		st.DECIMAL_KEYWORD,
		st.BOOLEAN_KEYWORD,
		st.STRING_KEYWORD,
		st.BYTE_KEYWORD,
		st.JSON_KEYWORD,
		st.HANDLE_KEYWORD,
		st.ANY_KEYWORD,
		st.ANYDATA_KEYWORD,
		st.NEVER_KEYWORD,
		st.VAR_KEYWORD,
		st.READONLY_KEYWORD:
		return true
	default:
		return false
	}
}

func isPredeclaredPrefix(nodeKind st.SyntaxKind) bool {
	switch nodeKind {
	case st.BOOLEAN_KEYWORD,
		st.DECIMAL_KEYWORD,
		st.ERROR_KEYWORD,
		st.FLOAT_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.FUTURE_KEYWORD,
		st.INT_KEYWORD,
		st.MAP_KEYWORD,
		st.NATURAL_KEYWORD,
		st.OBJECT_KEYWORD,
		st.STREAM_KEYWORD,
		st.STRING_KEYWORD,
		st.TABLE_KEYWORD,
		st.TRANSACTION_KEYWORD,
		st.TYPEDESC_KEYWORD,
		st.XML_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) getBuiltinTypeSyntaxKind(typeKeyword st.SyntaxKind) st.SyntaxKind {
	switch typeKeyword {
	case st.INT_KEYWORD:
		return st.INT_TYPE_DESC
	case st.FLOAT_KEYWORD:
		return st.FLOAT_TYPE_DESC
	case st.DECIMAL_KEYWORD:
		return st.DECIMAL_TYPE_DESC
	case st.BOOLEAN_KEYWORD:
		return st.BOOLEAN_TYPE_DESC
	case st.STRING_KEYWORD:
		return st.STRING_TYPE_DESC
	case st.BYTE_KEYWORD:
		return st.BYTE_TYPE_DESC
	case st.JSON_KEYWORD:
		return st.JSON_TYPE_DESC
	case st.HANDLE_KEYWORD:
		return st.HANDLE_TYPE_DESC
	case st.ANY_KEYWORD:
		return st.ANY_TYPE_DESC
	case st.ANYDATA_KEYWORD:
		return st.ANYDATA_TYPE_DESC
	case st.NEVER_KEYWORD:
		return st.NEVER_TYPE_DESC
	case st.VAR_KEYWORD:
		return st.VAR_TYPE_DESC
	case st.READONLY_KEYWORD:
		return st.READONLY_TYPE_DESC
	default:
		b.internalError(typeKeyword.StrValue() + "is not a built-in type")
		return st.NONE
	}
}

func isKeyKeyword(token st.STToken) bool {
	return ((token.Kind() == st.IDENTIFIER_TOKEN) && key == token.Text())
}

func isNaturalKeyword(token st.STToken) bool {
	return ((token.Kind() == st.IDENTIFIER_TOKEN) && natural == (token.Text()))
}

func isEndOfLetVarDeclarations(nextToken st.STToken, nextNextToken st.STToken) bool {
	tokenKind := nextToken.Kind()
	switch tokenKind {
	case st.COMMA_TOKEN, st.AT_TOKEN:
		return false
	case st.IN_KEYWORD:
		return true
	default:
		return (isGroupOrCollectKeyword(nextToken) || (!isTypeStartingToken(tokenKind, nextNextToken)))
	}
}

func isGroupOrCollectKeyword(nextToken st.STToken) bool {
	return (isKeywordMatch(st.COLLECT_KEYWORD, nextToken) || isKeywordMatch(st.GROUP_KEYWORD, nextToken))
}

func isKeywordMatch(syntaxKind st.SyntaxKind, token st.STToken) bool {
	return ((token.Kind() == st.IDENTIFIER_TOKEN) && syntaxKind.StrValue() == (token.Text()))
}

func isSingletonTypeDescStart(tokenKind st.SyntaxKind, nextNextToken st.STToken) bool {
	switch tokenKind {
	case st.STRING_LITERAL_TOKEN,
		st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.NULL_KEYWORD:
		return true
	case st.PLUS_TOKEN, st.MINUS_TOKEN:
		return isIntOrFloat(nextNextToken)
	default:
		return false
	}
}

func isIntOrFloat(token st.STToken) bool {
	switch token.Kind() {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
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
		case tab,
			newline,
			carriageReturn,
			space:
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
		case tab,
			newline,
			carriageReturn,
			space:
		case equal:
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

func (b *ballerinaParser) Parse() st.STNode {
	ast := b.parseCompUnit()
	debugcommon.DebugWriteLazy(debugcommon.DUMP_ST, func() string { return generateJSON(ast) })
	return ast
}

func (b *ballerinaParser) ParseImports() st.STNode {
	ast := b.parseCompUnitImports()
	debugcommon.DebugWriteLazy(debugcommon.DUMP_ST, func() string { return generateJSON(ast) })
	return ast
}

func (b *ballerinaParser) ParseAsStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	stmt := b.parseStatement()
	if (stmt == nil) || b.validateStatement(stmt) {
		stmt = b.createMissingSimpleVarDecl(false)
		stmt = b.invalidateRestAndAddToTrailingMinutiae(stmt)
		return stmt
	}
	if stmt.Kind() == st.NAMED_WORKER_DECLARATION {
		b.addInvalidNodeToNextToken(stmt, &common.ERROR_NAMED_WORKER_NOT_ALLOWED_HERE)
		stmt = b.createMissingSimpleVarDecl(false)
		stmt = b.invalidateRestAndAddToTrailingMinutiae(stmt)
		return stmt
	}
	stmt = b.invalidateRestAndAddToTrailingMinutiae(stmt)
	return stmt
}

func (b *ballerinaParser) ParseAsBlockStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	b.startContext(common.PARSER_RULE_CONTEXT_WHILE_BLOCK)
	blockStmtNode := b.parseBlockNode()
	blockStmtNode = b.invalidateRestAndAddToTrailingMinutiae(blockStmtNode)
	return blockStmtNode
}

func (b *ballerinaParser) ParseAsStatements() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	stmtsNode := b.parseStatements()
	stmtNodeList, ok := stmtsNode.(*st.STNodeList)
	if !ok {
		b.internalError("stmtsNode is not a STNodeList")
		return st.CreateEmptyNode()
	}
	var stmts []st.STNode
	for i := 0; i < (stmtNodeList.Size() - 1); i++ {
		stmts = append(stmts, stmtNodeList.Get(i))
	}
	var lastStmt st.STNode
	if stmtNodeList.Size() == 0 {
		lastStmt = b.createMissingSimpleVarDecl(false)
	} else {
		lastStmt = stmtNodeList.Get(stmtNodeList.Size() - 1)
	}
	lastStmt = b.invalidateRestAndAddToTrailingMinutiae(lastStmt)
	stmts = append(stmts, lastStmt)
	return st.CreateNodeList(stmts...)
}

func (b *ballerinaParser) ParseAsExpression() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	expr := b.parseExpression()
	expr = b.invalidateRestAndAddToTrailingMinutiae(expr)
	return expr
}

func (b *ballerinaParser) ParseAsActionOrExpression() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	actionOrExpr := b.parseActionOrExpression()
	actionOrExpr = b.invalidateRestAndAddToTrailingMinutiae(actionOrExpr)
	return actionOrExpr
}

func (b *ballerinaParser) ParseAsModuleMemberDeclaration() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	topLevelNode := b.parseTopLevelNode()
	if topLevelNode == nil {
		topLevelNode = b.createMissingSimpleVarDecl(true)
	}
	if topLevelNode.Kind() == st.IMPORT_DECLARATION {
		temp := topLevelNode
		topLevelNode = b.createMissingSimpleVarDecl(true)
		topLevelNode = st.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(topLevelNode, temp)
	}
	topLevelNode = b.invalidateRestAndAddToTrailingMinutiae(topLevelNode)
	return topLevelNode
}

func (b *ballerinaParser) ParseAsImportDeclaration() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	importDecl := b.parseImportDecl()
	importDecl = b.invalidateRestAndAddToTrailingMinutiae(importDecl)
	return importDecl
}

func (b *ballerinaParser) ParseAsTypeDescriptor() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION)
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF)
	typeDesc = b.invalidateRestAndAddToTrailingMinutiae(typeDesc)
	return typeDesc
}

func (b *ballerinaParser) ParseAsBindingPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	bindingPattern := b.parseBindingPattern()
	bindingPattern = b.invalidateRestAndAddToTrailingMinutiae(bindingPattern)
	return bindingPattern
}

func (b *ballerinaParser) ParseAsFunctionBodyBlock() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	funcBodyBlock := b.parseFunctionBodyBlock(false)
	funcBodyBlock = b.invalidateRestAndAddToTrailingMinutiae(funcBodyBlock)
	return funcBodyBlock
}

func (b *ballerinaParser) ParseAsObjectMember() st.STNode {
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

func (b *ballerinaParser) ParseAsIntermediateClause(allowActions bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	b.startContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
	var intermediateClause st.STNode
	if !b.isEndOfIntermediateClause(b.peek().Kind()) {
		intermediateClause = b.parseIntermediateClause(true, allowActions)
	}
	if intermediateClause == nil {
		intermediateClause = b.createMissingWhereClause()
	}
	if intermediateClause.Kind() == st.SELECT_CLAUSE {
		temp := intermediateClause
		intermediateClause = b.createMissingWhereClause()
		intermediateClause = st.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(intermediateClause, temp)
	}
	intermediateClause = b.invalidateRestAndAddToTrailingMinutiae(intermediateClause)
	return intermediateClause
}

func (b *ballerinaParser) ParseAsLetVarDeclaration(allowActions bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	b.switchContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
	b.switchContext(common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL)
	letVarDeclaration := b.parseLetVarDecl(common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL, true, allowActions)
	letVarDeclaration = b.invalidateRestAndAddToTrailingMinutiae(letVarDeclaration)
	return letVarDeclaration
}

func (b *ballerinaParser) ParseAsAnnotation() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATIONS)
	annotation := b.parseAnnotation()
	annotation = b.invalidateRestAndAddToTrailingMinutiae(annotation)
	return annotation
}

func (b *ballerinaParser) ParseAsMarkdownDocumentation() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	markdownDoc := b.parseMarkdownDocumentation()
	if st.ToSourceCode(markdownDoc) == "" {
		missingHash := st.CreateMissingTokenWithDiagnostics(st.HASH_TOKEN,
			&common.WARNING_MISSING_HASH_TOKEN)
		docLine := st.CreateMarkdownDocumentationLineNode(st.MARKDOWN_DOCUMENTATION_LINE,
			missingHash, st.CreateEmptyNodeList())
		markdownDoc = st.CreateMarkdownDocumentationNode(st.CreateNodeListFromNodes(docLine))
	}
	markdownDoc = b.invalidateRestAndAddToTrailingMinutiae(markdownDoc)
	return markdownDoc
}

func (b *ballerinaParser) ParseWithContext(context common.ParserRuleContext) st.STNode {
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
		b.internalError("Cannot start parsing from: " + context.String())
		return st.CreateEmptyNode()
	}
}

func (b *ballerinaParser) parseCompUnit() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	var otherDecls []st.STNode
	var importDecls []st.STNode
	processImports := true
	token := b.peek()
	for token.Kind() != st.EOF_TOKEN {
		decl := b.parseTopLevelNode()
		if decl == nil {
			break
		}
		if decl.Kind() == st.IMPORT_DECLARATION {
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
	return st.CreateModulePartNode(st.CreateNodeList(importDecls...), st.CreateNodeList(otherDecls...), eof)
}

func (b *ballerinaParser) parseCompUnitImports() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMP_UNIT)
	var importDecls []st.STNode
	for b.peek().Kind() == st.IMPORT_KEYWORD {
		importDecls = append(importDecls, b.parseImportDecl())
	}
	b.endContext()
	return st.CreateModulePartNode(
		st.CreateNodeList(importDecls...),
		st.CreateEmptyNodeList(),
		st.CreateMissingToken(st.EOF_TOKEN, nil),
	)
}

func (b *ballerinaParser) parseTopLevelNode() st.STNode {
	nextToken := b.peek()
	var metadata st.STNode
	switch nextToken.Kind() {
	case st.EOF_TOKEN:
		return nil
	case st.DOCUMENTATION_STRING, st.AT_TOKEN:
		metadata = b.parseMetaData()
		return b.parseTopLevelNodeWithMetadata(metadata)
	case st.IMPORT_KEYWORD,
		st.FINAL_KEYWORD,
		st.PUBLIC_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.TYPE_KEYWORD,
		st.LISTENER_KEYWORD,
		st.CONST_KEYWORD,
		st.ANNOTATION_KEYWORD,
		st.XMLNS_KEYWORD,
		st.ENUM_KEYWORD,
		st.CLASS_KEYWORD,
		st.TRANSACTIONAL_KEYWORD,
		st.ISOLATED_KEYWORD,
		st.DISTINCT_KEYWORD,
		st.CLIENT_KEYWORD,
		st.READONLY_KEYWORD,
		st.CONFIGURABLE_KEYWORD,
		st.SERVICE_KEYWORD:
		metadata = st.CreateEmptyNode()
	case st.RESOURCE_KEYWORD, st.REMOTE_KEYWORD:
		b.reportInvalidQualifier(b.consume())
		return b.parseTopLevelNode()
	case st.IDENTIFIER_TOKEN:
		if b.isModuleVarDeclStart(1) || nextToken.IsMissing() {
			return b.parseModuleVarDecl(st.CreateEmptyNode())
		}
		fallthrough
	default:
		if isTypeStartingToken(nextToken.Kind(), b.getNextNextToken()) && (nextToken.Kind() != st.IDENTIFIER_TOKEN) {
			metadata = st.CreateEmptyNode()
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE)
		if solution.Action == actionKeep {
			metadata = st.CreateEmptyNode()
			break
		}
		return b.parseTopLevelNode()
	}
	return b.parseTopLevelNodeWithMetadata(metadata)
}

func (b *ballerinaParser) parseTopLevelNodeWithMetadata(metadata st.STNode) st.STNode {
	nextToken := b.peek()
	var publicQualifier st.STNode
	switch nextToken.Kind() {
	case st.EOF_TOKEN:
		if metadata != nil {
			metadaNode, ok := metadata.(*st.STMetadataNode)
			if !ok {
				b.internalError("metadata is not a STMetadataNode")
				return st.CreateEmptyNode()
			}
			metadata = b.addMetadataNotAttachedDiagnostic(*metadaNode)
			return b.createMissingSimpleVarDeclInner(metadata, true)
		}
		return nil
	case st.PUBLIC_KEYWORD:
		publicQualifier = b.consume()
	case st.FUNCTION_KEYWORD,
		st.TYPE_KEYWORD,
		st.LISTENER_KEYWORD,
		st.CONST_KEYWORD,
		st.FINAL_KEYWORD,
		st.IMPORT_KEYWORD,
		st.ANNOTATION_KEYWORD,
		st.XMLNS_KEYWORD,
		st.ENUM_KEYWORD,
		st.CLASS_KEYWORD,
		st.TRANSACTIONAL_KEYWORD,
		st.ISOLATED_KEYWORD,
		st.DISTINCT_KEYWORD,
		st.CLIENT_KEYWORD,
		st.READONLY_KEYWORD,
		st.SERVICE_KEYWORD,
		st.CONFIGURABLE_KEYWORD:
		break
	case st.RESOURCE_KEYWORD, st.REMOTE_KEYWORD:
		b.reportInvalidQualifier(b.consume())
		return b.parseTopLevelNodeWithMetadata(metadata)
	case st.IDENTIFIER_TOKEN:
		if b.isModuleVarDeclStart(1) {
			return b.parseModuleVarDecl(metadata)
		}
		fallthrough
	default:
		if b.isTypeStartingToken(nextToken.Kind()) && (nextToken.Kind() != st.IDENTIFIER_TOKEN) {
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA)
		if solution.Action == actionKeep {
			publicQualifier = st.CreateEmptyNode()
			break
		}
		return b.parseTopLevelNodeWithMetadata(metadata)
	}
	return b.parseTopLevelNodeWithQualifiers(metadata, publicQualifier)
}

func (b *ballerinaParser) addMetadataNotAttachedDiagnostic(metadata st.STMetadataNode) st.STNode {
	docString := metadata.DocumentationString
	if docString != nil {
		docString = st.AddDiagnostic(docString, &common.ERROR_DOCUMENTATION_NOT_ATTACHED_TO_A_CONSTRUCT)
	}
	annotList, ok := metadata.Annotations.(*st.STNodeList)
	if !ok {
		b.internalError("annotations is not a STNodeList")
		return st.CreateEmptyNode()
	}
	annotations := b.addAnnotNotAttachedDiagnostic(annotList)
	return st.CreateMetadataNode(docString, annotations)
}

func (b *ballerinaParser) addAnnotNotAttachedDiagnostic(annotList *st.STNodeList) st.STNode {
	annotations := st.UpdateAllNodesInNodeListWithDiagnostic(annotList, &common.ERROR_ANNOTATION_NOT_ATTACHED_TO_A_CONSTRUCT)
	return annotations
}

func (b *ballerinaParser) isModuleVarDeclStart(lookahead int) bool {
	nextToken := b.peekN(lookahead + 1)
	switch nextToken.Kind() {
	case st.EQUAL_TOKEN, // Scenario: foo = . Even though this is not valid, consider this as a var-decl and
		// continue;
		st.OPEN_BRACKET_TOKEN,  // Scenario foo[] (Array type descriptor with custom type)
		st.QUESTION_MARK_TOKEN, // Scenario foo? (Optional type descriptor with custom type)
		st.PIPE_TOKEN,          // Scenario foo | (Union type descriptor with custom type)
		st.BITWISE_AND_TOKEN,   // Scenario foo & (Intersection type descriptor with custom type)
		st.OPEN_BRACE_TOKEN,    // Scenario foo{} (mapping-binding-pattern)
		st.ERROR_KEYWORD,       // Scenario foo error (error-binding-pattern)
		st.EOF_TOKEN:
		return true
	case st.IDENTIFIER_TOKEN:
		switch b.peekN(lookahead + 2).Kind() {
		case st.EQUAL_TOKEN,
			// Scenario: foo bar =
			st.SEMICOLON_TOKEN,
			// Scenario: foo bar;
			st.EOF_TOKEN:
			return true
		default:
			return false
		}
	case st.COLON_TOKEN:
		if lookahead > 1 {
			return false
		}
		switch b.peekN(lookahead + 2).Kind() {
		case st.IDENTIFIER_TOKEN:
			return b.isModuleVarDeclStart(lookahead + 2)
		case st.EOF_TOKEN:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func (b *ballerinaParser) parseImportDecl() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_IMPORT_DECL)
	b.tokenReader.StartMode(parserModeImportMode)
	importKeyword := b.parseImportKeyword()
	identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME)
	importDecl := b.parseImportDeclWithIdentifier(importKeyword, identifier)
	b.tokenReader.EndMode()
	b.endContext()
	return importDecl
}

func (b *ballerinaParser) parseImportKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.IMPORT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD)
		return b.parseImportKeyword()
	}
}

func (b *ballerinaParser) parseIdentifier(currentCtx common.ParserRuleContext) st.STNode {
	token := b.peek()
	if token.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else if token.Kind() == st.MAP_KEYWORD {
		mapKeyword := b.consume()
		return st.CreateIdentifierTokenWithDiagnostics(mapKeyword.Text(), mapKeyword.LeadingMinutiae(), mapKeyword.TrailingMinutiae(),
			mapKeyword.Diagnostics())
	} else {
		b.recoverWithBlockContext(token, currentCtx)
		return b.parseIdentifier(currentCtx)
	}
}

func (b *ballerinaParser) parseImportDeclWithIdentifier(importKeyword st.STNode, identifier st.STNode) st.STNode {
	nextToken := b.peek()
	var orgName st.STNode
	var moduleName st.STNode
	var alias st.STNode
	switch nextToken.Kind() {
	case st.SLASH_TOKEN:
		slash := b.parseSlashToken()
		orgName = st.CreateImportOrgNameNode(identifier, slash)
		moduleName = b.parseModuleName()
		alias = b.parseImportPrefixDecl()
	case st.DOT_TOKEN, st.AS_KEYWORD:
		orgName = st.CreateEmptyNode()
		moduleName = b.parseModuleNameInner(identifier)
		alias = b.parseImportPrefixDecl()
	case st.SEMICOLON_TOKEN:
		orgName = st.CreateEmptyNode()
		moduleName = b.parseModuleNameInner(identifier)
		alias = st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS)
		return b.parseImportDeclWithIdentifier(importKeyword, identifier)
	}
	semicolon := b.parseSemicolon()
	return st.CreateImportDeclarationNode(importKeyword, orgName, moduleName, alias, semicolon)
}

func (b *ballerinaParser) parseSlashToken() st.STToken {
	token := b.peek()
	if token.Kind() == st.SLASH_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SLASH)
		return b.parseSlashToken()
	}
}

func (b *ballerinaParser) parseDotToken() st.STNode {
	token := b.peek()
	if token.Kind() == st.DOT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_DOT)
		return b.parseDotToken()
	}
}

func (b *ballerinaParser) parseModuleName() st.STNode {
	moduleNameStart := b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME)
	return b.parseModuleNameInner(moduleNameStart)
}

func (b *ballerinaParser) parseModuleNameInner(moduleNameStart st.STNode) st.STNode {
	var moduleNameParts []st.STNode
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
	return st.CreateNodeList(moduleNameParts...)
}

func (b *ballerinaParser) parseModuleNameRhs() st.STNode {
	switch b.peek().Kind() {
	case st.DOT_TOKEN:
		return b.consume()
	case st.AS_KEYWORD, st.SEMICOLON_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME)
		return b.parseModuleNameRhs()
	}
}

func (b *ballerinaParser) isEndOfImportDecl(nextToken st.STToken) bool {
	switch nextToken.Kind() {
	case st.SEMICOLON_TOKEN,
		st.PUBLIC_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.TYPE_KEYWORD,
		st.ABSTRACT_KEYWORD,
		st.CONST_KEYWORD,
		st.EOF_TOKEN,
		st.SERVICE_KEYWORD,
		st.IMPORT_KEYWORD,
		st.FINAL_KEYWORD,
		st.TRANSACTIONAL_KEYWORD,
		st.ISOLATED_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseDecimalIntLiteral(context common.ParserRuleContext) st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.DECIMAL_INTEGER_LITERAL_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), context)
		return b.parseDecimalIntLiteral(context)
	}
}

func (b *ballerinaParser) parseImportPrefixDecl() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.AS_KEYWORD:
		asKeyword := b.parseAsKeyword()
		prefix := b.parseImportPrefix()
		return st.CreateImportPrefixNode(asKeyword, prefix)
	case st.SEMICOLON_TOKEN:
		return st.CreateEmptyNode()
	default:
		if b.isEndOfImportDecl(nextToken) {
			return st.CreateEmptyNode()
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL)
		return b.parseImportPrefixDecl()
	}
}

func (b *ballerinaParser) parseAsKeyword() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.AS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_AS_KEYWORD)
		return b.parseAsKeyword()
	}
}

func (b *ballerinaParser) parseImportPrefix() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.IDENTIFIER_TOKEN {
		identifier := b.consume()
		if b.isUnderscoreToken(identifier) {
			return b.getUnderscoreKeyword(identifier)
		}
		return identifier
	} else if isPredeclaredPrefix(nextToken.Kind()) {
		preDeclaredPrefix := b.consume()
		return st.CreateIdentifierToken(preDeclaredPrefix.Text(), preDeclaredPrefix.LeadingMinutiae(),
			preDeclaredPrefix.TrailingMinutiae())
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_IMPORT_PREFIX)
		return b.parseImportPrefix()
	}
}

func (b *ballerinaParser) parseTopLevelNodeWithQualifiers(metadata, publicQualifier st.STNode) st.STNode {
	res, _ := b.parseTopLevelNodeInner(metadata, publicQualifier, nil)
	return res
}

func (b *ballerinaParser) parseTopLevelNodeInner(metadata, publicQualifier st.STNode, qualifiers []st.STNode) (st.STNode, []st.STNode) {
	qualifiers = b.parseTopLevelQualifiers(qualifiers)
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.EOF_TOKEN:
		return b.createMissingSimpleVarDeclInnerWithQualifiers(metadata, publicQualifier, qualifiers, true), qualifiers
	case st.FUNCTION_KEYWORD:
		return b.parseFuncDefOrFuncTypeDesc(metadata, publicQualifier, qualifiers, false, false), qualifiers
	case st.TYPE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseModuleTypeDefinition(metadata, publicQualifier), qualifiers
	case st.CLASS_KEYWORD:
		return b.parseClassDefinition(metadata, publicQualifier, qualifiers), qualifiers
	case st.LISTENER_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseListenerDeclaration(metadata, publicQualifier), qualifiers
	case st.CONST_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseConstantDeclaration(metadata, publicQualifier), qualifiers
	case st.ANNOTATION_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		constKeyword := st.CreateEmptyNode()
		return b.parseAnnotationDeclaration(metadata, publicQualifier, constKeyword), qualifiers
	case st.IMPORT_KEYWORD:
		b.reportInvalidMetaData(metadata, "import declaration")
		b.reportInvalidQualifier(publicQualifier)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseImportDecl(), qualifiers
	case st.XMLNS_KEYWORD:
		b.reportInvalidMetaData(metadata, "XML namespace declaration")
		b.reportInvalidQualifier(publicQualifier)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseXMLNamespaceDeclaration(true), qualifiers
	case st.ENUM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseEnumDeclaration(metadata, publicQualifier), qualifiers
	case st.RESOURCE_KEYWORD, st.REMOTE_KEYWORD:
		b.reportInvalidQualifier(b.consume())
		return b.parseTopLevelNodeInner(metadata, publicQualifier, qualifiers)
	case st.IDENTIFIER_TOKEN:
		if b.isModuleVarDeclStart(1) {
			return b.parseModuleVarDeclInner(metadata, publicQualifier, qualifiers)
		}
		fallthrough
	default:
		if b.isPossibleServiceDecl(qualifiers) {
			return b.parseServiceDeclOrVarDecl(metadata, publicQualifier, qualifiers), qualifiers
		}
		if b.isTypeStartingToken(nextToken.Kind()) && (nextToken.Kind() != st.IDENTIFIER_TOKEN) {
			return b.parseModuleVarDeclInner(metadata, publicQualifier, qualifiers)
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER)
		if solution.Action == actionKeep {
			return b.parseModuleVarDeclInner(metadata, publicQualifier, qualifiers)
		}
		return b.parseTopLevelNodeInner(metadata, publicQualifier, qualifiers)
	}
}

func (b *ballerinaParser) parseModuleVarDecl(metadata st.STNode) st.STNode {
	var emptyList []st.STNode
	publicQualifier := st.CreateEmptyNode()
	res, _ := b.parseVariableDeclInner(metadata, publicQualifier, emptyList, emptyList, true)
	return res
}

func (b *ballerinaParser) parseModuleVarDeclInner(metadata st.STNode, publicQualifier st.STNode, topLevelQualifiers []st.STNode) (st.STNode, []st.STNode) {
	varDeclQuals, topLevelQualifiers := b.extractVarDeclQualifiers(topLevelQualifiers, true)
	res, _ := b.parseVariableDeclInner(metadata, publicQualifier, varDeclQuals, topLevelQualifiers, true)
	return res, topLevelQualifiers
}

func (b *ballerinaParser) extractVarDeclQualifiers(qualifiers []st.STNode, isModuleVar bool) ([]st.STNode, []st.STNode) {
	var varDeclQualList []st.STNode
	initialListSize := len(qualifiers)
	configurableQualIndex := (-1)
	i := 0
	for ; (i < 2) && (i < initialListSize); i++ {
		qualifierKind := qualifiers[0].Kind()
		if (!b.isSyntaxKindInList(varDeclQualList, qualifierKind)) && b.isModuleVarDeclQualifier(qualifierKind) {
			varDeclQualList = append(varDeclQualList, qualifiers[0])
			qualifiers = qualifiers[1:]
			if qualifierKind == st.CONFIGURABLE_KEYWORD {
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
				invalidQual := st.ToToken(varDeclQualList[i])
				configurableQual = st.CloneWithLeadingInvalidNodeMinutiae(configurableQual, invalidQual,
					b.getInvalidQualifierError(invalidQual.Kind()), (invalidQual).Text())
			} else if i > configurableQualIndex {
				invalidQual := st.ToToken(varDeclQualList[i])
				configurableQual = st.CloneWithTrailingInvalidNodeMinutiae(configurableQual, invalidQual,
					b.getInvalidQualifierError(invalidQual.Kind()), (invalidQual).Text())
			}
		}
		varDeclQualList = []st.STNode{configurableQual}
	}
	return varDeclQualList, qualifiers
}

func (b *ballerinaParser) getInvalidQualifierError(qualifierKind st.SyntaxKind) *common.DiagnosticErrorCode {
	if qualifierKind == st.FINAL_KEYWORD {
		return &common.ERROR_CONFIGURABLE_VAR_IMPLICITLY_FINAL
	}
	return &common.ERROR_QUALIFIER_NOT_ALLOWED
}

func (b *ballerinaParser) isModuleVarDeclQualifier(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.FINAL_KEYWORD, st.ISOLATED_KEYWORD, st.CONFIGURABLE_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) reportInvalidQualifier(qualifier st.STNode) {
	if (qualifier != nil) && (qualifier.Kind() != st.NONE) {
		b.addInvalidNodeToNextToken(qualifier, &common.ERROR_INVALID_QUALIFIER,
			st.ToToken(qualifier).Text())
	}
}

func (b *ballerinaParser) reportInvalidMetaData(metadata st.STNode, constructName string) {
	if (metadata != nil) && (metadata.Kind() != st.NONE) {
		b.addInvalidNodeToNextToken(metadata, &common.ERROR_INVALID_METADATA, constructName)
	}
}

func (b *ballerinaParser) reportInvalidQualifierList(qualifiers []st.STNode) {
	for _, qual := range qualifiers {
		b.addInvalidNodeToNextToken(qual, &common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qual).Text())
	}
}

func (b *ballerinaParser) reportInvalidStatementAnnots(annots st.STNode, qualifiers []st.STNode) {
	diagnosticErrorCode := common.ERROR_ANNOTATIONS_ATTACHED_TO_STATEMENT
	b.reportInvalidAnnotations(annots, qualifiers, diagnosticErrorCode)
}

func (b *ballerinaParser) reportInvalidExpressionAnnots(annots st.STNode, qualifiers []st.STNode) {
	diagnosticErrorCode := common.ERROR_ANNOTATIONS_ATTACHED_TO_EXPRESSION
	b.reportInvalidAnnotations(annots, qualifiers, diagnosticErrorCode)
}

func (b *ballerinaParser) reportInvalidAnnotations(annots st.STNode, qualifiers []st.STNode, errorCode common.DiagnosticErrorCode) {
	if b.isNodeListEmpty(annots) {
		return
	}
	if len(qualifiers) == 0 {
		b.addInvalidNodeToNextToken(annots, &errorCode)
	} else {
		b.updateFirstNodeInListWithLeadingInvalidNode(qualifiers, annots, &errorCode)
	}
}

func (b *ballerinaParser) isTopLevelQualifier(tokenKind st.SyntaxKind) bool {
	var nextNextToken st.STToken
	switch tokenKind {
	case st.FINAL_KEYWORD, // final-qualifier
		st.CONFIGURABLE_KEYWORD:
		return true
	case st.READONLY_KEYWORD:
		nextNextToken = b.getNextNextToken()
		switch nextNextToken.Kind() {
		case st.CLIENT_KEYWORD,
			st.SERVICE_KEYWORD,
			st.DISTINCT_KEYWORD,
			st.ISOLATED_KEYWORD,
			st.CLASS_KEYWORD:
			return true
		default:
			return false
		}
	case st.DISTINCT_KEYWORD:
		nextNextToken = b.getNextNextToken()
		switch nextNextToken.Kind() {
		case st.CLIENT_KEYWORD,
			st.SERVICE_KEYWORD,
			st.READONLY_KEYWORD,
			st.ISOLATED_KEYWORD,
			st.CLASS_KEYWORD:
			return true
		default:
			return false
		}
	default:
		return b.isTypeDescQualifier(tokenKind)
	}
}

func (b *ballerinaParser) isTypeDescQualifier(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.TRANSACTIONAL_KEYWORD, // func-type-dec, func-def
		st.ISOLATED_KEYWORD, // func-type-dec, object-type-desc, func-def, class-def, isolated-final-qual
		st.CLIENT_KEYWORD,   // object-type-desc, class-def
		st.ABSTRACT_KEYWORD, // object-type-desc(outdated)
		st.SERVICE_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) isObjectMemberQualifier(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.REMOTE_KEYWORD, // method-def, method-decl
		st.RESOURCE_KEYWORD, // resource-method-def
		st.FINAL_KEYWORD:
		return true
	default:
		return b.isTypeDescQualifier(tokenKind)
	}
}

func (b *ballerinaParser) isExprQualifier(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.TRANSACTIONAL_KEYWORD:
		nextNextToken := b.getNextNextToken()
		switch nextNextToken.Kind() {
		case st.CLIENT_KEYWORD,
			st.ABSTRACT_KEYWORD,
			st.ISOLATED_KEYWORD,
			st.OBJECT_KEYWORD,
			st.FUNCTION_KEYWORD:
			return true
		default:
			return false
		}
	default:
		return b.isTypeDescQualifier(tokenKind)
	}
}

func (b *ballerinaParser) parseTopLevelQualifiers(qualifiers []st.STNode) []st.STNode {
	for b.isTopLevelQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *ballerinaParser) parseTypeDescQualifiers(qualifiers []st.STNode) []st.STNode {
	for b.isTypeDescQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *ballerinaParser) parseObjectMemberQualifiers(qualifiers []st.STNode) []st.STNode {
	for b.isObjectMemberQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *ballerinaParser) parseExprQualifiers(qualifiers []st.STNode) []st.STNode {
	for b.isExprQualifier(b.peek().Kind()) {
		qualifier := b.consume()
		qualifiers = append(qualifiers, qualifier)
	}
	return qualifiers
}

func (b *ballerinaParser) parseOptionalRelativePath(isObjectMember bool) st.STNode {
	var resourcePath st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.DOT_TOKEN, st.IDENTIFIER_TOKEN, st.OPEN_BRACKET_TOKEN:
		resourcePath = b.parseRelativeResourcePath()
	case st.OPEN_PAREN_TOKEN:
		return st.CreateEmptyNodeList()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH)
		return b.parseOptionalRelativePath(isObjectMember)
	}
	if !isObjectMember {
		b.addInvalidNodeToNextToken(resourcePath, &common.ERROR_RESOURCE_PATH_IN_FUNCTION_DEFINITION)
		return st.CreateEmptyNodeList()
	}
	return resourcePath
}

func (b *ballerinaParser) parseFuncDefOrFuncTypeDesc(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, isObjectMember bool, isObjectTypeDesc bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE)
	functionKeyword := b.parseFunctionKeyword()
	funcDefOrType := b.parseFunctionKeywordRhs(metadata, visibilityQualifier, qualifiers, functionKeyword,
		isObjectMember, isObjectTypeDesc)
	return funcDefOrType
}

func (b *ballerinaParser) parseFunctionDefinition(metadata st.STNode, visibilityQualifier st.STNode, resourcePath st.STNode, qualifiers []st.STNode, functionKeyword st.STNode, name st.STNode, isObjectMember bool, isObjectTypeDesc bool) st.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	funcSignature := b.parseFuncSignature(false)
	funcDef := b.parseFuncDefOrMethodDeclEnd(metadata, visibilityQualifier, qualifiers, functionKeyword, name,
		resourcePath, funcSignature, isObjectMember, isObjectTypeDesc)
	b.endContext()
	return funcDef
}

func (b *ballerinaParser) parseFuncDefOrFuncTypeDescRhs(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, functionKeyword st.STNode, name st.STNode, isObjectMember bool, isObjectTypeDesc bool) st.STNode {
	switch b.peek().Kind() {
	case st.OPEN_PAREN_TOKEN,
		st.DOT_TOKEN,
		st.IDENTIFIER_TOKEN,
		st.OPEN_BRACKET_TOKEN:
		resourcePath := b.parseOptionalRelativePath(isObjectMember)
		return b.parseFunctionDefinition(metadata, visibilityQualifier, resourcePath, qualifiers, functionKeyword,
			name, isObjectMember, isObjectTypeDesc)
	case st.EQUAL_TOKEN,
		st.SEMICOLON_TOKEN:
		b.endContext()
		extractQualifiersList, qualifiers := b.extractVarDeclOrObjectFieldQualifiers(qualifiers, isObjectMember,
			isObjectTypeDesc)
		typeDesc := b.createFunctionTypeDescriptor(qualifiers, functionKeyword,
			st.CreateEmptyNode(), false)
		if isObjectMember {
			objectFieldQualNodeList := st.CreateNodeList(extractQualifiersList...)
			return b.parseObjectFieldRhs(metadata, visibilityQualifier, objectFieldQualNodeList, typeDesc, name,
				isObjectTypeDesc)
		}
		b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		funcTypeName := st.CreateSimpleNameReferenceNode(name)
		refNode, ok := funcTypeName.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("expected STSimpleNameReferenceNode")
			return st.CreateEmptyNode()
		}
		bindingPattern := b.createCaptureOrWildcardBP(refNode.Name)
		typedBindingPattern := st.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
		res, _ := b.parseVarDeclRhsInner(metadata, visibilityQualifier, extractQualifiersList, typedBindingPattern, true)
		return res
	default:
		token := b.peek()
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS)
		return b.parseFuncDefOrFuncTypeDescRhs(metadata, visibilityQualifier, qualifiers, functionKeyword, name,
			isObjectMember, isObjectTypeDesc)
	}
}

func (b *ballerinaParser) parseFunctionKeywordRhs(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, functionKeyword st.STNode, isObjectMember bool, isObjectTypeDesc bool) st.STNode {
	switch b.peek().Kind() {
	case st.IDENTIFIER_TOKEN:
		name := b.consume()
		return b.parseFuncDefOrFuncTypeDescRhs(metadata, visibilityQualifier, qualifiers, functionKeyword, name,
			isObjectMember, isObjectTypeDesc)
	case st.OPEN_PAREN_TOKEN:
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
				st.CreateEmptyNode(), isObjectMember,
				isObjectTypeDesc, false)
		}
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS)
		return b.parseFunctionKeywordRhs(metadata, visibilityQualifier, qualifiers, functionKeyword,
			isObjectMember, isObjectTypeDesc)
	}
}

func (b *ballerinaParser) isBindingPatternsStartToken(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.IDENTIFIER_TOKEN,
		st.OPEN_BRACKET_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.ERROR_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseFuncDefOrMethodDeclEnd(metadata st.STNode, visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode, name st.STNode, resourcePath st.STNode, funcSignature st.STNode, isObjectMember bool, isObjectTypeDesc bool) st.STNode {
	if !isObjectMember {
		return b.createFunctionDefinition(metadata, visibilityQualifier, qualifierList, functionKeyword, name,
			funcSignature)
	}
	hasResourcePath := (!b.isNodeListEmpty(resourcePath))
	hasResourceQual := b.isSyntaxKindInList(qualifierList, st.RESOURCE_KEYWORD)
	if hasResourceQual && (!hasResourcePath) {
		var relativePath []st.STNode
		relativePath = append(relativePath, st.CreateMissingToken(st.DOT_TOKEN, nil))
		resourcePath = st.CreateNodeList(relativePath...)
		var errorCode common.DiagnosticErrorCode
		if isObjectTypeDesc {
			errorCode = common.ERROR_MISSING_RESOURCE_PATH_IN_RESOURCE_ACCESSOR_DECLARATION
		} else {
			errorCode = common.ERROR_MISSING_RESOURCE_PATH_IN_RESOURCE_ACCESSOR_DEFINITION
		}
		name = st.AddDiagnostic(name, &errorCode)
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

func (b *ballerinaParser) createFunctionDefinition(metadata st.STNode, visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode, name st.STNode, funcSignature st.STNode) st.STNode {
	var validatedList []st.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, st.ToToken(qualifier).Text())
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = st.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		}
	}
	if visibilityQualifier != nil {
		validatedList = append([]st.STNode{visibilityQualifier}, validatedList...)
	}
	qualifiers := st.CreateNodeList(validatedList...)
	resourcePath := st.CreateEmptyNodeList()
	body := b.parseFunctionBody()
	return st.CreateFunctionDefinitionNode(st.FUNCTION_DEFINITION, metadata, qualifiers,
		functionKeyword, name, resourcePath, funcSignature, body)
}

func (b *ballerinaParser) validateMethodQualifiers(visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode) (st.STNode, st.STNode) {
	var validatedList []st.STNode
	hasRemoteQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, st.ToToken(qualifier).Text())
			continue
		}
		if qualifier.Kind() == st.REMOTE_KEYWORD {
			hasRemoteQual = true
			validatedList = append(validatedList, qualifier)
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = st.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		}
	}
	if visibilityQualifier != nil {
		if hasRemoteQual {
			b.updateFirstNodeInListWithLeadingInvalidNode(validatedList, visibilityQualifier,
				&common.ERROR_REMOTE_METHOD_HAS_A_VISIBILITY_QUALIFIER)
		} else {
			validatedList = append([]st.STNode{visibilityQualifier}, validatedList...)
		}
	}
	return st.CreateNodeList(validatedList...), functionKeyword
}

func (b *ballerinaParser) createMethodDefinition(metadata st.STNode, visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode, name st.STNode, funcSignature st.STNode) st.STNode {
	qualifiers, functionKeyword := b.validateMethodQualifiers(visibilityQualifier, qualifierList, functionKeyword)
	resourcePath := st.CreateEmptyNodeList()
	body := b.parseFunctionBody()
	return st.CreateFunctionDefinitionNode(st.OBJECT_METHOD_DEFINITION, metadata, qualifiers,
		functionKeyword, name, resourcePath, funcSignature, body)
}

func (b *ballerinaParser) createMethodDeclaration(metadata st.STNode, visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode, name st.STNode, funcSignature st.STNode) st.STNode {
	qualifiers, functionKeyword := b.validateMethodQualifiers(visibilityQualifier, qualifierList, functionKeyword)
	resourcePath := st.CreateEmptyNodeList()
	semicolon := b.parseSemicolon()
	return st.CreateMethodDeclarationNode(st.METHOD_DECLARATION, metadata, qualifiers,
		functionKeyword, name, resourcePath, funcSignature, semicolon)
}

func (b *ballerinaParser) createResourceAccessorDefnOrDecl(metadata st.STNode, visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode, name st.STNode, resourcePath st.STNode, funcSignature st.STNode, isObjectTypeDesc bool) st.STNode {
	var validatedList []st.STNode
	hasResourceQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, st.ToToken(qualifier).Text())
			continue
		}
		if qualifier.Kind() == st.RESOURCE_KEYWORD {
			hasResourceQual = true
			validatedList = append(validatedList, qualifier)
			continue
		}
		if b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			functionKeyword = st.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		}
	}
	if !hasResourceQual {
		validatedList = append(validatedList, st.CreateMissingToken(st.RESOURCE_KEYWORD, nil))
		functionKeyword = st.AddDiagnostic(functionKeyword, &common.ERROR_MISSING_RESOURCE_KEYWORD)
	}
	if visibilityQualifier != nil {
		b.updateFirstNodeInListWithLeadingInvalidNode(validatedList, visibilityQualifier,
			&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(visibilityQualifier).Text())
	}
	qualifiers := st.CreateNodeList(validatedList...)
	if isObjectTypeDesc {
		semicolon := b.parseSemicolon()
		return st.CreateMethodDeclarationNode(st.RESOURCE_ACCESSOR_DECLARATION, metadata,
			qualifiers, functionKeyword, name, resourcePath, funcSignature, semicolon)
	} else {
		body := b.parseFunctionBody()
		return st.CreateFunctionDefinitionNode(st.RESOURCE_ACCESSOR_DEFINITION, metadata,
			qualifiers, functionKeyword, name, resourcePath, funcSignature, body)
	}
}

func (b *ballerinaParser) parseFuncSignature(isParamNameOptional bool) st.STNode {
	openParenthesis := b.parseOpenParenthesis()
	parameters := b.parseParamList(isParamNameOptional)
	closeParenthesis := b.parseCloseParenthesis()
	b.endContext()
	returnTypeDesc := b.parseFuncReturnTypeDescriptor(isParamNameOptional)
	return st.CreateFunctionSignatureNode(openParenthesis, parameters, closeParenthesis, returnTypeDesc)
}

func (b *ballerinaParser) parseFunctionTypeDescRhs(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, functionKeyword st.STNode, funcSignature st.STNode, isObjectMember bool, isObjectTypeDesc bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACE_TOKEN, st.EQUAL_TOKEN:
		break
	case st.SEMICOLON_TOKEN, st.IDENTIFIER_TOKEN, st.OPEN_BRACKET_TOKEN:
		fallthrough
	default:
		return b.parseVarDeclWithFunctionType(metadata, visibilityQualifier, qualifiers, functionKeyword,
			funcSignature, isObjectMember, isObjectTypeDesc, true)
	}
	b.switchContext(common.PARSER_RULE_CONTEXT_FUNC_DEF)
	name := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_FUNCTION_NAME)
	fnSig, ok := funcSignature.(*st.STFunctionSignatureNode)
	if !ok {
		b.internalError("expected STFunctionSignatureNode")
		return st.CreateEmptyNode()
	}
	funcSignature = b.validateAndGetFuncParams(*fnSig)
	resourcePath := st.CreateEmptyNodeList()
	funcDef := b.parseFuncDefOrMethodDeclEnd(metadata, visibilityQualifier, qualifiers, functionKeyword,
		name, resourcePath, funcSignature, isObjectMember, isObjectTypeDesc)
	b.endContext()
	return funcDef
}

func (b *ballerinaParser) extractVarDeclOrObjectFieldQualifiers(qualifierList []st.STNode, isObjectMember bool, isObjectTypeDesc bool) ([]st.STNode, []st.STNode) {
	if isObjectMember {
		return b.extractObjectFieldQualifiers(qualifierList, isObjectTypeDesc)
	}
	return b.extractVarDeclQualifiers(qualifierList, false)
}

func (b *ballerinaParser) createFunctionTypeDescriptor(qualifierList []st.STNode, functionKeyword st.STNode, funcSignature st.STNode, hasFuncSignature bool) st.STNode {
	nodes := b.createFuncTypeQualNodeList(qualifierList, functionKeyword, hasFuncSignature)
	qualifierNodeList := nodes[0]
	functionKeyword = nodes[1]
	return st.CreateFunctionTypeDescriptorNode(qualifierNodeList, functionKeyword, funcSignature)
}

func (b *ballerinaParser) parseVarDeclWithFunctionType(metadata st.STNode, visibilityQualifier st.STNode, qualifierList []st.STNode, functionKeyword st.STNode, funcSignature st.STNode, isObjectMember bool, isObjectTypeDesc bool, hasFuncSignature bool) st.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	extractQualifiersList, qualifierList := b.extractVarDeclOrObjectFieldQualifiers(qualifierList, isObjectMember,
		isObjectTypeDesc)
	typeDesc := b.createFunctionTypeDescriptor(qualifierList, functionKeyword, funcSignature, hasFuncSignature)
	typeDesc = b.parseComplexTypeDescriptor(typeDesc,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	if isObjectMember {
		b.endContext()
		objectFieldQualNodeList := st.CreateNodeList(extractQualifiersList...)
		fieldName := b.parseVariableName()
		return b.parseObjectFieldRhs(metadata, visibilityQualifier, objectFieldQualNodeList, typeDesc, fieldName,
			isObjectTypeDesc)
	}
	typedBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	res, _ := b.parseVarDeclRhsInner(metadata, visibilityQualifier, extractQualifiersList, typedBindingPattern, true)
	return res
}

func (b *ballerinaParser) validateAndGetFuncParams(signature st.STFunctionSignatureNode) st.STNode {
	parameters := signature.Parameters
	paramCount := parameters.BucketCount()
	index := 0
	for ; index < paramCount; index++ {
		param := parameters.ChildInBucket(index)
		switch param.Kind() {
		case st.REQUIRED_PARAM:
			requiredParam, ok := param.(*st.STRequiredParameterNode)
			if !ok {
				b.internalError("expected STRequiredParameterNode")
				return st.CreateEmptyNode()
			}
			if b.isEmpty(requiredParam.ParamName) {
				break
			}
			continue
		case st.DEFAULTABLE_PARAM:
			defaultableParam, ok := param.(*st.STDefaultableParameterNode)
			if !ok {
				b.internalError("expected STDefaultableParameterNode")
				return st.CreateEmptyNode()
			}
			if b.isEmpty(defaultableParam.ParamName) {
				break
			}
			continue
		case st.REST_PARAM:
			restParam, ok := param.(*st.STRestParameterNode)
			if !ok {
				b.internalError("STRestParameterNode")
				return st.CreateEmptyNode()
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
	return st.CreateFunctionSignatureNode(signature.OpenParenToken, updatedParams,
		signature.CloseParenToken, signature.ReturnTypeDesc)
}

func (b *ballerinaParser) getUpdatedParamList(parameters st.STNode, index int) st.STNode {
	paramCount := parameters.BucketCount()
	newIndex := 0
	var newParams []st.STNode
	for ; newIndex < index; newIndex++ {
		newParams = append(newParams, parameters.ChildInBucket(index))
	}
	for ; newIndex < paramCount; newIndex++ {
		param := parameters.ChildInBucket(newIndex)
		paramName := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		switch param.Kind() {
		case st.REQUIRED_PARAM:
			requiredParam, ok := param.(*st.STRequiredParameterNode)
			if !ok {
				b.internalError("expected STRequiredParameterNode")
				return st.CreateEmptyNode()
			}
			if b.isEmpty(requiredParam.ParamName) {
				param = st.CreateRequiredParameterNode(requiredParam.Annotations,
					requiredParam.TypeName, paramName)
			}
		case st.DEFAULTABLE_PARAM:
			defaultableParam, ok := param.(*st.STDefaultableParameterNode)
			if !ok {
				b.internalError("expected STDefaultableParameterNode")
				return st.CreateEmptyNode()
			}
			if b.isEmpty(defaultableParam.ParamName) {
				param = st.CreateDefaultableParameterNode(defaultableParam.Annotations, defaultableParam.TypeName,
					paramName, defaultableParam.EqualsToken, defaultableParam.Expression)
			}
		case st.REST_PARAM:
			restParam, ok := param.(*st.STRestParameterNode)
			if !ok {
				b.internalError("expected STRestParameterNode")
				return st.CreateEmptyNode()
			}
			if b.isEmpty(restParam.ParamName) {
				param = st.CreateRestParameterNode(restParam.Annotations, restParam.TypeName,
					restParam.EllipsisToken, paramName)
			}
		default:
		}
		newParams = append(newParams, param)
	}
	return st.CreateNodeList(newParams...)
}

func (b *ballerinaParser) isEmpty(node st.STNode) bool {
	return (!st.IsSTNodePresent(node))
}

func (b *ballerinaParser) parseFunctionKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.FUNCTION_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD)
		return b.parseFunctionKeyword()
	}
}

func (b *ballerinaParser) parseFunctionName() st.STNode {
	token := b.peek()
	if token.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNC_NAME)
		return b.parseFunctionName()
	}
}

func (b *ballerinaParser) parseArgListOpenParenthesis() st.STNode {
	return b.parseOpenParenthesisInner(common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN)
}

func (b *ballerinaParser) parseOpenParenthesis() st.STNode {
	return b.parseOpenParenthesisInner(common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS)
}

func (b *ballerinaParser) parseOpenParenthesisInner(ctx common.ParserRuleContext) st.STNode {
	token := b.peek()
	if token.Kind() == st.OPEN_PAREN_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, ctx)
		return b.parseOpenParenthesisInner(ctx)
	}
}

func (b *ballerinaParser) parseArgListCloseParenthesis() st.STNode {
	return b.parseCloseParenthesisInner(common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN)
}

func (b *ballerinaParser) parseCloseParenthesis() st.STNode {
	return b.parseCloseParenthesisInner(common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS)
}

func (b *ballerinaParser) parseCloseParenthesisInner(ctx common.ParserRuleContext) st.STNode {
	token := b.peek()
	if token.Kind() == st.CLOSE_PAREN_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, ctx)
		return b.parseCloseParenthesisInner(ctx)
	}
}

func (b *ballerinaParser) parseParamList(isParamNameOptional bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_PARAM_LIST)
	token := b.peek()
	if b.isEndOfParametersList(token.Kind()) {
		return st.CreateEmptyNodeList()
	}
	var paramsList []st.STNode
	b.startContext(common.PARSER_RULE_CONTEXT_REQUIRED_PARAM)
	firstParam := b.parseParameterInner(st.REQUIRED_PARAM, isParamNameOptional)
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
		if prevParamKind == st.DEFAULTABLE_PARAM {
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
	return st.CreateNodeList(paramsList...)
}

func (b *ballerinaParser) validateParamOrder(param st.STNode, prevParamKind st.SyntaxKind) diagnostics.DiagnosticCode {
	if prevParamKind == st.REST_PARAM {
		return &common.ERROR_PARAMETER_AFTER_THE_REST_PARAMETER
	} else if (prevParamKind == st.DEFAULTABLE_PARAM) && (param.Kind() == st.REQUIRED_PARAM) {
		return &common.ERROR_REQUIRED_PARAMETER_AFTER_THE_DEFAULTABLE_PARAMETER
	}
	return nil
}

func (b *ballerinaParser) isSyntaxKindInList(nodeList []st.STNode, kind st.SyntaxKind) bool {
	for _, node := range nodeList {
		if node.Kind() == kind {
			return true
		}
	}
	return false
}

func (b *ballerinaParser) isPossibleServiceDecl(nodeList []st.STNode) bool {
	if len(nodeList) == 0 {
		return false
	}
	firstElement := nodeList[0]
	switch firstElement.Kind() {
	case st.SERVICE_KEYWORD:
		return true
	case st.ISOLATED_KEYWORD:
		return ((len(nodeList) > 1) && (nodeList[1].Kind() == st.SERVICE_KEYWORD))
	default:
		return false
	}
}

func (b *ballerinaParser) parseParameterRhs() st.STNode {
	return b.parseParameterRhsInner(b.peek().Kind())
}

func (b *ballerinaParser) parseParameterRhsInner(tokenKind st.SyntaxKind) st.STNode {
	switch tokenKind {
	case st.COMMA_TOKEN:
		return b.consume()
	case st.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_PARAM_END)
		return b.parseParameterRhs()
	}
}

func (b *ballerinaParser) parseParameter(annots st.STNode, prevParamKind st.SyntaxKind, isParamNameOptional bool) st.STNode {
	var inclusionSymbol st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ASTERISK_TOKEN:
		inclusionSymbol = b.consume()
	case st.IDENTIFIER_TOKEN:
		inclusionSymbol = st.CreateEmptyNode()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			inclusionSymbol = st.CreateEmptyNode()
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION)
		if solution.Action == actionKeep {
			inclusionSymbol = st.CreateEmptyNodeList()
			break
		}
		return b.parseParameter(annots, prevParamKind, isParamNameOptional)
	}
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	return b.parseAfterParamType(prevParamKind, annots, inclusionSymbol, ty, isParamNameOptional)
}

func (b *ballerinaParser) parseParameterInner(prevParamKind st.SyntaxKind, isParamNameOptional bool) st.STNode {
	var annots st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.AT_TOKEN:
		annots = b.parseOptionalAnnotations()
	case st.ASTERISK_TOKEN, st.IDENTIFIER_TOKEN:
		annots = st.CreateEmptyNodeList()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			annots = st.CreateEmptyNodeList()
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PARAMETER_START)
		if solution.Action == actionKeep {
			annots = st.CreateEmptyNodeList()
			break
		}
		return b.parseParameterInner(prevParamKind, isParamNameOptional)
	}
	return b.parseParameter(annots, prevParamKind, isParamNameOptional)
}

func (b *ballerinaParser) parseAfterParamType(prevParamKind st.SyntaxKind, annots st.STNode, inclusionSymbol st.STNode, ty st.STNode, isParamNameOptional bool) st.STNode {
	var paramName st.STNode
	token := b.peek()
	switch token.Kind() {
	case st.ELLIPSIS_TOKEN:
		if inclusionSymbol != nil {
			ty = st.CloneWithLeadingInvalidNodeMinutiae(ty, inclusionSymbol,
				&common.REST_PARAMETER_CANNOT_BE_INCLUDED_RECORD_PARAMETER)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_REST_PARAM)
		ellipsis := b.parseEllipsis()
		if isParamNameOptional && (b.peek().Kind() != st.IDENTIFIER_TOKEN) {
			paramName = st.CreateEmptyNode()
		} else {
			paramName = b.parseVariableName()
		}
		return st.CreateRestParameterNode(annots, ty, ellipsis, paramName)
	case st.IDENTIFIER_TOKEN:
		paramName = b.parseVariableName()
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	case st.EQUAL_TOKEN:
		if !isParamNameOptional {
			break
		}
		paramName = st.CreateEmptyNode()
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	default:
		if !isParamNameOptional {
			break
		}
		paramName = st.CreateEmptyNode()
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE)
	return b.parseAfterParamType(prevParamKind, annots, inclusionSymbol, ty, false)
}

func (b *ballerinaParser) parseEllipsis() st.STNode {
	token := b.peek()
	if token.Kind() == st.ELLIPSIS_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ELLIPSIS)
		return b.parseEllipsis()
	}
}

func (b *ballerinaParser) parseParameterRhsWithAnnots(prevParamKind st.SyntaxKind, annots st.STNode, inclusionSymbol st.STNode, ty st.STNode, paramName st.STNode) st.STNode {
	nextToken := b.peek()
	if b.isEndOfParameter(nextToken.Kind()) {
		if inclusionSymbol != nil {
			return st.CreateIncludedRecordParameterNode(annots, inclusionSymbol, ty, paramName)
		} else {
			return st.CreateRequiredParameterNode(annots, ty, paramName)
		}
	} else if nextToken.Kind() == st.EQUAL_TOKEN {
		if prevParamKind == st.REQUIRED_PARAM {
			b.switchContext(common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM)
		}
		equal := b.parseAssignOp()
		expr := b.parseInferredTypeDescDefaultOrExpression()
		if inclusionSymbol != nil {
			ty = st.CloneWithLeadingInvalidNodeMinutiae(ty, inclusionSymbol,
				&common.ERROR_DEFAULTABLE_PARAMETER_CANNOT_BE_INCLUDED_RECORD_PARAMETER)
		}
		return st.CreateDefaultableParameterNode(annots, ty, paramName, equal, expr)
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_PARAMETER_NAME_RHS)
		return b.parseParameterRhsWithAnnots(prevParamKind, annots, inclusionSymbol, ty, paramName)
	}
}

func (b *ballerinaParser) parseComma() st.STNode {
	token := b.peek()
	if token.Kind() == st.COMMA_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COMMA)
		return b.parseComma()
	}
}

func (b *ballerinaParser) parseFuncReturnTypeDescriptor(isFuncTypeDesc bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACE_TOKEN,
		st.EQUAL_TOKEN:
		return st.CreateEmptyNode()
	case st.RETURNS_KEYWORD:
		break
	case st.IDENTIFIER_TOKEN:
		if (!isFuncTypeDesc) || b.isSafeMissingReturnsParse() {
			break
		}
		fallthrough
	default:
		nextNextToken := b.getNextNextToken()
		if nextNextToken.Kind() == st.RETURNS_KEYWORD {
			break
		}
		return st.CreateEmptyNode()
	}
	returnsKeyword := b.parseReturnsKeyword()
	annot := b.parseOptionalAnnotations()
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC)
	return st.CreateReturnTypeDescriptorNode(returnsKeyword, annot, ty)
}

func (b *ballerinaParser) isSafeMissingReturnsParse() bool {
	for _, context := range b.errorHandler.GetContextStack() {
		if !b.isSafeMissingReturnsParseCtx(context) {
			return false
		}
	}
	return true
}

func (b *ballerinaParser) isSafeMissingReturnsParseCtx(ctx common.ParserRuleContext) bool {
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

func (b *ballerinaParser) parseReturnsKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.RETURNS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD)
		return b.parseReturnsKeyword()
	}
}

func (b *ballerinaParser) parseTypeDescriptor(context common.ParserRuleContext) st.STNode {
	return b.parseTypeDescriptorWithinContext(nil, context, false, false, typePrecedenceDefault)
}

func (b *ballerinaParser) parseTypeDescriptorWithPrecedence(context common.ParserRuleContext, precedence typePrecedence) st.STNode {
	return b.parseTypeDescriptorWithinContext(nil, context, false, false, precedence)
}

func (b *ballerinaParser) parseTypeDescriptorWithQualifier(qualifiers []st.STNode, context common.ParserRuleContext) st.STNode {
	return b.parseTypeDescriptorWithinContext(qualifiers, context, false, false, typePrecedenceDefault)
}

func (b *ballerinaParser) parseTypeDescriptorInExpression(isInConditionalExpr bool) st.STNode {
	return b.parseTypeDescriptorWithinContext(nil, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION, false, isInConditionalExpr,
		typePrecedenceDefault)
}

func (b *ballerinaParser) parseTypeDescriptorWithoutQualifiers(context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence typePrecedence) st.STNode {
	return b.parseTypeDescriptorWithinContext(nil, context, isTypedBindingPattern, isInConditionalExpr, precedence)
}

func (b *ballerinaParser) parseTypeDescriptorWithinContext(qualifiers []st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence typePrecedence) st.STNode {
	b.startContext(context)
	typeDesc := b.parseTypeDescriptorInner(qualifiers, context, isTypedBindingPattern, isInConditionalExpr,
		precedence)
	b.endContext()
	return typeDesc
}

func (b *ballerinaParser) parseTypeDescriptorInner(qualifiers []st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence typePrecedence) st.STNode {
	typeDesc := b.parseTypeDescriptorInternal(qualifiers, context, isInConditionalExpr)
	if ((typeDesc.Kind() == st.VAR_TYPE_DESC) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY) {
		var missingToken st.STNode
		missingToken = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		missingToken = st.CloneWithLeadingInvalidNodeMinutiae(missingToken, typeDesc,
			&common.ERROR_INVALID_USAGE_OF_VAR)
		typeDesc = st.CreateSimpleNameReferenceNode(missingToken.(st.STToken))
	}
	return b.parseComplexTypeDescriptorInternal(typeDesc, context, isTypedBindingPattern, precedence)
}

func (b *ballerinaParser) parseComplexTypeDescriptor(typeDesc st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool) st.STNode {
	b.startContext(context)
	complexTypeDesc := b.parseComplexTypeDescriptorInternal(typeDesc, context, isTypedBindingPattern,
		typePrecedenceDefault)
	b.endContext()
	return complexTypeDesc
}

func (b *ballerinaParser) parseComplexTypeDescriptorInternal(typeDesc st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, precedence typePrecedence) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.QUESTION_MARK_TOKEN:
		if precedence.isHigherThanOrEqual(typePrecedenceArrayOrOptional) {
			return typeDesc
		}
		isPossibleOptionalType := true
		nextNextToken := b.getNextNextToken()
		if ((context == common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION) && (!b.isValidTypeContinuationToken(nextNextToken))) && b.isValidExprStart(nextNextToken.Kind()) {
			if nextNextToken.Kind() == st.OPEN_BRACE_TOKEN {
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
	case st.OPEN_BRACKET_TOKEN:
		if isTypedBindingPattern {
			return typeDesc
		}
		if precedence.isHigherThanOrEqual(typePrecedenceArrayOrOptional) {
			return typeDesc
		}
		arrayTypeDesc := b.parseArrayTypeDescriptor(typeDesc)
		return b.parseComplexTypeDescriptorInternal(arrayTypeDesc, context, false, precedence)
	case st.PIPE_TOKEN:
		if precedence.isHigherThanOrEqual(typePrecedenceUnion) {
			return typeDesc
		}
		newTypeDesc := b.parseUnionTypeDescriptor(typeDesc, context, isTypedBindingPattern)
		return b.parseComplexTypeDescriptorInternal(newTypeDesc, context, isTypedBindingPattern, precedence)
	case st.BITWISE_AND_TOKEN:
		if precedence.isHigherThanOrEqual(typePrecedenceIntersection) {
			return typeDesc
		}
		newTypeDesc := b.parseIntersectionTypeDescriptor(typeDesc, context, isTypedBindingPattern)
		return b.parseComplexTypeDescriptorInternal(newTypeDesc, context, isTypedBindingPattern, precedence)
	default:
		return typeDesc
	}
}

func (b *ballerinaParser) isValidTypeContinuationToken(token st.STToken) bool {
	switch token.Kind() {
	case st.QUESTION_MARK_TOKEN, st.OPEN_BRACKET_TOKEN, st.PIPE_TOKEN, st.BITWISE_AND_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) validateForUsageOfVar(typeDesc st.STNode) st.STNode {
	if typeDesc.Kind() != st.VAR_TYPE_DESC {
		return typeDesc
	}
	var missingToken st.STNode
	missingToken = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
	missingToken = st.CloneWithLeadingInvalidNodeMinutiae(missingToken, typeDesc,
		&common.ERROR_INVALID_USAGE_OF_VAR)
	return st.CreateSimpleNameReferenceNode(missingToken)
}

func (b *ballerinaParser) parseTypeDescriptorInternal(qualifiers []st.STNode, context common.ParserRuleContext, isInConditionalExpr bool) st.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	if b.isQualifiedIdentifierPredeclaredPrefix(nextToken.Kind()) {
		return b.parseQualifiedTypeRefOrTypeDesc(qualifiers, isInConditionalExpr)
	}
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTypeReferenceInner(isInConditionalExpr)
	case st.RECORD_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRecordTypeDescriptor()
	case st.OBJECT_KEYWORD:
		objectTypeQualifiers := b.createObjectTypeQualNodeList(qualifiers)
		return b.parseObjectTypeDescriptor(b.consume(), objectTypeQualifiers)
	case st.OPEN_PAREN_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseNilOrParenthesisedTypeDesc()
	case st.MAP_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMapTypeDescriptor(b.consume())
	case st.STREAM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStreamTypeDescriptor(b.consume())
	case st.TABLE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTableTypeDescriptor(b.consume())
	case st.FUNCTION_KEYWORD:
		return b.parseFunctionTypeDesc(qualifiers)
	case st.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTupleTypeDesc()
	case st.DISTINCT_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		distinctKeyword := b.consume()
		return b.parseDistinctTypeDesc(distinctKeyword, context)
	case st.TRANSACTION_KEYWORD:
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
	if solution.Action == actionKeep {
		b.reportInvalidQualifierList(qualifiers)
		return b.parseSingletonTypeDesc()
	}
	return b.parseTypeDescriptorInternal(qualifiers, context, isInConditionalExpr)
}

func (b *ballerinaParser) parseTypeDescriptorInternalWithPrecedence(qualifiers []st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool, isInConditionalExpr bool, precedence typePrecedence) st.STNode {
	typeDesc := b.parseTypeDescriptorInternal(qualifiers, context, isInConditionalExpr)

	// var is parsed as a built-in simple type. However, since var is not allowed everywhere,
	// validate it here. This is done to give better error messages.
	if ((typeDesc.Kind() == st.VAR_TYPE_DESC) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)) && (context != common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY) {
		var missingToken st.STNode
		missingToken = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		missingToken = st.CloneWithLeadingInvalidNodeMinutiae(missingToken, typeDesc,
			&common.ERROR_INVALID_USAGE_OF_VAR)
		typeDesc = st.CreateSimpleNameReferenceNode(missingToken.(st.STToken))
	}

	return b.parseComplexTypeDescriptorInternal(typeDesc, context, isTypedBindingPattern, precedence)
}

func (b *ballerinaParser) getTypeDescRecoveryCtx(qualifiers []st.STNode) common.ParserRuleContext {
	if len(qualifiers) == 0 {
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	}
	lastQualifier := b.getLastNodeInList(qualifiers)
	switch lastQualifier.Kind() {
	case st.ISOLATED_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED
	case st.TRANSACTIONAL_KEYWORD:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC
	default:
		return common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR
	}
}

func (b *ballerinaParser) parseQualifiedIdentWithTransactionPrefix(context common.ParserRuleContext) st.STNode {
	transactionKeyword := b.consume()
	identifier := st.CreateIdentifierToken(transactionKeyword.Text(),
		transactionKeyword.LeadingMinutiae(), transactionKeyword.TrailingMinutiae())
	colon := st.CreateMissingTokenWithDiagnostics(st.COLON_TOKEN,
		&common.ERROR_MISSING_COLON_TOKEN)
	varOrFuncName := b.parseIdentifier(context)
	return b.createQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
}

func (b *ballerinaParser) parseQualifiedTypeRefOrTypeDesc(qualifiers []st.STNode, isInConditionalExpr bool) st.STNode {
	preDeclaredPrefix := b.consume()
	nextNextToken := b.getNextNextToken()
	if (preDeclaredPrefix.Kind() == st.TRANSACTION_KEYWORD) || (nextNextToken.Kind() == st.IDENTIFIER_TOKEN) {
		b.reportInvalidQualifierList(qualifiers)
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	var context common.ParserRuleContext
	switch preDeclaredPrefix.Kind() {
	case st.MAP_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF
	case st.OBJECT_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF
	case st.STREAM_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF
	case st.TABLE_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF
	default:
		if isParameterizedTypeToken(preDeclaredPrefix.Kind()) {
			context = common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF
		} else {
			context = common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF
		}
	}
	solution := b.recoverWithBlockContext(b.peek(), context)
	if solution.Action == actionKeep {
		b.reportInvalidQualifierList(qualifiers)
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	return b.parseTypeDescStartWithPredeclPrefix(preDeclaredPrefix, qualifiers)
}

func (b *ballerinaParser) parseTypeDescStartWithPredeclPrefix(preDeclaredPrefix st.STToken, qualifiers []st.STNode) st.STNode {
	switch preDeclaredPrefix.Kind() {
	case st.MAP_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMapTypeDescriptor(preDeclaredPrefix)
	case st.OBJECT_KEYWORD:
		objectTypeQualifiers := b.createObjectTypeQualNodeList(qualifiers)
		return b.parseObjectTypeDescriptor(preDeclaredPrefix, objectTypeQualifiers)
	case st.STREAM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStreamTypeDescriptor(preDeclaredPrefix)
	case st.TABLE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTableTypeDescriptor(preDeclaredPrefix)
	default:
		if isParameterizedTypeToken(preDeclaredPrefix.Kind()) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseParameterizedTypeDescriptor(preDeclaredPrefix)
		}
		return b.createBuiltinSimpleNameReference(preDeclaredPrefix)
	}
}

func (b *ballerinaParser) parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix st.STToken, isInConditionalExpr bool) st.STNode {
	identifier := st.CreateIdentifierToken(preDeclaredPrefix.Text(),
		preDeclaredPrefix.LeadingMinutiae(), preDeclaredPrefix.TrailingMinutiae())
	return b.parseQualifiedIdentifierNode(identifier, isInConditionalExpr)
}

func (b *ballerinaParser) parseDistinctTypeDesc(distinctKeyword st.STNode, context common.ParserRuleContext) st.STNode {
	typeDesc := b.parseTypeDescriptorWithPrecedence(context, typePrecedenceDistinct)
	return st.CreateDistinctTypeDescriptorNode(distinctKeyword, typeDesc)
}

func (b *ballerinaParser) parseNilOrParenthesisedTypeDesc() st.STNode {
	openParen := b.parseOpenParenthesis()
	return b.parseNilOrParenthesisedTypeDescRhs(openParen)
}

func (b *ballerinaParser) parseNilOrParenthesisedTypeDescRhs(openParen st.STNode) st.STNode {
	var closeParen st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.CLOSE_PAREN_TOKEN:
		closeParen = b.parseCloseParenthesis()
		return st.CreateNilTypeDescriptorNode(openParen, closeParen)
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			typedesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS)
			closeParen = b.parseCloseParenthesis()
			return st.CreateParenthesisedTypeDescriptorNode(openParen, typedesc, closeParen)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS)
		return b.parseNilOrParenthesisedTypeDescRhs(openParen)
	}
}

func (b *ballerinaParser) parseSimpleTypeInTerminalExpr() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION)
	simpleTypeDescriptor := b.parseSimpleTypeDescriptor()
	b.endContext()
	return simpleTypeDescriptor
}

func (b *ballerinaParser) parseSimpleTypeDescriptor() st.STNode {
	nextToken := b.peek()
	if isSimpleType(nextToken.Kind()) {
		token := b.consume()
		return b.createBuiltinSimpleNameReference(token)
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR)
		return b.parseSimpleTypeDescriptor()
	}
}

func (b *ballerinaParser) parseFunctionBody() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.EQUAL_TOKEN:
		return b.parseExternalFunctionBody()
	case st.OPEN_BRACE_TOKEN:
		return b.parseFunctionBodyBlock(false)
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		return b.parseExpressionFuncBody(false, false)
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNC_BODY)
		return b.parseFunctionBody()
	}
}

func (b *ballerinaParser) parseFunctionBodyBlock(isAnonFunc bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK)
	openBrace := b.parseOpenBrace()
	token := b.peek()
	firstStmtList := make([]st.STNode, 0)
	workers := make([]st.STNode, 0)
	secondStmtList := make([]st.STNode, 0)
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
			if stmt.Kind() != st.NAMED_WORKER_DECLARATION {
				firstStmtList = append(firstStmtList, stmt)
				break
			}
			currentCtx = common.PARSER_RULE_CONTEXT_NAMED_WORKERS
			hasNamedWorkers = true
			fallthrough
		case common.PARSER_RULE_CONTEXT_NAMED_WORKERS:
			if stmt.Kind() == st.NAMED_WORKER_DECLARATION {
				workers = append(workers, stmt)
				break
			}
			currentCtx = common.PARSER_RULE_CONTEXT_DEFAULT_WORKER
			fallthrough
		case common.PARSER_RULE_CONTEXT_DEFAULT_WORKER:
			fallthrough
		default:
			if stmt.Kind() == st.NAMED_WORKER_DECLARATION {
				b.updateLastNodeInListWithInvalidNode(secondStmtList, stmt,
					&common.ERROR_NAMED_WORKER_NOT_ALLOWED_HERE)
				break
			}
			secondStmtList = append(secondStmtList, stmt)
		}
		token = b.peek()
	}
	var namedWorkersList st.STNode
	var statements st.STNode
	if hasNamedWorkers {
		workerInitStatements := st.CreateNodeList(firstStmtList...)
		namedWorkers := st.CreateNodeList(workers...)
		namedWorkersList = st.CreateNamedWorkerDeclarator(workerInitStatements, namedWorkers)
		statements = st.CreateNodeList(secondStmtList...)
	} else {
		namedWorkersList = st.CreateEmptyNode()
		statements = st.CreateNodeList(firstStmtList...)
	}
	closeBrace := b.parseCloseBrace()
	var semicolon st.STNode
	if isAnonFunc {
		semicolon = st.CreateEmptyNode()
	} else {
		semicolon = b.parseOptionalSemicolon()
	}
	b.endContext()
	return st.CreateFunctionBodyBlockNode(openBrace, namedWorkersList, statements, closeBrace,
		semicolon)
}

func (b *ballerinaParser) isEndOfFuncBodyBlock(nextTokenKind st.SyntaxKind, isAnonFunc bool) bool {
	if isAnonFunc {
		switch nextTokenKind {
		case st.CLOSE_BRACE_TOKEN, st.CLOSE_PAREN_TOKEN, st.CLOSE_BRACKET_TOKEN,
			st.OPEN_BRACE_TOKEN, st.SEMICOLON_TOKEN, st.COMMA_TOKEN,
			st.PUBLIC_KEYWORD, st.EOF_TOKEN, st.EQUAL_TOKEN, st.BACKTICK_TOKEN:
			return true
		default:
			break
		}
	}
	return b.isEndOfStatements()
}

func (b *ballerinaParser) isEndOfRecordTypeNode() bool {
	return b.isEndOfModuleLevelNode()
}

func (b *ballerinaParser) isEndOfObjectTypeNode() bool {
	return b.isEndOfModuleLevelNodeInner(1, true)
}

func (b *ballerinaParser) isEndOfStatements() bool {
	switch b.peek().Kind() {
	case st.RESOURCE_KEYWORD:
		return true
	default:
		return b.isEndOfModuleLevelNode()
	}
}

func (b *ballerinaParser) isEndOfModuleLevelNode() bool {
	return b.isEndOfModuleLevelNodeInner(1, false)
}

func (b *ballerinaParser) isEndOfModuleLevelNodeInner(peekIndex int, isObject bool) bool {
	switch b.peekN(peekIndex).Kind() {
	case st.EOF_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.CLOSE_BRACE_PIPE_TOKEN,
		st.IMPORT_KEYWORD,
		st.ANNOTATION_KEYWORD,
		st.LISTENER_KEYWORD,
		st.CLASS_KEYWORD:
		return true
	case st.SERVICE_KEYWORD:
		return b.isServiceDeclStart(1)
	case st.PUBLIC_KEYWORD:
		return ((!isObject) && b.isEndOfModuleLevelNodeInner(peekIndex+1, false))
	case st.FUNCTION_KEYWORD:
		if isObject {
			return false
		}
		return ((b.peekN(peekIndex+1).Kind() == st.IDENTIFIER_TOKEN) && (b.peekN(peekIndex+2).Kind() == st.OPEN_PAREN_TOKEN))
	default:
		return false
	}
}

func (b *ballerinaParser) isEndOfParameter(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.CLOSE_PAREN_TOKEN,
		st.CLOSE_BRACKET_TOKEN,
		st.SEMICOLON_TOKEN,
		st.COMMA_TOKEN,
		st.RETURNS_KEYWORD,
		st.TYPE_KEYWORD,
		st.IF_KEYWORD,
		st.WHILE_KEYWORD,
		st.DO_KEYWORD,
		st.AT_TOKEN:
		return true
	default:
		return b.isEndOfModuleLevelNode()
	}
}

func (b *ballerinaParser) isEndOfParametersList(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.CLOSE_PAREN_TOKEN,
		st.SEMICOLON_TOKEN,
		st.RETURNS_KEYWORD,
		st.TYPE_KEYWORD,
		st.IF_KEYWORD,
		st.WHILE_KEYWORD,
		st.DO_KEYWORD,
		st.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return b.isEndOfModuleLevelNode()
	}
}

func (b *ballerinaParser) parseStatementStartIdentifier() st.STNode {
	return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME)
}

func (b *ballerinaParser) parseVariableName() st.STNode {
	token := b.peek()
	if token.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_VARIABLE_NAME)
		return b.parseVariableName()
	}
}

func (b *ballerinaParser) parseOpenBrace() st.STNode {
	token := b.peek()
	if token.Kind() == st.OPEN_BRACE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OPEN_BRACE)
		return b.parseOpenBrace()
	}
}

func (b *ballerinaParser) parseCloseBrace() st.STNode {
	token := b.peek()
	if token.Kind() == st.CLOSE_BRACE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSE_BRACE)
		return b.parseCloseBrace()
	}
}

func (b *ballerinaParser) parseExternalFunctionBody() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY)
	assign := b.parseAssignOp()
	return b.parseExternalFuncBodyRhs(assign)
}

func (b *ballerinaParser) parseExternalFuncBodyRhs(assign st.STNode) st.STNode {
	var annotation st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.AT_TOKEN:
		annotation = b.parseAnnotations()
	case st.EXTERNAL_KEYWORD:
		annotation = st.CreateEmptyNodeList()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS)
		return b.parseExternalFuncBodyRhs(assign)
	}
	externalKeyword := b.parseExternalKeyword()
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateExternalFunctionBodyNode(assign, annotation, externalKeyword, semicolon)
}

func (b *ballerinaParser) parseSemicolon() st.STNode {
	token := b.peek()
	if token.Kind() == st.SEMICOLON_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SEMICOLON)
		return b.parseSemicolon()
	}
}

func (b *ballerinaParser) parseOptionalSemicolon() st.STNode {
	token := b.peek()
	if token.Kind() == st.SEMICOLON_TOKEN {
		return b.consume()
	}
	return st.CreateEmptyNode()
}

func (b *ballerinaParser) parseExternalKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.EXTERNAL_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD)
		return b.parseExternalKeyword()
	}
}

func (b *ballerinaParser) parseAssignOp() st.STNode {
	token := b.peek()
	if token.Kind() == st.EQUAL_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ASSIGN_OP)
		return b.parseAssignOp()
	}
}

func (b *ballerinaParser) parseBinaryOperator() st.STNode {
	token := b.peek()
	if b.isBinaryOperator(token.Kind()) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR)
		return b.parseBinaryOperator()
	}
}

func (b *ballerinaParser) isBinaryOperator(kind st.SyntaxKind) bool {
	switch kind {
	case st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.SLASH_TOKEN,
		st.ASTERISK_TOKEN,
		st.GT_TOKEN,
		st.LT_TOKEN,
		st.DOUBLE_EQUAL_TOKEN,
		st.TRIPPLE_EQUAL_TOKEN,
		st.LT_EQUAL_TOKEN,
		st.GT_EQUAL_TOKEN,
		st.NOT_EQUAL_TOKEN,
		st.NOT_DOUBLE_EQUAL_TOKEN,
		st.BITWISE_AND_TOKEN,
		st.BITWISE_XOR_TOKEN,
		st.PIPE_TOKEN,
		st.LOGICAL_AND_TOKEN,
		st.LOGICAL_OR_TOKEN,
		st.PERCENT_TOKEN,
		st.DOUBLE_LT_TOKEN,
		st.DOUBLE_GT_TOKEN,
		st.TRIPPLE_GT_TOKEN,
		st.ELLIPSIS_TOKEN,
		st.DOUBLE_DOT_LT_TOKEN,
		st.ELVIS_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) getOpPrecedence(binaryOpKind st.SyntaxKind) operatorPrecedence {
	switch binaryOpKind {
	case st.ASTERISK_TOKEN, // multiplication
		st.SLASH_TOKEN, // division
		st.PERCENT_TOKEN:
		return operatorPrecedenceMultiplicative
	case st.PLUS_TOKEN, st.MINUS_TOKEN:
		return operatorPrecedenceAdditive
	case st.GT_TOKEN,
		st.LT_TOKEN,
		st.GT_EQUAL_TOKEN,
		st.LT_EQUAL_TOKEN,
		st.IS_KEYWORD,
		st.NOT_IS_KEYWORD:
		return operatorPrecedenceBinaryCompare
	case st.DOT_TOKEN,
		st.OPEN_BRACKET_TOKEN,
		st.OPEN_PAREN_TOKEN,
		st.ANNOT_CHAINING_TOKEN,
		st.OPTIONAL_CHAINING_TOKEN,
		st.DOT_LT_TOKEN,
		st.SLASH_LT_TOKEN,
		st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		st.SLASH_ASTERISK_TOKEN:
		return operatorPrecedenceMemberAccess
	case st.DOUBLE_EQUAL_TOKEN,
		st.TRIPPLE_EQUAL_TOKEN,
		st.NOT_EQUAL_TOKEN,
		st.NOT_DOUBLE_EQUAL_TOKEN:
		return operatorPrecedenceEquality
	case st.BITWISE_AND_TOKEN:
		return operatorPrecedenceBitwiseAnd
	case st.BITWISE_XOR_TOKEN:
		return operatorPrecedenceBitwiseXor
	case st.PIPE_TOKEN:
		return operatorPrecedenceBitwiseOr
	case st.LOGICAL_AND_TOKEN:
		return operatorPrecedenceLogicalAnd
	case st.LOGICAL_OR_TOKEN:
		return operatorPrecedenceLogicalOr
	case st.RIGHT_ARROW_TOKEN:
		return operatorPrecedenceRemoteCallAction
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		return operatorPrecedenceAnonFuncOrLet
	case st.SYNC_SEND_TOKEN:
		return operatorPrecedenceAction
	case st.DOUBLE_LT_TOKEN,
		st.DOUBLE_GT_TOKEN,
		st.TRIPPLE_GT_TOKEN:
		return operatorPrecedenceShift
	case st.ELLIPSIS_TOKEN,
		st.DOUBLE_DOT_LT_TOKEN:
		return operatorPrecedenceRange
	case st.ELVIS_TOKEN:
		return operatorPrecedenceElvisConditional
	case st.QUESTION_MARK_TOKEN, st.COLON_TOKEN:
		return operatorPrecedenceConditional
	default:
		b.internalError("Unsupported binary operator '" + binaryOpKind.StrValue() + "'")
		return operatorPrecedenceDefault
	}
}

func (b *ballerinaParser) getBinaryOperatorKindToInsert(opPrecedenceLevel operatorPrecedence) st.SyntaxKind {
	switch opPrecedenceLevel {
	case operatorPrecedenceMultiplicative:
		return st.ASTERISK_TOKEN
	case operatorPrecedenceDefault,
		operatorPrecedenceUnary,
		operatorPrecedenceAction,
		operatorPrecedenceExpressionAction,
		operatorPrecedenceRemoteCallAction,
		operatorPrecedenceAnonFuncOrLet,
		operatorPrecedenceQuery,
		operatorPrecedenceTrap,
		operatorPrecedenceAdditive:
		return st.PLUS_TOKEN
	case operatorPrecedenceShift:
		return st.DOUBLE_LT_TOKEN
	case operatorPrecedenceRange:
		return st.ELLIPSIS_TOKEN
	case operatorPrecedenceBinaryCompare:
		return st.LT_TOKEN
	case operatorPrecedenceEquality:
		return st.DOUBLE_EQUAL_TOKEN
	case operatorPrecedenceBitwiseAnd:
		return st.BITWISE_AND_TOKEN
	case operatorPrecedenceBitwiseXor:
		return st.BITWISE_XOR_TOKEN
	case operatorPrecedenceBitwiseOr:
		return st.PIPE_TOKEN
	case operatorPrecedenceLogicalAnd:
		return st.LOGICAL_AND_TOKEN
	case operatorPrecedenceLogicalOr:
		return st.LOGICAL_OR_TOKEN
	case operatorPrecedenceElvisConditional:
		return st.ELVIS_TOKEN
	default:
		b.internalError("Unsupported operator precedence level")
		return st.NONE
	}
}

func (b *ballerinaParser) getMissingBinaryOperatorContext(opPrecedenceLevel operatorPrecedence) common.ParserRuleContext {
	switch opPrecedenceLevel {
	case operatorPrecedenceMultiplicative:
		return common.PARSER_RULE_CONTEXT_ASTERISK
	case operatorPrecedenceDefault,
		operatorPrecedenceUnary,
		operatorPrecedenceAction,
		operatorPrecedenceExpressionAction,
		operatorPrecedenceRemoteCallAction,
		operatorPrecedenceAnonFuncOrLet,
		operatorPrecedenceQuery,
		operatorPrecedenceTrap,
		operatorPrecedenceAdditive:
		return common.PARSER_RULE_CONTEXT_PLUS_TOKEN
	case operatorPrecedenceShift:
		return common.PARSER_RULE_CONTEXT_DOUBLE_LT
	case operatorPrecedenceRange:
		return common.PARSER_RULE_CONTEXT_ELLIPSIS
	case operatorPrecedenceBinaryCompare:
		return common.PARSER_RULE_CONTEXT_LT_TOKEN
	case operatorPrecedenceEquality:
		return common.PARSER_RULE_CONTEXT_DOUBLE_EQUAL
	case bitwiseAnd:
		return common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR
	case bitwiseXor:
		return common.PARSER_RULE_CONTEXT_BITWISE_XOR
	case operatorPrecedenceBitwiseOr:
		return common.PARSER_RULE_CONTEXT_PIPE
	case operatorPrecedenceLogicalAnd:
		return common.PARSER_RULE_CONTEXT_LOGICAL_AND
	case operatorPrecedenceLogicalOr:
		return common.PARSER_RULE_CONTEXT_LOGICAL_OR
	case operatorPrecedenceElvisConditional:
		return common.PARSER_RULE_CONTEXT_ELVIS
	default:
		b.internalError("Unsupported operator precedence level")
		return common.ParserRuleContext{}
	}
}

func (b *ballerinaParser) parseModuleTypeDefinition(metadata st.STNode, qualifier st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION)
	typeKeyword := b.parseTypeKeyword()
	typeName := b.parseTypeName()
	typeDescriptor := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF)
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateTypeDefinitionNode(metadata, qualifier, typeKeyword, typeName, typeDescriptor,
		semicolon)
}

func (b *ballerinaParser) parseClassDefinition(metadata st.STNode, qualifier st.STNode, qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION)
	classTypeQualifiers := b.createClassTypeQualNodeList(qualifiers)
	classKeyword := b.parseClassKeyword()
	className := b.parseClassName()
	openBrace := b.parseOpenBrace()
	classMembers := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_CLASS_MEMBER)
	closeBrace := b.parseCloseBrace()
	semicolon := b.parseOptionalSemicolon()
	b.endContext()
	return st.CreateClassDefinitionNode(metadata, qualifier, classTypeQualifiers, classKeyword,
		className, openBrace, classMembers, closeBrace, semicolon)
}

func (b *ballerinaParser) isClassTypeQual(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.READONLY_KEYWORD, st.DISTINCT_KEYWORD, st.ISOLATED_KEYWORD:
		return true
	default:
		return b.isObjectNetworkQual(tokenKind)
	}
}

func (b *ballerinaParser) isObjectTypeQual(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.ISOLATED_KEYWORD:
		return true
	default:
		return b.isObjectNetworkQual(tokenKind)
	}
}

func (b *ballerinaParser) isObjectNetworkQual(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.SERVICE_KEYWORD, st.CLIENT_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) createClassTypeQualNodeList(qualifierList []st.STNode) st.STNode {
	return b.createTypeQualNodeList(qualifierList, b.isClassTypeQual)
}

func (b *ballerinaParser) createObjectTypeQualNodeList(qualifierList []st.STNode) st.STNode {
	return b.createTypeQualNodeList(qualifierList, b.isObjectTypeQual)
}

func (b *ballerinaParser) createTypeQualNodeList(qualifierList []st.STNode, isTypeQual func(st.SyntaxKind) bool) st.STNode {
	var validatedList []st.STNode
	hasNetworkQual := false
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, st.ToToken(qualifier).Text())
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
		if isTypeQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				st.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		}
	}
	return st.CreateNodeList(validatedList...)
}

func (b *ballerinaParser) parseClassKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.CLASS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLASS_KEYWORD)
		return b.parseClassKeyword()
	}
}

func (b *ballerinaParser) parseTypeKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.TYPE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TYPE_KEYWORD)
		return b.parseTypeKeyword()
	}
}

func (b *ballerinaParser) parseTypeName() st.STNode {
	token := b.peek()
	if token.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TYPE_NAME)
		return b.parseTypeName()
	}
}

func (b *ballerinaParser) parseClassName() st.STNode {
	token := b.peek()
	if token.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLASS_NAME)
		return b.parseClassName()
	}
}

func (b *ballerinaParser) parseRecordTypeDescriptor() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR)
	recordKeyword := b.parseRecordKeyword()
	bodyStartDelimiter := b.parseRecordBodyStartDelimiter()
	var recordFields []st.STNode
	recordRestDescriptor := st.CreateEmptyNode()
	for !b.isEndOfRecordTypeNode() {
		field := b.parseFieldOrRestDescriptor()
		if field == nil {
			break
		}
		if (field.Kind() == st.RECORD_REST_TYPE) && (bodyStartDelimiter.Kind() == st.OPEN_BRACE_TOKEN) {
			if len(recordFields) == 0 {
				bodyStartDelimiter = st.CloneWithTrailingInvalidNodeMinutiae(bodyStartDelimiter, field,
					&common.ERROR_INCLUSIVE_RECORD_TYPE_CANNOT_CONTAIN_REST_FIELD)
			} else {
				b.updateLastNodeInListWithInvalidNode(recordFields, field,
					&common.ERROR_INCLUSIVE_RECORD_TYPE_CANNOT_CONTAIN_REST_FIELD)
			}
			continue
		} else if field.Kind() == st.RECORD_REST_TYPE {
			recordRestDescriptor = field
			for !b.isEndOfRecordTypeNode() {
				invalidField := b.parseFieldOrRestDescriptor()
				if invalidField == nil {
					break
				}
				recordRestDescriptor = st.CloneWithTrailingInvalidNodeMinutiae(recordRestDescriptor,
					invalidField, &common.ERROR_MORE_RECORD_FIELDS_AFTER_REST_FIELD)
			}
			break
		}
		recordFields = append(recordFields, field)
	}
	fields := st.CreateNodeList(recordFields...)
	bodyEndDelimiter := b.parseRecordBodyCloseDelimiter(bodyStartDelimiter.Kind())
	b.endContext()
	return st.CreateRecordTypeDescriptorNode(recordKeyword, bodyStartDelimiter, fields,
		recordRestDescriptor, bodyEndDelimiter)
}

func (b *ballerinaParser) parseRecordBodyStartDelimiter() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACE_PIPE_TOKEN:
		return b.parseClosedRecordBodyStart()
	case st.OPEN_BRACE_TOKEN:
		return b.parseOpenBrace()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RECORD_BODY_START)
		return b.parseRecordBodyStartDelimiter()
	}
}

func (b *ballerinaParser) parseClosedRecordBodyStart() st.STNode {
	token := b.peek()
	if token.Kind() == st.OPEN_BRACE_PIPE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START)
		return b.parseClosedRecordBodyStart()
	}
}

func (b *ballerinaParser) parseRecordBodyCloseDelimiter(startingDelimeter st.SyntaxKind) st.STNode {
	if startingDelimeter == st.OPEN_BRACE_PIPE_TOKEN {
		return b.parseClosedRecordBodyEnd()
	}
	return b.parseCloseBrace()
}

func (b *ballerinaParser) parseClosedRecordBodyEnd() st.STNode {
	token := b.peek()
	if token.Kind() == st.CLOSE_BRACE_PIPE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END)
		return b.parseClosedRecordBodyEnd()
	}
}

func (b *ballerinaParser) parseRecordKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.RECORD_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RECORD_KEYWORD)
		return b.parseRecordKeyword()
	}
}

func (b *ballerinaParser) parseFieldOrRestDescriptor() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.CLOSE_BRACE_TOKEN,
		st.CLOSE_BRACE_PIPE_TOKEN:
		return nil
	case st.ASTERISK_TOKEN:
		b.startContext(common.PARSER_RULE_CONTEXT_RECORD_FIELD)
		asterisk := b.consume()
		ty := b.parseTypeReferenceInTypeInclusion()
		semicolonToken := b.parseSemicolon()
		b.endContext()
		return st.CreateTypeReferenceNode(asterisk, ty, semicolonToken)
	case st.DOCUMENTATION_STRING,
		st.AT_TOKEN:
		return b.parseRecordField()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseRecordField()
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END)
		return b.parseFieldOrRestDescriptor()
	}
}

func (b *ballerinaParser) parseRecordField() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RECORD_FIELD)
	metadata := b.parseMetaData()
	fieldOrRestDesc := b.parseRecordFieldInner(b.peek(), metadata)
	b.endContext()
	return fieldOrRestDesc
}

func (b *ballerinaParser) parseRecordFieldInner(nextToken st.STToken, metadata st.STNode) st.STNode {
	if nextToken.Kind() != st.READONLY_KEYWORD {
		ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD)
		return b.parseFieldOrRestDescriptorRhs(metadata, ty)
	}
	var ty st.STNode
	var readOnlyQualifier st.STNode
	readOnlyQualifier = b.parseReadonlyKeyword()
	nextToken = b.peek()
	if nextToken.Kind() == st.IDENTIFIER_TOKEN {
		fieldNameOrTypeDesc := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_RECORD_FIELD_NAME_OR_TYPE_NAME)
		if fieldNameOrTypeDesc.Kind() == st.QUALIFIED_NAME_REFERENCE {
			ty = fieldNameOrTypeDesc
		} else {
			nextToken = b.peek()
			switch nextToken.Kind() {
			case st.SEMICOLON_TOKEN, st.EQUAL_TOKEN:
				ty = b.createBuiltinSimpleNameReference(readOnlyQualifier)
				readOnlyQualifier = st.CreateEmptyNode()
				nameNode, ok := fieldNameOrTypeDesc.(*st.STSimpleNameReferenceNode)
				if !ok {
					b.internalError("expected STSimpleNameReferenceNode")
					return st.CreateEmptyNode()
				}
				fieldName := nameNode.Name
				return b.parseFieldDescriptorRhs(metadata, readOnlyQualifier, ty, fieldName)
			default:
				ty = b.parseComplexTypeDescriptor(fieldNameOrTypeDesc,
					common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD, false)
			}
		}
	} else if nextToken.Kind() == st.ELLIPSIS_TOKEN {
		ty = b.createBuiltinSimpleNameReference(readOnlyQualifier)
		return b.parseFieldOrRestDescriptorRhs(metadata, ty)
	} else if b.isTypeStartingToken(nextToken.Kind()) {
		ty = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD)
	} else {
		readOnlyQualifier = b.createBuiltinSimpleNameReference(readOnlyQualifier)
		ty = b.parseComplexTypeDescriptor(readOnlyQualifier, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD, false)
		readOnlyQualifier = st.CreateEmptyNode()
	}
	return b.parseIndividualRecordField(metadata, readOnlyQualifier, ty)
}

func (b *ballerinaParser) parseIndividualRecordField(metadata st.STNode, readOnlyQualifier st.STNode, ty st.STNode) st.STNode {
	fieldName := b.parseVariableName()
	return b.parseFieldDescriptorRhs(metadata, readOnlyQualifier, ty, fieldName)
}

func (b *ballerinaParser) parseTypeReferenceInTypeInclusion() st.STNode {
	typeReference := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION)
	if typeReference.Kind() == st.SIMPLE_NAME_REFERENCE {
		if typeReference.HasDiagnostics() {
			emptyNameReference := st.CreateSimpleNameReferenceNode(st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN, &common.ERROR_MISSING_IDENTIFIER))
			return emptyNameReference
		}
		return typeReference
	}
	if typeReference.Kind() == st.QUALIFIED_NAME_REFERENCE {
		return typeReference
	}
	emptyNameReference := st.CreateSimpleNameReferenceNode(st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil))
	emptyNameReference = st.CloneWithTrailingInvalidNodeMinutiae(emptyNameReference, typeReference,
		&common.ERROR_ONLY_TYPE_REFERENCE_ALLOWED_AS_TYPE_INCLUSIONS)
	return emptyNameReference
}

func (b *ballerinaParser) parseTypeReference() st.STNode {
	return b.parseTypeReferenceInner(false)
}

func (b *ballerinaParser) parseTypeReferenceInner(isInConditionalExpr bool) st.STNode {
	return b.parseQualifiedIdentifierInner(common.PARSER_RULE_CONTEXT_TYPE_REFERENCE, isInConditionalExpr)
}

func (b *ballerinaParser) parseQualifiedIdentifier(currentCtx common.ParserRuleContext) st.STNode {
	return b.parseQualifiedIdentifierInner(currentCtx, false)
}

func (b *ballerinaParser) parseQualifiedIdentifierInner(currentCtx common.ParserRuleContext, isInConditionalExpr bool) st.STNode {
	token := b.peek()
	var typeRefOrPkgRef st.STNode
	if token.Kind() == st.IDENTIFIER_TOKEN {
		typeRefOrPkgRef = b.consume()
	} else if b.isQualifiedIdentifierPredeclaredPrefix(token.Kind()) {
		preDeclaredPrefix := b.consume()
		typeRefOrPkgRef = st.CreateIdentifierToken(preDeclaredPrefix.Text(),
			preDeclaredPrefix.LeadingMinutiae(), preDeclaredPrefix.TrailingMinutiae())
	} else {
		b.recover(token, currentCtx, false)
		if b.peek().Kind() != st.IDENTIFIER_TOKEN {
			b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
			return b.parseQualifiedIdentifierInner(currentCtx, isInConditionalExpr)
		}
		typeRefOrPkgRef = b.consume()
	}
	return b.parseQualifiedIdentifierNode(typeRefOrPkgRef, isInConditionalExpr)
}

func (b *ballerinaParser) parseQualifiedIdentifierNode(identifier st.STNode, isInConditionalExpr bool) st.STNode {
	nextToken := b.peekN(1)
	if nextToken.Kind() != st.COLON_TOKEN {
		return st.CreateSimpleNameReferenceNode(identifier)
	}
	if isInConditionalExpr && (b.hasTrailingMinutiae(identifier) || b.hasTrailingMinutiae(nextToken)) {
		return st.GetSimpleNameRefNode(identifier)
	}
	nextNextToken := b.peekN(2)
	switch nextNextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		colon := b.consume()
		varOrFuncName := b.consume()
		return b.createQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
	case st.COLON_TOKEN:
		b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
		return b.parseQualifiedIdentifierNode(identifier, isInConditionalExpr)
	default:
		if (nextNextToken.Kind() == st.MAP_KEYWORD) && (b.peekN(3).Kind() != st.LT_TOKEN) {
			colon := b.consume()
			mapKeyword := b.consume()
			refName := st.CreateIdentifierTokenWithDiagnostics(mapKeyword.Text(),
				mapKeyword.LeadingMinutiae(), mapKeyword.TrailingMinutiae(), mapKeyword.Diagnostics())
			return b.createQualifiedNameReferenceNode(identifier, colon, refName)
		}
		if isInConditionalExpr {
			return st.GetSimpleNameRefNode(identifier)
		}
		colon := b.consume()
		varOrFuncName := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_IDENTIFIER)
		return b.createQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
	}
}

func (b *ballerinaParser) createQualifiedNameReferenceNode(identifier st.STNode, colon st.STNode, varOrFuncName st.STNode) st.STNode {
	if b.hasTrailingMinutiae(identifier) || b.hasTrailingMinutiae(colon) {
		colon = st.AddDiagnostic(colon,
			&common.ERROR_INTERVENING_WHITESPACES_ARE_NOT_ALLOWED)
	}
	return st.CreateQualifiedNameReferenceNode(identifier, colon, varOrFuncName)
}

func (b *ballerinaParser) parseFieldOrRestDescriptorRhs(metadata st.STNode, ty st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ELLIPSIS_TOKEN:
		b.reportInvalidMetaData(metadata, "record rest descriptor")
		ellipsis := b.parseEllipsis()
		semicolonToken := b.parseSemicolon()
		return st.CreateRecordRestDescriptorNode(ty, ellipsis, semicolonToken)
	case st.IDENTIFIER_TOKEN:
		readonlyQualifier := st.CreateEmptyNode()
		return b.parseIndividualRecordField(metadata, readonlyQualifier, ty)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS)
		return b.parseFieldOrRestDescriptorRhs(metadata, ty)
	}
}

func (b *ballerinaParser) parseFieldDescriptorRhs(metadata st.STNode, readonlyQualifier st.STNode, ty st.STNode, fieldName st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.SEMICOLON_TOKEN:
		questionMarkToken := st.CreateEmptyNode()
		semicolonToken := b.parseSemicolon()
		return st.CreateRecordFieldNode(metadata, readonlyQualifier, ty, fieldName,
			questionMarkToken, semicolonToken)
	case st.QUESTION_MARK_TOKEN:
		questionMarkToken := b.parseQuestionMark()
		semicolonToken := b.parseSemicolon()
		return st.CreateRecordFieldNode(metadata, readonlyQualifier, ty, fieldName,
			questionMarkToken, semicolonToken)
	case st.EQUAL_TOKEN:
		equalsToken := b.parseAssignOp()
		expression := b.parseExpression()
		semicolonToken := b.parseSemicolon()
		return st.CreateRecordFieldWithDefaultValueNode(metadata, readonlyQualifier, ty, fieldName,
			equalsToken, expression, semicolonToken)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS)
		return b.parseFieldDescriptorRhs(metadata, readonlyQualifier, ty, fieldName)
	}
}

func (b *ballerinaParser) parseQuestionMark() st.STNode {
	token := b.peek()
	if token.Kind() == st.QUESTION_MARK_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_QUESTION_MARK)
		return b.parseQuestionMark()
	}
}

func (b *ballerinaParser) parseStatements() st.STNode {
	return b.parseStatementsInner(nil)
}

func (b *ballerinaParser) parseStatementsInner(stmts []st.STNode) st.STNode {
	for !b.isEndOfStatements() {
		stmt := b.parseStatement()
		if stmt == nil {
			break
		}
		if stmt.Kind() == st.NAMED_WORKER_DECLARATION {
			b.addInvalidNodeToNextToken(stmt, &common.ERROR_NAMED_WORKER_NOT_ALLOWED_HERE)
			continue
		}
		if b.validateStatement(stmt) {
			continue
		}
		stmts = append(stmts, stmt)
	}
	return st.CreateNodeList(stmts...)
}

func (b *ballerinaParser) parseStatement() st.STNode {
	nextToken := b.peek()
	annots := st.CreateEmptyNodeList()
	switch nextToken.Kind() {
	case st.CLOSE_BRACE_TOKEN, st.EOF_TOKEN:
		return nil
	case st.SEMICOLON_TOKEN:
		b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
		return b.parseStatement()
	case st.AT_TOKEN:
		annots = b.parseOptionalAnnotations()
	default:
		if b.isStatementStartingToken(nextToken.Kind()) {
			break
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STATEMENT)
		if solution.Action == actionKeep {
			break
		}
		return b.parseStatement()
	}
	return b.parseStatementWithAnnotataions(annots)
}

func (b *ballerinaParser) validateStatement(statement st.STNode) bool {
	switch statement.Kind() {
	case st.LOCAL_TYPE_DEFINITION_STATEMENT:
		b.addInvalidNodeToNextToken(statement, &common.ERROR_LOCAL_TYPE_DEFINITION_NOT_ALLOWED)
		return true
	case st.CONST_DECLARATION:
		b.addInvalidNodeToNextToken(statement, &common.ERROR_LOCAL_CONST_DECL_NOT_ALLOWED)
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) getAnnotations(nullbaleAnnot st.STNode) st.STNode {
	if nullbaleAnnot != nil {
		return nullbaleAnnot
	}
	return st.CreateEmptyNodeList()
}

func (b *ballerinaParser) parseStatementWithAnnotataions(annots st.STNode) st.STNode {
	result, _ := b.parseStatementInner(annots, nil)
	return result
}

func (b *ballerinaParser) parseStatementInner(annots st.STNode, qualifiers []st.STNode) (st.STNode, []st.STNode) {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		return b.parseStmtStartsWithTypeOrExpr(b.getAnnotations(annots), qualifiers), qualifiers
	}
	switch nextToken.Kind() {
	case st.CLOSE_BRACE_TOKEN,
		st.EOF_TOKEN:
		publicQualifier := st.CreateEmptyNode()
		return b.createMissingSimpleVarDeclInnerWithQualifiers(b.getAnnotations(annots), publicQualifier, qualifiers, false), qualifiers
	case st.SEMICOLON_TOKEN:
		b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
		return b.parseStatementInner(annots, qualifiers)
	case st.FINAL_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		finalKeyword := b.consume()
		return b.parseVariableDecl(b.getAnnotations(annots), finalKeyword), qualifiers
	case st.IF_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseIfElseBlock(), qualifiers
	case st.WHILE_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseWhileStatement(), qualifiers
	case st.DO_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseDoStatement(), qualifiers
	case st.PANIC_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parsePanicStatement(), qualifiers
	case st.CONTINUE_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseContinueStatement(), qualifiers
	case st.BREAK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseBreakStatement(), qualifiers
	case st.RETURN_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseReturnStatement(), qualifiers
	case st.FAIL_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseFailStatement(), qualifiers
	case st.TYPE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseLocalTypeDefinitionStatement(b.getAnnotations(annots)), qualifiers
	case st.CONST_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseConstantDeclaration(annots, st.CreateEmptyNode()), qualifiers
	case st.LOCK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseLockStatement(), qualifiers
	case st.OPEN_BRACE_TOKEN:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStatementStartsWithOpenBrace(), qualifiers
	case st.WORKER_KEYWORD:
		return b.parseNamedWorkerDeclaration(b.getAnnotations(annots), qualifiers), qualifiers
	case st.FORK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseForkStatement(), qualifiers
	case st.FOREACH_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseForEachStatement(), qualifiers
	case st.START_KEYWORD,
		st.CHECK_KEYWORD,
		st.CHECKPANIC_KEYWORD,
		st.TRAP_KEYWORD,
		st.FLUSH_KEYWORD,
		st.LEFT_ARROW_TOKEN,
		st.WAIT_KEYWORD,
		st.FROM_KEYWORD,
		st.COMMIT_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseExpressionStatement(b.getAnnotations(annots)), qualifiers
	case st.XMLNS_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseXMLNamespaceDeclaration(false), qualifiers
	case st.TRANSACTION_KEYWORD:
		return b.parseTransactionStmtOrVarDecl(annots, qualifiers, b.consume())
	case st.RETRY_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRetryStatement(), qualifiers
	case st.ROLLBACK_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRollbackStatement(), qualifiers
	case st.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseStatementStartsWithOpenBracket(b.getAnnotations(annots), false), qualifiers
	case st.FUNCTION_KEYWORD,
		st.OPEN_PAREN_TOKEN,
		st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN,
		st.STRING_KEYWORD,
		st.XML_KEYWORD:
		return b.parseStmtStartsWithTypeOrExpr(b.getAnnotations(annots), qualifiers), qualifiers
	case st.MATCH_KEYWORD:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMatchStatement(), qualifiers
	case st.ERROR_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseErrorTypeDescOrErrorBP(b.getAnnotations(annots)), qualifiers
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseStatementStartWithExpr(b.getAnnotations(annots)), qualifiers
		}
		if b.isTypeStartingToken(nextToken.Kind()) {
			publicQualifier := st.CreateEmptyNode()
			res, _ := b.parseVariableDeclInner(b.getAnnotations(annots), publicQualifier, nil, qualifiers,
				false)
			return res, qualifiers
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS)
		if solution.Action == actionKeep {
			b.reportInvalidQualifierList(qualifiers)
			finalKeyword := st.CreateEmptyNode()
			return b.parseVariableDecl(b.getAnnotations(annots), finalKeyword), qualifiers
		}
		return b.parseStatementInner(annots, qualifiers)
	}
}

func (b *ballerinaParser) parseVariableDecl(annots st.STNode, finalKeyword st.STNode) st.STNode {
	var typeDescQualifiers []st.STNode
	var varDecQualifiers []st.STNode
	if finalKeyword != nil {
		varDecQualifiers = append(varDecQualifiers, finalKeyword)
	}
	publicQualifier := st.CreateEmptyNode()
	res, _ := b.parseVariableDeclInner(annots, publicQualifier, varDecQualifiers, typeDescQualifiers, false)
	return res
}

// Return result, and modified varDeclQuals
func (b *ballerinaParser) parseVariableDeclInner(annots st.STNode, publicQualifier st.STNode, varDeclQuals []st.STNode, typeDescQualifiers []st.STNode, isModuleVar bool) (st.STNode, []st.STNode) {
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	typeBindingPattern := b.parseTypedBindingPatternInner(typeDescQualifiers,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	return b.parseVarDeclRhsInner(annots, publicQualifier, varDeclQuals, typeBindingPattern, isModuleVar)
}

// Return result, and modified qualifiers
func (b *ballerinaParser) parseVarDeclTypeDescRhs(typeDesc st.STNode, metadata st.STNode, qualifiers []st.STNode, isTypedBindingPattern bool, isModuleVar bool) (st.STNode, []st.STNode) {
	publicQualifier := st.CreateEmptyNode()
	return b.parseVarDeclTypeDescRhsInner(typeDesc, metadata, publicQualifier, qualifiers, isTypedBindingPattern,
		isModuleVar)
}

// Return result, and modified qualifiers
func (b *ballerinaParser) parseVarDeclTypeDescRhsInner(typeDesc st.STNode, metadata st.STNode, publicQual st.STNode, qualifiers []st.STNode, isTypedBindingPattern bool, isModuleVar bool) (st.STNode, []st.STNode) {
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	typeDesc = b.parseComplexTypeDescriptor(typeDesc,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, isTypedBindingPattern)
	typedBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	return b.parseVarDeclRhsInner(metadata, publicQual, qualifiers, typedBindingPattern, isModuleVar)
}

// Return result, and modified varDeclQuals
func (b *ballerinaParser) parseVarDeclRhs(metadata st.STNode, typedBindingPattern st.STNode) (st.STNode, []st.STNode) {
	publicQualifier := st.CreateEmptyNode()
	return b.parseVarDeclRhsInner(metadata, publicQualifier, nil, typedBindingPattern, false)
}

// Return result, and modified varDeclQuals
func (b *ballerinaParser) parseVarDeclRhsInner(metadata st.STNode, publicQualifier st.STNode, varDeclQuals []st.STNode, typedBindingPattern st.STNode, isModuleVar bool) (st.STNode, []st.STNode) {
	var assign st.STNode
	var expr st.STNode
	var semicolon st.STNode
	hasVarInit := false
	isConfigurable := isModuleVar && b.isSyntaxKindInList(varDeclQuals, st.CONFIGURABLE_KEYWORD)

	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.EQUAL_TOKEN:
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
	case st.SEMICOLON_TOKEN:
		assign = st.CreateEmptyNode()
		expr = st.CreateEmptyNode()
		semicolon = b.parseSemicolon()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS)
		return b.parseVarDeclRhsInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, isModuleVar)
	}
	b.endContext()
	if !hasVarInit {
		typedBindingPatternNode, ok := typedBindingPattern.(*st.STTypedBindingPatternNode)
		if !ok {
			b.internalError("expected STTypedBindingPatternNode")
			return st.CreateEmptyNode(), nil
		}
		bindingPatternKind := typedBindingPatternNode.BindingPattern.Kind()
		if bindingPatternKind != st.CAPTURE_BINDING_PATTERN {
			assign = st.CreateMissingTokenWithDiagnostics(st.EQUAL_TOKEN,
				&common.ERROR_VARIABLE_DECL_HAVING_BP_MUST_BE_INITIALIZED)
			identifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
			expr = st.CreateSimpleNameReferenceNode(identifier)
		}
	}
	if isModuleVar {
		return b.createModuleVarDeclaration(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign,
			expr, semicolon, isConfigurable, hasVarInit)
	}
	var finalKeyword st.STNode
	if len(varDeclQuals) == 0 {
		finalKeyword = st.CreateEmptyNode()
	} else {
		finalKeyword = varDeclQuals[0]
	}
	if metadata.Kind() != st.LIST {
		b.internalError("assertion failed")
		return st.CreateEmptyNode(), nil
	}
	return st.CreateVariableDeclarationNode(metadata, finalKeyword, typedBindingPattern, assign,
		expr, semicolon), varDeclQuals
}

func (b *ballerinaParser) parseConfigurableVarDeclRhs() st.STNode {
	var expr st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.QUESTION_MARK_TOKEN:
		expr = st.CreateRequiredExpressionNode(b.consume())
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

func (b *ballerinaParser) createModuleVarDeclaration(metadata st.STNode, publicQualifier st.STNode, varDeclQuals []st.STNode, typedBindingPattern st.STNode, assign st.STNode, expr st.STNode, semicolon st.STNode, isConfigurable bool, hasVarInit bool) (st.STNode, []st.STNode) {
	if hasVarInit || len(varDeclQuals) == 0 {
		return b.createModuleVarDeclarationInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign,
			expr, semicolon), varDeclQuals
	}
	if isConfigurable {
		return b.createConfigurableModuleVarDeclWithMissingInitializer(metadata, publicQualifier, varDeclQuals,
			typedBindingPattern, semicolon), varDeclQuals
	}
	lastQualifier := b.getLastNodeInList(varDeclQuals)
	if lastQualifier.Kind() == st.ISOLATED_KEYWORD {
		lastQualifier = varDeclQuals[len(varDeclQuals)-1]
		varDeclQuals = varDeclQuals[:len(varDeclQuals)-1]
		typedBindingPattern = b.modifyTypedBindingPatternWithIsolatedQualifier(typedBindingPattern, lastQualifier)
	}
	return b.createModuleVarDeclarationInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign, expr,
		semicolon), varDeclQuals
}

func (b *ballerinaParser) createConfigurableModuleVarDeclWithMissingInitializer(metadata st.STNode, publicQualifier st.STNode, varDeclQuals []st.STNode, typedBindingPattern st.STNode, semicolon st.STNode) st.STNode {
	var assign st.STNode
	assign = st.CreateMissingToken(st.EQUAL_TOKEN, nil)
	assign = st.AddDiagnostic(assign,
		&common.ERROR_CONFIGURABLE_VARIABLE_MUST_BE_INITIALIZED_OR_REQUIRED)
	questionMarkToken := st.CreateMissingToken(st.QUESTION_MARK_TOKEN, nil)
	expr := st.CreateRequiredExpressionNode(questionMarkToken)
	return b.createModuleVarDeclarationInner(metadata, publicQualifier, varDeclQuals, typedBindingPattern, assign, expr,
		semicolon)
}

func (b *ballerinaParser) createModuleVarDeclarationInner(metadata st.STNode, publicQualifier st.STNode, varDeclQuals []st.STNode, typedBindingPattern st.STNode, assign st.STNode, expr st.STNode, semicolon st.STNode) st.STNode {
	if publicQualifier != nil {
		typedBindingPatternNode, ok := typedBindingPattern.(*st.STTypedBindingPatternNode)
		if !ok {
			b.internalError("expected STTypedBindingPatternNode")
			return st.CreateEmptyNode()
		}
		if typedBindingPatternNode.TypeDescriptor.Kind() == st.VAR_TYPE_DESC {
			if len(varDeclQuals) != 0 {
				b.updateFirstNodeInListWithLeadingInvalidNode(varDeclQuals, publicQualifier,
					&common.ERROR_VARIABLE_DECLARED_WITH_VAR_CANNOT_BE_PUBLIC)
			} else {
				typedBindingPattern = st.CloneWithLeadingInvalidNodeMinutiae(typedBindingPattern,
					publicQualifier, &common.ERROR_VARIABLE_DECLARED_WITH_VAR_CANNOT_BE_PUBLIC)
			}
			publicQualifier = st.CreateEmptyNode()
		} else if b.isSyntaxKindInList(varDeclQuals, st.ISOLATED_KEYWORD) {
			b.updateFirstNodeInListWithLeadingInvalidNode(varDeclQuals, publicQualifier,
				&common.ERROR_ISOLATED_VAR_CANNOT_BE_DECLARED_AS_PUBLIC)
			publicQualifier = st.CreateEmptyNode()
		}
	}
	varDeclQualifiersNode := st.CreateNodeList(varDeclQuals...)
	return st.CreateModuleVariableDeclarationNode(metadata, publicQualifier, varDeclQualifiersNode,
		typedBindingPattern, assign, expr, semicolon)
}

func (b *ballerinaParser) createMissingSimpleVarDecl(isModuleVar bool) st.STNode {
	var metadata st.STNode
	if isModuleVar {
		metadata = st.CreateEmptyNode()
	} else {
		metadata = st.CreateEmptyNodeList()
	}
	return b.createMissingSimpleVarDeclInner(metadata, isModuleVar)
}

func (b *ballerinaParser) createMissingSimpleVarDeclInner(metadata st.STNode, isModuleVar bool) st.STNode {
	publicQualifier := st.CreateEmptyNode()
	return b.createMissingSimpleVarDeclInnerWithQualifiers(metadata, publicQualifier, nil, isModuleVar)
}

func (b *ballerinaParser) createMissingSimpleVarDeclInnerWithQualifiers(metadata st.STNode, publicQualifier st.STNode, qualifiers []st.STNode, isModuleVar bool) st.STNode {
	emptyNode := st.CreateEmptyNode()
	simpleTypeDescIdentifier := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_TYPE_DESC)
	identifier := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_VARIABLE_NAME)
	simpleNameRef := st.CreateSimpleNameReferenceNode(simpleTypeDescIdentifier)
	semicolon := st.CreateMissingTokenWithDiagnostics(st.SEMICOLON_TOKEN,
		&common.ERROR_MISSING_SEMICOLON_TOKEN)
	captureBP := st.CreateCaptureBindingPatternNode(identifier)
	typedBindingPattern := st.CreateTypedBindingPatternNode(simpleNameRef, captureBP)
	if isModuleVar {
		varDeclQuals, qualifiers := b.extractVarDeclQualifiers(qualifiers, true)
		typedBindingPattern = b.modifyNodeWithInvalidTokenList(qualifiers, typedBindingPattern)
		if b.isSyntaxKindInList(varDeclQuals, st.CONFIGURABLE_KEYWORD) {
			return b.createConfigurableModuleVarDeclWithMissingInitializer(metadata, publicQualifier, varDeclQuals,
				typedBindingPattern, semicolon)
		}
		varDeclQualNodeList := st.CreateNodeList(varDeclQuals...)
		return st.CreateModuleVariableDeclarationNode(metadata, publicQualifier, varDeclQualNodeList,
			typedBindingPattern, emptyNode, emptyNode, semicolon)
	}
	typedBindingPattern = b.modifyNodeWithInvalidTokenList(qualifiers, typedBindingPattern)
	return st.CreateVariableDeclarationNode(metadata, emptyNode, typedBindingPattern, emptyNode,
		emptyNode, semicolon)
}

func (b *ballerinaParser) createMissingWhereClause() st.STNode {
	whereKeyword := st.CreateMissingTokenWithDiagnostics(st.WHERE_KEYWORD,
		&common.ERROR_MISSING_WHERE_KEYWORD)
	missingIdentifier := st.CreateMissingTokenWithDiagnostics(
		st.IDENTIFIER_TOKEN, &common.ERROR_MISSING_EXPRESSION)
	missingExpr := st.CreateSimpleNameReferenceNode(missingIdentifier)
	return st.CreateWhereClauseNode(whereKeyword, missingExpr)
}

func (b *ballerinaParser) createMissingSimpleObjectFieldInner(metadata st.STNode, qualifiers []st.STNode, isObjectTypeDesc bool) (st.STNode, []st.STNode) {
	emptyNode := st.CreateEmptyNode()
	simpleTypeDescIdentifier := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_TYPE_DESC)
	simpleNameRef := st.CreateSimpleNameReferenceNode(simpleTypeDescIdentifier)
	identifier := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_FIELD_NAME)
	semicolon := st.CreateMissingTokenWithDiagnostics(st.SEMICOLON_TOKEN,
		&common.ERROR_MISSING_SEMICOLON_TOKEN)
	objectFieldQualifiers, qualifiers := b.extractObjectFieldQualifiers(qualifiers, isObjectTypeDesc)
	objectFieldQualNodeList := st.CreateNodeList(objectFieldQualifiers...)
	simpleNameRef = b.modifyNodeWithInvalidTokenList(qualifiers, simpleNameRef)
	metadataNode, ok := metadata.(*st.STMetadataNode)
	if !ok {
		b.internalError("expected STMetadataNode")
		return st.CreateEmptyNode(), nil
	}
	if metadata != nil {
		metadata = b.addMetadataNotAttachedDiagnostic(*metadataNode)
	}
	return st.CreateObjectFieldNode(metadata, emptyNode, objectFieldQualNodeList,
		simpleNameRef, identifier, emptyNode, emptyNode, semicolon), qualifiers
}

func (b *ballerinaParser) createMissingSimpleObjectField() st.STNode {
	metadata := st.CreateEmptyNode()
	res, _ := b.createMissingSimpleObjectFieldInner(metadata, nil, false)
	return res
}

func (b *ballerinaParser) modifyNodeWithInvalidTokenList(qualifiers []st.STNode, node st.STNode) st.STNode {
	i := (len(qualifiers) - 1)
	for ; i >= 0; i-- {
		qualifier := qualifiers[i]
		node = st.CloneWithLeadingInvalidNodeMinutiae(node, qualifier, nil)
	}
	return node
}

func (b *ballerinaParser) modifyTypedBindingPatternWithIsolatedQualifier(typedBindingPattern st.STNode, isolatedQualifier st.STNode) st.STNode {
	typedBindingPatternNode, ok := typedBindingPattern.(*st.STTypedBindingPatternNode)
	if !ok {
		b.internalError("expected STTypedBindingPatternNode")
		return st.CreateEmptyNode()
	}
	typeDescriptor := typedBindingPatternNode.TypeDescriptor
	bindingPattern := typedBindingPatternNode.BindingPattern
	switch typeDescriptor.Kind() {
	case st.OBJECT_TYPE_DESC:
		typeDescriptor = b.modifyObjectTypeDescWithALeadingQualifier(typeDescriptor, isolatedQualifier)
	case st.FUNCTION_TYPE_DESC:
		typeDescriptor = b.modifyFuncTypeDescWithALeadingQualifier(typeDescriptor, isolatedQualifier)
	default:
		typeDescriptor = st.CloneWithLeadingInvalidNodeMinutiae(typeDescriptor, isolatedQualifier,
			&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(isolatedQualifier).Text())
	}
	return st.CreateTypedBindingPatternNode(typeDescriptor, bindingPattern)
}

func (b *ballerinaParser) modifyObjectTypeDescWithALeadingQualifier(objectTypeDesc st.STNode, newQualifier st.STNode) st.STNode {
	objectTypeDescriptorNode, ok := objectTypeDesc.(*st.STObjectTypeDescriptorNode)
	if !ok {
		b.internalError("expected STObjectTypeDescriptorNode")
		return st.CreateEmptyNode()
	}

	qualifierList, ok := objectTypeDescriptorNode.ObjectTypeQualifiers.(*st.STNodeList)
	if !ok {
		b.internalError("expected STNodeList")
		return st.CreateEmptyNode()
	}
	newObjectTypeQualifiers := b.modifyNodeListWithALeadingQualifier(qualifierList, newQualifier)
	return st.CreateObjectTypeDescriptorNode(newObjectTypeQualifiers, objectTypeDescriptorNode.ObjectKeyword,
		objectTypeDescriptorNode.OpenBrace, objectTypeDescriptorNode.Members,
		objectTypeDescriptorNode.CloseBrace)
}

func (b *ballerinaParser) modifyFuncTypeDescWithALeadingQualifier(funcTypeDesc st.STNode, newQualifier st.STNode) st.STNode {
	funcTypeDescriptorNode, ok := funcTypeDesc.(*st.STFunctionTypeDescriptorNode)
	if !ok {
		b.internalError("expected STFunctionTypeDescriptorNode")
		return st.CreateEmptyNode()
	}
	qualifierList := funcTypeDescriptorNode.QualifierList
	newfuncTypeQualifiers := b.modifyNodeListWithALeadingQualifier(qualifierList, newQualifier)
	return st.CreateFunctionTypeDescriptorNode(newfuncTypeQualifiers, funcTypeDescriptorNode.FunctionKeyword,
		funcTypeDescriptorNode.FunctionSignature)
}

func (b *ballerinaParser) modifyNodeListWithALeadingQualifier(qualifiers st.STNode, newQualifier st.STNode) st.STNode {
	var newQualifierList []st.STNode
	newQualifierList = append(newQualifierList, newQualifier)
	qualifierNodeList, ok := qualifiers.(*st.STNodeList)
	if !ok {
		b.internalError("expected STNodeList")
		return st.CreateEmptyNode()
	}
	i := 0
	for ; i < qualifierNodeList.Size(); i++ {
		qualifier := qualifierNodeList.Get(i)
		if qualifier.Kind() == newQualifier.Kind() {
			b.updateLastNodeInListWithInvalidNode(newQualifierList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, st.ToToken(qualifier).Text())
		} else {
			newQualifierList = append(newQualifierList, qualifier)
		}
	}
	return st.CreateNodeList(newQualifierList...)
}

func (b *ballerinaParser) parseAssignmentStmtRhs(lvExpr st.STNode) st.STNode {
	assign := b.parseAssignOp()
	expr := b.parseActionOrExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	if lvExpr.Kind() == st.ERROR_CONSTRUCTOR {
		errConstructor, ok := lvExpr.(*st.STErrorConstructorExpressionNode)
		if !ok {
			b.internalError("expected STErrorConstructorExpressionNode")
			return st.CreateEmptyNode()
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
		identifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		simpleNameRef := st.CreateSimpleNameReferenceNode(identifier)
		lvExpr = st.CloneWithLeadingInvalidNodeMinutiae(simpleNameRef, lvExpr,
			&common.ERROR_INVALID_EXPR_IN_ASSIGNMENT_LHS)
	}
	return st.CreateAssignmentStatementNode(lvExpr, assign, expr, semicolon)
}

func (b *ballerinaParser) parseExpression() st.STNode {
	return b.parseExpressionWithPrecedence(operatorPrecedenceDefault, true, false)
}

func (b *ballerinaParser) parseActionOrExpression() st.STNode {
	return b.parseExpressionWithPrecedence(operatorPrecedenceDefault, true, true)
}

func (b *ballerinaParser) parseActionOrExpressionInLhs(annots st.STNode) st.STNode {
	return b.parseExpressionInner(operatorPrecedenceDefault, annots, false, true, false)
}

func (b *ballerinaParser) parseExpressionPossibleRhsExpr() st.STNode {
	return b.parseExpressionWithPrecedence(operatorPrecedenceDefault, false, false)
}

func (b *ballerinaParser) isValidLVExpr(expression st.STNode) bool {
	switch expression.Kind() {
	case st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE,
		st.LIST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.ERROR_BINDING_PATTERN,
		st.WILDCARD_BINDING_PATTERN:
		return true
	case st.FIELD_ACCESS:
		fieldAccessExpressionNode, ok := expression.(*st.STFieldAccessExpressionNode)
		if !ok {
			b.internalError("expected STFieldAccessExpressionNode")
			return false
		}
		return b.isValidLVMemberExpr(fieldAccessExpressionNode.Expression)
	case st.INDEXED_EXPRESSION:
		indexedExpressionNode, ok := expression.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("expected STIndexedExpressionNode")
			return false
		}
		return b.isValidLVMemberExpr(indexedExpressionNode.ContainerExpression)
	default:
		_, ok := expression.(*st.STMissingToken)
		return ok
	}
}

func (b *ballerinaParser) isValidLVMemberExpr(expression st.STNode) bool {
	switch expression.Kind() {
	case st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE:
		return true
	case st.FIELD_ACCESS:
		fieldAccessExpressionNode, ok := expression.(*st.STFieldAccessExpressionNode)
		if !ok {
			b.internalError("expected STFieldAccessExpressionNode")
			return false
		}
		return b.isValidLVMemberExpr(fieldAccessExpressionNode.Expression)
	case st.INDEXED_EXPRESSION:
		indexedExpressionNode, ok := expression.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("expected STIndexedExpressionNode")
			return false
		}
		return b.isValidLVMemberExpr(indexedExpressionNode.ContainerExpression)
	case st.BRACED_EXPRESSION:
		bracedExpressionNode, ok := expression.(*st.STBracedExpressionNode)
		if !ok {
			b.internalError("expected STBracedExpressionNode")
			return false
		}
		return b.isValidLVMemberExpr(bracedExpressionNode.Expression)
	default:
		_, ok := expression.(*st.STMissingToken)
		return ok
	}
}

func (b *ballerinaParser) parseExpressionWithPrecedence(precedenceLevel operatorPrecedence, isRhsExpr bool, allowActions bool) st.STNode {
	return b.parseExpressionWithConditional(precedenceLevel, isRhsExpr, allowActions, false)
}

func (b *ballerinaParser) parseExpressionWithConditional(precedenceLevel operatorPrecedence, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	return b.parseExpressionWithMatchGuard(precedenceLevel, isRhsExpr, allowActions, false, isInConditionalExpr)
}

func (b *ballerinaParser) parseExpressionWithMatchGuard(precedenceLevel operatorPrecedence, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) st.STNode {
	expr := b.parseTerminalExpression(isRhsExpr, allowActions, isInConditionalExpr)
	return b.parseExpressionRhsInner(precedenceLevel, expr, isRhsExpr, allowActions, isInMatchGuard, isInConditionalExpr)
}

func (b *ballerinaParser) invalidateActionAndGetMissingExpr(node st.STNode) st.STNode {
	var identifier st.STNode
	identifier = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
	identifier = st.CloneWithTrailingInvalidNodeMinutiae(identifier, node, &common.ERROR_EXPRESSION_EXPECTED_ACTION_FOUND)
	return st.CreateSimpleNameReferenceNode(identifier)
}

func (b *ballerinaParser) parseExpressionInner(precedenceLevel operatorPrecedence, annots st.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	expr := b.parseTerminalExpressionWithAnnotations(annots, isRhsExpr, allowActions, isInConditionalExpr)
	return b.parseExpressionRhsInner(precedenceLevel, expr, isRhsExpr, allowActions, false, isInConditionalExpr)
}

func (b *ballerinaParser) parseTerminalExpression(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	annots := st.CreateEmptyNodeList()
	if b.peek().Kind() == st.AT_TOKEN {
		annots = b.parseOptionalAnnotations()
	}
	return b.parseTerminalExpressionWithAnnotations(annots, isRhsExpr, allowActions, isInConditionalExpr)
}

func (b *ballerinaParser) parseTerminalExpressionWithAnnotations(annots st.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	return b.parseTerminalExpressionInner(annots, nil, isRhsExpr, allowActions, isInConditionalExpr)
}

func (b *ballerinaParser) parseTerminalExpressionInner(annots st.STNode, qualifiers []st.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	qualifiers = b.parseExprQualifiers(qualifiers)
	nextToken := b.peek()
	annotNodeList := annots.(*st.STNodeList)
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
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		return b.parseBasicLiteral()
	case st.OPEN_PAREN_TOKEN:
		return b.parseBracedExpression(isRhsExpr, allowActions)
	case st.CHECK_KEYWORD,
		st.CHECKPANIC_KEYWORD:
		return b.parseCheckExpression(isRhsExpr, allowActions, isInConditionalExpr)
	case st.OPEN_BRACE_TOKEN:
		return b.parseMappingConstructorExpr()
	case st.TYPEOF_KEYWORD:
		return b.parseTypeofExpression(isRhsExpr, isInConditionalExpr)
	case st.PLUS_TOKEN, st.MINUS_TOKEN, st.NEGATION_TOKEN, st.EXCLAMATION_MARK_TOKEN:
		return b.parseUnaryExpression(isRhsExpr, isInConditionalExpr)
	case st.TRAP_KEYWORD:
		return b.parseTrapExpression(isRhsExpr, allowActions, isInConditionalExpr)
	case st.OPEN_BRACKET_TOKEN:
		return b.parseListConstructorExpr()
	case st.LT_TOKEN:
		return b.parseTypeCastExpr(isRhsExpr, allowActions, isInConditionalExpr)
	case st.TABLE_KEYWORD, st.STREAM_KEYWORD, st.FROM_KEYWORD, st.MAP_KEYWORD:
		return b.parseTableConstructorOrQuery(isRhsExpr, allowActions)
	case st.ERROR_KEYWORD:
		return b.parseErrorConstructorExpr(b.consume())
	case st.LET_KEYWORD:
		return b.parseLetExpression(isRhsExpr, isInConditionalExpr)
	case st.BACKTICK_TOKEN:
		return b.parseTemplateExpression()
	case st.OBJECT_KEYWORD:
		return b.parseObjectConstructorExpression(annots, qualifiers)
	case st.XML_KEYWORD:
		return b.parseXMLTemplateExpression()
	case st.RE_KEYWORD:
		return b.parseRegExpTemplateExpression()
	case st.STRING_KEYWORD:
		nextNextToken := b.getNextNextToken()
		if nextNextToken.Kind() == st.BACKTICK_TOKEN {
			return b.parseStringTemplateExpression()
		}
		return b.parseSimpleTypeInTerminalExpr()
	case st.FUNCTION_KEYWORD:
		return b.parseExplicitFunctionExpression(annots, qualifiers, isRhsExpr)
	case st.NEW_KEYWORD:
		return b.parseNewExpression()
	case st.START_KEYWORD:
		return b.parseStartAction(annots)
	case st.FLUSH_KEYWORD:
		return b.parseFlushAction()
	case st.LEFT_ARROW_TOKEN:
		return b.parseReceiveAction()
	case st.WAIT_KEYWORD:
		return b.parseWaitAction()
	case st.COMMIT_KEYWORD:
		return b.parseCommitAction()
	case st.TRANSACTIONAL_KEYWORD:
		return b.parseTransactionalExpression()
	case st.BASE16_KEYWORD,
		st.BASE64_KEYWORD:
		return b.parseByteArrayLiteral()
	case st.TRANSACTION_KEYWORD:
		return b.parseQualifiedIdentWithTransactionPrefix(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
	case st.IDENTIFIER_TOKEN:
		if b.isNaturalKeyword(nextToken) && (b.getNextNextToken().Kind() == st.OPEN_BRACE_TOKEN) {
			return b.parseNaturalExpression()
		}
		return b.parseQualifiedIdentifierInner(common.PARSER_RULE_CONTEXT_VARIABLE_REF, isInConditionalExpr)
	case st.CONST_KEYWORD:
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

func (b *ballerinaParser) parseNaturalExpression() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION)
	var optionalConstKeyword st.STNode
	if b.peek().Kind() == st.CONST_KEYWORD {
		optionalConstKeyword = b.consume()
	} else {
		optionalConstKeyword = st.CreateEmptyNode()
	}
	naturalKeyword := b.parseNaturalKeyword()
	optionalParenthesizedArgList := b.parseOptionalParenthesizedArgList()
	return b.parseNaturalExprBody(optionalConstKeyword, naturalKeyword, optionalParenthesizedArgList)
}

func (b *ballerinaParser) parseNaturalExprBody(optionalConstKeyword st.STNode, naturalKeyword st.STNode, optionalParenthesizedArgList st.STNode) st.STNode {
	openBrace := b.parseOpenBrace()
	if openBrace.IsMissing() {
		b.endContext()
		return b.createMissingNaturalExpressionNode(optionalConstKeyword, naturalKeyword,
			optionalParenthesizedArgList)
	}
	b.tokenReader.StartMode(parserModePrompt)
	prompt := b.parsePromptContent()
	closeBrace := b.parseCloseBrace()
	if b.tokenReader.GetCurrentMode() == parserModePrompt {
		b.tokenReader.EndMode()
	}
	b.endContext()
	return st.CreateNaturalExpressionNode(optionalConstKeyword, naturalKeyword,
		optionalParenthesizedArgList, openBrace, prompt, closeBrace)
}

func (b *ballerinaParser) createMissingNaturalExpressionNode(optionalConstKeyword st.STNode, naturalKeyword st.STNode, optionalParenthesizedArgList st.STNode) st.STNode {
	openBrace := st.CreateMissingToken(st.OPEN_BRACE_TOKEN, nil)
	closeBrace := st.CreateMissingToken(st.CLOSE_BRACE_TOKEN, nil)
	prompt := st.CreateEmptyNodeList()
	naturalExpr := st.CreateNaturalExpressionNode(optionalConstKeyword, naturalKeyword,
		optionalParenthesizedArgList, openBrace, prompt, closeBrace)
	naturalExpr = st.AddDiagnostic(naturalExpr, &common.ERROR_MISSING_NATURAL_PROMPT_BLOCK)
	return naturalExpr
}

func (b *ballerinaParser) parseOptionalParenthesizedArgList() st.STNode {
	if b.peek().Kind() == st.OPEN_PAREN_TOKEN {
		return b.parseParenthesizedArgList()
	}
	return st.CreateEmptyNode()
}

func (b *ballerinaParser) parsePromptContent() st.STNode {
	var items []st.STNode
	nextToken := b.peek()
	for !b.isEndOfPromptContent(nextToken.Kind()) {
		contentItem := b.parsePromptItem()
		items = append(items, contentItem)
		nextToken = b.peek()
	}
	return st.CreateNodeList(items...)
}

func (b *ballerinaParser) isEndOfPromptContent(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN, st.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parsePromptItem() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.INTERPOLATION_START_TOKEN {
		return b.parseInterpolation()
	}
	if nextToken.Kind() != st.PROMPT_CONTENT {
		nextToken = b.consume()
		return st.CreateLiteralValueTokenWithDiagnostics(st.PROMPT_CONTENT,
			nextToken.Text(), nextToken.LeadingMinutiae(), nextToken.TrailingMinutiae(),
			nextToken.Diagnostics())
	}
	return b.consume()
}

func (b *ballerinaParser) createMissingObjectConstructor(annots st.STNode, qualifierNodeList st.STNode) st.STNode {
	objectKeyword := st.CreateMissingToken(st.OBJECT_KEYWORD, nil)
	openBrace := st.CreateMissingToken(st.OPEN_BRACE_TOKEN, nil)
	closeBrace := st.CreateMissingToken(st.CLOSE_BRACE_TOKEN, nil)
	objConstructor := st.CreateObjectConstructorExpressionNode(annots, qualifierNodeList,
		objectKeyword, st.CreateEmptyNode(), openBrace, st.CreateEmptyNodeList(),
		closeBrace)
	objConstructor = st.AddDiagnostic(objConstructor,
		&common.ERROR_MISSING_OBJECT_CONSTRUCTOR_EXPRESSION)
	return objConstructor
}

func (b *ballerinaParser) parseQualifiedIdentifierOrExpression(isInConditionalExpr bool, isRhsExpr bool, allowActions bool) st.STNode {
	preDeclaredPrefix := b.consume()
	nextNextToken := b.getNextNextToken()
	if (nextNextToken.Kind() == st.IDENTIFIER_TOKEN) && (!isKeyKeyword(nextNextToken)) {
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	var context common.ParserRuleContext
	switch preDeclaredPrefix.Kind() {
	case st.TABLE_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF
	case st.STREAM_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF
	case st.ERROR_KEYWORD:
		context = common.PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF
	default:
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	solution := b.recoverWithBlockContext(b.peek(), context)
	if solution.Action == actionKeep {
		return b.parseQualifiedIdentifierWithPredeclPrefix(preDeclaredPrefix, isInConditionalExpr)
	}
	if preDeclaredPrefix.Kind() == st.ERROR_KEYWORD {
		return b.parseErrorConstructorExpr(preDeclaredPrefix)
	}
	b.startContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION)
	var tableOrQuery st.STNode
	if preDeclaredPrefix.Kind() == st.STREAM_KEYWORD {
		queryConstructType := b.parseQueryConstructType(preDeclaredPrefix, nil)
		tableOrQuery = b.parseQueryExprRhs(queryConstructType, isRhsExpr, allowActions)
	} else {
		tableOrQuery = b.parseTableConstructorOrQueryWithKeyword(preDeclaredPrefix, isRhsExpr, allowActions)
	}
	b.endContext()
	return tableOrQuery
}

func (b *ballerinaParser) validateExprAnnotsAndQualifiers(nextToken st.STToken, annots st.STNode, qualifiers []st.STNode) {
	switch nextToken.Kind() {
	case st.START_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
	case st.FUNCTION_KEYWORD, st.OBJECT_KEYWORD, st.AT_TOKEN:
		break
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			b.reportInvalidExpressionAnnots(annots, qualifiers)
			b.reportInvalidQualifierList(qualifiers)
		}
	}
}

func (b *ballerinaParser) isAnnotAllowedExprStart(nextToken st.STToken) bool {
	switch nextToken.Kind() {
	case st.START_KEYWORD, st.FUNCTION_KEYWORD, st.OBJECT_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) isValidExprStart(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN,
		st.IDENTIFIER_TOKEN,
		st.OPEN_PAREN_TOKEN,
		st.CHECK_KEYWORD,
		st.CHECKPANIC_KEYWORD,
		st.OPEN_BRACE_TOKEN,
		st.TYPEOF_KEYWORD,
		st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.NEGATION_TOKEN,
		st.EXCLAMATION_MARK_TOKEN,
		st.TRAP_KEYWORD,
		st.OPEN_BRACKET_TOKEN,
		st.LT_TOKEN,
		st.TABLE_KEYWORD,
		st.STREAM_KEYWORD,
		st.FROM_KEYWORD,
		st.ERROR_KEYWORD,
		st.LET_KEYWORD,
		st.BACKTICK_TOKEN,
		st.XML_KEYWORD,
		st.RE_KEYWORD,
		st.STRING_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.AT_TOKEN,
		st.NEW_KEYWORD,
		st.START_KEYWORD,
		st.FLUSH_KEYWORD,
		st.LEFT_ARROW_TOKEN,
		st.WAIT_KEYWORD,
		st.COMMIT_KEYWORD,
		st.SERVICE_KEYWORD,
		st.BASE16_KEYWORD,
		st.BASE64_KEYWORD,
		st.ISOLATED_KEYWORD,
		st.TRANSACTIONAL_KEYWORD,
		st.CLIENT_KEYWORD,
		st.NATURAL_KEYWORD,
		st.OBJECT_KEYWORD:
		return true
	default:
		if isPredeclaredPrefix(tokenKind) {
			return true
		}
		return b.isSimpleTypeInExpression(tokenKind)
	}
}

func (b *ballerinaParser) parseNewExpression() st.STNode {
	new := b.parseNewKeyword()
	return b.parseNewKeywordRhs(new)
}

func (b *ballerinaParser) parseNewKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.NEW_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_NEW_KEYWORD)
		return b.parseNewKeyword()
	}
}

func (b *ballerinaParser) parseNewKeywordRhs(new st.STNode) st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.OPEN_PAREN_TOKEN {
		return b.parseImplicitNewExpr(new)
	}
	if b.isClassDescriptorStartToken(nextToken.Kind()) {
		return b.parseExplicitNewExpr(new)
	}
	return b.createImplicitNewExpr(new, st.CreateEmptyNode())
}

func (b *ballerinaParser) isClassDescriptorStartToken(tokenKind st.SyntaxKind) bool {
	return ((tokenKind == st.STREAM_KEYWORD) || b.isPredeclaredIdentifier(tokenKind))
}

func (b *ballerinaParser) parseExplicitNewExpr(new st.STNode) st.STNode {
	typeDescriptor := b.parseClassDescriptor()
	parenthesizedArgsList := b.parseParenthesizedArgList()
	return st.CreateExplicitNewExpressionNode(new, typeDescriptor, parenthesizedArgsList)
}

func (b *ballerinaParser) parseClassDescriptor() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR)
	var classDescriptor st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.STREAM_KEYWORD:
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

func (b *ballerinaParser) parseImplicitNewExpr(new st.STNode) st.STNode {
	parenthesizedArgList := b.parseParenthesizedArgList()
	return b.createImplicitNewExpr(new, parenthesizedArgList)
}

func (b *ballerinaParser) createImplicitNewExpr(new st.STNode, parenthesizedArgList st.STNode) st.STNode {
	return st.CreateImplicitNewExpressionNode(new, parenthesizedArgList)
}

func (b *ballerinaParser) parseParenthesizedArgList() st.STNode {
	openParan := b.parseArgListOpenParenthesis()
	arguments := b.parseArgsList()
	closeParan := b.parseArgListCloseParenthesis()
	return st.CreateParenthesizedArgList(openParan, arguments, closeParan)
}

func (b *ballerinaParser) parseExpressionRhs(lhsExpr st.STNode, isRhsExpr bool, allowActions bool) st.STNode {
	return b.parseExpressionRhsInner(defaultOpPrecedence, lhsExpr, isRhsExpr, allowActions, false, false)
}

func (b *ballerinaParser) parseExpressionRhsInner(currentPrecedenceLevel operatorPrecedence, lhsExpr st.STNode, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) st.STNode {
	actionOrExpression := b.parseExpressionRhsInternal(currentPrecedenceLevel, lhsExpr, isRhsExpr, allowActions,
		isInMatchGuard, isInConditionalExpr)
	if ((!allowActions) && b.isAction(actionOrExpression)) && (actionOrExpression.Kind() != st.BRACED_ACTION) {
		actionOrExpression = b.invalidateActionAndGetMissingExpr(actionOrExpression)
	}
	return actionOrExpression
}

func (b *ballerinaParser) parseExpressionRhsInternal(currentPrecedenceLevel operatorPrecedence, lhsExpr st.STNode, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) st.STNode {
	nextToken := b.peek()
	if b.isAction(lhsExpr) || b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard) {
		return lhsExpr
	}
	nextTokenKind := nextToken.Kind()
	if !b.isValidExprRhsStart(nextTokenKind, lhsExpr.Kind()) {
		return b.recoverExpressionRhs(currentPrecedenceLevel, lhsExpr, isRhsExpr, allowActions, isInMatchGuard,
			isInConditionalExpr)
	}
	if (nextTokenKind == st.GT_TOKEN) && (b.peekN(2).Kind() == st.GT_TOKEN) {
		if b.peekN(3).Kind() == st.GT_TOKEN {
			nextTokenKind = st.TRIPPLE_GT_TOKEN
		} else {
			nextTokenKind = st.DOUBLE_GT_TOKEN
		}
	}
	nextOperatorPrecedence := b.getOpPrecedence(nextTokenKind)
	if currentPrecedenceLevel.isHigherThanOrEqual(nextOperatorPrecedence, allowActions) {
		return lhsExpr
	}
	var newLhsExpr st.STNode
	var operator st.STNode
	switch nextTokenKind {
	case st.OPEN_PAREN_TOKEN:
		newLhsExpr = b.parseFuncCallOrNaturalExpr(lhsExpr)
	case st.OPEN_BRACKET_TOKEN:
		newLhsExpr = b.parseMemberAccessExpr(lhsExpr, isRhsExpr)
	case st.DOT_TOKEN:
		newLhsExpr = b.parseFieldAccessOrMethodCall(lhsExpr, isInConditionalExpr)
	case st.IS_KEYWORD,
		st.NOT_IS_KEYWORD:
		newLhsExpr = b.parseTypeTestExpression(lhsExpr, isInConditionalExpr)
	case st.RIGHT_ARROW_TOKEN:
		newLhsExpr = b.parseRemoteMethodCallOrClientResourceAccessOrAsyncSendAction(lhsExpr, isRhsExpr,
			isInMatchGuard)
	case st.SYNC_SEND_TOKEN:
		newLhsExpr = b.parseSyncSendAction(lhsExpr)
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		newLhsExpr = b.parseImplicitAnonFuncWithParams(lhsExpr, isRhsExpr)
	case st.ANNOT_CHAINING_TOKEN:
		newLhsExpr = b.parseAnnotAccessExpression(lhsExpr, isInConditionalExpr)
	case st.OPTIONAL_CHAINING_TOKEN:
		newLhsExpr = b.parseOptionalFieldAccessExpression(lhsExpr, isInConditionalExpr)
	case st.QUESTION_MARK_TOKEN:
		newLhsExpr = b.parseConditionalExpression(lhsExpr, isInConditionalExpr)
	case st.DOT_LT_TOKEN:
		newLhsExpr = b.parseXMLFilterExpression(lhsExpr)
	case st.SLASH_LT_TOKEN,
		st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		st.SLASH_ASTERISK_TOKEN:
		newLhsExpr = b.parseXMLStepExpression(lhsExpr)
	default:
		if (nextTokenKind == st.SLASH_TOKEN) && (b.peekN(2).Kind() == st.LT_TOKEN) {
			expectedNodeType := b.getExpectedNodeKind(3)
			if expectedNodeType == st.XML_STEP_EXPRESSION {
				newLhsExpr = b.createXMLStepExpression(lhsExpr)
				break
			}
		}
		switch nextTokenKind {
		case st.DOUBLE_GT_TOKEN:
			operator = b.parseSignedRightShiftToken()
		case st.TRIPPLE_GT_TOKEN:
			operator = b.parseUnsignedRightShiftToken()
		default:
			operator = b.parseBinaryOperator()
		}
		rhsExpr := b.parseExpressionWithConditional(nextOperatorPrecedence, isRhsExpr, false, isInConditionalExpr)
		newLhsExpr = st.CreateBinaryExpressionNode(st.BINARY_EXPRESSION, lhsExpr, operator,
			rhsExpr)
	}
	return b.parseExpressionRhsInternal(currentPrecedenceLevel, newLhsExpr, isRhsExpr, allowActions, isInMatchGuard,
		isInConditionalExpr)
}

func (b *ballerinaParser) recoverExpressionRhs(currentPrecedenceLevel operatorPrecedence, lhsExpr st.STNode, isRhsExpr bool, allowActions bool, isInMatchGuard bool, isInConditionalExpr bool) st.STNode {
	token := b.peek()
	lhsExprKind := lhsExpr.Kind()
	var solution *solution
	if (lhsExprKind == st.QUALIFIED_NAME_REFERENCE) || (lhsExprKind == st.SIMPLE_NAME_REFERENCE) {
		solution = b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS)
	} else {
		solution = b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS)
	}
	if solution.Action == actionRemove {
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

func (b *ballerinaParser) createXMLStepExpression(lhsExpr st.STNode) st.STNode {
	var newLhsExpr st.STNode
	slashToken := b.parseSlashToken()
	ltToken := b.parseLTToken()
	var slashLT st.STNode
	if b.hasTrailingMinutiae(slashToken) || b.hasLeadingMinutiae(ltToken) {
		var diagnostics []st.STNodeDiagnostic
		diagnostics = append(diagnostics, st.CreateDiagnostic(&common.ERROR_INVALID_WHITESPACE_IN_SLASH_LT_TOKEN))
		slashLT = st.CreateMissingToken(st.SLASH_LT_TOKEN, diagnostics)
		slashLT = st.CloneWithLeadingInvalidNodeMinutiae(slashLT, slashToken, nil)
		slashLT = st.CloneWithLeadingInvalidNodeMinutiae(slashLT, ltToken, nil)
	} else {
		slashLT = st.CreateToken(st.SLASH_LT_TOKEN, slashToken.LeadingMinutiae(),
			ltToken.TrailingMinutiae())
	}
	namePattern := b.parseXMLNamePatternChain(slashLT)
	xmlStepExtends := b.parseXMLStepExtends()
	newLhsExpr = st.CreateXMLStepExpressionNode(lhsExpr, namePattern, xmlStepExtends)
	return newLhsExpr
}

func (b *ballerinaParser) getExpectedNodeKind(lookahead int) st.SyntaxKind {
	nextToken := b.peekN(lookahead)
	switch nextToken.Kind() {
	case st.ASTERISK_TOKEN:
		return st.XML_STEP_EXPRESSION
	case st.GT_TOKEN:
		break
	case st.PIPE_TOKEN:
		return b.getExpectedNodeKind(lookahead + 1)
	case st.IDENTIFIER_TOKEN:
		nextToken = b.peekN(lookahead + 1)
		switch nextToken.Kind() {
		case st.GT_TOKEN:
			break
		case st.PIPE_TOKEN:
			return b.getExpectedNodeKind(lookahead + 1)
		case st.COLON_TOKEN:
			nextToken = b.peekN(lookahead + 1)
			switch nextToken.Kind() {
			case st.ASTERISK_TOKEN, st.GT_TOKEN:
				return st.XML_STEP_EXPRESSION
			case st.IDENTIFIER_TOKEN:
				nextToken = b.peekN(lookahead + 1)
				if nextToken.Kind() == st.PIPE_TOKEN {
					return b.getExpectedNodeKind(lookahead + 1)
				}
			default:
				return st.TYPE_CAST_EXPRESSION
			}
		default:
			return st.TYPE_CAST_EXPRESSION
		}
	default:
		return st.TYPE_CAST_EXPRESSION
	}
	nextToken = b.peekN(lookahead + 1)
	switch nextToken.Kind() {
	case st.OPEN_BRACKET_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.FROM_KEYWORD,
		st.LET_KEYWORD:
		return st.XML_STEP_EXPRESSION
	default:
		if b.isValidExpressionStart(nextToken.Kind(), lookahead) {
			break
		}
		return st.XML_STEP_EXPRESSION
	}
	return st.TYPE_CAST_EXPRESSION
}

func (b *ballerinaParser) hasTrailingMinutiae(node st.STNode) bool {
	return (node.WidthWithTrailingMinutiae() > node.Width())
}

func (b *ballerinaParser) hasLeadingMinutiae(node st.STNode) bool {
	return (node.WidthWithLeadingMinutiae() > node.Width())
}

func (b *ballerinaParser) isValidExprRhsStart(tokenKind st.SyntaxKind, precedingNodeKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.OPEN_PAREN_TOKEN:
		return ((precedingNodeKind == st.QUALIFIED_NAME_REFERENCE) || (precedingNodeKind == st.SIMPLE_NAME_REFERENCE))
	case st.DOT_TOKEN,
		st.OPEN_BRACKET_TOKEN,
		st.IS_KEYWORD,
		st.RIGHT_ARROW_TOKEN,
		st.RIGHT_DOUBLE_ARROW_TOKEN,
		st.SYNC_SEND_TOKEN,
		st.ANNOT_CHAINING_TOKEN,
		st.OPTIONAL_CHAINING_TOKEN,
		st.COLON_TOKEN,
		st.DOT_LT_TOKEN,
		st.SLASH_LT_TOKEN,
		st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		st.SLASH_ASTERISK_TOKEN,
		st.NOT_IS_KEYWORD:
		return true
	case st.QUESTION_MARK_TOKEN:
		return ((b.getNextNextToken().Kind() != st.EQUAL_TOKEN) && (b.peekN(3).Kind() != st.EQUAL_TOKEN))
	default:
		return b.isBinaryOperator(tokenKind)
	}
}

func (b *ballerinaParser) parseMemberAccessExpr(lhsExpr st.STNode, isRhsExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR)
	openBracket := b.parseOpenBracket()
	keyExpr := b.parseMemberAccessKeyExprs(isRhsExpr)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	if isRhsExpr {
		listKeyExprNode, ok := keyExpr.(*st.STNodeList)
		if !ok {
			b.internalError("expected STNodeList")
			return st.CreateEmptyNode()
		}
		if listKeyExprNode.IsEmpty() {
			missingVarRef := st.CreateSimpleNameReferenceNode(st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil))
			keyExpr = st.CreateNodeList(missingVarRef)
			closeBracket = st.AddDiagnostic(closeBracket,
				&common.ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR)
		}
	}
	return st.CreateIndexedExpressionNode(lhsExpr, openBracket, keyExpr, closeBracket)
}

func (b *ballerinaParser) parseMemberAccessKeyExprs(isRhsExpr bool) st.STNode {
	var exprList []st.STNode
	var keyExpr st.STNode
	var keyExprEnd st.STNode
	for !b.isEndOfTypeList(b.peek().Kind()) {
		keyExpr = b.parseKeyExpr(isRhsExpr)
		exprList = append(exprList, keyExpr)
		keyExprEnd = b.parseMemberAccessKeyExprEnd()
		if keyExprEnd == nil {
			break
		}
		exprList = append(exprList, keyExprEnd)
	}
	return st.CreateNodeList(exprList...)
}

func (b *ballerinaParser) parseKeyExpr(isRhsExpr bool) st.STNode {
	if (!isRhsExpr) && (b.peek().Kind() == st.ASTERISK_TOKEN) {
		return st.CreateBasicLiteralNode(st.ASTERISK_LITERAL, b.consume())
	}
	return b.parseExpressionWithPrecedence(operatorPrecedenceDefault, isRhsExpr, false)
}

func (b *ballerinaParser) parseMemberAccessKeyExprEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END)
		return b.parseMemberAccessKeyExprEnd()
	}
}

func (b *ballerinaParser) parseCloseBracket() st.STNode {
	token := b.peek()
	if token.Kind() == st.CLOSE_BRACKET_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET)
		return b.parseCloseBracket()
	}
}

func (b *ballerinaParser) parseFieldAccessOrMethodCall(lhsExpr st.STNode, isInConditionalExpr bool) st.STNode {
	dotToken := b.parseDotToken()
	if b.isSpecialMethodName(b.peek()) {
		methodName := b.getKeywordAsSimpleNameRef()
		openParen := b.parseArgListOpenParenthesis()
		args := b.parseArgsList()
		closeParen := b.parseArgListCloseParenthesis()
		return st.CreateMethodCallExpressionNode(lhsExpr, dotToken, methodName, openParen, args,
			closeParen)
	}
	fieldOrMethodName := b.parseFieldAccessIdentifier(isInConditionalExpr)
	if fieldOrMethodName.Kind() == st.QUALIFIED_NAME_REFERENCE {
		return st.CreateFieldAccessExpressionNode(lhsExpr, dotToken, fieldOrMethodName)
	}
	nextToken := b.peek()
	if nextToken.Kind() == st.OPEN_PAREN_TOKEN {
		openParen := b.parseArgListOpenParenthesis()
		args := b.parseArgsList()
		closeParen := b.parseArgListCloseParenthesis()
		return st.CreateMethodCallExpressionNode(lhsExpr, dotToken, fieldOrMethodName, openParen, args,
			closeParen)
	}
	return st.CreateFieldAccessExpressionNode(lhsExpr, dotToken, fieldOrMethodName)
}

func (b *ballerinaParser) getKeywordAsSimpleNameRef() st.STNode {
	mapKeyword := b.consume()
	var methodName st.STNode
	methodName = st.CreateIdentifierTokenWithDiagnostics(mapKeyword.Text(), mapKeyword.LeadingMinutiae(),
		mapKeyword.TrailingMinutiae(), mapKeyword.Diagnostics())
	methodName = st.CreateSimpleNameReferenceNode(methodName)
	return methodName
}

func (b *ballerinaParser) parseBracedExpression(isRhsExpr bool, allowActions bool) st.STNode {
	openParen := b.parseOpenParenthesis()
	if b.peek().Kind() == st.CLOSE_PAREN_TOKEN {
		return st.CreateNilLiteralNode(openParen, b.consume())
	}
	b.startContext(common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS)
	var expr st.STNode
	if allowActions {
		expr = b.parseExpressionWithPrecedence(defaultOpPrecedence, isRhsExpr, true)
	} else {
		expr = b.parseExpressionWithPrecedence(defaultOpPrecedence, isRhsExpr, false)
	}
	return b.parseBracedExprOrAnonFuncParamRhs(openParen, expr, isRhsExpr)
}

func (b *ballerinaParser) parseBracedExprOrAnonFuncParamRhs(openParen st.STNode, expr st.STNode, isRhsExpr bool) st.STNode {
	nextToken := b.peek()
	if expr.Kind() == st.SIMPLE_NAME_REFERENCE {
		switch nextToken.Kind() {
		case st.CLOSE_PAREN_TOKEN:
			break
		case st.COMMA_TOKEN:
			return b.parseImplicitAnonFuncWithOpenParenAndFirstParam(openParen, expr, isRhsExpr)
		default:
			b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS)
			return b.parseBracedExprOrAnonFuncParamRhs(openParen, expr, isRhsExpr)
		}
	}
	closeParen := b.parseCloseParenthesis()
	b.endContext()
	if b.isAction(expr) {
		return st.CreateBracedExpressionNode(st.BRACED_ACTION, openParen, expr, closeParen)
	}
	return st.CreateBracedExpressionNode(st.BRACED_EXPRESSION, openParen, expr, closeParen)
}

func (b *ballerinaParser) isAction(node st.STNode) bool {
	switch node.Kind() {
	case st.REMOTE_METHOD_CALL_ACTION,
		st.BRACED_ACTION,
		st.CHECK_ACTION,
		st.START_ACTION,
		st.TRAP_ACTION,
		st.FLUSH_ACTION,
		st.ASYNC_SEND_ACTION,
		st.SYNC_SEND_ACTION,
		st.RECEIVE_ACTION,
		st.WAIT_ACTION,
		st.QUERY_ACTION,
		st.COMMIT_ACTION,
		st.CLIENT_RESOURCE_ACCESS_ACTION:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) isEndOfActionOrExpression(nextToken st.STToken, isRhsExpr bool, isInMatchGuard bool) bool {
	tokenKind := nextToken.Kind()
	if !isRhsExpr {
		if b.isCompoundAssignment(tokenKind) {
			return true
		}
		if isInMatchGuard && (tokenKind == st.RIGHT_DOUBLE_ARROW_TOKEN) {
			return true
		}
	}
	switch tokenKind {
	case st.EOF_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.CLOSE_BRACKET_TOKEN,
		st.SEMICOLON_TOKEN,
		st.COMMA_TOKEN,
		st.PUBLIC_KEYWORD,
		st.CONST_KEYWORD,
		st.LISTENER_KEYWORD,
		st.RESOURCE_KEYWORD,
		st.EQUAL_TOKEN,
		st.DOCUMENTATION_STRING,
		st.AT_TOKEN,
		st.AS_KEYWORD,
		st.IN_KEYWORD,
		st.FROM_KEYWORD,
		st.WHERE_KEYWORD,
		st.LET_KEYWORD,
		st.SELECT_KEYWORD,
		st.DO_KEYWORD,
		st.COLON_TOKEN,
		st.ON_KEYWORD,
		st.CONFLICT_KEYWORD,
		st.LIMIT_KEYWORD,
		st.JOIN_KEYWORD,
		st.OUTER_KEYWORD,
		st.ORDER_KEYWORD,
		st.BY_KEYWORD,
		st.ASCENDING_KEYWORD,
		st.DESCENDING_KEYWORD,
		st.EQUALS_KEYWORD,
		st.TYPE_KEYWORD:
		return true
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		return isInMatchGuard
	case st.IDENTIFIER_TOKEN:
		return isGroupOrCollectKeyword(nextToken)
	default:
		return isSimpleType(tokenKind)
	}
}

func (b *ballerinaParser) parseBasicLiteral() st.STNode {
	literalToken := b.consume()
	return b.parseBasicLiteralInner(literalToken)
}

func (b *ballerinaParser) parseBasicLiteralInner(literalToken st.STNode) st.STNode {
	var nodeKind st.SyntaxKind
	switch literalToken.Kind() {
	case st.NULL_KEYWORD:
		nodeKind = st.NULL_LITERAL
	case st.TRUE_KEYWORD, st.FALSE_KEYWORD:
		nodeKind = st.BOOLEAN_LITERAL
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		nodeKind = st.NUMERIC_LITERAL
	case st.STRING_LITERAL_TOKEN:
		nodeKind = st.STRING_LITERAL
	case st.ASTERISK_TOKEN:
		nodeKind = st.ASTERISK_LITERAL
	default:
		nodeKind = literalToken.Kind()
	}
	return st.CreateBasicLiteralNode(nodeKind, literalToken)
}

func (b *ballerinaParser) parseFuncCallOrNaturalExpr(identifier st.STNode) st.STNode {
	openParen := b.parseArgListOpenParenthesis()
	args := b.parseArgsList()
	closeParen := b.parseArgListCloseParenthesis()
	if (b.peek().Kind() == st.OPEN_BRACE_TOKEN) && b.isNaturalKeyword(identifier) {
		nameRef, ok := identifier.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("expected STSimpleNameReferenceNode")
			return st.CreateEmptyNode()
		}
		return b.parseNaturalExpressionInner(*nameRef, openParen, args, closeParen)
	}
	return st.CreateFunctionCallExpressionNode(identifier, openParen, args, closeParen)
}

func (b *ballerinaParser) parseNaturalExpressionInner(nameRef st.STSimpleNameReferenceNode, openParen st.STNode, args st.STNode, closeParen st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION)
	optionalConstKeyword := st.CreateEmptyNode()
	naturalKeyword := b.getNaturalKeyword(st.ToToken(nameRef.Name))
	parenthesizedArgList := st.CreateParenthesizedArgList(openParen, args, closeParen)
	return b.parseNaturalExprBody(optionalConstKeyword, naturalKeyword, parenthesizedArgList)
}

func (b *ballerinaParser) parseErrorBindingPatternOrErrorConstructor() st.STNode {
	return b.parseErrorConstructorExprAmbiguous(true)
}

func (b *ballerinaParser) parseErrorConstructorExpr(error st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR)
	return b.parseErrorConstructorExprInner(error, false)
}

func (b *ballerinaParser) parseErrorConstructorExprAmbiguous(isAmbiguous bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR)
	error := b.parseErrorKeyword()
	return b.parseErrorConstructorExprInner(error, isAmbiguous)
}

func (b *ballerinaParser) parseErrorConstructorExprInner(error st.STNode, isAmbiguous bool) st.STNode {
	typeReference := b.parseErrorTypeReference()
	openParen := b.parseArgListOpenParenthesis()
	functionArgs := b.parseArgsList()
	var errorArgs st.STNode
	if isAmbiguous {
		errorArgs = functionArgs
	} else {
		errorArgs = b.getErrorArgList(functionArgs)
	}
	closeParen := b.parseArgListCloseParenthesis()
	b.endContext()
	openParen = b.cloneWithDiagnosticIfListEmpty(errorArgs, openParen,
		&common.ERROR_MISSING_ARG_WITHIN_PARENTHESIS)
	return st.CreateErrorConstructorExpressionNode(error, typeReference, openParen, errorArgs,
		closeParen)
}

func (b *ballerinaParser) parseErrorTypeReference() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		return st.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			return b.parseTypeReference()
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS)
		return b.parseErrorTypeReference()
	}
}

func (b *ballerinaParser) getErrorArgList(functionArgs st.STNode) st.STNode {
	argList, ok := functionArgs.(*st.STNodeList)
	if !ok {
		b.internalError("expected *st.STNodeList")
		return st.CreateEmptyNode()
	}
	if argList.IsEmpty() {
		return argList
	}
	var errorArgList []st.STNode
	arg := argList.Get(0)
	switch arg.Kind() {
	case st.POSITIONAL_ARG:
		errorArgList = append(errorArgList, arg)
	case st.NAMED_ARG:
		arg = st.AddDiagnostic(arg,
			&common.ERROR_MISSING_ERROR_MESSAGE_IN_ERROR_CONSTRUCTOR)
		errorArgList = append(errorArgList, arg)
	default:
		arg = st.AddDiagnostic(arg,
			&common.ERROR_MISSING_ERROR_MESSAGE_IN_ERROR_CONSTRUCTOR)
		arg = st.AddDiagnostic(arg, &common.ERROR_REST_ARG_IN_ERROR_CONSTRUCTOR)
		errorArgList = append(errorArgList, arg)
	}
	diagnosticErrorCode := &common.ERROR_REST_ARG_IN_ERROR_CONSTRUCTOR
	hasPositionalArg := false
	var leadingComma st.STNode
	i := 1
	for ; i < argList.Size(); i = i + 2 {
		leadingComma = argList.Get(i)
		arg = argList.Get(i + 1)
		if arg.Kind() == st.NAMED_ARG {
			errorArgList = append(errorArgList, leadingComma, arg)
			continue
		}
		if arg.Kind() == st.POSITIONAL_ARG {
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
	return st.CreateNodeList(errorArgList...)
}

func (b *ballerinaParser) parseArgsList() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ARG_LIST)
	token := b.peek()
	if b.isEndOfParametersList(token.Kind()) {
		args := st.CreateEmptyNodeList()
		b.endContext()
		return args
	}
	firstArg := b.parseArgument()
	argsList := b.parseArgList(firstArg)
	b.endContext()
	return argsList
}

func (b *ballerinaParser) parseArgList(firstArg st.STNode) st.STNode {
	var argsList []st.STNode
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
			posArg, ok := curArg.(*st.STPositionalArgumentNode)
			if !ok {
				b.internalError("parseArgList: expected STPositionalArgumentNode")
				return st.CreateEmptyNode()
			}
			if posArg.Expression.Kind() == st.SIMPLE_NAME_REFERENCE {
				missingEqual := st.CreateMissingToken(st.EQUAL_TOKEN, nil)
				missingIdentifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
				nameRef := st.CreateSimpleNameReferenceNode(missingIdentifier)
				expr := posArg.Expression
				simpleNameExpr, ok := expr.(*st.STSimpleNameReferenceNode)
				if !ok {
					b.internalError("parseArgList: expected STSimpleNameReferenceNode")
					return st.CreateEmptyNode()
				}
				if simpleNameExpr.Name.IsMissing() {
					errorCode = &common.ERROR_MISSING_NAMED_ARG
					expr = nameRef
				}
				curArg = st.CreateNamedArgumentNode(expr, missingEqual, nameRef)
				curArg = st.AddDiagnostic(curArg, errorCode)
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
	return st.CreateNodeList(argsList...)
}

func (b *ballerinaParser) validateArgumentOrder(prevArgKind st.SyntaxKind, curArgKind st.SyntaxKind) *common.DiagnosticErrorCode {
	var errorCode *common.DiagnosticErrorCode
	switch prevArgKind {
	case st.POSITIONAL_ARG:
		// Positional args can be followed by any type of arg - no error
		errorCode = nil
	case st.NAMED_ARG:
		// Named args cannot be followed by positional args
		if curArgKind == st.POSITIONAL_ARG {
			errorCode = &common.ERROR_NAMED_ARG_FOLLOWED_BY_POSITIONAL_ARG
		}
	case st.REST_ARG:
		errorCode = &common.ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG
	default:
		b.internalError("Invalid st.SyntaxKind in an argument")
		return nil
	}
	return errorCode
}

func (b *ballerinaParser) parseArgEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ARG_END)
		return b.parseArgEnd()
	}
}

func (b *ballerinaParser) parseArgument() st.STNode {
	var arg st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ELLIPSIS_TOKEN:
		ellipsis := b.consume()
		expr := b.parseExpression()
		arg = st.CreateRestArgumentNode(ellipsis, expr)
	case st.IDENTIFIER_TOKEN:
		arg = b.parseNamedOrPositionalArg()
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			expr := b.parseExpression()
			arg = st.CreatePositionalArgumentNode(expr)
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ARG_START)
		return b.parseArgument()
	}
	return arg
}

func (b *ballerinaParser) parseNamedOrPositionalArg() st.STNode {
	argNameOrExpr := b.parseTerminalExpression(true, false, false)
	secondToken := b.peek()
	switch secondToken.Kind() {
	case st.EQUAL_TOKEN:
		if argNameOrExpr.Kind() != st.SIMPLE_NAME_REFERENCE {
			break
		}
		equal := b.parseAssignOp()
		valExpr := b.parseExpression()
		return st.CreateNamedArgumentNode(argNameOrExpr, equal, valExpr)
	case st.COMMA_TOKEN, st.CLOSE_PAREN_TOKEN:
		return st.CreatePositionalArgumentNode(argNameOrExpr)
	}
	argNameOrExpr = b.parseExpressionRhs(argNameOrExpr, true, false)
	return st.CreatePositionalArgumentNode(argNameOrExpr)
}

func (b *ballerinaParser) parseObjectTypeDescriptor(objectKeyword st.STNode, objectTypeQualifiers st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR)
	openBrace := b.parseOpenBrace()
	objectMemberDescriptors := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return st.CreateObjectTypeDescriptorNode(objectTypeQualifiers, objectKeyword, openBrace,
		objectMemberDescriptors, closeBrace)
}

func (b *ballerinaParser) parseObjectConstructorExpression(annots st.STNode, qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR)
	objectTypeQualifier := b.createObjectTypeQualNodeList(qualifiers)
	objectKeyword := b.parseObjectKeyword()
	typeReference := b.parseObjectConstructorTypeReference()
	openBrace := b.parseOpenBrace()
	objectMembers := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return st.CreateObjectConstructorExpressionNode(annots,
		objectTypeQualifier, objectKeyword, typeReference, openBrace, objectMembers, closeBrace)
}

func (b *ballerinaParser) parseObjectConstructorTypeReference() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACE_TOKEN:
		return st.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			return b.parseTypeReference()
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF)
		return b.parseObjectConstructorTypeReference()
	}
}

func (b *ballerinaParser) isPredeclaredIdentifier(tokenKind st.SyntaxKind) bool {
	return ((tokenKind == st.IDENTIFIER_TOKEN) || b.isQualifiedIdentifierPredeclaredPrefix(tokenKind))
}

func (b *ballerinaParser) parseObjectKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.OBJECT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD)
		return b.parseObjectKeyword()
	}
}

func (b *ballerinaParser) parseObjectMembers(context common.ParserRuleContext) st.STNode {
	var objectMembers []st.STNode
	for !b.isEndOfObjectTypeNode() {
		b.startContext(context)
		member := b.parseObjectMember(context)
		b.endContext()
		if member == nil {
			break
		}
		if (context == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER) && (member.Kind() == st.TYPE_REFERENCE) {
			b.addInvalidNodeToNextToken(member, &common.ERROR_TYPE_INCLUSION_IN_OBJECT_CONSTRUCTOR)
		} else {
			objectMembers = append(objectMembers, member)
		}
	}
	return st.CreateNodeList(objectMembers...)
}

func (b *ballerinaParser) parseObjectMember(context common.ParserRuleContext) st.STNode {
	var metadata st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.EOF_TOKEN,
		st.CLOSE_BRACE_TOKEN:
		return nil
	case st.ASTERISK_TOKEN,
		st.PUBLIC_KEYWORD,
		st.PRIVATE_KEYWORD,
		st.FINAL_KEYWORD,
		st.REMOTE_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.TRANSACTIONAL_KEYWORD,
		st.ISOLATED_KEYWORD,
		st.RESOURCE_KEYWORD:
		metadata = st.CreateEmptyNode()
	case st.DOCUMENTATION_STRING,
		st.AT_TOKEN:
		metadata = b.parseMetaData()
	case st.RETURN_KEYWORD:
		b.addInvalidNodeToNextToken(b.consume(), &common.ERROR_INVALID_TOKEN)
		return b.parseObjectMember(context)
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			metadata = st.CreateEmptyNode()
			break
		}
		var recoveryCtx common.ParserRuleContext
		if context == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER {
			recoveryCtx = common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START
		} else {
			recoveryCtx = common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START
		}
		solution := b.recoverWithBlockContext(b.peek(), recoveryCtx)
		if solution.Action == actionKeep {
			metadata = st.CreateEmptyNode()
			break
		}
		return b.parseObjectMember(context)
	}
	return b.parseObjectMemberWithoutMeta(metadata, context)
}

func (b *ballerinaParser) parseObjectMemberWithoutMeta(metadata st.STNode, context common.ParserRuleContext) st.STNode {
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

func (b *ballerinaParser) parseObjectMemberWithoutMetaInner(metadata st.STNode, qualifiers []st.STNode, recoveryCtx common.ParserRuleContext, isObjectTypeDesc bool) (st.STNode, []st.STNode) {
	qualifiers = b.parseObjectMemberQualifiers(qualifiers)
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.EOF_TOKEN,
		st.CLOSE_BRACE_TOKEN:
		if (metadata != nil) || (len(qualifiers) > 0) {
			return b.createMissingSimpleObjectFieldInner(metadata, qualifiers, isObjectTypeDesc)
		}
		return nil, nil
	case st.PUBLIC_KEYWORD,
		st.PRIVATE_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		var visibilityQualifier st.STNode
		visibilityQualifier = b.consume()
		if isObjectTypeDesc && (visibilityQualifier.Kind() == st.PRIVATE_KEYWORD) {
			b.addInvalidNodeToNextToken(visibilityQualifier,
				&common.ERROR_PRIVATE_QUALIFIER_IN_OBJECT_MEMBER_DESCRIPTOR)
			visibilityQualifier = st.CreateEmptyNode()
		}
		return b.parseObjectMethodOrField(metadata, visibilityQualifier, isObjectTypeDesc), qualifiers
	case st.FUNCTION_KEYWORD:
		visibilityQualifier := st.CreateEmptyNode()
		return b.parseObjectMethodOrFuncTypeDesc(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc), qualifiers
	case st.ASTERISK_TOKEN:
		b.reportInvalidMetaData(metadata, "object ty inclusion")
		b.reportInvalidQualifierList(qualifiers)
		asterisk := b.consume()
		ty := b.parseTypeReferenceInTypeInclusion()
		semicolonToken := b.parseSemicolon()
		return st.CreateTypeReferenceNode(asterisk, ty, semicolonToken), qualifiers
	case st.IDENTIFIER_TOKEN:
		if b.isObjectFieldStart() || nextToken.IsMissing() {
			return b.parseObjectField(metadata, st.CreateEmptyNode(), qualifiers, isObjectTypeDesc)
		}
		if b.isObjectMethodStart(b.getNextNextToken()) {
			b.addInvalidTokenToNextToken(b.errorHandler.ConsumeInvalidToken())
			return b.parseObjectMemberWithoutMetaInner(metadata, qualifiers, recoveryCtx, isObjectTypeDesc)
		}
		fallthrough
	default:
		if b.isTypeStartingToken(nextToken.Kind()) && (nextToken.Kind() != st.IDENTIFIER_TOKEN) {
			return b.parseObjectField(metadata, st.CreateEmptyNode(), qualifiers, isObjectTypeDesc)
		}
		solution := b.recoverWithBlockContext(b.peek(), recoveryCtx)
		if solution.Action == actionKeep {
			return b.parseObjectField(metadata, st.CreateEmptyNode(), qualifiers, isObjectTypeDesc)
		}
		return b.parseObjectMemberWithoutMetaInner(metadata, qualifiers, recoveryCtx, isObjectTypeDesc)
	}
}

func (b *ballerinaParser) isObjectFieldStart() bool {
	nextNextToken := b.getNextNextToken()
	switch nextNextToken.Kind() {
	case st.ERROR_KEYWORD, // error-binding-pattern not allowed in fields
		st.OPEN_BRACE_TOKEN:
		return false
	case st.CLOSE_BRACE_TOKEN:
		return true
	default:
		return b.isModuleVarDeclStart(1)
	}
}

func (b *ballerinaParser) isObjectMethodStart(token st.STToken) bool {
	switch token.Kind() {
	case st.FUNCTION_KEYWORD,
		st.REMOTE_KEYWORD,
		st.RESOURCE_KEYWORD,
		st.ISOLATED_KEYWORD,
		st.TRANSACTIONAL_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseObjectMethodOrField(metadata st.STNode, visibilityQualifier st.STNode, isObjectTypeDesc bool) st.STNode {
	result, _ := b.parseObjectMethodOrFieldInner(metadata, visibilityQualifier, nil, isObjectTypeDesc)
	return result
}

func (b *ballerinaParser) parseObjectMethodOrFieldInner(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, isObjectTypeDesc bool) (st.STNode, []st.STNode) {
	qualifiers = b.parseObjectMemberQualifiers(qualifiers)
	nextToken := b.peekN(1)
	nextNextToken := b.peekN(2)
	switch nextToken.Kind() {
	case st.FUNCTION_KEYWORD:
		return b.parseObjectMethodOrFuncTypeDesc(metadata, visibilityQualifier, qualifiers, isObjectTypeDesc), qualifiers
	case st.IDENTIFIER_TOKEN:
		if nextNextToken.Kind() != st.OPEN_PAREN_TOKEN {
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

func (b *ballerinaParser) parseObjectField(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, isObjectTypeDesc bool) (st.STNode, []st.STNode) {
	objectFieldQualifiers, qualifiers := b.extractObjectFieldQualifiers(qualifiers, isObjectTypeDesc)
	objectFieldQualNodeList := st.CreateNodeList(objectFieldQualifiers...)
	ty := b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	fieldName := b.parseVariableName()
	return b.parseObjectFieldRhs(metadata, visibilityQualifier, objectFieldQualNodeList, ty, fieldName,
		isObjectTypeDesc), qualifiers
}

func (b *ballerinaParser) extractObjectFieldQualifiers(qualifiers []st.STNode, isObjectTypeDesc bool) ([]st.STNode, []st.STNode) {
	var objectFieldQualifiers []st.STNode
	if len(qualifiers) != 0 && (!isObjectTypeDesc) {
		firstQualifier := qualifiers[0]
		if firstQualifier.Kind() == st.FINAL_KEYWORD {
			objectFieldQualifiers = append(objectFieldQualifiers, qualifiers[0])
			qualifiers = qualifiers[1:]
		}
	}
	return objectFieldQualifiers, qualifiers
}

func (b *ballerinaParser) parseObjectFieldRhs(metadata st.STNode, visibilityQualifier st.STNode, qualifiers st.STNode, ty st.STNode, fieldName st.STNode, isObjectTypeDesc bool) st.STNode {
	nextToken := b.peek()
	var equalsToken st.STNode
	var expression st.STNode
	var semicolonToken st.STNode
	switch nextToken.Kind() {
	case st.SEMICOLON_TOKEN:
		equalsToken = st.CreateEmptyNode()
		expression = st.CreateEmptyNode()
		semicolonToken = b.parseSemicolon()
	case st.EQUAL_TOKEN:
		equalsToken = b.parseAssignOp()
		expression = b.parseExpression()
		semicolonToken = b.parseSemicolon()
		if isObjectTypeDesc {
			fieldName = st.CloneWithTrailingInvalidNodeMinutiae(fieldName, equalsToken,
				&common.ERROR_FIELD_INITIALIZATION_NOT_ALLOWED_IN_OBJECT_TYPE)
			fieldName = st.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(fieldName, expression)
			equalsToken = st.CreateEmptyNode()
			expression = st.CreateEmptyNode()
		}
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS)
		return b.parseObjectFieldRhs(metadata, visibilityQualifier, qualifiers, ty, fieldName,
			isObjectTypeDesc)
	}
	return st.CreateObjectFieldNode(metadata, visibilityQualifier, qualifiers, ty, fieldName,
		equalsToken, expression, semicolonToken)
}

func (b *ballerinaParser) parseObjectMethodOrFuncTypeDesc(metadata st.STNode, visibilityQualifier st.STNode, qualifiers []st.STNode, isObjectTypeDesc bool) st.STNode {
	return b.parseFuncDefOrFuncTypeDesc(metadata, visibilityQualifier, qualifiers, true, isObjectTypeDesc)
}

func (b *ballerinaParser) parseRelativeResourcePath() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH)
	var pathElementList []st.STNode
	nextToken := b.peek()
	if nextToken.Kind() == st.DOT_TOKEN {
		pathElementList = append(pathElementList, b.consume())
		b.endContext()
		return st.CreateNodeList(pathElementList...)
	}
	pathSegment := b.parseResourcePathSegment(true)
	pathElementList = append(pathElementList, pathSegment)
	var leadingSlash st.STNode
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

func (b *ballerinaParser) isEndRelativeResourcePath(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.EOF_TOKEN, st.OPEN_PAREN_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) createResourcePathNodeList(pathElementList []st.STNode) st.STNode {
	if len(pathElementList) == 0 {
		return st.CreateEmptyNodeList()
	}
	var validatedList []st.STNode
	firstElement := pathElementList[0]
	validatedList = append(validatedList, firstElement)
	hasRestPram := (firstElement.Kind() == st.RESOURCE_PATH_REST_PARAM)
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
		hasRestPram = (pathSegment.Kind() == st.RESOURCE_PATH_REST_PARAM)
		validatedList = append(validatedList, leadingSlash)
		validatedList = append(validatedList, pathSegment)
	}
	return st.CreateNodeList(validatedList...)
}

func (b *ballerinaParser) parseResourcePathSegment(isFirstSegment bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		if ((isFirstSegment && nextToken.IsMissing()) && b.isInvalidNodeStackEmpty()) && (b.getNextNextToken().Kind() == st.SLASH_TOKEN) {
			b.removeInsertedToken()
			return st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
				&common.ERROR_RESOURCE_PATH_CANNOT_BEGIN_WITH_SLASH)
		}
		return b.consume()
	case st.OPEN_BRACKET_TOKEN:
		return b.parseResourcePathParameter()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT)
		return b.parseResourcePathSegment(isFirstSegment)
	}
}

func (b *ballerinaParser) parseResourcePathParameter() st.STNode {
	openBracket := b.parseOpenBracket()
	annots := b.parseOptionalAnnotations()
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM)
	ellipsis := b.parseOptionalEllipsis()
	paramName := b.parseOptionalPathParamName()
	closeBracket := b.parseCloseBracket()
	var pathPramKind st.SyntaxKind
	if ellipsis == nil {
		pathPramKind = st.RESOURCE_PATH_SEGMENT_PARAM
	} else {
		pathPramKind = st.RESOURCE_PATH_REST_PARAM
	}
	return st.CreateResourcePathParameterNode(pathPramKind, openBracket, annots, ty, ellipsis,
		paramName, closeBracket)
}

func (b *ballerinaParser) parseOptionalPathParamName() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		return b.consume()
	case st.CLOSE_BRACKET_TOKEN:
		return st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME)
		return b.parseOptionalPathParamName()
	}
}

func (b *ballerinaParser) parseOptionalEllipsis() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ELLIPSIS_TOKEN:
		return b.consume()
	case st.IDENTIFIER_TOKEN, st.CLOSE_BRACKET_TOKEN:
		return st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS)
		return b.parseOptionalEllipsis()
	}
}

func (b *ballerinaParser) parseRelativeResourcePathEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN, st.EOF_TOKEN:
		return nil
	case st.SLASH_TOKEN:
		return b.consume()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END)
		return b.parseRelativeResourcePathEnd()
	}
}

func (b *ballerinaParser) parseIfElseBlock() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_IF_BLOCK)
	ifKeyword := b.parseIfKeyword()
	condition := b.parseExpression()
	ifBody := b.parseBlockNode()
	b.endContext()
	elseBody := b.parseElseBlock()
	return st.CreateIfElseStatementNode(ifKeyword, condition, ifBody, elseBody)
}

func (b *ballerinaParser) parseIfKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.IF_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IF_KEYWORD)
		return b.parseIfKeyword()
	}
}

func (b *ballerinaParser) parseElseKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ELSE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ELSE_KEYWORD)
		return b.parseElseKeyword()
	}
}

func (b *ballerinaParser) parseBlockNode() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
	openBrace := b.parseOpenBrace()
	stmts := b.parseStatements()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return st.CreateBlockStatementNode(openBrace, stmts, closeBrace)
}

func (b *ballerinaParser) parseElseBlock() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() != st.ELSE_KEYWORD {
		return st.CreateEmptyNode()
	}
	elseKeyword := b.parseElseKeyword()
	elseBody := b.parseElseBody()
	return st.CreateElseBlockNode(elseKeyword, elseBody)
}

func (b *ballerinaParser) parseElseBody() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IF_KEYWORD:
		return b.parseIfElseBlock()
	case st.OPEN_BRACE_TOKEN:
		return b.parseBlockNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ELSE_BODY)
		return b.parseElseBody()
	}
}

func (b *ballerinaParser) parseDoStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_DO_BLOCK)
	doKeyword := b.parseDoKeyword()
	doBody := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateDoStatementNode(doKeyword, doBody, onFailClause)
}

func (b *ballerinaParser) parseWhileStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_WHILE_BLOCK)
	whileKeyword := b.parseWhileKeyword()
	condition := b.parseExpression()
	whileBody := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateWhileStatementNode(whileKeyword, condition, whileBody, onFailClause)
}

func (b *ballerinaParser) parseWhileKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.WHILE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_WHILE_KEYWORD)
		return b.parseWhileKeyword()
	}
}

func (b *ballerinaParser) parsePanicStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_PANIC_STMT)
	panicKeyword := b.parsePanicKeyword()
	expression := b.parseExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreatePanicStatementNode(panicKeyword, expression, semicolon)
}

func (b *ballerinaParser) parsePanicKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.PANIC_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PANIC_KEYWORD)
		return b.parsePanicKeyword()
	}
}

func (b *ballerinaParser) parseCheckExpression(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	checkingKeyword := b.parseCheckingKeyword()
	expr := b.parseExpressionWithConditional(operatorPrecedenceExpressionAction, isRhsExpr, allowActions, isInConditionalExpr)
	if b.isAction(expr) {
		return st.CreateCheckExpressionNode(st.CHECK_ACTION, checkingKeyword, expr)
	} else {
		return st.CreateCheckExpressionNode(st.CHECK_EXPRESSION, checkingKeyword, expr)
	}
}

func (b *ballerinaParser) parseCheckingKeyword() st.STNode {
	token := b.peek()
	if (token.Kind() == st.CHECK_KEYWORD) || (token.Kind() == st.CHECKPANIC_KEYWORD) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD)
		return b.parseCheckingKeyword()
	}
}

func (b *ballerinaParser) parseContinueStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT)
	continueKeyword := b.parseContinueKeyword()
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateContinueStatementNode(continueKeyword, semicolon)
}

func (b *ballerinaParser) parseContinueKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.CONTINUE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD)
		return b.parseContinueKeyword()
	}
}

func (b *ballerinaParser) parseFailStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FAIL_STATEMENT)
	failKeyword := b.parseFailKeyword()
	expr := b.parseExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateFailStatementNode(failKeyword, expr, semicolon)
}

func (b *ballerinaParser) parseFailKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.FAIL_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FAIL_KEYWORD)
		return b.parseFailKeyword()
	}
}

func (b *ballerinaParser) parseReturnStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RETURN_STMT)
	returnKeyword := b.parseReturnKeyword()
	returnRhs := b.parseReturnStatementRhs(returnKeyword)
	b.endContext()
	return returnRhs
}

func (b *ballerinaParser) parseReturnKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.RETURN_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RETURN_KEYWORD)
		return b.parseReturnKeyword()
	}
}

func (b *ballerinaParser) parseBreakStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BREAK_STATEMENT)
	breakKeyword := b.parseBreakKeyword()
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateBreakStatementNode(breakKeyword, semicolon)
}

func (b *ballerinaParser) parseBreakKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.BREAK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BREAK_KEYWORD)
		return b.parseBreakKeyword()
	}
}

func (b *ballerinaParser) parseReturnStatementRhs(returnKeyword st.STNode) st.STNode {
	var expr st.STNode
	token := b.peek()
	switch token.Kind() {
	case st.SEMICOLON_TOKEN:
		expr = st.CreateEmptyNode()
	default:
		expr = b.parseActionOrExpression()
	}
	semicolon := b.parseSemicolon()
	return st.CreateReturnStatementNode(returnKeyword, expr, semicolon)
}

func (b *ballerinaParser) parseMappingConstructorExpr() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
	openBrace := b.parseOpenBrace()
	fields := b.parseMappingConstructorFields()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return st.CreateMappingConstructorExpressionNode(openBrace, fields, closeBrace)
}

func (b *ballerinaParser) parseMappingConstructorFields() st.STNode {
	nextToken := b.peek()
	if b.isEndOfMappingConstructor(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	var fields []st.STNode
	field := b.parseMappingField(common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD)
	if field != nil {
		fields = append(fields, field)
	}
	return b.finishParseMappingConstructorFields(fields)
}

func (b *ballerinaParser) finishParseMappingConstructorFields(fields []st.STNode) st.STNode {
	var nextToken st.STToken
	var mappingFieldEnd st.STNode
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
	return st.CreateNodeList(fields...)
}

func (b *ballerinaParser) parseMappingFieldEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END)
		return b.parseMappingFieldEnd()
	}
}

func (b *ballerinaParser) isEndOfMappingConstructor(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.IDENTIFIER_TOKEN, st.READONLY_KEYWORD:
		return false
	case st.EOF_TOKEN,
		st.DOCUMENTATION_STRING,
		st.AT_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.SEMICOLON_TOKEN,
		st.PUBLIC_KEYWORD,
		st.PRIVATE_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.RETURNS_KEYWORD,
		st.SERVICE_KEYWORD,
		st.TYPE_KEYWORD,
		st.LISTENER_KEYWORD,
		st.CONST_KEYWORD,
		st.FINAL_KEYWORD,
		st.RESOURCE_KEYWORD:
		return true
	default:
		return isSimpleType(tokenKind)
	}
}

func (b *ballerinaParser) parseMappingField(fieldContext common.ParserRuleContext) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		readonlyKeyword := st.CreateEmptyNode()
		return b.parseSpecificFieldWithOptionalValue(readonlyKeyword)
	case st.STRING_LITERAL_TOKEN:
		readonlyKeyword := st.CreateEmptyNode()
		return b.parseQualifiedSpecificField(readonlyKeyword)
	case st.READONLY_KEYWORD:
		readonlyKeyword := b.parseReadonlyKeyword()
		return b.parseSpecificField(readonlyKeyword)
	case st.OPEN_BRACKET_TOKEN:
		return b.parseComputedField()
	case st.ELLIPSIS_TOKEN:
		ellipsis := b.parseEllipsis()
		expr := b.parseExpression()
		return st.CreateSpreadFieldNode(ellipsis, expr)
	case st.CLOSE_BRACE_TOKEN:
		if fieldContext == common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD {
			return nil
		}
		fallthrough
	default:
		b.recoverWithBlockContext(nextToken, fieldContext)
		return b.parseMappingField(fieldContext)
	}
}

func (b *ballerinaParser) parseSpecificField(readonlyKeyword st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.STRING_LITERAL_TOKEN:
		return b.parseQualifiedSpecificField(readonlyKeyword)
	case st.IDENTIFIER_TOKEN:
		return b.parseSpecificFieldWithOptionalValue(readonlyKeyword)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD)
		return b.parseSpecificField(readonlyKeyword)
	}
}

func (b *ballerinaParser) parseQualifiedSpecificField(readonlyKeyword st.STNode) st.STNode {
	key := b.parseStringLiteral()
	colon := b.parseColon()
	valueExpr := b.parseExpression()
	return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
}

func (b *ballerinaParser) parseSpecificFieldWithOptionalValue(readonlyKeyword st.STNode) st.STNode {
	key := b.parseIdentifier(common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME)
	return b.parseSpecificFieldRhs(readonlyKeyword, key)
}

func (b *ballerinaParser) parseSpecificFieldRhs(readonlyKeyword st.STNode, key st.STNode) st.STNode {
	var colon st.STNode
	var valueExpr st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COLON_TOKEN:
		colon = b.parseColon()
		valueExpr = b.parseExpression()
	case st.COMMA_TOKEN:
		colon = st.CreateEmptyNode()
		valueExpr = st.CreateEmptyNode()
	default:
		if b.isEndOfMappingConstructor(nextToken.Kind()) {
			colon = st.CreateEmptyNode()
			valueExpr = st.CreateEmptyNode()
			break
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS)
		return b.parseSpecificFieldRhs(readonlyKeyword, key)
	}
	return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
}

func (b *ballerinaParser) parseStringLiteral() st.STNode {
	token := b.peek()
	var stringLiteral st.STNode
	if token.Kind() == st.STRING_LITERAL_TOKEN {
		stringLiteral = b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN)
		return b.parseStringLiteral()
	}
	return b.parseBasicLiteralInner(stringLiteral)
}

func (b *ballerinaParser) parseColon() st.STNode {
	token := b.peek()
	if token.Kind() == st.COLON_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COLON)
		return b.parseColon()
	}
}

func (b *ballerinaParser) parseReadonlyKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.READONLY_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_READONLY_KEYWORD)
		return b.parseReadonlyKeyword()
	}
}

func (b *ballerinaParser) parseComputedField() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME)
	openBracket := b.parseOpenBracket()
	fieldNameExpr := b.parseExpression()
	closeBracket := b.parseCloseBracket()
	b.endContext()
	colon := b.parseColon()
	valueExpr := b.parseExpression()
	return st.CreateComputedNameFieldNode(openBracket, fieldNameExpr, closeBracket, colon, valueExpr)
}

func (b *ballerinaParser) parseOpenBracket() st.STNode {
	token := b.peek()
	if token.Kind() == st.OPEN_BRACKET_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OPEN_BRACKET)
		return b.parseOpenBracket()
	}
}

func (b *ballerinaParser) parseCompoundAssignmentStmtRhs(lvExpr st.STNode) st.STNode {
	binaryOperator := b.parseCompoundBinaryOperator()
	equalsToken := b.parseAssignOp()
	expr := b.parseActionOrExpression()
	semicolon := b.parseSemicolon()
	b.endContext()
	lvExprValid := b.isValidLVExpr(lvExpr)
	if !lvExprValid {
		identifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		simpleNameRef := st.CreateSimpleNameReferenceNode(identifier)
		lvExpr = st.CloneWithLeadingInvalidNodeMinutiae(simpleNameRef, lvExpr,
			&common.ERROR_INVALID_EXPR_IN_COMPOUND_ASSIGNMENT_LHS)
	}
	return st.CreateCompoundAssignmentStatementNode(lvExpr, binaryOperator, equalsToken, expr,
		semicolon)
}

func (b *ballerinaParser) parseCompoundBinaryOperator() st.STNode {
	token := b.peek()
	if b.isCompoundAssignment(token.Kind()) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR)
		return b.parseCompoundBinaryOperator()
	}
}

func (b *ballerinaParser) parseServiceDeclOrVarDecl(metadata st.STNode, publicQualifier st.STNode, qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_SERVICE_DECL)
	serviceDeclQualList, qualifiers := b.extractServiceDeclQualifiers(qualifiers)
	serviceKeyword, qualifiers := b.extractServiceKeyword(qualifiers)
	typeDesc := b.parseServiceDeclTypeDescriptor(qualifiers)
	if (typeDesc != nil) && (typeDesc.Kind() == st.OBJECT_TYPE_DESC) {
		return b.finishParseServiceDeclOrVarDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword,
			typeDesc)
	} else {
		return b.parseServiceDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword, typeDesc)
	}
}

func (b *ballerinaParser) finishParseServiceDeclOrVarDecl(metadata st.STNode, publicQualifier st.STNode, serviceDeclQualList []st.STNode, serviceKeyword st.STNode, typeDesc st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.SLASH_TOKEN, st.ON_KEYWORD:
		return b.parseServiceDecl(metadata, publicQualifier, serviceDeclQualList, serviceKeyword, typeDesc)
	case st.OPEN_BRACKET_TOKEN,
		st.IDENTIFIER_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.ERROR_KEYWORD:
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

func (b *ballerinaParser) extractServiceDeclQualifiers(qualifierList []st.STNode) ([]st.STNode, []st.STNode) {
	var validatedList []st.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if qualifier.Kind() == st.SERVICE_KEYWORD {
			qualifierList = qualifierList[i:]
			break
		}
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, st.ToToken(st.ToToken(qualifier)).Text())
			continue
		}
		if qualifier.Kind() == st.ISOLATED_KEYWORD {
			validatedList = append(validatedList, qualifier)
			continue
		}
		if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				st.ToToken(st.ToToken(qualifier)).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(st.ToToken(qualifier)).Text())
		}
	}
	return validatedList, qualifierList
}

func (b *ballerinaParser) extractServiceKeyword(qualifierList []st.STNode) (st.STNode, []st.STNode) {
	if len(qualifierList) == 0 {
		b.internalError("assertion failed")
		return st.CreateEmptyNode(), nil
	}
	serviceKeyword := qualifierList[0]
	qualifierList = qualifierList[1:]
	if serviceKeyword.Kind() != st.SERVICE_KEYWORD {
		b.internalError("assertion failed")
		return st.CreateEmptyNode(), nil
	}
	return serviceKeyword, qualifierList
}

func (b *ballerinaParser) parseServiceDecl(metadata st.STNode, publicQualifier st.STNode, qualList []st.STNode, serviceKeyword st.STNode, serviceType st.STNode) st.STNode {
	if publicQualifier != nil {
		if len(qualList) != 0 {
			b.updateFirstNodeInListWithLeadingInvalidNode(qualList, publicQualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED)
		} else {
			serviceKeyword = st.CloneWithLeadingInvalidNodeMinutiae(serviceKeyword, publicQualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED)
		}
	}
	qualNodeList := st.CreateNodeList(qualList...)
	resourcePath := b.parseOptionalAbsolutePathOrStringLiteral()
	onKeyword := b.parseOnKeyword()
	expressionList := b.parseListeners()
	openBrace := b.parseOpenBrace()
	objectMembers := b.parseObjectMembers(common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)
	closeBrace := b.parseCloseBrace()
	semicolon := b.parseOptionalSemicolon()
	onKeyword = b.cloneWithDiagnosticIfListEmpty(expressionList, onKeyword, &common.ERROR_MISSING_EXPRESSION)
	b.endContext()
	return st.CreateServiceDeclarationNode(metadata, qualNodeList, serviceKeyword, serviceType,
		resourcePath, onKeyword, expressionList, openBrace, objectMembers, closeBrace, semicolon)
}

func (b *ballerinaParser) parseServiceDeclTypeDescriptor(qualifiers []st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.SLASH_TOKEN,
		st.ON_KEYWORD,
		st.STRING_LITERAL_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return st.CreateEmptyNode()
	default:
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE)
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE)
		return b.parseServiceDeclTypeDescriptor(qualifiers)
	}
}

func (b *ballerinaParser) parseOptionalAbsolutePathOrStringLiteral() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.SLASH_TOKEN:
		return b.parseAbsoluteResourcePath()
	case st.STRING_LITERAL_TOKEN:
		stringLiteralToken := b.consume()
		stringLiteralNode := b.parseBasicLiteralInner(stringLiteralToken)
		return st.CreateNodeList(stringLiteralNode)
	case st.ON_KEYWORD:
		return st.CreateEmptyNodeList()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH)
		return b.parseOptionalAbsolutePathOrStringLiteral()
	}
}

func (b *ballerinaParser) parseAbsoluteResourcePath() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH)
	var identifierList []st.STNode
	nextToken := b.peek()
	var leadingSlash st.STNode
	isInitialSlash := true
	for !b.isEndAbsoluteResourcePath(nextToken.Kind()) {
		leadingSlash = b.parseAbsoluteResourcePathEnd(isInitialSlash)
		if leadingSlash == nil {
			break
		}
		identifierList = append(identifierList, leadingSlash)
		nextToken = b.peek()
		if isInitialSlash && (nextToken.Kind() == st.ON_KEYWORD) {
			break
		}
		isInitialSlash = false
		leadingSlash = b.parseIdentifier(common.PARSER_RULE_CONTEXT_IDENTIFIER)
		identifierList = append(identifierList, leadingSlash)
		nextToken = b.peek()
	}
	b.endContext()
	return st.CreateNodeList(identifierList...)
}

func (b *ballerinaParser) isEndAbsoluteResourcePath(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.EOF_TOKEN, st.ON_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseAbsoluteResourcePathEnd(isInitialSlash bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ON_KEYWORD, st.EOF_TOKEN:
		return nil
	case st.SLASH_TOKEN:
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
func (b *ballerinaParser) parseServiceKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.SERVICE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD)
		return b.parseServiceKeyword()
	}
}

func (b *ballerinaParser) isCompoundAssignment(tokenKind st.SyntaxKind) bool {
	return (isCompoundBinaryOperator(tokenKind) && (b.getNextNextToken().Kind() == st.EQUAL_TOKEN))
}

func (b *ballerinaParser) parseOnKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ON_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ON_KEYWORD)
		return b.parseOnKeyword()
	}
}

func (b *ballerinaParser) parseListeners() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LISTENERS_LIST)
	var listeners []st.STNode
	nextToken := b.peek()
	if b.isEndOfListeners(nextToken.Kind()) {
		b.endContext()
		return st.CreateEmptyNodeList()
	}
	expr := b.parseExpression()
	listeners = append(listeners, expr)
	var listenersMemberEnd st.STNode
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
	return st.CreateNodeList(listeners...)
}

func (b *ballerinaParser) isEndOfListeners(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.OPEN_BRACE_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseListenersMemberEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.OPEN_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END)
		return b.parseListenersMemberEnd()
	}
}

func (b *ballerinaParser) isServiceDeclStart(lookahead int) bool {
	switch b.peekN(lookahead + 1).Kind() {
	case st.IDENTIFIER_TOKEN:
		tokenAfterIdentifier := b.peekN(lookahead + 2).Kind()
		switch tokenAfterIdentifier {
		case st.ON_KEYWORD,
			// service foo on ...
			st.OPEN_BRACE_TOKEN:
			return true
		case st.EQUAL_TOKEN,
			// service foo = ...
			st.SEMICOLON_TOKEN,
			// service foo;
			st.QUESTION_MARK_TOKEN:
			return false
		default:
			return false
		}
	case st.ON_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseListenerDeclaration(metadata st.STNode, qualifier st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LISTENER_DECL)
	listenerKeyword := b.parseListenerKeyword()
	if b.peek().Kind() == st.IDENTIFIER_TOKEN {
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
	return st.CreateListenerDeclarationNode(metadata, qualifier, listenerKeyword, typeDesc, variableName,
		equalsToken, initializer, semicolonToken)
}

func (b *ballerinaParser) parseListenerKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.LISTENER_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD)
		return b.parseListenerKeyword()
	}
}

func (b *ballerinaParser) parseConstantDeclaration(metadata st.STNode, qualifier st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONSTANT_DECL)
	constKeyword := b.parseConstantKeyword()
	return b.parseConstDecl(metadata, qualifier, constKeyword)
}

func (b *ballerinaParser) parseConstDecl(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ANNOTATION_KEYWORD:
		b.endContext()
		return b.parseAnnotationDeclaration(metadata, qualifier, constKeyword)
	case st.IDENTIFIER_TOKEN:
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
	return st.CreateConstantDeclarationNode(metadata, qualifier, constKeyword, typeDesc, variableName,
		equalsToken, initializer, semicolonToken)
}

func (b *ballerinaParser) parseConstantOrListenerDeclWithOptionalType(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode, isListener bool) st.STNode {
	varNameOrTypeName := b.parseStatementStartIdentifier()
	return b.parseConstantOrListenerDeclRhs(metadata, qualifier, constKeyword, varNameOrTypeName, isListener)
}

func (b *ballerinaParser) parseConstantOrListenerDeclRhs(metadata st.STNode, qualifier st.STNode, keyword st.STNode, typeOrVarName st.STNode, isListener bool) st.STNode {
	if typeOrVarName.Kind() == st.QUALIFIED_NAME_REFERENCE {
		ty := typeOrVarName
		variableName := b.parseVariableName()
		return b.parseListenerOrConstRhs(metadata, qualifier, keyword, isListener, ty, variableName)
	}
	var ty st.STNode
	var variableName st.STNode
	switch b.peek().Kind() {
	case st.IDENTIFIER_TOKEN:
		ty = typeOrVarName
		variableName = b.parseVariableName()
	case st.EQUAL_TOKEN:
		simpleNameNode, ok := typeOrVarName.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("parseConstantOrListenerDeclRhs: expected STSimpleNameReferenceNode")
			return st.CreateEmptyNode()
		}
		variableName = simpleNameNode.Name
		ty = st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_CONST_DECL_RHS)
		return b.parseConstantOrListenerDeclRhs(metadata, qualifier, keyword, typeOrVarName, isListener)
	}
	return b.parseListenerOrConstRhs(metadata, qualifier, keyword, isListener, ty, variableName)
}

func (b *ballerinaParser) parseListenerOrConstRhs(metadata st.STNode, qualifier st.STNode, keyword st.STNode, isListener bool, ty st.STNode, variableName st.STNode) st.STNode {
	equalsToken := b.parseAssignOp()
	initializer := b.parseExpression()
	semicolonToken := b.parseSemicolon()
	if isListener {
		return st.CreateListenerDeclarationNode(metadata, qualifier, keyword, ty, variableName,
			equalsToken, initializer, semicolonToken)
	}
	return st.CreateConstantDeclarationNode(metadata, qualifier, keyword, ty, variableName,
		equalsToken, initializer, semicolonToken)
}

func (b *ballerinaParser) parseConstantKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.CONST_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CONST_KEYWORD)
		return b.parseConstantKeyword()
	}
}

func (b *ballerinaParser) parseTypeofExpression(isRhsExpr bool, isInConditionalExpr bool) st.STNode {
	typeofKeyword := b.parseTypeofKeyword()
	expr := b.parseExpressionWithConditional(operatorPrecedenceUnary, isRhsExpr, false, isInConditionalExpr)
	return st.CreateTypeofExpressionNode(typeofKeyword, expr)
}

func (b *ballerinaParser) parseTypeofKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.TYPEOF_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD)
		return b.parseTypeofKeyword()
	}
}

func (b *ballerinaParser) parseOptionalTypeDescriptor(typeDescriptorNode st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR)
	questionMarkToken := b.parseQuestionMark()
	b.endContext()
	return b.createOptionalTypeDesc(typeDescriptorNode, questionMarkToken)
}

func (b *ballerinaParser) createOptionalTypeDesc(typeDescNode st.STNode, questionMarkToken st.STNode) st.STNode {
	if typeDescNode.Kind() == st.UNION_TYPE_DESC {
		unionTypeDesc, ok := typeDescNode.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		middleTypeDesc := b.createOptionalTypeDesc(unionTypeDesc.RightTypeDesc, questionMarkToken)
		typeDescNode = b.mergeTypesWithUnion(unionTypeDesc.LeftTypeDesc, unionTypeDesc.PipeToken, middleTypeDesc)
	} else if typeDescNode.Kind() == st.INTERSECTION_TYPE_DESC {
		intersectionTypeDesc, ok := typeDescNode.(*st.STIntersectionTypeDescriptorNode)
		if !ok {
			b.internalError("expected st.STIntersectionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		middleTypeDesc := b.createOptionalTypeDesc(intersectionTypeDesc.RightTypeDesc, questionMarkToken)
		typeDescNode = b.mergeTypesWithIntersection(intersectionTypeDesc.LeftTypeDesc,
			intersectionTypeDesc.BitwiseAndToken, middleTypeDesc)
	} else {
		typeDescNode = b.validateForUsageOfVar(typeDescNode)
		typeDescNode = st.CreateOptionalTypeDescriptorNode(typeDescNode, questionMarkToken)
	}
	return typeDescNode
}

func (b *ballerinaParser) parseUnaryExpression(isRhsExpr bool, isInConditionalExpr bool) st.STNode {
	unaryOperator := b.parseUnaryOperator()
	expr := b.parseExpressionWithConditional(operatorPrecedenceUnary, isRhsExpr, false, isInConditionalExpr)
	return st.CreateUnaryExpressionNode(unaryOperator, expr)
}

func (b *ballerinaParser) parseUnaryOperator() st.STNode {
	token := b.peek()
	if b.isUnaryOperator(token.Kind()) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_UNARY_OPERATOR)
		return b.parseUnaryOperator()
	}
}

func (b *ballerinaParser) isUnaryOperator(kind st.SyntaxKind) bool {
	switch kind {
	case st.PLUS_TOKEN, st.MINUS_TOKEN, st.NEGATION_TOKEN, st.EXCLAMATION_MARK_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseArrayTypeDescriptor(memberTypeDesc st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR)
	openBracketToken := b.parseOpenBracket()
	arrayLengthNode := b.parseArrayLength()
	closeBracketToken := b.parseCloseBracket()
	b.endContext()
	return b.createArrayTypeDesc(memberTypeDesc, openBracketToken, arrayLengthNode, closeBracketToken)
}

func (b *ballerinaParser) createArrayTypeDesc(memberTypeDesc st.STNode, openBracketToken st.STNode, arrayLengthNode st.STNode, closeBracketToken st.STNode) st.STNode {
	memberTypeDesc = b.validateForUsageOfVar(memberTypeDesc)
	if arrayLengthNode != nil {
		switch arrayLengthNode.Kind() {
		case st.ASTERISK_LITERAL,
			st.SIMPLE_NAME_REFERENCE,
			st.QUALIFIED_NAME_REFERENCE:
			break
		case st.NUMERIC_LITERAL:
			numericLiteralKind := arrayLengthNode.ChildInBucket(0).Kind()
			if (numericLiteralKind == st.DECIMAL_INTEGER_LITERAL_TOKEN) || (numericLiteralKind == st.HEX_INTEGER_LITERAL_TOKEN) {
				break
			}
		default:
			openBracketToken = st.CloneWithTrailingInvalidNodeMinutiae(openBracketToken,
				arrayLengthNode, &common.ERROR_INVALID_ARRAY_LENGTH)
			arrayLengthNode = st.CreateEmptyNode()
		}
	}
	var arrayDimensions []st.STNode
	if memberTypeDesc.Kind() == st.ARRAY_TYPE_DESC {
		innerArrayType, ok := memberTypeDesc.(*st.STArrayTypeDescriptorNode)
		if !ok {
			b.internalError("expected st.STArrayTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		innerArrayDimensions := innerArrayType.Dimensions
		dimensionCount := innerArrayDimensions.BucketCount()
		i := 0
		for ; i < dimensionCount; i++ {
			arrayDimensions = append(arrayDimensions, innerArrayDimensions.ChildInBucket(i))
		}
		memberTypeDesc = innerArrayType.MemberTypeDesc
	}
	arrayDimension := st.CreateArrayDimensionNode(openBracketToken, arrayLengthNode,
		closeBracketToken)
	arrayDimensions = append(arrayDimensions, arrayDimension)
	arrayDimensionNodeList := st.CreateNodeList(arrayDimensions...)
	return st.CreateArrayTypeDescriptorNode(memberTypeDesc, arrayDimensionNodeList)
}

func (b *ballerinaParser) parseArrayLength() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.ASTERISK_TOKEN:
		return b.parseBasicLiteral()
	case st.CLOSE_BRACKET_TOKEN:
		return st.CreateEmptyNode()
	case st.IDENTIFIER_TOKEN:
		return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_ARRAY_LENGTH)
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ARRAY_LENGTH)
		return b.parseArrayLength()
	}
}

func (b *ballerinaParser) parseOptionalAnnotations() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATIONS)
	var annotList []st.STNode
	nextToken := b.peek()
	for nextToken.Kind() == st.AT_TOKEN {
		annotList = append(annotList, b.parseAnnotation())
		nextToken = b.peek()
	}
	b.endContext()
	return st.CreateNodeList(annotList...)
}

func (b *ballerinaParser) parseAnnotations() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATIONS)
	var annotList []st.STNode
	annotList = append(annotList, b.parseAnnotation())
	for b.peek().Kind() == st.AT_TOKEN {
		annotList = append(annotList, b.parseAnnotation())
	}
	b.endContext()
	return st.CreateNodeList(annotList...)
}

func (b *ballerinaParser) parseAnnotation() st.STNode {
	atToken := b.parseAtToken()
	var annotReference st.STNode
	if b.isPredeclaredIdentifier(b.peek().Kind()) {
		annotReference = b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE)
	} else {
		annotReference = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		annotReference = st.CreateSimpleNameReferenceNode(annotReference)
	}
	var annotValue st.STNode
	if b.peek().Kind() == st.OPEN_BRACE_TOKEN {
		annotValue = b.parseMappingConstructorExpr()
	} else {
		annotValue = st.CreateEmptyNode()
	}
	return st.CreateAnnotationNode(atToken, annotReference, annotValue)
}

func (b *ballerinaParser) parseAtToken() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.AT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_AT)
		return b.parseAtToken()
	}
}

func (b *ballerinaParser) parseMetaData() st.STNode {
	var docString st.STNode
	var annotations st.STNode
	switch b.peek().Kind() {
	case st.DOCUMENTATION_STRING:
		docString = b.parseMarkdownDocumentation()
		annotations = b.parseOptionalAnnotations()
	case st.AT_TOKEN:
		docString = st.CreateEmptyNode()
		annotations = b.parseOptionalAnnotations()
	default:
		return st.CreateEmptyNode()
	}
	return b.createMetadata(docString, annotations)
}

func (b *ballerinaParser) createMetadata(docString st.STNode, annotations st.STNode) st.STNode {
	if (annotations == nil) && (docString == nil) {
		return st.CreateEmptyNode()
	} else {
		return st.CreateMetadataNode(docString, annotations)
	}
}

func (b *ballerinaParser) parseTypeTestExpression(lhsExpr st.STNode, isInConditionalExpr bool) st.STNode {
	isOrNotIsKeyword := b.parseIsOrNotIsKeyword()
	typeDescriptor := b.parseTypeDescriptorInExpression(isInConditionalExpr)
	return st.CreateTypeTestExpressionNode(lhsExpr, isOrNotIsKeyword, typeDescriptor)
}

func (b *ballerinaParser) parseIsOrNotIsKeyword() st.STNode {
	token := b.peek()
	if (token.Kind() == st.IS_KEYWORD) || (token.Kind() == st.NOT_IS_KEYWORD) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IS_KEYWORD)
		return b.parseIsOrNotIsKeyword()
	}
}

func (b *ballerinaParser) parseLocalTypeDefinitionStatement(annots st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LOCAL_TYPE_DEFINITION_STMT)
	typeKeyword := b.parseTypeKeyword()
	typeName := b.parseTypeName()
	typeDescriptor := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF)
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateLocalTypeDefinitionStatementNode(annots, typeKeyword, typeName, typeDescriptor,
		semicolon)
}

func (b *ballerinaParser) parseExpressionStatement(annots st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
	expression := b.parseActionOrExpressionInLhs(annots)
	return b.getExpressionAsStatement(expression)
}

func (b *ballerinaParser) parseStatementStartWithExpr(annots st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	expr := b.parseActionOrExpressionInLhs(annots)
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *ballerinaParser) parseStatementStartWithExprRhs(expression st.STNode) st.STNode {
	nextTokenKind := b.peek().Kind()
	if b.isAction(expression) || (nextTokenKind == st.SEMICOLON_TOKEN) {
		return b.getExpressionAsStatement(expression)
	}
	switch nextTokenKind {
	case st.EQUAL_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
		return b.parseAssignmentStmtRhs(expression)
	case st.IDENTIFIER_TOKEN:
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

func (b *ballerinaParser) isPossibleExpressionStatement(expression st.STNode) bool {
	switch expression.Kind() {
	case st.METHOD_CALL,
		st.FUNCTION_CALL,
		st.CHECK_EXPRESSION,
		st.REMOTE_METHOD_CALL_ACTION,
		st.CHECK_ACTION,
		st.BRACED_ACTION,
		st.START_ACTION,
		st.TRAP_ACTION,
		st.FLUSH_ACTION,
		st.ASYNC_SEND_ACTION,
		st.SYNC_SEND_ACTION,
		st.RECEIVE_ACTION,
		st.WAIT_ACTION,
		st.QUERY_ACTION,
		st.COMMIT_ACTION:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) getExpressionAsStatement(expression st.STNode) st.STNode {
	switch expression.Kind() {
	case st.METHOD_CALL,
		st.FUNCTION_CALL:
		return b.parseCallStatement(expression)
	case st.CHECK_EXPRESSION:
		return b.parseCheckStatement(expression)
	case st.REMOTE_METHOD_CALL_ACTION,
		st.CHECK_ACTION,
		st.BRACED_ACTION,
		st.START_ACTION,
		st.TRAP_ACTION,
		st.FLUSH_ACTION,
		st.ASYNC_SEND_ACTION,
		st.SYNC_SEND_ACTION,
		st.RECEIVE_ACTION,
		st.WAIT_ACTION,
		st.QUERY_ACTION,
		st.COMMIT_ACTION,
		st.CLIENT_RESOURCE_ACCESS_ACTION:
		return b.parseActionStatement(expression)
	default:
		semicolon := b.parseSemicolon()
		b.endContext()
		expression = b.getExpression(expression)
		exprStmt := st.CreateExpressionStatementNode(st.INVALID_EXPRESSION_STATEMENT,
			expression, semicolon)
		exprStmt = st.AddDiagnostic(exprStmt, &common.ERROR_INVALID_EXPRESSION_STATEMENT)
		return exprStmt
	}
}

func (b *ballerinaParser) parseArrayTypeDescriptorNode(indexedExpr st.STIndexedExpressionNode) st.STNode {
	memberTypeDesc := b.getTypeDescFromExpr(indexedExpr.ContainerExpression)
	lengthExprs, ok := indexedExpr.KeyExpression.(*st.STNodeList)
	if !ok {
		b.internalError("expected st.STNodeList")
		return st.CreateEmptyNode()
	}
	if lengthExprs.IsEmpty() {
		return b.createArrayTypeDesc(memberTypeDesc, indexedExpr.OpenBracket, st.CreateEmptyNode(),
			indexedExpr.CloseBracket)
	}
	lengthExpr := lengthExprs.Get(0)
	switch lengthExpr.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		nameRef, ok := lengthExpr.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("expected st.STSimpleNameReferenceNode")
			return st.CreateEmptyNode()
		}
		if nameRef.Name.IsMissing() {
			return b.createArrayTypeDesc(memberTypeDesc, indexedExpr.OpenBracket, st.CreateEmptyNode(),
				indexedExpr.CloseBracket)
		}
	case st.ASTERISK_LITERAL,
		st.QUALIFIED_NAME_REFERENCE:
		break
	case st.NUMERIC_LITERAL:
		innerChildKind := lengthExpr.ChildInBucket(0).Kind()
		if (innerChildKind == st.DECIMAL_INTEGER_LITERAL_TOKEN) || (innerChildKind == st.HEX_INTEGER_LITERAL_TOKEN) {
			break
		}
	default:
		newOpenBracketWithDiagnostics := st.CloneWithTrailingInvalidNodeMinutiae(
			indexedExpr.OpenBracket, lengthExpr, &common.ERROR_INVALID_ARRAY_LENGTH)
		replacedNode := st.Replace(&indexedExpr, indexedExpr.OpenBracket, newOpenBracketWithDiagnostics)
		newIndexedExpr, ok := replacedNode.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("expected STIndexedExpressionNode")
			return st.CreateEmptyNode()
		}
		indexedExpr = *newIndexedExpr
		lengthExpr = st.CreateEmptyNode()
	}
	return b.createArrayTypeDesc(memberTypeDesc, indexedExpr.OpenBracket, lengthExpr, indexedExpr.CloseBracket)
}

func (b *ballerinaParser) parseCallStatement(expression st.STNode) st.STNode {
	return b.parseCallStatementOrCheckStatement(expression)
}

func (b *ballerinaParser) parseCheckStatement(expression st.STNode) st.STNode {
	return b.parseCallStatementOrCheckStatement(expression)
}

func (b *ballerinaParser) parseCallStatementOrCheckStatement(expression st.STNode) st.STNode {
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateExpressionStatementNode(st.CALL_STATEMENT, expression, semicolon)
}

func (b *ballerinaParser) parseActionStatement(action st.STNode) st.STNode {
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateExpressionStatementNode(st.ACTION_STATEMENT, action, semicolon)
}

func (b *ballerinaParser) parseClientResourceAccessAction(expression st.STNode, rightArrow st.STNode, slashToken st.STNode, isRhsExpr bool, isInMatchGuard bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION)
	resourceAccessPath := b.parseOptionalResourceAccessPath(isRhsExpr, isInMatchGuard)
	resourceAccessMethodDot := b.parseOptionalResourceAccessMethodDot(isRhsExpr, isInMatchGuard)
	resourceAccessMethodName := st.CreateEmptyNode()
	if resourceAccessMethodDot != nil {
		resourceAccessMethodName = st.CreateSimpleNameReferenceNode(b.parseFunctionName())
	}
	resourceMethodCallArgList := b.parseOptionalResourceAccessActionArgList(isRhsExpr, isInMatchGuard)
	b.endContext()
	return st.CreateClientResourceAccessActionNode(expression, rightArrow, slashToken,
		resourceAccessPath, resourceAccessMethodDot, resourceAccessMethodName, resourceMethodCallArgList)
}

func (b *ballerinaParser) parseOptionalResourceAccessPath(isRhsExpr bool, isInMatchGuard bool) st.STNode {
	resourceAccessPath := st.CreateEmptyNodeList()
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN,
		st.OPEN_BRACKET_TOKEN:
		resourceAccessPath = b.parseResourceAccessPath(isRhsExpr, isInMatchGuard)
	case st.DOT_TOKEN,
		st.OPEN_PAREN_TOKEN:
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

func (b *ballerinaParser) parseOptionalResourceAccessMethodDot(isRhsExpr bool, isInMatchGuard bool) st.STNode {
	dotToken := st.CreateEmptyNode()
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.DOT_TOKEN:
		dotToken = b.consume()
	case st.OPEN_PAREN_TOKEN:
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

func (b *ballerinaParser) parseOptionalResourceAccessActionArgList(isRhsExpr bool, isInMatchGuard bool) st.STNode {
	argList := st.CreateEmptyNode()
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
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

func (b *ballerinaParser) parseResourceAccessPath(isRhsExpr bool, isInMatchGuard bool) st.STNode {
	var pathSegmentList []st.STNode
	pathSegment := b.parseResourceAccessSegment()
	pathSegmentList = append(pathSegmentList, pathSegment)
	var leadingSlash st.STNode
	previousPathSegmentNode := pathSegment
	for !b.isEndOfResourceAccessPathSegments(b.peek(), isRhsExpr, isInMatchGuard) {
		leadingSlash = b.parseResourceAccessSegmentRhs(isRhsExpr, isInMatchGuard)
		if leadingSlash == nil {
			break
		}
		pathSegment = b.parseResourceAccessSegment()
		if previousPathSegmentNode.Kind() == st.RESOURCE_ACCESS_REST_SEGMENT {
			b.updateLastNodeInListWithInvalidNode(pathSegmentList, leadingSlash, nil)
			b.updateLastNodeInListWithInvalidNode(pathSegmentList, pathSegment,
				&common.RESOURCE_ACCESS_SEGMENT_IS_NOT_ALLOWED_AFTER_REST_SEGMENT)
		} else {
			pathSegmentList = append(pathSegmentList, leadingSlash)
			pathSegmentList = append(pathSegmentList, pathSegment)
			previousPathSegmentNode = pathSegment
		}
	}
	return st.CreateNodeList(pathSegmentList...)
}

func (b *ballerinaParser) parseResourceAccessSegment() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		return b.consume()
	case st.OPEN_BRACKET_TOKEN:
		return b.parseComputedOrResourceAccessRestSegment(b.consume())
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT)
		return b.parseResourceAccessSegment()
	}
}

func (b *ballerinaParser) parseComputedOrResourceAccessRestSegment(openBracket st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ELLIPSIS_TOKEN:
		ellipsisToken := b.consume()
		expression := b.parseExpression()
		closeBracketToken := b.parseCloseBracket()
		return st.CreateResourceAccessRestSegmentNode(openBracket, ellipsisToken,
			expression, closeBracketToken)
	default:
		if b.isValidExprStart(nextToken.Kind()) {
			expression := b.parseExpression()
			closeBracketToken := b.parseCloseBracket()
			return st.CreateComputedResourceAccessSegmentNode(openBracket, expression,
				closeBracketToken)
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT)
		return b.parseComputedOrResourceAccessRestSegment(openBracket)
	}
}

func (b *ballerinaParser) parseResourceAccessSegmentRhs(isRhsExpr bool, isInMatchGuard bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.SLASH_TOKEN:
		return b.consume()
	default:
		if b.isEndOfResourceAccessPathSegments(nextToken, isRhsExpr, isInMatchGuard) {
			return nil
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS)
		return b.parseResourceAccessSegmentRhs(isRhsExpr, isInMatchGuard)
	}
}

func (b *ballerinaParser) isEndOfResourceAccessPathSegments(nextToken st.STToken, isRhsExpr bool, isInMatchGuard bool) bool {
	switch nextToken.Kind() {
	case st.DOT_TOKEN, st.OPEN_PAREN_TOKEN:
		return true
	default:
		return b.isEndOfActionOrExpression(nextToken, isRhsExpr, isInMatchGuard)
	}
}

func (b *ballerinaParser) parseRemoteMethodCallOrClientResourceAccessOrAsyncSendAction(expression st.STNode, isRhsExpr bool, isInMatchGuard bool) st.STNode {
	rightArrow := b.parseRightArrow()
	return b.parseClientResourceAccessOrAsyncSendActionRhs(expression, rightArrow, isRhsExpr, isInMatchGuard)
}

func (b *ballerinaParser) parseClientResourceAccessOrAsyncSendActionRhs(expression st.STNode, rightArrow st.STNode, isRhsExpr bool, isInMatchGuard bool) st.STNode {
	var name st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.FUNCTION_KEYWORD:
		functionKeyword := b.consume()
		name = st.CreateSimpleNameReferenceNode(functionKeyword)
		return b.parseAsyncSendAction(expression, rightArrow, name)
	case st.CONTINUE_KEYWORD,
		st.COMMIT_KEYWORD:
		name = b.getKeywordAsSimpleNameRef()
	case st.SLASH_TOKEN:
		slashToken := b.consume()
		return b.parseClientResourceAccessAction(expression, rightArrow, slashToken, isRhsExpr, isInMatchGuard)
	default:
		if nextToken.Kind() == st.IDENTIFIER_TOKEN {
			nextNextToken := b.getNextNextToken()
			if ((nextNextToken.Kind() == st.OPEN_PAREN_TOKEN) || b.isEndOfActionOrExpression(nextNextToken, isRhsExpr, isInMatchGuard)) || nextToken.IsMissing() {
				name = st.CreateSimpleNameReferenceNode(b.parseFunctionName())
				break
			}
		}
		token := b.peek()
		solution := b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS)
		if solution.Action == actionKeep {
			name = st.CreateSimpleNameReferenceNode(b.parseFunctionName())
			break
		}
		return b.parseClientResourceAccessOrAsyncSendActionRhs(expression, rightArrow, isRhsExpr, isInMatchGuard)
	}
	return b.parseRemoteCallOrAsyncSendEnd(expression, rightArrow, name)
}

func (b *ballerinaParser) parseRemoteCallOrAsyncSendEnd(expression st.STNode, rightArrow st.STNode, name st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		return b.parseRemoteMethodCallAction(expression, rightArrow, name)
	case st.SEMICOLON_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.COMMA_TOKEN,
		st.FROM_KEYWORD,
		st.JOIN_KEYWORD,
		st.ON_KEYWORD,
		st.LET_KEYWORD,
		st.WHERE_KEYWORD,
		st.ORDER_KEYWORD,
		st.LIMIT_KEYWORD,
		st.SELECT_KEYWORD:
		return b.parseAsyncSendAction(expression, rightArrow, name)
	default:
		if isGroupOrCollectKeyword(nextToken) {
			return b.parseAsyncSendAction(expression, rightArrow, name)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END)
		return b.parseRemoteCallOrAsyncSendEnd(expression, rightArrow, name)
	}
}

func (b *ballerinaParser) parseAsyncSendAction(expression st.STNode, rightArrow st.STNode, peerWorker st.STNode) st.STNode {
	return st.CreateAsyncSendActionNode(expression, rightArrow, peerWorker)
}

func (b *ballerinaParser) parseRemoteMethodCallAction(expression st.STNode, rightArrow st.STNode, name st.STNode) st.STNode {
	openParenToken := b.parseArgListOpenParenthesis()
	arguments := b.parseArgsList()
	closeParenToken := b.parseArgListCloseParenthesis()
	return st.CreateRemoteMethodCallActionNode(expression, rightArrow, name, openParenToken, arguments,
		closeParenToken)
}

func (b *ballerinaParser) parseRightArrow() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.RIGHT_ARROW_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_RIGHT_ARROW)
		return b.parseRightArrow()
	}
}

func (b *ballerinaParser) parseMapTypeDescriptor(mapKeyword st.STNode) st.STNode {
	typeParameter := b.parseTypeParameter()
	return st.CreateMapTypeDescriptorNode(mapKeyword, typeParameter)
}

func (b *ballerinaParser) parseParameterizedTypeDescriptor(keywordToken st.STNode) st.STNode {
	var typeParamNode st.STNode
	nextToken := b.peek()
	if nextToken.Kind() == st.LT_TOKEN {
		typeParamNode = b.parseTypeParameter()
	} else {
		typeParamNode = st.CreateEmptyNode()
	}
	parameterizedTypeDescKind := b.getParameterizedTypeDescKind(keywordToken)
	return st.CreateParameterizedTypeDescriptorNode(parameterizedTypeDescKind, keywordToken,
		typeParamNode)
}

func (b *ballerinaParser) getParameterizedTypeDescKind(keywordToken st.STNode) st.SyntaxKind {
	switch keywordToken.Kind() {
	case st.TYPEDESC_KEYWORD:
		return st.TYPEDESC_TYPE_DESC
	case st.FUTURE_KEYWORD:
		return st.FUTURE_TYPE_DESC
	case st.XML_KEYWORD:
		return st.XML_TYPE_DESC
	default:
		return st.ERROR_TYPE_DESC
	}
}

func (b *ballerinaParser) parseGTToken() st.STToken {
	nextToken := b.peek()
	if nextToken.Kind() == st.GT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_GT)
		return b.parseGTToken()
	}
}

func (b *ballerinaParser) parseLTToken() st.STToken {
	nextToken := b.peek()
	if nextToken.Kind() == st.LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_LT)
		return b.parseLTToken()
	}
}

func (b *ballerinaParser) parseNilLiteral() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NIL_LITERAL)
	openParenthesisToken := b.parseOpenParenthesis()
	closeParenthesisToken := b.parseCloseParenthesis()
	b.endContext()
	return st.CreateNilLiteralNode(openParenthesisToken, closeParenthesisToken)
}

func (b *ballerinaParser) parseAnnotationDeclaration(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOTATION_DECL)
	annotationKeyword := b.parseAnnotationKeyword()
	annotDecl := b.parseAnnotationDeclFromType(metadata, qualifier, constKeyword, annotationKeyword)
	b.endContext()
	return annotDecl
}

func (b *ballerinaParser) parseAnnotationKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ANNOTATION_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD)
		return b.parseAnnotationKeyword()
	}
}

func (b *ballerinaParser) parseAnnotationDeclFromType(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode, annotationKeyword st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
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

func (b *ballerinaParser) parseAnnotationTag() st.STNode {
	token := b.peek()
	if token.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANNOTATION_TAG)
		return b.parseAnnotationTag()
	}
}

func (b *ballerinaParser) parseAnnotationDeclWithOptionalType(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode, annotationKeyword st.STNode) st.STNode {
	typeDescOrAnnotTag := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE)
	if typeDescOrAnnotTag.Kind() == st.QUALIFIED_NAME_REFERENCE {
		annotTag := b.parseAnnotationTag()
		return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword,
			typeDescOrAnnotTag, annotTag)
	}
	nextToken := b.peek()
	if (nextToken.Kind() == st.IDENTIFIER_TOKEN) || b.isValidTypeContinuationToken(nextToken) {
		typeDesc := b.parseComplexTypeDescriptor(typeDescOrAnnotTag,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL, false)
		annotTag := b.parseAnnotationTag()
		return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword, typeDesc,
			annotTag)
	}
	simplenameNode, ok := typeDescOrAnnotTag.(*st.STSimpleNameReferenceNode)
	if !ok {
		b.internalError("parseAnnotationDeclWithOptionalType: expected STSimpleNameReferenceNode")
		return st.CreateEmptyNode()
	}
	annotTag := simplenameNode.Name
	return b.parseAnnotationDeclRhs(metadata, qualifier, constKeyword, annotationKeyword, annotTag)
}

func (b *ballerinaParser) parseAnnotationDeclRhs(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode, annotationKeyword st.STNode, typeDescOrAnnotTag st.STNode) st.STNode {
	nextToken := b.peek()
	var typeDesc st.STNode
	var annotTag st.STNode
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		typeDesc = typeDescOrAnnotTag
		annotTag = b.parseAnnotationTag()
	case st.SEMICOLON_TOKEN,
		st.ON_KEYWORD:
		typeDesc = st.CreateEmptyNode()
		annotTag = typeDescOrAnnotTag
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANNOT_DECL_RHS)
		return b.parseAnnotationDeclRhs(metadata, qualifier, constKeyword, annotationKeyword, typeDescOrAnnotTag)
	}
	return b.parseAnnotationDeclAttachPoints(metadata, qualifier, constKeyword, annotationKeyword, typeDesc,
		annotTag)
}

func (b *ballerinaParser) parseAnnotationDeclAttachPoints(metadata st.STNode, qualifier st.STNode, constKeyword st.STNode, annotationKeyword st.STNode, typeDesc st.STNode, annotTag st.STNode) st.STNode {
	var onKeyword st.STNode
	var attachPoints st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.SEMICOLON_TOKEN:
		onKeyword = st.CreateEmptyNode()
		attachPoints = st.CreateEmptyNodeList()
	case st.ON_KEYWORD:
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
	return st.CreateAnnotationDeclarationNode(metadata, qualifier, constKeyword, annotationKeyword,
		typeDesc, annotTag, onKeyword, attachPoints, semicolonToken)
}

func (b *ballerinaParser) parseAnnotationAttachPoints() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST)
	var attachPoints []st.STNode
	nextToken := b.peek()
	if b.isEndAnnotAttachPointList(nextToken.Kind()) {
		b.endContext()
		return st.CreateEmptyNodeList()
	}
	attachPoint := b.parseAnnotationAttachPoint()
	attachPoints = append(attachPoints, attachPoint)
	nextToken = b.peek()
	var leadingComma st.STNode
	for !b.isEndAnnotAttachPointList(nextToken.Kind()) {
		leadingComma = b.parseAttachPointEnd()
		if leadingComma == nil {
			break
		}
		attachPoints = append(attachPoints, leadingComma)
		attachPoint = b.parseAnnotationAttachPoint()
		if attachPoint == nil {
			missingAttachPointIdent := st.CreateMissingToken(st.TYPE_KEYWORD, nil)
			identList := st.CreateNodeList(missingAttachPointIdent)
			attachPoint = st.CreateAnnotationAttachPointNode(st.CreateEmptyNode(), identList)
			attachPoint = st.AddDiagnostic(attachPoint,
				&common.ERROR_MISSING_ANNOTATION_ATTACH_POINT)
			attachPoints = append(attachPoints, attachPoint)
			break
		}
		attachPoints = append(attachPoints, attachPoint)
		nextToken = b.peek()
	}
	if (st.LastToken(attachPoint).IsMissing() && (b.tokenReader.Peek().Kind() == st.IDENTIFIER_TOKEN)) && (!b.tokenReader.Head().HasTrailingNewLine()) {
		nextNonVirtualToken := b.tokenReader.Read()
		b.updateLastNodeInListWithInvalidNode(attachPoints, nextNonVirtualToken,
			&common.ERROR_INVALID_TOKEN, nextNonVirtualToken.Text())
	}
	b.endContext()
	return st.CreateNodeList(attachPoints...)
}

func (b *ballerinaParser) parseAttachPointEnd() st.STNode {
	switch b.peek().Kind() {
	case st.SEMICOLON_TOKEN:
		return nil
	case st.COMMA_TOKEN:
		return b.consume()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ATTACH_POINT_END)
		return b.parseAttachPointEnd()
	}
}

func (b *ballerinaParser) isEndAnnotAttachPointList(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.EOF_TOKEN, st.SEMICOLON_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseAnnotationAttachPoint() st.STNode {
	switch b.peek().Kind() {
	case st.EOF_TOKEN:
		return nil
	case st.ANNOTATION_KEYWORD,
		st.EXTERNAL_KEYWORD,
		st.VAR_KEYWORD,
		st.CONST_KEYWORD,
		st.LISTENER_KEYWORD,
		st.WORKER_KEYWORD,
		st.SOURCE_KEYWORD:
		sourceKeyword := b.parseSourceKeyword()
		return b.parseAttachPointIdent(sourceKeyword)
	case st.OBJECT_KEYWORD,
		st.TYPE_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.PARAMETER_KEYWORD,
		st.RETURN_KEYWORD,
		st.SERVICE_KEYWORD,
		st.FIELD_KEYWORD,
		st.RECORD_KEYWORD,
		st.CLASS_KEYWORD:
		sourceKeyword := st.CreateEmptyNode()
		firstIdent := b.consume()
		return b.parseDualAttachPointIdent(sourceKeyword, firstIdent)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ATTACH_POINT)
		return b.parseAnnotationAttachPoint()
	}
}

func (b *ballerinaParser) parseSourceKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.SOURCE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD)
		return b.parseSourceKeyword()
	}
}

func (b *ballerinaParser) parseAttachPointIdent(sourceKeyword st.STNode) st.STNode {
	switch b.peek().Kind() {
	case st.ANNOTATION_KEYWORD,
		st.EXTERNAL_KEYWORD,
		st.VAR_KEYWORD,
		st.CONST_KEYWORD,
		st.LISTENER_KEYWORD,
		st.WORKER_KEYWORD:
		firstIdent := b.consume()
		identList := st.CreateNodeList(firstIdent)
		return st.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	case st.OBJECT_KEYWORD,
		st.RESOURCE_KEYWORD,
		st.RECORD_KEYWORD,
		st.TYPE_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.PARAMETER_KEYWORD,
		st.RETURN_KEYWORD,
		st.SERVICE_KEYWORD,
		st.FIELD_KEYWORD,
		st.CLASS_KEYWORD:
		firstIdent := b.consume()
		return b.parseDualAttachPointIdent(sourceKeyword, firstIdent)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT)
		return b.parseAttachPointIdent(sourceKeyword)
	}
}

func (b *ballerinaParser) parseDualAttachPointIdent(sourceKeyword st.STNode, firstIdent st.STNode) st.STNode {
	var secondIdent st.STNode
	switch firstIdent.Kind() {
	case st.OBJECT_KEYWORD:
		secondIdent = b.parseIdentAfterObjectIdent()
	case st.RESOURCE_KEYWORD:
		secondIdent = b.parseFunctionIdent()
	case st.RECORD_KEYWORD:
		secondIdent = b.parseFieldIdent()
	case st.SERVICE_KEYWORD:
		return b.parseServiceAttachPoint(sourceKeyword, firstIdent)
	case st.TYPE_KEYWORD, st.FUNCTION_KEYWORD, st.PARAMETER_KEYWORD,
		st.RETURN_KEYWORD, st.FIELD_KEYWORD, st.CLASS_KEYWORD:
		fallthrough
	default:
		identList := st.CreateNodeList(firstIdent)
		return st.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	}
	identList := st.CreateNodeList(firstIdent, secondIdent)
	return st.CreateAnnotationAttachPointNode(sourceKeyword, identList)
}

func (b *ballerinaParser) parseRemoteIdent() st.STNode {
	token := b.peek()
	if token.Kind() == st.REMOTE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_REMOTE_IDENT)
		return b.parseRemoteIdent()
	}
}

func (b *ballerinaParser) parseServiceAttachPoint(sourceKeyword st.STNode, firstIdent st.STNode) st.STNode {
	var identList st.STNode
	token := b.peek()
	switch token.Kind() {
	case st.REMOTE_KEYWORD:
		secondIdent := b.parseRemoteIdent()
		thirdIdent := b.parseFunctionIdent()
		identList = st.CreateNodeList(firstIdent, secondIdent, thirdIdent)
		return st.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	case st.COMMA_TOKEN,
		st.SEMICOLON_TOKEN:
		identList = st.CreateNodeList(firstIdent)
		return st.CreateAnnotationAttachPointNode(sourceKeyword, identList)
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS)
		return b.parseServiceAttachPoint(sourceKeyword, firstIdent)
	}
}

func (b *ballerinaParser) parseIdentAfterObjectIdent() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.FUNCTION_KEYWORD, st.FIELD_KEYWORD:
		return b.consume()
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT)
		return b.parseIdentAfterObjectIdent()
	}
}

func (b *ballerinaParser) parseFunctionIdent() st.STNode {
	token := b.peek()
	if token.Kind() == st.FUNCTION_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FUNCTION_IDENT)
		return b.parseFunctionIdent()
	}
}

func (b *ballerinaParser) parseFieldIdent() st.STNode {
	token := b.peek()
	if token.Kind() == st.FIELD_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FIELD_IDENT)
		return b.parseFieldIdent()
	}
}

func (b *ballerinaParser) parseXMLNamespaceDeclaration(isModuleVar bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION)
	xmlnsKeyword := b.parseXMLNSKeyword()
	namespaceUri := b.parseSimpleConstExpr()
	for !b.isValidXMLNameSpaceURI(namespaceUri) {
		xmlnsKeyword = st.CloneWithTrailingInvalidNodeMinutiae(xmlnsKeyword, namespaceUri,
			&common.ERROR_INVALID_XML_NAMESPACE_URI)
		namespaceUri = b.parseSimpleConstExpr()
	}
	xmlnsDecl := b.parseXMLDeclRhs(xmlnsKeyword, namespaceUri, isModuleVar)
	b.endContext()
	return xmlnsDecl
}

func (b *ballerinaParser) parseXMLNSKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.XMLNS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD)
		return b.parseXMLNSKeyword()
	}
}

func (b *ballerinaParser) isValidXMLNameSpaceURI(expr st.STNode) bool {
	switch expr.Kind() {
	case st.STRING_LITERAL, st.QUALIFIED_NAME_REFERENCE, st.SIMPLE_NAME_REFERENCE:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseSimpleConstExpr() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION)
	expr := b.parseSimpleConstExprInternal()
	b.endContext()
	return expr
}

func (b *ballerinaParser) parseSimpleConstExprInternal() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.STRING_LITERAL_TOKEN,
		st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.NULL_KEYWORD:
		return b.parseBasicLiteral()
	case st.PLUS_TOKEN, st.MINUS_TOKEN:
		return b.parseSignedIntOrFloat()
	case st.OPEN_PAREN_TOKEN:
		return b.parseNilLiteral()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			return b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		}
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START)
		return b.parseSimpleConstExprInternal()
	}
}

func (b *ballerinaParser) parseXMLDeclRhs(xmlnsKeyword st.STNode, namespaceUri st.STNode, isModuleVar bool) st.STNode {
	asKeyword := st.CreateEmptyNode()
	namespacePrefix := st.CreateEmptyNode()
	switch b.peek().Kind() {
	case st.AS_KEYWORD:
		asKeyword = b.parseAsKeyword()
		namespacePrefix = b.parseNamespacePrefix()
	case st.SEMICOLON_TOKEN:
		break
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL)
		return b.parseXMLDeclRhs(xmlnsKeyword, namespaceUri, isModuleVar)
	}
	semicolon := b.parseSemicolon()
	if isModuleVar {
		return st.CreateModuleXMLNamespaceDeclarationNode(xmlnsKeyword, namespaceUri, asKeyword,
			namespacePrefix, semicolon)
	}
	return st.CreateXMLNamespaceDeclarationNode(xmlnsKeyword, namespaceUri, asKeyword, namespacePrefix,
		semicolon)
}

func (b *ballerinaParser) parseNamespacePrefix() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_NAMESPACE_PREFIX)
		return b.parseNamespacePrefix()
	}
}

func (b *ballerinaParser) parseNamedWorkerDeclaration(annots st.STNode, qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL)
	transactionalKeyword := b.getTransactionalKeyword(qualifiers)
	workerKeyword := b.parseWorkerKeyword()
	workerName := b.parseWorkerName()
	returnTypeDesc := b.parseReturnTypeDescriptor()
	workerBody := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateNamedWorkerDeclarationNode(annots, transactionalKeyword, workerKeyword, workerName,
		returnTypeDesc, workerBody, onFailClause)
}

func (b *ballerinaParser) getTransactionalKeyword(qualifierList []st.STNode) st.STNode {
	var validatedList []st.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			qualifierToken, ok := qualifier.(st.STToken)
			if !ok {
				b.internalError("expected STToken")
				return st.CreateEmptyNode()
			}
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, qualifierToken.Text())
		} else if qualifier.Kind() == st.TRANSACTIONAL_KEYWORD {
			validatedList = append(validatedList, qualifier)
		} else if len(qualifierList) == nextIndex {
			b.addInvalidNodeToNextToken(qualifier, &common.ERROR_QUALIFIER_NOT_ALLOWED,
				st.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		}
	}
	var transactionalKeyword st.STNode
	if len(validatedList) == 0 {
		transactionalKeyword = st.CreateEmptyNode()
	} else {
		transactionalKeyword = validatedList[0]
	}
	return transactionalKeyword
}

func (b *ballerinaParser) parseReturnTypeDescriptor() st.STNode {
	token := b.peek()
	if token.Kind() != st.RETURNS_KEYWORD {
		return st.CreateEmptyNode()
	}
	returnsKeyword := b.consume()
	annot := b.parseOptionalAnnotations()
	ty := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC)
	return st.CreateReturnTypeDescriptorNode(returnsKeyword, annot, ty)
}

func (b *ballerinaParser) parseWorkerKeyword() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.WORKER_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WORKER_KEYWORD)
		return b.parseWorkerKeyword()
	}
}

func (b *ballerinaParser) parseWorkerName() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.IDENTIFIER_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WORKER_NAME)
		return b.parseWorkerName()
	}
}

func (b *ballerinaParser) parseLockStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LOCK_STMT)
	lockKeyword := b.parseLockKeyword()
	blockStatement := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateLockStatementNode(lockKeyword, blockStatement, onFailClause)
}

func (b *ballerinaParser) parseLockKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.LOCK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LOCK_KEYWORD)
		return b.parseLockKeyword()
	}
}

func (b *ballerinaParser) parseUnionTypeDescriptor(leftTypeDesc st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool) st.STNode {
	pipeToken := b.consume()
	rightTypeDesc := b.parseTypeDescriptorInternalWithPrecedence(nil, context, isTypedBindingPattern, false,
		typePrecedenceUnion)
	return b.mergeTypesWithUnion(leftTypeDesc, pipeToken, rightTypeDesc)
}

func (b *ballerinaParser) createUnionTypeDesc(leftTypeDesc st.STNode, pipeToken st.STNode, rightTypeDesc st.STNode) st.STNode {
	leftTypeDesc = b.validateForUsageOfVar(leftTypeDesc)
	rightTypeDesc = b.validateForUsageOfVar(rightTypeDesc)
	return st.CreateUnionTypeDescriptorNode(leftTypeDesc, pipeToken, rightTypeDesc)
}

func (b *ballerinaParser) parsePipeToken() st.STNode {
	token := b.peek()
	if token.Kind() == st.PIPE_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PIPE)
		return b.parsePipeToken()
	}
}

func (b *ballerinaParser) isTypeStartingToken(nodeKind st.SyntaxKind) bool {
	return isTypeStartingToken(nodeKind, b.getNextNextToken())
}

func (b *ballerinaParser) isSimpleTypeInExpression(nodeKind st.SyntaxKind) bool {
	switch nodeKind {
	case st.VAR_KEYWORD, st.READONLY_KEYWORD:
		return false
	default:
		return isSimpleType(nodeKind)
	}
}

func (b *ballerinaParser) isQualifiedIdentifierPredeclaredPrefix(nodeKind st.SyntaxKind) bool {
	return (isPredeclaredPrefix(nodeKind) && (b.getNextNextToken().Kind() == st.COLON_TOKEN))
}

func (b *ballerinaParser) parseForkKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.FORK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FORK_KEYWORD)
		return b.parseForkKeyword()
	}
}

func (b *ballerinaParser) parseForkStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FORK_STMT)
	forkKeyword := b.parseForkKeyword()
	openBrace := b.parseOpenBrace()
	var workers []st.STNode
	for !b.isEndOfStatements() {
		stmt := b.parseStatement()
		if stmt == nil {
			break
		}
		if b.validateStatement(stmt) {
			continue
		}
		switch stmt.Kind() {
		case st.NAMED_WORKER_DECLARATION:
			workers = append(workers, stmt)
		default:
			if len(workers) == 0 {
				openBrace = st.CloneWithTrailingInvalidNodeMinutiae(openBrace, stmt,
					&common.ERROR_ONLY_NAMED_WORKERS_ALLOWED_HERE)
			} else {
				b.updateLastNodeInListWithInvalidNode(workers, stmt,
					&common.ERROR_ONLY_NAMED_WORKERS_ALLOWED_HERE)
			}
		}
	}
	namedWorkerDeclarations := st.CreateNodeList(workers...)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	forkStmt := st.CreateForkStatementNode(forkKeyword, openBrace, namedWorkerDeclarations, closeBrace)
	if b.isNodeListEmpty(namedWorkerDeclarations) {
		return st.AddDiagnostic(forkStmt,
			&common.ERROR_MISSING_NAMED_WORKER_DECLARATION_IN_FORK_STMT)
	}
	return forkStmt
}

func (b *ballerinaParser) parseTrapExpression(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	trapKeyword := b.parseTrapKeyword()
	expr := b.parseExpressionWithConditional(operatorPrecedenceTrap, isRhsExpr, allowActions, isInConditionalExpr)
	if b.isAction(expr) {
		return st.CreateTrapExpressionNode(st.TRAP_ACTION, trapKeyword, expr)
	}
	return st.CreateTrapExpressionNode(st.TRAP_EXPRESSION, trapKeyword, expr)
}

func (b *ballerinaParser) parseTrapKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.TRAP_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TRAP_KEYWORD)
		return b.parseTrapKeyword()
	}
}

func (b *ballerinaParser) parseListConstructorExpr() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR)
	openBracket := b.parseOpenBracket()
	listMembers := b.parseListMembers()
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return st.CreateListConstructorExpressionNode(openBracket, listMembers, closeBracket)
}

func (b *ballerinaParser) parseListMembers() st.STNode {
	var listMembers []st.STNode
	if b.isEndOfListConstructor(b.peek().Kind()) {
		return st.CreateEmptyNodeList()
	}
	listMember := b.parseListMember()
	listMembers = append(listMembers, listMember)
	return b.parseListMembersInner(listMembers)
}

func (b *ballerinaParser) parseListMembersInner(listMembers []st.STNode) st.STNode {
	var listConstructorMemberEnd st.STNode
	for !b.isEndOfListConstructor(b.peek().Kind()) {
		listConstructorMemberEnd = b.parseListConstructorMemberEnd()
		if listConstructorMemberEnd == nil {
			break
		}
		listMembers = append(listMembers, listConstructorMemberEnd)
		listMember := b.parseListMember()
		listMembers = append(listMembers, listMember)
	}
	return st.CreateNodeList(listMembers...)
}

func (b *ballerinaParser) parseListMember() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.ELLIPSIS_TOKEN {
		return b.parseSpreadMember()
	} else {
		return b.parseExpression()
	}
}

func (b *ballerinaParser) parseSpreadMember() st.STNode {
	ellipsis := b.parseEllipsis()
	expr := b.parseExpression()
	return st.CreateSpreadMemberNode(ellipsis, expr)
}

func (b *ballerinaParser) isEndOfListConstructor(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACKET_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseListConstructorMemberEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return b.consume()
	case st.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END)
		return b.parseListConstructorMemberEnd()
	}
}

func (b *ballerinaParser) parseForEachStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FOREACH_STMT)
	forEachKeyword := b.parseForEachKeyword()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_FOREACH_STMT)
	inKeyword := b.parseInKeyword()
	actionOrExpr := b.parseActionOrExpression()
	blockStatement := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateForEachStatementNode(forEachKeyword, typedBindingPattern, inKeyword, actionOrExpr,
		blockStatement, onFailClause)
}

func (b *ballerinaParser) parseForEachKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.FOREACH_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD)
		return b.parseForEachKeyword()
	}
}

func (b *ballerinaParser) parseInKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.IN_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_IN_KEYWORD)
		return b.parseInKeyword()
	}
}

func (b *ballerinaParser) parseTypeCastExpr(isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TYPE_CAST)
	ltToken := b.parseLTToken()
	return b.parseTypeCastExprInner(ltToken, isRhsExpr, allowActions, isInConditionalExpr)
}

func (b *ballerinaParser) parseTypeCastExprInner(ltToken st.STNode, isRhsExpr bool, allowActions bool, isInConditionalExpr bool) st.STNode {
	typeCastParam := b.parseTypeCastParam()
	gtToken := b.parseGTToken()
	b.endContext()
	expression := b.parseExpressionWithConditional(operatorPrecedenceExpressionAction, isRhsExpr, allowActions, isInConditionalExpr)
	return st.CreateTypeCastExpressionNode(ltToken, typeCastParam, gtToken, expression)
}

func (b *ballerinaParser) parseTypeCastParam() st.STNode {
	var annot st.STNode
	var ty st.STNode
	token := b.peek()
	switch token.Kind() {
	case st.AT_TOKEN:
		annot = b.parseOptionalAnnotations()
		token = b.peek()
		if b.isTypeStartingToken(token.Kind()) {
			ty = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS)
		} else {
			ty = st.CreateEmptyNode()
		}
	default:
		annot = st.CreateEmptyNode()
		ty = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS)
	}
	return st.CreateTypeCastParamNode(b.getAnnotations(annot), ty)
}

func (b *ballerinaParser) parseTableConstructorExprRhs(tableKeyword st.STNode, keySpecifier st.STNode) st.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR)
	openBracket := b.parseOpenBracket()
	rowList := b.parseRowList()
	closeBracket := b.parseCloseBracket()
	return st.CreateTableConstructorExpressionNode(tableKeyword, keySpecifier, openBracket, rowList,
		closeBracket)
}

func (b *ballerinaParser) parseTableKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.TABLE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TABLE_KEYWORD)
		return b.parseTableKeyword()
	}
}

func (b *ballerinaParser) parseRowList() st.STNode {
	nextToken := b.peek()
	if b.isEndOfTableRowList(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	var mappings []st.STNode
	mapExpr := b.parseMappingConstructorExpr()
	mappings = append(mappings, mapExpr)
	nextToken = b.peek()
	var rowEnd st.STNode
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
	return st.CreateNodeList(mappings...)
}

func (b *ballerinaParser) isEndOfTableRowList(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACKET_TOKEN:
		return true
	case st.COMMA_TOKEN, st.OPEN_BRACE_TOKEN:
		return false
	default:
		return b.isEndOfMappingConstructor(tokenKind)
	}
}

func (b *ballerinaParser) parseTableRowEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACKET_TOKEN, st.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_ROW_END)
		return b.parseTableRowEnd()
	}
}

func (b *ballerinaParser) parseKeySpecifier() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_KEY_SPECIFIER)
	keyKeyword := b.parseKeyKeyword()
	openParen := b.parseOpenParenthesis()
	fieldNames := b.parseFieldNames()
	closeParen := b.parseCloseParenthesis()
	b.endContext()
	return st.CreateKeySpecifierNode(keyKeyword, openParen, fieldNames, closeParen)
}

func (b *ballerinaParser) parseKeyKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.KEY_KEYWORD {
		return b.consume()
	}
	if isKeyKeyword(token) {
		return b.getKeyKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_KEY_KEYWORD)
	return b.parseKeyKeyword()
}

func (b *ballerinaParser) getKeyKeyword(token st.STToken) st.STNode {
	return st.CreateTokenWithDiagnostics(st.KEY_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *ballerinaParser) getUnderscoreKeyword(token st.STToken) st.STToken {
	return st.CreateTokenWithDiagnostics(st.UNDERSCORE_KEYWORD, token.LeadingMinutiae(),
		token.TrailingMinutiae(), token.Diagnostics())
}

func (b *ballerinaParser) parseNaturalKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.NATURAL_KEYWORD {
		return b.consume()
	}
	if b.isNaturalKeyword(token) {
		return b.getNaturalKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD)
	return b.parseNaturalKeyword()
}

func (b *ballerinaParser) isNaturalKeyword(node st.STNode) bool {
	token, isToken := node.(st.STToken)
	if isToken {
		return isNaturalKeyword(token)
	}
	if node.Kind() != st.SIMPLE_NAME_REFERENCE {
		return false
	}
	simpleNameNode, ok := node.(*st.STSimpleNameReferenceNode)
	if !ok {
		b.internalError("isNaturalKeyword: expected STSimpleNameReferenceNode")
		return false
	}
	nameToken, ok := simpleNameNode.Name.(st.STToken)
	if !ok {
		b.internalError("isNaturalKeyword: expected STToken")
		return false
	}
	return isNaturalKeyword(nameToken)
}

func (b *ballerinaParser) getNaturalKeyword(token st.STToken) st.STNode {
	return st.CreateTokenWithDiagnostics(st.NATURAL_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *ballerinaParser) parseFieldNames() st.STNode {
	nextToken := b.peek()
	if b.isEndOfFieldNamesList(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	var fieldNames []st.STNode
	fieldName := b.parseVariableName()
	fieldNames = append(fieldNames, fieldName)
	nextToken = b.peek()
	var leadingComma st.STNode
	for !b.isEndOfFieldNamesList(nextToken.Kind()) {
		leadingComma = b.parseComma()
		fieldNames = append(fieldNames, leadingComma)
		fieldName = b.parseVariableName()
		fieldNames = append(fieldNames, fieldName)
		nextToken = b.peek()
	}
	return st.CreateNodeList(fieldNames...)
}

func (b *ballerinaParser) isEndOfFieldNamesList(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.COMMA_TOKEN, st.IDENTIFIER_TOKEN:
		return false
	default:
		return true
	}
}

func (b *ballerinaParser) parseErrorKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ERROR_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ERROR_KEYWORD)
		return b.parseErrorKeyword()
	}
}

func (b *ballerinaParser) parseStreamTypeDescriptor(streamKeywordToken st.STNode) st.STNode {
	var streamTypeParamsNode st.STNode
	nextToken := b.peek()
	if nextToken.Kind() == st.LT_TOKEN {
		streamTypeParamsNode = b.parseStreamTypeParamsNode()
	} else {
		streamTypeParamsNode = st.CreateEmptyNode()
	}
	return st.CreateStreamTypeDescriptorNode(streamKeywordToken, streamTypeParamsNode)
}

func (b *ballerinaParser) parseStreamTypeParamsNode() st.STNode {
	ltToken := b.parseLTToken()
	b.startContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC)
	leftTypeDescNode := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC)
	streamTypedesc := b.parseStreamTypeParamsNodeInner(ltToken, leftTypeDescNode)
	b.endContext()
	return streamTypedesc
}

func (b *ballerinaParser) parseStreamTypeParamsNodeInner(ltToken st.STNode, leftTypeDescNode st.STNode) st.STNode {
	var commaToken st.STNode
	var rightTypeDescNode st.STNode
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		commaToken = b.parseComma()
		rightTypeDescNode = b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC)
	case st.GT_TOKEN:
		commaToken = st.CreateEmptyNode()
		rightTypeDescNode = st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS)
		return b.parseStreamTypeParamsNodeInner(ltToken, leftTypeDescNode)
	}
	gtToken := b.parseGTToken()
	return st.CreateStreamTypeParamsNode(ltToken, leftTypeDescNode, commaToken, rightTypeDescNode,
		gtToken)
}

func (b *ballerinaParser) parseLetExpression(isRhsExpr bool, isInConditionalExpr bool) st.STNode {
	letKeyword := b.parseLetKeyword()
	letVarDeclarations := b.parseLetVarDeclarations(common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL, isRhsExpr, false)
	inKeyword := b.parseInKeyword()
	letKeyword = b.cloneWithDiagnosticIfListEmpty(letVarDeclarations, letKeyword,
		&common.ERROR_MISSING_LET_VARIABLE_DECLARATION)
	expression := b.parseExpressionWithConditional(operatorPrecedenceRemoteCallAction, isRhsExpr, false,
		isInConditionalExpr)
	return st.CreateLetExpressionNode(letKeyword, letVarDeclarations, inKeyword, expression)
}

func (b *ballerinaParser) parseLetKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.LET_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LET_KEYWORD)
		return b.parseLetKeyword()
	}
}

func (b *ballerinaParser) parseLetVarDeclarations(context common.ParserRuleContext, isRhsExpr bool, allowActions bool) st.STNode {
	b.startContext(context)
	var varDecls []st.STNode
	nextToken := b.peek()
	if isEndOfLetVarDeclarations(nextToken, b.getNextNextToken()) {
		b.endContext()
		return st.CreateEmptyNodeList()
	}
	varDec := b.parseLetVarDecl(context, isRhsExpr, allowActions)
	varDecls = append(varDecls, varDec)
	nextToken = b.peek()
	var leadingComma st.STNode
	for !isEndOfLetVarDeclarations(nextToken, b.getNextNextToken()) {
		leadingComma = b.parseComma()
		varDecls = append(varDecls, leadingComma)
		varDec = b.parseLetVarDecl(context, isRhsExpr, allowActions)
		varDecls = append(varDecls, varDec)
		nextToken = b.peek()
	}
	b.endContext()
	return st.CreateNodeList(varDecls...)
}

func (b *ballerinaParser) parseLetVarDecl(context common.ParserRuleContext, isRhsExpr bool, allowActions bool) st.STNode {
	annot := b.parseOptionalAnnotations()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL)
	assign := b.parseAssignOp()
	var expression st.STNode
	if context == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
		expression = b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, allowActions)
	} else {
		expression = b.parseExpressionWithPrecedence(operatorPrecedenceAnonFuncOrLet, isRhsExpr, false)
	}
	return st.CreateLetVariableDeclarationNode(annot, typedBindingPattern, assign, expression)
}

func (b *ballerinaParser) parseTemplateExpression() st.STNode {
	ty := st.CreateEmptyNode()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	content := b.parseTemplateContent()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	return st.CreateTemplateExpressionNode(st.RAW_TEMPLATE_EXPRESSION, ty, startingBackTick,
		content, endingBackTick)
}

func (b *ballerinaParser) parseTemplateContent() st.STNode {
	var items []st.STNode
	nextToken := b.peek()
	for !b.isEndOfBacktickContent(nextToken.Kind()) {
		contentItem := b.parseTemplateItem()
		items = append(items, contentItem)
		nextToken = b.peek()
	}
	return st.CreateNodeList(items...)
}

func (b *ballerinaParser) isEndOfBacktickContent(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN, st.BACKTICK_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseTemplateItem() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.INTERPOLATION_START_TOKEN {
		return b.parseInterpolation()
	}
	if nextToken.Kind() != st.TEMPLATE_STRING {
		nextToken = b.consume()
		return st.CreateLiteralValueTokenWithDiagnostics(st.TEMPLATE_STRING,
			nextToken.Text(), nextToken.LeadingMinutiae(), nextToken.TrailingMinutiae(),
			nextToken.Diagnostics())
	}
	return b.consume()
}

func (b *ballerinaParser) parseStringTemplateExpression() st.STNode {
	ty := b.parseStringKeyword()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	content := b.parseTemplateContent()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return st.CreateTemplateExpressionNode(st.STRING_TEMPLATE_EXPRESSION, ty, startingBackTick,
		content, endingBackTick)
}

func (b *ballerinaParser) parseStringKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.STRING_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_STRING_KEYWORD)
		return b.parseStringKeyword()
	}
}

func (b *ballerinaParser) parseXMLTemplateExpression() st.STNode {
	xmlKeyword := b.parseXMLKeyword()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	if startingBackTick.IsMissing() {
		return b.createMissingTemplateExpressionNode(xmlKeyword, st.XML_TEMPLATE_EXPRESSION)
	}
	content := b.parseTemplateContentAsXML()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return st.CreateTemplateExpressionNode(st.XML_TEMPLATE_EXPRESSION, xmlKeyword,
		startingBackTick, content, endingBackTick)
}

func (b *ballerinaParser) parseXMLKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.XML_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XML_KEYWORD)
		return b.parseXMLKeyword()
	}
}

func (b *ballerinaParser) parseTemplateContentAsXML() st.STNode {
	var expressions []st.STNode
	var xmlStringBuilder strings.Builder
	nextToken := b.peek()
	for !b.isEndOfBacktickContent(nextToken.Kind()) {
		contentItem := b.parseTemplateItem()
		if contentItem.Kind() == st.TEMPLATE_STRING {
			contentToken, ok := contentItem.(st.STToken)
			if !ok {
				b.internalError("parseTemplateContentAsXML: expected STToken")
				return st.CreateEmptyNode()
			}
			xmlStringBuilder.WriteString(contentToken.Text())
		} else {
			xmlStringBuilder.WriteString("${}")
			expressions = append(expressions, contentItem) //nolint:staticcheck // TODO
		}
		nextToken = b.peek()
	}
	charReader := text.CharReaderFromText(xmlStringBuilder.String())
	xl := newXMLLexer(b.tokenReader.ctx, b.tokenReader.fileName, charReader)
	tr := createTokenReader(b.tokenReader.ctx, b.tokenReader.fileName, xl)
	xp := newXMLParser(tr, expressions)
	return xp.Parse()
}

func (b *ballerinaParser) parseRegExpTemplateExpression() st.STNode {
	reKeyword := b.consume()
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	if startingBackTick.IsMissing() {
		return b.createMissingTemplateExpressionNode(reKeyword, st.REGEX_TEMPLATE_EXPRESSION)
	}
	content := b.parseTemplateContentAsRegExp()
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return st.CreateTemplateExpressionNode(st.REGEX_TEMPLATE_EXPRESSION, reKeyword,
		startingBackTick, content, endingBackTick)
}

func (b *ballerinaParser) createMissingTemplateExpressionNode(reKeyword st.STNode, kind st.SyntaxKind) st.STNode {
	startingBackTick := st.CreateMissingToken(st.BACKTICK_TOKEN, nil)
	endingBackTick := st.CreateMissingToken(st.BACKTICK_TOKEN, nil)
	content := st.CreateEmptyNodeList()
	templateExpr := st.CreateTemplateExpressionNode(kind, reKeyword, startingBackTick, content, endingBackTick)
	templateExpr = st.AddDiagnostic(templateExpr, &common.ERROR_MISSING_BACKTICK_STRING)
	return templateExpr
}

func (b *ballerinaParser) parseTemplateContentAsRegExp() st.STNode {
	b.tokenReader.StartMode(parserModeRegexp)
	b.unimplemented("Regexp parser not implemented")
	return st.CreateEmptyNode()
}

func (b *ballerinaParser) parseInterpolation() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_INTERPOLATION)
	interpolStart := b.parseInterpolationStart()
	expr := b.parseExpression()
	for !b.isEndOfInterpolation() {
		nextToken := b.consume()
		expr = st.CloneWithTrailingInvalidNodeMinutiae(expr, nextToken,
			&common.ERROR_INVALID_TOKEN, nextToken.Text())
	}
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return st.CreateInterpolationNode(interpolStart, expr, closeBrace)
}

func (b *ballerinaParser) isEndOfInterpolation() bool {
	nextTokenKind := b.peek().Kind()
	switch nextTokenKind {
	case st.EOF_TOKEN, st.BACKTICK_TOKEN:
		return true
	default:
		currentLexerMode := b.tokenReader.GetCurrentMode()
		return (((nextTokenKind == st.CLOSE_BRACE_TOKEN) && (currentLexerMode != parserModeInterpolation)) && (currentLexerMode != parserModeInterpolationBracedContent))
	}
}

func (b *ballerinaParser) parseInterpolationStart() st.STNode {
	token := b.peek()
	if token.Kind() == st.INTERPOLATION_START_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN)
		return b.parseInterpolationStart()
	}
}

func (b *ballerinaParser) parseBacktickToken(ctx common.ParserRuleContext) st.STNode {
	token := b.peek()
	if token.Kind() == st.BACKTICK_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, ctx)
		return b.parseBacktickToken(ctx)
	}
}

func (b *ballerinaParser) parseTableTypeDescriptor(tableKeywordToken st.STNode) st.STNode {
	rowTypeParameterNode := b.parseRowTypeParameter()
	var keyConstraintNode st.STNode
	nextToken := b.peek()
	if isKeyKeyword(nextToken) {
		keyKeywordToken := b.getKeyKeyword(b.consume())
		keyConstraintNode = b.parseKeyConstraint(keyKeywordToken)
	} else {
		keyConstraintNode = st.CreateEmptyNode()
	}
	return st.CreateTableTypeDescriptorNode(tableKeywordToken, rowTypeParameterNode, keyConstraintNode)
}

func (b *ballerinaParser) parseRowTypeParameter() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM)
	rowTypeParameterNode := b.parseTypeParameter()
	b.endContext()
	return rowTypeParameterNode
}

func (b *ballerinaParser) parseTypeParameter() st.STNode {
	ltToken := b.parseLTToken()
	typeNode := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS)
	gtToken := b.parseGTToken()
	return st.CreateTypeParameterNode(ltToken, typeNode, gtToken)
}

func (b *ballerinaParser) parseKeyConstraint(keyKeywordToken st.STNode) st.STNode {
	switch b.peek().Kind() {
	case st.OPEN_PAREN_TOKEN:
		return b.parseKeySpecifierWithKeyKeywordToken(keyKeywordToken)
	case st.LT_TOKEN:
		return b.parseKeyTypeConstraint(keyKeywordToken)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS)
		return b.parseKeyConstraint(keyKeywordToken)
	}
}

func (b *ballerinaParser) parseKeySpecifierWithKeyKeywordToken(keyKeywordToken st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_KEY_SPECIFIER)
	openParenToken := b.parseOpenParenthesis()
	fieldNamesNode := b.parseFieldNames()
	closeParenToken := b.parseCloseParenthesis()
	b.endContext()
	return st.CreateKeySpecifierNode(keyKeywordToken, openParenToken, fieldNamesNode, closeParenToken)
}

func (b *ballerinaParser) parseKeyTypeConstraint(keyKeywordToken st.STNode) st.STNode {
	typeParameterNode := b.parseTypeParameter()
	return st.CreateKeyTypeConstraintNode(keyKeywordToken, typeParameterNode)
}

func (b *ballerinaParser) parseFunctionTypeDesc(qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC)
	functionKeyword := b.parseFunctionKeyword()
	hasFuncSignature := false
	signature := st.CreateEmptyNode()
	if (b.peek().Kind() == st.OPEN_PAREN_TOKEN) || b.isSyntaxKindInList(qualifiers, st.TRANSACTIONAL_KEYWORD) {
		signature = b.parseFuncSignature(true)
		hasFuncSignature = true
	}
	nodes := b.createFuncTypeQualNodeList(qualifiers, functionKeyword, hasFuncSignature)
	qualifierList := nodes[0]
	functionKeyword = nodes[1]
	b.endContext()
	return st.CreateFunctionTypeDescriptorNode(qualifierList, functionKeyword, signature)
}

func (b *ballerinaParser) getLastNodeInList(nodeList []st.STNode) st.STNode {
	return nodeList[len(nodeList)-1]
}

func (b *ballerinaParser) createFuncTypeQualNodeList(qualifierList []st.STNode, functionKeyword st.STNode, hasFuncSignature bool) []st.STNode {
	var validatedList []st.STNode
	i := 0
	for ; i < len(qualifierList); i++ {
		qualifier := qualifierList[i]
		nextIndex := (i + 1)
		if b.isSyntaxKindInList(validatedList, qualifier.Kind()) {
			qualifierToken, ok := qualifier.(st.STToken)
			if !ok {
				b.internalError("createFuncTypeQualNodeList: expected STToken")
				return nil
			}
			b.updateLastNodeInListWithInvalidNode(validatedList, qualifier,
				&common.ERROR_DUPLICATE_QUALIFIER, qualifierToken.Text())
		} else if hasFuncSignature && b.isRegularFuncQual(qualifier.Kind()) {
			validatedList = append(validatedList, qualifier)
		} else if qualifier.Kind() == st.ISOLATED_KEYWORD {
			validatedList = append(validatedList, qualifier)
		} else if len(qualifierList) == nextIndex {
			functionKeyword = st.CloneWithLeadingInvalidNodeMinutiae(functionKeyword, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		} else {
			b.updateANodeInListWithLeadingInvalidNode(qualifierList, nextIndex, qualifier,
				&common.ERROR_QUALIFIER_NOT_ALLOWED, st.ToToken(qualifier).Text())
		}
	}
	nodeList := st.CreateNodeList(validatedList...)
	return []st.STNode{nodeList, functionKeyword}
}

func (b *ballerinaParser) isRegularFuncQual(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.ISOLATED_KEYWORD, st.TRANSACTIONAL_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseExplicitFunctionExpression(annots st.STNode, qualifiers []st.STNode, isRhsExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION)
	funcKeyword := b.parseFunctionKeyword()
	nodes := b.createFuncTypeQualNodeList(qualifiers, funcKeyword, true)
	qualifierList := nodes[0]
	funcKeyword = nodes[1]
	funcSignature := b.parseFuncSignature(false)
	funcBody := b.parseAnonFuncBody(isRhsExpr)
	return st.CreateExplicitAnonymousFunctionExpressionNode(annots, qualifierList, funcKeyword,
		funcSignature, funcBody)
}

func (b *ballerinaParser) parseAnonFuncBody(isRhsExpr bool) st.STNode {
	switch b.peek().Kind() {
	case st.OPEN_BRACE_TOKEN,
		st.EOF_TOKEN:
		body := b.parseFunctionBodyBlock(true)
		b.endContext()
		return body
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		b.endContext()
		return b.parseExpressionFuncBody(true, isRhsExpr)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY)
		return b.parseAnonFuncBody(isRhsExpr)
	}
}

func (b *ballerinaParser) parseExpressionFuncBody(isAnon bool, isRhsExpr bool) st.STNode {
	rightDoubleArrow := b.parseDoubleRightArrow()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceRemoteCallAction, isRhsExpr, false)
	var semiColon st.STNode
	if isAnon {
		semiColon = st.CreateEmptyNode()
	} else {
		semiColon = b.parseSemicolon()
	}
	return st.CreateExpressionFunctionBodyNode(rightDoubleArrow, expression, semiColon)
}

func (b *ballerinaParser) parseDoubleRightArrow() st.STNode {
	token := b.peek()
	if token.Kind() == st.RIGHT_DOUBLE_ARROW_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START)
		return b.parseDoubleRightArrow()
	}
}

func (b *ballerinaParser) parseImplicitAnonFuncWithParams(params st.STNode, isRhsExpr bool) st.STNode {
	switch params.Kind() {
	case st.SIMPLE_NAME_REFERENCE, st.INFER_PARAM_LIST:
		break
	case st.BRACED_EXPRESSION:
		bracedExpr, ok := params.(*st.STBracedExpressionNode)
		if !ok {
			b.internalError("parseImplicitAnonFunc: expected STBracedExpressionNode")
			return st.CreateEmptyNode()
		}
		params = b.getAnonFuncParam(*bracedExpr)
	case st.NIL_LITERAL:
		nilLiteralNode, ok := params.(*st.STNilLiteralNode)
		if !ok {
			b.internalError("expected STNilLiteralNode")
			return st.CreateEmptyNode()
		}
		params = st.CreateImplicitAnonymousFunctionParameters(nilLiteralNode.OpenParenToken,
			st.CreateNodeList(), nilLiteralNode.CloseParenToken)
	default:
		var syntheticParam st.STNode
		syntheticParam = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		syntheticParam = st.CloneWithLeadingInvalidNodeMinutiae(syntheticParam, params,
			&common.ERROR_INVALID_PARAM_LIST_IN_INFER_ANONYMOUS_FUNCTION_EXPR)
		params = st.CreateSimpleNameReferenceNode(syntheticParam)
	}
	rightDoubleArrow := b.parseDoubleRightArrow()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceRemoteCallAction, isRhsExpr, false)
	return st.CreateImplicitAnonymousFunctionExpressionNode(params, rightDoubleArrow, expression)
}

func (b *ballerinaParser) getAnonFuncParam(bracedExpression st.STBracedExpressionNode) st.STNode {
	var paramList []st.STNode
	innerExpression := bracedExpression.Expression
	openParen := bracedExpression.OpenParen
	if innerExpression.Kind() == st.SIMPLE_NAME_REFERENCE {
		paramList = append(paramList, innerExpression)
	} else {
		openParen = st.CloneWithTrailingInvalidNodeMinutiae(openParen, innerExpression,
			&common.ERROR_INVALID_PARAM_LIST_IN_INFER_ANONYMOUS_FUNCTION_EXPR)
	}
	return st.CreateImplicitAnonymousFunctionParameters(openParen,
		st.CreateNodeList(paramList...), bracedExpression.CloseParen)
}

func (b *ballerinaParser) parseImplicitAnonFuncWithOpenParenAndFirstParam(openParen st.STNode, firstParam st.STNode, isRhsExpr bool) st.STNode {
	var paramList []st.STNode
	paramList = append(paramList, firstParam)
	nextToken := b.peek()
	var paramEnd st.STNode
	var param st.STNode
	for !b.isEndOfAnonFuncParametersList(nextToken.Kind()) {
		paramEnd = b.parseImplicitAnonFuncParamEnd()
		if paramEnd == nil {
			break
		}
		paramList = append(paramList, paramEnd)
		param = b.parseIdentifier(common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM)
		param = st.CreateSimpleNameReferenceNode(param)
		paramList = append(paramList, param)
		nextToken = b.peek()
	}
	params := st.CreateNodeList(paramList...)
	closeParen := b.parseCloseParenthesis()
	b.endContext()
	inferedParams := st.CreateImplicitAnonymousFunctionParameters(openParen, params, closeParen)
	return b.parseImplicitAnonFuncWithParams(inferedParams, isRhsExpr)
}

func (b *ballerinaParser) parseImplicitAnonFuncParamEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS)
		return b.parseImplicitAnonFuncParamEnd()
	}
}

func (b *ballerinaParser) isEndOfAnonFuncParametersList(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.EOF_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.CLOSE_BRACKET_TOKEN,
		st.SEMICOLON_TOKEN,
		st.RETURNS_KEYWORD,
		st.TYPE_KEYWORD,
		st.LISTENER_KEYWORD,
		st.IF_KEYWORD,
		st.WHILE_KEYWORD,
		st.DO_KEYWORD,
		st.OPEN_BRACE_TOKEN,
		st.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseTupleTypeDesc() st.STNode {
	openBracket := b.parseOpenBracket()
	b.startContext(common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS)
	memberTypeDesc := b.parseTupleMemberTypeDescList()
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return st.CreateTupleTypeDescriptorNode(openBracket, memberTypeDesc, closeBracket)
}

func (b *ballerinaParser) parseTupleMemberTypeDescList() st.STNode {
	var typeDescList []st.STNode
	nextToken := b.peek()
	if b.isEndOfTypeList(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	typeDesc := b.parseTupleMember()
	return b.parseTupleTypeMembers(typeDesc, typeDescList)
}

func (b *ballerinaParser) parseTupleTypeMembers(firstMember st.STNode, memberList []st.STNode) st.STNode {
	var tupleMemberRhs st.STNode
	for !b.isEndOfTypeList(b.peek().Kind()) {
		if firstMember.Kind() == st.REST_TYPE {
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
	return st.CreateNodeList(memberList...)
}

func (b *ballerinaParser) parseTupleMember() st.STNode {
	annot := b.parseOptionalAnnotations()
	typeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	return b.createMemberOrRestNode(annot, typeDesc)
}

func (b *ballerinaParser) createMemberOrRestNode(annot st.STNode, typeDesc st.STNode) st.STNode {
	tupleMemberRhs := b.parseTypeDescInTupleRhs()
	if tupleMemberRhs != nil {
		annotList, ok := annot.(*st.STNodeList)
		if !ok {
			b.internalError("createMemberOrRestNode: expected st.STNodeList")
			return st.CreateEmptyNode()
		}
		if !annotList.IsEmpty() {
			typeDesc = st.CloneWithLeadingInvalidNodeMinutiae(typeDesc, annot,
				&common.ERROR_ANNOTATIONS_NOT_ALLOWED_FOR_TUPLE_REST_DESCRIPTOR)
		}
		return st.CreateRestDescriptorNode(typeDesc, tupleMemberRhs)
	}
	return st.CreateMemberTypeDescriptorNode(annot, typeDesc)
}

func (b *ballerinaParser) invalidateTypeDescAfterRestDesc(restDescriptor st.STNode) st.STNode {
	for !b.isEndOfTypeList(b.peek().Kind()) {
		tupleMemberRhs := b.parseTupleMemberRhs()
		if tupleMemberRhs == nil {
			break
		}
		restDescriptor = st.CloneWithTrailingInvalidNodeMinutiae(restDescriptor, tupleMemberRhs, nil)
		restDescriptor = st.CloneWithTrailingInvalidNodeMinutiae(restDescriptor, b.parseTupleMember(),
			&common.ERROR_TYPE_DESC_AFTER_REST_DESCRIPTOR)
	}
	return restDescriptor
}

func (b *ballerinaParser) parseTupleMemberRhs() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS)
		return b.parseTupleMemberRhs()
	}
}

func (b *ballerinaParser) parseTypeDescInTupleRhs() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN, st.CLOSE_BRACKET_TOKEN:
		return nil
	case st.ELLIPSIS_TOKEN:
		return b.parseEllipsis()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS)
		return b.parseTypeDescInTupleRhs()
	}
}

func (b *ballerinaParser) isEndOfTypeList(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.CLOSE_BRACKET_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.EOF_TOKEN,
		st.EQUAL_TOKEN,
		st.SEMICOLON_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseTableConstructorOrQuery(isRhsExpr bool, allowActions bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION)
	tableOrQueryExpr := b.parseTableConstructorOrQueryInner(isRhsExpr, allowActions)
	b.endContext()
	return tableOrQueryExpr
}

func (b *ballerinaParser) parseTableConstructorOrQueryInner(isRhsExpr bool, allowActions bool) st.STNode {
	var queryConstructType st.STNode
	switch b.peek().Kind() {
	case st.FROM_KEYWORD:
		queryConstructType = st.CreateEmptyNode()
		return b.parseQueryExprRhs(queryConstructType, isRhsExpr, allowActions)
	case st.TABLE_KEYWORD:
		tableKeyword := b.parseTableKeyword()
		return b.parseTableConstructorOrQueryWithKeyword(tableKeyword, isRhsExpr, allowActions)
	case st.STREAM_KEYWORD,
		st.MAP_KEYWORD:
		streamOrMapKeyword := b.consume()
		keySpecifier := st.CreateEmptyNode()
		queryConstructType = b.parseQueryConstructType(streamOrMapKeyword, keySpecifier)
		return b.parseQueryExprRhs(queryConstructType, isRhsExpr, allowActions)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START)
		return b.parseTableConstructorOrQueryInner(isRhsExpr, allowActions)
	}
}

func (b *ballerinaParser) parseTableConstructorOrQueryWithKeyword(tableKeyword st.STNode, isRhsExpr bool, allowActions bool) st.STNode {
	var keySpecifier st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACKET_TOKEN:
		keySpecifier = st.CreateEmptyNode()
		return b.parseTableConstructorExprRhs(tableKeyword, keySpecifier)
	case st.KEY_KEYWORD:
		keySpecifier = b.parseKeySpecifier()
		return b.parseTableConstructorOrQueryRhs(tableKeyword, keySpecifier, isRhsExpr, allowActions)
	case st.IDENTIFIER_TOKEN:
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

func (b *ballerinaParser) parseTableConstructorOrQueryRhs(tableKeyword st.STNode, keySpecifier st.STNode, isRhsExpr bool, allowActions bool) st.STNode {
	switch b.peek().Kind() {
	case st.FROM_KEYWORD:
		return b.parseQueryExprRhs(b.parseQueryConstructType(tableKeyword, keySpecifier), isRhsExpr, allowActions)
	case st.OPEN_BRACKET_TOKEN:
		return b.parseTableConstructorExprRhs(tableKeyword, keySpecifier)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS)
		return b.parseTableConstructorOrQueryRhs(tableKeyword, keySpecifier, isRhsExpr, allowActions)
	}
}

func (b *ballerinaParser) parseQueryConstructType(keyword st.STNode, keySpecifier st.STNode) st.STNode {
	return st.CreateQueryConstructTypeNode(keyword, keySpecifier)
}

func (b *ballerinaParser) parseQueryExprRhs(queryConstructType st.STNode, isRhsExpr bool, allowActions bool) st.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
	fromClause := b.parseFromClause(isRhsExpr, allowActions)
	var clauses []st.STNode
	var intermediateClause st.STNode
	var selectClause st.STNode
	var collectClause st.STNode
	for !b.isEndOfIntermediateClause(b.peek().Kind()) {
		intermediateClause = b.parseIntermediateClause(isRhsExpr, allowActions)
		if intermediateClause == nil {
			break
		}

		// If there are more clauses after select clause they are add as invalid nodes to the select clause
		if selectClause != nil {
			selectClause = st.CloneWithTrailingInvalidNodeMinutiae(selectClause, intermediateClause,
				&common.ERROR_MORE_CLAUSES_AFTER_SELECT_CLAUSE)
			continue
		} else if collectClause != nil {
			collectClause = st.CloneWithTrailingInvalidNodeMinutiae(collectClause, intermediateClause,
				&common.ERROR_MORE_CLAUSES_AFTER_COLLECT_CLAUSE)
			continue
		}
		if intermediateClause.Kind() == st.SELECT_CLAUSE {
			selectClause = intermediateClause
		} else if intermediateClause.Kind() == st.COLLECT_CLAUSE {
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
	if (b.peek().Kind() == st.DO_KEYWORD) && ((!b.isNestedQueryExpr()) || ((selectClause == nil) && (collectClause == nil))) {
		intermediateClauses := st.CreateNodeList(clauses...)
		queryPipeline := st.CreateQueryPipelineNode(fromClause, intermediateClauses)
		return b.parseQueryAction(queryConstructType, queryPipeline, selectClause, collectClause)
	}
	if (selectClause == nil) && (collectClause == nil) {
		selectKeyword := st.CreateMissingToken(st.SELECT_KEYWORD, nil)
		expr := st.CreateSimpleNameReferenceNode(st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil))
		selectClause = st.CreateSelectClauseNode(selectKeyword, expr)

		// Now we need to attach the diagnostic to the last intermediate clause.
		// If there are no intermediate clauses, then attach to the from clause.
		if len(clauses) == 0 {
			fromClause = st.AddDiagnostic(fromClause, &common.ERROR_MISSING_SELECT_CLAUSE)
		} else {
			lastIndex := (len(clauses) - 1)
			intClauseWithDiagnostic := st.AddDiagnostic(clauses[lastIndex],
				&common.ERROR_MISSING_SELECT_CLAUSE)
			clauses[lastIndex] = intClauseWithDiagnostic
		}
	}
	intermediateClauses := st.CreateNodeList(clauses...)
	queryPipeline := st.CreateQueryPipelineNode(fromClause, intermediateClauses)
	onConflictClause := b.parseOnConflictClause(isRhsExpr)
	var clause st.STNode
	if selectClause == nil {
		clause = collectClause
	} else {
		clause = selectClause
	}
	return st.CreateQueryExpressionNode(queryConstructType, queryPipeline,
		clause, onConflictClause)
}

func (b *ballerinaParser) isNestedQueryExpr() bool {
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

func (b *ballerinaParser) isValidIntermediateQueryStart(token st.STToken) bool {
	switch token.Kind() {
	case st.FROM_KEYWORD,
		st.WHERE_KEYWORD,
		st.LET_KEYWORD,
		st.SELECT_KEYWORD,
		st.JOIN_KEYWORD,
		st.OUTER_KEYWORD,
		st.ORDER_KEYWORD,
		st.BY_KEYWORD,
		st.ASCENDING_KEYWORD,
		st.DESCENDING_KEYWORD,
		st.LIMIT_KEYWORD:
		return true
	case st.IDENTIFIER_TOKEN:
		return isGroupOrCollectKeyword(token)
	default:
		return false
	}
}

func (b *ballerinaParser) parseIntermediateClause(isRhsExpr bool, allowActions bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.FROM_KEYWORD:
		return b.parseFromClause(isRhsExpr, allowActions)
	case st.WHERE_KEYWORD:
		return b.parseWhereClause(isRhsExpr)
	case st.LET_KEYWORD:
		return b.parseLetClause(isRhsExpr, allowActions)
	case st.SELECT_KEYWORD:
		return b.parseSelectClause(isRhsExpr, allowActions)
	case st.JOIN_KEYWORD, st.OUTER_KEYWORD:
		return b.parseJoinClause(isRhsExpr)
	case st.ORDER_KEYWORD,
		st.ASCENDING_KEYWORD,
		st.DESCENDING_KEYWORD:
		return b.parseOrderByClause(isRhsExpr)
	case st.LIMIT_KEYWORD:
		return b.parseLimitClause(isRhsExpr)
	case st.DO_KEYWORD,
		st.SEMICOLON_TOKEN,
		st.ON_KEYWORD,
		st.CONFLICT_KEYWORD:
		return nil
	default:
		if isKeywordMatch(st.COLLECT_KEYWORD, nextToken) {
			return b.parseCollectClause(isRhsExpr)
		}
		if isKeywordMatch(st.GROUP_KEYWORD, nextToken) {
			return b.parseGroupByClause(isRhsExpr)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS)
		return b.parseIntermediateClause(isRhsExpr, allowActions)
	}
}

func (b *ballerinaParser) parseCollectClause(isRhsExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE)
	collectKeyword := b.parseCollectKeyword()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	b.endContext()
	return st.CreateCollectClauseNode(collectKeyword, expression)
}

func (b *ballerinaParser) parseCollectKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.COLLECT_KEYWORD {
		return b.consume()
	}
	if isKeywordMatch(st.COLLECT_KEYWORD, token) {
		return b.getCollectKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD)
	return b.parseCollectKeyword()
}

func (b *ballerinaParser) getCollectKeyword(token st.STToken) st.STNode {
	return st.CreateTokenWithDiagnostics(st.COLLECT_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *ballerinaParser) parseJoinKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.JOIN_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_JOIN_KEYWORD)
		return b.parseJoinKeyword()
	}
}

func (b *ballerinaParser) parseEqualsKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.EQUALS_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD)
		return b.parseEqualsKeyword()
	}
}

func (b *ballerinaParser) isEndOfIntermediateClause(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.CLOSE_BRACE_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.CLOSE_BRACKET_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.SEMICOLON_TOKEN,
		st.PUBLIC_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.EOF_TOKEN,
		st.RESOURCE_KEYWORD,
		st.LISTENER_KEYWORD,
		st.DOCUMENTATION_STRING,
		st.PRIVATE_KEYWORD,
		st.RETURNS_KEYWORD,
		st.SERVICE_KEYWORD,
		st.TYPE_KEYWORD,
		st.CONST_KEYWORD,
		st.FINAL_KEYWORD,
		st.DO_KEYWORD,
		st.ON_KEYWORD,
		st.CONFLICT_KEYWORD:
		return true
	default:
		return b.isValidExprRhsStart(tokenKind, st.NONE)
	}
}

func (b *ballerinaParser) parseFromClause(isRhsExpr bool, allowActions bool) st.STNode {
	fromKeyword := b.parseFromKeyword()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_FROM_CLAUSE)
	inKeyword := b.parseInKeyword()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, allowActions)
	return st.CreateFromClauseNode(fromKeyword, typedBindingPattern, inKeyword, expression)
}

func (b *ballerinaParser) parseFromKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.FROM_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FROM_KEYWORD)
		return b.parseFromKeyword()
	}
}

func (b *ballerinaParser) parseWhereClause(isRhsExpr bool) st.STNode {
	whereKeyword := b.parseWhereKeyword()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	return st.CreateWhereClauseNode(whereKeyword, expression)
}

func (b *ballerinaParser) parseWhereKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.WHERE_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_WHERE_KEYWORD)
		return b.parseWhereKeyword()
	}
}

func (b *ballerinaParser) parseLimitKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.LIMIT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD)
		return b.parseLimitKeyword()
	}
}

func (b *ballerinaParser) parseLetClause(isRhsExpr bool, allowActions bool) st.STNode {
	letKeyword := b.parseLetKeyword()
	letVarDeclarations := b.parseLetVarDeclarations(common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL, isRhsExpr,
		allowActions)
	letKeyword = b.cloneWithDiagnosticIfListEmpty(letVarDeclarations, letKeyword,
		&common.ERROR_MISSING_LET_VARIABLE_DECLARATION)
	return st.CreateLetClauseNode(letKeyword, letVarDeclarations)
}

func (b *ballerinaParser) parseGroupByClause(isRhsExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE)
	groupKeyword := b.parseGroupKeyword()
	byKeyword := b.parseByKeyword()
	groupingKeys := b.parseGroupingKeyList(isRhsExpr)
	byKeyword = b.cloneWithDiagnosticIfListEmpty(groupingKeys, byKeyword,
		&common.ERROR_MISSING_GROUPING_KEY)
	b.endContext()
	return st.CreateGroupByClauseNode(groupKeyword, byKeyword, groupingKeys)
}

func (b *ballerinaParser) parseGroupKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.GROUP_KEYWORD {
		return b.consume()
	}
	if isKeywordMatch(st.GROUP_KEYWORD, token) {
		return b.getGroupKeyword(b.consume())
	}
	b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_GROUP_KEYWORD)
	return b.parseGroupKeyword()
}

func (b *ballerinaParser) getGroupKeyword(token st.STToken) st.STNode {
	return st.CreateTokenWithDiagnostics(st.GROUP_KEYWORD, token.LeadingMinutiae(), token.TrailingMinutiae(),
		token.Diagnostics())
}

func (b *ballerinaParser) parseOrderKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ORDER_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ORDER_KEYWORD)
		return b.parseOrderKeyword()
	}
}

func (b *ballerinaParser) parseByKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.BY_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BY_KEYWORD)
		return b.parseByKeyword()
	}
}

func (b *ballerinaParser) parseOrderByClause(isRhsExpr bool) st.STNode {
	orderKeyword := b.parseOrderKeyword()
	byKeyword := b.parseByKeyword()
	orderKeys := b.parseOrderKeyList(isRhsExpr)
	byKeyword = b.cloneWithDiagnosticIfListEmpty(orderKeys, byKeyword, &common.ERROR_MISSING_ORDER_KEY)
	return st.CreateOrderByClauseNode(orderKeyword, byKeyword, orderKeys)
}

func (b *ballerinaParser) parseGroupingKeyList(isRhsExpr bool) st.STNode {
	var groupingKeys []st.STNode
	nextToken := b.peek()
	if b.isEndOfGroupByKeyListElement(nextToken) {
		return st.CreateEmptyNodeList()
	}
	groupingKey := b.parseGroupingKey(isRhsExpr)
	groupingKeys = append(groupingKeys, groupingKey)
	nextToken = b.peek()
	var groupingKeyListMemberEnd st.STNode
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
	return st.CreateNodeList(groupingKeys...)
}

func (b *ballerinaParser) parseOrderKeyList(isRhsExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST)
	var orderKeys []st.STNode
	nextToken := b.peek()
	if b.isEndOfOrderKeys(nextToken) {
		b.endContext()
		return st.CreateEmptyNodeList()
	}
	orderKey := b.parseOrderKey(isRhsExpr)
	orderKeys = append(orderKeys, orderKey)
	nextToken = b.peek()
	var orderKeyListMemberEnd st.STNode
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
	return st.CreateNodeList(orderKeys...)
}

func (b *ballerinaParser) isEndOfGroupByKeyListElement(nextToken st.STToken) bool {
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return false
	case st.EOF_TOKEN:
		return true
	default:
		return b.isQueryClauseStartToken(nextToken)
	}
}

func (b *ballerinaParser) isEndOfOrderKeys(nextToken st.STToken) bool {
	switch nextToken.Kind() {
	case st.COMMA_TOKEN,
		st.ASCENDING_KEYWORD,
		st.DESCENDING_KEYWORD:
		return false
	case st.SEMICOLON_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return b.isQueryClauseStartToken(nextToken)
	}
}

func (b *ballerinaParser) isQueryClauseStartToken(nextToken st.STToken) bool {
	switch nextToken.Kind() {
	case st.SELECT_KEYWORD,
		st.LET_KEYWORD,
		st.WHERE_KEYWORD,
		st.OUTER_KEYWORD,
		st.JOIN_KEYWORD,
		st.ORDER_KEYWORD,
		st.DO_KEYWORD,
		st.FROM_KEYWORD,
		st.LIMIT_KEYWORD:
		return true
	case st.IDENTIFIER_TOKEN:
		return isGroupOrCollectKeyword(nextToken)
	default:
		return false
	}
}

func (b *ballerinaParser) parseGroupingKeyListMemberEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return b.consume()
	case st.EOF_TOKEN:
		return nil
	default:
		if b.isQueryClauseStartToken(nextToken) {
			return nil
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END)
		return b.parseGroupingKeyListMemberEnd()
	}
}

func (b *ballerinaParser) parseOrderKeyListMemberEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.EOF_TOKEN:
		return nil
	default:
		if b.isQueryClauseStartToken(nextToken) {
			return nil
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END)
		return b.parseOrderKeyListMemberEnd()
	}
}

func (b *ballerinaParser) parseGroupingKeyVariableDeclaration(isRhsExpr bool) st.STNode {
	groupingKeyElementTypeDesc := b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY)
	b.startContext(common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER)
	groupingKeySimpleBP := b.createCaptureOrWildcardBP(b.parseVariableName())
	b.endContext()
	equalsToken := b.parseAssignOp()
	groupingKeyExpression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	return st.CreateGroupingKeyVarDeclarationNode(groupingKeyElementTypeDesc, groupingKeySimpleBP,
		equalsToken, groupingKeyExpression)
}

func (b *ballerinaParser) parseGroupingKey(isRhsExpr bool) st.STNode {
	nextToken := b.peek()
	nextTokenKind := nextToken.Kind()
	if (nextTokenKind == st.IDENTIFIER_TOKEN) && (!b.isPossibleGroupingKeyVarDeclaration()) {
		return st.CreateSimpleNameReferenceNode(b.parseVariableName())
	} else if isTypeStartingToken(nextTokenKind, nextToken) {
		return b.parseGroupingKeyVariableDeclaration(isRhsExpr)
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT)
	return b.parseGroupingKey(isRhsExpr)
}

func (b *ballerinaParser) isPossibleGroupingKeyVarDeclaration() bool {
	nextNextTokenKind := b.getNextNextToken().Kind()
	return ((nextNextTokenKind == st.EQUAL_TOKEN) || ((nextNextTokenKind == st.IDENTIFIER_TOKEN) && (b.peekN(3).Kind() == st.EQUAL_TOKEN)))
}

func (b *ballerinaParser) parseOrderKey(isRhsExpr bool) st.STNode {
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	var orderDirection st.STNode
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ASCENDING_KEYWORD, st.DESCENDING_KEYWORD:
		orderDirection = b.consume()
	default:
		orderDirection = st.CreateEmptyNode()
	}
	return st.CreateOrderKeyNode(expression, orderDirection)
}

func (b *ballerinaParser) parseSelectClause(isRhsExpr bool, allowActions bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_SELECT_CLAUSE)
	selectKeyword := b.parseSelectKeyword()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, allowActions)
	b.endContext()
	return st.CreateSelectClauseNode(selectKeyword, expression)
}

func (b *ballerinaParser) parseSelectKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.SELECT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SELECT_KEYWORD)
		return b.parseSelectKeyword()
	}
}

func (b *ballerinaParser) parseOnConflictClause(isRhsExpr bool) st.STNode {
	nextToken := b.peek()
	if (nextToken.Kind() != st.ON_KEYWORD) && (nextToken.Kind() != st.CONFLICT_KEYWORD) {
		return st.CreateEmptyNode()
	}
	b.startContext(common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE)
	onKeyword := b.parseOnKeyword()
	conflictKeyword := b.parseConflictKeyword()
	b.endContext()
	expr := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	return st.CreateOnConflictClauseNode(onKeyword, conflictKeyword, expr)
}

func (b *ballerinaParser) parseConflictKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.CONFLICT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD)
		return b.parseConflictKeyword()
	}
}

func (b *ballerinaParser) parseLimitClause(isRhsExpr bool) st.STNode {
	limitKeyword := b.parseLimitKeyword()
	expr := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	return st.CreateLimitClauseNode(limitKeyword, expr)
}

func (b *ballerinaParser) parseJoinClause(isRhsExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_JOIN_CLAUSE)
	var outerKeyword st.STNode
	nextToken := b.peek()
	if nextToken.Kind() == st.OUTER_KEYWORD {
		outerKeyword = b.consume()
	} else {
		outerKeyword = st.CreateEmptyNode()
	}
	joinKeyword := b.parseJoinKeyword()
	typedBindingPattern := b.parseTypedBindingPatternWithContext(common.PARSER_RULE_CONTEXT_JOIN_CLAUSE)
	inKeyword := b.parseInKeyword()
	expression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	b.endContext()
	onCondition := b.parseOnClause(isRhsExpr)
	return st.CreateJoinClauseNode(outerKeyword, joinKeyword, typedBindingPattern, inKeyword, expression,
		onCondition)
}

func (b *ballerinaParser) parseOnClause(isRhsExpr bool) st.STNode {
	nextToken := b.peek()
	if b.isQueryClauseStartToken(nextToken) {
		return b.createMissingOnClauseNode()
	}
	b.startContext(common.PARSER_RULE_CONTEXT_ON_CLAUSE)
	onKeyword := b.parseOnKeyword()
	onExpression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	equalsKeyword := b.parseEqualsKeyword()
	b.endContext()
	equalsExpression := b.parseExpressionWithPrecedence(operatorPrecedenceQuery, isRhsExpr, false)
	return st.CreateOnClauseNode(onKeyword, onExpression, equalsKeyword, equalsExpression)
}

func (b *ballerinaParser) createMissingOnClauseNode() st.STNode {
	onKeyword := st.CreateMissingTokenWithDiagnostics(st.ON_KEYWORD,
		&common.ERROR_MISSING_ON_KEYWORD)
	identifier := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
		&common.ERROR_MISSING_IDENTIFIER)
	equalsKeyword := st.CreateMissingTokenWithDiagnostics(st.EQUALS_KEYWORD,
		&common.ERROR_MISSING_EQUALS_KEYWORD)
	onExpression := st.CreateSimpleNameReferenceNode(identifier)
	equalsExpression := st.CreateSimpleNameReferenceNode(identifier)
	return st.CreateOnClauseNode(onKeyword, onExpression, equalsKeyword, equalsExpression)
}

func (b *ballerinaParser) parseStartAction(annots st.STNode) st.STNode {
	startKeyword := b.parseStartKeyword()
	expr := b.parseActionOrExpression()
	switch expr.Kind() {
	case st.FUNCTION_CALL,
		st.METHOD_CALL,
		st.REMOTE_METHOD_CALL_ACTION,
		st.CLIENT_RESOURCE_ACCESS_ACTION:
		break
	case st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE,
		st.FIELD_ACCESS,
		st.ASYNC_SEND_ACTION:
		expr = b.generateValidExprForStartAction(expr)
	default:
		startKeyword = st.CloneWithTrailingInvalidNodeMinutiae(startKeyword, expr,
			&common.ERROR_INVALID_EXPRESSION_IN_START_ACTION)
		var funcName st.STNode
		funcName = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		funcName = st.CreateSimpleNameReferenceNode(funcName)
		openParenToken := st.CreateMissingToken(st.OPEN_PAREN_TOKEN, nil)
		closeParenToken := st.CreateMissingToken(st.CLOSE_PAREN_TOKEN, nil)
		expr = st.CreateFunctionCallExpressionNode(funcName, openParenToken,
			st.CreateEmptyNodeList(), closeParenToken)
	}
	return st.CreateStartActionNode(b.getAnnotations(annots), startKeyword, expr)
}

func (b *ballerinaParser) generateValidExprForStartAction(expr st.STNode) st.STNode {
	openParenToken := st.CreateMissingTokenWithDiagnostics(st.OPEN_PAREN_TOKEN,
		&common.ERROR_MISSING_OPEN_PAREN_TOKEN)
	arguments := st.CreateEmptyNodeList()
	closeParenToken := st.CreateMissingTokenWithDiagnostics(st.CLOSE_PAREN_TOKEN,
		&common.ERROR_MISSING_CLOSE_PAREN_TOKEN)
	switch expr.Kind() {
	case st.FIELD_ACCESS:
		fieldAccessExpr, ok := expr.(*st.STFieldAccessExpressionNode)
		if !ok {
			b.internalError("expected STFieldAccessExpressionNode")
			return st.CreateEmptyNode()
		}
		return st.CreateMethodCallExpressionNode(fieldAccessExpr.Expression,
			fieldAccessExpr.DotToken, fieldAccessExpr.FieldName, openParenToken, arguments,
			closeParenToken)
	case st.ASYNC_SEND_ACTION:
		asyncSendAction, ok := expr.(*st.STAsyncSendActionNode)
		if !ok {
			b.internalError("expected STAsyncSendActionNode")
			return st.CreateEmptyNode()
		}
		return st.CreateRemoteMethodCallActionNode(asyncSendAction.Expression,
			asyncSendAction.RightArrowToken, asyncSendAction.PeerWorker, openParenToken, arguments,
			closeParenToken)
	default:
		return st.CreateFunctionCallExpressionNode(expr, openParenToken, arguments, closeParenToken)
	}
}

func (b *ballerinaParser) parseStartKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.START_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_START_KEYWORD)
		return b.parseStartKeyword()
	}
}

func (b *ballerinaParser) parseFlushAction() st.STNode {
	flushKeyword := b.parseFlushKeyword()
	peerWorker := b.parseOptionalPeerWorkerName()
	return st.CreateFlushActionNode(flushKeyword, peerWorker)
}

func (b *ballerinaParser) parseFlushKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.FLUSH_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD)
		return b.parseFlushKeyword()
	}
}

func (b *ballerinaParser) parseOptionalPeerWorkerName() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.IDENTIFIER_TOKEN, st.FUNCTION_KEYWORD:
		return st.CreateSimpleNameReferenceNode(b.consume())
	default:
		return st.CreateEmptyNode()
	}
}

func (b *ballerinaParser) parseIntersectionTypeDescriptor(leftTypeDesc st.STNode, context common.ParserRuleContext, isTypedBindingPattern bool) st.STNode {
	bitwiseAndToken := b.consume()
	rightTypeDesc := b.parseTypeDescriptorInternalWithPrecedence(nil, context, isTypedBindingPattern, false,
		typePrecedenceIntersection)
	return b.mergeTypesWithIntersection(leftTypeDesc, bitwiseAndToken, rightTypeDesc)
}

func (b *ballerinaParser) createIntersectionTypeDesc(leftTypeDesc st.STNode, bitwiseAndToken st.STNode, rightTypeDesc st.STNode) st.STNode {
	leftTypeDesc = b.validateForUsageOfVar(leftTypeDesc)
	rightTypeDesc = b.validateForUsageOfVar(rightTypeDesc)
	return st.CreateIntersectionTypeDescriptorNode(leftTypeDesc, bitwiseAndToken, rightTypeDesc)
}

func (b *ballerinaParser) parseSingletonTypeDesc() st.STNode {
	simpleContExpr := b.parseSimpleConstExpr()
	return st.CreateSingletonTypeDescriptorNode(simpleContExpr)
}

func (b *ballerinaParser) parseSignedIntOrFloat() st.STNode {
	operator := b.parseUnaryOperator()
	var literal st.STNode
	nextToken := b.peek()

	switch nextToken.Kind() {

	case st.HEX_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		literal = b.parseBasicLiteral()
	default:
		literal = st.CreateBasicLiteralNode(st.NUMERIC_LITERAL,
			b.parseDecimalIntLiteral(common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN))
	}
	return st.CreateUnaryExpressionNode(operator, literal)
}

func (b *ballerinaParser) isValidExpressionStart(nextTokenKind st.SyntaxKind, nextTokenIndex int) bool {
	nextTokenIndex++
	switch nextTokenKind {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		nextNextTokenKind := b.peekN(nextTokenIndex).Kind()
		if (nextNextTokenKind == st.PIPE_TOKEN) || (nextNextTokenKind == st.BITWISE_AND_TOKEN) {
			nextTokenIndex++
			return b.isValidExpressionStart(b.peekN(nextTokenIndex).Kind(), nextTokenIndex)
		}
		return ((((nextNextTokenKind == st.SEMICOLON_TOKEN) || (nextNextTokenKind == st.COMMA_TOKEN)) || (nextNextTokenKind == st.CLOSE_BRACKET_TOKEN)) || b.isValidExprRhsStart(nextNextTokenKind, st.SIMPLE_NAME_REFERENCE))
	case st.IDENTIFIER_TOKEN:
		return b.isValidExprRhsStart(b.peekN(nextTokenIndex).Kind(), st.SIMPLE_NAME_REFERENCE)
	case st.OPEN_PAREN_TOKEN, st.CHECK_KEYWORD, st.CHECKPANIC_KEYWORD, st.OPEN_BRACE_TOKEN,
		st.TYPEOF_KEYWORD, st.NEGATION_TOKEN, st.EXCLAMATION_MARK_TOKEN, st.TRAP_KEYWORD,
		st.OPEN_BRACKET_TOKEN, st.LT_TOKEN, st.FROM_KEYWORD, st.LET_KEYWORD,
		st.BACKTICK_TOKEN, st.NEW_KEYWORD, st.LEFT_ARROW_TOKEN, st.FUNCTION_KEYWORD,
		st.TRANSACTIONAL_KEYWORD, st.ISOLATED_KEYWORD, st.BASE16_KEYWORD, st.BASE64_KEYWORD,
		st.NATURAL_KEYWORD:
		return true
	case st.PLUS_TOKEN, st.MINUS_TOKEN:
		return b.isValidExpressionStart(b.peekN(nextTokenIndex).Kind(), nextTokenIndex)
	case st.TABLE_KEYWORD, st.MAP_KEYWORD:
		return (b.peekN(nextTokenIndex).Kind() == st.FROM_KEYWORD)
	case st.STREAM_KEYWORD:
		nextNextToken := b.peekN(nextTokenIndex)
		return (((nextNextToken.Kind() == st.KEY_KEYWORD) || (nextNextToken.Kind() == st.OPEN_BRACKET_TOKEN)) || (nextNextToken.Kind() == st.FROM_KEYWORD))
	case st.ERROR_KEYWORD:
		return (b.peekN(nextTokenIndex).Kind() == st.OPEN_PAREN_TOKEN)
	case st.XML_KEYWORD, st.STRING_KEYWORD, st.RE_KEYWORD:
		return (b.peekN(nextTokenIndex).Kind() == st.BACKTICK_TOKEN)
	case st.START_KEYWORD,
		st.FLUSH_KEYWORD,
		st.WAIT_KEYWORD:
		fallthrough
	default:
		return false
	}
}

func (b *ballerinaParser) parseSyncSendAction(expression st.STNode) st.STNode {
	syncSendToken := b.parseSyncSendToken()
	peerWorker := b.parsePeerWorkerName()
	return st.CreateSyncSendActionNode(expression, syncSendToken, peerWorker)
}

func (b *ballerinaParser) parsePeerWorkerName() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.IDENTIFIER_TOKEN, st.FUNCTION_KEYWORD:
		return st.CreateSimpleNameReferenceNode(b.consume())
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME)
		return b.parsePeerWorkerName()
	}
}

func (b *ballerinaParser) parseSyncSendToken() st.STNode {
	token := b.peek()
	if token.Kind() == st.SYNC_SEND_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN)
		return b.parseSyncSendToken()
	}
}

func (b *ballerinaParser) parseReceiveAction() st.STNode {
	leftArrow := b.parseLeftArrowToken()
	receiveWorkers := b.parseReceiveWorkers()
	return st.CreateReceiveActionNode(leftArrow, receiveWorkers)
}

func (b *ballerinaParser) parseReceiveWorkers() st.STNode {
	switch b.peek().Kind() {
	case st.FUNCTION_KEYWORD, st.IDENTIFIER_TOKEN:
		return b.parseSingleOrAlternateReceiveWorkers()
	case st.OPEN_BRACE_TOKEN:
		return b.parseMultipleReceiveWorkers()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS)
		return b.parseReceiveWorkers()
	}
}

func (b *ballerinaParser) parseSingleOrAlternateReceiveWorkers() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER)
	var workers []st.STNode
	peerWorker := b.parsePeerWorkerName()
	workers = append(workers, peerWorker)
	nextToken := b.peek()
	if nextToken.Kind() != st.PIPE_TOKEN {
		b.endContext()
		return peerWorker
	}
	for nextToken.Kind() == st.PIPE_TOKEN {
		pipeToken := b.consume()
		workers = append(workers, pipeToken)
		peerWorker = b.parsePeerWorkerName()
		workers = append(workers, peerWorker)
		nextToken = b.peek()
	}
	b.endContext()
	return st.CreateAlternateReceiveNode(st.CreateNodeList(workers...))
}

func (b *ballerinaParser) parseMultipleReceiveWorkers() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS)
	openBrace := b.parseOpenBrace()
	receiveFields := b.parseReceiveFields()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	openBrace = b.cloneWithDiagnosticIfListEmpty(receiveFields, openBrace,
		&common.ERROR_MISSING_RECEIVE_FIELD_IN_RECEIVE_ACTION)
	return st.CreateReceiveFieldsNode(openBrace, receiveFields, closeBrace)
}

func (b *ballerinaParser) parseReceiveFields() st.STNode {
	var receiveFields []st.STNode
	nextToken := b.peek()
	if b.isEndOfReceiveFields(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	receiveField := b.parseReceiveField()
	receiveFields = append(receiveFields, receiveField)
	nextToken = b.peek()
	var recieveFieldEnd st.STNode
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
	return st.CreateNodeList(receiveFields...)
}

func (b *ballerinaParser) isEndOfReceiveFields(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseReceiveFieldEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END)
		return b.parseReceiveFieldEnd()
	}
}

func (b *ballerinaParser) parseReceiveField() st.STNode {
	switch b.peek().Kind() {
	case st.FUNCTION_KEYWORD:
		functionKeyword := b.consume()
		return st.CreateSimpleNameReferenceNode(functionKeyword)
	case st.IDENTIFIER_TOKEN:
		identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME)
		return b.createReceiveField(identifier)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RECEIVE_FIELD)
		return b.parseReceiveField()
	}
}

func (b *ballerinaParser) createReceiveField(identifier st.STNode) st.STNode {
	if b.peek().Kind() != st.COLON_TOKEN {
		return st.CreateSimpleNameReferenceNode(identifier)
	}
	identifier = st.CreateSimpleNameReferenceNode(identifier)
	colon := b.parseColon()
	peerWorker := b.parsePeerWorkerName()
	return st.CreateReceiveFieldNode(identifier, colon, peerWorker)
}

func (b *ballerinaParser) parseLeftArrowToken() st.STNode {
	token := b.peek()
	if token.Kind() == st.LEFT_ARROW_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN)
		return b.parseLeftArrowToken()
	}
}

func (b *ballerinaParser) parseSignedRightShiftToken() st.STNode {
	firstToken := b.consume()
	if firstToken.Kind() == st.DOUBLE_GT_TOKEN {
		return firstToken
	}
	endLGToken := b.consume()
	var doubleGTToken st.STNode
	doubleGTToken = st.CreateToken(st.DOUBLE_GT_TOKEN, firstToken.LeadingMinutiae(),
		endLGToken.TrailingMinutiae())
	if b.hasTrailingMinutiae(firstToken) {
		doubleGTToken = st.AddDiagnostic(doubleGTToken,
			&common.ERROR_NO_WHITESPACES_ALLOWED_IN_RIGHT_SHIFT_OP)
	}
	return doubleGTToken
}

func (b *ballerinaParser) parseUnsignedRightShiftToken() st.STNode {
	firstToken := b.consume()
	if firstToken.Kind() == st.TRIPPLE_GT_TOKEN {
		return firstToken
	}
	middleGTToken := b.consume()
	endLGToken := b.consume()
	var unsignedRightShiftToken st.STNode
	unsignedRightShiftToken = st.CreateToken(st.TRIPPLE_GT_TOKEN,
		firstToken.LeadingMinutiae(), endLGToken.TrailingMinutiae())
	validOpenGTToken := (!b.hasTrailingMinutiae(firstToken))
	validMiddleGTToken := (!b.hasTrailingMinutiae(middleGTToken))
	if validOpenGTToken && validMiddleGTToken {
		return unsignedRightShiftToken
	}
	unsignedRightShiftToken = st.AddDiagnostic(unsignedRightShiftToken,
		&common.ERROR_NO_WHITESPACES_ALLOWED_IN_UNSIGNED_RIGHT_SHIFT_OP)
	return unsignedRightShiftToken
}

func (b *ballerinaParser) parseWaitAction() st.STNode {
	waitKeyword := b.parseWaitKeyword()
	if b.peek().Kind() == st.OPEN_BRACE_TOKEN {
		return b.parseMultiWaitAction(waitKeyword)
	}
	return b.parseSingleOrAlternateWaitAction(waitKeyword)
}

func (b *ballerinaParser) parseWaitKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.WAIT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_WAIT_KEYWORD)
		return b.parseWaitKeyword()
	}
}

func (b *ballerinaParser) parseSingleOrAlternateWaitAction(waitKeyword st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS)
	nextToken := b.peek()
	if b.isEndOfWaitFutureExprList(nextToken.Kind()) {
		b.endContext()
		waitFutureExprs := st.CreateSimpleNameReferenceNode(st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil))
		waitFutureExprs = st.AddDiagnostic(waitFutureExprs,
			&common.ERROR_MISSING_WAIT_FUTURE_EXPRESSION)
		return st.CreateWaitActionNode(waitKeyword, waitFutureExprs)
	}
	var waitFutureExprList []st.STNode
	waitField := b.parseWaitFutureExpr()
	waitFutureExprList = append(waitFutureExprList, waitField)
	nextToken = b.peek()
	var waitFutureExprEnd st.STNode
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
	if len(waitFutureExprList) == 1 {
		return st.CreateWaitActionNode(waitKeyword, waitFutureExprList[0])
	}
	return st.CreateWaitActionNode(waitKeyword, st.CreateNodeList(waitFutureExprList...))
}

func (b *ballerinaParser) isEndOfWaitFutureExprList(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACE_TOKEN, st.SEMICOLON_TOKEN, st.OPEN_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseWaitFutureExpr() st.STNode {
	waitFutureExpr := b.parseExpressionWithPrecedence(operatorPrecedenceBitwiseOr, true, true)
	if waitFutureExpr.Kind() == st.MAPPING_CONSTRUCTOR {
		waitFutureExpr = st.AddDiagnostic(waitFutureExpr,
			&common.ERROR_MAPPING_CONSTRUCTOR_EXPR_AS_A_WAIT_EXPR)
	} else if b.isAction(waitFutureExpr) {
		waitFutureExpr = st.AddDiagnostic(waitFutureExpr, &common.ERROR_ACTION_AS_A_WAIT_EXPR)
	}
	return waitFutureExpr
}

func (b *ballerinaParser) parseWaitFutureExprEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.PIPE_TOKEN:
		return b.parsePipeToken()
	default:
		if b.isEndOfWaitFutureExprList(nextToken.Kind()) || (!b.isValidExpressionStart(nextToken.Kind(), 1)) {
			return nil
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END)
		return b.parseWaitFutureExprEnd()
	}
}

func (b *ballerinaParser) parseMultiWaitAction(waitKeyword st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS)
	openBrace := b.parseOpenBrace()
	waitFields := b.parseWaitFields()
	closeBrace := b.parseCloseBrace()
	b.endContext()
	openBrace = b.cloneWithDiagnosticIfListEmpty(waitFields, openBrace,
		&common.ERROR_MISSING_WAIT_FIELD_IN_WAIT_ACTION)
	waitFieldsNode := st.CreateWaitFieldsListNode(openBrace, waitFields, closeBrace)
	return st.CreateWaitActionNode(waitKeyword, waitFieldsNode)
}

func (b *ballerinaParser) parseWaitFields() st.STNode {
	var waitFields []st.STNode
	nextToken := b.peek()
	if b.isEndOfWaitFields(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	waitField := b.parseWaitField()
	waitFields = append(waitFields, waitField)
	nextToken = b.peek()
	var waitFieldEnd st.STNode
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
	return st.CreateNodeList(waitFields...)
}

func (b *ballerinaParser) isEndOfWaitFields(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseWaitFieldEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WAIT_FIELD_END)
		return b.parseWaitFieldEnd()
	}
}

func (b *ballerinaParser) parseWaitField() st.STNode {
	switch b.peek().Kind() {
	case st.IDENTIFIER_TOKEN:
		identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME)
		identifier = st.CreateSimpleNameReferenceNode(identifier)
		return b.createQualifiedWaitField(identifier)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME)
		return b.parseWaitField()
	}
}

func (b *ballerinaParser) createQualifiedWaitField(identifier st.STNode) st.STNode {
	if b.peek().Kind() != st.COLON_TOKEN {
		return identifier
	}
	colon := b.parseColon()
	waitFutureExpr := b.parseWaitFutureExpr()
	return st.CreateWaitFieldNode(identifier, colon, waitFutureExpr)
}

func (b *ballerinaParser) parseAnnotAccessExpression(lhsExpr st.STNode, isInConditionalExpr bool) st.STNode {
	annotAccessToken := b.parseAnnotChainingToken()
	annotTagReference := b.parseFieldAccessIdentifier(isInConditionalExpr)
	return st.CreateAnnotAccessExpressionNode(lhsExpr, annotAccessToken, annotTagReference)
}

func (b *ballerinaParser) parseAnnotChainingToken() st.STNode {
	token := b.peek()
	if token.Kind() == st.ANNOT_CHAINING_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN)
		return b.parseAnnotChainingToken()
	}
}

func (b *ballerinaParser) parseFieldAccessIdentifier(isInConditionalExpr bool) st.STNode {
	nextToken := b.peek()
	if !b.isPredeclaredIdentifier(nextToken.Kind()) {
		var identifier st.STNode = st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_IDENTIFIER)
		return b.parseQualifiedIdentifierNode(identifier, isInConditionalExpr)
	}
	return b.parseQualifiedIdentifierInner(common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER, isInConditionalExpr)
}

func (b *ballerinaParser) parseQueryAction(queryConstructType st.STNode, queryPipeline st.STNode, selectClause st.STNode, collectClause st.STNode) st.STNode {
	if queryConstructType != nil {
		queryPipeline = st.CloneWithLeadingInvalidNodeMinutiae(queryPipeline, queryConstructType,
			&common.ERROR_QUERY_CONSTRUCT_TYPE_IN_QUERY_ACTION)
	}
	if selectClause != nil {
		queryPipeline = st.CloneWithTrailingInvalidNodeMinutiae(queryPipeline, selectClause,
			&common.ERROR_SELECT_CLAUSE_IN_QUERY_ACTION)
	}
	if collectClause != nil {
		queryPipeline = st.CloneWithTrailingInvalidNodeMinutiae(queryPipeline, collectClause,
			&common.ERROR_COLLECT_CLAUSE_IN_QUERY_ACTION)
	}
	b.startContext(common.PARSER_RULE_CONTEXT_DO_CLAUSE)
	doKeyword := b.parseDoKeyword()
	blockStmt := b.parseBlockNode()
	b.endContext()
	return st.CreateQueryActionNode(queryPipeline, doKeyword, blockStmt)
}

func (b *ballerinaParser) parseDoKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.DO_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_DO_KEYWORD)
		return b.parseDoKeyword()
	}
}

func (b *ballerinaParser) parseOptionalFieldAccessExpression(lhsExpr st.STNode, isInConditionalExpr bool) st.STNode {
	optionalFieldAccessToken := b.parseOptionalChainingToken()
	fieldName := b.parseFieldAccessIdentifier(isInConditionalExpr)
	return st.CreateOptionalFieldAccessExpressionNode(lhsExpr, optionalFieldAccessToken, fieldName)
}

func (b *ballerinaParser) parseOptionalChainingToken() st.STNode {
	token := b.peek()
	if token.Kind() == st.OPTIONAL_CHAINING_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN)
		return b.parseOptionalChainingToken()
	}
}

func (b *ballerinaParser) parseConditionalExpression(lhsExpr st.STNode, isInConditionalExpr bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION)
	questionMark := b.parseQuestionMark()
	middleExpr := b.parseExpressionWithConditional(operatorPrecedenceAnonFuncOrLet, true, false, true)
	if b.peek().Kind() != st.COLON_TOKEN {
		if middleExpr.Kind() == st.CONDITIONAL_EXPRESSION {
			innerConditionalExpr, ok := middleExpr.(*st.STConditionalExpressionNode)
			if !ok {
				b.internalError("expected STConditionalExpressionNode")
				return st.CreateEmptyNode()
			}
			innerMiddleExpr := innerConditionalExpr.MiddleExpression
			rightMostQNameRef := st.GetQualifiedNameRefNode(innerMiddleExpr, false)
			if rightMostQNameRef != nil {
				middleExpr = b.generateConditionalExprForRightMost(innerConditionalExpr.LhsExpression,
					innerConditionalExpr.QuestionMarkToken, innerMiddleExpr, rightMostQNameRef)
				b.endContext()
				return st.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr,
					innerConditionalExpr.ColonToken, innerConditionalExpr.EndExpression)
			}
			leftMostQNameRef := st.GetQualifiedNameRefNode(innerMiddleExpr, true)
			if leftMostQNameRef != nil {
				middleExpr = b.generateConditionalExprForLeftMost(innerConditionalExpr.LhsExpression,
					innerConditionalExpr.QuestionMarkToken, innerMiddleExpr, leftMostQNameRef)
				b.endContext()
				return st.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr,
					innerConditionalExpr.ColonToken, innerConditionalExpr.EndExpression)
			}
		}
		rightMostQNameRef := st.GetQualifiedNameRefNode(middleExpr, false)
		if rightMostQNameRef != nil {
			b.endContext()
			return b.generateConditionalExprForRightMost(lhsExpr, questionMark, middleExpr, rightMostQNameRef)
		}
		leftMostQNameRef := st.GetQualifiedNameRefNode(middleExpr, true)
		if leftMostQNameRef != nil {
			b.endContext()
			return b.generateConditionalExprForLeftMost(lhsExpr, questionMark, middleExpr, leftMostQNameRef)
		}
	}
	return b.parseConditionalExprRhs(lhsExpr, questionMark, middleExpr, isInConditionalExpr)
}

func (b *ballerinaParser) generateConditionalExprForRightMost(lhsExpr st.STNode, questionMark st.STNode, middleExpr st.STNode, rightMostQualifiedNameRef st.STNode) st.STNode {
	qualifiedNameRef, ok := rightMostQualifiedNameRef.(*st.STQualifiedNameReferenceNode)
	if !ok {
		b.internalError("expected STQualifiedNameReferenceNode")
		return st.CreateEmptyNode()
	}
	endExpr := st.CreateSimpleNameReferenceNode(qualifiedNameRef.Identifier)
	simpleNameRef := st.GetSimpleNameRefNode(qualifiedNameRef.ModulePrefix)
	middleExpr = st.Replace(middleExpr, rightMostQualifiedNameRef, simpleNameRef)
	return st.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr, qualifiedNameRef.Colon,
		endExpr)
}

func (b *ballerinaParser) generateConditionalExprForLeftMost(lhsExpr st.STNode, questionMark st.STNode, middleExpr st.STNode, leftMostQualifiedNameRef st.STNode) st.STNode {
	qualifiedNameRef, ok := leftMostQualifiedNameRef.(*st.STQualifiedNameReferenceNode)
	if !ok {
		b.internalError("expected STQualifiedNameReferenceNode")
		return st.CreateEmptyNode()
	}
	simpleNameRef := st.CreateSimpleNameReferenceNode(qualifiedNameRef.Identifier)
	endExpr := st.Replace(middleExpr, leftMostQualifiedNameRef, simpleNameRef)
	middleExpr = st.GetSimpleNameRefNode(qualifiedNameRef.ModulePrefix)
	return st.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr, qualifiedNameRef.Colon,
		endExpr)
}

func (b *ballerinaParser) parseConditionalExprRhs(lhsExpr st.STNode, questionMark st.STNode, middleExpr st.STNode, isInConditionalExpr bool) st.STNode {
	colon := b.parseColon()
	b.endContext()
	endExpr := b.parseExpressionWithConditional(operatorPrecedenceAnonFuncOrLet, true, false,
		isInConditionalExpr)
	return st.CreateConditionalExpressionNode(lhsExpr, questionMark, middleExpr, colon, endExpr)
}

func (b *ballerinaParser) parseEnumDeclaration(metadata st.STNode, qualifier st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION)
	enumKeywordToken := b.parseEnumKeyword()
	identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_MODULE_ENUM_NAME)
	openBraceToken := b.parseOpenBrace()
	enumMemberList := b.parseEnumMemberList()
	closeBraceToken := b.parseCloseBrace()
	semicolon := b.parseOptionalSemicolon()
	b.endContext()
	enumDecl := st.CreateEnumDeclarationNode(metadata, qualifier, enumKeywordToken, identifier,
		openBraceToken, enumMemberList, closeBraceToken, semicolon)
	return b.cloneWithDiagnosticIfListEmpty(enumMemberList, enumDecl,
		&common.ERROR_MISSING_ENUM_MEMBER)
}

func (b *ballerinaParser) parseEnumKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ENUM_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ENUM_KEYWORD)
		return b.parseEnumKeyword()
	}
}

func (b *ballerinaParser) parseEnumMemberList() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST)
	if b.peek().Kind() == st.CLOSE_BRACE_TOKEN {
		return st.CreateEmptyNodeList()
	}
	var enumMemberList []st.STNode
	enumMember := b.parseEnumMember()
	var enumMemberRhs st.STNode
	for b.peek().Kind() != st.CLOSE_BRACE_TOKEN {
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
	return st.CreateNodeList(enumMemberList...)
}

func (b *ballerinaParser) parseEnumMember() st.STNode {
	var metadata st.STNode
	switch b.peek().Kind() {
	case st.DOCUMENTATION_STRING, st.AT_TOKEN:
		metadata = b.parseMetaData()
	default:
		metadata = st.CreateEmptyNode()
	}
	identifierNode := b.parseIdentifier(common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME)
	return b.parseEnumMemberRhs(metadata, identifierNode)
}

func (b *ballerinaParser) parseEnumMemberRhs(metadata st.STNode, identifierNode st.STNode) st.STNode {
	var equalToken st.STNode
	var constExprNode st.STNode
	switch b.peek().Kind() {
	case st.EQUAL_TOKEN:
		equalToken = b.parseAssignOp()
		constExprNode = b.parseExpression()
	case st.COMMA_TOKEN, st.CLOSE_BRACE_TOKEN:
		equalToken = st.CreateEmptyNode()
		constExprNode = st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS)
		return b.parseEnumMemberRhs(metadata, identifierNode)
	}
	return st.CreateEnumMemberNode(metadata, identifierNode, equalToken, constExprNode)
}

func (b *ballerinaParser) parseEnumMemberEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END)
		return b.parseEnumMemberEnd()
	}
}

func (b *ballerinaParser) parseTransactionStmtOrVarDecl(annots st.STNode, qualifiers []st.STNode, transactionKeyword st.STToken) (st.STNode, []st.STNode) {
	switch b.peek().Kind() {
	case st.OPEN_BRACE_TOKEN:
		b.reportInvalidStatementAnnots(annots, qualifiers)
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTransactionStatement(transactionKeyword), qualifiers
	case st.COLON_TOKEN:
		if b.getNextNextToken().Kind() == st.IDENTIFIER_TOKEN {
			typeDesc := b.parseQualifiedIdentifierWithPredeclPrefix(transactionKeyword, false)
			return b.parseVarDeclTypeDescRhs(typeDesc, annots, qualifiers, true, false)
		}
		fallthrough
	default:
		solution := b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF)
		if (solution.Action == actionKeep) || ((solution.Action == actionInsert) && (solution.TokenKind == st.COLON_TOKEN)) {
			typeDesc := b.parseQualifiedIdentifierWithPredeclPrefix(transactionKeyword, false)
			return b.parseVarDeclTypeDescRhs(typeDesc, annots, qualifiers, true, false)
		}
		return b.parseTransactionStmtOrVarDecl(annots, qualifiers, transactionKeyword)
	}
}

func (b *ballerinaParser) parseTransactionStatement(transactionKeyword st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_TRANSACTION_STMT)
	blockStmt := b.parseBlockNode()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateTransactionStatementNode(transactionKeyword, blockStmt, onFailClause)
}

func (b *ballerinaParser) parseCommitAction() st.STNode {
	commitKeyword := b.parseCommitKeyword()
	return st.CreateCommitActionNode(commitKeyword)
}

func (b *ballerinaParser) parseCommitKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.COMMIT_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD)
		return b.parseCommitKeyword()
	}
}

func (b *ballerinaParser) parseRetryStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_RETRY_STMT)
	retryKeyword := b.parseRetryKeyword()
	retryStmt := b.parseRetryKeywordRhs(retryKeyword)
	return retryStmt
}

func (b *ballerinaParser) parseRetryKeywordRhs(retryKeyword st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.LT_TOKEN:
		return b.parseRetryTypeParamRhs(retryKeyword, b.parseTypeParameter())
	case st.OPEN_PAREN_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.TRANSACTION_KEYWORD:
		return b.parseRetryTypeParamRhs(retryKeyword, st.CreateEmptyNode())
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS)
		return b.parseRetryKeywordRhs(retryKeyword)
	}
}

func (b *ballerinaParser) parseRetryTypeParamRhs(retryKeyword st.STNode, typeParam st.STNode) st.STNode {
	var args st.STNode
	switch b.peek().Kind() {
	case st.OPEN_PAREN_TOKEN:
		args = b.parseParenthesizedArgList()
	case st.OPEN_BRACE_TOKEN,
		st.TRANSACTION_KEYWORD:
		args = st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS)
		return b.parseRetryTypeParamRhs(retryKeyword, typeParam)
	}
	blockStmt := b.parseRetryBody()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateRetryStatementNode(retryKeyword, typeParam, args, blockStmt, onFailClause)
}

func (b *ballerinaParser) parseRetryBody() st.STNode {
	switch b.peek().Kind() {
	case st.OPEN_BRACE_TOKEN:
		return b.parseBlockNode()
	case st.TRANSACTION_KEYWORD:
		return b.parseTransactionStatement(b.consume())
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_RETRY_BODY)
		return b.parseRetryBody()
	}
}

func (b *ballerinaParser) parseOptionalOnFailClause() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.ON_KEYWORD {
		return b.parseOnFailClause()
	}
	if b.isEndOfRegularCompoundStmt(nextToken.Kind()) {
		return st.CreateEmptyNode()
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS)
	return b.parseOptionalOnFailClause()
}

func (b *ballerinaParser) isEndOfRegularCompoundStmt(nodeKind st.SyntaxKind) bool {
	switch nodeKind {
	case st.CLOSE_BRACE_TOKEN, st.SEMICOLON_TOKEN, st.AT_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return b.isStatementStartingToken(nodeKind)
	}
}

func (b *ballerinaParser) isStatementStartingToken(nodeKind st.SyntaxKind) bool {
	switch nodeKind {
	case st.FINAL_KEYWORD, st.IF_KEYWORD, st.WHILE_KEYWORD, st.DO_KEYWORD,
		st.PANIC_KEYWORD, st.CONTINUE_KEYWORD, st.BREAK_KEYWORD, st.RETURN_KEYWORD,
		st.LOCK_KEYWORD, st.OPEN_BRACE_TOKEN, st.FORK_KEYWORD, st.FOREACH_KEYWORD,
		st.XMLNS_KEYWORD, st.TRANSACTION_KEYWORD, st.RETRY_KEYWORD, st.ROLLBACK_KEYWORD,
		st.MATCH_KEYWORD, st.FAIL_KEYWORD, st.CHECK_KEYWORD, st.CHECKPANIC_KEYWORD,
		st.TRAP_KEYWORD, st.START_KEYWORD, st.FLUSH_KEYWORD, st.LEFT_ARROW_TOKEN,
		st.WAIT_KEYWORD, st.COMMIT_KEYWORD, st.WORKER_KEYWORD, st.TYPE_KEYWORD,
		st.CONST_KEYWORD:
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

func (b *ballerinaParser) parseOnFailClause() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE)
	onKeyword := b.parseOnKeyword()
	failKeyword := b.parseFailKeyword()
	typedBindingPattern := b.parseOnfailOptionalBP()
	blockStatement := b.parseBlockNode()
	b.endContext()
	return st.CreateOnFailClauseNode(onKeyword, failKeyword, typedBindingPattern,
		blockStatement)
}

func (b *ballerinaParser) parseOnfailOptionalBP() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.OPEN_BRACE_TOKEN {
		return st.CreateEmptyNode()
	} else if b.isTypeStartingToken(nextToken.Kind()) {
		return b.parseTypedBindingPattern()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN)
		return b.parseOnfailOptionalBP()
	}
}

func (b *ballerinaParser) parseTypedBindingPattern() st.STNode {
	typeDescriptor := b.parseTypeDescriptorWithoutQualifiers(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true, false, typePrecedenceDefault)
	bindingPattern := b.parseBindingPattern()
	return st.CreateTypedBindingPatternNode(typeDescriptor, bindingPattern)
}

func (b *ballerinaParser) parseRetryKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.RETRY_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_RETRY_KEYWORD)
		return b.parseRetryKeyword()
	}
}

func (b *ballerinaParser) parseRollbackStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ROLLBACK_STMT)
	rollbackKeyword := b.parseRollbackKeyword()
	var expression st.STNode
	if b.peek().Kind() == st.SEMICOLON_TOKEN {
		expression = st.CreateEmptyNode()
	} else {
		expression = b.parseExpression()
	}
	semicolon := b.parseSemicolon()
	b.endContext()
	return st.CreateRollbackStatementNode(rollbackKeyword, expression, semicolon)
}

func (b *ballerinaParser) parseRollbackKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.ROLLBACK_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD)
		return b.parseRollbackKeyword()
	}
}

func (b *ballerinaParser) parseTransactionalExpression() st.STNode {
	transactionalKeyword := b.parseTransactionalKeyword()
	return st.CreateTransactionalExpressionNode(transactionalKeyword)
}

func (b *ballerinaParser) parseTransactionalKeyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.TRANSACTIONAL_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD)
		return b.parseTransactionalKeyword()
	}
}

func (b *ballerinaParser) parseByteArrayLiteral() st.STNode {
	var ty st.STNode
	if b.peek().Kind() == st.BASE16_KEYWORD {
		ty = b.parseBase16Keyword()
	} else {
		ty = b.parseBase64Keyword()
	}
	startingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_START)
	if startingBackTick.IsMissing() {
		startingBackTick = st.CreateMissingToken(st.BACKTICK_TOKEN, nil)
		endingBackTick := st.CreateMissingToken(st.BACKTICK_TOKEN, nil)
		content := st.CreateEmptyNode()
		byteArrayLiteral := st.CreateByteArrayLiteralNode(ty, startingBackTick, content, endingBackTick)
		byteArrayLiteral = st.AddDiagnostic(byteArrayLiteral, &common.ERROR_MISSING_BYTE_ARRAY_CONTENT)
		return byteArrayLiteral
	}
	content := b.parseByteArrayContent()
	return b.parseByteArrayLiteralWithContent(ty, startingBackTick, content)
}

func (b *ballerinaParser) parseByteArrayLiteralWithContent(typeKeyword st.STNode, startingBackTick st.STNode, byteArrayContent st.STNode) st.STNode {
	content := st.CreateEmptyNode()
	newStartingBackTick := startingBackTick
	items, ok := byteArrayContent.(*st.STNodeList)
	if !ok {
		b.internalError("byteArrayContent is not a STNodeList")
		return st.CreateEmptyNode()
	}
	if items.Size() == 1 {
		item := items.Get(0)
		if (typeKeyword.Kind() == st.BASE16_KEYWORD) && (!isValidBase16LiteralContent(st.ToSourceCode(item))) {
			newStartingBackTick = st.CloneWithTrailingInvalidNodeMinutiae(startingBackTick, item,
				&common.ERROR_INVALID_BASE16_CONTENT_IN_BYTE_ARRAY_LITERAL)
		} else if (typeKeyword.Kind() == st.BASE64_KEYWORD) && (!isValidBase64LiteralContent(st.ToSourceCode(item))) {
			newStartingBackTick = st.CloneWithTrailingInvalidNodeMinutiae(startingBackTick, item,
				&common.ERROR_INVALID_BASE64_CONTENT_IN_BYTE_ARRAY_LITERAL)
		} else if item.Kind() != st.TEMPLATE_STRING {
			newStartingBackTick = st.CloneWithTrailingInvalidNodeMinutiae(startingBackTick, item,
				&common.ERROR_INVALID_CONTENT_IN_BYTE_ARRAY_LITERAL)
		} else {
			content = item
		}
	} else if items.Size() > 1 {
		clonedStartingBackTick := startingBackTick
		for index := 0; index < items.Size(); index++ {
			item := items.Get(index)
			clonedStartingBackTick = st.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(clonedStartingBackTick, item)
		}
		newStartingBackTick = st.AddDiagnostic(clonedStartingBackTick,
			&common.ERROR_INVALID_CONTENT_IN_BYTE_ARRAY_LITERAL)
	}
	endingBackTick := b.parseBacktickToken(common.PARSER_RULE_CONTEXT_TEMPLATE_END)
	return st.CreateByteArrayLiteralNode(typeKeyword, newStartingBackTick, content, endingBackTick)
}

func (b *ballerinaParser) parseBase16Keyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.BASE16_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BASE16_KEYWORD)
		return b.parseBase16Keyword()
	}
}

func (b *ballerinaParser) parseBase64Keyword() st.STNode {
	token := b.peek()
	if token.Kind() == st.BASE64_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BASE64_KEYWORD)
		return b.parseBase64Keyword()
	}
}

func (b *ballerinaParser) parseByteArrayContent() st.STNode {
	nextToken := b.peek()
	var items []st.STNode
	for !b.isEndOfBacktickContent(nextToken.Kind()) {
		content := b.parseTemplateItem()
		items = append(items, content)
		nextToken = b.peek()
	}
	return st.CreateNodeList(items...)
}

func (b *ballerinaParser) parseXMLFilterExpression(lhsExpr st.STNode) st.STNode {
	xmlNamePatternChain := b.parseXMLFilterExpressionRhs()
	return st.CreateXMLFilterExpressionNode(lhsExpr, xmlNamePatternChain)
}

func (b *ballerinaParser) parseXMLFilterExpressionRhs() st.STNode {
	dotLTToken := b.parseDotLTToken()
	return b.parseXMLNamePatternChain(dotLTToken)
}

func (b *ballerinaParser) parseXMLNamePatternChain(startToken st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN)
	xmlNamePattern := b.parseXMLNamePattern()
	gtToken := b.parseGTToken()
	b.endContext()
	startToken = b.cloneWithDiagnosticIfListEmpty(xmlNamePattern, startToken,
		&common.ERROR_MISSING_XML_ATOMIC_NAME_PATTERN)
	return st.CreateXMLNamePatternChainingNode(startToken, xmlNamePattern, gtToken)
}

func (b *ballerinaParser) parseXMLStepExtends() st.STNode {
	nextToken := b.peek()
	if b.isEndOfXMLStepExtend(nextToken.Kind()) {
		return st.CreateEmptyNodeList()
	}
	var xmlStepExtendList []st.STNode
	b.startContext(common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS)
	var stepExtension st.STNode
	for !b.isEndOfXMLStepExtend(nextToken.Kind()) {
		if nextToken.Kind() == st.DOT_TOKEN {
			stepExtension = b.parseXMLStepMethodCallExtend()
		} else if nextToken.Kind() == st.DOT_LT_TOKEN {
			stepExtension = b.parseXMLFilterExpressionRhs()
		} else {
			stepExtension = b.parseXMLIndexedStepExtend()
		}
		xmlStepExtendList = append(xmlStepExtendList, stepExtension)
		nextToken = b.peek()
	}
	b.endContext()
	return st.CreateNodeList(xmlStepExtendList...)
}

func (b *ballerinaParser) parseXMLIndexedStepExtend() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR)
	openBracket := b.parseOpenBracket()
	keyExpr := b.parseKeyExpr(true)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return st.CreateXMLStepIndexedExtendNode(openBracket, keyExpr, closeBracket)
}

func (b *ballerinaParser) parseXMLStepMethodCallExtend() st.STNode {
	dotToken := b.parseDotToken()
	methodName := b.parseMethodName()
	parenthesizedArgsList := b.parseParenthesizedArgList()
	return st.CreateXMLStepMethodCallExtendNode(dotToken, methodName, parenthesizedArgsList)
}

func (b *ballerinaParser) parseMethodName() st.STNode {
	if b.isSpecialMethodName(b.peek()) {
		return b.getKeywordAsSimpleNameRef()
	}
	return st.CreateSimpleNameReferenceNode(b.parseIdentifier(common.PARSER_RULE_CONTEXT_IDENTIFIER))
}

func (b *ballerinaParser) parseDotLTToken() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.DOT_LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN)
		return b.parseDotLTToken()
	}
}

func (b *ballerinaParser) parseXMLNamePattern() st.STNode {
	var xmlAtomicNamePatternList []st.STNode
	nextToken := b.peek()
	if b.isEndOfXMLNamePattern(nextToken.Kind()) {
		return st.CreateNodeList(xmlAtomicNamePatternList...)
	}
	xmlAtomicNamePattern := b.parseXMLAtomicNamePattern()
	xmlAtomicNamePatternList = append(xmlAtomicNamePatternList, xmlAtomicNamePattern)
	var separator st.STNode
	for !b.isEndOfXMLNamePattern(b.peek().Kind()) {
		separator = b.parseXMLNamePatternSeparator()
		if separator == nil {
			break
		}
		xmlAtomicNamePatternList = append(xmlAtomicNamePatternList, separator)
		xmlAtomicNamePattern = b.parseXMLAtomicNamePattern()
		xmlAtomicNamePatternList = append(xmlAtomicNamePatternList, xmlAtomicNamePattern)
	}
	return st.CreateNodeList(xmlAtomicNamePatternList...)
}

func (b *ballerinaParser) isEndOfXMLNamePattern(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.GT_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) isEndOfXMLStepExtend(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.OPEN_BRACKET_TOKEN, st.DOT_LT_TOKEN:
		return false
	case st.DOT_TOKEN:
		return b.peekN(3).Kind() != st.OPEN_PAREN_TOKEN
	default:
		return true
	}
}

func (b *ballerinaParser) parseXMLNamePatternSeparator() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.PIPE_TOKEN:
		return b.consume()
	case st.GT_TOKEN, st.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS)
		return b.parseXMLNamePatternSeparator()
	}
}

func (b *ballerinaParser) parseXMLAtomicNamePattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN)
	atomicNamePattern := b.parseXMLAtomicNamePatternBody()
	b.endContext()
	return atomicNamePattern
}

func (b *ballerinaParser) parseXMLAtomicNamePatternBody() st.STNode {
	token := b.peek()
	var identifier st.STNode
	switch token.Kind() {
	case st.ASTERISK_TOKEN:
		return b.consume()
	case st.IDENTIFIER_TOKEN:
		identifier = b.consume()
	default:
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START)
		return b.parseXMLAtomicNamePatternBody()
	}
	return b.parseXMLAtomicNameIdentifier(identifier)
}

func (b *ballerinaParser) parseXMLAtomicNameIdentifier(identifier st.STNode) st.STNode {
	token := b.peek()
	if token.Kind() == st.COLON_TOKEN {
		colon := b.consume()
		nextToken := b.peek()
		if (nextToken.Kind() == st.IDENTIFIER_TOKEN) || (nextToken.Kind() == st.ASTERISK_TOKEN) {
			endToken := b.consume()
			return st.CreateXMLAtomicNamePatternNode(identifier, colon, endToken)
		}
	}
	return st.CreateSimpleNameReferenceNode(identifier)
}

func (b *ballerinaParser) parseXMLStepExpression(lhsExpr st.STNode) st.STNode {
	xmlStepStart := b.parseXMLStepStart()
	xmlStepExtends := b.parseXMLStepExtends()
	return st.CreateXMLStepExpressionNode(lhsExpr, xmlStepStart, xmlStepExtends)
}

func (b *ballerinaParser) parseXMLStepStart() st.STNode {
	token := b.peek()
	var startToken st.STNode
	switch token.Kind() {
	case st.SLASH_ASTERISK_TOKEN:
		return b.consume()
	case st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
		startToken = b.parseDoubleSlashDoubleAsteriskLTToken()
	case st.SLASH_LT_TOKEN:
	default:
		startToken = b.parseSlashLTToken()
	}
	return b.parseXMLNamePatternChain(startToken)
}

func (b *ballerinaParser) parseSlashLTToken() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.SLASH_LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN)
		return b.parseSlashLTToken()
	}
}

func (b *ballerinaParser) parseDoubleSlashDoubleAsteriskLTToken() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN)
		return b.parseDoubleSlashDoubleAsteriskLTToken()
	}
}

func (b *ballerinaParser) parseMatchStatement() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MATCH_STMT)
	matchKeyword := b.parseMatchKeyword()
	actionOrExpr := b.parseActionOrExpression()
	b.startContext(common.PARSER_RULE_CONTEXT_MATCH_BODY)
	openBrace := b.parseOpenBrace()
	var matchClausesList []st.STNode
	for !b.isEndOfMatchClauses(b.peek().Kind()) {
		clause := b.parseMatchClause()
		matchClausesList = append(matchClausesList, clause)
	}
	matchClauses := st.CreateNodeList(matchClausesList...)
	if b.isNodeListEmpty(matchClauses) {
		openBrace = st.AddDiagnostic(openBrace,
			&common.ERROR_MATCH_STATEMENT_SHOULD_HAVE_ONE_OR_MORE_MATCH_CLAUSES)
	}
	closeBrace := b.parseCloseBrace()
	b.endContext()
	b.endContext()
	onFailClause := b.parseOptionalOnFailClause()
	return st.CreateMatchStatementNode(matchKeyword, actionOrExpr, openBrace, matchClauses, closeBrace,
		onFailClause)
}

func (b *ballerinaParser) parseMatchKeyword() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.MATCH_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MATCH_KEYWORD)
		return b.parseMatchKeyword()
	}
}

func (b *ballerinaParser) isEndOfMatchClauses(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACE_TOKEN, st.TYPE_KEYWORD:
		return true
	default:
		return b.isEndOfStatements()
	}
}

func (b *ballerinaParser) parseMatchClause() st.STNode {
	matchPatterns := b.parseMatchPatternList()
	matchGuard := b.parseMatchGuard()
	rightDoubleArrow := b.parseDoubleRightArrow()
	blockStmt := b.parseBlockNode()
	if b.isNodeListEmpty(matchPatterns) {
		identifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		constantPattern := st.CreateSimpleNameReferenceNode(identifier)
		matchPatterns = st.CreateNodeList(constantPattern)
		errorCode := &common.ERROR_MISSING_MATCH_PATTERN
		if matchGuard != nil {
			matchGuard = st.AddDiagnostic(matchGuard, errorCode)
		} else {
			rightDoubleArrow = st.AddDiagnostic(rightDoubleArrow, errorCode)
		}
	}
	return st.CreateMatchClauseNode(matchPatterns, matchGuard, rightDoubleArrow, blockStmt)
}

func (b *ballerinaParser) parseMatchGuard() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IF_KEYWORD:
		ifKeyword := b.parseIfKeyword()
		expr := b.parseExpressionWithMatchGuard(defaultOpPrecedence, true, false, true, false)
		return st.CreateMatchGuardNode(ifKeyword, expr)
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		return st.CreateEmptyNode()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD)
		return b.parseMatchGuard()
	}
}

func (b *ballerinaParser) parseMatchPatternList() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MATCH_PATTERN)
	var matchClauses []st.STNode
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
	return st.CreateNodeList(matchClauses...)
}

func (b *ballerinaParser) isEndOfMatchPattern(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.PIPE_TOKEN, st.IF_KEYWORD, st.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseMatchPattern() st.STNode {
	nextToken := b.peek()
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		typeRefOrConstExpr := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_MATCH_PATTERN)
		return b.parseErrorMatchPatternOrConsPattern(typeRefOrConstExpr)
	}
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN:
		return b.parseSimpleConstExpr()
	case st.VAR_KEYWORD:
		return b.parseVarTypedBindingPattern()
	case st.OPEN_BRACKET_TOKEN:
		return b.parseListMatchPattern()
	case st.OPEN_BRACE_TOKEN:
		return b.parseMappingMatchPattern()
	case st.ERROR_KEYWORD:
		return b.parseErrorMatchPattern()
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START)
		return b.parseMatchPattern()
	}
}

func (b *ballerinaParser) parseMatchPatternListMemberRhs() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.PIPE_TOKEN:
		return b.parsePipeToken()
	case st.IF_KEYWORD, st.RIGHT_DOUBLE_ARROW_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS)
		return b.parseMatchPatternListMemberRhs()
	}
}

func (b *ballerinaParser) parseVarTypedBindingPattern() st.STNode {
	varKeyword := b.parseVarKeyword()
	varTypeDesc := b.createBuiltinSimpleNameReference(varKeyword)
	bindingPattern := b.parseBindingPattern()
	return st.CreateTypedBindingPatternNode(varTypeDesc, bindingPattern)
}

func (b *ballerinaParser) parseVarKeyword() st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.VAR_KEYWORD {
		return b.consume()
	} else {
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_VAR_KEYWORD)
		return b.parseVarKeyword()
	}
}

func (b *ballerinaParser) parseListMatchPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN)
	openBracketToken := b.parseOpenBracket()
	var matchPatternList []st.STNode
	var listMatchPatternMemberRhs st.STNode
	isEndOfFields := false
	for !b.IsEndOfListMatchPattern() {
		listMatchPatternMember := b.parseListMatchPatternMember()
		matchPatternList = append(matchPatternList, listMatchPatternMember)
		listMatchPatternMemberRhs = b.parseListMatchPatternMemberRhs()
		if listMatchPatternMember.Kind() == st.REST_MATCH_PATTERN {
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
		if b.peek().Kind() == st.CLOSE_BRACKET_TOKEN {
			break
		}
		invalidField := b.parseListMatchPatternMember()
		b.updateLastNodeInListWithInvalidNode(matchPatternList, invalidField,
			&common.ERROR_MATCH_PATTERN_AFTER_REST_MATCH_PATTERN)
		listMatchPatternMemberRhs = b.parseListMatchPatternMemberRhs()
	}
	matchPatternListNode := st.CreateNodeList(matchPatternList...)
	closeBracketToken := b.parseCloseBracket()
	b.endContext()
	return st.CreateListMatchPatternNode(openBracketToken, matchPatternListNode, closeBracketToken)
}

func (b *ballerinaParser) IsEndOfListMatchPattern() bool {
	switch b.peek().Kind() {
	case st.CLOSE_BRACKET_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseListMatchPatternMember() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.ELLIPSIS_TOKEN:
		return b.parseRestMatchPattern()
	default:
		return b.parseMatchPattern()
	}
}

func (b *ballerinaParser) parseRestMatchPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN)
	ellipsisToken := b.parseEllipsis()
	varKeywordToken := b.parseVarKeyword()
	variableName := b.parseVariableName()
	b.endContext()
	simpleNameReferenceNode, ok := st.CreateSimpleNameReferenceNode(variableName).(*st.STSimpleNameReferenceNode)
	if !ok {
		b.internalError("expected STSimpleNameReferenceNode")
		return st.CreateEmptyNode()
	}
	return st.CreateRestMatchPatternNode(ellipsisToken, varKeywordToken, simpleNameReferenceNode)
}

func (b *ballerinaParser) parseListMatchPatternMemberRhs() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACKET_TOKEN, st.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS)
		return b.parseListMatchPatternMemberRhs()
	}
}

func (b *ballerinaParser) parseMappingMatchPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN)
	openBraceToken := b.parseOpenBrace()
	fieldMatchPatterns := b.parseFieldMatchPatternList()
	closeBraceToken := b.parseCloseBrace()
	b.endContext()
	return st.CreateMappingMatchPatternNode(openBraceToken, fieldMatchPatterns, closeBraceToken)
}

func (b *ballerinaParser) parseFieldMatchPatternList() st.STNode {
	var fieldMatchPatterns []st.STNode
	fieldMatchPatternMember := b.parseFieldMatchPatternMember()
	if fieldMatchPatternMember == nil {
		return st.CreateEmptyNodeList()
	}
	fieldMatchPatterns = append(fieldMatchPatterns, fieldMatchPatternMember)
	if fieldMatchPatternMember.Kind() == st.REST_MATCH_PATTERN {
		b.invalidateExtraFieldMatchPatterns(fieldMatchPatterns)
		return st.CreateNodeList(fieldMatchPatterns...)
	}
	return b.parseFieldMatchPatternListWithPatterns(fieldMatchPatterns)
}

func (b *ballerinaParser) parseFieldMatchPatternListWithPatterns(fieldMatchPatterns []st.STNode) st.STNode {
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
		if fieldMatchPatternMember.Kind() == st.REST_MATCH_PATTERN {
			b.invalidateExtraFieldMatchPatterns(fieldMatchPatterns)
			break
		}
	}
	return st.CreateNodeList(fieldMatchPatterns...)
}

func (b *ballerinaParser) createMissingFieldMatchPattern() st.STNode {
	fieldName := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
	colon := st.CreateMissingToken(st.COLON_TOKEN, nil)
	identifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
	matchPattern := st.CreateSimpleNameReferenceNode(identifier)
	fieldMatchPatternMember := st.CreateFieldMatchPatternNode(fieldName, colon, matchPattern)
	fieldMatchPatternMember = st.AddDiagnostic(fieldMatchPatternMember,
		&common.ERROR_MISSING_FIELD_MATCH_PATTERN_MEMBER)
	return fieldMatchPatternMember
}

func (b *ballerinaParser) invalidateExtraFieldMatchPatterns(fieldMatchPatterns []st.STNode) {
	for !b.IsEndOfMappingMatchPattern() {
		fieldMatchPatternRhs := b.parseFieldMatchPatternRhs()
		if fieldMatchPatternRhs == nil {
			break
		}
		fieldMatchPatternMember := b.parseFieldMatchPatternMember()
		if fieldMatchPatternMember == nil {
			rhsToken, ok := fieldMatchPatternRhs.(st.STToken)
			if !ok {
				b.internalError("invalidateExtraFieldMatchPatterns: expected STToken")
				return
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

func (b *ballerinaParser) parseFieldMatchPatternMember() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		return b.ParseFieldMatchPattern()
	case st.ELLIPSIS_TOKEN:
		return b.parseRestMatchPattern()
	case st.CLOSE_BRACE_TOKEN, st.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START)
		return b.parseFieldMatchPatternMember()
	}
}

func (b *ballerinaParser) ParseFieldMatchPattern() st.STNode {
	fieldNameNode := b.parseVariableName()
	colonToken := b.parseColon()
	matchPattern := b.parseMatchPattern()
	return st.CreateFieldMatchPatternNode(fieldNameNode, colonToken, matchPattern)
}

func (b *ballerinaParser) IsEndOfMappingMatchPattern() bool {
	switch b.peek().Kind() {
	case st.CLOSE_BRACE_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseFieldMatchPatternRhs() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACE_TOKEN, st.EOF_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS)
		return b.parseFieldMatchPatternRhs()
	}
}

func (b *ballerinaParser) parseErrorMatchPatternOrConsPattern(typeRefOrConstExpr st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		error := st.CreateMissingTokenWithDiagnostics(st.ERROR_KEYWORD,
			common.PARSER_RULE_CONTEXT_ERROR_KEYWORD.GetErrorCode())
		b.startContext(common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN)
		return b.parseErrorMatchPatternWithErrorKeywordAndTypeRef(error, typeRefOrConstExpr)
	default:
		if b.isMatchPatternEnd(b.peek().Kind()) {
			return typeRefOrConstExpr
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN)
		return b.parseErrorMatchPatternOrConsPattern(typeRefOrConstExpr)
	}
}

func (b *ballerinaParser) isMatchPatternEnd(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.RIGHT_DOUBLE_ARROW_TOKEN,
		st.COMMA_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.CLOSE_BRACKET_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.PIPE_TOKEN,
		st.IF_KEYWORD,
		st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseErrorMatchPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN)
	error := b.consume()
	return b.parseErrorMatchPatternWithErrorKeyword(error)
}

func (b *ballerinaParser) parseErrorMatchPatternWithErrorKeyword(error st.STNode) st.STNode {
	nextToken := b.peek()
	var typeRef st.STNode
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		typeRef = st.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			typeRef = b.parseTypeReference()
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS)
		return b.parseErrorMatchPatternWithErrorKeyword(error)
	}
	return b.parseErrorMatchPatternWithErrorKeywordAndTypeRef(error, typeRef)
}

func (b *ballerinaParser) parseErrorMatchPatternWithErrorKeywordAndTypeRef(error st.STNode, typeRef st.STNode) st.STNode {
	openParenthesisToken := b.parseOpenParenthesis()
	argListMatchPatternNode := b.parseErrorArgListMatchPatterns()
	closeParenthesisToken := b.parseCloseParenthesis()
	b.endContext()
	return st.CreateErrorMatchPatternNode(error, typeRef, openParenthesisToken,
		argListMatchPatternNode, closeParenthesisToken)
}

func (b *ballerinaParser) parseErrorArgListMatchPatterns() st.STNode {
	var argListMatchPatterns []st.STNode
	if b.isEndOfErrorFieldMatchPatterns() {
		return st.CreateNodeList(argListMatchPatterns...)
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
		if (firstArg.Kind() != st.NAMED_ARG_MATCH_PATTERN) && (firstArg.Kind() != st.REST_MATCH_PATTERN) {
			b.addInvalidNodeToNextToken(firstArg, &common.ERROR_MATCH_PATTERN_NOT_ALLOWED)
		} else {
			argListMatchPatterns = append(argListMatchPatterns, firstArg)
		}
	}
	argListMatchPatterns = b.parseErrorFieldMatchPatterns(argListMatchPatterns)
	return st.CreateNodeList(argListMatchPatterns...)
}

func (b *ballerinaParser) isSimpleMatchPattern(matchPatternKind st.SyntaxKind) bool {
	switch matchPatternKind {
	case st.IDENTIFIER_TOKEN,
		st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE,
		st.NUMERIC_LITERAL,
		st.STRING_LITERAL,
		st.NULL_LITERAL,
		st.NIL_LITERAL,
		st.BOOLEAN_LITERAL,
		st.TYPED_BINDING_PATTERN,
		st.UNARY_EXPRESSION:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) isValidSecondArgMatchPattern(syntaxKind st.SyntaxKind) bool {
	switch syntaxKind {
	case st.ERROR_MATCH_PATTERN,
		st.NAMED_ARG_MATCH_PATTERN,
		st.REST_MATCH_PATTERN:
		return true
	default:
		return b.isSimpleMatchPattern(syntaxKind)
	}
}

// Return modified argListMatchPatterns
func (b *ballerinaParser) parseErrorFieldMatchPatterns(argListMatchPatterns []st.STNode) []st.STNode {
	lastValidArgKind := st.NAMED_ARG_MATCH_PATTERN
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

func (b *ballerinaParser) isEndOfErrorFieldMatchPatterns() bool {
	return b.isEndOfErrorFieldBindingPatterns()
}

func (b *ballerinaParser) parseErrorArgListMatchPatternEnd(currentCtx common.ParserRuleContext) st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.consume()
	case st.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), currentCtx)
		return b.parseErrorArgListMatchPatternEnd(currentCtx)
	}
}

func (b *ballerinaParser) parseErrorArgListMatchPattern(context common.ParserRuleContext) st.STNode {
	nextToken := b.peek()
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		return b.parseNamedArgOrSimpleMatchPattern()
	}
	switch nextToken.Kind() {
	case st.ELLIPSIS_TOKEN:
		return b.parseRestMatchPattern()
	case st.OPEN_PAREN_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.OPEN_BRACKET_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.ERROR_KEYWORD:
		return b.parseMatchPattern()
	case st.VAR_KEYWORD:
		varType := b.createBuiltinSimpleNameReference(b.consume())
		variableName := b.createCaptureOrWildcardBP(b.parseVariableName())
		return st.CreateTypedBindingPatternNode(varType, variableName)
	case st.CLOSE_PAREN_TOKEN:
		return st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_MATCH_PATTERN)
	default:
		b.recoverWithBlockContext(nextToken, context)
		return b.parseErrorArgListMatchPattern(context)
	}
}

func (b *ballerinaParser) parseNamedArgOrSimpleMatchPattern() st.STNode {
	constRefExpr := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_MATCH_PATTERN)
	if (constRefExpr.Kind() == st.QUALIFIED_NAME_REFERENCE) || (b.peek().Kind() != st.EQUAL_TOKEN) {
		return constRefExpr
	}
	simpleNameNode, ok := constRefExpr.(*st.STSimpleNameReferenceNode)
	if !ok {
		b.internalError("parseNamedArgOrSimpleMatchPattern: expected STSimpleNameReferenceNode")
		return st.CreateEmptyNode()
	}
	return b.parseNamedArgMatchPattern(simpleNameNode.Name)
}

func (b *ballerinaParser) parseNamedArgMatchPattern(identifier st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN)
	equalToken := b.parseAssignOp()
	matchPattern := b.parseMatchPattern()
	b.endContext()
	return st.CreateNamedArgMatchPatternNode(identifier, equalToken, matchPattern)
}

func (b *ballerinaParser) validateErrorFieldMatchPatternOrder(prevArgKind st.SyntaxKind, currentArgKind st.SyntaxKind) *common.DiagnosticErrorCode {
	switch currentArgKind {
	case st.NAMED_ARG_MATCH_PATTERN,
		st.REST_MATCH_PATTERN:
		if prevArgKind == st.REST_MATCH_PATTERN {
			return &common.ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG
		}
		return nil
	default:
		return &common.ERROR_MATCH_PATTERN_NOT_ALLOWED
	}
}

func (b *ballerinaParser) parseMarkdownDocumentation() st.STNode {
	markdownDocLineList := make([]st.STNode, 0)
	nextToken := b.peek()
	for nextToken.Kind() == st.DOCUMENTATION_STRING {
		documentationString := b.consume()
		parsedDocLines := b.parseDocumentationString(documentationString)
		markdownDocLineList = b.appendParsedDocumentationLines(markdownDocLineList, parsedDocLines)
		nextToken = b.peek()
	}
	markdownDocLines := st.CreateNodeList(markdownDocLineList...)
	return st.CreateMarkdownDocumentationNode(markdownDocLines)
}

func (b *ballerinaParser) parseDocumentationString(documentationStringToken st.STToken) st.STNode {
	leadingTriviaList := b.getLeadingTriviaList(documentationStringToken.LeadingMinutiae())
	diagnostics := documentationStringToken.Diagnostics()

	charReader := text.CharReaderFromText(documentationStringToken.Text())
	documentationLexer := newDocumentationLexer(b.tokenReader.ctx, b.tokenReader.fileName, charReader, leadingTriviaList, diagnostics)
	tokenReader := createTokenReader(b.tokenReader.ctx, b.tokenReader.fileName, documentationLexer)
	documentationParser := newDocumentationParser(tokenReader)

	return documentationParser.Parse()
}

func (b *ballerinaParser) getLeadingTriviaList(leadingMinutiaeNode st.STNode) []st.STNode {
	leadingTriviaList := make([]st.STNode, 0)
	bucketCount := leadingMinutiaeNode.BucketCount()
	i := 0
	for ; i < bucketCount; i++ {
		leadingTriviaList = append(leadingTriviaList, leadingMinutiaeNode.ChildInBucket(i))
	}
	return leadingTriviaList
}

func (b *ballerinaParser) appendParsedDocumentationLines(markdownDocLineList []st.STNode, parsedDocLines st.STNode) []st.STNode {
	bucketCount := parsedDocLines.BucketCount()
	for i := range bucketCount {
		markdownDocLine := parsedDocLines.ChildInBucket(i)
		markdownDocLineList = append(markdownDocLineList, markdownDocLine)
	}
	return markdownDocLineList
}

func (b *ballerinaParser) parseStmtStartsWithTypeOrExpr(annots st.STNode, qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	typeOrExpr := b.parseTypedBindingPatternOrExprWithQualifiers(qualifiers, true)
	return b.parseStmtStartsWithTypedBPOrExprRhs(annots, typeOrExpr)
}

func (b *ballerinaParser) parseStmtStartsWithTypedBPOrExprRhs(annots st.STNode, typedBindingPatternOrExpr st.STNode) st.STNode {
	if typedBindingPatternOrExpr.Kind() == st.TYPED_BINDING_PATTERN {
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		res, _ := b.parseVarDeclRhs(annots, typedBindingPatternOrExpr)
		return res
	}
	expr := b.getExpression(typedBindingPatternOrExpr)
	expr = b.getExpression(b.parseExpressionRhs(expr, false, true))
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *ballerinaParser) parseTypedBindingPatternOrExpr(allowAssignment bool) st.STNode {
	typeDescQualifiers := make([]st.STNode, 0)
	return b.parseTypedBindingPatternOrExprWithQualifiers(typeDescQualifiers, allowAssignment)
}

func (b *ballerinaParser) parseTypedBindingPatternOrExprWithQualifiers(qualifiers []st.STNode, allowAssignment bool) st.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	var typeOrExpr st.STNode
	if b.isPredeclaredIdentifier(nextToken.Kind()) {
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME)
		return b.parseTypedBindingPatternOrExprRhs(typeOrExpr, allowAssignment)
	}
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseTypedBPOrExprStartsWithOpenParenthesis()
	case st.FUNCTION_KEYWORD:
		return b.parseAnonFuncExprOrTypedBPWithFuncType(qualifiers)
	case st.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseTupleTypeDescOrListConstructor(st.CreateEmptyNodeList())
		return b.parseTypedBindingPatternOrExprRhs(typeOrExpr, allowAssignment)
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		basicLiteral := b.parseBasicLiteral()
		return b.parseTypedBindingPatternOrExprRhs(basicLiteral, allowAssignment)
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseActionOrExpressionInLhs(st.CreateEmptyNodeList())
		}
		return b.parseTypedBindingPatternInner(qualifiers, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	}
}

func (b *ballerinaParser) parseTypedBindingPatternOrExprRhs(typeOrExpr st.STNode, allowAssignment bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.PIPE_TOKEN, st.BITWISE_AND_TOKEN:
		nextNextToken := b.peekN(2)
		if nextNextToken.Kind() == st.EQUAL_TOKEN {
			return typeOrExpr
		}
		pipeOrAndToken := b.parseBinaryOperator()
		rhsTypedBPOrExpr := b.parseTypedBindingPatternOrExpr(allowAssignment)
		if rhsTypedBPOrExpr.Kind() == st.TYPED_BINDING_PATTERN {
			typedBP, ok := rhsTypedBPOrExpr.(*st.STTypedBindingPatternNode)
			if !ok {
				b.internalError("expected STTypedBindingPatternNode")
				return st.CreateEmptyNode()
			}
			typeOrExpr = b.getTypeDescFromExpr(typeOrExpr)
			newTypeDesc := b.mergeTypes(typeOrExpr, pipeOrAndToken, typedBP.TypeDescriptor)
			return st.CreateTypedBindingPatternNode(newTypeDesc, typedBP.BindingPattern)
		}
		if b.peek().Kind() == st.EQUAL_TOKEN {
			return b.createCaptureBPWithMissingVarName(typeOrExpr, pipeOrAndToken, rhsTypedBPOrExpr)
		}
		return st.CreateBinaryExpressionNode(st.BINARY_EXPRESSION, typeOrExpr,
			pipeOrAndToken, rhsTypedBPOrExpr)
	case st.SEMICOLON_TOKEN:
		if b.isExpression(typeOrExpr.Kind()) {
			return typeOrExpr
		}
		if b.isDefiniteTypeDesc(typeOrExpr.Kind()) || (!b.isAllBasicLiterals(typeOrExpr)) {
			typeDesc := b.getTypeDescFromExpr(typeOrExpr)
			return b.parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc)
		}
		return typeOrExpr
	case st.IDENTIFIER_TOKEN, st.QUESTION_MARK_TOKEN:
		if b.isAmbiguous(typeOrExpr) || b.isDefiniteTypeDesc(typeOrExpr.Kind()) {
			typeDesc := b.getTypeDescFromExpr(typeOrExpr)
			return b.parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc)
		}
		return typeOrExpr
	case st.EQUAL_TOKEN:
		return typeOrExpr
	case st.OPEN_BRACKET_TOKEN:
		return b.parseTypedBindingPatternOrMemberAccess(typeOrExpr, false, allowAssignment,
			common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	case st.OPEN_BRACE_TOKEN, st.ERROR_KEYWORD:
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
		if (typeOrExprKind == st.QUALIFIED_NAME_REFERENCE) || (typeOrExprKind == st.SIMPLE_NAME_REFERENCE) {
			b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS)
		} else {
			b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS)
		}
		return b.parseTypedBindingPatternOrExprRhs(typeOrExpr, allowAssignment)
	}
}

func (b *ballerinaParser) createCaptureBPWithMissingVarName(lhsType st.STNode, separatorToken st.STNode, rhsType st.STNode) st.STNode {
	lhsType = b.getTypeDescFromExpr(lhsType)
	rhsType = b.getTypeDescFromExpr(rhsType)
	newTypeDesc := b.mergeTypes(lhsType, separatorToken, rhsType)
	identifier := createMissingTokenWithDiagnosticsFromParserRules(st.IDENTIFIER_TOKEN,
		common.PARSER_RULE_CONTEXT_VARIABLE_NAME)
	captureBP := st.CreateCaptureBindingPatternNode(identifier)
	return st.CreateTypedBindingPatternNode(newTypeDesc, captureBP)
}

func (b *ballerinaParser) parseTypeBindingPatternStartsWithAmbiguousNode(typeDesc st.STNode) st.STNode {
	typeDesc = b.parseComplexTypeDescriptor(typeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	return b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
}

func (b *ballerinaParser) parseTypedBPOrExprStartsWithOpenParenthesis() st.STNode {
	exprOrTypeDesc := b.parseTypedDescOrExprStartsWithOpenParenthesis()
	if b.isDefiniteTypeDesc(exprOrTypeDesc.Kind()) {
		return b.parseTypeBindingPatternStartsWithAmbiguousNode(exprOrTypeDesc)
	}
	return b.parseTypedBindingPatternOrExprRhs(exprOrTypeDesc, false)
}

func (b *ballerinaParser) isDefiniteTypeDesc(kind st.SyntaxKind) bool {
	return ((kind.CompareTo(st.RECORD_TYPE_DESC) >= 0) && (kind.CompareTo(st.FUTURE_TYPE_DESC) <= 0))
}

func (b *ballerinaParser) isDefiniteExpr(kind st.SyntaxKind) bool {
	if (kind == st.QUALIFIED_NAME_REFERENCE) || (kind == st.SIMPLE_NAME_REFERENCE) {
		return false
	}
	return ((kind.CompareTo(st.BINARY_EXPRESSION) >= 0) && (kind.CompareTo(st.ERROR_CONSTRUCTOR) <= 0))
}

func (b *ballerinaParser) isDefiniteAction(kind st.SyntaxKind) bool {
	return ((kind.CompareTo(st.REMOTE_METHOD_CALL_ACTION) >= 0) && (kind.CompareTo(st.CLIENT_RESOURCE_ACCESS_ACTION) <= 0))
}

func (b *ballerinaParser) parseTypedDescOrExprStartsWithOpenParenthesis() st.STNode {
	openParen := b.parseOpenParenthesis()
	nextToken := b.peek()
	if nextToken.Kind() == st.CLOSE_PAREN_TOKEN {
		closeParen := b.parseCloseParenthesis()
		return b.parseTypeOrExprStartWithEmptyParenthesis(openParen, closeParen)
	}
	typeOrExpr := b.parseTypeDescOrExpr()
	if b.isAction(typeOrExpr) {
		closeParen := b.parseCloseParenthesis()
		return st.CreateBracedExpressionNode(st.BRACED_ACTION, openParen, typeOrExpr,
			closeParen)
	}
	if b.isExpression(typeOrExpr.Kind()) {
		b.startContext(common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS)
		return b.parseBracedExprOrAnonFuncParamRhs(openParen, typeOrExpr, false)
	}
	typeDescNode := b.getTypeDescFromExpr(typeOrExpr)
	typeDescNode = b.parseComplexTypeDescriptor(typeDescNode, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS, false)
	closeParen := b.parseCloseParenthesis()
	return st.CreateParenthesisedTypeDescriptorNode(openParen, typeDescNode, closeParen)
}

func (b *ballerinaParser) parseTypeDescOrExpr() st.STNode {
	return b.parseTypeDescOrExprWithQualifiers(nil)
}

func (b *ballerinaParser) parseTypeDescOrExprWithQualifiers(qualifiers []st.STNode) st.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	var typeOrExpr st.STNode
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseTypedDescOrExprStartsWithOpenParenthesis()
	case st.FUNCTION_KEYWORD:
		typeOrExpr = b.parseAnonFuncExprOrFuncTypeDesc(qualifiers)
	case st.IDENTIFIER_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME)
		return b.parseTypeDescOrExprRhs(typeOrExpr)
	case st.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		typeOrExpr = b.parseTupleTypeDescOrListConstructor(st.CreateEmptyNodeList())
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.NULL_KEYWORD,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		basicLiteral := b.parseBasicLiteral()
		return b.parseTypeDescOrExprRhs(basicLiteral)
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseActionOrExpressionInLhs(st.CreateEmptyNodeList())
		}
		return b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
	}
	if b.isDefiniteTypeDesc(typeOrExpr.Kind()) {
		return b.parseComplexTypeDescriptor(typeOrExpr, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	}
	return b.parseTypeDescOrExprRhs(typeOrExpr)
}

func (b *ballerinaParser) isExpression(kind st.SyntaxKind) bool {
	switch kind {
	case st.NUMERIC_LITERAL,
		st.STRING_LITERAL_TOKEN,
		st.NIL_LITERAL,
		st.NULL_LITERAL,
		st.BOOLEAN_LITERAL:
		return true
	default:
		return ((kind.CompareTo(st.BINARY_EXPRESSION) >= 0) && (kind.CompareTo(st.ERROR_CONSTRUCTOR) <= 0))
	}
}

func (b *ballerinaParser) parseTypeOrExprStartWithEmptyParenthesis(openParen st.STNode, closeParen st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.RIGHT_DOUBLE_ARROW_TOKEN:
		params := st.CreateEmptyNodeList()
		anonFuncParam := st.CreateImplicitAnonymousFunctionParameters(openParen, params, closeParen)
		return b.parseImplicitAnonFuncWithParams(anonFuncParam, false)
	default:
		return st.CreateNilLiteralNode(openParen, closeParen)
	}
}

func (b *ballerinaParser) parseAnonFuncExprOrTypedBPWithFuncType(qualifiers []st.STNode) st.STNode {
	exprOrTypeDesc := b.parseAnonFuncExprOrFuncTypeDesc(qualifiers)
	if b.isAction(exprOrTypeDesc) || b.isExpression(exprOrTypeDesc.Kind()) {
		return exprOrTypeDesc
	}
	return b.parseTypedBindingPatternTypeRhs(exprOrTypeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
}

func (b *ballerinaParser) parseAnonFuncExprOrFuncTypeDesc(qualifiers []st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC)
	var qualifierList st.STNode
	functionKeyword := b.parseFunctionKeyword()
	var funcSignature st.STNode
	if b.peek().Kind() == st.OPEN_PAREN_TOKEN {
		funcSignature = b.parseFuncSignature(true)
		nodes := b.createFuncTypeQualNodeList(qualifiers, functionKeyword, true)
		qualifierList = nodes[0]
		functionKeyword = nodes[1]
		b.endContext()
		return b.parseAnonFuncExprOrFuncTypeDescWithComponents(qualifierList, functionKeyword, funcSignature)
	}
	funcSignature = st.CreateEmptyNode()
	nodes := b.createFuncTypeQualNodeList(qualifiers, functionKeyword, false)
	qualifierList = nodes[0]
	functionKeyword = nodes[1]
	funcTypeDesc := st.CreateFunctionTypeDescriptorNode(qualifierList, functionKeyword,
		funcSignature)
	if b.getCurrentContext() != common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	}
	return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
}

func (b *ballerinaParser) parseAnonFuncExprOrFuncTypeDescWithComponents(qualifierList st.STNode, functionKeyword st.STNode, funcSignature st.STNode) st.STNode {
	currentCtx := b.getCurrentContext()
	switch b.peek().Kind() {
	case st.OPEN_BRACE_TOKEN, st.RIGHT_DOUBLE_ARROW_TOKEN:
		if currentCtx != common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
			b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		}
		b.startContext(common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION)
		funcSignatureNode, ok := funcSignature.(*st.STFunctionSignatureNode)
		if !ok {
			b.internalError("parseAnonFuncExprOrFuncTypeDescWithComponents: expected STFunctionSignatureNode")
			return st.CreateEmptyNode()
		}
		funcSignature = b.validateAndGetFuncParams(*funcSignatureNode)
		funcBody := b.parseAnonFuncBody(false)
		annots := st.CreateEmptyNodeList()
		anonFunc := st.CreateExplicitAnonymousFunctionExpressionNode(annots, qualifierList,
			functionKeyword, funcSignature, funcBody)
		return b.parseExpressionRhs(anonFunc, false, true)
	case st.IDENTIFIER_TOKEN:
		fallthrough
	default:
		funcTypeDesc := st.CreateFunctionTypeDescriptorNode(qualifierList, functionKeyword,
			funcSignature)
		if currentCtx != common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
			b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
			return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
				true)
		}
		return b.parseComplexTypeDescriptor(funcTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
	}
}

func (b *ballerinaParser) parseTypeDescOrExprRhs(typeOrExpr st.STNode) st.STNode {
	nextToken := b.peek()
	var typeDesc st.STNode
	switch nextToken.Kind() {
	case st.PIPE_TOKEN,
		st.BITWISE_AND_TOKEN:
		nextNextToken := b.peekN(2)
		if nextNextToken.Kind() == st.EQUAL_TOKEN {
			return typeOrExpr
		}
		pipeOrAndToken := b.parseBinaryOperator()
		rhsTypeDescOrExpr := b.parseTypeDescOrExpr()
		if b.isExpression(rhsTypeDescOrExpr.Kind()) {
			return st.CreateBinaryExpressionNode(st.BINARY_EXPRESSION, typeOrExpr,
				pipeOrAndToken, rhsTypeDescOrExpr)
		}
		typeDesc = b.getTypeDescFromExpr(typeOrExpr)
		rhsTypeDescOrExpr = b.getTypeDescFromExpr(rhsTypeDescOrExpr)
		return b.mergeTypes(typeDesc, pipeOrAndToken, rhsTypeDescOrExpr)
	case st.IDENTIFIER_TOKEN,
		st.QUESTION_MARK_TOKEN:
		typeDesc = b.parseComplexTypeDescriptor(b.getTypeDescFromExpr(typeOrExpr),
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, false)
		return typeDesc
	case st.SEMICOLON_TOKEN:
		return b.getTypeDescFromExpr(typeOrExpr)
	case st.EQUAL_TOKEN, st.CLOSE_PAREN_TOKEN, st.CLOSE_BRACE_TOKEN, st.CLOSE_BRACKET_TOKEN, st.EOF_TOKEN, st.COMMA_TOKEN:
		return typeOrExpr
	case st.OPEN_BRACKET_TOKEN:
		return b.parseTypedBindingPatternOrMemberAccess(typeOrExpr, false, true,
			common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	case st.ELLIPSIS_TOKEN:
		ellipsis := b.parseEllipsis()
		typeOrExpr = b.getTypeDescFromExpr(typeOrExpr)
		return st.CreateRestDescriptorNode(typeOrExpr, ellipsis)
	default:
		if b.isCompoundAssignment(nextToken.Kind()) {
			return typeOrExpr
		}
		if b.isValidExprRhsStart(nextToken.Kind(), typeOrExpr.Kind()) {
			return b.parseExpressionRhsInner(defaultOpPrecedence, typeOrExpr, false, false, false, false)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS)
		return b.parseTypeDescOrExprRhs(typeOrExpr)
	}
}

func (b *ballerinaParser) isAmbiguous(node st.STNode) bool {
	switch node.Kind() {
	case st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE,
		st.NIL_LITERAL,
		st.NULL_LITERAL,
		st.NUMERIC_LITERAL,
		st.STRING_LITERAL,
		st.BOOLEAN_LITERAL,
		st.BRACKETED_LIST:
		return true
	case st.BINARY_EXPRESSION:
		binaryExpr, ok := node.(*st.STBinaryExpressionNode)
		if !ok {
			b.internalError("expected STBinaryExpressionNode")
			return false
		}
		if binaryExpr.Operator.Kind() != st.PIPE_TOKEN {
			return false
		}
		return (b.isAmbiguous(binaryExpr.LhsExpr) && b.isAmbiguous(binaryExpr.RhsExpr))
	case st.BRACED_EXPRESSION:
		bracedExpr, ok := node.(*st.STBracedExpressionNode)
		if !ok {
			b.internalError("isAmbiguous: expected STBracedExpressionNode")
			return false
		}
		return b.isAmbiguous(bracedExpr.Expression)
	case st.INDEXED_EXPRESSION:
		indexExpr, ok := node.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("expected STIndexedExpressionNode")
			return false
		}
		if !b.isAmbiguous(indexExpr.ContainerExpression) {
			return false
		}
		keys := indexExpr.KeyExpression
		i := 0
		for ; i < keys.BucketCount(); i++ {
			item := keys.ChildInBucket(i)
			if item.Kind() == st.COMMA_TOKEN {
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

func (b *ballerinaParser) isAllBasicLiterals(node st.STNode) bool {
	switch node.Kind() {
	case st.NIL_LITERAL, st.NULL_LITERAL, st.NUMERIC_LITERAL, st.STRING_LITERAL, st.BOOLEAN_LITERAL:
		return true
	case st.BINARY_EXPRESSION:
		binaryExpr, ok := node.(*st.STBinaryExpressionNode)
		if !ok {
			b.internalError("expected STBinaryExpressionNode")
			return false
		}
		if binaryExpr.Operator.Kind() != st.PIPE_TOKEN {
			return false
		}
		return (b.isAmbiguous(binaryExpr.LhsExpr) && b.isAmbiguous(binaryExpr.RhsExpr))
	case st.BRACED_EXPRESSION:
		bracedExpr, ok := node.(*st.STBracedExpressionNode)
		if !ok {
			b.internalError("isAllBasicLiterals: expected STBracedExpressionNode")
			return false
		}
		return b.isAmbiguous(bracedExpr.Expression)
	case st.BRACKETED_LIST:
		list, ok := node.(*st.STAmbiguousCollectionNode)
		if !ok {
			b.internalError("expected STAmbiguousCollectionNode")
			return false
		}
		for _, member := range list.Members {
			if member.Kind() == st.COMMA_TOKEN {
				continue
			}
			if !b.isAllBasicLiterals(member) {
				return false
			}
		}
		return true
	case st.UNARY_EXPRESSION:
		unaryExpr, ok := node.(*st.STUnaryExpressionNode)
		if !ok {
			b.internalError("expected STUnaryExpressionNode")
			return false
		}
		if (unaryExpr.UnaryOperator.Kind() != st.PLUS_TOKEN) && (unaryExpr.UnaryOperator.Kind() != st.MINUS_TOKEN) {
			return false
		}
		return b.isNumericLiteral(unaryExpr.Expression)
	default:
		return false
	}
}

func (b *ballerinaParser) isNumericLiteral(node st.STNode) bool {
	switch node.Kind() {
	case st.NUMERIC_LITERAL:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseBindingPattern() st.STNode {
	switch b.peek().Kind() {
	case st.OPEN_BRACKET_TOKEN:
		return b.parseListBindingPattern()
	case st.IDENTIFIER_TOKEN:
		return b.parseBindingPatternStartsWithIdentifier()
	case st.OPEN_BRACE_TOKEN:
		return b.parseMappingBindingPattern()
	case st.ERROR_KEYWORD:
		return b.parseErrorBindingPattern()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_BINDING_PATTERN)
		return b.parseBindingPattern()
	}
}

func (b *ballerinaParser) parseBindingPatternStartsWithIdentifier() st.STNode {
	argNameOrBindingPattern := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER)
	secondToken := b.peek()
	if secondToken.Kind() == st.OPEN_PAREN_TOKEN {
		b.startContext(common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN)
		error := st.CreateMissingTokenWithDiagnostics(st.ERROR_KEYWORD,
			common.PARSER_RULE_CONTEXT_ERROR_KEYWORD.GetErrorCode())
		return b.parseErrorBindingPatternWithTypeRef(error, argNameOrBindingPattern)
	}
	if argNameOrBindingPattern.Kind() != st.SIMPLE_NAME_REFERENCE {
		var identifier st.STNode
		identifier = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		identifier = st.CloneWithLeadingInvalidNodeMinutiae(identifier, argNameOrBindingPattern,
			&common.ERROR_FIELD_BP_INSIDE_LIST_BP)
		return st.CreateCaptureBindingPatternNode(identifier)
	}
	simpleNameNode, ok := argNameOrBindingPattern.(*st.STSimpleNameReferenceNode)
	if !ok {
		b.internalError("parseBindingPatternStartsWithIdentifier: expected STSimpleNameReferenceNode")
		return st.CreateEmptyNode()
	}
	return b.createCaptureOrWildcardBP(simpleNameNode.Name)
}

func (b *ballerinaParser) createCaptureOrWildcardBP(varName st.STNode) st.STNode {
	var bindingPattern st.STNode
	if b.isWildcardBP(varName) {
		bindingPattern = b.getWildcardBindingPattern(varName)
	} else {
		bindingPattern = st.CreateCaptureBindingPatternNode(varName)
	}
	return bindingPattern
}

func (b *ballerinaParser) parseListBindingPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN)
	openBracket := b.parseOpenBracket()
	listBindingPattern, _ := b.parseListBindingPatternWithOpenBracket(openBracket, nil)
	b.endContext()
	return listBindingPattern
}

func (b *ballerinaParser) parseListBindingPatternWithOpenBracket(openBracket st.STNode, bindingPatternsList []st.STNode) (st.STNode, []st.STNode) {
	if b.isEndOfListBindingPattern(b.peek().Kind()) && len(bindingPatternsList) == 0 {
		closeBracket := b.parseCloseBracket()
		bindingPatternsNode := st.CreateNodeList(bindingPatternsList...)
		return st.CreateListBindingPatternNode(openBracket, bindingPatternsNode, closeBracket), bindingPatternsList
	}
	listBindingPatternMember := b.parseListBindingPatternMember()
	bindingPatternsList = append(bindingPatternsList, listBindingPatternMember)
	listBindingPattern, bindingPatternsList := b.parseListBindingPatternWithFirstMember(openBracket, listBindingPatternMember, bindingPatternsList)
	return listBindingPattern, bindingPatternsList
}

func (b *ballerinaParser) parseListBindingPatternWithFirstMember(openBracket st.STNode, firstMember st.STNode, bindingPatterns []st.STNode) (st.STNode, []st.STNode) {
	member := firstMember
	token := b.peek()
	var listBindingPatternRhs st.STNode
	for (!b.isEndOfListBindingPattern(token.Kind())) && (member.Kind() != st.REST_BINDING_PATTERN) {
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
	bindingPatternsNode := st.CreateNodeList(bindingPatterns...)
	return st.CreateListBindingPatternNode(openBracket, bindingPatternsNode, closeBracket), bindingPatterns
}

func (b *ballerinaParser) parseListBindingPatternMemberRhs() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END)
		return b.parseListBindingPatternMemberRhs()
	}
}

func (b *ballerinaParser) isEndOfListBindingPattern(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.CLOSE_BRACKET_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseListBindingPatternMember() st.STNode {
	switch b.peek().Kind() {
	case st.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	case st.OPEN_BRACKET_TOKEN,
		st.IDENTIFIER_TOKEN,
		st.OPEN_BRACE_TOKEN,
		st.ERROR_KEYWORD:
		return b.parseBindingPattern()
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER)
		return b.parseListBindingPatternMember()
	}
}

func (b *ballerinaParser) parseRestBindingPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN)
	ellipsis := b.parseEllipsis()
	varName := b.parseVariableName()
	b.endContext()
	simpleNameReferenceNode, ok := st.CreateSimpleNameReferenceNode(varName).(*st.STSimpleNameReferenceNode)
	if !ok {
		b.internalError("expected STSimpleNameReferenceNode")
		return st.CreateEmptyNode()
	}
	return st.CreateRestBindingPatternNode(ellipsis, simpleNameReferenceNode)
}

func (b *ballerinaParser) parseTypedBindingPatternWithContext(context common.ParserRuleContext) st.STNode {
	return b.parseTypedBindingPatternInner(nil, context)
}

func (b *ballerinaParser) parseTypedBindingPatternInner(qualifiers []st.STNode, context common.ParserRuleContext) st.STNode {
	typeDesc := b.parseTypeDescriptorWithinContext(qualifiers,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true, false, typePrecedenceDefault)
	typeBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, context)
	return typeBindingPattern
}

func (b *ballerinaParser) parseMappingBindingPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN)
	openBrace := b.parseOpenBrace()
	token := b.peek()
	if b.isEndOfMappingBindingPattern(token.Kind()) {
		closeBrace := b.parseCloseBrace()
		bindingPatternsNode := st.CreateEmptyNodeList()
		b.endContext()
		return st.CreateMappingBindingPatternNode(openBrace, bindingPatternsNode, closeBrace)
	}
	var bindingPatterns []st.STNode
	prevMember := b.parseMappingBindingPatternMember()
	if prevMember.Kind() != st.REST_BINDING_PATTERN {
		bindingPatterns = append(bindingPatterns, prevMember)
	}
	res, _ := b.parseMappingBindingPatternInner(openBrace, bindingPatterns, prevMember)
	return res
}

func (b *ballerinaParser) parseMappingBindingPatternInner(openBrace st.STNode, bindingPatterns []st.STNode, prevMember st.STNode) (st.STNode, []st.STNode) {
	token := b.peek()
	var mappingBindingPatternRhs st.STNode
	for (!b.isEndOfMappingBindingPattern(token.Kind())) && (prevMember.Kind() != st.REST_BINDING_PATTERN) {
		mappingBindingPatternRhs = b.parseMappingBindingPatternEnd()
		if mappingBindingPatternRhs == nil {
			break
		}
		bindingPatterns = append(bindingPatterns, mappingBindingPatternRhs)
		prevMember = b.parseMappingBindingPatternMember()
		if prevMember.Kind() == st.REST_BINDING_PATTERN {
			break
		}
		bindingPatterns = append(bindingPatterns, prevMember)
		token = b.peek()
	}
	if prevMember.Kind() == st.REST_BINDING_PATTERN {
		bindingPatterns = append(bindingPatterns, prevMember)
	}
	closeBrace := b.parseCloseBrace()
	bindingPatternsNode := st.CreateNodeList(bindingPatterns...)
	b.endContext()
	return st.CreateMappingBindingPatternNode(openBrace, bindingPatternsNode, closeBrace), bindingPatterns
}

func (b *ballerinaParser) parseMappingBindingPatternMember() st.STNode {
	token := b.peek()
	switch token.Kind() {
	case st.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	default:
		return b.parseFieldBindingPattern()
	}
}

func (b *ballerinaParser) parseMappingBindingPatternEnd() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACE_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END)
		return b.parseMappingBindingPatternEnd()
	}
}

func (b *ballerinaParser) parseFieldBindingPattern() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME)
		simpleNameReference := st.CreateSimpleNameReferenceNode(identifier)
		return b.parseFieldBindingPatternWithName(simpleNameReference)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME)
		return b.parseFieldBindingPattern()
	}
}

func (b *ballerinaParser) parseFieldBindingPatternWithName(simpleNameReference st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.COMMA_TOKEN, st.CLOSE_BRACE_TOKEN:
		return st.CreateFieldBindingPatternVarnameNode(simpleNameReference)
	case st.COLON_TOKEN:
		colon := b.parseColon()
		bindingPattern := b.parseBindingPattern()
		return st.CreateFieldBindingPatternFullNode(simpleNameReference, colon, bindingPattern)
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END)
		return b.parseFieldBindingPatternWithName(simpleNameReference)
	}
}

func (b *ballerinaParser) isEndOfMappingBindingPattern(nextTokenKind st.SyntaxKind) bool {
	return ((nextTokenKind == st.CLOSE_BRACE_TOKEN) || b.isEndOfModuleLevelNode())
}

func (b *ballerinaParser) parseErrorTypeDescOrErrorBP(annots st.STNode) st.STNode {
	nextNextToken := b.peekN(2)
	switch nextNextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		return b.parseAsErrorBindingPattern()
	case st.LT_TOKEN:
		return b.parseAsErrorTypeDesc(annots)
	case st.IDENTIFIER_TOKEN:
		nextNextNextTokenKind := b.peekN(3).Kind()
		if (nextNextNextTokenKind == st.COLON_TOKEN) || (nextNextNextTokenKind == st.OPEN_PAREN_TOKEN) {
			return b.parseAsErrorBindingPattern()
		}
		fallthrough
	default:
		return b.parseAsErrorTypeDesc(annots)
	}
}

func (b *ballerinaParser) parseAsErrorBindingPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
	return b.parseAssignmentStmtRhs(b.parseErrorBindingPattern())
}

func (b *ballerinaParser) parseAsErrorTypeDesc(annots st.STNode) st.STNode {
	finalKeyword := st.CreateEmptyNode()
	return b.parseVariableDecl(b.getAnnotations(annots), finalKeyword)
}

func (b *ballerinaParser) parseErrorBindingPattern() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN)
	error := b.parseErrorKeyword()
	return b.parseErrorBindingPatternWithKeyword(error)
}

func (b *ballerinaParser) parseErrorBindingPatternWithKeyword(error st.STNode) st.STNode {
	nextToken := b.peek()
	var typeRef st.STNode
	switch nextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		typeRef = st.CreateEmptyNode()
	default:
		if b.isPredeclaredIdentifier(nextToken.Kind()) {
			typeRef = b.parseTypeReference()
			break
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS)
		return b.parseErrorBindingPatternWithKeyword(error)
	}
	return b.parseErrorBindingPatternWithTypeRef(error, typeRef)
}

func (b *ballerinaParser) parseErrorBindingPatternWithTypeRef(error st.STNode, typeRef st.STNode) st.STNode {
	openParenthesis := b.parseOpenParenthesis()
	argListBindingPatterns := b.parseErrorArgListBindingPatterns()
	closeParenthesis := b.parseCloseParenthesis()
	b.endContext()
	return st.CreateErrorBindingPatternNode(error, typeRef, openParenthesis,
		argListBindingPatterns, closeParenthesis)
}

func (b *ballerinaParser) parseErrorArgListBindingPatterns() st.STNode {
	var argListBindingPatterns []st.STNode
	if b.isEndOfErrorFieldBindingPatterns() {
		return st.CreateNodeList(argListBindingPatterns...)
	}
	return b.parseErrorArgListBindingPatternsWithList(argListBindingPatterns)
}

func (b *ballerinaParser) parseErrorArgListBindingPatternsWithList(argListBindingPatterns []st.STNode) st.STNode {
	firstArg := b.parseErrorArgListBindingPattern(common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START, true)
	if firstArg == nil {
		return st.CreateNodeList(argListBindingPatterns...)
	}
	switch firstArg.Kind() {
	case st.CAPTURE_BINDING_PATTERN, st.WILDCARD_BINDING_PATTERN:
		argListBindingPatterns = append(argListBindingPatterns, firstArg)
		return b.parseErrorArgListBPWithoutErrorMsg(argListBindingPatterns)
	case st.ERROR_BINDING_PATTERN:
		missingIdentifier := st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
		missingErrorMsgBP := st.CreateCaptureBindingPatternNode(missingIdentifier)
		missingErrorMsgBP = st.AddDiagnostic(missingErrorMsgBP,
			&common.ERROR_MISSING_ERROR_MESSAGE_BINDING_PATTERN)
		missingComma := st.CreateMissingTokenWithDiagnostics(st.COMMA_TOKEN,
			&common.ERROR_MISSING_COMMA_TOKEN)
		argListBindingPatterns = append(argListBindingPatterns, missingErrorMsgBP)
		argListBindingPatterns = append(argListBindingPatterns, missingComma)
		argListBindingPatterns = append(argListBindingPatterns, firstArg)
		return b.parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns, firstArg.Kind())
	case st.NAMED_ARG_BINDING_PATTERN, st.REST_BINDING_PATTERN:
		argListBindingPatterns = append(argListBindingPatterns, firstArg)
		return b.parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns, firstArg.Kind())
	default:
		b.addInvalidNodeToNextToken(firstArg, &common.ERROR_BINDING_PATTERN_NOT_ALLOWED)
		return b.parseErrorArgListBindingPatternsWithList(argListBindingPatterns)
	}
}

func (b *ballerinaParser) parseErrorArgListBPWithoutErrorMsg(argListBindingPatterns []st.STNode) st.STNode {
	argEnd := b.parseErrorArgsBindingPatternEnd(common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END)
	if argEnd == nil {
		// null marks the end of args
		return st.CreateNodeList(argListBindingPatterns...)
	}
	secondArg := b.parseErrorArgListBindingPattern(common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS, false)
	if secondArg == nil { // depending on the recovery context we will not get null here
		b.internalError("assertion failed")
		return st.CreateEmptyNode()
	}
	switch secondArg.Kind() {
	case st.CAPTURE_BINDING_PATTERN, st.WILDCARD_BINDING_PATTERN, st.ERROR_BINDING_PATTERN, st.REST_BINDING_PATTERN, st.NAMED_ARG_BINDING_PATTERN:
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

func (b *ballerinaParser) parseErrorArgListBPWithoutErrorMsgAndCause(argListBindingPatterns []st.STNode, lastValidArgKind st.SyntaxKind) st.STNode {
	for !b.isEndOfErrorFieldBindingPatterns() {
		argEnd := b.parseErrorArgsBindingPatternEnd(common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END)
		if argEnd == nil {
			// null marks the end of args
			break
		}
		currentArg := b.parseErrorArgListBindingPattern(common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN, false)
		if currentArg == nil { // depending on the recovery context we will not get null here
			b.internalError("assertion failed")
			return st.CreateEmptyNode()
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
	return st.CreateNodeList(argListBindingPatterns...)
}

func (b *ballerinaParser) isEndOfErrorFieldBindingPatterns() bool {
	nextTokenKind := b.peek().Kind()
	switch nextTokenKind {
	case st.CLOSE_PAREN_TOKEN, st.EOF_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseErrorArgsBindingPatternEnd(currentCtx common.ParserRuleContext) st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_PAREN_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), currentCtx)
		return b.parseErrorArgsBindingPatternEnd(currentCtx)
	}
}

func (b *ballerinaParser) parseErrorArgListBindingPattern(context common.ParserRuleContext, isFirstArg bool) st.STNode {
	switch b.peek().Kind() {
	case st.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	case st.IDENTIFIER_TOKEN:
		argNameOrSimpleBindingPattern := b.consume()
		return b.parseNamedOrSimpleArgBindingPattern(argNameOrSimpleBindingPattern)
	case st.OPEN_BRACKET_TOKEN, st.OPEN_BRACE_TOKEN, st.ERROR_KEYWORD:
		return b.parseBindingPattern()
	case st.CLOSE_PAREN_TOKEN:
		if isFirstArg {
			return nil
		}
		fallthrough
	default:
		b.recoverWithBlockContext(b.peek(), context)
		return b.parseErrorArgListBindingPattern(context, isFirstArg)
	}
}

func (b *ballerinaParser) parseNamedOrSimpleArgBindingPattern(argNameOrSimpleBindingPattern st.STNode) st.STNode {
	secondToken := b.peek()
	switch secondToken.Kind() {
	case st.EQUAL_TOKEN:
		equal := b.consume()
		bindingPattern := b.parseBindingPattern()
		return st.CreateNamedArgBindingPatternNode(argNameOrSimpleBindingPattern,
			equal, bindingPattern)
	case st.COMMA_TOKEN, st.CLOSE_PAREN_TOKEN:
		fallthrough
	default:
		return b.createCaptureOrWildcardBP(argNameOrSimpleBindingPattern)
	}
}

func (b *ballerinaParser) validateErrorFieldBindingPatternOrder(prevArgKind st.SyntaxKind, currentArgKind st.SyntaxKind) *common.DiagnosticErrorCode {
	switch currentArgKind {
	case st.NAMED_ARG_BINDING_PATTERN,
		st.REST_BINDING_PATTERN:
		if prevArgKind == st.REST_BINDING_PATTERN {
			return &common.ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG
		}
		return nil
	default:
		return &common.ERROR_BINDING_PATTERN_NOT_ALLOWED
	}
}

func (b *ballerinaParser) parseTypedBindingPatternTypeRhs(typeDesc st.STNode, context common.ParserRuleContext) st.STNode {
	return b.parseTypedBindingPatternTypeRhsWithRoot(typeDesc, context, true)
}

func (b *ballerinaParser) parseTypedBindingPatternTypeRhsWithRoot(typeDesc st.STNode, context common.ParserRuleContext, isRoot bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN, st.OPEN_BRACE_TOKEN, st.ERROR_KEYWORD:
		bindingPattern := b.parseBindingPattern()
		return st.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
	case st.OPEN_BRACKET_TOKEN:
		typedBindingPattern := b.parseTypedBindingPatternOrMemberAccess(typeDesc, true, true, context)
		if typedBindingPattern.Kind() != st.TYPED_BINDING_PATTERN {
			b.internalError("assertion failed")
			return st.CreateEmptyNode()
		}
		return typedBindingPattern
	case st.CLOSE_PAREN_TOKEN, st.COMMA_TOKEN, st.CLOSE_BRACKET_TOKEN, st.CLOSE_BRACE_TOKEN:
		if !isRoot {
			return typeDesc
		}
		fallthrough
	default:
		b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS)
		return b.parseTypedBindingPatternTypeRhsWithRoot(typeDesc, context, isRoot)
	}
}

func (b *ballerinaParser) parseTypedBindingPatternOrMemberAccess(typeDescOrExpr st.STNode, isTypedBindingPattern bool, allowAssignment bool, context common.ParserRuleContext) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	if b.isBracketedListEnd(b.peek().Kind()) {
		return b.parseAsArrayTypeDesc(typeDescOrExpr, openBracket, st.CreateEmptyNode(), context)
	}
	member := b.parseBracketedListMember(isTypedBindingPattern)
	currentNodeType := b.getBracketedListNodeType(member, isTypedBindingPattern)
	switch currentNodeType {
	case st.ARRAY_TYPE_DESC:
		typedBindingPattern := b.parseAsArrayTypeDesc(typeDescOrExpr, openBracket, member, context)
		return typedBindingPattern
	case st.LIST_BINDING_PATTERN:
		bindingPattern, _ := b.parseAsListBindingPatternWithMemberAndRoot(openBracket, nil, member, false)
		typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		return st.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
	case st.INDEXED_EXPRESSION:
		return b.parseAsMemberAccessExpr(typeDescOrExpr, openBracket, member)
	case st.ARRAY_TYPE_DESC_OR_MEMBER_ACCESS:
		break
	case st.NONE:
		fallthrough
	default:
		memberEnd := b.parseBracketedListMemberEnd()
		if memberEnd != nil {
			var memberList []st.STNode
			memberList = append(memberList, b.getBindingPattern(member, true))
			memberList = append(memberList, memberEnd)
			bindingPattern, memberList := b.parseAsListBindingPattern(openBracket, memberList) //nolint:staticcheck,ineffassign // memberList will be used when list binding pattern is fully implemented
			typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
			return st.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
		}
	}
	closeBracket := b.parseCloseBracket()
	b.endContext()
	return b.parseTypedBindingPatternOrMemberAccessRhs(typeDescOrExpr, openBracket, member, closeBracket,
		isTypedBindingPattern, allowAssignment, context)
}

func (b *ballerinaParser) parseAsMemberAccessExpr(typeNameOrExpr st.STNode, openBracket st.STNode, member st.STNode) st.STNode {
	member = b.parseExpressionRhs(member, false, true)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	keyExpr := st.CreateNodeList(member)
	memberAccessExpr := st.CreateIndexedExpressionNode(typeNameOrExpr, openBracket, keyExpr, closeBracket)
	return b.parseExpressionRhs(memberAccessExpr, false, false)
}

func (b *ballerinaParser) isBracketedListEnd(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACKET_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseBracketedListMember(isTypedBindingPattern bool) st.STNode {
	nextToken := b.peek()

	switch nextToken.Kind() {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN, st.HEX_INTEGER_LITERAL_TOKEN, st.ASTERISK_TOKEN, st.STRING_LITERAL_TOKEN:
		return b.parseBasicLiteral()
	case st.CLOSE_BRACKET_TOKEN:
		return st.CreateEmptyNode()
	case st.OPEN_BRACE_TOKEN, st.ERROR_KEYWORD, st.ELLIPSIS_TOKEN, st.OPEN_BRACKET_TOKEN:
		return b.parseStatementStartBracketedListMember()
	case st.IDENTIFIER_TOKEN:
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

func (b *ballerinaParser) parseAsArrayTypeDesc(typeDesc st.STNode, openBracket st.STNode, member st.STNode, context common.ParserRuleContext) st.STNode {
	typeDesc = b.getTypeDescFromExpr(typeDesc)
	b.switchContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
	b.startContext(common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	b.endContext()
	return b.parseTypedBindingPatternOrMemberAccessRhs(typeDesc, openBracket, member, closeBracket, true, true,
		context)
}

func (b *ballerinaParser) parseBracketedListMemberEnd() st.STNode {
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return b.parseComma()
	case st.CLOSE_BRACKET_TOKEN:
		return nil
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END)
		return b.parseBracketedListMemberEnd()
	}
}

func (b *ballerinaParser) parseTypedBindingPatternOrMemberAccessRhs(typeDescOrExpr st.STNode, openBracket st.STNode, member st.STNode, closeBracket st.STNode, isTypedBindingPattern bool, allowAssignment bool, context common.ParserRuleContext) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN, st.OPEN_BRACE_TOKEN, st.ERROR_KEYWORD:
		typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
		return b.parseTypedBindingPatternTypeRhs(arrayTypeDesc, context)
	case st.OPEN_BRACKET_TOKEN:
		if isTypedBindingPattern {
			typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
			arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
			return b.parseTypedBindingPatternTypeRhs(arrayTypeDesc, context)
		}
		keyExpr := b.getKeyExpr(member)
		expr := st.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr, closeBracket)
		return b.parseTypedBindingPatternOrMemberAccess(expr, false, allowAssignment, context)
	case st.QUESTION_MARK_TOKEN:
		typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
		typeDesc = b.parseComplexTypeDescriptor(arrayTypeDesc,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
		return b.parseTypedBindingPatternTypeRhs(typeDesc, context)
	case st.PIPE_TOKEN, st.BITWISE_AND_TOKEN:
		if (!isTypedBindingPattern) && allowAssignment && (b.peekN(2).Kind() == st.EQUAL_TOKEN) && b.isValidLVExpr(typeDescOrExpr) {
			keyExpr := b.getKeyExpr(member)
			typeDescOrExpr = b.getExpression(typeDescOrExpr)
			return st.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr, closeBracket)
		}
		return b.parseComplexTypeDescInTypedBPOrExprRhs(typeDescOrExpr, openBracket, member, closeBracket,
			isTypedBindingPattern)
	case st.IN_KEYWORD:
		if ((context != common.PARSER_RULE_CONTEXT_FOREACH_STMT) && (context != common.PARSER_RULE_CONTEXT_FROM_CLAUSE)) && (context != common.PARSER_RULE_CONTEXT_JOIN_CLAUSE) {
			break
		}
		return b.createTypedBindingPattern(typeDescOrExpr, openBracket, member, closeBracket)
	case st.EQUAL_TOKEN:
		if (context == common.PARSER_RULE_CONTEXT_FOREACH_STMT) || (context == common.PARSER_RULE_CONTEXT_FROM_CLAUSE) {
			break
		}
		if (isTypedBindingPattern || (!allowAssignment)) || (!b.isValidLVExpr(typeDescOrExpr)) {
			return b.createTypedBindingPattern(typeDescOrExpr, openBracket, member, closeBracket)
		}
		keyExpr := b.getKeyExpr(member)
		typeDescOrExpr = b.getExpression(typeDescOrExpr)
		return st.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr, closeBracket)
	case st.SEMICOLON_TOKEN:
		if (context == common.PARSER_RULE_CONTEXT_FOREACH_STMT) || (context == common.PARSER_RULE_CONTEXT_FROM_CLAUSE) {
			break
		}
		return b.createTypedBindingPattern(typeDescOrExpr, openBracket, member, closeBracket)
	case st.CLOSE_BRACE_TOKEN, st.COMMA_TOKEN:
		if context == common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT {
			keyExpr := b.getKeyExpr(member)
			return st.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr,
				closeBracket)
		}
		return nil
	default:
		if (!isTypedBindingPattern) && b.isValidExprRhsStart(nextToken.Kind(), closeBracket.Kind()) {
			keyExpr := b.getKeyExpr(member)
			typeDescOrExpr = b.getExpression(typeDescOrExpr)
			return st.CreateIndexedExpressionNode(typeDescOrExpr, openBracket, keyExpr,
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

func (b *ballerinaParser) getKeyExpr(member st.STNode) st.STNode {
	if member == nil {
		keyIdentifier := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR)
		missingVarRef := st.CreateSimpleNameReferenceNode(keyIdentifier)
		return st.CreateNodeList(missingVarRef)
	}
	return st.CreateNodeList(member)
}

func (b *ballerinaParser) createTypedBindingPattern(typeDescOrExpr st.STNode, openBracket st.STNode, member st.STNode, closeBracket st.STNode) st.STNode {
	bindingPatterns := st.CreateEmptyNodeList()
	if !b.isEmpty(member) {
		memberKind := member.Kind()
		if (memberKind == st.NUMERIC_LITERAL) || (memberKind == st.ASTERISK_LITERAL) {
			typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
			arrayTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, typeDesc)
			identifierToken := st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
				&common.ERROR_MISSING_VARIABLE_NAME)
			variableName := st.CreateCaptureBindingPatternNode(identifierToken)
			return st.CreateTypedBindingPatternNode(arrayTypeDesc, variableName)
		}
		bindingPattern := b.getBindingPattern(member, true)
		bindingPatterns = st.CreateNodeList(bindingPattern)
	}
	bindingPattern := st.CreateListBindingPatternNode(openBracket, bindingPatterns, closeBracket)
	typeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
	return st.CreateTypedBindingPatternNode(typeDesc, bindingPattern)
}

func (b *ballerinaParser) parseComplexTypeDescInTypedBPOrExprRhs(typeDescOrExpr st.STNode, openBracket st.STNode, member st.STNode, closeBracket st.STNode, isTypedBindingPattern bool) st.STNode {
	pipeOrAndToken := b.parseUnionOrIntersectionToken()
	typedBindingPatternOrExpr := b.parseTypedBindingPatternOrExpr(false)
	if typedBindingPatternOrExpr.Kind() == st.TYPED_BINDING_PATTERN {
		lhsTypeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		lhsTypeDesc = b.getArrayTypeDesc(openBracket, member, closeBracket, lhsTypeDesc)
		rhsTypedBindingPattern, ok := typedBindingPatternOrExpr.(*st.STTypedBindingPatternNode)
		if !ok {
			b.internalError("expected *st.STTypedBindingPatternNode")
			return st.CreateEmptyNode()
		}
		rhsTypeDesc := rhsTypedBindingPattern.TypeDescriptor
		newTypeDesc := b.mergeTypes(lhsTypeDesc, pipeOrAndToken, rhsTypeDesc)
		return st.CreateTypedBindingPatternNode(newTypeDesc, rhsTypedBindingPattern.BindingPattern)
	}
	if isTypedBindingPattern {
		lhsTypeDesc := b.getTypeDescFromExpr(typeDescOrExpr)
		lhsTypeDesc = b.getArrayTypeDesc(openBracket, member, closeBracket, lhsTypeDesc)
		return b.createCaptureBPWithMissingVarName(lhsTypeDesc, pipeOrAndToken, typedBindingPatternOrExpr)
	}
	keyExpr := b.getExpression(member)
	containerExpr := b.getExpression(typeDescOrExpr)
	lhsExpr := st.CreateIndexedExpressionNode(containerExpr, openBracket, keyExpr, closeBracket)
	return st.CreateBinaryExpressionNode(st.BINARY_EXPRESSION, lhsExpr, pipeOrAndToken,
		typedBindingPatternOrExpr)
}

func (b *ballerinaParser) mergeTypes(lhsTypeDesc st.STNode, pipeOrAndToken st.STNode, rhsTypeDesc st.STNode) st.STNode {
	if pipeOrAndToken.Kind() == st.PIPE_TOKEN {
		return b.mergeTypesWithUnion(lhsTypeDesc, pipeOrAndToken, rhsTypeDesc)
	} else {
		return b.mergeTypesWithIntersection(lhsTypeDesc, pipeOrAndToken, rhsTypeDesc)
	}
}

func (b *ballerinaParser) mergeTypesWithUnion(lhsTypeDesc st.STNode, pipeToken st.STNode, rhsTypeDesc st.STNode) st.STNode {
	if rhsTypeDesc.Kind() == st.UNION_TYPE_DESC {
		rhsUnionTypeDesc, ok := rhsTypeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		return b.replaceLeftMostUnionWithAUnion(lhsTypeDesc, pipeToken, rhsUnionTypeDesc)
	} else {
		return b.createUnionTypeDesc(lhsTypeDesc, pipeToken, rhsTypeDesc)
	}
}

func (b *ballerinaParser) mergeTypesWithIntersection(lhsTypeDesc st.STNode, bitwiseAndToken st.STNode, rhsTypeDesc st.STNode) st.STNode {
	if lhsTypeDesc.Kind() == st.UNION_TYPE_DESC {
		lhsUnionTypeDesc, ok := lhsTypeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		if rhsTypeDesc.Kind() == st.INTERSECTION_TYPE_DESC {
			rhsIntSecTypeDesc, ok := rhsTypeDesc.(*st.STIntersectionTypeDescriptorNode)
			if !ok {
				b.internalError("expected *st.STIntersectionTypeDescriptorNode")
				return st.CreateEmptyNode()
			}
			rhsTypeDesc = b.replaceLeftMostIntersectionWithAIntersection(lhsUnionTypeDesc.RightTypeDesc,
				bitwiseAndToken, rhsIntSecTypeDesc)
			return b.createUnionTypeDesc(lhsUnionTypeDesc.LeftTypeDesc, lhsUnionTypeDesc.PipeToken, rhsTypeDesc)
		} else if rhsTypeDesc.Kind() == st.UNION_TYPE_DESC {
			rhsUnionTypeDesc, ok := rhsTypeDesc.(*st.STUnionTypeDescriptorNode)
			if !ok {
				b.internalError("expected *st.STUnionTypeDescriptorNode")
				return st.CreateEmptyNode()
			}
			//nolint:staticcheck // rhsTypeDesc reassigned but not yet used in return path
			rhsTypeDesc = b.replaceLeftMostUnionWithAIntersection(lhsUnionTypeDesc.RightTypeDesc,
				bitwiseAndToken, rhsUnionTypeDesc)
			return b.replaceLeftMostUnionWithAUnion(lhsUnionTypeDesc.LeftTypeDesc,
				lhsUnionTypeDesc.PipeToken, rhsUnionTypeDesc)
		}
	}
	if rhsTypeDesc.Kind() == st.UNION_TYPE_DESC {
		rhsUnionTypeDesc, ok := rhsTypeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		return b.replaceLeftMostUnionWithAIntersection(lhsTypeDesc, bitwiseAndToken, rhsUnionTypeDesc)
	} else if rhsTypeDesc.Kind() == st.INTERSECTION_TYPE_DESC {
		rhsIntSecTypeDesc, ok := rhsTypeDesc.(*st.STIntersectionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STIntersectionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		return b.replaceLeftMostIntersectionWithAIntersection(lhsTypeDesc, bitwiseAndToken, rhsIntSecTypeDesc)
	}
	return b.createIntersectionTypeDesc(lhsTypeDesc, bitwiseAndToken, rhsTypeDesc)
}

func (b *ballerinaParser) replaceLeftMostUnionWithAUnion(typeDesc st.STNode, pipeToken st.STNode, unionTypeDesc *st.STUnionTypeDescriptorNode) st.STNode {
	leftTypeDesc := unionTypeDesc.LeftTypeDesc
	if leftTypeDesc.Kind() == st.UNION_TYPE_DESC {
		leftUnionTypeDesc, ok := leftTypeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newLeftTypeDesc := b.replaceLeftMostUnionWithAUnion(typeDesc, pipeToken, leftUnionTypeDesc)
		return st.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	leftTypeDesc = b.createUnionTypeDesc(typeDesc, pipeToken, leftTypeDesc)
	return st.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, leftTypeDesc)
}

func (b *ballerinaParser) replaceLeftMostUnionWithAIntersection(typeDesc st.STNode, bitwiseAndToken st.STNode, unionTypeDesc *st.STUnionTypeDescriptorNode) st.STNode {
	leftTypeDesc := unionTypeDesc.LeftTypeDesc
	if leftTypeDesc.Kind() == st.UNION_TYPE_DESC {
		leftUnionTypeDesc, ok := leftTypeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newLeftTypeDesc := b.replaceLeftMostUnionWithAIntersection(typeDesc, bitwiseAndToken, leftUnionTypeDesc)
		return st.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	if leftTypeDesc.Kind() == st.INTERSECTION_TYPE_DESC {
		leftIntersectionTypeDesc, ok := leftTypeDesc.(*st.STIntersectionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STIntersectionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newLeftTypeDesc := b.replaceLeftMostIntersectionWithAIntersection(typeDesc, bitwiseAndToken, leftIntersectionTypeDesc)
		return st.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	leftTypeDesc = b.createIntersectionTypeDesc(typeDesc, bitwiseAndToken, leftTypeDesc)
	return st.Replace(unionTypeDesc, unionTypeDesc.LeftTypeDesc, leftTypeDesc)
}

func (b *ballerinaParser) replaceLeftMostIntersectionWithAIntersection(typeDesc st.STNode, bitwiseAndToken st.STNode, intersectionTypeDesc *st.STIntersectionTypeDescriptorNode) st.STNode {
	leftTypeDesc := intersectionTypeDesc.LeftTypeDesc
	if leftTypeDesc.Kind() == st.INTERSECTION_TYPE_DESC {
		leftIntersectionTypeDesc, ok := leftTypeDesc.(*st.STIntersectionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STIntersectionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newLeftTypeDesc := b.replaceLeftMostIntersectionWithAIntersection(typeDesc, bitwiseAndToken, leftIntersectionTypeDesc)
		return st.Replace(intersectionTypeDesc, intersectionTypeDesc.LeftTypeDesc, newLeftTypeDesc)
	}
	leftTypeDesc = b.createIntersectionTypeDesc(typeDesc, bitwiseAndToken, leftTypeDesc)
	return st.Replace(intersectionTypeDesc, intersectionTypeDesc.LeftTypeDesc, leftTypeDesc)
}

func (b *ballerinaParser) getArrayTypeDesc(openBracket st.STNode, member st.STNode, closeBracket st.STNode, lhsTypeDesc st.STNode) st.STNode {
	if lhsTypeDesc.Kind() == st.UNION_TYPE_DESC {
		unionTypeDesc, ok := lhsTypeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		middleTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, unionTypeDesc.RightTypeDesc)
		lhsTypeDesc = b.mergeTypesWithUnion(unionTypeDesc.LeftTypeDesc, unionTypeDesc.PipeToken, middleTypeDesc)
	} else if lhsTypeDesc.Kind() == st.INTERSECTION_TYPE_DESC {
		intersectionTypeDesc, ok := lhsTypeDesc.(*st.STIntersectionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STIntersectionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		middleTypeDesc := b.getArrayTypeDesc(openBracket, member, closeBracket, intersectionTypeDesc.RightTypeDesc)
		lhsTypeDesc = b.mergeTypesWithIntersection(intersectionTypeDesc.LeftTypeDesc,
			intersectionTypeDesc.BitwiseAndToken, middleTypeDesc)
	} else {
		lhsTypeDesc = b.createArrayTypeDesc(lhsTypeDesc, openBracket, member, closeBracket)
	}
	return lhsTypeDesc
}

func (b *ballerinaParser) parseUnionOrIntersectionToken() st.STNode {
	token := b.peek()
	if (token.Kind() == st.PIPE_TOKEN) || (token.Kind() == st.BITWISE_AND_TOKEN) {
		return b.consume()
	} else {
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN)
		return b.parseUnionOrIntersectionToken()
	}
}

func (b *ballerinaParser) getBracketedListNodeType(memberNode st.STNode, isTypedBindingPattern bool) st.SyntaxKind {
	if b.isEmpty(memberNode) {
		return st.NONE
	}
	if b.isDefiniteTypeDesc(memberNode.Kind()) {
		return st.TUPLE_TYPE_DESC
	}
	switch memberNode.Kind() {
	case st.ASTERISK_LITERAL:
		return st.ARRAY_TYPE_DESC
	case st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.REST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.WILDCARD_BINDING_PATTERN:
		return st.LIST_BINDING_PATTERN
	case st.QUALIFIED_NAME_REFERENCE,
		st.REST_TYPE:
		return st.TUPLE_TYPE_DESC
	case st.NUMERIC_LITERAL:
		if isTypedBindingPattern {
			return st.ARRAY_TYPE_DESC
		}
		return st.ARRAY_TYPE_DESC_OR_MEMBER_ACCESS
	case st.SIMPLE_NAME_REFERENCE,
		st.BRACKETED_LIST,
		st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return st.NONE
	case st.ERROR_CONSTRUCTOR:
		if isTypedBindingPattern {
			return st.LIST_BINDING_PATTERN
		}
		errorCtorNode, ok := memberNode.(*st.STErrorConstructorExpressionNode)
		if !ok {
			b.internalError("getBracketedListNodeType: expected STErrorConstructorExpressionNode")
			return st.NONE
		}
		if b.isPossibleErrorBindingPattern(*errorCtorNode) {
			return st.NONE
		}
		return st.INDEXED_EXPRESSION
	default:
		if isTypedBindingPattern {
			return st.NONE
		}
		return st.INDEXED_EXPRESSION
	}
}

func (b *ballerinaParser) parseStatementStartsWithOpenBracket(annots st.STNode, possibleMappingField bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT)
	return b.parseStatementStartsWithOpenBracketWithRoot(annots, true, possibleMappingField)
}

func (b *ballerinaParser) parseMemberBracketedList() st.STNode {
	annots := st.CreateEmptyNodeList()
	return b.parseStatementStartsWithOpenBracketWithRoot(annots, false, false)
}

func (b *ballerinaParser) parseStatementStartsWithOpenBracketWithRoot(annots st.STNode, isRoot bool, possibleMappingField bool) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	var memberList []st.STNode
	for !b.isBracketedListEnd(b.peek().Kind()) {
		member := b.parseStatementStartBracketedListMember()
		currentNodeType := b.getStmtStartBracketedListType(member)
		switch currentNodeType {
		case st.TUPLE_TYPE_DESC:
			member = b.parseComplexTypeDescriptor(member, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
			member = b.createMemberOrRestNode(st.CreateEmptyNodeList(), member)
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot)
		case st.MEMBER_TYPE_DESC, st.REST_TYPE:
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot)
		case st.LIST_BINDING_PATTERN:
			res, _ := b.parseAsListBindingPatternWithMemberAndRoot(openBracket, memberList, member, isRoot)
			return res
		case st.LIST_CONSTRUCTOR:
			res, _ := b.parseAsListConstructor(openBracket, memberList, member, isRoot)
			return res
		case st.LIST_BP_OR_LIST_CONSTRUCTOR:
			res, _ := b.parseAsListBindingPatternOrListConstructor(openBracket, memberList, member, isRoot)
			return res
		case st.TUPLE_TYPE_DESC_OR_LIST_CONST:
			res, _ := b.parseAsTupleTypeDescOrListConstructor(annots, openBracket, memberList, member, isRoot)
			return res
		case st.NONE:
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

func (b *ballerinaParser) parseStatementStartBracketedListMember() st.STNode {
	return b.parseStatementStartBracketedListMemberWithQualifiers(nil)
}

func (b *ballerinaParser) parseStatementStartBracketedListMemberWithQualifiers(qualifiers []st.STNode) st.STNode {
	qualifiers = b.parseTypeDescQualifiers(qualifiers)
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACKET_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMemberBracketedList()
	case st.IDENTIFIER_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		identifier := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		if b.isWildcardBP(identifier) {
			simpleNameNode, ok := identifier.(*st.STSimpleNameReferenceNode)
			if !ok {
				b.internalError("parseStatementStartBracketedListMember: expected STSimpleNameReferenceNode")
				return st.CreateEmptyNode()
			}
			varName := simpleNameNode.Name
			return b.getWildcardBindingPattern(varName)
		}
		nextToken = b.peek()
		if nextToken.Kind() == st.ELLIPSIS_TOKEN {
			ellipsis := b.parseEllipsis()
			return st.CreateRestDescriptorNode(identifier, ellipsis)
		}
		if (nextToken.Kind() != st.OPEN_BRACKET_TOKEN) && b.isValidTypeContinuationToken(nextToken) {
			return b.parseComplexTypeDescriptor(identifier, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
		}
		return b.parseExpressionRhs(identifier, false, true)
	case st.OPEN_BRACE_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseMappingBindingPatterOrMappingConstructor()
	case st.ERROR_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		nextNextToken := b.getNextNextToken()
		if (nextNextToken.Kind() == st.OPEN_PAREN_TOKEN) || (nextNextToken.Kind() == st.IDENTIFIER_TOKEN) {
			return b.parseErrorBindingPatternOrErrorConstructor()
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case st.ELLIPSIS_TOKEN:
		b.reportInvalidQualifierList(qualifiers)
		return b.parseRestBindingOrSpreadMember()
	case st.XML_KEYWORD, st.STRING_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		if b.getNextNextToken().Kind() == st.BACKTICK_TOKEN {
			return b.parseExpressionPossibleRhsExpr()
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case st.TABLE_KEYWORD, st.STREAM_KEYWORD:
		b.reportInvalidQualifierList(qualifiers)
		if b.getNextNextToken().Kind() == st.LT_TOKEN {
			return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		return b.parseExpressionPossibleRhsExpr()
	case st.OPEN_PAREN_TOKEN:
		return b.parseTypeDescOrExprWithQualifiers(qualifiers)
	case st.FUNCTION_KEYWORD:
		return b.parseAnonFuncExprOrFuncTypeDesc(qualifiers)
	case st.AT_TOKEN:
		return b.parseTupleMember()
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			b.reportInvalidQualifierList(qualifiers)
			return b.parseExpressionPossibleRhsExpr()
		}
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseTypeDescriptorWithQualifier(qualifiers, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER)
		return b.parseStatementStartBracketedListMemberWithQualifiers(qualifiers)
	}
}

func (b *ballerinaParser) parseRestBindingOrSpreadMember() st.STNode {
	ellipsis := b.parseEllipsis()
	expr := b.parseExpression()
	if expr.Kind() == st.SIMPLE_NAME_REFERENCE {
		return st.CreateRestBindingPatternNode(ellipsis, expr)
	} else {
		return st.CreateSpreadMemberNode(ellipsis, expr)
	}
}

// return result and modified memberList
func (b *ballerinaParser) parseAsTupleTypeDescOrListConstructor(annots st.STNode, openBracket st.STNode, memberList []st.STNode, member st.STNode, isRoot bool) (st.STNode, []st.STNode) {
	memberList = append(memberList, member)
	memberEnd := b.parseBracketedListMemberEnd()
	var tupleTypeDescOrListCons st.STNode
	if memberEnd == nil {
		closeBracket := b.parseCloseBracket()
		tupleTypeDescOrListCons = b.parseTupleTypeDescOrListConstructorRhs(openBracket, memberList, closeBracket, isRoot)
	} else {
		memberList = append(memberList, memberEnd)
		tupleTypeDescOrListCons, memberList = b.parseTupleTypeDescOrListConstructorWithBracketAndMembers(annots, openBracket, memberList, isRoot)
	}
	return tupleTypeDescOrListCons, memberList
}

func (b *ballerinaParser) parseTupleTypeDescOrListConstructor(annots st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	var memberList []st.STNode
	result, _ := b.parseTupleTypeDescOrListConstructorWithBracketAndMembers(annots, openBracket, memberList, false)
	return result
}

func (b *ballerinaParser) parseTupleTypeDescOrListConstructorWithBracketAndMembers(annots st.STNode, openBracket st.STNode, memberList []st.STNode, isRoot bool) (st.STNode, []st.STNode) {
	nextToken := b.peek()
	for !b.isBracketedListEnd(nextToken.Kind()) {
		member := b.parseTupleTypeDescOrListConstructorMember(annots)
		currentNodeType := b.getParsingNodeTypeOfTupleTypeOrListCons(member)
		switch currentNodeType {
		case st.LIST_CONSTRUCTOR:
			return b.parseAsListConstructor(openBracket, memberList, member, isRoot)
		case st.REST_TYPE, st.MEMBER_TYPE_DESC:
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot), memberList
		case st.TUPLE_TYPE_DESC:
			member = b.parseComplexTypeDescriptor(member, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
			member = b.createMemberOrRestNode(st.CreateEmptyNodeList(), member)
			return b.parseAsTupleTypeDesc(annots, openBracket, memberList, member, isRoot), memberList
		case st.TUPLE_TYPE_DESC_OR_LIST_CONST:
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

func (b *ballerinaParser) parseTupleTypeDescOrListConstructorMember(annots st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACKET_TOKEN:
		return b.parseTupleTypeDescOrListConstructor(annots)
	case st.IDENTIFIER_TOKEN:
		identifier := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		if b.peek().Kind() == st.ELLIPSIS_TOKEN {
			ellipsis := b.parseEllipsis()
			return st.CreateRestDescriptorNode(identifier, ellipsis)
		}
		return b.parseExpressionRhs(identifier, false, false)
	case st.OPEN_BRACE_TOKEN:
		return b.parseMappingConstructorExpr()
	case st.ERROR_KEYWORD:
		nextNextToken := b.getNextNextToken()
		if (nextNextToken.Kind() == st.OPEN_PAREN_TOKEN) || (nextNextToken.Kind() == st.IDENTIFIER_TOKEN) {
			return b.parseErrorConstructorExprAmbiguous(false)
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case st.XML_KEYWORD, st.STRING_KEYWORD:
		if b.getNextNextToken().Kind() == st.BACKTICK_TOKEN {
			return b.parseExpressionPossibleRhsExpr()
		}
		return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
	case st.TABLE_KEYWORD, st.STREAM_KEYWORD:
		if b.getNextNextToken().Kind() == st.LT_TOKEN {
			return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		return b.parseExpressionPossibleRhsExpr()
	case st.OPEN_PAREN_TOKEN:
		return b.parseTypeDescOrExpr()
	case st.AT_TOKEN:
		return b.parseTupleMember()
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			return b.parseExpressionPossibleRhsExpr()
		}
		if b.isTypeStartingToken(nextToken.Kind()) {
			return b.parseTypeDescriptor(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE)
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER)
		return b.parseTupleTypeDescOrListConstructorMember(annots)
	}
}

func (b *ballerinaParser) getParsingNodeTypeOfTupleTypeOrListCons(memberNode st.STNode) st.SyntaxKind {
	return b.getStmtStartBracketedListType(memberNode)
}

func (b *ballerinaParser) parseTupleTypeDescOrListConstructorRhs(openBracket st.STNode, members []st.STNode, closeBracket st.STNode, isRoot bool) st.STNode {
	var tupleTypeOrListConst st.STNode
	switch b.peek().Kind() {
	case st.COMMA_TOKEN, st.CLOSE_BRACE_TOKEN, st.CLOSE_BRACKET_TOKEN, st.PIPE_TOKEN, st.BITWISE_AND_TOKEN:
		if !isRoot {
			b.endContext()
			return st.CreateAmbiguousCollectionNode(st.TUPLE_TYPE_DESC_OR_LIST_CONST, openBracket, members, closeBracket)
		}
	default:
		if b.isValidExprRhsStart(b.peek().Kind(), closeBracket.Kind()) || (isRoot && (b.peek().Kind() == st.EQUAL_TOKEN)) {
			members = b.getExpressionList(members, false)
			memberExpressions := st.CreateNodeList(members...)
			tupleTypeOrListConst = st.CreateListConstructorExpressionNode(openBracket,
				memberExpressions, closeBracket)
			break
		}
		memberTypeDescs := st.CreateNodeList(b.getTupleMemberList(members)...)
		tupleTypeDesc := st.CreateTupleTypeDescriptorNode(openBracket, memberTypeDescs, closeBracket)
		tupleTypeOrListConst = b.parseComplexTypeDescriptor(tupleTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
	}
	b.endContext()
	if !isRoot {
		return tupleTypeOrListConst
	}
	annots := st.CreateEmptyNodeList()
	return b.parseStmtStartsWithTupleTypeOrExprRhs(annots, tupleTypeOrListConst, true)
}

func (b *ballerinaParser) parseStmtStartsWithTupleTypeOrExprRhs(annots st.STNode, tupleTypeOrListConst st.STNode, isRoot bool) st.STNode {
	if (tupleTypeOrListConst.Kind().CompareTo(st.RECORD_TYPE_DESC) >= 0) && (tupleTypeOrListConst.Kind().CompareTo(st.TYPEDESC_TYPE_DESC) <= 0) {
		typedBindingPattern := b.parseTypedBindingPatternTypeRhsWithRoot(tupleTypeOrListConst, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, isRoot)
		if !isRoot {
			return typedBindingPattern
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		res, _ := b.parseVarDeclRhs(annots, typedBindingPattern)
		return res
	}
	expr := b.getExpression(tupleTypeOrListConst)
	expr = b.parseExpressionRhs(expr, false, true)
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *ballerinaParser) parseAsTupleTypeDesc(annots st.STNode, openBracket st.STNode, memberList []st.STNode, member st.STNode, isRoot bool) st.STNode {
	memberList = b.getTupleMemberList(memberList)
	b.startContext(common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS)
	tupleTypeMembers := b.parseTupleTypeMembers(member, memberList)
	closeBracket := b.parseCloseBracket()
	b.endContext()
	tupleType := st.CreateTupleTypeDescriptorNode(openBracket, tupleTypeMembers, closeBracket)
	typeDesc := b.parseComplexTypeDescriptor(tupleType, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
	b.endContext()
	if !isRoot {
		return typeDesc
	}
	typedBindingPattern := b.parseTypedBindingPatternTypeRhsWithRoot(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, true)
	b.switchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	res, _ := b.parseVarDeclRhs(annots, typedBindingPattern)
	return res
}

func (b *ballerinaParser) parseAsListBindingPatternWithMemberAndRoot(openBracket st.STNode, memberList []st.STNode, member st.STNode, isRoot bool) (st.STNode, []st.STNode) {
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

func (b *ballerinaParser) parseAsListBindingPattern(openBracket st.STNode, memberList []st.STNode) (st.STNode, []st.STNode) {
	memberList = b.getBindingPatternsList(memberList, true)
	b.switchContext(common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN)
	listBindingPattern, memberList := b.parseListBindingPatternWithOpenBracket(openBracket, memberList)
	b.endContext()
	return listBindingPattern, memberList
}

func (b *ballerinaParser) parseAsListBindingPatternOrListConstructor(openBracket st.STNode, memberList []st.STNode, member st.STNode, isRoot bool) (st.STNode, []st.STNode) {
	memberList = append(memberList, member)
	memberEnd := b.parseBracketedListMemberEnd()
	var listBindingPatternOrListCons st.STNode
	if memberEnd == nil {
		closeBracket := b.parseCloseBracket()
		listBindingPatternOrListCons = b.parseListBindingPatternOrListConstructorWithCloseBracket(openBracket, memberList, closeBracket, isRoot)
	} else {
		memberList = append(memberList, memberEnd)
		listBindingPatternOrListCons, memberList = b.parseListBindingPatternOrListConstructorInner(openBracket, memberList, isRoot)
	}
	return listBindingPatternOrListCons, memberList
}

func (b *ballerinaParser) getStmtStartBracketedListType(memberNode st.STNode) st.SyntaxKind {
	if (memberNode.Kind().CompareTo(st.RECORD_TYPE_DESC) >= 0) && (memberNode.Kind().CompareTo(st.FUTURE_TYPE_DESC) <= 0) {
		return st.TUPLE_TYPE_DESC
	}
	switch memberNode.Kind() {
	case st.WILDCARD_BINDING_PATTERN,
		st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.ERROR_BINDING_PATTERN:
		return st.LIST_BINDING_PATTERN
	case st.QUALIFIED_NAME_REFERENCE:
		return st.TUPLE_TYPE_DESC
	case st.LIST_CONSTRUCTOR,
		st.MAPPING_CONSTRUCTOR,
		st.SPREAD_MEMBER:
		return st.LIST_CONSTRUCTOR
	case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		st.REST_BINDING_PATTERN:
		return st.LIST_BP_OR_LIST_CONSTRUCTOR
	case st.SIMPLE_NAME_REFERENCE, // member is a simple type-ref/var-ref
		st.BRACKETED_LIST:
		return st.NONE
	case st.ERROR_CONSTRUCTOR:
		errorCtorNode, ok := memberNode.(*st.STErrorConstructorExpressionNode)
		if !ok {
			b.internalError("getStmtStartBracketedListType: expected STErrorConstructorExpressionNode")
			return st.NONE
		}
		if b.isPossibleErrorBindingPattern(*errorCtorNode) {
			return st.NONE
		}
		return st.LIST_CONSTRUCTOR
	case st.INDEXED_EXPRESSION:
		return st.TUPLE_TYPE_DESC_OR_LIST_CONST
	case st.MEMBER_TYPE_DESC:
		return st.MEMBER_TYPE_DESC
	case st.REST_TYPE:
		return st.REST_TYPE
	default:
		if (b.isExpression(memberNode.Kind()) && (!b.isAllBasicLiterals(memberNode))) && (!b.isAmbiguous(memberNode)) {
			return st.LIST_CONSTRUCTOR
		}
		return st.NONE
	}
}

func (b *ballerinaParser) isPossibleErrorBindingPattern(errorConstructor st.STErrorConstructorExpressionNode) bool {
	args := errorConstructor.Arguments
	size := args.BucketCount()
	i := 0
	for ; i < size; i++ {
		arg := args.ChildInBucket(i)
		if ((arg.Kind() != st.NAMED_ARG) && (arg.Kind() != st.POSITIONAL_ARG)) && (arg.Kind() != st.REST_ARG) {
			continue
		}
		functionArg := arg
		if !b.isPosibleArgBindingPattern(functionArg) {
			return false
		}
	}
	return true
}

func (b *ballerinaParser) isPosibleArgBindingPattern(arg st.STFunctionArgumentNode) bool {
	switch arg.Kind() {
	case st.POSITIONAL_ARG:
		positionalArg, ok := arg.(*st.STPositionalArgumentNode)
		if !ok {
			b.internalError("isPosibleArgBindingPattern: expected STPositionalArgumentNode")
			return false
		}
		return b.isPosibleBindingPattern(positionalArg.Expression)
	case st.NAMED_ARG:
		namedArg, ok := arg.(*st.STNamedArgumentNode)
		if !ok {
			b.internalError("isPosibleArgBindingPattern: expected STNamedArgumentNode")
			return false
		}
		return b.isPosibleBindingPattern(namedArg.Expression)
	case st.REST_ARG:
		restArg, ok := arg.(*st.STRestArgumentNode)
		if !ok {
			b.internalError("isPosibleArgBindingPattern: expected STRestArgumentNode")
			return false
		}
		return (restArg.Expression.Kind() == st.SIMPLE_NAME_REFERENCE)
	default:
		return false
	}
}

func (b *ballerinaParser) isPosibleBindingPattern(node st.STNode) bool {
	switch node.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		return true
	case st.LIST_CONSTRUCTOR:
		listConstructor, ok := node.(*st.STListConstructorExpressionNode)
		if !ok {
			b.internalError("isPosibleBindingPattern: expected STListConstructorExpressionNode")
			return false
		}
		i := 0
		for ; i < listConstructor.BucketCount(); i++ {
			expr := listConstructor.ChildInBucket(i)
			if !b.isPosibleBindingPattern(expr) {
				return false
			}
		}
		return true
	case st.MAPPING_CONSTRUCTOR:
		mappingConstructor, ok := node.(*st.STMappingConstructorExpressionNode)
		if !ok {
			b.internalError("isPosibleBindingPattern: expected STMappingConstructorExpressionNode")
			return false
		}
		i := 0
		for ; i < mappingConstructor.BucketCount(); i++ {
			expr := mappingConstructor.ChildInBucket(i)
			if !b.isPosibleBindingPattern(expr) {
				return false
			}
		}
		return true
	case st.SPECIFIC_FIELD:
		specificField, ok := node.(*st.STSpecificFieldNode)
		if !ok {
			b.internalError("isPosibleBindingPattern: expected STSpecificFieldNode")
			return false
		}
		if specificField.ReadonlyKeyword != nil {
			return false
		}
		if specificField.ValueExpr == nil {
			return true
		}
		return b.isPosibleBindingPattern(specificField.ValueExpr)
	case st.ERROR_CONSTRUCTOR:
		errorCtorNode, ok := node.(*st.STErrorConstructorExpressionNode)
		if !ok {
			b.internalError("isPosibleBindingPattern: expected STErrorConstructorExpressionNode")
			return false
		}
		return b.isPossibleErrorBindingPattern(*errorCtorNode)
	default:
		return false
	}
}

// return result, and modified memberList
func (b *ballerinaParser) parseStatementStartBracketedListRhs(annots st.STNode, openBracket st.STNode, members []st.STNode, closeBracket st.STNode, isRoot bool, possibleMappingField bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.EQUAL_TOKEN:
		if !isRoot {
			b.endContext()
			return st.CreateAmbiguousCollectionNode(st.BRACKETED_LIST, openBracket, members, closeBracket)
		}
		memberBindingPatterns := st.CreateNodeList(b.getBindingPatternsList(members, true)...)
		listBindingPattern := st.CreateListBindingPatternNode(openBracket,
			memberBindingPatterns, closeBracket)
		b.endContext() // end tuple typ-desc
		b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
		return b.parseAssignmentStmtRhs(listBindingPattern)
	case st.IDENTIFIER_TOKEN, st.OPEN_BRACE_TOKEN:
		if !isRoot {
			b.endContext()
			return st.CreateAmbiguousCollectionNode(st.BRACKETED_LIST, openBracket, members, closeBracket)
		}
		if len(members) == 0 {
			openBracket = st.AddDiagnostic(openBracket, &common.ERROR_MISSING_TUPLE_MEMBER)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		b.startContext(common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS)
		memberTypeDescs := st.CreateNodeList(b.getTupleMemberList(members)...)
		tupleTypeDesc := st.CreateTupleTypeDescriptorNode(openBracket, memberTypeDescs, closeBracket)
		b.endContext() // end tuple typ-desc
		typeDesc := b.parseComplexTypeDescriptor(tupleTypeDesc,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
		b.endContext() // end binding pattern
		typedBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, typedBindingPattern)
	case st.OPEN_BRACKET_TOKEN:
		// [a, ..][..
		// definitely not binding pattern. Can be type-desc or list-constructor
		if !isRoot {
			// if this is a member, treat as type-desc.
			// TODO: handle expression case.
			memberTypeDescs := st.CreateNodeList(b.getTupleMemberList(members)...)
			tupleTypeDesc := st.CreateTupleTypeDescriptorNode(openBracket, memberTypeDescs, closeBracket)
			b.endContext()
			typeDesc := b.parseComplexTypeDescriptor(tupleTypeDesc, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE, false)
			return typeDesc
		}
		list := st.CreateAmbiguousCollectionNode(st.BRACKETED_LIST, openBracket, members, closeBracket)
		b.endContext()
		tpbOrExpr := b.parseTypedBindingPatternOrExprRhs(list, true)
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, tpbOrExpr)
	case st.COLON_TOKEN: // "{[a]:" could be a computed-name-field in mapping-constructor
		if possibleMappingField && (len(members) == 1) {
			b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
			colon := b.parseColon()
			fieldNameExpr := b.getExpression(members[0])
			valueExpr := b.parseExpression()
			return st.CreateComputedNameFieldNode(openBracket, fieldNameExpr, closeBracket, colon,
				valueExpr)
		}
		// fall through
		fallthrough
	default:
		b.endContext()
		if !isRoot {
			return st.CreateAmbiguousCollectionNode(st.BRACKETED_LIST, openBracket, members, closeBracket)
		}
		list := st.CreateAmbiguousCollectionNode(st.BRACKETED_LIST, openBracket, members, closeBracket)
		exprOrTPB := b.parseTypedBindingPatternOrExprRhs(list, false)
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, exprOrTPB)
	}
}

func (b *ballerinaParser) isWildcardBP(node st.STNode) bool {
	switch node.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		simpleNameNode, ok := node.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("isWildcardBP: expected STSimpleNameReferenceNode")
			return false
		}
		nameToken, ok := simpleNameNode.Name.(st.STToken)
		if !ok {
			b.internalError("isWildcardBP: expected STToken")
			return false
		}
		return b.isUnderscoreToken(nameToken)
	case st.IDENTIFIER_TOKEN:
		identifierToken, ok := node.(st.STToken)
		if !ok {
			b.internalError("isWildcardBP: expected STToken")
			return false
		}
		return b.isUnderscoreToken(identifierToken)
	default:
		return false
	}
}

func (b *ballerinaParser) isUnderscoreToken(token st.STToken) bool {
	return token.Text() == "_"
}

func (b *ballerinaParser) getWildcardBindingPattern(identifier st.STNode) st.STNode {
	var underscore st.STNode
	switch identifier.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		simpleNameNode, ok := identifier.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("getWildcardBindingPattern: expected STSimpleNameReferenceNode")
			return st.CreateEmptyNode()
		}
		varName := simpleNameNode.Name
		nameToken, ok := varName.(st.STToken)
		if !ok {
			b.internalError("getWildcardBindingPattern: expected STToken")
			return st.CreateEmptyNode()
		}
		underscore = b.getUnderscoreKeyword(nameToken)
		return st.CreateWildcardBindingPatternNode(underscore)
	case st.IDENTIFIER_TOKEN:
		identifierToken, ok := identifier.(st.STToken)
		if !ok {
			b.internalError("getWildcardBindingPattern: expected STToken")
			return st.CreateEmptyNode()
		}
		underscore = b.getUnderscoreKeyword(identifierToken)
		return st.CreateWildcardBindingPatternNode(underscore)
	default:
		b.internalError("getWildcardBindingPattern: expected SIMPLE_NAME_REFERENCE or IDENTIFIER_TOKEN")
		return st.CreateEmptyNode()
	}
}

func (b *ballerinaParser) parseStatementStartsWithOpenBrace() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	openBrace := b.parseOpenBrace()
	if b.peek().Kind() == st.CLOSE_BRACE_TOKEN {
		closeBrace := b.parseCloseBrace()
		switch b.peek().Kind() {
		case st.EQUAL_TOKEN:
			b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
			fields := st.CreateEmptyNodeList()
			bindingPattern := st.CreateMappingBindingPatternNode(openBrace, fields,
				closeBrace)
			return b.parseAssignmentStmtRhs(bindingPattern)
		case st.RIGHT_ARROW_TOKEN, st.SYNC_SEND_TOKEN:
			b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
			fields := st.CreateEmptyNodeList()
			expr := st.CreateMappingConstructorExpressionNode(openBrace, fields, closeBrace)
			expr = b.parseExpressionRhs(expr, false, true)
			return b.parseStatementStartWithExprRhs(expr)
		default:
			statements := st.CreateEmptyNodeList()
			b.endContext()
			return st.CreateBlockStatementNode(openBrace, statements, closeBrace)
		}
	}
	member := b.parseStatementStartingBracedListFirstMember(openBrace.IsMissing())
	nodeType := b.getBracedListType(member)
	var stmt st.STNode
	switch nodeType {
	case st.MAPPING_BINDING_PATTERN:
		return b.parseStmtAsMappingBindingPatternStart(openBrace, member)
	case st.MAPPING_CONSTRUCTOR:
		return b.parseStmtAsMappingConstructorStart(openBrace, member)
	case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return b.parseStmtAsMappingBPOrMappingConsStart(openBrace, member)
	case st.BLOCK_STATEMENT:
		closeBrace := b.parseCloseBrace()
		stmt = st.CreateBlockStatementNode(openBrace, member, closeBrace)
		b.endContext()
		return stmt
	default:
		var stmts []st.STNode
		stmts = append(stmts, member)
		statements := b.parseStatementsInner(stmts)
		closeBrace := b.parseCloseBrace()
		b.endContext()
		return st.CreateBlockStatementNode(openBrace, statements, closeBrace)
	}
}

func (b *ballerinaParser) parseStmtAsMappingBindingPatternStart(openBrace st.STNode, firstMappingField st.STNode) st.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN)
	var bindingPatterns []st.STNode
	if firstMappingField.Kind() != st.REST_BINDING_PATTERN {
		bindingPatterns = append(bindingPatterns, b.getBindingPattern(firstMappingField, false))
	}
	mappingBP, _ := b.parseMappingBindingPatternInner(openBrace, bindingPatterns, firstMappingField)
	return b.parseAssignmentStmtRhs(mappingBP)
}

func (b *ballerinaParser) parseStmtAsMappingConstructorStart(openBrace st.STNode, firstMember st.STNode) st.STNode {
	b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
	mappingCons := b.parseAsMappingConstructor(openBrace, nil, firstMember)
	expr := b.parseExpressionRhs(mappingCons, false, true)
	return b.parseStatementStartWithExprRhs(expr)
}

func (b *ballerinaParser) parseAsMappingConstructor(openBrace st.STNode, members []st.STNode, member st.STNode) st.STNode {
	members = append(members, member)
	members = b.getExpressionList(members, true)
	b.switchContext(common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR)
	fields := b.finishParseMappingConstructorFields(members)
	closeBrace := b.parseCloseBrace()
	b.endContext()
	return st.CreateMappingConstructorExpressionNode(openBrace, fields, closeBrace)
}

func (b *ballerinaParser) parseStmtAsMappingBPOrMappingConsStart(openBrace st.STNode, member st.STNode) st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR)
	var members []st.STNode
	members = append(members, member)
	var bpOrConstructor st.STNode
	memberEnd := b.parseMappingFieldEnd()
	if memberEnd == nil {
		closeBrace := b.parseCloseBrace()
		bpOrConstructor = b.parseMappingBindingPatternOrMappingConstructorWithCloseBrace(openBrace, members, closeBrace)
	} else {
		members = append(members, memberEnd)
		bpOrConstructor = b.parseMappingBindingPatternOrMappingConstructor(openBrace, members)
	}
	switch bpOrConstructor.Kind() {
	case st.MAPPING_CONSTRUCTOR:
		b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		expr := b.parseExpressionRhs(bpOrConstructor, false, true)
		return b.parseStatementStartWithExprRhs(expr)
	case st.MAPPING_BINDING_PATTERN:
		b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
		bindingPattern := b.getBindingPattern(bpOrConstructor, false)
		return b.parseAssignmentStmtRhs(bindingPattern)
	case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		fallthrough
	default:
		if b.peek().Kind() == st.EQUAL_TOKEN {
			b.switchContext(common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT)
			bindingPattern := b.getBindingPattern(bpOrConstructor, false)
			return b.parseAssignmentStmtRhs(bindingPattern)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		expr := b.getExpression(bpOrConstructor)
		expr = b.parseExpressionRhs(expr, false, true)
		return b.parseStatementStartWithExprRhs(expr)
	}
}

func (b *ballerinaParser) parseStatementStartingBracedListFirstMember(isOpenBraceMissing bool) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.READONLY_KEYWORD:
		readonlyKeyword := b.parseReadonlyKeyword()
		return b.bracedListMemberStartsWithReadonly(readonlyKeyword)
	case st.IDENTIFIER_TOKEN:
		readonlyKeyword := st.CreateEmptyNode()
		return b.parseIdentifierRhsInStmtStartingBrace(readonlyKeyword)
	case st.STRING_LITERAL_TOKEN:
		key := b.parseStringLiteral()
		if b.peek().Kind() == st.COLON_TOKEN {
			readonlyKeyword := st.CreateEmptyNode()
			colon := b.parseColon()
			valueExpr := b.parseExpression()
			return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
		expr := b.parseExpressionRhs(key, false, true)
		return b.parseStatementStartWithExprRhs(expr)
	case st.OPEN_BRACKET_TOKEN:
		annots := st.CreateEmptyNodeList()
		return b.parseStatementStartsWithOpenBracket(annots, true)
	case st.OPEN_BRACE_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		return b.parseStatementStartsWithOpenBrace()
	case st.ELLIPSIS_TOKEN:
		return b.parseRestBindingPattern()
	default:
		if isOpenBraceMissing {
			readonlyKeyword := st.CreateEmptyNode()
			return b.parseIdentifierRhsInStmtStartingBrace(readonlyKeyword)
		}
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		return b.parseStatements()
	}
}

func (b *ballerinaParser) bracedListMemberStartsWithReadonly(readonlyKeyword st.STNode) st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.IDENTIFIER_TOKEN:
		return b.parseIdentifierRhsInStmtStartingBrace(readonlyKeyword)
	case st.STRING_LITERAL_TOKEN:
		if b.peekN(2).Kind() == st.COLON_TOKEN {
			key := b.parseStringLiteral()
			colon := b.parseColon()
			valueExpr := b.parseExpression()
			return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
		}
		fallthrough
	default:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		typeDesc := b.createBuiltinSimpleNameReference(readonlyKeyword)
		res, _ := b.parseVarDeclTypeDescRhs(typeDesc, st.CreateEmptyNodeList(), nil,
			true, false)
		return res
	}
}

func (b *ballerinaParser) parseIdentifierRhsInStmtStartingBrace(readonlyKeyword st.STNode) st.STNode {
	identifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
	switch b.peek().Kind() {
	case st.COMMA_TOKEN, st.CLOSE_BRACE_TOKEN:
		colon := st.CreateEmptyNode()
		value := st.CreateEmptyNode()
		return st.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, value)
	case st.COLON_TOKEN:
		colon := b.parseColon()
		if !b.isEmpty(readonlyKeyword) {
			value := b.parseExpression()
			return st.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, value)
		}
		switch b.peek().Kind() {
		case st.OPEN_BRACKET_TOKEN:
			bindingPatternOrExpr := b.parseListBindingPatternOrListConstructor()
			return b.getMappingField(identifier, colon, bindingPatternOrExpr)
		case st.OPEN_BRACE_TOKEN:
			bindingPatternOrExpr := b.parseMappingBindingPatterOrMappingConstructor()
			return b.getMappingField(identifier, colon, bindingPatternOrExpr)
		case st.ERROR_KEYWORD:
			bindingPatternOrExpr := b.parseErrorBindingPatternOrErrorConstructor()
			return b.getMappingField(identifier, colon, bindingPatternOrExpr)
		case st.IDENTIFIER_TOKEN:
			return b.parseQualifiedIdentifierRhsInStmtStartBrace(identifier, colon)
		default:
			expr := b.parseExpression()
			return b.getMappingField(identifier, colon, expr)
		}
	default:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		if !b.isEmpty(readonlyKeyword) {
			b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
			bindingPattern := st.CreateCaptureBindingPatternNode(identifier)
			typedBindingPattern := st.CreateTypedBindingPatternNode(readonlyKeyword, bindingPattern)
			annots := st.CreateEmptyNodeList()
			res, _ := b.parseVarDeclRhs(annots, typedBindingPattern)
			return res
		}
		b.startContext(common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
		qualifiedIdentifier := b.parseQualifiedIdentifierNode(identifier, false)
		expr := b.parseTypedBindingPatternOrExprRhs(qualifiedIdentifier, true)
		annots := st.CreateEmptyNodeList()
		return b.parseStmtStartsWithTypedBPOrExprRhs(annots, expr)
	}
}

func (b *ballerinaParser) parseQualifiedIdentifierRhsInStmtStartBrace(identifier st.STNode, colon st.STNode) st.STNode {
	secondIdentifier := b.parseIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
	secondNameRef := st.CreateSimpleNameReferenceNode(secondIdentifier)
	if b.isWildcardBP(secondIdentifier) {
		wildcardBP := b.getWildcardBindingPattern(secondIdentifier)
		nameRef := st.CreateSimpleNameReferenceNode(identifier)
		return st.CreateFieldBindingPatternFullNode(nameRef, colon, wildcardBP)
	}
	qualifiedNameRef := b.createQualifiedNameReferenceNode(identifier, colon, secondIdentifier)
	switch b.peek().Kind() {
	case st.COMMA_TOKEN:
		return st.CreateSpecificFieldNode(st.CreateEmptyNode(), identifier, colon,
			secondNameRef)
	case st.OPEN_BRACE_TOKEN, st.IDENTIFIER_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		typeBindingPattern := b.parseTypedBindingPatternTypeRhs(qualifiedNameRef, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		annots := st.CreateEmptyNodeList()
		res, _ := b.parseVarDeclRhs(annots, typeBindingPattern)
		return res
	case st.OPEN_BRACKET_TOKEN:
		return b.parseMemberRhsInStmtStartWithBrace(identifier, colon, secondIdentifier, secondNameRef)
	case st.QUESTION_MARK_TOKEN:
		typeDesc := b.parseComplexTypeDescriptor(qualifiedNameRef,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, true)
		typeBindingPattern := b.parseTypedBindingPatternTypeRhs(typeDesc, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		annots := st.CreateEmptyNodeList()
		res, _ := b.parseVarDeclRhs(annots, typeBindingPattern)
		return res
	case st.EQUAL_TOKEN, st.SEMICOLON_TOKEN:
		return b.parseStatementStartWithExprRhs(qualifiedNameRef)
	case st.PIPE_TOKEN, st.BITWISE_AND_TOKEN:
		fallthrough
	default:
		return b.parseMemberWithExprInRhs(identifier, colon, secondIdentifier, secondNameRef)
	}
}

func (b *ballerinaParser) getBracedListType(member st.STNode) st.SyntaxKind {
	switch member.Kind() {
	case st.FIELD_BINDING_PATTERN,
		st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.WILDCARD_BINDING_PATTERN:
		return st.MAPPING_BINDING_PATTERN
	case st.SPECIFIC_FIELD:
		specificFieldNode, ok := member.(*st.STSpecificFieldNode)
		if !ok {
			b.internalError("getBracedListType: expected STSpecificFieldNode")
			return st.NONE
		}
		expr := specificFieldNode.ValueExpr
		if expr == nil {
			return st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
		}
		switch expr.Kind() {
		case st.SIMPLE_NAME_REFERENCE,
			st.LIST_BP_OR_LIST_CONSTRUCTOR,
			st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
			return st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
		case st.ERROR_BINDING_PATTERN:
			return st.MAPPING_BINDING_PATTERN
		case st.ERROR_CONSTRUCTOR:
			errorCtorNode, ok := expr.(*st.STErrorConstructorExpressionNode)
			if !ok {
				b.internalError("getBracedListType: expected STErrorConstructorExpressionNode")
				return st.NONE
			}
			if b.isPossibleErrorBindingPattern(*errorCtorNode) {
				return st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
			}
			return st.MAPPING_CONSTRUCTOR
		default:
			return st.MAPPING_CONSTRUCTOR
		}
	case st.SPREAD_FIELD,
		st.COMPUTED_NAME_FIELD:
		return st.MAPPING_CONSTRUCTOR
	case st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE,
		st.LIST_BP_OR_LIST_CONSTRUCTOR,
		st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		st.REST_BINDING_PATTERN:
		return st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
	case st.LIST:
		return st.BLOCK_STATEMENT
	default:
		return st.NONE
	}
}

func (b *ballerinaParser) parseMappingBindingPatterOrMappingConstructor() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR)
	openBrace := b.parseOpenBrace()
	return b.parseMappingBindingPatternOrMappingConstructor(openBrace, nil)
}

func (b *ballerinaParser) isBracedListEnd(nextTokenKind st.SyntaxKind) bool {
	switch nextTokenKind {
	case st.EOF_TOKEN, st.CLOSE_BRACE_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) parseMappingBindingPatternOrMappingConstructor(openBrace st.STNode, memberList []st.STNode) st.STNode {
	nextToken := b.peek()
	for !b.isBracedListEnd(nextToken.Kind()) {
		member := b.parseMappingBindingPatterOrMappingConstructorMember()
		currentNodeType := b.getTypeOfMappingBPOrMappingCons(member)
		switch currentNodeType {
		case st.MAPPING_CONSTRUCTOR:
			return b.parseAsMappingConstructor(openBrace, memberList, member)
		case st.MAPPING_BINDING_PATTERN:
			node, _ := b.parseAsMappingBindingPattern(openBrace, memberList, member)
			return node
		case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
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
	return b.parseMappingBindingPatternOrMappingConstructorWithCloseBrace(openBrace, memberList, closeBrace)
}

func (b *ballerinaParser) parseMappingBindingPatterOrMappingConstructorMember() st.STNode {
	switch b.peek().Kind() {
	case st.IDENTIFIER_TOKEN:
		key := b.parseIdentifier(common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME)
		return b.parseMappingFieldRhs(key)
	case st.STRING_LITERAL_TOKEN:
		readonlyKeyword := st.CreateEmptyNode()
		key := b.parseStringLiteral()
		colon := b.parseColon()
		valueExpr := b.parseExpression()
		return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
	case st.OPEN_BRACKET_TOKEN:
		return b.parseComputedField()
	case st.ELLIPSIS_TOKEN:
		ellipsis := b.parseEllipsis()
		expr := b.parseExpression()
		if expr.Kind() == st.SIMPLE_NAME_REFERENCE {
			return st.CreateRestBindingPatternNode(ellipsis, expr)
		}
		return st.CreateSpreadFieldNode(ellipsis, expr)
	default:
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER)
		return b.parseMappingBindingPatterOrMappingConstructorMember()
	}
}

func (b *ballerinaParser) parseMappingFieldRhs(key st.STNode) st.STNode {
	var colon st.STNode
	var valueExpr st.STNode
	switch b.peek().Kind() {
	case st.COLON_TOKEN:
		colon = b.parseColon()
		return b.parseMappingFieldValue(key, colon)
	case st.COMMA_TOKEN, st.CLOSE_BRACE_TOKEN:
		readonlyKeyword := st.CreateEmptyNode()
		colon = st.CreateEmptyNode()
		valueExpr = st.CreateEmptyNode()
		return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, valueExpr)
	default:
		token := b.peek()
		b.recoverWithBlockContext(token, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END)
		readonlyKeyword := st.CreateEmptyNode()
		return b.parseSpecificFieldRhs(readonlyKeyword, key)
	}
}

func (b *ballerinaParser) parseMappingFieldValue(key st.STNode, colon st.STNode) st.STNode {
	var expr st.STNode
	switch b.peek().Kind() {
	case st.IDENTIFIER_TOKEN:
		expr = b.parseExpression()
	case st.OPEN_BRACKET_TOKEN:
		expr = b.parseListBindingPatternOrListConstructor()
	case st.OPEN_BRACE_TOKEN:
		expr = b.parseMappingBindingPatterOrMappingConstructor()
	default:
		expr = b.parseExpression()
	}
	if b.isBindingPattern(expr.Kind()) {
		key = st.CreateSimpleNameReferenceNode(key)
		return st.CreateFieldBindingPatternFullNode(key, colon, expr)
	}
	readonlyKeyword := st.CreateEmptyNode()
	return st.CreateSpecificFieldNode(readonlyKeyword, key, colon, expr)
}

func (b *ballerinaParser) isBindingPattern(kind st.SyntaxKind) bool {
	switch kind {
	case st.FIELD_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.WILDCARD_BINDING_PATTERN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParser) getTypeOfMappingBPOrMappingCons(memberNode st.STNode) st.SyntaxKind {
	switch memberNode.Kind() {
	case st.FIELD_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.WILDCARD_BINDING_PATTERN:
		return st.MAPPING_BINDING_PATTERN
	case st.SPECIFIC_FIELD:
		specificFieldNode, ok := memberNode.(*st.STSpecificFieldNode)
		if !ok {
			b.internalError("getTypeOfMappingBPOrMappingCons: expected STSpecificFieldNode")
			return st.NONE
		}
		expr := specificFieldNode.ValueExpr
		if (((expr == nil) || (expr.Kind() == st.SIMPLE_NAME_REFERENCE)) || (expr.Kind() == st.LIST_BP_OR_LIST_CONSTRUCTOR)) || (expr.Kind() == st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR) {
			return st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
		}
		return st.MAPPING_CONSTRUCTOR
	case st.SPREAD_FIELD,
		st.COMPUTED_NAME_FIELD:
		return st.MAPPING_CONSTRUCTOR
	case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR, st.SIMPLE_NAME_REFERENCE, st.QUALIFIED_NAME_REFERENCE, st.LIST_BP_OR_LIST_CONSTRUCTOR, st.REST_BINDING_PATTERN:
		fallthrough
	default:
		return st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR
	}
}

func (b *ballerinaParser) parseMappingBindingPatternOrMappingConstructorWithCloseBrace(openBrace st.STNode, members []st.STNode, closeBrace st.STNode) st.STNode {
	b.endContext()
	return st.CreateAmbiguousCollectionNode(st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR, openBrace, members, closeBrace)
}

func (b *ballerinaParser) parseAsMappingBindingPattern(openBrace st.STNode, members []st.STNode, member st.STNode) (st.STNode, []st.STNode) {
	members = append(members, member)
	members = b.getBindingPatternsList(members, false)
	b.switchContext(common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN)
	return b.parseMappingBindingPatternInner(openBrace, members, member)
}

func (b *ballerinaParser) parseListBindingPatternOrListConstructor() st.STNode {
	b.startContext(common.PARSER_RULE_CONTEXT_BRACKETED_LIST)
	openBracket := b.parseOpenBracket()
	res, _ := b.parseListBindingPatternOrListConstructorInner(openBracket, nil, false)
	return res
}

// return result, and modified memberList
func (b *ballerinaParser) parseListBindingPatternOrListConstructorInner(openBracket st.STNode, memberList []st.STNode, isRoot bool) (st.STNode, []st.STNode) {
	nextToken := b.peek()
	for !b.isBracketedListEnd(nextToken.Kind()) {
		member := b.parseListBindingPatternOrListConstructorMember()
		currentNodeType := b.getParsingNodeTypeOfListBPOrListCons(member)
		switch currentNodeType {
		case st.LIST_CONSTRUCTOR:
			return b.parseAsListConstructor(openBracket, memberList, member, isRoot)
		case st.LIST_BINDING_PATTERN:
			return b.parseAsListBindingPatternWithMemberAndRoot(openBracket, memberList, member, isRoot)
		case st.LIST_BP_OR_LIST_CONSTRUCTOR:
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

func (b *ballerinaParser) parseListBindingPatternOrListConstructorMember() st.STNode {
	nextToken := b.peek()
	switch nextToken.Kind() {
	case st.OPEN_BRACKET_TOKEN:
		return b.parseListBindingPatternOrListConstructor()
	case st.IDENTIFIER_TOKEN:
		identifier := b.parseQualifiedIdentifier(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		if b.isWildcardBP(identifier) {
			return b.getWildcardBindingPattern(identifier)
		}
		return b.parseExpressionRhs(identifier, false, false)
	case st.OPEN_BRACE_TOKEN:
		return b.parseMappingBindingPatterOrMappingConstructor()
	case st.ELLIPSIS_TOKEN:
		return b.parseRestBindingOrSpreadMember()
	default:
		if b.isValidExpressionStart(nextToken.Kind(), 1) {
			return b.parseExpression()
		}
		b.recoverWithBlockContext(b.peek(), common.PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER)
		return b.parseListBindingPatternOrListConstructorMember()
	}
}

func (b *ballerinaParser) getParsingNodeTypeOfListBPOrListCons(memberNode st.STNode) st.SyntaxKind {
	switch memberNode.Kind() {
	case st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.WILDCARD_BINDING_PATTERN:
		return st.LIST_BINDING_PATTERN
	case st.SIMPLE_NAME_REFERENCE, // member is a simple type-ref/var-ref
		st.LIST_BP_OR_LIST_CONSTRUCTOR, // member is again ambiguous
		st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		st.REST_BINDING_PATTERN:
		return st.LIST_BP_OR_LIST_CONSTRUCTOR
	default:
		return st.LIST_CONSTRUCTOR
	}
}

// Return res and modified memberList
func (b *ballerinaParser) parseAsListConstructor(openBracket st.STNode, memberList []st.STNode, member st.STNode, isRoot bool) (st.STNode, []st.STNode) {
	memberList = append(memberList, member)
	memberList = b.getExpressionList(memberList, false)
	b.switchContext(common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR)
	listMembers := b.parseListMembersInner(memberList)
	closeBracket := b.parseCloseBracket()
	listConstructor := st.CreateListConstructorExpressionNode(openBracket, listMembers, closeBracket)
	b.endContext()
	expr := b.parseExpressionRhs(listConstructor, false, true)
	if !isRoot {
		return expr, memberList
	}
	return b.parseStatementStartWithExprRhs(expr), memberList
}

func (b *ballerinaParser) parseListBindingPatternOrListConstructorWithCloseBracket(openBracket st.STNode, members []st.STNode, closeBracket st.STNode, isRoot bool) st.STNode {
	var lbpOrListCons st.STNode
	switch b.peek().Kind() {
	case st.COMMA_TOKEN,
		st.CLOSE_BRACE_TOKEN,
		st.CLOSE_BRACKET_TOKEN:
		if !isRoot {
			b.endContext()
			return st.CreateAmbiguousCollectionNode(st.LIST_BP_OR_LIST_CONSTRUCTOR, openBracket, members, closeBracket)
		}
		fallthrough
	default:
		nextTokenKind := b.peek().Kind()
		if b.isValidExprRhsStart(nextTokenKind, closeBracket.Kind()) || ((nextTokenKind == st.SEMICOLON_TOKEN) && isRoot) {
			members = b.getExpressionList(members, false)
			memberExpressions := st.CreateNodeList(members...)
			lbpOrListCons = st.CreateListConstructorExpressionNode(openBracket, memberExpressions,
				closeBracket)
			lbpOrListCons = b.parseExpressionRhs(lbpOrListCons, false, true)
			break
		}
		members = b.getBindingPatternsList(members, true)
		bindingPatternsNode := st.CreateNodeList(members...)
		lbpOrListCons = st.CreateListBindingPatternNode(openBracket, bindingPatternsNode,
			closeBracket)
	}
	b.endContext()
	if !isRoot {
		return lbpOrListCons
	}
	if lbpOrListCons.Kind() == st.LIST_BINDING_PATTERN {
		return b.parseAssignmentStmtRhs(lbpOrListCons)
	} else {
		return b.parseStatementStartWithExprRhs(lbpOrListCons)
	}
}

func (b *ballerinaParser) parseMemberRhsInStmtStartWithBrace(identifier st.STNode, colon st.STNode, secondIdentifier st.STNode, secondNameRef st.STNode) st.STNode {
	typedBPOrExpr := b.parseTypedBindingPatternOrMemberAccess(secondNameRef, false, true, common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT)
	if b.isExpression(typedBPOrExpr.Kind()) {
		return b.parseMemberWithExprInRhs(identifier, colon, secondIdentifier, typedBPOrExpr)
	}
	b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
	b.startContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
	varDeclQualifiers := []st.STNode{}
	annots := st.CreateEmptyNodeList()
	typedBP, ok := typedBPOrExpr.(*st.STTypedBindingPatternNode)
	if !ok {
		b.internalError("expected STTypedBindingPatternNode")
		return st.CreateEmptyNode()
	}
	qualifiedNameRef := b.createQualifiedNameReferenceNode(identifier, colon, secondIdentifier)
	newTypeDesc := b.mergeQualifiedNameWithTypeDesc(qualifiedNameRef, typedBP.TypeDescriptor)
	newTypeBP := st.CreateTypedBindingPatternNode(newTypeDesc, typedBP.BindingPattern)
	publicQualifier := st.CreateEmptyNode()
	res, _ := b.parseVarDeclRhsInner(annots, publicQualifier, varDeclQualifiers, newTypeBP, false)
	return res
}

func (b *ballerinaParser) parseMemberWithExprInRhs(identifier st.STNode, colon st.STNode, secondIdentifier st.STNode, memberAccessExpr st.STNode) st.STNode {
	expr := b.parseExpressionRhs(memberAccessExpr, false, true)
	switch b.peek().Kind() {
	case st.COMMA_TOKEN, st.CLOSE_BRACE_TOKEN:
		b.switchContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		readonlyKeyword := st.CreateEmptyNode()
		return st.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, expr)
	case st.EQUAL_TOKEN, st.SEMICOLON_TOKEN:
		fallthrough
	default:
		b.switchContext(common.PARSER_RULE_CONTEXT_BLOCK_STMT)
		b.startContext(common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT)
		qualifiedName := b.createQualifiedNameReferenceNode(identifier, colon, secondIdentifier)
		updatedExpr := b.mergeQualifiedNameWithExpr(qualifiedName, expr)
		return b.parseStatementStartWithExprRhs(updatedExpr)
	}
}

func (b *ballerinaParser) parseInferredTypeDescDefaultOrExpression() st.STNode {
	nextToken := b.peek()
	nextTokenKind := nextToken.Kind()
	if nextTokenKind == st.LT_TOKEN {
		return b.parseInferredTypeDescDefaultOrExpressionInner(b.consume())
	}
	if b.isValidExprStart(nextTokenKind) {
		return b.parseExpression()
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START)
	return b.parseInferredTypeDescDefaultOrExpression()
}

func (b *ballerinaParser) parseInferredTypeDescDefaultOrExpressionInner(ltToken st.STToken) st.STNode {
	nextToken := b.peek()
	if nextToken.Kind() == st.GT_TOKEN {
		return st.CreateInferredTypedescDefaultNode(ltToken, b.consume())
	}
	if b.isTypeStartingToken(nextToken.Kind()) || (nextToken.Kind() == st.AT_TOKEN) {
		b.startContext(common.PARSER_RULE_CONTEXT_TYPE_CAST)
		expr := b.parseTypeCastExprInner(ltToken, true, false, false)
		return b.parseExpressionRhs(expr, true, false)
	}
	b.recoverWithBlockContext(nextToken, common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END)
	return b.parseInferredTypeDescDefaultOrExpressionInner(ltToken)
}

func (b *ballerinaParser) mergeQualifiedNameWithExpr(qualifiedName st.STNode, exprOrAction st.STNode) st.STNode {
	switch exprOrAction.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		return qualifiedName
	case st.BINARY_EXPRESSION:
		binaryExpr, ok := exprOrAction.(*st.STBinaryExpressionNode)
		if !ok {
			b.internalError("expected STBinaryExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, binaryExpr.LhsExpr)
		return st.CreateBinaryExpressionNode(binaryExpr.Kind(), newLhsExpr, binaryExpr.Operator,
			binaryExpr.RhsExpr)
	case st.FIELD_ACCESS:
		fieldAccess, ok := exprOrAction.(*st.STFieldAccessExpressionNode)
		if !ok {
			b.internalError("expected STFieldAccessExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, fieldAccess.Expression)
		return st.CreateFieldAccessExpressionNode(newLhsExpr, fieldAccess.DotToken,
			fieldAccess.FieldName)
	case st.INDEXED_EXPRESSION:
		memberAccess, ok := exprOrAction.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("expected STIndexedExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, memberAccess.ContainerExpression)
		return st.CreateIndexedExpressionNode(newLhsExpr, memberAccess.OpenBracket,
			memberAccess.KeyExpression, memberAccess.CloseBracket)
	case st.TYPE_TEST_EXPRESSION:
		typeTest, ok := exprOrAction.(*st.STTypeTestExpressionNode)
		if !ok {
			b.internalError("expected STTypeTestExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, typeTest.Expression)
		return st.CreateTypeTestExpressionNode(newLhsExpr, typeTest.IsKeyword,
			typeTest.TypeDescriptor)
	case st.ANNOT_ACCESS:
		annotAccess, ok := exprOrAction.(*st.STAnnotAccessExpressionNode)
		if !ok {
			b.internalError("expected STAnnotAccessExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, annotAccess.Expression)
		return st.CreateFieldAccessExpressionNode(newLhsExpr, annotAccess.AnnotChainingToken,
			annotAccess.AnnotTagReference)
	case st.OPTIONAL_FIELD_ACCESS:
		optionalFieldAccess, ok := exprOrAction.(*st.STOptionalFieldAccessExpressionNode)
		if !ok {
			b.internalError("expected STOptionalFieldAccessExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, optionalFieldAccess.Expression)
		return st.CreateFieldAccessExpressionNode(newLhsExpr,
			optionalFieldAccess.OptionalChainingToken, optionalFieldAccess.FieldName)
	case st.CONDITIONAL_EXPRESSION:
		conditionalExpr, ok := exprOrAction.(*st.STConditionalExpressionNode)
		if !ok {
			b.internalError("expected STConditionalExpressionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, conditionalExpr.LhsExpression)
		return st.CreateConditionalExpressionNode(newLhsExpr, conditionalExpr.QuestionMarkToken,
			conditionalExpr.MiddleExpression, conditionalExpr.ColonToken, conditionalExpr.EndExpression)
	case st.REMOTE_METHOD_CALL_ACTION:
		remoteCall, ok := exprOrAction.(*st.STRemoteMethodCallActionNode)
		if !ok {
			b.internalError("expected STRemoteMethodCallActionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, remoteCall.Expression)
		return st.CreateRemoteMethodCallActionNode(newLhsExpr, remoteCall.RightArrowToken,
			remoteCall.MethodName, remoteCall.OpenParenToken, remoteCall.Arguments,
			remoteCall.CloseParenToken)
	case st.ASYNC_SEND_ACTION:
		asyncSend, ok := exprOrAction.(*st.STAsyncSendActionNode)
		if !ok {
			b.internalError("expected STAsyncSendActionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, asyncSend.Expression)
		return st.CreateAsyncSendActionNode(newLhsExpr, asyncSend.RightArrowToken,
			asyncSend.PeerWorker)
	case st.SYNC_SEND_ACTION:
		syncSend, ok := exprOrAction.(*st.STSyncSendActionNode)
		if !ok {
			b.internalError("expected STSyncSendActionNode")
			return st.CreateEmptyNode()
		}
		newLhsExpr := b.mergeQualifiedNameWithExpr(qualifiedName, syncSend.Expression)
		return st.CreateAsyncSendActionNode(newLhsExpr, syncSend.SyncSendToken, syncSend.PeerWorker)
	case st.FUNCTION_CALL:
		funcCall, ok := exprOrAction.(*st.STFunctionCallExpressionNode)
		if !ok {
			b.internalError("expected STFunctionCallExpressionNode")
			return st.CreateEmptyNode()
		}
		return st.CreateFunctionCallExpressionNode(qualifiedName, funcCall.OpenParenToken,
			funcCall.Arguments, funcCall.CloseParenToken)
	default:
		return exprOrAction
	}
}

func (b *ballerinaParser) mergeQualifiedNameWithTypeDesc(qualifiedName st.STNode, typeDesc st.STNode) st.STNode {
	switch typeDesc.Kind() {
	case st.SIMPLE_NAME_REFERENCE:
		return qualifiedName
	case st.ARRAY_TYPE_DESC:
		arrayTypeDesc, ok := typeDesc.(*st.STArrayTypeDescriptorNode)
		if !ok {
			b.internalError("expected STArrayTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newMemberType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, arrayTypeDesc.MemberTypeDesc)
		return st.CreateArrayTypeDescriptorNode(newMemberType, arrayTypeDesc.Dimensions)
	case st.UNION_TYPE_DESC:
		unionTypeDesc, ok := typeDesc.(*st.STUnionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STUnionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newlhsType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, unionTypeDesc.LeftTypeDesc)
		return b.mergeTypesWithUnion(newlhsType, unionTypeDesc.PipeToken, unionTypeDesc.RightTypeDesc)
	case st.INTERSECTION_TYPE_DESC:
		intersectionTypeDesc, ok := typeDesc.(*st.STIntersectionTypeDescriptorNode)
		if !ok {
			b.internalError("expected *st.STIntersectionTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newlhsType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, intersectionTypeDesc.LeftTypeDesc)
		return b.mergeTypesWithIntersection(newlhsType, intersectionTypeDesc.BitwiseAndToken,
			intersectionTypeDesc.RightTypeDesc)
	case st.OPTIONAL_TYPE_DESC:
		optionalType, ok := typeDesc.(*st.STOptionalTypeDescriptorNode)
		if !ok {
			b.internalError("expected STOptionalTypeDescriptorNode")
			return st.CreateEmptyNode()
		}
		newMemberType := b.mergeQualifiedNameWithTypeDesc(qualifiedName, optionalType.TypeDescriptor)
		return st.CreateOptionalTypeDescriptorNode(newMemberType, optionalType.QuestionMarkToken)
	default:
		return typeDesc
	}
}

func (b *ballerinaParser) getTupleMemberList(ambiguousList []st.STNode) []st.STNode {
	var tupleMemberList []st.STNode
	for _, item := range ambiguousList {
		if item.Kind() == st.COMMA_TOKEN {
			tupleMemberList = append(tupleMemberList, item)
		} else {
			tupleMemberList = append(tupleMemberList,
				st.CreateMemberTypeDescriptorNode(st.CreateEmptyNodeList(),
					b.getTypeDescFromExpr(item)))
		}
	}
	return tupleMemberList
}

func (b *ballerinaParser) getTypeDescFromExpr(expression st.STNode) st.STNode {
	if b.isDefiniteTypeDesc(expression.Kind()) || (expression.Kind() == st.COMMA_TOKEN) {
		return expression
	}
	switch expression.Kind() {
	case st.INDEXED_EXPRESSION:
		indexedExpr, ok := expression.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("getTypeDescFromExpr: expected STIndexedExpressionNode")
			return st.CreateEmptyNode()
		}
		return b.parseArrayTypeDescriptorNode(*indexedExpr)
	case st.NUMERIC_LITERAL,
		st.BOOLEAN_LITERAL,
		st.STRING_LITERAL,
		st.NULL_LITERAL,
		st.UNARY_EXPRESSION:
		return st.CreateSingletonTypeDescriptorNode(expression)
	case st.TYPE_REFERENCE_TYPE_DESC:
		typeRefNode, ok := expression.(*st.STTypeReferenceTypeDescNode)
		if !ok {
			b.internalError("getTypeDescFromExpr: expected STTypeReferenceTypeDescNode")
			return st.CreateEmptyNode()
		}
		return typeRefNode.TypeRef
	case st.BRACED_EXPRESSION:
		bracedExpr, ok := expression.(*st.STBracedExpressionNode)
		if !ok {
			b.internalError("expected STBracedExpressionNode")
			return st.CreateEmptyNode()
		}
		typeDesc := b.getTypeDescFromExpr(bracedExpr.Expression)
		return st.CreateParenthesisedTypeDescriptorNode(bracedExpr.OpenParen, typeDesc,
			bracedExpr.CloseParen)
	case st.NIL_LITERAL:
		nilLiteral, ok := expression.(*st.STNilLiteralNode)
		if !ok {
			b.internalError("expected STNilLiteralNode")
			return st.CreateEmptyNode()
		}
		return st.CreateNilTypeDescriptorNode(nilLiteral.OpenParenToken, nilLiteral.CloseParenToken)
	case st.BRACKETED_LIST,
		st.LIST_BP_OR_LIST_CONSTRUCTOR,
		st.TUPLE_TYPE_DESC_OR_LIST_CONST:
		innerList, ok := expression.(*st.STAmbiguousCollectionNode)
		if !ok {
			b.internalError("expected STAmbiguousCollectionNode")
			return st.CreateEmptyNode()
		}
		memberTypeDescs := st.CreateNodeList(b.getTupleMemberList(innerList.Members)...)
		return st.CreateTupleTypeDescriptorNode(innerList.CollectionStartToken, memberTypeDescs,
			innerList.CollectionEndToken)
	case st.BINARY_EXPRESSION:
		binaryExpr, ok := expression.(*st.STBinaryExpressionNode)
		if !ok {
			b.internalError("expected STBinaryExpressionNode")
			return st.CreateEmptyNode()
		}
		switch binaryExpr.Operator.Kind() {
		case st.PIPE_TOKEN,
			st.BITWISE_AND_TOKEN:
			lhsTypeDesc := b.getTypeDescFromExpr(binaryExpr.LhsExpr)
			rhsTypeDesc := b.getTypeDescFromExpr(binaryExpr.RhsExpr)
			return b.mergeTypes(lhsTypeDesc, binaryExpr.Operator, rhsTypeDesc)
		default:
			break
		}
		return expression
	case st.SIMPLE_NAME_REFERENCE,
		st.QUALIFIED_NAME_REFERENCE:
		return expression
	default:
		var simpleTypeDescIdentifier st.STNode
		simpleTypeDescIdentifier = st.CreateMissingTokenWithDiagnostics(
			st.IDENTIFIER_TOKEN, &common.ERROR_MISSING_TYPE_DESC)
		simpleTypeDescIdentifier = st.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(simpleTypeDescIdentifier,
			expression)
		return st.CreateSimpleNameReferenceNode(simpleTypeDescIdentifier)
	}
}

func (b *ballerinaParser) getBindingPatternsList(ambibuousList []st.STNode, isListBP bool) []st.STNode {
	var bindingPatterns []st.STNode
	for _, item := range ambibuousList {
		bindingPatterns = append(bindingPatterns, b.getBindingPattern(item, isListBP))
	}
	return bindingPatterns
}

func (b *ballerinaParser) getBindingPattern(ambiguousNode st.STNode, isListBP bool) st.STNode {
	errorCode := common.ERROR_INVALID_BINDING_PATTERN
	if b.isEmpty(ambiguousNode) {
		return nil
	}
	switch ambiguousNode.Kind() {
	case st.WILDCARD_BINDING_PATTERN,
		st.CAPTURE_BINDING_PATTERN,
		st.LIST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN,
		st.ERROR_BINDING_PATTERN,
		st.REST_BINDING_PATTERN,
		st.FIELD_BINDING_PATTERN,
		st.NAMED_ARG_BINDING_PATTERN,
		st.COMMA_TOKEN:
		return ambiguousNode
	case st.SIMPLE_NAME_REFERENCE:
		simpleNameNode, ok := ambiguousNode.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("getBindingPattern: expected STSimpleNameReferenceNode")
			return st.CreateEmptyNode()
		}
		varName := simpleNameNode.Name
		return b.createCaptureOrWildcardBP(varName)
	case st.QUALIFIED_NAME_REFERENCE:
		if isListBP {
			errorCode = common.ERROR_FIELD_BP_INSIDE_LIST_BP
			break
		}
		qualifiedName, ok := ambiguousNode.(*st.STQualifiedNameReferenceNode)
		if !ok {
			b.internalError("expected STQualifiedNameReferenceNode")
			return st.CreateEmptyNode()
		}
		fieldName := st.CreateSimpleNameReferenceNode(qualifiedName.ModulePrefix)
		return st.CreateFieldBindingPatternFullNode(fieldName, qualifiedName.Colon,
			b.createCaptureOrWildcardBP(qualifiedName.Identifier))
	case st.BRACKETED_LIST,
		st.LIST_BP_OR_LIST_CONSTRUCTOR:
		innerList, ok := ambiguousNode.(*st.STAmbiguousCollectionNode)
		if !ok {
			b.internalError("expected STAmbiguousCollectionNode")
			return st.CreateEmptyNode()
		}
		memberBindingPatterns := st.CreateNodeList(b.getBindingPatternsList(innerList.Members, true)...)
		return st.CreateListBindingPatternNode(innerList.CollectionStartToken, memberBindingPatterns,
			innerList.CollectionEndToken)
	case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		innerList, ok := ambiguousNode.(*st.STAmbiguousCollectionNode)
		if !ok {
			b.internalError("expected STAmbiguousCollectionNode")
			return st.CreateEmptyNode()
		}
		var bindingPatterns []st.STNode
		i := 0
		for ; i < len(innerList.Members); i++ {
			bp := b.getBindingPattern(innerList.Members[i], false)
			bindingPatterns = append(bindingPatterns, bp)
			if bp.Kind() == st.REST_BINDING_PATTERN {
				break
			}
		}
		memberBindingPatterns := st.CreateNodeList(bindingPatterns...)
		return st.CreateMappingBindingPatternNode(innerList.CollectionStartToken,
			memberBindingPatterns, innerList.CollectionEndToken)
	case st.SPECIFIC_FIELD:
		field, ok := ambiguousNode.(*st.STSpecificFieldNode)
		if !ok {
			b.internalError("expected STSpecificFieldNode")
			return st.CreateEmptyNode()
		}
		fieldName := st.CreateSimpleNameReferenceNode(field.FieldName)
		if field.ValueExpr == nil {
			return st.CreateFieldBindingPatternVarnameNode(fieldName)
		}
		return st.CreateFieldBindingPatternFullNode(fieldName, field.Colon,
			b.getBindingPattern(field.ValueExpr, false))
	case st.ERROR_CONSTRUCTOR:
		errorCons, ok := ambiguousNode.(*st.STErrorConstructorExpressionNode)
		if !ok {
			b.internalError("expected STErrorConstructorExpressionNode")
			return st.CreateEmptyNode()
		}
		args := errorCons.Arguments
		size := args.BucketCount()
		var bindingPatterns []st.STNode
		i := 0
		for ; i < size; i++ {
			arg := args.ChildInBucket(i)
			bindingPatterns = append(bindingPatterns, b.getBindingPattern(arg, false))
		}
		argListBindingPatterns := st.CreateNodeList(bindingPatterns...)
		return st.CreateErrorBindingPatternNode(errorCons.ErrorKeyword, errorCons.TypeReference,
			errorCons.OpenParenToken, argListBindingPatterns, errorCons.CloseParenToken)
	case st.POSITIONAL_ARG:
		positionalArg, ok := ambiguousNode.(*st.STPositionalArgumentNode)
		if !ok {
			b.internalError("expected STPositionalArgumentNode")
			return st.CreateEmptyNode()
		}
		return b.getBindingPattern(positionalArg.Expression, false)
	case st.NAMED_ARG:
		namedArg, nameOk := ambiguousNode.(*st.STNamedArgumentNode)
		if !nameOk {
			b.internalError("exprected STNamedArgumentNode")
			return st.CreateEmptyNode()
		}
		argNameNode, ok := namedArg.ArgumentName.(*st.STSimpleNameReferenceNode)
		if !ok {
			b.internalError("getBindingPattern: expected STSimpleNameReferenceNode for named argument")
			return st.CreateEmptyNode()
		}
		bindingPatternArgName := argNameNode.Name
		return st.CreateNamedArgBindingPatternNode(bindingPatternArgName, namedArg.EqualsToken,
			b.getBindingPattern(namedArg.Expression, false))
	case st.REST_ARG:
		restArg, ok := ambiguousNode.(*st.STRestArgumentNode)
		if !ok {
			b.internalError("expected STRestArgumentNode")
			return st.CreateEmptyNode()
		}
		return st.CreateRestBindingPatternNode(restArg.Ellipsis, restArg.Expression)
	}
	var identifier st.STNode
	identifier = st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil)
	identifier = st.CloneWithLeadingInvalidNodeMinutiae(identifier, ambiguousNode, &errorCode)
	return st.CreateCaptureBindingPatternNode(identifier)
}

func (b *ballerinaParser) getExpressionList(ambibuousList []st.STNode, isMappingConstructor bool) []st.STNode {
	var exprList []st.STNode
	for _, item := range ambibuousList {
		exprList = append(exprList, b.getExpressionInner(item, isMappingConstructor))
	}
	return exprList
}

func (b *ballerinaParser) getExpression(ambiguousNode st.STNode) st.STNode {
	return b.getExpressionInner(ambiguousNode, false)
}

func (b *ballerinaParser) getExpressionInner(ambiguousNode st.STNode, isInMappingConstructor bool) st.STNode {
	if ((b.isEmpty(ambiguousNode) || (b.isDefiniteExpr(ambiguousNode.Kind()) && (ambiguousNode.Kind() != st.INDEXED_EXPRESSION))) || b.isDefiniteAction(ambiguousNode.Kind())) || (ambiguousNode.Kind() == st.COMMA_TOKEN) {
		return ambiguousNode
	}
	switch ambiguousNode.Kind() {
	case st.BRACKETED_LIST, st.LIST_BP_OR_LIST_CONSTRUCTOR, st.TUPLE_TYPE_DESC_OR_LIST_CONST:
		innerList, ok := ambiguousNode.(*st.STAmbiguousCollectionNode)
		if !ok {
			b.internalError("getExpressionInner: expected STAmbiguousCollectionNode")
			return st.CreateEmptyNode()
		}
		memberExprs := st.CreateNodeList(b.getExpressionList(innerList.Members, false)...)
		return st.CreateListConstructorExpressionNode(innerList.CollectionStartToken, memberExprs,
			innerList.CollectionEndToken)

	case st.MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		innerList, ok := ambiguousNode.(*st.STAmbiguousCollectionNode)
		if !ok {
			b.internalError("getExpressionInner: expected STAmbiguousCollectionNode")
			return st.CreateEmptyNode()
		}
		var fieldList []st.STNode
		i := 0
		for ; i < len(innerList.Members); i++ {
			field := innerList.Members[i]
			var fieldNode st.STNode
			if field.Kind() == st.QUALIFIED_NAME_REFERENCE {
				qualifiedNameRefNode, ok := field.(*st.STQualifiedNameReferenceNode)
				if !ok {
					b.internalError("getExpressionInner: expected STQualifiedNameReferenceNode")
					return st.CreateEmptyNode()
				}
				readOnlyKeyword := st.CreateEmptyNode()
				fieldName := qualifiedNameRefNode.ModulePrefix
				colon := qualifiedNameRefNode.Colon
				valueExpr := b.getExpression(qualifiedNameRefNode.Identifier)
				fieldNode = st.CreateSpecificFieldNode(readOnlyKeyword, fieldName, colon, valueExpr)
			} else {
				fieldNode = b.getExpressionInner(field, true)
			}
			fieldList = append(fieldList, fieldNode)
		}
		fields := st.CreateNodeList(fieldList...)
		return st.CreateMappingConstructorExpressionNode(innerList.CollectionStartToken, fields,

			innerList.CollectionEndToken)

	case st.REST_BINDING_PATTERN:
		restBindingPattern, ok := ambiguousNode.(*st.STRestBindingPatternNode)
		if !ok {
			b.internalError("getExpressionInner: expected STRestBindingPatternNode")
			return st.CreateEmptyNode()
		}
		if isInMappingConstructor {
			return st.CreateSpreadFieldNode(restBindingPattern.EllipsisToken,
				restBindingPattern.VariableName)
		}

		return st.CreateSpreadMemberNode(restBindingPattern.EllipsisToken,

			restBindingPattern.VariableName)

	case st.SPECIFIC_FIELD:
		field, ok := ambiguousNode.(*st.STSpecificFieldNode)
		if !ok {
			b.internalError("getExpressionInner: expected STSpecificFieldNode")
			return st.CreateEmptyNode()
		}
		return st.CreateSpecificFieldNode(field.ReadonlyKeyword, field.FieldName, field.Colon,

			b.getExpression(field.ValueExpr))

	case st.ERROR_CONSTRUCTOR:
		errorCons, ok := ambiguousNode.(*st.STErrorConstructorExpressionNode)
		if !ok {
			b.internalError("getExpressionInner: expected STErrorConstructorExpressionNode")
			return st.CreateEmptyNode()
		}
		errorArgs := b.getErrorArgList(errorCons.Arguments)
		return st.CreateErrorConstructorExpressionNode(errorCons.ErrorKeyword,
			errorCons.TypeReference, errorCons.OpenParenToken, errorArgs, errorCons.CloseParenToken)

	case st.IDENTIFIER_TOKEN:
		return st.CreateSimpleNameReferenceNode(ambiguousNode)
	case st.INDEXED_EXPRESSION:
		indexedExpressionNode, ok := ambiguousNode.(*st.STIndexedExpressionNode)
		if !ok {
			b.internalError("getExpressionInner: expected STIndexedExpressionNode")
			return st.CreateEmptyNode()
		}
		keys, ok := indexedExpressionNode.KeyExpression.(*st.STNodeList)
		if !ok {
			b.internalError("getExpressionInner: expected STNodeList")
			return st.CreateEmptyNode()
		}
		if !keys.IsEmpty() {
			return ambiguousNode
		}
		lhsExpr := indexedExpressionNode.ContainerExpression
		openBracket := indexedExpressionNode.OpenBracket
		closeBracket := indexedExpressionNode.CloseBracket
		missingVarRef := st.CreateSimpleNameReferenceNode(st.CreateMissingToken(st.IDENTIFIER_TOKEN, nil))
		keyExpr := st.CreateNodeList(missingVarRef)
		closeBracket = st.AddDiagnostic(closeBracket,
			&common.ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR)
		return st.CreateIndexedExpressionNode(lhsExpr, openBracket, keyExpr, closeBracket)
	case st.SIMPLE_NAME_REFERENCE, st.QUALIFIED_NAME_REFERENCE, st.COMPUTED_NAME_FIELD, st.SPREAD_FIELD, st.SPREAD_MEMBER:
		return ambiguousNode
	default:
		var simpleVarRef st.STNode
		simpleVarRef = st.CreateMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN,
			&common.ERROR_MISSING_EXPRESSION)
		simpleVarRef = st.CloneWithTrailingInvalidNodeMinutiaeWithoutDiagnostics(simpleVarRef, ambiguousNode)
		return st.CreateSimpleNameReferenceNode(simpleVarRef)
	}
}

func (b *ballerinaParser) getMappingField(identifier st.STNode, colon st.STNode, bindingPatternOrExpr st.STNode) st.STNode {
	simpleNameRef := st.CreateSimpleNameReferenceNode(identifier)
	switch bindingPatternOrExpr.Kind() {
	case st.LIST_BINDING_PATTERN,
		st.MAPPING_BINDING_PATTERN:
		return st.CreateFieldBindingPatternFullNode(simpleNameRef, colon, bindingPatternOrExpr)
	case st.LIST_CONSTRUCTOR, st.MAPPING_CONSTRUCTOR:
		readonlyKeyword := st.CreateEmptyNode()
		return st.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, bindingPatternOrExpr)
	default:
		readonlyKeyword := st.CreateEmptyNode()
		return st.CreateSpecificFieldNode(readonlyKeyword, identifier, colon, bindingPatternOrExpr)
	}
}

func (b *ballerinaParser) recoverWithBlockContext(nextToken st.STToken, currentCtx common.ParserRuleContext) *solution {
	if b.isInsideABlock(nextToken) {
		return b.recover(nextToken, currentCtx, true)
	} else {
		return b.recover(nextToken, currentCtx, false)
	}
}

func (b *ballerinaParser) isInsideABlock(nextToken st.STToken) bool {
	if nextToken.Kind() != st.CLOSE_BRACE_TOKEN {
		return false
	}
	return slices.ContainsFunc(b.errorHandler.GetContextStack(), b.isBlockContext)
}

func (b *ballerinaParser) isBlockContext(ctx common.ParserRuleContext) bool {
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

func (b *ballerinaParser) isSpecialMethodName(token st.STToken) bool {
	return (((token.Kind() == st.MAP_KEYWORD) || (token.Kind() == st.START_KEYWORD)) || (token.Kind() == st.JOIN_KEYWORD))
}

// GetSyntaxTree parses content into a syntax tree, attributing it to fileName
// (used for diagnostics and the syntax tree's text document).
func GetSyntaxTree(ctx *context.CompilerContext, fileName string, content string) (*st.SyntaxTree, error) {
	textDocument := text.TextDocumentFromText(content)
	ctx.DiagnosticEnv().RegisterFile(fileName, textDocument)
	reader := text.CharReaderFromText(content)
	lexer := newLexer(ctx, fileName, reader)
	tokenReader := createTokenReader(ctx, fileName, lexer)
	ballerinaParser := newBallerinaParserFromTokenReader(tokenReader)
	root := ballerinaParser.Parse()
	rootNode, ok := root.(*st.STModulePart)
	if !ok {
		const message = "parser root is not an STModulePart"
		ctx.InternalError(message, diagnostics.NewLocation(ctx.DiagnosticEnv(), fileName, 0, 0))
		return nil, errors.New(message)
	}
	moduleNode := st.CreateUnlinkedFacade[*st.STModulePart, *st.ModulePart](rootNode)
	syntaxTree := st.NewSyntaxTreeFromNodeTextDocument(moduleNode, textDocument, fileName, false)
	return &syntaxTree, nil
}
