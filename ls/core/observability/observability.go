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

// Package observability provides the language server's structured slog facade.
// The host selects and owns the output writer, including any file lifecycle.
package observability

import (
	"io"
	"log/slog"
)

const defaultLevel = slog.LevelInfo

// Logger preserves slog's structured logging API and request correlation.
type Logger struct{ *slog.Logger }

// New writes Info and higher records to output. A nil writer discards output.
// The logger never closes output or redirects failed writes to another sink.
func New(output io.Writer) *Logger {
	if output == nil {
		output = io.Discard
	}
	return &Logger{Logger: slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: defaultLevel}))}
}

// NewNoop is the quiet default for services without an injected logger.
func NewNoop() *Logger { return New(io.Discard) }

// With returns a child logger carrying the supplied correlation attributes.
func (l *Logger) With(args ...any) *Logger { return &Logger{Logger: l.Logger.With(args...)} }
