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

	"ballerina/parser/common"
	"ballerina/parser/tree"
	"ballerina/tools/diagnostics"
)

// DocumentationParser is a parser for Ballerina documentation (markdown).
// Ballerina flavored markdown (BFM) is supported by the documentation.
// There is no error handler attached to this parser.
// In case of an error, simply missing token will be inserted.
type DocumentationParser struct {
	abstractParser
}

func NewDocumentationParser(tokenReader *TokenReader) *DocumentationParser {
	parser := &DocumentationParser{}
	parser.abstractParser = NewAbstractParserFromTokenReader(tokenReader)
	return parser
}

func (p *DocumentationParser) Parse() tree.STNode {
	return p.parseDocumentationLines()
}

func (p *DocumentationParser) parseDocumentationLines() tree.STNode {
	docLines := make([]tree.STNode, 0)
	nextToken := p.peek()
	for nextToken != nil && nextToken.Kind() == common.HASH_TOKEN {
		docLines = append(docLines, p.parseSingleDocumentationLine())
		nextToken = p.peek()
	}
	return tree.CreateNodeList(docLines...)
}

func (p *DocumentationParser) parseSingleDocumentationLine() tree.STNode {
	hashToken := p.consume()
	nextToken := p.peek()
	if nextToken == nil {
		return p.createMarkdownDocumentationLineNode(hashToken, tree.CreateEmptyNodeList())
	}

	switch nextToken.Kind() {
	case common.PLUS_TOKEN:
		return p.parseParameterDocumentationLine(hashToken)
	case common.DEPRECATION_LITERAL:
		return p.parseDeprecationDocumentationLine(hashToken)
	case common.TRIPLE_BACKTICK_TOKEN, common.DOUBLE_BACKTICK_TOKEN:
		return p.parseCodeBlockOrInlineCodeRef(hashToken)
	default:
		return p.parseDocumentationLine(hashToken)
	}
}

func (p *DocumentationParser) parseCodeBlockOrInlineCodeRef(startLineHash tree.STNode) tree.STNode {
	startBacktick := p.consume()
	nextToken := p.peek()
	if nextToken == nil || !p.isInlineCodeRef(nextToken.Kind()) {
		return p.parseCodeBlock(startLineHash, startBacktick)
	}

	inlineCodeNode := p.parseInlineCode(startBacktick)
	docElements := []tree.STNode{inlineCodeNode}
	p.parseDocElements(&docElements)
	docElementList := tree.CreateNodeList(docElements...)
	return p.createMarkdownReferenceDocumentationLineNode(startLineHash, docElementList)
}

func (p *DocumentationParser) isInlineCodeRef(nextTokenKind common.SyntaxKind) bool {
	nextNext := p.getNextNextToken()
	switch nextTokenKind {
	case common.HASH_TOKEN:
		return nextNext != nil && nextNext.Kind() == common.DOCUMENTATION_DESCRIPTION
	case common.CODE_CONTENT:
		if nextNext == nil {
			return true
		}
		return nextNext.Kind() != common.HASH_TOKEN
	default:
		return true
	}
}

func (p *DocumentationParser) parseDeprecationDocumentationLine(hashToken tree.STNode) tree.STNode {
	deprecationLiteral := p.consume()
	docElements := p.parseDocumentationElements()
	docElements = append([]tree.STNode{deprecationLiteral}, docElements...)
	docElementList := tree.CreateNodeList(docElements...)
	return p.createMarkdownDeprecationDocumentationLineNode(hashToken, docElementList)
}

func (p *DocumentationParser) parseDocumentationLine(hashToken tree.STNode) tree.STNode {
	docElements := p.parseDocumentationElements()
	docElementList := tree.CreateNodeList(docElements...)

	switch len(docElements) {
	case 0:
		// When documentation line is only a `#` token
		return p.createMarkdownDocumentationLineNode(hashToken, docElementList)
	case 1:
		docElement := docElements[0]
		if docElement.Kind() == common.DOCUMENTATION_DESCRIPTION {
			return p.createMarkdownDocumentationLineNode(hashToken, docElementList)
		}
		fallthrough
	default:
		return p.createMarkdownReferenceDocumentationLineNode(hashToken, docElementList)
	}
}

func (p *DocumentationParser) parseDocumentationElements() []tree.STNode {
	docElements := make([]tree.STNode, 0)
	p.parseDocElements(&docElements)
	return docElements
}

func (p *DocumentationParser) parseDocElements(docElements *[]tree.STNode) {
	var docElement tree.STNode
	var referenceType tree.STNode

	nextToken := p.peek()
	for nextToken != nil && !p.isEndOfIntermediateDocumentation(nextToken.Kind()) {
		switch nextToken.Kind() {
		case common.DOCUMENTATION_DESCRIPTION:
			docElement = p.consume()
		case common.CODE_CONTENT:
			token := p.consume()
			docElement = p.convertToDocDescriptionToken(token)
		case common.DOUBLE_BACKTICK_TOKEN, common.TRIPLE_BACKTICK_TOKEN:
			docElement = p.parseInlineCode(p.consume())
		case common.BACKTICK_TOKEN:
			referenceType = tree.CreateEmptyNode()
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

func (p *DocumentationParser) convertToDocDescriptionToken(token tree.STToken) tree.STNode {
	return tree.CreateLiteralValueToken(common.DOCUMENTATION_DESCRIPTION, token.Text(),
		token.LeadingMinutiae(), token.TrailingMinutiae())
}

func (p *DocumentationParser) convertToCodeContentToken(token tree.STToken) tree.STNode {
	return tree.CreateLiteralValueToken(common.CODE_CONTENT, token.Text(),
		token.LeadingMinutiae(), token.TrailingMinutiae())
}

func (p *DocumentationParser) parseInlineCode(startBacktick tree.STNode) tree.STNode {
	codeDescription := p.parseInlineCodeContentToken()
	endBacktick := p.parseCodeEndBacktick(startBacktick.Kind())
	return p.createInlineCodeReferenceNode(startBacktick, codeDescription, endBacktick)
}

func (p *DocumentationParser) parseInlineCodeContentToken() tree.STNode {
	token := p.peek()
	if token == nil {
		return p.createMissingTokenWithDiagnostics(common.CODE_CONTENT)
	}

	if token.Kind() == common.CODE_CONTENT {
		return p.consume()
	} else if token.Kind() == common.DOCUMENTATION_DESCRIPTION {
		token = p.consume()
		return p.convertToCodeContentToken(token)
	} else {
		return p.createMissingTokenWithDiagnostics(common.CODE_CONTENT)
	}
}

func (p *DocumentationParser) parseCodeBlock(startLineHash tree.STNode, startBacktick tree.STNode) tree.STNode {
	langAttribute := p.parseOptionalLangAttributeToken()
	codeLines := p.parseCodeLines()
	endLineHash := p.parseHashToken()
	endBacktick := p.parseCodeEndBacktick(startBacktick.Kind())

	// Handle any invalid tokens after the code block
	for p.peek() != nil && !p.isEndOfIntermediateDocumentation(p.peek().Kind()) {
		invalidToken := p.consume()
		endBacktick = tree.CloneWithTrailingInvalidNodeMinutiae(endBacktick, invalidToken,
			&common.WARNING_CANNOT_HAVE_DOCUMENTATION_INLINE_WITH_A_CODE_REFERENCE_BLOCK)
	}
	return p.createMarkdownCodeBlockNode(startLineHash, startBacktick, langAttribute, codeLines, endLineHash, endBacktick)
}

func (p *DocumentationParser) parseOptionalLangAttributeToken() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.CODE_CONTENT {
		return p.consume()
	} else {
		return tree.CreateEmptyNode()
	}
}

func (p *DocumentationParser) parseCodeLines() tree.STNode {
	codeLineList := make([]tree.STNode, 0)
	for !p.isEndOfCodeLines() {
		codeLineNode := p.parseCodeLine()
		codeLineList = append(codeLineList, codeLineNode)
	}
	return tree.CreateNodeList(codeLineList...)
}

func (p *DocumentationParser) parseCodeLine() tree.STNode {
	hash := p.parseHashToken()
	var codeDescription tree.STNode
	nextToken := p.peek()
	if nextToken != nil && nextToken.Kind() == common.HASH_TOKEN {
		// We reach here, when the code line is empty
		codeDescription = p.createEmptyCodeContentToken()
	} else {
		codeDescription = p.parseInlineCodeContentToken()
	}
	return p.createMarkdownCodeLineNode(hash, codeDescription)
}

func (p *DocumentationParser) createEmptyCodeContentToken() tree.STNode {
	emptyMinutiae := tree.CreateEmptyNodeList()
	return tree.CreateLiteralValueToken(common.CODE_CONTENT, "", emptyMinutiae, emptyMinutiae)
}

func (p *DocumentationParser) parseHashToken() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.HASH_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.HASH_TOKEN)
	}
}

func (p *DocumentationParser) parseCodeEndBacktick(backtickKind common.SyntaxKind) tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == backtickKind {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(backtickKind)
	}
}

func (p *DocumentationParser) isEndOfCodeLines() bool {
	nextToken := p.peek()
	if nextToken == nil {
		return true
	}
	if nextToken.Kind() == common.HASH_TOKEN {
		nextNextToken := p.getNextNextToken()
		if nextNextToken == nil {
			return true
		}
		switch nextNextToken.Kind() {
		case common.CODE_CONTENT, common.HASH_TOKEN:
			return false
		default:
			return true
		}
	}
	return true
}

func (p *DocumentationParser) parseBallerinaNameRefOrInlineCodeRef(referenceType tree.STNode) tree.STNode {
	startBacktick := p.parseBacktickToken()
	isCodeRef := false
	var contentToken tree.STNode
	referenceGenre := p.getReferenceGenre(referenceType)
	if p.isBallerinaNameRefTokenSequence(referenceGenre) {
		contentToken = p.parseNameReferenceContent()
	} else {
		contentToken = p.combineAndCreateCodeContentToken()
		if referenceGenre != ReferenceGenre_NO_KEY {
			contentToken = tree.AddDiagnostic(contentToken, &common.WARNING_INVALID_BALLERINA_NAME_REFERENCE, contentToken.(tree.STToken).Text())
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

type ReferenceGenre int

const (
	ReferenceGenre_NO_KEY ReferenceGenre = iota
	ReferenceGenre_SPECIAL_KEY
	ReferenceGenre_FUNCTION_KEY
)

type Lookahead struct {
	offset int
}

func (p *DocumentationParser) isBallerinaNameRefTokenSequence(refGenre ReferenceGenre) bool {
	hasMatch := false
	lookahead := &Lookahead{offset: 1}

	switch refGenre {
	case ReferenceGenre_SPECIAL_KEY:
		// Look for x, m:x match
		hasMatch = p.hasQualifiedIdentifier(lookahead)
	case ReferenceGenre_FUNCTION_KEY:
		// Look for x, m:x, x(), m:x(), T.y(), m:T.y() match
		hasMatch = p.hasBacktickExpr(lookahead, true)
	case ReferenceGenre_NO_KEY:
		// Look for x(), m:x(), T.y(), m:T.y() match
		hasMatch = p.hasBacktickExpr(lookahead, false)
	}

	if !hasMatch {
		return false
	}

	peekToken := p.peekN(lookahead.offset)
	return peekToken != nil && peekToken.Kind() == common.BACKTICK_TOKEN
}

func (p *DocumentationParser) hasBacktickExpr(lookahead *Lookahead, isFunctionKey bool) bool {
	if !p.hasQualifiedIdentifier(lookahead) {
		return false
	}

	nextToken := p.peekN(lookahead.offset)
	if nextToken == nil {
		return isFunctionKey
	}

	if nextToken.Kind() == common.OPEN_PAREN_TOKEN {
		return p.hasFuncSignature(lookahead)
	} else if nextToken.Kind() == common.DOT_TOKEN {
		lookahead.offset++
		if !p.hasIdentifier(lookahead) {
			return false
		}
		return p.hasFuncSignature(lookahead)
	}

	return isFunctionKey
}

func (p *DocumentationParser) hasFuncSignature(lookahead *Lookahead) bool {
	if !p.hasOpenParenthesis(lookahead) {
		return false
	}
	return p.hasCloseParenthesis(lookahead)
}

func (p *DocumentationParser) hasOpenParenthesis(lookahead *Lookahead) bool {
	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == common.OPEN_PAREN_TOKEN {
		lookahead.offset++
		return true
	}
	return false
}

func (p *DocumentationParser) hasCloseParenthesis(lookahead *Lookahead) bool {
	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == common.CLOSE_PAREN_TOKEN {
		lookahead.offset++
		return true
	}
	return false
}

func (p *DocumentationParser) hasQualifiedIdentifier(lookahead *Lookahead) bool {
	if !p.hasIdentifier(lookahead) {
		return false
	}

	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == common.COLON_TOKEN {
		lookahead.offset++
		return p.hasIdentifier(lookahead)
	}

	return true
}

func (p *DocumentationParser) hasIdentifier(lookahead *Lookahead) bool {
	nextToken := p.peekN(lookahead.offset)
	if nextToken != nil && nextToken.Kind() == common.IDENTIFIER_TOKEN {
		lookahead.offset++
		return true
	}
	return false
}

func (p *DocumentationParser) isDocumentReferenceType(kind common.SyntaxKind) bool {
	switch kind {
	case common.TYPE_DOC_REFERENCE_TOKEN,
		common.SERVICE_DOC_REFERENCE_TOKEN,
		common.VARIABLE_DOC_REFERENCE_TOKEN,
		common.VAR_DOC_REFERENCE_TOKEN,
		common.ANNOTATION_DOC_REFERENCE_TOKEN,
		common.MODULE_DOC_REFERENCE_TOKEN,
		common.FUNCTION_DOC_REFERENCE_TOKEN,
		common.PARAMETER_DOC_REFERENCE_TOKEN,
		common.CONST_DOC_REFERENCE_TOKEN:
		return true
	default:
		return false
	}
}

func (p *DocumentationParser) parseParameterDocumentationLine(hashToken tree.STNode) tree.STNode {
	plusToken := p.consume()
	parameterName := p.parseParameterName()
	dashToken := p.parseMinusToken()

	docElements := p.parseDocumentationElements()
	docElementList := tree.CreateNodeList(docElements...)

	var kind common.SyntaxKind
	if parameterName.Kind() == common.RETURN_KEYWORD {
		kind = common.MARKDOWN_RETURN_PARAMETER_DOCUMENTATION_LINE
	} else {
		kind = common.MARKDOWN_PARAMETER_DOCUMENTATION_LINE
	}
	return p.createMarkdownParameterDocumentationLineNode(kind, hashToken, plusToken, parameterName, dashToken, docElementList)
}

func (p *DocumentationParser) isEndOfIntermediateDocumentation(kind common.SyntaxKind) bool {
	switch kind {
	case common.DOCUMENTATION_DESCRIPTION,
		common.PLUS_TOKEN,
		common.PARAMETER_NAME,
		common.MINUS_TOKEN,
		common.BACKTICK_TOKEN,
		common.DOUBLE_BACKTICK_TOKEN,
		common.TRIPLE_BACKTICK_TOKEN,
		common.CODE_CONTENT,
		common.RETURN_KEYWORD,
		common.DEPRECATION_LITERAL:
		return false
	default:
		return !p.isDocumentReferenceType(kind)
	}
}

func (p *DocumentationParser) parseParameterName() tree.STNode {
	token := p.peek()
	if token == nil {
		return p.createMissingTokenWithDiagnostics(common.PARAMETER_NAME)
	}
	tokenKind := token.Kind()
	if tokenKind == common.PARAMETER_NAME || tokenKind == common.RETURN_KEYWORD {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.PARAMETER_NAME)
	}
}

func (p *DocumentationParser) parseMinusToken() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.MINUS_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.MINUS_TOKEN)
	}
}

func (p *DocumentationParser) parseBacktickToken() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.BACKTICK_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.BACKTICK_TOKEN)
	}
}

func (p *DocumentationParser) getReferenceGenre(referenceType tree.STNode) ReferenceGenre {
	if referenceType == nil || referenceType.Kind() == common.NONE {
		return ReferenceGenre_NO_KEY
	}
	if referenceType.Kind() == common.FUNCTION_DOC_REFERENCE_TOKEN {
		return ReferenceGenre_FUNCTION_KEY
	}
	return ReferenceGenre_SPECIAL_KEY
}

func (p *DocumentationParser) combineAndCreateCodeContentToken() tree.STNode {
	if p.peek() == nil || !p.isBacktickExprToken(p.peek().Kind()) {
		return p.createMissingTokenWithDiagnostics(common.CODE_CONTENT)
	}

	var backtickContent strings.Builder
	var token tree.STToken
	for p.peekN(2) != nil && p.isBacktickExprToken(p.peekN(2).Kind()) {
		token = p.consume()
		backtickContent.WriteString(token.Text())
	}
	token = p.consume()
	backtickContent.WriteString(token.Text())

	// We do not capture leading minutiae in DOCUMENTATION_BACKTICK_EXPR lexer mode.
	// Therefore, set only the trailing minutiae
	leadingMinutiae := tree.CreateEmptyNodeList()
	trailingMinutiae := token.TrailingMinutiae()
	return tree.CreateLiteralValueToken(common.CODE_CONTENT, backtickContent.String(),
		leadingMinutiae, trailingMinutiae)
}

func (p *DocumentationParser) isBacktickExprToken(kind common.SyntaxKind) bool {
	switch kind {
	case common.DOT_TOKEN,
		common.COLON_TOKEN,
		common.OPEN_PAREN_TOKEN,
		common.CLOSE_PAREN_TOKEN,
		common.IDENTIFIER_TOKEN,
		common.CODE_CONTENT:
		return true
	default:
		return false
	}
}

func (p *DocumentationParser) parseNameReferenceContent() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.IDENTIFIER_TOKEN {
		identifier := p.consume()
		return p.parseBacktickExpr(identifier)
	}
	identifier := p.createMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN)
	return p.parseBacktickExpr(identifier)
}

func (p *DocumentationParser) parseBacktickExpr(identifier tree.STNode) tree.STNode {
	referenceName := p.parseQualifiedIdentifier(identifier)

	nextToken := p.peek()
	if nextToken == nil {
		return referenceName
	}

	switch nextToken.Kind() {
	case common.DOT_TOKEN:
		dotToken := p.consume()
		return p.parseMethodCall(referenceName, dotToken)
	case common.OPEN_PAREN_TOKEN:
		return p.parseFuncCall(referenceName)
	default:
		return referenceName
	}
}

func (p *DocumentationParser) parseQualifiedIdentifier(identifier tree.STNode) tree.STNode {
	nextToken := p.peek()
	if nextToken != nil && nextToken.Kind() == common.COLON_TOKEN {
		colon := p.consume()
		return p.parseQualifiedIdentifierWithColon(identifier, colon)
	}
	return tree.CreateSimpleNameReferenceNode(identifier)
}

func (p *DocumentationParser) parseQualifiedIdentifierWithColon(identifier tree.STNode, colon tree.STNode) tree.STNode {
	refName := p.parseIdentifier()
	return tree.CreateQualifiedNameReferenceNode(identifier, colon, refName)
}

func (p *DocumentationParser) parseIdentifier() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.IDENTIFIER_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.IDENTIFIER_TOKEN)
	}
}

func (p *DocumentationParser) parseFuncCall(referenceName tree.STNode) tree.STNode {
	openParen := p.parseOpenParenthesis()
	args := tree.CreateEmptyNodeList()
	closeParen := p.parseCloseParenthesis()
	return tree.CreateFunctionCallExpressionNode(referenceName, openParen, args, closeParen)
}

func (p *DocumentationParser) parseMethodCall(referenceName tree.STNode, dotToken tree.STNode) tree.STNode {
	methodName := p.parseSimpleNameReference()
	openParen := p.parseOpenParenthesis()
	args := tree.CreateEmptyNodeList()
	closeParen := p.parseCloseParenthesis()
	return tree.CreateMethodCallExpressionNode(referenceName, dotToken, methodName, openParen, args, closeParen)
}

func (p *DocumentationParser) parseSimpleNameReference() tree.STNode {
	identifier := p.parseIdentifier()
	return tree.CreateSimpleNameReferenceNode(identifier)
}

func (p *DocumentationParser) parseOpenParenthesis() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.OPEN_PAREN_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.OPEN_PAREN_TOKEN)
	}
}

func (p *DocumentationParser) parseCloseParenthesis() tree.STNode {
	token := p.peek()
	if token != nil && token.Kind() == common.CLOSE_PAREN_TOKEN {
		return p.consume()
	} else {
		return p.createMissingTokenWithDiagnostics(common.CLOSE_PAREN_TOKEN)
	}
}

func (p *DocumentationParser) createMissingTokenWithDiagnostics(expectedKind common.SyntaxKind) tree.STToken {
	warningCode := p.getDocWarningCode(expectedKind)
	return tree.CreateMissingTokenWithDiagnostics(expectedKind, warningCode)
}

func (p *DocumentationParser) getDocWarningCode(expectedKind common.SyntaxKind) diagnostics.DiagnosticCode {
	var code diagnostics.DiagnosticCode
	switch expectedKind {
	case common.HASH_TOKEN:
		code = &common.WARNING_MISSING_HASH_TOKEN
	case common.BACKTICK_TOKEN:
		code = &common.WARNING_MISSING_SINGLE_BACKTICK_TOKEN
	case common.DOUBLE_BACKTICK_TOKEN:
		code = &common.WARNING_MISSING_DOUBLE_BACKTICK_TOKEN
	case common.TRIPLE_BACKTICK_TOKEN:
		code = &common.WARNING_MISSING_TRIPLE_BACKTICK_TOKEN
	case common.IDENTIFIER_TOKEN:
		code = &common.WARNING_MISSING_IDENTIFIER_TOKEN
	case common.OPEN_PAREN_TOKEN:
		code = &common.WARNING_MISSING_OPEN_PAREN_TOKEN
	case common.CLOSE_PAREN_TOKEN:
		code = &common.WARNING_MISSING_CLOSE_PAREN_TOKEN
	case common.MINUS_TOKEN:
		code = &common.WARNING_MISSING_HYPHEN_TOKEN
	case common.PARAMETER_NAME:
		code = &common.WARNING_MISSING_PARAMETER_NAME
	case common.CODE_CONTENT:
		code = &common.WARNING_MISSING_CODE_REFERENCE
	default:
		code = &common.WARNING_SYNTAX_WARNING
	}
	return code
}

func (p *DocumentationParser) createMarkdownDocumentationLineNode(hashToken tree.STNode, documentationElements tree.STNode) tree.STNode {
	return tree.CreateMarkdownDocumentationLineNode(common.MARKDOWN_DOCUMENTATION_LINE, hashToken, documentationElements)
}

func (p *DocumentationParser) createMarkdownDeprecationDocumentationLineNode(hashToken tree.STNode, documentationElements tree.STNode) tree.STNode {
	return tree.CreateMarkdownDocumentationLineNode(common.MARKDOWN_DEPRECATION_DOCUMENTATION_LINE, hashToken, documentationElements)
}

func (p *DocumentationParser) createMarkdownReferenceDocumentationLineNode(hashToken tree.STNode, documentationElements tree.STNode) tree.STNode {
	return tree.CreateMarkdownDocumentationLineNode(common.MARKDOWN_REFERENCE_DOCUMENTATION_LINE, hashToken, documentationElements)
}

func (p *DocumentationParser) createMarkdownParameterDocumentationLineNode(kind common.SyntaxKind, hashToken tree.STNode, plusToken tree.STNode, parameterName tree.STNode, dashToken tree.STNode, docElementList tree.STNode) tree.STNode {
	return tree.CreateMarkdownParameterDocumentationLineNode(kind, hashToken, plusToken, parameterName, dashToken, docElementList)
}

func (p *DocumentationParser) createInlineCodeReferenceNode(startBacktick tree.STNode, codeReference tree.STNode, endBacktick tree.STNode) tree.STNode {
	return tree.CreateInlineCodeReferenceNode(startBacktick, codeReference, endBacktick)
}

func (p *DocumentationParser) createBallerinaNameReferenceNode(referenceType tree.STNode, startBacktick tree.STNode, nameReference tree.STNode, endBacktick tree.STNode) tree.STNode {
	return tree.CreateBallerinaNameReferenceNode(referenceType, startBacktick, nameReference, endBacktick)
}

func (p *DocumentationParser) createMarkdownCodeBlockNode(startLineHashToken tree.STNode, startBacktick tree.STNode, langAttribute tree.STNode, codeLines tree.STNode, endLineHashToken tree.STNode, endBacktick tree.STNode) tree.STNode {
	return tree.CreateMarkdownCodeBlockNode(startLineHashToken, startBacktick, langAttribute, codeLines, endLineHashToken, endBacktick)
}

func (p *DocumentationParser) createMarkdownCodeLineNode(hashToken tree.STNode, codeDescription tree.STNode) tree.STNode {
	return tree.CreateMarkdownCodeLineNode(hashToken, codeDescription)
}
