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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

func runRawInput(t *testing.T, input string) (*Server, []protocol.Message, error) {
	t.Helper()
	transport := &batchTransport{reader: bytes.NewReader([]byte(input))}
	srv := New(transport, nil, nil, nil)
	err := srv.Serve()
	srv.cancelAndDrainTrackedRequests()
	reader := bufio.NewReader(bytes.NewReader(transport.writer.Bytes()))
	var replies []protocol.Message
	for {
		reply, readErr := protocol.ReadMessage(reader)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("read reply: %v", readErr)
		}
		replies = append(replies, reply)
	}
	return srv, replies, err
}

func rawFrame(body string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
}

const validInitialize = `{"jsonrpc":"2.0","id":"recovery","method":"initialize","params":{}}`

func TestCompleteMalformedJSONRecovery(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`{`, ``, `{"jsonrpc":`, `{} {}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			srv, replies, err := runRawInput(t, rawFrame(body)+rawFrame(validInitialize))
			if err != nil {
				t.Fatalf("Serve: %v", err)
			}
			if len(replies) != 2 {
				t.Fatalf("replies = %+v, want error and initialization", replies)
			}
			assertRPCError(t, replies[0], `null`, -32700)
			if !srv.initialized || string(replies[1].ID) != `"recovery"` || replies[1].Error != nil {
				t.Fatalf("initialization after malformed body failed: %+v", replies[1])
			}
		})
	}
}

func assertRPCError(t *testing.T, reply protocol.Message, id string, code int) {
	t.Helper()
	if !json.Valid(reply.ID) || string(reply.ID) != id || reply.Error == nil || reply.Error.Code != code || reply.JSONRPC != "2.0" || len(reply.Result) != 0 {
		t.Fatalf("reply = %+v (raw ID %s), want id %s, error %d", reply, reply.ID, id, code)
	}
}

func TestInvalidEnvelopeRecovery(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body, id string }{
		{"null body", `null`, `null`},
		{"array body", `[ {"jsonrpc":"2.0","id":1,"method":"shutdown"} ]`, `null`},
		{"string body", `"message"`, `null`},
		{"numeric body", `42`, `null`},
		{"boolean body", `true`, `null`},
		{"missing jsonrpc", `{"id":"bad","method":"initialize","params":{}}`, `"bad"`},
		{"incorrect jsonrpc", `{"jsonrpc":"1.0","id":9,"method":"initialize","params":{}}`, `9`},
		{"nonstring jsonrpc", `{"jsonrpc":2,"id":"bad","method":"initialize"}`, `"bad"`},
		{"numeric method", `{"jsonrpc":"2.0","id":9,"method":42}`, `9`},
		{"null method", `{"jsonrpc":"2.0","id":"bad","method":null}`, `"bad"`},
		{"missing method", `{"jsonrpc":"2.0","id":9}`, `9`},
		{"empty object", `{}`, `null`},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"initialize","params":{}}`, `null`},
		{"boolean id", `{"jsonrpc":"2.0","id":true,"method":"initialize","params":{}}`, `null`},
		{"array id", `{"jsonrpc":"2.0","id":[],"method":"initialize","params":{}}`, `null`},
		{"object id", `{"jsonrpc":"2.0","id":{},"method":"initialize","params":{}}`, `null`},
		{"fractional id", `{"jsonrpc":"2.0","id":1.5,"method":"initialize","params":{}}`, `null`},
		{"id above int32", `{"jsonrpc":"2.0","id":2147483648,"method":"initialize","params":{}}`, `null`},
		{"id below int32", `{"jsonrpc":"2.0","id":-2147483649,"method":"initialize","params":{}}`, `null`},
		{"oversized integral id", `{"jsonrpc":"2.0","id":999999999999999999999999999999,"method":"initialize","params":{}}`, `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv, replies, err := runRawInput(t, rawFrame(tc.body)+rawFrame(validInitialize))
			if err != nil {
				t.Fatalf("Serve: %v", err)
			}
			if len(replies) != 2 {
				t.Fatalf("replies = %+v, want rejection and initialization", replies)
			}
			assertRPCError(t, replies[0], tc.id, -32600)
			if !srv.initialized || replies[1].Error != nil || string(replies[1].ID) != `"recovery"` {
				t.Fatalf("initialization after invalid envelope failed: %+v", replies[1])
			}
		})
	}
}

func TestNotificationsAndInboundResponsesAreSilent(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"jsonrpc":"2.0","method":"unknown/method"}`,
		`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":false}`,
		`{"method":"exit"}`,
		`{"jsonrpc":"1.0","method":"exit"}`,
		`{"jsonrpc":"2.0","id":null,"result":null}`,
		`{"jsonrpc":"2.0","id":1,"result":{}}`,
		`{"jsonrpc":"2.0","id":"response","error":{"code":-32600,"message":"error"}}`,
		`{"error":"unsupported response shape"}`,
	} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			srv, replies, err := runRawInput(t, rawFrame(body)+rawFrame(validInitialize))
			if err != nil {
				t.Fatalf("Serve: %v", err)
			}
			if !srv.initialized || len(replies) != 1 || string(replies[0].ID) != `"recovery"` || replies[0].Error != nil {
				t.Fatalf("replies = %+v, want only initialization", replies)
			}
		})
	}
}

func TestFramingFailuresAreFatal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		want        error
	}{
		{"invalid length", "Content-Length: nope\r\n\r\n" + rawFrame(validInitialize), nil},
		{"negative length", "Content-Length: -1\r\n\r\n" + rawFrame(validInitialize), nil},
		{"missing length", "Other: header\r\n\r\n" + rawFrame(validInitialize), nil},
		{"truncated body", "Content-Length: 20\r\n\r\n{}", io.ErrUnexpectedEOF},
		{"absent body", "Content-Length: 20\r\n\r\n", io.ErrUnexpectedEOF},
		{"partial header", "Content-Length: 20", io.ErrUnexpectedEOF},
		{"unfinished headers", "Content-Length: 20\r\n", io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv, replies, err := runRawInput(t, tc.input)
			if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
				t.Fatalf("Serve = %v, want fatal framing/read failure %v", err, tc.want)
			}
			if len(replies) != 0 || srv.initialized {
				t.Fatalf("framing failure dispatched a message: %+v", replies)
			}
		})
	}
}

func TestValidRequestIDsArePreserved(t *testing.T) {
	t.Parallel()
	for _, id := range []string{`"9"`, `9`, `-9`, `""`, `"\u0039"`, `2147483647`, `-2147483648`, `"2147483648"`, `"-2147483649"`} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			_, replies, err := runRawInput(t, rawFrame(`{"jsonrpc":"2.0","id":`+id+`,"method":"unknown/method"}`)+rawFrame(validInitialize))
			if err != nil {
				t.Fatal(err)
			}
			if len(replies) != 2 {
				t.Fatalf("replies = %+v", replies)
			}
			assertRPCError(t, replies[0], id, -32002)
		})
	}
}

type failingOutputTransport struct {
	io.Reader
	err       error
	failAfter int
	writes    int
}

func (t *failingOutputTransport) Write(p []byte) (int, error) {
	t.writes++
	if t.writes > t.failAfter {
		return 0, t.err
	}
	return len(p), nil
}

func TestRejectionWriteFailuresPropagate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input      string
		successfulWrites int
	}{
		{"parse error", rawFrame(`{`), 0},
		{"invalid envelope", rawFrame(`[]`), 0},
		{"pre-initialization", rawFrame(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`), 0},
		{"invalid parameters", rawFrame(`{"jsonrpc":"2.0","id":2,"method":"initialize","params":false}`), 0},
		{"duplicate initialization", rawFrame(validInitialize) + rawFrame(validInitialize), 1},
		{"post-shutdown", rawFrame(validInitialize) + rawFrame(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`) + rawFrame(validInitialize), 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writeErr := errors.New("output unavailable")
			transport := &failingOutputTransport{Reader: strings.NewReader(tc.input + rawFrame(validInitialize)), err: writeErr, failAfter: tc.successfulWrites}
			srv := New(transport, nil, nil, nil)
			if err := srv.Serve(); !errors.Is(err, writeErr) {
				t.Fatalf("Serve = %v, want output error", err)
			}
			if transport.writes != tc.successfulWrites+1 {
				t.Fatalf("write attempts = %d, want no retry", transport.writes)
			}
		})
	}
}

func TestAsyncRejectionWriteFailureIsLogged(t *testing.T) {
	t.Parallel()
	input := rawFrame(validInitialize) + rawFrame(`{"jsonrpc":"2.0","id":2,"method":"unknown/method"}`) + rawFrame(`{"jsonrpc":"2.0","method":"$test/flushRequests"}`)
	writeErr := errors.New("output unavailable")
	transport := &failingOutputTransport{Reader: strings.NewReader(input), err: writeErr, failAfter: 1}
	var logs bytes.Buffer
	srv := New(transport, nil, nil, nil, WithLogger(observability.New(&logs)))
	if err := srv.Serve(); err != nil {
		t.Fatalf("Serve = %v, want async failure logged", err)
	}
	if transport.writes != 2 || srv.registry.lookup("2") != nil {
		t.Fatalf("writes = %d, want one terminal attempt and registry cleanup", transport.writes)
	}
	for _, want := range []string{"response write failed", "output unavailable", "method=unknown/method", "id=2"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log missing %q: %s", want, logs.String())
		}
	}
}

func TestInvalidNotificationsRemainSilentAfterInitialization(t *testing.T) {
	t.Parallel()
	input := rawFrame(validInitialize)
	for _, body := range []string{
		`{"jsonrpc":"2.0","method":"unknown/method"}`,
		`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":false}`,
		`{"jsonrpc":"2.0","method":"textDocument/didChange","params":false}`,
		`{"jsonrpc":"2.0","method":"textDocument/didClose","params":false}`,
		`{"jsonrpc":"2.0","method":"initialize","params":false}`,
		`{"method":"exit"}`,
		`{"jsonrpc":"1.0","method":"exit"}`,
		`{"jsonrpc":"2.0","id":null,"result":null}`,
	} {
		input += rawFrame(body)
	}
	input += rawFrame(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`)
	srv, replies, err := runRawInput(t, input)
	if err != nil {
		t.Fatal(err)
	}
	if !srv.shuttingDown || len(replies) != 2 || string(replies[0].ID) != `"recovery"` || string(replies[1].ID) != `2` || replies[1].Error != nil {
		t.Fatalf("replies = %+v, want only initialization and shutdown", replies)
	}
}
