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

	"ballerina-lang-go/bir"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/runtime/internal/modules"
	"ballerina-lang-go/values"
)

// InvokableHandle is provides a unified representation that can be used to execute any function/method
// in runtime
type InvokableHandle struct {
	invoke func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error)
}

func NewBIRHandle(fn *bir.BIRFunction) *InvokableHandle {
	return newBIRHandle(fn, nil)
}

func newBIRHandle(fn *bir.BIRFunction, parentFrame *Frame) *InvokableHandle {
	return &InvokableHandle{
		invoke: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return executeFunction(ctx, fn, args, parentFrame), nil
		},
	}
}

func NewNativeHandle(fn extern.NativeFunc) *InvokableHandle {
	return &InvokableHandle{
		invoke: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			return fn(ctx, args)
		},
	}
}

func NewFunctionValueHandle(env *extern.Env, fnValue *values.Function) (*InvokableHandle, error) {
	reg := env.Registry.(*modules.Registry)
	lookupKey := fnValue.LookupKey
	if fn := reg.GetBIRFunction(lookupKey); fn != nil {
		return newBIRHandle(fn, parentFrameFromFunctionValue(fnValue)), nil
	}
	if externFn := reg.GetNativeFunction(lookupKey); externFn != nil {
		return NewNativeHandle(externFn.Impl), nil
	}
	return nil, fmt.Errorf("function not found: %s", lookupKey)
}

func parentFrameFromFunctionValue(fnValue *values.Function) *Frame {
	if fnValue.ParentFrame == nil {
		return nil
	}
	return fnValue.ParentFrame.(*Frame)
}

func newResourceHandle(receiver *values.Object, match *values.ResourceEntry, path []values.BalValue) *InvokableHandle {
	return &InvokableHandle{
		invoke: func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			full := buildResourceCallArgs(ctx, receiver, match, path, args)
			return lookupAndExecute(ctx, nil, full, match.FunctionLookupKey)
		},
	}
}
