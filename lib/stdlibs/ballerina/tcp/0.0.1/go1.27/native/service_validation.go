// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

package native

import (
	"fmt"
	"slices"

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	onConnect = "onConnect"
	onBytes   = "onBytes"
	onError   = "onError"
	onClose   = "onClose"
)

// The validators below enforce at runtime the rules jBallerina's tcp compiler plugin (TCP_101-TCP_110) enforces at compile time.

func validateService(svcObj *values.Object) *values.Error {
	return rejectUnknownRemoteMethods(svcObj, onConnect)
}

func validateConnectionService(ctx *extern.Context, types tcpTypes, connSvcObj *values.Object) *values.Error {
	if err := rejectUnknownRemoteMethods(connSvcObj, onBytes, onError, onClose); err != nil {
		return err
	}
	if _, ok := ctx.LookupRemoteMethod(connSvcObj, onBytes); !ok {
		if _, plain := connSvcObj.MethodLookupKey(onBytes); !plain {
			return tcpError("Service does not contain `onBytes` function")
		}
	}
	tc := ctx.TypeCtx()
	dataTy := semtypes.Intersect(types.byteArrTy, semtypes.ValReadonly)
	optionalErrTy := semtypes.Union(semtypes.Error, semtypes.Nil)
	checks := []struct {
		name    string
		check   func(sig extern.FunctionSignature) *values.Error
		retTy   semtypes.SemType
		retDesc string
	}{
		{onBytes, func(sig extern.FunctionSignature) *values.Error { return checkOnBytesParams(tc, dataTy, sig) },
			semtypes.Union(types.byteArrTy, optionalErrTy), "byte[]|ballerina/tcp:Error?"},
		{onError, func(sig extern.FunctionSignature) *values.Error { return checkOnErrorParams(tc, sig) },
			optionalErrTy, "ballerina/tcp:Error|()"},
		{onClose, checkOnCloseParams, optionalErrTy, "ballerina/tcp:Error|()"},
	}
	for _, c := range checks {
		handle, ok := ctx.LookupRemoteMethod(connSvcObj, c.name)
		if !ok {
			if _, plain := connSvcObj.MethodLookupKey(c.name); plain {
				return tcpError(fmt.Sprintf("`remote` keyword expected in `%s` function signature.", c.name))
			}
			continue
		}
		sig, ok := ctx.MethodSignature(handle)
		if !ok {
			continue
		}
		if err := c.check(sig); err != nil {
			return err
		}
		if !semtypes.IsSubtype(tc, sig.ReturnType, c.retTy) {
			return tcpError(fmt.Sprintf("Invalid return type provided for function `%s`, return type should be a subtype of `%s`", c.name, c.retDesc))
		}
	}
	return nil
}

func rejectUnknownRemoteMethods(obj *values.Object, allowed ...string) *values.Error {
	names := obj.RemoteMethodNames()
	slices.Sort(names)
	for _, name := range names {
		if !slices.Contains(allowed, name) {
			return tcpError(fmt.Sprintf("Function `%s` not accepted by the service", name))
		}
	}
	return nil
}

func paramCount(sig extern.FunctionSignature) int {
	n := len(sig.Params)
	if sig.RestParam != nil {
		n++
	}
	return n
}

func checkOnBytesParams(tc semtypes.Context, dataTy semtypes.SemType, sig extern.FunctionSignature) *values.Error {
	n := paramCount(sig)
	if n == 0 {
		return tcpError("No parameter provided for `onBytes`, function expects `readonly & byte[]` as a parameter")
	}
	if n > 2 || sig.RestParam != nil {
		return tcpError(fmt.Sprintf("Provided %d parameters, `onBytes` can have only 2 parameters", n))
	}
	for _, p := range sig.Params {
		if isCallerParam(tc, p.Type) {
			continue
		}
		if semtypes.IsSubtype(tc, dataTy, p.Type) && semtypes.IsSubtype(tc, p.Type, semtypes.ValReadonly) {
			continue
		}
		if semtypes.IsSubtype(tc, dataTy, p.Type) {
			return tcpError(fmt.Sprintf("Invalid parameter `%s` provided for `onBytes`, function expects `readonly & byte[]`", p.Name))
		}
		return tcpError(fmt.Sprintf("Invalid parameter `%s` provided for `onBytes` function", p.Name))
	}
	return nil
}

func checkOnErrorParams(tc semtypes.Context, sig extern.FunctionSignature) *values.Error {
	n := paramCount(sig)
	if n == 0 {
		return tcpError("No parameter provided for `onError`, function expects `ballerina/tcp:Error` as a parameter")
	}
	if n > 1 || sig.RestParam != nil {
		return tcpError(fmt.Sprintf("Provided %d parameters, `onError` can have only 1 parameters", n))
	}
	if p := sig.Params[0]; !semtypes.IsSubtype(tc, semtypes.Error, p.Type) {
		return tcpError(fmt.Sprintf("Invalid parameter `%s` provided for `onError`, function expects `ballerina/tcp:Error`", p.Name))
	}
	return nil
}

func checkOnCloseParams(sig extern.FunctionSignature) *values.Error {
	if n := paramCount(sig); n > 0 {
		return tcpError(fmt.Sprintf("Provided %d parameters, `onClose` function cannot have any parameters", n))
	}
	return nil
}

// isCallerParam treats any object-typed parameter as the Caller; the runtime has no handle on the tcp:Caller class type to compare against.
func isCallerParam(tc semtypes.Context, ty semtypes.SemType) bool {
	return semtypes.IsSubtype(tc, ty, semtypes.Object)
}
