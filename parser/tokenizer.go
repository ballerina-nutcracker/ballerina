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
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

func failedToken() st.STToken {
	return st.CreateToken(st.EOF_TOKEN, st.CreateEmptyNodeList(), st.CreateEmptyNodeList())
}

type tokenReader struct {
	lexer             tokenLexer
	currentToken      st.STToken
	tokenBuffer       tokenBuffer
	currentTokenIndex int
	ctx               *compilercontext.CompilerContext
	fileName          string
	location          diagnostics.Location
}

func createTokenReader(ctx *compilercontext.CompilerContext, fileName string, lexer tokenLexer) *tokenReader {
	location := diagnostics.NewLocation(ctx.DiagnosticEnv(), fileName, 0, 0)
	return &tokenReader{
		lexer:             lexer,
		currentToken:      nil,
		currentTokenIndex: 0,
		ctx:               ctx,
		fileName:          fileName,
		location:          location,
		tokenBuffer: tokenBuffer{
			capacity:   20,
			tokens:     make([]st.STToken, bufferSize),
			endIndex:   -1,
			startIndex: -1,
			size:       0,
			ctx:        ctx,
			location:   location,
		},
	}
}

func (t *tokenReader) internalError(message string) {
	t.ctx.InternalError(message, t.location)
}

func (t *tokenReader) Read() st.STToken {
	if t.tokenBuffer.size > 0 {
		t.currentToken = t.tokenBuffer.consume()
	} else {
		t.currentToken = t.lexer.NextToken()
	}
	t.currentTokenIndex++
	return t.currentToken
}

func (t *tokenReader) Peek() st.STToken {
	if t.tokenBuffer.size > 0 {
		return t.tokenBuffer.peek()
	}
	token := t.lexer.NextToken()
	t.tokenBuffer.add(token)
	return token
}

func (t *tokenReader) PeekN(n int) st.STToken {
	if n >= bufferSize {
		t.internalError("n is too large")
		return failedToken()
	}
	remaining := n - t.tokenBuffer.size
	for remaining > 0 {
		t.tokenBuffer.add(t.lexer.NextToken())
		remaining--
	}
	return t.tokenBuffer.peekN(n)
}

func (t *tokenReader) Head() st.STToken {
	return t.currentToken
}

func (t *tokenReader) StartMode(mode parserMode) {
	t.lexer.StartMode(mode)
}

func (t *tokenReader) SwitchMode(mode parserMode) {
	t.lexer.SwitchMode(mode)
}

func (t *tokenReader) EndMode() {
	t.lexer.EndMode()
}

func (t *tokenReader) GetCurrentMode() parserMode {
	return t.lexer.GetCurrentMode()
}

func (t *tokenReader) GetCurrentTokenIndex() int {
	return t.currentTokenIndex
}

const bufferSize = 20

type tokenBuffer struct {
	capacity   int
	tokens     []st.STToken
	endIndex   int
	startIndex int
	size       int
	ctx        *compilercontext.CompilerContext
	location   diagnostics.Location
}

func (t *tokenBuffer) internalError(message string) {
	t.ctx.InternalError(message, t.location)
}

func (t *tokenBuffer) add(token st.STToken) {
	if t.size == t.capacity {
		t.internalError("buffer overflow")
		return
	}

	if t.endIndex == t.capacity-1 {
		t.endIndex = 0
	} else {
		t.endIndex++
	}

	if t.size == 0 {
		t.startIndex = t.endIndex
	}

	t.tokens[t.endIndex] = token
	t.size++
}

func (t *tokenBuffer) peek() st.STToken {
	return t.tokens[t.startIndex]
}

func (t *tokenBuffer) peekN(n int) st.STToken {
	if n > t.size {
		t.internalError("n is too large")
		return failedToken()
	}

	index := t.startIndex + n - 1
	if index >= t.capacity {
		index -= t.capacity
	}

	return t.tokens[index]
}

func (t *tokenBuffer) consume() st.STToken {
	token := t.tokens[t.startIndex]
	t.size--
	if t.startIndex == t.capacity-1 {
		t.startIndex = 0
	} else {
		t.startIndex++
	}
	return token
}

func (t *tokenBuffer) getCurrentTokenIndex() int {
	return t.startIndex
}
