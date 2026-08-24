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
	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
)

type xmlParser struct {
	abstractParser
	interpolationExprs []st.STNode
	interpIdx          int
}

func newXMLParser(tokenReader *tokenReader, interpolationExprs []st.STNode) *xmlParser {
	p := &xmlParser{
		abstractParser: abstractParser{
			tokenReader:          tokenReader,
			invalidNodeInfoStack: make([]invalidNodeInfo, 0),
			insertedToken:        nil,
		},
		interpolationExprs: interpolationExprs,
	}
	errorHandler := newXMLParserErrorHandlerFromTokenReader(tokenReader)
	p.errorHandler = &errorHandler
	return p
}

func (p *xmlParser) Parse() st.STNode {
	return p.parseXMLContent(false)
}

func (p *xmlParser) parseXMLContent(isInXMLElement bool) st.STNode {
	items := make([]st.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLContent(nextToken.Kind(), isInXMLElement) {
		contentItem := p.parseXMLContentItem()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	return st.CreateNodeList(items...)
}

func (p *xmlParser) isEndOfXMLContent(kind st.SyntaxKind, isInXMLElement bool) bool {
	switch kind {
	case st.EOF_TOKEN, st.BACKTICK_TOKEN:
		return true
	case st.LT_TOKEN:
		nextNextKind := p.getNextNextToken().Kind()
		return isInXMLElement && (nextNextKind == st.SLASH_TOKEN || nextNextKind == st.LT_TOKEN)
	}
	return false
}

func (p *xmlParser) parseXMLContentItem() st.STNode {
	switch p.peek().Kind() {
	case st.LT_TOKEN:
		return p.parseXMLElement()
	case st.XML_COMMENT_START_TOKEN:
		return p.parseXMLComment()
	case st.XML_PI_START_TOKEN:
		return p.parseXMLPI()
	case st.INTERPOLATION_START_TOKEN:
		return p.parseInterpolation()
	case st.XML_CDATA_START_TOKEN:
		return p.parseXMLCdataSection()
	default:
		return p.parseXMLText()
	}
}

func (p *xmlParser) parseInterpolation() st.STNode {
	// Consume the synthetic INTERPOLATION_START_TOKEN ("${") and CLOSE_BRACE_TOKEN ("}")
	// emitted around the placeholder, and pull the pre-parsed expression off the queue.
	p.consume()
	p.consume()
	expr := p.interpolationExprs[p.interpIdx]
	p.interpIdx++
	return expr
}

func (p *xmlParser) parseXMLElement() st.STNode {
	startTag := p.parseXMLElementStartOrEmptyTag()
	if startTag.Kind() == st.XML_EMPTY_ELEMENT {
		return startTag
	}

	content := p.parseXMLContent(true)
	endTag := p.parseXMLElementEndTag()
	return st.CreateXMLElementNode(startTag, content, endTag)
}

func (p *xmlParser) parseXMLElementStartOrEmptyTag() st.STNode {
	p.startContext(common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG)
	tagOpen := p.parseLTToken()
	name := p.parseXMLNCName()

	p.startContext(common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES)
	attributes := make([]st.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLAttributes(nextToken.Kind()) {
		attribute := p.parseXMLAttribute()
		if attribute.Kind() == st.INTERPOLATION {
			if len(attributes) == 0 {
				name = st.CloneWithTrailingInvalidNodeMinutiae(name, attribute,
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

	xmlAttributes := st.CreateNodeList(attributes...)
	return p.parseXMLElementTagEnd(tagOpen, name, xmlAttributes)
}

func (p *xmlParser) parseXMLElementTagEnd(tagOpen st.STNode, name st.STNode, attributes st.STNode) st.STNode {
	return p.parseXMLElementTagEndWithKind(p.peek().Kind(), tagOpen, name, attributes)
}

func (p *xmlParser) parseXMLElementTagEndWithKind(nextTokenKind st.SyntaxKind, tagOpen st.STNode, name st.STNode, attributes st.STNode) st.STNode {
	switch nextTokenKind {
	case st.SLASH_TOKEN:
		slash := p.parseSlashTokenForXML()
		tagClose := p.parseGTToken()
		p.endContext()
		return st.CreateXMLEmptyElementNode(tagOpen, name, attributes, slash, tagClose)
	case st.GT_TOKEN:
		tagClose := p.parseGTToken()
		p.endContext()
		return st.CreateXMLStartTagNode(tagOpen, name, attributes, tagClose)
	default:
		sol := p.recover(p.peek(), common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END, false)
		return p.parseXMLElementTagEndWithKind(sol.TokenKind, tagOpen, name, attributes)
	}
}

func (p *xmlParser) parseSlashTokenForXML() st.STNode {
	token := p.peek()
	if token.Kind() == st.SLASH_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_SLASH, false)
	return p.parseSlashTokenForXML()
}

func (p *xmlParser) parseXMLElementEndTag() st.STNode {
	p.startContext(common.PARSER_RULE_CONTEXT_XML_END_TAG)
	tagOpen := p.parseLTToken()
	slash := p.parseSlashTokenForXML()
	name := p.parseXMLNCName()
	tagClose := p.parseGTToken()
	p.endContext()
	return st.CreateXMLEndTagNode(tagOpen, slash, name, tagClose)
}

func (p *xmlParser) parseLTToken() st.STNode {
	token := p.peek()
	if token.Kind() == st.LT_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_LT_TOKEN, false)
	return p.parseLTToken()
}

func (p *xmlParser) parseGTToken() st.STNode {
	token := p.peek()
	if token.Kind() == st.GT_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_GT_TOKEN, false)
	return p.parseGTToken()
}

func (p *xmlParser) parseXMLNCName() st.STNode {
	token := p.peek()
	switch token.Kind() {
	case st.IDENTIFIER_TOKEN:
		return p.parseXMLQualifiedIdentifier(p.consume())
	case st.INTERPOLATION_START_TOKEN:
		interpolation := p.parseInterpolation()
		xmlNCName := p.parseXMLNCName()
		return st.CloneWithLeadingInvalidNodeMinutiae(xmlNCName, interpolation,
			&common.ERROR_INTERPOLATION_IS_NOT_ALLOWED_FOR_XML_TAG_NAMES)
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_NAME, false)
	return p.parseXMLNCName()
}

func (p *xmlParser) parseXMLQualifiedIdentifier(identifier st.STNode) st.STNode {
	nextToken := p.peekN(1)
	if nextToken.Kind() != st.COLON_TOKEN {
		return st.CreateXMLSimpleNameNode(identifier)
	}

	nextNextToken := p.peekN(2)
	if nextNextToken.Kind() == st.IDENTIFIER_TOKEN {
		colon := p.consume()
		varOrFuncName := st.CreateXMLSimpleNameNode(p.consume())
		identifier = st.CreateXMLSimpleNameNode(identifier)
		return st.CreateXMLQualifiedNameNode(identifier, colon, varOrFuncName)
	}
	p.addInvalidTokenToNextToken(p.errorHandler.ConsumeInvalidToken())
	return p.parseXMLQualifiedIdentifier(identifier)
}

func (p *xmlParser) isEndOfXMLAttributes(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN,
		st.BACKTICK_TOKEN,
		st.GT_TOKEN,
		st.LT_TOKEN,
		st.SLASH_TOKEN,
		st.XML_COMMENT_START_TOKEN,
		st.XML_PI_START_TOKEN,
		st.XML_CDATA_START_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLAttribute() st.STNode {
	if p.peek().Kind() == st.INTERPOLATION_START_TOKEN {
		return p.parseInterpolation()
	}
	attributeName := p.parseXMLNCName()
	equalToken := p.parseAssignOpForXML()
	value := p.parseAttributeValue()
	return st.CreateXMLAttributeNode(attributeName, equalToken, value)
}

func (p *xmlParser) parseAssignOpForXML() st.STNode {
	token := p.peek()
	if token.Kind() == st.EQUAL_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_ASSIGN_OP, false)
	return p.parseAssignOpForXML()
}

func (p *xmlParser) parseAttributeValue() st.STNode {
	startQuote := p.parseXMLStartQuote(common.PARSER_RULE_CONTEXT_XML_QUOTE_START)
	items := make([]st.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLAttributeValue(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	value := st.CreateNodeList(items...)
	endQuote := p.parseXMLStartQuote(common.PARSER_RULE_CONTEXT_XML_QUOTE_END)
	return st.CreateXMLAttributeValue(startQuote, value, endQuote)
}

func (p *xmlParser) parseXMLStartQuote(ctx common.ParserRuleContext) st.STNode {
	token := p.peek()
	if token.Kind() == st.DOUBLE_QUOTE_TOKEN || token.Kind() == st.SINGLE_QUOTE_TOKEN {
		return p.consume()
	}
	p.recover(token, ctx, false)
	return p.parseXMLStartQuote(ctx)
}

func (p *xmlParser) isEndOfXMLAttributeValue(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN,
		st.BACKTICK_TOKEN,
		st.LT_TOKEN,
		st.GT_TOKEN,
		st.DOUBLE_QUOTE_TOKEN,
		st.SINGLE_QUOTE_TOKEN,
		st.IDENTIFIER_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLText() st.STNode {
	switch p.peek().Kind() {
	case st.INTERPOLATION_START_TOKEN, st.EOF_TOKEN, st.BACKTICK_TOKEN, st.LT_TOKEN:
		return nil
	}
	content := p.parseCharData()
	return st.CreateXMLTextNode(content)
}

func (p *xmlParser) parseCharData() st.STNode {
	token := p.consume()
	if token.Kind() != st.XML_TEXT_CONTENT {
		return st.CreateLiteralValueTokenWithDiagnostics(st.XML_TEXT_CONTENT, token.Text(),
			token.LeadingMinutiae(), token.TrailingMinutiae(), token.Diagnostics())
	}
	return token
}

func (p *xmlParser) parseXMLComment() st.STNode {
	commentStart := p.parseXMLCommentStart()
	items := make([]st.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLComment(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	content := st.CreateNodeList(items...)
	commentEnd := p.parseXMLCommentEnd()
	return st.CreateXMLComment(commentStart, content, commentEnd)
}

func (p *xmlParser) parseXMLCommentStart() st.STNode {
	token := p.peek()
	if token.Kind() == st.XML_COMMENT_START_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_COMMENT_START, false)
	return p.parseXMLCommentStart()
}

func (p *xmlParser) parseXMLCommentEnd() st.STNode {
	token := p.peek()
	if token.Kind() == st.XML_COMMENT_END_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_COMMENT_END, false)
	return p.parseXMLCommentEnd()
}

func (p *xmlParser) isEndOfXMLComment(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN, st.BACKTICK_TOKEN, st.LT_TOKEN, st.GT_TOKEN, st.XML_COMMENT_END_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLCdataSection() st.STNode {
	cdataStart := p.consume()
	items := make([]st.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLCdata(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	content := st.CreateNodeList(items...)
	cdataEnd := p.parseXMLCdataEnd()
	return st.CreateXMLCDATANode(cdataStart, content, cdataEnd)
}

func (p *xmlParser) isEndOfXMLCdata(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN, st.BACKTICK_TOKEN, st.XML_CDATA_END_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLCdataEnd() st.STNode {
	token := p.peek()
	if token.Kind() == st.XML_CDATA_END_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_CDATA_END, false)
	return p.parseXMLCdataEnd()
}

func (p *xmlParser) parseXMLPI() st.STNode {
	p.startContext(common.PARSER_RULE_CONTEXT_XML_PI)
	piStart := p.parseXMLPIStart()
	target := p.parseXMLNCName()

	items := make([]st.STNode, 0)
	nextToken := p.peek()
	for !p.isEndOfXMLPI(nextToken.Kind()) {
		contentItem := p.parseXMLCharacterSet()
		items = append(items, contentItem)
		nextToken = p.peek()
	}
	data := st.CreateNodeList(items...)
	piEnd := p.parseXMLPIEnd()
	p.endContext()
	return st.CreateXMLProcessingInstruction(piStart, target, data, piEnd)
}

func (p *xmlParser) parseXMLPIStart() st.STNode {
	token := p.peek()
	if token.Kind() == st.XML_PI_START_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_PI_START, false)
	return p.parseXMLPIStart()
}

func (p *xmlParser) parseXMLPIEnd() st.STNode {
	token := p.peek()
	if token.Kind() == st.XML_PI_END_TOKEN {
		return p.consume()
	}
	p.recover(token, common.PARSER_RULE_CONTEXT_XML_PI_END, false)
	return p.parseXMLPIEnd()
}

func (p *xmlParser) isEndOfXMLPI(kind st.SyntaxKind) bool {
	switch kind {
	case st.EOF_TOKEN, st.BACKTICK_TOKEN, st.LT_TOKEN, st.GT_TOKEN, st.XML_PI_END_TOKEN:
		return true
	}
	return false
}

func (p *xmlParser) parseXMLCharacterSet() st.STNode {
	switch p.peek().Kind() {
	case st.XML_TEXT_CONTENT:
		return p.consume()
	case st.INTERPOLATION_START_TOKEN:
		return p.parseInterpolation()
	}
	p.internalError("xmlParser.parseXMLCharacterSet: unexpected token")
	return st.CreateEmptyNode()
}
