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
	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/runtime/internal/modules"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

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
	if callInfo.IsMethodCall {
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
	receiverObj := args[0].(*values.Object)
	lookupKey, found := receiverObj.MethodLookupKey(string(callInfo.Name))
	if !found {
		panic("function not found: " + callInfo.Name.Value())
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
	isResourceFnCall := callInfo == nil
	fn := reg.GetBIRFunction(lookupKey)
	if fn != nil {
		if !isResourceFnCall {
			callInfo.CachedBIRFunc = fn
		}
		return executeFunction(ctx, fn, args, nil), nil
	}
	externFn := reg.GetNativeFunction(lookupKey)
	if externFn != nil {
		if !isResourceFnCall {
			callInfo.CachedNativeFunc = externFn.Impl
		}
		return externFn.Impl(ctx, args)
	}
	// In resource function case we have already validated function exists using RTable
	panic(values.NewErrorWithMessage("function not found: " + callInfo.Name.Value()))
}

func execResourceCall(ctx *extern.Context, instr *bir.ResourceFunctionCall, frame *Frame) *bir.BIRBasicBlock {
	receiver := getOperandValue(ctx, &instr.Receiver, frame).(*values.Object)
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
	tyCx := ctx.TypeCtx
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
		// FIXME: https://github.com/ballerina-platform/ballerina-lang-go/issues/471
		listDefn := semtypes.NewListDefinition()
		restListTy := listDefn.DefineListTypeWrapped(ctx.Env.TypeEnv, []semtypes.SemType{}, 0, match.RestSegmentTy, semtypes.CellMutability_CELL_MUT_NONE)
		atomic := semtypes.ToListAtomicType(ctx.TypeCtx, restListTy)
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
	var parentFrame *Frame
	if fnValue.ParentFrame != nil {
		parentFrame = fnValue.ParentFrame.(*Frame)
	}
	reg := ctx.Env.Registry.(*modules.Registry)
	if builtin := reg.GetRuntimeBuiltin(lookupKey); builtin != nil {
		var err error
		result, err = builtin(ctx, args)
		if err != nil {
			panicWithExternError(err)
		}
	} else if fn := reg.GetBIRFunction(lookupKey); fn != nil {
		result = executeFunction(ctx, fn, args, parentFrame)
	} else if externFn := reg.GetNativeFunction(lookupKey); externFn != nil {
		var err error
		result, err = externFn.Impl(ctx, args)
		if err != nil {
			panicWithExternError(err)
		}
	} else {
		panic("function not found: " + callInfo.Name.Value())
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
