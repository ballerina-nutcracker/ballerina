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
	"ballerina/model"
	"ballerina/runtime/extern"
	"ballerina/runtime/internal/modules"
	"ballerina/values"
)

// LookupObjectMethod resolves a regular method named methodName on obj. The
// second return is false if obj has no such method. Remote methods are not
// resolved through this entry point; use LookupRemoteMethod for those.
func LookupObjectMethod(ctx *extern.Context, obj *values.Object, methodName string) (any, bool) {
	return lookupByMethodName(ctx, obj, methodName)
}

// LookupRemoteMethod resolves the remote method named methodName on obj.
// The second return is false if obj has no such remote method. Callers pass
// the declared method name not the mangled method name;
func LookupRemoteMethod(ctx *extern.Context, obj *values.Object, methodName string) (any, bool) {
	return lookupByMethodName(ctx, obj, model.RemoteMethodName(methodName))
}

func LookupFunction(env *extern.Env, org, module, name string) (any, bool) {
	reg := env.Registry.(*modules.Registry)
	key := org + "/" + module + ":" + name
	if fn := reg.GetBIRFunction(key); fn != nil {
		return NewBIRHandle(fn), true
	}
	if ef := reg.GetNativeFunction(key); ef != nil {
		return NewNativeHandle(ef.Impl), true
	}
	return nil, false
}

// LookupResourceMethod resolves a resource method named resourceMethodName
// on obj. The second return is false if no candidate matches or if more
// than one candidate matches (ambiguous dispatch).
//
// path contains a value for every path segment of the source-level
// resource access expression (literal AND computed). The matcher compares
// each value's shape against the candidate's segment types; the invoked
// function only receives the computed-segment values plus the rest list,
// as constructed by buildResourceCallArgs.
func LookupResourceMethod(ctx *extern.Context, obj *values.Object, resourceMethodName string, path []values.BalValue) (any, bool) {
	matches := resourceFnCandidates(ctx, obj, resourceMethodName, path)
	if len(matches) != 1 {
		return nil, false
	}
	return newResourceHandle(obj, matches[0], path), true
}

// Invoke calls the closure captured by the handle returned from one of
// the Lookup* functions.
func Invoke(ctx *extern.Context, h any, args []values.BalValue) (values.BalValue, error) {
	return h.(*InvokableHandle).invoke(ctx, args)
}

func lookupByMethodName(ctx *extern.Context, obj *values.Object, methodName string) (any, bool) {
	lookupKey, found := obj.MethodLookupKey(methodName)
	if !found {
		return nil, false
	}
	return lookupByKey(ctx, lookupKey)
}

func lookupByKey(ctx *extern.Context, lookupKey string) (any, bool) {
	reg := ctx.Env.Registry.(*modules.Registry)
	if fn := reg.GetBIRFunction(lookupKey); fn != nil {
		return NewBIRHandle(fn), true
	}
	if externFn := reg.GetNativeFunction(lookupKey); externFn != nil {
		return NewNativeHandle(externFn.Impl), true
	}
	return nil, false
}
