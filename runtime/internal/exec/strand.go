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

package exec

import (
	"ballerina/runtime/extern"
	"ballerina/values"
)

// StartMethod is the dispatch hook backing Context.StartMethod. It snapshots
// the parent's spawn-site frames (functionKey + location frozen at call
// time) and spawns a goroutine that runs the handle on a fresh Context whose
// call stack is seeded with that snapshot.
//
// Panics raised inside the started strand are *not* recovered here; they
// propagate as ordinary Go panics, matching the semantics of an uncaught
// Ballerina panic in a started strand. Only the explicit (nil, err) return
// from the handle is converted into a *values.Error and delivered on the
// channel.
func StartMethod(parent *extern.Context, h any, args []values.BalValue) (<-chan values.BalValue, error) {
	ch := make(chan values.BalValue, 1)
	impl := h.(*InvokableHandle)
	seed := snapshotSpawnFrames(parent.CallStack.(*callStack))
	go runStrand(parent.Env, seed, impl, args, ch)
	return ch, nil
}

// snapshotSpawnFrames returns a value-copy of every frame currently on cs so
// the started strand can carry parent context into its own call stack
// without aliasing the parent's mutable call-stack entries.
func snapshotSpawnFrames(cs *callStack) []callStackEntry {
	src := cs.Entries()
	out := make([]callStackEntry, len(src))
	for i, e := range src {
		frame := &Frame{}
		if e.frame != nil {
			frame.SetFunctionKey(e.frame.FunctionKey())
		}
		out[i] = callStackEntry{frame: frame, location: e.location}
	}
	return out
}

func runStrand(env *extern.Env, seed []callStackEntry, h *InvokableHandle,
	args []values.BalValue, ch chan<- values.BalValue,
) {
	ctx := CreateContext(env)
	elems := make([]callStackEntry, len(seed), len(seed)+32)
	copy(elems, seed)
	ctx.CallStack = &callStack{elements: elems}

	defer close(ch)
	v, err := h.invoke(ctx, args)
	if err != nil {
		ch <- values.NewErrorWithMessage(err.Error())
		return
	}
	ch <- v
}
