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
	"github.com/ballerina-nutcracker/ballerina/st"
)

type tokenReader struct {
	lexer             tokenLexer
	currentToken      st.STToken
	tokenBuffer       tokenBuffer
	currentTokenIndex int
}

func createTokenReader(lexer tokenLexer) *tokenReader {
	return &tokenReader{
		lexer:             lexer,
		currentToken:      nil,
		currentTokenIndex: 0,
		tokenBuffer: tokenBuffer{
			capacity:   bufferSize,
			tokens:     make([]st.STToken, bufferSize),
			endIndex:   -1,
			startIndex: -1,
			size:       0,
		},
	}
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
	} else {
		token := t.lexer.NextToken()
		t.tokenBuffer.add(token)
		return token
	}
}

func (t *tokenReader) PeekN(n int) st.STToken {
	if n >= bufferSize {
		panic("n is too large")
	}
	remaining := n - t.tokenBuffer.size
	for remaining > 0 {
		token := t.lexer.NextToken()
		t.tokenBuffer.add(token)
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
}

func (t *tokenBuffer) add(token st.STToken) {
	if t.size == t.capacity {
		panic("buffer overflow")
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
		panic("n is too large")
	}

	index := t.startIndex + n - 1
	if index >= t.capacity {
		index = index - t.capacity
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
