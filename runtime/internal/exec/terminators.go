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
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/runtime/internal/modules"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func execStartAction(ctx *extern.Context, action *bir.StartAction, frame *Frame) *bir.BIRBasicBlock {
	if ctx.HoldsLock() {
		panic(values.NewErrorWithMessage("attempted strand start while holding a lock"))
	}
	call := bootstrapCall(ctx, &action.Call, frame)
	spawnFrames := snapshotSpawnFrames(ctx.CallStack.(*callStack))
	spawnFrames[len(spawnFrames)-1].location = action.Pos
	future := startFuture(ctx, call, spawnFrames, action.IsIsolated, action.LhsOp.VariableDcl.GetType())
	setOperandValue(ctx, action.LhsOp, frame, future)
	return action.ThenBB
}

func bootstrapCall(ctx *extern.Context, call *bir.CallSite, frame *Frame) *bootstrappedCall {
	args := extractArgs(ctx, call.Args, frame)
	var handle *InvokableHandle
	var err error
	switch call.Kind {
	case bir.CallKindFunctionPointer:
		fn := getOperandValue(ctx, call.FpOperand, frame).(*values.Function)
		handle, err = NewFunctionValueHandle(ctx.Env, fn)
	case bir.CallKindMethod:
		receiver := getOperandValue(ctx, call.Receiver, frame).(*values.Object)
		lookupKey, found := receiver.MethodLookupKey(call.Name.Value())
		if !found {
			panic(values.NewErrorWithMessage("function not found: " + call.Name.Value()))
		}
		handle, err = newLookupKeyHandle(ctx.Env, lookupKey, nil)
	case bir.CallKindResource:
		receiver := getOperandValue(ctx, call.Receiver, frame).(*values.Object)
		path := extractArgs(ctx, call.PathSegments, frame)
		impl, ok := LookupResourceMethod(ctx, receiver, call.MethodName, path)
		if !ok {
			panic(values.NewErrorWithMessage("no matching resource method"))
		}
		handle = impl.(*InvokableHandle)
	case bir.CallKindFunction:
		handle, err = newLookupKeyHandle(ctx.Env, call.FunctionLookupKey, nil)
	default:
		panic(fmt.Sprintf("unexpected call kind: %d", call.Kind))
	}
	if err != nil {
		panic(err)
	}
	return &bootstrappedCall{handle: handle, args: args}
}

func execSingleWaitAction(ctx *extern.Context, action *bir.SingleWaitAction, frame *Frame) *bir.BIRBasicBlock {
	future := getOperandValue(ctx, &action.Future, frame).(*values.Future)
	setOperandValue(ctx, action.LhsOp, frame, waitFuture(ctx, future))
	return action.ThenBB
}

func execAlternateWaitAction(ctx *extern.Context, action *bir.AlternateWaitAction, frame *Frame) *bir.BIRBasicBlock {
	futures := make([]*values.Future, len(action.Futures))
	for i := range action.Futures {
		futures[i] = getOperandValue(ctx, &action.Futures[i], frame).(*values.Future)
	}
	setOperandValue(ctx, action.LhsOp, frame, waitAnyFuture(ctx, futures))
	return action.ThenBB
}

func execMultipleWaitAction(ctx *extern.Context, action *bir.MultipleWaitAction, frame *Frame) *bir.BIRBasicBlock {
	if len(action.Futures) != len(action.FieldNames) {
		// I dont' think this can happen at runtime, if so that's a bir gen error
		panic("multiple wait field and future count mismatch")
	}
	futures := make([]*values.Future, len(action.Futures))
	for i := range action.Futures {
		futures[i] = getOperandValue(ctx, &action.Futures[i], frame).(*values.Future)
	}
	results := waitAllFutures(ctx, futures)
	entries := make([]values.MapEntry, len(results))
	for i, result := range results {
		entries[i] = values.MapEntry{Key: action.FieldNames[i], Value: result}
	}
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), action.Type)
	if atomic == nil {
		panic("multiple wait result type has no mapping atomic representation")
	}
	setOperandValue(ctx, action.LhsOp, frame, values.NewMap(action.Type, atomic, false, entries))
	return action.ThenBB
}

func waitAllFutures(ctx *extern.Context, futures []*values.Future) []values.BalValue {
	results := make([]values.BalValue, len(futures))
	completed := make([]bool, len(futures))
	remaining := len(futures)
	for i, future := range futures {
		if future.Claim() {
			continue
		}
		// I am not entirely sure this is the correct behaviour. This means you can multiple wait on a given future any number of
		// of times as long as you have another fresh future.
		results[i] = values.NewErrorWithMessage("multiple waits on the same future is not allowed")
		completed[i] = true
		remaining--
	}
	for remaining > 0 {
		<-ctx.Yield()
		for i, future := range futures {
			if completed[i] || !future.IsComplete() {
				continue
			}
			results[i] = future.GetClaimed()
			completed[i] = true
			remaining--
		}
	}
	return results
}

func waitFuture(ctx *extern.Context, future *values.Future) values.BalValue {
	for {
		<-ctx.Yield()
		if future.IsComplete() {
			return future.Get()
		}
	}
}

func waitAnyFuture(ctx *extern.Context, futures []*values.Future) values.BalValue {
	completed := make([]bool, len(futures))
	remaining := len(futures)
	var lastError *values.Error
	for remaining > 0 {
		<-ctx.Yield()
		idx, found := earliestCompletedFuture(futures, completed)
		if !found {
			continue
		}
		future := futures[idx]
		result := future.Get()
		err, ok := result.(*values.Error)
		if !ok {
			return result
		}
		lastError = err
		for i, candidate := range futures {
			if !completed[i] && candidate == future {
				completed[i] = true
				remaining--
			}
		}
	}
	return lastError
}

func earliestCompletedFuture(futures []*values.Future, completed []bool) (int, bool) {
	earliest := 0
	found := false
	for i, future := range futures {
		if completed[i] || !future.IsComplete() {
			continue
		}
		if !found || futures[earliest].IsAfter(future) {
			earliest = i
			found = true
		}
	}
	return earliest, found
}

func execBranch(ctx *extern.Context, branchTerm *bir.Branch, frame *Frame) *bir.BIRBasicBlock {
	if getOperandValue(ctx, branchTerm.Op, frame).(bool) {
		return branchTerm.TrueBB
	}
	return branchTerm.FalseBB
}

func execCall(ctx *extern.Context, callInfo *bir.Call, frame *Frame) *bir.BIRBasicBlock {
	args := extractArgs(ctx, callInfo.Args, frame)
	result := executeCall(ctx, callInfo, args)
	if callInfo.LhsOp != nil {
		setOperandValue(ctx, callInfo.LhsOp, frame, result)
	}
	return callInfo.ThenBB
}

func executeCall(ctx *extern.Context, callInfo *bir.Call, args []values.BalValue) values.BalValue {
	if callInfo.Kind == bir.CallKindMethod {
		return dispatchMethodCall(ctx, callInfo, args)
	}
	if callInfo.CachedBIRFunc != nil {
		return executeFunction(ctx, callInfo.CachedBIRFunc, args, nil)
	}
	if callInfo.CachedNativeFunc != nil {
		result, err := callInfo.CachedNativeFunc(ctx, args)
		if err != nil {
			panicWithExternError(err)
		}
		return result
	}
	result, err := lookupAndExecute(ctx, callInfo, args, callInfo.FunctionLookupKey)
	if err != nil {
		panicWithExternError(err)
	}
	return result
}

func dispatchMethodCall(ctx *extern.Context, callInfo *bir.Call, args []values.BalValue) values.BalValue {
	receiver := args[0].(*values.Object)
	lookupKey, found := receiver.MethodLookupKey(callInfo.Name.Value())
	if !found {
		panic(values.NewErrorWithMessage("function not found: " + callInfo.Name.Value()))
	}
	// The same call site can be polymorphic across executions (e.g., iterating over a list
	// of objects with different concrete types). Cache only when it matches the receiver.
	if callInfo.CachedMethodLookupKey == lookupKey {
		if callInfo.CachedBIRFunc != nil {
			return executeFunction(ctx, callInfo.CachedBIRFunc, args, nil)
		}
		if callInfo.CachedNativeFunc != nil {
			result, err := callInfo.CachedNativeFunc(ctx, args)
			if err != nil {
				panicWithExternError(err)
			}
			return result
		}
	}
	callInfo.CachedBIRFunc = nil
	callInfo.CachedNativeFunc = nil
	callInfo.CachedMethodLookupKey = lookupKey
	result, err := lookupAndExecute(ctx, callInfo, args, lookupKey)
	if err != nil {
		panicWithExternError(err)
	}
	return result
}

func lookupAndExecute(ctx *extern.Context, callInfo *bir.Call, args []values.BalValue, lookupKey string) (values.BalValue, error) {
	reg := ctx.Env.Registry.(*modules.Registry)
	if builtin := reg.GetRuntimeBuiltin(lookupKey); builtin != nil {
		return builtin(ctx, args)
	}
	fn := reg.GetBIRFunction(lookupKey)
	if fn != nil {
		callInfo.CachedBIRFunc = fn
		return executeFunction(ctx, fn, args, nil), nil
	}
	externFn := reg.GetNativeFunction(lookupKey)
	if externFn != nil {
		callInfo.CachedNativeFunc = externFn.Impl
		return externFn.Impl(ctx, args)
	}
	panic(values.NewErrorWithMessage("function not found: " + callInfo.Name.Value()))
}

func execResourceCall(ctx *extern.Context, instr *bir.ResourceFunctionCall, frame *Frame) *bir.BIRBasicBlock {
	receiver := getOperandValue(ctx, instr.Receiver, frame).(*values.Object)
	pathVals := extractArgs(ctx, instr.PathSegments, frame)
	impl, ok := LookupResourceMethod(ctx, receiver, instr.MethodName, pathVals)
	if !ok {
		panic(values.NewErrorWithMessage("no matching resource method"))
	}
	argVals := extractArgs(ctx, instr.Args, frame)
	result, err := Invoke(ctx, impl, argVals)
	if err != nil {
		panicWithExternError(err)
	}
	if instr.LhsOp != nil {
		setOperandValue(ctx, instr.LhsOp, frame, result)
	}
	return instr.ThenBB
}

func resourceFnCandidates(ctx *extern.Context, receiver *values.Object, methodName string, pathVals []values.BalValue) []*values.ResourceEntry {
	candidates, ok := receiver.ResourceEntries(methodName)
	if !ok {
		return nil
	}
	shapes := make([]semtypes.SemType, len(pathVals))
	for i, v := range pathVals {
		shapes[i] = values.SemTypeForValue(v)
	}
	var matches []*values.ResourceEntry
	for i := range candidates {
		if resourcePathMatches(ctx, &candidates[i], shapes) {
			matches = append(matches, &candidates[i])
		}
	}
	return matches
}

func resourcePathMatches(ctx *extern.Context, entry *values.ResourceEntry, shapes []semtypes.SemType) bool {
	requiredLen := len(entry.PathSegments)
	if len(shapes) < requiredLen {
		return false
	}
	tyCx := ctx.TypeCtx()
	for i := range requiredLen {
		if !semtypes.IsSubtype(tyCx, shapes[i], entry.PathSegments[i].Ty) {
			return false
		}
	}
	if len(shapes) == requiredLen {
		return true
	}
	if semtypes.IsNever(entry.RestSegmentTy) {
		return false
	}
	for i := requiredLen; i < len(shapes); i++ {
		if !semtypes.IsSubtype(tyCx, shapes[i], entry.RestSegmentTy) {
			return false
		}
	}
	return true
}

func buildResourceCallArgs(ctx *extern.Context, receiver *values.Object, match *values.ResourceEntry, pathVals, argVals []values.BalValue) []values.BalValue {
	k := len(match.PathSegments)
	result := make([]values.BalValue, 0, 1+len(pathVals)+len(argVals))
	result = append(result, receiver)
	for i := range k {
		if _, isLiteral := values.LiteralPathSegment(match.PathSegments[i]); !isLiteral {
			result = append(result, pathVals[i])
		}
	}
	if !semtypes.IsNever(match.RestSegmentTy) {
		restVals := pathVals[k:]
		// FIXME: https://github.com/ballerina-nutcracker/ballerina/issues/471
		listDefn := semtypes.NewListDefinition()
		restListTy := listDefn.Define(ctx.TypeEnv(), nil, semtypes.ListRest(match.RestSegmentTy),
			semtypes.ListMutability(semtypes.CellMutabilityNone))
		atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), restListTy)
		if atomic == nil {
			panic("rest segment type has no list atomic representation")
		}
		initial := make([]values.BalValue, len(restVals))
		copy(initial, restVals)
		restList := values.NewList(restListTy, atomic, true, nil, len(restVals), initial)
		result = append(result, restList)
	}
	result = append(result, argVals...)
	return result
}

func execFpCall(ctx *extern.Context, callInfo *bir.Call, frame *Frame) *bir.BIRBasicBlock {
	args := extractArgs(ctx, callInfo.Args, frame)
	fnValue := getOperandValue(ctx, callInfo.FpOperand, frame).(*values.Function)
	lookupKey := fnValue.LookupKey
	var result values.BalValue
	reg := ctx.Env.Registry.(*modules.Registry)
	if builtin := reg.GetRuntimeBuiltin(lookupKey); builtin != nil {
		var err error
		result, err = builtin(ctx, args)
		if err != nil {
			panicWithExternError(err)
		}
	} else if fn := reg.GetBIRFunction(lookupKey); fn != nil {
		result = executeFunction(ctx, fn, args, parentFrameFromFunctionValue(fnValue))
	} else if externFn := reg.GetNativeFunction(lookupKey); externFn != nil {
		var err error
		result, err = externFn.Impl(ctx, args)
		if err != nil {
			panicWithExternError(err)
		}
	} else {
		panic(values.NewErrorWithMessage("function not found: " + lookupKey))
	}
	if callInfo.LhsOp != nil {
		setOperandValue(ctx, callInfo.LhsOp, frame, result)
	}
	return callInfo.ThenBB
}

func extractArgs(ctx *extern.Context, args []bir.BIROperand, frame *Frame) []values.BalValue {
	values := make([]values.BalValue, len(args))
	for i, op := range args {
		values[i] = getOperandValue(ctx, &op, frame)
	}
	return values
}

func execPanic(ctx *extern.Context, panicTerm *bir.Panic, frame *Frame) *bir.BIRBasicBlock {
	errVal := getOperandValue(ctx, panicTerm.ErrorOp, frame)
	panic(errVal)
}
