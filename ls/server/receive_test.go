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
	"strings"
	"testing"
	"time"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func TestReceiveFailures(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name string
		want string
		read func(*testSession) (protocol.Message, error)
	}{
		{"notification", "timeout waiting for textDocument/publishDiagnostics after 50ms", func(s *testSession) (protocol.Message, error) {
			return s.receiveWithin(50*time.Millisecond, "textDocument/publishDiagnostics", "")
		}},
		{"response", "timeout waiting for shutdown (request ID \"2\") after 50ms", func(s *testSession) (protocol.Message, error) {
			return s.receiveWithin(50*time.Millisecond, "shutdown", `"2"`)
		}},
		{"read-error", "read message for textDocument/publishDiagnostics: EOF", func(s *testSession) (protocol.Message, error) {
			s.clientR = bufio.NewReader(strings.NewReader(""))
			return s.receiveWithin(time.Minute, "textDocument/publishDiagnostics", "")
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			session := newTestSession(t, true)
			defer session.close()
			start := time.Now()
			_, err := scenario.read(session)
			if err == nil || err.Error() != scenario.want {
				t.Fatalf("error = %v, want %q", err, scenario.want)
			}
			if elapsed := time.Since(start); elapsed >= 5*time.Second {
				t.Fatalf("failure took %v, want prompt return", elapsed)
			}
		})
	}
}
