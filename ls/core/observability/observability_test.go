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

package observability

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

func fakeIO(buf *bytes.Buffer) pal.IO {
	return pal.IO{
		Stderr: func(p []byte) (int, error) { return buf.Write(p) },
	}
}

func TestNewWritesToStderrByDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := New(fakeIO(&buf), pal.OS{})
	logger.Info("hello", "key", "value")
	out := buf.String()
	if !strings.Contains(out, "hello") || !strings.Contains(out, "key=value") {
		t.Fatalf("stderr output = %q, want it to contain message and attrs", out)
	}
}

func TestNewDefaultLevelIsInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := New(fakeIO(&buf), pal.OS{})
	logger.Debug("should not appear")
	if buf.Len() != 0 {
		t.Fatalf("stderr output = %q, want empty (Debug below default Info level)", buf.String())
	}
	logger.Info("should appear")
	if buf.Len() == 0 {
		t.Fatal("stderr output empty, want Info-level record")
	}
}

func TestNewNoopDiscardsOutputWithoutPAL(t *testing.T) {
	logger := NewNoop()
	// Must not panic even though no PAL was supplied.
	logger.Info("hello")
	logger.Warn("world")
}

func TestWithFileSinkWritesThroughFS(t *testing.T) {
	var buf bytes.Buffer
	fs := pal.FS{
		AppendFile: func(path string, data []byte) error {
			if path != "/tmp/ls.log" {
				t.Fatalf("AppendFile path = %q, want /tmp/ls.log", path)
			}
			buf.Write(data)
			return nil
		},
	}
	var stderrBuf bytes.Buffer
	logger := New(fakeIO(&stderrBuf), pal.OS{}, WithFileSink(fs, "/tmp/ls.log"))
	logger.Info("to file")
	if !strings.Contains(buf.String(), "to file") {
		t.Fatalf("file sink output = %q, want it to contain the message", buf.String())
	}
	if stderrBuf.Len() != 0 {
		t.Fatalf("stderr output = %q, want empty when WithFileSink is set", stderrBuf.String())
	}
}

func TestLoggerWithReturnsScopedLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(fakeIO(&buf), pal.OS{})
	scoped := logger.With("method", "initialize")
	scoped.Info("handling message")
	out := buf.String()
	if !strings.Contains(out, "method=initialize") {
		t.Fatalf("stderr output = %q, want method attr", out)
	}
}
