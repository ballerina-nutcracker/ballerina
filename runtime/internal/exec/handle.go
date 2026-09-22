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
	"github.com/ballerina-nutcracker/ballerina/values"
)

// InvokableHandle is provides a unified representation that can be used to execute any function/method
// in runtime
type InvokableHandle struct {
	invoke func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error)
	// descriptor describes the parameters and the return type of the invokable.
	// It is nil when the invokable has no BIR declaration.
	descriptor *bir.BIRFunction
	// firstParam is the index of the first declared parameter within the
	// descriptor, skipping the parameters supplied by the invoke closure.
	firstParam int
}

func NewBIRHandle(fn *bir.BIRFunction) *InvokableHandle {
	return newBIRHandle(fn, nil)
}

func newBIRHandle(fn *bir.BIRFunction, parentFrame *Frame) *InvokableHandle {
	return newInvokableHandle(
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return executeFunction(ctx, fn, args, parentFrame), nil
		},
		fn,
		0,
	)
}

func newNativeHandle(fn extern.NativeFunc, descriptor *bir.BIRFunction) *InvokableHandle {
	return newInvokableHandle(
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return fn(ctx, args)
		},
		descriptor,
		0,
	)
}

// nativeHandleFor returns a handle for the extern function registered under
// lookupKey, or nil when the registry has no native implementation for it.
func nativeHandleFor(reg *modules.Registry, lookupKey string) *InvokableHandle {
	externFn := reg.GetNativeFunction(lookupKey)
	if externFn == nil {
		return nil
	}
	return newNativeHandle(externFn.Impl, reg.GetFunctionDescriptor(lookupKey))
}

func NewFunctionValueHandle(env *extern.Env, fnValue *values.Function) (*InvokableHandle, error) {
	return newLookupKeyHandle(env, fnValue.LookupKey, parentFrameFromFunctionValue(fnValue))
}

func newLookupKeyHandle(env *extern.Env, lookupKey string, parentFrame *Frame) (*InvokableHandle, error) {
	reg := env.Registry.(*modules.Registry)
	if builtin := reg.GetRuntimeBuiltin(lookupKey); builtin != nil {
		return newNativeHandle(builtin, reg.GetFunctionDescriptor(lookupKey)), nil
	}
	if fn := reg.GetBIRFunction(lookupKey); fn != nil {
		return newBIRHandle(fn, parentFrame), nil
	}
	if handle := nativeHandleFor(reg, lookupKey); handle != nil {
		return handle, nil
	}
	return nil, fmt.Errorf("function not found: %s", lookupKey)
}

func parentFrameFromFunctionValue(fnValue *values.Function) *Frame {
	if fnValue.ParentFrame == nil {
		return nil
	}
	return fnValue.ParentFrame.(*Frame)
}

func newResourceHandle(env *extern.Env, receiver *values.Object, match *values.ResourceEntry, path []values.BalValue) (*InvokableHandle, error) {
	target, err := newLookupKeyHandle(env, match.FunctionLookupKey, nil)
	if err != nil {
		return nil, err
	}
	return newInvokableHandle(
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			full := buildResourceCallArgs(ctx, receiver, match, path, args)
			return target.invoke(ctx, full)
		},
		target.descriptor,
		resourcePathParamCount(match),
	), nil
}

func newInvokableHandle(
	invoke func(*extern.Context, []values.BalValue) (values.BalValue, error),
	descriptor *bir.BIRFunction,
	firstParam int,
) *InvokableHandle {
	return &InvokableHandle{invoke: invoke, descriptor: descriptor, firstParam: firstParam}
}

// hasMaterializedSignature reports whether fn carries a signature that can be
// described. Native dependently typed functions do not, since their signature
// is only known at each call site, and they carry no locals or return variable.
func hasMaterializedSignature(fn *bir.BIRFunction) bool {
	return fn.ReturnVariable != nil
}

func describeFunctionSignature(fn *bir.BIRFunction, firstParam int) (extern.FunctionSignature, bool) {
	if firstParam > len(fn.RequiredParams) || !hasMaterializedSignature(fn) {
		return extern.FunctionSignature{}, false
	}
	paramLocalOffset := fn.ParamLocalVarOffset()
	params := make([]extern.Parameter, len(fn.RequiredParams)-firstParam)
	for i := firstParam; i < len(fn.RequiredParams); i++ {
		localIndex := paramLocalOffset + i
		if localIndex >= len(fn.LocalVars) {
			return extern.FunctionSignature{}, false
		}
		params[i-firstParam] = extern.Parameter{
			Name: fn.RequiredParams[i].Name.Value(),
			Type: fn.LocalVars[localIndex].Type,
		}
	}
	signature := extern.FunctionSignature{Params: params, ReturnType: fn.ReturnVariable.Type}
	if fn.RestParams != nil {
		restLocalIndex := paramLocalOffset + len(fn.RequiredParams)
		if restLocalIndex >= len(fn.LocalVars) {
			return extern.FunctionSignature{}, false
		}
		signature.RestParam = &extern.Parameter{
			Name: fn.RestParams.Name.Value(),
			Type: fn.LocalVars[restLocalIndex].Type,
		}
	}
	return signature, true
}

func describeFunctionMetadata(ctx *extern.Context, fn *bir.BIRFunction, firstParam int) (extern.FunctionMetadata, bool) {
	if firstParam > len(fn.RequiredParams) || !hasMaterializedSignature(fn) {
		return extern.FunctionMetadata{}, false
	}
	params := make([]extern.ParameterMetadata, len(fn.RequiredParams)-firstParam)
	for i := firstParam; i < len(fn.RequiredParams); i++ {
		annotations, ok := resolveAnnotationValues(ctx, fn.RequiredParams[i].Annotations)
		if !ok {
			return extern.FunctionMetadata{}, false
		}
		params[i-firstParam] = extern.ParameterMetadata{Annotations: annotations}
	}
	metadata := extern.FunctionMetadata{Params: params}
	if fn.RestParams != nil {
		annotations, ok := resolveAnnotationValues(ctx, fn.RestParams.Annotations)
		if !ok {
			return extern.FunctionMetadata{}, false
		}
		metadata.RestParam = &extern.ParameterMetadata{Annotations: annotations}
	}
	return metadata, true
}

func FunctionSignature(_ *extern.Context, impl any) (extern.FunctionSignature, bool) {
	handle, ok := impl.(*InvokableHandle)
	if !ok || handle.descriptor == nil {
		return extern.FunctionSignature{}, false
	}
	return describeFunctionSignature(handle.descriptor, handle.firstParam)
}

func FunctionMetadata(ctx *extern.Context, impl any) (extern.FunctionMetadata, bool) {
	handle, ok := impl.(*InvokableHandle)
	if !ok || handle.descriptor == nil {
		return extern.FunctionMetadata{}, false
	}
	return describeFunctionMetadata(ctx, handle.descriptor, handle.firstParam)
}
