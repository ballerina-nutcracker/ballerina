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

package native

import (
	"errors"
	"io"
	"testing"
)

// onceFailingReader returns data together with err on its first Read and io.EOF after that,
// as net/http's server request body does when the client sends less than its Content-Length.
type onceFailingReader struct {
	data string
	err  error
	done bool
}

func (r *onceFailingReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	return copy(p, r.data), r.err
}

func (r *onceFailingReader) Close() error { return nil }

// TestReadBlockKeepsReadError covers a request body truncated by a client that disconnects,
// which this module's own client cannot send: the bytes that arrived are returned first and
// the read error next, instead of the stream ending cleanly.
func TestReadBlockKeepsReadError(t *testing.T) {
	t.Parallel()
	body := &streamBody{reader: &onceFailingReader{data: "hello", err: io.ErrUnexpectedEOF}}
	block, err := body.readBlock(8192)
	if err != nil || string(block) != "hello" {
		t.Fatalf("first read: got %q, %v; want \"hello\"", block, err)
	}
	block, err = body.readBlock(8192)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("second read: got %q, %v; want %v", block, err, io.ErrUnexpectedEOF)
	}
	if block, err = body.readBlock(8192); block != nil || err != nil {
		t.Fatalf("third read: got %q, %v; want the end of the stream", block, err)
	}
}

// TestSSEFramerDeliversEventsBeforeReadError covers a reader that returns the last complete
// event together with an error, as a decompressing reader can: the event is handed on first.
func TestSSEFramerDeliversEventsBeforeReadError(t *testing.T) {
	t.Parallel()
	reset := errors.New("connection reset")
	framer := newSSEFramer(&onceFailingReader{data: "data: a\n\ndata: b\n\ndata: c", err: reset})
	for _, want := range []string{"data: a\n\n", "data: b\n\n"} {
		event, err := framer.next()
		if err != nil || string(event) != want {
			t.Fatalf("got %q, %v; want %q", event, err, want)
		}
	}
	if event, err := framer.next(); !errors.Is(err, reset) {
		t.Fatalf("got %q, %v; want %v", event, err, reset)
	}
}
