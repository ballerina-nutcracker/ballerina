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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"bufio"
	"bytes"
	"sync"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

// TestWriteSerializesConcurrentFrames verifies writeMu serializes framed
// writes so concurrent writes from the Serve goroutine and the CE-subscriber
// goroutine cannot interleave header/body bytes. Two goroutines each write N
// frames; the output must parse as 2*N clean frames (ls/AGENTS.md URI-unit-test
// exception).
func TestWriteSerializesConcurrentFrames(t *testing.T) {
	var buf bytes.Buffer
	s := &Server{transport: pipeTransport{writer: &buf}}

	const perGoroutine = 50
	const goroutines = 2
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				if err := s.write(protocol.Notification{
					JSONRPC: "2.0",
					Method:  "textDocument/publishDiagnostics",
					Params: protocol.PublishDiagnosticsParams{
						URI:         "file:///workspace/main.bal",
						Diagnostics: nil,
					},
				}); err != nil {
					t.Errorf("write: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	// The buffer must parse cleanly into exactly perGoroutine*goroutines
	// frames; any interleaving would corrupt framing and yield a parse error.
	reader := bufio.NewReader(bytes.NewReader(buf.Bytes()))
	count := 0
	for {
		msg, err := protocol.ReadMessage(reader)
		if err != nil {
			t.Fatalf("frame %d parse error: %v", count, err)
		}
		_ = msg
		count++
		if reader.Buffered() == 0 {
			peek, err := reader.Peek(1)
			if len(peek) == 0 || err != nil {
				break
			}
		}
	}
	if want := perGoroutine * goroutines; count != want {
		t.Fatalf("parsed %d frames, want %d (interleaving corrupted framing)", count, want)
	}
}
