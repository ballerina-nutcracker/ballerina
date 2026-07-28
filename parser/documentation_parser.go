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

package parser

import (
	"strings"

	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

// DocumentationParser is a parser for Ballerina documentation (markdown).
// Ballerina flavored markdown (BFM) is supported by the documentation.
// There is no error handler attached to this parser.
// In case of an error, simply missing token will be inserted.
type documentationParser struct {
	abstractParser
}

func newDocumentationParser(tokenReader *tokenReader) *documentationParser {
	parser := &documentationParser{}
	parser.abstractParser = newAbstractParserFromTokenReader(tokenReader)
	return parser
}

func (p *documentationParser) Parse() st.STNode {
	return p.parseDocumentationLines()
}

func (p *documentationParser) parseDocumentationLines() st.STNode {
	docLines := make([]st.STNode, 0)
	nextToken := p.peek()
	for nextToken != nil && nextToken.Kind() == st.HASH_TOKEN {
		docLines = append(docLines, p.parseSingleDocumentationLine())
		nextToken = p.peek()
	}
	return st.CreateNodeList(docLines...)
}

func (p *documentationParser) parseSingleDocumentationLine() st.STNode {
	hashToken := p.consume()
	nextToken := p.peek()
	if nextToken == nil {
		return p.createMarkdownDocumentationLineNode(hashToken, st.CreateEmptyNodeList())
	}

	switch nextToken.Kind() {
	case st.PLUS_TOKEN:
		return p.parseParameterDocumentationLine(hashToken)
	case st.DEPRECATION_LITERAL:
		return p.parseDeprecationDocumentationLine(hashToken)
	case st.TRIPLE_BACKTICK_TOKEN, st.DOUBLE_BACKTICK_TOKEN:
		return p.parseCodeBlockOrInlineCodeRef(hashToken)
	default:
		return p.parseDocumentationLine(hashToken)
	}
}

func (p *documentationParser) parseCodeBlockOrInlineCodeRef(startLineHash st.STNode) st.STNode {
	startBacktick := p.consume()
	nextToken := p.peek()
	if nextToken == nil || !p.isInlineCodeRef(nextToken.Kind()) {
		return p.parseCodeBlock(startLineHash, startBacktick)
	}

	inlineCodeNode := p.parseInlineCode(startBacktick)
	docElements := []st.STNode{inlineCodeNode}
	p.parseDocElements(&docElements)
	docElementList := st.CreateNodeList(docElements...)
	return p.createMarkdownReferenceDocumentationLineNode(startLineHash, docElementList)
}

func (p *documentationParser) isInlineCodeRef(nextTokenKind st.SyntaxKind) bool {
	nextNext := p.getNextNextToken()
	switch nextTokenKind {
	case st.HASH_TOKEN:
		return nextNext != nil && nextNext.Kind() == st.DOCUMENTATION_DESCRIPTION
	case st.CODE_CONTENT:
		if nextNext == nil {
			return true
		}
		return nextNext.Kind() != st.HASH_TOKEN
	default:
		return true
	}
}

func (p *documentationParser) parseDeprecationDocumentationLine(hashToken st.STNode) st.STNode {
	deprecationLiteral := p.consume()
	docElements := p.parseDocumentationElements()
	docElements = append([]st.STNode{deprecationLiteral}, docElements...)
	docElementList := st.CreateNodeList(docElements...)
	return p.createMarkdownDeprecationDocumentationLineNode(hashToken, docElementList)
}

func (p *documentationParser) parseDocumentationLine(hashToken st.STNode) st.STNode {
	docElements := p.parseDocumentationElements()
	docElementList := st.CreateNodeList(docElements...)

	switch len(docElements) {
	case 0:
		// When documentation line is only a `#` token
		return p.createMarkdownDocumentationLineNode(hashToken, docElementList)
	case 1:
		docElement := docElements[0]
		if docElement.Kind() == st.DOCUMENTATION_DESCRIPTION {
			return p.createMarkdownDocumentationLineNode(hashToken, docElementList)
		}
		fallthrough
	default:
		return p.createMarkdownReferenceDocumentationLineNode(hashToken, docElementList)
	}
}

func (p *documentationParser) parseDocumentationElements() []st.STNode {
	docElements := make([]st.STNode, 0)
	p.parseDocElements(&docElements)
	return docElements
}

func (p *documentationParser) parseDocElements(docElements *[]st.STNode) {
	var docElement st.STNode
	var referenceType st.STNode

	nextToken := p.peek()
	for nextToken != nil && !p.isEndOfIntermediateDocumentation(nextToken.Kind()) {
		switch nextToken.Kind() {
		case st.DOCUMENTATION_DESCRIPTION:
			docElement = p.consume()
		case st.CODE_CONTENT:
			token := p.consume()
			docElement = p.convertToDocDescriptionToken(token)
		case st.DOUBLE_BACKTICK_TOKEN, st.TRIPLE_BACKTICK_TOKEN:
			docElement = p.parseInlineCode(p.consume())
		case st.BACKTICK_TOKEN:
			referenceType = st.CreateEmptyNode()
			docElement = p.parseBallerinaNameRefOrInlineCodeRef(referenceType)
		default:
			if p.isDocumentReferenceType(nextToken.Kind()) {
				referenceType = p.consume()
				docElement = p.parseBallerinaNameRefOrInlineCodeRef(referenceType)
			} else {
				// We should not reach here.
				p.consume()
				nextToken = p.peek()
				continue
			}
		}

		*docElements = append(*docElements, docElement)
		nextToken = p.peek()
	}
}

func (p *documentationParser) convertToDocDescriptionToken(token st.STToken) st.STNode {
	return st.CreateLiteralValueToken(st.DOCUMENTATION_DESCRIPTION, token.Text(),
		token.LeadingMinutiae(), token.TrailingMinutiae())
}

func (p *documentationParser) convertToCodeContentToken(token st.STToken) st.STNode {
	return st.CreateLiteralValueToken(st.CODE_CONTENT, token.Text(),
		token.LeadingMinutiae(), token.TrailingMinutiae())
}

func (p *documentationParser) parseInlineCode(startBacktick st.STNode) st.STNode {
	codeDescription := p.parseInlineCodeContentToken()
	endBacktick := p.parseCodeEndBacktick(startBacktick.Kind())
	return p.createInlineCodeReferenceNode(startBacktick, codeDescription, endBacktick)
}

func (p *documentationParser) parseInlineCodeContentToken() st.STNode {
	token := p.peek()
	if token == nil {
		return p.createMissingTokenWithDiagnostics(st.CODE_CONTENT)
	}

	if token.Kind() == st.CODE_CONTENT {
		return p.consume()
	} else if token.Kind() == st.DOCUMENTATION_DESCRIPTION {
		token = p.consume()
		return p.convertToCodeContentToken(token)
	} else {
		return p.createMissingTokenWithDiagnostics(st.CODE_CONTENT)
	}
}

func (p *documentationParser) parseCodeBlock(startLineHash st.STNode, startBacktick st.STNode) st.STNode {
	langAttribute := p.parseOptionalLangAttributeToken()
	codeLines := p.parseCodeLines()
	endLineHash := p.parseHashToken()
	endBacktick := p.parseCodeEndBacktick(startBacktick.Kind())

	// Handle any invalid tokens after the code block
	for p.peek() != nil && !p.isEndOfIntermediateDocumentation(p.peek().Kind()) {
		invalidToken := p.consume()
		endBacktick = st.CloneWithTrailingInvalidNodeMinutiae(endBacktick, invalidToken,
			&common.WARNING_CANNOT_HAVE_DOCUMENTATION_INLINE_WITH_A_CODE_REFERENCE_BLOCK)
	}
	return p.createMarkdownCodeBlockNode(startLineHash, startBacktick, langAttribute, codeLines, endLineHash, endBacktick)
}

func (p *documentationParser) parseOptionalLangAttributeToken() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.CODE_CONTENT {
		return p.consume()
	} else {
		return st.CreateEmptyNode()
	}
}

func (p *documentationParser) parseCodeLines() st.STNode {
	codeLineList := make([]st.STNode, 0)
	for !p.isEndOfCodeLines() {
		codeLineNode := p.parseCodeLine()
		codeLineList = append(codeLineList, codeLineNode)
	}
	return st.CreateNodeList(codeLineList...)
}

func (p *documentationParser) parseCodeLine() st.STNode {
	hash := p.parseHashToken()
	var codeDescription st.STNode
	nextToken := p.peek()
	if nextToken != nil && nextToken.Kind() == st.HASH_TOKEN {
		// We reach here, when the code line is empty
		codeDescription = p.createEmptyCodeContentToken()
	} else {
		codeDescription = p.parseInlineCodeContentToken()
	}
	return p.createMarkdownCodeLineNode(hash, codeDescription)
}

func (p *documentationParser) createEmptyCodeContentToken() st.STNode {
	emptyMinutiae := st.CreateEmptyNodeList()
	return st.CreateLiteralValueToken(st.CODE_CONTENT, "", emptyMinutiae, emptyMinutiae)
}

func (p *documentationParser) parseHashToken() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.HASH_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.HASH_TOKEN)
	}
}

func (p *documentationParser) parseCodeEndBacktick(backtickKind st.SyntaxKind) st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == backtickKind {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(backtickKind)
	}
}

func (p *documentationParser) isEndOfCodeLines() bool {
	nextToken := p.peek()
	if nextToken == nil {
		return true
	}
	if nextToken.Kind() == st.HASH_TOKEN {
		nextNextToken := p.getNextNextToken()
		if nextNextToken == nil {
			return true
		}
		switch nextNextToken.Kind() {
		case st.CODE_CONTENT, st.HASH_TOKEN:
			return false
		default:
			return true
		}
	}
	return true
}

func (p *documentationParser) parseBallerinaNameRefOrInlineCodeRef(referenceType st.STNode) st.STNode {
	startBacktick := p.parseBacktickToken()
	isCodeRef := false
	var contentToken st.STNode
	referenceGenre := p.getReferenceGenre(referenceType)
	if p.isBallerinaNameRefTokenSequence(referenceGenre) {
		contentToken = p.parseNameReferenceContent()
	} else {
		contentToken = p.combineAndCreateCodeContentToken()
		if referenceGenre != referenceGenreNoKey {
			contentToken = st.AddDiagnostic(contentToken, &common.WARNING_INVALID_BALLERINA_NAME_REFERENCE, contentToken.(st.STToken).Text())
		} else {
			isCodeRef = true
		}
	}

	endBacktick := p.parseBacktickToken()

	if isCodeRef {
		return p.createInlineCodeReferenceNode(startBacktick, contentToken, endBacktick)
	} else {
		return p.createBallerinaNameReferenceNode(referenceType, startBacktick, contentToken, endBacktick)
	}
}

type referenceGenre int

const (
	referenceGenreNoKey referenceGenre = iota
	referenceGenreSpecialKey
	referenceGenreFunctionKey
)

type lookahead struct {
	offset int
}

func (p *documentationParser) isBallerinaNameRefTokenSequence(refGenre referenceGenre) bool {
	hasMatch := false
	lookahead := &lookahead{offset: 1}

	switch refGenre {
	case referenceGenreSpecialKey:
		// Look for x, m:x match
		hasMatch = p.hasQualifiedIdentifier(lookahead)
	case referenceGenreFunctionKey:
		// Look for x, m:x, x(), m:x(), T.y(), m:T.y() match
		hasMatch = p.hasBacktickExpr(lookahead, true)
	case referenceGenreNoKey:
		// Look for x(), m:x(), T.y(), m:T.y() match
		hasMatch = p.hasBacktickExpr(lookahead, false)
	}

	if !hasMatch {
		return false
	}

	peekToken := p.peekN(lookahead.offset)
	return peekToken != nil && peekToken.Kind() == st.BACKTICK_TOKEN
}

func (p *documentationParser) hasBacktickExpr(lookahead *lookahead, isFunctionKey bool) bool {
	if !p.hasQualifiedIdentifier(lookahead) {
		return false
	}

	nextToken := p.peekN(lookahead.offset)
	if nextToken == nil {
		return isFunctionKey
	}

	if nextToken.Kind() == st.OPEN_PAREN_TOKEN {
		return p.hasFuncSignature(lookahead)
	} else if nextToken.Kind() == st.DOT_TOKEN {
		lookahead.offset++
		if !p.hasIdentifier(lookahead) {
			return false
		}
		return p.hasFuncSignature(lookahead)
	}

	return isFunctionKey
}

func (p *documentationParser) hasFuncSignature(lookahead *lookahead) bool {
	if !p.hasOpenParenthesis(lookahead) {
		return false
	}
	return p.hasCloseParenthesis(lookahead)
}

func (p *documentationParser) hasOpenParenthesis(lookahead *lookahead) bool {
	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == st.OPEN_PAREN_TOKEN {
		lookahead.offset++
		return true
	}
	return false
}

func (p *documentationParser) hasCloseParenthesis(lookahead *lookahead) bool {
	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == st.CLOSE_PAREN_TOKEN {
		lookahead.offset++
		return true
	}
	return false
}

func (p *documentationParser) hasQualifiedIdentifier(lookahead *lookahead) bool {
	if !p.hasIdentifier(lookahead) {
		return false
	}

	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == st.COLON_TOKEN {
		lookahead.offset++
		return p.hasIdentifier(lookahead)
	}

	return true
}

func (p *documentationParser) hasIdentifier(lookahead *lookahead) bool {
	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == st.IDENTIFIER_TOKEN {
		lookahead.offset++
		return true
	}
	return false
}

func (p *documentationParser) isDocumentReferenceType(kind st.SyntaxKind) bool {
	switch kind {
	case st.TYPE_DOC_REFERENCE_TOKEN,
		st.SERVICE_DOC_REFERENCE_TOKEN,
		st.VARIABLE_DOC_REFERENCE_TOKEN,
		st.VAR_DOC_REFERENCE_TOKEN,
		st.ANNOTATION_DOC_REFERENCE_TOKEN,
		st.MODULE_DOC_REFERENCE_TOKEN,
		st.FUNCTION_DOC_REFERENCE_TOKEN,
		st.PARAMETER_DOC_REFERENCE_TOKEN,
		st.CONST_DOC_REFERENCE_TOKEN:
		return true
	default:
		return false
	}
}

func (p *documentationParser) parseParameterDocumentationLine(hashToken st.STNode) st.STNode {
	plusToken := p.consume()
	parameterName := p.parseParameterName()
	dashToken := p.parseMinusToken()

	docElements := p.parseDocumentationElements()
	docElementList := st.CreateNodeList(docElements...)

	var kind st.SyntaxKind
	if parameterName.Kind() == st.RETURN_KEYWORD {
		kind = st.MARKDOWN_RETURN_PARAMETER_DOCUMENTATION_LINE
	} else {
		kind = st.MARKDOWN_PARAMETER_DOCUMENTATION_LINE
	}
	return p.createMarkdownParameterDocumentationLineNode(kind, hashToken, plusToken, parameterName, dashToken, docElementList)
}

func (p *documentationParser) isEndOfIntermediateDocumentation(kind st.SyntaxKind) bool {
	switch kind {
	case st.DOCUMENTATION_DESCRIPTION,
		st.PLUS_TOKEN,
		st.PARAMETER_NAME,
		st.MINUS_TOKEN,
		st.BACKTICK_TOKEN,
		st.DOUBLE_BACKTICK_TOKEN,
		st.TRIPLE_BACKTICK_TOKEN,
		st.CODE_CONTENT,
		st.RETURN_KEYWORD,
		st.DEPRECATION_LITERAL:
		return false
	default:
		return !p.isDocumentReferenceType(kind)
	}
}

func (p *documentationParser) parseParameterName() st.STNode {
	token := p.peek()
	if token == nil {
		return p.createMissingTokenWithDiagnostics(st.PARAMETER_NAME)
	}
	tokenKind := token.Kind()
	if tokenKind == st.PARAMETER_NAME || tokenKind == st.RETURN_KEYWORD {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.PARAMETER_NAME)
	}
}

func (p *documentationParser) parseMinusToken() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.MINUS_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.MINUS_TOKEN)
	}
}

func (p *documentationParser) parseBacktickToken() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.BACKTICK_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.BACKTICK_TOKEN)
	}
}

func (p *documentationParser) getReferenceGenre(referenceType st.STNode) referenceGenre {
	if referenceType == nil || referenceType.Kind() == st.NONE {
		return referenceGenreNoKey
	}
	if referenceType.Kind() == st.FUNCTION_DOC_REFERENCE_TOKEN {
		return referenceGenreFunctionKey
	}
	return referenceGenreSpecialKey
}

func (p *documentationParser) combineAndCreateCodeContentToken() st.STNode {
	if p.peek() == nil || !p.isBacktickExprToken(p.peek().Kind()) {
		return p.createMissingTokenWithDiagnostics(st.CODE_CONTENT)
	}

	var backtickContent strings.Builder
	var token st.STToken
	for p.peekN(2) != nil && p.isBacktickExprToken(p.peekN(2).Kind()) {
		token = p.consume()
		backtickContent.WriteString(token.Text())
	}
	token = p.consume()
	backtickContent.WriteString(token.Text())

	// We do not capture leading minutiae in DOCUMENTATION_BACKTICK_EXPR lexer mode.
	// Therefore, set only the trailing minutiae
	leadingMinutiae := st.CreateEmptyNodeList()
	trailingMinutiae := token.TrailingMinutiae()
	return st.CreateLiteralValueToken(st.CODE_CONTENT, backtickContent.String(),
		leadingMinutiae, trailingMinutiae)
}

func (p *documentationParser) isBacktickExprToken(kind st.SyntaxKind) bool {
	switch kind {
	case st.DOT_TOKEN,
		st.COLON_TOKEN,
		st.OPEN_PAREN_TOKEN,
		st.CLOSE_PAREN_TOKEN,
		st.IDENTIFIER_TOKEN,
		st.CODE_CONTENT:
		return true
	default:
		return false
	}
}

func (p *documentationParser) parseNameReferenceContent() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.IDENTIFIER_TOKEN {
		identifier := p.consume()
		return p.parseBacktickExpr(identifier)
	}
	identifier := p.createMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN)
	return p.parseBacktickExpr(identifier)
}

func (p *documentationParser) parseBacktickExpr(identifier st.STNode) st.STNode {
	referenceName := p.parseQualifiedIdentifier(identifier)

	nextToken := p.peek()
	if nextToken == nil {
		return referenceName
	}

	switch nextToken.Kind() {
	case st.DOT_TOKEN:
		dotToken := p.consume()
		return p.parseMethodCall(referenceName, dotToken)
	case st.OPEN_PAREN_TOKEN:
		return p.parseFuncCall(referenceName)
	default:
		return referenceName
	}
}

func (p *documentationParser) parseQualifiedIdentifier(identifier st.STNode) st.STNode {
	nextToken := p.peek()
	if nextToken != nil && nextToken.Kind() == st.COLON_TOKEN {
		colon := p.consume()
		return p.parseQualifiedIdentifierWithColon(identifier, colon)
	}
	return st.CreateSimpleNameReferenceNode(identifier)
}

func (p *documentationParser) parseQualifiedIdentifierWithColon(identifier st.STNode, colon st.STNode) st.STNode {
	refName := p.parseIdentifier()
	return st.CreateQualifiedNameReferenceNode(identifier, colon, refName)
}

func (p *documentationParser) parseIdentifier() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.IDENTIFIER_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.IDENTIFIER_TOKEN)
	}
}

func (p *documentationParser) parseFuncCall(referenceName st.STNode) st.STNode {
	openParen := p.parseOpenParenthesis()
	args := st.CreateEmptyNodeList()
	closeParen := p.parseCloseParenthesis()
	return st.CreateFunctionCallExpressionNode(referenceName, openParen, args, closeParen)
}

func (p *documentationParser) parseMethodCall(referenceName st.STNode, dotToken st.STNode) st.STNode {
	methodName := p.parseSimpleNameReference()
	openParen := p.parseOpenParenthesis()
	args := st.CreateEmptyNodeList()
	closeParen := p.parseCloseParenthesis()
	return st.CreateMethodCallExpressionNode(referenceName, dotToken, methodName, openParen, args, closeParen)
}

func (p *documentationParser) parseSimpleNameReference() st.STNode {
	identifier := p.parseIdentifier()
	return st.CreateSimpleNameReferenceNode(identifier)
}

func (p *documentationParser) parseOpenParenthesis() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.OPEN_PAREN_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.OPEN_PAREN_TOKEN)
	}
}

func (p *documentationParser) parseCloseParenthesis() st.STNode {
	token := p.peek()
	if token != nil && token.Kind() == st.CLOSE_PAREN_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(st.CLOSE_PAREN_TOKEN)
	}
}

func (p *documentationParser) createMissingTokenWithDiagnostics(expectedKind st.SyntaxKind) st.STToken {
	warningCode := p.getDocWarningCode(expectedKind)
	return st.CreateMissingTokenWithDiagnostics(expectedKind, warningCode)
}

func (p *documentationParser) getDocWarningCode(expectedKind st.SyntaxKind) diagnostics.DiagnosticCode {
	var code diagnostics.DiagnosticCode
	switch expectedKind {
	case st.HASH_TOKEN:
		code = &common.WARNING_MISSING_HASH_TOKEN
	case st.BACKTICK_TOKEN:
		code = &common.WARNING_MISSING_SINGLE_BACKTICK_TOKEN
	case st.DOUBLE_BACKTICK_TOKEN:
		code = &common.WARNING_MISSING_DOUBLE_BACKTICK_TOKEN
	case st.TRIPLE_BACKTICK_TOKEN:
		code = &common.WARNING_MISSING_TRIPLE_BACKTICK_TOKEN
	case st.IDENTIFIER_TOKEN:
		code = &common.WARNING_MISSING_IDENTIFIER_TOKEN
	case st.OPEN_PAREN_TOKEN:
		code = &common.WARNING_MISSING_OPEN_PAREN_TOKEN
	case st.CLOSE_PAREN_TOKEN:
		code = &common.WARNING_MISSING_CLOSE_PAREN_TOKEN
	case st.MINUS_TOKEN:
		code = &common.WARNING_MISSING_HYPHEN_TOKEN
	case st.PARAMETER_NAME:
		code = &common.WARNING_MISSING_PARAMETER_NAME
	case st.CODE_CONTENT:
		code = &common.WARNING_MISSING_CODE_REFERENCE
	default:
		code = &common.WARNING_SYNTAX_WARNING
	}
	return code
}

func (p *documentationParser) createMarkdownDocumentationLineNode(hashToken st.STNode, documentationElements st.STNode) st.STNode {
	return st.CreateMarkdownDocumentationLineNode(st.MARKDOWN_DOCUMENTATION_LINE, hashToken, documentationElements)
}

func (p *documentationParser) createMarkdownDeprecationDocumentationLineNode(hashToken st.STNode, documentationElements st.STNode) st.STNode {
	return st.CreateMarkdownDocumentationLineNode(st.MARKDOWN_DEPRECATION_DOCUMENTATION_LINE, hashToken, documentationElements)
}

func (p *documentationParser) createMarkdownReferenceDocumentationLineNode(hashToken st.STNode, documentationElements st.STNode) st.STNode {
	return st.CreateMarkdownDocumentationLineNode(st.MARKDOWN_REFERENCE_DOCUMENTATION_LINE, hashToken, documentationElements)
}

func (p *documentationParser) createMarkdownParameterDocumentationLineNode(kind st.SyntaxKind, hashToken st.STNode, plusToken st.STNode, parameterName st.STNode, dashToken st.STNode, docElementList st.STNode) st.STNode {
	return st.CreateMarkdownParameterDocumentationLineNode(kind, hashToken, plusToken, parameterName, dashToken, docElementList)
}

func (p *documentationParser) createInlineCodeReferenceNode(startBacktick st.STNode, codeReference st.STNode, endBacktick st.STNode) st.STNode {
	return st.CreateInlineCodeReferenceNode(startBacktick, codeReference, endBacktick)
}

func (p *documentationParser) createBallerinaNameReferenceNode(referenceType st.STNode, startBacktick st.STNode, nameReference st.STNode, endBacktick st.STNode) st.STNode {
	return st.CreateBallerinaNameReferenceNode(referenceType, startBacktick, nameReference, endBacktick)
}

func (p *documentationParser) createMarkdownCodeBlockNode(startLineHashToken st.STNode, startBacktick st.STNode, langAttribute st.STNode, codeLines st.STNode, endLineHashToken st.STNode, endBacktick st.STNode) st.STNode {
	return st.CreateMarkdownCodeBlockNode(startLineHashToken, startBacktick, langAttribute, codeLines, endLineHashToken, endBacktick)
}

func (p *documentationParser) createMarkdownCodeLineNode(hashToken st.STNode, codeDescription st.STNode) st.STNode {
	return st.CreateMarkdownCodeLineNode(hashToken, codeDescription)
}
