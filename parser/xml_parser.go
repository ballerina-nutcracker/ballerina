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

// Ported from XMLParser.java.

package parser

import (
	"ballerina/parser/common"
	"ballerina/parser/tree"
)

type xmlParser struct {
	abstractParser
	interpolationExprs []tree.STNode
	interpIdx          int
}

func newXMLParser(tokenReader *TokenReader, interpolationExprs []tree.STNode) *xmlParser {
	p := &xmlParser{
		abstractParser: abstractParser{
			tokenReader:          tokenReader,
			invalidNodeInfoStack: make([]invalidNodeInfo, 0),
			insertedToken:        nil,
		},
		interpolationExprs: interpolationExprs,
	}
	errorHandler := NewXMLParserErrorHandlerFromTokenReader(tokenReader)
	p.errorHandler = &errorHandler
	return p
}

func (p *xmlParser) Parse() tree.STNode {
	return p.parseXMLContent(false)
}

func (p *xmlParser) parseXMLContent(isInXMLElement bool) tree.STNode {
	items := make([]tree.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLContent(nextToken.Kind(), isInXMLElement) {
		contentItem := p.parseXMLContentItem()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	return tree.CreateNodeList(items...)
}

func (p *xmlParser) isEndOfXMLContent(kind common.SyntaxKind, isInXMLElement bool) bool {
	switch kind {
	case common.EOF_TOKEN, common.BACKTICK_TOKEN:
		return true
	case common.LT_TOKEN:
		nextNextKind := p.getNextNextToken().Kind()
		return isInXMLElement && (nextNextKind == common.SLASH_TOKEN || nextNextKind == common.LT_TOKEN)
	}
	return false
}

func (p *xmlParser) parseXMLContentItem() tree.STNode {
	switch p.peek().Kind() {
	case common.LT_TOKEN:
		return p.parseXMLElement()
	case common.XML_COMMENT_START_TOKEN:
		return p.parseXMLComment()
	case common.XML_PI_START_TOKEN:
		return p.parseXMLPI()
	case common.INTERPOLATION_START_TOKEN:
		return p.parseInterpolation()
	case common.XML_CDATA_START_TOKEN:
		return p.parseXMLCdataSection()
	default:
		return p.parseXMLText()
	}
}

func (p *xmlParser) parseInterpolation() tree.STNode {
	// Consume the synthetic INTERPOLATION_START_TOKEN ("${") and CLOSE_BRACE_TOKEN ("}")
	// emitted around the placeholder, and pull the pre-parsed expression off the queue.
	p.consume()
	p.consume()
	expr := p.interpolationExprs[p.interpIdx]
	p.interpIdx++
	return expr
}

func (p *xmlParser) parseXMLElement() tree.STNode {
	startTag := p.parseXMLElementStartOrEmptyTag()
	if startTag.Kind() == common.XML_EMPTY_ELEMENT {
		return startTag
	}

	content := p.parseXMLContent(true)
	endTag := p.parseXMLElementEndTag()
	return tree.CreateXMLElementNode(startTag, content, endTag)
}

func (p *xmlParser) parseXMLElementStartOrEmptyTag() tree.STNode {
	p.startContext(common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG)
	tagOpen := p.parseLTToken()
	name := p.parseXMLNCName()

	p.startContext(common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES)
	attributes := make([]tree.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLAttributes(nextToken.Kind()) {
		attribute := p.parseXMLAttribute()
		if attribute.Kind() == common.INTERPOLATION {
			if len(attributes) == 0 {
				name = tree.CloneWithTrailingInvalidNodeMinutiae(name, attribute,
					&common.ERROR_INTERPOLATION_IS_NOT_ALLOWED_WITHIN_ELEMENT_TAGS)
			} else {
				attributes = p.updateLastNodeInListWithInvalidNode(attributes, attribute,
					&common.ERROR_INTERPOLATION_IS_NOT_ALLOWED_WITHIN_ELEMENT_TAGS)
			}
		} else {
			attributes = append(attributes, attribute)
		}
		nextToken = p.peek()
	}
	p.endContext()

	xmlAttributes := tree.CreateNodeList(attributes...)
	return p.parseXMLElementTagEnd(tagOpen, name, xmlAttributes)
}

func (p *xmlParser) parseXMLElementTagEnd(tagOpen tree.STNode, name tree.STNode, attributes tree.STNode) tree.STNode {
	return p.parseXMLElementTagEndWithKind(p.peek().Kind(), tagOpen, name, attributes)
}

func (p *xmlParser) parseXMLElementTagEndWithKind(nextTokenKind common.SyntaxKind, tagOpen tree.STNode, name tree.STNode, attributes tree.STNode) tree.STNode {
	switch nextTokenKind {
	case common.SLASH_TOKEN:
		slash := p.parseSlashTokenForXML()
		tagClose := p.parseGTToken()
		p.endContext()
		return tree.CreateXMLEmptyElementNode(tagOpen, name, attributes, slash, tagClose)
	case common.GT_TOKEN:
		tagClose := p.parseGTToken()
		p.endContext()
		return tree.CreateXMLStartTagNode(tagOpen, name, attributes, tagClose)
	default:
		sol := p.recover(p.peek(), common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END, false)
		return p.parseXMLElementTagEndWithKind(sol.TokenKind, tagOpen, name, attributes)
	}
}

func (p *xmlParser) parseSlashTokenForXML() tree.STNode {
	token := p.peek()
	if token.Kind() == common.SLASH_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_SLASH, false)
	return p.parseSlashTokenForXML()
}

func (p *xmlParser) parseXMLElementEndTag() tree.STNode {
	p.startContext(common.PARSER_RULE_CONTEXT_XML_END_TAG)
	tagOpen := p.parseLTToken()
	slash := p.parseSlashTokenForXML()
	name := p.parseXMLNCName()
	tagClose := p.parseGTToken()
	p.endContext()
	return tree.CreateXMLEndTagNode(tagOpen, slash, name, tagClose)
}

func (p *xmlParser) parseLTToken() tree.STNode {
	token := p.peek()
	if token.Kind() == common.LT_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_LT_TOKEN, false)
	return p.parseLTToken()
}

func (p *xmlParser) parseGTToken() tree.STNode {
	token := p.peek()
	if token.Kind() == common.GT_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_GT_TOKEN, false)
	return p.parseGTToken()
}

func (p *xmlParser) parseXMLNCName() tree.STNode {
	token := p.peek()
	switch token.Kind() {
	case common.IDENTIFIER_TOKEN:
		return p.parseXMLQualifiedIdentifier(p.consume())
	case common.INTERPOLATION_START_TOKEN:
		interpolation := p.parseInterpolation()
		xmlNCName := p.parseXMLNCName()
		return tree.CloneWithLeadingInvalidNodeMinutiae(xmlNCName, interpolation,
			&common.ERROR_INTERPOLATION_IS_NOT_ALLOWED_FOR_XML_TAG_NAMES)
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_NAME, false)
	return p.parseXMLNCName()
}

func (p *xmlParser) parseXMLQualifiedIdentifier(identifier tree.STNode) tree.STNode {
	nextToken := p.peekN(1)
	if nextToken.Kind() != common.COLON_TOKEN {
		return tree.CreateXMLSimpleNameNode(identifier)
	}

	nextNextToken := p.peekN(2)
	if nextNextToken.Kind() == common.IDENTIFIER_TOKEN {
		colon := p.consume()
		varOrFuncName := tree.CreateXMLSimpleNameNode(p.consume())
		identifier = tree.CreateXMLSimpleNameNode(identifier)
		return tree.CreateXMLQualifiedNameNode(identifier, colon, varOrFuncName)
	}
	p.addInvalidTokenToNextToken(p.errorHandler.ConsumeInvalidToken())
	return p.parseXMLQualifiedIdentifier(identifier)
}

func (p *xmlParser) isEndOfXMLAttributes(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN,
		common.BACKTICK_TOKEN,
		common.GT_TOKEN,
		common.LT_TOKEN,
		common.SLASH_TOKEN,
		common.XML_COMMENT_START_TOKEN,
		common.XML_PI_START_TOKEN,
		common.XML_CDATA_START_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLAttribute() tree.STNode {
	if p.peek().Kind() == common.INTERPOLATION_START_TOKEN {
		return p.parseInterpolation()
	}
	attributeName := p.parseXMLNCName()
	equalToken := p.parseAssignOpForXML()
	value := p.parseAttributeValue()
	return tree.CreateXMLAttributeNode(attributeName, equalToken, value)
}

func (p *xmlParser) parseAssignOpForXML() tree.STNode {
	token := p.peek()
	if token.Kind() == common.EQUAL_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_ASSIGN_OP, false)
	return p.parseAssignOpForXML()
}

func (p *xmlParser) parseAttributeValue() tree.STNode {
	startQuote := p.parseXMLStartQuote(common.PARSER_RULE_CONTEXT_XML_QUOTE_START)
	items := make([]tree.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLAttributeValue(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	value := tree.CreateNodeList(items...)
	endQuote := p.parseXMLStartQuote(common.PARSER_RULE_CONTEXT_XML_QUOTE_END)
	return tree.CreateXMLAttributeValue(startQuote, value, endQuote)
}

func (p *xmlParser) parseXMLStartQuote(ctx common.ParserRuleContext) tree.STNode {
	token := p.peek()
	if token.Kind() == common.DOUBLE_QUOTE_TOKEN || token.Kind() == common.SINGLE_QUOTE_TOKEN {
		return p.consume()
	}
	p.recover(token, ctx, false)
	return p.parseXMLStartQuote(ctx)
}

func (p *xmlParser) isEndOfXMLAttributeValue(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN,
		common.BACKTICK_TOKEN,
		common.LT_TOKEN,
		common.GT_TOKEN,
		common.DOUBLE_QUOTE_TOKEN,
		common.SINGLE_QUOTE_TOKEN,
		common.IDENTIFIER_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLText() tree.STNode {
	switch p.peek().Kind() {
	case common.INTERPOLATION_START_TOKEN, common.EOF_TOKEN, common.BACKTICK_TOKEN, common.LT_TOKEN:
		return nil
	}
	content := p.parseCharData()
	return tree.CreateXMLTextNode(content)
}

func (p *xmlParser) parseCharData() tree.STNode {
	token := p.consume()
	if token.Kind() != common.XML_TEXT_CONTENT {
		return tree.CreateLiteralValueTokenWithDiagnostics(common.XML_TEXT_CONTENT, token.Text(),
			token.LeadingMinutiae(), token.TrailingMinutiae(), token.Diagnostics())
	}
	return token
}

func (p *xmlParser) parseXMLComment() tree.STNode {
	commentStart := p.parseXMLCommentStart()
	items := make([]tree.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLComment(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	content := tree.CreateNodeList(items...)
	commentEnd := p.parseXMLCommentEnd()
	return tree.CreateXMLComment(commentStart, content, commentEnd)
}

func (p *xmlParser) parseXMLCommentStart() tree.STNode {
	token := p.peek()
	if token.Kind() == common.XML_COMMENT_START_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_COMMENT_START, false)
	return p.parseXMLCommentStart()
}

func (p *xmlParser) parseXMLCommentEnd() tree.STNode {
	token := p.peek()
	if token.Kind() == common.XML_COMMENT_END_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_COMMENT_END, false)
	return p.parseXMLCommentEnd()
}

func (p *xmlParser) isEndOfXMLComment(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN, common.BACKTICK_TOKEN, common.LT_TOKEN, common.GT_TOKEN, common.XML_COMMENT_END_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLCdataSection() tree.STNode {
	cdataStart := p.consume()
	items := make([]tree.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLCdata(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	content := tree.CreateNodeList(items...)
	cdataEnd := p.parseXMLCdataEnd()
	return tree.CreateXMLCDATANode(cdataStart, content, cdataEnd)
}

func (p *xmlParser) isEndOfXMLCdata(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN, common.BACKTICK_TOKEN, common.XML_CDATA_END_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLCdataEnd() tree.STNode {
	token := p.peek()
	if token.Kind() == common.XML_CDATA_END_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_CDATA_END, false)
	return p.parseXMLCdataEnd()
}

func (p *xmlParser) parseXMLPI() tree.STNode {
	p.startContext(common.PARSER_RULE_CONTEXT_XML_PI)
	piStart := p.parseXMLPIStart()
	target := p.parseXMLNCName()

	items := make([]tree.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLPI(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	data := tree.CreateNodeList(items...)
	piEnd := p.parseXMLPIEnd()
	p.endContext()
	return tree.CreateXMLProcessingInstruction(piStart, target, data, piEnd)
}

func (p *xmlParser) parseXMLPIStart() tree.STNode {
	token := p.peek()
	if token.Kind() == common.XML_PI_START_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_PI_START, false)
	return p.parseXMLPIStart()
}

func (p *xmlParser) parseXMLPIEnd() tree.STNode {
	token := p.peek()
	if token.Kind() == common.XML_PI_END_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_PI_END, false)
	return p.parseXMLPIEnd()
}

func (p *xmlParser) isEndOfXMLPI(kind common.SyntaxKind) bool {
	switch kind {
	case common.EOF_TOKEN, common.BACKTICK_TOKEN, common.LT_TOKEN, common.GT_TOKEN, common.XML_PI_END_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLCharacterSet() tree.STNode {
	switch p.peek().Kind() {
	case common.XML_TEXT_CONTENT:
		return p.consume()
	case common.INTERPOLATION_START_TOKEN:
		return p.parseInterpolation()
	}
	panic("xmlParser.parseXMLCharacterSet: unexpected token")
}
