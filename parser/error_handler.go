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
	debugcommon "ballerina/common"
	"ballerina/parser/common"
	"ballerina/parser/tree"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
)

func logRecoveredPanic(ctx common.ParserRuleContext, location string, recovered any) {
	traceRecovery(ctx, func() string {
		stackTrace := debug.Stack()
		return fmt.Sprintf("[parser] recovered panic in %s: %v\n[parser] stack trace:\n%s", location, recovered, stackTrace)
	})
}

// ============================================================================
// String formatting helpers for error recovery tracing
// ============================================================================

func formatParserRuleContext(ctx common.ParserRuleContext) string {
	return ctx.String()
}

func formatSTToken(token tree.STToken) string {
	if token == nil {
		return "nil"
	}
	kindStr := token.Kind().StrValue()
	if kindStr == "" {
		// For tokens without StrValue (like IDENTIFIER_TOKEN), use a descriptive name
		switch token.Kind() {
		case common.IDENTIFIER_TOKEN:
			kindStr = "IDENTIFIER_TOKEN"
		case common.STRING_LITERAL_TOKEN:
			kindStr = "STRING_LITERAL_TOKEN"
		case common.DECIMAL_INTEGER_LITERAL_TOKEN:
			kindStr = "DECIMAL_INTEGER_LITERAL_TOKEN"
		case common.HEX_INTEGER_LITERAL_TOKEN:
			kindStr = "HEX_INTEGER_LITERAL_TOKEN"
		case common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			kindStr = "DECIMAL_FLOATING_POINT_LITERAL_TOKEN"
		case common.HEX_FLOATING_POINT_LITERAL_TOKEN:
			kindStr = "HEX_FLOATING_POINT_LITERAL_TOKEN"
		case common.INVALID_TOKEN:
			kindStr = "INVALID_TOKEN"
		default:
			kindStr = fmt.Sprintf("TOKEN_%d", token.Kind().Tag())
		}
	}
	return fmt.Sprintf("%s:%s", kindStr, token.Text())
}

func formatSTNode(node tree.STNode) string {
	if node == nil {
		return "nil"
	}
	return tree.ToSexpr(node)
}

func formatSolution(sol *Solution) string {
	if sol == nil {
		return "nil"
	}
	actionStr := "UNKNOWN"
	switch sol.Action {
	case ACTION_INSERT:
		actionStr = "INSERT"
	case ACTION_REMOVE:
		actionStr = "REMOVE"
	case ACTION_KEEP:
		actionStr = "KEEP"
	}
	kindStr := sol.TokenKind.StrValue()
	if kindStr == "" {
		kindStr = fmt.Sprintf("KIND_%d", sol.TokenKind.Tag())
	}
	return fmt.Sprintf("%s:%s:%s", actionStr, formatParserRuleContext(sol.Ctx), kindStr)
}

func formatContextStack(stack []common.ParserRuleContext) string {
	if stack == nil {
		return "nil"
	}
	if len(stack) == 0 {
		return "[]"
	}
	parts := make([]string, len(stack))
	for i, ctx := range stack {
		parts[i] = formatParserRuleContext(ctx)
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, " "))
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func formatResult(result *Result) string {
	if result == nil {
		return "nil"
	}
	solutionStr := formatSolution(result.solution)
	return fmt.Sprintf("matches:%d removeFixes:%d fixes:%d solution:%s", result.matches, result.removeFixes, len(result.fixes), solutionStr)
}

func formatResultValue(result Result) string {
	solutionStr := formatSolution(result.solution)
	return fmt.Sprintf("matches:%d removeFixes:%d fixes:%d solution:%s", result.matches, result.removeFixes, len(result.fixes), solutionStr)
}

func traceRecovery(ctx common.ParserRuleContext, messageFn func() string) {
	debugcommon.DebugWriteLazy(debugcommon.DEBUG_ERROR_RECOVERY, messageFn)
}

// ============================================================================
// Solution struct - represents a fix for a parser error
// ============================================================================

type Solution struct {
	Ctx           common.ParserRuleContext
	Action        Action
	TokenText     string
	TokenKind     common.SyntaxKind
	RecoveredNode tree.STNode
	RemovedToken  tree.STToken
	Depth         int
}

func NewSolution(action Action, ctx common.ParserRuleContext, tokenKind common.SyntaxKind, tokenText string) *Solution {
	return NewSolutionWithDepth(action, ctx, tokenKind, tokenText, -1)
}

func NewSolutionWithDepth(action Action, ctx common.ParserRuleContext, tokenKind common.SyntaxKind, tokenText string, depth int) *Solution {
	return &Solution{
		Action:    action,
		Ctx:       ctx,
		TokenText: tokenText,
		TokenKind: tokenKind,
		Depth:     depth,
	}
}

func (s *Solution) ToString() string {
	actionStr := "UNKNOWN"
	switch s.Action {
	case ACTION_INSERT:
		actionStr = "INSERT"
	case ACTION_REMOVE:
		actionStr = "REMOVE"
	case ACTION_KEEP:
		actionStr = "KEEP"
	}
	return actionStr + "'" + s.TokenText + "'"
}

// ============================================================================
// Result struct - holds results of error recovery attempts
// ============================================================================

type Result struct {
	matches     int
	removeFixes int
	fixes       []*Solution
	solution    *Solution
}

func NewResult(fixes []*Solution, matches int) *Result {
	return &Result{
		fixes:   fixes,
		matches: matches,
	}
}

func (r *Result) peekFix() *Solution {
	if len(r.fixes) == 0 {
		return nil
	}
	return r.fixes[len(r.fixes)-1]
}

func (r *Result) popFix() *Solution {
	if len(r.fixes) == 0 {
		return nil
	}

	sol := r.fixes[len(r.fixes)-1]
	r.fixes = r.fixes[:len(r.fixes)-1]

	if sol.Action == ACTION_REMOVE {
		r.removeFixes--
	}
	return sol
}

func (r *Result) pushFix(sol *Solution) {
	if sol.Action == ACTION_REMOVE {
		r.removeFixes++
	}
	r.fixes = append(r.fixes, sol)
}

func (r *Result) fixesSize() int {
	return len(r.fixes)
}

// ============================================================================
// Constants
// ============================================================================

var (
	LOOKAHEAD_LIMIT        = 4
	RESOLUTION_ITTER_LIMIT = 7
	COMPLETION_ITTER_LIMIT = 15
)

// ============================================================================
// AbstractParserErrorHandlerData - Field access interface
// ============================================================================

type AbstractParserErrorHandlerData interface {
	GetTokenReader() *TokenReader
	SetTokenReader(*TokenReader)
	GetCtxStack() []common.ParserRuleContext
	SetCtxStack([]common.ParserRuleContext)
	GetPreviousTokenIndex() int
	SetPreviousTokenIndex(int)
	GetItterCount() int
	SetItterCount(int)
}

// ============================================================================
// AbstractParserErrorHandlerBase - Base struct with fields
// ============================================================================

type AbstractParserErrorHandlerBase struct {
	tokenReader        *TokenReader
	ctxStack           []common.ParserRuleContext
	previousTokenIndex int
	itterCount         int
}

func NewAbstractParserErrorHandlerBase(tokenReader *TokenReader) *AbstractParserErrorHandlerBase {
	return &AbstractParserErrorHandlerBase{
		tokenReader:        tokenReader,
		ctxStack:           make([]common.ParserRuleContext, 0),
		previousTokenIndex: -1,
		itterCount:         0,
	}
}

// Getter/setter implementations for AbstractParserErrorHandlerBase

func (b *AbstractParserErrorHandlerBase) GetTokenReader() *TokenReader {
	return b.tokenReader
}

func (b *AbstractParserErrorHandlerBase) SetTokenReader(tokenReader *TokenReader) {
	b.tokenReader = tokenReader
}

func (b *AbstractParserErrorHandlerBase) GetCtxStack() []common.ParserRuleContext {
	return b.ctxStack
}

func (b *AbstractParserErrorHandlerBase) SetCtxStack(ctxStack []common.ParserRuleContext) {
	b.ctxStack = ctxStack
}

func (b *AbstractParserErrorHandlerBase) GetPreviousTokenIndex() int {
	return b.previousTokenIndex
}

func (b *AbstractParserErrorHandlerBase) SetPreviousTokenIndex(previousTokenIndex int) {
	b.previousTokenIndex = previousTokenIndex
}

func (b *AbstractParserErrorHandlerBase) GetItterCount() int {
	return b.itterCount
}

func (b *AbstractParserErrorHandlerBase) SetItterCount(itterCount int) {
	b.itterCount = itterCount
}

// ============================================================================
// AbstractParserErrorHandler - Main interface
// ============================================================================

type AbstractParserTracer any

type AbstractParserErrorHandler interface {
	AbstractParserErrorHandlerData
	AbstractParserTracer

	// Abstract methods (to be implemented by concrete classes like BallerinaParserErrorHandler)
	HasAlternativePaths(context common.ParserRuleContext) bool
	SeekMatch(context common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) *Result
	GetNextRule(context common.ParserRuleContext, nextLookahead int) common.ParserRuleContext
	GetExpectedTokenKind(context common.ParserRuleContext) common.SyntaxKind
	GetInsertSolution(context common.ParserRuleContext) *Solution

	// Default/concrete methods (implemented in AbstractParserErrorHandlerMethods)
	Recover(currentCtx common.ParserRuleContext, nextToken tree.STToken, isCompletion bool) *Solution
	ConsumeInvalidToken() tree.STToken
	StartContext(context common.ParserRuleContext)
	EndContext()
	SwitchContext(context common.ParserRuleContext)
	GetParentContext() common.ParserRuleContext
	GetGrandParentContext() common.ParserRuleContext
	HasAncestorContext(context common.ParserRuleContext) bool
	GetContextStack() []common.ParserRuleContext
}

// ============================================================================
// AbstractParserErrorHandlerMethods - Default method implementations
// ============================================================================

type AbstractParserErrorHandlerMethods struct {
	Self AbstractParserErrorHandler
}

func (m *AbstractParserErrorHandlerMethods) Recover(currentCtx common.ParserRuleContext, nextToken tree.STToken, isCompletion bool) (result *Solution) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(Recover start %s %s %s)",
			formatParserRuleContext(currentCtx),
			formatSTToken(nextToken),
			formatBool(isCompletion))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(Recover end (%s %s %s) %s)", formatParserRuleContext(currentCtx), formatSTToken(nextToken), formatBool(isCompletion), formatSolution(result))
	})

	currentTokenIndex := m.Self.GetTokenReader().GetCurrentTokenIndex()
	if currentTokenIndex == m.Self.GetPreviousTokenIndex() {
		m.Self.SetItterCount(m.Self.GetItterCount() + 1)
	} else {
		m.Self.SetItterCount(0)
		m.Self.SetPreviousTokenIndex(currentTokenIndex)
	}
	var fix *Solution
	if isCompletion && (m.Self.GetItterCount() < COMPLETION_ITTER_LIMIT) {
		fix = m.getCompletion(currentCtx, nextToken)
	} else if m.Self.GetItterCount() < RESOLUTION_ITTER_LIMIT {
		fix = m.getResolution(currentCtx, nextToken)
	}
	if fix != nil {
		m.applyFix(currentCtx, fix)
		return fix
	}
	// Fail safe. This means we can't find a path to recover.
	if isCompletion {
		if m.Self.GetItterCount() == COMPLETION_ITTER_LIMIT {
			traceRecovery(currentCtx, func() string {
				return "fail safe reached"
			})
		}
	} else {
		if m.Self.GetItterCount() == RESOLUTION_ITTER_LIMIT {
			traceRecovery(currentCtx, func() string {
				return "fail safe reached"
			})
		}
	}
	return m.getFailSafeSolution(currentCtx, nextToken)
}

func (m *AbstractParserErrorHandlerMethods) getResolution(currentCtx common.ParserRuleContext, nextToken tree.STToken) *Solution {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(getResolution start %s %s)",
			formatParserRuleContext(currentCtx),
			formatSTToken(nextToken))
	})
	bestMatch := m.seekMatchStart(currentCtx)
	m.validateSolution(bestMatch, currentCtx, nextToken)
	var sol *Solution
	if bestMatch.matches > 0 {
		sol = bestMatch.solution
	}
	return sol
}

func (m *AbstractParserErrorHandlerMethods) getFailSafeSolution(currentCtx common.ParserRuleContext, nextToken tree.STToken) *Solution {
	sol := NewSolution(ACTION_REMOVE, currentCtx, nextToken.Kind(), nextToken.Text())
	sol.RemovedToken = m.Self.ConsumeInvalidToken()
	return sol
}

func (m *AbstractParserErrorHandlerMethods) validateSolution(bestMatch *Result, currentCtx common.ParserRuleContext, nextToken tree.STNode) {
	sol := bestMatch.solution
	if (sol == nil) || (sol.Action == ACTION_REMOVE) {
		return
	}
	if (sol.Action == ACTION_KEEP) && (nextToken.Kind() == common.DOCUMENTATION_STRING) {
		bestMatch.solution = NewSolution(ACTION_REMOVE, currentCtx, common.DOCUMENTATION_STRING, currentCtx.String())
	}
	if (sol.Action != ACTION_INSERT) || (bestMatch.fixesSize() < 2) {
		return
	}
	firstFix := bestMatch.popFix()
	secondFix := bestMatch.peekFix()
	bestMatch.pushFix(firstFix)
	if (secondFix.Action == ACTION_REMOVE) && (secondFix.Depth == 1) {
		bestMatch.solution = secondFix
	}
}

func (m *AbstractParserErrorHandlerMethods) getCompletion(context common.ParserRuleContext, nextToken tree.STToken) *Solution {
	tempCtxStack := m.Self.GetCtxStack()
	m.Self.SetCtxStack(m.getCtxStackSnapshot())
	var sol *Solution
	func() {
		// TODO: check if we panic inside this method
		defer func() {
			if r := recover(); r != nil {
				logRecoveredPanic(context, "getCompletion", r)
				if false {
					panic("assertion failed")
				}
				sol = m.getResolution(context, nextToken)
			}
		}()
		sol = m.Self.GetInsertSolution(context)
	}()

	m.Self.SetCtxStack(tempCtxStack)
	return sol
}

func (m *AbstractParserErrorHandlerMethods) ConsumeInvalidToken() (result tree.STToken) {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return "(ConsumeInvalidToken start)"
	})
	defer traceRecovery(ctx, func() string {
		return fmt.Sprintf("(ConsumeInvalidToken end %s)", formatSTToken(result))
	})
	return m.Self.GetTokenReader().Read()
}

func (m *AbstractParserErrorHandlerMethods) applyFix(currentCtx common.ParserRuleContext, fix *Solution) {
	switch fix.Action {
	case ACTION_REMOVE:
		fix.RemovedToken = m.Self.ConsumeInvalidToken()
		fix.RecoveredNode = m.Self.GetTokenReader().Peek()
		fix.TokenKind = m.Self.GetTokenReader().Peek().Kind()
	case ACTION_INSERT:
		fix.RecoveredNode = m.handleMissingToken(currentCtx, fix)
	}
}

func (m *AbstractParserErrorHandlerMethods) handleMissingToken(currentCtx common.ParserRuleContext, fix *Solution) tree.STNode {
	return tree.CreateMissingTokenWithDiagnosticsFromParserRules(fix.TokenKind, fix.Ctx)
}

func (m *AbstractParserErrorHandlerMethods) getCtxStackSnapshot() []common.ParserRuleContext {
	ctxStack := m.Self.GetCtxStack()
	snapshot := make([]common.ParserRuleContext, len(ctxStack))
	copy(snapshot, ctxStack)
	return snapshot
}

func (m *AbstractParserErrorHandlerMethods) seekMatchStart(currentCtx common.ParserRuleContext) (bestMatch *Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchStart start %s)", formatParserRuleContext(currentCtx))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchStart end (%s) %s)", formatParserRuleContext(currentCtx), formatResult(bestMatch))
	})
	tempCtxStack := m.Self.GetCtxStack()
	func() {
		defer func() {
			if r := recover(); r != nil {
				logRecoveredPanic(currentCtx, "seekMatchStart", r)
				if false {
					panic("assertion failed")
				}
				bestMatch = NewResult(make([]*Solution, 0), LOOKAHEAD_LIMIT-1)
				bestMatch.solution = NewSolution(ACTION_REMOVE, currentCtx, common.SyntaxKind(0), currentCtx.String())
			}
		}()
		bestMatch = m.seekMatchInSubTree(currentCtx, 1, 0, true)
	}()
	m.Self.SetCtxStack(tempCtxStack)

	return bestMatch
}

func (m *AbstractParserErrorHandlerMethods) seekMatchInSubTree(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) (result *Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInSubTree start %s %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInSubTree end (%s %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatBool(isEntryPoint), formatResult(result))
	})
	tempCtxStack := m.Self.GetCtxStack()
	m.Self.SetCtxStack(m.getCtxStackSnapshot())
	result = m.Self.SeekMatch(currentCtx, lookahead, currentDepth, isEntryPoint)
	m.Self.SetCtxStack(tempCtxStack)
	return result
}

func (m *AbstractParserErrorHandlerMethods) StartContext(context common.ParserRuleContext) {
	traceRecovery(context, func() string {
		return fmt.Sprintf("(StartContext start %s)", formatParserRuleContext(context))
	})
	ctxStack := m.Self.GetCtxStack()
	m.Self.SetCtxStack(append(ctxStack, context))
	traceRecovery(context, func() string {
		return fmt.Sprintf("(StartContext end (%s))", formatParserRuleContext(context))
	})
}

func (m *AbstractParserErrorHandlerMethods) EndContext() {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return "(EndContext start)"
	})
	ctxStack = m.Self.GetCtxStack()
	m.Self.SetCtxStack(ctxStack[:len(ctxStack)-1])
	traceRecovery(ctx, func() string {
		return "(EndContext end)"
	})
}

func (m *AbstractParserErrorHandlerMethods) SwitchContext(context common.ParserRuleContext) {
	traceRecovery(context, func() string {
		return fmt.Sprintf("(SwitchContext start %s)", formatParserRuleContext(context))
	})
	ctxStack := m.Self.GetCtxStack()
	ctxStack = ctxStack[:len(ctxStack)-1]
	m.Self.SetCtxStack(append(ctxStack, context))
	traceRecovery(context, func() string {
		return "(SwitchContext end)"
	})
}

func (m *AbstractParserErrorHandlerMethods) GetParentContext() (result common.ParserRuleContext) {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return "(GetParentContext start)"
	})
	defer traceRecovery(ctx, func() string {
		return fmt.Sprintf("(GetParentContext end %s)", formatParserRuleContext(result))
	})
	ctxStack = m.Self.GetCtxStack()
	return ctxStack[len(ctxStack)-1]
}

func (m *AbstractParserErrorHandlerMethods) GetGrandParentContext() (result common.ParserRuleContext) {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return "(GetGrandParentContext start)"
	})
	defer traceRecovery(ctx, func() string {
		return fmt.Sprintf("(GetGrandParentContext end %s)", formatParserRuleContext(result))
	})
	ctxStack = m.Self.GetCtxStack()
	parent := ctxStack[len(ctxStack)-1]
	ctxStack = ctxStack[:len(ctxStack)-1]

	grandParent := ctxStack[len(ctxStack)-1]

	m.Self.SetCtxStack(append(ctxStack, parent))
	return grandParent
}

func (m *AbstractParserErrorHandlerMethods) HasAncestorContext(context common.ParserRuleContext) (result bool) {
	traceRecovery(context, func() string {
		return fmt.Sprintf("(HasAncestorContext start %s)", formatParserRuleContext(context))
	})
	defer traceRecovery(context, func() string {
		return fmt.Sprintf("(HasAncestorContext end (%s) %s)", formatParserRuleContext(context), formatBool(result))
	})
	ctxStack := m.Self.GetCtxStack()
	return slices.Contains(ctxStack, context)
}

func (m *AbstractParserErrorHandlerMethods) GetContextStack() (result []common.ParserRuleContext) {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return "(GetContextStack start)"
	})
	defer traceRecovery(ctx, func() string {
		return fmt.Sprintf("(GetContextStack end %s)", formatContextStack(result))
	})
	return m.Self.GetCtxStack()
}

func (m *AbstractParserErrorHandlerMethods) seekInAlternativesPaths(lookahead int, currentDepth int, currentMatches int, alternativeRules []common.ParserRuleContext, isEntryPoint bool) (result *Result) {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return fmt.Sprintf("(seekInAlternativesPaths start %d %d %d %s %s)", lookahead, currentDepth, currentMatches, formatContextStack(alternativeRules), formatBool(isEntryPoint))
	})
	defer traceRecovery(ctx, func() string {
		return fmt.Sprintf("(seekInAlternativesPaths end (%d %d %d %s %s) %s)", lookahead, currentDepth, currentMatches, formatContextStack(alternativeRules), formatBool(isEntryPoint), formatResult(result))
	})
	results := make([][]*Result, LOOKAHEAD_LIMIT)
	bestMatchIndex := 0

	for _, rule := range alternativeRules {
		tempCtxStack := m.Self.GetCtxStack()
		var result *Result
		shouldContinue := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					logRecoveredPanic(rule, "seekInAlternativesPaths", r)
					if false {
						panic("assertion failed")
					}
					shouldContinue = true
				}
			}()
			result = m.seekMatchInSubTree(rule, lookahead, currentDepth, isEntryPoint)
		}()
		m.Self.SetCtxStack(tempCtxStack)

		if shouldContinue {
			continue
		}

		if m.hasFoundBestAlternative(result) {
			return m.getFinalResult(currentMatches, result)
		}
		similarResults := results[result.matches]
		if similarResults == nil {
			similarResults = make([]*Result, 0)
			results[result.matches] = similarResults
			if bestMatchIndex < result.matches {
				bestMatchIndex = result.matches
			}
		}
		results[result.matches] = append(results[result.matches], result)
	}

	bestMatches := results[bestMatchIndex]
	bestMatch := bestMatches[0]
	for i := 1; i < len(bestMatches); i++ {
		currentMatch := bestMatches[i]
		currentMatchRemoveFixes := currentMatch.removeFixes
		bestMatchRemoveFixes := bestMatch.removeFixes
		if bestMatchRemoveFixes == 0 {
			break
		}
		if currentMatchRemoveFixes == bestMatchRemoveFixes {
			currentSol := bestMatch.peekFix()
			foundSol := currentMatch.peekFix()
			if (currentSol.Action == ACTION_REMOVE) && (foundSol.Action == ACTION_INSERT) {
				bestMatch = currentMatch
			}
		} else if currentMatchRemoveFixes < bestMatchRemoveFixes {
			bestMatch = currentMatch
		}
	}
	return m.getFinalResult(currentMatches, bestMatch)
}

func (m *AbstractParserErrorHandlerMethods) hasFoundBestAlternative(result *Result) bool {
	if result.matches < (LOOKAHEAD_LIMIT - 1) {
		return false
	}
	if result.solution == nil {
		return true
	}
	return (result.solution.Action != ACTION_REMOVE)
}

func (m *AbstractParserErrorHandlerMethods) getFinalResult(currentMatches int, bestMatch *Result) (result *Result) {
	ctxStack := m.Self.GetCtxStack()
	var ctx common.ParserRuleContext
	if len(ctxStack) > 0 {
		ctx = ctxStack[len(ctxStack)-1]
	}
	traceRecovery(ctx, func() string {
		return fmt.Sprintf("(getFinalResult start %d %s)", currentMatches, formatResult(bestMatch))
	})
	defer traceRecovery(ctx, func() string {
		return fmt.Sprintf("(getFinalResult end (%d %s) %s)", currentMatches, formatResult(bestMatch), formatResult(result))
	})
	bestMatch.matches += currentMatches
	return bestMatch
}

func (m *AbstractParserErrorHandlerMethods) fixAndContinue(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result *Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(fixAndContinue start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(fixAndContinue end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResult(result))
	})
	fixedPathResult := m.fixAndContinueCore(currentCtx, lookahead, currentDepth)
	if isEntryPoint {
		fixedPathResult.solution = fixedPathResult.peekFix()
	} else {
		fixedPathResult.solution = NewSolution(ACTION_KEEP, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String())
	}
	return m.getFinalResult(matchingRulesCount, fixedPathResult)
}

func (m *AbstractParserErrorHandlerMethods) fixAndContinueCore(currentCtx common.ParserRuleContext, lookahead int, currentDepth int) (fixedPathResult *Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(fixAndContinueCore start %s %d %d)", formatParserRuleContext(currentCtx), lookahead, currentDepth)
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(fixAndContinueCore end (%s %d %d) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatResult(fixedPathResult))
	})
	deletionResult := m.seekMatchInSubTree(currentCtx, lookahead+1, currentDepth+1, false)
	nextCtx := m.Self.GetNextRule(currentCtx, lookahead)
	insertionResult := m.seekMatchInSubTree(nextCtx, lookahead, currentDepth+1, false)
	var action *Solution

	if (insertionResult.matches == 0) && (deletionResult.matches == 0) {
		action = NewSolutionWithDepth(ACTION_INSERT, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String(), currentDepth)
		insertionResult.pushFix(action)
		fixedPathResult = insertionResult
	} else if insertionResult.matches == deletionResult.matches {
		if insertionResult.removeFixes <= (deletionResult.removeFixes + 1) {
			action = NewSolutionWithDepth(ACTION_INSERT, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String(), currentDepth)
			insertionResult.pushFix(action)
			fixedPathResult = insertionResult
		} else {
			token := m.Self.GetTokenReader().PeekN(lookahead)
			action = NewSolutionWithDepth(ACTION_REMOVE, currentCtx, token.Kind(), token.Text(), currentDepth)
			deletionResult.pushFix(action)
			fixedPathResult = deletionResult
		}
	} else if insertionResult.matches > deletionResult.matches {
		action = NewSolutionWithDepth(ACTION_INSERT, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String(), currentDepth)
		insertionResult.pushFix(action)
		fixedPathResult = insertionResult
	} else {
		token := m.Self.GetTokenReader().PeekN(lookahead)
		action = NewSolutionWithDepth(ACTION_REMOVE, currentCtx, token.Kind(), token.Text(), currentDepth)
		deletionResult.pushFix(action)
		fixedPathResult = deletionResult
	}
	return fixedPathResult
}

type BallerinaParserErrorHandler struct {
	AbstractParserErrorHandlerBase
	AbstractParserErrorHandlerMethods
}

var (
	FUNC_TYPE_OR_DEF_OPTIONAL_RETURNS                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS}
	FUNC_BODY_OR_TYPE_DESC_RHS                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_BODY, common.PARSER_RULE_CONTEXT_MODULE_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS}
	FUNC_DEF_OPTIONAL_RETURNS                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_BODY}
	METHOD_DECL_OPTIONAL_RETURNS                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	FUNC_BODY                                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK, common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY}
	EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATIONS, common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD}
	ANNON_FUNC_OPTIONAL_RETURNS                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY}
	ANON_FUNC_BODY                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK, common.PARSER_RULE_CONTEXT_EXPLICIT_ANON_FUNC_EXPR_BODY_START}
	FUNC_TYPE_OPTIONAL_RETURNS                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END}
	FUNC_TYPE_OR_ANON_FUNC_OPTIONAL_RETURNS                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY}
	FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY, common.PARSER_RULE_CONTEXT_STMT_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS}
	WORKER_NAME_RHS                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_BLOCK_STMT}
	STATEMENTS                                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_IF_BLOCK, common.PARSER_RULE_CONTEXT_WHILE_BLOCK, common.PARSER_RULE_CONTEXT_CALL_STMT, common.PARSER_RULE_CONTEXT_PANIC_STMT, common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT, common.PARSER_RULE_CONTEXT_BREAK_STATEMENT, common.PARSER_RULE_CONTEXT_RETURN_STMT, common.PARSER_RULE_CONTEXT_MATCH_STMT, common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT, common.PARSER_RULE_CONTEXT_LOCK_STMT, common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL, common.PARSER_RULE_CONTEXT_FORK_STMT, common.PARSER_RULE_CONTEXT_FOREACH_STMT, common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION, common.PARSER_RULE_CONTEXT_TRANSACTION_STMT, common.PARSER_RULE_CONTEXT_RETRY_STMT, common.PARSER_RULE_CONTEXT_ROLLBACK_STMT, common.PARSER_RULE_CONTEXT_DO_BLOCK, common.PARSER_RULE_CONTEXT_FAIL_STATEMENT, common.PARSER_RULE_CONTEXT_BLOCK_STMT}
	ASSIGNMENT_STMT_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR}
	VAR_DECL_RHS                                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_SEMICOLON}
	TOP_LEVEL_NODE                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EOF, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	TOP_LEVEL_NODE_WITHOUT_METADATA                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EOF, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER, common.PARSER_RULE_CONTEXT_PUBLIC_KEYWORD}
	TOP_LEVEL_NODE_WITHOUT_MODIFIER                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EOF, common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL, common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION, common.PARSER_RULE_CONTEXT_SERVICE_DECL, common.PARSER_RULE_CONTEXT_LISTENER_DECL, common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION, common.PARSER_RULE_CONTEXT_CONSTANT_DECL, common.PARSER_RULE_CONTEXT_ANNOTATION_DECL, common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION, common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION, common.PARSER_RULE_CONTEXT_IMPORT_DECL}
	FUNC_DEF_START                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER}
	FUNC_DEF_WITHOUT_FIRST_QUALIFIER                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER}
	TYPE_OR_VAR_NAME                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN}
	FIELD_DESCRIPTOR_RHS                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_QUESTION_MARK, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	FIELD_OR_REST_DESCIPTOR_RHS                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	RECORD_BODY_START                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START, common.PARSER_RULE_CONTEXT_OPEN_BRACE}
	RECORD_BODY_END                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	TYPE_DESCRIPTORS                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE, common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE, common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START, common.PARSER_RULE_CONTEXT_STREAM_KEYWORD, common.PARSER_RULE_CONTEXT_TABLE_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC, common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION, common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START}
	TYPE_DESCRIPTOR_WITHOUT_ISOLATED                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR}
	CLASS_DESCRIPTOR                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_REFERENCE, common.PARSER_RULE_CONTEXT_STREAM_KEYWORD}
	RECORD_FIELD_OR_RECORD_END                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RECORD_BODY_END, common.PARSER_RULE_CONTEXT_RECORD_FIELD}
	RECORD_FIELD_START                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD, common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	RECORD_FIELD_WITHOUT_METADATA                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD}
	ARG_START_OR_ARG_LIST_END                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_END, common.PARSER_RULE_CONTEXT_ARG_START}
	ARG_START                                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_ELLIPSIS, common.PARSER_RULE_CONTEXT_EXPRESSION}
	ARG_END                                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_END, common.PARSER_RULE_CONTEXT_COMMA}
	NAMED_OR_POSITIONAL_ARG_RHS                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_END, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	OPTIONAL_FIELD_INITIALIZER                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_SEMICOLON}
	ON_FAIL_OPTIONAL_BINDING_PATTERN                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BLOCK_STMT, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN}
	GROUPING_KEY_LIST_ELEMENT                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY}
	GROUPING_KEY_LIST_ELEMENT_END                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE_END, common.PARSER_RULE_CONTEXT_COMMA}
	CLASS_MEMBER_OR_OBJECT_MEMBER_START                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	OBJECT_CONSTRUCTOR_MEMBER_START                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	OBJECT_CONS_MEMBER_WITHOUT_META                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	OBJECT_FUNC_OR_FIELD                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY, common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL}
	OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FIELD_START, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_START}
	OBJECT_FIELD_QUALIFIER                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER, common.PARSER_RULE_CONTEXT_FINAL_KEYWORD}
	OBJECT_METHOD_START                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER}
	OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER}
	OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER}
	OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER}
	OBJECT_TYPE_START                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER}
	OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER}
	OBJECT_CONSTRUCTOR_START                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER}
	OBJECT_CONS_WITHOUT_FIRST_QUALIFIER                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER}
	OBJECT_CONSTRUCTOR_RHS                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_BRACE, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	ELSE_BODY                                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_IF_BLOCK, common.PARSER_RULE_CONTEXT_OPEN_BRACE}
	ELSE_BLOCK                                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ELSE_KEYWORD, common.PARSER_RULE_CONTEXT_STATEMENT}
	CALL_STATEMENT                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD, common.PARSER_RULE_CONTEXT_VARIABLE_REF, common.PARSER_RULE_CONTEXT_EXPRESSION}
	IMPORT_PREFIX_DECL                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	IMPORT_DECL_ORG_OR_MODULE_NAME_RHS                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME}
	AFTER_IMPORT_MODULE_NAME                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_SEMICOLON}
	MAJOR_MINOR_VERSION_END                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	RETURN_RHS                                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_EXPRESSION}
	EXPRESSION_START                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BASIC_LITERAL, common.PARSER_RULE_CONTEXT_NIL_LITERAL, common.PARSER_RULE_CONTEXT_VARIABLE_REF, common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION, common.PARSER_RULE_CONTEXT_TYPE_CAST, common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION, common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_LET_EXPRESSION, common.PARSER_RULE_CONTEXT_TEMPLATE_START, common.PARSER_RULE_CONTEXT_XML_KEYWORD, common.PARSER_RULE_CONTEXT_STRING_KEYWORD, common.PARSER_RULE_CONTEXT_BASE16_KEYWORD, common.PARSER_RULE_CONTEXT_BASE64_KEYWORD, common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION, common.PARSER_RULE_CONTEXT_ERROR_KEYWORD, common.PARSER_RULE_CONTEXT_NEW_KEYWORD, common.PARSER_RULE_CONTEXT_START_KEYWORD, common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD, common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN, common.PARSER_RULE_CONTEXT_WAIT_KEYWORD, common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD, common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD, common.PARSER_RULE_CONTEXT_TYPEOF_EXPRESSION, common.PARSER_RULE_CONTEXT_TRAP_KEYWORD, common.PARSER_RULE_CONTEXT_UNARY_EXPRESSION, common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD, common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_RE_KEYWORD, common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION}
	FIRST_MAPPING_FIELD_START                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MAPPING_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	MAPPING_FIELD_START                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD, common.PARSER_RULE_CONTEXT_ELLIPSIS, common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME, common.PARSER_RULE_CONTEXT_READONLY_KEYWORD}
	SPECIFIC_FIELD                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME, common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN}
	SPECIFIC_FIELD_RHS                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END}
	MAPPING_FIELD_END                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA}
	CONST_DECL_RHS                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	ARRAY_LENGTH                                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN, common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN, common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_VARIABLE_REF}
	PARAM_LIST                                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_REQUIRED_PARAM}
	PARAMETER_START                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	PARAMETER_START_WITHOUT_ANNOTATION                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM, common.PARSER_RULE_CONTEXT_ASTERISK}
	REQUIRED_PARAM_NAME_RHS                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PARAM_END, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	PARAM_END                                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	STMT_START_WITH_EXPR_RHS                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR}
	EXPR_STMT_RHS                                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR}
	EXPRESSION_STATEMENT_START                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD, common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_START_KEYWORD, common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD}
	ANNOT_DECL_OPTIONAL_TYPE                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_TAG, common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER}
	CONST_DECL_TYPE                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER, common.PARSER_RULE_CONTEXT_VARIABLE_NAME}
	ANNOT_DECL_RHS                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_TAG, common.PARSER_RULE_CONTEXT_ON_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	ANNOT_OPTIONAL_ATTACH_POINTS                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ON_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	ATTACH_POINT                                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD, common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT}
	ATTACH_POINT_IDENT                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT, common.PARSER_RULE_CONTEXT_OBJECT_IDENT, common.PARSER_RULE_CONTEXT_SERVICE_IDENT, common.PARSER_RULE_CONTEXT_RECORD_IDENT}
	SERVICE_IDENT_RHS                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_REMOTE_IDENT, common.PARSER_RULE_CONTEXT_ATTACH_POINT_END}
	ATTACH_POINT_END                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_SEMICOLON}
	XML_NAMESPACE_PREFIX_DECL                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	CONSTANT_EXPRESSION                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BASIC_LITERAL, common.PARSER_RULE_CONTEXT_VARIABLE_REF, common.PARSER_RULE_CONTEXT_PLUS_TOKEN, common.PARSER_RULE_CONTEXT_MINUS_TOKEN, common.PARSER_RULE_CONTEXT_NIL_LITERAL}
	LIST_CONSTRUCTOR_FIRST_MEMBER                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER}
	LIST_CONSTRUCTOR_MEMBER                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	TYPE_CAST_PARAM                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	TYPE_CAST_PARAM_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS, common.PARSER_RULE_CONTEXT_GT}
	TABLE_KEYWORD_RHS                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_KEY_SPECIFIER, common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR}
	ROW_LIST_RHS                                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR}
	TABLE_ROW_END                                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	KEY_SPECIFIER_RHS                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_VARIABLE_NAME}
	TABLE_KEY_RHS                                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	LET_VAR_DECL_START                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	STREAM_TYPE_FIRST_PARAM_RHS                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_GT}
	TEMPLATE_MEMBER                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TEMPLATE_STRING, common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN, common.PARSER_RULE_CONTEXT_TEMPLATE_END}
	TEMPLATE_STRING_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN, common.PARSER_RULE_CONTEXT_TEMPLATE_END}
	KEY_CONSTRAINTS_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_LT}
	FUNCTION_KEYWORD_RHS                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_NAME, common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS}
	FUNC_TYPE_FUNC_KEYWORD_RHS_START                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END, common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS}
	TYPE_DESC_RHS                                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_END_OF_TYPE_DESC, common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_PIPE, common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR}
	TABLE_TYPE_DESC_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_KEY_KEYWORD, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS}
	NEW_KEYWORD_RHS                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS}
	TABLE_CONSTRUCTOR_OR_QUERY_START                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TABLE_KEYWORD, common.PARSER_RULE_CONTEXT_STREAM_KEYWORD, common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION, common.PARSER_RULE_CONTEXT_MAP_KEYWORD}
	TABLE_CONSTRUCTOR_OR_QUERY_RHS                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION}
	QUERY_PIPELINE_RHS                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_RHS, common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE, common.PARSER_RULE_CONTEXT_QUERY_ACTION_RHS}
	INTERMEDIATE_CLAUSE_START                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_WHERE_CLAUSE, common.PARSER_RULE_CONTEXT_FROM_CLAUSE, common.PARSER_RULE_CONTEXT_LET_CLAUSE, common.PARSER_RULE_CONTEXT_JOIN_CLAUSE, common.PARSER_RULE_CONTEXT_ORDER_BY_CLAUSE, common.PARSER_RULE_CONTEXT_LIMIT_CLAUSE, common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE}
	RESULT_CLAUSE                                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SELECT_CLAUSE, common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE}
	BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_COMMA}
	ANNOTATION_REF_RHS                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_END, common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR}
	INFER_PARAM_END_OR_PARENTHESIS_END                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START}
	OPTIONAL_PEER_WORKER                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS}
	TYPE_DESC_IN_TUPLE_RHS                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	TUPLE_TYPE_MEMBER_RHS                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA}
	LIST_CONSTRUCTOR_MEMBER_END                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA}
	NIL_OR_PARENTHESISED_TYPE_DESC_RHS                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR}
	BINDING_PATTERN                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER, common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN}
	LIST_BINDING_PATTERNS_START                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	LIST_BINDING_PATTERN_CONTENTS                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN}
	LIST_BINDING_PATTERN_MEMBER_END                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA}
	MAPPING_BINDING_PATTERN_MEMBER                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN}
	MAPPING_BINDING_PATTERN_END                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	FIELD_BINDING_PATTERN_END                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	ERROR_ARG_LIST_BINDING_PATTERN_START                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	ERROR_MESSAGE_BINDING_PATTERN_END                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	ERROR_MESSAGE_BINDING_PATTERN_RHS                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN}
	ERROR_FIELD_BINDING_PATTERN                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN}
	ERROR_FIELD_BINDING_PATTERN_END                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_COMMA}
	REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND, common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN, common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME, common.PARSER_RULE_CONTEXT_METHOD_NAME}
	REMOTE_CALL_OR_ASYNC_SEND_END                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_SEMICOLON}
	RECEIVE_WORKERS                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER, common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS}
	SINGLE_OR_ALTERNATE_WORKER_SEPARATOR                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_END, common.PARSER_RULE_CONTEXT_PIPE}
	RECEIVE_FIELD                                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME, common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME}
	RECEIVE_FIELD_END                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA}
	WAIT_KEYWORD_RHS                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS, common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS}
	WAIT_FIELD_NAME_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_WAIT_FIELD_END}
	WAIT_FIELD_END                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA}
	WAIT_FUTURE_EXPR_END                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPR_LIST_END, common.PARSER_RULE_CONTEXT_PIPE}
	ENUM_MEMBER_START                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	ENUM_MEMBER_RHS                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END}
	ENUM_MEMBER_END                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	MEMBER_ACCESS_KEY_EXPR_END                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	ROLLBACK_RHS                                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_EXPRESSION}
	RETRY_KEYWORD_RHS                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LT, common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS}
	RETRY_TYPE_PARAM_RHS                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_RETRY_BODY}
	RETRY_BODY                                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BLOCK_STMT, common.PARSER_RULE_CONTEXT_TRANSACTION_STMT}
	LIST_BP_OR_TUPLE_TYPE_MEMBER                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER}
	LIST_BP_OR_TUPLE_TYPE_DESC_RHS                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_VARIABLE_NAME}
	BRACKETED_LIST_MEMBER_END                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	BRACKETED_LIST_MEMBER                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_BINDING_PATTERN}
	LIST_BINDING_MEMBER_OR_ARRAY_LENGTH                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ARRAY_LENGTH_START}
	BRACKETED_LIST_RHS                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS}
	BINDING_PATTERN_OR_VAR_REF_RHS                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS, common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS}
	TYPE_DESC_RHS_OR_BP_RHS                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_IN_TYPED_BP, common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_RHS}
	XML_NAVIGATE_EXPR                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_XML_FILTER_EXPR, common.PARSER_RULE_CONTEXT_XML_STEP_EXPR}
	XML_NAME_PATTERN_RHS                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_GT, common.PARSER_RULE_CONTEXT_PIPE}
	XML_ATOMIC_NAME_PATTERN_START                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER}
	XML_ATOMIC_NAME_IDENTIFIER_RHS                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_IDENTIFIER}
	XML_STEP_START                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN, common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN, common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN}
	XML_STEP_EXTEND                                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND_END, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	XML_STEP_START_END                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION_RHS, common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS}
	MATCH_PATTERN_LIST_MEMBER_RHS                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MATCH_PATTERN_END, common.PARSER_RULE_CONTEXT_PIPE}
	OPTIONAL_MATCH_GUARD                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW, common.PARSER_RULE_CONTEXT_IF_KEYWORD}
	MATCH_PATTERN_START                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION, common.PARSER_RULE_CONTEXT_VAR_KEYWORD, common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN}
	LIST_MATCH_PATTERNS_START                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	LIST_MATCH_PATTERN_MEMBER                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	LIST_MATCH_PATTERN_MEMBER_RHS                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	FIELD_MATCH_PATTERNS_START                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	FIELD_MATCH_PATTERN_MEMBER                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	FIELD_MATCH_PATTERN_MEMBER_RHS                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	ERROR_MATCH_PATTERN_OR_CONST_PATTERN                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_MATCH_PATTERN_RHS}
	ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	ERROR_ARG_LIST_MATCH_PATTERN_START                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION, common.PARSER_RULE_CONTEXT_VAR_KEYWORD, common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	ERROR_MESSAGE_MATCH_PATTERN_END                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	ERROR_MESSAGE_MATCH_PATTERN_RHS                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_CAUSE_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN}
	ERROR_FIELD_MATCH_PATTERN                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	ERROR_FIELD_MATCH_PATTERN_RHS                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	NAMED_ARG_MATCH_PATTERN_RHS                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	ORDER_KEY_LIST_END                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ORDER_CLAUSE_END, common.PARSER_RULE_CONTEXT_COMMA}
	LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER}
	TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER}
	JOIN_CLAUSE_START                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_JOIN_KEYWORD, common.PARSER_RULE_CONTEXT_OUTER_KEYWORD}
	MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_MAPPING_FIELD}
	LISTENERS_LIST_END                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_BLOCK, common.PARSER_RULE_CONTEXT_COMMA}
	FUNC_TYPE_DESC_START                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER}
	FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER}
	MODULE_CLASS_DEFINITION_START                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER}
	CLASS_DEF_WITHOUT_FIRST_QUALIFIER                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER}
	CLASS_DEF_WITHOUT_SECOND_QUALIFIER                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER}
	CLASS_DEF_WITHOUT_THIRD_QUALIFIER                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER}
	REGULAR_COMPOUND_STMT_RHS                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE}
	NAMED_WORKER_DECL_START                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_WORKER_KEYWORD, common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD}
	SERVICE_DECL_START                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD, common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER}
	OPTIONAL_SERVICE_DECL_TYPE                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE, common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH}
	OPTIONAL_ABSOLUTE_PATH                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH, common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN, common.PARSER_RULE_CONTEXT_ON_KEYWORD}
	ABSOLUTE_RESOURCE_PATH_START                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH}
	ABSOLUTE_RESOURCE_PATH_END                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_SERVICE_DECL_RHS}
	SERVICE_DECL_OR_VAR_DECL                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH, common.PARSER_RULE_CONTEXT_SERVICE_VAR_DECL_RHS}
	OPTIONAL_RELATIVE_PATH                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH}
	FUNC_DEF_OR_TYPE_DESC_RHS                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH, common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	RELATIVE_RESOURCE_PATH_START                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT}
	RESOURCE_PATH_SEGMENT                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT, common.PARSER_RULE_CONTEXT_RESOURCE_PATH_PARAM}
	PATH_PARAM_OPTIONAL_ANNOTS                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	PATH_PARAM_ELLIPSIS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	OPTIONAL_PATH_PARAM_NAME                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	RELATIVE_RESOURCE_PATH_END                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RESOURCE_ACCESSOR_DEF_OR_DECL_RHS, common.PARSER_RULE_CONTEXT_SLASH}
	CONFIG_VAR_DECL_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_QUESTION_MARK}
	ERROR_CONSTRUCTOR_RHS                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	OPTIONAL_TYPE_PARAMETER                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LT, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS}
	MAP_TYPE_OR_TYPE_REF                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_LT}
	OBJECT_TYPE_OR_TYPE_REF                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OBJECT_KEYWORD_RHS}
	STREAM_TYPE_OR_TYPE_REF                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_LT}
	TABLE_TYPE_OR_TYPE_REF                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM}
	PARAMETERIZED_TYPE_OR_TYPE_REF                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER}
	TYPE_DESC_RHS_OR_TYPE_REF                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS}
	TRANSACTION_STMT_RHS_OR_TYPE_REF                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_REF_COLON, common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_TRANSACTION_KEYWORD_RHS}
	TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_REF_COLON, common.PARSER_RULE_CONTEXT_EXPRESSION_START_TABLE_KEYWORD_RHS}
	QUERY_EXPR_OR_VAR_REF                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_REF_COLON, common.PARSER_RULE_CONTEXT_QUERY_CONSTRUCT_TYPE_RHS}
	ERROR_CONS_EXPR_OR_VAR_REF                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_REF_COLON, common.PARSER_RULE_CONTEXT_ERROR_CONS_ERROR_KEYWORD_RHS}
	QUALIFIED_IDENTIFIER                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER, common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX}
	MODULE_VAR_DECL_START                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL}
	MODULE_VAR_WITHOUT_FIRST_QUAL                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL}
	MODULE_VAR_WITHOUT_SECOND_QUAL                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL}
	EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT}
	TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START, common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT}
	END_OF_PARAMS_OR_NEXT_PARAM_START                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_COMMA}
	PARAM_START                                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	PARAM_RHS                                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_REST_PARAM_RHS}
	FUNC_TYPE_PARAM_RHS                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PARAM_END, common.PARSER_RULE_CONTEXT_PARAM_RHS}
	ANNOTATION_DECL_START                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD, common.PARSER_RULE_CONTEXT_CONST_KEYWORD}
	OPTIONAL_RESOURCE_ACCESS_PATH                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD}
	RESOURCE_ACCESS_PATH_SEGMENT                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_IDENTIFIER, common.PARSER_RULE_CONTEXT_OPEN_BRACKET}
	COMPUTED_SEGMENT_OR_REST_SEGMENT                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	RESOURCE_ACCESS_SEGMENT_RHS                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD}
	OPTIONAL_RESOURCE_ACCESS_METHOD                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST}
	OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_ACTION_END}
	OPTIONAL_TOP_LEVEL_SEMICOLON                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE, common.PARSER_RULE_CONTEXT_SEMICOLON}
	TUPLE_MEMBER                                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATIONS, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE}
	NATURAL_EXPRESSION_START                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD, common.PARSER_RULE_CONTEXT_CONST_KEYWORD}
	OPTIONAL_PARENTHESIZED_ARG_LIST                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_OPEN_BRACE}
)

func NewBallerinaParserErrorHandlerFromTokenReader(tokenReader *TokenReader) BallerinaParserErrorHandler {
	this := BallerinaParserErrorHandler{}
	this.AbstractParserErrorHandlerBase = *NewAbstractParserErrorHandlerBase(tokenReader)
	this.Self = &this
	return this
}

func (b *BallerinaParserErrorHandler) isEndOfObjectTypeNode(nextLookahead int) bool {
	nextToken := b.tokenReader.PeekN(nextLookahead)
	switch nextToken.Kind() {
	case common.CLOSE_BRACE_TOKEN,
		common.EOF_TOKEN,
		common.CLOSE_BRACE_PIPE_TOKEN,
		common.TYPE_KEYWORD,
		common.SERVICE_KEYWORD:
		return true
	default:
		nextNextToken := b.tokenReader.PeekN(nextLookahead + 1)
		switch nextNextToken.Kind() {
		case common.CLOSE_BRACE_TOKEN,
			common.EOF_TOKEN,
			common.CLOSE_BRACE_PIPE_TOKEN,
			common.TYPE_KEYWORD,
			common.SERVICE_KEYWORD:
			return true
		default:
			return false
		}
	}
}

func (b *BallerinaParserErrorHandler) SeekMatch(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) (result *Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(SeekMatch start %s %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(SeekMatch end (%s %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatBool(isEntryPoint), formatResult(result))
	})
	var hasMatch bool
	var skipRule bool
	matchingRulesCount := 0
	for currentDepth < LOOKAHEAD_LIMIT {
		skipRule = false
		lookahead = b.getNextLookahead(lookahead)
		nextToken := b.tokenReader.PeekN(lookahead)
		switch currentCtx {
		case common.PARSER_RULE_CONTEXT_EOF:
			hasMatch = (nextToken.Kind() == common.EOF_TOKEN)
		case common.PARSER_RULE_CONTEXT_FUNC_NAME,
			common.PARSER_RULE_CONTEXT_CLASS_NAME,
			common.PARSER_RULE_CONTEXT_VARIABLE_NAME,
			common.PARSER_RULE_CONTEXT_TYPE_NAME,
			common.PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME,
			common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME,
			common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME,
			common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER,
			common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER,
			common.PARSER_RULE_CONTEXT_IDENTIFIER,
			common.PARSER_RULE_CONTEXT_ANNOTATION_TAG,
			common.PARSER_RULE_CONTEXT_NAMESPACE_PREFIX,
			common.PARSER_RULE_CONTEXT_WORKER_NAME,
			common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM,
			common.PARSER_RULE_CONTEXT_METHOD_NAME,
			common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME,
			common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME,
			common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME,
			common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER,
			common.PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN,
			common.PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN,
			common.PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT,
			common.PARSER_RULE_CONTEXT_MODULE_ENUM_NAME,
			common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME,
			common.PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN:
			hasMatch = (nextToken.Kind() == common.IDENTIFIER_TOKEN)
		case common.PARSER_RULE_CONTEXT_IMPORT_PREFIX:
			hasMatch = ((nextToken.Kind() == common.IDENTIFIER_TOKEN) || isPredeclaredPrefix(nextToken.Kind()))
		case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX:
			hasMatch = isPredeclaredPrefix(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS,
			common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START,
			common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN:
			hasMatch = (nextToken.Kind() == common.OPEN_PAREN_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS,
			common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN:
			hasMatch = (nextToken.Kind() == common.CLOSE_PAREN_TOKEN)
		case common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR:
			hasMatch = (((isSimpleType(nextToken.Kind()) || (nextToken.Kind() == common.ERROR_KEYWORD)) || (nextToken.Kind() == common.STREAM_KEYWORD)) || (nextToken.Kind() == common.TYPEDESC_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OPEN_BRACE:
			hasMatch = (nextToken.Kind() == common.OPEN_BRACE_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSE_BRACE:
			hasMatch = (nextToken.Kind() == common.CLOSE_BRACE_TOKEN)
		case common.PARSER_RULE_CONTEXT_ASSIGN_OP:
			hasMatch = (nextToken.Kind() == common.EQUAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_SEMICOLON:
			hasMatch = (nextToken.Kind() == common.SEMICOLON_TOKEN)
		case common.PARSER_RULE_CONTEXT_BINARY_OPERATOR:
			hasMatch = b.isBinaryOperator(nextToken)
		case common.PARSER_RULE_CONTEXT_COMMA,
			common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA,
			common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA:
			hasMatch = (nextToken.Kind() == common.COMMA_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END:
			hasMatch = (nextToken.Kind() == common.CLOSE_BRACE_PIPE_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START:
			hasMatch = (nextToken.Kind() == common.OPEN_BRACE_PIPE_TOKEN)
		case common.PARSER_RULE_CONTEXT_ELLIPSIS:
			hasMatch = (nextToken.Kind() == common.ELLIPSIS_TOKEN)
		case common.PARSER_RULE_CONTEXT_QUESTION_MARK:
			hasMatch = (nextToken.Kind() == common.QUESTION_MARK_TOKEN)
		case common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER,
			common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER:
			hasMatch = (((nextToken.Kind() == common.CLIENT_KEYWORD) || (nextToken.Kind() == common.ISOLATED_KEYWORD)) || (nextToken.Kind() == common.SERVICE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER:
			hasMatch = (((((nextToken.Kind() == common.DISTINCT_KEYWORD) || (nextToken.Kind() == common.CLIENT_KEYWORD)) || (nextToken.Kind() == common.READONLY_KEYWORD)) || (nextToken.Kind() == common.ISOLATED_KEYWORD)) || (nextToken.Kind() == common.SERVICE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OPEN_BRACKET,
			common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START:
			hasMatch = (nextToken.Kind() == common.OPEN_BRACKET_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSE_BRACKET:
			hasMatch = (nextToken.Kind() == common.CLOSE_BRACKET_TOKEN)
		case common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_METHOD_CALL_DOT:
			hasMatch = (nextToken.Kind() == common.DOT_TOKEN)
		case common.PARSER_RULE_CONTEXT_BOOLEAN_LITERAL:
			hasMatch = ((nextToken.Kind() == common.TRUE_KEYWORD) || (nextToken.Kind() == common.FALSE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == common.DECIMAL_INTEGER_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_SLASH,
			common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH,
			common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN:
			hasMatch = (nextToken.Kind() == common.SLASH_TOKEN)
		case common.PARSER_RULE_CONTEXT_BASIC_LITERAL:
			hasMatch = b.isBasicLiteral(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_COLON,
			common.PARSER_RULE_CONTEXT_VAR_REF_COLON,
			common.PARSER_RULE_CONTEXT_TYPE_REF_COLON:
			hasMatch = (nextToken.Kind() == common.COLON_TOKEN)
		case common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == common.STRING_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_UNARY_OPERATOR:
			hasMatch = b.isUnaryOperator(nextToken)
		case common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == common.HEX_INTEGER_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_AT:
			hasMatch = (nextToken.Kind() == common.AT_TOKEN)
		case common.PARSER_RULE_CONTEXT_RIGHT_ARROW:
			hasMatch = (nextToken.Kind() == common.RIGHT_ARROW_TOKEN)
		case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE:
			hasMatch = isParameterizedTypeToken(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_LT,
			common.PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN,
			common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT:
			hasMatch = (nextToken.Kind() == common.LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_GT,
			common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT:
			hasMatch = (nextToken.Kind() == common.GT_TOKEN)
		case common.PARSER_RULE_CONTEXT_FIELD_IDENT:
			hasMatch = (nextToken.Kind() == common.FIELD_KEYWORD)
		case common.PARSER_RULE_CONTEXT_FUNCTION_IDENT:
			hasMatch = (nextToken.Kind() == common.FUNCTION_KEYWORD)
		case common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT:
			hasMatch = ((nextToken.Kind() == common.FUNCTION_KEYWORD) || (nextToken.Kind() == common.FIELD_KEYWORD))
		case common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT:
			hasMatch = b.isSingleKeywordAttachPointIdent(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_OBJECT_IDENT:
			hasMatch = (nextToken.Kind() == common.OBJECT_KEYWORD)
		case common.PARSER_RULE_CONTEXT_RECORD_IDENT:
			hasMatch = (nextToken.Kind() == common.RECORD_KEYWORD)
		case common.PARSER_RULE_CONTEXT_SERVICE_IDENT:
			hasMatch = (nextToken.Kind() == common.SERVICE_KEYWORD)
		case common.PARSER_RULE_CONTEXT_REMOTE_IDENT:
			hasMatch = (nextToken.Kind() == common.REMOTE_KEYWORD)
		case common.PARSER_RULE_CONTEXT_DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_HEX_FLOATING_POINT_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == common.HEX_FLOATING_POINT_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_PIPE:
			hasMatch = (nextToken.Kind() == common.PIPE_TOKEN)
		case common.PARSER_RULE_CONTEXT_TEMPLATE_START, common.PARSER_RULE_CONTEXT_TEMPLATE_END:
			hasMatch = (nextToken.Kind() == common.BACKTICK_TOKEN)
		case common.PARSER_RULE_CONTEXT_ASTERISK:
			hasMatch = (nextToken.Kind() == common.ASTERISK_TOKEN)
		case common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR:
			hasMatch = (nextToken.Kind() == common.BITWISE_AND_TOKEN)
		case common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START,
			common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW:
			hasMatch = (nextToken.Kind() == common.RIGHT_DOUBLE_ARROW_TOKEN)
		case common.PARSER_RULE_CONTEXT_PLUS_TOKEN:
			hasMatch = (nextToken.Kind() == common.PLUS_TOKEN)
		case common.PARSER_RULE_CONTEXT_MINUS_TOKEN:
			hasMatch = (nextToken.Kind() == common.MINUS_TOKEN)
		case common.PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS:
			hasMatch = isIntOrFloat(nextToken)
		case common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN:
			hasMatch = (nextToken.Kind() == common.SYNC_SEND_TOKEN)
		case common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME:
			hasMatch = ((nextToken.Kind() == common.FUNCTION_KEYWORD) || (nextToken.Kind() == common.IDENTIFIER_TOKEN))
		case common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN:
			hasMatch = (nextToken.Kind() == common.LEFT_ARROW_TOKEN)
		case common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN:
			hasMatch = (nextToken.Kind() == common.ANNOT_CHAINING_TOKEN)
		case common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN:
			hasMatch = (nextToken.Kind() == common.OPTIONAL_CHAINING_TOKEN)
		case common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD:
			hasMatch = (nextToken.Kind() == common.TRANSACTIONAL_KEYWORD)
		case common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER:
			hasMatch = (nextToken.Kind() == common.ISOLATED_KEYWORD)
		case common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN:
			hasMatch = ((nextToken.Kind() == common.PIPE_TOKEN) || (nextToken.Kind() == common.BITWISE_AND_TOKEN))
		case common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN:
			hasMatch = (nextToken.Kind() == common.DOT_LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN:
			hasMatch = (nextToken.Kind() == common.SLASH_LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
			hasMatch = (nextToken.Kind() == common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN:
			hasMatch = (nextToken.Kind() == common.SLASH_ASTERISK_TOKEN)
		case common.PARSER_RULE_CONTEXT_KEY_KEYWORD:
			hasMatch = ((nextToken.Kind() == common.KEY_KEYWORD) || isKeyKeyword(nextToken))
		case common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
			hasMatch = ((nextToken.Kind() == common.NATURAL_KEYWORD) || isNaturalKeyword(nextToken))
		case common.PARSER_RULE_CONTEXT_VAR_KEYWORD:
			hasMatch = (nextToken.Kind() == common.VAR_KEYWORD)
		case common.PARSER_RULE_CONTEXT_ORDER_DIRECTION:
			hasMatch = ((nextToken.Kind() == common.ASCENDING_KEYWORD) || (nextToken.Kind() == common.DESCENDING_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL:
			hasMatch = ((nextToken.Kind() == common.PRIVATE_KEYWORD) || (nextToken.Kind() == common.PUBLIC_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER,
			common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER,
			common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER,
			common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER:
			hasMatch = ((((nextToken.Kind() == common.ISOLATED_KEYWORD) || (nextToken.Kind() == common.TRANSACTIONAL_KEYWORD)) || (nextToken.Kind() == common.REMOTE_KEYWORD)) || (nextToken.Kind() == common.RESOURCE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER:
			hasMatch = ((nextToken.Kind() == common.ISOLATED_KEYWORD) || (nextToken.Kind() == common.TRANSACTIONAL_KEYWORD))
		case common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL,
			common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL,
			common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL:
			hasMatch = (((nextToken.Kind() == common.FINAL_KEYWORD) || (nextToken.Kind() == common.ISOLATED_KEYWORD)) || (nextToken.Kind() == common.CONFIGURABLE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR:
			hasMatch = isCompoundBinaryOperator(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_IS_KEYWORD:
			hasMatch = ((nextToken.Kind() == common.IS_KEYWORD) || (nextToken.Kind() == common.NOT_IS_KEYWORD))
		case common.PARSER_RULE_CONTEXT_VARIABLE_REF,
			common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION,
			common.PARSER_RULE_CONTEXT_TYPE_REFERENCE,
			common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE,
			common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM,
			common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY:
			fallthrough
		default:
			if b.isKeyword(currentCtx) {
				expectedTokenKind := b.getExpectedKeywordKind(currentCtx)
				hasMatch = ((nextToken.Kind() == expectedTokenKind) || isKeywordMatch(expectedTokenKind, nextToken))
				break
			}
			if b.HasAlternativePaths(currentCtx) {
				result := b.seekMatchInAlternativePaths(currentCtx, lookahead, currentDepth, matchingRulesCount,
					isEntryPoint)
				return &result
			}
			skipRule = true
			hasMatch = true
		}
		if !hasMatch {
			return b.fixAndContinue(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
		}
		if !skipRule {
			currentDepth++
			matchingRulesCount++
			lookahead++
			isEntryPoint = false
		}
		currentCtx = b.GetNextRule(currentCtx, lookahead)
	}
	result = NewResult(make([]*Solution, 0), matchingRulesCount)
	result.solution = NewSolution(ACTION_KEEP, currentCtx, common.NONE, currentCtx.String())
	return result
}

func (b *BallerinaParserErrorHandler) getNextLookahead(lookahead int) int {
	for b.tokenReader.PeekN(lookahead).Kind() == common.DOCUMENTATION_STRING {
		lookahead++
	}
	return lookahead
}

func (b *BallerinaParserErrorHandler) isKeyword(currentCtx common.ParserRuleContext) bool {
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_EOF,
		common.PARSER_RULE_CONTEXT_PUBLIC_KEYWORD,
		common.PARSER_RULE_CONTEXT_PRIVATE_KEYWORD,
		common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD,
		common.PARSER_RULE_CONTEXT_NEW_KEYWORD,
		common.PARSER_RULE_CONTEXT_SELECT_KEYWORD,
		common.PARSER_RULE_CONTEXT_WHERE_KEYWORD,
		common.PARSER_RULE_CONTEXT_FROM_KEYWORD,
		common.PARSER_RULE_CONTEXT_ORDER_KEYWORD,
		common.PARSER_RULE_CONTEXT_GROUP_KEYWORD,
		common.PARSER_RULE_CONTEXT_BY_KEYWORD,
		common.PARSER_RULE_CONTEXT_START_KEYWORD,
		common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD,
		common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND,
		common.PARSER_RULE_CONTEXT_WAIT_KEYWORD,
		common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD,
		common.PARSER_RULE_CONTEXT_FAIL_KEYWORD,
		common.PARSER_RULE_CONTEXT_DO_KEYWORD,
		common.PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD,
		common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD,
		common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD,
		common.PARSER_RULE_CONTEXT_RETRY_KEYWORD,
		common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD,
		common.PARSER_RULE_CONTEXT_ENUM_KEYWORD,
		common.PARSER_RULE_CONTEXT_MATCH_KEYWORD,
		common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD,
		common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD,
		common.PARSER_RULE_CONTEXT_RECORD_KEYWORD,
		common.PARSER_RULE_CONTEXT_TYPE_KEYWORD,
		common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD,
		common.PARSER_RULE_CONTEXT_ABSTRACT_KEYWORD,
		common.PARSER_RULE_CONTEXT_CLIENT_KEYWORD,
		common.PARSER_RULE_CONTEXT_IF_KEYWORD,
		common.PARSER_RULE_CONTEXT_ELSE_KEYWORD,
		common.PARSER_RULE_CONTEXT_WHILE_KEYWORD,
		common.PARSER_RULE_CONTEXT_PANIC_KEYWORD,
		common.PARSER_RULE_CONTEXT_AS_KEYWORD,
		common.PARSER_RULE_CONTEXT_LOCK_KEYWORD,
		common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD,
		common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD,
		common.PARSER_RULE_CONTEXT_BREAK_KEYWORD,
		common.PARSER_RULE_CONTEXT_RETURN_KEYWORD,
		common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD,
		common.PARSER_RULE_CONTEXT_ON_KEYWORD,
		common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD,
		common.PARSER_RULE_CONTEXT_CONST_KEYWORD,
		common.PARSER_RULE_CONTEXT_FINAL_KEYWORD,
		common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD,
		common.PARSER_RULE_CONTEXT_IS_KEYWORD,
		common.PARSER_RULE_CONTEXT_NOT_IS_KEYWORD,
		common.PARSER_RULE_CONTEXT_NULL_KEYWORD,
		common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD,
		common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD,
		common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD,
		common.PARSER_RULE_CONTEXT_WORKER_KEYWORD,
		common.PARSER_RULE_CONTEXT_FORK_KEYWORD,
		common.PARSER_RULE_CONTEXT_TRAP_KEYWORD,
		common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD,
		common.PARSER_RULE_CONTEXT_IN_KEYWORD,
		common.PARSER_RULE_CONTEXT_TABLE_KEYWORD,
		common.PARSER_RULE_CONTEXT_KEY_KEYWORD,
		common.PARSER_RULE_CONTEXT_ERROR_KEYWORD,
		common.PARSER_RULE_CONTEXT_LET_KEYWORD,
		common.PARSER_RULE_CONTEXT_STREAM_KEYWORD,
		common.PARSER_RULE_CONTEXT_XML_KEYWORD,
		common.PARSER_RULE_CONTEXT_RE_KEYWORD,
		common.PARSER_RULE_CONTEXT_STRING_KEYWORD,
		common.PARSER_RULE_CONTEXT_BASE16_KEYWORD,
		common.PARSER_RULE_CONTEXT_BASE64_KEYWORD,
		common.PARSER_RULE_CONTEXT_DISTINCT_KEYWORD,
		common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD,
		common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD,
		common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD,
		common.PARSER_RULE_CONTEXT_JOIN_KEYWORD,
		common.PARSER_RULE_CONTEXT_OUTER_KEYWORD,
		common.PARSER_RULE_CONTEXT_CLASS_KEYWORD,
		common.PARSER_RULE_CONTEXT_MAP_KEYWORD,
		common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD,
		common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) HasAlternativePaths(currentCtx common.ParserRuleContext) bool {
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE,
		common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER,
		common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA,
		common.PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS,
		common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY,
		common.PARSER_RULE_CONTEXT_FUNC_BODY,
		common.PARSER_RULE_CONTEXT_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS,
		common.PARSER_RULE_CONTEXT_EXPRESSION_RHS,
		common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS,
		common.PARSER_RULE_CONTEXT_STATEMENT,
		common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS,
		common.PARSER_RULE_CONTEXT_PARAM_LIST,
		common.PARSER_RULE_CONTEXT_REQUIRED_PARAM_NAME_RHS,
		common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME,
		common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS,
		common.PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS,
		common.PARSER_RULE_CONTEXT_RECORD_BODY_END,
		common.PARSER_RULE_CONTEXT_RECORD_BODY_START,
		common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED,
		common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END,
		common.PARSER_RULE_CONTEXT_RECORD_FIELD_START,
		common.PARSER_RULE_CONTEXT_RECORD_FIELD_WITHOUT_METADATA,
		common.PARSER_RULE_CONTEXT_ARG_START,
		common.PARSER_RULE_CONTEXT_ARG_START_OR_ARG_LIST_END,
		common.PARSER_RULE_CONTEXT_NAMED_OR_POSITIONAL_ARG_RHS,
		common.PARSER_RULE_CONTEXT_ARG_END,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META,
		common.PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META,
		common.PARSER_RULE_CONTEXT_OPTIONAL_FIELD_INITIALIZER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_START,
		common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD,
		common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_START,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_START,
		common.PARSER_RULE_CONTEXT_ELSE_BLOCK,
		common.PARSER_RULE_CONTEXT_ELSE_BODY,
		common.PARSER_RULE_CONTEXT_CALL_STMT_START,
		common.PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL,
		common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS,
		common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME,
		common.PARSER_RULE_CONTEXT_RETURN_STMT_RHS,
		common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION,
		common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD,
		common.PARSER_RULE_CONTEXT_MAPPING_FIELD,
		common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD,
		common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS,
		common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END,
		common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH,
		common.PARSER_RULE_CONTEXT_CONST_DECL_TYPE,
		common.PARSER_RULE_CONTEXT_CONST_DECL_RHS,
		common.PARSER_RULE_CONTEXT_ARRAY_LENGTH,
		common.PARSER_RULE_CONTEXT_PARAMETER_START,
		common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION,
		common.PARSER_RULE_CONTEXT_STMT_START_WITH_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_EXPR_STMT_RHS,
		common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START,
		common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE,
		common.PARSER_RULE_CONTEXT_ANNOT_DECL_RHS,
		common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS,
		common.PARSER_RULE_CONTEXT_ATTACH_POINT,
		common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT,
		common.PARSER_RULE_CONTEXT_ATTACH_POINT_END,
		common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL,
		common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS,
		common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER,
		common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_ROW_LIST_RHS,
		common.PARSER_RULE_CONTEXT_TABLE_ROW_END,
		common.PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS,
		common.PARSER_RULE_CONTEXT_TABLE_KEY_RHS,
		common.PARSER_RULE_CONTEXT_LET_VAR_DECL_START,
		common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END,
		common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_TEMPLATE_MEMBER,
		common.PARSER_RULE_CONTEXT_TEMPLATE_STRING_RHS,
		common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START,
		common.PARSER_RULE_CONTEXT_WORKER_NAME_RHS,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERNS_START,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END,
		common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER,
		common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END,
		common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER,
		common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS,
		common.PARSER_RULE_CONTEXT_TABLE_TYPE_DESC_RHS,
		common.PARSER_RULE_CONTEXT_NEW_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START,
		common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS,
		common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS,
		common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_PARAM_END,
		common.PARSER_RULE_CONTEXT_ANNOTATION_REF_RHS,
		common.PARSER_RULE_CONTEXT_INFER_PARAM_END_OR_PARENTHESIS_END,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS,
		common.PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS,
		common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END,
		common.PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS,
		common.PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS,
		common.PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END,
		common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS,
		common.PARSER_RULE_CONTEXT_RECEIVE_FIELD,
		common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END,
		common.PARSER_RULE_CONTEXT_WAIT_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME_RHS,
		common.PARSER_RULE_CONTEXT_WAIT_FIELD_END,
		common.PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END,
		common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER,
		common.PARSER_RULE_CONTEXT_ENUM_MEMBER_START,
		common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS,
		common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END,
		common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END,
		common.PARSER_RULE_CONTEXT_ROLLBACK_RHS,
		common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_RETRY_BODY,
		common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER,
		common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_RHS,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_MEMBER_OR_ARRAY_LENGTH,
		common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR,
		common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS,
		common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START,
		common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER_RHS,
		common.PARSER_RULE_CONTEXT_XML_STEP_START,
		common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY,
		common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD,
		common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS,
		common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START,
		common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERNS_START,
		common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER,
		common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS,
		common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS,
		common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END,
		common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START,
		common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER,
		common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS,
		common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS,
		common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS,
		common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS,
		common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN_RHS,
		common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS,
		common.PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER,
		common.PARSER_RULE_CONTEXT_JOIN_CLAUSE_START,
		common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE_START,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END,
		common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS,
		common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL_START,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION_START,
		common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION_START,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_OBJECT_FIELD_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE,
		common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS,
		common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_START,
		common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END,
		common.PARSER_RULE_CONTEXT_SERVICE_DECL_OR_VAR_DECL,
		common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH,
		common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_START,
		common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END,
		common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT,
		common.PARSER_RULE_CONTEXT_PATH_PARAM_OPTIONAL_ANNOTS,
		common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS,
		common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME,
		common.PARSER_RULE_CONTEXT_OBJECT_CONS_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_CONFIG_VAR_DECL_RHS,
		common.PARSER_RULE_CONTEXT_SERVICE_DECL_START,
		common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS,
		common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER,
		common.PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF,
		common.PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF,
		common.PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF,
		common.PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF,
		common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_THIRD_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_START,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL_START,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_FIRST_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_SECOND_QUAL,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS,
		common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START,
		common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END,
		common.PARSER_RULE_CONTEXT_END_OF_PARAMS_OR_NEXT_PARAM_START,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT_RHS,
		common.PARSER_RULE_CONTEXT_PARAM_START,
		common.PARSER_RULE_CONTEXT_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_ANNOTATION_DECL_START,
		common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH,
		common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT,
		common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT,
		common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS,
		common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD,
		common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST,
		common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON,
		common.PARSER_RULE_CONTEXT_TUPLE_MEMBER,
		common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT,
		common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END,
		common.PARSER_RULE_CONTEXT_RESULT_CLAUSE,
		common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_SEPARATOR,
		common.PARSER_RULE_CONTEXT_XML_STEP_START_END,
		common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION_START,
		common.PARSER_RULE_CONTEXT_OPTIONAL_PARENTHESIZED_ARG_LIST:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) getShortestAlternative(currentCtx common.ParserRuleContext) common.ParserRuleContext {
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE,
		common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER,
		common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA:
		return common.PARSER_RULE_CONTEXT_EOF
	case common.PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS:
		return common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS:
		return common.PARSER_RULE_CONTEXT_FUNC_BODY
	case common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY:
		return common.PARSER_RULE_CONTEXT_EXPLICIT_ANON_FUNC_EXPR_BODY_START
	case common.PARSER_RULE_CONTEXT_FUNC_BODY:
		return common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK
	case common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF
	case common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_EXPRESSION_RHS, common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS:
		return common.PARSER_RULE_CONTEXT_BINARY_OPERATOR
	case common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT
	case common.PARSER_RULE_CONTEXT_PARAM_LIST:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM_NAME_RHS:
		return common.PARSER_RULE_CONTEXT_PARAM_END
	case common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_RECORD_BODY_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_RECORD_BODY_START, common.PARSER_RULE_CONTEXT_OPTIONAL_PARENTHESIZED_ARG_LIST:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END:
		return common.PARSER_RULE_CONTEXT_RECORD_BODY_END
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_START,
		common.PARSER_RULE_CONTEXT_RECORD_FIELD_WITHOUT_METADATA:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD
	case common.PARSER_RULE_CONTEXT_ARG_START:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_ARG_START_OR_ARG_LIST_END:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_END
	case common.PARSER_RULE_CONTEXT_NAMED_OR_POSITIONAL_ARG_RHS:
		return common.PARSER_RULE_CONTEXT_ARG_END
	case common.PARSER_RULE_CONTEXT_ARG_END:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_END
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META,
		common.PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_OPTIONAL_FIELD_INITIALIZER:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_START:
		return common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE
	case common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD:
		return common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY
	case common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY:
		return common.PARSER_RULE_CONTEXT_OBJECT_FIELD_START
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_START,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_START:
		return common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_ELSE_BLOCK:
		return common.PARSER_RULE_CONTEXT_STATEMENT
	case common.PARSER_RULE_CONTEXT_ELSE_BODY:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_CALL_STMT_START:
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF
	case common.PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS:
		return common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME
	case common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME,
		common.PARSER_RULE_CONTEXT_RETURN_STMT_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF
	case common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD:
		return common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD
	case common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD:
		return common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME
	case common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS:
		return common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH:
		return common.PARSER_RULE_CONTEXT_ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONST_DECL_TYPE:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_CONST_DECL_RHS:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_ARRAY_LENGTH:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_PARAMETER_START:
		return common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION
	case common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM
	case common.PARSER_RULE_CONTEXT_STMT_START_WITH_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_EXPR_STMT_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START:
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF
	case common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE:
		return common.PARSER_RULE_CONTEXT_ANNOTATION_TAG
	case common.PARSER_RULE_CONTEXT_ANNOT_DECL_RHS,
		common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT:
		return common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT:
		return common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT_END,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START:
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS:
		return common.PARSER_RULE_CONTEXT_END_OF_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM, common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_RHS:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS
	case common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR
	case common.PARSER_RULE_CONTEXT_ROW_LIST_RHS, common.PARSER_RULE_CONTEXT_TABLE_ROW_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS, common.PARSER_RULE_CONTEXT_TABLE_KEY_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_LET_VAR_DECL_START:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END:
		return common.PARSER_RULE_CONTEXT_ORDER_CLAUSE_END
	case common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END:
		return common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE_END
	case common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS:
		return common.PARSER_RULE_CONTEXT_GT
	case common.PARSER_RULE_CONTEXT_TEMPLATE_MEMBER, common.PARSER_RULE_CONTEXT_TEMPLATE_STRING_RHS:
		return common.PARSER_RULE_CONTEXT_TEMPLATE_END
	case common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END
	case common.PARSER_RULE_CONTEXT_WORKER_NAME_RHS:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERNS_START,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER:
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER:
		return common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_TABLE_TYPE_DESC_RHS:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_NEW_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START:
		return common.PARSER_RULE_CONTEXT_TABLE_KEYWORD
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS:
		return common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR
	case common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS:
		return common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_PARAM_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ANNOTATION_REF_RHS:
		return common.PARSER_RULE_CONTEXT_ANNOTATION_END
	case common.PARSER_RULE_CONTEXT_INFER_PARAM_END_OR_PARENTHESIS_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS:
		return common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME
	case common.PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS, common.PARSER_RULE_CONTEXT_RECEIVE_FIELD:
		return common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME
	case common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_WAIT_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME_RHS:
		return common.PARSER_RULE_CONTEXT_WAIT_FIELD_END
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END:
		return common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPR_LIST_END
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_START:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_ROLLBACK_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS:
		return common.PARSER_RULE_CONTEXT_RETRY_BODY
	case common.PARSER_RULE_CONTEXT_RETRY_BODY:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_RHS:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_IN_TYPED_BP
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_MEMBER_OR_ARRAY_LENGTH:
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR:
		return common.PARSER_RULE_CONTEXT_XML_FILTER_EXPR
	case common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS:
		return common.PARSER_RULE_CONTEXT_GT
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START:
		return common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER_RHS:
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_XML_STEP_START:
		return common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY:
		return common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY
	case common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD:
		return common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_END
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START:
		return common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERNS_START:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS:
		return common.PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS:
		return common.PARSER_RULE_CONTEXT_ERROR_CAUSE_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN_RHS:
		return common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS:
		return common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER:
		return common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_DEF:
		return common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE
	case common.PARSER_RULE_CONTEXT_JOIN_CLAUSE_START:
		return common.PARSER_RULE_CONTEXT_JOIN_KEYWORD
	case common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE_START:
		return common.PARSER_RULE_CONTEXT_WHERE_CLAUSE
	case common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER:
		return common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_BLOCK
	case common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS:
		return common.PARSER_RULE_CONTEXT_STATEMENT
	case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL_START:
		return common.PARSER_RULE_CONTEXT_WORKER_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_START,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION_START:
		return common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION_START:
		return common.PARSER_RULE_CONTEXT_CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH
	case common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS:
		return common.PARSER_RULE_CONTEXT_ATTACH_POINT_END
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_START:
		return common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END:
		return common.PARSER_RULE_CONTEXT_SERVICE_DECL_RHS
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_OR_VAR_DECL:
		return common.PARSER_RULE_CONTEXT_SERVICE_VAR_DECL_RHS
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_START:
		return common.PARSER_RULE_CONTEXT_DOT
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END:
		return common.PARSER_RULE_CONTEXT_RESOURCE_ACCESSOR_DEF_OR_DECL_RHS
	case common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT:
		return common.PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT
	case common.PARSER_RULE_CONTEXT_PATH_PARAM_OPTIONAL_ANNOTS:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM
	case common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS,
		common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_OBJECT_CONS_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONFIG_VAR_DECL_RHS:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_START:
		return common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD
	case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_LT
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OBJECT_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_LT
	case common.PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM
	case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF:
		return common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_TRANSACTION_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_START_TABLE_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF:
		return common.PARSER_RULE_CONTEXT_QUERY_CONSTRUCT_TYPE_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF:
		return common.PARSER_RULE_CONTEXT_ERROR_CONS_ERROR_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER:
		return common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_THIRD_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_WITHOUT_FIRST_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL:
		return common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL_START,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_FIRST_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_SECOND_QUAL:
		return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_TYPE_REFERENCE
	case common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END:
		return common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT
	case common.PARSER_RULE_CONTEXT_END_OF_PARAMS_OR_NEXT_PARAM_START:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT_RHS:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_PARAM_START:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM
	case common.PARSER_RULE_CONTEXT_PARAM_RHS:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_PARAM_RHS:
		return common.PARSER_RULE_CONTEXT_PARAM_END
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL_START:
		return common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH,
		common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD
	case common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT:
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST:
		return common.PARSER_RULE_CONTEXT_ACTION_END
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON:
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	case common.PARSER_RULE_CONTEXT_TUPLE_MEMBER:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE
	case common.PARSER_RULE_CONTEXT_RESULT_CLAUSE:
		return common.PARSER_RULE_CONTEXT_SELECT_CLAUSE
	case common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_SEPARATOR:
		return common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_END
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND:
		return common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND_END
	case common.PARSER_RULE_CONTEXT_XML_STEP_START_END:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION_START:
		return common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD
	default:
		panic("Alternative path entry not found")
	}
}

func (b *BallerinaParserErrorHandler) seekMatchInAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInAlternativePaths start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInAlternativePaths end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResultValue(result))
	})
	var alternativeRules []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE:
		alternativeRules = TOP_LEVEL_NODE
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER:
		alternativeRules = TOP_LEVEL_NODE_WITHOUT_MODIFIER
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA:
		alternativeRules = TOP_LEVEL_NODE_WITHOUT_METADATA
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_START:
		alternativeRules = FUNC_DEF_START
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = FUNC_DEF_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL:
		alternativeRules = FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL
	case common.PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS:
		parentCtx := b.GetParentContext()
		var alternatives []common.ParserRuleContext
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_FUNC_DEF:
			grandParentCtx := b.GetGrandParentContext()
			if grandParentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER {
				alternatives = METHOD_DECL_OPTIONAL_RETURNS
			} else {
				alternatives = FUNC_DEF_OPTIONAL_RETURNS
			}
		case common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION:
			alternatives = ANNON_FUNC_OPTIONAL_RETURNS
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC:
			alternatives = FUNC_TYPE_OPTIONAL_RETURNS
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC:
			alternatives = FUNC_TYPE_OR_ANON_FUNC_OPTIONAL_RETURNS
		default:
			alternatives = FUNC_TYPE_OR_DEF_OPTIONAL_RETURNS
		}
		alternativeRules = alternatives
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS:
		alternativeRules = FUNC_BODY_OR_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY:
		alternativeRules = FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY
	case common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY:
		alternativeRules = ANON_FUNC_BODY
	case common.PARSER_RULE_CONTEXT_FUNC_BODY:
		alternativeRules = FUNC_BODY
	case common.PARSER_RULE_CONTEXT_PARAM_LIST:
		alternativeRules = PARAM_LIST
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM_NAME_RHS:
		alternativeRules = REQUIRED_PARAM_NAME_RHS
	case common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS:
		alternativeRules = FIELD_DESCRIPTOR_RHS
	case common.PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS:
		alternativeRules = FIELD_OR_REST_DESCIPTOR_RHS
	case common.PARSER_RULE_CONTEXT_RECORD_BODY_END:
		alternativeRules = RECORD_BODY_END
	case common.PARSER_RULE_CONTEXT_RECORD_BODY_START:
		alternativeRules = RECORD_BODY_START
	case common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR:
		if !b.isInTypeDescContext() {
			panic("assertion failed")
		}
		alternativeRules = TYPE_DESCRIPTORS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED:
		alternativeRules = TYPE_DESCRIPTOR_WITHOUT_ISOLATED
	case common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR:
		alternativeRules = CLASS_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END:
		alternativeRules = RECORD_FIELD_OR_RECORD_END
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_START:
		alternativeRules = RECORD_FIELD_START
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_WITHOUT_METADATA:
		alternativeRules = RECORD_FIELD_WITHOUT_METADATA
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START:
		alternativeRules = CLASS_MEMBER_OR_OBJECT_MEMBER_START
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START:
		alternativeRules = OBJECT_CONSTRUCTOR_MEMBER_START
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META:
		alternativeRules = CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META
	case common.PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META:
		alternativeRules = OBJECT_CONS_MEMBER_WITHOUT_META
	case common.PARSER_RULE_CONTEXT_OPTIONAL_FIELD_INITIALIZER:
		alternativeRules = OPTIONAL_FIELD_INITIALIZER
	case common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN:
		alternativeRules = ON_FAIL_OPTIONAL_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_START:
		alternativeRules = OBJECT_METHOD_START
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER:
		alternativeRules = OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER:
		alternativeRules = OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD:
		alternativeRules = OBJECT_FUNC_OR_FIELD
	case common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY:
		alternativeRules = OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_START:
		alternativeRules = OBJECT_TYPE_START
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_START:
		alternativeRules = OBJECT_CONSTRUCTOR_START
	case common.PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL:
		alternativeRules = IMPORT_PREFIX_DECL
	case common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS:
		alternativeRules = IMPORT_DECL_ORG_OR_MODULE_NAME_RHS
	case common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME:
		alternativeRules = AFTER_IMPORT_MODULE_NAME
	case common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH:
		alternativeRules = OPTIONAL_ABSOLUTE_PATH
	case common.PARSER_RULE_CONTEXT_CONST_DECL_TYPE:
		alternativeRules = CONST_DECL_TYPE
	case common.PARSER_RULE_CONTEXT_CONST_DECL_RHS:
		alternativeRules = CONST_DECL_RHS
	case common.PARSER_RULE_CONTEXT_PARAMETER_START:
		alternativeRules = PARAMETER_START
	case common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION:
		alternativeRules = PARAMETER_START_WITHOUT_ANNOTATION
	case common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE:
		alternativeRules = ANNOT_DECL_OPTIONAL_TYPE
	case common.PARSER_RULE_CONTEXT_ANNOT_DECL_RHS:
		alternativeRules = ANNOT_DECL_RHS
	case common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS:
		alternativeRules = ANNOT_OPTIONAL_ATTACH_POINTS
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT:
		alternativeRules = ATTACH_POINT
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT:
		alternativeRules = ATTACH_POINT_IDENT
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT_END:
		alternativeRules = ATTACH_POINT_END
	case common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL:
		alternativeRules = XML_NAMESPACE_PREFIX_DECL
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_START:
		alternativeRules = ENUM_MEMBER_START
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS:
		alternativeRules = ENUM_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END:
		alternativeRules = ENUM_MEMBER_END
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS:
		alternativeRules = EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS
	case common.PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER:
		alternativeRules = LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER:
		alternativeRules = TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER
	case common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER:
		alternativeRules = MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION_START:
		alternativeRules = FUNC_TYPE_DESC_START
	case common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION_START:
		alternativeRules = MODULE_CLASS_DEFINITION_START
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = CLASS_DEF_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_SECOND_QUALIFIER:
		alternativeRules = CLASS_DEF_WITHOUT_SECOND_QUALIFIER
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_THIRD_QUALIFIER:
		alternativeRules = CLASS_DEF_WITHOUT_THIRD_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF:
		alternativeRules = OBJECT_CONSTRUCTOR_RHS
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_QUALIFIER:
		alternativeRules = OBJECT_FIELD_QUALIFIER
	case common.PARSER_RULE_CONTEXT_CONFIG_VAR_DECL_RHS:
		alternativeRules = CONFIG_VAR_DECL_RHS
	case common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE:
		alternativeRules = OPTIONAL_SERVICE_DECL_TYPE
	case common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS:
		alternativeRules = SERVICE_IDENT_RHS
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_START:
		alternativeRules = ABSOLUTE_RESOURCE_PATH_START
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END:
		alternativeRules = ABSOLUTE_RESOURCE_PATH_END
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_OR_VAR_DECL:
		alternativeRules = SERVICE_DECL_OR_VAR_DECL
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH:
		alternativeRules = OPTIONAL_RELATIVE_PATH
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_START:
		alternativeRules = RELATIVE_RESOURCE_PATH_START
	case common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT:
		alternativeRules = RESOURCE_PATH_SEGMENT
	case common.PARSER_RULE_CONTEXT_PATH_PARAM_OPTIONAL_ANNOTS:
		alternativeRules = PATH_PARAM_OPTIONAL_ANNOTS
	case common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS:
		alternativeRules = PATH_PARAM_ELLIPSIS
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME:
		alternativeRules = OPTIONAL_PATH_PARAM_NAME
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END:
		alternativeRules = RELATIVE_RESOURCE_PATH_END
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_START:
		alternativeRules = SERVICE_DECL_START
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER:
		alternativeRules = OPTIONAL_TYPE_PARAMETER
	case common.PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF:
		alternativeRules = MAP_TYPE_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF:
		alternativeRules = OBJECT_TYPE_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF:
		alternativeRules = STREAM_TYPE_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF:
		alternativeRules = TABLE_TYPE_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF:
		alternativeRules = PARAMETERIZED_TYPE_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF:
		alternativeRules = TYPE_DESC_RHS_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF:
		alternativeRules = TRANSACTION_STMT_RHS_OR_TYPE_REF
	case common.PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF:
		alternativeRules = TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF
	case common.PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF:
		alternativeRules = QUERY_EXPR_OR_VAR_REF
	case common.PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF:
		alternativeRules = ERROR_CONS_EXPR_OR_VAR_REF
	case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER:
		alternativeRules = QUALIFIED_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL_START:
		alternativeRules = MODULE_VAR_DECL_START
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_FIRST_QUAL:
		alternativeRules = MODULE_VAR_WITHOUT_FIRST_QUAL
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_SECOND_QUAL:
		alternativeRules = MODULE_VAR_WITHOUT_SECOND_QUAL
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS:
		alternativeRules = FUNC_DEF_OR_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START:
		alternativeRules = EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END:
		alternativeRules = TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END
	case common.PARSER_RULE_CONTEXT_END_OF_PARAMS_OR_NEXT_PARAM_START:
		alternativeRules = END_OF_PARAMS_OR_NEXT_PARAM_START
	case common.PARSER_RULE_CONTEXT_PARAM_START:
		alternativeRules = PARAM_START
	case common.PARSER_RULE_CONTEXT_PARAM_RHS:
		alternativeRules = PARAM_RHS
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_PARAM_RHS:
		alternativeRules = FUNC_TYPE_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL_START:
		alternativeRules = ANNOTATION_DECL_START
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON:
		alternativeRules = OPTIONAL_TOP_LEVEL_SEMICOLON
	case common.PARSER_RULE_CONTEXT_TUPLE_MEMBER:
		alternativeRules = TUPLE_MEMBER
	default:
		return b.seekMatchInStmtRelatedAlternativePaths(currentCtx, lookahead, currentDepth, matchingRulesCount,
			isEntryPoint)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternativeRules, isEntryPoint)
}

func (b *BallerinaParserErrorHandler) seekMatchInStmtRelatedAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInStmtRelatedAlternativePaths start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInStmtRelatedAlternativePaths end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResultValue(result))
	})
	var alternativeRules []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS:
		alternativeRules = VAR_DECL_RHS
	case common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		return b.seekInStatements(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
	case common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME:
		alternativeRules = TYPE_OR_VAR_NAME
	case common.PARSER_RULE_CONTEXT_ELSE_BLOCK:
		alternativeRules = ELSE_BLOCK
	case common.PARSER_RULE_CONTEXT_ELSE_BODY:
		alternativeRules = ELSE_BODY
	case common.PARSER_RULE_CONTEXT_CALL_STMT_START:
		alternativeRules = CALL_STATEMENT
	case common.PARSER_RULE_CONTEXT_RETURN_STMT_RHS:
		alternativeRules = RETURN_RHS
	case common.PARSER_RULE_CONTEXT_ARRAY_LENGTH:
		alternativeRules = ARRAY_LENGTH
	case common.PARSER_RULE_CONTEXT_STMT_START_WITH_EXPR_RHS:
		alternativeRules = STMT_START_WITH_EXPR_RHS
	case common.PARSER_RULE_CONTEXT_EXPR_STMT_RHS:
		alternativeRules = EXPR_STMT_RHS
	case common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START:
		alternativeRules = EXPRESSION_STATEMENT_START
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS:
		if !b.isInTypeDescContext() {
			panic("assertion failed")
		}
		alternativeRules = TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS:
		alternativeRules = STREAM_TYPE_FIRST_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS:
		alternativeRules = FUNCTION_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START:
		alternativeRules = FUNC_TYPE_FUNC_KEYWORD_RHS_START
	case common.PARSER_RULE_CONTEXT_WORKER_NAME_RHS:
		alternativeRules = WORKER_NAME_RHS
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN:
		alternativeRules = BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERNS_START:
		alternativeRules = LIST_BINDING_PATTERNS_START
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END:
		alternativeRules = LIST_BINDING_PATTERN_MEMBER_END
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER:
		alternativeRules = LIST_BINDING_PATTERN_CONTENTS
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END:
		alternativeRules = MAPPING_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END:
		alternativeRules = FIELD_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER:
		alternativeRules = MAPPING_BINDING_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS:
		alternativeRules = ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START:
		alternativeRules = ERROR_ARG_LIST_BINDING_PATTERN_START
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END:
		alternativeRules = ERROR_MESSAGE_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS:
		alternativeRules = ERROR_MESSAGE_BINDING_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN:
		alternativeRules = ERROR_FIELD_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END:
		alternativeRules = ERROR_FIELD_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS:
		alternativeRules = KEY_CONSTRAINTS_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_TYPE_DESC_RHS:
		alternativeRules = TABLE_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS:
		alternativeRules = TYPE_DESC_IN_TUPLE_RHS
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS:
		alternativeRules = TUPLE_TYPE_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END:
		alternativeRules = LIST_CONSTRUCTOR_MEMBER_END
	case common.PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS:
		alternativeRules = NIL_OR_PARENTHESISED_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS:
		alternativeRules = REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS
	case common.PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END:
		alternativeRules = REMOTE_CALL_OR_ASYNC_SEND_END
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH:
		alternativeRules = OPTIONAL_RESOURCE_ACCESS_PATH
	case common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT:
		alternativeRules = RESOURCE_ACCESS_PATH_SEGMENT
	case common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT:
		alternativeRules = COMPUTED_SEGMENT_OR_REST_SEGMENT
	case common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS:
		alternativeRules = RESOURCE_ACCESS_SEGMENT_RHS
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD:
		alternativeRules = OPTIONAL_RESOURCE_ACCESS_METHOD
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST:
		alternativeRules = OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST
	case common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS:
		alternativeRules = RECEIVE_WORKERS
	case common.PARSER_RULE_CONTEXT_RECEIVE_FIELD:
		alternativeRules = RECEIVE_FIELD
	case common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END:
		alternativeRules = RECEIVE_FIELD_END
	case common.PARSER_RULE_CONTEXT_WAIT_KEYWORD_RHS:
		alternativeRules = WAIT_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME_RHS:
		alternativeRules = WAIT_FIELD_NAME_RHS
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_END:
		alternativeRules = WAIT_FIELD_END
	case common.PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END:
		alternativeRules = WAIT_FUTURE_EXPR_END
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER:
		alternativeRules = OPTIONAL_PEER_WORKER
	case common.PARSER_RULE_CONTEXT_ROLLBACK_RHS:
		alternativeRules = ROLLBACK_RHS
	case common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS:
		alternativeRules = RETRY_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS:
		alternativeRules = RETRY_TYPE_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_RETRY_BODY:
		alternativeRules = RETRY_BODY
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER:
		alternativeRules = LIST_BP_OR_TUPLE_TYPE_MEMBER
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_RHS:
		alternativeRules = LIST_BP_OR_TUPLE_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END:
		alternativeRules = BRACKETED_LIST_MEMBER_END
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER:
		alternativeRules = BRACKETED_LIST_MEMBER
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS:
		alternativeRules = BRACKETED_LIST_RHS
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS:
		alternativeRules = BINDING_PATTERN_OR_VAR_REF_RHS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS:
		alternativeRules = TYPE_DESC_RHS_OR_BP_RHS
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_MEMBER_OR_ARRAY_LENGTH:
		alternativeRules = LIST_BINDING_MEMBER_OR_ARRAY_LENGTH
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS:
		alternativeRules = MATCH_PATTERN_LIST_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START:
		alternativeRules = MATCH_PATTERN_START
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERNS_START:
		alternativeRules = LIST_MATCH_PATTERNS_START
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER:
		alternativeRules = LIST_MATCH_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS:
		alternativeRules = LIST_MATCH_PATTERN_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START:
		alternativeRules = FIELD_MATCH_PATTERNS_START
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER:
		alternativeRules = FIELD_MATCH_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS:
		alternativeRules = FIELD_MATCH_PATTERN_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN:
		alternativeRules = ERROR_MATCH_PATTERN_OR_CONST_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS:
		alternativeRules = ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START:
		alternativeRules = ERROR_ARG_LIST_MATCH_PATTERN_START
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END:
		alternativeRules = ERROR_MESSAGE_MATCH_PATTERN_END
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS:
		alternativeRules = ERROR_MESSAGE_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN:
		alternativeRules = ERROR_FIELD_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS:
		alternativeRules = ERROR_FIELD_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN_RHS:
		alternativeRules = NAMED_ARG_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_JOIN_CLAUSE_START:
		alternativeRules = JOIN_CLAUSE_START
	case common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE_START:
		alternativeRules = INTERMEDIATE_CLAUSE_START
	case common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS:
		alternativeRules = REGULAR_COMPOUND_STMT_RHS
	case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL_START:
		alternativeRules = NAMED_WORKER_DECL_START
	case common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT_RHS:
		alternativeRules = ASSIGNMENT_STMT_RHS
	default:
		return b.seekMatchInExprRelatedAlternativePaths(currentCtx, lookahead, currentDepth, matchingRulesCount,
			isEntryPoint)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternativeRules, isEntryPoint)
}

func (b *BallerinaParserErrorHandler) seekMatchInExprRelatedAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result Result) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInExprRelatedAlternativePaths start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInExprRelatedAlternativePaths end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResultValue(result))
	})
	var alternativeRules []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION:
		alternativeRules = EXPRESSION_START
	case common.PARSER_RULE_CONTEXT_ARG_START:
		alternativeRules = ARG_START
	case common.PARSER_RULE_CONTEXT_ARG_START_OR_ARG_LIST_END:
		alternativeRules = ARG_START_OR_ARG_LIST_END
	case common.PARSER_RULE_CONTEXT_NAMED_OR_POSITIONAL_ARG_RHS:
		alternativeRules = NAMED_OR_POSITIONAL_ARG_RHS
	case common.PARSER_RULE_CONTEXT_ARG_END:
		alternativeRules = ARG_END
	case common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION:
		return b.seekInAccessExpression(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
	case common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD:
		alternativeRules = FIRST_MAPPING_FIELD_START
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD:
		alternativeRules = MAPPING_FIELD_START
	case common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD:
		alternativeRules = SPECIFIC_FIELD
	case common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS:
		alternativeRules = SPECIFIC_FIELD_RHS
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END:
		alternativeRules = MAPPING_FIELD_END
	case common.PARSER_RULE_CONTEXT_LET_VAR_DECL_START:
		alternativeRules = LET_VAR_DECL_START
	case common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END:
		alternativeRules = ORDER_KEY_LIST_END
	case common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT:
		alternativeRules = GROUPING_KEY_LIST_ELEMENT
	case common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END:
		alternativeRules = GROUPING_KEY_LIST_ELEMENT_END
	case common.PARSER_RULE_CONTEXT_TEMPLATE_MEMBER:
		alternativeRules = TEMPLATE_MEMBER
	case common.PARSER_RULE_CONTEXT_TEMPLATE_STRING_RHS:
		alternativeRules = TEMPLATE_STRING_RHS
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START:
		alternativeRules = CONSTANT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER:
		alternativeRules = LIST_CONSTRUCTOR_FIRST_MEMBER
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER:
		alternativeRules = LIST_CONSTRUCTOR_MEMBER
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM:
		alternativeRules = TYPE_CAST_PARAM
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_RHS:
		alternativeRules = TYPE_CAST_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS:
		alternativeRules = TABLE_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_ROW_LIST_RHS:
		alternativeRules = ROW_LIST_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_ROW_END:
		alternativeRules = TABLE_ROW_END
	case common.PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS:
		alternativeRules = KEY_SPECIFIER_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_KEY_RHS:
		alternativeRules = TABLE_KEY_RHS
	case common.PARSER_RULE_CONTEXT_NEW_KEYWORD_RHS:
		alternativeRules = NEW_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START:
		alternativeRules = TABLE_CONSTRUCTOR_OR_QUERY_START
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS:
		alternativeRules = TABLE_CONSTRUCTOR_OR_QUERY_RHS
	case common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS:
		alternativeRules = QUERY_PIPELINE_RHS
	case common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS:
		alternativeRules = BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_PARAM_END:
		alternativeRules = PARAM_END
	case common.PARSER_RULE_CONTEXT_ANNOTATION_REF_RHS:
		alternativeRules = ANNOTATION_REF_RHS
	case common.PARSER_RULE_CONTEXT_INFER_PARAM_END_OR_PARENTHESIS_END:
		alternativeRules = INFER_PARAM_END_OR_PARENTHESIS_END
	case common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR:
		alternativeRules = XML_NAVIGATE_EXPR
	case common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS:
		alternativeRules = XML_NAME_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START:
		alternativeRules = XML_ATOMIC_NAME_PATTERN_START
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER_RHS:
		alternativeRules = XML_ATOMIC_NAME_IDENTIFIER_RHS
	case common.PARSER_RULE_CONTEXT_XML_STEP_START:
		alternativeRules = XML_STEP_START
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND:
		alternativeRules = XML_STEP_EXTEND
	case common.PARSER_RULE_CONTEXT_XML_STEP_START_END:
		alternativeRules = XML_STEP_START_END
	case common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD:
		alternativeRules = OPTIONAL_MATCH_GUARD
	case common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END:
		alternativeRules = MEMBER_ACCESS_KEY_EXPR_END
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END:
		alternativeRules = LISTENERS_LIST_END
	case common.PARSER_RULE_CONTEXT_OBJECT_CONS_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = OBJECT_CONS_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_RESULT_CLAUSE:
		alternativeRules = RESULT_CLAUSE
	case common.PARSER_RULE_CONTEXT_EXPRESSION_RHS:
		return b.seekMatchInExpressionRhs(lookahead, currentDepth, matchingRulesCount, isEntryPoint, false)
	case common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS:
		return b.seekMatchInExpressionRhs(lookahead, currentDepth, matchingRulesCount, isEntryPoint, true)
	case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS:
		alternativeRules = ERROR_CONSTRUCTOR_RHS
	case common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_SEPARATOR:
		alternativeRules = SINGLE_OR_ALTERNATE_WORKER_SEPARATOR
	case common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION_START:
		alternativeRules = NATURAL_EXPRESSION_START
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PARENTHESIZED_ARG_LIST:
		alternativeRules = OPTIONAL_PARENTHESIZED_ARG_LIST
	default:
		panic("seekMatchInExprRelatedAlternativePaths found: " + currentCtx.String())
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternativeRules, isEntryPoint)
}

func (b *BallerinaParserErrorHandler) seekInStatements(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, currentMatches int, isEntryPoint bool) Result {
	nextToken := b.tokenReader.PeekN(lookahead)
	if nextToken.Kind() == common.SEMICOLON_TOKEN {
		result := b.seekMatchInSubTree(common.PARSER_RULE_CONTEXT_STATEMENT, lookahead+1, currentDepth+1,
			isEntryPoint)
		result.pushFix(NewSolutionWithDepth(ACTION_REMOVE, currentCtx, nextToken.Kind(), nextToken.Text(), currentDepth))
		return *b.getFinalResult(currentMatches, result)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, currentMatches, STATEMENTS, isEntryPoint)
}

func (b *BallerinaParserErrorHandler) seekInAccessExpression(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, currentMatches int, isEntryPoint bool) Result {
	nextToken := b.tokenReader.PeekN(lookahead)
	currentDepth++
	if nextToken.Kind() != common.IDENTIFIER_TOKEN {
		return *b.fixAndContinue(currentCtx, lookahead, currentDepth, currentMatches, isEntryPoint)
	}
	var nextContext common.ParserRuleContext
	nextNextToken := b.tokenReader.PeekN(lookahead + 1)
	switch nextNextToken.Kind() {
	case common.OPEN_PAREN_TOKEN:
		nextContext = common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.DOT_TOKEN:
		nextContext = common.PARSER_RULE_CONTEXT_DOT
	case common.OPEN_BRACKET_TOKEN:
		nextContext = common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR
	default:
		nextContext = b.getNextRuleForExpr()
	}
	currentMatches++
	lookahead++
	result := b.SeekMatch(nextContext, lookahead, currentDepth, isEntryPoint)
	return *b.getFinalResult(currentMatches, result)
}

func (b *BallerinaParserErrorHandler) seekMatchInExpressionRhs(lookahead int, currentDepth int, currentMatches int, isEntryPoint bool, allowFuncCall bool) Result {
	parentCtx := b.GetParentContext()
	var alternatives []common.ParserRuleContext
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ARG_LIST:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_ARG_LIST_END}
	case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	case common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_OPEN_BRACKET}
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST,
		common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	case common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_IN_KEYWORD, common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	case common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_LET_CLAUSE_END}
	case common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ORDER_DIRECTION, common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	case common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	case common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION:
		alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	default:
		if b.isParameter(parentCtx) {
			alternatives = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_COMMA}
		}
	}
	if alternatives != nil {
		if allowFuncCall {
			alternatives = b.modifyAlternativesWithArgListStart(alternatives)
		}
		return *b.seekInAlternativesPaths(lookahead, currentDepth, currentMatches, alternatives, isEntryPoint)
	}
	var nextContext common.ParserRuleContext
	if ((parentCtx == common.PARSER_RULE_CONTEXT_IF_BLOCK) || (parentCtx == common.PARSER_RULE_CONTEXT_WHILE_BLOCK)) || (parentCtx == common.PARSER_RULE_CONTEXT_FOREACH_STMT) {
		nextContext = common.PARSER_RULE_CONTEXT_BLOCK_STMT
	} else if parentCtx == common.PARSER_RULE_CONTEXT_MATCH_STMT {
		nextContext = common.PARSER_RULE_CONTEXT_MATCH_BODY
	} else if parentCtx == common.PARSER_RULE_CONTEXT_CALL_STMT {
		nextContext = common.PARSER_RULE_CONTEXT_METHOD_CALL_DOT
	} else if (((((b.isStatement(parentCtx) || (parentCtx == common.PARSER_RULE_CONTEXT_RECORD_FIELD)) || (parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)) || (parentCtx == common.PARSER_RULE_CONTEXT_CLASS_MEMBER)) || (parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER)) || (parentCtx == common.PARSER_RULE_CONTEXT_LISTENER_DECL)) || (parentCtx == common.PARSER_RULE_CONTEXT_CONSTANT_DECL) {
		nextContext = common.PARSER_RULE_CONTEXT_SEMICOLON
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ANNOTATIONS {
		nextContext = common.PARSER_RULE_CONTEXT_ANNOTATION_END
	} else if parentCtx == common.PARSER_RULE_CONTEXT_INTERPOLATION {
		nextContext = common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	} else if (parentCtx == common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION) || (parentCtx == common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS) {
		nextContext = common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	} else if parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF {
		nextContext = common.PARSER_RULE_CONTEXT_SEMICOLON
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS {
		nextContext = common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPR_LIST_END
	} else if parentCtx == common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION {
		nextContext = common.PARSER_RULE_CONTEXT_COLON
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST {
		nextContext = common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END
	} else if parentCtx == common.PARSER_RULE_CONTEXT_MATCH_BODY {
		nextContext = common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW
	} else if (parentCtx == common.PARSER_RULE_CONTEXT_SELECT_CLAUSE) || (parentCtx == common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE) {
		nextToken := b.tokenReader.PeekN(lookahead)
		switch nextToken.Kind() {
		case common.ON_KEYWORD, common.CONFLICT_KEYWORD:
			nextContext = common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE
		default:
			nextContext = common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_END
		}
	} else if parentCtx == common.PARSER_RULE_CONTEXT_JOIN_CLAUSE {
		nextContext = common.PARSER_RULE_CONTEXT_ON_CLAUSE
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ON_CLAUSE {
		nextContext = common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD
	} else if parentCtx == common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION {
		nextContext = common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	} else {
		panic("seekMatchInExpressionRhs found: " + parentCtx.String())
	}
	alternatives = b.getExpressionRhsAlternatives(nextContext)
	if allowFuncCall {
		alternatives = b.modifyAlternativesWithArgListStart(alternatives)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, currentMatches, alternatives, isEntryPoint)
}

func (b *BallerinaParserErrorHandler) getExpressionRhsAlternatives(nextContext common.ParserRuleContext) []common.ParserRuleContext {
	if ((nextContext == common.PARSER_RULE_CONTEXT_SEMICOLON) || (nextContext == common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_END)) || (nextContext == common.PARSER_RULE_CONTEXT_MATCH_BODY) {
		return []common.ParserRuleContext{nextContext, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_IS_KEYWORD, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN}
	}
	return []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_IS_KEYWORD, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN, nextContext}
}

func (b *BallerinaParserErrorHandler) modifyAlternativesWithArgListStart(alternatives []common.ParserRuleContext) []common.ParserRuleContext {
	// Create new slice with capacity for one additional element
	newAlternatives := make([]common.ParserRuleContext, len(alternatives)+1)
	// Copy all existing elements
	copy(newAlternatives, alternatives)
	// Add the new element at the end
	newAlternatives[len(alternatives)] = common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	return newAlternatives
}

func (b *BallerinaParserErrorHandler) GetNextRule(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
	b.startContextIfRequired(currentCtx)
	var parentCtx common.ParserRuleContext
	var nextToken tree.STToken
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_EOF:
		return common.PARSER_RULE_CONTEXT_EOF
	case common.PARSER_RULE_CONTEXT_COMP_UNIT:
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	case common.PARSER_RULE_CONTEXT_FUNC_DEF:
		return common.PARSER_RULE_CONTEXT_FUNC_DEF_START
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_FUNC_DEF_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE:
		return common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION_START
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC:
		return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_ASSIGN_OP:
		return b.getNextRuleForEqualOp()
	case common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_CLOSE_BRACE:
		return b.getNextRuleForCloseBrace(nextLookahead)
	case common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS:
		return b.getNextRuleForCloseParenthesis()
	case common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_BASIC_LITERAL:
		return b.getNextRuleForExpr()
	case common.PARSER_RULE_CONTEXT_FUNC_NAME:
		grandParentCtx := b.GetGrandParentContext()
		if (grandParentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER) || (grandParentCtx == common.PARSER_RULE_CONTEXT_CLASS_MEMBER) {
			return common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH
		}
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_OPEN_BRACE:
		return b.getNextRuleForOpenBrace()
	case common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS:
		return b.getNextRuleForOpenParenthesis()
	case common.PARSER_RULE_CONTEXT_SEMICOLON:
		return b.getNextRuleForSemicolon(nextLookahead)
	case common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_PARAMETER_NAME_RHS:
		return b.getNextRuleForVarName()
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM,
		common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM,
		common.PARSER_RULE_CONTEXT_REST_PARAM:
		return common.PARSER_RULE_CONTEXT_PARAM_START
	case common.PARSER_RULE_CONTEXT_REST_PARAM_RHS:
		b.SwitchContext(common.PARSER_RULE_CONTEXT_REST_PARAM)
		return common.PARSER_RULE_CONTEXT_ELLIPSIS
	case common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_VAR_DECL_STMT:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_EXPRESSION_RHS:
		return common.PARSER_RULE_CONTEXT_BINARY_OPERATOR
	case common.PARSER_RULE_CONTEXT_BINARY_OPERATOR:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_COMMA:
		return b.getNextRuleForComma()
	case common.PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE:
		return b.getNextRuleForParamType()
	case common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION:
		return common.PARSER_RULE_CONTEXT_TYPE_KEYWORD
	case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END:
		b.EndContext()
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == common.EOF_TOKEN {
			return common.PARSER_RULE_CONTEXT_EOF
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START:
		return common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END
	case common.PARSER_RULE_CONTEXT_ELLIPSIS:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR,
			common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR,
			common.PARSER_RULE_CONTEXT_ARG_LIST:
			return common.PARSER_RULE_CONTEXT_EXPRESSION
		case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST,
			common.PARSER_RULE_CONTEXT_BRACKETED_LIST,
			common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS:
			return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
		case common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN:
			return common.PARSER_RULE_CONTEXT_VAR_KEYWORD
		case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
			return common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME
		case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
			return common.PARSER_RULE_CONTEXT_EXPRESSION
		default:
			return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
		}
	case common.PARSER_RULE_CONTEXT_QUESTION_MARK:
		return b.getNextRuleForQuestionMark()
	case common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_RECORD_KEYWORD
	case common.PARSER_RULE_CONTEXT_ASTERISK:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
			return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
		case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN:
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS
		case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM,
			common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM:
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM
		default:
			return common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION
		}
	case common.PARSER_RULE_CONTEXT_TYPE_NAME:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_OBJECT_TYPE_START
	case common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONS_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OPEN_BRACKET:
		return b.getNextRuleForOpenBracket()
	case common.PARSER_RULE_CONTEXT_CLOSE_BRACKET:
		return b.getNextRuleForCloseBracket()
	case common.PARSER_RULE_CONTEXT_DOT:
		return b.getNextRuleForDot()
	case common.PARSER_RULE_CONTEXT_METHOD_CALL_DOT:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_BLOCK_STMT:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_IF_BLOCK:
		return common.PARSER_RULE_CONTEXT_IF_KEYWORD
	case common.PARSER_RULE_CONTEXT_WHILE_BLOCK:
		return common.PARSER_RULE_CONTEXT_WHILE_KEYWORD
	case common.PARSER_RULE_CONTEXT_DO_BLOCK:
		return common.PARSER_RULE_CONTEXT_DO_KEYWORD
	case common.PARSER_RULE_CONTEXT_CALL_STMT:
		return common.PARSER_RULE_CONTEXT_CALL_STMT_START
	case common.PARSER_RULE_CONTEXT_PANIC_STMT:
		return common.PARSER_RULE_CONTEXT_PANIC_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_CALL:
		return common.PARSER_RULE_CONTEXT_IMPORT_PREFIX
	case common.PARSER_RULE_CONTEXT_IMPORT_PREFIX, common.PARSER_RULE_CONTEXT_NAMESPACE_PREFIX:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_SLASH:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH:
			return common.PARSER_RULE_CONTEXT_IDENTIFIER
		case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
			return common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT
		case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
			return common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT
		}
		return common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME
	case common.PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME:
		return common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS
	case common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME:
		return common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME
	case common.PARSER_RULE_CONTEXT_IMPORT_DECL:
		return common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT:
		return common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD
	case common.PARSER_RULE_CONTEXT_BREAK_STATEMENT:
		return common.PARSER_RULE_CONTEXT_BREAK_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETURN_STMT:
		return common.PARSER_RULE_CONTEXT_RETURN_KEYWORD
	case common.PARSER_RULE_CONTEXT_FAIL_STATEMENT:
		return common.PARSER_RULE_CONTEXT_FAIL_KEYWORD
	case common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME:
		return common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS
	case common.PARSER_RULE_CONTEXT_COLON:
		return b.getNextRuleForColon()
	case common.PARSER_RULE_CONTEXT_VAR_REF_COLON:
		b.StartContext(common.PARSER_RULE_CONTEXT_VARIABLE_REF)
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_TYPE_REF_COLON:
		b.StartContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_SERVICE_DECL {
			return common.PARSER_RULE_CONTEXT_ON_KEYWORD
		}
		return common.PARSER_RULE_CONTEXT_COLON
	case common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL:
		return common.PARSER_RULE_CONTEXT_SERVICE_DECL_START
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD
	case common.PARSER_RULE_CONTEXT_LISTENER_DECL:
		return common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONSTANT_DECL:
		return common.PARSER_RULE_CONTEXT_CONST_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPEOF_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_QUESTION_MARK
	case common.PARSER_RULE_CONTEXT_UNARY_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_UNARY_OPERATOR
	case common.PARSER_RULE_CONTEXT_UNARY_OPERATOR:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_AT:
		return common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE
	case common.PARSER_RULE_CONTEXT_DOC_STRING:
		return common.PARSER_RULE_CONTEXT_ANNOTATIONS
	case common.PARSER_RULE_CONTEXT_ANNOTATIONS:
		return common.PARSER_RULE_CONTEXT_AT
	case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_VARIABLE_REF,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION,
		common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE,
		common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER:
		return common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER:
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == common.COLON_TOKEN {
			return common.PARSER_RULE_CONTEXT_COLON
		}
		fallthrough
	case common.PARSER_RULE_CONTEXT_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER:
		return b.getNextRuleForIdentifier()
	case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX:
		return common.PARSER_RULE_CONTEXT_COLON
	case common.PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT:
		return common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END
	case common.PARSER_RULE_CONTEXT_NIL_LITERAL:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_LOCAL_TYPE_DEFINITION_STMT:
		return common.PARSER_RULE_CONTEXT_TYPE_KEYWORD
	case common.PARSER_RULE_CONTEXT_RIGHT_ARROW:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN,
		common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN:
		return b.getNextRuleForDecimalIntegerLiteral()
	case common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START
	case common.PARSER_RULE_CONTEXT_LOCK_STMT:
		return common.PARSER_RULE_CONTEXT_LOCK_KEYWORD
	case common.PARSER_RULE_CONTEXT_LOCK_KEYWORD:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD:
		return common.PARSER_RULE_CONTEXT_RECORD_FIELD_START
	case common.PARSER_RULE_CONTEXT_ANNOTATION_TAG:
		return common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS
	case common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST:
		return common.PARSER_RULE_CONTEXT_ATTACH_POINT
	case common.PARSER_RULE_CONTEXT_FIELD_IDENT,
		common.PARSER_RULE_CONTEXT_FUNCTION_IDENT,
		common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT,
		common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT:
		return common.PARSER_RULE_CONTEXT_ATTACH_POINT_END
	case common.PARSER_RULE_CONTEXT_OBJECT_IDENT:
		return common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT
	case common.PARSER_RULE_CONTEXT_RECORD_IDENT:
		return common.PARSER_RULE_CONTEXT_FIELD_IDENT
	case common.PARSER_RULE_CONTEXT_SERVICE_IDENT:
		return common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS
	case common.PARSER_RULE_CONTEXT_REMOTE_IDENT:
		return common.PARSER_RULE_CONTEXT_FUNCTION_IDENT
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL:
		return common.PARSER_RULE_CONTEXT_ANNOTATION_DECL_START
	case common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION:
		return common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START
	case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL:
		return common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL_START
	case common.PARSER_RULE_CONTEXT_WORKER_NAME:
		return common.PARSER_RULE_CONTEXT_WORKER_NAME_RHS
	case common.PARSER_RULE_CONTEXT_FORK_STMT:
		return common.PARSER_RULE_CONTEXT_FORK_KEYWORD
	case common.PARSER_RULE_CONTEXT_XML_FILTER_EXPR:
		return common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN:
		return common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN
	case common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN:
		return common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN:
		return common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXPR:
		return common.PARSER_RULE_CONTEXT_XML_STEP_START
	case common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN,
		common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN:
		return common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_START
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_START
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH:
		return common.PARSER_RULE_CONTEXT_SERVICE_DECL_RHS
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_RHS:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_BLOCK:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_SERVICE_VAR_DECL_RHS:
		b.SwitchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_START
	case common.PARSER_RULE_CONTEXT_RESOURCE_PATH_PARAM:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_RESOURCE_ACCESSOR_DEF_OR_DECL_RHS:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_ERROR_KEYWORD
	case common.PARSER_RULE_CONTEXT_ERROR_CONS_ERROR_KEYWORD_RHS:
		b.StartContext(common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR)
		return common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_RHS:
		return b.getNextRuleForBindingPatternDefault()
	case common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS:
		return common.PARSER_RULE_CONTEXT_TUPLE_MEMBER
	case common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER:
		return common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS:
		return common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND_END,
		common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_END:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	default:
		return b.getNextRuleInternal(currentCtx, nextLookahead)
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleInternal(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
	var parentCtx common.ParserRuleContext
	var grandParentCtx common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_FOREACH_STMT:
		return common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPE_CAST:
		return common.PARSER_RULE_CONTEXT_LT
	case common.PARSER_RULE_CONTEXT_PIPE:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS:
			return common.PARSER_RULE_CONTEXT_EXPRESSION
		case common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN:
			return common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN
		case common.PARSER_RULE_CONTEXT_MATCH_PATTERN:
			return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START
		case common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER:
			return common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_KEY_SPECIFIER:
		return common.PARSER_RULE_CONTEXT_KEY_KEYWORD
	case common.PARSER_RULE_CONTEXT_LET_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_LET_KEYWORD
	case common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL:
		return common.PARSER_RULE_CONTEXT_LET_VAR_DECL_START
	case common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_END_OF_TYPE_DESC:
		return b.getNextRuleForTypeDescriptor()
	case common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ELLIPSIS
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME
	case common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME:
		return common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_LT:
		return b.getNextRuleForLt()
	case common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT:
		return common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT
	case common.PARSER_RULE_CONTEXT_GT:
		return b.getNextRuleForGt()
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT:
		return common.PARSER_RULE_CONTEXT_END_OF_PARAMS_OR_NEXT_PARAM_START
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START:
		b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_CAST)
		return common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM
	case common.PARSER_RULE_CONTEXT_TEMPLATE_END:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_TEMPLATE_START:
		return common.PARSER_RULE_CONTEXT_TEMPLATE_BODY
	case common.PARSER_RULE_CONTEXT_TEMPLATE_BODY:
		return common.PARSER_RULE_CONTEXT_TEMPLATE_MEMBER
	case common.PARSER_RULE_CONTEXT_TEMPLATE_STRING:
		return common.PARSER_RULE_CONTEXT_TEMPLATE_STRING_RHS
	case common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN:
		return common.PARSER_RULE_CONTEXT_ARG_LIST
	case common.PARSER_RULE_CONTEXT_ARG_LIST_END:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN
	case common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR:
			b.EndContext()
		case common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS:
			return common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND
		case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
			return common.PARSER_RULE_CONTEXT_ACTION_END
		case common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION:
			return common.PARSER_RULE_CONTEXT_OPEN_BRACE
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_ARG_LIST:
		return common.PARSER_RULE_CONTEXT_ARG_START_OR_ARG_LIST_END
	case common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_END:
		b.EndContext()
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR:
		return common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_VAR_DECL_STARTED_WITH_DENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_IN_TYPED_BP:
		b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM:
		return common.PARSER_RULE_CONTEXT_LT
	case common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_SELECT_CLAUSE:
		return common.PARSER_RULE_CONTEXT_SELECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE:
		return common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_WHERE_CLAUSE:
		return common.PARSER_RULE_CONTEXT_WHERE_KEYWORD
	case common.PARSER_RULE_CONTEXT_FROM_CLAUSE:
		return common.PARSER_RULE_CONTEXT_FROM_KEYWORD
	case common.PARSER_RULE_CONTEXT_LET_CLAUSE:
		return common.PARSER_RULE_CONTEXT_LET_KEYWORD
	case common.PARSER_RULE_CONTEXT_ORDER_BY_CLAUSE:
		return common.PARSER_RULE_CONTEXT_ORDER_KEYWORD
	case common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE:
		return common.PARSER_RULE_CONTEXT_GROUP_KEYWORD
	case common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE:
		return common.PARSER_RULE_CONTEXT_ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_LIMIT_CLAUSE:
		return common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_JOIN_CLAUSE:
		return common.PARSER_RULE_CONTEXT_JOIN_CLAUSE_START
	case common.PARSER_RULE_CONTEXT_ON_CLAUSE:
		return common.PARSER_RULE_CONTEXT_ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_FROM_CLAUSE
	case common.PARSER_RULE_CONTEXT_QUERY_CONSTRUCT_TYPE_RHS:
		b.StartContext(common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION)
		return common.PARSER_RULE_CONTEXT_FROM_CLAUSE
	case common.PARSER_RULE_CONTEXT_EXPRESSION_START_TABLE_KEYWORD_RHS:
		b.StartContext(common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION)
		return common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_RHS:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
			b.EndContext()
		}
		return common.PARSER_RULE_CONTEXT_RESULT_CLAUSE
	case common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
			b.EndContext()
		}
		return common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE_START
	case common.PARSER_RULE_CONTEXT_QUERY_ACTION_RHS:
		return common.PARSER_RULE_CONTEXT_DO_CLAUSE
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START
	case common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_MODULE_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS:
		b.EndContext()
		b.StartContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_STMT_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS:
		b.EndContext()
		if !b.isInTypeDescContext() {
			b.SwitchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
			b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS:
		return common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM
	case common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM:
		return common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_EXPLICIT_ANON_FUNC_EXPR_BODY_START:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		return common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START
	case common.PARSER_RULE_CONTEXT_ANNOTATION_END:
		return b.getNextRuleForAnnotationEnd(nextLookahead)
	case common.PARSER_RULE_CONTEXT_PLUS_TOKEN, common.PARSER_RULE_CONTEXT_MINUS_TOKEN:
		return common.PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS
	case common.PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS:
		return b.getNextRuleForExpr()
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START:
		return common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS
	case common.PARSER_RULE_CONTEXT_METHOD_NAME:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS {
			return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
		}
		return common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST
	case common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN:
		return common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME
	case common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN:
		return common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS
	case common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME:
		return common.PARSER_RULE_CONTEXT_COLON
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME:
		return common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME_RHS
	case common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPR_LIST_END:
		return b.getNextRuleForWaitExprListEnd()
	case common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN:
		return common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_DO_CLAUSE:
		return common.PARSER_RULE_CONTEXT_DO_KEYWORD
	case common.PARSER_RULE_CONTEXT_LET_CLAUSE_END:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS
		}
		return common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS
	case common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE_END,
		common.PARSER_RULE_CONTEXT_ORDER_CLAUSE_END,
		common.PARSER_RULE_CONTEXT_JOIN_CLAUSE_END:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS
	case common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN:
		return common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_QUESTION_MARK
	case common.PARSER_RULE_CONTEXT_TRANSACTION_STMT:
		return common.PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETRY_STMT:
		return common.PARSER_RULE_CONTEXT_RETRY_KEYWORD
	case common.PARSER_RULE_CONTEXT_ROLLBACK_STMT:
		return common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD
	case common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION:
		return common.PARSER_RULE_CONTEXT_ENUM_KEYWORD
	case common.PARSER_RULE_CONTEXT_MODULE_ENUM_NAME:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_START
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS:
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_MATCH_STMT:
		return common.PARSER_RULE_CONTEXT_MATCH_KEYWORD
	case common.PARSER_RULE_CONTEXT_MATCH_BODY:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_END:
		b.EndContext()
		return b.getNextRuleForMatchPattern()
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_RHS:
		return b.getNextRuleForMatchPattern()
	case common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACKET
	case common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ELLIPSIS
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_KEYWORD
	case common.PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA:
		return common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_KEYWORD
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG:
		return common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA:
		return common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_CAUSE_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION:
		return common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION_START
	case common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_SECOND_QUALIFIER
	case common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_THIRD_QUALIFIER
	case common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_CLASS_KEYWORD:
		return common.PARSER_RULE_CONTEXT_CLASS_NAME
	case common.PARSER_RULE_CONTEXT_CLASS_NAME:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL:
		return common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_START:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER
		}
		return common.PARSER_RULE_CONTEXT_OBJECT_FIELD_QUALIFIER
	case common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE:
		return common.PARSER_RULE_CONTEXT_ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS:
		grandParentCtx = b.GetGrandParentContext()
		if grandParentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR {
			return common.PARSER_RULE_CONTEXT_SEMICOLON
		} else {
			return common.PARSER_RULE_CONTEXT_OPTIONAL_FIELD_INITIALIZER
		}
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER:
		return common.PARSER_RULE_CONTEXT_FUNC_DEF
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL:
		return common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL_START
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL:
		return common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_FIRST_QUAL
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL:
		return common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_SECOND_QUAL
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL:
		return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT
	case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER
	case common.PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_MAP_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS:
		return b.getNextRuleForFuncTypeFuncKeywordRhs()
	case common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_TRANSACTION_KEYWORD_RHS:
		b.StartContext(common.PARSER_RULE_CONTEXT_TRANSACTION_STMT)
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ARRAY_LENGTH_START:
		b.SwitchContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		b.StartContext(common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR)
		return common.PARSER_RULE_CONTEXT_ARRAY_LENGTH
	case common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN:
		return common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION
	case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH
	case common.PARSER_RULE_CONTEXT_ACTION_END:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION {
			b.EndContext()
		}
		return b.getNextRuleForAction()
	case common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION_START
	default:
		return b.getNextRuleForKeywords(currentCtx, nextLookahead)
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForKeywords(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
	var parentCtx common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_PUBLIC_KEYWORD:
		parentCtx = b.GetParentContext()
		if (((parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR) || (parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER)) || (parentCtx == common.PARSER_RULE_CONTEXT_CLASS_MEMBER)) || (parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER) {
			return common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY
		} else if b.isParameter(parentCtx) {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM
		}
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER
	case common.PARSER_RULE_CONTEXT_PRIVATE_KEYWORD:
		return common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY
	case common.PARSER_RULE_CONTEXT_ON_KEYWORD:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL:
			return common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST
		case common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE:
			return common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD
		case common.PARSER_RULE_CONTEXT_ON_CLAUSE:
			return common.PARSER_RULE_CONTEXT_EXPRESSION
		case common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE:
			return common.PARSER_RULE_CONTEXT_FAIL_KEYWORD
		}
		return common.PARSER_RULE_CONTEXT_LISTENERS_LIST
	case common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE
	case common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_FINAL_KEYWORD:
		parentCtx = b.GetParentContext()
		if (parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER) || (parentCtx == common.PARSER_RULE_CONTEXT_CLASS_MEMBER) {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_CONST_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_ANNOTATION_DECL {
			return common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD
		}
		if parentCtx == common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION {
			return common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD
		}
		return common.PARSER_RULE_CONTEXT_CONST_DECL_TYPE
	case common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_PARENTHESIZED_ARG_LIST
	case common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_IS_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION
	case common.PARSER_RULE_CONTEXT_NULL_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD:
		return common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE
	case common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD:
		return common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT
	case common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD:
		return common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_WORKER_KEYWORD:
		return common.PARSER_RULE_CONTEXT_WORKER_NAME
	case common.PARSER_RULE_CONTEXT_IF_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_ELSE_KEYWORD:
		return common.PARSER_RULE_CONTEXT_ELSE_BODY
	case common.PARSER_RULE_CONTEXT_WHILE_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_FAIL_KEYWORD:
		if b.GetParentContext() == common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE {
			return common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_PANIC_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME
	case common.PARSER_RULE_CONTEXT_AS_KEYWORD:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_IMPORT_DECL:
			return common.PARSER_RULE_CONTEXT_IMPORT_PREFIX
		case common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION:
			return common.PARSER_RULE_CONTEXT_NAMESPACE_PREFIX
		}
		panic("next rule of as keyword found: " + parentCtx.String())
	case common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD, common.PARSER_RULE_CONTEXT_BREAK_KEYWORD:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_RETURN_KEYWORD:
		return common.PARSER_RULE_CONTEXT_RETURN_STMT_RHS
	case common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION:
			return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC:
			return common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START
		case common.PARSER_RULE_CONTEXT_FUNC_DEF:
			return common.PARSER_RULE_CONTEXT_FUNC_NAME
		}
		return common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_RECORD_KEYWORD:
		return common.PARSER_RULE_CONTEXT_RECORD_BODY_START
	case common.PARSER_RULE_CONTEXT_TYPE_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_NAME
	case common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR {
			return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF
		}
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OBJECT_KEYWORD_RHS:
		b.StartContext(common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR)
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_ABSTRACT_KEYWORD, common.PARSER_RULE_CONTEXT_CLIENT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_FORK_KEYWORD:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_TRAP_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_IN_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL {
			b.EndContext()
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_KEY_KEYWORD:
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS
		}
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_ERROR_KEYWORD:
		return b.getNextRuleForErrorKeyword()
	case common.PARSER_RULE_CONTEXT_LET_KEYWORD:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION:
			nextToken := b.tokenReader.PeekN(nextLookahead)
			nextNextToken := b.tokenReader.PeekN(nextLookahead + 1)
			if isEndOfLetVarDeclarations(nextToken, nextNextToken) {
				return common.PARSER_RULE_CONTEXT_LET_CLAUSE_END
			}
			return common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL
		case common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL:
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL
		}
		return common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL
	case common.PARSER_RULE_CONTEXT_TABLE_KEYWORD:
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM
		}
		return common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_STREAM_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION {
			return common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION
		}
		return common.PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN
	case common.PARSER_RULE_CONTEXT_NEW_KEYWORD:
		return common.PARSER_RULE_CONTEXT_NEW_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_XML_KEYWORD,
		common.PARSER_RULE_CONTEXT_RE_KEYWORD,
		common.PARSER_RULE_CONTEXT_STRING_KEYWORD,
		common.PARSER_RULE_CONTEXT_BASE16_KEYWORD,
		common.PARSER_RULE_CONTEXT_BASE64_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TEMPLATE_START
	case common.PARSER_RULE_CONTEXT_SELECT_KEYWORD, common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_WHERE_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
			b.EndContext()
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_ORDER_KEYWORD, common.PARSER_RULE_CONTEXT_GROUP_KEYWORD:
		return common.PARSER_RULE_CONTEXT_BY_KEYWORD
	case common.PARSER_RULE_CONTEXT_BY_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE {
			return common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT
		}
		return common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST
	case common.PARSER_RULE_CONTEXT_ORDER_DIRECTION:
		return common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END
	case common.PARSER_RULE_CONTEXT_FROM_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL {
			b.EndContext()
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_JOIN_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_START_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER
	case common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME:
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS:
			return common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END
		case common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER:
			return common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_SEPARATOR
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_WAIT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_WAIT_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_DO_KEYWORD, common.PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD:
		return common.PARSER_RULE_CONTEXT_ROLLBACK_RHS
	case common.PARSER_RULE_CONTEXT_RETRY_KEYWORD:
		return common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL {
			return common.PARSER_RULE_CONTEXT_WORKER_KEYWORD
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_ENUM_KEYWORD:
		return common.PARSER_RULE_CONTEXT_MODULE_ENUM_NAME
	case common.PARSER_RULE_CONTEXT_MATCH_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_READONLY_KEYWORD:
		parentCtx = b.GetParentContext()
		if ((parentCtx == common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR) || (parentCtx == common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR)) || (parentCtx == common.PARSER_RULE_CONTEXT_MAPPING_FIELD) {
			return common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD
		}
		panic("next rule of readonly keyword found: " + currentCtx.String())
	case common.PARSER_RULE_CONTEXT_DISTINCT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_VAR_KEYWORD:
		parentCtx = b.GetParentContext()
		if ((parentCtx == common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN) || (parentCtx == common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN)) || (parentCtx == common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG) {
			return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
		}
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD:
		if b.GetParentContext() != common.PARSER_RULE_CONTEXT_ON_CLAUSE {
			panic("assertion failed")
		}
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_OUTER_KEYWORD:
		return common.PARSER_RULE_CONTEXT_JOIN_KEYWORD
	case common.PARSER_RULE_CONTEXT_MAP_KEYWORD:
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION {
			return common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION
		}
		return common.PARSER_RULE_CONTEXT_LT
	default:
		panic("getNextRuleForKeywords found: " + currentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) startContextIfRequired(currentCtx common.ParserRuleContext) {
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_COMP_UNIT,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION,
		common.PARSER_RULE_CONTEXT_FUNC_DEF,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY,
		common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK,
		common.PARSER_RULE_CONTEXT_STATEMENT,
		common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT,
		common.PARSER_RULE_CONTEXT_REQUIRED_PARAM,
		common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM,
		common.PARSER_RULE_CONTEXT_REST_PARAM,
		common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION,
		common.PARSER_RULE_CONTEXT_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_ARG_LIST,
		common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD,
		common.PARSER_RULE_CONTEXT_IF_BLOCK,
		common.PARSER_RULE_CONTEXT_BLOCK_STMT,
		common.PARSER_RULE_CONTEXT_WHILE_BLOCK,
		common.PARSER_RULE_CONTEXT_PANIC_STMT,
		common.PARSER_RULE_CONTEXT_CALL_STMT,
		common.PARSER_RULE_CONTEXT_IMPORT_DECL,
		common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT,
		common.PARSER_RULE_CONTEXT_BREAK_STATEMENT,
		common.PARSER_RULE_CONTEXT_RETURN_STMT,
		common.PARSER_RULE_CONTEXT_FAIL_STATEMENT,
		common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME,
		common.PARSER_RULE_CONTEXT_LISTENERS_LIST,
		common.PARSER_RULE_CONTEXT_SERVICE_DECL,
		common.PARSER_RULE_CONTEXT_LISTENER_DECL,
		common.PARSER_RULE_CONTEXT_CONSTANT_DECL,
		common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_ANNOTATIONS,
		common.PARSER_RULE_CONTEXT_VARIABLE_REF,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE,
		common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE,
		common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_LOCAL_TYPE_DEFINITION_STMT,
		common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT,
		common.PARSER_RULE_CONTEXT_NIL_LITERAL,
		common.PARSER_RULE_CONTEXT_LOCK_STMT,
		common.PARSER_RULE_CONTEXT_ANNOTATION_DECL,
		common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST,
		common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION,
		common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION,
		common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL,
		common.PARSER_RULE_CONTEXT_FORK_STMT,
		common.PARSER_RULE_CONTEXT_FOREACH_STMT,
		common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_TYPE_CAST,
		common.PARSER_RULE_CONTEXT_KEY_SPECIFIER,
		common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST,
		common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM,
		common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS,
		common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS,
		common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS,
		common.PARSER_RULE_CONTEXT_DO_CLAUSE,
		common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR,
		common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION,
		common.PARSER_RULE_CONTEXT_DO_BLOCK,
		common.PARSER_RULE_CONTEXT_TRANSACTION_STMT,
		common.PARSER_RULE_CONTEXT_RETRY_STMT,
		common.PARSER_RULE_CONTEXT_ROLLBACK_STMT,
		common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION,
		common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST,
		common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN,
		common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN,
		common.PARSER_RULE_CONTEXT_MATCH_STMT,
		common.PARSER_RULE_CONTEXT_MATCH_BODY,
		common.PARSER_RULE_CONTEXT_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG,
		common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_SELECT_CLAUSE,
		common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE,
		common.PARSER_RULE_CONTEXT_JOIN_CLAUSE,
		common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE,
		common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE,
		common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS,
		common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH,
		common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH,
		common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR,
		common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION,
		common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION,
		common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS,
		common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER,
		common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS,
		common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY:
		b.StartContext(currentCtx)
	default:
		break
	}
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE,
		common.PARSER_RULE_CONTEXT_ON_CLAUSE:
		b.SwitchContext(currentCtx)
	default:
		break
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForCloseParenthesis() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_PARAM_LIST {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS
	} else if b.isParameter(parentCtx) {
		b.EndContext()
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS
	} else if parentCtx == common.PARSER_RULE_CONTEXT_NIL_LITERAL {
		b.EndContext()
		return b.getNextRuleForExpr()
	} else if parentCtx == common.PARSER_RULE_CONTEXT_KEY_SPECIFIER {
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS
	} else if parentCtx == common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	} else if b.isInTypeDescContext() {
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	} else if parentCtx == common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_INFER_PARAM_END_OR_PARENTHESIS_END
	} else if parentCtx == common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN {
		b.EndContext()
		return b.getNextRuleForMatchPattern()
	} else if parentCtx == common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN {
		b.EndContext()
		b.EndContext()
		return b.getNextRuleForMatchPattern()
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN {
		b.EndContext()
		return b.getNextRuleForBindingPatternDefault()
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG {
		b.EndContext()
		b.EndContext()
		return b.getNextRuleForBindingPatternDefault()
	}
	return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
}

func (b *BallerinaParserErrorHandler) getNextRuleForOpenParenthesis() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT {
		return common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START
	} else if ((b.isStatement(parentCtx) || b.isExpressionContext(parentCtx)) || (parentCtx == common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR)) || (parentCtx == common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION) {
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	} else if ((((parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE) || (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC)) || (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF)) || (parentCtx == common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION)) || (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC) {
		b.StartContext(common.PARSER_RULE_CONTEXT_PARAM_LIST)
		return common.PARSER_RULE_CONTEXT_PARAM_LIST
	} else if parentCtx == common.PARSER_RULE_CONTEXT_NIL_LITERAL {
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	} else if parentCtx == common.PARSER_RULE_CONTEXT_KEY_SPECIFIER {
		return common.PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS
	} else if b.isInTypeDescContext() {
		b.StartContext(common.PARSER_RULE_CONTEXT_KEY_SPECIFIER)
		return common.PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS
	} else if b.isParameter(parentCtx) {
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN {
		return common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG
	} else if b.isInMatchPatternCtx(parentCtx) {
		b.StartContext(common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN)
		return common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN {
		return common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START
	}
	return common.PARSER_RULE_CONTEXT_EXPRESSION
}

func (b *BallerinaParserErrorHandler) isInMatchPatternCtx(context common.ParserRuleContext) bool {
	switch context {
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForOpenBrace() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER
	case common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION:
		return common.PARSER_RULE_CONTEXT_CLASS_MEMBER
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_SERVICE_DECL:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER
	case common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_RECORD_FIELD
	case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD
	case common.PARSER_RULE_CONTEXT_FORK_STMT:
		return common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL
	case common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS:
		return common.PARSER_RULE_CONTEXT_RECEIVE_FIELD
	case common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS:
		return common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME
	case common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START
	case common.PARSER_RULE_CONTEXT_MATCH_BODY:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	default:
		return common.PARSER_RULE_CONTEXT_STATEMENT
	}
}

func (b *BallerinaParserErrorHandler) isExpressionContext(ctx common.ParserRuleContext) bool {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST,
		common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME,
		common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_INTERPOLATION,
		common.PARSER_RULE_CONTEXT_ARG_LIST,
		common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST,
		common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE,
		common.PARSER_RULE_CONTEXT_SELECT_CLAUSE,
		common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE,
		common.PARSER_RULE_CONTEXT_JOIN_CLAUSE,
		common.PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForParamType() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM, common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM:
		if b.HasAncestorContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC) {
			return common.PARSER_RULE_CONTEXT_FUNC_TYPE_PARAM_RHS
		}
		return common.PARSER_RULE_CONTEXT_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_REST_PARAM:
		return common.PARSER_RULE_CONTEXT_ELLIPSIS
	default:
		panic("getNextRuleForParamType found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForComma() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_PARAM_LIST,
		common.PARSER_RULE_CONTEXT_REQUIRED_PARAM,
		common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM,
		common.PARSER_RULE_CONTEXT_REST_PARAM:
		b.EndContext()
		return parentCtx
	case common.PARSER_RULE_CONTEXT_ARG_LIST:
		return common.PARSER_RULE_CONTEXT_ARG_START
	case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_MAPPING_FIELD
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST, common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE:
		return common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT
	case common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST:
		return common.PARSER_RULE_CONTEXT_ATTACH_POINT
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR
	case common.PARSER_RULE_CONTEXT_KEY_SPECIFIER:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL:
		return common.PARSER_RULE_CONTEXT_LET_VAR_DECL_START
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC:
		return common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR
	case common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS:
		return common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM
	case common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS:
		return common.PARSER_RULE_CONTEXT_TUPLE_MEMBER
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS:
		return common.PARSER_RULE_CONTEXT_RECEIVE_FIELD
	case common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS:
		return common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_START
	case common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR:
		return common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST:
		return common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST:
		return common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN_RHS
	default:
		panic("getNextRuleForComma found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForTypeDescriptor() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_ANNOTATION_TAG
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_GT
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC:
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		case common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE:
			return common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC:
			return common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY
		case common.PARSER_RULE_CONTEXT_FUNC_DEF:
			grandParentCtx := b.GetGrandParentContext()
			if grandParentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER {
				return common.PARSER_RULE_CONTEXT_SEMICOLON
			} else {
				return common.PARSER_RULE_CONTEXT_FUNC_BODY
			}
		case common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION:
			return common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY
		case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL:
			return common.PARSER_RULE_CONTEXT_BLOCK_STMT
		default:
			panic("next rule of type-desc-in-return-type found: " + parentCtx.String())
		}
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_COMP_UNIT:
		b.StartContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL:
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC:
		return common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS:
		return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS
	case common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER
	default:
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	}
}

func (b *BallerinaParserErrorHandler) isInTypeDescContext() bool {
	switch b.GetParentContext() {
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM,
		common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForEqualOp() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY:
		return common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM, common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM:
		return common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_ARG_LIST,
		common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER,
		common.PARSER_RULE_CONTEXT_LISTENER_DECL,
		common.PARSER_RULE_CONTEXT_CONSTANT_DECL,
		common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST,
		common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE:
		b.SwitchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_BINDING_PATTERN
	default:
		if b.isStatement(parentCtx) {
			return common.PARSER_RULE_CONTEXT_EXPRESSION
		}
		panic("getNextRuleForEqualOp found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForCloseBrace(nextLookahead int) common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	var nextToken tree.STToken
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK:
		b.EndContext()
		return b.getNextRuleForCloseBraceInFuncBody()
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER:
		b.EndContext()
		fallthrough
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL, common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER:
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_SERVICE_DECL {
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
		}
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		b.EndContext()
		fallthrough
	case common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_BLOCK_STMT, common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT:
		b.EndContext()
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_LOCK_STMT,
			common.PARSER_RULE_CONTEXT_FOREACH_STMT,
			common.PARSER_RULE_CONTEXT_WHILE_BLOCK,
			common.PARSER_RULE_CONTEXT_DO_BLOCK,
			common.PARSER_RULE_CONTEXT_RETRY_STMT:
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS
		case common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE:
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_STATEMENT
		case common.PARSER_RULE_CONTEXT_IF_BLOCK:
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_ELSE_BLOCK
		case common.PARSER_RULE_CONTEXT_TRANSACTION_STMT:
			b.EndContext()
			parentCtx = b.GetParentContext()
			if parentCtx == common.PARSER_RULE_CONTEXT_RETRY_STMT {
				b.EndContext()
			}
			return common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS
		case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL:
			b.EndContext()
			parentCtx = b.GetParentContext()
			if parentCtx == common.PARSER_RULE_CONTEXT_FORK_STMT {
				nextToken = b.tokenReader.PeekN(nextLookahead)
				switch nextToken.Kind() {
				case common.CLOSE_BRACE_TOKEN:
					return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
				default:
					return common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS
				}
			} else {
				return common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS
			}
		case common.PARSER_RULE_CONTEXT_MATCH_BODY:
			return common.PARSER_RULE_CONTEXT_MATCH_PATTERN
		case common.PARSER_RULE_CONTEXT_DO_CLAUSE:
			return common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_END
		default:
			return common.PARSER_RULE_CONTEXT_STATEMENT
		}
	case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR:
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR {
			return common.PARSER_RULE_CONTEXT_TABLE_ROW_END
		}
		if parentCtx == common.PARSER_RULE_CONTEXT_ANNOTATIONS {
			return common.PARSER_RULE_CONTEXT_ANNOTATION_END
		}
		return b.getNextRuleForExpr()
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST:
		return common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		b.EndContext()
		return b.getNextRuleForBindingPatternDefault()
	case common.PARSER_RULE_CONTEXT_FORK_STMT:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_STATEMENT
	case common.PARSER_RULE_CONTEXT_INTERPOLATION:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TEMPLATE_MEMBER
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS,
		common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS,
		common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST:
		b.EndContext()
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON
	case common.PARSER_RULE_CONTEXT_MATCH_BODY:
		b.EndContext()
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		b.EndContext()
		return b.getNextRuleForMatchPattern()
	case common.PARSER_RULE_CONTEXT_MATCH_STMT:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS
	default:
		panic("getNextRuleForCloseBrace found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForCloseBraceInFuncBody() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		return common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START
	case common.PARSER_RULE_CONTEXT_COMP_UNIT:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON
	case common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE:
		b.EndContext()
		return b.getNextRuleForCloseBraceInFuncBody()
	default:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForAnnotationEnd(nextLookahead int) common.ParserRuleContext {
	var parentCtx common.ParserRuleContext
	var nextToken = b.tokenReader.PeekN(nextLookahead)
	if nextToken.Kind() == common.AT_TOKEN {
		return common.PARSER_RULE_CONTEXT_AT
	}
	b.EndContext()
	parentCtx = b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_COMP_UNIT:
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA
	case common.PARSER_RULE_CONTEXT_FUNC_DEF,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD:
		return common.PARSER_RULE_CONTEXT_RECORD_FIELD_WITHOUT_METADATA
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER:
		return common.PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		return common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK:
		return common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY:
		return common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPE_CAST:
		return common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_RHS
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST:
		return common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM
	case common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS:
		return common.PARSER_RULE_CONTEXT_TUPLE_MEMBER
	default:
		if b.isParameter(parentCtx) {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForVarName() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT:
		return common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT_RHS
	case common.PARSER_RULE_CONTEXT_CALL_STMT:
		return common.PARSER_RULE_CONTEXT_ARG_LIST
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM, common.PARSER_RULE_CONTEXT_PARAM_LIST:
		return common.PARSER_RULE_CONTEXT_REQUIRED_PARAM_NAME_RHS
	case common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_REST_PARAM:
		return common.PARSER_RULE_CONTEXT_PARAM_END
	case common.PARSER_RULE_CONTEXT_FOREACH_STMT:
		return common.PARSER_RULE_CONTEXT_IN_KEYWORD
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER:
		return b.getNextRuleForBindingPatternWithCapture(true)
	case common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER,
		common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return b.getNextRuleForBindingPatternDefault()
	case common.PARSER_RULE_CONTEXT_LISTENER_DECL, common.PARSER_RULE_CONTEXT_CONSTANT_DECL:
		return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD:
		return common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS
	case common.PARSER_RULE_CONTEXT_ARG_LIST:
		return common.PARSER_RULE_CONTEXT_NAMED_OR_POSITIONAL_ARG_RHS
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		return common.PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_KEY_SPECIFIER:
		return common.PARSER_RULE_CONTEXT_TABLE_KEY_RHS
	case common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL:
		return common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS
	case common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION, common.PARSER_RULE_CONTEXT_JOIN_CLAUSE:
		return common.PARSER_RULE_CONTEXT_IN_KEYWORD
	case common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN:
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN {
			return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
		}
		if (parentCtx == common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN) || (parentCtx == common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG) {
			return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
		}
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_COLON
	case common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE:
		return common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END
	default:
		if b.isStatement(parentCtx) {
			return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS
		}
		panic("getNextRuleForVarName found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForSemicolon(nextLookahead int) common.ParserRuleContext {
	var nextToken tree.STToken
	parentCtx := b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY {
		b.EndContext()
		return b.getNextRuleForSemicolon(nextLookahead)
	} else if parentCtx == common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION {
		b.EndContext()
		return b.getNextRuleForSemicolon(nextLookahead)
	} else if b.isExpressionContext(parentCtx) {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_STATEMENT
	} else if parentCtx == common.PARSER_RULE_CONTEXT_VAR_DECL_STMT {
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_COMP_UNIT {
			return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
		}
		return common.PARSER_RULE_CONTEXT_STATEMENT
	} else if b.isStatement(parentCtx) {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_STATEMENT
	} else if parentCtx == common.PARSER_RULE_CONTEXT_RECORD_FIELD {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END
	} else if parentCtx == common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION {
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_COMP_UNIT {
			return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
		}
		return common.PARSER_RULE_CONTEXT_STATEMENT
	} else if ((parentCtx == common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION) || (parentCtx == common.PARSER_RULE_CONTEXT_LISTENER_DECL)) || (parentCtx == common.PARSER_RULE_CONTEXT_ANNOTATION_DECL) {
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else if parentCtx == common.PARSER_RULE_CONTEXT_CONSTANT_DECL {
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK {
			return common.PARSER_RULE_CONTEXT_STATEMENT
		}
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else if ((parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER) || (parentCtx == common.PARSER_RULE_CONTEXT_CLASS_MEMBER)) || (parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER) {
		if b.isEndOfObjectTypeNode(nextLookahead) {
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
		}
		if parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER {
			return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START
		} else {
			return common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START
		}
	} else if parentCtx == common.PARSER_RULE_CONTEXT_IMPORT_DECL {
		b.EndContext()
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == common.EOF_TOKEN {
			return common.PARSER_RULE_CONTEXT_EOF
		}
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST {
		b.EndContext()
		b.EndContext()
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == common.EOF_TOKEN {
			return common.PARSER_RULE_CONTEXT_EOF
		}
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else if (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF) || (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE) {
		b.EndContext()
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == common.EOF_TOKEN {
			return common.PARSER_RULE_CONTEXT_EOF
		}
		return b.getNextRuleForSemicolon(nextLookahead)
	} else if parentCtx == common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION {
		return common.PARSER_RULE_CONTEXT_CLASS_MEMBER
	} else if parentCtx == common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR {
		return common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER
	} else if parentCtx == common.PARSER_RULE_CONTEXT_COMP_UNIT {
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else {
		panic("getNextRuleForSemicolon found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForDot() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_IMPORT_DECL:
		return common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_RESOURCE_ACCESSOR_DEF_OR_DECL_RHS
	case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
		return common.PARSER_RULE_CONTEXT_METHOD_NAME
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS:
		return common.PARSER_RULE_CONTEXT_METHOD_NAME
	default:
		return common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForQuestionMark() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	default:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForOpenBracket() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_ARRAY_LENGTH
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_ROW_LIST_RHS
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERNS_START
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERNS_START
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_PATH_PARAM_OPTIONAL_ANNOTS
	case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
		return common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT
	default:
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS
		}
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForCloseBracket() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_TUPLE_MEMBERS:
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
			return b.getNextRuleForCloseBracket()
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_COLON
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN:
		b.EndContext()
		return b.getNextRuleForBindingPatternDefault()
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR,
		common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR:
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS {
			return common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND
		}
		return b.getNextRuleForExpr()
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST:
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST {
			return common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END
		}
		return common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_RHS
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN:
		b.EndContext()
		return b.getNextRuleForMatchPattern()
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END
	case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
		return common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS
	default:
		return b.getNextRuleForExpr()
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForDecimalIntegerLiteral() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION:
		b.EndContext()
		return b.getNextRuleForConstExpr()
	default:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForExpr() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION {
		b.EndContext()
		return b.getNextRuleForConstExpr()
	}
	return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
}

func (b *BallerinaParserErrorHandler) getNextRuleForExprStartsWithVarRef() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION:
		b.EndContext()
		return b.getNextRuleForConstExpr()
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	case common.PARSER_RULE_CONTEXT_CALL_STMT:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	}
	return common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS
}

func (b *BallerinaParserErrorHandler) getNextRuleForConstExpr() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION:
		return common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL
	default:
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return b.getNextRuleForMatchPattern()
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForLt() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_TYPE_CAST:
		return common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM
	default:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForGt() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC {
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR {
			b.EndContext()
			return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
		}
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	}
	if b.isInTypeDescContext() {
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
	}
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_TABLE_TYPE_DESC_RHS
	case common.PARSER_RULE_CONTEXT_RETRY_STMT:
		return common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS
	}
	if parentCtx == common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN {
		b.EndContext()
		parentCtx = b.GetParentContext()
		if parentCtx == common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS {
			return common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND
		}
		return common.PARSER_RULE_CONTEXT_XML_STEP_START_END
	}
	b.EndContext()
	return common.PARSER_RULE_CONTEXT_EXPRESSION
}

func (b *BallerinaParserErrorHandler) getNextRuleForBindingPatternDefault() common.ParserRuleContext {
	return b.getNextRuleForBindingPatternWithCapture(false)
}

func (b *BallerinaParserErrorHandler) getNextRuleForBindingPatternWithCapture(isCaptureBP bool) common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN:
		b.EndContext()
		return b.getNextRuleForBindingPatternWithCapture(isCaptureBP)
	case common.PARSER_RULE_CONTEXT_FOREACH_STMT,
		common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_JOIN_CLAUSE:
		return common.PARSER_RULE_CONTEXT_IN_KEYWORD
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST,
		common.PARSER_RULE_CONTEXT_BRACKETED_LIST:
		return common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN:
		b.EndContext()
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN:
			return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
		case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
			return common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS
		}
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACE
	case common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT:
		b.SwitchContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
		if isCaptureBP {
			return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS
		} else {
			return common.PARSER_RULE_CONTEXT_ASSIGN_OP
		}
	case common.PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT:
		if isCaptureBP {
			return common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS
		} else {
			return common.PARSER_RULE_CONTEXT_ASSIGN_OP
		}
	case common.PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT,
		common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	default:
		return b.getNextRuleForMatchPattern()
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForWaitExprListEnd() common.ParserRuleContext {
	b.EndContext()
	return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
}

func (b *BallerinaParserErrorHandler) getNextRuleForIdentifier() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_VARIABLE_REF:
		b.EndContext()
		return b.getNextRuleForExprStartsWithVarRef()
	case common.PARSER_RULE_CONTEXT_TYPE_REFERENCE:
		b.EndContext()
		return b.getNextRuleForTypeReference()
	case common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION:
		b.EndContext()
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_ANNOT_REFERENCE:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ANNOTATION_REF_RHS
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL:
		return common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS
	case common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ASSIGN_OP
	case common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_COMP_UNIT:
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER,
		common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH:
		return common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END
	case common.PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION:
		return common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	default:
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		panic("getNextRuleForIdentifier found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForColon() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS:
		return common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME
	case common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS:
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_EXPRESSION
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_VARIABLE_NAME
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN:
		return common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER_RHS
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN
	default:
		return common.PARSER_RULE_CONTEXT_IDENTIFIER
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForMatchPattern() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END
	default:
		return common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForTypeReference() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_OPEN_BRACE
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR:
		b.EndContext()
		return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	default:
		if b.isInTypeDescContext() {
			return common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS
		}
		panic("getNextRuleForTypeReference found: " + parentCtx.String())
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForErrorKeyword() common.ParserRuleContext {
	if b.isInTypeDescContext() {
		return common.PARSER_RULE_CONTEXT_LT
	}
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS
	case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR:
		return common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS
	default:
		return common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	}
}

func (b *BallerinaParserErrorHandler) getNextRuleForFuncTypeFuncKeywordRhs() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE {
		b.EndContext()
		parentCtx = b.GetParentContext()
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER,
			common.PARSER_RULE_CONTEXT_CLASS_MEMBER,
			common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER:
			b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
		case common.PARSER_RULE_CONTEXT_COMP_UNIT:
			fallthrough
		default:
			b.StartContext(common.PARSER_RULE_CONTEXT_VAR_DECL_STMT)
			b.StartContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN)
		}
	} else if b.GetGrandParentContext() == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER {
		b.SwitchContext(common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER)
	}
	if !b.isInTypeDescContext() {
		panic("assertion failed")
	}
	b.StartContext(common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC)
	return common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START
}

func (b *BallerinaParserErrorHandler) getNextRuleForAction() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_MATCH_STMT:
		return common.PARSER_RULE_CONTEXT_MATCH_BODY
	case common.PARSER_RULE_CONTEXT_FOREACH_STMT:
		return common.PARSER_RULE_CONTEXT_BLOCK_STMT
	default:
		return common.PARSER_RULE_CONTEXT_SEMICOLON
	}
}

func (b *BallerinaParserErrorHandler) isStatement(parentCtx common.ParserRuleContext) bool {
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_STATEMENT,
		common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT,
		common.PARSER_RULE_CONTEXT_IF_BLOCK,
		common.PARSER_RULE_CONTEXT_BLOCK_STMT,
		common.PARSER_RULE_CONTEXT_WHILE_BLOCK,
		common.PARSER_RULE_CONTEXT_DO_BLOCK,
		common.PARSER_RULE_CONTEXT_CALL_STMT,
		common.PARSER_RULE_CONTEXT_PANIC_STMT,
		common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT,
		common.PARSER_RULE_CONTEXT_BREAK_STATEMENT,
		common.PARSER_RULE_CONTEXT_RETURN_STMT,
		common.PARSER_RULE_CONTEXT_FAIL_STATEMENT,
		common.PARSER_RULE_CONTEXT_LOCAL_TYPE_DEFINITION_STMT,
		common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT,
		common.PARSER_RULE_CONTEXT_LOCK_STMT,
		common.PARSER_RULE_CONTEXT_FORK_STMT,
		common.PARSER_RULE_CONTEXT_FOREACH_STMT,
		common.PARSER_RULE_CONTEXT_TRANSACTION_STMT,
		common.PARSER_RULE_CONTEXT_RETRY_STMT,
		common.PARSER_RULE_CONTEXT_ROLLBACK_STMT,
		common.PARSER_RULE_CONTEXT_AMBIGUOUS_STMT,
		common.PARSER_RULE_CONTEXT_MATCH_STMT:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) isBinaryOperator(token tree.STToken) bool {
	switch token.Kind() {
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
		common.DOUBLE_LT_TOKEN,
		common.DOUBLE_GT_TOKEN,
		common.TRIPPLE_GT_TOKEN,
		common.ELLIPSIS_TOKEN,
		common.DOUBLE_DOT_LT_TOKEN,
		common.ELVIS_TOKEN:
		return true
	case common.RIGHT_ARROW_TOKEN,
		common.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) isParameter(ctx common.ParserRuleContext) bool {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM, common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM, common.PARSER_RULE_CONTEXT_REST_PARAM, common.PARSER_RULE_CONTEXT_PARAM_LIST:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) GetInsertSolution(ctx common.ParserRuleContext) *Solution {
	kind := b.GetExpectedTokenKind(ctx)
	if kind != common.NONE {
		return NewSolution(ACTION_INSERT, ctx, kind, ctx.String())
	}
	if b.HasAlternativePaths(ctx) {
		ctx = b.getShortestAlternative(ctx)
		return b.GetInsertSolution(ctx)
	}
	ctx = b.GetNextRule(ctx, 1)
	return b.GetInsertSolution(ctx)
}

func (b *BallerinaParserErrorHandler) GetExpectedTokenKind(ctx common.ParserRuleContext) common.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY:
		return common.EQUAL_TOKEN
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK:
		return common.OPEN_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_FUNC_DEF,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC:
		return common.FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR:
		return common.ANY_KEYWORD
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM,
		common.PARSER_RULE_CONTEXT_VAR_DECL_STMT,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT,
		common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM,
		common.PARSER_RULE_CONTEXT_REST_PARAM,
		common.PARSER_RULE_CONTEXT_TYPE_NAME,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION,
		common.PARSER_RULE_CONTEXT_TYPE_REFERENCE,
		common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_NAME,
		common.PARSER_RULE_CONTEXT_CLASS_NAME,
		common.PARSER_RULE_CONTEXT_VARIABLE_NAME,
		common.PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME,
		common.PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME,
		common.PARSER_RULE_CONTEXT_IMPORT_PREFIX,
		common.PARSER_RULE_CONTEXT_VARIABLE_REF,
		common.PARSER_RULE_CONTEXT_BASIC_LITERAL, // return var-ref for any kind of terminal expression
		common.PARSER_RULE_CONTEXT_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_NAMESPACE_PREFIX,
		common.PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM,
		common.PARSER_RULE_CONTEXT_METHOD_NAME,
		common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME,
		common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME,
		common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME,
		common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME,
		common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME,
		common.PARSER_RULE_CONTEXT_WORKER_NAME,
		common.PARSER_RULE_CONTEXT_NAMED_WORKERS,
		common.PARSER_RULE_CONTEXT_ANNOTATION_TAG,
		common.PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE,
		common.PARSER_RULE_CONTEXT_MODULE_ENUM_NAME,
		common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME,
		common.PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS,
		common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT,
		common.PARSER_RULE_CONTEXT_EXPRESSION,
		common.PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION,
		common.PARSER_RULE_CONTEXT_XML_NAME,
		common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER,
		common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME,
		common.PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN,
		common.PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT,
		common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR,
		common.PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN:
		return common.IDENTIFIER_TOKEN
	case common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN,
		common.PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS:
		return common.DECIMAL_INTEGER_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN:
		return common.STRING_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR:
		return common.OPTIONAL_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
		return common.ARRAY_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN:
		return common.HEX_INTEGER_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS:
		return common.SEMICOLON_TOKEN
	case common.PARSER_RULE_CONTEXT_DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
		return common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_HEX_FLOATING_POINT_LITERAL_TOKEN:
		return common.HEX_FLOATING_POINT_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		return common.CLOSE_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_NIL_LITERAL:
		return common.OPEN_PAREN_TOKEN
	default:
		return b.getExpectedSeperatorTokenKind(ctx)
	}
}

func (b *BallerinaParserErrorHandler) getExpectedSeperatorTokenKind(ctx common.ParserRuleContext) common.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR:
		return common.BITWISE_AND_TOKEN
	case common.PARSER_RULE_CONTEXT_EQUAL_OR_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_ASSIGN_OP:
		return common.EQUAL_TOKEN
	case common.PARSER_RULE_CONTEXT_EOF:
		return common.EOF_TOKEN
	case common.PARSER_RULE_CONTEXT_BINARY_OPERATOR:
		return common.PLUS_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSE_BRACE:
		return common.CLOSE_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN:
		return common.CLOSE_PAREN_TOKEN
	case common.PARSER_RULE_CONTEXT_COMMA,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA:
		return common.COMMA_TOKEN
	case common.PARSER_RULE_CONTEXT_OPEN_BRACE:
		return common.OPEN_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN,
		common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START:
		return common.OPEN_PAREN_TOKEN
	case common.PARSER_RULE_CONTEXT_SEMICOLON:
		return common.SEMICOLON_TOKEN
	case common.PARSER_RULE_CONTEXT_ASTERISK:
		return common.ASTERISK_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END:
		return common.CLOSE_BRACE_PIPE_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START:
		return common.OPEN_BRACE_PIPE_TOKEN
	case common.PARSER_RULE_CONTEXT_ELLIPSIS:
		return common.ELLIPSIS_TOKEN
	case common.PARSER_RULE_CONTEXT_QUESTION_MARK:
		return common.QUESTION_MARK_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSE_BRACKET:
		return common.CLOSE_BRACKET_TOKEN
	case common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_METHOD_CALL_DOT:
		return common.DOT_TOKEN
	case common.PARSER_RULE_CONTEXT_OPEN_BRACKET, common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START:
		return common.OPEN_BRACKET_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH,
		common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH,
		common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN:
		return common.SLASH_TOKEN
	case common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_TYPE_REF_COLON, common.PARSER_RULE_CONTEXT_VAR_REF_COLON:
		return common.COLON_TOKEN
	case common.PARSER_RULE_CONTEXT_UNARY_OPERATOR,
		common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR,
		common.PARSER_RULE_CONTEXT_UNARY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_EXPRESSION_RHS:
		return common.PLUS_TOKEN
	case common.PARSER_RULE_CONTEXT_AT:
		return common.AT_TOKEN
	case common.PARSER_RULE_CONTEXT_RIGHT_ARROW:
		return common.RIGHT_ARROW_TOKEN
	case common.PARSER_RULE_CONTEXT_GT, common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT:
		return common.GT_TOKEN
	case common.PARSER_RULE_CONTEXT_LT,
		common.PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN,
		common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT:
		return common.LT_TOKEN
	case common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN:
		return common.SYNC_SEND_TOKEN
	case common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN:
		return common.ANNOT_CHAINING_TOKEN
	case common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN:
		return common.OPTIONAL_CHAINING_TOKEN
	case common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN:
		return common.DOT_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN:
		return common.SLASH_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
		return common.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN:
		return common.SLASH_ASTERISK_TOKEN
	case common.PARSER_RULE_CONTEXT_PLUS_TOKEN:
		return common.PLUS_TOKEN
	case common.PARSER_RULE_CONTEXT_MINUS_TOKEN:
		return common.MINUS_TOKEN
	case common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN:
		return common.LEFT_ARROW_TOKEN
	case common.PARSER_RULE_CONTEXT_TEMPLATE_END, common.PARSER_RULE_CONTEXT_TEMPLATE_START:
		return common.BACKTICK_TOKEN
	case common.PARSER_RULE_CONTEXT_LT_TOKEN:
		return common.LT_TOKEN
	case common.PARSER_RULE_CONTEXT_GT_TOKEN:
		return common.GT_TOKEN
	case common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN:
		return common.INTERPOLATION_START_TOKEN
	case common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START,
		common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW:
		return common.RIGHT_DOUBLE_ARROW_TOKEN
	default:
		return b.getExpectedKeywordKind(ctx)
	}
}

func (b *BallerinaParserErrorHandler) getExpectedKeywordKind(ctx common.ParserRuleContext) common.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD:
		return common.EXTERNAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD,
		common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT,
		common.PARSER_RULE_CONTEXT_FUNCTION_IDENT,
		common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER,
		common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND:
		return common.FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD:
		return common.RETURNS_KEYWORD
	case common.PARSER_RULE_CONTEXT_PUBLIC_KEYWORD:
		return common.PUBLIC_KEYWORD
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_RECORD_KEYWORD,
		common.PARSER_RULE_CONTEXT_RECORD_IDENT:
		return common.RECORD_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPE_KEYWORD,
		common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT:
		return common.TYPE_KEYWORD
	case common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD,
		common.PARSER_RULE_CONTEXT_OBJECT_IDENT,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR:
		return common.OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_PRIVATE_KEYWORD:
		return common.PRIVATE_KEYWORD
	case common.PARSER_RULE_CONTEXT_REMOTE_IDENT:
		return common.REMOTE_KEYWORD
	case common.PARSER_RULE_CONTEXT_ABSTRACT_KEYWORD:
		return common.ABSTRACT_KEYWORD
	case common.PARSER_RULE_CONTEXT_CLIENT_KEYWORD:
		return common.CLIENT_KEYWORD
	case common.PARSER_RULE_CONTEXT_IF_KEYWORD:
		return common.IF_KEYWORD
	case common.PARSER_RULE_CONTEXT_ELSE_KEYWORD:
		return common.ELSE_KEYWORD
	case common.PARSER_RULE_CONTEXT_WHILE_KEYWORD:
		return common.WHILE_KEYWORD
	case common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD:
		return common.CHECK_KEYWORD
	case common.PARSER_RULE_CONTEXT_FAIL_KEYWORD:
		return common.FAIL_KEYWORD
	case common.PARSER_RULE_CONTEXT_AS_KEYWORD:
		return common.AS_KEYWORD
	case common.PARSER_RULE_CONTEXT_BOOLEAN_LITERAL:
		return common.TRUE_KEYWORD
	case common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD:
		return common.IMPORT_KEYWORD
	case common.PARSER_RULE_CONTEXT_ON_KEYWORD:
		return common.ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_PANIC_KEYWORD:
		return common.PANIC_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETURN_KEYWORD:
		return common.RETURN_KEYWORD
	case common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD, common.PARSER_RULE_CONTEXT_SERVICE_IDENT:
		return common.SERVICE_KEYWORD
	case common.PARSER_RULE_CONTEXT_BREAK_KEYWORD:
		return common.BREAK_KEYWORD
	case common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD:
		return common.LISTENER_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD:
		return common.CONTINUE_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONST_KEYWORD:
		return common.CONST_KEYWORD
	case common.PARSER_RULE_CONTEXT_FINAL_KEYWORD:
		return common.FINAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_IS_KEYWORD:
		return common.IS_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD:
		return common.TYPEOF_KEYWORD
	case common.PARSER_RULE_CONTEXT_MAP_KEYWORD, common.PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR:
		return common.MAP_KEYWORD
	case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE,
		common.PARSER_RULE_CONTEXT_ERROR_KEYWORD,
		common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return common.ERROR_KEYWORD
	case common.PARSER_RULE_CONTEXT_NULL_KEYWORD:
		return common.NULL_KEYWORD
	case common.PARSER_RULE_CONTEXT_LOCK_KEYWORD:
		return common.LOCK_KEYWORD
	case common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD:
		return common.ANNOTATION_KEYWORD
	case common.PARSER_RULE_CONTEXT_FIELD_IDENT:
		return common.FIELD_KEYWORD
	case common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD,
		common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION:
		return common.XMLNS_KEYWORD
	case common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD:
		return common.SOURCE_KEYWORD
	case common.PARSER_RULE_CONTEXT_START_KEYWORD:
		return common.START_KEYWORD
	case common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD:
		return common.FLUSH_KEYWORD
	case common.PARSER_RULE_CONTEXT_WAIT_KEYWORD:
		return common.WAIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD:
		return common.TRANSACTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD:
		return common.TRANSACTIONAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD:
		return common.COMMIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETRY_KEYWORD:
		return common.RETRY_KEYWORD
	case common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD:
		return common.ROLLBACK_KEYWORD
	case common.PARSER_RULE_CONTEXT_ENUM_KEYWORD:
		return common.ENUM_KEYWORD
	case common.PARSER_RULE_CONTEXT_MATCH_KEYWORD:
		return common.MATCH_KEYWORD
	case common.PARSER_RULE_CONTEXT_NEW_KEYWORD:
		return common.NEW_KEYWORD
	case common.PARSER_RULE_CONTEXT_FORK_KEYWORD:
		return common.FORK_KEYWORD
	case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL, common.PARSER_RULE_CONTEXT_WORKER_KEYWORD:
		return common.WORKER_KEYWORD
	case common.PARSER_RULE_CONTEXT_TRAP_KEYWORD:
		return common.TRAP_KEYWORD
	case common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD:
		return common.FOREACH_KEYWORD
	case common.PARSER_RULE_CONTEXT_IN_KEYWORD:
		return common.IN_KEYWORD
	case common.PARSER_RULE_CONTEXT_PIPE, common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN:
		return common.PIPE_TOKEN
	case common.PARSER_RULE_CONTEXT_TABLE_KEYWORD:
		return common.TABLE_KEYWORD
	case common.PARSER_RULE_CONTEXT_KEY_KEYWORD:
		return common.KEY_KEYWORD
	case common.PARSER_RULE_CONTEXT_STREAM_KEYWORD:
		return common.STREAM_KEYWORD
	case common.PARSER_RULE_CONTEXT_LET_KEYWORD:
		return common.LET_KEYWORD
	case common.PARSER_RULE_CONTEXT_XML_KEYWORD:
		return common.XML_KEYWORD
	case common.PARSER_RULE_CONTEXT_RE_KEYWORD:
		return common.RE_KEYWORD
	case common.PARSER_RULE_CONTEXT_STRING_KEYWORD:
		return common.STRING_KEYWORD
	case common.PARSER_RULE_CONTEXT_BASE16_KEYWORD:
		return common.BASE16_KEYWORD
	case common.PARSER_RULE_CONTEXT_BASE64_KEYWORD:
		return common.BASE64_KEYWORD
	case common.PARSER_RULE_CONTEXT_SELECT_KEYWORD:
		return common.SELECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_WHERE_KEYWORD:
		return common.WHERE_KEYWORD
	case common.PARSER_RULE_CONTEXT_FROM_KEYWORD:
		return common.FROM_KEYWORD
	case common.PARSER_RULE_CONTEXT_ORDER_KEYWORD:
		return common.ORDER_KEYWORD
	case common.PARSER_RULE_CONTEXT_GROUP_KEYWORD:
		return common.GROUP_KEYWORD
	case common.PARSER_RULE_CONTEXT_BY_KEYWORD:
		return common.BY_KEYWORD
	case common.PARSER_RULE_CONTEXT_ORDER_DIRECTION:
		return common.ASCENDING_KEYWORD
	case common.PARSER_RULE_CONTEXT_DO_KEYWORD:
		return common.DO_KEYWORD
	case common.PARSER_RULE_CONTEXT_DISTINCT_KEYWORD:
		return common.DISTINCT_KEYWORD
	case common.PARSER_RULE_CONTEXT_VAR_KEYWORD:
		return common.VAR_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD:
		return common.CONFLICT_KEYWORD
	case common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD:
		return common.LIMIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD:
		return common.EQUALS_KEYWORD
	case common.PARSER_RULE_CONTEXT_JOIN_KEYWORD:
		return common.JOIN_KEYWORD
	case common.PARSER_RULE_CONTEXT_OUTER_KEYWORD:
		return common.OUTER_KEYWORD
	case common.PARSER_RULE_CONTEXT_CLASS_KEYWORD:
		return common.CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD:
		return common.COLLECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
		return common.NATURAL_KEYWORD
	default:
		return b.getExpectedQualifierKind(ctx)
	}
}

func (b *BallerinaParserErrorHandler) getExpectedQualifierKind(ctx common.ParserRuleContext) common.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER:
		return common.OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER:
		return common.CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER:
		return common.FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL,
		common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL:
		return common.IDENTIFIER_TOKEN
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER:
		return common.SERVICE_KEYWORD
	default:
		return common.NONE
	}
}

func (b *BallerinaParserErrorHandler) isBasicLiteral(kind common.SyntaxKind) bool {
	switch kind {
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.STRING_LITERAL_TOKEN,
		common.TRUE_KEYWORD,
		common.FALSE_KEYWORD,
		common.NULL_KEYWORD,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) isUnaryOperator(token tree.STToken) bool {
	switch token.Kind() {
	case common.PLUS_TOKEN,
		common.MINUS_TOKEN,
		common.NEGATION_TOKEN,
		common.EXCLAMATION_MARK_TOKEN:
		return true
	default:
		return false
	}
}

func (b *BallerinaParserErrorHandler) isSingleKeywordAttachPointIdent(tokenKind common.SyntaxKind) bool {
	switch tokenKind {
	case common.ANNOTATION_KEYWORD,
		common.EXTERNAL_KEYWORD,
		common.VAR_KEYWORD,
		common.CONST_KEYWORD,
		common.LISTENER_KEYWORD,
		common.WORKER_KEYWORD,
		common.TYPE_KEYWORD,
		common.FUNCTION_KEYWORD,
		common.PARAMETER_KEYWORD,
		common.RETURN_KEYWORD,
		common.FIELD_KEYWORD,
		common.CLASS_KEYWORD:
		return true
	default:
		return false
	}
}
