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
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
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
	go runStrand(parent, seed, impl, args, ch)
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

type bootstrappedCall struct {
	handle *InvokableHandle
	args   []values.BalValue
}

func invokeBootstrappedCall(ctx *extern.Context, call *bootstrappedCall) values.BalValue {
	result, err := call.handle.invoke(ctx, call.args)
	if err != nil {
		panic(err)
	}
	return result
}

func startFuture(parent *extern.Context, call *bootstrappedCall, spawnFrames []callStackEntry, isolated bool, ty semtypes.SemType) *values.Future {
	future := values.NewFuture(ty)
	ctx := contextForStartedStrand(parent, spawnFrames, isolated)
	run := func() {
		var result values.BalValue
		defer func() {
			panicValue := recover()
			if panicValue != nil {
				panicValue = capturePanicStack(panicValue, formatCallStack(ctx.CallStack.(*callStack)))
				ctx.ReleaseAllHeldLocks()
			}
			future.Complete(result, panicValue)
		}()
		result = invokeBootstrappedCall(ctx, call)
	}
	if isolated {
		go run()
	} else {
		continuation := ctx.Schedule()
		go func() {
			<-continuation
			run()
			ctx.Complete()
		}()
	}
	return future
}

func contextForStartedStrand(parent *extern.Context, seed []callStackEntry, isolated bool) *extern.Context {
	var ctx *extern.Context
	if isolated {
		ctx = extern.CreateContext(parent.Env)
	} else {
		ctx = extern.CreateContextInSameThread(parent)
	}
	elems := make([]callStackEntry, len(seed), len(seed)+32)
	copy(elems, seed)
	ctx.CallStack = &callStack{elements: elems}
	return ctx
}

func runStrand(parent *extern.Context, seed []callStackEntry, h *InvokableHandle,
	args []values.BalValue, ch chan<- values.BalValue,
) {
	ctx := contextForStartedStrand(parent, seed, true)

	defer close(ch)
	v, err := h.invoke(ctx, args)
	if err != nil {
		ch <- values.NewErrorWithMessage(err.Error())
		return
	}
	ch <- v
}
