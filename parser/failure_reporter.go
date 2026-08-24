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
	"sync/atomic"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type parserFailureReporter struct {
	ctx      *compilercontext.CompilerContext
	location diagnostics.Location
	failed   atomic.Bool
}

func newParserFailureReporter(ctx *compilercontext.CompilerContext, fileName string) *parserFailureReporter {
	return &parserFailureReporter{
		ctx:      ctx,
		location: diagnostics.NewLocation(ctx.DiagnosticEnv(), fileName, 0, 0),
	}
}

func (r *parserFailureReporter) internalError(message string) {
	if r == nil || !r.failed.CompareAndSwap(false, true) {
		return
	}
	r.ctx.InternalError(message, r.location)
}

func (r *parserFailureReporter) unimplemented(message string) {
	if r == nil || !r.failed.CompareAndSwap(false, true) {
		return
	}
	r.ctx.Unimplemented(message, r.location)
}

func (r *parserFailureReporter) hasFailed() bool {
	return r != nil && r.failed.Load()
}

func failedToken() st.STToken {
	return st.CreateToken(st.EOF_TOKEN, st.CreateEmptyNodeList(), st.CreateEmptyNodeList())
}

func zeroValue[T any]() T {
	var value T
	return value
}
