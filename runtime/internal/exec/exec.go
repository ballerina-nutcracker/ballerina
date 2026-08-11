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

// Package exec defines parts of the runtime related to execution of instructions.
// This is an internal runtime package and only runtime should depend on this package
package exec

import "github.com/ballerina-nutcracker/ballerina/runtime/extern"

// CreateContext builds an extern.Context wired with a fresh call stack
// ready to execute BIR functions. Runtime callers must use this rather
// than extern.CreateContext directly so the call stack is populated. This is
// the unpooled path — see extern.CreateContext for when to use it versus
// Runtime.AcquirePooledContext.
func CreateContext(env *extern.Env) *extern.Context {
	ctx := extern.CreateContext(env)
	ctx.CallStack = &callStack{elements: make([]callStackEntry, 0, 32)}
	return ctx
}

// ResetContextForReuse clears a pooled context's per-request state before it
// goes back into the pool: it truncates the call stack and resets
// per-request state, including clearing TypeCtx if one was built
// (extern.ResetForReuse). Pairs with Runtime.ReleasePooledContext.
func ResetContextForReuse(ctx *extern.Context) {
	ctx.ResetForReuse()
	if cs, ok := ctx.CallStack.(*callStack); ok {
		// Zero the whole backing array before truncating: a pooled context
		// outlives the request, so leftover entries would keep request-local
		// *Frame graphs reachable and unreclaimable until a deeper later
		// request overwrote each slot.
		clear(cs.elements[:cap(cs.elements)])
		cs.elements = cs.elements[:0]
	}
}
