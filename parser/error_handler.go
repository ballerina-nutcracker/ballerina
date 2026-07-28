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
	"fmt"
	debugcommon "github.com/ballerina-nutcracker/ballerina/common"
	"github.com/ballerina-nutcracker/ballerina/parser/common"
	"github.com/ballerina-nutcracker/ballerina/st"
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

func formatSTToken(token st.STToken) string {
	if token == nil {
		return "nil"
	}
	kindStr := token.Kind().StrValue()
	if kindStr == "" {
		// For tokens without StrValue (like IDENTIFIER_TOKEN), use a descriptive name
		switch token.Kind() {
		case st.IDENTIFIER_TOKEN:
			kindStr = "IDENTIFIER_TOKEN"
		case st.STRING_LITERAL_TOKEN:
			kindStr = "STRING_LITERAL_TOKEN"
		case st.DECIMAL_INTEGER_LITERAL_TOKEN:
			kindStr = "DECIMAL_INTEGER_LITERAL_TOKEN"
		case st.HEX_INTEGER_LITERAL_TOKEN:
			kindStr = "HEX_INTEGER_LITERAL_TOKEN"
		case st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			kindStr = "DECIMAL_FLOATING_POINT_LITERAL_TOKEN"
		case st.HEX_FLOATING_POINT_LITERAL_TOKEN:
			kindStr = "HEX_FLOATING_POINT_LITERAL_TOKEN"
		case st.INVALID_TOKEN:
			kindStr = "INVALID_TOKEN"
		default:
			kindStr = fmt.Sprintf("TOKEN_%d", token.Kind().Tag())
		}
	}
	return fmt.Sprintf("%s:%s", kindStr, token.Text())
}

func formatSTNode(node st.STNode) string {
	if node == nil {
		return "nil"
	}
	return st.ToSexpr(node)
}

func formatSolution(sol *solution) string {
	if sol == nil {
		return "nil"
	}
	actionStr := "UNKNOWN"
	switch sol.Action {
	case actionInsert:
		actionStr = "INSERT"
	case actionRemove:
		actionStr = "REMOVE"
	case actionKeep:
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

func formatResult(result *recoveryResult) string {
	if result == nil {
		return "nil"
	}
	solutionStr := formatSolution(result.solution)
	return fmt.Sprintf("matches:%d removeFixes:%d fixes:%d solution:%s", result.matches, result.removeFixes, len(result.fixes), solutionStr)
}

func formatResultValue(result recoveryResult) string {
	solutionStr := formatSolution(result.solution)
	return fmt.Sprintf("matches:%d removeFixes:%d fixes:%d solution:%s", result.matches, result.removeFixes, len(result.fixes), solutionStr)
}

func traceRecovery(ctx common.ParserRuleContext, messageFn func() string) {
	debugcommon.DebugWriteLazy(debugcommon.DEBUG_ERROR_RECOVERY, messageFn)
}

// ============================================================================
// Solution struct - represents a fix for a parser error
// ============================================================================

type solution struct {
	Ctx           common.ParserRuleContext
	Action        action
	TokenText     string
	TokenKind     st.SyntaxKind
	RecoveredNode st.STNode
	RemovedToken  st.STToken
	Depth         int
}

func newSolution(action action, ctx common.ParserRuleContext, tokenKind st.SyntaxKind, tokenText string) *solution {
	return newSolutionWithDepth(action, ctx, tokenKind, tokenText, -1)
}

func newSolutionWithDepth(action action, ctx common.ParserRuleContext, tokenKind st.SyntaxKind, tokenText string, depth int) *solution {
	return &solution{
		Action:    action,
		Ctx:       ctx,
		TokenText: tokenText,
		TokenKind: tokenKind,
		Depth:     depth,
	}
}

func (s *solution) ToString() string {
	actionStr := "UNKNOWN"
	switch s.Action {
	case actionInsert:
		actionStr = "INSERT"
	case actionRemove:
		actionStr = "REMOVE"
	case actionKeep:
		actionStr = "KEEP"
	}
	return actionStr + "'" + s.TokenText + "'"
}

// ============================================================================
// Result struct - holds results of error recovery attempts
// ============================================================================

type recoveryResult struct {
	matches     int
	removeFixes int
	fixes       []*solution
	solution    *solution
}

func newResult(fixes []*solution, matches int) *recoveryResult {
	return &recoveryResult{
		fixes:   fixes,
		matches: matches,
	}
}

func (r *recoveryResult) peekFix() *solution {
	if len(r.fixes) == 0 {
		return nil
	}
	return r.fixes[len(r.fixes)-1]
}

func (r *recoveryResult) popFix() *solution {
	if len(r.fixes) == 0 {
		return nil
	}

	sol := r.fixes[len(r.fixes)-1]
	r.fixes = r.fixes[:len(r.fixes)-1]

	if sol.Action == actionRemove {
		r.removeFixes--
	}
	return sol
}

func (r *recoveryResult) pushFix(sol *solution) {
	if sol.Action == actionRemove {
		r.removeFixes++
	}
	r.fixes = append(r.fixes, sol)
}

func (r *recoveryResult) fixesSize() int {
	return len(r.fixes)
}

// ============================================================================
// Constants
// ============================================================================

var (
	lookaheadLimit       = 4
	resolutionItterLimit = 7
	completionItterLimit = 15
)

// ============================================================================
// AbstractParserErrorHandlerData - Field access interface
// ============================================================================

type abstractParserErrorHandlerData interface {
	GetTokenReader() *tokenReader
	SetTokenReader(*tokenReader)
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

type abstractParserErrorHandlerBase struct {
	tokenReader        *tokenReader
	ctxStack           []common.ParserRuleContext
	previousTokenIndex int
	itterCount         int
}

func newAbstractParserErrorHandlerBase(tokenReader *tokenReader) *abstractParserErrorHandlerBase {
	return &abstractParserErrorHandlerBase{
		tokenReader:        tokenReader,
		ctxStack:           make([]common.ParserRuleContext, 0),
		previousTokenIndex: -1,
		itterCount:         0,
	}
}

// Getter/setter implementations for AbstractParserErrorHandlerBase

func (b *abstractParserErrorHandlerBase) GetTokenReader() *tokenReader {
	return b.tokenReader
}

func (b *abstractParserErrorHandlerBase) SetTokenReader(tokenReader *tokenReader) {
	b.tokenReader = tokenReader
}

func (b *abstractParserErrorHandlerBase) GetCtxStack() []common.ParserRuleContext {
	return b.ctxStack
}

func (b *abstractParserErrorHandlerBase) SetCtxStack(ctxStack []common.ParserRuleContext) {
	b.ctxStack = ctxStack
}

func (b *abstractParserErrorHandlerBase) GetPreviousTokenIndex() int {
	return b.previousTokenIndex
}

func (b *abstractParserErrorHandlerBase) SetPreviousTokenIndex(previousTokenIndex int) {
	b.previousTokenIndex = previousTokenIndex
}

func (b *abstractParserErrorHandlerBase) GetItterCount() int {
	return b.itterCount
}

func (b *abstractParserErrorHandlerBase) SetItterCount(itterCount int) {
	b.itterCount = itterCount
}

// ============================================================================
// AbstractParserErrorHandler - Main interface
// ============================================================================

type abstractParserTracer any

type abstractParserErrorHandler interface {
	abstractParserErrorHandlerData
	abstractParserTracer

	// Abstract methods (to be implemented by concrete classes like BallerinaParserErrorHandler)
	HasAlternativePaths(context common.ParserRuleContext) bool
	SeekMatch(context common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) *recoveryResult
	GetNextRule(context common.ParserRuleContext, nextLookahead int) common.ParserRuleContext
	GetExpectedTokenKind(context common.ParserRuleContext) st.SyntaxKind
	GetInsertSolution(context common.ParserRuleContext) *solution

	// Default/concrete methods (implemented in AbstractParserErrorHandlerMethods)
	Recover(currentCtx common.ParserRuleContext, nextToken st.STToken, isCompletion bool) *solution
	ConsumeInvalidToken() st.STToken
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

type abstractParserErrorHandlerMethods struct {
	Self abstractParserErrorHandler
}

func (m *abstractParserErrorHandlerMethods) Recover(currentCtx common.ParserRuleContext, nextToken st.STToken, isCompletion bool) (result *solution) {
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
	var fix *solution
	if isCompletion && (m.Self.GetItterCount() < completionItterLimit) {
		fix = m.getCompletion(currentCtx, nextToken)
	} else if m.Self.GetItterCount() < resolutionItterLimit {
		fix = m.getResolution(currentCtx, nextToken)
	}
	if fix != nil {
		m.applyFix(currentCtx, fix)
		return fix
	}
	// Fail safe. This means we can't find a path to recover.
	if isCompletion {
		if m.Self.GetItterCount() == completionItterLimit {
			traceRecovery(currentCtx, func() string {
				return "fail safe reached"
			})
		}
	} else {
		if m.Self.GetItterCount() == resolutionItterLimit {
			traceRecovery(currentCtx, func() string {
				return "fail safe reached"
			})
		}
	}
	return m.getFailSafeSolution(currentCtx, nextToken)
}

func (m *abstractParserErrorHandlerMethods) getResolution(currentCtx common.ParserRuleContext, nextToken st.STToken) *solution {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(getResolution start %s %s)",
			formatParserRuleContext(currentCtx),
			formatSTToken(nextToken))
	})
	bestMatch := m.seekMatchStart(currentCtx)
	m.validateSolution(bestMatch, currentCtx, nextToken)
	var sol *solution
	if bestMatch.matches > 0 {
		sol = bestMatch.solution
	}
	return sol
}

func (m *abstractParserErrorHandlerMethods) getFailSafeSolution(currentCtx common.ParserRuleContext, nextToken st.STToken) *solution {
	sol := newSolution(actionRemove, currentCtx, nextToken.Kind(), nextToken.Text())
	sol.RemovedToken = m.Self.ConsumeInvalidToken()
	return sol
}

func (m *abstractParserErrorHandlerMethods) validateSolution(bestMatch *recoveryResult, currentCtx common.ParserRuleContext, nextToken st.STNode) {
	sol := bestMatch.solution
	if (sol == nil) || (sol.Action == actionRemove) {
		return
	}
	if (sol.Action == actionKeep) && (nextToken.Kind() == st.DOCUMENTATION_STRING) {
		bestMatch.solution = newSolution(actionRemove, currentCtx, st.DOCUMENTATION_STRING, currentCtx.String())
	}
	if (sol.Action != actionInsert) || (bestMatch.fixesSize() < 2) {
		return
	}
	firstFix := bestMatch.popFix()
	secondFix := bestMatch.peekFix()
	bestMatch.pushFix(firstFix)
	if (secondFix.Action == actionRemove) && (secondFix.Depth == 1) {
		bestMatch.solution = secondFix
	}
}

func (m *abstractParserErrorHandlerMethods) getCompletion(context common.ParserRuleContext, nextToken st.STToken) *solution {
	tempCtxStack := m.Self.GetCtxStack()
	m.Self.SetCtxStack(m.getCtxStackSnapshot())
	var sol *solution
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

func (m *abstractParserErrorHandlerMethods) ConsumeInvalidToken() (result st.STToken) {
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

func (m *abstractParserErrorHandlerMethods) applyFix(currentCtx common.ParserRuleContext, fix *solution) {
	switch fix.Action {
	case actionRemove:
		fix.RemovedToken = m.Self.ConsumeInvalidToken()
		fix.RecoveredNode = m.Self.GetTokenReader().Peek()
		fix.TokenKind = m.Self.GetTokenReader().Peek().Kind()
	case actionInsert:
		fix.RecoveredNode = m.handleMissingToken(currentCtx, fix)
	}
}

func (m *abstractParserErrorHandlerMethods) handleMissingToken(currentCtx common.ParserRuleContext, fix *solution) st.STNode {
	return createMissingTokenWithDiagnosticsFromParserRules(fix.TokenKind, fix.Ctx)
}

func (m *abstractParserErrorHandlerMethods) getCtxStackSnapshot() []common.ParserRuleContext {
	ctxStack := m.Self.GetCtxStack()
	snapshot := make([]common.ParserRuleContext, len(ctxStack))
	copy(snapshot, ctxStack)
	return snapshot
}

func (m *abstractParserErrorHandlerMethods) seekMatchStart(currentCtx common.ParserRuleContext) (bestMatch *recoveryResult) {
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
				bestMatch = newResult(make([]*solution, 0), lookaheadLimit-1)
				bestMatch.solution = newSolution(actionRemove, currentCtx, st.SyntaxKind(0), currentCtx.String())
			}
		}()
		bestMatch = m.seekMatchInSubTree(currentCtx, 1, 0, true)
	}()
	m.Self.SetCtxStack(tempCtxStack)

	return bestMatch
}

func (m *abstractParserErrorHandlerMethods) seekMatchInSubTree(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) (result *recoveryResult) {
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

func (m *abstractParserErrorHandlerMethods) StartContext(context common.ParserRuleContext) {
	traceRecovery(context, func() string {
		return fmt.Sprintf("(StartContext start %s)", formatParserRuleContext(context))
	})
	ctxStack := m.Self.GetCtxStack()
	m.Self.SetCtxStack(append(ctxStack, context))
	traceRecovery(context, func() string {
		return fmt.Sprintf("(StartContext end (%s))", formatParserRuleContext(context))
	})
}

func (m *abstractParserErrorHandlerMethods) EndContext() {
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

func (m *abstractParserErrorHandlerMethods) SwitchContext(context common.ParserRuleContext) {
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

func (m *abstractParserErrorHandlerMethods) GetParentContext() (result common.ParserRuleContext) {
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

func (m *abstractParserErrorHandlerMethods) GetGrandParentContext() (result common.ParserRuleContext) {
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

func (m *abstractParserErrorHandlerMethods) HasAncestorContext(context common.ParserRuleContext) (result bool) {
	traceRecovery(context, func() string {
		return fmt.Sprintf("(HasAncestorContext start %s)", formatParserRuleContext(context))
	})
	defer traceRecovery(context, func() string {
		return fmt.Sprintf("(HasAncestorContext end (%s) %s)", formatParserRuleContext(context), formatBool(result))
	})
	ctxStack := m.Self.GetCtxStack()
	return slices.Contains(ctxStack, context)
}

func (m *abstractParserErrorHandlerMethods) GetContextStack() (result []common.ParserRuleContext) {
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

func (m *abstractParserErrorHandlerMethods) seekInAlternativesPaths(lookahead int, currentDepth int, currentMatches int, alternativeRules []common.ParserRuleContext, isEntryPoint bool) (result *recoveryResult) {
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
	results := make([][]*recoveryResult, lookaheadLimit)
	bestMatchIndex := 0

	for _, rule := range alternativeRules {
		tempCtxStack := m.Self.GetCtxStack()
		var result *recoveryResult
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
			similarResults = make([]*recoveryResult, 0)
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
			if (currentSol.Action == actionRemove) && (foundSol.Action == actionInsert) {
				bestMatch = currentMatch
			}
		} else if currentMatchRemoveFixes < bestMatchRemoveFixes {
			bestMatch = currentMatch
		}
	}
	return m.getFinalResult(currentMatches, bestMatch)
}

func (m *abstractParserErrorHandlerMethods) hasFoundBestAlternative(result *recoveryResult) bool {
	if result.matches < (lookaheadLimit - 1) {
		return false
	}
	if result.solution == nil {
		return true
	}
	return (result.solution.Action != actionRemove)
}

func (m *abstractParserErrorHandlerMethods) getFinalResult(currentMatches int, bestMatch *recoveryResult) (result *recoveryResult) {
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

func (m *abstractParserErrorHandlerMethods) fixAndContinue(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result *recoveryResult) {
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
		fixedPathResult.solution = newSolution(actionKeep, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String())
	}
	return m.getFinalResult(matchingRulesCount, fixedPathResult)
}

func (m *abstractParserErrorHandlerMethods) fixAndContinueCore(currentCtx common.ParserRuleContext, lookahead int, currentDepth int) (fixedPathResult *recoveryResult) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(fixAndContinueCore start %s %d %d)", formatParserRuleContext(currentCtx), lookahead, currentDepth)
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(fixAndContinueCore end (%s %d %d) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatResult(fixedPathResult))
	})
	deletionResult := m.seekMatchInSubTree(currentCtx, lookahead+1, currentDepth+1, false)
	nextCtx := m.Self.GetNextRule(currentCtx, lookahead)
	insertionResult := m.seekMatchInSubTree(nextCtx, lookahead, currentDepth+1, false)
	var action *solution

	if (insertionResult.matches == 0) && (deletionResult.matches == 0) {
		action = newSolutionWithDepth(actionInsert, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String(), currentDepth)
		insertionResult.pushFix(action)
		fixedPathResult = insertionResult
	} else if insertionResult.matches == deletionResult.matches {
		if insertionResult.removeFixes <= (deletionResult.removeFixes + 1) {
			action = newSolutionWithDepth(actionInsert, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String(), currentDepth)
			insertionResult.pushFix(action)
			fixedPathResult = insertionResult
		} else {
			token := m.Self.GetTokenReader().PeekN(lookahead)
			action = newSolutionWithDepth(actionRemove, currentCtx, token.Kind(), token.Text(), currentDepth)
			deletionResult.pushFix(action)
			fixedPathResult = deletionResult
		}
	} else if insertionResult.matches > deletionResult.matches {
		action = newSolutionWithDepth(actionInsert, currentCtx, m.Self.GetExpectedTokenKind(currentCtx), currentCtx.String(), currentDepth)
		insertionResult.pushFix(action)
		fixedPathResult = insertionResult
	} else {
		token := m.Self.GetTokenReader().PeekN(lookahead)
		action = newSolutionWithDepth(actionRemove, currentCtx, token.Kind(), token.Text(), currentDepth)
		deletionResult.pushFix(action)
		fixedPathResult = deletionResult
	}
	return fixedPathResult
}

type ballerinaParserErrorHandler struct {
	abstractParserErrorHandlerBase
	abstractParserErrorHandlerMethods
}

var (
	funcTypeOrDefOptionalReturns                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS}
	funcBodyOrTypeDescRhs                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_BODY, common.PARSER_RULE_CONTEXT_MODULE_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS}
	funcDefOptionalReturns                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_BODY}
	methodDeclOptionalReturns                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	funcBody                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK, common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY}
	externalFuncBodyOptionalAnnots                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATIONS, common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD}
	annonFuncOptionalReturns                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY}
	anonFuncBody                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK, common.PARSER_RULE_CONTEXT_EXPLICIT_ANON_FUNC_EXPR_BODY_START}
	funcTypeOptionalReturns                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END}
	funcTypeOrAnonFuncOptionalReturns              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY}
	funcTypeDescRhsOrAnonFuncBody                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY, common.PARSER_RULE_CONTEXT_STMT_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS}
	workerNameRhs                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD, common.PARSER_RULE_CONTEXT_BLOCK_STMT}
	statements                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT, common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_IF_BLOCK, common.PARSER_RULE_CONTEXT_WHILE_BLOCK, common.PARSER_RULE_CONTEXT_CALL_STMT, common.PARSER_RULE_CONTEXT_PANIC_STMT, common.PARSER_RULE_CONTEXT_CONTINUE_STATEMENT, common.PARSER_RULE_CONTEXT_BREAK_STATEMENT, common.PARSER_RULE_CONTEXT_RETURN_STMT, common.PARSER_RULE_CONTEXT_MATCH_STMT, common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT, common.PARSER_RULE_CONTEXT_LOCK_STMT, common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL, common.PARSER_RULE_CONTEXT_FORK_STMT, common.PARSER_RULE_CONTEXT_FOREACH_STMT, common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION, common.PARSER_RULE_CONTEXT_TRANSACTION_STMT, common.PARSER_RULE_CONTEXT_RETRY_STMT, common.PARSER_RULE_CONTEXT_ROLLBACK_STMT, common.PARSER_RULE_CONTEXT_DO_BLOCK, common.PARSER_RULE_CONTEXT_FAIL_STATEMENT, common.PARSER_RULE_CONTEXT_BLOCK_STMT}
	assignmentStmtRhs                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR}
	varDeclRhs                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_SEMICOLON}
	topLevelNode                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EOF, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	topLevelNodeWithoutMetadata                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EOF, common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER, common.PARSER_RULE_CONTEXT_PUBLIC_KEYWORD}
	topLevelNodeWithoutModifier                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EOF, common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL, common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION, common.PARSER_RULE_CONTEXT_SERVICE_DECL, common.PARSER_RULE_CONTEXT_LISTENER_DECL, common.PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION, common.PARSER_RULE_CONTEXT_CONSTANT_DECL, common.PARSER_RULE_CONTEXT_ANNOTATION_DECL, common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION, common.PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION, common.PARSER_RULE_CONTEXT_IMPORT_DECL}
	funcDefStart                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER}
	funcDefWithoutFirstQualifier                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER}
	typeOrVarName                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN}
	fieldDescriptorRhs                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_QUESTION_MARK, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	fieldOrRestDesciptorRhs                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	recordBodyStart                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START, common.PARSER_RULE_CONTEXT_OPEN_BRACE}
	recordBodyEnd                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	typeDescriptors                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE, common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE, common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START, common.PARSER_RULE_CONTEXT_STREAM_KEYWORD, common.PARSER_RULE_CONTEXT_TABLE_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC, common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION, common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START}
	typeDescriptorWithoutIsolated                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR}
	classDescriptor                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_REFERENCE, common.PARSER_RULE_CONTEXT_STREAM_KEYWORD}
	recordFieldOrRecordEnd                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RECORD_BODY_END, common.PARSER_RULE_CONTEXT_RECORD_FIELD}
	recordFieldStart                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD, common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	recordFieldWithoutMetadata                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD}
	argStartOrArgListEnd                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_END, common.PARSER_RULE_CONTEXT_ARG_START}
	argStart                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_ELLIPSIS, common.PARSER_RULE_CONTEXT_EXPRESSION}
	argEnd                                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_END, common.PARSER_RULE_CONTEXT_COMMA}
	namedOrPositionalArgRhs                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_END, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	optionalFieldInitializer                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_SEMICOLON}
	onFailOptionalBindingPattern                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BLOCK_STMT, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN}
	groupingKeyListElement                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY}
	groupingKeyListElementEnd                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE_END, common.PARSER_RULE_CONTEXT_COMMA}
	classMemberOrObjectMemberStart                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	objectConstructorMemberStart                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	classMemberOrObjectMemberWithoutMeta           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	objectConsMemberWithoutMeta                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	objectFuncOrField                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY, common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL}
	objectFuncOrFieldWithoutVisibility             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_FIELD_START, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_START}
	objectFieldQualifier                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER, common.PARSER_RULE_CONTEXT_FINAL_KEYWORD}
	objectMethodStart                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER}
	objectMethodWithoutFirstQualifier              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER}
	objectMethodWithoutSecondQualifier             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER}
	objectMethodWithoutThirdQualifier              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_DEF, common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER}
	objectTypeStart                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER}
	objectTypeWithoutFirstQualifier                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER}
	objectConstructorStart                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER}
	objectConsWithoutFirstQualifier                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD, common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER}
	objectConstructorRhs                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_BRACE, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	elseBody                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_IF_BLOCK, common.PARSER_RULE_CONTEXT_OPEN_BRACE}
	elseBlock                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ELSE_KEYWORD, common.PARSER_RULE_CONTEXT_STATEMENT}
	callStatement                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD, common.PARSER_RULE_CONTEXT_VARIABLE_REF, common.PARSER_RULE_CONTEXT_EXPRESSION}
	importPrefixDecl                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	importDeclOrgOrModuleNameRhs                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME}
	afterImportModuleName                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_SEMICOLON}
	returnRhs                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_EXPRESSION}
	expressionStart                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BASIC_LITERAL, common.PARSER_RULE_CONTEXT_NIL_LITERAL, common.PARSER_RULE_CONTEXT_VARIABLE_REF, common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION, common.PARSER_RULE_CONTEXT_TYPE_CAST, common.PARSER_RULE_CONTEXT_BRACED_EXPRESSION, common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_LET_EXPRESSION, common.PARSER_RULE_CONTEXT_TEMPLATE_START, common.PARSER_RULE_CONTEXT_XML_KEYWORD, common.PARSER_RULE_CONTEXT_STRING_KEYWORD, common.PARSER_RULE_CONTEXT_BASE16_KEYWORD, common.PARSER_RULE_CONTEXT_BASE64_KEYWORD, common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION, common.PARSER_RULE_CONTEXT_ERROR_KEYWORD, common.PARSER_RULE_CONTEXT_NEW_KEYWORD, common.PARSER_RULE_CONTEXT_START_KEYWORD, common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD, common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN, common.PARSER_RULE_CONTEXT_WAIT_KEYWORD, common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD, common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD, common.PARSER_RULE_CONTEXT_TYPEOF_EXPRESSION, common.PARSER_RULE_CONTEXT_TRAP_KEYWORD, common.PARSER_RULE_CONTEXT_UNARY_EXPRESSION, common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD, common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_RE_KEYWORD, common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION}
	firstMappingFieldStart                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MAPPING_FIELD, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	mappingFieldStart                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD, common.PARSER_RULE_CONTEXT_ELLIPSIS, common.PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME, common.PARSER_RULE_CONTEXT_READONLY_KEYWORD}
	specificField                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME, common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN}
	specificFieldRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END}
	mappingFieldEnd                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA}
	constDeclRhs                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	arrayLength                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN, common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN, common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_VARIABLE_REF}
	paramList                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_REQUIRED_PARAM}
	parameterStart                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	parameterStartWithoutAnnotation                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM, common.PARSER_RULE_CONTEXT_ASTERISK}
	requiredParamNameRhs                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PARAM_END, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	paramEnd                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	stmtStartWithExprRhs                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR}
	exprStmtRhs                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR}
	expressionStatementStart                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD, common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_START_KEYWORD, common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD}
	annotDeclOptionalType                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_TAG, common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER}
	constDeclType                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER, common.PARSER_RULE_CONTEXT_VARIABLE_NAME}
	annotDeclRhs                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_TAG, common.PARSER_RULE_CONTEXT_ON_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	annotOptionalAttachPoints                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ON_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	attachPoint                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD, common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT}
	attachPointIdent                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT, common.PARSER_RULE_CONTEXT_OBJECT_IDENT, common.PARSER_RULE_CONTEXT_SERVICE_IDENT, common.PARSER_RULE_CONTEXT_RECORD_IDENT}
	serviceIdentRhs                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_REMOTE_IDENT, common.PARSER_RULE_CONTEXT_ATTACH_POINT_END}
	attachPointEnd                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_SEMICOLON}
	xmlNamespacePrefixDecl                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_AS_KEYWORD, common.PARSER_RULE_CONTEXT_SEMICOLON}
	constantExpression                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BASIC_LITERAL, common.PARSER_RULE_CONTEXT_VARIABLE_REF, common.PARSER_RULE_CONTEXT_PLUS_TOKEN, common.PARSER_RULE_CONTEXT_MINUS_TOKEN, common.PARSER_RULE_CONTEXT_NIL_LITERAL}
	listConstructorFirstMember                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER}
	listConstructorMember                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	typeCastParam                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	typeCastParamRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS, common.PARSER_RULE_CONTEXT_GT}
	tableKeywordRhs                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_KEY_SPECIFIER, common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR}
	rowListRhs                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR}
	tableRowEnd                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	keySpecifierRhs                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_VARIABLE_NAME}
	tableKeyRhs                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	letVarDeclStart                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	streamTypeFirstParamRhs                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_GT}
	templateMember                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TEMPLATE_STRING, common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN, common.PARSER_RULE_CONTEXT_TEMPLATE_END}
	templateStringRhs                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN, common.PARSER_RULE_CONTEXT_TEMPLATE_END}
	keyConstraintsRhs                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_LT}
	functionKeywordRhs                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_NAME, common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS}
	funcTypeFuncKeywordRhsStart                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END, common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS}
	typeDescRhs                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_END_OF_TYPE_DESC, common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_PIPE, common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR}
	tableTypeDescRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_KEY_KEYWORD, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS}
	newKeywordRhs                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS}
	tableConstructorOrQueryStart                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TABLE_KEYWORD, common.PARSER_RULE_CONTEXT_STREAM_KEYWORD, common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION, common.PARSER_RULE_CONTEXT_MAP_KEYWORD}
	tableConstructorOrQueryRhs                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR, common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION}
	queryPipelineRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_RHS, common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE, common.PARSER_RULE_CONTEXT_QUERY_ACTION_RHS}
	intermediateClauseStart                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_WHERE_CLAUSE, common.PARSER_RULE_CONTEXT_FROM_CLAUSE, common.PARSER_RULE_CONTEXT_LET_CLAUSE, common.PARSER_RULE_CONTEXT_JOIN_CLAUSE, common.PARSER_RULE_CONTEXT_ORDER_BY_CLAUSE, common.PARSER_RULE_CONTEXT_LIMIT_CLAUSE, common.PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE}
	resultClause                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SELECT_CLAUSE, common.PARSER_RULE_CONTEXT_COLLECT_CLAUSE}
	bracedExprOrAnonFuncParamRhs                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_COMMA}
	annotationRefRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_END, common.PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR}
	inferParamEndOrParenthesisEnd                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START}
	optionalPeerWorker                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS}
	typeDescInTupleRhs                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	tupleTypeMemberRhs                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA}
	listConstructorMemberEnd                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA}
	nilOrParenthesisedTypeDescRhs                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR}
	bindingPattern                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER, common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN}
	listBindingPatternsStart                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	listBindingPatternContents                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN}
	listBindingPatternMemberEnd                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_COMMA}
	mappingBindingPatternMember                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN}
	mappingBindingPatternEnd                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	fieldBindingPatternEnd                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	errorBindingPatternErrorKeywordRhs             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	errorArgListBindingPatternStart                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	errorMessageBindingPatternEnd                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	errorMessageBindingPatternRhs                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN}
	errorFieldBindingPattern                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_REST_BINDING_PATTERN}
	errorFieldBindingPatternEnd                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_COMMA}
	remoteOrResourceCallOrAsyncSendRhs             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND, common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN, common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME, common.PARSER_RULE_CONTEXT_METHOD_NAME}
	remoteCallOrAsyncSendEnd                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_SEMICOLON}
	receiveWorkers                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER, common.PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS}
	singleOrAlternateWorkerSeparator               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_END, common.PARSER_RULE_CONTEXT_PIPE}
	receiveField                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME, common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME}
	receiveFieldEnd                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA}
	waitKeywordRhs                                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS, common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS}
	waitFieldNameRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_WAIT_FIELD_END}
	waitFieldEnd                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACE, common.PARSER_RULE_CONTEXT_COMMA}
	waitFutureExprEnd                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPR_LIST_END, common.PARSER_RULE_CONTEXT_PIPE}
	enumMemberStart                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME, common.PARSER_RULE_CONTEXT_DOC_STRING, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	enumMemberRhs                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END}
	enumMemberEnd                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	memberAccessKeyExprEnd                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	rollbackRhs                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_EXPRESSION}
	retryKeywordRhs                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LT, common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS}
	retryTypeParamRhs                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_RETRY_BODY}
	retryBody                                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BLOCK_STMT, common.PARSER_RULE_CONTEXT_TRANSACTION_STMT}
	listBpOrTupleTypeMember                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER}
	listBpOrTupleTypeDescRhs                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_VARIABLE_NAME}
	bracketedListMemberEnd                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	bracketedListMember                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_BINDING_PATTERN}
	listBindingMemberOrArrayLength                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_BRACKET, common.PARSER_RULE_CONTEXT_BINDING_PATTERN, common.PARSER_RULE_CONTEXT_ARRAY_LENGTH_START}
	bracketedListRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS, common.PARSER_RULE_CONTEXT_EXPRESSION_RHS}
	bindingPatternOrVarRefRhs                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS, common.PARSER_RULE_CONTEXT_ASSIGN_OP, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS}
	typeDescRhsOrBpRhs                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_IN_TYPED_BP, common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_RHS}
	xmlNavigateExpr                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_XML_FILTER_EXPR, common.PARSER_RULE_CONTEXT_XML_STEP_EXPR}
	xmlNamePatternRhs                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_GT, common.PARSER_RULE_CONTEXT_PIPE}
	xmlAtomicNamePatternStart                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER}
	xmlAtomicNameIdentifierRhs                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ASTERISK, common.PARSER_RULE_CONTEXT_IDENTIFIER}
	xmlStepStart                                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN, common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN, common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN}
	xmlStepExtend                                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND_END, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR}
	xmlStepStartEnd                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION_RHS, common.PARSER_RULE_CONTEXT_XML_STEP_EXTENDS}
	matchPatternListMemberRhs                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MATCH_PATTERN_END, common.PARSER_RULE_CONTEXT_PIPE}
	optionalMatchGuard                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW, common.PARSER_RULE_CONTEXT_IF_KEYWORD}
	matchPatternStart                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION, common.PARSER_RULE_CONTEXT_VAR_KEYWORD, common.PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN}
	listMatchPatternsStart                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	listMatchPatternMember                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	listMatchPatternMemberRhs                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	fieldMatchPatternsStart                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	fieldMatchPatternMember                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	fieldMatchPatternMemberRhs                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_BRACE}
	errorMatchPatternOrConstPattern                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_MATCH_PATTERN_RHS}
	errorMatchPatternErrorKeywordRhs               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	errorArgListMatchPatternStart                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION, common.PARSER_RULE_CONTEXT_VAR_KEYWORD, common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	errorMessageMatchPatternEnd                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	errorMessageMatchPatternRhs                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ERROR_CAUSE_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN}
	errorFieldMatchPattern                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	errorFieldMatchPatternRhs                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COMMA, common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS}
	namedArgMatchPatternRhs                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_REST_MATCH_PATTERN}
	orderKeyListEnd                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ORDER_CLAUSE_END, common.PARSER_RULE_CONTEXT_COMMA}
	listBpOrListConstructorMember                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER}
	tupleTypeDescOrListConstMember                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR, common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER}
	joinClauseStart                                = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_JOIN_KEYWORD, common.PARSER_RULE_CONTEXT_OUTER_KEYWORD}
	mappingBpOrMappingConstructorMember            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER, common.PARSER_RULE_CONTEXT_MAPPING_FIELD}
	listenersListEnd                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_BLOCK, common.PARSER_RULE_CONTEXT_COMMA}
	funcTypeDescStart                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER}
	funcTypeDescStartWithoutFirstQual              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD, common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER}
	moduleClassDefinitionStart                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER}
	classDefWithoutFirstQualifier                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER}
	classDefWithoutSecondQualifier                 = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER}
	classDefWithoutThirdQualifier                  = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLASS_KEYWORD, common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER}
	regularCompoundStmtRhs                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE}
	namedWorkerDeclStart                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_WORKER_KEYWORD, common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD}
	serviceDeclStart                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD, common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER}
	optionalServiceDeclType                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE, common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH}
	optionalAbsolutePath                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH, common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN, common.PARSER_RULE_CONTEXT_ON_KEYWORD}
	absoluteResourcePathStart                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH}
	absoluteResourcePathEnd                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_SERVICE_DECL_RHS}
	serviceDeclOrVarDecl                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH, common.PARSER_RULE_CONTEXT_SERVICE_VAR_DECL_RHS}
	optionalRelativePath                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH}
	funcDefOrTypeDescRhs                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS, common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH, common.PARSER_RULE_CONTEXT_SEMICOLON, common.PARSER_RULE_CONTEXT_ASSIGN_OP}
	relativeResourcePathStart                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT}
	resourcePathSegment                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT, common.PARSER_RULE_CONTEXT_RESOURCE_PATH_PARAM}
	pathParamOptionalAnnots                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	pathParamEllipsis                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	optionalPathParamName                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_CLOSE_BRACKET}
	relativeResourcePathEnd                        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RESOURCE_ACCESSOR_DEF_OR_DECL_RHS, common.PARSER_RULE_CONTEXT_SLASH}
	configVarDeclRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_QUESTION_MARK}
	errorConstructorRhs                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_TYPE_REFERENCE}
	optionalTypeParameter                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_LT, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS}
	mapTypeOrTypeRef                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_LT}
	objectTypeOrTypeRef                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OBJECT_KEYWORD_RHS}
	streamTypeOrTypeRef                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_LT}
	tableTypeOrTypeRef                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_ROW_TYPE_PARAM}
	parameterizedTypeOrTypeRef                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER}
	typeDescRhsOrTypeRef                           = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS}
	transactionStmtRhsOrTypeRef                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_REF_COLON, common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_TRANSACTION_KEYWORD_RHS}
	tableConsOrQueryExprOrVarRef                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_REF_COLON, common.PARSER_RULE_CONTEXT_EXPRESSION_START_TABLE_KEYWORD_RHS}
	queryExprOrVarRef                              = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_REF_COLON, common.PARSER_RULE_CONTEXT_QUERY_CONSTRUCT_TYPE_RHS}
	errorConsExprOrVarRef                          = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_REF_COLON, common.PARSER_RULE_CONTEXT_ERROR_CONS_ERROR_KEYWORD_RHS}
	qualifiedIdentifier                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER, common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX}
	moduleVarDeclStart                             = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL}
	moduleVarWithoutFirstQual                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL}
	moduleVarWithoutSecondQual                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VAR_DECL_STMT, common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL}
	exprStartOrInferredTypedescDefaultStart        = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT}
	typeCastParamStartOrInferredTypedescDefaultEnd = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START, common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT}
	endOfParamsOrNextParamStart                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS, common.PARSER_RULE_CONTEXT_COMMA}
	paramStart                                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM, common.PARSER_RULE_CONTEXT_ANNOTATIONS}
	paramRhs                                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_VARIABLE_NAME, common.PARSER_RULE_CONTEXT_REST_PARAM_RHS}
	funcTypeParamRhs                               = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_PARAM_END, common.PARSER_RULE_CONTEXT_PARAM_RHS}
	annotationDeclStart                            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD, common.PARSER_RULE_CONTEXT_CONST_KEYWORD}
	optionalResourceAccessPath                     = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD}
	resourceAccessPathSegment                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_IDENTIFIER, common.PARSER_RULE_CONTEXT_OPEN_BRACKET}
	computedSegmentOrRestSegment                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_ELLIPSIS}
	resourceAccessSegmentRhs                       = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_SLASH, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD}
	optionalResourceAccessMethod                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST}
	optionalResourceAccessActionArgList            = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_ACTION_END}
	optionalTopLevelSemicolon                      = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE, common.PARSER_RULE_CONTEXT_SEMICOLON}
	tupleMember                                    = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ANNOTATIONS, common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE}
	naturalExpressionStart                         = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD, common.PARSER_RULE_CONTEXT_CONST_KEYWORD}
	optionalParenthesizedArgList                   = []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN, common.PARSER_RULE_CONTEXT_OPEN_BRACE}
)

func newBallerinaParserErrorHandlerFromTokenReader(tokenReader *tokenReader) ballerinaParserErrorHandler {
	this := ballerinaParserErrorHandler{}
	this.abstractParserErrorHandlerBase = *newAbstractParserErrorHandlerBase(tokenReader)
	this.Self = &this
	return this
}

func (b *ballerinaParserErrorHandler) isEndOfObjectTypeNode(nextLookahead int) bool {
	nextToken := b.tokenReader.PeekN(nextLookahead)
	switch nextToken.Kind() {
	case st.CLOSE_BRACE_TOKEN,
		st.EOF_TOKEN,
		st.CLOSE_BRACE_PIPE_TOKEN,
		st.TYPE_KEYWORD,
		st.SERVICE_KEYWORD:
		return true
	default:
		nextNextToken := b.tokenReader.PeekN(nextLookahead + 1)
		switch nextNextToken.Kind() {
		case st.CLOSE_BRACE_TOKEN,
			st.EOF_TOKEN,
			st.CLOSE_BRACE_PIPE_TOKEN,
			st.TYPE_KEYWORD,
			st.SERVICE_KEYWORD:
			return true
		default:
			return false
		}
	}
}

func (b *ballerinaParserErrorHandler) SeekMatch(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, isEntryPoint bool) (result *recoveryResult) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(SeekMatch start %s %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(SeekMatch end (%s %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, formatBool(isEntryPoint), formatResult(result))
	})
	var hasMatch bool
	var skipRule bool
	matchingRulesCount := 0
	for currentDepth < lookaheadLimit {
		skipRule = false
		lookahead = b.getNextLookahead(lookahead)
		nextToken := b.tokenReader.PeekN(lookahead)
		switch currentCtx {
		case common.PARSER_RULE_CONTEXT_EOF:
			hasMatch = (nextToken.Kind() == st.EOF_TOKEN)
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
			hasMatch = (nextToken.Kind() == st.IDENTIFIER_TOKEN)
		case common.PARSER_RULE_CONTEXT_IMPORT_PREFIX:
			hasMatch = ((nextToken.Kind() == st.IDENTIFIER_TOKEN) || isPredeclaredPrefix(nextToken.Kind()))
		case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX:
			hasMatch = isPredeclaredPrefix(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS,
			common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START,
			common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN:
			hasMatch = (nextToken.Kind() == st.OPEN_PAREN_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS,
			common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN:
			hasMatch = (nextToken.Kind() == st.CLOSE_PAREN_TOKEN)
		case common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR:
			hasMatch = (((isSimpleType(nextToken.Kind()) || (nextToken.Kind() == st.ERROR_KEYWORD)) || (nextToken.Kind() == st.STREAM_KEYWORD)) || (nextToken.Kind() == st.TYPEDESC_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OPEN_BRACE:
			hasMatch = (nextToken.Kind() == st.OPEN_BRACE_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSE_BRACE:
			hasMatch = (nextToken.Kind() == st.CLOSE_BRACE_TOKEN)
		case common.PARSER_RULE_CONTEXT_ASSIGN_OP:
			hasMatch = (nextToken.Kind() == st.EQUAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_SEMICOLON:
			hasMatch = (nextToken.Kind() == st.SEMICOLON_TOKEN)
		case common.PARSER_RULE_CONTEXT_BINARY_OPERATOR:
			hasMatch = b.isBinaryOperator(nextToken)
		case common.PARSER_RULE_CONTEXT_COMMA,
			common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA,
			common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA:
			hasMatch = (nextToken.Kind() == st.COMMA_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END:
			hasMatch = (nextToken.Kind() == st.CLOSE_BRACE_PIPE_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START:
			hasMatch = (nextToken.Kind() == st.OPEN_BRACE_PIPE_TOKEN)
		case common.PARSER_RULE_CONTEXT_ELLIPSIS:
			hasMatch = (nextToken.Kind() == st.ELLIPSIS_TOKEN)
		case common.PARSER_RULE_CONTEXT_QUESTION_MARK:
			hasMatch = (nextToken.Kind() == st.QUESTION_MARK_TOKEN)
		case common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER,
			common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER:
			hasMatch = (((nextToken.Kind() == st.CLIENT_KEYWORD) || (nextToken.Kind() == st.ISOLATED_KEYWORD)) || (nextToken.Kind() == st.SERVICE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER:
			hasMatch = (((((nextToken.Kind() == st.DISTINCT_KEYWORD) || (nextToken.Kind() == st.CLIENT_KEYWORD)) || (nextToken.Kind() == st.READONLY_KEYWORD)) || (nextToken.Kind() == st.ISOLATED_KEYWORD)) || (nextToken.Kind() == st.SERVICE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OPEN_BRACKET,
			common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START:
			hasMatch = (nextToken.Kind() == st.OPEN_BRACKET_TOKEN)
		case common.PARSER_RULE_CONTEXT_CLOSE_BRACKET:
			hasMatch = (nextToken.Kind() == st.CLOSE_BRACKET_TOKEN)
		case common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_METHOD_CALL_DOT:
			hasMatch = (nextToken.Kind() == st.DOT_TOKEN)
		case common.PARSER_RULE_CONTEXT_BOOLEAN_LITERAL:
			hasMatch = ((nextToken.Kind() == st.TRUE_KEYWORD) || (nextToken.Kind() == st.FALSE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == st.DECIMAL_INTEGER_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_SLASH,
			common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH,
			common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN:
			hasMatch = (nextToken.Kind() == st.SLASH_TOKEN)
		case common.PARSER_RULE_CONTEXT_BASIC_LITERAL:
			hasMatch = b.isBasicLiteral(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_COLON,
			common.PARSER_RULE_CONTEXT_VAR_REF_COLON,
			common.PARSER_RULE_CONTEXT_TYPE_REF_COLON:
			hasMatch = (nextToken.Kind() == st.COLON_TOKEN)
		case common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == st.STRING_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_UNARY_OPERATOR:
			hasMatch = b.isUnaryOperator(nextToken)
		case common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == st.HEX_INTEGER_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_AT:
			hasMatch = (nextToken.Kind() == st.AT_TOKEN)
		case common.PARSER_RULE_CONTEXT_RIGHT_ARROW:
			hasMatch = (nextToken.Kind() == st.RIGHT_ARROW_TOKEN)
		case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE:
			hasMatch = isParameterizedTypeToken(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_LT,
			common.PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN,
			common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT:
			hasMatch = (nextToken.Kind() == st.LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_GT,
			common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT:
			hasMatch = (nextToken.Kind() == st.GT_TOKEN)
		case common.PARSER_RULE_CONTEXT_FIELD_IDENT:
			hasMatch = (nextToken.Kind() == st.FIELD_KEYWORD)
		case common.PARSER_RULE_CONTEXT_FUNCTION_IDENT:
			hasMatch = (nextToken.Kind() == st.FUNCTION_KEYWORD)
		case common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT:
			hasMatch = ((nextToken.Kind() == st.FUNCTION_KEYWORD) || (nextToken.Kind() == st.FIELD_KEYWORD))
		case common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT:
			hasMatch = b.isSingleKeywordAttachPointIdent(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_OBJECT_IDENT:
			hasMatch = (nextToken.Kind() == st.OBJECT_KEYWORD)
		case common.PARSER_RULE_CONTEXT_RECORD_IDENT:
			hasMatch = (nextToken.Kind() == st.RECORD_KEYWORD)
		case common.PARSER_RULE_CONTEXT_SERVICE_IDENT:
			hasMatch = (nextToken.Kind() == st.SERVICE_KEYWORD)
		case common.PARSER_RULE_CONTEXT_REMOTE_IDENT:
			hasMatch = (nextToken.Kind() == st.REMOTE_KEYWORD)
		case common.PARSER_RULE_CONTEXT_DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_HEX_FLOATING_POINT_LITERAL_TOKEN:
			hasMatch = (nextToken.Kind() == st.HEX_FLOATING_POINT_LITERAL_TOKEN)
		case common.PARSER_RULE_CONTEXT_PIPE:
			hasMatch = (nextToken.Kind() == st.PIPE_TOKEN)
		case common.PARSER_RULE_CONTEXT_TEMPLATE_START, common.PARSER_RULE_CONTEXT_TEMPLATE_END:
			hasMatch = (nextToken.Kind() == st.BACKTICK_TOKEN)
		case common.PARSER_RULE_CONTEXT_ASTERISK:
			hasMatch = (nextToken.Kind() == st.ASTERISK_TOKEN)
		case common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR:
			hasMatch = (nextToken.Kind() == st.BITWISE_AND_TOKEN)
		case common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START,
			common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW:
			hasMatch = (nextToken.Kind() == st.RIGHT_DOUBLE_ARROW_TOKEN)
		case common.PARSER_RULE_CONTEXT_PLUS_TOKEN:
			hasMatch = (nextToken.Kind() == st.PLUS_TOKEN)
		case common.PARSER_RULE_CONTEXT_MINUS_TOKEN:
			hasMatch = (nextToken.Kind() == st.MINUS_TOKEN)
		case common.PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS:
			hasMatch = isIntOrFloat(nextToken)
		case common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN:
			hasMatch = (nextToken.Kind() == st.SYNC_SEND_TOKEN)
		case common.PARSER_RULE_CONTEXT_PEER_WORKER_NAME:
			hasMatch = ((nextToken.Kind() == st.FUNCTION_KEYWORD) || (nextToken.Kind() == st.IDENTIFIER_TOKEN))
		case common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN:
			hasMatch = (nextToken.Kind() == st.LEFT_ARROW_TOKEN)
		case common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN:
			hasMatch = (nextToken.Kind() == st.ANNOT_CHAINING_TOKEN)
		case common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN:
			hasMatch = (nextToken.Kind() == st.OPTIONAL_CHAINING_TOKEN)
		case common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD:
			hasMatch = (nextToken.Kind() == st.TRANSACTIONAL_KEYWORD)
		case common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER:
			hasMatch = (nextToken.Kind() == st.ISOLATED_KEYWORD)
		case common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN:
			hasMatch = ((nextToken.Kind() == st.PIPE_TOKEN) || (nextToken.Kind() == st.BITWISE_AND_TOKEN))
		case common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN:
			hasMatch = (nextToken.Kind() == st.DOT_LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN:
			hasMatch = (nextToken.Kind() == st.SLASH_LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
			hasMatch = (nextToken.Kind() == st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN)
		case common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN:
			hasMatch = (nextToken.Kind() == st.SLASH_ASTERISK_TOKEN)
		case common.PARSER_RULE_CONTEXT_KEY_KEYWORD:
			hasMatch = ((nextToken.Kind() == st.KEY_KEYWORD) || isKeyKeyword(nextToken))
		case common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
			hasMatch = ((nextToken.Kind() == st.NATURAL_KEYWORD) || isNaturalKeyword(nextToken))
		case common.PARSER_RULE_CONTEXT_VAR_KEYWORD:
			hasMatch = (nextToken.Kind() == st.VAR_KEYWORD)
		case common.PARSER_RULE_CONTEXT_ORDER_DIRECTION:
			hasMatch = ((nextToken.Kind() == st.ASCENDING_KEYWORD) || (nextToken.Kind() == st.DESCENDING_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL:
			hasMatch = ((nextToken.Kind() == st.PRIVATE_KEYWORD) || (nextToken.Kind() == st.PUBLIC_KEYWORD))
		case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER,
			common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER,
			common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER,
			common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER:
			hasMatch = ((((nextToken.Kind() == st.ISOLATED_KEYWORD) || (nextToken.Kind() == st.TRANSACTIONAL_KEYWORD)) || (nextToken.Kind() == st.REMOTE_KEYWORD)) || (nextToken.Kind() == st.RESOURCE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER,
			common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER:
			hasMatch = ((nextToken.Kind() == st.ISOLATED_KEYWORD) || (nextToken.Kind() == st.TRANSACTIONAL_KEYWORD))
		case common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL,
			common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL,
			common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL:
			hasMatch = (((nextToken.Kind() == st.FINAL_KEYWORD) || (nextToken.Kind() == st.ISOLATED_KEYWORD)) || (nextToken.Kind() == st.CONFIGURABLE_KEYWORD))
		case common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR:
			hasMatch = isCompoundBinaryOperator(nextToken.Kind())
		case common.PARSER_RULE_CONTEXT_IS_KEYWORD:
			hasMatch = ((nextToken.Kind() == st.IS_KEYWORD) || (nextToken.Kind() == st.NOT_IS_KEYWORD))
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
	result = newResult(make([]*solution, 0), matchingRulesCount)
	result.solution = newSolution(actionKeep, currentCtx, st.NONE, currentCtx.String())
	return result
}

func (b *ballerinaParserErrorHandler) getNextLookahead(lookahead int) int {
	for b.tokenReader.PeekN(lookahead).Kind() == st.DOCUMENTATION_STRING {
		lookahead++
	}
	return lookahead
}

func (b *ballerinaParserErrorHandler) isKeyword(currentCtx common.ParserRuleContext) bool {
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

func (b *ballerinaParserErrorHandler) HasAlternativePaths(currentCtx common.ParserRuleContext) bool {
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

func (b *ballerinaParserErrorHandler) getShortestAlternative(currentCtx common.ParserRuleContext) common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) seekMatchInAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result recoveryResult) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInAlternativePaths start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInAlternativePaths end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResultValue(result))
	})
	var alternativeRules []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE:
		alternativeRules = topLevelNode
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER:
		alternativeRules = topLevelNodeWithoutModifier
	case common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA:
		alternativeRules = topLevelNodeWithoutMetadata
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_START:
		alternativeRules = funcDefStart
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = funcDefWithoutFirstQualifier
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL:
		alternativeRules = funcTypeDescStartWithoutFirstQual
	case common.PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS:
		parentCtx := b.GetParentContext()
		var alternatives []common.ParserRuleContext
		switch parentCtx {
		case common.PARSER_RULE_CONTEXT_FUNC_DEF:
			grandParentCtx := b.GetGrandParentContext()
			if grandParentCtx == common.PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER {
				alternatives = methodDeclOptionalReturns
			} else {
				alternatives = funcDefOptionalReturns
			}
		case common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION:
			alternatives = annonFuncOptionalReturns
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC:
			alternatives = funcTypeOptionalReturns
		case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC:
			alternatives = funcTypeOrAnonFuncOptionalReturns
		default:
			alternatives = funcTypeOrDefOptionalReturns
		}
		alternativeRules = alternatives
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS:
		alternativeRules = funcBodyOrTypeDescRhs
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY:
		alternativeRules = funcTypeDescRhsOrAnonFuncBody
	case common.PARSER_RULE_CONTEXT_ANON_FUNC_BODY:
		alternativeRules = anonFuncBody
	case common.PARSER_RULE_CONTEXT_FUNC_BODY:
		alternativeRules = funcBody
	case common.PARSER_RULE_CONTEXT_PARAM_LIST:
		alternativeRules = paramList
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM_NAME_RHS:
		alternativeRules = requiredParamNameRhs
	case common.PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS:
		alternativeRules = fieldDescriptorRhs
	case common.PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS:
		alternativeRules = fieldOrRestDesciptorRhs
	case common.PARSER_RULE_CONTEXT_RECORD_BODY_END:
		alternativeRules = recordBodyEnd
	case common.PARSER_RULE_CONTEXT_RECORD_BODY_START:
		alternativeRules = recordBodyStart
	case common.PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR:
		if !b.isInTypeDescContext() {
			panic("assertion failed")
		}
		alternativeRules = typeDescriptors
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED:
		alternativeRules = typeDescriptorWithoutIsolated
	case common.PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR:
		alternativeRules = classDescriptor
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END:
		alternativeRules = recordFieldOrRecordEnd
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_START:
		alternativeRules = recordFieldStart
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD_WITHOUT_METADATA:
		alternativeRules = recordFieldWithoutMetadata
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START:
		alternativeRules = classMemberOrObjectMemberStart
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START:
		alternativeRules = objectConstructorMemberStart
	case common.PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META:
		alternativeRules = classMemberOrObjectMemberWithoutMeta
	case common.PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META:
		alternativeRules = objectConsMemberWithoutMeta
	case common.PARSER_RULE_CONTEXT_OPTIONAL_FIELD_INITIALIZER:
		alternativeRules = optionalFieldInitializer
	case common.PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN:
		alternativeRules = onFailOptionalBindingPattern
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_START:
		alternativeRules = objectMethodStart
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = objectMethodWithoutFirstQualifier
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER:
		alternativeRules = objectMethodWithoutSecondQualifier
	case common.PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER:
		alternativeRules = objectMethodWithoutThirdQualifier
	case common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD:
		alternativeRules = objectFuncOrField
	case common.PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY:
		alternativeRules = objectFuncOrFieldWithoutVisibility
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_START:
		alternativeRules = objectTypeStart
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_START:
		alternativeRules = objectConstructorStart
	case common.PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL:
		alternativeRules = importPrefixDecl
	case common.PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS:
		alternativeRules = importDeclOrgOrModuleNameRhs
	case common.PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME:
		alternativeRules = afterImportModuleName
	case common.PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH:
		alternativeRules = optionalAbsolutePath
	case common.PARSER_RULE_CONTEXT_CONST_DECL_TYPE:
		alternativeRules = constDeclType
	case common.PARSER_RULE_CONTEXT_CONST_DECL_RHS:
		alternativeRules = constDeclRhs
	case common.PARSER_RULE_CONTEXT_PARAMETER_START:
		alternativeRules = parameterStart
	case common.PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION:
		alternativeRules = parameterStartWithoutAnnotation
	case common.PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE:
		alternativeRules = annotDeclOptionalType
	case common.PARSER_RULE_CONTEXT_ANNOT_DECL_RHS:
		alternativeRules = annotDeclRhs
	case common.PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS:
		alternativeRules = annotOptionalAttachPoints
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT:
		alternativeRules = attachPoint
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT:
		alternativeRules = attachPointIdent
	case common.PARSER_RULE_CONTEXT_ATTACH_POINT_END:
		alternativeRules = attachPointEnd
	case common.PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL:
		alternativeRules = xmlNamespacePrefixDecl
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_START:
		alternativeRules = enumMemberStart
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS:
		alternativeRules = enumMemberRhs
	case common.PARSER_RULE_CONTEXT_ENUM_MEMBER_END:
		alternativeRules = enumMemberEnd
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS:
		alternativeRules = externalFuncBodyOptionalAnnots
	case common.PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER:
		alternativeRules = listBpOrListConstructorMember
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER:
		alternativeRules = tupleTypeDescOrListConstMember
	case common.PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER:
		alternativeRules = mappingBpOrMappingConstructorMember
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION_START:
		alternativeRules = funcTypeDescStart
	case common.PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION_START:
		alternativeRules = moduleClassDefinitionStart
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = classDefWithoutFirstQualifier
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_SECOND_QUALIFIER:
		alternativeRules = classDefWithoutSecondQualifier
	case common.PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_THIRD_QUALIFIER:
		alternativeRules = classDefWithoutThirdQualifier
	case common.PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF:
		alternativeRules = objectConstructorRhs
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_QUALIFIER:
		alternativeRules = objectFieldQualifier
	case common.PARSER_RULE_CONTEXT_CONFIG_VAR_DECL_RHS:
		alternativeRules = configVarDeclRhs
	case common.PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE:
		alternativeRules = optionalServiceDeclType
	case common.PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS:
		alternativeRules = serviceIdentRhs
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_START:
		alternativeRules = absoluteResourcePathStart
	case common.PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END:
		alternativeRules = absoluteResourcePathEnd
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_OR_VAR_DECL:
		alternativeRules = serviceDeclOrVarDecl
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH:
		alternativeRules = optionalRelativePath
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_START:
		alternativeRules = relativeResourcePathStart
	case common.PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT:
		alternativeRules = resourcePathSegment
	case common.PARSER_RULE_CONTEXT_PATH_PARAM_OPTIONAL_ANNOTS:
		alternativeRules = pathParamOptionalAnnots
	case common.PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS:
		alternativeRules = pathParamEllipsis
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME:
		alternativeRules = optionalPathParamName
	case common.PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END:
		alternativeRules = relativeResourcePathEnd
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_START:
		alternativeRules = serviceDeclStart
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER:
		alternativeRules = optionalTypeParameter
	case common.PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF:
		alternativeRules = mapTypeOrTypeRef
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF:
		alternativeRules = objectTypeOrTypeRef
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF:
		alternativeRules = streamTypeOrTypeRef
	case common.PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF:
		alternativeRules = tableTypeOrTypeRef
	case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF:
		alternativeRules = parameterizedTypeOrTypeRef
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF:
		alternativeRules = typeDescRhsOrTypeRef
	case common.PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF:
		alternativeRules = transactionStmtRhsOrTypeRef
	case common.PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF:
		alternativeRules = tableConsOrQueryExprOrVarRef
	case common.PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF:
		alternativeRules = queryExprOrVarRef
	case common.PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF:
		alternativeRules = errorConsExprOrVarRef
	case common.PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER:
		alternativeRules = qualifiedIdentifier
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_DECL_START:
		alternativeRules = moduleVarDeclStart
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_FIRST_QUAL:
		alternativeRules = moduleVarWithoutFirstQual
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_SECOND_QUAL:
		alternativeRules = moduleVarWithoutSecondQual
	case common.PARSER_RULE_CONTEXT_OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = objectTypeWithoutFirstQualifier
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS:
		alternativeRules = funcDefOrTypeDescRhs
	case common.PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START:
		alternativeRules = exprStartOrInferredTypedescDefaultStart
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END:
		alternativeRules = typeCastParamStartOrInferredTypedescDefaultEnd
	case common.PARSER_RULE_CONTEXT_END_OF_PARAMS_OR_NEXT_PARAM_START:
		alternativeRules = endOfParamsOrNextParamStart
	case common.PARSER_RULE_CONTEXT_PARAM_START:
		alternativeRules = paramStart
	case common.PARSER_RULE_CONTEXT_PARAM_RHS:
		alternativeRules = paramRhs
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_PARAM_RHS:
		alternativeRules = funcTypeParamRhs
	case common.PARSER_RULE_CONTEXT_ANNOTATION_DECL_START:
		alternativeRules = annotationDeclStart
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON:
		alternativeRules = optionalTopLevelSemicolon
	case common.PARSER_RULE_CONTEXT_TUPLE_MEMBER:
		alternativeRules = tupleMember
	default:
		return b.seekMatchInStmtRelatedAlternativePaths(currentCtx, lookahead, currentDepth, matchingRulesCount,
			isEntryPoint)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternativeRules, isEntryPoint)
}

func (b *ballerinaParserErrorHandler) seekMatchInStmtRelatedAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result recoveryResult) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInStmtRelatedAlternativePaths start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInStmtRelatedAlternativePaths end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResultValue(result))
	})
	var alternativeRules []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS:
		alternativeRules = varDeclRhs
	case common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		return b.seekInStatements(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
	case common.PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME:
		alternativeRules = typeOrVarName
	case common.PARSER_RULE_CONTEXT_ELSE_BLOCK:
		alternativeRules = elseBlock
	case common.PARSER_RULE_CONTEXT_ELSE_BODY:
		alternativeRules = elseBody
	case common.PARSER_RULE_CONTEXT_CALL_STMT_START:
		alternativeRules = callStatement
	case common.PARSER_RULE_CONTEXT_RETURN_STMT_RHS:
		alternativeRules = returnRhs
	case common.PARSER_RULE_CONTEXT_ARRAY_LENGTH:
		alternativeRules = arrayLength
	case common.PARSER_RULE_CONTEXT_STMT_START_WITH_EXPR_RHS:
		alternativeRules = stmtStartWithExprRhs
	case common.PARSER_RULE_CONTEXT_EXPR_STMT_RHS:
		alternativeRules = exprStmtRhs
	case common.PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START:
		alternativeRules = expressionStatementStart
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS:
		if !b.isInTypeDescContext() {
			panic("assertion failed")
		}
		alternativeRules = typeDescRhs
	case common.PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS:
		alternativeRules = streamTypeFirstParamRhs
	case common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS:
		alternativeRules = functionKeywordRhs
	case common.PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START:
		alternativeRules = funcTypeFuncKeywordRhsStart
	case common.PARSER_RULE_CONTEXT_WORKER_NAME_RHS:
		alternativeRules = workerNameRhs
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN:
		alternativeRules = bindingPattern
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERNS_START:
		alternativeRules = listBindingPatternsStart
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END:
		alternativeRules = listBindingPatternMemberEnd
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER:
		alternativeRules = listBindingPatternContents
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END:
		alternativeRules = mappingBindingPatternEnd
	case common.PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END:
		alternativeRules = fieldBindingPatternEnd
	case common.PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER:
		alternativeRules = mappingBindingPatternMember
	case common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS:
		alternativeRules = errorBindingPatternErrorKeywordRhs
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START:
		alternativeRules = errorArgListBindingPatternStart
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END:
		alternativeRules = errorMessageBindingPatternEnd
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS:
		alternativeRules = errorMessageBindingPatternRhs
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN:
		alternativeRules = errorFieldBindingPattern
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END:
		alternativeRules = errorFieldBindingPatternEnd
	case common.PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS:
		alternativeRules = keyConstraintsRhs
	case common.PARSER_RULE_CONTEXT_TABLE_TYPE_DESC_RHS:
		alternativeRules = tableTypeDescRhs
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS:
		alternativeRules = typeDescInTupleRhs
	case common.PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS:
		alternativeRules = tupleTypeMemberRhs
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END:
		alternativeRules = listConstructorMemberEnd
	case common.PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS:
		alternativeRules = nilOrParenthesisedTypeDescRhs
	case common.PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS:
		alternativeRules = remoteOrResourceCallOrAsyncSendRhs
	case common.PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END:
		alternativeRules = remoteCallOrAsyncSendEnd
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH:
		alternativeRules = optionalResourceAccessPath
	case common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT:
		alternativeRules = resourceAccessPathSegment
	case common.PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT:
		alternativeRules = computedSegmentOrRestSegment
	case common.PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS:
		alternativeRules = resourceAccessSegmentRhs
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD:
		alternativeRules = optionalResourceAccessMethod
	case common.PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST:
		alternativeRules = optionalResourceAccessActionArgList
	case common.PARSER_RULE_CONTEXT_RECEIVE_WORKERS:
		alternativeRules = receiveWorkers
	case common.PARSER_RULE_CONTEXT_RECEIVE_FIELD:
		alternativeRules = receiveField
	case common.PARSER_RULE_CONTEXT_RECEIVE_FIELD_END:
		alternativeRules = receiveFieldEnd
	case common.PARSER_RULE_CONTEXT_WAIT_KEYWORD_RHS:
		alternativeRules = waitKeywordRhs
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_NAME_RHS:
		alternativeRules = waitFieldNameRhs
	case common.PARSER_RULE_CONTEXT_WAIT_FIELD_END:
		alternativeRules = waitFieldEnd
	case common.PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END:
		alternativeRules = waitFutureExprEnd
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER:
		alternativeRules = optionalPeerWorker
	case common.PARSER_RULE_CONTEXT_ROLLBACK_RHS:
		alternativeRules = rollbackRhs
	case common.PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS:
		alternativeRules = retryKeywordRhs
	case common.PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS:
		alternativeRules = retryTypeParamRhs
	case common.PARSER_RULE_CONTEXT_RETRY_BODY:
		alternativeRules = retryBody
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER:
		alternativeRules = listBpOrTupleTypeMember
	case common.PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_RHS:
		alternativeRules = listBpOrTupleTypeDescRhs
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END:
		alternativeRules = bracketedListMemberEnd
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER:
		alternativeRules = bracketedListMember
	case common.PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS,
		common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS,
		common.PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS:
		alternativeRules = bracketedListRhs
	case common.PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS:
		alternativeRules = bindingPatternOrVarRefRhs
	case common.PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS:
		alternativeRules = typeDescRhsOrBpRhs
	case common.PARSER_RULE_CONTEXT_LIST_BINDING_MEMBER_OR_ARRAY_LENGTH:
		alternativeRules = listBindingMemberOrArrayLength
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS:
		alternativeRules = matchPatternListMemberRhs
	case common.PARSER_RULE_CONTEXT_MATCH_PATTERN_START:
		alternativeRules = matchPatternStart
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERNS_START:
		alternativeRules = listMatchPatternsStart
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER:
		alternativeRules = listMatchPatternMember
	case common.PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS:
		alternativeRules = listMatchPatternMemberRhs
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START:
		alternativeRules = fieldMatchPatternsStart
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER:
		alternativeRules = fieldMatchPatternMember
	case common.PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS:
		alternativeRules = fieldMatchPatternMemberRhs
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN:
		alternativeRules = errorMatchPatternOrConstPattern
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS:
		alternativeRules = errorMatchPatternErrorKeywordRhs
	case common.PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START:
		alternativeRules = errorArgListMatchPatternStart
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END:
		alternativeRules = errorMessageMatchPatternEnd
	case common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS:
		alternativeRules = errorMessageMatchPatternRhs
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN:
		alternativeRules = errorFieldMatchPattern
	case common.PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS:
		alternativeRules = errorFieldMatchPatternRhs
	case common.PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN_RHS:
		alternativeRules = namedArgMatchPatternRhs
	case common.PARSER_RULE_CONTEXT_JOIN_CLAUSE_START:
		alternativeRules = joinClauseStart
	case common.PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE_START:
		alternativeRules = intermediateClauseStart
	case common.PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS:
		alternativeRules = regularCompoundStmtRhs
	case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL_START:
		alternativeRules = namedWorkerDeclStart
	case common.PARSER_RULE_CONTEXT_ASSIGNMENT_STMT_RHS:
		alternativeRules = assignmentStmtRhs
	default:
		return b.seekMatchInExprRelatedAlternativePaths(currentCtx, lookahead, currentDepth, matchingRulesCount,
			isEntryPoint)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternativeRules, isEntryPoint)
}

func (b *ballerinaParserErrorHandler) seekMatchInExprRelatedAlternativePaths(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, matchingRulesCount int, isEntryPoint bool) (result recoveryResult) {
	traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInExprRelatedAlternativePaths start %s %d %d %d %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint))
	})
	defer traceRecovery(currentCtx, func() string {
		return fmt.Sprintf("(seekMatchInExprRelatedAlternativePaths end (%s %d %d %d %s) %s)", formatParserRuleContext(currentCtx), lookahead, currentDepth, matchingRulesCount, formatBool(isEntryPoint), formatResultValue(result))
	})
	var alternativeRules []common.ParserRuleContext
	switch currentCtx {
	case common.PARSER_RULE_CONTEXT_EXPRESSION, common.PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION:
		alternativeRules = expressionStart
	case common.PARSER_RULE_CONTEXT_ARG_START:
		alternativeRules = argStart
	case common.PARSER_RULE_CONTEXT_ARG_START_OR_ARG_LIST_END:
		alternativeRules = argStartOrArgListEnd
	case common.PARSER_RULE_CONTEXT_NAMED_OR_POSITIONAL_ARG_RHS:
		alternativeRules = namedOrPositionalArgRhs
	case common.PARSER_RULE_CONTEXT_ARG_END:
		alternativeRules = argEnd
	case common.PARSER_RULE_CONTEXT_ACCESS_EXPRESSION:
		return b.seekInAccessExpression(currentCtx, lookahead, currentDepth, matchingRulesCount, isEntryPoint)
	case common.PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD:
		alternativeRules = firstMappingFieldStart
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD:
		alternativeRules = mappingFieldStart
	case common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD:
		alternativeRules = specificField
	case common.PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS:
		alternativeRules = specificFieldRhs
	case common.PARSER_RULE_CONTEXT_MAPPING_FIELD_END:
		alternativeRules = mappingFieldEnd
	case common.PARSER_RULE_CONTEXT_LET_VAR_DECL_START:
		alternativeRules = letVarDeclStart
	case common.PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END:
		alternativeRules = orderKeyListEnd
	case common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT:
		alternativeRules = groupingKeyListElement
	case common.PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END:
		alternativeRules = groupingKeyListElementEnd
	case common.PARSER_RULE_CONTEXT_TEMPLATE_MEMBER:
		alternativeRules = templateMember
	case common.PARSER_RULE_CONTEXT_TEMPLATE_STRING_RHS:
		alternativeRules = templateStringRhs
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START:
		alternativeRules = constantExpression
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER:
		alternativeRules = listConstructorFirstMember
	case common.PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER:
		alternativeRules = listConstructorMember
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM:
		alternativeRules = typeCastParam
	case common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_RHS:
		alternativeRules = typeCastParamRhs
	case common.PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS:
		alternativeRules = tableKeywordRhs
	case common.PARSER_RULE_CONTEXT_ROW_LIST_RHS:
		alternativeRules = rowListRhs
	case common.PARSER_RULE_CONTEXT_TABLE_ROW_END:
		alternativeRules = tableRowEnd
	case common.PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS:
		alternativeRules = keySpecifierRhs
	case common.PARSER_RULE_CONTEXT_TABLE_KEY_RHS:
		alternativeRules = tableKeyRhs
	case common.PARSER_RULE_CONTEXT_NEW_KEYWORD_RHS:
		alternativeRules = newKeywordRhs
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START:
		alternativeRules = tableConstructorOrQueryStart
	case common.PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS:
		alternativeRules = tableConstructorOrQueryRhs
	case common.PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS:
		alternativeRules = queryPipelineRhs
	case common.PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS,
		common.PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS:
		alternativeRules = bracedExprOrAnonFuncParamRhs
	case common.PARSER_RULE_CONTEXT_PARAM_END:
		alternativeRules = paramEnd
	case common.PARSER_RULE_CONTEXT_ANNOTATION_REF_RHS:
		alternativeRules = annotationRefRhs
	case common.PARSER_RULE_CONTEXT_INFER_PARAM_END_OR_PARENTHESIS_END:
		alternativeRules = inferParamEndOrParenthesisEnd
	case common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR:
		alternativeRules = xmlNavigateExpr
	case common.PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS:
		alternativeRules = xmlNamePatternRhs
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START:
		alternativeRules = xmlAtomicNamePatternStart
	case common.PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER_RHS:
		alternativeRules = xmlAtomicNameIdentifierRhs
	case common.PARSER_RULE_CONTEXT_XML_STEP_START:
		alternativeRules = xmlStepStart
	case common.PARSER_RULE_CONTEXT_XML_STEP_EXTEND:
		alternativeRules = xmlStepExtend
	case common.PARSER_RULE_CONTEXT_XML_STEP_START_END:
		alternativeRules = xmlStepStartEnd
	case common.PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD:
		alternativeRules = optionalMatchGuard
	case common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END:
		alternativeRules = memberAccessKeyExprEnd
	case common.PARSER_RULE_CONTEXT_LISTENERS_LIST_END:
		alternativeRules = listenersListEnd
	case common.PARSER_RULE_CONTEXT_OBJECT_CONS_WITHOUT_FIRST_QUALIFIER:
		alternativeRules = objectConsWithoutFirstQualifier
	case common.PARSER_RULE_CONTEXT_RESULT_CLAUSE:
		alternativeRules = resultClause
	case common.PARSER_RULE_CONTEXT_EXPRESSION_RHS:
		return b.seekMatchInExpressionRhs(lookahead, currentDepth, matchingRulesCount, isEntryPoint, false)
	case common.PARSER_RULE_CONTEXT_VARIABLE_REF_RHS:
		return b.seekMatchInExpressionRhs(lookahead, currentDepth, matchingRulesCount, isEntryPoint, true)
	case common.PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS:
		alternativeRules = errorConstructorRhs
	case common.PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_SEPARATOR:
		alternativeRules = singleOrAlternateWorkerSeparator
	case common.PARSER_RULE_CONTEXT_NATURAL_EXPRESSION_START:
		alternativeRules = naturalExpressionStart
	case common.PARSER_RULE_CONTEXT_OPTIONAL_PARENTHESIZED_ARG_LIST:
		alternativeRules = optionalParenthesizedArgList
	default:
		panic("seekMatchInExprRelatedAlternativePaths found: " + currentCtx.String())
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, matchingRulesCount, alternativeRules, isEntryPoint)
}

func (b *ballerinaParserErrorHandler) seekInStatements(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, currentMatches int, isEntryPoint bool) recoveryResult {
	nextToken := b.tokenReader.PeekN(lookahead)
	if nextToken.Kind() == st.SEMICOLON_TOKEN {
		result := b.seekMatchInSubTree(common.PARSER_RULE_CONTEXT_STATEMENT, lookahead+1, currentDepth+1,
			isEntryPoint)
		result.pushFix(newSolutionWithDepth(actionRemove, currentCtx, nextToken.Kind(), nextToken.Text(), currentDepth))
		return *b.getFinalResult(currentMatches, result)
	}
	return *b.seekInAlternativesPaths(lookahead, currentDepth, currentMatches, statements, isEntryPoint)
}

func (b *ballerinaParserErrorHandler) seekInAccessExpression(currentCtx common.ParserRuleContext, lookahead int, currentDepth int, currentMatches int, isEntryPoint bool) recoveryResult {
	nextToken := b.tokenReader.PeekN(lookahead)
	currentDepth++
	if nextToken.Kind() != st.IDENTIFIER_TOKEN {
		return *b.fixAndContinue(currentCtx, lookahead, currentDepth, currentMatches, isEntryPoint)
	}
	var nextContext common.ParserRuleContext
	nextNextToken := b.tokenReader.PeekN(lookahead + 1)
	switch nextNextToken.Kind() {
	case st.OPEN_PAREN_TOKEN:
		nextContext = common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS
	case st.DOT_TOKEN:
		nextContext = common.PARSER_RULE_CONTEXT_DOT
	case st.OPEN_BRACKET_TOKEN:
		nextContext = common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR
	default:
		nextContext = b.getNextRuleForExpr()
	}
	currentMatches++
	lookahead++
	result := b.SeekMatch(nextContext, lookahead, currentDepth, isEntryPoint)
	return *b.getFinalResult(currentMatches, result)
}

func (b *ballerinaParserErrorHandler) seekMatchInExpressionRhs(lookahead int, currentDepth int, currentMatches int, isEntryPoint bool, allowFuncCall bool) recoveryResult {
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
		case st.ON_KEYWORD, st.CONFLICT_KEYWORD:
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

func (b *ballerinaParserErrorHandler) getExpressionRhsAlternatives(nextContext common.ParserRuleContext) []common.ParserRuleContext {
	if ((nextContext == common.PARSER_RULE_CONTEXT_SEMICOLON) || (nextContext == common.PARSER_RULE_CONTEXT_QUERY_EXPRESSION_END)) || (nextContext == common.PARSER_RULE_CONTEXT_MATCH_BODY) {
		return []common.ParserRuleContext{nextContext, common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_IS_KEYWORD, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN}
	}
	return []common.ParserRuleContext{common.PARSER_RULE_CONTEXT_BINARY_OPERATOR, common.PARSER_RULE_CONTEXT_IS_KEYWORD, common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN, common.PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION, common.PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR, common.PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR, common.PARSER_RULE_CONTEXT_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN, nextContext}
}

func (b *ballerinaParserErrorHandler) modifyAlternativesWithArgListStart(alternatives []common.ParserRuleContext) []common.ParserRuleContext {
	// Create new slice with capacity for one additional element
	newAlternatives := make([]common.ParserRuleContext, len(alternatives)+1)
	// Copy all existing elements
	copy(newAlternatives, alternatives)
	// Add the new element at the end
	newAlternatives[len(alternatives)] = common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN
	return newAlternatives
}

func (b *ballerinaParserErrorHandler) GetNextRule(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
	b.startContextIfRequired(currentCtx)
	var parentCtx common.ParserRuleContext
	var nextToken st.STToken
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
		if nextToken.Kind() == st.EOF_TOKEN {
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
		if nextToken.Kind() == st.COLON_TOKEN {
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

func (b *ballerinaParserErrorHandler) getNextRuleInternal(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForKeywords(currentCtx common.ParserRuleContext, nextLookahead int) common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) startContextIfRequired(currentCtx common.ParserRuleContext) {
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

func (b *ballerinaParserErrorHandler) getNextRuleForCloseParenthesis() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForOpenParenthesis() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) isInMatchPatternCtx(context common.ParserRuleContext) bool {
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

func (b *ballerinaParserErrorHandler) getNextRuleForOpenBrace() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) isExpressionContext(ctx common.ParserRuleContext) bool {
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

func (b *ballerinaParserErrorHandler) getNextRuleForParamType() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForComma() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForTypeDescriptor() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) isInTypeDescContext() bool {
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

func (b *ballerinaParserErrorHandler) getNextRuleForEqualOp() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForCloseBrace(nextLookahead int) common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	var nextToken st.STToken
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
				case st.CLOSE_BRACE_TOKEN:
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

func (b *ballerinaParserErrorHandler) getNextRuleForCloseBraceInFuncBody() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForAnnotationEnd(nextLookahead int) common.ParserRuleContext {
	var parentCtx common.ParserRuleContext
	var nextToken = b.tokenReader.PeekN(nextLookahead)
	if nextToken.Kind() == st.AT_TOKEN {
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

func (b *ballerinaParserErrorHandler) getNextRuleForVarName() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForSemicolon(nextLookahead int) common.ParserRuleContext {
	var nextToken st.STToken
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
		if nextToken.Kind() == st.EOF_TOKEN {
			return common.PARSER_RULE_CONTEXT_EOF
		}
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else if parentCtx == common.PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST {
		b.EndContext()
		b.EndContext()
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == st.EOF_TOKEN {
			return common.PARSER_RULE_CONTEXT_EOF
		}
		return common.PARSER_RULE_CONTEXT_TOP_LEVEL_NODE
	} else if (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF) || (parentCtx == common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE) {
		b.EndContext()
		nextToken = b.tokenReader.PeekN(nextLookahead)
		if nextToken.Kind() == st.EOF_TOKEN {
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

func (b *ballerinaParserErrorHandler) getNextRuleForDot() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForQuestionMark() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForOpenBracket() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForCloseBracket() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForDecimalIntegerLiteral() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION:
		b.EndContext()
		return b.getNextRuleForConstExpr()
	default:
		return common.PARSER_RULE_CONTEXT_CLOSE_BRACKET
	}
}

func (b *ballerinaParserErrorHandler) getNextRuleForExpr() common.ParserRuleContext {
	var parentCtx = b.GetParentContext()
	if parentCtx == common.PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION {
		b.EndContext()
		return b.getNextRuleForConstExpr()
	}
	return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
}

func (b *ballerinaParserErrorHandler) getNextRuleForExprStartsWithVarRef() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForConstExpr() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForLt() common.ParserRuleContext {
	parentCtx := b.GetParentContext()
	switch parentCtx {
	case common.PARSER_RULE_CONTEXT_TYPE_CAST:
		return common.PARSER_RULE_CONTEXT_TYPE_CAST_PARAM
	default:
		return common.PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS
	}
}

func (b *ballerinaParserErrorHandler) getNextRuleForGt() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForBindingPatternDefault() common.ParserRuleContext {
	return b.getNextRuleForBindingPatternWithCapture(false)
}

func (b *ballerinaParserErrorHandler) getNextRuleForBindingPatternWithCapture(isCaptureBP bool) common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForWaitExprListEnd() common.ParserRuleContext {
	b.EndContext()
	return common.PARSER_RULE_CONTEXT_EXPRESSION_RHS
}

func (b *ballerinaParserErrorHandler) getNextRuleForIdentifier() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForColon() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForMatchPattern() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForTypeReference() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForErrorKeyword() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForFuncTypeFuncKeywordRhs() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) getNextRuleForAction() common.ParserRuleContext {
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

func (b *ballerinaParserErrorHandler) isStatement(parentCtx common.ParserRuleContext) bool {
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

func (b *ballerinaParserErrorHandler) isBinaryOperator(token st.STToken) bool {
	switch token.Kind() {
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
		st.DOUBLE_LT_TOKEN,
		st.DOUBLE_GT_TOKEN,
		st.TRIPPLE_GT_TOKEN,
		st.ELLIPSIS_TOKEN,
		st.DOUBLE_DOT_LT_TOKEN,
		st.ELVIS_TOKEN:
		return true
	case st.RIGHT_ARROW_TOKEN,
		st.RIGHT_DOUBLE_ARROW_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParserErrorHandler) isParameter(ctx common.ParserRuleContext) bool {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_REQUIRED_PARAM, common.PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM, common.PARSER_RULE_CONTEXT_REST_PARAM, common.PARSER_RULE_CONTEXT_PARAM_LIST:
		return true
	default:
		return false
	}
}

func (b *ballerinaParserErrorHandler) GetInsertSolution(ctx common.ParserRuleContext) *solution {
	kind := b.GetExpectedTokenKind(ctx)
	if kind != st.NONE {
		return newSolution(actionInsert, ctx, kind, ctx.String())
	}
	if b.HasAlternativePaths(ctx) {
		ctx = b.getShortestAlternative(ctx)
		return b.GetInsertSolution(ctx)
	}
	ctx = b.GetNextRule(ctx, 1)
	return b.GetInsertSolution(ctx)
}

func (b *ballerinaParserErrorHandler) GetExpectedTokenKind(ctx common.ParserRuleContext) st.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY:
		return st.EQUAL_TOKEN
	case common.PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK:
		return st.OPEN_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_FUNC_DEF,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC:
		return st.FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR:
		return st.ANY_KEYWORD
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
		return st.IDENTIFIER_TOKEN
	case common.PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN,
		common.PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS:
		return st.DECIMAL_INTEGER_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN:
		return st.STRING_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR:
		return st.OPTIONAL_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR:
		return st.ARRAY_TYPE_DESC
	case common.PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN:
		return st.HEX_INTEGER_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS:
		return st.SEMICOLON_TOKEN
	case common.PARSER_RULE_CONTEXT_DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
		return st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_HEX_FLOATING_POINT_LITERAL_TOKEN:
		return st.HEX_FLOATING_POINT_LITERAL_TOKEN
	case common.PARSER_RULE_CONTEXT_STATEMENT, common.PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		return st.CLOSE_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN, common.PARSER_RULE_CONTEXT_NIL_LITERAL:
		return st.OPEN_PAREN_TOKEN
	default:
		return b.getExpectedSeperatorTokenKind(ctx)
	}
}

func (b *ballerinaParserErrorHandler) getExpectedSeperatorTokenKind(ctx common.ParserRuleContext) st.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR:
		return st.BITWISE_AND_TOKEN
	case common.PARSER_RULE_CONTEXT_EQUAL_OR_RIGHT_ARROW, common.PARSER_RULE_CONTEXT_ASSIGN_OP:
		return st.EQUAL_TOKEN
	case common.PARSER_RULE_CONTEXT_EOF:
		return st.EOF_TOKEN
	case common.PARSER_RULE_CONTEXT_BINARY_OPERATOR:
		return st.PLUS_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSE_BRACE:
		return st.CLOSE_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN:
		return st.CLOSE_PAREN_TOKEN
	case common.PARSER_RULE_CONTEXT_COMMA,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA,
		common.PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA:
		return st.COMMA_TOKEN
	case common.PARSER_RULE_CONTEXT_OPEN_BRACE:
		return st.OPEN_BRACE_TOKEN
	case common.PARSER_RULE_CONTEXT_OPEN_PARENTHESIS,
		common.PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN,
		common.PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START:
		return st.OPEN_PAREN_TOKEN
	case common.PARSER_RULE_CONTEXT_SEMICOLON:
		return st.SEMICOLON_TOKEN
	case common.PARSER_RULE_CONTEXT_ASTERISK:
		return st.ASTERISK_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END:
		return st.CLOSE_BRACE_PIPE_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START:
		return st.OPEN_BRACE_PIPE_TOKEN
	case common.PARSER_RULE_CONTEXT_ELLIPSIS:
		return st.ELLIPSIS_TOKEN
	case common.PARSER_RULE_CONTEXT_QUESTION_MARK:
		return st.QUESTION_MARK_TOKEN
	case common.PARSER_RULE_CONTEXT_CLOSE_BRACKET:
		return st.CLOSE_BRACKET_TOKEN
	case common.PARSER_RULE_CONTEXT_DOT, common.PARSER_RULE_CONTEXT_METHOD_CALL_DOT:
		return st.DOT_TOKEN
	case common.PARSER_RULE_CONTEXT_OPEN_BRACKET, common.PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START:
		return st.OPEN_BRACKET_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH,
		common.PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH,
		common.PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN:
		return st.SLASH_TOKEN
	case common.PARSER_RULE_CONTEXT_COLON, common.PARSER_RULE_CONTEXT_TYPE_REF_COLON, common.PARSER_RULE_CONTEXT_VAR_REF_COLON:
		return st.COLON_TOKEN
	case common.PARSER_RULE_CONTEXT_UNARY_OPERATOR,
		common.PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR,
		common.PARSER_RULE_CONTEXT_UNARY_EXPRESSION,
		common.PARSER_RULE_CONTEXT_EXPRESSION_RHS:
		return st.PLUS_TOKEN
	case common.PARSER_RULE_CONTEXT_AT:
		return st.AT_TOKEN
	case common.PARSER_RULE_CONTEXT_RIGHT_ARROW:
		return st.RIGHT_ARROW_TOKEN
	case common.PARSER_RULE_CONTEXT_GT, common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT:
		return st.GT_TOKEN
	case common.PARSER_RULE_CONTEXT_LT,
		common.PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN,
		common.PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT:
		return st.LT_TOKEN
	case common.PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN:
		return st.SYNC_SEND_TOKEN
	case common.PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN:
		return st.ANNOT_CHAINING_TOKEN
	case common.PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN:
		return st.OPTIONAL_CHAINING_TOKEN
	case common.PARSER_RULE_CONTEXT_DOT_LT_TOKEN:
		return st.DOT_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH_LT_TOKEN:
		return st.SLASH_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
		return st.DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN
	case common.PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN:
		return st.SLASH_ASTERISK_TOKEN
	case common.PARSER_RULE_CONTEXT_PLUS_TOKEN:
		return st.PLUS_TOKEN
	case common.PARSER_RULE_CONTEXT_MINUS_TOKEN:
		return st.MINUS_TOKEN
	case common.PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN:
		return st.LEFT_ARROW_TOKEN
	case common.PARSER_RULE_CONTEXT_TEMPLATE_END, common.PARSER_RULE_CONTEXT_TEMPLATE_START:
		return st.BACKTICK_TOKEN
	case common.PARSER_RULE_CONTEXT_LT_TOKEN:
		return st.LT_TOKEN
	case common.PARSER_RULE_CONTEXT_GT_TOKEN:
		return st.GT_TOKEN
	case common.PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN:
		return st.INTERPOLATION_START_TOKEN
	case common.PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START,
		common.PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW:
		return st.RIGHT_DOUBLE_ARROW_TOKEN
	default:
		return b.getExpectedKeywordKind(ctx)
	}
}

func (b *ballerinaParserErrorHandler) getExpectedKeywordKind(ctx common.ParserRuleContext) st.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD:
		return st.EXTERNAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNCTION_KEYWORD,
		common.PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT,
		common.PARSER_RULE_CONTEXT_FUNCTION_IDENT,
		common.PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER,
		common.PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND:
		return st.FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETURNS_KEYWORD:
		return st.RETURNS_KEYWORD
	case common.PARSER_RULE_CONTEXT_PUBLIC_KEYWORD:
		return st.PUBLIC_KEYWORD
	case common.PARSER_RULE_CONTEXT_RECORD_FIELD,
		common.PARSER_RULE_CONTEXT_RECORD_KEYWORD,
		common.PARSER_RULE_CONTEXT_RECORD_IDENT:
		return st.RECORD_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPE_KEYWORD,
		common.PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT:
		return st.TYPE_KEYWORD
	case common.PARSER_RULE_CONTEXT_OBJECT_KEYWORD,
		common.PARSER_RULE_CONTEXT_OBJECT_IDENT,
		common.PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR:
		return st.OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_PRIVATE_KEYWORD:
		return st.PRIVATE_KEYWORD
	case common.PARSER_RULE_CONTEXT_REMOTE_IDENT:
		return st.REMOTE_KEYWORD
	case common.PARSER_RULE_CONTEXT_ABSTRACT_KEYWORD:
		return st.ABSTRACT_KEYWORD
	case common.PARSER_RULE_CONTEXT_CLIENT_KEYWORD:
		return st.CLIENT_KEYWORD
	case common.PARSER_RULE_CONTEXT_IF_KEYWORD:
		return st.IF_KEYWORD
	case common.PARSER_RULE_CONTEXT_ELSE_KEYWORD:
		return st.ELSE_KEYWORD
	case common.PARSER_RULE_CONTEXT_WHILE_KEYWORD:
		return st.WHILE_KEYWORD
	case common.PARSER_RULE_CONTEXT_CHECKING_KEYWORD:
		return st.CHECK_KEYWORD
	case common.PARSER_RULE_CONTEXT_FAIL_KEYWORD:
		return st.FAIL_KEYWORD
	case common.PARSER_RULE_CONTEXT_AS_KEYWORD:
		return st.AS_KEYWORD
	case common.PARSER_RULE_CONTEXT_BOOLEAN_LITERAL:
		return st.TRUE_KEYWORD
	case common.PARSER_RULE_CONTEXT_IMPORT_KEYWORD:
		return st.IMPORT_KEYWORD
	case common.PARSER_RULE_CONTEXT_ON_KEYWORD:
		return st.ON_KEYWORD
	case common.PARSER_RULE_CONTEXT_PANIC_KEYWORD:
		return st.PANIC_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETURN_KEYWORD:
		return st.RETURN_KEYWORD
	case common.PARSER_RULE_CONTEXT_SERVICE_KEYWORD, common.PARSER_RULE_CONTEXT_SERVICE_IDENT:
		return st.SERVICE_KEYWORD
	case common.PARSER_RULE_CONTEXT_BREAK_KEYWORD:
		return st.BREAK_KEYWORD
	case common.PARSER_RULE_CONTEXT_LISTENER_KEYWORD:
		return st.LISTENER_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONTINUE_KEYWORD:
		return st.CONTINUE_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONST_KEYWORD:
		return st.CONST_KEYWORD
	case common.PARSER_RULE_CONTEXT_FINAL_KEYWORD:
		return st.FINAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_IS_KEYWORD:
		return st.IS_KEYWORD
	case common.PARSER_RULE_CONTEXT_TYPEOF_KEYWORD:
		return st.TYPEOF_KEYWORD
	case common.PARSER_RULE_CONTEXT_MAP_KEYWORD, common.PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR:
		return st.MAP_KEYWORD
	case common.PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE,
		common.PARSER_RULE_CONTEXT_ERROR_KEYWORD,
		common.PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN:
		return st.ERROR_KEYWORD
	case common.PARSER_RULE_CONTEXT_NULL_KEYWORD:
		return st.NULL_KEYWORD
	case common.PARSER_RULE_CONTEXT_LOCK_KEYWORD:
		return st.LOCK_KEYWORD
	case common.PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD:
		return st.ANNOTATION_KEYWORD
	case common.PARSER_RULE_CONTEXT_FIELD_IDENT:
		return st.FIELD_KEYWORD
	case common.PARSER_RULE_CONTEXT_XMLNS_KEYWORD,
		common.PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION:
		return st.XMLNS_KEYWORD
	case common.PARSER_RULE_CONTEXT_SOURCE_KEYWORD:
		return st.SOURCE_KEYWORD
	case common.PARSER_RULE_CONTEXT_START_KEYWORD:
		return st.START_KEYWORD
	case common.PARSER_RULE_CONTEXT_FLUSH_KEYWORD:
		return st.FLUSH_KEYWORD
	case common.PARSER_RULE_CONTEXT_WAIT_KEYWORD:
		return st.WAIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD:
		return st.TRANSACTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD:
		return st.TRANSACTIONAL_KEYWORD
	case common.PARSER_RULE_CONTEXT_COMMIT_KEYWORD:
		return st.COMMIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_RETRY_KEYWORD:
		return st.RETRY_KEYWORD
	case common.PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD:
		return st.ROLLBACK_KEYWORD
	case common.PARSER_RULE_CONTEXT_ENUM_KEYWORD:
		return st.ENUM_KEYWORD
	case common.PARSER_RULE_CONTEXT_MATCH_KEYWORD:
		return st.MATCH_KEYWORD
	case common.PARSER_RULE_CONTEXT_NEW_KEYWORD:
		return st.NEW_KEYWORD
	case common.PARSER_RULE_CONTEXT_FORK_KEYWORD:
		return st.FORK_KEYWORD
	case common.PARSER_RULE_CONTEXT_NAMED_WORKER_DECL, common.PARSER_RULE_CONTEXT_WORKER_KEYWORD:
		return st.WORKER_KEYWORD
	case common.PARSER_RULE_CONTEXT_TRAP_KEYWORD:
		return st.TRAP_KEYWORD
	case common.PARSER_RULE_CONTEXT_FOREACH_KEYWORD:
		return st.FOREACH_KEYWORD
	case common.PARSER_RULE_CONTEXT_IN_KEYWORD:
		return st.IN_KEYWORD
	case common.PARSER_RULE_CONTEXT_PIPE, common.PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN:
		return st.PIPE_TOKEN
	case common.PARSER_RULE_CONTEXT_TABLE_KEYWORD:
		return st.TABLE_KEYWORD
	case common.PARSER_RULE_CONTEXT_KEY_KEYWORD:
		return st.KEY_KEYWORD
	case common.PARSER_RULE_CONTEXT_STREAM_KEYWORD:
		return st.STREAM_KEYWORD
	case common.PARSER_RULE_CONTEXT_LET_KEYWORD:
		return st.LET_KEYWORD
	case common.PARSER_RULE_CONTEXT_XML_KEYWORD:
		return st.XML_KEYWORD
	case common.PARSER_RULE_CONTEXT_RE_KEYWORD:
		return st.RE_KEYWORD
	case common.PARSER_RULE_CONTEXT_STRING_KEYWORD:
		return st.STRING_KEYWORD
	case common.PARSER_RULE_CONTEXT_BASE16_KEYWORD:
		return st.BASE16_KEYWORD
	case common.PARSER_RULE_CONTEXT_BASE64_KEYWORD:
		return st.BASE64_KEYWORD
	case common.PARSER_RULE_CONTEXT_SELECT_KEYWORD:
		return st.SELECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_WHERE_KEYWORD:
		return st.WHERE_KEYWORD
	case common.PARSER_RULE_CONTEXT_FROM_KEYWORD:
		return st.FROM_KEYWORD
	case common.PARSER_RULE_CONTEXT_ORDER_KEYWORD:
		return st.ORDER_KEYWORD
	case common.PARSER_RULE_CONTEXT_GROUP_KEYWORD:
		return st.GROUP_KEYWORD
	case common.PARSER_RULE_CONTEXT_BY_KEYWORD:
		return st.BY_KEYWORD
	case common.PARSER_RULE_CONTEXT_ORDER_DIRECTION:
		return st.ASCENDING_KEYWORD
	case common.PARSER_RULE_CONTEXT_DO_KEYWORD:
		return st.DO_KEYWORD
	case common.PARSER_RULE_CONTEXT_DISTINCT_KEYWORD:
		return st.DISTINCT_KEYWORD
	case common.PARSER_RULE_CONTEXT_VAR_KEYWORD:
		return st.VAR_KEYWORD
	case common.PARSER_RULE_CONTEXT_CONFLICT_KEYWORD:
		return st.CONFLICT_KEYWORD
	case common.PARSER_RULE_CONTEXT_LIMIT_KEYWORD:
		return st.LIMIT_KEYWORD
	case common.PARSER_RULE_CONTEXT_EQUALS_KEYWORD:
		return st.EQUALS_KEYWORD
	case common.PARSER_RULE_CONTEXT_JOIN_KEYWORD:
		return st.JOIN_KEYWORD
	case common.PARSER_RULE_CONTEXT_OUTER_KEYWORD:
		return st.OUTER_KEYWORD
	case common.PARSER_RULE_CONTEXT_CLASS_KEYWORD:
		return st.CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_COLLECT_KEYWORD:
		return st.COLLECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
		return st.NATURAL_KEYWORD
	default:
		return b.getExpectedQualifierKind(ctx)
	}
}

func (b *ballerinaParserErrorHandler) getExpectedQualifierKind(ctx common.ParserRuleContext) st.SyntaxKind {
	switch ctx {
	case common.PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER:
		return st.OBJECT_KEYWORD
	case common.PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER:
		return st.CLASS_KEYWORD
	case common.PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER,
		common.PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER:
		return st.FUNCTION_KEYWORD
	case common.PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL,
		common.PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL,
		common.PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL:
		return st.IDENTIFIER_TOKEN
	case common.PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER:
		return st.SERVICE_KEYWORD
	default:
		return st.NONE
	}
}

func (b *ballerinaParserErrorHandler) isBasicLiteral(kind st.SyntaxKind) bool {
	switch kind {
	case st.DECIMAL_INTEGER_LITERAL_TOKEN,
		st.HEX_INTEGER_LITERAL_TOKEN,
		st.STRING_LITERAL_TOKEN,
		st.TRUE_KEYWORD,
		st.FALSE_KEYWORD,
		st.NULL_KEYWORD,
		st.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		st.HEX_FLOATING_POINT_LITERAL_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParserErrorHandler) isUnaryOperator(token st.STToken) bool {
	switch token.Kind() {
	case st.PLUS_TOKEN,
		st.MINUS_TOKEN,
		st.NEGATION_TOKEN,
		st.EXCLAMATION_MARK_TOKEN:
		return true
	default:
		return false
	}
}

func (b *ballerinaParserErrorHandler) isSingleKeywordAttachPointIdent(tokenKind st.SyntaxKind) bool {
	switch tokenKind {
	case st.ANNOTATION_KEYWORD,
		st.EXTERNAL_KEYWORD,
		st.VAR_KEYWORD,
		st.CONST_KEYWORD,
		st.LISTENER_KEYWORD,
		st.WORKER_KEYWORD,
		st.TYPE_KEYWORD,
		st.FUNCTION_KEYWORD,
		st.PARAMETER_KEYWORD,
		st.RETURN_KEYWORD,
		st.FIELD_KEYWORD,
		st.CLASS_KEYWORD:
		return true
	default:
		return false
	}
}
