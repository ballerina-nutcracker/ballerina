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

	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// handlerSpec describes the signature jBallerina's compiler plugin accepts for
// one udp:Service remote method.
type handlerSpec struct {
	name        string
	maxParams   int
	dataTy      semtypes.SemType
	dataTyName  string
	allowCaller bool
	returnTy    semtypes.SemType
	returnName  string
}

func handlerSpecs(types udpTypes) []handlerSpec {
	dataReturnTy := semtypes.Union(semtypes.Union(types.byteArrTy, types.datagramTy), semtypes.Union(semtypes.Error, semtypes.Nil))
	return []handlerSpec{
		{
			name: "onBytes", maxParams: 2, allowCaller: true,
			dataTy: semtypes.Intersect(types.byteArrTy, semtypes.ValReadonly), dataTyName: "readonly & byte[]",
			returnTy: dataReturnTy, returnName: "byte[] | udp:Datagram | udp:Error?",
		},
		{
			name: "onDatagram", maxParams: 2, allowCaller: true,
			dataTy: semtypes.Intersect(types.datagramTy, semtypes.ValReadonly), dataTyName: "readonly & udp:Datagram",
			returnTy: dataReturnTy, returnName: "byte[] | udp:Datagram | udp:Error?",
		},
		{
			name: "onError", maxParams: 1,
			dataTy: semtypes.Error, dataTyName: "udp:Error",
			returnTy: semtypes.Union(semtypes.Error, semtypes.Nil), returnName: "udp:Error | ()",
		},
	}
}

// validateService applies the rules jBallerina's compiler plugin enforces at
// compile time (UDP_101–UDP_105), returning the first violation as a udp:Error.
func validateService(ctx *extern.Context, types udpTypes, svcObj *values.Object) *values.Error {
	specs := handlerSpecs(types)
	for _, spec := range specs {
		if _, ok := svcObj.MethodLookupKey(spec.name); ok {
			return udpError(fmt.Sprintf("`remote` keyword expected in `%s` function signature.", spec.name))
		}
	}
	for _, name := range svcObj.RemoteMethodNames() {
		if name != "onBytes" && name != "onDatagram" && name != "onError" {
			return udpError(fmt.Sprintf("Function `%s` not accepted by the service", name))
		}
	}
	_, hasOnBytes := ctx.LookupRemoteMethod(svcObj, "onBytes")
	_, hasOnDatagram := ctx.LookupRemoteMethod(svcObj, "onDatagram")
	if hasOnBytes && hasOnDatagram {
		return udpError("Service cannot contain both `onDatagram` and `onBytes` functions.")
	}
	if !hasOnBytes && !hasOnDatagram {
		return udpError("Service does not contain `onDatagram` or `onBytes` function.")
	}
	for _, spec := range specs {
		if handle, ok := ctx.LookupRemoteMethod(svcObj, spec.name); ok {
			if err := validateHandler(ctx, handle, spec); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateHandler(ctx *extern.Context, handle extern.MethodHandle, spec handlerSpec) *values.Error {
	sig, ok := ctx.MethodSignature(handle)
	if !ok {
		return udpError(fmt.Sprintf("Unable to resolve the signature of `%s`", spec.name))
	}
	if len(sig.Params) == 0 {
		return udpError(fmt.Sprintf("No parameter provided for `%s`, function expects `%s` as a parameter.", spec.name, spec.dataTyName))
	}
	paramCount := len(sig.Params)
	if sig.RestParam != nil {
		paramCount++
	}
	if paramCount > spec.maxParams {
		return udpError(fmt.Sprintf("Provided %d parameters, `%s` can have only %d parameters", paramCount, spec.name, spec.maxParams))
	}
	tc := ctx.TypeCtx()
	for _, p := range sig.Params {
		isData := semtypes.IsSameType(tc, p.Type, spec.dataTy)
		isCaller := spec.allowCaller && semtypes.IsSubtype(tc, p.Type, semtypes.Object)
		if !isData && !isCaller {
			return udpError(fmt.Sprintf("Invalid parameter `%s` provided for `%s`, function expects `%s`.", p.Name, spec.name, spec.dataTyName))
		}
	}
	if !semtypes.IsSubtype(tc, sig.ReturnType, spec.returnTy) {
		return udpError(fmt.Sprintf("Invalid return type provided for function `%s`, return type should be a subtype of `%s`", spec.name, spec.returnName))
	}
	return nil
}
