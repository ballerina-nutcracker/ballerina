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

// Package observability provides the language server's structured-logging
// facade: a thin, PAL-mediated wrapper around log/slog. It writes exclusively
// through PAL (pal.IO.Stderr by default, or an optional PAL-FS-backed file
// sink via WithFileSink) — never bare os/stdlib log — per ls/AGENTS.md's PAL
// mandate. Metrics/telemetry are explicitly out of scope (ticket 38/41
// ADRs); this package is logging only.
package observability

import (
	"io"
	"log/slog"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

// defaultLevel is the hardcoded default log level. Env-var-based level
// control is deferred to a follow-up ticket (ticket 41 ADR, decision 8).
const defaultLevel = slog.LevelInfo

// Logger is the language server's structured logger. It wraps *slog.Logger
// so callers get slog's Info/Warn/Error/Debug/With API for free.
type Logger struct {
	*slog.Logger

	// os is reserved for a future env-gated level override (ticket 41 ADR,
	// decision 8 defers this); unused today.
	os pal.OS
}

// config accumulates constructor options before the handler is built.
type config struct {
	handler slog.Handler
}

// Option configures a Logger at construction time.
type Option func(cfg *config)

// writerFunc adapts a PAL sink function (matching pal.IO.Stderr's signature)
// to io.Writer, the only PAL boundary a slog.Handler needs.
type writerFunc func(p []byte) (int, error)

func (w writerFunc) Write(p []byte) (int, error) { return w(p) }

// New builds a Logger writing to pal.IO.Stderr by default, at a hardcoded
// slog.LevelInfo. Pass WithFileSink to redirect output to a PAL-backed file
// instead. os is accepted for parity with the narrow PAL surface this facade
// is scoped to; it carries no behavior yet.
func New(io pal.IO, os pal.OS, opts ...Option) *Logger {
	cfg := &config{
		handler: slog.NewTextHandler(writerFunc(io.Stderr), &slog.HandlerOptions{Level: defaultLevel}),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Logger{Logger: slog.New(cfg.handler), os: os}
}

// NewNoop builds a Logger that discards everything it writes. It needs no PAL
// and is the default for tests and any constructor that does not opt into a
// real logger (e.g. server.New, event.New's production adapter default).
func NewNoop() *Logger {
	return &Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// WithFileSink redirects the logger's output to path, appended through
// pal.FS.AppendFile (one append call per record; no open file handle or
// Close lifecycle). It replaces the default stderr sink rather than teeing to
// both. Nothing in this ticket wires WithFileSink as active by default —
// pal.IO.Stderr remains the only sink used unless a caller opts in.
func WithFileSink(fs pal.FS, path string) Option {
	return func(cfg *config) {
		sink := writerFunc(func(p []byte) (int, error) {
			if err := fs.AppendFile(path, p); err != nil {
				return 0, err
			}
			return len(p), nil
		})
		cfg.handler = slog.NewTextHandler(sink, &slog.HandlerOptions{Level: defaultLevel})
	}
}

// With returns a child Logger with the given attributes attached to every
// subsequent record, mirroring slog.Logger.With but preserving the
// *observability.Logger type so it keeps flowing through this package's API.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{Logger: l.Logger.With(args...), os: l.os}
}
