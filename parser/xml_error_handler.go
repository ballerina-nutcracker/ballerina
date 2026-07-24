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

// Ported from XMLParserErrorHandler.java.

package parser

import (
	"ballerina/parser/common"
)

var (
	xmlContentAlternatives = []common.ParserRuleContext{
		common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG,
		common.PARSER_RULE_CONTEXT_XML_TEXT,
		common.PARSER_RULE_CONTEXT_XML_END_TAG,
		common.PARSER_RULE_CONTEXT_XML_COMMENT_START,
		common.PARSER_RULE_CONTEXT_XML_PI,
		common.PARSER_RULE_CONTEXT_XML_CDATA_START,
	}
	xmlAttributesAlternatives = []common.ParserRuleContext{
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE,
		common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END,
	}
	xmlStartOrEmptyTagEndAlternatives = []common.ParserRuleContext{
		common.PARSER_RULE_CONTEXT_GT_TOKEN,
		common.PARSER_RULE_CONTEXT_SLASH,
	}
	xmlAttributeValueItemAlternatives = []common.ParserRuleContext{
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_TEXT,
		common.PARSER_RULE_CONTEXT_XML_QUOTE_END,
	}
	xmlPITargetRHSAlternatives = []common.ParserRuleContext{
		common.PARSER_RULE_CONTEXT_XML_PI_END,
		common.PARSER_RULE_CONTEXT_XML_PI_DATA,
	}
	xmlOptionalCDATAContentAlternatives = []common.ParserRuleContext{
		common.PARSER_RULE_CONTEXT_XML_CDATA_END,
		common.PARSER_RULE_CONTEXT_XML_CDATA_CONTENT,
	}
)

type XMLParserErrorHandler struct {
	AbstractParserErrorHandlerBase
	AbstractParserErrorHandlerMethods
}

func NewXMLParserErrorHandlerFromTokenReader(tokenReader *TokenReader) XMLParserErrorHandler {
	p := XMLParserErrorHandler{}
	p.AbstractParserErrorHandlerBase = *NewAbstractParserErrorHandlerBase(tokenReader)
	p.Self = &p
	return p
}

func (p *XMLParserErrorHandler) HasAlternativePaths(currentCtx common.ParserRuleContext) bool {
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_XML_CONTENT,
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES,
		common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END,
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_ITEM,
		common.PARSER_RULE_CONTEXT_XML_PI_TARGET_RHS,
		common.PARSER_RULE_CONTEXT_XML_OPTIONAL_CDATA_CONTENT:
		return true
	default:
		return false
	}
}

func (p *XMLParserErrorHandler) SeekMatch(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) *Result {
	var hasMatch bool
	var skipRule bool
	matchingRulesCount := 0

	for currentDepth < LOOKAHEAD_LIMIT {
		hasMatch = true
		skipRule = false
		nextToken := p.tokenReader.PeekN(lookahead)

		// Skip interpolation head + close brace so any underlying token lines up with the rule.
		if nextToken.Kind() == common.INTERPOLATION_START_TOKEN {
			lookahead += 2
			nextToken = p.tokenReader.PeekN(lookahead)
		}

		switch currentCtx {
		case common.PARSER_RULE_CONTEXT_EOF:
			hasMatch = nextToken.Kind() == common.EOF_TOKEN
		case common.PARSER_RULE_CONTEXT_LT_TOKEN:
			hasMatch = nextToken.Kind() == common.LT_TOKEN
		case common.PARSER_RULE_CONTEXT_GT_TOKEN:
			hasMatch = nextToken.Kind() == common.GT_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_NAME:
			hasMatch = nextToken.Kind() == common.IDENTIFIER_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_TEXT:
			hasMatch = nextToken.Kind() == common.XML_TEXT
		case common.PARSER_RULE_CONTEXT_SLASH:
			hasMatch = nextToken.Kind() == common.SLASH_TOKEN
		case common.PARSER_RULE_CONTEXT_ASSIGN_OP:
			hasMatch = nextToken.Kind() == common.EQUAL_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_COMMENT_START:
			hasMatch = nextToken.Kind() == common.XML_COMMENT_START_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_COMMENT_CONTENT,
			common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_TEXT,
			common.PARSER_RULE_CONTEXT_XML_PI_DATA,
			common.PARSER_RULE_CONTEXT_XML_CDATA_CONTENT:
			hasMatch = nextToken.Kind() == common.XML_TEXT_CONTENT
		case common.PARSER_RULE_CONTEXT_XML_COMMENT_END:
			hasMatch = nextToken.Kind() == common.XML_COMMENT_END_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_PI_START:
			hasMatch = nextToken.Kind() == common.XML_PI_START_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_PI_END:
			hasMatch = nextToken.Kind() == common.XML_PI_END_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_QUOTE_START,
			common.PARSER_RULE_CONTEXT_XML_QUOTE_END:
			hasMatch = nextToken.Kind() == common.DOUBLE_QUOTE_TOKEN || nextToken.Kind() == common.SINGLE_QUOTE_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_CDATA_START:
			hasMatch = nextToken.Kind() == common.XML_CDATA_START_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_CDATA_END:
			hasMatch = nextToken.Kind() == common.XML_CDATA_END_TOKEN
		default:
			if p.HasAlternativePaths(currentCtx) {
				result := p.seekMatchInAlternativePaths(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
				return result
			}
			skipRule = true
		}

		if !hasMatch {
			return p.fixAndContinue(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
		}

		currentCtx = p.GetNextRule(currentCtx, lookahead+1)
		if !skipRule {
			currentDepth++
			matchingRulesCount++
			lookahead++
			isEntryPoint = false
		}
	}

	result := NewResult(make([]*Solution, 0), matchingRulesCount)
	result.solution = NewSolution(ACTION_KEEP, currentCtx, p.GetExpectedTokenKind(currentCtx), currentCtx.String())
	return result
}

func (p *XMLParserErrorHandler) seekMatchInAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) *Result {
	var alternatives []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_XML_CONTENT:
		alternatives = xmlContentAlternatives
	case common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES:
		alternatives = xmlAttributesAlternatives
	case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END:
		alternatives = xmlStartOrEmptyTagEndAlternatives
	case common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_ITEM:
		alternatives = xmlAttributeValueItemAlternatives
	case common.PARSER_RULE_CONTEXT_XML_PI_TARGET_RHS:
		alternatives = xmlPITargetRHSAlternatives
	case common.PARSER_RULE_CONTEXT_XML_OPTIONAL_CDATA_CONTENT:
		alternatives = xmlOptionalCDATAContentAlternatives
	default:
		panic("XMLParserErrorHandler.seekMatchInAlternativePaths: " + currentCtx.String())
	}
	return p.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternatives, isEntryPoint)
}

func (p *XMLParserErrorHandler) GetNextRule(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG,
		common.PARSER_RULE_CONTEXT_XML_END_TAG,
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES,
		common.PARSER_RULE_CONTEXT_XML_PI:
		p.StartContext(currentCtx)
	}

	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG,
		common.PARSER_RULE_CONTEXT_XML_END_TAG:
		return common.PARSER_RULE_CONTEXT_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_LT_TOKEN:
		parent := p.GetParentContext()
		switch parent {
		case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG:
			return common.PARSER_RULE_CONTEXT_XML_NAME
		case common.PARSER_RULE_CONTEXT_XML_END_TAG:
			return common.PARSER_RULE_CONTEXT_SLASH
		}
		panic("< cannot exist in: " + parent.String())
	case common.PARSER_RULE_CONTEXT_GT_TOKEN,
		common.PARSER_RULE_CONTEXT_XML_PI_END:
		p.EndContext()
		return common.PARSER_RULE_CONTEXT_XML_CONTENT
	case common.PARSER_RULE_CONTEXT_XML_NAME:
		parent := p.GetParentContext()
		switch parent {
		case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG:
			return common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES
		case common.PARSER_RULE_CONTEXT_XML_END_TAG:
			return common.PARSER_RULE_CONTEXT_GT_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES:
			return common.PARSER_RULE_CONTEXT_ASSIGN_OP
		case common.PARSER_RULE_CONTEXT_XML_PI:
			return common.PARSER_RULE_CONTEXT_XML_PI_TARGET_RHS
		}
		panic("XML name cannot exist in: " + parent.String())
	case common.PARSER_RULE_CONTEXT_SLASH:
		parent := p.GetParentContext()
		switch parent {
		case common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES:
			p.EndContext()
			return common.PARSER_RULE_CONTEXT_GT_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG:
			return common.PARSER_RULE_CONTEXT_GT_TOKEN
		case common.PARSER_RULE_CONTEXT_XML_END_TAG:
			return common.PARSER_RULE_CONTEXT_XML_NAME
		}
		panic("slash cannot exist in: " + parent.String())
	case common.PARSER_RULE_CONTEXT_ASSIGN_OP:
		return common.PARSER_RULE_CONTEXT_XML_QUOTE_START
	case common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE:
		return common.PARSER_RULE_CONTEXT_XML_NAME
	case common.PARSER_RULE_CONTEXT_XML_QUOTE_END:
		return common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES
	case common.PARSER_RULE_CONTEXT_XML_COMMENT_START:
		return common.PARSER_RULE_CONTEXT_XML_COMMENT_CONTENT
	case common.PARSER_RULE_CONTEXT_XML_COMMENT_CONTENT:
		return common.PARSER_RULE_CONTEXT_XML_COMMENT_END
	case common.PARSER_RULE_CONTEXT_XML_TEXT,
		common.PARSER_RULE_CONTEXT_XML_COMMENT_END:
		return common.PARSER_RULE_CONTEXT_XML_CONTENT
	case common.PARSER_RULE_CONTEXT_XML_PI:
		return common.PARSER_RULE_CONTEXT_XML_PI_START
	case common.PARSER_RULE_CONTEXT_XML_PI_START:
		return common.PARSER_RULE_CONTEXT_XML_NAME
	case common.PARSER_RULE_CONTEXT_XML_PI_DATA:
		return common.PARSER_RULE_CONTEXT_XML_PI_END
	case common.PARSER_RULE_CONTEXT_XML_QUOTE_START,
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_TEXT:
		return common.PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_ITEM
	case common.PARSER_RULE_CONTEXT_XML_CDATA_START:
		return common.PARSER_RULE_CONTEXT_XML_OPTIONAL_CDATA_CONTENT
	case common.PARSER_RULE_CONTEXT_XML_CDATA_CONTENT:
		return common.PARSER_RULE_CONTEXT_XML_CDATA_END
	case common.PARSER_RULE_CONTEXT_XML_CDATA_END:
		return common.PARSER_RULE_CONTEXT_XML_CONTENT
	}
	panic("cannot find the next rule for: " + currentCtx.String())
}

func (p *XMLParserErrorHandler) GetInsertSolution(ctx common.ParserRuleContext) *Solution {
	kind := p.GetExpectedTokenKind(ctx)
	return NewSolution(ACTION_INSERT, ctx, kind, ctx.String())
}

func (p *XMLParserErrorHandler) GetExpectedTokenKind(ctx common.ParserRuleContext) common.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_LT_TOKEN,
		common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG,
		common.PARSER_RULE_CONTEXT_XML_END_TAG:
		return common.LT_TOKEN
	case common.PARSER_RULE_CONTEXT_GT_TOKEN:
		return common.GT_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH:
		return common.SLASH_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_KEYWORD:
		return common.XML_KEYWORD
	case common.PARSER_RULE_CONTEXT_XML_NAME:
		return common.IDENTIFIER_TOKEN
	case common.PARSER_RULE_CONTEXT_ASSIGN_OP:
		return common.EQUAL_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END,
		common.PARSER_RULE_CONTEXT_XML_ATTRIBUTES:
		return common.GT_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_CONTENT,
		common.PARSER_RULE_CONTEXT_XML_TEXT:
		return common.BACKTICK_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_COMMENT_START:
		return common.XML_COMMENT_START_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_COMMENT_CONTENT:
		return common.XML_TEXT_CONTENT
	case common.PARSER_RULE_CONTEXT_XML_COMMENT_END:
		return common.XML_COMMENT_END_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_PI,
		common.PARSER_RULE_CONTEXT_XML_PI_START:
		return common.XML_PI_START_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_PI_END:
		return common.XML_PI_END_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_PI_DATA:
		return common.XML_TEXT_CONTENT
	case common.PARSER_RULE_CONTEXT_XML_QUOTE_END,
		common.PARSER_RULE_CONTEXT_XML_QUOTE_START:
		return common.DOUBLE_QUOTE_TOKEN
	case common.PARSER_RULE_CONTEXT_XML_CDATA_END:
		return common.XML_CDATA_END_TOKEN
	}
	return common.NONE
}

var (
	_ AbstractParserErrorHandler = (*XMLParserErrorHandler)(nil)
	_ ParserErrorHandler         = (*XMLParserErrorHandler)(nil)
)
