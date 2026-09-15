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

package parser

import (
	"fmt"
	"io"
)

// DebugOptions selects which parser debug traces are emitted. All of them are no-ops
// unless the binary is built with the `debug` build tag.
type DebugOptions struct {
	DumpTokens     bool
	DumpSyntaxTree bool
	TraceRecovery  bool
}

// debugOutput is the sink the parser writes its enabled debug traces to.
type debugOutput struct {
	options DebugOptions
	writer  io.Writer
}

// newDebugOutput creates a debug sink for the given options, discarding output when
// writer is nil.
func newDebugOutput(options DebugOptions, writer io.Writer) *debugOutput {
	if writer == nil {
		writer = io.Discard
	}
	return &debugOutput{options: options, writer: writer}
}

// write emits message as a line when enabled and this is a debug build; the message is
// only built if it is actually written.
func (d *debugOutput) write(enabled bool, message func() string) {
	if parserDebugBuild && enabled {
		_, _ = fmt.Fprintln(d.writer, message())
	}
}
