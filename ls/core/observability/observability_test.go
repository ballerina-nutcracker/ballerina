// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package observability

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNewWritesToSuppliedWriter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := New(&buf)
	logger.Info("hello", "key", "value")
	out := buf.String()
	for _, want := range []string{"time=", "level=INFO", "msg=hello", "key=value"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	}
}

func TestNewDefaultLevelIsInfo(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := New(&buf)
	logger.Debug("should not appear")
	if buf.Len() != 0 {
		t.Fatalf("Debug output = %q", buf.String())
	}
	logger.Info("should appear")
	if buf.Len() == 0 {
		t.Fatal("missing Info record")
	}
}

func TestNewNoopDiscardsOutput(t *testing.T) {
	t.Parallel()
	NewNoop().Info("hello")
	New(nil).Warn("world")
}

func TestLoggerWithReturnsScopedLogger(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	New(&buf).With("method", "initialize").Info("handling message")
	if !strings.Contains(buf.String(), "method=initialize") {
		t.Fatalf("output = %q", buf.String())
	}
}

type ownedWriter struct {
	bytes.Buffer
	closed bool
	err    error
	calls  int
}

func (w *ownedWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.err != nil {
		return 0, w.err
	}
	return w.Buffer.Write(p)
}
func (w *ownedWriter) Close() error { w.closed = true; return nil }

func TestLoggerLeavesWriterOwnershipToHost(t *testing.T) {
	t.Parallel()
	writer := &ownedWriter{}
	New(writer).With("method", "shutdown").Info("drained")
	if writer.closed || writer.Len() == 0 {
		t.Fatalf("closed=%v output=%q", writer.closed, writer.String())
	}
}

func TestLoggerWriterErrorUsesSlogPolicy(t *testing.T) {
	t.Parallel()
	writer := &ownedWriter{err: errors.New("injected write failure")}
	New(writer).Error("not redirected")
	if writer.calls != 1 || writer.closed {
		t.Fatalf("calls=%d closed=%v", writer.calls, writer.closed)
	}
}
